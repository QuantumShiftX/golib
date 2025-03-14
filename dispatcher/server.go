package dispatcher

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"github.com/hibiken/asynqmon"
	"github.com/zeromicro/go-zero/core/logx"
)

// Priority 定义任务优先级
type Priority string

func (t Priority) String() string {
	return string(t)
}

// 预定义优先级
const (
	PriorityLow    Priority = "low"
	PriorityNormal Priority = "normal"
	PriorityHigh   Priority = "high"
)

// Server 任务服务器结构体
type Server struct {
	opts             *Options
	srv              *asynq.Server
	scheduler        *asynq.Scheduler
	mux              *asynq.ServeMux
	monitoringServer *http.Server
	wg               sync.WaitGroup
	mu               sync.Mutex
	running          bool
}

// NewServer 创建新服务器
func NewServer(opts *Options) (*Server, error) {
	if opts == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}

	redisOpt := opts.ToRedisClientOpt()

	// 创建日志适配器
	logger := NewLogxAdapter()

	// 创建任务服务器
	srv := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: opts.Server.Concurrency,
			Queues: map[string]int{
				PriorityLow.String():    opts.Server.QueuePriorities.Low,
				PriorityNormal.String(): opts.Server.QueuePriorities.Normal,
				PriorityHigh.String():   opts.Server.QueuePriorities.High,
			},
			ShutdownTimeout: time.Duration(opts.Server.ShutdownTimeout) * time.Second,
			Logger:          logger,
			LogLevel:        asynq.InfoLevel,
		},
	)

	// 创建定时调度器
	scheduler := asynq.NewScheduler(
		redisOpt,
		&asynq.SchedulerOpts{
			Location: time.Local,
			Logger:   logger,
			LogLevel: asynq.InfoLevel,
		},
	)

	server := &Server{
		opts:      opts,
		mux:       asynq.NewServeMux(),
		srv:       srv,
		scheduler: scheduler,
	}

	return server, nil
}

// Register 注册定时任务
func (s *Server) Register(cronspec string, task *asynq.Task, opts ...asynq.Option) error {
	entryID, err := s.scheduler.Register(cronspec, task, opts...)
	if err != nil {
		return fmt.Errorf("failed to register task: %w", err)
	}
	logx.Infof("Registered task with ID: %s, cronspec: %s", entryID, cronspec)
	return nil
}

// HandleFunc 注册处理函数
func (s *Server) HandleFunc(pattern string, handler asynq.HandlerFunc) {
	s.mux.HandleFunc(pattern, handler)
	logx.Infof("Registered handler for pattern: %s", pattern)
}

// Handle 注册处理器
func (s *Server) Handle(pattern string, handler asynq.Handler) {
	s.mux.Handle(pattern, handler)
	logx.Infof("Registered handler for pattern: %s", pattern)
}

// StartMonitoring 启动监控服务
func (s *Server) StartMonitoring() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.opts.Monitoring.Enabled {
		logx.Info("Monitoring is disabled")
		return nil
	}

	address := s.opts.Monitoring.Address
	h := asynqmon.New(asynqmon.Options{
		RootPath:     s.opts.Monitoring.Path,
		RedisConnOpt: s.opts.ToRedisClientOpt(),
	})

	rootPath := h.RootPath()
	if !strings.HasSuffix(rootPath, "/") {
		rootPath += "/"
	}

	mux := http.NewServeMux()
	mux.Handle(rootPath, h)

	s.monitoringServer = &http.Server{
		Addr:         address,
		Handler:      mux,
		ReadTimeout:  1 * time.Minute,
		WriteTimeout: 1 * time.Minute,
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		logx.Infof("Starting monitoring server at %s%s", address, rootPath)
		if err := s.monitoringServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logx.Errorf("Monitoring server error: %v", err)
		}
	}()

	return nil
}

// Start 启动服务
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server is already running")
	}
	s.running = true
	s.mu.Unlock()

	// 启动监控
	if err := s.StartMonitoring(); err != nil {
		return err
	}

	// 错误通道
	errChan := make(chan error, 2)

	// 启动任务服务器
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		logx.Info("Starting asynq server")
		if err := s.srv.Run(s.mux); err != nil {
			errChan <- fmt.Errorf("asynq server error: %w", err)
		}
	}()

	// 启动调度器
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		logx.Info("Starting asynq scheduler")
		if err := s.scheduler.Run(); err != nil {
			errChan <- fmt.Errorf("asynq scheduler error: %w", err)
		}
	}()

	// 等待结束或错误
	select {
	case <-ctx.Done():
		return s.Stop(context.Background())
	case err := <-errChan:
		s.Stop(context.Background())
		return err
	}
}

// GracefulStop 优雅停止服务
func (s *Server) GracefulStop(ctx context.Context) error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	s.mu.Unlock()

	// 关闭监控服务器
	if s.monitoringServer != nil {
		logx.Info("Shutting down monitoring server")
		if err := s.monitoringServer.Shutdown(ctx); err != nil {
			logx.Errorf("Error shutting down monitoring server: %v", err)
		}
	}

	// 优雅关闭调度器
	logx.Info("Shutting down scheduler")
	s.scheduler.Shutdown()

	// 优雅关闭服务器
	logx.Info("Shutting down server")
	s.srv.Shutdown()

	// 等待所有goroutine完成
	waitCh := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop 立即停止服务
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	s.mu.Unlock()

	// 关闭监控服务器
	if s.monitoringServer != nil {
		logx.Info("Stopping monitoring server")
		if err := s.monitoringServer.Close(); err != nil {
			logx.Errorf("Error closing monitoring server: %v", err)
		}
	}

	// 立即停止服务器
	logx.Info("Stopping scheduler")
	s.scheduler.Shutdown()

	logx.Info("Stopping server")
	s.srv.Stop()

	return nil
}

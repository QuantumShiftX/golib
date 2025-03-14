package dispatcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
)

// 全局变量
var (
	clientInstance *Client
	clientOnce     sync.Once
	clientMu       sync.RWMutex
)

// Client 异步任务客户端
type Client struct {
	cli          *asynq.Client
	inspector    *asynq.Inspector
	defaultOpts  []asynq.Option
	redisOptions asynq.RedisClientOpt
}

// TaskOption 任务选项别名
type TaskOption = asynq.Option

// 导出常用的任务选项
var (
	TaskMaxRetry = asynq.MaxRetry
	TaskQueue    = asynq.Queue
	TaskTaskId   = asynq.TaskID
	TaskTimeout  = asynq.Timeout
	TaskUnique   = asynq.Unique
	ProcessAt    = asynq.ProcessAt
	ProcessIn    = asynq.ProcessIn
	Retention    = asynq.Retention
)

// NewClient 创建客户端
func NewClient(opts *Options, defaultTaskOpts ...TaskOption) (*Client, error) {
	if opts == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}

	redisOpt := opts.ToRedisClientOpt()

	client := &Client{
		cli:          asynq.NewClient(redisOpt),
		inspector:    asynq.NewInspector(redisOpt),
		defaultOpts:  defaultTaskOpts,
		redisOptions: redisOpt,
	}

	return client, nil
}

// GetClient 获取全局客户端单例
func GetClient() *Client {
	clientMu.RLock()
	defer clientMu.RUnlock()

	if clientInstance == nil {
		logx.Error("asynq client not initialized, please call SetupClient first")
	}
	return clientInstance
}

// SetupClient 设置全局客户端
func SetupClient(opts *Options, defaultTaskOpts ...TaskOption) error {
	var err error

	clientOnce.Do(func() {
		var client *Client
		client, err = NewClient(opts, defaultTaskOpts...)
		if err != nil {
			return
		}

		clientMu.Lock()
		clientInstance = client
		clientMu.Unlock()

		logx.Infof("AsyncQ client initialized with options: %+v", opts)
	})

	return err
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	return c.cli.Close()
}

// Enqueue 排队任务
func (c *Client) Enqueue(ctx context.Context, method string, args interface{}, opts ...TaskOption) (string, error) {
	payload, err := jsonx.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task payload: %w", err)
	}

	task := asynq.NewTask(method, payload)

	// 合并默认选项和用户提供的选项
	options := append([]asynq.Option{}, c.defaultOpts...)
	options = append(options, opts...)

	info, err := c.cli.EnqueueContext(ctx, task, options...)
	if err != nil {
		return "", fmt.Errorf("failed to enqueue task: %w", err)
	}

	return info.ID, nil
}

// Schedule 计划定时任务
func (c *Client) Schedule(ctx context.Context, method string, args interface{}, processAt time.Time, opts ...TaskOption) (string, error) {
	options := append([]asynq.Option{}, c.defaultOpts...)
	options = append(options, opts...)
	options = append(options, asynq.ProcessAt(processAt))

	return c.Enqueue(ctx, method, args, options...)
}

// ScheduleIn 计划延迟任务
func (c *Client) ScheduleIn(ctx context.Context, method string, args interface{}, delay time.Duration, opts ...TaskOption) (string, error) {
	options := append([]asynq.Option{}, c.defaultOpts...)
	options = append(options, opts...)
	options = append(options, asynq.ProcessIn(delay))

	return c.Enqueue(ctx, method, args, options...)
}

// CancelTask 取消任务
func (c *Client) CancelTask(ctx context.Context, taskID string) error {
	return c.inspector.CancelProcessing(taskID)
}

// TaskStatus 获取任务状态
func (c *Client) TaskStatus(ctx context.Context, taskID string) (string, error) {
	// 获取所有队列名称
	queues, err := c.inspector.Queues()
	if err != nil {
		return "", fmt.Errorf("获取队列列表失败: %w", err)
	}

	// 在所有队列中查找任务
	for _, queue := range queues {
		// 查询处于等待状态的任务
		pendingInfo, err := c.inspector.GetTaskInfo(queue, taskID)
		if err == nil {
			return pendingInfo.State.String(), nil
		}
	}

	// 如果在等待队列中没有找到，检查其他状态
	// 由于 asynq 不直接提供按 ID 查询活跃任务的方法，我们需要获取所有活跃任务然后筛选
	for _, queue := range queues {
		// 获取活跃任务列表
		activeInfos, err := c.inspector.ListActiveTasks(queue)
		if err == nil {
			for _, info := range activeInfos {
				if info.ID == taskID {
					return "active", nil
				}
			}
		}

		// 获取已完成任务列表（如果支持）
		completedInfos, err := c.inspector.ListCompletedTasks(queue)
		if err == nil {
			for _, info := range completedInfos {
				if info.ID == taskID {
					return "completed", nil
				}
			}
		}

		// 获取延迟任务列表
		scheduledInfos, err := c.inspector.ListScheduledTasks(queue)
		if err == nil {
			for _, info := range scheduledInfos {
				if info.ID == taskID {
					return "scheduled", nil
				}
			}
		}

		// 获取重试任务列表
		retryInfos, err := c.inspector.ListRetryTasks(queue)
		if err == nil {
			for _, info := range retryInfos {
				if info.ID == taskID {
					return "retry", nil
				}
			}
		}
	}

	// 如果在所有队列和状态中都找不到该任务
	return "", fmt.Errorf("任务未找到: %s", taskID)
}

// GetTaskByQueue 从指定队列获取任务信息
func (c *Client) GetTaskByQueue(ctx context.Context, queue string, taskID string) (*asynq.TaskInfo, error) {
	return c.inspector.GetTaskInfo(queue, taskID)
}

// DispatchClient 实现grpc.ClientConnInterface的客户端
type DispatchClient struct {
	client      *Client
	defaultOpts []TaskOption
}

// NewDispatchClient 创建GRPC风格的分发客户端
func NewDispatchClient(client *Client, opts ...TaskOption) *DispatchClient {
	return &DispatchClient{
		client:      client,
		defaultOpts: opts,
	}
}

// Invoke 实现grpc.ClientConnInterface接口
func (c *DispatchClient) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	if c.client == nil {
		return fmt.Errorf("asynq client not initialized")
	}

	_, err := c.client.Enqueue(ctx, method, args, c.defaultOpts...)
	return err
}

// NewStream 实现grpc.ClientConnInterface接口
func (c *DispatchClient) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, fmt.Errorf("streaming not supported")
}

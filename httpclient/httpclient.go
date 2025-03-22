package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

// Client 是 HTTP 客户端的封装
type Client struct {
	client        *resty.Client
	baseURL       string
	timeout       time.Duration
	headers       map[string]string
	debugMode     bool
	retryCount    int
	retryWaitTime time.Duration
}

// Option 是创建客户端的选项函数
type Option func(*Client)

// Response 是 HTTP 响应的封装
type Response struct {
	StatusCode int
	Body       []byte
	Headers    map[string][]string
	Error      error // 存储HTTP请求过程中产生的错误
}

// NewClient 创建一个新的 HTTP 客户端
func NewClient(options ...Option) *Client {
	c := &Client{
		client:        resty.New(),
		timeout:       10 * time.Second,
		headers:       make(map[string]string),
		debugMode:     false,
		retryCount:    3,
		retryWaitTime: 100 * time.Millisecond,
	}

	// 应用选项
	for _, option := range options {
		option(c)
	}

	// 设置客户端选项
	c.client.SetTimeout(c.timeout)
	for k, v := range c.headers {
		c.client.SetHeader(k, v)
	}
	c.client.SetDebug(c.debugMode)
	c.client.SetRetryCount(c.retryCount)
	c.client.SetRetryWaitTime(c.retryWaitTime)

	return c
}

// WithBaseURL 设置基础 URL
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
		c.client.SetBaseURL(url)
	}
}

// WithTimeout 设置超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithHeader 添加请求头
func WithHeader(key, value string) Option {
	return func(c *Client) {
		c.headers[key] = value
	}
}

// WithHeaders 批量添加请求头
func WithHeaders(headers map[string]string) Option {
	return func(c *Client) {
		for k, v := range headers {
			c.headers[k] = v
		}
	}
}

// WithDebug 设置调试模式
func WithDebug(debug bool) Option {
	return func(c *Client) {
		c.debugMode = debug
	}
}

// WithRetry 设置重试机制
func WithRetry(count int, waitTime time.Duration) Option {
	return func(c *Client) {
		c.retryCount = count
		c.retryWaitTime = waitTime
	}
}

// 处理resty响应，转换为我们的Response类型
func handleRestyResponse(resp *resty.Response, err error) (*Response, error) {
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}

	// 创建自定义响应对象
	response := &Response{
		StatusCode: resp.StatusCode(),
		Body:       resp.Body(),
		Headers:    resp.Header(),
		Error:      nil,
	}

	// 如果状态码表示错误，或者响应本身表示错误
	if resp.IsError() {
		// 获取错误对象（如果使用了SetError设置）
		errorObj := resp.Error()
		if errorObj != nil {
			// 尝试将错误对象转换为error类型
			if e, ok := errorObj.(error); ok {
				response.Error = e
			} else {
				// 如果不是error类型，创建一个包含错误信息的新错误
				response.Error = fmt.Errorf("请求错误: %v", errorObj)
			}
		} else if resp.StatusCode() >= 400 {
			// 如果没有设置特定的错误对象，但状态码表示错误
			response.Error = fmt.Errorf("HTTP错误状态码: %d", resp.StatusCode())
		}
	}

	return response, nil
}

// Get 发送 GET 请求
func (c *Client) Get(ctx context.Context, path string, params map[string]string) (*Response, error) {
	req := c.client.R().SetContext(ctx)

	if params != nil {
		req.SetQueryParams(params)
	}

	resp, err := req.Get(path)
	return handleRestyResponse(resp, err)
}

// GetJSON 发送 GET 请求并解析 JSON 响应
func (c *Client) GetJSON(ctx context.Context, path string, params map[string]string, result interface{}) error {
	resp, err := c.Get(ctx, path, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return resp.Error
	}

	return json.Unmarshal(resp.Body, result)
}

// Post 发送 POST 请求
func (c *Client) Post(ctx context.Context, path string, body interface{}) (*Response, error) {
	req := c.client.R().SetContext(ctx)

	if body != nil {
		req.SetBody(body)
	}

	resp, err := req.Post(path)
	return handleRestyResponse(resp, err)
}

// PostJSON 发送 POST 请求并解析 JSON 响应
func (c *Client) PostJSON(ctx context.Context, path string, body interface{}, result interface{}) error {
	resp, err := c.Post(ctx, path, body)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return resp.Error
	}

	return json.Unmarshal(resp.Body, result)
}

// PostForm 发送 POST 表单请求
func (c *Client) PostForm(ctx context.Context, path string, formData map[string]string) (*Response, error) {
	req := c.client.R().SetContext(ctx)

	if formData != nil {
		req.SetFormData(formData)
	}

	resp, err := req.Post(path)
	return handleRestyResponse(resp, err)
}

// Put 发送 PUT 请求
func (c *Client) Put(ctx context.Context, path string, body interface{}) (*Response, error) {
	req := c.client.R().SetContext(ctx)

	if body != nil {
		req.SetBody(body)
	}

	resp, err := req.Put(path)
	return handleRestyResponse(resp, err)
}

// Delete 发送 DELETE 请求
func (c *Client) Delete(ctx context.Context, path string) (*Response, error) {
	req := c.client.R().SetContext(ctx)

	resp, err := req.Delete(path)
	return handleRestyResponse(resp, err)
}

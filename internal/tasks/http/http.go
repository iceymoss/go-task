package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/iceymoss/go-task/internal/core"
	"github.com/iceymoss/go-task/pkg/logger"
	"go.uber.org/zap"
)

const TaskName = "http"

// HttpTask HTTP 请求任务
type HttpTask struct {
	client *http.Client
}

func init() {
	// tasks.Register(TaskName, NewHttpTask)
}

func NewHttpTask() core.Task {
	return &HttpTask{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (t *HttpTask) Identifier() string {
	return TaskName
}

// HttpParams 参数结构
type HttpParams struct {
	URL            string            `json:"url" binding:"required"`
	Method         string            `json:"method"`
	Headers        map[string]string `json:"headers"`
	Body           any               `json:"body"`
	Timeout        int               `json:"timeout"`
	ExpectedStatus int               `json:"expected_status"`
}

func (t *HttpTask) Run(ctx context.Context, params map[string]any) error {
	// 解析参数
	p := parseParams(params)
	if p.URL == "" {
		return fmt.Errorf("url is required")
	}

	if p.Method == "" {
		p.Method = "GET"
	}

	logger.Info("🚀 [HttpTask] Starting HTTP request",
		zap.String("url", p.URL),
		zap.String("method", p.Method),
	)

	// 设置超时
	if p.Timeout > 0 {
		t.client.Timeout = time.Duration(p.Timeout) * time.Second
	}

	// 准备请求体
	var body io.Reader
	if p.Body != nil {
		jsonData, err := json.Marshal(p.Body)
		if err != nil {
			return fmt.Errorf("failed to marshal body: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, p.Method, p.URL, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	if p.Headers != nil {
		for key, value := range p.Headers {
			req.Header.Set(key, value)
		}
	}

	// 如果有 body，设置 Content-Type
	if p.Body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// 执行请求
	startTime := time.Now()
	resp, err := t.client.Do(req)
	if err != nil {
		logger.Error("❌ [HttpTask] Request failed",
			zap.String("url", p.URL),
			zap.Error(err),
		)
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	duration := time.Since(startTime)

	// 检查状态码
	if p.ExpectedStatus > 0 && resp.StatusCode != p.ExpectedStatus {
		logger.Error("❌ [HttpTask] Status code mismatch",
			zap.String("url", p.URL),
			zap.Int("expected", p.ExpectedStatus),
			zap.Int("actual", resp.StatusCode),
			zap.String("response", string(respBody)),
		)
		return fmt.Errorf("status code mismatch: expected %d, got %d", p.ExpectedStatus, resp.StatusCode)
	}

	logger.Info("✅ [HttpTask] Request completed",
		zap.String("url", p.URL),
		zap.Int("status", resp.StatusCode),
		zap.Duration("duration", duration),
		zap.String("response", string(respBody)),
	)

	return nil
}

func parseParams(params map[string]any) HttpParams {
	p := HttpParams{
		Method:         "GET",
		Timeout:        30,
		ExpectedStatus: 200,
	}

	if v, ok := params["url"].(string); ok {
		p.URL = v
	}
	if v, ok := params["method"].(string); ok {
		p.Method = v
	}
	if v, ok := params["headers"].(map[string]any); ok {
		p.Headers = make(map[string]string)
		for k, val := range v {
			if str, ok := val.(string); ok {
				p.Headers[k] = str
			}
		}
	}
	if v := params["body"]; v != nil {
		p.Body = v
	}
	if v, ok := params["timeout"].(float64); ok {
		p.Timeout = int(v)
	}
	if v, ok := params["expected_status"].(float64); ok {
		p.ExpectedStatus = int(v)
	}

	return p
}

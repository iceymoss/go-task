package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/iceymoss/go-task/internal/core"
	"github.com/iceymoss/go-task/internal/tasks"
)

// WriterTask AI 写作任务
type WriterTask struct{}

func init() {
	// 注册任务名，config.yaml 中会引用这个名字
	tasks.Register("ai:writer", NewWriterTask)
}

func NewWriterTask() core.Task {
	return &WriterTask{}
}

func (t *WriterTask) Identifier() string {
	return "ai:writer"
}

func (t *WriterTask) Run(ctx context.Context, params map[string]any) error {
	// 1. 解析参数
	apiKey, _ := params["api_key"].(string)
	repoPath, _ := params["repo_path"].(string)
	authorName, _ := params["author_name"].(string)
	authorEmail, _ := params["author_email"].(string)
	randomDelay, _ := params["random_delay"].(bool) // 是否开启随机延迟

	if apiKey == "" || repoPath == "" {
		return fmt.Errorf("missing required params: api_key or repo_path")
	}

	// 2. 随机延迟逻辑 (针对每日0点触发，随机延后到0-60分钟内)
	if randomDelay {
		rand.Seed(time.Now().UnixNano())
		minutes := rand.Intn(60)
		delay := time.Duration(minutes) * time.Minute
		log.Printf("💤 [AI Task] Sleeping for %d minutes...", minutes)

		select {
		case <-time.After(delay):
			log.Println("⏰ [AI Task] Waking up to write article...")
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// 3. 调用 AI 生成
	title, content, err := t.callAI(ctx, apiKey)
	if err != nil {
		return fmt.Errorf("AI call failed: %w", err)
	}

	// 4. 保存文件
	filename, err := t.saveFile(repoPath, authorName, title, content)
	if err != nil {
		return fmt.Errorf("save file failed: %w", err)
	}

	// 5. Git 提交
	if err := t.gitPush(ctx, repoPath, filename, authorName, authorEmail); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}

	return nil
}

// callAI 调用 OpenAI 接口
func (t *WriterTask) callAI(ctx context.Context, apiKey string) (string, string, error) {
	prompt := "请写一篇关于“现代软件架构设计”的技术短文，要求Markdown格式。返回JSON格式: {\"title\": \"标题\", \"content\": \"正文内容\"}。"

	reqBody := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]string{"type": "json_object"},
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("API Error: %s", string(body))
	}

	var aiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &aiResp); err != nil {
		return "", "", err
	}
	if len(aiResp.Choices) == 0 {
		return "", "", fmt.Errorf("empty choice")
	}

	var result struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	// 容错处理：如果 AI 返回的不是纯 JSON，这里可能会失败
	if err := json.Unmarshal([]byte(aiResp.Choices[0].Message.Content), &result); err != nil {
		return "Untitled", aiResp.Choices[0].Message.Content, nil
	}

	return result.Title, result.Content, nil
}

func (t *WriterTask) saveFile(repoPath, author, title, content string) (string, error) {
	safeTitle := strings.ReplaceAll(title, " ", "_")
	filename := fmt.Sprintf("%s-%s.md", time.Now().Format("2006-01-02"), safeTitle)
	fullPath := filepath.Join(repoPath, "posts", filename)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", err
	}

	fileContent := fmt.Sprintf("---\ntitle: %s\ndate: %s\nauthor: %s\n---\n\n%s",
		title, time.Now().Format(time.RFC3339), author, content)

	return filename, os.WriteFile(fullPath, []byte(fileContent), 0644)
}

func (t *WriterTask) gitPush(ctx context.Context, repoPath, filename, name, email string) error {
	run := func(args ...string) error {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = repoPath
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git %v failed: %s, out: %s", args, err, string(out))
		}
		return nil
	}

	_ = run("config", "user.name", name)
	_ = run("config", "user.email", email)
	_ = run("pull", "--rebase")
	if err := run("add", "."); err != nil {
		return err
	}
	if err := run("commit", "-m", "feat: auto post "+filename); err != nil {
		return nil
	} // 没变化忽略
	return run("push")
}

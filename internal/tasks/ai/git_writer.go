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
	tasks.Register("ai:writer", NewWriterTask)
}

func NewWriterTask() core.Task {
	return &WriterTask{}
}

func (t *WriterTask) Identifier() string {
	return "ai:writer"
}

// WriterParams Params 参数结构体定义，方便阅读
type WriterParams struct {
	ApiKey      string `json:"api_key"`
	RemoteURL   string `json:"remote_url"`   // Git 远程地址 (git@github.com:xxx/xxx.git)
	WorkDir     string `json:"work_dir"`     // 指定的工作根目录，例如 /tmp/tasks
	SSHKeyPath  string `json:"ssh_key_path"` // SSH 私钥的绝对路径
	AuthorName  string `json:"author_name"`
	AuthorEmail string `json:"author_email"`
	RandomDelay bool   `json:"random_delay"`
}

func (t *WriterTask) Run(ctx context.Context, params map[string]any) error {
	// 1. 解析参数
	p := parseParams(params)
	if p.ApiKey == "" || p.RemoteURL == "" || p.SSHKeyPath == "" {
		return fmt.Errorf("missing required params: api_key, remote_url, or ssh_key_path")
	}

	// 2. 随机延迟逻辑
	if p.RandomDelay {
		doRandomDelay(ctx)
	}

	// 3. 准备工作目录 (Clone -> Process -> Push -> Clean)
	// 我们在 WorkDir 下创建一个带时间戳的随机目录，防止并发冲突
	taskID := fmt.Sprintf("task_%d_%d", time.Now().Unix(), rand.Intn(1000))
	repoLocalPath := filepath.Join(p.WorkDir, taskID)

	// 确保最终清理
	defer func() {
		log.Printf("🧹 [AI Task] Cleaning up workspace: %s", repoLocalPath)
		_ = os.RemoveAll(repoLocalPath)
	}()

	// 4. Git Clone 项目
	log.Printf("📥 [AI Task] Cloning %s into %s", p.RemoteURL, repoLocalPath)
	if err := t.gitClone(ctx, p.RemoteURL, repoLocalPath, p.SSHKeyPath); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	// 5. 调用 AI 生成
	log.Println("🤖 [AI Task] Generating content...")
	title, content, err := t.callAI(ctx, p.ApiKey)
	if err != nil {
		return fmt.Errorf("AI call failed: %w", err)
	}

	// 6. 保存文件到克隆下来的目录中
	filename, err := t.saveFile(repoLocalPath, p.AuthorName, title, content)
	if err != nil {
		return fmt.Errorf("save file failed: %w", err)
	}

	// 7. Git 提交并推送
	log.Println("🚀 [AI Task] Pushing changes...")
	if err := t.gitPush(ctx, repoLocalPath, filename, p, p.SSHKeyPath); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}

	log.Println("✅ [AI Task] Completed successfully.")
	return nil
}

// gitClone 拉取项目
func (t *WriterTask) gitClone(ctx context.Context, remoteURL, localPath, sshKeyPath string) error {
	// 确保父目录存在
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}

	// 使用 ssh-agent 或指定 key 的方式。这里使用 GIT_SSH_COMMAND 环境变量最简单，无需系统配置
	// -o StrictHostKeyChecking=no 防止第一次连接时卡在 yes/no 确认上
	sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null", sshKeyPath)

	// --depth 1 浅克隆，加快速度，减少流量
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", remoteURL, localPath)
	cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCmd)

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("output: %s, error: %w", string(out), err)
	}
	return nil
}

// gitPush 提交更改
func (t *WriterTask) gitPush(ctx context.Context, repoPath, filename string, p WriterParams, sshKeyPath string) error {
	sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null", sshKeyPath)
	env := append(os.Environ(), "GIT_SSH_COMMAND="+sshCmd)

	run := func(args ...string) error {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = repoPath // 必须在仓库目录下执行
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git %v failed: %s, out: %s", args, err, string(out))
		}
		return nil
	}

	// 配置本地用户信息（只影响这个临时仓库）
	_ = run("config", "user.name", p.AuthorName)
	_ = run("config", "user.email", p.AuthorEmail)

	if err := run("add", "."); err != nil {
		return err
	}

	commitMsg := fmt.Sprintf("feat: auto post %s", filename)
	if err := run("commit", "-m", commitMsg); err != nil {
		// 如果没有变化（git commit 返回非0），可能是 AI 生成了重复内容，这不算严重错误
		log.Println("⚠️ No changes to commit.")
		return nil
	}

	// 推送
	return run("push", "origin", "main") // 假设主分支是 main，如果是 master 请修改
}

// saveFile 保存文件
func (t *WriterTask) saveFile(repoPath, author, title, content string) (string, error) {
	// 简单过滤标题中的非法字符
	safeTitle := strings.ReplaceAll(title, " ", "_")
	safeTitle = strings.ReplaceAll(safeTitle, "/", "-")

	filename := fmt.Sprintf("%s-%s.md", time.Now().Format("2006-01-02"), safeTitle)
	// 假设文章保存在 posts 目录下
	fullDir := filepath.Join(repoPath, "posts")
	fullPath := filepath.Join(fullDir, filename)

	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return "", err
	}

	// 简单的 Front Matter
	fileContent := fmt.Sprintf("---\ntitle: %s\ndate: %s\nauthor: %s\n---\n\n%s",
		title, time.Now().Format(time.RFC3339), author, content)

	return filename, os.WriteFile(fullPath, []byte(fileContent), 0644)
}

// callAI (保持原有逻辑，稍作优化)
func (t *WriterTask) callAI(ctx context.Context, apiKey string) (string, string, error) {
	prompt := "请写一篇关于“现代软件架构设计”的技术短文，要求Markdown格式。返回严格的JSON格式: {\"title\": \"标题\", \"content\": \"正文内容\"}。"

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

	client := &http.Client{Timeout: 3 * time.Minute} // 增加一点超时时间
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
	if err := json.Unmarshal([]byte(aiResp.Choices[0].Message.Content), &result); err != nil {
		// 容错：如果 JSON 解析失败，直接使用原始内容作为 Content
		return "Untitled_AI_Article", aiResp.Choices[0].Message.Content, nil
	}

	return result.Title, result.Content, nil
}

// 辅助函数：解析参数
func parseParams(params map[string]any) WriterParams {
	p := WriterParams{}
	if v, ok := params["api_key"].(string); ok {
		p.ApiKey = v
	}
	if v, ok := params["remote_url"].(string); ok {
		p.RemoteURL = v
	}
	if v, ok := params["work_dir"].(string); ok {
		p.WorkDir = v
	} else {
		p.WorkDir = os.TempDir() // 默认使用系统临时目录
	}
	if v, ok := params["ssh_key_path"].(string); ok {
		p.SSHKeyPath = v
	}
	if v, ok := params["author_name"].(string); ok {
		p.AuthorName = v
	}
	if v, ok := params["author_email"].(string); ok {
		p.AuthorEmail = v
	}
	if v, ok := params["random_delay"].(bool); ok {
		p.RandomDelay = v
	}
	return p
}

// 辅助函数：随机延迟
func doRandomDelay(ctx context.Context) {
	rand.Seed(time.Now().UnixNano())
	minutes := rand.Intn(60)
	delay := time.Duration(minutes) * time.Minute
	log.Printf("💤 [AI Task] Sleeping for %d minutes...", minutes)
	select {
	case <-time.After(delay):
		log.Println("⏰ [AI Task] Waking up...")
	case <-ctx.Done():
	}
}

package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/iceymoss/go-task/internal/core"
	"github.com/iceymoss/go-task/internal/tasks"
	"github.com/iceymoss/go-task/pkg/db"
	"github.com/iceymoss/go-task/pkg/db/objects"
	"github.com/iceymoss/go-task/pkg/logger"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"go.uber.org/zap"
)

const (
	TaskName = "ai:writer"
	LastID   = TaskName + ":last_id"
)

// WriterTask AI 写作任务
type WriterTask struct{}

func init() {
	tasks.Register(TaskName, NewWriterTask)
}

func NewWriterTask() core.Task {
	return &WriterTask{}
}

func (t *WriterTask) Identifier() string {
	return TaskName
}

// WriterParams 参数结构体
type WriterParams struct {
	ApiKey      string `json:"api_key"`
	BaseURL     string `json:"base_url"`     // 新增：支持自定义 BaseURL (DeepSeek)
	Model       string `json:"model"`        // 新增：支持自定义模型 (deepseek-reasoner)
	RemoteURL   string `json:"remote_url"`   // Git 远程地址
	WorkDir     string `json:"work_dir"`     // 工作目录
	SSHKeyPath  string `json:"ssh_key_path"` // SSH 私钥路径
	AuthorName  string `json:"author_name"`
	AuthorEmail string `json:"author_email"`
	Topic       string `json:"topic"` // 可选：写作主题
	RandomDelay bool   `json:"random_delay"`
}

func (t *WriterTask) Run(ctx context.Context, params map[string]any) error {
	// 1. 解析参数
	p := parseParams(params)
	if p.ApiKey == "" || p.RemoteURL == "" || p.SSHKeyPath == "" {
		return fmt.Errorf("missing required params: api_key, remote_url, or ssh_key_path")
	}

	// 2. 随机延迟
	if p.RandomDelay {
		doRandomDelay(ctx)
	}

	// 3. 准备工作目录
	taskID := fmt.Sprintf("task_%d_%d", time.Now().Unix(), rand.Intn(1000))
	repoLocalPath := filepath.Join(p.WorkDir, taskID)

	// 确保最终清理
	defer func() {
		log.Printf("🧹 [AI Task] Cleaning up workspace: %s", repoLocalPath)
		_ = os.RemoveAll(repoLocalPath)
	}()

	// 4. Git Clone
	log.Printf("📥 [AI Task] Cloning %s into %s", p.RemoteURL, repoLocalPath)
	if err := t.gitClone(ctx, p.RemoteURL, repoLocalPath, p.SSHKeyPath); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	// 数据库中获取文章话题
	dbConn := db.GetMysqlConn(db.MYSQL_DB_GO_TASK)

	// 自动迁移表结构 (为了方便，生产环境建议手动建表)
	_ = dbConn.AutoMigrate(&objects.SysArticle{})

	// 没错从数据库中一篇文章来做,需要使用Redis来保存读取指针
	var lastId string
	rdb := db.GetRedisConn()
	lastId, err := rdb.Get(ctx, LastID).Result()
	if err != nil {
		logger.Error("Failed to get last id from redis", zap.Error(err))
		return err
	}
	if lastId == "" {
		logger.Error("No last id found in redis", zap.Error(err))
		return fmt.Errorf("No last id found in redis")
	}

	article := &objects.SysArticle{}
	// id >= db.id 的一条，注意排序
	err = dbConn.Model(article).Where("id > ?", lastId).Order("id ASC").First(article).Error
	if err != nil {
		logger.Error("Failed to get article from db", zap.Error(err))
		return err
	}

	topic := strings.Join(article.Topics, "+")
	if topic != "" {
		p.Topic = topic
	}

	// 5. 调用 AI 生成 (封装在 callAI 中)
	log.Printf("🤖 [AI Task] Generating content using %s (Model: %s)...", p.BaseURL, p.Model)
	title, content, err := t.callAI(ctx, p)
	if err != nil {
		return fmt.Errorf("AI call failed: %w", err)
	}

	fmt.Println("ai创作标题：", title)
	fmt.Println("ai创作内容: ", content)

	// 6. 保存文件
	filename, err := t.saveFile(repoLocalPath, p.AuthorName, title, content)
	if err != nil {
		return fmt.Errorf("save file failed: %w", err)
	}

	// 7. Git 提交并推送
	log.Println("🚀 [AI Task] Pushing changes...")
	if err := t.gitPush(ctx, repoLocalPath, filename, p, p.SSHKeyPath); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}

	rdb.Set(ctx, LastID, article.ID+1, 0)

	log.Println("✅ [AI Task] Completed successfully.")
	return nil
}

// -------------------------------------------------------------------------
// 使用 LangChain 调用 DeepSeek R1
// -------------------------------------------------------------------------
func (t *WriterTask) callAI(ctx context.Context, p WriterParams) (string, string, error) {
	// 1. 初始化 LangChain Client
	// DeepSeek 兼容 OpenAI 协议，所以使用 openai 包，通过 BaseURL 指向 DeepSeek
	llm, err := openai.New(
		openai.WithToken(p.ApiKey),
		openai.WithBaseURL(p.BaseURL),
		openai.WithModel(p.Model),
	)
	if err != nil {
		return "", "", fmt.Errorf("init llm client failed: %w", err)
	}

	// 2. 构造 Prompt
	// DeepSeek R1 是推理模型，虽然 LangChain 会自动提取最终内容，
	// 但我们依然需要明确要求 JSON 格式以便程序处理。
	topic := p.Topic
	if topic == "" {
		topic = "现代软件架构设计"
	}

	prompt := fmt.Sprintf(`你是一个资深技术博主。请写一篇关于“%s”的技术文章。
要求：
1. 必须返回严格的 JSON 格式，不要包含 Markdown 代码块标记（如 '''json）。
2. JSON 格式必须包含两个字段：{"title": "文章标题", "content": "Markdown正文"}。
3. 内容要有深度，包含代码示例，语气幽默。
4. 创作内容必须使用中文。
5. 只返回 JSON，不要包含其他解释性文字。`, topic)

	// 3. 调用生成
	// GenerateFromSinglePrompt 会处理 HTTP 请求并提取 content 字段
	// (DeepSeek R1 的 reasoning_content 会被 LangChain 忽略，只保留最终结果)
	responseContent, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt,
		llms.WithTemperature(0.6), // R1 建议 Temperature 0.5-0.7
	)
	if err != nil {
		return "", "", fmt.Errorf("generate failed: %w", err)
	}

	// 4. 解析结果 (LangChain 返回的是纯文本 String)
	var result struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	// 清理可能存在的 Markdown 标记 (容错处理)
	// 有时候模型还是会忍不住加 ```json ... ```
	cleanJSON := strings.TrimSpace(responseContent)
	cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
	cleanJSON = strings.TrimPrefix(cleanJSON, "```")
	cleanJSON = strings.TrimSuffix(cleanJSON, "```")

	if err := json.Unmarshal([]byte(cleanJSON), &result); err != nil {
		// 如果解析失败，可能是 AI 没听话返回 JSON，直接用全文当正文
		log.Printf("⚠️ JSON parse failed, using raw content. Err: %v", err)
		// 生成一个默认标题
		return fmt.Sprintf("AI_Article_%d", time.Now().Unix()), responseContent, nil
	}

	return result.Title, result.Content, nil
}

// -------------------------------------------------------------------------
// 辅助函数 (Git 操作 & 文件处理)
// -------------------------------------------------------------------------

func (t *WriterTask) gitClone(ctx context.Context, remoteURL, localPath, sshKeyPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}
	sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null", sshKeyPath)
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", remoteURL, localPath)
	cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND="+sshCmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("output: %s, error: %w", string(out), err)
	}
	return nil
}

func (t *WriterTask) gitPush(ctx context.Context, repoPath, filename string, p WriterParams, sshKeyPath string) error {
	sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null", sshKeyPath)
	env := append(os.Environ(), "GIT_SSH_COMMAND="+sshCmd)

	run := func(args ...string) error {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = repoPath
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git %v failed: %s, out: %s", args, err, string(out))
		}
		return nil
	}

	_ = run("config", "user.name", p.AuthorName)
	_ = run("config", "user.email", p.AuthorEmail)

	if err := run("add", "."); err != nil {
		return err
	}

	commitMsg := fmt.Sprintf("feat: auto post %s", filename)
	if err := run("commit", "-m", commitMsg); err != nil {
		log.Println("⚠️ No changes to commit.")
		return nil
	}

	return run("push", "origin", "HEAD:main")
}

func (t *WriterTask) saveFile(repoPath, author, title, content string) (string, error) {
	safeTitle := strings.ReplaceAll(title, " ", "_")
	safeTitle = strings.ReplaceAll(safeTitle, "/", "-")
	filename := fmt.Sprintf("%s-%s.md", time.Now().Format("2006-01-02"), safeTitle)

	fullDir := filepath.Join(repoPath, "posts")
	fullPath := filepath.Join(fullDir, filename)

	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return "", err
	}

	fileContent := fmt.Sprintf("---\ntitle: %s\ndate: %s\nauthor: %s\n---\n\n%s",
		title, time.Now().Format(time.RFC3339), author, content)

	return filename, os.WriteFile(fullPath, []byte(fileContent), 0644)
}

// 辅助函数：解析参数 (增加了 BaseURL 和 Model 的解析)
func parseParams(params map[string]any) WriterParams {
	p := WriterParams{
		WorkDir:     os.TempDir(),
		AuthorName:  "AI Bot",
		AuthorEmail: "bot@example.com",
		// 设置 DeepSeek 默认值
		BaseURL: "[https://api.deepseek.com](https://api.deepseek.com)",
		Model:   "deepseek-reasoner", // 默认使用 R1
	}

	getString := func(key string) string {
		if v, ok := params[key].(string); ok && v != "" {
			return v
		}
		return ""
	}

	fmt.Println("api_key:++++++++++:", getString("api_key"))

	if v := getString("api_key"); v != "" {
		p.ApiKey = v
	}
	if v := getString("remote_url"); v != "" {
		p.RemoteURL = v
	}
	if v := getString("work_dir"); v != "" {
		p.WorkDir = v
	}
	if v := getString("ssh_key_path"); v != "" {
		p.SSHKeyPath = v
	}
	if v := getString("author_name"); v != "" {
		p.AuthorName = v
	}
	if v := getString("author_email"); v != "" {
		p.AuthorEmail = v
	}
	// 新增参数解析
	if v := getString("base_url"); v != "" {
		p.BaseURL = v
	}
	if v := getString("model"); v != "" {
		p.Model = v
	}
	if v := getString("topic"); v != "" {
		p.Topic = v
	}

	if v, ok := params["random_delay"].(bool); ok {
		p.RandomDelay = v
	}
	return p
}

func doRandomDelay(ctx context.Context) {
	// 注意：如果是 Go 1.20 之前的版本，rand.Seed 最好放在 main() 或 init() 中全局执行一次，
	// 不要放在函数内部，否则高并发下可能导致生成的随机数重复。
	// Go 1.20+ 已经不需要手动 Seed 了。
	rand.Seed(time.Now().UnixNano())

	// 生成 0 到 600 之间的随机整数 (包含 0，包含 600)
	// 10小时 * 60分钟 = 600分钟
	minutes := rand.Intn(4)

	delay := time.Duration(minutes) * time.Minute

	// 为了方便看日志，我增加了一个显示小时数的转换
	log.Printf("💤 [AI Task] Sleeping for %d minutes (approx %.1f hours)...", minutes, float64(minutes)/60.0)

	select {
	case <-time.After(delay):
		log.Println("⏰ [AI Task] Waking up...")
	case <-ctx.Done():
		log.Println("⚠️ [AI Task] Context cancelled")
	}
}

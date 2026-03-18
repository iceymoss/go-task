package ai

import (
	"bytes"
	"context"
	"crypto/tls"
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
	"github.com/iceymoss/go-task/internal/tasks/base_task"
	"github.com/iceymoss/go-task/pkg/constants"
	"github.com/iceymoss/go-task/pkg/db"
	"github.com/iceymoss/go-task/pkg/db/objects"
	"github.com/iceymoss/go-task/pkg/logger"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"go.uber.org/zap"
)

const (
	aiWriterTaskName = "ai:writer"
	LastID           = aiWriterTaskName + ":last_id"
)

// WriterTask AI 写作任务
type WriterTask struct {
	base_task.BaseTask
}

func NewWriterTask() core.Task {
	return &WriterTask{
		BaseTask: base_task.BaseTask{
			Name:     aiWriterTaskName,
			TaskType: constants.TaskTypeYAML,
		},
	}
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

	// 1. 登录信息
	username := "ai_bot"
	password := "admin123"

	// 执行登录获取 Token
	token, err := login(username, password)
	if err != nil {
		logger.Error("登录失败: %v", zap.Error(err))
		return err
	}
	// 调试用，打印部分 token
	// fmt.Printf("获取到的 Token (前20位): %s...\n", token[:20])

	// 2. 准备要创建的文章数据
	newArticle := ArticleCreateRequest{
		Title:       title,
		Content:     content,
		Summary:     strings.Join(article.Topics, " "),
		Cover:       "/uploads/images/2026/02/12/85a5205c-7c4f-49d3-81db-f542b5d7b502.jpg",
		CategoryID:  23,
		TagIDs:      []int{},
		Status:      1,
		IsTop:       false,
		IsRecommend: false,
	}

	// 执行创建文章
	if err := createArticle(token, newArticle); err != nil {
		logger.Error("流程终止", zap.Error(err))
		return err
	}

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
3. 内容要有深度，包含代码示例，严谨，语气根据文章主题定，必须谦虚，不要说大话，说空话，记住你叫icey或者iceymoss。
4. 创作内容必须使用中文。
5. 文章应该是一个开发者、技术极客写出来的水平，需要逻辑严谨，循序渐进，知识点详细，面向初级，中级，高级开发者。
6. 产出文章要专业，代码实例要全面，内容要通俗易懂，把别人教会这才是真正的能力，我们程序员群里应该是最务实和实践性的人群。
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

// ==================== 数据结构定义 ====================

// 1. 登录请求参数
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// 2. 登录响应数据 (只定义我们需要提取的字段)
type LoginResponseData struct {
	Token string `json:"token"`
	// User 字段可以省略，如果我们只需要 token 的话
}

// 登录接口的完整响应结构
type LoginResponse struct {
	Code    int               `json:"code"`
	Message string            `json:"message"`
	Data    LoginResponseData `json:"data"`
}

// 3. 创建文章请求参数
type ArticleCreateRequest struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	Summary     string `json:"summary"`
	Cover       string `json:"cover"`
	CategoryID  int    `json:"category_id"`
	TagIDs      []int  `json:"tag_ids"`
	Status      int    `json:"status"`
	IsTop       bool   `json:"is_top"`
	IsRecommend bool   `json:"is_recommend"`
}

// 4. 通用基础响应 (用于检查创建文章是否成功)
type BasicResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ==================== 配置常量 ====================

const (
	BaseURL        = "http://is.iceymoss.com"
	LoginEndpoint  = BaseURL + "/api/login"
	CreateEndpoint = BaseURL + "/api/articles"
	UserAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36"
)

// 全局 HTTP 客户端，配置了跳过 TLS 验证 (对应 --insecure)
var httpClient = &http.Client{
	Timeout: time.Second * 30,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

// ==================== 核心函数 ====================

// login 执行登录操作并返回 Token
func login(username, password string) (string, error) {
	fmt.Println("正在发起登录请求...")

	// 1. 准备请求数据
	reqBody := LoginRequest{
		Username: username,
		Password: password,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化登录请求失败: %v", err)
	}

	// 2. 创建 HTTP 请求
	req, err := http.NewRequest(http.MethodPost, LoginEndpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("创建登录请求失败: %v", err)
	}

	// 3. 设置必要的请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	// 添加 curl 中其他的 header，虽然不一定是必须的，但为了保持一致性
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", BaseURL+"/login")
	req.Header.Set("Origin", BaseURL)

	// 4. 发送请求
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送登录请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 5. 读取并解析响应
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取登录响应失败: %v", err)
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(respBytes, &loginResp); err != nil {
		return "", fmt.Errorf("解析登录响应 JSON 失败: %v, 原始内容: %s", err, string(respBytes))
	}

	// 6. 检查业务状态码
	if loginResp.Code != 0 {
		return "", fmt.Errorf("登录失败，API返回错误: [%d] %s", loginResp.Code, loginResp.Message)
	}

	fmt.Println("登录成功！")
	return loginResp.Data.Token, nil
}

// createArticle 使用 Token 创建文章
func createArticle(token string, article ArticleCreateRequest) error {
	fmt.Println("\n正在发起创建文章请求...")

	// 1. 准备请求数据
	jsonBody, err := json.Marshal(article)
	if err != nil {
		return fmt.Errorf("序列化文章数据失败: %v", err)
	}

	// 2. 创建 HTTP 请求
	req, err := http.NewRequest(http.MethodPost, CreateEndpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("创建文章请求失败: %v", err)
	}

	// 3. 设置必要的请求头，最重要的是 Authorization
	// 注意 Bearer 后面的空格
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Referer", BaseURL+"/dashboard/articles/create")

	// 4. 发送请求
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送创建文章请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 5. 读取并解析响应
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取文章创建响应失败: %v", err)
	}

	var basicResp BasicResponse
	if err := json.Unmarshal(respBytes, &basicResp); err != nil {
		// 如果解析 JSON 失败，打印原始响应体以便调试
		return fmt.Errorf("解析文章创建响应 JSON 失败: %v, 原始内容: %s", err, string(respBytes))
	}

	// 6. 检查业务状态码
	if basicResp.Code != 0 {
		return fmt.Errorf("创建文章失败，API返回错误: [%d] %s", basicResp.Code, basicResp.Message)
	}

	fmt.Printf("文章创建成功！响应信息: %s\n", basicResp.Message)
	return nil
}

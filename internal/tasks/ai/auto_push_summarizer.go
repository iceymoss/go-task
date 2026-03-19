package ai

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
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

	"github.com/go-redis/redis/v8"
	"github.com/iceymoss/go-task/internal/core"
	"github.com/iceymoss/go-task/internal/tasks/base_task"
	"github.com/iceymoss/go-task/pkg/constants"
	"github.com/iceymoss/go-task/pkg/db"
	"github.com/iceymoss/go-task/pkg/db/models"
	"github.com/iceymoss/go-task/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	aiAutoPushSummarizerTaskName = "ai:auto:push:tech_summarizer"

	lastID = aiAutoPushSummarizerTaskName + ":last:id"
)

// AutoPushSummarizerTask 结构体
type AutoPushSummarizerTask struct {
	base_task.BaseTask
	params AutoPushSummarizerTaskParams
}

// AutoPushSummarizerTaskParams 任务参数
type AutoPushSummarizerTaskParams struct {
	RemoteURL   string `json:"remote_url"`   // Git 远程地址
	WorkDir     string `json:"work_dir"`     // 工作目录
	SSHKeyPath  string `json:"ssh_key_path"` // SSH 私钥路径
	AuthorName  string `json:"author_name"`
	AuthorEmail string `json:"author_email"`
}

func NewAutoPushSummarizerTask() core.Task {
	return &AutoPushSummarizerTask{
		BaseTask: base_task.BaseTask{
			Name:        aiAutoPushSummarizerTaskName,
			TaskType:    constants.TaskTypeSYSTEM,
			DefaultCron: "@every 10m",
		},
		params: AutoPushSummarizerTaskParams{
			RemoteURL:   "git@github.com:iceymoss/iceymoss.github.io.git",
			WorkDir:     "/tmp/iceymoss_tasks/autopush",
			SSHKeyPath:  "/root/.ssh/id_rsa",
			AuthorName:  "iceymoss",
			AuthorEmail: "ice_moss@163.com",
		},
	}
}

// AutoPushSummarizerTaskArticleCreateRequest 创建文章请求参数
type AutoPushSummarizerTaskArticleCreateRequest struct {
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

func (t *AutoPushSummarizerTask) Run(ctx context.Context, params map[string]any) error {
	taskID := fmt.Sprintf("task_%d_%d", time.Now().Unix(), rand.Intn(1000))
	repoLocalPath := filepath.Join(t.params.WorkDir, taskID)

	// 确保最终清理
	defer func() {
		log.Printf("Cleaning up workspace: %s", repoLocalPath)
		_ = os.RemoveAll(repoLocalPath)
	}()

	// Git Clone
	log.Printf("Cloning %s into %s", t.params.RemoteURL, repoLocalPath)
	if err := t.gitClone(ctx, t.params.RemoteURL, repoLocalPath, t.params.SSHKeyPath); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	// 读取数据库， 随机读取条数
	rdb := db.GetRedisConn()
	mysqlConn := db.GetMysqlConn(db.MYSQL_DB_GO_TASK)
	res := rdb.Get(ctx, LastID)
	if res.Err() != nil {
		if !errors.Is(res.Err(), redis.Nil) {
			return fmt.Errorf("get last id failed: %w", res.Err())
		}
		err := rdb.Set(ctx, LastID, 1, 0).Err()
		if err != nil {
			return fmt.Errorf("set last id failed: %w", err)
		}
	}

	// 标准库，生成1-5之间的随机数
	randNum := rand.Intn(5) + 1

	lastId := res.Val()

	var articles []models.SysArticle
	dbRes := mysqlConn.Model(&models.SysArticle{}).Where("id > ?", lastId).Limit(randNum).Find(&articles)
	if dbRes.Error != nil {
		if errors.Is(dbRes.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("no articles found")
		}
		return fmt.Errorf("query articles failed: %w", dbRes.Error)
	}

	for _, article := range articles {
		log.Printf("🚀 [AutoPushSummarizerTask] Processing article: %s", article.Title)

		input := saveFileInput{
			RepoPath: t.params.WorkDir,
			Author:   t.params.AuthorName,
			Title:    article.Title,
			Content:  article.Summary,
			Source:   article.Source,
			Link:     article.Link,
			Topics:   article.Topics,
		}
		fileName, err := t.saveFile(input)
		if err != nil {
			return fmt.Errorf("save file failed: %w", err)
		}

		// Git 提交并推送
		log.Println("🚀 [AI Task] Pushing changes...")
		if err := t.gitPush(ctx, repoLocalPath, fileName, t.params, t.params.SSHKeyPath); err != nil {
			return fmt.Errorf("git push failed: %w", err)
		}

		rdb.Set(ctx, LastID, article.ID+1, 0)

		// 发布文章
		// 1. 登录信息
		username := "ai_bot"
		password := "admin123"

		// 执行登录获取 Token
		token, err := loginAutoPushSummarizerTask(username, password)
		if err != nil {
			logger.Error("登录失败: %v", zap.Error(err))
			return err
		}

		// 准备要创建的文章数据
		newArticle := AutoPushSummarizerTaskArticleCreateRequest{
			Title:       article.AITitle,
			Content:     article.Summary,
			Summary:     strings.Join(article.Topics, ""),
			Cover:       "/uploads/images/2026/02/12/85a5205c-7c4f-49d3-81db-f542b5d7b502.jpg",
			CategoryID:  23,
			TagIDs:      []int{},
			Status:      1,
			IsTop:       false,
			IsRecommend: false,
		}

		// 执行创建文章
		if err := createAutoPushSummarizerTaskArticle(token, newArticle); err != nil {
			logger.Error("流程终止", zap.Error(err))
			return err
		}

		log.Println("✅ Completed successfully.")
		return nil
	}

	log.Println("No new articles found.")
	return nil
}

// -------------------------------------------------------------------------
// 辅助函数 (Git 操作 & 文件处理)
// -------------------------------------------------------------------------

func (t *AutoPushSummarizerTask) gitClone(ctx context.Context, remoteURL, localPath, sshKeyPath string) error {
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

func (t *AutoPushSummarizerTask) gitPush(ctx context.Context, repoPath, filename string, p AutoPushSummarizerTaskParams, sshKeyPath string) error {
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

type saveFileInput struct {
	RepoPath string
	Author   string
	Title    string
	Content  string
	Source   string
	Link     string
	Topics   []string
}

func (t *AutoPushSummarizerTask) saveFile(input saveFileInput) (string, error) {
	safeTitle := strings.ReplaceAll(input.Title, " ", "_")
	safeTitle = strings.ReplaceAll(safeTitle, "/", "-")
	filename := fmt.Sprintf("%s-%s.md", time.Now().Format("2006-01-02"), safeTitle)

	fullDir := filepath.Join(input.RepoPath, "posts")
	fullPath := filepath.Join(fullDir, filename)

	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return "", err
	}

	// 切片边字符串
	topics := strings.Join(input.Topics, ", ")

	fileContent := fmt.Sprintf("---\ntitle: %s\ntopics: %s\ndate: %s\nLink: %s\nSource: %s\n---\n\n%s",
		input.Title, topics, time.Now().Format(time.RFC3339), input.Link, input.Source, input.Content)

	return filename, os.WriteFile(fullPath, []byte(fileContent), 0644)
}

// ==================== 配置常量 ====================

const (
	BaseURLAutoPushSummarizerTaskTask    = "http://is.iceymoss.com"
	LoginEndpointAutoPushSummarizerTask  = BaseURL + "/api/login"
	CreateEndpointAutoPushSummarizerTask = BaseURL + "/api/articles"
	UserAgentAutoPushSummarizerTask      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36"
)

// 全局 HTTP 客户端，配置了跳过 TLS 验证 (对应 --insecure)
var httpClientAutoPushSummarizerTask = &http.Client{
	Timeout: time.Second * 30,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

// ==================== 核心函数 ====================

// login 执行登录操作并返回 Token
func loginAutoPushSummarizerTask(username, password string) (string, error) {
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
func createAutoPushSummarizerTaskArticle(token string, article AutoPushSummarizerTaskArticleCreateRequest) error {
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

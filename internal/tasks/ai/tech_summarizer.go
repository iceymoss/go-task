package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/iceymoss/go-task/internal/core"
	"github.com/iceymoss/go-task/internal/tasks/base_task"
	"github.com/iceymoss/go-task/pkg/constants"
	"github.com/iceymoss/go-task/pkg/db"
	"github.com/iceymoss/go-task/pkg/db/objects"

	"github.com/mmcdole/gofeed"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"gorm.io/gorm"
)

const (
	aiTecSummarizerTaskName = "ai:tech_summarizer"
)

// TechSummarizerTask 结构体
type TechSummarizerTask struct {
	base_task.BaseTask
}

func NewTechSummarizerTask() core.Task {
	return &TechSummarizerTask{
		BaseTask: base_task.BaseTask{
			Name:     aiTecSummarizerTaskName,
			TaskType: constants.TaskTypeYAML,
		},
	}
}

func (t *TechSummarizerTask) Identifier() string {
	return "ai:tech_summarizer"
}

// SummarizerParams 任务参数
type SummarizerParams struct {
	ApiKey  string   `json:"api_key"`
	BaseURL string   `json:"base_url"`
	Model   string   `json:"model"`
	Sources []string `json:"sources"` // RSS 源列表
}

// AIAnalysisResult  AI 返回的结构体 (用于解析 JSON)
type AIAnalysisResult struct {
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Topics  []string `json:"topics"`
}

func (t *TechSummarizerTask) Run(ctx context.Context, params map[string]any) error {
	p := t.parseParams(params)
	if p.ApiKey == "" {
		return fmt.Errorf("missing api_key")
	}

	fp := gofeed.NewParser()
	dbConn := db.GetMysqlConn(db.MYSQL_DB_GO_TASK)

	// 自动迁移表结构 (为了方便，生产环境建议手动建表)
	_ = dbConn.AutoMigrate(&objects.SysArticle{})

	totalProcessed := 0

	// 1. 遍历所有 RSS 源
	for _, url := range p.Sources {
		log.Printf("🕷️ [Crawler] Fetching: %s", url)
		feed, err := fp.ParseURLWithContext(url, ctx)
		if err != nil {
			log.Printf("⚠️ [Crawler] Failed to parse %s: %v", url, err)
			continue
		}

		fmt.Printf(" address: %s, Rss Feed: %s, total: %d \n", url, feed.Title, len(feed.Items))

		// 2. 遍历文章
		for _, item := range feed.Items {

			fmt.Println(url+" => 文章：", "标题: "+item.Title, "文章地址："+item.Link, "时间：", item.PublishedParsed)

			// 检查时间：只处理最近 24 小时的 (可选)
			if item.PublishedParsed != nil && time.Since(*item.PublishedParsed) > 24*time.Hour {
				continue
			}

			// 3. 去重检查
			hash := t.calculateHash(item.Link) // 使用 Link 做唯一标识
			if t.isDuplicate(dbConn, hash) {
				log.Printf("⏭️ [Crawler] Skip duplicate: %s", item.Title)
				continue
			}

			log.Printf("🤖 [Crawler] Summarizing: %s", item.Title)

			// 4. 调用 AI 进行总结
			// 组合输入内容：标题 + 描述 + 正文(如果有)
			inputText := fmt.Sprintf("标题：%s\n链接：%s\n摘要：%s\n正文：%s",
				item.Title, item.Link, item.Description, item.Content)

			analysis, err := t.callAI(ctx, p, inputText)
			if err != nil {
				log.Printf("❌ [Crawler] AI failed: %v", err)
				continue
			}

			now := time.Now()
			// 5. 存入数据库
			article := &objects.SysArticle{
				Title:           item.Title,
				Link:            item.Link,
				ContentHash:     hash,
				AITitle:         analysis.Title,   // AI 的新标题
				Summary:         analysis.Summary, // AI 的深度总结
				Topics:          analysis.Topics,  // GORM 自动转为 JSON ["Go", "AI"]
				Source:          feed.Title,
				CreatedAt:       now,
				UpdatedAt:       now,
				PublishedParsed: now,
			}
			if err := dbConn.Create(article).Error; err != nil {
				log.Printf("❌ [Crawler] DB Save failed: %v", err)
			} else {
				log.Printf("✅ [Crawler] Saved: %s", item.Title)
				totalProcessed++
			}
		}
	}

	log.Printf("🎉 [Crawler] Task finished. New articles: %d", totalProcessed)
	return nil
}

// -------------------------------------------------------------------------
// AI 深度分析逻辑 (DeepSeek / OpenAI)
// -------------------------------------------------------------------------
func (t *TechSummarizerTask) callAI(ctx context.Context, p SummarizerParams, content string) (*AIAnalysisResult, error) {
	// 1. 截断过长内容 (DeepSeek V3/R1 支持 32k+，但为了省钱和速度，保留前 20000 字符通常足够)
	if len(content) > 20000 {
		content = content[:20000] + "..."
	}

	// 2. 初始化客户端
	llm, err := openai.New(
		openai.WithToken(p.ApiKey),
		openai.WithBaseURL(p.BaseURL),
		openai.WithModel(p.Model),
	)
	if err != nil {
		return nil, err
	}

	// 3. 构造高密度 Prompt
	prompt := fmt.Sprintf(`你是一个资深技术情报专家。请阅读下文，提取关键技术情报并以 JSON 格式输出。

文章内容：
%s

---
任务要求：
1. **重拟标题 (title)**：
   - 必须中文。
   - 即使原标题是英文，也要翻译并润色。
   - 标题要一针见血，包含核心技术关键词（例如："DeepSeek V3 架构详解：如何实现 MoE 负载均衡"）。

2. **深度总结 (summary)**：
   - 必须中文。
   - **拒绝**“这篇文章介绍了...”这种废话。
   - **必须**采用“背景/问题 -> 核心方案/技术细节 -> 价值/影响”的逻辑。
   - **必须**包含具体的编程语言、框架名称、算法名称或架构模式。
   - 字数控制在 500 字左右，要让资深开发者不看原文也能获取 80%% 的信息量。
   - 如果有示例代码，最好保留有代表意义的，这对开发者来说很重要。
   - **【重要】过滤噪音**：如果文章内容主要是关于“币价涨跌预测”、“K线分析”、“空投教程”或“项目软文广告”，请在总结中一笔带过或直接忽略，重点关注**技术创新、行业数据、监管政策、黑客攻击事件**。

3. **技术话题 (topics)**：
   - 提取 1 到 3 个最核心的技术标签（如 ["Go", "Microservices", "K8s"]）。
   - 英文标签优先（如用 "Kubernetes" 而不是 "k8s"）。

4. **输出格式**：
   - 严格返回 JSON 格式：{"title": "...", "summary": "...", "topics": ["tag1", "tag2"]}
   - 不要包含 Markdown 标记（如 '''json）。
`, content)

	// 针对特别硬核的内容，降低 Temperature，防止 AI "一本正经地胡说八道"
	temperature := 0.4
	for _, v := range p.Sources {
		if strings.Contains(v, "vitalik") || strings.Contains(v, "mysql") {
			temperature = 0.1 // 需要绝对的严谨
		}
	}

	// 4. 调用生成
	responseContent, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt,
		llms.WithTemperature(temperature), // 稍微降低温度，保证 JSON 格式稳定
	)
	if err != nil {
		return nil, err
	}

	// 5. 清洗与解析
	cleanJSON := strings.TrimSpace(responseContent)
	// 容错处理：去掉可能存在的 Markdown 代码块标记
	cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
	cleanJSON = strings.TrimPrefix(cleanJSON, "```")
	cleanJSON = strings.TrimSuffix(cleanJSON, "```")

	var result AIAnalysisResult
	if err := json.Unmarshal([]byte(cleanJSON), &result); err != nil {
		log.Printf("⚠️ JSON parse failed, raw content: %s", cleanJSON)
		// 降级策略：如果 JSON 解析失败，把整个内容当作总结，标题用默认的
		return &AIAnalysisResult{
			Title:   "AI解析失败-原标题",
			Summary: responseContent,
			Topics:  []string{"ParseError"},
		}, nil
	}

	return &result, nil
}

// -------------------------------------------------------------------------
// 辅助函数
// -------------------------------------------------------------------------

// calculateHash 计算 SHA256 哈希
func (t *TechSummarizerTask) calculateHash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// isDuplicate 检查是否已存在
func (t *TechSummarizerTask) isDuplicate(db *gorm.DB, hash string) bool {
	var count int64
	db.Model(&objects.SysArticle{}).Where("content_hash = ?", hash).Count(&count)
	return count > 0
}

func (t *TechSummarizerTask) parseParams(params map[string]any) SummarizerParams {
	p := SummarizerParams{
		BaseURL: "https://api.deepseek.com",
		Model:   "deepseek-chat", // 总结任务用 V3 (chat) 足够且便宜，R1 有点大材小用且慢
		Sources: []string{
			"https://go.dev/blog/feed.atom",    // Go 官方博客
			"https://news.ycombinator.com/rss", // Hacker News
			"https://openai.com/blog/rss.xml",  // OpenAI Blog
			"https://github.blog/feed/",        // GitHub Blog
		},
	}

	getString := func(k string) string {
		if v, ok := params[k].(string); ok && v != "" {
			return v
		}
		return ""
	}

	if v := getString("api_key"); v != "" {
		p.ApiKey = v
	}
	if v := getString("base_url"); v != "" {
		p.BaseURL = v
	}
	if v := getString("model"); v != "" {
		p.Model = v
	}

	// 解析自定义源列表 (如果 YAML 里配了的话)
	if v, ok := params["sources"].([]interface{}); ok {
		var customSources []string
		for _, s := range v {
			if str, ok := s.(string); ok {
				customSources = append(customSources, str)
			}
		}
		if len(customSources) > 0 {
			p.Sources = customSources
		}
	}

	return p
}

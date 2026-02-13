package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// -------------------------------------------------------------------------
// 1. 数据结构定义 (与 Prompt 的 JSON 输出严格对应)
// -------------------------------------------------------------------------

// ArticleMeta 对应 JSON 中的 "meta" 字段
type ArticleMeta struct {
	Topic       string `json:"topic"`
	Difficulty  string `json:"difficulty"`
	ReadingTime string `json:"estimated_reading_time"`
}

// GeneratedArticle 对应 AI 返回的完整 JSON 结构
type GeneratedArticle struct {
	Meta    ArticleMeta `json:"meta"`
	Title   string      `json:"title"`
	Summary string      `json:"summary"`
	Content string      `json:"content"`
}

// WriterParams 保持不变，用于传参
type WriterParams struct {
	ApiKey    string `json:"api_key"`
	BaseURL   string `json:"base_url"`
	Model     string `json:"model"`
	Topic     string `json:"topic"`
	MaxTokens int    `json:"max_tokens"` // 新增：控制生成长度
}

// -------------------------------------------------------------------------
// 2. 核心逻辑
// -------------------------------------------------------------------------

// generateArticle 负责调用 AI 并解析结果
func generateArticle(ctx context.Context, p WriterParams) (*GeneratedArticle, error) {
	// 2.1 初始化 Client
	llm, err := openai.New(
		openai.WithToken(p.ApiKey),
		openai.WithBaseURL(p.BaseURL),
		openai.WithModel(p.Model),
	)
	if err != nil {
		return nil, fmt.Errorf("init llm client failed: %w", err)
	}

	// 2.2 构造 Prompt (使用常量或独立函数，保持代码整洁)
	prompt := buildAgentPrompt(p.Topic)

	// 2.3 设置超时上下文 (生成长文可能需要较长时间，例如 5 分钟)
	genCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	log.Printf("🤖 AI is thinking about '%s' (Model: %s)...", p.Topic, p.Model)

	// 2.4 调用生成
	// 注意：MaxTokens 设置大一点，防止长文被截断
	respContent, err := llms.GenerateFromSinglePrompt(genCtx, llm, prompt,
		llms.WithTemperature(0.6),
		llms.WithMaxTokens(p.MaxTokens), // 如果模型支持，设置为 8192 或更高
	)

	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}

	// 2.5 智能清洗 JSON (这是最关键的一步优化)
	cleanJSON := extractJSON(respContent)

	// 2.6 解析结果
	var article GeneratedArticle
	if err := json.Unmarshal([]byte(cleanJSON), &article); err != nil {
		// 记录原始返回以便调试
		log.Printf("⚠️ JSON parse failed. Raw content head: %s...", getHead(cleanJSON, 100))
		return nil, fmt.Errorf("json unmarshal failed: %w", err)
	}

	return &article, nil
}

func extractJSON(s string) string {
	// 1. 处理常见的 Markdown 包装
	s = strings.TrimSpace(s)

	// 移除可能存在的 Markdown 标签（如 ```json ... ```）
	re := regexp.MustCompile(`(?s)^` + "```" + `(?:json)?\s*(.*?)\s*` + "```" + `$`)
	if matches := re.FindStringSubmatch(s); len(matches) > 1 {
		s = matches[1]
	}

	// 2. 精准定位最外层括号
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")

	if start == -1 || end == -1 || start >= end {
		return s
	}

	s = s[start : end+1]

	// 3. 关键清洗：处理 AI 偶尔产生的非法控制字符
	// 有些模型在输出长文时，JSON 内部的换行符处理不规范
	// 我们只保留标准的 JSON 字符，移除那些可能导致解析失败的控制字符
	s = strings.Map(func(r rune) rune {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			return -1
		}
		return r
	}, s)

	return s
}

// buildAgentPrompt 封装 Prompt 构造逻辑
func buildAgentPrompt(topic string) string {
	// 这里直接使用你优化好的 Prompt
	return fmt.Sprintf(`
# Role: 资深技术专家 Agent (Principal Engineer & Tech Writer)

# Mission
你现在的任务是针对主题 **“%s”** 撰写一篇教科书级的技术长文。
你需要模拟一个完整的【技术写作工作流】，从底层原理讲到生产实践。

# Agent Workflow (内部执行步骤)
1. **深度分析**: 拆解该技术的核心痛点、底层原理（源码级）、应用场景。
2. **大纲构建**: 设计一个面向 初级->中级->高级 的渐进式结构。
3. **代码编写**: 编写 Production-Ready（生产环境可用）的代码示例，拒绝 Hello World 级别的玩具代码。
4. **自我审查**: 检查内容是否包含“废话”、“营销话术”，必须确保每一段都是干货。
5. **格式输出**: 将最终结果封装为严格的 JSON。
6. **输出语言**: 必须使用简体中文。

# Content Guidelines (内容准则)
- **深度要求**: 必须包含 "What" (是什么), "Why" (为什么设计成这样), "How" (最佳实践), "Anti-Patterns" (反模式/避坑指南)。
- **代码要求**: 代码必须符合惯用语法(Idiomatic)，包含详细注释，解释关键设计决策。
- **语气风格**: 务实、谦虚、严谨。像一个老同事在 Code Review 时给你讲干货，而不是像教科书那样死板，也不是营销号那样浮夸。
- **篇幅**: 尽可能详尽，覆盖该主题的方方面面，目标是成为该主题在中文互联网上的 "Definitive Guide" (终极指南)。

# Output Format (严格遵守)
### 你必须返回JSON的数据结构，结构定义：
{
  "meta": {
    "topic": "文章主题",
    "difficulty": "Advanced",
    "estimated_reading_time": "30min+"
  },
  "title": "极具吸引力且专业的主题",
  "summary": "文章的简短摘要（200字以内）",
  "content": "这里是完整的 Markdown 格式正文，包含所有章节、代码块和详细说明..."
}
你必须只返回一个可以直接被 json.Unmarshal 解析的 JSON 字符串。
content 字段的值必须是经过 JSON 字符串转义 的文本。确保所有的换行符转为 \n，双引号转为 \"。确保输出是一个合法的、单行的或标准格式的 JSON，没有任何前缀或后缀。
不要包含 Markdown 标记（如 '''json ...），不要包含任何开场白。

`, topic)
}

// getHead 获取字符串前 n 个字符用于日志
func getHead(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		return string(runes[:n])
	}
	return s
}

// -------------------------------------------------------------------------
// 4. Main 入口
// -------------------------------------------------------------------------

func main() {
	// 配置参数
	params := WriterParams{
		ApiKey:    "037582ceb2e",
		BaseURL:   "https://open.bigmodel.cn/api/paas/v4/", // 智谱地址
		Model:     "glm-4-plus",                            // 推荐使用 Plus 写长文
		Topic:     "万字长文：带您入门rust",
		MaxTokens: 10000, // 根据模型能力调整，GLM-4 支持很长
	}

	start := time.Now()
	article, err := generateArticle(context.Background(), params)
	if err != nil {
		log.Fatalf("❌ Generation failed: %v", err)
	}

	duration := time.Since(start)

	// 输出结果
	fmt.Println("--------------------------------------------------")
	fmt.Printf("✅ Generated successfully in %s\n", duration)
	fmt.Printf("📌 Title:   %s\n", article.Title)
	fmt.Printf("📊 Meta:    Difficulty: %s | Time: %s\n", article.Meta.Difficulty, article.Meta.ReadingTime)
	fmt.Printf("📝 Summary: %s\n", article.Summary)
	fmt.Println("--------------------------------------------------")

	// 这里可以将 article.Content 写入文件
	// os.WriteFile("output.md", []byte(article.Content), 0644)
	//fmt.Printf("\n(Content snippet): %s...\n", getHead(article.Content, 200))
	log.Println("📝 Article content:", article.Content)
}

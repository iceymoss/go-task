package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
)

// 1. 定义 JSON 树的节点结构 (SchemaNode)
// 这对应了 schema.json 中的每一个 UI 块
type SchemaNode struct {
	Type     string         `json:"type"`               // 组件类型，如 "Container", "Header", "Button"
	Props    map[string]any `json:"props"`              // 组件的动态属性，如颜色、文字、样式类
	Children []*SchemaNode  `json:"children,omitempty"` // 嵌套的子组件列表
}

// 2. 定义组件渲染器函数签名
// 接收当前组件的 Props 和已经渲染好的子节点 HTML 字符串，返回自身的 HTML
type ComponentRenderer func(props map[string]any, childrenHtml string) (string, error)

// 3. 建立组件注册表 (Component Registry)
// 就像一个施工队的仓库，存放着各种处理不同组件的方法
var componentRegistry = make(map[string]ComponentRenderer)

func init() {
	// 注册 Container 组件 (通常用于布局，会包裹子元素)
	componentRegistry["Container"] = func(props map[string]any, childrenHtml string) (string, error) {
		class, _ := props["className"].(string)
		// 将 childrenHtml 嵌入到 div 内部
		return fmt.Sprintf("<div class=\"%s\">\n%s\n</div>", class, childrenHtml), nil
	}

	// 注册 Header 组件 (叶子节点，通常没有子元素)
	componentRegistry["Header"] = func(props map[string]any, childrenHtml string) (string, error) {
		title, _ := props["title"].(string)
		return fmt.Sprintf("  <h1>%s</h1>", title), nil
	}

	// 注册 Button 组件
	componentRegistry["Button"] = func(props map[string]any, childrenHtml string) (string, error) {
		label, _ := props["label"].(string)
		action, _ := props["action"].(string)
		return fmt.Sprintf("  <button onclick=\"%s\">%s</button>", action, label), nil
	}
}

// 4. 核心：递归渲染逻辑 (Renderer)
func RenderSchema(node *SchemaNode) (string, error) {
	if node == nil {
		return "", nil
	}

	// 第一步：自底向上，先递归渲染所有的子节点 (Children)
	var childrenHtmlBuf bytes.Buffer
	for _, child := range node.Children {
		childHtml, err := RenderSchema(child)
		if err != nil {
			return "", fmt.Errorf("failed to render child node: %w", err)
		}
		childrenHtmlBuf.WriteString(childHtml)
		childrenHtmlBuf.WriteString("\n")
	}

	// 第二步：去注册表里寻找当前节点类型对应的“渲染器”
	renderer, exists := componentRegistry[node.Type]
	if !exists {
		// 找不到对应的组件时，可以做降级处理或直接忽略
		log.Printf("warning: unknown component type '%s'", node.Type)
		return fmt.Sprintf("", node.Type), nil
	}

	// 第三步：执行渲染，将属性(Props)和已经组装好的子节点HTML(ChildrenHtml)传进去
	return renderer(node.Props, childrenHtmlBuf.String())
}

func main() {
	// 模拟从数据库或 Wafer 框架拿到的 JSON Schema 字符串
	schemaJSON := `
	{
		"type": "Container",
		"props": { "className": "shop-main-layout" },
		"children": [
			{
				"type": "Header",
				"props": { "title": "Welcome to My Store" }
			},
			{
				"type": "Container",
				"props": { "className": "actions-group" },
				"children": [
					{
						"type": "Button",
						"props": { "label": "Buy Now", "action": "checkout()" }
					}
				]
			}
		]
	}`

	// 解析 JSON 到结构体
	var rootNode SchemaNode
	if err := json.Unmarshal([]byte(schemaJSON), &rootNode); err != nil {
		log.Fatalf("JSON parse error: %v", err)
	}

	// 启动渲染引擎
	htmlOutput, err := RenderSchema(&rootNode)
	if err != nil {
		log.Fatalf("Render error: %v", err)
	}

	// 输出最终的精装房 HTML
	fmt.Println(htmlOutput)
}

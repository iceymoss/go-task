package web

import "embed"

//go:embed *
var StaticFiles embed.FS // 必须在 main 包或同级目录，路径要是相对路径

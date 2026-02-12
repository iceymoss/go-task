package objects

import (
	"time"

	"gorm.io/gorm"
)

// SysArticle 对应数据库表 sys_articles
// 用于存储爬虫获取的技术文章及 AI 总结
type SysArticle struct {
	// ID 主键
	ID uint64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 标题
	Title string `gorm:"type:varchar(255);not null" json:"title"`

	// 原文链接
	Link string `gorm:"type:varchar(512);not null" json:"link"`

	// 内容哈希 (用于去重)
	// uniqueIndex:idx_hash 对应 SQL 中的 UNIQUE KEY `idx_hash`
	ContentHash string `gorm:"type:varchar(64);not null;uniqueIndex:idx_hash;comment:标题或链接的Hash，用于去重" json:"content_hash"`

	// 来源 (例如: GoBlog, HackerNews)
	Source string `gorm:"type:varchar(64);comment:来源网站，如 GoBlog" json:"source"`

	AITitle string `gorm:"size:255;comment:AI重拟的标题"`
	Summary string `gorm:"type:text;comment:AI深度总结"`

	// 使用 GORM 的序列化功能，自动将 []string 转为 JSON 字符串存入数据库
	Topics []string `gorm:"serializer:json;type:json;comment:技术话题标签"`

	// 创建时间
	// autoCreateTime 会在创建时自动填入当前时间
	CreatedAt       time.Time `gorm:"autoCreateTime;type:datetime" json:"created_at"`
	PublishedParsed time.Time `gorm:"index;comment:文章发布时间"`
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名
func (SysArticle) TableName() string {
	return "sys_articles"
}

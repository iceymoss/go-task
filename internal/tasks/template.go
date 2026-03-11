package tasks

import (
	"encoding/json"
	"fmt"
	"strings"
)

// JobTemplate 任务模板
type JobTemplate struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Type        string         `json:"type"`
	CronExpr    string         `json:"cron_expr"`
	Params      map[string]any `json:"params"`
	Variables   []TemplateVar  `json:"variables"`
}

// TemplateVar 模板变量
type TemplateVar struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Type        string `json:"type"` // string, number, boolean
	Default     string `json:"default"`
	Description string `json:"description"`
}

// 预设任务模板
var templates = map[string]JobTemplate{
	"backup_database": {
		ID:          "backup_database",
		Name:        "数据库备份",
		Description: "定期备份数据库到指定目录",
		Type:        "shell",
		CronExpr:    "0 2 * * *", // 每天凌晨2点
		Params: map[string]any{
			"command":     "mysqldump -u {{username}} -p{{password}} {{database}} > /backup/{{database}}_{{date}}.sql",
			"working_dir": "/backup",
		},
		Variables: []TemplateVar{
			{
				Name:        "username",
				Label:       "数据库用户名",
				Type:        "string",
				Default:     "root",
				Description: "MySQL用户名",
			},
			{
				Name:        "password",
				Label:       "数据库密码",
				Type:        "string",
				Default:     "",
				Description: "MySQL密码",
			},
			{
				Name:        "database",
				Label:       "数据库名称",
				Type:        "string",
				Default:     "myapp",
				Description: "要备份的数据库名",
			},
			{
				Name:        "date",
				Label:       "日期标识",
				Type:        "string",
				Default:     "{{date}}",
				Description: "备份文件日期（会自动生成）",
			},
		},
	},
	"clean_logs": {
		ID:          "clean_logs",
		Name:        "日志清理",
		Description: "定期清理过期日志文件",
		Type:        "shell",
		CronExpr:    "0 3 * * 0", // 每周日凌晨3点
		Params: map[string]any{
			"command": "find {{log_dir}} -name \"{{log_pattern}}\" -mtime +{{days}} -delete",
		},
		Variables: []TemplateVar{
			{
				Name:        "log_dir",
				Label:       "日志目录",
				Type:        "string",
				Default:     "/var/log/myapp",
				Description: "日志文件所在目录",
			},
			{
				Name:        "log_pattern",
				Label:       "日志文件模式",
				Type:        "string",
				Default:     "*.log",
				Description: "匹配的文件模式",
			},
			{
				Name:        "days",
				Label:       "保留天数",
				Type:        "number",
				Default:     "7",
				Description: "保留多少天的日志",
			},
		},
	},
	"health_check": {
		ID:          "health_check",
		Name:        "健康检查",
		Description: "定期检查服务健康状态",
		Type:        "http",
		CronExpr:    "*/5 * * * *", // 每5分钟
		Params: map[string]any{
			"url":             "http://{{host}}:{{port}}/health",
			"method":          "GET",
			"expected_status": 200,
			"timeout":         30,
		},
		Variables: []TemplateVar{
			{
				Name:        "host",
				Label:       "主机地址",
				Type:        "string",
				Default:     "localhost",
				Description: "服务主机地址",
			},
			{
				Name:        "port",
				Label:       "端口",
				Type:        "number",
				Default:     "8080",
				Description: "服务端口",
			},
		},
	},
	"send_report": {
		ID:          "send_report",
		Name:        "发送日报",
		Description: "每天早上发送统计日报",
		Type:        "email",
		CronExpr:    "0 8 * * 1-5", // 工作日早上8点
		Params: map[string]any{
			"to":      []string{"{{recipient}}"},
			"subject": "每日统计报告 - {{date}}",
			"body":    "{{report_content}}",
			"is_html": true,
		},
		Variables: []TemplateVar{
			{
				Name:        "recipient",
				Label:       "收件人",
				Type:        "string",
				Default:     "user@example.com",
				Description: "报告接收邮箱",
			},
			{
				Name:        "date",
				Label:       "日期",
				Type:        "string",
				Default:     "{{date}}",
				Description: "报告日期",
			},
			{
				Name:        "report_content",
				Label:       "报告内容",
				Type:        "string",
				Default:     "",
				Description: "报告HTML内容",
			},
		},
	},
	"data_cleanup": {
		ID:          "data_cleanup",
		Name:        "数据清理",
		Description: "清理过期数据",
		Type:        "sql",
		CronExpr:    "0 4 * * *", // 每天凌晨4点
		Params: map[string]any{
			"query":    "DELETE FROM {{table}} WHERE created_at < DATE_SUB(NOW(), INTERVAL {{days}} DAY)",
			"database": "mysql",
		},
		Variables: []TemplateVar{
			{
				Name:        "table",
				Label:       "表名",
				Type:        "string",
				Default:     "logs",
				Description: "要清理的表名",
			},
			{
				Name:        "days",
				Label:       "保留天数",
				Type:        "number",
				Default:     "30",
				Description: "保留多少天的数据",
			},
		},
	},
	"monitor_disk": {
		ID:          "monitor_disk",
		Name:        "磁盘监控",
		Description: "监控磁盘使用率，超过阈值发送告警",
		Type:        "shell",
		CronExpr:    "*/30 * * * *", // 每30分钟
		Params: map[string]any{
			"command":     "df -h {{mount_point}} | awk 'NR==2 {print $5}' | sed 's/%//'",
			"working_dir": "/",
		},
		Variables: []TemplateVar{
			{
				Name:        "mount_point",
				Label:       "挂载点",
				Type:        "string",
				Default:     "/",
				Description: "要监控的磁盘挂载点",
			},
		},
	},
	"restart_service": {
		ID:          "restart_service",
		Name:        "重启服务",
		Description: "定时重启指定服务",
		Type:        "shell",
		CronExpr:    "0 3 * * 0", // 每周日凌晨3点
		Params: map[string]any{
			"command":     "systemctl restart {{service_name}} && echo 'Service restarted successfully'",
			"working_dir": "/",
		},
		Variables: []TemplateVar{
			{
				Name:        "service_name",
				Label:       "服务名称",
				Type:        "string",
				Default:     "nginx",
				Description: "要重启的服务名称",
			},
		},
	},
	"api_sync": {
		ID:          "api_sync",
		Name:        "API数据同步",
		Description: "从外部API同步数据到本地",
		Type:        "http",
		CronExpr:    "0 */2 * * *", // 每2小时
		Params: map[string]any{
			"url":             "{{api_url}}",
			"method":          "GET",
			"headers":         map[string]string{"Authorization": "Bearer {{token}}"},
			"expected_status": 200,
			"timeout":         60,
		},
		Variables: []TemplateVar{
			{
				Name:        "api_url",
				Label:       "API地址",
				Type:        "string",
				Default:     "https://api.example.com/data",
				Description: "要同步的API地址",
			},
			{
				Name:        "token",
				Label:       "API Token",
				Type:        "string",
				Default:     "",
				Description: "API认证Token",
			},
		},
	},
	"generate_report": {
		ID:          "generate_report",
		Name:        "生成报表",
		Description: "生成业务数据报表",
		Type:        "shell",
		CronExpr:    "0 6 * * *", // 每天早上6点
		Params: map[string]any{
			"command":     "python /scripts/generate_report.py --date {{report_date}} --output /reports/{{report_name}}.pdf",
			"working_dir": "/",
		},
		Variables: []TemplateVar{
			{
				Name:        "report_date",
				Label:       "报表日期",
				Type:        "string",
				Default:     "{{date}}",
				Description: "报表日期",
			},
			{
				Name:        "report_name",
				Label:       "报表名称",
				Type:        "string",
				Default:     "daily_report",
				Description: "报表文件名",
			},
		},
	},
	"cache_clear": {
		ID:          "cache_clear",
		Name:        "缓存清理",
		Description: "清理应用缓存",
		Type:        "shell",
		CronExpr:    "0 */6 * * *", // 每6小时
		Params: map[string]any{
			"command":     "rm -rf {{cache_dir}}/* && echo 'Cache cleared successfully'",
			"working_dir": "/",
		},
		Variables: []TemplateVar{
			{
				Name:        "cache_dir",
				Label:       "缓存目录",
				Type:        "string",
				Default:     "/var/cache/myapp",
				Description: "缓存文件所在目录",
			},
		},
	},
	"send_notification": {
		ID:          "send_notification",
		Name:        "发送通知",
		Description: "发送系统通知",
		Type:        "email",
		CronExpr:    "0 9 * * 1-5", // 工作日早上9点
		Params: map[string]any{
			"to":      []string{"{{recipients}}"},
			"subject": "{{subject}}",
			"body":    "{{content}}",
			"is_html": false,
		},
		Variables: []TemplateVar{
			{
				Name:        "recipients",
				Label:       "收件人（逗号分隔）",
				Type:        "string",
				Default:     "admin@example.com",
				Description: "通知接收邮箱",
			},
			{
				Name:        "subject",
				Label:       "主题",
				Type:        "string",
				Default:     "系统通知",
				Description: "邮件主题",
			},
			{
				Name:        "content",
				Label:       "内容",
				Type:        "string",
				Default:     "",
				Description: "通知内容",
			},
		},
	},
}

// GetJobTemplates 获取所有模板
func GetJobTemplates() []JobTemplate {
	result := make([]JobTemplate, 0, len(templates))
	for _, t := range templates {
		result = append(result, t)
	}
	return result
}

// GetJobTemplate 根据ID获取模板
func GetJobTemplate(id string) (JobTemplate, error) {
	t, ok := templates[id]
	if !ok {
		return JobTemplate{}, fmt.Errorf("template not found: %s", id)
	}
	return t, nil
}

// ApplyTemplateVariables 应用变量替换
func ApplyTemplateVariables(template JobTemplate, variables map[string]string) string {
	// 将 params 转换为 JSON 字符串
	paramsJSON, _ := json.Marshal(template.Params)

	// 替换变量
	result := string(paramsJSON)
	for key, value := range variables {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// 替换特殊变量（如日期）
	if strings.Contains(result, "{{date}}") {
		// 这里可以生成实际日期，简化处理
		result = strings.ReplaceAll(result, "{{date}}", "2024-01-01")
	}

	return result
}

// AddTemplate 添加自定义模板
func AddTemplate(template JobTemplate) {
	templates[template.ID] = template
}

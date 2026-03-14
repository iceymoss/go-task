package email

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/iceymoss/go-task/internal/core"
	"github.com/iceymoss/go-task/internal/tasks/base_task"
	"github.com/iceymoss/go-task/pkg/constants"
	"github.com/iceymoss/go-task/pkg/logger"

	"go.uber.org/zap"
)

const TaskName = "email:email"

// EmailTask 邮件发送任务
type EmailTask struct {
	base_task.BaseTask
}

func NewEmailTask() core.Task {
	return &EmailTask{
		BaseTask: base_task.BaseTask{
			Name:     TaskName,
			TaskType: constants.TaskTypeSYSTEM,
		},
	}
}

// EmailParams 参数结构
type EmailParams struct {
	To      []string `json:"to" binding:"required"`
	CC      []string `json:"cc"`
	BCC     []string `json:"bcc"`
	Subject string   `json:"subject" binding:"required"`
	Body    string   `json:"body" binding:"required"`
	IsHTML  bool     `json:"is_html"`
}

func (t *EmailTask) Run(ctx context.Context, params map[string]any) error {
	// 解析参数
	p := parseParams(params)
	if len(p.To) == 0 {
		return fmt.Errorf("to is required")
	}
	if p.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	if p.Body == "" {
		return fmt.Errorf("body is required")
	}

	logger.Info("🚀 [EmailTask] Sending email",
		zap.Strings("to", p.To),
		zap.String("subject", p.Subject),
	)

	// 发送邮件
	if err := t.sendEmail(p); err != nil {
		logger.Error("❌ [EmailTask] Failed to send email",
			zap.Strings("to", p.To),
			zap.Error(err),
		)
		return fmt.Errorf("failed to send email: %w", err)
	}

	logger.Info("✅ [EmailTask] Email sent successfully",
		zap.Strings("to", p.To),
	)

	return nil
}

// sendEmail 发送邮件
func (t *EmailTask) sendEmail(p EmailParams) error {
	// 从配置获取邮件服务器信息（这里简化处理，实际应该从配置读取）
	// 暂时使用环境变量或默认值
	host := "smtp.example.com"
	port := "587"
	username := "user@example.com"
	password := "password"

	// 构建邮件头部
	headers := make(map[string]string)
	headers["From"] = username
	headers["To"] = strings.Join(p.To, ",")
	if len(p.CC) > 0 {
		headers["Cc"] = strings.Join(p.CC, ",")
	}
	if len(p.BCC) > 0 {
		headers["Bcc"] = strings.Join(p.BCC, ",")
	}
	headers["Subject"] = p.Subject
	headers["MIME-Version"] = "1.0"

	contentType := "text/plain; charset=UTF-8"
	if p.IsHTML {
		contentType = "text/html; charset=UTF-8"
	}
	headers["Content-Type"] = contentType

	// 构建邮件内容
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + p.Body

	// 身份验证
	auth := smtp.PlainAuth("", username, password, host)

	// 发送邮件
	err := smtp.SendMail(
		host+":"+port,
		auth,
		username,
		p.To,
		[]byte(message),
	)

	return err
}

func parseParams(params map[string]any) EmailParams {
	p := EmailParams{
		IsHTML: false,
	}

	if v, ok := params["to"].([]string); ok {
		p.To = v
	} else if v, ok := params["to"].(string); ok {
		// 支持逗号分隔的字符串
		p.To = strings.Split(v, ",")
	}

	if v, ok := params["cc"].([]string); ok {
		p.CC = v
	} else if v, ok := params["cc"].(string); ok {
		p.CC = strings.Split(v, ",")
	}

	if v, ok := params["bcc"].([]string); ok {
		p.BCC = v
	} else if v, ok := params["bcc"].(string); ok {
		p.BCC = strings.Split(v, ",")
	}

	if v, ok := params["subject"].(string); ok {
		p.Subject = v
	}
	if v, ok := params["body"].(string); ok {
		p.Body = v
	}
	if v, ok := params["is_html"].(bool); ok {
		p.IsHTML = v
	}

	return p
}

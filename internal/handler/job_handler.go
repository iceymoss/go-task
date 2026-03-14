package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/iceymoss/go-task/internal/engine"
	"github.com/iceymoss/go-task/internal/tasks"
	"github.com/iceymoss/go-task/pkg/constants"
	"github.com/iceymoss/go-task/pkg/db/models"
	"github.com/iceymoss/go-task/pkg/db/objects"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// JobHandler 任务管理处理器
type JobHandler struct {
	db        *gorm.DB
	scheduler *engine.Scheduler
}

// NewJobHandler 创建任务处理器
func NewJobHandler(db *gorm.DB, scheduler *engine.Scheduler) *JobHandler {
	return &JobHandler{
		db:        db,
		scheduler: scheduler,
	}
}

// CreateJobRequest 创建任务请求
type CreateJobRequest struct {
	Name         string         `json:"name" binding:"required"`
	DisplayName  string         `json:"display_name" binding:"required"`
	Type         string         `json:"type" binding:"required"`
	CronExpr     string         `json:"cron_expr" binding:"required"`
	Params       map[string]any `json:"params"`
	Enable       bool           `json:"enable"`
	Dependencies []string       `json:"dependencies"`
	Priority     int            `json:"priority"`
	Timeout      int            `json:"timeout"`
	MaxRetries   int            `json:"max_retries"`
	Description  string         `json:"description"`
	Tags         []string       `json:"tags"`
}

// UpdateJobRequest 更新任务请求
type UpdateJobRequest struct {
	DisplayName  *string        `json:"display_name"`
	Type         *string        `json:"type"`
	CronExpr     *string        `json:"cron_expr"`
	Params       map[string]any `json:"params"`
	Enable       *bool          `json:"enable"`
	Dependencies []string       `json:"dependencies"`
	Priority     *int           `json:"priority"`
	Timeout      *int           `json:"timeout"`
	MaxRetries   *int           `json:"max_retries"`
	Description  *string        `json:"description"`
	Tags         []string       `json:"tags"`
}

// JobResponse 任务响应
type JobResponse struct {
	ID           uint           `json:"id"`
	Name         string         `json:"name"`
	DisplayName  string         `json:"display_name"`
	Type         string         `json:"type"`
	CronExpr     string         `json:"cron_expr"`
	Enable       bool           `json:"enable"`
	Source       string         `json:"source"`
	Params       map[string]any `json:"params"`
	Dependencies []string       `json:"dependencies"`
	Priority     int            `json:"priority"`
	Timeout      int            `json:"timeout"`
	MaxRetries   int            `json:"max_retries"`
	Description  string         `json:"description"`
	Tags         []string       `json:"tags"`
	Status       string         `json:"status"`
	LastRunAt    *time.Time     `json:"last_run_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// GetJobs 获取任务列表
func (h *JobHandler) GetJobs(c *gin.Context) {
	var jobs []models.Job
	query := h.db.Model(&models.Job{})

	// 支持筛选
	if jobType := c.Query("type"); jobType != "" {
		query = query.Where("type = ?", jobType)
	}
	if enable := c.Query("enable"); enable != "" {
		if enable == "true" {
			query = query.Where("enable = ?", true)
		} else {
			query = query.Where("enable = ?", false)
		}
	}

	if err := query.Order("created_at DESC").Find(&jobs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	responses := make([]JobResponse, len(jobs))
	for i, job := range jobs {
		responses[i] = h.jobToResponse(&job)
	}

	c.JSON(http.StatusOK, gin.H{"data": responses})
}

// GetJob 获取任务详情
func (h *JobHandler) GetJob(c *gin.Context) {
	id := c.Param("id")
	var job models.Job
	if err := h.db.First(&job, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": h.jobToResponse(&job)})
}

// CreateJob 创建任务
func (h *JobHandler) CreateJob(c *gin.Context) {
	var req CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证任务类型
	if !isValidTaskType(req.Type) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid task type: %s", req.Type)})
		return
	}

	// 验证 Cron 表达式
	if _, err := parseCronExpr(req.CronExpr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid cron expression: %v", err)})
		return
	}

	// 检查任务名称是否已存在
	var existingJob models.Job
	if err := h.db.Where("name = ?", req.Name).First(&existingJob).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "job name already exists"})
		return
	}

	// 转换参数
	paramsJSON, _ := json.Marshal(req.Params)
	depsJSON, _ := json.Marshal(req.Dependencies)
	tagsJSON, _ := json.Marshal(req.Tags)

	// 创建任务记录
	job := &models.Job{
		Name:         req.Name,
		DisplayName:  req.DisplayName,
		Type:         req.Type,
		CronExpr:     req.CronExpr,
		Params:       string(paramsJSON),
		Enable:       req.Enable,
		Dependencies: string(depsJSON),
		Priority:     req.Priority,
		Timeout:      req.Timeout,
		MaxRetries:   req.MaxRetries,
		Description:  req.Description,
		Tags:         string(tagsJSON),
		Source:       string(constants.TaskTypeWEB),
	}

	if err := h.db.Create(job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 动态添加到调度器
	if job.Enable {
		if err := h.scheduler.AddJob(
			job.CronExpr,
			job.Name,
			job.Name,
			req.Params,
			job.Source,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to add job to scheduler: %v", err)})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{"data": h.jobToResponse(job)})
}

// UpdateJob 更新任务
func (h *JobHandler) UpdateJob(c *gin.Context) {
	id := c.Param("id")
	var job models.Job
	if err := h.db.First(&job, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var req UpdateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证任务类型
	if req.Type != nil && !isValidTaskType(*req.Type) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid task type: %s", *req.Type)})
		return
	}

	// 验证 Cron 表达式
	if req.CronExpr != nil {
		if _, err := parseCronExpr(*req.CronExpr); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid cron expression: %v", err)})
			return
		}
	}

	// 更新字段
	if req.DisplayName != nil {
		job.DisplayName = *req.DisplayName
	}
	if req.Type != nil {
		job.Type = *req.Type
	}
	if req.CronExpr != nil {
		job.CronExpr = *req.CronExpr
	}
	if req.Params != nil {
		paramsJSON, _ := json.Marshal(req.Params)
		job.Params = string(paramsJSON)
	}
	if req.Enable != nil {
		job.Enable = *req.Enable
	}
	if req.Dependencies != nil {
		depsJSON, _ := json.Marshal(req.Dependencies)
		job.Dependencies = string(depsJSON)
	}
	if req.Priority != nil {
		job.Priority = *req.Priority
	}
	if req.Timeout != nil {
		job.Timeout = *req.Timeout
	}
	if req.MaxRetries != nil {
		job.MaxRetries = *req.MaxRetries
	}
	if req.Description != nil {
		job.Description = *req.Description
	}
	if req.Tags != nil {
		tagsJSON, _ := json.Marshal(req.Tags)
		job.Tags = string(tagsJSON)
	}

	if err := h.db.Save(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 重新加载到调度器
	if err := h.reloadJobToScheduler(&job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to reload job: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": h.jobToResponse(&job)})
}

// DeleteJob 删除任务
func (h *JobHandler) DeleteJob(c *gin.Context) {
	id := c.Param("id")
	var job models.Job
	if err := h.db.First(&job, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 软删除
	if err := h.db.Delete(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "job deleted successfully"})
}

// EnableJob 启用任务
func (h *JobHandler) EnableJob(c *gin.Context) {
	id := c.Param("id")
	var job models.Job
	if err := h.db.First(&job, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	job.Enable = true
	if err := h.db.Save(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 添加到调度器
	if err := h.reloadJobToScheduler(&job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to enable job: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": h.jobToResponse(&job)})
}

// DisableJob 禁用任务
func (h *JobHandler) DisableJob(c *gin.Context) {
	id := c.Param("id")
	var job models.Job
	if err := h.db.First(&job, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	job.Enable = false
	if err := h.db.Save(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": h.jobToResponse(&job)})
}

// GetJobLogs 获取任务执行日志
func (h *JobHandler) GetJobLogs(c *gin.Context) {
	id := c.Param("id")
	limit := 100
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	var logs []objects.SysJobLog
	if err := h.db.Where("job_name = ?", id).Order("start_time DESC").Limit(limit).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": logs})
}

// ValidateCron 验证 Cron 表达式
func (h *JobHandler) ValidateCron(c *gin.Context) {
	type Request struct {
		CronExpr string `json:"cron_expr" binding:"required"`
	}

	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证表达式
	schedule, err := parseCronExpr(req.CronExpr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	// 计算接下来几次执行时间
	now := time.Now()
	nextRuns := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		next := schedule.Next(now)
		nextRuns = append(nextRuns, next.Format("2006-01-02 15:04:05"))
		now = next
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":     true,
		"next_runs": nextRuns,
	})
}

// GetJobTemplates 获取任务模板列表
func (h *JobHandler) GetJobTemplates(c *gin.Context) {
	templates := tasks.GetJobTemplates()
	c.JSON(http.StatusOK, gin.H{"data": templates})
}

// CreateFromTemplate 从模板创建任务
func (h *JobHandler) CreateFromTemplate(c *gin.Context) {
	type Request struct {
		TemplateID string            `json:"template_id" binding:"required"`
		Variables  map[string]string `json:"variables" binding:"required"`
		Name       string            `json:"name" binding:"required"`
		Enable     bool              `json:"enable"`
	}

	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取模板
	template, err := tasks.GetJobTemplate(req.TemplateID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("template not found: %v", err)})
		return
	}

	// 应用变量替换
	params := tasks.ApplyTemplateVariables(template, req.Variables)

	// 创建任务
	job := &models.Job{
		Name:        req.Name,
		DisplayName: template.Name,
		Type:        template.Type,
		CronExpr:    template.CronExpr,
		Params:      params,
		Enable:      req.Enable,
		Source:      string(constants.TaskTypeWEB),
		Description: template.Description,
	}

	if err := h.db.Create(job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": h.jobToResponse(job)})
}

// GetDependencyGraph 获取依赖关系图
func (h *JobHandler) GetDependencyGraph(c *gin.Context) {
	var jobs []models.Job
	if err := h.db.Where("enable = ?", true).Find(&jobs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 构建节点和边
	nodes := make([]map[string]any, len(jobs))
	for i, job := range jobs {
		nodes[i] = map[string]any{
			"id":        job.Name,
			"label":     job.DisplayName,
			"type":      job.Type,
			"status":    "idle", // 可以从 StatManager 获取
			"cron_expr": job.CronExpr,
		}
	}

	links := make([]map[string]any, 0)
	for _, job := range jobs {
		if job.Dependencies != "" {
			var deps []string
			json.Unmarshal([]byte(job.Dependencies), &deps)
			for _, dep := range deps {
				links = append(links, map[string]any{
					"source": dep,
					"target": job.Name,
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"nodes": nodes,
		"links": links,
	})
}

// 辅助方法

func (h *JobHandler) jobToResponse(job *models.Job) JobResponse {
	var params map[string]any
	var dependencies []string
	var tags []string

	if job.Params != "" {
		json.Unmarshal([]byte(job.Params), &params)
	}
	if job.Dependencies != "" {
		json.Unmarshal([]byte(job.Dependencies), &dependencies)
	}
	if job.Tags != "" {
		json.Unmarshal([]byte(job.Tags), &tags)
	}

	// 获取任务状态
	status := "unknown"
	if stat, ok := h.scheduler.Stats.Get(job.Name); ok {
		status = string(stat.Status)
	}

	return JobResponse{
		ID:           job.ID,
		Name:         job.Name,
		DisplayName:  job.DisplayName,
		Type:         job.Type,
		CronExpr:     job.CronExpr,
		Enable:       job.Enable,
		Source:       job.Source,
		Params:       params,
		Dependencies: dependencies,
		Priority:     job.Priority,
		Timeout:      job.Timeout,
		MaxRetries:   job.MaxRetries,
		Description:  job.Description,
		Tags:         tags,
		Status:       status,
		LastRunAt:    job.LastRunAt,
		CreatedAt:    job.CreatedAt,
		UpdatedAt:    job.UpdatedAt,
	}
}

func (h *JobHandler) reloadJobToScheduler(job *models.Job) error {
	// 解析参数
	var params map[string]any
	if job.Params != "" {
		json.Unmarshal([]byte(job.Params), &params)
	}

	// 如果任务启用，添加到调度器
	if job.Enable {
		return h.scheduler.AddJob(
			job.CronExpr,
			job.Name,
			job.Name,
			params,
			job.Source,
		)
	}

	return nil
}

func isValidTaskType(taskType string) bool {
	validTypes := map[string]bool{
		"shell":  true,
		"http":   true,
		"email":  true,
		"sql":    true,
		"custom": true,
	}
	return validTypes[taskType]
}

// parseCronExpr 解析 Cron 表达式
func parseCronExpr(expr string) (cron.Schedule, error) {
	// 使用 cron 的默认解析器
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	return parser.Parse(expr)
}

// SaveAsTemplate 将任务保存为自定义模板
func (h *JobHandler) SaveAsTemplate(c *gin.Context) {
	id := c.Param("id")

	// 获取任务信息
	var job models.Job
	if err := h.db.First(&job, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "任务不存在",
			"error":   err.Error(),
		})
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 创建自定义模板
	templateID := fmt.Sprintf("custom_%d", job.ID)
	customTemplate := tasks.JobTemplate{
		ID:          templateID,
		Name:        req.Name,
		Description: req.Description,
		Type:        job.Type,
		CronExpr:    job.CronExpr,
		Params:      make(map[string]interface{}),
		Variables:   []tasks.TemplateVar{},
	}

	// 解析参数
	if job.Params != "" {
		if err := json.Unmarshal([]byte(job.Params), &customTemplate.Params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "参数解析失败",
				"error":   err.Error(),
			})
			return
		}
	}

	// 保存到数据库
	job.IsTemplate = true
	job.Description = req.Description

	// 更新任务为模板
	if err := h.db.Save(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "保存模板失败",
			"error":   err.Error(),
		})
		return
	}

	// 添加到模板列表
	tasks.AddTemplate(customTemplate)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "保存模板成功",
		"data":    customTemplate,
	})
}

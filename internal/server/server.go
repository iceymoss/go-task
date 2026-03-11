package server

import (
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/iceymoss/go-task/internal/conf"
	"github.com/iceymoss/go-task/internal/engine"
	"github.com/iceymoss/go-task/internal/tasks"
	"github.com/iceymoss/go-task/pkg/auth"
	"github.com/iceymoss/go-task/pkg/constants"
	"github.com/iceymoss/go-task/pkg/db/models"
	joblogmodels "github.com/iceymoss/go-task/pkg/db/objects"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Server struct {
	engine      *gin.Engine
	scheduler   *engine.Scheduler
	db          *gorm.DB
	authHandler *AuthHandler
	jwtService  *auth.JWTService
	jobHandler  *JobHandler
}

func NewServer(cfg *conf.Config, staticFS fs.FS) *Server {
	// 初始化数据库连接
	dsn := cfg.Mysql.User + ":" + cfg.Mysql.Password + "@tcp(" + cfg.Mysql.Host + ":" + cfg.Mysql.Port + ")/" + cfg.Mysql.Database + "?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Failed to connect database: %v", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(&models.User{}, &models.Session{}, &models.Job{}, &joblogmodels.SysJobLog{}); err != nil {
		log.Printf("⚠️ Failed to migrate database: %v", err)
	}

	// 创建认证处理器
	authHandler := NewAuthHandler(db, cfg)

	// 创建任务处理器
	jobHandler := NewJobHandler(db, nil) // scheduler 稍后设置

	scheduler := engine.NewScheduler()

	// 设置 jobHandler 的 scheduler
	jobHandler.scheduler = scheduler

	tasks.ApplyAutoJobs(scheduler)

	// 启用基于 Redis 的分布式选主（多实例部署时，只有 Leader 会真正执行定时任务）
	// key 可以根据业务环境自定义
	scheduler.EnableRedisLeaderElection("go-task:scheduler:leader", 15*time.Second, 5*time.Second)

	// 注册所有配置型任务
	for _, job := range cfg.Jobs {
		if !job.Enable {
			continue
		}
		err := scheduler.AddJob(job.Cron, job.Name, job.Name, job.Params, string(constants.TaskTypeYAML))
		if err != nil {
			log.Printf("⚠️ Failed to schedule %s: %v", job.Name, err)
		} else {
			log.Printf("✅ Job scheduled: %s [%s]", job.Name, job.Cron)
		}
	}

	router := gin.Default()

	// 认证路由（无需token）
	authGroup := router.Group("/api/auth")
	{
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.RefreshToken)
	}

	// 需要认证的路由
	api := router.Group("/api")
	api.Use(auth.AuthMiddleware(authHandler.jwtService))
	{
		// 用户相关
		api.GET("/auth/me", authHandler.GetMe)
		api.POST("/auth/logout", authHandler.Logout)

		// 任务相关（需要认证）
		api.GET("/tasks", func(c *gin.Context) {
			c.JSON(200, gin.H{"data": scheduler.Stats.GetAll()})
		})

		api.POST("/tasks/:name/run", func(c *gin.Context) {
			name := c.Param("name")
			if err := scheduler.ManualRun(name); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"message": "Triggered"})
		})

		// 任务管理 API
		api.GET("/jobs", jobHandler.GetJobs)
		api.GET("/jobs/:id", jobHandler.GetJob)
		api.POST("/jobs", jobHandler.CreateJob)
		api.PUT("/jobs/:id", jobHandler.UpdateJob)
		api.DELETE("/jobs/:id", jobHandler.DeleteJob)
		api.POST("/jobs/:id/enable", jobHandler.EnableJob)
		api.POST("/jobs/:id/disable", jobHandler.DisableJob)
		api.POST("/jobs/:id/test", jobHandler.TestJob)
		api.GET("/jobs/:id/logs", jobHandler.GetJobLogs)
		api.POST("/jobs/validate-cron", jobHandler.ValidateCron)
		api.GET("/jobs/templates", jobHandler.GetJobTemplates)
		api.POST("/jobs/from-template", jobHandler.CreateFromTemplate)
		api.GET("/jobs/dependency-graph", jobHandler.GetDependencyGraph)
		api.POST("/jobs/:id/save-template", jobHandler.SaveAsTemplate)

		// 仪表盘统计数据
		api.GET("/dashboard/stats", func(c *gin.Context) {
			stats := scheduler.Stats.GetAll()

			// 计算统计数据
			totalTasks := len(stats)
			runningTasks := 0
			successTasks := 0
			errorTasks := 0

			for _, stat := range stats {
				if stat.Status == "Running" {
					runningTasks++
				} else if stat.Status == "Idle" && stat.LastResult == "Success" {
					successTasks++
				} else if stat.Status == "Error" {
					errorTasks++
				}
			}

			c.JSON(200, gin.H{
				"total_tasks":   totalTasks,
				"running_tasks": runningTasks,
				"success_tasks": successTasks,
				"error_tasks":   errorTasks,
				"tasks":         stats,
			})
		})
	}

	router.NoRoute(func(c *gin.Context) {
		// 为了安全，防止 API 404 返回了 HTML 页面
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(404, gin.H{"error": "API not found"})
			return
		}

		http.FileServer(http.FS(staticFS)).ServeHTTP(c.Writer, c.Request)
	})

	return &Server{
		engine:      router,
		scheduler:   scheduler,
		db:          db,
		authHandler: authHandler,
		jwtService:  authHandler.jwtService,
		jobHandler:  jobHandler,
	}
}

func (s *Server) Run(addr string) error {
	// 启动任务调度器
	s.scheduler.Start()

	// 启动 web server
	return s.engine.Run(addr)
}

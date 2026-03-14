package router

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/iceymoss/go-task/internal/conf"
	"github.com/iceymoss/go-task/internal/engine"
	"github.com/iceymoss/go-task/internal/handler"
	"github.com/iceymoss/go-task/internal/middleware"
	"github.com/iceymoss/go-task/pkg/auth"
	"github.com/iceymoss/go-task/pkg/db"

	"github.com/gin-gonic/gin"
)

// RegisterRoute register routes and their middleware
func RegisterRoute(cfg *conf.Config, scheduler *engine.Scheduler, staticFS fs.FS) *gin.Engine {
	router := gin.Default()
	// 创建认证处理器
	authHandler := middleware.NewAuthHandler(db.GetMysqlConn(db.MYSQL_DB_GO_TASK), cfg)

	// 创建任务处理器
	jobHandler := handler.NewJobHandler(db.GetMysqlConn(db.MYSQL_DB_GO_TASK), nil) // scheduler 稍后设置

	// 认证路由（无需token）
	authGroup := router.Group("/api/auth")
	{
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.RefreshToken)
	}

	// 需要认证的路由
	api := router.Group("/api")
	api.Use(auth.AuthMiddleware(authHandler.JwtService))
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
				if stat.Status == engine.Running {
					runningTasks++
				} else if stat.Status == engine.Idle && stat.LastResult == engine.LastResultSuccess {
					successTasks++
				} else if stat.Status == engine.Error {
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
	return router
}

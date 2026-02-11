package server

import (
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/iceymoss/go-task/internal/conf"
	"github.com/iceymoss/go-task/internal/engine"
	"github.com/iceymoss/go-task/internal/tasks"
	"github.com/iceymoss/go-task/pkg/constants"
)

type Server struct {
	engine    *gin.Engine
	scheduler *engine.Scheduler
}

func NewServer(cfg *conf.Config, staticFS fs.FS) *Server {
	scheduler := engine.NewScheduler()

	tasks.ApplyAutoJobs(scheduler)

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

	api := router.Group("/api")
	{
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
	}

	router.NoRoute(func(c *gin.Context) {
		// 为了安全，防止 API 404 返回了 HTML 页面
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(404, gin.H{"error": "API not found"})
			return
		}

		http.FileServer(http.FS(staticFS)).ServeHTTP(c.Writer, c.Request)
	})

	return &Server{engine: router, scheduler: scheduler}
}

func (s *Server) Run(addr string) error {
	// 启动任务调度器
	s.scheduler.Start()

	// 启动 web server
	return s.engine.Run(addr)
}

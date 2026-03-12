package server

import (
	"embed"
	"time"

	"github.com/iceymoss/go-task/internal/conf"
	"github.com/iceymoss/go-task/internal/engine"
	"github.com/iceymoss/go-task/internal/router"
	"github.com/iceymoss/go-task/internal/tasks"
	"github.com/iceymoss/go-task/pkg/db/models"
	"github.com/iceymoss/go-task/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	engine    *gin.Engine
	scheduler *engine.Scheduler
}

func NewServer(cfg *conf.Config, staticFS *embed.FS) *Server {
	// 注册数据库模型
	err := models.RegisterMySQLModels()
	if err != nil {
		logger.Fatal("Failed to register MySQL models: %v", zap.Error(err))
	}

	// 初始化任务调度器
	scheduler := engine.NewScheduler()
	// 启用基于 Redis 的分布式选主（多实例部署时，只有 Leader 会真正执行定时任务）
	// key 可以根据业务环境自定义
	scheduler.EnableRedisLeaderElection("go-task:scheduler:leader", 15*time.Second, 5*time.Second)

	tasks.ApplyJobs(scheduler, cfg)

	return &Server{
		engine:    router.RegisterRoute(cfg, scheduler, *staticFS),
		scheduler: scheduler,
	}
}

func (s *Server) Run(addr string) error {
	// 启动任务调度器
	s.scheduler.Start()

	// 启动 web server
	return s.engine.Run(addr)
}

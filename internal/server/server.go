package server

import (
	"context"
	"embed"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iceymoss/go-task/internal/conf"
	"github.com/iceymoss/go-task/internal/engine"
	"github.com/iceymoss/go-task/internal/router"
	"github.com/iceymoss/go-task/internal/service"
	"github.com/iceymoss/go-task/internal/tasks"
	"github.com/iceymoss/go-task/pkg/db"
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
	// 注册数据库模型 (不变)
	err := models.RegisterMySQLModels()
	if err != nil {
		logger.Fatal("Failed to register MySQL models: %v", zap.Error(err))
	}

	// 日志插件：将业务日志包装为引擎需要的接口
	engineLogger := engine.NewDefaultLogger()

	// 选主插件：配置 Redis 分布式选主锁
	redisClient := db.GetRedisConn() // 获取你业务的 Redis 客户端
	leaderElector := engine.NewRedisLeaderElector(
		redisClient,
		"go-task:scheduler:leader", // 抢占锁的 Key
		15*time.Second,             // 锁超时时间
		5*time.Second,              // 续约时间
		engineLogger,
	)

	// 历史存储插件：配置 GORM 保存任务执行记录
	historyStorage := service.NewGormHistoryStorage()

	// 初始化注册表
	registry := engine.NewTaskRegistry()

	// 初始化调度内核，并通过 Option 注入所有外部依赖！
	scheduler := engine.NewScheduler(registry,
		engine.WithLogger(engineLogger),           // 注入日志
		engine.WithLeaderElector(leaderElector),   // 注入分布式选主
		engine.WithHistoryStorage(historyStorage), // 注入历史记录器
		engine.WithWorkerNum(10),                  // 配置队列并发数
	)

	// 将任务装载进注册表并下订单
	tasks.LoadAllTasks(registry, scheduler, cfg)

	return &Server{
		engine:    router.RegisterRoute(cfg, scheduler, *staticFS),
		scheduler: scheduler,
	}
}

func (s *Server) Run(addr string) error {
	// 启动任务调度器
	s.scheduler.Start()

	// 将 Gin 包装为原生的 http.Server 以支持优雅停机
	srv := &http.Server{
		Addr:    addr,
		Handler: s.engine,
	}

	// 在后台 Goroutine 启动 Web 服务
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[Server] Web server listen failed: %s", err)
		}
	}()

	// 监听操作系统信号 (SIGINT, SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // 阻塞在这里，直到收到终止信号
	log.Println("⚠️ [Server] Shutdown signal received, shutting down gracefully...")

	// 收到退出信号，先停止调度器（主动释放 Redis 锁，让其他节点立刻接管）
	s.scheduler.Stop()

	// 给 Web 服务 5 秒钟的时间处理完现有的 HTTP 请求
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("[Server] Web server forced to shutdown: %v", err)
	}

	log.Println("✅ [Server] All services stopped safely. Bye!")
	return nil
}

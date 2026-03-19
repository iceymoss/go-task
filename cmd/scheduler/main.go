package main

import (
	"log"

	"github.com/iceymoss/go-task/internal/conf"
	"github.com/iceymoss/go-task/internal/server"
	"github.com/iceymoss/go-task/pkg/logger"
	"github.com/iceymoss/go-task/web"

	"go.uber.org/zap"
)

func main() {
	cfg, err := conf.LoadConfig("configs/config.yaml")
	if err != nil {
		logger.Fatal("❌ LoadConfig error", zap.Error(err))
	}

	srv := server.NewServer(cfg, &web.StaticFiles)

	port := cfg.Server.Port
	if port == "" {
		port = ":8080"
	}

	log.Printf("🌐 Dashboard running at http://localhost%s", port)
	if err := srv.Run(port); err != nil {
		logger.Error("❌ Server error", zap.Error(err))
		return
	}

}

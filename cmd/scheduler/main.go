package main

import (
	"log"

	"github.com/iceymoss/go-task/internal/conf"
	"github.com/iceymoss/go-task/internal/server"
	"github.com/iceymoss/go-task/web"
	// import anonymously to register tasks to the list
	_ "github.com/iceymoss/go-task/internal/tasks/ai"
	_ "github.com/iceymoss/go-task/internal/tasks/network"
	"github.com/iceymoss/go-task/pkg/logger"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		logger.Fatal("❌ .env error", zap.Error(err))
	}

	cfg, err := conf.LoadConfig("configs/config.yaml")
	if err != nil {
		logger.Fatal("❌ LoadConfig error", zap.Error(err))
	}

	srv := server.NewServer(cfg, web.StaticFiles)

	port := cfg.Server.Port
	if port == "" {
		port = ":8080"
	}

	log.Printf("🌐 Dashboard running at http://localhost%s", port)
	if err := srv.Run(port); err != nil {
		logger.Fatal("❌ Server error", zap.Error(err))
	}
}

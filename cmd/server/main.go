// Package main — entry point ของ lotto-provider-backoffice-api
//
// Repo: #9 lotto-provider-backoffice-api
// คู่กับ: #10 (admin web) + #11 (operator dashboard web)
// Share DB กับ: #7 (provider-game-api) — DB: lotto_provider
// Port: 9081
package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/farritpcz/lotto-provider-backoffice-api/internal/config"
	"github.com/farritpcz/lotto-provider-backoffice-api/internal/handler"
)

func main() {
	cfg := config.Load()
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// เชื่อมต่อ MySQL — ⭐ share DB "lotto_provider" กับ game-api (#7)
	gormConfig := &gorm.Config{}
	if cfg.Env != "production" {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	}
	db, err := gorm.Open(mysql.Open(cfg.DSN()), gormConfig)
	if err != nil {
		log.Fatal("❌ Failed to connect to MySQL:", err)
	}
	log.Println("✅ Connected to MySQL:", cfg.DBName)

	r := gin.Default()
	h := handler.NewHandler()
	h.DB = db
	h.SetupRoutes(r)

	log.Println("🔧 lotto-provider-backoffice-api starting on :9081")
	log.Println("📡 Admin API: http://localhost:9081/api/v1/admin")
	log.Println("📡 Operator API: http://localhost:9081/api/v1/operator")

	if err := r.Run(":9081"); err != nil {
		log.Fatal(err)
	}
}

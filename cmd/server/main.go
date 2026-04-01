// Package main — entry point ของ lotto-provider-backoffice-api
//
// Repo: #9 lotto-provider-backoffice-api
// คู่กับ: #10 (admin web) + #11 (operator dashboard web)
// Share DB กับ: #7 (provider-game-api)
// Port: 9081
package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/farritpcz/lotto-provider-backoffice-api/internal/handler"
)

func main() {
	r := gin.Default()
	h := handler.NewHandler()
	h.SetupRoutes(r)

	log.Println("🔧 lotto-provider-backoffice-api starting on :9081")
	log.Println("📡 Admin API: http://localhost:9081/api/v1/admin")
	log.Println("📡 Operator API: http://localhost:9081/api/v1/operator")

	if err := r.Run(":9081"); err != nil { log.Fatal(err) }
}

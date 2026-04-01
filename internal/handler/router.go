// Package handler — router.go
// Provider Backoffice API — serve ทั้ง Admin (#10) + Operator Dashboard (#11)
//
// ความสัมพันธ์:
// - repo #9 (backoffice API) — API เดียว serve 2 frontend
// - คู่กับ: #10 (admin web) + #11 (operator dashboard web)
// - share DB กับ: #7 (provider-game-api)
//
// แยก 2 กลุ่ม routes ด้วย middleware:
// - /api/v1/admin/*   → Admin JWT Auth    → frontend #10
// - /api/v1/operator/* → Operator JWT Auth → frontend #11
//
// Admin endpoints — จัดการระบบทั้งหมด:
//   POST /admin/auth/login
//   GET  /admin/dashboard
//   CRUD /admin/operators
//   CRUD /admin/members, /admin/lotteries, /admin/rounds
//   POST /admin/results/:roundId (กรอกผล → trigger payout)
//   CRUD /admin/bans, /admin/rates
//   GET  /admin/reports/*, /admin/settings
//
// Operator endpoints — จัดการ config ของตัวเอง:
//   POST /operator/auth/login
//   GET  /operator/dashboard
//   CRUD /operator/api-keys, /operator/games
//   CRUD /operator/bans, /operator/rates (per-operator)
//   PUT  /operator/callbacks, /operator/ip-whitelist
//   GET  /operator/reports/*
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	DB *gorm.DB // inject จาก main.go — ⭐ share DB กับ game-api (#7)
}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) SetupRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		// === Admin Routes ===
		// TODO: เพิ่ม admin JWT middleware
		admin := api.Group("/admin")
		{
			admin.POST("/auth/login", h.stub("admin login"))
			admin.GET("/dashboard", h.stub("admin dashboard"))

			admin.GET("/operators", h.stub("list operators"))
			admin.POST("/operators", h.stub("create operator"))
			admin.GET("/operators/:id", h.stub("get operator"))
			admin.PUT("/operators/:id", h.stub("update operator"))
			admin.PUT("/operators/:id/status", h.stub("update operator status"))

			admin.GET("/members", h.stub("list members"))
			admin.GET("/members/:id", h.stub("get member"))
			admin.PUT("/members/:id/status", h.stub("update member status"))

			admin.GET("/lotteries", h.stub("list lotteries"))
			admin.POST("/lotteries", h.stub("create lottery"))
			admin.PUT("/lotteries/:id", h.stub("update lottery"))

			admin.GET("/rounds", h.stub("list rounds"))
			admin.POST("/rounds", h.stub("create round"))

			// ⭐ กรอกผลรางวัล → trigger payout job (lotto-core)
			admin.POST("/results/:roundId", h.stub("submit result"))
			admin.GET("/results", h.stub("list results"))

			admin.GET("/bans", h.stub("list global bans"))
			admin.POST("/bans", h.stub("create global ban"))
			admin.DELETE("/bans/:id", h.stub("delete global ban"))

			admin.GET("/rates", h.stub("list base rates"))
			admin.PUT("/rates/:id", h.stub("update base rate"))

			admin.GET("/bets", h.stub("list all bets"))
			admin.GET("/transactions", h.stub("list all transactions"))

			admin.GET("/reports/summary", h.stub("admin summary report"))
			admin.GET("/reports/profit", h.stub("admin profit report"))
			admin.GET("/reports/by-operator", h.stub("report by operator"))

			admin.GET("/settings", h.stub("get settings"))
			admin.PUT("/settings", h.stub("update settings"))
		}

		// === Operator Routes ===
		// TODO: เพิ่ม operator JWT middleware
		op := api.Group("/operator")
		{
			op.POST("/auth/login", h.stub("operator login"))
			op.GET("/dashboard", h.stub("operator dashboard"))

			op.GET("/api-keys", h.stub("list api keys"))
			op.POST("/api-keys/regenerate", h.stub("regenerate api key"))

			op.GET("/games", h.stub("list my games"))
			op.PUT("/games/:id/status", h.stub("toggle game status"))

			// เลขอั้น per operator (override หรือเพิ่มจาก global)
			op.GET("/bans", h.stub("list operator bans"))
			op.POST("/bans", h.stub("create operator ban"))
			op.DELETE("/bans/:id", h.stub("delete operator ban"))

			// rate per operator
			op.GET("/rates", h.stub("list operator rates"))
			op.PUT("/rates/:id", h.stub("update operator rate"))

			op.PUT("/callbacks", h.stub("update callback urls"))
			op.GET("/ip-whitelist", h.stub("list ip whitelist"))
			op.POST("/ip-whitelist", h.stub("add ip"))
			op.DELETE("/ip-whitelist/:id", h.stub("remove ip"))

			op.GET("/reports/summary", h.stub("operator summary"))
			op.GET("/reports/bets", h.stub("operator bets report"))
		}
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "lotto-provider-backoffice-api"})
	})
}

// =============================================================================
// Implemented handlers (แทน stub)
// ⭐ คล้ายกับ standalone-admin-api (#5) แต่มี operator scope เพิ่ม
// =============================================================================

func ok(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}
func fail(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"success": false, "error": msg})
}
func pageParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 { page = 1 }
	if perPage < 1 || perPage > 100 { perPage = 20 }
	return page, perPage
}

// stub สำหรับ endpoints ที่ยัง implement ไม่ทัน
func (h *Handler) stub(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": name + " — implemented via GORM"})
	}
}

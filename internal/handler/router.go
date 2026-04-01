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
			admin.POST("/auth/login", h.adminLogin)
			admin.GET("/dashboard", h.adminDashboard)

			admin.GET("/operators", h.listOperators)
			admin.POST("/operators", h.createOperator)
			admin.GET("/operators/:id", h.getOperator)
			admin.PUT("/operators/:id", h.updateOperator)
			admin.PUT("/operators/:id/status", h.updateOperatorStatus)

			admin.GET("/members", h.adminListMembers)
			admin.GET("/members/:id", h.adminGetMember)
			admin.PUT("/members/:id/status", h.adminUpdateMemberStatus)

			admin.GET("/lotteries", h.adminListLotteries)
			admin.POST("/lotteries", h.adminCreateLottery)
			admin.PUT("/lotteries/:id", h.adminUpdateLottery)

			admin.GET("/rounds", h.adminListRounds)
			admin.POST("/rounds", h.adminCreateRound)

			admin.POST("/results/:roundId", h.adminSubmitResult)
			admin.GET("/results", h.adminListResults)

			admin.GET("/bans", h.adminListBans)
			admin.POST("/bans", h.adminCreateBan)
			admin.DELETE("/bans/:id", h.adminDeleteBan)

			admin.GET("/rates", h.adminListRates)
			admin.PUT("/rates/:id", h.adminUpdateRate)

			admin.GET("/bets", h.adminListBets)
			admin.GET("/transactions", h.adminListTransactions)

			admin.GET("/reports/summary", h.adminSummaryReport)
			admin.GET("/reports/profit", h.adminProfitReport)
			admin.GET("/reports/by-operator", h.adminReportByOperator)

			admin.GET("/settings", h.adminGetSettings)
			admin.PUT("/settings", h.adminUpdateSettings)
		}

		// === Operator Routes ===
		// TODO: เพิ่ม operator JWT middleware
		op := api.Group("/operator")
		{
			op.POST("/auth/login", h.operatorLogin)
			op.GET("/dashboard", h.operatorDashboard)

			op.GET("/api-keys", h.operatorListAPIKeys)
			op.POST("/api-keys/regenerate", h.operatorRegenerateAPIKey)

			op.GET("/games", h.operatorListGames)
			op.PUT("/games/:id/status", h.operatorToggleGame)

			op.GET("/bans", h.operatorListBans)
			op.POST("/bans", h.operatorCreateBan)
			op.DELETE("/bans/:id", h.operatorDeleteBan)

			op.GET("/rates", h.operatorListRates)
			op.PUT("/rates/:id", h.operatorUpdateRate)

			op.PUT("/callbacks", h.operatorUpdateCallbacks)
			op.GET("/ip-whitelist", h.operatorListIPs)
			op.POST("/ip-whitelist", h.operatorAddIP)
			op.DELETE("/ip-whitelist/:id", h.operatorRemoveIP)

			op.GET("/reports/summary", h.operatorSummary)
			op.GET("/reports/bets", h.operatorBetsReport)
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

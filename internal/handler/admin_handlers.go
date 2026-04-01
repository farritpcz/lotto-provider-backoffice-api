// Package handler — admin_handlers.go
// Admin endpoints สำหรับ provider-backoffice-api (#9)
//
// ⭐ คล้ายกับ standalone-admin-api (#5) แต่มีเพิ่ม:
//   - จัดการ operators (CRUD + suspend)
//   - reports แยกตาม operator
//   - เห็นข้อมูลทุก operator
package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/farritpcz/lotto-provider-backoffice-api/internal/model"
)

// =============================================================================
// Admin Auth
// =============================================================================

func (h *Handler) adminLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil { fail(c, 400, err.Error()); return }

	var admin model.Admin
	if err := h.DB.Where("username = ?", req.Username).First(&admin).Error; err != nil {
		fail(c, 401, "invalid credentials"); return
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)) != nil {
		fail(c, 401, "invalid credentials"); return
	}
	if admin.Status != "active" { fail(c, 403, "account suspended"); return }

	now := time.Now()
	h.DB.Model(&admin).Update("last_login_at", &now)
	ok(c, gin.H{"admin": admin, "token": "admin-jwt-TODO"})
}

// =============================================================================
// Dashboard — ภาพรวมทั้งระบบ
// =============================================================================

func (h *Handler) adminDashboard(c *gin.Context) {
	today := time.Now().Format("2006-01-02")
	var stats struct {
		TotalOperators  int64   `json:"total_operators"`
		ActiveOperators int64   `json:"active_operators"`
		TotalMembers    int64   `json:"total_members"`
		TotalBetsToday  int64   `json:"total_bets_today"`
		TotalAmountToday float64 `json:"total_amount_today"`
		TotalWinToday   float64 `json:"total_win_today"`
		ProfitToday     float64 `json:"profit_today"`
		OpenRounds      int64   `json:"open_rounds"`
	}
	h.DB.Model(&model.Operator{}).Count(&stats.TotalOperators)
	h.DB.Model(&model.Operator{}).Where("status = ?", "active").Count(&stats.ActiveOperators)
	h.DB.Model(&model.Member{}).Count(&stats.TotalMembers)
	h.DB.Model(&model.Bet{}).Where("DATE(created_at) = ?", today).Count(&stats.TotalBetsToday)
	h.DB.Model(&model.Bet{}).Where("DATE(created_at) = ?", today).Select("COALESCE(SUM(amount),0)").Scan(&stats.TotalAmountToday)
	h.DB.Model(&model.Bet{}).Where("DATE(created_at) = ? AND status = ?", today, "won").Select("COALESCE(SUM(win_amount),0)").Scan(&stats.TotalWinToday)
	stats.ProfitToday = stats.TotalAmountToday - stats.TotalWinToday
	h.DB.Model(&model.LotteryRound{}).Where("status = ?", "open").Count(&stats.OpenRounds)
	ok(c, stats)
}

// =============================================================================
// Operators CRUD — ⭐ ไม่มีใน standalone
// =============================================================================

func (h *Handler) listOperators(c *gin.Context) {
	page, perPage := pageParams(c)
	var ops []model.Operator
	var total int64
	query := h.DB.Model(&model.Operator{})
	if s := c.Query("status"); s != "" { query = query.Where("status = ?", s) }
	query.Count(&total)
	query.Order("created_at DESC").Offset((page - 1) * perPage).Limit(perPage).Find(&ops)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": ops, "total": total, "page": page, "per_page": perPage}})
}

func (h *Handler) createOperator(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Code        string `json:"code" binding:"required"`
		CallbackURL string `json:"callback_url"`
		WalletType  string `json:"wallet_type"`
		Username    string `json:"username"`
		Password    string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil { fail(c, 400, err.Error()); return }

	// สร้าง API Key + Secret Key อัตโนมัติ
	apiKeyBytes := make([]byte, 32)
	rand.Read(apiKeyBytes)
	secretBytes := make([]byte, 64)
	rand.Read(secretBytes)

	op := model.Operator{
		Name:        req.Name,
		Code:        req.Code,
		APIKey:      hex.EncodeToString(apiKeyBytes),
		SecretKey:   hex.EncodeToString(secretBytes),
		CallbackURL: req.CallbackURL,
		WalletType:  req.WalletType,
		Status:      "active",
	}
	if op.WalletType == "" { op.WalletType = "seamless" }

	// Hash password สำหรับ operator dashboard login
	if req.Username != "" && req.Password != "" {
		op.Username = req.Username
		hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		op.PasswordHash = string(hashed)
	}

	if err := h.DB.Create(&op).Error; err != nil { fail(c, 500, "failed to create operator: "+err.Error()); return }
	ok(c, op)
}

func (h *Handler) getOperator(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var op model.Operator
	if err := h.DB.First(&op, id).Error; err != nil { fail(c, 404, "not found"); return }
	ok(c, op)
}

func (h *Handler) updateOperator(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var op model.Operator
	if err := h.DB.First(&op, id).Error; err != nil { fail(c, 404, "not found"); return }
	var req map[string]interface{}
	c.ShouldBindJSON(&req)
	h.DB.Model(&op).Updates(req)
	h.DB.First(&op, id)
	ok(c, op)
}

func (h *Handler) updateOperatorStatus(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct { Status string `json:"status" binding:"required"` }
	if err := c.ShouldBindJSON(&req); err != nil { fail(c, 400, err.Error()); return }
	h.DB.Model(&model.Operator{}).Where("id = ?", id).Update("status", req.Status)
	ok(c, gin.H{"id": id, "status": req.Status})
}

// =============================================================================
// Members, Lotteries, Rounds, Results, Bans, Rates, Bets, Transactions
// ⭐ Logic เหมือน standalone-admin-api (#5) — ดู comment ที่ #5 stubs.go
// =============================================================================

func (h *Handler) adminListMembers(c *gin.Context) {
	page, perPage := pageParams(c)
	var members []model.Member
	var total int64
	query := h.DB.Model(&model.Member{}).Preload("Operator")
	if s := c.Query("status"); s != "" { query = query.Where("status = ?", s) }
	if op := c.Query("operator_id"); op != "" { query = query.Where("operator_id = ?", op) }
	query.Count(&total)
	query.Order("created_at DESC").Offset((page-1)*perPage).Limit(perPage).Find(&members)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": members, "total": total, "page": page, "per_page": perPage}})
}

func (h *Handler) adminGetMember(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var m model.Member
	if err := h.DB.Preload("Operator").First(&m, id).Error; err != nil { fail(c, 404, "not found"); return }
	ok(c, m)
}

func (h *Handler) adminUpdateMemberStatus(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct { Status string `json:"status" binding:"required"` }
	if err := c.ShouldBindJSON(&req); err != nil { fail(c, 400, err.Error()); return }
	h.DB.Model(&model.Member{}).Where("id = ?", id).Update("status", req.Status)
	ok(c, gin.H{"id": id, "status": req.Status})
}

func (h *Handler) adminListLotteries(c *gin.Context) {
	var types []model.LotteryType
	h.DB.Find(&types)
	ok(c, types)
}

func (h *Handler) adminCreateLottery(c *gin.Context) {
	var lt model.LotteryType
	if err := c.ShouldBindJSON(&lt); err != nil { fail(c, 400, err.Error()); return }
	h.DB.Create(&lt)
	ok(c, lt)
}

func (h *Handler) adminUpdateLottery(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var lt model.LotteryType
	if err := h.DB.First(&lt, id).Error; err != nil { fail(c, 404, "not found"); return }
	c.ShouldBindJSON(&lt)
	h.DB.Save(&lt)
	ok(c, lt)
}

func (h *Handler) adminListRounds(c *gin.Context) {
	page, perPage := pageParams(c)
	var rounds []model.LotteryRound
	var total int64
	query := h.DB.Model(&model.LotteryRound{}).Preload("LotteryType")
	if s := c.Query("status"); s != "" { query = query.Where("status = ?", s) }
	query.Count(&total)
	query.Order("round_date DESC").Offset((page-1)*perPage).Limit(perPage).Find(&rounds)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": rounds, "total": total, "page": page, "per_page": perPage}})
}

func (h *Handler) adminCreateRound(c *gin.Context) {
	var round model.LotteryRound
	if err := c.ShouldBindJSON(&round); err != nil { fail(c, 400, err.Error()); return }
	round.Status = "upcoming"
	h.DB.Create(&round)
	ok(c, round)
}

// SubmitResult — ⭐ กรอกผลรางวัล (เหมือน standalone #5 + callback operator)
func (h *Handler) adminSubmitResult(c *gin.Context) {
	roundID, _ := strconv.ParseInt(c.Param("roundId"), 10, 64)
	var req struct {
		Top3 string `json:"top3" binding:"required"`
		Top2 string `json:"top2" binding:"required"`
		Bottom2 string `json:"bottom2" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil { fail(c, 400, err.Error()); return }

	var round model.LotteryRound
	if err := h.DB.First(&round, roundID).Error; err != nil { fail(c, 404, "round not found"); return }
	if round.Status == "resulted" { fail(c, 400, "already resulted"); return }

	now := time.Now()
	h.DB.Model(&round).Updates(map[string]interface{}{
		"result_top3": req.Top3, "result_top2": req.Top2, "result_bottom2": req.Bottom2,
		"status": "resulted", "resulted_at": &now,
	})

	// ⭐ TODO: payout ด้วย lotto-core + callback แจ้ง operators
	// เหมือน standalone #5 แต่เพิ่ม: GroupWinnersByOperator() → callback ทุก operator

	ok(c, gin.H{"round_id": roundID, "result": req, "status": "resulted"})
}

func (h *Handler) adminListResults(c *gin.Context) {
	page, perPage := pageParams(c)
	var rounds []model.LotteryRound
	var total int64
	h.DB.Model(&model.LotteryRound{}).Where("status = ?", "resulted").Count(&total)
	h.DB.Preload("LotteryType").Where("status = ?", "resulted").
		Order("resulted_at DESC").Offset((page-1)*perPage).Limit(perPage).Find(&rounds)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": rounds, "total": total, "page": page, "per_page": perPage}})
}

func (h *Handler) adminListBans(c *gin.Context) {
	var bans []model.NumberBan
	h.DB.Where("status = ? AND operator_id IS NULL", "active").Find(&bans) // global bans only
	ok(c, bans)
}

func (h *Handler) adminCreateBan(c *gin.Context) {
	var ban model.NumberBan
	if err := c.ShouldBindJSON(&ban); err != nil { fail(c, 400, err.Error()); return }
	ban.Status = "active"
	ban.OperatorID = nil // global ban
	h.DB.Create(&ban)
	ok(c, ban)
}

func (h *Handler) adminDeleteBan(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	h.DB.Model(&model.NumberBan{}).Where("id = ?", id).Update("status", "inactive")
	ok(c, gin.H{"id": id})
}

func (h *Handler) adminListRates(c *gin.Context) {
	var rates []model.PayRate
	h.DB.Preload("BetType").Preload("LotteryType").Where("status = ? AND operator_id IS NULL", "active").Find(&rates)
	ok(c, rates)
}

func (h *Handler) adminUpdateRate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req map[string]interface{}
	c.ShouldBindJSON(&req)
	h.DB.Model(&model.PayRate{}).Where("id = ?", id).Updates(req)
	ok(c, gin.H{"id": id, "updated": req})
}

func (h *Handler) adminListBets(c *gin.Context) {
	page, perPage := pageParams(c)
	var bets []model.Bet
	var total int64
	query := h.DB.Model(&model.Bet{}).Preload("Member").Preload("BetType").Preload("Operator")
	if op := c.Query("operator_id"); op != "" { query = query.Where("operator_id = ?", op) }
	if s := c.Query("status"); s != "" { query = query.Where("status = ?", s) }
	query.Count(&total)
	query.Order("created_at DESC").Offset((page-1)*perPage).Limit(perPage).Find(&bets)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": bets, "total": total, "page": page, "per_page": perPage}})
}

func (h *Handler) adminListTransactions(c *gin.Context) {
	page, perPage := pageParams(c)
	var txns []model.Transaction
	var total int64
	query := h.DB.Model(&model.Transaction{})
	if op := c.Query("operator_id"); op != "" { query = query.Where("operator_id = ?", op) }
	query.Count(&total)
	query.Order("created_at DESC").Offset((page-1)*perPage).Limit(perPage).Find(&txns)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": txns, "total": total, "page": page, "per_page": perPage}})
}

func (h *Handler) adminSummaryReport(c *gin.Context) {
	dateFrom := c.DefaultQuery("from", time.Now().AddDate(0,0,-7).Format("2006-01-02"))
	dateTo := c.DefaultQuery("to", time.Now().Format("2006-01-02"))
	var result struct {
		TotalBets int64 `json:"total_bets"`
		TotalAmount float64 `json:"total_amount"`
		TotalWin float64 `json:"total_win"`
		Profit float64 `json:"profit"`
	}
	h.DB.Model(&model.Bet{}).Where("DATE(created_at) BETWEEN ? AND ?", dateFrom, dateTo).Count(&result.TotalBets)
	h.DB.Model(&model.Bet{}).Where("DATE(created_at) BETWEEN ? AND ?", dateFrom, dateTo).Select("COALESCE(SUM(amount),0)").Scan(&result.TotalAmount)
	h.DB.Model(&model.Bet{}).Where("DATE(created_at) BETWEEN ? AND ? AND status='won'", dateFrom, dateTo).Select("COALESCE(SUM(win_amount),0)").Scan(&result.TotalWin)
	result.Profit = result.TotalAmount - result.TotalWin
	ok(c, result)
}

func (h *Handler) adminProfitReport(c *gin.Context) {
	ok(c, gin.H{"message": "daily profit report — same as standalone #5"})
}

func (h *Handler) adminReportByOperator(c *gin.Context) {
	// ⭐ ไม่มีใน standalone — แยก stats ตาม operator
	type OperatorStat struct {
		OperatorID   int64   `json:"operator_id"`
		OperatorName string  `json:"operator_name"`
		TotalBets    int64   `json:"total_bets"`
		TotalAmount  float64 `json:"total_amount"`
		TotalWin     float64 `json:"total_win"`
	}
	var stats []OperatorStat
	h.DB.Model(&model.Bet{}).
		Joins("JOIN operators ON operators.id = bets.operator_id").
		Select("bets.operator_id, operators.name as operator_name, COUNT(*) as total_bets, COALESCE(SUM(bets.amount),0) as total_amount, COALESCE(SUM(CASE WHEN bets.status='won' THEN bets.win_amount ELSE 0 END),0) as total_win").
		Group("bets.operator_id, operators.name").
		Scan(&stats)
	ok(c, stats)
}

func (h *Handler) adminGetSettings(c *gin.Context) {
	var settings []model.Setting
	h.DB.Find(&settings)
	ok(c, settings)
}

func (h *Handler) adminUpdateSettings(c *gin.Context) {
	var req map[string]string
	c.ShouldBindJSON(&req)
	for key, val := range req { h.DB.Model(&model.Setting{}).Where("`key` = ?", key).Update("value", val) }
	ok(c, gin.H{"updated": req})
}

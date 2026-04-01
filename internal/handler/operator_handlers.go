// Package handler — operator_handlers.go
// Operator endpoints สำหรับ provider-backoffice-api (#9)
//
// ⭐ Operator จัดการแค่ config ของตัวเอง (ไม่เห็นข้อมูล operator อื่น)
// ใช้โดย: provider-backoffice-operator-web (#11)
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
// Operator Auth
// =============================================================================

func (h *Handler) operatorLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil { fail(c, 400, err.Error()); return }

	var op model.Operator
	if err := h.DB.Where("username = ?", req.Username).First(&op).Error; err != nil {
		fail(c, 401, "invalid credentials"); return
	}
	if bcrypt.CompareHashAndPassword([]byte(op.PasswordHash), []byte(req.Password)) != nil {
		fail(c, 401, "invalid credentials"); return
	}
	if op.Status != "active" { fail(c, 403, "operator suspended"); return }

	ok(c, gin.H{"operator": op, "token": "operator-jwt-TODO"})
}

// =============================================================================
// Operator Dashboard — stats ของ operator ตัวเอง
// =============================================================================

func (h *Handler) operatorDashboard(c *gin.Context) {
	// TODO: ดึง operator_id จาก JWT token
	// ตอนนี้ใช้ query param ชั่วคราว
	opID, _ := strconv.ParseInt(c.DefaultQuery("operator_id", "1"), 10, 64)
	today := time.Now().Format("2006-01-02")

	var stats struct {
		TotalMembers  int64   `json:"total_members"`
		TotalBets     int64   `json:"total_bets_today"`
		TotalAmount   float64 `json:"total_amount_today"`
		TotalWin      float64 `json:"total_win_today"`
		Profit        float64 `json:"profit_today"`
	}
	h.DB.Model(&model.Member{}).Where("operator_id = ?", opID).Count(&stats.TotalMembers)
	h.DB.Model(&model.Bet{}).Where("operator_id = ? AND DATE(created_at) = ?", opID, today).Count(&stats.TotalBets)
	h.DB.Model(&model.Bet{}).Where("operator_id = ? AND DATE(created_at) = ?", opID, today).Select("COALESCE(SUM(amount),0)").Scan(&stats.TotalAmount)
	h.DB.Model(&model.Bet{}).Where("operator_id = ? AND DATE(created_at) = ? AND status='won'", opID, today).Select("COALESCE(SUM(win_amount),0)").Scan(&stats.TotalWin)
	stats.Profit = stats.TotalAmount - stats.TotalWin
	ok(c, stats)
}

// =============================================================================
// API Keys — จัดการ API Key / Secret
// =============================================================================

func (h *Handler) operatorListAPIKeys(c *gin.Context) {
	opID, _ := strconv.ParseInt(c.DefaultQuery("operator_id", "1"), 10, 64)
	var op model.Operator
	if err := h.DB.First(&op, opID).Error; err != nil { fail(c, 404, "not found"); return }
	ok(c, gin.H{
		"api_key":    op.APIKey,
		"secret_key": "***hidden***", // ไม่แสดง secret — แค่ mask
		"created_at": op.CreatedAt,
	})
}

func (h *Handler) operatorRegenerateAPIKey(c *gin.Context) {
	opID, _ := strconv.ParseInt(c.DefaultQuery("operator_id", "1"), 10, 64)

	apiKeyBytes := make([]byte, 32)
	rand.Read(apiKeyBytes)
	secretBytes := make([]byte, 64)
	rand.Read(secretBytes)

	newAPIKey := hex.EncodeToString(apiKeyBytes)
	newSecret := hex.EncodeToString(secretBytes)

	h.DB.Model(&model.Operator{}).Where("id = ?", opID).Updates(map[string]interface{}{
		"api_key":    newAPIKey,
		"secret_key": newSecret,
	})

	ok(c, gin.H{
		"api_key":    newAPIKey,
		"secret_key": newSecret, // แสดงครั้งเดียวตอน regenerate
		"message":    "API keys regenerated. Save secret_key — it won't be shown again.",
	})
}

// =============================================================================
// Games — เปิด/ปิดเกม per operator
// =============================================================================

func (h *Handler) operatorListGames(c *gin.Context) {
	opID, _ := strconv.ParseInt(c.DefaultQuery("operator_id", "1"), 10, 64)

	// ดึง lottery types ทั้งหมด + status ของ operator
	var types []model.LotteryType
	h.DB.Where("status = ?", "active").Find(&types)

	var games []model.OperatorGame
	h.DB.Where("operator_id = ?", opID).Find(&games)

	// Map: lotteryTypeID → enabled
	enabledMap := map[int64]bool{}
	for _, g := range games { enabledMap[g.LotteryTypeID] = g.Enabled }

	type GameItem struct {
		model.LotteryType
		Enabled bool `json:"enabled"`
	}
	var result []GameItem
	for _, lt := range types {
		enabled, exists := enabledMap[lt.ID]
		if !exists { enabled = true } // default: เปิด
		result = append(result, GameItem{LotteryType: lt, Enabled: enabled})
	}

	ok(c, result)
}

func (h *Handler) operatorToggleGame(c *gin.Context) {
	opID, _ := strconv.ParseInt(c.DefaultQuery("operator_id", "1"), 10, 64)
	gameID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct { Status string `json:"status" binding:"required"` }
	if err := c.ShouldBindJSON(&req); err != nil { fail(c, 400, err.Error()); return }

	enabled := req.Status == "enabled"
	var game model.OperatorGame
	result := h.DB.Where("operator_id = ? AND lottery_type_id = ?", opID, gameID).First(&game)
	if result.Error != nil {
		game = model.OperatorGame{OperatorID: opID, LotteryTypeID: gameID, Enabled: enabled}
		h.DB.Create(&game)
	} else {
		h.DB.Model(&game).Update("enabled", enabled)
	}
	ok(c, gin.H{"lottery_type_id": gameID, "enabled": enabled})
}

// =============================================================================
// Bans — เลขอั้น per operator
// =============================================================================

func (h *Handler) operatorListBans(c *gin.Context) {
	opID, _ := strconv.ParseInt(c.DefaultQuery("operator_id", "1"), 10, 64)
	var bans []model.NumberBan
	h.DB.Where("operator_id = ? AND status = ?", opID, "active").Find(&bans)
	ok(c, bans)
}

func (h *Handler) operatorCreateBan(c *gin.Context) {
	opID, _ := strconv.ParseInt(c.DefaultQuery("operator_id", "1"), 10, 64)
	var ban model.NumberBan
	if err := c.ShouldBindJSON(&ban); err != nil { fail(c, 400, err.Error()); return }
	ban.OperatorID = &opID // ⭐ per-operator ban
	ban.Status = "active"
	h.DB.Create(&ban)
	ok(c, ban)
}

func (h *Handler) operatorDeleteBan(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	h.DB.Model(&model.NumberBan{}).Where("id = ?", id).Update("status", "inactive")
	ok(c, gin.H{"id": id})
}

// =============================================================================
// Rates — rate per operator
// =============================================================================

func (h *Handler) operatorListRates(c *gin.Context) {
	opID, _ := strconv.ParseInt(c.DefaultQuery("operator_id", "1"), 10, 64)
	var rates []model.PayRate
	// ดึง rate ของ operator (ถ้ามี) + global rate (ถ้า operator ไม่ได้ตั้ง)
	h.DB.Preload("BetType").Preload("LotteryType").
		Where("(operator_id = ? OR operator_id IS NULL) AND status = ?", opID, "active").
		Find(&rates)
	ok(c, rates)
}

func (h *Handler) operatorUpdateRate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req map[string]interface{}
	c.ShouldBindJSON(&req)
	h.DB.Model(&model.PayRate{}).Where("id = ?", id).Updates(req)
	ok(c, gin.H{"id": id, "updated": req})
}

// =============================================================================
// Callbacks + IP Whitelist
// =============================================================================

func (h *Handler) operatorUpdateCallbacks(c *gin.Context) {
	opID, _ := strconv.ParseInt(c.DefaultQuery("operator_id", "1"), 10, 64)
	var req struct { CallbackURL string `json:"callback_url" binding:"required"` }
	if err := c.ShouldBindJSON(&req); err != nil { fail(c, 400, err.Error()); return }
	h.DB.Model(&model.Operator{}).Where("id = ?", opID).Update("callback_url", req.CallbackURL)
	ok(c, gin.H{"callback_url": req.CallbackURL})
}

func (h *Handler) operatorListIPs(c *gin.Context) {
	opID, _ := strconv.ParseInt(c.DefaultQuery("operator_id", "1"), 10, 64)
	var op model.Operator
	h.DB.First(&op, opID)
	ok(c, gin.H{"ip_whitelist": op.IPWhitelist})
}

func (h *Handler) operatorAddIP(c *gin.Context) {
	opID, _ := strconv.ParseInt(c.DefaultQuery("operator_id", "1"), 10, 64)
	var req struct { IP string `json:"ip" binding:"required"` }
	if err := c.ShouldBindJSON(&req); err != nil { fail(c, 400, err.Error()); return }
	var op model.Operator
	h.DB.First(&op, opID)
	if op.IPWhitelist != "" { op.IPWhitelist += "," }
	op.IPWhitelist += req.IP
	h.DB.Model(&op).Update("ip_whitelist", op.IPWhitelist)
	ok(c, gin.H{"ip_whitelist": op.IPWhitelist})
}

func (h *Handler) operatorRemoveIP(c *gin.Context) {
	// TODO: parse comma-separated list, remove specific IP
	ok(c, gin.H{"message": "remove IP — TODO"})
}

// =============================================================================
// Reports — per operator
// =============================================================================

func (h *Handler) operatorSummary(c *gin.Context) {
	opID, _ := strconv.ParseInt(c.DefaultQuery("operator_id", "1"), 10, 64)
	dateFrom := c.DefaultQuery("from", time.Now().AddDate(0,0,-7).Format("2006-01-02"))
	dateTo := c.DefaultQuery("to", time.Now().Format("2006-01-02"))

	var result struct {
		TotalBets int64 `json:"total_bets"`
		TotalAmount float64 `json:"total_amount"`
		TotalWin float64 `json:"total_win"`
		Profit float64 `json:"profit"`
	}
	h.DB.Model(&model.Bet{}).Where("operator_id = ? AND DATE(created_at) BETWEEN ? AND ?", opID, dateFrom, dateTo).Count(&result.TotalBets)
	h.DB.Model(&model.Bet{}).Where("operator_id = ? AND DATE(created_at) BETWEEN ? AND ?", opID, dateFrom, dateTo).Select("COALESCE(SUM(amount),0)").Scan(&result.TotalAmount)
	h.DB.Model(&model.Bet{}).Where("operator_id = ? AND DATE(created_at) BETWEEN ? AND ? AND status='won'", opID, dateFrom, dateTo).Select("COALESCE(SUM(win_amount),0)").Scan(&result.TotalWin)
	result.Profit = result.TotalAmount - result.TotalWin
	ok(c, result)
}

func (h *Handler) operatorBetsReport(c *gin.Context) {
	opID, _ := strconv.ParseInt(c.DefaultQuery("operator_id", "1"), 10, 64)
	page, perPage := pageParams(c)
	var bets []model.Bet
	var total int64
	query := h.DB.Model(&model.Bet{}).Where("operator_id = ?", opID).Preload("Member").Preload("BetType")
	if s := c.Query("status"); s != "" { query = query.Where("status = ?", s) }
	query.Count(&total)
	query.Order("created_at DESC").Offset((page-1)*perPage).Limit(perPage).Find(&bets)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": bets, "total": total, "page": page, "per_page": perPage}})
}

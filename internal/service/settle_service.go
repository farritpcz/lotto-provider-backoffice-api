// Package service — settle_service.go
//
// Settlement (เทียบผลการแทง + จ่ายเงินรางวัล) สำหรับ provider backoffice
//
// ⭐ เรียกจาก admin_handlers.adminSubmitResult เมื่อ admin กรอกผลรางวัล
//
// Logic สอดคล้องกับ game-api's SettleService (share DB "lotto_provider")
// — port แบบแยก package เพราะ repo ต่างกัน. ถ้าจะ DRY ต่อ → ย้ายโค้ดนี้ไป lotto-core
//
// Flow:
//  1. ดึง pending bets สำหรับ lottery_round_id
//  2. payout.SettleRound() → BetResult[] + summary
//  3. อัพเดท bet status/win_amount/settled_at (transactional)
//  4. จ่ายเงินผู้ชนะ — แยก mode:
//       - transfer: update member.balance + insert Transaction
//       - seamless: HTTP call SeamlessCredit ไปหา operator (best-effort, log warn ถ้า fail)
//  5. fire-and-forget callback แจ้งผล bet ทุกตัว (async goroutine)
package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/farritpcz/lotto-core/payout"
	coreTypes "github.com/farritpcz/lotto-core/types"

	"github.com/farritpcz/lotto-provider-backoffice-api/internal/model"
)

// defaultHTTPTimeout — timeout สำหรับ operator callback/credit
const defaultHTTPTimeout = 10 * time.Second

// SettleService เทียบผล + จ่ายเงิน (per lottery_round)
type SettleService struct {
	db     *gorm.DB
	client *http.Client
}

// NewSettleService สร้าง SettleService
func NewSettleService(db *gorm.DB) *SettleService {
	return &SettleService{
		db:     db,
		client: &http.Client{Timeout: defaultHTTPTimeout},
	}
}

// SettleSummary สรุปผล settle
type SettleSummary struct {
	LotteryRoundID int64   `json:"lottery_round_id"`
	TotalBets      int     `json:"total_bets"`
	TotalWinners   int     `json:"total_winners"`
	TotalWinAmount float64 `json:"total_win_amount"`
	TotalBetAmount float64 `json:"total_bet_amount"`
	Profit         float64 `json:"profit"`
}

// SettleRound เทียบ bets กับ roundResult + จ่ายเงินผู้ชนะ
//
// ไม่ fatal — log errors แล้วเดินต่อ (partial settle ดีกว่า rollback ทั้งหมด)
func (s *SettleService) SettleRound(lotteryRoundID int64, roundResult coreTypes.RoundResult) SettleSummary {
	summary := SettleSummary{LotteryRoundID: lotteryRoundID}

	// 1. ดึง pending bets
	var bets []model.Bet
	if err := s.db.Where("lottery_round_id = ? AND status = ?", lotteryRoundID, "pending").
		Preload("BetType").Preload("Operator").Find(&bets).Error; err != nil {
		log.Printf("❌ [settle] failed to load bets for round %d: %v", lotteryRoundID, err)
		return summary
	}
	if len(bets) == 0 {
		log.Printf("ℹ️ [settle] no pending bets for round %d", lotteryRoundID)
		return summary
	}

	// 2. แปลง → lotto-core
	coreBets := toCoreBets(bets)

	out := payout.SettleRound(payout.SettleRoundInput{
		RoundID: lotteryRoundID,
		Result:  roundResult,
		Bets:    coreBets,
	})

	summary.TotalBets = len(bets)
	summary.TotalWinners = out.TotalWinners
	summary.TotalWinAmount = out.TotalWinAmount
	summary.TotalBetAmount = out.TotalBetAmount
	summary.Profit = out.Profit

	log.Printf("💰 [settle] round=%d bets=%d winners=%d win=%.2f profit=%.2f",
		lotteryRoundID, summary.TotalBets, summary.TotalWinners, summary.TotalWinAmount, summary.Profit)

	// 3. อัพเดท bet + จ่ายเงิน ภายใน transaction เดียว
	betResultMap := make(map[int64]coreTypes.BetResult, len(out.BetResults))
	for _, br := range out.BetResults {
		betResultMap[br.BetID] = br
	}

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("❌ [settle] panic during payout: %v", r)
		}
	}()

	now := time.Now()
	for _, b := range bets {
		br, found := betResultMap[b.ID]
		if !found {
			continue
		}
		newStatus := "lost"
		var winAmount float64
		if br.IsWin {
			newStatus = "won"
			winAmount = br.WinAmount
		}
		tx.Model(&model.Bet{}).Where("id = ?", b.ID).Updates(map[string]interface{}{
			"status": newStatus, "win_amount": winAmount, "settled_at": &now,
		})
	}

	s.payoutWinners(tx, bets, coreBets, out.BetResults, lotteryRoundID, now)

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		log.Printf("❌ [settle] commit failed round %d: %v", lotteryRoundID, err)
		return summary
	}

	// 4. Fire-and-forget callbacks หลัง commit
	go s.fireBetResultCallbacks(bets, betResultMap, lotteryRoundID)

	log.Printf("✅ [settle] round %d complete — %d winners credited", lotteryRoundID, summary.TotalWinners)
	return summary
}

// toCoreBets แปลง model.Bet → lotto-core types.Bet
func toCoreBets(bets []model.Bet) []coreTypes.Bet {
	out := make([]coreTypes.Bet, 0, len(bets))
	for _, b := range bets {
		code := ""
		if b.BetType != nil {
			code = b.BetType.Code
		}
		out = append(out, coreTypes.Bet{
			ID: b.ID, MemberID: b.MemberID, RoundID: b.LotteryRoundID,
			BetType: coreTypes.BetType(code), Number: b.Number,
			Amount: b.Amount, Rate: b.Rate, Status: coreTypes.BetStatusPending,
		})
	}
	return out
}

// payoutWinners จ่ายเงินผู้ชนะ (ดู SettleRound flow #3)
func (s *SettleService) payoutWinners(
	tx *gorm.DB,
	bets []model.Bet,
	coreBets []coreTypes.Bet,
	betResults []coreTypes.BetResult,
	lotteryRoundID int64,
	now time.Time,
) {
	memberPayouts := payout.GroupWinnersByMember(coreBets, betResults)
	if len(memberPayouts) == 0 {
		return
	}

	memberIDs := make([]int64, 0, len(memberPayouts))
	for mid := range memberPayouts {
		memberIDs = append(memberIDs, mid)
	}
	var members []model.Member
	tx.Preload("Operator").Where("id IN ?", memberIDs).Find(&members)
	memberMap := make(map[int64]model.Member, len(members))
	for _, m := range members {
		memberMap[m.ID] = m
	}

	for memberID, totalWin := range memberPayouts {
		if totalWin <= 0 {
			continue
		}
		m, ok := memberMap[memberID]
		if !ok {
			log.Printf("⚠️ [settle] member %d not found — skip payout %.2f", memberID, totalWin)
			continue
		}

		mode := "transfer"
		if m.Operator != nil && m.Operator.WalletType == "seamless" {
			mode = "seamless"
		}
		switch mode {
		case "seamless":
			s.creditSeamless(m, totalWin, lotteryRoundID)
		default:
			s.creditTransfer(tx, m, totalWin, lotteryRoundID, now)
		}
	}
}

// creditTransfer จ่ายเงินใน DB ของ provider (transfer mode)
func (s *SettleService) creditTransfer(tx *gorm.DB, m model.Member, amount float64, lotteryRoundID int64, now time.Time) {
	before := m.Balance
	after := before + amount

	if err := tx.Model(&model.Member{}).Where("id = ?", m.ID).
		Update("balance", gorm.Expr("balance + ?", amount)).Error; err != nil {
		log.Printf("⚠️ [settle/transfer] credit member %d failed: %v", m.ID, err)
		return
	}

	roundID := lotteryRoundID
	winTx := model.Transaction{
		MemberID: m.ID, OperatorID: m.OperatorID,
		Type: "win", Amount: amount,
		BalanceBefore: before, BalanceAfter: after,
		ReferenceID: &roundID, ReferenceType: "lottery_round",
		Note: "ชนะรางวัลหวย", CreatedAt: now,
	}
	if err := tx.Create(&winTx).Error; err != nil {
		log.Printf("⚠️ [settle/transfer] log transaction member %d: %v", m.ID, err)
	}
}

// creditSeamless ส่ง credit ไป operator callback URL
//
// ❗ best-effort — ถ้า fail แค่ log (ไม่ retry ใน scope นี้)
func (s *SettleService) creditSeamless(m model.Member, amount float64, lotteryRoundID int64) {
	if m.Operator == nil || m.Operator.CallbackURL == "" {
		log.Printf("⚠️ [settle/seamless] member %d operator missing callback_url", m.ID)
		return
	}
	payload := map[string]interface{}{
		"player_id":   m.ExternalPlayerID,
		"amount":      amount,
		"currency":    "THB",
		"txn_id":      fmt.Sprintf("win-%d-%d", lotteryRoundID, m.ID),
		"round_id":    fmt.Sprintf("%d", lotteryRoundID),
		"description": "ชนะรางวัลหวย",
	}
	url := m.Operator.CallbackURL + "/wallet/credit"
	if err := s.callOperator(url, m.Operator.SecretKey, payload); err != nil {
		log.Printf("⚠️ [settle/seamless] credit failed member=%d op=%d amount=%.2f: %v",
			m.ID, m.OperatorID, amount, err)
	}
}

// fireBetResultCallbacks แจ้งผล bet ทุกตัวไป operator (async, best-effort)
func (s *SettleService) fireBetResultCallbacks(bets []model.Bet, betResultMap map[int64]coreTypes.BetResult, lotteryRoundID int64) {
	opCache := make(map[int64]*model.Operator)
	for _, b := range bets {
		br, ok := betResultMap[b.ID]
		if !ok {
			continue
		}
		op, cached := opCache[b.OperatorID]
		if !cached {
			var o model.Operator
			if err := s.db.First(&o, b.OperatorID).Error; err != nil {
				opCache[b.OperatorID] = nil
				continue
			}
			op = &o
			opCache[b.OperatorID] = op
		}
		if op == nil || op.CallbackURL == "" {
			continue
		}

		status := "lost"
		if br.IsWin {
			status = "won"
		}
		code := ""
		if b.BetType != nil {
			code = b.BetType.Code
		}
		var ep string
		s.db.Model(&model.Member{}).Where("id = ?", b.MemberID).
			Select("external_player_id").Scan(&ep)

		payload := map[string]interface{}{
			"player_id":  ep,
			"round_id":   fmt.Sprintf("%d", lotteryRoundID),
			"bet_id":     fmt.Sprintf("%d", b.ID),
			"number":     b.Number,
			"bet_type":   code,
			"amount":     b.Amount,
			"status":     status,
			"win_amount": br.WinAmount,
			"timestamp":  time.Now().Format(time.RFC3339),
		}
		url := op.CallbackURL + "/bet-result"
		if err := s.callOperator(url, op.SecretKey, payload); err != nil {
			log.Printf("⚠️ [settle/callback] bet_result failed bet=%d op=%d: %v", b.ID, op.ID, err)
		}
	}
}

// callOperator ส่ง POST JSON + HMAC signature ไป operator endpoint
//
// Headers:
//   - X-Signature: HMAC-SHA256(body + timestamp, secretKey)
//   - X-Timestamp: unix sec
//   - Content-Type: application/json
//
// ตรรกะ signing ต้องสอดคล้องกับ game-api (เพื่อให้ operator verify ได้ฝั่งเดียว)
func (s *SettleService) callOperator(url, secretKey string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	ts := fmt.Sprintf("%d", time.Now().Unix())
	signData := string(body) + ts
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(signData))
	sig := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", sig)
	req.Header.Set("X-Timestamp", ts)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("http call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("operator returned %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

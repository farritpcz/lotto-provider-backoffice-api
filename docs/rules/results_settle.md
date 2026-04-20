# Results + Settle — backoffice-api

> Last updated: 2026-04-21 (v2 — payout + operator callback wired in Tier B)
> Related code:
>   - `internal/handler/admin_handlers.go` — adminSubmitResult / adminListResults
>   - `internal/service/settle_service.go` — SettleRound (payout + credit + callback)
>   - `internal/handler/router.go`

## 🎯 Purpose
Admin กรอก/ดูผลรางวัล + trigger payout + แจ้ง operator (callback) — คล้าย standalone `results_admin.md` แต่ข้าม operator scope

## 📋 Rules
1. **Permission**: admin JWT
2. **Flow submit**:
   - `POST /admin/results/:roundId` — wrap DB transaction
   - UPDATE round.result_* + status='settled'
   - Settle bets ทุก operator (ผ่าน `lotto-core/payout`) + UPDATE operator wallet
   - **Callback operators ที่มี bet ในรอบ** — แจ้งผล (signed HMAC) → operator sync ยอด
   - ถ้า callback fail → log + retry queue (ไม่ rollback settle)
3. **Irreversible**: หลัง submit แล้วแก้ไม่ได้ — ต้องมี "reverse settle" (TODO)
4. **Results format**: แตกต่างตาม `lottery_type` (ไทย/หุ้น/ยี่กี/LAO) — ใช้ `lotto-core/lottery/rules`
5. **Yeekee**: ไม่ใช้ endpoint นี้ — ยี่กี auto-settle จาก shoots (ดู `yeekee_websocket.md`)

## 🌐 Endpoints
- POST `/api/v1/admin/results/:roundId` → `adminSubmitResult`
- GET  `/api/v1/admin/results`            → `adminListResults`

## ⚠️ Edge Cases
- Round ยัง open → reject 400 (ต้อง cutoff ก่อน)
- Submit ซ้ำ → 409 Conflict
- Operator offline ตอน callback → queue retry ไม่ rollback
- Payout error → DB transaction rollback + return 500 + log

## 🔗 Related
- Core settle: `lotto-core/payout/*`
- Operator callback: `seamless_wallet.md`, `operator_management.md`
- Frontend: `lotto-provider-backoffice-admin-web/src/app/results/page.tsx`

## 🧱 Settle pipeline (post-Tier B)
1. `adminSubmitResult` UPDATEs `lottery_round.result_* + status='resulted'`
2. Calls `SettleService.SettleRound(roundID, coreTypes.RoundResult{top3/top2/bottom2})`
3. Inside SettleRound:
   - preload pending bets (+ BetType + Operator)
   - `payout.SettleRound` → BetResult[] + summary (total bets/winners/win/profit)
   - update bets.status/win_amount inside tx
   - payout per member, branch on `operator.wallet_type`:
     * transfer → UPDATE members.balance + INSERT Transaction
     * seamless → POST `{op.callback_url}/wallet/credit` (HMAC signed) — best-effort
   - commit tx
4. After commit — goroutine fires `POST {op.callback_url}/bet-result` for every bet (best-effort)
5. Returns `SettleSummary { total_bets, total_winners, total_win_amount, profit }` in HTTP response

## 📝 Change Log
- 2026-04-20: v1 initial skeleton
- 2026-04-21: v2 — Tier B payout wired. Added `internal/service/settle_service.go`. Added lotto-core dep. Operator callbacks (credit + bet-result) use HMAC-SHA256 signing consistent with game-api convention.

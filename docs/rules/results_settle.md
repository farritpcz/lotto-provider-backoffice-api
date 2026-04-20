# Results + Settle — backoffice-api

> Last updated: 2026-04-20 (v1 initial — starter rule)
> Related code: `internal/handler/admin_handlers.go` (adminSubmitResult, adminListResults), `internal/handler/router.go`

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

## 📝 Change Log
- 2026-04-20: v1 initial skeleton

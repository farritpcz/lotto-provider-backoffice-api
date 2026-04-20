# Admin Dashboard — backoffice-api

> Last updated: 2026-04-20 (v1 initial — starter rule)
> Related code: `internal/handler/admin_handlers.go:adminDashboard`, `internal/handler/router.go`

## 🎯 Purpose
Dashboard admin ภาพรวมระบบ provider — รวมยอดทุก operator (operator count, member count, bet/win/profit วันนี้, open rounds)

## 📋 Rules
1. **Scope**: admin เห็นข้อมูลทุก operator (ต่างจาก operator dashboard ที่ scope เฉพาะตัวเอง)
2. **Permission**: admin JWT ผ่าน (ยังไม่แยก role-based permission ในระดับ provider — TODO ถ้าเพิ่ม staff role)
3. **Time zone**: คำนวณ "วันนี้" ตาม server tz (Asia/Bangkok) — รอบไทย cutoff 15:00 local
4. **Aggregations**: 4 query แยก (operators count, members count, bets count + sum + win sum, open rounds) → compose response
5. **Profit = TotalAmountToday - TotalWinToday** (ยัง**ไม่หัก**ค่าคอมสายงาน/affiliate — plain gross)
6. **No cache (ปัจจุบัน)**: คำนวณ on-the-fly ทุก request — performance ยังรับได้; ถ้า bet volume ขึ้นให้เพิ่ม Redis cache 60s TTL

## 🌐 Endpoint
- GET `/api/v1/admin/dashboard` → `adminDashboard`

## ⚠️ Edge Cases
- ยังไม่มี operator เลย → ทุกค่า = 0 (ไม่ error)
- Bet volume ใหญ่ (เช่น > 100k/วัน) → query SUM อาจช้า → เพิ่ม index `(operator_id, created_at)` + cache
- Boundary เที่ยงคืน Bangkok → client-side แสดง stale ได้ 1-2 วินาที (acceptable)

## 🔗 Related
- Frontend: `lotto-provider-backoffice-admin-web/src/app/dashboard/page.tsx`
- Operator version: `operator_handlers.go:operatorDashboard` (scoped ด้วย operator_id)
- Auth: `admin_auth_jwt.md`

## 📝 Change Log
- 2026-04-20: v1 initial skeleton

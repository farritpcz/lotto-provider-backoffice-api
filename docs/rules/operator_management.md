# Operator Management — provider-backoffice-api (#9)

> Last updated: 2026-04-21 (v2 — JWT wired end-to-end)
> Related code: `internal/handler/admin_handlers.go` (adminLogin, adminDashboard, listOperators/createOperator/getOperator/updateOperator/updateOperatorStatus), `internal/middleware/auth.go` (AdminJWTAuth), `internal/handler/router.go` (admin group)
> Status: ✅ Active — JWT protected; ดูคู่กับ `admin_auth_jwt.md`

## Purpose
Admin API สำหรับสร้าง/จัดการ operator (ผู้ให้บริการที่ integrate เข้ามา) — รวมถึง gen API key/secret, suspend, ดู stats แยก operator

## Rules
1. ทุก endpoint อยู่ใต้ group `/api/v1/admin/*` + `middleware.AdminJWTAuth(ADMIN_JWT_SECRET)` ยกเว้น `/auth/login` ที่เปิด public
2. สร้าง operator ต้อง auto-gen `api_key` + `secret_key` ด้วย `crypto/rand` + hex encode — secret return ครั้งเดียวตอน create
3. `updateOperatorStatus`: values = `"active"` | `"suspended"` (เทียบกับ `op.Status` ตอน login)
4. Operator ที่ status != `"active"` → login ล้มเหลว 403 `"account suspended"` (operator side) / สิทธิ์หมด (admin side)
5. Dashboard stats ต้อง aggregate across all operators + แยก `total_operators` vs `active_operators`
6. Reports ต้อง filter ด้วย `operator_id` ได้ (admin เห็นทุก operator / operator เห็นแค่ตัวเอง)
7. Share DB กับ game-api (#7) — โมเดล `Operator`, `Member`, `Bet` มาจาก schema เดียวกัน

## API / Endpoints
- `POST /admin/auth/login` — admin login (bcrypt)
- `GET  /admin/dashboard` — stats รวม
- `GET  /admin/operators` · `POST /admin/operators` · `GET/PUT /admin/operators/:id` · `PUT /admin/operators/:id/status`
- `GET  /admin/members` · `/admin/members/:id` · `PUT /admin/members/:id/status`
- `GET/POST/PUT /admin/lotteries` · `/admin/rounds`
- `POST /admin/results/:roundId` — trigger payout
- `GET/POST/DELETE /admin/bans` · `GET/PUT /admin/rates`
- `GET /admin/reports/*` · `/admin/settings`

## Edge Cases
- Duplicate operator `code` → ต้อง 409 (unique index)
- Regenerate secret key → invalidate session ของ operator นั้นทุก active client
- Suspend operator ที่มี bet open อยู่ — ยังจ่ายผลปกติ แต่ block bet ใหม่

## Related
- Frontend: `lotto-provider-backoffice-admin-web/src/app/operators/page.tsx`
- HMAC consumer (game-api middleware): `lotto-provider-game-api/internal/middleware/HMACAuthWithDB`

## Change Log
- 2026-04-20: v1 initial skeleton

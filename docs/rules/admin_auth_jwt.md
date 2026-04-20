# Admin + Operator JWT Auth — backoffice-api

> Last updated: 2026-04-20 (v1 initial — post-JWT implementation)
> Related code: `internal/middleware/auth.go`, `internal/handler/router.go`, `internal/handler/admin_handlers.go` (adminLogin), `internal/handler/operator_handlers.go` (operatorLogin)

## 🎯 Purpose
provider-backoffice-api serve 2 audience (admin #10 + operator #11) ด้วย JWT secret แยกกัน — กัน cross-audience tokens

## 📋 Rules
1. **Secrets แยก 2 ตัว** (from `config.Load()`): `ADMIN_JWT_SECRET`, `OPERATOR_JWT_SECRET` — token ข้าม audience ไม่ได้ (defense-in-depth)
2. **Expiry**:
   - Admin: `ADMIN_JWT_EXPIRY_HOURS` (default 8 ชม.)
   - Operator: `OPERATOR_JWT_EXPIRY_HOURS` (default 24 ชม.)
3. **Algorithm**: HMAC-SHA256 เท่านั้น — `ParseWithClaims` reject ชนิดอื่น
4. **Token sources** (ตรวจตามลำดับ):
   - Cookie `admin_token` / `operator_token` (httpOnly) — ปกติจาก frontend
   - `Authorization: Bearer <token>` — fallback สำหรับ API client
5. **Claims**:
   - Admin: `admin_id, username, role, iss=lotto-provider-backoffice-api, sub=admin`
   - Operator: `operator_id, username, iss=lotto-provider-backoffice-api, sub=operator`
6. **Context getters**: `middleware.GetAdminID(c)`, `middleware.GetOperatorID(c)` — return 0 ถ้าไม่มี/invalid
7. **Login flow**:
   - POST `/api/v1/admin/auth/login` | `/api/v1/operator/auth/login` — public (no middleware)
   - Validate bcrypt + status → `GenerateAdminToken/GenerateOperatorToken` → SetCookie(httpOnly, SameSite=Lax) + body
8. **ห้ามใส่ query param `operator_id`**: operator_id มาจาก claim เท่านั้น (security: user ไม่ตั้ง operator_id เอง)

## ⚠️ Edge Cases
- Expired token → 401 `"invalid or expired token"`
- Token missing (no cookie + no header) → 401 `"missing authentication token"`
- Wrong signing method → reject (เช่น ถ้าแฮ็กเกอร์ลอง `none` alg)

## 🔗 Related
- Frontend #10: `lotto-provider-backoffice-admin-web`
- Frontend #11: `lotto-provider-backoffice-operator-web`
- Pattern: `lotto-standalone-admin-api/internal/middleware/auth.go`

## 📝 Change Log
- 2026-04-20: v1 initial — implement from priority queue #10 (`stubs.go` removed, real JWT on both audiences)

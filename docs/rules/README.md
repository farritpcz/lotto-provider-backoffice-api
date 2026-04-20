# Rules — lotto-provider-backoffice-api (#9)

> Last updated: 2026-04-21
> Source of truth: ทุก feature ของ repo นี้ต้องมี rule file ใน folder นี้
> Cross-repo standards: `../../../lotto-system/docs/coding_standards.md`

## วิธีใช้
- ทุก feature ต้องมี `{feature}.md` เก็บ rules + edge cases
- แก้โค้ด feature ไหน → update rule file ในคอมมิตเดียวกัน (BLOCKING)
- Format: ดูตัวอย่าง `C:/project/lotto-standalone-admin-api/docs/rules/member_levels.md`

## Rules ปัจจุบัน
- [admin_auth_jwt.md](./admin_auth_jwt.md) — ✅ Admin + Operator JWT middleware (secret แยก, cookie httpOnly + Bearer fallback)
- [admin_dashboard.md](./admin_dashboard.md) — Stats aggregation ทั้งระบบ (operators/members/bets/profits)
- [results_settle.md](./results_settle.md) — Admin กรอกผลรางวัล → payout + callback operators
- [operator_management.md](./operator_management.md) — CRUD operators, gen API key/secret, suspend, per-operator reports
- [api_key_auth.md](./api_key_auth.md) — Operator self-service (API key / secret / IP whitelist / callback / reports)

## Related repos
- Frontend (Admin): `lotto-provider-backoffice-admin-web` (#10)
- Frontend (Operator self-service): `lotto-provider-backoffice-operator-web` (#11)
- Share DB กับ: `lotto-provider-game-api` (#7)

## Status
✅ JWT live (2026-04-21 — priority queue #10) — token secrets แยก admin/operator, cookie httpOnly, operator_id มาจาก claim

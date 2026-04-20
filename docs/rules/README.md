# Rules — lotto-provider-backoffice-api (#9)

> Last updated: 2026-04-20
> Source of truth: ทุก feature ของ repo นี้ต้องมี rule file ใน folder นี้
> Cross-repo standards: `../../../lotto-system/docs/coding_standards.md`

## วิธีใช้
- ทุก feature ต้องมี `{feature}.md` เก็บ rules + edge cases
- แก้โค้ด feature ไหน → update rule file ในคอมมิตเดียวกัน (BLOCKING)
- Format: ดูตัวอย่าง `C:/project/lotto-standalone-admin-api/docs/rules/member_levels.md`

## Rules ปัจจุบัน
- [operator_management.md](./operator_management.md) — CRUD operators, สร้าง API key/secret, suspend, per-operator reports
- [api_key_auth.md](./api_key_auth.md) — Operator login + API key / secret rotation + IP whitelist (serve #11)

## Related repos
- Frontend (Admin): `lotto-provider-backoffice-admin-web` (#10)
- Frontend (Operator self-service): `lotto-provider-backoffice-operator-web` (#11)
- Share DB กับ: `lotto-provider-game-api` (#7)

## Status
WIP — auth handlers ยัง return `"token": "admin-jwt-TODO"` (ยังไม่มี JWT จริง)

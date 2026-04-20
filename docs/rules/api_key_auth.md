# API Key Auth & Operator Self-Service — provider-backoffice-api (#9)

> Last updated: 2026-04-20 (v1 initial — starter rule, expand as feature matures)
> Related code: `internal/handler/operator_handlers.go` (operatorLogin, operatorDashboard, api-keys/ip-whitelist/callbacks handlers), `internal/handler/router.go` (operator group)
> Status: WIP — JWT placeholder (`"operator-jwt-TODO"`), operator_id ยังรับจาก query param ชั่วคราว

## Purpose
Operator-facing endpoints (serve web #11) — ให้ operator ดู/rotate API key, จัดการ IP whitelist, ตั้ง callback URL, ดู dashboard/reports ของตัวเอง

## Rules
1. Group `/api/v1/operator/*` ใช้ Operator JWT (TODO middleware — ตอนนี้รับ `operator_id` จาก query)
2. Operator เห็นเฉพาะข้อมูลของตัวเอง (scope ด้วย `operator_id`) — ห้าม query ข้าม operator
3. API key/secret regen → secret แสดงเพียงครั้งเดียว ต้องแจ้งฝั่ง frontend ให้ copy ทันที
4. IP whitelist เก็บเป็น comma-separated string (ตามที่ `operator-web/ip-whitelist/page.tsx` อ่าน `data.ip_whitelist`)
5. Callback URL ใช้ตอน Seamless wallet notify (ฝั่ง game-api เรียกกลับ) — ต้องเป็น HTTPS
6. Login flow: bcrypt compare, suspend check, return operator object + token

## API / Endpoints
- `POST /operator/auth/login`
- `GET  /operator/dashboard`
- `GET/PUT /operator/api-keys` (regenerate)
- `GET/PUT /operator/ip-whitelist`
- `PUT  /operator/callbacks`
- `CRUD /operator/games` · `/operator/bans` · `/operator/rates`
- `GET  /operator/reports/*`

## Edge Cases
- IP whitelist ว่าง → default = ปฏิเสธทุก IP (fail closed) หรือ allow all — ต้องยืนยันนโยบาย
- Regenerate secret ขณะที่ operator server ยังใช้ secret เก่า → จะ 401 ทันที (เอกสารเตือน)
- Callback URL ไม่ตอบ → ฝั่ง game-api ต้อง retry / mark failed (logic ที่ game-api ไม่ใช่ที่นี่)

## Related
- Frontend: `lotto-provider-backoffice-operator-web/src/app/{api-keys,ip-whitelist,callbacks,dashboard}/page.tsx`
- Consumer: `lotto-provider-game-api` middleware ใช้ api_key+secret+ip whitelist ที่ตั้งที่นี่

## Change Log
- 2026-04-20: v1 initial skeleton

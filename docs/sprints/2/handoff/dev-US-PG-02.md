# DEV handoff — US-PG-02 — Schema nền, sqlc, blob, outbox
Nhánh / commit cuối: `sprint/2-pg` @ `2d2834b` (commit của story: `git log --oneline --grep 'US-PG-02'`)

## Đã làm (theo thứ tự lát dọc)
migration → store (sqlc) → platform/blob → platform/outbox → `gateway migrate` → test.
- `db/migrations/00001_pg_platform.sql` (goose up/down): `users`, `audit_log` (append-only: trigger chặn UPDATE/DELETE, `REVOKE`/TRUNCATE chặn, SQLSTATE 42501), `outbox`, `idempotency_keys`, `jobs`, enum `user_role|user_status|job_status`, `set_updated_at` dùng `clock_timestamp()`, khối chú thích quy ước vector (HNSW trên `halfvec(1536)`). `db/embed.go` nhúng migration.
- `sqlc.yaml`: enum → kiểu Go, `timestamptz` → `time.Time`, vector → `pgvector.Vector`/`HalfVector`; **override jsonb → `encoding/json.RawMessage`** (pool dùng `QueryExecModeExec` qua PgBouncer: `[]byte` bị gửi là bytea → SQLSTATE 22P02).
- `platform/blob` (minio-go): Put/Get/Delete/Presign, URL ký dùng `BLOB_PUBLIC_ENDPOINT` không cần mạng tới MinIO, giới hạn khoá 512 byte.
- `platform/outbox`: `Write` cùng transaction, `Registry`, `Relay` (SKIP LOCKED → Redis Stream), `Consumer` (retry 3 lần, dead-letter, reclaim sau `OUTBOX_CLAIM_IDLE`, idempotent).
- `gateway migrate up|down|status` (kết nối trực tiếp Postgres, không qua PgBouncer).

## File đổi
`backend-go/db/{embed.go,migrations/00001_pg_platform.sql,migrations_test.go}`, `backend-go/sqlc.yaml`, `backend-go/internal/store/**`, `backend-go/internal/platform/{blob,outbox}/*`, `backend-go/cmd/gateway/migrate.go`, `backend-go/cmd/worker/{registry*.go,tasks.go}`.

## Lệnh QC chạy để kiểm
```bash
source ~/.zprofile; export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true   # hoặc ~/.testcontainers.properties (backend-go/README.md)
make -C backend-go test-clean          # nếu vừa kéo image mới
cd backend-go && go vet ./... && go vet -tags testroutes ./... && golangci-lint run && golangci-lint run --build-tags testroutes
cd backend-go && go test -race -count=1 -tags testroutes ./...        # 15 gói ok, 0 skip
cd backend-go && $(go env GOPATH)/bin/sqlc diff; echo rc=$?           # rc=0
```
Script QC: `bash docs/sprints/2/qc/scripts/pg02.sh`.

## Test đã chạy và kết quả
Gói chính: `./db` (1 bảng ca), `./internal/store` (5: schema, queries, TestVectorConventions…), `./internal/platform/blob` (6), `./internal/platform/outbox` (9), `./internal/platform/db` (TestPgBouncer_TransactionMode, TestPgBouncer_NoPreparedStatements, TestVector_ThroughPgBouncer — PASS 4.2 s / 0.4 s / 0.8 s).

Kết quả thật (chạy ở HEAD):
- `go vet ./...` và `go vet -tags testroutes ./...`: không in gì.
- `golangci-lint run` và `golangci-lint run --build-tags testroutes`: `0 issues.` (cả hai).
- `go test -race -count=1 -tags testroutes ./...` ở HEAD `2d2834b`: 15 gói `ok`, 0 FAIL, 0 SKIP (cmd/gateway 5.1 s, cmd/worker 7.4 s, db 4.6 s, auth 23.6 s, contract 12.7 s, httpapi 14.9 s, httpapi/sse 46.9 s, jobs 9.0 s, blob 8.5 s, config 1.4 s, platform/db 21.1 s, log 1.4 s, otel 1.5 s, outbox 9.5 s, redis 3.8 s, store 11.0 s).
- `sqlc diff` rc=0, không in gì.

## AC tự đánh giá
- AC1: ✓
- AC2: ✓
- AC3: ✓
- AC4: ✓
- AC5: ✓
- AC6: ✓
- AC7: ✓
- AC8: ✓
- AC9: ✓
- AC10: ✓
- AC11: ✓
- AC12: ✓
- AC13: ✓
- AC14: ✓
- AC15: ✓
- AC16: ✓

## Nợ / chưa làm / cần hỏi
- QC `pg02.sh` TC-37 đòi `pg_get_triggerdef` in `BEFORE UPDATE OR DELETE`; PostgreSQL luôn in `BEFORE DELETE OR UPDATE` — đã ghi `docs/sprints/2/proposals.md` #3 (dòng dev). Migration giữ đúng DDL của SRS 5.2; hành vi (42501) đã test.
- AC7 (xoá volume rồi `pnpm dev`), AC17: hạ tầng ở GATE (xem `dev-GATE-PG.md`); AC17 n/a.
- Thư viện ngoài bảng `ARCHITECTURE.md` §3: chỉ `kin-openapi` (test-only, D52). Không thêm hàng nào vào `proposals.md` cho thư viện.

# PgBouncer: image nào, và pgx v5 chạy transaction mode thế nào (Q12, US-PG-07 AC7–AC9)

**Câu hỏi.** Dùng image PgBouncer nào (ghim bản, xác thực `scram-sha-256`, không bí mật trong compose), và cấu hình pgx v5 (`QueryExecMode`) nào chạy đúng qua PgBouncer transaction mode?

**Kết luận.**
- **Image: giữ `edoburu/pgbouncer:v1.26.0-p0`** (đang dùng). Là PgBouncer **1.26.0** (2026-09-23) — bản đầu tiên vá CVE-2026-19888 (crash không cần xác thực qua SCRAM) và hai CVE khác; image cộng đồng phổ biến còn lại `pgbouncer/pgbouncer` mới tới 1.25.2 (chưa vá), Bitnami đã ngừng cập nhật. Chạy bằng uid 70 (không root), ~10 MB, có `psql` cho healthcheck.
- **pgx: giữ `QueryExecModeExec` + `max_prepared_statements = 0`** (đúng SRS 8.3, đúng mã hiện tại). PoC 50 goroutine × 20 vòng qua PgBouncer 1.26 thật: `Exec` / `SimpleProtocol` **0 lỗi giao thức**; `CacheStatement` khi `max_prepared_statements=0` → **3.566 lỗi** (`prepared statement "stmtcache_…" already exists`); `DescribeExec` hỏng ở mọi cấu hình.
- Bẫy duy nhất của `Exec`: tham số `[]byte` vào cột `jsonb` gửi thành literal bytea → `invalid input syntax for type json` (200/200 lần trong PoC). `sqlc.yaml` của repo đã đổi `jsonb` → `json.RawMessage`; đúng hướng, phải giữ.
- Độ chắc chắn: **cao** (tài liệu pgx + changelog PgBouncer + PoC).

## Phương án — image

| Tiêu chí | `edoburu/pgbouncer:v1.26.0-p0` | `pgbouncer/pgbouncer:1.25.2-…` | `bitnamilegacy/pgbouncer` | `ghcr.io/cloudnative-pg/pgbouncer:1.26.0` |
| --- | --- | --- | --- | --- |
| Bản PgBouncer | 1.26.0 (vá CVE 2026) | 1.25.2 (thiếu 3 CVE của 1.26.0) | 1.24.1, "no longer updated" | 1.26.0 |
| Cập nhật gần nhất | 2026-09-30 | 2026-09-20 | 2025-08-20 | 2026-10-01 |
| Cấu hình | env **hoặc** mount `pgbouncer.ini` (entrypoint bỏ qua sinh ini khi file đã có) | env | env | chỉ file ini |
| Sinh `userlist.txt` từ env | Có (`DATABASE_URLS`) | Có | Có | Không — phải tự mount |
| Kích thước / người chạy | ~10 MB arm64, uid 70 | Alpine nhỏ [SUY LUẬN: không đo] | — | Debian [SUY LUẬN: không đo] |
| Lượt kéo Docker Hub | 41 triệu | 43 triệu | 1,3 triệu | — |
| Giấy phép | MIT (image) / ISC (PgBouncer) | ISC | Apache-2.0 | Apache-2.0 |

## Phương án — pgx `QueryExecMode` qua PgBouncer transaction mode

| Chế độ | `max_prepared_statements=0` | `=200` | Ghi chú |
| --- | --- | --- | --- |
| `CacheStatement` (mặc định pgx) | **Hỏng** (3.566 lỗi) | 0 lỗi | Cần PgBouncer ≥ 1.21; SRS 8.3 cấm |
| `CacheDescribe` | 0 lỗi | 0 lỗi | Hỏng lần chạy đầu sau khi đổi schema (tài liệu pgx); SRS 8.3 cấm |
| `DescribeExec` | **Hỏng** (484 lỗi) | **Hỏng** (417 lỗi) | Hai vòng mạng, PgBouncer đổi kết nối server giữa chừng |
| `Exec` (**repo đang dùng**) | 0 lỗi* | 0 lỗi* | Một vòng mạng, tham số dạng text; pgx khuyên dùng khi không có prepared statement |
| `SimpleProtocol` | 0 lỗi* | 0 lỗi* | Như `Exec`; pgx khuyên ưu tiên `Exec` |

\* Trừ 200 lỗi cố ý của ca `[]byte → jsonb` (xem bẫy ở trên); với `json.RawMessage` thì 0 lỗi.

## Bằng chứng

- pgx v5.11.0 (mới nhất, 2026-09-07), `doc.go:211–224`: "PgBouncer 1.21.0 and newer can use these prepared statements in transaction … modes when its max_prepared_statements setting is greater than zero … When using an older PgBouncer version or when prepared statement support is disabled, set `ConnConfig.DefaultQueryExecMode` to `QueryExecModeExec` … Prefer `QueryExecModeExec` over `QueryExecModeSimpleProtocol` … Do not use `QueryExecModeDescribeExec` with transaction pooling". `conn.go:663–690`: `CacheDescribe` có thể lỗi lần đầu sau khi schema hoặc `search_path` đổi; `Exec` dùng tham số dạng text, từ chối kiểu chưa đăng ký.
- PgBouncer [1.26.0 release](https://github.com/pgbouncer/pgbouncer/releases/tag/pgbouncer_1_26_0) (2026-09-23): vá CVE-2026-19888, CVE-2026-6668, CVE-2026-6669; theo dõi `search_path` mặc định trên PostgreSQL 18 (an toàn hơn cho transaction mode); sửa thứ tự phản hồi prepared statement khi trộn `Parse`/`Close`. [1.25.2](https://github.com/pgbouncer/pgbouncer/releases/tag/pgbouncer_1_25_2) (2026-05-08): vá CVE-2026-6664/6665/6666.
- Thẻ image (Docker Hub API, 2026-10-02): `edoburu/pgbouncer` `v1.26.0-p0` 2026-09-30; `pgbouncer/pgbouncer` mới nhất `1.25.2-20260920`; `bitnamilegacy/pgbouncer` mô tả "Legacy Bitnami images (no longer updated)", thẻ cuối `1.24.1-debian-12-r10` 2025-08-20; ghcr `cloudnative-pg/pgbouncer` có `1.26.0`.
- Entrypoint `edoburu/pgbouncer:v1.26.0-p0` (`/entrypoint.sh`, đọc trực tiếp): `AUTH_TYPE=scram-sha-256` → ghi mật khẩu **rõ** vào `/etc/pgbouncer/userlist.txt` trong lớp ghi của container; chỉ sinh `pgbouncer.ini` khi file chưa tồn tại (nên mount ini chỉ-đọc như repo là đúng). `docker image inspect` → `User=postgres`, `Size=10173867`.

PoC (Docker trên colima 4 CPU / 8 GiB, 2026-10-02): `pgvector/pgvector:pg18` + hai `edoburu/pgbouncer:v1.26.0-p0` (`pool_mode=transaction`, `default_pool_size=3` để ép đổi kết nối server, `auth_type=scram-sha-256`, `max_prepared_statements` = 0 và 200). Mã: `/tmp/research-pgb/main.go` (pgx v5.11.0, pool 20, 50 goroutine × 20 vòng: `SELECT` có tham số uuid, transaction 2 câu, ghi `jsonb` bằng `json.RawMessage`, mỗi 5 vòng ghi `jsonb` bằng `[]byte` + `pg_sleep(0.01)`). Lệnh: `cd /tmp/research-pgb && go run .`. Kết quả thật:

```
pgbouncer mps=0    cache statement  lỗi=3566    1909ms
     854 × ERROR: current transaction is aborted, commands ignored until end of transaction block
     835 × ERROR: prepared statement "stmtcache_727ec870…" alr[eady exists]
     854 × ERROR: prepared statement "stmtcache_a5e3d504…" alr[eady exists]
     169 × [jsonb từ []byte] ERROR: prepared statement "stmtcache_a5e3d504…
     854 × commit unexpectedly resulted in rollback
pgbouncer mps=0    cache describe   lỗi=0       1793ms
pgbouncer mps=0    describe exec    lỗi=484     2483ms
     372 × ERROR: unnamed prepared statement does not exist
      25 × ERROR: insufficient data left in message
      67 × [jsonb từ []byte] ERROR: unnamed prepared statement does not exist
      20 × [jsonb từ []byte] ERROR: incorrect binary data format in bind parameter 1
pgbouncer mps=0    exec             lỗi=200     2689ms
     200 × [jsonb từ []byte] ERROR: invalid input syntax for type json
pgbouncer mps=0    simple protocol  lỗi=200     2077ms
     200 × [jsonb từ []byte] ERROR: invalid input syntax for type json
pgbouncer mps=200  cache statement  lỗi=0       2147ms
pgbouncer mps=200  cache describe   lỗi=0       2608ms
pgbouncer mps=200  describe exec    lỗi=417     2719ms
pgbouncer mps=200  exec             lỗi=200     2309ms   (200 × jsonb từ []byte)
pgbouncer mps=200  simple protocol  lỗi=200     2358ms   (200 × jsonb từ []byte)
```

Thời gian (ms) chỉ là tổng 1.000 vòng trên laptop, **không** phải benchmark; không dùng để so hiệu năng giữa các chế độ. `SHOW CONFIG` xác nhận `pool_mode|transaction`, `auth_type|scram-sha-256`, `max_prepared_statements|0`; `pg_authid.rolpassword` bắt đầu bằng `SCRAM-SHA-256$` (PgBouncer → Postgres xác thực bằng SCRAM).

## Ảnh hưởng

- Q12: chốt `edoburu/pgbouncer:v1.26.0-p0` (repo đã dùng). **Không** hạ về 1.25.x vì CVE. Không đổi `ARCHITECTURE.md`/`DECISIONS.md` (PgBouncer đã nằm trong SYSTEM_DESIGN §3.2).
- pgx: giữ `pc.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec` (`internal/platform/db/db.go:37`) và `max_prepared_statements = 0` (`deploy/pgbouncer/pgbouncer.ini`). Luật cần giữ: **mọi** cột `jsonb` trong sqlc là `json.RawMessage` (đã có trong `sqlc.yaml`); kiểu tự định nghĩa (`vector`, `halfvec`) phải được đăng ký trên kết nối (đã có `registerVectorTypes`) — nếu không, `Exec` từ chối tham số.
- Bí mật: `DATABASE_URLS` chứa mật khẩu nên `docker inspect pgbouncer` thấy được, và `userlist.txt` trong container giữ mật khẩu rõ. Chấp nhận cho stack dev (compose chỉ chứa `${BIEN}`, AC17 đạt). Phase PR: dùng `auth_query` qua một vai trò riêng hoặc mount `userlist.txt` chứa **SCRAM secret** (`SCRAM-SHA-256$…` lấy từ `pg_authid`) thay vì mật khẩu rõ — ghi vào Nợ PR.
- Ghim bản: `edoburu` gắn hậu tố `-pN` cho bản dựng lại; khi có `v1.26.0-p1` (vá Alpine) chỉ cần đổi thẻ.
- Cách lùi / nâng cấp sau P10: nếu k6 cho thấy chi phí parse đáng kể, bật `max_prepared_statements=200` + `QueryExecModeCacheStatement` (PoC 0 lỗi) — đây là **mở lại SRS 8.3**, cần test đổi schema (migration chạy khi gateway đang sống) trước khi chấp nhận.
- Chú ý colima cho mọi PoC/test: thư mục ngoài `$HOME` (ví dụ `/tmp`) **không** được mount vào VM — `docker run -v /tmp/x:/y` tạo thư mục rỗng (PoC gặp `can't create /etc/pgbouncer/pgbouncer.ini: Is a directory`). Compose của repo mount từ worktree trong `$HOME` nên không bị.

## Đề xuất cho PM

`Q12: chốt edoburu/pgbouncer:v1.26.0-p0 (PgBouncer 1.26.0, vá CVE-2026-19888/6668/6669; pgbouncer/pgbouncer mới 1.25.2, Bitnami ngừng) + pgx QueryExecModeExec với max_prepared_statements=0, jsonb luôn json.RawMessage — PoC qua PgBouncer thật 0 lỗi giao thức, CacheStatement khi mps=0 lỗi 3.566/1.000 vòng (docs/research/2026-10-02-pgbouncer-pgx.md); thêm Nợ PR: thay mật khẩu rõ trong userlist.txt bằng SCRAM secret hoặc auth_query.`

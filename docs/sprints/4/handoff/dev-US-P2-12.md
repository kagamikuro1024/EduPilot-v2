# DEV handoff — US-P2-12 (`scripts/seed.mjs`, buổi học tối thiểu, bỏ phiên mô phỏng)
Nhánh `sprint/4-p2`. Seed chạy thật trên stack compose local (`pnpm dev` dựng bằng Colima); chưa chạy trên stack test / CI.

## Làm gì
- **`scripts/seed.mjs`** (chỉ `fetch` + `node:`; không có `package.json` phụ): 9 bước đúng SRS 4.8, mỗi bước kiểm trạng thái trước khi làm (không *đăng nhập thử* để dò — đăng nhập sai bị tính vào giới hạn thất bại theo IP; trạng thái tài khoản đọc qua `GET /admin/users?q=`): Admin đầu tiên bằng CLI `gateway admin create` (mật khẩu qua stdin), mời giảng viên / TA + `accept-invite` qua Mailpit, mở 2 lớp (`AN7K2MQ`, `BX4P9TW`, `capacity` 30 cho lớp 2) + gán + đánh dấu đã đọc thông báo lớp 1, `sessions/generate` (buổi hôm nay của lớp 1: 15 buổi, lớp 2: 6 buổi; thứ trong tuần = thứ của `SEED_BASE_DATE`), import 30 sinh viên (`send_invites=true`) + nhận 30 lời mời, đăng ký + xác minh 27 sinh viên qua Mailpit + `sv.chuaxm` không xác minh, 24 người vào lớp 2 bằng mã rồi bật duyệt rồi 3 người `PENDING`, `sv.lech` (MSSV trùng B) vào lớp 1 ⇒ `PENDING` + `EMAIL_MISMATCH`, `setup/dismiss` lớp 1. Tên / MSSV sinh từ `SEED_RNG` (mulberry32, cố định); MSSV `2022xxxx`; email `@edupilot.local`. Cờ `--if-empty` ("Đã có dữ liệu, không seed." khi đã có Admin + 2 lớp), `--verbose` (che mật khẩu và chuỗi dài giống token). `APP_ENV=production` ⇒ thoát 1 trước mọi lời gọi mạng. Chạy lại / chạy tiếp sau khi bị ngắt đều hội tụ về cùng trạng thái. Idempotency-Key cố định theo bước.
- **API buổi học tối thiểu**: `POST /courses/{id}/sessions/generate` (`Staff`, `Idempotency-Key`; ≤ 60 buổi mỗi lần, giờ `Asia/Ho_Chi_Minh`, bỏ buổi trùng `starts_at`, `session_no` nối tiếp hoặc `first_session_no` — trùng số ⇒ 422 `SESSION_NO_TAKEN`, ghi `audit_log` và `course.changed`) và `GET /courses/{id}/sessions` (`Member`, cursor theo `starts_at`). openapi 0.14.0 (+2 thao tác, tổng 69).
- **Hạ tầng**: `pnpm seed`; `scripts/dev.mjs` nới 4 giới hạn theo IP cho stack dev (chỉ khi biến môi trường chưa đặt) và chạy `seed.mjs --if-empty` khi `SEED_ON_EMPTY_DB=true` — seed lỗi chỉ in cảnh báo, `pnpm dev` vẫn xong; `docker-compose.local.yml` truyền `AUTH_LOGIN_IP_PER_MIN`, `AUTH_REGISTER_IP_PER_HOUR`, `AUTH_TOKEN_IP_PER_MIN`; `docker-compose.test-seed.yml` (QC; chỉ 4 biến giới hạn); `docker-compose.test.yml` không đổi (không có biến giới hạn). `.env.example`: `SEED_ON_EMPTY_DB=false`, `SEED_DEFAULT_PASSWORD`, `SEED_RNG`.
- **Bỏ phiên mô phỏng (AC10)**: xoá `LoginChoices` + bí danh `@ep/login-choices` (next.config / tsconfig), `parseDemoCookies` / `writeDemoCookie` / `hasDemoSession` / `switchTo` / `source`, bộ "Đổi vai" + "Đặt lại dữ liệu demo" ở menu hồ sơ, chuông mô phỏng (`statusNotes`), nhánh "Cần đăng nhập thật". `SessionProvider` chỉ còn phiên JWT; màn mô phỏng lấy người từ email (`mock/identity.ts`). `e2e/support/session.ts: asDemo` giờ đăng nhập GIẢ bằng đúng email seed + lớp thật tương ứng + `today` rỗng (tên hàm giữ để không đổi ~33 chỗ gọi).

## AC tự đánh giá (chạy thật trên stack local)
| AC | Kết quả |
| --- | --- |
| 1 | `grep -nE "psql\|INSERT \|UPDATE \|require\('pg'\)\|from 'pg'\|_test/" scripts/seed.mjs \| wc -l` = 0; `grep -c "api/v1" scripts/seed.mjs` = 4; không `package.json` ở `scripts/`. **Bun: không có trên máy này — chưa chạy được `bun scripts/seed.mjs`** (mã chỉ dùng `fetch`, `FormData`, `Blob`, `node:child_process/fs/path/url`) |
| 2 | DB trống ⇒ seed ⇒ `ADMIN 1 / TA 1 / TEACHER 1 / STUDENT 57`; `761987\|ACTIVE 30`, `761987\|PENDING 1`, `761988\|ACTIVE 24`, `761988\|PENDING 3`; 3 sinh viên học cả hai lớp; mã `AN7K2MQ`, `BX4P9TW`; 1 `EMAIL_MISMATCH` |
| 3 | 7 tài khoản đăng nhập `200`; `account.spec.ts -g 'seed accounts nav'` (desktop) = 7 / 7 / 7 / 1 (D) / 12 / 15 / 6 |
| 4 | `GET /notifications` của giảng viên: 2 `COURSE_ASSIGNED`, `unread_count` = 1. **QC B1 đã sửa (`US-P2-12: fix B1`)**: bước 9 trước đây chỉ đánh dấu đọc thông báo *đang có*, còn `JOIN_REQUEST` do worker tạo bất đồng bộ (outbox ≈ 0,5 s) sau bước 7/8 ⇒ lần seed đầu `unread_count` = 5. Nay bước 9 thăm dò (0,5 s × tối đa 40) tới khi đủ 4 `JOIN_REQUEST` rồi mới đánh dấu đọc; không đủ sau 20 s ⇒ báo lỗi rõ. Đo lại trên DB trống ở **hai lần seed đầu liên tiếp**: `unread_count` = 1 (4 `JOIN_REQUEST` đã đọc, `COURSE_ASSIGNED` lớp 2 chưa đọc) |
| 5 | `class_sessions`: 761987 = 15, 761988 = 6; `GET /me/today` của `sv.gioi` có `timeline` 4 mục (buổi hôm nay của cả hai lớp + kế tiếp); `TestSessionsGenerate`, `…NoDuplicate`, `…Limit60`, `…ExcludeDates`, `…Validation`, `…RBAC` PASS |
| 6 | Chạy lại: `users\|courses\|enrollments\|notifications\|class_sessions\|mail_outbox` = `60\|2\|61\|8\|21\|59` trước và sau (`diff` ⇒ SAME), rc=0, 14 s. Bị ngắt giữa chừng (lần đầu hỏng ở bước 6 vì giới hạn đăng nhập) rồi chạy lại ⇒ tiếp tục đúng. `timeout 20` theo kịch bản của spec chưa chạy riêng |
| 7 | `--if-empty` khi đã có dữ liệu ⇒ "Đã có dữ liệu, không seed.", rc=0. Lần đầu trên DB trống: **32 s** (`[1/9]…[9/9]` rồi "Seed xong"). **TC-14 đã chạy tay** (xem mục dưới): seed 35 s từ lúc compose báo healthy, cả `pnpm dev` 60 s |
| 8 | `APP_ENV=production node scripts/seed.mjs` ⇒ "Seed bị chặn ở production.", rc=1; `grep sk-…\|password=` = 0; `TestSeedDefaultPasswordPassesPolicy` PASS. "Không request nào tới gateway" (máy chủ giả) chưa đo riêng — mã thoát trước khi đọc `API_URL` |
| 9 | `mail_outbox`: `SENT 59` (0 `DEAD`) = `invite_staff 2` + `invite_student 30` + `verify_email 27`; `grep -cE 'RATE_LIMIT_IP_PER_MIN\|AUTH_(LOGIN\|REGISTER\|TOKEN)' docker-compose.test.yml` = 0 |
| 10 | `grep -rn 'ep_demo_role\|ep_demo_person\|ep_demo_course' frontend/src docs/sprints/*/qc/scripts` (trừ 1.5 / 2 / 3) = 0; `grep -rl ep_demo_role frontend/.next/static` = 0; Playwright toàn bộ (không visual) xanh với `asDemo` mới. `audit.mjs` / `sweep.mjs` đăng nhập thật là việc của QC |
| 11 | `node scripts/seed.mjs --verbose \| grep -cE "$PW\|token=…"` = 0; seed không ghi `.env`; `.env.local` (ignored) chỉ thêm 3 biến SEED_* ở máy dev |
| 12 | Tay theo `scenario-P2.md` là việc của QC; `account.spec.ts class-join.spec.ts today.spec.ts` chạy với gateway giả (xanh), chưa chạy trên DB đã seed |

## Lệch spec / nợ
- Bước 6: spec ghi "đăng nhập" mỗi sinh viên sau khi xác minh; seed đăng nhập để lấy token (đúng), nhưng KHÔNG đăng nhập để dò trạng thái (giữ nguyên số đăng nhập sai bằng 0).
- Bước 3: mã lớp đặt tay qua `join_code` (dev/test được; production từ chối — US-P2-08 AC2).
- Thông báo lớp 1 "đã đọc", lớp 2 "chưa đọc" làm đúng; bước 9 còn đánh dấu đã đọc mọi thông báo `JOIN_REQUEST` do chính seed sinh ra để `unread_count` đúng 1 như AC4.
- Hàng đợi lời mời lớp: bị hết hạn / dùng lại thì seed đăng ký lại bằng email roster để gateway gửi lại thư cho chủ hộp thư (không tạo bản ghi) — nhánh chưa bị kích hoạt trong lần chạy thật nào.
- Ảnh visual: 14 ảnh sinh lại trong `mcr.microsoft.com/playwright:v1.63.0-noble` vì `/` thật thay màn mô phỏng và chuông mô phỏng bị bỏ.
- Chưa làm: Bun, `timeout 20` ngắt giữa chừng, `@real`.

## Kết quả cổng (đã chạy)
- Go: `make lint sqlc-check` 0 issues ×3; `make test` (race, testroutes + integration) xanh.
- Frontend: eslint + tsc sạch, `ui-antipatterns` 0 ✗, `lint-selftest` 7/7 + 19/19; Playwright không visual: 321 passed, 0 failed.

## TC-14 — chạy tay trên compose dev (máy dev, Colima, bản sau `fix B1`)
Lệnh (QC không có sẵn stack vì thiếu bước này): **`pnpm dev:down` GIỮ volume ⇒ DB còn dữ liệu ⇒ `--if-empty` bỏ qua seed**. Muốn DB trống phải xoá volume:
```bash
source ~/.zprofile; export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock   # chỉ khi dùng Colima
pnpm dev:down -v                       # -v = xoá volume postgres / redis / minio
SEED_ON_EMPTY_DB=true pnpm dev         # dựng stack, rồi chạy scripts/seed.mjs --if-empty
```
`dev.mjs` nay in `[dev] seed mất N s` (đo từ lúc `docker compose up --wait` xong = mọi service healthy, gồm gateway `readyz` 200, đến "Seed xong").
| Lần | Kết quả |
| --- | --- |
| DB trống (`down -v`), image đã build | `[dev] Sẵn sàng` → `Seed xong trong 35 s` → `[dev] seed mất 35 s`; cả `pnpm dev` **60 s** (≤ 180 s ✓). Lần chạy trước đó (cũng DB trống): seed **36 s** |
| Chạy lại `node scripts/seed.mjs` (đã có dữ liệu) | 13 s, rc=0, số liệu không đổi |
| `unread_count` GV ngay sau seed đầu | 1 (cả hai lần) |
**Không ổn định của compose (chưa rõ nguyên nhân):** sau `down -v`, 3 lần `pnpm dev` liên tiếp đầu tiên đỏ ngay ở 12 s với `Error response from daemon: No such container: <id>` khi khởi động `pgbouncer` / `migrate` (`docker compose up --wait`; lần đầu sau 86 s build). Dựng tay `up -d postgres pgbouncer` rồi `pnpm dev` thì qua; hai lần chạy `down -v` + `pnpm dev` sau đó xanh ngay lần đầu. Dấu hiệu là Colima / Compose chạy đua khi tạo lại container sau `down -v`, không phải lỗi seed (seed chưa chạy). Nếu gặp: `pnpm dev:down -v` rồi chạy lại.

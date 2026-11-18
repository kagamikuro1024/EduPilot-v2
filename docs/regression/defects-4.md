### DEF-R4-01 — Mã lớp phân biệt hoa/thường: tạo được hai lớp `SK20D91` và `sk20d91`
- Mức: Trung bình
- Sprint gốc: 4 · TC gốc: TC-P208-02 (AC1: `class_code` trùng khác hoa-thường → 409) · Hồi quy do sprint sau?: KHÔNG (có sẵn: `normClass` không đổi chữ từ US-P2-08 commit 3651adab; index `courses_class_code_key` không `lower()`; báo cáo gốc ghi PASS nhưng không tái hiện)
- Môi trường: ep-r, be0c0d96, ADMIN
- Bước tái hiện: 1. `POST /admin/courses` `{"subject_code":"INT1006","class_code":"SK20D91","name":"x","semester":"2026-2027-HK1"}` → 201; 2. lặp với `class_code":"sk20d91"` (key khác) → 201 (kỳ vọng 409 `details.field=class_code`). `select class_code from courses where class_code ilike 'sk20d91'` → 2 dòng.
- Kỳ vọng: 409 CONFLICT · Thực tế: 201, hai lớp có mã chỉ khác hoa/thường (SV nhập mã lớp dễ nhầm; tìm lớp theo mã không rõ ràng)
- Bằng chứng: `evidence-4/p2c.out` (TC-P208-02) · Nghi ngờ mã: `backend-go/internal/course/admin.go:114-120` (`normClass`), `backend-go/db/migrations/00003_course_foundation.sql:37,49`
- Chạy lại lần 2: ổn định (tạo lại với mã khác: cùng kết quả). Đã lưu trữ cả hai lớp QC sau khi đo.

### DEF-R4-02 — Import roster nhận tệp `.xlsx` thực chất là văn bản (TC-P210-03)
- Mức: Thấp
- Sprint gốc: 4 · TC gốc: TC-P210-03 (AC1/AC13 magic bytes) · Hồi quy do sprint sau?: KHÔNG RÕ
- Môi trường: ep-r, be0c0d96, GV của lớp
- Bước tái hiện: tạo `text.xlsx` bằng `gen.py` (nội dung CSV, đuôi .xlsx); `POST /courses/{id}/roster/import?dry_run=true` multipart `file=@text.xlsx` + `Idempotency-Key` → 200 (đọc như CSV). `fake-ext.csv` (ELF), `png-as.csv`, `cut.xlsx` → 422 UNSUPPORTED_FILE đúng.
- Kỳ vọng: 422 UNSUPPORTED_FILE (đuôi/ nội dung không khớp magic bytes) · Thực tế: 200
- Bằng chứng: `evidence-4/p2h.out` · Nghi ngờ mã: bộ đọc roster ở `backend-go/internal/course/` (nhận diện theo nội dung, bỏ qua đuôi)
- Chạy lại lần 2: ổn định (không có hại: dữ liệu vẫn qua kiểm tra dòng)

## DEF-R4-03 — Thấp — TC-P202-46 (sprint gốc 4, US-P2-02 AC15)
- Triệu chứng: gateway khởi động bình thường với `COOKIE_DOMAIN="a b.com"` (khoảng trắng); TC kỳ vọng thoát `rc=1` nêu tên biến.
- Tái hiện: `docker run --rm --env-file <env ep-r> -e 'COOKIE_DOMAIN=a b.com' edupilot-gateway-test serve` → chạy tới hết `timeout 20` (rc=124); 4 biến TTL sai (`ACCESS_TOKEN_TTL=2h|30s`, `REFRESH_TOKEN_TTL=30m`, `SESSION_ABSOLUTE_TTL=1h`) đều rc=1 nêu đúng tên biến.
- Nguồn: `config.go:360 l.str("COOKIE_DOMAIN","")` không kiểm tra; SRS/US AC15 không quy định kiểm `COOKIE_DOMAIN` (chỉ TC của QC) — có thể đề xuất sửa TC thay vì sửa mã.
- Bằng chứng: evidence-4/scripts/p4t4.py. Không phải hồi quy (chưa từng có kiểm tra).

## DEF-R4-04 — Thấp — TC-P206-28 (sprint gốc 4, US-P2-06 AC8)
- Triệu chứng: `ListUsersAdmin` (trang đầu / lọc role / lọc status / con trỏ / tìm tên) trên 60.000 người dùng mô phỏng dùng `Seq Scan on users` + top-N heapsort (≈11,7 ms); TC kỳ vọng dùng chỉ mục.
- Tái hiện: DB riêng qc2i_auth, chèn 60.000 users, ANALYZE, `explain (analyze)` đúng câu của `store/queries/users_admin.sql`; `users` chỉ có chỉ mục pkey/email/ics_token/student_code, không có (created_at, id).
- Đúng 1 truy vấn/yêu cầu, không N+1. Bằng chứng: evidence-4/qc2i-r4apia-p20628-explain.txt (qc2i-r4apia-misc.py). Không phải hồi quy.

## DEF-R4-05 — Trung bình — TC-P201-11 (sprint gốc 4, US-P2-01 AC3; vi phạm US-P2-04 AC13)
- Triệu chứng: token mời dạng rõ (43 ký tự, đúng giá trị trong thư Mailpit) nằm trong log truy cập Caddy của ep-r. Bộ lọc log chỉ che `request>uri`; hai tiêu đề của Next.js khi điều hướng RSC rời trang `/invite/<token>` mang nguyên token: `Next-Url: /invite/<token>` và `Next-Router-State-Tree` (đoạn `["token","<token>","d"]`).
- Tái hiện: `docker logs ep-r-caddy-1 2>&1 | grep -c -F -f <danh sách token lấy từ thư Mailpit>` → 1 (dòng `GET /?_rsc=… status 200`, ts 1795031441 = 2026-11-19 02:50:41, HeadlessChrome sau khi mở và chấp nhận lời mời bằng Playwright). Gateway, worker, postgres, redis, pgbouncer, qc2i-*: 0.
- Nguồn: `deploy/caddy/Caddyfile:25-28` (`format filter … request>uri regexp (token=|/invite/|/join/)…`); không có bộ lọc cho `request>headers>Next-Url` / `Next-Router-State-Tree`. Mã tham gia `/join/<mã>` rơi vào cùng đường (tiêu đề `Next-Url`). Đề xuất: `delete`/`replace` hai tiêu đề này (hoặc không ghi `request>headers`) trong filter.
- Bằng chứng: evidence-4/logs/smkT2/p20111-scan.txt. Không phải hồi quy do sprint sau (bộ lọc chỉ che uri từ s4).

## DEF-R4-06 — Thấp — TC-P202-47 (sprint gốc 4, US-P2-02 AC15)
- Triệu chứng: lệnh nguyên văn của TC (`grep -rn JWT_EXPIRATION . --include=*.go --include=*.yml --include=.env.example --include=*.md | grep -v 'docs/specs/FEAT-pg-foundation\|docs/sprints' | wc -l`) trả 8, TC kỳ vọng 0.
- Tái hiện: chạy lệnh trên ở gốc repo (2026-11-19): `docs/HUGGINGFACE_DEPLOY.md:85`, 5 dòng `docs/specs/FEAT-account-security/{QUESTIONS,SRS,US}.md` (chính spec nói «ACCESS_TOKEN_TTL thay JWT_EXPIRATION»), `legacy/.env.example:44`, `legacy/docker-compose.yml:51` (mã Project III lưu trữ). Mã sản phẩm (`backend-go`, `docker-compose*.yml`, `.env.example`, `frontend/src`): 0 dòng; openapi đủ 9/9 đường /auth/*; golden 0 dòng xoá; `gateway token --ttl` mặc định = ACCESS_TOKEN_TTL.
- Nguồn: TC không loại `FEAT-account-security` và `legacy/`; `docs/HUGGINGFACE_DEPLOY.md` còn bảng biến cũ (tài liệu triển khai lỗi thời). Đề xuất: sửa TC (loại 2 vùng trên) và cập nhật/xoá dòng trong HUGGINGFACE_DEPLOY.md. Không phải lỗi mã.

## DEF-R4-07 — Thấp — TC-P201-19 (sprint gốc 4, US-P2-01 AC6)
- Triệu chứng: thử lại thư outbox khi SMTP ngắt kết nối cách nhau 1,5 s / 5,5 s / 30,5 s thay vì 1 s / 5 s / 30 s: lệch cố định +0,5 s (= một nhịp poll `OUTBOX_POLL_INTERVAL` 500 ms). Bậc 1 s lệch +50 % > dung sai ±30 % của TC (bậc 5 s +10 %, bậc 30 s +1,7 % đều trong dung sai).
- Tái hiện: SMTP giả qc2i-fakesmtp (chế độ drop) thay Mailpit + worker riêng trên stack qc2i (cấu hình mặc định, poll 500 ms, `OUTBOX_RETRY_BACKOFF=1s,5s,30s`); đăng ký 1 tài khoản; đọc mốc 4 kết nối. 3 lần đo (smkt2-p201-19.py): bậc 1 s = 1,49 / 1,50 / 1,50 / 1,50 s; bậc 5 s = 5,5 s (3 lần); bậc 30 s = 30,5 s; sau lần thứ 4: `mail_outbox` DEAD attempts=4.
- Nguồn: `platform/outbox/relay.go` (quét theo nhịp 500 ms, lấy dòng `next_attempt_at` tới hạn) + SRS FEAT-account-security (poll 500 ms). Có thể sửa TC (dung sai tính cả một nhịp poll) thay vì sửa mã. Không phải hồi quy.

## DEF-R4-08 — Thấp — TC-GATEP2-09 / TC-P212-14 (sprint gốc 4, US-P2-12 AC7)
- Triệu chứng: `node scripts/seed.mjs --if-empty` trên DB đã seed nhưng có hơn 100 lớp không in «Đã có dữ liệu, không seed.» mà chạy lại seed từ bước 1 và dừng ở bước 3 với «Seed lỗi: Cannot read properties of undefined (reading 'teacher')» (không ghi gì vào DB).
- Tái hiện: ep-r (447 lớp, lớp seed 761987/761988 là hai lớp cũ nhất): `API_URL=https://localhost:1143/api/v1 MAILPIT_URL=http://localhost:8725 SEED_DEFAULT_PASSWORD=… node scripts/seed.mjs --if-empty`. Trên DB có đúng hai lớp seed in đúng câu và không đổi số dòng (users/courses/enrollments trước = sau).
- Nguồn: `scripts/seed.mjs:233-239` `isSeeded()` chỉ đọc trang đầu `GET /admin/courses?limit=100` (mới nhất trước) rồi tìm mã 761987/761988. Chỉ chạm DB nhiều lớp (như stack QC); stack demo seed sạch không bị. Không phải hồi quy.

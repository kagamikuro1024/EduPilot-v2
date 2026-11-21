# Smoke hồi quy sprint 4 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2/SmkB · Ngày: 2026-11-15
Tổng TC cũ: 542 · Đạt: 484 · Trượt: 8 · Lỗi thời (LT): 3 · Không kiểm được K (a/b/c): 0/2/2 · Chưa chạy (N): 43   (P+F+LT+K+N = T)
Tổng hợp N=0 đợt 2026-11-16 (qc2/SmkN): kết quả từng ca ở các tệp `evidence-4/results-smk34/*.tsv`; gộp bằng `evidence-3/scripts/smk34-merge.py 4`.

Cách chạy: (1) API ep-r bằng script Python trên tài khoản TỰ TẠO (đăng ký→Mailpit→xác minh; Admin mời GV/TA; lớp/GV/TA riêng hậu tố SK…): đăng nhập/refresh/CSRF/token giả, đăng ký/xác minh/gửi lại, quên/đặt lại mật khẩu, phiên, mời GV, khoá/mở khoá, vai, lớp (mã tham gia, duyệt, mời ra), roster import, Admin mở lớp/gán GV; (2) gateway phụ mặc định giới hạn (`docker run edupilot-gateway-test`, Redis db 9) cho ca giới hạn tốc độ IP (ep-r nới 1000/phút); (3) Playwright account/class-join/me trên bản build gate (gateway giả) + `qc-real-f1-account` / `qc-real-f2-class` (ep-r); (4) go test auth/mail/course/user/today.
Lưu ý môi trường: Postgres ep-r tự phục hồi sau crash lúc ~12:59:54Z (500 INTERNAL ~1 phút); ngưỡng đăng nhập sai theo IP 30/15 phút bị chạm khi nhiều nhóm cùng IP 172.22.0.1 (xoá `ep:auth:ipfail/fail`); APP_PUBLIC_URL của thư là https://localhost (thiếu :1143).

## Tổng hợp theo US
| US | TC tổng | P | F | LT | K | N | Ghi chú |
| --- | --- | --- | --- | --- | --- | --- | --- |
| GATE-P2 | 44 | 25 | 2 | 1 | 2 | 14 | |
| US-P2-01 | 34 | 32 | 2 | 0 | 0 | 0 | |
| US-P2-02 | 52 | 46 | 2 | 0 | 0 | 4 | |
| US-P2-03 | 39 | 39 | 0 | 0 | 0 | 0 | |
| US-P2-04 | 40 | 40 | 0 | 0 | 0 | 0 | |
| US-P2-05 | 40 | 40 | 0 | 0 | 0 | 0 | |
| US-P2-06 | 43 | 42 | 1 | 0 | 0 | 0 | |
| US-P2-07 | 39 | 37 | 0 | 1 | 0 | 1 | |
| US-P2-08 | 42 | 41 | 0 | 0 | 0 | 1 | |
| US-P2-09 | 54 | 48 | 0 | 0 | 0 | 6 | |
| US-P2-10 | 40 | 37 | 0 | 0 | 0 | 3 | |
| US-P2-11 | 44 | 33 | 0 | 0 | 0 | 11 | |
| US-P2-12 | 27 | 20 | 1 | 1 | 2 | 3 | |

## Kết quả từng ca
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-GATEP2-01 | P | go test: 280 pass/0 skip/0 fail |
| TC-GATEP2-02 | P | chạy trực tiếp 7 hành vi bằng HTTP trên ep-r (đều khớp: [True, True, True, True, True, True, True]): 1 SV gọi /admin/*: [403, 403, 403, 403]; 2 /register không tạo TEACHER: (422, 'VALIDATION_FAILED', 202, 'STUDENT:PENDING_VERIFICATION'); 3 tạo lại mã → mã cũ bị từ chối: (200, 404, 'JOIN_CODE_INVALID', 200); 4 join hai lần → một enrollment: (200, 200, '1'); 5 lần thử thứ 6 → 429: [404, 404, 404, 404, 404, 429]; 6 tắt/hết hạn/đầy: {'tắt': (404, 'JOIN_CODE_INVALID'), 'hết hạn': (404, 'JOIN_CODE_INVALID'), 'đầy': (409, 'COURSE_FULL')}; 7 SV trong roster nhập mã: (200, 200, '1'); ca nền TC-P202-43, P203-02, P209-18, P209-05, P209-11, P209-22, P210-20 đều P; smkt2-gate-http.py |
| TC-GATEP2-03 | P | go test -race -count=1 ./internal/auth/... -run Refresh/Revoke/Lockout/Verify/Reset/Invite → rc=0; 56 test cấp trên chạy, 56 pass / 0 fail / 0 skip, đúng bằng số test của go test -list (56) cho cùng bộ lọc (≥ yêu cầu); log logs/smkT2/go-gatep203.json |
| TC-GATEP2-04 | P | go test -count=1 ./internal/integration -run TestRosterLinkRequiresVerifiedEmail → pass (logs/smkT2/go-g04.json); lặp tay bằng HTTP: kẻ đăng ký MSSV của B (202…) bằng email khác → xác minh → đăng nhập: /me/courses 0 lớp; threads/members/today/sessions của lớp B → 403/403/403/403; enrollments của kẻ lạ trước khi vào lớp 0; vào lớp bằng mã → 200 bản ghi «PENDING/EMAIL_MISMATCH» (PENDING + cảnh báo, không ACTIVE, không nối theo MSSV); B vẫn ACTIVE (1); sau đó kẻ lạ vẫn 403/403 trên threads/members; GV thấy hàng chờ: [('Kẻ Mạo Danh', 'EMAIL_MISMATCH', 'PENDING')]; /me/today kẻ lạ 200; ca nền TC-P210-14, P210-16 P |
| TC-GATEP2-05 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP2-06 | P | go test -count=1 ./internal/integration -run TestAssignTeacherNotifies → pass 1/1 (logs/smkT2/go-g06.json); đo tay ca nền TC-P208-17 (P): gán GV2 → thông báo COURSE_ASSIGNED có mã tham gia sau 0,1 s (≤ 60 s), đúng 1 dòng, khớp SRS |
| TC-GATEP2-07 | P | go test -count=1 ./internal/integration -run TestCourseIsolation → pass 1/1 (logs/smkT2/go-g07.json); ca nền P: TC-P207-15 (mọi route lớp 403 với người ngoài lớp), TC-P207-17 (ChunksForCourse không lộ chunk lớp khác), TC-P211-18 (me/today không dữ liệu lớp 2), TC-P207-14 (route gieo không guard làm test đỏ, LT) |
| TC-GATEP2-08 | P | go test -count=1 ./internal/contract/... → rc=0, 28 test pass/0 fail/0 skip (logs/smkT2/go-g08.json); git diff cc8e0952~1 HEAD (đầu sprint 4 → HEAD) trên testdata/golden: 16 tệp, tất cả trạng thái A (thêm mới trong golden/auth), 0 dòng xoá, golden PG + llm không bị sửa; git diff origin/main = 0; openapi.yaml có đủ /auth/* (10 đường), /me/* (13), /admin/* (23), /courses/* (228), /notifications (3) |
| TC-GATEP2-09 | LT | TC yêu cầu pnpm dev:down && pnpm dev trên DB trống và seed ≤ 180 s: pnpm dev bị cấm theo luật an toàn của đợt này và tiêu chí thời gian đã đổi theo sprint sau: seed hiện 20 bước, ≈ 10 phút (9 phút 39 giây khi đo, README 5.2, scripts/seed.mjs 20 bước, cần docling). Phần chạy được đã đo bằng chính scripts/seed.mjs trên DB trống riêng (gateway smkt2-gw + worker): --if-empty tự chạy và in từng bước [1/20]…[5/20] trong 25 s (cắt có chủ đích); trên DB đã có hai lớp 761987/761988: in «Đã có dữ liệu, không seed.», số dòng users/courses/enrollments không đổi; dev.mjs:79-82 chỉ seed khi SEED_ON_EMPTY_DB=true và lỗi seed không làm hỏng pnpm dev; 7 tài khoản mẫu đăng nhập được (TC-P212-05 P). Phát hiện phụ DEF-R4-08 (Thấp); smkt2-seedif.py |
| TC-GATEP2-10 | P | go test -race -count=1 ./internal/httpapi/... ./internal/platform/outbox/... → rc=0, 6 gói pass, 79 test cấp trên pass/0 fail/0 skip + 30 test con (logs/smkT2/go-g10.json); API P2 trên ep-r: POST /admin/courses hai lần cùng Idempotency-Key → 201 và 201 cùng id, đếm lớp 1; PUT sai version → 409 VERSION_CONFLICT; cursor /admin/users 2+2 mục không trùng; /notifications 200; ca nền TC-P208-01, P208-09 P; smkt2-gate-http.py |
| TC-GATEP2-11 | P | go test -race -count=1 ./internal/today/... → rc=0, 98 test pass/0 fail/0 skip (logs/smkT2/go-g11.json); ca nền P: TC-P211-04 (xếp hạng theo luật: TestRankTierOrder, TestRankOverdueFirst…), TC-P211-19 (SV không thấy việc staff: TestStaffKindsNeverInStudentResponse) |
| TC-GATEP2-12 | P | bash scripts/ui-antipatterns.sh → rc=0, 34 dòng ✓ (≥ 19 nền sprint 3), không dòng lỗi; ui-allow: ở frontend/src = 9 chỗ (≤ 10; 27 khi đếm cả chính script); pnpm lint: 0 lỗi ở mã sản phẩm (frontend/src), 55 lỗi/39 cảnh báo chỉ nằm ở tệp QC e2e/qc2-*, dbg91b.mjs, playwright.qc2-vis.config.ts; today.spec.ts (không @real) chạy trong CI run 44394768852 @a5fb185d (Frontend e2e Playwright 1–4/4 success, gồm today.spec.ts) |
| TC-GATEP2-13 | K(c) | (c) CI GitHub / nhánh ci/ui-drift của PR sprint 3 — ngoài phạm vi đồ án |
| TC-GATEP2-14 | P | HEAD 40a582b8 trên máy QC (không dùng go test ./... một lệnh, chạy TỪNG GÓI): go vet ./... rc=0; sqlc diff rc=0 (không lệch); golangci-lint run rc=0 (log trống); go test -race -count=1 từng gói: 100 gói, 100 lần chạy rc=0 (66 gói pass, 34 gói không có tệp test), 2826 test cấp trên: 2824 pass / 0 fail / 2 skip (llm TestRecordReplay, privacy TestQCRedisMapping — skip do thiếu cờ ghi/cầu QC, không phải Docker) + 606 test con; go test -race -tags integration -count=1 ./internal/integration/... rc=0; log logs/smkT2/gate14-*.log, go-all.jsonl, go-all-int.jsonl |
| TC-GATEP2-15 | P | chạy lại phần không phá volume của cổng PG trên HEAD 40a582b8: go vet ./... và go vet -tags testroutes ./... rc=0 không cảnh báo; golangci-lint rc=0; sqlc diff rc=0; go test -race từng gói 100 gói rc=0; internal/contract không tag 28 test pass và có -tags testroutes 27 test pass (DEF-R2-04 của smoke 2 đã hết); cổng PG trên stack (smoke-2, be0c0d96): các ca chạy được đều P, còn lại LT/X ghi ở smoke-2; cổng P1: 22 P + 1 LT ở smoke-3; 00001/00002 không đổi (sha256 aac87ca60f76… / 14c303b5ebcf… trùng đầu sprint 4, git log --diff-filter=M rỗng); goose max version_id trên ep-r = 27 thay cho 4 của TC (LT: sprint 5–11 thêm 00005–00027, 00001–00004 còn nguyên); không chạy pnpm dev / xoá volume (luật an toàn) |
| TC-GATEP2-16 | P | bản mã sản phẩm: CI run 44394768852 @a5fb185d (sprint/11-p10-eval) success đủ job: Frontend (lint + build), Frontend e2e (Playwright 1/4–4/4, playwright test --grep-invert @real, build:gate), Lighthouse ×4, Go ×11; pnpm lint ở frontend/src 0 lỗi (lỗi chỉ ở tệp QC e2e/qc2-*); ui-antipatterns.sh 34 ✓ rc=0. audit.mjs / sweep.mjs / proto-curl.sh của TC đã bị thay bằng audit.spec.ts và các spec e2e (không còn tệp) — chạy trong shard CI; audit đăng nhập thật 7 tài khoản thuộc TC-P212-21 (qc2). Ghi nhận cho qc2: CI của nhánh qc2/regression (run 44381978247 @40a582b8) ĐỎ vì spec QC e2e/qc2-* bị gom vào playwright test (thiếu /tmp/qc2-s8r/ids.json, ghi docs/regression) và lint các tệp QC — không phải lỗi sản phẩm |
| TC-GATEP2-17 | K(c) | (c) CI GitHub / nhánh ci/ui-drift của PR sprint 3 — ngoài phạm vi đồ án |
| TC-GATEP2-20 | P | mạo danh MSSV bằng HTTP trên ep-r: kẻ đăng ký MSSV của B bằng email khác → xác minh → đăng nhập: /me/courses 0 lớp; threads/members/today/sessions lớp B → 403 ×4; vào lớp bằng mã → 200 bản ghi PENDING + cảnh báo EMAIL_MISMATCH (không ACTIVE, không nối theo MSSV), B vẫn ACTIVE 1 bản ghi; sau đó vẫn 403 ở threads/members; GV thấy hàng chờ có cảnh báo EMAIL_MISMATCH và việc «1 yêu cầu có email chưa khớp MSSV» (kind EMAIL_MISMATCH) ở Hôm nay của lớp; ca nền TC-P203-07, P209-30, P210-14 đều P; smkt2-gate-imp.py |
| TC-GATEP2-21 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP2-22 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP2-23 | P | quét mẫu nguy hiểm (thay diff-review.sh, vì HEAD đã gộp nhiều sprint sau; việc đọc từng dòng diff vẫn là của chủ dự án theo Q-QC-GATEP2-1) trên 12 tệp sản xuất backend-go/internal/auth + httpapi/authhttp: math/rand 0; InsecureSkipVerify 0; TODO/FIXME 0; SigningMethodNone/ParseUnverified 0 (jwt.go:119 WithValidMethods chỉ HS256, iss/aud cố định, leeway 5 s); ghép chuỗi vào SQL 0; slog in token/mật khẩu/email 0 (3 dòng log chỉ in err.Error() của Redis: passwords.go:191,205; session.go:526); token tra bằng sha256 hex trong DB (tokens.go:44), không so sánh bí mật bằng == ; mật khẩu bcrypt; cookie ep_rt ở authhttp/handler.go:143-153 HttpOnly+Secure+SameSite=Lax; next kiểm ở frontend (TC-P202-41/42); không phát hiện nào cần ghi file:dòng |
| TC-GATEP2-24 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP2-25 | P | tạo lại mã rồi dùng mã cũ → 404 JOIN_CODE_INVALID «Mã không hợp lệ hoặc đã hết hạn. Kiểm tra lại với giảng viên.»; cùng mã lỗi và cùng câu chữ với mã không tồn tại, mã bị tắt và mã hết hạn (4 nguyên nhân không phân biệt được; câu chung nêu hai khả năng gộp nên không lộ lý do nào); không có chữ riêng cho đã bị đổi / đã tắt; ca nền TC-P209-18, P209-07 đều P; smkt2-gate-http.py |
| TC-GATEP2-26 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP2-27 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP2-28 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP2-29 | P | tệp 30 dòng dữ liệu, 2 dòng sai MSSV (dòng 7 và 19 của tệp có hàng tiêu đề): dry_run → 200, errors.row = [7, 19]; ca nền TC-P210-06 (UI) P |
| TC-GATEP2-30 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP2-31 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP2-40 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP2-41 | P | HTTP trên ep-r: access trước 200; kẻ trộm xoay T1→T2 200; nạn nhân mang T1 → 401 SESSION_REVOKED; sau đó access cũ 401, access mới của kẻ trộm 401, T2 401 (mọi thứ 401 ≤ 0.11 s từ lúc phát hiện); auth_sessions REFRESH_REUSE/t; audit_log refresh_reuse=1; ca nền TC-P202-11/12/13/14 đều P (smkt3-gate-http.py) |
| TC-GATEP2-42 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP2-43 | P | 50 yêu cầu song song mỗi loại trên ep-r: xác minh: mã {410: 49, 200: 1}, lý do {'used': 49}, code {'LINK_INVALID': 49}, ACTIVE; đặt lại: mã {200: 1, 410: 49}, lý do {'used': 49}, code {'LINK_INVALID': 49}; mời: mã {410: 49, 200: 1}, lý do {'used': 49}, code {'LINK_INVALID': 49}, phiên=1; ca nền TC-P203-13/14, P204-05/06, P206-06/07 đều P |
| TC-GATEP2-44 | F | TC-P201-11 F (DEF-R4-05): token mời rõ nằm trong log Caddy (tiêu đề Next-Url và Next-Router-State-Tree); quét 229 token thật: gateway, worker, postgres, redis, pgbouncer, qc2i-* = 0; token_hash/refresh_hash 64 hex và bcrypt: ca nền TC-P201-09, P201-10, P202-08, P204-09, P206-09 đều P |
| TC-GATEP2-45 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP2-46 | P | SV A xoá phiên của B → 404 NOT_FOUND; Admin xoá qua /me/sessions → 404; id không tồn tại → 404 NOT_FOUND (đồng nhất); phiên B vẫn dùng được (200); SV A / Admin đánh dấu đọc thông báo của B → 404/404, thông báo B vẫn chưa đọc (t); ca nền TC-P204-23, P204-24, P208-23 đều P |
| TC-GATEP2-47 | P | SV sv.kha (không thuộc lớp QS470E05F) gọi 258 thao tác (mọi phương thức) của mọi route lớp theo openapi: 2xx/5xx = 0; /today?user_id=lạ, /members, /threads, header X-Course-Id → [403, 403, 403, 403]; DB lớp trước/sau =; ca nền TC-P207-08/15, P209-37/38, P211-20/21 đều P |
| TC-GATEP2-48 | P | /admin/users với alg=none / sửa role / secret sai → {'alg=none': 401, 'sửa role=ADMIN, giữ chữ ký sai': 401, 'HS256 secret sai': 401} (kỳ vọng 401); SV gửi X-Role/X-User-Id giả → 403; SV POST /admin/users role=ADMIN → 403; GV gọi assign → 403; md5 bảng users trước/sau khớp: True; ca nền TC-P202-29, P203-02, P206-02, P206-22, P206-23, P208-27 đều P |
| TC-GATEP2-49 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP2-50 | P | có–không tài khoản, trên ep-r, 3–7 lần mỗi bên: forgot-password: cùng mã/thân=True (mã [202]), trung vị 85 ms vs 85 ms (tỉ lệ 1.00); login sai: cùng mã/thân=True (mã [401]), trung vị 235 ms vs 233 ms (tỉ lệ 0.99); register: cùng mã/thân=True (mã [202]), trung vị 234 ms vs 230 ms (tỉ lệ 0.99); resend-verification: cùng mã/thân=True (mã [202]), trung vị 61035 ms vs 61043 ms (tỉ lệ 1.00); mã tham gia: cùng mã/thân=True (mã [404]), trung vị 6 ms vs 5 ms (tỉ lệ 0.90); ca nền TC-P202-04/05, P203-04, P204-01, P205-10/28, P206-16, P209-07/08 đều P |
| TC-GATEP2-51 | P | ep-r: 14 lần sai, mỗi lần X-Forwarded-For khác → [401, 401, 401, 401, 401, 429, 429, 429, 429, 429, 429, 429, 429, 429]; đăng nhập ĐÚNG ngay sau → 429 RATE_LIMITED; DB failed_logins/locked 5/f (header giả vô tác dụng); các bước chờ tăng dần, khoá 15 phút (đo thật 899,8 s), IP chặn 40 lần, Admin cũng bị: ca nền TC-P205-01…19, P205-35, P205-36 đều P |
| TC-GATEP2-52 | P | dry_run: {'ELF→.csv': 422, 'PNG→.csv': 422, 'zip bomb .xlsx': 422, 'rỗng': 422} (zip bomb 60 MB giải nén → 422 sau 0.0 s; gateway còn sống 200); tệp công thức =HYPERLINK / @SUM / +1+1: dry_run 200, nhập thật 200; tên lưu trong users: '+1+1' (lưu nguyên văn như chữ, không thực thi); ca nền TC-P210-03/04/36; ghi nhận DEF-R4-02 (.xlsx thực chất là văn bản được đọc như CSV, vẫn qua kiểm tra dòng: Thấp) không thuộc ca độc hại |
| TC-GATEP2-53 | F | quét log 2026-11-19 bằng mật khẩu thử, token, email thử: gateway, worker, redis, pgbouncer, qc2i-* = 0 dòng; Caddy: đường dẫn /api/v1/courses/join/REDACTED và /invite/REDACTED đã che đúng, nhưng 1 dòng chứa token mời rõ ở tiêu đề Next-Url / Next-Router-State-Tree (DEF-R4-05); postgres có dòng email thử nhưng do chính câu lệnh psql của QC (statement log), không phải sản phẩm; repo: chỉ .env.example (và legacy/.env.example) được commit, không .env.local, .pem là gốc tin cậy/testdata công khai |
| TC-GATEP2-99 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P201-01 | P | 00003 và 00004 áp dụng; phiên bản hiện tại 25 |
| TC-P201-02 | P | 4 bảng 00004 có đủ; enum kind: VERIFY_EMAIL,RESET_PASSWORD,INVITE |
| TC-P201-03 | P | git log --diff-filter=M trên 00001/00002 = rỗng (chỉ commit tạo: 240621b1 s1, 01242098 s2); git diff --stat origin/main = 0 dòng; sha256 aac87ca60f76b485 14c303b5ebcf21a0  |
| TC-P201-04 | P | goose up-to 4 → v4; down×2 → v2, enum/bảng của 00003-00004 còn lại=0/0; up-to 4 lại rc=0, v4, 4 bảng auth/mail=4 (DB tạm smkt_goose trên ep-r-postgres, đã xoá); log /tmp/smkT/goose.log |
| TC-P201-05 | P | go test ./internal/store -count=1 -json -v (HEAD 7c9f8b56): 1/1 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P201-06 | P | 3 ca CHECK/enum trên auth_tokens (ROLLBACK): lệch [] |
| TC-P201-07 | P | INSERT hợp lệ vào auth_sessions/auth_tokens/login_attempts/mail_outbox (2 dòng SENT/QUEUED) trong 1 transaction: đếm '1/1/1/2', ROLLBACK, còn lại 0 |
| TC-P201-08 | P | go test ./internal/store -run TestAuthConstraints -v: pass; thân test có 23 ca INSERT sai (token_hash/refresh_hash/revoked/to_addr/status/dedupe/email_hash/outcome…) mỗi ca đúng mã 23514/23505/22P02, require.GreaterOrEqual(len(cases),12); log logs/smkT/go-store.json |
| TC-P201-09 | P | register→token từ thư dài 43 khớp [A-Za-z0-9_-]{43}=True; auth_tokens bản rõ=0, sha256=1; pg_dump toàn DB grep bản rõ=0 |
| TC-P201-10 | P | đo lại: khoá "token/password/link/url" trong payload mail_outbox = 0; pg_dump không có liên kết verify-email?token= |
| TC-P201-11 | F | quét log bằng 229 token thật (33 trong /tmp/smkT/seen.txt + 226 token trích từ thư Mailpit ep-r và qc2i): gateway/worker/postgres/redis/pgbouncer/qc2i-* = 0, nhưng ep-r-caddy-1 = 1 dòng: token mời rõ nằm ở tiêu đề Next-Url và Next-Router-State-Tree (bộ lọc Caddy chỉ che request>uri) → DEF-R4-05; evidence-4/logs/smkT2/p20111-scan.txt |
| TC-P201-12 | P | mọi token_hash dài 64 hex |
| TC-P201-13 | P | 1.000 token phát qua auth.Tokens.Issue (test Go tạm qc_probe_test.go, đã xoá): 1.000 khác nhau, mỗi cái khớp [A-Za-z0-9_-]{43}, tiền tố chung = "" (rỗng), ký tự đầu nhận 64 giá trị, tỉ lệ bit 1 = 0,5000 trên 256.000 bit; go test ./internal/auth -run TestQCProbeTokenRandomness -v |
| TC-P201-14 | P | go test: 3 pass/0 skip/0 fail |
| TC-P201-15 | P | commit (đăng ký hợp lệ): mail_outbox=1 (QUEUED) + outbox topic mail.send payload={"mail_id": "01a1f038-83f1-768a-96ab-1e798c470de4"} (đúng {mail_id}); rollback (đăng ký mật khẩu yếu → 422): mail_outbox 78→78, outbox mail.send 78→78 (không thêm); rollback do lỗi sau khi xếp thư: go test TestEnqueueRollback (xanh) [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P201-16 | P | go test: 2 pass/0 skip/0 fail |
| TC-P201-17 | P | thư mời GV: tiêu đề 'Lời mời tham gia EduPilot'; có Text=True, HTML=True; From=no-reply@edupilot.local; link dùng APP_PUBLIC_URL=https://localhost (thiếu :1143 → cấu hình ep-r; kiểm theo đường dẫn /invite/<token>) |
| TC-P201-18 | P | go test: 1 pass/0 skip/0 fail |
| TC-P201-19 | F | duyệt lại SmkT, đo lại 3 lần (smkt2-p201-19.py, SMTP giả drop, worker riêng, cấu hình mặc định poll 500 ms): 4 lần thử (mail_outbox DEAD attempts=4, dead_at có giá trị, tin outbox.dispatch.dead); mốc cách nhau bậc 1 s = 1,49/1,50/1,50/1,50 s (+50 % > ±30 % của TC), bậc 5 s = 5,5 s (+10 %), bậc 30 s = 30,5 s (+1,7 %): lệch cố định +0,5 s = một nhịp poll → DEF-R4-07 (Thấp, có thể sửa TC) |
| TC-P201-20 | P | last_error dòng DEAD dài 16 (≤1000): 'smtp_unavailable'; không chứa "@"/"token="/nội dung thư; select count(*) from mail_outbox where last_error ~* '@/token=' = 0 [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P201-21 | P | SMTP drop 8 s rồi bật lại (trước lần thử cuối ở ≈36 s): mail_outbox 01a1f039-37c9-76cc-9880-f865e4dce7e1/SENT/3//2026-11-18 19:54:41.647382+00; SMTP giả nhận ACCEPTED 1 lần (đúng 1 thư, không gửi đôi) [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P201-22 | P | go test: 2 pass/0 skip/0 fail |
| TC-P201-23 | P | mail đã SENT (01a1f03a-6aab-74c7-a2f3-e9e719041dae/SENT/0//2026-11-18 19:55:23.146866+00); XADD lại 3 lần cùng outbox_id vào outbox.dispatch → SMTP giả vẫn ACCEPTED 1 lần (khoá dòng mail_outbox.id + SENT chặn); dedupe_key chặn xếp hai thư cùng sự kiện: go test TestDedupeKey (xanh); kill -9 worker giữa lúc gửi + XAUTOCLAIM chưa làm (cần worker thứ hai; TC cho phép chọn XADD lại) [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P201-24 | P | go test: 2 pass/0 skip/0 fail |
| TC-P201-25 | P | full_name='<script>alert(1)</script> "Nguyễn"' qua 7 mẫu thư: verify_email, email_exists, reset_password, password_changed, invite_staff (Admin mời), invite_student (roster GV), account_locked: ở mọi thư HTML không còn <script> thô, có &lt;script&gt; và dấu " thoát (&quot;), phần Text giữ nguyên văn tên, tiêu đề KHÔNG chứa tên (verify "Xác minh email EduPilot của bạn", …); p4t11.py + p4t11b.py (ep-r) + p4t12.py (stack qc2i, account_locked) |
| TC-P201-26 | P | go test: 3 pass/0 skip/0 fail |
| TC-P201-27 | P | khoá payload mail_outbox: answer_message_id,assignment_id,assignment_title,at,class_code,course_id,course_name,criteria,event,event_title,full_name,inviter_name,max,overall_feedback,post_id,role_vn,student_name,teacher_name,thread_id,ticket_id,total,until,user_id |
| TC-P201-28 | P | go test: 1 pass/0 skip/0 fail |
| TC-P201-29 | P | SMTP giả (qc2i-fakesmtp, container riêng) + worker riêng: 550 → DEAD ngay lần đầu (mail_outbox attempts=1, last_error=smtp_permanent_550), log worker chứa email = 0 lần; 451 → thử lại (3 lần sau 20 s, tổng 4 kết nối rồi DEAD như AC6); mẫu không tồn tại (template mau_khong_ton_tai) → DEAD ngay attempts=1 last_error=template_unknown; p4t10.py [stack riêng qc2i] |
| TC-P201-30 | P | go test: 2 pass/0 skip/0 fail |
| TC-P201-31 | P | SQL trần trong internal/auth, internal/mail ngoài store: 0 |
| TC-P201-32 | P | go test: 1 pass/0 skip/0 fail |
| TC-P201-33 | P | topic mail.send chạy trên stream outbox.dispatch (nhóm outbox, 82 consumer lịch sử); consumer hoạt động <120 s: {'16e2ff879aae': 'ep-r-worker-1'}; không consumer nào là container gateway; cmd/gateway không đăng ký mail.send (0), cmd/worker/registry.go có 1 dòng mail |
| TC-P201-34 | P | CI repo kagamikuro1024/EduPilot-v2 run 44343596969 @777bb5e2 success (job "Go (static)" 134940237507: go vet ./... + go vet -tags testroutes + golangci-lint v2.14.0 + scripts/go-test-count.sh; jobs "Go test (auth/course/rest-1/rest-2/rest-3)" chạy go test -race -tags testroutes -v: auth PASS=147 FAIL=0, course PASS=185 FAIL=0; 4 job "Go test không tag"; log /tmp/qc2-ci/go-*.log) + chạy hẹp tại HEAD 7c9f8b56: go test ./internal/contract -count=1 -json → 28 test pass (logs/smkT/go-contract.json); hợp đồng PG/LLM nguyên vẹn |
| TC-P202-01 | P | 200 expires_in=900 user keys=['id', 'email', 'full_name', 'role', 'status', 'email_verified'] |
| TC-P202-02 | P | login 200: auth_sessions 1→2; last_login_at 2026-11-18 19:19:44→2026-11-18 19:19:44 |
| TC-P202-03 | P | header HS256 |
| TC-P202-04 | P | mật khẩu sai vs email không tồn tại → 401 cùng thân: {"code": "INVALID_CREDENTIALS", "message": "Email ho\u1eb7c m\u1eadt kh\u1ea9u kh\u00f4ng  |
| TC-P202-05 | P | trung vị email không tồn tại 1100 ms vs mật khẩu sai 486 ms (tỉ lệ 2.26); 12 mẫu/loại, máy đang tải nặng |
| TC-P202-06 | P | go test: 2 pass/0 skip/0 fail |
| TC-P202-07 | P | refresh 200; sid giữ nguyên=True; cookie xoay=True |
| TC-P202-08 | P | refresh 200; DB: hash mới/prev=sha256(cũ)/rotated_at/last_used = t/t/t/t; bản rõ ở auth_sessions=0, pg_dump bảng=0, log gateway+caddy=0 |
| TC-P202-09 | P | refresh trượt expires_at +336h (mặc định 336h), trần tuyệt đối 720h (720h); kẹp trần: absolute=now+1h → sau refresh expires=60 phút ≤ absolute=60 phút; hết trần (chỉnh DB, ghi rõ) → refresh 401 TOKEN_INVALID |
| TC-P202-10 | P | go test: 4 pass/0 skip/0 fail |
| TC-P202-11 | P | dùng lại T1 → 401 SESSION_REVOKED; T2 → 401; access mới sau thu hồi → 401; audit refresh_reuse=1 |
| TC-P202-12 | P | kẻ trộm xoay trước: T1→T2 200; nạn nhân T1 401 SESSION_REVOKED; cả phiên revoked_reason=REFRESH_REUSE; T2 sau đó 401; thứ tự ngược: T1 tái dùng 401, reason=REFRESH_REUSE |
| TC-P202-13 | P | audit_log (cột before/after thay cho details của spec cũ) action=refresh_reuse entity=auth_session entity_id=phiên bị trộm: 1/auth_session/refresh_reuse/t/; không token/email người trộm; 11 dòng refresh_reuse trong ep-r |
| TC-P202-14 | P | 2 refresh song song cùng T1: [(401, 'SESSION_REVOKED'), (200, None)]; số cookie hợp lệ sinh ra=1; phiên chưa thu hồi/tổng=1/2; không 500 |
| TC-P202-15 | P | go test: 4 pass/0 skip/0 fail |
| TC-P202-16 | P | không cookie 401 UNAUTHENTICATED; cookie lạ 401 TOKEN_INVALID |
| TC-P202-17 | P | go test: 5 pass/0 skip/0 fail |
| TC-P202-18 | P | logout 204,204; refresh A 401; access cũ A 401 SESSION_REVOKED; máy B refresh 200 |
| TC-P202-19 | P | go test: 3 pass/0 skip/0 fail |
| TC-P202-20 | P | Origin evil → 403 {'reason': 'origin'} |
| TC-P202-21 | P | Sec-Fetch-Site cross-site 403; Origin null 403; Origin hợp lệ 200 |
| TC-P202-22 | P | GET 405, PUT 405, form-urlencoded 415 |
| TC-P202-23 | P | Playwright thật trên ep-r (frontend/e2e/qc2-r4/r4ui-auth.spec.ts, qc2-r4 config, HEAD ep-r 7c9f8b56): [desktop] trạng thái refresh từ trang giả: 403; cookie ep_rt SameSite=Lax ; auth_sessions của nạn nhân: còn hiệu lực 2, bị thu hồi 0 |
| TC-P202-24 | P | go test: 4 pass/0 skip/0 fail |
| TC-P202-25 | P | 401 SESSION_REVOKED ≤1 s ở gateway ep-r-gateway-1 (:1143) và gateway thứ hai ep-r-gateway-smkt (:18241, cùng Redis/DB): logout (0,0)s; refresh_reuse (0.01,0.0)s; disabled(admin API): status 200, gw1/gw2 (0.0, 'SESSION_REVOKED')/(0.0, 'SESSION_REVOKED'); password(phiên khác): status 204, gw1/gw2 (0.0, 'SESSION_REVOKED')/(0.0, 'SESSION_REVOKED'); khoá bằng UPDATE users thẳng DB không phát tín hiệu thu hồi (chỉ đường API admin) — xem TC-P206-18 |
| TC-P202-26 | P | Redis dừng 336s: 60 lần gọi API với access hợp lệ (gw1+gw2) → {200: 60}, trễ trung vị 186 ms, tối đa 517 ms; /healthz [200, 200]; log ERROR trong cửa sổ 30 s tối đa {'qc2i-gw1': 1, 'qc2i-gw2': 1} (tổng {'qc2i-gw1': 7, 'qc2i-gw2': 8}; nội dung {'qc2i-gw1': {'Redis không với tới: bỏ qua kiểm thu hồi access token (chấp ': 7}, 'qc2i-gw2': {'Redis không với tới: bỏ qua kiểm thu hồi access token (chấp ': 8}}); Redis về → mốc thu hồi: logout rồi access cũ → {'gw1': (401, 'SESSION_REVOKED', 0.01), 'gw2': (401, 'SESSION_REVOKED', 0.01)} [qc2i-r4apia-redisdown.py, stack qc2i riêng, container qc2i-redis dừng] |
| TC-P202-27 | P | token `gateway token --role ADMIN` (claims không có sid=True) gọi /admin/users: {'test (ep-r :1143)': (200, None), 'dev (:18242)': (200, None), 'production (:18243)': (401, 'TOKEN_INVALID')}; ép ACCESS_TOKEN_TTL=7m → ttl token=420s; gateway production = image edupilot-gateway:latest (không testroutes) ở container tạm ep-r-gateway-smkt-prod |
| TC-P202-28 | P | go test: 5 pass/0 skip/0 fail |
| TC-P202-29 | P | token A(sub) + sid phiên B (ký đúng secret) → 200, dữ liệu vẫn của A (không đoạt phiên B); sid ngẫu nhiên ký đúng secret → 200 (kiểm thu hồi theo sid bị thu hồi, không bắt buộc sid tồn tại); 7 ca giả khác (secret sai, none, exp, nbf, iss, aud, sub đổi) đều 401 |
| TC-P202-30 | P | playwright r4ui-auth (ep-r, spec chỉnh đo bằng request.timing() của trình duyệt thay cho Date.now() hai tab, vì sai số ~50 ms của đo ngoài làm 2 lần chạy trước báo chồng): 2 request /auth/refresh từ 2 tab cùng context: tab 1 bắt đầu 0 ms nhận đầu thư 12 ms xong 13 ms; tab 2 bắt đầu 19 ms (sau khi tab 1 xong) xong 34 ms → tuần tự, không chồng, đúng 2 (≤ 2); auth_sessions còn hiệu lực 4, thu hồi 0; không dòng revoked_reason reuse (navigator.locks ep-refresh trong authSession.ts / earlyRefresh.ts); /tmp/qc2-i/res/r4ui-auth.jsonl |
| TC-P202-31 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P202-32 | P | playwright qc2-r4 (ep-r, HEAD eb903410, spec frontend/e2e/qc2-r4/r4ui-*.spec.ts, kết quả /tmp/qc2-i/res/*.jsonl, log evidence-4/logs/smkT3/pw-run*.out) r4ui-auth desktop: (a) refresh 401 → /login?next=%2Fsettings, refresh gọi 2 lần (1 lúc tải trang + 1 khi 401), không vòng lặp; (b) phát lại vẫn 401 → báo lỗi, không làm mới nữa (refresh 2 lần, GET /me/sessions 2 lần, ở lại /settings); (c) SESSION_REVOKED → không làm mới (refresh 1 lần, chỉ lúc tải), /login?next=%2Fsettings&revoked=refresh_reuse kèm thông báo bị đăng xuất |
| TC-P202-33 | P | Playwright data-layer.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P202-34 | P | qc-real-f1: không token ở storage/cookie đọc được sau đăng nhập+tải lại (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P202-35 | P | bundle build ep-r: 0 tệp; frontend/src: 0 |
| TC-P202-36 | P | playwright r4ui-auth (ep-r, GV): quét outerHTML, mọi thuộc tính data-*, biến toàn cục window (bỏ lớp/hàm gốc của trình trình duyệt như DOMTokenList, MIDIAccess mà lần chạy trước tính nhầm), localStorage/sessionStorage, URL → 0 vi phạm; window.__token = undefined |
| TC-P202-37 | P | Playwright thật trên ep-r (frontend/e2e/qc2-r4/r4ui-auth.spec.ts, qc2-r4 config, HEAD ep-r 7c9f8b56): [desktop] logo cao 40px @desktop ; [mobile] logo cao 32px @mobile |
| TC-P202-38 | P | playwright qc2-r4 (ep-r, HEAD eb903410, spec frontend/e2e/qc2-r4/r4ui-*.spec.ts, kết quả /tmp/qc2-i/res/*.jsonl, log evidence-4/logs/smkT3/pw-run*.out) r4ui-auth desktop: đăng nhập sai → role=alert «Email hoặc mật khẩu không đúng.» (không nêu cái nào sai); bấm đúp nhanh → 1 request /auth/login; email giữ, mật khẩu xoá; nút loading |
| TC-P202-39 | P | Playwright thật trên ep-r (frontend/e2e/qc2-r4/r4ui-auth.spec.ts, qc2-r4 config, HEAD ep-r 7c9f8b56): [mobile] ox=0 cut=0 touch=0 axe=[] |
| TC-P202-40 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P202-41 | P | playwright smkt2-next.spec.ts (ep-r, đăng nhập thật qua /login?next=…, tài khoản qc2iu.gv1, 15 lượt context mới): 11 giá trị độc hại (//evil.example, https://evil.example, /\evil.example, javascript:alert(1), /%2F%2Fevil.example, /%5Cevil.example, /foo%0d%0aSet-Cookie:x=1, /a:b, data:text/html,x, ///evil.example, ' //evil.example') đều về «/», origin giữ https://localhost:1143, 0 request ra ngoài origin, không cookie x; 4 giá trị hợp lệ giữ đường dẫn (/threads, /class/x?tab=a, /settings/llm, /ok?x=//evil); log /tmp/qc2-i/res/smkt2-next.jsonl |
| TC-P202-42 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P202-43 | P | {'admin': 200, 'teacher': 403, 'ta': 403, 'student': 403} |
| TC-P202-44 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P202-45 | P | IDOR chéo vai: SV/GV/TA → 403 |
| TC-P202-46 | F | {'ACCESS_TOKEN_TTL=2h': (1, '"ACCESS_TOKEN_TTL"'), 'ACCESS_TOKEN_TTL=30s': (1, '"ACCESS_TOKEN_TTL"'), 'REFRESH_TOKEN_TTL=30m': (1, '"REFRESH_TOKEN_TTL"'), 'SESSION_ABSOLUTE_TTL=1h': (1, '"SESSION_ABSOLUTE_TTL"'), 'COOKIE_DOMAIN=a b.com': (124, None)}; mặc định 15m/336h/720h (config.go:354-356); COOKIE_DOMAIN="a b.com" KHÔNG bị từ chối (gateway chạy tới hết timeout 20 s, rc=124): chỉ 4 biến TTL bị rc=1; SRS/US AC15 không quy định kiểm COOKIE_DOMAIN nhưng TC có yêu cầu |
| TC-P202-47 | F | duyệt lại SmkT, chạy lại lệnh nguyên văn của TC ngày 2026-11-19: 8 dòng (≠ 0): docs/HUGGINGFACE_DEPLOY.md:85, 5 dòng ở docs/specs/FEAT-account-security (spec nói ACCESS_TOKEN_TTL thay JWT_EXPIRATION), legacy/.env.example:44 và legacy/docker-compose.yml:51 (Project III lưu trữ); mã sản phẩm backend-go/docker-compose*/.env.example/frontend/src = 0; openapi 9/9 đường /auth/*; golden 0 dòng xoá; gateway token --ttl mặc định = ACCESS_TOKEN_TTL (7m→420 s) → DEF-R4-06 (Thấp: tài liệu/TC, không phải mã) |
| TC-P202-48 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P202-49 | P | ep_demo_role/ep_demo_person ngoài DEV_TOOLS: 0 |
| TC-P202-50 | P | curl -sk https://localhost:1143/login → 200; grep -c EduPilot=1; /api/v1/healthz → 200 (cổng QC 1143 thay 443; Caddy: /api/* → gateway, còn lại → frontend) |
| TC-P202-51 | P | qc-real-f1: đăng nhập + tải lại qua Caddy cùng origin: giữ phiên, đúng 1 refresh (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P202-52 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P203-01 | P | 202; DB STUDENT/PENDING_VERIFICATION/f/$2a$ |
| TC-P203-02 | P | 5 trường nâng quyền bị 422, 0 user mới |
| TC-P203-03 | P | go test: 3 pass/0 skip/0 fail |
| TC-P203-04 | P | 3 loại email (mới/ACTIVE/PENDING) → cùng 202 và thân; thời gian [1.85, 2.94, 3.45] |
| TC-P203-05 | P | dăng ký lại cùng email theo trạng thái (đều 202, users=1): mới→"Xác minh email EduPilot của bạn"; ACTIVE→"Bạn đã có tài khoản EduPilot"; PENDING (đo lại đúng: 1 thư xác minh đã có, đăng ký lần 2 → thêm đúng 1 thư "Bạn đã có tài khoản"); INVITED GV→"Lời mời tham gia EduPilot"; INVITED SV có roster→"Bạn được thêm vào lớp"; DISABLED→0 thư [stack riêng qc2i; qc2i-r4apia-acct.py + p4t6.py] |
| TC-P203-06 | P | go test: 5 pass/0 skip/0 fail |
| TC-P203-07 | P | MSSV 296***: đăng ký+xác minh+đăng nhập; /me/courses=0 lớp; enrollments=0 |
| TC-P203-08 | P | 3 biến thể MSSV (/me/courses, enrollments, MSSV lưu) {'chữ thường': (0, '0', 'AB2022908'), 'có khoảng trắng': (0, '0', 'AB2022908'), 'lẫn hoa-thường': (0, '0', 'AB2022908')}; MSSV người đã vào lớp (ACTIVE) → kẻ mạo /me/courses=0 enr=0; đăng ký trước rồi nạp roster: kẻ mạo lớp=0 enr=0, có trong members=False; bản ghi roster theo email nạn nhân: qc2ia-v10.04a50@edupilot.local/ACTIVE [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P203-09 | P | duyệt lại SmkT, chạy lại lệnh TC ngày 2026-11-19: grep student_code(_snapshot)=$ ngoài test = 2 dòng (store/courses.sql.go:228 EnrollmentConflictByStudentCode thuộc danh sách trắng SRS 4.2.5(a); seedtool/activity.go:65 công cụ seed dev, không có đường HTTP) ≤ 3; grep WHERE.*student_code ở store/queries = 1 dòng (courses.sql:172, cùng truy vấn (a), chỉ chặn trùng); 0 truy vấn dùng MSSV nối tài khoản/mở dữ liệu |
| TC-P203-10 | P | gieo zz_qcprobe.sql (where student_code=$1) → test ĐỎ (True); xoá tệp → ok   github.com/edupilot/backend-go/internal/auth 0.499s; git status backend-go=0 dòng (gồm các sửa không phải của QC nếu có) |
| TC-P203-11 | P | {'AB12': 422, 'ABCDEFGHIJKLMNOP': 422, '2022 9002': 422, '2022-9002': 422, '<script>': 422, ' ': 202, 'abc123456': 202}; lưu chữ hoa=ABC123456 |
| TC-P203-12 | P | go test: 3 pass/0 skip/0 fail |
| TC-P203-13 | P | lần1 200; lần2 410 LINK_INVALID {'reason': 'used'}; DB ACTIVE/t |
| TC-P203-14 | P | 50 song song: {410: 49, 200: 1} |
| TC-P203-15 | P | bản rõ token không có trong auth_tokens |
| TC-P203-16 | P | go test: 5 pass/0 skip/0 fail |
| TC-P203-17 | P | hết hạn 410 {'reason': 'expired'}; rác 410 {'reason': 'invalid'}; rỗng 422; thiếu 422 |
| TC-P203-18 | P | gateway phụ: 20 token đoán → 410 rồi 429 ở lần 21 |
| TC-P203-19 | P | go test: 3 pass/0 skip/0 fail |
| TC-P203-20 | P | 202,429 RATE_LIMITED retry=60; thư +1; token cũ 410 |
| TC-P203-21 | P | email lạ 202,429 (giống email thật 202,429); đã xác minh 202 |
| TC-P203-22 | P | go test: 4 pass/0 skip/0 fail |
| TC-P203-23 | P | login 200 email_verified=False; me/courses rỗng; preview 403 EMAIL_NOT_VERIFIED; join 403 EMAIL_NOT_VERIFIED |
| TC-P203-24 | P | roster có email chưa xác minh → enrollments PENDING/EMAIL_UNVERIFIED; sau xác minh email (200) → ACTIVE/ [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P203-25 | P | go test: 2 pass/0 skip/0 fail; thiếu test ['TestUnverifiedCannotJoin'] |
| TC-P203-26 | P | 12 đầu vào; lệch []; email chữ hoa lưu thường=True |
| TC-P203-27 | P | 202; thư HTML thoát <script>=True; users còn bảng |
| TC-P203-28 | P | go test: 1 pass/0 skip/0 fail |
| TC-P203-29 | P | tiêu đề 'Xác minh email EduPilot của bạn'; 1 thư: 1; link True (host trong link: https://localhost — thiếu cổng :1143: cấu hình APP_PUBLIC_URL ep-r) |
| TC-P203-30 | P | go test: 1 pass/0 skip/0 fail |
| TC-P203-31 | P | Playwright thật trên ep-r (frontend/e2e/qc2-r4/r4ui-auth.spec.ts, qc2-r4 config, HEAD ep-r 7c9f8b56): [desktop] nút gửi lại: "Gửi lại thư (60 giây)" |
| TC-P203-32 | P | Playwright thật trên ep-r (frontend/e2e/qc2-r4/r4ui-auth.spec.ts, qc2-r4 config, HEAD ep-r 7c9f8b56): [desktop] Referrer-Policy=no-referrer; request mang token ở Referer: 0; request ra ngoài: 0 |
| TC-P203-33 | P | Playwright thật trên ep-r (frontend/e2e/qc2-r4/r4ui-auth.spec.ts, qc2-r4 config, HEAD ep-r 7c9f8b56): [mobile] /register ox=0 cut=0 touch=0 axe=[] ; /verify-email ox=0 cut=0 touch=0 axe=[] |
| TC-P203-34 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P203-35 | P | register+JWT ADMIN+role→422; không role → vai STUDENT; PATCH 405; PUT /me/profile role 422 |
| TC-P203-36 | P | go test: 2 pass/0 skip/0 fail |
| TC-P203-37 | P | qc-real-f1: đăng ký → Mailpit → xác minh → đăng nhập → / (ô nhập mã lớp) (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P203-38 | P | email/MSSV/mật khẩu thử trong log gateway+worker: 0/0/0 |
| TC-P203-39 | P | go test: 175 pass/0 skip/0 fail |
| TC-P204-01 | P | 3 loại email → cùng 202 + thân; thời gian [0.1, 0.09, 0.09] |
| TC-P204-02 | P | forgot-password → mã {202}; thư mới: ACTIVE ['Đặt lại mật khẩu EduPilot'], PENDING_VERIFICATION ['Đặt lại mật khẩu EduPilot'], INVITED(GV) [], INVITED(SV) [], DISABLED [], không tồn tại [] [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P204-03 | P | go test: 3 pass/0 skip/0 fail |
| TC-P204-04 | P | reset 200; đăng nhập mật khẩu mới 200, cũ 401; token preview 200 ['valid', 'kind', 'expires_at'] |
| TC-P204-05 | P | dùng lại token → 410 {'reason': 'used'} |
| TC-P204-06 | P | 30 song song: {200: 1, 410: 29} |
| TC-P204-07 | P | sau reset: refresh A/B 401/401; access cũ A/B 401/401; revoked_reason=PASSWORD_RESET |
| TC-P204-08 | P | trước: failed_logins/locked_until/status/verified = 10/2026-11-18 19:52:47.743+00/PENDING_VERIFICATION/f; reset 200; sau = 0/NULL/ACTIVE/t; thư password_changed=True; đăng nhập mk mới 200 [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P204-09 | P | token rõ/mật khẩu mới ở DB+log gateway: 0 |
| TC-P204-10 | P | go test: 6 pass/0 skip/0 fail |
| TC-P204-11 | P | mật khẩu yếu [422, 422, 422, 422] (token còn hiệu lực sau 422?); token lạ 410 |
| TC-P204-12 | P | preview token lạ: {410: 20, 429: 3}; 429 đầu tiên ở lần 21 |
| TC-P204-13 | P | go test: 3 pass/0 skip/0 fail |
| TC-P204-14 | P | qc-real-f1: reset mật khẩu máy A → token máy B chết ngay, mở nguội về /login (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P204-15 | P | hai jar: refresh → 401 cả hai |
| TC-P204-16 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P204-17 | P | POST /me/password 204 (khớp openapi và báo cáo gốc; TC ghi 200); phiên hiện tại giữ (refresh 200), phiên khác bị thu hồi (401) |
| TC-P204-18 | P | sai hiện tại 422 [{'field': 'current_password', 'code': 'WRONG_PASSWORD', 'message': 'Mật khẩu hiện tại chưa đúng.'}]; mới=cũ 422; yếu 422; không JWT 401 |
| TC-P204-19 | P | go test: 5 pass/0 skip/0 fail |
| TC-P204-20 | P | khoá ['created_at', 'current', 'device_label', 'id', 'ip_masked', 'last_used_at']; không UA thô/refresh |
| TC-P204-21 | P | sv.gioi 50 phiên, sv.kha 50 phiên, không giao nhau |
| TC-P204-22 | P | go test: 3 pass/0 skip/0 fail |
| TC-P204-23 | P | IDOR DELETE phiên của người khác 404; id lạ 404; "abc" 404; Admin 404; phiên nạn nhân vẫn dùng được (200) |
| TC-P204-24 | P | Admin DELETE /me/sessions/{sid của k} → 404; kèm ?user_id → 404; DELETE /me/sessions?user_id=k → 200; thân {"user_id"} → 200 (hai lệnh sau chỉ tác động phiên KHÁC của chính Admin: 3 phiên Admin bị thu hồi); phiên của k còn 2/2, access của k [200, 200] [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P204-25 | P | DELETE phiên khác 204; (máy bị đăng xuất ≤1 s: kiểm refresh dưới) |
| TC-P204-26 | P | go test: 4 pass/0 skip/0 fail |
| TC-P204-27 | P | playwright qc2-r4 (ep-r, HEAD eb903410, spec frontend/e2e/qc2-r4/r4ui-*.spec.ts, kết quả /tmp/qc2-i/res/*.jsonl, log evidence-4/logs/smkT3/pw-run*.out) r4ui-acct desktop: /forgot-password 1 ô Email + 1 nút primary «Gửi hướng dẫn», cùng câu AC1 cho email có / không, nút «Gửi lại hướng dẫn» khoá có đếm giây; /reset-password token hợp lệ: «Mật khẩu mới» + «Nhập lại mật khẩu» type=password autocomplete=new-password, nêu chính sách, location.search rỗng; thành công «Mật khẩu đã được đổi. Hãy đăng nhập lại.» + «Đăng nhập» href /login; dùng lại / token rác / thiếu token → «Liên kết đã hết hạn hoặc đã được dùng.» + «Yêu cầu liên kết mới»; mật khẩu mới đăng nhập 200, cũ 401 (lần trước thất bại vì Mailpit khởi động lại: fetch failed 0,5 s, lỗi môi trường chạy lại) |
| TC-P204-28 | P | playwright qc2-r4 (ep-r, HEAD eb903410, spec frontend/e2e/qc2-r4/r4ui-*.spec.ts, kết quả /tmp/qc2-i/res/*.jsonl, log evidence-4/logs/smkT3/pw-run*.out) r4ui-acct desktop: /reset-password Referrer-Policy=no-referrer, Cache-Control=no-store; request mang token ở Referer 0, request ra ngoài origin 0; docker logs ep-r-caddy-1 đếm token rõ = 0, dòng reset-password?token=REDACTED = 1 |
| TC-P204-29 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P204-30 | P | playwright qc2-r4 (ep-r, HEAD eb903410, spec frontend/e2e/qc2-r4/r4ui-*.spec.ts, kết quả /tmp/qc2-i/res/*.jsonl, log evidence-4/logs/smkT3/pw-run*.out) r4ui-acct desktop (SV có lớp, 3 phiên): «Đổi mật khẩu» primary, «Đăng xuất mọi thiết bị khác» không primary; đúng 1 hàng «Thiết bị này»; xác nhận nêu «Đăng xuất 2 thiết bị khác. Họ sẽ phải đăng nhập lại.»; Đăng xuất thiết bị này thực hiện ngay (không hộp thoại) + dòng tĩnh «Đã đăng xuất thiết bị này»; còn 1 thiết bị khác → «Đăng xuất 1 thiết bị khác…»; hết → «Không có thiết bị nào khác.»; DB còn 1 phiên hiệu lực. (Sửa kịch bản: hộp «Thời gian học trên EduPilot» chặn SV vừa vào lớp, thêm bước «Đã hiểu») |
| TC-P204-31 | P | playwright qc2-r4 (ep-r, HEAD eb903410, spec frontend/e2e/qc2-r4/r4ui-*.spec.ts, kết quả /tmp/qc2-i/res/*.jsonl, log evidence-4/logs/smkT3/pw-run*.out) r4ui-acct desktop + mobile: desktop chỉ bàn phím (Tab tới «Đăng xuất mọi thiết bị khác» → Enter mở xác nhận «Đăng xuất 2 thiết bị khác» → Esc; Shift+Tab tới «Thao tác cho…» → Enter mở menu «Đăng xuất thiết bị này»); scripts/ui-antipatterns.sh exit 0; axe serious/critical = []; không «session»/«IP» trần, không từ token/refresh/jwt/bcrypt; 375: tràn ngang 0, cắt 0, vùng bấm < 44 px = 0, axe = [] |
| TC-P204-32 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P204-33 | P | không JWT: [401, 401, 401] |
| TC-P204-34 | P | go test: 5 pass/0 skip/0 fail |
| TC-P204-35 | P | preview token hợp lệ 200, sai kind 422; giới hạn 20/phút/IP: 21 lần → 429 đo ở gateway phụ giới hạn mặc định (TC-P204-12) |
| TC-P204-36 | P | go test: 4 pass/0 skip/0 fail |
| TC-P204-37 | P | reset_password đúng lời văn SRS 6.5 (liên kết https://localhost/reset-password?token=…, 30 phút)=True; password_changed đúng lời văn=True, giờ trong thư "02:35 19/11/2026" vs giờ VN hiện tại 02:35 19/11/2026 (lệch 0.8 phút); mật khẩu mới trong thư (Text+HTML) = False; tiêu đề 'Đặt lại mật khẩu EduPilot' / 'Mật khẩu EduPilot của bạn đã được đổi' [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P204-38 | P | go test: 2 pass/0 skip/0 fail |
| TC-P204-39 | P | log gateway không chứa mật khẩu |
| TC-P204-40 | P | go test: 153 pass/0 skip/0 fail |
| TC-P205-01 | P | qc-real-f1 «dò mật khẩu»: 5 lần sai → chờ tăng dần (429/423) (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P205-02 | P | qc-real-f1: trong lúc chờ, đúng mật khẩu cũng bị từ chối (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P205-03 | P | go test: 3 pass/0 skip/0 fail |
| TC-P205-04 | P | qc-real-f1: lần sai thứ 10 → khoá 15 phút (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P205-05 | P | qc-real-f1: đang khoá, đúng mật khẩu bị từ chối (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P205-06 | P | qc-real-f1: đúng MỘT thư cảnh báo khoá (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P205-07 | P | khoá thật (10/2026-11-18 19:52:48.767+00/ACTIVE/t) → ĐẶT locked_until=now()-1s (DB) và hset lock=1 (Redis, mốc quá khứ) → đăng nhập đúng 200; sau: 0/NULL/ACTIVE/t. (Hết khoá thật 15 phút đo ở TC-P205-09) [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P205-08 | P | go test: 4 pass/0 skip/0 fail |
| TC-P205-09 | P | khoá đo thật (không rút gọn): locked_until−mốc khoá = 899.8s (≈900±10); +1:00 → (429, 'LOGIN_THROTTLED', 840, 60.0); +14:30 → (429, 'LOGIN_THROTTLED', 30, 870.0) (kỳ vọng 429, retry_after ≈30); +15:05 → (200, None, None, 905.3) (kỳ vọng 200) [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P205-10 | P | 12 lần song song (thật vs không tồn tại): khác khuôn ở 0 bước []; mẫu mã/retry_after (bước,mã,code,retry) [(1, 401, 'INVALID_CREDENTIALS', None), (2, 401, 'INVALID_CREDENTIALS', None), (3, 401, 'INVALID_CREDENTIALS', None), (4, 401, 'INVALID_CREDENTIALS', None), (5, 401, 'INVALID_CREDENTIALS', None), (6, 429, 'LOGIN_THROTTLED', 2), (7, 429, 'LOGIN_THROTTLED', 2), (8, 429, 'LOGIN_THROTTLED', 2), (9, 429, 'LOGIN_THROTTLED', 2), (10, 429, 'LOGIN_THROTTLED', 2), (11, 429, 'LOGIN_THROTTLED', 2), (12, 429, 'LOGIN_THROTTLED', 2)]; chờ/khoá tới 10: thật 5 lần sai, không tồn tại 5; 429 khoá cùng khuôn= [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P205-11 | P | sau 10 lần sai: email không tồn tại nhận 0 thư; email thật nhận thư ['Tài khoản EduPilot bị khoá tạm thời', 'Xác minh email EduPilot của bạn'] (account_locked đúng 1) [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P205-12 | P | go test: 1 pass/0 skip/0 fail |
| TC-P205-13 | P | 4 sai → failed_logins 4/NULL; 1 đúng → 0/NULL (khoá Redis n=''); 4 sai nữa → 4/NULL (không cộng dồn); 3 sai → 3/NULL; rút gọn: kéo `last` Redis lùi 31 phút (HSET) rồi 1 sai → 1/NULL, Redis n=1 [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P205-14 | P | đang khoá (10/2026-11-18 19:52:48.813+00/ACTIVE/t) → đặt lại qua thư 200; đăng nhập mk mới 200; sau: 0/NULL/ACTIVE/t [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P205-15 | P | go test: 3 pass/0 skip/0 fail |
| TC-P205-16 | P | Redis chết: 10 lần sai tài khoản thật → Counter({401: 10}), DB 10/2026-11-18 20:32:27.825884+00; đăng nhập đúng ngay sau → 429 LOGIN_THROTTLED; email không tồn tại 6 lần sai → [401, 401, 401, 401, 401, 401] (không bị chờ — chấp nhận); tài khoản chưa khoá đăng nhập đúng → 200; log ERROR/30 s ≤ 1; Redis về: tài khoản vẫn khoá (DB) → 429; email mới 6 lần sai → [401, 401, 401, 401, 401, 429] (chờ tăng dần hoạt động lại) [qc2i-r4apia-redisdown.py, stack qc2i riêng, container qc2i-redis dừng] |
| TC-P205-17 | P | go test: 1 pass/0 skip/0 fail |
| TC-P205-18 | P | gateway phụ giới hạn mặc định: login 10×401 rồi 429; register 5/giờ rồi 429; forgot-password 5/giờ rồi 429 (ep-r nới 1000/phút cho gateway chính) |
| TC-P205-19 | P | 40 lần sai rải 4 phút (10/phút) từ 1 IP cho 40 email khác nhau: 30×401 rồi 10×429; đăng nhập ĐÚNG sau đó → 429 RATE_LIMITED |
| TC-P205-20 | P | 11 lần đăng nhập sai xen kẽ gw1/gw2 (email khác nhau, 1 IP, giới hạn mặc định 10/phút/IP): [401, 401, 401, 401, 401, 401, 401, 401, 401, 401, 429] → lần 11 ở gw1 = 429 [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P205-21 | P | Redis chết, 15 đăng nhập sai nhanh từ 1 IP (gw2 giới hạn mặc định 10/phút/IP): {401: 15} → giới hạn IP MỞ CỬA (fail-open) đúng thiết kế; khoá theo tài khoản vẫn dựa DB (TC-P205-16: failed_logins=10, locked_until có) [qc2i-r4apia-redisdown.py, stack qc2i riêng, container qc2i-redis dừng] |
| TC-P205-22 | P | go test: 3 pass/0 skip/0 fail |
| TC-P205-23 | P | register: {'1234567890': (422, ['PASSWORD_COMMON']), 'matkhau12345': (422, ['PASSWORD_COMMON']), 'password1234': (422, ['PASSWORD_COMMON']), 'qwertyuiop12': (422, ['PASSWORD_COMMON']), 'aaaaaaaaaaaa': (422, ['PASSWORD_TOO_LONG']), 'Edupilot2026': (422, ['PASSWORD_COMMON']), 'short': (422, ['PASSWORD |
| TC-P205-24 | P | common_passwords.txt: 22344 dòng, 0 mục ngắn hơn 10 |
| TC-P205-25 | P | 7 yêu cầu mang mật khẩu thử (login sai, register hợp lệ/yếu 422, /me/password sai, reset/accept-invite token lạ): mã [401, 202, 422, 422, 422, 410, 410]; mật khẩu lặp lại trong thân lỗi=0; trong log gw1+gw2+gw3+worker=0 (25293 byte log) [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P205-26 | P | go test: 3 pass/0 skip/0 fail |
| TC-P205-27 | P | tiền tố bcrypt $2a$12$ |
| TC-P205-28 | P | trung vị 30 mẫu/loại: mật khẩu sai 233 ms; email không tồn tại 229 ms (tỉ lệ 0.98); INVITED 229 ms (tỉ lệ 0.98); ngưỡng ≥ 0,65. Xoá bộ đếm sai Redis mỗi 4 mẫu [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P205-29 | P | go test: 2 pass/0 skip/0 fail; thiếu test ['TestDummyCompareRuns'] |
| TC-P205-30 | P | outcome: UNKNOWN_EMAIL:16,SUCCESS:1552,BAD_PASSWORD:12,THROTTLED:14; 0 email rõ trong email_hash |
| TC-P205-31 | P | API đọc login_attempts: [404, 404] |
| TC-P205-32 | P | go test: 2 pass/0 skip/0 fail |
| TC-P205-33 | P | Playwright thật trên ep-r (frontend/e2e/qc2-r4/r4ui-acct.spec.ts, qc2-r4 config, HEAD ep-r 7c9f8b56): [desktop] (1) khoá 3143 ms (≈3,1 s); các chữ đếm ngược: Bạn đã thử quá nhiều lần. Thử lại sau 00:03. / Bạn đã thử quá nhiều lần. Thử lại sau 00:02. / Bạn đã thử quá nhiều |
| TC-P205-34 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P205-35 | P | 12 lần sai, mỗi lần X-Forwarded-For khác: [401, 401, 401, 401, 401, 429, 429, 429, 429, 429, 429, 429] |
| TC-P205-36 | P | ADMIN (vai ADMIN, tài khoản bootstrap `gateway admin create`) sai 10 lần: bị chờ rồi khoá; đăng nhập ĐÚNG sau đó → 429 LOGIN_THROTTLED; DB 10/2026-11-18 19:52:47.453+00/ACTIVE/t [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P205-37 | P | đường mở khoá thủ công: [404, 422] |
| TC-P205-38 | P | go test: 2 pass/0 skip/0 fail |
| TC-P205-39 | P | RỦI RO ĐÃ DUYỆT (QUESTIONS Q3): kẻ xấu sai 10 lần → nạn nhân đăng nhập ĐÚNG → 429 LOGIN_THROTTLED retry_after=900; DB 10/2026-11-18 19:52:48.813+00/ACTIVE/t; lối thoát = đặt lại mật khẩu (TC-14) [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P205-40 | P | go test: 153 pass/0 skip/0 fail |
| TC-P206-01 | P | 201 ['email', 'full_name', 'id', 'last_login_at', 'role', 'status', 'version'] status=INVITED |
| TC-P206-02 | P | 9 vai/trường lạ → 422, 0 user mới |
| TC-P206-03 | P | email trùng 409/409(hoa-thường); idem replay=true; thiếu key 422 |
| TC-P206-04 | P | thư mời tới GV: 'Lời mời tham gia EduPilot' |
| TC-P206-05 | P | accept-invite 200; DB TEACHER/ACTIVE/t; preview 200 |
| TC-P206-06 | P | dùng lại token mời → 410 {'reason': 'used'} |
| TC-P206-07 | P | 30 song song {410: 29, 200: 1}; phiên=1 |
| TC-P206-08 | P | token mời lùi 72h → preview 410 {'reason': 'expired'}, accept 410 {'reason': 'expired'}; 1000 token ngẫu nhiên 43 ký tự (gw3, giới hạn IP nới): {410: 1000}, lý do {'invalid': 1000}, 1 khuôn khoá JSON [('code', 'details', 'message')]; gw1 mặc định 25 lần preview: Counter({410: 20, 429: 5}) (429 đầu ở lần 21; kỳ vọng sau 20/phút/IP) [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P206-09 | P | token mời không ở DB (chỉ băm) |
| TC-P206-10 | P | go test: 6 pass/0 skip/0 fail |
| TC-P206-11 | P | grep openapi (password/token trên dòng /admin/) = 0; 6 phản hồi thật (invite, list×3, patch, resend) khớp password/hash/token = 0; PATCH {"password"} → 422; POST /admin/users/{id}/reset-password → 404; PUT …/password → 404; POST /admin/users/{id} → 405; chuỗi đặt ở thân (họ tên) xuất hiện trong log gateway = 0 [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P206-12 | P | go test ./internal/user -count=1 -json -v (HEAD 7c9f8b56): 2/2 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P206-13 | P | resend 200,429; người ACTIVE 409; TEACHER 403 |
| TC-P206-14 | P | go test ./internal/user -count=1 -json -v (HEAD 7c9f8b56): 3/3 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P206-15 | P | PATCH DISABLED 200; access cũ 401 SESSION_REVOKED; revoked_reason=ACCOUNT_DISABLED |
| TC-P206-16 | P | mật khẩu đúng → 403 ACCOUNT_DISABLED; sai → 401 |
| TC-P206-17 | P | mở khoá 200, đăng nhập lại 200; sai version 409 |
| TC-P206-18 | P | go test ./internal/user -count=1 -json -v (HEAD 7c9f8b56): 6/6 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P206-19 | P | TEACHER→TA 200; →ADMIN 422; →STUDENT 422; tự khoá 409; tự đổi vai 409; admin còn ACTIVE |
| TC-P206-20 | P | go test ./internal/user -count=1 -json -v (HEAD 7c9f8b56): 5/5 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P206-21 | P | 4 thao tác × 4 danh tính: chỉ ADMIN [] |
| TC-P206-22 | P | JWT STUDENT + DB role=ADMIN → /admin/users 403 (vai trong JWT thắng); đã trả vai về STUDENT |
| TC-P206-23 | P | token giả [401, 401] |
| TC-P206-24 | P | IDOR [403, 403, 403] |
| TC-P206-25 | P | ma trận đo qua API ep-r (Go test TestAdminUsersRBACMatrix chỉ có 21 ca, thấp hơn ≥32 của TC nên đo bù qua HTTP): 45 ca = 8 thao tác (GET list/role/q, POST invite, PATCH tên/vai/trạng thái, POST resend-invite) × {TEACHER, TA, STUDENT, không JWT, hết hạn} = 40 ca bị chặn đúng (TEACHER/TA/STUDENT → 403 FORBIDDEN reason=role kể cả GET; không JWT → 401; hết hạn → 401 TOKEN_EXPIRED) + ADMIN 5 ca (3 đọc, 1 mời, 1 gửi lại) đều 2xx; 0 ca lệch; md5 bảng users trước/sau các ca bị chặn khớp; smkt2-p206-25.py |
| TC-P206-26 | P | đi hết 107 người (4 trang, không trùng); limit=101 → 422; lọc role+q 0 |
| TC-P206-27 | P | khoá ['email', 'full_name', 'id', 'last_login_at', 'role', 'status', 'version'] |
| TC-P206-28 | F | users=60048/15 MB (60 000 dòng mô phỏng, ANALYZE): Seq Scan? {'mặc định (trang đầu)': (True, False), 'lọc role': (True, False), 'lọc status': (True, False), 'con trỏ': (True, False), 'tìm theo tên': (True, False)}; mã: List gọi đúng 1 truy vấn ListUsersAdmin (user/service.go: 1 lời gọi), không vòng lặp N+1; EXPLAIN đủ ở evidence-4/qc2i-r4apia-p20628-explain.txt; chỉ mục: users_pkey,users_email_key,users_ics_token_key,users_student_code_idx [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |
| TC-P206-29 | P | go test ./internal/user -count=1 -json -v (HEAD 7c9f8b56): 5/5 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P206-30 | P | DB trống qc2i_empty (migrate up0, users=0): admin create với ADMIN_PASSWORD hợp lệ → rc=0, bản ghi ADMIN/ACTIVE/true, audit_log (actor_id NULL, admin_bootstrap) t/admin_bootstrap; lần hai → rc=0 «Quản trị viên này đã tồn tại; không đổi gì.», md5(password_hash) không đổi (True), count(ADMIN)=1; ADMIN_PASSWORD=123 → rc=1 «admin: password: Mật khẩu cần ít nhất 10 ký tự và không quá 72 byte.», chưa tạo (0); --password trên dòng lệnh → rc=2 «admin: Dùng biến ADMIN_PASSWORD hoặc stdin. Dùng: gateway admin create --email E --name N », chưa tạo (0) (mật khẩu chỉ đọc từ biến môi trường, cli không có cờ --password nên không thể lộ ở ps); không route HTTP tạo ADMIN: /auth/register role=ADMIN → 422, POST /admin/users role=ADMIN → 422 (Admin), 403 (SV); smkt2-p206-30.py |
| TC-P206-31 | P | register role ADMIN 422; admin/users role ADMIN 422 |
| TC-P206-32 | P | Playwright thật trên ep-r (frontend/e2e/qc2-r4/r4ui-admin.spec.ts, qc2-r4 config, HEAD ep-r 7c9f8b56): [desktop] getByRole('dialog') hiện = 0 ; Vai: Giảng viên/Trợ giảng |
| TC-P206-33 | P | playwright r4ui-admin (ep-r, spec sửa cột mail_outbox to_addr): mời → dòng tĩnh «Đã gửi link mời, hạn 72 giờ.» (users INVITED); Gửi lại lời mời ở hàng INVITED → «Đã gửi lại lời mời», mail_outbox ≥ 2 thư cho địa chỉ đó; Khoá tài khoản → «Đã khoá … Hoàn tác» lạc quan, DB DISABLED rồi Hoàn tác → ACTIVE; rỗng «Chưa có người dùng khớp…»; không hộp thoại |
| TC-P206-34 | P | playwright qc2-r4 r4ui-admin mobile 375 /admin/users (ep-r, HEAD eb903410; /tmp/qc2-i/res/r4ui-admin.jsonl): tràn ngang 0, AUDIT cut 0, TOUCH 0, axe serious/critical = [], bảng hiện ở 375 = 0 (thành danh sách), grep useDemoSlice ở admin = 0, không từ kỹ thuật |
| TC-P206-35 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P206-36 | P | Playwright thật trên ep-r (frontend/e2e/qc2-r4/r4ui-admin.spec.ts, qc2-r4 config, HEAD ep-r 7c9f8b56): [desktop] lời chào đúng; preview 1 lần; URL /invite/·; sau đặt mật khẩu → / |
| TC-P206-37 | P | Playwright thật trên ep-r (frontend/e2e/qc2-r4/r4ui-admin.spec.ts, qc2-r4 config, HEAD ep-r 7c9f8b56): [desktop] Referrer-Policy=no-referrer; log Caddy đếm token=0, dòng /invite/REDACTED=4 |
| TC-P206-38 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P206-39 | P | playwright qc2-r4 (ep-r, HEAD eb903410, spec frontend/e2e/qc2-r4/r4ui-*.spec.ts, kết quả /tmp/qc2-i/res/*.jsonl, log evidence-4/logs/smkT3/pw-run*.out) r4ui-admin desktop: 409 email trùng → «Email này đã có tài khoản.» tại ô (aria-invalid, tên giữ); 422 → «Họ và tên cần từ 1 đến 100 ký tự.» tại ô; 403 → alert «Bạn không có quyền thực hiện việc này.» (không lộ FORBIDDEN/403); offline (abort POST) → OfflineBanner «Mất kết nối mạng.», chữ giữ, «Gửi lại» → thành công, 3 lần gửi (gói apiClient tự thử lại 0,3 s + 0,9 s) cùng 1 Idempotency-Key, DB 1 user; VERSION_CONFLICT thật → «Tài khoản này vừa được người khác sửa. Giữ thay đổi của bạn hay dùng bản mới?» (lần trước treo vì kịch bản cho mạng về trước khi hiện «Gửi lại»; đã sửa) |
| TC-P206-40 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P206-41 | P | qc-real-f1: Admin mời GV → Mailpit → đặt mật khẩu → khoá → mất phiên → mở khoá (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P206-42 | P | email GV/token/mật khẩu trong log gateway+worker: 0/0/0 |
| TC-P206-43 | P | go test: 153 pass/0 skip/0 fail |
| TC-P207-01 | P | duyệt lại SmkT: 8/8 bảng, đủ cột documents và content_chunks (course_ids uuid[] + GIN, audience, embedding vector(1536)); chỉ mục phức hợp thuộc lớp bắt đầu bằng course_id, ngoại lệ duy nhất là PK (document_id, course_id) của document_courses đúng SRS FEAT-course-foundation 5.9 (kèm document_courses_course_idx (course_id, document_id)); 00001/00002 không bị sửa |
| TC-P207-02 | P | pg_indexes content_chunks GIN course_ids: CREATE INDEX content_chunks_course_ids_gin ON public.content_chunks USING gin (course_ids); extension vector |
| TC-P207-03 | P | go test -count=1 -run TestCourseSchema ./internal/store (HEAD d935d374) → pass 0,62 s (log logs/smkT2/go-p20703.json); git diff --stat origin/main và git log --diff-filter=M trên migrations 00001*/00002* = rỗng |
| TC-P207-04 | P | 24 INSERT sai mỗi ca trong transaction ROLLBACK đều bị từ chối: 23514 (join_code 0/O/1/I/L/6 ký tự/chữ thường, semester=2026, ARCHIVED⇄archived_at, ARCHIVED+join_enabled, capacity 0/1001, domain x, snapshot cho TA, REMOVED thiếu removed_at, ends<=starts), 23505 (class_code trùng hoa/thường, join_code trùng, enrollments trùng, 2 GV ACTIVE, buổi trùng session_no/starts_at), join_code 8 ký tự bị 22001 (cột character(7), vẫn từ chối); ca lành OK; courses trước/sau 1/1; p4t7.py trên DB qc2i_auth |
| TC-P207-05 | P | go test ./internal/store -run TestCourseConstraints -v: pass; thân test require.GreaterOrEqual(len(cases),20) đạt; log logs/smkT/go-store.json |
| TC-P207-06 | P | EXPLAIN (FORMAT JSON) trên DB tạm smkt2_exp (migrate tới bản mới nhất, analyze), số dòng {'users': 20000, 'courses': 500, 'enrollments': 20000, 'notifications': 20000, 'documents': 500, 'content_chunks': 20000, 'enrollments_courseX': 40}: GetMembership (course_id,user_id) → Index Scan:enrollments_user_idx; ListMyCourses (user_id,status) → Index Scan:enrollments_user_idx, Index Only Scan:courses_pkey, Index Only Scan:users_pkey; ListMembers (course_id,status,…) → Index Only Scan:enrollments_course_status_idx, Index Scan:users_pkey; CountCourseStudents (course_id) → Index Only Scan:enrollments_course_status_idx; GetCourseByJoinCode (join_code) → Index Scan:courses_join_code_key; ChunksForCourse (course_ids @> ARRAY) → Bitmap Index Scan:content_chunks_course_ids_gin; ListNotifications (user_id,created_at desc,id desc) → Index Scan:notifications_user_created_idx; CountUnreadNotifications (user_id) → Index Only Scan:notifications_unread_idx; không Seq Scan ở bảng > 1.000 dòng, chunk dùng Bitmap Index Scan content_chunks_course_ids_gin; kế hoạch đầy đủ logs/smkT2/p20706-explain.txt; smkt2-p207-06.py |
| TC-P207-07 | P | go test ./internal/store -count=1 -json -v (HEAD 7c9f8b56): 1/1 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P207-08 | P | như TC-P207-15 (IDOR lớp): 0 ca lệch |
| TC-P207-09 | P | 144 ca (4 vai JWT × 6 tình trạng [ACTIVE STUDENT/TA/TEACHER, PENDING, REMOVED, không ghi danh] × 6 chế độ: Member=GET today, Staff=POST sessions/generate, Teacher=GET share-sources, StaffOrAdmin=GET members, Manage=GET assistant-candidates, MemberOrAdmin=GET /courses/{id}): lệch ma trận SRS 4.1 = không; mọi từ chối là 403 details.reason=course/role; đổi vai DB STUDENT→TA→STUDENT với cùng JWT: members 403→200→403 (vai lấy từ enrollments, không từ JWT) |
| TC-P207-10 | P | openapi: members là route «TA, giảng viên, Admin» (Manage/xem quản lý) → Admin 200 đúng; today/sessions kiểm riêng ở TC-P211-16 |
| TC-P207-11 | P | gateway riêng smkt2-gw: /courses/abc/today → 404; uuid không có → 403 FORBIDDEN; SV đang ACTIVE → 200; gỡ thành viên (REMOVED) → request kế tiếp 403 ngay (không cache); phục hồi → 200; Postgres chết (alter database qc2i_empty allow_connections false + ngắt phiên, ~4 s, đã bật lại): 3 lần gọi → [(503, 'SERVICE_UNAVAILABLE'), (503, 'SERVICE_UNAVAILABLE'), (503, 'SERVICE_UNAVAILABLE')] (không 200), /readyz 503; sau khi DB về → [200, 200, 200]; smkt2-iso.py |
| TC-P207-12 | P | JWT alg=none / ký secret sai / chữ ký hỏng / rác vào route Teacher (GET share-sources) → 401 cả 4 |
| TC-P207-13 | P | go test: 5 pass/0 skip/0 fail |
| TC-P207-14 | LT | duyệt lại SmkT: gieo route GET /courses/{courseId}/qc-probe không guard → TestAllCourseRoutesGuarded ĐỎ, xoá → xanh (243 route lớp đều 403 với người ngoài lớp). Danh sách adminOnly nay có 4 route thay vì đúng 2 như TC: thêm GET/PUT /courses/{id}/decision-gates của sprint 10.5 (US-DM) theo docs/sprints/10.5/techlead.md:236 và chú thích course_guard_test.go:61 (commit 6853a190); người không phải ADMIN vẫn 403 reason=role. TC lỗi thời do spec đổi hợp lệ ở sprint sau |
| TC-P207-15 | P | 258 thao tác của mọi route lớp (openapi) gọi bởi SV KHÔNG thuộc lớp (sv.kha) vào lớp 2e1a: 2xx/5xx = 0 |
| TC-P207-16 | P | go test ./internal/integration -count=1 -json -v (HEAD 7c9f8b56): 2/2 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P207-17 | P | 25 truy vấn sqlc chạm content_chunks: mọi truy vấn ĐỌC có điều kiện course_ids (24/25 nhắc course_ids; duy nhất KnowledgeChunkDelete là ghi theo id, không đọc); không có hàm đọc chunk không lọc lớp (ChunksForCourse là hàm duy nhất theo lớp); TestChunksForCourseIsolation thân test dựng X∈[c1], Y∈[c1,c2], Z∈[c3] bằng INSERT và kiểm từng lớp (xanh ở lô go test ./internal/store); TestNoUnscopedChunkQuery xanh (lô integration) |
| TC-P207-18 | P | go test ./internal/store -count=1 -json -v (HEAD 7c9f8b56): 1/1 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P207-19 | P | qc-real-f2: bộ chọn lớp theo seed (GV 2 lớp, SV A/B ≥1 lớp ACTIVE, sv.moi chưa lớp) (spec @real ep-r, 4/4 đạt) |
| TC-P207-20 | P | SV vừa xin vào lớp cần duyệt: me/courses [('SHAACDD0', 'PENDING')] |
| TC-P207-21 | P | limit=101 → 422; ETag W/"eAvw8yVlYS1mWshE"; If-None-Match → 304 |
| TC-P207-22 | P | go test: 6 pass/0 skip/0 fail |
| TC-P207-23 | P | SV: không join_code, my_role=STUDENT, GV chỉ {full_name}; Admin 200; mã tham gia của GV lấy qua GET /courses/{id}/join-code (TC-P209-21 đạt) |
| TC-P207-24 | P | go test: 4 pass/0 skip/0 fail |
| TC-P207-25 | P | đổi MSSV ở hồ sơ → student_code_snapshot của enrollments không đổi (khớp mới: 0) |
| TC-P207-26 | P | trường chỉ đọc/sai định dạng → [('role', 422), ('email', 422), ('id', 422), ('status', 422), ('full_name', 422), ('student_code', 422)]; sai version 409; sửa hợp lệ 200 |
| TC-P207-27 | P | go test ./internal/user -count=1 -json -v (HEAD 7c9f8b56): 5/5 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P207-28 | P | GET 200 ['notify_answer_by_mail', 'notify_ticket_by_mail', 'remind_deadline_by_mail', 'reminders', 'study_time', 'track_study_time', 'version']; PUT khoá lạ 422; GET settings người khác 404 |
| TC-P207-29 | P | go test ./internal/user -count=1 -json -v (HEAD 7c9f8b56): 4/4 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P207-30 | P | qc-real-f2: bộ chọn lớp theo seed (UI) (spec @real ep-r, 4/4 đạt) |
| TC-P207-31 | P | Playwright thật trên ep-r (frontend/e2e/qc2-r4/r4ui-class.spec.ts, qc2-r4 config, HEAD ep-r 7c9f8b56): [desktop] request /courses/01a1f03b…/ = 0; ep:ui:course sau tải lại = null |
| TC-P207-32 | P | Playwright class-join.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P207-33 | P | qc-real-f2: SV chưa lớp thấy «Bạn chưa vào lớp nào» + ô nhập mã (spec @real ep-r, 4/4 đạt) |
| TC-P207-34 | P | Playwright class-join.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P207-35 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P207-36 | P | không JWT: [401, 401, 401, 401] |
| TC-P207-37 | P | go test: 4 pass/0 skip/0 fail |
| TC-P207-38 | P | SV GET members/{uid} → 405 (route không tồn tại cho GET), không lộ dữ liệu; liệt kê members → 403 (TC-P209-54) |
| TC-P207-39 | P | go vet/golangci-lint/go test -race toàn cây: bằng chứng CI run 44343596969 @777bb5e2 xanh (Go static + 5 job Go test -race -tags testroutes FAIL=0 + 4 job không tag); go test -tags integration ./internal/integration chạy hẹp tại HEAD: 100 test pass, 0 fail (logs/smkT/goint-integration.json); ngoài ra các gói auth/mail/store/user/course/today/httpapi/platform-outbox/contract chạy hẹp đều pass |
| TC-P208-01 | P | 201; replay=true; count=1; join_code=None  |
| TC-P208-02 | P | RERUN 2026-11-16 sau 56ad200c (DEF-R4-01 đã sửa, migration 00026 áp, goose=26): tạo SK21D91 → 201; sk21d91 → 409 CONFLICT field class_code; ca 422 còn lại không đổi |
| TC-P208-03 | P | 10 song song cùng class_code: {409: 9, 201: 1} |
| TC-P208-04 | P | teacher_id hợp lệ → 201, enrollment GV TEACHER/ACTIVE/ADMIN trong cùng lệnh; sai vai (SV/TA)/uuid lạ/rác → 422 và 0 lớp được tạo (rollback): lệch không |
| TC-P208-05 | P | go test: 4 pass/0 skip/0 fail |
| TC-P208-06 | P | 100.000 mã qua course.NewJoinCode (test Go tạm qc_probe_test.go, đã xoá): đúng 31 ký tự khác nhau, 0 ký tự cấm 0/O/1/I/L, chi-bình-phương=16,67 (df=30, ngưỡng p=0,001 là 59,70), lệch lớn nhất 0,037 điểm phần trăm quanh 3,226 % (≤0,25); grep "math/rand" trong internal/course = 0 (chỉ crypto/rand + math/big) |
| TC-P208-07 | P | thân TestJoinCodeCollisionRetry (xanh): ép trùng mã 4 lần rồi mã mới ở lần 5 → tạo lớp thành công, calls=5; trùng suốt → dừng đúng maxCodeAttempts=5 lần với ErrCodeExhausted, 0 lớp được tạo (count=0), UNIQUE join_code_key giữ ở DB; ErrCodeExhausted không có ánh xạ riêng trong httpapi nên ra 500 chung (không vòng vô hạn) |
| TC-P208-08 | P | go test: 2 pass/0 skip/0 fail; thiếu test ['TestJoinCodeAlphabet', 'TestJoinCodeDistribution'] |
| TC-P208-09 | P | sửa 200; sai version 409 VERSION_CONFLICT; archive 200; lần hai 200 |
| TC-P208-10 | P | lưu trữ lớp 200; thao tác ghi → {'sửa': (409, 'COURSE_ARCHIVED'), 'assign': (409, 'COURSE_ARCHIVED'), 'regenerate': (409, 'COURSE_ARCHIVED'), 'join-settings': (409, 'COURSE_ARCHIVED'), 'approve': (409, 'COURSE_ARCHIVED'), 'DELETE member': (409, 'COURSE_ARCHIVED'), 'import roster': (409, 'COURSE_ARCHIVED'), 'share': (409, 'COURSE_ARCHIVED'), 'assistants': (409, 'COURSE_ARCHIVED')}; DB (courses+enrollments+sessions md5) trước/sau True |
| TC-P208-11 | P | go test: 5 pass/0 skip/0 fail |
| TC-P208-12 | P | assign hợp lệ 200; sai vai/rác → 422 trừ [] |
| TC-P208-13 | P | GV1 trước 200; gán GV2 200; GV1 (access cũ, không đăng nhập lại) 403; GV2 200; DB GV1=REMOVED, GV2=ACTIVE/ADMIN, số GV ACTIVE=1 |
| TC-P208-14 | P | [TA1,TA2]→2 TA; assign {ta_ids:[TA3]} 200 → thay toàn bộ: True, TA1/TA2 REMOVED=2; assign {} 200 và {ta_ids:null} → giữ nguyên (True,True); assign {ta_ids:[]} 200 → còn 0 TA (hành vi ghi nhận: [] = gỡ hết, khớp SRS 4.4) |
| TC-P208-15 | P | gán lại đúng người đang có (GV2,TA1) 200: outbox course.assigned 6→6, notifications 5→5 (không thông báo lại); gán GV1 + TA2 + SV (sai vai) → 422: GV vẫn như cũ=True, TA vẫn 1, outbox 6 (rollback nguyên vẹn); gán GV1 + TA1(cũ) + TA2(mới) hợp lệ 200 → outbox 6→8 (đúng +2: GV1 mới vào lại + TA2 mới; TA1 đã có không thêm); 2 dòng mới nhất: TEACHER:smkt.h1.c35d9@edupilot.local TA:smkt.u2.c35d9@edupilot.local; "ép outbox lỗi" = go test TestAssignNotifyOnce (đạt) |
| TC-P208-16 | P | go test: 8 pass/0 skip/0 fail |
| TC-P208-17 | P | gán GV2 → thông báo COURSE_ASSIGNED sau 0.1s (≤60 s); đúng 1 dòng; tiêu đề/thân/link khớp SRS=True (Bạn được phân công lớp Lớp SmkT 17 151f4 – ST17151…); GET /notifications của GV2 thấy=True |
| TC-P208-18 | P | TA nhận thông báo COURSE_ASSIGNED=3: ['Bạn được phân công làm trợ giảng lớp Lớp SmkB S acdd0 – SHSACDD0'] |
| TC-P208-19 | P | XADD lại tin course.assigned (outbox_id=01a1f030…, entry 1795031053596-0) vào stream outbox.dispatch của ep-r; sau 6 s notifications vẫn 1 dòng (dedupe_key course.assigned:<outbox_id>); (kill -9 worker + XAUTOCLAIM không thực hiện vì là thay đổi toàn cục — dùng XADD lại như TC cho phép) |
| TC-P208-20 | P | go test ./internal/integration -count=1 -json -v (HEAD 7c9f8b56): 1/1 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P208-21 | P | GV lớp thay TA 200; GV lớp khác 403; TA 403; SV 403; candidates 200 |
| TC-P208-22 | P | go test: 4 pass/0 skip/0 fail |
| TC-P208-23 | P | SV-B đọc thông báo của GV2 → 404 NOT_FOUND; id lạ → 404 NOT_FOUND (cùng mã); id rác → 404; GET /notifications của SV-B không có ID_GV=True; read_at của ID_GV vẫn NULL=True |
| TC-P208-24 | P | limit=101 422; limit=1 → 1 mục (next_cursor=True); unread_only: 4 mục chưa đọc |
| TC-P208-25 | P | go test: 5 pass/0 skip/0 fail |
| TC-P208-26 | P | 5 thao tác × 4 danh tính []; GV của lớp tự assign 403 |
| TC-P208-27 | P | POST /admin/courses & assign: {'alg=none': 401, 'sửa role (không ký lại)': 401, 'secret sai': 401, 'rác': 401, 'X-Role: ADMIN': 403, 'GV của lớp tự assign': 403, 'GV2 (GV lớp) assign': 403, 'token cũ sau đổi vai GV→TA': 401}; đổi vai GV2→TA qua API 200; lớp ZZ* tạo ra=0 |
| TC-P208-28 | P | go test: 1 pass/0 skip/0 fail |
| TC-P208-29 | P | khoá mục lớp: ['assistants_count', 'capacity', 'class_code', 'id', 'name', 'semester', 'status', 'students_active', 'students_pending', 'subject_code', 'teacher', 'version'] |
| TC-P208-30 | P | go test: 4 pass/0 skip/0 fail |
| TC-P208-31 | P | Playwright thật trên ep-r (frontend/e2e/qc2-r4/r4ui-admin.spec.ts, qc2-r4 config, HEAD ep-r 7c9f8b56): [desktop] nhấp đúp → 1 lớp QC2IUNMWMJG86J |
| TC-P208-32 | P | playwright qc2-r4 (ep-r, HEAD eb903410, spec frontend/e2e/qc2-r4/r4ui-*.spec.ts, kết quả /tmp/qc2-i/res/*.jsonl, log evidence-4/logs/smkT3/pw-run*.out) r4ui-admin: desktop bàn phím Tab tới «Mở lớp», Enter mở khung tại chỗ, không từ token/session/join_code; mobile 375: tràn ngang 0, cut 0, touch 0, axe = [], bảng hiện = 0 (danh sách); grep useDemoSlice ở admin = 0 |
| TC-P208-33 | P | Playwright class-join.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P208-34 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P208-35 | P | playwright qc2-r4 (ep-r, HEAD eb903410, spec frontend/e2e/qc2-r4/r4ui-*.spec.ts, kết quả /tmp/qc2-i/res/*.jsonl, log evidence-4/logs/smkT3/pw-run*.out) r4ui-admin desktop (SV svd, sửa created_at trong DB): 58 s → «vừa xong» (59 s trừ ≈1 s trễ tải khung), 5 phút → «5 phút trước», 3 giờ → «3 giờ trước», hôm qua 16:40 → «Hôm qua, 16:40», 5 ngày → «Thứ Bảy, 07:04», 10 ngày → «Thứ Hai, 09/11 · 07:04»; grep -rnE 'N (ngày/giờ/phút) trước' frontend/src ngoài shared/lib/timeAgo = 0. Mệnh đề 5 ngày → dd/MM của TC gốc đã đổi từ 9.1 (formatWhen: trong 6 ngày «Thứ X, HH:mm», từ 7 ngày «Thứ X, dd/MM · HH:mm»; src/shared/lib/datetime.ts) nên đo thêm mốc 10 ngày; chữ «Hôm qua, 16:40» (viết hoa + dấu phẩy) thay «hôm qua 16:40» |
| TC-P208-36 | P | Playwright class-join.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P208-37 | P | playwright qc2-r4 (ep-r, HEAD eb903410, spec frontend/e2e/qc2-r4/r4ui-*.spec.ts, kết quả /tmp/qc2-i/res/*.jsonl, log evidence-4/logs/smkT3/pw-run*.out) r4ui-admin desktop: mã lớp trùng 409 → «Mã lớp này đã có.» tại ô (aria-invalid, tên giữ); 422 sai vai → «Người này không phải giảng viên.»; offline → OfflineBanner + «Gửi lại», 3 lần gửi cùng 1 Idempotency-Key, DB 1 lớp; sai version → «Lớp này vừa được người khác sửa. Giữ thay đổi của bạn hay dùng bản mới?»; sửa lớp đã lưu trữ (API lưu trữ khi khung sửa còn mở) → 409 COURSE_ARCHIVED, «Lớp này đã được lưu trữ.» tại khung, chữ giữ; lưu trữ lần hai = 200 không đổi theo TC-P208-09 (đã sửa kịch bản: «lưu trữ lớp đã lưu trữ» của AC12 là thao tác ghi lên lớp ARCHIVED; list API tối đa limit 100) |
| TC-P208-38 | P | Playwright class-join.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P208-39 | P | DB trống riêng qc2i_empty (gateway smkt2-gw + worker smkt2-wk, chỉ Admin bootstrap): Admin mở lớp (201) + gán GV (200/201) → GV thấy thông báo «Bạn được phân công lớp Lớp cô lập – ISO101» sau 0.4 s (≤ 60 s), nội dung có mã tham gia V2G74VB=True; GET /courses/{id}/join-code của GV trả V2G74VB = courses.join_code (True); /me/courses của GV 1 lớp. Phần giao diện (chuông, «Quản lý lớp này») xem TC-GATEP2-24; smkt2-iso.py |
| TC-P208-40 | P | grep openapi POST …/admin/courses/…/(students/members/enroll) = 0; thử /members,/enroll,/students,assign{student_ids} → [404, 404, 404, 422]; enrollments STUDENT mới của lớp = 0; nút "Thêm sinh viên" kiểm ở TC-P208-31 (r4ui-admin) |
| TC-P208-41 | P | go test ./internal/integration -count=1 -json -v (HEAD 7c9f8b56): 1/1 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P208-42 | P | go test: 127 pass/0 skip/0 fail |
| TC-P209-01 | P | preview chữ thường 200 state=OPEN; có khoảng trắng 200; thân ['class_code', 'name', 'semester', 'state', 'teachers'] |
| TC-P209-02 | P | join 200 {'already_member': False, 'course_id': '01a1df57-fc09-769e-8f5d-3d3989668510', 'status': 'ACTIVE'}; DB ACTIVE/CODE |
| TC-P209-03 | P | preview theo trạng thái: lớp cap 1 đã có 1 SV → FULL; lớp bật duyệt → REQUIRES_APPROVAL; SV đã là thành viên → ALREADY_MEMBER; sau join vào lớp bật duyệt (join → 200 PENDING) → PENDING; preview không có mã/id: khoá ['class_code', 'name', 'semester', 'state', 'teachers'] |
| TC-P209-04 | P | go test: 5 pass/0 skip/0 fail |
| TC-P209-05 | P | join lần hai already_member=True; 10 song song → {200: 10}; enrollments: 1/1 |
| TC-P209-06 | P | go test: 2 pass/0 skip/0 fail |
| TC-P209-07 | P | 3/6 nguyên nhân (không tồn tại, mã cũ, tắt) → (404, 'JOIN_CODE_INVALID', 'Mã không hợp lệ hoặc đã hết hạn. Kiểm tra lại với giảng viên.'); giống nhau=True (hết hạn/lưu trữ: xem TC-P209-22) |
| TC-P209-08 | P | trung vị (ms) — 6 nguyên nhân: sai 4.0, tắt 4.1, hết hạn 4.0, miền email 4.1, lớp lưu trữ 4.1, không tồn tại (lệch độ dài) 4.1; chênh 2 % ≤ 35 % (không phân biệt được nguyên nhân bằng thời gian; 30 mẫu mỗi nguyên nhân, lấy xen kẽ vòng tròn) |
| TC-P209-09 | P | đầy → 409 COURSE_FULL «Lớp đã đủ sĩ số. Hãy báo giảng viên.»; cần duyệt → 200 status PENDING; mã sai → 404 JOIN_CODE_INVALID (chỉ hai trạng thái COURSE_FULL / PENDING khác biệt) |
| TC-P209-10 | P | go test: 3 pass/0 skip/0 fail |
| TC-P209-11 | P | qc-real-f2: 6 mã sai → 429 (UI 375×812) (spec @real ep-r, 4/4 đạt) |
| TC-P209-12 | P | 4 sai {404}; 10 join thành công (already_member) [200] + COURSE_FULL 409 không đếm; sai lần 5 → 404; lần 6 → 429 (đã 5 thất bại); lùi điểm đếm 11 phút (chỉnh score ZSET, rút gọn 10 phút — ghi rõ) rồi thử lại → 404 (cửa sổ trượt 10 phút) |
| TC-P209-13 | P | 21 tài khoản khác nhau, mỗi người 1 mã sai, cùng một IP (172.22.0.1): 20 lần đầu {404}, lần 21 → 429 (20/10 phút/IP); tài khoản thứ 22 cùng IP → 429; cùng một mã sai gửi từ IP khác (container frontend → gateway:8080 nội bộ) → 404 (không bị chặn) |
| TC-P209-14 | P | 2 gateway dùng chung Redis (qc2i-gw1 :18221 / qc2i-gw2 :18222, DB qc2i_auth), xen kẽ 5 mã sai ([404, 404, 404, 404, 404]) → lần 6 ở gw2 429 và ở gw1 429; ZCARD ep:join:fail:user = 5 (đếm chung); dừng qc2i-redis: 7 lần thử sai (không Idempotency-Key, như UI join) → [404] (mở cửa, 4.8 s); ghi nhận: cùng lời gọi có Idempotency-Key → 503 SERVICE_UNAVAILABLE details {'redis': 'down'} (bộ chống trùng yêu cầu cần Redis, đóng cửa theo thiết kế); log: 51 dòng ERROR về giới hạn/redis; số lần mã đã thử xuất hiện trong log = 0 |
| TC-P209-15 | P | 100 mã ngẫu nhiên (7 ký tự bảng 31) một tài khoản: trúng 0; 5 lần đầu 404. 95 lần sau toàn 429; 5 tài khoản × 5 lần: 20 × 404 rồi 5 × 429 theo IP (20/10 phút). Xác suất trúng của kẻ có nhiều IP + tài khoản ≈ N/31^7 ≈ 2.7·10⁻¹¹ mỗi lần (≈ 1/27.512.614.111) |
| TC-P209-16 | P | trung vị 30 mẫu: mã đúng 6/7 ký tự 6.2 ms vs sai hoàn toàn 6.2 ms (chênh 1 % ≤ 35 %) |
| TC-P209-17 | P | go test: 6 pass/0 skip/0 fail |
| TC-P209-18 | P | qc-real-f2: tạo lại mã, mã cũ chết ngay (UI 375×812) (spec @real ep-r, 4/4 đạt) |
| TC-P209-19 | P | regenerate 200; audit_log (entity course, action join_code_regenerated) có actor_id (người làm)=t; before/after «{"code_prefix": "QV"} → {"code_prefix": "DT"}» — chỉ 2 ký tự đầu, không có mã cũ (QVHDAQ4) hay mới (DTXUFPU) dạng rõ (SQL: select before, after, actor_id from audit_log where action=join_code_regenerated; TC ghi cột «details» — schema thật là before/after) |
| TC-P209-20 | P | go test: 5 pass/0 skip/0 fail |
| TC-P209-21 | P | GET join-code {'TA': 200, 'GV': 200, 'ADMIN': 200, 'SV': 403} |
| TC-P209-22 | P | tắt/hết hạn/miền email → 404 JOIN_CODE_INVALID (cùng một câu); capacity=số hiện có → 409 COURSE_FULL; require_approval → 200 PENDING; tắt: 404 JOIN_CODE_INVALID; hết hạn: 404 JOIN_CODE_INVALID; miền email: 404 JOIN_CODE_INVALID; sĩ số: 409 COURSE_FULL; duyệt: 200 PENDING |
| TC-P209-23 | P | go test: 8 pass/0 skip/0 fail |
| TC-P209-24 | P | bật duyệt 200; join → 200 PENDING; PENDING gọi members 403, today 403 |
| TC-P209-25 | P | go test: 4 pass/0 skip/0 fail |
| TC-P209-26 | P | TA duyệt 200; GV từ chối 200; TA mời ra → 403 (kỳ vọng 403) |
| TC-P209-27 | P | go test: 8 pass/0 skip/0 fail |
| TC-P209-28 | P | trước 200; GV mời ra 200; access cũ gọi ngay → 403 |
| TC-P209-29 | P | go test: 3 pass/0 skip/0 fail |
| TC-P209-30 | P | qc-real-f1: đăng ký bằng MSSV của SV trong lớp, email khác → PENDING EMAIL_MISMATCH + cảnh báo, không thấy dữ liệu SV thật (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P209-31 | P | X khai MSSV trùng B (20226980) → join 200 PENDING; DB X: PENDING/EMAIL_MISMATCH/20226980; TA approve → 403 reason mismatch_needs_teacher; GV approve thiếu confirm_mismatch → 422; có cờ → 200 ACTIVE; student_code_snapshot của X = MSSV tự khai; danh sách có cả X và B (4 hàng, hai user_id khác nhau) |
| TC-P209-32 | P | X vào trước (join 200 ACTIVE, DB trạng thái ACTIVE); B (MSSV 20222003) vào sau → join 200 PENDING; DB hàng B: PENDING/EMAIL_MISMATCH; B chưa đọc được dữ liệu lớp (403); hai người không đọc được dữ liệu của nhau |
| TC-P209-33 | P | X (MSSV trùng B, đã được GV xác nhận) gọi me/today (200), me/courses (200), courses/{id}/today (200): không có user_id/email/tên của B trong thân; /members → 403; rò = [] |
| TC-P209-34 | P | go test ./internal/integration -count=1 -json -v (HEAD 7c9f8b56): 4/4 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P209-35 | P | {'unverified': 403, 'GV': 403, 'TA': 403, 'ADMIN': 403, 'none': 401} |
| TC-P209-36 | P | go test: 3 pass/0 skip/0 fail |
| TC-P209-37 | P | SV (ACTIVE) → route quản lý: đều 403  (4/9 thao tác) |
| TC-P209-38 | P | GV lớp mới approve/DELETE uid lớp 2 → 404/404, SV DELETE → 403; (đếm DB lớp 2 không đối chiếu) |
| TC-P209-39 | P | go test: 1 pass/0 skip/0 fail |
| TC-P209-40 | P | limit=101 → 422; khoá thành viên ['email', 'full_name', 'joined_via', 'role_in_course', 'status', 'status_changed_at', 'student_code', 'user_id', 'warning'] |
| TC-P209-41 | P | go test: 4 pass/0 skip/0 fail |
| TC-P209-54 | P | SV gọi members [403, 403, 403] |
| TC-P209-42 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P209-43 | P | chưa đăng nhập mở /join/VFC45PW → https://localhost:1143/login?next=%2Fjoin%2F<mã>; sau đăng nhập → https://localhost:1143/join (mã bị xoá khỏi thanh địa chỉ bằng history.replaceState); xem trước hiện «Tham gia lớp Mã tham gia 7 ký tự, không có 0, O, 1, I, L. Lớp QC2S4 QSUR00C · QSUR00C · TS. Lê Thu Hà · HK1 20»; tràn ngang 0; log Caddy: 2 dòng chứa mã (mã nằm trong đường dẫn /join/<mã> — ghi nhận Q-QC), dòng ngoài đường dẫn/invite: 2 |
| TC-P209-44 | P | Playwright class-join.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P209-45 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P209-46 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P209-47 | P | Playwright class-join.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P209-48 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P209-49 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P209-50 | P | Playwright class-join.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P209-51 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P209-52 | P | mã tham gia trong log gateway+worker: 0 |
| TC-P209-53 | P | go test: 127 pass/0 skip/0 fail |
| TC-P210-01 | P | dry_run ok-30.csv total=30; bom-semicolon.csv 200 total=5; ok.xlsx 200 total=5 |
| TC-P210-02 | P | 501 dòng 422; >2MiB 413; thiếu cột 422; thiếu Idempotency-Key 422 |
| TC-P210-03 | P | RERUN 2026-11-16 sau b6d20a67 (DEF-R4-02 đã sửa): text.xlsx, cut.xlsx, fake-ext.csv, png-as.csv (dry_run) → 422 VALIDATION_FAILED, details[file]=UNSUPPORTED_FILE; ok.xlsx 200 total=5, ok-30.csv 200 |
| TC-P210-04 | P | zip bomb → 422 sau 0.0s; gateway còn sống (200) |
| TC-P210-05 | P | go test: 2 pass/0 skip/0 fail; thiếu test ['TestRosterCSVParse', 'TestRosterXLSXParse', 'TestRosterBOMAndSemicolon', 'TestRosterRowLimit500', 'TestRosterMagicBytes', 'TestRosterZipBomb', 'TestRosterMissingColumn'] |
| TC-P210-06 | P | qc-real-f2: import roster-2-loi.csv báo đúng dòng 7 và 19 (UI) (spec @real ep-r, 4/4 đạt) |
| TC-P210-07 | P | dry_run: users/enrollments/mail_outbox/audit_log trước 122/257/417/2798 sau 122/257/417/2798 |
| TC-P210-08 | P | import thật loi-2.csv → 200, created_users=28, 2 dòng lỗi không ghi; lớp có 30 enrollments (28 SV + GV + TA) |
| TC-P210-09 | P | dry_run tệp 13 dòng → từng dòng đúng mã: {3: 'DUPLICATE_EMAIL_IN_FILE', 5: 'DUPLICATE_STUDENT_CODE_IN_FILE', 6: 'STUDENT_CODE_CONFLICT', 7: 'INVALID_STUDENT_CODE', 8: 'EMAIL_BELONGS_TO_STAFF', 9: 'EMAIL_DISABLED', 10: 'COURSE_FULL', 11: 'COURSE_FULL', 12: 'COURSE_FULL', 13: 'COURSE_FULL'}; hai mã MSSV không lẫn nghĩa (DUPLICATE_STUDENT_CODE_IN_FILE ở dòng 5 vs STUDENT_CODE_CONFLICT ở dòng 6); COURSE_FULL xuất hiện khi vượt sĩ số 3 |
| TC-P210-10 | P | go test: 5 pass/0 skip/0 fail |
| TC-P210-11 | P | import 3 dòng: 1 tạo mới / 1 nối / 1 chờ xác minh; enrollments {'gvh': ['ACTIVE', 'ADMIN', ''], 'tah': ['ACTIVE', 'ADMIN', ''], 'mok': ['ACTIVE', 'ROSTER', ''], 'mun': ['PENDING', 'ROSTER', 'EMAIL_UNVERIFIED'], 'new': ['ACTIVE', 'ROSTER', '']} |
| TC-P210-12 | P | xác minh email → enrollment PENDING → ACTIVE tự động |
| TC-P210-13 | P | go test: 5 pass/0 skip/0 fail; thiếu test ['TestRosterLinkMatrix'] |
| TC-P210-14 | P | MSSV 2022*** của roster bằng email khác (đã xác minh): me/courses=0; enrollments=0; today 403; members 403 |
| TC-P210-15 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P210-16 | P | go test ./internal/integration -count=1 -json -v (HEAD 7c9f8b56): 1/1 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P210-17 | P | đăng ký bằng email roster: 202 cùng thân email lạ=True; đăng nhập bằng mật khẩu kẻ tấn công 401; tài khoản PENDING_VERIFICATION |
| TC-P210-18 | P | import → 200; chủ hộp thư nhận thư mời, accept-invite (token trong liên kết, mật khẩu mới) → 200; login → 200; me/courses có lớp; courses/{id}/today 200; enrollment.id trước/sau accept giống, tổng bản ghi enrollments của user = 1 (không bản ghi mới) |
| TC-P210-19 | P | go test ./internal/integration -count=1 -json -v (HEAD 7c9f8b56): 2/2 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P210-20 | P | import 30 dòng → 200 (enrollments STUDENT 30); B tự join → 200 already_member=True, enrollments vẫn 30; import lại cùng tệp: already_member 30 = total 30; MSSV lặp trong tệp → ['DUPLICATE_STUDENT_CODE_IN_FILE']; MSSV của B ở email khác → ['STUDENT_CODE_CONFLICT'] |
| TC-P210-21 | P | go test: 2 pass/0 skip/0 fail; thiếu test ['TestJoinByCodeAfterRosterNoDuplicate'] |
| TC-P210-22 | P | import: {'GV': 422, 'TA': 403, 'ADMIN': 403, 'SV': 403, 'none': 401} (GV dùng tệp thiếu cột → 422 là qua quyền) |
| TC-P210-23 | P | go test: 1 pass/0 skip/0 fail |
| TC-P210-24 | P | import 30 dòng (27 chưa có tài khoản): 27 thư «Bạn được thêm vào lớp» (mỗi người 1); import lại: tổng thư mỗi người vẫn [1] (không gửi lại); send_invites=false (4 dòng mới): 0 thư, tài khoản INVITED; thư mẫu: «Chào Mời 0, Giảng viên TS. Lê Thu Hà đã thêm bạn vào lớp Lớp QC2S4 QSP0D2 – QSP0D2 trên EduPilot. Hãy đặt mật khẩu để vào lớp (liên kết dùng một lần, có hiệu»; MSSV trong thư: 0; tên GV trong thư: True |
| TC-P210-25 | P | go test: 3 pass/0 skip/0 fail |
| TC-P210-26 | P | share-from lớp 761987 (nguồn: 11 tài liệu thật READY, trong đó 1 COURSE_POLICY, và 1 tài liệu ảo THREAD_KNOWLEDGE không được chia sẻ theo AC11) → 200 shared {'documents': 10}, skipped [{"kind": "documents", "reason": "Quy chế môn học không dùng lại được: mỗi lớp có quy chế riêng."}]; document_courses mới = 10 (không có COURSE_POLICY: LECTURE,LECTURE,OTHER,EXAM_PAPER,ANSWER_KEY,EXAM_PAPER,ANSWER_KEY,EXAM_PAPER,EXAM_PAPER,ANSWER_KEY); chunk có course_ids chứa lớp đích đủ (thiếu 0); số chunk + md5 embedding nguồn trước/sau 135/135 khớp (không nhúng lại); chia sẻ lại → 200, document_courses 10, md5 không đổi |
| TC-P210-27 | P | TA [403]; khác học phần 422; loại chưa hỗ trợ 422 |
| TC-P210-28 | P | go test ./internal/course -count=1 -json -v (HEAD 7c9f8b56): 7/7 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P210-29 | P | lớp 3 (không chia sẻ): documents 0, content_chunks có course_ids chứa lớp 3 = 0; SV lớp đích gọi documents/library → 403/200 không có tiêu đề ANSWER_KEY / visible_to_students=false (6 tiêu đề kiểm, rò = []); SV chỉ lớp 1 vào lớp 2 → 403; GV lớp đích PATCH/DELETE tài liệu nguồn qua đường lớp đích → 409/409 DOCUMENT_SHARED_READONLY «chỉ lớp gốc mới sửa được» (nguồn còn nguyên: 1, tiêu đề «Network Security Threats»); chia sẻ một chiều |
| TC-P210-30 | P | go test ./internal/integration -count=1 -json -v (HEAD 7c9f8b56): 3/3 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P210-31 | P | GV (teacher của cả hai lớp): 66 nguồn, đều cùng subject_code INT1006 và ACTIVE; có lớp 761987, không có lớp khác học phần (QCOTH) và lớp đã lưu trữ, không có chính lớp đích; thân mẫu [{"id": "01a1df27-62b5-7775-a346-5c602a43e2f9", "class_code": "761987", "name": "An ninh mạng", "semester": "2026-2027-HK1", "documents": 10}]; TA → 403; SV → 403 |
| TC-P210-32 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P210-33 | P | Playwright class-join.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P210-34 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P210-35 | P | Playwright class-join.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P210-36 | P | dry_run formula/injection/control/tên dài/dòng dài → {'formula.csv': 200, 'injection.csv': 200, 'control.csv': 422, 'ten-dai.csv': 200, 'dong-dai.csv': 422}; tên lưu: '=HYPERLINK("http://evil","x") ¦ +cmd/\' /C calc\'!A0 ¦ @SUM(1) ¦ <script>alert(1)</script> ¦ \'; drop table users;--' (lưu nguyên vă |
| TC-P210-37 | P | import formula 200, injection 200; bảng users còn nguyên; thân thư HTML thoát (TC-P203-27) |
| TC-P210-38 | P | go test ./internal/course -count=1 -json -v (HEAD 7c9f8b56): 4/4 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P210-39 | P | email/MSSV roster trong log: 0/0 |
| TC-P210-40 | P | go test: 127 pass/0 skip/0 fail |
| TC-P211-01 | P | grep import internal/llm ở internal/today (không _test) = 0; go doc: Provider{Name(); Items(ctx, Viewer, Scope) ([]Item, error)}, Register(p) trên Aggregator; Viewer gồm UserID, Role, EmailVerified, Courses (chỉ lớp ACTIVE theo chú thích), thêm MaskedEmail và Now (bổ sung sau, không nhận id người/lớp ngoài Viewer) |
| TC-P211-02 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P211-03 | P | go test ./internal/today -count=1 -json -v (HEAD 7c9f8b56): 7/7 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P211-04 | P | GV teacher: 50 mục (count 361) gồm ['AI_CONFIRM', 'CHECKIN_REVIEW', 'EMAIL_MISMATCH', 'GRADING_ERROR', 'GRADING_REVIEW', 'JOIN_REQUEST', 'POST_REPORTED', 'PUBLISH_PENDING', 'SESSION_UNMARKED', 'TICKET']; thứ tự khớp khoá (quá hạn trước, bậc tăng dần theo hằng Tier* trong today.go, tuổi giảm dần, mã lớp tăng dần, id tăng dần) cho 50 mục có bậc đã biết (bỏ qua loại chưa ánh xạ: []); ; JOIN_REQUEST quá 48 h (3 mục) nằm nhóm quá hạn, xếp theo tuổi giảm dần; danh sách cắt ở 50 mục nên COURSE_SETUP (bậc 80) kiểm ở màn một lớp: thứ tự ['GRADE_SCHEME_UNCONFIRMED', 'COURSE_SETUP'] (GRADE_SCHEME_UNCONFIRMED 70 trước COURSE_SETUP 80) |
| TC-P211-05 | P | chưa lớp: no_course=True recommended=JOIN_CODE |
| TC-P211-06 | P | chưa xác minh: recommended=VERIFY_EMAIL |
| TC-P211-07 | P | chỉ PENDING: recommended=JOIN_PENDING no_course=True |
| TC-P211-08 | P | SV mới verify, 2 lớp, mỗi lớp 1 buổi hôm nay (sessions/generate 201/201): khoá ['continue', 'email_verified', 'no_course', 'recommended', 'timeline']; recommended=None; timeline 2 mục: [[null, "QST125F"], [null, "QST2B9A"]]; continue []; courses/{id}/today → 200 |
| TC-P211-09 | P | go test ./internal/today -count=1 -json -v (HEAD 7c9f8b56): 6/6 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P211-10 | P | GV me/today khoá ['actions', 'attention', 'attention_total', 'count', 'upcoming']; courses/{id}/today 200; TA khoá ['actions', 'attention', 'attention_total', 'count', 'upcoming'] |
| TC-P211-11 | P | go test ./internal/today -count=1 -json -v (HEAD 7c9f8b56): 5/5 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P211-12 | P | GV thấy việc: ['JOIN_REQUEST', 'GRADE_SCHEME_UNCONFIRMED', 'GRADE_SCHEME_UNCONFIRMED', 'GRADE_SCHEME_UNCONFIRMED', 'COURSE_SETUP', 'COURSE_SETUP'] |
| TC-P211-13 | P | COURSE_SETUP: ban đầu «0/4 bước xong: chia sẻ mã lớp → tải quy chế môn học → tạo lịch buổi học → tải tài liệu.»; sau khi có SV vào lớp «1/4 bước xong: chia sẻ mã lớp → tải quy chế môn học → tạo lịch buổi học → tải tài liệu.»; sau tạo buổi «2/4 bước xong: chia sẻ mã lớp → tải quy chế môn học → tạo lịch buổi học → tải tài liệu.»; sau thêm quy chế + tài liệu bằng SQL fixture (không qua outbox → chỉ hết hạn nhờ TTL 60 s của cache, đo 60 s) → mục biến mất; setup/dismiss lần 1/2 → 204/204 (idempotent), mục ẩn khỏi today: True |
| TC-P211-14 | P | go test ./internal/today -count=1 -json -v (HEAD 7c9f8b56): 6/6 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P211-15 | P | Admin me/today 200; việc: ['Nhà cung cấp AI đang lỗi', 'Lớp SJ20D91 chưa có giảng viên'] |
| TC-P211-16 | P | Admin courses/{id}/today → 403 |
| TC-P211-17 | P | go test ./internal/today -count=1 -json -v (HEAD 7c9f8b56): 5/5 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P211-18 | P | 4 người (A: 2 lớp; B: lớp 1; X: PENDING vì MSSV trùng B; D: chưa có lớp) gọi me/today + courses/<lớp mình>/today: A: me/today=200, courses/01a1f035/today=200; B: me/today=200, courses/01a1f035/today=200; X: me/today=200, courses/01a1f035/today=403; D: me/today=200; quét user_id, họ tên, email, MSSV của người khác, join_code, id lớp ngoài trong mọi thân: vi phạm = []; X vào today lớp đang chờ → 403; D vào today lớp lạ → 403 |
| TC-P211-19 | P | kind chỉ-dành-cho-staff trong today của SV: 0 |
| TC-P211-20 | P | PENDING / ngoài lớp / SV lớp khác gọi courses/{id}/today: [403, 403, 403] |
| TC-P211-21 | P | 60 cặp (người, lớp) ngẫu nhiên: 2xx ⇔ thành viên ACTIVE; lệch [] |
| TC-P211-22 | P | go test ./internal/today -count=1 -json -v (HEAD 7c9f8b56): 4/4 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P211-23 | P | không JWT 401; hết hạn 401; PENDING + user_id giả 403 |
| TC-P211-24 | P | go test ./internal/today -count=1 -json -v (HEAD 7c9f8b56): 1/1 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P211-25 | P | khoá ep:today:<uid>:all ttl=60 (≤60) |
| TC-P211-26 | P | duyệt yêu cầu → JOIN_REQUEST của lớp biến khỏi today GV sau 0.58 s (trước đó có=True) |
| TC-P211-27 | P | trước 200; mời ra 200; me/today ngay sau không còn lớp: True |
| TC-P211-28 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P211-29 | P | go test ./internal/today -count=1 -json -v (HEAD 7c9f8b56): 4/4 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P211-30 | N | chưa chạy (mẫu): tải/đo hiệu năng nặng (k6, Lighthouse) — chưa thu nhỏ để chạy trên ep-r |
| TC-P211-31 | P | go test ./internal/today -count=1 -json -v (HEAD 7c9f8b56): 2/2 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P211-32 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P211-33 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P211-34 | P | qc2 2026-11-19 @40a582b8: build:gate xong rc=0 rồi pnpm exec playwright test e2e/today.spec.ts -g 'student today' → desktop+mobile 14 passed, rc=0 (log /tmp/qc2-s4-gate.out chép vào evidence-4/logs/qc2-gate-today.out) |
| TC-P211-35 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P211-36 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P211-37 | P | cùng build: -g 'staff today/admin today' → 16 passed, rc=0 (desktop 1440 + mobile 390) |
| TC-P211-38 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P211-39 | P | cùng build: -g 'no mock items' → 2 passed (desktop + mobile), rc=0 |
| TC-P211-40 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P211-41 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P211-42 | P | go test ./internal/today -run 'TestTodayProviderErrorPartial/TestTodayDeadline504' -v: PASS cả hai (0,70 s / 0,49 s); playwright today.spec.ts -g 'today errors' → 2 passed rc=0 |
| TC-P211-43 | P | llm_audit trước/sau 3 lần me/today: 2235→2235 (chênh do nhóm khác dùng chung stack) |
| TC-P211-44 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P212-01 | P | seed.mjs: 0 psql/INSERT/pg/_test; ≥1 api/v1 |
| TC-P212-02 | P | qc2 2026-11-19 trên DB trống qc2i_seed (stack qc2s, PM duyệt): lấy mẫu pg_stat_activity mỗi 2 s × 25 lần trong lúc seed.mjs chạy lại — mọi kết nối tới DB chỉ từ 172.22.0.24 (application_name edupilot-worker, 10 kết nối) và 172.22.0.23 (edupilot-gateway, 4 kết nối); không kết nối nào từ máy chạy seed; grep -nE "psql/INSERT /UPDATE /require.pg./from .pg./_test/" scripts/seed.mjs = 0. Ghi chú: log_line_prefix của Postgres ep-r là «%m [%p] » (không có application_name) nên không lọc được câu INSERT theo application_name; dùng pg_stat_activity thay thế |
| TC-P212-03 | P | users @edupilot.local theo vai (stack đã dùng chung nên lớn hơn seed gốc): ADMIN=1 STUDENT=74 TA=3 TEACHER=3 |
| TC-P212-04 | P | lớp 2 join_require_approval=t; SV theo lớp/trạng thái: ; sv.moi enrollments=1 |
| TC-P212-05 | P | 7 tài khoản mẫu đăng nhập: [('admin', 200), ('teacher', 200), ('ta', 200), ('sv.gioi', 200), ('sv.kha', 200), ('sv.nguyco', 200), ('sv.moi', 200)] |
| TC-P212-06 | P | cùng build: playwright account.spec.ts -g 'seed accounts nav' → 1 passed rc=0 (nav Sinh viên 7, TA 12, Giảng viên 15, Admin 6, D 1 mục); đối chiếu thật trên ep-r ở spec ui-s4 (xem dòng sau nếu có) |
| TC-P212-07 | P | mật khẩu mặc định không cứng trong seed.mjs (đọc từ env) |
| TC-P212-08 | P | GV seed có 16 thông báo COURSE_ASSIGNED; unread_count=13 |
| TC-P212-09 | P | 4 tuần, thứ 2+4, loại trừ 1 ngày: 201 created 7 (session_no 1,2,3,4,5,6,7); ngày loại trừ có buổi: 0; chạy lại: created 0, tổng 7, trùng starts_at 0; >60 buổi → 422; TA → 201; SV → 403; GV lớp khác → 403; ADMIN → 403 (ghi hành vi: không) |
| TC-P212-10 | P | buổi học lớp 1=15, lớp 2=6 (seed gốc 15/6) |
| TC-P212-11 | P | go test ./internal/course -count=1 -json -v (HEAD 7c9f8b56): 5/5 test pass; log docs/regression/evidence-4/logs/smkT/ |
| TC-P212-12 | P | sau seed đủ trên DB trống: đếm (users, enrollments, mail_outbox, notifications, class_sessions, courses) = 60/61/79/240/21/2; chạy lại seed (rc=0, 24 s, in «Đã tạo 0 lượt luyện mới») → 60/61/79/240/21/2; diff = «files are identical» (SAME). Log evidence-4/logs/qc2s/seed12.log |
| TC-P212-13 | P | DB trống qc2i_seed, giới hạn đăng nhập nới như dev.mjs: ngắt bằng timeout 10 s, 30 s, 60 s (rc=124 mỗi lần, dừng ở bước 3–6) rồi chạy đủ: tiếp tục từ chỗ dở, rc=0, «Seed xong trong 752 s». Số liệu AC2: ADMIN 1, TEACHER 1, TA 1, STUDENT 57; lớp 761987 ACTIVE 30 + PENDING 1; lớp 761988 ACTIVE 24 + PENDING 3; 3 sinh viên học cả hai lớp; mã AN7K2MQ / BX4P9TW; buổi học 15 / 6. mail_outbox DEAD = 1: đó là mẫu cố ý của màn quan sát (to_addr khong-ton-tai@example.invalid, template seed_observability_failed, dedupe_key seed-obs-dead-mail), không phải do ngắt giữa chừng (không có DEAD nào sau ba lần ngắt). Log seed13.log. Ghi nhận: hai lần chạy đầu thất bại do hạ tầng QC (PGBOUNCER_URL, mạng judge, bucket) — đã sửa script qc2s-stack.sh, không phải lỗi sản phẩm |
| TC-P212-14 | F | seed --if-empty trên DB trống: 20 bước, «Seed xong trong 752 s» (AC7 đòi ≤ 3 phút, nhãn [1/9]); các phần khác của AC7 đạt (lần hai «Đã có dữ liệu, không seed» do SmkT2 đo trên DB riêng; SEED_ON_EMPTY_DB khác true thì không chạy và seed lỗi chỉ cảnh báo theo scripts/dev.mjs 80–86). DEF-Q2-S4-01 (Thấp, lệch spec); log evidence-4/logs/qc2s/seed13.log |
| TC-P212-15 | P | APP_ENV=production → rc=1: Seed bị chặn: APP_ENV=production. Seed chỉ chạy ở môi trường dev / test. |
| TC-P212-16 | P | seed.mjs không chứa secret/mật khẩu cứng |
| TC-P212-17 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P212-18 | LT | LỖI THỜI: seed sprint 11 (observability) cố ý thêm 1 thư DEAD `seed_observability_failed`; còn lại SENT |
| TC-P212-19 | P | grep -nE "RATE_LIMIT_IP_PER_MIN/AUTH_(LOGIN/REGISTER/TOKEN)" docker-compose.test.yml / wc -l = 0; docker-compose.local.yml có các biến này (dòng 196–202) và scripts/dev.mjs devLimits nới (5000/1000/1000/1000). Seed trên giới hạn MẶC ĐỊNH (AUTH_LOGIN_IP_PER_MIN=10): dừng ở bước 5–6, đăng nhập bị 429 (17 lần 429 trong log gateway, evidence-4/logs/qc2s/seed-default-limits.log); với cấu hình dev seed xong rc=0 → «chỉ chạy trên cấu hình dev» đúng như TC cho phép ghi rõ |
| TC-P212-20 | P | ep_demo_*: 0 |
| TC-P212-21 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P212-22 | P | seed.mjs --verbose (636 dòng, rc=0): đếm mật khẩu seed = 0; token=<43 ký tự> = 0; chuỗi ≥ 40 ký tự = 0; liên kết /invite/ hoặc /verify-email = 0 (dòng log che cursor và token bằng ***); git status --short / grep -E ".env" / wc -l = 0. Log evidence-4/logs/qc2s/seed22-verbose.log |
| TC-P212-23 | P | đọc scripts/seed.mjs: token Admin (ctx.tokens.admin) chỉ dùng cho /admin/users, /admin/users/{id}/resend-invite, /admin/courses, /admin/courses/{id}/assign, /courses/{c1}/integrations/{imap/teams} (cấu hình tích hợp theo thiết kế ADMIN) và, khi SEED_DECISION_DEMO=1, /admin/llm/decision + /courses/{c1}/decision-gates (cổng quyết định, ADMIN); việc của GV (import, setup/dismiss, sessions/generate, duyệt) dùng ctx.tokens.teacher, của sinh viên (join) dùng token từng người; không dùng token Admin làm việc của GV/SV. Ghi chú: đối chiếu theo mã nguồn vì gateway ghi log truy cập không có sub/vai; Caddy của ep-r không nằm trên đường qc2s |
| TC-P212-24 | K(b) | (b) cần người thật chấm / kịch bản tay có ảnh |
| TC-P212-25 | K(b) | (b) cần người thật chấm / kịch bản tay có ảnh |
| TC-P212-26 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P212-27 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P209-31-admin | P | ADMIN approve không cờ → 422, có cờ → 200 ACTIVE (ghi hành vi theo SRS) |
| TC-P209-42a | P | ô mã 375 px: gõ «an7k2mq» → «AN7K2MQ» (tự viết hoa); gõ «0O1IL» → «» (không nhận); « a n 7 k » → «AN7K»; nút chính Xem lớp (≤ 1 primary: 1; khoá khi chưa đủ 7 ký tự: true); xem trước «Lớp QC2S4 QSUO28A · QSUO28A · TS. Lê Thu Hà · HK1 2026–2027 Tham gia lớp»; nút «Tham gia lớp» 1; tràn ngang 0; sau tham gia: «Tham gia lớp Mã tham gia 7 ký tự, không có 0, O, 1, I, L. Lớp QC2S4 QSUO28A · QS» |
| TC-P201-25-b | P | {'password_changed': ('Mật khẩu EduPilot của bạn đã được đổi', False, True, True, True, False)} |
| TC-P201-25-lock | P | account_locked sau 10 lần sai: tiêu đề 'Tài khoản EduPilot bị khoá tạm thời'; HTML có <script> thô=False, &lt;script&gt;=True, Text nguyên văn=True [stack auth riêng qc2i-*: DB qc2i_auth trên ep-r-postgres; script evidence-4/scripts/qc2i-r4apia-*.py] |

## Trượt: bước tái hiện
Xem `defects-4.md`.

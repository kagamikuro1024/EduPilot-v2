# QC test case — US-NFR-03 Bộ sinh dữ liệu T1 (1.000 SV, 20 lớp, một học kỳ)
Nguồn: US-NFR-03 + SRS FR-3, 5, §5, §9, `SYSTEM_DESIGN` §1.2. Ước lượng đích: 1.000 SV · 20 lớp (≈ 50 SV/lớp) · 5 TA · 3 GV · `chat_messages` ≈ 90.000 · `study_time_daily` ≈ 525.000 · bài nộp ≈ 4 bài/SV ≈ 4.000 · sai số ≤ 10%. DB riêng `edupilot_t1`. **Chạy nặng + lâu**: pha 2, một việc một lúc. Bảng «chính của §1.2» và cách đếm: Q-N-6.

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-N03-01 | AC1 | Stack `ep-qc` trống (Postgres lên; chưa DB `edupilot_t1`) | `time node scripts/seed-t1.mjs` (mật khẩu từ `SEED_DEFAULT_PASSWORD`) | Mã 0; thời gian ghi lại; DB `edupilot_t1` tạo; `\l` không đổi DB `edupilot` | RUN |
| TC-N03-02 | AC1 | TC-N03-01 | `psql -d edupilot_t1`: đếm `users` theo vai; `courses`; `enrollments` mỗi lớp; `chat_messages`; `study_time_daily`; `submissions`; `attendance`; điểm thành phần | SV = 1.000; lớp = 20 (mỗi lớp 45–55 SV); TA = 5; GV = 3; `chat_messages` ∈ [81.000, 99.000]; `study_time_daily` ∈ [472.500, 577.500]; bài nộp ∈ [3.600, 4.400] (±10%); có điểm danh, điểm thành phần, chat theo tuần | RUN |
| TC-N03-03 | AC1 | – | `node scripts/check-t1-seed.mjs; echo $?` | Mã 0 (đếm dòng ≈ bảng §1.2, tất định, FK) | RUN |
| TC-N03-04 | AC2 | `SEED_RNG=123` | Seed T1 vào DB A rồi `--reset` + seed lần hai (hoặc hai DB trống); so `SELECT relname, n_live_tup` / `count(*)` **mọi bảng** hai lần | Số dòng từng bảng bằng nhau 100% (tất định); đổi `SEED_RNG=456` → vẫn đúng quy mô ±10% (có thể khác chi tiết) | RUN |
| TC-N03-05 | AC3 | T1 xong | SQL: với mọi bảng con có `course_id`, join cha: `count(*)` dòng có `course_id` khác cha = 0; dòng thuộc `course_id` không có trong `courses` = 0 (`LEFT JOIN … IS NULL`); danh sách bảng kiểm lấy từ `information_schema` (mọi FK phức hợp `(course_id,…)`) | Toàn 0 | DB |
| TC-N03-06 | AC3 | – | `SELECT conname, convalidated FROM pg_constraint WHERE contype='f' AND NOT convalidated`; `SHOW session_replication_role`; `SELECT tgname FROM pg_trigger WHERE tgenabled='D'` | 0 FK chưa validate; `session_replication_role = origin`; không trigger bị tắt (FK không bị tắt trong lúc nạp) | DB |
| TC-N03-07 | AC3 | – | Đăng nhập 5 SV ngẫu nhiên, 1 TA, 1 GV bằng `SEED_DEFAULT_PASSWORD`; sai mật khẩu; kiểm hash trong `users` (không bản rõ): `grep -c "$SEED_DEFAULT_PASSWORD"` trong dump DB | Đăng nhập được; sai → `401`; không có mật khẩu rõ trong DB; `grep -rn "<chuỗi mật khẩu mẫu>" scripts/seed-t1.mjs` 0 (mật khẩu không cố định trong mã) | RUN + SCAN |
| TC-N03-08 | AC4 | T1 xong | `ANALYZE`; gọi 6 trang chính (Hôm nay, danh sách thread, sổ điểm, hồ sơ 360, thư viện, điểm danh; Q-N-19) bằng STUDENT / TEACHER ở 3 lớp (Q-N-19) | 0 `5xx` trên mọi lời gọi (ghi số lời gọi) | RUN |
| TC-N03-09 | AC4 | – | Chọn ≥ 20 cặp ngẫu nhiên (SV lớp X, tài nguyên lớp Y ≠ X): `GET /courses/{Y}/…` (hôm nay, bài nộp, điểm, thread) bởi SV lớp X | Mọi cặp `403` hoặc `404`; 0 `200` | RUN |
| TC-N03-10 | AC5 | T1 đang chạy | Trong lúc seed T1: `docker compose -p ep-qc stop postgres` | Seed thoát mã ≠ 0 và **in tên bảng đang ghi**; không treo; không đụng DB `edupilot` | RUN |
| TC-N03-11 | AC5 | Sau TC-N03-10, Postgres bật lại | `node scripts/seed-t1.mjs --reset` | Xoá sạch `edupilot_t1` **chỉ** DB này (đếm dòng `users` của DB `edupilot` trước/sau bằng nhau) rồi chạy xong mã 0; đạt quy mô TC-N03-02 | RUN |
| TC-N03-12 | AC5 | Đĩa gần đầy (nếu dựng được trên VM: tạo tệp đệm lấp đầy volume Postgres) | Seed T1 | Mã ≠ 0 và in bảng đang ghi (không dựng được → «KHÔNG KIỂM ĐƯỢC», ghi lý do, không FAIL) | RUN |
| TC-N03-13 | AC6 | – | `DATABASE_URL=postgres://…/edupilot node scripts/seed-t1.mjs --reset; echo $?` rồi các biến thể: `…/EDUPILOT`, `…/edupilot?sslmode=disable`, `…/edupilot/`, `postgres://u@host/edupilot#x`, `postgres://…/postgres`, `…/edupilot_t10` | `edupilot` và biến thể trỏ cùng DB: **từ chối**, mã ≠ 0, DB `edupilot` còn nguyên (đếm dòng trước/sau); `…/postgres` từ chối (không phải `edupilot_t1`); chỉ chấp nhận đúng `edupilot_t1` | RUN |
| TC-N03-14 | AC6 | – | `--reset` không `--yes` (nếu có cờ xác nhận) ; `--reset` khi DB `edupilot_t1` không tồn tại | Hành vi an toàn: không xoá DB khác; DB chưa có → tạo mới hoặc báo rõ | RUN |
| TC-N03-15 | AC7 | T1 xong | Báo cáo (file nguồn: Q-N-9) ghi kích thước DB (`SELECT pg_database_size('edupilot_t1')`) và RAM Postgres **nghỉ ghi** (`docker stats --no-stream` sau 2 phút); QC đo lại | Số báo cáo khớp số QC đo (kích thước DB ±5%; RAM ±30%, Q-N-13); không có `--scale` mặc định | FILE + RUN |
| TC-N03-16 | AC7 | Báo cáo T1 | `jq` `meta.scale`/cấu hình; `docker inspect <ct> --format '{{.State.OOMKilled}}'`, `docker events` ghi trong báo cáo | **Không thu nhỏ trước**: nếu báo cáo dùng `--scale 0.5` thì phải có bằng chứng OOM-kill thật (ghi sự kiện) và quy mô thực ghi đúng, không tuyên bố «đạt T1» cho phần đã hạ; không có bằng chứng → FAIL | FILE |
| TC-N03-17 | Quy ước | – | `grep -rEn` tên / MSSV / email thật trong `scripts/seed-t1.mjs` và dữ liệu sinh; log | Toàn dữ liệu mô phỏng (D44); không PII thật; `canary-scan` mã 0 | SCAN |

## Phân quyền
TC-N03-09 (≥ 20 cặp chéo lớp).

## Lịch sử sửa TC
(trống)

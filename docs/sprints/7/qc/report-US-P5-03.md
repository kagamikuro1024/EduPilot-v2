# QC report — US-P5-03 (thời gian học: heartbeat, rollup, hộp thoại minh bạch, công tắc đo, luật cứng `internal/grade`)
Ngày 2026-10-15. Commit kiểm: `sprint/7-p4-p5` @ `1dbda5eb` (code dev `1c6a01e`, CI xanh). Bộ TC: `tc-US-P5-03.md` (không sửa chữ TC). Handoff: `dev-US-P5-03.md`. Stack `edupilot100` dựng lại (colima khởi động lại; `docling-serve` không chạy nên seed dừng ở bước 11 — không ảnh hưởng story này). Số migration thực là `00015_observation` (PM đổi theo #25; TC viết `00014`).

## Kết luận: **FAIL** — 3 lỗi (1 trung bình, 2 thấp) + 1 TC không kiểm được
Phần lõi đạt: heartbeat chỉ Redis, hai tab, trần ngày, rollup tuyệt đối/chịu crash/mất khoá/nửa đêm, hộp thoại + ack, công tắc đo, khoá cờ lớp + đua 50 lượt, luật cứng `grade` (tự thử đỏ hai đường), k6 p95 7,55 ms. FAIL vì ba điểm lệch AC/TC dưới đây.

## 1. Lỗi
| # | Mức | TC / AC | Mô tả + tái hiện | Nghi ngờ |
| --- | --- | --- | --- | --- |
| BUG-1 | trung bình | TC-26 · AC9 | AC9: "sinh viên không `ACTIVE` ở lớp → `403`". Thực tế `404 NOT_FOUND`. Tái hiện: `GET /courses/$C1/students/<uid sinh viên PENDING/REMOVED>/study-time -H "$TCH"` → `404 {"code":"NOT_FOUND"}` (uid `01a13fa9-392b-71d0-ba98-7284756523f2`, lớp 761987). Handoff mục HTTP ghi cố ý `404` — lệch chữ AC9. | `internal/httpapi/activityhttp` (đọc staff) |
| BUG-2 | thấp | TC-29 · AC10 | Redis dừng → `204` đúng, log WARN `activity_heartbeat_dropped_total` có, không PII; **nhưng** không có cách "thấy tăng" trên `/metrics` (gateway không có `/metrics`; `$GW/metrics` trả HTML của frontend). Bộ đếm chỉ là `Service.Dropped atomic.Int64` (`internal/activity/activity.go:70`). Thêm: lúc Redis chết mỗi nhịp mất **10,4 s** (`duration_ms=10465`, lỗi `lookup redis … no such host`) trước khi trả `204` — nhịp treo, hook không biết. | `activity.go:69-70, 228-230` |
| BUG-3 | thấp | TC-01 | `docs/PROGRESS.md` bảng migration dừng ở `00013 crm`; không có dòng `00015 observation` (P5-01 từng có `00013`). | `docs/PROGRESS.md` ~dòng 78 |
| KKĐ-1 | — (tính FAIL theo luật) | TC-35 (vế `student_risk`) · AC11 | "tín hiệu `STUDY` trong `student_risk` là `null` khi đã tắt" và hồ sơ 360 "Đã tắt đo": `student_risk` rỗng (0 dòng), chưa có job tính rủi ro (`internal/crm` không có `risk`). Handoff nói chuyển sang US-P5-04 (`TestRiskStudyNullWhenOptedOut`, `TestRiskStudyPerCourseConfig`). Vế công tắc `/settings` (lạc quan, `409` hoàn lại + dòng "Có người vừa sửa mục này…", không toast, 375 px không tràn) đạt. | PM quyết: chuyển vế này khỏi AC11 sang P5-04 (cần proposal) hoặc giữ FAIL |

## 2. Kết quả từng TC
| TC | KQ | Bằng chứng ngắn |
| --- | --- | --- |
| 01 | **FAIL** (BUG-3) | `ALTER TABLE` = 2; hai cột mặc định `true`; goose up/down/up: `TestMigration00015TwoAlters`, `TestStudent360StudySecondsFrom00015` (DownTo 14 / UpTo 15) xanh; thiếu dòng PROGRESS |
| 02 | PASS | `area='X'`, `-1`, `86401` → `23514`; trùng → `23505`; UNIQUE `(course_id,student_id,date,area)`; `insight_reports` khớp SRS 5.8 |
| 03 | PASS | nhịp: Redis 0 → 30, `204`; log Postgres 5 s quanh nhịp: 0 câu INSERT/UPDATE/DELETE (chỉ SELECT outbox của worker) |
| 04 | PASS | k6 10 req/s × 60 s: 600 req, p95 **7,55 ms**, 0 lỗi (12 sinh viên đã ack) |
| 05 | PASS | cùng khung hai `tab_id` → +30 tổng; hai khung kế → +30 mỗi khung (30→60→90); TTL khoá 691200 s (8 ngày) |
| 06 | PASS theo Q-QC-P5-05 (chờ PM xác nhận) | đồng hồ giả không đẩy giờ máy chủ; thay bằng TC-05 (máy chủ) + TC-11 (hook) |
| 07 | PASS | ẩn 4 phút giả lập: 0 nhịp; hiện lại + thao tác: 1 nhịp trong 1 s |
| 08 | PASS (+ ghi chú N1) | thao tác t0 rồi dừng 5 phút: nhịp cuối ≈ 103 s, sau mốc 2 phút không còn nhịp; thao tác lại → tiếp ngay |
| 09 | PASS | `/chat → /library` chỉ còn `LIBRARY`; `/library → /` 0 nhịp; đổi khu vực ghi đúng |
| 10 | PASS | offline: nhịp thất bại không xếp hàng, online lại tiếp bình thường; `403` → 1 nhịp rồi ngừng, giao diện im lặng |
| 11 | PASS | 10 phút giả lập, thao tác 3 phút đầu: **10 nhịp = 300 s** |
| 12 | PASS | 10 phút, ẩn 4 phút (0 nhịp trong 4 phút đó): 14 nhịp = 420 s (hai đoạn hiện mỗi đoạn 7 nhịp tính cả hai mút; tổng trong ±30 s mỗi đoạn) |
| 13 | PASS | `study_time_daily.area` chỉ 5 giá trị; Redis chỉ `{CHAT,THREADS,PRACTICE,LIBRARY,CALENDAR}`; `/`, `/me`, `/exams` 0 nhịp; thân nhịp chỉ `course_id,area,tab_id` |
| 14 | PASS | rollup ghi tuyệt đối: DB 120 → 570 (= Redis); Redis đặt thấp 100 → DB giữ 570; chạy lại không đổi |
| 15 | PASS | `kill -s KILL worker` + `up` + chờ 150 s: tổng 810 s / 5 hàng không đổi, không âm (không canh trúng đúng giữa lô; có `TestRollupCrashSafe`) |
| 16 | PASS | `DEL` khoá → DB giữ 570 |
| 17 | PASS | khoá `…:20261014:LIBRARY=90` và `…:20261015:LIBRARY=60` → hai hàng đúng ngày ICT; TTL khoá do nhịp tạo 8 ngày |
| 18 | PASS | `Dialog` một lần đúng nội dung SRS 7.5 (Chat, Threads, Luyện đề, Thư viện, Lịch; không ghi nội dung; giảng viên xem được; không tính điểm; "Bạn có thể tắt ở Cài đặt."); nút `Đã hiểu` |
| 19 | PASS (ghi chú) | ack đặt `tracking_notice_ack_at`; gọi lần hai idempotent nhưng trả `204` (TC viết `200`; SRS/US chỉ nói "idempotent" — không đổi hành vi); không hiện lại sau tải lại; nhịp ngay sau ack (token cũ) đo +30 |
| 20 | PASS | trước ack: `204`, không khoá Redis; hook 35 s không gửi nhịp |
| 21 | PASS | TEACHER/TA/ADMIN: `204`, không khoá; không hộp thoại (e2e) |
| 22 | PASS | `go list -deps ./internal/grade/...` không có `activity`/`crm`; hai bài kiểm xanh |
| 23 | PASS | gói giả `internal/grade/zz` import `httpapi/crmhttp` (`today` không import `crm` — như handoff nêu) → `FAIL … phụ thuộc (bắc cầu) …/internal/crm`; đã xoá |
| 24 | PASS | gói giả gọi `q.CrmActiveStudentIDs` → `FAIL … gọi CrmActiveStudentIDs` (cả hai bài); đã xoá, cây sạch |
| 25 | PASS | 0 tệp |
| 26 | **FAIL** (BUG-1) | `200 {weeks,this_week_minutes,tracking}` / bản staff `tracked`; STUDENT & ADMIN → `403`; TA `200`; `/me/study-time` bằng TEACHER/ADMIN `403`; sinh viên không ACTIVE → `404` ≠ `403` |
| 27 | PASS | `weeks=0`, `21`, `abc` → `422`; bỏ trống `200` (8 tuần) |
| 28 | PASS | sinh viên không dữ liệu: `{"weeks":[],"this_week_minutes":0,"tracking":{…}}`; `TestAllCourseRoutesGuarded` ok (cả `./internal/httpapi/...`) |
| 29 | **FAIL** (BUG-2) | `204` + WARN không PII; không `5xx`; không có `/metrics` để thấy bộ đếm tăng; trễ 10,4 s |
| 30 | PASS | lớp lạ / PENDING → `403`; `area:"X"` → `422`; thân thiếu / `course_id:123` / `tab_id` không UUID → `422` (lưu ý: `tab_id` phải là UUID — TC viết `t1`) |
| 31 | PASS | tổng 43.170: nhịp 1 → 43.200; nhịp 2 bỏ (`204`); lớp 2 khu vực khác khung kế → bỏ |
| 32 | PASS | 6 nhịp liên tiếp `tab_id` khác nhau: cả 6 `204`, chỉ +30 (khung 30 s); `ep:hb:rate`=6; không `429` (Q-QC-P5-06: không đặt được "nhịp thứ 5 trong 60 s" bằng tay) |
| 33 | PASS | `HLEN flushed` = `SCARD dirty` = 3; dọn > 1 ngày: `TestRollupPrunesFlushed` |
| 34 | PASS | tắt: nhịp `204`, khoá Redis không đổi, `GET me/study-time` vẫn đọc được `tracking.enabled=false`; bật lại đo tiếp; sai `version` → `409` |
| 35 | **KKĐ-1** | vế UI đạt; vế `student_risk`/hồ sơ 360 chưa có (US-P5-04) |
| 36 | PASS | `PUT track_study_time=false` `200`; lớp khoá vẫn đo (+30, +60); `/settings` công tắc bật + disabled, `title` "Lớp này luôn ghi thời gian học" ở chính công tắc, không dòng mô tả |
| 37 | PASS | `POST` và `PUT /admin/courses` có `allow_study_time_opt_out` → `422 unknown`; `AdminCourses.tsx` không có trường; lớp mới mặc định `true` |
| 38 | PASS | teacher `200` + `audit_log` `{version 1, true}` → `{version 2, false}`; TA/STUDENT/ADMIN `403` |
| 39 | PASS | lớp có sinh viên ACTIVE và có sinh viên PENDING: đổi / giữ nguyên giá trị → `409 COURSE_HAS_STUDENTS`; sai version → `409 VERSION_CONFLICT`; cờ không đổi |
| 40 | PASS | 50 lớp mới, mỗi lớp join + PUT song song: 25 (join 200 / PUT 200, cờ `false`, sinh viên vào **sau** `audit_log`) + 25 (join 200 / PUT 409, cờ `true`); 0 lượt sinh viên vào trước mà cờ bị đổi |
| 41 | PASS | teacher: công tắc disabled + `title` "Lớp đã có sinh viên"; TA: mọi công tắc disabled (chỉ đọc); `GET /courses/{id}` (Member) có cờ; trạng thái chưa-có-sinh-viên đổi được: e2e |
| 42 | PASS | ADMIN `GET /me/settings` chỉ của mình; `PUT` có `user_id` → `422 unknown` |
| 43 | PASS | sv.kha `track=false`, X=761987, Y=lớp khoá: X `204` không đo, không chiếm khung (Y cùng khung +30, khung kế +60); `study_time_daily`: Y có hàng, X giữ giá trị cũ (30, đo trước khi tắt); `tracking` X `{false,false}`, Y `{true,true}`; `/settings` hai dòng (X đổi được, Y bật khoá) |
| 44 | PASS | `activity.spec.ts` desktop **16 passed**; `ui-antipatterns.sh` rc=0 |
| E1 | PASS | sinh viên `REMOVED` / `PENDING` → `403` |
| E2 | PASS | xoá hết khoá `ep:study:*` giữa ngày + rollup: DB giữ nguyên, `min(active_seconds)=30` |
| P1 | PASS | TA của lớp 1 gọi `…/students/<uid>/study-time` ở lớp 2 → `403`; sinh viên lớp 1 đọc `/courses/{lớp 2}/me/study-time` → `403` |

## 3. Lệnh của handoff
| Lệnh | KQ |
| --- | --- |
| `go test -race -tags testroutes ./internal/activity -v` | ok 72,7 s (44 test, gồm 3 bài luật cứng) |
| `go test -race -tags testroutes ./internal/store ./db/... ./internal/contract/... ./internal/course ./internal/user ./internal/httpapi/...` | 455 passed / 24 gói |
| `grep -c '^ALTER TABLE' 00015_observation.sql` | 2 |
| `sqlc generate && sqlc diff` | rc=0 |
| `pnpm build:gate` + `playwright test activity.spec.ts --project=desktop` | 16 passed |
| `bash scripts/ui-antipatterns.sh` | rc=0 (chạy từ gốc repo) |
| `k6 run benchmarks/load/heartbeat.js` (nợ 4 của handoff) | p95 7,55 ms, 0 lỗi — đạt |

## 4. Ghi nhận (không đổi kết luận)
- **N1 (hook):** sau thao tác đơn lẻ rồi để yên, nhịp đo bằng đồng hồ giả cách nhau 13 → 43 → 58 → 73 → 88 → 103 s (chu kỳ ≈ 15 s từ nhịp thứ ba), trong khi thao tác liên tục đều 30 s. Máy chủ khử trùng theo khung nên số giây không đổi, nhưng tải nhịp gấp đôi và sát trần 4 nhịp/phút. `frontend/src/shared/data/useActivityHeartbeat.ts` có một `setInterval`; nguyên nhân chưa xác định (không tái hiện bằng đọc mã). Dev nên xem.
- `scripts/ui-antipatterns.sh` bị sửa trong `f45454c7` (thêm `TrackingNotice` vào danh sách được dùng `<Dialog`) — không có số proposal; Dialog này do SRS 7.5 quy định, tiền lệ tương tự (`PIIChannelDialog`…). PM xác nhận.
- Test cũ chỉ sửa đúng phần `#27` (`TestCourseSchema`, `TestSettingsDefaultsLazyCreate`) + `migrations_test.go` (14 → 15); `scenarios_test.go`/`contract_test.go` thêm thao tác (#11). Không thấy `fetch` trần, thư viện lạ, secret.
- Spec AC8 ví dụ (1) dùng `today` làm gói trung gian nhưng `today` không import `crm` — nợ BA (handoff đã nêu).
- Seed: lớp 2 (`761988`) có `allow_study_time_opt_out=true`; AC10 của US-S7-SEED yêu cầu `false` — thuộc story seed, không kiểm ở đây.

## 5. Dữ liệu thử để lại trên stack `edupilot100`
50 lớp `QC r*` + `QC a/b/strict` (giảng viên `teacher@`), sv04–sv53 ghi danh; `ep:*` Redis và `study_time_daily` có dữ liệu QC. `log_statement` đã trả về mặc định.

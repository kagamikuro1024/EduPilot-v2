# QC cổng P5 — CRM, hồ sơ 360, thời gian học (M5, M6, M13-C) + seed sprint 7
Nguồn: `docs/phases/P5.md` (cổng nghiệm thu, "Bạn tự kiểm"), `docs/specs/FEAT-crm-360/` (US v1.1 + SRS), `docs/sprints/7/prompts/qc-tc-p5.md`, `.claude/skills/gate/SKILL.md`. Chạy ở pha 2 sau khi có handoff đủ 6 story + SEED; mỗi dòng ghi PASS/FAIL + 10 dòng đầu ra. Kết luận cổng: **PASS chỉ khi mọi dòng PASS** và mọi `tc-US-P5-01…06.md`, `tc-US-S7-SEED.md` không FAIL. AC13 của US-P5-02 chỉ được cắt khi QC FAIL sau **2 vòng** (PM 2026-10-13, Q20): report ghi vòng nào.

## A. Lệnh cổng của phase (`P5.md`)
| ID | Lệnh | Kết quả mong đợi |
| --- | --- | --- |
| G5-01 | `cd backend-go && go test -race ./internal/crm/... ./internal/activity/...` | `ok`, không `DATA RACE`; gồm ca "idle 2 phút ngừng đếm; tab ẩn không đếm; hai tab không đếm đôi; STUDENT đọc observation → 403" |
| G5-02 | `cd backend-go && go test ./internal/agent -run 'TestToolMyAttendance\|TestToolMyParticipation' -v` | `ok`; đủ test; `TestPersonalToolsHaveNoIdentityParam` vẫn xanh |
| G5-03 | `pnpm -C frontend exec playwright test attendance.spec.ts` | xanh toàn bộ (có bước ngắt mạng); ghi số test, flaky (≥ 3 flaky → FAIL) |
| G5-04 | `bash scripts/gate-p5.sh; echo rc=$?` | `rc=0`; in một dòng PASS/FAIL mỗi lệnh |
| G5-05 | `go vet ./... && golangci-lint run && go test -race ./...` (toàn bộ, không `-run`) | xanh; số test cũ không giảm so với mốc (`1518f83`: 1436 top-level Go, 822 Playwright; QC đo lại ở đầu pha 2) |
| G5-06 | `go test ./internal/contract/...`; `sqlc diff`; `goose up/down/up` từ DB trống (`00013`, `00014`) | xanh; hợp đồng chỉ cộng |
| G5-07 | `pnpm -C frontend lint && pnpm -C frontend build`; `bash scripts/ui-antipatterns.sh` | xanh; `rc=0` |
| G5-08 | `git diff 1518f83.. --stat -- '*_test.go' 'frontend/e2e/*'` | chỉ **thêm** test mới; test cũ chỉ bị sửa nếu có số proposal PM đã chấp nhận (#8, #10 của P4, hoặc proposal mới cho P5); mọi sửa khác → FAIL nghiêm trọng |
| G5-09 | `k6 run benchmarks/load/heartbeat.js` | p95 < 20 ms ở 10 req/s, 0 lỗi 5xx |
| G5-10 | `cd backend-go && go test ./internal/activity -run 'TestGradeNeverImportsActivity\|TestGradeQueriesNeverReadStudyTime' -v` | `ok`; tự thử đỏ hai đường (nhánh tạm, TC-P5-03-23, 24) → `FAIL` rồi xoá nhánh |

## B. Kiểm tay "Bạn tự kiểm" (`P5.md`) — `playwright-cli`
| ID | Việc | Kết quả mong đợi |
| --- | --- | --- |
| G5-11 | **Bấm giờ** điểm danh 30 sinh viên bằng giả lập 375 px **và** bàn phím (điện thoại thật là phần chủ dự án tự kiểm, ghi rõ giới hạn) | ≤ 60 s mỗi cách; 0 hộp thoại; một `PUT` ghi cả lưới (TC-P5-02-19, 29) |
| G5-12 | **Chế độ máy bay giữa lúc điểm danh:** đánh vắng 3 người, tắt chế độ máy bay | dữ liệu tự lên, không mất (TC-P5-02-31…36); nếu AC13 bị cắt (PM xác nhận): báo rõ "chưa lưu" và còn nguyên khi có mạng (AC13b) |
| G5-13 | **Sinh viên C hiện ở "Cần chú ý" với đúng lý do** | TC-P5-04-17, TC-S7-SEED-10, 11: câu "Cần chú ý: vắng … buổi trong 2 tuần gần nhất; học … khoảng … phút mỗi tuần." |
| G5-14 | **Đăng nhập sinh viên:** không thấy ghi chú / nhãn; thấy thời gian học của mình | TC-P5-06-06, 07, 13, 08 |
| G5-15 | **Q1:** lớp không cho tắt → công tắc `/settings` khoá; lớp cho tắt → tắt công tắc rồi mở `/chat` 1 phút: `study_time_daily` không tăng | TC-P5-03-34, 36; TC-P5-06-22 |

## C. Kiểm chéo bắt buộc (QC.md pha 2 mục 3) — chạy lại bằng TC đã viết
| ID | Mục | TC | Kết quả mong đợi |
| --- | --- | --- | --- |
| G5-16 | **Dữ liệu nhạy cảm:** STUDENT (kể cả qua API trực tiếp, qua `/me`, qua câu hỏi AI) không bao giờ thấy ghi chú quan sát, nhãn / câu nhận định rủi ro, `note` trên dòng điểm danh của mình → 403 / không có trường | TC-P5-04-29, 30, 31; TC-P5-05-08, 28; TC-P5-06-01, 02, 06, 07; TC-P5-02-41 | 0 canary (`CANARY-OBS-4F`, `CANARY-NOTE-9`) ở mọi phản hồi / payload LLM của sinh viên; không khoá `risk*`, `observation*`, `at_risk`, `note` |
| G5-17 | **ADMIN và TA/TEACHER lớp khác không mở được hồ sơ** | TC-P5-05-08; TC-P5-04-P1; TC-S7-SEED-20 | `403` (ADMIN, staff lớp khác), `404` (sinh viên lớp khác) |
| G5-18 | **Audit:** mỗi lần mở hồ sơ (đếm request, cả khi quay lại trang) = 1 dòng `audit_log`; đọc ghi chú cũng ghi; fail-closed | TC-P5-05-02, 03, 04, 05, 31 | n+3 dòng sau 3 lần mở; `view_observations` có; `503` khi audit lỗi |
| G5-19 | **Điểm danh:** 30 SV ≤ 60 s (375 px + phím 1–4); hai người cùng lưu → 409 rõ ràng; ngắt mạng đánh vắng 3 người rồi nối lại → tự đồng bộ; đăng xuất khi còn thay đổi chờ thì được hỏi; sửa sau buổi ghi audit; `get_my_attendance`/`get_my_participation` đúng số của chính mình, hỏi hộ người khác bị từ chối | TC-P5-02-19, 29, 08, 10, 31–35, 14, 38–41 | đúng |
| G5-20 | **Lịch buổi học:** sinh hàng loạt có xem trước (không cần `Idempotency-Key` khi `dry_run`), ngày loại trừ, xoá buổi chưa điểm danh, buổi đã điểm danh không xoá được, học bù | TC-P5-01-05…22 | đúng |
| G5-21 | **Thời gian học:** idle 2 phút ngừng đếm; tab ẩn không đếm; hai tab không đếm đôi; heartbeat không ghi Postgres; lớp cho tắt / không cho tắt / SV học 2 lớp khác cấu hình; chỉ TEACHER đổi `allow_study_time_opt_out` và chỉ khi lớp chưa có sinh viên (409 `COURSE_HAS_STUDENTS`); không đo `/exams/**`; `internal/grade` không phụ thuộc `activity` / `crm` (kể cả bắc cầu) | TC-P5-03-03, 05–08, 13, 34–43, 22–25 | đúng |
| G5-22 | **At-risk + `/students`:** đúng lý do bằng lời cho sinh viên C; `EXPLAIN` lớp 100 SV không `Seq Scan`; lớp 1.000 SV trang đầu p95 ≤ 300 ms; phân trang con trỏ; "Lớp cần chú ý" ≤ 5 dòng, ẩn khi rỗng | TC-P5-04-17, 08, 09, 02, 23–25 | đúng |
| G5-23 | **Không gamify** ở `/me` (không streak, xếp hạng, vòng tiến độ); 375 px cho `/attendance`, `/me` | TC-P5-06-10, 16; TC-P5-02-16 | `0`; không tràn ngang |
| G5-24 | **Phân quyền ma trận** các route mới (SRS 2) | TC-P5-01-23; TC-P5-02-42, 43; TC-P5-03-26, 38; TC-P5-04-29; TC-P5-05-08; TC-P5-06-04 | đúng từng ô |
| G5-25 | **Idempotency** POST có tác dụng phụ (`generate`, `outreach`, `participation`) | TC-P5-01-12; TC-P5-05-27; TC-P5-02-27 | không tạo đôi |
| G5-26 | **Phân trang / không `Seq Scan` / N+1**: `students`, `me/attendance`, `observations`, `sessions` | TC-P5-04-02, 08; TC-P5-06-05; TC-P5-05-10 | con trỏ, `limit ≤ 100`, không OFFSET |
| G5-27 | **Trạng thái tải / rỗng / lỗi và 375 px** cho màn mới (`/attendance`, `/students`, `/students/[id]`, `/me`, ngăn kéo lịch) | TC-P5-02-44; TC-P5-04-28; TC-P5-05-15; TC-P5-06-18; TC-P5-01-28 | đủ ba trạng thái |
| G5-28 | **Không PII trong log và payload LLM:** `ai-summary` (không tên/MSSV/email/ghi chú), log gateway/worker (`grep` canary) | TC-P5-05-13, 16; `docker compose logs gateway worker \| grep -c CANARY-` | `0` |
| G5-29 | **Migration `00013`, `00014` sạch trên DB trống**; đúng số `ALTER` (1 và 2) | TC-P5-01-01, 02; TC-P5-03-01 | đúng |
| G5-30 | **Mock → thật:** `grep -rn "@/mock" frontend/src/features/{attendance,students,me}` | TC-P5-02-45; TC-P5-05-32; TC-P5-06-21 | `0` mỗi nơi |
| G5-31 | **Luật chữ UI** (`CLAUDE.md`): không câu mô tả thừa dưới tiêu đề / dòng; tooltip chỉ khi cần; không "AI/RAG/PII/risk" ở màn sinh viên | `eval innerText` mọi màn mới + `ui-antipatterns.sh` | 0 vi phạm |
| G5-32 | **Seed:** TC-S7-SEED-01…25 (số liệu A, B, C; 270 dòng; 5 ticket đủ trạng thái; idempotent; chặn production) | `tc-US-S7-SEED.md` | đúng; `seed/expected_crm.csv` khớp |
| G5-33 | **Luồng F7 và F8 đi trọn** (FLOWS) kể cả nhánh lỗi liệt kê (điểm danh mất mạng, hai người cùng điểm danh, hồ sơ audit lỗi) và F14 (nguồn việc `SESSION_UNMARKED`, `STUDENT_ATTENTION`, khối `attention` ở "Hôm nay") | TC-P5-01-29, 30; TC-P5-04-23; TC-P5-02-10, 31 | E2E đi từ đầu đến cuối |
| G5-34 | **Đọc diff** (`git diff 1518f83..`): lan phạm vi, `fetch` trần, thư viện ngoài bảng `ARCHITECTURE.md`, secret, `float64` cho điểm cộng (phải `decimal`), `internal/grade` import `activity`/`crm` | – | không có; nếu có → báo PM |

## D. Số liệu ghi vào report cổng
Số TC mỗi story (PASS/FAIL/KHÔNG KIỂM ĐƯỢC); thời gian điểm danh 30 SV (giả lập 375 px / bàn phím); p95 `PUT` lưới 30 hàng; p95 heartbeat; p95 `students` trang đầu lớp 1.000 SV; hồ sơ lớp 1.000 SV; số test Go / Playwright trước-sau; vòng chấm AC13.

## Câu hỏi (Q-QC)
- **Q-QC-P5-14** G5-11/12 "điện thoại thật" là phần chủ dự án tự kiểm; QC báo cáo bằng giả lập 375 px và nêu giới hạn. PM xác nhận không tính "KHÔNG KIỂM ĐƯỢC" cho phần thiết bị thật.
- **Q-QC-P5-15** G5-08/G5-05: mốc "số test không giảm" lấy lại đầu pha 2 (sau các commit P4 của dev).

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-13: viết lần đầu.

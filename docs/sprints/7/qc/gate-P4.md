# QC cổng P4 — Escalation, thông báo, thư qua mail, kiểm duyệt Threads, an toàn con người
Nguồn: `docs/phases/P4.md` (cổng nghiệm thu, "Bạn tự kiểm"), `docs/specs/FEAT-escalation-notify/` (US v1.1 + SRS), `docs/sprints/7/prompts/qc-tc-p4.md`, `.claude/skills/gate/SKILL.md`. Chạy ở pha 2 sau khi có handoff của cả 7 story; mỗi dòng ghi PASS/FAIL + 10 dòng đầu ra. Kết luận cổng: **PASS chỉ khi mọi dòng PASS** và mọi `tc-US-P4-01…07.md` không FAIL.

## A. Lệnh cổng của phase (`P4.md`)
| ID | Lệnh | Kết quả mong đợi |
| --- | --- | --- |
| G4-01 | `cd backend-go && go test -race ./internal/escalation/... ./internal/notify/... ./internal/mail/...` | `ok` mọi gói, không `DATA RACE` |
| G4-02 | `cd backend-go && go test -tags integration ./internal/integration -run TestEscalationToMail -v` | `ok`; in `bell_latency_p95_ms` ≤ 5000; thư trong Mailpit ≤ 60 s; gửi lại cùng `message_id` không tạo ticket thứ hai |
| G4-03 | `cd backend-go && go test -tags integration ./internal/integration -run TestNotifyAcrossTwoGateways -v` | `ok` (nối bản A, sinh ở bản B, và đảo lại) |
| G4-04 | `cd backend-go && go test -tags integration ./internal/integration -run TestOutboxSurvivesWorkerKill -v` | `ok` (hai điểm giết, mỗi điểm Mailpit đúng 1 thư) |
| G4-05 | `pnpm -C frontend exec playwright test escalation.spec.ts` | xanh toàn bộ; ghi số test, flaky (flaky ≥ 1 cần ghi, ≥ 3 → FAIL) |
| G4-06 | `bash scripts/gate-p4.sh; echo rc=$?` | `rc=0`; gồm bước `--scale gateway=2`, ba test tích hợp và `escalation.spec.ts` |
| G4-07 | `go vet ./... && golangci-lint run && go test -race ./...` (toàn bộ, không `-run`) | xanh; đếm test: số test cũ **không giảm** so với `1518f83` (1436 top-level ở mốc story CI-02 + test mới) |
| G4-08 | `go test ./internal/contract/...`; `sqlc diff`; `goose up/down/up` từ DB trống | xanh; không sửa test hợp đồng (`git diff 1518f83.. -- backend-go/internal/contract` chỉ cộng) |
| G4-09 | `pnpm -C frontend lint && pnpm -C frontend build`; `bash scripts/ui-antipatterns.sh` | xanh; `rc=0` |
| G4-10 | `git diff 1518f83.. --stat -- '*_test.go' 'frontend/e2e/*'` | chỉ **thêm** test mới; test cũ bị sửa chỉ gồm danh sách `proposals.md` #8 (`TestSchemaChat`, `TestSettingsDefaultsLazyCreate`, TC sprint 6 "không có nút Nhờ giảng viên") và #10 (`TestSettingsOwnerOnly`). Mọi test cũ khác bị sửa → FAIL nghiêm trọng. Ghi: dev đã sửa #10 trước khi PM chốt (lỗi quy trình, đã ghi) |

## B. Kiểm tay "Bạn tự kiểm" (`P4.md`) — `playwright-cli` (global), hai ngữ cảnh
| ID | Việc | Kết quả mong đợi |
| --- | --- | --- |
| G4-11 | Hai trình duyệt (sinh viên `sv.gioi@`, giảng viên `teacher@`): sinh viên hỏi câu ngoài tài liệu | giảng viên thấy chuông nổi ≤ 5 s không reload |
| G4-12 | Giảng viên mở `/inbox`, `Nhận`, trả lời | sinh viên thấy trả lời trong `/chat` với nhãn "Giảng viên · {tên}" |
| G4-13 | Mailpit `localhost:8025` | đúng 1 thư "Giảng viên đã trả lời câu hỏi của bạn" ≤ 60 s |
| G4-14 | Đọc thư | không có thông tin của sinh viên khác (canary khác) |

## C. Kiểm chéo bắt buộc (QC.md pha 2 mục 3) — chạy lại bằng TC đã viết
| ID | Mục | TC | Kết quả mong đợi |
| --- | --- | --- | --- |
| G4-15 | **Quyền đọc:** TEACHER / TA lớp khác, ADMIN, STUDENT khác không đọc được ticket; giảng viên chỉ thấy đúng cặp hỏi–đáp đã nhờ | TC-P4-01-28, 29, 33; TC-P4-03-09, 10, 11, 39; TC-P4-07-11, 26 | không rò (canary `CANARY-9K3Z` = 0 ở mọi phản hồi); 403/404 đúng ma trận SRS 2 |
| G4-16 | **Idempotent:** 2 lần bấm / cùng `message_id` → 1 ticket; hai người `Nhận` đồng thời → người sau 409 thấy ai nhận; `answer` lặp `Idempotency-Key` | TC-P4-01-08, 09, 22; TC-P4-03-14, 20 | đúng 1 ticket; đúng 1 `200`; không tin đôi |
| G4-17 | **Outbox + mail:** giết worker giữa chừng → Mailpit đúng 1 thư mang `Message-ID` của dòng; thư không chứa dữ liệu SV khác; mail lỗi 3 lần → "Thư chưa gửi được" ở `/observability` (ADMIN) | TC-P4-04-06, 07, 09, 10, 19, 31 | đúng; 0 chéo; `DEAD` sau 3 lần thử lại, hiện cho ADMIN, không hiện cho TEACHER |
| G4-18 | **Chuông qua 2 bản gateway** (`--scale gateway=2`), ≤ 5 s; `Last-Event-ID` / `resync` | TC-P4-02-07, 08, 09, 10, 11 | nhận ở cả hai chiều; đọc bù đúng, không trùng |
| G4-19 | **SLA** (T2–T6 cả ngày; 24 h nhắc; 72 h lên đầu "Hôm nay" + Admin; hỏi tiếp đặt lại `waiting_since`; cuối tuần không tính) | TC-P4-05-07, 16–26 | khớp bảng SRS 9.3; một lần nhắc; Admin một thông báo mỗi ngày khi không ai hoạt động |
| G4-20 | **Threads kiểm duyệt:** 3 báo cáo → tự ẩn; mỗi người 1 báo cáo / bài; sửa chạy lại PII; bài ẩn không lọt ở tìm kiếm / tương tự / Hôm nay; bài `Loại` ra khỏi RAG | TC-P4-06-02, 10, 11, 23, 41, 52 | đúng |
| G4-21 | **Khủng hoảng:** 20 câu dương (≥ 19), 17 câu âm (0), gồm "từ từ", "từ hai", "kết thúc tất cả các tiến trình con"…; không báo ai; giờ thi vẫn có đường hỗ trợ | TC-P4-07-01, 02, 03, 09, 17 | `crisis=≥19/20`, `crisis=0/17`; canary chỉ ở `chat_messages.content` |
| G4-22 | **Giờ thi:** "Nhờ giảng viên" bị khoá (`EXAM_LOCKED`) | TC-P4-01-32 | cùng mã, cùng HTTP với chat riêng khi thi |
| G4-23 | **PII trong câu trả lời của giảng viên:** tên / MSSV / email SV khác → `422 PII_IN_ANSWER`; email chính giảng viên cho phép | TC-P4-03-28, 29, 30 | đúng; không lối ghi đè |
| G4-24 | **375 px** `/inbox` (danh sách → chi tiết) và `/chat` | TC-P4-03-33, 34; TC-P4-01-18…24 (ở 375) | không tràn ngang; vùng chạm ≥ 44 px |
| G4-25 | **Không PII / nội dung riêng ở log và payload LLM** (story đụng chat): log gateway/worker, `llm_audit`, payload gửi LLM | TC-P4-01-E3; TC-P4-03-24; TC-P4-07-09 | grep canary = 0 |
| G4-26 | **Phân trang:** danh sách ticket, hàng chờ kiểm duyệt, thư lỗi | TC-P4-03-01–04; TC-P4-06-16; TC-P4-04-34 | con trỏ, `limit ≤ 100`, không OFFSET, không `Seq Scan` |
| G4-27 | **Trạng thái tải / rỗng / lỗi và 375 px** cho màn mới (`/inbox`, chuông, `/threads` kiểm duyệt, mục `/observability`) | TC-P4-03-38; TC-P4-02-26, 28; TC-P4-06-47; TC-P4-04-33 | đủ ba trạng thái; dùng được ở 375 px |
| G4-28 | **Migration sạch trên DB trống** và `down`/`up` lại | TC-P4-01-01…06 | sạch |
| G4-29 | **Luật chữ UI** (`CLAUDE.md`): không câu mô tả thừa dưới tiêu đề / nút; tooltip chỉ khi cần; chữ cho sinh viên không có "ticket", "SLA", "AI/RAG/PII", độ tin cậy | TC-P4-01-20; TC-P4-05-27; `eval innerText` grep | 0 từ kỹ thuật ở màn sinh viên |
| G4-30 | **Đọc diff** (`git diff 1518f83..`): lan phạm vi, `fetch` trần ở frontend, thư viện ngoài bảng `ARCHITECTURE.md`, secret, `float64` điểm | – | không có; nếu có → báo PM |
| G4-31 | **Mock → thật:** `grep -rn "@/mock" frontend/src/features/inbox \| wc -l` | TC-P4-03-40 | `0` |
| G4-32 | **Luồng F5 và F4 đi trọn** (FLOWS) kể cả nhánh lỗi liệt kê (mail lỗi, không ai nhận, hai người cùng nhận, báo cáo → ẩn → hiện lại) và F14 (nguồn việc `TICKET`, `POST_REPORTED`, `STAFF_ANSWER_UNREAD`, `COURSE_TICKETS_STALLED` ở "Hôm nay") | TC-P4-03-37; TC-P4-05-23, 29; TC-P4-06-40 | E2E đi từ đầu đến cuối |

## D. Số liệu ghi vào report cổng
Bảng: số TC mỗi story (PASS/FAIL/KHÔNG KIỂM ĐƯỢC); `bell_latency_p95_ms`; `mail_latency_p95_s` (10 lần); recall khủng hoảng; số test Go trước/sau; số test Playwright trước/sau; thời gian CI của lượt chạy đủ.

## Câu hỏi (Q-QC)
- **Q-QC-P4-20** `scripts/gate-p4.sh` (G4-06) do dev tạo; spec chỉ mô tả (SRS 9.2). Nếu chưa có khi chấm → G4-06 "KHÔNG KIỂM ĐƯỢC" = FAIL.
- **Q-QC-P4-21** G4-07 "số test không giảm": mốc là `1518f83` (1436 top-level Go, 822 Playwright); mốc mới sau `73a9cbe` (fix judge) có thể +1. QC đo lại ở `origin/sprint/7-p4-p5` đầu pha 2.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-13: viết lần đầu.

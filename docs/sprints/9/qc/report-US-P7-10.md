# QC report — US-P7-10 (phúc khảo bài tập) · Kết luận vòng 1: **FAIL** (chỉ AC11) → **vòng 2 (`8e478910`): PASS** — xem mục 4
Ngày 2026-11-01. Commit kiểm: `sprint/9-p7` @ `e7aeff77` (máy chủ `3f0f663f`, frontend `d10af653`). Bộ TC: `tc-US-P7-10.md`. Handoff: `dev-US-P7-10.md`. CI `e7aeff77`: **success**.

## 1. Cổng đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `p7-go-tests.sh`, `-race -tags integration`, mọi tên của dòng `Kiểm:` và của handoff: assignment 31, escalation 4, grade 4, today 3 | PASS 42/42 |
| Hộp đen trên stack thật (`scripts/p7-api-10.py`; 3 sinh viên nộp → nhập điểm tay → duyệt → công bố; lần chạy trên `49e4c9b3`, máy chủ không đổi sau đó) | PASS 0 lỗi. Gửi: lý do 9 / 1.001 ký tự, không chọn tiêu chí, tiêu chí lạ → 422; B gửi → 201 `OPEN`; lần hai → 409 `APPEAL_ALREADY_FILED`; một dòng `grade_appeals`, **0 `escalation_tickets`, 0 lời gọi LLM**. Hạn theo đồng hồ máy chủ: −7 ngày −1 phút → 409 `APPEAL_WINDOW_CLOSED` (kèm `closes_at`), −7 ngày +1 phút → 201; bài chưa công bố → 404. Quyết định: phản hồi 9 ký tự → 422 `RESPONSE_REQUIRED`; `ADJUSTED` không đổi tiêu chí → 422 `NO_CHANGE`; hai TEACHER cùng lúc → một 200, một 409 `APPEAL_ALREADY_DECIDED`; `ADJUSTED` sửa tại chỗ (4,00 → 4,50), vẫn `PUBLISHED`, `version + 1`, giữ `published_at` và `ai_scores`, `score_after` ghi; `UPHELD` giữ 8,00; quyết lần hai 409. Phân quyền: STUDENT / ADMIN `GET appeals` 403, TA đọc 200, TA / STUDENT / ADMIN `decide` 403, TEACHER gửi 403, A không đọc phúc khảo của B. "Hôm nay": TEACHER đúng một `APPEAL` ("2 yêu cầu xem lại điểm · …"), `overdue` khi chờ 49 giờ, TA không có; chuông `APPEAL_NEW` gộp 1; `APPEAL_REPLIED` tới đúng B và C; đúng 2 thư (Mailpit); thư B: "Đã sửa điểm", "4,00 → 4,50", phản hồi nguyên văn, **không** nhắc AI / nháp; `stats`: `appealed 2`, `appeal_rate 0.6667`, `adjusted 1`, `appeal_success_rate 0.5000`; route cũ `claim` / `answer` với id phúc khảo → 404 / 405; log không chứa lý do / phản hồi |
| **Đầu-cuối bằng giao diện trên stack dựng lại từ `e7aeff77`** | B mở `/assignments/{id}` → `Yêu cầu xem lại` → chọn tiêu chí "Phân tích tấn công" + lý do → `Gửi`: "Đã gửi yêu cầu xem lại · Chủ Nhật, 13:54"; 375 px không cuộn ngang. TEACHER mở `/grading/{id}`: khối "Yêu cầu xem lại" (sinh viên, tiêu chí, lý do, "Điểm đã công bố 4,00", `Giữ nguyên điểm` / `Sửa điểm`) → `Sửa điểm` (1,5) + phản hồi → `Quyết định` → hộp thoại "Sửa 1 tiêu chí · sinh viên nhận thư phản hồi · không quyết lại được" → DB `ADJUSTED 4.00 → 4.50`. B thấy "4,50 / 10,00", tiêu chí 1,50 / 2,50, "Đã sửa" + phản hồi, nút phúc khảo biến mất; thư tới `sv.kha@`. Ảnh: `shots/p710-student-form-1440.png`, `-student-sent-1440.png`, `-student-sent-375.png`, `-teacher-decide-1440.png`, `-teacher-adjust-1440.png`, `-teacher-confirm-1440.png`, `-student-replied-1440.png` |
| Playwright desktop `grade-appeal.spec.ts` (7 ca: `student files appeal`, `appeal window and states`, `appeal already filed`, `appeal offline draft`, `teacher decides appeal`, `ta cannot decide appeal`, `appeal decided twice`) + `assignment a11y`, `grading a11y` | 11/11 xanh; axe 10 lượt, 0 lỗi |
| Đột biến (`/tmp/qc-mut9`, đã xoá) V1 lệch hạn +48 giờ · V2 bỏ cả hai chặn quyết định lần hai (`status != OPEN`, `n == 0`) · V3 `ADJUSTED` không đổi vẫn được | cả 3 **đỏ**: `TestAppealWindowClosed` + `TestAppealWindowBoundaryTable`; `TestAppealClosedOnDecide`; `TestAppealAdjustedNeedsChange`. (Không chạy M2 / M3 của TC: ghi `escalation_tickets`, gọi LLM — đã kiểm bằng hộp đen và `TestAppealPathNeverCallsLLM`) |
| `ui-antipatterns.sh` | rc 0 |

## 2. Kết quả theo AC
| AC | Kết quả | Ghi chú |
| --- | --- | --- |
| AC1 gửi (tiêu chí + lý do 10–1.000) | PASS | hộp đen + UI + `student files appeal` |
| AC2 hạn do máy chủ, một lần, tắt phúc khảo, chưa công bố | PASS | V1; `TestAppealWindowBoundaryTable`, `…Disabled`, `…NotPublished` |
| AC3 giảng viên quyết định tại chỗ, phản hồi bắt buộc | PASS | UI + hộp đen; `teacher decides appeal` |
| AC4 sổ đã chốt: không chặn, bản chốt không đổi, hiện lệch | PASS ở mức Go | `TestAdjustedWhenLockedKeepsSnapshot`, `…ShowsDriftWhenLocked`, `…AfterUnlockThenRefinalizeNotifies`, `…SerializesWithFinalize`; chưa dựng trên stack |
| AC5 không gọi LLM | PASS | hộp đen (`llm_audit` không tăng); `TestAppealPathNeverCallsLLM`, `TestAppealNotInAgentTools` |
| AC6 chuông, thư, "Hôm nay" | PASS | hộp đen (chuông, 2 thư, `APPEAL` overdue 49 giờ) |
| AC7 không đi qua `escalation_tickets` | PASS | hộp đen; `TestAppealNotInInboxOrMeTickets`, `…NotCountedInTicketCap`, `…AdminStalledIgnoresAppeal`, `…NotInKnowledge` |
| AC8 số liệu | PASS | hộp đen (`0.6667`, `0.5000`); `TestStatsAppealRates`, `TestStatsNullWhenZero` |
| AC9 đồng thời, bấm đúp, mất mạng không mất chữ | PASS | hộp đen (hai quyết định song song); `appeal offline draft`, `appeal decided twice` |
| AC10 phân quyền | PASS | hộp đen ma trận + `ta cannot decide appeal` |
| **AC11 `grade-appeal.spec.ts` đi trọn F11 + nhánh lỗi, gồm sổ đã chốt → mở khoá → chốt lại → `GRADE_CHANGED`** | **FAIL** | 7 ca có (xanh), **thiếu** nhánh sổ chốt (dev tự ghi nợ ở handoff "Bổ sung cùng đợt FE của US-P7-05"). Phía Go phủ nhưng dòng `Kiểm:` / TC-13 đòi spec |

## 3. Lỗi / ghi nhận
- **BUG-P7-10-1 (Trung bình, AC11):** thiếu ca Playwright nhánh sổ đã chốt trong `grade-appeal.spec.ts`. Dev bổ sung (mock) rồi QC chấm lại vòng 2; mã máy chủ không cần kiểm lại ngoài hồi quy.
- Ghi nhận: khối "Yêu cầu xem lại" của TEACHER không hiện điểm AI nháp khi kết quả do giảng viên nhập tay (AI `NULL`) — đúng; ca có điểm AI chỉ kiểm bằng Go / Playwright.
- Ghi nhận: không có job thư nhắc quá hạn / SLA (dev ghi rõ); student view chỉ nêu một mục `recommended` nên `ASSIGNMENT_APPEAL_REPLY` (bậc 48) đứng sau `ASSIGNMENT_GRADED` trong 7 ngày đầu — dev đã ghi.
- Test cũ dev sửa: `contract_test.go` 275 thao tác + `appealScenarios`.

## 4. Vòng 2 — `8e478910` · Kết luận: **PASS**
CI `8e478910`: **success**.
- `grade-appeal.spec.ts` giờ **8 ca**, thêm `appeal locked gradebook drift` (sửa điểm khi sổ đã chốt → "Khác bản chốt" → `Mở khoá` có lý do → `Chốt điểm` v3 → chuông `GRADE_CHANGED` của sinh viên): xanh; nằm trong lượt Playwright của `gate-p7.sh` (`assignment-lifecycle` + `grade-appeal`, `--workers=2 --retries=0`, xanh). BUG-P7-10-1 đã sửa → AC11 **PASS**.
- Hồi quy: `go test -race ./internal/grading/... ./internal/submission/... ./internal/jobs/...` xanh; 42 test của story (assignment 31, escalation 4, grade 4, today 3) xanh ở lượt `go-test-all.sh` (trừ `TestRetryThreeThenDead`, xem report US-P7-07, BUG-P7-07-5).
- Máy chủ không đổi từ `3f0f663f`; AC1–AC10 giữ PASS.

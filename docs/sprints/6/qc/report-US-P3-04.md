# QC report — US-P3-04 (`privacy.Classifier`, `internal/agent`)  · Kết luận: FAIL

Handoff: `docs/sprints/6/handoff/dev-US-P3-04.md`. Bộ TC: `tc-US-P3-04.md` (55 TC). Story giao **gói Go** (không HTTP, chưa có `internal/chat`/`thread`): TC theo API chuyển sang P3-05/06 (Q1), phần còn lại QC kiểm bằng test hộp đen mới `backend-go/internal/agent/qc_p304_test.go` (chỉ dùng hàm export, câu do QC soạn từ SRS 4.4–4.6; lint sạch).

## Cổng đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `go test -count=1 -race ./internal/agent` | PASS — 26 test |
| `go test ./internal/privacy -run 'TestClassify|TestExemplars'` | PASS (2) |
| Tên test của TC (23 tên: `TestClassifyOncePerMessage`, `…RulesSkipEmbed`, `…NoGeneration`, `TestOneEmbedPerMessage`, `TestRouteTable`, `TestOneGenerationPerMessage`, `TestPersonalToolsHaveNoIdentityParam`, `TestThreadsAgentHasNoPersonalTools`, `TestPublicRegistryRejectsPersonalTool`, `TestAskOnBehalfOfOtherRefused`, `TestSelfMentionAllowed`, `TestOtherNameAcademicAllowed`, `TestSelfDeclaredMSSVIgnored`, `TestPhase5And6ToolsNoData`, `TestToolSeamUsesTrustedUser`, `TestCalendarLibraryToolsSeam`, `TestPersonalNeverCached`, `TestPIIMessageNeverCached`, `TestCacheInvalidatedOnDocumentChange`, `TestCrisisCannedReply`, `TestCrisisNoLLMNoNotify`, `TestPromptContextIsQuotedData`, `TestPromptKeepsPlaceholderRule`, `TestInjectedChunkStaysData`) | tất cả **tồn tại**; `TestClassifyOncePerMessage` / `TestOneGenerationPerMessage` ở `chat`/`thread` chưa có gói (dev khai) |
| Probe AC7 (gieo `PublicRegistry.RegisterPersonal` trong `internal/agent`, `go build`) | build **thất bại** (`r.RegisterPersonal undefined`); xoá → ok |
| `grep UserID internal/agent` ngoài test | chỉ `TrustedContext.UserID` và hai chỗ gán từ `tc.UserID` (`infra.go`); không có `json:"user_id"`… trong kiểu tham số tool |

## TC (mức gói)
| TC | Kết quả | Ghi chú |
| --- | --- | --- |
| 02, 07, 11, 14, 15, 19(chat), 20, 29, 32, 35, 38, 44, 47, 51 | PASS (test dev xanh; 19, 14 phần `chat` → P3-05) | |
| 08 / 11 | **FAIL** | bảng QC 39 câu / 12 intent: 36 đúng, **3 sai**: `"Khi nào em thi cuối kỳ?"` → `COURSE_QA` (kỳ vọng `EXAM_SCHEDULE`; đúng ví dụ của AC11); `"giả sử giữa kỳ em được 9 thì sao"` → `COURSE_QA` (kỳ vọng `WHAT_IF_GRADE`); `"cảm ơn bạn nhé"` → `COURSE_QA` (kỳ vọng `SMALLTALK`) |
| 09 | PASS | khủng hoảng thắng hỏi người khác; tên roster → `OTHER_PERSON`; "nếu cuối kỳ em được 8 thì…" → `WHAT_IF_GRADE` |
| 10 | PASS | 5 lần cùng câu → cùng intent |
| 21 | PASS | biên dịch chặn |
| 24 (mức gói) | PASS | điểm / lịch thi / điểm danh × tên có dấu, không dấu, đảo, MSSV, email → `OTHER_PERSON`; `TestObjectMSSVRefusedEvenOutsideRoster` xanh |
| 27, 28, 30, 31 | PASS | tự nhắc mình → `PERSONAL_ATTENDANCE`; câu học thuật có tên → không `OTHER_PERSON`, `HasPII=true` (không cache); MSSV tự khai không ai → như mình; MSSV của SVB → `OTHER_PERSON` |
| 45 (mức gói) | PASS | 9 câu khủng hoảng (có dấu / không dấu: `tự tử`, `tu tu`, `tự hại`, `tự làm đau`, `không muốn sống`, `khong muon song`, `muốn chết`, `muon chet`, `kết thúc cuộc sống`) → `CRISIS`; 3 câu "chết" nghĩa bóng → `COURSE_QA` (không báo động nhầm) |
| 17, 53 | PASS (một phần) | 17 chưa gieo trường `student_code` (chỉ đọc test); 53 xem cổng |
| 01, 03–06, 12, 13, 18, 22, 23, 25, 26, 33, 34, 36, 37, 39–43, 46, 48–50, 52, 54, 55 | KHÔNG KIỂM ĐƯỢC → chuyển P3-05 (chat + SSE + payload + `pii_events`/log thật) / P3-06 (22, 23 Threads) / P8-02, P8-03 (37) | cần `/chat/sessions`, `/threads` |
| 16 | PASS | không `json:"user_id"…` trong tham số tool |

## AC
| AC | Kết quả |
| --- | --- |
| AC1 phân loại một lần | PASS ở `agent`; chat/thread → P3-05/06 |
| AC2 luật → embedding, không LLM | PASS (test dev) |
| AC3 bảng định tuyến | **FAIL** (3/39 sai trên bảng QC) |
| AC4 một lần sinh / 0 lần | PASS ở `agent` |
| AC5 tool không có danh tính | PASS |
| AC6 `trusted_context` từ JWT | chưa giao (dev khai; P3-05) |
| AC7 Threads không tool cá nhân | PASS (compile + test) |
| AC8 hỏi hộ | PASS mức gói |
| AC9 MSSV tự khai | PASS mức gói |
| AC10–11 NoData / seam | PASS (test dev) |
| AC12 cache | PASS (test dev, Redis thật); QC chưa chạy ngoài |
| AC13 khủng hoảng | PASS mức gói; log / `notifications` chưa đo |
| AC14 tiêm lời nhắc | PASS (test dev); kịch bản Threads → P3-06 |
| AC15 phân quyền | chờ P3-05 |

## Lỗi
- **BUG-1 (Trung bình)** — định tuyến bỏ sót cách hỏi tự nhiên: `"Khi nào em thi cuối kỳ?"` (ví dụ nêu trong spec AC11) → `COURSE_QA`, sẽ trả lời bằng RAG thay vì `get_exam_schedule`. Tái hiện: `go test -count=1 ./internal/agent -run TestQCRouteTable -v`. SRS ghi dấu hiệu "lịch thi, khi nào thi" — chen "em" làm hụt. Nghi: `internal/agent/intent.go` khớp chuỗi liền kề.
- **BUG-2 (Thấp)** — `"giả sử giữa kỳ em được 9 thì sao"` không vào `WHAT_IF_GRADE` (SRS chỉ ghi dạng "nếu … được 8 thì"; có thể chấp nhận → Q-QC).
- **BUG-3 (Thấp)** — `"cảm ơn bạn nhé"` (14 ký tự) không phải `SMALLTALK`; SRS ghi "chào, cảm ơn, ≤ 12 ký tự không từ khoá" (đọc được hai cách → Q-QC).

## Đề nghị
FAIL tới khi BUG-1 sửa và BA trả lời BUG-2/3. TC cần API chuyển sang P3-05 / P3-06 theo Q1.

## Trạng thái QC (tạm dừng theo lệnh chủ dự án)
Đã xong: report P3-01…P3-04, P8-01. Việc dở khi dừng: (1) cập nhật TC theo spec v1.3/v1.4 (#8–#13) chưa làm; (2) TC chuyển sang P3-05/06, P8-02/03 chờ handoff; (3) chưa chạy P8-01 TC-26/27/44/55. Stack QC cục bộ đã dừng (DB `qc_*` đã xoá), stack dev không đụng.

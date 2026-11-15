# QC test case — US-DM-05 (cổng chat), US-DM-09 (ranh giới an toàn)
Nguồn: US.md + SRS §3.4, 4.3, 8, 9; D47(1), D60, Q-D-9; US-P3-04 AC13. Hộp đen; quy ước: `tc-00-chung.md`. Chat: sinh chữ qua 9router (`ag/gemini-3.8-flash`) khi đo; kiểm logic bằng `fake`. Bật `CHAT_ROUTING` ở lớp 1. `sv.kha` hỏi qua `POST /chat/sessions/{id}/messages` (SSE).
11 nhãn: `WHAT_IF_GRADE`, `GRADE_FORMULA`, `PERSONAL_ATTENDANCE`, `PERSONAL_PARTICIPATION`, `PERSONAL_GRADE`, `EXAM_SCHEDULE`, `UPCOMING_EVENTS`, `LIBRARY_SEARCH`, `COURSE_QA`, `OTHER_PERSON`, `SMALLTALK` (không `CRISIS`).

## US-DM-05 Cổng `CHAT_ROUTING`
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-D05-01 | AC1 | Cổng bật | Tin có luật ý định cụ thể khớp (ví dụ hỏi điểm chuyên cần của tôi) | **0** dòng `DECIDE` (luật đã khớp, decision không được gọi); ý định theo luật | API+DB |
| TC-D05-02 | AC1 | Cổng bật | Tin không khớp ý định cụ thể (rơi `COURSE_QA`) | Đúng **1** dòng `DECIDE` (`cascade_gate=CHAT_ROUTING`, làn `INTERACTIVE`) | API+DB |
| TC-D05-03 | AC1 | Cổng bật | Thứ tự sự kiện SSE: bắt `received`, mốc thời gian | `received` phát **trước** mọi bước, ≤ 300 ms từ lúc gửi; không đổi so với tắt cổng | API |
| TC-D05-04 | AC2 | `FAKE_P50_MS=600` | Đo thời gian tới sự kiện nhúng + quyết định trong request (qua log `trace` / SSE `intent`) | Decision chạy **song song** nhúng (tổng ≈ max, không cộng); `TestOneEmbedPerMessage` PASS (đúng 1 nhúng mỗi tin) | API+GO |
| TC-D05-05 | AC2 | `FAKE_P50_MS=3000` | Chat | Chờ tối đa 800 ms rồi bỏ decision, dùng luật; không treo | API |
| TC-D05-06 | AC3 | Replay: confidence 0,97 (≥ τ), 0,95, 0,949, 0,5 | Chat từng câu | ≥ 0,95 → nhãn decision (`FAST`); 0,949 và 0,5 → **giữ luật** (`FALLBACK`); không lời gọi LLM thêm cho định tuyến (`llm_audit` không dòng `CLASSIFY` mới) | API+DB |
| TC-D05-07 | AC3 | `FAKE_MODE=error` / `timeout`; cầu dao mở; `budget_blocked` | Chat | Giữ luật (`FALLBACK`), trả lời như tắt cổng, không treo | API |
| TC-D05-08 | AC3 | Cổng bật, 1 tin | Đếm lời gọi trong `llm_audit` theo `trace_id` | ≤ 1 sinh chữ + ≤ 1 nhúng + ≤ 1 decision; `go test … -run TestCallBudgetPerMessage -v` PASS | API+GO |
| TC-D05-09 | AC4 | Decision (giả) trả nhãn khác luật khi luật đã khớp | Chat | Không ghi đè: dùng luật; `TestRouteDecisionNoOverride` PASS | GO |
| TC-D05-10 | AC4 | Mẫu khủng hoảng trong luật (ít nhất 5 mẫu) | Chat, cổng bật | Phản hồi hỗ trợ khủng hoảng như hiện hành; **0** `DECIDE`, 0 lời gọi LLM, 0 `llm_audit`, log không có nội dung (`TestCrisisNeverReachesDecide` chat PASS) | API+DB+GO |
| TC-D05-11 | AC5 | Replay trả `OTHER_PERSON` cho «điểm của bạn Lan» | Chat | Từ chối + log như hiện hành (cùng đường với luật); công cụ cá nhân lấy danh tính từ `trusted_context`, không từ decision (`student_code` không do LLM / decision đề xuất) | API |
| TC-D05-12 | AC5 | Replay trả `PERSONAL_GRADE` cho câu mơ hồ | Chat | Đi cùng đường như luật cho nhãn đó; `sv.kha` chỉ thấy dữ liệu **của mình** (không thấy dữ liệu SV khác dù câu chứa MSSV người khác) | API |
| TC-D05-13 | AC6 | Cùng seed, 9router thật, `FAKE_P50/P95=426/682` | `benchmarks` (hoặc script QC): 30 câu × 3 lượt, tắt rồi bật cổng; đo TTFT p50 / p95 và SSE đầu | SSE đầu ≤ 300 ms **không đổi**; TTFT p95 bật ≤ **+20 %** so với tắt; nếu vượt → ghi số, cổng giữ **tắt**, **không** nới ngưỡng; ghi «trên MacBook Apple Silicon, Colima 6 CPU / 10 GB» | TAY |
| TC-D05-14 | AC7 | Bộ 61 câu định tuyến (`benchmarks`, PoC C) + bộ diễn đạt lại | Chạy bật vs luật một mình | Độ chính xác bật ≥ luật (61/61); nhãn khác luật chỉ nhận khi ≥ τ; báo riêng số ca decision đổi nhãn đúng / sai | API |
| TC-D05-15 | AC8 | `GEMINI_API_KEY` trống / hết hạn mức / mạng lỗi | Chat 20 câu | Trả lời bằng luật như tắt; không treo; không thêm trễ vượt 800 ms (so p95) | API |
| TC-D05-16 | AC9 | Chat nhiều lượt: lượt 1 chứa MSSV người khác + điểm; lượt 5 hỏi mơ hồ | Bắt thân tới provider giả (máy chủ giả) | Chỉ **tin hiện tại đã che**; không lịch sử, không tên / MSSV / dữ liệu điểm; ≤ 2.000 ký tự (tin 5.000 ký tự → cắt 2.000); `llm_audit` không nội dung | API+DB |
| TC-D05-17 | – | Cổng chat tắt ở lớp | Chat | 0 `DECIDE`; hành xử như trước | API+DB |
| TC-D05-18 | – | SV lớp 2 (cổng bật chỉ lớp 1) | Chat cùng tin | Không `DECIDE` | API+DB |
| TC-D05-19 | AC1 | Hai tin liên tiếp rất nhanh (race) | Chat | Mỗi tin một quyết định đúng; không lẫn nhãn giữa phiên | API |

## US-DM-09 Ranh giới an toàn (xuyên suốt)
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-D09-01 | AC1 | Repo | `go test ./... -run TestDecideCallersAllowlist -v`; `grep -rn "\.Decide(\|llm.Cascade(\|Cascade(" backend-go/internal --include=*.go \| grep -v _test` | Chỉ 3 nơi: tường lửa Threads (`internal/privacy`), `flagInjectionDecision` (`internal/grading`), định tuyến ý định (`internal/agent` / `internal/chat`); **không** từ `internal/grade`, tính / làm tròn điểm, mức rubric, công thức, escalate, đường khủng hoảng, `internal/rag` | GO+DIFF |
| TC-D09-02 | AC1 | – | Thêm bản sao tạm lời gọi `Decide` trong `internal/grade` (QC, không commit) | `TestDecideCallersAllowlist` **đỏ**; xoá bản sao | GO |
| TC-D09-03 | AC2 | – | `go test … -run TestCrisisNeverReachesDecide -v` (chat **và** Threads) | PASS cả hai | GO |
| TC-D09-04 | AC2 | Tin khủng hoảng ở chat và bài Threads, cổng bật cả hai | Gửi 10 mẫu; đếm `llm_audit` | 0 dòng `DECIDE`, 0 `CLASSIFY` / `UTILITY` / sinh chữ cho các tin đó; `grep` log gateway / worker nội dung tin = 0 | API+DB |
| TC-D09-05 | AC3 | Cổng bật cả ba | `grep` toàn bộ thân gửi provider giả (máy chủ giả) của ≥ 100 lời gọi | Không `student_id`, tên, MSSV, email, điểm; trần độ dài đúng theo cổng; `TestDecideMasked` PASS | API |
| TC-D09-06 | AC3 | – | Quét schema DB: `SELECT table_name, column_name FROM information_schema.columns WHERE …` các bảng mới + `llm_audit`; `grep` migration | Không bảng / cột lưu đầu vào hay đầu ra của decision (chỉ `llm_audit` metadata, `decision_gates`, `course_decision_gates`) | DB |
| TC-D09-07 | AC4 | Repo | Đọc `docs/PRODUCTION_READINESS.md` §1, `docs/guides/decision-model.md` mục đầu, `.env.example` | Ghi «triển khai thật dùng gói trả phí»; `LLM_DECISION_ALLOW_REAL` mặc định `false`; guide nhắc điều này ở mục đầu | DIFF |
| TC-D09-08 | AC5 | Tắt provider / cầu dao mở / khoá trống | Chạy lại bộ E2E hiện có của Threads, chấm, chat (`threads`, `private-chat`, `grading`) | Hành xử **như trước 10.5** (xanh, không sửa assertion); riêng Threads dùng `Structured` rồi chặn khi lỗi cả hai; test «provider decision tắt → mọi cổng như cũ» tồn tại và PASS | E2E+GO |
| TC-D09-09 | AC6 | – | `git diff <gốc>... -- backend-go/go.mod go.sum frontend/package.json docs/ARCHITECTURE.md` | Không nhà cung cấp / khoá / SDK / thư viện mới; khoá dùng `GEMINI_API_KEY` sẵn có | DIFF |
| TC-D09-10 | AC7 | Cổng chat bật | 20 tin; `llm_audit` theo `trace_id`; `TestCallBudgetPerMessage` | ≤ 1 sinh chữ + ≤ 1 nhúng + ≤ 1 decision mỗi câu; không vòng lặp agent mở | API+GO |
| TC-D09-11 | Tổng | `ep-qc`, mạng | Bắt gói / `lsof -i` gateway + worker suốt pha 2 | **Không** kết nối ra `generativelanguage.googleapis.com`; chỉ 9router `host.docker.internal:20128`; nếu có → lỗi nghiêm trọng | TAY |

## Nhánh lỗi
TC-D05-05, 07, 15, 19; TC-D09-04, 08.

## Phân quyền
TC-D05-11, 12, 18; TC-D09-05.

## Lịch sử sửa TC
(chưa có)

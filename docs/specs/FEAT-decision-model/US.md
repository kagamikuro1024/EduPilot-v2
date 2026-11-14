# FEAT-decision-model Decision model (`gemini-decision-1`) và cascade ba cổng
Nguồn: ADR **D60** (`docs/DECISIONS.md`), D47(2), P-11-9, P-11-10, P-11-11 (`docs/sprints/11/proposals.md`) · `docs/sprints/10.5/plan.md` · `docs/research/2026-11-14-jev-decision-model.md` mục 6, `docs/research/poc/jev-decision/cascade.md` · `docs/ARCHITECTURE.md` §`internal/llm`, §Cascade, env · `docs/SYSTEM_DESIGN.md` (đường nóng hỏi–đáp, decision) · FLOWS F3, F4, F9 + «Decision cascade» · PRD M1, M8 · `docs/design/DESIGN.md` §13, §14.22, §14.23 · `docs/UX.md` mục 6 · FEAT-llm-gateway · FEAT-private-chat-pii · FEAT-assignments · FEAT-p10-eval US-P10-01, 03, 12.
Phase: sprint 10.5. Trạng thái: **APPROVED** v1.0 (2026-11-14; PM uỷ quyền; câu không đụng điểm / auth / PII: **PM chốt theo đề xuất BA**). Tách từ **US-P10-13** của FEAT-p10-eval (nay ghi «chuyển sang FEAT-decision-model»).

**Luật chạy (chủ dự án / PM 2026-11-14):** sinh chữ chỉ qua 9router; `gemini-decision-1` thật và embedding thật **chỉ khi PM cho phép từng lượt**. Mặc định mọi môi trường (dev, test, CI, seed, demo, eval) dùng provider `decision-fake`. Mọi cổng **mặc định tắt**.

**Nguyên tắc bất biến của feature**
1. Decision model là lớp **thêm** xác suất, **không bao giờ là cổng duy nhất** (nhà cung cấp tự công bố: «treat input as untrusted»).
2. Chỉ ba cổng: `THREADS_PRECHECK`, `INJECTION_FLAG`, `CHAT_ROUTING`.
3. **Cấm** dùng cho khủng hoảng (US-P3-04 AC13), điểm, mức rubric, công thức điểm, escalate. Tin khủng hoảng không rời máy chủ.
4. Mọi nội dung gửi decision đã che danh tính như mọi lời gọi LLM; `llm_audit` không có nội dung.
5. Decision lỗi không bao giờ làm hỏng chat, đăng bài hay việc chấm: mọi lỗi → đường fallback của cổng; cổng an toàn lỗi hết đường → fail-closed.
6. Chỉ ADMIN cấu hình và xem số liệu; không từ kỹ thuật nào xuất hiện ở màn sinh viên, giảng viên, trợ giảng.

Mã vai: ADMIN, TEACHER, TA, STUDENT. Tên cổng ở giao diện: «Kiểm duyệt Threads» (`THREADS_PRECHECK`), «Cờ chèn lệnh khi chấm» (`INJECTION_FLAG`), «Định tuyến câu hỏi chat» (`CHAT_ROUTING`).

---

## US-DM-01: Dev muốn một hàm `Decide()` và provider loại `decision` đi cùng đường ống với mọi lời gọi LLM, để dùng decision model an toàn và đo được
Ưu tiên: Must · Ước lượng: L · Sprint: 10.5 · **L1**

### Tiêu chí nghiệm thu
- AC1 (hợp đồng). Given `internal/llm` Then `Client` có **năm** hàm; `Decide(ctx, DecideRequest) (DecideResponse, error)`; `DecideRequest{Task, Lane, Gate, Input, Questions}` với `Questions` là kiểu Go đóng `noul` / `choice` / `score` (không JSON tự do, không chuỗi chỉ thị từ người dùng); `DecideResponse` mang từng câu trả lời: `noul` hoặc `choice` + `probabilities` + `confidence`, và `Usage{TokensIn, TokensOut=0}`.
- AC2 (provider thật). Given provider loại `decision`, model `gemini-decision-1` When gọi thật Then `POST https://generativelanguage.googleapis.com/v1beta/decisions` bằng `net/http`, header `x-goog-api-key`, thân `{model, input, questions}`; **không SDK** (kể cả `google.golang.org/genai`); khoá = bản ghi `llm_providers` (mã hoá) hoặc `GEMINI_API_KEY` khi bản ghi trống; không khoá / bên xử lý mới.
- AC3 (provider giả). Given model `decision-fake` Then không mạng, không khoá, tất định: trả xác suất theo bản phát lại (`internal/llm/testdata/decision-replay.json`, khoá = sha256 của đầu vào đã che + id câu hỏi); đầu vào không có trong bản phát lại → xác suất 0,5 (vùng lưỡng lự, ép đi fallback); độ trễ đặt được (`LLM_DECISION_FAKE_P50_MS`, `…_P95_MS`; mặc định 426 / 682, số PoC), tỉ lệ lỗi đặt được (`LLM_DECISION_FAKE_ERROR_RATE`, mặc định 0). **Mặc định của dev / test / CI / seed / demo / eval.**
- AC4 (đường ống). Given mọi lời gọi `Decide` Then đi: che PII (`Masker`) → giấy phép Scheduler (task `DECIDE`; làn `INTERACTIVE` cho `CHAT_ROUTING` và `THREADS_PRECHECK`, `BATCH` cho `INJECTION_FLAG`; **token bucket riêng** của provider `decision`, mặc định 150 RPM) → provider → `llm_audit` (task, `provider`, `model`, `tokens_in`, `latency_ms`, `status`, `error_kind`, `cost_est`, `cascade_gate`, `cascade_path`; **không nội dung**) → cầu dao riêng của provider `decision`.
- AC5 (thời hạn, không thử lại). Given `Decide` Then mỗi lời gọi có thời hạn của cổng (`timeout_ms`, mặc định `LLM_DECISION_TIMEOUT` = 800 ms; `INJECTION_FLAG` 3.000 ms) và không vượt phần hạn còn lại của request gốc; **không thử lại** (lỗi → trả ngay để cascade chọn fallback).
- AC6 (ánh xạ lỗi). Given phản hồi `429` → `rate_limited`; `5xx` / hết giờ / mạng → `error`/`timeout`; `400` thuộc lọc an toàn (`safety`, `prohibited_content`) → `error_kind=SAFETY_BLOCK`; cầu dao mở → `circuit_open`; hết ngân sách → `budget_blocked`; khoá trống / không đọc được → `not_configured`. Mỗi trạng thái là một giá trị đã có của `llm_audit.status` (không thêm trạng thái mới trừ `error_kind`).
- AC7 (chặn gọi thật ngoài phép). Given biến `LLM_DECISION_ALLOW_REAL` (mặc định `false`) Then khi `false`, chọn model `gemini-decision-1` cho provider → `409 DECISION_REAL_DISABLED`, và nếu bản ghi sẵn có thì `Decide` trả `not_configured` (không gọi mạng); khi `true`, trần **`LLM_DECISION_REAL_DAILY_CAP` = 300 lời gọi thật / ngày** (UTC+7); vượt → `budget_blocked`, cascade đi fallback.
- AC8 (chi phí). Given `llm_models` của `gemini-decision-1` Then `price_in` seed = 2.600 VND / 1 triệu token (≈ $0,10, tỉ giá ghi chú `[ƯỚC TÍNH]`), `price_out` = 0; `cost_est` tính theo token vào; `decision-fake` giá 0.
- AC9 (kiểm). Given test Then `TestDecideAudit` (một dòng `llm_audit` mỗi lời gọi, không nội dung), `TestDecideNoSDK` (`go list -deps` không có `google.golang.org/genai`), `TestDecideMasked` (tên + MSSV thật → không có trong thân yêu cầu), `TestDecideErrorMap` (bảng lỗi → status), `TestDecideRealGuard`, `TestDecideFakeDeterministic`.
- AC10 (nhánh lỗi). Given provider `decision` không cấu hình hoặc bị tắt Then `Decide` trả `not_configured` trong < 1 ms (không chờ), không treo, không ghi `llm_audit` lỗi lặp (một dòng / lần gọi).

### Ngoài phạm vi của story này
- Cascade (US-DM-02); từng cổng (US-DM-03…05); màn hình (US-DM-06, 07); Jev / loại `typesafe`.

### Phụ thuộc
- D60; `FEAT-llm-gateway` (Scheduler, cầu dao, `llm_audit`, `Masker`); migration `llm_providers.type` thêm `decision`.

---

## US-DM-02: Dev muốn engine cascade và cấu hình cổng (ngưỡng, bật / tắt toàn hệ thống và theo lớp, mặc định tắt), để bật từng cổng từng lớp mà không sửa mã
Ưu tiên: Must · Ước lượng: L · Sprint: 10.5 · **L1**

### Tiêu chí nghiệm thu
- AC1 (chính sách). Given `internal/llm/cascade.go` Then có hai chính sách: `Symmetric{τ}` (nhận câu trả lời của decision khi `confidence ≥ τ`, ngược lại fallback) và `AddOnly{ε}` («có» với `P(có) ≥ 0,5` nhận ngay; «không» chỉ nhận khi `P(có) ≤ ε`; vùng giữa → fallback).
- AC2 (đường đi). Given `Cascade(ctx, gate, req, policy, fallback)` Then trả `Verdict` + `Path ∈ {FAST, FALLBACK, FAIL_CLOSED}`: `FAST` = dùng kết quả decision; `FALLBACK` = decision lỗi / hết hạn / lưỡng lự / không sẵn sàng và fallback dùng được; `FAIL_CLOSED` = decision **và** fallback đều không dùng được → giá trị an toàn của cổng (Threads: chặn; chèn lệnh: coi như không có thêm cờ từ tầng này; chat: giữ luật). `Path` ghi vào `llm_audit.cascade_path`.
- AC3 (cấu hình hai tầng). Given cổng Then **hiệu lực = công tắc toàn hệ thống VÀ công tắc của lớp** (cả hai mặc định `false`); tắt một trong hai → cổng chạy như **trước sprint 10.5**, không gọi `Decide`, không ghi `llm_audit` decision.
- AC4 (ngưỡng và thời hạn theo cổng). Given bảng cấu hình Then mỗi cổng có `threshold` (`THREADS_PRECHECK` ε mặc định 0,05, khoảng 0,01–0,20; `CHAT_ROUTING` τ mặc định 0,95, khoảng 0,70–0,99; `INJECTION_FLAG` τ mặc định 0,50, khoảng 0,30–0,90) và `timeout_ms` (mặc định 800 / 800 / 3.000, khoảng 200–5.000); ngoài khoảng → `422 VALIDATION_FAILED` kèm `field`.
- AC5 (đọc cấu hình ở đường nóng). Given request Then cấu hình đọc từ **ảnh chụp trong bộ nhớ**, làm mới theo sự kiện Redis (`ep:decision:cfg`) khi ADMIN đổi, TTL 60 s làm lưới an toàn; **không truy vấn DB** trên đường chat; không tải được cấu hình → coi như tắt.
- AC6 (đổi cấu hình). Given ADMIN `PUT` đổi công tắc / ngưỡng Then ghi `audit_log` (entity `decision_gate`, `before` / `after`, actor), tăng `version`, phát sự kiện; `version` cũ → `409 VERSION_CONFLICT`; hiệu lực ≤ 5 s ở mọi bản gateway (kiểm với hai bản).
- AC7 (cầu dao). Given cầu dao provider `decision` mở Then mọi cổng đi thẳng fallback (`FALLBACK`), không gọi mạng, không chờ.
- AC8 (an toàn khi lỗi cả hai). Given test bảng `TestCascadeTable` (policy × xác suất × lỗi) và `TestCascadeFailClosed` Then đúng `Verdict` và `Path` ở mọi ô; `AddOnly` không bao giờ **gỡ** một chặn có sẵn.
- AC9 (phân quyền). Given Teacher / TA / STUDENT gọi API cấu hình Then `403`; sinh viên và giảng viên không thấy tên cổng, ngưỡng hay đường đi ở bất kỳ phản hồi nào (kiểm hợp đồng: không trường `cascade_*` trong DTO không phải ADMIN).

### Ngoài phạm vi của story này
- Cổng cụ thể; màn hình cấu hình (US-DM-06).

### Phụ thuộc
- US-DM-01; bảng `decision_gates`, `course_decision_gates` (SRS 5).

---

## US-DM-03: Quản trị muốn cổng «Kiểm duyệt Threads» có decision chỉ thêm chặn, để bắt câu hỏi riêng tư tinh vi mà luật bỏ sót, nhanh hơn chỉ-LLM
Ưu tiên: Must · Ước lượng: L · Sprint: 10.5 · **L2** · F4

### Tiêu chí nghiệm thu
- AC1 (vị trí). Given sinh viên bấm **đăng** bài / bình luận (hoặc sửa bài) ở lớp có cổng hiệu lực When máy chủ kiểm trước khi lưu Then thứ tự: luật tất định hiện có (regex MSSV / email / SĐT / CCCD, từ điển roster, mẫu câu cá nhân, embedding) → nếu luật đã chặn: **dừng, không gọi decision** → nếu luật cho qua: hỏi decision `noul` «câu hỏi riêng tư hoặc nêu tên người cụ thể» (`AddOnly{ε}`).
- AC2 (kết quả). Given decision Then `P(có) ≥ 0,5` → chặn (`FAST`); `P(có) ≤ ε` → cho qua (`FAST`); giữa → LLM `Structured` qua 9router với cùng câu hỏi (`FALLBACK`); decision lỗi / hết hạn / cầu dao mở → thẳng `Structured` (`FALLBACK`); `Structured` cũng lỗi hoặc quá `THREADS_STRUCTURED_TIMEOUT` (6 s) → **chặn** (`FAIL_CLOSED`).
- AC3 (chỉ thêm chặn). Given một bài mà luật đã chặn Then **vẫn chặn** bất kể decision / `Structured`; không đường nào gỡ chặn của luật; `TestPrecheckAddOnly` kiểm trên toàn bộ mẫu `dev` của E1.
- AC4 (câu chỉ có placeholder). Given nội dung chỉ chứa `[[MSSV_1]]` / `[[SV_1]]` Then luật chặn trước khi tới decision (model đọc không ra placeholder, F1 0,883); `TestPlaceholderOnlyBlockedByRules`.
- AC5 (không áp lúc gõ). Given sinh viên đang soạn (`POST …/threads/precheck`) Then chỉ chạy luật + embedding như cũ, **không** gọi decision (tiết kiệm hạn mức / độ trễ); giao diện không đổi.
- AC6 (người dùng thấy gì). Given bài bị chặn (do luật, decision, `Structured` hay fail-closed) Then hộp thoại hai lối như hiện hành («Chuyển sang chat riêng» / «Ẩn thông tin rồi đăng»), **cùng lời**, không nêu nguyên nhân kỹ thuật; chữ đã gõ không mất (`useAutosaveDraft`); lúc chờ > 800 ms hiện chỉ báo «Đang kiểm tra».
- AC7 (khủng hoảng). Given bài khủng hoảng Then luật xử lý trước và trả `422 CRISIS_SUPPORT` như hiện hành, **không** tới decision / `Structured`; `TestCrisisNeverReachesDecide` (Threads).
- AC8 (cổng đo). Given tập `dev` của E1 (provider giả, bản phát lại có nhãn) Then (i) không mẫu dương nào luật đã bắt mà nay lọt, (ii) số dương không do luật bắt được **không giảm** so với luật một mình, (iii) chặn nhầm trên mẫu âm ≤ 0,05, (iv) rò rỉ payload = 0; tập `test` chỉ chạy một lần ở lần chốt. Số đo thật (nếu PM cho phép) ghi nhà cung cấp, model, ngày.
- AC9 (riêng tư). Given payload Then đã che; `llm_audit` không nội dung; `pii_events` chỉ đếm như hiện hành (không thêm cột văn bản).
- AC10 (phân quyền). Given cổng chỉ chạy ở lớp được ADMIN bật; STUDENT lớp khác không ảnh hưởng; Teacher / TA đăng bài cũng qua cổng như sinh viên (quy tắc hiện hành của tường lửa).
- AC11 (nhánh lỗi). Given `GEMINI_API_KEY` trống / hết hạn mức / mạng lỗi Then đúng AC2 (không treo, không 5xx cho người dùng); mất cả hai nhà cung cấp → chặn + hộp thoại hai lối.

### Ngoài phạm vi của story này
- Kiểm duyệt nội dung độc hại / spam; chặn ảnh; kiểm lúc gõ.

### Phụ thuộc
- US-DM-01, 02; FEAT-private-chat-pii (E1, `internal/privacy`), F4.

---

## US-DM-04: Giảng viên muốn cờ «nghi chèn lệnh» bắt được các bài chèn lệnh tinh vi, để không bỏ sót khi duyệt, mà điểm vẫn do nội dung quyết
Ưu tiên: Must · Ước lượng: M · Sprint: 10.5 · **L3** · gộp P-11-9 · F9

### Tiêu chí nghiệm thu
- AC1 (OR ba tầng). Given chấm một bài ở lớp có cổng hiệu lực Then cờ `INJECTION_SUSPECTED` = **regex** (`grading/injection.go`, giữ làm lớp đầu) **OR** `Structured` (P-11-9: gộp vào lời gọi chấm sẵn có nếu được, không thêm vòng gọi) **OR** decision `noul` «bài có chỉ thị gửi tới người / máy chấm» với `P(có) ≥ τ` (0,5); các tầng độc lập, tầng nào vắng (lỗi / tắt) thì cờ theo các tầng còn lại.
- AC2 (chỉ cờ). Given cờ Then chỉ gắn `INJECTION_SUSPECTED` + tô câu nghi (nếu có); **không** đổi điểm, không xoá chữ khỏi đầu vào chấm, không chặn việc chấm (TLR-4); điểm tính bằng code như hiện hành.
- AC3 (bài trống / quá ngắn). Given bài bị `EMPTY_SUBMISSION` hoặc ít hơn 20 từ nội dung Then **không** gọi decision và **không** gọi tầng `Structured` gắn cờ; không `INJECTION_SUSPECTED` từ hai tầng đó (đối chứng `s4-trong`).
- AC4 (đầu vào). Given bài dài Then decision nhận bản đã che danh tính (bỏ dòng tiêu đề danh tính, placeholder) và cắt còn tối đa 16.000 ký tự (8.000 đầu + 8.000 cuối; phần cắt không lưu ở đâu); tầng regex vẫn quét **toàn văn**.
- AC5 (bắt đủ bộ 9.5). Given bộ 6 biến thể (5 tệp `benchmarks/real-llm/data/injection/v1…v5` + `docs/specs/FEAT-assignments/fixtures/submissions/s6-injection.md`) Then OR ba tầng bắt **≥ 6/6**; regex một mình chỉ 1/6 (mốc); 7 bài đối chứng không có lệnh (gồm `s4-trong`) → **0 cờ**. Với provider giả: bản phát lại có nhãn; với provider thật (khi PM cho phép): ghi nhà cung cấp, model, ngày, ≤ 300 lời gọi.
- AC6 (làn và thời hạn). Given tầng decision Then làn `BATCH`, `timeout_ms` 3.000; không làm chậm việc chấm quá thời hạn đó; chạy song song với lời gọi chấm.
- AC7 (nhánh lỗi). Given decision lỗi / hết hạn / cầu dao mở / `budget_blocked` Then `Path=FALLBACK` (tầng vắng), việc chấm vẫn hoàn tất, điểm không đổi, hàng chờ không treo; `llm_audit` ghi lỗi.
- AC8 (riêng tư và quyền). Given payload Then đã che; cờ chỉ TA / TEACHER của lớp thấy ở màn duyệt (như hiện hành); **sinh viên không bao giờ thấy cờ**; cờ không vào bản công bố.
- AC9 (ranh giới điểm). Given mã Then `Decide` ở luồng chấm chỉ được gọi từ một hàm tên rõ (`flagInjectionDecision`); `TestDecideCallersAllowlist` chặn mọi lời gọi khác (tính điểm, mức rubric, tổng, làm tròn).

### Ngoài phạm vi của story này
- Chọn mức rubric / điểm bằng decision (chờ nhãn người E3, sau bảo vệ); chặn bài bị cờ.

### Phụ thuộc
- US-DM-01, 02; P-11-9; FEAT-assignments (F9, 4.6.3); bộ 9.5.

---

## US-DM-05: Sinh viên muốn câu hỏi chat được hiểu đúng ý nhanh hơn, mà không chờ lâu hơn để thấy chữ đầu tiên
Ưu tiên: Must · Ước lượng: L · Sprint: 10.5 · **L4** · F3

### Tiêu chí nghiệm thu
- AC1 (khi nào gọi). Given tin chat ở lớp có cổng hiệu lực Then thứ tự giữ nguyên: `received` phát trước mọi bước (≤ 300 ms); **luật khủng hoảng** (0 lời gọi model); luật ý định cụ thể. Chỉ khi luật **không khớp ý định cụ thể** (rơi về mặc định `COURSE_QA`) mới hỏi decision `choice` trong các nhãn `WHAT_IF_GRADE`, `GRADE_FORMULA`, `PERSONAL_ATTENDANCE`, `PERSONAL_PARTICIPATION`, `PERSONAL_GRADE`, `EXAM_SCHEDULE`, `UPCOMING_EVENTS`, `LIBRARY_SEARCH`, `COURSE_QA`, `OTHER_PERSON`, `SMALLTALK` (**không có** `CRISIS`).
- AC2 (song song). Given quyết định Then gọi `Decide` **song song** với lần nhúng duy nhất (`TestOneEmbedPerMessage` giữ nguyên); chờ tối đa `timeout_ms` (800 ms); quá hạn → bỏ qua decision.
- AC3 (ngưỡng và fallback = luật). Given `confidence ≥ τ` (0,95, `Symmetric`) Then dùng nhãn decision (`FAST`); dưới τ / lỗi / hết hạn / cầu dao mở / `budget_blocked` → **giữ kết quả luật** (`FALLBACK`), **không gọi LLM** (D47(1) giữ: ≤ 1 sinh chữ + ≤ 1 nhúng + ≤ 1 decision mỗi câu).
- AC4 (không ghi đè). Given luật đã khớp ý định cụ thể Then decision **không** được gọi và không ghi đè; tin `CRISIS` không bao giờ tới decision (`TestCrisisNeverReachesDecide`, `TestRouteDecisionNoOverride`).
- AC5 (an toàn nhãn). Given decision trả `OTHER_PERSON` hoặc nhãn dùng công cụ cá nhân Then đi **cùng đường** như khi luật cho nhãn đó (công cụ lấy danh tính từ `trusted_context`, không từ decision; câu hỏi về người khác → từ chối + log như hiện hành).
- AC6 (độ trễ — cổng). Given cùng seed, cùng máy, provider giả có độ trễ thật (426 / 682 ms) When đo bật / tắt cổng (≥ 30 câu × 3 lượt, sinh chữ qua 9router) Then sự kiện SSE đầu ≤ 300 ms **không đổi**; thời gian tới token đầu p95 bật không xấu quá **20 %** so với tắt; vượt → giữ mặc định **tắt**, ghi số vào báo cáo, **không** nới ngưỡng. Ghi «trên MacBook Apple Silicon, Colima 6 CPU / 10 GB».
- AC7 (chất lượng — cổng). Given bộ 61 câu định tuyến (`benchmarks`, PoC C) + bộ diễn đạt lại giữ riêng Then độ chính xác bật ≥ **luật một mình** (61/61); nhãn khác luật chỉ được nhận khi `≥ τ`; số ca decision đổi nhãn đúng / sai báo cáo riêng.
- AC8 (nhánh lỗi). Given `GEMINI_API_KEY` trống / hết hạn mức / mạng lỗi Then chat vẫn trả lời bằng luật như tắt cổng, không treo, không thêm trễ vượt `timeout_ms`.
- AC9 (riêng tư). Given payload gửi decision Then chỉ **tin nhắn hiện tại đã che** (không lịch sử, không tên, không MSSV, không dữ liệu điểm); tối đa 2.000 ký tự; `llm_audit` không nội dung.

### Ngoài phạm vi của story này
- Escalate / độ tin cậy trả lời (chờ nhãn E2); dùng decision để viết lại câu hỏi hay xếp hạng truy xuất.

### Phụ thuộc
- US-DM-01, 02; D47 (đã sửa theo D60); `internal/agent/intent.go`, `internal/chat`; `TestOneEmbedPerMessage`.

---

## US-DM-06: Admin muốn cấu hình provider decision, công tắc và ngưỡng từng cổng, và bật theo từng lớp ngay trên `/settings/llm`
Ưu tiên: Must · Ước lượng: L · Sprint: 10.5 · **L5**

### Tiêu chí nghiệm thu
- AC1 (vị trí). Given Admin mở `/settings/llm` Then thêm **một** khối «Quyết định nhanh» ngay sau «Mô hình theo tác vụ» (trước «Mô hình tìm kiếm tài liệu»), cùng bố cục một cột, ngăn bằng khoảng trắng + đường kẻ 1 px; không khung lồng khung, không thẻ số liệu.
- AC2 (provider). Given khối Then một hàng provider: nhãn «Quyết định» · mô hình (`decision-fake` | `gemini-decision-1`) · trạng thái 1–2 từ («Giả», «Đã kết nối», «Chưa có khoá», «Tạm dừng») · `Test kết nối` (cục bộ). Chọn `gemini-decision-1` khi `LLM_DECISION_ALLOW_REAL=false` → báo tại chỗ «Chưa được phép dùng bản thật» (mã `DECISION_REAL_DISABLED`), không đổi. Khoá API chỉ ghi, sau lưu chỉ hiện trạng thái (như provider khác).
- AC3 (ba cổng). Given khối Then ba hàng, mỗi hàng: tên cổng · công tắc toàn hệ thống (mặc định tắt) · trạng thái («Tắt», «Bật ở N lớp») · `Cài đặt nâng cao` (ngưỡng, thời hạn; đơn vị và khoảng ở `placeholder`; tooltip chỉ nêu cú pháp / khoảng hợp lệ).
- AC4 (theo lớp). Given hàng cổng When bấm `Theo lớp` Then mở tại chỗ `DataTable` (lớp: tên + mã, công tắc lớp) có tìm theo tên / mã, phân trang con trỏ (`limit` ≤ 100); công tắc lớp **khoá** khi cổng tắt toàn hệ thống (tooltip trên công tắc: «Đang tắt toàn hệ thống» — lý do khoá, theo luật chữ UI).
- AC5 (hệ quả của lựa chọn). Given provider thật và cổng bật When bật công tắc Then tooltip tại chính công tắc nêu hệ quả một dòng: «Gửi nội dung đã che tới nhà cung cấp»; khi provider là bản giả **không** có tooltip này. Không thêm câu mô tả dưới tiêu đề khối, dưới hàng, không icon Info (luật chữ UI).
- AC6 (hành vi). Given đổi công tắc / ngưỡng Then cập nhật lạc quan + dòng «Hoàn tác» tĩnh lặng (UX.md), không hộp xác nhận, không toast «Thành công»; lỗi `409 VERSION_CONFLICT` → tải lại hàng + báo tại chỗ; `422` → báo tại ô; ghi `audit_log` (US-DM-02 AC6).
- AC7 (trạng thái). Given tải / rỗng / lỗi Then `<PageState>`; chưa có provider decision → hàng «Quyết định» «Chưa cấu hình» + `Thêm` (một hành động); lỗi API → `Thử lại`; không spinner riêng.
- AC8 (phân quyền). Given TEACHER / TA / STUDENT Then không thấy khối (route `/settings/llm` chỉ ADMIN như hiện hành), gọi API → `403`.
- AC9 (di động). Given 375 px Then ba hàng cổng xếp dọc, công tắc vùng chạm ≥ 44 px, bảng `Theo lớp` cuộn trong khung.
- AC10 (lời văn). Given quét luật chữ UI Then không từ «RAG / PII / fallback / trace»; tên cổng và trạng thái đúng bảng ở đầu file; `bash scripts/ui-antipatterns.sh` sạch; SRS `FEAT-llm-gateway` 7.3 vá thêm nhãn tác vụ `DECIDE` («Quyết định nhanh», làn «Trả lời ngay» / «Chạy nền») cùng đợt.

### Ngoài phạm vi của story này
- Cấu hình decision cho Teacher; sửa câu chỉ thị / câu hỏi của cổng (cố định trong mã).

### Phụ thuộc
- US-DM-01, 02; DESIGN §14.23; FEAT-llm-gateway 7.

---

## US-DM-07: Admin muốn thấy decision đang chạy ra sao ở `/observability`, và được báo khi cổng đóng-an-toàn nhiều bất thường
Ưu tiên: Must · Ước lượng: M · Sprint: 10.5 · **L5**

### Tiêu chí nghiệm thu
- AC1 (khối số liệu). Given Admin mở `/observability` Then thêm **một** bảng «Quyết định nhanh» dưới bảng 7 ngày, theo cổng × (số lời gọi, % đường nhanh, % dự phòng, % đóng-an-toàn, độ trễ p50 / p95 của lời gọi decision, lỗi %, chi phí ước tính VND); chọn cửa sổ 24 giờ / 7 ngày; số lấy từ `llm_audit` (`task='DECIDE'`, `cascade_gate`, `cascade_path`); **không nội dung**.
- AC2 (không lặp). Given dải «24 giờ qua» của US-P10-01 Then **không** thêm chỉ số decision vào dải; mỗi số đúng một chỗ.
- AC3 (trạng thái cổng). Given cùng bảng Then mỗi hàng ghi «Tắt» hoặc «Bật ở N/M lớp»; cổng tắt hiện `—` thay số.
- AC4 (cảnh báo). Given trong 1 giờ có ≥ 20 lời gọi của một cổng và tỉ lệ `FAIL_CLOSED` > 5 % Then một thông báo `DECISION_FAIL_CLOSED` cho mọi ADMIN (chuông), khử trùng 6 giờ, worker 5 phút (cùng cơ chế US-P10-03); liên kết `/observability`.
- AC5 (cầu dao / hết hạn mức). Given cầu dao `decision` mở hoặc `budget_blocked` ≥ 1 trong 1 giờ Then hàng provider trên `/settings/llm` hiện «Tạm dừng»; không thông báo riêng (tránh dội).
- AC6 (phân quyền). Given Teacher / TA / STUDENT Then không thấy bảng (`403` ở `GET /admin/observability/decision`); Teacher tổng hợp lớp **không** có số decision.
- AC7 (hiệu năng). Given `llm_audit` ≥ 1 triệu hàng Then API p95 ≤ 500 ms (chỉ mục `task, created_at`, `cascade_gate`); không `OFFSET`.
- AC8 (rỗng / lỗi). Given chưa có lời gọi nào Then `EmptyState` một dòng; lỗi API → `Thử lại`.
- AC9 (di động). Given 375 px Then bảng cuộn trong khung của nó, không tràn trang.

### Ngoài phạm vi của story này
- Xem nội dung đầu vào của decision (không lưu); biểu đồ chuỗi thời gian riêng.

### Phụ thuộc
- US-DM-01, 02; FEAT-p10-eval US-P10-01, 03 (`/observability`, cảnh báo).

---

## US-DM-08: Đội muốn đo trước / sau, hướng dẫn vận hành, seed và cổng nghiệm thu, để quyết định bật cổng bằng số liệu
Ưu tiên: Must · Ước lượng: L · Sprint: 10.5 · **L6**

### Tiêu chí nghiệm thu
- AC1 (seed). Given stack trống When `node scripts/seed.mjs` Then provider `decision-fake` có sẵn, ba hàng `decision_gates` (`enabled=false`, ngưỡng mặc định); `SEED_DECISION_DEMO=1` bật cả ba cổng cho lớp 1 để demo / E2E; `scripts/check-decision-seed.mjs` xanh; chạy lại idempotent.
- AC2 (cổng). Given repo When chạy:
  - `cd backend-go && go test ./internal/llm/... -run 'TestDecide|TestCascade' -v`
  - `cd backend-go && go test ./internal/privacy ./internal/agent ./internal/grading ./internal/chat -run 'TestPrecheckAddOnly|TestPlaceholderOnlyBlockedByRules|TestCrisisNeverReachesDecide|TestRouteDecisionNoOverride|TestDecideCallersAllowlist|TestInjectionOR|TestOneEmbedPerMessage' -v`
  - `pnpm -C frontend exec playwright test decision-settings.spec.ts decision-observability.spec.ts cascade.spec.ts`
  Then tất cả xanh (provider `fake`, không gọi mạng thật).
- AC3 (E2E `cascade.spec.ts`). Given provider giả When (a) decision trả lưỡng lự → Threads đi `Structured`; (b) decision lỗi → `Structured`; (c) cả hai lỗi → bài bị chặn; (d) chèn lệnh: decision lỗi → cờ theo regex + `Structured`, điểm không đổi; (e) chat: decision lỗi / dưới τ → trả lời như tắt cổng; (f) tin khủng hoảng → 0 lời gọi decision (đếm ở bộ đếm giả) Then đúng từng nhánh và đúng `cascade_path` ở `llm_audit`.
- AC4 (đo trước / sau — provider giả). Given `benchmarks/decision/run.sh` (Python chuẩn, chỉ HTTP, D46) When chạy ba kịch bản bật / tắt cùng seed, cùng máy: **(1)** chat 30 câu × 3 lượt (TTFT p50 / p95, SSE đầu); **(2)** đăng 40 bài Threads (20 sạch, 10 riêng tư tinh vi, 10 luật tự bắt): độ trễ đăng p50 / p95, tỉ lệ chặn đúng / nhầm, `FAST` / `FALLBACK` / `FAIL_CLOSED`; **(3)** chấm 13 bài (6 biến thể + 7 đối chứng): số cờ đúng / nhầm, độ trễ chấm Then ghi `benchmarks/reports/decision-before-after.{json,md}` + biểu đồ **SVG** (thư viện chuẩn); mỗi bảng ghi cấu hình máy «MacBook Apple Silicon, Colima 6 CPU / 10 GB», độ trễ giả (p50 / p95), lượt chạy, và mục «Giới hạn» (bản phát lại mô phỏng, không phải trễ mạng thật).
- AC5 (tiêu chí quyết định bật). Given báo cáo Then mỗi cổng có dòng kết luận **Bật được / Giữ tắt** theo luật cứng: chat — TTFT p95 ≤ +20 % **và** độ chính xác ≥ luật; Threads — chặn nhầm ≤ 0,05 **và** không lọt thêm so với luật; chèn lệnh — 6/6 **và** 0 cờ nhầm trên 7 đối chứng; cổng không đạt giữ **tắt**, ghi lý do.
- AC6 (lượt thật nhỏ — chỉ khi PM cho phép). Given PM cho phép bằng văn bản và `LLM_DECISION_ALLOW_REAL=true` When chạy lượt thật Then tối đa **300 lời gọi** (đề xuất 100 chat + 100 Threads + 100 chèn lệnh), trần ngày AC7 của US-DM-01 giữ; báo cáo thêm cột «thật» (nhà cung cấp, model, ngày, độ trễ từ Việt Nam, chi phí đo = token × giá niêm yết); chưa có phép → ghi «chờ phép PM», **không** chặn merge phần mã.
- AC7 (hướng dẫn vận hành). Given `docs/guides/decision-model.md` Then có các mục: bật / tắt cổng (toàn hệ thống, theo lớp), thứ tự bật thử đề xuất (một lớp → vài lớp → toàn bộ), chọn ngưỡng và hệ quả (ε / τ), đọc số ở `/observability`, xử lý sự cố (hết hạn mức, cầu dao mở, `FAIL_CLOSED` cao, tắt khẩn: một công tắc toàn hệ thống), chi phí (số PoC + công thức), Free Tier dùng dữ liệu (`PRODUCTION_READINESS.md` §1), câu lệnh thường dùng; kiểm: QC làm theo từng bước trên stack thử.
- AC8 (hợp đồng và tài liệu). Given `openapi.yaml` Then có mọi endpoint SRS 6; `go test ./internal/contract/...` xanh; `ARCHITECTURE.md` / `SYSTEM_DESIGN.md` / `DECISIONS.md` D60 khớp mã (đổi tên `FAIL_RULES` → `FAIL_CLOSED`, cột `cascade_gate`, bảng cấu hình, env `LLM_DECISION_*`); FLOWS F3, F4, F9 rà lại sau khi có mã.
- AC9 (nhánh lỗi). Given `benchmarks/decision/run.sh` khi gateway không chạy Then thoát mã ≠ 0 kèm «Gateway chưa chạy», không ghi báo cáo rỗng.

### Ngoài phạm vi của story này
- Bench 9.5 và sprint 11 đợt 1–5 (chạy tiếp sau khi 10.5 merge); điểm / rubric / escalate.

### Phụ thuộc
- US-DM-01…07; `benchmarks/pii` (E1), bộ 9.5, golden E2 (`benchmarks/e2`).

---

## US-DM-09: Chủ dự án muốn chắc chắn decision model không đụng vào điểm, khủng hoảng và quyền riêng tư, để dùng an toàn
Ưu tiên: Must · Ước lượng: S · Sprint: 10.5 · **xuyên suốt**

### Tiêu chí nghiệm thu
- AC1 (danh sách nơi gọi). Given `TestDecideCallersAllowlist` Then `Decide` / `Cascade` chỉ được gọi từ ba nơi: tường lửa Threads (`internal/privacy`), `flagInjectionDecision` (`internal/grading`), định tuyến ý định (`internal/agent`/`internal/chat`); **không** từ `internal/grade`, tính / làm tròn điểm, chọn mức rubric, công thức, escalate, đường khủng hoảng, `internal/rag` truy xuất.
- AC2 (khủng hoảng). Given tin chat / bài Threads bị `IntentCrisis` bắt Then **không** gửi tới decision, `Structured` hay LLM nào, không `llm_audit`, không log nội dung; `TestCrisisNeverReachesDecide` ở chat **và** Threads.
- AC3 (riêng tư). Given mọi lời gọi `Decide` Then payload đã che (`TestDecideMasked`), có trần độ dài theo cổng, **không** chứa `student_id`, tên, MSSV, email, điểm; `llm_audit` không nội dung; không bảng nào lưu đầu vào / đầu ra của decision.
- AC4 (Free Tier). Given gói miễn phí dùng dữ liệu gửi vào để cải thiện sản phẩm Then chỉ dữ liệu mô phỏng đã che được gửi thật; `PRODUCTION_READINESS.md` §1 ghi «triển khai thật dùng gói trả phí» và `LLM_DECISION_ALLOW_REAL` mặc định `false`; hướng dẫn vận hành nhắc điều này ở mục đầu.
- AC5 (không cổng duy nhất). Given mỗi cổng Then luôn có đường không phụ thuộc decision (luật / `Structured`); tắt decision hoàn toàn (provider tắt, cầu dao mở, khoá trống) → hệ thống hành xử **như trước sprint 10.5**, chỉ riêng Threads dùng `Structured` rồi chặn khi lỗi cả hai; kiểm bằng test «provider decision tắt → mọi cổng như cũ».
- AC6 (không thêm bên xử lý). Given khoá Then dùng `GEMINI_API_KEY` sẵn có; không nhà cung cấp, khoá, SDK, thư viện mới ngoài bảng `ARCHITECTURE.md`.
- AC7 (không ghi đè D47). Given đường hỏi–đáp Then số lời gọi mỗi câu: 1 sinh chữ + ≤ 1 nhúng + ≤ 1 decision (kiểm bằng bộ đếm giả ở `TestCallBudgetPerMessage`); không vòng lặp agent mở.

### Ngoài phạm vi của story này
- Chính sách giữ dữ liệu của nhà cung cấp (PR).

### Phụ thuộc
- US-DM-01…05; D47, D60; US-P3-04 AC13.

# DEV handoff — US-P3-04
Nhánh: `sprint/6-p3-p8`. Commit: `US-P3-04: …`.

## Đã làm
- **`privacy.Classifier`** (`internal/privacy/classify.go`): luật (PII qua `Detect`, 8 mẫu câu cá nhân tiếng Việt khớp trên chữ bỏ dấu nên phủ có dấu / không dấu) → chỉ khi luật chưa quyết và văn bản ≥ 20 ký tự mới nhúng **một** lần, so cosine với mẫu cá nhân; ≥ 0,78 → cá nhân, vùng giữa và thấp → công khai; nhúng lỗi → chỉ luật (`warn`). Không có bước LLM sinh chữ. Trả vectơ ra cho `rag` dùng lại. Mẫu: `internal/privacy/personal-exemplars.txt` (46 câu, `go:embed`; `seed/privacy/personal-exemplars.txt` là symlink tới đúng tệp đó để không có hai bản). `Detector.Entities` trả khoá thực thể + cờ roster.
- **`internal/agent`** (không import `net/http`, không gọi `privacy.Mask*`):
  - `intent.go`: `DetectIntent` bằng luật, thứ tự `CRISIS > WHAT_IF > GRADE_FORMULA > PERSONAL_* > EXAM_SCHEDULE > UPCOMING_EVENTS > LIBRARY_SEARCH > COURSE_QA > SMALLTALK`; `OTHER_PERSON` do `Analyzer` quyết (cần roster).
  - `analyze.go`: `Analyzer.Analyze` = phân loại **một lần** + intent + "hỏi hộ" (định danh khác mình: roster ≠ mình, MSSV / email lạ làm đối tượng — kể cả ngoài roster; câu **tự khai** "MSSV của em là …" bỏ qua nếu không thuộc ai, từ chối nếu thuộc roster khác).
  - `tools.go`: 7 tool, `PrivateRegistry` (`RegisterPersonal`, `RegisterShared`) và `PublicRegistry` (không có `RegisterPersonal`, chỉ `search_library`; tool lạ → `ErrToolNotRegistered`). Tool cá nhân chỉ nhận `TrustedContext`; kiểu tham số không có trường danh tính; `what_if` dùng `decimal`. Nguồn dữ liệu là giao diện (`AttendanceSource`, `ParticipationSource`, `GradeSource`, `GradeSchemeSource`, `ScheduleSource`, `LibrarySource`); chưa nối → `NoData`.
  - `agent.go`: `Agent.Respond` = phân loại → định tuyến → (câu mẫu | tool | truy xuất `rag.SearchStudent`) → **đúng một** `Generator.Stream`. Câu mẫu ở `replies_vi.go`. Sàn ngữ cảnh `RagSimFloor` 0,25 (bản tạm). Cache câu trả lời `ep:ans:{course}:{ver}:{sha}` (`AnswerCache`) chỉ cho COURSE_QA / LIBRARY_SEARCH không PII; intent cá nhân và tin có PII không đọc / ghi; `Outcome.CacheKey` để chat ghi sau khi sinh xong (phiên có `document_id` có khoá riêng).
  - `prompt.go`: system chỉ có quy tắc (giữ `[[…]]`, chỉ dùng `<ngữ_cảnh>`, trích `[n]`, ngữ cảnh là dữ liệu); ngữ cảnh nằm trong khối rào ở tin `user`, chuỗi mở / đóng rào trong dữ liệu bị vô hiệu.
  - `infra.go`: `NewEmbedder` (làn INTERACTIVE qua `llm`, cache `ep:emb:{sha256(chữ chuẩn hoá)}` TTL 10 phút, giá trị là vectơ), `StoreSelf` (`AgentSelf` trong `privacy.sql`), `StoreEvents` (`pii_events`), `NewRedisKV`.
- Ghi `pii_events` `BLOCKED` / `OTHER_PERSON` + một dòng log không tên khi từ chối; `CRISIS` không log nội dung, không ghi sự kiện, không báo ai.

## File đổi
`internal/privacy/{classify.go,classify_test.go,detect.go,roster.go,personal-exemplars.txt}`, `internal/agent/*` (mới), `internal/store/queries/privacy.sql` + sinh, `seed/privacy/personal-exemplars.txt` (symlink).

## Lệnh QC chạy để kiểm
```bash
cd backend-go; export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
go test -count=1 -race ./internal/agent -v        # gồm cache Redis
go test -count=1 ./internal/privacy -run 'TestClassify|TestExemplars' -v
go test ./internal/llm -run TestMaskOnlyInLLMGateway -v    # agent không gọi Mask
```

## Test đã chạy và kết quả
`go test -race ./internal/agent ./internal/privacy` xanh; `go vet`, `golangci-lint` sạch cho mã dev. Full suite: xem commit.

## AC tự đánh giá
AC1 ✓ ở `agent` (`TestClassifyOncePerMessage`; chat / thread thêm ở P3-05 / 06) · AC2 ✓ · AC3 ✓ (44 câu, 11 intent ≥ 3; `OTHER_PERSON` kiểm riêng ở AC8) · AC4 ✓ (9 nhánh 1 lần, 5 nhánh + no-context + dưới sàn 0 lần) · AC5 ✓ · AC6 ✗ ở story này (cần `internal/chat`: `TestTrustedContextFromJWT`, `TestBodyIdentityFieldsRejected` làm ở P3-05) · AC7 ✓ · AC8 ✓ (13 câu) · AC9 ✓ · AC10 ✓ · AC11 ✓ (nguồn thật nối ở P8-02 / P8-03) · AC12 ✓ (Redis thật) · AC13 ✓ (8 câu có dấu / không dấu) · AC14 ✓ · AC15 ✓ (tầng `agent`; chặn vai ở handler P3-05).

## Nợ / cần hỏi
1. AC6 và `TestClassifyOncePerMessage` ở `chat` / `thread` thuộc P3-05 / 06 (chưa có gói).
2. Phát hiện "hỏi hộ" chỉ thấy tên có trong roster của lớp (không NER, D46): tên người ngoài lớp lọt qua — ghi trung thực ở E1.
3. `what_if` tự phân tích số theo từ khoá "giữa kỳ / cuối kỳ / quá trình / bài tập" (id thành phần `midterm|final|process|assignment`); P6 ánh xạ sang thành phần thật khi nối `GradeSource`.
4. Thư viện (`search_library`) là giao diện `LibrarySource`; `LIBRARY_SEARCH` ở chat riêng dùng tool này, không phải truy xuất RAG.

## Vòng sửa 1 (PM, theo `qc/report-US-P3-04.md`)
- **BUG-1 — "Khi nào em thi cuối kỳ?" → `EXAM_SCHEDULE`.** Luật `reExam` (`internal/agent/intent.go`) không còn đòi các từ liền kề: cho phép đại từ / trợ từ chen giữa (`khi nào|bao giờ|lúc nào|ngày nào` + `em|mình|tôi|lớp|sẽ|có|được|phải|nhóm`* + `thi|kiểm tra`) và dạng đảo (`thi|kiểm tra` + `cuối kỳ|giữa kỳ|môn này|lần 2…`* + `khi nào|bao giờ|lúc nào|ngày nào`). `routeCases` thêm 4 câu tự nhiên ("Khi nào em thi cuối kỳ?", "Bao giờ mình thi giữa kỳ vậy ạ", "Thi cuối kỳ khi nào ạ", "Mình thi môn này lúc nào"); `TestQCRouteTable` hết báo câu này.
- **BUG-2 / BUG-3: chờ BA** — dev không đổi luật. Hệ quả: `TestQCRouteTable` (file của QC) còn đỏ đúng 2 câu ("giả sử giữa kỳ em được 9 thì sao", "cảm ơn bạn nhé") cho tới khi BA chốt; nếu BA chọn "chấp nhận cả hai" dev chỉ cần thêm `gia su` vào `reWhatIf` và nới ngưỡng 12 ký tự của `reGreet`.

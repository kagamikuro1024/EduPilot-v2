# DEV handoff — Sprint 10 (P9 Luyện đề), US-P9-01 … P9-12
Nhánh: `sprint/10-practice`. Commit cuối: xem `git log -1`. Spec: `docs/specs/FEAT-practice/{SRS,US,QUESTIONS}.md` (APPROVED v1.0); đề xuất đổi tài liệu nền P-10-1…6 đã áp (PM chấp nhận).

**Chưa chạy trên stack compose thật** (stack test nhường QC): chưa có số đo trên gateway thật, chưa chạy seed, chưa chạy `docling` / judge, chưa chạy k6, Lighthouse, axe. Mọi test dưới đây là Go (Postgres + Redis testcontainers) và Playwright với gateway giả; ngoài ra có một lượt thử tay bằng gateway native + frontend dev (luyện chủ đề, thi thử mất mạng / có mạng, nộp, 375 px).

## Đã làm (theo thứ tự lát dọc)
- **Quiz Engine** (`internal/quiz`): `SHORT` (`NormalizeShort`), `BuildMatrix` (ma trận chủ đề × độ khó, tránh câu đúng 30 ngày), `Shuffle` — code thuần, không import `internal/llm`. `exam.Validate` mở `SHORT` / `ESSAY` cho ngân hàng, vẫn chặn ở bài thi (`TYPE_NOT_SUPPORTED_IN_EXAM`).
- **Migration** `00019_practice` (`practice_attempts/items/answers`, `forms_import_batches/rows`); sqlc `practice.sql`, `forms.sql`.
- **`internal/practice`**: dựng đề (`start.go`), làm bài TOPIC (chấm ngay, khoá câu) / MOCK (`version` + `X-Practice-Tab`, đáp án chỉ lộ sau nộp, dung sai ngoại tuyến 60 s) (`attempt.go`), đóng lượt hết giờ (`job.go`, task `practice.close`), lịch sử / điểm yếu (`history.go`), giải thích SSE cache 24 h + nhận xét tự luận qua `grading.Feedback` (`llm.go`), trích đề / sinh câu có trích dẫn (`extract.go`, `generate.go`), duyệt hàng loạt (`bulk.go`), **nhập Forms** (`forms.go`).
- **REST**: 16 thao tác (#1–#16, `practice.Routes()`), `openapi.yaml`, contract test (304 thao tác), kịch bản mọi status.
- **Hôm nay**: `PRACTICE_WEAK_TOPIC` (bậc 85, STUDENT); outbox `practice.updated`.
- **Liêm chính**: `library.Service.Search` ẩn `EXAM_PAPER` khi sinh viên đang có lượt bài thi; `exam.Review` chặn duyệt câu EXTRACTED / GENERATED không trích dẫn (`CITATION_REQUIRED`).
- **Frontend**: `features/practice/*` (trang chủ, làm theo chủ đề, thi thử, kết quả, lịch sử), `/questions` (Trích từ đề, Sinh từ tài liệu, thanh duyệt hàng loạt, SHORT / ESSAY), **Sổ điểm → Nhập từ Forms** (`features/gradebook/FormsImportDialog.tsx`).
- **Seed**: `step19Practice` (`scripts/seed.mjs`, 82 APPROVED + 20 PENDING + Quiz 01 + ≈ 40 lượt) + `scripts/check-practice-seed.mjs`. **Chưa chạy.**
- **k6**: `benchmarks/load/practice-mock.js` (viết, chưa chạy).

## Nhập Forms (US-P9-11) — hành vi cần QC biết
- `uploads/presign` `purpose=forms_import` (`.xlsx` / `.csv` ≤ 5 MB, chỉ TEACHER). `POST …/forms-import/preview {upload_id}` → cột + 20 dòng đầu + gợi ý cột; **cùng đường đó với `mapping` + `item_id` + `answer_source`** → thêm bảng điểm đã chấm (AC5 "Xem trước kết quả"; không ghi). Cả hai đánh dấu tệp "đã xem trước" (Redis, 1 giờ); `POST …/forms-import` thiếu dấu → `409 PREVIEW_REQUIRED`.
- Khớp sinh viên: email **đã xác minh** hoặc MSSV của dòng `ROSTER` / `ADMIN` (MSSV `CODE` tự khai không mở dữ liệu). Cả email và MSSV cùng có mà trỏ hai người khác nhau → "Chưa khớp" `MISMATCH`. Trùng sinh viên → dòng có dấu thời gian mới nhất (không phân tích được dấu thời gian → dòng sau thắng).
- Chấm bằng `grade()` của `internal/practice` (= `quiz.Grade`), điểm = đúng × thang cột ÷ tổng, `decimal`, làm tròn 2. Đáp án không khớp lựa chọn nào → 0 và đếm "ô không khớp". Nhiều lựa chọn tách bằng `;` (cách Forms ghi).
- Ghi `grade_entries(source='QUIZ')` vào cột MANUAL đã chọn qua `GradeImportApply` (thêm tham số `source`; nhập XLSX truyền `IMPORT`), **không** đặt `published_at`; audit `forms_imports` (hash, số khớp / chưa khớp, không có email / MSSV). Cùng tệp (hash) vào cùng cột → trả lại lô cũ (`replayed: true`, vẫn `201`), không ghi gì.
- Test: `internal/practice/forms_test.go` (`TestFormsImportFlow`, `TestFormsMatchRules`, `TestFormsImportScoreDecimal`, `TestFormsImportNoLLM`, `TestFormsManualAnswerAndSemicolonCSV`, `TestFormsImportGuards`), kịch bản hợp đồng ở `scenarios_grade_test.go`, e2e `gradebook.spec.ts` ("forms import …").

## Lệnh QC chạy
```bash
cd backend-go && go vet ./... && go test -race ./...                 # gồm internal/practice, internal/contract (-tags testroutes cho contract)
cd backend-go && go test -tags testroutes ./internal/contract
pnpm -C frontend lint && pnpm -C frontend exec tsc --noEmit && bash scripts/ui-antipatterns.sh
pnpm -C frontend build:gate && pnpm -C frontend exec playwright test e2e/practice.spec.ts e2e/quiz-integrity.spec.ts e2e/questions-extract.spec.ts e2e/gradebook.spec.ts --project=desktop
node scripts/check-practice-seed.mjs          # sau khi `pnpm dev` + seed (STEPS=20)
k6 run -e BASE=https://localhost -e COURSE=<lớp> -e TOKENS="…" benchmarks/load/practice-mock.js
```

## Khác biệt so với spec / quyết định của dev (đề nghị PM / BA xác nhận)
1. **Lượt luyện chụp câu** (`practice_items.question` + `answer_key`) — SRS 5 chỉ ghi `answer_key`; thêm `question` để sửa câu giữa chừng không làm lệch lượt.
2. **Ngân hàng cho luyện đề** = câu `APPROVED`, chưa lưu trữ, và không thuộc bài thi chưa `PUBLISHED` (Q-P9-3).
3. **`practice.updated`** ghi `{user_id, student_id}` (không có `course_id` như FR-19) — chỉ để huỷ cache "Hôm nay" của sinh viên.
4. **FR-23** chỉ áp ở `library.Service.Search` (công cụ `search_library`); Chat và Threads đã khoá theo khoá bài thi sẵn có.
5. **Bulk "Loại"** hoãn 5 s để Hoàn tác; đóng tab trong 5 s thì lệnh Loại mất.
6. **Hôm nay "Ôn chủ đề"** ẩn 3 ngày sau khi luyện chủ đề đó nên seed không hiện ngay (API không lùi ngày được).
7. **Forms**: không có thao tác "gán tay" riêng cho dòng Chưa khớp (không có trong danh sách thao tác #15–#16): dòng chưa khớp hiện trong xem trước / kết quả (định danh đã che), giảng viên nhập tay ở ô điểm của Sổ điểm.
8. Sửa lỗi nhân đường: `MockRun` lặp vô hạn "Đang nộp…" khi máy chủ đã đóng lượt (`ATTEMPT_CLOSED`) mà hàng đợi còn câu chưa gửi — nay nộp lấy kết quả theo phần đã lưu (ca e2e "ATTEMPT_CLOSED").
9. `shared/shell/nav.ts`: `/practice` bỏ khỏi bảng "màn chưa có backend" (build `NEXT_PUBLIC_MOCK_SCREENS=0` hiện màn thật).

## AC tự đánh giá
| Story | Kết quả | Ghi chú |
| --- | --- | --- |
| P9-01 loại SHORT / ESSAY | ✓ | `TestGradeShort`, `exam.Validate` |
| P9-02 trích đề cũ | ✓ trên fake LLM | `TestExtractQuestions` (đề 24 câu); chưa chạy docling + LLM thật |
| P9-03 sinh có trích dẫn | ✓ trên fake LLM | `TestGenerateCitations`; E5 (tỷ lệ giữ nguyên / sửa / loại) cần chủ dự án duyệt 20 câu thật |
| P9-04 duyệt hàng loạt | ✓ | e2e `questions-extract.spec.ts` |
| P9-05 dựng đề | ✓ | `TestBuildMatrix`; "≤ 3 s" chưa đo trên stack thật |
| P9-06 luyện chủ đề | ✓ | `practice.spec.ts` |
| P9-07 thi thử | ✓ | `practice.spec.ts` (mất mạng, tải lại, `ATTEMPT_CLOSED`, `TAB_TAKEN_OVER`); k6 chưa chạy |
| P9-08 tự luận | ✓ | `grading.Feedback` không điểm, trích dẫn phải nguyên văn |
| P9-09 điểm yếu + Hôm nay | ✓ | `TestWrongAnswerSignalsNoIdentity` (k ≥ 3) |
| P9-10 liêm chính | ✓ | `TestPracticeNoAnswerLeak`, `TestPracticeNoExamLeak`, `TestLibraryToolHidesExamPaperWhileLocked`, `quiz-integrity.spec.ts` |
| P9-11 nhập Forms | ✓ AC1–AC12 (xem chú ý 7) | `forms_test.go` |
| P9-12 seed + cổng | ✗ chưa chạy | seed, `check-practice-seed.mjs`, k6, Lighthouse, axe, E5, "import một file Forms thật" (chủ dự án) |

## Nợ / cần hỏi
- Chạy seed + `check-practice-seed.mjs`, k6, Lighthouse 3 route, axe khi có stack; ghi số đo "MacBook Apple Silicon, Colima 6 CPU / 10 GB".
- Chủ dự án tự kiểm: duyệt 20 câu AI sinh (E5), nhập một tệp Forms thật (đặc biệt định dạng dấu thời gian của MS / Google Forms: dev hỗ trợ `YYYY-MM-DD HH:mm:ss`, `M/D/YY`, `M/D/YYYY`, `YYYY/MM/DD h:mm:ss A` — định dạng khác coi như không có dấu thời gian).
- `practice.updated` thiếu `course_id` (mục 3) — nếu PM muốn đúng FR-19 thì sửa SRS hoặc thêm trường.

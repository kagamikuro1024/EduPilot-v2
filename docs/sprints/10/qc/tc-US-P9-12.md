# QC test case — US-P9-12 Seed, cổng nghiệm thu, E2E, xoá mock
Nguồn: US-P9-12 + SRS §9, FLOWS F12. Đây là story «đóng phase»: QC chạy cả cổng một lần ở cuối.

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-12-01 | AC1 | Stack trống `-p ep-qc` | `node scripts/seed.mjs`; ghi thời gian | Mã 0 | GO |
| TC-12-02 | AC1 | Seed xong | `GET …/questions?review_status=APPROVED&limit=100` (lớp 1); `PENDING`; theo chủ đề / độ khó / `citations` | 80 `APPROVED`: ≥ 6 chủ đề, đủ EASY/MEDIUM/HARD, mọi câu có `citations` thật (tài liệu seed tồn tại, trang hợp lệ); 20 `PENDING` trong đó có `EXTRACTED` | API |
| TC-12-03 | AC1 | – | Kiểm 1 bài QUIZ (bài thi PE `PUBLISHED`); ≈ 40 lượt luyện; ≥ 3 SV sai cùng 2 chủ đề (`DB`: đếm theo `student_id` × chủ đề sai) | Đúng số lượng (±; ghi số thật); đủ cho k-ẩn danh | DB |
| TC-12-04 | AC1 | – | Lớp 2: `GET …/questions` | Ngân hàng nhỏ khác; **không** chứa câu của lớp 1 (so `id` và `stem`) | API |
| TC-12-05 | AC1 | – | `node scripts/check-practice-seed.mjs` | Xanh | GO |
| TC-12-06 | AC1 | Seed xong | Chạy lại `node scripts/seed.mjs`; đếm dòng `question_bank`, `practice_attempts`, `exams` trước/sau | Mã 0; số dòng không đổi (idempotent) | GO + DB |
| TC-12-07 | AC2 | repo | `cd backend-go && go test -race ./internal/quiz/...` | `ok` | GO |
| TC-12-08 | AC2 | – | `go test ./internal/practice -run 'TestExtractQuestions\|TestBuildMatrix\|TestPracticeNoAnswerLeak\|TestPracticeNoExamLeak\|TestWrongAnswerSignalsNoIdentity' -v` | Cả 5 PASS; xác nhận mỗi test **chạy thật** (không `t.Skip`: `-v` không có `SKIP`) | GO |
| TC-12-09 | AC2 | – | `pnpm -C frontend exec playwright test quiz-integrity.spec.ts practice.spec.ts` (cả bộ `playwright.config.ts`: một lần, `--workers=1`) | Xanh | E2E |
| TC-12-10 | AC3 | `practice.spec.ts` | Đọc spec + chạy | Có đủ: thi thử 20 câu; tải lại giữa chừng ở **câu 8**; tắt/bật mạng ở câu 12–14; nộp; kết quả so với đáp án **tính tay trong spec** (không đọc đáp án từ API rồi so lại với chính nó); luyện chủ đề 10 câu có phản hồi ngay; `ATTEMPT_CLOSED` khi hết giờ (bài vẫn chấm); STUDENT B không mở lượt A | E2E |
| TC-12-11 | AC3 | – | QC tự chạy kịch bản của TC-12-10 bằng spec riêng của QC (`qc-practice-flow.spec.ts`) với đáp án QC tính tay | Kết quả `{đúng}/{tổng}` khớp | E2E |
| TC-12-12 | AC4 | repo | `grep -rn "@/mock/practice" frontend/src \| wc -l`; `ls frontend/src/mock/practice.ts frontend/src/mock/questions.ts`; `bash scripts/ui-antipatterns.sh` | 0 import; `mock/practice.ts` không còn; `mock/questions.ts` không còn nếu không còn người dùng; antipatterns sạch (10 dòng cuối) | DIFF |
| TC-12-13 | AC4 | – | Mở `/practice*` và `/questions` trên stack thật, đối chiếu với dữ liệu seed | Không còn dữ liệu mô phỏng cứng (chủ đề / số liệu khác DB) | E2E |
| TC-12-14 | AC5 | – | `grep` openapi: mọi route SRS §6 (#1–#16); `go test ./internal/contract/...` | Có đủ endpoint mới (trừ #15–#16 nếu P9-11 không làm); contract test xanh (response khớp spec, kể cả lỗi `409/422`) | GO |
| TC-12-15 | AC6 | Stack thật | Đi trọn F12: staff trích đề → duyệt hàng loạt → SV luyện (TOPIC + MOCK) → lịch sử → «Chủ đề cần ôn» → Hôm nay `Ôn chủ đề` ; nhánh lỗi: trích lỗi (`OVERLOADED`), mất mạng khi nộp | Mỗi bước đạt; nhánh lỗi đúng chữ thông báo; QC chụp ảnh từng bước | E2E |
| TC-12-16 | AC6 | – | `docs/PROGRESS.md` | F12 ☑; P9 «xong» chỉ khi cổng đạt (QC đối chiếu sau khi chạy xong, không trước) | DIFF |
| TC-12-17 | AC7 | Bản build production `ep-qc` | Lighthouse (`@lhci` / `lighthouserc*.json`) cho `/practice`, `/practice/[id]` | LCP ≤ 2.500 ms (devtools), TBT ≤ 200 ms (mô phỏng), script ≤ 256 KB; trượt → có dòng trong `proposals.md`, **không** nới ngưỡng (kiểm diff cấu hình Lighthouse) | GO |
| TC-12-18 | AC8 | – | `grep` `ARCHITECTURE.md` §4 / §5; `DEMO_SCRIPT.md` | Số migration `00019_practice`, route thật; không còn `practice_exams`; `DEMO_SCRIPT` có bước luyện đề nếu PM đồng ý (ghi) | DIFF |
| TC-12-19 | Quy ước | – | `bash scripts/ui-antipatterns.sh`; axe (0 serious / critical) trên `/practice`, `/practice/[id]`, `/practice/history`, `/questions` | Sạch / 0 lỗi | E2E |
| TC-12-20 | Quy ước | `prefers-reduced-motion: reduce` | Mở các màn mới | Không animation (đồng hồ vẫn chạy); chữ số cố định | E2E |
| TC-12-21 | Quy ước | – | `goose down && goose up` trên DB seed | Thành công | DB |
| TC-12-22 | Cấm | diff | Tìm: sửa contract test / `expected_final_grades.csv`, nới assertion, `t.Skip`, secret, `float64` điểm, `fetch(` trần, thư viện mới | Không có | DIFF |

## Phân quyền
Phủ ở các story; ở đây TC-12-04 (cách ly lớp).

## Lịch sử sửa TC
- 2026-11-14 — PM quyết P-10-8 (F1–F7, Q-S10-27…31): F1: TC-12-22 — ngưỡng Hôm nay SV 6 → 7 được chấp nhận; kiểm commit test có số P-10-8. Q28: TC-12-02 ≥ 80 `APPROVED` đạt. F2: TC-12-12 chờ dev xoá `mock/questions.ts`. F4: TC-12-14 thêm kiểm `429` ở openapi cho `explain`, `extract`, `generate`.
(trống)

# Sprint 8: câu hỏi kỹ thuật cho Tech Lead

Dev / QC ghi câu hỏi ở đây (`docs/team/RESEARCH.md`): `## TL-<số> (<ai>, <ngày>): <tiêu đề>`, Tech Lead trả lời ngay dưới.

## TL-1 (PM, 2026-10-31): `proposals.md` #13 (ba test `internal/today/staff_test.go`) là hệ quả của nguồn việc mới hay nới test?

**TL trả lời:**

**Kết luận.** Số 165 và thứ tự kind là **hệ quả đúng** của nguồn việc mới. Số 7 cũng đúng là hệ quả, nhưng **chỉ khi không lớp nào có bản nháp**. Test ngân sách không dựng bản nháp, nên che mất một N+1 thật trong `GradeProvider`. Chấp nhận #13 **có điều kiện**: sửa N+1 và thêm ca có bản nháp vào `TestTodayQueryBudget` trong cùng commit.

Tính lại độc lập, không dựa vào số dev đưa:
- **`TestStaffTodayCountAndCap` 110 → 165: đúng.**
  - Test tạo 55 lớp. Giảng viên `TEACHER ACTIVE` ở cả 55 lớp, mỗi lớp 1 yêu cầu vào lớp, không lớp nào có quy chế hay công thức.
  - Theo US-P6-05 AC5 / SRS 4.5.3 ("mỗi lớp `ACTIVE` có `state ≠ CONFIRMED` một mục"), cả 55 lớp đều ở `NO_POLICY`, nên thêm 55 mục.
  - F14 (`FLOWS.md` dòng 139) liệt kê "công thức điểm chưa xác nhận" và "thiết lập lớp mới" là **hai** nguồn riêng, nên `COURSE_SETUP` không nuốt mục mới.
  - Tổng: 55 + 55 + 55 = 165. Trần 50 mục (`actions`) giữ nguyên.
- **`TestProviderOverdueAfter48h`: đúng.**
  - Bậc theo mã: `EMAIL_MISMATCH` 40 · `JOIN_REQUEST` 45 · `GRADE_SCHEME_UNCONFIRMED` 70 · `COURSE_SETUP` 80.
  - Mục mới không quá hạn, nên đứng giữa `EMAIL_MISMATCH`/`JOIN_REQUEST` và `COURSE_SETUP` ở **cả hai** khẳng định (49 giờ và 47 giờ).
  - Khẳng định chính của test ("quá hạn nổi lên trước mọi bậc") không đổi.
- **`TestTodayQueryBudget` 6 → 7: đúng cho ca không có bản nháp.**
  - Truy vấn thứ 7 là `GradeStatusFacts`: một câu `any($ids)` cho mọi lớp.
  - Tiền lệ: #28 sprint 7 cũng +1 cho một nguồn mới, hằng số, không N+1.
  - Không gộp được vào `TodayStaffPending`, vì nguồn khác bảng (`grade_schemes`, `documents`, `gradebooks`).
  - **Nhưng** `grade.StatusBatch` → `InspectDrafts` (`internal/grade/scheme.go:111-150`) có hai nhánh truy vấn thêm:
    - +3 câu theo lô khi có ≥ 1 bản nháp: `GradeSchemesByIDs`, `GradeComponentsOf`, `GradeRulesOf`.
    - **Mỗi bản nháp** thêm `citationVerifier`, tức `DocPages` (≥ 1 câu, lặp theo trang `DocChunks`) cộng `VerifyCitations` (1 câu). Đây là N+1 theo số lớp có bản nháp.
  - Mỗi lần cache "Hôm nay" hụt, provider còn đọc lại **toàn văn quy chế**. Việc này chiếm vào hạn 150 ms của provider (SRS 4.5.3).

**Số đo.** Chạy trên bản sao mã của dev ở `/tmp/tl1/backend-go`; worktree dev không bị đụng.
- Lệnh: `go test ./internal/today -count=1 -run 'TestStaffTodayCountAndCap|TestProviderOverdueAfter48h|TestTodayQueryBudget|TestTLDraftQueryCount' -v`.
- Ba test sửa theo đúng #13 đều `PASS`.
- `TestTLDraftQueryCount` là test tạm của Tech Lead: 5 lớp, giảng viên dạy cả 5, K lớp có `DRAFT` (quy chế `READY` một trang + một thành phần có trích dẫn), xoá Redis trước mỗi lần gọi.

| Lớp có bản nháp | `/me/today` | `…/{c1}/today` |
| --- | --- | --- |
| 0 | 6 | 7 |
| 1 | 11 | 12 |
| 2 | 13 | 12 |
| 5 | 19 | 12 |

Mỗi bản nháp thêm 2 truy vấn: đúng hình N+1. Ngân sách 7 hiện chỉ đúng với lớp `NO_POLICY` / `NOT_EXTRACTED` / `CONFIRMED`. Nó sai đúng ở hai trạng thái mà mục này sinh ra để nhắc: `DRAFT_UNCLEAR`, `DRAFT_READY`.

**Cách sửa (chọn một; không đổi AC):**
- **(a) Đề xuất.** Lưu kết quả bộ kiểm khi **ghi**, không tính lại khi **đọc**.
  - Ở cùng giao dịch lưu bản nháp (SRS 4.2.7) và `PUT …/grade-scheme` (4.4.2), ghi `unclear_count` (hoặc `unclear_codes text[]`) lên `grade_schemes`.
  - `GradeStatusFacts` đọc cột đó; Today và `GET …/status` không còn gọi `InspectDrafts`. Kết quả: 7 truy vấn hằng số, không đọc toàn văn.
  - Trích dẫn chỉ hỏng khi tài liệu nguồn đổi hoặc bị gỡ. Handler `course.policy_changed` (đã có trong 4.5.3) tính lại cho bản nháp của lớp đó.
  - Cần thêm một cột vào `00016`. `00016` chưa vào `main` nên vẫn sửa được theo luật 6, nhưng là đổi lược đồ của US-P6-01 nên phải qua `proposals.md`.
- **(b) Không đổi lược đồ.** Gộp phần tra trích dẫn của mọi bản nháp vào **một** câu SQL: `content_chunks` theo `(document_id, page_no) = any(…)` cộng `position(vn_fold…)` cho cả lô.
  - Kết quả: hằng số 7 + 3 khi có bản nháp. Vẫn đọc văn bản quy chế mỗi lần cache hụt.

**Điều kiện kèm #13 (không phải nới).**
- `TestTodayQueryBudget` thêm ca: "giảng viên 2 lớp, cả hai có `DRAFT` có trích dẫn". Giới hạn của ca này giữ đúng số của phương án đã chọn: (a) ≤ 7, (b) ≤ 10. Không dùng giới hạn rời theo số lớp.
- Chữ giải thích ở `:118` ghi "55 yêu cầu + 55 mục thiết lập + 55 mục công thức điểm".

**Rủi ro nếu duyệt #13 nguyên trạng:** "Hôm nay" của giảng viên nhiều lớp đang soạn công thức (đầu học kỳ) tốn 2 truy vấn + đọc toàn văn mỗi lớp mỗi lần cache hụt. Test xanh vẫn hứa "≤ 7".

Test tạm `tl_draft_test.go` chỉ có ở `/tmp/tl1`, không commit.

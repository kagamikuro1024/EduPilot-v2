# Sprint 5 — PE Thi hằng tuần (trắc nghiệm + lập trình C/C++)

Trạng thái: **chủ dự án chốt hướng 2026-10-03** (D54–D57); PM lập kế hoạch · Nhánh `sprint/5-pe`, worktree `../TA_Agent_v2-s5`, xếp chồng trên `sprint/4-p2`. Dev bắt đầu sau khi sprint 4 xong cổng P2; BA viết spec ngay (chạy cuốn chiếu).

## Mục tiêu
Trọn `docs/phases/PE.md`: giảng viên soạn đề (ngân hàng câu hỏi + bài code có bộ test), giao bài thi theo khung giờ; sinh viên làm trắc nghiệm hoặc code C/C++ có đồng hồ, tự lưu; máy chấm (Quiz Engine cho trắc nghiệm, sandbox cho code); điểm tự công bố khi bài đóng; liêm chính theo D56. Luồng mới **F19** đi được trọn. Cuối sprint: `gate PE`.

## Story (lát dọc)

| # | Story | Lát PE | Ước lượng | Phụ thuộc |
| --- | --- | --- | --- | --- |
| 0 | US-PU-06 (kỹ thuật, nợ #14 + #15 sprint 4): khung trang có chữ (h1 + mô tả) vẽ từ máy chủ trước khi có phiên, bớt client component ở layout → Lighthouse LCP ≤ 2,5 s và TBT ≤ 200 ms về `error` trên CI cho 7 route (TBT trung vị 12× trên máy ≤ 170 ms; cách đo ở `docs/sprints/4/techlead.md` TL-1) | PU | M | P2 |
| 1 | US-PE-01 Migration bài thi + ngân hàng câu hỏi (dạng cuối) + quyền theo lớp + Quiz Engine (`internal/quiz`: một / nhiều đáp án, đúng–sai, điểm `decimal`) | L1 | M | P2 |
| 2 | US-PE-02 Sandbox chấm code: container (theo research), `internal/judge`, hàng đợi `judge.submit`, kết quả AC/WA/TLE/MLE/RE/CE/OLE, test tấn công | L2 | L | 01, research |
| 3 | US-PE-03 `/questions`: soạn câu trắc nghiệm + bài code (đề, giới hạn, test mẫu / ẩn, nhập zip), chạy lời giải mẫu qua sandbox, duyệt; gợi ý nháp AI (BATCH, `Structured`) | L3 | L | 01, 02, P1 |
| 4 | US-PE-04 `/exams` (giảng viên): tạo bài thi từ ngân hàng, khung giờ + thời lượng, xáo trộn, xem trước, khoá sửa khi đã có người làm; Provider "Hôm nay" | L3, L4 | M | 01, 03 |
| 5 | US-PE-05 Làm bài trắc nghiệm `/exams/[id]/take`: đồng hồ máy chủ, xáo theo `attempt_id`, tự lưu, mất mạng không mất bài, hết giờ tự nộp, 375 px | L4 | M | 04 |
| 6 | US-PE-06 Làm bài code: soạn thảo (không thêm thư viện editor nếu chưa duyệt), chọn C/C++, `Chạy thử` test mẫu (giới hạn), `Nộp bài`, tiến độ chấm qua SSE | L4 | L | 02, 04 |
| 7 | US-PE-07 Liêm chính: `ep:exam_lock` + hàm kiểm cho chat (P3 dùng), log rời tab / dán, so độ giống mã sau khi đóng | L5 | M | 05, 06 |
| 8 | US-PE-08 Đóng bài → chấm xong → tự công bố; kết quả SV (không lộ test ẩn / đáp án ngoài phạm vi cho phép); `/exams/[id]/results` giảng viên (bảng, phân bố, câu sai nhiều, cặp nghi chép, CSV, sửa điểm có lý do), phúc khảo tối thiểu | L6 | L | 05–07 |
| 9 | US-PE-09 Seed bài thi mẫu (1 trắc nghiệm 10 câu, 1 code 2 bài × 5 test) + k6 `exam-submit.js` + cổng PE | – | M | 01–08 |

Spec: `docs/specs/FEAT-weekly-exam/` (BA). TC: `docs/sprints/5/qc/`.

## Quyết định PM tự chốt (đổi được)
- Bài code: nộp **nhiều lần trong giờ**, lần nộp cuối tính điểm (giống kỳ thi lập trình phổ biến; một lần nộp quá khắt với code). "Một lần nộp" của D56 áp cho **bài thi** (một lượt làm), không phải từng bài code. Nếu chủ dự án muốn chỉ một lần thì sửa spec.
- Sinh viên sau khi công bố thấy: điểm, đúng/sai từng câu trắc nghiệm + đáp án đúng, kết quả **test mẫu**; test ẩn chỉ thấy số đạt / tổng (không thấy input/expected).
- Trình soạn code giai đoạn này: `textarea` có số dòng + tab; thư viện editor (CodeMirror) chỉ thêm khi có góp ý được PM duyệt.
- Điểm bài thi lưu ở `exam_attempts`; sổ điểm (P6, sprint 8) sẽ nối vào.

## Rủi ro
- Sandbox trên colima macOS có thể cần quyền đặc biệt (cgroups v2, `--privileged`) → research PoC trước khi dev bắt đầu US-PE-02.
- Chấm dồn cuối giờ: hàng đợi riêng, không chung làn LLM; k6 kiểm p95 ≤ 60 s cho 60 bài / 5 phút.
- Rò đáp án qua API: test `TestNoAnswerLeak` cho mọi endpoint sinh viên.

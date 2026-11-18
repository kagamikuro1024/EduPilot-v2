### DEF-R10-01 — go test ./internal/contract không xanh (TC-12-14)
- Mức: Trung bình
- Sprint gốc: 10 · TC gốc: TC-12-14 · Hồi quy do sprint sau?: KHÔNG RÕ (7 test đỏ nằm ở thread/auth/results, không ở practice; nghi quá tải môi trường hoặc lỗi 500 của sprint 11/10.8)
- Môi trường: máy QC, Colima 10 GB chung với ~40 container ep-*, be0c0d96
- Bước tái hiện: 1. `cd backend-go && go test ./internal/contract/ -run 'TestStudentNeverSeesConfidence|TestChatThreadsContract' -v`  2. so với `go test ./internal/contract/ -run 'TestNoAnswerLeak|TestGolden_Auth'` (xanh).
- Kỳ vọng (TC-12-14): contract test xanh · Thực tế: 500 INTERNAL ở `POST /api/v1/courses/{id}/posts/{id}/report` (scenarios_thread_test.go:247) và `POST /api/v1/auth/verify-email` (scenarios_test.go:389); chạy cả gói: 7 test đỏ.
- Bằng chứng: evidence-10/logs/contract-fail.log, contract-fail2.log, go-tests.log · Nghi mã: backend-go/internal/contract/scenarios_thread_test.go:247, scenarios_test.go:389
- Chạy lại lần 2: trượt (khác endpoint mỗi lần ⇒ có thể chập chờn do tải)

Ghi nhận không phải lỗi sản phẩm: (1) check-practice-seed.mjs đỏ do dữ liệu seed lớp 1 bị ca QC đổi; (2) sập Postgres ep-r lúc 12:59 UTC do nhóm khác nạp T1 vào cùng container; (3) specs `qc-s10-*` gốc giả định lớp mặc định 761987 nhưng sv.kha nay thuộc nhiều lớp (đã khắc phục ở bản sao).

## Lượt 2026-11-18 (QC cặp 3, mã dc67bd93, ep-qc, lớp QD10A) — lỗi mới
| ID | Mức | TC | Mô tả / bằng chứng |
| --- | --- | --- | --- |
| DEF-R10S-01 | Trung bình | TC-05-04 | Ma trận ô thiếu: `BuildMatrix` (matrix.go) chỉ ghi `notes` khi cả ngân hàng hết câu; bù từ chủ đề khác thì `notes=[]` — US-P9-05 AC1 (P-10-9): «mỗi lần bù ghi vào notes». Tái hiện: matrix [{QC-r10-F4:6},{big:4}] ô F4 chỉ 3 câu → 10 câu, notes rỗng. |
| DEF-R10S-02 | Thấp | TC-10-16 | AC8: `question_id` lớp khác trong PUT answers phải `422 INVALID_QUESTION`; thực tế option của câu lớp 2 → `422 INVALID_OPTION_ID`, trường `question_id` → `422 VALIDATION_FAILED` (unknown). Không rò dữ liệu (GET câu lớp 2 → 403). Bất đồng mã lỗi giữa AC/SRS và mã — nếu chủ ý: PM→BA đổi AC. |
| DEF-R10S-03 | Thấp | TC-12-16 | docs/PROGRESS.md: F12 ☐, P9 «Dev xong, chờ QC» chưa cập nhật (PM). |
| DEF-R10S-04 | Thấp | TC-06-07 | Lượt TOPIC ở ≥1024 px không có «Danh sách câu» (chỉ MockRun dựng aside). AC3 nói danh sách số câu cho màn làm bài; TC dùng lượt TOPIC. Có thể là bất đồng spec/TC — PM→BA nếu AC chỉ áp thi thử. |
| DEF-R10S-05 | Trung bình | TC-06-10 | TOPIC mất mạng khi «Kiểm tra»: thông báo «Không kết nối được tới máy chủ. Chữ bạn đã nhập vẫn được giữ.» (không phải «Chưa gửi được»); khi có mạng KHÔNG tự gửi lại (DB practice_answers = 0 sau 25 s); phải bấm lại. Ảnh: /tmp/qc9s/s10-0610-offline.png. |
| DEF-R10S-06 | Thấp | TC-06-21 | Tải lại giữa lượt TOPIC: về đúng câu 6 nhưng lựa chọn chưa gửi ở câu 6 mất (nút Kiểm tra tắt) — TC kỳ vọng còn (nháp phía máy). |
| DEF-R10S-07 | Thấp | TC-06-24 | 375 px lượt TOPIC: không tràn, nút ≥44 px, nhưng thanh «Kiểm tra / Câu tiếp theo» không sticky/fixed (TC: thanh hành động cố định dưới). |
| DEF-R10S-08 | Thấp | TC-07-09 | Hàng chờ ngoại tuyến MOCK lưu ở localStorage (`ep:draft:…:practice:…`), IndexedDB 0 mục; US AC4 ghi IndexedDB. Chức năng đúng (thanh «Chưa gửi được 5 câu», gửi lại đúng thứ tự). |

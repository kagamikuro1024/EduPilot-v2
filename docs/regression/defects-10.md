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
| DEF-R10S-01 | ĐÓNG @186bfe7b | TC-05-04 | Ma trận ô thiếu: `BuildMatrix` (matrix.go) chỉ ghi `notes` khi cả ngân hàng hết câu; bù từ chủ đề khác thì `notes=[]` — US-P9-05 AC1 (P-10-9): «mỗi lần bù ghi vào notes». Tái hiện: matrix [{QC-r10-F4:6},{big:4}] ô F4 chỉ 3 câu → 10 câu, notes rỗng. |
| DEF-R10S-02 | ĐÓNG (P-11-63) | TC-10-16 | AC8: `question_id` lớp khác trong PUT answers phải `422 INVALID_QUESTION`; thực tế option của câu lớp 2 → `422 INVALID_OPTION_ID`, trường `question_id` → `422 VALIDATION_FAILED` (unknown). Không rò dữ liệu (GET câu lớp 2 → 403). Bất đồng mã lỗi giữa AC/SRS và mã — nếu chủ ý: PM→BA đổi AC. |
| DEF-R10S-03 | Thấp | TC-12-16 | docs/PROGRESS.md: F12 ☐, P9 «Dev xong, chờ QC» chưa cập nhật (PM). |
| DEF-R10S-04 | ĐÓNG @186bfe7b | TC-06-07 | Lượt TOPIC ở ≥1024 px không có «Danh sách câu» (chỉ MockRun dựng aside). AC3 nói danh sách số câu cho màn làm bài; TC dùng lượt TOPIC. Có thể là bất đồng spec/TC — PM→BA nếu AC chỉ áp thi thử. |
| DEF-R10S-05 | ĐÓNG @186bfe7b | TC-06-10 | TOPIC mất mạng khi «Kiểm tra»: thông báo «Không kết nối được tới máy chủ. Chữ bạn đã nhập vẫn được giữ.» (không phải «Chưa gửi được»); khi có mạng KHÔNG tự gửi lại (DB practice_answers = 0 sau 25 s); phải bấm lại. Ảnh: /tmp/qc9s/s10-0610-offline.png. |
| DEF-R10S-06 | ĐÓNG @186bfe7b | TC-06-21 | Tải lại giữa lượt TOPIC: về đúng câu 6 nhưng lựa chọn chưa gửi ở câu 6 mất (nút Kiểm tra tắt) — TC kỳ vọng còn (nháp phía máy). |
| DEF-R10S-07 | ĐÓNG @186bfe7b | TC-06-24 | 375 px lượt TOPIC: không tràn, nút ≥44 px, nhưng thanh «Kiểm tra / Câu tiếp theo» không sticky/fixed (TC: thanh hành động cố định dưới). |
| DEF-R10S-08 | ĐÓNG (P-11-62) | TC-07-09 | Hàng chờ ngoại tuyến MOCK lưu ở localStorage (`ep:draft:…:practice:…`), IndexedDB 0 mục; US AC4 ghi IndexedDB. Chức năng đúng (thanh «Chưa gửi được 5 câu», gửi lại đúng thứ tự). |
| DEF-R10S-09 | Thấp | TC-11-13 | US-P9-11 AC8 / Q-P9-7: tệp Forms gốc lưu tối đa 90 ngày — không có hạn: bucket MinIO không có lifecycle (`mc ilm`: «lifecycle configuration does not exist»), `forms_import_batches` không có cột hết hạn, không job xoá blob forms. Nếu chính sách giữ thuộc phase PR (Q-P9-7 «chính sách giữ chung thuộc PR») → PM→BA đổi AC/TC. |
| DEF-R10S-10 | Thấp | TC-07-21 | Mất mạng khi bấm Nộp bài (thi thử): chữ «Chưa gửi được. Bài vẫn giữ trên máy bạn.» thay vì «Chưa nộp được. Đang thử lại.»; chức năng đúng (tự nộp 1 lần khi có mạng, cùng Idempotency-Key). Ảnh /tmp/qc9s/s10-0721-offline.png. |
| DEF-R10S-11 | Thấp | TC-09-19 | `/practice/history` SV chưa có lượt: thiếu nút «Luyện ngay» (US-P9-09 AC; TC-09-19). |
| DEF-R10S-12 | Thấp | TC-02-14, TC-03-08 | Chưa có bằng chứng nào cho «OVERLOADED → retry đúng 3 lần → dead-letter → «Chưa trích được. Thử lại.»»: trên stack không gây được (provider 429 → chờ nhà cung cấp theo P-11-23; 10 khe LLM đầy → việc BATCH chờ 37 s rồi chạy, không OVERLOADED); handoff dev nêu mức Go `Worker.call` thử lại 2 s, 5 s «chưa có test riêng». Xin dev3 viết test Go cho nhánh này (extract + generate). |
| DEF-R10S-13 | Thấp | TC-10-09, TC-12-10 | `quiz-integrity.spec.ts` và `practice.spec.ts` chạy bằng mock (route.fulfill), không phủ các bước TC yêu cầu (thi thử 20 câu, tải lại ở câu 8, mạng tắt câu 12–14, so đáp án tính tay, STUDENT B; chiều «mở lại khi hết giờ»). |
| DEF-R10S-14 | Cao | TC-12-11 | Thi thử (MOCK) 20 câu: tải lại trang giữa chừng (ở câu 8) làm MẤT 7/12 đáp án chọn sau đó (practice_answers = {} lúc nộp; kết quả 10/20 thay vì 15/20). Không tải lại thì đạt 15/20 (kể cả tắt/bật mạng ở câu 12–14). Tái hiện ở nhịp bấm 1200 ms và 250 ms; log request: sau tải lại, bấm câu 9–11 không phát PUT answers; lúc nộp chỉ 5 PUT mới. Lựa chọn đã chọn không được tô lại sau tải lại. |
| DEF-R10S-15 | Trung bình | TC-02-20 | `extract` tài liệu > 60 trang: mã xếp việc (202) rồi cắt 60 trang + skipped PAGE_LIMIT; SRS FR-28 / US-P9-02 AC9 đòi `429 RATE_LIMITED {reason:pages}`, không tạo việc, không gọi LLM (đã tốn 15 lời gọi LLM cho 70 trang). Bằng chứng: job 01a1f007-bc0a-78aa-9300-e9ad9be7117c. PM/BA xác nhận lệch mã; dev đang sửa. |
| DEF-R10S-16 | Thấp | TC-03-09 | Dedupe câu sinh dùng `quiz.NormalizeShort` (NFC, gộp khoảng trắng, chữ thường, bỏ dấu câu cuối) — KHÔNG bỏ số thứ tự đầu câu: ở lớp sạch QD10C, X = «12. Thiết bị nào chuyển khung theo địa chỉ MAC?» (APPROVED) không chặn câu sinh «Thiết bị nào chuyển khung theo địa chỉ MAC?» (created, không DUPLICATE); hoa/thường + dấu cách + dấu câu cuối (REJECTED, PENDING) được chặn đúng. TC-03-09 đòi chuẩn hoá cả «số thứ tự đầu». |
| DEF-R10S-17 | Thấp | TC-03-05 | Câu sinh từ tài liệu `visible_to_students=false`, SV sau khi trả lời thấy «Nguồn: trang 1» (không bấm được, không lộ tên tài liệu) — Q-P9-9 chốt chữ «Nguồn trong tài liệu»; chuỗi này không có trong `frontend/src`. |

Đóng sau dựng lại ep-qc @4ac1638a (gateway + frontend, 2026-11-19 02:3x):
- DEF-R10S-14 ĐÓNG — nguyên nhân thật (PM, P-11-66): thi thử không lưu `current_index` nên tải lại về câu 1; QC đã hiểu sai là «mất đáp án» (spec bấm tiếp «Câu sau» từ câu 1 nên ghi đè sai câu). TC-12-11 chạy lại: sau tải lại ở câu 8 trang ở đúng câu 8, lựa chọn còn tô, kết quả «đúng 15/20», DB correct_count 15.
- DEF-R10S-15 ĐÓNG — đề 70 trang → 429 RATE_LIMITED {reason: pages}, jobs 16→16, llm_audit 61→61; đề đúng 60 trang → 202.

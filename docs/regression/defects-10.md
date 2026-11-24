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
| DEF-R10S-09 | ĐÓNG @55664e51 (a9924f2b) | TC-11-13 | US-P9-11 AC8 / Q-P9-7: tệp Forms gốc lưu tối đa 90 ngày — không có hạn: bucket MinIO không có lifecycle (`mc ilm`: «lifecycle configuration does not exist»), `forms_import_batches` không có cột hết hạn, không job xoá blob forms. Nếu chính sách giữ thuộc phase PR (Q-P9-7 «chính sách giữ chung thuộc PR») → PM→BA đổi AC/TC. |
| DEF-R10S-10 | Thấp | TC-07-21 | Mất mạng khi bấm Nộp bài (thi thử): chữ «Chưa gửi được. Bài vẫn giữ trên máy bạn.» thay vì «Chưa nộp được. Đang thử lại.»; chức năng đúng (tự nộp 1 lần khi có mạng, cùng Idempotency-Key). Ảnh /tmp/qc9s/s10-0721-offline.png. |
| DEF-R10S-11 | Thấp | TC-09-19 | `/practice/history` SV chưa có lượt: thiếu nút «Luyện ngay» (US-P9-09 AC; TC-09-19). |
| DEF-R10S-12 | Thấp | TC-02-14, TC-03-08 | Chưa có bằng chứng nào cho «OVERLOADED → retry đúng 3 lần → dead-letter → «Chưa trích được. Thử lại.»»: trên stack không gây được (provider 429 → chờ nhà cung cấp theo P-11-23; 10 khe LLM đầy → việc BATCH chờ 37 s rồi chạy, không OVERLOADED); handoff dev nêu mức Go `Worker.call` thử lại 2 s, 5 s «chưa có test riêng». Xin dev3 viết test Go cho nhánh này (extract + generate). |
| DEF-R10S-13 | ĐÓNG @44991583+deac438b (QC xác nhận qua gate-pe @64ba1549) | TC-10-09, TC-12-10 | `quiz-integrity.spec.ts` và `practice.spec.ts` chạy bằng mock (route.fulfill), không phủ các bước TC yêu cầu (thi thử 20 câu, tải lại ở câu 8, mạng tắt câu 12–14, so đáp án tính tay, STUDENT B; chiều «mở lại khi hết giờ»). |
| DEF-R10S-14 | Cao | TC-12-11 | Thi thử (MOCK) 20 câu: tải lại trang giữa chừng (ở câu 8) làm MẤT 7/12 đáp án chọn sau đó (practice_answers = {} lúc nộp; kết quả 10/20 thay vì 15/20). Không tải lại thì đạt 15/20 (kể cả tắt/bật mạng ở câu 12–14). Tái hiện ở nhịp bấm 1200 ms và 250 ms; log request: sau tải lại, bấm câu 9–11 không phát PUT answers; lúc nộp chỉ 5 PUT mới. Lựa chọn đã chọn không được tô lại sau tải lại. |
| DEF-R10S-15 | Trung bình | TC-02-20 | `extract` tài liệu > 60 trang: mã xếp việc (202) rồi cắt 60 trang + skipped PAGE_LIMIT; SRS FR-28 / US-P9-02 AC9 đòi `429 RATE_LIMITED {reason:pages}`, không tạo việc, không gọi LLM (đã tốn 15 lời gọi LLM cho 70 trang). Bằng chứng: job 01a1f007-bc0a-78aa-9300-e9ad9be7117c. PM/BA xác nhận lệch mã; dev đang sửa. |
| DEF-R10S-16 | ĐÓNG @424b1f77 (QC xác nhận @76c33c0e) | TC-03-09 | Dedupe câu sinh dùng `quiz.NormalizeShort` (NFC, gộp khoảng trắng, chữ thường, bỏ dấu câu cuối) — KHÔNG bỏ số thứ tự đầu câu: ở lớp sạch QD10C, X = «12. Thiết bị nào chuyển khung theo địa chỉ MAC?» (APPROVED) không chặn câu sinh «Thiết bị nào chuyển khung theo địa chỉ MAC?» (created, không DUPLICATE); hoa/thường + dấu cách + dấu câu cuối (REJECTED, PENDING) được chặn đúng. TC-03-09 đòi chuẩn hoá cả «số thứ tự đầu». |
| DEF-R10S-17 | Thấp | TC-03-05 | Câu sinh từ tài liệu `visible_to_students=false`, SV sau khi trả lời thấy «Nguồn: trang 1» (không bấm được, không lộ tên tài liệu) — Q-P9-9 chốt chữ «Nguồn trong tài liệu»; chuỗi này không có trong `frontend/src`. |

Đóng sau dựng lại ep-qc @4ac1638a (gateway + frontend, 2026-11-19 02:3x):
- DEF-R10S-14 ĐÓNG — nguyên nhân thật (PM, P-11-66): thi thử không lưu `current_index` nên tải lại về câu 1; QC đã hiểu sai là «mất đáp án» (spec bấm tiếp «Câu sau» từ câu 1 nên ghi đè sai câu). TC-12-11 chạy lại: sau tải lại ở câu 8 trang ở đúng câu 8, lựa chọn còn tô, kết quả «đúng 15/20», DB correct_count 15.
- DEF-R10S-15 ĐÓNG — đề 70 trang → 429 RATE_LIMITED {reason: pages}, jobs 16→16, llm_audit 61→61; đề đúng 60 trang → 202.
| DEF-R10S-18 | Thấp | TC-12-13 | `/questions` (GV) tiêu đề «N câu chờ duyệt» đếm theo hàng đã tải (`QuestionBank.tsx:124` `rows.filter(PENDING)`, trang đầu 25 hàng) — DB có 43 PENDING nhưng hiện «25». |
| DEF-R10S-19 | Thấp | TC-07-25 | `benchmarks/load/practice-mock.js` gửi `{answer:{text:'x'}}` cho câu không có phương án, gồm cả TRUE_FALSE → 422; với ngân hàng có TRUE_FALSE: 196/2.000 PUT lỗi, `http_req_failed` 8,9% vượt ngưỡng `<0,5%`. Cần nhánh `{value:true}` cho TRUE_FALSE (bản vá của QC ở evidence-9s/s10/perf/). |

Mã xoá (LT có nguồn quyết định):
- TC-04-08: xoá — nguồn handoff `docs/sprints/10/handoff/dev-sprint10.md` L50 («P9-03 sinh có trích dẫn ✓») + điều kiện trong `tc-US-P9-03.md` dòng 2 («chỉ chạy nếu handoff ghi đã làm»); PM xác nhận 2026-11-19.
- TC-11-18: xoá — nguồn handoff `docs/sprints/10/handoff/dev-sprint10.md` L17 (mục «Nhập Forms (US-P9-11)») + điều kiện trong `tc-US-P9-11.md` dòng 2; PM xác nhận 2026-11-19.
| DEF-R10S-20 | Cao | TC-12-17 | Hiệu năng tải: LCP (Lighthouse devtools) `/practice` 4.086 ms và `/practice/[id]` (lượt TOPIC đang làm) 17.723 ms, vượt ngưỡng 2.500 ms (TBT/script đạt). Phần tử LCP của `/practice/[id]` là đoạn chữ hiện sau khi dữ liệu lượt về; cùng môi trường `/chat` 2.031 ms. Số liệu `evidence-9s/s10/perf/lh-1217-summary.txt`. Theo TC: trượt → PM ghi dòng proposals.md, không nới ngưỡng. |
| DEF-R10S-21 | Thấp | TC-02-18 | Chat riêng 1 QPS: TTFT p95 nền 21,6–44,6 ms (3 lượt) → 197,4 ms khi một việc `extract` đề 60 trang chạy nền (stub chậm 8 s/lời gọi), 1/60 yêu cầu lỗi; ngưỡng TC «xấu đi ≤ 20%». Kịch bản: `evidence-9s/s10/perf/chat-ttft.js` + `run0218.py`. |

## Lượt 2026-11-22 (QC cặp 3, qc2seedt3 ở 49ebd12a, seed lần đầu) — lỗi
| ID | Mức | TC | Mô tả / bằng chứng |
| --- | --- | --- | --- |
| DEF-R10S-07 | RÚT (P-10-11) | TC-12-02 | Seed lớp 761987: 107 câu APPROVED nhưng chỉ 3 có `citations` (104 câu `citations=[]`); TC: mọi câu APPROVED có citations thật (tài liệu seed tồn tại, trang hợp lệ). `SELECT count(*) FILTER (WHERE jsonb_array_length(citations)>0) FROM question_bank WHERE review_status='APPROVED' AND course_id=<761987>` = 3. |
| DEF-R10S-08 | RÚT (P-10-11) | TC-12-04 | Câu «Mô tả nào đúng về MFA?» có ở cả lớp 761987 và 761988 (cùng stem, APPROVED, id khác); TC: ngân hàng lớp 2 không chứa câu của lớp 1 (so id và stem). |
| DEF-R10S-22 | Thấp | TC-09-05, TC-12-15 | `/practice/history`: khối chủ đề yếu hiện tiêu đề «Chủ đề bạn hay sai» (`PracticeHistory.tsx:33`), trong khi US-P9-09 AC2 và SRS bảng route (`/practice/history`) ghi «Chủ đề cần ôn». Lượt cũ của TC-09-05 dò chữ «Chủ đề cần ôn» (không có trên màn) nên phép «ẩn khi < 10 câu» vô nghĩa; đã dò lại bằng chữ thật (xem TC-12-15 bước 3b / 5) | MỞ — PM/BA chốt chữ (sửa spec theo UI hoặc UI theo spec) |

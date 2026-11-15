# Smoke hồi quy F1–F19 — đợt R (main `6b98a36e`, stack sạch ep-qc)

Môi trường: ep-qc dựng lại từ DB trống (`down -v` chỉ ep-qc) → `seed` (`SEED_GUIDANCE_DEMO=1 SEED_DECISION_DEMO=1`, 9 p 39 s). LLM sinh chữ qua 9router (`ag/gemini-3.8-flash`); decision giả; embedding giả, **riêng lượt F3/F4/F6 dùng embedding thật** (xem `dot-r.md`, cuối). Kiểm bằng script API trên stack thật (`scripts/w*.py`, kết quả từng TC ở `scripts/results-r.jsonl`) + Playwright UI (audit 4 vai × 1440/375) + bộ e2e dev.
Kết luận chỉ PASS / FAIL; mục không chạy được ghi rõ lý do.

| Luồng | Đường chính | Nhánh lỗi kiểm | Vai | Kết quả |
| --- | --- | --- | --- | --- |
| F1 Tài khoản | đăng ký → mail → xác minh → đăng nhập; quên MK → đặt lại | link dùng lại 410; 6 lần sai → 429; phiên cũ bị thu hồi; MSSV mạo danh không có lớp | SV, ADMIN | **PASS** 14/14 |
| F2 Mở lớp / tham gia | ADMIN mở lớp → mã → SV xem trước → vào lớp ACTIVE | mã sai 404; lưu trữ → mã tắt 404 | ADMIN, GV, SV | **PASS** 9/9 |
| F3 Chat riêng | hỏi → stream → trích nguồn (embedding thật: 2–3 trích dẫn, sự kiện đầu 26 ms); tải lại phiên; feedback; xoá | ngoài tài liệu; hỏi người khác bị từ chối; khủng hoảng 0 lời gọi LLM | SV | **PASS** (D-R-07 TB: từ chối của LLM không kèm `low_confidence`/ticket) |
| F4 Threads | tường lửa PII → đăng → bình luận → báo cáo → ẩn/bỏ ẩn → xoá; AI trả lời có 2 trích dẫn, chờ GV xác nhận | đăng PII 422; báo cáo trùng; ẩn thiếu `version` 409 | SV, GV, TA | **PASS** (D-R-01 TB: «Em nghĩ …» bị chặn nhầm) |
| F5 Escalation | SV nhờ GV → GV nhận → trả lời → mail | TA nhận sau GV 409; SV khác 403; giới hạn 5 ticket 409 | SV, GV, TA | **PASS** |
| F6 Tài liệu | PDF → presign → MinIO → READY (docling) → chunk; truy xuất sau tải; xoá → hết truy xuất | trùng 409 `DOCUMENT_DUPLICATE`; sai loại / quá 50 MB 422; SV 403; ANSWER_KEY không lộ qua Thư viện/chat | GV, SV | **PASS** (D-R-05 TB UI bảng Tài liệu; D-R-08 Thấp reindex cả lớp với khoá Gemini miễn phí) |
| F7 Điểm danh | chốt buổi (3 vắng) → TA sửa khoá lạc quan → phát biểu | `client_event_id` lặp không ghi đôi; SV 403 ghi/đọc người khác | GV, TA, SV | **PASS** 8/8 |
| F8 Theo dõi SV | «Cần chú ý» → hồ sơ → ghi chú → nhận xét AI (9router 8,7 s) → nhắn riêng (ticket) | SV 403 hồ sơ/ghi chú; xem hồ sơ ghi `audit_log` | GV, SV | **PASS** 7/7 |
| F9 Bài tập | tạo (rubric, muộn) → công bố → SV nộp PDF → chấm 9router (10 s/2 bài) → duyệt → công bố → SV thấy điểm | nộp muộn `late_days`; hết hạn 409; sai loại 422; SV không thấy điểm nháp; công bố bài chưa duyệt 422 | GV, SV | **PASS** 19/19 (D-R-03, D-R-06 Thấp: bài ngắn bị coi «trống») |
| F10 Sổ điểm | công thức CONFIRMED → sổ → what-if (code) → chốt → snapshot → mở khoá có lý do → XLSX | TA không chốt 403; sổ đã chốt sửa điểm 409; điểm vượt thang 422; lớp chưa có quy chế: không có cột | GV, TA, SV | **PASS** |
| F11 Phúc khảo | SV gửi → GV quyết định + phản hồi → mail | một lần/bài 409; thiếu phản hồi 422; bài chưa công bố 404 | SV, GV | **PASS** 7/7 |
| F12 Luyện đề | luyện chủ đề 5 câu → tự lưu → nộp → giải thích (9router 5 s) → điểm yếu → lịch sử; đề thử có `deadline_at` | không lộ đáp án trong lúc làm; SV khác/GV 403; SV chưa vào lớp 403; nộp lại idempotent | SV | **PASS** 12/13; «báo câu sai» (R-F12-08) **chưa kiểm được** (payload `feedback` không rõ, trả 422) |
| F13 Lịch | sự kiện EXAM → SV thấy → ICS token | SV tạo 403 | GV, SV | **PASS** 4/4 |
| F14 Hôm nay | `/me/today` | đủ 4 vai 200 | 4 vai | **PASS** 6/6 |
| F15 LLM / ngân sách / tích hợp | xem provider (khoá không lộ), test 9router, tuyến đủ 7 tác vụ + embedding, ngân sách ngày/tháng, IMAP/Teams | sửa đồng thời 409; GV sửa tuyến 403; SV 403 | ADMIN, GV, SV | **PASS** 11/11 |
| F16 Quan sát AI | — | GV/TA/SV 403; ADMIN xem số liệu Decision | ADMIN, GV, TA, SV | Phân quyền **PASS** 4/4. Màn `/observability` là **bản mô phỏng chưa nối API** (mã nguồn: «sprint 11, M13»): **chưa phát hành** — không tính FAIL |
| F17 Lỗ hổng kiến thức | — | — | — | `/insights` chưa nối API (mã nguồn: «sprint 11, M13»): **chưa phát hành** — không kiểm được |
| F18 Kết thúc học kỳ | lưu trữ lớp | lớp lưu trữ chỉ-đọc (tạo bài tập, đăng Threads 409 `COURSE_ARCHIVED`); SV vẫn xem điểm / thư viện; SV 403 lưu trữ | ADMIN, GV, SV | Lưu trữ **PASS** 6/6. Xuất ZIP + nhân bản sang học kỳ mới **chưa có API**: chưa phát hành |
| F19 Thi hằng tuần | trắc nghiệm: tạo → chọn câu → xem trước → lên lịch → mở giờ → làm (tự lưu, sự kiện tab/dán) → nộp → hết giờ tự đóng + công bố → phúc khảo → GV trả lời → CSV. Code C: chạy thử AC, vòng lặp vô hạn TLE, nộp, điểm 10/10 | chưa tới giờ 409; chat khoá khi đang thi 409 `EXAM_IN_PROGRESS` + `exam-lock`; chưa công bố 409; SV khác 404; SV xem bảng lớp 403 | GV, SV | **PASS** 37/37 (D-R-04 Thấp) |

## UI 4 vai × 1440 / 375 (`audit-r/`)
`qc-ui-audit-real.spec.ts` (khung 10.7): mọi route × SV, TA, GV, ADMIN × 1440 + 375, trạng thái «dữ liệu» — **8/8 lượt chạy**, 108 trang không lỗi tải. Lệch 5: E2 chat SV (đã N/A từ 10.7) + 4 D3 (`/documents`, `/grading` × GV, TA) → D-R-05 (TB), D-R-06 (Thấp). Ảnh: `audit-r/*.jpg`.
`ui-antipatterns.sh` sạch, `--selftest` 31/31.

## check-*-seed
Đạt: assignments-seed, assignments-v2, chat-seed (lúc dựng), decision-seed, grade-seed, guidance-seed, knowledge-seed, practice-seed. Đỏ: `check-docs-seed` (ANSWER_KEY = 2), `check-exam-seed bank|demo` → D-R-02 (Thấp, script lệch seed hiện hành). `check-crm-seed` (tickets 5→6) và `check-chat-seed` (160 ≠ 150±5) đỏ **do QC ghi thêm dữ liệu sau khi seed** (ticket, tin chat) — không tính lỗi.

## Không kiểm được
- F16/F17 end-to-end, F18 xuất ZIP + nhân bản: chưa có API/chưa nối (xem trên).
- Hạn chế: lượt chạy trước embedding thật dùng embedding giả → đường «có nguồn» chỉ kiểm ở lượt embedding thật (đã kiểm).
- R-F12-08 (báo câu hỏi sai).

# QC test case — US-NFR-02 Seed demo (T0) chạy qua sạch trên Mac
Nguồn: US-NFR-02 + SRS FR-4, §3, §9. Stack trống `-p ep-qc` (`SEED_ON_EMPTY_DB=false` để tự chạy seed bằng tay; `SEED_DEFAULT_PASSWORD` đặt). Bảng «chính» để đếm dòng (Q-N-5): `users, courses, enrollments, documents, content_chunks, question_bank, exams, assignments, submissions, grade_items, grade_entries, attendance_*, chat_sessions, chat_messages, threads, posts`.

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-N02-01 | AC1 | Stack trống, chưa seed | `time node scripts/seed.mjs; echo $?` | Mã 0; thời gian thật ghi lại (báo cáo của dev ghi thời gian chạy; Q-N-2) | RUN |
| TC-N02-02 | AC1 | Seed xong | `for f in scripts/check-{assignments,chat,crm,docs,exam,grade,knowledge}-seed.mjs; do node $f; echo "$f $?"; done` | 7/7 mã 0 (tệp nào không tồn tại → ghi, dev xác nhận tên) | RUN |
| TC-N02-03 | AC2 | Seed xong lần 1; đếm dòng `N1[bảng]` | Chạy lại `node scripts/seed.mjs`; đếm `N2[bảng]` | Mã 0; `N1 == N2` mọi bảng chính (và `SELECT count(*)` mọi bảng `public` không tăng); không bản ghi trùng (`SELECT … GROUP BY khoá nghiệp vụ HAVING count>1` rỗng: email, mã lớp, `(course_id,title)` tài liệu, `(course_id,stem)` câu hỏi) | RUN |
| TC-N02-04 | AC2 | – | Chạy lần 3, đồng thời lấy `docker compose logs gateway --since <t>` | Không có `409`/`23505` bị script nuốt không giải thích: log gateway của lượt chạy lại cho thấy số `409` bằng số «đã có» mà script in; không `23505` ở log Postgres (Q-N-3) | RUN |
| TC-N02-05 | AC2 | – | Sửa tay một dòng (đổi tên SV A) rồi chạy lại seed | Không ghi đè vô cớ / không tạo bản trùng; hành vi ghi lại (seed idempotent theo khoá, không «reset») | RUN |
| TC-N02-06 | AC3 | Seed xong, `SEED_DEFAULT_PASSWORD` | Đăng nhập 7 tài khoản: SV A, B, C (`sv.kha`, …: Q-N-4), TA, GV, Admin; mỗi tài khoản `GET /courses/{id}/today` (hoặc `/me/today`) | Đăng nhập được cả 4 vai bằng mật khẩu từ env, sai mật khẩu → `401`; `Hôm nay` mỗi vai có **≥ 1 việc** đúng nguồn theo bảng F14 (QC lập bảng đối chiếu vai → nguồn từ `docs/FLOWS.md` F14: kiểm `kind` mỗi việc thuộc nguồn hợp lệ của vai đó) | RUN |
| TC-N02-07 | AC3 | – | `GET …/students` lớp 1; scheme `GET …/grade-scheme` (hoặc API công thức) của lớp 1 và 2 | Lớp 1: 30 SV; công thức lớp 2 **thiếu bước làm tròn**, lớp 1 có; hai lớp khác nhau | RUN |
| TC-N02-08 | AC4 | Seed xong | Đọc `docs/DEMO_SCRIPT.md` bước 2–7; với mỗi dữ kiện kịch bản cần: BT03 có **22** bài chưa chấm (`GET …/assignments/{BT03}/submissions?status=…` đếm 22); SV A, B, C có điểm **tính tay được** (QC tính tay từ thành phần điểm A/B/C theo công thức lớp rồi so `GET …/gradebook`/giải trình; khớp); câu chat mẫu của kịch bản gửi chat (provider `fake`) trả câu trả lời **có trích dẫn tài liệu** (nguồn trong phản hồi SSE) | Mọi dữ kiện có; mục nào thiếu → ghi trong «danh sách bổ sung» của handoff (nếu handoff ghi thiếu mà không có trong seed = FAIL) | RUN |
| TC-N02-09 | AC5 | Stack lên, **chưa** đặt `SEED_DEFAULT_PASSWORD` (`env -u SEED_DEFAULT_PASSWORD`); đếm dòng trước | `node scripts/seed.mjs; echo $?` | Mã ≠ 0; **một dòng** tiếng Việt, không stack trace; số dòng mọi bảng không đổi (không ghi gì) | RUN |
| TC-N02-10 | AC5 | `SEED_DEFAULT_PASSWORD` đặt, stack **tắt** | `node scripts/seed.mjs; echo $?` | Mã ≠ 0, một dòng thông báo tiếng Việt ngay (≤ 10 s, không treo/thử lại vô hạn) | RUN |
| TC-N02-11 | AC5 | `SEED_DEFAULT_PASSWORD=""` (rỗng) và toàn khoảng trắng | Chạy seed | Coi là chưa đặt: mã ≠ 0 | RUN |
| TC-N02-12 | AC6 | Seed xong | `bash scripts/canary-scan.sh; echo $?`; `grep -rEn` họ tên / MSSV / email trong `seed/**` và `scripts/*seed*` đối chiếu **danh sách mô phỏng** (Q-N-5); email phải thuộc `@edupilot.local` / tên miền mô phỏng | 0 kết quả ngoài danh sách mô phỏng; `canary-scan` mã 0 | SCAN |
| TC-N02-13 | AC6 | Không `GEMINI_API_KEY`, không internet (ngắt mạng của VM hoặc đặt proxy sai `HTTPS_PROXY=http://127.0.0.1:1`) | Seed trên stack trống | Seed vẫn thành công; `content_chunks.embedding` không NULL (embedding giả/tất định); không lỗi kết nối ngoài; `llm_audit` không có provider ngoài | RUN |
| TC-N02-14 | AC6 | – | Seed hai lần trên hai stack trống | Embedding tất định: vector của cùng chunk giống nhau giữa hai lần (hash vector của 20 chunk bằng nhau) | RUN |
| TC-N02-15 | Quy ước | – | `docker compose logs gateway worker` sau seed | Không PII ngoài dữ liệu mô phỏng (`grep` tên/email/MSSV mô phỏng **được phép**; tên người thật / email ngoài `.local` không) | SCAN |

## Phân quyền
TC-N02-06 (4 vai thấy đúng việc; STUDENT không thấy việc của TA). Thêm: SV A `GET …/gradebook` → `403`.

## Lịch sử sửa TC
(trống)

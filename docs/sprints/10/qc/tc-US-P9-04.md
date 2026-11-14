# QC test case — US-P9-04 `/questions`: luồng «Tạo câu hỏi» và duyệt hàng loạt
Nguồn: US-P9-04 + SRS §6 #3, §7 (`/questions`), DESIGN §14.17. Staff dùng desktop ≥ 1024 px (không bắt 375 px cho staff).

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-04-01 | AC1 | GV, lớp 1 | Mở `/questions` 1440×900 | Bảng + lọc (trạng thái, chủ đề, độ khó, loại, nguồn); **một** nút chính `Tạo câu hỏi`; không có điều khiển sinh AI hiển thị mặc định; chụp ảnh | E2E |
| TC-04-02 | AC2 | – | Bấm `Tạo câu hỏi` | Ba lựa chọn: `Soạn tay`, `Trích từ đề`, `Sinh từ tài liệu` (hiện **chỉ nếu** US-P9-03 làm; ghi nhận) | E2E |
| TC-04-03 | AC2 | Đề `EXAM_PAPER` + đáp án `READY` | `Trích từ đề` → chọn tài liệu + tài liệu đáp án → xác nhận | Gọi `POST …/questions/extract` (kiểm bằng bắt request: có `answer_key_document_id`, `Idempotency-Key`); tiến độ hiện **trong luồng**, không modal chặn (bấm sang nơi khác trong trang được); xong: «bỏ qua N câu» mở rộng ra lý do từng câu | E2E |
| TC-04-04 | AC3 | ≥ 5 câu `PENDING` | Chọn 3 hàng; xem thanh hành động | Hiện `Duyệt`, `Loại`, gắn chủ đề / độ khó hàng loạt; chọn 51 hàng → bị chặn ở 50 (ô thứ 51 không chọn thêm hoặc nút khoá, ghi hành vi) | E2E |
| TC-04-05 | AC3 | 4 id: 2 `PENDING` hợp lệ, 1 id `EXTRACTED` có `citations=[]` (dev tạo qua DB), 1 id lớp 2 | `POST /courses/$C1/questions/bulk '{"action":"APPROVE","ids":[…4]}'` | `200 {results:[{id,ok,code?}×4]}`: 2 `ok:true`; câu `citations` rỗng `ok:false code=CITATION_REQUIRED`; id lớp 2 `ok:false` (mã `NOT_FOUND`/`FORBIDDEN`, không lộ nội dung); lô **không** bị huỷ (2 câu hợp lệ đã `APPROVED`) | API |
| TC-04-06 | AC3 | – | `bulk` `REJECT` 3 câu qua UI | Bảng cập nhật ngay (lạc quan); hiện dòng `Hoàn tác` tại chỗ; **không** hộp xác nhận, **không** toast «Thành công»; `Hoàn tác` trong cửa sổ cho phép → 3 câu trở về `PENDING` | E2E |
| TC-04-06b | AC3, AC7 | Cùng; bấm `Loại` 2 câu (hộp `Hoàn tác` đang đếm 5 s) | **Trong 5 s** đóng tab (`page.close({runBeforeUnload:true})`) hoặc điều hướng đi (`pagehide`/`visibilitychange`) | Lệnh `Loại` **được gửi ngay** (`fetch keepalive` lúc `pagehide`, P-10-8 Q-S10-31): mở lại `/questions` thì 2 câu `REJECTED`; DB không còn `PENDING`; không gửi hai lần (cùng `Idempotency-Key`); bấm `Hoàn tác` **trước** khi rời thì không gửi gì | E2E + DB |
| TC-04-07 | AC3 | – | `bulk` `SET_TAGS {topic,difficulty}` 3 câu, rồi `ids` rỗng, rồi `ids` 51 phần tử, rồi `action` lạ | Lần 1: 3 `ok` và đúng `topic`/`difficulty`; còn lại `422 VALIDATION_FAILED` | API |
| TC-04-08 | AC2 | US-P9-03 chưa làm | Mở luồng `Tạo câu hỏi` | Không có `Sinh từ tài liệu`; không route `generate` mở (nếu có → `404`/`501`) | E2E |
| TC-04-09 | AC4 | Câu `PENDING` | Mở Drawer; sửa đề, phương án, đáp án đúng, giải thích, chủ đề, độ khó; `Duyệt` trong Drawer | Lưu; trạng thái `APPROVED`; không cần mở trang khác | E2E |
| TC-04-10 | AC4 | Drawer đang mở, gõ dở 20 ký tự vào «đề» | Đóng nhầm (Esc / bấm ngoài); mở lại cùng câu | Bản gõ dở còn nguyên (`useAutosaveDraft`) | E2E |
| TC-04-11 | AC4 | Hai phiên GV (GV + TA) cùng mở câu X | GV `PUT` sửa; TA `PUT` sửa với `version` cũ | TA nhận `409 VERSION_CONFLICT` kèm bản hiện tại; UI cho chọn giữ bản mình / dùng bản kia, không mất chữ | API + E2E |
| TC-04-12 | AC5 | Câu `EXTRACTED` (có `citations`) | Mở xem | Nhãn nguồn + trích dẫn (tên tài liệu, trang) bấm được → `/library` đúng đoạn / tài liệu; câu `GENERATED` tương tự | E2E |
| TC-04-13 | AC5 | Câu `EXTRACTED`/`GENERATED` `citations` rỗng | `PUT …/questions/{id}/review {action:APPROVE}` | `422 CITATION_REQUIRED` | API |
| TC-04-14 | AC7 | 3 câu chờ duyệt | `context.setOffline(true)`; bấm `Duyệt` hàng loạt; bật mạng | Hàng báo «Chưa gửi được» tại chỗ; khi có mạng gửi lại **một lần** với cùng `Idempotency-Key` (đếm request `bulk`: 1 thành công, 0 trùng; DB không duyệt hai lần) | E2E |
| TC-04-15 | AC8 | GV có N câu `PENDING` | `GET /courses/$C1/today` (nguồn `QUESTION_REVIEW`, bậc 90) rồi duyệt hết | Số việc = N lúc đầu; sau khi duyệt hết, mục biến mất trong ≤ 5 s (đo bằng poll 1 s) | API |
| TC-04-16 | SRS §7 | – | `/questions` rỗng (lớp 2 chưa có câu, hoặc lọc không khớp); lỗi tải (chặn API `route.abort`); đang tải | Rỗng: «Chưa có câu hỏi» + `Tạo câu hỏi`; lỗi: `PageState` + `Thử lại`; tải: skeleton hàng; chụp | E2E |
| TC-04-17 | AC1 | – | `GET …/questions?limit=101`; theo `cursor` đến hết | `limit` bị chặn / cắt ≤ 100; trang cuối `next_cursor` rỗng; không trùng / thiếu id giữa các trang | API |

## Phân quyền
| TC | Bước | Kết quả |
| --- | --- | --- |
| TC-04-P1 | STUDENT `GET/POST/PUT …/questions*`, `…/bulk`, `…/gen-stats` | `403` mọi cái |
| TC-04-P2 | TA `bulk APPROVE` | `200` (duyệt được) |
| TC-04-P3 | SV `GET /practice/overview` và `POST /practice/start` | Chỉ thấy câu `APPROVED` (đối chiếu TC-05-06: câu `PENDING`/`REJECTED` không bao giờ vào đề SV) |
| TC-04-P4 | GV lớp 2 `bulk` trên câu lớp 1 | mỗi id `ok:false`, hoặc `403`/`404` toàn lô (ghi hành vi) |
| TC-04-P5 | ADMIN | `403` |

## Lịch sử sửa TC
- 2026-11-14 — PM quyết P-10-8 (F1–F7, Q-S10-27…31): thêm TC-04-06b (Q31: rời trang / đóng tab trong 5 s → lệnh `Loại` gửi ngay, không mất).
- 2026-11-14 — P-10-9 (PM chấp nhận): TC-04-02: «Soạn tay» là nhóm cha gồm «Câu trắc nghiệm» / «Bài lập trình»; không dòng phụ dưới mục menu (luật chữ UI).

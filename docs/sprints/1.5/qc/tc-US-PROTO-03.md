# QC test case — US-PROTO-03 (Đánh giá)
Nguồn: `docs/sprints/1.5/spec/US.md` + `SRS.md` 4.1, 4.5. Hộp đen. Công cụ: **C** = `proto-curl.sh`, **B** = trình duyệt. Đặt lại dữ liệu demo trước chuỗi TC.

**Điểm (SRS 4.1, cộng/nhân bằng số nguyên phần trăm, làm tròn nửa lên ở cuối).** Con số BT03 / QT B sau công bố đang mâu thuẫn rubric: xem proposals #13. TC chấm theo spec hiện hành (BT03 = 8,5; QT B = 8,8); khi PM quyết thì sửa TC-03-04, TC-01-19, TC-DEMO-05 theo số mới (dẫn số proposal).

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-03-01 | AC1 | – | **C** `tc_03_01` (GV: `/gradebook /gradebook/scheme /grading /grading/sub-bt03-sv-2 /questions /documents`) | Đều `MO` |
| TC-03-02 | AC1 | – | **B** Sweep GV 1440 px 6 route | Mỗi route một `h1`, khung nhìn đầu khớp DESIGN §14.10–14.14, 14.17; một nút chính đỏ mỗi vùng; ảnh `shots/` |
| TC-03-03 | AC2 | `/grading`, lọc mặc định | **B** Mở `/grading` | Lọc `Cần xem kỹ` + `Chưa duyệt` = 4 bài, **B đầu tiên**; tab Bài tập có BT01–BT03, QUIZ01; 28 bài BT03 trong hàng chờ |
| TC-03-04 | AC2 | – | **B** Mở bài B (`/grading/sub-bt03-sv-2`) | Văn bản 2 trang bên trái; bên phải 4 tiêu chí (điểm AI, đoạn trích, nhận xét); thông báo vàng "Hai lượt chấm lệch 1,5 điểm" ở tiêu chí 2; dòng trừ nộp muộn −0,5; tổng hiện tại khớp lượt chấm |
| TC-03-05 | AC2 | – | **B** Sửa tiêu chí 2 thành 2,5 (bước 0,25: thử 2,6 → bị chặn/làm tròn; 3,0 → trên trần 2,5 bị chặn); sửa một nhận xét | Tổng tính lại **ngay** khi sửa; không cho vượt trần 2,5; nhận xét sửa giữ lại sau reload |
| TC-03-06 | AC2 | Sau TC-03-05 | **B** `Duyệt bài` | Trạng thái đã duyệt; quay về hàng chờ **giữ vị trí**; thông báo vàng không chặn `Duyệt bài` (SRS mục 3) |
| TC-03-07 | AC2 | Sau TC-03-06 | **B** Chọn bài đã duyệt → `Công bố` | Dialog xác nhận có hậu quả; sau xác nhận: đã công bố; tổng **8,5 (đã trừ nộp muộn 0,5)** theo spec; sổ điểm có BT03 của B |
| TC-03-08 | AC2 | Sau TC-03-07 | **B** Đổi vai B → `/assignments/bt03`, `/me` | Có điểm và nhận xét; `/me` QT = **8,8** (theo spec; xem proposals #13) |
| TC-03-09 | AC2 / SRS mục 3 | – | **B** Chỉ GV thấy `Công bố`; người duyệt khác (TA) | `Công bố` có ở GV; TA thấy "Chỉ giảng viên công bố điểm" |
| TC-03-10 | AC3 | – | **B** GV đổi lớp 2 → `/gradebook` | Banner cố định "công thức điểm chưa xác nhận"; `Chốt điểm` khoá, lý do hiện khi rê chuột **và khi focus bàn phím** |
| TC-03-11 | AC3 | Lớp 2, `/gradebook/scheme` | **B** Xem bản nháp | QT 30% / CK 70%, +0,2/lần phát biểu trần +0,6, −0,5/buổi vắng từ buổi thứ 3; mỗi mục cạnh trích dẫn có số trang; mục "chưa rõ: quy tắc làm tròn" chặn nút xác nhận |
| TC-03-12 | AC3 | Sau TC-03-11 | **B** Điền D5 `Làm tròn đến 0,1` → `Xác nhận công thức` (sticky) → xác nhận trong hộp thoại nêu hậu quả | Mục hết chặn trước khi nút mở; sau xác nhận trạng thái "đã xác nhận" (CONFIRMED) |
| TC-03-13 | AC3 | Sau TC-03-12 | **B** Về `/gradebook` lớp 2 | Hết banner; `Chốt điểm` mở |
| TC-03-14 | AC3 (nhánh) | – | **B** `/gradebook/scheme` lớp 2, chưa điền D5, bấm `Xác nhận công thức` bằng bàn phím | Bị khoá, nêu lý do; không mở hộp xác nhận |
| TC-03-15 | AC4 | Lớp 1 `/gradebook` | **B** Mở giải trình hàng B (trước khi US-02 điểm danh) | TB bài tập 7,5 + cộng 0,75 = 8,25 → **8,3**; chuyên cần 2 vắng/9 buổi (không bị trừ); ghi chú "điểm chính thức nằm ở hệ thống quản lý đào tạo của trường" |
| TC-03-16 | AC4 | Sau điểm danh + công bố | **B** Giải trình B | Thành phần có +0,25 từ điểm danh (tổng cộng 1,0 = trần) và BT03; QT 8,8 (spec) |
| TC-03-17 | AC4 | – | **B** Hàng C (`Lê Quang Huy`) | Vắng 5/9 buổi → trừ 1,5 (= (5−2) × 0,5), thiếu BT02 |
| TC-03-18 | AC4 | – | **B** Menu → `Xuất XLSX` | "Đã tạo file sổ điểm (mô phỏng)"; không tải file thật; menu overflow (không nút phụ thứ 3) |
| TC-03-19 | AC5 (nhánh lỗi) | `/documents` | **B** Tải "scan-khong-co-chu.pdf" | Tiến độ → FAILED "File không có lớp chữ, không đọc được" (nói vấn đề + cách khắc phục) |
| TC-03-20 | AC5 | `/gradebook` lớp 1 | **B** Sửa ô mẫu xung đột | Dòng 409 "Lê Thu Hà vừa sửa ô này thành 8,0 · Giữ của tôi / Dùng bản mới"; hai nút đều chạy đúng; không mất chữ |
| TC-03-21 | AC5 | Lớp 1, CK trống | **B** `Chốt điểm` | `ConfirmIrreversible` nêu số ("Chốt 30 sinh viên; 30 chưa có điểm cuối kỳ") và **không cho chốt** khi thiếu CK |
| TC-03-22 | AC6 (phân quyền) | – | **C** `tc_03_06` | TA ở `/grading` có "Chỉ giảng viên công bố điểm"; ở `/gradebook/scheme` không có `Xác nhận công thức`; SV và Admin bị chặn 5 route của US |
| TC-03-23 | AC6 | TA | **B** `/grading` (Duyệt bài), `/gradebook` | TA `Duyệt bài` được; ô sổ điểm **không sửa được**; không `Chốt điểm`; `/gradebook/scheme` xem được, không nút xác nhận (SRS: R) |
| TC-03-24 | SRS 4.5 `/questions` | GV | **B** Lọc trạng thái/chủ đề/độ khó/loại/nguồn; mở Drawer → `Duyệt` / `Chỉnh sửa` / `Loại` | 100 câu (80 duyệt + 20 chờ); `Duyệt` hành động chính, `Chỉnh sửa` / `Loại` trong overflow; Hoàn tác; `Tạo câu hỏi` → "đang tạo" rồi +5 câu chờ duyệt; lọc rỗng có hướng |
| TC-03-25 | SRS 4.5 `/documents` | GV | **B** Danh sách | 12 tài liệu (6 bài giảng, quy chế trường, quy chế lớp 1, 2 đề cũ, 2 đáp án); `ANSWER_KEY` ghi "Không hiển thị cho sinh viên · Không dùng cho AI của sinh viên"; bật/tắt cờ có Hoàn tác; rỗng → dropzone |
| TC-03-26 | **Chéo: SV không thấy đáp án** | Sau bật/tắt cờ ở TC-03-25 | **B** SV `/library` | Đáp án (`ANSWER_KEY`) không hiện dù GV thử bật "hiện cho SV" (nếu cho bật → ghi BUG nghiêm trọng: luật "Không cho sinh viên thấy đáp án khi bài còn mở") |
| TC-03-27 | SRS 4.5 `/gradebook/scheme` rỗng | – | **B** `?state=empty` | "Tải quy chế" → tải giả lập có tiến độ → bản nháp |
| TC-03-28 | Trạng thái | – | **B** `?state=loading|empty|error` 6 route | Skeleton đúng hình, rỗng có hành động, lỗi có `Thử lại` |

## Nhánh lỗi
TC-03-14, 19, 20, 21, 28; TC-03-05 (số ngoài thang).

## Phân quyền
TC-03-09, 22, 23, 26 (chỉ TEACHER công bố / xác nhận công thức / chốt điểm; TA chỉ đọc sổ điểm; SV/Admin bị chặn; đáp án không tới SV).

## Kiểm chéo
- Nguyên tắc "AI chỉ nháp, người quyết": AI không tự công bố ở bất kỳ bước nào (không có đường đến "đã công bố" mà không qua `Công bố` của GV) — TC-03-07, 09.
- Điểm bằng code: số hiển thị khớp công thức SRS 4.1 (cộng nguyên phần trăm, làm tròn nửa lên; dấu phẩy) — TC-03-15…17.
- Phản mẫu §21: `/gradebook` không tường KPI; Dialog chỉ cho `Công bố`, `Chốt điểm`, `Xác nhận công thức` (danh sách trắng); không toast "Thành công!".

## Điểm khó kiểm
- Con số BT03/QT: proposals #13 chưa quyết.
- Bảng dài 30 hàng: kiểm tràn chữ ở 1440 px và 390 px bằng ảnh.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
(chưa có)

# QC test case — US-PROTO-03 (Đánh giá)
Nguồn: `docs/sprints/1.5/spec/US.md` + `SRS.md` 4.1, 4.5. Hộp đen. Công cụ: **C** = `proto-curl.sh`, **B** = trình duyệt. Đặt lại dữ liệu demo trước chuỗi TC.

**Điểm (SRS 4.1, cộng/nhân bằng số nguyên phần trăm, làm tròn nửa lên ở cuối).** Con số BT03 / QT B sau công bố đang mâu thuẫn rubric: xem proposals #13. Đã cập nhật theo spec v3 (proposals #13): BT03 công bố = 8,0 (tổng 8,5 trừ muộn 0,5); QT B sau công bố = 8,7.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-03-01 | AC1 | – | **C** `tc_03_01` (GV: `/gradebook /gradebook/scheme /grading /grading/sub-bt03-sv-2 /questions /documents`) | Đều `MO` |
| TC-03-02 | AC1 | – | **B** Sweep GV 1440 px 6 route | Mỗi route một `h1`, khung nhìn đầu khớp DESIGN §14.10–14.14, 14.17; một nút chính đỏ mỗi vùng; ảnh `shots/` |
| TC-03-03 | AC1 | GV ở `/grading/[submissionId]` (1440 px) | **B** Mở `/grading/sub-bt03-sv-2` | Phân định rõ 2 panel độc lập (FR-X11, Proposal #16): Panel xem bài nộp sinh viên (bên trái, 52–58%) và Panel rubric / điểm số (bên phải, 42–48%), đều có viền `1px solid var(--ep-rule)`, nền `var(--ep-surface)`, bo góc, padding rõ ràng và cuộn độc lập giữa 2 cột |
| TC-03-04 | AC2 | `/grading`, lọc mặc định | **B** Mở `/grading` | Lọc `Cần xem kỹ` + `Chưa duyệt` = 4 bài, **B đầu tiên**; tab Bài tập có BT01–BT03, QUIZ01; 28 bài BT03 trong hàng chờ |
| TC-03-05 | AC2 | – | **B** Mở bài B (`/grading/sub-bt03-sv-2`) | Văn bản 2 trang bên trái; bên phải 4 tiêu chí (điểm AI, đoạn trích, nhận xét); thông báo vàng "Hai lượt chấm lệch 1,5 điểm" ở tiêu chí 2; dòng trừ nộp muộn −0,5; tổng hiện tại khớp lượt chấm |
| TC-03-06 | AC2 | – | **B** Sửa tiêu chí 2 thành 2,5 (bước 0,25: thử 2,6 → bị chặn/làm tròn; 3,0 → trên trần 2,5 bị chặn); sửa một nhận xét | Tổng tính lại **ngay** khi sửa; không cho vượt trần 2,5; nhận xét sửa giữ lại sau reload |
| TC-03-07 | AC2 | Sau TC-03-05 | **B** `Duyệt bài` | Trạng thái đã duyệt; quay về hàng chờ **giữ vị trí**; thông báo vàng không chặn `Duyệt bài` (SRS mục 3) |
| TC-03-08 | AC2 | Sau TC-03-06 | **B** Chọn bài đã duyệt → `Công bố` | Dialog xác nhận có hậu quả; sau xác nhận: đã công bố; tổng các tiêu chí 8,5, trừ nộp muộn 0,5 → công bố **8,0** (SRS 4.1 v3); sổ điểm có BT03 của B |
| TC-03-09 | AC2 | Sau TC-03-07 | **B** Đổi vai B → `/assignments/bt03`, `/me` | Có điểm 8,0 và nhận xét; `/me` QT = **8,7** (SRS 4.1 v3) — chuỗi đủ: GV `Lưu điểm danh` buổi 10 (B có phát biểu) → QT 8,5; rồi chỉnh + công bố BT03 → 8,7; chỉ công bố BT03 mà chưa lưu điểm danh cho QT 8,2–8,4 |
| TC-03-10 | AC2 / SRS mục 3 | – | **B** Chỉ GV thấy `Công bố`; người duyệt khác (TA) | `Công bố` có ở GV; TA thấy "Chỉ giảng viên công bố điểm" |
| TC-03-11 | AC3 | – | **B** GV đổi lớp 2 → `/gradebook` | Banner cố định "công thức điểm chưa xác nhận"; `Chốt điểm` khoá, lý do hiện khi rê chuột **và khi focus bàn phím** |
| TC-03-12 | AC3 | Lớp 2, `/gradebook/scheme` | **B** Xem bản nháp | QT 30% / CK 70%, +0,2/lần phát biểu trần +0,6, −0,5/buổi vắng từ buổi thứ 3; mỗi mục cạnh trích dẫn có số trang; mục "chưa rõ: quy tắc làm tròn" chặn nút xác nhận |
| TC-03-13 | AC3 | Sau TC-03-11 | **B** Điền D5 `Làm tròn đến 0,1` → `Xác nhận công thức` (sticky) → xác nhận trong hộp thoại nêu hậu quả | Mục hết chặn trước khi nút mở; sau xác nhận trạng thái "đã xác nhận" (CONFIRMED) |
| TC-03-14 | AC3 | Sau TC-03-12 | **B** Về `/gradebook` lớp 2 | Hết banner; `Chốt điểm` mở |
| TC-03-15 | AC3 (nhánh) | – | **B** `/gradebook/scheme` lớp 2, chưa điền D5, bấm `Xác nhận công thức` bằng bàn phím | Bị khoá, nêu lý do; không mở hộp xác nhận |
| TC-03-16 | AC4 | Lớp 1 `/gradebook` | **B** Mở giải trình hàng B (trước khi US-02 điểm danh) | TB bài tập 7,5 + cộng 0,75 = 8,25 → **8,3**; chuyên cần 2 vắng/9 buổi (không bị trừ); ghi chú "điểm chính thức nằm ở hệ thống quản lý đào tạo của trường" |
| TC-03-17 | AC4 | Sau điểm danh + công bố | **B** Giải trình B | Thành phần có +0,25 từ điểm danh (tổng cộng 1,0 = trần) và BT03; QT 8,7 (spec v3) |
| TC-03-18 | AC4 | – | **B** Hàng C (`Lê Quang Huy`) | Vắng 5/9 buổi → trừ 1,5 (= (5−2) × 0,5), thiếu BT02 |
| TC-03-19 | AC4 | – | **B** Menu → `Xuất XLSX` | "Đã tạo file sổ điểm (mô phỏng)"; không tải file thật; menu overflow (không nút phụ thứ 3) |
| TC-03-20 | AC5 (nhánh lỗi) | `/documents` | **B** Tải "scan-khong-co-chu.pdf" | Tiến độ → FAILED "File không có lớp chữ, không đọc được" (nói vấn đề + cách khắc phục) |
| TC-03-21 | AC5 | `/gradebook` lớp 1 | **B** Sửa ô mẫu xung đột | Dòng 409 "Lê Thu Hà vừa sửa ô này thành 8,0 · Giữ của tôi / Dùng bản mới"; hai nút đều chạy đúng; không mất chữ |
| TC-03-22 | AC5 | Lớp 1, CK trống | **B** `Chốt điểm` | `ConfirmIrreversible` nêu số ("Chốt 30 sinh viên; 30 chưa có điểm cuối kỳ") và **không cho chốt** khi thiếu CK |
| TC-03-23 | AC6 (phân quyền) | – | **C** `tc_03_06` | TA ở `/grading` có "Chỉ giảng viên công bố điểm"; ở `/gradebook/scheme` không có `Xác nhận công thức`; SV và Admin bị chặn 5 route của US |
| TC-03-24 | AC6 | TA | **B** `/grading` (Duyệt bài), `/gradebook` | TA `Duyệt bài` được; ô sổ điểm **không sửa được**; không `Chốt điểm`; `/gradebook/scheme` xem được, không nút xác nhận (SRS: R) |
| TC-03-25 | SRS 4.5 `/questions` | GV | **B** Lọc trạng thái/chủ đề/độ khó/loại/nguồn; mở Drawer → `Duyệt` / `Chỉnh sửa` / `Loại` | 100 câu (80 duyệt + 20 chờ); `Duyệt` hành động chính, `Chỉnh sửa` / `Loại` trong overflow; Hoàn tác; `Tạo câu hỏi` → "đang tạo" rồi +5 câu chờ duyệt; lọc rỗng có hướng |
| TC-03-26 | SRS 4.5 `/documents` | GV | **B** Danh sách | 12 tài liệu (6 bài giảng, quy chế trường, quy chế lớp 1, 2 đề cũ, 2 đáp án); `ANSWER_KEY` ghi "Không hiển thị cho sinh viên · Không dùng cho AI của sinh viên"; bật/tắt cờ có Hoàn tác; rỗng → dropzone |
| TC-03-27 | **Chéo: SV không thấy đáp án** | Sau bật/tắt cờ ở TC-03-25 | **B** SV `/library` | Đáp án (`ANSWER_KEY`) không hiện dù GV thử bật "hiện cho SV" (nếu cho bật → ghi BUG nghiêm trọng: luật "Không cho sinh viên thấy đáp án khi bài còn mở") |
| TC-03-28 | SRS 4.5 `/gradebook/scheme` rỗng | – | **B** `?state=empty` | "Tải quy chế" → tải giả lập có tiến độ → bản nháp |
| TC-03-29 | Trạng thái | – | **B** `?state=loading|empty|error` 6 route | Skeleton đúng hình, rỗng có hành động, lỗi có `Thử lại` |
| TC-03-30 | AC7 (#19 E21) | GV, `/gradebook`, 390 px | **A** (SPEC_V51 "03-AC7"); **B** Console `['col-qt','col-status'].map(p => document.querySelector('[data-part=' + p + ']').getBoundingClientRect().right <= innerWidth)` | `[true, true]`: cột tên (dính trái), "QT tạm tính" và "Trạng thái" đều trong khung nhìn đầu, không phải cuộn ngang; dòng gợi ý "Kéo ngang để xem BT01–BT03 và Cuối kỳ" phía trên bảng |
| TC-03-31 | AC7 | Như TC-03-30 | **B** Vuốt ngang bảng; ảnh 390 | Các cột còn lại (BT01–BT03, cộng / trừ, Cuối kỳ) cuộn ngang, cột tên giữ chỗ khi cuộn; mép phải bảng lộ một phần cột kế (gợi ý có thể kéo); không cột nào bị mất; `ox` 0 |
| TC-03-32 | AC7 | GV, 390 px | **A** AUDIT + **B** `/students`, `/grading`, `/documents` (GV), `/admin/courses`, `/admin/users` (Admin) | Dạng danh sách, không cột nào biến mất: số cặp "Nhãn: giá trị" ở dòng phụ = số cột − 1 (`/documents`: loại · tuần · Dùng cho AI · Hiện cho SV · trạng thái · ngày); `cut: []` |
| TC-03-33 | AC7 | GV, `/students/sv-3`, 390 px | **A** + **B** vuốt dải tab; chọn "Ghi chú" | Thanh tab cuộn ngang, mép tab kế lộ ra; "Ghi chú" tới được và tự cuộn vào khung khi chọn; `cut: []` |
| TC-03-34 | AC7 (biên) | GV | **A** 719 / 720 px và 375 px trên các route TC-03-30…33 | 720: dạng bảng đúng ngưỡng; 719 và 375: dạng danh sách / vẫn thấy `col-qt`, `col-status` trong khung nhìn đầu; không tràn |
| TC-03-35 | AC8 (#19 E10, E33) | GV, `/documents`, 1440 | **B** + **C** `tc_03_08` | 12 dòng đúng bảng SRS 4.8 N5 (tên hiển thị, loại, tuần, ngày tải): Chương 1 (tuần 1 · 27/08) … Chương 5 (tuần 10 · 28/10), Modern Network Security Threats (tuần 2 · 03/09), Quy chế đào tạo của trường, Quy chế môn học An ninh mạng – 761987, 2 đề cũ, 2 đáp án; không còn "Hạ tầng khoá công khai" (v4) |
| TC-03-36 | AC8 | Như TC-03-35 | **B** Mở Drawer chi tiết của từng dòng; `Ctrl+F` ở danh sách | Tên tệp gốc (`an-ninh-mang-ch3.pdf`, `Mordern_Network_Security_Threats.pdf`) **chỉ** ở Drawer và tên khi tải; danh sách không có tên dạng gạch dưới / "Mordern" (`… \| grep -cE 'Chuong[0-9]_\|Mordern'` → `0`) |
| TC-03-37 | AC8 / E10 | GV, SV B | **B** So tên 10 tài liệu SV thấy ở `/documents` với `/library` (chữ, tuần, ngày, trang, dung lượng) | Trùng chữ từng dòng; `Nguồn tham khảo` của AI (chat, Threads, luyện đề) và chuông dùng cùng cột "Tên hiển thị" |
| TC-03-38 | AC8 / luật đáp án | GV, SV B | **B** 2 `ANSWER_KEY` ở `/documents`; thử bật cờ "Hiện cho SV"; SV `/library` | Hai dòng ghi "Không hiển thị cho sinh viên" (cột SV thấy / AI: **Không**); không bật được hoặc bật không làm SV thấy (nếu SV thấy → BUG nghiêm trọng); SV `/library` không có "Đáp án đề" |
| TC-03-39 | AC8 (nhánh; siết theo #20a) | GV | **B** Tải một file mẫu; quan sát trong lúc "Đang xử lý" | Cột "Dùng cho AI" ghi đúng **"Chờ xử lý"** (không "Có", không "Chờ xác nhận" — chữ này dành riêng cho câu AI); cờ "Hiện cho SV" khoá trong lúc xử lý; xong: `READY`, "Dùng cho AI" = "Có", cờ mở |
| TC-03-40 | AC8 (biên) | GV, 1440 và 390 | **B** Tên dài ("Quy chế môn học An ninh mạng – 761987", "Đề thi giữa kỳ An ninh mạng — HK1 2024–2025") | Xuống dòng giữa các từ, không ngắt giữa từ, không `…` mất chữ; AUDIT `cut: []` |
| TC-03-41 | AC8 / N5 | GV lớp 2 | **B** `/documents` lớp 761988; SV A / D chọn lớp 2 `/library` | Lớp 2: như bảng N5 trừ `Quy chế môn học … 761987` → GV 11 dòng, SV 9 tài liệu; không chữ "761987" ở lớp 2 |


## Công cụ bổ sung (v5.1)
**A** = `scripts/audit.mjs` (phép đo "03-AC7", `TOUCH` / `LEFT` / header toàn ma trận); **C** `tc_03_08`.

## Nhánh lỗi
TC-03-14, 19, 20, 21, 28; TC-03-05 (số ngoài thang); v5.1: TC-03-34 (ngưỡng 720), 39 (tài liệu đang xử lý), 40 (tên dài).

## Phân quyền
TC-03-09, 22, 23, 26 (chỉ TEACHER công bố / xác nhận công thức / chốt điểm; TA chỉ đọc sổ điểm; SV/Admin bị chặn; đáp án không tới SV).

## Kiểm chéo
- Nguyên tắc "AI chỉ nháp, người quyết": AI không tự công bố ở bất kỳ bước nào (không có đường đến "đã công bố" mà không qua `Công bố` của GV) — TC-03-07, 09.
- Điểm bằng code: số hiển thị khớp công thức SRS 4.1 (cộng nguyên phần trăm, làm tròn nửa lên; dấu phẩy) — TC-03-15…17.
- Phản mẫu §21: `/gradebook` không tường KPI; Dialog chỉ cho `Công bố`, `Chốt điểm`, `Xác nhận công thức` (danh sách trắng); không toast "Thành công!".

## Điểm khó kiểm
- Con số BT03/QT: đã chốt theo proposals #13 (v3: BT03 = 8,0; QT = 8,7).
- Bảng dài 30 hàng: kiểm tràn chữ ở 1440 px và 390 px bằng ảnh.
- 03-AC8 (đã chốt, proposals #20a): tài liệu `PROCESSING` ghi **"Chờ xử lý"** ở "Dùng cho AI"; "Chờ xác nhận" dành riêng cho câu AI — TC-03-39 kiểm đúng chữ; tài liệu seed không có dòng nào ở trạng thái này (mọi dòng `READY`).
- 03-AC7 "mép phải bảng lộ một phần cột kế" không có phép đo trong spec — TC-03-31 kiểm bằng mắt / ảnh; `audit.mjs` chỉ đo `col-qt`, `col-status` và dòng gợi ý.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-01 · TC-03-07, 08, 16: Cập nhật BT03 công bố = 8,0 và QT sau công bố = 8,7 theo spec v3 (proposals #13).
- 2026-10-01 · TC-03-03: Thêm kiểm tra bố cục 2 panel độc lập (viền, nền surface, bo góc, cuộn độc lập giữa bài nộp và rubric) của `/grading/[submissionId]` theo spec v4 (03-AC1, proposals #16).
- 2026-10-02 · Thêm TC-03-30…41 (không sửa TC cũ): spec v5.1 (#19) — 03-AC7 (E21), 03-AC8 (E10, E33) + SRS 4.8 N5. TC-03-26 vẫn đúng (12 dòng, 2 đáp án).
- 2026-10-02 · **Siết TC-03-39:** "Chờ xử lý" (PM chốt, proposals #20a, ACCEPTED).
- 2026-10-02 · **Sửa chữ TC-03-09 (proposals #25, ACCEPTED):** thêm bước lưu điểm danh trước khi QT = 8,7.

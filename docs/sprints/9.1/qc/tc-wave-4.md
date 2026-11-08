# TC đợt 4 — Chấm bài / Sổ điểm / Điểm danh (US-UX91-14, 15, 16, 12)

Phiên bản 1 · 2026-11-08. Quy ước và kiểm chung: `tc-00-chung.md`. Các TC có ghi dữ liệu (duyệt, lưu điểm, chốt) chạy trên stack `ep-qc`, không bao giờ trên demo.
Lớp mẫu: 761987 (30 SV, 1 bài tập có 4 bài nộp chưa duyệt, có điểm). Lớp **không có công thức xác nhận** và lớp **chưa có bài nộp**: lớp mới do admin mở trên `ep-qc` (ghi trong báo cáo cách dựng).

---
## US-UX91-14 — Hàng chờ chấm bài

| TC | AC | Vai / cỡ | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- |
| TC-UX14-1 | AC1 | giảng viên, 1440 | `/grading`, bộ lọc «Bài tập» chọn **một** bài | Cột «Bài tập» ẩn; tên bài hiện đúng một lần ở tiêu đề bảng | E2E |
| TC-UX14-2 | AC1 | giảng viên | Lớp có ≥ 2 bài tập (tạo thêm bài trên `ep-qc` hoặc mock danh sách) chọn «Tất cả bài tập» | Cột hiện; tên bài **một dòng** cắt `…` (`text-overflow: ellipsis`, `white-space: nowrap`); đầy đủ trong tooltip khi rê / focus | E2E |
| TC-UX14-3 | AC1 | giảng viên | Lớp chỉ có một bài (không lọc) | Cột ẩn như TC-UX14-1 | E2E |
| TC-UX14-4 | AC2 | giảng viên | Hàng có «Lý do cờ» | Lý do nằm ngay dưới tên sinh viên, trước cột điểm (so `top` của phần tử); hàng không cờ không chừa khoảng trống (chiều cao hàng nhỏ hơn hàng có cờ) | E2E |
| TC-UX14-5 | AC3 | giảng viên | Rê / Tab vào một hàng | Nền hover (chỉ `hover: hover`); có liên kết `Duyệt` ở cuối (trung tính, mũi tên); click bất kỳ chỗ nào trên hàng mở `/grading/‹id›`; Enter khi hàng focus mở cùng trang | E2E |
| TC-UX14-6 | AC3 | giảng viên | `getComputedStyle(hàng).cursor`; trên `hasTouch` | `pointer` chỉ khi `hover: hover` | E2E |
| TC-UX14-7 | AC4 | giảng viên, 1440 | Đo `getBoundingClientRect().bottom` chip «Cần xem kỹ» và hai ô chọn; đo rộng ô «Bài tập» so với nội dung dài nhất | Cùng hàng, đáy lệch ≤ 1 px; rộng ô ≤ nội dung dài nhất + 24 px | E2E |
| TC-UX14-8 | AC5 | giảng viên | Lớp chưa có bài nộp → `Chấm tất cả`; lớp có bài (761987) | Khoá + lý do (US-03); có bài → bấm được | E2E |
| TC-UX14-9 | AC6 | giảng viên | Chọn bộ lọc rồi huỷ API; tải lại danh sách | `ApiErrorNotice` + `Thử lại`; bộ lọc đã chọn còn nguyên | E2E |
| TC-UX14-10 | AC7 | STUDENT, TA | `/grading` | STUDENT 403; TA mở được | E2E |
| TC-UX14-11 | AC8 | giảng viên, 375 | `/grading` | Hàng xếp dọc (tên → lý do cờ → điểm → trạng thái); `Duyệt` xuống dòng riêng cao ≥ 44 px; K4 | E2E |
| TC-UX14-12 | AC9 | — | Chụp `grading-queue-1440` trước / sau | Đủ ảnh | MAC |
| TC-UX14-PQ | (kiểm chéo) | giảng viên khác lớp | `GET /courses/‹lớp lạ›/submissions` bằng token GV không thuộc lớp | 403 / 404 (CourseAccessGuard) | E2E |
| TC-UX14-PG | (phân trang) | giảng viên | Quan sát request danh sách bài | Có `limit` ≤ 100 và `cursor` (không OFFSET, không trả toàn bộ) | E2E |

---
## US-UX91-15 — Duyệt bài

Dữ liệu: bài `‹sid›` trong lớp 761987 chưa duyệt, có ≥ 2 tiêu chí, ≥ 1 tiêu chí có đoạn trích, tên tiêu chí dài (≥ 40 ký tự). Bài có hai lượt chấm lệch > 1 điểm ở lớp 761987 (P-91-5 Q4: dev / seed thêm, mã bài ghi ở handoff).

| TC | AC | Vai / cỡ | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- |
| TC-UX15-1 | AC1 | giảng viên, 1440 | `/grading/‹sid›`; đọc DOM cột phải | Mỗi tiêu chí = **một hàng**: tên · ô điểm · `/{tối đa}` cùng dòng (cùng `top`±8 px); không còn nhãn «Điểm {tên tiêu chí}» | E2E |
| TC-UX15-2 | AC1 | giảng viên, 1440 và 1100 | Tên tiêu chí ≥ 40 ký tự | Tên xuống dòng, **không** chồng lên ô điểm (hai hộp không giao nhau) | E2E |
| TC-UX15-3 | AC2 | giảng viên | Tiêu chí có đoạn trích | Dòng trích có biểu tượng nhỏ dưới tiêu chí; câu «Đoạn trích từ bài làm · bấm để xem trong bài» xuất hiện **tối đa 1 lần** trên trang (hoặc chỉ ở tooltip biểu tượng) | E2E |
| TC-UX15-4 | AC2 | giảng viên | Bấm dòng trích; lấy mẫu `background-color` của đoạn được tô mỗi 50 ms trong 2 s | Bài làm cuộn tới đoạn; nền `--ep-red-soft`; **giữ ≈ 1 s** (±0,2 s) rồi mờ trong ≤ 200 ms | E2E |
| TC-UX15-5 | AC2 | giảng viên, `reduce` | Như trên | Giữ ≈ 1 s rồi tắt tức thì (≤ 1 frame) | E2E |
| TC-UX15-6 | AC3 | giảng viên, 1440×900 | Cột phải dài hơn khung; đo `getBoundingClientRect()` thanh dưới lúc mới mở và sau mỗi lần cuộn cột | Thanh luôn nằm **trọn** trong khung (`bottom ≤ 900`, không bị cắt); chứa `Duyệt bài` (nút đỏ duy nhất trong thanh), «Đã duyệt x/N», gợi ý phím `A`, hai nút `‹` `›` | E2E |
| TC-UX15-7 | AC3 | giảng viên | Đếm nút đỏ trong khung nhìn đầu | 1 (`Duyệt bài`) | E2E |
| TC-UX15-8 | AC4 | giảng viên | Focus trong trang; nhấn `A` | Bài ở trạng thái đã duyệt, màn sang bài kế (URL đổi); số «Đã duyệt» tăng 1 | E2E |
| TC-UX15-9 | AC4 | giảng viên | Nhấn `›` / `‹`; `getAnimations()` ngay sau phím | Đổi bài; không hoạt cảnh (số animation = 0); gợi ý phím có nhãn trong thanh | E2E |
| TC-UX15-10 | AC4 (idempotency) | giảng viên | Nhấn `A` hai lần nhanh / bấm đúp `Duyệt bài`; quan sát request | Chỉ một lần duyệt có hiệu lực (kết quả `approved` một lần; mã lặp trả cùng kết quả / 409 không vỡ giao diện) | E2E |
| TC-UX15-11 | AC5 | giảng viên | Bài có hai lượt lệch 1,5 điểm | Thông báo hổ phách inline cạnh tiêu chí «Lệch 1,5 điểm so với lượt 2»; không modal | E2E |
| TC-UX15-12 | AC6 | giảng viên | Sửa nhận xét và điểm; **không** lưu thủ công; `reload` | Nháp khôi phục; đổi điểm tự lưu lạc quan, trạng thái lưu hiện tại ô | E2E |
| TC-UX15-13 | AC7 | giảng viên | Huỷ API rồi bấm `Duyệt bài` | Báo lỗi **tại thanh dưới**; điểm đã nhập còn nguyên; nút `Thử lại`; bấm khi bỏ chặn → duyệt được | E2E |
| TC-UX15-14 | AC7 | giảng viên | Mock 409 ở lưu điểm (phiên bản cũ) | Hiện giá trị của người kia và hỏi giữ bên nào; không mất giá trị đã nhập | E2E |
| TC-UX15-15 | AC8 | giảng viên → sinh viên | Duyệt bài; sau đó đăng nhập SV của bài đó xem kết quả | Bài «Đã duyệt» ở phía giảng viên; **SV chưa thấy điểm** (chưa công bố — công bố là thao tác riêng) | E2E |
| TC-UX15-16 | AC9 | STUDENT | `/grading/‹sid›` | 403 | E2E |
| TC-UX15-17 | AC9 | STUDENT | Màn bài tập của SV khi bài còn mở / chưa công bố | Không điểm nháp, không ghi chú quan sát, không đáp án | E2E |
| TC-UX15-18 | AC9 | TA | Duyệt một bài | Làm được theo quyền hiện có; `Chấm tất cả` / công bố theo quyền cũ | E2E |
| TC-UX15-19 | AC10 | giảng viên, 375 | `/grading/‹sid›` | Hai tab «Bài làm» / «Chấm»; thanh hành động dính đáy cao ≥ 44 px, nằm trên thanh điều hướng đáy; K4 | E2E |
| TC-UX15-20 | AC11 | — | `assignment-lifecycle.spec.ts` diff + chạy | Sửa theo AC1–AC4, không nới; ảnh mốc `grading-review-1440` thêm | E2E |
| TC-UX15-21 | (K7) | — | Chụp `grading-review` 1440 + 375 trước (`audit/shots/grading-ID__teacher__*`) và sau | Đủ ảnh | MAC |
| TC-UX15-PQ | (kiểm chéo) | giảng viên khác lớp | `GET /courses/‹lớp lạ›/submissions/‹sid›` | 403 / 404 | E2E |

---
## US-UX91-16 — Sổ điểm

Dữ liệu định dạng: QC chặn phản hồi sổ điểm, đặt cột «Quá trình» = `"6"`, `"5.5"`, `"7.915"`; cột «Giữa kỳ» = `"7"`, `"8.5"`, `"9"`; một hàng thiếu điểm thi cuối kỳ (`null`).

| TC | AC | Vai / cỡ | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- |
| TC-UX16-1 | AC1 | giảng viên, 1440 | Mở `/gradebook` với dữ liệu trên | Cột Quá trình: «6,000», «5,500», «7,915» (3 chữ số lẻ, **không** thành «7,92»); cột Giữa kỳ: «7,00», «8,50», «9,00» (tối thiểu 2); dấu phẩy | E2E |
| TC-UX16-2 | AC1 | giảng viên | `getComputedStyle` ô số | `text-align: right`; `font-variant-numeric` chứa `tabular-nums` | E2E |
| TC-UX16-3 | AC1 | giảng viên | Lớp 761987 thật: so mọi ô số với giá trị API (`GET /courses/‹id›/gradebook`) bằng cách bỏ đệm số 0 | Giá trị khớp từng ô (UI không làm tròn, không tính lại) | E2E |
| TC-UX16-4 | AC2 | giảng viên | Đo tiêu đề cột | Một dòng, cùng chiều cao; cột đã công bố có chấm nhỏ trước tên; chân bảng có chú giải «● đã công bố»; không còn dòng phụ «Đã công bố» và `⋯` thường trực | E2E |
| TC-UX16-5 | AC2 | giảng viên | Rê / focus tiêu đề cột | Menu `⋯` của cột hiện | E2E |
| TC-UX16-6 | AC3 | giảng viên, 1440×900 | Đếm phần tử có `overflow-y: auto\|scroll` và `scrollHeight > clientHeight` trong Panel | Đúng **1** vùng cuộn dọc; chiều cao ≈ `100dvh − đầu trang` (±8 px); tiêu đề cột và cột «Sinh viên» `position: sticky` trong vùng cuộn | E2E |
| TC-UX16-7 | AC3 | giảng viên | Cuộn tới cuối | Hàng cuối nhìn thấy trọn (không cắt giữa chừng) | E2E |
| TC-UX16-8 | AC3 | giảng viên | Mock 1.000 sinh viên; đếm số `tr` / hàng trong DOM | Còn ảo hoá: số hàng DOM ≪ 1.000 (≤ 80); cuộn vẫn mượt (không > 50 ms/frame trong trace) | E2E |
| TC-UX16-9 | AC4 | giảng viên | Dữ liệu có hàng thiếu | Hàng thiếu: «Thiếu điểm» hổ phách + chấm; hàng đủ để trống; tiêu đề phụ «Thiếu điểm: N sinh viên» (N đúng) là liên kết lọc; bấm → chỉ còn hàng thiếu | E2E |
| TC-UX16-10 | AC4 | giảng viên | Dữ liệu **không** hàng nào thiếu | Cột «Trạng thái» ẩn hoàn toàn; không tiêu đề phụ «Thiếu điểm» | E2E |
| TC-UX16-11 | AC4 | giảng viên | Tìm ô có chữ chính xác «Đủ» trong bảng | Không còn ô nào hiện «Đủ» (giá trị cũ bị bỏ khỏi màn) | E2E |
| TC-UX16-12 | AC5 | hai giảng viên | Hai context cùng sửa một ô; B lưu sau A | Ô viền hổ phách + dòng «{tên} vừa đổi thành 6,5 · Giữ của tôi · Lấy 6,5»; giá trị B nhập không mất; chọn từng nút hoạt động | E2E |
| TC-UX16-13 | AC6 | giảng viên | Enter / Tab trong ô | Enter xuống dòng, Tab sang ô; ô lưu xong nháy nền xanh nhạt 200 ms | E2E |
| TC-UX16-14 | AC6 | giảng viên, `reduce` | Như trên | Đổi màu tức thì, giữ ≥ 300 ms | E2E |
| TC-UX16-15 | AC7 | giảng viên | Lớp công thức chưa xác nhận | `Chốt điểm` khoá + lý do | E2E |
| TC-UX16-16 | AC7 | giảng viên | Lớp công thức đã xác nhận, đủ điều kiện | Nút bấm được; bấm → `ConfirmIrreversible` nêu hậu quả **bằng số** (phân bố, số SV liệt); `Esc` không chốt | E2E |
| TC-UX16-17 | AC7 (idempotency) | giảng viên | Xác nhận chốt bấm đúp (lớp thử) | Chốt đúng một lần; lần hai không đổi dữ liệu / báo đã chốt | E2E |
| TC-UX16-18 | AC8 | giảng viên | Huỷ API; sau đó lớp **đã chốt** | `ApiErrorNotice` + `Thử lại`; sổ điểm đã chốt chỉ đọc (không ô sửa, không `input`) | E2E |
| TC-UX16-19 | AC9 | STUDENT | `/gradebook` | 403 | E2E |
| TC-UX16-20 | AC9 | TA / giảng viên lớp khác | TA xem / sửa theo quyền hiện có; `GET /courses/‹lớp lạ›/gradebook` bằng token GV lớp khác; TA bấm `Chốt điểm` | TA không chốt được (chỉ TEACHER); lớp lạ 403 / 404 | E2E |
| TC-UX16-21 | AC10 | giảng viên, 375 | `/gradebook` | Cuộn ngang **trong bảng** với cột tên dính; `scrollWidth(document) ≤ clientWidth`; ô sửa cao ≥ 44 px | E2E |
| TC-UX16-22 | AC11 | — | `gradebook.spec.ts` diff + chạy | Sửa kiểm định dạng số và cột trạng thái, không nới; ảnh mốc `gradebook-1440`, `gradebook-390` đổi | E2E |
| TC-UX16-23 | (K7) | — | Chụp trước (`audit/shots/gradebook__teacher__*`) và sau | Đủ ảnh | MAC |
| TC-UX16-24 | (tính điểm) | giảng viên | So điểm học phần 30 SV mẫu với `expected_final_grades.csv` (kiểm API, không phải UI) | Khớp 30/30 (sprint không đổi giá trị; UI chỉ đổi hiển thị) | UNIT |

---
## US-UX91-12 — Điểm danh

Lớp 761987 (30 SV, buổi chưa lưu). Lớp thử: lớp mới với công thức chưa xác nhận (cho AC6).

| TC | AC | Vai / cỡ | Bước | Kết quả mong đợi | Chạy |
| --- | --- | --- | --- | --- | --- |
| TC-UX12-1 | AC1 | giảng viên, 1440 | `/attendance` buổi chưa lưu; đọc màu chấm «Có mặt» | Chấm của lựa chọn «Có mặt» màu mực (`--ep-ink`), không đỏ; số chấm đỏ = 0 | E2E |
| TC-UX12-2 | AC1 | giảng viên | Chọn Muộn ở hàng 2, Vắng phép ở hàng 5, Vắng ở hàng 9 | Ba chấm của lựa chọn đó **đỏ**; số chấm đỏ = 3 = số SV khác mặc định | E2E |
| TC-UX12-3 | AC2 | giảng viên | Hàng Vắng; cây truy cập | Ngoài chấm đỏ có nhãn chữ trạng thái; trình đọc đọc được «Vắng» (`radio` có tên + `checked`) | E2E |
| TC-UX12-4 | AC3 | giảng viên, 1440 | Không rê chuột; quét chữ «+ phát biểu · +0,25» | Không lặp trên mọi hàng | E2E |
| TC-UX12-5 | AC3 | giảng viên | Rê / focus hàng; phím `P` | Nút thêm phát biểu hiện; `P` thêm một lần phát biểu cho hàng đang chọn | E2E |
| TC-UX12-6 | AC3 | giảng viên | Hàng đã có 2 lần phát biểu | Hiện `+2 · +0,25×2` (hoặc `+N · +0,25×N` với N = 2) | E2E |
| TC-UX12-7 | AC3 | giảng viên, 375 | Không hover | Nút `+` thường trực mỗi hàng, vùng chạm ≥ 44 px; không chữ lặp | E2E |
| TC-UX12-8 | AC3 | — | `rg -n "0[,.]25" frontend/src/features/attendance` | Không viết cứng 0,25 (đọc từ công thức) | GREP |
| TC-UX12-9 | AC4 | giảng viên, 375 | Cuộn tới hàng 20; đổi 1 SV sang Vắng | Thanh dính đáy «Lưu điểm danh · 29 có mặt · 1 vắng» (số cập nhật theo lựa chọn), nút chính ≥ 44 px, nằm **trên** thanh điều hướng đáy | E2E |
| TC-UX12-10 | AC4 | giảng viên, 375 | Cuộn xuống hàng cuối | Hàng cuối không bị thanh che (cách thanh ≥ 0 px) | E2E |
| TC-UX12-11 | AC4 | giảng viên, 375 | Bấm nút trên thanh | Buổi được lưu; trạng thái đổi «Đã lưu HH:mm» | E2E |
| TC-UX12-12 | AC4 (idempotency) | giảng viên | Bấm `Lưu điểm danh` hai lần nhanh | Một bản ghi buổi (không nhân đôi phát biểu / điểm cộng) | E2E |
| TC-UX12-13 | AC5 | giảng viên, 1440 | Đọc vị trí `Lưu điểm danh` + trạng thái; cuộn bảng | Nút và «Chưa có thay đổi» / «Đã lưu HH:mm» nằm trên cùng cột với đầu bảng; tiêu đề cột và cột tên dính khi cuộn | E2E |
| TC-UX12-14 | AC6 | giảng viên, 1440 | Lớp thử, công thức chưa xác nhận | Đúng một dòng cảnh báo mỏng phía trên bảng + nút `Xem công thức` | E2E |
| TC-UX12-15 | AC6 | giảng viên, 375 | như trên | Nhãn «Công thức chưa xác nhận» + nút trên **một hàng**, cao ≤ 56 px | E2E |
| TC-UX12-16 | AC7 | giảng viên | `context.setOffline(true)`; đổi trạng thái, bấm lưu | Hàng đợi cục bộ giữ thay đổi; dải mất mạng; thanh lưu «Chờ mạng»; bật mạng → tự đồng bộ, dữ liệu lên máy chủ đủ | E2E |
| TC-UX12-17 | AC8 | hai giảng viên | A và B cùng sửa một buổi; B lưu sau A | 409 → hiện giá trị của A và hai lựa chọn; lựa chọn của B không mất | E2E |
| TC-UX12-18 | AC9 | STUDENT, TA | `/attendance` | STUDENT 403; TA điểm danh được | E2E |
| TC-UX12-19 | AC10 | giảng viên, 375 | Kịch bản: mở `/attendance`, chạm 2 SV sang Vắng, bấm lưu; ghi giờ bằng đồng hồ; đếm hộp thoại xác nhận | Hoàn tất ≤ 60 s (tự động ≈ vài giây; QC thao tác tay thêm 1 lượt và bấm giờ); 0 hộp thoại xác nhận | E2E + TAY |
| TC-UX12-20 | AC11 | giảng viên | Đổi lựa chọn, đo thời gian đổi chấm; `reduce` | ≤ 100 ms, không hoạt cảnh; `reduce` như nhau; sau lưu có nháy xanh theo US-04 AC7 | E2E |
| TC-UX12-21 | AC12 | — | `attendance.spec.ts` diff + chạy; ảnh `attendance-1440`, `attendance-390` | Sửa theo AC1–AC5, không nới; có ảnh mốc | E2E |
| TC-UX12-22 | (K7) | — | Chụp trước (`audit/shots/attendance__teacher__*`) và sau | Đủ ảnh | MAC |
| TC-UX12-PQ | (kiểm chéo) | giảng viên khác lớp | `POST /courses/‹lớp lạ›/attendance` | 403 / 404; không ghi | E2E |

---
## Bảng phủ AC đợt 4
| Story | AC | TC |
| --- | --- | --- |
| US-14 | AC1 | 14-1, 2, 3 |
| | AC2 | 14-4 |
| | AC3 | 14-5, 6 |
| | AC4 | 14-7 |
| | AC5 | 14-8 |
| | AC6 | 14-9 |
| | AC7 | 14-10 |
| | AC8 | 14-11 |
| | AC9 | 14-12 |
| US-15 | AC1 | 15-1, 2 |
| | AC2 | 15-3, 4, 5 |
| | AC3 | 15-6, 7 |
| | AC4 | 15-8, 9, 10 |
| | AC5 | 15-11 |
| | AC6 | 15-12 |
| | AC7 | 15-13, 14 |
| | AC8 | 15-15 |
| | AC9 | 15-16, 17, 18 |
| | AC10 | 15-19 |
| | AC11 | 15-20, 21 |
| US-16 | AC1 | 16-1, 2, 3 |
| | AC2 | 16-4, 5 |
| | AC3 | 16-6, 7, 8 |
| | AC4 | 16-9, 10, 11 |
| | AC5 | 16-12 |
| | AC6 | 16-13, 14 |
| | AC7 | 16-15, 16, 17 |
| | AC8 | 16-18 |
| | AC9 | 16-19, 20 |
| | AC10 | 16-21 |
| | AC11 | 16-22, 23 |
| US-12 | AC1 | 12-1, 2 |
| | AC2 | 12-3 |
| | AC3 | 12-4…8 |
| | AC4 | 12-9, 10, 11, 12 |
| | AC5 | 12-13 |
| | AC6 | 12-14, 15 |
| | AC7 | 12-16 |
| | AC8 | 12-17 |
| | AC9 | 12-18 |
| | AC10 | 12-19 |
| | AC11 | 12-20 |
| | AC12 | 12-21, 22 |

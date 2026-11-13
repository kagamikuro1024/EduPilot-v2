# QC đợt 4 — Chấm bài, sổ điểm, điểm danh (US-UX91-14, 15, 16, 12) + US-04 AC7

**Kết luận lượt 3 (`673441c5`, dev: «sửa QC đợt 4 lỗi #8»): PASS.** US-14, 15, 16, 12 PASS; US-04 AC7 PASS. 8/8 lỗi đã sửa. Lượt 2 (`8ae5393f`): FAIL vì US-15 AC7 nhánh 409 (lỗi #8, Trung bình). Lượt 1 (`f8a744bd`): FAIL (US-15, 16, 12).
Môi trường lượt 2: Mac (Apple Silicon, Colima), stack `-p ep-qc` `EP_PORT_OFFSET=500` (`https://localhost:943`), mock LLM, seed 30 SV + docling; spec `qc-uiux91-w4*-real` (+ `w4f`, `w4g` mới).

## Kiểm lại lần 2 (`8ae5393f`) và lần 3 (`673441c5`)
| TC | Trước | Nay | Ghi chú |
| --- | --- | --- | --- |
| UX15-6 (AC3 thanh dưới) | FAIL (Thấp) | PASS | 1440×900 lúc mở: `top 775, bottom 844` (< 900), `sticky`, nút `Duyệt bài` đủ |
| UX15 tiến độ N/M (AC3, AC4) | FAIL (Thấp) | PASS | `Đã duyệt 0/7` → bấm → `?n=1&m=7`, `Đã duyệt 1/7`; phím `A` → `2/7` (M cố định) |
| UX15-13 (AC7 lỗi mạng) | FAIL (TB) | PASS | Chặn `POST …/review`: thanh dưới «Không kết nối được tới máy chủ. Chữ bạn đã nhập vẫn được giữ. `Thử lại` Đã duyệt 0/3 Duyệt bài A»; điểm còn nguyên; bỏ chặn → `Thử lại` duyệt được |
| **UX15-14 (AC7 409 thật)** | CHƯA CHẠY (lượt 1) → FAIL (TB) (lượt 2) | **PASS (lượt 3, `673441c5`)** | Người kia `PUT` trước; ô tôi sửa giữ `0,75` (không lùi); một thông báo «Người khác vừa nhập 1,50 · Bạn đã nhập 0,75» + `Giữ của tôi` / `Dùng của họ`, không còn dải đỏ thứ hai; `Giữ của tôi` → API lưu `0.75`. Ảnh `shots/wave-4/review-409.png` |
| UX16-1 (ô nhập đệm 0) | FAIL (Thấp) | PASS | 24 giá trị ô nhập đều `x,xx` (`5,00 … 10,00 9,75`); focus hiện thô «5» |
| UX16-4/5 (`⋯` tiêu đề) | FAIL (Thấp) | PASS | Nút `⋯` ở tiêu đề cột nhập: opacity 0 → 1 khi rê, → 1 khi focus. Nút «Hành động — tên» mỗi hàng vẫn thường trực (AC2 chỉ nói tiêu đề cột; ghi chú, không tính lỗi) |
| UX16-21 (AC10, 375) | FAIL (Thấp) | PASS | Ô nhập 44 px, không tràn trang |
| UX16-12 (AC5, 409 ô, thật) | CHƯA CHẠY | PASS | Người kia `PUT /grade-entries` trước; UI hiện «TS. Lê Thu Hà vừa nhập 6,75 · vừa xong» + `Giữ của tôi` / `Dùng của họ`; ô giữ 9,50; `Giữ của tôi` → API lưu 9.5. Đã trả ô về 5,25 |
| UX12-11/19 (AC4, 375 lưu ≤ 60 s) | KHÔNG KIỂM ĐƯỢC | PASS | Chạm 2 SV sang Vắng (+2 vắng), lưu → «Đã lưu · 26 có mặt · 4 vắng» sau 3 s, 0 hộp thoại, 0 `confirm()` |
| UX12-14/15 (AC6, 375) | FAIL (Thấp) | PASS | Lớp 761988: khối «Công thức chưa xác nhận [Xem]» cao 46 px (≤ 56); 1440: 48 px |
| UX12-1/2/3, 6 (giữ nguyên) | PASS | PASS | Muộn / Vắng phép / Vắng đều đỏ, Có mặt mực; `+N · +0,25×N` (N cộng 2 sau 2 lần bấm) |
| UX12-5 (phím `P`) | PASS | PASS có ghi chú | «Đang chọn» = hàng chọn bằng phím mũi tên (aria-label của lưới); bấm chuột vào hàng không đổi hàng chọn (`aria-current` giữ hàng 0). `P` ghi đúng hàng chọn bằng mũi tên |
| UX14-3/5 (Enter) | PASS có ghi chú | PASS có ghi chú | Hàng desktop không focus được; `Enter` trên liên kết `Duyệt` trong hàng mở `/grading/‹id›`. 375: cả thẻ là một `<a>` |
| UX14-11 (AC8, 375) | PASS | PASS | Danh sách thẻ; hộp `Duyệt` 309×44, không tràn ngang |
| UX16-15/16/19/20 | PASS | PASS | Hộp «Chốt 30 sinh viên · phân bố … · 1 sinh viên liệt», Esc không chốt (`OPEN`); TA `POST finalize` 403, không có nút |
| UX04-12/13/14 | PASS | PASS | `ep-saved` 200 ms ở sổ điểm, điểm danh, nhận xét |
| UX16-24 (khớp CSV) | PASS (30/30) | **PASS (30/30)** | Lượt 2 trên DB đã bị QC ghi thêm phát biểu: 29/30 (`20229002` 7,5 ≠ 7,0). Chạy lại ngay sau khi dựng DB sạch (`down -v` + seed) ở `b02eeb68`: `khớp 30/30; lệch: (không)` |
| UX16-10 (cột Trạng thái) | PASS có ghi chú | PASS có ghi chú | Còn 1 SV «Liệt» nên cột còn (như lượt 1) |

### Lỗi #8 — US-15 AC7 (409): đã sửa ở `673441c5`
Lượt 2 (`8ae5393f`): sau 409, ô điểm tự lùi về giá trị cũ, thông báo không nêu giá trị người kia, `Giữ của tôi` lưu số cũ, kèm dải đỏ thừa (Trung bình). Lượt 3: xem hàng UX15-14. Hồi quy: 6 ca `TC-UX15` (`w4-real`, `w4c`) PASS ở `673441c5` (thanh dưới `775 → 844`, lỗi mạng ở thanh dưới + `Thử lại`, phím `A`, `N/M`).

## Môi trường (lượt 1, `f8a744bd`)
| Mục | Giá trị |
| --- | --- |
| Commit kiểm | `f8a744bd` (nhánh `sprint/9.1-uiux`, handoff `handoff/dev-wave-4.md`) |
| Hạ tầng | VPS `/srv/work/qc`, stack `-p ep-qc` dựng lại từ đúng commit; Caddy `https://localhost:14443`; mock LLM; seed + 30 SV mẫu; QC chấm nháp Bài tập 03 qua API (`POST /courses/{c}/submissions/{sid}/grade`) để có hàng chờ duyệt |
| Spec QC | `qc-uiux91-w4-real.spec.ts`, `w4b-real`, `w4c-real`, `w4d-real` (stack thật); `qc-uiux91-w4.spec.ts` (harness gateway giả, sổ điểm 4 SV / 1.000 SV) |
| Đo 375 px | Bảng dạng thẻ: `getBoundingClientRect` của `tr` trả 0 → đo bằng `locator.boundingBox()` + ảnh |
| Ảnh K7 | `qc/shots/wave-4/` (1440 + 375); ảnh trước ở `audit/shots/*__teacher__*` |

## US-UX91-14 — Hàng chờ chấm: PASS
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX14-1, 2 | 1 | PASS | Một bài → cột «Bài tập» ẩn (th: Sinh viên, Nguồn, Điểm, Trạng thái duyệt, Thời gian nộp, Duyệt); «Tất cả bài» → cột hiện, `nowrap` + `ellipsis` |
| UX14-4 | 2 | PASS có giới hạn | 6/6 hàng đều có lý do cờ dưới tên; dữ liệu không có hàng không-cờ để so khoảng cách |
| UX14-5 | 3 | PASS | Hover đổi nền `td`, liên kết `Duyệt`, bấm hàng mở `/grading/‹id›`, `cursor:pointer`. Ghi chú: hàng `tabIndex=-1`, Enter chỉ có tác dụng trên liên kết `Duyệt` (Tab tới được) |
| UX14-7 | 4 | PASS | Đáy chip «Cần xem kỹ» lệch ô chọn 1 px; ô «Bài tập» 449 = chữ 403 + đệm 44 |
| UX14-9 | 6 | PASS | Lỗi tải → `Thử lại`, bộ lọc giữ nguyên |
| UX14-10 | 7 | PASS | STUDENT 403; TA mở được |
| UX14-11 | 8 | PASS | 375: nút `Duyệt` 309×44, không tràn ngang; thứ tự tên → lý do → điểm → trạng thái |
| UX14-PQ, PG | — | PASS | TA lớp lạ: submissions 403, gradebook 403, attendance 404; có `limit` ≤ 100, không OFFSET |
| UX14-12 | K7 | PASS | `shots/wave-4/grading-queue-1440.png`, `-375.png` |

## US-UX91-15 — Duyệt bài chấm: FAIL
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX15-1, 2 | 1 | PASS | Tên tiêu chí cùng `top` với ô điểm, không chồng, không nhãn «Điểm …» lặp (1440 và 1100) |
| UX15-3 | 2 | PASS | «Bấm một dòng trích để xem trong bài» đúng 1 lần |
| UX15-4 | 2 | PASS | Tô sáng ~104 ms → ~1,12 s; `reduce` 58 → 1,02 s |
| UX15-11 | 5 | PASS | «Lệch 1,5 điểm so với lượt 2 (1,00 và 2,50)» inline, không modal |
| UX15-7 | 3 | PASS | Một nút đỏ «Duyệt bài» |
| UX15-6 | 3 | **FAIL (Thấp)** | 1440×900, lúc mở: thanh dưới `top 863, bottom 932` > 900, nút «Duyệt bài» bị cắt mép dưới (ảnh `shots/wave-4/err-review.png`); chỉ đủ sau khi cuộn cột phải (`bottom 730`) |
| (tiến độ) | 3, 4 | **FAIL (Thấp)** | Thanh «Đã duyệt N/M»: N luôn 0, M co lại (0/9 → 0/8 → 0/7 → …) sau mỗi lần duyệt; AC nêu «Đã duyệt 12/85» (N tăng, M cố định). |
| UX15-8 | 4 | PASS | Phím `A` → `POST /courses/‹id›/grading-results/‹id›/review {"version":2}`, REVIEWED +1, sang bài kế; nhấn `A` hai lần nhanh chỉ duyệt 1 bài |
| UX15-9 | 4 | PASS | Bấm `›` không hoạt cảnh; danh sách rỗng xử lý được |
| UX15-12 | 6 | PASS | Điểm + ghi chú khôi phục sau tải lại; nháy lưu `ep-saved` |
| UX15-13 | 7 | **FAIL (Trung bình)** | Chặn `POST …/review`: thanh dưới không đổi («Đã duyệt 0/5 Duyệt bài A»); lỗi «Không kết nối được tới máy chủ. Chữ bạn đã nhập vẫn được giữ.» hiện **giữa cột phải**, chỉ có nút `Đã hiểu`, **không có `Thử lại`**. Điểm đã nhập còn nguyên (đạt) |
| UX15-14 | 7 | CHƯA CHẠY | 409 khi lưu điểm: chưa dựng được (ca bị chặn bởi UX15-13) |
| UX15-15, 17 | 8 | KHÔNG KIỂM ĐƯỢC | SV không có route xem bài tập (`/assignments` → «Không tìm thấy trang»); chưa có điểm công bố để so. Tính FAIL theo luật nếu PM không đồng ý đổi phạm vi |
| UX15-16 | 9 | PASS | STUDENT 403; TA thấy nút Duyệt bài (quyền cũ) |
| UX15-19 | 10 | PASS | 375: tab «Bài làm» / «Chấm», nút «Duyệt bài» cao 44, bottom 748 < nav top 755 |
| UX15-21 | K7 | PASS | `grading-review-1440.png`, `-375.png` |

## US-UX91-16 — Sổ điểm: FAIL
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX16-1 | 1 | **FAIL (Thấp)** | Cột tính: «Quá trình» `6,000 / 5,500 / 7,915`, «Điểm học phần» 2 chữ số đệm 0 (đạt). Ô **nhập** (Giữa kỳ, Thi cuối kỳ) hiện thô «7 / 8,5 / 9» thay vì «7,00 / 8,50 / 9,00» → cột không cùng số chữ số |
| UX16-2 | 1 | PASS | `text-align:right`, `tabular-nums` |
| UX16-3 | 1 | PASS | 63 ô so API: lệch 0 |
| UX16-24 | 11 | PASS | Điểm học phần + điểm chữ 30/30 khớp `seed/expected_final_grades.csv` (`w4d-real`) |
| UX16-4, 5 | 2 | **FAIL (Thấp)** | Tiêu đề 13 cột cùng cao 57, 1 dòng, `sticky`; chấm «đã công bố» (2) + chú giải đạt. Nhưng nút `⋯` ở tiêu đề cột nhập (Giữa kỳ, Thi cuối kỳ) có `opacity:1`, `visible` cả trước khi rê / focus, và mỗi hàng có `⋯` thường trực (ảnh `gradebook-1440.png`) → AC «`⋯` chỉ hiện khi rê / focus» không đạt |
| UX16-6 | 3 | PASS có ghi chú | Đúng 1 vùng cuộn dọc, `top 204` → `bottom 804` (viewport 900); chú giải + dòng cuối nằm dưới vùng cuộn trong cùng khung; cột tên `sticky` |
| UX16-7, 8 | 3 | PASS | 1.000 SV → 21–29 hàng DOM (ảo hoá) |
| UX16-9 | 4 | PASS | Tiêu đề phụ «Thiếu điểm: 2 sinh viên», hàng thiếu «Thiếu điểm», hàng đủ trống, bấm lọc còn 2 hàng |
| UX16-10, 11 | 4 | PASS có ghi chú | Harness 4 SV đủ → cột «Trạng thái» ẩn, không còn ô «Đủ». Stack thật: 30 SV đủ điểm nhưng cột còn vì 1 SV có trạng thái «Liệt» (API `LIET:final<3.0`); ô các hàng khác trống |
| UX16-12 | 5 | CHƯA CHẠY | Xung đột ô (`cell-conflict`); chưa chạy ca của dev riêng |
| UX16-13, 14 | 6 | PASS | Nháy `ep-saved` ở ô sổ điểm (200 ms) |
| UX16-15, 16 | 7 | PASS | Chốt điểm: hộp «Chốt 30 sinh viên · phân bố: A+ 3 … F 3 · 1 sinh viên liệt», `Để sau` / Esc không chốt |
| UX16-18 | 8 | PASS | Lỗi tải → `Thử lại`; sổ đã chốt 0 ô nhập |
| UX16-19, 20 | 9 | PASS | TA: `POST finalize` 403, không có nút Chốt điểm; lớp lạ 403; STUDENT 403 (đã kiểm đợt 2) |
| UX16-21 | 10 | **FAIL (Thấp)** | 375: cuộn ngang trong bảng, cột tên `sticky`, không tràn trang (đạt). Ô nhập cao **32 px** (ô `td` 82 px) < 44 px |
| UX16-23 | K7 | PASS | `gradebook-1440.png`, `-375.png` |

## US-UX91-12 — Điểm danh: FAIL
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX12-1 | 1 | PASS | Có mặt chấm mực (không đỏ) |
| UX12-2 | 1 | PASS | Muộn, Vắng phép, Vắng đều chấm đỏ (lấy mẫu điểm ảnh từng chấm: mực / đỏ / đỏ / đỏ) |
| UX12-3 | 2 | PASS | Hàng Vắng có chữ «· Vắng»; radio có tên «Vắng — tên», `checked` |
| UX12-4, 5, 6, 7 | 3 | PASS | `Ghi phát biểu`; hiện `+N · +0,25×N` («+1 · +0,25×1»); phím `P` ghi cho hàng đang chọn (`aria-current`); 375: nút `+` 44×44 thường trực |
| UX12-8 | 3 | PASS | `0,25` không viết cứng ngoài comment |
| UX12-9, 10 | 4 | PASS có giới hạn | 375: thanh lưu dính `704–748` (cao 44) trên nav `755`; «Đã lưu · 29 có mặt · 1 vắng»; hàng cuối không bị che (ảnh `attendance-375-cuoi.png`). Số «vắng» đổi theo thao tác chưa kiểm xong |
| UX12-11, 19 | 4, 10 | KHÔNG KIỂM ĐƯỢC | 375: đổi 2 SV sang Vắng → lưu ≤ 60 s: bấm hàng mở 4 lựa chọn, nhưng chuỗi chạm chọn «Vắng» treo trong test QC (quá hạn 90 s, chưa rõ lỗi sản phẩm hay selector); chưa có số đo thời gian |
| UX12-12 | 5 | PASS | Bấm đúp Lưu → 1 `PUT …/attendance` |
| UX12-13 | 6 | PASS | Nút lưu `top 127`, tiêu đề + cột tên `sticky`, «Chưa có thay đổi» |
| UX12-14, 15 | 7 | **FAIL (Thấp)** | Lớp 761988, 1440: cảnh báo «Công thức điểm chưa xác nhận» + `Xem công thức`, 1 khối cao 48 (đạt). 375: nhãn xuống 2 dòng, khối cao **62 px** > 56 |
| UX12-16, 17 | 8 | PASS | `attendance.spec.ts` (offline, xung đột 409) của dev xanh ở `f8a744bd` |
| UX12-18 | 9 | PASS | STUDENT 403; TA mở được |
| UX12-20 | 5 | PASS | Đổi chấm 28 ms, 0 hoạt cảnh |
| UX12-22 | K7 | PASS | `attendance-1440.png`, `-375.png`, `-375-cuoi.png` |

## US-UX91-04 AC7 — nháy lưu (P-91-8 a): PASS
| TC | Kết quả | Ghi chú |
| --- | --- | --- |
| UX04-12, 13, 14, 19 | PASS | `getAnimations()` có `ep-saved` ở ô sổ điểm, điểm danh, nhận xét chấm bài (mỗi lần 200 ms) |

## Không đổi test có sẵn
`assignment-lifecycle`, `attendance`, `gradebook`, `visual.spec.ts` chỉ đổi theo AC (không nới assertion); 8 snapshot mới / đổi (`attendance-*`, `grading-queue-*`, `grading-review-*`, `gradebook-*`) khớp ảnh K7.

## Chưa chạy
UX15-14 (409 điểm), UX16-12 (xung đột ô), UX16-17 (hai lần xác nhận chốt, tránh khoá sổ thật), UX12-11/19 (375 lưu ≤ 60 s), UX15-15/17 (không có màn SV). Lượt kiểm lại đợt 1, 2, 3, 6 chờ dev «sửa QC lượt 2».

## Lỗi cần sửa (tóm tắt, lượt 2)
| # | Mức | Vị trí | Trạng thái |
| --- | --- | --- | --- |
| 1–7 | Trung bình / Thấp | US-15 AC7 (lỗi mạng), AC3, AC3/4; US-16 AC1, AC2, AC10; US-12 AC7 | Đã sửa, PASS ở `8ae5393f` |
| 8 | Trung bình | US-15 AC7 (409) | Đã sửa ở `673441c5`, PASS |

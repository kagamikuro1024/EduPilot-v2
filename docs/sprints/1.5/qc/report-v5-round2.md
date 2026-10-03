# QC report v5 vòng 2 — sprint 1.5

Bản dựng: worktree riêng @ `b15a5ba` (spec v5.2), `pnpm -C frontend build`, `next start -p 3400`, Chrome for Testing headless riêng. Đo 2026-10-03. Server đã tắt.

**Kết luận: PASS — BUG-v5-01-6 đã đóng ở lượt chạy lại cuối report; DEMO 15 phút PASS.**

## Cổng đã chạy
| Cổng | Kết quả |
| --- | --- |
| `pnpm -C frontend lint` · `bash scripts/ui-antipatterns.sh` | rc=0 · rc=0 (0 dòng ✗) |
| `proto-curl.sh all` | **497 PASS / 0 FAIL** |
| `regress-v24.mjs` | **30/30 PASS** (TC-00-69, 01-153, 02-90, 02-91) |
| `threads-timeline.mjs` | 62 dòng, 0 FAIL |
| `pii-matrix.mjs` | 48/48 PASS |
| `audit.mjs` (SV 165, GV 170, TA 106, Admin 54 dòng; gồm `wide`) | **495 dòng, 0 FAIL** (v5: 9 + 6 FAIL) |
| `demo-run.mjs` (DEMO 15 phút, chuột thật) | **22/22 nhóm bước PASS**, 170,4 s (ngưỡng 13:45) |
| Chạy tay các TC FAIL của v5 | xem bảng; script tay chạy trong Chrome riêng, đo hộp / văn bản |

## BUG → PASS/FAIL
| Mã | Lỗi | TC kiểm | Kết quả | Bằng chứng |
| --- | --- | --- | --- | --- |
| BUG-v5-00-1 | state hỏng → mọi route sự cố | TC-00-58 | PASS | 4 giá trị hỏng × `/`,`/inbox`,`/students`: 0 màn sự cố, có `nav` |
| BUG-v5-00-2 | 404/lỗi ngoài khung | TC-00-52, 59 | PASS | 4 vai: có `nav`, header, tiêu đề left 240 (390 px: 16) |
| BUG-v5-00-3 | nhãn D3 đứng yên, chuông thiếu D3 | TC-00-62, 63 | PASS | chuông + nhãn: vừa gửi → 58 phút → 59 phút → hôm qua 09:21; thật 2 phút → "2 phút trước" |
| BUG-v5-00-4 | "23 giờ" ghi "hôm qua" | TC-00-62 | PASS (đóng theo #26) | +23 giờ "hôm qua 09:21" đúng luật ngày lịch v5.2; TC-00-62 đã sửa chữ |
| BUG-v5-00-5 (#24b) | nút hồ sơ ghi tên vai | TC-00-69 | PASS | `regress-v24` 12/12 |
| BUG-v5-00-6 | `data-scroll-x` trên vùng không cuộn | TC-00-47 | PASS | 14 route × 390/375: mọi vùng `scrollWidth > clientWidth` |
| BUG-v5-DEMO-12 | Đặt lại giữ lớp 761988 | TC-DEMO-01… | PASS | demo-run chạy lại từ Admin, đặt lại về 761987 |
| BUG-v5-01-1 (cao) | `/threads` 390/375 nở khung 772 px | TC-01-15 | PASS | audit student/teacher/ta: 0 FAIL kể cả `wide` |
| BUG-v5-01-2 (#24c) | chip chủ đề cắt | TC-01-153 | PASS | `regress-v24` 8/8 |
| BUG-v5-01-3 | nguồn D1 chỉ 1, thu gọn | TC-01-04 | PASS | "Nguồn tham khảo (2)" mở sẵn: Quy chế tr. 2 + Sổ điểm danh lớp 761987 |
| BUG-v5-01-4 | What-if trống báo lỗi | TC-01-18 | PASS | trống: `aria-invalid` null, 0 alert; 11/abc: lỗi tại ô; 8,0 → 8,1 |
| BUG-v5-01-5 = DEMO-2 | trích đoạn BT03 không phải của B | TC-01-33 | PASS | 4 tiêu chí; đoạn trích SolarWinds/SBOM/Golden SAML; form xem lại mở |
| BUG-v5-01-6 | nhãn lịch tháng cắt … | TC-01-143 | **FAIL** | 1100 px, Tháng: nhãn QUIZ01 hết dấu … nhưng rộng 101 px và xuống **3 dòng** (Range rects), TC đòi tối đa 2 dòng; handoff ghi "đủ 2 dòng" — lệch 1 dòng |
| BUG-v5-DEMO-4 | mất câu trả lời GV sau `Đã rõ` | TC-DEMO-14/15 | PASS | demo-run PASS ("nhãn giảng viên + đóng câu hỏi") |
| BUG-v5-DEMO-7 | SV D lớp 2 thấy Buổi 10 | TC-DEMO-05 | PASS | demo-run PASS |
| BUG-v5-DEMO-11 | thân tin nhắn ≠ chip | TC-DEMO-06 | PASS | D1: "vắng 2 / +0,75" nhất quán |
| BUG-v5-02-1 | badge + thẻ kẹt 3 | TC-02-04, 61, 62 | PASS | duyệt 4 bài: badge 3→2→1→ẩn, thẻ biến; Hôm nay 6→2 việc (còn setup, members của lớp 2) |
| BUG-v5-02-2 = DEMO-3 | link thẻ thiếu `course=` | TC-02-52 | PASS | từ 761988 bấm Điểm danh / Trả lời → 761987 đúng; link có `course=int1006-1` |
| BUG-v5-02-3 (#24a) | tab inbox 1440 co 3 px | TC-02-90, 38 | PASS | `regress-v24` 8/8; tab cao 30, lọc 5/1/0/6 hàng |
| BUG-v5-02-4 (#24d) | ô Buổi cắt | TC-02-91, 68 | PASS | `regress-v24` 2/2; 375: "Buổi 9 · 22/10" 106 ≤ 151, "Buổi 11 · 05/11" 104 ≤ 166 |
| BUG-v5-02-5/6 | chuông thiếu yêu cầu lớp 2; TA nhận thông báo lớp 2 | TC-02-55, 56, 22 | PASS | GV 761987 thấy "xin vào lớp 761988" → `/class/members?tab=pending`, chooser 761988; TA không có 761988; TA lớp 1 không có nút cấm (TC sửa theo #25) |
| BUG-v5-02-7 | hàng ticket cắt đáy | TC-02-57 | PASS | hàng [751, 875] trong khung [230, 876] |
| BUG-v5-02-8 | /students thiếu lý do | TC-02-76 | PASS | câu "Vắng 5/9 buổi, đã bị trừ 1,5 điểm; thiếu Bài tập 02" có ở Hôm nay, `/students`, `/students/sv-3` |
| BUG-v5-DEMO-1 (chặn) | mất trạng thái chốt khi giả lập mất mạng | TC-DEMO-20 | PASS | lưu → bật mất mạng → đổi ô → tắt: dải "Đã hoàn tất buổi 10" còn, "Đã lưu", QT B 8,5 |
| BUG-v5-DEMO-6 | Hôm nay còn việc điểm danh | TC-DEMO-16..18 | PASS | demo-run PASS; sau lưu thẻ điểm danh biến |
| BUG-v5-DEMO-8 | sổ điểm lớp 2 "24 sinh viên" | TC-DEMO-27..30 | PASS | demo-run PASS (sổ điểm lớp 2 theo roster) |
| BUG-v5-03-1 = DEMO-5 | ô làm tròn còn 1 ký tự | TC-03-13, DEMO-29 | PASS | gõ 120 ms/ký tự: ô giữ "Làm tròn đến 0,1"; hộp thoại "… làm tròn đến 0,1."; nút sticky |
| BUG-v5-03-2 | `/documents` lớp 2 lẫn quy chế lớp 1 | TC-03-41 | PASS | lớp 2: 11 tài liệu, không có quy chế 761987; lớp 1: 12 |
| BUG-v5-03-3 (30/31/32/34) | bảng 390/720 mất cột, cắt | TC-03-30, 31, 32, 34 | PASS | 390 `/gradebook`: col-qt 255, col-status 341 ≤ 390, 0 cột width 0; nhãn dòng phụ đủ ở `/documents`, `/grading`, `/students`, `/admin/users`; 720/719: 0 phần tử cắt ngoài vùng cuộn, 0 tràn |
| BUG-v5-03-6 | công bố trên lựa chọn ẩn | TC-03-xx (tay) | PASS | tích B → bật chip `Chưa duyệt`: B bị ẩn, nút `Công bố` tắt (chỉ bật khi B hiện) |
| BUG-v5-03-7, 03-8 | FAILED "Chờ xử lý"; `?state=empty` còn số | — (tay) | PASS | `Thử tệp mẫu lỗi` → "Không dùng được"; `/documents?state=empty` "0 tài liệu · 0 đang dùng cho AI" |
| BUG-v5-03-9, 03-10, DEMO-9 | giá trị điểm sai, nút +, nhật ký sửa điểm | — | Chưa đo tay riêng | chỉ máy đo (audit 0 FAIL; demo-run chấm + công bố PASS) |
| BUG-v5-04-1 | lý do xem nội dung không báo lỗi | TC-04-09 | PASS | 4 khoảng trắng + Tab: `aria-invalid=true`, "Cần nhập lý do…", nút Mở nội dung tắt |
| BUG-v5-04-2 | /analytics 7→30 ngày lệch | TC-04-19 | PASS | 30 ngày: 31 = 19+7+5; lớp 761988: 5 = 3+2 |
| BUG-v5-04-3, 04-7 | rỗng chung / số đếm | TC-04-24 | PASS | 8 route `?state=empty`: 0 màn "Chưa có gì ở đây" |
| BUG-v5-04-4/5 | /admin/users 720, /observability 375 | TC-04-33 | PASS | `/observability` 390/375 GV + Admin: innerWidth không nở; tiêu đề ≥ 343 px |
| BUG-v5-04-6 | gạch thừa panel embedding | — | Chưa đo tay riêng | chỉ máy đo |
| BUG-v5-DEMO-10 | khối rỗng ~270 px | — (tay) | PASS | `/inbox` lớp 761988: tiêu đề → câu giải thích cách nhau 8 px |

Đếm: 49 BUG dev nhận sửa (BUG-v5-00-4 đóng theo #26). Đã kiểm và PASS: toàn bộ lỗi **cao** của v5 (00-1, 01-1, 02-1, 02-2, 03-1, 03-2, 04-2, DEMO-1). **FAIL: 1** (01-6). **Chưa đo tay riêng (chỉ có máy đo, không có TC FAIL ở v5 gắn riêng):** BUG-v5-03-9, 03-10, DEMO-9, 04-6.

## Lỗi mới
Không có lỗi sản phẩm mới. Một quan sát: TC-04-19 ban đầu tôi đọc "lớp 2 = 3+0+0" vì nhãn khác ("Yêu cầu vào lớp 2 lần" thay "Bài nộp qua email") — đúng (3+2=5), không phải lỗi.

## Kết luận từng story
| Story | v5 FAIL | Vòng 2 |
| --- | --- | --- |
| US-PROTO-00 | 47, 52, 58, 59, 62, 63, 69 | **PASS** hết (TC-00-62 sửa chữ theo spec v5.2/#26) |
| US-PROTO-01 | 04, 15, 18, 33, 143, 153 | PASS 04, 15, 18, 33, 153; **FAIL TC-01-143** |
| US-PROTO-02 | 04, 38, 52, 55, 57, 61, 62, 68, 76, 90, 91 (22/56 sửa theo #25) | **PASS** hết |
| US-PROTO-03 | 13, 30, 31, 32, 34, 41 | **PASS** hết |
| US-PROTO-04 | 09, 19, 24, 33 | **PASS** hết |
| DEMO | 20, 26, 29, 38 | **PASS** hết |

## Việc cho PM / dev
- BUG-v5-01-6 (thấp): lịch Tháng 1100 px, nhãn "QUIZ01 đóng · Mật mã đối xứng" xuống 3 dòng (cột ngày 101 px); TC tối đa 2 dòng. Chọn: rút nhãn ở Tháng ("QUIZ01 đóng") hoặc đổi TC thành 3 dòng.
- Ghi chú môi trường: ảnh chụp trong Chrome treo nếu màn hình ngủ (đã dùng `caffeinate`); tôi không chụp ảnh ở vòng này, bằng chứng là số đo và văn bản.

## Chạy lại BUG-v5-01-6 (dev sửa `a4a8184`) — 2026-10-03
Bản dựng: worktree riêng @ `a4a8184`, `next start -p 3400`, Chrome riêng. 

| Phép đo | Kết quả |
| --- | --- |
| `audit.mjs` `/calendar` (SV, GV, TA; 1440 → 375 + biên 1100/1099/720/719) | **22 dòng, 0 FAIL** |
| **TC-01-143** Tháng 1440 | nhãn "QUIZ01 đóng", 149 px, 1 dòng, không cắt (`overflow` clip nhưng `scrollWidth ≤ clientWidth`) — **PASS** |
| TC-01-143 Tháng 1100 | "QUIZ01 đóng", 101 px, **1 dòng** (v5: 3 dòng) — **PASS** |
| TC-01-143 Tháng 720 | "QUIZ01 đóng", 58 px, **2 dòng**, ngắt đúng chỗ trắng ("QUIZ01 " / "đóng"), không ngắt giữa từ — **PASS** |
| TC-01-143 Danh sách 1440/1100/720 | "QUIZ01 đóng · Mật mã đối xứng" 247 px, 1 dòng — **PASS** |

**Kết luận: BUG-v5-01-6 ĐÓNG — TC-01-143 PASS.** Lưu ý: ở chế độ Tháng nhãn đã **rút** còn "QUIZ01 đóng" (bỏ "· Mật mã đối xứng"); Danh sách vẫn đủ. Tôi đã sửa chữ TC-01-143 cho khớp. Kết luận vòng 2 cập nhật: US-PROTO-01 **PASS** hết (153/153), 1.5 vòng 2 không còn FAIL.

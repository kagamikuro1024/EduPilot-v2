# Smoke hồi quy sprint 1.5 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm A) · Ngày: 2026-11-15

Tổng TC cũ: 443 · Đạt (P): 21 · Trượt (F): 3 · Không chạy K (a/b/c): 0 · Lỗi thời (D): 419 · Chưa chạy — cần PM quyết (X): 0 · Chưa chạy — ngoài mẫu (N): 0   (P+F+K+D+X+N = 443)
Quy ước: **K(a)** thiết bị không có · **K(b)** cần người thật · **K(c)** ngoài phạm vi đồ án · **D** TC lỗi thời (spec đã đổi hợp lệ, ghi mã đề xuất/quyết định) · **X** chưa chạy vì lệnh phá huỷ/dựng stack thứ hai bị cấm (chờ PM) · **N** chưa chạy: dạng tài liệu, ngoài mẫu smoke «1 chính + 1 lỗi / US».
Cách chạy:
- Ma trận vai × route (SRS PROTO §2) trên app thật: `docs/regression/evidence-1.5/scripts/matrix.mjs` (Playwright, https://localhost:1143, 4 tài khoản seed, 1440 + 375 px) → `matrix-<vai>.json`; kiểm thêm `extras.mjs` (TA /grading, GV /settings/llm, /observability, /insights) → `extras.json`.
- `proto-curl.sh tc_00_static tc_00_15 tc_00_16 tc_00_14` (F=http://localhost:3700) + `scripts/ui-antipatterns.sh` + `pnpm -C frontend lint/build`.
- Cookie vai giả (`ep_demo_role`) không còn trong frontend ⇒ phần lớn TC (D1–D5, mock id, `?state=`, «Đặt lại dữ liệu demo») là D (lỗi thời, D51). Các TC route/RBAC chạy lại trên app thật bằng bản tương đương.

## Tổng hợp theo US
| US | TC tổng | P | F | K(a/b/c) | D | X | N | Ghi chú |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| US-PROTO-00 | 69 | 8 | 3 | 0 | 58 | 0 | 0 | nền/đổi vai/RBAC/đo UI |
| US-PROTO-01 | 153 | 3 | 0 | 0 | 150 | 0 | 0 | SV |
| US-PROTO-02 | 91 | 2 | 0 | 0 | 89 | 0 | 0 | GV/TA |
| US-PROTO-03 | 41 | 1 | 0 | 0 | 40 | 0 | 0 | điểm/chấm |
| US-PROTO-04 | 50 | 7 | 0 | 0 | 43 | 0 | 0 | insights/admin |
| US-PROTO-DEMO | 39 | 0 | 0 | 0 | 39 | 0 | 0 | kịch bản demo |

## Kết quả từng ca
| TC | KQ | Ghi chú |
| --- | --- | --- |
| **US-PROTO-00** | | |
| TC-00-01 | D | cookie ep_demo_role/dải «Bản mô phỏng» đã bỏ (D51); phần chuyển /inbox,/chat,/gradebook,/me → /login vẫn đúng, «/» nay là trang giới thiệu |
| TC-00-02…TC-00-08 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| TC-00-09 | P | `pnpm -C frontend lint` + `build` rc 0; ui-antipatterns sạch |
| TC-00-10 | D | thêm phụ thuộc uqr (10.8) + sessionStorage của mã điểm danh (DESIGN §14.8.2) ngoài luật cũ |
| TC-00-11…TC-00-16 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| TC-00-17 | P | GV/Admin bị chặn ở route SV; SV/Admin bị chặn ở route GV (ma trận 4 vai × 25 route cố định, 1440 px) |
| TC-00-18 | P | màn chặn: «Bạn không có quyền xem màn này», nút «Về Hôm nay», «Trang này dành cho giảng viên và trợ giảng.» |
| TC-00-19 | P | ma trận 4 vai × 25 route cố định: 0 lệch (6 route id mock bỏ: không còn tồn tại) |
| TC-00-20…TC-00-35 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| TC-00-36 | F | DEF-R1.5-02: TA tại «/» tràn ngang +9 px (390) / +24 px (375) — nút hồ sơ ở topbar |
| TC-00-37 | P | 375 px: không tràn ngang trên 25 route × 4 vai (trừ TA «/» — xem TC-00-36) |
| TC-00-38…TC-00-47 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| TC-00-48 | P | ui-antipatterns.sh sạch (rc 0) |
| TC-00-49…TC-00-53 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| TC-00-54 | P | tc_00_15: không backdrop-filter/transparent; antipatterns có mẫu. Bước «thử đỏ» trong worktree tạm không chạy |
| TC-00-55…TC-00-56 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| TC-00-57 | F | DEF-R1.5-01: đã đăng nhập mở đường dẫn không có thật → /login?next=…, không có màn «Không tìm thấy trang» |
| TC-00-58 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| TC-00-59 | F | DEF-R1.5-01: không thấy màn 404 để bấm «Về Hôm nay» |
| TC-00-60 | D | chuỗi «hôm qua 16:40» là dữ liệu mock |
| TC-00-61…TC-00-67 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| TC-00-68 | P | tc_00_16 7/7 |
| TC-00-69 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| **US-PROTO-01** | | |
| TC-01-01 | P | 9 route SV cố định mở được (3 route id mock không còn) |
| TC-01-02…TC-01-25 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| TC-01-26 | P | TA/GV/Admin bị chặn ở route SV; Admin bị chặn /threads, /calendar |
| TC-01-27 | P | ma trận như TC-00-19 |
| TC-01-28…TC-01-99 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| **US-PROTO-02** | | |
| TC-02-01 | P | GV mở /, /inbox, /students, /attendance, /class/members (id mock bỏ) |
| TC-02-02…TC-02-20 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| TC-02-21 | P | SV, Admin bị chặn ở route GV/TA |
| TC-02-22…TC-02-91 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| **US-PROTO-03** | | |
| TC-03-01 | P | GV mở /gradebook, /gradebook/scheme, /grading, /questions, /documents (id mock bỏ) |
| TC-03-02…TC-03-22 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| TC-03-23 | D | câu chữ «Chỉ giảng viên công bố điểm» đã đổi trên màn thật |
| TC-03-24…TC-03-41 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| **US-PROTO-04** | | |
| TC-04-01 | P | GV 5 route + Admin 6 route mở được |
| TC-04-02…TC-04-03 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| TC-04-04 | P | GV /insights không MSSV |
| TC-04-05…TC-04-09 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| TC-04-10 | P | GV /observability không «[[SV_» |
| TC-04-11…TC-04-13 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| TC-04-14 | P | GV /settings/llm không nút «Test kết nối» |
| TC-04-15…TC-04-16 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| TC-04-17 | P | SV/TA bị chặn ở /observability, /settings/*, /admin/*; GV bị chặn /admin/courses |
| TC-04-18 | P | ma trận như TC-00-19 |
| TC-04-19…TC-04-48 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| TC-04-49 | P | tc_00_16 7/7 |
| TC-04-50 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |
| **US-PROTO-DEMO** | | |
| TC-DEMO-00…TC-DEMO-38 | D | mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn) |

## Trượt: bước tái hiện
### DEF-R1.5-01 — đường dẫn không tồn tại → /login dù đã đăng nhập (TC-00-57, 00-59)
### DEF-R1.5-02 — TA: topbar tràn ngang ở 375/390 px (TC-00-36)
(chi tiết: `defects-1.5.md`)

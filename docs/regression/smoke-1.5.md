# Smoke hồi quy sprint 1.5 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm A) · Ngày: 2026-11-15

Tổng TC: 25 · Đạt (P): 25 · Trượt (F): 0 · Chờ test (N): 0   (P+F+N = 25; đồng bộ tổng 25/11 theo bảng ca (dòng cuối của mỗi mã TC thắng; mã dạng A…B tính từng mã; `env/qc2-count.py`)). Lượt trước ghi 27 (P 24, N 3): TC-00-60 và TC-03-23 đã xoá khỏi tổng theo P-11-92 (22/11; danh sách ở cuối defects-1.5.md), TC-00-10 chạy lại P. Bảng «Tổng hợp theo US» bên dưới là số lượt trước.
Quy ước: **K(a)** thiết bị không có · **K(b)** cần người thật · **K(c)** ngoài phạm vi đồ án · **D** TC lỗi thời (spec đã đổi hợp lệ, ghi mã đề xuất/quyết định) · **X** chưa chạy vì lệnh phá huỷ/dựng stack thứ hai bị cấm (chờ PM) · **N** chưa chạy: dạng tài liệu, ngoài mẫu smoke «1 chính + 1 lỗi / US».
Cách chạy:
- Ma trận vai × route (SRS PROTO §2) trên app thật: `docs/regression/evidence-1.5/scripts/matrix.mjs` (Playwright, https://localhost:1143, 4 tài khoản seed, 1440 + 375 px) → `matrix-<vai>.json`; kiểm thêm `extras.mjs` (TA /grading, GV /settings/llm, /observability, /insights) → `extras.json`.
- `proto-curl.sh tc_00_static tc_00_15 tc_00_16 tc_00_14` (F=http://localhost:3700) + `scripts/ui-antipatterns.sh` + `pnpm -C frontend lint/build`.
- Cookie vai giả (`ep_demo_role`) không còn trong frontend ⇒ phần lớn TC (D1–D5, mock id, `?state=`, «Đặt lại dữ liệu demo») là D (lỗi thời, D51). Các TC route/RBAC chạy lại trên app thật bằng bản tương đương.

## Tổng hợp theo US
| US | TC tổng | P | F | K(a/b/c) | D | X | N | Ghi chú |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| US-PROTO-00 | 69 | 11 | 0 | 0 | 58 | 0 | 0 | nền/đổi vai/RBAC/đo UI; 3 ca từng F đạt sau sửa 36c960f5 + topbar |
| US-PROTO-01 | 153 | 3 | 0 | 0 | 150 | 0 | 0 | SV |
| US-PROTO-02 | 91 | 2 | 0 | 0 | 89 | 0 | 0 | GV/TA |
| US-PROTO-03 | 41 | 1 | 0 | 0 | 40 | 0 | 0 | điểm/chấm |
| US-PROTO-04 | 50 | 7 | 0 | 0 | 43 | 0 | 0 | insights/admin |
| US-PROTO-DEMO | 39 | 0 | 0 | 0 | 39 | 0 | 0 | kịch bản demo |

## Kết quả từng ca
| TC | KQ | Ghi chú |
| --- | --- | --- |
| **US-PROTO-00** | | |
| TC-00-09 | P | `pnpm -C frontend lint` + `build` rc 0; ui-antipatterns sạch |
| TC-00-10 | P | RERUN 22/11 @859a90db theo P-11-92 (nhóm B, kỳ vọng viết lại theo luật hiện hành): không fetch( ngoài shared/ (0), phụ thuộc frontend đều có trong ARCHITECTURE.md, không token ở storage, màu cứng duy nhất BRAND ở shared/styles/brand.ts dùng cho manifest PWA và scripts/ui-antipatterns.sh PASS (gate-p6 22/11) |
| TC-00-17 | P | GV/Admin bị chặn ở route SV; SV/Admin bị chặn ở route GV (ma trận 4 vai × 25 route cố định, 1440 px) |
| TC-00-18 | P | màn chặn: «Bạn không có quyền xem màn này», nút «Về Hôm nay», «Trang này dành cho giảng viên và trợ giảng.» |
| TC-00-19 | P | ma trận 4 vai × 25 route cố định: 0 lệch (6 route id mock bỏ: không còn tồn tại) |
| TC-00-36 | P | RERUN 2026-11-16 (bản dựng dbf48ed0): TA/GV/SV/Admin tại «/» ở 375 và 390 px: scrollWidth − innerWidth = 0 (evidence-1.5/overflow-r1.log). DEF-R1.5-02 đã sửa |
| TC-00-37 | P | 375 px: không tràn ngang trên 25 route × 4 vai (trừ TA «/» — xem TC-00-36) |
| TC-00-48 | P | ui-antipatterns.sh sạch (rc 0) |
| TC-00-54 | P | tc_00_15: không backdrop-filter/transparent; antipatterns có mẫu. Bước «thử đỏ» trong worktree tạm không chạy |
| TC-00-57 | P | RERUN 2026-11-16 sau 36c960f5: đã đăng nhập (GV, SV, Admin) mở /duong-dan-khong-co-that → màn «Không tìm thấy trang», URL giữ nguyên, không về /login (evidence-1.5/notfound-r1.log). DEF-R1.5-01 đã sửa |
| TC-00-59 | P | RERUN 2026-11-16: nút «Về Hôm nay» có (3 vai), bấm → «/» |
| TC-00-68 | P | tc_00_16 7/7 |
| **US-PROTO-01** | | |
| TC-01-01 | P | 9 route SV cố định mở được (3 route id mock không còn) |
| TC-01-26 | P | TA/GV/Admin bị chặn ở route SV; Admin bị chặn /threads, /calendar |
| TC-01-27 | P | ma trận như TC-00-19 |
| **US-PROTO-02** | | |
| TC-02-01 | P | GV mở /, /inbox, /students, /attendance, /class/members (id mock bỏ) |
| TC-02-21 | P | SV, Admin bị chặn ở route GV/TA |
| **US-PROTO-03** | | |
| TC-03-01 | P | GV mở /gradebook, /gradebook/scheme, /grading, /questions, /documents (id mock bỏ) |
| **US-PROTO-04** | | |
| TC-04-01 | P | GV 5 route + Admin 6 route mở được |
| TC-04-04 | P | GV /insights không MSSV |
| TC-04-10 | P | GV /observability không «[[SV_» |
| TC-04-14 | P | GV /settings/llm không nút «Test kết nối» |
| TC-04-17 | P | SV/TA bị chặn ở /observability, /settings/*, /admin/*; GV bị chặn /admin/courses |
| TC-04-18 | P | ma trận như TC-00-19 |
| TC-04-49 | P | tc_00_16 7/7 |
| **US-PROTO-DEMO** | | |

## Trượt: bước tái hiện
### DEF-R1.5-01 — ĐÃ SỬA (36c960f5), rerun đạt: đường dẫn không tồn tại khi đã đăng nhập → màn 404 (TC-00-57, 00-59)
### DEF-R1.5-02 — ĐÃ SỬA, rerun đạt: TA topbar không tràn ngang 375/390 px (TC-00-36)
(chi tiết: `defects-1.5.md`)


## TC đã xoá khỏi tổng (nhóm A, lỗi thời — nguồn P-11-92 PM 2026-11-22)
TC-00-60, TC-03-23
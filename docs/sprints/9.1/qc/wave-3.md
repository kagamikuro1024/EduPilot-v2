# QC đợt 3 — Hôm nay (US-UX91-10, 11)

**Kết luận: FAIL.** US-10 FAIL (AC3 với dữ liệu thật; suite có sẵn đỏ 4 ca `panels.spec.ts`). US-11 PASS.

## Môi trường
| Mục | Giá trị |
| --- | --- |
| Commit kiểm | `6a0e8584` (nhánh `sprint/9.1-uiux`; chứa `c7083a3d` sửa QC đợt 1, 2, 6 và đợt 3) |
| Hạ tầng | VPS `/srv/work/qc`, stack `-p ep-qc` dựng lại từ đúng commit trên (frontend build lại; seed giữ nguyên), Caddy `https://localhost:14443`. Dữ liệu cố định F-GV / F-SV bằng `page.route` trên harness e2e (`next start` + gateway giả, bản `build:gate`) |
| Spec QC | `frontend/e2e/qc-uiux91-w3.spec.ts` (harness, 17 ca), `qc-uiux91-w3-real.spec.ts` (stack thật + ảnh) |

## US-UX91-10 — Hôm nay giảng viên: nhóm việc
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX10-1 | 1 | PASS | F-GV: `Gấp · 3` (a1, a2, a3) → `Hôm nay · 2` (a4, a7) → `Khi rảnh · 2` (a5, a6), thứ tự máy chủ giữ trong nhóm |
| UX10-2, 3 | 1 | PASS | Bỏ a5, a6 → không nhóm «Khi rảnh»; chỉ `normal` thường → chỉ «Hôm nay · 2» |
| UX10-4 | 1 | PASS | `EXAM_SIMILARITY` `high` vào «Gấp», `UNMATCHED_SUBMISSION` `normal` vào «Khi rảnh» |
| UX10-5 | 1 | PASS | Chỉ `/me/courses`, `/courses/‹id›/today` (hoặc `/me/today`), `/auth/refresh`, `/events`, `/notifications` (hạ tầng có từ trước); không API mới |
| UX10-6, 7, 8 | 2 | PASS | Đúng 1 nút đỏ (hàng `AI_PENDING`, nhãn «Mở»); các hàng khác `ghost` nền trong suốt, có mũi tên, không gạch chân; bỏ a1–a3 → nút đỏ sang `REPORTED_POSTS` |
| UX10-9 | 2 | PASS | `ui-antipatterns.sh` sạch |
| UX10-10 | 3 | **FAIL (Thấp)** | Harness (tiêu đề không chứa mã lớp): đạt. **Stack thật**, giảng viên một lớp đang chọn (761988): chữ máy chủ vẫn mang mã lớp ở mọi hàng: «Buổi 4 đã kết thúc, chưa điểm danh · 761988», «3 yêu cầu vào lớp 761988 đang chờ duyệt», «3 câu trả lời AI chờ xác nhận · lớp 761988», «Công thức điểm chưa xác nhận · 761988», «Thiết lập lớp mới · 761988» (5/5 hàng; TA: «· lớp 761987»). AC3: «không còn lớp 761987 lặp ở từng hàng». Ảnh `shots/wave-3/hom-nay-teacher-{1440,375}.png`. Tiêu đề trang «N việc cần xử lý hôm nay», không còn khối con trùng tiêu đề (đạt) |
| UX10-11 | 3 | PASS | «Tất cả lớp của tôi»: mã lớp hiện ở hàng và ở phạm vi dưới tiêu đề |
| UX10-12 | 4 | PASS | Chấm: `overdue` đỏ, `high` hổ phách, `normal` xám |
| UX10-13 | 4 | PASS | Tên truy cập `h3`: «Gấp, 3 việc», «Hôm nay, 2 việc», «Khi rảnh, 2 việc», đứng trước hàng đầu |
| UX10-14, 15, 27 | 5, 13 | PASS | 1440: «Lớp cần chú ý», «Sắp tới» ở cột phải (bên phải danh sách, cùng hàng); 1099: nằm dưới danh sách; văn bản cùng dữ liệu ở cả hai bề rộng |
| UX10-16 | 5 | PASS | `attention_total` 7 > 2 hiện → «Xem tất cả (7)» |
| UX10-17 | 6 | PASS có giới hạn | Bấm việc: không hoạt cảnh nào ≥ 300 ms (ngoài chỉ báo tải); `reduce`: không hoạt cảnh (> 1 ms). Sợi chỉ đỏ không bắt được bằng `getAnimations()` (cả harness và stack thật), nên không đo được 280 ms của chính nó |
| UX10-18, 19 | 7 | PASS | Lỗi lần đầu: «Chưa tải được việc hôm nay…» + `Thử lại`, bấm sau khi bỏ chặn tải được; làm mới lỗi: dải «Chưa làm mới được. Đang hiện dữ liệu lần trước.» và giữ nhóm đã hiện |
| UX10-20, 21 | 8, 9 | PASS | `actions: []` → «Bạn đã xử lý hết việc.», không nhóm trống; `count: 40` + 15 mục → «Đang hiện 15 việc đầu tiên.» |
| UX10-22, 23, 24 | 10 | PASS | SV không có `h3` Gấp / Hôm nay / Khi rảnh; admin: việc hệ thống, không nhóm trống (stack thật: «Không có việc cần xử lý hôm nay.»); TA thấy nhóm như giảng viên, không `Bỏ qua` ở `COURSE_SETUP` |
| UX10-25 | 11 | PASS | 375 px: không tràn ngang, nút ≥ 44 px, hàng xếp dọc |
| UX10-26 | 12 | PASS | `today.spec.ts` chỉ thêm (64 dòng, không xoá); thêm ảnh mốc `hom-nay-gv-1440`, `hom-nay-gv-390`; `home-*` không đổi |
| UX10-28 | K7 | PASS | Ảnh sau 1440 / 375 ở `shots/wave-3/`, ảnh trước `audit/shots/home__teacher__1440.png` |
| UX10-PQ | — | PASS | TA / GV gọi `/courses/‹id›/today` hoặc `/me/today` theo phiên, không lớp khác |
| (suite có sẵn) | 12 | **FAIL (Thấp)** | `panels.spec.ts:419` 4 ca đỏ (`/ teacher @1440`, `@1024`, `/ ta @1440`, `@1024`): kỳ vọng 2 Panel («Việc cần xử lý» + «Sắp tới»), nay còn 1 vì US-10 AC3 bỏ khối «Việc cần xử lý». Dev chưa cập nhật spec; handoff chỉ ghi `today.spec`, `visual.spec`. Chạy riêng cô lập vẫn đỏ |

## US-UX91-11 — Hôm nay sinh viên: PASS
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX11-1, 2 | 1 | PASS | Thứ tự: lời chào + ngày → «Việc nên làm tiếp» (y 203) → «Hôm nay» (y 448, trong khung nhìn đầu) → «Tiếp tục học» (y 695) |
| UX11-3, 4 | 2 | PASS | Ba hàng tối đa; «Sâu Morris là gì» một hàng (phiên gần nhất, không phân biệt hoa thường); cùng tiêu đề khác lớp (761987 / 761988) cả hai hiện |
| UX11-5 | 3 | PASS | «Thứ Hai, 00:13» và «Hôm qua, 00:13», không «Thứ 2, 2/11» |
| UX11-6 | 4 | PASS | Đúng 1 nút đỏ (`Mở bài tập`) |
| UX11-7 | 5 | PASS | `sv.moi`: «Nhập mã tham gia lớp», không khối «Hôm nay» trống |
| UX11-8 | 6 | PASS | GV / TA không có «Việc nên làm tiếp» / «Tiếp tục học» |
| UX11-9 | 7 | PASS | 375 px: cùng thứ tự, không tràn ngang, thanh dưới «Hôm nay» `aria-current` |
| UX11-10 | 8 | PASS | Ảnh `hom-nay-sv.gioi-{1440,375}.png`; `visual.spec` chỉ thêm `hom-nay-gv-*` |
| UX11-PQ | — | PASS | Trang không gọi API phiên chat / sinh viên khác. Seed `sv.gioi` không có phiên chat nên khối «Tiếp tục học» trống ở stack thật; kiểm thứ tự / khử trùng bằng dữ liệu giả |

## Suite có sẵn ở `6a0e8584`
`pnpm exec playwright test` (desktop + mobile): 893 passed, 679 skipped, 16 failed. 6 ca của dev (`attendance`, `a11y`, `dev-ui`, `documents` ×2, `shell`) đỏ lúc chạy chung (QC chạy song song trên cùng VPS) và **chạy lại riêng đều pass**; 6 ca là spec QC (đã sửa giả định, pass ở lần chạy lại). Còn 4 ca `panels.spec.ts` đỏ thật (xem trên). `lint` sạch, `tsc` sạch, `ui-antipatterns.sh` sạch.

## File
Spec: `frontend/e2e/qc-uiux91-w3.spec.ts`, `qc-uiux91-w3-real.spec.ts`. Ảnh: `docs/sprints/9.1/qc/shots/wave-3/`. Chạy lại bản thật: `QC_REAL=1 pnpm exec playwright test -c playwright.qc-real.config.ts qc-uiux91-w3-real --project=desktop`. Log VPS: `/srv/work/qc/out/{w3a,w3b,w3real,full3,rerun-dev}.log`.

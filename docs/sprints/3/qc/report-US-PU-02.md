# Báo cáo QC — US-PU-02 (primitive đủ trạng thái, `/dev/ui`, CitationList, VerificationState) — **CHƯA XONG, DỪNG GIỮA CHỪNG**
Bản chấm `5dfca9b` (chứa `a5e219a`), `build:gate`, `next start -p 3400`. Chủ dự án chuyển chỗ → QC dừng theo lệnh PM. **Không có kết luận story**; dưới đây là cái đã đo.

## Lỗi tìm thấy (có tái hiện)
- **BUG-PU02-1 (thấp, TC-06):** DataTable hàng `tr` focus bằng bàn phím không có `box-shadow = --ep-focus`; chỉ có vạch đỏ 2 px inset ở ô đầu (`td:first-child`) + đổi nền (`DataTable.module.css:33-34`, có `ui-allow`). 15/16 khối còn lại khớp `--ep-focus`. SRS 7.3: focus = `--ep-focus`. Tái hiện: `/dev/ui`, Tab vào hàng bảng, đọc `getComputedStyle(tr/td).boxShadow`. Dev sửa hoặc BA nới SRS 7.3.
- **BUG-PU02-2 (thấp, TC-26):** `Dialog`, `Drawer`, `ConfirmIrreversible` (đều `<dialog>`) **không khoá cuộn nền**: mở hộp thoại rồi `mouse.wheel(800)` → `window.scrollY` 11368 → 12168 (Drawer 11927 → 12727, Confirm 12486 → 13286); `body`/`html` `overflow: visible`, `aria-modal` không có. Bẫy focus (Tab/Shift+Tab ×20 → 0 lần ra ngoài), Esc đóng và trả focus đều đạt.
- **Chờ BA (TC-11, TC-12):** đếm `button, a` trong ô `empty`: ActionList 1, DataTable 1 (đạt); Field 0, CitationList 0 (spec cho CitationList chỉ có câu), Popover 2, Composer 2, CommandPalette 16 — khác "đúng 1" nếu đếm cả khung bao. Ô `error`: Field, Checkbox, StatusText không có `Thử lại` (lỗi nhập, không thử lại được); Field/StatusText không `role=alert`; UndoLine "Không hoàn tác được. Thử lại. Thử lại" thiếu câu "dữ liệu vẫn an toàn". SRS 7.3 không phân biệt; dev's Playwright đạt theo định nghĩa riêng. QC chưa chấm FAIL; cần BA nói rõ.
- `lint-selftest.sh` vẫn in `19 / 19 phép bắt được` (BUG-PU01-1 chưa sửa).

## Kết quả đã đo
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01–03 | PASS | `/dev/ui`: 24 khối, 117 ô áp dụng, 75 ô N/A (192 = 24×8), 0 ô N/A thiếu `data-reason`; tên và thứ tự khớp từng hàng bảng SRS 7.2 (QC tự phân tích bảng từ SRS), tập ô áp dụng của từng khối khớp, 0 ô áp dụng rỗng |
| 04, 09, 13, 17, 23, 30, 42, 45 | PASS | `playwright test` (cấu hình QC cổng 3500, cùng spec của dev): **68/68 pass, 0 skip**, desktop + mobile |
| 05 | PASS | hover thật bằng chuột: 15/16 khối đổi ≥ 1 thuộc tính (Tabs/Segmented chỉ mục chưa chọn; UndoLine đổi `text-decoration-color`) |
| 06 | **FAIL** | BUG-PU02-1 |
| 08 | PASS | ô `disabled`: mọi phần tử khoá có `cursor:not-allowed` (14 khối) |
| 10 | PASS | ô `loading`: 13/13 `aria-busy=true`, 0 spinner |
| 11, 12 | chờ BA | xem trên |
| 14–16 | PASS | `AUDIT_SRC` ở 375/640/900/1280/1440: `{ox:0,cut:[],ell:[]}` mọi bề rộng; `TOUCH_SRC` ở 375 = `[]`; 0 phần tử bị cắt dấu |
| 18 | PASS | bảng 1.000 dòng (48.044 px): tối đa **27** `tr`, tới cuối trong **895 ms**, p95 khung **16,7 ms** (≤ 25) |
| 19, 21 | PASS (một phần) | `th` `position: sticky` còn thấy sau khi cuộn 5.000 px; dòng cao 48 px; mở/đóng Drawer giữ `scrollTop` 12.000. Bàn phím ↑↓/Home/End (TC-20): QC thử chưa ra kết quả (không tìm thấy `tr[tabindex]`) — chỉ có spec dev `datatable` pass |
| 26 | **FAIL** | BUG-PU02-2 |
| 27, 28 | PASS | Esc đóng + trả focus nút mở; Drawer 480 px ở 900/1440, = innerWidth ở 719 và 375 |
| 31 | PASS | Composer 343 px (≤ 820), 1 nút primary, nút phụ `ghost` |
| 35 | PASS | "Nguồn tham khảo (2)", bấm → `aria-expanded` false→true, URL không đổi, 0 tab mới; rỗng đúng câu |
| 36, 37 | PASS (một phần) | bốn lời có mặt; `?as=student` 0 nút duyệt; teacher/ta/admin đủ ba nút. Chưa đo nền / viền trái 1 px |
| 38 | PASS | grep từ cấm = 0; `ui-antipatterns.sh` rc=0 (19 ✓); `lint` rc=0; `ui-allow` = 10 |
| 39 | một phần | `audit.mjs` (đã sửa theo góp ý #13) **SV 165 hàng, FAIL 0**; `proto-curl.sh all` **497 PASS / 0 FAIL** (phép kiểm đã sửa). TA/GV/Admin/spec **chưa chạy xong** (server `next` bị giết giữa chừng — cổng 3400 mất); `sweep`, 18 `data-part` chưa chạy |
| 40 | **chưa chạy** | production build (`/dev/ui` 404, 0 bundle chứa `state-cell`): chỉ có `/dev/data` 404 ở bản gate; chưa chạy `pnpm build` thường |
| 32, 33, 34, 41, 43, 44 | chỉ có spec dev (PASS) | chưa có phép đo độc lập (CLS, offline-input, one-primary sweep) |
| 46, 47 | **chưa làm** | đối chiếu `DESIGN.md` §22 bằng mắt, đo diện tích đỏ, ảnh 1440/375 |

## Việc dở (cho phiên sau)
TC-39 (audit TA/GV/Admin/spec, sweep, 18 `data-part`), TC-40, TC-20, TC-24/25 độc lập, TC-46/47 (bảng §22, ảnh), nốt TC-36 (nền/viền), chốt TC-11/12 với BA. Lưu ý chạy: cổng 3300 là của Playwright dev (giết server của QC) — dùng 3400 và kiểm `lsof -i :3400` trước mỗi lượt; server dựng bằng `python3 subprocess.Popen(start_new_session=True)`.

## Cập nhật khi chạy lại (`a0ecef6`) — DỪNG GIỮA CHỪNG theo lệnh PM
- **TC-39 (một phần → đạt phần đo được):** `audit.mjs` 5 lượt: **683 hàng, FAIL 0** (SV 165, GV 170, TA 106, Admin 54, spec 188); `sweep` SV 42 hàng `FORBIDDEN`=0. `proto-curl.sh` 496/1 do dương tính giả ở `tokenStore.ts` (xem report PU-01); 18 `data-part` chưa đối chiếu.
- Vẫn dở: TC-40 (production build), TC-20 bàn phím DataTable độc lập, TC-46/47, chốt TC-11/12 với BA. BUG-PU02-1, BUG-PU02-2 chưa kiểm lại (dev chưa báo sửa).

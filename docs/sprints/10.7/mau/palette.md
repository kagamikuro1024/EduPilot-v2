# Bộ màu theo nhận diện Đại học Bách khoa Hà Nội (đề xuất, sprint 10.7)

Tech Lead UI · 2026-11-14 · chờ PM / chủ dự án duyệt. Chỉ đổi ở `frontend/src/shared/styles/tokens.css` (và bản sao `docs/design/DESIGN_TOKENS.css`); không màu cứng ở trang.

## 1. Nguồn mã màu chính thức

**«Sổ tay nhận diện thương hiệu ĐHBK Hà Nội – version 2026»**, tệp PDF 81 trang, ngày tạo 23/10/2026, tải từ cổng trường: <https://hust.edu.vn/vi/he-thong-nhan-dien-thuong-hieu/He-thong-NDTH/so-tay-huong-dan-nhan-dien-thuong-hieu-dhbk-ha-noi-pdf.html> (đọc 2026-11-14, trích chữ bằng `pypdf`).

| Trang | Hệ màu | Màu | CMYK | RGB | HEX |
| --- | --- | --- | --- | --- | --- |
| 4 | Quy chuẩn | Đỏ | 0, 100, 90, 15 | 206, 22, 40 | `#CE1628` |
| 4 | Quy chuẩn | Vàng | 0, 20, 100, 5 | 243, 193, 8 | `#F3C108` |
| 4 | Quy chuẩn | Đen | 0, 0, 0, 100 | 35, 31, 32 | `#231F20` |
| 6 | Thứ cấp | Xanh đậm | 100, 67, 0, 38 | 0, 51, 102 | `#003366` |
| 6 | Thứ cấp | Đỏ | 17, 100, 85, 7 | 192, 32, 52 | `#C02034` |
| 6 | Thứ cấp | Cam | 0, 62, 100, 0 | 245, 127, 32 | `#F57F20` |
| 6 | Thứ cấp | Trắng | 0, 0, 0, 0 | 255, 255, 255 | `#FFFFFF` |

Sổ tay chỉ cho CMYK và RGB; cột HEX là đổi thẳng từ RGB. Trang 3: logo có «màu chính: vàng, đỏ và màu phụ: đen». Trang 11: không vẽ lại, không đổi màu logo trường. Ta **không dùng logo trường** (giữ logo EduPilot), nên các quy định về logo không áp vào app.

Hai mã tạm ở bản mẫu đầu tiên (`#B41D24`, `#FFD200`, lấy từ CSS cổng `hust.edu.vn` / `qldt.hust.edu.vn`) **đã bỏ**: CSS của cổng không khớp sổ tay.

## 2. Token đề xuất

| Token | Hiện tại (`tokens.css`) | Đề xuất | Nguồn | Dùng cho |
| --- | --- | --- | --- | --- |
| `--ep-red` | `oklch(54% 0.22 27)` | **`#CE1628`** | Sổ tay tr. 4, đỏ | Nút chính, mục đang chọn, việc cần làm, số việc chờ |
| `--ep-red-hover` | `oklch(49% 0.22 27)` | **`#A81220`** | Suy ra (đỏ tr. 4 hạ 7,5 điểm L) | Nút chính khi rê / nhấn |
| `--ep-red-soft` | `oklch(96% 0.025 25)` | **`#FDEEF0`** | Suy ra | Nền mục đang chọn, nền lỗi |
| `--ep-yellow` | *(chưa có)* | **`#F3C108`** | Sổ tay tr. 4, vàng | **MỚI.** Điểm nhấn hiếm: chấm «cần chú ý», vạch «kế tiếp / đang mở». Chỉ làm nền, chấm, vạch |
| `--ep-yellow-soft` | *(chưa có)* | **`#FDF4CF`** | Suy ra | **MỚI.** Nền nhãn «cần chú ý», chữ `--ep-ink` |
| `--ep-ink` | `oklch(19% 0.02 25)` | **`#231F20`** | Sổ tay tr. 4, đen | Chữ chính |
| `--ep-canvas` | `oklch(95% 0.008 60)` (`#F3EDE9`) | **`#FFFFFF`** | Chủ dự án 14/11: «Nền trắng». Sổ tay tr. 6, trắng | Nền trang. Panel cũng trắng, tách khỏi nền bằng viền 1 px `--ep-panel-border` + bóng tĩnh `--ep-elevation-1` (đã có từ D59, phương án (a)). Trái điều «nền trang tối hơn panel 5 điểm L» của D59 → P-107-10 |
| `--ep-blue` | `oklch(51% 0.12 245)` | **`#003366`** | Sổ tay tr. 6, xanh đậm | Thông tin (6 chỗ dùng) |
| `--ep-surface` | `#FFFFFF` | giữ | Sổ tay tr. 6, trắng | Panel, ô nhập, thanh bên, thanh trên |
| `--ep-surface-subtle`, `--ep-surface-strong` | `#F9F3F3`, `#FAEFEE` | giữ | – | Rê hàng, đầu bảng, ô nhấn |
| `--ep-ink-2`, `--ep-ink-3` | `#65595B`, `#746968` | giữ | – | Chữ phụ, chữ meta |
| `--ep-rule`, `--ep-panel-border` | `#E8DDDD` | giữ | – | Đường kẻ, viền panel |
| `--ep-green` | `#287A4A` | giữ | Ngoài sổ tay | Thành công / đã xác nhận |
| `--ep-amber` | `#A86600` | giữ | Ngoài sổ tay | Chữ cảnh báo; viền 1 px quanh chấm vàng |

Ghi mã dạng HEX (không đổi sang OKLCH) cho bốn màu lấy từ sổ tay, để giá trị trên màn khớp đúng RGB chính thức.

**Ba trạng thái:** lỗi = `--ep-red` trên `--ep-red-soft`; cảnh báo = chữ `--ep-amber`, chấm `--ep-yellow`; thành công = `--ep-green`.

### Thay đổi nhìn thấy được, nói thật

- **Đỏ gần như không đổi.** Đỏ hiện tại của app (`oklch(54% 0.22 27)`) và đỏ Bách khoa (`#CE1628` = `oklch(54,3% 0,211 25)`) lệch nhau dưới 1 điểm L. Người dùng sẽ không nhận ra khác biệt ở nút và thanh điều hướng.
- Cái người dùng thấy khác là: **nền trang trắng** (trước đây be hồng `#F3EDE9`), **vàng xuất hiện** (trước đây không có), mục điều hướng đang chọn có nền đỏ nhạt.
- Logo EduPilot đang dùng `#C81D32` và `#251D1E` (`frontend/public/brand/*.svg`, 3 tệp). Đề xuất đổi sang `#CE1628` và `#231F20` cho khớp token. Chủ dự án quyết (P-107-8).

## 3. Tương phản WCAG 2.1 (đã đo, công thức độ chói tương đối)

Ngưỡng: chữ thường ≥ 4,5 : 1; chữ lớn (≥ 24 px, hoặc ≥ 18,66 px đậm) và thành phần không phải chữ ≥ 3 : 1.

| Chữ / nền | Tỉ lệ | Kết quả |
| --- | --- | --- |
| Trắng trên `--ep-red` (nút chính) | 5,57 | Đạt AA |
| Trắng trên `--ep-red-hover` | 7,57 | Đạt AA |
| `--ep-red` trên trắng (chữ đỏ, số việc chờ) | 5,57 | Đạt AA |
| `--ep-red` trên `--ep-red-soft` | 4,95 | Đạt AA |
| `--ep-red` trên `--ep-surface-subtle` / `-strong` | 5,08 / 4,95 | Đạt AA |
| `--ep-ink` trên trắng | 16,30 | Đạt AAA |
| `--ep-ink` trên `--ep-yellow` | 9,66 | Đạt AAA |
| `--ep-ink` trên `--ep-yellow-soft` | 14,76 | Đạt AAA |
| `--ep-ink-2` trên trắng / subtle / red-soft | 6,70 / 6,11 / 5,96 | Đạt AA |
| `--ep-ink-3` trên trắng / subtle / red-soft | 5,30 / 4,83 / 4,71 | Đạt AA |
| `--ep-green` trên trắng | 5,29 | Đạt AA |
| `--ep-blue` (`#003366`) trên trắng | 12,61 | Đạt AAA |
| `--ep-amber` trên trắng | 4,60 | Đạt AA |
| `--ep-amber` trên `--ep-yellow-soft` | 4,17 | **Không đạt** cho chữ thường. Chữ cảnh báo chỉ đặt trên trắng; trên nhãn vàng nhạt dùng chữ `--ep-ink` |
| `--ep-yellow` trên trắng | 1,69 | **Không đạt.** Cấm chữ vàng, cấm viền / icon vàng đứng một mình |
| Chấm `--ep-yellow` có viền 1 px `--ep-amber` trên trắng | 4,60 (viền) | Đạt 3 : 1 cho thành phần không phải chữ |
| Viền panel `--ep-panel-border` (`#E8DDDD`) trên trắng | 1,33 | Không thuộc phạm vi 1.4.11 (khung nhóm nội dung, không phải điều khiển). Vì nền và panel cùng trắng, panel **phải** giữ cả viền lẫn bóng; bỏ một trong hai là mất ranh giới |

Nền trang trắng làm mọi cặp «chữ trên nền trang» bằng đúng cặp «chữ trên trắng» ở bảng trên; không còn cặp nào sát ngưỡng vì nền xám.

Tự đo lại: mở `mau-1.html`, DevTools › Accessibility › Contrast; hoặc chạy `frontend/e2e/contrast.spec.ts` sau khi dev đổi token (spec đã có từ sprint 5.5, cần thêm hai cặp vàng).

## 4. Luật dùng màu (thêm vào `DESIGN.md` §4 «Color rules»)

1. Đỏ vẫn là tín hiệu, không phải nền: đang ở đâu, cần hành động, đã sửa / xác nhận. Diện tích đỏ < 8 % mỗi màn ứng dụng. **Không** tô đỏ thanh trên hay thanh bên (khác cổng `qldt.hust.edu.vn`).
2. Vàng là điểm nhấn **hiếm**: tối đa **3 phần tử vàng trong một khung nhìn**, mỗi phần tử ≤ 24 px một chiều (chấm 8 px, vạch 3 px, nhãn `--ep-yellow-soft`). Nghĩa duy nhất: «cần chú ý, chưa gấp» hoặc «kế tiếp / đang mở».
3. Vàng không bao giờ là màu chữ, màu icon đứng một mình, hay nền của nút.
4. Vàng và đỏ không chồng lên nhau (đỏ trên vàng 3,30 : 1, không đạt).
5. Trạng thái luôn có chữ hoặc biểu tượng đi kèm màu (đã có ở §10.6).

Kiểm bằng máy: xem luật 31 ở `techlead.md` TL-1 mục 4 (cấm `color: var(--ep-yellow)`) và phép đo `C3` ở mục 1.

## 5. Màn mẫu dùng bộ màu này

| Tệp | Màn | Ảnh |
| --- | --- | --- |
| `mau-1.html` | «Hôm nay» giảng viên, 1440 × 900 | `sau-web.png` |
| `mau-app.html` | App điện thoại sinh viên, điểm danh bằng mã QR (sprint 10.8), 390 × 844 | `sau-app.png` |
| `khung-trang.html` | Ba màn khung × bốn trạng thái (`?m=work\|read\|split&s=data\|empty\|filtered\|loading\|error`) | `khung/*.png` |

Cả ba là HTML tĩnh tự chứa (phông tải từ Google Fonts), không phải mã sản phẩm.

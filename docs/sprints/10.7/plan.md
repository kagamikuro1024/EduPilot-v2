# Sprint 10.7 (UI-R): rà và làm lại khung giao diện toàn bộ (chủ dự án giao PM quyết phạm vi, 2026-11-14)

## Vì sao
Chủ dự án chụp 3 màn cùng một vai, cùng một lớp: mỗi màn một kiểu panel và đầu trang.

| Màn | Panel | Đầu trang | Trạng thái rỗng |
| --- | --- | --- | --- |
| `/insights` Lỗ hổng kiến thức | rộng ≈ 840 px, lệch trái | 2 dòng meta + nút chính dưới meta + tiêu đề mục có dòng phụ | – |
| `/question-bank` | rộng tới mép phải − 115 px | nút chính góc phải | icon + tiêu đề + câu lặp tiêu đề + **nút chính thứ hai** trùng nút đầu trang |
| `/inbox` | rộng hết màn, cao hết màn | dòng meta lặp mã lớp | tiêu đề + câu giải thích + nút ghost thụt lề |

Thêm một **lỗi dữ liệu**: `/insights` khi đang chọn lớp KC106 lại hiện báo cáo của lớp 761987.

**Nguyên nhân gốc (PM nhận):** sprint 5.5 (panel nổi) và 9.1 (khảo sát UI/UX) duyệt **từng màn theo AC của màn đó**, với dữ liệu seed. Không có **hợp đồng khung trang** chung bắt buộc bằng mã, không có lượt so **chéo giữa các màn**, không quét trạng thái rỗng. Primitive `Page` / `Panel` / `PageHeader` / `EmptyState` cho phép mỗi route tự chọn độ rộng, padding, vị trí nút, thân chữ — nên mỗi route chọn một kiểu.

## Phạm vi (PM quyết)
Không vẽ lại thương hiệu (giữ Red Thread, token `--ep-*`, Be Vietnam Pro). Làm lại **khung**: mọi route theo một hợp đồng, ép bằng primitive + script, kiểm bằng ảnh chụp toàn bộ.

1. **Hợp đồng khung trang** (`docs/design/DESIGN.md` §23, Tech Lead soạn, PM duyệt): 
   - Đúng **2 kiểu độ rộng**: `work` (danh sách, bảng, chia đôi: hết vùng nội dung) và `read` (báo cáo, văn bản dài: cột đọc cố định, căn giữa). Không route nào tự đặt width.
   - **Một mẫu đầu trang**: tiêu đề · tối đa một dòng meta (không lặp lớp đang chọn) · hành động chính bên phải cùng hàng tiêu đề · ≤ 2 hành động phụ.
   - **Panel**: một panel cho một vùng làm việc; padding, bo góc, bóng do primitive quyết; panel cao theo nội dung (không kéo hết màn khi rỗng), trừ màn chia đôi.
   - **Tiêu đề mục**: không dòng phụ.
   - **Trạng thái rỗng**: một mẫu duy nhất (tiêu đề ngắn + tối đa một nút, căn giữa vùng); không thân chữ; không nút trùng hành động đầu trang (gộp `ui-empty`).
2. **Ép bằng mã**: thu hẹp API primitive (bỏ prop độ rộng tuỳ ý, bỏ `children` chữ của `EmptyState`, `PageHeader.actions` một chỗ); `scripts/ui-antipatterns.sh` thêm luật: cấm `maxWidth`/`width` trong CSS module của trang cho khung, cấm `EmptyState` có chữ, cấm meta lặp mã lớp, cấm dòng phụ dưới tiêu đề mục.
3. **Sửa mọi route** cho đúng hợp đồng (4 vai, 1440 + 375).
4. **Kiểm bằng ảnh toàn bộ**: script Playwright chụp mọi route × vai × 1440/375 × (có dữ liệu, rỗng) → một trang HTML ảnh trước/sau; QC so chéo theo bảng kiểm hợp đồng, không theo AC từng màn.
5. **Persona-eval** (P-11-12) chạy trên bản sau UI-R ở sprint 11.

## Thứ tự và vai
| # | Ai | Việc | Ra |
| --- | --- | --- | --- |
| 0 | dev | Điều tra (chỉ đọc) lỗi `/insights` hiện báo cáo lớp khác | báo PM |
| 1 | QC | Script chụp toàn bộ + bảng kiểm chéo hiện trạng (dựa ảnh, không cảm tính) | `docs/sprints/10.7/qc/audit.md`, `audit.html` |
| 1 | Tech Lead UI (Fable 5.2, `techlead-ui`) | Kế hoạch đánh giá + refine; hợp đồng §23 + danh sách thay đổi primitive + luật script | `docs/sprints/10.7/techlead.md`, nháp §23 |
| 2 | dev | (sau hàng sửa 10.6) primitive trước → route sau; nhánh `fix/ui-empty` đổi thành nền của UI-R | commit nhỏ theo nhóm route |
| 3 | QC | Chụp lại, kiểm chéo, 0 mục còn lại; Playwright + `ui-antipatterns` sạch | `docs/sprints/10.7/qc/summary.md` |
| 4 | PM | Báo cáo trước/sau cho chủ dự án | `~/Documents/EduPilot-bao-cao/ui-r.html` |

Thứ tự lộ trình: 10.6 → **10.7 (UI-R)** → lượt kiểm + bench chung → 10.8 điểm danh → 11. UI-R trước lượt kiểm chung để smoke đợt R chạy trên giao diện cuối.

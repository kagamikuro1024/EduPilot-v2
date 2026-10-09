# Sprint 5.5 — Panel nổi: tách lớp giao diện (UI)

Trạng thái: **KẾ HOẠCH, chưa thi công.** Chủ dự án góp ý 2026-10-08. Chạy **sau cổng PE (sprint 5)**, trước sprint 6. Nhánh dự kiến `sprint/5.5-ui-panels`, xếp chồng trên `sprint/5-pe` nếu sprint 5 chưa vào `main`.

## Góp ý của chủ dự án
Các khối phải **nổi bật hơn**, kiểu panel nổi, để nhìn rõ từng vùng; không phải một nền cùng màu từ đầu đến cuối. Tham chiếu: trang Start Page của Safari. Ở đó nền trang tối hơn; mỗi nhóm nội dung nằm trên một panel sáng hơn, bo góc rộng, tiêu đề nằm ngoài panel; bên trong có ô nhấn tương phản mạnh cho số liệu.

## Vướng `DESIGN.md` hiện hành → cần quyết định mới (D59)
`DESIGN.md` đang cấm đúng điều chủ dự án muốn:
- "No floating-card aesthetic" (§ Surface character).
- "Use whitespace before containers".
- "Panels: 8–12px only where a true contained surface is necessary".
- "No decorative shadows on ordinary surfaces".

`AGENTS.md` (luật giao diện) cũng ghi "không bọc mọi mục trong khung bo góc". Muốn làm thì phải sửa ba chỗ:
- `DESIGN.md`;
- `AGENTS.md`;
- `scripts/ui-antipatterns.sh`.

Việc này ghi thành **D59** khi chủ dự án duyệt hướng ở bước 1. Đây là đổi hướng thị giác có chủ đích, không phải sửa lặt vặt.

**Đề xuất D59 — "panel có kỷ luật".** Lấy cái chủ dự án muốn, giữ lại những luật vẫn đúng.

| Đổi | Giữ nguyên |
| --- | --- |
| Nền trang (`--ep-canvas`) **đậm hơn panel một bậc**; mỗi vùng làm việc nằm trên **một panel** `--ep-surface` | Không card lồng card: bên trong panel tách nhóm bằng khoảng trắng và đường kẻ 1 px |
| Panel bo **12–16 px**, viền 1 px rất nhạt + bóng mềm một bậc (`--ep-elevation-1`) | Không tường thẻ KPI: số liệu là một dải gọn trong panel, không 6 thẻ to |
| Tiêu đề vùng nằm **ngoài** panel, như Safari ("Favorites", "Privacy Report") | Đỏ chỉ là tín hiệu: đang ở đâu / cần làm / đã xác nhận; không tô nền panel |
| Ô nhấn bên trong panel (`--ep-surface-strong`), chỉ cho 1–3 con số hoặc trạng thái quan trọng | Không kính mờ, không gradient, không neon; một họ chữ |
| Sidebar và thanh trên tách khỏi nền bằng tông màu | Một hành động chính mỗi vùng; màn sinh viên dùng tốt ở 375 px |

## Câu hỏi cho chủ dự án (hỏi ở bước 1, có mặc định)
1. **Chế độ tối.**
   - Mặc định: chỉ làm chế độ sáng. Bộ token sẵn sàng cho chế độ tối, nhưng chế độ tối để sau.
   - Ảnh tham chiếu là Safari chế độ tối. Nếu muốn có cả chế độ tối thì sprint thêm khoảng 1 story và phải chụp ảnh mốc gấp đôi.
2. **Độ nổi.** Chủ dự án chọn trên `/dev/ui` giữa 3 phương án dựng sẵn:
   - (a) nền xám ấm + panel trắng + bóng mềm (mặc định);
   - (b) chỉ dùng chênh tông nền, không bóng;
   - (c) panel có viền rõ, không bóng.

## Story

| # | Story | Ước lượng | Ghi chú |
| --- | --- | --- | --- |
| UI-01 | Dựng 3 phương án độ nổi trên `/dev/ui` (trang mẫu "Hôm nay" GV + SV) → **chủ dự án chọn** → ghi D59, sửa `DESIGN.md` (Surface character, Radius, phản mẫu §21), `AGENTS.md`, `ui-antipatterns.sh` | M | Dev làm theo skill `impeccable` (shape / critique). Chủ dự án duyệt ảnh chụp trước khi làm tiếp |
| UI-02 | Token mới (`--ep-canvas`, `--ep-surface-strong`, `--ep-elevation-1`, `--ep-radius-panel`) + primitive `Panel` / `PanelSection` trong `frontend/src/shared/` + ma trận `/dev/ui` + luật chặn `Panel` lồng `Panel` | M | Không màu / bóng / bo góc viết cứng trong trang |
| UI-03 | Khung ứng dụng: sidebar, thanh trên, vùng cuộn trên nền canvas; bảng "Thêm" trên mobile | S | |
| UI-04 | Màn sinh viên: Hôm nay, lớp của tôi, bài thi (danh sách + làm bài + kết quả), cài đặt; kiểm 375 px | M | Màn mock của các phase sau dùng chung `Panel` nên đổi theo |
| UI-05 | Màn giảng viên / TA: Hôm nay, thành viên, nạp danh sách, bài thi, ngân hàng câu hỏi, kết quả thi | M | |
| UI-06 | Màn Admin: `/admin/users`, `/settings/llm`, lớp học | S | |
| UI-07 | Sinh lại ảnh mốc (trong image Playwright, có ảnh trước / sau trong handoff), axe 0 serious (độ tương phản chữ trên nền mới), Lighthouse không hồi quy LCP / TBT, `ui-antipatterns.sh` sạch | S | Cổng UI |

## Quy trình
1. BA viết `docs/specs/FEAT-ui-panels/` (ngắn: AC theo từng story).
2. Tech Lead thẩm định spec (`TL-REVIEW.md`); PM quyết.
3. Dev làm UI-01 và **dừng chờ chủ dự án chọn phương án**. Làm UI-02…07 theo phương án đã chọn.
4. QC chạy TC song song từng story. Dùng `playwright-cli` chụp mọi màn ở 1440 / 1024 / 375 px để so trước và sau.

## Cổng nghiệm thu sprint 5.5
- Chủ dự án xem ảnh chụp trước / sau các màn chính và gật.
- Không còn trang nào có "nền một màu từ đầu đến cuối": mọi vùng làm việc nằm trên panel.
- Không panel lồng panel (`ui-antipatterns.sh`).
- Không tường thẻ KPI.
- axe 0 critical / serious; độ tương phản chữ phụ trên panel và trên canvas ≥ 4,5 : 1.
- LCP / TBT không kém hơn cuối sprint 5. CI xanh.

## Rủi ro
- **Đổi giao diện toàn bộ:** gần như mọi ảnh mốc đổi. Ảnh mốc chỉ được sinh lại khi handoff liệt kê từng ảnh, lý do, ảnh trước / sau (quy tắc Q-QC-PU06-4).
- **Bóng làm nặng trang:** chỉ dùng `box-shadow` tĩnh, không `filter` / `backdrop-filter`.

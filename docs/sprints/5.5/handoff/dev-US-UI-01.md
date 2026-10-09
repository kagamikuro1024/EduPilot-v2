# DEV handoff — US-UI-01 (ba phương án độ nổi của panel) — **DỪNG, chờ PM / chủ dự án chọn**
Nhánh `sprint/5.5-ui-panels` (đã merge `sprint/5-pe` @ `bc09dd2`). Spec `FEAT-ui-panels` v1.1. Làm AC1–AC4; **AC5: dừng**. Chưa làm AC6–AC8 và US-UI-02…07. Không đụng `DESIGN.md`, `AGENTS.md`, `ui-antipatterns.sh`, `Panel.tsx`, `tokens.css` (AC5: xem cuối).

## Đã dựng
- `/dev/panels?surface=<a|b|c>&role=<teacher|student>` — `frontend/src/app/dev/panels/` (`layout.tsx` = `AuthGate` + `AppShell` thật, `page.dev.tsx`, `PanelSamples.tsx`, `fixtures.ts` dữ liệu giả cố định, `SurfaceMarker.tsx` đặt `data-surface` trên `<html>`). Thiếu / sai tham số → `a` / `teacher`. **Khung theo vai của phiên đăng nhập**, nội dung mẫu theo `role`: xem trên trình duyệt thì đăng nhập đúng vai (`teacher@…` / `sv.gioi@…`) để thanh nav khớp mẫu; e2e và ảnh dùng `asDemo`.
- `frontend/src/shared/styles/panel-variants.css` (tạm): `html[data-surface="a|b|c"]` đặt lại token của `SRS 5.2` (`--ep-canvas`, `--ep-panel-border`, `--ep-elevation-1`, `--ep-ink-3` ở (b)); phần chung `--ep-surface-strong`, `--ep-radius-panel: 14px`, và — chỉ dưới `data-surface` — `--ep-paper: var(--ep-canvas)` + `[data-part=sidebar|topbar] { background: var(--ep-surface) }` để khung ứng dụng thật trông đúng như UI-03 mà **không sửa `AppShell.module.css`**.
- `frontend/e2e/panels-variants.spec.ts` (tạm, xoá cùng trang ở AC8): `tokens` ×3, `sample rules` ×6, `shots` (chỉ khi `PANEL_SHOTS=1`).

## Số đo AC2 (đo trong trình duyệt, màu đã giải → sRGB 8 bit → WCAG / OKLab; `contrast-*.json` cạnh ảnh)
ΔL đo được: **(a) 5.03** (≥ 5) · **(b) 7.1** (≥ 7) · **(c) 3.06** (≥ 3).

| Chữ | Nền | (a) | (b) | (c) |
| --- | --- | --- | --- | --- |
| ink | canvas | 16.01 | 15.05 | 16.99 |
| ink-2 | canvas | 8.02 | 7.53 | 8.50 |
| ink-3 | canvas | 4.57 | 4.69 | 4.85 |
| red | canvas | 4.90 | 4.60 | 5.20 |
| green | canvas | 5.08 | 4.77 | 5.39 |
| blue | canvas | 4.91 | 4.61 | 5.21 |
| ink | panel `--ep-surface` | 18.58 | 18.58 | 18.58 |
| ink-2 | panel `--ep-surface` | 9.30 | 9.30 | 9.30 |
| ink-3 | panel `--ep-surface` | 5.30 | 5.79 | 5.30 |
| red | panel `--ep-surface` | 5.69 | 5.69 | 5.69 |
| green | panel `--ep-surface` | 5.89 | 5.89 | 5.89 |
| blue | panel `--ep-surface` | 5.70 | 5.70 | 5.70 |
| ink | ô nhấn `--ep-surface-strong` | 16.50 | 16.50 | 16.50 |
| ink-2 | ô nhấn `--ep-surface-strong` | 8.26 | 8.26 | 8.26 |
| ink-3 | ô nhấn `--ep-surface-strong` | 4.70 | 5.14 | 4.70 |
| red | ô nhấn `--ep-surface-strong` | 5.05 | 5.05 | 5.05 |
| green | ô nhấn `--ep-surface-strong` | 5.23 | 5.23 | 5.23 |
| blue | ô nhấn `--ep-surface-strong` | 5.06 | 5.06 | 5.06 |

Thấp nhất: **4,57** — `ink-3` trên canvas (a) (BA tính 4,59; chênh do làm tròn 8 bit). Mọi ô ≥ 4,5. Không phải chỉnh giá trị khởi điểm của bảng 5.2. (b) dùng `--ep-ink-3: oklch(51% …)`; `red` / `blue` trên canvas (b) = 4,60 / 4,61 — sát ngưỡng. Bóng: `--ep-elevation-1 ≠ none` chỉ ở (a); viền: (a) = `--ep-rule`, (b) trong suốt, (c) `oklch(80% …)`; bán kính panel 14 px ở cả ba.

## 18 ảnh (AC4) — 1440 / 1024 / 375 px
| Ảnh | Phương án | Vai | Bề rộng |
| --- | --- | --- | --- |
| `ui-01/a-teacher-1440.png` | (a) | Giảng viên | 1440 px |
| `ui-01/a-teacher-1024.png` | (a) | Giảng viên | 1024 px |
| `ui-01/a-teacher-375.png` | (a) | Giảng viên | 375 px |
| `ui-01/a-student-1440.png` | (a) | Sinh viên | 1440 px |
| `ui-01/a-student-1024.png` | (a) | Sinh viên | 1024 px |
| `ui-01/a-student-375.png` | (a) | Sinh viên | 375 px |
| `ui-01/b-teacher-1440.png` | (b) | Giảng viên | 1440 px |
| `ui-01/b-teacher-1024.png` | (b) | Giảng viên | 1024 px |
| `ui-01/b-teacher-375.png` | (b) | Giảng viên | 375 px |
| `ui-01/b-student-1440.png` | (b) | Sinh viên | 1440 px |
| `ui-01/b-student-1024.png` | (b) | Sinh viên | 1024 px |
| `ui-01/b-student-375.png` | (b) | Sinh viên | 375 px |
| `ui-01/c-teacher-1440.png` | (c) | Giảng viên | 1440 px |
| `ui-01/c-teacher-1024.png` | (c) | Giảng viên | 1024 px |
| `ui-01/c-teacher-375.png` | (c) | Giảng viên | 375 px |
| `ui-01/c-student-1440.png` | (c) | Sinh viên | 1440 px |
| `ui-01/c-student-1024.png` | (c) | Sinh viên | 1024 px |
| `ui-01/c-student-375.png` | (c) | Sinh viên | 375 px |

## Nhận xét thị giác (một dòng mỗi phương án)
- **(a) xám ấm + bóng mềm** — panel trắng viền mảnh nổi nhẹ trên canvas 95 %; phân cấp (canvas → panel → ô nhấn) dễ đọc nhất khi các vùng xếp dọc; rủi ro: cần ngoại lệ bóng ở máy kiểm (TLR-1) và `ink-3` trên canvas chỉ 4,57.
- **(b) chênh tông, không bóng, không viền** — ranh giới chỉ nhờ canvas 93 %; panel "nổi" rõ nhất ở 375 px; rủi ro lớn nhất: tương phản mỏng (`red` 4,60, `blue` 4,61, bắt buộc hạ `ink-3` xuống 51 %), cả trang ngả "be" đậm, amber không được đặt trên canvas (2,91 < 3).
- **(c) viền rõ, không bóng** — canvas 97 % gần như trắng, viền 1 px đậm hơn đường kẻ; gần luật hiện hành nhất (không bóng, tương phản dư nhất: `ink-3` trên canvas 4,85); rủi ro: panel "nằm" chứ không "nổi", và viền đậm quanh mọi vùng gần phản mẫu §21 "bọc mọi mục trong khung" khi trang có nhiều panel.

## Khuyến nghị của dev (không ràng buộc)
**(a).** Nó là phương án duy nhất cho cảm giác "panel nổi" mà chủ dự án yêu cầu mà không đổi cả bảng màu chữ (như (b)) và không làm phản mẫu §21 (như (c)). Cái giá là một ngoại lệ máy kiểm đã có sẵn trong spec (TLR-1) và biên tương phản 4,57 của `ink-3` trên canvas — không được làm canvas (a) sáng hơn 1 điểm L nữa mà không đo lại. Nếu muốn giữ nguyên "không bóng trên bề mặt thường" của `DESIGN.md` §0 thì chọn **(c)** (rủi ro thấp nhất); (b) không khuyên.

## Phát hiện cho các story sau (chưa sửa — ngoài phạm vi UI-01)
1. `ActionList` luôn vẽ `border-top` ở gốc: đặt ở đầu panel thì đường kẻ trùng viền panel. Trang mẫu dùng `.panel > ul:first-child {{ border-top: 0 }}`; `Panel` / `ActionList` thật (UI-02) phải xử lý một chỗ.
2. `Layout.module.css` ép `.page` ở < 720 px thành 16 px (`padding: … var(--ep-space-4) …`, dòng 53) và thắng class của trang tôi cùng độ đặc hiệu → trang mẫu dùng khung riêng `.sample` (12 px đúng SRS 4.2 điều 8). UI-04 AC7 sẽ đổi `.page`.
3. `AppShell.module.css`: sidebar `--ep-surface-subtle`, thanh trên `--ep-paper`; UI-03 đổi sang `--ep-surface` thật. Ở trang mẫu đã mô phỏng bằng `panel-variants.css`.
4. Dải "Bản mô phỏng · dữ liệu giả" của `AppShell` ở di động là một dải hồng (`--ep-red-soft`) rộng đủ chiều ngang, nằm trên canvas — cần xem lại ở UI-03 (đỏ chỉ là tín hiệu).

## AC (kết quả thật)
| AC | Kết quả |
| --- | --- |
| 1 | `gbuild` + `next start`: `200 200 200 200 200 200` cho 6 URL, tham số sai → 200 (mặc định `a` / `teacher`); `pnpm build` (production) + `next start`: `404 404 404 404 404 404`. `grep -rn 'data-surface' frontend/src \| grep -v 'shared/styles/panel-variants.css\|app/dev/' \| wc -l` = **0** |
| 2 | `panels-variants.spec.ts -g tokens` → 3 pass (cả `desktop` và `mobile`): ΔL, 18 cặp × 3 nền ≥ 4,5, bóng / viền / bán kính đúng bảng 5.2 |
| 3 | `-g 'sample rules'` → 6 URL × (desktop + mobile) = 12 pass: 3 / 2 vùng đúng tên, tiêu đề `h2` ngoài panel, NEST = 0, STRONG ≤ 3 (≥ 1), WALL < 3, một nút `primary` (trong panel đầu), nền canvas / panel / ô nhấn không đỏ, diện tích đỏ < 8 %, `AUDIT_SRC` `{{ ox: 0, cut: [] }}` ở 1440 / 1024 (desktop) và 375 / 390 (mobile), 375 px: một cột, mép trái / phải đúng 12 px, không tràn ngang, `TOUCH_SRC` rỗng ở hai URL Sinh viên |
| 4 | 18 ảnh trong `ui-01/` (`ls ui-01/*.png \| wc -l` = 18); bảng ảnh + số đo + nhận xét + khuyến nghị ở trên |
| 5 | Dừng. Chưa có commit nào chạm `DESIGN.md`, `AGENTS.md`, `ui-antipatterns.sh`, `Panel.tsx`, `tokens.css` hay CSS màn nào ngoài `/dev/panels` |

Gates: `pnpm lint` sạch, `tsc --noEmit` sạch, `ui-antipatterns.sh` 0 ✗ (19 ✓; dòng `box-shadow: var(--ep-elevation-1)` của trang mẫu có `ui-allow:` tạm), `lint-selftest` 7/7 + 19/19. **Chưa chạy**: toàn bộ Playwright không `@real` (chỉ `panels-variants`), Lighthouse, axe riêng cho `/dev/panels`; CI GitHub của nhánh 5.5 chưa xem sau commit này.

## Cần PM
Chủ dự án chọn (a) / (b) / (c) và trả lời `QUESTIONS.md` Q1 (chế độ tối). Sau đó PM ghi D59 (AC6); dev làm AC7–AC8 rồi US-UI-02…07.

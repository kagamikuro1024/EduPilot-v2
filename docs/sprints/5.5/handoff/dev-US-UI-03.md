# DEV handoff — US-UI-03 (khung ứng dụng trên canvas)
Nhánh `sprint/5.5-ui-panels`; D59 (a). Chưa đổi nội dung màn nào (UI-04…06) ngoài `AuthShell` (AC5 yêu cầu).

## Làm gì
- `AppShell.module.css`: sidebar `--ep-surface` + `border-right 1px --ep-rule`; thanh trên `--ep-surface` (đã ở UI-02); mục nav `hover` / đang chọn `--ep-surface-subtle`, vạch đỏ 2 px giữ; thanh dưới mobile và bảng "Thêm" giữ nguyên (đã đúng). `tokens.css`: `body` nền `--ep-canvas` (`html` đã có); `main` không tự đặt nền.
- `AuthShell.tsx` (+ `.module.css`): nền canvas, cột ≤ **440 px**, thêm `AuthPanel({ title, children })` = `h1` NGOÀI + **một** `Panel` ngay sau; sáu màn (`login`, `register`, `forgot-password`, `reset-password`, `verify-email`, `invite/[token]`) chuyển mọi biến thể (đang kiểm tra / thành công / lỗi / biểu mẫu) sang `AuthPanel`; chữ và hành vi không đổi. ≤ 719 px: lề 12 px (`--ep-space-3`).
- `PreShell` giữ nguyên: nó dùng đúng các lớp CSS của `AppShell` nên nền ba vùng khớp tự động; không import `Panel`.
- Test: `shell.spec.ts` + `frame layers`, `nav states`, `preshell parity`, `bottom nav + more sheet`, `route access`; `panels.spec.ts` + `auth shell` (6 route × 1440 / 1024 / 375); `contrast.spec.ts` + `sidebar`; `shots.spec.ts` (chỉ khi đặt `SHOTS=ui03`, chụp 12 ảnh).
- Sửa kèm (không liên quan khung): ca `bell real` ở `class-join.spec.ts` dùng đồng hồ thật nên đỏ khi chạy trong 5 phút đầu sau nửa đêm giờ VN ("5 phút trước" thành "hôm qua 23:55"; CI từng đỏ lúc 00:02) → cố định đồng hồ trang và `now` ở 10:00 ngày 29/10.

## AC
| AC | Kết quả |
| --- | --- |
| 1 | `-g 'frame layers'` pass (body = canvas; sidebar, thanh trên = surface, đặc, `border-right` / `border-bottom` 1 px `--ep-rule`; `main` trong suốt; không `backdrop-filter`); `ui-antipatterns` "Khung vỏ dùng nền đặc" ✓ |
| 2 | `-g 'nav states'` pass (hover và đang chọn = `--ep-surface-subtle`; `::before` rộng 2 px màu `--ep-red`; focus bàn phím có vòng `--ep-focus`; vòng đỏ 5,69 : 1 trên surface và 4,90 : 1 trên canvas ≥ 3); `contrast.spec.ts -g sidebar` pass (nhãn nhóm, mục nav, mục đang chọn ≥ 4,5 trên nền thật của chúng) |
| 3 | Padding `.page` không đổi (không sửa `Layout.module.css`); `AUDIT` `ox = 0` ở 1440 / 1024 qua `auth shell` + các ca hiện có (`regression-1.5: LEFT page-title 240 / 16`, `touch`, `a11y`) vẫn pass trong lần chạy toàn bộ |
| 4 | `-g 'bottom nav|more sheet'` pass (thanh dưới surface + kẻ trên 1 px; bảng Thêm nền surface, có bóng, không chứa `Panel`; mục ≥ 48 px); ca `mobile:` / `touch:` hiện có pass |
| 5 | `-g 'auth shell'` pass: **một** `[data-ep-panel]`, h1 liền trước panel, NEST = TITLE = 0, panel ≤ 440 px, canvas, ≤ 1 nút primary, 375 px lề 12 px, AUDIT sạch ở 1440 / 1024 / 375; `account.spec.ts`, `class-join.spec.ts` (không `@real`) pass |
| 6 | `-g 'preshell parity'` pass: nền `body` / sidebar / thanh trên của `PreShell` = của `AppShell`, `h1` không dịch quá 2 px khi phiên về. `lcp.spec.ts` ('lcp before refresh') pass trong lần chạy toàn bộ. **Chưa đo** CLS bằng `lhci` (Lighthouse không chạy ở worktree này) |
| 7 | `-g 'nav per role|route access'` pass (8 / 13 / 16 / 6); `git diff HEAD -- nav.ts` = **0** dòng; không `backdrop-filter` (AC1) |
| 8 | 12 ảnh `ui-03/` (bảng dưới); 6 / 14 ảnh mốc đổi (bảng dưới; 8 ảnh còn lại — 7 ảnh 390 px và `dev-ui-1440` — giống từng byte vì không có sidebar / thanh trên đổi nền); `visual.spec.ts` **14 passed ba lần liên tiếp** trong `mcr.microsoft.com/playwright:v1.63.0-noble` (`--update-snapshots=all` rồi hai lần kiểm). CI: xem cuối |

### 12 ảnh khung (`/`)
| Ảnh | Vai | Bề rộng |
| --- | --- | --- |
| `ui-03/student-1440.png` | Sinh viên | 1440 px |
| `ui-03/student-1024.png` | Sinh viên | 1024 px |
| `ui-03/student-375.png` | Sinh viên | 375 px |
| `ui-03/ta-1440.png` | TA | 1440 px |
| `ui-03/ta-1024.png` | TA | 1024 px |
| `ui-03/ta-375.png` | TA | 375 px |
| `ui-03/teacher-1440.png` | Giảng viên | 1440 px |
| `ui-03/teacher-1024.png` | Giảng viên | 1024 px |
| `ui-03/teacher-375.png` | Giảng viên | 375 px |
| `ui-03/admin-1440.png` | Admin | 1440 px |
| `ui-03/admin-1024.png` | Admin | 1024 px |
| `ui-03/admin-375.png` | Admin | 375 px |

### Ảnh mốc đổi (6)
| Ảnh | Lý do | Trước / sau |
| --- | --- | --- |
| `frontend/e2e/visual.spec.ts-snapshots/home-1440.png` | / Sinh viên: sidebar nền `--ep-surface` (trắng, trước `--ep-surface-subtle`), mục đang chọn nền `--ep-surface-subtle` | trước: ảnh ở commit `3f83d84` (git); sau: ảnh trong commit này |
| `frontend/e2e/visual.spec.ts-snapshots/chat-1440.png` | /chat Sinh viên: như `home` | trước: ảnh ở commit `3f83d84` (git); sau: ảnh trong commit này |
| `frontend/e2e/visual.spec.ts-snapshots/threads-1440.png` | /threads Sinh viên: như `home` | trước: ảnh ở commit `3f83d84` (git); sau: ảnh trong commit này |
| `frontend/e2e/visual.spec.ts-snapshots/inbox-1440.png` | /inbox Giảng viên: như `home` | trước: ảnh ở commit `3f83d84` (git); sau: ảnh trong commit này |
| `frontend/e2e/visual.spec.ts-snapshots/gradebook-1440.png` | /gradebook Giảng viên: như `home` | trước: ảnh ở commit `3f83d84` (git); sau: ảnh trong commit này |
| `frontend/e2e/visual.spec.ts-snapshots/settings-llm-1440.png` | /settings/llm Admin: như `home` | trước: ảnh ở commit `3f83d84` (git); sau: ảnh trong commit này |

## Gates
`pnpm lint`, `tsc` sạch; Playwright không `@real` / `visual`: **466 passed / 127 skipped** + 1 đỏ là ca `bell real` (đã sửa, 4 / 4 pass khi chạy riêng); `ui-antipatterns` 22 ✓, `ui-allow:` 9 → 9; `lint-selftest` 7 / 7 + 22 / 22. **Chưa có**: CLS `lhci`; CI GitHub của commit này.
CI của US-UI-02 (`3f83d84`): `Judge` + `Go` xanh, `Frontend` đỏ duy nhất ở ca `bell real` nói trên (lỗi nửa đêm, không phải hồi quy của story).

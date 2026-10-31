# DEV handoff — US-S8-UI (bỏ câu mô tả dưới tiêu đề ở cấu hình / tích hợp / bảo mật)
Nhánh: `sprint/8-p6`. Luật: `~/.claude/CLAUDE.md` "Luật cứng — chữ trên UI". Xoá hẳn, không dời vào tooltip.

## Đã xoá
| Màn | Câu | Cách |
| --- | --- | --- |
| `/settings/llm` | mô tả tiêu đề trang; mô tả 4 khối (Kết nối nhà cung cấp, Mô hình theo tác vụ, Chuỗi dự phòng, Mức dùng và ngân sách) | bỏ `description` ở `PageHeader` và 4 `Section` |
| `/settings/llm` (khung máy chủ) | cùng câu mô tả trong `LLM_INTRO` | bỏ; `RouteIntro.description` thành tuỳ chọn, `PreShell` chỉ vẽ `<p>` khi có |
| `/settings/integrations` | mô tả tiêu đề trang; dòng mô tả từng tích hợp (3) | bỏ `description` ở `PageHeader` và `Section`; xoá trường `purpose` chết trong `mock/system.ts` |
| `/settings` (bảo mật) | "Đổi mật khẩu và xem nơi tài khoản của bạn đang đăng nhập." | bỏ `description` |

Giữ nguyên (không phải câu mô tả tiêu đề): `InlineNotice` "Chỉ quản trị viên…", dòng "Đổi mục này cần lập chỉ mục lại tài liệu." trong khối Mô hình tìm kiếm (hệ quả của việc đổi), trạng thái rỗng / lỗi.

## Ảnh mốc, SRS, test
- Ảnh mốc `settings-llm-1440.png` / `-390.png` sinh lại trong `mcr.microsoft.com/playwright:v1.63.0-noble` (cài `pnpm` trong container, như US-UI AC9). Cả 14 ảnh `visual.spec.ts` trong image: 14 passed.
- `docs/specs/**`: không còn câu nào trong các câu đã xoá (grep chuỗi chính xác → 0 khớp).
- **Proposal #1** (`proposals.md`): ca `e2e/lcp.spec.ts` `lcp before refresh /settings/llm` đỏ vì ứng viên LCP sau phiên là thông báo lỗi khi test không giả API (trước đây câu mô tả ở khung máy chủ che đi). PM ACCEPTED: đã thêm `mockLlmApi(page)` cho ca đó trong `e2e/lcp.spec.ts` (commit ghi #1); assertion và ngưỡng không đổi.

## Kiểm
`tsc --noEmit`, `pnpm lint`, `ui-antipatterns.sh` sạch. Playwright (desktop + mobile, `--retries=0`) `lcp`, `settings-llm`, `account`, `panels`, `a11y`, `shell`: 161 passed, 130 skipped (ca chỉ chạy một dự án), 1 failed = ca LCP của proposal #1; sau khi sửa theo #1, `lcp.spec` desktop 7 passed.
Không đổi Go.

## Vòng 2 (sau QC FAIL vòng 1, `report-US-S8-UI.md`)
- **BUG-S8-UI-1 (đã sửa):** bỏ hẳn câu thường trực "Đổi mục này cần lập chỉ mục lại tài liệu." ở `EmbeddingSection.tsx`. Hệ quả của việc đổi vẫn nằm ở nơi luật cho phép: hộp thoại xác nhận khi đổi mô hình và `InlineNotice` điều kiện "Cần lập chỉ mục lại".
- **BUG-S8-UI-4 (đã sửa):** `/settings/integrations`: trạng thái `Đã kết nối` / `Chưa cấu hình` / `Chờ duyệt` (1–2 từ); bỏ `meta` "Kiểm gần nhất 08:55 hôm nay" dưới tiêu đề; giờ kiểm chỉ còn MỘT lần, thành dòng `Kiểm gần nhất — 08:55 hôm nay` trong danh sách trường của "Thư đi (SMTP)" (`mock/system.ts`, `IntegrationsSettings.tsx`).
- **BUG-S8-UI-2, BUG-S8-UI-3 (CHƯA sửa — chờ proposal #10):** hai test cũ khẳng định đúng các chữ cần đổi (`e2e/settings-llm.spec.ts:41`, `:138`). Theo luật mới của PM: mở proposal #10, không sửa test trước. Sau ACCEPTED dev sửa code + test cùng commit.
- Ảnh mốc: `visual.spec.ts` trong image `playwright:v1.63.0-noble`: 14 passed, `settings-llm-*.png` không đổi (hai thay đổi vòng này nằm dưới nếp gấp / ở màn không có ảnh mốc).
- Kiểm: `tsc`, `pnpm lint`, `ui-antipatterns.sh` sạch; Playwright `settings-llm`, `shell`, `lcp`, `a11y`, `panels` (desktop + mobile): 112 passed.

## Vòng 2b (proposal #10 ACCEPTED)
- **BUG-S8-UI-2:** "Chuỗi dự phòng": bỏ `InlineNotice` câu dài lặp mỗi dòng; mỗi tác vụ chỉ có MỘT mô hình hiện nhãn ngắn `Chưa có dự phòng` (`StatusText`, `RoutesSection.tsx`).
- **BUG-S8-UI-3:** trạng thái nhà cung cấp chỉ `Đã kết nối` (bỏ giờ kiểm; đã có nút `Test kết nối`); bỏ dòng `Khoá API •••••••• · đã kết nối`; bỏ dòng phụ trùng tên nhà cung cấp; dòng rỗng "Mặc định máy chủ" (ở "Mô hình theo tác vụ" / chuỗi dự phòng / mô hình tìm kiếm).
- **Test đã sửa theo #10:** `e2e/settings-llm.spec.ts`: `:41` → `toContainText("Đã kết nối")`; khối dự phòng → nhãn ngắn `Chưa có dự phòng` đếm đúng số dòng một mô hình (`toHaveCount(oneModel)`) và không còn câu "nếu nhà cung cấp lỗi". Không đổi gì khác.
- SRS `FEAT-llm-gateway` có câu cảnh báo cũ → BA vá SRS + Confluence (luật cứng PO).
- Kiểm: `tsc`, lint, `ui-antipatterns.sh` sạch; Playwright `settings-llm`, `shell`, `a11y`, `panels`, `lcp` (desktop + mobile): 112 passed; `visual.spec.ts` trong image `playwright:v1.63.0-noble`: 14 passed (ảnh `settings-llm-*` sinh lại).

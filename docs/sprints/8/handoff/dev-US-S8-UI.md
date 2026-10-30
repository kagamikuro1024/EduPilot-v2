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
- **Proposal #1** (`proposals.md`): ca `e2e/lcp.spec.ts` `lcp before refresh /settings/llm` đỏ vì ứng viên LCP sau phiên là thông báo lỗi khi test không giả API (trước đây câu mô tả ở khung máy chủ che đi). Đề xuất thêm `mockLlmApi(page)` cho ca đó; chờ ACCEPTED, **chưa sửa test**.

## Kiểm
`tsc --noEmit`, `pnpm lint`, `ui-antipatterns.sh` sạch. Playwright (desktop + mobile, `--retries=0`) `lcp`, `settings-llm`, `account`, `panels`, `a11y`, `shell`: 161 passed, 130 skipped (ca chỉ chạy một dự án), 1 failed = ca LCP của proposal #1. Với `mockLlmApi` thêm vào bản sao tạm của spec: ca xanh (ứng viên duy nhất là h1).
Không đổi Go.

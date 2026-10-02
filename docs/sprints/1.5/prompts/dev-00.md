# Prompt cho `dev` — US-PROTO-00 Nền prototype

1. Đọc `docs/team/CONTEXT.md` (mới), `docs/sprints/1.5/plan.md`, rồi `docs/design/DESIGN.md` §0–§13, §15–§22 và hướng thiết kế `frontend/.impeccable/surfaces/frontend-src-app.md`. Nếu có skill `impeccable`, đọc `reference/craft-floor.md` của nó trước khi sửa UI.
2. Nhánh `sprint/1.5-mock-ui` (đã có). Luật git như sprint 1: không `git add -A`, commit đúng đường dẫn, `US-PROTO-00: <việc>`.

## Việc (spec đầy đủ đang được BA viết ở `docs/specs/FEAT-prototype-ui/`; story này không cần chờ)
PM đã dựng **nháp chưa chạy thử** ở commit `eb488d6` (danh sách trong `plan.md`). Từ giờ bạn làm chủ nó — sửa, xoá, viết lại tuỳ bạn, miễn giữ hướng thiết kế và DESIGN.md.
- Cho nháp build + lint sạch (`pnpm -C frontend lint && pnpm -C frontend build`), `bash scripts/ui-antipatterns.sh` sạch (có mẹo: `<Dialog` ngoài `shared/ui/` chỉ được trong file tên `ConfirmIrreversible|PIIChannelDialog|FinalizeGrades|PublishGrades|ConfirmGradeScheme|DeleteDocument`; bóng chỉ trong file `shared/ui/(Popover|Dialog|Menu|Drawer|Composer)`; màu viết cứng chỉ trong `shared/styles/`).
- Route group `src/app/(app)/layout.tsx` (server) đọc cookie `ep_demo_role` / `ep_demo_course` (`src/shared/session/cookies.ts`) → `SessionProvider` + `AppShell`. Chuyển `/` hiện tại vào group. `/login` ngoài shell: chọn một trong 4 tài khoản demo → đặt cookie → về `/`; ghi rõ "Bản mô phỏng".
- Mỗi route trong `plan.md` có một trang tạm (PageHeader đúng tên tiếng Việt) để điều hướng / chặn quyền / ⌘K / bottom nav điện thoại chạy được ngay; các story sau thay nội dung.
- Kiểm tận mắt bằng trình duyệt ở 1440 px và 390 px: shell, đổi vai, chọn lớp, thông báo, ⌘K, thu gọn sidebar, bottom nav + "Thêm", màn chặn quyền. Lưu 4 ảnh chụp vào `docs/sprints/1.5/shots/00-*.png`.

## Khi xong
Handoff `docs/sprints/1.5/handoff/dev-US-PROTO-00.md` (mẫu TEMPLATES), trả lời ≤ 8 dòng, dừng. Được dùng subagent song song nếu cần.

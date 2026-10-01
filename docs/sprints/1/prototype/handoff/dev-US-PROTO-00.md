# DEV handoff — US-PROTO-00 Nền prototype
Nhánh `sprint/1-mock-ui` · commit mã cuối: xem `git log --oneline -4` (các commit `US-PROTO-00: …`).

## Đã làm (theo thứ tự lát dọc)
- Nháp `eb488d6` build + lint + `tsc` sạch sẵn; chỉ sửa chỗ vướng: 3 dòng `box-shadow: none` (CommandPalette, DataTable, AppShell) bị `ui-antipatterns.sh` bắt → thêm chú thích `ui-allow`.
- `src/app/layout.tsx`: nạp thêm `base.css`; tiêu đề `%s · EduPilot`. `next.config.ts`: `agentRules: false` (Next 16 tự sinh `frontend/AGENTS.md` + `CLAUDE.md` khi chạy dev), `devIndicators: false`.
- Nhóm route `src/app/(app)/layout.tsx` (server): đọc cookie `ep_demo_role` / `ep_demo_course` → `SessionProvider` + `AppShell`. **Không có cookie vai trò → chuyển về `/login`.** `parseRole` giờ trả `Role | null` (không còn mặc định `teacher`); ghi/xoá cookie gom ở `shared/session/cookies.ts` (`writeDemoCookie`, `clearDemoSession`). "Đăng xuất" xoá cookie rồi về `/login`.
- `/login` (ngoài shell): 4 tài khoản demo là hàng kẻ (Sinh viên, Trợ giảng, Giảng viên, Admin) → đặt cookie, lớp = INT1006 1 → `/`; ghi "Đây là bản mô phỏng: mọi dữ liệu đều là giả".
- 30 trang tạm (`PageHeader` đúng tên tiếng Việt, dùng chung `app/(app)/RouteStub.tsx`): `/`, chat, threads(+[id]), practice(+[attemptId], history), library, calendar, me, assignments/[id], join, inbox, students(+[id]), attendance, class/members, gradebook(+scheme), grading(+[submissionId]), questions, documents, insights, analytics, observability, settings/llm, settings/integrations, admin/courses, admin/users. Story sau thay nội dung từng file, giữ nguyên đường dẫn.
- Sửa lỗi tìm thấy khi chạy thật: thanh bên không vừa 900 px (bỏ khối tên lớp lặp với thanh trên, chặt khoảng cách); drawer rộng 354 px thay vì toàn màn hình trên 390 px (thiếu `max-width`); `<dialog>` trùng `id` tiêu đề (đổi `useId`); nhãn bottom nav "Hộp thư hỗ trợ" xuống dòng (thêm `short: "Hộp thư"`); cỡ chữ 10/11 px ngoài thang → `--ep-text-meta`; màn chặn quyền dùng `PageHeader` (có `h1`) thay `EmptyState`; thanh bên thu gọn hiện chấm đỏ thay số đếm; ⌘K tìm không dấu ("diem" ra "Điểm danh").

## File đổi
`frontend/next.config.ts`; `frontend/src/app/{layout.tsx,login/*,(app)/**}`; `frontend/src/shared/session/{cookies.ts,session.tsx}`; `frontend/src/shared/shell/{AppShell.tsx,AppShell.module.css,nav.ts}`; `frontend/src/shared/ui/{CommandPalette.tsx,CommandPalette.module.css,DataTable.module.css,Dialog.tsx,Dialog.module.css}`; `docs/sprints/1/prototype/shots/00-*.png`. Xoá `src/app/page.tsx` (chuyển vào `(app)/page.tsx`).

## Lệnh QC chạy để kiểm
```bash
source ~/.zprofile
pnpm -C frontend lint && pnpm -C frontend build && bash scripts/ui-antipatterns.sh   # sạch
pnpm -C frontend exec next dev -p 3100    # rồi mở http://localhost:3100 (hoặc `pnpm dev` → :3000)
```
Tay: `/` khi chưa có cookie → `/login`; chọn Giảng viên → `/`; thanh bên 5 nhóm; bấm chip lớp → đổi INT1006 2 (cookie `ep_demo_course`); chuông → 3 thông báo; ⌘K / Ctrl+K / `/` → gõ "diem" → Enter vào `/attendance`; menu tài khoản → đổi sang Sinh viên khi đang ở `/attendance` → về `/`; gõ thẳng `/inbox` khi là Sinh viên → màn "Vai trò Sinh viên không mở được trang này"; "Thu gọn" → thanh 72 px; 390 px: bottom nav 4 mục + "Thêm" mở drawer toàn màn hình.

## Test đã chạy và kết quả
- `eslint .` / `tsc --noEmit` / `next build` (31 route) / `ui-antipatterns.sh` (0 `✗`): xanh.
- Trình duyệt thật (Chromium, `next dev -p 3100`), 1440×900 và 390×844, console không lỗi: đã bấm đủ các mục "Lệnh QC" ở trên. Ảnh: `docs/sprints/1/prototype/shots/00-desktop-teacher-1440.png`, `00-desktop-student-blocked-1440.png`, `00-mobile-390.png`, `00-mobile-more-390.png`.
- Không có test tự động mới (nền giao diện, story sau mới có hành vi); Playwright thuộc PU.

## AC tự đánh giá
Theo prompt `dev-00.md`: build/lint/phản mẫu sạch ✓ · `(app)` layout đọc cookie + shell ✓ · `/login` ✓ · 30 route có trang tạm ✓ · kiểm tận mắt 1440 + 390 ✓ (shell, đổi vai, chọn lớp, thông báo, ⌘K, thu gọn, bottom nav + Thêm, chặn quyền).

## Nợ / chưa làm / cần hỏi
- Chưa kiểm bằng bàn phím từng primitive (Tab qua popover/palette, trả focus); ô tìm trong palette tắt vòng focus (`ui-allow`) — các story sau cần soát lại khi dùng.
- Thông báo chưa có trạng thái đã đọc; nhấn vào chỉ đi tới trang đích.
- Đổi vai trò ở trang có `[id]` chỉ kiểm tiền tố đường dẫn (`canOpen`), chưa kiểm sở hữu bản ghi.
- `frontend/Dockerfile` chưa được chạy lại với bản nền mới (chỉ `next build` cục bộ).
- Palette chỉ liệt kê mục điều hướng của vai trò; tìm sinh viên / tài liệu / thread sẽ thêm cùng story tương ứng.

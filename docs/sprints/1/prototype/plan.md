# Sprint 1 bổ sung — Prototype giao diện toàn bộ tính năng

Trạng thái: **ĐÃ DUYỆT** (chủ dự án yêu cầu 2026-10-01: "bổ sung vào sprint 1… giao diện mock với toàn bộ tính năng… bấm được, có tương tác giả lập") · Nhánh `sprint/1-mock-ui` từ `main` · Hạn: trước buổi thuyết trình cuối tuần.

## Mục tiêu
Một prototype chạy trong app Next.js thật (`pnpm dev` → http://localhost:3000), phủ mọi route dự định, dữ liệu mô phỏng, đổi vai Sinh viên / Trợ giảng / Giảng viên / Admin, tương tác giả lập bằng trạng thái cục bộ, đi trọn `docs/DEMO_SCRIPT.md`. Không backend, không gọi API.

## Đã có (PM dựng nháp — `dev` làm chủ từ giờ, được sửa)
Commit `eb488d6`: `frontend/PRODUCT.md`; hướng thiết kế `frontend/.impeccable/surfaces/frontend-src-app.md`; `src/mock/core.ts` (lớp, người, 57 SV giả, định dạng); phiên mô phỏng bằng cookie (`src/shared/session/`); app shell theo vai + điều hướng + chặn quyền (`src/shared/shell/`); primitive (`src/shared/ui/`: Button, Field, Page/PageHeader/Section/Toolbar/Split, ActionList, DataTable, Tabs/SegmentedControl/FilterChips, InlineNotice/StatusText/EmptyState/Skeleton/UndoLine, Dialog/Drawer, Popover/OverflowMenu, Composer, TrendChart/BarList, CommandPalette); Red Thread Transition (`src/shared/motion/`); hook chữ chảy (`src/shared/lib/useStreamedText.ts`). **Chưa build, chưa chạy thử.**

## Story
| # | Story | Phạm vi | Ai |
| --- | --- | --- | --- |
| 0 | US-PROTO-00 Nền | Sửa nháp cho build + lint + `ui-antipatterns` sạch; `(app)` layout đọc cookie, gắn `AppShell`; `/login` (chọn vai demo) | dev |
| 1 | US-PROTO-01 Sinh viên | `/` (SV), `/chat`, `/threads`, `/threads/[id]`, `/practice`, `/practice/[attemptId]`, `/practice/history`, `/library`, `/calendar`, `/me`, `/assignments/[id]`, `/join` | dev |
| 2 | US-PROTO-02 Giảng viên — vận hành lớp | `/` (GV/TA), `/inbox`, `/students`, `/students/[id]`, `/attendance`, `/class/members` | dev |
| 3 | US-PROTO-03 Đánh giá | `/gradebook`, `/gradebook/scheme`, `/grading`, `/grading/[submissionId]`, `/questions`, `/documents` | dev |
| 4 | US-PROTO-04 Hiểu lớp + hệ thống | `/insights`, `/analytics`, `/observability`, `/settings/llm`, `/settings/integrations`, `/` (Admin), `/admin/courses`, `/admin/users` | dev |
| 5 | US-PROTO-05 Kiểm | TC viết song song từ spec; chạy sau mỗi story; kịch bản demo đi trọn; 1440 + 390 px | qc |

`dev` được dùng subagent song song bên trong một story (nền trước, rồi chia route), nhưng giao handoff theo từng story. PM duyệt bằng ảnh chụp sau mỗi story.

## Quyết định PM
- Prototype nằm trong app thật, không làm file HTML riêng: chủ dự án muốn "khởi động stack lên xem"; primitive dùng lại ở PU.
- Không thêm thư viện ngoài `lucide-react` (đã có trong ARCHITECTURE). Không tailwind, không thư viện chart: CSS Modules + token, biểu đồ SVG tự vẽ.
- Mọi dữ liệu nằm ở `frontend/src/mock/*.ts`, ghi rõ là mô phỏng; giao diện có dòng "Bản mô phỏng · dữ liệu giả".
- Spec gộp một feature `FEAT-prototype-ui`, US theo nhóm route; AC bám DESIGN.md §14 (không chép lại, trỏ mục) + tương tác cụ thể + kịch bản dữ liệu. **Spec nằm ở `docs/sprints/1/prototype/spec/`, không ở `docs/specs/`** (chủ dự án chốt, D51): đây là spec của mock, spec thật viết lại theo từng sprint build thật.

# Sprint 10.7 (UI-R): ghi chú khảo sát bàn giao (techlead-mem → techlead-ui)

Đây **chưa phải hợp đồng §23**. PM chuyển việc sang techlead-ui khi Tech Lead mới đọc mã xong, chưa soạn nháp. Các số dưới đây đếm bằng `grep` trên `origin/main` @`e9af9d15`. Ước tính là ước tính, chưa đối chiếu ảnh.

## Hiện trạng primitive (`frontend/src/shared/ui/`)

| Primitive | Tệp | API hiện tại | Vì sao mỗi route ra một kiểu |
| --- | --- | --- | --- |
| `Page` | `Layout.tsx`, `Layout.module.css` | `width?: "reading" \| "wide" \| "full"` (mặc định `reading` = 960 px, **căn trái**), `className?` | 47 chỗ không truyền `width` nên ngầm là `reading`. 32 chỗ `wide`, 9 `full`, 1 `reading`. 4 chỗ truyền `className`: chat ×2, inbox, `me/RealMe` |
| `PageHeader` | `Layout.tsx` | `title, description?, meta?, actions?, back?`; hàng tiêu đề `align-items: flex-end; flex-wrap: wrap` | Khoảng 16 chỗ có `description`, 15 chỗ có `meta`. `actions` là ReactNode tự do: không giới hạn số nút, và nút rơi xuống dòng dưới khi hẹp |
| `Section` | `Layout.tsx` | `title, description?, action?, panel?` | Khoảng 5 chỗ có `description` (dòng phụ). Ngoài ra 23 chỗ tự dựng `<h2 className="ep-section-title">` ngoài `shared/ui`, bỏ qua `Section` |
| `Panel` / `PanelSection` | `Panel.tsx`, `.module.css` | Bản thân không đặt chiều cao | Panel «cao hết màn» là do CSS route: `inbox/RealInbox.module.css:21–34` (`height: calc(100dvh …)`, `[data-ep-panel] { height: 100% }`), `chat/ChatScreen.module.css:9`, `grading/SubmissionReview.module.css:11,58` |
| `EmptyState` | `Feedback.tsx:51`, `Feedback.module.css:36` | `title, children?, action?, icon?`; CSS `justify-items: start; align-self: start; max-width: 56ch` | 96 chỗ gọi, khoảng 58 chỗ có thân chữ, 11 chỗ có icon. Mặc định **căn trái-trên**, nên ra lỗi bố cục (3) của `ui-empty` |
| `PageState` | `PageState.tsx` | Rỗng mặc định = `<EmptyState title="Chưa có gì ở đây">Khi có dữ liệu…</EmptyState>`. Chú thích ghi «câu dạy bước kế» | Mặc định trái luật chữ UI của PO |
| `Split` | `Layout.tsx` | `main, aside, ratio` | **Không route nào dùng.** Chat, inbox, duyệt bài đều tự dựng chia đôi bằng CSS riêng |

`meta` lặp lớp: thấy ở `inbox/RealInbox.tsx:103` (`meta={<span>{courseLabel}</span>}`). Còn cần quét hết.

Độ rộng tự đặt trong CSS route: 73 dòng `max-width: <số>` ở 41 tệp `*.module.css`. Nhiều nhất ở `exam/Exam`, `exam/take/Take`, `members/ClassSettings`, `chat/ChatScreen`, `practice/Practice`, `gradebook/Gradebook`. Không phải dòng nào cũng là khung: có cả cột chữ, ô nhập.

## Xung đột phải chốt trước khi viết §23

- **«`read` căn giữa» (plan.md) trái với 00-AC11.** AC này (`FEAT-ui-foundation/US.md:219` AC13, `FEAT-ui-panels/US.md:114` AC3) ghi: mép trái `h1` là 240 px ở 1440 trên **mọi** route, và `Layout.module.css:1` có chú thích «không căn giữa». Nếu chọn căn giữa thì BA phải vá ba AC đó, và QC phải đổi phép đo `LEFT` của `audit.mjs`.
- **DESIGN.md mâu thuẫn với luật PO.** §10.2 ghi «One short supporting sentence only if needed». §15 «Empty states: Teach the next action», kèm ví dụ «Good» là một câu giải thích dài. `UX.md:21` (quy tắc 5): «mỗi trạng thái rỗng nói *vì sao rỗng*». `UX.md:82` (cổng UX). Skill impeccable `operate.md` cũng nói «empty states that teach». Luật PO (CLAUDE.md 09/10/2026) thắng; cả bốn chỗ phải vá.
- **Script hiện có.** `scripts/ui-antipatterns.sh` có 23 phép. Phép 23 chỉ bắt `description` tĩnh > 40 ký tự. Đã có cơ chế `--selftest` (mỗi phép một hạt giống) nên luật mới cần thêm hạt giống vào mảng `seeds`.

## Hướng Tech Lead định đề xuất (techlead-ui tự quyết)

1. Ép bằng **kiểu TypeScript** trước, script sau:
   - `Page.width: "work" | "read"` bắt buộc, bỏ `className`;
   - bỏ `PageHeader.description` / `actions`, thay bằng `primary` (một nút) + `secondary` (tuple ≤ 2, dạng dữ liệu để thu vào menu khi < 720 px);
   - bỏ `Section.description`;
   - `EmptyState` chỉ còn `title` + `action?`, không `children`, không `icon`.

   `tsc` trong `pnpm build` sẽ chỉ ra mọi chỗ gọi.
2. Chia đôi cao hết màn chỉ qua `Split` (thêm `fill`, `ratio="list-detail"`). Xoá CSS chiều cao ở inbox, chat, duyệt bài.
3. Script chỉ bắt những gì kiểu không bắt được:
   - độ rộng ≥ 480 px / `vw` trong CSS route;
   - `height` / `min-height` `100%` / `100dvh` ngoài `Split`;
   - `meta` chứa `courseLabel` / `classCode` / `course?.code|name`;
   - `<h2 className="ep-section-title">` tự dựng;
   - nút rỗng trùng nhãn nút chính đầu trang trong cùng tệp.
4. Commit 1 chỉ đổi API primitive + sửa máy móc mọi chỗ gọi, để build xanh. Sau đó mỗi nhóm route một commit.

## Phân nhóm route (đề xuất kiểu: `work` / `read` / `work·split`)

| Nhóm | Route |
| --- | --- |
| G1 Hôm nay | `/` (3 vai): read |
| G2 Hỏi đáp | `/chat` work·split · `/inbox` work·split · `/threads` work · `/threads/[id]` read |
| G3 Lớp & người | `/students` work · `/students/[id]` work · `/class/members` work · `/attendance` work · `/class/settings` read · `/join`, `/join/[code]` read |
| G4 Đánh giá | `/grading` work · `/grading/[submissionId]` work·split · `/gradebook` work · `/gradebook/scheme` read · `/assignments/[id]` read · `/exams`, `/exams/[id]`, `/exams/[id]/results`, `/exams/[id]/similarity` work · `/exams/[id]/take` read · `/questions` work · `/practice` read · `/practice/[attemptId]` read · `/practice/history` work · `/me` read |
| G5 Nội dung | `/documents` work · `/library` work · `/library/[id]` read · `/calendar` work |
| G6 Hiểu lớp & hệ thống | `/insights` read · `/analytics` read · `/observability` work · `/settings` read · `/settings/llm` work · `/settings/integrations` read · `/admin/courses`, `/admin/users` work |
| G7 Ngoài app | `login`, `register`, `forgot-password`, `reset-password`, `verify-email`, `invite/[token]`, `error`, `not-found`, `RouteStub`: chỉ áp luật rỗng / tiêu đề |

Kết quả của persona-eval (`research/persona-eval`, §3.1) liên quan trực tiếp tới UI-R: trạng thái rỗng «Chưa có bài nộp.» sai nghĩa ở `/grading`, Markdown thô ở `/assignments/[id]`, hai nút «Tải bài lên» khi hàng chờ rỗng, nút ở «Hôm nay» lặp tiêu đề.

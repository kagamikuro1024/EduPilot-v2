# Nháp `docs/design/DESIGN.md` §23 + các chỗ vá kèm (Tech Lead UI, sprint 10.7, chờ PM duyệt)

Phần 1 là nguyên văn đề nghị thêm vào `DESIGN.md` (tiếng Anh, cùng ngôn ngữ với tệp gốc). Phần 2 là các dòng cũ của `DESIGN.md` / `UX.md` phải vá vì trái §23 hoặc trái luật chữ UI của chủ dự án (`~/.claude/CLAUDE.md`, 09/10/2026). Lý do và số đo: `techlead.md` TL-1. Mẫu nhìn được: `mau/khung-trang.html`.

---

## Phần 1. Nguyên văn §23

````markdown
# 23. Page frame contract

Every signed-in route is built from the same frame. A route chooses a **kind** and fills slots; it never sets width, padding, height, alignment or title typography of the frame. The primitives in `frontend/src/shared/ui/` own all of that, and `frame.spec.ts` measures it on every route.

## 23.1 Page: exactly two kinds

| Kind | Use when | Content width | At 1440 (sidebar 216) |
|---|---|---|---|
| `work` | The body holds a table, a list with row actions, a grid, or two panes | fills the main area, capped at `--ep-page-work` (100 rem) | x 240 → 1416, width 1176 |
| `read` | The body is one column: a report, a form, a settings page, a document, a single task | `--ep-page-read` (50 rem) | x 428 → 1228, width 800 |

- The page is always centered in the main area (`margin-inline: auto`). Side padding is `--ep-space-6` (`--ep-space-3` below 720 px). Top padding `--ep-space-10`, bottom `--ep-space-20`.
- Below 1100 px both kinds are the same: full width minus side padding.
- Decision rule, in order: (1) a `DataTable`, a grid (calendar, attendance, gradebook) or a `Split` → `work`; (2) otherwise → `read`. A route never mixes kinds between its states (data, empty, loading, error). `/` is a different screen per role, so its kind is set per role.
- `--ep-page-work` and `--ep-page-read` are read only by `shared/ui/Layout.module.css`. `Page` accepts no `className` and no width prop.

## 23.2 PageHeader: one pattern

```
[← back link]                                   optional, its own row
Title (h1)                    [secondary ≤ 2] [primary]     row 1, min-height 36 px
meta · meta · meta                                          row 2, optional, one line
```

- Title: `h1.ep-page-title`, one per route, rendered only by `PageHeader`.
- No description sentence. The `description` prop does not exist.
- Meta: at most one line of data (counts, dates, a status). It truncates with an ellipsis, never wraps at ≥ 720 px. It never repeats the course code or name shown in the top bar, and never contains a sentence.
- Actions sit on the title row, right-aligned to the page edge: at most one primary (rightmost) and two secondary. Anything else goes into the overflow menu. Actions are passed as data (`primary`, `secondary`, `more`), not as free JSX.
- A route with one working region puts its primary action in the header, never in the region's toolbar and never again in the empty state.
- Row 1 has the same height with or without actions, so the body starts at the same y on every route (two values only: without meta, with meta).
- Below 720 px: title, then meta, then one action row (primary first, full width; secondary actions collapse into one "Thêm" menu button).
- Header → body gap `--ep-space-10` (`--ep-space-8` below 720 px).

## 23.3 Body: sections and panels

- One working region = one `Panel`. A `Panel` is always as wide as the page column, as tall as its content, with one padding (`--ep-space-6`; `--ep-space-4` below 720 px). `Panel` has no `padding` and no `className` prop.
- The page ground and the panel are both white (`--ep-canvas` = `--ep-surface`). A panel is therefore always drawn with its 1 px `--ep-panel-border` and its static `--ep-elevation-1`; no route removes either.
- A page with one region has no `h2`: the `h1` names it. A page with two or more regions gives each a `Section` whose `h2.ep-section-title` sits outside the panel. `h2` is rendered only by `Section`.
- A section title has no second line. `Section` has `title` and at most one `action`; the `description` prop does not exist. Data that used to live there moves into the panel or into the header meta.
- Section → section gap `--ep-space-12`; title → panel gap `--ep-space-4`.
- Page-level `Tabs` or `Toolbar` (they control the whole body) sit between header and body. A toolbar that controls one table sits inside that table's panel, first. `Toolbar.end` holds table controls only (a switch, a sort); page actions belong to the header (§23.2).
- Filter state of page-level `Tabs` and `Toolbar` lives in the URL query, not in `sessionStorage`: leaving the route, reloading and sharing the link all keep it.
- The result line of a background job ("Đã chấm xong 12 bài") is a quiet confirmation line directly under the region's toolbar, inside the panel. It reads the job state, so a reload shows it again; it never shifts the layout.
- Side-by-side regions exist only through `Split`:
  - `ratio="main-aside"`: main `1fr`, aside 340 px, gap `--ep-space-6`; stacks below 1100 px.
  - `ratio="list-detail"` with `fill`: one panel, list 380 px + detail, as tall as the viewport (`100dvh − top bar − header`), panes scroll on their own; below 1100 px it becomes list → detail navigation. `fill` applies only while the list has rows. An empty or failed list renders a normal content-height panel.
  - Viewport-relative heights are allowed nowhere else.

## 23.4 EmptyState: one pattern

- A short title (≤ 40 characters, no full stop) and at most one button. Centered horizontally in its container, vertical padding `--ep-space-16`. No body text, no icon, no illustration.
- The button exists only when no control already visible on the screen does the same thing. It is never the header's primary action, never a duplicate of a tab or filter, and is always `variant="secondary"`.
- The title states the fact ("Chưa có câu hỏi", "Không có câu hỏi khớp bộ lọc"). It does not explain the feature.
- A reason the user needs in order not to make a mistake (why an action is locked) goes into the tooltip of that control, per the UI text rule.
- The container keeps its frame: same panel, same width, content height. A panel is never stretched to fill the screen around an empty state.
- `EmptyState` takes `title` and `action`; it has no `children` and no `icon`. `PageState.empty` takes the same two fields as data.

## 23.5 Loading and error

- Loading: the header renders immediately; the body shows a `Skeleton` inside the same panel frame the data state will use. The `h1` and the panel edges do not move between loading and data (CLS ≤ 0.1).
- Error: one `InlineNotice tone="danger"` as wide as the page column, in place of the region: one line saying what failed, and `Thử lại`. A second sentence is allowed only when the screen holds unsaved input ("Chữ bạn đã nhập vẫn còn").
- Loading, empty and error never change the page kind, the header, or the tabs.

## 23.6 Density

Fixed values; a route does not override them.

| Thing | Value |
|---|---|
| Control height (button, input, select) | 36 px; 44 px below 720 px or on coarse pointers |
| Small button | 30 px (inside rows and notices only) |
| Table row | 48 px; `dense` 44 px |
| List row (`ActionList`) | min 56 px |
| Text sizes in `main` | the seven roles of §5, nothing else |
| Gap between sibling controls | `--ep-space-2` |

## 23.7 Enforcement

1. Types: the removed props make every old call site a compile error.
2. `scripts/ui-antipatterns.sh` checks 24–31 (frame width, viewport height, reaching into primitives, meta repeating the course, hand-made titles, empty-state duplicates, long titles, yellow as text).
3. `frontend/e2e/frame.spec.ts` measures the rules above in the DOM on every route × role × 1440 / 375 × data / empty / loading / error. A route is done when it has zero deviations.
````

---

## Phần 2. Dòng cũ phải vá cùng đợt

| Tệp · vị trí | Cũ (nguyên văn) | Mới | Vì sao |
| --- | --- | --- | --- |
| `DESIGN.md` §10.2 | `- One short supporting sentence only if needed.` | `- No supporting sentence. At most one line of data meta (§23.2).` | Luật chữ UI: không câu mô tả dưới tiêu đề |
| `DESIGN.md` §10.2 | `- Optional compact secondary action(s), maximum 2 visible.` | giữ, thêm: `Layout and limits: §23.2.` | Trỏ về hợp đồng |
| `DESIGN.md` §10.14 | `` `padding` `md \| lg \| none` (`--ep-space-6` / `--ep-space-8` / 0) `` | `` one padding (`--ep-space-6`); flush content only through `Split` (§23.3) `` | `lg` không nơi nào dùng; `none` có 2 chỗ, đều là chia đôi |
| `DESIGN.md` §15 «Empty states» | `Teach the next action.` + ví dụ «Good» hai câu | `State the fact in a short title; at most one button (§23.4).` Ví dụ Good: `Chưa có câu hỏi cần xử lý` | Luật chữ UI thắng |
| `DESIGN.md` §15 «Errors» | ví dụ ba câu «Không tải được bài nộp từ IMAP. Kết nối vẫn được giữ. …» | `Không tải được bài nộp · Thử lại` + trỏ §23.5 | Một dòng + một nút |
| `DESIGN.md` §21 | *(thêm 5 dòng)* | `- A route that sets its own frame width, padding or viewport height.` / `- A sentence under a page or section title.` / `- An empty state with body text, an icon, or a button that repeats a header action or a tab.` / `- Header meta that repeats the selected course.` / `- Yellow used as text, or more than three yellow marks in one viewport.` | Phản mẫu mới |
| `DESIGN.md` §22 | *(thêm điều 11)* | `11. The route passes the page frame contract (§23) with zero deviations in `frame.spec.ts`.` | Định nghĩa xong |
| `DESIGN.md` §4 bảng màu + «Color rules» | xem `mau/palette.md` mục 2 và 4 | | Bộ màu Bách khoa |
| `UX.md` mục 2 quy tắc 5 | `Mỗi trạng thái rỗng nói *vì sao rỗng* và có *một nút hành động kế tiếp*. Mỗi lỗi nói *chuyện gì xảy ra*, *dữ liệu của bạn có an toàn không*, và có nút *Thử lại*.` | `Mỗi trạng thái rỗng là một tiêu đề ngắn nói rõ đang rỗng cái gì, và có tối đa một nút khi màn chưa có nút nào làm việc đó (DESIGN §23.4). Mỗi lỗi nói chuyện gì xảy ra trong một dòng và có nút *Thử lại*; chỉ nói thêm về dữ liệu khi màn đang giữ chữ chưa lưu (§23.5).` | Luật chữ UI thắng (P-107-2, P-107-3) |
| `UX.md` mục 6 dòng 3 | `… và trạng thái rỗng có nút hành động` | `… và trạng thái rỗng theo DESIGN §23.4` | Như trên |
| `AGENTS.md` «Luật giao diện» | *(thêm 1 dòng)* | ``- Khung trang theo `DESIGN.md` §23: `Page` kiểu `work` hoặc `read`, một mẫu đầu trang, panel rộng đúng cột và cao theo nội dung, trạng thái rỗng chỉ có tiêu đề + tối đa một nút. Trang không tự đặt độ rộng, khoảng đệm, chiều cao theo màn.`` | Luật hay bị quên |

Các AC đã duyệt bị ảnh hưởng (00-AC11, `FEAT-ui-foundation` US-PU-04 AC13, `FEAT-ui-panels` US-UI-02 AC3 và US-UI-03 AC3, `FEAT-ui-foundation` AC11 «route chưa có backend»): xem `proposals.md` P-107-1 … P-107-5. Dev không sửa mã trước khi PM quyết các dòng đó.

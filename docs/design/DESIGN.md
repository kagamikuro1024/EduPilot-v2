# EduPilot — Product UI Design System

> Direction: **Red Thread / Academic Instrument**  
> Surface mode: **Operate**  
> Language: **Vietnamese-first**  
> Audience: Students, Teaching Assistants, Teachers, Admins  
> Product truth source: `PRD.md` (EduPilot v2, 2026-09-20)

---

## 0. Design thesis

EduPilot is not an ERP dashboard and not a chatbot with extra menus. It is a **classroom operating surface** that helps a student know the next useful learning action and helps a teacher know the next decision that requires human judgment.

The interface should feel like a precise academic instrument with a human red-pen signal running through it. The white surface carries the work; the red thread marks the current path, action, correction, or important state. Most content is black/ink, not colored.

### One-sentence product experience

**Open EduPilot, understand what matters now, complete one clear task, and move on.**

### What this direction refuses

- No generic SaaS hero-metric dashboard.
- No card wall.
- No card-inside-card.
- No glassmorphism.
- No gradient text.
- No decorative AI neon.
- No icon-only navigation for primary desktop tasks.
- No giant rounded rectangles everywhere.
- No decorative shadows on ordinary surfaces.
- No badges for information that can be plain text.
- No showing every filter/action/state at once.

---

## 1. Product hierarchy

EduPilot has four jobs. Every screen must clearly belong to one.

1. **Learn** — private AI chat, practice, library, calendar, personal progress.
2. **Discuss** — Threads, Verify, citations, public academic discussion.
3. **Operate class** — inbox/escalation, attendance, students, gradebook, grading, questions, documents.
4. **Understand system** — insights, analytics, observability, LLM/integration settings.

Do not make all four jobs equally visible to every role.

### Role entry points

#### Student
Primary question: **“Hôm nay mình nên làm gì?”**

Primary navigation:
- Hôm nay
- Chat riêng
- Threads
- Luyện đề
- Thư viện
- Lịch
- Kết quả của tôi

Student should not see admin concepts such as “confidence threshold”, “PII event”, “RAG”, “provider”, “fallback”, or “trace” unless a transparent explanation is needed.

#### Teacher / TA
Primary question: **“Việc nào đang cần tôi quyết định?”**

Primary navigation:
- Hôm nay
- Hộp thư hỗ trợ
- Sinh viên
- Điểm danh
- Sổ điểm
- Chấm bài
- Ngân hàng câu hỏi
- Tài liệu
- Lịch
- Insights
- Analytics

Teacher-only / system tools live under **Hệ thống**:
- Observability
- Cấu hình LLM
- Tích hợp

#### Admin
Primary question: **“Hệ thống có đang vận hành đúng và an toàn không?”**

Primary navigation:
- Observability
- Cấu hình LLM
- Tích hợp
- Audit / health-related entry points when implemented

---

## 2. Navigation model

### Desktop shell

Use a stable, familiar app shell:

- Left sidebar: 216px expanded, 72px collapsed.
- Top utility bar: 56px.
- Main content max width depends on task:
  - Reading / forms: 960px.
  - Tables / gradebook / attendance: fluid, up to viewport width minus shell.
  - Side-by-side grading: full available width.
- Optional contextual right panel only when the task actually needs simultaneous context.

### Sidebar rules

The sidebar is not a decorative object. It is an index.

- White or very light warm-neutral ground.
- One 1px vertical divider.
- Text labels remain visible in default desktop state.
- Current item gets:
  - stronger text,
  - a 2px red indicator on the left OR a subtle red text state,
  - never a large filled red pill.
- Group secondary routes under collapsible sections when there are too many items.
- Do not show counters unless they affect action priority. Example: Inbox `5` is useful; Documents `24` is not.

### Top bar rules

Only global utilities:
- Course selector.
- Command/search trigger (`⌘ K` or `/`).
- Notifications.
- Profile.

Do not duplicate route titles in the top bar.

### Mobile shell

- Bottom nav: maximum 5 primary destinations.
- Remaining routes behind “Thêm”.
- Top bar: course, page title, one most important contextual action.
- Tables become list/detail or horizontal scroll only when data comparison requires columns.

---

## 3. Visual world: Red Thread / Academic Instrument

### Visual metaphor

The “red thread” is not decoration. It has three meanings only:

1. **Where I am** — active navigation / current step.
2. **What needs action** — primary CTA / unresolved state.
3. **What was corrected or verified** — academic review signal.

If a red element does not express one of these, remove it.

### Surface character

- Mostly light.
- Precise alignment.
- High contrast text.
- Thin rules.
- Small radii.
- No floating-card aesthetic.
- Use whitespace before containers.
- Use proximity before borders.
- Use hierarchy before color.

---

## 4. Color system

Use OKLCH in implementation where supported. Fallback HEX values are supplied only for compatibility.

### Core palette

| Token | OKLCH intent | Fallback | Use |
|---|---:|---:|---|
| `--ep-paper` | `oklch(99% 0.006 25)` | `#fffafa` | Main canvas |
| `--ep-surface` | `oklch(100% 0 0)` | `#ffffff` | Inputs, elevated focus surfaces |
| `--ep-surface-subtle` | `oklch(97% 0.008 25)` | `#f9f3f3` | Sidebar / grouped secondary region |
| `--ep-ink` | `oklch(19% 0.02 25)` | `#251d1e` | Primary text |
| `--ep-ink-2` | `oklch(40% 0.018 25)` | `#65595b` | Secondary text |
| `--ep-ink-3` | `oklch(53% 0.014 25)` | `#746968` | Metadata / disabled labels — hạ từ 58 % (`#958a8c`, 3,9 : 1) để chữ phụ đạt WCAG AA 4,5 : 1 (góp ý #25, PM ACCEPTED) |
| `--ep-rule` | `oklch(89% 0.012 25)` | `#e8dddd` | Dividers |
| `--ep-red` | `oklch(54% 0.22 27)` | `#c81d32` | Primary brand/action |
| `--ep-red-hover` | `oklch(49% 0.22 27)` | `#af1729` | Primary hover |
| `--ep-red-soft` | `oklch(96% 0.025 25)` | `#fff0f1` | Selected / mild warning context |
| `--ep-green` | `oklch(49% 0.13 145)` | `#287a4a` | Verified / success |
| `--ep-amber` | `oklch(63% 0.14 75)` | `#a86600` | Needs review / warning |
| `--ep-blue` | `oklch(51% 0.12 245)` | `#356d9e` | Informational only |

### Color rules

- Red is not a background theme. It is a signal.
- Aim for < 8% red area on ordinary application screens.
- Use `red-soft` only behind selected/attention content, never as default card fill.
- Semantic colors never compete with red CTA. If warning/error is primary on a screen, primary action can become neutral-dark.
- Secondary text on a red surface must be tinted from the red/foreground relationship, not generic gray.
- Never use pure `#000` as body text.

---

## 5. Typography

### Family

Use one family across product UI:

```css
font-family: "Be Vietnam Pro", "Noto Sans", ui-sans-serif, system-ui, sans-serif;
```

Why:
- Designed for Vietnamese readability.
- Good numeral and diacritic clarity.
- Works for dense product UI without needing a decorative display face.

Load only weights: `400`, `500`, `600`, `700`.

### Fixed type roles

| Role | Size | Weight | Line height | Use |
|---|---:|---:|---:|---|
| Page title | 28px | 650/700 | 1.25 | One per route |
| Section title | 20px | 600 | 1.35 | Major groups |
| Item title | 16px | 600 | 1.4 | Rows / object titles |
| Body | 15px | 400 | 1.6 | Default text |
| UI label | 14px | 500 | 1.4 | Buttons / inputs / nav |
| Metadata | 12px | 500 | 1.45 | Time, source, helper state |
| Data emphasis | 24px | 650 | 1.15 | Only when a number is itself the task |

Rules:
- Do not use fluid/clamp typography in authenticated app surfaces.
- No uppercase eyebrow above page headings.
- Headings use sentence case.
- Prefer weight + spacing changes over adding more sizes.
- Table numerals should use `font-variant-numeric: tabular-nums`.

---

## 6. Spacing and rhythm

4px base scale:

```text
2, 4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 80
```

Semantic use:
- `4`: icon/text micro gap.
- `8`: compact inline relation.
- `12`: control inner rhythm.
- `16`: row/item padding.
- `24`: related group gap.
- `32`: section internal gap.
- `48`: major section separation.
- `64`: large route separation / empty state breathing room.

Rules:
- More space above a heading than below it.
- Related items stay tighter than adjacent groups.
- Do not solve weak hierarchy by putting every group in a bordered box.

---

## 7. Shape and depth

### Radius

```css
--radius-xs: 4px;
--radius-sm: 6px;
--radius-md: 8px;
--radius-lg: 12px;
--radius-pill: 999px;
```

Usage:
- Buttons / inputs: 6–8px.
- Panels: 8–12px only where a true contained surface is necessary.
- Tags/status chips: pill allowed.
- Avoid 16–24px generic SaaS rounding.

### Shadow

Default: none.

Allowed:
- Menus/popovers/dialogs only.
- Sticky floating composer only if separation from scrolling content is otherwise unclear.

Shadow must have offset + soft blur, e.g.:

```css
box-shadow: 0 10px 30px rgba(71, 32, 37, 0.12);
```

No colored glow.

---

## 8. Iconography

- Use Lucide or Phosphor, one library only.
- Stroke: 1.7–2px.
- Default size: 18px.
- Icons support text; they do not replace labels for primary desktop navigation.
- Avoid emoji and Unicode symbols as functional icons.
- Use filled icon only for one or two strong states such as verified/notification unread when necessary.

---

## 9. Logo system

The EduPilot mark represents:
- an open learning path,
- a red teacher annotation stroke,
- forward guidance without replacing the teacher.

The mark should remain recognizable at 20px.

### Lockup

- Mark + `EduPilot` wordmark.
- Wordmark weight 650/700.
- Tight but readable tracking (`-0.02em`).
- Use ink wordmark + red mark on light surfaces.
- On red surface use white mark + white wordmark.

Do not put the logo inside a rounded-square tile.

Assets:
- `logo-edupilot.svg`
- `logo-edupilot-mark.svg`
- `favicon.svg`

---

## 10. Component vocabulary

The product should be built from a small reusable vocabulary. Do not create route-specific variants unless the shape expresses a genuinely new task.

### 10.1 AppShell

Contains sidebar, top bar, main scroll region, optional contextual panel.

States:
- Desktop expanded.
- Desktop collapsed.
- Tablet compact.
- Mobile bottom-nav.

### 10.2 PageHeader

Anatomy:
- Page title.
- One short supporting sentence only if needed.
- Primary action on the right if route has a global primary action.
- Optional compact secondary action(s), maximum 2 visible.

Do not add a KPI row directly under every PageHeader by habit.

### 10.3 ActionList

Use for teacher “Hôm nay” and Inbox.

Anatomy:
- urgency/state marker,
- title,
- one-line context,
- time/source metadata,
- one direct action.

Rows are separated by rules, not cards.

### 10.4 DataTable

Use when comparison across rows/columns is core to the task.

Rules:
- Sticky header for long tables.
- Row height 44–52px.
- Minimal vertical rules; usually none.
- Highlight current edited row with `surface-subtle`, not a box shadow.
- Bulk actions appear only after selection.
- Column filters live in a compact toolbar or popover.

### 10.5 InlineNotice

Use instead of a modal for most warnings/information.

Anatomy:
- icon,
- concise title/message,
- optional action.

Examples:
- Grade scheme not confirmed.
- PII was hidden.
- Answer key is excluded from student RAG.

### 10.6 StatusText / StatusChip

Prefer plain status text with a small dot for ordinary state.
Use pill chip only when the item must remain recognizable while scanning dense data.

Examples:
- `Đã xác nhận` green dot.
- `Cần duyệt` amber dot.
- `Chưa xử lý` red dot.

### 10.7 SegmentedControl

Use for 2–4 mutually exclusive modes:
- Tuần / Tháng / Danh sách.
- Theo chủ đề / Thi thử.

Do not use tabs when the control changes a local view rather than navigation.

### 10.8 Tabs

Use when switching major peer subviews within one object.
Example student profile: Tổng quan / Chuyên cần / Điểm / Hoạt động.

### 10.9 Field

Label above control.
Helper/error below only when needed.
Required state expressed in text or semantic marker, not only color.

### 10.10 Drawer

Use for contextual review that should not destroy list position:
- Student quick profile.
- Thread verification details.
- Request trace detail in observability.

Desktop width 420–520px.
Mobile becomes full-screen sheet/page.

### 10.11 Dialog

Use only for protected/interruption tasks:
- destructive confirmation,
- PII channel decision,
- finalize grades,
- publish grades.

Do not use dialogs for basic edit forms.

### 10.12 Composer

Chat/thread composer is a distinct working surface.
- Maximum width 820px.
- One primary send action.
- Attach/source controls remain secondary.
- PII state appears inline above composer when triggered.

### 10.13 Skeleton

Use shape-matched skeletons, not centered spinners.

---

## 11. Motion

Purpose: communicate state, not decorate.

Timing:
- Hover/focus: 120–160ms.
- Local state change: 160–220ms.
- Drawer/dialog: 200–250ms.

Easing:

```css
--ease-out: cubic-bezier(.16, 1, .3, 1);
--ease-standard: cubic-bezier(.2, .8, .2, 1);
```

One authored signature interaction:

**Red Thread Transition** — when the user chooses the next recommended task from “Hôm nay”, a thin red line briefly extends from the selected row toward the destination title before the content settles. It must last < 300ms and respect `prefers-reduced-motion`.

No staggered page-load choreography.
No bounce/elastic easing.

---

## 12. Global interaction rules

- One obvious primary action per screen or working state.
- Secondary actions: max 2 visible near the primary task.
- Everything else moves to overflow/context menus.
- Prefer inline edit to modal.
- Prefer progressive disclosure to showing every setting.
- Smart defaults first; ask only when uncertainty matters.
- Preserve list position when opening/closing detail views.
- Keyboard focus is always visible.
- Clickable rows must have clear hover/focus, but row text remains selectable where useful.

---

## 13. Global language / UX copy

UI is Vietnamese-first and plainspoken.

Translate technical system concepts into user intent:

| Internal concept | Student-facing | Teacher-facing |
|---|---|---|
| Escalation | Cần giảng viên hỗ trợ | Cần xử lý |
| Confidence | Mức độ chắc chắn | Độ tin cậy |
| PII redaction | Đã ẩn thông tin cá nhân | Đã bảo vệ thông tin cá nhân |
| RAG source | Nguồn tham khảo | Nguồn dùng để trả lời |
| Fallback model | — | Model dự phòng |
| Trace | — | Chi tiết yêu cầu |
| Verify | Đã được giảng viên xác nhận | Xác nhận |
| Reject | Không hiển thị | Loại khỏi tri thức |

Rules:
- Button = verb/action: `Duyệt bài`, `Gửi trả lời`, `Xác nhận công thức`.
- Error states say problem + recovery.
- Avoid “Thành công!” toasts for ordinary saves; use quiet inline confirmation unless the action is high stakes.

---

# 14. Route-by-route UI contract

Each route below describes the primary job, first viewport, essential objects, and what must be hidden until needed.

## 14.1 `/` — Hôm nay

### Student
Primary job: know the next useful learning action.

First viewport:
1. Greeting + date, quiet.
2. **One recommended action** with reason and estimated time.
3. Today timeline: next class/deadline/exam.
4. Recent learning continuation.

Do not show:
- dashboard metric cards,
- weekly charts above the fold,
- more than one primary CTA.

Recommended action examples:
- “Ôn lại Mật mã đối xứng — bạn sai 4/7 câu gần nhất.”
- “Bài tập 03 còn 18 giờ — tiếp tục từ câu 5.”

### Teacher / TA
Primary job: resolve work requiring human judgment.

First viewport:
1. `5 việc cần xử lý hôm nay`.
2. ActionList ordered by urgency and consequence.
3. A narrow “Lớp cần chú ý” summary below the action list.
4. Upcoming class/deadline strip.

No hero metrics.

---

## 14.2 `/chat` — Chat riêng

Primary job: ask personal/course questions safely.

Layout:
- Conversation column centered, max 840px.
- Left history collapsible on desktop; hidden on small screens.
- No permanent right sidebar.
- Citations expand inline under AI answer.
- Tool result (attendance/grades/calendar) renders as compact structured block, not a dashboard card.

PII:
- After redaction, show a compact inline line: `Đã ẩn 2 thông tin cá nhân trước khi gửi cho AI` with `Tìm hiểu`.
- Never show placeholders like `[[SV_1]]`.

Confidence:
- Student sees language such as `AI chưa đủ chắc chắn về câu này` and a clear `Nhờ giảng viên hỗ trợ` action.
- Do not show raw 0.76 bar by default.

---

## 14.3 `/threads` — Threads

Primary job: browse and ask public academic questions.

Layout:
- Single main feed + narrow optional topic filter rail on wide screens.
- Filters collapse to popover under 1100px.
- Thread rows use title, preview, topic/week, response state, latest activity.
- Verified state is green text/icon, not green card fill.
- Pinned thread uses subtle icon + ordering, not a special giant card.

Create thread flow:
- Inline composer or dedicated simple page.
- PII firewall happens on submit.
- If personal content detected, use a protected dialog with exactly two primary paths:
  - `Chuyển sang chat riêng`
  - `Ẩn thông tin rồi đăng`

---

## 14.4 `/threads/[id]` — Thread detail

Primary job: understand one discussion and its verified answer.

- Thread content first.
- Verified teacher answer receives a 1px green rule + verification label.
- AI draft awaiting review is visually quieter and marked `Chờ xác nhận`.
- Teacher review actions live directly beside the draft: `Xác nhận`, `Chỉnh sửa`, overflow `Loại`.
- Citations expand inline.

---

## 14.5 `/inbox` — Hộp thư hỗ trợ

Primary job: process escalation tickets quickly.

Use split view on desktop:
- left 380px ticket list,
- right flexible conversation detail.

List filters: Open / Claimed / Answered / All.
Do not display every ticket property in the row.
Row essentials: student alias/name allowed for teacher, short question, age, state, urgency reason.

Detail:
- original question and AI context,
- relevant citations/tool data,
- reply composer,
- optional `Lưu thành tri thức` checkbox/action after answer.

---

## 14.6 `/students` — Sinh viên

Primary job: find a student and notice who needs attention.

First viewport:
- Search.
- Compact filter chips: `Cần chú ý`, `Vắng nhiều`, `Điểm giảm`, `Ít hoạt động`.
- Table: Student / Attendance / Current grade / Participation / Learning activity / Risk.

No row-level action buttons by default. Click row to open profile.

---

## 14.7 `/students/[id]` — Hồ sơ 360

Primary job: understand one student before making a teaching decision.

Header:
- name, student ID, class context,
- risk sentence in plain language if relevant,
- `Thêm ghi chú` secondary action.

Tabs:
- Tổng quan
- Chuyên cần
- Điểm
- Hoạt động học
- Ghi chú

Overview uses a narrative summary first, metrics second.
Charts only when trend/comparison needs visual form.
Do not put six KPI cards at top.

Observation is private and clearly marked `Chỉ giảng viên/TA thấy`.

---

## 14.8 `/attendance` — Điểm danh

Primary job: mark 30 students in < 60 seconds.

This is a dense keyboard-first working surface.

Top:
- session/date selector,
- save state,
- `Lưu điểm danh` primary action.

Table:
- Student
- Có mặt
- Muộn
- Vắng phép
- Vắng
- Participation quick add

Use radio-like keyboard cells with visible focus.
Sticky student/name column and header on desktop.
No cards.

If grade scheme missing/unconfirmed, show one slim notice above table; do not block attendance.

---

## 14.9 `/me` — Kết quả của tôi

Primary job: understand personal standing and what affects it.

First viewport:
1. Current course grade sentence / status.
2. `Giải trình điểm` breakdown as a linear calculation.
3. Attendance + participation summary.
4. Upcoming graded work.
5. Weekly learning time as a small trend lower on page.

What-if action:
- Inline question/control: `Nếu cuối kỳ được [ 8.0 ], điểm học phần sẽ là…`
- Also support natural-language chat handoff.

Do not gamify with streaks or ranks.

---

## 14.10 `/gradebook` — Sổ điểm

Primary job: inspect and prepare final grades safely.

Header actions:
- `Xem công thức` secondary.
- `Chốt điểm` primary only when allowed.

If scheme not confirmed:
- persistent slim warning at top,
- `Chốt điểm` disabled with reason visible on focus/hover.

Table:
- Student, components, bonuses/penalties, current total, final exam, final grade, status.
- Freeze name and key total columns where useful.
- Editing is inline.
- Audit/history entry appears from row context, not always-visible button.

---

## 14.11 `/gradebook/scheme` — Công thức điểm

Primary job: turn extracted policy into an explicit confirmed rule.

Layout is a **review document**, not a form wall.

- Left/main: extracted structured formula, each rule in readable sections.
- Right/context panel: source citation/page preview on wide screens; drawer on smaller screens.
- Ambiguities appear inline exactly where the missing rule belongs.
- `Xác nhận công thức` fixed/sticky only after all blockers resolved.
- New policy version shows a focused diff view.

---

## 14.12 `/grading` — Hàng chờ chấm bài

Primary job: find submissions needing human review.

Rows:
- Student
- Assignment
- Source
- AI score / deterministic score
- review state
- flag reason
- submitted time

Filters default to `Cần xem kỹ` and `Chưa duyệt` when outstanding work exists.
Bulk publish only appears after valid selection.

---

## 14.13 `/grading/[submissionId]` — Duyệt bài

Primary job: compare student work with rubric evidence and make a final grading decision.

Desktop layout:
- 52–58% left: submission document.
- 42–48% right: rubric criteria + AI evidence + score controls.

Right panel structure:
1. Overall status + total score.
2. Each criterion in a flat section separated by rules.
3. Student excerpt evidence quoted under criterion.
4. Editable score.
5. AI comment, editable.

Flag discrepancy >1 point as an inline amber notice.
Primary action at bottom: `Duyệt bài`.
Publish remains a separate deliberate action.

---

## 14.14 `/documents` — Tài liệu giảng viên

Primary job: manage source material and AI visibility.

Table/list:
- name,
- type,
- week/topic,
- use for AI,
- visible to students,
- processing status,
- updated date.

Upload opens an inline dropzone at top or a side panel, not a modal unless file metadata requires protected focus.

`ANSWER_KEY` clearly communicates:
- `Không hiển thị cho sinh viên`
- `Không dùng cho AI của sinh viên`

---

## 14.15 `/library` — Thư viện

Primary job: find a learning resource and use it.

Student view:
- Search first.
- Filters: week/topic/type.
- Prefer a clean list with file-type mark and useful metadata.
- PDF preview is detail view.
- Actions: `Xem`, `Hỏi AI về tài liệu`, and for exam paper `Luyện đề này`.

Avoid marketplace-style cards.

---

## 14.16 `/calendar` — Lịch

Primary job: see what happens next.

Default to week/list based on viewport and user history.
- Month view available, not necessarily default.
- Events use restrained semantic accents; not rainbow category blocks.
- Upcoming list on mobile.
- `Thêm vào lịch` / ICS is secondary utility.

---

## 14.17 `/questions` — Ngân hàng câu hỏi

Primary job: review and curate exam/practice questions.

Use table/list + review drawer.
Filters:
- state,
- topic,
- difficulty,
- type,
- source.

Question content opens in a wide drawer or detail page.
Review actions: `Duyệt`, `Chỉnh sửa`, overflow `Loại`.

Do not show AI generation controls by default; place under `Tạo câu hỏi` flow.

---

## 14.18 `/practice` — Luyện đề

Primary job: choose a useful practice mode.

First viewport:
- one recommended practice block based on recent mistakes,
- `Theo chủ đề` and `Thi thử` as two clear mode choices below,
- recent attempt continuation.

No game dashboard.
No streak/leaderboard.

---

## 14.19 `/practice/[attemptId]` — Làm bài

Primary job: answer without distraction.

- Minimal header: progress, time only in mock exam, exit.
- One question at a time by default for small screens; desktop can show question navigator.
- Immediate feedback only in topic practice mode.
- Explanation appears after answer, with citation.
- Essay answer uses calm writing area with character/word helper only if necessary.

---

## 14.20 `/practice/history` — Lịch sử luyện tập

Primary job: identify what to practice next.

Use chronological rows + topic weakness summary.
No achievement badges.

---

## 14.21 `/insights` — Lỗ hổng kiến thức

Primary job: decide what teaching material should change.

First viewport:
- report date + `Tạo báo cáo mới` when new data exists,
- ranked topics as an editorial list, not score cards.

Each topic:
- clear title,
- why it ranked high,
- key signals,
- 3–5 anonymized example questions in collapsible section,
- suggested teaching action.

Actions: `Tạo thread ghim`, `Tạo buổi ôn tập`.
Comparison with previous snapshot appears later.

---

## 14.22 `/observability` — Quan sát AI

Primary job: diagnose system behavior.

This surface can be denser than teacher/student pages.

Top is a compact status strip, not metric cards:
- Request volume
- p95 latency
- error rate
- fallback rate
- daily cost

Below:
- time series only when trend matters,
- request table,
- detail drawer with masked prompt, tools, confidence, trace link.

PII events have explicit privacy-safe presentation.

---

## 14.23 `/settings/llm` — Cấu hình LLM

Primary job: assign provider/model safely by task.

Structure:
1. Provider connections.
2. Task routing table.
3. Fallback chain.
4. Embedding config, visually separated because changing it requires re-index.

API keys are write-only; after save show only masked/connected state.
`Test kết nối` is local secondary action on provider row.

Avoid exposing advanced fields until `Cài đặt nâng cao` expands.

---

## 14.24 `/settings/integrations` — Tích hợp

Primary job: connect and test Mail / IMAP / Teams.

Use one flat section per integration separated by large spacing and rules.
Do not create three giant cards.

Show:
- state,
- required fields,
- test action,
- last successful check.

Teams explains admin-consent dependency in plain language.

---

## 14.25 `/analytics` — Analytics lớp học

Primary job: understand class-level patterns, not monitor students obsessively.

Sections:
- Learning activity
- Escalation / support
- PII protection
- Grade/review quality
- Cost only for authorized roles

Use charts only for comparisons/trends. If the answer is a number, show a number with context instead of a chart.

---

# 15. State design

Every interactive component and route must support:
- default,
- hover,
- focus,
- active/selected,
- disabled,
- loading,
- empty,
- error.

### Empty states

Teach the next action.

Bad:
> Không có dữ liệu.

Good:
> Chưa có câu hỏi nào cần bạn xử lý. Khi AI không đủ chắc chắn hoặc sinh viên yêu cầu hỗ trợ, câu hỏi sẽ xuất hiện ở đây.

### Loading

- Skeleton mirrors final content.
- Keep shell/navigation stable.
- Never block whole screen because one panel is loading.

### Errors

Problem + recovery:
> Không tải được bài nộp từ IMAP. Kết nối vẫn được giữ. **Thử lại** hoặc **kiểm tra cấu hình**.

---

# 16. Accessibility

Minimum:
- WCAG AA text contrast.
- Visible focus ring: 2px red/ink combination with offset.
- 44x44px effective touch target for touch UI.
- Do not encode status by color alone.
- `aria-live` for chat streaming status and notification arrival.
- Tables use proper headers, captions where useful, and keyboard-safe editing.
- Dialog focus trap + restore focus.
- Respect `prefers-reduced-motion`.
- Browser zoom to 200% must remain operable.
- Vietnamese diacritics must never clip due to line-height.

---

# 17. Responsive behavior

### Breakpoints

Suggested structural breakpoints, not typography breakpoints:
- `< 720px`: mobile.
- `720–1099px`: compact/tablet.
- `>= 1100px`: desktop.
- `>= 1440px`: wide task surfaces may use context panels.

### Rules

- Collapse navigation before shrinking content typography.
- Split views become list → detail navigation on mobile.
- Side panels become full-screen sheets/pages.
- Dense gradebook/attendance may use deliberate horizontal scrolling with frozen first column.
- Chat composer remains reachable above mobile keyboard.

---

# 18. Design tokens contract

Coding Agent must create semantic tokens, not scatter literal values.

Minimum token categories:

```text
color.background.*
color.text.*
color.action.*
color.state.*
color.rule.*
space.*
radius.*
type.*
motion.*
size.sidebar.*
size.header.*
z.*
```

Page CSS/components may not introduce a one-off color, radius, or arbitrary shadow without a documented reason.

---

# 19. Component architecture recommendation

```text
AppShell
├── Sidebar
├── TopBar
├── NotificationPopover
├── CommandPalette
└── RouteOutlet

Page primitives
├── PageHeader
├── Section
├── ActionList
├── Toolbar
├── DataTable
├── Tabs
├── SegmentedControl
├── InlineNotice
├── StatusText
├── EmptyState
├── Skeleton
├── Drawer
├── Dialog
└── Composer

Domain components
├── PIIProtectionNotice
├── CitationList
├── VerificationState
├── EscalationRow
├── StudentRiskSummary
├── AttendanceGrid
├── GradeCalculation
├── GradeSchemeReview
├── SubmissionReview
├── QuestionReview
├── KnowledgeGapTopic
├── LLMRouteTable
└── RequestTraceDetail
```

Rule: domain components compose primitives; they do not reinvent buttons, spacing, dialogs, or typography.

---

# 20. Implementation order for Coding Agent

1. Build tokens and base typography.
2. Build AppShell + responsive navigation.
3. Build primitives and all states.
4. Build role-specific “Hôm nay” screens.
5. Build highest-frequency work surfaces:
   - Chat
   - Inbox
   - Attendance
   - Gradebook
   - Grading detail
6. Build remaining routes.
7. Add responsive transformations.
8. Add keyboard/accessibility states.
9. Add motion last.
10. Perform one complete visual consistency pass and one responsive/a11y pass.

Do not polish each route into a separate design language. The system wins through consistency.

---

# 21. Agent anti-pattern detector checklist

Reject or refactor when any of these appears without a task-specific reason:

- 3+ same-size KPI cards at page top.
- A card containing multiple smaller cards.
- Every section wrapped in a bordered rounded rectangle.
- More than one primary filled-red button in the same working region.
- Icon-only desktop primary navigation.
- A modal for a simple edit form.
- Red used as decorative background on large areas.
- More than two visible secondary actions beside primary action.
- Random font sizes outside the role scale.
- Generic `gray-*` colors instead of semantic tinted neutrals.
- Giant empty hero/title region in an app screen.
- Loading spinner replacing the entire route.
- Technical AI terms shown to students when plain language is possible.
- Progress rings, gamified streaks, confetti, or leaderboard patterns in normal learning flows.

---

# 22. Definition of done

A route is not done because it “looks clean.” It is done when:

1. A role-appropriate user can identify the primary task within ~3 seconds.
2. The first viewport has one obvious action path.
3. Secondary complexity is available but not competing.
4. All component states exist.
5. Keyboard/focus behavior is complete.
6. Narrow and wide layouts are structurally sound.
7. Long Vietnamese text does not break layout.
8. The screen uses the same tokens/primitives as the rest of EduPilot.
9. There is no unnecessary card/container/badge.
10. Red has a semantic reason everywhere it appears.


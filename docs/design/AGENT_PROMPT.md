# Coding Agent Prompt — Rebuild EduPilot UI

You are rebuilding the frontend UI/UX of **EduPilot v2**, an AI-assisted classroom operating system for Vietnamese university courses.

## Inputs you must read before editing

1. `PRD.md` — product truth, roles, modules, acceptance criteria.
2. `ARCHITECTURE.md` — routes and technical constraints.
3. `DESIGN.md` — the authoritative visual and interaction system.
4. Existing frontend code — preserve working behavior/data contracts unless the PRD requires a change.

If implementation conflicts with `DESIGN.md`, preserve product behavior and refactor the visual implementation to the design system. Do not invent new product capabilities.

---

## Design objective

Rebuild the product around the direction **Red Thread / Academic Instrument**.

The interface must feel:
- modern,
- minimal without hiding necessary functionality,
- extremely legible in Vietnamese,
- calm for students,
- efficient for teachers,
- precise and trustworthy around grades/privacy/AI review.

This is an **Operate** product UI. Task completion, scanability, consistency, keyboard behavior, responsive structure, and safety outrank decorative expression.

---

## Non-negotiable visual rules

- Red + white is the brand, but red is a signal, not a wallpaper.
- No generic SaaS metric-card dashboard.
- No nested cards.
- Prefer whitespace, proximity, typography and 1px rules over containers.
- Small radii (mostly 6–12px), almost no shadows.
- One font family: Be Vietnam Pro with Vietnamese-safe fallbacks.
- One primary action per working region.
- Standard familiar controls; do not invent strange affordances for visual novelty.
- Primary desktop navigation has text labels.
- Students never see unnecessary AI infrastructure jargon.
- Use inline editing/progressive disclosure before modals.
- Build semantic tokens first; no page-specific random colors/radii/shadows.

---

## Required global components

Implement/reuse these consistently:

`AppShell`, `Sidebar`, `TopBar`, `PageHeader`, `Section`, `ActionList`, `Toolbar`, `DataTable`, `Tabs`, `SegmentedControl`, `InlineNotice`, `StatusText`, `EmptyState`, `Skeleton`, `Drawer`, `Dialog`, `Composer`.

Domain components:

`PIIProtectionNotice`, `CitationList`, `VerificationState`, `EscalationRow`, `StudentRiskSummary`, `AttendanceGrid`, `GradeCalculation`, `GradeSchemeReview`, `SubmissionReview`, `QuestionReview`, `KnowledgeGapTopic`, `LLMRouteTable`, `RequestTraceDetail`.

Do not create route-specific clones of buttons, fields, status chips, dialogs, tabs, tables, or notices.

---

## Required routes

Rebuild all existing routes defined by the architecture:

- `/chat`
- `/threads`
- `/threads/[id]`
- `/inbox`
- `/students`
- `/students/[id]`
- `/attendance`
- `/me`
- `/gradebook`
- `/gradebook/scheme`
- `/grading`
- `/grading/[submissionId]`
- `/documents`
- `/library`
- `/calendar`
- `/questions`
- `/practice`
- `/practice/[attemptId]`
- `/practice/history`
- `/insights`
- `/observability`
- `/settings/llm`
- `/settings/integrations`
- `/analytics`

Also create role-aware `/` (“Hôm nay”) entry surfaces for STUDENT and TEACHER/TA.

Follow the route contracts in `DESIGN.md` exactly unless existing product behavior requires a documented adaptation.

---

## Implementation sequence

### Phase 1 — foundation
- Introduce semantic design tokens.
- Install/configure Be Vietnam Pro if project constraints allow; otherwise use documented fallback without blocking rendering.
- Normalize typography, focus, selection, scrollbar, inputs and buttons.
- Build app shell and role-aware navigation.

### Phase 2 — primitives
Build all primitives with states:
- default
- hover
- focus
- active/selected
- disabled
- loading
- empty
- error

### Phase 3 — core routes
Prioritize:
1. Student Today
2. Teacher Today
3. Chat
4. Inbox
5. Attendance
6. Gradebook
7. Grading detail

### Phase 4 — remaining routes
Complete all routes above using the same primitives.

### Phase 5 — responsive + accessibility
- mobile <720
- tablet 720–1099
- desktop >=1100
- wide >=1440

Check keyboard navigation, 200% zoom, long Vietnamese copy, empty/error/loading states, and reduced motion.

### Phase 6 — motion
Add only purposeful state motion.
Implement the optional **Red Thread Transition** for recommended-task navigation if it remains smooth and accessible.

---

## Copy rules

Use Vietnamese plain language.

Prefer:
- `Cần giảng viên hỗ trợ`
- `Đã ẩn thông tin cá nhân`
- `Độ tin cậy`
- `Nguồn dùng để trả lời`
- `Loại khỏi tri thức`

Avoid exposing to students:
- PII event
- redaction
- RAG
- fallback
- trace
- provider routing

Buttons are verbs: `Duyệt bài`, `Gửi trả lời`, `Xác nhận công thức`, `Bắt đầu ôn`.

---

## Visual QA before finishing

Perform exactly two bounded review passes:

### Pass A — desktop + mobile visual review
Check representative routes:
- `/`
- `/chat`
- `/attendance`
- `/gradebook`
- `/grading/[submissionId]`
- `/insights`
- `/settings/llm`

Fix in one batch:
- hierarchy,
- clutter,
- spacing rhythm,
- unnecessary cards/borders,
- route inconsistencies,
- overflow,
- long Vietnamese copy,
- visible focus,
- responsive structure.

### Pass B — confirmation
Re-run screenshots/tests only once to confirm the batch fixes.
Do not enter an endless polish loop.

---

## Final acceptance questions

Before declaring completion, answer yes with evidence from the implementation:

- Can a student see the next learning action immediately?
- Can a teacher see the next human decision immediately?
- Can attendance be operated quickly with keyboard?
- Can grading be reviewed side-by-side without context switching?
- Is the grade scheme understandable as a review document rather than a form wall?
- Is PII/privacy language understandable without technical jargon?
- Are all routes visually one product?
- Are there zero nested cards?
- Does every red element have a semantic reason?
- Are every interactive primitive’s focus/loading/error/disabled states implemented?

If any answer is no, continue fixing before completion.

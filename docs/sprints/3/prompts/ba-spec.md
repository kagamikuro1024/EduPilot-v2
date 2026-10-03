# Prompt cho `ba` — Spec sprint 3 (PU + P1)

**Làm trong worktree `/Users/kuro/Documents/TA_Agent_v2-s3` (nhánh `sprint/3-pu-p1`, xếp chồng trên sprint 2 + 1.5)**; git dùng `git -C /Users/kuro/Documents/TA_Agent_v2-s3`.

## Đọc
`docs/team/CONTEXT.md`, `docs/team/BA.md`, `docs/team/TEMPLATES.md`, **`docs/sprints/3/plan.md`** (story + quyết định PM: D53 CSS Modules, PU L4 thu hẹp, token Admin dev, `make eval` hoãn), `docs/phases/PU.md`, `docs/phases/P1.md`, `docs/design/DESIGN.md` (§1–§2, §10, §13–§15, §14.23, §21–§22), `docs/design/INTEGRATION.md`, `docs/UX.md` (mục 3, 4, 6), `docs/ARCHITECTURE.md` (§2–§5, bảng `llm_*`, §8 env), `docs/SYSTEM_DESIGN.md` §3.1 (Scheduler), `docs/DECISIONS.md` D46, D47, D51–D53. Nền đã có: spec `docs/specs/FEAT-pg-foundation/` (định dạng lỗi, cursor, Idempotency-Key, SSE, JWT/RBAC), mã `frontend/src/shared/` của prototype, `backend-go/`.

## Việc
1. `docs/specs/FEAT-ui-foundation/{US,SRS,QUESTIONS}.md` — US-PU-01…05 đúng bảng plan. AC đo được: token/lint (lệnh, số vi phạm = 0), mỗi primitive × 8 trạng thái trên `/dev/ui` ở 4 bề rộng, `apiClient` ánh xạ đúng `{code,message,details,retry_after}` của PG, `useSSE` nối lại bằng `Last-Event-ID`, ngân sách Lighthouse theo `UX.md` mục 3, axe 0 lỗi nghiêm trọng, 7 route ảnh mốc. AC "không vỡ màn mock": `audit.mjs` của QC 1.5 vẫn 0 lỗi sau PU.
2. `docs/specs/FEAT-llm-gateway/{US,SRS,QUESTIONS}.md` — US-P1-01…05. SRS: schema `00002` 5 bảng đủ cột/kiểu/index (theo ARCHITECTURE, `course_id` đầu index), API cấu hình theo `ARCHITECTURE.md` §5 (đường dẫn, quyền ADMIN, TEACHER chỉ xem), mã lỗi, khoá Redis của Scheduler (tiền tố, TTL), bảng `task → làn`, ngưỡng cầu dao, trần ngân sách 80/100%, hành vi suy giảm. Nhánh lỗi bắt buộc: key sai → Test báo lỗi rõ và không lưu; key không bao giờ trả về/log; hàng INTERACTIVE đầy → 503 `OVERLOADED` + `retry_after`; mọi provider chết → `degraded=true`; client huỷ → huỷ lời gọi provider; BATCH không làm INTERACTIVE chờ quá ngưỡng (số cụ thể theo SLO); TEACHER/TA/STUDENT gọi API ghi → 403.
3. `/settings/llm` thật (US-P1-05): theo DESIGN §14.23; đối chiếu màn mock 1.5 (`docs/sprints/1.5/spec/`) — chỗ nào mock lệch hợp đồng thật thì spec thật thắng (D51), liệt kê khác biệt.
4. Câu hỏi mở → `QUESTIONS.md`, mỗi câu có mặc định an toàn (chủ dự án vắng mặt). Câu cần secret thật (key provider) ghi rõ là việc của chủ dự án.

Không mở subagent. Chỉ sửa `docs/specs/FEAT-ui-foundation/**`, `docs/specs/FEAT-llm-gateway/**`. Commit `sprint 3: spec FEAT-ui-foundation + FEAT-llm-gateway`, push. Trả lời số AC từng story + câu hỏi mở, rồi dừng. Làm thật kỹ.

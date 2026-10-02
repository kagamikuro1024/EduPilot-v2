# Prompt cho `ba` — Spec sprint 4 (P2 Lớp học)

**Worktree `/Users/kuro/Documents/TA_Agent_v2-s4`, nhánh `sprint/4-p2`** (xếp chồng trên sprint 3). Git: `git -C /Users/kuro/Documents/TA_Agent_v2-s4`.

## Đọc
`docs/team/CONTEXT.md`, `docs/team/BA.md`, `docs/team/TEMPLATES.md`, **`docs/sprints/4/plan.md`** (12 story, quyết định PM), `docs/phases/P2.md`, `docs/FLOWS.md` F1 + F2 (đủ nhánh lỗi) + F14 (bảng Provider "Hôm nay"), `docs/PRD.md` M0, M14, §3 ma trận quyền, `docs/ARCHITECTURE.md` §4 (00003, 00004), §5 (Tài khoản, Quản trị lớp, Lớp, Mã tham gia, Thành viên, Tham gia lớp, Hôm nay), §8, §9 seed, `docs/design/DESIGN.md` §14.1 và các route `/login`…`/class/*`, `docs/design/INTEGRATION.md` mục 2, 4, `docs/UX.md`, `docs/PRODUCTION_READINESS.md` (phần tài khoản, dữ liệu cá nhân), `AGENTS.md` mục "Cấm tuyệt đối" (MSSV tự khai, token băm, không localStorage). Nền đã có: spec `FEAT-pg-foundation` (lỗi, cursor, Idempotency, outbox, JWT), `FEAT-ui-foundation` (`apiClient`, primitive), `FEAT-llm-gateway`; màn mock 1.5 (`docs/sprints/1.5/spec/`) để đối chiếu lời văn và bố cục.

## Việc
1. `docs/specs/FEAT-account-security/{US,SRS,QUESTIONS}.md` — US-P2-01…06.
2. `docs/specs/FEAT-course-foundation/{US,SRS,QUESTIONS}.md` — US-P2-07…12.
3. Yêu cầu bắt buộc:
   - Mỗi story có AC nhánh lỗi + AC phân quyền thật (vai không được phép → 403 / không thấy).
   - AC tấn công: mạo danh MSSV, dùng lại refresh token, đoán mã tham gia (rate limit + lỗi đồng nhất), link mời / token đặt lại dùng hai lần hoặc hết hạn, token lưu dạng băm (kiểm DB), không còn token trong localStorage.
   - SRS: DDL `00003`, `00004` đủ cột / kiểu / CHECK / index (`course_id` đầu index), API theo §5 với mã lỗi, cookie (tên, cờ, path, hạn), khoá Redis (rate limit, cache today), mẫu mail (tiêu đề, thân tiếng Việt), bảng Provider "Hôm nay" + luật xếp hạng, chuỗi tiếng Việt chính.
   - Chỗ màn mock 1.5 lệch hợp đồng thật: spec thật thắng (D51), liệt kê khác biệt.
   - Chốt cách prototype mock và đăng nhập thật cùng tồn tại trong sprint này (plan: "đổi vai bằng đăng nhập thật").
4. Câu hỏi mở → `QUESTIONS.md`, mỗi câu kèm **mặc định an toàn**; đánh dấu **[CHỦ DỰ ÁN]** cho câu đụng quyền / dữ liệu cá nhân / chính sách (PM sẽ chốt mặc định và báo chủ dự án).

Không mở subagent. Chỉ sửa hai thư mục spec trên. Commit `sprint 4: spec FEAT-account-security + FEAT-course-foundation`, push. Trả lời số AC từng story + câu hỏi mở (số câu [CHỦ DỰ ÁN]), rồi dừng. Làm thật kỹ.

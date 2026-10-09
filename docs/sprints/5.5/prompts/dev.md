# Prompt cho `dev` — Thi công sprint 5.5 (UI panel nổi)

**Worktree `/Users/kuro/Documents/TA_Agent_v2-s55`, nhánh `sprint/5.5-ui-panels`** (xếp chồng trên `sprint/5-pe`). Lỗi cổng PE của sprint 5 vẫn ưu tiên hơn; sửa ở worktree s5. Sau mỗi lần sửa ở s5: `git merge origin/sprint/5-pe` vào 5.5.

## Đọc
- `docs/team/DEV.md`.
- `docs/sprints/5.5/plan.md`.
- Spec **APPROVED** `docs/specs/FEAT-ui-panels/`: US, SRS, QUESTIONS, và `TL-REVIEW.md` mục "Quyết định PM".
- `docs/design/DESIGN.md`, `docs/UX.md`.
- Dùng skill `impeccable` (shape / critique / polish) cho phần thiết kế.

## Thứ tự
1. **US-UI-01**
   - Dựng 3 phương án (a) / (b) / (c) trên trang mẫu theo AC.
   - Chụp 18 ảnh (1440 / 1024 / 375 px × GV / SV × 3 phương án) vào `docs/sprints/5.5/shots/ui01/`.
   - Commit, ghi handoff `dev-US-UI-01.md` (bảng tương phản từng phương án).
   - **DỪNG và báo PM**: PM chọn phương án rồi mới ghi D59 / sửa `DESIGN.md` / `AGENTS.md` / `ui-antipatterns.sh`.
2. Sau khi PM chọn: US-UI-02 → 03 → 04 → 05 → 06 → 07.
   - Mỗi story: ảnh mốc sinh lại ngay trong story (TLR-3), CI xanh, handoff liệt kê ảnh trước / sau.

## Luật
- Chỉ token `--ep-*` và primitive `shared/`.
- Không panel lồng panel; không tường thẻ KPI; đỏ chỉ là tín hiệu.
- Bóng tĩnh, không `filter` / `backdrop-filter`.
- `Panel` là Server Component.
- RAM có hạn: một stack, Playwright ≤ 2 worker, `docker volume prune -f` sau mỗi story.
- Phân vân kỹ thuật hỏi Tech Lead; spec sai thì ghi `docs/sprints/5.5/proposals.md`.

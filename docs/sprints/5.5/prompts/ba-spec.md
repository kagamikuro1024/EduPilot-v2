# Prompt cho `ba` — Spec sprint 5.5 (UI panel nổi)

**Worktree `/Users/kuro/Documents/TA_Agent_v2-s55`, nhánh `sprint/5.5-ui-panels`** (xếp chồng trên `sprint/5-pe`). Repo `origin` = `github.com/kagamikuro1024/EduPilot-v2`.

## Đọc
- `docs/sprints/5.5/plan.md`: góp ý của chủ dự án, đề xuất D59 "panel có kỷ luật", 7 story UI-01…07, cổng UI.
- `docs/design/DESIGN.md`: Surface character, Radius, Color tokens, §13, §14, §21, §22.
- `docs/UX.md` mục 6.
- `AGENTS.md`, phần "Luật giao diện".
- `frontend/src/shared/styles/tokens.css`.
- Danh sách màn hiện có trong `frontend/src/app/`.
- Spec nền `docs/specs/FEAT-ui-foundation/`.

## Việc
Viết `docs/specs/FEAT-ui-panels/{US,SRS,QUESTIONS}.md`, 7 story US-UI-01…07 đúng bảng của plan. Mỗi AC phải đo được: token, CSS tính toán, axe, ảnh chụp ở 1440 / 1024 / 375 px, `ui-antipatterns.sh`, Lighthouse.

Bắt buộc có:
1. **US-UI-01 có điểm dừng chờ chủ dự án.**
   - Dev dựng 3 phương án độ nổi (a) / (b) / (c) trên `/dev/ui` (trang mẫu "Hôm nay" GV + SV).
   - Chủ dự án chọn một phương án. Chỉ sau đó mới ghi D59 và sửa `DESIGN.md`, `AGENTS.md`, `ui-antipatterns.sh`.
   - AC nêu rõ đoạn chữ nào của `DESIGN.md` / `AGENTS.md` bị thay.
2. **Token mới và primitive.**
   - Token: `--ep-canvas`, `--ep-surface-strong`, `--ep-elevation-1`, `--ep-radius-panel`.
   - Primitive `Panel` / `PanelSection`.
   - Luật chặn `Panel` lồng `Panel`.
   - Tương phản chữ phụ ≥ 4,5 : 1 trên cả panel và canvas.
3. **Danh sách màn phải đổi theo từng story** (UI-03…06), lấy từ route thật trong `frontend/src/app/`. Mỗi màn: khung nhìn đầu, vùng nào thành panel, tiêu đề vùng nằm ngoài panel.
4. **Giữ nguyên:**
   - không tường thẻ KPI;
   - đỏ chỉ là tín hiệu;
   - một hành động chính mỗi vùng;
   - 375 px cho màn sinh viên.
5. **Cổng UI:**
   - ảnh trước / sau của các màn chính;
   - ảnh mốc chỉ sinh lại khi handoff liệt kê từng ảnh (quy tắc Q-QC-PU06-4);
   - LCP / TBT không kém cuối sprint 5;
   - CI xanh.
6. **`QUESTIONS.md`:** câu nào cần chủ dự án thì đánh **[CHỦ DỰ ÁN]**, có mặc định. Câu chế độ tối mặc định là chưa làm.

Không mở subagent. Chỉ sửa `docs/specs/FEAT-ui-panels/**`. Commit `sprint 5.5: spec FEAT-ui-panels`, push, báo PM ≤ 5 dòng (số AC mỗi story, số câu [CHỦ DỰ ÁN]), dừng. Sau đó Tech Lead thẩm định (`TL-REVIEW.md`).

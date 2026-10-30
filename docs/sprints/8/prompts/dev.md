# Prompt cho `dev`: thi công sprint 8 (P6)

Nơi làm: `/Users/kuro/Documents/TA_Agent_v2-s7`: `git fetch && git switch -c sprint/8-p6 origin/sprint/8-p6` (hoặc switch nếu đã có), push lên `origin sprint/8-p6`.

## Đọc
`docs/team/DEV.md`, `docs/team/CONTEXT.md`, `docs/sprints/8/plan.md`; spec **APPROVED** `docs/specs/FEAT-gradebook/` (US, SRS, QUESTIONS cột Trả lời, `TL-REVIEW.md` "Quyết định PM"); `docs/sprints/8/proposals.md`; `AGENTS.md` (nhớ luật 5 tính điểm, luật 12 việc nặng chạy nền, luật 14 idempotent; "Cấm tuyệt đối": sửa `expected_final_grades.csv`, tự xác nhận công thức, dùng thời gian học trong `internal/grade`).

## Bắt đầu ngay: US-S8-UI (không cần spec sản phẩm)
Xoá câu mô tả dưới tiêu đề ở `/settings/llm` (tiêu đề + 4 khối), `/settings/integrations` (tiêu đề + dòng mô tả từng tích hợp), khối bảo mật `/settings` (danh sách ở `docs/sprints/7/handoff/dev-PM-42.md` mục "Ngoài phạm vi #42"). Xoá hẳn, không dời vào tooltip. Ảnh mốc liên quan cập nhật; test cũ nào khẳng định đúng câu đó → proposal và chờ ACCEPTED. Handoff `docs/sprints/8/handoff/dev-US-S8-UI.md`, commit `US-S8-UI: …`.

## Sau đó (khi PM báo spec APPROVED)
US-P6-01 → 02 → 03 → 05 → 04 → 06 → 07 → 08 → 09 → US-S8-SEED, không chờ PM giữa các story. Mỗi story: test Go + frontend xanh, **CI xanh trước khi sang story kế**, handoff `docs/sprints/8/handoff/dev-<story>.md`, commit `<story>: …`, push.

## Luật riêng sprint này
- `internal/grade`: chỉ `shopspring/decimal`, không `float64`; không import LLM, `activity`, thời gian học. Test 30/30 theo `seed/expected_final_grades.csv`; lệch → sửa mã, hoặc proposal nếu tin đáp án sai (không bao giờ tự sửa CSV).
- Trích công thức là việc nền (202 + tiến độ SSE), qua `llm.Structured`; provider giả có bản ghi cho quy chế seed.
- Test cũ chỉ sửa khi proposal ACCEPTED (2 lần vi phạm ở sprint 7 — không lặp lại).
- Chỉ một stack chạy; Playwright ≤ 2 worker; `docker volume prune -f` sau mỗi story; dùng `scripts/go-test-all.sh` cho lượt Go đầy đủ.

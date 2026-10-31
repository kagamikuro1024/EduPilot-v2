# Prompt cho `dev`: thi công sprint 9 (P7)

Khi sprint 8 xong (mọi story có handoff, CI xanh): trong worktree dev `git fetch && git switch -c sprint/9-p7 origin/sprint/9-p7 && git merge --no-edit origin/sprint/8-p6`, push `sprint/9-p7`. Từ đó làm trên `sprint/9-p7`; sửa vòng sau của sprint 8 (nếu QC FAIL) làm trên `sprint/8-p6` rồi gộp lại vào `sprint/9-p7`.

## Đọc
`docs/sprints/9/plan.md`; spec **APPROVED** `docs/specs/FEAT-assignments/` (US, SRS, QUESTIONS, `TL-REVIEW.md` "Quyết định PM", `fixtures/`); `docs/sprints/9/proposals.md`, `techlead.md`; `docs/phases/P7.md`; `AGENTS.md` (luật 2, 3, 4, 11, 12, 14; "Cấm tuyệt đối": tự công bố điểm AI chấm, sinh viên thấy nháp).

## Thứ tự
US-P7-01 → 06 → 02 → 03 → 04 → 07 → 08 → 09 → 10 → 05 → 11 → US-S9-SEED, không chờ PM giữa các story. Mỗi story: test Go (kèm `-tags testroutes` như CI) + frontend xanh, **CI xanh trước khi sang story kế**, handoff `docs/sprints/9/handoff/dev-<story>.md`, commit `<story>: …`, push.

## Luật riêng
- Hàng đợi theo mẫu `internal/judge/queue.go` (TLR-18), không viết biến thể mới.
- Che / khôi phục danh tính để gateway làm (TLR-6); `internal/grading` chỉ xoá dòng tiêu đề + gắn cờ injection.
- Điểm là chuỗi → `decimal.NewFromString`, không `float64` (TLR-12).
- `go-imap/v2` ghim `v2.0.0-beta.8` + `go-message v0.18.2` (TLR-19); test bằng `imapserver` + `imapmemserver`.
- `mock-graph` chỉ ở compose local; adapter không có nhánh `if mock`.
- Test cũ đỏ → proposal + báo PM, chờ; số đếm thao tác OpenAPI cập nhật theo route mới (như #12 sprint 8), ghi ở handoff.
- Một stack chạy; Playwright ≤ 2 worker; `docker volume prune -f` sau mỗi story.

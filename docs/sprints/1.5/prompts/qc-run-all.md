# Prompt cho `qc` — chạy toàn bộ TC sprint 1.5

Dev đã xong toàn bộ mã nguồn và có đủ handoff trong `docs/sprints/1.5/handoff/`:
- `dev-US-PROTO-00.md` (đã sửa BUG-1, 2, 3 ở commit `875f7b6`)
- `dev-US-PROTO-01.md` (commit `0c45f15`)
- `dev-US-PROTO-02.md` (commit `990c243`)
- `dev-US-PROTO-03.md` (commit `87f2cf6`)
- `dev-US-PROTO-04.md` (commit `1915734`)

## Việc
1. Chấm lại US-PROTO-00 (kiểm tra BUG-1, 2, 3 đã khắc phục chưa) → cập nhật `docs/sprints/1.5/qc/report-US-PROTO-00.md`.
2. Chạy lần lượt `tc-US-PROTO-01.md`, `tc-US-PROTO-02.md`, `tc-US-PROTO-03.md`, `tc-US-PROTO-04.md` → sinh các báo cáo:
   - `docs/sprints/1.5/qc/report-US-PROTO-01.md`
   - `docs/sprints/1.5/qc/report-US-PROTO-02.md`
   - `docs/sprints/1.5/qc/report-US-PROTO-03.md`
   - `docs/sprints/1.5/qc/report-US-PROTO-04.md`
3. Chạy `tc-US-PROTO-DEMO.md`: đi trọn kịch bản demo 15 phút (F2→F3→F5→F7→F9→F10→F17) trên prototype, kiểm tra tính toán điểm (8,3 → 8,5 → 8,7 theo proposals #13, What-if 8,3) → `docs/sprints/1.5/qc/report-US-PROTO-DEMO.md`.
4. Chạy kiểm tra tĩnh: `pnpm -C frontend lint`, `pnpm -C frontend build`, `bash scripts/ui-antipatterns.sh`.
5. Đưa ra kết luận tổng quan PASS/FAIL cho sprint 1.5.

Mỗi shell: `source ~/.zprofile`.
Commit chỉ đường dẫn `docs/sprints/1.5/qc/**`.
Trả lời tóm tắt kết quả từng story và kết luận cuối cùng.

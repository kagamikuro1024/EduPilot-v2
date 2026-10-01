# Prompt cho `qc` — Chạy nghiệm thu toàn bộ sprint 1.5 theo spec v4

Dev đã hoàn thành thi công theo spec v4 và đã cập nhật đầy đủ handoff:
- `docs/sprints/1.5/handoff/dev-US-PROTO-00.md` (sửa BUG-1, 2, 3)
- `docs/sprints/1.5/handoff/dev-US-PROTO-01.md` (cập nhật form tạo thread, reply composer, citations)
- `docs/sprints/1.5/handoff/dev-US-PROTO-02.md` (cập nhật phân tách 2 panel /inbox, quyền kiểm duyệt Threads GV/TA)
- `docs/sprints/1.5/handoff/dev-US-PROTO-03.md` (cập nhật phân tách 2 panel /grading/[submissionId])
- `docs/sprints/1.5/handoff/dev-US-PROTO-04.md` (cập nhật phân tách panel /settings, /insights)

## Việc
1. Chạy nghiệm thu toàn bộ các bộ test case:
   - `docs/sprints/1.5/qc/tc-US-PROTO-00.md` -> cập nhật `docs/sprints/1.5/qc/report-US-PROTO-00.md`
   - `docs/sprints/1.5/qc/tc-US-PROTO-01.md` (bao gồm 01-AC4 form tạo thread, 01-AC10 reply composer & hỏi AI) -> `docs/sprints/1.5/qc/report-US-PROTO-01.md`
   - `docs/sprints/1.5/qc/tc-US-PROTO-02.md` (bao gồm 02-AC5 phân tách panel /inbox, 02-AC8 kiểm duyệt Threads) -> `docs/sprints/1.5/qc/report-US-PROTO-02.md`
   - `docs/sprints/1.5/qc/tc-US-PROTO-03.md` (bao gồm 03-AC1 phân tách panel /grading) -> `docs/sprints/1.5/qc/report-US-PROTO-03.md`
   - `docs/sprints/1.5/qc/tc-US-PROTO-04.md` -> `docs/sprints/1.5/qc/report-US-PROTO-04.md`
2. Chạy `docs/sprints/1.5/qc/tc-US-PROTO-DEMO.md`: đi trọn kịch bản demo 15 phút (F2->F3->F5->F7->F9->F10->F17) trên prototype, xác nhận số liệu điểm (8,3 -> 8,5 -> 8,7 theo proposal #13, What-if 8,3) -> `docs/sprints/1.5/qc/report-US-PROTO-DEMO.md`.
3. Kiểm tra tĩnh: `pnpm -C frontend lint`, `pnpm -C frontend build`, `bash scripts/ui-antipatterns.sh`.
4. Tổng hợp kết luận từng story PASS/FAIL và kết luận chung sprint 1.5.

Mỗi shell: `source ~/.zprofile`.
Commit chỉ `docs/sprints/1.5/qc/**`.
Trả lời tóm tắt kết quả và dừng.

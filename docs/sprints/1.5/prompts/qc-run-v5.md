# Prompt cho `qc` — Chạy nghiệm thu sprint 1.5 vòng 2 (spec v5 + v5.1 + #20–#23)

Dev đã bàn giao (HEAD `78510e1`, handoff `docs/sprints/1.5/handoff/dev-US-PROTO-0{0..4}.md`, ảnh `shots/v5/`). PM đã quyết #21 (TOUCH chỉ tính label của checkbox/radio), #22, #23 (sửa tc_04_08 + lỗi `$s»` trong `proto-curl.sh`) ở `docs/sprints/1.5/proposals.md`.

## Việc
1. Sửa công cụ theo #21, #23 (trích số góp ý).
2. `source ~/.zprofile; pnpm -C frontend build && pnpm -C frontend exec next start -p 3400`.
3. Chạy **toàn bộ** TC `tc-US-PROTO-0{0..4}.md` + `tc-US-PROTO-DEMO.md` trên trình duyệt thật (không chỉ curl): `audit.mjs`, `threads-timeline.mjs`, `pii-matrix.mjs`, `proto-curl.sh`, và đi tay mọi TC có tương tác. Hồi quy đủ E1–E37.
4. Đi lại như chủ dự án: luồng Threads trọn vẹn với vai SV B rồi GV (tạo thread có/không khớp mẫu, Hỏi trợ lý AI ô trống/có chữ, @AI, phản hồi trễ TA + chuông, PII, GV xác nhận/sửa/loại, Hôm nay 6→7→6). Ghi cảm nhận: còn chỗ nào trông giả, rỗng, gượng.
5. Report `docs/sprints/1.5/qc/report-v5-US-PROTO-0{0..4}.md` + `report-v5-DEMO.md`: PASS/FAIL từng TC, bằng chứng (ảnh `shots/qc-v5/`, log), lỗi mới dạng bảng (route, vai, bước, thấy, mong đợi, mức, AC). Kết luận từng story.
6. `pnpm -C frontend lint`, `bash scripts/ui-antipatterns.sh`. Tắt server.

Commit chỉ `docs/sprints/1.5/qc/**`, `docs/sprints/1.5/shots/qc-v5/`. Trả lời: số PASS/FAIL theo story, lỗi mức cao. Làm thật kỹ.

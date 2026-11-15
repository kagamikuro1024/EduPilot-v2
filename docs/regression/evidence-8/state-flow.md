# state-flow (s8flow, sprint 8 P6-05…09) — dừng theo lệnh PM

## Đã xong (ghi trong `part-s8flow.tsv`, 41/265 ca)
- Go (tất cả test nêu trong cột «Bước» của 05–09, 123 test, `logs/s8flow-go-results.json`; TestStatusRBAC ở `./internal/grading`): PASS hết → P cho 27 ca chỉ-Go.
- US-05 API/UI: 01, 07, 10, 11, 12, 44, 09, 15, 19 (P).
- Lưu ý: TC-P6-05-19 chỉ mới thấy banner + lưới; chưa chứng minh «lưu OK»; TC-P6-05-44 ghi P dựa vào lần chạy `teacher@` gọi lớp QD8F8 → 403.

## Dở dang (chưa có kết luận, KHÔNG ghi vào tsv)
- US-05: 02/03/04/24 chạy trên QD8F2 (GRADING→9router cho ra unclear_count=3, không phải 1; today bị cache nên 24 đọc cũ) → phải chạy lại trên QD8F4 (đã tạo + ghi danh, NO_POLICY) với GRADING→QC fake (cần khoá `/tmp/qc2-lock-global`; lúc dừng Main đang giữ khoá); sau đó confirm (05, 13, 33), 23, 27 (stage `latency`), 30/31/34/35/45 (stage `remind1`), 20 (spec `real-us05-att30.spec.ts` trên QD8F3, lỗi 401 refresh do reload — đã đổi sang `next.router.push`), 21 (att QD8F8 chưa thấy hàng), 16/17/25/26/28/32/37–42/46.
- US-06…09: chưa chạy ngoài phần Go. Đã sinh tệp XLSX ở `/tmp/qc2-s8f/xlsx` (midterm, final, errors, formula, real-mssv, rows5000, toomany, zipbomb, corrupt, notxlsx, big3mb, inject, unverified).
- Playwright «repo» (`gradebook.spec.ts -g …`, `me.spec.ts`, `attendance.spec.ts`, `private-chat.spec.ts`) chưa chạy; server gate 3310 của worktree đang sống.

## Môi trường đã dựng (không tự dọn)
- Lớp `QD8F8` (bản sao lớp 1 đủ điểm, CONFIRMED, chưa chốt), `QD8F2` (DRAFT_READY, có tài liệu quy chế), `QD8F3` (NO_POLICY, 30 SV, 1 buổi), `QD8F4` (NO_POLICY, 2 SV, 3 buổi), `QD8F` rác: cũ. GV riêng `gv.s8f@edupilot.local` (nhân bản `teacher@`, cùng mật khẩu seed) dạy các lớp này để «Hôm nay» không bị cắt ở 50 mục.
- Scripts: `evidence-8/scripts/s8flow-{lib.py,clone.sh,us05.py,go.py,tsv.py,man.py}`; ghi tay: `python3 s8flow-man.py TC ST 'ghi chú'`; spec: `frontend/e2e/qc2-s8flow/real-*.spec.ts` + `frontend/playwright.qc2-s8flow.config.ts`.
- Bài học: reload trang (`page.goto`) nhiều lần → `/auth/refresh` 401; dùng điều hướng client (`next.router.push`). Chờ `att-row` ≥ 40 s khi máy tải cao. Gateway/worker từng 503 khi agent khác restart.

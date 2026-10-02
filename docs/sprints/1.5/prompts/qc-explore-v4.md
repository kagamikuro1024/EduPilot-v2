# Prompt cho `qc` — Kiểm thử thăm dò UI/UX trên bản v4 (song song lúc BA viết spec v5)

Chủ dự án chưa ưng UI và luồng Threads (góp ý #17, #18 ở `docs/sprints/1.5/proposals.md`, ảnh `docs/sprints/1.5/shots/pm-review-v4/`). BA đang viết spec v5. Việc của bạn: **tìm thêm lỗi** mà PM chưa nêu, để PM chuyển cho BA trước khi chốt spec.

## Việc
1. `source ~/.zprofile; pnpm -C frontend build && pnpm -C frontend exec next start -p 3400` (dừng khi xong).
2. Đi trọn như người dùng thật, cả 4 vai (`/login`), 1440 px và 390 px:
   - Luồng DEMO F2→F3→F5→F7→F9→F10→F17 (`docs/DEMO_SCRIPT.md`).
   - Từng nút, từng ô nhập, từng menu: bấm có phản hồi nhìn thấy không, có nút chết, chữ cắt, tràn ngang, lệch cột, khoảng trống vô nghĩa, số liệu mâu thuẫn giữa hai màn (ví dụ danh sách nói "2 phản hồi" mà chi tiết không có), dữ liệu mock chung chung không theo ngữ cảnh.
   - Đối chiếu `docs/design/DESIGN.md` §21 (phản mẫu) và §22 (định nghĩa xong).
3. Ghi `docs/sprints/1.5/qc/explore-v4.md`: bảng `# | route | vai | bề rộng | bước tái hiện | thấy gì | mong đợi | mức (cao/vừa/thấp) | ảnh`. Ảnh vào `docs/sprints/1.5/shots/qc-explore-v4/`. Không trùng 7+5 lỗi đã có ở #17, #18.
4. Không sửa spec, không sửa code, không sửa TC lúc này.

Commit chỉ `docs/sprints/1.5/qc/explore-v4.md` và `docs/sprints/1.5/shots/qc-explore-v4/`. Trả lời số lỗi theo mức và dừng.

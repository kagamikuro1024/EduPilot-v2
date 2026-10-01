# Prompt cho `qc` — chạy TC US-P0-02

Chạy `docs/sprints/1/qc/tc-US-P0-02.md` cho **US-P0-02**; handoff `docs/sprints/1/handoff/dev-US-P0-02.md`; spec `docs/specs/FEAT-scaffold/` (v2).

- Mỗi shell: `source ~/.zprofile`.
- Spec vừa đổi theo proposals #9 (AC2 dùng `pnpm --silent`; BA đang cập nhật US). Cập nhật TC tương ứng trước khi chạy, ghi lý do "spec đổi theo proposals #9".
- `dev` có thể đang làm US-P0-03 song song (chỉ đụng `.github/workflows/`, `package.json`). Nếu cần dựng stack, dùng `pnpm dev`; xong thì `pnpm dev:down`. Không sửa code.
- Báo cáo `docs/sprints/1/qc/report-US-P0-02.md` theo mẫu TEMPLATES (PASS/FAIL). Phần "bằng mắt" của AC2 (trang `/` đúng logo/font/màu) ghi "chủ dự án tự kiểm", không tính FAIL.
- Commit chỉ `docs/sprints/1/qc/**`. Trả lời ≤ 10 dòng rồi dừng.

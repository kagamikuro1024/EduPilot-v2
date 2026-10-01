# Prompt cho `dev` — US-P0-02

1. Nhận vai theo `docs/team/DEV.md`. Luật chung: `CLAUDE.md` (đang được cập nhật theo D46; chỗ nào còn nói Python/gRPC/`src/` thì `docs/DECISIONS.md` D45–D48 thắng).
2. Thi công **US-P0-02**, spec `docs/specs/FEAT-scaffold/` (US.md, SRS.md, QUESTIONS.md đã chốt). Nhánh `sprint/1-p0-prep` — không đổi nhánh, không tạo nhánh khác.
3. Bối cảnh: `docs/sprints/1/plan.md` (bản 3), `docs/phases/P0.md` L1.

## Môi trường máy (PM đã dựng)
- Mỗi shell mới: `source ~/.zprofile` (Homebrew, Node 24, Go 1.27, JDK, Python 3.11, pnpm, gh, k6). Docker qua colima (đang chạy; nếu không: `colima start`).
- Được tự cài thêm công cụ bằng `brew` nếu cần (ghi vào handoff). Không cần quyền sudo.
- `gh` đã đăng nhập; được `git push` nhánh `sprint/1-p0-prep`.

## Luật git trong sprint này (nhiều agent cùng repo)
- Không bao giờ `git add -A`, `git add .`, `git commit -a`. Chỉ stage đúng đường dẫn của mình; `git mv` thì stage sẵn — commit ngay sau mỗi bước.
- Không đụng `docs/**` (trừ file handoff của bạn), `CLAUDE.md`, `.claude/**`: đang có agent khác sửa.
- Gặp `index.lock` → chờ vài giây rồi thử lại, không xoá lock.
- Commit: `US-P0-02: <việc>`. Push sau khi xong.

## Khi xong
Handoff `docs/sprints/1/handoff/dev-US-P0-02.md` theo mẫu `docs/team/TEMPLATES.md` (mục "Lệnh QC chạy để kiểm" = đúng lệnh trong AC). Trả lời ≤ 10 dòng rồi dừng. Không làm US-P0-03.

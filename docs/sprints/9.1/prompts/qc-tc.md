# Prompt cho `qc`: TC cho FEAT-uiux-91 (pha 2)

Khi BA đổi spec `docs/specs/FEAT-uiux-91/` sang APPROVED:
- Viết TC theo đợt (1, 2, 6, 3, 4, 5), mỗi AC ít nhất một TC.
- Kiểm 375 px khi liên quan; kiểm `prefers-reduced-motion`; ảnh trước / sau.
- Kiểm chéo bắt buộc theo `QC.md`, gồm phân quyền của "Dùng thử" và PII.

TC viết ở nơi `QC.md` quy định. Push `sprint/9.1-uiux`, báo PM.

Mỗi khi dev báo xong một đợt (`handoff/dev-wave-<n>.md`):
- chạy TC của đợt đó trên stack `-p ep-qc` của VPS, trình duyệt `playwright-cli` trên Mac qua `ssh -L`;
- ghi `docs/sprints/9.1/qc/wave-<n>.md` (PASS / FAIL theo AC, ảnh, lỗi).

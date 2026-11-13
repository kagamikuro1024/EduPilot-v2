# Prompt cho `dev`: sprint 9.1 pha 1

## A. Sửa CI `ops/vps-deploy` trước tiên
`Frontend Lighthouse (simulate)` đỏ ở `resource-summary.script.size` > 256 KB trên 2 URL. Sửa mã, **không nới ngân sách**. Chạy lại cả bộ Playwright (gồm mobile) trên VPS. CI xanh thì báo PM. Sau đó gộp `ops/vps-deploy` vào `sprint/9.1-uiux`.

## B. Mẫu trang giới thiệu mới + video (sau khi BA có `audit/findings.md` mục trang giới thiệu; trong lúc chờ thì đọc plan và bắt đầu video)
- **Đọc trước:** `docs/sprints/9.1/plan.md` (mục Trang giới thiệu) và các skill `skill://impeccable` (chế độ Persuade, `animate`), `skill://design-taste-frontend`, `skill://emil-design-eng`, `skill://animate`, `skill://improve-animations`.
- **Video giới thiệu:**
  - kịch bản 60–90 s rút từ `DEMO_SCRIPT.md`: sinh viên hỏi chat (che tên) → giảng viên duyệt bài AI chấm → sổ điểm giải trình → Hôm nay;
  - quay bằng Playwright trên VPS (cửa sổ 1440 × 900, con trỏ hiện);
  - ffmpeg ghép đoạn, phụ đề tiếng Việt in vào hình, poster;
  - mp4 H.264 + webm, < 8 MB;
  - script dựng lại được ở `scripts/landing-video/`, tệp ra ở `frontend/public/landing/`.
- **Mẫu trang:** dựng trực tiếp trên nhánh (route `/` như hiện tại), chuyển động bằng CSS / WAAPI / `IntersectionObserver`, không thêm thư viện. Video `preload="none"` + poster, bật khi bấm. Giữ LCP ≤ 2,5 s, JS ≤ 256 KB, axe 0, `prefers-reduced-motion`.
- **Deploy mẫu lên VPS** (chỉ dựng lại frontend). Gửi PM ảnh 1440 / 375 + link video.

Ghi `docs/sprints/9.1/handoff/dev-phase1.md`. Build / test trên VPS (`/srv/work/dev`). Không mở subagent.

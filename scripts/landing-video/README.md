# Dựng video, clip và ảnh trang giới thiệu

Quay một bản chạy EduPilot đã seed (chỉ đọc, không gửi câu hỏi, không sửa ô nào) rồi mã hoá thành `frontend/public/landing/`.

## Cần
- Docker (ảnh `mcr.microsoft.com/playwright:v1.63.0-noble` và `debian:trixie-slim`), chạy ở gốc repo.
- Một bản EduPilot đã chạy với dữ liệu seed: lớp `761987`, tài khoản sinh viên `sv.kha`, giảng viên `teacher`, cùng mật khẩu seed.

## Chạy
```bash
# bản demo công khai (HTTPS)
EP_HOST=<host> SEED_DEFAULT_PASSWORD=<mật khẩu seed> bash scripts/landing-video/run.sh
# hoặc đọc từ một tệp .env
ENV_FILE=/opt/edupilot/.env.local bash scripts/landing-video/run.sh
# stack thử (ví dụ ep-qc) chỉ bind 127.0.0.1: chạy trên chính máy đó, trỏ vào cổng Caddy (có /api), không phải cổng frontend; HTTPS chứng chỉ nội bộ được chấp nhận (EP_BASE_URL=https://localhost:<cổng HTTPS>)
EP_BASE_URL=http://127.0.0.1:<cổng> SEED_DEFAULT_PASSWORD=… bash scripts/landing-video/run.sh
# chỉ chụp lại ba ảnh khung ghim (≈ 1 phút)
STILLS_ONLY=1 EP_HOST=… SEED_DEFAULT_PASSWORD=… bash scripts/landing-video/run.sh
```

| Biến | Mặc định | Ý nghĩa |
| --- | --- | --- |
| `EP_HOST` | – | host của bản chạy (HTTPS). Hoặc `EP_BASE_URL` |
| `EP_BASE_URL` | – | địa chỉ đầy đủ; có biến này thì container dùng `--network host` |
| `SEED_DEFAULT_PASSWORD` | – | mật khẩu seed (không in ra, không ghi vào tệp) |
| `ENV_FILE` | – | tệp `.env` để đọc hai biến trên nếu chưa đặt |
| `EP_CLASS_CODE` | `761987` | lớp quay (lớp có bài nộp và công thức đã xác nhận) |
| `EP_STUDENT_EMAIL`, `EP_TEACHER_EMAIL` | `sv.kha@…`, `teacher@…` | tài khoản seed |
| `OUT`, `PWMOD` | `.landing-out`, `.landing-pw` | thư mục khung hình trung gian và nơi cài Playwright |

## Đầu ra (`frontend/public/landing/`)
| Tệp | Ngân sách |
| --- | --- |
| `gioi-thieu-v1.{mp4,webm,webp}` video đầy đủ ≈ 65 s + poster | mp4 ≤ 1,2 MB |
| `gioi-thieu-clip-v1.{mp4,webm}` clip lặp ≈ 7 s (màn đầu) | mỗi định dạng ≤ 1 MB |
| `gioi-thieu-clip-v1.webp` (1280 px), `-640.webp` (điện thoại) | ≤ 60 KB, ≤ 25 KB |
| `story-{chat,review,gradebook}-v1.webp` (1600 px) và `-800.webp` | ≤ 90 KB |

Ảnh khung ghim chụp riêng ở cửa sổ 1360 × 850, độ phân giải gấp đôi, không phụ đề, không con trỏ. Phụ đề của video nằm ở mảng `CAPTIONS` trong `record.mjs`; lời thay cho video trong `Landing.tsx` (`VIDEO_CAPTIONS`) phải khớp.
Mỗi cảnh đăng nhập mới (refresh token xoay vòng); con trỏ vòng đỏ do script tự vẽ.

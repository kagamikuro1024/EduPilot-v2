# Dev — đợt 6 (trang giới thiệu v2) · sprint 9.1 pha 2

Commit: `6159c3e0` (và `09960dd1`), nhánh `sprint/9.1-uiux`. Đã deploy frontend lên bản demo công khai `https://v2202609419695521098.happysrv.de/` để chủ dự án xem (cùng bản dựng có đợt 1 + 2 vì chung một ảnh frontend; QC chưa PASS hai đợt đó — PM quyết định giữ hay lùi).
Stack kiểm: `-p ep-qc`, `/srv/work/qc`, nhánh `sprint/9.1-uiux` ở commit trên (theo `qc/tc-00-chung.md`). Không còn "Dùng thử" (P-91-4): US-22 không làm, không code nào liên quan.
Ảnh sau 1440 + 375: `dev-wave-6-shots/` (ảnh trước là `dev-phase1-shots/`). Video: `/landing/gioi-thieu-v1.mp4`.

## Làm được (US-08, US-09)
- `features/landing/Landing.tsx` + `Landing.module.css` viết lại: chữ lớn (h1 clamp), khung ứng dụng bóng họ popover, "Xem EduPilot làm việc" (khung ghim ≥ 1100 px dùng `view-timeline` + `timeline-scope`, 3 ảnh `story-*-v1.webp`; < 1100 px hoặc `reduce`: mỗi bước kèm ảnh, không ghim), sơ đồ ba bước nối bằng đường chỉ, số đo (146 đếm bằng `@property --n`; "30/30", "+0,05 %" tĩnh), hai cột vai (không tab), CTA cuối, hover nâng 6 px (`hover: hover`), mở đầu ≤ 600 ms.
- `features/landing/LandingMedia.tsx` (client, không render gì): clip màn đầu nạp **sau** `load` (không bao giờ khi `reduce`, `Save-Data`, < 720 px; tab ẩn thì dừng, hiện lại thì phát lại), nút Tạm dừng / Phát (`aria-pressed`, nhớ trong `sessionStorage`); nút `Xem video` mở `<dialog>` gốc (Esc, bấm ngoài khung, trả focus), đóng thì dừng; poster của video đầy đủ chỉ gán khi mở. Thay `PlayState.tsx` (đã xoá).
- `scripts/landing-video/`: quay thêm clip 7 s (`gioi-thieu-clip-v1.{mp4,webm,webp}`) và 3 ảnh khung ghim (chụp không phụ đề, không con trỏ); `bash scripts/landing-video/run.sh` trên VPS dựng lại tất cả. Video 64,7 s quay lại với nhãn mới ("Hỏi đáp công khai").
- Kích thước: clip mp4 214 KB / webm 400 KB (≤ 1 MB), poster clip 64 KB (mục tiêu 60 KB, vượt 4 KB), ảnh story 28 / 28 / 53 KB (≤ 90 KB).

## Đo (VPS, bản demo công khai, ẩn danh)
| Phép đo | Kết quả | Ngưỡng |
| --- | --- | --- |
| Lighthouse throttle thật (devtools), di động, 3 lượt | LCP 2,37 / 2,24 / 2,35 s; TBT 128–159 ms; CLS 0,001; perf 97 | LCP ≤ 2,5 s |
| JS tải ở `/` | 217 KB | ≤ 256 KB |
| Lighthouse mô phỏng, di động, 3 lượt | LCP 2,41 / 2,71 / 2,56 s (trung vị 2,56 s); perf 91 / 94 / 97; a11y 100 | giới hạn: mô phỏng trên VPS dao động ±0,3 s, `lighthouserc.json` đã tắt LCP mô phỏng, cổng LCP là bản throttle thật |
| Lighthouse desktop, 3 lượt | LCP 0,53–0,61 s; perf 100; a11y 100; CLS 0 | |
| Playwright `landing.spec.ts` (desktop) | 13 passed | |

## Lệch spec / quyết định
- **Số đo**: "22 bài / 20 SV" **không có** trong `docs/sprints/9/report.md` (dòng 25 chỉ có "TTFT +0,05 %"), nên bỏ 22; còn 146 (đếm), 30/30, +0,05 % — mỗi số khớp một dòng báo cáo sprint 6 / 8 / 9. Chú thích "Đo trên dữ liệu mô phỏng." (AC9).
- **JS**: dùng một client component `LandingMedia` thay vì script nội tuyến, vì script do React render không chạy khi điều hướng client (ví dụ `/login` → `/`). DESIGN §11.1 cho phép một client component nhỏ để mở video.
- **Di động < 720 px**: khối chữ đầu `min-height: calc(100svh - 12rem)` để chữ là LCP (poster chỉ preload ≥ 720 px) — để lại khoảng trống ~300 px giữa nút và ảnh; đổi lấy LCP 2,3 s. Nếu muốn ảnh lên cao hơn thì LCP sang ảnh.
- Nút phụ `Xem video` có từ v1 → AC2 đủ ba nút; không có "Dùng thử".
- Test `landing.spec.ts` viết lại theo AC21 (bỏ test tab vai và `PlayState` của v1).

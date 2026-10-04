# Câu hỏi Tech Lead — sprint 4

## TL-1 (Cổng P2 G1 — Lighthouse TBT > 200 ms trên runner CI chậm)
**Bối cảnh.** CI `lighthouse ci` đỏ 4 lần liên tiếp (404e0ec, beda12b, c6985d8, da287fe): TBT trung vị 6 route người dùng 184–239 ms (ngưỡng `error` 200); chỉ `/dev/ui` (đã `warn`) cao hơn. Runner `benchmarkIndex` ≈ 2100–2300; lần xanh duy nhất (9a62f5b) benchmarkIndex ≈ 3900 và TBT vẫn 71–217 ms. Máy dev (benchmarkIndex ≈ 3700): TBT 40–90 ms. Báo cáo `.lighthouseci` từ artifact: `gh run download 37168027868 -n frontend-reports`.
**Đã làm (528b9db).** Hồ sơ CDP `/inbox` thấy 2 tác vụ dài: (a) ~107 ms nạp/chạy tập lệnh, (b) ~170 ms dựng khung + trang ngay khi `POST /auth/refresh` xong (`useSyncExternalStore` luôn đồng bộ). Sửa (b) bằng `useDeferredValue` quanh snapshot phiên (dựng theo lát). Trên CI trung vị giảm ≈ 45 ms (243→219, 259→216, 281→214, 256→184, 386→204, 265→239) nhưng chưa đủ.
**Còn lại (giả lập CPU 12× ≈ runner CI):** tác vụ nạp ~205 ms (chặn ≈ 155 ms) — thời gian tự thân: `react-dom` 248, runtime turbopack 178, `next` client 93, chunk mock (`BT03_SEED`/`STUDENTS`) 87, `scrollTo` 53; cộng 1–3 tác vụ ~100 ms sau khi có `/me/courses`. Layout bundle (mọi route) kéo `mock/*` (≈ 180 KB giải nén, đề xuất #26 sprint 3) vì `AppShell` (badge Hộp thư / Chấm bài) và `session.tsx` (`STUDENTS`, `COURSES`, `mockCourseFor`).
**Phương án.**
1. Tách mock khỏi layout: badge nav và ánh xạ lớp mô phỏng nạp bằng `import()` sau lần vẽ đầu (hook `useMockBadges`), `session.tsx` chỉ giữ lớp thật. Lợi ≈ 10–20 % nạp; đổi cấu trúc shell/session (rủi ro hồi quy ~20 màn mô phỏng).
2. Không dựng khung + trang cho tới khi `/me/courses` về (bỏ 1–2 lần dựng lại); rẻ, nhưng khung xương lâu hơn ≈ 1 RTT.
3. Nới ngưỡng TBT (`warn`) — PM đã bác.
**Nghiêng về:** 2 trước (đo), rồi 1 nếu chưa đủ. Câu hỏi: có cách nào rẻ hơn để giảm chi phí nạp của framework (ví dụ bỏ `scrollTo`/focus của app-router lúc mount, chia chunk) mà tôi bỏ sót? Có chấp nhận 1 không?

**Bổ sung (sau thử nghiệm, CPU 12× ≈ runner CI):**
- Tác vụ nạp **không do mock**: `/login` (không `AppShell`, không mock) cũng có tác vụ nạp 236–316 ms; `/inbox` 231–298 ms; `/dev/ui` 233–277 ms. Phương án 1 (tách mock) vì vậy chỉ rút được phần nhỏ (chunk mock tự thân ≈ 87/1100 ms).
- Phương án 2 (giữ khung xương tới khi `/me/courses` về) **làm xấu hơn**: tác vụ dựng sau đó thành 260–470 ms (cập nhật của react-query là đồng bộ nên không còn lát thời gian). Thử thêm `useDeferredValue` cho `mine.data`: không cải thiện (thêm 1–2 tác vụ 60–100 ms). Đã hoàn lại cả hai; chỉ giữ `useDeferredValue` của phiên (528b9db).
- Tính theo công thức TBT: chỉ phần vượt 50 ms của MỖI tác vụ được tính ⇒ tác vụ nạp ~205 ms (CPU 12×) đã chặn ≈ 155 ms; mọi việc sau đó chỉ cần ≥ 45 ms chặn là vượt 200. Tức ngưỡng 200 ms ở tốc độ runner này gần như bằng chi phí nền của React 19 + Next 16 + react-query trên mọi route.
- Phương án mới 4: **hiệu chỉnh `cpuSlowdownMultiplier`** theo tốc độ runner (Lighthouse khuyến nghị hiệu chỉnh khi `benchmarkIndex` máy đo khác thiết bị tham chiếu): runner 2100–2300 so với máy dev 3700 ⇒ nhân 4 × 2200/3700 ≈ 2,4 để cùng kịch bản "Moto G4-class". Đây là đổi cấu hình đo, không đổi ngưỡng — cần Tech Lead / PM chấp nhận vì giống nới ngưỡng.

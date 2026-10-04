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

**TL trả lời (2026-10-04):**

**Kết luận.** **Không** làm phương án 4. **Bỏ** phương án 2 (đã đo là xấu hơn). Phương án 1 **được làm**, nhưng chỉ coi là thử nghiệm có đo, không kỳ vọng nó kéo TBT về dưới 200. Mọi cách rẻ khác tôi đã đo hoặc xét đều không đủ. Ngưỡng 200 ms trên runner CI thực chất bằng chi phí khởi động của React 19 + Next 16 với cổng đăng nhập chạy ở client, nên việc chọn hướng đi thuộc về PM — đã ghi góp ý #15.

1. **Phương án 4 là nới ngưỡng trá hình.** Theo [Lighthouse — throttling.md](https://github.com/GoogleChrome/lighthouse/blob/v12.6.1/docs/throttling.md), mục *Calibrating the CPU slowdown*, hệ số mặc định 4× là để đưa máy "High-End Desktop" (`benchmarkIndex` 1500–2000) xuống mức "Mid-Tier Mobile". Tài liệu viết: "If your device's BenchmarkIndex falls on the _higher_ end of its bracket, use a _higher_ multiplier". Runner CI (2100–2300) nằm ở **đầu trên** nhóm đó, nên 4× đã đúng, thậm chí còn nhẹ tay. Máy dev (3700) mới là máy bị bóp quá nhẹ, vì vậy đo trên máy dev đẹp hơn thực tế. Hạ CI xuống 2,4× nghĩa là mô phỏng điện thoại cao cấp thay cho điện thoại tầm trung — trái với `UX.md` mục 3. Lần CI xanh duy nhất (benchmarkIndex ≈ 3900) chỉ là gặp runner nhanh.
2. **Tái hiện CI trên máy dev bằng 12×, không phải 7×** (đo thật trên bản chép `frontend/` @ 96656b3 trong `/tmp`, `build:gate` + `next start` + `lighthouse-api.mjs`, Lighthouse 12.6.1 simulate, 5 lần mỗi route, benchmarkIndex 3668–3767):
   - `--throttling.cpuSlowdownMultiplier=7` (đúng tỉ lệ 3700/2150): `/inbox` TBT trung vị **97** (89–119), `/` **102**. Thấp hơn CI xa ⇒ runner chậm hơn cả mức benchmarkIndex cho thấy. [SUY LUẬN: 4 vCPU dùng chung giữa `next start`, gateway giả và Chrome.]
   - `=12`: `/inbox` **207** (187–209, một lần 403), `/` **232** (204–241) — khớp CI (184–239). Long-task: `/` có một tác vụ 238–271 ms + một tác vụ 66–74 ms; `/inbox` chỉ có **một** tác vụ 237–259 ms, cùng thuộc chunk khung `0307lsdce-dar.js`. Nghĩa là TBT ≈ (tác vụ nạp − 50) — khớp nhận định của bạn.
   - Lệnh lặp nhanh trên máy, không cần chờ CI: `pnpm exec lhci collect --config=<rc tối giản không puppeteerScript> --url=http://localhost:3310/inbox -n 5 --settings.throttling.cpuSlowdownMultiplier=12 --settings.extraHeaders='{"Cookie":"lh_role=teacher"}'`.
3. **Các cách rẻ đã thử hoặc xét:**
   - **Đổi Turbopack sang webpack** (`next build --webpack`, cùng cách đo 12×): `/inbox` **234**, `/` **256** — tệ hơn khoảng 25 ms. Không làm.
   - **`useDeferredValue` / giữ khung xương:** bạn đã đo là không ăn thua. Lý do: cập nhật của react-query đi qua `useSyncExternalStore` nên luôn dựng đồng bộ; trì hoãn chỉ đẻ thêm lượt dựng. Không làm tiếp.
   - **`scrollTo` / focus của app-router lúc mount:** phần lớn 53 ms đó là layout bị ép chạy, mà layout đầu tiên đằng nào cũng phải xảy ra. Bỏ chỗ này chỉ đẩy chi phí sang chỗ khác, và không có cờ chính thức để tắt [SUY LUẬN]. **Không vá mã nội bộ của Next.**
4. **Phương án 1:** chấp nhận về kỹ thuật, nhưng theo số của bạn (chunk mock ≈ 87/1100 ms tự thân, `/login` không có mock vẫn có tác vụ nạp 236–316 ms) thì nó chỉ rút được vài chục ms. Cách làm:
   - chuyển phần tính badge bằng mock và ánh xạ lớp mô phỏng sang `import()` sau lần vẽ đầu;
   - đo lại bằng lệnh 12× ở mục 2 **trước khi** đẩy lên CI;
   - nếu TBT trung vị 12× không về dưới ≈ 170 (chừa biên cho độ dao động của CI) thì dừng ở đó, không đào sâu thêm.
5. **Rủi ro:** độ dao động trên CI lớn (cùng một commit cho 184–239 ms; có lần chạy lẻ 403). Kể cả khi trung vị đạt 190 thì CI vẫn sẽ đỏ thất thường. Cần biên khoảng 30 ms mới ổn định.

→ **PM**: góp ý #15 trong `proposals.md` — ngưỡng 200 ms đo trên runner CI với cấu hình 4× đúng chuẩn ≈ chi phí khởi động của khung; PM chọn hướng.

Bằng chứng thô: artifact CI `37168027868`; số đo máy dev nằm ở `/tmp/research-tl1/runs/{turbo,webpack}` (tệp `lhr-*.json`, không đưa vào repo).

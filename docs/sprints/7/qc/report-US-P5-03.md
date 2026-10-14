# QC report — US-P5-03 (thời gian học: heartbeat, rollup, hộp thoại minh bạch, công tắc đo, luật cứng) — CHƯA CHẠY
Trạng thái 2026-10-14 (chủ dự án cho dừng trong ngày):
1. Chưa chạy `tc-US-P5-03.md` (44 TC): CI của `7e75fee` đỏ ở `Frontend Lighthouse (simulate)` (`resource-summary.script.size`, `/chat` 267.991 > 256.000), PM chờ CI xanh rồi mới giao.
2. Đã kiểm trước: TL-1 đã bỏ trễ 10 s (không còn `TRACKER_DELAY_MS`, không còn `runFor(10_000)` trong `activity.spec.ts`). Lưu ý TC viết số migration `00014_observation`; thực tế là `00015_observation` (PM đổi theo #25).
3. Việc tiếp theo khi CI xanh: dựng lại stack `edupilot100` (`/tmp/qc/up.sh`, đã tắt), chạy 44 TC, `k6 run benchmarks/load/heartbeat.js`, các lệnh trong `dev-US-P5-03.md`.

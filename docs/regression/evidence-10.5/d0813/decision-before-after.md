# Decision model: đo trước / sau

Cấu hình: MacBook Apple Silicon, Colima 6 CPU / 10 GB · provider decision GIẢ (p50 426 / p95 682 ms) · lượt chat 3 · 2026-11-24 23:13 · lớp 761987.

## Chat (30 câu × lượt)

| | tắt | bật |
|---|---|---|
| TTFT p50 / p95 (ms) | 10.5 / 9224.9 | 815.0 / 10979.1 |
| Sự kiện đầu p50 / p95 (ms) | 6.3 / 12.3 | 7.2 / 14.5 |
| Độ chính xác (rules / course_qa / paraphrase) | 0.667 (1.0 / 1.0 / 0.0) | 0.689 (1.0 / 1.0 / 0.067) |
| Loại khỏi TTFT (trả từ cache) / lỗi / lần chờ 429 | 0 / 0 / 3 | 0 / 0 / 3 |
| FAST / FALLBACK / FAIL_CLOSED | — | 3 / 51 / 0 |

## Threads (40 bài)

| | tắt | bật |
|---|---|---|
| Đăng p50 / p95 (ms) | 7.7 / 14.9 | 5123.2 / 6831.9 |
| Chặn nhầm (trên 20 bài sạch) | 0 (0.0) | 11 (0.55) |
| Chặn đúng / lọt (20 bài cần chặn) | 11 / 9 | 20 / 0 |
| FAST / FALLBACK / FAIL_CLOSED | — | 6 / 11 / 12 |

## Chấm 12 bài (6 biến thể chèn lệnh + 6 đối chứng)

| | tắt | bật |
|---|---|---|
| Cờ đúng / cờ nhầm | 6/6 · 0/6 | 6/6 · 0/6 |
| Thời gian chấm cả lô (s) / mỗi bài | 218.5 / 18.2 | 194.2 / 16.2 |
| FAST / FALLBACK / FAIL_CLOSED | — | 4 / 7 / 0 |

## Kết luận từng cổng

| Cổng | Kết luận | Số liệu |
|---|---|---|
| CHAT_ROUTING | **Bật được** | TTFT p95 ×1.19 (≤ 1,20), độ chính xác 0.667 → 0.689 |
| THREADS_PRECHECK | **Giữ tắt** | chặn nhầm 0.55 (≤ 0,05), lọt 9 → 0 |
| INJECTION_FLAG | **Bật được** | cờ đúng 6/6, cờ nhầm 0/6 |

## Giới hạn

- Độ trễ decision là mẫu log-chuẩn của provider giả (p50 / p95 ở trên), không phải trễ mạng từ Việt Nam; nhãn trả theo bản phát lại `llm/testdata/decision-replay.json` dựng từ chính bộ nhãn này (mô phỏng, không phải độ chính xác của Gemini).
- Độ chính xác chat đo bằng block dữ liệu của trợ lý (điểm danh / điểm / cộng điểm) trong câu trả lời, không đọc nhãn ý định nội bộ.
- Câu trả lời trả từ cache bị loại khỏi TTFT; cache là theo lớp nên lượt «bật» có thể có nhiều câu cache hơn lượt «tắt» (xem hàng «Loại khỏi TTFT»).
- Cột «thật» (AC6): xem bảng dưới.

## Lượt thật (connected · gemini-decision-1 · 2026-11-24 · kế hoạch 142/300 lời gọi)

| Cổng | Lời gọi | p50 / p95 (ms, từ máy chạy) | Lỗi % | Chi phí (VND) |
|---|---|---|---|---|
| THREADS_PRECHECK | 29 | 801 / 804 | 79.3103448275862 | 0 |
| INJECTION_FLAG | 11 | 2466 / 2826.5 | 9.090909090909092 | 0 |
| CHAT_ROUTING | 54 | 801 / 804 | 88.88888888888889 | 0 |

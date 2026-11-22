# Decision model: đo trước / sau

Cấu hình: MacBook Apple Silicon, Colima 6 CPU / 10 GB · provider decision GIẢ (p50 426 / p95 682 ms) · lượt chat 3 · 2026-11-22 20:01 · lớp 761987.

## Chat (30 câu × lượt)

| | tắt | bật |
|---|---|---|
| TTFT p50 / p95 (ms) | 8.5 / 17.1 | 317.5 / 635.4 |
| Sự kiện đầu p50 / p95 (ms) | 5.6 / 9.7 | 6.1 / 22.5 |
| Độ chính xác (rules / course_qa / paraphrase) | 0.667 (1.0 / 1.0 / 0.0) | 0.922 (1.0 / 1.0 / 0.767) |
| Loại khỏi TTFT (trả từ cache) / lỗi / lần chờ 429 | 0 / 0 / 3 | 0 / 0 / 5 |
| FAST / FALLBACK / FAIL_CLOSED | — | 53 / 1 / 0 |

## Threads (40 bài)

| | tắt | bật |
|---|---|---|
| Đăng p50 / p95 (ms) | 7.0 / 15.9 | 408.8 / 604.3 |
| Chặn nhầm (trên 20 bài sạch) | 0 (0.0) | 0 (0.0) |
| Chặn đúng / lọt (20 bài cần chặn) | 11 / 9 | 11 / 9 |
| FAST / FALLBACK / FAIL_CLOSED | — | 0 / 29 / 0 |

## Chấm 12 bài (6 biến thể chèn lệnh + 6 đối chứng)

| | tắt | bật |
|---|---|---|
| Cờ đúng / cờ nhầm | 6/6 · 0/6 | 6/6 · 0/6 |
| Thời gian chấm cả lô (s) / mỗi bài | 27.4 / 2.3 | 27.3 / 2.3 |
| FAST / FALLBACK / FAIL_CLOSED | — | 6 / 5 / 0 |

## Kết luận từng cổng

| Cổng | Kết luận | Số liệu |
|---|---|---|
| CHAT_ROUTING | **Giữ tắt** | TTFT p95 ×37.16 (≤ 1,20), độ chính xác 0.667 → 0.922 |
| THREADS_PRECHECK | **Bật được** | chặn nhầm 0.0 (≤ 0,05), lọt 9 → 9 |
| INJECTION_FLAG | **Bật được** | cờ đúng 6/6, cờ nhầm 0/6 |

## Giới hạn

- Độ trễ decision là mẫu log-chuẩn của provider giả (p50 / p95 ở trên), không phải trễ mạng từ Việt Nam; nhãn trả theo bản phát lại `llm/testdata/decision-replay.json` dựng từ chính bộ nhãn này (mô phỏng, không phải độ chính xác của Gemini).
- Độ chính xác chat đo bằng block dữ liệu của trợ lý (điểm danh / điểm / cộng điểm) trong câu trả lời, không đọc nhãn ý định nội bộ.
- Câu trả lời trả từ cache bị loại khỏi TTFT; cache là theo lớp nên lượt «bật» có thể có nhiều câu cache hơn lượt «tắt» (xem hàng «Loại khỏi TTFT»).
- Cột «thật» (AC6): chờ phép PM.

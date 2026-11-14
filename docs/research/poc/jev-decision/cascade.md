# Cascade theo độ tin cậy: ngưỡng đề xuất cho đợt 6 (D60, P-11-10)

Nguồn số: PoC 14/11/2026, chạy một lần. Decision gọi thẳng `gemini-decision-1`, LLM là `ag/gemini-3.8-flash` qua 9router, cùng dữ liệu mô phỏng đã che. **Không gọi thêm model thật để tinh chỉnh:** ngưỡng chỉnh lại trên tập `dev` bằng provider `decision` giả cộng bản ghi, hoặc khi PM cho phép một lượt thật.

Ký hiệu: `p` là `noul` (xác suất "có"); `c` là `confidence` của choice; "LLM" là `Structured` qua 9router.

| Điểm | Luật quyết định | Ngưỡng | Lỗi / hết giờ | Bằng chứng PoC | Đo nghiệm thu |
| --- | --- | --- | --- | --- | --- |
| Tường lửa Threads precheck (chỉ thêm chặn) | Luật tất định chạy trước, giữ nguyên. Phần luật cho qua → decision noul "riêng tư / tên người cụ thể": `p ≥ 0,5` chặn; `p ≤ ε` cho qua; ở giữa → LLM | **ε = 0,05** | Decision lỗi → LLM `Structured` qua 9router; LLM cũng lỗi → `FAIL_CLOSED`: chặn bài, không lưu, cảnh báo `DECISION_FAIL_CLOSED` (SRS `FEAT-decision-model`) | Phần luật không tự chặn (142 mẫu): 129/142 đi đường nhanh, đúng 139/142, chặn nhầm 0; S7 (tên ngoài danh sách lớp) 4/4. Cộng với luật (S6 38/38 ở E1) → 100/100 | E1 tập `test`: `false_block` ≤ 0,05, recall không giảm so với luật; p95 precheck |
| Lưới khủng hoảng | **Không dùng decision** (P-11-11). Giữ regex hai tầng, **0 lời gọi model** (US-P3-04 AC13). Định tuyến chạy decision song song nhưng không được ghi đè kết quả CRISIS của luật | – | – | Tham khảo (không áp dụng): PoC `gemini-decision-1` 37/37 trên bộ 37 câu | `crisis_20` / `crisis_neg_17` vẫn qua bằng regex |
| Cờ chèn lệnh (làn BATCH) | Cờ = regex OR `Structured` (P-11-9, gộp trong lời gọi chấm) OR decision `p ≥ 0,5`. Chỉ gắn `INJECTION_SUSPECTED`, không đổi điểm | **τ = 0,5** | Lớp nào lỗi thì lớp đó không thêm cờ; ghi audit | 6/6 dương; gắn cờ nhầm 1/6 (`s4` "Em nộp nhầm tệp…", p = 0,91, nhãn còn bàn được). Dương thấp nhất 0,79 | Bộ 6 biến thể 9.5 + 6 bài sạch: recall 6/6 |
| Định tuyến ý định | Decision choice (10 ý định, **không** có CRISIS vì lưới riêng lo): `c ≥ τ` dùng ngay; dưới τ → **bộ định tuyến luật hiện có** | **τ = 0,95** | Lỗi / hết giờ (ngân sách 800 ms, chạy song song với lần nhúng) → luật | 84 % đi đường nhanh, đúng 100 % phần nhanh. Ba câu decision sai có c = 0,91 / 0,89 / 0,63, đều dưới τ nên rơi về luật → **61/61** | `routeCases` 61/61 + bộ diễn đạt lại giữ riêng; **TTFT trước / sau** (k6 `chat.js`, cùng stack) |
| Khủng hoảng, escalate, mức rubric, điểm, công thức | **Không dùng** (D60 (5), P-11-11) | – | – | – | – |

Định tuyến: dưới τ thì về **bộ định tuyến luật**, không gọi LLM (P-11-11, khớp D47(1)). Thứ tự trên chat:
1. luật khủng hoảng;
2. nếu không phải khủng hoảng: decision (ngân sách 800 ms, chạy song song với lần nhúng). `c ≥ 0,95` thì dùng nhãn của decision; dưới ngưỡng hoặc lỗi thì dùng luật.

Đo TTFT trước / sau trên cùng stack, với provider `decision` giả có độ trễ đặt bằng p50 / p95 đo ở PoC (426 / 682 ms).

## Bảng mô phỏng đầy đủ (sinh bởi `cascade.py`)

| Tác vụ | Cấu hình | % đi đường nhanh | Độ đúng phần nhanh | Độ đúng tổng | Sót dương | Trễ p50 / p95 (ms) | Chi phí / 1.000 yêu cầu (USD, ước tính) |
| --- | --- | --- | --- | --- | --- | --- | --- |
| A tường lửa (200) | chỉ LLM | 0% | – | 0.960 | 8 | 4642 / 12344 | 0.328 |
| A tường lửa (200) | chỉ gdec | 100% | 0.895 | 0.895 | 21 | 455 / 808 | 0.027 |
| A tường lửa (200) | gdec τ=0.5 | 100% | 0.895 | 0.895 | 21 | 455 / 808 | 0.027 |
| A tường lửa (200) | gdec τ=0.6 | 96% | 0.916 | 0.915 | 17 | 456 / 2199 | 0.041 |
| A tường lửa (200) | gdec τ=0.7 | 88% | 0.955 | 0.945 | 11 | 463 / 9013 | 0.066 |
| A tường lửa (200) | gdec τ=0.8 | 82% | 0.970 | 0.960 | 8 | 476 / 9502 | 0.086 |
| A tường lửa (200) | gdec τ=0.9 | 73% | 0.979 | 0.960 | 8 | 496 / 11291 | 0.115 |
| A tường lửa (200) | gdec τ=0.95 | 68% | 0.993 | 0.970 | 6 | 546 / 11508 | 0.133 |
| A tường lửa (200) | gdec τ=0.98 | 47% | 1.000 | 0.970 | 6 | 4084 / 11739 | 0.200 |
| A tường lửa (200) | gdec bất đối xứng ε=0.02 | 80% | 1.000 | 0.985 | 3 | 493 / 5596 | 0.092 |
| A tường lửa (200) | gdec bất đối xứng ε=0.05 | 88% | 0.994 | 0.980 | 4 | 464 / 5565 | 0.066 |
| A tường lửa (200) | gdec bất đối xứng ε=0.1 | 90% | 0.983 | 0.970 | 6 | 463 / 5272 | 0.059 |
| A tường lửa (200) | gdec bất đối xứng ε=0.2 | 92% | 0.973 | 0.960 | 8 | 463 / 5068 | 0.053 |
| A tường lửa (200) | chỉ jev | 100% | 0.975 | 0.975 | 5 | 308 / 549 | 0.020 |
| A tường lửa (200) | jev τ=0.5 | 100% | 0.975 | 0.975 | 5 | 308 / 549 | 0.020 |
| A tường lửa (200) | jev τ=0.6 | 98% | 0.979 | 0.980 | 4 | 308 / 654 | 0.028 |
| A tường lửa (200) | jev τ=0.7 | 96% | 0.984 | 0.975 | 5 | 308 / 672 | 0.033 |
| A tường lửa (200) | jev τ=0.8 | 91% | 0.995 | 0.975 | 5 | 309 / 5614 | 0.049 |
| A tường lửa (200) | jev τ=0.9 | 86% | 1.000 | 0.975 | 5 | 312 / 7547 | 0.066 |
| A tường lửa (200) | jev τ=0.95 | 63% | 1.000 | 0.970 | 6 | 359 / 9498 | 0.141 |
| A tường lửa (200) | jev τ=0.98 | 2% | 1.000 | 0.960 | 8 | 4996 / 12662 | 0.341 |
| A tường lửa (200) | jev bất đối xứng ε=0.02 | 48% | 1.000 | 0.985 | 3 | 3744 / 5979 | 0.192 |
| A tường lửa (200) | jev bất đối xứng ε=0.05 | 86% | 1.000 | 0.985 | 3 | 310 / 5150 | 0.064 |
| A tường lửa (200) | jev bất đối xứng ε=0.1 | 96% | 1.000 | 0.985 | 3 | 308 / 669 | 0.033 |
| A tường lửa (200) | jev bất đối xứng ε=0.2 | 98% | 0.995 | 0.985 | 3 | 308 / 654 | 0.028 |
| B khủng hoảng (37) | chỉ LLM | 0% | – | 1.000 | 0 | 4165 / 5286 | 0.249 |
| B khủng hoảng (37) | chỉ gdec | 100% | 1.000 | 1.000 | 0 | 400 / 743 | 0.017 |
| B khủng hoảng (37) | gdec τ=0.5 | 100% | 1.000 | 1.000 | 0 | 400 / 743 | 0.017 |
| B khủng hoảng (37) | gdec τ=0.6 | 100% | 1.000 | 1.000 | 0 | 400 / 743 | 0.017 |
| B khủng hoảng (37) | gdec τ=0.7 | 95% | 1.000 | 1.000 | 0 | 418 / 5259 | 0.030 |
| B khủng hoảng (37) | gdec τ=0.8 | 92% | 1.000 | 1.000 | 0 | 433 / 5259 | 0.035 |
| B khủng hoảng (37) | gdec τ=0.9 | 89% | 1.000 | 1.000 | 0 | 433 / 5259 | 0.042 |
| B khủng hoảng (37) | gdec τ=0.95 | 81% | 1.000 | 1.000 | 0 | 461 / 5259 | 0.062 |
| B khủng hoảng (37) | gdec τ=0.98 | 73% | 1.000 | 1.000 | 0 | 482 / 5259 | 0.083 |
| B khủng hoảng (37) | gdec bất đối xứng ε=0.02 | 100% | 1.000 | 1.000 | 0 | 400 / 743 | 0.017 |
| B khủng hoảng (37) | gdec bất đối xứng ε=0.05 | 100% | 1.000 | 1.000 | 0 | 400 / 743 | 0.017 |
| B khủng hoảng (37) | gdec bất đối xứng ε=0.1 | 100% | 1.000 | 1.000 | 0 | 400 / 743 | 0.017 |
| B khủng hoảng (37) | gdec bất đối xứng ε=0.2 | 100% | 1.000 | 1.000 | 0 | 400 / 743 | 0.017 |
| B khủng hoảng (37) | chỉ jev | 100% | 0.838 | 0.838 | 6 | 304 / 558 | 0.016 |
| B khủng hoảng (37) | jev τ=0.5 | 100% | 0.838 | 0.838 | 6 | 304 / 558 | 0.016 |
| B khủng hoảng (37) | jev τ=0.6 | 95% | 0.886 | 0.892 | 4 | 304 / 678 | 0.029 |
| B khủng hoảng (37) | jev τ=0.7 | 89% | 0.909 | 0.919 | 3 | 305 / 4892 | 0.041 |
| B khủng hoảng (37) | jev τ=0.8 | 81% | 0.933 | 0.946 | 2 | 315 / 4892 | 0.062 |
| B khủng hoảng (37) | jev τ=0.9 | 70% | 1.000 | 1.000 | 0 | 315 / 5309 | 0.087 |
| B khủng hoảng (37) | jev τ=0.95 | 65% | 1.000 | 1.000 | 0 | 323 / 5309 | 0.101 |
| B khủng hoảng (37) | jev τ=0.98 | 49% | 1.000 | 1.000 | 0 | 3499 / 5309 | 0.142 |
| B khủng hoảng (37) | jev bất đối xứng ε=0.02 | 81% | 1.000 | 1.000 | 0 | 305 / 4892 | 0.062 |
| B khủng hoảng (37) | jev bất đối xứng ε=0.05 | 84% | 1.000 | 1.000 | 0 | 305 / 4892 | 0.055 |
| B khủng hoảng (37) | jev bất đối xứng ε=0.1 | 84% | 1.000 | 1.000 | 0 | 305 / 4892 | 0.055 |
| B khủng hoảng (37) | jev bất đối xứng ε=0.2 | 89% | 0.939 | 0.946 | 2 | 305 / 4297 | 0.043 |
| C định tuyến (61) | chỉ LLM | 0% | – | 0.951 | – | 4166 / 5865 | 0.390 |
| C định tuyến (61) | chỉ gdec | 100% | 0.951 | 0.951 | – | 426 / 682 | 0.033 |
| C định tuyến (61) | gdec τ=0.5 | 100% | 0.951 | 0.951 | – | 426 / 682 | 0.033 |
| C định tuyến (61) | gdec τ=0.6 | 100% | 0.951 | 0.951 | – | 426 / 682 | 0.033 |
| C định tuyến (61) | gdec τ=0.7 | 93% | 0.965 | 0.951 | – | 427 / 4996 | 0.059 |
| C định tuyến (61) | gdec τ=0.8 | 93% | 0.965 | 0.951 | – | 427 / 4996 | 0.059 |
| C định tuyến (61) | gdec τ=0.9 | 89% | 0.981 | 0.951 | – | 428 / 6100 | 0.078 |
| C định tuyến (61) | gdec τ=0.95 | 84% | 1.000 | 0.951 | – | 441 / 6173 | 0.097 |
| C định tuyến (61) | gdec τ=0.98 | 59% | 1.000 | 0.951 | – | 609 / 6407 | 0.193 |
| C định tuyến (61) | chỉ jev | 100% | 0.967 | 0.967 | – | 293 / 323 | 0.027 |
| C định tuyến (61) | jev τ=0.5 | 100% | 0.967 | 0.967 | – | 293 / 323 | 0.027 |
| C định tuyến (61) | jev τ=0.6 | 100% | 0.967 | 0.967 | – | 293 / 323 | 0.027 |
| C định tuyến (61) | jev τ=0.7 | 100% | 0.967 | 0.967 | – | 293 / 323 | 0.027 |
| C định tuyến (61) | jev τ=0.8 | 97% | 0.983 | 0.967 | – | 293 / 326 | 0.040 |
| C định tuyến (61) | jev τ=0.9 | 93% | 0.982 | 0.951 | – | 295 / 3572 | 0.053 |
| C định tuyến (61) | jev τ=0.95 | 93% | 0.982 | 0.951 | – | 295 / 3572 | 0.053 |
| C định tuyến (61) | jev τ=0.98 | 85% | 1.000 | 0.951 | – | 296 / 5776 | 0.085 |
| D chèn lệnh (12) | chỉ LLM | 0% | – | 0.917 | 0 | 5413 / 7170 | 0.476 |
| D chèn lệnh (12) | chỉ gdec | 100% | 0.917 | 0.917 | 0 | 653 / 820 | 0.046 |
| D chèn lệnh (12) | gdec τ=0.5 | 100% | 0.917 | 0.917 | 0 | 653 / 820 | 0.046 |
| D chèn lệnh (12) | gdec τ=0.6 | 100% | 0.917 | 0.917 | 0 | 653 / 820 | 0.046 |
| D chèn lệnh (12) | gdec τ=0.7 | 100% | 0.917 | 0.917 | 0 | 653 / 820 | 0.046 |
| D chèn lệnh (12) | gdec τ=0.8 | 92% | 0.909 | 0.917 | 0 | 722 / 1177 | 0.087 |
| D chèn lệnh (12) | gdec τ=0.9 | 75% | 0.889 | 0.917 | 0 | 722 / 5142 | 0.170 |
| D chèn lệnh (12) | gdec τ=0.95 | 50% | 1.000 | 0.917 | 0 | 2794 / 7990 | 0.278 |
| D chèn lệnh (12) | gdec τ=0.98 | 25% | 1.000 | 0.917 | 0 | 5606 / 7990 | 0.430 |
| D chèn lệnh (12) | gdec bất đối xứng ε=0.02 | 83% | 0.900 | 0.917 | 0 | 788 / 5869 | 0.168 |
| D chèn lệnh (12) | gdec bất đối xứng ε=0.05 | 100% | 0.917 | 0.917 | 0 | 653 / 820 | 0.046 |
| D chèn lệnh (12) | gdec bất đối xứng ε=0.1 | 100% | 0.917 | 0.917 | 0 | 653 / 820 | 0.046 |
| D chèn lệnh (12) | gdec bất đối xứng ε=0.2 | 100% | 0.917 | 0.917 | 0 | 653 / 820 | 0.046 |
| D chèn lệnh (12) | chỉ jev | 100% | 0.833 | 0.833 | 1 | 322 / 344 | 0.029 |
| D chèn lệnh (12) | jev τ=0.5 | 100% | 0.833 | 0.833 | 1 | 322 / 344 | 0.029 |
| D chèn lệnh (12) | jev τ=0.6 | 100% | 0.833 | 0.833 | 1 | 322 / 344 | 0.029 |
| D chèn lệnh (12) | jev τ=0.7 | 83% | 0.900 | 0.833 | 1 | 339 / 5425 | 0.097 |
| D chèn lệnh (12) | jev τ=0.8 | 67% | 1.000 | 0.917 | 0 | 344 / 7508 | 0.180 |
| D chèn lệnh (12) | jev τ=0.9 | 58% | 1.000 | 0.917 | 0 | 344 / 7508 | 0.221 |
| D chèn lệnh (12) | jev τ=0.95 | 58% | 1.000 | 0.917 | 0 | 344 / 7508 | 0.221 |
| D chèn lệnh (12) | jev τ=0.98 | 42% | 1.000 | 0.917 | 0 | 4592 / 7508 | 0.342 |
| D chèn lệnh (12) | jev bất đối xứng ε=0.02 | 75% | 0.889 | 0.917 | 0 | 334 / 6762 | 0.194 |
| D chèn lệnh (12) | jev bất đối xứng ε=0.05 | 92% | 0.909 | 0.917 | 0 | 322 / 354 | 0.072 |
| D chèn lệnh (12) | jev bất đối xứng ε=0.1 | 92% | 0.909 | 0.917 | 0 | 322 / 354 | 0.072 |
| D chèn lệnh (12) | jev bất đối xứng ε=0.2 | 92% | 0.909 | 0.917 | 0 | 322 / 354 | 0.072 |

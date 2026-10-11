# E1 — tường lửa PII của Threads

Kết quả: **ĐẠT** — recall 0.960 (cần ≥ 0.95), chặn nhầm 0.000 (cần ≤ 0.05), precision 1.000, F1 0.980.
Bộ dữ liệu `e1_dataset.jsonl` sha256 `da27354ae47b0e9c…`; TP 96 · FP 0 · FN 4 · TN 100. Hằng số: PII_PERSONAL_SIM_HIGH=0.78, PII_PERSONAL_SIM_LOW=0.55, RAG_SIM_FLOOR=0.25, RAG_SIM_CEIL=0.65, THREAD_SIMILAR_MIN=0.75.

## Theo tập
| tập | TP | FN | FP | TN | recall | chặn nhầm |
| --- | --- | --- | --- | --- | --- | --- |
| dev | 29 | 1 | 0 | 30 | 0.967 | 0.000 |
| test | 67 | 3 | 0 | 70 | 0.957 | 0.000 |

## Theo nhóm
| nhóm | TP | FN | FP | TN | recall / chặn nhầm |
| --- | --- | --- | --- | --- | --- |
| N1 | 0 | 0 | 0 | 50 | 0.000 |
| N2 | 0 | 0 | 0 | 15 | 0.000 |
| N3 | 0 | 0 | 0 | 15 | 0.000 |
| N4 | 0 | 0 | 0 | 10 | 0.000 |
| N5 | 0 | 0 | 0 | 10 | 0.000 |
| S1 | 12 | 0 | 0 | 0 | 1.000 |
| S2 | 8 | 0 | 0 | 0 | 1.000 |
| S3 | 8 | 0 | 0 | 0 | 1.000 |
| S4 | 6 | 0 | 0 | 0 | 1.000 |
| S5 | 20 | 0 | 0 | 0 | 1.000 |
| S6 | 38 | 0 | 0 | 0 | 1.000 |
| S7 | 0 | 4 | 0 | 0 | 0.000 |
| S8 | 4 | 0 | 0 | 0 | 1.000 |

## Theo loại PII (recall)
| loại | recall | FN |
| --- | --- | --- |
| CCCD | 1.000 | 0 |
| EMAIL | 1.000 | 0 |
| MSSV | 1.000 | 0 |
| NAME | 0.857 | 4 |
| PERSONAL_QUESTION | 1.000 | 0 |
| PHONE | 1.000 | 0 |

## Ma trận nhầm lẫn kênh
| kỳ vọng → dự đoán | số mẫu |
| --- | --- |
| PRIVATE->PRIVATE | 96 |
| PRIVATE->PUBLIC | 4 |
| PUBLIC->PUBLIC | 100 |

## Giới hạn đã biết
- **S7 — tên người ngoài roster, không tín hiệu khác:** tường lửa không có NER (D46) nên vùng này không được bảo vệ. 4/4 mẫu S7 lọt; vẫn tính vào chỉ số chung (tối đa làm recall tụt xuống 0,96).
- Bộ dữ liệu là mô phỏng (D44), soạn bởi Dev và soát độc lập bởi QC (≥ 50 mẫu); không đại diện cách viết của sinh viên thật. Luật chỉ được chỉnh trên tập `dev`.
- Nhúng ở stack seed là `fake` (tất định): bước tương đồng ngữ nghĩa không đóng góp; kết quả phản ánh **luật** (regex + roster + mẫu câu).
- Chỉ đo `allowed` của `precheck`; chưa đo chất lượng câu trả lời có / không che (ghi cho luận văn).

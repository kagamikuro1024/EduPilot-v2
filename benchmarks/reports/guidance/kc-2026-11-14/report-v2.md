# KC106 vòng v2 — BT-A đo, n = 35 bài

Tập đo bỏ 5 bài đã duyệt (kc021, kc022, kc023, kc024, kc025); B0, B1, B2 tính lại trên cùng 35 bài.

| Bản | MAE tổng | MAE tiêu chí | Lệch TB | Tiêu chí phải sửa (|Δ| ≥ 0,5) |
| --- | --- | --- | --- | --- |
| B0 (v0) | 1.521 | 0.416 | +1.521 | 57/140 |
| B1 (v1) | 0.875 | 0.335 | +0.811 | 41/140 |
| B2 (v2) | 0.657 | 0.245 | +0.400 | 26/140 |

| Tiêu chí | MAE B0 | MAE B1 | MAE B2 | nhiễu B2 | so B0 | so B1 |
| --- | --- | --- | --- | --- | --- | --- |
| c1 | 0.332 | 0.246 | 0.229 | 0.029 | ĐẠT | ĐẠT |
| c2 | 0.668 | 0.629 | 0.307 | 0.000 | ĐẠT | ĐẠT |
| c3 | 0.482 | 0.275 | 0.293 | 0.014 | ĐẠT | KHÔNG |
| c4 | 0.182 | 0.189 | 0.150 | 0.000 | ĐẠT | ĐẠT |

Nền nhiễu ε = 0.029. BT-B sau v2 lệch 0.000 (nền nhiễu BT-B 0.062).
Kiểm: V2_not_worse_than_v1 ĐẠT, V2_better_than_v0 ĐẠT, V2_every_criterion_vs_b0 ĐẠT, V2_every_criterion_vs_b1 KHÔNG, BT_B_isolated ĐẠT

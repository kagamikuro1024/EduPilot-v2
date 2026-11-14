# Kiểm chứng «học từ lần sửa» — ĐẠT

Nguồn nhãn: `giaovien` (agent `claude-fable-5-2`, effort medium), chấm mù. Bằng chứng kỹ thuật cho harness, không thay nhãn người thật.

| Chỉ số (BT-A đo, n = 40) | B0 (v0, TB 2 lần) | B1 (v1, TB 2 lần) |
| --- | --- | --- |
| MAE tổng | 1.519 | 0.875 |
| MAE tiêu chí | 0.414 | 0.328 |
| Lệch TB có dấu, tổng (AI − GV) | 1.519 | 0.819 |
| Lệch TB theo tiêu chí | 0.300, 0.666, 0.478, 0.075 | 0.097, 0.628, 0.078, 0.016 |
| Tiêu chí phải sửa tay (Δ tuyệt đối ≥ 0,5) / 160 | 65 | 46 |

Nền nhiễu ε = 0.062. Vốn đúng nay sai: 8 (nhiễu tự nhiên: 8). Bài tốt lên / xấu đi (≥ 0,25): 37 / 1.

BT-B (đối chứng, n = 24): MAE tổng b0r1 1.115, b0r2 1.177, sau bước 7 1.177; lệch cách ly 0.000.

## Tiêu chí chất lượng (SRS §9)
1. MAE tổng B1 ≤ 0,8 × B0 và mức giảm > 2ε: đạt
2. |lệch TB B1| ≤ |lệch TB B0| + 0,20: đạt
3. Vốn đúng nay sai ≤ 1,5 × nhiễu (hoặc ≤ 4 nếu nhiễu = 0): đạt
4. Số tiêu chí phải sửa tay B1 < B0: đạt

## Giới hạn
Bài mô phỏng; một model chấm, một model sinh; một giảng viên là agent; một bài tập học; n = 40. Số tốt lên / xấu đi, vốn đúng nay sai và số sửa tay tính trên điểm AI trung bình của hai lượt chấm.

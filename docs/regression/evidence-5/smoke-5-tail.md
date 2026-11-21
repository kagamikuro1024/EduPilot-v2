
K (a/b/c) = 0; không còn ca S: mọi ca ra P, F hoặc LT. (Dòng «TC-id» ở đầu bảng là dòng mẫu của bảng nguồn, không phải ca.)

**F còn lại (9):** TC-PE04-50, TC-PE05-60, TC-PE06-47, TC-PE08-68 → DEF-R5-18 (golangci-lint ở cmd/regress/main.go:213, commit 658e6719 của dev3); TC-PE09-12 → DEF-R5-20 (seed lần hai in «Seed xong trong 23 s»: ctx.changed = true vô điều kiện, scripts/seed.mjs:843); TC-PE09-13 → DEF-R5-16 (seed sạch 618 s; P-11-76 đặt trần ≤ 6 phút trên máy KHÔNG chạy stack khác — máy QC dùng chung nên điều kiện đo chưa đạt); TC-PE09-25 → DEF-R5-21 (kết quả SV tràn ngang 375/720 px); TC-PE09-20 → DEF-R5-22 (exam.spec.ts thiếu ca cho ý 9, 11 của AC7); TC-PU06-11 → DEF-R5-19 (CLS /chat 0,16).

**Đã chạy lại sau sửa của dev:** R5-03/04/06/07/08/09/10/11 (qc2, 2026-11-18); R5-12 (lint spec QC chuyển specs-qc2), R5-13 (9754c523), R5-14 (d27b87b0: TLE, 30/30), R5-17 (58448e0a, audit 4 vai 115 hàng FAIL 0) — qc3, 2026-11-21; R5-15 (d0d548e0) sửa chưa đủ → R5-20.

**k6 (qc3, stack qc3s, 2026-11-21):** PE02-48/49, PE05-25, PE09-15/16/17/19 đều P; số đo trong ghi chú từng ca.

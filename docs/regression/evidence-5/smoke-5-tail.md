
K (a/b/c) = 0. Ca chưa chạy (N): TC-PE09-12 (dev đã sửa DEF-R5-20 = 581dbccd, chờ cửa sổ SEED để chạy lại: seed lần hai phải in «Seed xong (không đổi)») và TC-PE09-13 (DEF-R5-16 = P-11-76: trần ≤ 6 phút đo trên máy KHÔNG chạy stack khác — chờ cửa sổ SEED cuối). Dòng «TC-id» ở đầu bảng là dòng mẫu của bảng nguồn, không phải ca.

**F còn lại (1):** TC-PE09-25 → DEF-R5-23 (sau sửa R5-21 a0d98671 hết tràn ngang nhưng <pre> «Mã của bạn» cuộn được mà không focus được bằng bàn phím: axe serious scrollable-region-focusable ở 375/720 px).

**Đã chạy lại sau sửa của dev (qc3, 2026-11-21):** R5-12 (lint spec QC chuyển specs-qc2), R5-13 (9754c523), R5-14 (d27b87b0: TLE, 30/30), R5-17 (58448e0a: audit 4 vai FAIL 0), R5-18 (39887d6e: golangci-lint 0 issues; 4 ca cổng tổng P), R5-19 (e345ef3a: CLS 0,0002), R5-22 (d0dcf4fe: exam.spec 118 ca 0 failed); R5-21 (a0d98671) sửa tràn ngang nhưng lộ R5-23; R5-15 (d0d548e0) sửa chưa đủ → R5-20 (581dbccd).

**k6 (qc3, stack qc3s, 2026-11-21):** PE02-48/49, PE05-25, PE09-15/16/17/19 đều P; số đo trong ghi chú từng ca.

# QC test case — US-NFR-05 `EXPLAIN ANALYZE` 10 truy vấn nặng + chỉ mục `00018_indexes_tuning`
Nguồn: US-NFR-05 + SRS FR-9–10, §5. Cần DB T1 (US-NFR-03) và kịch bản (d). `pg_stat_statements` chỉ ở stack đo (SRS §5).

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-N05-01 | AC1 | `docs/research/2026-11-xx-explain.md`, `benchmarks/reports/explain.json` | Đọc báo cáo | Đúng **10** truy vấn, mỗi cái: văn bản truy vấn (đã chuẩn hoá, không PII), kế hoạch **trước**, số dòng, thời gian; nêu nguồn xếp hạng «nặng nhất» (`pg_stat_statements` hoặc log chậm sau (d)) và xếp theo tổng hay trung bình thời gian (Q-N-11) | FILE |
| TC-N05-02 | AC1 | T1, stack đo có `pg_stat_statements` | Sau khi chạy (d) (hoặc kịch bản nhẹ), QC lấy top 10: `SELECT queryid, calls, total_exec_time, mean_exec_time, left(query,80) FROM pg_stat_statements ORDER BY total_exec_time DESC LIMIT 10` | ≥ 7/10 truy vấn của QC có trong danh sách của dev (độc lập chọn; chênh quá nhiều → hỏi dev cách chọn; Q-N-11) | RERUN |
| TC-N05-03 | AC2 | repo | `git diff --stat origin/main...HEAD -- backend-go/db/migrations`; `ls backend-go/db/migrations \| tail -5` | Chỉ **thêm** `00018_indexes_tuning.sql`; **không sửa** migration đã merge (`git diff … --diff-filter=M` rỗng); không có `00018_*` khác | DIFF |
| TC-N05-04 | AC2 | DB T1 sau `goose up` | `SELECT indexname, indexdef FROM pg_indexes WHERE indexname IN (<chỉ mục của 00018>)` (liệt kê từ `grep CREATE INDEX` trong 00018) | Mọi chỉ mục **phức hợp** bắt đầu bằng `course_id` (luật 13); chỉ mục không phức hợp được giải thích (ví dụ HNSW) | RUN |
| TC-N05-05 | AC2 | – | `grep -n -i 'hnsw' 00018…sql`; báo cáo | HNSW cho `content_chunks.embedding` **chỉ khi** báo cáo có kế hoạch cho thấy tìm vector đang quét tuần tự (Seq Scan) trước đó; không có bằng chứng → không được tạo HNSW | FILE |
| TC-N05-06 | AC2 | – | Mỗi chỉ mục: có đúng **một dòng lý do** trong báo cáo (tên chỉ mục → truy vấn nào cần) | Đủ; chỉ mục không có lý do = FAIL | FILE |
| TC-N05-07 | AC3 | DB T1 **trước** 00018 và **sau** (hai bản sao hoặc `down`/`up`) | Chạy lại đúng 10 truy vấn của báo cáo bằng `EXPLAIN (ANALYZE, BUFFERS)`: 5 lần/truy vấn, lấy trung vị, cache ấm | Mỗi truy vấn: nhanh hơn, hoặc «không đổi, không cần chỉ mục» có lý do; **không** truy vấn nào chậm hơn > 10% (so trung vị) (dung sai nhiễu: Q-N-12); bảng trước/sau của QC khớp bảng dev ±25% | RERUN |
| TC-N05-08 | AC3 | – | Với truy vấn dùng chỉ mục mới | `EXPLAIN` thật dùng `Index Scan`/`Index Only Scan` trên chỉ mục mới (không `Seq Scan` trên bảng ≥ 100k dòng); chỉ mục mới không bị bỏ không dùng (`pg_stat_user_indexes.idx_scan > 0` sau chạy (d)) | RERUN |
| TC-N05-09 | AC4 | DB T1 | `goose up; goose down 1; goose up` (đúng `-dir db/migrations`); ghi mã thoát từng bước | Cả ba thành công; sau `down 1` các chỉ mục 00018 biến mất, sau `up` trở lại; `CONCURRENTLY`/`NO TRANSACTION` đúng (nếu dùng) không để lại chỉ mục `INVALID` (`SELECT … FROM pg_index WHERE NOT indisvalid` rỗng) | RUN |
| TC-N05-10 | AC4 | repo | `cd backend-go && sqlc diff; echo $?`; `go test ./...` (hoặc gói đụng truy vấn do dev liệt kê) | `sqlc diff` rỗng; test xanh | RUN |
| TC-N05-11 | AC5 | DB T1 | Đo ghi: 200 lần chèn điểm danh và 200 lần chèn bài nộp (trong transaction + rollback, hoặc qua API) **trước** và **sau** 00018, trung vị | Mỗi thao tác ghi chậm hơn ≤ 20%; chỉ mục nào làm chậm > 20% mà **vẫn còn** trong 00018 = FAIL; chỉ mục đã bị bỏ có dòng «bỏ» + lý do trong báo cáo | RERUN |
| TC-N05-12 | AC6 | DB T1 | `SELECT pg_size_pretty(pg_relation_size(indexrelid))` cho chỉ mục 00018; so `pg_relation_size` bảng chính | Tổng MB ghi trong báo cáo khớp số QC đo ±10%; nếu một chỉ mục / tổng > 25% kích thước bảng chính thì `QUESTIONS.md` / `proposals.md` **có dòng giải thích** | FILE + RUN |
| TC-N05-13 | Luật 13 | – | Đọc 10 truy vấn: dấu hiệu N+1 / OFFSET / không phân trang | Truy vấn nào vi phạm luật 13 được ghi thành lỗi (US-NFR-10) chứ không chỉ chỉ mục che | FILE |
| TC-N05-14 | AC2 | – | `00018` không chứa `ALTER TABLE`, `DROP`, `CREATE EXTENSION pg_stat_statements` (SRS §5: chỉ stack đo) | Chỉ `CREATE INDEX` (+ `IF NOT EXISTS`); không đổi cột/bảng; extension không nằm trong migration sản phẩm | DIFF |

## Phân quyền
Không có route mới (không áp dụng). Kiểm chéo: sau `00018`, TC-N03-09 (chéo lớp) vẫn 403/404.

## Lịch sử sửa TC
(trống)

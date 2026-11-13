# QC test case — US-NFR-10 Sửa lỗi mà phép đo tìm ra, trong ngân sách
Nguồn: US-NFR-10 + SRS FR-17, Q-NFR-6 (lỗi **cao** sửa ngay, tối đa **5** story sửa; vừa/thấp → Nợ). Đầu vào: `docs/sprints/9.5/techlead.md` (TL-n), `proposals.md`, các báo cáo. Chạy cuối sprint, trước khi PM đóng.

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-N10-01 | AC1 | `techlead.md`/`proposals.md` | Lập danh sách mọi lỗi: (a) mọi FAIL / lỗi trong `benchmarks/reports/**` và `docs/research/…nfr…`; (b) mọi dòng TL-n / P-95-n mới. Đối chiếu hai chiều | Mỗi lỗi ở (a) có dòng ở (b) và ngược lại; mỗi dòng có: **tái hiện (lệnh)**, mức (cao/vừa/thấp), đề xuất, **bằng chứng là tệp có thật** trong `benchmarks/reports/` (`test -f`) | FILE |
| TC-N10-02 | AC1 | – | Chạy lệnh tái hiện của 2 lỗi ngẫu nhiên (ở commit **trước** sửa nếu đã sửa: `git worktree add /tmp/pre <commit_cha>`) | Lệnh chạy được; tái hiện đúng hiện tượng nêu | RERUN |
| TC-N10-03 | AC2 | Danh sách lỗi mức cao (rò đáp án / điểm / PII; chèn lệnh kéo điểm mà không cờ; việc thất lạc; TTFT/5xx trượt) | Với mỗi lỗi cao: tìm commit `9.5: <việc>` + số TL, test hồi quy, và hai lần đo (trước/sau) trong báo cáo | Cả ba có đủ; PM duyệt trước khi sửa (có ghi trong `proposals.md`/`techlead.md`) | FILE |
| TC-N10-04 | AC2 | Commit sửa `S`, cha `P`, test hồi quy `T` (tên test do dev nêu ở handoff/commit; Q-N-23) | `git worktree add /tmp/pre P`; chép **chỉ** `T` vào; chạy `T` ở `P`, rồi ở `S` | `T` **FAIL ở `P`**, **PASS ở `S`** (test có răng); test không phân biệt trước/sau = FAIL | RERUN |
| TC-N10-05 | AC2 | – | Chạy lại phép đo liên quan lỗi cao (kịch bản/chaos/EXPLAIN tương ứng, bản nhẹ) | PASS sau sửa; báo cáo ghi **cả hai lần** (trước trượt, sau đạt); không xoá số lần trượt | RERUN + FILE |
| TC-N10-06 | AC3 | Lỗi vừa/thấp | `git diff origin/main...HEAD --stat -- backend-go frontend` đối chiếu: không có sửa mã cho lỗi vừa/thấp; `docs/PROGRESS.md` mục Nợ (hoặc sprint sau) có dòng của từng lỗi vừa/thấp | Khớp; sửa mã vừa/thấp trong 9.5 mà không có quyết định PM = FAIL | DIFF |
| TC-N10-07 | AC4 | commit trên nhánh | `git log --format='%s' origin/main..HEAD \| grep -v '^9\.5:'`; mỗi commit sửa mã sản phẩm có số TL / proposal | Mọi commit sửa mã là `9.5: <việc>` + TL/proposal | DIFF |
| TC-N10-08 | AC4 | – | `git diff origin/main...HEAD -- backend-go/api/openapi.yaml`; `git diff … --diff-filter=M -- backend-go/db/migrations`; `--diff-filter=A` | `openapi.yaml` **không đổi hợp đồng** (không thêm/sửa/xoá endpoint, không đổi response); migration **chỉ thêm** `00018_indexes_tuning.sql`, không sửa migration cũ | DIFF |
| TC-N10-09 | AC5 | Mỗi gói Go đã sửa | `cd backend-go && go test -race <các gói> 2>&1 \| tail`; `pnpm -C frontend test` / playwright liên quan | Xanh; **cả bộ** `go test -race ./...` và `playwright test` chạy **một lần** trước handoff (kết quả trong handoff; QC chạy lại cả hai một lần) | RUN |
| TC-N10-10 | AC5 | – | `git diff origin/main...HEAD -- '**/*_test.go' 'frontend/e2e/**' 'frontend/**/*.test.*'` | Không sửa/bỏ/nới test có sẵn để xanh (`t.Skip`, đổi kỳ vọng, nới ngưỡng); chỉ **thêm** test hồi quy; sửa test có sẵn = báo PM lỗi nghiêm trọng | DIFF |
| TC-N10-11 | AC6 | SLO không đạt bằng sửa nhỏ | Đọc `techlead.md`/`proposals.md` | Có dòng «dừng, ghi PM» thay vì thêm hạ tầng; `git diff origin/main...HEAD -- docker-compose*.yml go.mod go.sum frontend/package.json pnpm-lock.yaml`: không thêm service (Kafka, K8s, shard…), không thư viện mới (hoặc có hỏi PM) | DIFF |
| TC-N10-12 | Q-NFR-6 | – | Đếm số story sửa lỗi cao | ≤ 5; lỗi cao nào còn mở phải có quyết định hoãn bằng văn bản của PM | FILE |
| TC-N10-13 | Cấm | – | Tìm: `float64` mới cho điểm, `fetch(` trần, secret, PII log trong diff sửa lỗi | Không có | DIFF |
| TC-N10-14 | Phi hồi quy | Sau mọi sửa | `bash scripts/ui-antipatterns.sh` (nếu đụng frontend); `go vet ./...`; TC-N03-09 / TC-N04-07 (chéo lớp, 403) lại | Sạch; phân quyền không hỏng sau sửa | RUN |

## Phân quyền
Kiểm chéo sau sửa: TC-N10-14.

## Lịch sử sửa TC
(trống)

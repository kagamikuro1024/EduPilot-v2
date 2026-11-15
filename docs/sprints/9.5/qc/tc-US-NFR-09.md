# QC test case — US-NFR-09 Báo cáo tổng hợp và bảng số liệu duy nhất
Nguồn: US-NFR-09 + SRS FR-16, §8. Tệp: `docs/research/2026-11-xx-nfr-real-llm.md` (tên ngày thật), `benchmarks/reports/**`. QC kiểm bằng cách **truy nguồn từng con số** và đọc theo yêu cầu cấu trúc. Chạy sau khi NFR-02…08 xong.

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-N09-01 | AC1 | Báo cáo tồn tại | `ls docs/research/*nfr-real-llm*.md`; đọc mục lục | Tệp tên có ngày thật (không còn `xx`); mỗi phép đo (NFR-02…08) có mục riêng gồm 6 phần: câu hỏi · phương pháp · cấu hình · số liệu · PASS/FAIL · giới hạn | FILE |
| TC-N09-02 | AC1 | – | Tìm bảng tổng «SLO → số đo → kết quả» | Có; **mọi** SLO của `SYSTEM_DESIGN` §5 (đọc p95, ghi p95, SSE đầu, TTFT hai nhóm, chuông ≤ 5 s nếu đo, dashboard/360, chấm 1.000 bài + TTFT +20%, 0 việc thất lạc, 5xx) có một hàng; kết quả PASS/FAIL/«chưa đo» (QC tính lại nhãn từ số so ngưỡng) | FILE |
| TC-N09-03 | AC1 | – | Tìm mục «chưa đo được và vì sao» | Có đủ: judge amd64 có seccomp (D58), E3 chấm tay, nhiều provider (E4), mạng thật; mỗi cái một lý do | FILE |
| TC-N09-04 | AC2 | Báo cáo + `benchmarks/reports/**` | Script QC `docs/sprints/9.5/qc/scripts/trace-numbers.mjs`: trích mọi số có đơn vị (ms, s, %, MB, GB, h, QPS, điểm) từ báo cáo, tìm giá trị tương ứng trong các `.json`/`.md` của `benchmarks/reports/` (làm tròn theo số chữ số báo cáo) | 100% số **đo được** có nguồn; số không khớp → danh sách; số suy ra (tỉ số, hiệu, trung bình) phải ghi công thức + nguồn các toán hạng (P-95-7 Q22); số chỉ nằm trong văn xuôi = FAIL | FILE |
| TC-N09-05 | AC2 | – | Lấy ngẫu nhiên 15 số trong bảng tổng; mở tệp nguồn nêu trong báo cáo | 15/15 đúng (khớp trong giới hạn làm tròn); đường dẫn tệp nguồn trỏ tới tệp **có thật** (`test -f`) | FILE |
| TC-N09-06 | AC2 | – | `grep -nEi 'ước tính|ngoại suy|tương đương|khoảng .* (nếu|giả sử)|extrapolat'` | Mỗi lần xuất hiện là nêu rõ «không ngoại suy» hoặc phần ghi «chưa hoàn tất»; không số nào là kết quả ngoại suy trình bày như đo | SCAN |
| TC-N09-07 | AC3 | `docs/sprints/8/report.md`, `docs/sprints/9/report.md`, `docs/PROGRESS.md` | `grep -nEi 'giả lập|chưa đo|chưa test LLM thật' …`; tìm tài liệu «câu cần vá» (trong báo cáo hoặc `docs/sprints/9.5/…`, gửi PM) | Danh sách «câu cần vá» có: vị trí (tệp:dòng) → câu mới, cho từng chỗ tìm thấy (đối chiếu `grep`: số vị trí QC tìm = số trong danh sách; thiếu vị trí = FAIL) | FILE |
| TC-N09-08 | AC3 | – | `git diff --stat origin/main...HEAD -- docs/sprints/8/report.md docs/sprints/9/report.md docs/PROGRESS.md` | Không sửa 3 tệp đó (BA/PM không sửa báo cáo cũ nếu chưa có proposal); có `proposals.md` nếu sửa | DIFF |
| TC-N09-09 | AC4 | Phép đo nào không hoàn tất (ví dụ hết hạn mức, NFR-04 AC9 chưa xong) | Đọc mục của phép đo đó | Ghi «chưa hoàn tất» + lý do; không ô trống, không «TBD», không số ngoại suy; `grep -nE 'TBD|TODO|\.\.\.|\?\?'` trong bảng số: 0 | SCAN |
| TC-N09-10 | AC5 | – | Đếm: `wc -w` mục tóm tắt (≤ 3 trang ≈ ≤ 1.500 từ, P-95-7 Q21); tiêu đề trang/nhãn; tiếng Việt; bảng; đoạn log nguyên văn (khối code > 20 dòng) | Mục tóm tắt ≤ 3 trang; đọc được trong ≤ 5 phút (QC đo thời gian đọc thật bằng đồng hồ, ghi); tiếng Việt có dấu; có bảng; **không** chép nguyên log | FILE |
| TC-N09-11 | AC1 | – | Mỗi mục: «giới hạn» nêu hạn chế Mac (k6 + stack chung 6 vCPU → cận trên bảo thủ [INFERENCE chưa kiểm]; loopback → p95 là cận dưới) và câu «đo trên MacBook Apple Silicon, Colima 6 CPU / 10 GB» | Có ở mọi mục có số | SCAN |
| TC-N09-12 | Quy ước | – | `grep -rEn 'AIza|Bearer |sk-'` báo cáo; tên/MSSV/email thật | 0 | SCAN |
| TC-N09-13 | Trung thực | – | Mọi SLO FAIL ở `load-*`, `baseline`, `chaos` xuất hiện **đúng** ở bảng tổng với số thật (không lọc bỏ lần trượt, không chỉ ghi lần tốt nhất) | Khớp 100% (so từng FAIL từ tệp nguồn với bảng tổng) | FILE |

## Phân quyền
Không áp dụng (tài liệu). Bí mật/PII: TC-N09-12.

## Lịch sử sửa TC
- 2026-11-14 — PM trả lời P-95-7 (Q-N-1…23 chấp nhận theo đề xuất QC): Q21 (≤ 1.500 từ), Q22 (số suy ra cần công thức).

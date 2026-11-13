# QC test case — US-NFR-04 Đo tải k6 theo SLO ở quy mô T1 (×1 và ×2)
Nguồn: US-NFR-04 + SRS FR-6–8, §3, §6 (giới hạn tần suất), §8, `SYSTEM_DESIGN` §5. QC **không đo lại cả bốn kịch bản** (SRS §9: ≥ 1 kịch bản). Đề xuất QC chạy lại: **(d)** (dashboard/360/sổ điểm trên T1, nhẹ nhất, ×1) và bước 403 của AC4; chọn khác do PM. Ngưỡng SLO (`SYSTEM_DESIGN` §5): đọc p95 ≤ 300 ms · ghi p95 ≤ 500 ms · SSE đầu ≤ 300 ms · TTFT p95 ≤ 1,5 s (cache hit) / ≤ 4 s (RAG) · dashboard / 360 ≤ 3 s trên 1.000 SV · chấm 1.000 bài ≤ 3 giờ **và** TTFT không xấu > 20% · 5xx < 0,5% · 0 việc thất lạc.

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-N04-01 | AC1 | `benchmarks/reports/` | `ls benchmarks/reports/load-{a,b,c,d}.json`; `node benchmarks/check-meta.mjs` mỗi tệp | Đủ 4 tệp; `meta` hợp lệ; `k6_cpus` = 1,5; `demo_stack` ∈ {stopped, running} | FILE |
| TC-N04-02 | AC1 | – | Mỗi tệp: `jq` các dòng SLO (p50/p95/p99, tỉ lệ 5xx, PASS/FAIL); lập bảng kịch bản × SLO | Mỗi kịch bản có p50 ≤ p95 ≤ p99 (số hợp lý); mỗi dòng SLO của §5 liên quan **có** PASS/FAIL; (a): 300 người dùng + 300 SSE thông báo; (b): chat 1 QPS đỉnh 5 QPS; (c): chấm 1.000 bài song song chat; (d): dashboard/360/sổ điểm 1.000 SV | FILE |
| TC-N04-03 | AC1 | – | **Tính lại** PASS/FAIL từ số: so p95 đọc với 300, ghi với 500, SSE đầu 300, TTFT 1.500/4.000, 5xx 0,5%, dashboard 3.000 | Mọi nhãn PASS/FAIL trong báo cáo **khớp** phép so sánh số với ngưỡng (nhãn sai = FAIL) | FILE |
| TC-N04-04 | AC1, AC6 | – | `grep -n 'threshold\|thresholds' benchmarks/load/*.js`; đối chiếu với bảng §5 ở trên; `git log -p --follow benchmarks/load/*.js` ngưỡng | Ngưỡng trong k6 = SLO §5, **không nới** qua các commit (không có diff nới ngưỡng) | DIFF |
| TC-N04-05 | AC2 | `load-c.json` | Đọc hai pha `baseline` và `grading` (cùng lần chạy, cùng `commit`, thời điểm liền nhau); tính `TTFT_p95(grading) / TTFT_p95(baseline) − 1` | ≤ 20% hoặc FAIL kèm số thật; cả hai pha có số liệu trong tệp (không chỉ pha xấu) | FILE |
| TC-N04-06 | AC3 | `load-a/b/c.json` | Mỗi tệp có hai cột ×1 và ×2 (`--scale gateway=2 --scale worker=2`); so `meta.commit` và `meta.cấu hình` hai cột | Cùng `commit`; số bản gateway/worker khác nhau đúng (1/1 và 2/2); kịch bản giống nhau; không commit nào giữa hai cột đổi mã sản phẩm (`git log <commit_x1>..<commit_x2> -- backend-go frontend` rỗng) | FILE + DIFF |
| TC-N04-07 | AC4 | T1 + k6 đang chạy (kịch bản (d) hoặc (b)) | Trong lúc tải: vòng lặp QC 100 lần `GET /courses/{C}/gradebook` bởi STUDENT (một SV lớp C); đếm mã | **100/100 `403`**, 0 `200`; ghi `time` đỉnh; (đồng thời k6 báo bước kiểm `403` ≥ 100 lần trong `load-*.json`, số lần `200` = 0) | RERUN |
| TC-N04-08 | AC4 | – | Lặp với `GET …/grade-items`, `POST …/questions` (SV) | `403` toàn bộ | RERUN |
| TC-N04-09 | AC5 | `load-b.json` | Đọc sự kiện SSE đầu tiên và TTFT, hai nhóm riêng: cache hit / RAG | Sự kiện đầu p95 ≤ 300 ms; TTFT p95 ≤ 1,5 s (cache hit) và ≤ 4 s (RAG), **ghi riêng hai nhóm**; hoặc FAIL kèm số | FILE |
| TC-N04-10 | AC6 | Mọi SLO FAIL trong báo cáo | Với mỗi FAIL: có số thật + lần thử lại (nếu có) + nghi ngờ nguyên nhân (CPU chung 6 vCPU, RAM) | Có đủ ba thứ; **ngưỡng không đổi**; không chỉ ghi lần tốt nhất: `số lần chạy` > 1 thì JSON có kết quả **từng lần** (kể cả lần trượt) | FILE |
| TC-N04-11 | AC6 | – | Lỗi hiệu năng do mã (N+1, thiếu chỉ mục) | Có dòng chuyển sang US-NFR-10 (`techlead.md` TL-n) với tệp chứng cứ | FILE |
| TC-N04-12 | AC7 | `load-a/c.json` | Có mục `docker stats` mẫu hoặc sự kiện OOM / CPU ≥ 95% suốt ≥ 60 s | Nếu có sự kiện: lượt bị dừng và ghi; không chạy hai kịch bản cùng lúc (`thời điểm` các lượt không chồng lấp); nếu hạ quy mô, chỉ theo Q-NFR-3 (OOM thật) | FILE |
| TC-N04-13 | AC8 | QC chạy lại 1 kịch bản | Chạy k6 theo README: `docker run --rm --network <mạng project ep-qc> --cpus 1.5 --memory 1.5g grafana/k6 run …`; trong lúc chạy: `docker inspect <k6> --format '{{.HostConfig.NanoCpus}} {{.HostConfig.Memory}} {{.HostConfig.NetworkMode}}'` | `1500000000 1610612736` (hoặc tương đương 1,5 GiB), mạng = mạng compose của `ep-qc`; **không** bắn từ macOS qua cổng chuyển tiếp Colima (không dùng `localhost:943` từ host) | RERUN |
| TC-N04-14 | AC8 | `load-*.json` | Đọc `meta.demo_stack`; đối chiếu với bằng chứng dừng/chạy lại stack demo (`docker compose ls` / ghi chú trong báo cáo; Q-N-8) | `stopped` chỉ khi có bằng chứng `docker compose -p edupilot stop` (không `down`, không xoá volume) rồi `start` lại; không bằng chứng → phải là `running`; stack demo **còn nguyên volume** sau đó (QC không động vào) | FILE |
| TC-N04-15 | AC8 | Báo cáo | `grep -in 'cận dưới\|loopback\|6 vCPU\|chung' benchmarks/reports/load-*.md` | Có nêu hạn chế: mạng loopback trong VM nên p95 là **cận dưới**, k6 và stack chung 6 vCPU | FILE |
| TC-N04-16 | AC9 | `load-c.json` | Đọc `start/end`, số bài chấm, độ trễ `fake`, số luồng BATCH | Thời gian thật ≈ 2,2 h (khoảng 1,8–2,8 h; Q-N-10), 5 luồng BATCH, trễ `fake` ≈ 20 s, **1.000** bài đã chấm; **không ngoại suy** (nếu báo cáo ghi «ước tính từ mẫu 100 bài» → FAIL) | FILE |
| TC-N04-17 | AC9 | – | Lượt LLM thật `VUS=4` trên BT03 | Ghi rõ «mẫu nhỏ» và thuộc US-NFR-08 AC7 (kiểm ở NFR-08) | FILE |
| TC-N04-18 | AC1 | QC chạy lại kịch bản (d) | Dựng stack T1 (giữ DB T1 nếu còn), chạy (d) bằng container k6 (TC-N04-13), cùng `k6-stack.yml` | Ra `load-d.qc.json` có `meta` hợp lệ; so từng dòng SLO với `load-d.json`: cùng PASS/FAIL; p95 mỗi dòng trong ±30% hoặc nhỏ hơn (Q-N-7); khác → ghi, không tự kết luận | RERUN |
| TC-N04-19 | SRS §6 | – | `benchmarks/load/k6-stack.yml` nới hạn mức IP/người, `LLM_MAX_CONCURRENCY`, `FAKE_LLM_LATENCY=5000-15000` | Báo cáo **nêu giá trị đã nới**; override chỉ áp cho stack đo (không nằm trong `docker-compose.local.yml` / prod: `grep` 0) | FILE + SCAN |
| TC-N04-20 | Quy ước | – | `jq`/`grep` mọi báo cáo `load-*`: `Authorization`, `Bearer `, email thật | 0 token / khoá / PII | SCAN |

## Phân quyền
TC-N04-07, 08.

## Lịch sử sửa TC
(trống)

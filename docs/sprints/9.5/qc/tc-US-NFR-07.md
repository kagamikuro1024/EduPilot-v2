# QC test case — US-NFR-07 Số đo nền của bản Go trên Mac (RAM, khởi động, image, p95, Lighthouse)
Nguồn: US-NFR-07 + SRS FR-12–13, §8. Ngưỡng hiện hành: đọc p95 ≤ 300 ms, ghi p95 ≤ 500 ms; Lighthouse LCP ≤ 2.500 ms (devtools), TBT ≤ 200 ms (mô phỏng), script ≤ 256 KB. Mọi số ghi «MacBook Apple Silicon, Colima 6 CPU / 10 GB».

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-N07-01 | AC1 | `benchmarks/reports/baseline.{json,md}` | `node benchmarks/check-meta.mjs baseline.json`; đọc bảng RAM/CPU nghỉ | Có hàng cho: gateway, worker, postgres, pgbouncer, redis, minio, docling-serve (nếu bật), caddy, frontend; tổng RAM = tổng các hàng (tính lại); đo sau 5 phút nghỉ | FILE |
| TC-N07-02 | AC1 | Stack T0 `ep-qc` nghỉ 5 phút | `docker stats --no-stream --format '{{.Name}} {{.MemUsage}} {{.CPUPerc}}'` | RAM từng container và tổng khớp bảng dev trong ±30% (P-95-7 Q13); CPU nghỉ thấp (≈ 0–5%) | RERUN |
| TC-N07-03 | AC2 | Image đã build | `docker images --format '{{.Repository}} {{.Size}}'` cho gateway, worker, frontend | Kích thước báo cáo (MB) khớp ±5% | RERUN |
| TC-N07-04 | AC2 | `docker compose -p ep-qc down` (giữ image) | `time` từ `up -d` đến `GET /api/v1/readyz` = 200; 3 lần lạnh, trung vị | Báo cáo có 3 số + trung vị; số QC trong ±30% | RERUN |
| TC-N07-05 | AC3 | T0 | Báo cáo: ≥ **12** endpoint đại diện: đọc (Hôm nay, danh sách thread, sổ điểm, hồ sơ 360, thư viện) và ghi (điểm danh, gửi chat không LLM, nộp bài) × **200** lần tuần tự; p50/p95 từng cái | Đủ ≥ 12 endpoint cả đọc lẫn ghi, mỗi cái 200 mẫu; p95 đọc ≤ 300, ghi ≤ 500, hoặc **FAIL kèm số** (không bỏ endpoint trượt khỏi bảng) | FILE |
| TC-N07-06 | AC3 | – | QC chạy lại 3 endpoint (một đọc: Hôm nay; một đọc nặng: sổ điểm; một ghi: điểm danh) × 200 lần tuần tự bằng `curl`/k6 trong VM | p95 QC cùng PASS/FAIL với báo cáo; chênh ≤ ±30% (P-95-7 Q7) | RERUN |
| TC-N07-07 | AC4 | Frontend `ep-qc` | `docker inspect <frontend> --format '{{.Config.Cmd}} {{.Config.Entrypoint}}'`; `curl -sI /` ; `ls .next/BUILD_ID` trong container | **Bản build production** (`next start`/standalone), **không** `next dev` (không có HMR `/_next/webpack-hmr`) | RUN |
| TC-N07-08 | AC4 | – | Báo cáo Lighthouse: 6 route của `lighthouserc*.json` × (mô phỏng, devtools): LCP, TBT, script size; `/assignments/[id]` có **dòng riêng** (nợ G7-23, LCP 4,7 s) | Đủ 6 route × 2 chế độ; số đối chiếu ngưỡng; route trượt ghi rõ; ngưỡng **không nới** (`git diff origin/main -- lighthouserc*.json`: không đổi ngưỡng) | FILE + DIFF |
| TC-N07-09 | AC4 | Build production | QC chạy lại Lighthouse cho 2 route (`/` và `/assignments/[id]`) | Số trong phạm vi hợp lý: PASS/FAIL cùng nhãn với báo cáo; `/assignments/[id]` ghi số thật | RERUN |
| TC-N07-10 | AC5 | – | Báo cáo | Có ghi các lần **bị nhiễu** (CPU nền > 30%) và đã đo lại sau khi tắt tiến trình nền; không loại lần xấu mà không ghi | FILE |
| TC-N07-11 | AC6 | Báo cáo + `docs/research/…` | `grep -nEi 'VPS|legacy|Project III|hệ cũ|so với (bản|hệ) cũ'`; `grep -c 'MacBook Apple Silicon, Colima 6 CPU / 10 GB'` | Không đối chiếu với hệ cũ (D45) hay VPS cũ; câu «đo trên MacBook Apple Silicon, Colima 6 CPU / 10 GB» có mặt | SCAN |
| TC-N07-12 | Ngoài phạm vi | – | Tìm số đo judge amd64 / seccomp trên Mac | **Không** có số đo judge seccomp trên arm64 (D58); có dòng «không đo được, chuyển PR» (Q-NFR-4) | FILE |

## Phân quyền
Không có route mới.

## Lịch sử sửa TC
- 2026-11-14 — PM trả lời P-95-7 (Q-N-1…23 chấp nhận theo đề xuất QC): Q7, Q13 (±30%).

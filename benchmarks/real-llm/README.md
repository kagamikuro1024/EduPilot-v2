# Đo LLM thật (Gemini Free Tier, sprint 9.5)

Script `real_llm.py` là client ngoài: chỉ gọi HTTP API của gateway, chỉ dùng thư viện chuẩn của Python. Mô hình đo là mô hình stack đang cấu hình (`gemini-chat` + `gemini-embed`, chuỗi model chọn đầu phiên bằng `PHASE=probe node deploy/vps-llm.mjs`).
- Kết quả ghi vào `benchmarks/reports/real-llm/*.json` (có khối `meta`). Báo cáo: `docs/research/2026-11-xx-nfr-real-llm.md`.
- Dữ liệu gửi Gemini Free (nội dung có thể được Google dùng để cải thiện sản phẩm) chỉ gồm dữ liệu mô phỏng (D44) đã che tên / MSSV.

**Chạy trên Mac, stack `-p ep-research`, cổng lệch (`EP_PORT_OFFSET`), chỉ bind `127.0.0.1`; không dùng stack demo.** Khoá `GEMINI_API_KEY` chỉ ở `.env.local` của stack đo (không in, không commit). Hạn mức Gemini Free theo project, đặt lại 15:00 giờ Việt Nam: < 200 lời gọi sinh chữ mỗi ngày, chia hai ngày; hết hạn mức thì `--resume` sau 15:00.

Dựng stack như `benchmarks/README.md` (project `ep-research`, `EP_PORT_OFFSET=300`), `PHASE=seed` → `node scripts/seed.mjs` (cần `--profile ingest`) → `PHASE=live`.

**Trước khi `down -v` stack đo** (P-95-7 Q20) bắt buộc: `node benchmarks/real-llm/export-evidence.mjs` — gom kết quả thô, `llm_audit` tổng hợp (không payload) và ≥ 15 mẫu che danh tính vào `benchmarks/reports/real-llm/`; lệnh thoát mã ≠ 0 nếu thiếu.

```bash
export BASE=https://localhost:<443+offset của ep-research> NODE_TLS_REJECT_UNAUTHORIZED=0
# đọc llm_audit / chat_messages; CHỈ ĐỌC (script tự bọc mỗi truy vấn trong `begin read only`)
export AUDIT_PSQL="docker exec -i ep-research-postgres-1 psql -U edupilot -d edupilot -tA -v ON_ERROR_STOP=1"
cd benchmarks/real-llm
python3 real_llm.py setup                       # lớp đo EVALLLM + 2 quy chế + 12 sinh viên seed (không phải A, B, C)
python3 real_llm.py extract --runs 5            # mục 1
python3 real_llm.py grade-setup                 # mục 2: EVAL03 + 7 bài mẫu + 5 biến thể chèn lệnh (data/injection)
# mục 6 (k6, chạy trong VM Colima bằng image grafana/k6) trước vòng 1 của EVAL03, vì k6 bấm "Chấm tất cả" ở giây 5:
#   COMPOSE_PROJECT_NAME=ep-research node benchmarks/load/run-t1.mjs grading-baseline --vus 4 --course <eval> --assignment <eval_assignment>
#   … grading-grading … --baseline <p95 TTFT ms>   (k6 trong VM, chung mạng với caddy; thêm `--accounts t0` cho tài khoản seed T0)
python3 real_llm.py grade-run --target eval --rounds 3
python3 real_llm.py grade-run --target bt03 --rounds 2
python3 real_llm.py grade-report
python3 real_llm.py mask                        # mục 3
python3 real_llm.py chat                        # mục 4
python3 real_llm.py threads                     # mục 4–5 (thread [EVAL-LLM] bị xoá mềm sau khi đo)
python3 real_llm.py misc                        # mục 5
```

## Dữ liệu đo tạo ra (chỉ trên `ep-research`, mất khi `down -v`)

| Ở đâu | Dữ liệu |
| --- | --- |
| Lớp `EVALLLM` | Bản nháp công thức, bài tập `EVAL03` và các kết quả chấm |
| `BT03` lớp 1 | Kết quả chấm nháp vòng 1–2 (`DRAFT`, không công bố) |
| Câu Q10, Q11 của `chat` | Có thể tự mở ticket (đường leo thang theo độ tin cậy) |

Không có mật khẩu hay khoá nào trong tệp này. Mật khẩu seed lấy từ `SEED_DEFAULT_PASSWORD`; không đặt thì dùng mặc định công khai của `seed.mjs`.

## Hạn mức

Mọi lời gọi LLM chạy tuần tự, giữa các câu chat có giãn cách. Gặp `429` / `503` thì lùi theo `Retry-After`. Ước tính tổng cộng khoảng 330 lời gọi LLM:

| Phần đo | Số lời gọi |
| --- | --- |
| Trích công thức | 10 |
| Chấm `EVAL03`: 12 bài × 2 lượt × 3 vòng | 72 |
| Chấm `BT03`: 22 bài × 2 lượt × 2 vòng | 88 |
| Chat + che danh tính | 33 |
| Threads | 4 |
| Việc khác | 6 |
| k6 với `VUS=4` | khoảng 90 |

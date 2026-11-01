# Đo LLM thật (OPS-VPS)

Script `real_llm.py` là client ngoài: nó chỉ gọi HTTP API của gateway và chỉ dùng thư viện chuẩn của Python. Mô hình được đo là mô hình mà stack đang cấu hình (OPS-VPS: `ag/gemini-3.8-flash` qua 9router).
- Kết quả ghi vào `benchmarks/reports/real-llm/*.json`.
- Báo cáo nằm ở `docs/research/2026-11-01-real-llm-eval.md`.

```bash
export BASE=https://v2202609419695521098.happysrv.de
# tuỳ chọn, để đọc llm_audit / chat_messages; CHỈ ĐỌC (script tự bọc mỗi truy vấn trong `begin read only`)
export AUDIT_PSQL="ssh root@152.53.64.175 docker exec -i <postgres-container> psql -U <user> -d <db> -tA -v ON_ERROR_STOP=1"
cd benchmarks/real-llm
python3 real_llm.py setup                       # lớp đo EVALLLM + 2 quy chế + 12 sinh viên seed (không phải A, B, C)
python3 real_llm.py extract --runs 5            # mục 1
python3 real_llm.py grade-setup                 # mục 2: EVAL03 + 7 bài mẫu + 5 biến thể chèn lệnh (data/injection)
# mục 6 (k6) trước vòng 1 của EVAL03, vì k6 bấm "Chấm tất cả" ở giây 5:
#   k6 run ../load/grading-vs-chat.js -e BASE=$BASE -e VUS=4 -e PHASE=baseline
#   k6 run ../load/grading-vs-chat.js -e BASE=$BASE -e VUS=4 -e PHASE=grading -e COURSE_ID=<eval> -e ASSIGNMENT_ID=<eval_assignment> -e BASELINE_TTFT_MS=<p95>
python3 real_llm.py grade-run --target eval --rounds 3
python3 real_llm.py grade-run --target bt03 --rounds 2
python3 real_llm.py grade-report
python3 real_llm.py mask                        # mục 3
python3 real_llm.py chat                        # mục 4
python3 real_llm.py threads                     # mục 4–5 (thread [EVAL-LLM] bị xoá mềm sau khi đo)
python3 real_llm.py misc                        # mục 5
```

## Việc chạy để lại trên stack

| Ở đâu | Để lại gì |
| --- | --- |
| Lớp `EVALLLM` | Bản nháp công thức, bài tập `EVAL03` và các kết quả chấm |
| `BT03` lớp 1 | Kết quả chấm nháp vòng 1–2 (đều `DRAFT`, không công bố) |
| Câu Q10, Q11 của `chat` | Có thể tự mở ticket ở lớp 1 (đường leo thang theo độ tin cậy) |

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

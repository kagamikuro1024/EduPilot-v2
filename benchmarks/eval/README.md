# `make eval` — E1–E6 (US-P10-10)

Client ngoài, chỉ thư viện chuẩn Python, chỉ gọi HTTP API của gateway (D46). Ngoại lệ ghi trong từng báo cáo: đọc Postgres của stack đo trong giao dịch `READ ONLY` cho số API không trả (cos_top1 của truy xuất, `llm_audit`, quét rò rỉ `llm_request_payloads`).

```bash
. ~/Documents/ep-qc-env/env.sh        # API_URL, SEED_DEFAULT_PASSWORD, EVAL_DATE, EVAL_FAKE_KEY của stack ep-qc (ngoài repo)
make eval                             # đo + báo cáo; bước nào đã có raw/ thì bỏ qua (chạy lại = tiếp tục)
EVAL_REAL_EMBED=1 make eval           # nhúng thật cho E1/E2/E4 (chỉ khi PM cho phép); xong tự trả về nhúng giả + reindex
make eval-report                      # chỉ tính lại báo cáo từ raw/, không gọi LLM
python3 benchmarks/eval/selftest.py   # kiểm công thức (QWK, Spearman, độ tin cậy giả định)
```

Kết quả: `benchmarks/reports/eval-<ngày>/e{1..6}.json` + `.md` + `.svg`; kết quả thô ở `raw/`.

| E | Lệnh | Cần người |
| --- | --- | --- |
| E1 | `e1.py run --tag …` (gọi `benchmarks/eval_pii.py`), `e1.py leak` | soát ≥ 50 nhãn (P-11-3) |
| E2 | `e2.py run --tag P1 / P2` (Threads lớp 1, 120 câu vàng + 20 câu ngoài chủ đề) | soát 24 nhãn (`benchmarks/e2/README.md`) |
| E3 | `e3.py setup`, `e3.py grade --tag P1r1 --first / P1r2 / P2r1` (60 bài KC106 BT-A) | chấm 60 bài: điền `benchmarks/eval/human/e3_scores.csv` theo `e3-cham-tay.csv` |
| E4 | `e4.py` (đọc raw E2, E3) | – |
| E5 | `e5.py run` (10 lô × 10 câu; trần 5 lô / lớp / giờ — gặp 429 thì chạy lại sau) | duyệt câu trên UI ngân hàng câu hỏi |
| E6 | `e6.py run --tag r1 / r2` | – |

Đổi tuyến model: `stack.py route --tag P1 --model ag/gemini-3.8-flash`; nhúng: `stack.py embed-real` / `embed-fake` (khoá chỉ đọc trong tiến trình, không in, không ghi).

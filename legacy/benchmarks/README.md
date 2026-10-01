# Benchmark (theo Kế hoạch Benchmark Hệ thống)

Xem runbook đầy đủ theo tình huống tại `benchmarks/BENCHMARK_USAGE_GUIDE.md`.

Ba tầng:

1. **Tier 1 — Load & performance (Locust)**  
   Đo TTFT (SSE), throughput, failure rate. Profile: `baseline` (1 user, 10 câu đầu dataset tuần tự), `stress` (50 CCU, 5 phút), `spike` (200 user, 10s).

2. **Tier 2 — Chất lượng RAG / orchestrator (RAGAS)**  
   Dùng thư viện **RAGAS** (OpenAI): `faithfulness`, `context_precision` (with reference), `context_recall`.  
   Bổ sung rule-based theo plan: `must_contain_pass_rate`, `intent_accuracy`, `uncertain_behavior_pass_rate` (subset UNCERTAIN).

3. **Tier 3 — Cost & cache**  
   Chỉ khi backend gửi metadata trong SSE: `metadata.cache_hit` (bool), `metadata.usage` (`input_tokens` / `output_tokens`). Nếu thiếu, báo cáo ghi `MISSING` (không dùng proxy).

---

## Cài đặt

```powershell
pip install -r requirements.txt
```

Cần: `OPENAI_API_KEY` cho Tier 2. Tier 1/Locust cần `BENCHMARK_BEARER_TOKEN` hoặc `BENCHMARK_LOGIN_EMAIL` + `BENCHMARK_LOGIN_PASSWORD`.

---

## Preflight

```powershell
python -m benchmarks.preflight_check --dataset data/benchmark_ground_truth.jsonl --require-auth --require-openai
```

Kiểm tra: 100 dòng dataset, trường bắt buộc, `expected_pages` là mảng số nguyên hoặc `[]`, auth, OpenAI key, và template endpoint (chat/thread mặc định không cần `BENCHMARK_CONTEXT_ID` vì collect tạo context mới mỗi case).

---

## Chạy full (collect + RAGAS + postprocess)

```powershell
$env:SPRING_PROFILES_ACTIVE = "benchmark"   # khuyến nghị, tránh rate-limit
python -m benchmarks.run_benchmark --host "http://localhost:8080" --run-id "release-20260508"
```

Tùy chọn:

- `--run-locust` — chạy Locust profiles trước khi eval (ghi `reports/benchmark/<run_id>/load/*.csv`).
- `--predictions-input path\to\predictions.jsonl` — bỏ collect, copy file vào run dir.
- `--skip-postprocess` — chỉ collect + eval RAGAS.
- `--skip-preflight` — bỏ bước preflight.
- `--endpoint-template "/api/v1/threads/{id}/ask-ai"` — mỗi case tạo thread forum mới rồi gọi ask-ai.
- `--ragas-model gpt-4.1-mini` — model cho RAGAS (mặc định từ env `BENCHMARK_RAGAS_MODEL`).

---

## Từng bước thủ công

**Collect predictions (1 session/thread mới mỗi case):**

```powershell
python -m benchmarks.collect_predictions --host http://localhost:8080 --output reports/benchmark/my-run/predictions.jsonl
```

**RAGAS:**

```powershell
python -m benchmarks.eval_ragas --dataset data/benchmark_ground_truth.jsonl --predictions reports/benchmark/my-run/predictions.jsonl --run-id my-run
```

**Locust:**

```powershell
python -m benchmarks.run_locust_profiles --host http://localhost:8080 --run-id my-run --profiles baseline,stress,spike
```

**Postprocess:**

```powershell
python -m benchmarks.postprocess --run-id my-run
```

---

## Artifact trong `reports/benchmark/<run_id>/`

| File | Mô tả |
|------|--------|
| `predictions.jsonl` | `case_id`, `predicted_answer`, `predicted_intent`, `retrieved_contexts`, `metadata`, `ttft_ms`, `latency_ms` |
| `ragas_per_row.jsonl` | Điểm RAGAS + rule flags từng case |
| `ragas_summary.json` | Trung bình RAGAS + intent / must_contain / uncertain |
| `tier1_summary.json` | Tóm tắt Locust (nếu đã chạy) |
| `tier3_summary.json` | Cache/token từ metadata (nếu có) |
| `kpi_gate.json` | Actual vs target + cờ instrumentation |
| `target_vs_actual.md` | Đối chiếu KPI kế hoạch §4.1 |
| `final_benchmark_report.md` | Tóm tắt + P0/P1 |
| `run_meta.json` | Metadata run (runner) |

---

## Gaps backend (P1)

Để đủ Tier 3 đúng plan, Java SSE nên bổ sung trong `metadata`:

- `cache_hit`: boolean  
- `usage`: `{ "input_tokens": int, "output_tokens": int }`

Hiện đã có `citations[]` (snippet, page, chunk_id) — dùng làm `retrieved_contexts` cho RAGAS.

---

## Biến môi trường thường dùng

| Biến | Mô tả |
|------|--------|
| `BENCHMARK_HOST` | Host Java (mặc định `http://localhost:8080`) |
| `BENCHMARK_BEARER_TOKEN` | JWT |
| `BENCHMARK_LOGIN_EMAIL` / `BENCHMARK_LOGIN_PASSWORD` | Login thay token |
| `BENCHMARK_ENDPOINT_TEMPLATE` | Mặc định `/api/v1/chat/sessions/{id}/ask-ai` |
| `BENCHMARK_DATASET` | Dataset cho Locust |
| `OPENAI_API_KEY` | Bắt buộc cho RAGAS |
| `BENCHMARK_RAGAS_MODEL` | Model chấm (mặc định `gpt-4.1-mini`) |

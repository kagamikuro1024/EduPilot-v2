# Hướng Dẫn Sử Dụng Hệ Thống Đánh Giá & Benchmark

Tài liệu này là runbook thực hành: **khi nào dùng benchmark nào**, chạy lệnh nào, và đọc kết quả ra quyết định thế nào.

## 1) Dùng benchmark trong trường hợp nào?

- **Dev loop hằng ngày (sửa prompt/routing nhỏ):**
  - Mục tiêu: bắt regression nhanh.
  - Chạy: `collect + eval_ragas` trên subset (`--limit 10..20`).
  - Không bắt buộc chạy Locust.
- **Trước merge thay đổi vừa/lớn:**
  - Mục tiêu: kiểm chất lượng toàn bộ 100 case.
  - Chạy: full `run_benchmark` (không Locust hoặc có baseline/stress).
- **Trước release / demo lớn:**
  - Mục tiêu: đủ 3 tier theo kế hoạch.
  - Chạy: preflight + full quality + đầy đủ Locust profiles + postprocess.
- **Điều tra incident hiệu năng:**
  - Mục tiêu: tìm bottleneck nhanh.
  - Chạy: chỉ `run_locust_profiles` + đọc `load/*.csv` + `tier1_summary.json`.

## 2) Quy trình chuẩn (khuyến nghị cho release)

1. **Preflight**
2. **Collect predictions**
3. **Tier 2 RAGAS**
4. **Tier 1 Locust** (`baseline,stress,spike`)
5. **Postprocess** (`kpi_gate.json`, `target_vs_actual.md`, `final_benchmark_report.md`)

Lệnh một bước:

```powershell
$env:SPRING_PROFILES_ACTIVE = "benchmark"
python -m benchmarks.run_benchmark --host "http://localhost:8080" --run-id "release-YYYYMMDD" --run-locust
```

## 3) Trước khi chạy: checklist môi trường

- Dataset tồn tại: `data/benchmark_ground_truth.jsonl` (100 dòng).
- Có auth:
  - `BENCHMARK_BEARER_TOKEN`, hoặc
  - `BENCHMARK_LOGIN_EMAIL` + `BENCHMARK_LOGIN_PASSWORD`.
- Có OpenAI key cho RAGAS: `OPENAI_API_KEY`.
- Cài package:

```powershell
pip install -r requirements.txt
```

## 4) Các chế độ chạy thực tế

### A. Smoke nhanh sau khi sửa nhỏ

```powershell
python -m benchmarks.preflight_check --dataset data/benchmark_ground_truth.jsonl --require-auth --require-openai
python -m benchmarks.collect_predictions --host http://localhost:8080 --limit 20 --output reports/benchmark/smoke/predictions.jsonl
python -m benchmarks.eval_ragas --dataset data/benchmark_ground_truth.jsonl --predictions reports/benchmark/smoke/predictions.jsonl --run-id smoke-20
python -m benchmarks.postprocess --run-id smoke-20
```

Dùng khi: vừa đổi prompt/routing nhỏ, cần phản hồi nhanh.

### B. Full quality trước merge

```powershell
python -m benchmarks.run_benchmark --host "http://localhost:8080" --run-id "premerge-YYYYMMDD"
```

Dùng khi: thay đổi nhiều ở retrieval/orchestrator.

### C. Full release với tải

```powershell
python -m benchmarks.run_benchmark --host "http://localhost:8080" --run-id "release-YYYYMMDD" --run-locust
```

Dùng khi: chuẩn bị release, cần đủ Tier 1 + Tier 2 + Tier 3.

### D. Chỉ kiểm hiệu năng

```powershell
python -m benchmarks.run_locust_profiles --host "http://localhost:8080" --run-id "perf-YYYYMMDD" --profiles baseline,stress,spike
python -m benchmarks.postprocess --run-id "perf-YYYYMMDD"
```

Dùng khi: nghi ngờ nghẽn hệ thống, không cần chấm chất lượng nội dung.

## 5) Ý nghĩa từng artifact

- `predictions.jsonl`: câu trả lời thực tế từ endpoint, kèm `retrieved_contexts`, `metadata`, `ttft_ms`, `latency_ms`.
- `ragas_per_row.jsonl`: điểm từng case (`faithfulness`, `context_precision`, `context_recall`) + rule checks.
- `ragas_summary.json`: trung bình toàn bộ run cho Tier 2.
- `load/*_stats.csv`: dữ liệu Locust thô.
- `tier1_summary.json`: tóm tắt hiệu năng từ Locust (nếu có).
- `tier3_summary.json`: cache/token từ metadata backend (nếu có instrumentation).
- `kpi_gate.json`: trạng thái đạt/chưa đạt theo target kế hoạch.
- `target_vs_actual.md`: bảng đối chiếu chỉ số thực tế với target.
- `final_benchmark_report.md`: kết luận ngắn + next actions.
- `run_meta.json`: metadata kỹ thuật của run.

## 6) Cách đọc kết quả để ra quyết định

Ưu tiên đọc theo thứ tự:

1. `kpi_gate.json`
2. `target_vs_actual.md`
3. `ragas_summary.json`
4. `tier1_summary.json`
5. `final_benchmark_report.md`

Quy tắc ra quyết định nhanh:

- **Go (đề xuất)**:
  - `context_precision_mean >= 0.90`
  - `faithfulness_mean >= 0.95`
  - `ttft_p95_ms < 3000` (khi có Locust data)
- **Hold (cần tối ưu)**:
  - 1 trong các KPI chính chưa đạt.
  - `tier1_locust_artifacts = false` trong run release.
- **Need instrumentation**:
  - `cache_hit_rate = null`, hoặc token usage null.
  - Khi đó Tier 3 chưa đủ dữ liệu để chốt tối ưu chi phí/cache.

## 7) Mapping nhanh: yêu cầu -> lệnh

- **“Kiểm tra nhanh có vỡ logic không?”** -> chế độ A.
- **“So sánh bản mới với baseline trước merge”** -> chế độ B.
- **“Chốt release có đủ tải + chất lượng chưa?”** -> chế độ C.
- **“Sao gần đây chậm, cần đo tail latency?”** -> chế độ D.

## 8) Các lỗi hay gặp và cách xử lý

- `preflight` fail vì auth:
  - Set token hoặc cặp email/password benchmark.
- `preflight` fail vì OpenAI:
  - Set `OPENAI_API_KEY`.
- `eval_ragas` fail vì thiếu prediction:
  - Chạy lại `collect_predictions` hoặc kiểm tra `--predictions`.
- `Tier 3` thiếu cache/token:
  - Backend chưa emit `metadata.cache_hit` và `metadata.usage`.

## 9) Giới hạn hiện tại (biết trước để tránh hiểu nhầm)

- Tier 3 không dùng proxy; nếu backend chưa instrument thì báo `MISSING`.
- Điểm RAGAS phụ thuộc model judge và chất lượng context trích xuất.
- Nên dùng cùng dataset/model/env khi so sánh giữa các run.

# Đánh giá offline E1–E6 (US-P10-10). Script Python chỉ gọi HTTP API của gateway (D46). Xem benchmarks/eval/run.py.
.PHONY: eval eval-report

eval:
	@python3 benchmarks/eval/run.py

eval-report:
	@python3 benchmarks/eval/run.py --report

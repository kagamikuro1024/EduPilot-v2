# Dựng bài tập + 12 bài nộp (6 chèn lệnh + 6 đối chứng) ở QD105A bằng hàm của benchmarks/decision/run.py (nguyên bản, chỉ HTTP). Chấm round 1 khi cổng tắt.
import os, sys, importlib.util
os.environ.update(API_URL="https://localhost:943/api/v1", BENCH_COURSE="QD105A", GRADE_TIMEOUT="900")
spec = importlib.util.spec_from_file_location("bench", os.path.expanduser("~/Documents/TA_Agent_qc106/benchmarks/decision/run.py")); b = importlib.util.module_from_spec(spec); spec.loader.exec_module(b)
c1 = b.course_id(); print("QD105A", c1)
print(b.scenario_grading(c1, sys.argv[1] if len(sys.argv) > 1 else "S105A"))

# Handoff dev3 (sprint 11)

## Nợ / theo dõi
- `TestTwoProcessesRPM` (`backend-go/internal/llm/scheduler/integration_test.go:250`, tag `integration`): lệch dưới tải — chạy cùng cả cụm `./internal/llm/... ./internal/decision/... ./internal/llmconfig/...` thì «19 lời gọi được phép trong 10 s, trần tính được 16» (ngưỡng ≤ 17); chạy riêng xanh. Không đụng mã scheduler; nếu CI đỏ vì nó thì sửa gốc (đo cửa sổ 10 s theo đồng hồ thật), không retry.
- `benchmarks/guidance/baseline.json`: đo trên ep-qc, 30 ca đều «giảng viên sửa c2→0» nên hẹp; đo lại khi đổi prompt / model / rubric.
- /practice/[id] LCP: nợ AC6 trong `docs/PROGRESS.md` (không SSR dữ liệu).

## Việc

## Kiểm
- [ ] Test liên quan đã chạy xanh
- [ ] Đổi migration / schema: `go test -tags integration ./internal/store/...` xanh
- [ ] Đổi prompt chấm / model: `cmd/regress run --assignment <id> --baseline benchmarks/guidance/baseline.json` ở môi trường đo, không «Có lùi» (guide `docs/guides/grading-guidance.md` §3; CI không chạy lệnh này)
- [ ] Đổi giao diện: `bash scripts/ui-antipatterns.sh` sạch

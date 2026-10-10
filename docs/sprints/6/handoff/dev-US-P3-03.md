# DEV handoff — US-P3-03
Nhánh: `sprint/6-p3-p8`. Commit: `US-P3-03: …`.

## Đã làm
- **`internal/llm/mask.go`**: giao diện `llm.Masker` (`*privacy.Masker` cài đặt), `Options.Masker` / `Options.NoMask`, `Options.Validate()`. Che **một lần** ở đầu `Chat` / `Stream` / `Structured` / `Embed`, SAU `begin` và TRƯỚC vòng dự phòng của registry (mọi nhà cung cấp nhận cùng payload). Che: mọi `Message.Content` (system, lịch sử, tin hiện tại, kết quả tool) và `EmbedRequest.Inputs`; `Passages` không phải payload gửi provider nên không che. Phạm vi roster = `CourseID` của `llm.Identity` trong ctx; phiên = `privacy.SessionFrom(ctx)`, không có thì phạm vi yêu cầu. Không có `CourseID` → `ErrMaskFailed`, 0 lời gọi provider, một dòng `llm_audit` `status='error'`, `error_kind='MASK_FAILED'`.
- **Đầu ra**: `Chat` khôi phục `Response.Text`; `Stream` bọc từng `Chunk.Text` bằng `StreamUnmasker`, `Flush()` trước `Chunk.Done`, `Response.Text` đã khôi phục; `Structured` kiểm schema trên bản thô rồi khôi phục các **chuỗi giá trị** (đi cây JSON, khoá giữ nguyên). Đường suy giảm dùng `Passages` gốc, không qua che.
- `llm_audit.pii_masked_count` = tổng lần thay của lời gọi (`AuditRow.PIIMaskedCount`).
- **`llmrt`**: `New` dựng `privacy.Masker` thật (roster Postgres + cache Redis, ánh xạ Redis, hạn `PRIVACY_MASK_TIMEOUT_MS`); `NewWithMasker(…, nil)` trả lỗi cấu hình (gateway / worker không khởi động không che).
- **Cấu hình**: `PRIVACY_MASK_TIMEOUT_MS` (mặc định 50, 1–60000) → `config.PrivacyMaskTimeout` → `privacy.Masker.Timeout` (proposals #9: QC hạ thấp để ép `MASK_FAILED`).
- **Route thử (build `testroutes`, ADMIN, proposals #9)**: `GET /api/v1/_test/llm/payloads` (vòng đệm ≤ 200 lời gọi gần nhất mà **provider giả nhận**, tức SAU khi che: `{items:[{task, messages[], inputs[]}]}`), `POST /api/v1/_test/llm/payloads/reset` (204). `POST /_test/llm/chat` nhận thêm `course_id`, `session_id` để che thật; không có `course_id` thì chạy không che (hành vi cũ của route, qua `llm.WithNoMask` chỉ tồn tại ở bản dựng `testroutes` — kiểm `go tool nm` bản thường: 0 ký hiệu). Cả hai route có trong `openapi.test.yaml`, kịch bản hợp đồng 200 / 401 / 403 / 204; số thao tác test spec 19 → 21.
- `fake.Controller.Payloads/ResetPayloads` (vòng đệm 200) và `provider.ChatOpts.Task`.
- Test cũ dựng `llm.New` giờ đặt `NoMask: true` (12 chỗ trong test của `internal/llm`, `scheduler`).

## File đổi
`internal/llm/{mask,nomask_on,nomask_off,gateway,stream}.go`, `llm/fake/fake.go`, `llm/provider/provider.go`, `llm/llmrt/runtime.go`, `privacy/mask.go` (Timeout), `platform/config/config.go`, `testroutes/llm.go`, `api/openapi.test.yaml`, test: `llm/mask_test.go`, `llm/audit_pg_test.go`, `integration/no_payload_leak_test.go`, `cmd/{gateway,worker}/masker_test.go`, `contract/{scenarios_test,contract_test}.go`.

## Lệnh QC chạy để kiểm
```bash
cd backend-go; export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
go test -count=1 -race ./internal/llm -run 'TestMaskOnlyInLLMGateway|TestMaskCoversAllPayloadParts|TestMaskNoCourseFailsClosed|TestMaskOnceBeforeFallback|TestUnmaskBeforeCaller|TestStructuredUnmaskAfterSchema|TestAuditPIIMaskedCount|TestAuditPIIMaskedCountPG|TestProviderErrorsNotLogged|TestValidateRequiresMasker' -v
go test -count=1 -race ./internal/integration -run TestNoPayloadLeak -v     # in: payload đã quét, placeholder
go test -count=1 ./cmd/gateway ./cmd/worker -run TestStartupRequiresMasker -v
go test -count=1 -tags testroutes ./internal/contract
go build -o /tmp/gw ./cmd/gateway && go tool nm /tmp/gw | grep -c WithNoMask      # 0
```

## Test đã chạy và kết quả
Các lệnh trên xanh (`TestNoPayloadLeak`: 122 payload, 874 placeholder, 0 rò — roster 30 sinh viên Postgres + Redis, chat / Stream / Structured / Embed / kết quả tool / đường suy giảm). `go vet`, `golangci-lint` (build tag `testroutes`) sạch cho mã của dev; `internal/privacy/qc_p302_test.go` (QC) còn 7 lỗi lint (errcheck / ineffassign / contextcheck) — PM chuyển QC sửa. Full suite: xem commit.

## AC tự đánh giá
AC1 ✓ (`TestMaskOnlyInLLMGateway` quét AST: file nhập `privacy` ngoài `internal/llm`, `internal/privacy`, `cmd/*` mà gọi `Mask/Unmask/NewStreamUnmasker` → đỏ) · AC2 ✓ · AC3 ✓ · AC4 ✓ · AC5 ✓ · AC6 ✓ ở **tầng cổng LLM**; chat / Threads dịch vụ thật chưa có nên kịch bản đi qua `internal/chat`, `internal/thread` thêm ở US-P3-05 / 06 vào cùng bảng secrets · AC7 ✓ (`llmrt.NewWithMasker(nil)` lỗi; `TestStartupRequiresMasker` ở `cmd/gateway` và `cmd/worker`) · AC8 ✓ · AC9 ✓ (giữ nguyên `FEAT-llm-gateway`; `TestUsageNoContent` đã có từ P1).

## Nợ / cần hỏi
1. `TestNoPayloadLeak` thiếu kịch bản dịch vụ thật (chat riêng, Threads, đường suy giảm do chat tự dựng) — thêm ở P3-05 / 06.
2. Che quá `PRIVACY_MASK_TIMEOUT_MS` chỉ chặn phần CPU (nạp roster khi trượt cache có hạn riêng 2 s, không tính vào 50 ms) — ghi trong `privacy/mask.go`. Muốn tính cả nạp roster: báo.
3. `contract_test.go` ghim 21 thao tác test (đổi số, chữ trong thông báo đổi theo).

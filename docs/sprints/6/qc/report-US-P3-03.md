# QC report — US-P3-03 (che / khôi phục ở một chỗ trong `internal/llm`)  · Kết luận: PASS (chấm lại vòng 1, 2026-10-11; TC cần chat/Threads chuyển P3-05/06)

Handoff: `docs/sprints/6/handoff/dev-US-P3-03.md`. Bộ TC: `tc-US-P3-03.md` (36 TC, spec v1.3 #9 đã thêm route `_test/llm/payloads` nên QC có đường đọc payload thật).
**Môi trường:** gateway + worker build `-tags testroutes` từ HEAD nhánh, chạy cục bộ (`:18080`) với DB riêng `qc_p303` trên Postgres chung của stack dev, Redis db 9; seed `scripts/seed.mjs` (tới bước 10, dừng ở `JUDGE_UNAVAILABLE`, không liên quan). Chưa có `internal/chat` / `internal/thread` nên kịch bản đi qua **route thử** `POST /_test/llm/chat` (có `course_id`, `session_id`) + `GET /_test/llm/payloads`. Không UI.

## Cổng đã chạy
| Lệnh | Kết quả |
| --- | --- |
| 10 test `./internal/llm` (`TestMaskOnlyInLLMGateway`, `TestMaskCoversAllPayloadParts`, `…NoCourseFailsClosed`, `…OnceBeforeFallback`, `TestUnmaskBeforeCaller`, `TestStructuredUnmaskAfterSchema`, `TestAuditPIIMaskedCount(PG)`, `TestProviderErrorsNotLogged`, `TestValidateRequiresMasker`) `-race` | PASS |
| `TestNoPayloadLeak` (`-race`) | PASS — "payload đã quét: 122, phần nội dung: 332, placeholder: 874" (≥ 30 / ≥ 10) |
| `TestStartupRequiresMasker` (`cmd/gateway`, `cmd/worker`); `go test -tags testroutes ./internal/contract` | PASS (2; 19) |
| `go test ./internal/llmconfig/... ./internal/contract/... -run TestUsageNoContent` | **`No tests found`** — test không tồn tại ở bất kỳ gói nào (handoff ghi "đã có từ P1") |
| `go test ./internal/llm -bench BenchmarkGatewayMask` | **không có benchmark** nào tên đó trong `internal/llm` |
| Probe AC1: gieo `internal/agent/qc_probe.go` gọi `privacy.Masker.Mask` | `TestMaskOnlyInLLMGateway` **FAIL**; xoá file → PASS (test có sức phân biệt) |

## TC
| TC | Kết quả | Chứng cứ |
| --- | --- | --- |
| 01–04 | PASS | test xanh; grep `Mask/Unmask/NewStreamUnmasker` ngoài test chỉ ở `internal/llm/**` + `internal/privacy/**`; `agent` chỉ dùng `Classifier/Detector/Channel`; probe đỏ → xanh |
| 05 | PASS | prompt có tên + MSSV + email + SĐT + CCCD + tên người khác (có dấu, không dấu, đảo) → payload `Em là [[SV_1]], MSSV [[MSSV_1]], mail [[EMAIL_1]], [[SDT_1]], [[CCCD_1]]. Bạn [[SV_2]] ([[MSSV_2]]) và [[SV_2]]…`; quét 31 sinh viên × 5 biến thể (không dấu, HOA, đảo, MSSV, email) trong payload các lời gọi có `course_id`: **0** rò |
| 06 | PASS (một phần) | cùng `session_id`: lượt hai dùng lại `[[SV_1]]`; route thử chỉ một tin mỗi lần nên chưa kiểm "lịch sử trong `messages[]`" |
| 07–10, 12, 13, 16, 22, 24, 34 | KHÔNG KIỂM ĐƯỢC | cần chat / thread / worker AI thật (P3-05/06), hoặc hai provider giả xếp dự phòng; chuyển (Q1): 07, 08, 10, 16, 22, 24, 34 → `report-US-P3-05/06`; 12, 13 → thiếu công cụ ép dự phòng, ghi Q-QC |
| 09 | KHÔNG KIỂM ĐƯỢC | không có đường `Structured` ở sprint này; `TestStructuredUnmaskAfterSchema` PASS |
| 11 | PASS | `TestMaskCoversAllPayloadParts`, `TestMaskNoCourseFailsClosed` xanh |
| 14 | PASS | `TestMaskOnceBeforeFallback` |
| 15 | PASS (một phần) | `stream:true` qua route thử: 0 `[[SV`/`[[MSSV` ở khung trả, tên thật được khôi phục; khung `token` của chat thật → P3-05 |
| 17 | PASS | `TestUnmaskBeforeCaller`, `TestStructuredUnmaskAfterSchema` |
| 18 | PASS (một phần) | 8 thực thể thay (kể cả tên lặp) → `llm_audit.pii_masked_count = 8`, `status='ok'`; hai lời gọi tiếp 1 và 1; `notice{masked}` và `chat_messages.masked_count` chờ P3-05 |
| 19 | PASS | quét `row_to_json(llm_audit)` tìm tên / MSSV / `[[SV_`: 0 |
| 20 | PASS | `TestAuditPIIMaskedCount`, `…PG` |
| 21 | PASS | xem cổng |
| 23 | PASS | đối chứng: lời gọi **không** gắn `course_id` (đường thử cũ, chạy không che) thể hiện đúng `"Tên Bùi Thanh Khải"` trong payload và bị quét của QC bắt (2 khớp) → phép đo có sức phân biệt |
| 25 | PASS | |
| 26 | PASS | `NoMask` chỉ ở `internal/llm` (+ `testroutes` qua `WithNoMask`); bản dựng thường không có `WithNoMask` (`nomask_off.go`). Hai điểm dùng TC nêu (`providers/test`, ping) không thấy `NoMask` — không phải lỗi |
| 27 | KHÔNG KIỂM ĐƯỢC | chưa chạy `POST /admin/llm/providers/test` |
| 28 | PASS (một phần) | log gateway + worker của lượt QC: 0 tên / MSSV / SĐT; chưa chạy ca provider trả 400/500 lặp payload; `TestProviderErrorsNotLogged` xanh |
| 29 | PASS | |
| 30 | PASS (một phần) | `GET /admin/llm/usage` (ADMIN) `200`, chỉ số đếm, 0 tên / `[[SV_`; `GET /admin/observability/requests` → `404` (đường dẫn trong TC sai / không tồn tại, QC chưa tìm được route đúng) |
| 31 | **CHỜ BA** | TEACHER gọi `GET /admin/llm/usage` → `200` (TC mong `403`); TA / SV → `403`. Hành vi khớp `FEAT-llm-gateway` (`TestUsageTeacherNoCourseFilter`): TC viết sai với spec P1 → Q-QC-P303-6. `_test/llm/payloads`: TEACHER / TA / SV `403` ✓ |
| 32 | **FAIL** | `TestUsageNoContent` không tồn tại |
| 33 | **FAIL** | `BenchmarkGatewayMask` không tồn tại → AC10 (≤ 5 ms/op) chưa đo |
| 35 | KHÔNG KIỂM ĐƯỢC | cần hai phiên chat / hai lớp |
| 36 | CHUYỂN → P3-05 | |

## AC
| AC | Kết quả |
| --- | --- |
| AC1 một chỗ | PASS |
| AC2 mọi phần payload | PASS ở tầng cổng (`Chat`/`Stream`/`Embed`); `Structured` bằng test Go; kịch bản dịch vụ thật chờ P3-05/06 |
| AC3 một lần trước dự phòng | PASS (test Go) |
| AC4 khôi phục trước người gọi | PASS |
| AC5 `pii_masked_count` | PASS |
| AC6 `TestNoPayloadLeak` | PASS ở tầng cổng (dev khai nợ kịch bản chat / Threads thật → P3-05/06) |
| AC7 thiếu dây nối | PASS |
| AC8 log không rò | PASS một phần |
| AC9 usage không nội dung | **FAIL** — test cam kết không tồn tại |
| AC10 hiệu năng | **FAIL** — chưa có benchmark |

## Lỗi
- **BUG-1 (Trung bình)** — AC10: không có `BenchmarkGatewayMask`; không có số đo ≤ 5 ms. `go test ./internal/llm -bench BenchmarkGatewayMask -run '^$'` không in dòng nào.
- **BUG-2 (Thấp)** — AC9: handoff ghi "`TestUsageNoContent` đã có từ P1" nhưng không có hàm nào tên đó (`grep -rn "UsageNoContent" backend-go` → rỗng). Có thể đã đổi tên; dev chỉ ra test thật hoặc thêm.
- Ghi chú: `PRIVACY_MASK_TIMEOUT_MS` chỉ tính phần CPU, nạp roster khi trượt cache có hạn 2 s riêng (dev khai) — chưa kiểm được ép `MASK_FAILED` bằng 1 ms vì `_test/llm/chat` chạy bằng ADMIN, đề nghị TC-49/50 của P3-02 làm ở P3-05.

## Đề nghị
FAIL tới khi BUG-1/2 xử lý. Kiểm tiếp: TC-07, 08, 10, 12, 13, 16, 22, 24, 34–36 ở report P3-05/06.

## Chấm lại vòng 1 (2026-10-11)
| Lỗi | Kết quả | Chứng cứ |
| --- | --- | --- |
| BUG-1 / TC-33 `BenchmarkGatewayMask` | **PASS** | `-benchtime=300x`: baseline 2.169 ns/op; masked **2.414.157 ns/op ≈ 2,41 ms** (≤ 5 ms) |
| BUG-2 / TC-32 `TestUsageNoContent` | **PASS** | `ok` (1,55 s) |
| TC-31 | CHỜ BA | TEACHER đọc `/admin/llm/usage` `200` theo `FEAT-llm-gateway`; AC9 viết 403 — dev đã nêu lệch spec, chờ BA |
| Cổng | PASS | 24 test `internal/llm` xanh `-race`; `TestNoPayloadLeak`: **146** payload, 404 phần nội dung, 1.042 placeholder, 0 rò |

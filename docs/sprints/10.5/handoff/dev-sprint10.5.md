# Handoff Dev — Sprint 10.5 (Decision model + cascade, D60)

Nhánh `sprint/10.5-decision`. Mỗi lát một commit `10.5: L<n> …`. Cập nhật theo từng lát.

## Bản decision giả (`decision-fake`) — bảng ca replay (P-105-1 Q2/Q3)

Bản giả không gọi mạng, tất định. Khoá phát lại = `sha256(đầu vào ĐÃ CHE) + ":" + id câu hỏi`, nạp từ `backend-go/internal/llm/testdata/decision-replay.json` (nhúng vào binary; Go test thêm bằng `Controller.AddDecision(maskedInput, qid, DecisionRecord)`).

| Đầu vào | Câu hỏi | Kết quả | Đường đi cascade |
| --- | --- | --- | --- |
| Có trong replay, `noul` ≥ 0,5 | `noul` | P(có) = giá trị replay (kẹp 0,01–0,99) | `AddOnly`: FAST (có) |
| Có trong replay, `noul` ≤ ε | `noul` | như trên | FAST (không) |
| Có trong replay, ε < `noul` < 0,5 | `noul` | như trên | FALLBACK |
| Có trong replay, `choice` conf ≥ τ | `choice` | nhãn + xác suất của replay | `Symmetric`: FAST |
| **Không có trong replay (đầu vào lạ)** | `noul` | **0,25** (trung tính, dưới ngưỡng "có", trên ε 0,01–0,20) | FALLBACK |
| Không có trong replay | `choice` | các nhãn chia đều, confidence = 1/n | FALLBACK |
| không có trong replay, `FAKE_DECISION_PROB=p` (replay thắng) | `noul` / `choice` | `noul` = p; nhãn đứng đầu (theo bảng chữ cái) có confidence p | theo p |
| bất kỳ, `LLM_DECISION_FAKE_MODE=always_yes` / `always_no` | `noul` | 0,99 / 0,01 | FAST |
| bất kỳ, `…MODE=error` | — | HTTP 503 giả | FALLBACK (hoặc FAIL_CLOSED nếu fallback cũng lỗi) |
| bất kỳ, `…MODE=timeout` | — | treo tới hết hạn ctx | FALLBACK |

Độ trễ log-chuẩn: `LLM_DECISION_FAKE_P50_MS` / `…_P95_MS` (mặc định 426 / 682); tỉ lệ lỗi `LLM_DECISION_FAKE_ERROR_RATE`.

## Base URL decision (P-105-1 Q1)

Bản thật gọi `POST {base}/decisions` (header `x-goog-api-key`). `{base}` lấy theo thứ tự: cột `base_url` của bản ghi provider `type=decision` → env `LLM_DECISION_BASE_URL` → Google. QC dựng máy chủ giả và trỏ vào đây. Mô hình thật chỉ chạy khi `LLM_DECISION_ALLOW_REAL=true` VÀ provider chọn mô hình `gemini-decision-1`; mặc định `409 DECISION_REAL_DISABLED`.

## Biến môi trường mới

`LLM_DECISION_TIMEOUT` (800ms), `LLM_DECISION_ALLOW_REAL` (false), `LLM_DECISION_REAL_DAILY_CAP` (300/ngày UTC+7), `LLM_DECISION_RPM` (150), `LLM_DECISION_BASE_URL`, `LLM_DECISION_FAKE_{P50_MS,P95_MS,ERROR_RATE,MODE}`, `FAKE_DECISION_PROB`.

## Điểm spec ≠ TL-1 (spec FEAT-decision-model thắng)

- Tên cổng `THREADS_PRECHECK` / `INJECTION_FLAG` / `CHAT_ROUTING`; `cascade_path` ∈ FAST | FALLBACK | FAIL_CLOSED.
- Không thêm `DECIDE` vào `llm_task_routes`: chọn provider bằng đúng MỘT hàng `llm_providers` `type='decision'` đang bật (unique index).
- Cấu hình cổng ở hai bảng `decision_gates` / `course_decision_gates` (không phải cột trên `courses`); ngưỡng ở `decision_gates`.
- `Client` có 5 hàm: `TestClientSurface` đổi từ 4 → 5 (đúng SRS 4.1). Tên kiểu cổng là `llm.DecideGate` (`llm.Gate` đã là giao diện Scheduler).
- Migration seed một nhà `decision` nên `ListProviders` trả thêm 1 hàng: `TestNoNPlusOne` đổi `Len 20 → 21`; hạn mức 20 nhà không tính nhà `decision` (`CountLLMProviders`).

## P-105-2 (đã làm)

- `TestStudentNeverSeesDraftGrade`: «6.37» khớp vào dấu thời gian `…46.379254Z` (dương tính giả theo giờ chạy). Không bề mặt nào lộ điểm nháp (kiểm lại cả 8 trạng thái × 7 bề mặt + tool tóm tắt). Phép kiểm số đổi từ chuỗi con sang `leaksNumber` (số đứng riêng, ranh giới chữ số / dấu chấm; thử cả dấu phẩy); chuỗi canary chữ và các khoá cấm giữ nguyên. Thêm `TestLeaksNumber` (dấu thời gian không báo nhầm; số thật, chuỗi `"6,37"`, `"được 6.37 điểm"` vẫn bị bắt, kể cả khi cùng body có dấu thời gian vô hại).
- Hai test sửa ở L1 theo spec: `TestClientSurface` 4→5 hàm; `TestNoNPlusOne` `Len 20→21` (seed nhà `decision`).

## Lát đã bàn giao
- **L1** (`425d2a74` + openapi `e896b20d`): US-DM-01, US-DM-02. Test: `go test ./internal/llm/... ./internal/llmconfig ./db`; contract cần `-tags testroutes` (không có tag → `_test/llm/fake` 404, đây là bản dựng mặc định).
- **L2**: US-DM-03 — `thread.Firewall.CheckPublish` (luật `CheckPost` → nếu sạch mới `Cascade` THREADS_PRECHECK, `AddOnly`; fallback `Structured` task CLASSIFY, hạn `THREADS_STRUCTURED_TIMEOUT` 6 s; fail-closed = chặn). Dùng ở `Service.gate` (đăng, bình luận, sửa, cả lần kiểm lại bản «đã ẩn»); `Precheck` và `RecordSwitched` vẫn `CheckPost` (không decision). Luật mới: nội dung chỉ có placeholder → chặn (AC4). Test: `TestThreadsDecisionFlow` (10 ca), `TestPrecheckAddOnly`, `TestThreadsDecisionComment`, `TestThreadsCrisisNeverReachesDecide`, `TestPlaceholderOnlyBlockedByRules` (`go test ./internal/thread`).
- **L3**: US-DM-04 (gộp P-11-9) — `grading.Grader.Decide` (worker truyền `llmRT.Gateway`). Cờ `INJECTION_SUSPECTED` = regex (toàn văn, giữ nguyên) OR LLM `Structured` OR decision. Hai tầng sau chỉ chạy khi `Gateway.GateEffective(INJECTION_FLAG)` cho lớp VÀ bài ≥ 20 từ: tầng Structured **gộp vào hai lượt chấm sẵn có** (`EssaySchemaInjection` cho phép thêm giá trị `INJECTION_SUSPECTED` trong `flags` + một dòng luật trong prompt; cổng tắt → schema / prompt y như trước), tầng decision chạy song song (`flagInjectionDecision`, làn BATCH, 3 s). Chỉ cờ: điểm không đổi. Test: `TestInjectionOR` (7 ca, `go test ./internal/grading`), `TestDecideCallersAllowlist` (`internal/llm`, chặn mọi lời gọi `Decide`/`Cascade` ngoài danh sách trắng; `agent.routeDecision` thêm ở L4). Helper test dùng chung: `internal/llm/llmtest.Gateway`. Bản phát lại cho 6 biến thể 9.5 (AC5): dựng ở L6 cùng bench.
- **L4**: US-DM-05 — `agent.Analyzer.Decider` (`routeDecision`, 11 nhãn, `Symmetric` τ 0,95). Chỉ hỏi khi `DetectIntent` = COURSE_QA; khởi động đúng lúc Classify gọi nhúng (song song với nhúng); luật PII / mẫu cá nhân / hỏi hộ đã quyết → không gọi; không có nhúng (tin ngắn / chưa có mẫu cá nhân) → hỏi tại chỗ. Nhãn ≥ τ → đổi `Intent` (OTHER_PERSON → `OtherPerson=true`, đường từ chối cũ); dưới τ / lỗi / hết hạn / cổng tắt → giữ luật, không gọi LLM. Test: `TestRouteDecisionTable` (9 ca), `TestRouteDecisionNoOverride`, `TestCrisisNeverReachesDecide`, `TestRouteDecisionParallelWithEmbed` (`go test ./internal/agent`). `TestOneEmbedPerMessage` giữ nguyên, xanh.
- **L5**: US-DM-06, US-DM-07.
  - API (`internal/decision`, `internal/httpapi/llmhttp/decision.go`, openapi 6 thao tác): `GET/PUT /admin/llm/decision…`, `GET /admin/llm/decision/courses`, `PUT/GET /courses/{id}/decision-gates…`, `GET /admin/observability/decision`. Provider `type=decision` dùng API nhà cung cấp sẵn có (`POST …/providers/{id}/test` cục bộ với bản giả; bản thật khi `LLM_DECISION_ALLOW_REAL=false` → `409 DECISION_REAL_DISABLED`, không gọi mạng). Cảnh báo `DECISION_FAIL_CLOSED`: worker `decision.alert` mỗi `DECISION_ALERT_TICK` (5 phút), khử trùng `DECISION_ALERT_DEDUPE` (6 giờ), một bản nhờ khoá leader Redis. «Tạm dừng» của provider tính lúc đọc (cầu dao mở hoặc `budget_blocked` trong 1 giờ), không thông báo riêng.
  - Test Go: `go test ./internal/decision` (service, RBAC, số liệu, cảnh báo), `go test -tags testroutes ./internal/contract` (6 thao tác với mọi status; thêm `TestCascadeThreadsHTTP`: HTTP thật + Postgres + provider giả, các nhánh (a)(b)(c)(f) của AC3 kèm `cascade_path` ở llm_audit).
  - Frontend: khối «Quyết định nhanh» (`features/settings/llm/DecisionSection.tsx`, sau «Chuỗi dự phòng», chỉ ADMIN); bảng «Quyết định nhanh» ở `/observability` (`features/observability/DecisionTable.tsx`, chỉ ADMIN); chỉ báo «Đang kiểm tra» khi đăng Threads chờ > 800 ms. Playwright (máy chủ giả, CI): `decision-settings.spec.ts` (9 ca), `decision-observability.spec.ts` (4 ca), `cascade.spec.ts` (2 ca: hộp thoại hai lối khi bị cổng decision chặn + chỉ báo chờ). Ba test cũ đổi theo spec: `settings-llm.spec.ts` «4 phần» → 5 phần; số nút `Test kết nối` (admin) 2 → 3 (hàng «Quyết định»; ca provider-row giới hạn trong `[data-part=provider-row]`). Ảnh mốc `settings-llm-*` KHÔNG đổi (khối mới nằm ngoài khung nhìn đầu; đã kiểm trong image Playwright v1.63.0-noble, byte-identical).
  - Test route (`-tags testroutes`): `POST /_test/llm/fake` nhận thêm `decision_mode`, `decision_prob`, `decision_p50_ms`, `decision_p95_ms`, `decision_error_rate`; `GET /_test/llm/stats` thêm `fake_decisions`.
  - Chưa làm (có chủ đích): `cascade.spec.ts` bản chạy trên stack thật (cần stack dựng bằng `-tags testroutes`); nhánh (d) chèn lệnh và (e) chat đã kiểm ở mức Go (`TestInjectionOR`, `TestRouteDecisionTable`).
- **Sửa QC** (`4ffffbcd`, `3b902c37`, `b879d1f5`): (1) type/kind `decision` qua API nhà cung cấp (đã sửa từ L5; thêm: mô hình thật đang TẮT lưu được, chỉ BẬT mới cần `ALLOW_REAL`); (2) TEACHER không thấy provider `decision` và tác vụ `DECIDE` ở `/admin/llm/usage`; (3) `Test kết nối` decision-fake ok cục bộ; (4) compose chuyển `LLM_DECISION_*` / `FAKE_DECISION_PROB` / `DECISION_ALERT_*` cho worker. Test: `TestDecisionProviderHTTP` (contract, `-tags testroutes`), `TestComposeDecisionEnvReachesWorker` (`internal/platform/config`).
- **L6**: US-DM-08, US-DM-09.
  - Seed: `SEED_DECISION_DEMO=1 node scripts/seed.mjs` bật 3 cổng hệ thống + lớp 761987 bằng API ADMIN; mặc định ba cổng tắt, provider `decision-fake` có sẵn từ migration. Kiểm: `node scripts/check-decision-seed.mjs` (chạy lại seed rồi so, idempotent). **Chưa chạy trên stack** (không dựng stack theo PM): QC chạy.
  - Bản phát lại 61 ca (`internal/llm/testdata/decision-replay.json`): 20 chat (10 paraphrase + 10 COURSE_QA) + 30 Threads (20 sạch, 10 riêng tư; 10 «luật bắt» không tới decision) + 11 bài chấm (6 chèn lệnh = 5 biến thể 9.5 + s6; 5 đối chứng, s4 trống không gọi). Dựng và kiểm bởi `internal/contract/decision_replay_test.go` (`UPDATE_DECISION_REPLAY=1` để dựng lại; test cũng kiểm tiền đề: luật chưa quyết đúng các câu paraphrase). Dữ liệu: `benchmarks/decision/data/{chat,threads}.jsonl`.
  - Đo trước / sau: `bash benchmarks/decision/run.sh` (Python chuẩn, HTTP; thoát 2 «Gateway chưa chạy»; từ chối chạy khi provider decision không giả). **Chưa chạy trên stack** (PM hoãn đo): QC / đợt đo chung chạy. Chat đo độ chính xác bằng block dữ liệu (`blocks` của câu trả lời), loại câu trả lời từ cache khỏi TTFT (đoán qua việc thiếu giai đoạn `searching`).
  - Hướng dẫn: `docs/guides/decision-model.md` (hoàn chỉnh, không còn `[CHỜ]`). Docs vá: FLOWS F3/F4/F9 + «Decision cascade» (sprint 10.5, hết hạn theo cổng, F9 lỗi không tự gắn cờ); `FEAT-llm-gateway` SRS 7.3 thêm `DECIDE` («Quyết định nhanh»), `labels.ts` thêm nhãn. ARCHITECTURE / SYSTEM_DESIGN / DECISIONS D60 đã dùng `FAIL_CLOSED` + `decision_gates` + `LLM_DECISION_*` (TL vá trước).
  - DM-09: `TestDecideCallersAllowlist` (ranh giới gọi), `TestCrisisNeverReachesDecide` (chat + Threads), `TestDecideMasked`, `TestInjectionOR`, `TestCallBudgetPerMessage` (agent: ≤ 1 nhúng, ≤ 1 decision, luật quyết = 0 decision), `TestOneEmbedPerMessage`.
  - Đổi tên: `TestInjectionFlagTiers` → `TestInjectionOR` (khớp AC2 của US-DM-08).

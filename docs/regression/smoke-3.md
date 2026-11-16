# Smoke hồi quy sprint 3 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2/SmkB · Ngày: 2026-11-15
Tổng TC cũ: 523 · Đạt: 443 · Trượt: 1 · Lỗi thời (LT): 5 · Không kiểm được K (a/b/c): 1/7/8 · Chưa chạy (N): 58   (P+F+LT+K+N = T)
Tổng hợp N=0 đợt 2026-11-16 (qc2/SmkN): kết quả từng ca ở các tệp `evidence-3/results-smk34/*.tsv`; gộp bằng `evidence-3/scripts/smk34-merge.py 3`.

Cách chạy: (1) Go: `go test -race -json ./internal/llm/... ./internal/llmconfig/... ./internal/platform/crypto/...` + `-tags integration` + `-tags testroutes ./internal/store ./internal/contract` → map `-run` của TC sang kết quả từng test; (2) API trên ep-r bằng script Python (RBAC 78 ca, Test kết nối, idempotency, version, SSRF, canary, ngân sách, tuyến); stub OpenAI cục bộ `evidence-3/scripts/qstub.py` (:9741/:9742); (3) DB ep-r: ràng buộc/chỉ mục/AES-GCM giải mã chéo bằng Python; (4) gateway phụ `docker run edupilot-gateway-test` cho ca biến môi trường sai; (5) Playwright trên bản build gate sao chép sang /tmp (next standalone :3721 + api giả :3722): tokens, ui-foundation, shell, dev-ui, settings-llm, a11y, data-layer, visual; (6) UI thật ep-r `/settings/llm` 1440+390 (Playwright, ảnh ở `evidence-3/shots/`); (7) `ui-antipatterns.sh`, `--selftest`, `lint-selftest.sh`, `pnpm lint` trong bản sao.
Lưu ý môi trường: Chromium 1255 chưa có lúc đầu → dùng headless-shell 1247; CORS_ORIGINS ep-r thiếu https://localhost:1143 (mọi F5 văng /login) — đã nhờ PM-QC sửa, tôi dùng route đổi Origin của /auth/refresh; khoá global đang bị nhóm khác giữ nên cụm ca đổi tuyến/ngân sách/cầu dao (TC-P102-14…19/42, TC-P103-03/14…41, TC-P104-27/34, TC-P105-20/22) CHƯA chạy ("chưa chạy (mẫu)" — script `/tmp/qc2-b/p10x.py` sẵn sàng).

## Tổng hợp theo US
| US | TC tổng | P | F | LT | K | N | Ghi chú |
| --- | --- | --- | --- | --- | --- | --- | --- |
| GATE-P1 | 23 | 11 | 0 | 0 | 3 | 9 | |
| GATE-PU | 17 | 5 | 0 | 0 | 1 | 11 | |
| US-P1-01 | 35 | 34 | 0 | 0 | 0 | 1 | |
| US-P1-02 | 50 | 50 | 0 | 0 | 0 | 0 | |
| US-P1-03 | 50 | 45 | 0 | 0 | 0 | 5 | |
| US-P1-04 | 48 | 45 | 0 | 1 | 1 | 1 | |
| US-P1-05 | 57 | 51 | 1 | 0 | 2 | 3 | |
| US-PU-01 | 36 | 21 | 0 | 3 | 0 | 12 | |
| US-PU-02 | 47 | 43 | 0 | 0 | 2 | 2 | |
| US-PU-03 | 57 | 57 | 0 | 0 | 0 | 0 | |
| US-PU-04 | 59 | 58 | 0 | 0 | 0 | 1 | |
| US-PU-05 | 44 | 23 | 0 | 1 | 7 | 13 | |

## Kết quả từng ca
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-GATEP1-01 | P | grep ngoài internal/llm = 0; go.mod anthropic/genai = 0 |
| TC-GATEP1-02 | P | go test: 138 pass/1 skip/0 fail |
| TC-GATEP1-03 | P | go test: 82 pass/2 skip/0 fail |
| TC-GATEP1-04 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP1-05 | P | go test: 1 pass/0 skip/0 fail |
| TC-GATEP1-06 | P | go test: 55 pass/0 skip/0 fail |
| TC-GATEP1-07 | K(c) | (c) make eval có target (python3 benchmarks/eval/run.py) nhưng là phép đo P10 trên stack ep-qc với LLM thật; plan sprint 3 hoãn, BA trả lời «không FAIL», không là điều kiện PASS cổng — không chạy |
| TC-GATEP1-08 | P | go test: 25 pass/0 skip/0 fail |
| TC-GATEP1-09 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP1-10 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP1-11 | K(c) | (c) CI GitHub / nhánh ci/ui-drift của PR sprint 3 — ngoài phạm vi đồ án |
| TC-GATEP1-12 | P | stack cô lập, 2 nhà fake (SmkG-F1/F2), đổi CHAT 6 lần và chat ngay: lời gọi kế tiếp luôn dùng nhà mới (0,10–0,18 s sau PUT, ≤1 s); llm_audit 6 dòng cuối khớp từng provider/model/fallback_index (lượt 1). Lượt chạy lại lần 2 báo lệch khi so 6 dòng audit vì ghi audit bất đồng bộ chưa xả kịp lúc truy vấn, không phải lỗi sản phẩm (smk3-gate1.py) |
| TC-GATEP1-13 | K(a) | (a) thiếu khoá nhà cung cấp thật (OpenAI/Gemini): BLOCKED có chủ đích theo BA Q-QC-GATEP1-2, chỉ chứng minh được bằng fake (TC-GATEP1-12) |
| TC-GATEP1-14 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP1-15 | P | (a) tắt công tắc nhà chính F1: chat 200 dùng F2 fallback_index=0, hàng F1 enabled=false; (b) nhà chính lỗi còn bật (stub 503): 200 dùng F2 fallback_index=1, llm_audit fallback_index=1/ok, log WARN «chuyển nhà cung cấp dự phòng» có from=SmkG-S3 to=SmkG-F2 (smk3-gate1.py) |
| TC-GATEP1-16 | N | chưa chạy (mẫu): tải/đo hiệu năng nặng (k6, Lighthouse) — chưa thu nhỏ để chạy trên ep-r |
| TC-GATEP1-17 | P | mọi nhà chết (chuỗi 1 nhà 503, retries 0) + 5 passages: 200, degraded=true, 36 ms (≤2 s), mở đầu «AI tạm thời không khả dụng. Dưới đây là các đoạn tài liệu liên quan nhất:», trích đúng 3 đoạn nguyên văn kèm nguồn/trang, không từ kỹ thuật |
| TC-GATEP1-18 | P | go test: 0 pass/1 skip/0 fail |
| TC-GATEP1-19 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP1-20 | P | chạy lại trên ep-r hiện tại (smk3-gate20.py): ma trận 13 thao tác × 6 danh tính = 78/78 đúng (đường dẫn đúng /courses/{id}/llm-budget), 14 ca token giả (alg=none, secret sai, sửa vai, SUPERUSER, thiếu sub, rỗng, a.b.c) đều 401, số dòng llm_providers/routes/budgets trước = sau (10/6/1) |
| TC-GATEP1-21 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP1-22 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP1-99 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEPU-01 | P | lint mã sản phẩm 0 cảnh báo (xem TC-PU01-05); chưa dựng lại pbuild/gbuild để đếm «Failed to load font» nên chỉ chốt phần lint |
| TC-GATEPU-02 | P | ui-antipatterns 32 ✓ / 0 ✗; --selftest 32/32; lint-selftest 7/7; ui-allow 9 (≤10) |
| TC-GATEPU-03 | P | playwright desktop: ui-foundation + tokens + shell + dev-ui + settings-llm + a11y + data-layer đạt (ban đầu 5 ca fail vì bản sao thiếu audit.mjs, đã bổ sung) |
| TC-GATEPU-04 | P | Playwright (bỏ @real) trên bản build gate: tokens/ui-foundation/shell/dev-ui/settings-llm/a11y/data-layer đạt; visual: xem TC-PU05-01 |
| TC-GATEPU-05 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEPU-06 | N | chưa chạy (mẫu): tải/đo hiệu năng nặng (k6, Lighthouse) — chưa thu nhỏ để chạy trên ep-r |
| TC-GATEPU-07 | P | go test: 25 pass/0 skip/0 fail |
| TC-GATEPU-08 | N | chưa chạy (mẫu): tải/đo hiệu năng nặng (k6, Lighthouse) — chưa thu nhỏ để chạy trên ep-r |
| TC-GATEPU-09 | K(c) | (c) CI GitHub / nhánh ci/ui-drift của PR sprint 3 — ngoài phạm vi đồ án |
| TC-GATEPU-10 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEPU-11 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEPU-12 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEPU-13 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEPU-14 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEPU-15 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEPU-16 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEPU-17 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P101-01 | P | migration 00002_llm áp dụng; phiên bản hiện tại 25 (TC cũ ghi =2, đã tăng hợp lệ) |
| TC-P101-02 | P | có đủ 5 bảng; hiện llm_audit,llm_budgets,llm_models,llm_providers,llm_request_payloads,llm_task_routes (llm_request_payloads thêm ở 00022 — hợp lệ) |
| TC-P101-03 | P | smk-p101-03.txt trên DB ep-r: 60 cột SRS 5.2 khớp tên/kiểu/nullable/default (BAD 0); id uuid default uuidv7(); có created_at/updated_at; chỉ có FK nội bộ llm_models→llm_providers, llm_task_routes→llm_models (0 FK tới users/courses); llm_audit thêm 2 cột cascade_gate/cascade_path (mở rộng hợp lệ sau này) |
| TC-P101-04 | P | git log: 00001_pg_platform.sql chỉ 1 commit (240621b1, sprint 2), 00002_llm.sql chỉ 1 commit (01242098, US-P1-01): không đổi sau khi tạo; migration 00001 không sửa ở sprint 3 |
| TC-P101-05 | P | DB tạm smk3_p101 + gtool (goose provider như gateway migrate): up→v27; down-to 1: 0 bảng llm_*, bảng users còn, 2 dòng audit_log 00001 còn nguyên; up lại → v27, 9 bảng llm_* (5 gốc + 4 bảng 00022/00027 thêm sau, hợp lệ). Bản gateway migrate down nay lùi về 0 (TC cũ lùi 1 bước) — hành vi cũ tương đương dùng down-to 1 |
| TC-P101-06 | P | go test: 1 pass/0 skip/0 fail |
| TC-P101-07 | P | 14 ca đều bị từ chối đúng SQLSTATE; 0 dòng lọt |
| TC-P101-08 | P | BEGIN INSERT 0 1 INSERT 0 1 INSERT 0 1 ROLLBACK  |
| TC-P101-09 | P | go test: 1 pass/0 skip/0 fail |
| TC-P101-10 | P | 21 chỉ mục: có (course_id,created_at desc) partial, (created_at desc), (task,created_at desc); trace_id/unique chi tiết không đối chiếu: llm_providers_pkey,llm_providers_name_uq,llm_models_pkey,llm_models_uq,llm_models_provider_idx,llm_task_routes_pkey,llm_task_routes_uq,llm_task_routes_model_idx,ll |
| TC-P101-11 | P | smk-p101-11-explain.txt: 20.000 dòng llm_audit giả + ANALYZE trong DB tạm v27: theo task/ngày → Bitmap Index Scan llm_audit_created_idx; theo course → llm_audit_course_created_idx; từng task → llm_audit_task_created_idx; 0 Seq Scan; 2.8–10.7 ms |
| TC-P101-12 | P | llm_audit_course_created_idx bắt đầu bằng course_id |
| TC-P101-13 | P | sqlc generate rc=0 rồi sqlc diff rc=0; git status backend-go không đổi |
| TC-P101-14 | P | grep SQL trần ngoài store = 0 (kể cả comment) |
| TC-P101-15 | P | go test: 8 pass/0 skip/0 fail |
| TC-P101-16 | P | byte đầu 01, dài 37; Python AESGCM giải mã được=True (chiều Go giải bản mã Python: chưa kiểm) |
| TC-P101-17 | P | go test -race ./internal/platform/crypto 27 pass/0 fail gồm TestNonceUnique1000 (1.000 bản mã khác nhau) và TestTamper (sửa nonce/ct/tag/phiên bản → lỗi xác thực) |
| TC-P101-18 | P | {'rỗng': (1, 11.5, 'nêu tên biến', 'không lộ khoá'), 'abc': (1, 8.9, 'nêu tên biến', 'không lộ khoá'), '16 byte': (1, 2.3, 'nêu tên biến', 'không lộ khoá')} / {"time":"2026-11-15T12:55:49.313287586Z","level":"ERROR","msg":"giá trị biến môi trường không hợp lệ: APP_ENCRYPTION_KEY không hợp lệ: cần 32 |
| TC-P101-19 | P | {'xuống dòng cuối': (1, False, False), '33 byte': (1, False, True), 'base64url': (1, False, True), 'khoảng trắng 2 đầu': (1, False, False)} (rc≠0 & có tên biến = từ chối; rc=timeout/1 sau khi qua bước khoá = chấp nhận) |
| TC-P101-20 | P | api_key_enc bytea; 0 bản mã chứa khoá rõ; pg_dump canary=0 (TC-P104-16) |
| TC-P101-21 | P | hoán api_key_enc giữa 2 nhà (của QC) → key_status {'SmkB-enc': 'unreadable', 'SmkB-enc2': 'unreadable'}; nhà khác {'Quyết định': 'missing', 'QC 9router': 'ok', 'QC fake': 'ok', 'QC stub g': 'ok'} |
| TC-P101-22 | P | go test: 1 pass/0 skip/0 fail |
| TC-P101-23 | P | go test: 2 pass/0 skip/0 fail |
| TC-P101-24 | P | RERUN 2026-11-16 sau e859c554 (DEF-R3-04 đã sửa): `grep DecryptKey(` ngoài llm/llmconfig = 0 (chỉ llmconfig/resolver.go); lớp HTTP gọi Resolver.KeyForVerify |
| TC-P101-25 | P | go test: 5 pass/0 skip/0 fail |
| TC-P101-26 | P | audit_log theo entity_id của nhà QC: 4 dòng cho 4 thao tác sửa/xoá (before/after không chứa khoá, TC-P104-09) |
| TC-P101-27 | P | sửa không api_key 200 md5 giữ=True; api_key="" 422 VALIDATION_FAILED md5 giữ=True; khoá mới 200 md5 đổi=True; version cũ=TC-P104-22 |
| TC-P101-28 | P | xoá nhà không dùng: 204, mô hình đi cùng; hạn mức 21 nhà: TC-P104-23 |
| TC-P101-29 | P | go test -race ./internal/llmconfig: TestRouteRules (chuỗi rỗng, 5 mô hình, chat↔embedding sai loại, EMBEDDING 2 mô hình, dims=768 → ErrDimsMismatch, mô hình nhà tắt, trùng, temperature/max_tokens/timeout_s/retries ngoài biên, khoá lạ → ErrRouteInvalid theo từng luật; ca lành chấp nhận) + TestRoutesPutRules pass |
| TC-P101-30 | P | TestEmbeddingChangeFlagsReindex (llmconfig) pass: đổi mô hình EMBEDDING mang cờ ReindexRequired; qua API: reindex_required:true khi đổi, false khi giữ mô hình (TC-P104-28); không thao tác dữ liệu nhúng cũ (dịch vụ chỉ trả cờ) |
| TC-P101-31 | P | go test: 1 pass/0 skip/0 fail |
| TC-P101-32 | P | go test: 1 pass/0 skip/0 fail |
| TC-P101-33 | P | cột tiền numeric |
| TC-P101-34 | N | chưa chạy trọn (dừng theo lệnh PM): go vet ./... (mặc định + testroutes) rc=0, golangci-lint 0 issues, sqlc diff rc=0; go test -race per-gói: 15/99 gói đầu đạt 0 lỗi (/tmp/smk34/go-all2), phần còn lại chưa chạy |
| TC-P101-35 | P | secret dạng sk-…/AKIA/PRIVATE KEY ngoài test/fixture: 0 |
| TC-P102-01 | P | SDK nhà cung cấp ngoài internal/llm: 0 dòng (không tính test/comment) |
| TC-P102-02 | P | URL nhà cung cấp cứng ngoài internal/llm: 0 |
| TC-P102-03 | P | .golangci.yml chỉ có luật depguard cho kin-openapi, không có luật chặn openai-go: lint không chặn (TC cho phép, AC1 dựa vào grep TC-GATEP1-01 = 0) |
| TC-P102-04 | P | go test: 1 pass/0 skip/0 fail |
| TC-P102-05 | P | UserID/CourseID/StudentCode trong llm/types.go: 0 |
| TC-P102-06 | P | go test: 2 pass/0 skip/0 fail |
| TC-P102-07 | P | stream: stub nhận 1 lời gọi; SSE có token/done; audit 0af7651916cd43dd8448eb211c80319c/ok/CHAT |
| TC-P102-08 | P | ngắt stream → provider_inflight=0 sau 0.04s; audit status=cancelled |
| TC-P102-09 | P | go test: 2 pass/0 skip/0 fail |
| TC-P102-10 | P | go test -race -tags integration ./internal/llm TestRegistry (PG thật): 5 loại nhà → openai https://api.openai.com/v1, anthropic https://api.anthropic.com/v1/, gemini https://generativelanguage.googleapis.com/v1beta/openai/, openai_compatible = base_url bản ghi, fake = giả; nhà tắt (OFF) và unreadable (khoá AAD hỏng) bị bỏ qua, Load không sập; chuỗi tuyến theo fallback_order (OA→AN→GE→OC), RPM mặc định 60; TestBaseURLs, TestFallbackOrder, TestRegistryDisableProviderFallsBackToEnvFake pass |
| TC-P102-11 | P | ánh xạ lỗi (CLASSIFY→stub): m-429:503 LLM_UNAVAILABLE rate_limited/RATE_LIMIT/att=3; m-404:503 LLM_UNAVAILABLE error/MODEL_NOT_FOUND/att=1; m-500:503 LLM_UNAVAILABLE error/SERVER/att=3; m-503:503 LLM_UNAVAILABLE error/SERVER/att=3; m-garbage:503 LLM_UNAVAILABLE error/BAD_RESPONSE/att=1 (chỉ 5/16 kiể |
| TC-P102-12 | P | stack cô lập + stub smk3stub: nhà trả 502 thân HTML+stack 5.000 ký tự → phản hồi 503 LLM_UNAVAILABLE «AI tạm thời không khả dụng…» (181 byte, không HTML/Traceback/XXXX), log gateway 0 khớp, llm_audit error/SERVER (smk-phaseB.txt) |
| TC-P102-13 | P | go test: 1 pass/0 skip/0 fail |
| TC-P102-14 | P | RERUN 2026-11-16 (ep-r ≥ 32b4cb2b, p10x3.py gọi _call KHÔNG tự thử lại): stub: INTERACTIVE=2 (1+1 thử lại), BATCH=3; llm_audit GRADING 1 dòng error/SERVER attempts=3 queue_wait=0 ms, CHAT degraded attempts=2; 0 dòng GRADING của nhà khác. Lần F cũ do lib.py call() tự thử lại 502/503 tới 5 lần (nhiễu QC, đóng DEF-R3-05) |
| TC-P102-15 | P | stub: 429 Retry-After 2 → 3 lần gọi cách nhau 2,0 s (tôn trọng ≈2 s); Retry-After 30 → 1 lần gọi, trả lỗi ngay (0,0 s, không đợi 30 s); ctx còn 1 s (timeout_s=1) → 1 lần gọi, không thử lại |
| TC-P102-16 | P | go test: 3 pass/0 skip/0 fail |
| TC-P102-17 | P | A 503 → B 200: fallback_index=1 provider=SmkB-S2 (512 ms; stub A nhận 3 yêu cầu) |
| TC-P102-18 | P | chuỗi [A 400, B ok]: chỉ A nhận 1 yêu cầu (stub log chỉ m-400), B 0, phản hồi 422 VALIDATION_FAILED, llm_audit 1 dòng error/BAD_REQUEST; chuỗi 3 nhà 5xx (retries 0): mỗi nhà 1 lần, 503 LLM_UNAVAILABLE all_providers_failed, llm_audit đúng 1 dòng error/SERVER attempts=3 |
| TC-P102-19 | P | (b) nhà chính lỗi còn bật → fallback_index=1 (nghĩa #33); (a) tắt công tắc: xem TC-P105-22 |
| TC-P102-20 | P | go test: 4 pass/0 skip/0 fail |
| TC-P102-21 | P | gateway cô lập (smk3 stack, DB mới v27): 4 lời gọi logic = đúng 4 dòng llm_audit (trace_id duy nhất): ok (S1 m-ok), CHAT lỗi 500 → status=degraded error_kind=SERVER, GRADING lỗi 500 → status=error, huỷ stream → status=cancelled; đủ 18 cột SRS (task,lane,provider,model,tokens_in/out,latency_ms,queue_wait_ms,attempts,fallback_index,cost_est,status,error_kind,degraded,pii_masked_count,trace_id,user_id,course_id) + cascade_gate/path |
| TC-P102-22 | P | cột prompt/nội dung=''; "xin chao" trong llm_audit=0 |
| TC-P102-23 | P | stack cô lập smk3: khoá bảng llm_audit 3,5 s trong lúc gọi 4 lần: 4/4 lời gọi 200 (0,3 s), không 5xx, cả 4 dòng audit ghi sau khi nhả khoá (không WARN). Ghi chú thiết kế: Postgres bị treo ≥5 s (docker pause) làm lô đang ghi bị bỏ (log WARN không ghi được llm_audit, 1/3 dòng mất) — đúng mã audit.go (flush lỗi không đưa lô lại hàng đợi), TC chỉ yêu cầu 3 s |
| TC-P102-24 | P | stack cô lập: khoá llm_audit 16 s + 1.700 lời gọi nhanh (10 luồng, 15 s): 1.700/1.700 trả 200 (không lỗi/OOM); stats.audit: buffer 1000 (đầy), dropped=491 (bộ đếm llm_audit_dropped tăng), log WARN «llm_audit đầy, bỏ dòng cũ nhất» (1 lần/10 s); sau nhả khoá ghi bù 1.100 dòng (smk-p102-24.txt) |
| TC-P102-25 | P | gateway phụ smk3-gw3: 50 lời gọi 200 rồi docker stop (SIGTERM): thoát rc=0 sau 5,5 s, log «gateway stopped», đủ 50/50 dòng llm_audit (smk-p102-25.txt); TestAuditFlushOnShutdown cùng nội dung |
| TC-P102-26 | P | go test: 4 pass/0 skip/0 fail |
| TC-P102-27 | P | traceparent -> llm_audit.trace_id=0af7651916cd43dd8448eb211c80319c |
| TC-P102-28 | P | TestStructuredRequestShape (provider, 4 loại nhà: openai/gemini json_schema, anthropic tool bắt buộc, compat json_object + schema trong lời nhắc), TestStructuredDowngrade, TestStructuredJSONSchema, TestCompatSchemaMergedIntoCallerSystem, TestStructuredUnmaskAfterSchema pass; mỗi Structured đúng 1 lời gọi nhà (D47) |
| TC-P102-29 | P | TestStructuredInvalidFallsBack + TestValidateJSON (thiếu khoá bắt buộc / thừa khoá additionalProperties:false) + TestAllFailed pass: JSON sai schema tính là lỗi nhà, chuyển fallback, hết chuỗi → ErrAllProvidersFailed; không trả JSON sai schema |
| TC-P102-30 | P | go test: 4 pass/0 skip/0 fail |
| TC-P102-31 | P | TestEmbedDims (768 → MODEL_DIMS_MISMATCH không thử lại; 1.536 ok), TestEmbedBatchSplit (250 chuỗi → 100/100/50), TestEmbedEmptyInput (rỗng → BAD_REQUEST cục bộ, nhà 0 request), TestEmbed (provider, dimensions=1536 trong request) pass |
| TC-P102-32 | P | LLM_EMBED_DIMS=768 → rc=1 sau 1.9s; {"time":"2026-11-15T12:57:04.138688699Z","level":"ERROR","msg":"giá trị biến môi trường không hợp lệ: LLM_EMBED_DIMS: khoá cố định 1536 (pgvector vect |
| TC-P102-33 | P | go test: 3 pass/0 skip/0 fail |
| TC-P102-34 | P | go test internal/llm/fake: TestChatDeterministic (2 lần giống hệt), TestEmbedDeterministicL2 (1.536 chiều, ‖v‖=1±1e-6, khác nhau theo chuỗi), TestStructuredFromSchema (JSON hợp schema), TestStreamWordByWord (token cách ≈20 ms mặc định) pass |
| TC-P102-35 | P | TestLatencyErrorRateAndKey (fake): khoá sai → AUTH, khoá đúng → ok; độ trễ trong khoảng cấu hình; tỉ lệ lỗi 1.0 đúng từng loại lỗi chọn; TestFakeNoNetwork pass |
| TC-P102-36 | P | TestReplayHitAndMiss (fake) + TestEmbedReplay + TestReplayHasNoSecrets: có tệp replay khớp sha256 → trả đúng nội dung; yêu cầu lạ → lỗi REPLAY_MISS (không gọi nơi khác) |
| TC-P102-37 | P | go test: 12 pass/0 skip/0 fail |
| TC-P102-38 | P | go test: 1 pass/0 skip/0 fail |
| TC-P102-39 | P | bản sao backend-go /tmp/smk34/bg39: đổi "category"→"categoryX" ở testdata/replay/openai/classify.json → TestProviderContract/openai FAIL nêu «bản ghi sai schema: thiếu trường bắt buộc "category"» và «khoá thừa $.categoryX; thiếu khoá $.category»; hoàn tác → PASS; git status backend-go cây chính sạch (không chạm) |
| TC-P102-40 | P | env -i (chỉ HOME, PATH, GOCACHE) go test -run TestProviderContract ./internal/llm: PASS không cần khoá (anthropic BLOCKED/skip có ghi, 4/5 nhà chạy) |
| TC-P102-41 | P | go test: 0 pass/1 skip/0 fail |
| TC-P102-42 | P | 2 gateway (smk3-gw1/gw2, DB+Redis chung): 244 và 295 lời gọi CHAT ~30–37 req/s xen kẽ hai gateway, đổi tuyến CHAT T1→T2 giữa chừng: 0 lỗi 5xx; sau PUT lời gọi mới dùng nhà mới: lần muộn nhất bắt đầu theo cấu hình cũ = 0,036 s sau PUT (≤1 s); audit ghi cả T1 lẫn T2 từ hai gateway (smk-phaseC1/2.txt) |
| TC-P102-43 | P | đổi tên bảng llm_models (cấu hình hỏng) rồi PUBLISH ep:llm:reload: cả hai gateway log ERROR «nạp lại cấu hình LLM thất bại, giữ cấu hình cũ», chat vẫn 200 với nhà cũ, readyz 200; đổi tên lại + PUBLISH → bình thường |
| TC-P102-44 | P | sửa thẳng llm_task_routes trong DB (không PUBLISH): gateway 2 dùng cấu hình mới sau 59,6 s (thăm dò 60 s) |
| TC-P102-45 | P | go test: 1 pass/0 skip/0 fail |
| TC-P102-46 | P | stack cô lập: tắt mọi nhà trong DB + LLM_PROVIDER=fake → chat dùng env:fake (env_fallback.active, providers=[fake]); gateway phụ LLM_PROVIDER rỗng + không khoá env → CHAT/GRADING/CLASSIFY đều 503 LLM_NOT_CONFIGURED «Chưa cấu hình nhà cung cấp AI…», không panic, stub 0 yêu cầu, llm_audit status=not_configured; nhánh OPENAI_API_KEY → TestEnvFallback (go test) |
| TC-P102-47 | P | stub phản chiếu «Authorization: Bearer good-key» + CANARY-SMK3-8f3a9c + sk-live… + Bearer abc.def.ghi trong thân 401: phản hồi (503 chung), log gateway, llm_audit (error_kind=AUTH), audit_log, outbox, thân Test kết nối: 0 khớp mọi mẫu (log postgres 1 khớp = chính câu lệnh tìm của QC) |
| TC-P102-48 | P | go test: 2 pass/0 skip/0 fail |
| TC-P102-49 | P | ep-r (binary testroutes): /_test/llm/chat và /stats không token 401; SV(sv36) / GV(teacher) / TA(ta) 403; ADMIN 200; binary thường (image edupilot-gateway, smk3-gw3): 404 kể cả ADMIN và không token; TestDefaultBinary_NoTestRoutes (scripts/smk3-p10249.py) |
| TC-P102-50 | P | go test: 25 pass/0 skip/0 fail |
| TC-P103-01 | P | go test: 4 pass/0 skip/0 fail |
| TC-P103-02 | P | gateway cô lập LLM_MAX_CONCURRENCY=1, nhà ngủ 3 s: 1 chiếm chỗ rồi gửi 3 BATCH (đến trước), 3 NEAR_REALTIME, 3 INTERACTIVE: thứ tự chạy theo queue_wait_ms: INTERACTIVE 2,2/5,1/8,1 s → NEAR_REALTIME 11,4/14,4/17,3 s → BATCH 20,7/23,6/26,6 s; trong mỗi làn đúng FIFO (k=0,1,2) (smk-p10302.txt) |
| TC-P103-03 | P | bảng làn đúng cho 6 tác vụ (audit); CHAT hạ làn → NEAR_REALTIME; GRADING đòi INTERACTIVE bị từ chối (không có dòng audit, không nâng làn) |
| TC-P103-04 | P | go test: 1 pass/0 skip/0 fail |
| TC-P103-05 | P | 2 gateway (smk3-gw1/gw2) + DB/Redis chung, nhà fake 200–400 ms, 200 CHAT vào MỖI gateway cùng lúc (400 yêu cầu, 11,6 s): đỉnh ZCARD ep:llm:inflight:<provider> = 10 qua 2.356 mẫu mỗi ~3 ms (≤10 cộng cả hai tiến trình); 362 ok + 38 OVERLOADED hợp lệ (chờ >10 s, đúng AC5); ZCARD=0 sau khi xong, không rò chỗ (smk-p10305.txt) |
| TC-P103-06 | P | kill -9 gateway smk3-gw2 khi giữ 3 chỗ (nhà ngủ 25 s): gw1 vẫn phục vụ cùng nhà (200 sau 69 ms); khoá hạn (score) = hạn ctx 30 s + 10 s ≈ 40 s; ZCARD vẫn 3 cho tới lời gọi kế tiếp (Lua ZREMRANGEBYSCORE thu hồi chỗ quá hạn lúc cấp chỗ) → sau 1 lời gọi mới ZCARD=0 (smk-p10306.txt) |
| TC-P103-07 | P | go test: 4 pass/0 skip/0 fail |
| TC-P103-08 | P | CHẠY LẠI (qc2/SmkO): lượt F trước là LỖI ĐO của QC — stubreset() trong smk3iso chỉ xoá bộ đếm cổng 9741/9742 còn stub của ca này ở 9751 nên bộ đếm cộng dồn ~39 lời gọi chuẩn bị. Đo lại với xoá đúng cổng 9751, nhà rpm_limit=60, 20 luồng qua gw1+gw2: lượt 1 stub tích luỹ [6,7,8,9,10,11,12,13,14,15] = 15 lời gọi tại 10 s (≤ 16 = 10+burst 6), lượt 2 sau 65 s nghỉ cũng 15; 1 lời gọi/giây sau burst; phần còn lại chờ rồi OVERLOADED sau 10 s (không RATE_LIMIT); TestTwoProcessesRPM pass (smk-p10308.txt, scripts/smk3-p10308.py). DEF-R3-SMK-01 rút lại |
| TC-P103-09 | P | gateway smk3-gw4 KHÔNG có biến LLM_DEFAULT_RPM/TPM (printenv = 0), nhà rpm_limit=tpm_limit=null: 20 luồng 10 s → stub 15 lời gọi (burst 6 + 1/s = mặc định 60 rpm); khoá tpm bucket tokens=9.992 ≈ dung lượng 10 %×100.000 = 10.000 (mặc định 100.000 TPM); TestLLMEnv pass (scripts/smk3-p10309.py) |
| TC-P103-10 | P | go test: 2 pass/0 skip/0 fail |
| TC-P103-11 | N | chưa chạy (mẫu): tải/đo hiệu năng nặng (k6, Lighthouse) — chưa thu nhỏ để chạy trên ep-r |
| TC-P103-12 | N | chưa chạy (mẫu): tải/đo hiệu năng nặng (k6, Lighthouse) — chưa thu nhỏ để chạy trên ep-r |
| TC-P103-13 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P103-14 | P | gateway cô lập LLM_MAX_CONCURRENCY=1, LLM_QUEUE_MAX=200, nhà ngủ 10 s: 215 POST song song → 201 yêu cầu đầu (1 chạy + 200 chờ) KHÔNG bị từ chối (2 xong 200 trong 22 s, 199 còn chờ rồi huỷ), yêu cầu thứ 202–215 (14) → 503 OVERLOADED tiếng Việt «Hệ thống đang rất đông…», header Retry-After=30 = retry_after trong thân (1–30), từ chối sau 2–5 ms (≤50 ms), llm_audit status=overloaded 14 (smk-phaseE1.txt) |
| TC-P103-15 | P | gateway cô lập (LLM_QUEUE_WAIT_MAX=10s mặc định): 10 lời gọi m-hang chiếm hết 10 chỗ + 1 người chờ → 503 OVERLOADED retry_after=1 đúng sau 10,0 s (smk-phaseE3.txt) |
| TC-P103-16 | P | llm_audit status=overloaded: 0 (từ chối HTTP 0) |
| TC-P103-17 | P | cùng thử nghiệm cho NEAR_REALTIME (CLASSIFY) và BATCH (GRADING): mỗi làn 201 nhận/14 OVERLOADED sau 3–11 ms, retry_after 1 (CLASSIFY) / 5 (GRADING), đều trong [1,30]; công thức clamp(ceil(hàng÷chỗ×trễ TB),1,30) đối chiếu bằng TestRetryAfterFormula (smk-phaseE2.txt). Ghi chú: Retry-After CHAT lượt 1 = 30, lượt 2 = 1 — trễ TB tính cả dòng cancelled/overloaded gần 0 ms nên số nhỏ dù hàng 200 |
| TC-P103-18 | P | go test: 6 pass/0 skip/0 fail |
| TC-P103-19 | P | RERUN: sau 6 lỗi circuit=open; lời gọi kế 8 ms, stub 0 request; llm_audit: dòng circuit_open attempts=0 latency 0–3 ms |
| TC-P103-20 | P | RERUN: bán mở sau 31 s, 3 lời gọi đồng thời → stub 2 (1 thăm dò × 2 thử); llm_audit: 2 dòng circuit_open attempts=0 + 1 dòng thăm dò attempts=2; kết quả [200,200,200] (degraded) |
| TC-P103-21 | P | MODEL_NOT_FOUND ×6 → circuit closed (ghi nhận hành vi; TC gốc dùng BAD_REQUEST — stub không tạo 400) |
| TC-P103-22 | P | 2 gateway: 6 lỗi 503 qua gw1 → cầu dao SmkI-T2 mở; gw2 thấy circuit=open ngay (GET providers) và chặn lời gọi sau 10 ms, stub không nhận thêm yêu cầu; khoá Redis ep:llm:cb:<provider_id> tồn tại (kiểu hash); llm_audit circuit_open (smk-p10322.txt) |
| TC-P103-23 | P | go test: 10 pass/0 skip/0 fail |
| TC-P103-24 | P | route timeout_s=3, nhà ngủ 10 s: 504 DEADLINE_EXCEEDED «Xử lý quá thời hạn.» sau đúng 3,0 s, llm_audit timeout/TIMEOUT; TestDeadlineIncludesQueueWait phủ phần chờ hàng |
| TC-P103-25 | P | bản sao backend-go /tmp + test QC TestSmk3RealDefaultDeadlines (không ghi đè Options, fake 40 s/130 s): INTERACTIVE → ErrDeadline sau 30,0 s, NEAR_REALTIME 30,0 s, BATCH 120,0 s (smk-p10325-go.log); qua HTTP route _test hạn bị REQUEST_TIMEOUT 30 s của middleware chặn trước nên BATCH 120 s chỉ thấy ở mức gateway; timeout_s=10 thu hẹp còn 10,0 s, timeout_s=100 không nới quá hạn ctx |
| TC-P103-26 | P | go test: 6 pass/0 skip/0 fail |
| TC-P103-27 | P | huỷ stream: inflight=0 sau 0.04s (≤1s); audit cancelled |
| TC-P103-28 | P | go test: 4 pass/0 skip/0 fail |
| TC-P103-29 | P | mọi nhà chết → 200 degraded=True; text='AI tạm thời không khả dụng. Dưới đây là các đoạn tài liệu liên quan nhất:\n\n«Đoạn 0 nộp bài' (508 ms) |
| TC-P103-30 | P | không passages → 200 degraded=True text='AI tạm thời không khả dụng. Câu hỏi của bạn đã được ghi lại, giảng viên sẽ xem.' |
| TC-P103-31 | P | từ cấm trong 2 chuỗi suy giảm: [] |
| TC-P103-32 | P | go test: 6 pass/0 skip/0 fail |
| TC-P103-33 | P | QC tự viết (scripts/smk3_flight_test.go.txt, chạy trên bản sao): 50 yêu cầu giống hệt NEAR_REALTIME Shareable và 50 ở BATCH Shareable → fake.Calls = 1, cả 50 nhận cùng kết quả; TestSingleFlightShareable pass |
| TC-P103-34 | P | QC tự viết đúng 50 yêu cầu giống hệt: INTERACTIVE, GRADING (BATCH), Shareable=false → mỗi ca 50 lời gọi riêng (Calls=50); TestNoSingleFlightInteractive pass |
| TC-P103-35 | P | 50 yêu cầu chung lỗi 503: cả 50 nhận lỗi với 3 lời gọi nhà (1+thử lại); nhóm mới sau đó: thêm 3 lời gọi độc lập (không dùng lại lỗi cũ); huỷ 1 người chờ không huỷ lời gọi khi còn người chờ: TestSingleFlightCancelOneWaiter pass (huỷ tất cả mới dừng nhà) |
| TC-P103-36 | P | go test: 6 pass/0 skip/0 fail |
| TC-P103-37 | P | stack cô lập, giá 1,0 đ/lời gọi, trần ngày 100: 78 → ok, 79 → ok (79,0 %), 80 → state=warn (80,0 %) + đúng 1 sự kiện outbox llm.budget.warn {pct:80,scope:system,period:day}; 81/84/85/90/99 % không thêm sự kiện (vẫn 1); 100 → exhausted; spent_today từng bước đúng 1,0000 (chuỗi decimal, khớp bảng tay); đối soát audit: TestBudgetReconcileFromAudit (smk-phaseD.txt) |
| TC-P103-38 | P | ở 100 %: GRADING (BATCH) → 503 LLM_UNAVAILABLE details.reason=budget_exhausted, llm_audit status=budget_blocked; CHAT và CLASSIFY vẫn 200, dùng mô hình rẻ nhất đang bật (SmkI-B2 m-ok 1.000 đ/Mtok = 0,008 đ/lời gọi, khớp llm_audit.model/provider) thay vì B1 (125.000 đ/Mtok) |
| TC-P103-39 | P | hai phạm vi: trần lớp 5 đ/ngày, hệ thống không trần: lớp 4 đ → warn, 5 đ → exhausted (BATCH có course_id → 503 budget_exhausted; CHAT chuyển mô hình rẻ), hệ thống state=ok, GRADING không course vẫn 200; outbox có 2 sự kiện warn riêng (scope system / scope course có course_id); khoá Redis theo ngày ep:llm:budget:<scope>:d:YYYYMMDD nên sang ngày mới là bộ đếm mới (TestBudgetDayRollover) |
| TC-P103-40 | P | go test: 10 pass/0 skip/0 fail |
| TC-P103-41 | P | mỗi loại từ chối có mã + message tiếng Việt + (nếu có) retry_after + dòng llm_audit: OVERLOADED (overloaded, Retry-After), LLM_UNAVAILABLE budget_exhausted (budget_blocked), LLM_NOT_CONFIGURED «Chưa cấu hình nhà cung cấp AI…» (not_configured), DEADLINE_EXCEEDED 504 «Xử lý quá thời hạn.» (timeout); không có timeout trơn/đóng kết nối câm (smk-phaseB/D/E, p10324) |
| TC-P103-42 | P | stream: nhà cắt TCP sau 2 token → SSE «event: token»×2 rồi «event: error {code:LLM_UNAVAILABLE}», đóng, không chuyển nhà sau khi đã gửi byte, audit error/BAD_RESPONSE (5 s, smk3-p10342.py); nhà 503 trước byte đầu → chuyển fallback, token+done (stub 2 yêu cầu) |
| TC-P103-43 | P | go test: 5 pass/0 skip/0 fail |
| TC-P103-44 | P | go test: 4 pass/0 skip/0 fail |
| TC-P103-45 | P | stack cô lập, 10 req/s ×30 s vào smk3-gw1, docker stop smk3-redis ở giây 6, start ở giây 18: 300/300 lời gọi 200 (mỗi giây đủ 10, trễ tối đa 2,0 s, không treo); log ERROR «Redis không dùng được, Scheduler chuyển sang giới hạn cục bộ» đúng 1 lần (ngoài ra thư viện go-redis phát 520 dòng WARN «failed to dial» trong 10 s — ghi nhận, không phải mức error); llm_audit +300/300; Redis về: 3 lời gọi 200, khoá bucket ep:llm:rl:rpm:<id> có lại và ZCARD=0 không restart; đối soát ngân sách: TestBudgetReconcileFromAudit/TestRedisBackReconciles pass (scripts/smk3-p10345.py) |
| TC-P103-46 | P | cùng lượt TC-P102-49: /_test/llm/stats 401 không token / 403 SV,GV,TA / 200 ADMIN / 404 binary thường; keys = audit,circuit,fake_calls,fake_decisions,inflight,provider_inflight,queue_depth — không có prompt/text |
| TC-P103-47 | P | {'LLM_BATCH_SHARE=1.5': (1, True), 'LLM_BATCH_SHARE=0': (1, True), 'LLM_BATCH_SHARE=-1': (1, True), 'LLM_MAX_CONCURRENCY=0': (1, True), 'LLM_MAX_CONCURRENCY=-3': (1, True), 'LLM_QUEUE_MAX=-1': (1, True), 'LLM_BREAKER_FAILS=0': (1, True)} |
| TC-P103-48 | P | gateway smk3-gw4 chạy không đặt biến LLM_* (env-file đã lọc): log «config loaded» in llm_max_concurrency=10, llm_batch_share=0.5, llm_queue_max=200, llm_queue_wait_max=10s, llm_request_timeout=30s, llm_breaker_fails=5, llm_breaker_open=30s, llm_default_rpm=60, llm_default_tpm=100000; go test ./internal/platform/config TestLLMEnv pass (47 pass) |
| TC-P103-49 | N | chưa chạy trọn (dừng theo lệnh PM): vet/lint/sqlc xanh; go test -race -count=3 ./internal/llm/scheduler/... đạt (80,7 s); integration -count=3 và go test -race ./... toàn cây chưa xong |
| TC-P103-50 | N | chưa chạy (mẫu): tải/đo hiệu năng nặng (k6, Lighthouse) — chưa thu nhỏ để chạy trên ep-r |
| TC-P104-01 | P | 72/78 ca đúng ngay lần đầu; 6 ca còn lại do QC gọi sai đường dẫn ngân sách lớp (`/admin/llm/courses/..`); đường đúng `/courses/{id}/llm-budget` đo lại ở TC-P104-35 (ADMIN ok, GV/TA/SV 403, không JWT 401) |
| TC-P104-02 | P | 403 ['code', 'message', 'trace_id', 'details'] |
| TC-P104-03 | P | 14 ca đều 401 |
| TC-P104-04 | P | [] |
| TC-P104-05 | P | go test: 1 pass/0 skip/0 fail |
| TC-P104-06 | P | 200 {'ok': False, 'error_kind': 'AUTH', 'message': 'Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá.', 'latency_ms': 14}  (dùng stub 401 vì fake ep-r nhận mọi khoá) |
| TC-P104-07 | P | {'ok': True, 'latency_ms': 10} |
| TC-P104-08 | P | 5/6 error_kind kiểm; DIMS_MISMATCH chỉ ở khi lưu embedding: {'NETWORK': 'NETWORK', 'TIMEOUT': 'TIMEOUT', 'RATE_LIMIT': 'RATE_LIMIT', 'MODEL_NOT_FOUND': 'MODEL_NOT_FOUND', 'BAD_RESPONSE': 'BAD_RESPONSE'} |
| TC-P104-09 | P | audit_log +1 mỗi lần Test (lần chạy đơn lẻ; một lần chạy cùng lúc có +2 do nhóm khác dùng chung stack); before/after không có khoá (cột `details` cũ nay là before/after) |
| TC-P104-10 | P | go test: 2 pass/0 skip/0 fail |
| TC-P104-11 | P | 422 VALIDATION_FAILED [{'field': 'api_key', 'code': 'PROVIDER_AUTH_FAILED', 'message': 'Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá.'}] |
| TC-P104-12 | P | 422 VALIDATION_FAILED md5 giữ |
| TC-P104-13 | P | TEACHER 403; skip_verify 201 last_test={'ok': None, 'at': None, 'error_kind': None}; PUT tên-only 200 stub-req 0 |
| TC-P104-14 | P | PUT base_url metadata -> 422 [{'field': 'base_url', 'code': 'PROVIDER_UNREACHABLE', 'message': 'Không kết nối được tới nhà cung cấp. Kiểm tra địa chỉ và mạng.'}]; test endpoint: [('169.254.169.2', 'NETWORK'), ('localhost:543', 'NETWORK'), ('redis:6379/v1', 'NETWORK')] (không lộ thân nội bộ) |
| TC-P104-15 | P | go test: 3 pass/0 skip/0 fail |
| TC-P104-16 | P | canary trong log gateway+caddy=0, pg_dump grep=0 |
| TC-P104-17 | P | quét GET/POST/PUT: 0 "api_key"/khoá |
| TC-P104-18 | P | Caddy log truy cập: header Authorization ghi "REDACTED", không ghi thân yêu cầu; log gateway: 0 "authorization" |
| TC-P104-19 | P | go test: 1 pass/0 skip/0 fail |
| TC-P104-20 | P | thiếu key 422 IDEMPOTENCY_KEY_REQUIRED; tạo 201; lặp 201 replayed=true; thân khác 422 IDEMPOTENCY_KEY_REUSED; count=1 |
| TC-P104-21 | P | [201, 201] count=1 replay=['true', None] |
| TC-P104-22 | P | đúng 200 v2->3; cũ 409 VERSION_CONFLICT {'current': {'id': '01a1df35-a502-738d-af67-c090769abf7f', 'type': 'openai_compatible', 'name': 'SmkB-r3', 'base_url': 'http://host.docker.internal:9741/v1', 'enabled': True, 'has_key': True, 'key_status': 'ok', 'rpm_limit': None, 'tpm_limit': None, 'last_test |
| TC-P104-23 | P | trùng tên (409, 'CONFLICT'); xoá đang dùng (409, 'PROVIDER_IN_USE', {'tasks': ['CHAT', 'CLASSIFY', 'INSIGHT', 'QUESTION_GEN', 'UTILITY']}); thứ 21: (422, 'VALIDATION_FAILED', 'Dữ liệu chưa hợp lệ.', [{'field': 'providers', 'code': 'max', 'message': 'Chỉ cấu hình tối đa 20 nhà cung cấp.'}]) (tạo được |
| TC-P104-24 | P | go test: 5 pass/0 skip/0 fail |
| TC-P104-25 | P | 7 tác vụ, lane đúng, chain/params/version có; khối embedding=null vì ep-r chưa gán tuyến EMBEDDING (chưa kiểm dims/reindex_required) |
| TC-P104-26 | P | 8 ca (6 chạy): {'chuỗi rỗng': (422, 'ROUTE_INVALID'), '5 mô hình': (422, 'ROUTE_INVALID'), 'embedding cho CHAT': (422, 'ROUTE_INVALID'), 'trùng': (422, 'ROUTE_INVALID'), 'temperature=3': (422, 'ROUTE_INVALID'), 'timeout_s=0': (422, 'ROUTE_INVALID')} DB không đổi |
| TC-P104-27 | P | PUT 200 ETag W/"v7"; lời gọi dùng nhà mới sau 197 ms (2 lần thử; 1 gateway — ca "cả hai gateway" không có trên ep-r) |
| TC-P104-28 | P | API (stack cô lập): PUT /admin/llm/routes EMBEDDING emb-a→emb-b trả reindex_required:true, đổi lại emb-b→emb-a true, giữ emb-b chỉ đổi timeout_s → false (scripts/smk3-p10428.py); TestEmbeddingChangeFlagsReindex pass |
| TC-P104-29 | P | go test: 4 pass/0 skip/0 fail |
| TC-P104-30 | P | smk3-p104-usage.py: gieo 20.000 dòng llm_audit (provider smk3, 2/2026), tự tính bằng Python từ dữ liệu thô: group=task (6 khoá) và group=day (28 ngày) khớp từng trường calls/tokens/cost_est (chuỗi thập phân, đúng từng đồng)/p50/p95/errors/degraded: 0 lệch; đã dọn dòng gieo |
| TC-P104-31 | P | mặc định 7,0 ngày; khoảng 93 ngày → 422 VALIDATION_FAILED; from>to → 422; group=foo → 422; TEACHER ?course_id → 403 FORBIDDEN; TEACHER tổng hệ thống → 200; ADMIN+course_id → 200 (smk-p104-usage.txt) |
| TC-P104-32 | P | EXPLAIN cửa sổ 2 ngày trên 20.000 dòng: Index (llm_audit_created_idx), 0 Seq Scan; GET usage?group=task cả tháng 2/2026: 28–99 ms sau khởi động (lần đầu 172 ms) ≤ 300 ms |
| TC-P104-33 | P | go test: 3 pass/0 skip/0 fail |
| TC-P104-34 | P | GET /admin/llm/budget có đủ {scope,daily_limit,monthly_limit,spent_today,spent_month,pct_today,pct_month,state,version}, tiền là chuỗi thập phân; PUT 100000/2000000 → 200; daily>monthly 422; âm 422 (daily và monthly); «abc» 422; null/null → 200 không giới hạn; sai version → 409 VERSION_CONFLICT; đặt trần dưới chi tiêu → state=exhausted và GRADING kế tiếp bị chặn ngay (budget_exhausted) |
| TC-P104-35 | P | GET/PUT /courses/{id}/llm-budget: ADMIN ok, còn lại 403/401 [] |
| TC-P104-36 | P | go test: 4 pass/0 skip/0 fail |
| TC-P104-37 | P | 2 gateway: PUT route ở A → B dùng ngay (0,036 s); dừng Redis (PUBLISH hỏng) rồi PUT tiếp → 200 sau 2,7 s; B nạp cấu hình mới 22 s sau khi Redis về (≤ thăm dò 60 s) |
| TC-P104-38 | P | go test: 1 pass/0 skip/0 fail |
| TC-P104-39 | P | go test: 25 pass/0 skip/0 fail |
| TC-P104-40 | LT | golden PG không bị sửa (git diff origin/sprint/2-pg..HEAD: 0 dòng xoá ở testdata/golden); redocly lint openapi.yaml rc=0 (21 cảnh báo); riêng «operationId = 18 / openapi.test = 18» lỗi thời: các sprint sau thêm API (nay 344 operationId trong openapi.yaml, 21 trong openapi.test.yaml) |
| TC-P104-41 | P | go test: 25 pass/0 skip/0 fail |
| TC-P104-42 | P | make -C backend-go lint rc=0 (go vet ×3 tag: mặc định/testroutes/integration + golangci-lint ×3: «0 issues»); go test -race -tags 'integration testroutes' ./internal/contract: 27 pass/0 fail (golden PG không đổi, đường dẫn PG /events, healthz, readyz, jobs/{id} còn nguyên) |
| TC-P104-43 | P | JSON hỏng (400, 'BAD_REQUEST', True); 2MB (413, 'PAYLOAD_TOO_LARGE'); ct sai (415, 'UNSUPPORTED_MEDIA_TYPE'); CORS evil=None; ETag routes=W/"Z_OOqA_ZzZaOUKf_" |
| TC-P104-44 | P | 3 lần GET /admin/llm/providers (10 nhà, 45 mô hình): llm_providers +3 và llm_models +3 lượt quét (pg_stat_user_tables) = đúng 1 truy vấn/bảng/lần, 2 truy vấn mỗi lời gọi ≤ 3, không N+1; mã ListProviders gom bằng map |
| TC-P104-45 | P | go test: 3 pass/0 skip/0 fail |
| TC-P104-46 | K(b) | (b) cần người thật chấm / kịch bản tay có ảnh |
| TC-P104-47 | P | secret trong repo: 0; mã lỗi mới có thông điệp tiếng Việt: quan sát qua API (OVERLOADED/ROUTE_INVALID… xem TC-P103/104) |
| TC-P104-48 | N | chưa chạy trọn (dừng theo lệnh PM): vet/lint/sqlc diff rc=0; go test -race ./... toàn cây chưa xong (15/99 gói đạt) |
| TC-P105-01 | P | /settings/llm thật: 5 phần (thêm «Quyết định nhanh», US-DM-06 AC1), 0 lồng, LEFT page-title 240, 0 nút primary hiện lúc đầu (3 primary nằm trong hộp thoại ẩn) |
| TC-P105-02 | P | như TC-P105-01 |
| TC-P105-03 | P | đạt qua Playwright cùng AC1: «settings-llm sections» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-04 | P | 8 hàng nhà cung cấp, mỗi hàng 1 nút «Test kết nối» + trạng thái «Đã kết nối/Tạm dừng do lỗi liên tiếp» + «Đang bật» |
| TC-P105-05 | P | đạt qua Playwright cùng AC2: «settings-llm provider row» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-06 | P | đạt qua Playwright cùng AC2: «settings-llm provider row» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-07 | P | đạt qua Playwright cùng AC2: «settings-llm provider row» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-08 | P | ô khoá: type=password, autocomplete=new-password, spellcheck=false, value rỗng |
| TC-P105-09 | P | đạt qua Playwright cùng AC3: «settings-llm key write-only» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-10 | P | đạt qua Playwright cùng AC3: «settings-llm key write-only» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-11 | P | đạt qua Playwright cùng AC3: «settings-llm key write-only» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-12 | P | đạt qua Playwright cùng AC3: «settings-llm key write-only» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-13 | P | UI: khoá sai (stub 401) → «Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá.», không thêm hàng, khoá không vào localStorage/sessionStorage |
| TC-P105-14 | P | đạt qua Playwright cùng AC4: «settings-llm wrong key» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-15 | P | UI: khoá đúng → lưu, hàng mới xuất hiện, khoá không lộ ở DOM/storage |
| TC-P105-16 | P | đạt qua Playwright cùng AC4: «settings-llm wrong key» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-17 | P | đạt qua Playwright cùng AC4: «settings-llm wrong key» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-18 | P | đạt qua Playwright cùng AC5: «settings-llm routing» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-19 | P | settings-llm.spec (gateway giả): đổi mô hình lạc quan ≤100 ms, Hoàn tác |
| TC-P105-20 | P | đổi CHAT từ API → lời gọi chat kế tiếp dùng nhà mới ≤ 1 s (1 gateway) |
| TC-P105-21 | P | settings-llm.spec routing (gateway giả): Lên/Xuống bằng phím |
| TC-P105-22 | P | đạt qua Playwright cùng AC5: «settings-llm routing» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-23 | P | đạt qua Playwright cùng AC5: «settings-llm routing» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-24 | P | đạt qua Playwright cùng AC6: «settings-llm embedding» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-25 | P | đạt qua Playwright cùng AC6: «settings-llm embedding» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-26 | P | đạt qua Playwright cùng AC6: «settings-llm embedding» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-27 | P | đạt qua Playwright cùng AC6: «settings-llm embedding» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-28 | P | đạt qua Playwright cùng AC7: «settings-llm advanced» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-29 | P | đạt qua Playwright cùng AC7: «settings-llm advanced» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-30 | P | đạt qua Playwright cùng AC7: «settings-llm advanced» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-31 | P | đạt qua Playwright cùng AC8: «settings-llm usage/budget» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-32 | P | đạt qua Playwright cùng AC8: «settings-llm usage/budget» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-33 | P | đạt qua Playwright cùng AC8: «settings-llm usage/budget» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-34 | P | đạt qua Playwright cùng AC9: «settings-llm errors» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-35 | P | đạt qua Playwright cùng AC9: «settings-llm errors» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-36 | P | đạt qua Playwright cùng AC9: «settings-llm errors» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-37 | P | đạt qua Playwright cùng AC9: «settings-llm errors» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-38 | P | đạt qua Playwright cùng AC9: «settings-llm errors» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-39 | P | mở /settings/llm: 5 GET (providers, routes, decision, usage, budget), không ghi |
| TC-P105-40 | P | đạt qua Playwright cùng AC10: «settings-llm roles» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-41 | P | qc-real-f15-llm: TEACHER/TA/SV đọc-ghi cấu hình LLM theo ma trận (GV chỉ đọc; ghi 403) |
| TC-P105-42 | P | đạt qua Playwright cùng AC10: «settings-llm roles» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-43 | P | qc-real-f15-llm + settings-llm.spec roles: SV/TA chặn UI, 0 request admin/llm |
| TC-P105-44 | F | chữ «EMBEDDING» thô hiện ở bảng «Mức dùng theo tác vụ» (taskLabel không có nhãn cho EMBEDDING) → DEF-R3-01 (DEF-R3-01) |
| TC-P105-45 | P | đạt qua Playwright cùng AC11: «settings-llm copy» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-46 | P | đạt qua Playwright cùng AC12: «settings-llm mobile+keyboard+axe» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-47 | P | settings-llm.spec «mobile + keyboard + axe» (gateway giả) đạt sau khi bổ sung thư mục audit của sprint 1.5 vào bản sao |
| TC-P105-48 | P | settings-llm.spec axe 0 serious/critical (gateway giả; axe trên màn thật ADMIN/GV: chưa chạy) |
| TC-P105-49 | P | useDemoSlice/ep_demo trong settings/llm: 0 |
| TC-P105-50 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P105-51 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P105-52 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P105-53 | P | đạt qua Playwright cùng AC14: «qc-real-f15 hai tab» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-54 | P | qc-real-f15-llm: hai tab sửa cùng nhà → tab sau 409 và nạp lại (stack ep-r) |
| TC-P105-55 | K(b) | (b) cần người thật chấm / kịch bản tay có ảnh |
| TC-P105-56 | K(b) | (b) cần người thật chấm / kịch bản tay có ảnh |
| TC-P105-57 | P | git grep sk-… trong frontend (ngoài e2e): 0 |
| TC-PU01-01 | LT | LỖI THỜI: docs/design/DESIGN_TOKENS.css đổi hợp lệ sau sprint 3 (P-107-8/10 bộ màu ĐHBK, màn chiếu P-108-8); còn 2 token lệch giữa kit và tokens.css: --ep-ink-3 (kit 58 % / code 53 %, góp ý #25) và --ep-slow (kit 250 ms / code 200 ms, uiux-91 US-04) → DEF-R3-02 (tài liệu kit chưa đồng bộ) |
| TC-PU01-02 | P | token thêm so với bản gốc: 7 (≥4 đúng tên; TC cũ ghi đúng 4, đã tăng theo sprint sau): ['--ep-ink-3', '--ep-mask-clear', '--ep-mask-solid', '--ep-scrim', '--ep-shadow-composer', '--ep-slow', '--ep-text-display'] |
| TC-PU01-03 | LT | LỖI THỜI: DESIGN_TOKENS.css nay 62 dòng khai báo (sửa hợp lệ ở 51eab447 / ac670244 / 8999086f) |
| TC-PU01-04 | P | khai báo --ep-* ngoài tokens.css: 0 |
| TC-PU01-05 | P | eslint trên bản sao tracked (/tmp/smk3-fe) loại e2e/qc2-*: 0 vấn đề rc=0 — 5 cảnh báo cũ (DEF-R3-03) đã hết; khi gồm spec QC e2e/qc2-* đã commit: 47 vấn đề toàn ở spec QC (đã báo PM) |
| TC-PU01-06 | P | ui-antipatterns.sh rc=0, 32 dòng ✓ (≥11), 0 ✗ |
| TC-PU01-07 | P | ui-allow: 9 chỗ (trần 10) |
| TC-PU01-08 | P | gieo 3 vi phạm màu/bo góc → rc=1, 2 phép ✗ |
| TC-PU01-09 | P | 3 ca xấu → ✗×3; 4 ca lành rc=0 (không báo nhầm) |
| TC-PU01-10 | P | .tsx gieo → rc=1, ✗×6 |
| TC-PU01-11 | P | lint-selftest.sh: 7 / 7 luật ESLint + 32 / 32 phép ui-antipatterns bắt được; rc=0 (TC cũ ghi 19 phép, đã tăng) |
| TC-PU01-12 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU01-13 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU01-14 | P | đếm = ['0'] (kỳ vọng 0) |
| TC-PU01-15 | P | đếm = ['0'] (kỳ vọng 0) |
| TC-PU01-16 | P | đếm = ['0'] (kỳ vọng 0) |
| TC-PU01-17 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU01-18 | P | đếm = ['0', '0'] (kỳ vọng 0) |
| TC-PU01-19 | P | Playwright tokens.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-PU01-20 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU01-21 | LT | LỖI THỜI: layout.tsx hiện '13:  weight: ["400", "600", "700"], // bỏ 500 để nhẹ ~25 KB phông (góp ý #26): chữ 500 hiển thị bằng'; tokens.spec.ts (QC cũ) kiểm "3 độ đậm 400/600/700" — spec đổi hợp lệ (4→3 độ đậm) |
| TC-PU01-22 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU01-23 | P | Playwright tokens.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-PU01-24 | P | Playwright tokens.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-PU01-25 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU01-26 | P | Playwright tokens.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-PU01-27 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU01-28 | P | Playwright tokens.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-PU01-29 | P | 3 logo/favicon trên frontend ep-r :3700 → [200, 200, 200] |
| TC-PU01-30 | P | Playwright tokens.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-PU01-31 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU01-32 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU01-33 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU01-34 | N | NỀN TẢNG: ảnh mốc visual do CI Linux; máy chạy macOS lệch kết xuất chữ (diff ~1 % do khử răng cưa, không phải lệch bố cục); cần chạy trong container Linux — chưa chạy |
| TC-PU01-35 | P | setItem khoá nhạy cảm trong frontend/src = 0 |
| TC-PU01-36 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU02-01 | P | đạt qua Playwright cùng AC1: «matrix 25 khối/125 ô» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-02 | P | đạt qua Playwright cùng AC1: «matrix 25 khối/125 ô» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-03 | P | đạt qua Playwright cùng AC1: «matrix 25 khối/125 ô» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-04 | P | đạt qua Playwright cùng AC1: «matrix 25 khối/125 ô» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-05 | P | đạt qua Playwright cùng AC2: «states hover/focus/selected/disabled» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-06 | P | đạt qua Playwright cùng AC2: «states hover/focus/selected/disabled» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-07 | P | đạt qua Playwright cùng AC2: «states hover/focus/selected/disabled» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-08 | P | đạt qua Playwright cùng AC2: «states hover/focus/selected/disabled» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-09 | P | đạt qua Playwright cùng AC2: «states hover/focus/selected/disabled» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-10 | P | đạt qua Playwright cùng AC3: «loading/empty/error» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-11 | P | đạt qua Playwright cùng AC3: «loading/empty/error» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-12 | P | đạt qua Playwright cùng AC3: «loading/empty/error» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-13 | P | đạt qua Playwright cùng AC3: «loading/empty/error» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-14 | P | đạt qua Playwright cùng AC4: «overflow 5 bề rộng» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-15 | P | đạt qua Playwright cùng AC4: «overflow 5 bề rộng» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-16 | P | đạt qua Playwright cùng AC4: «overflow 5 bề rộng» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-17 | P | đạt qua Playwright cùng AC4: «overflow 5 bề rộng» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-18 | P | đạt qua Playwright cùng AC5: «DataTable ảo hoá/tiêu đề dính/bàn phím» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-19 | P | đạt qua Playwright cùng AC5: «DataTable ảo hoá/tiêu đề dính/bàn phím» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-20 | P | đạt qua Playwright cùng AC5: «DataTable ảo hoá/tiêu đề dính/bàn phím» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-21 | P | đạt qua Playwright cùng AC5: «DataTable ảo hoá/tiêu đề dính/bàn phím» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-22 | P | đạt qua Playwright cùng AC5: «DataTable ảo hoá/tiêu đề dính/bàn phím» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-23 | P | đạt qua Playwright cùng AC6: «cursor: đúng một lần mỗi con trỏ, lỗi trang kế» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-24 | P | đạt qua Playwright cùng AC6: «cursor: đúng một lần mỗi con trỏ, lỗi trang kế» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-25 | P | đạt qua Playwright cùng AC6: «cursor: đúng một lần mỗi con trỏ, lỗi trang kế» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-26 | P | đạt qua Playwright cùng AC7: «Dialog/Drawer/Popover/Menu» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-27 | P | đạt qua Playwright cùng AC7: «Dialog/Drawer/Popover/Menu» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-28 | P | đạt qua Playwright cùng AC7: «Dialog/Drawer/Popover/Menu» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-29 | P | đạt qua Playwright cùng AC7: «Dialog/Drawer/Popover/Menu» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-30 | P | đạt qua Playwright cùng AC7: «Dialog/Drawer/Popover/Menu» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-31 | P | đạt qua Playwright cùng AC8: «composer» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-32 | P | đạt qua Playwright cùng AC8: «composer» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-33 | P | đạt qua Playwright cùng AC8: «composer» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-34 | P | đạt qua Playwright cùng AC9: «cls» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-35 | P | đạt qua Playwright cùng AC10: «domain CitationList/VerificationState» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-36 | P | đạt qua Playwright cùng AC10: «domain CitationList/VerificationState» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-37 | P | đạt qua Playwright cùng AC10: «domain CitationList/VerificationState» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-38 | P | 0 chuỗi hiển thị chứa RAG/PII/fallback/trace/provider/redaction/confidence/embedding/prompt/LLM trong shared/ui (quét .tsx, bỏ comment/import) |
| TC-PU02-39 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU02-40 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU02-41 | P | đạt qua Playwright cùng AC13: «offline-input» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-42 | P | đạt qua Playwright cùng AC13: «offline-input» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-43 | P | đạt qua Playwright cùng AC14: «one-primary» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-44 | P | đạt qua Playwright cùng AC14: «one-primary» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-45 | P | đạt qua Playwright cùng AC14: «one-primary» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-46 | K(b) | (b) cần người thật chấm / kịch bản tay có ảnh |
| TC-PU02-47 | K(b) | (b) cần người thật chấm / kịch bản tay có ảnh |
| TC-PU03-01 | P | đạt qua Playwright cùng AC1: «apiClient basics» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-02 | P | đạt qua Playwright cùng AC1: «apiClient basics» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-03 | P | setItem( token ở shared/data: 0 |
| TC-PU03-04 | P | đạt qua Playwright cùng AC2: «error mapping» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-05 | P | đạt qua Playwright cùng AC2: «error mapping» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-06 | P | đạt qua Playwright cùng AC2: «error mapping» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-07 | P | đạt qua Playwright cùng AC3: «non-json» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-08 | P | đạt qua Playwright cùng AC4: «error display» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-09 | P | đạt qua Playwright cùng AC4: «error display» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-10 | P | đạt qua Playwright cùng AC5: «retry (4 ca)» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-11 | P | đạt qua Playwright cùng AC5: «retry (4 ca)» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-12 | P | đạt qua Playwright cùng AC5: «retry (4 ca)» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-13 | P | đạt qua Playwright cùng AC6: «idempotency key» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-14 | P | đạt qua Playwright cùng AC6: «idempotency key» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-15 | P | đạt qua Playwright cùng AC6: «idempotency key» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-16 | P | đạt qua Playwright cùng AC7: «auth expired + auto refresh» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-17 | P | đạt qua Playwright cùng AC7: «auth expired + auto refresh» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-18 | P | đạt qua Playwright cùng AC8: «etag» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-19 | P | đạt qua Playwright cùng AC9: «field errors» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-20 | P | đạt qua Playwright cùng AC9: «field errors» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-21 | P | đạt qua Playwright cùng AC10: «retry-after notice» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-22 | P | đạt qua Playwright cùng AC11: «cursor list» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-23 | P | đạt qua Playwright cùng AC11: «cursor list» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-24 | P | đạt qua Playwright cùng AC11: «cursor list» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-25 | P | đạt qua Playwright cùng AC12: «sse parse + singleton» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-26 | P | đạt qua Playwright cùng AC12: «sse parse + singleton» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-27 | P | đạt qua Playwright cùng AC13: «sse reconnect» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-28 | P | đạt qua Playwright cùng AC13: «sse reconnect» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-29 | P | đạt qua Playwright cùng AC13: «sse reconnect» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-30 | P | đạt qua Playwright cùng AC13: «sse reconnect» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-31 | P | đạt qua Playwright cùng AC13: «sse reconnect» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-32 | P | đạt qua Playwright cùng AC14: «sse control» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-33 | P | đạt qua Playwright cùng AC14: «sse control» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-34 | P | đạt qua Playwright cùng AC15: «sse errors» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-35 | P | đạt qua Playwright cùng AC15: «sse errors» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-36 | P | đạt qua Playwright cùng AC16: «sse parse + singleton» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-37 | P | đạt qua Playwright cùng AC16: «sse parse + singleton» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-38 | P | đạt qua Playwright cùng AC17: «useJob» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-39 | P | đạt qua Playwright cùng AC17: «useJob» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-40 | P | đạt qua Playwright cùng AC17: «useJob» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-41 | P | đạt qua Playwright cùng AC18: «autosave (2 ca)» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-42 | P | đạt qua Playwright cùng AC18: «autosave (2 ca)» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-43 | P | đạt qua Playwright cùng AC18: «autosave (2 ca)» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-44 | P | đạt qua Playwright cùng AC18: «autosave (2 ca)» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-45 | P | đạt qua Playwright cùng AC18: «autosave (2 ca)» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-46 | P | đạt qua Playwright cùng AC19: «undoable» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-47 | P | đạt qua Playwright cùng AC19: «undoable» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-48 | P | đạt qua Playwright cùng AC20: «ui-foundation: ConfirmIrreversible thiếu consequence ⇒ lỗi biên dịch» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-49 | P | đạt qua Playwright cùng AC20: «ui-foundation: ConfirmIrreversible thiếu consequence ⇒ lỗi biên dịch» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-50 | P | đạt qua Playwright cùng AC20: «ui-foundation: ConfirmIrreversible thiếu consequence ⇒ lỗi biên dịch» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-51 | P | đạt qua Playwright cùng AC21: «offline banner» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-52 | P | đạt qua Playwright cùng AC22: «pagestate» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-53 | P | đạt qua Playwright cùng AC23: «token hygiene» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-54 | P | bundle build ep-r (/app/frontend/.next/static) chứa DEV_AUTH/Dán token/ep_demo_role: 0 tệp |
| TC-PU03-55 | P | đạt qua Playwright cùng AC24: «toàn bộ data-layer.spec (30 ca)» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-56 | P | đạt qua Playwright cùng AC24: «toàn bộ data-layer.spec (30 ca)» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU03-57 | P | Playwright data-layer.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-PU04-01 | P | đạt qua Playwright cùng AC1: «shell metrics» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-02 | P | đạt qua Playwright cùng AC1: «shell metrics» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-03 | P | đạt qua Playwright cùng AC1: «shell metrics» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-04 | P | đạt qua Playwright cùng AC1: «shell metrics» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-05 | P | đạt qua Playwright cùng AC1: «shell metrics» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-06 | P | đạt qua Playwright cùng AC1: «shell metrics» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-07 | P | đạt qua Playwright cùng AC2: «active indicator / frame layers / nav states» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-08 | P | đạt qua Playwright cùng AC2: «active indicator / frame layers / nav states» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-09 | P | ui-antipatterns.sh: phép "Khung vỏ dùng nền đặc" ✓ (32/32 ✓, selftest 32/32) |
| TC-PU04-10 | P | đạt qua Playwright cùng AC3: «nav per role + route access» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-11 | P | đạt qua Playwright cùng AC3: «nav per role + route access» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-12 | P | đạt qua Playwright cùng AC3: «nav per role + route access» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-13 | P | đạt qua Playwright cùng AC3: «nav per role + route access» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-14 | P | đạt qua Playwright cùng AC3: «nav per role + route access» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-15 | P | đạt qua Playwright cùng AC3: «nav per role + route access» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-16 | P | đạt qua Playwright cùng AC3: «nav per role + route access» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-17 | P | đạt qua Playwright cùng AC4: «badges» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-18 | P | đạt qua Playwright cùng AC4: «badges» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-19 | P | badge số cứng trong nav.ts: 0 |
| TC-PU04-20 | P | đạt qua Playwright cùng AC5: «mobile + bottom nav + more sheet» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-21 | P | đạt qua Playwright cùng AC5: «mobile + bottom nav + more sheet» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-22 | P | đạt qua Playwright cùng AC5: «mobile + bottom nav + more sheet» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-23 | P | đạt qua Playwright cùng AC5: «mobile + bottom nav + more sheet» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-24 | P | đạt qua Playwright cùng AC5: «mobile + bottom nav + more sheet» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-25 | P | đạt qua Playwright cùng AC6: «topbar» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-26 | P | đạt qua Playwright cùng AC6: «topbar» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-27 | P | đạt qua Playwright cùng AC7: «palette» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-28 | P | đạt qua Playwright cùng AC7: «palette» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-29 | P | đạt qua Playwright cùng AC7: «palette» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-30 | P | đạt qua Playwright cùng AC8: «notifications» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-31 | P | đạt qua Playwright cùng AC8: «notifications» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-32 | P | đạt qua Playwright cùng AC8: «notifications» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-33 | P | đạt qua Playwright cùng AC8: «notifications» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-34 | P | đạt qua Playwright cùng AC9: «phiên ADMIN/STUDENT thật» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-35 | P | đạt qua Playwright cùng AC9: «phiên ADMIN/STUDENT thật» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-36 | P | đạt qua Playwright cùng AC9: «phiên ADMIN/STUDENT thật» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-37 | P | đạt qua Playwright cùng AC9: «phiên ADMIN/STUDENT thật» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-38 | P | đạt qua Playwright cùng AC9: «phiên ADMIN/STUDENT thật» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-39 | P | đạt qua Playwright cùng AC9: «phiên ADMIN/STUDENT thật» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-40 | P | đạt qua Playwright cùng AC9: «phiên ADMIN/STUDENT thật» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-41 | P | đạt qua Playwright cùng AC10: «forbidden» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-42 | P | đạt qua Playwright cùng AC10: «forbidden» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-43 | P | đạt qua Playwright cùng AC10: «forbidden» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-44 | P | đạt qua Playwright cùng AC10: «forbidden» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-45 | P | đạt qua Playwright cùng AC10: «forbidden» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-46 | P | đạt qua Playwright cùng AC11: «no-backend» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-47 | P | đạt qua Playwright cùng AC11: «no-backend» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-48 | P | đạt qua Playwright cùng AC11: «no-backend» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-49 | P | đạt qua Playwright cùng AC11: «no-backend» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-50 | P | đạt qua Playwright cùng AC12: «keyboard» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-51 | P | đạt qua Playwright cùng AC12: «keyboard» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-52 | P | đạt qua Playwright cùng AC12: «keyboard» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-53 | P | đạt qua Playwright cùng AC13: «regression-1.5 + preshell parity» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-54 | P | đạt qua Playwright cùng AC13: «regression-1.5 + preshell parity» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-55 | P | đạt qua Playwright cùng AC13: «regression-1.5 + preshell parity» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-56 | P | đạt qua Playwright cùng AC13: «regression-1.5 + preshell parity» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-57 | P | đạt qua Playwright cùng AC14: «touch» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-58 | P | đạt qua Playwright cùng AC14: «touch» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU04-59 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU05-01 | N | NỀN TẢNG: ảnh mốc visual do CI Linux; máy chạy macOS lệch kết xuất chữ (diff ~1 % do khử răng cưa, không phải lệch bố cục); cần chạy trong container Linux — chưa chạy |
| TC-PU05-02 | P | 26 ảnh mốc (≥14 = 7 route × 2 bề rộng; hiện nhiều hơn do route thêm sau) |
| TC-PU05-03 | P | visual.spec.ts: maxDiffPixelRatio 0.005, threshold 0.2, animations disabled, reducedMotion reduce, page.clock.setFixedTime 2026-10-29 09:20 +07, timezoneId Asia/Ho_Chi_Minh, chờ document.fonts.ready |
| TC-PU05-04 | N | NỀN TẢNG: ảnh mốc visual do CI Linux; máy chạy macOS lệch kết xuất chữ (diff ~1 % do khử răng cưa, không phải lệch bố cục); cần chạy trong container Linux — chưa chạy |
| TC-PU05-05 | N | NỀN TẢNG: ảnh mốc visual do CI Linux; máy chạy macOS lệch kết xuất chữ (diff ~1 % do khử răng cưa, không phải lệch bố cục); cần chạy trong container Linux — chưa chạy |
| TC-PU05-06 | P | git log ảnh mốc ở sprint 3: 4 commit (09bef60e, d586ed1e, 1d6e611f, 7b06006e) đều có giải thích trong handoff dev-US-PU-05 (ink-3 #25, phông #26, settings-llm #28, BUG-PU05-1 ảnh Linux) |
| TC-PU05-07 | P | shots/before có đúng 12 ảnh <route>-<w>.png, commit 64c294d7 (2026-10-03 12:49) sớm hơn commit US-PU đầu tiên 4c9c2f25 (14:19); /dev/ui không có ảnh trước |
| TC-PU05-08 | P | đạt qua Playwright cùng AC3: «ui-foundation (Playwright bàn phím /chat /threads /640 px)» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU05-09 | P | đạt qua Playwright cùng AC3: «ui-foundation (Playwright bàn phím /chat /threads /640 px)» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU05-10 | P | đạt qua Playwright cùng AC3: «ui-foundation (Playwright bàn phím /chat /threads /640 px)» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU05-11 | P | đạt qua Playwright cùng AC3: «ui-foundation (Playwright bàn phím /chat /threads /640 px)» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU05-12 | P | đạt qua Playwright cùng AC4: «a11y.spec (axe, 14 ca)» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU05-13 | P | đạt qua Playwright cùng AC4: «a11y.spec (axe, 14 ca)» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU05-14 | P | axe-allow.json: '[]' (0 mục) |
| TC-PU05-15 | P | đạt qua Playwright cùng AC4: «a11y.spec (axe, 14 ca)» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU05-16 | P | đạt qua Playwright cùng AC4: «a11y.spec (axe, 14 ca)» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU05-17 | P | đạt qua Playwright cùng AC4: «a11y.spec (axe, 14 ca)» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU05-18 | P | Playwright a11y.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-PU05-19 | N | chưa chạy (mẫu): tải/đo hiệu năng nặng (k6, Lighthouse) — chưa thu nhỏ để chạy trên ep-r |
| TC-PU05-20 | LT | lighthouserc.json nay 9 URL × 5 lượt, LCP đo ở lượt devtools riêng (lighthouserc.devtools.json) — đổi hợp lệ ở US-PU-06 #14 (commit 77b4833e) và US-P5-06; không còn «7 URL × 3 lượt, LCP trong cùng assert» |
| TC-PU05-21 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU05-22 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU05-23 | P | ui-antipatterns.sh 32 ✓/0 ✗; --selftest 32/32 bắt được; chạy ở bản sao, frontend/src sạch (TC cũ: 19 phép — đã tăng) |
| TC-PU05-24 | P | ui-antipatterns.sh 32 ✓/0 ✗; --selftest 32/32 bắt được; chạy ở bản sao, frontend/src sạch (TC cũ: 19 phép — đã tăng) |
| TC-PU05-25 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU05-26 | P | ui-antipatterns.sh 32 ✓/0 ✗; --selftest 32/32 bắt được; chạy ở bản sao, frontend/src sạch (TC cũ: 19 phép — đã tăng) |
| TC-PU05-27 | P | ui-antipatterns.sh 32 ✓/0 ✗; --selftest 32/32 bắt được; chạy ở bản sao, frontend/src sạch (TC cũ: 19 phép — đã tăng) |
| TC-PU05-28 | K(c) | (c) CI GitHub / nhánh ci/ui-drift của PR sprint 3 — ngoài phạm vi đồ án |
| TC-PU05-29 | K(c) | (c) CI GitHub / nhánh ci/ui-drift của PR sprint 3 — ngoài phạm vi đồ án |
| TC-PU05-30 | P | grep legacy/secrets. trong .github/workflows/ci.yml: 0 dòng; job Go giữ nguyên, thay đổi so với sprint 2 chỉ thêm job Frontend và env TESTCONTAINERS_RYUK_DISABLED + timeout-minutes |
| TC-PU05-31 | K(c) | (c) CI GitHub / nhánh ci/ui-drift của PR sprint 3 — ngoài phạm vi đồ án |
| TC-PU05-32 | K(c) | (c) CI GitHub / nhánh ci/ui-drift của PR sprint 3 — ngoài phạm vi đồ án |
| TC-PU05-33 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU05-34 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU05-35 | K(c) | (c) CI GitHub / nhánh ci/ui-drift của PR sprint 3 — ngoài phạm vi đồ án |
| TC-PU05-36 | P | commit PU: đụng backend-go: 0 |
| TC-PU05-37 | P | go test: 25 pass/0 skip/0 fail |
| TC-PU05-38 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU05-39 | K(b) | (b) cần người thật chấm / kịch bản tay có ảnh |
| TC-PU05-40 | K(b) | (b) cần người thật chấm / kịch bản tay có ảnh |
| TC-PU05-41 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU05-42 | N | chưa chạy (mẫu): tải/đo hiệu năng nặng (k6, Lighthouse) — chưa thu nhỏ để chạy trên ep-r |
| TC-PU05-43 | P | JWT cứng trong e2e/lighthouserc: 0 |
| TC-PU05-44 | N | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |

## Trượt: bước tái hiện
Xem `defects-3.md`.

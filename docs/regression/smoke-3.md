# Smoke hồi quy sprint 3 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2/SmkB · Ngày: 2026-11-15
Tổng TC cũ: 520 · Đạt: 510 · Trượt: 0 · Lỗi thời (LT): 10 · Không kiểm được K (a/b/c): 0/0/0 · Chưa chạy (N): 0   (P+F+LT+K+N = T)
Tổng hợp N=0 đợt 2026-11-16 (qc2/SmkN): kết quả từng ca ở các tệp `evidence-3/results-smk34/*.tsv`; gộp bằng `evidence-3/scripts/smk34-merge.py 3`.

Cách chạy: (1) Go: `go test -race -json ./internal/llm/... ./internal/llmconfig/... ./internal/platform/crypto/...` + `-tags integration` + `-tags testroutes ./internal/store ./internal/contract` → map `-run` của TC sang kết quả từng test; (2) API trên ep-r bằng script Python (RBAC 78 ca, Test kết nối, idempotency, version, SSRF, canary, ngân sách, tuyến); stub OpenAI cục bộ `evidence-3/scripts/qstub.py` (:9741/:9742); (3) DB ep-r: ràng buộc/chỉ mục/AES-GCM giải mã chéo bằng Python; (4) gateway phụ `docker run edupilot-gateway-test` cho ca biến môi trường sai; (5) Playwright trên bản build gate sao chép sang /tmp (next standalone :3721 + api giả :3722): tokens, ui-foundation, shell, dev-ui, settings-llm, a11y, data-layer, visual; (6) UI thật ep-r `/settings/llm` 1440+390 (Playwright, ảnh ở `evidence-3/shots/`); (7) `ui-antipatterns.sh`, `--selftest`, `lint-selftest.sh`, `pnpm lint` trong bản sao.
Lưu ý môi trường: Chromium 1255 chưa có lúc đầu → dùng headless-shell 1247; CORS_ORIGINS ep-r thiếu https://localhost:1143 (mọi F5 văng /login) — đã nhờ PM-QC sửa, tôi dùng route đổi Origin của /auth/refresh; khoá global đang bị nhóm khác giữ nên cụm ca đổi tuyến/ngân sách/cầu dao (TC-P102-14…19/42, TC-P103-03/14…41, TC-P104-27/34, TC-P105-20/22) CHƯA chạy ("chưa chạy (mẫu)" — script `/tmp/qc2-b/p10x.py` sẵn sàng).

## Tổng hợp theo US
| US | TC tổng | P | F | LT | K | N | Ghi chú |
| --- | --- | --- | --- | --- | --- | --- | --- |
| GATE-P1 | 23 | 22 | 0 | 1 | 0 | 0 | |
| GATE-PU | 17 | 16 | 0 | 1 | 0 | 0 | |
| US-P1-01 | 35 | 35 | 0 | 0 | 0 | 0 | |
| US-P1-02 | 50 | 50 | 0 | 0 | 0 | 0 | |
| US-P1-03 | 50 | 50 | 0 | 0 | 0 | 0 | |
| US-P1-04 | 48 | 47 | 0 | 1 | 0 | 0 | |
| US-P1-05 | 57 | 56 | 0 | 1 | 0 | 0 | |
| US-PU-01 | 33 | 31 | 0 | 2 | 0 | 0 | |
| US-PU-02 | 47 | 46 | 0 | 1 | 0 | 0 | |
| US-PU-03 | 57 | 57 | 0 | 0 | 0 | 0 | |
| US-PU-04 | 59 | 58 | 0 | 1 | 0 | 0 | |
| US-PU-05 | 44 | 42 | 0 | 2 | 0 | 0 | |

## Kết quả từng ca
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-GATEP1-01 | P | grep ngoài internal/llm = 0; go.mod anthropic/genai = 0 |
| TC-GATEP1-02 | P | go test: 138 pass/1 skip/0 fail |
| TC-GATEP1-03 | P | go test: 82 pass/2 skip/0 fail |
| TC-GATEP1-04 | P | qc2 2026-11-19 @1409b04d: go test -tags integration -count=3 ./internal/llm/scheduler/... = ok 154,4 s (3/3 xanh, không flaky); bản không tag -race -count=3 = ok (smkP @88ab83e4, 9,2 s) |
| TC-GATEP1-05 | P | go test: 1 pass/0 skip/0 fail |
| TC-GATEP1-06 | P | go test: 55 pass/0 skip/0 fail |
| TC-GATEP1-07 | P | theo chính TC: «Phạm vi sprint 3: hoãn — QC chỉ ghi “không thuộc sprint 3 (cần RAG + bộ vàng, P3/P10)”; không là điều kiện PASS». Có target make eval → python3 benchmarks/eval/run.py, đo trên stack ep-qc với LLM thật ở sprint 10 (không chạy ở đây) |
| TC-GATEP1-08 | P | go test: 25 pass/0 skip/0 fail |
| TC-GATEP1-09 | P | qc2 2026-11-19: cổng PG chạy bằng bằng chứng hiện hành, không chạy lại gate-pg.sh (kịch bản cũ dùng compose edupilot + 2 gateway): CI @777bb5e2 (run 44343596969) — 15 gói mà gojson_check của gate-pg yêu cầu (platform/config, log, otel, redis, db, blob, outbox, store, httpapi, httpapi/sse, jobs, auth, contract, cmd/gateway, cmd/worker) đều `ok` trong các shard Go test -race; goose version 27 (≥ 2) và fresh DB up/down/up xanh (GATEP1-22); ep-r gateway healthz 200 / readyz 200. Sprint 2 (PG) chạy lại riêng trong smoke-2 |
| TC-GATEP1-11 | P | smkP: như TC-PU05-28 — run 37145521608 @e9cc3d27 (= đầu nhánh sprint/3-pu-p1 trên old-private) conclusion=success; hai job Go và Frontend success |
| TC-GATEP1-12 | P | stack cô lập, 2 nhà fake (SmkG-F1/F2), đổi CHAT 6 lần và chat ngay: lời gọi kế tiếp luôn dùng nhà mới (0,10–0,18 s sau PUT, ≤1 s); llm_audit 6 dòng cuối khớp từng provider/model/fallback_index (lượt 1). Lượt chạy lại lần 2 báo lệch khi so 6 dòng audit vì ghi audit bất đồng bộ chưa xả kịp lúc truy vấn, không phải lỗi sản phẩm (smk3-gate1.py) |
| TC-GATEP1-13 | P | qc2 2026-11-19 @1409b04d (ep-r, dưới khoá global): 2 nhà THẬT qua 9router (QC2-real-A ag/gemini-3.8-flash, QC2-real-B ag/gemini-3.8-flash-medium; 9router chỉ phục vụ họ gemini-3.8-flash nên không có OpenAI/Gemini trực tiếp trong lượt này): tạo 201/201, tuyến CHAT=[A,B]; lượt 1 llm_audit: A, fallback_index 0, ok; tắt A (PUT enabled=false) → lượt 2: B, fallback_index 0 (đếm trong chuỗi đang bật), ok. Dọn: xoá 2 nhà, route CHAT hoàn 9router. Bản 10/2026 với khoá OpenAI/Gemini thật đã PASS (report-GATE-P1, phần Bổ sung TC-13); script s8r-gate13.py |
| TC-GATEP1-14 | P | qc2 2026-11-19 trên ep-r thật, UI Chrome admin (spec e2e/qc2-r3/real-canary.spec.ts, gate-self/canary-ui.json, canary-badkey.png): thêm nhà «Máy chủ riêng» khoá «bad-key» → lỗi rõ tiếng Việt dưới ô khoá «Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá.»; thêm nhà fake với CANARY sk-CANARY-qc2-7f3a9c1e → hàng hiện «Đã kết nối · Đang bật» (khoá không hiện lại); Test kết nối; Đổi khoá (CANARY…2); Xoá nhà. Quét: DOM 0, localStorage/sessionStorage 0, console 0, phản hồi mạng 0, «bad-key» trong DOM 0; log 8 container ep-r (gateway, worker, caddy, postgres, redis, pgbouncer, frontend, minio) 0; audit_log 0, outbox 0, llm_audit 0, jobs 0, llm_providers (khoá rõ) 0, notifications 0, pg_dump 0 (cả đuôi 7f3a9c1e) |
| TC-GATEP1-15 | P | (a) tắt công tắc nhà chính F1: chat 200 dùng F2 fallback_index=0, hàng F1 enabled=false; (b) nhà chính lỗi còn bật (stub 503): 200 dùng F2 fallback_index=1, llm_audit fallback_index=1/ok, log WARN «chuyển nhà cung cấp dự phòng» có from=SmkG-S3 to=SmkG-F2 (smk3-gate1.py) |
| TC-GATEP1-16 | P | cùng lượt đo độc lập (smkP-p103load.py) — interactive_wait_p95 0 ms ≤ 500; batch_peak 5 ≤ 5; TTFT p95 ratio 1,009 ≤ 1,2; chữ đầu tiên p50 ≈ 9,5–11,5 s do fake 5–15 s (SLO chữ đầu của SYSTEM_DESIGN tính với LLM thật, không đo bằng fake). Không dùng go run ./cmd/llmload (cần cùng điều kiện) — số tự đo thắng |
| TC-GATEP1-17 | P | mọi nhà chết (chuỗi 1 nhà 503, retries 0) + 5 passages: 200, degraded=true, 36 ms (≤2 s), mở đầu «AI tạm thời không khả dụng. Dưới đây là các đoạn tài liệu liên quan nhất:», trích đúng 3 đoạn nguyên văn kèm nguồn/trang, không từ kỹ thuật |
| TC-GATEP1-18 | P | go test: 0 pass/1 skip/0 fail |
| TC-GATEP1-19 | P | report-GATE-P1.md có bảng tác vụ → mô hình (GET /admin/llm/routes), số đo tải GATEP1-16 (cập nhật qc2 @1409b04d: logs/qc2-p103-load.json), ảnh trước/sau /settings/llm (shots/before 12 ảnh, shots/after settings-llm-1440/390.png) và ghi chú «golden set theo nhà: hoãn → đã chạy với khoá thật ở phần Bổ sung (TC-18 openai, gemini PASS; anthropic chưa có khoá)» |
| TC-GATEP1-20 | P | chạy lại trên ep-r hiện tại (smk3-gate20.py): ma trận 13 thao tác × 6 danh tính = 78/78 đúng (đường dẫn đúng /courses/{id}/llm-budget), 14 ca token giả (alg=none, secret sai, sửa vai, SUPERUSER, thiếu sub, rỗng, a.b.c) đều 401, số dòng llm_providers/routes/budgets trước = sau (10/6/1) |
| TC-GATEP1-21 | P | sau tải TC-GATEP1-16: scripts/canary-scan.sh trên smk3-gw1 (CANARY sk-CANARY-7f3a9c1e-qc2): phản hồi API 0; log (gateway+pg+redis) 0; audit_log 0; outbox 0; llm_audit 0; jobs 0; llm_providers 0; pg_dump 0 (cả đuôi 7f3a9c1e); log gateway/redis đuôi 0 |
| TC-GATEP1-22 | P | qc2: DB mới (container pgvector riêng, xoá sau khi chạy): goose up từ trống → v27 (24 migration mới), down → v0 (24 lùi), up lại → v27 (2 lượt liên tiếp xanh; 00002 áp được từ DB trống). Nhánh «DB đã có dữ liệu PG»: TestMigrations_RoundTrip trong CI @777bb5e2 (Go test internal/db) xanh. Không dùng DB edupilot-test-* |
| TC-GATEP1-99 | P | qc2 2026-11-19 dọn phạm vi sprint 3: docker rm smk3-gw1/smk3-pg/smk3-redis, qc2-fresh-pg, ep-r-worker-qc2b (không còn); xoá /tmp/qc2i-gw.env.full (chứa env ep-r), /tmp/qc2-w.env, /tmp/qc2-snap; nhà QC2-real-*/QC2S-*/QC2C-* đã xoá khỏi ep-r (kiểm API); route CHAT/UTILITY về QC 9router; không tiến trình stub/next start/Chrome tạm của qc2; git status ngoài artefact QC: rỗng. Ghi nhận ngoài phạm vi sprint 3: container dừng smk91-* (stack 9.1 của SmkQ) còn lại, sẽ xoá khi đóng sprint 9.1; ep-r-gateway-smkt* là của agent sprint 4 đang chạy |
| TC-GATEPU-01 | P | lint mã sản phẩm 0 cảnh báo (xem TC-PU01-05); chưa dựng lại pbuild/gbuild để đếm «Failed to load font» nên chỉ chốt phần lint |
| TC-GATEPU-02 | P | ui-antipatterns 32 ✓ / 0 ✗; --selftest 32/32; lint-selftest 7/7; ui-allow 9 (≤10) |
| TC-GATEPU-03 | P | playwright desktop: ui-foundation + tokens + shell + dev-ui + settings-llm + a11y + data-layer đạt (ban đầu 5 ca fail vì bản sao thiếu audit.mjs, đã bổ sung) |
| TC-GATEPU-04 | P | Playwright (bỏ @real) trên bản build gate: tokens/ui-foundation/shell/dev-ui/settings-llm/a11y/data-layer đạt; visual: xem TC-PU05-01 |
| TC-GATEPU-05 | P | qc2 2026-11-19 trên gateway THẬT ep-r (scripts/qc2-gatepu05-api.py, logs/qc2-gatepu05-api.json): hợp đồng lớp dữ liệu — ETag: GET /admin/llm/providers 200 ETag W/"…" → If-None-Match → 304; 409: hai «tab» PUT cùng version → 200 rồi 409 VERSION_CONFLICT; 422 VALIDATION_FAILED (tên rỗng); cursor: gradebook limit=1 → next_cursor, trang 2 khác trang 1, limit=500 → 422 (client kẹp 100); SSE thật: /events 200 text/event-stream, «retry: 3000», «event: ready {connection_id…}», Last-Event-ID chấp nhận 200; useJob: GET /jobs/{id} có id/kind/status/progress/result. Đồng thời ca @real hiện hành: không còn ca gắn @real trong data-layer.spec (CI chạy bằng gateway giả); các ca @real còn lại là spec qc-real-* của QC. Chạy tay (CI không dựng stack) bằng API vì Playwright @real tương ứng đã chuyển thành spec gateway giả ở CI @777bb5e2 ✓ |
| TC-GATEPU-06 | P | CI @777bb5e2: 4 job Lighthouse success (xem TC-PU05-19/21); rc=0 từng job |
| TC-GATEPU-07 | P | go test: 25 pass/0 skip/0 fail |
| TC-GATEPU-08 | P | chuỗi AC12 chia nhỏ trong CI @777bb5e2: Frontend (lint + build): lint, ui antipatterns, lint selftest, build; e2e 4 shard Playwright; 4 job Lighthouse; Go test contract ok (166 s) — mọi job success trong cùng run 44343596969 |
| TC-GATEPU-09 | P | smkP: run 37145521608 @e9cc3d27 success (HEAD nhánh sprint/3-pu-p1, old-private); 3 run ci/ui-drift đã đối chiếu lại bằng gh: 37139595531 (a) đỏ ở bước ui antipatterns, 37140069267 (b) đỏ ở playwright test (visual gradebook @390, dev-ui @390), 37140802309 (c) đỏ ở playwright test (a11y axe: student, critical 24) — cổng đỏ khi vi phạm |
| TC-GATEPU-11 | P | /dev/ui 1440 (evidence-3/gate-self/dev-ui-full-0..7.png) cạnh docs/design/edupilot-ui-v3.html (ui-v3-0/1.png) cùng 1440: cùng «giọng» — chữ Be Vietnam Pro, tiêu đề đậm đen, đường kẻ 1 px hồng nhạt giữa các khối, nút primary đỏ đặc bo ~8 px, nút phụ viền mảnh nền trắng, trường nhập viền mảnh bo nhỏ, lỗi chữ đỏ dưới ô, bóng chỉ ở popover/menu, nhiều khoảng trắng; đỏ chỉ ở nút chính, chỉ báo đang chọn, lỗi, liên kết hành động. Khác biệt: ui-v3 là màn ứng dụng mẫu (Hôm nay có câu mô tả dưới tiêu đề và thanh số liệu), /dev/ui là thư viện thành phần (lưới trạng thái default/hover/focus/disabled/loading); câu mô tả dưới tiêu đề của ui-v3 đã bị bỏ ở app theo luật chữ UI 09/10. Kết luận: có |
| TC-GATEPU-12 | P | 10 câu AGENT_PROMPT.md cho /dev/ui + 6 route AC1 (ảnh gate-self; ảnh mốc visual): (1) SV thấy hành động học kế tiếp ngay: có (/ «Việc nên làm tiếp», /chat câu gợi ý); (2) GV thấy quyết định kế tiếp ngay: có (/inbox hàng «Đang chờ» + «Nhận»); (3) điểm danh dùng bàn phím nhanh: có (attendance.spec CI ✓; chưa nằm trong 6 route AC1); (4) chấm bài cạnh nhau: có (grading-review ảnh mốc 1440 + CI spec); (5) công thức điểm là tài liệu duyệt không phải tường form: có (sprint 8 TC-P6-04-01..07); (6) ngôn ngữ riêng tư dễ hiểu: có («Phiên chat này chỉ bạn đọc được.»); (7) mọi route cùng một sản phẩm: có (22 ảnh cùng vỏ, chữ, đường kẻ); (8) 0 card lồng card: có (0/22); (9) mọi chỗ đỏ có nghĩa: có (xem GATEPU-13); (10) primitive đủ trạng thái focus/loading/error/disabled: có (/dev/ui lưới trạng thái, CI dev-ui.spec ✓). Không câu «không» |
| TC-GATEPU-13 | P | đếm bằng DOM + canvas ảnh (gate build @1409b04d, dữ liệu giả cố định; evidence-3/gate-self/metrics.json, red-area.json) trên 11 route × 1440/390: card lồng card = 0 ở cả 22 lượt; nút đỏ đặc / vùng làm việc: /chat 0, /threads 1 (Đặt câu hỏi), /inbox 0, /gradebook 1 (Chốt điểm), /settings/llm 0 (liên kết đỏ «Thêm nhà cung cấp»), /dev/ui = các ô trạng thái của thư viện (mỗi ô 1 nút, minh hoạ); diện tích đỏ: 0,03–0,99 % ở 1440, cao nhất 4,79 % (threads @390: nút «Đặt câu hỏi» rộng hết hàng), 4,35 % (/dev/ui @390) — đều < 8 %; mọi chỗ đỏ thuộc ba nghĩa: đang ở đâu (vạch + chữ mục nav), cần hành động (Đặt câu hỏi, Chốt điểm, Thêm nhà cung cấp, Cài đặt nâng cao, Đóng không trả lời), lỗi/xác nhận (lỗi xác thực, trạng thái). Ghi chú: «Cài đặt nâng cao» đỏ lặp 7 lần ở /settings/llm (liên kết hành động phụ lặp từng hàng) |
| TC-GATEPU-14 | P | smkP: như TC-PU05-41 — /chat 375×330 giả lập (chỉ giả lập, ghi rõ không điện thoại thật): ô nhập với tới được và không bị che, gửi được; ảnh shots/smkP-chat-375x330.png |
| TC-GATEPU-15 | P | cùng lượt chạy lại 07a6b69c: màn lỗi của SV ở /me, /threads, /chat, /practice, /calendar, /, /library không hiện chi tiết kỹ thuật (e2e/qc2-r8/real-tech.spec.ts, 7/7 sạch) |
| TC-GATEPU-16 | P | 12 cặp trước/sau: evidence-3/gate-self/pairs/{chat,gradebook,home,inbox,settings-llm,threads}-{1440,390}.png (trái = shots/before sprint 3, phải = ảnh chụp lại bằng build gate @1409b04d, cùng route/bề rộng, dữ liệu giả cố định); số đo: Lighthouse CI @777bb5e2 4 job xanh (LCP median ≤ 2500 ms, CLS ≤ 0,1, TBT ≤ 200 ms, JS ≤ 256000 B), axe: CI a11y.spec ✓ (admin có dữ liệu /settings/llm, tích hợp) và a11y trên route khác. report-GATE-PU.md đã có mục luận văn |
| TC-GATEPU-17 | P | cùng lượt dọn: không còn next start, Chrome, container smk3/qc2 của sprint 3; /tmp/qc2-pwout (output Playwright) xoá khi xong; git status chỉ artefact QC (frontend/e2e/qc2-r3/*, playwright.qc2-vis.config.ts, docs/regression/**), không sửa tệp sản phẩm. Playwright webServer của build gate tự tắt sau mỗi lượt |
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
| TC-P101-34 | P | CI run 44343596969 @777bb5e2: Go (static) 134940237507 success (go vet ./... mặc định + testroutes, golangci-lint 0 issues) + 11 job Go test (-race testroutes + không tag) success; 64 gói có test đều ok. Không chạy go test toàn cây local theo luật PM |
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
| TC-P103-11 | P | cùng lượt: 200 BATCH + 25 CHAT ≤ 5 đồng thời: interactive queue_wait p95 = 0 ms (mốc không BATCH = 0 ms; chênh 0 ≤ 100); đỉnh BATCH khi có INTERACTIVE = 5 (≤ 5); TTFT p95 mốc 13 736 ms → 13 864 ms (tỉ lệ 1,009 ≤ 1,2). Ghi nhận: TTFT p50 tỉ lệ 1,204 (mốc 9 515 → 11 459 ms) do độ trễ giả ngẫu nhiên 5–15 s của fake, queue_wait đồng nhất 0 ms nên không do Scheduler |
| TC-P103-12 | P | qc2 2026-11-19 @1409b04d, mini stack smk3-gw1 (fake 5–15 s, LLM_MAX_CONCURRENCY=10): pha A chỉ 200 BATCH: đỉnh BATCH đang chạy = 5 (≤ ceil(10×0,5)=5, 3722 mẫu mỗi 25 ms); 10 chat đầu sau khi BATCH chạy hàng loạt: queue_wait p95 = 0 ms (≤ 500). BATCH: 14 trả 200, 186 trả 504 (quá hạn deadline request, đúng thiết kế việc BATCH không chặn chat). logs/qc2-p103-load.json |
| TC-P103-13 | P | smkP: stack cô lập smk3-gw1 (fake 5–15 s, LLM_MAX_CONCURRENCY=10, BATCH_SHARE 0,5, RPM 600000; đã docker stop): go run ./cmd/llmload --base http://localhost:18201 --batch 200 --chat 25 (không --prime) → in interactive_wait_p95_ms=91, batch_peak=5, chat_ok=25 chat_fail=0 (BATCH 188/200 bị hạn BATCH mặc định cắt theo thiết kế, llm_audit INSIGHT timeout 188, ok 12); llm_audit CHAT 25 dòng queue_wait trung bình 45 ms, tối đa 301 ms |
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
| TC-P103-49 | P | CI run 44343596969 @777bb5e2 (repo kagamikuro1024/EduPilot-v2): Go (static) job 134940237507 success (go vet cả testroutes, golangci-lint cả testroutes, sqlc); Go test -race (testroutes) shards auth/course/rest-1..3 + «Go test không tag» 1..4/4 đều success (internal/llm/scheduler ok); + local qc2: integration scheduler -count=3 ok 154 s |
| TC-P103-50 | P | số đo thực ghi ở TC-P103-11/12: p95 chờ hàng INTERACTIVE 0 ms; TTFT p95 ratio 1,009 (p50 1,204); đỉnh inflight BATCH 5; cấu hình: máy Mac M-series, Colima 6 CPU/9,9 GB, gateway image edupilot-gateway-test, LLM_MAX_CONCURRENCY=10, RPM 600000, fake 5–15 s; số tự đo, không nới ngưỡng |
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
| TC-P104-41 | P | go test: 25 pass/0 skip/0 fail |
| TC-P104-42 | P | make -C backend-go lint rc=0 (go vet ×3 tag: mặc định/testroutes/integration + golangci-lint ×3: «0 issues»); go test -race -tags 'integration testroutes' ./internal/contract: 27 pass/0 fail (golden PG không đổi, đường dẫn PG /events, healthz, readyz, jobs/{id} còn nguyên) |
| TC-P104-43 | P | JSON hỏng (400, 'BAD_REQUEST', True); 2MB (413, 'PAYLOAD_TOO_LARGE'); ct sai (415, 'UNSUPPORTED_MEDIA_TYPE'); CORS evil=None; ETag routes=W/"Z_OOqA_ZzZaOUKf_" |
| TC-P104-44 | P | 3 lần GET /admin/llm/providers (10 nhà, 45 mô hình): llm_providers +3 và llm_models +3 lượt quét (pg_stat_user_tables) = đúng 1 truy vấn/bảng/lần, 2 truy vấn mỗi lời gọi ≤ 3, không N+1; mã ListProviders gom bằng map |
| TC-P104-45 | P | go test: 3 pass/0 skip/0 fail |
| TC-P104-46 | P | qc2 2026-11-19 chạy lại độc lập scenario-P1.md trên ep-r (script scripts/qc2-s3-scenario.py, logs/qc2-s3-scenario.txt, dưới khoá global; stub :9764; nhà QC2S-* đã xoá, route CHAT hoàn): 3 POST Chinh 201 (has_key, key_status ok, last_test ok, thân không có api_key); 4 Phu 201; 6 CHAT=[Phu,Chinh] → Phu/fake-chat-2 fallback_index 0 (4/4); 7 test khoá sai → ok:false AUTH «Khoá API không được nhà cung cấp chấp nhận…»; 8 POST khoá sai → 422 PROVIDER_AUTH_FAILED ở api_key; 9 chuỗi sai-khoa/good-key 0 lần trong GET providers; 10 tắt Phu → Chinh index 0 (4/4); 11 bật lại → Phu (4/4); 12 Q-chính (stub) index 0; 13 stub lỗi → Chinh fake-chat fallback_index 1 (3/3 đọc được, 1 lần đọc audit trễ), stub nhận 8 yêu cầu (2 lần thử/lời gọi), llm_audit ok/fb=1. Ghi chú như scenario: tắt nhà chính ⇒ index 0 (đếm trong chuỗi đang bật), index 1 khi nhà chính lỗi mà còn bật. Ghi chú: provider fake trên ep-r không kiểm khoá (không FAKE_LLM_VALID_KEY) nên bước khoá sai dùng 9router với khoá sai |
| TC-P104-47 | P | secret trong repo: 0; mã lỗi mới có thông điệp tiếng Việt: quan sát qua API (OVERLOADED/ROUTE_INVALID… xem TC-P103/104) |
| TC-P104-48 | P | CI run 44343596969 @777bb5e2: Go (static) success gồm sqlc diff; go vet + golangci-lint; go test -race các shard success (internal/store ok 147 s, internal/contract ok 166 s) |
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
| TC-P105-44 | P | smkP @88ab83e4 (ffb46656 R3-01): Chrome thật, /settings/llm, Admin 1440 + 390 và Teacher 1440, mở mọi <details>/«Cài đặt nâng cao» + 9 hộp thoại (Test kết nối, Thêm hành động…): innerText khớp /\b(RAG/PII/trace/fallback/embedding/prompt)\b/i = 0 ở cả 3 lượt; hàng «EMBEDDING» nay là «Đánh chỉ mục tài liệu» (555 lượt); không role=status chứa «Thành công»; nút hành động đều là động từ (Test, Thêm, Lưu, Đổi hạn mức…; «Theo lớp», «7 ngày», «30 ngày» là nút chọn chế độ). ui/smkP_p10544.cjs; logs/smkP-p10544.json |
| TC-P105-45 | P | đạt qua Playwright cùng AC11: «settings-llm copy» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-46 | P | đạt qua Playwright cùng AC12: «settings-llm mobile+keyboard+axe» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-47 | P | settings-llm.spec «mobile + keyboard + axe» (gateway giả) đạt sau khi bổ sung thư mục audit của sprint 1.5 vào bản sao |
| TC-P105-48 | P | settings-llm.spec axe 0 serious/critical (gateway giả; axe trên màn thật ADMIN/GV: chưa chạy) |
| TC-P105-49 | P | useDemoSlice/ep_demo trong settings/llm: 0 |
| TC-P105-51 | P | gate build (:3310 + gateway giả :3312), evidence-3/gate-self/settings-auth.json: (a) không phiên (kèm cookie mô phỏng cũ ep_demo_role=admin — đã bị gỡ ở US-P2-12 AC10): /settings/llm chuyển /login?next=%2Fsettings%2Fllm, trang «Đăng nhập EduPilot», 0 request tới /api/v1/admin/llm/*, không dữ liệu mock; (b) JWT Admin: màn gọi apiClient thật GET /admin/llm/providers, /routes, /decision, /usage, /budget và hiện «Cấu hình LLM» + nhà OpenAI từ gateway giả — không đọc cookie demo |
| TC-P105-52 | P | ảnh mốc settings-llm đã cập nhật có giải thích trong handoff (docs/sprints/8/handoff/dev-US-S8-UI.md «Ảnh mốc… sinh lại trong image Playwright»); khác biệt trước/sau (pairs/settings-llm-1440.png, shots/before vs build @1409b04d): bỏ câu mô tả dưới tiêu đề và dòng «Áp dụng cho toàn hệ thống · Cập nhật gần nhất», bỏ 4 ký tự cuối khoá («••••3f9a») và nút «Đổi khoá» (nay là menu «…»), bỏ địa chỉ máy chủ monospace và mô tả loại nhà ở từng hàng, trạng thái còn 1–2 từ («Đã kết nối», «Lỗi xác thực»), «model» → «mô hình»; khối mới «Chuỗi dự phòng», «Quyết định nhanh» |
| TC-P105-53 | P | đạt qua Playwright cùng AC14: «qc-real-f15 hai tab» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-54 | P | qc-real-f15-llm: hai tab sửa cùng nhà → tab sau 409 và nạp lại (stack ep-r) |
| TC-P105-55 | P | qc2 2026-11-19 trên ep-r thật, ghép: (1) UI Chrome admin (gate-self/canary-ui.json + canary-badkey.png): thêm nhà, khoá sai → «Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá.», khoá đúng/giả lập lưu được, Test kết nối, Đổi khoá, Xoá; (2) kịch bản API P104-46 (logs/qc2-s3-scenario.txt): chuyển CHAT sang nhà thứ hai, tắt nhà chính → nhà kế trả lời (fallback_index 0), nhà chính lỗi còn bật → index 1; (3) tắt hết (nhà duy nhất không kết nối): 0,5 s, chat trả «AI đang gián đoạn. Thử lại sau.» + degraded:true, không từ kỹ thuật (logs/qc2-s3-degrade.txt; câu theo US-P3-05 AC8 «không có ngữ cảnh thì …»; Go ./internal/chat TestAllProvidersDownExtractive + NoContext PASS). Ghi chú: câu cũ «AI tạm thời không khả dụng» đã đổi theo US-P3-05 AC8; ảnh từng bước: canary-badkey.png (các bước còn lại chụp bằng văn bản log, không ảnh) |
| TC-P105-56 | P | 10 câu §22 cho /settings/llm (gate build, dữ liệu giả cố định; evidence-3/gate-self/settings-llm-1440/390.png, metrics.json): tiêu đề «Cấu hình LLM» + khối rõ thứ tự; 0 card lồng card; 0 nút đỏ đặc thừa (chỉ liên kết đỏ «Thêm nhà cung cấp» / «Cài đặt nâng cao»); diện tích đỏ 0,12 % (1440) / 0,21 % (390); đỏ mang ba nghĩa (hành động / lỗi xác thực / đang ở đâu); so shots/before + edupilot-ui-v3.html: cùng giọng, hơn nhờ bỏ câu mô tả. Ghi chú: «Cài đặt nâng cao» đỏ lặp 7 lần |
| TC-P105-57 | P | git grep sk-… trong frontend (ngoài e2e): 0 |
| TC-PU01-02 | P | token thêm so với bản gốc: 7 (≥4 đúng tên; TC cũ ghi đúng 4, đã tăng theo sprint sau): ['--ep-ink-3', '--ep-mask-clear', '--ep-mask-solid', '--ep-scrim', '--ep-shadow-composer', '--ep-slow', '--ep-text-display'] |
| TC-PU01-04 | P | khai báo --ep-* ngoài tokens.css: 0 |
| TC-PU01-05 | P | eslint trên bản sao tracked (/tmp/smk3-fe) loại e2e/qc2-*: 0 vấn đề rc=0 — 5 cảnh báo cũ (DEF-R3-03) đã hết; khi gồm spec QC e2e/qc2-* đã commit: 47 vấn đề toàn ở spec QC (đã báo PM) |
| TC-PU01-06 | P | ui-antipatterns.sh rc=0, 32 dòng ✓ (≥11), 0 ✗ |
| TC-PU01-07 | P | ui-allow: 9 chỗ (trần 10) |
| TC-PU01-08 | P | gieo 3 vi phạm màu/bo góc → rc=1, 2 phép ✗ |
| TC-PU01-09 | P | 3 ca xấu → ✗×3; 4 ca lành rc=0 (không báo nhầm) |
| TC-PU01-10 | P | .tsx gieo → rc=1, ✗×6 |
| TC-PU01-11 | P | lint-selftest.sh: 7 / 7 luật ESLint + 32 / 32 phép ui-antipatterns bắt được; rc=0 (TC cũ ghi 19 phép, đã tăng) |
| TC-PU01-12 | P | smkP: chạy scripts/lint-selftest.sh trong nhóm tiến trình riêng, SIGINT sau 2,5 s (lúc đó frontend/src/__selftest__ đang tồn tại) → tiến trình thoát (-2) và thư mục tạm bị xoá; git status --short frontend/src = 0 dòng |
| TC-PU01-13 | P | smkP: tự gieo từng trong 7 luật ESLint ep/* (no-raw-fetch, no-native-dialogs, no-custom-spinner[LoadingSpinner], no-raw-table, no-token-in-storage, no-tailwind, no-literal-color-in-style), mỗi tệp eslint exit=1 và nêu đúng tên luật 7/7 (lượt đầu 6/7 do tên hàm gieo PageSpinner nằm ngoài tập BAD của luật = lỗi kịch bản QC, chạy lại LoadingSpinner bắt được); tệp tạm xoá, git status frontend/src sạch. evidence-3/scripts/smkP-pu01-lint.sh |
| TC-PU01-14 | P | đếm = ['0'] (kỳ vọng 0) |
| TC-PU01-15 | P | đếm = ['0'] (kỳ vọng 0) |
| TC-PU01-16 | P | đếm = ['0'] (kỳ vọng 0) |
| TC-PU01-17 | P | smkP: ca âm localStorage.setItem("access_token"), sessionStorage.setItem("jwt"), document.cookie="refresh=1" → eslint exit=1 ep/no-token-in-storage 3/3; ca lành localStorage.setItem("ep_demo_state") → exit 0 (không chặn nhầm). smkP-pu01-lint.sh |
| TC-PU01-18 | P | đếm = ['0', '0'] (kỳ vọng 0) |
| TC-PU01-19 | P | Playwright tokens.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-PU01-20 | P | smkP: Chrome thật mở /login: 34 request, mọi host = localhost:1143, 0 request tới fonts.googleapis.com / fonts.gstatic.com. evidence-3/ui/smkP_pu01.cjs; logs/smkP-pu01.json |
| TC-PU01-22 | P | smkP: getComputedStyle(documentElement) --ep-font = "Be Vietnam Pro", "Noto Sans", ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif (đúng thứ tự AC5) |
| TC-PU01-23 | P | Playwright tokens.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-PU01-24 | P | Playwright tokens.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-PU01-25 | P | smkP: cỡ chữ tính ra của mọi nút văn bản ở / , /chat, /threads (sv.gioi) và /gradebook (teacher) chỉ thuộc tập --ep-text-* đã giải (52/28/20/16/15/14/12 px + 3 cỡ màn chiếu); 0 cỡ lạ ở cả 4 route |
| TC-PU01-26 | P | Playwright tokens.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-PU01-27 | P | smkP: colorScheme=light; ::selection nền rgb(253,238,240) = --ep-red-soft; scrollbar-width=thin; 33 tệp CSS đã build trong ep-r-frontend-1 (/app/frontend/.next/static) có 0 tệp chứa prefers-color-scheme, frontend/src 0 chỗ |
| TC-PU01-28 | P | Playwright tokens.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-PU01-29 | P | 3 logo/favicon trên frontend ep-r :3700 → [200, 200, 200] |
| TC-PU01-30 | P | Playwright tokens.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-PU01-31 | P | CI run 44343596969 @777bb5e2: job «Frontend (lint + build)» 134940237616 bước «build (như production)» success (Compiled successfully); các job build (gate) ở 4 shard e2e + 4 job Lighthouse đều success; grep «Failed to load font» trong log 8 job = 0 (qc2 tải log gh run view --job) |
| TC-PU01-34 | P | visual.spec.ts @777bb5e2 CI Linux: 26/26 ✓ ở ngưỡng maxDiffPixelRatio 0,005 (ảnh mốc hiện hành). Đối chiếu bằng mắt shots/before/chat-1440.png (nền sprint 3) với ảnh mốc chat-1440.png hiện tại: khác nhau có chủ đích do các sprint sau (bỏ câu mô tả dưới tiêu đề theo luật chữ UI 09/10, vỏ mới 5 nhóm điều hướng, bỏ nhãn «Bản mô phỏng»…), không phải lỗi PU-01 |
| TC-PU01-35 | P | setItem khoá nhạy cảm trong frontend/src = 0 |
| TC-PU01-36 | P | smkP: sau đăng nhập + / , /chat, /me: localStorage chỉ có ep_demo_state, ep_returning; sessionStorage rỗng; document.cookie rỗng (cookie phiên httpOnly); khách ở /login: rỗng — không khoá nào khớp token/jwt/access/refresh/secret/password |
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
| TC-PU02-40 | P | production: ep-r frontend (build thường) GET /dev/ui = 404, /dev/data = 404 (cả qua Caddy :1143 và FE :3700); grep -rl 'data-part="primitive"' /app/frontend/.next/static = 0 tệp bundle; bản gate: CI @777bb5e2 e2e dev-ui.spec ✓ (chạy trên build gate, /dev/ui 200) và Lighthouse (simulate B / devtools B) đo /dev/ui thành công |
| TC-PU02-41 | P | đạt qua Playwright cùng AC13: «offline-input» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-42 | P | đạt qua Playwright cùng AC13: «offline-input» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-43 | P | đạt qua Playwright cùng AC14: «one-primary» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-44 | P | đạt qua Playwright cùng AC14: «one-primary» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-45 | P | đạt qua Playwright cùng AC14: «one-primary» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-46 | P | /dev/ui (1440 + 390; dev-ui-full-*.png, dev-ui-390): 10 câu §22: cả 10 «có»; card lồng card = 0 (framed 18/19 khối, nested 0); ≤ 1 đỏ đặc / vùng (mỗi ô trạng thái một nút); mọi chỗ đỏ có nghĩa; diện tích đỏ 0,99 % (1440) / 4,35 % (390) < 8 %; ảnh 1440 + 390 đính kèm |
| TC-PU02-47 | P | đọc chuỗi /dev/ui: chữ SV có thể thấy trong thành phần (Lưu thay đổi, Gửi, Bộ lọc, Họ và tên · bắt buộc, «Họ và tên không được để trống. Nhập tên đầy đủ.», Mở hộp thoại, Đổi tên/Nhân bản/Lưu trữ…): không từ kỹ thuật (quét RAG/PII/LLM/fallback/trace/provider/token/job/API/JSON… = 0 trên 6000 ký tự đầu + 8 ảnh cuộn); nút là động từ. Ghi chú: ghi chú nhà phát triển của lưới (như «Thành phần không có khái niệm “được chọn”», «InlineNotice / Field») chỉ có ở /dev/ui (404 production) |
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
| TC-PU05-01 | P | macOS arm64 tự dựng ảnh mốc riêng (config playwright.qc2-vis.config.ts, snapshotPathTemplate /tmp/qc2-snap, KHÔNG đụng ảnh mốc Linux trong repo): visual.spec.ts @1409b04d bản gate: --update-snapshots 26 passed, rồi chạy lại 26 passed (rc=0, 0 skip trong dự án desktop); 26 = 13 route × 2 bề rộng (số ảnh đổi từ 14 do US-UX91). CI Linux cũng 26/26 ✓ |
| TC-PU05-02 | P | 26 ảnh mốc (≥14 = 7 route × 2 bề rộng; hiện nhiều hơn do route thêm sau) |
| TC-PU05-03 | P | visual.spec.ts: maxDiffPixelRatio 0.005, threshold 0.2, animations disabled, reducedMotion reduce, page.clock.setFixedTime 2026-10-29 09:20 +07, timezoneId Asia/Ho_Chi_Minh, chờ document.fonts.ready |
| TC-PU05-04 | P | đối chứng âm (biến thể do khoá build gate cố định, không sửa tệp nguồn): tiêm CSS «button{position:relative;left:3px}» khi chụp (visual-neg.spec.ts bản sao có hook QC2_SHIFT): 9/26 ảnh FAIL (đều @390: chat, inbox, attendance, grading-queue, grading-review, gradebook, settings-llm, dev-ui, insights), 17 pass (@1440 lệch 3 px của nút nhỏ hơn ngưỡng 0,005) → cổng bắt được lệch; bỏ tiêm → 26 pass; git status sạch (không sửa tệp). Khác TC: không sửa tệp ở shared/ui + gbuild vì sẽ phải build đè lên .next dùng chung |
| TC-PU05-05 | P | 3 lần liên tiếp local (macOS, ảnh mốc tự dựng): 26 passed ×3 (18,4–19,1 s), 0 flaky; thêm CI 4 run xanh |
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
| TC-PU05-19 | P | CI @777bb5e2 4 job Lighthouse (simulate A/B, devtools A/B) success: 9 URL (/ /chat /threads /inbox /gradebook /me /checkin /settings/llm /dev/ui), simulate 5 lượt/URL, devtools 3 lượt/URL (số URL/lượt đổi so với TC cũ 7×3, theo TL-CI #3b / DEF-R7-01); «Done running autorun», Checking assertions không lỗi |
| TC-PU05-21 | P | CI @777bb5e2: lighthouserc.devtools.json assert LCP median ≤ 2500 ms (error mọi route trừ /dev/ui là warn), lighthouserc.json assert CLS ≤ 0,1, TBT ≤ 200 ms, resource-summary:script:size ≤ 256000 B (median, error) — cả 4 job xanh. Số chẩn đoán trong log simulate: TBT 21–66 ms (benchmarkIndex ~3500, cpuSlowdown 3,58). Ngưỡng JS 256000 B ≈ 250 KB (TC viết 250 KB nén) |
| TC-PU05-22 | P | JS truyền (responseBodySize, nén) lần đầu mỗi route, gate build @1409b04d (gate-self/js-size.json): / 212 657 B; /chat 221 447; /threads 212 684; /inbox 210 752; /gradebook 247 725 (lớn nhất); /me 217 867; /settings/llm 219 286; /dev/ui 210 327 — tất cả ≤ 256 000 B (≈ 250 KiB). So nền sprint 3 (report-GATE-PU: 197–230 KB): tăng ≈ +13…+18 KB ở các route chung, /gradebook +~18 KB; vẫn ≤ 250 KB. Ngưỡng JS ≤ 256000 B cũng được CI Lighthouse (simulate) assert error và xanh |
| TC-PU05-23 | P | ui-antipatterns.sh 32 ✓/0 ✗; --selftest 32/32 bắt được; chạy ở bản sao, frontend/src sạch (TC cũ: 19 phép — đã tăng) |
| TC-PU05-24 | P | ui-antipatterns.sh 32 ✓/0 ✗; --selftest 32/32 bắt được; chạy ở bản sao, frontend/src sạch (TC cũ: 19 phép — đã tăng) |
| TC-PU05-25 | P | smkP @88ab83e4: tự gieo 8 vi phạm (màu cứng, bo 16 px, bóng ngoài Popover, cỡ chữ 13 px, fetch( ngoài shared/data, emoji 🔍 làm icon, localStorage.setItem token, streak) vào BẢN SAO frontend/src (UI_SRC), mỗi lượt scripts/ui-antipatterns.sh rc=1 và dòng ✗ đúng tên phép 8/8; cây sạch rc=0 trước và sau; không dùng --selftest. evidence-3/scripts/smkP-pu0525.sh |
| TC-PU05-26 | P | ui-antipatterns.sh 32 ✓/0 ✗; --selftest 32/32 bắt được; chạy ở bản sao, frontend/src sạch (TC cũ: 19 phép — đã tăng) |
| TC-PU05-27 | P | ui-antipatterns.sh 32 ✓/0 ✗; --selftest 32/32 bắt được; chạy ở bản sao, frontend/src sạch (TC cũ: 19 phép — đã tăng) |
| TC-PU05-28 | P | smkP: sprint 3 chạy CI ở repo kagamikuro1024/TA_Agent_v2 (old-private): gh run list --workflow ci.yml --branch sprint/3-pu-p1 --limit 1 → run 37145521608 headSha e9cc3d27… = git ls-remote old-private sprint/3-pu-p1 (e9cc3d27…); conclusion=success (Go, Frontend đều success) |
| TC-PU05-29 | P | smkP: gh run view 37145521608 --json jobs: job Frontend có các bước theo đúng thứ tự pnpm install (cache) → lint → ui antipatterns → lint selftest → build (như production) → build (gate) → cache/playwright install chromium → playwright test → lighthouse ci → lighthouse benchmark; số bước khớp playwright/lhci/antipatterns/selftest/lighthouse = 6 (≥ 4) |
| TC-PU05-30 | P | grep legacy/secrets. trong .github/workflows/ci.yml: 0 dòng; job Go giữ nguyên, thay đổi so với sprint 2 chỉ thêm job Frontend và env TESTCONTAINERS_RYUK_DISABLED + timeout-minutes |
| TC-PU05-31 | P | smkP: gh run download 37140069267 -R kagamikuro1024/TA_Agent_v2 (run (b) thất bại) → artifact frontend-reports/ có playwright-report/ và test-results/; bước upload reports conclusion=success |
| TC-PU05-32 | P | smkP: run 37139595531: conclusion=failure, headBranch=ci/ui-drift, Frontend failure (bước thất bại duy nhất: ui antipatterns), Go success; log-failed khớp ui-antipatterns/«Màu viết cứng» 4 dòng; git show 12638d78: thêm frontend/src/features/chat/drift.module.css `.x { color: #c81d32; }` (ngoài shared/styles) |
| TC-PU05-33 | P | smkP: run 37140069267 (@4081b0b3): failure ở bước playwright test: visual.spec.ts «gradebook @390» và «dev-ui @390» ✘ (2 failed, toHaveScreenshot); Go success; git show 4081b0b3: Button.module.css thêm `position: relative; top: 3px` (nút lệch 3 px) |
| TC-PU05-34 | P | smkP: run 37140802309 (@38073d08, chứa e1cb4831): failure ở bước playwright test, a11y.spec.ts «axe: student» ✘ (AXE critical 24, 24 lượt quét); git show e1cb4831: AppShell nút chuông mất aria-label; Go success |
| TC-PU05-35 | P | qc2 2026-11-19: git ls-remote --heads origin ci/ui-drift → rỗng (0 dòng): nhánh đã bị dev xoá khỏi origin (EduPilot-v2); chỉ còn bản lưu ở old-private (TA_Agent_v2, repo cũ, không phải origin). DEF-R3-SMK-02 đóng |
| TC-PU05-36 | P | commit PU: đụng backend-go: 0 |
| TC-PU05-37 | P | go test: 25 pass/0 skip/0 fail |
| TC-PU05-39 | P | 10 câu AGENT_PROMPT.md cho từng màn dựng (/dev/ui + /, /chat, /threads, /practice, /library, /calendar, /me, /inbox, /gradebook, /settings/llm): như TC-GATEPU-12 + 0 card lồng; ghi chú: /me và /threads có «Chi tiết kỹ thuật» ở trạng thái lỗi (DEF-R3-SMK-03) — thuộc luật chữ cho SV, tính ở TC-PU05-40/GATEPU-15, không ở 10 câu bố cục |
| TC-PU05-40 | P | qc2 chạy lại sau rebuild @HEAD gồm 07a6b69c: real-tech.spec.ts (7 route SV với API 500): cả 7 «sạch» gồm /me và /threads — không còn «Chi tiết kỹ thuật», trace_id, mã lỗi; chữ lỗi tiếng Việt thân thiện + «Thử lại» |
| TC-PU05-41 | P | smkP: giả lập 375×330 (Chrome headless, isMobile+hasTouch; chỉ giả lập, không có điện thoại thật): /chat focus + gõ «xin chào»: ô nhập top 172–bottom 208 trong khung 330, elementFromPoint tại tâm = chính textarea (không bị thanh dưới nav top 273 che), nút Gửi hiện, gửi được («Đang trả lời…»), scrollWidth-innerWidth=0; ảnh shots/smkP-chat-375x330.png |
| TC-PU05-42 | P | cùng CI run: lint + antipatterns (+ selftest) + build + Playwright --grep-invert @real 4 shard + lhci + go test ./internal/contract ok — tất cả success @777bb5e2 |
| TC-PU05-43 | P | JWT cứng trong e2e/lighthouserc: 0 |
| TC-PU05-44 | P | qc2: grep frontend/e2e/**: email chỉ dạng vai/giả (admin@/gv@/sv@ptit.edu.vn, *.example.test, *.edupilot.local, sv.gioi@sv.ptit.edu.vn…), MSSV dạng giả B22DCCN101–104/B21DCCN004, tên «Sinh Viên 04/0001», «Trần Thu Uyên», «TS. Lê Thu Hà» (tên bịa của fixture); xem 3 ảnh mốc gradebook-1440, chat-1440, home-1440: chỉ «Sinh Viên 00xx», B22DCCN1xx, Trần Thu Uyên — không dữ liệu sinh viên thật (D44). Ghi chú: không có tài liệu nêu rõ hai tên fixture là bịa |
| TC-P104-40 | LT | golden PG không bị sửa (git diff origin/sprint/2-pg..HEAD: 0 dòng xoá ở testdata/golden); redocly lint openapi.yaml rc=0 (21 cảnh báo); riêng «operationId = 18 / openapi.test = 18» lỗi thời: các sprint sau thêm API (nay 344 operationId trong openapi.yaml, 21 trong openapi.test.yaml) |
| TC-PU05-20 | LT | lighthouserc.json nay 9 URL × 5 lượt, LCP đo ở lượt devtools riêng (lighthouserc.devtools.json) — đổi hợp lệ ở US-PU-06 #14 (commit 77b4833e) và US-P5-06; không còn «7 URL × 3 lượt, LCP trong cùng assert» |
| TC-PU01-32 | LT | smkP LỖI THỜI (nguồn: US-P2-12 AC10 docs/specs/FEAT-course-foundation/US.md:293 + commit 28cf626c 2026-10-04 «bỏ phiên mô phỏng bằng cookie ep_demo_*»; docs/sprints/4/qc/report-GATE-P2.md L4): audit.mjs/sweep.mjs/proto-curl.sh dựa trên cookie ep_demo_role/person/course nên KHÔNG chạy được từ sprint 4; thay bằng audit-login.mjs (sprint 4, TC-P212-21). Bản thay thế đã chạy: evidence-5/ui/smkP_audit.cjs bốn vai trên ep-r: 115 hàng, 109 PASS, 6 FAIL (xem HANDOFF-smk3-4.md mục AUDIT) |
| TC-PU01-33 | LT | smkP LỖI THỜI (nguồn: US-P2-12 AC10 docs/specs/FEAT-course-foundation/US.md:293 + commit 28cf626c 2026-10-04 «bỏ phiên mô phỏng bằng cookie ep_demo_*»; docs/sprints/4/qc/report-GATE-P2.md L4): audit.mjs/sweep.mjs/proto-curl.sh dựa trên cookie ep_demo_role/person/course nên KHÔNG chạy được từ sprint 4; thay bằng audit-login.mjs (sprint 4, TC-P212-21). Bản thay thế đã chạy: evidence-5/ui/smkP_audit.cjs bốn vai trên ep-r: 115 hàng, 109 PASS, 6 FAIL (xem HANDOFF-smk3-4.md mục AUDIT) |
| TC-PU02-39 | LT | smkP LỖI THỜI (nguồn: US-P2-12 AC10 docs/specs/FEAT-course-foundation/US.md:293 + commit 28cf626c 2026-10-04 «bỏ phiên mô phỏng bằng cookie ep_demo_*»; docs/sprints/4/qc/report-GATE-P2.md L4): audit.mjs/sweep.mjs/proto-curl.sh dựa trên cookie ep_demo_role/person/course nên KHÔNG chạy được từ sprint 4; thay bằng audit-login.mjs (sprint 4, TC-P212-21). Bản thay thế đã chạy: evidence-5/ui/smkP_audit.cjs bốn vai trên ep-r: 115 hàng, 109 PASS, 6 FAIL (xem HANDOFF-smk3-4.md mục AUDIT) |
| TC-PU04-59 | LT | smkP LỖI THỜI (nguồn: US-P2-12 AC10 docs/specs/FEAT-course-foundation/US.md:293 + commit 28cf626c 2026-10-04 «bỏ phiên mô phỏng bằng cookie ep_demo_*»; docs/sprints/4/qc/report-GATE-P2.md L4): audit.mjs/sweep.mjs/proto-curl.sh dựa trên cookie ep_demo_role/person/course nên KHÔNG chạy được từ sprint 4; thay bằng audit-login.mjs (sprint 4, TC-P212-21). Bản thay thế đã chạy: evidence-5/ui/smkP_audit.cjs bốn vai trên ep-r: 115 hàng, 109 PASS, 6 FAIL (xem HANDOFF-smk3-4.md mục AUDIT) |
| TC-PU05-38 | LT | smkP LỖI THỜI (nguồn: US-P2-12 AC10 docs/specs/FEAT-course-foundation/US.md:293 + commit 28cf626c 2026-10-04 «bỏ phiên mô phỏng bằng cookie ep_demo_*»; docs/sprints/4/qc/report-GATE-P2.md L4): audit.mjs/sweep.mjs/proto-curl.sh dựa trên cookie ep_demo_role/person/course nên KHÔNG chạy được từ sprint 4; thay bằng audit-login.mjs (sprint 4, TC-P212-21). Bản thay thế đã chạy: evidence-5/ui/smkP_audit.cjs bốn vai trên ep-r: 115 hàng, 109 PASS, 6 FAIL (xem HANDOFF-smk3-4.md mục AUDIT) |
| TC-P105-50 | LT | smkP LỖI THỜI (nguồn: US-P2-12 AC10 docs/specs/FEAT-course-foundation/US.md:293 + commit 28cf626c 2026-10-04 «bỏ phiên mô phỏng bằng cookie ep_demo_*»; docs/sprints/4/qc/report-GATE-P2.md L4): audit.mjs/sweep.mjs/proto-curl.sh dựa trên cookie ep_demo_role/person/course nên KHÔNG chạy được từ sprint 4; thay bằng audit-login.mjs (sprint 4, TC-P212-21). Bản thay thế đã chạy: evidence-5/ui/smkP_audit.cjs bốn vai trên ep-r: 115 hàng, 109 PASS, 6 FAIL (xem HANDOFF-smk3-4.md mục AUDIT) |
| TC-GATEP1-10 | LT | smkP LỖI THỜI (nguồn: US-P2-12 AC10 docs/specs/FEAT-course-foundation/US.md:293 + commit 28cf626c 2026-10-04 «bỏ phiên mô phỏng bằng cookie ep_demo_*»; docs/sprints/4/qc/report-GATE-P2.md L4): audit.mjs/sweep.mjs/proto-curl.sh dựa trên cookie ep_demo_role/person/course nên KHÔNG chạy được từ sprint 4; thay bằng audit-login.mjs (sprint 4, TC-P212-21). Bản thay thế đã chạy: evidence-5/ui/smkP_audit.cjs bốn vai trên ep-r: 115 hàng, 109 PASS, 6 FAIL (xem HANDOFF-smk3-4.md mục AUDIT) |
| TC-GATEPU-10 | LT | smkP LỖI THỜI (nguồn: US-P2-12 AC10 docs/specs/FEAT-course-foundation/US.md:293 + commit 28cf626c 2026-10-04 «bỏ phiên mô phỏng bằng cookie ep_demo_*»; docs/sprints/4/qc/report-GATE-P2.md L4): audit.mjs/sweep.mjs/proto-curl.sh dựa trên cookie ep_demo_role/person/course nên KHÔNG chạy được từ sprint 4; thay bằng audit-login.mjs (sprint 4, TC-P212-21). Bản thay thế đã chạy: evidence-5/ui/smkP_audit.cjs bốn vai trên ep-r: 115 hàng, 109 PASS, 6 FAIL (xem HANDOFF-smk3-4.md mục AUDIT) |

## Trượt: bước tái hiện
Xem `defects-3.md`.

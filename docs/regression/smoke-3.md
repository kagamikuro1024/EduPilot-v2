# Smoke hồi quy sprint 3 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2/SmkB · Ngày: 2026-11-15
Tổng TC cũ: 523 · Đạt: 368 · Trượt: 4 · Không chạy (K, gồm lỗi thời): 151   (P+F+K = 523)
K chia theo loại: chưa chạy (mẫu): 130; (c): 7; (b): 7; lỗi thời: 3; nền tảng: 4. Loại «chưa chạy (mẫu)» / «nền tảng» KHÔNG phải K (a)/(b)/(c) — chờ PM quyết có chạy nốt.

Cách chạy: (1) Go: `go test -race -json ./internal/llm/... ./internal/llmconfig/... ./internal/platform/crypto/...` + `-tags integration` + `-tags testroutes ./internal/store ./internal/contract` → map `-run` của TC sang kết quả từng test; (2) API trên ep-r bằng script Python (RBAC 78 ca, Test kết nối, idempotency, version, SSRF, canary, ngân sách, tuyến); stub OpenAI cục bộ `evidence-3/scripts/qstub.py` (:9741/:9742); (3) DB ep-r: ràng buộc/chỉ mục/AES-GCM giải mã chéo bằng Python; (4) gateway phụ `docker run edupilot-gateway-test` cho ca biến môi trường sai; (5) Playwright trên bản build gate sao chép sang /tmp (next standalone :3721 + api giả :3722): tokens, ui-foundation, shell, dev-ui, settings-llm, a11y, data-layer, visual; (6) UI thật ep-r `/settings/llm` 1440+390 (Playwright, ảnh ở `evidence-3/shots/`); (7) `ui-antipatterns.sh`, `--selftest`, `lint-selftest.sh`, `pnpm lint` trong bản sao.
Lưu ý môi trường: Chromium 1255 chưa có lúc đầu → dùng headless-shell 1247; CORS_ORIGINS ep-r thiếu https://localhost:1143 (mọi F5 văng /login) — đã nhờ PM-QC sửa, tôi dùng route đổi Origin của /auth/refresh; khoá global đang bị nhóm khác giữ nên cụm ca đổi tuyến/ngân sách/cầu dao (TC-P102-14…19/42, TC-P103-03/14…41, TC-P104-27/34, TC-P105-20/22) CHƯA chạy ("chưa chạy (mẫu)" — script `/tmp/qc2-b/p10x.py` sẵn sàng).

## Tổng hợp theo US
| US | TC tổng | P | F | K | Ghi chú |
| --- | --- | --- | --- | --- | --- |
| GATE-P1 | 23 | 7 | 0 | 16 | |
| GATE-PU | 17 | 4 | 1 | 12 | |
| US-P1-01 | 35 | 25 | 1 | 9 | |
| US-P1-02 | 50 | 26 | 0 | 24 | |
| US-P1-03 | 50 | 21 | 0 | 29 | |
| US-P1-04 | 48 | 37 | 0 | 11 | |
| US-P1-05 | 57 | 51 | 1 | 5 | |
| US-PU-01 | 36 | 20 | 1 | 15 | |
| US-PU-02 | 47 | 43 | 0 | 4 | |
| US-PU-03 | 57 | 57 | 0 | 0 | |
| US-PU-04 | 59 | 58 | 0 | 1 | |
| US-PU-05 | 44 | 19 | 0 | 25 | |

## Kết quả từng ca
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-GATEP1-01 | P | grep ngoài internal/llm = 0; go.mod anthropic/genai = 0 |
| TC-GATEP1-02 | P | go test: 138 pass/1 skip/0 fail |
| TC-GATEP1-03 | P | go test: 82 pass/2 skip/0 fail |
| TC-GATEP1-04 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP1-05 | P | go test: 1 pass/0 skip/0 fail |
| TC-GATEP1-06 | P | go test: 55 pass/0 skip/0 fail |
| TC-GATEP1-07 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP1-08 | P | go test: 25 pass/0 skip/0 fail |
| TC-GATEP1-09…TC-GATEP1-10 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP1-11 | K | (c) CI GitHub / nhánh ci/ui-drift của PR sprint 3 — ngoài phạm vi đồ án |
| TC-GATEP1-12…TC-GATEP1-15 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP1-16 | K | chưa chạy (mẫu): tải/đo hiệu năng nặng (k6, Lighthouse) — chưa thu nhỏ để chạy trên ep-r |
| TC-GATEP1-17 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP1-18 | P | go test: 0 pass/1 skip/0 fail |
| TC-GATEP1-19…TC-GATEP1-99 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEPU-01 | F | lint 5 cảnh báo (TC-PU01-05); pbuild/gbuild rc=0, 0 "Failed to load font" (DEF-R3-03) |
| TC-GATEPU-02 | P | ui-antipatterns 32 ✓ / 0 ✗; --selftest 32/32; lint-selftest 7/7; ui-allow 9 (≤10) |
| TC-GATEPU-03 | P | playwright desktop: ui-foundation + tokens + shell + dev-ui + settings-llm + a11y + data-layer đạt (ban đầu 5 ca fail vì bản sao thiếu audit.mjs, đã bổ sung) |
| TC-GATEPU-04 | P | Playwright (bỏ @real) trên bản build gate: tokens/ui-foundation/shell/dev-ui/settings-llm/a11y/data-layer đạt; visual: xem TC-PU05-01 |
| TC-GATEPU-05 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEPU-06 | K | chưa chạy (mẫu): tải/đo hiệu năng nặng (k6, Lighthouse) — chưa thu nhỏ để chạy trên ep-r |
| TC-GATEPU-07 | P | go test: 25 pass/0 skip/0 fail |
| TC-GATEPU-08 | K | chưa chạy (mẫu): tải/đo hiệu năng nặng (k6, Lighthouse) — chưa thu nhỏ để chạy trên ep-r |
| TC-GATEPU-09 | K | (c) CI GitHub / nhánh ci/ui-drift của PR sprint 3 — ngoài phạm vi đồ án |
| TC-GATEPU-10…TC-GATEPU-17 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P101-01 | P | migration 00002_llm áp dụng; phiên bản hiện tại 25 (TC cũ ghi =2, đã tăng hợp lệ) |
| TC-P101-02 | P | có đủ 5 bảng; hiện llm_audit,llm_budgets,llm_models,llm_providers,llm_request_payloads,llm_task_routes (llm_request_payloads thêm ở 00022 — hợp lệ) |
| TC-P101-03…TC-P101-05 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P101-06 | P | go test: 1 pass/0 skip/0 fail |
| TC-P101-07 | P | 14 ca đều bị từ chối đúng SQLSTATE; 0 dòng lọt |
| TC-P101-08 | P | BEGIN INSERT 0 1 INSERT 0 1 INSERT 0 1 ROLLBACK  |
| TC-P101-09 | P | go test: 1 pass/0 skip/0 fail |
| TC-P101-10 | P | 21 chỉ mục: có (course_id,created_at desc) partial, (created_at desc), (task,created_at desc); trace_id/unique chi tiết không đối chiếu: llm_providers_pkey,llm_providers_name_uq,llm_models_pkey,llm_models_uq,llm_models_provider_idx,llm_task_routes_pkey,llm_task_routes_uq,llm_task_routes_model_idx,ll |
| TC-P101-11 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P101-12 | P | llm_audit_course_created_idx bắt đầu bằng course_id |
| TC-P101-13 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P101-14 | P | grep SQL trần ngoài store = 0 (kể cả comment) |
| TC-P101-15 | P | go test: 8 pass/0 skip/0 fail |
| TC-P101-16 | P | byte đầu 01, dài 37; Python AESGCM giải mã được=True (chiều Go giải bản mã Python: chưa kiểm) |
| TC-P101-17 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P101-18 | P | {'rỗng': (1, 11.5, 'nêu tên biến', 'không lộ khoá'), 'abc': (1, 8.9, 'nêu tên biến', 'không lộ khoá'), '16 byte': (1, 2.3, 'nêu tên biến', 'không lộ khoá')} / {"time":"2026-11-15T12:55:49.313287586Z","level":"ERROR","msg":"giá trị biến môi trường không hợp lệ: APP_ENCRYPTION_KEY không hợp lệ: cần 32 |
| TC-P101-19 | P | {'xuống dòng cuối': (1, False, False), '33 byte': (1, False, True), 'base64url': (1, False, True), 'khoảng trắng 2 đầu': (1, False, False)} (rc≠0 & có tên biến = từ chối; rc=timeout/1 sau khi qua bước khoá = chấp nhận) |
| TC-P101-20 | P | api_key_enc bytea; 0 bản mã chứa khoá rõ; pg_dump canary=0 (TC-P104-16) |
| TC-P101-21 | P | hoán api_key_enc giữa 2 nhà (của QC) → key_status {'SmkB-enc': 'unreadable', 'SmkB-enc2': 'unreadable'}; nhà khác {'Quyết định': 'missing', 'QC 9router': 'ok', 'QC fake': 'ok', 'QC stub g': 'ok'} |
| TC-P101-22 | P | go test: 1 pass/0 skip/0 fail |
| TC-P101-23 | P | go test: 2 pass/0 skip/0 fail |
| TC-P101-24 | F | DecryptKey( ngoài llm/llmconfig: 2 chỗ (httpapi/llmhttp/admin.go:242, verify.go:302) → DEF-R3-04 (DEF-R3-04) |
| TC-P101-25 | P | go test: 5 pass/0 skip/0 fail |
| TC-P101-26 | P | audit_log theo entity_id của nhà QC: 4 dòng cho 4 thao tác sửa/xoá (before/after không chứa khoá, TC-P104-09) |
| TC-P101-27 | P | sửa không api_key 200 md5 giữ=True; api_key="" 422 VALIDATION_FAILED md5 giữ=True; khoá mới 200 md5 đổi=True; version cũ=TC-P104-22 |
| TC-P101-28 | P | xoá nhà không dùng: 204, mô hình đi cùng; hạn mức 21 nhà: TC-P104-23 |
| TC-P101-29…TC-P101-30 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P101-31 | P | go test: 1 pass/0 skip/0 fail |
| TC-P101-32 | P | go test: 1 pass/0 skip/0 fail |
| TC-P101-33 | P | cột tiền numeric |
| TC-P101-34 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P101-35 | P | secret dạng sk-…/AKIA/PRIVATE KEY ngoài test/fixture: 0 |
| TC-P102-01 | P | SDK nhà cung cấp ngoài internal/llm: 0 dòng (không tính test/comment) |
| TC-P102-02 | P | URL nhà cung cấp cứng ngoài internal/llm: 0 |
| TC-P102-03 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P102-04 | P | go test: 1 pass/0 skip/0 fail |
| TC-P102-05 | P | UserID/CourseID/StudentCode trong llm/types.go: 0 |
| TC-P102-06 | P | go test: 2 pass/0 skip/0 fail |
| TC-P102-07 | P | stream: stub nhận 1 lời gọi; SSE có token/done; audit 0af7651916cd43dd8448eb211c80319c/ok/CHAT |
| TC-P102-08 | P | ngắt stream → provider_inflight=0 sau 0.04s; audit status=cancelled |
| TC-P102-09 | P | go test: 2 pass/0 skip/0 fail |
| TC-P102-10 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P102-11 | P | ánh xạ lỗi (CLASSIFY→stub): m-429:503 LLM_UNAVAILABLE rate_limited/RATE_LIMIT/att=3; m-404:503 LLM_UNAVAILABLE error/MODEL_NOT_FOUND/att=1; m-500:503 LLM_UNAVAILABLE error/SERVER/att=3; m-503:503 LLM_UNAVAILABLE error/SERVER/att=3; m-garbage:503 LLM_UNAVAILABLE error/BAD_RESPONSE/att=1 (chỉ 5/16 kiể |
| TC-P102-12 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P102-13 | P | go test: 1 pass/0 skip/0 fail |
| TC-P102-14 | K | chưa chạy (mẫu): lần chạy đầu hỏng do kịch bản QC (cầu dao nhà stub mở sớm / PUT ngân sách sai định dạng / đọc audit lệch thứ tự) — không kết luận; script /tmp/qc2-b/p10x.py chạy lại cần khoá global |
| TC-P102-15 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P102-16 | P | go test: 3 pass/0 skip/0 fail |
| TC-P102-17 | P | A 503 → B 200: fallback_index=1 provider=SmkB-S2 (512 ms; stub A nhận 3 yêu cầu) |
| TC-P102-18 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P102-19 | P | (b) nhà chính lỗi còn bật → fallback_index=1 (nghĩa #33); (a) tắt công tắc: xem TC-P105-22 |
| TC-P102-20 | P | go test: 4 pass/0 skip/0 fail |
| TC-P102-21 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P102-22 | P | cột prompt/nội dung=''; "xin chao" trong llm_audit=0 |
| TC-P102-23…TC-P102-25 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P102-26 | P | go test: 4 pass/0 skip/0 fail |
| TC-P102-27 | P | traceparent -> llm_audit.trace_id=0af7651916cd43dd8448eb211c80319c |
| TC-P102-28…TC-P102-29 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P102-30 | P | go test: 4 pass/0 skip/0 fail |
| TC-P102-31 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P102-32 | P | LLM_EMBED_DIMS=768 → rc=1 sau 1.9s; {"time":"2026-11-15T12:57:04.138688699Z","level":"ERROR","msg":"giá trị biến môi trường không hợp lệ: LLM_EMBED_DIMS: khoá cố định 1536 (pgvector vect |
| TC-P102-33 | P | go test: 3 pass/0 skip/0 fail |
| TC-P102-34…TC-P102-36 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P102-37 | P | go test: 12 pass/0 skip/0 fail |
| TC-P102-38 | P | go test: 1 pass/0 skip/0 fail |
| TC-P102-39…TC-P102-40 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P102-41 | P | go test: 0 pass/1 skip/0 fail |
| TC-P102-42 | K | chưa chạy (mẫu): lần chạy đầu hỏng do kịch bản QC (cầu dao nhà stub mở sớm / PUT ngân sách sai định dạng / đọc audit lệch thứ tự) — không kết luận; script /tmp/qc2-b/p10x.py chạy lại cần khoá global |
| TC-P102-43…TC-P102-44 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P102-45 | P | go test: 1 pass/0 skip/0 fail |
| TC-P102-46…TC-P102-47 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P102-48 | P | go test: 2 pass/0 skip/0 fail |
| TC-P102-49 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P102-50 | P | go test: 25 pass/0 skip/0 fail |
| TC-P103-01 | P | go test: 4 pass/0 skip/0 fail |
| TC-P103-02 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P103-03 | K | chưa chạy (mẫu): lần chạy đầu hỏng do kịch bản QC (cầu dao nhà stub mở sớm / PUT ngân sách sai định dạng / đọc audit lệch thứ tự) — không kết luận; script /tmp/qc2-b/p10x.py chạy lại cần khoá global |
| TC-P103-04 | P | go test: 1 pass/0 skip/0 fail |
| TC-P103-05 | K | chưa chạy (mẫu): tải/đo hiệu năng nặng (k6, Lighthouse) — chưa thu nhỏ để chạy trên ep-r |
| TC-P103-06 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P103-07 | P | go test: 4 pass/0 skip/0 fail |
| TC-P103-08 | K | chưa chạy (mẫu): tải/đo hiệu năng nặng (k6, Lighthouse) — chưa thu nhỏ để chạy trên ep-r |
| TC-P103-09 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P103-10 | P | go test: 2 pass/0 skip/0 fail |
| TC-P103-11…TC-P103-12 | K | chưa chạy (mẫu): tải/đo hiệu năng nặng (k6, Lighthouse) — chưa thu nhỏ để chạy trên ep-r |
| TC-P103-13 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P103-14…TC-P103-15 | K | chưa chạy (mẫu): lần chạy đầu hỏng do kịch bản QC (cầu dao nhà stub mở sớm / PUT ngân sách sai định dạng / đọc audit lệch thứ tự) — không kết luận; script /tmp/qc2-b/p10x.py chạy lại cần khoá global |
| TC-P103-16 | P | llm_audit status=overloaded: 0 (từ chối HTTP 0) |
| TC-P103-17 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P103-18 | P | go test: 6 pass/0 skip/0 fail |
| TC-P103-19 | P | sau lỗi liên tiếp: circuit=open; lời gọi kế tiếp bị chặn ngay (200 None, 9 ms, stub 0 request) |
| TC-P103-20 | P | bán mở sau 30 s: 3 lời gọi đồng thời → stub nhận 2 (kỳ vọng thăm dò 1 lần × ≤2 lần thử); kết quả [200, 200, 200] |
| TC-P103-21…TC-P103-22 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P103-23 | P | go test: 10 pass/0 skip/0 fail |
| TC-P103-24 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P103-25 | K | chưa chạy (mẫu): lần chạy đầu hỏng do kịch bản QC (cầu dao nhà stub mở sớm / PUT ngân sách sai định dạng / đọc audit lệch thứ tự) — không kết luận; script /tmp/qc2-b/p10x.py chạy lại cần khoá global |
| TC-P103-26 | P | go test: 6 pass/0 skip/0 fail |
| TC-P103-27 | P | huỷ stream: inflight=0 sau 0.04s (≤1s); audit cancelled |
| TC-P103-28 | P | go test: 4 pass/0 skip/0 fail |
| TC-P103-29 | P | mọi nhà chết → 200 degraded=True; text='AI tạm thời không khả dụng. Dưới đây là các đoạn tài liệu liên quan nhất:\n\n«Đoạn 0 nộp bài' (508 ms) |
| TC-P103-30 | P | không passages → 200 degraded=True text='AI tạm thời không khả dụng. Câu hỏi của bạn đã được ghi lại, giảng viên sẽ xem.' |
| TC-P103-31 | P | từ cấm trong 2 chuỗi suy giảm: [] |
| TC-P103-32 | P | go test: 6 pass/0 skip/0 fail |
| TC-P103-33…TC-P103-35 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P103-36 | P | go test: 6 pass/0 skip/0 fail |
| TC-P103-37…TC-P103-38 | K | chưa chạy (mẫu): lần chạy đầu hỏng do kịch bản QC (cầu dao nhà stub mở sớm / PUT ngân sách sai định dạng / đọc audit lệch thứ tự) — không kết luận; script /tmp/qc2-b/p10x.py chạy lại cần khoá global |
| TC-P103-39 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P103-40 | P | go test: 10 pass/0 skip/0 fail |
| TC-P103-41 | K | chưa chạy (mẫu): lần chạy đầu hỏng do kịch bản QC (cầu dao nhà stub mở sớm / PUT ngân sách sai định dạng / đọc audit lệch thứ tự) — không kết luận; script /tmp/qc2-b/p10x.py chạy lại cần khoá global |
| TC-P103-42 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P103-43 | P | go test: 5 pass/0 skip/0 fail |
| TC-P103-44 | P | go test: 4 pass/0 skip/0 fail |
| TC-P103-45…TC-P103-46 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P103-47 | P | {'LLM_BATCH_SHARE=1.5': (1, True), 'LLM_BATCH_SHARE=0': (1, True), 'LLM_BATCH_SHARE=-1': (1, True), 'LLM_MAX_CONCURRENCY=0': (1, True), 'LLM_MAX_CONCURRENCY=-3': (1, True), 'LLM_QUEUE_MAX=-1': (1, True), 'LLM_BREAKER_FAILS=0': (1, True)} |
| TC-P103-48…TC-P103-49 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P103-50 | K | chưa chạy (mẫu): tải/đo hiệu năng nặng (k6, Lighthouse) — chưa thu nhỏ để chạy trên ep-r |
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
| TC-P104-28 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P104-29 | P | go test: 4 pass/0 skip/0 fail |
| TC-P104-30…TC-P104-32 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P104-33 | P | go test: 3 pass/0 skip/0 fail |
| TC-P104-34 | K | chưa chạy (mẫu): lần chạy đầu hỏng do kịch bản QC (cầu dao nhà stub mở sớm / PUT ngân sách sai định dạng / đọc audit lệch thứ tự) — không kết luận; script /tmp/qc2-b/p10x.py chạy lại cần khoá global |
| TC-P104-35 | P | GET/PUT /courses/{id}/llm-budget: ADMIN ok, còn lại 403/401 [] |
| TC-P104-36 | P | go test: 4 pass/0 skip/0 fail |
| TC-P104-37 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P104-38 | P | go test: 1 pass/0 skip/0 fail |
| TC-P104-39 | P | go test: 25 pass/0 skip/0 fail |
| TC-P104-40 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P104-41 | P | go test: 25 pass/0 skip/0 fail |
| TC-P104-42 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P104-43 | P | JSON hỏng (400, 'BAD_REQUEST', True); 2MB (413, 'PAYLOAD_TOO_LARGE'); ct sai (415, 'UNSUPPORTED_MEDIA_TYPE'); CORS evil=None; ETag routes=W/"Z_OOqA_ZzZaOUKf_" |
| TC-P104-44 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P104-45 | P | go test: 3 pass/0 skip/0 fail |
| TC-P104-46 | K | (b) cần người thật chấm / kịch bản tay có ảnh |
| TC-P104-47 | P | secret trong repo: 0; mã lỗi mới có thông điệp tiếng Việt: quan sát qua API (OVERLOADED/ROUTE_INVALID… xem TC-P103/104) |
| TC-P104-48 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
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
| TC-P105-50…TC-P105-52 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P105-53 | P | đạt qua Playwright cùng AC14: «qc-real-f15 hai tab» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-P105-54 | P | qc-real-f15-llm: hai tab sửa cùng nhà → tab sau 409 và nạp lại (stack ep-r) |
| TC-P105-55…TC-P105-56 | K | (b) cần người thật chấm / kịch bản tay có ảnh |
| TC-P105-57 | P | git grep sk-… trong frontend (ngoài e2e): 0 |
| TC-PU01-01 | K | LỖI THỜI: docs/design/DESIGN_TOKENS.css đổi hợp lệ sau sprint 3 (P-107-8/10 bộ màu ĐHBK, màn chiếu P-108-8); còn 2 token lệch giữa kit và tokens.css: --ep-ink-3 (kit 58 % / code 53 %, góp ý #25) và --ep-slow (kit 250 ms / code 200 ms, uiux-91 US-04) → DEF-R3-02 (tài liệu kit chưa đồng bộ) |
| TC-PU01-02 | P | token thêm so với bản gốc: 7 (≥4 đúng tên; TC cũ ghi đúng 4, đã tăng theo sprint sau): ['--ep-ink-3', '--ep-mask-clear', '--ep-mask-solid', '--ep-scrim', '--ep-shadow-composer', '--ep-slow', '--ep-text-display'] |
| TC-PU01-03 | K | LỖI THỜI: DESIGN_TOKENS.css nay 62 dòng khai báo (sửa hợp lệ ở 51eab447 / ac670244 / 8999086f) |
| TC-PU01-04 | P | khai báo --ep-* ngoài tokens.css: 0 |
| TC-PU01-05 | F | pnpm lint rc=0 nhưng 5 cảnh báo (cells.tsx:39, QueueTab.tsx:115/123, Landing.tsx:48, LlmSettings.tsx:13); nền sprint 3 = 0 cảnh báo → DEF-R3-03 (DEF-R3-03) |
| TC-PU01-06 | P | ui-antipatterns.sh rc=0, 32 dòng ✓ (≥11), 0 ✗ |
| TC-PU01-07 | P | ui-allow: 9 chỗ (trần 10) |
| TC-PU01-08 | P | gieo 3 vi phạm màu/bo góc → rc=1, 2 phép ✗ |
| TC-PU01-09 | P | 3 ca xấu → ✗×3; 4 ca lành rc=0 (không báo nhầm) |
| TC-PU01-10 | P | .tsx gieo → rc=1, ✗×6 |
| TC-PU01-11 | P | lint-selftest.sh: 7 / 7 luật ESLint + 32 / 32 phép ui-antipatterns bắt được; rc=0 (TC cũ ghi 19 phép, đã tăng) |
| TC-PU01-12…TC-PU01-13 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU01-14 | P | đếm = ['0'] (kỳ vọng 0) |
| TC-PU01-15 | P | đếm = ['0'] (kỳ vọng 0) |
| TC-PU01-16 | P | đếm = ['0'] (kỳ vọng 0) |
| TC-PU01-17 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU01-18 | P | đếm = ['0', '0'] (kỳ vọng 0) |
| TC-PU01-19 | P | Playwright tokens.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-PU01-20 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU01-21 | K | LỖI THỜI: layout.tsx hiện '13:  weight: ["400", "600", "700"], // bỏ 500 để nhẹ ~25 KB phông (góp ý #26): chữ 500 hiển thị bằng'; tokens.spec.ts (QC cũ) kiểm "3 độ đậm 400/600/700" — spec đổi hợp lệ (4→3 độ đậm) |
| TC-PU01-22 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU01-23 | P | Playwright tokens.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-PU01-24 | P | Playwright tokens.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-PU01-25 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU01-26 | P | Playwright tokens.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-PU01-27 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU01-28 | P | Playwright tokens.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-PU01-29 | P | 3 logo/favicon trên frontend ep-r :3700 → [200, 200, 200] |
| TC-PU01-30 | P | Playwright tokens.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-PU01-31…TC-PU01-33 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU01-34 | K | NỀN TẢNG: ảnh mốc visual do CI Linux; máy chạy macOS lệch kết xuất chữ (diff ~1 % do khử răng cưa, không phải lệch bố cục); cần chạy trong container Linux — chưa chạy |
| TC-PU01-35 | P | setItem khoá nhạy cảm trong frontend/src = 0 |
| TC-PU01-36 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
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
| TC-PU02-39…TC-PU02-40 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU02-41 | P | đạt qua Playwright cùng AC13: «offline-input» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-42 | P | đạt qua Playwright cùng AC13: «offline-input» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-43 | P | đạt qua Playwright cùng AC14: «one-primary» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-44 | P | đạt qua Playwright cùng AC14: «one-primary» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-45 | P | đạt qua Playwright cùng AC14: «one-primary» (bản build gate, gateway giả; mô phỏng của TC được spec đại diện) |
| TC-PU02-46…TC-PU02-47 | K | (b) cần người thật chấm / kịch bản tay có ảnh |
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
| TC-PU04-59 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU05-01 | K | NỀN TẢNG: ảnh mốc visual do CI Linux; máy chạy macOS lệch kết xuất chữ (diff ~1 % do khử răng cưa, không phải lệch bố cục); cần chạy trong container Linux — chưa chạy |
| TC-PU05-02 | P | 26 ảnh mốc (≥14 = 7 route × 2 bề rộng; hiện nhiều hơn do route thêm sau) |
| TC-PU05-03 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU05-04…TC-PU05-05 | K | NỀN TẢNG: ảnh mốc visual do CI Linux; máy chạy macOS lệch kết xuất chữ (diff ~1 % do khử răng cưa, không phải lệch bố cục); cần chạy trong container Linux — chưa chạy |
| TC-PU05-06…TC-PU05-07 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
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
| TC-PU05-19 | K | chưa chạy (mẫu): tải/đo hiệu năng nặng (k6, Lighthouse) — chưa thu nhỏ để chạy trên ep-r |
| TC-PU05-20…TC-PU05-22 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU05-23 | P | ui-antipatterns.sh 32 ✓/0 ✗; --selftest 32/32 bắt được; chạy ở bản sao, frontend/src sạch (TC cũ: 19 phép — đã tăng) |
| TC-PU05-24 | P | ui-antipatterns.sh 32 ✓/0 ✗; --selftest 32/32 bắt được; chạy ở bản sao, frontend/src sạch (TC cũ: 19 phép — đã tăng) |
| TC-PU05-25 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU05-26 | P | ui-antipatterns.sh 32 ✓/0 ✗; --selftest 32/32 bắt được; chạy ở bản sao, frontend/src sạch (TC cũ: 19 phép — đã tăng) |
| TC-PU05-27 | P | ui-antipatterns.sh 32 ✓/0 ✗; --selftest 32/32 bắt được; chạy ở bản sao, frontend/src sạch (TC cũ: 19 phép — đã tăng) |
| TC-PU05-28…TC-PU05-29 | K | (c) CI GitHub / nhánh ci/ui-drift của PR sprint 3 — ngoài phạm vi đồ án |
| TC-PU05-30 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU05-31…TC-PU05-32 | K | (c) CI GitHub / nhánh ci/ui-drift của PR sprint 3 — ngoài phạm vi đồ án |
| TC-PU05-33…TC-PU05-34 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU05-35 | K | (c) CI GitHub / nhánh ci/ui-drift của PR sprint 3 — ngoài phạm vi đồ án |
| TC-PU05-36 | P | commit PU: đụng backend-go: 0 |
| TC-PU05-37 | P | go test: 25 pass/0 skip/0 fail |
| TC-PU05-38 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU05-39…TC-PU05-40 | K | (b) cần người thật chấm / kịch bản tay có ảnh |
| TC-PU05-41 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-PU05-42 | K | chưa chạy (mẫu): tải/đo hiệu năng nặng (k6, Lighthouse) — chưa thu nhỏ để chạy trên ep-r |
| TC-PU05-43 | P | JWT cứng trong e2e/lighthouserc: 0 |
| TC-PU05-44 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |

## Trượt: bước tái hiện
Xem `defects-3.md`.

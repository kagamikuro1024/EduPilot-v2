# QC — kiểm lát L2 + L3 (US-DM-03, US-DM-04) và phần L1 còn lại, ở `6bc650f8` — stack `ep-qc` thật
2026-11-14. Stack dựng từ trống ở `6bc650f8` (`-p ep-qc`, offset 500, docling bật để seed, **provider giả**: `LLM_PROVIDER=fake`, `decision-fake`, embedding giả, `GEMINI_API_KEY` để trống, `LLM_DECISION_ALLOW_REAL=false`; seed `scripts/seed.mjs` mã 0, 670 s). Không gọi Gemini / `gemini-decision-1` / 9router. Máy: MacBook Apple Silicon, Colima 6 CPU / 10 GB. Cách điều khiển: bật cổng bằng ghi DB (`decision_gates`, `course_decision_gates`) + `PUBLISH ep:decision:cfg`; xác suất `FAKE_DECISION_PROB` / `LLM_DECISION_FAKE_MODE` bằng khởi động lại; kết quả `Structured` (tác vụ `CLASSIFY`) bằng **máy chủ giả OpenAI-compatible của QC** (`scripts/stub_llm.py`, route `CLASSIFY` trỏ vào đó). Script: `scripts/dm_a_provider.py`, `dm_b2.py`, `dm_b3.py`, `dm_c.py`. Kết luận **một phần** (US-DM-03: PASS phần đã chạy; US-DM-04: PASS mức đã chạy, còn mức 6/6 biến thể).

## Lỗi
| Mã | Mức | TC | Mô tả, tái hiện | Nghi ngờ |
| --- | --- | --- | --- | --- |
| DM-BUG-1 | **Trung bình** | TC-D01-19, D01-28b | ADMIN `PUT /admin/llm/providers/{id}` (và `POST`) với `type:"decision"` / model `kind:"decision"` → `422 VALIDATION_FAILED` (`details`: `type` và `kind` `oneof`). Hệ quả: không chọn được `gemini-decision-1` qua API nên `409 DECISION_REAL_DISABLED` **không bao giờ tới được**; không tạo / sửa được provider `decision` (kể cả cập nhật `decision-fake`). Openapi liệt kê `decision` trong enum nhưng bộ kiểm không | `backend-go/internal/httpapi/llmhttp/admin.go:112,120`, `verify.go:313,317` (`validate:"…oneof=…fake"` thiếu `decision`; `kind` thiếu `decision`) |
| DM-BUG-2 | **Trung bình** | TC-D01-26 | `teacher@` `GET /admin/llm/providers` → `200` và **thấy** provider `type=decision` (TA, SV: `403`). SRS §2: Teacher không thấy cấu hình decision. (Endpoint cũ cho TEACHER đọc danh sách) | bộ lọc danh sách provider ở `llmhttp` cho vai TEACHER |
| DM-BUG-3 | Thấp | TC-D01-27 | `POST /admin/llm/providers/{id}/test` với `decision-fake` → `200` nhưng thân `{"ok":false,"error_kind":"NETWORK","message":"Không kết nối được …"}`; spec: thử cục bộ bằng một câu `noul` vô hại → `ok:true` với bản giả (không mạng) | `llmhttp` test provider |
| DM-BUG-4 | Thấp | TC-D04-* (cách điều khiển) | `docker-compose.local.yml` chuyển `LLM_DECISION_*` / `FAKE_DECISION_PROB` chỉ cho `gateway` (dòng ~231–232), **không** cho `worker` — mà cổng chèn lệnh chạy ở worker. Nên E2E / đo không điều khiển được bản giả cho cổng chấm bằng env compose (QC dùng `scripts/qc-worker-override.yml` cục bộ). Dev đã thêm `DECISION_ALERT_*` ở `acb704cc` nhưng chưa thêm cho worker | `docker-compose.local.yml` service `worker` |
| DM-BUG-5 | Thấp | US-DM-08 AC2 | Nhiều test **được spec / lệnh cổng nêu tên không tồn tại**: `TestInjectionOR`, `TestInjectionEmptySkipped`, `TestInjectionScoreUnchanged`, `TestInjectionSet6of6`, `TestRouteFallbackToRules`, `TestCallBudgetPerMessage`, `TestDecisionRBAC`, `TestDecisionGateVersionConflict`, `TestDecisionAudit` (có `TestDecisionAudited`). Dev dùng tên khác (`TestInjectionFlagTiers`, `TestRouteDecisionTable`, `TestDecisionAdminOnly`, `TestDecisionConflict409`…). Lệnh `go test -run 'A\|B\|…'` của AC2 vẫn "ok" nhưng **không chạy** các test đó (không lỗi khi thiếu) → cổng yếu | `grep -rn "func Test…" backend-go/internal` |

## Kết quả TC đã chạy
**L1 còn lại (API provider #6, phân quyền, mặc định)**
| TC | Kết quả | Ghi chú |
| --- | --- | --- |
| TC-D01-19 | **FAIL** | DM-BUG-1 |
| TC-D01-19b, D01-28b | PASS (mô hình không đổi; tạo provider thứ hai bị từ chối 4xx) — nhưng lý do là DM-BUG-1, không phải unique index (đã PASS ở mức DB) | |
| TC-D01-26 | **FAIL** (teacher) / PASS (TA, SV, không token: `403`/`401`; `PUT`, `POST test`: `403`) | DM-BUG-2 |
| TC-D01-27 | **FAIL** | DM-BUG-3 |
| TC-D02-13, D02-06 | PASS | ba cổng tắt, ngưỡng đúng, 0 `course_decision_gates` bật, 0 dòng `DECIDE` sau seed |

**US-DM-03 — cổng Threads (stub `Structured`: allow / block / error / invalid / slow)**
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| TC-D03-off (cổng tắt: đăng được, 0 `DECIDE`) / TC-D02-07…09 | PASS | cổng tắt, hoặc chỉ bật hệ thống, hoặc chỉ lớp 2 → 0 `DECIDE` ở lớp 1 / 2 đúng theo bật; lớp 2 (cổng chỉ lớp 1) 201, 0 `DECIDE` |
| TC-D03-01 (luật chặn → 0 `DECIDE`) | PASS | MSSV → `422 PII_DETECTED`, 0 `DECIDE` |
| TC-D03-09 (chỉ placeholder) | PASS | `422`, 0 `DECIDE` |
| TC-D03-10 (precheck lúc gõ) | PASS | 3 lần `precheck`: 0 `DECIDE` |
| TC-D03-14 (khủng hoảng) | PASS | `422 CRISIS_SUPPORT`, 0 `DECIDE`, 0 `CLASSIFY` |
| TC-D03-02 (P=0,9) | PASS | `422`, `FAST`, đúng 1 dòng `DECIDE`, **0** `CLASSIFY` |
| TC-D03-03 (P=0,01 ≤ ε) | PASS | `201`, `FAST`, 0 `CLASSIFY` |
| TC-D03-04 (P=0,2 lưỡng lự) | PASS | `FALLBACK` + 1 `CLASSIFY`; stub `allow` → `201`; stub `block` → `422` |
| TC-D01-06 / D03-04 (đầu vào lạ → 0,25) | PASS | `FALLBACK`, đi `Structured` (P-105-1 Q3) |
| TC-D03-05 (decision lỗi, `FAKE_MODE=error`) | PASS | `DECIDE` `error`/`FALLBACK`; stub allow → `201`; **cả hai lỗi → `422` `FAIL_CLOSED`** (TC-D03-06) |
| TC-D03-06 (cả hai lỗi: decision + `Structured` lỗi / sai JSON) | PASS | `FAIL_CLOSED`, bài bị chặn (`422`), người dùng thấy lỗi `PII_DETECTED` quen thuộc |
| TC-D03-07 (`Structured` chậm 12 s) | PASS | `422` sau **6,36 s** (`THREADS_STRUCTURED_TIMEOUT=6s`), `CLASSIFY` `timeout`, `FAIL_CLOSED` |
| TC-D03-17 (lớp 2 không gọi) | PASS | lớp 2 `201`, 0 `DECIDE` |
| TC-D01-09 (một dòng `DECIDE` / lời gọi: `lane=INTERACTIVE`, `provider`, `model`, `cascade_gate`, `cascade_path`, `status`) | PASS | `llm_audit`: `Quyết định` / `decision-fake` / `THREADS_PRECHECK` / `INTERACTIVE` |
| Ghi nhận: một dòng `DECIDE` có `status=timeout` ở 821 ms khi `FAKE_P95_MS=682` (hạn 800 ms): đuôi log-chuẩn của bản giả, đúng hành vi | – | |
| TC-D03-11/12/13/15/16/18/19/20 (comment / edit, hộp thoại UI, chỉ báo, E2E, bộ E1 `dev`, PII payload, vai Teacher / TA, đua) | **CHƯA CHẠY** | comment / edit và vai Teacher–TA lượt đầu gặp 422 do cấu hình stub; chạy lại ở L4–L5 cùng stack test; UI / E1 `dev` chờ lượt kế |

**US-DM-04 — cổng chèn lệnh** (16 bài seed BT03 chấm lại round 2+; so với round 1 cổng tắt; fake LLM chấm theo mẫu S1…S7; xác suất do `FAKE_DECISION_PROB` ở **worker** qua override)
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| TC-D04-01/02 (OR ba tầng) | PASS (phần decision + regex) | P=0,9: **14** bài mới có `INJECTION_SUSPECTED` (15 bài đủ điều kiện − `sv14` đã bị regex bắt) và `sv14` (S6) giữ cờ regex; P=0,01 hoặc lỗi → 0 cờ mới, `sv14` vẫn có cờ (regex) |
| TC-D04-03 (chỉ cờ, điểm không đổi) | PASS | tổng điểm 16/16 bài **y hệt round 1** ở P=0,9, P=0,01, lỗi |
| TC-D04-05 (bài trống / < 20 từ) | PASS một phần | `sv24` (S4 trống): `EMPTY_SUBMISSION`, **không** `INJECTION_SUSPECTED`; **15 lời gọi `DECIDE` cho 16 bài** (loại đúng bài trống). Chưa kiểm biên 19 / 20 từ |
| TC-D04-12 (làn, thời hạn) | PASS | `DECIDE` làn `BATCH`, `cascade_gate=INJECTION_FLAG` |
| TC-D04-14 (cầu dao / lỗi) | PASS | `FAKE_MODE=error`: 5 `error` rồi **10 `circuit_open`** (cầu dao mở), mọi bài chấm xong, điểm không đổi, cờ chỉ regex |
| TC-D04-19 (cổng tắt) | PASS | 0 `DECIDE` khi cổng tắt |
| TC-D04-09 (6/6 + 0 cờ trên 6 đối chứng), 07 (cắt 16.000 ký tự), 16 (SV không thấy cờ), 17, 18, 20 | **CHƯA CHẠY** | fake LLM chấm theo mẫu cố định nên không có biến thể `v1…v5`; cần bài nộp mới + hướng `Structured` điều khiển được — kiểm ở lượt kế (route `GRADING` trỏ stub QC) |
| Ghi nhận | – | với P ≤ ε hoặc lỗi, `cascade_path` của cổng chèn lệnh ghi `FALLBACK` (không `FAST` cho «không»): phù hợp spec (chỉ cờ khi `P ≥ τ`), nên giữ nguyên |

## Chưa kiểm được ở lượt này
TC-D03-11…13, 15, 16, 18…20; TC-D04-07, 09, 16…18, 20; TC-D01-03, 15, 21 (máy chủ giả cho đường mã provider thật — cần DM-BUG-1 sửa để tạo provider với `base_url`); TC-D02-14, 16 (cấu hình không DB trên đường nóng, hai gateway); mọi TC có «LLM thật» / «chờ phép PM». Kiểm tiếp ở L4–L5 trên stack `testroutes` (`docker-compose.test.yml`).

# Smoke hồi quy sprint 10.5 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm G, SmkG) · Ngày: 2026-11-15
Tổng TC cũ: 185 · Đạt: 80 · Trượt: 0 · Không chạy (K, gồm lỗi thời): 105   (P+F+K = 185)
Tách K: K(a/b/c) = 0 (a=0, b=0, c=0) · lỗi thời = 2 · **chưa chạy (mẫu) = 103** (ngoài mẫu smoke «1 chính + 1 lỗi/phân quyền mỗi US»; chờ PM quyết có chạy nốt)

Cách chạy (bản sao script ở `docs/regression/evidence-10.5/scripts/`, sửa cho ep-r): `dm_a_provider` (23 PASS), `dm_e_api` (58 PASS), `dm_f_alert_obs` (obs khớp SQL), `dm_smoke.py` (mới: ma trận cổng theo cấu hình) qua các lần restart gateway+worker dưới khoá toàn cục với `FAKE_DECISION_PROB`=0,9/0,01/0,2/0,97/mặc định, `LLM_DECISION_FAKE_MODE`=error/timeout, CLASSIFY→stub :8778 (đã hoàn); `dm_b_threads` (rules/precheck); `dm_c` (cờ chèn lệnh, 3 cấu hình — kết quả KHÔNG hợp lệ: chỉ 7–22/24 bài chấm lại được, P=0,9 chưa chạy xong hàng đợi, dữ liệu BT03 bị lẫn bởi 10.6 → ca D04-01/02/03/05/09/14/20 ghi chưa chạy); Go gate (llm, agent, thread, privacy, grading) PASS; Playwright `qc-dm-ui-real` 3 xanh; `LLM_DECISION_ALLOW_REAL=false` giữ nguyên, 0 gói tới googleapis. Chưa chạy: cảnh báo DECISION_FAIL_CLOSED (cần `DECISION_ALERT_TICK` + dữ liệu mới), `benchmarks/decision/run.sh`, thêm ca Threads comment/roles.

## Tổng hợp theo US
| US | TC tổng | P | F | K(a/b/c)+lỗi thời | chưa chạy (mẫu) | Ghi chú |
| --- | --- | --- | --- | --- | --- | --- |
| US-DM-01 `Decide()` và provider | 28 | 18 | 0 | 0 | 10 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| US-DM-02 Cascade + cấu hình | 25 | 15 | 0 | 0 | 10 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| US-DM-03 Cổng `THREADS_PRECHECK` | 20 | 11 | 0 | 0 | 9 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| US-DM-04 Cổng `INJECTION_FLAG` | 20 | 5 | 0 | 2 | 13 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| US-DM-05 Cổng `CHAT_ROUTING` | 19 | 6 | 0 | 0 | 13 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| US-DM-09 Ranh giới an toàn (xuyên suốt) | 11 | 6 | 0 | 0 | 5 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| US-DM-06 Màn cấu hình | 23 | 7 | 0 | 0 | 16 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| US-DM-07 Bảng số liệu | 16 | 7 | 1 | 0 | 8 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |
| QC test case — US-DM-08 (đo trước / sau, seed, hướng dẫn, cổ | 23 | 4 | 0 | 0 | 19 | mẫu 1 chính + 1 lỗi/phân quyền; xem bảng từng ca |

## Kết quả từng ca (đã chạy: P/F; K gom theo dải)
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-D01-01 | P | internal/llm: Client.Decide / DecideRequest / DecideResponse có mặt; chỉ ADMIN cấu hình |
| TC-D01-02 | P | go list -deps ./...: 0 gói google.golang.org/genai |
| TC-D01-03 | P | Go TestDecideTimeoutRedirectAndNoGeneration / TestDecideErrorMap / TestDecideNoSDK (provider) PASS (máy chủ giả qua HTTP: chưa chạy) |
| TC-D01-04 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D01-05 | P | Go TestDecideFakeDeterministic PASS |
| TC-D01-06 | P | stack: đầu vào lạ → P=0,25 → FALLBACK + Structured |
| TC-D01-07 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D01-08 | P | LLM_DECISION_FAKE_MODE=timeout: DECIDE timeout rồi FALLBACK; stub allow → 201 (restart dưới khoá, đã hoàn) |
| TC-D01-09 | P | llm_audit: 1 dòng DECIDE / lời gọi: THREADS_PRECHECK / FAST|FALLBACK / ok|error|timeout / INTERACTIVE, provider Quyết định, model decision-fake |
| TC-D01-10 | P | Go TestDecideMasked PASS (thân tới provider đã che) |
| TC-D01-11…TC-D01-12 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D01-13 | P | mỗi lần đăng bài / mỗi tin chat: đúng 1 dòng DECIDE (≤ 1) |
| TC-D01-14 | P | Go TestCascadeTimeout PASS |
| TC-D01-15 | P | Go TestDecideBudgetNoCheapest / TestDecideInputCeiling PASS |
| TC-D01-16…TC-D01-18 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D01-19 | P | gemini-decision-1 BẬT khi ALLOW_REAL=false → 409 DECISION_REAL_DISABLED, mô hình không đổi |
| TC-D01-20…TC-D01-21 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D01-22 | P | .env.example: LLM_DECISION_ALLOW_REAL=false, GEMINI_API_KEY trống; compose không đặt mặc định bật; gateway + worker ALLOW_REAL=false |
| TC-D01-23 | P | llm_models kind=decision: decision-fake giá 0; gemini-decision-1 2600 / 0 |
| TC-D01-24 | P | Go TestDecide* PASS (llm, llm/fake, llm/provider, llm/scheduler) |
| TC-D01-25 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D01-26 | P | teacher/TA/SV: GET provider decision ẩn/403, PUT + POST test 403; không token 401; teacher không thấy tác vụ DECIDE |
| TC-D01-27 | P | POST providers/{id}/test với decision-fake → 200 ok:true (cục bộ) |
| TC-D01-28 | P | tạo provider decision thứ hai bị từ chối 422; baseline 1 provider decision |
| TC-D02-01 | P | Go TestCascadeTable / TestCascadeFailClosed / TestCascadeNotUsed PASS |
| TC-D02-02 | P | cascade_path FAST / FALLBACK / FAIL_CLOSED quan sát đủ ở Threads, chat (P=0,9 / 0,01 / 0,2 / lỗi) |
| TC-D02-03 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D02-04 | P | mỗi lần cascade ghi cascade_gate + cascade_path vào llm_audit |
| TC-D02-05 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D02-06 | P | mặc định: ba cổng tắt, 0 course_decision_gates bật, 0 dòng DECIDE khi cổng tắt |
| TC-D02-07…TC-D02-10 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D02-11 | P | ngưỡng / hạn ở biên: 0,01/0,20/0,70/0,99/0,30/0,90, 200/5000 → 200; ngoài biên → 422 |
| TC-D02-12 | P | gate lạ 404; thân rỗng / sai kiểu → 422, không 5xx |
| TC-D02-13 | P | mặc định THREADS ε0,05/800, INJECTION τ0,5/3000, CHAT τ0,95/800, tắt |
| TC-D02-14…TC-D02-16 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D02-17 | P | PUT đổi enabled / ngưỡng → 200, version+1, audit_log |
| TC-D02-18 | P | version cũ → 409 VERSION_CONFLICT |
| TC-D02-19 | P | PUT theo lớp 200; lớp lạ / UUID rác → 4xx |
| TC-D02-20 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D02-21 | P | Go TestPrecheckAddOnly PASS (decision chỉ thêm chặn, không bỏ chặn của luật) |
| TC-D02-22 | P | teacher/TA/SV: 19 API cổng 403; không token 401 |
| TC-D02-23 | P | dm_m: SV/GV/TA × 30 phản hồi (me, courses, threads, assignments, notifications, chat, grading-results, settings): 0 cascade_* / tên cổng / FAST·FALLBACK |
| TC-D02-24 | P | GET cổng theo lớp: 3 cổng {enabled, effective, version} |
| TC-D02-25 | P | GET decision/courses: cursor duyệt hết, không trùng |
| TC-D03-01 | P | luật chặn (MSSV) → 422 PII_DETECTED, 0 DECIDE |
| TC-D03-02 | P | P=0,9: 422, FAST, đúng 1 dòng DECIDE, 0 CLASSIFY (hộp thoại UI: chưa chạy) |
| TC-D03-03 | P | P=0,01: 201, FAST, 0 CLASSIFY |
| TC-D03-04 | P | P=0,2: FALLBACK + 1 CLASSIFY; stub allow → 201; stub block → 422 |
| TC-D03-05 | P | FAKE_MODE=error: DECIDE error → FALLBACK; stub allow → 201 |
| TC-D03-06 | P | decision lỗi + Structured lỗi / sai JSON → 422, FAIL_CLOSED (hộp thoại UI: chưa chạy) |
| TC-D03-07 | P | Structured chậm 12 s → 422 sau ≈ 6,9 s, FAIL_CLOSED |
| TC-D03-08 | P | Go TestPrecheckAddOnly PASS (E1 dev: chưa chạy) |
| TC-D03-09 | P | chỉ placeholder → 422 bởi luật, 0 DECIDE |
| TC-D03-10 | P | precheck lúc gõ: 0 DECIDE |
| TC-D03-11…TC-D03-13 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D03-14 | P | chat khủng hoảng: 0 DECIDE; Threads khủng hoảng: Go TestCrisisNeverReachesDecide PASS (lượt stack bị 503 do chaos nhóm khác, chưa chạy lại) |
| TC-D03-15…TC-D03-20 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D04-01 | K (lỗi thời) | LỖI THỜI (P-11-26): bản phát lại decision-replay.json nay chứa mẫu chấm của seed nên FAKE_DECISION_PROB không ghi đè được (chỉ áp cho đầu vào ngoài replay) → không dựng được bảng OR 7 tổ hợp bằng P; với P=0,9 chỉ 1–6 lời gọi FAST, 20 FALLBACK |
| TC-D04-02 | P | FAKE_MODE=error: DECIDE error → circuit_open; mọi bài chấm xong (22/22 DONE), cờ chỉ từ regex (sv14), điểm khớp các round cùng điều kiện, hàng chờ không treo (batchB5c/B6) |
| TC-D04-03 | P | điểm y hệt giữa cổng tắt / P=0,9 / P=0,01 / lỗi trên cùng round-điều kiện (vd sv07 5,75, sv15 9,00 ở round 2–6, 8–10); chênh với round 1 là do round 1 chấm trước khi nhóm khác đổi route, không do cổng (batchB6: cổng TẮT cũng cho đúng các chênh đó) |
| TC-D04-04…TC-D04-08 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D04-09 | K (lỗi thời) | LỖI THỜI (P-11-26): 6 biến thể + 6 đối chứng nay đo bằng bộ replay riêng (P-11-26: 0 cờ nhầm trên bộ sạch); không đo được bằng FAKE_DECISION_PROB |
| TC-D04-10…TC-D04-11 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D04-12 | P | DECIDE của chấm: làn BATCH, cascade_gate=INJECTION_FLAG |
| TC-D04-13 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D04-14 | P | FAKE_MODE=error: DECIDE error → circuit_open (cầu dao mở), cờ chỉ từ regex (sv14 giữ INJECTION_SUSPECTED), 0 cờ mới; chấm vẫn chạy (một phần bài còn QUEUED lúc đọc) |
| TC-D04-15…TC-D04-18 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D04-19 | P | cổng tắt: 0 dòng DECIDE khi chấm lại (dm_c off) |
| TC-D04-20 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D05-01 | P | tin cụ thể («Điểm chuyên cần của tôi…»): 0 dòng DECIDE |
| TC-D05-02 | P | tin COURSE_QA mơ hồ: đúng 1 dòng DECIDE, cascade_gate=CHAT_ROUTING (P=0,9 → FALLBACK; P=0,97 → FAST) |
| TC-D05-03…TC-D05-05 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D05-06 | P | P=0,97 (≥ τ 0,95) → FAST; P=0,9 và 0,25 → FALLBACK (giữ luật); 0 dòng CLASSIFY thêm |
| TC-D05-07 | P | FAKE_MODE=error: chat vẫn 200, ≤ 1 DECIDE (error → FALLBACK), TTFB 35 ms; (timeout / cầu dao mở / budget_blocked: chưa chạy) |
| TC-D05-08 | P | mỗi tin: ≤ 1 DECIDE (stack); Go TestCallBudgetPerMessage PASS |
| TC-D05-09 | P | Go TestRouteDecisionNoOverride PASS |
| TC-D05-10…TC-D05-19 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D09-01 | P | Go TestDecideCallersAllowlist PASS; ngoài internal/llm chỉ thread service gọi .Decide |
| TC-D09-02 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D09-03 | P | Go TestCrisisNeverReachesDecide PASS (chat + Threads); stack chat khủng hoảng: 0 DECIDE |
| TC-D09-04…TC-D09-05 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D09-06 | P | schema: decision_gates / course_decision_gates chỉ có cờ + ngưỡng; llm_audit không cột nội dung |
| TC-D09-07…TC-D09-08 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D09-09 | P | git diff gốc..10.5: go.mod / go.sum / package.json không đổi (chỉ docs/ARCHITECTURE.md) |
| TC-D09-10 | P | mỗi tin ≤ 1 DECIDE; Go TestCallBudgetPerMessage + TestOneEmbedPerMessage PASS |
| TC-D09-11 | P | log gateway / worker / caddy: 0 gói tới googleapis; ALLOW_REAL=false, GEMINI_API_KEY trống (không dùng lsof / tcpdump) |
| TC-D06-01 | P | /settings/llm 1440 + 375: đúng một khối «Quyết định nhanh» sau «Chuỗi dự phòng», trước «Mô hình tìm kiếm tài liệu», không tràn ngang |
| TC-D06-02 | P | hàng «Quyết định» · decision-fake · «Giả» · Test kết nối |
| TC-D06-03…TC-D06-06 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D06-07 | P | 3 hàng cổng đúng tên, công tắc mặc định Tắt, «Cài đặt nâng cao», «Theo lớp»; công tắc 36×36 (1440) / 44×44 (375) |
| TC-D06-08…TC-D06-09 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D06-10 | P | tìm lớp theo mã trong bảng Theo lớp |
| TC-D06-11…TC-D06-14 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D06-15 | P | dm_m: đổi + hoàn tác ngưỡng → 2 dòng audit_log decision_gate có before/after/actor |
| TC-D06-16 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D06-17 | P | teacher không thấy khối / bảng «Quyết định nhanh» ở /settings/llm và /observability |
| TC-D06-18…TC-D06-22 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D06-23 | P | dm_m: GV/SV/TA không thấy «Quyết định nhanh» / tên cổng trong 30 phản hồi |
| TC-D07-01 | P | số khớp SQL (calls, FAST %, FALLBACK %, FAIL_CLOSED %, error %): 3 cổng |
| TC-D07-02 | P | RERUN 2026-11-16 sau sửa DEF-R10.5-01 (1c0700e2): /observability (Admin, 1440): y(«7 ngày qua»)=405 < y(«Quyết định nhanh»)=903 — bảng 7 ngày ở trên (evidence-10.5/scripts/obs-order-r1.cjs, shots/observability-order-r1.png) |
| TC-D07-03 | P | dải «24 giờ qua» không có chỉ số decision |
| TC-D07-04 | P | dm_m: «Bật ở N/M lớp» = courses_enabled/total khớp SQL (1/51); cổng tắt → state off |
| TC-D07-05…TC-D07-08 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D07-09 | P | teacher: /admin/observability/decision 403; /observability không có bảng decision |
| TC-D07-10 | P | teacher / TA / SV không nhận chuông DECISION_FAIL_CLOSED |
| TC-D07-11…TC-D07-13 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D07-14 | P | GET observability/decision: Cache-Control no-store; window lạ → 422 |
| TC-D07-15 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D07-16 | P | dm_m: DECIDE THREADS chỉ thuộc 1 lớp có cổng (quan sát) |
| TC-D08-01 | P | seed sạch: llm_providers có decision-fake bật + gemini-decision-1 tắt; ba cổng tắt, ngưỡng mặc định |
| TC-D08-02…TC-D08-03 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D08-04 | P | Go TestDecide* + TestCascade* (llm, llm/fake, llm/provider, llm/scheduler) PASS |
| TC-D08-05…TC-D08-16 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D08-17 | P | go test -tags testroutes ./internal/contract/... PASS (môi trường sạch) |
| TC-D08-18…TC-D08-21 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |
| TC-D08-22 | P | scripts/ui-antipatterns.sh exit 0; không khoá AIza trong mã (chỉ trong chữ TC cũ) |
| TC-D08-23 | chưa chạy (mẫu) | chưa chạy (mẫu): dạng tài liệu / cần dữ liệu hoặc kịch bản chưa có; ngoài mẫu «1 chính + 1 lỗi/phân quyền mỗi US» |

## Trượt: bước tái hiện
### DEF-R10.5-01 — ĐÃ SỬA, rerun 2026-11-16 đạt (TC-D07-02): «Quyết định nhanh» nằm dưới bảng 7 ngày.

## Ca K và lý do (bảng gom)
| Mã loại | Ca | Lý do |
| --- | --- | --- |
| KL | TC-D04-01 | LỖI THỜI (P-11-26): bản phát lại decision-replay.json nay chứa mẫu chấm của seed nên FAKE_DECISION_PROB không ghi đè được (chỉ áp cho đầu vào ngoài replay) → không dựng được bảng OR 7 tổ hợp bằng P; với P=0,9 chỉ 1–6 lời gọi FAST, 20 FALLBACK |
| KL | TC-D04-09 | LỖI THỜI (P-11-26): 6 biến thể + 6 đối chứng nay đo bằng bộ replay riêng (P-11-26: 0 cờ nhầm trên bộ sạch); không đo được bằng FAKE_DECISION_PROB |

## Chưa chạy (mẫu)
103 ca: dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có script; liệt kê theo dải ở bảng trên (cột «chưa chạy (mẫu)»).

## Lượt 2026-11-18 — luật chủ dự án: không còn nhãn chặn/LT thiếu nguồn; chạy tới P/F (mã dc67bd93, stack ep-qc)
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-D04-01 | N | «LT (P-11-26)» cũ thiếu nguồn quyết định (P-11-26 không đổi hành vi TC) → chạy lại (dựng bảng OR bằng bộ replay/đầu vào ngoài replay) |
| TC-D04-09 | N | như TC-D04-01 |

## Lượt 2026-11-19 — chạy nốt 105 ca «N» (mã gateway 4ac1638a, worker 6abb09af, stack ep-qc; lớp riêng QD105A/QD105B; không đụng 761987/761988)
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-D01-07 | N | chưa xong: lần 1 (8 luồng, 200 đăng bài) p50=101/p95=150 ms trên 127 dòng ok (đúng ±20 %) nhưng 73/200 bị 'overloaded' (vượt RPM 150 của provider decision) nên không đủ 200 mẫu; lỗi 0,5: 46 error/102 lời gọi thực = 45,1 % (đúng ±10 điểm) nhưng cầu dao mở làm 84 dòng circuit_open. Lượt lặp có nhịp 1,6/s + mặc định 426/682 bị dừng khi ep-qc dựng lại — chạy lại sau |
| TC-D01-11 | P | llm_audit task=DECIDE theo cổng (ep-qc, lớp QD105A): THREADS_PRECHECK → INTERACTIVE, CHAT_ROUTING → INTERACTIVE, INJECTION_FLAG → BATCH (đủ 3 cổng, SELECT cascade_gate,lane); đúng làn từng cổng |
| TC-D01-18 | P | Go (test tạm của QC, evidence-10.5/s105/go/qc105_test.go) 2.000 lần Decide khi chưa có provider/khoá: p50 0,67 µs · p99 32 µs · max 0,68 ms < 1 ms, không treo; llm_audit đúng 2.000 dòng / 2.000 lần gọi (status not_configured, không lặp); thêm TestDecideNotConfigured của dev PASS |
| TC-D01-21 | P | Go (test tạm QC): model gemini-decision-1 (máy chủ giả trong tiến trình), cap 3, 5 lời gọi → 3 đầu qua (provider gọi 3), lời 4–5 budget_blocked; Cascade lời 6 cùng ngày → path FALLBACK, fallback chạy 1 lần; 23:59:59 UTC+7 vẫn chặn, 00:00 UTC+7 (17:00 UTC) đếm lại: 3 lời qua, lời 4 chặn. Không gọi mạng thật |
| TC-D02-05 | P | Go: TestCascadeTable ca «AddOnly safety → fallback (không phải 'có')» (provider 400 safety → KindBlocked) → path FALLBACK, verdict không-chặn-do-SAFETY, đúng 1 dòng DECIDE; provider TestDecideErrorMap «400 safety» → SAFETY_BLOCK không retry, không tính cầu dao: PASS |
| TC-D02-07 | P | hệ thống bật 3 cổng, lớp QD105A tắt: đăng bài (201, None), chat 200, chấm lại DONE; DECIDE của lớp = 0 |
| TC-D02-08 | P | lớp bật 3 cổng, hệ thống tắt: đăng bài (201, None), chat 200, chấm lại DONE; DECIDE = 0 |
| TC-D02-09 | P | lớp 1 (cả hai bật): DECIDE 3 dòng, cổng ['CHAT_ROUTING', 'INJECTION_FLAG', 'THREADS_PRECHECK'], [('FAST', 'ok'), ('FALLBACK', 'ok'), ('FAST', 'ok')]; lớp 2 (chỉ hệ thống): đăng (201, None), chat 200 → DECIDE 0 |
| TC-D03-11 | P | đăng ok (decision «không») rồi bật «có»: PUT sửa bài → 422 PII_DETECTED DECIDE 1; bình luận → 422 PII_DETECTED DECIDE 1; precheck → 200 DECIDE 0 (precheck không qua decision) |
| TC-D03-17 | P | cổng chỉ lớp QD105A, decision giả luôn «có»: sv.kha lớp 1 → 422 PII_DETECTED, DECIDE [['THREADS_PRECHECK', 'FAST', 'ok']]; sv38 lớp 2 cùng nội dung → 201, DECIDE 0 (đường cũ) |
| TC-D03-18 | P | teacher@, ta@ lớp 1 đăng cùng bài (decision giả «có»): {'teacher': (422, 'PII_DETECTED', [['THREADS_PRECHECK', 'FAST', 'ok']]), 'ta': (422, 'PII_DETECTED', [['THREADS_PRECHECK', 'FAST', 'ok']])} → đi qua cổng như SV |
| TC-D03-20 | P | 10 đăng song song (10 SV khác nhau), decision giả p50 1.500 ms > hạn cổng 800 ms: 10 request đều trả có kết quả (201×5, 422×5; không 5xx, không treo; mỗi request 4,5–6,9 s ≈ hạn riêng 0,8 s decision + tối đa 6 s Structured); DECIDE đúng 10 dòng (1/request): timeout 9 + overloaded 1 → FALLBACK 4 / FAIL_CLOSED 6 (Structured 9router quá 6 s khi 10 request cùng lúc → chặn an toàn, 422 + hộp thoại); số thread tạo = số 201 (5), tiêu đề duy nhất 5 → không đăng trùng, không deadlock |
| TC-D04-04 | P | go test ./internal/grading -run 'TestInjectionOR/TestInjectionScoreUnchanged' -v: TestInjectionOR 7/7 ca PASS (OR ba tầng, cờ không đổi điểm — assert Total == Expected.Total, tầng vắng khi lỗi, bài < 20 từ bỏ qua); TestInjectionScoreUnchanged không còn trong mã (BA bỏ tên ở commit 13437795) — phủ bởi TestInjectionOR + TestInjectionDoesNotRaiseScore PASS |
| TC-D04-11 | N | chờ phép PM (cố ý không chạy; luật chạy tc-00-chung: lượt thật ≤ 300 lời gọi cần PM cho phép, ALLOW_REAL=false giữ nguyên) — không tính FAIL |
| TC-D04-18 | P | grep Decide(/Cascade( không-test: ở internal/grading chỉ injection_decision.go:27 (g.Decide.Cascade trong flagInjectionDecision); không từ tính điểm/rubric/tổng/làm tròn. TestDecideCallersAllowlist PASS. Bản sao tạm trong internal/grading (hàm qcRoundScore gọi .Cascade) → test ĐỎ «gọi Decide / Cascade ngoài danh sách trắng: [grading.qcRoundScore]»; xoá bản sao → PASS lại |
| TC-D05-17 | P | cổng CHAT_ROUTING tắt ở lớp (hệ thống bật, 2 cổng kia bật): chat mơ hồ → HTTP 200, DECIDE = 0 |
| TC-D05-18 | P | CHAT_ROUTING bật chỉ ở QD105A: sv38 (lớp 2) hỏi cùng tin mơ hồ → HTTP 200, DECIDE lớp 2 = 0, lớp 1 trong khoảng đó = 0 |
| TC-D05-19 | P | 10 tin gửi cùng lúc (5 mơ hồ, 5 cụ thể — mỗi tin một SV/phiên): HTTP [200]; DECIDE theo người: mơ hồ [1, 1, 1, 1, 1], cụ thể [0, 0, 0, 0, 0] (đúng 1 / 0): không lẫn nhãn giữa phiên |
| TC-D06-20 | P | docs/specs/FEAT-llm-gateway/SRS.md bảng 7.3 dòng 534: `DECIDE` (D60, chỉ ADMIN thấy) · «Quyết định nhanh» · «Trả lời ngay (chat, Threads) / Chạy nền (cờ chèn lệnh)»; frontend/src/features/settings/llm/labels.ts: DECIDE → «Quyết định nhanh» |
| TC-D07-15 | P | go test ./internal/... -run 'TestDecisionRBAC/TestDecisionGateVersionConflict/TestDecisionAudit': TestDecisionAudited (thread) PASS; hai tên đầu không còn trong mã (BA 13437795 bỏ 7 tên không tồn tại) → chạy các test thật cùng nội dung: TestDecisionAdminOnly (RBAC), TestDecisionOverviewAndGateUpdate (409 phiên bản, khoảng, audit_log), TestDecisionCourseGates, TestDecisionObservability, TestDecisionFailClosedAlert PASS |
| TC-D08-06 | F | DEF-R105S-01: frontend/e2e/cascade.spec.ts (58 dòng) chỉ có 2 ca giao diện (hộp thoại hai lối, chỉ báo «Đang kiểm tra»), API giả hoàn toàn (threadApi), không có nhánh (a)–(f) và không assert cascade_path ở llm_audit; US-DM-08 AC3 yêu cầu E2E đủ 6 nhánh. Các nhánh nằm ở Go (TestCascadeThreadsHTTP a,b,c,f; TestInjectionOR d; TestRouteDecisionTable e) — chính chú thích đầu spec và handoff L5 thừa nhận |
| TC-D08-08 | P | benchmarks/decision/run.sh→run.py: chỉ import thư viện chuẩn (hashlib json os re ssl statistics sys time urllib uuid pathlib, subprocess); grep -r 'import.*backend' benchmarks/decision = 0; chỉ HTTP; 3 kịch bản: chat.jsonl 30 câu × CHAT_ROUNDS 3, threads.jsonl 40 bài (20 sạch/10 private/10 rules), chấm 12 bài = 5 biến thể v1–v5 + s6 (6 chèn lệnh) + s1–s5,s7 (6 đối chứng; đúng SRS sau P-105-1 Q6, không còn «13 / 7»); mỗi kịch bản chạy off rồi on cùng lớp/seed (set_mode) |
| TC-D08-12 | P | git log -- benchmarks/decision/run.py (4 commit: 1fb80cfa, 8719db26, 5a47d77f, 2177e056): ba ngưỡng giữ nguyên từ bản đầu 1fb80cfa — chat ratio ≤ 1.2 (+20 %) & độ chính xác ≥; Threads false_block_rate ≤ 0.05 & missed ≤; chèn lệnh flag_correct == flag_expected (6/6) & flag_false == 0; khớp SRS/US-DM-08 AC5; không commit nào nới |
| TC-D08-14 | N | Hỏi QC cha/BA: run.py KHÔNG có đường chạy «thật» nào: từ chối cứng khi provider decision không phải bản giả («lượt thật cần phép PM… Dừng»), không có cột «thật», không có trần 300 / cờ xác nhận rõ. TC kỳ vọng «có chặn: cột thật chỉ chạy khi ALLOW_REAL=true và có xác nhận rõ; trần 300»; US-DM-08 AC6 chỉ nói lượt thật «khi PM cho phép» + trần 300. Chặn hiện đủ an toàn (mặc định bỏ qua) nhưng trần 300/xác nhận chưa có vì chưa có lượt thật → chờ BA/PM xác nhận có cần mã chặn sẵn trước khi có phép |
| TC-D08-19 | P | go test ./db -run TestMigrations: RoundTrip PASS (goose up trên DB trống → up lần hai no-op → down về 0 → up lại, lược đồ giống hệt) + UpAppliesMissingOlder PASS. git diff 8a1d4d02..8f5615b8 -- migrations: chỉ THÊM 00020_decision.sql (+00021 của nhánh khác), không sửa tệp cũ; 00020 liền sau 00019 (cuối đã merge lúc tách nhánh) |
| TC-D09-02 | P | Bản sao tạm (ovnfs ~/checkouts/qc105go) gọi .Decide( trong internal/grade (qcMutation) → TestDecideCallersAllowlist ĐỎ «grade.qcMutation (internal/grade/qc105_mut.go)»; xoá bản sao → PASS lại; không commit |
| TC-D09-07 | P | docs/PRODUCTION_READINESS.md §1 dòng 23: «Triển khai thật bắt buộc dùng gói trả phí cho khoá Gemini»; .env.example dòng 54: LLM_DECISION_ALLOW_REAL=false (chú thích dòng 51); docs/guides/decision-model.md đoạn đầu: «Trước khi dùng dữ liệu sinh viên thật… Free Tier… triển khai thật bắt buộc…» |

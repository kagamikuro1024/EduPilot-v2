# Smoke hồi quy sprint 10.5 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm G, SmkG) · Ngày: 2026-11-15
Tổng TC cũ: 185 · Đạt: 79 · Trượt: 1 · Không chạy (K, gồm lỗi thời): 105   (P+F+K = 185)
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
| TC-D07-02 | **F** | bảng «Quyết định nhanh» nằm TRÊN bảng «7 ngày qua» (AdminView.tsx:58 trước :65); US-DM-07 AC1: dưới bảng 7 ngày · **DEF-R10.5-01** |
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
### DEF-R10.5-01 — bảng «Quyết định nhanh» ở /observability nằm trên bảng 7 ngày (TC-D07-02; Thấp, hồi quy do sprint 11). Chi tiết `defects-10.5.md`.

## Ca K và lý do (bảng gom)
| Mã loại | Ca | Lý do |
| --- | --- | --- |
| KL | TC-D04-01 | LỖI THỜI (P-11-26): bản phát lại decision-replay.json nay chứa mẫu chấm của seed nên FAKE_DECISION_PROB không ghi đè được (chỉ áp cho đầu vào ngoài replay) → không dựng được bảng OR 7 tổ hợp bằng P; với P=0,9 chỉ 1–6 lời gọi FAST, 20 FALLBACK |
| KL | TC-D04-09 | LỖI THỜI (P-11-26): 6 biến thể + 6 đối chứng nay đo bằng bộ replay riêng (P-11-26: 0 cờ nhầm trên bộ sạch); không đo được bằng FAKE_DECISION_PROB |

## Chưa chạy (mẫu)
103 ca: dạng tài liệu / cần dữ liệu lớn hoặc kịch bản chưa có script; liệt kê theo dải ở bảng trên (cột «chưa chạy (mẫu)»).

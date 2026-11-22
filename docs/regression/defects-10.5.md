### DEF-R10.5-01 — Bảng «Quyết định nhanh» ở /observability nằm trên bảng «7 ngày qua»
- Mức: Thấp
- Sprint gốc: 10.5 · TC gốc: TC-D07-02 (US-DM-07 AC1: «một bảng … dưới bảng 7 ngày») · Hồi quy do sprint sau?: CÓ — sprint 11 viết lại màn «Quan sát AI» (`AdminView.tsx`, các commit 7993ff72 merge wip/11-wave1, 78b0a2be «Section/Page/EmptyState»); ở 10.5 (`ObservabilityScreen.tsx:159`) bảng nằm cuối trang
- Môi trường: ep-r, be0c0d96, ADMIN, 1440 và 375
- Bước tái hiện: 1. Đăng nhập `admin@edupilot.local`. 2. Mở `/observability`. 3. Thứ tự các khối: «Thư chưa gửi được» → «Quyết định nhanh» → «7 ngày qua» → «Chi phí theo tác vụ» → «Yêu cầu». (`frontend/e2e/qc-dm-ui-real.spec.ts`, ảnh `evidence-10.5/shots/observability-admin-1440.png`)
- Kỳ vọng (theo AC): bảng «Quyết định nhanh» dưới bảng 7 ngày  · Thực tế: trên bảng 7 ngày (`frontend/src/features/observability/AdminView.tsx:58` trước `:65`)
- Bằng chứng: `evidence-10.5/pw-dm.log`, `evidence-10.5/shots/observability-admin-1440.png`  · Nghi ngờ mã: `AdminView.tsx:58-65`
- Chạy lại lần 2: ổn định

### Mục TC-D04 (cổng chèn lệnh) — không có lỗi sản phẩm
- TC-D04-01, D04-09 ghi `LỖI THỜI (P-11-26)`: `FAKE_DECISION_PROB` chỉ ghi đè đầu vào NGOÀI `decision-replay.json`; sau P-11-26 replay chứa mẫu chấm của seed (20 FALLBACK + 1 FAST ở P=0,9) nên kịch bản `dm_c.py` (14 cờ mới ở P=0,9) không còn dựng được. Cần dev/PM cho cách điều khiển mới nếu muốn kiểm OR ba tầng.
- TC-D04-02/03 PASS: lỗi decision (circuit_open) → chấm xong 22/22, điểm y hệt giữa cổng tắt / bật / lỗi cùng điều kiện. Chênh điểm so với round 1 trong `dm_c.py` KHÔNG do cổng (cổng tắt cũng chênh y hệt): round 1 chấm lúc route GRADING do nhóm khác đang đổi.
- Lưu ý môi trường: Postgres vào recovery giữa lượt B5a (nhóm khác chaos) làm hỏng một lượt; đã chạy lại ở B6.

## Lượt 2026-11-19 — chạy nốt 105 ca «N» (gateway 4ac1638a, worker 6abb09af, stack ep-qc)

### DEF-R105S-01 — `cascade.spec.ts` không phủ các nhánh (a)–(f) của US-DM-08 AC3
- Mức: Trung bình
- TC: TC-D08-06 (US-DM-08 AC3: «E2E `cascade.spec.ts`: (a) lưỡng lự → Structured; (b) decision lỗi → Structured; (c) cả hai lỗi → chặn; (d) chèn lệnh decision lỗi → cờ regex + Structured, điểm không đổi; (e) chat dưới τ / lỗi → như tắt; (f) khủng hoảng → 0 lời gọi decision; đúng `cascade_path` ở `llm_audit` mỗi nhánh»)
- Môi trường: tĩnh (repo `4ac1638a`), không cần stack
- Bước tái hiện: 1. Mở `frontend/e2e/cascade.spec.ts` (58 dòng). 2. Liệt kê `test(...)`: chỉ 2 ca — «bài bị chặn bởi cổng decision: cùng hộp thoại hai lối…» và «kiểm duyệt chậm > 800 ms: hiện «Đang kiểm tra»». 3. Cả hai dùng `threadApi`/`json(...)` (API giả hoàn toàn), không có nhánh (a)–(f), không có assert nào về `cascade_path` / `llm_audit`.
- Kỳ vọng (theo AC): spec đủ 6 nhánh + assert `cascade_path` từng nhánh · Thực tế: 2 ca giao diện; 6 nhánh nằm ở Go (`TestCascadeThreadsHTTP` a,b,c,f; `TestInjectionOR` d; `TestRouteDecisionTable` e) — chú thích đầu spec và handoff L5 («Chưa làm (có chủ đích): cascade.spec.ts bản chạy trên stack thật») thừa nhận
- Bằng chứng: `evidence-10.5/s105/logs/cascade-spec-lines.txt`; tự chạy lại 6 nhánh bằng API ở TC-D08-07 (kết quả ở smoke)
- Lưu ý: nhánh vẫn được kiểm ở mức Go + API của QC; thiếu là bản E2E theo đúng chữ của AC3 → BA/PM quyết «sửa AC cho khớp (Go + contract)» hoặc dev bổ sung spec

## Lượt 2026-11-21 — S10X (tĩnh, HEAD c12be038)

### DEF-R105S-02 — `benchmarks/decision/run.py` không có đường «lượt thật» (cột «thật», cờ xác nhận, trần 300)
- Mức: Thấp
- TC: TC-D08-14 (US-DM-08 AC6: «PM cho phép bằng văn bản và `LLM_DECISION_ALLOW_REAL=true` → tối đa 300 lời gọi; báo cáo thêm cột «thật»»)
- Môi trường: tĩnh (repo `c12be038`), không cần stack
- Bước tái hiện: 1. Đọc `benchmarks/decision/run.sh` (chỉ `exec python3 run.py`). 2. `grep -n -i "real\|thật\|ALLOW\|300\|xác nhận" benchmarks/decision/run.py` → dòng 8 (docstring), 356 («Cột «thật» (AC6): chờ phép PM.» ghi cứng vào báo cáo), 374 (`die("Provider decision không phải bản giả: lượt thật cần phép PM (AC6). Dừng.")`). 3. Không có biến `LLM_DECISION_ALLOW_REAL`, cờ xác nhận, bộ đếm / trần 300, cột «thật» (nhà cung cấp, model, ngày, độ trễ, chi phí).
- Kỳ vọng (theo TC/AC): cột «thật» chỉ chạy khi ALLOW_REAL=true **và** có xác nhận rõ; mặc định bỏ qua; trần 300 · Thực tế: từ chối cứng khi provider không phải bản giả; không có mã cho lượt thật nên không có chặn «có xác nhận» / trần 300 để kiểm (mặc định an toàn nhưng AC chưa có mã)
- Bằng chứng: `evidence-10.5/s10x/d0814.txt`
- Lưu ý: kéo theo TC-D04-11 (lượt thật ≤ 300 lời gọi) chưa chạy được ở ep-r: không có `GEMINI_API_KEY` (.env.local và gateway ep-r đều rỗng) và không có công cụ đo lượt thật → ghi N.

## DEF-R105S-03 — Provider «Quyết định» tắt nhưng cổng chat / chèn lệnh vẫn gọi decision
- Mức: Trung bình · TC: TC-D02-10, TC-D01-25 · Mã: 37084f11 (ep-qc, CLASSIFY/CHAT qua provider fake)
- Bước tái hiện: 1. Bật CHAT_ROUTING (hệ thống + lớp) và INJECTION_FLAG (hệ thống + lớp). 2. `PUT /admin/llm/providers/{id}` provider «Quyết định» enabled=false (GET /admin/llm/decision → provider=null). 3. Chat một câu; chấm một bài; đăng bài. 4. `select … from llm_audit where task='DECIDE'`.
- Kỳ vọng (SRS 4.4, TC): hiệu lực = cờ hệ thống ∧ cờ lớp ∧ provider sẵn sàng → provider tắt: 0 lời gọi DECIDE, mọi cổng như trước 10.5.
- Thực tế: THREADS_PRECHECK đúng (0 DECIDE); CHAT_ROUTING vẫn 1 dòng DECIDE (ok, FALLBACK) ở chat sau +3 / +23 / +63 s; INJECTION_FLAG 1 dòng DECIDE ở lượt chấm.
- Bằng chứng: `evidence-9s/s106/g105j.py`, `g105i.py`.
- Trạng thái: ĐÓNG — P ở f26b15db (dev3 sửa Registry.build; g105i: 0 DECIDE ở cả 3 cổng)

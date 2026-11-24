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

## DEF-R105S-04 — Thân gửi decision của cổng INJECTION_FLAG dài 16.001 ký tự (> 16.000)
- Mức: Thấp · TC: TC-D04-07, TC-D09-05 · Mã: cbbea55a (ep-qc, máy chủ giả HTTP :9770)
- Bước tái hiện: 1. ALLOW_REAL=true, `LLM_DECISION_BASE_URL` trỏ máy chủ giả, provider model gemini-decision-1, bật INJECTION_FLAG lớp QD9G. 2. Chấm bài nộp 40.000 ký tự (chèn lệnh ở vị trí 20.000; bài khác chèn ở đầu). 3. Đọc thân POST `/decisions` ở máy chủ giả.
- Kỳ vọng (US-DM-04 AC4, TC): tối đa 16.000 ký tự = 8.000 đầu + 8.000 cuối.
- Thực tế: `input` dài 16.001 ký tự (8.000 đầu + «\n» + 8.000 cuối; `llm/decide.go` `clip`: `r[:limit/2] + "\n" + r[len(r)-limit/2:]`). Đầu/cuối đúng, phần giữa (MIDMARK) không có trong thân, log, `llm_request_payloads` của DECIDE. Regex vẫn gắn cờ cả hai bài.
- Bằng chứng: `evidence-9s/s106/g105h2_2b.py`.
- Trạng thái: ĐÓNG — P ở 1fdcda73 (thân đúng 16.000: 8.000 đầu + «\n» + 7.999 cuối)

## DEF-R105S-05 — RÚT LẠI (lỗi môi trường của QC, không phải lỗi mã)
- Nguyên nhân: shell của QC còn `COMPOSE_PROJECT_NAME=ep-qc`; `scripts/seed.mjs` / `check-decision-seed.mjs` chạy lệnh admin bước 16 (seed-study-time, seed-age --activity, seed-activity, crm-risk-run) bằng `docker compose exec gateway` của THƯ MỤC HIỆN HÀNH (SEED_ADMIN_CMD mặc định) → ghi vào DB ep-qc, trong khi API gọi qc2seedt / qc2seedt2. `crm-risk-run` tính trên dữ liệu ep-qc nên ra «AT_RISK 1, WATCH 1».
- Hệ quả: TC-D08-03 chuyển F → N (lượt chạy vô hiệu); TC-D08-02 vẫn N. Chạy lại ở qc2seedt3 sau khi có `SEED_ADMIN_CMD` tường minh.
- Ghi chú ep-qc: lệnh admin đã ghi vào lớp 761987 / 761988 của ep-qc (số dòng đã báo PM). question_bank 761987 vẫn 150 PENDING / 107 APPROVED (E5 nguyên).
- Kiểm chốt chặn của dev3 (49ebd12a, chạy 1 lần theo PM): `seed.mjs` với API_URL=qc2seedt3 và SEED_ADMIN_CMD cố ý trỏ project qc2seedt2 → thoát 2 sau bước 3/20, in «Seed dừng: Lệnh admin ghi vào DB KHÁC với API_URL … lớp 761987 qua API = 01a20315…, qua lệnh admin = 01a202be…»; trước/sau: qc2seedt2 student_risk max(computed_at) 10:20:24.834Z, 54 dòng, outbox 1.297 dòng — không đổi; qc2seedt3 student_risk không đổi. Chốt chặn hoạt động.
- Trạng thái: ĐÓNG (không phải lỗi sản phẩm)

## DEF-R105S-06 — /settings/llm: 8 chữ đỏ cùng lúc, liên kết «Cài đặt nâng cao» đỏ ở bảng tác vụ nhưng xám ở khối «Quyết định nhanh» (DESIGN §22 đạt 6/10)
- Mức: Thấp · TC: TC-D06-22 · Mã: 1fdcda73 (ep-qc, admin, 1440 và 375 px)
- Bước tái hiện: `/settings/llm` (ADMIN); đọc màu chữ các nút/liên kết trong `main`; Tab qua 40 điểm dừng; đo tràn ngang.
- Kỳ vọng (TC; DESIGN §14.23, §22): ≥ 8/10 điều kiện §22; một đường hành động rõ trong khung nhìn đầu (§22.2); phức tạp phụ không cạnh tranh (§22.3); đỏ chỉ có nghĩa «đang ở đâu / cần hành động / đã sửa» (§22.10).
- Thực tế: đạt §22.1, 4, 5 (40/40 điểm dừng có viền focus), 6 (không tràn ngang ở 1440 và 375), 8 (ui-antipatterns sạch), 9 = 6/10. Chưa đạt: §22.2 và §22.3 — khung nhìn đầu 1440×900 có «Thêm nhà cung cấp» + 5 liên kết «Cài đặt nâng cao» cùng màu đỏ (tổng 8 chữ đỏ: 1 «Thêm nhà cung cấp», 6 «Cài đặt nâng cao», 1 «Đổi hạn mức»); §22.10 — «Cài đặt nâng cao» là hành động phụ nhưng đỏ ở bảng «Mô hình theo tác vụ», còn ở khối «Quyết định nhanh» cùng liên kết màu xám (không nhất quán). §22.7 (chữ Việt dài) chưa kiểm được: tên nhà cung cấp tối đa 60 ký tự và base_url phải hợp lệ khi tạo.
- Bằng chứng: `evidence-9s/s106/specs/qc-real-105-ui.spec.ts` (D06-22), ảnh mốc `docs/sprints/10.5/qc/shots/settings-llm-admin-1440.png`.
- Trạng thái: MỞ
| DEF-R105S-07 | Thấp | TC-D08-13 | `benchmarks/decision/run.py` (dòng ~350 và mục «Giới hạn») luôn in «provider decision GIẢ (p50 426 / p95 682 ms)» và «Độ trễ decision là mẫu log-chuẩn của provider giả» ngay cả khi chạy `--real` (meta.config.decision_provider = real); mục «Lượt thật» đứng riêng bên dưới nên người đọc thấy hai kết luận mâu thuẫn | MỞ — dev3 (đổi chữ đầu báo cáo + «Giới hạn» theo `real`) |

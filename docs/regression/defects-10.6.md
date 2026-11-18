### DEF-R10.6-01 — reflect / regrade-stale qua lớp có cờ tắt trả 409 GUIDANCE_DISABLED trước 404
- Mức: Thấp
- Sprint gốc: 10.6 · TC gốc: TC-G03-24, TC-G05-24 · Hồi quy do sprint sau?: KHÔNG (có sẵn; summary 10.6 «Ghi nhận (1)» đã nêu, PM chưa đổi)
- Môi trường: ep-r, be0c0d96, vai teacher (GV lớp 761987 và lớp 761988), 1440
- Bước tái hiện: 1. `teacher` đăng nhập; lấy `C2` = id lớp 761988, `A` = id «Bài tập 03» của lớp 761987 (không thuộc lớp 761988). 2. Cờ `GRADING_GUIDANCE` của lớp 761988 đang tắt. 3. `POST /courses/$C2/assignments/$A/guidance/reflect` (Idempotency-Key bất kỳ) và `POST /courses/$C2/assignments/$A/regrade-stale`. (Script: `docs/regression/evidence-10.6/scripts/gd_k.py`, `gd_d.py`)
- Kỳ vọng (theo TC): `404` (bài tập không thuộc lớp)  · Thực tế: `409 GUIDANCE_DISABLED` (kiểm cờ lớp chạy trước kiểm sở hữu bài tập); bật cờ lớp 761988 thì ra 404 (suy luận, chưa chạy)
- Bằng chứng: `docs/regression/evidence-10.5/batchB1.log` (G03-24), `docs/regression/evidence-10.6/gd_d.log` (G05-24)  · Nghi ngờ mã: `internal/httpapi/assignmenthttp/guidance.go` (thứ tự guard cờ → tải bài tập)
- Chạy lại lần 2: ổn định (2/2)

### Ghi nhận không phải lỗi sản phẩm (đã loại)
- TC-G03-22 (reflect `error` / `invalid` → FAILED sau 3 lần): hành vi đổi theo P-11-23 (`llm.AwaitOutage`: mất nhà cung cấp không tiêu lượt, hoãn tới OutageMax) → job FAILED sau ≈ 10 phút, vẫn 0 DRAFT, mã `REFLECT_FAILED`. Ghi `LỖI THỜI (P-11-23)`.
- TC-G01-05 «lớp lạ → 404»: GV/TA/SV không thuộc lớp nhận 403 từ CourseAccessGuard (không đổi từ 10-2026; tc-00 sprint 10: «403 hoặc 404»); ADMIN nhận 404.
- `go test -tags testroutes ./internal/contract` đỏ ở `TestNoAnswerLeak` khi chạy trong shell đã export `QC_*`/`SEED_DEFAULT_PASSWORD`; chạy trong môi trường sạch: PASS (cả gói 57 s). Do môi trường chạy của QC, không phải lỗi.

## Đã xoá (lỗi thời có nguồn quyết định; lệnh chủ dự án 2026-11-18)
- TC-G03-22: xoá khỏi smoke-10.6.md và khỏi tổng — nguồn P-11-23 (PM 2026-11-15 chấp nhận: lỗi LLM_UNAVAILABLE/circuit_open xếp lại không tiêu lượt thử, nên hành vi «FAILED sau 3 lần ≈30 s» đã đổi).

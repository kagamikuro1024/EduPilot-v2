# QC sprint 10.6 (D61 / D62, «học từ lần giảng viên sửa») — kết luận

Nhánh `sprint/10.6-guidance`, HEAD kiểm `b9a3bdd5` (L1–L6 + sửa GD-BUG-4/5/6/8 + P-106-7/8). `ep-qc` **dựng lại từ DB trống** (`down -v`, migration `00021` bản đã sửa, seed `SEED_GUIDANCE_DEMO=1` mã 0, 508 s). Sinh chữ chỉ qua stub QC (`scripts/stub_llm.py`) trỏ route `GRADING`; 0 gói ra `googleapis` (gateway / worker), `llm_audit` chỉ có `env:fake` và `QC stub`. Chi tiết: `report-L1-L3.md`, `report-L4-backend.md`, `report-stack-L1-L4.md`; script `scripts/gd_a…gd_g.py`; UI `frontend/e2e/qc-gd-*-real.spec.ts`; ảnh `shots/`.

| Story | Kết luận | Căn cứ chính |
| --- | --- | --- |
| US-GD-01 Cờ theo lớp | **PASS** | 4 cờ, RBAC, `409` / `422`, audit; `GUIDANCE_ENABLED=false` qua compose → `409 GUIDANCE_DISABLED` (GD-BUG-8 đã sửa); lớp lạ → `404` (GD-BUG-4 đã sửa); `/class/settings` đủ 4 nhãn đúng AC5 + trạng thái Bật / Tắt (GD-BUG-6 đã sửa); SV không lộ |
| US-GD-02 Nền dữ liệu, ghim, bất biến | **PASS** | migration (trống → 21, down → up, backfill, CHECK / UNIQUE / FK, trigger trước / sau `DONE`); prompt v0 **giống từng byte** bản trước 10.6; tập lưu ý theo phiên bản v0…v3; ghim = bản hiện hành ở 8 lần chấm lại; `ai_comments` bất biến. Chưa kiểm: ép lỗi giữa `apply` (G02-14) |
| US-GD-03 Rút lưu ý | **PASS** | `reflect` → stub: DRAFT, `support` do code đếm; bỏ từ khoá / PII / trùng / `case_id` lạ (cả mục) / 0 mục; `error` / `invalid` → `FAILED`; `429` lần 4; cùng khoá → cùng job |
| US-GD-04 Duyệt / Áp dụng / Hoàn tác | **PASS** | Duyệt / Bỏ / sửa chữ / `422`; Áp dụng (v1, `409` ×2, Idempotency-Key, đua `[200, 409]`); Cho nghỉ; **Hoàn tác = `decide RESET`** (TC-G04-19…30): về `DRAFT`, giữ chữ đã sửa + `text_original`, `409 ITEM_NOT_RESETTABLE` khi đã áp dụng, `409 VERSION_CONFLICT` hai tab, đua RESET × Áp dụng nhất quán, offline giữ chữ + «Thử lại», `Áp dụng` qua xác nhận «Áp dụng bản vN?», chữ «Chờ duyệt» thay «Nháp» (GD-BUG-5 đã sửa), 375 px nút ≥ 44 px |
| US-GD-05 Ghim, chấm lại, công bố miễn | **PASS** | `stale` 8 / 0 / 8 khớp SQL; chấm lại 8 bài round + 1 ghim v3, bài đã duyệt / round cũ nguyên văn; khối lưu ý đúng chỗ trong 14 / 14 prompt; công bố bài đã duyệt bản cũ → `exempt{EDITED:6}` không `409`; bài nháp `422 NOT_REVIEWED`; cờ tắt `exempt` 0. Chưa kiểm: 250 nháp, round 20, kill worker, trừ muộn thật (Go `TestRegradeKeepsLatePolicy` PASS) |
| US-GD-06 Số đo | **PASS** | `stats` v0 (`n=8`, MAE tiêu chí 0,25, MAE tổng 1,00, lệch +1,00) **khớp SQL độc lập**; dòng số đo trong ngăn bài tập; màn duyệt «bản v3 · k lưu ý» + chip «Bản cũ». Chưa kiểm: p95 1.000 kết quả |
| US-GD-07 Ca hồi quy (M2) | **PASS** (trừ phần hoãn) | cờ tắt 0 hàng; lệch đúng 1,0 ghi 1 hàng `GRADING_EDIT` (0,5 không ghi); `expected` chỉ số / enum + `rubric_hash`; sửa lại vẫn 1 hàng; `regress export` → thoát 3 từ chối. **TC-G07-13 / AC6 (kiểm lại ở `975f9541`)**: `GET …/guidance/stats` có `regression_cases` = số `GRADING_EDIT`+`GRADING_APPEAL` của bài tập, khớp SQL độc lập (2 = 2), TEACHER / TA `200`, SV / ADMIN `403`. Hoãn: `regress run` / baseline (cần LLM), `export` thật (chờ phép PM) |
| US-GD-08 Câu tương tự (M4) | **PASS** | `POST similar-confirmed`: thân có chữ, không lộ ở URL / log Caddy + gateway, `no-store`, GET cũ `404`, ≤ 3 + DTO `{thread_id,title}` (đúng chữ đã nhúng), `q` ngắn / rỗng / 6.000 ký tự không 500, lớp khác `403`, `429` từ lần 31 (người khác không ảnh hưởng), cờ tắt `[]` + 0 nhúng, p95 5 ms, 10 nhúng / 20 lần (cache), 0 lời gọi sinh chữ, 0 `pii_events`. Chưa kiểm: ngưỡng cosine 0,79 / 0,81, thread chưa xác nhận / ẩn / `TICKET` (cần fixture), lỗi nhúng |
| US-GD-09 Lý do loại (M6) | **PASS** | 7 lý do lưu đúng; lý do lạ `422`; `reject_reason` kèm duyệt `422`; CHECK DB; bulk có lý do lưu cả lô, lý do lạ → từng id `ok:false`; SV `403`, DTO SV không có. **GD-BUG-10 đã sửa** (kiểm lại ở `975f9541`): `REJECT` đơn và bulk không lý do → `UNSPECIFIED`; có lý do (`MO_HO`) giữ nguyên; audit `question.review` chỉ mang `reject_reason` là enum (allow-list của `TestReviewAudit` thêm đúng khoá này; ràng buộc `NotContains(secret)` và kiểm khoá còn nguyên, giá trị bị chặn bởi enum + CHECK nên **không** nới phép kiểm PII). Chưa kiểm trên stack: khối «Tránh…» trong prompt sinh (Go `TestRejectReasonHints`, `TestRejectDefaultsUnspecified` PASS) |
| US-GD-10 Seed, đo, hướng dẫn | **PASS** (AC5 / GD-BUG-11 chuyển lượt kiểm chứng theo P-106-10) | Seed demo mã 0; «Hôm nay» đúng (teacher: chờ duyệt + bản cũ; TA: bản cũ; SV / ADMIN không), 375 px không tràn. **GD-BUG-9 đã sửa**: `seed-guidance` `PUBLISH` cờ, sau khi tắt cờ qua API rồi chạy seed-guidance, `overview.enabled=true` ngay (0,01 s, trước đây ≤ 60 s); Go `TestSeedGuidancePublishesFlag` PASS. **GD-BUG-12 đã sửa** (kiểm trên bản sao DB của `ep-qc`, đã xoá lưu ý): chạy `seed-guidance` nạp đúng 2 nháp `c1` (`STRICTER` support 8, `CLARIFY` support 4, không PII), chạy lại vẫn 2, DB chính không đổi. Chưa chạy `check-guidance-seed` trên DB trống mới (DB `ep-qc` đã bẩn: v4, cờ khác) — làm khi dựng lại DB |
| US-GD-11 Ranh giới an toàn | **PASS** | `guidance_version` chỉ đổi ở `apply`; khối lưu ý không có tổng / trừ / làm tròn; SV không thấy; cờ tắt → prompt như trước (golden); `go.mod` / `package.json` không đổi; 0 `googleapis`; audit từng quyết định. `TestGuidanceIsolation`, `TestReflectOutputPII`… PASS (`check_run_names.sh`: **0 không đạt**). Chưa kiểm: G11-08 (xoá SV, `needs_recheck`) ở stack, cô lập hai bài tập bằng API |

**Tổng: 11 / 11 story PASS** (sau kiểm lại `975f9541`; GD-BUG-11 và đo thật chuyển lượt kiểm chứng). Gate «`-run` không rỗng» xanh; test cũ chỉ đổi số (contract 310 → 319 — chạy với `-tags testroutes` như CI, `expected_grading.json` chỉ thêm `teacher_label`, `expected` S7 = 7,00 giữ nguyên); dev Playwright (`grading-guidance`, `thread-suggest`, `question-hints`) 14 pass, 14 mobile bỏ theo thiết kế; `scripts/ui-antipatterns.sh` mã 0.

## Lỗi
| # | Mức | Tình trạng |
| --- | --- | --- |
| GD-BUG-1…3 | – | đã sửa, kiểm lại PASS |
| GD-BUG-4 (admin PUT / GET cờ lớp lạ → 500 / 200) | TB | **đã sửa** → `404` |
| GD-BUG-5 (chữ «Nháp») | Thấp | **đã sửa** → «Chờ duyệt» |
| GD-BUG-6 (nhãn công tắc `/class/settings`) | Thấp | **đã sửa** (đủ 4 nhãn + Bật / Tắt) |
| GD-BUG-8 (compose không chuyển `GUIDANCE_*`) | Thấp–TB | **đã sửa** (`GUIDANCE_ENABLED=false` có hiệu lực) |
| GD-BUG-9 seed-guidance không `PUBLISH` cờ | Thấp | **đã sửa** (`140a43e0`), kiểm lại PASS |
| GD-BUG-10 `REJECT` không lý do lưu NULL | Thấp | **đã sửa** (`19874727`), kiểm lại PASS |
| GD-BUG-11 `benchmarks/guidance/run.py` thiếu giao thức E3 (`teacher_label`, `WAITING_HUMAN`, báo cáo) | Thấp–TB | **chuyển lượt kiểm chứng** (P-106-10), không tính FAIL |
| GD-BUG-12 seed demo không nạp nháp lưu ý mẫu | Thấp | **đã sửa** (`975f9541`), kiểm lại PASS trên bản sao DB |
| Ghi nhận | – | (1) `regrade-stale` của bài tập lớp khác khi cờ lớp đó tắt → `409` trước `404`. (2) Lưu ý một ký tự duyệt được. (3) Đầu ra `reflect` vi phạm schema strict → job `FAILED` và mở cầu dao `GRADING` ~30 s (chấm bài cùng route bị chặn). (4) Q-S106-16 đã giải quyết bằng `regression_cases` ở `stats` (P-106-10) |

## Hoãn / chưa kiểm
Hoãn (PM): `benchmarks/guidance/run.sh`, `cmd/regress run` / `evaluate` thật, `regress export` (chờ phép), nhãn E3, smoke hồi quy toàn bộ (đợt R). Chưa kiểm: G02-14, G03 biên (5 / 3 / 0,49 / 0,50, QUIZ, 60 ca), G05 (250 nháp, round 20, kill worker), G06 UI + hiệu năng, G08 ngưỡng cosine + fixture thread, G09 khối «Tránh…» trên stack, G11-05 / 08.

## Trạng thái stack
`ep-qc` giữ ở `975f9541` (gateway / worker / frontend dựng lại, DB giữ nguyên nhưng **bẩn**: lưu ý v0…v4, 4 nháp đã duyệt / loại, 2 ca hồi quy, route `GRADING` trỏ **stub QC** cổng 8777). Muốn kiểm chứng giaovien sạch → dựng lại DB + seed (≈ 9 phút) rồi trả route về mặc định.

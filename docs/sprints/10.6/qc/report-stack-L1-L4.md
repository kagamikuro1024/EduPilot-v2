# QC sprint 10.6 — kiểm L1–L4 trên stack `ep-qc` (HEAD `9a8c8b20`)

Stack dựng từ HEAD (migration `00021`), 1440 + 375. Sinh chữ chỉ qua **stub QC** (`scripts/stub_llm.py`: tác vụ `reflect` chế độ `rf_*`, chấm bài hợp lệ, ghi system prompt) trỏ route `GRADING`; không 9router, không Gemini. Dữ liệu: BT03 lớp 761987, 16 bài; 8 bài duyệt có sửa theo «sở thích» (c2 chấm gắt hơn 1,0 ở 6 bài; c3 nhẹ hơn 0,5 ở 2 bài) bằng API `PUT/POST review`. Script: `scripts/gd_a…gd_e.py`, `gdlib.py`; UI: `frontend/e2e/qc-gd-ui-real.spec.ts`; ảnh: `shots/`. Can thiệp DB chỉ để dựng dữ liệu (xoá lưu ý nháp, đẩy `reviewed_at` để đổi khoá job, xoá khoá rate limit).

## 1. PASS
| TC | Nội dung | Bằng chứng |
| --- | --- | --- |
| G01-01, 02, 04, 05, 06, 16 | 4 cờ mặc định tắt; teacher `can_edit` 3/4 (`REGRESSION_CAPTURE` chỉ ADMIN); TA / SV `403`, không token `401`; PUT tăng `version`, `audit_log` (`guidance_flag.set`); `409` version cũ; `422` thiếu / sai kiểu; cờ lạ `422` | `gd_a.py` |
| G01-07 | `GUIDANCE_ENABLED=false` (qua override compose của QC): ghi cờ `409 GUIDANCE_DISABLED`, overview `enabled=false`, `reflect` `409`, 0 job mới | xem GD-BUG-8 |
| G01-09 | Cờ lớp tắt sau khi bật: `reflect` / `apply` / `regrade-stale` `409 GUIDANCE_DISABLED`; dữ liệu cũ còn (8 mục); màn duyệt không có khối `guidance` | `gd_e.py` |
| G01-18 | 6 phản hồi cho SV (`assignments`, `mine`, `threads`, `notifications`, `me/grades`) không có `guidance` / `ai_comments` / `first_reviewed` | `gd_e.py`; UI SV không thấy «Lưu ý chấm», 0 request `guidance` |
| G03-04, 11, 19 | `reflect` → `202`, job `SUCCEEDED`, 2 `DRAFT`; `support` = số `source_result_ids` (5, 2) do **code đếm**; không id nào trùng `users.id`; `llm_audit` GRADING / BATCH / `ok` | `gd_b.py` |
| G03-05 | Hai khoá Idempotency khác nhau, cùng `(assignment, max(reviewed_at), version)` → **cùng `job_id`** | |
| G03-13 | `case_id` lạ → bỏ cả mục (`dropped.case`); chiều sai → `dropped.support`, không DRAFT (TC sửa theo P-106-7) | |
| G03-15, 16, 17, 21 | từ khoá (6 biến thể «tổng», «trừ muộn», «làm tròn», «phạt», «TONG», «LAM TRON») → `dropped.keyword=6`; PII (tên, MSSV, email, SĐT) → `dropped.pii=4`; trùng chữ đã có → `dropped.duplicate`; 0 mục → `SUCCEEDED`, 0 DRAFT | |
| G03-22 | stub `error` / `invalid`: job `FAILED` (`REFLECT_FAILED` «Chưa rút được lưu ý. Thử lại.»), 0 DRAFT, `llm_audit` lỗi | |
| G03-23 | `REFLECT_MAX_PER_HOUR`: lần 4 trong giờ `429` | |
| G04-03, 05, 06 | `APPROVE` → `APPROVED` chờ áp dụng, `audit_log`; `409 VERSION_CONFLICT`, `409 ITEM_NOT_DRAFT`; sửa chữ hợp lệ giữ `text_original`; từ khoá / tên / MSSV / rỗng / > 400 ký tự → `422 GUIDANCE_TEXT_REJECTED`; `RETIRE` kèm `text` → `422` | `gd_c.py` |
| G04-07…10 | `apply` `{expected_version:0}` → `200 {version:1, stale}`, `assignments.guidance_version` = 1, `grading_guidance_versions` ghi, 0 việc chấm mới; cùng Idempotency-Key trả lại kết quả cũ; `409 VERSION_CONFLICT` / `409 NOTHING_TO_APPLY`; thiếu `expected_version` `422`; hai `apply` đồng thời: `[200, 409]`, phiên bản không trùng | |
| G04-11 | TA / SV / ADMIN `decide` + `apply` → `403`; TA xem danh sách `200`; SV `403`; bài tập lớp 1 qua đường lớp 2 → `404` | |
| G04-13 | `RETIRE` mục đã áp dụng → `pending_retire`; sau `apply` v3 → `RETIRED`, `retired_in_version=3` | |
| G05-05, 06, 08, 13, 20, 21, 24 | `stale` = `{count:8, editing:0, reviewed:8}` khớp SQL; TA `regrade-stale` → `202 {queued:8, skipped:{reviewed:8,edited:0,max_round:0}}`; 8 kết quả round + 1 `DONE`, ghim = bản hiện hành (3), có `ai_comments` (trừ bài trống — theo thiết kế); bài `REVIEWED` không đổi (băm điểm / tiêu chí / round / ghim); round cũ giữ nguyên `ai_scores`, `ai_comments`, `guidance_version`; chạy lại `200 queued 0`; không round trùng; SV `403` | `gd_d.py` |
| G05-01, 02, 04, 27 | Cả 14 lời gọi chấm tới stub có khối «Lưu ý của giảng viên (bản v3) — chỉ về điểm từng tiêu chí:», **sau RUBRIC, trước ĐÁP ÁN CHUẨN, trong system**; chỉ mục `APPROVED` đang hiệu lực v3 (không RETIRED / DRAFT / REJECTED); không từ «tổng / trừ / làm tròn / phạt»; cùng một khối giữa các bài | |
| G05-14, 15, 18 | Cờ tắt: công bố `200`, `exempt` 0. Cờ bật: 6 bài đã duyệt ghim v0 (bản hiện hành 3) công bố `200`, **không** `409`; `exempt{EDITED:3, ACCEPTED_AS_IS:3}` đúng so SQL; `audit_log` có `guidance_exempt`; bài nháp → `422 NOT_REVIEWED` | |
| G06-01 | `stats` v0: `n=8, mae_criterion=0,19, mae_total=0,50, bias_total=0,50, edited_criteria=7, edited_rate=0,22` — **khớp SQL độc lập** (cả `total`+`late_penalty`) | |
| UI | Ngăn bài tập 1440 + 375: khối «Lưu ý chấm», «Bản vN», Rút lưu ý / Áp dụng / Duyệt / Bỏ / Sửa / Cho nghỉ, dòng số đo «v0 · 8 bài · lệch tiêu chí 0,19 · lệch tổng 0,50 · sửa 22 %»; không tràn ngang; 0 icon Info; nút cao 44 px ở 375; `Duyệt` lạc quan «Chờ áp dụng», không toast «Thành công»; `Áp dụng` → «Đã áp dụng bản v4» + «8 bài chấm theo bản cũ · Chấm lại»; `/grading` chọn BT03 → «8 bài chấm theo bản cũ» + `Chấm lại`; màn duyệt bài bản cũ: «bản v3 · 3 lưu ý» + chip «Bản cũ»; SV không thấy | `shots/` |

## 2. Lỗi
| # | Mức | Mô tả |
| --- | --- | --- |
| **GD-BUG-4** | **Trung bình** | ADMIN `PUT /courses/{id-không-tồn-tại}/guidance-flags/{feature}` → **`500 INTERNAL`** (vi phạm FK); `GET` cùng lớp lạ → `200` với 4 cờ mặc định. TC-G01-05 kỳ vọng `404`, quy tắc chung «không 500». Sửa: kiểm lớp tồn tại → `404` |
| GD-BUG-5 | Thấp | Ngăn «Lưu ý chấm» hiện nhãn trạng thái **«Nháp»** cho mục chờ duyệt; US-GD-04 AC11 / TC-G04-17 cấm từ «nháp» (dùng «Chờ duyệt») |
| GD-BUG-6 | Thấp | `/class/settings`, nhóm «Học từ lần sửa»: nhãn công tắc là «Học từ lần sửa điểm», «Gợi ý bài đã trả lời khi đặt câu hỏi», «Nhớ lý do bỏ câu hỏi khi sinh đề», «Lưu ca kiểm thử chấm»; US-GD-01 AC5 quy định «Lưu ý chấm», «Ca kiểm thử», «Gợi ý câu hỏi tương tự», «Gợi ý khi sinh câu hỏi»; không có trạng thái 1–2 từ cạnh công tắc |
| GD-BUG-8 | Thấp–TB | `docker-compose*.yml` **không chuyển** `GUIDANCE_*` / `REGRADE_MAX_BATCH` / `REGRESSION_*` tới gateway và worker (bài học 10.5: `TestComposeDecisionEnvReachesWorker`). Đặt `GUIDANCE_ENABLED=false` trong môi trường compose không có tác dụng (ghi cờ vẫn `200`, `reflect` `202`); chỉ đúng khi QC thêm override (`scripts/qc-guidance-env.yml`) |
| Ghi nhận (Thấp) | – | (1) `regrade-stale` của bài tập lớp khác khi cờ lớp đó tắt → `409 GUIDANCE_DISABLED` trước `404`. (2) Lưu ý một ký tự («x») duyệt được (không có chữ tối thiểu). (3) Đầu ra `reflect` vi phạm schema strict (criterion lạ, vượt 8 mục) bị cổng gateway chặn **trước** bước kiểm (i) / (viii): job `FAILED` và 5 lỗi `BAD_RESPONSE` mở cầu dao GRADING (chấm bài cùng route bị `circuit_open` ~30 s). TC-G03-12 / 18 chỉ chạm được các bước (i), (viii) khi schema không được thực thi |

## 3. CHƯA CHẠY / chờ
- **Hoàn tác (P-106-8, TC-G04-19…28) — chờ dev L5.** Hiện: `Duyệt` / `Bỏ` **không** có «Hoàn tác»; `Áp dụng` **không** có `ConfirmIrreversible` (đúng như handoff; AC3a / AC5 do BA thêm ở SRS v1.5).
- Chưa kiểm: G02-04…06, 08, 09, 14 (ghim giữa hai lượt, ép lỗi giữa `apply`); G03-01…03, 07, 10 (biên 5 / 3 / 0,49 / 0,50, QUIZ, 60 ca); G05-09…11, 22, 23, 25 (250 nháp, round 20, kill worker, trừ muộn thật); G06-05…11 (bảng UI, 1.000 kết quả p95); G04-14, 16…18 (UI chi tiết), mức 375 của màn duyệt / công bố; hộp công bố nêu số bài miễn (G05-17); `bash scripts/ui-antipatterns.sh`.
- Hoãn theo PM: mọi lượt đo thật (E3), `regress export`, bench.

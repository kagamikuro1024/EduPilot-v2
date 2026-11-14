# QC sprint 10.6 — báo cáo L1–L3 (mức Go / DB, chưa cần stack)

HEAD kiểm `8a234e4b` (nhánh `sprint/10.6-guidance`, đã gồm 10.5 `6f6ca1f6`). Cơ sở dữ liệu thử: hai DB tạm trong container postgres của `ep-qc` (`qc106` trống; `qc106b` sao từ DB seed ở bản 20 rồi nâng lên 21), đã xoá sau lượt. Không chạm DB seed, không gọi LLM.

## 1. Kết quả
| Mục | TC | Kết quả |
| --- | --- | --- |
| Migration `00021` từ DB trống lên 21; `down` rồi `up` lại | G02-01 | **PASS**; chỉ thêm file mới, `git diff` không sửa migration cũ (chỉ `migrations_test.go` 20→21) |
| Bài đã duyệt trước 10.6 (5 `REVIEWED`, 2 `PUBLISHED` dựng trên bản sao DB): `first_reviewed_at/by` = `reviewed_at/by`; 169 `DRAFT` để `NULL`; sau `down` → `up` điền lại đúng 7 | G02-01, P-106-5 | **PASS** |
| Bảng / ràng buộc: `text` 401 ký tự, `direction` lạ, `support` âm, `APPROVED` không `decided_by`, `approved_in_version` khi `DRAFT`, `retired ≤ approved`, FK lớp lạ; `grc_unique` (kể cả `criterion_id` NULL trùng), `kind` lạ, `expected` không phải object; cờ lạ / trùng khoá; `reject_reason` lạ / không ở `REJECTED` | G02-02, 03 | **PASS** (mỗi ca bị đúng CHECK / UNIQUE / FK chặn) |
| Chỉ mục: `ggi_assignment_idx`, `grc_unique`, `grc_course_idx`, `question_bank_rejected_idx` đều bắt đầu `course_id`; FK sang `assignments` / `courses` là `ON DELETE CASCADE`, sang `users` không cascade | G02-02 | **PASS** |
| Trigger bất biến: sau `DONE` cấm đổi `guidance_version`, `ai_comments`, `first_reviewed_at/by`, `ai_scores`; đổi `reviewed_at` / cột khác vẫn được; trước `DONE` (đưa về `QUEUED`) ghi được `ai_comments`, `guidance_version`, rồi `DONE` thì khoá | G02-07, 16 | **PASS** |
| Tập lưu ý theo phiên bản (`GuidanceSetAt` chạy trên dữ liệu i1…i5): v0 ∅; v1 {i1,i3}; v2 {i1,i2,i3}; v3 {i1,i2}; i4 (DRAFT), i5 (REJECTED) không bao giờ | G02-10 | **PASS** |
| **Prompt v0 giống từng byte bản trước 10.6**: dựng prompt từ `c492c6e1` (trước L3) bằng cùng đầu vào, `cmp` với `testdata/essay_prompt_v0.golden` → trùng. Golden không do mã mới sinh | G02-12, G05-26 | **PASS** |
| Test cũ chỉ đổi số / thêm khoá, không bỏ assertion: `migrations_test.go` 20→21; `contract_test.go` 310→314 thao tác, so tập khoá `METHOD /path`: 0 bị xoá, +4 (đúng 4 route lưu ý); `llm_test.go` thêm cột | G02-15 | **PASS** |
| Env (SRS 4.1): mặc định `5 / 3 / 0,5 / 2 / 8 / 60 / 400 / 200 / 1,0 / 0,10 / false`; giá trị sai (`abc`, `0`, `1001`, `-1`, `401`, `maybe`, `2.5`, `REGRESSION_MAX_REL=2`) → lỗi nêu **tên biến** và khoảng cho phép | G01-08 (phần cấu hình) | **PASS** (mức `config.Load`) |
| Reflect: `buildReflectInput` chỉ rubric / ca / `existing`; `case_id` ngẫu nhiên; ≤ `MAX_CASES`; 8 bước kiểm đúng thứ tự SRS 4.2.8; `support` do code đếm; bỏ cả mục có từ khoá, PII (không dò được cũng bỏ), trùng Jaccard ≥ 0,8, cắt 8 mục; DRAFT + outbox ghi một giao dịch; `Det` được nối ở worker | G03-08…18 (đọc mã) | khớp SRS; chạy thật ở L4 |
| Test dev: `go test ./db ./internal/store ./internal/grading -count=1` | – | **xanh** (`db` 5,8 s, `store` 13,5 s, `grading` 62 s); contract `-tags testroutes` xanh (4 route lưu ý trong `scenarios_guidance_test.go`: 200 / 401 / 403 / 404 / 409 / 422) |

## 2. Ghi nhận
1. **GD-BUG-1 (Thấp)** — tên test mà SRS §9 / TC nêu nhưng **chưa có**: `TestReflectOutputPII`, `TestReflectInputNoIdentity` (TC-G03-09, 16 chạy `-run` sẽ **vacuous-pass**), `TestGuidanceSetAtVersion`, `TestGuidanceIsolation`. Dev có test tương đương tên khác (`TestReflectValidate`, `TestReflectInputNoSubmissionText`, `TestGuidancePinIsVersionUsed`). Các tên còn lại (`TestGuidanceVersionOnlyViaApply`, `TestPublishMixedGuidance`, `TestRegrade*`, `TestRegress*`, `TestRegressionCase*`, `TestRejectReasonHints`, `TestSimilarNoBodyLog`) thuộc lát L4+; kiểm lại khi bàn giao. Đề nghị dev đặt đúng tên hoặc BA vá SRS §9 (như ở 10.5).
2. **Q-S106-14** — một mục lưu ý có `case_id` **không thuộc đầu vào** (hoặc thuộc tiêu chí khác): mã **bỏ cả mục** (`dropped.case`); TC-G03-13 viết «ca sai bị loại khi đếm, mục còn `support` do code đếm». SRS 4.2.8 (ii) «mọi `case_id` ∈ đầu vào» đọc được theo cách của dev (an toàn hơn). QC **không sửa TC** khi chưa có đề xuất: xin PM / BA chốt; đề xuất QC: giữ hành vi của dev và sửa TC-G03-13 thành «có `case_id` lạ → bỏ cả mục; chiều sai (`wrong_direction`) chỉ không được đếm».
3. Không có gì phải sửa ở L1–L3 về hành vi; mọi TC cần stack (cờ / route, job `reflect`, ghim ở lần chấm thật, `ai_comments` sau `review`, TC-G01-07 `GUIDANCE_ENABLED=false` ở gateway) chờ L4 và lượt dựng `ep-qc`.

## 3. Chưa kiểm (chờ stack)
TC-G01-01…07, 09…12, 16, 18; TC-G02-04…06, 08, 09, 11, 13, 14; TC-G03-01…07, 10, 19…25 và các ca stub (`bad_*`, `pii_text`, `dup_text`, `error`/`invalid`/`slow`); TC-G05-01…04, 26 ở mức API.

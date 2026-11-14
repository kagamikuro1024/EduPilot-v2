# QC test case — US-GD-03 (rút lưu ý), US-GD-04 (duyệt / Áp dụng)
Nguồn: US.md + SRS 3.2, 3.3, 4.1, 4.2.5–4.2.10, 6 #3–6; Q-G-4, Q-G-10; P-106-1. Hộp đen; quy ước `tc-00-chung.md` (stub của QC điều khiển đầu ra LLM của `reflect`). `G=/courses/$C/assignments/$A/guidance`. Ca dữ liệu: BT03 lớp 1; kết quả `REVIEWED` có sửa tiêu chí (dựng qua API `review`).
Điều kiện đủ (SRS 4.2.5): ≥ 5 kết quả hiện hành `REVIEWED`/`PUBLISHED` **và** ≥ 3 điểm tiêu chí lệch ≥ 0,5 so `ai_scores`.

## US-GD-03 Rút lưu ý
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-G03-01 | AC1, AC2 | Cờ bật; 4 kết quả REVIEWED (< 5) | teacher `POST $G/reflect` (Idem K1) | `409 NOT_ENOUGH_DATA {results:4, edits:n}`; không job; giao diện «Cần thêm bài đã duyệt» | API+E2E |
| TC-G03-02 | AC1 | 5 REVIEWED nhưng chỉ 2 tiêu chí lệch ≥ 0,5 (biên: lệch 0,49 không tính; đúng 0,5 tính) | `reflect` | `409 NOT_ENOUGH_DATA` (2 edits); thêm 1 lệch đúng 0,50 → đủ; biên 0,49 → vẫn thiếu | API |
| TC-G03-03 | AC1 | Bài tập loại QUIZ / không ESSAY | `reflect` | Từ chối (`409`/`422`, ghi rõ); không job | API |
| TC-G03-04 | AC2 | Đủ điều kiện, stub `ok` | `reflect` | `202 {job_id}`; SSE `job.progress` theo giai đoạn; xong có mục `DRAFT` | API |
| TC-G03-05 | AC2, AC3 | Sau TC-G03-04 | `reflect` lại **cùng** `Idempotency-Key`; rồi khoá mới nhưng không bài mới được duyệt | Cùng khoá: trả `job_id` cũ; khoá khác nhưng cùng `(assignment, max(reviewed_at), guidance_version)`: không sinh nháp lần hai (`drafts:0` hoặc trả job cũ — ghi rõ) | API+DB |
| TC-G03-06 | AC3 | Duyệt thêm 1 bài | `reflect` | Khoá mới → job mới, sinh nháp mới (không trùng ≥ 0,8) | API |
| TC-G03-07 | AC2 | – | Không tự kích: đủ ngưỡng nhưng không bấm | `jobs(kind='grading.reflect')` = 0 sau 10 phút / sau mỗi lần `review` | API+DB |
| TC-G03-08 | AC4 | Bắt thân lời gọi tới stub (đầu vào `reflect`) | `reflect` ok | Chỉ gồm `rubric` (tên + hướng dẫn), `cases[]` (`case_id` ngẫu nhiên theo lần chạy, `criterion_id`, `ai_score`, `teacher_score`, `ai_comment`) ≤ 60 ca **gần nhất**, `existing` (lưu ý đã duyệt); chạy 2 lần → `case_id` khác nhau | API |
| TC-G03-09 | AC4 | Seed có tên / MSSV / email của SV trong văn bản bài, trong `ai_comments`, trong nhận xét sửa tay của GV | `grep` thân tới stub | 0: văn bản bài, **nhận xét sửa tay của GV (Q-G-10)**, `student_id`, `submission_id`, tên, MSSV, email, SĐT; `go test … -run 'TestReflectInputNoSubmissionText\|TestReflectInputNoIdentity' -v` PASS | API+GO |
| TC-G03-10 | AC4 | > 60 ca | `reflect` | Đúng 60 ca mới nhất | API |
| TC-G03-11 | AC5 | stub `ok` | Kiểm thân phản hồi stub và schema gửi đi | Schema strict: `items[{criterion_id enum, text ≤ 400, direction ∈ {STRICTER,LENIENT,CLARIFY}, case_ids ≥ 2}]`; **không** trường tổng / phạt / làm tròn | API |
| TC-G03-12 | AC6(i) | stub `bad_criterion` (criterion lạ) | `reflect` | Mục bị bỏ; `dropped` ghi lý do; không `DRAFT` cho nó | API+DB |
| TC-G03-13 | AC6(ii) | stub `bad_case` (case_id không có trong đầu vào) / `wrong_direction` (STRICTER nhưng ai ≤ teacher) | `reflect` | Ca sai bị loại khi đếm; mục còn `support` do **code đếm** (không lấy số LLM trả); nếu `support < 2` → bỏ | API+DB |
| TC-G03-14 | AC6(iii) | stub trả `support:99` giả, chỉ 1 ca hợp lệ | `reflect` | `support` lưu = 1 → bỏ (`< GUIDANCE_MIN_SUPPORT=2`); biên: đúng 2 ca → giữ | API+DB |
| TC-G03-15 | AC6(iv) | stub `total_keyword` (văn bản chứa «tổng», «trừ muộn», «làm tròn», «phạt», cả biến thể không dấu «TONG», «lam tron») | `reflect` | Mọi mục bị bỏ (so khớp `fold`); mục «tổng hợp» có từ khoá cũng bỏ (nêu ghi nhận: dương tính giả có thể xảy ra, không phải lỗi) | API+DB |
| TC-G03-16 | AC6(v), AC12 | stub `pii_text`: chứa tên SV của lớp (roster), MSSV, email, SĐT | `reflect` | Bỏ từng mục khớp; `go test … -run TestReflectOutputPII -v` PASS; log không PII | API+GO |
| TC-G03-17 | AC6(vi) | Đã có lưu ý APPROVED cùng tiêu chí; stub `dup_text` (Jaccard ≥ 0,8 và 0,79) | `reflect` | ≥ 0,8 bỏ; 0,79 giữ; REJECTED > 30 ngày không tính trùng (SRS 4.2.8 vii) | API+DB |
| TC-G03-18 | AC6(vii) | stub trả 12 mục hợp lệ | `reflect` | Tối đa **8** mục `DRAFT` | API+DB |
| TC-G03-19 | AC7 | Sau `reflect` ok | Kiểm DB | `DRAFT` có `support`, `source_result_ids[]` (id **kết quả**, không id SV — kiểm từng id không khớp `users.id`), `origin_job`; `assignments.guidance_version` **không đổi**; không mục nào tự `APPROVED` | DB |
| TC-G03-20 | AC7 | – | `llm_audit` của job | Task `GRADING`, làn `BATCH`, không nội dung; `pii_masked_count` hợp lý | DB |
| TC-G03-21 | AC8 | stub trả 0 mục hợp lệ | `reflect` | Job `DONE` `{drafts:0, dropped:{support, pii, trùng, …}}`; giao diện «Không có lưu ý mới» | API+E2E |
| TC-G03-22 | AC9 | stub `error` / `invalid` / `slow` (hạn) | `reflect` | Thử lại tối đa 3 lần (backoff); vẫn lỗi → dead-letter + job `FAILED`; **không** bản nháp nửa vời (0 `DRAFT` mới); giao diện «Chưa rút được lưu ý. Thử lại.»; chấm bài / duyệt trong lúc đó bình thường | API+DB+E2E |
| TC-G03-23 | AC10 | – | 4 lần `reflect` / giờ / bài tập (mỗi lần có bài duyệt thêm để khoá khác) | Lần 4 `429 RATE_LIMITED` + `Retry-After`; làn `BATCH`: chat trong lúc đó vẫn nhanh (so p95) | API |
| TC-G03-24 | AC11 | – | ta, `sv.kha`, admin: `reflect`; GV lớp 2; cờ tắt | TA `403`, SV `403`, **ADMIN `403`**, lớp khác `404`, cờ tắt `409 GUIDANCE_DISABLED` | API |
| TC-G03-25 | – | – | `GET $G` (teacher / ta) | `{current_version, items[], counts, stale:{count,versions}, reflect:{eligible, results, edits}}`; `eligible` đúng; TA đọc được; SV `403` | API |

## US-GD-04 Duyệt / Áp dụng
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-G04-01 | AC1 | Staff `/assignments/[id]`, cờ bật | Mở màn | Một khối «Lưu ý chấm» dưới rubric: dòng `bản v2 · 3 chờ duyệt` + `Rút lưu ý` (phụ); cờ tắt → khối **ẩn** | E2E |
| TC-G04-02 | AC2 | 3 DRAFT | Xem danh sách | Mỗi mục: tiêu chí (tên) · nội dung · `n lần sửa` · chiều (Gắt hơn / Nhẹ hơn / Làm rõ) · `Duyệt` · `Bỏ`; ngăn bằng đường kẻ 1 px (không card lồng); 6 trạng thái: `Chờ duyệt`, `Chờ áp dụng`, `Đã áp dụng`, `Chờ nghỉ`, `Đã bỏ`, `Đã nghỉ` | E2E |
| TC-G04-03 | AC3 | DRAFT i1 | `POST $G/items/{i1}/decide {decision:"APPROVE", version}` | `200`; trạng thái APPROVED chờ áp dụng (`approved_in_version` NULL); `audit_log` (mục, quyết định, người) | API+DB |
| TC-G04-04 | AC3 | DRAFT i2 | `decision:"REJECT"`; giao diện: `Bỏ` rồi «Hoàn tác» trong 5 s | REJECTED; hoàn tác khôi phục DRAFT; sau 5 s không hoàn tác được | API+E2E |
| TC-G04-05 | AC3 | i1 đã APPROVED | `decide` lại; `decide` với `version` cũ | `409 ITEM_NOT_DRAFT`; `409 VERSION_CONFLICT` | API |
| TC-G04-06 | AC4 | DRAFT i3 | `decide APPROVE {text:"mới …"}` với (a) chữ sạch; (b) có MSSV / tên roster; (c) từ khoá «làm tròn»; (d) 401 ký tự | (a) lưu, `text_original` giữ bản gốc; (b), (c) `422 GUIDANCE_TEXT_REJECTED`; (d) `422` | API+DB |
| TC-G04-07 | AC5 | i1 APPROVED chờ | `POST $G/apply {expected_version:0}` (Idem) | `200 {version:1, stale}`; `assignments.guidance_version=1`; i1 `approved_in_version=1`; hàng `grading_guidance_versions(v1)`; `audit_log`; **không** tự chấm lại (0 việc chấm mới) | API+DB |
| TC-G04-08 | AC5 | – | `apply` lại `expected_version:0`; `apply` khi không còn thay đổi | `409 VERSION_CONFLICT`; `409 NOTHING_TO_APPLY` | API |
| TC-G04-09 | AC5 | – | Hai `apply` đồng thời cùng `expected_version` | Một thắng, một `409 VERSION_CONFLICT`; không phiên bản trùng (khoá `FOR UPDATE`) | API |
| TC-G04-10 | AC5 | – | `apply` cùng `Idempotency-Key` lần hai | Trả lại kết quả cũ, không v+1 thêm | API |
| TC-G04-11 | AC6 | – | `ta@`: `decide`, `apply`, `reflect`; xem danh sách | `403` cả ba; TA xem được danh sách (`GET $G` 200); SV `403`; ADMIN `403`; lớp khác `404` | API |
| TC-G04-12 | AC6 | TA mở màn | Quan sát | Có danh sách; **không** nút `Duyệt` / `Bỏ` / `Áp dụng` (hoặc khoá, ghi rõ) | E2E |
| TC-G04-13 | AC7 | i1 đã áp dụng ở v1 | `decide {decision:"RETIRE"}`; `apply` | Mục «Chờ nghỉ» (`pending_retire`), **chưa** đổi hành vi (prompt v1 vẫn có i1); sau `apply` v2: `RETIRED`, `retired_in_version=2`; dựng lại v1 vẫn có i1, v2 không | API+DB |
| TC-G04-14 | AC8 | Sau `apply` | `/grading` | Hiện «N bài chấm theo bản cũ · Chấm lại» ngay; không tự chấm lại | E2E |
| TC-G04-15 | AC9 | Chưa lưu ý; chặn `GET $G` `500`; gõ chữ sửa rồi mất mạng | Mở khối / sửa | `EmptyState` một dòng; `<PageState>` + `Thử lại`; chữ sửa còn sau tải lại (`useAutosaveDraft`) | E2E |
| TC-G04-16 | AC10 | 375 px | Mở khối | Xếp dọc; nút vùng chạm ≥ 44 px; không tràn ngang | E2E |
| TC-G04-17 | AC11 | – | Quét chữ UI; `bash scripts/ui-antipatterns.sh` | Không dòng mô tả dưới tiêu đề, không icon Info, không từ «AI / RAG / PII / nháp» (kể cả trạng thái «Chờ duyệt» thay «Nháp»); script mã 0 | TAY |
| TC-G04-18 | RBAC | SV `/assignments/[id]` | Mở màn bằng `sv.kha` | Không thấy bất cứ gì về lưu ý (khối, chữ, DOM, mạng) | E2E |

## Nhánh lỗi
TC-G03-01, 02, 12…18, 21…23; TC-G04-05, 06, 08, 09, 15.

## Phân quyền
TC-G03-24, 25; TC-G04-11, 12, 18.

## Script chạy được
Pha 2: `docs/sprints/10.6/qc/scripts/gd_reflect.py` (dùng `stub_llm.py` mở rộng chế độ `reflect_*`), `frontend/e2e/qc-gd-guidance.spec.ts` (1440 + 375).

## Lịch sử sửa TC
(chưa có)

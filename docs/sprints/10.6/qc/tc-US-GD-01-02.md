# QC test case — US-GD-01 (cờ theo lớp), US-GD-02 (nền dữ liệu, ghim, bất biến)
Nguồn: US.md + SRS §2, 4.1, 4.2.1–4.2.4, 5, 6 #1–2; D61/D62; P-106-1. Hộp đen; quy ước `tc-00-chung.md`. `FL=/courses/$C/guidance-flags`; cờ: `GRADING_GUIDANCE`, `REGRESSION_CAPTURE`, `THREAD_SUGGEST`, `QUESTION_HINTS`. `A` = BT03 lớp 1.

## US-GD-01 Cờ theo lớp
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-G01-01 | AC1 | Seed sạch | `GET $FL` (teacher) | Đúng 4 cờ `{feature, enabled=false, version, can_edit}`; không hàng trong DB cũng ra 4 cờ tắt; `can_edit`: teacher 3 cờ true, `REGRESSION_CAPTURE` false | API |
| TC-G01-02 | AC2 | – | teacher `PUT $FL/GRADING_GUIDANCE {enabled:true, version}`; lặp với `THREAD_SUGGEST`, `QUESTION_HINTS` | `200`, `version` +1, `audit_log` (ai, cờ, trước / sau) | API+DB |
| TC-G01-03 | AC2 | – | teacher `PUT $FL/REGRESSION_CAPTURE` | `403`; không đổi; không `audit_log` thành công | API |
| TC-G01-04 | AC2 | – | admin `PUT $FL/REGRESSION_CAPTURE {enabled:true}`; admin bật `GRADING_GUIDANCE` | `200` cả hai (ADMIN cả 4 cờ); `audit_log` | API |
| TC-G01-05 | AC2 | – | ta, `sv.kha`, `sv.moi`, không token: `PUT` mọi cờ; GV / TA lớp 2: `PUT $FL/…` của lớp 1 | TA, SV: `403`; không token `401`; lớp khác `404` | API |
| TC-G01-06 | AC2 | – | ta / sv `GET $FL` | TA `403` hoặc chỉ đọc (Q-S106-2: SRS §6 #1 chỉ TEACHER, ADMIN → TA `403`); SV `403` | API |
| TC-G01-07 | AC3 | Gateway khởi động với `GUIDANCE_ENABLED=false` | `PUT` bật cờ; `POST …/guidance/reflect`, `…/apply`, `…/regrade-stale`; `GET $FL` | Ghi → `409 GUIDANCE_DISABLED`; cờ coi như tắt; `jobs(kind='grading.reflect')` = 0; chấm bài như trước | API+DB |
| TC-G01-08 | AC3 | `GUIDANCE_ENABLED=true` (mặc định) | Khởi động gateway với `GUIDANCE_MIN_RESULTS=abc` (sai kiểu) | Gateway **không chạy**, log nêu **tên biến** `GUIDANCE_MIN_RESULTS` (SRS 4.1) | TAY |
| TC-G01-09 | AC4 | Cờ `GRADING_GUIDANCE` đã bật rồi tắt; dữ liệu lưu ý cũ còn | Chấm 1 bài (round mới); bắt prompt (stub); `POST reflect` | Không job `reflect` mới (`409 GUIDANCE_DISABLED`); prompt **không** khối lưu ý; dữ liệu cũ còn trong DB nhưng không vào prompt | API+DB |
| TC-G01-10 | AC4 | `THREAD_SUGGEST` tắt | `GET …/threads/similar-confirmed?q=…` | `[]`; 0 lượt nhúng mới | API |
| TC-G01-11 | AC4 | `QUESTION_HINTS` tắt, có ≥ 5 câu loại có lý do | `GenerateQuestions` | Prompt sinh **không** khối «Tránh…» | API |
| TC-G01-12 | AC4 | `REGRESSION_CAPTURE` tắt | Sửa tiêu chí ≥ 1,0 + `review` | 0 hàng `grading_regression_cases` mới | API+DB |
| TC-G01-13 | AC5 | teacher, `/class/settings`, 1440 | Mở màn | Một nhóm «Học từ lần sửa», bốn công tắc nhãn «Lưu ý chấm», «Ca kiểm thử», «Gợi ý câu hỏi tương tự», «Gợi ý khi sinh câu hỏi»; trạng thái 1–2 từ; **không** câu mô tả; tooltip chỉ trên công tắc khoá («Chỉ quản trị viên bật»); công tắc «Ca kiểm thử» khoá cho teacher | E2E |
| TC-G01-14 | AC5 | – | Đổi công tắc | Lạc quan + «Hoàn tác»; không hộp xác nhận, không toast «Thành công»; hoàn tác khôi phục (`version` +2, 2 dòng `audit_log`) | E2E+DB |
| TC-G01-15 | AC5 | 375 px | Mở nhóm | Công tắc vùng chạm ≥ 44 px (đo); không tràn ngang | E2E |
| TC-G01-16 | AC6 | – | `PUT` `version` cũ; `feature` lạ; thân rỗng; `enabled` chuỗi | `409 VERSION_CONFLICT`; `422`; `422`; không `500` | API |
| TC-G01-17 | AC6 | Hai tab cùng teacher | Tab 2 lưu `version` cũ | `409` → tải lại hàng + báo tại chỗ | E2E |
| TC-G01-18 | – | `GET /courses/{id}/…` mọi DTO SV | Mở `/assignments/[id]`, `/threads`, `/questions` bằng SV khi cờ bật | Không từ / trường về «lưu ý», phiên bản, cờ (`grep` mọi phản hồi) | API |

## US-GD-02 Nền dữ liệu
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-G02-01 | AC1 | DB trống | `gateway migrate up` → `down` (về 0) → `up` | Sạch; số kế tiếp liền sau migration cuối đã merge; **không sửa file migration cũ** (`git diff`) | DB+DIFF |
| TC-G02-02 | AC1 | Sau migrate | Kiểm bảng / cột: `grading_guidance_items`, `grading_guidance_versions`, `course_guidance_flags`, `grading_regression_cases`; `assignments.guidance_version smallint NOT NULL DEFAULT 0`; `grading_results.guidance_version`, `ai_comments jsonb NULL`; `question_bank.reject_reason` | Đủ; FK phức hợp có `course_id`; mọi index phức hợp bắt đầu `course_id`; `grc_unique` dùng `coalesce(criterion_id,'')`; CHECK `feature`, `reject_reason`, `direction`, `text 1–400` chặn giá trị sai (thử chèn từng cái) | DB |
| TC-G02-03 | AC1 | – | Thử chèn: `retired_in_version <= approved_in_version`; `status` APPROVED không `decided_by`; `reject_reason` khi `review_status<>'REJECTED'`; hai hàng cùng `(course, kind, source, coalesce(criterion,''))` | Mỗi cái bị CHECK / UNIQUE chặn | DB |
| TC-G02-04 | AC2 | Cờ bật; bài tập `guidance_version=1` có 2 lưu ý | Chấm 1 bài (stub, bắt hai lượt) | `grading_results.guidance_version = 1` (cùng giao dịch ghi kết quả); **cả hai lượt** chấm dùng prompt có cùng khối «bản v1» | API+DB |
| TC-G02-05 | AC2 | Cờ tắt | Chấm 1 bài | Ghim `0`; prompt không khối | API+DB |
| TC-G02-06 | AC2 | Đổi `assignments.guidance_version` 1→2 **giữa lượt 1 và lượt 2** (stub chậm 5 s, `Áp dụng` trong lúc đó) | Chấm | Cả hai lượt vẫn dùng bản **ghim lúc nhận việc** (v1); không trộn | API+DB |
| TC-G02-07 | AC3 | Kết quả `DONE` | `UPDATE grading_results SET ai_scores/ai_comments/guidance_version` trực tiếp bằng DB | Lỗi do trigger (cả ba cột); `UPDATE` cột khác (feedback giảng viên) vẫn được | DB |
| TC-G02-08 | AC3 | Việc lỗi giữa chừng (attempt chưa `DONE`) | Thử lại (gateway kill worker) | Ghim lại ở attempt mới; không trùng `UNIQUE (course, submission, round)` | API+DB |
| TC-G02-09 | AC4 | Chấm 1 bài | Đọc `ai_comments` (DB) | `{criteria:{<id>: "<chữ AI>"}, overall}`; GV sửa nhận xét (`review`) **không** ghi đè `ai_comments`; kết quả cũ trước migration: `NULL` (không backfill) | API+DB |
| TC-G02-10 | AC5 | Lưu ý: i1 `approved_in_version=1`; i2 `=2`; i3 `=1, retired_in_version=3`; i4 DRAFT; i5 REJECTED | `go test ./internal/grading -run TestGuidanceSetAtVersion -v`; dựng tập tại v0, v1, v2, v3 qua kiểm DB | v0 ∅; v1 {i1,i3}; v2 {i1,i2,i3}; v3 {i1,i2}; i4, i5 không bao giờ; tất định (gọi 5 lần, đổi thời điểm → cùng kết quả) | GO+DB |
| TC-G02-11 | AC5 | Kết quả cũ ghim v1 | Dựng lại prompt tại v1 sau khi apply v2 (test Go) | Chuỗi y hệt chuỗi đã dùng ở lượt chấm cũ | GO |
| TC-G02-12 | AC6 | Bài tập chưa lưu ý | `go test ./internal/grading -run TestEssayPromptUnchangedAtV0 -v`; chấm bài, so prompt (stub) với prompt trước 10.6 (snapshot ở `main`) | **Giống từng byte** (`diff` rỗng); toàn bộ test chấm sprint 7 / 9 / 10.5 xanh (chạy lại `go test ./internal/grading/...`) | GO |
| TC-G02-13 | AC7 | – | GV / TA lớp 2, SV: mọi API guidance lớp 1 | `404` / `403` | API |
| TC-G02-14 | AC7 | Ép DB lỗi giữa `apply` (đổi quyền bảng `grading_guidance_versions` trên DB thử, khôi phục) | `apply` | Lỗi; **không** phiên bản nửa vời (`guidance_version` không đổi, mục không gán `approved_in_version`) | DB+API |
| TC-G02-15 | – | – | `git diff <gốc>... -- backend-go/internal/grading/*_test.go` | Chỉ thêm; không sửa assertion cũ (đặc biệt test prompt) | DIFF |
| TC-G02-16 | AC3 | – | `UPDATE grading_results SET guidance_version` khi `run_state='RUNNING'` | Cho phép (chưa `DONE`); sau `DONE` bị chặn | DB |

## Nhánh lỗi
TC-G01-07, 08, 16, 17; TC-G02-08, 14.

## Phân quyền
TC-G01-03, 05, 06, 18; TC-G02-13.

## Lịch sử sửa TC
(chưa có)

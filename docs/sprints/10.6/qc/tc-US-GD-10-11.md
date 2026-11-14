# QC test case — US-GD-10 (seed, cổng, đo E3, guide), US-GD-11 (ranh giới an toàn)
Nguồn: US.md + SRS §8, §9 (giao thức đo), 4.2.14; P-106-1 (nhãn S7: `teacher_label` = 8,00 cho mọi MAE; `expected` = 7,00 giữ nguyên). Hộp đen; quy ước `tc-00-chung.md`. **Lượt đo thật / bench / LLM thật chờ phép PM; TC kiểm cấu trúc dùng stub.**

## US-GD-10 Seed, cổng, đo
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-G10-01 | AC1 | Stack trống (`down -v` → `pnpm dev`, `SEED_ON_EMPTY_DB=true`) | `node scripts/seed.mjs`; SQL BT03 lớp 1 | Mã 0; provider `fake` có; BT03 có ≥ 8 kết quả `REVIEWED` với ≥ 4 tiêu chí bị sửa ≥ 0,5 theo «sở thích» (ví dụ c2 gắt hơn: kiểm bằng SQL cùng chiều ≥ 4/4); **bài tập khác để nguyên**; 4 cờ đều tắt sau seed | DB |
| TC-G10-02 | AC1 | – | `SEED_GUIDANCE_DEMO=1` | `GRADING_GUIDANCE` bật cho lớp 1 (lớp 2 tắt); có bản nháp lưu ý mẫu (`DRAFT`), chưa áp dụng (`guidance_version=0`) | DB |
| TC-G10-03 | AC1 | – | `node scripts/check-guidance-seed.mjs`; chạy seed lần hai | Mã 0; số hàng `grading_guidance_items`, `grading_results`, `course_guidance_flags` **không đổi** (idempotent) | TAY+DB |
| TC-G10-04 | AC2 | Repo | Chạy đủ 4 lệnh (3 `go test` + `playwright test grading-guidance.spec.ts thread-suggest.spec.ts question-hints.spec.ts`) | Xanh; log nêu từng test thật (không «no tests to run»); **không** kết nối mạng ngoài (đối chiếu `lsof`/log) | GO+E2E |
| TC-G10-05 | AC3 | `grading-guidance.spec.ts` | Đọc spec (không mock toàn bộ API) | Đi (a)…(f): `Rút lưu ý` → nháp có `n lần sửa`; `Duyệt` hai + `Bỏ` một; `Áp dụng` → v1; `/grading` «N bài chấm theo bản cũ»; `Chấm lại` (UNREVIEWED) → ghim v1; công bố lô trộn `409 MIXED_GUIDANCE` rồi công bố được sau chấm lại; nhánh lỗi: LLM lỗi → «Chưa rút được lưu ý»; TA `Duyệt` → `403`; SV `/assignments/[id]` không thấy gì | DIFF |
| TC-G10-06 | AC3 | – | Tự đi lại (a)–(f) + nhánh lỗi ở `ep-qc` bằng API / giao diện | Đúng từng bước (dùng stub) | E2E |
| TC-G10-07 | AC4 | – | `go test ./internal/contract/...`; so `openapi.yaml` với SRS §6 #1–12; `bash scripts/ui-antipatterns.sh`; `grep -rn "@/mock" frontend/src/features/<lưu ý chấm, câu tương tự>` | Xanh; đủ 12 endpoint + 6 mã lỗi mới; mock liên quan đã xoá; antipatterns mã 0 | GO+DIFF |
| TC-G10-08 | AC5 | `benchmarks/guidance/run.py` | Đọc mã + `git diff` | Python chuẩn (không thư viện ngoài), chỉ HTTP (không import mã Go: `grep -r "import.*backend" benchmarks/guidance` = 0); giao thức SRS 9: lưu ý rút từ 20 bài đầu (qua duyệt), đo trên 40 bài sau, thêm bài tập thứ hai cùng lớp | DIFF |
| TC-G10-09 | AC5, P-106-1 | – | `grep -rn "teacher_label\|expected" benchmarks/guidance cmd/regress backend-go/internal/guidance`; chạy `run.py` với bộ nhỏ có S7 | Phép MAE đọc **`teacher_label`** (S7 = 8,00); `expected` (S7 = 7,00) không dùng cho MAE nhưng **còn nguyên** trong `expected_grading.json`; `git diff <gốc>... -- docs/specs/FEAT-assignments/fixtures backend-go/**/expected_final_grades.csv` không sửa `expected` | DIFF+TAY |
| TC-G10-10 | AC5 | Bộ mẫu QC (≥ 60 bài + nhãn tạm) **hoặc** bộ E3 người (chờ chủ dự án) | `run.py` (provider stub / fake) | Báo cáo `benchmarks/reports/guidance-before-after.{json,md}` + **SVG** (không PNG): MAE tiêu chí / tổng trước – sau, lệch trung bình **có dấu**, số tiêu chí xấu đi, số lần sửa tay, **nguồn nhãn** («nhãn S7 = 8,00 teacher_label»), cỡ mẫu, «Giới hạn»; QC tính lại MAE từ dữ liệu thô (sai số ≤ 0,005) | TAY |
| TC-G10-11 | AC5 | Chưa có nhãn người | Chạy `run.py` | `WAITING_HUMAN` + lý do; **không** chạy giả, không dùng điểm LLM làm chuẩn (AC10) | TAY |
| TC-G10-12 | AC6 | Báo cáo | Dòng kết luận | **Bật được / Giữ tắt** theo: MAE tổng giảm ≥ 20 % **và** |lệch TB| không xấu hơn cơ sở quá 0,20 **và** tiêu chí vốn đúng nay sai ≤ 1,5× cơ sở; QC tự áp ngưỡng lên số của mình, kết luận khớp; không đạt → «Giữ tắt», **ngưỡng không nới** (`git diff` hằng); số PoC 49 % ghi «tham chiếu, không cam kết» | TAY+DIFF |
| TC-G10-13 | AC7 | PM giao dựng agent «Giáo viên» (nếu có) | Xem báo cáo kiểm chứng | QC ghi: nguồn nhãn (agent), tiêu chí duyệt / bỏ từng lưu ý, nhãn S7; kết quả **không** dùng làm nhãn người thật; PM tắt agent sau kiểm chứng. Nếu chưa giao: «chưa kiểm (không tính FAIL)» | TAY |
| TC-G10-14 | AC8 | `docs/guides/grading-guidance.md` | Làm theo **từng bước** trên stack | Có: bật cờ, khi nào bấm `Rút lưu ý`, duyệt, `Áp dụng`, `Chấm lại`, đọc số đo, lùi (tắt cờ, `GUIDANCE_ENABLED=false`), quên / xoá dữ liệu, giới hạn, chi phí (số lời gọi mỗi lần rút / chấm lại — QC đo bằng `llm_audit` và so với guide); mỗi bước đúng | TAY |
| TC-G10-15 | AC9 | – | `ARCHITECTURE.md`, `SYSTEM_DESIGN.md`, `DECISIONS.md` D61/D62, FLOWS F9/F4/F12, `PROGRESS.md` | Khớp mã (bảng, cột, env §4.1, job `grading.reflect`, stream `ep:grading.reflect`); `PROGRESS` ghi Nợ (M3, M5, M7, M8) | DIFF |
| TC-G10-16 | AC10 | Gateway tắt | `benchmarks/guidance/run.py` | Mã ≠ 0 «Gateway chưa chạy»; không ghi / sửa báo cáo (hash trước / sau) | TAY |
| TC-G10-17 | – | – | `git diff <gốc>... -- '*_test.go' frontend/e2e '*expected_final_grades*' docs/specs/FEAT-assignments/fixtures`; quét khoá | Chỉ thêm test; không nới / `skip`; không sửa contract test hay `expected_final_grades.csv`; 0 khoá | DIFF |
| TC-G10-18 | – | – | Toàn bộ playwright cũ ở stack có / không cờ | Không hồi quy; mặc định tắt: 0 lời gọi `reflect`, prompt chấm giống `main` | E2E |

## US-GD-11 Ranh giới an toàn (xuyên suốt)
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-G11-01 | AC1 | Repo | `grep -rn "guidance_version" backend-go/internal --include=*.go | grep -iE "UPDATE|SET "`; `go test … -run TestGuidanceVersionOnlyViaApply -v` | Chỉ `apply` (TEACHER) đổi `assignments.guidance_version`; `reflect`, worker chấm, job xoá dữ liệu SV **không**; test PASS (và đỏ nếu thêm UPDATE thử vào bản sao tạm) | GO+DIFF |
| TC-G11-02 | AC1 | Chạy `reflect` + chấm + xoá SV | Theo dõi `assignments.guidance_version` (trigger log / so trước-sau) | Không đổi cho tới khi `apply` | DB |
| TC-G11-03 | AC2 | – | `go test … -run TestGuidanceNoTotalsInPrompt -v`; duyệt chữ sửa tay có «tổng», «trừ muộn», «làm tròn» | PASS; chữ bị chặn `422` (cả lúc sửa tay); schema lưu ý không có trường tổng / phạt | GO+API |
| TC-G11-04 | AC3 | Seed PII trong bài / `ai_comments` / nhận xét sửa tay | Quét: đầu vào `reflect`, mọi bảng mới, log (`docker compose logs`), `llm_audit` | 0 văn bản bài, `student_id`, tên, MSSV, email; `source_result_ids` là id kết quả; không có API ánh xạ ngược id kết quả → SV ngoài người có quyền xem kết quả đó | DB+API |
| TC-G11-05 | AC4 | Hai bài tập A, B (cùng lớp) và lớp 2 | Duyệt + áp lưu ý «ZQ-A-1» ở A; chấm B và bài lớp 2 | Prompt B / lớp 2 **không** có «ZQ-A-1»; `TestGuidanceIsolation` PASS; thử gọi API guidance với `assignment` của lớp khác → `404` | API+GO |
| TC-G11-06 | AC5 | Cờ bật, có lưu ý | `sv.kha`: `/assignments/[id]`, kết quả đã công bố, `/threads`, thông báo, mọi DTO + DOM; quét ≥ 30 phản hồi | 0 trường / chữ về lưu ý, phiên bản, cờ, số đo («lưu ý chấm», «guidance», «bản v», «AI», «nháp»); kiểm hợp đồng DTO SV không đổi so `main` (`openapi` diff chỉ phía nhân viên) | API+E2E |
| TC-G11-07 | AC6 | Tắt `GRADING_GUIDANCE`; rồi `GUIDANCE_ENABLED=false` | Chấm 1 bài có lưu ý trong DB; so prompt với `main` (stub) | **Byte-for-byte** như trước 10.6 (`diff` rỗng); dữ liệu lưu ý còn trong DB | API+DB |
| TC-G11-08 | AC7 | Lưu ý i1 `support=2` nguồn {R1, R2}; xoá / ẩn danh SV của R1 (job xoá PR) | Chạy job | R1 bỏ khỏi `source_result_ids`; `support'=1 < 2` → `needs_recheck=true` hiện «Cần xem lại»; **không** tự nghỉ, hành vi chấm không đổi (prompt giữ i1); ca hồi quy liên quan xoá | DB+E2E |
| TC-G11-09 | AC8 | Repo | `git diff <gốc>... -- backend-go/go.mod go.sum frontend/package.json`; `grep -rniE "hindsight|mem0|letta|zep" backend-go frontend/src docker-compose*.yml` | Không thư viện / dịch vụ bộ nhớ agent; không mục ngoài bảng ARCHITECTURE | DIFF |
| TC-G11-10 | AC9 | – | Mỗi quyết định: duyệt, bỏ, áp dụng, nghỉ, bật cờ, xuất ca (nếu có) | Một dòng `audit_log` (ai, gì, trước / sau); `reflect` có `llm_audit` (không nội dung) | DB |
| TC-G11-11 | Tổng | `ep-qc` | Bắt kết nối ra ngoài suốt pha 2 | Không `generativelanguage.googleapis.com`; chỉ 9router / stub QC; nếu có → lỗi nghiêm trọng | TAY |
| TC-G11-12 | AC3 | – | Quyền xem kết quả: teacher lớp, TA lớp, SV chính chủ (đã công bố) | Chỉ họ thấy kết quả tương ứng; SV X không thấy `source_result_ids` / lưu ý | API |

## Nhánh lỗi
TC-G10-11, 16; TC-G11-03, 07, 08.

## Phân quyền
TC-G11-05, 06, 12.

## Lịch sử sửa TC
(chưa có)

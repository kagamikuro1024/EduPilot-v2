# Câu hỏi của QC — sprint 10.6 (FEAT-grading-guidance)
Phát sinh khi viết TC pha 1 (2026-11-14). Mỗi câu có phương án QC; TC dùng phương án đó cho tới khi PM trả lời. Không tự đoán hành vi sản phẩm.

| # | TC | Câu hỏi (chỗ SRS/AC mơ hồ, mâu thuẫn hoặc thiếu) | Phương án QC đề xuất | PM trả lời |
| --- | --- | --- | --- | --- |
| Q-S106-1 | G03 (mọi TC `reflect`) | QC không có cách đọc đầu vào / ép đầu ra của lời gọi `reflect` nếu tác vụ LLM của nó không cấu hình được (task `GRADING`, Q-G-15). Route `GRADING` có thể trỏ vào stub OpenAI-compatible của QC được không, và prompt / schema nằm trong thân yêu cầu (như `json_object` + «JSON Schema sau:» ở Threads) để stub parse `case_id`? | Có: route `GRADING` / tác vụ của `reflect` chỉnh được ở `/settings/llm`; dev nêu trong handoff nơi schema và `case_id` xuất hiện trong thân yêu cầu |  **Chấp nhận phương án QC** (PM, P-106-4) |
| Q-S106-2 | G01-06, G04-11 | `GET …/guidance-flags` (SRS §6 #1): quyền ghi «TEACHER, ADMIN»; TA đọc cờ được không? US-GD-01 AC2: TA `403` (cho `PUT`). `/class/settings` TA có thấy nhóm «Học từ lần sửa» (chỉ xem) hay không? | TA `403` ở `GET` (cờ là cấu hình lớp); `/class/settings` chỉ TEACHER (+ ADMIN) |  **Chấp nhận phương án QC** (PM, P-106-4) |
| Q-S106-3 | G06-04 | Quy ước dấu `bias_total` / lệch trung bình có dấu: `ai − chốt` (dương = AI chấm cao) hay ngược lại? | `ai_total_before_late − total_before_late chốt`; dương = AI chấm cao hơn GV (nên đổi chiều lưu ý sang STRICTER) |  **Chấp nhận phương án QC** (PM, P-106-4) |
| Q-S106-4 | G03-05 | AC3 US-GD-03: cùng khoá `(assignment, max(reviewed_at), guidance_version)` mà **khoá `Idempotency-Key` khác** — trả `job_id` cũ, `202` job mới `drafts:0`, hay `409`? SRS 3.2 nói «trả lại `job_id` cũ». | Trả `200/202` kèm `job_id` cũ (không job mới, không LLM) |  **Chấp nhận phương án QC** (PM, P-106-4) |
| Q-S106-5 | G05-14, 15 | AC6 US-GD-05: «tập sẽ công bố có > 1 phiên bản **hoặc có giá trị khác** `assignments.guidance_version`». Tập chỉ gồm toàn bộ v0 khi hiện hành v1 (chưa ai chấm lại) bị chặn — kể cả công bố lần đầu bài chưa từng chấm bằng v1? | Bị chặn (`MIXED_GUIDANCE` với `versions:[0]`) cho tới khi chấm lại hoặc công bố bằng `result_ids[]` tường minh; đúng AC |  **Chấp nhận phương án QC** (PM, P-106-4; Q5/Q6: nháp ≠ hiện hành `409` kể cả `result_ids`, bài GV đã duyệt miễn — P-106-2) |
| Q-S106-6 | G05-17 | `result_ids[]` tường minh cùng phiên bản **khác** `assignments.guidance_version` (ví dụ chọn các bài v0 khi hiện hành v1)? AC6: «công bố tường minh `result_ids[]` cùng một phiên bản vẫn được» — có kể cả bản cũ? Q-G-1 nói «không có công bố bản cũ» | Bản cũ (≠ hiện hành) bị `409 MIXED_GUIDANCE` kể cả `result_ids` tường minh; chỉ cùng một phiên bản **bằng** hiện hành được công bố |  **Chấp nhận phương án QC** (PM, P-106-4; Q5/Q6: nháp ≠ hiện hành `409` kể cả `result_ids`, bài GV đã duyệt miễn — P-106-2) |
| Q-S106-7 | G03-15 | Từ khoá cấm (`tổng`, `trừ`, `muộn`, `làm tròn`, `phạt`) khớp theo **chuỗi con** sau `fold` — «tổng hợp», «trừng phạt», «muộn màng» bị loại nhầm. Chấp nhận dương tính giả? | Chấp nhận (an toàn hơn); ghi nhận ở guide; không sửa để nới |  **Chấp nhận phương án QC** (PM, P-106-4) |
| Q-S106-8 | G07-03 | «không chuỗi dài» ở `expected`: có ngưỡng độ dài tối đa (SRS 5.5 «service kiểm không có chuỗi dài»)? | `expected` chỉ khoá `{ai_score, teacher_score, direction, rubric_hash, state}`; mọi chuỗi ≤ 64 ký tự |  **Chấp nhận phương án QC** (PM, P-106-4) |
| Q-S106-9 | G08-06 | Cách dựng mục kho `source_type='TICKET'` qua API công khai để kiểm AC2 US-GD-08 (không rò nguồn riêng): hỏi riêng → escalate → GV trả lời có đủ để sinh `TICKET`? | Có; QC dựng theo F5; nếu không có đường nào sinh `TICKET` trong stack giả → ghi «không kiểm được, test Go của dev phủ» |  **Chấp nhận phương án QC** (PM, P-106-4) |
| Q-S106-10 | G10-01 | Seed: «≥ 8 kết quả `REVIEWED` với ≥ 4 tiêu chí sửa ≥ 0,5 theo một sở thích» — BT03 sẵn có 16 bài; seed hiện **chưa chấm** (`grading_results` rỗng sau seed, đã thấy ở 10.5). Seed mới sẽ tự chấm + duyệt bằng API không? Cần LLM giả nào để ra điểm tất định? | Seed chạy `grade-all` bằng provider `fake` rồi `review` có sửa; điểm tất định từ fixtures S1…S7 |  **Chấp nhận phương án QC** (PM, P-106-4) |
| Q-S106-11 | G10-09 | `teacher_label` nằm ở đâu (`expected_grading.json` khoá mới?) và ai dựng cho 7 mẫu S1…S7 ngoài S7 (chỉ S7 = 8,00 được nêu) — các mẫu khác `teacher_label = expected`? | Mặc định `teacher_label = expected` trừ S7 = 8,00 (P-106-1); BA / dev ghi bảng trong handoff |  **Chấp nhận phương án QC** (PM, P-106-4) |
| Q-S106-12 | G05-12 | `regrade-stale` `UNREVIEWED` đếm theo `status=DRAFT`; bài `NEEDS_REVIEW` / `FAILED` (run_state FAILED) có chấm lại không? | Chỉ `DRAFT` có `run_state=DONE`; `FAILED` chấm lại bằng đường hiện hành |  **Chấp nhận phương án QC** (PM, P-106-4) |
| Q-S106-13 | G05-33 | P-106-2: «Không bao giờ ghi đè điểm GV đã chốt khi chấm lại» — `regrade-stale ALL_UNPUBLISHED` có còn chọn bài **GV đã duyệt** (`REVIEWED`) không? Nếu còn, chúng quay về nháp ở round mới (điểm chốt ở round cũ), hay bị loại khỏi lô? | Loại bài đã duyệt khỏi `regrade-stale` (không bao giờ tạo nháp mới đè lên bài GV đã chốt); muốn chấm lại phải thao tác riêng của GV; `ConfirmIrreversible` chỉ còn cho trường hợp này nếu BA giữ | **Đã trả lời bởi spec v1.2** (BA `0ca2e48e`): `regrade-stale` không bao giờ chọn REVIEWED / bài đã sửa; không còn `scope` |

## Việc QC ghi nhận không cần trả lời
- Mọi lượt đo thật (E3, bench 1.000 bài, `regress run` thật, `regress export` khi `REGRESSION_EXPORT_ALLOWED=true`) chờ phép PM từng lượt; TC cấu trúc dùng stub / `fake`.
- Nhãn S7: `expected` = 7,00 (nháp AI lượt 1) **giữ nguyên**; MAE dùng `teacher_label` (S7 = 8,00) — theo P-106-1.
- TC can thiệp DB (đổi quyền bảng, nạp 1.000 kết quả, kill worker) chỉ trên `ep-qc`, khôi phục ngay, ghi ở báo cáo.

## Hỏi thêm sau khi kiểm L1–L3 (2026-11-14)
| # | Câu hỏi | Đề xuất QC |
| --- | --- | --- |
| Q-S106-14 | Mục lưu ý do LLM trả có `case_id` ngoài đầu vào / thuộc tiêu chí khác: bỏ **cả mục** (mã hiện hành, SRS 4.2.8 (ii)) hay chỉ không đếm ca đó rồi xét `support`? | Giữ hành vi hiện hành; sửa TC-G03-13 (chiều sai chỉ không được đếm; `case_id` lạ → bỏ cả mục) |

## Hỏi thêm khi viết TC Hoàn tác (P-106-8)
| # | Câu hỏi | Đề xuất QC |
| --- | --- | --- |
| Q-S106-15 | Hoàn tác `Duyệt` / `Bỏ`: (a) hoãn gửi `decide` 5 s hay API đặt lại `DRAFT`; (b) mục đã sửa chữ trước khi duyệt: hoàn tác giữ chữ đã sửa hay khôi phục chữ gốc; (c) mã lỗi khi mục đã thuộc phiên bản đã áp dụng | (a) cách nào cũng được, TC-G04-19/20 chấp nhận cả hai; (b) giữ chữ đã sửa, `text_original` giữ bản gốc của AI; (c) `409 ITEM_NOT_DRAFT` (đã có) |

## Hỏi thêm sau khi kiểm L5 / L6 trên stack (2026-11-14)
| # | Câu hỏi | Đề xuất QC |
| --- | --- | --- |
| Q-S106-16 | US-GD-07 AC6: «Teacher thấy **số ca** của lớp mình ở màn thống kê» — SRS §6 không có endpoint nào trả số ca hồi quy, giao diện cũng chưa hiện. Làm hay bỏ AC? | Thêm `regression_cases:{GRADING_EDIT, …}` vào `GET …/guidance` (hoặc bỏ AC6 phần Teacher); TC-G07-13 tạm ghi «không kiểm được» |

## Hỏi thêm khi viết TC P-106-11 / P-106-12 / UX (2026-11-14)
| # | Câu hỏi | Đề xuất QC |
| --- | --- | --- |
| Q-S106-17 | Nhãn UX «(a)…(m)» của PM chưa có văn bản gốc trong repo; QC ánh xạ theo nhật ký giaovien thành TC-UX-01…16 (a = Duyệt bài sau gõ nhận xét, b = khoá đăng nhập, c = lọc mặc định + «Chấm tất cả» khoá, d = «Đã nộp/Tổng 0/60», e = hàng chờ 20 dòng, f = tự nhảy bài, g = bộ đếm + «Cần xem kỹ», h/i = nhãn số đo, j = Hoàn tác ngắn, k = chữ gốc sau sửa, l = Chấm lại không tiến độ / không tổng kết, m = hộp Công bố). | PM xác nhận / chỉnh danh sách; mục nào PM loại thì QC bỏ TC tương ứng |
| Q-S106-18 | `dropped` ở mã dev (`16fdf4c9`): đối tượng `{criterion,case,…,by_criterion:{c2:{keyword:1}}}`; SRS v1.7 / openapi: mảng `[{criterion_id,reason,count}]`. UI dev in «N lưu ý bị loại · <tiêu chí>: <lý do>»; SRS: một dòng «N lưu ý bị loại», không liệt kê lý do; N dev bỏ `duplicate`, SRS tính cả `duplicate`. | Theo SRS v1.7 (TC-P12-13, 16, 18 sẽ FAIL nếu giữ mã); nếu PM thích giao diện có tiêu chí + lý do thì BA vá SRS + openapi + luật chữ UI |
| Q-S106-19 | `TestReflectKeywordExempt` (SRS 4.1, 4.2.8) chưa có trong mã; QC không sửa mã dev. | QC đưa mẫu `TestReflectKeywordExempt.go.txt`; dev thêm vào `reflect_internal_test.go` (gate `-run` P-106-7) |

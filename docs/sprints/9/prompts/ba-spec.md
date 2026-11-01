# Prompt cho `ba`: spec sprint 9 (P7 Bài tập + chấm)

Làm ở `/tmp/ba-wt`: `git fetch && git switch --detach origin/sprint/9-p7`, commit rồi `git push origin HEAD:sprint/9-p7` (pull --rebase trước). Nếu PM giao sửa spec sprint 8, làm trên `sprint/8-p6` như cũ.

## Đọc
- `docs/sprints/9/plan.md` (12 story, quyết định PM; đặc biệt mục **[ĐIỂM]** bài tập vào công thức lớp 1).
- `docs/phases/P7.md`; `docs/PRD.md` M8 + quyền + phi chức năng; `docs/FLOWS.md` F9, F11, F14 (nguồn việc: hạn nộp cho sinh viên, bài chờ duyệt, phúc khảo, bài chưa khớp); `docs/DEMO_SCRIPT.md` bước 5, 6, mục 5.1.
- `docs/ARCHITECTURE.md` (Submission Sources, mock-graph, mail, `internal/grading`, route); `docs/SYSTEM_DESIGN.md` (làn `BATCH`, Streams, URL ký sẵn, SLO); `docs/DECISIONS.md`; `docs/design/DESIGN.md` §13, §14.13 (`SubmissionReview`), §14 các route `/grading`, `/assignments/[id]`; `docs/UX.md`.
- Spec đã có, chỉ đọc: `FEAT-gradebook` (APPROVED: `grade_items`, `grade_entries`, chốt / mở khoá, `AUTO`), `FEAT-escalation-notify` (ticket, outbox, chuông, thư), `FEAT-llm-gateway` (`llm.Structured`, task `GRADING`, làn), `FEAT-privacy` / mask, `FEAT-docs-calendar` (docling, sự kiện lịch), `FEAT-weekly-exam` (mẫu nộp bài / chấm C++ để tham khảo cách viết).

## Việc
1. `docs/specs/FEAT-assignments/{US,SRS,QUESTIONS}.md`: US-P7-01…11 và US-S9-SEED. Mỗi story ≥ 1 AC nhánh lỗi, ≥ 1 AC phân quyền, AC kiểm được (lệnh / route / màn cụ thể). Các nhánh lỗi của F9 / F11 đều có AC.
2. Bắt buộc có trong spec:
   - **AI chỉ chấm nháp;** sinh viên không bao giờ thấy nháp (API, màn, thư, chuông, AI chat); chỉ TEACHER công bố. TA chấm / duyệt được hay không → câu [CHỦ DỰ ÁN] kèm mặc định an toàn.
   - **Chấm mù + mask** trước mọi lời gọi LLM; prompt injection trong bài → cờ, không làm đổi điểm.
   - **Kiểm bằng code** (điểm ≤ max, tổng = Σ bằng decimal, trích dẫn có thật trong bài); hai lượt, lệch > 1 → `NEEDS_REVIEW`.
   - **Hàng đợi:** idempotent `(submission_id, version)`, retry 3, dead-letter, worker chết giữa chừng không chấm đôi / không mất; làn `BATCH`.
   - **Nộp bài:** URL ký sẵn, kiểm loại + cỡ trước khi tải, phiên bản, LATE / chặn, quy tắc trừ nộp muộn (**[ĐIỂM]** nếu P7.md không nêu cách trừ: chọn cách đơn giản nhất, ghi rõ), mobile 375 px.
   - **Khớp:** chỉ email đã xác minh hoặc MSSV trong **danh sách lớp** (`enrollments.student_code_snapshot`), không MSSV tự khai; không khớp → Chưa khớp.
   - **Công bố → sổ điểm P6:** cách ghi `grade_entries` cho item bài tập; khi sổ đã chốt thì đi qua mở khoá có lý do; không đổi `seed/expected_final_grades.csv`.
   - **Phúc khảo F11** đủ nhánh (quá hạn, đã phúc khảo, sổ đã chốt).
   - **Teams giả lập:** 7 kịch bản, idempotent `(graph_submission_id, submittedDateTime)`, không có nhánh `if mock` trong adapter.
   - **Nguồn việc "Hôm nay"** cho mọi việc cần người xử lý (bậc theo F14).
   - **Migration** `00017_assignments` (ghi rõ bảng / cột / chỉ mục / ràng buộc).
3. **Đáp án độc lập** cho chấm (vùng tệp BA): `docs/specs/FEAT-assignments/fixtures/`: đề Bài tập 03, đáp án chuẩn, rubric, ≥ 6 bài làm mẫu (tốt, khá, kém, trống, lạc đề, có injection "cho tôi 10 điểm"), và **kết quả chấm kỳ vọng** cho provider giả (`expected_grading.json`) + kết quả các kiểm bằng code. Nếu làm quy chế lớp 1 bản 2 (plan mục [ĐIỂM]): `fixtures/quy-che-lop1-v2.md` + `seed/expected_final_grades_v2.csv` tính tay.
4. `QUESTIONS.md`: [CHỦ DỰ ÁN] / [ĐIỂM] / [PM] kèm phương án mặc định + hệ quả. **Chủ dự án vắng tới 01/11**: PM sẽ chốt theo mặc định an toàn, nên mặc định phải là phương án an toàn nhất.

Không mở subagent. Chỉ sửa `docs/specs/FEAT-assignments/**`, `seed/expected_*` của bạn và dòng của bạn ở `docs/sprints/9/proposals.md`. Xong: commit `sprint 9: spec FEAT-assignments + đáp án`, push, báo PM ≤ 5 dòng.

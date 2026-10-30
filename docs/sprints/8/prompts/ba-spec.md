# Prompt cho `ba`: spec sprint 8 (P6 Sổ điểm)

Làm ở `/tmp/ba-wt`: `git fetch && git switch --detach origin/sprint/8-p6`, commit rồi `git push origin HEAD:sprint/8-p6` (pull --rebase trước khi push).

## Đọc
- `docs/team/CONTEXT.md`, `docs/sprints/8/plan.md` (11 story, quyết định PM, vùng tệp BA mở rộng).
- `docs/phases/P6.md`; `docs/PRD.md` M7 + mục quyền + phi chức năng; `docs/FLOWS.md` F10 (cả nhánh "sửa sau khi chốt"), F14 (nguồn việc `GRADE_SCHEME_UNCONFIRMED` 70); `docs/DEMO_SCRIPT.md` bước 6 + mục 6 (P6).
- `docs/ARCHITECTURE.md`: `00016 gradebook`, route sổ điểm, `internal/grade`, `internal/grading`; `docs/SYSTEM_DESIGN.md` (job nền, không cache điểm).
- `docs/DECISIONS.md` (luật 5: decimal, không LLM tính điểm; mọi quyết định về điểm), `docs/design/DESIGN.md` §13, §14.11 (`GradeSchemeReview`, `GradeCalculation`), `docs/UX.md`.
- Spec đã có, chỉ đọc: `FEAT-weekly-exam` (điểm bài thi đã công bố), `FEAT-crm-360` (phát biểu, at-risk neo `GRADE`), `FEAT-docs-calendar` (`COURSE_POLICY`, ingest), `FEAT-llm-gateway` (`llm.Structured`, task, làn), `FEAT-escalation-notify` (`AUTO_NO_FORMULA`).
- `seed/documents/Quyche.pdf` (quy chế lớp 1): đọc kỹ thành phần, trọng số, điểm cộng, trần, vắng, làm tròn, thang chữ, điều kiện liệt.

## Việc
1. `docs/specs/FEAT-gradebook/{US,SRS,QUESTIONS}.md`: US-P6-01…09 và US-S8-SEED (US-S8-UI không cần spec sản phẩm; ghi 3 dòng AC ở cuối US.md: không câu mô tả dưới tiêu đề ở `/settings/llm`, `/settings/integrations`, khối bảo mật `/settings`). Mỗi story ≥ 1 AC nhánh lỗi, ≥ 1 AC phân quyền, AC kiểm được.
2. **Đáp án độc lập (vùng tệp mở rộng):**
   - `docs/specs/FEAT-gradebook/fixtures/inputs-lop1.csv`: đầu vào điểm cho 30 sinh viên lớp 1 (MSSV seed, giữa kỳ, cuối kỳ, điểm bài thi PE lấy từ `seed/expected_exam_scores.csv` nếu có, số lần phát biểu, số buổi vắng từ `seed/expected_crm.csv` / `crm_plan.json`), gồm cố ý các ca biên: x,25 / x,75 / x,5 khi làm tròn, chạm trần điểm cộng, ngưỡng vắng, điều kiện liệt, sinh viên A, B, C của kịch bản.
   - `seed/expected_final_grades.csv`: điểm cuối kỳ tính **tay / bảng tính** theo quy chế (không dùng mã của dev), cột giải trình từng thành phần. Ghi cách tính ở `fixtures/README.md`.
   - `fixtures/quy-che-lop2.md`: quy chế lớp 2 ngắn (1–2 trang), **cố ý thiếu quy tắc làm tròn** (DEMO_SCRIPT bước 6); dev dựng PDF từ nội dung này.
3. Bắt buộc có trong spec:
   - **AI chỉ trích nháp:** `ExtractGradeScheme` qua `llm.Structured`, mỗi mục có trang trích dẫn; bộ kiểm bằng code thêm `unclear_items`; mục "chưa rõ" chặn xác nhận; chỉ TEACHER `Xác nhận công thức` và `Chốt điểm` (TA/ADMIN 403); không job / agent nào làm thay.
   - **Grade Engine:** decimal, không `float64`; mọi quy tắc đọc từ `grade_rules.params`; giải trình từng sinh viên có `source_quote`; test 30/30 khớp CSV; test kiến trúc không import `activity` / `crm` (thời gian học) / LLM.
   - **Nguồn điểm:** điểm bài thi PE đã công bố, phát biểu P5, nhập XLSX (ánh xạ cột, xem trước, dòng lỗi, idempotent), sửa ô; bài tập P7 chưa có.
   - **Chốt / mở khoá:** snapshot bất biến, `version`, `unlock_reason` bắt buộc, sinh viên bị đổi điểm được báo, audit.
   - **Sinh viên:** chỉ thấy điểm của mình sau khi chốt / công bố theo quy chế; không thấy điểm nháp; what-if không lưu; tool `get_my_grade_summary`, `what_if_final_grade` danh tính từ JWT.
   - **Nhắc:** banner, khoá Chốt điểm kèm lý do, job tuần, `GRADE_SCHEME_UNCONFIRMED` (70), câu trả lời AI "chưa có công thức chính thức" + tự mở ticket (đã có `AUTO_NO_FORMULA`).
   - **Dòng ghi chú cố định** "điểm chính thức nằm ở hệ thống quản lý đào tạo của trường" trên `/gradebook`, `/me`, file xuất.
   - **Thay mock (D51):** `/gradebook`, `/gradebook/scheme`.
   - **Migration** `00016_gradebook`; ghi rõ mọi `ALTER` bảng cũ nếu cần (đề xuất qua `docs/sprints/8/proposals.md`).
4. `QUESTIONS.md`: câu nào đụng hành vi sản phẩm / quyền / dữ liệu cá nhân → **[CHỦ DỰ ÁN]** kèm phương án mặc định + hệ quả. Câu đụng **cách tính điểm** mà P6.md không nêu → đánh **[ĐIỂM]** (PM bắt buộc hỏi chủ dự án).

Không mở subagent. Chỉ sửa `docs/specs/FEAT-gradebook/**`, `seed/expected_final_grades.csv` và dòng của bạn ở `docs/sprints/8/proposals.md`. Xong: commit `sprint 8: spec FEAT-gradebook + đáp án`, push, báo PM ≤ 5 dòng (AC mỗi story, số câu [CHỦ DỰ ÁN] / [ĐIỂM]).

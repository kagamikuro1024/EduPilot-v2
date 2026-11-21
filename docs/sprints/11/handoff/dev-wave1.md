# Dev · sprint 11 đợt 1 (nhánh `sprint/11-p10-eval`)

## Gộp nhánh
`origin/main` (b97e1afd) + `wip/11-wave1` (b06bbd64) vào `sprint/11-p10-eval`. Xung đột 8 tệp: giữ cả hai phía (`seed-llm-audit` + `seed-guidance`; `observabilityMessage` + `decisionMessage` + `guidanceMessage`; cột `id` + `cascade_gate/cascade_path` ở `InsertLLMAudit`; hai hằng thông báo). Migration quan sát AI `00020_llm_request_payloads` → **`00022`** (`00024` dành cho 10.8). Màn `/observability` THẬT dựng lại trên khung 10.7 (`Section` luôn panel, `Page kind`, `meta` mảng); `DecisionTable` + `FailedMail` giữ trong `AdminView`.
Sửa test: `store/llm_test` (6 bảng `llm_*`, chỉ mục `llm_audit_status_created_idx`), `ObsCourseQuality` có `-- hidden-ok`, `migrations_test` version 22.

## Lỗi thật tìm thấy khi chạy test DB của đợt 1 (WIP ghi «chưa chạy test DB»)
`GET /courses/{id}/observability/summary` với ADMIN và uuid lớp không tồn tại trả **200** (số 0) — đúng ra 404 như mọi route lớp. Sửa ở `observability.CourseSummary` (`CourseExists` → `ErrNotFound`) + handler.
Hợp đồng: 4 API quan sát AI vào `contract` (`scenarios_observability_test.go`, thao tác 319 → 323, ma trận bảo mật), thêm mã lỗi `PAYLOAD_EXPIRED`, `REASON_REQUIRED` vào enum `Error.code`.

## Việc đã chốt trong đợt (proposals)
| # | Việc | Ở đâu |
| --- | --- | --- |
| P-11-13 | hạn seed 23:59 giờ Việt Nam: `seed-age` lùi BT03 về 23:59 gần nhất (mọi mốc cùng lùi một khoảng ⇒ «trễ 2 giây» giữ nguyên), BT04 `at2359`, `check-assignments-seed.mjs` kiểm | `seedtool.AgeAssignment`/`DueTarget`, `scripts/seed.mjs`, test `TestSeedAgeAssignment` |
| P-11-14 | route `/assignments` (STUDENT): ba nhóm, một hành động mỗi hàng; nav «Bài tập» giữa «Hỏi đáp công khai» và «Luyện đề», đầu «Thêm» ở điện thoại; Staff → màn chặn không gọi API | `features/assignments/StudentAssignmentList.tsx`, `shared/shell/nav.ts`, `e2e/assignments-list.spec.ts` |
| P-11-15 | DTO sinh viên bỏ `expr`, thêm `text` + `detail` do Go dựng từ CÙNG số với `expr` (staff `explain` giữ `expr`); bản chốt cũ không có `text` → «nhãn: giá trị» | `grade/engine.go` (`Line.Text/Detail`), `me.go`, `finalize.go`; test `TestLineTextSameNumbersNoCode` (30 SV + ca tổng hợp: tập số ⊆ expr/value/nhãn, không `min(`/`max(`/`÷`/ngoặc); `/me` hiện câu, «Chi tiết» mở danh sách |
| P-11-26 | 7 mẫu cờ chèn lệnh, 0 cờ nhầm trên 91 bài sạch; sửa đo R3; đính chính báo cáo 9.5 | `dev-P-11-26.md` |
| P-11-25(1)/P-11-27/28 | decision chat 429, THREADS_PRECHECK 3,6 s = DECIDE 0,45 s + CLASSIFY 3,4 s (add-only không FAST được bài sạch); PM chọn (A) giữ tắt; **(D) cho decision tự cho qua khi «no» đủ chắc = nợ sau bảo vệ** | `docs/sprints/9.5/report.md` mục 4 |
| P-11-25(3) | R1 `not_found` là lỗi phép đo (docling dựng «- » đầu mục) | `benchmarks/real-llm/real_llm.py` |

## Lệch spec cần BA biết
- SRS FEAT-gradebook 4.8.2a ghi «trung bình {n} bài **theo trọng số**» và detail «… · trọng số {w}»: engine tính trung bình ĐỀU các bài của thành phần (không có trọng số theo bài). Giao diện ghi «trung bình {n} bài» và detail «{bài} {điểm}». BA vá câu chữ (hoặc cho biết nếu cần trọng số thật).
- 4.8.2a `HP_RAW` ghi «mọi thành phần»; engine gộp theo nhóm Quá trình / Cuối kỳ, nên câu là «Điểm học phần thô v = QT × w% + CK × w%».
- SRS FEAT-assignments 4.6.3 + `fixtures/build_fixtures.py` `INJECTION`: thêm 7 mẫu (xem `dev-P-11-26.md`).

## Chưa làm (đợt 1)
Ảnh mốc visual `observability-1440/390` (màn thật) — lấy từ artifact CI rồi commit (Mac lệch Linux). Chạy lại R1 sau khi sửa script. D-R-02/04/08, P-11-19, P-11-20: theo kế hoạch đợt sau.

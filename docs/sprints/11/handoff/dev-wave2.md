# Dev · sprint 11 đợt 2 (nhánh `sprint/11-p10-eval`)

## Đã làm
| Story | Nội dung | Ở đâu |
| --- | --- | --- |
| US-P10-04 | `POST …/insights/knowledge-gap/generate`, `GET …/latest`, `GET …/{rid}/export?format=md`. Việc nền `insight.knowledge_gap` (BATCH): COLLECT → CLUSTER → NAME → DONE (SSE `job.progress`, `detail.stage`). Xếp hạng bằng code (`rank.go`, công thức SRS 4.1, nguồn thiếu chia đều trọng số), LLM chỉ đặt tên + khuyến nghị (1 lời gọi `Structured`, lỗi → tên từ từ khoá, `named_by:"fallback"`). Chặn trùng: so `data_watermark` cùng `period_days` (`reused`), ≥ 20 câu (409 `NOT_ENOUGH_DATA {count,required}`), 3 việc / giờ / lớp (Redis, `reused` không tính), hai việc đồng thời cùng lớp + kỳ chỉ ghi một báo cáo (khoá tư vấn Postgres) | `internal/insight`, `internal/httpapi/insighthttp`, `store/queries/insight.sql`, `cmd/worker/registry.go` |
| US-P10-05 | `/insights` thật (bỏ `@/mock`): danh sách chủ đề (8 đầu + «Chủ đề khác»), câu mẫu gập, «Ít người hỏi», Tài liệu chưa đề cập, so với báo cáo trước, tạo báo cáo (chỉ báo giai đoạn), thread ghim + buổi ôn tập **mở tại chỗ** (không `Dialog`: `ui-antipatterns` cấm), xuất Markdown / PDF (`window.print`) | `features/insights/*`, `e2e/insights.spec.ts` (12 ca) |
| US-P10-06 | 5 API thống kê điểm thi (`overview`, `trend`, `hard-items`, `declining`, `compare`); tab «Điểm thi» ở `/analytics` | `exam/analytics.go`, `examhttp/analytics.go`, `features/analytics/ExamStats.tsx`, `e2e/analytics-exam.spec.ts` |
| US-P10-07 | Seed B: lớp 1 chủ đề D đúng 2 SV (đổi chia câu hỏi trong `seed.mjs`: A 85, B 45, D 10, ngoài 10), lớp 2 chủ đề C 40 câu, 4 bài PE `PUBLISHED` quá khứ + 2 SV giảm 3 bài liên tiếp (`seed-exam-history`, ghi thẳng DB); Seed A bổ sung dòng `degraded` + 1 thư DEAD; `scripts/check-p10-seed.mjs` | `seedtool/examhistory.go`, `seedtool/llmaudit.go`, `scripts/seed.mjs`, `scripts/chat-seed-data.mjs` |
| US-P10-14 (P-11-30) | `GET /admin/audit` (lọc entity / actor / khoảng, cursor có dấu vân tay bộ lọc, `changed_fields` không có giá trị, ghi `VIEW_LIST` cùng giao dịch), migration `00025_audit_log_indexes`, màn `/admin/audit` (nav «Nhật ký») | `internal/audit`, `audithttp`, `features/admin/AdminAudit.tsx`, `e2e/audit.spec.ts` |

Hợp đồng: `openapi.yaml` 323 → 332 thao tác; `contract` có `insightScenarios`, `examAnalyticsScenarios`, audit trong `observabilityScenarios`; thêm ngoại lệ `text/markdown` ở bộ giải mã (nội dung do `TestKnowledgeGap` kiểm).

## Quyết định cần BA / QC biết (spec mơ hồ, đã chọn)
1. **`period_days`** ∈ {7, 14, 30, **120**}, mặc định 14 (Q-E-7). «Toàn kỳ» = **120** vì schema lớp không có ngày bắt đầu / kết thúc kỳ. 0 / âm / giá trị khác → 422. TC-04-04 gửi «toàn kỳ» bằng `120`.
2. **`signals`** của chủ đề: `{students, questions, low_confidence_rate, not_helpful_rate, practice_wrong_rate}` (= S, Q, L, N, P). `practice_wrong_rate` = **null** khi lớp chưa có lượt luyện đề (TC-04-07). `score` ghi kèm để đối chiếu công thức §4.1.
3. **Gom cụm**: tham lam theo cosine ≥ `Service.Tau` (mặc định 0,80). Hiệu chỉnh cùng P-11-19 khi đo embedding thật (embedding giả của ep-qc không có ngữ nghĩa).
4. **Ngưỡng đạt** thống kê điểm thi = 50 % điểm tối đa của bài (bài thi chưa có cấu hình ngưỡng). `week` nhận `YYYY-Www` hoặc một ngày `YYYY-MM-DD` (thứ Hai → Chủ nhật, +07). `overview` có cả `exam_id` và `week` → 422; không có cả hai → bài `PUBLISHED` mới nhất.
5. **`declining`**: chuỗi giảm **nghiêm ngặt** ở 3 bài `PUBLISHED` **liền nhau** của lớp (sinh viên vắng bài giữa → đứt chuỗi); mỗi người một dòng = bộ ba mới nhất. Không ghi `student_risk`.
6. **`compare`**: chỉ TEACHER (JWT); ADMIN / TA / STUDENT 403 (Q-S11-10 chọn 403). Lớp < 5 bài nộp: `enough:false`, không số.
7. **`Cache-Control: private, no-cache`** cho 5 API điểm thi (TC-06-22); không cache phía máy chủ.
8. Không dùng `recharts` (chưa có trong `package.json`): histogram là cột CSS, xu hướng là `TrendChart` sẵn có. Thêm `recharts` cần PM duyệt.
9. Biên trên khoảng thời gian của báo cáo = `now + 1 phút` (đồng hồ Postgres trong Colima chạy nhanh hơn máy chủ vài giây → câu vừa gửi rơi ngoài kỳ).

## Sửa lỗi hạ tầng CI (đợt 1, đã gộp)
Nav SV 8 → 9 mục (`/assignments`, P-11-14): `shell.spec`, `account.spec` (+ palette), nav admin 6 → 7 (`/admin/audit`). `FailedMail` dùng `Section fault` (alert đúng bề rộng cột). `TestObservabilityDetailNoPII` thiếu `LLM_PROVIDER=fake` ở rig; `TestAlertDedup` khoá khử trùng trùng giây. Bốn test tĩnh: `InsightWatermark` lọc thread ẩn; `InsightNearestChunk` thêm vào danh sách đường đọc `content_chunks` (`course_guard_test`) và loại tài liệu tri thức.

## Chưa làm / nợ
- **Chạy đủ `scripts/seed.mjs` + `check-p10-seed.mjs` trên stack sạch** (cần `down -v`; ep-qc đang giữ cho QC 10.8, và seed lớp 1 đổi cách chia câu nên không áp được lên DB cũ). Đã kiểm: kế hoạch chat bằng `node` (150 / D 10 / ngoài 10 / lớp 2 40), `SeedExamHistory` + `SeedLLMAudit` bằng test Go, SQL của `check-p10-seed.mjs` chạy được trên ep-qc (các ✗ còn lại là do DB cũ).
- Ảnh mốc visual `insights-1440`, `analytics-1440` (đổi màn): lấy từ artifact CI rồi commit.
- Màn `/admin/audit`: bộ lọc «Người» nhận mã người dùng (chưa có ô chọn người).
- E2E thật F16 / F17 trên stack (QC `real-flows.md`).

# Trạng thái slice G_s7_p5b (SmkE.SliceG) — dừng theo lệnh PM

Kết quả đã ghi: `/tmp/qc2e/res/G_s7_p5b.jsonl` (37 TC duy nhất; TC-P5-04-11 ghi hai dòng — lấy dòng sau). Chưa có `/tmp/qc2e/defects/G_s7_p5b.md` (chưa tạo; lỗi: không có DEF mới; TC-20 = F trùng DEF-R7-01).

## Đã xong (id status)
```
TC-01 P
TC-02 P
TC-03 P
TC-04 P
TC-05 P
TC-06 P
TC-10 P
TC-11 P
TC-12 P
TC-13 P
TC-14 P
TC-15 P
TC-19 P
TC-20 F
TC-22 P
TC-21 P
TC-23 P
TC-24 P
TC-25 P
TC-26 P
TC-08 K
TC-16 K
TC-17 K
TC-18 K
TC-09 P
TC-P5-04-11 P
TC-P5-04-32 P
TC-P5-05-29 P
TC-S7-SEED-05 P
TC-S7-SEED-21 P
TC-07 P
TC-P5-04-03 P
TC-P5-04-14 P
TC-P5-04-13 P
TC-P5-04-12 P
TC-P5-04-15 P
TC-P5-04-16 P
```
Ghi chú chính: CI (TC-01…26) chấm trên run PR 43712425198 @be0c0d96 và main 9bcad074 (TRƯỚC); TC-08/16/17/18 = K loại b (cần đẩy nhánh tạm). TC-19 không áp dụng (không có bước kiểm hợp). TC-20 = F (wall 15 m 58 s > 10 m, DEF-R7-01).
Go test đã PASS: TestRiskTable (16 ca), TestSensitiveQueriesOnlyFromCRM/TestStudentNeverSeesRisk (nay ở internal/crm), agent TestAgentHasNoRiskOrObservationAccess + TestStudentAsksAboutOwnNotes/TestNoToolReadsObservations, seedtool.TestSeedCRMExpected, crm.TestSeedIsolation (lần 1 FAIL do tải 27, lần 2 PASS)/TestSeedAdminForbidden.
**Chưa rec nhưng đã có bằng chứng (chỉ cần ghi):** TC-P5-04-35/P5-05-33/P5-06-26/SEED-19 — CI log be0c0d96 (`/tmp/qc2e/G/pwtests.json`) có students/me/activity/today/attendance/escalation spec ✓ desktop (mobile skip theo thiết kế), ui-antipatterns trong job fe-check xanh. TC-P5-04-04/P5-05-07 — v_student_360 không phụ thuộc chat_messages (pg_depend=0), SQL list/profile không đọc chat_messages, dev test EXPLAIN PASS. TC-P5-05-32/P5-06-21 — grep `@/mock` + build/lint xanh ở CI fe-check. TC-P5-04-10/08 — `TestStudents360ExplainNoSeqScan`, `TestStudents360LargeCoursePerf`, `TestProfileUnder3s` PASS (log `/tmp/qc2e/G/perf.full`). TC-P5-05-16/19/21/22 — dev test TestAISummary* PASS (payload không danh tính, Redact, 503, 504, 429); ai-summary dùng task UTILITY thay INSIGHT (BA sửa spec, sprint 7 report). TC-P5-06-19 → status T (#34: nhân viên vào /me thấy màn chặn chung, không redirect). TC-S7-SEED-10 → T (proposals sprint 8 #5: thêm tín hiệu GRADE, WATCH không ghim; hiện AT_RISK 1 (C) + WATCH 3 (sv12, sv19, sv26)).
**Lỗi/FAIL cần chạy lại:** TC-P5-04-09: `TestStudentsListP95Under300ms` FAIL lần 1 (p50 185 ms, p95 622 ms) khi máy tải 27 — cần chạy lại 1 lần khi nhẹ rồi chấm (nếu vẫn > 300 ms → F, DEF-R7-G-01).

## Còn lại (chưa rec)
TC-P5-04-04 TC-P5-04-08 TC-P5-04-09 TC-P5-04-10 TC-P5-04-17 TC-P5-04-18 TC-P5-04-22 TC-P5-04-23 TC-P5-04-24 TC-P5-04-25 TC-P5-04-26 TC-P5-04-27 TC-P5-04-28 TC-P5-04-30 TC-P5-04-31 TC-P5-04-33 TC-P5-04-34 TC-P5-04-35 TC-P5-05-02 TC-P5-05-03 TC-P5-05-04 TC-P5-05-05 TC-P5-05-07 TC-P5-05-12 TC-P5-05-14 TC-P5-05-15 TC-P5-05-16 TC-P5-05-17 TC-P5-05-18 TC-P5-05-19 TC-P5-05-20 TC-P5-05-21 TC-P5-05-22 TC-P5-05-24 TC-P5-05-26 TC-P5-05-27 TC-P5-05-28 TC-P5-05-30 TC-P5-05-31 TC-P5-05-32 TC-P5-05-33 TC-P5-06-03 TC-P5-06-06 TC-P5-06-07 TC-P5-06-08 TC-P5-06-09 TC-P5-06-10 TC-P5-06-11 TC-P5-06-12 TC-P5-06-13 TC-P5-06-14 TC-P5-06-15 TC-P5-06-16 TC-P5-06-17 TC-P5-06-18 TC-P5-06-19 TC-P5-06-20 TC-P5-06-21 TC-P5-06-22 TC-P5-06-23 TC-P5-06-24 TC-P5-06-25 TC-P5-06-26 TC-S7-SEED-01 TC-S7-SEED-02 TC-S7-SEED-03 TC-S7-SEED-04 TC-S7-SEED-06 TC-S7-SEED-07 TC-S7-SEED-08 TC-S7-SEED-09 TC-S7-SEED-10 TC-S7-SEED-11 TC-S7-SEED-12 TC-S7-SEED-13 TC-S7-SEED-14 TC-S7-SEED-15 TC-S7-SEED-16 TC-S7-SEED-17 TC-S7-SEED-18 TC-S7-SEED-19 TC-S7-SEED-20 TC-S7-SEED-22 TC-S7-SEED-23 TC-S7-SEED-24 TC-S7-SEED-25
(phần cắt trong log: xem `/tmp/qc2e/G/rem.txt`)

## Dữ liệu seed đã xác minh (cho SEED-01…03, 06, 07, 24, 25 — chưa rec)
- 761987: attendance_records tạo lúc seed (created_at < 12:30) = 270; buổi 1–9 `attendance_done_at` có; buổi 10 do QC khác điểm danh lúc 13:11 (nay 300 dòng). ABSENT buổi 1–9 = 21. A (sv.gioi) vắng 0; B (sv.kha) vắng 2/muộn 1; C (sv.nguyco) vắng 5/muộn 1 — khớp seed/expected_crm.csv. participation_events lúc seed 65 (A 12, B 2 = 0,50, C 1); nay 67.
- study_time_daily 4200 dòng, 70 ngày distinct; courses: 761987 allow_study_time_opt_out=t, 761988 = f; user_settings track_study_time=false: 0. student_risk 761987: AT_RISK 1 (C, entered_at 12:26:02), WATCH 3; 761988: 24 NONE.
- Tickets 761987 hiện 11 (OPEN 8, CLAIMED 1, ANSWERED 1, CLOSED 1); post_reports 5; knowledge_items 6.
- Chưa chạy: seed.mjs (cấm), SEED-04/16/17/22/23 → đề xuất K loại b (chạm seed chung / dừng gateway) hoặc kiểm bằng đọc mã.
- `/me/today` của teacher bị cắt ở 50 mục (106 mục) nên STUDENT_ATTENTION (bậc 95) không hiện; `GET /courses/{id}/today` trả đủ item "1 sinh viên mới vào diện cần chú ý · 761987" + href `/students?filter=attention&course=…`, khối attention 1 dòng, attention_total 1.

## Thế giới thử đã dựng (cần dọn nếu muốn)
- Users `g7t`, `g7t2` (TEACHER), `g7ta` (TA), `g7s01…g7s14` @edupilot.local (mật khẩu seed). Courses `G7A` (01a1e00d-e964-7d39-864e-500cbd54f3e5, GV g7t, join QKGAZZH), `G7B` (…ecca…, GV g7t2), `G7N` (…eef8…, không dữ liệu, s11 s12) — id trong `/tmp/qc2e/G/courses.json`.
- G7A dữ liệu qua SQL: 4 buổi (10-27, 10-29, 11-03, 11-05), s01 vắng 4/4 AT_RISK, s02 WATCH, s03 tỉ lệ 0,5 AT_RISK (ngưỡng 0,50 → NONE), s04 EXCUSED NONE, s05 chưa ack (STUDY null) AT_RISK, s06 chỉ thấp thời gian học NONE, s07 thiếu tuần W2 WATCH, s08–s12 NONE, s13 PENDING, s14 REMOVED. Settings risk đã trả 0,70.
- Chưa tạo: ghi chú canary, trigger audit, outreach, ai-summary trên stack. Không có trigger hay cấu hình toàn cục nào đang bật. Khoá Redis `ep:crm.risk:done:20261115` đã được worker đặt lại (bình thường).
- Script: `/tmp/qc2e/G/{h,w1,w2,t1,t2,t3}.py` (h.py: AR/ready helper).

## Bước tiếp theo đề xuất
1. rec các ca có bằng chứng ở trên (P5-04-35, P5-05-33, P5-06-26, SEED-19, P5-04-04/05-07/05-32/06-21, 04-08/10, 05-16/19/21/22 (dev test), 06-19 T, SEED-10 T, SEED-01…03, 06…09 theo dữ liệu seed).
2. API G7A: profile/audit (05-02, 05-05, 05-04 trigger WHEN entity_id=s01), sentence 04-17/18, today 04-23/24/25 (dùng g7t, g7t2), 04-33/34 (G7N), crawl sinh viên 04-30, 05-26, 06-06; outreach 05-27; ai-summary 05-22.
3. UI Playwright (qua `/tmp/qc2e/heavy.sh`): /students, /students/[id], /me, /settings, /class/settings.
4. Chạy lại TestStudentsListP95Under300ms; dedupe jsonl; tạo defects/G_s7_p5b.md.

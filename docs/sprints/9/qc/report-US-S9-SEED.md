# QC report — US-S9-SEED (seed sprint 9: Bài tập 03 / 04, `mock-imap`, `mock-graph`, bản ghi LLM giả, quy chế bản 2, cổng P7) · Kết luận vòng 1: **PASS** (BUG-S9-SEED-1 đã sửa trước vòng này; AC9 kiểm bằng đọc mã + bản chạy của dev)
Ngày 2026-11-01. Commit kiểm: `sprint/9-p7` @ `8e478910` (HEAD `61c35b39`). Bộ TC: `tc-US-S9-SEED.md`. Handoff: `dev-US-S9-SEED.md`. CI `8e478910`: **success** (`92718308` đỏ vì `TestSeedGradeAllMatchesExpected` có `DATA RACE` — QC bắt được ở lượt chạy trên `92718308`, dev sửa ở `0a409f71`).

## 1. Cổng đã chạy
| Lệnh | Kết quả |
| --- | --- |
| **Stack dựng từ DB trống ở HEAD** (`dev.mjs up` + `seed.mjs`, cổng host +100; `mock-imap` / `mock-graph` ở 8191 / 8190) | `seed rc 0`; ~6 phút (gồm chờ hạn 75 s) |
| SQL trên DB mới seed | `grading_results` = **0**; `BT03`: `IN_APP MATCHED 13`, `UPLOAD MATCHED 4` + `UNMATCHED 1`, `IMAP MATCHED 6` + `UNMATCHED 1` (+1 thư lạ không gắn bài) = 26 bản ghi; 22 sinh viên MATCHED (sv12 hai phiên bản); `BT03` đã công bố `IMAP+IN_APP+UPLOAD`; `BT04` đã công bố `IN_APP+TEAMS` (kèm `teams_class_id` / `teams_assignment_id`); thẻ IMAP + Teams của lớp 1 đã cấu hình; công thức điểm lớp 1 `CONFIRMED`; `sha256sum -c seed/expected_final_grades.csv.sha256` **OK** (CSV không đổi) |
| `check-assignments-seed.mjs bt03` / `counts` (seed lần hai không tạo đôi) | `OK 22/22 khớp ngay + 3 chưa khớp + 6 không nộp; IN_APP 13 · IMAP 8 · UPLOAD 5`; `counts` không đổi |
| **QC độc lập AC2** (`scripts/p7-seed-grade.py`, không dùng mã dev): `grade-all` BT03 (provider giả `DEMO_MODE`) rồi so **từng ô** `ai_scores`, Σ, trước muộn, trừ muộn, cờ với `fixtures/expected_grading.json` theo `seed-plan.csv` | 22/22: S1 9,00; S2 5,75 (`EVIDENCE_NOT_FOUND`); S3 3,00 (`TOTAL_MISMATCH`); S5 0,00 (`OFF_TOPIC`); S6 4,50 (`INJECTION_SUSPECTED`); S7 (B) 7,00 − 0,50 = **6,50** `NEEDS_REVIEW`; sv08 5,75 − 1,00 (nộp muộn 2 ngày) = 4,75; sv24 (S4 trống) `DONE`, **0 lời gọi**; tổng **42 lời gọi LLM** = 2 × 21 bài có chữ; mọi lời gọi làn `BATCH` |
| `check-assignments-seed.mjs demo5` (DEMO bước 5) | B nộp muộn đang chấm → giảng viên sửa tiêu chí 2 → duyệt → công bố → B thấy **7,50**; DB: `sv.kha PUBLISHED 7.50` |
| `check-assignments-seed.mjs teams` (BT04) | `24 khớp + 1 chưa khớp`; DB `MATCHED 24, UNMATCHED 1` |
| `check-assignments-seed.mjs v2` + QC `p7-hand-calc.py v2` (phân số, từ `seed/expected_final_grades_v2.csv`) | `30/30 OK`; hand-calc `30/30 khớp, 22/30 khác bản 1` |
| `NODE_ENV=production node scripts/seed.mjs` | rc 1, "Seed bị chặn: NODE_ENV=production" |
| **Tự thử đỏ cổng** (TC-P7-SEED-10): đổi `S2` trong bản sao `expected_grading.json` (5,75 → 5,50) | `check-assignments-seed.mjs grade` rc 1 ("11/22 khớp"), `TestSeedGradeAllMatchesExpected` đỏ; đã khôi phục tệp |
| `bash scripts/gate-p7.sh` (QC chạy, stack sạch) | `GATE P7: PASS`, rc 0; 19 dòng PASS (vet, `-race` submission + jobs + grading + assignment, `TestStudentNeverSeesDraftGrade`, `TestImapToDraftGrade`, `TestGradingJobSurvivesWorkerKill`, `TestTeamsSyncAgainstMockGraph`, `-race Teams`, 9 `TestSeed*`, `TestExtractLop1V2MatchesExpected`, contract, sha256, Playwright `assignment-lifecycle` + `grade-appeal`, `ui-antipatterns`, 6 mode `check-assignments-seed`) + khối "Bạn tự kiểm" |
| Playwright `assignment-lifecycle.spec.ts -g 'demo step 5'` | xanh |
| `go test -race` `TestSeed*` (9) + `TestExtractLop1V2MatchesExpected` + `TestDemoModeRecordingsCoverPolicies` + `TestSeedAgeAssignment` + `TestMockImap*` (2) | xanh ở HEAD (lượt `go-test-all.sh`), `DATA RACE` đã hết |

## 2. Kết quả theo AC
| AC | Kết quả | Ghi chú |
| --- | --- | --- |
| AC1 BT03: 26 bản ghi theo `seed-plan.csv` (13 app · 5 ZIP · 8 thư), 3 chưa khớp, `joined_via` đúng | PASS | SQL; `bt03`; `TestSeedImapAuthHeaders`, `TestSeedRosterJoinedVia` |
| AC2 chưa có `grading_results` sau seed; `Chấm tất cả` ra đúng `expected_grading.json` | PASS | QC độc lập 22/22 (bảng trên) |
| AC3 BT04 + `mock-graph`, đồng bộ `happy` 24 + 1 | PASS | `teams`; `TestSeedTeamsHappyCounts` |
| AC4 quy chế bản 2 (PDF đúng chữ, 4 thành phần, 30/30 điểm v2) | PASS | `v2`; hand-calc; `TestExtractLop1V2MatchesExpected`, `TestSeedV2AfterResolvingUnmatched`; CSV gốc không đổi (sha256) |
| AC5 DEMO bước 5 → B thấy 7,50 | PASS | `demo5`; Playwright `demo step 5`; ca `student grading shows no score` giữ nguyên (#16 ACCEPTED) |
| AC6 chỉ HTTP API thật (+ `mock-imap`, `seed-age`) | PASS | đọc `seed.mjs` bước 19 |
| AC7 `gate-p7.sh` đúng lệnh `P7.md` + `-race`, mỗi lệnh một dòng, tự thử đỏ | PASS | bảng trên |
| AC8 chạy lại không tạo đôi | PASS | `counts` sau seed lần hai |
| AC9 thiếu `mock-imap` → dừng, nêu tên dịch vụ | PASS (đọc mã + bản chạy của dev) | `waitService` (60 s, `Dịch vụ mock-imap chưa sẵn sàng sau 60 s (url)`); QC **không** chạy lại được trên stack này vì seed lần hai sau `v2` dừng sớm ở bước 16 ("buổi 10 phải là HÔM NAY", dev đã ghi: không hỗ trợ seed lại sau `v2`) |

## 3. Lỗi / ghi nhận
- **BUG-S9-SEED-1 (Trung bình, đã sửa ở `0a409f71`):** ở `92718308`, `TestSeedGradeAllMatchesExpected` có `DATA RACE` ở bộ đếm `s.llmCalls++` (`seed_assignments_test.go:144`, hai lượt chấm chạy song song) → đỏ dưới `-race` (lệnh bắt buộc theo `AGENTS.md`); CI `92718308` đỏ. Dev đổi sang đếm an toàn; QC xác nhận xanh ở HEAD.
- Ghi nhận (thấp): `demo_test.go` đổi `Len(Recordings, 2)` thành đếm theo loại (#15 ACCEPTED — chặt hơn, mỗi bản ghi kiểm schema).
- Ghi nhận: seed lần hai sau khi đã chạy `check … v2` hoặc sau ngày seed không hỗ trợ (dev đã ghi).
- Ghi nhận: BT03 / BT04 `due_at` đặt "bây giờ + 75 s" nên seed mất ~6 phút; chạy `seed-age` để đưa hạn về quá khứ khi cần.

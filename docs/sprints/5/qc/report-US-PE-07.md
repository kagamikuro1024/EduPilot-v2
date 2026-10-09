# Báo cáo QC — US-PE-07 (liêm chính: khoá chat AI trong giờ làm bài, ghi rời tab / dán, so độ giống mã nguồn)
**Kết luận: PASS có điều kiện** (chấm lại ở `a1f00c0`, mã gồm `32a21fd`). B1 (TC-35, ngưỡng cờ #17c) **đã sửa**: chạy lại `q7e.py` → 6 cặp chép điểm 1,000 **đều `flagged`**, cặp nền 0,471 không gắn cờ. Lần chấm đầu ở `dd12672` là FAIL TC-35 vì build trước bản sửa. Điều kiện còn lại: giao diện (TC-18, 27–29, 39) chỉ chấm bằng e2e của dev — thăm dò tay bằng `playwright-cli` gom về lượt cổng PE (F19); TC-20 / 22 (PE-08), TC-47 (chat P3); L1 token lệch 0,039 ghi Q-QC (PM: không FAIL); CI GitHub không chạy (billing).

## Lỗi / lệch
- **B1 (đã sửa).** `dd12672`: lớp 8 bài, 3 chép → 0 cặp gắn cờ (ngưỡng `mean + 3σ` > 1,0). Sửa ở `32a21fd` (`max(SIMILARITY_MIN, min(mean+3σ, SIMILARITY_CAP))`, `SIMILARITY_CAP` 0,9). Chấm lại: `scripts/q7e.py` + `q7f.py`: 14 dòng, **6 cờ** (mọi cặp 1,000), cặp nền 0,471 không cờ; `GET …/similarity?flagged=true` 6 dòng `NEW`; GV có `EXAM_SIMILARITY` ở "Hôm nay", TA / SV không; xử lý hết (`CLEARED` ×6) → việc biến mất sau vài giây (cache vô hiệu theo sự kiện).
- **L1 (TC-31).** Điểm hệ thống vs QC: A~B 1,000 = 1,000; A~A' (đổi thứ tự hàm) hệ thống **0,986**, QC **0,947** (lệch 0,039 > 0,02 của TC); A~C QC 0,051. Cả ba đạt AC7 (≥ 0,95 / ≥ 0,60 / ≤ 0,35); chỉ khác định nghĩa token. Ghi Q-QC, chờ BA; không tính FAIL.
- **L2 (TC-13).** Giá trị `meta` sai kiểu (`chars` âm / chữ / số thực) **không làm mất sự kiện**: dòng `PASTE` vẫn lưu với `meta={}`; khoá lạ (`clipboard`, `ip`) bỏ im lặng. Đúng "bỏ không lỗi".
- **L3 (TC-04).** Gia hạn `closes_at` không kéo dài lượt đã bắt đầu quá `started_at + duration` (deadline 21:43:10 → 21:43:49 = start + 5 phút); `TTL` khoá tăng đúng theo `deadline` mới (270 → 307 s).
- **L4.** Playwright lượt đầu đỏ 241 ca vì Next chết (`ERR_CONNECTION_REFUSED` tại `:3330`, L5 cũ, không phải lỗi mã); chạy lại đủ bộ: **429 pass, 0 fail**.

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01 | PASS | bắt đầu M1 (5 phút): `GET ep:exam_lock:<uid>` = `attempt_id`, `TTL 310` (= 300 + grace 10); `/me/exam-lock` → `{locked:true, until}`; không lộ `attempt_id` |
| 02 | PASS | nộp lượt cuối → khoá gỡ sau **0,1 s**; `/me/exam-lock` → `{locked:false}` |
| 03 | PASS | hai lượt (5 / 9 phút): khoá theo lượt muộn nhất (`TTL 550`, id = M2); nộp M2 → khoá về M1 (`TTL 310`) ngay; nộp M1 → gỡ |
| 04 | PASS (L3) | `extend` `closes_at` +10 phút: `TTL` 270 → 307 s theo `deadline_at` mới |
| 05, 12, 17, 22, 26, 33, 38, 41, 43, 46 | PASS (dev) | `TestLockSetOnStart`, `TestLockTTLEqualsRemaining`, `TestLockClearedOnSubmit`, `TestLockClearedOnClose`, `TestLockMultipleAttemptsLatestDeadline`, `TestLockSweepRemovesOrphans`, `TestExamLockContract`, `TestLockerRedisMissFallsBackDB`, `TestLockerRedisDownUsesDB`, `TestLockerBothDownDeniesChat`, `TestLockerHitLatencyP95`, `TestEvents*`, `TestScoreIgnoresIntegritySignals`, `TestNoPenaltyColumns`, `TestNoAccusatoryCopy`, `TestWinnowing*`, `TestSimilarity*`, `TestIntegrityDataMinimal`, `TestEventsCascadeDelete`, `TestLockFollowsExtend`, `TestLockSurvivesRedisRestart` PASS trong `go test -race -tags integration ./...` (**979 PASS**, 0 FAIL); `TestFlagRelativeThreshold` PASS nhưng kiểm công thức cũ (B1) |
| 06 | PASS (một phần) | route `_test/chat-gate` chỉ ở build `testroutes` (không có trên stack thường); QC dùng `/me/exam-lock` (cùng `Locker`): SV đang thi `true`, sau nộp `false`, SVB `false` |
| 07 | PASS | `DEL ep:exam_lock:<uid>` giữa giờ → `/me/exam-lock` vẫn `locked:true`; khoá nạp lại (`TTL 550`) |
| 08 | PASS | `docker stop redis`: 20 lần `/me/exam-lock` → **20 × `200 locked:true`** (DB); Redis lên lại → khoá nạp lại (`TTL 280`) |
| 09 | PASS (dev) | `TestLockerBothDownDeniesChat`: QC không dừng Postgres (tránh làm hỏng stack) |
| 10 | PASS (một phần) | `TestLockerHitLatencyP95` (dev: 0,3 ms); QC đo 600 lần `/me/exam-lock` qua Caddy+TLS bằng Python: p50 3,6 / p95 7,3 / p99 22,2 ms (gồm bắt tay TLS, không phải đo server) |
| 11 | PASS | `/me/exam-lock`: SV `{locked, until}`; ADMIN, GV `200 {locked:false}`; ẩn danh `401`; không `attempt_id` / tên bài / lớp |
| 13 | PASS | `PASTE{chars:812, clipboard:"SECRET", ip:"1.2.3.4"}` → `204`, DB `meta={"chars":812}`; `TAB_VISIBLE{duration_ms:4000}`, `OFFLINE`, `ONLINE` lưu; `FOO` và `TAB_TAKEOVER` do client **bỏ**; `chars` âm / chữ / 1,5 → `meta={}` (L2); giờ lưu là giờ máy chủ |
| 14 | PASS | 51 sự kiện → `204`, nhận 50 + **một** dòng `WARN "lô sự kiện liêm chính vượt giới hạn" {attempt_id, dropped:1}` (không nội dung); 600 sự kiện → DB **500** dòng, Redis `dropped=159`; `meta` 301 byte → `204` |
| 15 | PASS | người khác `404`; ẩn danh `401`; lượt đã nộp → `204`, không ghi thêm |
| 16 | PASS | canary `SECRET` / `1.2.3.4`: DB 0, log gateway 0; cột `exam_events` = `id, course_id, exam_id, attempt_id, student_id, type, occurred_at, client_at, meta` (không IP / UA / nội dung) |
| 18 | PASS (dev e2e) | `take: integrity events` PASS; QC chưa xem request tay |
| 19 | PASS | `GET …/events?attempt=`: GV `200`; TA, SV chính chủ, SV khác, ADMIN → `403`; tóm tắt `{tab_hidden_count:492, paste_count:5, paste_chars:813, offline_count:1, …}` |
| 20, 22 | KHÔNG KIỂM ĐƯỢC | `TestResultsHideIntegrityFromTA` chờ trang kết quả PE-08 |
| 21 | PASS | quét `exam.Routes()` 57: `GET …/events` Giảng viên; không route nào khác trả `exam_events` cho SV (SV `403`/`404`) |
| 23 | PASS | cột điểm trừ / "gian lận" trong `exam_attempts, exam_answers, exam_events, similarity_reports`: **0** |
| 24 | PASS | `go list -deps`/`grep` trên `score.go`: 0 nhắc `exam_events`/`similarity`; gieo `score_zz.go` (truy vấn `exam_events`) → `TestScoreIgnoresIntegritySignals` **đỏ**; xoá → **xanh** |
| 25 | PASS | `TestNoAccusatoryCopy` PASS |
| 27, 28, 29 | PASS (dev e2e) | `take: integrity notice`, `take: integrity notice gating` PASS; `ui-antipatterns.sh` rc=0 |
| 30 | PASS | `p507-winnow.py`: A~B **1,000** (≥ 0,95); A~C **0,051** (≤ 0,35); A~A' **0,947** (≥ 0,60); `short` bị bỏ (< 30 token); đối xứng; tất định |
| 31 | PASS (L1) | hệ thống A~B 1,000 / A~A' 0,986; QC 1,000 / 0,947; thứ tự cặp `attempt_a < attempt_b`, mỗi bài một `run_id` |
| 32 | PASS (dev) | `TestWinnowingStarterSubtracted`; QC chưa chứng minh bằng cặp có khung |
| 34 | PASS | sau đóng bài job tự chạy: 9 dòng, 1 `run_id`, dấu vết `exam.similarity.queued` = 1; `POST similarity/run` (GV) `202 {job_id}` → `run_id` thứ 2; TA `403`, SV `403`; bài không code `422`; bài chưa đóng: `TestEnqueueSimilarity` |
| 35 | PASS (chấm lại) | lớp 8 bài / 3 chép: 6 cặp 1,000 đều `flagged`; nền 0,471 không (xem B1) |
| 36 | PASS (dev) | `TestSimilarityScale1000` 1.000 bài 0,31 s; QC chưa gieo 1.000 bài |
| 37 | PASS | `go list -deps ./internal/exam/similarity \| grep -c net/http` = **0** |
| 39 | PASS (một phần) | `GET …/similarity` phân trang con trỏ ổn định (không trùng id giữa hai trang); #57 chi tiết GV `200` `{pair, a{language,source,match_lines}, b}`; TA, SV `403`; review: ghi chú 501 → `422`, `state` lạ `422`, TA / SV `403`, `FOLLOW_UP` + ghi chú `200`, `CLEARED` `200` (DB `CLEARED`); giao diện: dev e2e `similarity review` |
| 40 | PASS (chấm lại) | `EXAM_SIMILARITY` chỉ GV thấy (TA, SV không); xử lý hết → biến mất; `TestSimilarityTodayProvider` PASS |
| 42 | PASS | cột `similarity_reports`: `id, course_id, exam_id, problem_id, run_id, submission_a/b, attempt_a/b, score, shared_fingerprints, flagged, algorithm, review_state, reviewed_by, reviewed_at, note, created_at` (không mã nguồn); `exam_events` như TC-16 |
| 44 | PASS | khoá bám DB trong 30 s: nộp / gia hạn / mở Redis lại đều đúng ≤ 1 s |
| 45 | PASS | `docker restart redis` giữa giờ: `/me/exam-lock` `locked:true`; khoá còn / nạp lại |
| 47 | KHÔNG KIỂM ĐƯỢC | chat (P3) chưa có |
| 48 | PASS | `go vet`, `golangci-lint`, `sqlc diff` rc=0; `go test -race -tags integration -p 1 -parallel 2 ./...` rc=0 (**979 PASS**, 0 FAIL); `pnpm lint`, `build:gate` rc=0; Playwright (không `@real`, `visual`): **429 pass, 103 skip, 0 fail** (L4) |

## Việc sau
- **QC:** thăm dò tay `/exams/[id]/similarity` và `take` (dải liêm chính, ảnh); TC-20 / 22 sau PE-08; TC-47 sau P3.

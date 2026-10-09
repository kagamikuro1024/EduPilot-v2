# Báo cáo QC — US-PE-08 (đóng, chấm, công bố, kết quả, thống kê, CSV, sửa điểm, override, chấm lại, phúc khảo)
**Kết luận: PASS có điều kiện** — không FAIL. Điều kiện: (1) giao diện kết quả (TC-31–33, 38, 39, 59) và **F19 trọn vẹn (TC-67)** chưa thăm dò tay: PM gom về lượt cổng PE bằng `playwright-cli`; (2) việc "Hôm nay" `EXAM_PUBLISH_HOLD` không thấy ở +45 s sau khi đóng (L2, chưa tái hiện chắc); (3) TC-12 (làm tròn một lần: 20 lượt có (a) ≠ (b)), TC-20 (`IE` ở lại `GRADING`), TC-44 (RAM khi xuất), TC-61 (grep log), TC-65 (`kill -9` worker, `docker pause` Redis) chỉ có bằng chứng của dev; CI GitHub không chạy (billing). Bản chấm `1891aab` (mã PE-08 = `4b34d59` + `9869209`). Stack thật `EP_PORT_OFFSET=100`; scripts `q8a.py`…`q8g.py` (QC tự viết). Bài `F19`: 12 SV (`sv04…sv15`), 3 câu trắc nghiệm (đơn, nhiều đáp án K = 3, đúng / sai) + 1 câu code 4 test (2 mẫu + 2 ẩn, trọng số 1), hai SV để hết giờ.

## Lỗi / lệch
- **L1 (TC-13 / 25).** Câu `void`: điểm tổng cộng đúng +1 (sv05 6,75 → 7,75 như tính tay) nhưng `items[].earned` của chính câu đó vẫn `0.00` (có `overridden:true`, `max:1.00`) → tổng `earned` từng câu (6,75) ≠ `score` (7,75) trên màn kết quả của SV. Dev ghi "câu void chỉ cờ overridden"; chờ BA có hiển thị `earned` = `max` hay không (Q-QC).
- **L2 (TC-22).** Bài `hold` (khung 300 s, 3 lượt đã chấm): sau đóng 45 s `GV /me/today` **không** có `EXAM_PUBLISH_HOLD`; đặt trạng thái `CLOSED` + `publish_hold` trong DB rồi gọi lại thì có ngay ("Bài thi QC8 hold đã chấm xong nhưng đang hoãn công bố", `urgency:high`, `href` đúng; TA không thấy). Nghi bộ nhớ đệm "Hôm nay" (TTL 60 s) chưa vô hiệu khi bài chuyển `CLOSED` có hoãn. Chưa tái hiện sạch (không có lần gọi `/me/today` của GV trước đó trong 60 s); nhờ dev kiểm tra sự kiện xoá cache lúc đóng.
- **L3 (TC-17).** Công bố đến `closes_at + 10,3 s` (22:41:02,3 → 22:41:12,6) khi còn lượt phải ép nộp (grace 10 s); TC viết "≤ 10 s" tính từ khi hết lượt chưa chấm — chấm theo chữ AC1 ("sau grace"): đạt.
- **L4 (TC-36).** Ô `q` lọc theo họ tên / MSSV, không theo email (`q=sv04` → 0 dòng) — đúng thiết kế (dev: MSSV không làm điều kiện truy vấn).
- **L5.** `GET …/results/{aid}` của SV vắng / id lạ → `404` (dev ghi lệch nhỏ so với AC16 "403": không lộ tồn tại).

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01, 02 | PASS | 2 lượt `sv14`, `sv15` không nộp → khi đóng: `GRADED`, `submit_reason=CLOSED`, `submitted_at = deadline_at`; còn 10 lượt `MANUAL`; `TestCloseForceSubmitsAll` PASS |
| 03, 08 | PASS (dev) | `TestCloseForceSubmitsAll`, `TestCloseBlocksNewWrites`, `TestGradeMCQSync`, `TestGradeCodeWaitsForJudge`… PASS |
| 04 | PASS | trắc nghiệm chấm ngay lúc nộp tay (`GRADED`, `auto_score` có) |
| 05 | PASS | câu code: `points × Σweight_đạt ÷ Σweight`: `ok` 5,00; `small` (qua 3 / 4 test) 3,75; `wa` / `ce` / không nộp 0 — **12 / 12 điểm khớp tính tay của QC** (bảng `q8a.expected`) |
| 06, 20 | PASS (dev) | `TestGradeErrorKeepsGrading`: lượt ở `GRADING`, `EXAM_GRADE_ERROR`; QC chưa dừng judge ở bài này |
| 07 | PASS | không nộp code → `earned 0` (sv08, sv13: tổng 2,00 / 0,00 đúng) |
| 09 | PASS | `MCQ_MULTI` K = 3, 5 lựa chọn, 12 SV cố ý theo từng tổ hợp `(TP,FP)` = (3,0) (2,0) (1,0) (1,1) (0,1) (2,1) (0,0) (3,1) (3,2) (0,2) (1,2) (2,2): `earned` = 3 · max(0, (TP−FP)/3) = 3, 2, 1, 0, 0, 1, 0, 2, 1, 0, 0, 0 — **khớp tuyệt đối**, không điểm âm |
| 10 | PASS (dev) | `ALL_OR_NOTHING`: `TestComputeScoreStores` |
| 11 | PASS | `auto_score` 9,00 / 6,75 / 3,00 / 0,00 / 2,00 / 7,00 / 4,75 / 3,00 / 3,00 / 0,00 / 7,00 / 4,75 đúng tới 0,01; `earned` đủ chữ số (ví dụ `2.000000000000000000000000000`) |
| 12 | PASS (dev) | `TestGradeCompletesWhenSubmissionDone` (8,33 từ `5/3·10/2`); QC chưa gieo 20 lượt (a) ≠ (b) |
| 13 | PASS (L1) | `override void` câu 0: `recomputed:12`, mọi lượt (kể cả sai) +1 điểm đúng bằng `exp + (1 − q0)` (**0 lệch**); mẫu số giữ nguyên (max 10) |
| 14, 15, 16 | PASS (dev) | kẹp, bước, idempotent: `TestComputeScoreStores`, `TestRecomputeIdempotent` |
| 17 | PASS (L3) | `PUBLISHED` sau đóng 10,3 s; `published_at` 1 lần; outbox `exam.published` **1**; thông báo `EXAM_PUBLISHED` **12** (đúng số SV có lượt; SV vắng không nhận) |
| 18, 19, 21, 24 | PASS (dev) | `TestPublishConditions` (8 goroutine → 1 lần), `TestPublishNotifiesAttemptedOnly`, `TestSetHold`; QC chưa lặp 20 lần |
| 22 | PASS (một phần, L2) | bài `hold`: sau đóng + chấm xong **không** tự công bố (`CLOSED`, `publish_hold=t`, 3 lượt `GRADED`); `hold:false` → `PUBLISHED` ngay (**0 s**), outbox **1** |
| 23 | PASS | TA `403`; sai `version` `409`; `hold` sau `PUBLISHED` → `409 EXAM_LOCKED` |
| 25 | PASS | `GET attempts/{aid}/result` (SV): `score:"9.00"` (chuỗi), `score_adjusted:false`, khoá `exam{…,max_score,published_at,reveal_answers}`, `items[{earned,max,correct,mine,answer,explanation,overridden,samples,hidden,…}]`, `appeal`, `appeal_open_until`; test ẩn chỉ `{passed,total}` (không tên / input / expected) |
| 26 | PASS | `reveal_answers` bật: `answer`, `correct` và giải thích canary **có** (đối chứng dương); tắt: `answer = null`, `explanation = null`, vẫn thấy `mine` + `correct`; canary giải thích **0** trong thân |
| 27 | PASS | sau `PUT …/score 8.5`: SV thấy `score 8.50`, `score_adjusted:true`; item `overridden:true` hiện cờ, không lộ `override` |
| 28 | PASS | `hidden:{passed:2,…}` chỉ số; `QC-HID-`, `QC-REF-`, `hid-big` **0** lần trong mọi thân |
| 29 | PASS | trước công bố `409 RESULT_NOT_PUBLISHED`, thân không dữ liệu |
| 30, 35, 37, 41, 45, 48, 51, 54, 57, 58 (G), 64, 66 | PASS (dev) | các test `-run` tương ứng PASS trong `go test -race -tags integration ./...` (**1005 PASS**, 0 FAIL) |
| 31, 32, 33, 38, 39, 59 (UI) | PASS (dev e2e) | `student result` ×3, `results (Giảng viên / TA)`, thống kê + CSV + phúc khảo, 1.000 dòng ảo hoá PASS; QC chưa thăm dò tay / axe |
| 34 | PASS | bảng điểm: 18 SV vắng → hàng `ABSENT`, CSV `ABSENT` ô điểm trống; SV vắng gọi `result` → `404` (L5); `appeal_days=0` → `409 APPEAL_WINDOW_CLOSED` |
| 36 | PASS | `GET …/results`: `progress{not_started:0,in_progress:0,grading:0,graded:12,absent:18}`; sắp xếp `score` đúng thứ tự giảm; `status=GRADED` 12; con trỏ `next_cursor`; `flags` chỉ GV (`{similarity, tab_hidden, paste}`), TA không có `flags` / `integrity` |
| 40 | PASS | `stats`: `mean 4.19` (QC 4,1875 → 4,19), `median 3.88` (3,875 → 3,88), phân bố 10 khoảng đúng (`{0:2, 2:1, 3:3, 4:2, 6:1, 7:2, 9:1}`), `hardest` đúng `qc8-1` 0,08 (chỉ 1 / 12 làm trọn) và `qc8-0` 0,50 |
| 42 | PASS | `results.csv`: 3 byte đầu `ef bb bf`; ngăn cột `;`; số dấu phẩy (`7,00`); 31 dòng (12 + 18 vắng + tiêu đề); tiêu đề `mssv;ho_ten;trang_thai;diem_tu_dong;diem_chinh_thuc;nop_luc;ly_do_nop;cau_1…` |
| 43 | PASS | họ tên `=HYPERLINK("http://x")`, `+84 123`, `@SUM(1)` → xuất `'=HYPERLINK(…)`, `'+84 123`, `'@SUM(1)` (tiền tố `'`); `-1` kiểm bằng `TestCSVFormulaInjection` |
| 44 | PASS (một phần) | GV, TA `200`; SV, ADMIN `403`; RAM khi xuất chưa đo; > 5.000 dòng `TestResultsCSVTooLarge` |
| 46 | PASS | `score` 10,01 / −1 / 8,505 → `422`; thiếu lý do, lý do 501 ký tự → `422`; sai `version` → `409 VERSION_CONFLICT`; TA `403`; hợp lệ: `adjusted_score 8.50`, `auto_score 9.00` **giữ**, `adjusted_reason`, `version` 2 → 3; `score:null` gỡ → `adjusted_score` NULL |
| 47 | PASS | sau công bố: SV thấy 8,50 + `score_adjusted:true`; thông báo `EXAM_REGRADED` được tạo (12 sau override + sửa điểm) |
| 49, 50 | PASS | override: TA `403`; `void` → tính lại **12** lượt (đồng bộ ≤ 200), không job; điểm khớp tính tay; `override` không lộ ngoài cờ `overridden` |
| 52 | PASS | thêm test ẩn mới → `regrade scope=all` `202 {job_id}`; gọi lại cùng khoá → cùng `job_id`; sau ~25 s điểm **giảm** đúng cho 3 SV `small`: 7,75 → 7,00; 4,75 → 4,00; 5,75 → 5,00 (3/5 thay vì 3/4 test); mọi SV khác giữ; bài về `PUBLISHED`, `regrading=f`; không `regrade` khi thiếu khoá `422`; TA, SV `403`; `scope` lạ `422` |
| 53 | PASS (một phần) | phạm vi `item` / `attempt` / `errors`: `TestRegradeFlow`, `TestRegradeIdempotent` (QC chỉ chạy `all`) |
| 55 | PASS | `reason` rỗng / 1.001 → `422`; tạo `201 OPEN`; lần hai `409 APPEAL_EXISTS`; SV vắng `404`, ADMIN / GV `403`; `appeal_days=0` → `409 APPEAL_WINDOW_CLOSED` |
| 56 | PASS | `llm_audit` = 0 sau phúc khảo; `internal/exam` chỉ `suggest.go` thực sự import `internal/llm` (`score.go` chỉ nhắc trong chú thích) |
| 58 | PASS | TA `403`; thiếu `response` `422`; `ADJUSTED` thiếu / vượt `score` `422`; sai `version` `409`; `ADJUSTED` `4.00`: `adjusted_score 4.00`, `exam_appeals` `score_before 3.00`, `score_after 4.00`; lần hai `409`; SV thấy 4,00 + phản hồi |
| 59 | PASS | `GET …/appeals` GV, TA `200`; SV, ADMIN `403` |
| 60 | PASS | QC quét tay `exam`, `attempts/mine`, `attempts/{aid}/result`, `/me/exam-lock`, `/me/today`, `…/exams` ở hai bài (reveal bật / tắt): 0 byte `QC-HID-` / `QC-REF-` / `hid-big`; canary giải thích **chỉ** ở `result` khi `PUBLISHED` + `reveal_answers`; ma trận dev `leak_matrix 20x6 clean (221 lời gọi)` PASS |
| 61 | PASS (dev) | grep log: QC chưa chạy `docker compose logs`; dev: log rig `io.Discard` |
| 62 | PASS | `TestNoAnswerLeak` in `leak_matrix 20x6 clean` |
| 63 | PASS | bảng điểm / thống kê / CSV / phúc khảo: GV 200, TA 200, SV 403, SV ngoài lớp 403, ADMIN 403; `TestResultsPermissionMatrix` PASS |
| 65 | PASS (dev) | `TestPublishFlowConcurrency` (`-race`), `TestPublishFlowWorkerKilled`; QC chưa `kill -9` / `docker pause` |
| 67 | CHƯA CHẠY | F19 trọn vẹn bằng UI + ảnh: gom về lượt cổng PE (PM) |
| 68 | PASS | `go vet`, `golangci-lint`, `sqlc diff` rc=0; `go test -race -tags integration -p 1 -parallel 2 ./...` rc=0 (**1005 PASS**, 0 FAIL); `pnpm lint`, `build:gate` rc=0; Playwright (không `@real`, `visual`): **443 pass, 103 skip, 0 fail**; `audit-login.mjs` chưa chạy ở story này |

## Việc sau
- **Dev:** L2 (vô hiệu cache "Hôm nay" khi bài `CLOSED` có hoãn); L1 (`earned` của câu `void`) nếu BA muốn đồng nhất.
- **QC:** cổng PE: F19 bằng `playwright-cli` + ảnh (cả UI các story trước), `gate-PE.md`, k6 PE-09; TC-12, 20, 44, 61, 65. Scripts: `scripts/q8a.py` … `q8g.py`.

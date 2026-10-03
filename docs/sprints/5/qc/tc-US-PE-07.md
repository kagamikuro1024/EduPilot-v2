# QC test case — US-PE-07 (liêm chính cơ bản: khoá chat trong giờ thi, log rời tab / dán, so độ giống mã — không giám thị, không tự trừ điểm)
Nguồn: `docs/specs/FEAT-weekly-exam/US.md` US-PE-07 AC1–AC11 + `SRS.md` 4.7.1–4.7.4 (khoá `ep:exam_lock`, `exam.Locker`, sự kiện, winnowing), 5.12–5.13, 6.2 #38, #43, #51–#54, T1 (`/_test/chat-gate`). D56. Research mục 4. Hộp đen; stack như `tc-US-PE-02.md` (+ Redis; gateway `testroutes` cho `/_test/chat-gate`).

Tiền điều kiện chung: bài thi `E` đang `OPEN`; `SVA`, `SVB`, `SVC`; GV (TEACHER), TA, ADMIN; `UIDA` = id của SVA. Mẫu mã so độ giống do QC tự viết: `docs/sprints/5/qc/scripts/p507-src/` (A gốc, B đổi tên + định dạng, C thuật toán khác, A' đổi thứ tự hàm, `short.cpp`). QC **tự cài** winnowing bằng Python (`scripts/p507-winnow.py`: tách từ vựng C/C++, chuẩn hoá `I`/`N`/`S`, k=5, w=4, FNV-1a 64 bit, Jaccard, trừ `starter_code`) để đối chiếu — không dùng mã của dev. Công cụ: **S** shell, **R** Redis, **D** SQL, **A** Chrome, **G** `go test`, **P** Python.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-PE07-01 | AC1 (đặt khoá) | SVA bắt đầu lượt | **R** `GET ep:exam_lock:$UIDA`; `TTL`; **D** `deadline_at` | giá trị = `attempt_id`; `TTL` ≈ `deadline_at − now + EXAM_GRACE_SECONDS` (±2 s); không bao giờ `-1`; mỗi người **một** khoá |
| TC-PE07-02 | AC1 (gỡ khoá) | – | **S** nộp tay; hoặc hết giờ; hoặc bài đóng; đo thời gian tới khi `redis-cli exists` = 0 | gỡ ≤ **10 s** sau khi **không còn** lượt `IN_PROGRESS`; nộp một trong hai lượt thì khoá **còn** (xem TC-PE07-03) |
| TC-PE07-03 | AC1 (nhiều lượt) | SVA thuộc 2 lớp, 2 bài mở cùng lúc, `deadline_at` khác nhau | **R** `TTL` khi có lượt 1 (hạn 20 phút), lượt 2 (hạn 40 phút); nộp lượt 2; nộp lượt 1 | khoá lấy lượt có `deadline_at` **muộn nhất**; nộp lượt muộn nhất → TTL co về lượt còn lại; gỡ khi hết lượt |
| TC-PE07-04 | AC1 (gia hạn, làm tiếp) | – | **S** `extend` +10 phút; làm tiếp từ thiết bị khác | `TTL` dài ra theo `deadline_at` mới ≤ 10 s; làm tiếp không đổi khoá sai |
| TC-PE07-05 | AC1 | – | **G** `-tags integration -run 'TestLockSetOnStart|TestLockTTLEqualsRemaining|TestLockClearedOnSubmit|TestLockClearedOnClose|TestLockMultipleAttemptsLatestDeadline' -v` | `ok` |
| TC-PE07-06 | AC2 (`Locker`) | SVA đang làm bài | **S** `curl $GW/api/v1/_test/chat-gate -H "$SVA"`; nộp bài; lại; SVB (không làm bài) | `{allowed:false}` khi đang làm; `{allowed:true}` sau nộp; SVB `true`; route thử chỉ ở `testroutes` (build thường `404`) |
| TC-PE07-07 | AC2 (nguồn sự thật DB) | – | **R** `DEL ep:exam_lock:$UIDA` khi lượt `IN_PROGRESS`; gọi `chat-gate`; đọc lại khoá | vẫn `allowed:false` (tra DB); Redis được nạp lại khoá |
| TC-PE07-08 | AC2 (Redis chết) | – | **S** `docker stop <redis>` giữa giờ thi; `chat-gate` ×20; bật Redis | `allowed:false` nhờ DB (không `error`, không `true` sai); sau Redis lên lại khoá nạp lại ở yêu cầu kế |
| TC-PE07-09 | AC2 (cả hai chết) | – | **S** dừng DB (và Redis) | `chat-gate` trả lỗi (`5xx`/`SERVICE_UNAVAILABLE`), **không** `allowed:true`; (người gọi phải từ chối chat) |
| TC-PE07-10 | AC2 (hiệu năng) | – | **S** 2.000 lần `IsLocked` qua `chat-gate` khi Redis trúng (đo p95 phía server qua `Server-Timing`/log hoặc test Go riêng của QC) | p95 ≤ **5 ms** (ghi số đo và cấu hình) |
| TC-PE07-11 | AC2 (`/me/exam-lock`) | – | **S** `GET /me/exam-lock` bằng SVA đang làm, SVA sau nộp, ADMIN, GV, người không đăng nhập; đọc thân | `{locked:true, until:RFC3339}` / `{locked:false}`; **không** lộ tên bài / lớp / `attempt_id`; `401` khi không token; chỉ của chính mình (không có tham số `user_id`) |
| TC-PE07-12 | AC2 | – | **G** `-tags integration -run 'TestExamLockContract|TestLockerRedisMissFallsBackDB|TestLockerRedisDownUsesDB|TestLockerBothDownDeniesChat' -v` | `ok` |
| TC-PE07-13 | AC3 (nhận sự kiện) | lượt đang làm | **S** `POST …/attempts/{aid}/events` với `PASTE {chars:812, clipboard:"SECRET", ip:"1.2.3.4"}`; `TAB_HIDDEN`, `TAB_VISIBLE {duration_ms}`, `OFFLINE`, `ONLINE`; loại lạ `FOO`; `TAB_TAKEOVER` do client gửi; **D** `exam_events` | `204`; `meta` chỉ khoá `duration_ms,chars,item_id` (`{"chars":812}`) — `clipboard`, `ip` **bị bỏ không lỗi**; `occurred_at` = giờ máy chủ (không tin `client_at` lệch giờ); loại lạ / `TAB_TAKEOVER`/`CHAT_BLOCKED` từ client → bỏ hoặc `422` (ghi lại hành vi) |
| TC-PE07-14 | AC3 (giới hạn) | – | **S** 51 sự kiện / yêu cầu; 600 sự kiện tổng; `meta` 301 byte; giá trị không phải số | 51 → xử lý theo spec (cắt ≤ 50 hoặc `422`: ghi lại); vượt 500 / lượt → `204` nhưng phần dư bỏ (đếm `dropped`); `meta` > 300 byte bị bỏ; số lạ bị bỏ |
| TC-PE07-15 | AC3 (không đổi lượt) | – | **S** sự kiện khi lượt đã nộp; của lượt người khác; ghi sự kiện lỗi (DB phạt) lúc lưu bài | `204` bỏ qua; `404` (người khác); lưu bài **không** thất bại vì sự kiện lỗi |
| TC-PE07-16 | AC3 (nội dung) | canary `QC-PASTE-<uuid>` trong `meta` và clipboard | **D** `pg_dump`; log gateway/worker; Redis | canary **0** lần |
| TC-PE07-17 | AC3 | – | **G** `-run 'TestEventsWhitelistMeta|TestEventsNoContentStored|TestEventsCapPerAttempt|TestEventsBatchLimit|TestEventsNeverBreakSave' -v` | `ok` |
| TC-PE07-18 | AC3 (máy khách gửi gộp) | – | **A** rời tab (`visibilitychange`), dán vào ô mã và `textarea`, offline/online; theo dõi request `…/events` | gửi gộp ~15 s và khi `pagehide` (`sendBeacon`/`fetch keepalive`); `PASTE` chỉ `chars` + `item_id` (xem payload); **không** ký tự đã dán |
| TC-PE07-19 | AC4 (ai xem log) | log của lượt SVA | **S** `GET …/exams/{eid}/events?attempt=<aid>` bằng GV, TA, SVA (của chính mình), SVB, ADMIN, người ngoài lớp, GV lớp khác | GV `200`; TA, SV (kể cả chính mình), ADMIN, ngoài lớp → `403`; GV lớp khác `403`/`404` |
| TC-PE07-20 | AC4 (tóm tắt, TA không thấy) | – | **S** `GET …/results/{aid}` và `GET …/results` bằng GV và TA; đọc khoá | GV: tóm tắt (số lần rời tab, tổng thời gian rời tab, số lần dán + tổng ký tự, offline, `TAB_TAKEOVER`); **TA: không** cột / trường liêm chính (bảng điểm cho TA không có) |
| TC-PE07-21 | AC4 | – | **S** tìm API nào cho SV trả sự kiện: quét 56 route (QC liệt kê) bằng SVA | **0** route trả `exam_events` cho SV |
| TC-PE07-22 | AC4 | – | **G** `-run 'TestEventsTeacherOnlyMatrix|TestResultsHideIntegrityFromTA' -v` | `ok` |
| TC-PE07-23 | AC5 (**không trừ điểm**) | 2 lượt cùng đáp án: SVA sạch, SVB có 100 `TAB_HIDDEN`, 50 `PASTE`, cặp `flagged` | **S** chấm; so điểm | **cùng điểm**; không cột "điểm trừ" / cờ "gian lận" (`select column_name from information_schema.columns where column_name ~* 'penalt|cheat|gian|flag'` chỉ có `similarity_reports.flagged`) |
| TC-PE07-24 | AC5 (tách mã) | – | **S** QC tự dùng `go list -f '{{.Imports}}'` / `grep` trên mọi `score*.go`: có import / truy vấn `exam_events` hoặc `similarity_reports` không; gieo tệp tạm `score_zz.go` truy vấn `exam_events` → test | **0**; gieo → `TestScoreIgnoresIntegritySignals` **đỏ**; xoá → xanh |
| TC-PE07-25 | AC5 (từ cấm) | – | **S** `grep -rniE 'gian lận|vi phạm|nghi gian lận' frontend/src/features/exam* frontend/src/i18n` và quét giao diện GV (`document.body.innerText`) ở tab kết quả / liêm chính | **0**; dùng "tín hiệu để tham khảo" |
| TC-PE07-26 | AC5 | – | **G** `-run 'TestScoreIgnoresIntegritySignals|TestNoPenaltyColumns|TestNoAccusatoryCopy' -v` | `ok` |
| TC-PE07-27 | AC6 (minh bạch) | SV, màn bắt đầu | **A** màn trước khi bắt đầu và dải cố định trong lúc làm | có **nguyên văn**: "Trong giờ làm bài, chat AI tạm khoá. Hệ thống ghi lại số lần bạn rời trang hoặc dán nội dung để giảng viên xem khi cần; đây không phải giám thị và không tự trừ điểm của bạn." + liên kết "Tìm hiểu thêm" (nói ghi gì: loại sự kiện + thời điểm + độ dài dán; không nội dung dán, không camera, không màn hình) |
| TC-PE07-28 | AC6 (nút bắt đầu) | – | **A** `Bắt đầu làm bài` bấm được khi nào; có ô tích không | bấm được sau khi cuộn tới / thấy câu (không bắt tích ô); 0 từ `RAG|PII|provider|trace` |
| TC-PE07-29 | AC6 | – | **S** `$PW exam.spec.ts -g 'integrity notice'`; `bash scripts/ui-antipatterns.sh` | `rc=0` |
| TC-PE07-30 | AC7 (lexer) | – | **P** QC tự đo winnowing trên `p507-src` bằng `p507-winnow.py`: A~B, A~C, A~A', A~short; trừ `starter_code` | A~B ≥ 0,95; A~C ≤ 0,35; A~A' ≥ 0,60; `short` (< 30 token) bị bỏ; kết quả đối xứng (A,B = B,A) và tất định (chạy 3 lần) |
| TC-PE07-31 | AC7 (so với hệ thống) | 4 mẫu nộp qua đường thật bởi 4 SV | **S** job so sánh; **D** `similarity_reports.score` cho các cặp | điểm hệ thống ≈ điểm của QC (lệch ≤ 0,02 vì định nghĩa token; nếu lệch lớn → ghi Q, xem nguyên nhân); thứ tự: A~B > A~A' > A~C |
| TC-PE07-32 | AC7 (trừ khung) | `starter_code` chứa 20 dòng của bài; hai SV chỉ sửa 2 dòng | **S** cặp 2 SV | độ giống **thấp** sau khi trừ khung (so với khi không trừ: QC tính cả hai để chứng minh) |
| TC-PE07-33 | AC7 | – | **G** `-run 'TestWinnowingRenameReformat|TestWinnowingDifferentSolution|TestWinnowingSymmetric|TestWinnowingDeterministic|TestWinnowingStarterSubtracted|TestWinnowingShortSkipped|TestLexerCpp' -v` | `ok` |
| TC-PE07-34 | AC8 (job, đóng bài) | bài `CLOSED`, mọi lượt `GRADED`, 30 lượt | **S** đợi job tự chạy; **D** `similarity_reports`: số cặp, `run_id`, `flagged`; GV `POST …/similarity/run` | tự chạy **một lần** sau khi đóng + chấm xong (outbox `exam.graded_all`); `202 {job_id}` khi chạy lại với `run_id` **mới**, bản cũ giữ; chỉ `SUBMIT` cuối (kể cả `auto`) |
| TC-PE07-35 | AC8 (ngưỡng) | phân phối điểm biết trước (QC gieo bài: 28 bài độc lập + 2 bài copy) | **S** xem `flagged`; QC tự tính `mean + 3×stddev` trên **mọi cặp** của bài | cặp copy `flagged` (≥ `max(0,60, mean+3σ)`); cặp nền ≈ 0,27 **không** flagged; lưu tối đa 200 cặp, `score ≥ 0,40`, ≥ 10 dấu vân tay chung; thứ tự chuẩn `attempt_a < attempt_b` |
| TC-PE07-36 | AC8 (quy mô) | 1.000 bản nộp (QC sinh mã biến thể bằng Python, chèn thẳng DB) | **S** chạy job; đo thời gian; RAM worker | ≤ **30 s**; không OOM; không so O(n²) thô (ghi số cặp được so) |
| TC-PE07-37 | AC8 (mã không rời hệ thống) | – | **S** `go list -deps ./internal/exam/similarity \| grep -c net/http`; chặn mạng của worker (`docker network`) khi chạy job | `0`; job vẫn chạy khi worker không ra internet |
| TC-PE07-38 | AC8 | – | **G** `-tags integration -run 'TestSimilarityJobTopPairs|TestSimilarityFlagRelativeThreshold|TestSimilarityRerunNewRunID|TestSimilarityScale1000' -v` | `ok` |
| TC-PE07-39 | AC9 (xem / xử lý) | cặp `flagged` | **A** GV mở "Nghi giống nhau": bảng cặp (tên hai SV, bài, độ giống, `flagged`); mở cặp: hai mã cạnh nhau, tô phần khớp; `Đã xem — không có vấn đề` (`CLEARED`), `Cần trao đổi` (`FOLLOW_UP`) kèm ghi chú 500 và 501 ký tự | đúng; ghi chú 501 → `422`; dòng cố định "Độ giống chỉ là gợi ý — nhiều bài đúng cùng một cách làm tự nhiên giống nhau. Quyết định là của thầy/cô."; **không** nút "trừ điểm"; mã SV hiển thị như **văn bản** (XSS: mã chứa `<script>`) |
| TC-PE07-40 | AC9 (quyền, "Hôm nay") | – | **S** `PUT …/similarity/{id}/review` bằng TA, SV; `GET /me/today` GV với 3 cặp `NEW flagged` rồi xử lý hết | TA/SV `403`; việc `EXAM_SIMILARITY` bậc 55 "{N} cặp bài code nghi giống nhau · {tên bài thi}" chỉ GV; biến mất khi hết cặp `flagged` `NEW` |
| TC-PE07-41 | AC9 | – | **G** `-run 'TestSimilarityReviewStates|TestSimilarityTeacherOnly|TestSimilarityTodayProvider' -v`; `$PW exam.spec.ts -g 'similarity review'` | `ok`; `rc=0` |
| TC-PE07-42 | AC10 (tối thiểu hoá) | – | **D** cột của `exam_events`, `similarity_reports`; xoá lượt; `pg_dump`/log | `exam_events`: chỉ `type`, thời điểm, `meta` ≤ 300 byte; **không** IP / user-agent / nội dung; `similarity_reports`: chỉ id + số liệu; xoá lượt xoá luôn sự kiện (cascade); log không chứa nội dung sự kiện |
| TC-PE07-43 | AC10 | – | **G** `-run 'TestIntegrityDataMinimal|TestEventsCascadeDelete' -v` | `ok` |
| TC-PE07-44 | AC11 (không lệch pha) | – | **S** gia hạn; làm tiếp thiết bị khác; hết giờ; so `ep:exam_lock` với DB mỗi giây trong 30 s sau mỗi sự kiện | lệch ≤ **10 s**; takeover không đổi khoá; hết giờ mất khoá |
| TC-PE07-45 | AC11 (Redis restart) | – | **S** restart Redis giữa giờ thi (mất khoá); `chat-gate` | `allowed:false` nhờ DB; khoá nạp lại ở yêu cầu kế |
| TC-PE07-46 | AC11 | – | **G** `-tags integration -run 'TestLockFollowsExtend|TestLockSurvivesRedisRestart' -v` | `ok` |
| TC-PE07-47 | chéo P3 | – | **S** khi P3 (chat) có: SVA đang thi hỏi chat → bị từ chối, hết giờ hỏi lại được (kịch bản "Bạn tự kiểm" của `PE.md`); trước P3: chỉ `chat-gate` | (chạy ở cổng PE nếu chat đã có; trước đó chỉ TC-PE07-06) |
| TC-PE07-48 | tổng | – | **S** `go vet ./... && golangci-lint run && go test -race -count=1 ./... && go test -count=1 ./internal/contract/...` | `rc=0` |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| Khoá chat lệch pha (Redis mất khoá / chết / DB chết) | 07–09, 44, 45 |
| Hai lượt cùng lúc ở hai lớp | 03 |
| Nội dung clipboard / IP / user-agent bị ghi | 13, 16, 18, 42 |
| TA / SV / ADMIN đọc log liêm chính | 19–21 |
| Log liêm chính làm tụt điểm / từ ngữ buộc tội | 23–26 |
| So giống: đổi tên biến, thuật toán khác, khung chung, quy mô 1.000 | 30–36 |
| Mã SV ra ngoài (MOSS) | 37 |

## Câu hỏi cho BA / PM
- **Q-QC-PE07-1** — AC3: 51 sự kiện / yêu cầu — cắt còn 50 hay `422`? AC chỉ nói "≤ 50"; QC ghi lại hành vi và chấm theo SRS 4.7.3 nếu có câu rõ. — *chờ BA*.
- **Q-QC-PE07-2** — AC7: định nghĩa số token của QC (bỏ `#include`, giữ `#define`) có thể lệch ≤ 0,02 so với hệ thống; chấp nhận ngưỡng lệch 0,02 khi đối chiếu độc lập (TC-PE07-31)? — *chờ BA*.
- **Q-QC-PE07-3** — AC2 "p95 ≤ 5 ms khi trúng cache": QC đo qua `chat-gate` (có chi phí HTTP); QC viết test Go riêng đo `IsLocked` trực tiếp? Chấp nhận đo qua HTTP trừ 1 ms nền? — *chờ BA*.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md v1.1 (FEAT-weekly-exam, APPROVED). Mẫu mã: `scripts/p507-src/`.

Tổng: 48 TC.

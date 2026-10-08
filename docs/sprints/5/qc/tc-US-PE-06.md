# QC test case — US-PE-06 (làm bài lập trình C / C++: soạn mã, `Chạy thử`, `Nộp lời giải` nhiều lần, SSE, hết giờ khi đang gõ, không lộ test ẩn)
Nguồn: `docs/specs/FEAT-weekly-exam/US.md` US-PE-06 AC1–AC14 + `SRS.md` 4.4.1–4.4.5 (bản nháp, chạy thử, nộp, SV thấy gì, chốt bản tính điểm), 4.5.2 (đường chấm), 6.4 (trường cho phép), 7.2–7.3. Quyết định PM `plan.md`: nộp nhiều lần, lần cuối tính. Hộp đen; stack như `tc-US-PE-02.md` (cần `judge`); Chrome/CDP thật.

Tiền điều kiện chung: bài thi `E2` loại `CODE` đang `OPEN`: 2 câu code `QC1` (cho phép `c11` + `cpp17`; 3 test: `sample1` mẫu, `h1` ẩn trọng số 2, `h2` ẩn trọng số 3; canary `QC-HID-<uuid>` ở tên + input + expected test ẩn; lời giải mẫu chứa `QC-REF-<uuid>`), `QC2` (chỉ `cpp17`); câu đã verify (PE-03). `EXAM_RUN_LIMIT`=10 / 10 phút, `EXAM_SUBMIT_COOLDOWN`=15 s, `EXAM_SUBMISSION_CAP`=30, `EXAM_GRACE_SECONDS`=10. Mã thử do QC viết (`scripts/p506-src/`: `ok.cpp` đúng, `wa.cpp` sai 1 nhánh, `ce.cpp`, `tle.cpp`, `re.cpp`). Công cụ: **S** shell, **D** SQL, **A** Chrome/CDP, **G** `go test`, **P** Python.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-PE06-01 | AC1 (**test ẩn không lộ**) | lượt `IN_PROGRESS` | **S** `GET …/attempts/mine`; **P** duyệt toàn thân tìm canary `QC-HID-`, `QC-REF-`; liệt kê khoá của `items[].code` | `code` chỉ `languages, time_limit_ms, memory_limit_mb, starter_code, samples[{name,input,expected}]`; `samples` = **chỉ** test `is_sample` và `approved`; **0** canary; không `reference*`, `checker`, `tests_version`, số lượng / trọng số / tên test ẩn |
| TC-PE06-02 | AC1 | – | **D** thêm test mẫu chưa duyệt (`approved=false`, `is_sample=true`) rồi tải lại | không xuất hiện trong `samples` |
| TC-PE06-03 | AC1 | – | **G** `-run 'TestCodeItemPayloadWhitelist|TestHiddenTestsNeverInPayload' -v` | `ok` |
| TC-PE06-04 | AC2 (< 1024 px) | SV, bài `MIXED` | **A** mở `/exams/[id]/take` ở 390, 1023, 1024, 1440 | 390 và 1023: trắc nghiệm làm được; mỗi câu code có **đề đọc được** + dải "Bài lập trình cần màn hình rộng hơn (từ 1.024 px). Hãy mở bài thi này trên máy tính — bài của bạn vẫn là một lượt duy nhất và tự đồng bộ." (+ `Làm tiếp ở đây` **chỉ** ở dải chỉ-đọc khi **không** phải nơi ghi — PM #16b; ở nơi đang ghi **không** có nút); **0** `textarea`; 1024/1440: bố cục hai cột (đề + mẫu \| soạn mã + kết quả), có `textarea` |
| TC-PE06-05 | AC2 | – | **A** màn **trước khi bắt đầu** bài `CODE`/`MIXED`; bài `MCQ` | có dòng "Bài này có phần lập trình, cần màn hình ≥ 1024 px" chỉ ở `CODE`/`MIXED`; `AUDIT_SRC` 1024/1440 sạch |
| TC-PE06-06 | AC3 (editor) | 1440 | **A** `textarea`: cột số dòng cuộn đồng bộ; `spellcheck=false`, `autocapitalize=off`, `aria-label="Mã nguồn bài {n}"`; `Tab` → 4 dấu cách; `Shift+Tab` bỏ thụt; `Esc` rồi `Tab` rời ô; dòng gợi ý lối thoát | đúng từng thuộc tính; **không** thư viện editor (`grep -E 'codemirror|monaco|ace-builds|prism' frontend/package.json` = 0); bẫy bàn phím có lối thoát và có chữ nói rõ |
| TC-PE06-07 | AC3 (giới hạn, dán, hiệu năng) | – | **A** dán 64 KiB; gõ vượt 65.536 byte (đếm byte, ký tự tiếng Việt 2–3 byte); dán nhiều dòng; dòng dài 5.000 ký tự; gõ liên tục 64 KiB đo INP; chú thích tiếng Việt `// Nguyễn Văn Á ệ ữ` | đếm "{x}/65.536" đúng **theo byte**; chặn gõ quá giới hạn kèm lời; dán giữ nguyên xuống dòng; trang **không** tràn ngang (`scrollWidth ≤ innerWidth`), dòng dài cuộn trong ô; INP ≤ 200 ms; dấu tiếng Việt không vỡ khi lưu / tải lại |
| TC-PE06-08 | AC3 | – | **A** axe 1024/1440; `$PW exam.spec.ts -g 'code editor'` | 0 `serious`/`critical`; `rc=0` |
| TC-PE06-09 | AC4 (ngôn ngữ) | `QC1` cho `c11`+`cpp17` | **A** gõ mã ở `c11`, đổi `cpp17`, gõ khác, quay `c11`, tải lại; **S** `PUT draft` `language=java`; **D** `code_drafts` | mỗi ngôn ngữ **bản nháp riêng**, đổi qua lại không mất; lần đầu chọn ngôn ngữ nạp `starter_code` của ngôn ngữ đó; `java` → `422 LANGUAGE_NOT_ALLOWED`; tải lại mở ngôn ngữ của bản nháp lưu gần nhất; khoá `(attempt,item,language)` đúng |
| TC-PE06-10 | AC4 | – | **G** `-run 'TestDraftPerLanguage|TestLanguageNotAllowed' -v`; **A** `$PW exam.spec.ts -g 'language switch keeps drafts'` | `ok`; `rc=0` |
| TC-PE06-11 | AC5 (tự lưu nháp, rev) | – | **S** `PUT …/code/{itemId}/draft {language,source,base_rev}` với `X-Exam-Tab`: `base_rev=0` lần đầu, rồi đúng `rev`; `source` 64 KiB và 64 KiB + 1 | trả `{rev,saved_at}`, `rev` +1 mỗi ghi; 64 KiB + 1 → `422 SOURCE_TOO_LARGE`; thân phản hồi không chứa mã của người khác |
| TC-PE06-12 | AC5 (xung đột) | hai tab (T1, T2) | **S** T1 và T2 cùng `base_rev`; sau takeover; `base_rev` cũ | `409 DRAFT_CONFLICT` `{current_rev,updated_at}`; **không** ghi đè |
| TC-PE06-13 | AC5 (UI xung đột) | hai tab cùng phiên | **A** tab 1 gõ + lưu; tab 2 (đã takeover) gõ với bản cũ | "Bản trên máy chủ mới hơn (lưu lúc …). [Dùng bản trên máy này] [Dùng bản đã lưu]"; **giữ nguyên chữ đang gõ**; chọn mỗi nút cho kết quả đúng; không mất chữ |
| TC-PE06-14 | AC5 (offline) | – | **A** gõ khi offline 30 s rồi online | khoá `exam-code:<attempt>:<item>:<language>` ở `localStorage`; sau online bản nháp lên máy chủ đủ (so `source_sha256`); backoff như PE05 |
| TC-PE06-15 | AC5 | – | **G** `-run 'TestDraftRevIncrements|TestDraftConflict409|TestDraftAfterDeadline' -v`; `$PW exam.spec.ts -g 'draft conflict resolution'` | `ok`; `rc=0` |
| TC-PE06-16 | AC6 (`Chạy thử` — chỉ mẫu) | lượt, judge sống | **S** `POST …/code/{itemId}/run` có `Idempotency-Key` với `ok.cpp`, `wa.cpp`, `ce.cpp`, `tle.cpp`, `re.cpp`; theo dõi SSE `exam.run` và `GET …/runs/{run_id}` | `202 {run_id}`; chỉ chạy **test mẫu** (đếm `POST /run` tới judge = 1 + số mẫu, không có test ẩn); kết quả `{status, compile_ok, compile_log?, samples:[…]}`; `ok`→`AC`, `wa`→`WA`, `ce`→`compile_ok=false`, `tle`→`TLE`, `re`→`RE`; **không** thay đổi điểm (`exam_attempts.auto_score`, `code_submissions kind=SUBMIT` không đổi) |
| TC-PE06-17 | AC6 (giới hạn 10 / 10 phút) | – | **S** `run` 11 lần liên tiếp (cùng SV, qua **hai bài thi đang mở** nếu có để kiểm "mọi bài"); đợi `retry_after` rồi gọi lại; đo cửa sổ trượt | lần 1–10 `202`; lần 11 `429 RATE_LIMITED` `retry_after` = giây tới khi phần tử cũ nhất hết hạn (làm tròn lên); sau `retry_after` lần kế `202`; hạn mức tính trên **mọi** bài thi của SV (không theo bài) |
| TC-PE06-18 | AC6 (đầu vào) | – | **S** thiếu `Idempotency-Key`; `source` rỗng; `source` 64 KiB + 1; `language` lạ; SV khác chạy vào lượt người khác; đã nộp bài thi; hết giờ | `422 IDEMPOTENCY_KEY_REQUIRED`; `422 SOURCE_EMPTY`; `422 SOURCE_TOO_LARGE`; `422 LANGUAGE_NOT_ALLOWED`; `404`; `409 ATTEMPT_ALREADY_SUBMITTED`; `409 ATTEMPT_CLOSED` |
| TC-PE06-19 | AC6 (judge chết / làn riêng) | – | **S** `docker compose stop judge`, `run`; judge sống + 60 bài trong `judge.submit` rồi `run` | `503 JUDGE_UNAVAILABLE` ≤ 3 s + `retry_after`; `run` xong ≤ 5 s dù hàng nộp đầy |
| TC-PE06-20 | AC6 | – | **G** `-tags integration -run 'TestRunSamplesOnly|TestRunRateLimit10Per10Min|TestRunLaneNotBlockedBySubmitBacklog|TestRunDoesNotAffectScore' -v` | `ok` |
| TC-PE06-21 | AC7 (hiển thị) | `ok/wa/ce/tle/re/ole/mle/IE` | **A** từng ca: nhãn "Đúng", "Sai kết quả", "Quá thời gian", "Quá bộ nhớ", "Lỗi khi chạy", "In ra quá nhiều", "Lỗi biên dịch", "Hệ thống chưa chạy được"; `WA` hiện đầu vào / mong đợi / của bạn (≤ 2 KiB mỗi khối, ký hiệu khoảng trắng / xuống dòng); `CE` hiện `compile_log` ≤ 8 KiB với `main.cpp`; **không** `stderr` chương trình | nhãn Việt đúng, chữ + biểu tượng (không chỉ màu); khối cắt ≤ 2 KiB; đường dẫn tuyệt đối thay bằng `main.c`/`main.cpp`; `stderr` (canary in ra `stderr`) **không hiện**; 0 từ `sandbox`, `judge`, `verdict`, `go-judge`, mã trần `AC/WA/TLE…` |
| TC-PE06-22 | AC7 | – | **S** `$PW exam.spec.ts -g 'run result display'`; `bash scripts/ui-antipatterns.sh` | `rc=0`; 19 phép ✓ |
| TC-PE06-23 | AC8 (`Nộp lời giải`) | lượt | **S** `POST …/code/{itemId}/submit` `Idempotency-Key`; **D** `code_submissions` + `outbox` `judge.enqueue` cùng transaction (tắt worker, đếm); `source_sha256` | `202 {submission_id}`; `kind=SUBMIT`; outbox cùng giao dịch; `source_sha256 = sha256(source)` |
| TC-PE06-24 | AC8 (giãn cách 15 s) | – | **S** nộp hai lần trong 15 s; sau 15 s nộp lại; đo `retry_after` | lần hai `429 RATE_LIMITED` `retry_after` ≤ 15; sau 15 s `202`; giãn cách **theo `(attempt,item)`** (nộp bài khác cùng lúc → `202`) |
| TC-PE06-25 | AC8 (trần 30) | – | **S** nộp 30 lần (QC đặt `EXAM_SUBMIT_COOLDOWN=1` ở stack QC cho ca này) rồi lần 31 | `409 SUBMISSION_LIMIT_REACHED` `{cap:30}`; lần 30 vẫn `202` |
| TC-PE06-26 | AC8 (lần cuối tính + superseded) | judge chậm | **S** nộp `wa.cpp` rồi (QUEUED) nộp `ok.cpp`; xem `status` lần 1 | lần 1 `SUPERSEDED` (0 lần sandbox), lần 2 `DONE`; `is_final` = lần 2; `source` rỗng `422`; sau nộp bài thi / hết giờ `409` |
| TC-PE06-27 | AC8 (SSE tiến độ) | – | **A** theo dõi SSE `exam.submission` | `QUEUED` "Đã nhận, đang xếp hàng" → `RUNNING` "Đang chấm" → `DONE`; không lặp; không từ kỹ thuật |
| TC-PE06-28 | AC8 | – | **G** `-tags integration -run 'TestSubmitCodeEnqueues|TestSubmitCooldown|TestSubmitCap30|TestLastSubmissionCounts|TestSubmitSupersedes' -v` | `ok` |
| TC-PE06-29 | AC9 (**trong giờ thi**: chỉ biên dịch + mẫu) | lần nộp `DONE`, bài `OPEN` rồi `CLOSED` (chưa công bố) | **S** `GET …/submissions/{sid}`; **P** liệt kê khoá và quét canary `QC-HID-` | chỉ `{id,status,language,created_at,compile_ok,compile_log?,samples:[{name,verdict,time_ms,memory_kb}],is_final,source}` — `source` (PM #16a) **chỉ** khi người gọi là chủ bản nộp; SV khác `404` và không có `source` trong bất kỳ thân nào; danh sách #36 **không** kèm `source`; **không** `results` test ẩn, `passed_weight`, `total_weight`, `verdict` chung, `tests_version`, điểm, số test ẩn; canary 0 |
| TC-PE06-30 | AC9 (CE / IE) | – | **S** nộp `ce.cpp`; dừng judge cho lần nộp `IE` | `CE` hiện lỗi biên dịch; `IE` hiện "Hệ thống chưa chấm được bài của bạn. Bài đã được lưu và giảng viên sẽ xử lý."; bản mới nhất có nhãn "Lần nộp tính điểm" |
| TC-PE06-31 | AC9 | – | **G** `-run 'TestSubmissionStudentViewHidesHidden|TestSubmissionViewCE|TestSubmissionViewIE' -v` | `ok` |
| TC-PE06-32 | AC10 (lịch sử) | 5 lần nộp | **S** `GET …/code/{itemId}/submissions?limit=2&cursor=…`; **A** "Lần nộp": xem lại mã, `Dùng lại mã này`; SVB đọc lịch sử của SVA | mới nhất trước; cursor ổn định; trạng thái chữ ("Đang chấm", "Biên dịch được · 2/3 test mẫu đúng", "Lỗi biên dịch"); nhãn "Lần nộp tính điểm" ở bản cuối; `Xem mã` đọc `source` từ #37 (danh sách không có mã); `Dùng lại mã này` nạp vào ô; SVB `404` (cả danh sách lẫn một bản) |
| TC-PE06-33 | AC10 | – | **G** `-run 'TestSubmissionsListOwnOnly|TestSubmissionsListCursor' -v`; `$PW exam.spec.ts -g 'submission history'` | `ok`; `rc=0` |
| TC-PE06-34 | AC11 (lần nộp cuối) | `QC1`: nộp `ok.cpp` rồi `wa.cpp` | **S** nộp bài thi; **D** `code_submissions`, điểm | lấy **lần `SUBMIT` cuối** (`wa`) — điểm theo `wa`; không lấy lần đúng trước đó |
| TC-PE06-35 | AC11 (nộp tự động từ nháp) | `QC1` chưa nộp lần nào, nháp `ok.cpp` đã lưu | **S** nộp bài thi (hoặc hết giờ); **D** `code_submissions where auto` | tạo `SUBMIT` `auto=true` từ **bản nháp đã lưu trên máy chủ** (ngôn ngữ cập nhật gần nhất), chấm như thường; lịch sử có "Nộp tự động khi hết giờ" |
| TC-PE06-36 | AC11 (không đè) | đã có `SUBMIT` `ok`; nháp mới gõ dở sai | **S** nộp bài thi | **không** dùng nháp; điểm theo `SUBMIT`; không tạo `SUBMIT` auto |
| TC-PE06-37 | AC11 (nháp rỗng) | không nộp, nháp rỗng | **D** điểm câu | `earned=0` (không phải `IE`); không dòng `SUBMIT` |
| TC-PE06-38 | AC11 | – | **G** `-tags integration -run 'TestFinalPicksLastSubmit|TestAutoSubmitDraftWhenNone|TestNoAutoSubmitWhenSubmitExists|TestEmptyDraftNoSubmit' -v` | `ok` |
| TC-PE06-39 | AC12 (**hết giờ khi đang gõ**) | bài `duration=1 phút` (stack QC) | **A** gõ liên tục mã tới giây cuối; ghi thời điểm `PUT draft` cuối; **D** bản nháp trên máy chủ, `SUBMIT auto` | khi về 0: máy khách gửi **ngay** 1 `PUT draft` (không chờ debounce 2 s) và **khoá ô soạn**; máy chủ nhận tới `deadline_at + 10 s`; bản đó là bản dùng cho `SUBMIT auto=true`; chữ vẫn còn trong ô (chỉ đọc) |
| TC-PE06-40 | AC12 (sau ngưỡng) | – | **S** `PUT draft` tại `deadline_at + 9 s` (nhận) và `+ 11 s` (từ chối) | `409 ATTEMPT_CLOSED` ở 11 s; màn "Hết giờ — phần bạn gõ sau giờ không được tính" |
| TC-PE06-41 | AC12 | – | **S** `$PW exam.spec.ts -g 'timeout while typing'` | `rc=0` |
| TC-PE06-42 | AC13 (SSE đứt) | đang chấm | **A** chặn SSE bằng CDP (`Network.emulateNetworkConditions offline` chỉ cho `EventSource`, hoặc `page.route` 5xx), chờ judge xong, bật lại | máy khách `GET runs/{id}` / `submissions/{sid}` mỗi 2 s tới `DONE`/`ERROR`; kết quả hiện **đúng một lần**; không thêm kết nối SSE (đếm `EventSource`/request `/events` ≤ 2 / người) |
| TC-PE06-43 | AC13 | – | **S** `$PW exam.spec.ts -g 'sse drop during judge'` | `rc=0` |
| TC-PE06-44 | AC14 (**không rò — mọi endpoint code**) | canary ở test ẩn (tên, input, expected), lời giải mẫu, `answer_key` câu khác, bài nộp của SVB (`QC-SUB-<uuid>`) | **S** SVA gọi `…/attempts/mine`, `…/draft`, `…/run`, `…/runs/{id}`, `…/submit`, `…/submissions`, `…/submissions/{id}` ở 3 trạng thái (đang làm / đã nộp / đóng chưa công bố); quét **thân, header, thông điệp lỗi, `details`, sự kiện SSE** (`curl -N`) | **0** canary ở mọi nơi; chỉ khoá trong danh sách cho phép; header không chứa `Server`/`X-…` lộ nội bộ |
| TC-PE06-45 | AC14 (chéo người, vai) | – | **S** SVA đọc `submission` / `run` / `draft` của SVB (đổi id); TA, GV, ADMIN vào các route này | `404`; `403 reason=role`; ADMIN `403` |
| TC-PE06-46 | AC14 | – | **G** `-tags integration -run TestNoAnswerLeakCodeEndpoints -v` | `ok` |
| TC-PE06-47 | tổng | – | **S** `go vet ./... && golangci-lint run && go test -race -count=1 ./... && go test -count=1 ./internal/contract/...`; `pnpm -C frontend lint && build`; `audit.mjs` | `rc=0` |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| SV gõ code trên điện thoại rồi mất | 04, 05 |
| Mất chữ khi hai tab / mất mạng / hết giờ | 12–14, 39, 40 |
| `Chạy thử` lạm dụng / judge chết / hàng đầy | 17, 19 |
| Nộp dồn cuối giờ / vượt trần | 24–26 |
| Nháp gõ dở thay lời giải đúng | 36 |
| Lộ test ẩn / lời giải mẫu / bài người khác | 01, 29, 44, 45 |
| Lỗi biên dịch / `IE` hiển thị cho SV | 21, 30 |
| SSE đứt khi đang chấm | 42 |

## Câu hỏi cho BA / PM
- **Q-QC-PE06-1** — AC6 "hạn mức tính trên mọi bài thi đang mở của người đó": QC kiểm bằng 2 bài thi mở cùng lúc; nếu seed chỉ có 1 bài, QC tự tạo bài thứ hai. Chấp nhận? — *chờ BA*.
- **Q-QC-PE06-2** — AC3 "INP ≤ 200 ms khi gõ 64 KiB": QC đo bằng `PerformanceObserver` `event` entries trong Chrome headless trên máy dev; chấp nhận làm bằng chứng? — *chờ BA*.
- **Q-QC-PE06-3** — AC11: "ngôn ngữ cập nhật gần nhất" khi hai ngôn ngữ cùng nháp: QC chấm theo `updated_at` lớn nhất. Đúng? — *chờ BA*.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md v1.1 (FEAT-weekly-exam, APPROVED). Mã thử: `docs/sprints/5/qc/scripts/p506-src/` (QC tự viết khi chạy).

Tổng: 47 TC.

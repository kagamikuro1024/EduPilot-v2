# QC cổng P7 — Bài tập: tạo, nộp (app / ZIP / thư / Teams giả lập), chấm AI nháp, duyệt, công bố vào sổ điểm, phúc khảo + seed sprint 9
Nguồn: `docs/phases/P7.md` ("Cổng nghiệm thu", "Bạn tự kiểm", "Ghi cho luận văn"), `docs/specs/FEAT-assignments/{US,SRS,TL-REVIEW}.md`, `docs/sprints/9/prompts/qc-tc.md`, `docs/sprints/9/plan.md`. Chạy ở pha 2 sau khi có handoff đủ US-P7-01…11 + US-S9-SEED và CI xanh. Mọi lệnh dùng `LLM_PROVIDER=fake`; **không** LLM thật, Microsoft thật, thư thật. Quy ước và biến: `README.md`, `scripts/p7-env.sh`. v1, 2026-11-01.

## A. Bảy lệnh của `P7.md`
| ID | Lệnh | Kết quả mong đợi |
| --- | --- | --- |
| G7-01 | `pnpm -C frontend exec playwright test assignment-lifecycle.spec.ts grade-appeal.spec.ts --workers=2 --retries=0` | xanh toàn bộ, 0 flaky; ghi số test; mọi `-g` nêu ở US có ≥ 1 test (QC `grep` tên: `assignments tab layout`, `student page states`, `dropzone rejects before upload`, `submit confirms with server timestamp`, `unmatched assign`, `unmatched dismiss`, `queue default filter and flags`, `review two columns`, `review edit score undo`, `review key A next`, `publish bulk confirm`, `student sees published result`, `teams sync happy then throttle`, `demo step 5`, `student files appeal`, `teacher decides appeal`…) |
| G7-02 | `cd backend-go && go test ./internal/integration -run TestStudentNeverSeesDraftGrade -v` (+ QC `scripts/p7-student-sweep.sh`, TC-P7-09-13) | `ok`; đủ 6 trạng thái × mọi bề mặt; canary `CANARY-DRAFT-7K3`, `CANARY-AISCORE` = 0 ở API, SSE, chuông, thư, chat AI, lịch, "Hôm nay", `/me/grades`; mã lỗi nộp lại giống nhau giữa `DRAFT` và `REVIEWED` |
| G7-03 | `cd backend-go && go test -race ./internal/submission/... ./internal/jobs/...` | `ok`, không `DATA RACE`; số test không thấp hơn đầu sprint |
| G7-04 | `cd backend-go && go test ./internal/integration -run TestImapToDraftGrade -v` | `ok`: thư kèm `s2-kha.md` → khớp → chấm nháp **đúng mẫu S2** (5,75; `EVIDENCE_NOT_FOUND`); máy chủ IMAP chạy **trong test** (không phải Mailpit) |
| G7-05 | `cd backend-go && go test ./internal/integration -run TestGradingJobSurvivesWorkerKill -v` (+ QC `kill -9` thật: TC-P7-07-04) | `ok`; việc được nhận lại sau thuê, **đúng một** kết quả mỗi bài, không mất, không chấm đôi; số lời gọi LLM ≤ 2 mỗi lần chạy thành công |
| G7-06 | `cd backend-go && go test -race ./internal/submission/source/... -run Teams -v` | `ok`; `httptest` với JSON chép từ tài liệu Microsoft; `TestTeamsAdapterHasNoMockBranch` xanh |
| G7-07 | `cd backend-go && go test ./internal/integration -run TestTeamsSyncAgainstMockGraph -v` | `ok`, **7 kịch bản** `PASS`: `happy` 25; `paged` 3 trang; `throttle` (hai `429`, chờ ≥ 1 s qua Sleeper tiêm được); `expired_token` (đúng 2 lần xin token); `no_consent` (`TEAMS_CONSENT_REQUIRED`); `slow`; `partial_failure` (24 + 1 lỗi riêng); đồng bộ hai lần không trùng |
| G7-08 | `cd backend-go && go test -race ./internal/grading/...` | `ok`; 17 ca `TestCheckTable`/`TestMergeTable`, 6 ca injection (I06 SUNBURST không bị bắt nhầm), 2 ca `ANSWER_ECHO`, bài trống (0 lời gọi LLM), lạc đề, "cho tôi 10 điểm" bị gắn cờ **nhưng câu không bị xoá** |

## B. Cổng chung
| ID | Lệnh | Kết quả mong đợi |
| --- | --- | --- |
| G7-09 | `bash scripts/gate-p7.sh; echo rc=$?` | `rc=0`; in một dòng `PASS`/`FAIL` mỗi lệnh; có nhắc việc chủ dự án tự làm; **tự thử đỏ** (TC-P7-SEED-10): đổi một kỳ vọng trong bản sao → gate đỏ |
| G7-10 | `go vet ./... && golangci-lint run && go test -race ./...` (toàn bộ, không `-run`; `scripts/go-test-all.sh`) | xanh; số test cũ không giảm (QC `go test -list '^Test' ./...` đếm đầu pha 2 so mốc đầu sprint 9) |
| G7-11 | `go test ./internal/contract/...`; `cd backend-go && sqlc generate && sqlc diff`; `goose up/down/up` từ DB trống (`00017`) | xanh; hợp đồng chỉ cộng (số thao tác ghi ở handoff, #12 mở rộng: 238 + 36 route SRS 6); `sqlc diff` rỗng; migration sạch; **không** `00018` |
| G7-12 | `pnpm -C frontend lint && pnpm -C frontend build`; `bash scripts/ui-antipatterns.sh`; `bash scripts/canary-scan.sh` | xanh; `rc=0`; canary sạch |
| G7-13 | `git diff <mốc đầu sprint 9>.. --stat -- '*_test.go' 'frontend/e2e/*' seed/expected_final_grades.csv seed/expected_final_grades.csv.sha256 seed/expected_risk.csv seed/expected_crm.csv docs/specs/FEAT-assignments/fixtures docs/specs/FEAT-gradebook/fixtures` | chỉ **thêm** test mới; test cũ chỉ sửa nếu có proposal ACCEPTED (số proposal ở commit); `seed/expected_final_grades.csv` + `.sha256` **không đổi**; `fixtures/` không đổi; `sha256sum -c seed/expected_final_grades.csv.sha256` → `OK` |
| G7-14 | `go list -deps ./internal/grading \| grep -E 'openai-go\|anthropic'`; `grep -rn 'openai\|anthropic' backend-go/internal/grading backend-go/internal/assignment backend-go/internal/submission --include=*.go \| grep -v _test`; `go test ./internal/llm -run TestMaskOnlyInLLMGateway` | rỗng (mọi lời gọi LLM qua `internal/llm`); xanh |
| G7-15 | quét cấm: `! grep -rnE 'float(32\|64)\|ParseFloat\|math\.Round' backend-go/internal/grading/grade*.go backend-go/internal/grading/check*.go backend-go/internal/assignment --include=*.go \| grep -v _test`; `grep -rn 'users.student_code' backend-go/internal/submission --include=*.go \| grep -v _test` | rỗng (điểm là `decimal`; khớp không đọc MSSV tự khai); không `if mock` ở `teams.go` |
| G7-16 | `grep -c mock-graph docker-compose.local.yml docker-compose.prod.yml`; `APP_ENV=production GRAPH_BASE_URL=http://mock-graph:8090/v1.0` khởi động gateway | local ≥ 1, prod 0; gateway không khởi động, nêu tên biến |

## C. Kiểm độc lập của QC
| ID | Việc | Kết quả mong đợi |
| --- | --- | --- |
| G7-17 | `python3 scripts/p7-hand-calc.py all` (đáp án BA, không dùng mã dev) | **đã chạy pha 1 trên `fixtures/`: 7/7 mẫu chấm và 30/30 điểm học phần v2 khớp, 22/30 khác bản 1**; pha 2 chạy lại sau khi dev sinh lại `fixtures` (nếu BA đổi) |
| G7-18 | `bash scripts/p7-rbac.sh` trên stack seed (36 route × 7 vai) | `RESULT: PASS`, không `LỆCH` (chạy trên DB thử: có tác dụng phụ) |
| G7-19 | `p7-student-sweep.sh` (TC-P7-09-13) + `p7-seed-diff.py` (22/22 chấm, từng dòng `seed-plan.csv`) | `0` canary; 22/22 |
| G7-20 | đột biến (`/tmp/qc-mut`, mỗi story 3–6 ca ở các hàng `-M`) | mọi đột biến làm đỏ ≥ 1 test; đột biến sống sót ghi vào report (bug thấp / đột biến tương đương) |
| G7-21 | sinh viên không thấy nháp — DOM thật: `playwright-cli` đăng nhập B ở 6 trạng thái, dump `body.innerText` + `localStorage` + console + tiêu đề tab | không nháp, không cờ, không tiến độ batch; chỉ "Đang chấm" |
| G7-22 | đo: `GET /submissions` 1.000 bài p95 ≤ 300 ms; chat không chậm hơn +20 % khi chấm 200 bài (`k6 … grading-vs-chat.js`); `EXPLAIN` không `Seq Scan` | đạt SLO ở `SYSTEM_DESIGN.md` §5 |
| G7-23 | kiểm 375 px: `/assignments/[id]` 4 trạng thái; LCP ≤ 2,5 s; không cuộn ngang; vùng thả ≥ 48 px | đạt |
| G7-24 | mock-graph so tên trường với ví dụ trên docs.microsoft.com (`educationSubmission`, `educationClass` members) — QC mở tài liệu, so từng chữ (TC-P7-11-04) | khớp từng chữ; ghi URL tài liệu và ngày kiểm |

## D. "Bạn tự kiểm" (`P7.md`) — QC làm bằng `playwright-cli` / tay, **chủ dự án vẫn tự làm lại**
| ID | Việc | Kết quả mong đợi |
| --- | --- | --- |
| G7-25 | Gửi 1 thư đúng mẫu `[BT03] <MSSV>` và 1 thư sai mẫu vào hộp thư thử (máy chủ IMAP thử, **không** mail thật) | một thư khớp, một thư vào `Chưa khớp` (đúng lý do bằng chữ) |
| G7-26 | Đọc 5 bài AI chấm (provider giả, bản ghi sẵn): S1, S2, S3, S6, S7 | bằng chứng trích đúng từ bài; nhận xét dùng được (QC ghi nhận xét thực tế — **chất lượng LLM thật là việc chủ dự án**) |
| G7-27 | Sinh viên B **không** thấy điểm trước khi công bố; sau công bố thấy **7,50** | đúng |
| G7-28 | `Đồng bộ từ Teams` kịch bản `happy` rồi `throttle`; bật `no_consent` | đủ 25 bài, tiến độ mượt; `no_consent` hiện câu **người không rành kỹ thuật** hiểu |
| G7-29 | So một response mock-graph với ví dụ Microsoft Graph | tên trường khớp từng chữ |
| G7-30 | **Chủ dự án** chấm tay 60 bài (E3) và quyết lại [ĐIỂM] ngày 01/11 (bài tập vào công thức lớp 1) | ngoài phạm vi QC; ghi nhắc ở report |

## E. Rủi ro QC để ý khi chấm (từ đọc spec, chưa thấy code)
1. Nháp lộ qua bề mặt phụ: SSE `job.progress`, chuông gộp, thư, câu lỗi nộp lại, lịch, `/me`, tool AI — gom ở TC-P7-09-13/14.
2. Khớp danh tính: MSSV tự khai (`joined_via='CODE'`), `From` giả thiếu `dmarc=pass`, header `Authentication-Results` tự thêm (TC-P7-04-04, TC-P7-05-12/13).
3. Điểm bài tập ≠ ghi sổ: công bố **không** ghi `grade_entries`; `seed/expected_final_grades.csv` không đổi (TC-P7-09-03/06).
4. Hàng đợi: `XADD` sau `COMMIT`, thuê + nhịp tim, `SUPERSEDED` không gọi LLM (TC-P7-07-01/04/10).
5. Bài chưa khớp không thành 0 (TLR-9) và `Bỏ qua` có đường kết thúc (TC-P7-04-16/17, TC-P7-SEED-07).
6. Hạn nộp / gia hạn / `late_days` tính lúc chấm, không lưu (TLR-8) (TC-P7-02-14, TC-P7-03-12/13).
7. Phúc khảo không đi qua `escalation_tickets`, AI không tham gia (TC-P7-10-07/09).
8. Câu hỏi mở cho PM: xem mục "Câu hỏi" ở các `tc-*.md` (Q-01-a, Q-02-a, Q-05-a, Q-06-a, Q-06-b, Q-06-c, Q-09-a, Q-S9-a).

---
# KẾT QUẢ PHA 2 — chạy ngày 2026-11-01 trên `sprint/9-p7` @ `8e478910` (HEAD `61c35b39`, chỉ thêm tài liệu), stack dựng từ DB trống (`dev.mjs up` + `seed.mjs`, cổng host +100)
CI `8e478910`: success. Proposals #15, #16 ACCEPTED. Script QC: `scripts/p7-*.py|sh` (đã commit).

| ID | Kết quả | Bằng chứng |
| --- | --- | --- |
| G7-01 Playwright `assignment-lifecycle` + `grade-appeal` `--workers=2 --retries=0` | **PASS** | xanh trong `gate-p7.sh` (`assignment-lifecycle` + `grade-appeal` 8 ca), 0 flaky; 0 ca bị xoá so với đầu sprint |
| G7-02 `TestStudentNeverSeesDraftGrade` | **PASS** | `ok`; QC thêm `p7-student-sweep.py`: 6 trạng thái (không nộp / QUEUED / RUNNING / DRAFT / REVIEWED / FAILED) × 12 bề mặt API + SSE + `GET /jobs/{id}` (404) + chat → 0 dấu vết nháp; DOM thật `/assignments/{id}` ở 1440 và 390: chỉ "Đang chấm", 0 nháp / cờ / điểm, 0 trong `localStorage` và console |
| G7-03 `go test -race ./internal/submission/... ./internal/jobs/...` | **PASS** | 57 test, không `DATA RACE` |
| G7-04 `TestImapToDraftGrade` | **PASS** | máy chủ IMAP trong test; QC thêm thư giả `From` trên `mock-imap` thật (`p7-imap-live.py`) |
| G7-05 `TestGradingJobSurvivesWorkerKill` | **PASS** (Go) | QC không `kill -9` được giữa hai lượt (provider giả quá nhanh) — chỉ test Go; việc `docker pause` worker + `SUPERSEDED` đã kiểm ở US-P7-07 |
| G7-06 `go test -race ./internal/submission/source/... -run Teams` | **PASS** | 10 test; `TestTeamsAdapterHasNoMockBranch` xanh |
| G7-07 `TestTeamsSyncAgainstMockGraph` | **PASS** | 7 kịch bản; QC chạy tay 8 kịch bản trên stack (US-P7-11) |
| G7-08 `go test -race ./internal/grading/...` | **PASS** | 128 test, không `DATA RACE` |
| G7-09 `bash scripts/gate-p7.sh` | **PASS** | rc 0, `GATE P7: PASS`, 19 dòng PASS, có khối "Bạn tự kiểm"; **tự thử đỏ**: đổi S2 5,75 → 5,50 ở bản sao → `check … grade` rc 1 ("11/22") và `TestSeedGradeAllMatchesExpected` đỏ |
| G7-10 `go vet`, `golangci-lint run`, `go test -race ./...` (`scripts/go-test-all.sh`), `sqlc diff` | **PASS** (vòng 3, `d37c8138`) | `go vet -tags testroutes ./...` rc 0; `golangci-lint` 0 issues; `sqlc diff` rc 0; **`go-test-all.sh` rc 0: 57 gói `ok` với `-race -tags testroutes` + bước `TimingEqualized` 3 lượt**, chạy riêng một mình trên máy QC (vòng 2 đỏ 2/2 ở `TestRetryThreeThenDead`; đã sửa test, BUG-P7-07-5). `TestRetryThreeThenDead` `-race` thêm 3 lượt riêng: xanh. Sprint thêm ≥ 426 hàm `Test…` Go, 0 xoá |
| G7-11 `go test ./internal/contract/...`; `sqlc diff`; `goose up/down/up` | **PASS** | contract xanh (cổng, 287 thao tác); `sqlc diff` sạch; `00017` up / down / up: `TestMigration00016UpDownUp` và `db` xanh trong lượt toàn bộ |
| G7-12 `pnpm lint && pnpm build`; `ui-antipatterns.sh`; `canary-scan.sh` | **PASS** | lint rc 0, build rc 0; antipatterns rc 0 (trong cổng); canary: log compose / `audit_log` / `outbox` / `llm_audit` / `jobs` / `llm_providers` đều 0 |
| G7-13 diff test cũ / e2e / `expected_*` / fixtures so mốc `2fea08ec` | **PASS** | `seed/expected_final_grades.csv` không đổi (`sha256sum -c` OK); fixtures BA chỉ **thêm** (20 tệp); test cũ sửa đều có proposal ACCEPTED (#7 migration, #8 / #9 `today.Topics()` 24 → 28 + `TestTodayQueryBudget`, #12 12 ảnh mốc + nhãn demo, #13 `AssignmentQueue` + `share_test`, #15 `demo_test` đếm theo loại, #16 `student page states`, `me_test` `noStamps`, `rig_test` `TEAMS_ENABLED`); 0 hàm `Test…` bị xoá, 0 ca Playwright bị xoá |
| G7-14 không SDK provider ngoài `internal/llm` | **PASS** | 0 import trực tiếp `openai` / `anthropic` ở `grading`, `assignment`, `submission` (`go list -f '{{.Imports}}'`); `go list -deps` chỉ thấy SDK gián tiếp qua `internal/llm` |
| G7-15 quét cấm `float` / `ParseFloat` / `math.Round`; `users.student_code` | **PASS** | chỉ `[]float64` ở `grading/queue.go` cho thời lượng giây (số đo, không phải điểm); `users.student_code` ở `submission` / `assignment` / `grading`: 0 |
| G7-16 `mock-graph` trong compose; production từ chối | **PASS** (theo #14) | local 12, test-seed 0 (chồng lên local), `docker-compose.prod.yml` chưa tồn tại (PR); `TestProductionRejectsMockGraph` xanh |
| G7-17 `p7-hand-calc.py all` | **PASS** | 7/7 mẫu chấm; 30/30 điểm v2, 22/30 khác bản 1 |
| G7-18 `p7-rbac.sh` (35 route × vai) | **PASS** (vòng 3) | `RESULT: PASS` kể cả route SRS #15 `GET …/submissions/{id}` (đã có ở `d37c8138`); sửa harness (sinh viên ngoài lớp = `sv.moi`; seed chỉ có **một** giảng viên dạy cả hai lớp → bỏ hàng "giảng viên lớp khác"); vòng 2 đã phát hiện route #15 chưa có (BUG-P7-03-4, đã sửa) |
| G7-19 quét sinh viên + `p7-seed-grade.py` 22/22 | **PASS** | 0 canary / nháp; 22/22 từng ô (US-S9-SEED) |
| G7-20 đột biến | **PASS** | 31 đột biến sprint 9 làm đỏ ≥ 1 test; T4 (BUG-P7-11-1) từng sống sót, **vòng 3 đã đỏ** `TestTeamsNoAuthHeaderToListedSecondaryHost`; đột biến V1 ở route mới (bỏ cách ly lớp) đỏ `TestSubmissionDetailVersions`; các lỗ hổng cũ đã sửa (BUG-P7-01-1, 06-1) |
| G7-21 DOM sinh viên 6 trạng thái | **PASS** | xem G7-02 |
| G7-22 `GET /submissions` 1.000 bài p95 ≤ 300 ms, không `Seq Scan`; chat không chậm hơn +20 % khi chấm | **PASS** (30 bài cho k6) | `TestQueueP95Under300ms`, `TestQueueExplainNoSeqScan` xanh; k6 `grading` p95 TTFT 44,8 ms ≤ 143 ms, chờ hàng 28 ms ≤ 500 ms, 0 lỗi (**30 bài, không phải 200** — seed chỉ có 30 sinh viên; `TestChatUnaffectedDuringBatchGrading` phủ 200 việc ở Scheduler) |
| G7-23 375 px `/assignments/[id]`; LCP ≤ 2,5 s | **chờ chủ dự án #10** | 375 px, vùng thả ≥ 48 px, không cuộn ngang, axe 0 lỗi: đạt. LCP 4.723 ms > 2.500 ms (US-P7-03 AC9) — không phải việc của dev; chủ dự án chọn phương án ngày 01/11 (proposals #10) |
| G7-24 mock-graph khớp tài liệu Microsoft | **PASS** | so từng khoá ngày 01/11/2026 (report US-P7-11) |
| G7-25…G7-29 "Bạn tự kiểm" | **QC đã làm** | thư đúng mẫu → khớp, thư sai → `Chưa khớp` (đúng lý do bằng chữ); 22 bài chấm khớp mẫu; B không thấy điểm trước công bố, sau công bố **7,50**; Teams `happy` / `throttle` / `no_consent` ("Cần quản trị Microsoft 365 của trường cấp quyền đọc bài tập và danh sách lớp."); mock-graph khớp tài liệu. **Chủ dự án vẫn tự làm lại** |
| G7-30 chấm tay 60 bài (E3) và quyết lại [ĐIỂM] ngày 01/11 | ngoài phạm vi QC | nhắc chủ dự án |

## Kết luận cổng P7 (cập nhật vòng 3, `d37c8138`, CI success): **PASS về kỹ thuật — G7-23 ghi "chờ chủ dự án #10"**
- Hai chặn của dev đã đóng: **G7-10** (`go-test-all.sh` rc 0, 57 gói) và **BUG-P7-03-4** (route `GET …/submissions/{id}` có, QC hộp đen + `p7-rbac.sh` 35 route PASS). **BUG-P7-11-1** đã có test và đột biến T4 đỏ.
- **Chỉ còn G7-23:** LCP `/assignments/[id]` 4,7 s > 2,5 s (US-P7-03 AC9) — chờ chủ dự án quyết ngày 01/11 (proposals #10); cổng chỉ xanh trọn khi AC9 được xử lý.
- Chủ dự án vẫn tự làm: "Bạn tự kiểm" (G7-25…G7-29), chấm tay 60 bài (E3), quyết lại [ĐIỂM] 01/11, duyệt thư viện ngoài bảng (`x/text` — TL-3).
- Còn mở (không chặn): AC5 US-P7-07 đo ở 30 bài (PM đã chấp nhận, tải 200 bài thuộc P10 `seed-t1`); chưa kiểm trên stack: `kill -9` worker giữa hai lượt, Redis dừng 70 s, Teams hết thời gian thật.

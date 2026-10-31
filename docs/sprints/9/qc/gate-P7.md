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

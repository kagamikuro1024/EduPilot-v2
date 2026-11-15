# Slice G_s7_p5b: 124 ca còn N

### TC-id [AC] pre: Tiền điều kiện
STEP: Bước / lệnh
EXPECT: Kết quả mong đợi

### TC-01 [#3a tổng test Playwright] pre: Run `$S` xanh, đủ shard; run TRƯỚC cùng SHA (hoặc chỉ khác `ci.yml`)
STEP: Với mỗi shard `i/N` lấy dòng tổng kết cuối của bước `playwright test` (`X passed`, `Y skipped`, `Z flaky`, `W failed`) từ `--log`; cộng các shard; so với dòng tổng kết của run TRƯỚC; so `PW_N`
EXPECT: `Σ(passed+skipped+flaky+failed)` của các shard = tổng của run TRƯỚC = `PW_N`; `Σ passed` bằng `passed` TRƯỚC (sai lệch chỉ do test flaky đã ghi; mỗi lệch phải kèm tên test); `failed = 0`. Không test nào chạy 0 lần hay 2 lần

### TC-02 [#3a] pre: Như TC-01
STEP: `playwright test --list --grep-invert @real --shard=i/N` cho i = 1..N, gom tên test (`... > shard-i.txt`); `sort shard-*.txt \| uniq -d \| wc -l`; so `sort -u shard-*.txt` với `--list` không shard
EXPECT: Giao các shard rỗng (0 dòng trùng); hợp = tập đầy đủ, đúng `PW_N`

### TC-03 [#3a giữ cấu hình] pre: `ci.yml` ở `$S`
STEP: `grep -n 'playwright test' .github/workflows/ci.yml`; đọc `frontend/playwright.config.ts` (`workers`, `retries`)
EXPECT: Lệnh vẫn có `--grep-invert @real` và `--shard`; `workers: 2`, `retries: 1` không đổi so với trước; không có `--workers`, `--retries`, `--max-failures`, `--grep` mới, `continue-on-error`, hay `\|\| true`

### TC-04 [#3 báo cáo] pre: Run có shard đỏ (xem TC-12)
STEP: Tải artifact bằng `gh run download <id>`
EXPECT: Mỗi shard Playwright đỏ có artifact riêng, tên khác nhau (không ghi đè nhau), chứa `playwright-report` và `test-results`; lhci đỏ có artifact riêng cho từng nửa URL, chứa cả `.lighthouseci` (thư mục ẩn → cần `include-hidden-files`)

### TC-05 [#5 `-race -tags testroutes`] pre: Run `$S`
STEP: `gh run view <id> --log \| grep -E 'go test '` (mọi dòng matrix)
EXPECT: MỌI dòng lệnh `go test` ở các dòng matrix có đủ `-race` và `-tags testroutes`; env `TESTCONTAINERS_RYUK_DISABLED=true` và `EP_SKIP_TIMING=1` có ở mỗi dòng; không dòng nào có `-short`, `-run` hạn chế (trừ bước đo thời gian), `-failfast`

### TC-06 [#5 tổng gói Go] pre: Run `$S`, run TRƯỚC
STEP: Với mỗi dòng matrix lấy các dòng `ok  \tgithub.com/.../internal/...` và `?   ... [no test files]` từ `--log`; gom, `sort`, đếm; so với `GO_N` và với danh sách `ok` của run TRƯỚC
EXPECT: Tập gói `ok`/`no test files` của các dòng matrix hợp lại = `go list ./...` (đúng `GO_N`); không gói nào xuất hiện ở hai dòng; số gói `ok` bằng run TRƯỚC (42 ở mốc sprint 6, trừ gói mới thêm giữa hai SHA, ghi rõ)

### TC-07 [#5 tổng test Go] pre: Như TC-06
STEP: Chạy `go test -race -tags testroutes -v ./... \| grep -c '^--- PASS'` ở máy QC trên SHA `$S` ghi `P`; ở CI không có `-v` nên đối chiếu theo gói: mỗi gói `ok` có mặt (TC-06) và `GO_T` ở SHA trước tách = `GO_T` ở `$S` (`git stash`/`git worktree` hai SHA)
EXPECT: `GO_T` trước = `GO_T` sau; không dòng `--- FAIL`, không `panic`; tổng test không giảm

### TC-08 [#5 gói mới tự rơi vào "phần còn lại"] pre: Nhánh tạm `qc/tmp-ci-newpkg` từ `$S`
STEP: Thêm `backend-go/internal/zzqcprobe/probe_test.go` (`package zzqcprobe`, một `func TestProbe(t *testing.T){}`; gói có thêm file `doc.go`); push; xem run
EXPECT: Gói `internal/zzqcprobe` xuất hiện `ok` đúng MỘT lần, ở dòng "phần còn lại" (không ở dòng riêng của `course`/`auth`); bước kiểm hợp (nếu có) vẫn xanh; không phải sửa `ci.yml`. Xong **xoá nhánh tạm**

### TC-09 [#5 gói tách riêng không lọt] pre: Như TC-08
STEP: Trong log dòng "phần còn lại" tìm `internal/course`, `internal/auth`, `internal/auth/...`, `internal/course/...`
EXPECT: Không xuất hiện trong dòng "phần còn lại"; xuất hiện ở dòng riêng của chúng, kể cả gói con nếu có. Gói con mới `internal/auth/xxx` (thử bằng nhánh tạm như TC-08) rơi vào dòng `auth`, không rơi vào cả hai, không rơi khỏi cả hai

### TC-10 [#5 bước đo thời gian] pre: Run `$S`
STEP: Tìm bước `-p 1 -count=3 ... TimingEqualized$` trong log
EXPECT: Vẫn chạy cho cả `./internal/auth/` và `./internal/course/` (mỗi lần ở dòng matrix của gói đó, hoặc một lần gộp); `-p 1 -count=3`, `-tags testroutes`, không `-race`; có dòng `ok` cho cả hai gói; không bị `skipped`. Nếu chạy ở dòng matrix thì dòng "phần còn lại" không chạy nó

### TC-11 [#5 phần tĩnh] pre: Run `$S`
STEP: Xem job static (vet, golangci-lint, sqlc diff)
EXPECT: Vẫn đủ 5 bước: `go vet`, `go vet -tags testroutes`, `golangci-lint`, `golangci-lint --build-tags testroutes`, `sqlc diff` (phiên bản `v2.14.0`, `1.31.1` không đổi); chạy đúng một lần, không lặp ở mỗi dòng matrix

### TC-12 [Lighthouse ngưỡng] pre: `ci.yml`, `frontend/lighthouserc*.json` ở `$S` và ở SHA TRƯỚC
STEP: `git diff <TRƯỚC> $S -- frontend/lighthouserc*.json`; trích mọi `maxNumericValue`, `aggregationMethod`, mức `error/warn/off`, `numberOfRuns`, `throttlingMethod`, `onlyCategories`, `matchingUrlPattern` của MỌI tệp cấu hình (kể cả tệp con a/b). Lập danh sách assertion TRƯỚC (từ `lighthouserc.json` + `lighthouserc.devtools.json`: audit-id × nhóm URL × mức) và SAU (hợp mọi tệp), so từng dòng
EXPECT: Mỗi ngưỡng giữ nguyên (LCP ≤ 2500 median devtools; CLS ≤ 0.1; TBT ≤ 200; `resource-summary:script:size` ≤ 256000; `dev/ui` LCP vẫn `warn`); `numberOfRuns: 3` ở TẤT CẢ tệp; `aggregationMethod: median` giữ; không thêm `off`/hạ `error`→`warn`. **PM 2026-10-13:** `onlyCategories: ["performance"]` ở simulate được phép CHỈ KHI mọi assertion có ở TRƯỚC vẫn được chấm ở SAU (audit có mặt trong `lhr-*.json` simulate, không bị `off`); danh sách SAU ⊇ TRƯỚC. Assertion nào bị mất (kể cả audit không còn trong báo cáo vì nhóm bị cắt) → FAIL

### TC-13 [Lighthouse đủ URL] pre: Như TC-12
STEP: Lấy `collect.url` của mọi tệp simulate; hợp riêng các tệp devtools; đếm trong log `lhr-*.json`/dòng benchmark từ bước "lighthouse benchmark" của mỗi job
EXPECT: Hợp các nửa devtools = đúng 7 URL cũ: `/`, `/chat`, `/threads`, `/inbox`, `/gradebook`, `/settings/llm`, `/dev/ui`; hai nửa không giao nhau; simulate vẫn đủ 7. Số báo cáo: 7 URL × 3 lượt = 21 cho simulate và 21 cho devtools (cộng các nửa), không ít hơn

### TC-14 [Lighthouse quy tắc theo URL] pre: Như TC-12
STEP: Xem `matchingUrlPattern` ở tệp con
EXPECT: `dev/ui` ở nửa nào cũng vẫn dùng nhánh `warn`; 6 URL còn lại `error`; pattern không bị đổi cổng (3310) làm assertion không khớp URL nào (một assertion không khớp URL nào = cổng mất tác dụng → FAIL). Kiểm bằng log: mỗi URL có dòng assert kết quả

### TC-15 [Lighthouse hợp lệ khi chạy] pre: Run `$S`
STEP: Xem log từng job lhci
EXPECT: Mỗi job in `Done running Lighthouse!` và `All results processed`; không `Assertion failed`; cổng 3310 không đụng nhau (mỗi job một runner nên ổn); `CHROME_PATH` đặt ở mọi job

### TC-16 [Một test Playwright đỏ → CI đỏ (mỗi shard)] pre: Nhánh tạm `qc/tmp-ci-red-pw`; spec `frontend/e2e/zzqc-red.spec.ts` có `test('red', () => { expect(1).toBe(2) })` (file mới, đúng quyền QC)
STEP: Với N shard: tạo N spec có tên chia đều (hoặc 10 spec, mỗi lần push chỉ `--list --shard=i/N` gán cho shard i; kiểm bằng TC-02) để ép đỏ ở TỪNG shard, push từng lần; đọc `gh run view --json conclusion,jobs`
EXPECT: Với mỗi lần: shard chứa test đỏ có `conclusion: failure`; `workflow conclusion = failure`; các shard khác có thể xanh. (Lưu ý `retries: 1`: test đỏ cố định vẫn đỏ sau retry.) Xoá nhánh tạm

### TC-17 [Một dòng matrix Go đỏ → CI đỏ] pre: Nhánh tạm `qc/tmp-ci-red-go`
STEP: Thêm `func TestZZRed(t *testing.T){ t.Fatal("red") }` lần lượt vào (a) một gói `internal/course`, (b) một gói `internal/auth`, (c) một gói thuộc "phần còn lại" (ví dụ `internal/today`); mỗi lần một push
EXPECT: Mỗi lần: đúng dòng matrix tương ứng đỏ, workflow đỏ. Dòng khác không che lỗi. Xác nhận không có `fail-fast: false` kèm `continue-on-error` làm job tổng xanh

### TC-18 [Lighthouse vượt ngưỡng → CI đỏ] pre: Nhánh tạm `qc/tmp-ci-red-lh`
STEP: Hạ ngưỡng `lighthouserc.devtools` nửa A xuống `maxNumericValue: 1` rồi riêng lần khác nửa B; một lần đổi simulate `total-blocking-time` xuống `1`
EXPECT: Mỗi lần job lhci tương ứng đỏ → workflow đỏ. Xoá nhánh tạm

### TC-19 [Cổng kiểm hợp đỏ khi thiếu gói (nếu dev có bước kiểm hợp A∪B∪C)] pre: Nhánh tạm; chỉ khi TC-08 thấy có bước kiểm hợp
STEP: Cố ý đổi bộ lọc dòng "phần còn lại" (ví dụ thêm `-e /internal/today` vào `grep -v`) trên nhánh tạm
EXPECT: Bước kiểm hợp đỏ, nêu tên gói bị bỏ sót, CI đỏ. Nếu không có bước kiểm hợp → TC-08/09 phải đủ chứng minh; ghi "không áp dụng" kèm lý do

### TC-20 [Thời gian wall] pre: Run `$S` và các run TRƯỚC ở trên
STEP: `gh run view <id> --json createdAt,updatedAt,jobs` → wall = từ `Changes` bắt đầu đến job cuối kết thúc; ghi số run, SHA, từng job (Go từng dòng, e2e từng shard, lhci từng job), lần chạy lại
EXPECT: Bảng TRƯỚC/SAU đủ số run, wall tổng, job dài nhất. **Ngưỡng cứng (PM 2026-10-13):** lượt chạy đủ (PR vào `main` hoặc đổi `.github/**`, mọi job chạy) có wall ≤ 10 phút VÀ nhanh hơn mốc TRƯỚC cùng loại cache (ấm 13 m 43 s, nguội 14 m 40 s). Mục tiêu 6–8 phút của TL chỉ ghi nhận, không chấm. Chạy ≥ 2 lượt chạy đủ (một cache nguội, một ấm; cache Go ghi lại mỗi lượt); MỌI lượt đều chấm, không bỏ lượt chậm. Vượt 10 phút hoặc không nhanh hơn → FAIL kèm lý do (tổng runner tăng, biên dịch lặp)

### TC-21 [PR vào `main` vẫn chạy đủ (luật #1)] pre: Nhánh tạm `qc/tmp-ci-pr` chỉ đổi `README.md` hoặc `docs/` không (vì `paths-ignore: docs/**`) — dùng thay đổi chỉ ở `frontend/` rồi ở `backend-go/`; mở PR nháp vào `main`
STEP: Xem run `pull_request`; `gh run view --json jobs`
EXPECT: Mọi job (`Changes`, static, mọi dòng matrix Go, `Judge attacks`, `fe-check`, mọi shard e2e, mọi job lhci) chạy (không `skipped`) dù chỉ đổi một phía. Đóng PR, xoá nhánh. (Lưu ý `docs/**` bị `paths-ignore` ở cấp workflow: không tính.)

### TC-22 [Lọc đường dẫn (#1) không bị phá] pre: Push lên nhánh `sprint/*`
STEP: (a) chỉ đổi `backend-go/**` và (b) chỉ đổi `frontend/**` rồi xem run
EXPECT: (a) job frontend (`fe-check`, mọi shard e2e, mọi lhci) `skipped`, job Go chạy hết; (b) ngược lại. Kiểm ở mức "job" **và** ở mức matrix: ma trận bị `skipped` nguyên khối, không mở ra từng dòng rồi lỗi (check đỏ do matrix rỗng)

### TC-23 [Tên check / branch protection] pre: `gh api repos/{owner}/{repo}/branches/main/protection` (hoặc `rulesets`)
STEP: Liệt kê required status checks; đối chiếu tên job ở `$S`
EXPECT: Mọi check bắt buộc (nếu có) tồn tại với đúng tên ở `$S`; tên mới của job matrix/shard được dev ghi trong handoff. Nếu không có protection → ghi "không có". Một check bắt buộc không còn xuất hiện = FAIL (PR không merge được)

### TC-24 [Không bỏ cổng khác] pre: `ci.yml` TRƯỚC vs SAU
STEP: Liệt kê bước (name) mỗi job TRƯỚC (`go`, `judge-attacks`, `fe-check`, `e2e`, `lhci`, `changes`) và SAU, so ánh xạ
EXPECT: Mọi bước TRƯỚC còn một bước tương đương SAU: `ci-changes.sh --selftest`, `ui-antipatterns.sh`, `lint-selftest.sh`, `pnpm lint`, `build`, `build:gate`, `lighthouse benchmark (chẩn đoán)`, `Judge attacks` (`-race -tags integration`, `-timeout 15m`). Không có `continue-on-error`, `if: false`, hay `

### TC-25 [Cache Go (#4) không hỏng] pre: Run `$S`
STEP: Xem bước cache của các dòng matrix
EXPECT: Mỗi dòng matrix khôi phục cache (key theo `go.sum`); nếu mỗi dòng ghi cache cùng khoá `run_id` → không báo lỗi "Unable to reserve cache" gây đỏ (cảnh báo OK, đỏ là FAIL); kích thước ghi thêm mỗi lượt không vượt giới hạn repo 10 GB một cách rõ rệt (`gh cache list` tổng; ghi con số)

### TC-26 [Phạm vi diff] pre: `git diff --stat <SHA-trước> $S`
STEP: Đọc
EXPECT: Chỉ chạm `.github/workflows/ci.yml`, `.github/actions/**`, `frontend/lighthouserc*.json`, `scripts/ci-changes.sh` (nếu cần), doc/handoff. Không sửa test Go/Playwright, `playwright.config.ts`, ngưỡng; nếu có → báo PM nghiêm trọng

### TC-P5-04-03 [AC1 (chỉ ACTIVE)] pre: 1 `PENDING`, 1 `REMOVED`
STEP: liệt kê
EXPECT: không có trong danh sách và `total`

### TC-P5-04-04 [AC1 (không đọc chat)] pre: –
STEP: bật `log_statement='all'`, gọi danh sách; `grep -c chat_messages`
EXPECT: 0 truy vấn chạm `chat_messages`

### TC-P5-04-08 [AC3a (EXPLAIN lớp 100)] pre: gieo: **1 lớp 1.000 SV + 19 lớp 100 SV**, ≥ 15 buổi (15.000 + 28.500 `attendance_records`), ≥ 280.000 `study_time_daily`, ≥ 5.000 `participation_events` bằng `generate_series` (script `docs/sprints/7/qc/scripts/seed-perf.sql`), `ANALYZE`
STEP: lấy câu SQL của trang đầu, trang giữa, `filter=attention`, `q=`, hồ sơ một SV cho **lớp 100 SV** (từ `log_statement` hoặc `auto_explain` khi gọi API); `EXPLAIN (FORMAT JSON)`
EXPECT: không `Seq Scan` trên `attendance_records`, `study_time_daily`, `participation_events`, `student_risk`, `student_observations`; mọi truy cập bằng chỉ mục bắt đầu `course_id`

### TC-P5-04-09 [AC3b (lớp 1.000 SV)] pre: dữ liệu TC-08
STEP: gọi `students?limit=50` 30 lần, đo (`curl -w '%{time_total}'`); đo hồ sơ 360
EXPECT: trang đầu p95 ≤ 300 ms; hồ sơ ≤ 3 s

### TC-P5-04-10 [AC3b (hồ sơ dùng chỉ mục)] pre: –
STEP: `EXPLAIN` truy vấn hồ sơ **một** SV lớp 1.000
EXPECT: `Index Scan` theo `(course_id, student_id…)`; không `SET enable_seqscan=off` (QC grep log `SET enable_seqscan` = 0, và `docker compose` không đặt biến)

### TC-P5-04-11 [AC4 (at-risk — bảng ca)] pre: –
STEP: chạy `cd backend-go && go test ./internal/crm -run 'TestRiskTable' -v` và đếm 16 ca ở SRS 9.3 (1: 0,5/0/30 hai tuần → AT_RISK; 2: W2 không có neo thấp → WATCH; 3: chỉ 1 tín hiệu → NONE; 4: neo + hoạt động → AT_RISK; 5: không neo → NONE; 6 → WATCH; 7: tuần nghỉ → NONE; 8: chưa ack → AT_RISK; 9 → NONE; 10: đúng ngưỡng → NONE; 11: `signals_required=1` → AT_RISK; 12: `=3` → NONE; 13: vào lớp giữa kỳ → WATCH; 14: EXCUSED → NONE; 15: `STUDY=null` → AT_RISK; 16 → NONE)
EXPECT: `ok`; đủ 16 ca; QC cũng kiểm tay ca 10, 14, 15 qua dữ liệu seed riêng ở lớp thử (đặt dữ liệu bằng API + `seed-study-time` rồi `crm-risk-run`)

### TC-P5-04-12 [AC4 (neo bắt buộc; thiếu dữ liệu)] pre: lớp thử
STEP: sinh viên chỉ thấp thời gian học (không chuyên cần) 2 tuần; sinh viên có tuần giữa không đủ dữ liệu
EXPECT: cả hai `NONE`/không `AT_RISK`

### TC-P5-04-13 [AC4 (idempotent, `entered_at` giữ)] pre: C `AT_RISK`
STEP: `crm-risk-run` hai lần liên tiếp (hai "đêm"); `select entered_at, computed_at from student_risk where student_id=<C>`
EXPECT: cùng kết quả; `entered_at` **không đổi** (chỉ `computed_at` đổi); rời `AT_RISK` → `entered_at` xoá; vào lại → `now()`

### TC-P5-04-14 [AC4 (REMOVED/PENDING)] pre: –
STEP: `select count(*) from student_risk where student_id in (<removed>,<pending>)`
EXPECT: 0

### TC-P5-04-15 [AC4 (đúng một lần mỗi ngày, chạy bù)] pre: worker
STEP: `$RDS GET ep:crm.risk:done:<yyyymmdd>`; chạy `crm-risk-run`; kiểm `$RDS TTL`; dừng worker qua 02:00 (đặt đồng hồ máy chạy thử không khả thi → dùng `$RDS DEL ep:crm.risk:done:<hôm nay>` rồi bật worker sau 02:00)
EXPECT: khoá có TTL ≈ 48 h; job chạy bù khi worker lên lại, **một** lần (hai bản worker → một lần); lỗi giữa chừng → `DEL` khoá, tick sau thử lại

### TC-P5-04-16 [AC4 (tuần/ranh giới, đếm theo `created_at`)] pre: dữ liệu hoạt động
STEP: kiểm `student_risk.weeks` JSON của C
EXPECT: mỗi tuần T2–CN trọn; hoạt động đếm theo `created_at` (không theo khoảng `id`)

### TC-P5-04-17 [AC5 (lý do bằng lời)] pre: C `AT_RISK`; seed
STEP: `j "$Q/students/$uidC/profile" -H "$TCH" \| jq '.risk'`
EXPECT: `risk:{level:"AT_RISK",sentence,reasons:[{code,text}]}`; câu dạng "Cần chú ý: vắng 3/4 buổi trong 2 tuần gần nhất; học trên EduPilot khoảng 20 phút mỗi tuần." (chuyên cần + thời gian học); số khớp danh sách và tab; không nhắc tên khác, không nêu điểm AI hay nội dung chat

### TC-P5-04-18 [AC5 (WATCH, NONE)] pre: một sinh viên `WATCH`, A `NONE`
STEP: hồ sơ
EXPECT: `WATCH` mở đầu "Đang theo dõi:"; `NONE` không có `sentence`

### TC-P5-04-22 [AC6 (UI)] pre: `playwright-cli` `/class/settings` bằng teacher
STEP: nhóm "Ngưỡng cần chú ý"
EXPECT: thanh/ô có nhãn + giá trị; không câu mô tả dưới tiêu đề

### TC-P5-04-23 [AC7 (Hôm nay)] pre: C `AT_RISK` mới (`entered_at` ≤ 7 ngày)
STEP: `GET /me/today -H "$TCH"`
EXPECT: mục `STUDENT_ATTENTION` (bậc 95) "1 sinh viên mới vào diện cần chú ý · 761987", `Href` `/students?filter=attention&course=…`; khối `attention` liệt kê ≤ 5 sinh viên `AT_RISK` (tên + mã lớp + cụm lý do ngắn) + "Xem tất cả (n)"; biến ≤ 60 s sau khi job tính lại

### TC-P5-04-24 [AC7 (ẩn khi rỗng, vai)] pre: lớp 2 không `AT_RISK`; `$SVA`, `$ADM`
STEP: `GET /me/today`
EXPECT: khối `attention` **không** hiện (không rỗng giả); sinh viên/ADMIN không thấy cả hai

### TC-P5-04-25 [AC7 (tối đa 5)] pre: gieo 8 `AT_RISK` ở lớp thử
STEP: xem khối
EXPECT: ≤ 5 dòng + "Xem tất cả (8)"

### TC-P5-04-26 [AC8 (UI)] pre: `playwright-cli` 1280
STEP: `/students`
EXPECT: ô tìm + chip `Cần chú ý` · `Vắng nhiều` · `Ít hoạt động` (**không** chip `Điểm giảm`); `DataTable` cột: Sinh viên · Chuyên cần ("Vắng 2/9") · Phát biểu · Hoạt động học ("45 phút · 3 câu hỏi") · Rủi ro ("Cần chú ý" đỏ / "Theo dõi" hổ phách / trống); không nút trên dòng; bấm dòng → `/students/[id]`

### TC-P5-04-27 [AC8 (giữ trạng thái, deep link)] pre: –
STEP: gõ tìm + chọn chip → vào hồ sơ → quay lại; mở `/students?filter=attention` từ "Hôm nay"
EXPECT: ô tìm và chip giữ; `?filter=` bật đúng chip

### TC-P5-04-28 [AC8 (trạng thái, ≥ 720 px)] pre: –
STEP: tải chậm; `page.route` 500; tìm không có kết quả; `resize 720 900` và `600 900`; cuộn lớp 1.000 SV
EXPECT: khung xương; lỗi + `Thử lại`; "Không có sinh viên nào khớp." + nút `Bỏ lọc`; ≥ 720 dùng được, dưới đó bảng cuộn ngang trong khung; cuộn 1.000 dòng mượt (ảo hoá: số `<tr>` trong DOM ≪ 1.000)

### TC-P5-04-30 [AC9 (không lộ qua API sinh viên)] pre: C có `AT_RISK` và ghi chú canary
STEP: crawl mọi route của STUDENT bằng `$SVC` (C): `me/*`, `chat/*`, `threads`, `me/today`, `notifications`, `calendar`, `library`; `jq '[..\|objects\|keys[]]\|unique'`
EXPECT: không có khoá `risk`, `risk_level`, `at_risk`, `observation`, `watch`; không câu nhận định

### TC-P5-04-31 [AC9 (không lộ qua AI)] pre: `$SVC`
STEP: hỏi chat "Em có bị coi là cần chú ý không?", "Giảng viên ghi gì về em?"; xem payload LLM (`/_test/llm/stats` hoặc log provider giả)
EXPECT: câu trả lời thông thường, không tiết lộ; payload không có `risk`/`AT_RISK`/ghi chú

### TC-P5-04-32 [AC9 (kiến trúc)] pre: repo
STEP: `cd backend-go && go test ./internal/crm ./internal/contract -run 'TestSensitiveQueriesOnlyFromCRM\|TestStudentNeverSeesRisk' -v; go test ./internal/agent -run TestAgentHasNoRiskOrObservationAccess -v`
EXPECT: `ok`

### TC-P5-04-33 [AC10 (lớp mới)] pre: `$CN` không buổi/không dữ liệu
STEP: `GET $CNQ/students -H "$TCH"`
EXPECT: đủ sinh viên, số 0, `risk.level='NONE'`; chuyên cần "—" khi chưa ghi buổi nào (UI); không chia cho 0; không lỗi

### TC-P5-04-34 [AC10 (job chưa chạy; lớp lưu trữ)] pre: `student_risk` trống; lớp `ARCHIVED`
STEP: danh sách; UI
EXPECT: `NONE`, `computed_at` rỗng, giao diện không hiện "Rủi ro"; lớp lưu trữ đọc được (chỉ đọc)

### TC-P5-04-35 [UI] pre: frontend
STEP: `bash scripts/ui-antipatterns.sh; echo $?`; `$PW students.spec.ts -g 'students list filters search keep state\|students deep link filter\|students states'`; `$PW a11y.spec.ts -g students`; `$PW class-join.spec.ts -g 'risk thresholds'`; `$PW today.spec.ts -g 'staff attention block'`
EXPECT: `0`; xanh

### TC-P5-05-02 [AC1 (audit mỗi lần gọi)] pre: `select count(*) from audit_log where entity='student_profile' and entity_id='$uidC'` = n
STEP: gọi `profile` 3 lần bằng `curl`
EXPECT: n+3: mỗi lần gọi 1 dòng (`action='view'`, `actor_id`, `course_id`); nội dung ghi chú không có ở audit

### TC-P5-05-03 [AC1 (UI: mở 3 lần = 3 request, 3 dòng audit)] pre: `playwright-cli`, teacher
STEP: mở `/students/$uidC` → quay lại → mở lại → quay lại → mở lại; đếm request `GET …/profile` trong `network`; so dòng audit
EXPECT: đúng 3 request (không cache: `staleTime 0`, `refetchOnMount always`, `gcTime 0`) và 3 dòng `audit_log`

### TC-P5-05-04 [AC1 (fail-closed)] pre: cài trigger tạm lỗi ở `audit_log` insert (DB test)
STEP: `profile`
EXPECT: `503`, **không** trả dữ liệu; gỡ trigger

### TC-P5-05-05 [AC1 (đọc ghi chú)] pre: có ghi chú
STEP: `GET …/observations`
EXPECT: `action='view_observations'` thêm 1 dòng audit; `audit_log` không chứa nội dung ghi chú (grep canary `CANARY-OBS-4F` = 0)

### TC-P5-05-07 [AC1 (không đọc chat)] pre: –
STEP: `log_statement='all'`, gọi `profile`; grep `chat_messages`
EXPECT: 0

### TC-P5-05-12 [AC3 (UI)] pre: `playwright-cli`, tab `Ghi chú`
STEP: quan sát; gõ dở; offline; online
EXPECT: `PrivateMark` "Chỉ giảng viên/TA thấy"; ô nhập tự lưu nháp (`obs:<uid>`); không mất chữ khi mạng lỗi; nút `Lưu` hoàn lại + `Thử lại`

### TC-P5-05-14 [AC4 (UI hồ sơ)] pre: `playwright-cli` 1280, `/students/$uidC`
STEP: quan sát đầu trang và tab
EXPECT: đầu trang: tên, MSSV, lớp, `StudentRiskSummary` **chỉ khi** `WATCH`/`AT_RISK`, hành động phụ `Thêm ghi chú`, `Nhắn sinh viên` (không nút chính thứ ba); `Tabs`: Tổng quan (đoạn tóm tắt bằng lời + danh sách `term/value`; **không** 6 thẻ KPI), Chuyên cần (bảng buổi × trạng thái), Điểm ("Điểm cộng đã ghi" = tổng điểm sự kiện tham gia; rỗng "Lớp chưa có công thức điểm đã xác nhận." + `Xác nhận công thức` cho TEACHER), Hoạt động học (biểu đồ nhỏ xu hướng phút học theo tuần + số câu hỏi/bài đăng mỗi tuần — chỉ ở đây có biểu đồ), Ghi chú

### TC-P5-05-15 [AC4 (trạng thái, nhãn)] pre: –
STEP: tải chậm; lỗi; `resize 720 900`; `eval innerText`
EXPECT: `<PageState>` đủ; dùng được ≥ 720 px; chữ "Tuần trước" ở khối hoạt động; không câu mô tả dưới tiêu đề khối

### TC-P5-05-16 [AC5 (AI tóm tắt, một lần gọi, không danh tính)] pre: teacher; provider giả ghi thân yêu cầu
STEP: `POST …/students/$uidC/ai-summary`; `GET $GW/api/v1/_test/llm/stats` trước/sau; xem thân yêu cầu
EXPECT: đúng **1** lời gọi `llm.Structured` (task `INSIGHT`, làn `NEAR_REALTIME`, schema `{summary,strengths[0–3],concerns[0–3],suggestions[0–3]}`); thân yêu cầu chỉ số liệu (đếm chuyên cần, phát biểu, phút học 4 tuần, câu hỏi/bài đăng mỗi tuần, mã lý do); **không** tên, MSSV, email, nội dung ghi chú, nội dung chat (grep tên/MSSV/email của C = 0); `masked_count = 0`

### TC-P5-05-17 [AC5 (không lưu tới khi `Lưu`)] pre: –
STEP: gọi `ai-summary`; `select count(*) from student_observations` trước/sau
EXPECT: `200 {draft}` không lưu gì; UI hiện bản nháp nhãn `AI` trong ô sửa được

### TC-P5-05-18 [AC5 (lưu, đã sửa)] pre: –
STEP: lưu bản gốc → `kind='AI_SUMMARY'`, `ai_edited=false`; sửa chữ rồi lưu → `ai_edited=true`
EXPECT: nhãn "Nhận xét tổng hợp · AI" / "· Đã sửa bởi giảng viên"

### TC-P5-05-19 [AC5 (mô hình bịa tên)] pre: provider giả trả bản nháp chứa tên một sinh viên trong roster
STEP: `ai-summary`
EXPECT: tên bị `Redact` (`[đã ẩn]`) trong phản hồi

### TC-P5-05-20 [AC5 (nháp AI ở máy)] pre: –
STEP: nhận nháp, tải lại trang
EXPECT: nháp AI còn (`aisum:<uid>`)

### TC-P5-05-21 [AC6 (lỗi AI)] pre: provider giả chết / quá tải / chậm
STEP: `ai-summary`
EXPECT: `503 LLM_UNAVAILABLE` hoặc `OVERLOADED` + `retry_after`; UI "Chưa soạn được nhận xét. Thử lại" + `Thử lại`; chữ ghi chú đang gõ không mất; quá 20 s → `504`

### TC-P5-05-22 [AC6 (giới hạn, không dữ liệu, vai)] pre: –
STEP: gọi 11 lần trong 1 giờ; sinh viên lớp mới không dữ liệu (`$CN`); `$SVA`, `$ADM`
EXPECT: lần 11 `429 RATE_LIMITED`; `409 NOT_ENOUGH_DATA` và **không** gọi LLM (`/_test/llm/stats` không tăng); `403` ×2

### TC-P5-05-24 [AC7 (sinh viên đọc, hỏi tiếp)] pre: –
STEP: `$SVC` mở `/chat`; `Hỏi tiếp`
EXPECT: thấy tin nhãn "Giảng viên · …"; `Hỏi tiếp` → ticket `OPEN` ở `/inbox` của giảng viên

### TC-P5-05-26 [AC7 (không lộ nhãn rủi ro)] pre: C `AT_RISK`
STEP: grep tiếng "rủi ro", "cần chú ý", nhãn, lý do at-risk trong: tin chat của C, `notifications.title/body`, thư Mailpit, JSON mọi route sinh viên
EXPECT: 0 chỗ

### TC-P5-05-27 [AC7 (trần/ngày, trạng thái, idempotent)] pre: –
STEP: 4 `outreach` trong 1 ngày tới C (nhiều nhân viên cộng lại); sinh viên `REMOVED`/`PENDING`; cùng `Idempotency-Key` hai lần
EXPECT: tin thứ 4 `429 OUTREACH_LIMIT`; `409 STUDENT_NOT_ACTIVE`; lần hai cùng key → không thêm tin

### TC-P5-05-28 [AC8 (hỏi AI về ghi chú)] pre: C có ghi chú canary và `AT_RISK`
STEP: `$SVC` hỏi "Giảng viên ghi gì về em?", "Em có bị coi là cần chú ý không?"
EXPECT: câu mẫu / trả lời học thuật thông thường (không tiết lộ), không `[[…]]`; ghi chú và nhãn không nằm trong lời nhắc, ngữ cảnh RAG hay kết quả tool

### TC-P5-05-29 [AC8 (kiến trúc)] pre: repo
STEP: `cd backend-go && go test ./internal/agent -run 'TestStudentAsksAboutOwnNotes\|TestNoToolReadsObservations' -v`
EXPECT: `ok`

### TC-P5-05-30 [AC9 (mất mạng / đồng thời)] pre: `playwright-cli`
STEP: gõ ghi chú rồi offline; hai giảng viên cùng sửa một ghi chú (hai ngữ cảnh)
EXPECT: nháp giữ; `Lưu` hoàn lại + `Thử lại`; sửa đồng thời → `409` và hiện bản của người kia để chọn

### TC-P5-05-31 [AC9 (mở hồ sơ khi mạng lỗi)] pre: `page.route` 500 / offline
STEP: mở `/students/$uidC`; kiểm `audit_log`
EXPECT: `PageState` lỗi (nói dữ liệu không mất, có `Thử lại`); **không** dòng audit mới khi dữ liệu chưa trả về

### TC-P5-05-32 [AC10 (mock)] pre: repo
STEP: `grep -rn "@/mock" frontend/src/features/students \| wc -l`; `pnpm -C frontend build && pnpm -C frontend lint`; mở `/` (Hôm nay staff)
EXPECT: `0`; xanh; Hôm nay staff không còn "Lớp cần chú ý" giả

### TC-P5-05-33 [UI] pre: –
STEP: `bash scripts/ui-antipatterns.sh; echo $?`; `$PW students.spec.ts` (toàn bộ)
EXPECT: `0`; xanh; ghi số test, flaky

### TC-P5-06-03 [AC1 (rỗng, ngoài lớp)] pre: sinh viên mới không điểm danh; `PENDING`/`REMOVED`/ngoài lớp
STEP: gọi
EXPECT: `recorded:0, sessions:[]` (không lỗi); `403` mỗi ca

### TC-P5-06-06 [AC2 (crawl không canary)] pre: B có ghi chú canary `CANARY-OBS-4F`, `AT_RISK` (hoặc C)
STEP: bằng tài khoản sinh viên: duyệt đủ route STUDENT (`me/*`, `chat/*`, `threads`, `me/today`, `notifications`, `calendar`, `library`, SSE `/events` 30 s); lưu mọi phản hồi vào `crawl/`; `grep -rlE 'CANARY-OBS-4F\|"risk\|"observation\|at_risk\|Cần chú ý:\|Đang theo dõi:' crawl/`
EXPECT: 0 tệp; không khoá `risk*`, `observation*`, `at_risk`, `note` (ngoài `note` của chính tin chat)

### TC-P5-06-07 [AC2 (duyệt UI)] pre: `playwright-cli` đăng nhập C
STEP: duyệt `/`, `/me`, `/chat`, `/threads`, `/settings`, `/calendar`; `eval innerText` mỗi trang
EXPECT: không câu nhận định rủi ro, không canary

### TC-P5-06-08 [AC3 (nội dung `/me`)] pre: `playwright-cli`, `$SVB`, 1280
STEP: mở `/me`
EXPECT: tiêu đề "Kết quả của tôi" + bộ chọn lớp nếu học nhiều lớp; (1) **Chuyên cần**: một câu ("Bạn có mặt 7/9 buổi, vắng 2.") + danh sách buổi vắng/muộn (ngày, buổi); (2) **Điểm cộng đã ghi**: một câu ("Phát biểu 2 lần · +0,50.") + sự kiện gần đây; (3) cuối trang **Thời gian học**: xu hướng nhỏ 8 tuần (`TrendChart` cao ≈ 96 px) + "Tuần này: 45 phút"; số thập phân dùng dấu phẩy

### TC-P5-06-09 [AC3 (không giả)] pre: –
STEP: tìm: điểm thành phần, giải trình điểm, "nếu cuối kỳ được…", việc sắp tới có điểm, ô giả, dữ liệu mô phỏng
EXPECT: **không** hiển thị (không mock)

### TC-P5-06-10 [AC3 (không gamify)] pre: –
STEP: `playwright-cli eval "document.body.innerText + document.body.innerHTML"` \| `grep -ciE 'streak\|rank\|progress ring\|confetti\|xếp hạng\|huy hiệu\|badge\|cố lên\|tuyệt vời'`; tìm `<progress>`, `role=progressbar`, thẻ KPI (lưới ≥ 3 ô số lớn)
EXPECT: `0`; không vòng/thanh tiến độ, không so sánh với bạn học, không câu động viên, không thẻ KPI

### TC-P5-06-11 [AC4 (khớp AI)] pre: `$SVB`
STEP: mở `/me` ghi số; hỏi chat "Em đã nghỉ mấy buổi và được cộng bao nhiêu điểm phát biểu rồi ạ?"
EXPECT: `/me` và chat cho cùng số: 2 buổi vắng; +0,50

### TC-P5-06-12 [AC4 (cập nhật ≤ 60 s)] pre: giảng viên (ngữ cảnh khác) ghi thêm 1 phát biểu cho B ở `/attendance`
STEP: tải lại/làm mới `/me`; hỏi lại chat
EXPECT: ≤ 60 s: `/me` và chat đều +0,75; không số cũ (dữ liệu điểm danh không cache phía server: `Cache-Control`/ETag trên `…/me/attendance` — kiểm header; `no-store` hoặc dữ liệu luôn mới)

### TC-P5-06-13 [AC5 (thời gian học của mình)] pre: `$SVB` đã `Đã hiểu`
STEP: `j "$Q/me/study-time?weeks=8" -H "$SVB"`
EXPECT: `{weeks:[{week_start,minutes}],this_week_minutes,tracking:{enabled,locked}}`; chỉ của mình; qua guard lớp

### TC-P5-06-14 [AC5 (rỗng; tắt)] pre: sinh viên mới chưa có dữ liệu; `tracking.enabled=false`
STEP: mở `/me`
EXPECT: mảng rỗng → khối **không hiển thị** (không "0 phút" giả); tắt → khối hiện "Đang tắt" thay biểu đồ

### TC-P5-06-15 [AC5 (không phán xét)] pre: tuần thấp (C: tuần 7–9 < 60 phút)
STEP: `/me` của C
EXPECT: không "mục tiêu", không so sánh, không tô đỏ tuần thấp (kiểm màu `getComputedStyle` của cột/điểm thấp = màu thường)

### TC-P5-06-16 [AC6 (375 px)] pre: `resize 375 812`, `/me`
STEP: đo `scrollWidth`, `getBoundingClientRect`
EXPECT: không tràn ngang; một cột; vùng chạm ≥ 44 px; biểu đồ co theo bề rộng; chữ Việt dài không vỡ bố cục

### TC-P5-06-17 [AC6 (a11y, Lighthouse)] pre: –
STEP: `$PW a11y.spec.ts -g me`; `cd frontend && pnpm exec lhci autorun` (route `/me`, mobile)
EXPECT: 0 vi phạm nghiêm trọng; LCP ≤ 2,5 s, JS ≤ 250 KB gzip (ngân sách `UX.md` mục 3)

### TC-P5-06-18 [AC7 (trạng thái)] pre: –
STEP: tải chậm; lớp chưa điểm danh; `page.route` 500; offline sau khi đã tải
EXPECT: khung xương; "Chưa có buổi nào được điểm danh." (không nút); lỗi nói chuyện gì xảy ra + `Thử lại`; offline giữ nội dung cũ + dải mất mạng

### TC-P5-06-19 [AC8 (phân quyền route)] pre: –
STEP: giảng viên/TA/ADMIN mở `/me`
EXPECT: được chuyển về `/`

### TC-P5-06-20 [AC8 (hai lớp không lẫn)] pre: A ghi danh lớp 1 và 2
STEP: đổi lớp ở bộ chọn; so số với DB từng lớp
EXPECT: số đổi theo lớp; không lẫn

### TC-P5-06-21 [AC9 (mock)] pre: repo
STEP: `grep -rn "@/mock" frontend/src/features/me \| wc -l`; `pnpm -C frontend build && pnpm -C frontend lint`
EXPECT: `0`; xanh

### TC-P5-06-22 [AC10 (`/settings` công tắc theo lớp)] pre: A ghi danh lớp 1 (cho tắt), lớp 2 (không cho tắt), đã `Đã hiểu`
STEP: `/settings` bằng `playwright-cli`
EXPECT: nhóm "Thời gian học" có **một dòng mỗi lớp**: nhãn mã lớp + công tắc "Ghi thời gian học"; lớp 1 đổi được (lạc quan, không toast); lớp 2 **bật và khoá** (`disabled`), tooltip lý do ở chính công tắc, không dòng mô tả

### TC-P5-06-23 [AC10 (hiệu lực)] pre: –
STEP: tắt công tắc lớp 1; `GET /courses/{1}/me/study-time`; `/me` lớp 1; `/me` lớp 2; sai `version`
EXPECT: `tracking.enabled=false`; `/me` lớp 1 "Đang tắt"; `/me` lớp 2 vẫn đủ biểu đồ; sai `version` → `409` hoàn lại + `InlineNotice`

### TC-P5-06-24 [AC10 (chưa ack / vai)] pre: sinh viên chưa `Đã hiểu`; TA/TEACHER/ADMIN
STEP: mở `/settings`
EXPECT: không thấy nhóm "Thời gian học"

### TC-P5-06-25 [AC10 (375 px, class settings)] pre: `resize 375 812`; `/class/settings` lớp 2 (đã có sinh viên)
STEP: –
EXPECT: công tắc cao ≥ 44 px; lớp 2 hiện công tắc "Cho sinh viên tắt đo thời gian học" **khoá**, giá trị "Không"

### TC-P5-06-26 [UI] pre: –
STEP: `bash scripts/ui-antipatterns.sh; echo $?`; `$PW me.spec.ts`; `$PW settings.spec.ts -g 'study time switch per course\|study time locked course'`
EXPECT: `0`; xanh; ghi số test, flaky

### TC-S7-SEED-01 [AC1 (điểm danh)] pre: DB trống
STEP: `node scripts/seed.mjs`; `$PSQL -c "select count(*) from attendance_records where course_id='$C1'"`; `select session_no, attendance_done_at is not null from class_sessions where course_id='$C1' order by session_no`
EXPECT: `270` dòng; buổi 1–9 `attendance_done_at` có giá trị; **buổi 10 (hôm nay 09:00–11:30) chưa điểm danh**

### TC-S7-SEED-02 [AC1 (A, B, C)] pre: –
STEP: `select … from v_attendance_summary` cho `sv.gioi`, `sv.kha`, `sv.nguyco`; so `seed/expected_crm.csv`
EXPECT: A vắng 0; B vắng **2**, muộn 1; C vắng **5**, muộn 1; khớp từng cột csv

### TC-S7-SEED-03 [AC1 (tổng, phát biểu)] pre: –
STEP: `select count(*) from attendance_records where status='ABSENT' and course_id='$C1'`; `select count(*), sum(points) from participation_events where course_id='$C1'`; B
EXPECT: tổng ABSENT trong 18–26; `participation_events` ≈ 60, lệch (A nhiều nhất); **B đúng 2 sự kiện `SPEAK` = +0,50**

### TC-S7-SEED-04 [AC1 (tất định)] pre: –
STEP: seed trên DB trống lần hai (cùng `SEED_RNG`); so số liệu
EXPECT: giống hệt lần một

### TC-S7-SEED-05 [AC1 (test của dev)] pre: repo
STEP: `cd backend-go && go test -tags integration ./internal/integration -run TestSeedCRMExpected -v`
EXPECT: `ok`

### TC-S7-SEED-06 [AC2 (thời gian học 10 tuần)] pre: sau TC-01
STEP: `gateway admin seed-study-time` (cú pháp: xem `Q-QC-S7-01`); `select count(distinct date), count(*) from study_time_daily where course_id='$C1'`
EXPECT: in "N dòng"; `count(distinct date) ≥ 60`; ≈ 70 ngày × 30 SV × 1–3 khu vực

### TC-S7-SEED-07 [AC2 (số mẫu)] pre: –
STEP: phút học theo tuần của C, A, B, lớp 2
EXPECT: C tuần 7–9 dưới 60 phút/tuần; A ≈ 300; B ≈ 150; lớp 2 có 3 tuần

### TC-S7-SEED-08 [AC2 (chặn môi trường; theo #35)] pre: –
STEP: `APP_ENV=production gateway admin seed-study-time …; echo $?`; `APP_ENV=staging`; `APP_ENV=` (rỗng); `APP_ENV=local`; `APP_ENV=dev`; `APP_ENV=test`
EXPECT: production/staging/`local`/rỗng → thoát **khác 0**, nêu lý do; `dev`, `test` → `0`, in "N dòng"

### TC-S7-SEED-09 [AC2 (idempotent)] pre: –
STEP: chạy `seed-study-time` lần hai; so `count(*)` và tổng `active_seconds`
EXPECT: không đổi (không nhân đôi)

### TC-S7-SEED-10 [AC3 (at-risk C)] pre: seed + `gateway admin crm-risk-run`
STEP: `j "$Q/students?filter=attention" -H "$TCH"`; `select level, count(*) from student_risk where course_id='$C1' group by 1`
EXPECT: đúng **1** `AT_RISK` (C) với lý do chuyên cần + thời gian học (câu khớp SRS 4.7.5); đúng **2** `WATCH`; A, B `NONE`; lớp 2 không `AT_RISK`

### TC-S7-SEED-11 [AC3 (Hôm nay)] pre: –
STEP: `GET /me/today -H "$TCH"`; `playwright-cli` `/`
EXPECT: mục `STUDENT_ATTENTION` "1 sinh viên mới vào diện cần chú ý · 761987"; khối `attention` có Sinh viên C

### TC-S7-SEED-12 [AC4 (ticket đủ trạng thái)] pre: seed (luồng thật) + `gateway admin seed-age`
STEP: `j "$Q/tickets?status=ALL" -H "$TCH"`; `curl -s $MP/api/v1/messages \| jq '.messages\|length'`; `select count(*) from escalation_tickets where status='OPEN' and waiting_since < now() - interval '80 hours'`
EXPECT: 5 phần tử, đủ 4 trạng thái (OPEN mới, OPEN quá hạn, CLAIMED bởi teacher, ANSWERED, CLOSED `STUDENT_CONFIRMED`); Mailpit ≥ 1 thư; ≥ 1 ticket `waiting_since` quá 80 giờ; lớp 2 không ticket; **không** ticket của B

### TC-S7-SEED-13 [AC4 (`seed-age` chỉ đổi thời gian)] pre: –
STEP: chụp `escalation_tickets` (bỏ cột thời gian) và `chat_messages.content` trước/sau `seed-age`; chạy `APP_ENV=production gateway admin seed-age; echo $?`
EXPECT: nội dung không đổi (chỉ `created_at`, `answered_at`, `waiting_since` của ticket; `created_at` của tin chat/thread seed); in số dòng đã đổi; production → thoát khác 0

### TC-S7-SEED-14 [AC4 (quá hạn thật sự)] pre: OPEN quá hạn
STEP: chờ job SLA (hoặc `gateway admin run-job ticket.sla`, cú pháp xem `tc-US-P4-05`)
EXPECT: `sla_stage=2`, mục `TICKET` `Overdue=true` (chứng minh `seed-age` lùi cả `waiting_since`, không chỉ `created_at`)

### TC-S7-SEED-15 [AC5 (Threads)] pre: seed
STEP: `j "$Q/moderation/queue" -H "$TCH"`; `select count(*) from post_reports where course_id='$C1'`; ghim; RAG
EXPECT: ≥ 2 phần tử trong hàng chờ; `post_reports` ≥ 5; có: 1 thread 2 báo cáo `OPEN`, 1 thread tự ẩn (3 người khác nhau), 1 bình luận đã `Ẩn bài` có lý do, 1 thread ghim, 1 bài AI `VERIFIED` đã nạp tri thức (hỏi câu tương tự → có nguồn "Hỏi đáp đã được …"); `Hôm nay` có `POST_REPORTED`

### TC-S7-SEED-16 [AC6 (idempotent)] pre: DB đã seed
STEP: chạy `node scripts/seed.mjs` lần hai; đếm: `attendance_records` (270), `participation_events`, ticket (5), `post_reports`, `knowledge_items`, thread
EXPECT: số **không đổi**

### TC-S7-SEED-17 [AC6 (tự chạy, production)] pre: –
STEP: `pnpm dev` trên DB trống; `NODE_ENV=production node scripts/seed.mjs; echo $?`; `grep -n SUPPORT_RESOURCES_VI docker-compose.local.yml`
EXPECT: `pnpm dev` seed đủ dữ liệu của story; production → thoát khác 0; có `SUPPORT_RESOURCES_VI` mô phỏng

### TC-S7-SEED-18 [AC7 (cổng)] pre: repo
STEP: `bash scripts/gate-p4.sh; echo $?`; `bash scripts/gate-p5.sh; echo $?`
EXPECT: `0`; mỗi lệnh in một dòng `PASS`/`FAIL`; `gate-p4.sh` chạy đúng lệnh `P4.md` (+ `-race`); `gate-p5.sh` chạy lệnh `P5.md` (`./internal/crm/... ./internal/activity/...`, `TestToolMyAttendance\|TestToolMyParticipation`, `attendance.spec.ts`); khi **một** lệnh đỏ → script thoát khác 0 (QC cố ý làm đỏ ở nhánh tạm, xoá sau)

### TC-S7-SEED-19 [AC7 (E2E)] pre: –
STEP: `$PW escalation.spec.ts attendance.spec.ts`
EXPECT: xanh; `escalation.spec.ts` phủ F5 (đường chính + hai người cùng `Nhận` + mail lỗi) và F4 kiểm duyệt; `attendance.spec.ts` phủ F7 (đường chính + **bước ngắt mạng** + hai người cùng điểm danh) ở 375 px, dùng tài khoản seed. Đọc tên test để đối chiếu (`--list`)

### TC-S7-SEED-20 [AC8 (không rò)] pre: seed
STEP: `$SVA`, `$SVB`, `$SVC`: chỉ `me/*` của mình (`GET $Q/me/attendance` mỗi người; thử `…/students/{uid}/…` → `403`); `sv.moi` (D, chưa vào lớp): `GET /me/today`, `GET $Q/me/attendance`
EXPECT: A/B/C thấy đúng của mình; D không thấy dữ liệu lớp nào (`403`); TA thấy lớp 1 và **không** thấy lớp 2 (`403`/`404`); ADMIN `403` ở ticket/hồ sơ/điểm danh; canary (ghi chú, ticket) không ở phản hồi nào dành cho sinh viên (crawl như TC-P5-06-06)

### TC-S7-SEED-21 [AC8 (test của dev)] pre: repo
STEP: `cd backend-go && go test -tags integration ./internal/integration -run 'TestSeedIsolation\|TestSeedAdminForbidden' -v`
EXPECT: `ok`

### TC-S7-SEED-22 [AC9 (seed gãy giữa chừng)] pre: DB trống
STEP: chạy `node scripts/seed.mjs`, `kill -9` tiến trình khi đang bước điểm danh (theo log); chạy lại; `select count(*) from attendance_records where course_id='$C1'`
EXPECT: `270`; tiếp tục từ chỗ dở; không tạo đôi, không sót; kết thúc cùng số liệu như chạy một lần (đối chiếu TC-01…03)

### TC-S7-SEED-23 [AC9 (lỗi API)] pre: chặn gateway giữa seed (`docker compose stop gateway` khi đang chạy)
STEP: quan sát
EXPECT: in bước, mã lỗi, **dừng với mã khác 0** (không nuốt lỗi)

### TC-S7-SEED-24 [AC10 (cờ đo của hai lớp)] pre: seed đã chạy
STEP: `$PSQL -c "select class_code, allow_study_time_opt_out from courses order by class_code"`; `select count(*) from user_settings where track_study_time=false`; `j -X PUT "$Q/settings" -H "$TCH" -d '{"allow_study_time_opt_out":false,"version":1}'`; cùng lệnh bằng `$TA_`, `$SVA`, `$ADM`
EXPECT: `761987 \| t`, `761988 \| f`; `0`; `409 COURSE_HAS_STUDENTS`; `403` ×3

### TC-S7-SEED-25 [AC10 (A hai dòng; chạy lại)] pre: –
STEP: `/settings` của `sv.gioi`; seed chạy lại
EXPECT: hai dòng (lớp 1 đổi được, lớp 2 bật và khoá); seed chạy lại không đổi hai giá trị, không lỗi (409 được coi là đã áp dụng)

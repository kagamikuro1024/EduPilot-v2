# QC — báo cáo cổng P4 (escalation, thông báo, thư, kiểm duyệt Threads, an toàn con người)
Ngày 2026-10-16. Commit: `sprint/7-p4-p5` @ `0f9ff865`; CI `38782040813` **xanh** (10 ph 59 s) — các lượt CI trước đó của nhánh (`a908fd9` …) đỏ ở Lighthouse, đã sửa. Bộ kiểm: `gate-P4.md` (G4-01…32), `tc-US-P4-01…07.md`, `.claude/skills/gate/SKILL.md`. Stack `edupilot100` **sạch** dựng từ nhánh (`down -v`, `up --build --profile ingest`, `--scale gateway=2`, docling thật, seed đầy đủ — xem `report-US-S7-SEED.md`).

## Kết luận cổng P4: **ĐẠT CÓ ĐIỀU KIỆN** — 0 dòng FAIL; 4 điều kiện (mục 5)

## 1. Rà soát thay đổi spec / TC từ APPROVED (yêu cầu của PM)
Phạm vi: `git log -p` của `docs/specs/**` (mọi `FEAT-*`) và `docs/sprints/7/qc/tc-*`.
- **`docs/sprints/7/qc/tc-*`**: 4 commit (`e3f49272`, `aa03369f`, `27b4c93a`, `aa7c71af`), toàn bộ của QC ở pha 1 (ngày 13/10); **không có commit nào sửa TC sau khi viết** bởi vai khác. Trong pha 2 QC chỉ sửa **một** TC: `TC-S7-SEED-08` (16/10, có ghi "Lịch sử sửa TC", dẫn `#35`).
- **`docs/specs/FEAT-escalation-notify`** (APPROVED `5c7f14fb`, 13/10) — sau đó 4 commit: `383acb80` (#15 #16 #17), `2250c229` (#20, số thật #22), `803393b7` (không số), `30f81cae` (không số). Các số đã dẫn đều có dòng **ACCEPTED** trong `proposals.md` (#15, #16, #17, #22; #24, #25 qua `803393b7`).
- **`docs/specs/FEAT-crm-360`** (APPROVED `0c35bf6a`, 13/10) — sau đó 4 commit: `f22f4ef2` (#20), `803393b7` (không số), `43b45949` (#31), `e8fba173` (#34, #35). #20, #31, #34, #35 đều ACCEPTED. Các `FEAT-*` khác: 0 commit kể từ 13/10.
- **Hai commit sửa spec KHÔNG ghi số proposal trong thông điệp** (vi phạm quy trình "commit phải ghi số proposal", `CLAUDE.md` mục đội herdr):
  1. `803393b7` "US-P5-02: vòng 2 — câu ghép nhiều ý … (AC14, TC-38)" — sửa `FEAT-crm-360/{QUESTIONS,SRS,US}.md` và `FEAT-escalation-notify/{SRS,US}.md`: nội dung diff là đổi số migration `00014 → 00015` (đúng **#25 ACCEPTED**) và đoạn SRS 4.4.1 / 4.10.7 / bảng sự kiện về `notify.SendTx` (đúng **#24 ACCEPTED**); AC14 đã có từ trước khi APPROVED (`git show 0c35bf6a:…/US.md` có AC14). Nội dung phủ bởi #24, #25; thiếu số trong thông điệp.
  2. `30f81cae` "US-P4-07: tài liệu AC7" — thêm `Q-crisis-hyperbole` vào `QUESTIONS.md` và mở rộng SRS mục 7 ("giới hạn đã biết … đủ ba ý — US-P4-07 AC7"). Không có proposal; nội dung là thực thi **AC7 đã duyệt** (tài liệu), không đổi hành vi / AC. PM xác nhận hoặc bổ sung số.
- **Test cũ bị sửa** (`git diff --diff-filter=M 1518f83.. -- '*_test.go' 'frontend/e2e/*'`): 61 tệp Go + 10 tệp e2e (+ 6 ảnh `visual.spec` — chụp lại theo #11/#29…, chỉ ảnh). Tôi lần theo từng tệp tới commit và số proposal: **tất cả** đều dẫn tới #4, #5, #8, #10, #11, #13, #14, #18, #19, #20, #21, #23, #24, #25, #26, #27, #28–#34 (đều ACCEPTED) **trừ ba tệp** `contract/rig_test.go`, `httpapi/pg03_routes_test.go`, `llmconfig/api_test.go` (mỗi tệp **một dòng**: `MigratedPostgresURL` → `SharedMigratedPostgresURL`) do commit `a5c13b99` "fix(testutil) … (BUG-S7-02)" — số của lỗi PM giao, không phải số proposal; hành vi giữ nguyên. Ghi chú: `0d13ff62` (BUG-S7-01, #19 / #23 / #24) đổi ≈ 40 test chờ đồng hồ thật: bỏ 11 khẳng định thời gian thật (`require.Less(took, 2*time.Second)` …) và thêm 8 khẳng định tất định (`testutil.TimingOn`); phù hợp #19 (không nới ngưỡng). Hai lỗi quy trình đã ghi từ trước ở #10 và #21 (sửa test trước khi PM quyết).
- Không test nào bị **xoá** (`diff` danh sách `func Test…` Go: 0 hàm mất; tiêu đề `test(` Playwright: 0 mất).

## 2. Số liệu (mục D)
| Số liệu | Kết quả |
| --- | --- |
| Go top-level `Test…` | 1.477 (`1518f83`) → **1.927**, 0 hàm bị xoá |
| Playwright `test(` | 279 → **394** khai báo; lượt chạy đầy đủ (`playwright test`, mọi project): **692 passed, 13 failed, 369 skipped, 0 flaky** (8,2 phút). 13 failed đều là `visual.spec.ts` (ảnh chụp lệch dựng máy macOS 27, dev đã ghi); **CI Linux xanh** ở cùng commit |
| `go test -race -count=1 -tags testroutes ./...` (timing bật, không `EP_SKIP_TIMING`) | lần 1: 46 gói ok, **1 test đỏ** `internal/course` `TestJoinFailureTimingEqualized` (`join_limit_test.go:59`: 0,633 > 0,35 khi máy tải); chạy riêng 4/4 xanh; **lần 2: 47 gói xanh, rc=0**. Chạy riêng đủ ba test của cổng: `TestEscalationToMail`, `TestNotifyAcrossTwoGateways` (`bell_latency_ms=14`), `TestOutboxSurvivesWorkerKill` — ok |
| `go vet -tags testroutes ./...` / `golangci-lint run` | rc=0 / "0 issues" |
| Chuông thật (giảng viên mở `/`, sinh viên gửi câu hỏi ngoài tài liệu) | đổi từ "Thông báo" → "Thông báo, 1 chưa đọc" **≤ 5,0 s** không tải lại (đo từ lúc gửi, gồm chờ ≈ 5 s trả lời của trợ lý — thực tế sớm hơn) |
| Thư trả lời (API `claim` + `answer`) → Mailpit | **2 s**, 1 thư "Giảng viên đã trả lời câu hỏi của bạn", 0 chữ của sinh viên khác |
| Recall khủng hoảng (`crisis-run.sh`) | dương **20/20**, âm **0/17** |
| Volume test (BUG-S7-02) | 18 volume, 2,80 GB → 2,71 GB sau một lượt full Go; 0 container mồ côi |

## 3. Từng dòng cổng
| ID | KQ | Bằng chứng ngắn |
| --- | --- | --- |
| G4-01 | PASS | `go test -race ./internal/escalation/... notify mail thread` trong `gate-p4.sh` (59 s) và trong lượt full; không `DATA RACE` |
| G4-02 | PASS (ghi chú) | `TestEscalationToMail` ok (vị trí `./internal/escalation`, không cần tag `integration` như `gate-P4.md` ghi); test không in `bell_latency_p95_ms` / `mail_latency_p95_s` — số đo ở mục 2 |
| G4-03 | PASS | `TestNotifyAcrossTwoGateways` ok, `bell_latency_ms=14` |
| G4-04 | PASS | `TestOutboxSurvivesWorkerKill` ok |
| G4-05 | PASS | `escalation.spec.ts` 25 passed (desktop), 0 flaky; `threads-moderation` 8, `threads-crisis` 3 |
| G4-06 | PASS | `bash scripts/gate-p4.sh` rc=0 "GATE P4: PASS" (8 bước); thoát `rc=1` khi chèn một lệnh đỏ |
| G4-07 | PASS có điều kiện | vet / lint rc=0; full `-race`: lần 1 một test thời gian đỏ (không thuộc P4), lần 2 xanh; số test không giảm |
| G4-08 | PASS | `internal/contract` ok; `sqlc generate && sqlc diff` rc=0; `TestMigrations_RoundTrip` (up/down/up trên DB trống) ok; sửa hợp đồng chỉ gồm các dòng ghi ở mục 1 |
| G4-09 | PASS | `pnpm lint` sạch; `pnpm build` ok; `ui-antipatterns.sh` rc=0 |
| G4-10 | PASS có điều kiện | mục 1: mọi test cũ bị sửa đều có proposal ACCEPTED, trừ ba dòng helper của BUG-S7-02 (PM) |
| G4-11 | PASS (ghi chú) | chuông nổi ≤ 5 s không tải lại. **Ghi chú**: seed có 126 ticket `OPEN` tự sinh nên thông báo `TICKET_NEW` gộp đã chưa đọc sẵn — giảng viên chưa đọc thì **số** trên chuông không đổi (chỉ chữ gộp đổi); phải "đọc hết" trước mới thấy 0 → 1 (BUG-S1 ở US-S7-SEED) |
| G4-12 | PASS (một phần) | `claim` + `answer` qua API: `200`/`200`; e2e `answer appears in student chat` ("Giảng viên · {tên}", không Hữu ích/Không hữu ích) xanh. QC không chọn được dòng `/inbox` bằng bộ chọn tay (126 dòng, tiêu đề dài) nên bước bấm UI thật dùng e2e |
| G4-13 | PASS | Mailpit: đúng 1 thư trong 2 s |
| G4-14 | PASS | thư chỉ có tên người nhận, câu hỏi của người nhận, tên giảng viên, liên kết; 0 khớp tên / email / MSSV sinh viên khác |
| G4-15 | PASS | API trên stack sạch: STUDENT ×3 và ADMIN `403` ở danh sách và chi tiết ticket; TA / teacher lớp đó `200`; TA chỉ lớp 1 gọi route lớp 2 `403`; canary `CANARY-9K3Z` 0 ở log / payload |
| G4-16 | PASS | hai người `claim` đồng thời: `ta=200`, `teacher=409 TICKET_ALREADY_CLAIMED`; `answer` hai lần cùng `Idempotency-Key` → 1 tin |
| G4-17 | PASS (gián tiếp) | `TestOutboxSurvivesWorkerKill` ok; `DEAD` sau 3 lần + `/observability`: e2e `observability failed mail` xanh và `report-US-P4-04.md` (PASS) — không chạy lại trên stack thật |
| G4-18 | PASS | `TestNotifyAcrossTwoGateways` hai chiều; e2e `bell realtime`, `bell third tab falls back to polling` |
| G4-19 | PASS | `gateway admin run-job ticket.sla`: ticket `OPEN` lùi 192 h → `sla_stage 2`, mục `TICKET` `overdue`; bảng SRS 9.3 trong `go test ./internal/escalation` (xanh) |
| G4-20 | PASS | hàng chờ 2 mục, 5 báo cáo, thread ẩn / ghim / bình luận ẩn có lý do; e2e `threads-moderation` 8 xanh; test thread xanh |
| G4-21 | PASS | 20/20 dương, 0/17 âm (kể cả "từ từ", "kết thúc tất cả các tiến trình con") |
| G4-22 | PASS (Go) | `EXAM_IN_PROGRESS` / giờ thi: test Go của `escalation` / `chat` xanh; không chạy lại tay |
| G4-23 | PASS | tên, MSSV, email sinh viên khác → `422 PII_IN_ANSWER`; email của chính giảng viên `200` |
| G4-24 | PASS | 375×812 thật: `/inbox`, `/chat` không tràn ngang, 0 vùng chạm < 44 px; e2e `inbox 375` xanh |
| G4-25 | PASS | log gateway + worker, `llm_audit`, `audit_log`, `outbox`, payload LLM: 0 khớp `CANARY-OBS-4F`, `CANARY-NOTE-9`, `CANARY-9K3Z`, tên / MSSV / email của sinh viên thử |
| G4-26 | PASS | `tickets?limit=500` → `422`, `cursor=zz` → `422`; không `OFFSET` trong `store/queries/*.sql` |
| G4-27 | PASS | e2e `inbox states`, `bell refresh error`, `observability failed mail` xanh; UI 375 px ở G4-24 |
| G4-28 | PASS | `TestMigrations_RoundTrip`; `goose` trong `up --build` sạch trên DB trống |
| G4-29 | PASS | 0 từ kỹ thuật (RAG, PII, provider, fallback, trace, confidence, ticket, SLA, risk) ở 6 màn sinh viên (`eval innerText`) |
| G4-30 | PASS | không `fetch(` ngoài `frontend/src/shared/`; không `OFFSET` mới; không thư viện mới (`go.mod`, `package.json` không đổi); không secret trong diff; `float64` ở `internal/crm` chỉ trong chú thích |
| G4-31 | PASS | `grep -rn "@/mock" frontend/src/features/inbox` = 0 |
| G4-32 | PASS (một phần gián tiếp) | F5 (hỏi → chuông → nhận → trả lời → thư) và F4 (báo cáo → ẩn): e2e + API ở trên; nhánh lỗi mail lỗi / hai người cùng nhận / báo cáo → ẩn: e2e. F14: `TICKET` `overdue` và `POST_REPORTED` thấy trên `Hôm nay` thật; `STAFF_ANSWER_UNREAD`, `COURSE_TICKETS_STALLED` chỉ qua test Go / e2e |

## 4. Kiểm chéo và luật mở rộng
Không `fetch` trần, không `OFFSET`, không ghi đĩa cục bộ mới, lời gọi LLM mới có `task` (`UTILITY` cho `ai-summary`); POST có tác dụng phụ của P4 (`escalate`, `claim`, `answer`) nhận và tôn trọng `Idempotency-Key`. Màn mới: `/inbox`, chuông, `/threads` kiểm duyệt, mục `/observability` có tải / rỗng / lỗi (e2e). `ui-antipatterns.sh` rc=0.

## 5. Điều kiện của kết luận "ĐẠT CÓ ĐIỀU KIỆN"
1. **Thời gian**: `internal/course` `TestJoinFailureTimingEqualized` đỏ một lần khi chạy cả bộ `-race` có tải (dev xem ngưỡng 0,35 — test sprint 2, không thuộc P4).
2. **Quy trình**: hai commit sửa spec không có số proposal (`803393b7`, `30f81cae`) và ba dòng test của BUG-S7-02 — PM xác nhận hoặc bổ sung số.
3. **`visual.spec.ts`** 13 ca đỏ trên macOS dev (đã có CI Linux xanh) — QC không có nền Linux.
4. **Ba dòng kiểm gián tiếp** (G4-17, G4-22, G4-32 một phần) dựa trên test Go / e2e / report story, không chạy lại bằng tay trên stack thật.

## 6. "Bạn tự kiểm" (chủ dự án) và việc ghi nhận
Hai trình duyệt thật: sinh viên hỏi ngoài tài liệu → giảng viên thấy chuông nổi ≤ 5 s; giảng viên `Nhận`, trả lời → sinh viên thấy "Giảng viên · …"; Mailpit đúng 1 thư; đọc thư không lộ dữ liệu người khác. Ghi `docs/thesis-notes/P4.md` (đã có giới hạn nhận diện khủng hoảng, US-P4-07 AC7).

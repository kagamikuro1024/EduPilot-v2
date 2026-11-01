# QC cổng P6 — Sổ điểm và điểm cuối kỳ từ quy chế môn học (M7) + seed sprint 8
Nguồn: `docs/phases/P6.md` (cổng nghiệm thu, "Bạn tự kiểm"), `docs/specs/FEAT-gradebook/` (US v1.1 + SRS), `docs/sprints/8/prompts/qc-tc.md`, `.claude/skills/gate/SKILL.md`. Chạy ở pha 2 sau khi có handoff đủ US-P6-01…09 + US-S8-SEED (+ US-S8-UI); mỗi dòng ghi PASS/FAIL + 10 dòng đầu ra. **Cổng PASS chỉ khi mọi dòng PASS**, mọi `tc-US-P6-01…09.md`, `tc-US-S8-SEED.md`, `tc-US-S8-UI.md` PASS, **và chủ dự án tự tính tay A, B, C** (G6-30, việc của chủ dự án; QC không thay).

## A. Lệnh cổng của phase (`P6.md`)
| ID | Lệnh | Kết quả mong đợi |
| --- | --- | --- |
| G6-01 | `cd backend-go && go test -race -count=1 ./internal/grade/... -v \| tee /tmp/g6-01.log; grep -c -- '--- PASS: TestEngineExpected30/' /tmp/g6-01.log` | `ok`, không `DATA RACE`; đúng **30** ca con `PASS` của `TestEngineExpected30` (30/30 khớp CSV); `TestEngineCases` 18 ca con |
| G6-02 | `! grep -rn "float64" backend-go/internal/grade --include=*.go \| grep -v _test.go; echo rc=$?` và quét mở rộng của TC-P6-02-14, 15 | rc=0; không `float32/64`, `math.Round/Floor/Ceil`, `ParseFloat`, `RoundBank`, `StringFixedBank`; không điểm nào đi qua `float64` ở nơi khác |
| G6-03 | `cd backend-go && go test -count=1 ./internal/grading -run TestExtractGradeScheme -v` | `ok`; quy chế seed: đúng toàn bộ trọng số (20/20/60) và luật |
| G6-04 | `curl -s -X POST $Q2/gradebook/finalize -H "$TCH" -H "$(idem)" -o /dev/null -w "%{http_code}"` (lớp 2 seed gốc) | `409` |
| G6-05 | `bash scripts/gate-p6.sh; echo rc=$?` | `rc=0`; in một dòng `PASS`/`FAIL` mỗi lệnh; có bước nhắc "chủ dự án tự tính tay A, B, C"; **tự thử đỏ** (TC-P6-SEED-25): thêm `float64` → gate đỏ; đổi làm tròn → gate đỏ |
| G6-06 | `pnpm -C frontend exec playwright test gradebook.spec.ts --workers=2` | xanh toàn bộ; ghi số test, flaky (≥ 3 flaky → FAIL); mọi tên `-g` nêu ở US có ≥ 1 test (TC-P6-SEED-27) |
| G6-07 | `go vet ./... && golangci-lint run && go test -race ./...` (toàn bộ, không `-run`; `scripts/go-test-all.sh`) | xanh; số test cũ không giảm so với mốc đầu sprint (QC đo lại ở đầu pha 2: `go test -list` đếm `^Test`; Playwright `--list`) |
| G6-08 | `go test ./internal/contract/...`; `cd backend-go && sqlc generate && sqlc diff`; `goose up/down/up` từ DB trống (`00016`) | xanh; hợp đồng chỉ cộng; `sqlc diff` rỗng; migration sạch |
| G6-09 | `pnpm -C frontend lint && pnpm -C frontend build`; `bash scripts/ui-antipatterns.sh` | xanh; `rc=0` |
| G6-10 | `git diff <mốc đầu sprint>.. --stat -- '*_test.go' 'frontend/e2e/*' seed/expected_final_grades.csv seed/expected_risk.csv seed/expected_crm.csv docs/specs/FEAT-gradebook/fixtures` | chỉ **thêm** test mới; test cũ chỉ bị sửa nếu có số proposal PM chấp nhận (#1 `lcp.spec.ts`; #5 `check-crm-seed.mjs` ghim `WATCH`); **`seed/expected_final_grades.csv` và `fixtures/` không bị dev/QC sửa** (chỉ BA, theo commit có TLR/proposal); mọi sửa khác → FAIL nghiêm trọng |
| G6-11 | `cd backend-go && go list -deps ./internal/grade \| grep -E 'internal/(activity\|crm\|llm\|grading\|privacy\|agent)\|openai-go'; echo rc=$?` | rỗng; `go test ./internal/grade -run 'TestForbiddenImports\|TestNoFloatInGrade\|TestConfirmOnlyFromHandler'` PASS; tự thử đỏ (TC-P6-02-18, 21) |
| G6-12 | `bash docs/sprints/8/qc/scripts/p6-hand-calc.py A B C sv04 sv05 sv06 sv13 sv09 sv20` | `30/30`, `18/18` khớp, rc=0 — đáp án BA đúng (độc lập với mã của dev) |

## B. Kiểm tay "Bạn tự kiểm" (`P6.md`) — `playwright-cli`
| ID | Việc | Kết quả mong đợi |
| --- | --- | --- |
| G6-13 | **QC tự tính tay A, B, C** (và sv04, sv05, sv13, sv09) bằng máy tính / `p6-hand-calc.py`, so với **giải trình trên màn hình** `/gradebook` | A `9,5 A+`; B `7,0 B`; C `3,0 F`; từng dòng giải trình khớp phép tính tay (`QT₀ 7,75 → +0,50 → 8,25 → HP thô 7,20 → 7,0 → B`); ảnh `shots/abc-gradebook.png`, `abc-explain-{A,B,C}.png` |
| G6-14 | **Quy chế cố tình thiếu quy tắc** (`quy-che-lop2.pdf`, lớp 2) | liệt kê đúng một "chưa rõ: làm tròn"; `Xác nhận công thức` khoá (tooltip lý do); điền "Làm tròn đến 0,1" → mở; xác nhận → banner biến mất, `Chốt điểm` mở |
| G6-15 | **Lớp chưa có quy chế** (lớp 2 seed gốc): banner hiện, `Chốt điểm` khoá (tooltip), `finalize` → `409`, việc trên "Hôm nay", AI nói "chưa có công thức chính thức" + ticket `AUTO_NO_FORMULA` | đúng cả 5 (TC-P6-05-01…41) |
| G6-16 | **Bấm giờ** F10 đường chính theo `DEMO_SCRIPT` bước 6 (10:00–12:00 = 2 phút): đổi lớp → tải quy chế lớp 2 → điền → xác nhận → quay lại → giải trình B → xuất XLSX | ≤ 2:00 (ghi giây thực; chênh > 30 s so với mốc → ghi nhận cho PM, DEMO_SCRIPT) |
| G6-17 | **Chốt → khoá → mở khoá có lý do → sửa → chốt lại v2 → sinh viên bị đổi nhận chuông**; v1 bất biến | đúng (tc-US-P6-08 TC-01…29); băm v1 trước = sau |
| G6-18 | **Đăng nhập sinh viên B** trên 375 px: `/me` điểm đúng, what-if, không điểm nháp; hỏi AI điểm của C → từ chối | đúng (tc-US-P6-09) |

## C. Kiểm chéo bắt buộc (QC.md pha 2 mục 3) — chạy lại bằng TC đã viết
| ID | Mục | TC | Kết quả mong đợi |
| --- | --- | --- | --- |
| G6-19 | **Dữ liệu nhạy cảm:** STUDENT (API trực tiếp, `/me`, chat, Threads) không thấy điểm nháp / cột chưa công bố / điểm học phần trước khi chốt / `unlock_reason` / ghi chú / nhãn rủi ro; canary 0 | TC-P6-09-07…12, 27, 28; TC-P6-SEED-32 | `0` canary (`CANARY-GRADE-7Q`) ở mọi phản hồi/DOM sinh viên |
| G6-20 | **Phân quyền ma trận 20 route × 5 vai** (SRS 2): STUDENT chỉ `me/*`; TA `GET`; TEACHER đủ; ADMIN `403`; không đăng nhập `401`; ngoài lớp (kể cả uuid lớp không tồn tại) `403`, id không phải uuid `404` (#11); ghi/chốt/xuất/xác nhận/nhập chỉ TEACHER | TC-P6-01-36…39; `bash docs/sprints/8/qc/scripts/p6-rbac.sh` | `RBAC: PASS`; `TestAllCourseRoutesGuarded` PASS; mọi route trong `openapi.yaml` |
| G6-21 | **Toàn vẹn điểm:** snapshot bất biến (UPDATE/DELETE/TRUNCATE chặn), chốt = một giao dịch, 409 thiếu điểm → 0 snapshot, đua `PUT` × `finalize` 20 lần không snapshot lẻ, mọi `PUT` điểm có `audit_log` (fail-closed) | TC-P6-01-14…19; TC-P6-08-01…34; TC-P6-06-43…45 | đúng |
| G6-22 | **AI chỉ nháp:** `ConfirmScheme` chỉ từ handler; seed xác nhận bằng TEACHER qua HTTP; mục "chưa rõ" chặn; chỉ TEACHER xác nhận / chốt / xuất / mở khoá | TC-P6-03-25…28; TC-P6-04-08, 23; TC-P6-SEED-03 | đúng |
| G6-23 | **Không LLM tính điểm / thời gian học không vào điểm:** `grep` cổng; `go list -deps`; thử `study_time_daily` rất cao không đổi điểm; prompt trích công thức không chứa số điểm sinh viên | TC-P6-02-14…21; TC-P6-01-30; TC-P6-09-44 | đúng |
| G6-24 | **Idempotency** POST có tác dụng phụ: `extract`, `confirm`, `import commit`, `finalize`, `unlock` (gửi lại cùng khoá → cùng phản hồi, không tạo đôi); thiếu khoá → 4xx | TC-P6-03-38; TC-P6-04-21, 42; TC-P6-07-20, 52; TC-P6-08-04, 43 | không tạo đôi |
| G6-25 | **Phân trang / không `Seq Scan` / N+1:** `…/gradebook`, `…/preview`, `…/history`, `…/snapshots/{v}`, import/export/finalize 1.000 sv: số câu SQL không tỉ lệ số sinh viên; `limit ≤ 100` | TC-P6-06-30…35; TC-P6-07-42; TC-P6-08-33, 34 | p95 trong ngưỡng; không OFFSET |
| G6-26 | **An toàn tệp XLSX:** zip bomb, > 5.000 dòng, hỏng, công thức, injection `=+-@` ở xuất, sha lệch, không đĩa cục bộ | TC-P6-07-01…05, 29…38 | đúng, không panic, gateway không phình RAM |
| G6-27 | **Trạng thái tải / rỗng / lỗi + 375 px** các màn mới (`/gradebook`, `/gradebook/scheme`, `/me` phần điểm); không `fetch` trần; không thư viện lạ | TC-P6-04-37…41; TC-P6-06-37…41; TC-P6-09-46; `git diff <mốc>.. -U0 -- frontend/src \| grep -nE '^\+.*\bfetch\('`; `git diff <mốc>.. -- backend-go/go.mod frontend/package.json` | không `fetch` trần; không thư viện ngoài bảng `ARCHITECTURE.md` (`excelize` đã có) |
| G6-28 | **Migration sạch trên DB trống** `00016` (không `ALTER`), `down` một bước, `up` lại; `PROGRESS.md` ánh xạ | TC-P6-01-01…05 | đúng |
| G6-29 | **Luật chữ UI & không gamify** ở mọi màn mới + 3 màn US-S8-UI: không câu mô tả dưới tiêu đề, không icon Info, không toast "Thành công", dòng ghi chú điểm chính thức đúng 1 lần ở `/gradebook`, `/me`, tệp xuất | TC-P6-06-04; TC-P6-09-13…15; TC-S8-UI-01…20; `ui-antipatterns.sh` | đúng; rc=0 |
| G6-30 | **Chủ dự án tự tính tay A, B, C** (`P6.md` "Bạn tự kiểm", `plan.md` "điều kiện cổng") | – | **KHÔNG TỰ ĐỘNG** — QC ghi "chờ chủ dự án"; cổng chỉ ĐẠT khi chủ dự án xác nhận (QC cung cấp bảng phép tính ở `tc-US-P6-02`) |
| G6-31 | **Docker rác dọn:** `docker volume prune -f` sau mỗi lượt (không đụng volume có tên / llm-eval) | `CONTEXT.md` mục 4 | đã chạy; ghi số volume |

## Kết luận cổng
Điền ở pha 2: số dòng PASS/FAIL; lỗi theo mức; đường dẫn `report-GATE-P6.md`. **PASS chỉ khi 31/31 PASS** (G6-30 chờ chủ dự án).

## Lịch sử sửa
(chưa có)

- 2026-10-31 — **proposal #11 ACCEPTED**: G6-20 "lớp khác `404`" → "`403` (id không phải uuid `404`)".

## Kết quả pha 2 (QC, 2026-10-31, `sprint/8-p6` @ `6711408d`; chi tiết ở `report-US-S8-SEED.md`)
| ID | Kết quả |
| --- | --- |
| G6-01…03, G6-07…11 | PASS (qua `scripts/gate-p6.sh`: `go vet`, `go test -race ./internal/grade/...` 30/30, không `float64`, `TestExtractGradeScheme`/demo, contract, ui-antipatterns; `go-test-all.sh` rc=0 51 gói ở `d563edcf`) |
| G6-04 | PASS (lớp 2 `finalize` → 409 khi chưa có công thức; sau khi QC xác nhận bản 2 → 409 `GRADES_INCOMPLETE`) |
| G6-05 | PASS (`GATE P6: PASS`; tự thử đỏ với `float64` → đỏ) |
| G6-06 | PASS (vòng 2): 54 test xanh; 38 tên `-g` của US đều có test (đã sửa BUG-S8-SEED-3) |
| G6-12 | PASS (`p6-hand-calc.py` 30/30, 18/18; `p6-preview-vs-csv.py` 30/30 trên stack thật) |
| G6-13…15 | QC đã tự tính tay A 9,5 A+, B 7,0 B, C 3,0 F khớp giải trình; lớp 2: một mục chưa rõ → điền → xác nhận qua API thật; **chủ dự án vẫn tự tính tay** |
| Tải 1.000 | `impact` p95 52 ms; `finalize` 0,12 s; `PUT` p95 18 ms; `/me/grades` p95 16 ms; trang sổ p95 19 ms |

## Kết luận cổng P6 (QC, 2026-10-31, `sprint/8-p6` @ `2fea08ec`, CI `41213938391` xanh)
**CỔNG KỸ THUẬT P6: PASS.** Trên stack QC dựng lại từ DB trống: `seed.mjs` rc=0 và idempotent; `check-grade-seed` (30/30, A B C, lớp 2 → 409, `impact` p95 8 ms, số không đổi), `p6-preview-vs-csv.py` 30/30 (MSSV và account), `gate-p6.sh` `GATE P6: PASS` (13 bước, Playwright 54 passed), seed sprint 5–8 xanh; `SEED_RNG=1` chạy trọn (đầu vào khớp theo account; điểm cộng / trừ theo RNG nên 13 sinh viên khác CSV — engine = tính tay 30/30); lớp 2 qua API thật (một mục chưa rõ → điền → xác nhận); tải 1.000 sinh viên: `impact` p95 52 ms, `finalize` 0,12 s, `PUT` p95 18 ms, `/me/grades` p95 16 ms; US-P6-01…09, US-S8-UI, US-S8-SEED đều có report PASS.
**Còn lại cho chủ dự án (không thuộc QC):** G6-13 tự tính tay A (9,5 A+), B (7,0 B), C (3,0 F) bằng máy tính rồi so với giải trình trên màn hình (QC đã tự tính trước, khớp từng dòng); xem lớp 2 demo bước 6. Ghi nhận mở (thấp): BUG-P6-02-1 (thiếu test đột biến làm tròn TLR-1), BUG-P6-05-1 (thiếu test khử trùng thông báo nhắc), BUG-P6-08-1 (test đọc mã nguồn), BUG-P6-08-2 (`data-layer.spec` flaky ở CI), nhãn ô chọn dài ở 390 px (S8-UI), `dev.mjs` + `BLOB_PUBLIC_ENDPOINT` với `EP_PORT_OFFSET`.

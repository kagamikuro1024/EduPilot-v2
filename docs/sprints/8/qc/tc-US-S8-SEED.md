# QC test case — US-S8-SEED (seed sprint 8: quy chế + công thức lớp 1 qua luồng thật, điểm qua XLSX thật, lớp 2 cố ý chưa quy chế, `gate-p6.sh`, `gradebook.spec.ts`)
Nguồn: `docs/specs/FEAT-gradebook/US.md` US-S8-SEED (AC1–AC8) + `SRS.md` 4.9, 9; `docs/DEMO_SCRIPT.md` bước 6; `seed/expected_final_grades.csv`, `seed/expected_risk.csv`; `proposals.md` #2, #5. Viết trước khi có code, không đọc code của dev (v1, 2026-10-31).

**Quy ước.** "DB trống" = `pnpm dev:down -v && pnpm dev` (`SEED_ON_EMPTY_DB=true`); chỉ **một** stack chạy. `PSQL`, `$Q`, `$Q2`… từ `p6-env.sh`. `GO` = `p6-go-tests.sh`.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P6-SEED-01 | AC1 | DB trống + seed xong | `$PSQL "select d.title, d.type, d.category, d.status from documents d join courses c on c.id=d.course_id where c.class_code='761987' and (d.title ilike '%quy chế%' or d.title ilike '%quyche%' or d.title ilike '%quy-che%') order by 1"` | hai dòng: `quy-che-lop1.pdf` `COURSE_POLICY` `READY`; `Quyche.pdf` `OTHER` `Quy chế trường`; **không** `COURSE_POLICY` nào khác |
| TC-P6-SEED-02 | AC1 | như trên | `$PSQL "select class_code, s.status, scheme_no, confirmed_by is not null from grade_schemes s join courses c on c.id=s.course_id order by 1"` | **đúng một** dòng `761987 | CONFIRMED | 1 | t`; lớp 761988 không có dòng; 3 thành phần (20/20/60) + luật BONUS, PENALTY, ROUNDING, LETTER_SCALE, 2 × FAIL_CONDITION, MISSING_SCORE (`select kind, count(*) from grade_rules …`) |
| TC-P6-SEED-03 | AC1 | như trên | Xác nhận đi **qua luồng thật**: `$PSQL "select entity, action, actor_id from audit_log where entity='grade_schemes' order by created_at"`; `jobs` có `grade_scheme.extract`; `grep -nE "UPDATE grade_|INSERT INTO grade_|confirmed_by" scripts/seed.mjs scripts/*.mjs` | `audit_log` có `confirm` do **teacher@** (không phải `system`/NULL); job extract tồn tại; script seed **không** có SQL ghi thẳng bảng `grade_*` / không gọi hàm nội bộ (chỉ HTTP) |
| TC-P6-SEED-04 | AC1 | như trên | `j $Q2/grade-scheme/status -H "$TCH" \| jq -r .state`; `$PSQL "select count(*) from documents d join courses c on c.id=d.course_id where c.class_code='761988' and d.type='COURSE_POLICY'"` | `NO_POLICY`; `0` |
| TC-P6-SEED-05 | AC2 | seed xong | `j "$Q/gradebook/preview?limit=100" -H "$TCH" > /tmp/pv.json`; `node scripts/check-grade-seed.mjs preview` | `30/30 OK`; QC đối chiếu độc lập bằng `p6-preview-vs-csv.mjs` (so `hp`, `letter`, `verdict` 30 dòng theo `student_code`) → 30/30 |
| TC-P6-SEED-06 | AC2 | như trên | Điểm vào bằng **nhập XLSX thật**: `$PSQL "select source, count(*) from grade_entries e join grade_items i on i.id=e.item_id where i.key in ('midterm','final') group by 1"`; `$PSQL "select action, count(*) from audit_log where entity='grade_entries' group by 1"` | `IMPORT | 60`; audit có dòng của lô nhập (không `set` từng ô) |
| TC-P6-SEED-07 | AC2 | như trên | Item EXAM: `$PSQL "select i.key, i.source, e.name, e.status from grade_items i join exams e on e.id=i.exam_id where i.course_id='$C1'"` | item tuần 9 nguồn `EXAM` nối `Kiểm tra tuần 9 — Mật mã`, `PUBLISHED` |
| TC-P6-SEED-08 | AC2 | như trên | Tệp lỗi cố ý **không** ghi DB: `$PSQL "select count(*) from grade_entries where score is null or score > 10"` và đếm dòng sau khi chạy kèm tệp lỗi | `0` dòng lỗi trong DB; số `grade_entries` bằng 60 (+ không dòng từ tệp lỗi) |
| TC-P6-SEED-09 | AC2 | như trên | `$PSQL "select i.key, i.published_at is not null from grade_items i where i.course_id='$C1' order by position"` | giữa kỳ và cuối kỳ **đã công bố** (`t`); `j $Q/me/grades -H "$SVA"` (A, B, C) cho thấy điểm thành phần giữa kỳ / cuối kỳ |
| TC-P6-SEED-10 | AC3 | seed xong | `node scripts/check-grade-seed.mjs abc`; đọc `…/explain` của B, C, A | `A B C OK`; A `9,5 A+`, B `7,0 B`, C `3,0 F` — **QC tự tính tay** (xem `tc-US-P6-02` bảng phép tính + `p6-hand-calc.py A B C` rc=0) và so với giải trình trên **màn hình** `/gradebook` (ảnh `shots/abc-gradebook.png`) |
| TC-P6-SEED-11 | AC3 | như trên | Giải trình B: `jq '.lines[] \| select(.label\|test("phát biểu\|vắng\|Kiểm tra"))'` | có "phát biểu 2 × 0,25 = 0,50", "vắng 2 (miễn 2)", điểm kiểm tra tuần "8,00" từ bài thi PE |
| TC-P6-SEED-12 | AC3 | như trên | `$PSQL "select status, snapshot_version from gradebooks"`; mở `/gradebook` lớp 2 | sổ lớp 1 `OPEN`, `snapshot_version 0` (chưa chốt — để demo); lớp 2: **không** có cột điểm |
| TC-P6-SEED-13 | AC4 | repo | `ls -l seed/demo/quy-che-lop2.pdf`; `pdftotext seed/demo/quy-che-lop2.pdf - \| grep -ciE 'làm tròn\|quy tròn\|bước điểm'`; so chữ với `fixtures/quy-che-lop2.md`: QC `python3` lấy các đoạn `x.y.` của md, tách từ `pdftotext` và kiểm từng câu có trong PDF | tệp có; `0` lần xuất hiện của "làm tròn"; mọi câu của md có trong PDF **đúng chữ** (dev không đổi chữ); 2 trang |
| TC-P6-SEED-14 | AC4 | seed; `DEMO_MODE`/provider giả; TEACHER; lớp 2 | UI `/gradebook/scheme?course=<C2>` → `Tải quy chế` → chọn `seed/demo/quy-che-lop2.pdf` | tiến độ nền; bản nháp đúng `extraction-lop2.json`: **3 thành phần 25/25/50**, BONUS trần 1,5, PENALTY miễn 1, LETTER_SCALE 5 mức, FAIL CK < 3,0; **một** mục chưa rõ `ROUNDING_MISSING`; `Xác nhận công thức` khoá kèm tooltip |
| TC-P6-SEED-15 | AC4 | như trên | Điền "Làm tròn đến 0,1" (bước 0,1, nửa lên) → `Xác nhận công thức` → xác nhận hộp thoại | nút mở sau điền; xác nhận → `CONFIRMED`; banner lớp 2 biến mất; `Chốt điểm` **mở** (không khoá lý do); `status.state=CONFIRMED` |
| TC-P6-SEED-16 | AC4 | repo | `jq 'keys' seed/demo/fake_llm.json` (hoặc đường dẫn thật); `GO ./internal/grading "TestExtractLop2OneUnclear|TestDemoModeRecordingsCoverPolicies"` | có bản ghi cho **cả hai** quy chế; PASS |
| TC-P6-SEED-17 | AC4 | repo | `playwright test gradebook.spec.ts -g 'lop2 demo step 6'` | xanh |
| TC-P6-SEED-18 | AC5 | DB đã seed | Chạy seed lần hai: `node scripts/seed.mjs`; đếm `grade_schemes, grade_components, grade_items, grade_entries, documents` trước/sau; `node scripts/check-grade-seed.mjs counts` | số **không đổi**; `documents` không thêm bản `quy-che-lop1.pdf` nào; không lỗi 409 làm seed dừng |
| TC-P6-SEED-19 | AC5 | seed gãy giữa chừng: `kill` seed khi đang nhập XLSX (hoặc sau `extract` chưa `confirm`) | chạy lại | tiếp tục; không tạo đôi bản nháp / điểm; kết quả cuối giống chạy sạch (30/30) |
| TC-P6-SEED-20 | AC5 | — | `NODE_ENV=production node scripts/seed.mjs; echo rc=$?` | `rc ≠ 0`; không ghi |
| TC-P6-SEED-21 | AC5 | DB trống | `pnpm dev` (không lệnh seed riêng) | seed tự chạy đủ dữ liệu P6 (TC-01…12 đạt) |
| TC-P6-SEED-22 | AC5 | seed xong | Sprint 5–7 không vỡ: `node scripts/check-crm-seed.mjs`, `check-chat-seed.mjs`, `check-exam-seed.mjs`, `check-docs-seed.mjs`; `bash scripts/gate-p5.sh; echo rc=$?` | các check xanh; điểm danh 270, ticket 5…; `gate-p5.sh` rc=0 (`seed/expected_crm.csv` không đổi); `WATCH` = {sv12, sv19, sv26} so `expected_risk.csv` (đổi ghim **được PM chấp nhận #5**) |
| TC-P6-SEED-23 | AC5 | repo | `git diff <main>.. --stat -- scripts/check-crm-seed.mjs` và đọc diff | thay đổi duy nhất: ghim `WATCH === 2` → đối chiếu `expected_risk.csv`; không nới assertion khác (`G5-08`) |
| TC-P6-SEED-24 | AC6 | repo | `bash scripts/gate-p6.sh; echo rc=$?` | in một dòng `PASS`/`FAIL` mỗi lệnh; có `go test -race ./internal/grade/... -v`, `! grep float64`, `go test ./internal/grading -run TestExtractGradeScheme`, `curl … finalize` lớp 2 `409`, `gradebook.spec.ts`; có bước nhắc "chủ dự án tự tính tay A, B, C"; `rc=0` khi xanh |
| TC-P6-SEED-25 | AC6 (cổng không "xanh giả") | repo | **Đột biến**: thêm `var _ = float64(1)` vào `internal/grade/*.go`; chạy `gate-p6.sh` | gate **đỏ** ở dòng `float64` (rc ≠ 0); hoàn tác. Đột biến 2: đổi 1 giá trị của một dòng trong bản sao `expected` không được (cấm sửa CSV) → thay bằng đổi mã làm tròn (nửa-chẵn) trong `/tmp/qc-mut` → gate đỏ ở bước `go test … 30/30` |
| TC-P6-SEED-26 | AC7 | repo | `pnpm -C frontend exec playwright test gradebook.spec.ts --workers=2`; ghi số test, flaky | xanh toàn bộ; đi trọn F10: lớp 2 banner + `Chốt điểm` khoá (tooltip) → tải `quy-che-lop2.pdf` → bản nháp mục chưa rõ chặn → điền → hộp thoại có con số → banner biến mất; lớp 1: sửa ô (Enter/Tab) → xem trước đổi → 409 hai phiên → giải trình B → nhập XLSX có dòng lỗi → xuất XLSX (dòng ghi chú) → `Chốt điểm` → ô khoá → `Mở khoá` có lý do → sửa → chốt lại v2 → sinh viên bị đổi nhận chuông → `/me` của B có what-if; TA chỉ đọc; sinh viên không thấy điểm nháp (≥ 3 flaky → FAIL) |
| TC-P6-SEED-27 | AC7 | repo | `pnpm -C frontend exec playwright test gradebook.spec.ts --list \| grep -c` ; đối chiếu với tên `-g` đã nêu ở mọi US (QC lập danh sách: `scheme review layout`, `unclear blocks confirm`, `scheme edit optimistic`, `impact preview`, `confirm irreversible`, `new policy diff`, `scheme offline retry`, `scheme rbac`, `scheme states`, `scheme mobile`, `banner unconfirmed`, `finalize locked reason`, `attendance notice`, `gradebook layout`, `official note`, `cell edit keys`, `cell conflict choose`, `explain student B`, `virtualized 1000`, `type then scroll keeps text`, `gradebook states`, `offline cell`, `gradebook rbac`, `publish column`, `penalty not in scheme notice`, `import xlsx flow`, `import errors listed`, `unlock reason`, `finalize flow`, `finalize incomplete list`, `lop2 demo step 6`) | mọi tên có ≥ 1 test (không `-g` nào khớp 0 test: Playwright báo "No tests found" → FAIL) |
| TC-P6-SEED-28 | AC8 | seed | A, B, C: `j $Q/me/grades` của mình đúng; thử chéo `SVB` gọi `$Q/gradebook/students/$SVC_ID/explain` | đúng; chéo `403` |
| TC-P6-SEED-29 | AC8 | seed; sinh viên D (chưa vào lớp) | đăng nhập D, `GET /courses` / dashboard | không thấy lớp nào; `403/404` ở `$Q/me/grades` |
| TC-P6-SEED-30 | AC8 | seed | TA: `code -H "$TA_" $Q/gradebook` → `200`; `code -H "$TA_" $Q2/gradebook` (TA không thuộc lớp 2) | `200`; `404` (không thấy lớp 2) — nếu seed gán TA vào cả hai lớp → ghi nhận và đối chiếu `ARCHITECTURE.md` §9 |
| TC-P6-SEED-31 | AC8 | seed | ADMIN: chạy `p6-rbac.sh` nhóm `[ADM]` | `403` mọi route lớp |
| TC-P6-SEED-32 | AC8 | seed + CANARY (xem tc-US-P6-09 quy ước) | `p6-student-sweep.sh` | canary không xuất hiện ở phản hồi sinh viên nào |
| TC-P6-SEED-33 | AC8 | repo | `GO ./internal/integration "TestSeedGradebookExpected|TestSeedGradebookIsolation|TestSeedGradebookAdminForbidden" -tags integration` | PASS |

## Nhánh lỗi (SRS mục 3)
| TC-id | Tình huống | Bước | Kết quả mong đợi |
| --- | --- | --- | --- |
| TC-P6-SEED-34 | Provider giả thiếu bản ghi cho quy chế lạ | tải một PDF tuỳ ý ở lớp thử với `DEMO_MODE=true` | lỗi rõ ràng (`EXTRACTION_INVALID`/nêu "chưa có bản ghi"), không bản nháp rác, không treo |
| TC-P6-SEED-35 | Seed khi stack chưa sẵn sàng | chạy `scripts/seed.mjs` khi gateway tắt | thoát lỗi rõ (không treo); chạy lại sau khi bật → thành công |
| TC-P6-SEED-36 | `SEED_RNG` khác | `SEED_RNG=1 node scripts/seed.mjs` trên DB trống | MSSV đổi; vì nhập XLSX **khớp theo `student_code_snapshot`** nên preview vẫn 30/30 (so bằng `account` — `fixtures/README.md`) |

## Phân quyền
Phủ ở TC-28…32.

## Script chạy được
`scripts/p6-env.sh`, `p6-hand-calc.py`, `p6-rbac.sh`, `p6-go-tests.sh`; QC pha 2: `p6-preview-vs-csv.mjs`, `p6-student-sweep.sh`.

## Câu hỏi cho BA / PM (không tự đoán)
- Q-SEED-a (TC-P6-SEED-30): TA có thuộc lớp 2 ở seed không? US-S8-SEED AC8 nói "TA thấy sổ lớp 1, không thấy lớp 2". QC dùng đúng câu đó.

## Lịch sử sửa TC
(chưa có)

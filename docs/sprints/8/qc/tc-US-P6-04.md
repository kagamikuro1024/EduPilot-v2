# QC test case — US-P6-04 (`/gradebook/scheme`: duyệt công thức như văn bản, mục chưa rõ chặn, xem thử 3 sinh viên, xác nhận có hậu quả bằng số)
Nguồn: `docs/specs/FEAT-gradebook/US.md` US-P6-04 (AC1–AC10) + `SRS.md` 3, 4.4, 6.1, 7; `docs/design/DESIGN.md` §14.11, §21–§22; `docs/UX.md` mục 6; luật chữ UI (`~/.claude/CLAUDE.md`). Viết trước khi có code, không đọc code của dev (v1, 2026-10-31).

**Quy ước.** UI kiểm bằng `playwright-cli` (global) ở 1440 × 900 và 375 × 812, đăng nhập `teacher@edupilot.local` / `ta@edupilot.local` / `sv.gioi@edupilot.local`, mật khẩu seed. Ảnh vào `docs/sprints/8/qc/shots/`. `GO` = `bash docs/sprints/8/qc/scripts/p6-go-tests.sh`. Lớp 2 phải ở trạng thái seed gốc (chưa quy chế) khi bắt đầu nhóm TC "lớp 2"; làm nhóm đó **một lần** rồi dựng lại DB (`pnpm dev:down -v && pnpm dev`) trước khi lặp.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P6-04-01 | AC1 | TEACHER, lớp 2 đã tải `quy-che-lop2.pdf` và trích xong (DRAFT, một mục chưa rõ) | `playwright-cli`: mở `/gradebook/scheme?course=<C2>` ở 1440 px; chụp; đếm `[data-panel]`/vùng `Panel` (đối chiếu primitive ở `frontend/src/shared/Panel`): lồng nhau? | **một** `Panel`; cột chính: các khối theo thứ tự Thành phần · Điểm cộng · Điểm trừ · Làm tròn · Thang chữ · Liệt, mỗi khối có tiêu đề + giá trị + đoạn trích ngắn; cột phải hiện **trang PDF** của mục đang chọn; **không** `Panel` lồng `Panel`; không tường ô nhập; một nút chính `Xác nhận công thức` |
| TC-P6-04-02 | AC1 | như trên | Click lần lượt khối "Thành phần: Giữa kỳ", "Điểm cộng", "Thang chữ" | cột phải mở **đúng trang** có đoạn trích (Giữa kỳ → tr.1; Điểm cộng → tr.2; Thang chữ → tr.2 theo `extraction-lop2.json`); đoạn trích được bôi/nhấn mạnh; số trang hiển thị khớp `python3 …/p6-citation-check.py` |
| TC-P6-04-03 | AC1 | như trên | Thu hẹp cửa sổ 1.023 px rồi 1.024 px | ≥ 1.024: cột phải cố định; < 1.024: trích dẫn trong `Drawer` (mở khi chọn mục, `Esc` đóng, trả focus đúng nút) |
| TC-P6-04-04 | AC1, luật chữ | như trên | Đọc toàn văn trang (`playwright-cli` → `page.innerText('main')`); đếm icon Info / CircleHelp (`[data-icon="info"], svg[aria-label*="Thông tin"]`) và dòng mô tả dưới tiêu đề | **không** câu mô tả dưới tiêu đề trang / khối; không icon Info/CircleHelp; tooltip chỉ ở nhãn / nút cần (lý do nút khoá, cú pháp); nút là động từ |
| TC-P6-04-05 | AC1 | repo | `bash scripts/ui-antipatterns.sh; echo rc=$?`; `pnpm -C frontend exec playwright test gradebook.spec.ts -g 'scheme review layout'` | `rc=0`; xanh (ghi số test, flaky) |
| TC-P6-04-06 | AC2 | như TC-01 (lớp 2, `ROUNDING_MISSING`) | Xem khối "Làm tròn" và nút chính | khối "Làm tròn" **trống tại chỗ** với ô điền + (nếu quy tắc cho phép) lựa chọn `Không áp dụng`; `Xác nhận công thức` **disabled** (`aria-disabled`); rê chuột / focus bàn phím → tooltip **trên chính nút**: "Còn 1 mục chưa rõ: làm tròn"; không modal chặn |
| TC-P6-04-07 | AC2 | như trên | Điền "Làm tròn đến 0,1" (bước 0,1, nửa lên) rồi rời ô | mục "chưa rõ" biến mất; nút mở khoá (`enabled`) khi hết mục; tooltip lý do biến mất; **demo D5 (DEMO_SCRIPT bước 6)** đi được trọn |
| TC-P6-04-08 | AC2 | API, bản nháp còn mục chưa rõ | `curl -X POST $Q2/grade-scheme/confirm -H "$TCH" -H "$(idem)" -d '{"version":<v>}' -w '\n%{http_code}'` | `422 SCHEME_HAS_UNCLEAR`, `details` liệt kê mã (`ROUNDING_MISSING`); không đổi `status` (kiểm `$PSQL "select status from grade_schemes where course_id='$C2'"` = `DRAFT`) |
| TC-P6-04-09 | AC2 | repo | `GO ./internal/grade "TestConfirmBlockedByUnclear" -tags integration`; `pnpm -C frontend exec playwright test gradebook.spec.ts -g 'unclear blocks confirm'` | PASS; xanh |
| TC-P6-04-10 | AC3 | bản nháp lớp 2 | Điền "Làm tròn" qua UI; ngay sau đó `PUT` xem `network` | `PUT …/grade-scheme {…, version}` chạy ngay khi rời ô; phản hồi `version + 1`, kèm `unclear_items` mới do máy chủ tính lại; mục vừa điền có `filled_by_user = true` và **không** trích dẫn (UI không có đoạn trích cho mục này; giải trình sau này ghi "do giảng viên đặt") |
| TC-P6-04-11 | AC3 | như trên | Gõ trọng số `abc`, `-5`, `101`, `12,5`; xem ô | số không hợp lệ → lỗi **tại ô**, **giữ chữ đã gõ** (không xoá); `12,5` hợp lệ nếu quy tắc cho (QC ghi) ; trọng số tổng ≠ 100 → xuất hiện mục chưa rõ `WEIGHT_SUM` kèm tổng thật |
| TC-P6-04-12 | AC3 | hai tab TEACHER cùng bản nháp | Tab 1 sửa trọng số → lưu; Tab 2 (chưa tải lại, `version` cũ) sửa ô khác | Tab 2 nhận `409 VERSION_CONFLICT` kèm bản hiện tại; UI hoàn lại + `InlineNotice` (không toast "Thành công"); không mất chữ Tab 2 đã gõ (hiện lại để gõ lại) |
| TC-P6-04-13 | AC3 | lớp 1 `CONFIRMED` | `curl -X PUT $Q/grade-scheme -H "$TCH" -d '{"version":1,"components":[…],"rules":[…]}'` khi không có DRAFT; và khi chỉ có `CONFIRMED` | `409 NO_DRAFT` / `409 SCHEME_NOT_DRAFT`; công thức đã xác nhận không đổi |
| TC-P6-04-14 | AC3 | repo | `GO ./internal/grade "TestSchemePutRecheck|TestSchemePutVersionConflict|TestSchemePutOnlyDraft|TestFilledByUserNoCitation" -tags integration`; `playwright test gradebook.spec.ts -g 'scheme edit optimistic'` | PASS; xanh |
| TC-P6-04-15 | AC4 | lớp 1 `CONFIRMED`, 30 sinh viên đủ điểm; có DRAFT mới (hoặc dùng `POST …/grade-scheme/impact` trực tiếp) | Đổi bước làm tròn 0,5 → 0,1 trong bản nháp; xem khối "Xem thử" | trong ≤ 300 ms (đo `performance.now()` từ lúc rời ô đến lúc khối đổi; p95 của 20 lần) hiện **3** sinh viên (cao nhất, giữa, thấp nhất theo HP trong số sinh viên **đủ điểm**) với `trước → sau`: sv06 `10,0 → 9,8`; C `3,0 → 3,2`; sinh viên giữa khớp `python3 …/p6-hand-calc.py --table step=0.1` (QC đối chiếu, xem Q-02-b) |
| TC-P6-04-16 | AC4 | như trên | Gọi `impact` hai lần với cùng thân; đếm `$PSQL "select count(*) from audit_log"` và `grade_entries` trước/sau | **không ghi** gì (hai số không đổi); phản hồi `{students_total:30, students_computable:30, changed_vs_confirmed:N, entries_carried, entries_dropped, samples[3]}`; N khớp số sinh viên đổi HP/chữ so với công thức đã xác nhận (QC tính: step 0,1 ⇒ HP đổi cho những sv có HP thô không phải bội 0,5: `python3 …/p6-hand-calc.py --table` hai lần rồi `diff`) |
| TC-P6-04-17 | AC4 | lớp 2 (chưa đủ điểm; không ai có điểm) | Xem khối "Xem thử" của bản nháp lớp 2 | dòng "Chưa đủ điểm để xem thử"; không lỗi; không số bịa |
| TC-P6-04-18 | AC4 | repo | `GO ./internal/grade "TestImpactThreeSamples|TestImpactNoWrite|TestImpactNoCompleteStudents" -tags integration`; `playwright test gradebook.spec.ts -g 'impact preview'` | PASS; xanh |
| TC-P6-04-19 | AC5 | lớp 2, DRAFT đã điền, 0 điểm đã nhập | Bấm `Xác nhận công thức` | `<ConfirmIrreversible>` nêu **số**: "Áp cho 0 sinh viên có đủ điểm / lớp N sinh viên · điểm học phần của 0 sinh viên sẽ đổi so với công thức hiện tại · 0 điểm đã nhập bị bỏ" (lớp 2 chưa có điểm, số = 0; khớp `impact.entries_dropped`); có đủ hậu quả, nút xác nhận là động từ, `Esc` huỷ, không đóng khi click ngoài |
| TC-P6-04-20 | AC5 | lớp 1 `CONFIRMED` v1 có điểm; DRAFT mới đổi bước làm tròn 0,1 | Bấm `Xác nhận công thức` | hộp thoại nêu "Áp cho 30 sinh viên · điểm học phần của **X** sinh viên sẽ đổi so với công thức hiện tại · **Y** điểm đã nhập bị bỏ"; X = số sinh viên QC tự tính ở TC-16; Y = 0 (cùng key + nguồn + max_score) |
| TC-P6-04-21 | AC5 | như trên | Xác nhận; chặn mạng 1 lần (gửi lại cùng khoá) | `POST …/grade-scheme/confirm {version}` có header `Idempotency-Key`; `200`; DB: DRAFT → `CONFIRMED` (`confirmed_by` = teacher, `confirmed_at` mới); bản `CONFIRMED` cũ → `SUPERSEDED`; `audit_log` (`entity='grade_schemes', action='confirm'`, số liệu hậu quả); outbox `grade.scheme_confirmed`; gửi lại cùng `Idempotency-Key` → **cùng phản hồi**, không `scheme_no` thứ hai |
| TC-P6-04-22 | AC5 | như trên | Sau xác nhận lớp 2: mở "Hôm nay" (`/`) của TEACHER trong ≤ 5 s | mục `GRADE_SCHEME_UNCONFIRMED` của lớp 2 biến mất; banner `/gradebook` lớp 2 biến mất |
| TC-P6-04-23 | AC5 (phân quyền) | — | `for T in "$TA_" "$SVA" "$ADM"; do code -X POST -H "$T" -H "$(idem)" -H 'Content-Type: application/json' -d '{"version":1}' $Q2/grade-scheme/confirm; done`; UI TA mở `/gradebook/scheme` | `403 403 403`; TA **không** thấy nút `Xác nhận công thức`, `Tải quy chế`, ô điền (xem TC-34) |
| TC-P6-04-24 | AC5 | repo | `GO ./internal/grade "TestConfirmFlow|TestConfirmIdempotent|TestConfirmAudit|TestConfirmRBAC" -tags integration`; `playwright test gradebook.spec.ts -g 'confirm irreversible'` | PASS; xanh |
| TC-P6-04-25 | AC6 | lớp 1 `CONFIRMED` v1, 60 điểm giữa/cuối đã nhập; DRAFT mới cùng `key` (`midterm`, `final`), cùng `max_score 10`, nguồn MANUAL/IMPORT | Xác nhận DRAFT mới; sau đó `$PSQL "select i.key, count(e.*) from grade_items i left join grade_entries e on e.item_id=i.id where i.course_id='$C1' group by 1, i.scheme_id order by 2"` và đọc `/gradebook` | điểm giữa/cuối **chuyển** sang item mới (60 dòng; số hiện ở hộp thoại); `/gradebook` vẫn ra đúng 30/30; item cũ giữ trong DB (ẩn khỏi bảng) |
| TC-P6-04-26 | AC6 | như trên; DRAFT mới đổi `max_score` giữa kỳ 10 → 100 và đổi `weekly` sang nguồn EXAM | Xem hộp thoại, xác nhận | `entries_dropped:[{item_key:"midterm", count:30, reason:…}, …]` hiện trong hộp thoại bằng số; **không** mang điểm giữa kỳ sang item `max_score 100` (30 điểm mất khỏi bảng nhưng còn trong DB) |
| TC-P6-04-27 | AC6 | `GradebookStatus = LOCKED` (sau chốt) | `POST …/grade-scheme/confirm` | `409 GRADEBOOK_LOCKED` (phải `Mở khoá` trước); DRAFT không đổi |
| TC-P6-04-28 | AC6 | repo | `GO ./internal/grade "TestConfirmCarryOverEntries|TestConfirmNoCarryOnMaxScoreChange|TestConfirmNoCarryOnSourceChange|TestConfirmSupersedeBeforeConfirm|TestConfirmCreatesGradebookRow|TestConfirmBlockedWhenLocked" -tags integration` | PASS |
| TC-P6-04-29 | AC7 | lớp 1 `CONFIRMED` | Tải `COURSE_POLICY` khác (copy `quy-che-lop1` đổi trọng số 20/20/60 → 15/25/60, giữ chữ khác) qua `Tải quy chế` | chuỗi tải → ingest → trích chạy nền; tiến độ `useSSE` (`job.progress`) hiện; sau xong có bản nháp **mới** cạnh bản `CONFIRMED`; `/gradebook` vẫn tính theo bản `CONFIRMED` (HP của A, B, C không đổi) cho tới khi xác nhận |
| TC-P6-04-30 | AC7 | như trên | Bật chế độ so sánh: `GET $Q/grade-scheme?diff=confirmed` và UI | diff do **code** tính: thành phần đổi trọng số (`midterm 20 → 15`, `weekly 20 → 25`), luật đổi tham số (cũ → mới); thành phần thêm / bớt (nếu có) |
| TC-P6-04-31 | AC7 | như trên | Tải lên hỏng: ngắt mạng giữa lúc tải | giữ tên tệp + nút `Tải lại`; không bản nháp rác |
| TC-P6-04-32 | AC7 | repo | `GO ./internal/grade "TestNewPolicyMakesNewDraft|TestDiffAgainstConfirmed|TestConfirmedUnchangedUntilConfirm" -tags integration`; `playwright test gradebook.spec.ts -g 'new policy diff'` | PASS; xanh |
| TC-P6-04-33 | AC8 | — | Hai giảng viên (hai phiên) cùng sửa; `confirm` với `version` cũ | `409 VERSION_CONFLICT` + giá trị hiện tại; **không** xác nhận bản đã đổi (`status` vẫn `DRAFT`); sau đó `confirm` với `version` mới → `200` |
| TC-P6-04-34 | AC8 | việc trích `FAILED` (xem tc-US-P6-03 TC-35) | UI | khối lỗi tại chỗ (nguyên nhân + `Thử lại`); bản nháp cũ nguyên vẹn |
| TC-P6-04-35 | AC8 | đang sửa ô | `playwright-cli`: `context.setOffline(true)`; gõ, rời ô; `setOffline(false)` | thay đổi xếp hàng, nhãn tại ô ("Chưa lưu"/tương đương, 1–2 từ), tự gửi lại khi có mạng; **không mất chữ**; sau đó DB khớp (`useAutosaveDraft`) |
| TC-P6-04-36 | AC8 | repo | `GO ./internal/grade "TestSchemeEditRace|TestConfirmStaleVersion" -tags integration`; `playwright test gradebook.spec.ts -g 'scheme offline retry'` | PASS; xanh |
| TC-P6-04-37 | AC10 | repo | `grep -rn "mock/gradebook" frontend/src; ls frontend/src/mock/gradebook.ts 2>&1` | `grep` rỗng; tệp không tồn tại (D51) |
| TC-P6-04-38 | AC10 | TEACHER, DB trống lớp 2 | Mở `/gradebook/scheme` khi chưa có quy chế | trạng thái rỗng "Lớp chưa có quy chế." + `Tải quy chế`; skeleton khi tải (chặn mạng chậm `route.fulfill` trễ 3 s → có skeleton); lỗi 500 (chặn `…/grade-scheme` trả 500) → nguyên nhân + `Thử lại`, bấm `Thử lại` tải lại được |
| TC-P6-04-39 | AC10 | 375 × 812 | Mở trang có bản nháp; chụp; `document.documentElement.scrollWidth <= innerWidth` | một cột; trích dẫn mở bằng `Drawer`; không cuộn ngang; nút ≥ 44 px chạm (kiểm `getBoundingClientRect` của nút chính) |
| TC-P6-04-40 | AC10 | repo | `playwright test gradebook.spec.ts -g 'scheme states\|scheme mobile'` | xanh |
| TC-P6-04-41 | AC10 | — | Xác nhận xong, tìm toast | **không** toast "Thành công"; trạng thái đổi tại chỗ (chữ "Đã xác nhận · {thời điểm}") |

## Nhánh lỗi (SRS mục 3)
Đã phủ: `SCHEME_HAS_UNCLEAR` (TC-08), `VERSION_CONFLICT` (TC-12, 33), `NO_DRAFT` / `SCHEME_NOT_DRAFT` (TC-13), `GRADEBOOK_LOCKED` (TC-27), tải lỗi (TC-31), trích lỗi (TC-34), mất mạng (TC-35).
| TC-id | Tình huống | Bước | Kết quả mong đợi |
| --- | --- | --- | --- |
| TC-P6-04-42 | `confirm` thiếu `Idempotency-Key` | `curl -X POST $Q2/grade-scheme/confirm -H "$TCH" -d '{"version":1}'` (không header) | `400`/`422` có mã lỗi rõ (luật 14), không xác nhận |
| TC-P6-04-43 | `confirm` thân thiếu `version` | `-d '{}'` | `422 VALIDATION_FAILED` |
| TC-P6-04-44 | Tooltip lý do bị khoá | rê chuột / `Tab` tới nút khoá | tooltip xuất hiện cả khi focus bàn phím (không chỉ hover) |

## Phân quyền (AC9)
| TC-id | Tình huống | Bước | Kết quả mong đợi |
| --- | --- | --- | --- |
| TC-P6-04-45 | TA mở `/gradebook/scheme` | `playwright-cli` đăng nhập `ta@`; mở trang lớp 1 và lớp 2 | thấy công thức / bản nháp + trích dẫn **chỉ đọc**; **không** thấy `Xác nhận công thức`, `Tải quy chế`, ô điền (`input` count trong vùng công thức = 0) |
| TC-P6-04-46 | STUDENT mở route | đăng nhập `sv.gioi@`; mở `/gradebook/scheme` | trang 403 chuẩn của shell; **không** có chữ công thức nháp trong DOM (kiểm `page.content()` không chứa "Thi cuối kỳ" / "60"); `code -H "$SVA" $Q/grade-scheme` → `403` |
| TC-P6-04-47 | Lớp khác | giảng viên lớp khác mở `$Q/grade-scheme` | `404` |
| TC-P6-04-48 | repo | `playwright test gradebook.spec.ts -g 'scheme rbac'` | xanh |
| TC-P6-04-49 | ADMIN | `code -H "$ADM" $Q/grade-scheme` | `403` |

## Script chạy được
`scripts/p6-hand-calc.py --table …` (đối chiếu `impact`), `p6-citation-check.py`, `p6-go-tests.sh`, `p6-rbac.sh`; UI: `playwright-cli` thủ công + `frontend/e2e/gradebook.spec.ts` của dev (QC chỉ chạy, không sửa).

## Câu hỏi cho BA / PM (không tự đoán)
- Q-04-a (TC-P6-04-19): chuỗi con số trong hộp thoại ở lớp chưa có điểm — AC5 mẫu "Áp cho 30 sinh viên…". QC kỳ vọng số khớp `impact` thực (có thể "Áp cho 0 sinh viên đủ điểm" ở lớp 2); chấp nhận mọi cách diễn đạt miễn **đủ ba con số** (sinh viên áp, số đổi, số điểm bỏ) và khớp `impact`.
- Q-04-b (TC-P6-04-11): `12,5` làm trọng số có hợp lệ không (cột `numeric(5,2)` cho phép)? QC ghi nhận hành vi thực; không FAIL nếu nhất quán với `weight BETWEEN 0 AND 100`.

## Lịch sử sửa TC
(chưa có)

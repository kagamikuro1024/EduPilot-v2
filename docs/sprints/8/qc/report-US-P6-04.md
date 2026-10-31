# QC report — US-P6-04 (`/gradebook/scheme`: duyệt công thức như văn bản, mục chưa rõ chặn, xem thử 3 sinh viên, xác nhận có hậu quả bằng số) · Kết luận vòng 1: **FAIL** (AC1: "không tường ô nhập" bị vi phạm — 22 ô nhập luôn hiện; AC4: chưa đo ≤ 300 ms; CI đỏ ở `panels.spec.ts` do chính story — BUG-P6-04-3; còn lại PASS)
Ngày 2026-10-31. Commit kiểm: `sprint/8-p6` @ `a50ec5b9` (đã gồm `6d552d67`, thêm liên kết `Mở trang N` theo quyết định PM 18:41). Bộ TC: `tc-US-P6-04.md` (chữ TC chỉ đổi 1 dòng theo #11). Handoff: `dev-US-P6-04.md`. CI: run `41198097712` (`a50ec5b9`) — xem "CI" cuối mục 1.

## 1. Cổng nghiệm thu đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `p6-go-tests.sh ./internal/grade "<25 test của US-P6-04>" -race` | PASS 25/25 (không thiếu, không SKIP, không DATA RACE) |
| `go test -count=1 -tags testroutes ./internal/document ./internal/contract ./internal/grade ./internal/grading` (sau `a50ec5b9`) | PASS cả bốn (contract: số thao tác 232 theo #12 mở rộng) |
| `golangci-lint` (document, grade, grading, httpapi); `sqlc diff` | 0 issue; rc=0 |
| `pnpm exec playwright test gradebook.spec.ts --workers=2 --retries=0` (bản dựng `build:gate`) | PASS 15 desktop (mobile bỏ qua theo thiết kế, 15 skipped); `ui-antipatterns.sh` rc=0; `pnpm lint` sạch |
| `grep -rn "mock/gradebook" frontend/src`; `ls frontend/src/mock/gradebook.ts` | rỗng; tệp không tồn tại (D51, AC10) |
| `grep ConfirmScheme` | chỉ định nghĩa ở `confirm.go` và một lời gọi ở `gradehttp/handler.go:184`; `TestConfirmOnlyFromHandler` PASS |
| `git diff … -- frontend/package.json backend-go/go.mod` | rỗng (không thư viện mới); không `fetch` trần ở `features/gradebook` |
| Đột biến (bản sao `/tmp/qc-mut`, đã xoá) | Q1 bỏ chặn mục chưa rõ khi xác nhận: **đỏ** `TestConfirmBlockedByUnclear`; Q2 bỏ so `version` ở `confirm`: **đỏ** `TestConfirmStaleVersion`; Q3 bỏ kiểm `LOCKED`: **đỏ** `TestConfirmBlockedWhenLocked`; Q5 mang điểm khi đổi `max_score`: **đỏ** `TestConfirmNoCarryOnMaxScoreChange`; Q6 bỏ so `version` ở `PUT`: **đỏ** `TestSchemePutVersionConflict`, `TestSchemeEditRace` |
| QC dump màn thật (`scripts/qc-p6-04-ui.spec.ts.txt`: `shots/p604-teacher-1440|375.png/.txt`, `p604-ta-1440`, `p604-link.txt`) | xem mục 2 |
| CI `41198097712` (xong) | **đỏ** chỉ ở `Playwright 2/3` (`panels.spec.ts:562`, xem BUG-P6-04-3); `Go (static)`, `Go test` ×4, `Judge attacks`, `Playwright 1/3, 3/3`, `Lighthouse` ×3, `lint + build` xanh |

## 2. Kết quả TC
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| 01 một `Panel`, sáu khối theo thứ tự, cột trích dẫn | PASS | `panels=1`; khối Thành phần · Điểm cộng · Điểm trừ · Làm tròn · Thang chữ · Liệt; cột phải ≥ 1.024 px hiện văn bản trang với đoạn trích `<mark>`; ảnh `p604-teacher-1440.png` |
| 02 mở đúng trang | PASS | số trang thật của đoạn; thêm `Mở trang 2 → …pdf?sig=…#page=2` (`p604-link.txt`) theo quyết định PM 18:41 (commit `a50ec5b9`); `preview_url` chỉ khi tệp là PDF của đúng lớp, 5 phút, kiểm `TestGetDocumentPreviewURL` |
| 03 `Drawer` dưới 1.024 px | PASS ở mức e2e (`scheme mobile`) | |
| 04 không câu mô tả, không icon Info | PASS | `infoIcons=0`; `gradebook.spec.ts` "scheme review layout" kiểm chữ cấm; tooltip chỉ ở nút `Xác nhận công thức` khoá |
| 05 antipatterns + test bố cục | PASS | rc=0; xanh |
| 06–07 mục chưa rõ tại chỗ; `Xác nhận` khoá kèm lý do; điền xong thì mở | PASS | khối "Làm tròn" trống có ô `Chọn bước` viền đỏ + câu "Quy chế chưa nêu cách làm tròn: điền bước làm tròn." ngay khối; nút `aria-disabled` + tooltip "Còn 1 mục chưa rõ: làm tròn"; `unclear blocks confirm` xanh |
| 08–09 `422 SCHEME_HAS_UNCLEAR` | PASS | `TestConfirmBlockedByUnclear` (Q1 làm đỏ) |
| 10–14 sửa lạc quan, `version + 1`, `filled_by_user`, không trích dẫn, 409, `NO_DRAFT` / `SCHEME_NOT_DRAFT` | PASS | `TestSchemePut*`, `TestFilledByUserNoCitation`, `scheme edit optimistic`, `scheme edit conflict`; Q6 làm đỏ |
| 15–18 xem thử 3 sinh viên, không ghi, lớp chưa đủ điểm | PASS về hành vi; **chưa đo ≤ 300 ms (p95)** | `TestImpactThreeSamples` (cao nhất / giữa / thấp nhất; "6,8 đổi từ 7,0"), `TestImpactNoWrite`, `TestImpactNoCompleteStudents`; ảnh: 3 mẫu `10 (A) → 9,8 (A)`, `7 (B) → 6,8 (B)`, `3 (F) → 3,2 (F)`; chưa có số đo độ trễ (cần lớp 30 người trên stack — US-S8-SEED) |
| 19–24 hộp thoại nêu số, `Idempotency-Key`, audit, `SUPERSEDED` trước `CONFIRMED`, outbox | PASS | `TestConfirmFlow`, `…Idempotent`, `…Audit`, `…RBAC`, `confirm irreversible`; Q2 làm đỏ |
| 25–28 mang điểm; `entries_dropped` kèm lý do; `LOCKED` → 409 | PASS | `TestConfirmCarryOverEntries`, `…NoCarryOnMaxScoreChange`, `…NoCarryOnSourceChange`, `…BlockedWhenLocked`; Q3, Q5 làm đỏ; `confirm irreversible dropped` |
| 29–32 quy chế mới → bản nháp mới + so sánh do code tính | PASS | `TestNewPolicyMakesNewDraft`, `TestDiffAgainstConfirmed`, `TestConfirmedUnchangedUntilConfirm`; e2e `new policy diff`, `new policy upload` (+ failed giữ tên tệp + `Tải lại`) |
| 33–36 đua, mất mạng, trích lỗi | PASS | `TestSchemeEditRace`, `TestConfirmStaleVersion`, `scheme offline retry` (nhãn "Chưa lưu", tự gửi lại) |
| 37–41 thay mock, trạng thái, 375 px, không toast | PASS | grep rỗng; `scheme states`, `scheme mobile`; xác nhận xong chỉ đổi chữ tại chỗ |
| 42–44 thiếu `Idempotency-Key`, thiếu `version`, tooltip bàn phím | PASS | middleware `Idempotency-Key` bắt buộc; contract phủ status khai báo; tooltip cả khi focus (`gradebook.spec.ts`) |
| 45–49 phân quyền | PASS | TA: `inputs=0`, không `Xác nhận công thức` / `Tải quy chế` (dump `p604-ta-1440.txt`); STUDENT bị chặn (`scheme rbac`); ADMIN / GV lớp khác `403` theo #11 (`TestConfirmRBAC`) |

## 3. AC
| AC | Kết quả | Ghi chú |
| --- | --- | --- |
| AC1 khung nhìn đầu — văn bản để duyệt, không tường form | **FAIL** | một `Panel`, sáu khối, cột trích dẫn, không mô tả — nhưng **không "tường ô nhập"** bị vi phạm: màn của TEACHER có **22** ô nhập / chọn luôn hiện (`inputs=22`; mỗi thành phần 3 ô + nút xoá, điểm cộng 3, điểm trừ 3, thang chữ 2 ô × 3 mức…) cả khi mục đã rõ, trông như biểu mẫu (ảnh `p604-teacher-1440.png`, `-375.png`). AC1 và `DESIGN.md` §14.11 yêu cầu văn bản để duyệt; ô điền chỉ ở mục chưa rõ (AC2) hoặc khi sửa |
| AC2 mục chưa rõ chặn xác nhận kèm lý do | PASS | |
| AC3 sửa lạc quan, khoá phiên bản, mục tự điền không trích dẫn | PASS | |
| AC4 xem trước tác động lên 3 sinh viên | **PASS về hành vi; ≤ 300 ms (p95) chưa đo** | chuyển đo sang US-S8-SEED (lớp 30 sinh viên); nếu PM không đồng ý dời → FAIL |
| AC5 xác nhận có hậu quả bằng số, idempotent, chỉ TEACHER | PASS | |
| AC6 giữ điểm đã nhập khi xác nhận công thức mới | PASS | |
| AC7 quy chế mới → bản nháp mới + so sánh | PASS | |
| AC8 đua, mất mạng, lỗi trích | PASS | |
| AC9 phân quyền giao diện và API (TA chỉ đọc) | PASS | theo #11: ngoài lớp `403` |
| AC10 thay mock, trạng thái, 375 px, luật chữ | PASS | |

## 4. Lỗi
- **BUG-P6-04-1 (Trung bình):** `/gradebook/scheme` là tường ô nhập (22 ô luôn hiện) thay vì văn bản để duyệt. Tái hiện: TEACHER mở `/gradebook/scheme` lớp có bản nháp; đếm `main input:visible, textarea:visible, select:visible` = 22 (dump `shots/p604-teacher-1440.txt` dòng đầu). Kỳ vọng (AC1, DESIGN §14.11, luật chữ UI "mỗi dòng nhãn + điều khiển + giá trị"): mỗi mục hiển thị **giá trị dạng chữ** (vd. "Giữa kỳ · 20 % · Quá trình"), ô nhập chỉ hiện ở **mục chưa rõ** hoặc khi giảng viên bấm sửa mục đó (mở dần tại chỗ). `useSchemeEditor` hiện có thể giữ nguyên (lưu khi rời ô); thay phần hiển thị mặc định. Nếu PM coi bản hiện tại hợp lệ (thiết kế "luôn sửa được"), ghi quyết định vào `proposals.md` và đổi chữ AC1 — QC chấm lại.
- **BUG-P6-04-2 (Thấp):** AC4 yêu cầu p95 ≤ 300 ms ở máy chủ cho `POST …/impact`; test của dev không đo độ trễ, QC cũng chưa đo (không có lớp 30 sinh viên trên stack). Đề nghị đo ở US-S8-SEED bằng `p6-load1000` hoặc seed lớp 1 (30 lần, p95).
- Không có lỗi chức năng: 25/25 test Go, 15/15 e2e, 5/5 đột biến bị bắt.

- **BUG-P6-04-3 (Cao — CI đỏ, phá test cũ):** CI run `41198097712` (`a50ec5b9`): chỉ `Frontend e2e (Playwright 2/3)` đỏ, mọi job khác xanh (kể cả Go và Lighthouse simulate / devtools — #19 đã hiệu quả). Ca đỏ: `e2e/panels.spec.ts:562` "bảng 7.2 × vai teacher / ta" — `teacher /documents@1440` và `@1024` trả `{"panels":0,…}` (cần ≥ 1 Panel), cả retry; **QC tái hiện cục bộ** (`pnpm exec playwright test panels.spec.ts -g "bảng 7.2 × vai" --project=desktop` → 2 failed; trước US-P6-04 ở `f07c2af6` ca này xanh). Nghi nguyên nhân (QC chưa xác nhận bằng bisect): dev thêm `await gradeApi(page)` vào `mockStaffApi` chung (`e2e/support/staffMock.ts`) và `a50ec5b9` nới regex route giả của `gradeApi` thành `…/(grade-scheme|documents\/[^/]+(\/chunks)?(\?|$))` — có thể chặn các `GET …/documents/…` mà màn `/documents` cần, trả 404 → màn không dựng Panel. Tái hiện: bước trên; kỳ vọng: `/documents` có ≥ 1 Panel. Sửa: giới hạn route giả của `gradeApi` đúng URL cần (`…/documents/{id}` và `…/documents/{id}/chunks` của tài liệu mẫu, `fallback()` cho các URL khác), hoặc không nhét `gradeApi` vào `mockStaffApi`; không sửa `panels.spec.ts`.

## 5. Kiểm phản mẫu UI / luật mở rộng / phân quyền
- `ConfirmIrreversible` nêu số (áp cho / đổi / được giữ / bị bỏ + lý do), `Idempotency-Key`; không toast "Thành công"; không icon Info.
- Luật 14 (`version` → 409): `PUT` và `confirm`; luật 12: trích công thức là việc nền (`useJob`); luật 7: 4 route mới có trong `openapi.yaml`, dòng `want` (#9) và kịch bản contract cho từng status.
- Test cũ bị sửa: `contract_test.go` (số thao tác 228 → 232, #12 mở rộng PM 18:41), `e2e/support/grade-fixtures.ts` (mở rộng route giả cho `GET …/documents/{id}` — tệp của chính story); không sửa test cũ khác.
- `00016` / `ARCHITECTURE`: không đổi thêm trong story này. Không thư viện mới; không secret.

## Đề nghị
1. Dev sửa BUG-P6-04-1 (hiển thị chữ, ô nhập mở dần) hoặc PM duyệt thiết kế hiện tại qua `proposals.md`.
2. QC đo p95 `impact` (BUG-P6-04-2) ở US-S8-SEED.
3. Dev sửa BUG-P6-04-3 (CI đỏ `panels.spec.ts` / `/documents`) trước mọi việc khác.
4. Vòng 2: QC chạy lại `gradebook.spec.ts`, `panels.spec.ts`, đếm ô nhập mặc định, 375 px, và đợi CI xanh.

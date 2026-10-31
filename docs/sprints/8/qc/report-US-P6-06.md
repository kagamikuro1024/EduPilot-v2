# QC report — US-P6-06 (`/gradebook`: sổ điểm thật, sửa ô kiểu bảng tính, xem trước tức thì, 409, giải trình, lịch sử, công bố cột, 1.000 dòng) · Kết luận vòng 2: **PASS** (xem "Vòng 2"). Vòng 1: **FAIL** (CI đỏ do chính story: `shell.spec.ts:395` dùng `/gradebook` thật không có API giả; AC1–AC13 PASS ở mức hành vi, có nợ chuyển đo ở stack)

## Vòng 2 (2026-10-31, `sprint/8-p6` @ `13646f87`)
| Lỗi vòng 1 | Vòng 2 |
| --- | --- |
| BUG-P6-06-1 `shell.spec.ts:395` đỏ | **ĐÃ SỬA** (#23: `shell.spec` 375 / 400 / 405 → `/analytics`, không đổi assertion); QC chạy `gradebook.spec shell.spec tokens.spec` → 71 passed, 0 failed (gồm `regression-1.5`); CI run `41202190312` **success** (mọi job xanh) |
| BUG-P6-06-2 tự sửa test trước ACCEPTED | đóng (#22 ACCEPTED, PM ghi lỗi quy trình lần 4) |
| BUG-P6-06-3 `private-chat.spec:219` flaky | không tái hiện ở CI vòng này |
Mọi AC giữ PASS. Story **PASS** (nợ: đo p95 / đua `PUT` trên stack lớp 1.000 → US-S8-SEED).

---
## Vòng 1
Ngày 2026-10-31. Commit kiểm: `sprint/8-p6` @ `b8284b73`. Bộ TC: `tc-US-P6-06.md` (chữ TC chỉ đổi theo #11). Handoff: `dev-US-P6-06.md`. CI run `41200524277`: **đỏ** chỉ ở `Frontend e2e (Playwright 2/3)` — `shell.spec.ts:395` (đỏ cả retry) và `private-chat.spec.ts:219` "escalate offline" (đỏ một lần, không thuộc story); Go ×4, Judge, Lighthouse, Playwright 1/3, 3/3 xanh.

## 1. Cổng nghiệm thu đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `p6-go-tests.sh ./internal/grade "<24 test của US-P6-06>" -race` (`TestPutEntry*`, `TestEntryHistoryFromAudit`, `TestPublish*`, `TestGradebookWarningsPenaltyNotInScheme`, `TestGradebookPageP95`, `TestGradebookNoNPlusOne`, `TestGradebookExplainNoSeqScan`, `TestPreviewUsesCompute`, `TestGradebookRBAC`, `TestPreviewRBAC`, `TestRBACMatrix`) | PASS 24/24 (không thiếu, không SKIP, không DATA RACE) |
| `gradebook.spec.ts` (bản dựng `build:gate`, `--workers=2 --retries=0`) | PASS 29 desktop (mobile bỏ qua theo thiết kế); `ui-antipatterns.sh` rc=0; `pnpm lint` sạch |
| Hồi quy rộng `panels tokens class-join students a11y shell lcp settings-llm attendance me private-chat` | 260 passed, 198 skipped, 0 failed khi chạy cả bộ; **nhưng chạy riêng `shell.spec.ts -g "regression-1.5"` → FAIL** (tái hiện CI; xem BUG-P6-06-1) |
| Đột biến (bản sao `/tmp/qc-mut`, đã xoá) | G1 nới thang `max_score`: **đỏ** `TestPutEntryOutOfRange`; G2 tắt kiểm 3+ chữ số thập phân: **đỏ** `TestPutEntryPrecision`; G3 bỏ so `version`: **đỏ** `TestPutEntryClear`, `TestPutEntryConflictBody`; G4 bỏ khoá `LOCKED`: **đỏ** `TestPutEntryLocked`; G6 cho sửa ô bài thi: **đỏ** `TestPutEntryExamItemRejected`; G5 audit không fail-closed: xanh — **đột biến tương đương**: lỗi `INSERT audit_log` làm giao dịch PG bị huỷ nên các câu lệnh sau cũng lỗi, điểm không ghi (không phải lỗ hổng) |
| QC dump (`scripts/qc-p6-06-ui.spec.ts.txt` → `shots/p606-*.png/.txt`) | `/gradebook` teacher 1440: `panels=1`, `officialNote=1`, `scroll 1440/1440`; TA: `inputs=0`; lớp chưa có công thức: banner + `Tải quy chế` + `Chốt điểm` khoá + EmptyState "Chưa có công thức điểm." + `Thiết lập công thức` |
| `grep -rn "mock/gradebook" frontend/src`; `go.mod` / `package.json` | rỗng; không đổi (không thư viện mới) |
| `golangci-lint`, `sqlc diff`, `contract` (238 thao tác, #12 mở rộng) | PASS (dev; CI Go xanh) |

## 2. Kết quả TC
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| 01 một `Panel`; cột Sinh viên sticky; cột theo item + Điểm cộng/trừ, Quá trình, Điểm học phần, Điểm chữ, Trạng thái; không KPI | PASS | dump + `gradebook layout`; ảnh `p606-teacher-1440.png` |
| 01 (phần menu) nút `Chốt điểm` chính và `⋯` (`Nhập từ XLSX`, `Xuất XLSX`, `Mở khoá`) | **chưa có khi công thức đã xác nhận** — handoff: chỉ biến thể `Chốt điểm` khoá; `⋯` ở US-P6-07 / 08 | kiểm ở US-P6-07 AC9, US-P6-08 AC9 (AC của hai story đó nêu đúng các mục này); ghi để PM biết, không FAIL riêng |
| 02 đúng một dòng ghi chú | PASS | `officialNote=1`; `official note` |
| 03–04 Enter xuống, Tab sang, Esc hoàn tác; `8,5`/`8.5`; ngoài thang / 3 chữ số thập phân / chữ → lỗi tại ô giữ chữ; xoá = xoá điểm; ô EXAM khoá + tooltip | PASS | `cell edit keys`, `cell edit instant`; Go `TestPutEntry{OK,OutOfRange,Precision,Clear,ExamItemRejected}`; G1, G2, G6 làm đỏ |
| 15–17 xem trước tức thì ≤ 300 ms, ô "…" trong lúc chờ | PASS về hành vi | `TestPutEntryReturnsRowResult`, `TestPutEntryUnder300ms`; "…" kiểm ở `cell edit instant`; p95 trên stack: US-S8-SEED |
| 18–22 409: popover "{tên} vừa nhập {điểm} · {tuổi}", `Giữ của tôi` / `Dùng của họ` | PASS | `cell conflict choose`, `TestPutEntryConflictBody`; G3 làm đỏ; chỉ Staff thấy tên (`updated_by` chỉ ở route Staff) |
| 23–27 giải trình B: `QT₀ 7,75 → +0,50 → 8,25 → HP thô 7,20 → 7,0 → B`, `Căn cứ · tr. N`, "do giảng viên đặt" | PASS ở mức e2e với dữ liệu giả cố định (`explain student B`); khớp dữ liệu thật trên lớp 1: chờ US-S8-SEED | |
| 28–29 lịch sử ≤ 50 mục từ `audit_log`, phân trang con trỏ, không nút luôn hiện | PASS | `TestEntryHistoryFromAudit`; menu dòng |
| 30–36 1.000 dòng: trang đầu p95 ≤ 500 ms, DOM ≤ 60 hàng, gõ dở rồi cuộn giữ chữ, không `Seq Scan`, không N+1; `cellNav` ở `shared/` | PASS ở mức test | `virtualized 1000`, `type then scroll keeps text`, `TestGradebookPageP95`, `TestGradebookNoNPlusOne`, `TestGradebookExplainNoSeqScan`; `DataTableVirtual` mở rộng ở `shared/ui` (không bảng riêng); p95 trên stack lớp 1.000: US-S8-SEED. **Ghi nhận:** `limit > 100` hoặc `< 1` được thay bằng 30 (không 422, không kẹp 100) — Q-06-b chấp nhận |
| 37–42 chưa công thức → EmptyState + `Thiết lập công thức`; lớp rỗng; mất mạng; `LOCKED` → 409; trạng thái tải / lỗi | PASS | `gradebook states`, `offline cell`, `TestPutEntryLocked`; dump lớp chưa công thức |
| 43–46 audit cùng giao dịch, fail-closed; thay mock; tab Điểm hồ sơ 360 | PASS | `TestPutEntryAuditRow`, `TestPutEntryAuditFailClosed`; grep rỗng; `GradeCalculation` dùng ở tab Điểm (`shared/domain`) |
| 47–52 công bố cột `Công bố cho sinh viên` / `Gỡ`, cột EXAM không có mục, `LOCKED` → 409, TA không thấy | PASS | `publish column`, `TestPublish{ItemToggle,ItemAudit,ExamItemNotToggleable,LockedRejected}` |
| 53–55 cảnh báo `PENALTY_NOT_IN_SCHEME` / `BONUS_NOT_IN_SCHEME` | PASS | `penalty not in scheme notice`, `TestGradebookWarningsPenaltyNotInScheme` |
| 56–59 `PUT` thiếu `version`, đua 20 lượt | PASS ở mức test (`TestPutEntryConflictBody`; đua song song trên stack: QC chưa chạy `p6-put-race.sh`) | |
| 60–63 phân quyền | PASS | TA: `inputs=0`, không `Chốt điểm`, `⋯` hay menu cột công bố; `TestGradebookRBAC`, `TestRBACMatrix`, `TestPreviewRBAC`; STUDENT chặn (`gradebook rbac`); ngoài lớp `403` theo #11 |

## 3. AC
AC1 PASS (phần menu `⋯` / `Chốt điểm` chính chuyển nơi kiểm sang US-P6-07 / 08 như trên) · AC2 PASS · AC3 PASS · AC4 PASS (≤ 300 ms đo trong test) · AC5 PASS · AC6 PASS (mức e2e) · AC7 PASS · AC8 PASS (mức test; p95 stack ở SEED) · AC9 PASS · AC10 PASS · AC11 PASS · AC12 PASS · AC13 PASS.

## 4. Lỗi
- **BUG-P6-06-1 (Cao — CI đỏ, phá test cũ):** `e2e/shell.spec.ts:395` "regression-1.5: LEFT page-title 240 / 12, brand 56, logo 32, mark 28" gọi `page.goto("/gradebook")` (dòng 400, 405) khi màn đã thành thật và không có API giả → không có tiêu đề trang → `page.evaluate(LEFT_SRC)` = `null` (CI: `Expected: 240, Received: null`, đỏ cả retry; QC tái hiện: `pnpm exec playwright test shell.spec.ts -g "regression-1.5" --project=desktop` → 1 failed). Cùng dạng #21 / #22 (ca cũ dùng `/gradebook` làm màn mô phỏng), dev bỏ sót hai chỗ `shell.spec.ts:400` và `:405` (ca dòng 375 cũng dùng `/gradebook` nhưng vẫn xanh). Chạy cả bộ `shell.spec` trong một lượt cho kết quả xanh trên máy QC nên khó thấy; chạy riêng thì đỏ. Kỳ vọng: sửa như #21 (đổi route sang `/analytics`, không đổi assertion) hoặc thêm `gradebookApi(page)` — **mở proposal và chờ PM** trước khi sửa test cũ.
- **BUG-P6-06-2 (Quy trình, Trung bình):** `tokens.spec.ts` (+`gradebookApi`) và `visual.spec.ts` (+`gradebookApi`) đã bị sửa theo **#22** trong khi `proposals.md` chưa có quyết định ACCEPTED cho #22 ("PM hoàn tác nếu không đồng ý" — dev tự chạy trước). Assertion giữ nguyên, thay đổi chỉ thêm dòng thiết lập; vẫn là lần nữa sửa test cũ trước ACCEPTED (luật `dev.md`, PM nhắc ở #7).
- **BUG-P6-06-3 (Thấp, ngoài story):** `private-chat.spec.ts:219` "escalate offline" đỏ một lần ở CI (xanh ở retry / máy QC): nghi ca mong manh, không chặn.

## 5. Kiểm phản mẫu UI / luật mở rộng / phân quyền
- Một `Panel`, không KPI, không card lồng card; ảnh mốc `gradebook-1440/390.png` sinh lại trong image Playwright (liệt kê ở handoff; CI `visual.spec.ts` xanh); số thập phân dấu phẩy; không toast.
- Luật 13: phân trang con trỏ ≤ 100, không N+1 (test); luật 14: `version` → 409; luật 15: điểm không cache phía máy chủ.
- Test cũ bị sửa: `contract_test.go` 232 → 238 (#12 mở rộng, PM đã cho); `class-join.spec.ts` ×2 route → `/analytics` (#21 ACCEPTED); `tokens.spec.ts`, `visual.spec.ts` (#22, chưa ACCEPTED — BUG-P6-06-2).

## Cập nhật 19:40 — quyết định PM (`proposals.md` #22, #23 ACCEPTED; #24)
- BUG-P6-06-2: **đóng** (PM chấp nhận #22; ghi lỗi quy trình lần 4; từ nay cùng dạng được làm trước, ghi số proposal vào commit).
- BUG-P6-06-1: PM chấp nhận #23 (`shell.spec.ts` 375 / 400 / 405 → `/analytics`, không đổi assertion). **Vẫn FAIL cho tới khi dev đẩy sửa và CI xanh**; vòng 2 QC chạy lại `shell.spec.ts -g "regression-1.5"` riêng từng ca.
- #24 (`ui-antipatterns.sh` thêm `ImportDialog`) thuộc US-P6-07, QC kiểm ở đó.

## Đề nghị
1. Dev mở proposal sửa `shell.spec.ts:400, :405` (route hoặc `gradebookApi`), báo PM qua herdr, chờ ACCEPTED rồi sửa; PM quyết #22.
2. Chuyển đo p95 (500 ms trang đầu 1.000 sinh viên; 300 ms `PUT`) + đua `PUT` song song sang US-S8-SEED (như US-P6-04).
3. Vòng 2: QC chạy lại `shell.spec.ts` từng ca, `gradebook.spec.ts`, CI xanh.

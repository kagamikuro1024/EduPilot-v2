# QC report — US-P6-08 (chốt điểm: snapshot bất biến, khoá, mở khoá có lý do, chốt lại v(N+1), thông báo, đua) · Kết luận vòng 1: **PASS** (CI: xem mục 4)
Ngày 2026-10-31. Commit kiểm: `sprint/8-p6` @ `309576d4`. Bộ TC: `tc-US-P6-08.md` (TC-P6-08-01…47; chữ không đổi từ lúc viết, chỉ ghi #11). Handoff: `dev-US-P6-08.md`.

## 1. Cổng nghiệm thu đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `p6-go-tests.sh ./internal/grade "<31 test: toàn bộ func Test trong finalize_test.go + mọi tên trong TC-P6-08>" -race -tags integration` | PASS 31/31 (không thiếu, không SKIP, không DATA RACE): `TestFinalizeSnapshotsAll`, `…EqualsPreview`, `…Idempotent`, `…OneTxAndAudit`, `…Lop1MatchesExpected` (30/30 khớp `seed/expected_final_grades.csv`), `…NotConfirmed`, `…IncompleteAllOrNothing`, `…Incomplete40`, `…AlreadyLocked`, `…UnpublishedExamIsIncomplete`, `TestLockedBlocksWrites`, `TestSnapshotUnaffectedBySourceChange`, `TestDriftFlag`, `TestUnlockRequiresReason`, `…WhenOpen`, `…Audit`, `…RBAC`, `TestFinalizeRBAC`, `TestRefinalizeNewVersion`, `TestOldSnapshotHashUnchanged`, `TestSnapshotHistoryReadable`, `TestRefinalizeSkipsLeftStudent`, `TestFinalizeNotifiesAll`, `TestRefinalizeNotifiesOnlyChanged`, `TestNotifyNoNumbers`, `TestNotifyDedupe`, `TestFinalizeConcurrent`, `TestFinalizeVsPutRace`, `TestWriteTxLocksGradebookShared`, `TestFinalize1000Under3s`, `TestFinalizeNoNPlusOne` |
| Đột biến (bản sao `/tmp/qc-mut`, đã xoá) | F1 `version` không +1: **đỏ** 24 test; F2 bỏ cận dưới lý do mở khoá (10): **đỏ** `TestUnlockRequiresReason`; F3 bỏ `NOT_LOCKED`: **đỏ** `TestUnlockWhenOpen`; F4 bỏ `GRADES_INCOMPLETE`: **đỏ** `…IncompleteAllOrNothing`, `…Incomplete40`, `…UnpublishedExamIsIncomplete`; F5 `GRADE_CHANGED` gửi cho cả lớp: **đỏ** `TestRefinalizeNotifiesOnlyChanged`; F6 bỏ `GRADEBOOK_LOCKED` ở `finalize`: **đỏ** `…Idempotent`, `…AlreadyLocked`, `…Concurrent`; F7 bỏ `FOR UPDATE` hàng `gradebooks`: **đỏ** `…Concurrent`, `…VsPutRace` — 7/7 bị bắt |
| **Probe QC đua `import commit` ↔ `finalize`** (`scripts/p6_qc_probe_import_race_test.go.txt`, chép vào gói rồi xoá; **đóng proposal #25**) | 25 vòng: 9 lần nhập thắng (snapshot có điểm mới = 10), 16 lần nhập bị `GRADEBOOK_LOCKED` (snapshot giữ 6), không trạng thái nửa vời; **đột biến F7 làm probe đỏ** ("nhập thành công trước chốt phải có trong snapshot") → probe nhạy với khoá |
| `gradebook.spec.ts shell.spec.ts tokens.spec.ts panels.spec.ts class-join.spec.ts a11y.spec.ts` (`build:gate`, `--workers=2 --retries=0`) | 216 passed, 0 failed (gồm `finalize flow`, `finalize incomplete list`, `unlock reason`, `snapshot history`); `ui-antipatterns.sh` rc=0; `pnpm lint` sạch |
| QC dump (`scripts/qc-p6-08-ui.spec.ts.txt` → `shots/p608-*`) | sổ chốt: "Đã chốt v2 · 10:00 30-10", `inputs=0` (ô khoá), **không** nút `Chốt điểm`, đúng 1 dòng ghi chú điểm chính thức, 1440 và 375 không tràn ngang; menu TEACHER `Xuất XLSX \| Lịch sử chốt \| Mở khoá` (`Nhập từ XLSX` ẩn khi chốt), TA chỉ `Lịch sử chốt`; `Khác bản chốt` hiện ở hàng drift (số = 1) và không hiện khi không drift; hộp Mở khoá: "Mở khoá bản chốt v2 của 30 sinh viên. Bản v2 vẫn được giữ nguyên. Nhập lý do 10–500 ký tự."; nút khoá ở độ dài 0, 9, 501 và 10 dấu cách, mở ở 10 và 500 |

## 2. Kết quả theo AC
| AC | Kết quả | Bằng chứng |
| --- | --- | --- |
| AC1 chốt: snapshot 30 sinh viên bằng xem trước, một giao dịch, audit, idempotent | PASS | `SnapshotsAll`, `EqualsPreview`, `Idempotent`, `OneTxAndAudit`, `Lop1MatchesExpected`; F1, F6 làm đỏ |
| AC2 `409 SCHEME_NOT_CONFIRMED` / `GRADES_INCOMPLETE` (≤ 30, `total`) / `GRADEBOOK_LOCKED`; EXAM chưa công bố không thành 0 | PASS | `NotConfirmed`, `IncompleteAllOrNothing`, `Incomplete40`, `AlreadyLocked`, `UnpublishedExamIsIncomplete`; F4, F6 |
| AC3 khoá ghi; snapshot không đổi khi nguồn đổi; nhãn "Khác bản chốt" | PASS | `LockedBlocksWrites`, `SnapshotUnaffectedBySourceChange`, `DriftFlag`; dump drift |
| AC4 mở khoá có lý do 10–500, `NOT_LOCKED`, phân quyền | PASS | `UnlockRequiresReason`, `WhenOpen`, `Audit`, `RBAC`; F2, F3; dump hộp Mở khoá |
| AC5 chốt lại v(N+1), v1 không đổi (băm), lịch sử đọc được | PASS | `RefinalizeNewVersion`, `OldSnapshotHashUnchanged`, `SnapshotHistoryReadable`, `RefinalizeSkipsLeftStudent`; F1 |
| AC6 chuông `GRADE_FINALIZED` (cả lớp) / `GRADE_CHANGED` (chỉ người đổi), không số điểm, khử trùng | PASS | `FinalizeNotifiesAll`, `RefinalizeNotifiesOnlyChanged`, `NotifyNoNumbers`, `NotifyDedupe`; F5 |
| AC7 đua: hai `finalize`, `finalize` ↔ `PUT`, `finalize` ↔ nhập | PASS | `FinalizeConcurrent`, `FinalizeVsPutRace`; probe QC (nhập ↔ chốt); F7 |
| AC8 hiệu năng, không N+1, không `Seq Scan` | PASS ở mức test (đo trên stack 1.000 sinh viên → US-S8-SEED) | `Finalize1000Under3s`, `FinalizeNoNPlusOne` |
| AC9 UI: hậu quả bằng số, "Đã chốt v{n}", danh sách thiếu kèm liên kết, không toast | PASS ở mức e2e với dữ liệu giả; **phân bố thật "A+ 3, A 3, B+ 3, B 5, C+ 4, C 4, D+ 2, D 3, F 3 · 1 sinh viên liệt" kiểm ở US-S8-SEED** (TC-P6-08-36) | `finalize flow`, `finalize incomplete list` |
| AC10 phân quyền | PASS | `TestFinalizeRBAC`, `TestUnlockRBAC`; dump TA |

## 3. Lỗi / ghi nhận
- **BUG-P6-08-1 (Thấp):** `TestWriteTxLocksGradebookShared` đọc **mã nguồn** (`os.ReadFile`) và kiểm chuỗi `GradebookStatusShare` / `FOR SHARE`; chuỗi này cũng có trong chú thích của `import.go`, nên xoá lời gọi khoá vẫn xanh. Đây là test kiểm chữ trong nguồn, không kiểm hành vi; hành vi thật do `TestFinalizeVsPutRace` và probe QC (ở trên) giữ. Đề nghị thay bằng test đua cho nhập (probe QC có thể dùng làm mẫu) hoặc xoá.
- Proposal #25 (QC): **đã kiểm xong ở story này** → US-P6-07 giữ PASS, không còn nợ đua.

## 4. CI
Run `41203465230` (`309576d4`): lần đầu **đỏ ở `Frontend e2e (Playwright 1/3)`** do `data-layer.spec.ts` (test cũ, không nằm trong diff story): `:344` "sse parse + singleton" (`log[0].headers.authorization` = `undefined`, đỏ cả retry) và `:258` "auto refresh" (flaky). QC chạy cục bộ `data-layer.spec.ts`: 29 passed. QC `gh run rerun --failed` → **success** (mọi job xanh). Kết luận: **flaky ở CI, không do story**; verdict **PASS**.
- **BUG-P6-08-2 (Thấp, ngoài story):** `data-layer.spec.ts:344` / `:258` đỏ một lần ở CI rồi xanh khi chạy lại (nghi phụ thuộc thời gian / tải runner); đề nghị dev xem lại ổn định (không chặn).

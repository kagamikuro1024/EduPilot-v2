# QC test case — US-P6-08 (`Chốt điểm` snapshot bất biến, khoá, `Mở khoá có lý do`, chốt lại v(N+1), thông báo, đua)
Nguồn: `docs/specs/FEAT-gradebook/US.md` US-P6-08 (AC1–AC10) + `SRS.md` 3, 4.7, 6.2 (#14–#16); TLR-6, TLR-15, TLR-16; `FLOWS.md` F10 nhánh "Sửa sau khi chốt". Viết trước khi có code, không đọc code của dev (v1, 2026-10-31).

**Quy ước.** Lớp 1 seed: `CONFIRMED`, đủ điểm, chưa chốt. Chuỗi TC này **tuần tự trên một DB dựng lại** (chốt v1 → mở khoá → sửa → chốt v2). `SNAP(v)` = `$PSQL "select student_id, numeric_grade, letter, failed_reason, md5(breakdown::text) from final_grade_snapshots where course_id='$C1' and version=v order by student_id"`; băm cả bảng: `… | md5sum`. `p6-put-race.sh` QC viết pha 2 (xargs -P song song). `GO` = `p6-go-tests.sh`.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P6-08-01 | AC1 | lớp 1 `CONFIRMED`, 30 sinh viên đủ điểm | `curl -X POST $Q/gradebook/finalize -H "$TCH" -H "$(idem)"` → ghi `K` | `200 {version:1, students:30, locked_at}`; `final_grade_snapshots` có **30** dòng `version=1` (mọi sinh viên `ACTIVE`); `breakdown` có đầu vào + trung gian + dòng giải trình có `source_quote`; `scheme_id`, `locked_at`, `locked_by` = teacher; `gradebooks.status='LOCKED'`, `snapshot_version=1`; `audit_log` `action='finalize'` kèm số sinh viên 30; outbox `grade.finalized` |
| TC-P6-08-02 | AC1 | như trên | So `SNAP(1)` với `seed/expected_final_grades.csv` (QC: `numeric_grade`↔`hp`, `letter`, `failed_reason`↔`verdict`) | **30/30 khớp** (A 9,5 A+; B 7,0 B; C 3,0 F; sv20 5,0 F kèm lý do liệt; sv06 10,0 A+) |
| TC-P6-08-03 | AC1 | trước khi chốt | Lưu `GET $Q/gradebook/preview?limit=100` ra `/tmp/pre.json` rồi chốt; so `numeric_grade` với `hp` trong `/tmp/pre.json` | **bằng nhau** từng sinh viên (snapshot = xem trước tại thời điểm chốt) |
| TC-P6-08-04 | AC1 | như trên | Gửi lại `finalize` cùng `Idempotency-Key K` | cùng phản hồi, **không** snapshot thứ hai (`count(*)` = 30, `snapshot_version=1`) |
| TC-P6-08-05 | AC1 | repo | `GO ./internal/grade "TestFinalizeSnapshotsAll|TestFinalizeEqualsPreview|TestFinalizeIdempotent|TestFinalizeOneTxAndAudit|TestFinalizeLop1MatchesExpected" -tags integration` | PASS |
| TC-P6-08-06 | AC2 | seed lớp 2 `NO_POLICY` | `curl -s -X POST $Q2/gradebook/finalize -H "$TCH" -H "$(idem)" -o /dev/null -w "%{http_code}"` | `409` `SCHEME_NOT_CONFIRMED` (lệnh cổng P6) |
| TC-P6-08-07 | AC2 | lớp 1 `CONFIRMED`; xoá ô cuối kỳ sv30 và sv27 (hai sinh viên thiếu) | `finalize` | `409 GRADES_INCOMPLETE`, `details.total = 2`, `details.students[]` = `[{id, missing:["final"]},…]` (≤ 30 phần tử); **không** snapshot nào (`count(*)` = 0; `gradebooks.status` vẫn `OPEN`) — tất cả hoặc không |
| TC-P6-08-08 | AC2 | `details.students` | Với lớp có 40 sinh viên thiếu (QC dựng lớp thử) | `details.students.length = 30`, `details.total = 40` |
| TC-P6-08-09 | AC2 | sổ `LOCKED` | `finalize` lần hai (khoá mới) | `409 GRADEBOOK_LOCKED` |
| TC-P6-08-10 | AC2 | UI | `Chốt điểm` khi thiếu điểm | hiện danh sách sinh viên thiếu kèm liên kết tới ô (click → cuộn tới/focus ô đó) |
| TC-P6-08-11 | AC2 | repo | `GO ./internal/grade "TestFinalizeNotConfirmed|TestFinalizeIncompleteAllOrNothing|TestFinalizeAlreadyLocked" -tags integration` | PASS |
| TC-P6-08-12 | AC3 | sổ `LOCKED` v1 | `PUT $Q/grade-entries` (ô giữa kỳ sv04); `POST …/grade-entries/import commit`; `POST …/grade-items/{id}/publish`; `POST …/grade-scheme/confirm` | cả 4 `409 GRADEBOOK_LOCKED`; `GET` vẫn `200`; `/gradebook` hiện "Đã chốt v1" và mọi ô khoá |
| TC-P6-08-13 | AC3 | như trên | Sửa điểm bài thi PE tuần 9 của sv04 ở `/exams` (nguồn sống) và ghi thêm điểm danh vắng; xem `SNAP(1)` | snapshot **không** đổi (băm bảng trước = sau); `/gradebook` hiển thị nhãn **"Khác bản chốt"** ở hàng sv04 (chỉ Staff; STUDENT không có đường thấy) |
| TC-P6-08-14 | AC3 | repo | `GO ./internal/grade "TestLockedBlocksWrites|TestSnapshotUnaffectedBySourceChange|TestDriftFlag" -tags integration` | PASS |
| TC-P6-08-15 | AC4 | sổ `LOCKED` v1 | UI: `⋯` → `Mở khoá` | `<ConfirmIrreversible>` nêu số: "Mở khoá bản chốt v1 của 30 sinh viên. Bản v1 vẫn được giữ nguyên."; ô lý do bắt buộc **10–500** ký tự; nút xác nhận khoá khi lý do ngắn |
| TC-P6-08-16 | AC4 | như trên | `curl -X POST $Q/gradebook/unlock -H "$TCH" -H "$(idem)" -d '{"reason":"Phúc khảo sv04: sửa điểm giữa kỳ"}'` | `200`; `gradebooks.status='OPEN'`, `unlock_reason` lưu đúng, `unlocked_by` = teacher, `unlocked_at`; `audit_log` có dòng `unlock`; `SNAP(1)` không đổi |
| TC-P6-08-17 | AC4 | sổ `LOCKED` | `reason` thiếu / `""` / 9 ký tự / 501 ký tự / chỉ khoảng trắng (10 dấu cách) | `422 VALIDATION_FAILED` mọi ca; chốt biên: đúng 10 và đúng 500 ký tự → `200` |
| TC-P6-08-18 | AC4 | sổ `OPEN` | `unlock` | `409 NOT_LOCKED` |
| TC-P6-08-19 | AC4 (phân quyền) | — | `for T in "$TA_" "$SVA" "$ADM"`: `finalize` và `unlock` | `403` 6/6; giảng viên lớp khác → `404` |
| TC-P6-08-20 | AC4 | repo | `GO ./internal/grade "TestUnlockRequiresReason|TestUnlockWhenOpen|TestUnlockAudit|TestUnlockRBAC" -tags integration`; `playwright test gradebook.spec.ts -g 'unlock reason'` | PASS; xanh |
| TC-P6-08-21 | AC5 | đã mở khoá (TC-16); lưu `H1=$(SNAP(1) \| md5sum)` | Sửa **hai** ô: (a) sv08 cuối kỳ `8,25 → 9,25` (QC tính: QT 8,25; HP thô (40·8,25+60·9,25)/100 = 8,85 → **9,0 A+**; trước đó 8,25 → 8,5 A) — sinh viên **đổi**; (b) sv04 giữa kỳ `6,25 → 7,25` (QT₀ 6,625 +0,5 = 7,125; HP thô (285+360)/100 = 6,45 → **6,5 C+**, không đổi so với trước 6,25 → 6,5 C+) — điểm thành phần đổi nhưng `numeric_grade` và `letter` **không** đổi | sửa thành công (200) cả hai; xem trước sv08 `9,0 A+`, sv04 `6,5 C+` (khớp tính tay) |
| TC-P6-08-22 | AC5 | như trên | `finalize` lần hai | `200 {version:2}`; `snapshot` v2 **30 dòng** mang `unlock_reason` của lần mở khoá; `gradebooks.snapshot_version=2`, `status='LOCKED'` |
| TC-P6-08-23 | AC5 | như trên | `H2=$(SNAP(1) \| md5sum)` | `H2 == H1` — bản v1 **không đổi một byte** (so băm trước/sau); thử `UPDATE final_grade_snapshots SET letter='A' WHERE version=1` bằng vai trò ứng dụng → `P0001` |
| TC-P6-08-24 | AC5 | như trên | `/gradebook` và `export.xlsx` | dùng v2: hàng đầu "Bản chốt v2"; sv08 `9,0 A+`; `Lịch sử chốt` chọn được v1 (xem `GET $Q/gradebook/snapshots`, `…/snapshots/1?cursor=`; sv08 ở v1 = `8,5 A`) |
| TC-P6-08-25 | AC5 | repo | `GO ./internal/grade "TestRefinalizeNewVersion|TestOldSnapshotHashUnchanged|TestSnapshotHistoryReadable" -tags integration` | PASS |
| TC-P6-08-26 | AC6 | sau TC-22 | `$PSQL "select u.email, n.type, n.body from notifications n join users u on u.id=n.user_id where n.type='GRADE_CHANGED'"` | **đúng 1** thông báo `GRADE_CHANGED` (sv08); nội dung "Điểm học phần lớp 761987 đã được cập nhật." **không có con số** (không "9,0", không "A+"); sv04 (HP không đổi) và 28 còn lại **không** nhận |
| TC-P6-08-27 | AC6 | sau TC-01 (lần chốt đầu) | `… type='GRADE_FINALIZED'` | **30** thông báo (cả lớp); "Điểm học phần lớp 761987 đã được chốt."; khử trùng: gửi lại `finalize` cùng khoá → không thêm |
| TC-P6-08-28 | AC6 | — | Kiểm kênh | chỉ chuông: không mail (Mailpit `:8025` không có thư điểm), không mục "Hôm nay" ở sinh viên cho sự kiện này |
| TC-P6-08-29 | AC6 | repo | `GO ./internal/grade "TestRefinalizeNotifiesOnlyChanged|TestFinalizeNotifiesAll|TestNotifyNoNumbers|TestNotifyDedupe" -tags integration` | PASS |
| TC-P6-08-30 | AC7 | `p6-put-race.sh` | 2 `finalize` song song (khoá khác nhau) | đúng **một** `200`; kia `409 GRADEBOOK_LOCKED`; `count(distinct version)` = 1; 30 dòng |
| TC-P6-08-31 | AC7 | như trên | `PUT` điểm ngay lúc `finalize`: lặp **20 lần** (dựng lại DB mỗi lần hoặc mở khoá + chốt lại): `finalize & PUT` song song | mỗi lần: nếu `PUT` thành công (200) **trước** chốt → giá trị có trong snapshot; nếu `409 GRADEBOOK_LOCKED` → snapshot không có; **không bao giờ** snapshot "nửa mới nửa cũ" (so snapshot với `grade_entries` tại `locked_at`); 20/20 lần đúng |
| TC-P6-08-32 | AC7 | repo | `GO ./internal/grade "TestFinalizeConcurrent|TestFinalizeVsPutRace|TestWriteTxLocksGradebookShared" -race -tags integration` | PASS, không `DATA RACE` |
| TC-P6-08-33 | AC8 | `LOAD1000` | `finalize` đo `time_total` (3 lần, max); đọc bản chốt trang đầu `GET …/snapshots/1?limit=100` | ≤ 3 s; ≤ 500 ms; số câu SQL của `finalize` 1.000 sv = số câu của 30 sv (bật log) |
| TC-P6-08-34 | AC8 | `LOAD1000`; `EXPLAIN` truy vấn đọc bản chốt | không `Seq Scan` trên `final_grade_snapshots` (TLR-15) |
| TC-P6-08-35 | AC8 | repo | `GO ./internal/grade "TestFinalize1000Under3s|TestFinalizeNoNPlusOne" -tags integration` | PASS |
| TC-P6-08-36 | AC9 | UI, lớp 1 đủ điểm | `Chốt điểm` | `<ConfirmIrreversible>` nêu số, QC tự đếm từ CSV (`p6-hand-calc.py --table`): **30 sinh viên · A+ 3, A 3, B+ 3, B 5, C+ 4, C 4, D+ 2, D 3, F 3 · 1 sinh viên liệt** (sv20; C `sv.nguyco` và sv26 là F do thang chữ, không liệt); khớp từng chữ cái; `Idempotency-Key` có trong request |
| TC-P6-08-37 | AC9 | sau chốt | UI | đầu bảng "Đã chốt v1 · {thời điểm}"; ô khoá; `Mở khoá` ở menu; **không** toast "Thành công"; dòng ghi chú cố định vẫn có (đúng 1) |
| TC-P6-08-38 | AC9 | UI | `finalize` khi 409 | hiển thị nguyên nhân + liên kết (thiếu điểm → danh sách); chữ tiếng Việt, không mã thô |
| TC-P6-08-39 | AC9 | repo | `playwright test gradebook.spec.ts -g 'finalize flow\|finalize incomplete list'` | xanh |
| TC-P6-08-40 | AC10 | sau chốt | `for T in "$TA_" "$SVA" "$ADM"; do code -X POST -H "$T" -H "$(idem)" $Q/gradebook/finalize; done`; giảng viên lớp khác | `403 403 403`; `404` |
| TC-P6-08-41 | AC10 | sinh viên B | `GET $Q/gradebook/snapshots/1` bằng `$SVB`; đoán `snapshot_id` của A | `403`/`404`; **không** dòng của A (IDOR); xem tc-US-P6-09 |
| TC-P6-08-42 | AC10 | repo | `GO ./internal/grade "TestFinalizeRBAC" -tags integration` | PASS |

## Nhánh lỗi (SRS mục 3)
Đã phủ: `SCHEME_NOT_CONFIRMED` (TC-06), `GRADES_INCOMPLETE` (TC-07, 08), `GRADEBOOK_LOCKED` (TC-09, 12), `NOT_LOCKED` (TC-18), `VALIDATION_FAILED` lý do (TC-17).
| TC-id | Tình huống | Bước | Kết quả mong đợi |
| --- | --- | --- | --- |
| TC-P6-08-43 | `finalize`/`unlock` thiếu `Idempotency-Key` | gọi không header | `400/422`; không đổi trạng thái |
| TC-P6-08-44 | Sinh viên rời lớp trước lần chốt lại | cho sv12 rời lớp rồi chốt lại v2 | không snapshot v2 cho sv12; snapshot v1 của sv12 còn nguyên; sv12 không nhận thông báo |
| TC-P6-08-45 | Chốt khi `entries` của ô EXAM chưa công bố | chuyển bài thi tuần 9 về chưa công bố → `finalize` | `409 GRADES_INCOMPLETE` (quên công bố ≠ cả lớp 0 — TLR-2) |
| TC-P6-08-46 | Ghi audit lỗi lúc chốt | `REVOKE INSERT ON audit_log` tạm (DB riêng) rồi `finalize` | fail-closed: không snapshot, sổ vẫn `OPEN` (cùng một giao dịch) |
| TC-P6-08-47 | Ba lần chốt–mở–chốt liên tiếp | v1 → v2 → v3 | `version` 1,2,3 không nhảy/không trùng; mỗi bản mang `unlock_reason` của lần mở trước nó; v1, v2 không đổi |

## Phân quyền
Phủ ở TC-19, TC-40, TC-41; thêm `p6-rbac.sh` đã gồm `finalize`/`unlock`.

## Script chạy được
`scripts/p6-put-race.sh` (QC pha 2), `p6-hand-calc.py`, `p6-env.sh`, `p6-go-tests.sh`.

## Câu hỏi cho BA / PM (không tự đoán)
- Q-08-a (TC-P6-08-36): ví dụ "2 sinh viên liệt" ở AC9 — theo CSV lớp 1 chỉ sv20 liệt (`LIET:final<3.0`); sv.nguyco = F nhưng **không liệt** (F do thang chữ). QC kỳ vọng "1 sinh viên liệt" và đếm chữ F riêng ("F 3": C, sv20, sv26). Nếu BA định nghĩa "liệt" gồm mọi F → báo.

## Lịch sử sửa TC
(chưa có)

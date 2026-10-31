# QC test case — US-P6-07 (nhập điểm XLSX: ánh xạ → xem trước → báo dòng lỗi → ghi; xuất XLSX)
Nguồn: `docs/specs/FEAT-gradebook/US.md` US-P6-07 (AC1–AC9) + `SRS.md` 3, 4.6, 6.2 (#10, #11, #17); TLR-3, TLR-7, TLR-14, TLR-16; `fixtures/inputs-lop1.csv`. Viết trước khi có code, không đọc code của dev (v1, 2026-10-31).

**Quy ước.** Tệp thử do QC sinh bằng `python3 docs/sprints/8/qc/scripts/p6-make-xlsx.py` → `/tmp/qc-p6-xlsx/` (`midterm.xlsx`, `final.xlsx`, `errors.xlsx`, `formula.xlsx`, `real-mssv.xlsx`, `rows5000.xlsx`, `toomany.xlsx`, `zipbomb.xlsx`, `corrupt.xlsx`, `notxlsx.xlsx`, `big3mb.xlsx`, `inject.xlsx`). Tải lên qua `POST $Q/uploads/presign {purpose:"grade_import", filename, mime_type, size_bytes, sha256}` rồi `PUT` lên URL ký (hàm `up()` trong `p6-env.sh`, QC thêm ở pha 2). `MIME`=`application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`. Ca ghi đổi điểm lớp 1: làm trên DB dựng lại; `ITEM_MID`, `ITEM_FIN` = id item `midterm`, `final` lấy từ `GET $Q/gradebook`. `GO` = `p6-go-tests.sh`.

Kỳ vọng `errors.xlsx` (mục tiêu `ITEM_MID`; `rows_total = 9` — dòng trống bỏ qua): dòng 2 `STUDENT_NOT_IN_COURSE` (99999999) · dòng 3 hợp lệ (B = 7) · dòng 4 `DUPLICATE_IN_FILE` (B lần 2) · dòng 5 `SCORE_OUT_OF_RANGE` (C = 11) · dòng 6 `SCORE_PRECISION` (sv04 = 8.333) · dòng 7 `SCORE_NOT_NUMBER` (sv05 = "abc") · dòng 8 trống → bỏ qua, không lỗi · dòng 9 hợp lệ (sv06 = "8,5") · dòng 10 `SCORE_OUT_OF_RANGE` (sv07 = −1) · dòng 11 điểm trống → không xoá điểm cũ, không lỗi.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P6-07-01 | AC1 | TEACHER | presign `midterm.xlsx` (đúng MIME, size, sha256 thật) → `PUT` lên URL ký | `200` có `upload_id`, URL ký; `PUT` thành công; **không** thấy tệp ở đĩa cục bộ gateway: `docker compose exec gateway sh -c 'find / -name "*.xlsx" -newer /proc/1 2>/dev/null \| head'` rỗng (luật 10) |
| TC-P6-07-02 | AC1 | như trên | presign `notxlsx.xlsx` đổi đuôi `.pdf`/`.csv`; hoặc đúng `.xlsx` nhưng `mime_type:"application/pdf"` | `422 FILE_TYPE_NOT_ALLOWED` |
| TC-P6-07-03 | AC1 | như trên | presign `big3mb.xlsx` (> 2 MiB, `size_bytes` thật) | `422 FILE_TOO_LARGE`; khai `size_bytes` nhỏ nhưng `PUT` tệp lớn → bị chặn ở bước đọc (`FILE_TOO_LARGE`/`FILE_INVALID`), không đọc cả tệp |
| TC-P6-07-04 | AC1 | như trên | 11 lần presign trong 1 phút | lần 11 `429` (≤ 10 / phút / người); người khác không bị ảnh hưởng |
| TC-P6-07-05 | AC1, AC6 | như trên | presign khai `sha256` sai; `PUT` tệp thật; `mode:"preview"` | `422 FILE_INVALID` (hoặc mã sha mismatch theo SRS); không ghi |
| TC-P6-07-06 | AC1 (phân quyền) | — | `for T in "$TA_" "$SVA" "$ADM"`: presign `grade_import`; `POST …/grade-entries/import` `mode:"preview"` và `mode:"commit"` | `403` ở cả 3 vai × 3 lệnh (9 phản hồi); không `upload` được tạo |
| TC-P6-07-07 | AC1 | repo | `GO ./internal/grade "TestImportPresignValidation|TestImportNoLocalDisk|TestImportRBAC" -tags integration` | PASS |
| TC-P6-07-08 | AC2 | `midterm.xlsx` đã lên | `POST $Q/grade-entries/import {upload_id,item_id:ITEM_MID,mode:"preview",mapping:{student_code_col:"A",score_col:"C",header_row:1}}` | `200 {rows_total:30, rows_ok:30, overwrites, unchanged, errors:[], sample:[≤20 {student,before,after}]}`; `sample.length ≤ 20`; **không** ghi: `count(grade_entries)` và `count(audit_log)` trước = sau |
| TC-P6-07-09 | AC2 | UI | `/gradebook` → `⋯` → `Nhập từ XLSX`; chọn tệp | bước 2 "Ánh xạ cột" **gợi ý** cột theo tên ("MSSV" → MSSV, "Điểm" → điểm); đổi được; bước 3 xem trước |
| TC-P6-07-10 | AC2 | repo | `GO ./internal/grade "TestImportPreviewNoWrite|TestImportMappingSuggest" -tags integration` | PASS |
| TC-P6-07-11 | AC3 | `errors.xlsx` → `ITEM_MID` | preview (mapping A, C, hàng 1) | `errors[]` **đúng** 6 phần tử, đúng `row` và `code` như bảng kỳ vọng ở đầu file; dòng 8 (trống) không có trong `errors`; dòng 11 không lỗi; `rows_ok` = 3 (dòng 3, 9, 11?) — QC ghi số thực, đối chiếu: dòng 11 điểm trống không phải lỗi và **không** xoá điểm cũ |
| TC-P6-07-12 | AC3 | `formula.xlsx` | preview | dòng B (`=22/3`, định dạng `0.00`): đọc giá trị **đã định dạng** `7.33` → hợp lệ; dòng C (`=22/3` định dạng chung → 7.333…): `SCORE_PRECISION` (TLR-7) |
| TC-P6-07-13 | AC3 | `real-mssv.xlsx` (ô số thực `20229002.0`) | preview | khớp sinh viên B (chuẩn hoá bỏ đuôi `.0`) |
| TC-P6-07-14 | AC3, TLR-3 | seed; sv `X` khai MSSV của bạn `Y` ở hồ sơ tài khoản (`users.student_code` = MSSV của Y, QC đặt bằng `PUT /me` hoặc SQL trên DB riêng) | tệp có dòng `Y_mssv, 9` → preview/commit | điểm vào **đúng sinh viên có `enrollments.student_code_snapshot` = MSSV đó** (Y), **không** vào X; X không nhận điểm của Y |
| TC-P6-07-15 | AC3 | hai sinh viên lớp có cùng `student_code_snapshot` (QC dựng trên DB riêng) | preview dòng MSSV đó | `AMBIGUOUS_STUDENT_CODE` |
| TC-P6-07-16 | AC3 | MSSV có số 0 đầu (QC thêm sinh viên `00123456` ở lớp thử) | dòng `00123456` kiểu chuỗi; dòng `123456` | khớp đúng người đầu; dòng sau `STUDENT_NOT_IN_COURSE` (không mất số 0) |
| TC-P6-07-17 | AC3 | repo | `GO ./internal/grade "TestImportParseErrors|TestImportDecimalComma|TestImportPrecisionRejected|TestImportFormattedCellValue|TestImportStudentCodeAsText|TestImportMatchesEnrollmentSnapshot|TestImportSelfDeclaredCodeIgnored|TestImportBlankKeepsOld"` | PASS 8 test |
| TC-P6-07-18 | AC4 | `midterm.xlsx`, DB dựng lại; điểm giữa kỳ đã có (seed) | preview rồi `commit` có `Idempotency-Key` | `200`; `grade_entries` item `midterm` = 30 dòng `source='IMPORT'`; những ô giá trị **khác** cũ: `version + 1`; ô giống cũ: không `version` mới; `audit_log`: một dòng mỗi điểm **đổi** (cũ → mới) **+ một dòng tổng của lô** |
| TC-P6-07-19 | AC4 | như trên | `commit` lần hai **cùng tệp** (khoá mới) | `unchanged = rows_ok` (30), 0 `version` mới, **0 audit mới** (đếm trước-sau) |
| TC-P6-07-20 | AC4 | như trên | `commit` lần hai **cùng `Idempotency-Key`** | cùng phản hồi lần đầu, không ghi lại |
| TC-P6-07-21 | AC4 | `errors.xlsx` | `commit` không `skip_errors` | `409 IMPORT_HAS_ERRORS`; không ghi dòng nào (kể cả dòng hợp lệ — tất cả hoặc không) |
| TC-P6-07-22 | AC4 | như trên | `commit` với `skip_errors:true` | ghi các dòng hợp lệ trong **một** giao dịch; dòng lỗi không ghi; UI nút `Bỏ qua N dòng lỗi và ghi M dòng` có số khớp |
| TC-P6-07-23 | AC4 | `ITEM_EXAM` (Kiểm tra tuần 9) làm đích | `commit`/`preview` | `409 ITEM_NOT_EDITABLE` |
| TC-P6-07-24 | AC4 | sổ `LOCKED` | `commit` | `409 GRADEBOOK_LOCKED` |
| TC-P6-07-25 | AC4 | `p6-put-race.sh` (QC) | `commit` và chốt (`finalize`) đồng thời | đúng một thắng; không snapshot nửa mới nửa cũ (xem TC-08) |
| TC-P6-07-26 | AC4 | repo | `GO ./internal/grade "TestImportCommitOneTx|TestImportIdempotentSameFile|TestImportHasErrorsBlocks|TestImportSkipErrors|TestImportAuditPerChange|TestImportLockedOrExam" -tags integration` | PASS |
| TC-P6-07-27 | AC5 | DB trống điểm giữa/cuối (xoá `grade_entries` của hai item hoặc dựng lại không nhập) | nhập `midterm.xlsx` vào `midterm`, `final.xlsx` vào `final` (commit); rồi `GET $Q/gradebook/preview?limit=100` | `select count(*) from grade_entries where item_id in (ITEM_MID, ITEM_FIN)` = **60**; từng giá trị khớp `inputs-lop1.csv` (QC so bằng script); preview 30/30 khớp `seed/expected_final_grades.csv` (QC: `p6-hand-calc.py` rc=0 + so `hp`/`letter`) |
| TC-P6-07-28 | AC5 | repo | `GO ./internal/grade "TestImportLop1MatchesExpected" -tags integration` | PASS |
| TC-P6-07-29 | AC6 | `zipbomb.xlsx` (~175 KiB nén, XML ~64 MiB) | preview; theo dõi RSS gateway (`docker stats --no-stream`) trước/sau | `422 FILE_INVALID` (vượt `UnzipXMLSizeLimit` 20 MiB); gateway không phình RAM (> 100 MiB tăng → FAIL), không OOM/restart (`docker ps` uptime) |
| TC-P6-07-30 | AC6 | `rows5000.xlsx` / `toomany.xlsx` | preview cả hai | 5.000 dòng: qua giới hạn (xử lý bình thường, phần lớn `STUDENT_NOT_IN_COURSE`); 5.001 dòng: `422 TOO_MANY_ROWS` (dừng ở dòng 5.001, không đọc hết) |
| TC-P6-07-31 | AC6 | `corrupt.xlsx`, `notxlsx.xlsx` (đúng MIME khai) | preview | `422 FILE_INVALID`; **không panic** (`docker compose logs gateway \| grep -ci panic` = 0); không `500` |
| TC-P6-07-32 | AC6 | `formula.xlsx`, `inject.xlsx` | preview; kiểm log | không thực thi ô công thức; tên tệp gốc / nội dung **không** có trong log ứng dụng (`docker compose logs gateway \| grep -c 'formula.xlsx\|20229002'` = 0) |
| TC-P6-07-33 | AC6 | tệp đã presign với `sha256` A, thay nội dung object storage bằng tệp B giữa preview và commit (QC dùng `mc`/`aws s3` tới MinIO) | `commit` | đọc lại từ object storage so `sha256` đã khai → từ chối (`FILE_INVALID`/sha mismatch); xem trước = ghi |
| TC-P6-07-34 | AC6 | repo | `GO ./internal/grade "TestImportZipBomb|TestImportTooManyRows|TestImportCorruptFile|TestImportSha256Mismatch|TestImportFormulaCellNotExecuted"` | PASS |
| TC-P6-07-35 | AC7 | lớp 1, sổ OPEN | `curl -D- -o /tmp/ex.xlsx -H "$TCH" $Q/gradebook/export.xlsx` | `200`; header `Cache-Control: no-store`, `Content-Disposition: attachment`; tệp mở được |
| TC-P6-07-36 | AC7 | `/tmp/ex.xlsx` | Đọc bằng `python3` (zipfile + XML; hoặc `unzip -p … xl/sharedStrings.xml`) | hàng đầu: tên lớp, mã lớp `761987`, thời điểm, **"Bản xem trước — chưa chốt"**; hàng tiêu đề cột đúng thứ tự `MSSV, Họ tên, <item…>, Điểm cộng, Điểm trừ, Điểm quá trình, Điểm thi cuối kỳ, Điểm học phần, Điểm chữ, Ghi chú`; 30 dòng sinh viên; dòng cuối **"Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường."**; A `9,5`/`A+`; sv20 Ghi chú "Liệt: …"; QC so `Điểm học phần` với CSV 30/30 |
| TC-P6-07-37 | AC7 | sổ `LOCKED` v1; sau đó sửa điểm nguồn (sửa điểm bài thi) mà chưa chốt lại | export | hàng đầu "Bản chốt v1"; **số lấy từ snapshot** (không theo điểm sống đã đổi) |
| TC-P6-07-38 | AC7 | tên sinh viên bắt đầu `=`, `+`, `-`, `@` (QC đổi tên 4 sinh viên lớp thử: `=1+1`, `+84`, `-2+3`, `@SUM(A1)`) | export; kiểm XML ô | các ô là **chuỗi** (`t="s"`/`inlineStr`), **không** có thẻ `<f>`; mở bằng thư viện không ra công thức |
| TC-P6-07-39 | AC7 | — | `$PSQL "select action, after from audit_log where action='export' order by created_at desc limit 1"` | có dòng `action='export'` với số dòng (30) mỗi lần xuất |
| TC-P6-07-40 | AC7 (phân quyền) | — | `for T in "$TA_" "$SVA" "$ADM"; do code -H "$T" $Q/gradebook/export.xlsx; done` | `403 403 403` |
| TC-P6-07-41 | AC7 | repo | `GO ./internal/grade "TestExportLayoutAndNote|TestExportFromSnapshotWhenLocked|TestExportFormulaInjection|TestExportHeadersAndAudit|TestExportRBAC" -tags integration` | PASS |
| TC-P6-07-42 | AC8 | `LOAD1000` | xuất; xem trước nhập 1.000 dòng; ghi nhập 1.000 dòng (đo `time_total`, 5 lần, lấy max) | xuất ≤ 3 s; xem trước ≤ 2 s; ghi ≤ 3 s; ghi 1.000 dòng và ghi 10 dòng: **cùng số câu SQL** (bật `log_statement=all` trong lúc chạy, đếm) |
| TC-P6-07-43 | AC8 | repo | `GO ./internal/grade "TestExport1000Under3s|TestImport1000Under3s|TestImportNoNPlusOne" -tags integration` | PASS |
| TC-P6-07-44 | AC9 | `/gradebook` | `Nhập từ XLSX`: bước 1 chọn tệp + item → bước 2 ánh xạ → bước 3 xem trước; nút chính theo bước | `Dialog` 3 bước, nút chính `Tiếp tục`, ..., `Ghi 30 dòng` (số khớp `rows_ok`); lỗi liệt kê theo hàng với mã tiếng Việt ("MSSV không có trong lớp", "Điểm ngoài thang 0–10", "Điểm có quá 2 chữ số thập phân", …) — không hiện mã tiếng Anh thô cho người dùng |
| TC-P6-07-45 | AC9 | như trên | tuỳ chọn `Công bố cột cho sinh viên` | **mặc định tắt**; bật + ghi → cột `published_at` có (khớp US-P6-09) |
| TC-P6-07-46 | AC9 | như trên | ngắt mạng khi tải lên | giữ tên tệp + `Tải lại`; không mất lựa chọn item/ánh xạ |
| TC-P6-07-47 | AC9 | như trên | ghi xong | ô trong bảng cập nhật không cần tải lại trang; đóng `Dialog` trả focus về nút menu |
| TC-P6-07-48 | AC9 | 375 px | mở `/gradebook`, vào `Nhập từ XLSX` | dùng được cho **xem**; không tràn ngang; (nhập cần desktop — chấp nhận thông báo ngắn) |
| TC-P6-07-49 | AC9 | repo | `playwright test gradebook.spec.ts -g 'import xlsx flow\|import errors listed'` | xanh |

## Nhánh lỗi (SRS mục 3)
Đã phủ: loại tệp / cỡ (TC-02, 03), dòng lỗi (TC-11, 12, 15, 16), `IMPORT_HAS_ERRORS` (TC-21), `ITEM_NOT_EDITABLE` (TC-23), `GRADEBOOK_LOCKED` (TC-24), tệp hỏng / zip bomb / quá dòng (TC-29…31), sha lệch (TC-33).
| TC-id | Tình huống | Bước | Kết quả mong đợi |
| --- | --- | --- | --- |
| TC-P6-07-50 | `upload_id` của lớp khác / không tồn tại | `import` lớp 1 bằng `upload_id` đã tải ở lớp 2 | `404`/`422`; không ghi |
| TC-P6-07-51 | `item_id` của lớp khác | `import` với item lớp 2 vào `$Q` | `404` |
| TC-P6-07-52 | `commit` thiếu `Idempotency-Key` | gọi không header | `400/422`; không ghi (luật 14) |
| TC-P6-07-53 | `mapping` sai cột (cột điểm = cột MSSV) | preview | mọi dòng `SCORE_*` lỗi, không `500` |
| TC-P6-07-54 | Hai `commit` cùng tệp song song, hai khoá khác nhau | `xargs -P2` | kết quả sau cùng bằng một lần commit; không `version` đôi; không deadlock (`5xx`) |

## Phân quyền
Phủ ở TC-06, TC-40; thêm: giảng viên lớp khác import → `404`.

## Script chạy được
`scripts/p6-make-xlsx.py`, `p6-env.sh`, `p6-go-tests.sh`; QC viết ở pha 2: `p6-import.sh` (presign + PUT + preview/commit), `p6-put-race.sh`.

## Câu hỏi cho BA / PM (không tự đoán)
- Q-07-a (TC-P6-07-11): `rows_ok` ở `errors.xlsx` khi dòng 11 (điểm trống): AC3 nói "điểm blank không xoá điểm cũ" nhưng không nói blank tính vào `rows_ok` hay `unchanged`. QC ghi số thực, không FAIL nếu nhất quán và `errors` đúng 6 dòng.

## Lịch sử sửa TC
(chưa có)

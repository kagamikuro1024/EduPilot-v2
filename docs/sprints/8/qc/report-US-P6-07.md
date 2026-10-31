# QC report — US-P6-07 (nhập điểm XLSX: ánh xạ → xem trước → báo dòng lỗi → ghi; xuất XLSX) · Kết luận vòng 1: **PASS** (theo proposal #25 cho đua nhập ↔ chốt)
Ngày 2026-10-31. Commit kiểm: `sprint/8-p6` @ `13646f87`. Bộ TC: `tc-US-P6-07.md` (TC-P6-07-01…59; sửa theo #11, #20). Handoff: `dev-US-P6-07.md`.

## 1. Cổng nghiệm thu đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `p6-go-tests.sh ./internal/grade "<36 test: mọi test trong TC-P6-07 + toàn bộ func Test trong import_test.go>" -race -tags integration` | PASS: `TestImportPresignValidation`, `NoLocalDisk`, `RBAC`, `ExportRBAC`, `PreviewNoWrite`, `MappingSuggest`, `ParseErrors`, `DecimalComma`, `PrecisionRejected`, `FormattedCellValue`, `StudentCodeAsText`, `MatchesEnrollmentSnapshot`, `SelfDeclaredCodeIgnored`, `CodeJoinedRowUnverified`, `RosterWinsOverCodeRow`, `BlankKeepsOld`, `CommitOneTx`, `IdempotentSameFile`, `HasErrorsBlocks`, `SkipErrors`, `AuditPerChange`, `LockedOrExam`, `Lop1MatchesExpected`, `ZipBomb`, `TooManyRows`, `CorruptFile`, `Sha256Mismatch`, `FormulaCellNotExecuted`, `ExportLayoutAndNote`, `ExportFromSnapshotWhenLocked`, `ExportFormulaInjection`, `ExportHeadersAndAudit`, `Import1000Under3s` (0,68 s), `Export1000Under3s` (0,59 s), `ImportNoNPlusOne`, `ImportMauFixture`; không thiếu, không SKIP, không DATA RACE; mọi tên test trong TC có thật |
| Đột biến (bản sao `/tmp/qc-mut`, đã xoá) | I1 tin dòng `joined_via='CODE'`: **đỏ** `TestImportParseErrors`, `…CodeJoinedRowUnverified`, `…RosterWinsOverCodeRow`; I2 bỏ `IMPORT_HAS_ERRORS`: **đỏ** `…HasErrorsBlocks`, `…CommitOneTx`; I3 bỏ `GRADEBOOK_LOCKED`: **đỏ** `…LockedOrExam`; I4 bỏ so `sha256`: **đỏ** `…Sha256Mismatch`; I5 bỏ `UnzipXMLSizeLimit`: **đỏ** `…ZipBomb`; I6 cho sửa cột `EXAM`: **đỏ** `…LockedOrExam`; I7 export để ô chữ `=…` thành công thức: **đỏ** `TestExportFormulaInjection`; I8 nới `TOO_MANY_ROWS` ×100: **đỏ** `…TooManyRows` — 8/8 đột biến bị bắt |
| `gradebook.spec.ts shell.spec.ts tokens.spec.ts` (`build:gate`, `--workers=2 --retries=0`) | 71 passed (desktop; mobile bỏ qua theo thiết kế), 0 failed — gồm `import xlsx flow`, `import errors listed`, `import upload failed`, `import publish option`, `export xlsx`, và `shell.spec` `regression-1.5` đã sang `/analytics` (#23: BUG-P6-06-1 đã được sửa) |
| `bash scripts/ui-antipatterns.sh` | rc=0 (có `ImportDialog` trong danh sách Dialog, #24) |
| QC dump (`scripts/qc-p6-07-ui.spec.ts.txt` → `shots/p607-*`) | menu TEACHER `Nhập từ XLSX \| Xuất XLSX`; TA không có menu (`p606-ta-1440.txt`); Dialog 3 bước; 1440 và 375 không tràn ngang (`scroll 1440/1440`, `375/375`); `Công bố cột cho sinh viên` mặc định tắt; lỗi theo hàng bằng chữ ("Dòng 3 · 99999999 · MSSV không có trong lớp", "Dòng 6 · 20229777 · MSSV chưa có trong danh sách lớp"); nút chính `Bỏ qua 2 dòng lỗi và ghi 1 dòng` |
| `go.mod` | `excelize` có trong bảng thư viện (`ARCHITECTURE.md`); không thư viện mới |

## 2. Kết quả theo AC
| AC | Kết quả | Bằng chứng |
| --- | --- | --- |
| AC1 presign `grade_import`, `.xlsx` ≤ 2 MiB, URL ký, không đĩa cục bộ, 10 / phút, TEACHER | PASS | `TestImportPresignValidation`, `NoLocalDisk`, `RBAC` (TA / STUDENT / ADMIN `403` ở presign + preview + commit) |
| AC2 xem trước không ghi, gợi ý ánh xạ, mẫu ≤ 20 | PASS | `TestImportPreviewNoWrite`, `MappingSuggest`; dump bước 2 |
| AC3 dòng lỗi theo hàng, giá trị đã định dạng, MSSV theo `ROSTER`/`ADMIN` (#20), `STUDENT_CODE_UNVERIFIED` | PASS | `ParseErrors`, `DecimalComma`, `PrecisionRejected`, `FormattedCellValue`, `StudentCodeAsText`, `MatchesEnrollmentSnapshot`, `SelfDeclaredCodeIgnored`, `CodeJoinedRowUnverified`, `RosterWinsOverCodeRow`; I1 làm đỏ; câu hiển thị "MSSV chưa có trong danh sách lớp" |
| AC4 ghi một giao dịch, khoá `FOR SHARE`, một câu lệnh theo lô, `version`, audit từng điểm đổi, nhập lại cùng tệp không audit | PASS (đua nhập ↔ chốt theo #25 → US-P6-08) | `CommitOneTx`, `IdempotentSameFile`, `HasErrorsBlocks`, `SkipErrors`, `AuditPerChange`, `LockedOrExam`; I2, I3, I6 làm đỏ |
| AC5 nhập đủ giữa / cuối kỳ → 30/30 khớp CSV độc lập | PASS | `TestImportLop1MatchesExpected` (đáp án BA, QC đã kiểm chéo 30/30 bằng `p6-hand-calc.py` ở sprint này) |
| AC6 an toàn tệp: zip bomb, ≤ 5.000 dòng, hỏng, công thức không chạy, `sha256` | PASS | `ZipBomb`, `TooManyRows`, `CorruptFile`, `Sha256Mismatch`, `FormulaCellNotExecuted`; I4, I5, I8 làm đỏ |
| AC7 xuất XLSX: bố cục, từ snapshot khi chốt, chống chèn công thức, audit, `no-store`, chỉ TEACHER | PASS | `ExportLayoutAndNote`, `FromSnapshotWhenLocked`, `FormulaInjection`, `HeadersAndAudit`, `ExportRBAC`; I7 làm đỏ |
| AC8 1.000 dòng: xuất ≤ 3 s, xem trước ≤ 2 s, ghi ≤ 3 s, không N+1 | PASS ở mức test (đo trên stack 1.000 sinh viên → US-S8-SEED) | `Export1000Under3s`, `Import1000Under3s`, `ImportNoNPlusOne` |
| AC9 `Dialog` ba bước, nút chính theo bước, lỗi liệt kê, công bố mặc định tắt, tải hỏng giữ tên tệp, 375 px xem | PASS | `import xlsx flow`, `import errors listed`, `import upload failed`, `import publish option`; dump bước 1–3; xem ảnh `p607-step3-1440.png` |

## 3. Lỗi / ghi nhận
- **BUG-P6-07-1 (Thấp, không thuộc AC):** đóng `Dialog` nhập (Esc hoặc `Đóng`) làm focus rơi về `<body>` thay vì nút `Thêm hành động` (QC đo `document.activeElement` = `BODY`; Dialog gốc `<dialog>` trả focus về phần tử đang focus lúc mở — ở đây là mục menu đã bị gỡ khỏi DOM). Người dùng bàn phím mất vị trí. US AC9 không đòi; đề nghị dev trả focus về nút menu (thấp).
- Dev chưa có test đua `commit` nhập ↔ chốt → chuyển sang US-P6-08 (#25, QC mở).
- Ghi chú dữ liệu giả: dòng tóm tắt "1 dòng hợp lệ · 1 ghi đè" nhưng mẫu liệt kê 3 dòng (do API giả của `gradebook.spec`, không phải lỗi mã thật).

## 4. CI
Run `41202190312` (`13646f87`): **success**, mọi job xanh (Go ×4, Judge, Lighthouse, Playwright 1/3, 2/3, 3/3). Verdict **PASS**.

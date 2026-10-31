# QC report — US-P7-04 (`UploadSource`, bộ khớp sinh viên, tab `Chưa khớp`) · Kết luận vòng 1: **FAIL** (chỉ AC4: hộp thoại gán thiếu câu "Sinh viên này đã có {n} phiên bản"; 8/9 AC còn lại PASS)
Ngày 2026-11-01. Commit kiểm: `sprint/9-p7` @ `c487e4fa` (mã US-P7-04 không đổi từ `f5bbfa57`). Bộ TC: `tc-US-P7-04.md`. Handoff: `dev-US-P7-04.md`. CI `c487e4fa`: **success** (`6f9f219a` và `acd5c294` đỏ do hai test cũ + ba test `activity` chập chờn; QC chạy lại cả năm test ở `c487e4fa`: xanh).

## 1. Cổng đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `p7-go-tests.sh` theo gói, `-race -tags integration` (httpapi `testroutes`), mọi tên của dòng `Kiểm:` | PASS 39/39: submission 31, grade 3, httpapi 3, today 2 |
| Hộp đen trên stack seed thật (`scripts/p7-api-04.py`) | PASS 0 lỗi. Tệp lẻ md → MATCHED đúng sv.gioi; `.exe` / rỗng bị từ chối; không mã / MSSV ngoài lớp → UNMATCHED. **TLR-1:** `sv.lech` tự khai MSSV `20229002` (= B); tệp `BT03_20229002.md` chỉ gán B theo danh sách lớp, 0 hàng của `sv.lech`; `grep users.student_code` ở `internal/submission` rỗng. ZIP 10 mục: 2 tệp hợp lệ tạo bản, thư mục / `__MACOSX` / `.DS_Store` bỏ im, `..` + `/abs` + zip lồng → `ENTRY_UNSAFE`, `x.exe` → `ENTRY_TYPE_NOT_ALLOWED`, 0 byte → `FILE_EMPTY`, không có tệp ngoài kho. Tải lại cùng ZIP → `deduped`, không bản mới; ZIP sửa một tệp → chỉ tệp đó lên phiên bản kế (sv.nguyco 1,2). ZIP bom (305 KB nén → 300 MiB) bị từ chối, không bản ghi. ZIP 201 mục: 200 mục đầu được nhập rồi `ZIP_LIMIT` (SRS 4.4.3 "dừng và báo"). Bài chưa công bố → 409 `ASSIGNMENT_NOT_PUBLISHED`; kênh `UPLOAD` tắt → 409 `CHANNEL_NOT_ALLOWED`. Sinh viên đã REVIEWED nhận tệp mới → `UNMATCHED` `ALREADY_REVIEWED` + ứng viên, điểm không đổi. STUDENT / ADMIN: presign, upload, danh sách, tải tệp đều 403; TA, TEACHER 200. Gán tay: sinh viên chưa vào lớp 422, sai `version` 409, B đã REVIEWED 409 `ALREADY_REVIEWED`, TA gán bài chưa công bố điểm → 200, `version = max + 1`, `MANUAL`, `audit_log`; gán lần hai cùng `version` 409. `Bỏ qua`: lý do 0 / 201 ký tự → 422; hợp lệ → bản ghi + tệp còn, biến khỏi danh sách; `DELETE` hoàn tác trả lại. "Hôm nay": TEACHER và TA đúng một mục `UNMATCHED_SUBMISSION`; STUDENT / ADMIN không có; hết chưa khớp → mục biến mất ≤ 6 s |
| Playwright desktop `unmatched assign`, `unmatched dismiss` (+ `me.spec.ts assignments section` của US-P7-03 AC10) | 3/3 xanh |
| Ảnh tab `Chưa khớp` thật 1440 (TEACHER, TA) | `shots/p704-unmatched-teacher-1440.png`, `…-ta-1440.png`; `scrollWidth = 1440`, không nhãn "Bản mô phỏng" (BUG-S9-01 đã sửa) |
| Đột biến (`/tmp/qc-mut9`, đã xoá) N1 MSSV tự khai thành tín hiệu khớp · N2 bỏ kiểm `..` ở tên mục ZIP · N3 bỏ `REVIEWED` khỏi `ALREADY_REVIEWED` | cả 3 **đỏ**: `TestMatchTable` + `TestMatchCodeJoinedRowNotVerified` + `TestSelfDeclaredCodeNeverMatches`; `TestZipSlipRejected`; `TestUploadAlreadyReviewed` + `TestIngestAlreadyReviewedAllSources` |
| `ui-antipatterns.sh`; grep `mock/assess` | rc 0; rỗng |

## 2. Kết quả theo AC
| AC | Kết quả | Ghi chú |
| --- | --- | --- |
| AC1 bộ khớp xác định (≥ 20 ca) | PASS | `TestMatchTable`, N1; hộp đen UNMATCHED / MATCHED |
| AC2 MSSV tự khai không mở dữ liệu (TLR-1) | PASS | `TestSelfDeclaredCodeNeverMatches`, `TestCodeJoinedXWithYCodeNeverMatchesX`; kịch bản `sv.lech` ↔ B |
| AC3 tệp lẻ / ZIP an toàn | PASS | N2; bom; zip-slip; nested; `ENTRY_*` |
| AC4 tab `Chưa khớp` — gán tay, có dấu vết | **FAIL** | Mọi hành vi PASS (API, audit, `version`, 409 / 422, UI). **Thiếu** câu hộp thoại gán nêu "Sinh viên này đã có {n} phiên bản" mà US ghi trong AC; dev ghi "API chưa trả `n`, cần BA quyết" (không có proposal). Chuyển PASS khi dev bổ sung hoặc BA bỏ câu khỏi AC (proposal) |
| AC5 idempotent theo `sha256 + tên mục` | PASS | hộp đen; `TestUploadIdempotentBySha`, `TestReuploadAfterManualMatch` |
| AC6 bài / tệp không hợp lệ, bài đã duyệt | PASS | N3; hộp đen. TLR-10 phần IMAP / Teams: kiểm ở US-P7-05 / 11 |
| AC7 phân quyền | PASS | hộp đen + `TestUploadRBAC`, `TestUnmatchedListRBAC`, `TestSignedGetChecksRole`; ca "TA gán bài đã công bố điểm → 403" chỉ kiểm bằng Go (chưa có route `publish` để dựng ở stack) |
| AC8 nguồn việc "Hôm nay" | PASS | hộp đen + `TestProviderUnmatchedSubmission`, `…Cleared` |
| AC9 `Bỏ qua`, không bao giờ thành 0 điểm | PASS | hộp đen (dismiss / hoàn tác); `TestUnmatchedBlocksZeroScore`, `TestNoSubmissionZeroOnlyWhenNoUnmatched`, `TestFinalizeIncompleteWhenUnmatched` xanh. Hộp đen sổ điểm (TC-17) chưa dựng được: item `ASSIGNMENT` ở công thức là US-P7-09 |

## 3. Lỗi / ghi nhận
- **BUG-P7-04-1 (Trung bình, AC4):** thiếu câu "Sinh viên này đã có {n} phiên bản" ở hộp thoại gán (xem trên).
- **BUG-P7-04-2 (Thấp, hợp đồng):** `openapi.yaml` `PresignInput.purpose` có `enum: [document, grade_import]` nhưng route nhận `submission_upload` (mô tả của chính `uploadSubmissions` và SRS 4.4.3 nêu). Máy khách sinh từ spec sẽ bị từ chối; thêm `submission_upload` vào enum.
- Ghi nhận: ZIP 201 mục đã nhập 200 mục đầu rồi mới báo `ZIP_LIMIT` — đúng SRS ("dừng và báo"), TC-07 viết "từ chối" → hiểu là dừng ở mục thứ 201.
- Ghi nhận: `GET /jobs/{id}` của job `submission.zip` chỉ người tạo đọc được (TEACHER không đọc job của TA) — đúng `jobs.Scope` hiện tại, SRS chưa nói rõ.
- Handoff nêu `-tags integration` không dùng; QC dùng `integration` (không ảnh hưởng kết quả). Test cũ dev sửa: `today.Topics()` 27 → 28, `TestTodayQueryBudget` (#9 ACCEPTED), `grade/me_test.go noStamps()` (chặt hơn).

# QC report — BUG-S7-01 (#19, #23, #26: test Go chờ đồng hồ thật; `Go test` đỏ ngẫu nhiên; thời gian CI) — **vòng 2**
Ngày 2026-10-14. Commit kiểm: `sprint/7-p4-p5` @ `2ad5764` (sửa vòng 2: `0ea7be6`, `1e71299`; gồm #26 `TestSchemaAttendance`, tách `rest-1` / `rest-2`). Handoff: `dev-BUG-S7-01.md` mục "Vòng 2". Cách đo như vòng 1: 6 nhánh tạm `tmp-qc5-a…f` (cùng cây `2ad5764` + một tệp `.github/qc-probe.txt` để `paths-ignore: docs/**` không bỏ qua), đẩy 3 + 3 lượt, runner nguội, đủ mọi job. Ngưỡng PM: ≤ 10 phút mỗi run.

## Kết luận: **PASS** (6 / 6 xanh; mọi run ≤ 10 phút)
`TestSchemaAttendance` không còn đỏ; `Go test (rest-1)` + `(rest-2)` thay `rest`; dòng dài nhất 444 s; run dài nhất 593 s (9 m 53 s).

## 1. Đo 6 lượt nguội
| Lượt | Run | Kết quả | `rest-1` | `rest-2` | `course` | `auth` | Toàn run |
| --- | --- | --- | --- | --- | --- | --- | --- |
| a | `38474751763` | success | 262 s | 347 s | 417 s | 395 s | 9 m 20 s |
| b | `38474759834` | success | 257 s | 413 s | 352 s | 346 s | 9 m 06 s |
| c | `38474764957` | success | 248 s | 358 s | 426 s | 407 s | 8 m 55 s |
| d | `38475972735` | success | 369 s | 428 s | 444 s | 406 s | 8 m 06 s |
| e | `38475978002` | success | 365 s | 409 s | 430 s | 286 s | 9 m 10 s |
| f | `38475981716` | success | 341 s | 426 s | 421 s | 407 s | 9 m 53 s |

(Cột "Toàn run" lấy từ `createdAt` đến `updatedAt` của run, gồm thời gian xếp hàng runner.) Không còn lượt đỏ ở bất kỳ job nào; `TestAuditBatches`, nhóm R1–R4, `TestFromDraftNoMessageStored` (#23), `TestSchemaAttendance` (#26) đều xanh cả 6 lượt.

## 2. Hệ quả với US-CI-02
Tiêu chí đo của US-CI-02 (TC-20 ≤ 10 phút; không đỏ ngẫu nhiên ở lượt nguội) nay đạt ở `2ad5764`. Lần FAIL của US-CI-02 là FAIL lịch sử (vòng 3: 1/6 đỏ do `TestAuditBatches`); BUG-S7-01 đã đóng phần đo.

## 3. Ghi nhận ngoài phạm vi (không chấm)
- Run của nhánh chính `sprint/7-p4-p5` ở `f45454c` đỏ ở `Frontend Lighthouse (simulate)` (`resource-summary.script.size … maxNumericValue`); run ở `1d841b1` đỏ ở `Judge attacks` và Lighthouse. Đây không phải job Go; báo PM theo dõi.
- Nếu có "required check" tên `Go test (rest)` thì PM đổi sang `Go test (rest-1)` + `Go test (rest-2)` (dev đã nêu).

## 4. Phụ lục: vòng 1 (đã đóng)
### Kết luận: **FAIL** (5 / 6 xanh; 1 lượt đỏ ở `Go test (rest)`; ngoài ra thời gian vượt 10 phút ở 4 / 6 lượt)
Các test thời gian đã sửa **không còn đỏ**: `TestAuditBatches` và nhóm R1–R4 xanh cả 6 lượt; `TestFromDraftNoMessageStored` xanh cả 6 lượt. Lượt đỏ là **một test khác, không phải test chờ đồng hồ**: `TestSchemaAttendance` (mới từ US-P5-01). Theo ngưỡng PM "xanh 6/6", chưa đạt; US-CI-02 chưa thể ghi "chỉ còn FAIL lịch sử".

### 1. Đo 6 lượt nguội
| Lượt | Run | Kết quả | `Go test (rest)` | Toàn run |
| --- | --- | --- | --- | --- |
| a | `38459332203` | success | 627 s | 10 m 45 s |
| b | `38459330338` | success | 649 s | 11 m 03 s |
| c | `38459334626` | success | ≈ 440 s | 8 m 39 s |
| d | `38461148462` | success | 630 s | 10 m 49 s |
| e | `38461153164` | success | 615 s | 10 m 28 s |
| f | `38461157946` | **failure** (`Go test (rest)`) | 494 s | 9 m 50 s |

Lượt a–c đẩy cùng lúc, d–f đẩy cùng lúc (như vòng 3 của US-CI-02, khi toàn run chỉ 7 m 43 s … 8 m 44 s).

### 2. Lỗi
**BUG-1 (TRUNG BÌNH): `TestSchemaAttendance` đỏ ngẫu nhiên (1/6).** Log run `38461157946`, job `Go test (rest)`:
```
Error Trace: backend-go/internal/store/chat_schema_test.go:63
             backend-go/internal/store/crm_schema_test.go:64
Error:       Not equal: expected: "23503"   actual  : "23505"
Messages:    người điểm lạ
--- FAIL: TestSchemaAttendance (2.62s)
```
Ca "người điểm lạ" (sinh viên không ghi danh) mong khoá ngoại `23503` nhưng nhận `23505` (trùng `UNIQUE (course_id, session_id, student_id)`): Postgres kiểm unique trước khoá ngoại, nên kết quả phụ thuộc việc dữ liệu thử đã có dòng cùng bộ ba hay chưa. Không tái hiện ở 5 lượt khác. Không liên quan đồng hồ; test do US-P5-01 thêm sau khi dev báo "đo 6/6". Đề xuất: bảo đảm dữ liệu của ca này không trùng bộ ba đã chèn ở ca trước.
**BUG-2 (THẤP–TRUNG BÌNH): thời gian `Go test (rest)` tăng.** 5 / 6 lượt 494–649 s (vòng 3: 6–8 phút); toàn run 4 / 6 lượt > 10 phút (TC-20 của US-CI-02 đòi ≤ 10 phút, `ci.yml` chú thích "dòng `rest` 6–8 phút"). Có thể do nhiều test mới (US-P5-01/02) hoặc hạn chống treo 30–60 s; QC chưa tách nguyên nhân.

### 3. Dọn
Nhánh tạm `tmp-qc4-a…f` xoá khi nộp báo cáo.

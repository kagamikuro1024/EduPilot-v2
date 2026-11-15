# Lỗi QC — sprint 10.8 (FEAT-attendance-selfcheck)

Đợt 1: ep-qc @`a1c7ab30` (L1–L5). Tái hiện bằng `scripts/as_b2.py`, `scripts/as_d3.py` (đã dán dưới).

| Mã | Mức | TC | Mô tả | Trạng thái |
| --- | --- | --- | --- | --- |
| **D-AS-01** | **Cao** | TC-AS04-07, TC-AS04-08 (US-AS-04 AC4, AC9; US-AS-03 AC6) | Hai SV **cùng lớp** điểm danh bằng **cùng khoá thiết bị** (cùng `WEB_KEY` hoặc cùng `ANDROID_KEY`): SV thứ hai nhận **`503 SERVICE_UNAVAILABLE`** (`retry_after` 5), **không** có lượt nào được ghi (giao dịch rollback). Kỳ vọng `201` + `DEVICE_SHARED` ở cả hai lượt. Hệ quả: tín hiệu chống gian lận chính (một điện thoại điểm danh hộ nhiều người) **không bao giờ xuất hiện** và SV thật dùng chung máy (điện thoại gia đình, máy trạm phòng lab) **không điểm danh được**. Tái hiện 4/4 (cả đua 4 vòng). Log gateway chỉ có dòng `request … status 503`, không dòng lỗi | Mở |
| D-AS-02 | Thấp | TC-AS03-02b (US-AS-03 AC1) | Thân `POST …/checkins` có trường lạ (`student_id`, `user_id`, bất kỳ) → `422 VALIDATION_FAILED` `details[0].code=unknown`; `openapi.yaml` + SRS nói «bị bỏ qua». An toàn (chặt hơn), nhưng lệch hợp đồng. Chọn một: sửa `openapi.yaml`/US (khuyến nghị, đúng quy ước repo) hoặc code | Mở |
| D-AS-03 | Thấp (tài liệu) | TC-AS12-03c | `openapi.yaml` `DeviceRegisterInput.attestation_chain` `minItems: 1, maxItems: 10`; thực tế `422 VALIDATION_FAILED` «Cần 2 đến 6 chứng chỉ và khoá công khai.» | Mở |

## D-AS-01 — nguyên nhân (đọc mã sau khi tái hiện)
`AttFlagShared` (`internal/store/attendance_checkin.sql.go:471`, `StudentIds []uuid.UUID`) gọi từ `attendance/checkin.go:225` khi có `DEVICE_SHARED`. Gateway chạy qua **PgBouncer (`QueryExecModeExec`)**, nơi `[]uuid.UUID` không mã hoá được (repo đã ghi ở `llmconfig/service.go:633`, `integration/no_payload_leak_test.go:61` «QC BUG-1 US-P3-05»). Lỗi mã hoá không phải `*pgconn.PgError` ⇒ `dbErr` đổi thành `503`. `go test` dùng pool trực tiếp nên không bắt được. **Gợi ý sửa:** truyền `[]string` + `::uuid[]` (hoặc `text[]`), thêm ca `TestCheckinFlags`/`TestDeviceSharedRace` chạy bằng `testutil.RuntimePool(t)`. Cùng loại cần rà: `AttSharedDeviceStudents` trả `[]uuid.UUID` (đọc, an toàn), `CourseIds []uuid.UUID` (`attendance_checkin.sql.go:598`).

## Tái hiện nhanh
```
python3 scripts/as_b2.py   # TC-AS04-07/08 nằm trong as_b.py; hoặc thủ công:
K=Key(); checkin("sv.gioi", C, {token:T, device:K.dev("sv.gioi",token=T)}) -> 201
checkin("sv.kha",  C, {token:T, device:K.dev("sv.kha", token=T)})  -> 503
```

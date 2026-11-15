# Lỗi QC — sprint 10.8 (FEAT-attendance-selfcheck)

Đợt 1: ep-qc @`a1c7ab30` (L1–L5); kiểm lại đợt 2 @`b54d020e` (gồm sửa của dev2). Tái hiện bằng `scripts/as_b2.py`, `scripts/as_d3.py` (đã dán dưới).

| Mã | Mức | TC | Mô tả | Trạng thái |
| --- | --- | --- | --- | --- |
| **D-AS-01** | **Cao** | TC-AS04-07, TC-AS04-08 (US-AS-04 AC4, AC9; US-AS-03 AC6) | Hai SV **cùng lớp** điểm danh bằng **cùng khoá thiết bị** (cùng `WEB_KEY` hoặc cùng `ANDROID_KEY`): SV thứ hai nhận **`503 SERVICE_UNAVAILABLE`** (`retry_after` 5), **không** có lượt nào được ghi (giao dịch rollback). Kỳ vọng `201` + `DEVICE_SHARED` ở cả hai lượt. Hệ quả: tín hiệu chống gian lận chính (một điện thoại điểm danh hộ nhiều người) **không bao giờ xuất hiện** và SV thật dùng chung máy (điện thoại gia đình, máy trạm phòng lab) **không điểm danh được**. Tái hiện 4/4 (cả đua 4 vòng). Log gateway chỉ có dòng `request … status 503`, không dòng lỗi | **Đóng** — kiểm lại @`b54d020e`: 2–3 SV cùng khoá đều 201 + `DEVICE_SHARED` (4/4 vòng đua); app emulator: hai tài khoản cùng điện thoại đều «Đã điểm danh» (`scripts/as_h.py`) |
| D-AS-02 | Thấp | TC-AS03-02b (US-AS-03 AC1) | Thân `POST …/checkins` có trường lạ (`student_id`, `user_id`, bất kỳ) → `422 VALIDATION_FAILED` `details[0].code=unknown`; `openapi.yaml` + SRS nói «bị bỏ qua». An toàn (chặt hơn), nhưng lệch hợp đồng. Chọn một: sửa `openapi.yaml`/US (khuyến nghị, đúng quy ước repo) hoặc code | **Đóng** — kiểm lại @`b54d020e`: hai và ba SV cùng khoá đều 201 + `DEVICE_SHARED` (4/4 vòng đua); trong app emulator hai tài khoản cùng điện thoại đều «Đã điểm danh» (`scripts/as_h.py`) || **Đóng** — BA vá spec (P-108-13, SRS v1.6): 422 `unknown`; kiểm lại: 422, không ghi lượt |
| **D-AS-06** | **Trung bình** | TC-AS01-10a (US-AS-01 AC5) | Tắt `self_checkin` khi cửa sổ đang mở: US nói «cửa sổ đang mở vẫn nhận lượt tới khi đóng», thực tế `POST …/checkins` (token và mã ngắn) → **`409 SELF_CHECKIN_DISABLED`**; SV đang quét giữa buổi bị chặn dù màn chiếu (`display` 200) vẫn hiện mã. `rotate_seconds` đã ghim thì đúng (5 s). Sửa: bước «lớp bật self_checkin» (SRS FR-10) chỉ áp khi KHÔNG có cửa sổ mở / dùng chính sách ghim lúc mở cửa sổ (`checkin_windows` đã có `rotate_seconds`; thêm cột hoặc coi «có cửa sổ mở» là đủ) | Mở |
| D-AS-05 | Thấp | TC-AS08-UX1 (`/attendance` 375 px) | Thông báo «Còn N sinh viên chưa điểm danh» xuống dòng từng chữ (nút «Đánh vắng N SV chưa điểm danh» chiếm bề rộng của `InlineNotice`); `shots/att-confirm2-375.png` | Mở |
| D-AS-04 | Thấp | android-checklist AC-0 (US-AS-11 AC1, AC8) | `mobile/README.md` và `scripts/prepare.mjs` cho phép `APP_SERVER_URL` = `http://localhost`, `127.0.0.1`, `10.0.2.2` «khi thử», nhưng `capacitor.config.js` đặt `cleartext: false`: APK dựng với `http://localhost:580` mở ra trang «Không kết nối được máy chủ» (đo trên emulator; đổi `cleartext: true` thì tải được) | **Đóng** — dev sửa @`089b9d01`: `prepare.mjs` chỉ nhận HTTPS (gợi ý tunnel), README đã sửa |
| D-AS-03 | Thấp (tài liệu) | TC-AS12-03c | `openapi.yaml` `DeviceRegisterInput.attestation_chain` `minItems: 1, maxItems: 10`; thực tế `422 VALIDATION_FAILED` «Cần 2 đến 6 chứng chỉ và khoá công khai.» | **Đóng** — openapi `minItems: 2, maxItems: 6`; 1 và 7 chứng chỉ → 422 |

## D-AS-01 — nguyên nhân (đọc mã sau khi tái hiện)
`AttFlagShared` (`internal/store/attendance_checkin.sql.go:471`, `StudentIds []uuid.UUID`) gọi từ `attendance/checkin.go:225` khi có `DEVICE_SHARED`. Gateway chạy qua **PgBouncer (`QueryExecModeExec`)**, nơi `[]uuid.UUID` không mã hoá được (repo đã ghi ở `llmconfig/service.go:633`, `integration/no_payload_leak_test.go:61` «QC BUG-1 US-P3-05»). Lỗi mã hoá không phải `*pgconn.PgError` ⇒ `dbErr` đổi thành `503`. `go test` dùng pool trực tiếp nên không bắt được. **Gợi ý sửa:** truyền `[]string` + `::uuid[]` (hoặc `text[]`), thêm ca `TestCheckinFlags`/`TestDeviceSharedRace` chạy bằng `testutil.RuntimePool(t)`. Cùng loại cần rà: `AttSharedDeviceStudents` trả `[]uuid.UUID` (đọc, an toàn), `CourseIds []uuid.UUID` (`attendance_checkin.sql.go:598`).

## Tái hiện nhanh
```
python3 scripts/as_b2.py   # TC-AS04-07/08 nằm trong as_b.py; hoặc thủ công:
K=Key(); checkin("sv.gioi", C, {token:T, device:K.dev("sv.gioi",token=T)}) -> 201
checkin("sv.kha",  C, {token:T, device:K.dev("sv.kha", token=T)})  -> 503
```

# DEV handoff — US-P3-02
Nhánh: `sprint/6-p3-p8`. Commit: `US-P3-02: …`.

## Đã làm
Gói `backend-go/internal/privacy` (không có service / handler; story sau cắm vào):
- `detect.go`: `Detector.Detect` (regex RE2: MSSV 3 dạng + cụm "MSSV…", email, SĐT VN 03/05/07/08/09, CCCD 12 số liền / nhóm 3-3-3-3), bỏ chồng lấn, chỉ số **rune**; văn bản > 20.000 rune quét theo cửa sổ chồng 256 rune.
- `roster.go`: từ điển theo lớp. Cache Redis `ep:roster:{course}` TTL 1 giờ; biến thể tên: nguyên, không dấu / hoa-thường (qua `auth.Fold`), đảo (`An Nguyễn Văn`, `An Nguyễn`), rút (`Nguyễn An`); khớp theo từ + chỉ khoảng trắng giữa các từ; tên 1 âm tiết không vào từ điển. `Roster.Invalidate` là `outbox.Handler`.
- `store.go` + query `PrivacyRoster` (`privacy.sql`): chỉ `STUDENT` + `ACTIVE`.
- `redact.go`: `Redact` → `[đã ẩn]`, gộp khoảng chạm / chồng, cùng chuỗi khi sạch.
- `mask.go`: `Masker.Mask / Unmask`, `Sess` (`NewSession(id)`; `""` = phạm vi yêu cầu), `WithSession` / `SessionFrom` cho hook của US-P3-03. Ánh xạ Redis `HASH ep:mask:{sid}` (`p:` / `r:` / `n:`) qua **một** Lua script mỗi lần `Mask` (cấp số nguyên tử, EXPIRE 24 h mỗi lượt). Redis lỗi / chậm > 15 ms → ánh xạ trong bộ nhớ + `warn` (không nội dung). Panic / roster không nạp được / quá 50 ms → `ErrMaskFailed` (`MASK_FAILED`).
- `stream.go`: `StreamUnmasker` máy trạng thái theo tiền tố, giữ ≤ 32 rune, chỉ giữ khi gặp `[`. `Unmask(toàn bộ)` dùng **cùng** máy này nên bằng nối các `Write` + `Flush` theo cấu trúc. Placeholder không có trong ánh xạ / dạng hỏng (`[[SV]]`, `[[SV_`) → `bạn` + MỘT dòng `warn` chỉ có `count`.
- `cmd/worker/registry.go`: `course.member_changed` và `roster.imported` giờ là `outbox.Chain(inv.Handle, roster.Invalidate)` (tách khỏi vòng `inv.Handle` cũ).

## File đổi
`backend-go/internal/privacy/*` (mới), `backend-go/internal/store/queries/privacy.sql` + `privacy.sql.go` (query `PrivacyRoster`), `backend-go/cmd/worker/registry.go`, `backend-go/cmd/worker/roster_test.go`.

## Lệnh QC chạy để kiểm
```bash
cd backend-go
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
go test -count=1 -race ./internal/privacy -v
go test -count=1 ./internal/privacy -run 'TestUnmaskStream$' -v        # log "số tổ hợp đã chạy: 72470"
go test ./internal/privacy -bench BenchmarkMask4k -run '^$' -benchtime=200x   # ~1,3 ms/op
go test -count=1 ./internal/privacy -run TestDetectLinearTime -v
go test -count=1 ./cmd/worker -run TestRosterInvalidateRegistered -v
$RDS TTL ep:mask:$SID   # sau một lượt Mask có phiên: số dương ≤ 86400
```

## Test đã chạy và kết quả
`go test -race ./internal/privacy ./cmd/worker`: xanh. `go vet`, `golangci-lint run` (`internal/privacy`, `cmd`): sạch. `sqlc diff`: sạch. Đo thật (không `-race`): `BenchmarkMask4k` 1,26 ms/op (ngưỡng 5 ms); chuỗi lặp 100.000 ký tự: lâu nhất 10,9 ms (ngưỡng 50 ms; test dùng ngưỡng ×10 khi chạy `-race`, hệ số trong `race_on_test.go`).

## AC tự đánh giá
AC1 ✓ (22 mẫu dương, 21 âm) · AC2 ✓ · AC3 ✓ một phần (xem nợ 1) · AC4 ✓ · AC5 ✓ · AC6 ✓ · AC7 ✓ (`TestMaskMappingTTL`, `TestMaskingNeverLogged`, `TestSessionlessMaskNoRedisKey`) · AC8 ✓ · AC9 ✓ · AC10 ✓ · AC11 ✓ · AC12 ✓ · AC13 ✓ · AC14 ✓.

## Nợ / cần hỏi
1. **AC3 `TestRosterInvalidateNotBlockedByLongJob`** chưa có dưới tên đó: việc AI dài của Threads (hàng riêng) chưa tồn tại tới US-P3-06. Hiện chứng minh bằng `TestRosterInvalidateRegistered` (handler đã Chain, xoá khoá < 5 s). Sẽ bổ sung ca "đang có việc dài" cùng US-P3-06.
2. Các test Redis / Postgres của `internal/privacy` chạy mặc định (không cần `-tags integration`), theo thói quen của repo (container dùng chung).
3. Người đang chat được che nhờ có mặt trong roster (ACTIVE). SRS 4.2.5 còn nhắc thêm `users.full_name` / `student_code` của chính người đó; không làm riêng vì chủ phiên luôn là sinh viên ACTIVE của lớp (CourseAccess). Nếu muốn che cả khi chưa ACTIVE: báo.
4. Placeholder số `0` hoặc có số 0 đầu (`[[SV_01]]`) được chuẩn hoá bỏ số 0 đầu; `[[SV_0]]` → coi là sót.

## Cập nhật (US-P3-06) — nợ 1 đã đóng
`TestRosterInvalidateNotBlockedByLongJob` (`cmd/worker/thread_job_test.go`): việc AI của Threads treo ở LLM chạy ở consumer `ep:ingest`; `course.member_changed` vẫn xoá `ep:roster:{course}` trong ≤ 5 s. AC3 ✓ đầy đủ.

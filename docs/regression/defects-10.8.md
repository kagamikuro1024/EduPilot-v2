# Lỗi hồi quy sprint 10.8 — be0c0d96 (sprint/11-p10-eval), ep-r, qc2/SmkH (đã phân loại; cập nhật khi còn ca chạy)

Phân loại các lượt F thô: không còn F do môi trường — mọi lượt F ban đầu của AS12-03/13/14/15, AS06-06/08, AS13-01a/03, AS02-15, AS03-07, AS03-18, AS04-10, AS10-08b… đã chạy lại ra P (nguyên nhân bảng cuối). Lỗi thật xác nhận (trượt 2 lần, ổn định):

### DEF-R10.8-01 — `label-content-name-mismatch` (axe, serious) ở ba nút thanh trên của khung ứng dụng
- Mức: Trung bình (WCAG 2.5.3 Label in Name; xuất hiện ở mọi màn có khung, kể cả `/attendance` và `/checkin`)
- Sprint gốc: 10.8 · TC gốc: TC-AS08-25, TC-AS09-36, TC-AS09-39 (axe 0 lỗi nghiêm trọng) · Hồi quy do sprint sau?: KHÔNG RÕ — mã nút từ 10-2026 (`AppShell.tsx` dòng 245/266, blame 09bef60ed, f7a15c7d6), QC gốc đo axe 0; nghi do phiên bản axe-core 4.14.0 hiện tại hoặc chữ hiển thị đổi (số «22» chưa đọc)
- Môi trường: ep-r, be0c0d96, GV `r108t`, 1440 (và 375 theo `qc-as-real`)
- Bước tái hiện: đăng nhập GV → mở `/attendance` → `new AxeBuilder({page}).analyze()` (hoặc `qc-as-real.spec.ts` ca «TC-AS08-25»)
- Kỳ vọng: 0 lỗi serious/critical · Thực tế: 3 nút: `button.search` (`aria-label="Tìm nhanh hoặc đi đến"`, chữ hiển thị «Tìm nhanh hoặc đi đến…»), `button.iconBtn` (`aria-label="Thông báo, 22 chưa đọc"`), `button.profile` (`aria-label="Tài khoản: <tên>"`)
- Bằng chứng: `evidence-10.8/ui-real.log`, `evidence-10.8/ui-real/ui.jsonl` · Nghi ngờ mã: `frontend/src/shared/shell/AppShell.tsx:245-266` (tên truy cập phải chứa chữ hiển thị)
- Chạy lại lần 2: ổn định ở 1440 (5/6 lượt; lượt `qc-as-ui` ca 25b cho 0 lỗi); ở 375 chỉ `/checkin` còn 1 nút (TC-AS09-36)

### DEF-R10.8-02 — bảng truy vết SRS §11 trỏ tới 13 tên test không tồn tại
- Mức: Thấp (tài liệu) · TC gốc: TC-AS14-11, TC-AS13-05 · Hồi quy do sprint sau?: KHÔNG (có sẵn từ 10.8: tên đổi khi viết test)
- Bước tái hiện: `grep -rn "func TestWindow\|TestDisplayNoStore\|TestAssuranceLevels\|TestDeviceSharedRace\|TestAbsentUnchecked\|TestClearFlagsAudit\|TestTodayCheckinReview\|TestOverHeadcount\|TestFlagRecheckMiss\|TestFlagGeo\|TestFlagCampus\|TestNoCoordsOrIPStored\|TestAccountDeleteRemovesCheckins" backend-go --include=*_test.go` → 0 kết quả
- Kỳ vọng: tên test ở SRS §11 khớp mã · Thực tế: test thật tên khác (`TestGeoFlagsAndAssurance`, `TestRoundsRecheckAndLimit`, `TestSpotCheckPrefersFlagged`, `TestForgetStudentKeepsOthers`, …); TC-AS13-05 chạy test thật
- Nghi ngờ mã: `docs/specs/FEAT-attendance-selfcheck/SRS.md` §11 · Chạy lại lần 2: ổn định

### DEF-R10.8-03 — «Đóng» khi mất mạng thử lại bằng Idempotency-Key mới
- Mức: Thấp (không ghi đôi: server chỉ 1 cửa sổ đóng, hai yêu cầu cùng 200) · Sprint gốc: 10.8 · TC gốc: TC-AS08-20 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Môi trường: ep-r, GV `r108t`, 1440, Playwright `context.setOffline`
- Bước tái hiện: mở `/attendance?session=…` có cửa sổ đang mở → `setOffline(true)` → bấm «Đóng» (dòng «Chưa đóng») → `setOffline(false)` → quan sát các `POST …/close` (`e2e/qc-r108-more.spec.ts` ca TC-AS08-20)
- Kỳ vọng (TC): tự thử lại cùng `Idempotency-Key`, không ghi đôi · Thực tế: 3 lần offline cùng một khoá, rồi sau khi có mạng 1–2 yêu cầu với khoá khác (`keys=2…3`, `resp=fail:ep-b021c×3, 200:ep-1c28a`)
- Bằng chứng: `evidence-10.8/ui-more4.log` · Nghi ngờ mã: `frontend/src/features/attendance/RealAttendance.tsx:215-243` (`closeWindow` gọi lại `actions.close.mutateAsync`), `checkinApi.ts:101` (`useMutation` thường, khoá mới mỗi lần; cần `useIdempotentMutation`)
- Chạy lại lần 2: ổn định (3 lượt)

### DEF-R10.8-04 — nút «Quét mã điểm danh» hiện cả khi `qr_code` không được hỗ trợ
- Mức: Thấp · Sprint gốc: 10.8 · TC gốc: TC-AS11-07 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Bước tái hiện: SV mở `/checkin` với `Capacitor.isNativePlatform()=true` và `BarcodeDetector` có nhưng `getSupportedFormats()` rỗng / hàm dựng ném lỗi (`qc-r108-more2.spec.ts` ca TC-AS11-07)
- Kỳ vọng: không nút quét, ô «Mã điểm danh» hiện ngay · Thực tế: nút «Quét mã điểm danh» hiện (`btn=1, input=0`); bấm → rơi vào nhánh «từ chối camera»
- Nghi ngờ mã: `frontend/src/features/checkin/Scanner.tsx:10` (`hasBarcodeDetector` chỉ kiểm `"BarcodeDetector" in window`), `CheckinScreen.tsx:60-61` · Chạy lại lần 2: ổn định

## Ghi nhận (không tính lỗi)
1. `/me/today` mục `CHECKIN_REVIEW` đôi lúc vắng khi máy tải cao (provider «Hôm nay» hạn 60 ms, `today/providers.go:18`): 3 lần vắng / 8 lần đo; chờ cache 60 s hoặc gọi lại thì có. Go test `TestMeAttendanceCheckedAtAndTodayReview` trượt cùng lý do khi tải cao.
2. `backend-go/internal/attendance/token.go:65,88` còn chú thích tên cũ `TOKEN_INVALID`/`TOKEN_EXPIRED` (mã chạy đúng `CHECKIN_TOKEN_*`).
3. Go: `TestAttestRealChains`, `TestCheckinFlow`, `TestCheckinFlags`, `TestPurgeKeepsRecords`, `TestMeAttendanceCheckedAtAndTodayReview` hết hạn / trượt khi máy tải 30+; chạy lại khi tải thấp: toàn bộ PASS (`go-attendance3.log`, `go-attest3.log`, `-race` ok). `go test ./internal/contract` cần `-tags testroutes` (Makefile:17).

# Lỗi hồi quy sprint 10.8 — be0c0d96 (sprint/11-p10-eval), ep-r, qc2/SmkH (đã phân loại; cập nhật khi còn ca chạy)

Phân loại các lượt F thô: không còn F do môi trường — mọi lượt F ban đầu của AS12-03/13/14/15, AS06-06/08, AS13-01a/03, AS02-15, AS03-07, AS03-18, AS04-10, AS10-08b… đã chạy lại ra P (nguyên nhân bảng cuối). Lỗi thật xác nhận (trượt 2 lần, ổn định):

### DEF-R10.8-01 — `label-content-name-mismatch` (axe, serious) ở ba nút thanh trên của khung ứng dụng
- Mức: Trung bình (WCAG 2.5.3 Label in Name; xuất hiện ở mọi màn có khung, kể cả `/attendance` và `/checkin`)
- Sprint gốc: 10.8 · TC gốc: TC-AS08-25, TC-AS09-36, TC-AS09-39 (axe 0 lỗi nghiêm trọng) · Hồi quy do sprint sau?: KHÔNG RÕ — mã nút từ 10-2026 (`AppShell.tsx` dòng 245/266, blame 09bef60ed, f7a15c7d6), QC gốc đo axe 0; nghi do phiên bản axe-core 4.14.0 hiện tại hoặc chữ hiển thị đổi (số «22» chưa đọc)
- Môi trường: ep-r, be0c0d96, GV `r108t`, 1440 (và 375 theo `qc-as-real`)
- Bước tái hiện: đăng nhập GV → mở `/attendance` → `new AxeBuilder({page}).analyze()` (hoặc `qc-as-real.spec.ts` ca «TC-AS08-25»)
- Kỳ vọng: 0 lỗi serious/critical · Thực tế: 3 nút: `button.search` (`aria-label="Tìm nhanh hoặc đi đến"`, chữ hiển thị «Tìm nhanh hoặc đi đến…»), `button.iconBtn` (`aria-label="Thông báo, 22 chưa đọc"`), `button.profile` (`aria-label="Tài khoản: <tên>"`)
- Bằng chứng: `evidence-10.8/ui-real.log`, `evidence-10.8/ui-real/ui.jsonl` · Nghi ngờ mã: `frontend/src/shared/shell/AppShell.tsx:245-266` (tên truy cập phải chứa chữ hiển thị)
- Trạng thái: ĐÃ SỬA (P-11-39, dbf48ed0). Chạy lại trên ep-r sau dựng lại: TC-AS08-25 (1440 và 375) P · TC-AS09-36 P (axe 0, LCP 56 ms) · TC-AS09-39a P (axe 0 ở /checkin 375). Bằng chứng: `evidence-10.8/rerun-axe.jsonl`, `ui-real3.log`, `ui-more7.log`, `ui-real2x/`. Lưu ý: ô chọn lớp đổi nhãn («QC-AS · …»), bản sao `qc-as-real.spec.ts` chọn mục bằng `getByRole("menuitem", {name: /^QC-AS · /})`.
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
- Trạng thái DEF-03: ĐÃ SỬA (b3c039c0). Chạy lại sau dựng lại: TC-AS08-20 P (queued=1, closed=1, 4 yêu cầu cùng một khoá, `ui-def03.log`, spec tách `e2e/qc-r108-def03.spec.ts`). Ghi chú: ca gốc trong `qc-r108-more.spec.ts` ca «B /attendance» lượt này đỏ vì `pickClass` chạy trước khi nút chọn lớp hiện (đua giao diện sau dựng lại), không phải lỗi sản phẩm; bản tách đã chờ nút.

### DEF-R10.8-04 — nút «Quét mã điểm danh» hiện cả khi `qr_code` không được hỗ trợ
- Mức: Thấp · Sprint gốc: 10.8 · TC gốc: TC-AS11-07 · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Bước tái hiện: SV mở `/checkin` với `Capacitor.isNativePlatform()=true` và `BarcodeDetector` có nhưng `getSupportedFormats()` rỗng / hàm dựng ném lỗi (`qc-r108-more2.spec.ts` ca TC-AS11-07)
- Kỳ vọng: không nút quét, ô «Mã điểm danh» hiện ngay · Thực tế: nút «Quét mã điểm danh» hiện (`btn=1, input=0`); bấm → rơi vào nhánh «từ chối camera»
- Nghi ngờ mã: `frontend/src/features/checkin/Scanner.tsx:10` (`hasBarcodeDetector` chỉ kiểm `"BarcodeDetector" in window`), `CheckinScreen.tsx:60-61` · Chạy lại lần 2: ổn định
- Trạng thái DEF-04: ĐÃ SỬA (b3c039c0). Chạy lại sau dựng lại frontend 01:03: TC-AS11-07 P (`ui-more9.log`, `rerun-def.jsonl`).

### DEF-R10.8-05 — Back khi đang quét QR đưa cả app Android ra launcher thay vì đóng khung quét
- Mức: Trung bình · Sprint gốc: 10.8 · TC gốc: TC-AS11-17 (US-AS-15 AC7(a)) · Hồi quy do sprint sau?: KHÔNG (có sẵn)
- Môi trường: AVD `ep-api36` (Android 16), APK debug HEAD, VPS e8b2162d, SV `sv.gioi`
- Bước tái hiện: mở app, vào `/checkin`, bấm «Quét mã điểm danh» (hoặc tap thật), bấm Back (phím, nút 3 nút)
- Kỳ vọng: khung quét đóng, về ô nhập mã · Thực tế: app thoát ra launcher; `history.state` có `epScan:true`, `history.length` 3; `history.back()` qua CDP đóng khung quét đúng ⇒ phía web đúng
- Nghi ngờ mã: `mobile/android` `MainActivity.java` (vỏ Capacitor, không xử lý Back / không gọi `webView.goBack()`)
- Bằng chứng: `evidence-10.8/android/ket-qua-android-vps-8302ecca.md`, ảnh `shots-vps/` (AC-16) · Chạy lại lần 2: ổn định
- Trạng thái: MỞ (dev đang sửa; chạy lại TC-AS11-17 sau khi sửa)

### DEF-R10.8-06 — gateway không nhận `APP_PUBLIC_URL`: `qr_url` / `join_url` / feed lịch dựng từ `https://localhost`
- Mức: Cao · Sprint gốc: 10.8 · TC gốc: TC-AS11-17 (AC-3) · Hồi quy do sprint sau?: KHÔNG RÕ
- Môi trường: VPS, AVD `ep-api36`
- Kỳ vọng: URL công khai theo `APP_PUBLIC_URL` · Thực tế: compose chỉ truyền biến cho worker ⇒ gateway dựng URL từ `https://localhost`; app bỏ qua QR khác origin (quét 0/N)
- Trạng thái: ĐÃ SỬA e8b2162d (thêm một dòng compose; VPS chạy bản đó); chạy lại: AC-3 P (1,7 s, WEB mức 2), AC-3b P

## Ghi nhận (không tính lỗi)
1. `/me/today` mục `CHECKIN_REVIEW` đôi lúc vắng khi máy tải cao (provider «Hôm nay» hạn 60 ms, `today/providers.go:18`): 3 lần vắng / 8 lần đo; chờ cache 60 s hoặc gọi lại thì có. Go test `TestMeAttendanceCheckedAtAndTodayReview` trượt cùng lý do khi tải cao.
2. `backend-go/internal/attendance/token.go:65,88` còn chú thích tên cũ `TOKEN_INVALID`/`TOKEN_EXPIRED` (mã chạy đúng `CHECKIN_TOKEN_*`).
3. Go: `TestAttestRealChains`, `TestCheckinFlow`, `TestCheckinFlags`, `TestPurgeKeepsRecords`, `TestMeAttendanceCheckedAtAndTodayReview` hết hạn / trượt khi máy tải 30+; chạy lại khi tải thấp: toàn bộ PASS (`go-attendance3.log`, `go-attest3.log`, `-race` ok). `go test ./internal/contract` cần `-tags testroutes` (Makefile:17).
4. Android emulator: chỉ quyền Xấp xỉ mà provider mạng chưa có vị trí thì `getCurrentPosition` hết 8 s ⇒ `GEO_DENIED` (máy thật ⇒ `GEO_WEAK`); hộp «Chính xác / Gần đúng» của Android 12+ làm trễ lượt (có thể `TOKEN_LATE`).

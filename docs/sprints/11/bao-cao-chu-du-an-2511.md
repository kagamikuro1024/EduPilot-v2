# Báo cáo chủ dự án — đêm 25→26/11 (dừng theo lệnh chủ dự án)

## Kết quả chính
- **main = 27ea911a**: PR #7 → #16 đã merge, CI xanh đúng đầu nhánh mỗi lần. **PR #17 còn mở** (DEF-MOB-12/12b, LH-GRADEBOOK-JS, P-11-149): chờ CI + qc2 nghiệm lại trên AVD API 28.
- **Hạ tầng**: Docker/colima trên Mac đã xoá (trả ~10 GB RAM, ổ trống 181 → 225 GB). Mọi stack + test chạy trên VPS (`ep-vps-test`). Mac chỉ chạy giả lập điện thoại.
- **Demo trên VPS** https://v2202609419695521098.happysrv.de — LLM thật (OpenCode Go: qwen3-embedding-8b + deepseek-v4.1-flash), chứng chỉ thật, seed đủ, insights 30 ngày hai lớp.
- **Đội agent** chạy DeepSeek V4.1 Flash (thinking low); PM giữ Opus, 429 tự rơi sang DeepSeek.

## Đánh giá E1–E5 (nhãn người, chấm mù)
| E | Kết quả |
|---|---|
| E1 tường lửa | κ thô 0,860 (53/57), sau phân xử 57/57; F1 0,980 |
| E2 hỏi đáp | 24/24 nhãn đúng; đúng P1 0,862 / P2 0,860; bịa 0 |
| E3 chấm tự luận | QWK 0,529 (60 bài) — **G3 chưa đạt** (mục tiêu ≥ 0,7) |
| E5 câu AI sinh | dùng được 45 %; 2 lỗi đã sửa ([Cn], đáp án dồn A) |

## Đồ án
- Chốt 109 tr (thân bài 88 tr), 0 lỗi, 4 ảnh UI chụp lại trên VPS (LLM thật), soát chéo BA + techlead-ui đạt.

## Giao diện điện thoại
- techlead-ui soát 360–430 + ngang: 21 lỗi M1–M21 → đã sửa, soát lại đạt.
- Bộ TC điện thoại 345 ca — **lúc dừng: 288 mã có kết quả = 223 P / 1 F / 64 N** (gộp theo mã, khử trùng; qc2/regression 34b68b9e, 7391e857). Android 102 TC = 85 P / 1 F / 16 N; iOS (qc3) lô A–E + một phần F/G xong; còn ~57 mã chưa chạy (iOS SES/ENT/MSG/DAT còn lại).
- 1 F còn mở: **INS-09** (Android 9 / WebView 66). Đã sửa cú pháp (980b705d, qc2 xác nhận hết SyntaxError) + DEF-MOB-12b (0316ef2b: `flatMap` trong khung app, `AbortSignal.any` trong apiClient); dev3 chỉ kiểm bằng mô phỏng, **chưa nghiệm trên AVD thật** (dừng giữa chừng).
- Lỗi từ lượt chạy TC, đã sửa + nghiệm lại đạt trên VPS / giả lập:
  - DEF-MOB-01 lớp lưu trữ ở /checkin · DEF-MOB-02 thiếu «Thử lại» khi mất mạng · DEF-MOB-03 treo «đang gửi» tới 45 s → trần 15 s.
  - DEF-MOB-05 liên kết QR mới bị mã cũ trong tab chặn (kèm hồi quy skip link do techlead-ui bắt) · DEF-MOB-06 /attendance 360 px hai nút cách 4 px → 8 px · DEF-MOB-07 mất câu «Mã đã hết hạn» sau đăng nhập chậm.
  - Định vị (lớp bật geofence): DEF-MOB-08 hộp quyền bị tính vào trần mạng ⇒ «Chưa có mạng.»; DEF-MOB-09 hộp quyền hiện sau khi bấm gửi ⇒ mã xoay 10 s hết hạn; DEF-MOB-10 đọc hộp quyền > 10 s bị cờ GEO_DENIED oan.
  - DEF-MOB-11 tắt camera toàn máy ⇒ khung quét đen, không lối ra.
  - DEF-MOB-04 rút (TC sai, P-11-143). TC sửa qua proposal: P-11-142..149.
- ~35 ca cần **máy thật** (camera thật, VoiceOver/TalkBack, định vị thật iOS, PWA thao tác tay, attestation, APK release ký): danh sách trong `tc-mobile-run-android.md` và §9 `tc-mobile-app.md` (qc2/regression).
- Hiệu năng: `/settings/llm` JS 255.159 B, chỉ còn 841 B dưới ngân sách 256 KB — nợ phase PR.

## Chờ chủ dự án
1. **P-11-140** (auth): tải lại nhanh / nhiều tab văng /login do xoay refresh token không ân hạn. Chọn A (ân hạn ~10 s, khuyến nghị) / B (giữ) / C.
2. **PR #17 — CI ĐỎ lúc dừng (chưa merge)**: 1 ca e2e `today.spec.ts:248` («Thiết lập lớp mới» → «Bỏ qua»: hàng biến mất nhưng KHÔNG có POST `…/setup/dismiss`), 419 ca khác xanh. Nghi hồi quy từ `0316ef2b`/`980b705d` (compat: `anySignal`, `newUuid` thay `crypto.randomUUID`) trên đường mutation — cần dev xem trước khi merge. Muốn chắc INS-09 thì chạy thêm `docs/regression/evidence-mobile-android/avd-api28.sh` (≈10 phút).
3. Bìa, cảm ơn, cam đoan; thân bài 88 tr < 110 (muốn dài thêm thì chỉ mục).
4. H4.7 (tuỳ chọn); đổi khoá OpenCode Go khi xong; xoá/giữ 3 thư mục `backup/wt-*`; khoá ký APK release (INS-01/02).
5. VPS (đã xong, chỉ để biết): chạy `0316ef2b`; giới hạn đăng nhập ĐÃ TRẢ về mặc định 10 / 300 / 30 (kiểm: 10 lượt sai → 401, lượt 11 → 429). Dữ liệu QC còn lại: lớp QCMOB1 có thêm buổi 104 (đã có điểm danh, không xoá được).

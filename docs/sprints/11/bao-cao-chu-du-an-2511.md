# Báo cáo chủ dự án — 25→27/11

## Kết quả chính
- **main = 8da158a5**: PR #7 → #21 đã merge, CI xanh đúng đầu nhánh mỗi lần.
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
- **Bộ TC điện thoại XONG**: 345 mã định nghĩa = **261 P / 0 F / 84 N**; kèm 15 mã `CL-*` ngoài bảng: 360 mã = 271 P / 0 F / 89 N. Nguồn: qc3 iOS 108/0/26, qc2 Android 141 TC = 117/0/24, qc (VPS Chromium) 84/0/55. N = cần máy thật / APK release ký / thao tác tay (lý do từng mã ở §9 `tc-mobile-app.md`, qc2/regression).
- Android 9 / WebView 66 (INS-09) chạy đủ: đăng nhập, điểm danh; app cam kết WebView ≥ 64 (P-11-149).
- Lỗi từ lượt chạy TC, đã sửa + nghiệm lại đạt trên VPS / giả lập:
  - DEF-MOB-01 lớp lưu trữ ở /checkin · DEF-MOB-02 thiếu «Thử lại» khi mất mạng · DEF-MOB-03 treo «đang gửi» tới 45 s → trần 15 s.
  - DEF-MOB-05 liên kết QR mới bị mã cũ trong tab chặn (kèm hồi quy skip link do techlead-ui bắt) · DEF-MOB-06 /attendance 360 px hai nút cách 4 px → 8 px · DEF-MOB-07 mất câu «Mã đã hết hạn» sau đăng nhập chậm.
  - Định vị (lớp bật geofence): DEF-MOB-08 hộp quyền bị tính vào trần mạng ⇒ «Chưa có mạng.»; DEF-MOB-09 hộp quyền hiện sau khi bấm gửi ⇒ mã xoay 10 s hết hạn; DEF-MOB-10 đọc hộp quyền > 10 s bị cờ GEO_DENIED oan.
  - DEF-MOB-11 tắt camera toàn máy ⇒ khung quét đen, không lối ra.
  - DEF-MOB-12/12b/12c WebView 66: cú pháp mới, `flatMap`/`AbortSignal.any`, Capacitor chèn `\n` sau `<head>` làm React #418.
  - DEF-MOB-13 liên kết QR + phiên bị thu hồi ⇒ mất mã sau đăng nhập lại · DEF-MOB-14 tài khoản thật 0 lớp thấy nhãn «Bản mô phỏng» (P-11-150).
  - DEF-MOB-15 (a–d) TalkBack ở /checkin: «Đang gửi», lỗi `role=alert`, đếm ngược 429 không đọc mỗi giây, giữ focus (P-11-151) · DEF-MOB-16 Android tắt app ≤ 20 s sau đăng nhập ⇒ mất phiên (flush cookie WebView).
  - DEF-MOB-04 rút (TC sai, P-11-143). TC sửa qua proposal: P-11-142..152.
- Hiệu năng: 9 URL đều dưới ngân sách JS 256 KB (`/gradebook` 249.360 B, `/settings/llm` 241.131 B).
- Nợ: rà các gói test Go dùng đồng hồ giả từ giờ thật (đỏ khi CI chạy quanh 00:00 hoặc ~06:00 ICT) — dev đang rà.

## Chờ chủ dự án
1. **P-11-140** (auth): tải lại nhanh / nhiều tab văng /login do xoay refresh token không ân hạn. Chọn A (ân hạn ~10 s, khuyến nghị) / B (giữ) / C.
2. (đã xong) PR #17 → #21 merge 26–27/11.
3. Bìa, cảm ơn, cam đoan; thân bài 88 tr < 110 (muốn dài thêm thì chỉ mục).
4. H4.7 (tuỳ chọn); đổi khoá OpenCode Go khi xong; xoá/giữ 3 thư mục `backup/wt-*`; khoá ký APK release (INS-01/02).
5. VPS (đã xong, chỉ để biết): frontend chạy `7b6b5ab3` (= mã sản phẩm của main 8da158a5; các commit sau chỉ là test/tài liệu/native Android); APK debug mới (DEF-MOB-16) ở `~/Documents/as-emu-evidence/app-debug-vps-HEAD.apk`. Giới hạn đăng nhập mặc định 10 / 300 / 30. Dữ liệu QC còn lại: lớp QCMOB1 có thêm các buổi 104–118.
6. **Tài khoản demo `sv.gioi`, `sv.kha`**: một script QC 25/11 đã cho hai SV này vào 10 lớp QC; 26/11 đã gỡ (9 lớp qua `DELETE /members`; QCMOB4 đã lưu trữ nên PM duyệt sửa tay 2 hàng `enrollments`, ghi trong `tc-mobile-run-qc.md`). Nay chỉ còn lớp demo 761987 (+ 761988 cho `sv.gioi`). Lượt điểm danh trong lớp QC vẫn nằm trong DB. Trong lúc nghiệm, phiên `sv.gioi` bị đăng xuất mọi thiết bị vài lần — nếu anh đang đăng nhập `sv.gioi` thì đăng nhập lại.

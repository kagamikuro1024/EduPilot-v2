# dev2 — US-AS-15 (trải nghiệm điện thoại `/checkin`) + QC VPS

Nhánh `sprint/11-p10-eval`. Test: `frontend/e2e/checkin-mobile.spec.ts` (desktop project, tự đặt viewport; đỏ trước trên mã cũ, xanh sau).

## Bảng AC → mã / test

| AC | Trạng thái | Mã | Test |
|---|---|---|---|
| AC1 độ rộng 360/390/411/430 × 7 trạng thái | Đạt | dải mất mạng trừ khỏi chiều cao khung: `OfflineBanner.tsx` đặt `--ep-banner-h`, `AppShell.module.css` `.shell`/`.main` trừ biến | `checkin mobile @W: <trạng thái>` (28 ca: không cuộn ngang, nút chính trong màn, không cuộn dọc, dock không đè thanh dưới) |
| AC2 xoay ngang | Đạt | `CheckinScreen.module.css`: màn thấp + ngang ⇒ khung quét lấp giữa thanh trên / dưới, camera vuông theo chiều cao | `checkin landscape @667x375 / 844x390 / 915x412` |
| AC3 vùng an toàn | Đạt (đo bằng CDP `Emulation.setSafeAreaInsetsOverride`) | `layout.tsx` `viewport` `viewportFit: "cover"`; `tokens.css` `--ep-topbar` cộng `env(safe-area-inset-top)`; `AppShell.module.css` thanh trên + thanh dưới đệm top / left / right / bottom; `base.css` body đệm trái / phải; dock `/checkin` đệm trái / phải | `checkin @390 dọc có tai thỏ…`, `checkin ngang … tai thỏ trái / phải 44`, `viewport-fit=cover…` |
| AC4 cỡ chữ lớn | Chưa đo (tay) | — | — |
| AC5 chế độ tối | Giữ sáng (D59); `theme-color` trắng khớp thanh trên | `layout.tsx` `themeColor`, `manifest.ts` | `có apple-touch-icon…` |
| AC6 bàn phím | Một phần: `interactiveWidget: "resizes-content"` (Chrome Android co khung nhìn ⇒ ô nhập + dock cùng thấy); `enterkeyhint=go`, nhãn thật, khoá khi đang gửi giữ nguyên | `layout.tsx` | `checkin-code.spec.ts` (giữ). Bàn phím ảo thật: kiểm tay |
| AC7a Back khi quét | Đạt | `CheckinScreen.tsx`: vào quét đẩy một mục lịch sử, `popstate` ⇒ về ô nhập mã; `Scanner` bỏ qua `play()` bị huỷ khi rời màn | `Back đang quét…` (lặp 12 lần: không nháy) |
| AC7b–d | (b) giữ nguyên (token đã xoá); (c), (d) native: dev3 | — | — |
| AC8 cấp lại quyền | Đạt | `CheckinScreen.tsx`: `visibilitychange` → `permissions.query("camera")`, khác `denied` ⇒ trả nút quét, bỏ dòng từ chối; không tự xin quyền | `từ chối camera rồi cấp lại…` |
| AC9 mạng yếu | Đạt (không đổi mã) | `apiClient` tự gửi lại lỗi mạng với CÙNG `Idempotency-Key`; `Scanner` dừng sau mã đầu | `quét cùng mã nhiều lần ⇒ một lượt gửi…` (đếm khoá, không đếm yêu cầu) |
| AC10 chạy nền | Giữ nguyên (`Scanner` thoát khi ẩn tab) | — | `checkin-scan.spec.ts` |
| AC11 PWA iOS | Đạt phần mã | `layout.tsx` `icons.apple`, `appleWebApp`; `manifest.ts` `display: standalone` | `có apple-touch-icon, tên «EduPilot»…`. Máy thật: KHÔNG KIỂM ĐƯỢC |
| AC12 trình đọc màn hình | Một phần | vùng kết quả `aria-live="polite"` | `«Đã điểm danh» nằm trong vùng aria-live polite`. TalkBack / VoiceOver: kiểm tay |
| AC13 vùng chạm ≥ 44 | Đạt | (đã đạt từ trước) | trong 28 ca AC1: mọi `button, a, input` ở nội dung + thanh dưới + dock |
| AC14 lỗi tiếng Việt | Đạt | `Scanner.tsx` phân loại `NotAllowedError`/`SecurityError` ⇒ «Chưa cấp quyền camera.»; còn lại ⇒ «Không mở được camera.» | `checkin camera <lỗi> ⇒ …` (4 ca) |
| AC15 | Không đổi | — | — |

Cùng đợt, `canScanQr()` thử `detect()` trên QR mẫu vẽ tại chỗ (P-11-131, `8048e549`).

## Phối hợp vùng an toàn với app Android (dev3)

`viewport-fit=cover` + `env(safe-area-inset-*)` ở khung là MỘT cách cho PWA iOS và app Android. `SystemBars` `insetsHandling: "css"` (dev3) cấp giá trị cho `env()` ở WebView ≥ 140; WebView cũ thì decor tự đệm và `env()` = 0 — cùng CSS vẫn đúng (cộng 0).

## QC VPS (P-11-128 mở rộng)

- `/attendance` 375 px: ô chọn Buổi xếp nhãn «Buổi» trên ô, ô đủ bề rộng; test đo chiều rộng chữ dài nhất so với chỗ trống của ô.
- `/observability` (Admin): bỏ dòng chú lặp «Thư chưa gửi được, mới nhất trước» (nay là tên bảng đọc cho trình đọc màn hình: «Danh sách thư lỗi»).

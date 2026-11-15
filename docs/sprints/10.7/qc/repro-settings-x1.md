# `/settings` (giảng viên): ca «alert×0» của QC là do cách chặn của QC, không phải lỗi dev (đã đóng)

Bản đầu của script QC giữ cả `/api/v1/me/*` (qua `route.continue()`), mà `/settings` dùng `/me/sessions` và `/me/settings`, nên trang không bao giờ vào lỗi. Đã sửa: chỉ giữ `/api/v1/auth/*`, `/api/v1/me`, `/api/v1/me/courses`. Ở HEAD `a99a84e9` `/settings` GV ra 2 alert (hợp lệ theo TL 14.7), còn lệch rộng alert 750 / 317 so với cột 800 / 351 (xem `summary.md` mục 2). Phần dưới giữ để tái hiện.

## Tái hiện cũ

Đo ở HEAD `1935435a` (audit `sau` đầu tiên). Cùng kiểu ở `analytics`, `observability` (giảng viên), `student/chat`. Chưa đo lại ở HEAD mới khi viết file này; kết quả lượt sau ở `audit.md`.

| Mục | Giá trị |
| --- | --- |
| Vai, tài khoản | giảng viên `teacher@edupilot.local`, lớp 761987 đang chọn |
| Route | `/settings` |
| Bề rộng | 1440 và 375 |
| Cách vào trạng thái | Đăng nhập, mở `/` bình thường (ấm). Sau đó đặt `page.route("**/api/v1/**")` và vào `/settings` bằng `page.goto(url, { waitUntil: "domcontentloaded" })` |
| Giữ nguyên | `/api/v1/auth/*`, `/api/v1/me`, `/api/v1/me/courses` (qua `route.continue()`) |
| Lỗi | mọi yêu cầu `/api/v1/**` khác: `route.fulfill({ status: 500, contentType: "application/json", body: '{"code":"INTERNAL","message":"Lỗi máy chủ."}' })` |
| Đang tải | cùng bộ chặn nhưng `await sleep(5000); route.continue()` |
| Đo | lỗi: chờ 2000 ms rồi đếm `main [role=alert]` → **0**. Đang tải: chờ 800 ms rồi đếm `main [aria-busy=true]` → **0** |
| Mã đo | `frontend/e2e/qc-ui-audit-real.spec.ts` dòng `await page.route("**/api/v1/**"` và hàm `probe` (`alerts`, `busy`) |

Lệnh chạy riêng một route (cần stack `ep-qc` và mật khẩu seed):

```
QC_REAL=1 QC_PW=<SEED_DEFAULT_PASSWORD> QC_STATE=error QC_CLASS=761987 QC_ROLE=teacher QC_ROUTES='^/settings$' \
  QC_OUT=/tmp/qc pnpm exec playwright test -c playwright.qc-audit.config.ts
```

Kết quả nằm ở `$QC_OUT/sau/audit-error.jsonl` (`alertCount`, `busy`, `apiHits` = số yêu cầu bị chặn; `apiHits = 0` nghĩa là màn không gọi API nên L/X không áp dụng).

Khác biệt có thể giải thích số của dev (busy=2 / alert=2): chặn cả `/api/v1/me/*` hoặc `/api/v1/auth/*`, hoặc vào route qua điều hướng trong ứng dụng thay vì `goto`. QC giữ ba đường dẫn trên để phiên không rơi.

# Handoff — OPS-VPS landing (Dev)

Nhánh `ops/vps-deploy`. Đã triển khai: `https://v2202609419695521098.happysrv.de/` (chỉ dựng lại container `frontend`).

## Làm gì
- `/` chưa đăng nhập → trang giới thiệu (HTML tĩnh từ máy chủ, không gọi API): hero + Đăng nhập / Tạo tài khoản + ảnh màn duyệt bài; 3 điểm khác biệt có số đo; giảng viên (6 dòng, 2 ảnh); sinh viên (6 dòng, ảnh chat + ảnh 375); chân trang đồ án + GitHub + "Dữ liệu trên bản demo là dữ liệu mô phỏng". Không có tài khoản demo trên trang.
- `/` đã đăng nhập → "Hôm nay" như cũ (`HomeGate`, title "Hôm nay · EduPilot"). Route sâu (`/gradebook`...) vẫn về `/login?next=`.
- Người quay lại: cờ `ep_returning` (localStorage, không phải bí mật) + script đầu trang đặt `data-returning` trên `<html>` → vẽ khung PreShell "Hôm nay" thay vì nháy trang giới thiệu (giữ LCP cũ). Phiên hết hạn → cờ bị gỡ, trang giới thiệu hiện.
- Logo ở `/login` là liên kết về `/` (chỉ `/login`, vì test invite khẳng định trang đó không có liên kết nào).
- Token `--ep-text-display` (chữ tiêu đề trang giới thiệu). Ảnh WebP thật chụp từ bản demo trong `frontend/public/landing/` (hero ẩn ở ≤ 719 px để chữ là LCP; ảnh 640 px cho điện thoại).
- Số liệu lấy từ báo cáo sprint 6 (146 payload, 0 rò), 8 (30/30), 9 (+0,05 %, 22 bài, 20 SV, nhà cung cấp AI mô phỏng).

## Kết quả (chạy trên VPS `/srv/work/dev`)
| Cổng | Kết quả |
|---|---|
| `pnpm lint`, `tsc --noEmit`, `scripts/ui-antipatterns.sh` | sạch |
| Playwright desktop, cả bộ (trước chỉnh cuối) | 633 pass; 3 đỏ do hai `<main>` cùng HTML và logo có liên kết → sửa code (không sửa assertion) |
| Sau sửa: landing + account + shell + lcp + a11y + today | 92 pass; axe 154 lượt quét: critical 0, serious 0 |
| `landing.spec.ts` (mới, 7 ca) | pass: anonymous, đã đăng nhập, deep link, logo từ /login, cờ cũ hết hạn, 1440 và 375 (không cuộn ngang, axe, ảnh có kích thước) |
| Lighthouse (VPS, `/`, 3 lần) | desktop perf 100, LCP 0,57–0,65 s; mobile perf 96–99, a11y 100, LCP 2,1 / 2,1 / 2,57 s (phần tử LCP = h1), CLS 0 |

Ghi chú LCP mobile: trung vị 2,1 s; một lần 2,57 s trên mô phỏng 4G chậm (đo từ chính VPS). Ảnh hero từng làm LCP 2,4–3,2 s nên đã ẩn ở điện thoại.

## Test cũ bị đụng
Không sửa assertion nào. Chỉ `e2e/support/session.ts` (`asDemo`) đặt thêm cờ `ep_returning` để `/` của phiên giả vẫn vẽ khung "Hôm nay" ngay (case `lcp before refresh /`). Không cần dòng proposal.

## Ảnh
`docs/sprints/ops-vps/handoff/landing/pm-landing-{1440,375}-{top,sv}.png`, `pm-landing-1440-gv.png`.

## Còn lại
- BT03 trên demo vẫn có 22 bản nháp từ lần chấm thử (xem dev-deploy.md).
- Chưa chạy lại cả bộ Playwright sau 3 chỉnh cuối (chỉ 6 spec liên quan + landing); Playwright mobile project chưa chạy.

# PR — Sẵn sàng thí điểm thật

| Ước lượng | Phụ thuộc | Nhánh |
| --- | --- | --- |
| 2 tuần | P10 (hoặc P5 nếu đi đường thí điểm sớm, xem `PRODUCTION_READINESS.md` mục 5) | `feat/pr-production` |

**Mục tiêu:** hệ thống chạy được với người thật và dữ liệu thật mà không có bạn ngồi cạnh. Không bắt buộc cho buổi bảo vệ; **bắt buộc** trước khi bất kỳ sinh viên thật nào đăng nhập.

Đọc trước: `PRODUCTION_READINESS.md` toàn bộ; `FLOWS.md` F1, F15, F16, F18.

## Lát việc

**L1. Bảo mật và file**
- [ ] Rà OWASP ASVS mức 1 (ghi kết quả vào `docs/security-review.md`); header bảo mật ở Caddy; cookie `Secure`; `govulncheck`, `pip-audit`, `pnpm audit` trong CI
- [ ] Danh sách trắng loại file theo magic bytes; container ClamAV; quét trong `ingest.jobs` và khi nhận bài nộp; file nhiễm → cách ly + báo Admin
- [ ] Công tắc tính năng AI theo lớp (chat / tự trả lời Threads / chấm bài) ở `/admin/courses`

**L2. Dữ liệu cá nhân và vòng đời**
- [ ] Màn đồng ý có phiên bản (`consents(user_id, doc, version, accepted_at, optional_flags)`); theo dõi thời gian học là mục tuỳ chọn tách riêng
- [ ] `/settings/privacy`: xuất dữ liệu của tôi (việc nền → ZIP qua URL ký sẵn), yêu cầu xoá (việc cho Admin; ẩn danh hoá bản ghi học vụ, xoá hẳn phần còn lại)
- [ ] F18: lưu trữ lớp (chỉ-đọc), xuất dữ liệu lớp, nhân bản sang học kỳ mới, job xoá theo thời hạn cấu hình ở `retention_policies`
- [ ] Trang tĩnh: thông báo quyền riêng tư, điều khoản, giới hạn của AI (nội dung do trường duyệt; repo chỉ có bản mẫu)

**L3. Vận hành**
- [ ] Sao lưu tự động Postgres + object storage, mã hoá, xoay vòng; `make restore-drill` khôi phục vào máy sạch và chạy smoke test
- [ ] Giám sát + cảnh báo theo bảng ở `PRODUCTION_READINESS.md` mục 3; `/admin/health`, `/admin/audit`
- [ ] `docker-compose.staging.yml`; quy trình: staging → smoke → production; quay lui bằng tag image
- [ ] `docs/RUNBOOK.md` với 7 kịch bản; diễn tập mỗi kịch bản một lần, ghi thời gian xử lý
- [ ] SMTP thật có SPF / DKIM; kiểm điểm thư rác của mail mẫu

**L4. Đăng nhập bằng tài khoản trường (tuỳ chọn, sau cờ)**
- [ ] OIDC Microsoft Entra ID: `openid profile email`; `OIDC_ENABLED=false` mặc định; nối tài khoản theo email đã xác minh; test với máy chủ OIDC giả lập trong compose

**L5. Con người**
- [ ] `/help` + hướng dẫn 1 trang (sinh viên), 2 trang (giảng viên), video 3 phút
- [ ] Cho 2 người ngoài dự án dùng thử theo kịch bản, không hướng dẫn miệng; sửa những chỗ họ kẹt
- [ ] Chạy lại `make load-t1` trên hạ tầng thí điểm

## Cổng nghiệm thu
```bash
make restore-drill                       # khôi phục + smoke test xanh; ghi thời gian
make security-check                      # govulncheck + pip-audit + pnpm audit: 0 lỗ hổng mức cao
pnpm -C frontend exec playwright test semester-end.spec.ts privacy-rights.spec.ts
curl -sI https://$STAGING/ | grep -iE "strict-transport|content-security|x-content-type"
bash scripts/chaos-prod.sh               # tắt lần lượt Redis, Python, SMTP → cảnh báo về hộp thư trong 5 phút, hệ thống tự hồi
```

## Bạn tự kiểm
- Đưa điện thoại cho một người chưa từng thấy EduPilot: họ đăng ký, xác minh, vào lớp bằng mã, hỏi một câu, trong 5 phút, không hỏi bạn câu nào?
- Tải thử file EICAR lên: bị chặn, có thông báo dễ hiểu.
- Yêu cầu xuất dữ liệu của chính mình: file ZIP có đủ và chỉ có dữ liệu của bạn.
- Đọc `RUNBOOK.md` như thể bạn là người khác lúc 2 giờ sáng: làm theo được không?
- Đã có văn bản đồng ý của trường cho các mục ở `PRODUCTION_READINESS.md` mục 2 chưa?

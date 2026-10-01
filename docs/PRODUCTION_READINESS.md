# EduPilot v2 — Sẵn sàng đưa vào trường dùng thật

**Trạng thái (D44): toàn bộ tài liệu này nằm NGOÀI phạm vi đồ án.** Đồ án chạy trên dữ liệu mô phỏng nên không phát sinh nghĩa vụ về dữ liệu cá nhân. Tài liệu này để sẵn cho ngày trường muốn dùng thật; khi đó trường là bên lo pháp lý, và phase PR là phần việc kỹ thuật đi kèm. Ranh giới duy nhất phải giữ: chưa làm PR thì không nạp dữ liệu sinh viên thật vào hệ thống, kể cả "import thử một danh sách lớp".

Có hai vạch đích khác nhau, đừng nhập làm một:

| Vạch đích | Nghĩa là | Đạt khi |
| --- | --- | --- |
| **Bảo vệ đồ án** | Hệ thống chạy trọn mọi luồng trên dữ liệu seed, có số đo E1–E6, có test tải | Xong P10 |
| **Thí điểm thật với một lớp** | Người thật, dữ liệu thật, không có bạn ngồi cạnh | Xong phase **PR** + trường đồng ý các mục 1 và 2 dưới đây |

Tài liệu này không phải tư vấn pháp lý. Mục 1 liệt kê những việc phải làm cùng bộ phận pháp chế và CNTT của trường.

## 1. Pháp lý và dữ liệu cá nhân

Luật Bảo vệ dữ liệu cá nhân số 91/2025/QH15 và Nghị định 356/2025/NĐ-CP có hiệu lực từ 01/01/2026, thay Nghị định 13/2023. Những điểm chạm trực tiếp vào EduPilot:

| Việc | Ai làm | Hệ thống hỗ trợ bằng |
| --- | --- | --- |
| Xác định vai: **trường là bên kiểm soát dữ liệu**; người vận hành EduPilot là bên xử lý | Trường + bạn | Tài liệu mô tả luồng dữ liệu (`SYSTEM_DESIGN.md`, `FLOWS.md`) |
| Hồ sơ đánh giá tác động xử lý dữ liệu cá nhân (gửi cơ quan chuyên trách trong 60 ngày từ khi bắt đầu xử lý) | Trường | Bảng "dữ liệu nào, để làm gì, giữ bao lâu" ở mục 1.1 |
| **Gọi LLM đặt ở nước ngoài = dùng nền tảng ngoài lãnh thổ để xử lý dữ liệu thu thập tại Việt Nam = chuyển dữ liệu xuyên biên giới** → cần hồ sơ đánh giá tác động chuyển dữ liệu xuyên biên giới | Trường | (a) Che tên / MSSV trước mọi lời gọi LLM giảm mạnh lượng dữ liệu định danh rời hệ thống; (b) **lối thoát kỹ thuật:** cấu hình provider "OpenAI-compatible" trỏ vào model tự host hoặc đặt trong nước → không còn chuyển xuyên biên giới |
| Thông báo và sự đồng ý của sinh viên | Trường duyệt nội dung | Màn đồng ý lần đăng nhập đầu, lưu phiên bản + thời điểm; không đồng ý phần tuỳ chọn (theo dõi thời gian học) vẫn dùng được phần còn lại |
| Quyền của chủ thể dữ liệu: xem, sửa, xuất, xoá | Trường tiếp nhận | `/settings/privacy`: xuất dữ liệu của tôi (ZIP), yêu cầu xoá → việc cho Admin; xoá = ẩn danh hoá bản ghi học vụ phải giữ, xoá hẳn phần còn lại |
| Thông báo sự cố lộ, mất dữ liệu | Trường | Runbook sự cố (mục 3), `audit_log`, nhật ký truy cập prompt |

### 1.1. Dữ liệu nào, để làm gì, giữ bao lâu (mặc định, trường chỉnh được)

| Dữ liệu | Mục đích | Ai thấy | Giữ |
| --- | --- | --- | --- |
| Tài khoản (tên, email, MSSV) | Định danh, phân quyền | Bản thân, giảng viên lớp, Admin | Đến khi yêu cầu xoá hoặc 24 tháng không hoạt động |
| Hội thoại chat riêng | Trả lời sinh viên | Bản thân; giảng viên CHỈ khi escalate; Admin khi điều tra sự cố (có ghi audit) | 12 tháng sau khi lớp lưu trữ |
| Câu hỏi đã ẩn danh | Báo cáo lỗ hổng kiến thức | Giảng viên lớp (dạng gộp) | Cùng hội thoại |
| Điểm, điểm danh, điểm cộng, bài nộp | Học vụ | Bản thân, giảng viên / TA lớp | Theo quy định lưu trữ hồ sơ của trường |
| Ghi chú quan sát của giảng viên | Hỗ trợ giảng dạy | Giảng viên / TA lớp | Đến khi lớp lưu trữ + 12 tháng |
| Thời gian học on-screen | Tín hiệu cần chú ý, KHÔNG tính điểm | Bản thân, giảng viên lớp | Thô 7 ngày; tổng hợp theo ngày đến khi lớp lưu trữ |
| `llm_audit` (prompt đã che) | Vận hành, chi phí, điều tra | Admin (có ghi audit) | 6 tháng |
| `audit_log` | Truy vết | Admin | 24 tháng |

## 2. Những điều phải thống nhất với trường trước khi thí điểm

- [ ] Hạ tầng: máy chủ của trường hay VPS; tên miền; ai giữ quyền root; ai trực khi hỏng.
- [ ] Mail: dùng SMTP relay của trường (tốt nhất) hoặc dịch vụ gửi mail có SPF / DKIM; hộp thư nhận bài riêng cho từng lớp hay chung.
- [ ] Đăng nhập: mật khẩu riêng hay tài khoản Microsoft của trường (cần IT bật ứng dụng OIDC).
- [ ] LLM: nhà cung cấp nào được phép; ngân sách tháng; hay bắt buộc model tự host.
- [ ] Vị thế của điểm: EduPilot là công cụ hỗ trợ; điểm chính thức nhập vào hệ thống quản lý đào tạo như cũ.
- [ ] Chính sách khi AI trả lời sai, khi sinh viên khiếu nại điểm (đã có F11), khi có tin nhắn khủng hoảng (F3).
- [ ] Nội dung: thông báo quyền riêng tư, điều khoản sử dụng, tuyên bố giới hạn của AI — do trường duyệt.

## 3. Danh sách kỹ thuật của phase PR

| Nhóm | Hạng mục | Kiểm bằng |
| --- | --- | --- |
| Bảo mật | Rà theo OWASP ASVS mức 1; header bảo mật (CSP, HSTS, X-Content-Type-Options, Referrer-Policy) ở Caddy; cookie `Secure`; quét phụ thuộc (`govulncheck`, `pip-audit`, `pnpm audit`) trong CI; secret qua biến môi trường / file secret của Docker, xoay được | Checklist + CI |
| File tải lên | Danh sách trắng loại file theo magic bytes; giới hạn kích thước; quét ClamAV trong `ingest.jobs` và khi nhận bài nộp; phục vụ file bằng `Content-Disposition: attachment` từ tên miền không cookie | Test tải file EICAR |
| Sao lưu | `pg_dump` hằng đêm + WAL nếu có thể; sao lưu object storage; mã hoá bản sao lưu; giữ 7 ngày / 4 tuần / 3 tháng; **diễn tập khôi phục** ghi lại thời gian | `make restore-drill` |
| Giám sát | Uptime ngoài (kiểm `/healthz` mỗi phút); cảnh báo qua mail khi: 5xx > 1%, p95 > 2 s, hàng đợi > ngưỡng, dead-letter > 0, đĩa > 80%, sao lưu thất bại, sắp chạm trần ngân sách LLM | `/admin/health` + thử tắt từng dịch vụ |
| Vận hành | `docs/RUNBOOK.md`: triển khai, quay lui, khôi phục, xoay secret, xử lý sự cố lộ dữ liệu, LLM chết, mail chết; môi trường **staging** giống production; migration chạy trên staging trước | Diễn tập một lần mỗi kịch bản |
| Vòng đời dữ liệu | Lưu trữ lớp, xuất dữ liệu lớp, nhân bản sang kỳ mới, job xoá theo thời hạn, xuất / xoá dữ liệu cá nhân, màn đồng ý có phiên bản | E2E `semester-end.spec.ts`, `privacy-rights.spec.ts` |
| Đăng nhập trường (tuỳ chọn) | OIDC với Microsoft Entra ID, chỉ `openid profile email`, sau cờ `OIDC_ENABLED`; nối tài khoản theo email đã xác minh | Test với ứng dụng OIDC giả lập |
| Quản trị | `/admin/audit` (lọc theo người, thực thể, thời gian), `/admin/health` | – |
| Tải | Chạy lại `make load-t1` trên hạ tầng giống thật | Báo cáo SLO |
| Tài liệu người dùng | Hướng dẫn 1 trang cho sinh viên, 2 trang cho giảng viên, video 3 phút; trang `/help` trong app; kênh hỗ trợ (mail / nhóm) | Cho 2 người ngoài dùng thử không kèm hướng dẫn miệng |

## 4. Kế hoạch thí điểm

| Giai đoạn | Thời lượng | Việc | Tiêu chí đi tiếp |
| --- | --- | --- | --- |
| 0. Chạy bóng | 1 tuần | Bạn + giảng viên dùng trên dữ liệu lớp thật đã ẩn danh, sinh viên chưa vào | Không lỗi chặn; giảng viên làm được 5 việc chính không cần hỏi |
| 1. Một lớp, tính năng an toàn | 3–4 tuần | Hỏi đáp hai kênh, escalation, tài liệu, lịch, điểm danh. CHƯA bật chấm tự động, CHƯA công bố điểm qua hệ thống | ≥ 60% sinh viên dùng hằng tuần; ≥ 70% câu trả lời được đánh giá Hữu ích; 0 sự cố lộ dữ liệu; thời gian chờ ticket trung vị < 24 h |
| 2. Bật chấm bài có người duyệt | 3–4 tuần | Một bài tập: AI chấm nháp, giảng viên duyệt 100%, so với chấm tay | QWK ≥ 0,7; giảng viên tiết kiệm ≥ 30% thời gian; phúc khảo ≤ 10% |
| 3. Sổ điểm song song | Cuối kỳ | Tính điểm trên EduPilot VÀ theo cách cũ, đối chiếu | Khớp 100% |
| 4. Mở rộng | Kỳ sau | Thêm lớp, thêm giảng viên | Theo số liệu giai đoạn 1–3 |

Công tắc an toàn ở mọi giai đoạn: Admin tắt được từng tính năng AI theo lớp (chat, tự trả lời Threads, chấm bài) mà không phải triển khai lại; khi tắt, hệ thống vẫn chạy như một công cụ quản lý lớp thông thường.

## 5. Nếu cần thí điểm sớm hơn

Đường ngắn nhất tới một thí điểm thật có giá trị (giai đoạn 1 ở trên): **P0 → PG → PU → P1 → P2 → P3 → P4 → P5 → PR rút gọn** (bảo mật, sao lưu, giám sát, đồng ý, runbook; bỏ OIDC, quét file có thể tạm thay bằng danh sách trắng loại file). Khoảng 16 tuần. P6–P10 làm tiếp trong lúc thí điểm đang chạy, và dữ liệu thật từ thí điểm làm chương thực nghiệm của luận văn thuyết phục hơn hẳn dữ liệu seed.

# Sprint 9.1: khảo sát và cải thiện UI/UX (chủ dự án yêu cầu 2026-11-08)

**Nhánh:** `sprint/9.1-uiux`, tách từ `ops/vps-deploy` (đã có trang giới thiệu bản đầu, bản demo VPS). Làm xong trước sprint 9.5.

**Hai pha, có điểm duyệt ở giữa:**
1. **Khảo sát → báo cáo.** Đi toàn bộ màn hình trên bản demo thật, chấm theo `DESIGN.md`, `UX.md`, luật chữ UI của chủ dự án và các skill thiết kế. Ra **báo cáo cải thiện UI/UX** (HTML, có ảnh trước, mẫu sau cho các màn chính, danh sách việc có mức ưu tiên). Chủ dự án duyệt.
2. **Cải thiện theo mẫu đã duyệt.** BA viết spec / AC từ báo cáo; dev làm theo mẫu; QC kiểm; ảnh mốc cập nhật.

## Phạm vi khảo sát
- **Mọi route trong `frontend/src/app`:**
  - trang giới thiệu, đăng nhập, đăng ký, quên / đặt lại mật khẩu, xác minh, mời;
  - toàn bộ `(app)/*` (Hôm nay, chat, threads, tài liệu, thư viện, lịch, bài thi, luyện đề, bài tập, sổ điểm, chấm bài, inbox, điểm danh, sinh viên, hồ sơ, kết quả của tôi, insights, analytics, quan sát AI, cài đặt, admin, lớp, tham gia, ngân hàng câu hỏi).
- **4 vai** (giảng viên, trợ giảng, quản trị, sinh viên) × **1440 px và 375 px** (sinh viên, `/attendance`, `/inbox` bắt buộc 375).
- **Trạng thái:** đang tải, rỗng, lỗi, đầy dữ liệu, xung đột 409, mất mạng.
- **Chuyển động:** khi nào có, khi nào nên có, `prefers-reduced-motion`.

## Trang giới thiệu (chủ dự án 2026-11-08: "chưa ổn, cần hiện đại hơn, có animation và video giới thiệu trực quan")
- Thiết kế lại theo chế độ *Persuade*: hiện đại, có chủ ý, chuyển động có mục đích (xuất hiện khi cuộn, sơ đồ luồng "AI soạn nháp → giảng viên duyệt → mã tính" động, số đo đếm lên…), vẫn cùng họ chữ và token `--ep-*`.
- **Video giới thiệu sản phẩm** khoảng 60–90 s:
  - quay màn hình bản demo thật bằng Playwright trên VPS, theo kịch bản rút gọn từ `DEMO_SCRIPT.md`;
  - dựng bằng ffmpeg: phụ đề tiếng Việt in vào hình, có ảnh bìa (poster), không âm thanh hoặc nhạc nền nhẹ không bản quyền;
  - xuất `mp4` (H.264) + `webm`, nén < 8 MB.

  Trên trang: video bật khi người xem bấm, có poster; không tự phát kèm tiếng.
- **Chuyển động** chỉ bằng CSS / Web Animations API + `IntersectionObserver`; không thêm thư viện ngoài bảng `ARCHITECTURE.md` (cần thì hỏi chủ dự án).
- **Giữ ngân sách:** LCP ≤ 2,5 s, JS ≤ 256 KB; video không tải trước khi bấm (`preload="none"`).
- **Tôn trọng `prefers-reduced-motion`**, axe 0 lỗi.

## Skill dùng
`impeccable` (critique / audit / polish / animate), `design-taste-frontend`, `emil-design-eng`, `animate`, `review-animations`, `improve-animations`, `design` (ui-ux-pro-max). Đọc `skill://<tên>` trước khi dùng.

## Ai làm gì
| Vai | Pha 1 | Pha 2 |
| --- | --- | --- |
| qc | Chụp bằng chứng: mọi màn × vai × cỡ, trạng thái, lỗi console, axe → `audit/shots/`, `audit/evidence.md` | Kiểm theo AC + ảnh mốc |
| ba | Khảo sát chuyên môn (heuristic + skill) từ bằng chứng của QC + tự đi demo → `audit/findings.md` (mỗi vấn đề: màn, mức, bằng chứng, đề xuất) | Spec `docs/specs/FEAT-uiux-91/` từ báo cáo đã duyệt |
| dev | Sửa CI `ops/vps-deploy` (ngân sách JS); dựng **mẫu** (mockup HTML tĩnh hoặc nhánh mẫu) cho trang giới thiệu mới + 3–4 màn chính theo đề xuất; quay video thử | Làm theo mẫu đã duyệt |
| research | Rà khả thi kỹ thuật các đề xuất (hiệu năng, a11y, ngân sách JS, video) | Trả lời câu hỏi kỹ thuật |
| pm | Tổng hợp **báo cáo HTML** cho chủ dự án | Duyệt, báo cáo sprint |

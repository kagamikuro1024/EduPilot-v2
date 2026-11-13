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

## Hướng chuyển động (chủ dự án chốt 2026-11-08)
- **Trang giới thiệu là nơi đầu tư chuyển động:** hiện đại, tạo cảm giác muốn dùng ngay.
  - Hiệu ứng mở đầu có chủ ý, xuất hiện theo nhịp khi cuộn.
  - Sơ đồ luồng sống động; ảnh / khung sản phẩm có chiều sâu và chuyển động nhẹ theo cuộn.
  - Số liệu đếm lên; nút chính có phản hồi tinh tế.
  - Video giới thiệu.

  Mặc định dùng CSS (scroll-driven animations, `@starting-style`, view transitions) + WAAPI. Nếu một hiệu ứng thật sự cần thư viện, chỉ được dùng **cho riêng trang giới thiệu**, nạp lười (không vào bundle của ứng dụng), nhỏ, và phải mở proposal để PM / chủ dự án duyệt trước.
- **Ứng dụng (sau đăng nhập) không thêm thư viện chuyển động nào.** Chỉ chuyển động chức năng, ngắn (≤ 200 ms):
  - phản hồi bấm, mở / đóng drawer, dialog, popover;
  - chuyển trạng thái, xác nhận lưu.

  Theo `emil-design-eng`. Không trang trí, không làm chậm thao tác lặp lại.

## Quyết định của chủ dự án (2026-11-08, "Đồng ý hết")
| # | Quyết định |
| --- | --- |
| Q-UX91-1 | Nhãn điều hướng = tiêu đề trang, tiếng Việt: `Threads` → **Hỏi đáp công khai**, `Insights` → **Lỗ hổng kiến thức**, `Analytics` → **Số liệu lớp học**. Sửa `DESIGN.md` §2 + ảnh mốc. |
| Q-UX91-2 | Hôm nay của giảng viên chia nhóm **Gấp / Hôm nay / Khi rảnh**. Gấp = quá 72 giờ làm việc hoặc có cờ rủi ro; Khi rảnh = phát hiện giống bài, bài chưa khớp. |
| Q-UX91-3 | Cột "Trạng thái" sổ điểm: "Đủ" = đã có đủ điểm thành phần → nhãn **Đủ điểm / Thiếu điểm** (bỏ cột nếu mọi hàng đều Đủ điểm là tuỳ BA đề xuất trong spec). |
| Q-UX91-4 | Điểm danh: mặc định "Có mặt" dùng chấm mực; đỏ chỉ cho Vắng / Vắng phép / Muộn. |
| Q-UX91-5 | Trang giới thiệu được nêu số đo từ môi trường mô phỏng, kèm dòng chú thích "đo trên dữ liệu mô phỏng". |
| [ĐỔI DESIGN] trang giới thiệu | Thêm mục "Trang giới thiệu (Persuade)" vào `DESIGN.md`: cho phép chuyển động cuộn / sơ đồ / số đếm; §11 ("không dàn dựng khi tải trang") chỉ áp dụng cho ứng dụng sau đăng nhập. |

## Quyết định của PM cho các câu TL-2 (2026-11-08)
- **UX-16:** không làm tròn khi hiển thị (P-91-1).
- **UX-5, "Gấp":** dùng `urgency == "overdue"` của backend. Ngưỡng riêng từng nguồn việc đã định ở F14; "quá 72 giờ làm việc" là ngưỡng của các nguồn đó, không đặt ngưỡng mới ở client. Có thêm cờ rủi ro thì cũng vào nhóm Gấp.
- **UX-15, highlight trích dẫn:** chuyển động ≤ 200 ms theo luật ứng dụng. Nền highlight được **giữ** khoảng 1 s rồi mờ đi trong 200 ms; khoảng giữ là trạng thái, không phải chuyển động.
- **UX-25(a), lớp mặc định:** cách S. Nhớ lớp đã chọn; lần đầu chọn lớp đầu tiên **có dữ liệu** theo thứ tự API. "Lớp có việc chờ" để Nợ.
- **TL-2 đề xuất 4, 5:** chấp nhận (P-91-1, P-91-2). Gộp việc pha 2 theo **6 đợt** của TL-2.

## Chủ dự án duyệt báo cáo (2026-11-08, "Duyệt hết, bắt đầu làm luôn")
Duyệt cả 3 mục:
1. Báo cáo `bao-cao-uiux.html` → pha 2 theo 6 đợt.
2. Trang giới thiệu v2:
   - clip sản phẩm lặp 6–8 s không tiếng ở màn đầu (poster trước, giữ LCP), video đầy đủ mở bằng "Xem video";
   - kể chuyện theo cuộn: khung sản phẩm ghim, đổi Chat → Duyệt bài → Sổ điểm;
   - chữ lớn, khoảng thở; khung có chiều sâu, nổi khi rê chuột (máy tính).
3. Nút **"Dùng thử"** trên trang giới thiệu, đăng nhập sẵn vai sinh viên / giảng viên.
   - **Ràng buộc an toàn (PM):** chỉ bật khi `DEMO_MODE=true`.
   - Dùng **tài khoản và lớp sandbox riêng**, không phải tài khoản / lớp dùng để trình diễn, để khách không làm hỏng dữ liệu buổi bảo vệ.
   - Dữ liệu sandbox được **đặt lại định kỳ bằng job seed** (không `down -v`, không đụng lớp trình diễn).
   - Có giới hạn tần suất; không lộ mật khẩu ra HTML.
   - Tech Lead chốt thiết kế (TL-3) trước khi dev làm.

## Pha 2: thứ tự làm
Dev làm tuần tự: **đợt 1** (nền chung) → **đợt 2** (chữ) → **đợt 6** (trang giới thiệu v2 + Dùng thử) → **đợt 3** (Hôm nay + Hộp thư) → **đợt 4** (Chấm bài, Sổ điểm, Điểm danh) → **đợt 5** (Chat, Hỏi đáp, `/me`, Tài liệu).

Mỗi đợt gồm:
- `docs/sprints/9.1/handoff/dev-wave-<n>.md` (ảnh trước / sau 1440 + 375);
- QC kiểm theo TC → `docs/sprints/9.1/qc/wave-<n>.md` (PASS / FAIL);
- đợt PASS thì dev deploy frontend lên demo.

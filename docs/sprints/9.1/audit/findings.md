# Sprint 9.1: khảo sát UI/UX chuyên môn (BA, pha 1)

**Người viết:** `ba` · **Ngày:** 2026-11-08 · **Bản demo:** `https://v2202609419695521098.happysrv.de` (đọc nhánh `sprint/9.1-uiux`).

**Cách làm:**
- đi tay bản demo bằng trình duyệt Playwright (Mac), đo bằng `eval` trong trang;
- đọc mã nguồn CSS/TSX của `frontend/src` để đếm lỗi hệ thống;
- chấm theo `DESIGN.md` §0–§22, `UX.md`, luật chữ UI trong `~/.claude/CLAUDE.md` (09/10/2026), và skill `impeccable` (critique + audit), `design-taste-frontend`, `emil-design-eng`, `review-animations`.

**Đã đi (ảnh ở `audit/ba-shots/`, 34 ảnh):**

| Vai × cỡ | Route |
| --- | --- |
| Chưa đăng nhập, 1440 + 375 | `/`, `/login` |
| Giảng viên, 1440 | Hôm nay (lớp 761987 và 761988), `/inbox`, `/students`, `/attendance`, `/gradebook`, `/gradebook/scheme`, `/grading`, `/grading/[id]`, `/threads`, `/documents`, `/calendar`, `/insights`, `/analytics`, `/observability`, `/exams`, `/questions`, bộ chọn lớp |
| Giảng viên, 375 | `/attendance`, `/inbox` |
| Sinh viên, 1440 | Hôm nay, `/chat` |
| Sinh viên, 375 | Hôm nay, `/chat`, `/threads`, `/me`, `/practice` |

**Chưa đi (chờ bằng chứng của QC):**
- vai quản trị và trợ giảng;
- `/settings/*`, `/admin`, `/class`, `/join`, `/register`, `/forgot-password`;
- trạng thái xung đột 409, mất mạng, đang tải chậm, hộp thoại / drawer;
- axe, LCP đo thật, lỗi console từng route;
- video chuyển động (QC đang quay).

Mọi điểm trong báo cáo ghi rõ nguồn: ảnh BA, mã nguồn, hoặc "cần QC xác minh". Không đề xuất nào đã thử trên bản chạy.

---

## 0. Đính chính sau TL-2 và quyết định của PM (2026-11-08)

Báo cáo này là bản khảo sát gốc. Các điểm dưới đây **thay** nội dung cũ trong các mục 2–5; spec `docs/specs/FEAT-uiux-91/` là bản có hiệu lực.

| Mục | Đính chính | Nguồn |
| --- | --- | --- |
| UX-5 | Khối "Lớp cần chú ý" đã có trong mã (dữ liệu = sinh viên AT_RISK), chỉ nằm dưới danh sách; chuyển sang cột phải ≥ 1440. Hành động hàng dùng `ghost` + mũi tên, **không** thêm [ĐỔI DESIGN] nút `link`. "Gấp" = `urgency overdue` + `high` | TL-2, P-91-2, Q-UX91-8 |
| UX-18 | Sai: 960 px là đúng DESIGN §2. Chỉ còn việc dùng khoảng trống bên phải của Hôm nay | PM |
| UX-25 | Chỉ làm (a) theo cách S (nhớ lớp; lần đầu chọn lớp đầu tiên có dữ liệu). (b), (c) ghi Nợ | PM, Q-UX91-9 |
| UX-16 | Không làm tròn khi hiển thị (đệm `0` tới số chữ số lẻ lớn nhất của cột), giữ ảo hoá, một vùng cuộn cao bằng khung nhìn; **không** "hai chữ số thập phân", **không** "cuộn theo trang" | P-91-1 |
| UX-1, UX-3(b) | Hai cột vai, mỗi vai 4 mục, **không** tab; không dự phòng `IntersectionObserver` cho `view()` (chỉ một đoạn script nội tuyến ≤ 0,5 KB cho số đếm và sơ đồ chạy một lần) | P-91-2, TL-2 |
| UX-3(d) | Làm: không gọi `/auth/refresh` ở `/` khi chưa có cờ "đã từng đăng nhập" | TL-2, Q-UX91-10 |
| UX-2 | Chữ số liệu phải khớp nguồn: không "mọi lời gọi"; nêu recall 0,960 và "tên và MSSV trong danh sách lớp" | Q-UX91-11 |
| UX-10 | Không thêm phép tự động bắt Panel lồng; QC kiểm bằng `panels.spec` NEST | P-91-2 |
| UX-11 | Bỏ gợi ý "chủ đề sai nhiều" (Nợ); giữ gợi ý từ hạn bài / lịch | P-91-2 |
| UX-15 | Highlight giữ ≈ 1 s rồi mờ 200 ms (không 600 ms) | PM |

---

## 1. Điểm tổng quan

Thang 1–5 (5 = xuất sắc). Điểm của **ứng dụng sau đăng nhập**; trang giới thiệu chấm riêng ở dòng cuối.

| Mục | Điểm | Lý do |
| --- | :---: | --- |
| Nhất quán hệ thiết kế | 3 | Token, chữ, primitive dùng đồng đều. Lệch ở chỗ chi tiết: 4 kiểu ngày giờ, nhãn điều hướng Anh–Việt lẫn, khung lồng khung ở Insights / Quan sát AI, chiều rộng nội dung đổi giữa các route (UX-8, 9, 10, 18). |
| Phân cấp thị giác | 3 | Tiêu đề và khung rõ. Nhưng Hôm nay giảng viên có 9 hàng cùng trọng số, điểm danh chấm đỏ 24 hàng, chấm đỏ ở mọi hàng hộp thư: đỏ mất nghĩa "cần làm" (UX-5, 6, 7). |
| Mật độ thông tin | 3 | Bảng gọn, nhiều khoảng thở. Bị kéo xuống bởi chữ lặp: "lớp 761987" ở mỗi hàng, "+ phát biểu · +0,25" 24 lần, "Chia sẻ từ 761987" 5 lần (UX-4, 5, 7, 19). |
| Lời văn | 2 | Vi phạm luật chữ UI: 55 chỗ `description=` (PageHeader, Section, Drawer, Switch) trong 34 tệp `features/*`, 12 chỗ là câu giải thích dài > 60 ký tự dưới tiêu đề (UX-4). Phần còn lại tiếng Việt tốt, nút là động từ. |
| Trạng thái | 3 | Có tải / rỗng / lỗi. Trạng thái rỗng thường thiếu "vì sao rỗng"; nút khoá không nói lý do (UX-13, 22). |
| Di động | 4 | Không tràn ngang ở mọi route đã đo (375 px, `scrollWidth = 375`). Hôm nay sinh viên là mẫu tốt. Điểm trừ: nút Lưu điểm danh không dính đáy; ảnh sản phẩm ở trang giới thiệu quá nhỏ để đọc (UX-3, 7). |
| A11y | 3 | Có focus ring, nhãn, `prefers-reduced-motion`. Chưa đo axe; trạng thái hàng Hôm nay phân biệt chỉ bằng màu chấm (UX-6). **Cần QC xác minh.** |
| Chuyển động | 2 | Không có :active, hover không gắn `@media (hover: hover)`, transition `width` / `margin-left`, tắt chuyển động bằng 1 ms thay vì giảm bớt (UX-26). Trang giới thiệu: 0 animation. |
| Cảm giác hiện đại / đáng tin | 3 | Ứng dụng nghiêm, sạch, có chủ đích; cảm giác "công cụ học thuật" đúng định vị. Trang giới thiệu chưa đủ sức thuyết phục: tĩnh, ảnh nhỏ, không video (UX-1, 2, 3). |

**Trang giới thiệu (chế độ Persuade): 2/5.** Chữ đúng giọng, bố cục sạch, nhưng không có chuyển động, không video, ảnh chụp bị cắt và quá nhỏ.

### Điểm Nielsen (10 heuristic, ứng dụng sau đăng nhập)

| # | Heuristic | Điểm | Vấn đề chính |
| --- | --- | :---: | --- |
| 1 | Hiển thị trạng thái hệ thống | 3 | Có trạng thái lưu, tiến độ. Nút khoá không nói lý do. |
| 2 | Khớp thế giới thực | 3 | Tiếng Việt gần gũi. "Đủ" (trạng thái sổ điểm), "Insights", "Analytics" khó hiểu. |
| 3 | Kiểm soát và tự do | 3 | Có hoàn tác, quay lại. Chưa thấy lối thoát ở trạng thái rỗng. |
| 4 | Nhất quán và chuẩn | 2 | Ngày giờ 4 kiểu; nhãn Anh–Việt; chiều rộng nội dung không đồng đều. |
| 5 | Ngăn lỗi | 3 | Lớp bảo vệ tốt (xác nhận không hoàn tác, tự lưu). |
| 6 | Nhận ra hơn nhớ | 3 | Hai lớp trùng tên "An ninh mạng", chỉ phân biệt bằng mã 6 số. |
| 7 | Linh hoạt và hiệu quả | 3 | Có ⌘K, phím 1–4, J/K. Thiếu gợi ý phím tắt hiện trên màn duyệt bài (chưa xác nhận). |
| 8 | Tối giản thẩm mỹ | 2 | Chữ lặp, giải thích dư, khung lồng khung. |
| 9 | Phục hồi lỗi | 3 | Lỗi đăng nhập rõ. Chưa thử các lỗi khác. |
| 10 | Trợ giúp | 2 | Không có trợ giúp theo ngữ cảnh; theo luật chữ UI thì đúng, chỉ cần tooltip cho điểm dễ làm sai. |
| | **Tổng** | **27/40** | **Chấp nhận được, gần mức Tốt (68%)** |

Trang giới thiệu: heuristic 7 và 10 là `n/a` (Persuade). 8 heuristic còn lại được **19/32 (59%)**: nhất quán và minh bạch tốt, thiếu phản hồi chuyển động, thiếu bằng chứng trực quan.

### Điểm mạnh nên giữ

- Hôm nay **sinh viên**: một việc kế tiếp có lý do và thời gian ước lượng, một nút đỏ (`hom-nay__sv__375.png`).
- Điểm danh mobile: hàng ngăn bằng đường kẻ, không thẻ, chữ trạng thái "Có mặt" kèm chấm (`attendance__gv__375.png`).
- Màn duyệt bài: hai cột đúng tỷ lệ §14.13, tổng điểm lớn, trích dẫn bài làm dưới tiêu chí (`grading-review__gv__1440.png`).
- Sổ điểm: ô sửa tại chỗ có viền, ô chỉ đọc phẳng, phân biệt được bằng mắt (`gradebook__gv-lop1__1440.png`).

---

## 2. Vấn đề

Cột **ƯT**: **N** = làm ngay (sửa nhỏ, ảnh hưởng lớn); **9.1** = làm trong sprint này; **S** = để sau.
Cột **Nguyên tắc**: trích điều khoản đã vi phạm.
Ảnh trong `audit/ba-shots/`.

| ID | Màn / route | Vai | Mức | Mô tả ngắn | Bằng chứng | Nguyên tắc | Đề xuất cụ thể | ƯT |
| --- | --- | --- | :---: | --- | --- | --- | --- | :---: |
| **UX-1** | Trang giới thiệu `/` | Chưa đăng nhập | **Cao** | Tĩnh hoàn toàn (`document.getAnimations()` = 0; `Landing.module.css` không có `transition` / `animation`). Hero chỉ có chữ + ảnh cắt ngang khung ứng dụng. Không video, không sơ đồ luồng, số liệu đứng yên. | `landing__anon__1440-top.png`, `…-full.png` | Plan 9.1 mục Trang giới thiệu; skill impeccable (Persuade); design-taste-frontend (mỗi trang có hero entrance + 1–2 khoảnh khắc) | Thiết kế lại theo mục 3.A. Hero chữ vẫn là LCP (không ẩn bằng `opacity: 0`). Khung sản phẩm + sơ đồ luồng "AI soạn nháp → giảng viên duyệt → mã tính" + số đếm lên + video bấm mới phát. | 9.1 |
| **UX-2** | Trang giới thiệu, khối "Khác biệt, có số đo" | Chưa đăng nhập | **Cao** | Ba số liệu khó hiểu với người ngoài: `0 / 146`, `30 / 30`, `+0,05 %`. Số thứ ba dễ đọc nhầm là "tăng 0,05 % người dùng". Nhà cung cấp AI là mô phỏng nên cần nói rõ nhưng đang nằm trong ngoặc cuối câu. | `landing__anon__1440-full.png` | impeccable: bằng chứng phải đọc được trong 3 giây; `emil`: số liệu minh hoạ phải gắn với cái nó đo | Mỗi số một câu đầu hiểu ngay ("146 yêu cầu gửi AI, 0 lần lộ tên"; "Chấm 22 bài cùng lúc, chat không chậm thêm"). Số đếm lên khi vào khung nhìn. Giữ dòng "mô phỏng" nhưng chuyển xuống dòng chú thích riêng. | 9.1 |
| **UX-3** | Trang giới thiệu, di động + chung | Chưa đăng nhập | TB | (a) Ảnh hero bị ẩn ở < 720 px, ảnh sản phẩm còn lại quá nhỏ để đọc (375 px). (b) Hai danh sách 6 mục × 2 vai (12 dòng chữ) thay vì chọn vai. (c) Không CTA lặp lại ở cuối trang. (d) Mọi lần vào trang công khai đều có lỗi console `401 /auth/refresh`. | `landing__anon__375-full.png`, `console log` (QC sẽ ghi) | `UX.md` §2 quy tắc 5; DESIGN §22.1 (3 giây); luật ≤ 4 lựa chọn (cognitive load) | (a) Dùng khung điện thoại dựng bằng CSS cho mobile, hoặc cắt ảnh vào đúng vùng cần xem. (b) Hai thẻ tab "Giảng viên / Sinh viên", mỗi vai 4 mục. (c) Thêm khối CTA cuối. (d) Không gọi `/auth/refresh` khi không có cờ "đã từng đăng nhập" (đã có `RETURNING_KEY`). | 9.1 |
| **UX-4** | Toàn app (34 tệp) | Mọi vai | **Cao** | Dòng giải thích dưới tiêu đề trang / khối: `grep 'description='` thấy 55 chỗ trong 34 tệp `features/*` (gồm cả Drawer / Switch, cần rà từng chỗ), 12 chỗ là chuỗi tĩnh dài > 60 ký tự. Ví dụ: Insights ("Câu hỏi của lớp, đã bỏ tên và mã số sinh viên, nhóm theo chủ đề để bạn quyết định dạy lại phần nào."), Observability, Analytics (4 khối × 1 câu), Practice, Exams, Question bank, Library, Threads, Chat. Insights còn thêm hộp thông báo riêng về riêng tư. | `insights__gv__1440.png`, `analytics__gv__1440.png`, `observability__gv__1440.png`, `exams__gv__1440.png`, `questions__gv__1440.png`, `practice__sv__375.png` | **Luật cứng chữ UI (CLAUDE.md 09/10/2026):** không câu mô tả dưới tiêu đề khối, câu thừa thì xoá | **Xoá** mô tả dưới tiêu đề trang / khối. Dòng chỉ còn là *dữ liệu* (số lượng, lớp, ngày) thì giữ ở dạng `meta` ngắn. Tooltip chỉ cho cú pháp / lý do khoá / hệ quả (ví dụ lý do khoá nút ở UX-13). Quy tắc kiểm tự động: `ui-antipatterns.sh` thêm phép bắt `description=` của `PageHeader` / `Section` có chuỗi tĩnh > 40 ký tự. Vá SRS + Confluence nếu có message trong SRS. | **N** |
| **UX-5** | `/` Hôm nay | Giảng viên | **Cao** | Lớp 761987: 9 hàng cùng cỡ chữ, cùng chấm, cùng nút "Mở" đỏ (×9 chữ đỏ). "lớp 761987" lặp ở 7/9 dòng và còn lặp ở phụ đề trang và thanh trên. Tiêu đề trang `9 việc cần xử lý hôm nay` rồi ngay dưới `Việc cần xử lý` (lặp). Khối "Lớp cần chú ý" (§14.1) **đã có trong mã** (`StaffToday.tsx`, dữ liệu `attention` = sinh viên AT_RISK) nhưng nằm dưới danh sách, ngoài khung nhìn đầu. Người dùng không biết bấm đâu trước. *(Đính chính 2026-11-08, TL-2 + PM: bản đầu ghi "không có khối Lớp cần chú ý".)* | `hom-nay__gv-lop1__1440.png` | DESIGN §14.1 (first viewport: ActionList xếp theo mức khẩn + "Lớp cần chú ý"); §22.1, §22.2; §4 (đỏ = cần hành động; 9 hành động đỏ = 0 tín hiệu) | Xem mẫu 3.B: một việc đầu tiên nổi bật (tiêu đề + 1 nút chính), phần còn lại gom theo 3 nhóm có nhãn chữ (Gấp / Hôm nay / Khi rảnh; Gấp = `urgency overdue` + `high`, tính ở client, không ngưỡng mới), hành động hàng dùng `Button` `ghost` + mũi tên (không thêm biến thể `link`, P-91-2), bỏ lặp mã lớp, chuyển khối "Lớp cần chú ý" (sinh viên AT_RISK) và "Sắp tới" sang cột phải ≥ 1440. | 9.1 |
| **UX-6** | Hôm nay, Hộp thư | Giảng viên | TB | Mức khẩn chỉ nằm ở màu chấm (đỏ / hổ phách / xám), không có chữ. Hộp thư lớp 761987: 5/5 hàng có chấm đỏ + nhãn "Đang chờ" giống hệt, chấm đỏ không phân biệt được hàng nào gấp. | `hom-nay__gv-lop1__1440.png`, `inbox__gv-lop1__1440.png` | DESIGN §16 (không mã hoá chỉ bằng màu); §4 (đỏ có nghĩa) | Chấm đỏ chỉ cho hàng quá hạn (> 72 giờ, như dữ liệu đã có); các hàng còn lại chấm xám. Thêm chữ ngắn "Quá 72 giờ" / "14 ngày" cạnh chấm (đang có tuổi ở Inbox, chưa có ở Hôm nay). Bỏ nhãn "Đang chờ" lặp ở mỗi hàng (đã là tab "Đang chờ"). | 9.1 |
| **UX-7** | `/attendance` | Giảng viên | **Cao** | Mặc định cả 24 hàng là chấm radio **đỏ** đặc ("Có mặt"): đỏ chiếm cả cột nhưng chỉ là trạng thái mặc định, không phải tín hiệu. "+ phát biểu · +0,25" lặp nguyên văn 24 lần. Nút `Lưu điểm danh` ở góc trên phải cách xa bảng; ở 375 px không dính đáy nên sau 30 hàng phải cuộn ngược lên. Cảnh báo "Công thức điểm chưa xác nhận" đúng §14.8 (một dòng mỏng) nhưng chiếm chỗ trong khung nhìn đầu di động. | `attendance__gv__1440.png`, `attendance__gv__375.png` | DESIGN §4 (đỏ = hành động / lỗi, không phải mặc định); UX.md §5 (`/attendance` một tay, nút chính hoàn tất buổi); DESIGN §21 (red decorative) | "Có mặt" dùng chấm mực (`--ep-ink`), **chỉ các lựa chọn khác mặc định** (Muộn, Vắng phép, Vắng) mới đỏ / hổ phách; nhãn "+ phát biểu" chỉ hiện biểu tượng + khi hover / focus hàng, số điểm xuất hiện sau khi thêm; di động: thanh dưới dính "Lưu điểm danh · 24 có mặt · 1 vắng" thay cho nút trên cùng. Giữ cảnh báo công thức, thu thành 1 dòng ở mobile. | 9.1 |
| **UX-8** | Toàn app | Mọi vai | TB | Ngày giờ có ít nhất 4 kiểu: `00:10 02-11` (Threads, Tài liệu, Chat), `Thứ 2, 2/11 · 00:13` (Hôm nay SV), `Thứ Năm, 29/10 · 00:10` (Chấm bài), `Thứ Hai, 00:07 02/11` (Bài thi). 13 tệp tự gọi `toLocale*String` / `Intl.DateTimeFormat`. | `threads__gv__1440.png`, `documents__gv__1440.png`, `hom-nay__sv__1440.png`, `grading__gv-lop1__1440.png`, `exams__gv__1440.png` | `UX.md` §2.10 (`Thứ Hai, 21/09 · 14:00`, múi giờ `Asia/Ho_Chi_Minh`) | Một hàm định dạng dùng chung ở `shared/lib` (ngày đầy đủ, ngày ngắn, tương đối), thay mọi chỗ gọi `toLocale*`. Quy ước: hôm nay → "Hôm nay, 14:00"; trong tuần → "Thứ Hai, 14:00"; xa hơn → "Thứ Hai, 21/09 · 14:00". | **N** |
| **UX-9** | Điều hướng | Giảng viên | TB | Nhãn điều hướng Anh–Việt lẫn: `Threads`, `Insights`, `Analytics`; tiêu đề trang khác nhãn nav: nav "Insights" → h1 "Lỗ hổng kiến thức", nav "Analytics" → h1 "Số liệu lớp học". Sinh viên thấy `Threads`. | `hom-nay__gv-lop1__1440.png`, `insights__gv__1440.png`, `analytics__gv__1440.png` | DESIGN §13 (tiếng Việt, đúng ngôn ngữ người dùng); heuristic 4 (nhất quán) | Hỏi chủ dự án (xem QUESTIONS Q-UX91-1): giữ "Threads" như tên riêng hay đổi thành "Hỏi đáp công khai". Với Insights / Analytics: nhãn nav = h1 ("Lỗ hổng kiến thức", "Số liệu lớp học"). **Nếu đổi: [ĐỔI DESIGN] §2 bảng điều hướng.** | 9.1 |
| **UX-10** | `/insights`, `/observability` | Giảng viên | TB | Khung lồng khung: Insights "Chủ đề nên dạy lại" là Panel chứa khung chủ đề có viền riêng. Quan sát AI: dải 5 chỉ số cùng cỡ ở đầu trang (KPI wall) rồi lặp lại cùng số (yêu cầu, độ trễ p95, tỷ lệ lỗi) ở bảng "Lớp 761987 trong 7 ngày"; hàng đầu tiên của bảng trống. Tiêu đề khối thứ hai dính sát Panel trên (~14 px). | `insights__gv__1440.png`, `observability__gv__1440.png` | DESIGN §21 (card trong card; 3+ KPI cùng cỡ ở đầu trang); §22.9 | Insights: bỏ khung của từng chủ đề, ngăn bằng đường kẻ 1 px trong cùng một Panel. Observability: giữ **một** nơi hiển thị mỗi chỉ số; số "hôm nay" thành dòng chữ (không dải KPI), 7 ngày thành bảng; sửa khoảng cách trên tiêu đề khối (§6: nhiều hơn dưới). `ui-antipatterns.sh` thêm phép bắt Panel chứa phần tử có `border` + `border-radius` riêng. | 9.1 |
| **UX-11** | `/chat` | Sinh viên | **Cao** | Desktop: hai Panel cạnh nhau (lịch sử + hội thoại) trong khi §14.2 yêu cầu cột trò chuyện ở giữa, lịch sử có thể thu gọn, không phải Panel thứ hai. Nút `Phiên mới` (hành động chính) hiện **nhạt như bị khoá**. Trạng thái đầu: ~600 px trống, chỉ một gợi ý ("Quy chế thi cuối kỳ nói gì về tài liệu?" – câu gượng) dạng nút. Lịch sử 6 mục cắt 4 dòng, 6 biểu tượng thùng rác luôn hiện. Di động: danh sách "Phiên trước (6)" chiếm một ô riêng trên đầu, đẩy ô nhập xuống. | `chat__sv__1440.png`, `chat__sv__375.png` | DESIGN §14.2; §21 (nhiều Panel trong một vùng làm việc); §12 (một hành động chính rõ ràng); UX.md §2.5 (không ngõ cụt) | Xem mẫu 3.C: một Panel hội thoại căn giữa tối đa 840 px; lịch sử là cột phẳng bên trái (đường kẻ, không Panel) thu gọn được, xoá trong menu `⋯` mỗi hàng; `Phiên mới` ẩn khi phiên hiện tại còn trống (không hiện nút khoá); 3 gợi ý lấy từ ngữ cảnh sinh viên (hạn bài sắp tới, chủ đề sai nhiều) dạng hàng chữ. | 9.1 |
| **UX-12** | Hộp thư | Giảng viên | TB | Danh sách rỗng nhưng vùng phải vẫn viết "Chọn một câu hỏi để xem" (mâu thuẫn). Câu rỗng "Không có câu hỏi nào đang chờ." không nói vì sao, kèm liên kết đỏ lẻ "Xem tất cả" thụt lề lệch. Chi tiết: "Đã chờ 14 ngày" và "Trạng thái Đang chờ" lặp thông tin của hàng đã chọn. | `inbox__gv-lop2-rong__1440.png`, `inbox__gv-lop1__1440.png`, `inbox__gv-lop2__375.png` | UX.md §2.5; DESIGN §15 (rỗng dạy việc kế tiếp) | Rỗng: một dòng "Chưa có câu hỏi nào cần thầy/cô xử lý. Khi AI không đủ chắc chắn hoặc sinh viên nhờ hỗ trợ, câu sẽ hiện ở đây." (đúng mẫu §15), ẩn vùng chi tiết. Bỏ hai hàng lặp ở chi tiết, gộp "14 ngày" vào tiêu đề chi tiết. | N |
| **UX-13** | Nhiều màn | Giảng viên | TB | Nút chính **bị khoá** trông như nút chính nhạt, không nói lý do: `Chốt điểm` (sổ điểm lớp 2), `Chấm tất cả` + `Tải bài lên` (Chấm bài), `Xác nhận công thức`, `Phiên mới`. | `gradebook__gv-lop2-rong__1440.png`, `grading__gv-lop1__1440.png`, `gradebook-scheme__gv__1440.png` | DESIGN §14.10 (`Chốt điểm` khoá, lý do hiện khi focus / hover); luật CLAUDE.md "tooltip cho lý do nút bị khoá"; UX.md §2.5 | Nút khoá giữ nguyên nhãn, thêm lý do ngắn bằng tooltip + `aria-describedby` ("Chưa có bài nộp", "Cần xác nhận công thức trước"). Cùng một cơ chế trong primitive `Button` (`disabledReason`), không viết riêng từng màn. | **N** |
| **UX-14** | `/grading` | Giảng viên | TB | Bộ lọc thô: hai select rất rộng, chip "Cần xem kỹ" lệch dòng với select; cột "Bài tập" lặp một tiêu đề 3 dòng cho cả 4 hàng; "Nguồn: Trong EduPilot" lặp; hàng không có dấu hiệu bấm được (không nút "Duyệt", không mũi tên). | `grading__gv-lop1__1440.png` | DESIGN §14.12; §12 (hàng bấm được phải có hover / focus rõ); heuristic 8 | Khi chỉ có một bài tập thì ẩn cột "Bài tập" và dựng tên bài thành tiêu đề bảng; hàng có chữ "Duyệt" ở cuối (liên kết trung tính) + hover nền; chip lọc thẳng hàng với select; "Lý do cờ" lên đầu hàng nếu có (đang ở cuối). | 9.1 |
| **UX-15** | `/grading/[id]` | Giảng viên | TB | Mỗi tiêu chí lặp tên hai lần (nhãn đậm trái + "Điểm <tên tiêu chí>" bên phải) và hai dòng trên đầu ô nhập chồng lên nhau ở 1440; dòng nhắc "Đoạn trích từ bài làm · bấm để xem trong bài" lặp ở **mỗi** trích dẫn; nút `Duyệt bài` và thanh "Đã duyệt 0/22" bị cắt ngay mép dưới khung nhìn 1440 × 900 (cần QC xác minh có dính đáy thật hay không). Không thấy gợi ý phím `A` trên giao diện. | `grading-review__gv__1440.png` | DESIGN §14.13; UX.md §5 (`/grading/[id]`: `A` duyệt, thanh tiến độ); luật chữ UI (câu lặp) | Mẫu 3.D: mỗi tiêu chí một hàng: tên + ô điểm + `/2,5` ở cùng dòng; trích dẫn là dòng trích có biểu tượng nhỏ (nhắc một lần ở đầu cột); thanh hành động dính đáy cột phải luôn thấy: `Duyệt bài` + tiến độ + gợi ý phím. | 9.1 |
| **UX-16** | `/gradebook` | Giảng viên | TB | Số không căn thập phân: `0,5` / `0` / `7,915` / `8,25` lẫn nhau, không `tabular-nums` thấy được; tiêu đề cột hai dòng với dòng phụ "Đã công bố" + `⋯` làm chiều cao cột lệch; bảng cuộn **trong** Panel cuộn trong trang (cắt cứng dòng cuối); cột "Trạng thái: Đủ" khó hiểu; ghi chú cuối bảng nhỏ. | `gradebook__gv-lop1__1440.png` | DESIGN §5 (`tabular-nums`); §14.10 (đóng băng cột tên + tổng); UX.md §2 quy tắc 4, §5 (`/gradebook`) | Mẫu 3.E: số cố định 2 chữ số thập phân, căn phải, `tabular-nums`; tiêu đề cột một dòng, trạng thái công bố thành chấm nhỏ trong ô; cuộn theo trang với tiêu đề + cột tên dính (không cuộn lồng); nhãn "Đủ" → "Đủ điều kiện thi" (hỏi PO Q-UX91-3). | 9.1 |
| **UX-17** | `/analytics` | Giảng viên | TB | Biểu đồ đường vượt thang chữ: nhãn trục 20 px+, nhãn "47 câu" đậm lớn, chấm tròn viền dày; nặng nề hơn mọi chữ khác trên trang. Hai cột "T6 … T5" thay vì ngày; không đánh dấu ngày hôm nay. | `analytics__gv__1440.png` | DESIGN §5 (thang chữ cố định, không cỡ tuỳ ý); §7 | Dùng token `--ep-text-meta` cho nhãn trục, đường 1,5 px, chấm chỉ ở điểm cuối và điểm cao nhất; nhãn trục dạng ngày ngắn (`02/11`). | S |
| **UX-18** | Hôm nay | Giảng viên | Thấp | ~~Chiều rộng nội dung không đồng nhất giữa các route~~ **Đính chính (2026-11-08, PM):** DESIGN §2 đã quy định "đọc / form 960 px, bảng fluid", nên 960 px của Hôm nay / Insights / Analytics là đúng thiết kế, không phải lỗi nhất quán. Còn lại một việc: ở ≥ 1440 px, 240 px trống bên phải của **Hôm nay** chưa dùng. | `hom-nay__gv-lop1__1440.png` | DESIGN §2, §17 (≥ 1440 dùng được panel ngữ cảnh) | Chỉ Hôm nay giảng viên: dùng khoảng trống làm cột phải "Lớp cần chú ý" + "Sắp tới" (gộp vào UX-5). Insights / Analytics giữ 960 px. | 9.1 (gộp UX-5) |
| **UX-19** | `/documents` | Giảng viên | TB | Ô chọn "Loại" cắt chữ (`Quy chế m`). "Chia sẻ từ 761987" lặp ở 5/6 hàng. Hai công tắc "Dùng cho AI" / "Hiện cho sinh viên" xám đậm vs đen khác nhau mà không có chú giải trạng thái (đang bật?). | `documents__gv__1440.png` | DESIGN §22.7 (chữ Việt dài không vỡ bố cục); luật chữ UI | Mở rộng ô chọn hoặc hiển thị nhãn loại dạng chữ (đổi bằng menu); bỏ "Chia sẻ từ" ở hàng, chỉ hiện khi tài liệu **không** thuộc lớp đang xem; công tắc thêm nhãn bật / tắt cho trình đọc màn hình và đồng nhất màu. | 9.1 |
| **UX-20** | `/threads` | SV + GV | TB | Đoạn trích hiện ký tự markdown thô (`- - Individual uses email…`, `· Phisher`). Nhãn "Chờ xác nhận" + chấm hổ phách lặp ở mọi hàng, ngày `00:10 02-11` (xem UX-8). Di động: nhãn "Bộ lọc" trơ như chữ, không giống điều khiển. | `threads__gv__1440.png`, `threads__sv__375.png` | UX.md §2.7 (nhãn nguồn gốc rõ); DESIGN §12 | Làm sạch đoạn trích (bỏ `-`, `·` đầu dòng, rút gọn 140 ký tự); "Chờ xác nhận" chỉ hiện khi **khác** trạng thái đa số của tab (hoặc chuyển thành nhóm); "Bộ lọc" thành nút có biểu tượng + số bộ lọc đang bật. | 9.1 |
| **UX-21** | `/me` | Sinh viên | TB | Cùng dữ liệu hiển thị hai lần: khối thành phần (Giữa kỳ · 20% / 7,5 …) rồi lại khối giải trình ("Giữa kỳ (20%) 7,5 = 7,5"). Phụ đề chỉ có "761987" trần. Trang dài 2.645 px ở 375. | `me__sv__375.png` | DESIGN §14.9 (hiểu điểm và cái gì ảnh hưởng); UX.md §2.7 | Một khối giải trình duy nhất (mỗi dòng: thành phần · trọng số · điểm → đóng góp, "Căn cứ · tr. 1" ở cùng dòng); bỏ phụ đề mã lớp (đã có ở thanh trên). | 9.1 |
| **UX-22** | `/gradebook` (lớp 2), `/grading` (lớp 2) | Giảng viên | Thấp | Trạng thái rỗng thiếu "vì sao rỗng" và lối ra: "Chưa có bài nộp" (không bảo làm gì), sổ điểm lớp trống kèm câu nhỏ cuối "Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường." | `grading__gv-lop1__1440.png`, `gradebook__gv-lop2-rong__1440.png` | DESIGN §15; UX.md §2.5 | Theo mẫu §15: "Chưa có bài nộp. Sinh viên nộp trong ứng dụng, hoặc thầy/cô tải ZIP lên." kèm nút `Tải bài lên`. | N |
| **UX-23** | `/calendar` | Giảng viên | Thấp | Ba nút đầu trang gần nghĩa: `Thêm vào lịch`, `Thêm sự kiện`, `Tạo lịch buổi học`. Ba hành động cùng hàng = 1 chính + 2 phụ (đúng luật) nhưng tên mơ hồ. | `calendar__gv__1440.png` | DESIGN §12; heuristic 2 | Gộp "Thêm vào lịch" và "Thêm sự kiện" thành một menu `Thêm` (buổi học · sự kiện · nhập từ tệp). | S |
| **UX-24** | `/questions` | Giảng viên | Thấp | Sáu bộ lọc hiện cùng lúc; 5 câu tên "mẫu" (dữ liệu demo). | `questions__gv__1440.png` | DESIGN §0 ("không hiện mọi bộ lọc"); cognitive load (≤ 4 lựa chọn) | Giữ ô tìm + 2 bộ lọc (Trạng thái, Chủ đề), phần còn lại vào nút `Bộ lọc`. Seed đặt tên câu có nghĩa. | S |
| **UX-25** | Dữ liệu demo + bộ chọn lớp | Giảng viên | TB | (a) Đăng nhập vào **lớp 761988 trống** (đưa đầu danh sách), nên giảng viên thấy sổ điểm trống, chấm bài trống, sinh viên toàn "—" rồi mới thấy lớp thật. (b) Hai lớp cùng tên "An ninh mạng", chỉ khác mã. (c) Lớp 2: mọi sinh viên có cùng chuyên cần "—", phát biểu "0 · 0,00", học "14 phút · 0 câu hỏi" (seed đồng nhất). | `hom-nay__gv-lop2__1440.png`, `students__gv-lop2__1440.png`, `bo-chon-lop__gv__1440.png` | UX.md §2.5; DESIGN §0 (hiểu việc cần làm ngay) | (a) **Cách S (PM, TL-2):** nhớ lớp đã chọn (`ep:ui:course`); lần đầu chọn lớp đầu tiên có dữ liệu theo thứ tự API. "Lớp có việc chờ nhiều nhất" cần API: **Nợ**. (b) Bộ chọn lớp hiện buổi học tiếp theo cần dữ liệu theo lớp: **Nợ**. (c) Seed lớp 2 đa dạng: việc của dev / QC, **Nợ** (`docs/PROGRESS.md`), không có story. | N (chỉ phần a) |
| **UX-26** | Toàn app (motion) | Mọi vai | TB | Phản hồi bấm: **không** `:active` (không có `scale(.97)` hay đổi màu). Hover nằm ngoài `@media (hover: hover) and (pointer: fine)` (71 `:hover`, 0 gate) → "dính" hover trên cảm ứng. `transition: width` ở 4 thanh tiến độ, `margin-left` + `width` ở khung vỏ (đổi layout, không chạy GPU). `prefers-reduced-motion` đặt `1ms` cho **mọi** thứ (xoá cả phản hồi chức năng). Dialog / Menu có `translateY` nhưng `transform-origin` giữ mặc định. | mã nguồn: `shared/ui/Button.module.css`, `shared/shell/AppShell.module.css:17,82,150`, `shared/styles/base.css:51`, `features/*/*.module.css` | `emil-design-eng`; `review-animations` tiêu chuẩn 5, 7, 8; DESIGN §11, §16 | Xem mục 4. Gộp thành **một đợt** sửa ở `shared/ui` + `base.css`, không đụng từng màn. | **N** |
| **UX-27** | `/login` | Chưa đăng nhập | Thấp | Sau lỗi sai mật khẩu, ô mật khẩu bị xoá và nút `Đăng nhập` về nhạt (khoá) ngay cả khi email còn: người dùng không thấy cách thử lại ngoài gõ lại. Trang không có đường quay lại giới thiệu (chỉ logo). Footer "Cần giúp?…" cách xa khung (cuối màn). | `login__anon__1440.png` | UX.md §2.3 (không mất chữ); DESIGN §15 | Giữ email, đặt con trỏ ở ô mật khẩu (đã làm) và nút luôn bấm được; lỗi gắn dưới ô mật khẩu (đã làm), thêm liên kết "Quên mật khẩu?" cạnh lỗi. Đưa "Cần giúp?" lên sát khung. | S |

**Quan sát cần QC xác minh (BA chưa có bằng chứng chạy):** dính đáy của `Duyệt bài` ở UX-15; axe và tương phản chữ `--ep-ink-3` trên nền `--ep-canvas`; lỗi console từng route; hành vi xung đột 409 / mất mạng.

---

## 3. Mẫu đề xuất cho các màn chính

Bố cục dưới đây dựng được bằng token `--ep-*` và primitive có sẵn (`Panel`, `ActionList`, `DataTable`, `Button`, `Tabs`). Chỗ cần đổi `DESIGN.md` ghi **[ĐỔI DESIGN]**.

### 3.A Trang giới thiệu `/` (chế độ Persuade; bắt buộc)

**Vấn đề.** UX-1, 2, 3. Trang đẹp nhưng im lặng. Người xem không thấy sản phẩm *làm việc*: ảnh chụp bị cắt, không có luồng, không có video.

**Hướng mới.** Một câu chuyện cuộn dọc, mỗi màn hình một ý. Thông điệp trung tâm vẫn là **"AI soạn nháp → giảng viên duyệt → mã tính điểm"**.

```text
┌ Logo EduPilot ──────────────────────────────── Đăng nhập ┐
│                                                          │
│  Vận hành lớp học có trợ lý AI,                          │   ← h1, hiện ngay (là LCP), không ẩn bằng opacity
│  bảo vệ dữ liệu cá nhân                                   │
│  AI soạn nháp. Giảng viên quyết. Điểm do mã tính.         │
│  [ Đăng nhập ]  [ Xem video 75 giây ▶ ]                   │   ← nút chính + nút phụ mở video
│                                                          │
│   ┌─ khung màn hình duyệt bài (ảnh) ─────────────────┐   │   ← vào khung nhìn: trượt lên 24 px + mờ → rõ
│   │ đường đỏ mảnh chạy từ "5,25" tới "Duyệt bài"      │   │      (Red Thread, chỉ ở trang này)
│   └───────────────────────────────────────────────────┘   │
├──────────────────────────────────────────────────────────┤
│ Ba bước, một người quyết                                  │   ← SƠ ĐỒ LUỒNG (SVG, vẽ theo cuộn)
│  ① AI soạn nháp ──▶ ② Giảng viên duyệt ──▶ ③ Mã tính điểm │
│   (chấm theo rubric)   (sửa, công bố)        (không LLM)  │      mỗi nút hiện khi cuộn tới; đường đỏ nối
├──────────────────────────────────────────────────────────┤
│  146    30/30     22                                      │   ← SỐ ĐẾM LÊN, mỗi số 1 câu người đọc hiểu
│  yêu cầu  điểm      bài chấm cùng lúc,                    │
│  gửi AI,  khớp      chat không chậm thêm                  │
│  0 lộ tên tính tay                                        │
├──────────────────────────────────────────────────────────┤
│ [ Giảng viên | Sinh viên ]   ← 2 tab, mỗi vai 4 mục       │
│  khung sản phẩm đổi theo tab (chuyển mờ 160 ms)            │
├──────────────────────────────────────────────────────────┤
│  Bắt đầu với lớp của bạn   [ Đăng nhập ] [ Tạo tài khoản ]│   ← CTA cuối
└ Đồ án tốt nghiệp · GitHub · "Dữ liệu demo là mô phỏng" ───┘
```

**Quy tắc chuyển động riêng của trang** (chi tiết mục 4):
- chữ hero không ẩn (đảm bảo LCP ≤ 2,5 s);
- hiệu ứng theo cuộn bằng CSS `animation-timeline: view()`, có dự phòng `IntersectionObserver` (cho trình duyệt thiếu);
- số đếm lên bằng CSS `@property` + `counter`, không JS;
- video `preload="none"`, có poster, chỉ phát khi bấm, kèm bản chữ cho người dùng không xem video;
- tắt chuyển động → trang hiện **trạng thái cuối** ngay (không chớp, không mất nội dung).

**[ĐỔI DESIGN]** Thêm mục "Trang giới thiệu (Persuade)" vào `DESIGN.md`: cho phép chuyển động cuộn / sơ đồ / số đếm, giữ nguyên token và Be Vietnam Pro; ghi rõ §11 ("không dàn dựng khi tải trang") chỉ áp dụng cho ứng dụng sau đăng nhập.

### 3.B Hôm nay

**Giảng viên.** Vấn đề: UX-5, 6. Hướng mới: **một việc kế tiếp**, rồi nhóm.

```text
Chủ Nhật, 8/11                                      ┌ Lớp cần chú ý ───────┐
9 việc cần xử lý hôm nay                            │ 761987               │  ← chỉ ≥ 1440, cột phải
                                                    │ 5 câu chờ quá 72 giờ │     (DESIGN §14.1 mục 3)
┌ Việc kế tiếp ──────────────────────────────────┐  │ 4 bài cần xem kỹ     │
│ ● Câu hỏi chờ giảng viên · quá 72 giờ           │  └──────────────────────┘
│   5 câu, lâu nhất 240 giờ          [ Mở hộp thư ]│
└─────────────────────────────────────────────────┘
Gấp
  ● 4 bài cần xem kỹ · Bài tập 03                          Xem bài →
  ● 1 yêu cầu có email chưa khớp MSSV                      Xác nhận →
Hôm nay
  ○ 2 bài bị báo cáo chờ xem                               Mở →
  ○ 5 câu trả lời AI chờ xác nhận                          Mở →
Khi rảnh
  ○ 1 cặp bài code nghi giống nhau                         Mở →
  …
Sắp tới   Buổi 5 · An ninh mạng · Thứ 2, 9/11 · 13:30 · P.405
```

- Chỉ **một** nút đỏ trong khung nhìn đầu (việc kế tiếp). Các hàng sau dùng liên kết trung tính có mũi tên **[ĐỔI DESIGN]** (thêm biến thể `link` cho hàng; hiện `text` luôn đỏ).
- Mã lớp một lần ở thanh trên, không lặp ở hàng; chỉ hiện khi chọn "Tất cả lớp".
- Nhãn nhóm bằng chữ (Gấp / Hôm nay / Khi rảnh) thay cho việc chỉ dựa màu chấm.
- Chuyển động: Red Thread khi bấm hàng (đã có, giữ nguyên).

**Sinh viên.** Đã tốt (xem điểm mạnh). Chỉnh nhỏ (UX-4, 8):
- thêm khối "Hôm nay" (buổi học / hạn bài) **lên trên** "Tiếp tục học" (§14.1 yêu cầu timeline trong khung nhìn đầu; hiện đang ở dưới màn hình ở 1440);
- loại câu hỏi trùng ở "Tiếp tục học" (hiện hai hàng "Sâu Morris là gì…" lặp);
- ngày giờ dùng định dạng chung (UX-8).

### 3.C Chat riêng (sinh viên)

Vấn đề: UX-11.

```text
Chat riêng                                              [ Phiên mới ]
┌─ Lịch sử ──┐ ┌──────────────── Panel hội thoại (≤ 840 px) ───────────┐
│ (cột phẳng,│ │  Chào Uyên. Em có thể hỏi về điểm, hạn bài hay bài học.│
│  đường kẻ, │ │  Gợi ý                                                  │
│  thu gọn)  │ │   · Bài tập 04 còn 6 giờ: nộp thế nào?                  │
│            │ │   · Bạn sai 4/7 câu "Mật mã đối xứng": xem lại chỗ nào? │
│ Sâu Morris… │ │   · Điểm quá trình hiện tại tính ra sao?                │
│ Mỗi lần…   │ │ ─────────────────────────────────────────────────────  │
│ …    ⋯     │ │  [ Hỏi bất cứ điều gì về lớp này…            ] [ Gửi ]  │
└────────────┘ └────────────────────────────────────────────────────────┘
```

- **Một Panel** (hội thoại). Lịch sử là cột phẳng bên trái, thu gọn được; ở di động thành nút `Phiên trước (6)` mở bảng trượt từ đáy, không chiếm đầu trang.
- `Phiên mới` hiện **chỉ khi** phiên hiện tại đã có tin nhắn.
- Gợi ý lấy từ dữ liệu sinh viên (hạn bài, chủ đề sai nhiều), dạng hàng chữ.
- Chuyển động: tin nhắn mới hiện bằng mờ + `translateY(4px)` 160 ms (tắt khi stream đang chạy token); nút Gửi `:active` thu nhẹ.
- Xoá phiên trong menu `⋯` của hàng; dùng hoàn tác 5 giây (UX.md §2.2), không hộp thoại.

### 3.D Duyệt bài

Vấn đề: UX-14, 15. Giữ nguyên bố cục hai cột §14.13 (đã đúng).

```text
← Hàng chờ chấm                                       ‹ 3/12 ›   (trước / sau)
Bài tập 03 — Phân tích một vụ tấn công thực tế
Phiên bản 1 · Nộp Thứ Năm, 29/10 · Chưa duyệt

┌ Bài làm ────────────────────┐ ┌ Chấm ───────────────────────────────┐
│ bai-tap-03.md · Mở tệp gốc   │ │ Tổng điểm                     5,25  │
│ …nội dung…                   │ │ ─────────────────────────────────── │
│ (đoạn trích được tô sáng khi │ │ Xác định tác nhân      [ 2,00 ] /2,5│  ← 1 dòng: tên · ô điểm · tối đa
│  bấm vào dòng trích bên phải)│ │  ❝Kẻ tấn công là nhóm APT29…❞       │  ← dòng trích có biểu tượng, nhắc một lần ở đầu cột
│                              │ │ Phân tích tấn công     [ 1,25 ] /2,5│  ⚠ lệch > 1 điểm so với lượt 2
│                              │ │ Nhận xét cho sinh viên              │
│                              │ │ [ ………………………………………… ]            │
│                              │ ├─────────────────────────────────────┤
│                              │ │ Đã duyệt 0/22   [ Duyệt bài  A ]    │  ← dính đáy cột, luôn thấy
└──────────────────────────────┘ └─────────────────────────────────────┘
```

- Mỗi tiêu chí **một** hàng, không lặp tên.
- "Đoạn trích từ bài làm · bấm để xem trong bài": một dòng chú thích ở **đầu** cột, không lặp.
- Thanh hành động dính đáy: `Duyệt bài` (phím `A`) + tiến độ "Đã duyệt 0/22"; `‹ ›` chuyển bài kế (không rời bàn phím).
- Chuyển động: khi bấm dòng trích, bài làm cuộn tới đoạn và tô sáng bằng nền `--ep-red-soft` mờ dần 600 ms (đã dùng để *báo vị trí*, đúng nghĩa đỏ "đang ở đâu"). Sau `Duyệt bài` → bài kế vào bằng mờ 120 ms.

### 3.E Sổ điểm

Vấn đề: UX-13, 16.

```text
Sổ điểm                          [ Xem công thức ]  ⋯  [ Chốt điểm ]
                                                      ↑ khoá: "Cần xác nhận công thức"
┌──────────────────────────────────────────────────────────────────────┐
│ Sinh viên        │ Giữa kỳ  Kiểm tra  Thi cuối │ Cộng  Trừ │ Quá trình │ Học phần │ Chữ │
│ (dính)           │   20 %     20 %     60 %    │           │           │          │     │
│ Bùi Khánh Việt   │ [5,00]    —       [5,00]    │ 0,50 0,00 │   6,00    │   5,50   │  C  │
│ Bùi Minh Yến     │ [3,00]    —       [3,00]    │ 0,00 0,00 │   5,50    │   4,00   │  D  │
└──────────────────────────────────────────────────────────────────────┘
 (cuộn theo trang; tiêu đề cột + cột tên dính)       ● Đã công bố giữa kỳ
```

- Mọi số **hai chữ số thập phân, căn phải, `tabular-nums`**; ô sửa có viền, ô chỉ đọc phẳng (giữ).
- Tiêu đề cột một dòng; "Đã công bố" thành chấm nhỏ cạnh tiêu đề (chú giải ở chân bảng).
- **Không cuộn lồng**: bảng cuộn theo trang, header dính (`position: sticky`).
- 409 xung đột: ô đổi sang viền hổ phách + dòng "Bạn kia vừa đổi thành 6,5 · Giữ của tôi · Lấy 6,5" ngay dưới ô (UX.md §5).
- Chuyển động: ô vừa lưu nháy nền xanh nhạt 200 ms rồi mờ (xác nhận lưu, không toast).

---

## 4. Chuyển động

Quyết định của chủ dự án (plan 9.1, mục cuối): **trang giới thiệu** là nơi đầu tư; **ứng dụng** chỉ chuyển động chức năng ≤ 200 ms, không thêm thư viện.

### 4.1 Ứng dụng sau đăng nhập

Tiêu chuẩn: `emil-design-eng` (tần suất → mục đích → easing → thời lượng) và `review-animations` (10 chuẩn). Hiện tại token: `--ep-fast: 140ms`, `--ep-normal: 200ms`, `--ep-ease-out: cubic-bezier(.16,1,.3,1)` — **đã đúng** ngưỡng.

| Chỗ | Hiện tại | Đề xuất | Thời lượng · easing | Lý do |
| --- | --- | --- | --- | --- |
| Nút `.btn` (mọi nơi) | Chỉ đổi màu khi hover | Thêm `:active { transform: scale(.97) }`; hover gắn `@media (hover: hover) and (pointer: fine)` | 100 ms · `--ep-ease-out` | Nút phải "nghe" được lần bấm; không hover dính trên cảm ứng. |
| Hàng `ActionList`, `DataTable` | Đổi nền 140 ms | Giữ; gắn `hover: hover`. **Không** thêm chuyển động cho thao tác bàn phím (J/K, `A`, 1–4). | 140 ms | Tần suất hàng trăm lần mỗi ngày → không hoạt cảnh (emil). |
| Drawer / Dialog | `fadeIn` 6 px / `slideIn` 24 px, 200 ms | Giữ; Dialog di động trượt từ đáy 200 ms; `transform-origin` đúng nguồn gốc cho Menu / popover | 160–200 ms · ease-out | Đã đúng; sửa gốc xuất phát. |
| Menu, popover | `pop` 4 px, 200 ms | Giảm còn 160 ms; `transform-origin` ở nút gọi | 160 ms · ease-out | Nhỏ hơn 200 ms cho thứ hiện hàng chục lần/ngày. |
| Thanh tiến độ (4 chỗ) | `transition: width` (đổi layout) | `transform: scaleX()` với `transform-origin: left` | 140 ms · ease-out | Chỉ `transform` / `opacity` chạy GPU (chuẩn 7). |
| Khung vỏ thu gọn sidebar | `transition: width` + `margin-left` (3 chỗ) | Giữ **tức thì** hoặc `transform` cho sidebar, không đổi `margin-left` của `main` | ≤ 160 ms | Thu gọn sidebar không nên giật layout nội dung. |
| Xác nhận lưu (sổ điểm, điểm danh, nhận xét) | Chữ "Đã lưu" | Thêm nháy nền xanh nhạt 200 ms cho ô / hàng vừa lưu | 200 ms · ease-out | Phản hồi tại chỗ thay cho toast (UX.md §2.2). |
| Red Thread (Hôm nay → đích) | 280 ms | Giữ (đặc trưng thương hiệu, tần suất thấp) | 280 ms | Đã nằm trong DESIGN §11. |
| Skeleton shimmer | `translateX` | Giữ | 1,2 s lặp | Hợp lệ (chỉ `transform`). |
| Stream chat | `blink` con trỏ | Giữ | — | Chức năng. |
| **Bỏ / không thêm** | — | Stagger khi tải trang trong ứng dụng; hiệu ứng mờ + trượt cho danh sách dài; bounce; parallax; đếm số trong báo cáo. | — | Trang trí, làm chậm thao tác lặp. |

**`prefers-reduced-motion`** hiện đặt mọi `animation-duration` / `transition-duration` về `1ms`. Đề xuất: **giảm thay vì tắt** — giữ `opacity` và màu (xác nhận trạng thái), bỏ `transform` / trượt. Dòng hiện tại phá phản hồi chức năng (nháy lưu thành không có gì).

### 4.2 Trang giới thiệu

Bảng nhịp chuyển động (mọi thứ chỉ `transform` + `opacity`, đều qua CSS hoặc WAAPI; **không thư viện**):

| Khoảnh khắc | Hiệu ứng | Cách dựng | Thời lượng · easing |
| --- | --- | --- | --- |
| Mở trang (hero) | Chữ hero hiện ngay (LCP). Khung sản phẩm vào bằng mờ + `translateY(24px → 0)`; nút chính có vòng sáng đỏ mảnh một lần | `@starting-style` hoặc CSS `animation` có `animation-delay` nhỏ | 500–600 ms · `cubic-bezier(.16,1,.3,1)` |
| Đường đỏ nối trong khung sản phẩm | Vạch đỏ mảnh chạy từ "5,25" tới nút `Duyệt bài` | SVG `stroke-dashoffset`, gắn `animation-timeline: view()` | theo cuộn |
| Sơ đồ luồng ① → ② → ③ | Từng nút hiện khi cuộn tới, đường nối đỏ vẽ dần | `animation-timeline: view()` + `animation-range: entry 10% cover 40%`; dự phòng `IntersectionObserver` + WAAPI | 400 ms mỗi bước, so le 80 ms |
| Số liệu | Đếm lên khi vào khung nhìn, chạy một lần | CSS `@property --n` (kiểu `<integer>`) + `counter()`; dự phòng WAAPI | 900 ms · `ease-out` |
| Đổi tab Giảng viên / Sinh viên | Khung sản phẩm chuyển mờ chéo | `view-transition` hoặc `opacity` 160 ms | 160 ms |
| Ảnh / khung sản phẩm khi cuộn | Chiều sâu nhẹ: `translateY` ≤ 16 px theo cuộn (parallax rất khẽ) | `animation-timeline: scroll()` | theo cuộn |
| Nút chính | `:active` thu 0,97; hover đổi màu | CSS | 100 ms |
| Video | Bấm mới tải và phát; poster tĩnh; không tự phát, không tiếng | `preload="none"`, `poster`, `controls` | — |

**Ràng buộc kỹ thuật** (chuyển cho `research` đánh giá, xem `tl-feasibility.md`):
- h1 và đoạn dẫn **không** dùng `opacity: 0` ban đầu (sẽ trượt LCP);
- `animation-timeline` chưa có ở mọi trình duyệt → bọc `@supports (animation-timeline: view())`; nơi không hỗ trợ dùng `IntersectionObserver` nhỏ (ước < 1 KB gzip) để thêm lớp `.in`;
- `prefers-reduced-motion: reduce` → hiện thẳng **trạng thái cuối** (số hiện luôn giá trị cuối, sơ đồ vẽ đủ);
- `will-change` chỉ trong lúc chạy;
- video: poster WebP, `mp4` H.264 + `webm`, < 8 MB, kèm bản chữ (transcript) cho a11y;
- ngân sách giữ: LCP ≤ 2,5 s, JS ≤ 256 KB.

**Không cần thư viện.** Nếu dev thấy một hiệu ứng cần (ví dụ cuộn mượt), mở proposal theo quyết định của chủ dự án.

---

## 5. Ưu tiên

### Làm ngay (sửa nhỏ, ảnh hưởng lớn)

| ID | Việc | Ghi chú |
| --- | --- | --- |
| UX-4 | Xoá 55 `description=` (ưu tiên 12 câu dài) và các dòng giải thích lặp | Luật cứng của PO; cần vá SRS + Confluence nếu có message trong SRS |
| UX-8 | Một hàm định dạng ngày giờ dùng chung | Thay ~13 tệp |
| UX-13 | Nút khoá có lý do ngắn (tooltip + `aria-describedby`) | Một lần trong `Button` |
| UX-26 | `:active`, `hover: hover`, thanh tiến độ `scaleX`, reduced-motion giảm thay vì tắt | Một lần ở `shared/ui` + `base.css` |
| UX-22, UX-12 (phần chữ) | Câu rỗng đúng mẫu §15 | Chữ |
| UX-25 (a) | Nhớ lớp đã chọn; lần đầu chọn lớp đầu tiên có dữ liệu (cách S). (b), (c) ghi Nợ | Nhỏ |

### Làm trong 9.1

| ID | Việc |
| --- | --- |
| UX-1, 2, 3 | Trang giới thiệu mới + video (mẫu 3.A, mục 4.2) |
| UX-5, 6 | Hôm nay giảng viên (mẫu 3.B) |
| UX-7 | Điểm danh: chấm "Có mặt" không đỏ, thanh lưu dính đáy di động |
| UX-11 | Chat riêng (mẫu 3.C) |
| UX-14, 15 | Hàng chờ + Duyệt bài (mẫu 3.D) |
| UX-16 | Sổ điểm (mẫu 3.E) |
| UX-9, 10, 18, 19, 20, 21 | Nhãn điều hướng, khung lồng khung, chiều rộng, Tài liệu, Threads, `/me` |

### Để sau

UX-17 (biểu đồ), UX-23 (menu Thêm của Lịch), UX-24 (bộ lọc câu hỏi), UX-27 (đăng nhập), cùng mọi mục chưa đi ở đầu báo cáo (quản trị, trợ giảng, cài đặt).

### Phụ thuộc

- UX-1, 5, 11, 15, 16 có **[ĐỔI DESIGN]** hoặc đụng bố cục → PM cần chốt trước khi viết spec.
- Mọi mục UX-n làm trong 9.1 sẽ có AC kiểm được trong `docs/specs/FEAT-uiux-91/US.md` (pha 2), kèm ảnh mốc cần sinh lại.

---

## 6. Câu hỏi mở cho chủ dự án (không tự chọn thay)

| # | Câu hỏi | Đề xuất của BA | Lý do |
| --- | --- | --- | --- |
| Q-UX91-1 | Nhãn điều hướng `Threads`, `Insights`, `Analytics` giữ tiếng Anh hay đổi? | Nhãn nav = tiêu đề trang: "Hỏi đáp công khai" (hoặc giữ "Threads" như tên riêng, ghi rõ), "Lỗ hổng kiến thức", "Số liệu lớp học" | Hiện nav và h1 không trùng nhau; DESIGN §13 nói giao diện tiếng Việt. Đổi nhãn sửa `DESIGN.md` §2 và nhiều ảnh mốc. |
| Q-UX91-2 | Việc giảng viên ở Hôm nay có chia nhóm "Gấp / Hôm nay / Khi rảnh" không? Ngưỡng "Gấp" là gì? | "Gấp" = quá 72 giờ làm việc (đã là ngưỡng trong dữ liệu Hôm nay) hoặc có cờ rủi ro; "Khi rảnh" = phát hiện giống bài, bài chưa khớp | Đổi thứ tự / cách nhóm việc là hành vi sản phẩm (F14 bảng nguồn việc). |
| Q-UX91-3 | Cột "Trạng thái" của sổ điểm (giá trị "Đủ") nghĩa là gì và gọi là gì? | "Đủ điều kiện thi" (nếu đúng nghĩa) hoặc bỏ cột khi tất cả đều Đủ | BA chưa tìm thấy định nghĩa trong PRD M7; không tự đặt tên. |
| Q-UX91-4 | Mặc định "Có mặt" ở điểm danh dùng chấm mực thay cho chấm đỏ hiện nay có được không? | Có: đỏ chỉ cho Vắng / Vắng phép / Muộn | Đỏ là tín hiệu (DESIGN §4); 24 chấm đỏ mặc định làm mất ý nghĩa. Cần PO duyệt vì đổi quy ước màu của một màn đã có AC. |
| Q-UX91-5 | Trang giới thiệu có được nêu số đo từ môi trường mô phỏng (nhà cung cấp AI giả) không? | Có, nhưng thành dòng chú thích riêng "đo trên dữ liệu mô phỏng" | Số liệu đồ án phải có nguồn (CONTEXT mục 4 về quyển đồ án). |

# QC test case — US-PROTO-00 (Nền: vào bằng một vai, đổi vai)
Nguồn: `docs/sprints/1.5/spec/US.md` + `SRS.md` (v2 APPROVED; spec dời vào đây theo PM). Viết hộp đen, không đọc code dev.
Chạy: dev server `pnpm dev` hoặc `pnpm -C frontend exec next dev -p 3000` → `F=http://localhost:3000`. Mỗi shell `source ~/.zprofile`.
Công cụ: **C** = `F=… bash docs/sprints/1.5/qc/scripts/proto-curl.sh <TC>`; **B** = trình duyệt thật qua `sweep.mjs` / thao tác tay trong Eval (`browser`), ảnh vào `qc/shots/`.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-00-01 | AC1 | Không cookie | **C** `tc_00_01`: `curl -s -o /dev/null -w '%{redirect_url}' $F/` cho `/`, `/inbox`, `/chat`, `/gradebook`, `/me` | Mọi route chuyển về `/login` |
| TC-00-02 | AC1 | – | **B** Mở `/login` (xoá cookie + localStorage) | Có 4 vai: Sinh viên, Trợ giảng, Giảng viên, Admin; chọn Sinh viên → chọn tiếp A / B / C / D (tên + mô tả một dòng) |
| TC-00-03 | AC1 | – | **B** Chọn Sinh viên B | Vào `/`; menu hồ sơ ghi "Trần Thu Uyên"; dải "Bản mô phỏng · dữ liệu giả" hiện, chữ phụ, **không đỏ**; kiểm lại ở 4 vai và ở 390 px (đầu trang) |
| TC-00-04 | AC1 | – | **C** `tc_00_01` (phần `visible … grep -c 'Bản mô phỏng'`) | ≥ 1 ở `/` của cả 4 vai |
| TC-00-05 | AC1 / FR-X1 | – | **B** Sau khi chọn vai, `document.cookie` | Có `ep_demo_role`, `ep_demo_person`, `ep_demo_course`; không `httpOnly` yêu cầu (ngoài phạm vi) |
| TC-00-06 | AC2 / FR-X2, X3 | B đã gửi D3 ở `/chat` (làm sau US-PROTO-01) | **B** Menu hồ sơ → `Đổi vai` → Giảng viên → `/inbox` | Có ticket D3; không phải tải lại tay; `Đổi vai` mở lại lựa chọn ngay tại chỗ |
| TC-00-07 | AC2 | Sau TC-00-06 | **B** `Đặt lại dữ liệu demo` | Ticket D3 biến mất; điểm, điểm danh, công thức… về dữ liệu gốc SRS 4.1 |
| TC-00-08 | AC2 / FR-X3 | – | **B** DevTools → Application → Local Storage; cookies | Chỉ một khoá `ep_demo_state`; không giá trị giống token/JWT (`eyJ`, `Bearer`, `token`); sau `Đặt lại` khoá bị xoá |
| TC-00-09 | AC3 / FR-X7, X10 | Cây sạch | `pnpm -C frontend lint && pnpm -C frontend build && bash scripts/ui-antipatterns.sh` | Cả ba exit 0; 0 `✗` |
| TC-00-10 | AC3 | – | **C** `tc_00_static` | Không `fetch(` ngoài `src/shared/`; `package.json` không thêm phụ thuộc ngoài `lucide-react`; `localStorage` chỉ khoá `ep_demo_state`; không token ở storage; không màu cứng ngoài `tokens.css` |
| TC-00-11 | AC4 / FR-X8 | – | **B** GV mở bộ chọn lớp | Có 761987, 761988, "Tất cả lớp của tôi", "Quản lý lớp này"; đổi lớp → mọi màn đổi dữ liệu theo lớp |
| TC-00-12 | AC4 | – | **B** Sinh viên A / B / C / D mở bộ chọn lớp | A thấy 2 lớp; B, C 1 lớp (761987); D không có lớp |
| TC-00-13 | AC4 | – | **C** `tc_00_04` | `visible student / sv-4 \| grep -c 'mã tham gia'` ≥ 1; "Hôm nay" của D là ô nhập mã |
| TC-00-14 | AC5 (nhánh lỗi) | – | **C** `tc_00_05` (`?state=empty|error` trên `/inbox`, `/students`, `/gradebook`, `/documents`, `/insights`, `/analytics`, `/questions`, `/grading`, SV: `/threads`, `/library`, `/calendar`, `/me`, `/practice/history`) | `error` có `Thử lại`; `/inbox?state=empty` có "Không còn câu hỏi" |
| TC-00-15 | AC5 | – | **B** `?state=loading` trên 3 route mỗi nhóm (SV: `/threads`, `/me`, `/library`; GV: `/inbox`, `/students`, `/gradebook`; Admin: `/admin/users`, `/observability`, `/`) | Skeleton đúng hình vùng nội dung; **không spinner giữa trang** (DESIGN §21) |
| TC-00-16 | AC5 | – | **B** `?state=error` → bấm `Thử lại` | Bỏ tham số, về trạng thái thường; câu lỗi nêu vấn đề + cách khắc phục (DESIGN §15), không từ kỹ thuật với SV |
| TC-00-17 | AC6 (phân quyền) | – | **C** `tc_00_06` | Các lệnh `open_as` của US: GV bị chặn `/chat /me /practice /library /join`; SV bị chặn `/inbox /attendance /gradebook /grading /admin/courses /observability`; Admin bị chặn `/chat /inbox /gradebook /calendar`; TA mở `/class/members`, bị chặn `/admin/users` |
| TC-00-18 | AC6 | – | **C** `tc_00_06` (nội dung màn chặn) | Có "Bạn không có quyền mở trang này" (FR-X4), một câu "Trang này dành cho …", nút `Về Hôm nay`; không lộ nội dung trang |
| TC-00-19 | AC6 / FR-X4 | – | **C** `tc_00_matrix` (toàn bộ ma trận SRS mục 2, 4 vai × 31 route) | Mọi ô đúng MO/CHAN; R (TA `/gradebook`, GV `/observability`) mở được |
| TC-00-20 | AC6 | – | **B** Điều hướng (sidebar / bottom nav / ⌘K) của từng vai | Không hiện route bị chặn của vai đó (kể cả kết quả ⌘K) |
| TC-00-21 | AC6 | – | **B** Đang ở `/attendance` (GV) → `Đổi vai` sang Sinh viên | Không để lại nội dung `/attendance` cho SV: về `/` hoặc màn chặn |
| TC-00-22 | AC7 (#17a) | 1440 × 900, 4 vai | **A** `audit.mjs` phép đo "00-AC7 brand" ×4 vai; **B** Console theo US 00-AC7 | `[data-part=brand] img` cao đúng 32; `[data-part=brand]` cao 56, `bottom` = `header` `bottom` = 56, viền dưới 1 px (một đường liền với topbar); tỉ lệ logo 188:40 (±0,15); ảnh `/` 4 vai |
| TC-00-23 | AC7 | GV, 1440 | **A** bấm `Thu gọn` rồi đo | Chỉ còn mark 32 × 32, căn giữa cột 72 px (±1 px); brand vẫn cao 56 và viền dưới vẫn liền với topbar; ảnh |
| TC-00-24 | AC7 | SV B, GV; 390 × 844 | **A** đo `header a[href="/"] img` | Mark 28 × 28, vùng chạm của link ≥ 44 × 44, đứng bên trái, trước bộ chọn lớp; bấm → về `/` |
| TC-00-25 | AC7 | Không cookie | **A** `/login` ở 1440, 390, 375 | `img[alt="EduPilot"]` cao 40 (1440) / 32 (390, 375); không cuộn ngang; logo phía trên tiêu đề; ảnh |
| TC-00-26 | AC7 (biên) | – | **B** 414 và 375 px, mọi vai; đổi route / đổi vai / `Đặt lại dữ liệu demo` | Mark vẫn 28 × 28 và không méo ở 414 / 375; chuyển route hay đổi vai không làm logo đổi cỡ hay nhảy chỗ; link logo có tên truy cập được (`alt` / `aria-label`) |
| TC-00-27 | AC8 (#17b) | GV, 1440 | **A** "00-AC8 chooser" ở `/`, `/inbox`, `/gradebook`; **B** | Bộ chọn lớp hiện đủ "761987 · An ninh mạng" (không `…`), rộng ≤ 360 px; ô tìm nhanh hiện đủ "Tìm nhanh hoặc đi đến…" không bị cắt; AUDIT `ell: []` |
| TC-00-28 | AC8 | GV, 1024 / 1099 / 720 px | **A** đo nút bộ chọn lớp; rê chuột | Rộng ≤ 240 px; nếu chữ bị cắt thì cuối có `…` **và** `title` + `aria-label` bằng tên đầy đủ; rê chuột thấy tooltip đủ tên; AUDIT `ell: []` |
| TC-00-29 | AC8 (biên) | GV | **A** 1100 → 1099 → 720 → 719 px | 1100: đủ tên, ≤ 360; 1099 và 720: ≤ 240; 719: chỉ "761987" + mũi tên (SRS 4.7 b1–b3, ngưỡng đúng 720 / 1100) |
| TC-00-30 | AC8 | GV, 390 px | **A** đo nút; mở bộ chọn lớp | Nút chỉ "761987" + mũi tên; popover có tên đầy đủ ("An ninh mạng – 761987") ở đầu, mục xuống dòng, không cắt chữ, không tràn ngang; ảnh |
| TC-00-31 | AC8 (nhánh) | SV A (2 lớp), SV D (chưa có lớp), Admin | **B** 390 và 1440: bộ chọn lớp / nhãn trên topbar | A: cả hai lớp đọc đủ trong popover; D: "Chưa có lớp" không cắt; Admin: "Toàn hệ thống · INT1006" không cắt; AUDIT `ell: []`, `cut: []` |
| TC-00-32 | AC8 | – | **C** `tc_00_08` | HTML GV có `title="761987 · An ninh mạng"` (≥ 1) và `aria-label` chứa tên đủ |
| TC-00-33 | AC9 (#17f) | – | **C** `tc_00_09` | `Tài khoản: TS. Lê Thu Hà`, `Phạm Quốc Bảo`, `Đỗ Hoàng Nam`, `Trần Thu Uyên` mỗi chuỗi ≥ 1; SV A, C, D ra đúng tên; aria-label **không** ghi tên vai ("Giảng viên", "Trợ giảng", "Admin") |
| TC-00-34 | AC9 | 4 vai; SV A, B, C, D | **A** "00-AC9" 1440 và 390 (mở menu); **B** ảnh | Dòng 1 = tên người, dòng 2 (chữ phụ) = vai ("Giảng viên", "Trợ giảng", "Admin", "Sinh viên"); không vai nào hiện tên vai ở chỗ tên; tên dài không cắt ở 390 / 375; ảnh menu 3 vai |
| TC-00-35 | AC9 (biên) | Menu hồ sơ đang mở | **B** `Đổi vai` sang vai khác không tải lại | Nút hồ sơ đổi `aria-label` và dòng 1 theo người mới ngay; avatar giữ chữ cái như cũ |
| TC-00-36 | AC10 (#17h) | – | **A** ma trận: mọi route × vai × 1440 / 390 (kể cả màn chặn quyền, `/login`) | Mọi dòng `{ ox: 0, cut: [], ell: [] }`; bảng route × bề rộng đính report; không lỗi console |
| TC-00-37 | AC10 | – | **A** ma trận 375: route SV, `/attendance`, `/inbox` | `{ ox: 0, cut: [], ell: [] }` ở 375 |
| TC-00-38 | AC10 (h1) | – | **A** `/library` (SV), `/attendance`, `/observability` (GV và Admin), `/admin/users` ở 390; `/library`, `/attendance` thêm 375 | `ox` = 0 (v4 lỗi 4, 36, 34, 101 px); ảnh 390 của 4 route đính report |
| TC-00-39 | AC10 (h3) | 390 và 719 px | **A** "00-AC10 h3" `/students`, `/grading`, `/documents`, `/admin/courses`, `/admin/users` | Không còn `<table>` nhìn thấy; mỗi hàng 2–3 dòng, cột chính đậm, cột còn lại "Nhãn: giá trị", chip đủ chữ ("Chưa duyệt", "Rủi ro", "Dùng cho AI" không khuất); `cut: []`; ảnh |
| TC-00-40 | AC10 (h3) | GV, 390 | **A** `/gradebook`: cuộn vùng `data-scroll-x` 220 px | Vùng cuộn ngang có `data-scroll-x`; cột tên dính trái (vị trí không đổi khi cuộn); đầu cột "QT tạm tính" không cắt chữ; trang không tràn (`ox` 0) |
| TC-00-41 | AC10 (h2) | GV, 375 | **A** `/attendance` | Chú thích phím tắt (↑/↓, 1–4, P) ẩn; hàng dạng danh sách, nhãn "Vắng phép" đủ chữ; `cut: []`; vùng chạm ≥ 44 px |
| TC-00-42 | AC10 (h4) | GV, 390 | **A** `/students/sv-3`: bấm tab "Ghi chú" | Dải tab cuộn ngang; "Hoạt động học" không cắt; tab vừa chọn tự cuộn vào khung; `ox` 0 |
| TC-00-43 | AC10 (biên 720) | – | **A** 720 / 719 px trên 5 route DataTable | 720: dạng bảng, không tràn; 719: dạng danh sách (đúng ngưỡng SRS 4.7 h3) |
| TC-00-44 | AC10 (nhánh lỗi) | – | **A** `states:true`: `?state=empty` và `?state=error` mọi route, 390 / 375 | Trạng thái rỗng / lỗi cũng đạt `{ ox: 0, cut: [], ell: [] }` (không vỡ khi thay nội dung) |
| TC-00-45 | AC10 | – | **A** kịch bản tương tác: chat sau D1, thread sau Hỏi trợ lý AI, form + hộp thoại hai lối, menu hồ sơ, bộ chọn lớp, sidebar thu gọn, ticket mở, drawer Thêm | Mọi trạng thái đạt AUDIT ở 1440 / 390 / 375 (theo kịch bản) |
| TC-00-46 | AC10 / US "Móc đo" | – | **C** `tc_00_07`, `tc_00_hooks` | Có `data-part`: `brand` (4 vai), `chat-history` / `chat-thread` / `chat-composer` (SV `/chat`), `inbox-list` (GV `/inbox`), `provider-status` / `provider-action` ≥ 3 (Admin `/settings/llm`); `inbox-detail` + `inbox-reply` có khi mở ticket (đo bằng **A**) |
| TC-00-47 | AC10 (chống lách) | – | **B** Console: `document.querySelectorAll('[data-scroll-x]')` ở mọi route 390 | `data-scroll-x` chỉ nằm trên vùng cố ý (hàng tab/chip lọc, bảng `/gradebook`, dải tab hồ sơ); không nằm trên `body`, `main`, `header` hoặc khối bao cả trang; mỗi vùng thực sự cuộn được (`scrollWidth` > `clientWidth`) |
| TC-00-48 | AC10 / SRS 4.7 cuối | – | `bash scripts/ui-antipatterns.sh`; `grep -rnE '#[0-9a-fA-F]{3,8}\b' frontend/src --include=*.css \| grep -v tokens.css` | Antipatterns 0 `✗`; không màu cứng ngoài `tokens.css` (chỉ token `--ep-*`) |
| TC-00-49 | AC11 (#19 E23, FR-X15) | SV B và GV, 1440 × 900 | **A** `LEFT (00-AC11) = 240` trên ma trận; **B** Console `LEFT` của US | `/`, `/chat`, `/threads`, `/practice`, `/library`, `/me`, `/calendar` (SV B), `/inbox`, `/students`, `/gradebook` (GV) đều `240` (v4 lệch 348 / 240 / 248); mọi route còn lại của 4 vai cũng `240` |
| TC-00-50 | AC11 | 390 px | **A** `LEFT = 16` | Mọi route (trừ `/login`) `16` |
| TC-00-51 | AC11 (biên) | 1920 × 1080 và 1100 px | **B** Console `LEFT` + bề rộng khối đọc / biểu mẫu | Khối đọc / biểu mẫu rộng tối đa 960 px và **căn trái** (tiêu đề không dịch ra giữa khi màn rộng); bảng và lưới rộng hơn vẫn bắt đầu từ cùng lề; `/chat` và `/calendar` cùng lề 240 (Q12) |
| TC-00-52 | AC11 (nhánh) | – | **B** `?state=loading`, `?state=empty`, `?state=error`; màn chặn quyền; màn "Bạn chưa vào lớp nào" (D); 404 | Tiêu đề trang vẫn ở lề 240 / 16 ở mọi trạng thái (không nhảy khi nội dung đổi); mỗi màn có đúng một `[data-part=page-title]` |
| TC-00-53 | AC12 (E31) | 1440, 390, 375 | **A** `HEADER đặc (00-AC12)` trên ma trận, đã cuộn 200 px | `background-color` dạng `rgb(…)` (không `rgba(`, không `/ 0.`) và `backdrop-filter: none` ở mọi route |
| TC-00-54 | AC12 | Cây sạch | **C** `tc_00_15`, `tc_00_16`; `bash scripts/ui-antipatterns.sh`; **thử đỏ** trong `git worktree` tạm ở `/tmp` (không đụng cây làm việc): thêm `backdrop-filter: blur(4px)` vào một CSS trong `shared/shell`, chạy script, xoá worktree | `shared/shell`, `shared/ui` không còn `backdrop-filter` / `transparent)` / `color-mix(… transparent`; script có mẫu `backdrop-filter` và **đỏ** khi thêm dòng thử, xanh khi cây sạch |
| TC-00-55 | AC12 (tay) | SV B, `/chat` 390 | **B** Gửi một tin, cuộn 200 px, chụp ảnh | Không chữ nào lộ mờ qua thanh trên (v4: "Chat riêng" lộ ra); ảnh đính report |
| TC-00-56 | AC13 (#19 E1, FR-X16) | – | **C** `tc_00_13` | Không cookie: không "This page could…"; GV / SV / TA / Admin: "Không tìm thấy trang" + `Về Hôm nay`; có `app/error.tsx` và `app/not-found.tsx` |
| TC-00-57 | AC13 | 4 vai | **A** 404 và id lạ: `/khong-co-trang`, `/threads/khong-co`, `/assignments/khong-co`, `/practice/khong-co`, `/students/sv-999` | Không bao giờ trang tiếng Anh mặc định ("404 This page could not be found", "This page couldn't load"); id lạ trong route có thật ra màn rỗng / 404 của app có hướng đi tiếp, không trắng trang |
| TC-00-58 | AC13 (nhánh lỗi) | SV B | **A** ghi đè `ep_demo_state` = `{`, `null`, `[]`, `{"x":1}` rồi tải `/`, `/practice`, `/chat`, `/threads`, `/me`, `/calendar`; thêm `{"practice":"x","tickets":5,"bell":7}` | Bốn dạng đầu: app dùng dữ liệu gốc, vào được, không màn lỗi; dạng sai cấu trúc: dữ liệu gốc **hoặc** màn "Trang này gặp sự cố" tiếng Việt có `Thử lại` + `Về Hôm nay`; không trang tiếng Anh; sau `Đặt lại dữ liệu demo` mọi thứ bình thường |
| TC-00-59 | AC13 | – | **B** Ở màn "Không tìm thấy trang" bấm `Về Hôm nay`; ở màn "Trang này gặp sự cố" (từ TC-00-58) bấm `Thử lại` rồi `Về Hôm nay` | Về `/` của vai đang chọn; `Thử lại` không treo vòng lặp, không để lại màn lỗi nếu dữ liệu đã sạch; màn lỗi nằm trong khung app (sidebar / thanh trên còn) |
| TC-00-60 | AC14 (#19 E26, E30, FR-X13) | – | **C** `tc_00_14` | Không còn chuỗi "N ngày / giờ / phút trước" ngoài `mock/derive.ts`; SV B không có "12 ngày trước"; Admin `/` và `/admin/courses` có "hôm qua 16:40"; Admin `/` không có "08:30" |
| TC-00-61 | AC14 | Sau `Đặt lại dữ liệu` | **B** Mở chuông SV B, chuông GV, Admin `/` | B: "Bài tập 03 đã nộp, đang chờ chấm · 6 ngày trước", "Tài liệu mới: Chương 5 — Quản lý khoá và PKI · hôm qua 14:00"; GV: phân công 761988 "hôm qua 16:40"; Admin: "Phân công · hôm qua 16:40" và "Mới mở · hôm qua 16:40" |
| TC-00-62 | AC14 / SRS 4.8 N6 (biên) | B đã gửi D3 | **B** Ghi đè `Date.now` bằng `page.evaluateOnNewDocument` lệch +59 / +60 phút / +23 giờ / +24 giờ rồi mở chuông | Mốc "vừa gửi" → "59 phút trước" → "1 giờ trước" → "23 giờ trước"; BT03 nộp 23/10 08:10 → "7 ngày trước" ở +24 giờ; Chương 5 (28/10 14:00) ở +24 giờ không còn "hôm qua" mà là "N ngày trước"; không chuỗi nào đứng yên |
| TC-00-63 | AC14 / N6 | B | **B** Gửi D3, chờ 2 phút thật, mở chuông; rồi `Đặt lại dữ liệu demo` | Mục D3 ghi "2 phút trước" (đồng hồ chạy theo giờ thật); sau Đặt lại: đồng hồ về 09:20 ("Thứ Năm, 29 tháng 10 09:20" ở `/me`, Hôm nay) |
| TC-00-64 | AC15 (#19 E22, FR-X14) | 375 và 390 px | **A** `TOUCH (00-AC15)` trên 12 route SV (B) + `/attendance`, `/inbox` (GV) | Mọi dòng `[]` (không vùng bấm nào < 44 × 44, trừ `data-inline`); ảnh 375 px của route FAIL |
| TC-00-65 | AC15 | SV B, 375 | **B** Từng phần tử v4 lỗi: 11 chip chủ đề Threads (cao 40), ô tích "Nhờ AI trả lời…" (14 × 16), `Báo cáo` (76 × 40), `Hỏi trợ lý AI` (117 × 40), bộ lọc Thư viện, Tuần / Tháng / Danh sách, liên kết quay lại "Threads" / "Thoát về Luyện đề" / "Kết quả của tôi" / "Hôm nay" / "Luyện đề" | Mỗi hộp bấm được ≥ 44 × 44 (ô tích tính theo vùng nhãn); bấm vào chữ nhãn tích / bỏ tích ô |
| TC-00-66 | AC15 (chống lách) | SV B, 375 | **B** Console: `[...document.querySelectorAll('[data-inline]')]` ở mọi route SV | `data-inline` chỉ trên liên kết **nằm giữa câu chữ** (có chữ đứng trước hoặc sau trong cùng khối); không trên nút, chip, tab, liên kết quay lại, nút ở hàng riêng |
| TC-00-67 | AC15 (nhánh) | SV B, GV; 375 | **A** kịch bản tương tác + `?state=`: Dialog hai lối, drawer Thêm, popover bộ chọn lớp, dải chip đã cuộn; `/inbox` sau mở ticket | `TOUCH` chạy trong các trạng thái đó cũng `[]` (nút `Đóng`, mục popover, `← Hộp thư`) |
| TC-00-68 | FR-X13, FR-X15 (tĩnh) | Cây sạch | **C** `tc_00_16` | Có `frontend/src/mock/derive.ts` xuất `ticketStats`, `attentionSet`, `navBadges`, `ago`; không còn số cứng `overdue: [`, `answeredByAi`, `escalated: [`; shell / ui không `color-mix(… transparent` |

## Truy vết hồi quy E1–E37 → TC (spec v5.1, SRS 4.10)
Mỗi lỗi QC thăm dò (`qc/explore-v4.md`) có ít nhất một TC chính và một TC nhánh lỗi / biên. Mức theo 4.10.

| E | Mức | TC chính | TC nhánh lỗi / biên |
| --- | --- | --- | --- |
| E1 `/practice` `Xem kết quả` sập | cao | TC-01-88, 90 | TC-01-91, 94, 95; TC-00-56…59 (màn lỗi tiếng Việt) |
| E2 SV D thấy dữ liệu lớp / phiên của B | cao | TC-01-98, 99 | TC-01-100…107 |
| E3 thẻ 761988 mở thành viên 761987 | cao | TC-02-50, 51 | TC-02-52, 53, 55 |
| E4 công bố BT03 không báo chuông SV | cao | TC-01-135 | TC-01-136, 137, 138 |
| E5 email / SĐT không qua hộp PII | cao | TC-01-108, 109 | TC-01-110…120 (P1–P4, bảng, GV / TA, rò rỉ) |
| E6 "chờ > 24 h": 1 ≠ 3 | cao | TC-04-35 | TC-02-89, TC-04-37 |
| E7 `/attendance` 390: ô Buổi co, đè nhau | cao | TC-02-64 | TC-02-68 |
| E8 `/attendance` 390: cột tên 89 px, cột ngoài màn | cao | TC-02-65 | TC-02-66, 67 |
| E9 thẻ "chờ 3 ngày" mở sai phiếu | vừa | TC-02-57 | TC-02-58, 59 |
| E10 `/documents` ≠ `/library` | vừa | TC-03-35, 37 | TC-01-140, 141; TC-03-41 |
| E11 câu 1 luyện đề trùng QUIZ01 | vừa | TC-01-96 | TC-01-97 |
| E12 "Cần chú ý": 8 / 2 / 3 | vừa | TC-02-74 | TC-02-75…77 |
| E13 chủ đề mặc định "Thông báo" | vừa | TC-01-121 | TC-01-128 |
| E14 thread mới: meta "Câu hỏi của bạn" | vừa | TC-01-122 | TC-01-123 |
| E15 `Hỏi trợ lý AI` gợi ý sai chủ đề, chèn vào ô | vừa | TC-01-124 | TC-01-125, 126 |
| E16 GV lưu câu AI rỗng vẫn "đã sửa" | vừa | TC-02-72 | TC-02-73 |
| E17 `Tạo thread ghim` nhiều lần | vừa | TC-04-40 | TC-04-41, 42 |
| E18 điểm danh: công tắc / nút lưu | vừa | TC-02-69, 70 | TC-02-71 |
| E19 badge sidebar không giảm | vừa | TC-02-60, 61 | TC-02-62, 63 |
| E20 `/chat` 390 không mở phiên cũ | vừa | TC-01-131, 132 | TC-01-133 |
| E21 bảng 390: cột chính khuất | vừa | TC-03-30, 32 | TC-03-31, 33, 34 |
| E22 vùng chạm < 44 px (SV 390) | vừa | TC-00-64, 65 | TC-00-66, 67 |
| E23 lề trái 348 / 240 / 248 | vừa | TC-00-49, 50 | TC-00-51, 52 |
| E24 lịch tuần 175 px, chữ ngắt | thấp | TC-01-142 | TC-01-143 |
| E25 "AI trả lời 94%" ≠ 6/392 | vừa | TC-04-36 | TC-04-37…39 |
| E26 chuông "12 ngày trước", tên tài liệu lệch | thấp | TC-00-60, 61 | TC-00-62, 63; TC-01-134, 141 |
| E27 `/me` "Cập nhật 09:20" đứng yên | thấp | TC-01-144 | TC-01-145 |
| E28 thứ tự SV khác nhau | thấp | TC-02-78 | TC-02-79 |
| E29 thẻ "4 bài" sai lý do | thấp | TC-02-80, 81 | TC-02-82 |
| E30 mốc phân công 08:30 / 16:40 | thấp | TC-04-43, 44 | TC-00-60, 61 |
| E31 thanh trên bán trong suốt | thấp | TC-00-53, 54 | TC-00-55 |
| E32 nút gửi mờ, không nói thiếu gì | thấp | TC-01-128, 129 | TC-01-130 |
| E33 `/documents`: tên tệp thô, "Mordern" | thấp | TC-03-35, 36 | TC-03-39, 40 |
| E34 form `/threads` chiếm ~530 px đầu | thấp | TC-01-127 | TC-01-128 |
| E35 `/settings/llm` khoảng cách các phần | thấp | TC-04-45 | TC-04-50 |
| E36 thẻ "Câu hỏi gốc" trống 80 px | thấp | TC-01-146 | TC-01-152 |
| E37 biểu đồ "Hoạt động học" không đọc được | thấp | TC-04-46, 47 | TC-04-48 |
| J1, J2 (PM chốt, 01-AC28) | – | TC-01-147, 150 | TC-01-148, 149, 151 |
| J3 (02-AC21) | – | TC-02-83, 84 | TC-02-85…87 |

FR-X13: TC-00-60…63, 68; TC-02-60, 74, 78; TC-04-35…39, 43, 49 · FR-X14: TC-00-64…67 · FR-X15: TC-00-49…55, 68 · FR-X16: TC-00-56…59 · FR-X17: TC-01-135…139, TC-02-50…56, 88 · FR-X18: TC-01-98…120 · FR-X19: TC-01-128…130.

## Công cụ bổ sung (spec v5, v5.1)
**A** = `docs/sprints/1.5/qc/scripts/audit.mjs` (Eval JS, global `browser`): chạy đoạn `AUDIT` nguyên văn của US.md trên mọi route × vai × bề rộng, thêm phép đo riêng cho SRS 4.7 (brand, bộ chọn lớp, menu hồ sơ, chat, inbox, provider, bảng danh sách). **v5.1:** ma trận có thêm ba dòng mỗi route: `TOUCH (00-AC15)` (nguyên văn `TOUCH` của US.md, 375 / 390, 12 route SV + `/attendance`, `/inbox`), `LEFT (00-AC11)` (240 ở 1440, 16 ở 390), `HEADER đặc (00-AC12)` (sau cuộn 200 px); và `SPEC_V51` (404 / `ep_demo_state` hỏng, SV D, form Threads, chat 390, thẻ câu hỏi gốc, `/attendance` 375 / 390, danh sách sinh viên, badge, analytics, biểu đồ, `settings-section`, `/gradebook` 390). Xuất bảng PASS/FAIL (`table(rows)`, `summary(rows)`). Frontend không có Playwright nên dùng Puppeteer của `browser`, không thêm phụ thuộc. Cách chạy: đầu file script.
**T** = `scripts/threads-timeline.mjs` (đo bước thời gian Threads, dùng ở US-PROTO-01 / 02). **P** = `scripts/pii-matrix.mjs` (bảng nhận diện PII, US-PROTO-01). **C** v5.1: `tc_00_13…16`, `tc_01_17`, `tc_01_20`, `tc_01_24`, `tc_02_14`, `tc_02_18`, `tc_02_20`, `tc_03_08`, `tc_04_08`, `tc_04_10`.

## Nhánh lỗi
TC-00-14, 15, 16 (`?state=`); TC-00-01 (không cookie); thêm: cookie vai rác `ep_demo_role=hacker` → không 5xx (`tc_00_01`). Spec v5: TC-00-26 (375/414), 29 (ngưỡng 720 / 1100), 31 (D chưa có lớp), 35, 43, 44 (`?state=` vẫn đạt AUDIT), 47 (chống lách `data-scroll-x`). Spec v5.1: TC-00-52, 57, 58, 59 (404, `ep_demo_state` hỏng, màn lỗi), 62 (biên đồng hồ), 66 (chống lách `data-inline`), 67.

## Phân quyền
TC-00-17…21 (ma trận SRS mục 2, gõ thẳng URL bằng cookie vai khác).

## Kiểm chéo
- Phản mẫu UI: `ui-antipatterns.sh` (TC-00-09) + xem ảnh 1440 / 390 theo DESIGN §21, §22 (sweep `shell`, `/login`, màn chặn).
- 375/390 px: shell + `/login` + màn chặn không cuộn ngang; bottom nav ≤ 5 đích.
- Không PII/secret: storage chỉ `ep_demo_state` (TC-00-08).
- Không gọi mạng ra ngoài trừ font: DevTools Network không có host lạ (B).

## Điểm khó kiểm
- `curl` chỉ thấy HTML phía server; nội dung chỉ sinh ở client (localStorage) phải kiểm bằng trình duyệt (B).
- AC1 cho phép "HTML trả về là màn chọn vai" thay vì redirect: `tc_00_01` chấp nhận redirect; nếu không redirect, so tay.
- AC10: AUDIT chỉ bắt chữ bị cắt ngang và `…` thiếu `title`; không bắt chồng lấp (ô chọn co nhỏ đè lên hàng khác như v4 `/attendance`) hay chữ rớt từng chữ. Vì vậy TC-00-39, 41 có thêm phép đo / ảnh riêng; mỗi route 390 phải xem ảnh, không chỉ tin `AUDIT = 0`.
- AC10: `data-scroll-x` là cửa thoát của AUDIT; TC-00-47 chặn việc đánh dấu cả trang để qua.
- Tên ngưỡng 720 / 1100 lấy từ SRS 4.7; spec không nói gì cho 1100–1439 ngoài b1 (≥ 1100) nên không có TC riêng cho 1280.
- Phép đo của **A** dùng `data-part` do dev thêm; thiếu móc thì dòng FAIL ghi "null" — coi là lỗi của AC10 (US 00 mục "Móc đo").

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-02 · Thêm TC-00-22…48 (không sửa TC cũ): spec v5 — 00-AC7…AC10 (#17a, b, f, h; SRS 4.7 a1–a4, b1–b3, f1, h1–h4). Thêm script `audit.mjs`, hàm `tc_00_07…09`, `tc_00_hooks` trong `proto-curl.sh`.
- 2026-10-02 · Thêm TC-00-49…68 (không sửa TC cũ): spec v5.1 (#19) — 00-AC11 (E23, FR-X15), 00-AC12 (E31), 00-AC13 (E1, FR-X16), 00-AC14 (E26, E30, FR-X13), 00-AC15 (E22, FR-X14). `audit.mjs` thêm `TOUCH_SRC`, `LEFT_SRC`, `SPEC_V51`; `proto-curl.sh` thêm `tc_00_13…16`.

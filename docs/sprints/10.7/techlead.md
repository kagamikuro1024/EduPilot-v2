# Sprint 10.7 (UI-R): ghi chú khảo sát bàn giao (techlead-mem → techlead-ui)

Đây **chưa phải hợp đồng §23**. PM chuyển việc sang techlead-ui khi Tech Lead mới đọc mã xong, chưa soạn nháp. Các số dưới đây đếm bằng `grep` trên `origin/main` @`e9af9d15`. Ước tính là ước tính, chưa đối chiếu ảnh.

## Hiện trạng primitive (`frontend/src/shared/ui/`)

| Primitive | Tệp | API hiện tại | Vì sao mỗi route ra một kiểu |
| --- | --- | --- | --- |
| `Page` | `Layout.tsx`, `Layout.module.css` | `width?: "reading" \| "wide" \| "full"` (mặc định `reading` = 960 px, **căn trái**), `className?` | 47 chỗ không truyền `width` nên ngầm là `reading`. 32 chỗ `wide`, 9 `full`, 1 `reading`. 4 chỗ truyền `className`: chat ×2, inbox, `me/RealMe` |
| `PageHeader` | `Layout.tsx` | `title, description?, meta?, actions?, back?`; hàng tiêu đề `align-items: flex-end; flex-wrap: wrap` | Khoảng 16 chỗ có `description`, 15 chỗ có `meta`. `actions` là ReactNode tự do: không giới hạn số nút, và nút rơi xuống dòng dưới khi hẹp |
| `Section` | `Layout.tsx` | `title, description?, action?, panel?` | Khoảng 5 chỗ có `description` (dòng phụ). Ngoài ra 23 chỗ tự dựng `<h2 className="ep-section-title">` ngoài `shared/ui`, bỏ qua `Section` |
| `Panel` / `PanelSection` | `Panel.tsx`, `.module.css` | Bản thân không đặt chiều cao | Panel «cao hết màn» là do CSS route: `inbox/RealInbox.module.css:21–34` (`height: calc(100dvh …)`, `[data-ep-panel] { height: 100% }`), `chat/ChatScreen.module.css:9`, `grading/SubmissionReview.module.css:11,58` |
| `EmptyState` | `Feedback.tsx:51`, `Feedback.module.css:36` | `title, children?, action?, icon?`; CSS `justify-items: start; align-self: start; max-width: 56ch` | 96 chỗ gọi, khoảng 58 chỗ có thân chữ, 11 chỗ có icon. Mặc định **căn trái-trên**, nên ra lỗi bố cục (3) của `ui-empty` |
| `PageState` | `PageState.tsx` | Rỗng mặc định = `<EmptyState title="Chưa có gì ở đây">Khi có dữ liệu…</EmptyState>`. Chú thích ghi «câu dạy bước kế» | Mặc định trái luật chữ UI của PO |
| `Split` | `Layout.tsx` | `main, aside, ratio` | **Không route nào dùng.** Chat, inbox, duyệt bài đều tự dựng chia đôi bằng CSS riêng |

`meta` lặp lớp: thấy ở `inbox/RealInbox.tsx:103` (`meta={<span>{courseLabel}</span>}`). Còn cần quét hết.

Độ rộng tự đặt trong CSS route: 73 dòng `max-width: <số>` ở 41 tệp `*.module.css`. Nhiều nhất ở `exam/Exam`, `exam/take/Take`, `members/ClassSettings`, `chat/ChatScreen`, `practice/Practice`, `gradebook/Gradebook`. Không phải dòng nào cũng là khung: có cả cột chữ, ô nhập.

## Xung đột phải chốt trước khi viết §23

- **«`read` căn giữa» (plan.md) trái với 00-AC11.** AC này (`FEAT-ui-foundation/US.md:219` AC13, `FEAT-ui-panels/US.md:114` AC3) ghi: mép trái `h1` là 240 px ở 1440 trên **mọi** route, và `Layout.module.css:1` có chú thích «không căn giữa». Nếu chọn căn giữa thì BA phải vá ba AC đó, và QC phải đổi phép đo `LEFT` của `audit.mjs`.
- **DESIGN.md mâu thuẫn với luật PO.** §10.2 ghi «One short supporting sentence only if needed». §15 «Empty states: Teach the next action», kèm ví dụ «Good» là một câu giải thích dài. `UX.md:21` (quy tắc 5): «mỗi trạng thái rỗng nói *vì sao rỗng*». `UX.md:82` (cổng UX). Skill impeccable `operate.md` cũng nói «empty states that teach». Luật PO (CLAUDE.md 09/10/2026) thắng; cả bốn chỗ phải vá.
- **Script hiện có.** `scripts/ui-antipatterns.sh` có 23 phép. Phép 23 chỉ bắt `description` tĩnh > 40 ký tự. Đã có cơ chế `--selftest` (mỗi phép một hạt giống) nên luật mới cần thêm hạt giống vào mảng `seeds`.

## Hướng Tech Lead định đề xuất (techlead-ui tự quyết)

1. Ép bằng **kiểu TypeScript** trước, script sau:
   - `Page.width: "work" | "read"` bắt buộc, bỏ `className`;
   - bỏ `PageHeader.description` / `actions`, thay bằng `primary` (một nút) + `secondary` (tuple ≤ 2, dạng dữ liệu để thu vào menu khi < 720 px);
   - bỏ `Section.description`;
   - `EmptyState` chỉ còn `title` + `action?`, không `children`, không `icon`.

   `tsc` trong `pnpm build` sẽ chỉ ra mọi chỗ gọi.
2. Chia đôi cao hết màn chỉ qua `Split` (thêm `fill`, `ratio="list-detail"`). Xoá CSS chiều cao ở inbox, chat, duyệt bài.
3. Script chỉ bắt những gì kiểu không bắt được:
   - độ rộng ≥ 480 px / `vw` trong CSS route;
   - `height` / `min-height` `100%` / `100dvh` ngoài `Split`;
   - `meta` chứa `courseLabel` / `classCode` / `course?.code|name`;
   - `<h2 className="ep-section-title">` tự dựng;
   - nút rỗng trùng nhãn nút chính đầu trang trong cùng tệp.
4. Commit 1 chỉ đổi API primitive + sửa máy móc mọi chỗ gọi, để build xanh. Sau đó mỗi nhóm route một commit.

## Phân nhóm route (đề xuất kiểu: `work` / `read` / `work·split`)

| Nhóm | Route |
| --- | --- |
| G1 Hôm nay | `/` (3 vai): read |
| G2 Hỏi đáp | `/chat` work·split · `/inbox` work·split · `/threads` work · `/threads/[id]` read |
| G3 Lớp & người | `/students` work · `/students/[id]` work · `/class/members` work · `/attendance` work · `/class/settings` read · `/join`, `/join/[code]` read |
| G4 Đánh giá | `/grading` work · `/grading/[submissionId]` work·split · `/gradebook` work · `/gradebook/scheme` read · `/assignments/[id]` read · `/exams`, `/exams/[id]`, `/exams/[id]/results`, `/exams/[id]/similarity` work · `/exams/[id]/take` read · `/questions` work · `/practice` read · `/practice/[attemptId]` read · `/practice/history` work · `/me` read |
| G5 Nội dung | `/documents` work · `/library` work · `/library/[id]` read · `/calendar` work |
| G6 Hiểu lớp & hệ thống | `/insights` read · `/analytics` read · `/observability` work · `/settings` read · `/settings/llm` work · `/settings/integrations` read · `/admin/courses`, `/admin/users` work |
| G7 Ngoài app | `login`, `register`, `forgot-password`, `reset-password`, `verify-email`, `invite/[token]`, `error`, `not-found`, `RouteStub`: chỉ áp luật rỗng / tiêu đề |

Kết quả của persona-eval (`research/persona-eval`, §3.1) liên quan trực tiếp tới UI-R: trạng thái rỗng «Chưa có bài nộp.» sai nghĩa ở `/grading`, Markdown thô ở `/assignments/[id]`, hai nút «Tải bài lên» khi hàng chờ rỗng, nút ở «Hôm nay» lặp tiêu đề.

---

# TL-1 (techlead-ui, 2026-11-14): kế hoạch đánh giá và làm lại khung giao diện

Kế thừa ghi chú khảo sát ở trên (số đếm đã kiểm lại trên `origin/main` @`ebc60676`). Không có mã sản phẩm nào bị sửa. Tệp đi kèm:

| Tệp | Nội dung |
| --- | --- |
| `design-23-draft.md` | Nguyên văn nháp `DESIGN.md` §23 + các dòng cũ phải vá |
| `proposals.md` | P-107-1 … P-107-9: chỗ trái AC đã duyệt, việc cần PM / chủ dự án quyết |
| `mau/palette.md` | Bộ màu Bách khoa: mã chính thức, token, bảng tương phản |
| `mau/khung-trang.html` + `mau/khung/*.png` | Ba màn khung × bốn trạng thái, 1440 và 375 |
| `mau/mau-1.html`, `mau/mau-app.html` + `sau-web.png`, `sau-app.png` | «Hôm nay» giảng viên; app sinh viên điểm danh bằng mã QR |
| `mau/landing.html` + `mau/landing-*.png` | Trang giới thiệu (mục 8) |

## 0. Kết luận trước

1. Lỗi gốc là **primitive cho phép**: `Page` có 3 độ rộng + `className`, `PageHeader.actions` là JSX tự do, `EmptyState` nhận thân chữ + icon, `Section` có dòng phụ, chiều cao theo màn nằm trong CSS của route. Sửa từng màn sẽ tái phát. Phải **xoá khả năng đó khỏi API**, rồi đo bằng DOM.
2. Hợp đồng khung (§23): `Page` đúng 2 kiểu (`work` 1176 px ở 1440, `read` 800 px căn giữa); một mẫu đầu trang; panel rộng đúng cột, cao theo nội dung; trạng thái rỗng = tiêu đề + tối đa 1 nút, căn giữa.
3. Ba lớp ép: **kiểu TypeScript** (prop bị xoá → build đỏ ở mọi chỗ gọi cũ), **8 luật mới** trong `ui-antipatterns.sh` (24–31), **`frame.spec.ts`** đo hợp đồng trên mọi route × vai × 1440 / 375 × 4 trạng thái. Lớp thứ ba là lớp quyết định: «xong» = 0 độ lệch.
4. Thứ tự: lô 0 (primitive + sửa máy móc, build xanh) → lô 1 (3 màn chủ dự án chụp, dừng cho duyệt) → lô 2–7 theo nhóm route → lô 8 (bộ màu) → lô 9 (trang giới thiệu). Ước 7 ngày dev cho khung, 1,5 ngày cho màu và trang giới thiệu.
5. Bộ màu: mã chính thức lấy từ sổ tay nhận diện 2026 của trường (đỏ `#CE1628`, vàng `#F3C108`, đen `#231F20`). **Đỏ gần như trùng đỏ hiện tại**; thay đổi nhìn thấy là **nền trang trắng** (chủ dự án 14/11) và vàng điểm nhấn. Mọi cặp chữ / nền đạt AA, trừ chữ vàng (cấm).
6. Việc phải có quyết định trước khi dev bắt đầu (`proposals.md`): P-107-1 (cột `read` căn giữa trái 00-AC11), P-107-2 / -3 (trạng thái rỗng và lỗi trái `DESIGN` §15, `UX.md` quy tắc 5), P-107-6 (mẫu «Hôm nay» giảng viên gộp hai vùng phụ vào một panel), P-107-9 (trang giới thiệu: luật nào thắng khi `DESIGN` §11.1 và skill `design-taste-frontend` trái nhau), P-107-10 (nền trắng trái D59), P-107-11 (đổi lời trang giới thiệu).

## 1. Kế hoạch đánh giá

Nguyên tắc: mỗi tiêu chí là **một con số đọc từ DOM hoặc từ ảnh**, có ngưỡng, không có chữ «trông». QC đang đo K1–K6 bằng `frontend/e2e/qc-ui-audit-real.spec.ts` (đọc ở worktree `qc-106`, không sửa); cột «QC» ghi mã tương ứng, «mới» là phép cần thêm.

### 1.1 Ma trận

| Chiều | Giá trị |
| --- | --- |
| Route × vai | Theo bảng `ROUTES` của QC: sinh viên 13, trợ giảng ≈ 19, giảng viên ≈ 22, admin 7 (chỉ cặp vai được phép vào) |
| Bề rộng | 1440 × 900 và 375 × 812 |
| Trạng thái | có dữ liệu (lớp 761987) · rỗng (lớp `QCEMPTY`) · đang tải · lỗi |
| Đang tải, lỗi | Dựng bằng Playwright, không đụng mã app: `page.route('**/api/v1/**', …)` giữ phản hồi 5 s rồi chụp ở 800 ms (đang tải); trả 500 (lỗi). Cho qua `/api/v1/auth/**` và `/api/v1/me`. Bố cục hai trạng thái này không phụ thuộc vai → chỉ chạy **một vai mỗi route** |

Tổng: khoảng 230 lượt (dữ liệu + rỗng) + 180 lượt (tải + lỗi). Mỗi lượt lưu một dòng JSON + một ảnh.

### 1.2 Tiêu chí

Sai số ±1 px nếu không ghi khác. «Cột» = vùng nội dung của `Page` (đã trừ đệm).

| Mã | Đo gì | Cách đo | Đạt khi | QC |
| --- | --- | --- | --- | --- |
| **W1** | Kiểu trang | đếm `main [data-ep-page]` | đúng 1, giá trị `work` hoặc `read` | K1 |
| **W2** | Bề rộng cột ở 1440 | `getBoundingClientRect` | `work` = 1176; `read` = 800 và lề trái = lề phải | K1 |
| **W3** | Mép panel | mọi `[data-ep-panel]` cấp cao nhất | `left` = mép trái cột, `right` = mép phải cột (trong `Split`: hợp hai pane phủ đúng cột) | K1 |
| **W4** | Mép tiêu đề và nút | `h1.left`, `actions.right` | = mép trái / phải cột | K2 |
| **W5** | 375 | cột, `scrollWidth − innerWidth` | cột 12 → 363; tràn ngang = 0 | K6 |
| **H1** | Tiêu đề | đếm `h1` | đúng 1, nằm trong `[data-ep-page-header]` | K2 |
| **H2** | Chữ thừa ở đầu trang | con của header | chỉ có: liên kết quay lại, `h1`, meta, cụm nút. 0 thẻ `<p>` khác | K2 |
| **H3** | Meta | cao / `line-height`; so chữ với `[data-part=topbar]` | ≤ 1 dòng ở 1440; không chứa mã lớp, tên lớp đang chọn | K2 |
| **H4** | Nút cùng hàng tiêu đề | tâm dọc cụm nút − tâm dọc `h1` | ≤ 4 px ở 1440 | K2 |
| **H5** | Số nút đầu trang | nút thấy được trong `[data-ep-header-actions]` | ≤ 1 `primary` (ngoài cùng bên phải) + ≤ 2 phụ + ≤ 1 nút «Thêm» | K2 |
| **H6** | Nút chính lạc chỗ | `[data-variant=primary]` ngoài header | 0 khi trang có 1 vùng; ≤ 1 mỗi vùng khi nhiều vùng | K2 |
| **H7** | Chiều cao đầu trang | `header.height` gom theo «có meta / không meta» | mỗi nhóm đúng **một** giá trị trên toàn app (±2) → thân trang bắt đầu ở cùng toạ độ | mới |
| **H8** | Đầu trang ở 375 | cao hàng nút | 1 dòng (≤ 48 px) | mới |
| **P1** | Đệm panel | `padding` tính được | 24 (16 ở 375); 0 chỉ ở panel của `Split` | K1 |
| **P2** | Panel cao theo nội dung | `panel.height` so tổng cao con + đệm | chênh ≤ 2 px. Ngoại lệ `[data-ep-split=fill]`: đáy panel = đáy màn − 24 | K1 |
| **P3** | Lồng, số panel mỗi vùng | `[data-ep-panel] [data-ep-panel]`; panel trong mỗi `[data-ep-section]` | 0; đúng 1 (đã có ở `panels.spec.ts`) | – |
| **P4** | Tiêu đề mục | mọi `main h2` | có `data-ep-section-title`; phần đầu mục không có `<p>` | K3 |
| **P5** | Nhịp dọc | khoảng cách đầu trang → thân, mục → mục, `h2` → panel | 40 / 48 / 16 (ở 375: 32 / 48 / 16) | mới |
| **E1** | Hình khối rỗng | con của `[data-ep-empty]` | đúng 1 tiêu đề + ≤ 1 nút; 0 `svg` ngoài nút; 0 chữ khác | K4 |
| **E2** | Căn giữa | tâm ngang khối − tâm ngang vật chứa | ≤ 2 px (QC đang dùng 40) | K4 |
| **E3** | Tiêu đề rỗng | độ dài, số dòng | ≤ 40 ký tự, 1 dòng ở 1440, không dấu chấm cuối | mới |
| **E4** | Nút rỗng trùng | nhãn nút rỗng so nhãn nút đầu trang, tab, nút thanh công cụ | không trùng | K4, K5 |
| **E5** | Kiểu nút rỗng | `data-variant` | `secondary` | K4 |
| **E6** | Panel khi rỗng | cao vật chứa ở 1440 | ≤ 320 px | K4 |
| **E7** | Rỗng tự dựng | route ở lớp rỗng mà 0 `[data-ep-empty]` | liệt kê, xem ảnh từng cái | K4 |
| **L1** | Đầu trang khi đang tải | `h1` rect so trạng thái có dữ liệu | bằng nhau | mới |
| **L2** | Khung xương | `[aria-busy=true]` | nằm trong panel cùng mép với panel có dữ liệu; 0 spinner | mới |
| **L3** | Nhảy bố cục | CLS từ đang tải → có dữ liệu | ≤ 0,1 | mới |
| **X1** | Khối lỗi | `[role=alert]` trong vùng | đúng 1, rộng = cột, có nút «Thử lại» | mới |
| **X2** | Chữ lỗi | nội dung | không mã trần (`4xx`, `5xx`, `undefined`, `Error:`); ≤ 1 câu khi màn không có ô nhập | mới |
| **X3** | Khung khi lỗi | header, tab so trạng thái có dữ liệu | giữ nguyên | mới |
| **D1** | Cỡ chữ | tập `font-size` tính được trong `main` | ⊆ {28, 24, 20, 16, 15, 14, 12} px | mới |
| **D2** | Cao điều khiển | nút, ô nhập, ô chọn | 36 hoặc 30 ở 1440; ≥ 44 ở 375 (`TOUCH` đã có) | mới |
| **D3** | Cao hàng bảng | `tbody tr` | 48 hoặc 44 | mới |
| **T1** | Icon giải thích | `svg.lucide-info`, `svg.lucide-circle-help` trong `main`, ngoài `InlineNotice` | 0 | mới |
| **T2** | Câu lặp | chuỗi > 30 ký tự giống hệt nhau trong `main` | < 3 lần | mới |
| **C1** | Tương phản | các cặp ở `mau/palette.md` mục 3 | đạt ngưỡng (`contrast.spec.ts`) | – |
| **C2** | Diện tích đỏ | số điểm ảnh trong ảnh chụp cách `--ep-red` ΔE < 10 / tổng | < 8 % | mới |
| **C3** | Vàng | phần tử có `background-color` = `--ep-yellow` trong một khung nhìn; phần tử có `color` vàng | ≤ 3; 0 | mới |

### 1.3 So chéo

Mỗi lượt sinh một **chữ ký khung**: `(kiểu, h1.left, cột.left, cột.width, có meta?, đệm panel, hình khối rỗng)`.

- Đạt khi ở 1440 toàn app có **≤ 4 chữ ký** (2 kiểu × có / không meta), ở 375 có **≤ 2**.
- `audit.md` thêm bảng «giá trị → danh sách route» cho từng thành phần của chữ ký, sắp theo số route giảm dần. Route lạc nằm cuối bảng, không cần nhìn ảnh để tìm.
- Ảnh: trước khi chụp, chèn hai đường gióng 1 px ở mép trái / phải cột mong đợi (mẫu: `khung-trang.html?guides=1`). Lưới ảnh `audit.html` xếp **theo kiểu** (mọi màn `work` cạnh nhau), không xếp theo vai, để mắt người thấy ngay màn nào lệch đường gióng.

### 1.4 Đề nghị QC bổ sung (qua PM, P-107-7)

1. Đọc **móc DOM ổn định** thay cho hậu tố lớp CSS Module (`page`, `meta`, `empty`…): sau lô 0 các lớp đó đổi tên, script hiện tại sẽ mù. Danh sách móc ở mục 3.
2. Thêm hai trạng thái đang tải / lỗi theo 1.1.
3. Thêm các phép «mới» ở 1.2 và bảng chữ ký 1.3; siết E2 từ 40 px xuống 2 px.
4. Chụp cả trang (`fullPage`) cho trạng thái có dữ liệu ở 375, vì lỗi nhịp dọc nằm dưới nếp gấp.

### 1.5 Cổng thường trực

Sau UI-R, phần đo của QC chuyển thành `frontend/e2e/frame.spec.ts` (chạy trên build giả lập như `panels.spec.ts`, không cần stack thật; trạng thái rỗng dùng fixture). Trong lúc sửa nó chạy ở chế độ báo cáo; hết lô 7 thì thành chặn trong `/gate`. Route mới về sau tự vào ma trận vì spec duyệt bảng điều hướng `nav.ts`.

## 2. Hợp đồng khung trang

Nguyên văn: `design-23-draft.md`. Các con số chính ở 1440 (thanh bên 216):

| | `work` | `read` |
| --- | --- | --- |
| Cột nội dung | 240 → 1416 (1176 px), trần 1600 px | 428 → 1228 (800 px), căn giữa |
| Dùng khi | có `DataTable`, lưới (lịch, điểm danh, sổ điểm) hoặc `Split` | còn lại: báo cáo, biểu mẫu, cài đặt, danh sách một cột |
| Ở 375 | 12 → 363 | 12 → 363 |

Ba quyết định thiết kế cần chủ dự án nhìn mẫu rồi gật:

1. **`read` căn giữa, 800 px.** Tiêu đề lệch 188 px giữa màn `work` và màn `read` khi chuyển trang. Đổi lại không còn panel hẹp dính trái với khoảng trống 376 px bên phải (đúng lỗi chủ dự án chụp ở `/insights`). 800 thay cho 960 hiện tại: 960 trên vùng 1176 chỉ hở mỗi bên 108 px, đọc ra là «gần đầy nhưng lệch», không phải một kiểu riêng. So `khung/work-data-1440.png` với `khung/read-data-1440.png`.
2. **Nút chính luôn ở đầu trang** khi trang có một vùng. Hiện `/grading` đặt «Chấm tất cả» trong thanh công cụ của panel, `/insights` đặt dưới meta, `/questions` đặt ở góc phải: ba màn ba chỗ.
3. **Khối rỗng không có nút nếu đầu trang đã có nút đó.** `khung/work-empty-1440.png` (không nút) so với `khung/work-filtered-1440.png` (một nút «Xoá bộ lọc», vì không nơi nào khác làm được).

## 3. Thay đổi primitive (`frontend/src/shared/ui/`)

Số chỗ gọi đếm trên `frontend/src` (kể cả `Demo*`, trừ `app/dev`).

| Primitive | API hiện tại | API mới | Chỗ gọi phải sửa | Route không tự đặt được gì |
| --- | --- | --- | --- | --- |
| `Page` | `width?: "reading" \| "wide" \| "full"`, `className?` | `kind: "work" \| "read"` (**bắt buộc**), `className?: never` | 90 thẻ ở 55 tệp (47 ngầm `reading`, 32 `wide`, 9 `full`, 4 có `className`) | Độ rộng, đệm, căn lề. Hai token `--ep-page-work` / `--ep-page-read` chỉ `Layout.module.css` đọc |
| `PageHeader` | `title, description?, meta?, actions?, back?` | `title, back?, meta?: readonly ReactNode[]` (≤ 3 mẩu dữ liệu, tự nối bằng « · », một dòng, cắt bằng dấu ba chấm), `primary?: HeaderAction`, `secondary?: readonly [HeaderAction?, HeaderAction?]`, `more?: HeaderAction[]` | 91 thẻ: 17 có `description`, 13 có `meta`, 26 có `actions` | Vị trí, số lượng, thứ tự nút; câu mô tả; meta nhiều dòng |
| `HeaderAction` (mới) | – | `{ label, icon?, onClick? \| href?, disabled?, reason?, busy?, menu?: MenuItem[] }`. `reason` = lý do khoá, hiện bằng tooltip sẵn có của `Button`. `menu` = nút mở `Menu` (vd. «Tạo câu hỏi») | 26 chỗ `actions`: 18 `Button`, 7 `ButtonLink`, 1 `Popover`, 1 `OverflowMenu`; 1 `Select` + 1 `SegmentedControl` chuyển xuống `Toolbar` | Ở < 720 px primitive tự gom nút phụ vào một nút «Thêm» (làm được vì nút là dữ liệu, không phải JSX) |
| `Section` | `title?, description?, action?, panel?: boolean \| "lg" \| "none", className?, id?, part?` | `title, action?, id?, part?`; luôn bọc con trong **một** `Panel` | 5 chỗ có `description` (3 ngoài `app/dev`: `insights`, `today/StudentToday`, `threads/DemoThreads`); 99 chỗ `panel` bỏ prop; 0 chỗ dùng `"lg"` / `"none"` | Dòng phụ; chọn có panel hay không; `className` |
| `Panel` | `as?, padding?: "md" \| "lg" \| "none", aria-label?` | `as?, aria-label?` | 2 chỗ `padding="none"` (đều là chia đôi → chuyển sang `Split`) | Đệm. Chiều cao: không ai đặt được vì `className` vốn đã cấm, và luật 25–26 chặn CSS route |
| `Split` | `main, aside, ratio?: "wide-main" \| "half"`; **0 chỗ dùng** | `ratio: "main-aside" \| "half" \| "list-detail"`, `fill?: boolean`, `detailOpen?: boolean` (dưới 1100 px: danh sách → chi tiết). `list-detail` và `fill` tự bọc một panel sát mép; `fill` tự tắt khi danh sách rỗng / lỗi | `chat` (2), `inbox`, `grading/SubmissionReview`, `gradebook/GradeScheme`, `today/StaffToday`, `exam/take`, `practice/PracticeRun` | Chiều cao theo màn; tỉ lệ cột; đường kẻ dọc |
| `EmptyState` | `title, children?, action?, icon?` | `title: string, action?: ReactNode`; CSS: căn giữa, đệm dọc `--ep-space-16`, bỏ `max-width: 56ch` và `justify-items: start` | 96 thẻ: 65 có thẻ đóng (có thân), 11 có `icon`, 28 có `action` | Thân chữ, icon, căn lề |
| `PageState` | `empty?: ReactNode`, mặc định có câu «Khi có dữ liệu…»; lỗi mặc định hai câu | `empty?: { title: string; action?: ReactNode }`, mặc định `{ title: "Chưa có dữ liệu" }`; `error?: { problem: string }`; lỗi mặc định `Không tải được nội dung` + «Thử lại»; `unsaved?: boolean` thêm câu «Chữ bạn đã nhập vẫn còn» | 36 chỗ truyền `empty={…}` | Hình khối rỗng và khối lỗi |
| `Toolbar` | `children, end?, alignEnd?` | giữ. Thêm luật: `end` không chứa nút `primary` khi trang có một vùng (đo bằng H6) | `grading/GradingQueue` và các màn H6 chỉ ra | – |

**Móc DOM ổn định** (primitive tự gắn; QC và `frame.spec.ts` chỉ đọc các móc này):

`data-ep-page="work|read"` · `data-ep-page-header` · `data-ep-header-meta` · `data-ep-header-actions` · `data-ep-section` · `data-ep-section-head` · `data-ep-section-title` · `data-ep-panel` (đã có) · `data-ep-split="main-aside|half|list-detail"` + `data-ep-split-fill` · `data-ep-empty` · `data-ep-toolbar`. Giữ `data-part="page-title"` trên `h1` (các spec cũ đang dùng).

**Vì sao là kiểu trước, script sau:** `pnpm -C frontend build` chạy `tsc`. Xoá prop → mọi chỗ gọi cũ thành lỗi biên dịch có tên tệp và dòng, không sót, không cần regex. Script chỉ lo những gì kiểu không thấy (CSS của route, nội dung chuỗi).

## 4. Luật mới cho `scripts/ui-antipatterns.sh` (24–31)

Cùng cơ chế hiện có: mỗi phép in `✓` / `✗`, miễn bằng `ui-allow: <lý do>` cùng dòng (trần 10 chỗ giữ nguyên), mỗi phép có hạt giống trong `--selftest` (kỳ vọng `31 / 31 phép bắt được`). «CSS route» = `*.module.css` dưới `features/` và `app/`, trừ `features/landing/` và `app/dev/`.

| # | Tên phép | Bắt | Biểu thức (ERE) | Hạt giống |
| --- | --- | --- | --- | --- |
| 24 | Khung tự đặt độ rộng | CSS route đặt bề rộng cỡ khung, hoặc tự căn giữa | `(max-\|min-)?width[[:space:]]*:[^;]*(([6-9][0-9]{2}\|[0-9]{4,})px\|[3-9][0-9](\.[0-9]+)?rem\|[0-9]+vw)` và `margin(-inline)?[[:space:]]*:[^;]*\bauto\b`. Ngưỡng 600 px: phân bố hiện tại có các giá trị 320–560 px (ô nhập, cột chữ) không bị bắt; 11 dòng từ 640 px trở lên (640 ×5, 720 ×2, 820 ×2, 1180 ×2) bị bắt, dev xét từng dòng. Giá trị tính bằng `ch` không bị bắt | `.a { max-width: 840px; }` |
| 25 | Chiều cao theo màn ngoài `shared/` | CSS route dùng `vh` / `dvh` / `svh` | `(min-\|max-)?height[[:space:]]*:[^;]*[0-9](d\|s\|l)?vh`. Hiện dính: `inbox`, `chat`, `grading/SubmissionReview`, `exam/take`, `practice` (5 tệp) | `.b { height: calc(100dvh - 56px); }` |
| 26 | Với tay vào primitive | CSS route chọn phần tử của primitive | `data-ep-(panel\|page\|empty\|section\|split)\|\.page\.page`. Hiện dính: `inbox/RealInbox.module.css`, `chat/ChatScreen.module.css` | `.c > :global([data-ep-panel]) { height: 100%; }` |
| 27 | Meta lặp lớp | `meta=` của `PageHeader` chứa lớp đang chọn | perl trên thẻ mở `<PageHeader …>`: thuộc tính `meta` khớp `course(Label\|Code\|Name)\|class_?[cC]ode\|course\??\.(label\|code\|name\|class_code)`. Hiện dính: `inbox`, `insights`, `analytics`, `observability` | `<PageHeader title="x" meta={[course.label]} />` |
| 28 | Tiêu đề tự dựng | `h1` / `h2` ngoài primitive | `<h1\b\|<h2\b\|ep-page-title\|ep-section-title` ngoài `shared/ui/`. Hiện 23 chỗ `ep-section-title` + 4 chỗ `h1` | `<h2 className="ep-section-title">x</h2>` |
| 29 | Nút rỗng trùng đầu trang | cùng tệp: nhãn trong `action` của `EmptyState` / `empty={{…}}` trùng `label` của `primary` | perl gom `primary={{ label: "…"` rồi tìm chuỗi đó trong các khối rỗng. Trần: không thấy qua ranh giới tệp → phần đó do E4 | tệp có `primary={{ label: "Tạo câu hỏi" }}` và `<EmptyState title="x" action={<Button>Tạo câu hỏi</Button>} />` |
| 30 | Tiêu đề rỗng là câu | `title` tĩnh của `EmptyState` / `empty={{ title }}` dài hơn 40 ký tự, hoặc có `. ` giữa chuỗi, hoặc kết thúc bằng `.` | perl như phép 23 | `<EmptyState title="Chưa có gì. Khi có dữ liệu nó sẽ hiện ở đây." />` |
| 31 | Vàng làm chữ | `--ep-yellow` dùng cho chữ / viền / icon | `(^\|[^-])(color\|border(-[a-z]+)?-color\|stroke\|fill)[[:space:]]*:[^;]*--ep-yellow\b` (không khớp `background-color`) | `.d { color: var(--ep-yellow); }` |

Phép 23 hiện có (câu mô tả > 40 ký tự ở `description=`) thành thừa sau lô 0 vì prop không còn; **giữ nguyên**, không xoá (nó là AC của `FEAT-uiux-91` US-01 AC6).

Giới hạn nói thẳng: grep không thấy độ rộng sinh ra từ `grid-template-columns` hay từ nội dung. Phần đó chỉ `frame.spec.ts` (W2, W3) bắt được. Script là lưới thô chạy trong 2 giây; spec là cổng.

## 5. Lô sửa route

Đơn vị ước lượng: ngày làm việc của `dev` (một phiên agent liền mạch). Mỗi lô một commit `10.7: lô N <nhóm>`; mỗi lô kết thúc bằng: build xanh, `ui-antipatterns` sạch, `frame.spec.ts` lọc theo route của lô = 0 lệch, ảnh trước / sau lấy từ `audit.html` lọc theo route (trước = bản chụp hiện trạng của QC, sau = chụp lại cùng script). Cột «Số chỗ» đếm bằng `grep` trong thư mục feature của lô (`<EmptyState`, `description=`, ` meta=`, dòng CSS có bề rộng từ 3 chữ số px): là cỡ việc, không phải danh sách lỗi.

| Lô | Route (kiểu) | Việc chính | Số chỗ | Ước |
| --- | --- | --- | --- | --- |
| **0** Primitive | – | Đổi API 7 primitive + `HeaderAction`; thêm móc DOM; sửa **máy móc** mọi chỗ gọi để build xanh (chưa chỉnh bố cục); luật 24–31 + hạt giống; `frame.spec.ts` chế độ báo cáo; sinh lại ảnh mốc `visual.spec.ts` | 88 `Page`, 91 `PageHeader`, 96 `EmptyState`, 99 `Section panel` | 1,5 |
| **1** Ba màn chủ dự án chụp → **DỪNG cho duyệt** | `/insights` (read) · `/questions` (work) · `/inbox` (work + `Split` list-detail fill) | Màn mẫu thật cho ba hình thái. `/inbox` là nơi dựng `Split fill` lần đầu | rỗng 6, meta 4, CSS chiều cao 1 tệp | 0,75 |
| **2** Hỏi đáp | `/chat` (work + Split fill) · `/threads` (work) · `/threads/[id]` (read) | Chuyển `chat` sang `Split`; bỏ `Page className` ×2 | rỗng 8, mô tả 6, meta 4, CSS rộng 14 dòng | 0,75 |
| **3** Chấm và điểm | `/grading` (work) · `/grading/[submissionId]` (work + Split half fill) · `/gradebook` (work) · `/gradebook/scheme` (work + Split main-aside) · `/assignments/[id]` (read) | «Chấm tất cả» lên đầu trang; hai nút «Tải bài lên» còn một; rỗng «Chưa có bài nộp.» sửa nghĩa (persona-eval) | rỗng 8, meta 4, CSS rộng 12 dòng | 1 |
| **4** Thi và luyện | `/exams` (read) · `/exams/[id]` (read, dev xác nhận) · `/exams/[id]/results`, `/similarity` (work) · `/exams/[id]/take` (work + Split main-aside) · `/practice` (read) · `/practice/[attemptId]` (work + Split main-aside) · `/practice/history` (read) · `/me` (read) | Nhiều khối rỗng có thân chữ nhất (22) | rỗng 27, mô tả 5, meta 8, CSS rộng 27 dòng | 1 |
| **5** Lớp và người | `/students` (work) · `/students/[id]` (work) · `/class/members` (work) · `/class/settings` (read) · `/attendance` (work) · `/join`, `/join/[code]` (read) | `members` có 11 dòng CSS rộng | rỗng 14, mô tả 3, CSS rộng 18 dòng | 0,75 |
| **6** Nội dung và Hôm nay | `/documents` (work) · `/library` (read) · `/library/[id]` (read) · `/calendar` (work) · `/` sinh viên (read), giảng viên / trợ giảng (work + Split main-aside), admin (read) | «Hôm nay» giảng viên theo `mau/mau-1.html` nếu P-107-6 được nhận | rỗng 16, mô tả 9, meta 11 | 0,75 |
| **7** Hiểu lớp, hệ thống, ngoài khung | `/analytics` (read) · `/observability` (work) · `/settings` (read) · `/settings/llm` (work) · `/settings/integrations` (read) · `/admin/courses`, `/admin/users` (work) · `error`, `not-found`, `RouteStub` (read) · trang tài khoản (`AuthShell`: chỉ áp luật chữ và khối rỗng) | Cuối lô: `frame.spec.ts` thành chặn | rỗng 9, meta 4 | 0,5 |
| **8** Bộ màu | toàn app | Đổi 8 token ở `tokens.css` + `DESIGN_TOKENS.css`; thêm `--ep-yellow`, `--ep-yellow-soft`; ba chỗ dùng vàng đầu tiên (chấm «cần xem kỹ», nhãn «cần chú ý», vạch «kế tiếp»); 3 tệp logo; `contrast.spec.ts` thêm cặp vàng; sinh lại ảnh mốc | 8 token, 3 svg | 0,5 |
| **9** Trang giới thiệu | `/` chưa đăng nhập | Theo mục 8 | 1 màn | 1 |

Tổng khoảng **8,5 ngày dev** (7 cho khung, 1,5 cho màu và trang giới thiệu) + QC chụp lại sau mỗi lô. Lô 8 tách sau lô 7 để ảnh trước / sau của lô 1–7 chỉ khác về khung; nếu chủ dự án muốn thấy màu sớm, lô 8 đổi lên ngay sau lô 0 được (chỉ đụng token).

Nhánh `fix/ui-empty` (BA đã vá SRS bỏ câu giải thích trong trạng thái rỗng) gộp vào trước lô 0.

## 6. Mẫu tham chiếu cho chủ dự án duyệt

HTML tĩnh tự chứa ở `docs/sprints/10.7/mau/`, mở bằng trình duyệt; đều dùng bộ màu mới.

| Mẫu | Chứng minh điều gì | Ảnh |
| --- | --- | --- |
| `khung-trang.html?m=work` · `/questions` | Kiểu `work`; nút chính ở đầu trang; thanh lọc trong panel; rỗng không nút; rỗng do lọc có một nút; đang tải giữ khung; lỗi một dòng | `khung/work-{data,empty,filtered,loading,error}-1440.png` |
| `khung-trang.html?m=read` · `/insights` | Kiểu `read` 800 px căn giữa; meta một dòng; tiêu đề mục không dòng phụ | `khung/read-{data,empty}-1440.png` |
| `khung-trang.html?m=split` · `/inbox` | `Split` cao hết màn **chỉ khi có dữ liệu**; rỗng thì panel thấp, không nút «Xem tất cả» | `khung/split-{data,empty}-1440.png` |
| cả ba ở 375 | Một cột, nút chính đủ rộng, nút phụ gom vào «Thêm» | `khung/375-work-data_work-empty_read-data_split-empty.png` |
| `mau-1.html` · «Hôm nay» giảng viên | `Split main-aside`; đỏ chỉ ở nút chính, mục đang chọn, số việc chờ; vàng ở 4 chỗ nhỏ | `sau-web.png` |
| `mau-app.html` · app sinh viên | Cùng token trên điện thoại (sprint 10.8) | `sau-app.png` |
| `landing.html` | Mục 8; có chuyển động, mở bằng trình duyệt | `landing-chuyen-dong.png`, `landing-man-dau-1440.png`, `landing-1440.png`, `landing-390.png` |

Thêm `&guides=1` vào URL của `khung-trang.html` để thấy hai đường gióng mép cột.

## 7. Bộ màu Bách khoa

Chi tiết: `mau/palette.md`. Tóm tắt: 6 token đổi giá trị (`--ep-red`, `--ep-red-hover`, `--ep-red-soft`, `--ep-ink`, `--ep-canvas`, `--ep-blue`), 2 token mới (`--ep-yellow`, `--ep-yellow-soft`), 9 token giữ nguyên. Nguồn: sổ tay nhận diện 2026 trang 4 và 6. Không dùng logo trường. Không tô đỏ thanh trên (khác `qldt.hust.edu.vn`) vì trái luật «đỏ là tín hiệu»; nếu chủ dự án muốn thanh trên đỏ thì đó là quyết định mở lại D28, ghi ở P-107-8.

**Nền trắng** (chủ dự án, 14/11, sau khi xem mẫu đầu): `--ep-canvas` = `#FFFFFF`. Hệ quả: nền trang và panel cùng trắng, panel chỉ còn tách bằng viền 1 px + bóng tĩnh; điều «nền trang tối hơn panel 5 điểm L» của D59 và phép đo ΔL ở `contrast.spec.ts` / `panels.spec.ts` phải đổi (P-107-10). Mọi mẫu ở mục 6 đã dựng lại trên nền trắng. Vì vậy §23.3 thêm một câu: panel luôn có đủ viền và bóng, không route nào được bỏ.

## 8. Trang giới thiệu (`/` chưa đăng nhập)

### 8.1 Hiện trạng đo được

Chạy bộ kiểm của skill `design-taste-frontend` trên bản đang chạy ở `ep-qc` (`http://localhost:580/`, không đăng nhập), 1440 và 390, ngày 2026-11-14:

| Công cụ | Kết quả |
| --- | --- |
| `consistency.mjs --page software` | **42 FAIL, 20 WARN** |
| `copy-lint.mjs --kind software` | 0 FAIL, 0 WARN (561 từ) |
| `perf.mjs` | 2 FAIL (9 tệp phông), 6 WARN; LCP, CLS, thời gian chặn đều đạt |
| `assets/layout/check.mjs` | không chạy được: trang không có cấu trúc `main > section` |

| Mã kiểm | Số | Nghĩa | Thật hay do công cụ |
| --- | --- | --- | --- |
| `scroll-linked` | 23 FAIL + 10 WARN | Phần tử chuyển động theo cuộn | **Trái luật nhau**: `DESIGN.md` §11.1 cho phép, skill cấm → P-107-9 |
| `container-edges` | 8 | Thanh trên chỉ có nút bên phải; mục «Xem EduPilot làm việc» lề 48 px trong khi các mục khác 160 px; không có một khung chứa chung | Thật. Cùng loại lỗi với «mỗi trang một kiểu panel» |
| `hero-screen`, `section-height` | 4 | Mục không cao đúng 1–2 màn | Một phần do công cụ (không có `main > section` nên coi cả trang là một khối); nhịp mục không đều là thật |
| `focus-visible`, `focus-ring` | 5 | Vòng focus của nút đọc ra 0 : 1; `.Chart…point:focus-visible` bỏ viền | Cần kiểm tay; cái thứ hai là thật |
| `loop-pause` | 1 | Đoạn phim nền lặp quá 5 giây, không có nút dừng | Thật (WCAG 2.2.2) |
| `copy-fact` | 2 | Màn đầu không có một dữ kiện cứng (con số, nền tảng, ngày) | Thật |
| `copy-headings` | 1 (6 tiêu đề) | «Hỏi riêng, có nguồn», «Khác biệt, có số đo»… không gọi tên vật | Thật |
| `copy-headline` | WARN | `h1` 14 từ (nên 2–8) | Thật |
| `body-size` | WARN | 8 / 12 đoạn chữ chạy 15 px (nên ≥ 16) | Thật |
| `tap-size`, `skip-link`, `share-meta`, `moment-none` | WARN | Liên kết chân trang 18 px; thiếu liên kết bỏ qua; thiếu thẻ chia sẻ | Thật, nhỏ |

Nhìn bằng mắt (ảnh `mau/landing-truoc-720.png`, thu 50 %): ảnh sản phẩm ở màn đầu chồng chữ lên nhau ở cột phải; con số thứ ba («+0,05 %») không phải lợi ích người dùng hiểu được; lời kêu gọi cuối trang trống.

### 8.2 Hướng sửa

Mẫu: `mau/landing.html`, **bản có chuyển động** (mở bằng trình duyệt để xem chạy). Ảnh: `landing-chuyen-dong.png` (12 khung hình của các chuyển động), `landing-man-dau-1440.png` (màn đầu), `landing-1440.png` (cả trang ở trạng thái đứng, tức là khi bật «giảm chuyển động»), `landing-390.png` (bảy màn ở 390). Dựng theo phương án (a) của P-107-9 và P-107-12: token và phông của `DESIGN.md` §11.1, bố cục và lời văn theo skill `design-taste-frontend`, chuyển động theo P-107-12, dùng bộ kiểm của skill làm thước đo (không dùng bộ CSS của skill, vì D53).

Bản đầu của mẫu là bản tĩnh; chủ dự án chê «đơn giản quá, hiệu ứng đâu» (đúng: trang giới thiệu phải thuyết phục, mà bản tĩnh chỉ sửa lỗi đo được). Bản này thêm chuyển động có việc để làm và phóng to phần sản phẩm.

| Lỗi đo được ở 8.1 | Cách sửa trong mẫu |
| --- | --- |
| `container-edges` ×8 | **Một** khung chứa `.wrap` cho thanh trên, sáu mục và chân trang: mép 80 → 1360 ở 1440, 24 → 366 ở 390. Lưới 12 cột, mọi khối bắt đầu và kết thúc trên đường cột |
| `hero-screen`, `section-height` | Trang là `header` + `main > section` ×5 + `footer.closing`; mỗi mục cao đúng một màn (`100svh`), riêng mục khung ghim cao hai màn khi đang ghim; tiêu đề là nội dung đầu tiên, căn trên |
| `copy-fact` | Màn đầu có một dữ kiện cứng, **nguyên văn nguồn**: «146 yêu cầu gửi AI: 0 lần lộ họ tên hoặc MSSV trong danh sách lớp» |
| `copy-headline` (14 từ) | `h1` còn 8 từ: «Vận hành lớp học có trợ lý AI»; nửa sau («bảo vệ dữ liệu cá nhân») chuyển thành câu thứ hai của đoạn dẫn |
| `copy-headings` ×6 | Tiêu đề gọi tên vật: «Bài nộp đi qua ba bước», «EduPilot trong ba màn», «Ba số đo trên dữ liệu mô phỏng», «Việc của giảng viên và sinh viên», «Lớp của bạn trên EduPilot». Không có dòng chú thích dưới tiêu đề mục |
| Màn đầu 3 nút | Còn 2 («Đăng nhập» có mũi tên, «Xem video»); «Tạo tài khoản» lên thanh trên và xuống lời kết |
| `body-size` (15 px) | Chữ chạy 16 px, đoạn dẫn 18 px |
| `tap-size`, `skip-link`, `share-meta` | Liên kết chân trang cao 44 px; có «Tới nội dung»; có thẻ `og:` |
| `focus-visible` | Vòng focus 2 px `--ep-red` lệch 3 px trên mọi điều khiển của trang |
| `loop-pause` | Đoạn sản phẩm chạy ở màn đầu có nút tạm dừng 32 px ở góc khung |
| Ảnh màn đầu chồng chữ | Màn đầu là **HTML thật** của màn duyệt bài ở cỡ đọc được, đang chạy (xem 8.3), kèm một thẻ hỏi đáp có nguồn nổi ở góc dưới |
| Ảnh sản phẩm nhỏ | Mục «EduPilot trong ba màn»: ảnh chụp thật rộng hết khung chứa (1280 px), ba tên màn xếp ngang phía trên |
| Mục vai chỉ có chữ | Hai tab «Giảng viên» / «Sinh viên»; bấm thì đổi danh sách việc và ảnh «Hôm nay» của vai đó |
| Lời kết trống | Lời kết + chân trang là **một** mục: tiêu đề, hai nút, ảnh Sổ điểm, dòng pháp lý |
| Bộ màu | Nền trắng toàn trang; đỏ chỉ ở nút chính, sợi chỉ nối ba bước, vạch màn đang xem, tab đang chọn, chấm «đã sửa»; vàng ở 2 chỗ nhỏ (vạch hai màu dưới `h1` và ở lời kết theo tỉ lệ dải đỏ / vàng của nhận diện trường, nhãn «Nháp») |

Giữ **nguyên văn** ba câu số đo và ba câu nguồn của chúng (US-UX91-08 AC9, kể cả dấu «:»), và vế «trong danh sách lớp» ở câu 146 (P-91-7(2)). Chỉ 146 đếm lên; «30/30» và «+0,05 %» đứng yên, như bản 9.1.

Mẫu **đổi lời** so với trang hiện tại ở ba chỗ (P-107-11 đã nhận): `h1` rút gọn; tiêu đề mục đổi tên; dòng «Ví dụ minh hoạ, số liệu mẫu.» dưới ba bước bị bỏ. Con số «1.000 sinh viên trên 20 lớp» của bản tĩnh đã rút khỏi màn đầu; mẫu không còn số nào ngoài nguồn AC9. Số trong các khối minh hoạ giao diện (8,0 / 7,5 / 8,5 / 9,0; 0,1 × 9,0 + 0,3 × 8,0 + 0,6 × 8,3 = 8,3) là dữ liệu vẽ, không phải số đo; dòng pháp lý cuối trang đã ghi «dữ liệu mô phỏng».

### 8.3 Chuyển động (hợp đồng cho lô 9, theo P-107-12)

Luật chung, đo được:

- Chỉ `transform` và `opacity` được chạy (đổi màu khi rê chuột là đổi tức thì, không chuyển tiếp).
- Không thư viện chuyển động: CSS `@keyframes` / `transition`, một `IntersectionObserver`, một lần đọc cuộn thụ động mỗi khung hình. Script của mẫu 51 dòng, 4,3 KB, gồm cả tab, hộp thoại video và nút tạm dừng.
- Phần hiện theo cuộn chỉ bị ẩn **sau khi** script chạy tới chỗ gắn bộ quan sát (lớp `m-armed`); script hỏng giữa chừng thì không có gì bị ẩn. Bộ quan sát không gọi lại trong 1,5 s thì hiện hết.
- Nội dung màn đầu hiện ngay từ khung hình đầu: `h1`, đoạn dẫn, nút, dòng số đo chỉ **dịch** (12–20 px), không bắt đầu từ `opacity: 0`. `h1` là LCP; đo bằng `perf.mjs`: LCP là dòng chữ của `h1`, thời gian chặn 0 ms.
- `prefers-reduced-motion: reduce` hoặc không có script: không gắn lớp `m-on`, mọi thứ đứng ở **hình đã xong**. Đã đo: 0 animation đang chạy, cả trang đọc được (`landing-1440.png` chính là ảnh của chế độ này).
- Cái gì tự chạy quá 5 giây thì có nút tạm dừng. Chỉ có **một** thứ tự chạy: đoạn sản phẩm ở màn đầu.
- Điện thoại (< 640 px): màn đầu vẫn vào, các mục vẫn hiện theo cuộn; đoạn sản phẩm đứng ở hình đã xong, không có con trỏ giả, không ghim.

| # | Ở đâu | Chuyển động | Để làm gì | Số đo |
| --- | --- | --- | --- | --- |
| 1 | Màn đầu, lúc tải | `h1` hai dòng dịch lên, lệch nhau 80 ms; vạch đỏ vàng vẽ từ trái; đoạn dẫn, nút, dòng số đo, thẻ hỏi đáp dịch lên lần lượt cách nhau 60 ms; khung sản phẩm từ 0,98 về 1 | Tên sản phẩm đến trước, rồi tới việc cần bấm | Xong trong 1,2 s, một lần |
| 2 | Màn đầu, khung sản phẩm | **Đoạn sản phẩm chạy**: bản AI chấm nháp → con trỏ tới ô điểm tiêu chí 2, bấm, 7,5 bị gạch và 8,5 hiện ra kèm chấm đỏ «đã sửa», điểm bài 8,2 → 8,5 → con trỏ tới nút «Duyệt», bấm, nhãn «Nháp» đổi thành «Đã duyệt» | Cho thấy lời hứa của sản phẩm trong 9 giây: AI soạn nháp, giảng viên quyết | Vòng 9 s; giữ hình đã xong 3,4 s; cuối vòng mờ còn 30 % rồi viết lại, không bao giờ trống; nút tạm dừng |
| 3 | «Bài nộp đi qua ba bước», khi cuộn tới | Sợi chỉ đỏ vẽ từ nút 1 tới nút 3 trong 1 s; sợi chỉ tới nút nào thì nút đó và thẻ của nó hiện | Ba bước là một trình tự, không phải ba ô ngang hàng | Một lần; thẻ cách nhau 500 ms |
| 4 | Ba thẻ bước, khi rê chuột | Thẻ diễn lại bước của nó: các dòng hiện lần lượt (thẻ 2: 8,5 thay 7,5 rồi nhãn «Đã duyệt») | Mời bấm thử; trả lời «bước này làm gì» | 300 ms mỗi dòng; chỉ chuột thật, không chạy trên màn chạm |
| 5 | «EduPilot trong ba màn» | **Khung ghim**: mục cao 2 màn, nội dung ghim bằng `position: sticky`; cuộn thì vạch đỏ trên tên màn chạy theo, ảnh đổi sang màn kế bằng mờ dần | Ba màn trong chỗ của một | Bám theo thanh cuộn, cuộn ngược thì chạy ngược; không chiếm quyền cuộn |
| 6 | «Ba số đo», khi cuộn tới | Ba vạch kẻ vẽ từ trái, chữ hiện theo; **146** đếm từ 0 | Con số là bằng chứng, đáng một nhịp dừng | Một lần; đếm 1 s |
| 7 | «Việc của giảng viên và sinh viên» | Bấm tab: vạch đỏ trượt sang tab mới, danh sách và ảnh đổi | Người xem chọn vai của mình | 240 ms; mũi tên trái / phải đổi tab |
| 8 | Mọi nút, liên kết | Nút nhích lên 1 px khi rê, lún 1 px khi bấm, mũi tên nhích 3 px; liên kết thanh trên vẽ gạch chân từ trái | Mọi thứ bấm được đều trả lời | 150 ms vào, 90 ms bấm |
| 9 | Lời kết, khi cuộn tới | Vạch đỏ vàng vẽ từ trái | Khép lại bằng đúng dấu hiệu mở đầu | 700 ms, một lần |

Không làm: chữ chạy ngang, trượt ảnh tự động, thẻ nghiêng theo chuột, nút hút chuột, hạt / tuyết / pháo giấy, chữ đổi màu gradient, kính mờ, phát sáng (`DESIGN.md` §11.1 và §21 vẫn cấm).

Khác bản đang chạy (lô 9 phải đổi): đoạn phim `gioi-thieu-clip-v1` ở màn đầu được thay bằng đoạn sản phẩm dựng bằng HTML (nét ở mọi cỡ màn, nhẹ hơn 200 KB phim, đọc được bằng trình đọc màn hình qua `aria-label`); «Xem video» vẫn mở phim 75 giây trong hộp thoại.

### 8.4 Bộ kiểm chạy trên mẫu

Cùng lệnh với 8.1, chạy trên `mau/landing.html` (bản có chuyển động), 1440 và 390, ngày 2026-11-14. Số FAIL qua các lượt: bản tĩnh 42 → 22 → 11; thêm chuyển động lên 28; sửa còn 20 → **18**.

| Công cụ | Trang hiện tại | Mẫu bản tĩnh | Mẫu có chuyển động |
| --- | --- | --- | --- |
| `consistency.mjs` | 42 FAIL, 20 WARN | 11 FAIL, 25 WARN | **18 FAIL, 17 WARN** |
| `copy-lint.mjs` | 0 FAIL, 0 WARN | 0 FAIL, 0 WARN | 0 FAIL, 0 WARN |
| `perf.mjs` | 2 FAIL, 6 WARN | 2 FAIL, 4 WARN | 4 FAIL, 3 WARN; LCP 576 ms ở 1440 (dòng chữ `h1`), 324 ms ở 390 (đoạn dẫn); thời gian chặn 0 ms |

Bản có chuyển động nhiều FAIL hơn bản tĩnh vì skill cấm đúng thứ P-107-12 cho phép. 18 FAIL, không giấu:

| Nhóm | Mã | Số | Vì sao còn | Xử lý |
| --- | --- | --- | --- | --- |
| Chủ ý, theo P-107-12 | `scroll-reveal` | 6 | Ba bước, ba số đo và vạch lời kết hiện khi cuộn tới (skill cấm mọi thứ hiện theo cuộn) | Miễn trừ, ghi lý do là P-107-12 |
| Chủ ý, theo §11.1 | `scroll-linked`, `section-air`, `empty-area` | 3 | Khung ghim: mục cao hai màn, nội dung ghim một màn, nên bộ kiểm thấy «61 % mục trống». Skill chỉ miễn cho khung ghim dựng bằng module của chính nó | Miễn trừ, ghi lý do là `DESIGN.md` §11.1 |
| Thật | `no-visual` | 3 | Mục «Ba số đo» chỉ có chữ (cả hai bề rộng); mục vai ở 390 ẩn ảnh để vừa một màn | Chủ dự án chọn: thêm hình cho mục số đo, hoặc nhận là dải số. Ở 390 cho ảnh hiện dưới danh sách nếu chịu mục dài hai màn |
| Thật | `one-heading` | 2 | Con số 48 px bị tính như tiêu đề: «30/30» và tiêu đề mục kế cùng hiện ở 5,5 % vị trí cuộn (ngưỡng 3 %) | Hạ số xuống 40 px, hoặc nhận: số to là chủ ý |
| Thật | `container-edges` | 1 | Ở 390 chữ trong thẻ lùi 25 px so với mép khung chứa | Ở điện thoại bỏ viền thẻ, chữ ra sát mép |
| Do mẫu | `links` | 2 | `#register` không có đích trong tệp tĩnh | Tự hết khi nối `/register` |
| Do công cụ | `copy-headings` | 1 | Bộ kiểm so tiêu đề với danh sách danh từ tiếng Anh và tên sản phẩm (`consistency.mjs` dòng 9362, 9417–9422); tiêu đề tiếng Việt không chứa «EduPilot» đều bị đánh | Miễn trừ |

`perf.mjs` 4 FAIL: «8 tệp phông» (ngưỡng của skill là 4; ba độ đậm × hai subset là điều kiện PM đã nhận ở góp ý #14 sprint 5); hai lần «yêu cầu hỏng» là ảnh bìa của phim, do mẫu trỏ ra ngoài thư mục khi chạy qua máy chủ tạm của bộ kiểm; «ảnh quá cỡ» ở 390 là **thật**: mẫu nhúng ảnh 1600 px cho mọi bề rộng, lô 9 phải dùng `srcset` với các bản 640 / 800 đã có trong `frontend/public/landing/`.

Bộ kiểm bố cục `assets/layout/check.mjs` và cổng `gate.mjs` của skill chưa chạy trên mẫu: chúng đòi trang dựng từ `starter.html` và bộ CSS của skill (xem P-107-9).

## 9. Rủi ro và cách lùi

| Rủi ro | Mức | Giảm |
| --- | --- | --- |
| Lô 0 đụng ~280 chỗ gọi trong một commit | Cao | Sửa máy móc, không đổi bố cục; `tsc` là lưới; `git revert` một commit là lùi được hết |
| `Split fill` thay CSS riêng của `chat` / `inbox` / duyệt bài, là ba màn dùng nhiều nhất | Cao | Lô 1 làm `/inbox` trước và dừng cho duyệt; `escalation.spec.ts`, `private-chat.spec.ts` phải xanh nguyên, không sửa assertion |
| Ảnh mốc `visual.spec.ts` (26 ảnh) đổi hết | Trung bình | Sinh lại ở lô 0 và lô 8, liệt kê từng ảnh + lý do trong handoff (quy tắc Q-QC-PU06-4) |
| Cột `read` 800 làm chật màn đang dùng 960 | Trung bình | Quy tắc chọn kiểu đẩy màn có bảng sang `work`; `--ep-page-read` là một token, đổi một chỗ |
| Spec cũ ghim `LEFT = 240` trên mọi route | Chắc chắn (P-107-1) | BA vá AC trước lô 0 |
| Bỏ thân chữ làm mất thông tin thật sự cần | Thấp | Luật chữ UI đã chỉ chỗ: nhãn nút hoặc tooltip trên chính nút. `fix/ui-empty` của BA đã rà SRS |

## 10. Việc cần PM / chủ dự án quyết

`proposals.md` P-107-1 … P-107-11. Chặn lô 0: P-107-1, -2, -3, -4, -5. Chặn lô 6: P-107-6. Chặn lô 8: P-107-8, -10. Chặn lô 9: P-107-9, -11. Không chặn dev: P-107-7 (việc của QC).

## 11. Nợ từ handoff dev đưa vào hợp đồng §23 (PM 14/11)

Nguồn: `handoff/dev-sprint10.7.md` trên nhánh này, và tin PM.

| Nợ | Thuộc §23? | Xử lý | Lô | Đo bằng |
| --- | --- | --- | --- | --- |
| **Toolbar 390 px: ba nút xuống 2 dòng** (`/grading` tab Hàng chờ: «Tải bài lên» + «Chấm tất cả» + «Công bố»; `Layout.module.css` `.toolbarEnd` không `flex-wrap`) | Có, §23.2 | Không vá `.toolbarEnd`. Ba nút này là hành động cấp trang, không phải của thanh lọc → chuyển lên `PageHeader`: `primary` = «Chấm tất cả», `secondary` = [«Tải bài lên», «Công bố»]. Dưới 720 px primitive tự gom hai nút phụ vào nút «Thêm», hàng nút còn đúng một dòng. `Toolbar.end` từ nay chỉ chứa điều khiển của bảng (công tắc, sắp xếp). Ảnh chuẩn `grading-queue-390.png` sinh lại trên Linux / CI | 3 | H5, H6, H8 |
| **(d) Bộ lọc «Bài tập» mất khi rời route** | Không (hành vi, không phải khung) | Ghi vào §23.3 một câu để mọi màn làm giống nhau: trạng thái lọc cấp trang (`Tabs`, `Toolbar`) sống trong **URL** (`?assignment=…&status=…`), không ở `sessionStorage`. Quay lại, tải lại, gửi liên kết đều giữ lọc; danh sách tải sau không ghi đè được vì URL là nguồn. Bỏ khoá `sessionStorage` của 10.6 khi chuyển | 3 | e2e: chọn bài tập → sang `/gradebook` → quay lại → còn lọc; tải lại trang → còn lọc |
| **(k) Dòng báo xong sau «Chấm lại»** (`Đã chấm xong N bài`) | Một phần: vị trí | Dòng báo việc nền là dòng xác nhận tĩnh lặng (`UndoLine` không nút), nằm ngay dưới `Toolbar` của vùng, trong panel. Không toast, không đẩy bố cục (chỗ đã chừa sẵn một dòng). Nhánh «lô không tới SUCCEEDED» và «tải lại giữa chừng»: dòng đọc trạng thái từ `useJob`, nên tải lại vẫn hiện đúng; lô lỗi hiện `InlineNotice` một dòng + «Thử lại» theo §23.5 | 3 | e2e hiện có của 10.6 + L3 (không nhảy bố cục) |
| **Landing: axe báo tương phản chập chờn** | Có, mục 8 | `[SUY LUẬN]` chưa tái hiện: chữ đang ở giữa chuyển động theo cuộn có `opacity` < 1 lúc axe lấy mẫu, nên tỉ lệ tương phản tính trên màu đã pha. Mẫu mới chỉ dùng `transform` cho chữ; `opacity` chỉ dùng cho ảnh. Thêm vào `landing.spec.ts`: chạy axe với `reducedMotion: "reduce"` **và** một lượt sau khi cuộn hết trang, cả hai phải 0 lỗi. Nếu còn chập chờn thì nguyên nhân khác, dev báo lại | 9 | axe 0 serious / critical ở 20 lượt liên tiếp |

Nợ «`/insights` và các màn mô phỏng còn `?? X[COURSE_1]`» (trái D51) không thuộc khung; giữ ở sprint 11 như handoff ghi. `frame.spec.ts` chạy được trên màn mô phỏng nên không chặn UI-R.

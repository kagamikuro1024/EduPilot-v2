# FEAT-uiux-91 Cải thiện UI/UX sau khảo sát (sprint 9.1)
Nguồn: `docs/sprints/9.1/audit/findings.md` (UX-1…27) · `docs/sprints/9.1/plan.md` (mục "Quyết định của chủ dự án": Q-UX91-1…5, [ĐỔI DESIGN] trang giới thiệu) · `docs/sprints/9.1/techlead.md` TL-1 · DESIGN §0–§22 · UX.md · luật chữ UI (`~/.claude/CLAUDE.md`, 09/10/2026).
Phase: sprint 9.1 pha 2 (không thuộc P0–P9). Trạng thái: **APPROVED** toàn bộ 21 story. Căn cứ: chủ dự án duyệt báo cáo UI/UX và trang giới thiệu v2 (2026-11-08, `plan.md` mục "Chủ dự án duyệt báo cáo"). **"Dùng thử" (US-UX91-22) đã gỡ khỏi spec theo P-91-4** (chủ dự án bỏ, huỷ P-91-3). Đã theo: `techlead.md` TL-1, TL-2; `plan.md` mục "Quyết định của PM cho các câu TL-2"; `proposals.md` P-91-1, P-91-2, P-91-4. Thứ tự làm của dev: đợt 1 → 2 → 6 → 3 → 4 → 5; mỗi story ghi đợt ở dòng "Ưu tiên".

**Quy ước chung cho mọi story** (không lặp lại trong từng AC):
- Ảnh mốc: `visual.spec.ts` có 7 route × 2 bề rộng (`home`, `chat`, `threads`, `inbox`, `gradebook`, `settings-llm`, `dev-ui` × 1440 / 390). Story nào đổi hình một route đó thì ghi "Ảnh mốc đổi: …"; dev sinh lại bằng `--update-snapshots` kèm giải thích trong handoff. Route chưa có ảnh mốc mà story đổi nhiều (trang giới thiệu, điểm danh, Hôm nay giảng viên, duyệt bài) thì QC thêm ảnh mốc mới ở story đó.
- 375 px: không tràn ngang (`scrollWidth ≤ clientWidth`), vùng chạm ≥ 44 px.
- `prefers-reduced-motion: reduce`: chuyển động chức năng thành đổi `opacity` / màu tức thì hoặc ≤ 100 ms; chuyển động trang trí biến mất, nội dung hiện **trạng thái cuối**.
- `bash scripts/ui-antipatterns.sh` sạch; axe 0 lỗi mức serious / critical trên route đổi.
- Chữ UI tiếng Việt, không từ kỹ thuật AI cho sinh viên (DESIGN §13).
- Các story sửa hành vi / chữ đã nằm trong SPEC cũ ghi ở SRS mục 10 (bảng "Spec cũ cần vá"). Vá spec cũ chỉ sau khi PM ghi số proposal.

Story gộp theo **6 đợt** của TL-2. Thứ tự dev làm (plan 9.1): **đợt 1 → 2 → 6 → 3 → 4 → 5**. Đợt 1 + 2 sinh lại ảnh mốc **một lần chung**. Mỗi story có dòng "Đợt n" ở đầu.

| Thứ tự | Đợt | Story | Findings | Cỡ (TL-2) | Ưu tiên |
| :---: | --- | --- | --- | :---: | --- |
| 1 | 1 Nền chung | US-UX91-02 Định dạng ngày giờ dùng chung | UX-8 | M | Must |
| 1 | 1 | US-UX91-03 Nút khoá nói lý do | UX-13 | M | Must |
| 1 | 1 | US-UX91-04 Chuyển động chức năng của ứng dụng | UX-26, TL-1 đề xuất 3 | M | Must |
| 1 | 1 | US-UX91-21 Trang công khai không gọi làm mới phiên | UX-3(d) | S | Should |
| 2 | 2 Chữ | US-UX91-01 Xoá câu giải thích dưới tiêu đề | UX-4 | M | Must |
| 2 | 2 | US-UX91-05 Trạng thái rỗng đúng mẫu | UX-12, 22 | S | Should |
| 2 | 2 | US-UX91-07 Nhãn điều hướng bằng tiếng Việt | UX-9, Q-UX91-1 | S | Must |
| 2 | 2 | US-UX91-06 Nhớ lớp đã chọn, lớp đầu có dữ liệu | UX-25(a) cách S | S | Should |
| 3 | 6 Trang giới thiệu v2 + clip + video | US-UX91-08 Trang giới thiệu v2 | UX-1, 2, 3(a–c), Q-UX91-5, duyệt v2 | L | Must |
| 3 | 6 | US-UX91-09 Clip màn đầu + video đầy đủ | UX-1, duyệt v2 | L | Must |
| 4 | 3 Hôm nay + Hộp thư | US-UX91-10 Hôm nay giảng viên: nhóm việc | UX-5, 6, Q-UX91-2 | M | Must |
| 4 | 3 | US-UX91-11 Hôm nay sinh viên: chỉnh nhỏ | §3.B | S | Should |
| 5 | 4 Chấm bài / Sổ điểm / Điểm danh | US-UX91-14 Hàng chờ chấm bài | UX-14 | S | Should |
| 5 | 4 | US-UX91-15 Duyệt bài: một hàng một tiêu chí, thanh hành động dính | UX-15 | M | Must |
| 5 | 4 | US-UX91-16 Sổ điểm: số căn thập phân, Đủ điểm / Thiếu điểm | UX-16, Q-UX91-3, P-91-1 | M | Must |
| 5 | 4 | US-UX91-12 Điểm danh: "Có mặt" không đỏ, lưu dính đáy | UX-7, Q-UX91-4 | M | Must |
| 6 | 5 Chat / Hỏi đáp / Me / Tài liệu / Insights | US-UX91-13 Chat riêng: một Panel, lịch sử phẳng | UX-11 | M | Must |
| 6 | 5 | US-UX91-19 Hỏi đáp công khai: đoạn trích sạch, bớt nhãn lặp | UX-20 | S | Should |
| 6 | 5 | US-UX91-20 Kết quả của tôi: một khối giải trình | UX-21 | S | Should |
| 6 | 5 | US-UX91-18 Tài liệu giảng viên: bớt chữ lặp | UX-19 | S | Should |
| 6 | 5 | US-UX91-17 Insights và Quan sát AI: bỏ khung lồng khung | UX-10 | M | Should |

Đã bỏ khỏi phạm vi theo P-91-2: dự phòng `IntersectionObserver` cho `view()`; tab GV / SV (thay bằng hai cột, mỗi vai 4 mục); [ĐỔI DESIGN] nút `link` (dùng `ghost` + mũi tên); phép tự động bắt "Panel chứa khung có viền"; gợi ý "chủ đề sai nhiều" ở chat (ghi Nợ, cần API). Bộ chọn lớp hiện buổi học tiếp theo và lớp "có nhiều việc nhất" (UX-25 L) ghi Nợ.

---

## US-UX91-01: Mọi người dùng muốn màn hình không có câu giải thích thừa dưới tiêu đề, để nhìn ra việc cần làm trong 3 giây
Ưu tiên: Must · Ước lượng: M · Sprint: 9.1 · **Đợt 2**

### Tiêu chí nghiệm thu
- AC1 (xoá đúng danh sách). Given bảng "Từng chỗ `description=`" bên dưới When dev làm xong Then mọi dòng có hành động **Xoá** không còn chuỗi đó trên màn (cả khi truyền qua `intro.ts`); `grep` mã nguồn không còn chuỗi cũ. Chuỗi mới (nếu có) chỉ là dữ liệu (số lượng, ngày, mã) hoặc tooltip / placeholder theo cột "Thay bằng".
- AC2 (không xoá nhầm). Given dòng có hành động **Giữ** When làm xong Then chữ đó vẫn hiện y nguyên (trạng thái rỗng, mô tả hộp thoại hệ quả, `description` của Drawer là dữ liệu).
- AC3 (LCP không lùi). Given `intro.ts` (khung máy chủ `PreShell`) đổi theo AC1 When chạy `lcp.spec.ts` ở các route `/`, `/chat`, `/threads`, `/me`, `/inbox`, `/gradebook`, `/settings/llm` Then vẫn PASS: ứng viên LCP đầu là khối chữ lớn nhất sau phiên (US-PU-06 AC8). Nếu bỏ câu phụ làm `h1` là khối lớn nhất thì ứng viên LCP là `h1` ở cả trước và sau phiên.
- AC4 (hộp thoại bảo vệ hệ quả). Given hộp thoại ở danh sách DESIGN §10.11 (vd. `PIIChannelDialog`) When mở Then mô tả hệ quả vẫn còn đủ.
- AC5 (tooltip đúng chỗ). Given dòng có hành động "tooltip" When người dùng rê / focus vào nhãn hoặc nút Then tooltip hiện đúng một câu ≤ 12 từ, gắn `aria-describedby`; không có icon `Info` / `CircleHelp` mới.
- AC6 (kiểm tự động). Given `scripts/ui-antipatterns.sh` When thêm phép "Câu mô tả tĩnh dưới tiêu đề": `description=` của `PageHeader` / `Section` là chuỗi tĩnh > 40 ký tự Then script in `✗` và thoát 1; `--selftest` có ca gieo cho phép này và vẫn bắt `N / N` phép.
- AC7 (nhánh lỗi: dữ liệu rỗng không làm vỡ). Given `meta` / `description` nhận `undefined` hoặc chuỗi rỗng When render Then không để lại khoảng trống dòng (chiều cao tiêu đề không đổi chỗ, CLS ≤ 0,05).
- AC8 (di động). Given 375 px When mở từng route đổi Then tiêu đề không còn dòng thứ hai trống; không tràn ngang.
- AC9 (phân quyền). Given vai không được phép (vd. sinh viên mở `/observability`) When mở Then vẫn hiện thông báo 403 hiện có ("Bạn không có quyền xem màn này" — `AppShell` `whoCanOpen`), không đổi.
- AC10 (ảnh mốc). Ảnh mốc đổi: `home-*`, `chat-*`, `threads-*`, `inbox-*`, `gradebook-*` (mất dòng phụ) — dev sinh lại.

### Từng chỗ `description=` (đếm ngày 2026-11-08: 57 dòng `description=` trong `frontend/src` ngoài `shared/ui`; 35 tệp). **Không sửa `Demo*.tsx`** (`DemoChat`, `DemoCalendar`, `DemoDocuments`, `DemoLibrary`, `DemoThreads`): chỉ phiên mô phỏng dùng (TL-2 UX-4); các dòng đó ghi "Không sửa". Số chỗ thật sửa ít hơn 55.
Hành động: **Xoá** = bỏ chuỗi; **Giữ (dữ liệu)** = số liệu / mã / ngày, không phải câu giải thích; **Giữ (trạng thái / hộp thoại)** = lý do rỗng / 403 / hệ quả không đảo ngược.

| # | Tệp:dòng | Chuỗi hiện tại (rút gọn) | Hành động | Thay bằng |
| --- | --- | --- | --- | --- |
| 1 | `admin/AdminCourses.tsx:84` | "Mở lớp, phân công giảng viên và lưu trữ lớp đã kết thúc." | Xoá | — |
| 2 | `admin/AdminUsers.tsx:124` | "Mời giảng viên và trợ giảng, khoá hoặc mở khoá… Sinh viên tự đăng ký…" | Xoá | — |
| 3 | `admin/courses/CoursePanel.tsx:110` | "Người được gán nhận thông báo kèm mã tham gia ngay khi lớp mở…" | Xoá | Tooltip nút `Mở lớp`: "Người được gán nhận thông báo kèm mã tham gia." |
| 4 | `admin/users/InvitePanel.tsx:34` | "Người được mời nhận một liên kết đặt mật khẩu, dùng trong 72 giờ. Quản trị viên không biết…" | Xoá | Tooltip nút gửi lời mời: "Liên kết đặt mật khẩu dùng trong 72 giờ." |
| 5 | `analytics/AnalyticsScreen.tsx:31` | "Xu hướng của cả lớp để điều chỉnh cách dạy — không phải bảng theo dõi từng sinh viên." | Xoá | — |
| 6 | `analytics/AnalyticsScreen.tsx:71` | "Câu hỏi sinh viên gửi trong {days}" | Xoá | — (bộ chọn 7 / 30 ngày đã nói) |
| 7 | `analytics/AnalyticsScreen.tsx:80` | "Câu AI không tự trả lời được và thời gian lớp chờ bạn" | Xoá | — |
| 8 | `analytics/AnalyticsScreen.tsx:93` | "Thông tin cá nhân bị che trước khi câu hỏi đi tới model" | Xoá | — |
| 9 | `analytics/AnalyticsScreen.tsx:100` | "AI chỉ ra bản nháp; con số dưới đây cho biết…" | Xoá | — |
| 10 | `analytics/AnalyticsScreen.tsx:120` | "Tiền trả cho model trong khoảng đang xem" | Xoá | — |
| 11 | `attendance/RealAttendance.tsx:117` | `{lớp} · buổi N · ngày giờ · phòng` | Giữ (dữ liệu) | Bỏ mã lớp (đã có thanh trên): "buổi 4 · Thứ Hai, 02/11 · 13:30–16:00 · P.405" |
| 12 | `calendar/DemoCalendar.tsx:56` | `{lớp} · {lịch học} · {phòng}` | Giữ (dữ liệu) | — |
| 13 | `chat/DemoChat.tsx:113` | "Hỏi về điểm, chuyên cần và bài học của chính bạn — {lớp}. Giảng viên không đọc được phiên chat này." | Không sửa (`Demo*`) | — (chỉ khung máy chủ `intro.ts` `/chat` đổi, dòng 59) |
| 14 | `chat/SessionSheet.tsx:38` | "N phiên của bạn" | Giữ (dữ liệu) | — |
| 15 | `documents/DemoDocuments.tsx:171` | "Nguồn để AI trả lời sinh viên và nội dung hiện trong Thư viện của lớp." | Không sửa (`Demo*`) | — |
| 16 | `documents/DemoDocuments.tsx:279` | nhãn loại tài liệu (Drawer) | Giữ (dữ liệu) | — |
| 17 | `exam/ExamEditor.tsx:147` | trạng thái · loại · giờ · "N sinh viên đã bắt đầu" | Giữ (dữ liệu) | — |
| 18 | `exam/ExamItems.tsx:147` | "Chỉ hiện câu đã duyệt và chưa lưu trữ." | Xoá | Tiêu đề Drawer: "Chọn câu đã duyệt từ ngân hàng" |
| 19 | `exam/ExamsHome.tsx:52` | "Bài kiểm tra tính điểm của lớp {code}: soạn từ ngân hàng…" | Xoá | — |
| 20 | `exam/ExamsHome.tsx:144` | "Bài kiểm tra tính điểm của lớp. Đồng hồ chạy khi bạn bấm Bắt đầu làm bài." | Xoá | Tooltip nút `Bắt đầu làm bài`: "Đồng hồ chạy ngay khi bấm." (hệ quả của lựa chọn) |
| 21 | `exam/results/ResultDrawer.tsx:21` | trạng thái bài làm | Giữ (dữ liệu) | — |
| 22 | `exam/take/TakeRunning.tsx:245` | "Đã làm x/y câu." | Giữ (dữ liệu) | — |
| 23 | `insights/InsightsScreen.tsx:65` | "Câu hỏi của lớp, đã bỏ tên và mã số sinh viên, nhóm theo chủ đề…" | Xoá | — |
| 24 | `insights/InsightsScreen.tsx:110` | "Việc chạy nền, bạn đi làm việc khác được; báo cáo sẽ hiện ở đây khi xong." | Xoá | Chỉ báo giai đoạn theo UX.md §2.4 ("Đang nhóm 66 câu hỏi…") |
| 25 | `insights/InsightsScreen.tsx:124` (`InlineNotice` riêng tư) | "Báo cáo chỉ dùng câu hỏi của {lớp}. Tên và mã số… chủ đề dưới 3 sinh viên không hiện câu mẫu." | Xoá | `meta` ≤ 12 từ: "Đã bỏ tên và MSSV · chủ đề dưới 3 sinh viên không hiện câu mẫu" |
| 26 | `insights/InsightsScreen.tsx:128` | `{window} · xếp theo mức độ cả lớp đang vướng` | Xoá | `meta` chỉ còn `{window}` |
| 27 | `insights/InsightsScreen.tsx:197` | "Sinh viên hỏi nhưng không tài liệu nào của lớp trả lời được — AI phải chuyển…" | Xoá | — |
| 28 | `insights/InsightsScreen.tsx:212` | "Bản trước tạo ngày 22/10; chỉ liệt kê thay đổi đáng kể." | Xoá | Tiêu đề: "So với báo cáo ngày {dd/MM}" |
| 29 | `join/JoinScreen.tsx:111` | "Nhập mã tham gia giảng viên gửi cho bạn. Mã gồm 7 ký tự, không phân biệt hoa thường." | Xoá | Placeholder ô nhập: `AN7K2MQ` (ví dụ cú pháp) |
| 30 | `library/DemoLibrary.tsx:41` | "Bài giảng, quy chế và đề cũ của lớp. Tìm theo tên, chủ đề hoặc tuần học." | Không sửa (`Demo*`) | — |
| 31 | `library/DemoLibrary.tsx:113` | tóm tắt tài liệu (`doc.summary`) | Giữ (dữ liệu) | — |
| 32 | `me/RealMe.tsx:40` | mã lớp trần "761987" | Xoá | — (US-UX91-20) |
| 33 | `members/ClassSettings.tsx:66` | "Lớp {code}" | Xoá | — (mã lớp đã ở thanh trên) |
| 34 | `members/ClassSettings.tsx:170` (`Switch`) | "Sinh viên vào bằng mã sẽ chờ bạn duyệt." | Xoá | Tooltip nhãn "Cần giảng viên duyệt": "Sinh viên vào bằng mã sẽ chờ bạn duyệt." |
| 35 | `members/MembersView.tsx:184` | "Lớp {code}. Duyệt yêu cầu vào lớp và quản lý người học." | Xoá | — |
| 36 | `observability/ObservabilityScreen.tsx:94` | "Số liệu AI của lớp bạn. Nội dung câu hỏi chỉ quản trị viên hệ thống mở được… ghi nhật ký." / "Hệ thống đang chạy thế nào…" | Xoá | — |
| 37 | `observability/ObservabilityScreen.tsx:129` | "Số tổng hợp của lớp bạn phụ trách." | Xoá | — |
| 38 | `observability/ObservabilityScreen.tsx:136` | "Chọn một hàng để xem nội dung đã che…" / "Giảng viên xem được số liệu; nội dung… không mở được từ đây." | Xoá | — |
| 39 | `observability/ObservabilityScreen.tsx:167` | `{giờ} · {model} · lớp {id}` (Drawer) | Giữ (dữ liệu) | — |
| 40 | `practice/AttemptScreen.tsx:115` | `{chủ đề} · N câu · {khi nào}` | Giữ (dữ liệu) | — |
| 41 | `practice/AttemptScreen.tsx:426` | "Xem lại lượt đã làm · đúng x/y câu" | Giữ (dữ liệu) | — |
| 42 | `practice/PracticeHistory.tsx:23` | "N lượt gần đây. Bấm một lượt để xem lại từng câu và giải thích." / "Chưa có lượt luyện nào." | Xoá | `meta` "N lượt gần đây"; rỗng dùng `EmptyState` |
| 43 | `practice/PracticeHistory.tsx:59` | "Tính trên các câu đã làm trong các lượt gần nhất." | Xoá | — |
| 44 | `practice/PracticeScreen.tsx:74` | "Luyện theo chủ đề bạn hay sai, hoặc thi thử với đề cũ." | Xoá | — |
| 45 | `questions/QuestionBank.tsx:95` | "Câu hỏi dùng cho bài thi và luyện đề của lớp {code}. Chỉ giảng viên và trợ giảng thấy." | Xoá | — (quyền đã chặn ở route) |
| 46 | `questions/QuestionReview.tsx:79` | `{loại} · {chủ đề} · {độ khó}` (Drawer) | Giữ (dữ liệu) | — |
| 47 | `students/RealProfile.tsx:51` | `{MSSV} · {lớp}` | Giữ (dữ liệu) | Bỏ mã lớp |
| 48 | `students/RealStudents.tsx:85` | `{N} sinh viên · {lớp}` | Giữ (dữ liệu) | Bỏ mã lớp: "24 sinh viên" |
| 49 | `threads/DemoThreads.tsx:159` | "Câu hỏi công khai của {lớp}. Câu trả lời có nhãn xác nhận là đã được giảng viên duyệt." | Không sửa (`Demo*`) | — (khung máy chủ `intro.ts` `/threads` đổi, dòng 59) |
| 50 | `threads/DemoThreads.tsx:171` | "Hỏi về nội dung môn học. Đừng ghi mã số sinh viên… chuyện riêng hỏi ở Chat riêng." | Không sửa (`Demo*`) | — (Threads thật `RealThreads.tsx` không có câu này; `PIIProtectionNotice` khi gõ giữ nguyên) |
| 51 | `threads/PIIChannelDialog.tsx:34` | "Threads là nơi cả lớp cùng đọc. Chúng tôi tìm thấy: {…}." | Giữ (hộp thoại) | Đổi "Threads" → "Hỏi đáp công khai" (US-UX91-07) |
| 52 | `threads/PIIChannelDialog.tsx:63` | `{message}` | Giữ (hộp thoại) | — |
| 53 | `today/AdminToday.tsx:15` | `{ngày dài} · việc của hệ thống` | Giữ (dữ liệu) | — |
| 54 | `today/StaffToday.tsx:19` | `{ngày dài} · {Tất cả lớp / Lớp N}` | Giữ (dữ liệu) | Chỉ hiện phần phạm vi khi "Tất cả lớp" (US-UX91-10) |
| 55 | `today/StudentToday.tsx:29` | `{ngày dài}` + "bạn chưa vào lớp nào" | Giữ (dữ liệu) | — |
| 56 | `today/StudentToday.tsx:59` | "Bạn chưa vào lớp nào. Nhập mã do giảng viên cung cấp…" | Rút còn "Bạn chưa vào lớp nào." (ui-empty) | Trạng thái rỗng "chưa vào lớp" + một hành động `Nhập mã` (cú pháp mã ở `placeholder`) |
| 57 | `shared/shell/AppShell.tsx:360` | `whoCanOpen(pathname)` | Giữ (trạng thái / 403) | — |
| 58 | `shared/shell/AppShell.tsx:475` | "Nhập mã tham gia do giảng viên cung cấp để dùng tính năng này." | Giữ (trạng thái) | — |
| 59 | `shared/shell/intro.ts` | `INBOX_INTRO.description`, `/threads` (+ nhãn "Threads" → "Hỏi đáp công khai"), `/me`, `/gradebook`, `/chat` (`description` và `body` nói về riêng tư của phiên), `/` | Xoá `description` (giữ `title`; `body` của `/chat` giữ vì là khối chữ lớn nhất của màn đầu và là cam kết riêng tư, chỉ đổi nếu `lcp.spec` đòi) | Khung máy chủ phải khớp màn thật (AC3) |

### Ngoài phạm vi của story này
- Chữ trên màn quản trị / cài đặt chưa khảo sát (BA chưa đi); QC / BA rà ở sprint sau.
- Đổi `Section` / `PageHeader` API (vẫn nhận `description`); chỉ ngừng truyền câu giải thích.

### Phụ thuộc
- US-UX91-07 (đổi tên Threads trong dòng 51, 59), US-UX91-10 (dòng 54, 59). Ghi nhận quyết định #42 sprint 7 (đã bỏ một phần câu mô tả, `handoff/dev-PM-42.md`).

---

## US-UX91-02: Mọi người dùng muốn ngày giờ hiển thị một kiểu, đúng múi giờ Việt Nam, để không phải đoán
Ưu tiên: Must · Ước lượng: M · Sprint: 9.1 · **Đợt 1**

### Tiêu chí nghiệm thu
- AC1 (một hàm chung). Given một mô-đun định dạng ngày giờ ở `shared/lib` When dev làm xong Then `grep` các tệp ngoài mô-đun đó không còn `Intl.DateTimeFormat`, `toLocaleDateString`, `toLocaleTimeString` cho **hiển thị** ngày giờ (danh sách hiện có: `admin/users/api.ts`, `chat/RealChat.tsx`, `documents/RealDocuments.tsx`, `inbox/RealInbox.tsx`, `library/RealLibrary.tsx`, `me/meApi.ts`, `members/ClassSettings.tsx`, `settings/security/DevicesSection.tsx`, `students/RealProfile.tsx`, `threads/RealThreadDetail.tsx`, `threads/RealThreads.tsx` (cả dòng 158), `today/format.ts`, `shared/lib/timeAgo.ts`).
- AC2 (quy ước hiển thị). Given đồng hồ cố định `2026-11-08 14:00 ICT` When định dạng Then:

  | Thời điểm | Chuỗi |
  | --- | --- |
  | cùng ngày lịch (ICT) | `Hôm nay, 09:20` |
  | hôm qua | `Hôm qua, 16:40` |
  | ngày mai (ICT) | `Ngày mai, 09:00` |
  | từ hôm kia tới 6 ngày trước, và từ 2 tới 6 ngày tới (cùng tuần) | `Thứ Hai, 14:00` |
  | từ đúng 7 ngày trước trở về trước, và từ đúng 7 ngày tới trở đi, cùng năm | `Thứ Hai, 21/09 · 14:00` |
  | khác năm | `Thứ Hai, 22/09/2025 · 14:00` (22/09/2025 là thứ Hai; P-91-7 (1)) |
  | chỉ ngày | `21/09` (cùng năm) / `21/09/2025` |
  | tương đối ngắn (thông báo, hàng danh sách) | giữ `vừa xong`, `N phút trước`, `N giờ trước` của `timeAgo`, sau đó dùng dòng "cùng ngày / hôm qua / thứ" ở trên |

  Chữ "Thứ Hai … Chủ Nhật", không "Thứ 2".
- AC3 (múi giờ). Given trình duyệt đặt múi giờ `America/Los_Angeles` When mở Threads, Tài liệu, Chat, Thư viện Then giờ hiển thị vẫn theo `Asia/Ho_Chi_Minh` (hiện 5 tệp `Real*` chưa đặt `timeZone`).
- AC4 (nhánh lỗi). Given chuỗi ISO sai hoặc `null` When định dạng Then trả "—" (không `Invalid Date`, không ném lỗi).
- AC5 (kiểm tự động). Given test đơn vị của mô-đun When chạy Then phủ: các hàng bảng AC2, ranh giới nửa đêm ICT (23:59 / 00:01), đổi tuần, khác năm, `null`, múi giờ trình duyệt khác.
- AC6 (ảnh mốc). Ảnh mốc đổi: `threads-*`, `chat-*`, `inbox-*` (ngày giờ trong danh sách).
- AC7 (di động). Given 375 px When mở Threads / Chat Then chuỗi ngày giờ không làm hàng tràn ngang (có thể xuống dòng).

### Ngoài phạm vi
- Đổi định dạng ngày trong tệp tải xuống / email (back-end); nhãn trục biểu đồ (UX-17, để sau).

### Phụ thuộc
- UX.md §2.10; không có.

---

## US-UX91-03: Giảng viên muốn biết vì sao nút bị khoá, để biết phải làm gì tiếp
Ưu tiên: Must · Ước lượng: S · Sprint: 9.1 · **Đợt 1**

### Tiêu chí nghiệm thu
- AC1 (primitive). Given `Button` / `ButtonLink` nhận `disabledReason: string` When `disabled` và có lý do Then nhãn nút giữ nguyên; rê chuột hoặc focus bàn phím hiện tooltip chứa lý do; nút có `aria-disabled="true"` và `aria-describedby` trỏ tới `span` ẩn trực quan chứa lý do; **không** dùng thuộc tính `disabled` gốc khi có lý do (để còn focus được); `onClick` bị chặn; kiểu "mờ" `opacity: .45` của `.btn:disabled` áp cả `[aria-disabled="true"]` (TL-2 UX-13). Test cũ dùng `toBeDisabled()` vẫn đúng (Playwright tính cả `aria-disabled`); dev chạy lại toàn bộ e2e.
- AC2 (đủ chỗ). Given các nút khoá hiện có When làm xong Then mỗi nút có lý do bằng chữ cụ thể: `Chốt điểm` (sổ điểm, công thức chưa xác nhận: "Chưa có công thức điểm đã xác nhận", theo chữ của mã, P-91-7 (4)), `Xác nhận công thức` (khi còn mục "chưa rõ": "Còn N mục chưa rõ"), `Chấm tất cả` (hàng chờ: "Chưa có bài nộp"), `Tải bài lên` (hàng chờ chấm: "Chọn một bài tập trước", khoá khi bộ lọc "Bài tập" đang là "Tất cả bài tập"). `Phiên mới` **không thuộc story này**: nút ẩn khi phiên trống, xem US-UX91-13 AC4 (P-91-5 Q2).
- AC3 (bàn phím). Given nút khoá có lý do When Tab tới Then focus ring hiện; tooltip hiện; `Enter` / `Space` không kích hoạt hành động.
- AC4 (cảm ứng, 375 px). Given nút khoá When chạm Then lý do hiện dưới nút ≥ 3 giây hoặc tới khi chạm chỗ khác (không phụ thuộc hover).
- AC5 (nhánh lỗi). Given `disabledReason` có nhưng nút không `disabled` When render Then lý do bị bỏ qua (không hiện tooltip) và nút hoạt động bình thường.
- AC6 (phân quyền). Given STUDENT / vai không có quyền When mở màn Then nút giảng viên **không có** trong DOM (không chỉ khoá); lý do khoá không lộ dữ liệu của người khác.
- AC7 (chuyển động). Tooltip hiện bằng `opacity` ≤ 120 ms; `reduce` → hiện tức thì.
- AC8 (trưng bày và kiểm). `/dev/ui` có ca nút khoá có lý do (kể cả ca có `disabledReason` nhưng không `disabled`), vị trí nêu ở handoff; QC kiểm bằng E2E (rê / focus / chạm), không bằng ảnh mốc vì `visual.spec` chụp khung nhìn tĩnh còn lý do chỉ hiện khi rê / focus (P-91-9 (1)).

### Ngoài phạm vi
- Nút khoá ở màn quản trị chưa khảo sát.

### Phụ thuộc
- DESIGN §14.10 (lý do khi focus / hover); UX.md §2.5; luật chữ UI (tooltip cho lý do nút bị khoá).

---

## US-UX91-04: Mọi người dùng muốn thao tác trong ứng dụng có phản hồi ngắn gọn, không làm chậm việc lặp lại
Ưu tiên: Must · Ước lượng: M · Sprint: 9.1 · **Đợt 1**

Căn cứ: plan 9.1 "Hướng chuyển động" (ứng dụng: chỉ chuyển động chức năng ≤ 200 ms, không thư viện), `emil-design-eng`, `review-animations`, TL-1 mục C.

### Tiêu chí nghiệm thu
- AC1 (phản hồi bấm). Given `Button` các biến thể (`primary`, `secondary`, `ghost`, `text`, icon) When nhấn giữ (`:active`) Then nút thu `transform: scale(0.97)` trong ≤ 100 ms; thả ra trở lại; dùng `--ep-ease-out`.
- AC2 (hover chỉ cho chuột). Given thiết bị cảm ứng (`@media (hover: none)`) When chạm Then không có trạng thái hover "dính" (mọi `:hover` trong `shared/ui` và `features/*` bọc `@media (hover: hover) and (pointer: fine)`); kiểm bằng `grep` — mọi quy tắc `:hover` nằm trong khối đó.
- AC3 (token ≤ 200 ms). Given `--ep-slow` (hiện 250 ms, dùng ở `Dialog.module.css` dòng 10, 25) When sửa Then không còn token chuyển động > 200 ms trong `tokens.css`; Dialog / Drawer mở ≤ 200 ms; `DESIGN.md` §11 ghi "Drawer / dialog: ≤ 200 ms" (TL-1 đề xuất 3).
- AC4 (thanh tiến độ không đổi layout). Given 4 thanh tiến độ hiện dùng `transition: width` (`GradeScheme`, `Documents`, `Questions`, `Practice`) When tiến độ đổi Then chuyển bằng `transform: scaleX()` với `transform-origin: left`, 140 ms, không còn `transition: width`.
- AC5 (khung vỏ). Given thu gọn / mở sidebar When bấm `Thu gọn` Then nội dung chính đổi bề rộng **tức thì** hoặc ≤ 160 ms bằng `transform`, không còn `transition: margin-left` trên `main`; không giật nội dung (CLS ≤ 0,05).
- AC6 (gốc xuất phát). Given Menu / popover When mở Then `transform-origin` ở nút gọi, `translateY(-4px)` + `opacity`, ≤ 160 ms; Dialog ở giữa màn không đổi gốc.
- AC7 (xác nhận lưu tại chỗ). Given sổ điểm / điểm danh / nhận xét chấm When một ô vừa lưu thành công Then nền ô đổi sang tông `--ep-green` rất nhạt (lớp / keyframe `ep-saved`) rồi mờ về trạng thái thường trong 200 ms (không toast "Thành công", không âm thanh). **Kiểm ở đợt 4** khi dev áp vào từng ô (US-12, US-15, US-16); đợt 1 chỉ cần `ep-saved` có sẵn và trưng bày ở `/dev/ui` (P-91-8 (a)).
- AC8 (không thêm trang trí). Given toàn ứng dụng sau đăng nhập When `grep` Then không có thư viện chuyển động mới trong `package.json`, không có hiệu ứng so le (stagger) khi tải trang, không có bounce / parallax; mọi keyframe chỉ đổi `transform` / `opacity`, **trừ**: (i) chỉ báo tải chạy liên tục (không phải chuyển trạng thái): spinner của `Button` (700 ms), ba chấm của Hỏi đáp công khai và `shimmer` của khung xương ở `Feedback` (1400 ms, đổi `background-position`), giữ nguyên, không bắt buộc đổi sang `transform` (P-91-5 Q6); (ii) `ep-saved` (AC7, đổi **màu nền** trong 200 ms, P-91-7 (3b)).
- AC9 (reduced-motion, giảm thay vì tắt). Given `prefers-reduced-motion: reduce` When thao tác Then luật lưới `base.css` (đặt `1ms`) **giữ nguyên**; chuyển động chức năng mới (nháy lưu, mờ dần của highlight, tooltip) được **cho phép riêng** `opacity` / màu bằng lớp `.ep-motion-safe` (không rà 71 chỗ hiện có), còn `transform` / trượt bị bỏ. Kiểm bằng Playwright `reducedMotion: "reduce"`: ô vừa lưu vẫn đổi màu nền và tắt sau ≤ 200 ms (TL-2 UX-26).
- AC10 (bàn phím). Given thao tác bàn phím (J / K, `A`, `1`–`4`, `⌘K`) When dùng Then không có hoạt cảnh kèm theo (đổi trạng thái tức thì).
- AC11 (nhánh lỗi). Given thanh tiến độ nhận giá trị ngoài 0–100 When render Then kẹp về 0–100, không tràn.
- AC12 (ảnh mốc). Ảnh mốc không đổi (`visual.spec` chạy `reduce` + `animations: "disabled"`); riêng `dev-ui-*` có thể đổi nếu trạng thái `:active` được trưng bày — dev ghi trong handoff.

### Ngoài phạm vi
- Red Thread Transition (giữ nguyên, 280 ms, DESIGN §11); spinner `Button` và ba chấm Threads (chỉ báo chạy liên tục).
- Mọi chuyển động của trang giới thiệu (US-UX91-08).

### Phụ thuộc
- TL-1 mục C (đề xuất 3); DESIGN §11, §16.

---

## US-UX91-05: Người dùng muốn trạng thái rỗng gọn, có tối đa một việc kế tiếp
Ưu tiên: Should · Ước lượng: S · Sprint: 9.1 · **Đợt 2**

**ui-empty (chủ dự án 2026-11-14; luật chữ UI 09/10/2026):** trạng thái rỗng = **tiêu đề ngắn + tối đa một hành động**, không câu giải thích; thay luật cũ «nói vì sao rỗng». AC1, AC3, AC6 dưới đây đã sửa theo đó.

### Tiêu chí nghiệm thu
- AC1 (hộp thư). Given giảng viên mở `/inbox` với tab "Đang chờ" không có ticket When tải xong Then hiện `EmptyState`: **chỉ** tiêu đề "Chưa có câu hỏi nào cần thầy/cô xử lý.", không câu giải thích, căn giữa vùng danh sách; **không** hiện vùng chi tiết "Chọn một câu hỏi để xem" khi danh sách rỗng; **không** có nút "Xem tất cả" (trùng tab «Tất cả»).
- AC2 (chi tiết hộp thư). Given ticket đang mở When xem chi tiết Then không còn hai hàng "Đã chờ …" và "Trạng thái …" lặp thông tin; tuổi ticket hiện một lần cạnh tên sinh viên.
- AC3 (hàng chờ chấm). Given `/grading` chưa có bài nộp When tải xong Then `EmptyState`: "Chưa có bài nộp." + nút `Tải bài lên` (nếu có quyền); không câu giải thích.
- AC4 (sổ điểm trống). Given lớp chưa có công thức When mở `/gradebook` Then `EmptyState` có một nút chính `Thiết lập công thức`; câu nhỏ "Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường." không còn ở trạng thái rỗng (nếu cần, đặt ở chân bảng khi **có** dữ liệu, một dòng `meta`).
- AC5 (nhánh lỗi tải). Given API lỗi When tải Then thông báo lỗi là **một dòng** nói chuyện gì xảy ra + nút `Thử lại` (đã có `ApiErrorNotice`; sửa 10.7, P-107-3: bỏ vế «dữ liệu có an toàn không» trừ khi màn đang giữ chữ chưa lưu).
- AC6 (phân quyền). Given STUDENT mở `/inbox`, `/grading`, `/gradebook` When vào Then vẫn thấy thông báo 403 hiện có, không thấy `EmptyState` của giảng viên.
- AC7 (di động 375). Given `/inbox` rỗng ở 375 px When xem Then `EmptyState` không tràn; vùng chạm nút ≥ 44 px.
- AC8 (ảnh mốc). Ảnh mốc đổi: `inbox-*` nếu dữ liệu cố định của `visual.spec` có trạng thái rỗng; ngược lại không đổi — dev ghi.

### Ngoài phạm vi
- Trạng thái rỗng của màn chưa khảo sát.

### Phụ thuộc
- DESIGN §15; UX.md §2.5.

---

## US-UX91-06: Giảng viên muốn mở lại đúng lớp mình đã chọn, và lần đầu vào thấy lớp có dữ liệu
Ưu tiên: Should · Ước lượng: S · Sprint: 9.1 · **Đợt 2**

Quyết định PM (TL-2 câu 4): cách S. "Lớp có việc chờ" (chọn theo số việc, cần API) ghi Nợ.

### Tiêu chí nghiệm thu
- AC1 (nhớ lớp). Given giảng viên chọn lớp B rồi đăng xuất, đăng nhập lại cùng trình duyệt When vào `/` Then lớp B được chọn (`ep:ui:course`), không về lớp đầu danh sách.
- AC2 (lần đầu). Given trình duyệt mới (không có `ep:ui:course`) và giảng viên có ≥ 2 lớp ACTIVE When đăng nhập Then lớp mặc định là lớp **đầu tiên có dữ liệu** theo thứ tự `GET /me/courses`, tức lớp đầu tiên mà `GET /me/today` có ≥ 1 mục thuộc lớp đó (Q-UX91-9, PM đồng ý); không lớp nào có thì lớp đầu danh sách.
- AC3 (nhánh lỗi). Given `ep:ui:course` trỏ tới lớp không còn trong `me/courses` (bị gỡ), hoặc lớp có trạng thái `ARCHIVED` (máy chủ vẫn trả lớp đã lưu trữ trong `/me/courses`) When tải Then bỏ giá trị; lớp `ARCHIVED` **không** được làm lớp nhớ hay lớp mặc định (cả ở AC1 lẫn AC2): chọn theo AC2 trong các lớp `ACTIVE`, không báo lỗi, không trang trống (P-91-9 (2)).
- AC4 (phân quyền). Given STUDENT / TA When dùng bộ chọn Then chỉ thấy lớp của mình; không mục "Quản lý lớp này" / "Mời trợ giảng" nếu không có quyền (như hiện tại).
- AC5 (di động 375). Given 375 px When mở bộ chọn lớp Then mỗi mục vùng chạm ≥ 44 px (như hiện tại).
- AC6 (ảnh mốc). Không đổi ảnh mốc.

### Ngoài phạm vi
- Bộ chọn lớp hiện buổi học tiếp theo (cần dữ liệu theo lớp: Nợ); chọn lớp theo số việc chờ lớn nhất (L, API mới: Nợ); dữ liệu seed lớp 2 đồng nhất (việc của dev / QC, ghi `docs/PROGRESS.md`).

### Phụ thuộc
- `GET /me/courses` (đã có); không đổi API.

---

## US-UX91-07: Người dùng muốn nhãn điều hướng trùng tiêu đề trang và bằng tiếng Việt
Ưu tiên: Must · Ước lượng: S · Sprint: 9.1 · **Đợt 2**

Quyết định chủ dự án Q-UX91-1: `Threads` → **Hỏi đáp công khai**, `Insights` → **Lỗ hổng kiến thức**, `Analytics` → **Số liệu lớp học**.

### Tiêu chí nghiệm thu
- AC1 (điều hướng). Given sidebar desktop, thanh dưới / "Thêm" ở 375 px, bảng lệnh `⌘K` When mở Then các nhãn hiển thị đúng ba tên mới ở cả 4 vai có mục đó (sinh viên: Hỏi đáp công khai; giảng viên / TA / admin: cả ba nếu có quyền); không còn chữ `Threads`, `Insights`, `Analytics` trong DOM điều hướng.
- AC2 (tiêu đề trang = nhãn nav). Given `/threads`, `/insights`, `/analytics` When mở Then `h1` trùng nhãn nav; `<title>` = "{nhãn} · EduPilot"; `metadata.title` của ba `page.tsx` đổi; `intro.ts` (`/threads`) đổi.
- AC3 (chi tiết). Given `/threads/[id]` When mở Then liên kết quay lại ghi "Hỏi đáp công khai"; tiêu đề trang chi tiết giữ nghĩa hiện tại.
- AC4 (mọi chỗ nhắc). Given `grep` toàn `frontend/src` (ngoài mã định tuyến, tên biến, comment) When tìm "Threads" / "Insights" / "Analytics" hiển thị cho người dùng Then còn 0 chỗ, gồm: thông báo (`notifications.ts` `AI_PENDING`, `POST_REPORTED`), `threadsActions.ts` meta, `PIIChannelDialog`, câu `activity` ("Chat, Threads, Luyện đề…"), trang giới thiệu, `DemoThreads` thông báo "Câu hỏi mới trong Threads".
- AC5 (không phá đường dẫn). Given URL cũ `/threads`, `/insights`, `/analytics` When mở Then vẫn đúng route (không đổi đường dẫn).
- AC6 (độ dài). Given nhãn "Hỏi đáp công khai" ở sidebar thu gọn 72 px và thanh dưới 375 px When hiển thị Then không bị cắt / không đẩy tràn (thanh dưới dùng "Hỏi đáp" ≤ 8 ký tự nếu 5 mục không vừa — QC kiểm bằng ảnh 375 px).
- AC7 (e2e). Given `shell.spec.ts` bảng nhãn nav (dòng 120–124) When đổi Then sửa theo spec này (QC / dev sửa test theo AC1, không nới assertion).
- AC8 (phân quyền). Given TA / STUDENT When xem sidebar Then danh sách mục theo vai không đổi (chỉ đổi nhãn).
- AC9 (docs). Given `DESIGN.md` §2 / §14.3 / §14.21 / §14.25 When làm xong Then tên mới được cập nhật; ảnh mốc đổi: `threads-*`, `home-*` (sidebar), `chat-*` (sidebar), `inbox-*`, `gradebook-*`, `settings-llm-*` (sidebar có nhãn).

### Ngoài phạm vi
- Đổi đường dẫn URL; đổi tên trong Go / OpenAPI / tên bảng.

### Phụ thuộc
- Q-UX91-1 (đã chốt); ảnh mốc sidebar nên sinh lại một lần cùng US-UX91-10 để đỡ trùng.

---

## US-UX91-08: Khách ghé trang giới thiệu muốn thấy EduPilot làm việc ngay màn đầu và theo cuộn, để muốn dùng thử (v2)
Ưu tiên: Must · Ước lượng: L · Sprint: 9.1 · **Đợt 6**

Căn cứ: `plan.md` (chủ dự án duyệt trang giới thiệu v2, 2026-11-08), `DESIGN.md` §11.1 (đã sửa), TL-1 B, TL-2, `handoff/dev-phase1.md`. Chế độ *Persuade*; vẫn Be Vietnam Pro và token `--ep-*`. **Bản v1 đã dựng ở pha 1** (hero + video bấm phát, sơ đồ ba bước, số đếm, đổi vai); story này mô tả **v2** và những chỗ phải đổi so với v1. Trang chỉ có `Đăng nhập` / `Tạo tài khoản` + `Xem video`; không có nút "Dùng thử" (P-91-4).

### Tiêu chí nghiệm thu
- AC1 (cấu trúc). Given khách chưa đăng nhập mở `/` ở 1440 px When tải xong Then thấy theo thứ tự: màn đầu (`h1` "Vận hành lớp học có trợ lý AI, bảo vệ dữ liệu cá nhân" giữ nguyên chữ; dòng dẫn "AI soạn nháp. Giảng viên quyết. Điểm do mã tính."; nút; clip sống) → các `h2`: "Xem EduPilot làm việc" (kể chuyện theo cuộn, AC5), "AI soạn nháp, giảng viên quyết, mã tính điểm" (sơ đồ, AC8), "Khác biệt, có số đo" (AC9), "Cho giảng viên và sinh viên" (AC10), "Bắt đầu với lớp của bạn" (CTA cuối). `landing.spec` khớp đúng danh sách này.
- AC2 (nút ở màn đầu). Given màn đầu When xem Then có đúng **một** nút đỏ `Đăng nhập` và hai nút phụ `Tạo tài khoản`, `Xem video`; không có nút "Dùng thử", không liên kết `/login?thu=…` (P-91-4). Nút có `:active` thu 0,97; hover đổi màu chỉ khi `hover: hover`.
- AC3 (clip sống ở màn đầu). Given màn đầu ở ≥ 720 px When tải Then thấy **poster** tĩnh của clip ngay (là phần tử LCP); sau sự kiện `load` clip mới được tải và phát: lặp liên tục 6–8 s, không tiếng, `muted loop playsinline`, tệp ≤ 1 MB mỗi định dạng (mp4 + webm), `aria-hidden` (nội dung đã có trong chữ). Clip **không tải và không phát** khi `prefers-reduced-motion: reduce` hoặc `Save-Data`; khi tab ẩn thì không phát, và **khi tab hiện lại thì clip tải (nếu chưa) và phát**, trừ `reduce` / `Save-Data` (P-91-5 Q5); lỗi tải → giữ poster, không lỗi console. Ở < 720 px dev chọn poster tĩnh hoặc clip dưới nút miễn đạt AC13.
- AC4 (dừng được, WCAG 2.2.2). Given clip đang chạy hơn 5 s When người dùng muốn dừng Then có nút `Tạm dừng` / `Phát` (vùng chạm ≥ 44 px, `aria-pressed`, dùng được bằng bàn phím) cạnh clip; dừng xong poster hiện lại, trạng thái nhớ trong phiên.
- AC5 (`Xem video`). Given nút `Xem video` When bấm Then video đầy đủ mở trong hộp thoại (`<dialog>` hoặc `Dialog`: focus trap, `Esc` đóng, trả focus về nút, dừng và bỏ nguồn khi đóng); không có request tới tệp video đầy đủ trước khi bấm (`preload="none"`); JS thêm ≤ 1 KB gzip; video không có rãnh âm; bản chữ thay thế (`#video-loi`) có trong DOM. `reduce` → hộp thoại mở tức thì. (Phương án dùng neo `#video` cuộn xuống thay hộp thoại không đạt yêu cầu "mở" của chủ dự án.)
- AC6 (kể chuyện theo cuộn, khung ghim). Given ≥ 1100 px When cuộn khối "Xem EduPilot làm việc" Then một **khung sản phẩm ghim** (`position: sticky`) ở một bên, ba bước chữ cuộn bên kia, mỗi bước là `h3` + một câu: (1) chat riêng có nguồn và dòng "đã ẩn thông tin cá nhân", (2) duyệt bài AI chấm nháp, (3) sổ điểm có giải trình. Khi bước *i* chiếm ≥ 60 % khung nhìn thì khung hiện màn *i* (Chat → Duyệt bài → Sổ điểm), đổi bằng mờ chéo ≤ 200 ms hoặc theo dòng thời gian cuộn; hết khối thì bỏ ghim. Chỉ `transform` / `opacity`. Ba ảnh WebP (mỗi ảnh ≤ 90 KB, có `width` / `height`, nạp lười, `alt` riêng, chữ trong ảnh đọc được).
- AC7 (ghim chỉ khi đủ chỗ). Given < 1100 px hoặc `prefers-reduced-motion: reduce` hoặc không hỗ trợ `animation-timeline` When mở Then **không ghim**: ba bước xếp dọc, mỗi bước kèm ảnh của nó (trạng thái cuối); không bước nào mất chữ hay ảnh.
- AC8 (sơ đồ luồng, gắn cuộn). Given khách cuộn tới "AI soạn nháp, giảng viên quyết, mã tính điểm" When khối đi qua khung nhìn Then ba bước "AI soạn nháp" → "Giảng viên duyệt" → "Mã tính điểm" hiện lần lượt **theo vị trí cuộn** (view timeline; cuộn lên thì chạy ngược), vạch đỏ mảnh nối vẽ dần bằng `transform: scaleX` (`scaleY` ở điện thoại) theo cùng dòng thời gian cuộn; chữ ba bước là HTML thật, vạch đỏ `aria-hidden`; đỏ chỉ ở vạch nối. Không đo thời lượng tính bằng ms vì tốc độ do người cuộn quyết định; `reduce` hoặc không hỗ trợ `animation-timeline` → hiện **trạng thái cuối** (AC14, AC15). (P-91-7 (3b))
- AC9 (số liệu đếm lên, trung thực). Given "Khác biệt, có số đo" đi qua khung nhìn When cuộn Then số nguyên (146) đếm từ 0 tới giá trị cuối **gắn cuộn** (view timeline; cuộn lên đếm ngược), không chạy theo thời gian; "30/30" và "0,05 %" hiện tĩnh; `reduce` hoặc không hỗ trợ → hiện ngay giá trị cuối; giá trị thật nằm trong DOM, bản đếm `aria-hidden`. Mỗi số một câu người ngoài hiểu ngay, **khớp từng chữ với nguồn**, có dòng chú thích riêng "Đo trên dữ liệu mô phỏng." dưới khối (Q-UX91-5, Q-UX91-11, P-91-7 (2), (3b)):
  1. "146 yêu cầu gửi AI: 0 lần lộ họ tên hoặc MSSV trong danh sách lớp", kèm "bộ phát hiện đạt recall 0,960 trên 200 mẫu tự soạn" (nguồn `docs/sprints/6/report.md` dòng 34, 35). **Không** dùng "mọi", "toàn bộ", "được che trước mọi lời gọi AI";
  2. "30/30 sinh viên mẫu có điểm học phần khớp tính tay" (nguồn `docs/sprints/8/report.md` dòng 24);
  3. "Chấm hàng loạt, thời gian đến chữ đầu tiên của chat chỉ tăng 0,05 %" (nguồn `docs/sprints/9/report.md` dòng 25: TTFT +0,05 %); không viết "độ trễ chat" chung chung. Bỏ hẳn số "22 bài / 20 sinh viên": `9/report.md` không có số này (P-91-7 (2)).
  **Chữ v1 từng cần sửa:** dòng 1 ("Quét toàn bộ yêu cầu…", thiếu "trong danh sách lớp" và recall) và dòng 3 ("độ trễ chat tăng 0,05 %"); QC kiểm lại chữ ở bản hiện hành.
- AC10 (vai, hai cột). Given khối "Cho giảng viên và sinh viên" When hiện Then là **hai cột** "Giảng viên" và "Sinh viên" cạnh nhau (≥ 720 px; xếp dọc ở < 720 px), mỗi vai tối đa **4** mục, mỗi mục một câu; **không** tab, không JS (P-91-2). Tên mục dùng nhãn mới: "Hỏi đáp công khai" (không "Threads", Q-UX91-1). **v1 đang đổi vai bằng chuột / mũi tên và còn chữ "Threads": phải sửa.**
- AC10a (ảnh lệch UI sau đợt 3–5). Given đợt 6 làm trước đợt 3–5 When đợt 5 xong Then dev chụp lại ba ảnh khung ghim, clip và poster **một lượt** từ UI mới và ghi vào handoff đợt 5; trước đó dùng ảnh chụp từ giao diện hiện tại.
- AC11 (chữ lớn, khoảng thở). Given 1440 px / 375 px When đo kiểu tính toán Then `h1` ≥ 56 / 36 px, `h2` ≥ 40 / 28 px, chữ thân ≥ 18 / 16 px, khoảng cách dọc giữa hai mục ≥ 96 / 64 px (DESIGN §11.1). *(Đã bỏ điều kiện "một `h2` mỗi khung nhìn": không phải tiêu chí chất lượng, P-91-8 (b).)*
- AC12 (chiều sâu, nổi khi rê chuột). Given khung sản phẩm / ảnh When có chiều sâu và rê chuột (máy tính) Then khung có một bóng tĩnh thuộc họ popover (§7, không màu mới) và nâng ≤ 6 px bằng `translateY` trong ≤ 200 ms; chỉ trong `@media (hover: hover) and (pointer: fine)` và `no-preference`; trên cảm ứng không hiệu ứng hover; focus bàn phím vẫn có vòng focus thấy được; không glow, không nghiêng, không kính mờ.
- AC13 (hiệu năng). Given đo ẩn danh, 4G chậm (Lighthouse mobile, throttle thật, ≥ 3 lượt) When tải `/` Then LCP ≤ 2,5 s ở 375 px và 1440 px; JS của route `/` ≤ 256 KB gzip (v1 đo 216 KB); `Landing` vẫn Server Component; JS thêm chỉ gồm một đoạn nội tuyến ≤ 0,5 KB (nếu còn cần) và phần mở video ≤ 1 KB; CLS ≤ 0,1; không thêm thư viện; hoạt cảnh chỉ đổi `transform` / `opacity`, **trừ**: số đếm (thuộc tính `--n` kiểu `<integer>`, AC9) và đổi màu nhỏ của sơ đồ (`mark-on`, `strike`, AC8) (P-91-7 (3)).
- AC14 (reduced-motion). Given `prefers-reduced-motion: reduce` When mở và cuộn Then mọi khối hiện **ngay ở trạng thái cuối**: không clip chạy (chỉ poster), sơ đồ đủ, số hiện giá trị cuối, bước xếp dọc không ghim; mọi hiệu ứng theo cuộn nằm trong `@media (prefers-reduced-motion: no-preference) { @supports (animation-timeline: view()) { … } }`.
- AC15 (trình duyệt cũ, không dự phòng). Given trình duyệt không hỗ trợ `animation-timeline` When mở Then trang hiện đủ nội dung tĩnh (không `IntersectionObserver` dự phòng cho hiệu ứng theo cuộn, P-91-2) và không khối ẩn vĩnh viễn: trạng thái ẩn ban đầu chỉ áp khi có cờ `data-anim` do script nội tuyến đặt.
- AC16 (trợ năng). Given axe When chạy ở 1440 và 375 Then 0 lỗi serious / critical; thứ bậc `h1` → `h2` → `h3` đúng; mọi chức năng (nút `Xem video`, `Tạm dừng`, đóng hộp thoại) dùng được bằng bàn phím; focus ring thấy được; không truyền nghĩa chỉ bằng màu; mọi ảnh có `alt`.
- AC17 (không rò, giữ nguyên). Given `/` When đọc HTML Then không có `@edupilot.local`, `Edupilot#`, "mật khẩu mặc định" (`landing.spec`); giữ dòng "Dữ liệu trên bản demo là dữ liệu mô phỏng.", "Đồ án tốt nghiệp · Lê Văn Quang Trung" và liên kết GitHub.
- AC18 (đã đăng nhập). Given người đã đăng nhập (có cờ `RETURNING_KEY` / phiên) When mở `/` Then vẫn là "Hôm nay", không thấy trang giới thiệu, không nháy.
- AC19 (nhánh lỗi). Given một ảnh 404, clip lỗi hoặc video đầy đủ lỗi When tải Then khung giữ kích thước (`width` / `height`), có `alt`, bố cục không dịch; clip lỗi → poster; video đầy đủ lỗi → hộp thoại hiện poster + liên kết "Mở video trong tab mới"; không lỗi console.
- AC20 (CTA). Given nút `Đăng nhập` / `Tạo tài khoản` ở màn đầu và CTA cuối When bấm Then tới `/login` / `/register` (không đổi đường dẫn).
- AC21 (e2e / ảnh mốc). Given `landing.spec.ts` When đổi Then sửa danh sách `h2` (AC1), thêm: clip không tải khi `reduce` / `Save-Data`; nút `Tạm dừng`; `Xem video` mở / đóng đúng, không request video trước khi bấm; khung ghim ở 1440 và xếp dọc ở 1024 / `reduce`; hai cột vai (không tab); số hiện giá trị cuối khi `reduce`; QC sửa test theo US này, không nới. Thêm ảnh mốc `landing-1440`, `landing-390` (anonymous, `reduce`, đồng hồ đóng băng).

### Ngoài phạm vi
- Thư viện chuyển động ngoài (cần proposal riêng); trang giới thiệu theo vai; tiếng Anh; video dọc 9:16 (TL-1: để sau); nút / luồng "Dùng thử" (chủ dự án bỏ, P-91-4).

### Phụ thuộc
- US-UX91-09 (clip và video); `DESIGN.md` §11.1 (đã sửa); đo LCP ẩn danh (dev); US-UX91-21 (không gọi refresh ở `/`).

---

## US-UX91-09: Khách muốn clip sản phẩm ở màn đầu và xem video đầy đủ khi chủ động, không bị tự phát tiếng
Ưu tiên: Must · Ước lượng: L · Sprint: 9.1 · **Đợt 6**

### Tiêu chí nghiệm thu
- AC1 (video đầy đủ, nội dung). Given video When xem Then dài 60–90 s (QC đo bằng `ffprobe`; bản v1 handoff chưa ghi độ dài), kịch bản rút từ `DEMO_SCRIPT.md`: sinh viên hỏi chat (tên được che) → hộp thư → duyệt bài AI chấm → sổ điểm giải trình → Hôm nay; 1440 × 900, con trỏ hiện; phụ đề tiếng Việt in vào hình, dấu đúng; không rãnh âm.
- AC2 (clip màn đầu). Given clip lặp When xem Then dài 6–8 s, cắt từ cùng nguồn quay, không phụ đề, lặp liền mạch (khung đầu ≈ khung cuối), không con trỏ giữa chừng gây giật; tệp `gioi-thieu-clip-v1.mp4` (H.264, `+faststart`) + `.webm` + poster `.webp`, mỗi video ≤ 1 MB, poster ≤ 60 KB.
- AC3 (tệp video đầy đủ). Given `frontend/public/landing/` When kiểm Then có `gioi-thieu-v1.mp4`, `.webm`, `.webp` (poster); mỗi định dạng < 8 MB; tên có số phiên bản.
- AC4 (không tự phát video đầy đủ, không tải trước). Given `/` When tải Then thẻ `<video>` của video đầy đủ có `controls`, `preload="none"`, `playsinline`, `poster`, không `autoplay`; không request `.mp4` / `.webm` của nó trước khi người xem bấm `Xem video`.
- AC5 (poster không phá LCP). Given đo LCP ở 1440 px và 375 px When có poster clip trong màn đầu Then LCP ≤ 2,5 s: poster ≤ 60 KB, tải trước (`preload` ảnh) ở ≥ 720 px; ở < 720 px poster không phải ứng viên LCP (h1 là LCP) hoặc LCP vẫn đạt (bài học v1: poster ≥ 84 KB làm LCP di động 3,1 s).
- AC6 (bản chữ thay thế). Given video đầy đủ không tiếng When trình đọc màn hình duyệt Then có đoạn chữ ẩn trực quan (`#video-loi`, `aria-describedby`) chứa đủ các câu phụ đề theo thứ tự và khớp phụ đề in trong video; không thêm chữ lên giao diện.
- AC7 (bàn phím và 375 px). Given 375 px When Tab / chạm Then phát / dừng / tua bằng điều khiển gốc; vùng chạm ≥ 44 px; video co theo bề rộng, không tràn ngang.
- AC8 (tua). Given Safari / Chrome When kéo thanh tua Then phát tiếp (máy chủ trả `206` + `accept-ranges: bytes`).
- AC9 (an toàn dữ liệu quay). Given kịch bản quay When ghi Then **chỉ đọc** trên bản demo (không gửi chat, không duyệt / công bố / sửa điểm; chat mở lại một hội thoại seed có sẵn; không ghi DB, không tốn hạn mức 9router) hoặc quay trên stack riêng `-p ep-video`; không hiện email thật, MSSV thật, khoá, URL nội bộ; tên là dữ liệu mô phỏng (D44). *(Pha 1 dev quay chỉ đọc trên demo theo PM; chấp nhận thay cho yêu cầu stack riêng của bản nháp trước.)*
- AC10 (dựng lại được). Given `scripts/landing-video/` When chạy `run.sh` Then sinh lại cả clip và video đầy đủ cùng poster; README ngắn nêu lệnh.
- AC11 (nhánh lỗi). Given trình duyệt không hỗ trợ WebM When phát Then rơi về MP4; không hỗ trợ cả hai → poster + liên kết tải; không lỗi console.
- AC12 (reduced-motion). Given `reduce` When mở trang Then clip không phát (poster) và video đầy đủ không tự phát; không chuyển động quanh video.
- AC13 (kiểm tay). QC xem hết video đầy đủ và vòng lặp clip: phụ đề không tràn, không trùng cảnh, không lộ dữ liệu cá nhân, clip lặp không giật; ghi giây của lỗi nếu có.

### Ngoài phạm vi
- Nhạc nền; bản dọc 9:16; phụ đề `.vtt` bật tắt (đã có bản chữ thay thế).

### Phụ thuộc
- TL-1 mục A; US-UX91-08 (vị trí clip, nút `Xem video`).

---

## US-UX91-10: Giảng viên muốn mở Hôm nay là thấy việc nào làm trước và các việc còn lại gom theo mức khẩn
Ưu tiên: Must · Ước lượng: M · Sprint: 9.1 · **Đợt 3**

Căn cứ: DESIGN §14.1, §22; Q-UX91-2; `findings.md` UX-5, 6, 18 và mẫu 3.B. Không đổi API: nhóm dựng ở frontend từ `urgency` và `kind` của `GET /me/today` / `GET /courses/{id}/today`.

### Tiêu chí nghiệm thu
- AC1 (nhóm). Given giảng viên / TA có việc When mở `/` Then việc chia nhóm có **nhãn chữ** (`h3`), giữ nguyên thứ tự máy chủ trả trong từng nhóm; nhóm tính ở frontend từ `urgency` và `kind` (không ngưỡng mới ở client; ngưỡng "quá hạn" là của từng nguồn việc đã định ở F14 và backend):
  - **Gấp**: `urgency = overdue` hoặc `urgency = high` (cờ rủi ro hiện có: bài AI "cần xem kỹ", email chưa khớp MSSV, yêu cầu vào lớp) — Q-UX91-8, PM đồng ý;
  - **Khi rảnh**: `kind ∈ {EXAM_SIMILARITY, UNMATCHED_SUBMISSION}` và `urgency = normal`;
  - **Hôm nay**: mọi việc còn lại.
  Nhóm rỗng không hiện; nhãn nhóm kèm số việc ("Gấp · 2").
- AC2 (một việc kế tiếp). Given nhóm đầu tiên không rỗng When hiển thị Then việc đầu tiên của nhóm đó có **một** nút chính đỏ (nhãn động từ theo `CTA[kind]`, ví dụ `Xem bài`); các việc còn lại dùng `Button` biến thể `ghost` có `iconEnd` mũi tên (không thêm biến thể `link`, P-91-2) — không còn chữ đỏ ở mọi hàng. Trong khung nhìn đầu chỉ có **một** nút đỏ (DESIGN §22.2; `ui-antipatterns.sh` phép "hơn một nút đỏ" vẫn sạch).
- AC3 (không lặp). Given giảng viên đang chọn một lớp When xem Then không còn "lớp 761987" lặp ở từng hàng và ở dòng phụ; mã lớp chỉ hiện khi chọn "Tất cả lớp của tôi" (`classLabel`). Khối "Việc cần xử lý" trùng tiêu đề trang bị bỏ; tiêu đề trang vẫn là "N việc cần xử lý hôm nay".
- AC4 (màu có chữ). Given hàng việc When xem Then chấm đỏ chỉ ở `overdue`, hổ phách ở `high`, xám còn lại; **mỗi hàng đã nằm dưới nhãn nhóm bằng chữ** nên mức khẩn không chỉ truyền bằng màu; trình đọc màn hình nghe "Gấp, 2 việc" trước hàng đầu của nhóm.
- AC5 (Lớp cần chú ý và Sắp tới). Given ≥ 1440 px và có `attention` / `upcoming` When xem Then hai khối này nằm ở **cột phải** ≤ 320 px, cạnh danh sách; < 1100 px nằm dưới danh sách như hiện nay. Dữ liệu `attention` / `attention_total` / nút "Xem tất cả (N)" giữ nguyên.
- AC6 (Red Thread). Given bấm một hàng việc When điều hướng Then Red Thread chạy như hiện tại (< 300 ms; `reduce` → không chạy). Không thêm chuyển động.
- AC7 (nhánh lỗi). Given API lỗi lần đầu When tải Then thông báo lỗi + `Thử lại` như hiện tại; lỗi làm mới → dải "Chưa làm mới được…" và giữ dữ liệu cũ; **không** mất nhóm / nhãn đã hiển thị.
- AC8 (rỗng). Given 0 việc When tải Then `EmptyState` "Bạn đã xử lý hết việc." như hiện tại (không nhóm trống).
- AC9 (nhiều việc). Given > 15 việc (`count > actions.length`) When tải Then dòng "Đang hiện N việc đầu tiên." giữ; nhóm vẫn đúng.
- AC10 (phân quyền). Given STUDENT mở `/` When tải Then thấy Hôm nay sinh viên, không có nhóm Gấp / Hôm nay / Khi rảnh; Admin thấy việc hệ thống (không nhóm "Khi rảnh" nếu rỗng); TA thấy cùng nhóm như giảng viên nhưng không có nút bỏ qua thiết lập (như hiện tại, `canDismiss`).
- AC11 (di động 375). Given 375 px When mở Then nhãn nhóm + hàng xếp dọc, nút hành động của hàng xuống dòng riêng (như `ActionList` hiện tại), vùng chạm ≥ 44 px; không tràn ngang.
- AC12 (e2e / ảnh mốc). Given `today.spec.ts` When đổi Then cập nhật kiểm nhóm theo AC1 (fixture đủ `overdue`, `high`, `normal`, `EXAM_SIMILARITY`); thêm ảnh mốc `hom-nay-gv-1440`, `hom-nay-gv-390`; `home-*` (sinh viên) không đổi bởi story này.
- AC13 (đính chính findings). Khối "Lớp cần chú ý" đã tồn tại trong mã (`StaffToday.tsx`) nhưng nằm dưới danh sách; story này chỉ **di chuyển** lên cột phải ở ≥ 1440 px, không thêm dữ liệu mới.

### Ngoài phạm vi
- Đổi luật xếp thứ tự của máy chủ (bậc `Tier`, SRS 4.7); thêm nhóm mới; đổi API.

### Phụ thuộc
- Q-UX91-2, Q-UX91-8 (đã chốt); P-91-2 (dùng `ghost`).

---

## US-UX91-11: Sinh viên muốn Hôm nay có lịch trong ngày ngay đầu trang và không lặp việc
Ưu tiên: Should · Ước lượng: S · Sprint: 9.1 · **Đợt 3**

### Tiêu chí nghiệm thu
- AC1 (thứ tự). Given sinh viên có `timeline` (buổi học / hạn bài trong ngày) When mở `/` ở 1440 px Then thứ tự là: lời chào + ngày → "Việc nên làm tiếp" (một việc, một nút chính) → khối **Hôm nay** (timeline) → "Tiếp tục học"; khối Hôm nay hiện trong khung nhìn đầu ở 1440 × 900 (DESIGN §14.1 mục 3).
- AC2 (không trùng). Given "Tiếp tục học" có nhiều phiên chat cùng tiêu đề, cùng lớp When hiển thị Then chỉ phiên gần nhất cho mỗi cặp (tiêu đề, lớp); tối đa 3 hàng.
- AC3 (ngày giờ). Given hàng "Tiếp tục học" When hiển thị Then dùng định dạng chung của US-UX91-02 ("Thứ Hai, 02/11 · 00:13" hoặc "Hôm qua, …"), không "Thứ 2, 2/11".
- AC4 (một nút chính). Given khung nhìn đầu When xem Then đúng một nút đỏ (`Mở bài tập` hoặc tương đương); không thêm nút đỏ.
- AC5 (nhánh rỗng). Given sinh viên chưa vào lớp When mở `/` Then trạng thái "Nhập mã tham gia lớp" (`StudentToday:59`) như hiện tại, không khối Hôm nay trống.
- AC6 (phân quyền). Given giảng viên / TA mở `/` Then không thấy khối của sinh viên.
- AC7 (di động 375). Given 375 px When mở Then thứ tự như AC1, không tràn ngang; thanh dưới "Hôm nay" active.
- AC8 (ảnh mốc). Ảnh mốc đổi: `home-1440`, `home-390`.

### Ngoài phạm vi
- Đổi thuật toán "việc nên làm tiếp" (backend).

### Phụ thuộc
- US-UX91-02.

---

## US-UX91-12: Giảng viên muốn điểm danh 30 sinh viên bằng một tay, trong đó chỉ người vắng nổi lên
Ưu tiên: Must · Ước lượng: M · Sprint: 9.1 · **Đợt 4**

Căn cứ: DESIGN §14.8; UX.md §5 `/attendance`; Q-UX91-4 (đã chốt: "Có mặt" dùng chấm mực; đỏ chỉ cho Vắng / Vắng phép / Muộn).

### Tiêu chí nghiệm thu
- AC1 (chấm mực). Given buổi chưa lưu, mặc định cả lớp "Có mặt" When mở `/attendance` ở 1440 px Then 24 chấm "Có mặt" màu mực (`--ep-ink`), không đỏ; chọn Muộn / Vắng phép / Vắng thì chấm của lựa chọn đó **đỏ**; số chấm đỏ trên màn = số sinh viên khác mặc định.
- AC2 (không dùng màu một mình). Given hàng sinh viên When trạng thái là Vắng Then ngoài chấm đỏ còn có nhãn chữ trạng thái (như `Có mặt` / `Vắng` ở di động); trình đọc màn hình đọc được trạng thái.
- AC3 (phát biểu). Given hàng sinh viên chưa có phát biểu When xem desktop ≥ 1100 px Then không có chữ "+ phát biểu · +0,25" lặp trên mọi hàng; nút thêm hiện khi hover / focus hàng, hoặc qua phím `P` (giữ phím tắt); hàng đã có phát biểu hiện `+N · +0,25×N`. **Ở 375 px (không có hover)** nút thêm phát biểu hiện thường trực dạng biểu tượng `+` (vùng chạm ≥ 44 px), chỉ bỏ chữ lặp. Giá trị điểm cộng đọc từ công thức lớp, không viết cứng 0,25.
- AC4 (thanh lưu dính đáy ở di động). Given 375 px, danh sách 30 hàng, người dùng đã cuộn xuống hàng 20 When xem Then thanh dưới dính màn hình hiển thị `Lưu điểm danh · 29 có mặt · 1 vắng` (nút chính ≥ 44 px, trên thanh điều hướng đáy, không che hàng cuối); bấm lưu hoàn tất buổi.
- AC5 (desktop). Given 1440 px When xem Then nút `Lưu điểm danh` + trạng thái lưu ("Chưa có thay đổi" / "Đã lưu 14:02") nằm trên cùng cột với đầu bảng; tiêu đề cột và cột tên dính khi cuộn (DESIGN §14.8).
- AC6 (cảnh báo công thức). Given công thức lớp chưa xác nhận When mở Then đúng **một dòng** cảnh báo mỏng phía trên bảng (đã có), có nút `Xem công thức`; ở 375 px dòng này thu một hàng (nhãn ngắn "Công thức chưa xác nhận" + nút), không chiếm quá 56 px.
- AC7 (nhánh lỗi: mất mạng). Given mất mạng giữa lúc điểm danh When bấm lưu Then hàng đợi cục bộ giữ thay đổi, dải mất mạng hiện, tự đồng bộ khi có mạng (UX.md §2.6, hành vi hiện có không đổi); thanh lưu hiện "Chờ mạng".
- AC8 (nhánh lỗi: xung đột). Given hai giảng viên cùng sửa buổi When lưu Then 409 → hiện giá trị của người kia và hai lựa chọn (hành vi hiện có), không mất lựa chọn của người dùng.
- AC9 (phân quyền). Given STUDENT mở `/attendance` Then 403 như hiện nay; TA điểm danh được; không đổi.
- AC10 (60 giây). Given 30 sinh viên, 2 vắng When điểm danh trên 375 px Then hoàn tất ≤ 60 s, 0 hộp thoại xác nhận (UX.md §3).
- AC11 (chuyển động). Given đổi lựa chọn When bấm Then chấm đổi ngay (≤ 100 ms, không hoạt cảnh); `reduce` → như nhau; nháy lưu theo US-UX91-04 AC7.
- AC12 (e2e / ảnh mốc). Given `attendance.spec.ts` When đổi Then sửa theo AC1–AC5 (QC sửa test theo spec; không nới); thêm ảnh mốc `attendance-1440`, `attendance-390`.

### Ngoài phạm vi
- Đổi cách tính điểm cộng / trừ (code thuần `internal/grade`); đổi 4 trạng thái.

### Phụ thuộc
- Q-UX91-4 (đã chốt); US-UX91-04 AC7 (nháy lưu); US-UX91-03.

---

## US-UX91-13: Sinh viên muốn chat riêng gọn gàng, bắt đầu hỏi được ngay
Ưu tiên: Must · Ước lượng: M · Sprint: 9.1 · **Đợt 5**

Căn cứ: DESIGN §14.2; `findings.md` UX-11 và mẫu 3.C.

### Tiêu chí nghiệm thu
- AC1 (một Panel). Given `/chat` ở 1440 px When mở Then có **một** Panel hội thoại rộng tối đa 840 px, căn giữa **trong phần bên phải cột lịch sử** (cột lịch sử rộng 240 px + khoảng cách 24 px nằm ngoài phép căn; ở 1440 px lề trái / phải của Panel trong phần còn lại bằng nhau, P-91-13); lịch sử phiên là cột phẳng bên trái (đường kẻ 1 px, không Panel thứ hai), thu gọn được; `ui-antipatterns.sh` phép "nhiều Panel trong một vùng làm việc" sạch.
- AC2 (lịch sử). Given ≥ 1 phiên trước When xem lịch sử Then mỗi hàng hiện tiêu đề (tối đa 2 dòng, cắt có dấu `…`) + ngày giờ theo US-UX91-02; **không** hiện biểu tượng thùng rác trên mọi hàng; xoá nằm trong menu `⋯` của hàng.
- AC3 (xoá có hoàn tác). Given chọn xoá phiên When xác nhận bằng một chạm Then phiên biến ngay và hiện dòng "Đã xoá · Hoàn tác" 5 giây (hành vi `useUndoableAction` hiện có); không hộp thoại xác nhận; quá 5 giây xoá thật.
- AC4 (Phiên mới). Given phiên hiện tại chưa có tin nhắn When xem Then **không** có nút `Phiên mới` (không nút khoá); có ≥ 1 tin → hiện nút `Phiên mới` (nút phụ, không đỏ).
- AC5 (trạng thái đầu). Given phiên trống When mở Then lời chào ngắn + tối đa **3** gợi ý dạng hàng chữ (không nút bo): các gợi ý dựng từ dữ liệu `GET /me/today` đã có (việc nên làm, hạn bài) và hai câu cố định ("Điểm quá trình của tôi hiện là bao nhiêu?", "Tuần này tôi có những buổi học nào?"); bấm gợi ý điền vào ô nhập (không tự gửi). Khoảng trống phía trên ô nhập ≤ 1/3 chiều cao khung.
- AC6 (riêng tư). Given màn chat When xem Then không có từ kỹ thuật AI (RAG, PII, provider…); dòng "Đã ẩn N thông tin cá nhân trước khi gửi cho AI" giữ nguyên (DESIGN §14.2); lời giới thiệu phiên trống giữ cam kết "chỉ bạn đọc được" như hiện có ở khung máy chủ (không thêm `meta` mới).
- AC7 (ô nhập). Given đang trả lời When gõ tiếp Then gõ được; nút `Dừng` hiện; tự cuộn dừng khi kéo lên (UX.md §5); nháp tự lưu 2 s; tất cả giữ nguyên.
- AC8 (nhánh lỗi). Given stream đứt When mất mạng Then tự nối lại, giữ phần đã sinh (hành vi hiện có), hiện dải mất mạng; gửi lỗi → giữ nội dung ô nhập + `Gửi lại` (`Idempotency-Key`).
- AC9 (phân quyền). Given giảng viên / TA mở `/chat` Then không đọc được chat riêng của sinh viên (như hiện tại; chỉ thấy phản hồi ticket đã escalate). STUDENT khác không thấy phiên của người khác.
- AC10 (di động 375). Given 375 px When mở Then ô nhập luôn trên bàn phím ảo; lịch sử là nút `Phiên trước (N)` mở bảng trượt từ đáy (không chiếm đầu trang); vùng chạm ≥ 44 px; không tràn ngang.
- AC11 (chuyển động). Given tin nhắn mới When hiện Then `opacity` + `translateY(4px)` 160 ms, **tắt** trong khi stream đang chạy token; nút `Gửi` có `:active`; `reduce` → hiện tức thì.
- AC12 (e2e / ảnh mốc). Given `private-chat.spec.ts` When đổi Then sửa theo AC1–AC5; ảnh mốc đổi: `chat-1440`, `chat-390`.

### Ngoài phạm vi
- Đổi hành vi SSE / agent; trích dẫn (citation) trong câu trả lời.

### Phụ thuộc
- US-UX91-02, US-UX91-04; Q-UX91-6.

---

## US-UX91-14: Giảng viên muốn hàng chờ chấm nói rõ bài nào, vì sao cờ, và bấm vào đâu
Ưu tiên: Should · Ước lượng: S · Sprint: 9.1 · **Đợt 4**

### Tiêu chí nghiệm thu
- AC1 (một bài tập). Given bộ lọc "Bài tập" đang chọn một bài hoặc lớp chỉ có một bài When xem bảng Then cột "Bài tập" ẩn; tên bài hiện một lần trên tiêu đề bảng; nhiều bài → cột hiện, tên một dòng cắt `…`, đầy đủ trong tooltip.
- AC2 (lý do cờ). Given bài có "Lý do cờ" When xem Then lý do hiện ngay dưới tên sinh viên (trước cột điểm); bài không cờ không chừa khoảng trống.
- AC3 (bấm được). Given hàng bài When rê / focus Then có hover nền, hàng có liên kết `Duyệt` ở cuối (trung tính, mũi tên) và toàn hàng bấm được; Enter mở `/grading/[id]`; con trỏ `pointer` chỉ khi `hover: hover`.
- AC4 (bộ lọc). Given thanh lọc When xem Then chip "Cần xem kỹ" cùng hàng và căn đáy với hai ô chọn; ô "Bài tập" không rộng hơn nội dung dài nhất + 24 px.
- AC5 (nút khoá). Given hàng chờ rỗng When xem Then `Chấm tất cả` khoá kèm lý do (US-UX91-03); có bài → bấm được.
- AC6 (nhánh lỗi). Given API lỗi When tải Then `ApiErrorNotice` + `Thử lại`, giữ bộ lọc đã chọn.
- AC7 (phân quyền). Given STUDENT mở `/grading` Then 403; TA mở được (như hiện nay).
- AC8 (di động 375). Given 375 px When xem Then hàng xếp dọc (tên, lý do cờ, điểm, trạng thái), `Duyệt` xuống dòng riêng ≥ 44 px; không tràn ngang.
- AC9 (ảnh mốc). Thêm ảnh mốc `grading-queue-1440`.

### Ngoài phạm vi
- Thay đổi logic chấm; thêm bộ lọc mới.

### Phụ thuộc
- US-UX91-03, 05.

---

## US-UX91-15: Giảng viên muốn duyệt bài mà luôn thấy nút Duyệt và tiến độ
Ưu tiên: Must · Ước lượng: M · Sprint: 9.1 · **Đợt 4**

Căn cứ: DESIGN §14.13; UX.md §5 `/grading/[id]`; `findings.md` UX-15 và mẫu 3.D. Giữ nguyên hai cột 52–58 % / 42–48 %.

### Tiêu chí nghiệm thu
- AC1 (một hàng một tiêu chí). Given màn duyệt bài When xem cột phải Then mỗi tiêu chí là **một hàng**: tên tiêu chí · ô điểm · `/{tối đa}` cùng dòng; không còn nhãn "Điểm {tên tiêu chí}" lặp ở góc phải; tên tiêu chí dài xuống dòng không đè lên ô điểm ở 1440 px và 1100 px.
- AC2 (trích dẫn). Given tiêu chí có đoạn trích When xem Then đoạn trích hiện dạng dòng trích có biểu tượng nhỏ dưới tiêu chí; câu "Đoạn trích từ bài làm · bấm để xem trong bài" **không** lặp ở mỗi đoạn (chỉ một chú thích ở đầu cột, hoặc tooltip của biểu tượng); bấm dòng trích → bài làm cuộn tới và tô sáng đoạn (nền `--ep-red-soft`): nền được **giữ ≈ 1 s** (trạng thái, không phải chuyển động) rồi **mờ đi trong 200 ms**; `reduce` → giữ ≈ 1 s rồi tắt tức thì (PM, TL-2 câu 3).
- AC3 (thanh hành động dính). Given cột phải dài hơn khung nhìn (1440 × 900) When mở và khi cuộn Then thanh dưới cột phải luôn **nhìn thấy đủ**, gồm `Duyệt bài` (nút đỏ duy nhất), tiến độ "Đã duyệt N/M", gợi ý phím `A`; không bị cắt ở mép dưới; có `‹ ›` chuyển bài trước / sau. **N** = số bài đã duyệt; **M** = tổng cố định của hàng chờ lúc mở màn (không giảm khi duyệt: duyệt thêm một bài thì N tăng 1, M giữ nguyên) (P-91-10 (3)).
- AC4 (phím tắt). Given focus trong trang When nhấn `A` Then duyệt và sang bài kế (UX.md §5); phím có nhãn gợi ý trong thanh; `‹` / `›` đổi bài; không hoạt cảnh khi dùng phím.
- AC5 (lệch điểm). Given hai lượt chấm lệch > 1 điểm When xem Then thông báo hổ phách inline cạnh tiêu chí ("Lệch 1,5 điểm so với lượt 2"), không modal.
- AC6 (không mất chữ). Given đang sửa nhận xét / điểm When tải lại trang Then nháp khôi phục (UX.md §2.3); đổi điểm tự lưu lạc quan với trạng thái lưu tại chỗ.
- AC7 (nhánh lỗi). Given lưu / duyệt lỗi mạng When bấm `Duyệt bài` Then báo lỗi tại thanh dưới, giữ nguyên điểm đã nhập, nút `Thử lại`; xung đột phiên bản 409 hiện giá trị của người kia và hỏi giữ bên nào.
- AC8 (không tự công bố; sinh viên không thấy). Given duyệt xong When xem Then bài ở trạng thái "Đã duyệt", **không** tự công bố cho sinh viên (công bố là thao tác riêng, giảng viên bấm). Sinh viên **không có màn xem bài tập** ở 9.1, nên kiểm bằng API: STUDENT gọi `GET /courses/{id}/submissions` (và chi tiết bài) → 403; điểm nháp / nhận xét không lọt vào phản hồi nào dành cho sinh viên khi bài còn mở (P-91-10 (1)).
- AC9 (phân quyền). Given STUDENT mở `/grading/[id]` Then 403; STUDENT chưa thấy điểm nháp / ghi chú quan sát / đáp án khi bài còn mở (không đổi); TA duyệt được theo quyền hiện có.
- AC10 (di động 375). Given 375 px When mở Then hai cột thành hai chế độ "Bài làm" / "Chấm" (tab); thanh hành động dính đáy cao ≥ 44 px, không tràn ngang.
- AC11 (e2e / ảnh mốc). Given `assignment-lifecycle.spec.ts` When đổi Then sửa theo AC1–AC4; thêm ảnh mốc `grading-review-1440`.

### Ngoài phạm vi
- Thay đổi cách AI chấm hai lượt, rubric; thêm phím tắt mới ngoài `A`, `‹`, `›`.

### Phụ thuộc
- US-UX91-02, 03, 04.

---

## US-UX91-16: Giảng viên muốn đọc sổ điểm mà số thẳng hàng và biết ai còn thiếu điểm
Ưu tiên: Must · Ước lượng: M · Sprint: 9.1 · **Đợt 4**

Căn cứ: DESIGN §5 (`tabular-nums`), §14.10; Q-UX91-3 (đã chốt: "Đủ" = đã có đủ điểm thành phần, nhãn **Đủ điểm / Thiếu điểm**; BA đề xuất bỏ cột nếu mọi hàng đều Đủ điểm). Tính điểm là code thuần; **không** đổi giá trị, chỉ đổi cách hiển thị.

### Tiêu chí nghiệm thu
- AC1 (không làm tròn khi hiển thị). Given giá trị điểm từ máy chủ (chuỗi decimal) When hiển thị Then UI **không** làm tròn; mỗi cột hiện cùng số chữ số thập phân = số lớn nhất trong cột (tối thiểu 2), đệm `0` bên phải ("7,915" → cột Quá trình hiện 3 chữ số cho mọi hàng: "6,000", "5,500", "7,915"); dấu phẩy thập phân; căn phải; `font-variant-numeric: tabular-nums`. **Ô nhập** (Giữa kỳ, Thi cuối kỳ…) cũng đệm `0` theo số chữ số của cột khi **không** focus; khi focus hiện **giá trị gốc** (thô, như đã lưu) để sửa, và giá trị lưu là đúng chữ người dùng nhập, không làm tròn (P-91-10 (2)).
- AC2 (tiêu đề cột). Given bảng When xem Then tiêu đề cột **một dòng**, cùng chiều cao; trạng thái "đã công bố" của cột thành chấm nhỏ trước tiêu đề (chú giải một dòng ở chân bảng: "● đã công bố"), thay cho dòng phụ "Đã công bố" + `⋯`; menu `⋯` của cột hiện khi hover / focus tiêu đề.
- AC3 (một thanh cuộn, giữ ảo hoá). Given sổ điểm When mở Then bảng có **một** vùng cuộn dọc cao bằng khung nhìn còn lại (`calc(100dvh − đầu trang)`), tiêu đề cột và cột "Sinh viên" dính **trong vùng cuộn** (`position: sticky`), giữ ảo hoá của `DataTable` (T1 1.000 SV, UX.md §3); không thêm vùng cuộn dọc thứ hai bên trong Panel; hàng cuối không bị cắt (P-91-1).
- AC4 (Đủ điểm / Thiếu điểm). Given sinh viên thiếu ≥ 1 điểm thành phần bắt buộc When xem Then cột "Trạng thái" hiện chữ "Thiếu điểm" (hổ phách, kèm chấm); sinh viên đủ điểm để trống; nếu **không có** hàng nào thiếu thì **ẩn cả cột**; tiêu đề phụ trang hiện "Thiếu điểm: N sinh viên" khi N > 0 (liên kết lọc nhanh). Giá trị "Đủ" cũ không còn trên màn.
- AC5 (xung đột 409). Given hai giảng viên sửa cùng ô When lưu Then ô viền hổ phách + dòng dưới ô "{tên} vừa đổi thành 6,5 · Giữ của tôi · Lấy 6,5" (UX.md §5); không mất giá trị người dùng đã nhập.
- AC6 (sửa tại chỗ). Given Enter / Tab When sửa ô Then xuống dòng / sang ô (như hiện tại); ô lưu xong nháy nền xanh nhạt 200 ms (US-UX91-04 AC7); `reduce` → đổi màu tức thì, giữ ≥ 300 ms.
- AC7 (nút chính). Given công thức chưa xác nhận When xem Then `Chốt điểm` khoá kèm lý do (US-UX91-03); công thức đã xác nhận và đủ điều kiện → nút bấm được; `Chốt điểm` vẫn dùng `<ConfirmIrreversible>` nêu hậu quả bằng số (không đổi).
- AC8 (nhánh lỗi). Given API lỗi When tải Then `ApiErrorNotice` + `Thử lại`; sổ điểm đã chốt hiện chỉ đọc, không ô sửa.
- AC9 (phân quyền). Given STUDENT mở `/gradebook` Then 403; chỉ TEACHER chốt điểm; TA xem / sửa theo quyền hiện có; không ai thấy điểm của lớp khác.
- AC10 (di động 375). Given 375 px When xem Then cuộn ngang có chủ đích với **cột tên dính** (DESIGN §17); không tràn ngang cả trang; ô sửa ≥ 44 px.
- AC11 (e2e / ảnh mốc). Given `gradebook.spec.ts` When đổi Then sửa kiểm định dạng số và cột trạng thái (QC sửa test theo spec); ảnh mốc đổi: `gradebook-1440`, `gradebook-390`.

### Ngoài phạm vi
- Quy tắc làm tròn của quy chế (code thuần `internal/grade`, UI **không** làm tròn khi hiển thị, P-91-1); đổi sang cuộn theo trang bằng `useWindowVirtualizer` (L, đụng AC hiệu năng T1); thêm cột; xuất tệp.

### Phụ thuộc
- Q-UX91-3 (đã chốt); US-UX91-03, 04; `docs/specs/FEAT-gradebook`.

---

## US-UX91-17: Giảng viên muốn Insights và Quan sát AI đọc một lượt, không khung lồng khung, không số liệu lặp
Ưu tiên: Should · Ước lượng: M · Sprint: 9.1 · **Đợt 5**

### Tiêu chí nghiệm thu
- AC1 (Lỗ hổng kiến thức). Given `/insights` có báo cáo When xem Then mỗi khối ("Chủ đề nên dạy lại", "Tài liệu chưa đề cập", "So với báo cáo ngày …") là **một** Panel; các chủ đề bên trong ngăn bằng đường kẻ 1 px, không khung có viền / bo góc riêng. Kiểm bằng `panels.spec` NEST (đã có từ sprint 5.5) và QC xem ảnh; **không** thêm phép tự động vào `ui-antipatterns.sh` (P-91-2).
- AC2 (chữ). Given trang When xem Then áp US-UX91-01 dòng 23–28 (không còn câu giải thích; hộp riêng tư thành `meta` ngắn).
- AC3 (Quan sát AI: không lặp). Given `/observability` When xem Then mỗi chỉ số (yêu cầu, độ trễ p95, tỷ lệ lỗi, chuyển giảng viên, lần che thông tin) hiện **đúng một chỗ**: số "hôm nay" là một dòng chữ (không dải 5 ô KPI cùng cỡ), số 7 ngày là bảng; hàng đầu bảng không trống.
- AC4 (khoảng cách). Given hai khối liên tiếp When xem Then khoảng cách trên tiêu đề khối ≥ 32 px (§6: nhiều hơn dưới), không dính sát Panel trước.
- AC5 (nội dung prompt). Given giảng viên mở `/observability` Then **không** đọc được nội dung câu hỏi / prompt (chỉ ADMIN, có `audit_log`); không đổi. **Kiểm sau khi nối API thật (nợ P10, P-91-15): TC UX17-8 «KHÔNG KIỂM ĐƯỢC» ở sprint 9.1 vì `/observability` đang dữ liệu mô phỏng, không tính FAIL.**
- AC6 (nhánh lỗi / rỗng; **kiểm sau khi nối API thật, nợ P10 — UX17-9 không tính FAIL ở 9.1, P-91-15**). Given chưa có báo cáo Insights When mở Then `EmptyState` + một nút `Tạo báo cáo mới`; đang tạo → chỉ báo giai đoạn ("Đang nhóm 66 câu hỏi…"), rời trang được, báo cáo cũ vẫn xem được (UX.md §5).
- AC7 (phân quyền). Given STUDENT / TA không có quyền Then 403; giảng viên không đọc chat riêng chưa escalate qua trang này.
- AC8 (di động). Given 375 px Then khối xếp dọc, không tràn ngang.
- AC9 (chuyển động). Không thêm chuyển động; `Tạo báo cáo mới` có `:active` theo US-UX91-04.
- AC10 (ảnh mốc). Thêm ảnh mốc `insights-1440`, `observability-1440` (QC quyết số lượng).

### Ngoài phạm vi
- Biểu đồ Analytics (UX-17, để sau); thêm chỉ số mới.

### Phụ thuộc
- US-UX91-01, 07.

---

## US-UX91-18: Giảng viên muốn danh sách tài liệu không bị cắt chữ và bớt lặp
Ưu tiên: Should · Ước lượng: S · Sprint: 9.1 · **Đợt 5**

### Tiêu chí nghiệm thu
- AC1 (không cắt). Given `/documents` When xem ở 1440 px và 1100 px Then ô "Loại" hiển thị đủ "Quy chế học phần", "Tài liệu khác"… không cắt chữ giữa từ (độ rộng theo nội dung dài nhất hoặc hiển thị chữ + đổi bằng menu).
- AC2 (không lặp nguồn). Given tài liệu thuộc lớp đang xem When xem hàng Then **không** có dòng "Chia sẻ từ {lớp}"; dòng đó chỉ hiện khi tài liệu được chia sẻ **từ lớp khác**.
- AC3 (công tắc). Given hai công tắc "Dùng cho AI", "Hiện cho sinh viên" When xem Then bật / tắt khác nhau cả ở màu **lẫn** nhãn chữ cho trình đọc màn hình ("Dùng cho AI: bật"); trạng thái bật / tắt đồng nhất màu ở mọi hàng; chạm ≥ 44 px ở 375 px.
- AC4 (không rò). Given tài liệu "Đáp án" When sinh viên xem Library Then không hiện khi bài còn mở; công tắc "Hiện cho sinh viên" tắt (không đổi).
- AC5 (nhánh lỗi). Given đổi công tắc lỗi mạng When bấm Then công tắc hoàn lại + dòng lỗi tại hàng + `Thử lại` (cập nhật lạc quan hiện có).
- AC6 (phân quyền). Given STUDENT mở `/documents` Then 403 / không thấy; sinh viên chỉ vào `/library`.
- AC7 (di động). Given 375 px Then hàng tài liệu xếp dọc, công tắc ≥ 44 px, không tràn ngang.
- AC8 (ảnh mốc). Thêm ảnh mốc `documents-1440` (QC quyết).

### Ngoài phạm vi
- Thêm cột / bộ lọc mới.

### Phụ thuộc
- US-UX91-01, 02.

---

## US-UX91-19: Người dùng muốn danh sách Hỏi đáp công khai dễ quét, không ký tự thừa
Ưu tiên: Should · Ước lượng: S · Sprint: 9.1 · **Đợt 5**

### Tiêu chí nghiệm thu
- AC1 (đoạn trích sạch). Given thread có nội dung bắt đầu bằng ký tự markdown (`-`, `·`, `*`, `#`, `>` hoặc chuỗi `- -`) When xem danh sách Then đoạn trích bỏ các ký tự đầu dòng đó, gộp xuống dòng thành khoảng trắng, cắt ≤ 140 ký tự kèm `…`; không bao giờ hiện `- - ` hay `· ` ở đầu đoạn trích.
- AC2 (nhãn lặp). Given tab "Câu hỏi" mà ≥ 80 % thread cùng trạng thái "Chờ xác nhận" When xem Then nhãn trạng thái **chỉ** hiện trên thread khác trạng thái đa số; trạng thái đa số được nêu một lần ở dòng đầu danh sách ("{N} câu chờ xác nhận"). Trạng thái "Đã được giảng viên xác nhận" / "Đã sửa bởi giảng viên" vẫn hiện trên **từng** thread có nó (UX.md §2.7).
- AC3 (an toàn nội dung). Given thread chưa xác nhận When sinh viên xem danh sách Then **không** hiện câu trả lời AI chưa được giảng viên xác nhận (như hiện nay).
- AC4 (bộ lọc di động). Given 375 px When xem Then "Bộ lọc" là nút có biểu tượng + số bộ lọc đang bật, vùng chạm ≥ 44 px, mở bảng trượt từ đáy; không còn nhãn chữ trơ.
- AC5 (ngày giờ). Given hàng thread Then dùng định dạng chung US-UX91-02 (không `00:10 02-11`).
- AC6 (nhánh lỗi). Given đoạn trích rỗng / chỉ có ký tự markdown Then hiện tiêu đề, không dòng trống; API lỗi → `ApiErrorNotice` + `Thử lại`.
- AC7 (phân quyền). Given STUDENT ngoài lớp When mở thread của lớp khác Then 403; hộp PII khi đăng giữ nguyên.
- AC8 (tên hiển thị). Given US-UX91-07 Then h1 "Hỏi đáp công khai".
- AC9 (ảnh mốc). Ảnh mốc đổi: `threads-1440`, `threads-390`.

### Ngoài phạm vi
- Hiển thị markdown đầy đủ trong chi tiết thread (đã có `markdown.ts`).

### Phụ thuộc
- US-UX91-02, 07.

---

## US-UX91-20: Sinh viên muốn hiểu điểm của mình trong một khối giải trình, không đọc cùng số hai lần
Ưu tiên: Should · Ước lượng: S · Sprint: 9.1 · **Đợt 5**

### Tiêu chí nghiệm thu
- AC1 (một khối). Given `/me` có điểm When xem Then chỉ **một** khối giải trình; mỗi dòng: tên thành phần · trọng số % · điểm ("Giữa kỳ · 20 % · 7,5"), "Căn cứ · tr. 1" nằm cùng dòng; khối liệt kê thành phần riêng trước đó bị bỏ. **Không có cột "đóng góp"** ở 9.1: `/courses/{id}/me/grades` không trả giá trị này và UI không tự tính điểm; đổi hợp đồng API không thuộc sprint giao diện (P-91-11).
- AC2 (đầu trang). Given tiêu đề When xem Then `h1` "Kết quả của tôi" không còn dòng phụ mã lớp trần "761987" (US-UX91-01 dòng 32); câu tóm tắt "Điểm quá trình hiện tại của bạn là 8,25 trên 10." giữ.
- AC3 (số chính xác). Given điểm When hiển thị Then dùng đúng giá trị máy chủ trả (không tự tính lại hay làm tròn ở UI; điểm là code thuần).
- AC4 (không lộ). Given sinh viên chưa được công bố điểm nháp / ghi chú quan sát / nhãn rủi ro When xem Then **không** hiện; chỉ điểm đã công bố.
- AC5 (yêu cầu xem lại). Given điểm đã công bố When bấm "Yêu cầu xem lại" Then hành vi hiện có (không đổi), đúng một nút phụ.
- AC6 (nhánh lỗi / rỗng). Given chưa có điểm công bố Then `EmptyState` "Chưa có điểm nào được công bố." (không câu giải thích); API lỗi → `ApiErrorNotice` + `Thử lại`.
- AC7 (phân quyền). Given sinh viên A When gọi dữ liệu của sinh viên B Then 403 (danh tính từ JWT, không từ tham số).
- AC8 (di động 375). Given 375 px (< 720 px) Then «Thời gian học» và «Bài tập» **gập sẵn** (mở dần bằng một chạm, nội dung giữ nguyên, tiêu đề kèm số tóm tắt) và trang dài ≤ 1.600 px cho lớp mẫu khi hai khối đang gập (hiện 2.645 px; P-91-14), không tràn ngang, dòng giải trình xuống dòng gọn.
- AC9 (e2e). Given `me.spec.ts` When đổi Then sửa theo AC1–AC2 (QC sửa theo spec).
- AC10 (ảnh mốc). Thêm ảnh mốc `me-390` (QC quyết).

### Ngoài phạm vi
- Đổi cách giải trình điểm ở backend (`internal/grade`).

### Phụ thuộc
- US-UX91-01, 02; `docs/specs/FEAT-gradebook` (giải trình từng dòng).

---

## US-UX91-21: Khách muốn trang giới thiệu không gọi làm mới phiên khi chưa từng đăng nhập
Ưu tiên: Should · Ước lượng: S · Sprint: 9.1 · **Đợt 1** (độc lập trang mới)

Căn cứ: UX-3(d) và TL-2 (`AuthProvider.tsx` luôn gọi `refreshSession()`; mỗi lần vào trang công khai có `401 /auth/refresh` ở console).

### Tiêu chí nghiệm thu
- AC1 (không gọi). Given trình duyệt chưa có cờ "đã từng đăng nhập ở máy này" (`RETURNING_KEY`) When mở `/` Then **không** có request `POST /auth/refresh`; trạng thái phiên là "anonymous" ngay; console không có lỗi 401.
- AC2 (người quay lại). Given trình duyệt có cờ và cookie phiên hợp lệ When mở `/` Then vẫn làm mới phiên và hiện "Hôm nay" như hiện tại (không nháy trang giới thiệu).
- AC3 (an toàn). Given cookie `ep_rt` còn nhưng mất cờ (xoá `localStorage`) When mở `/` Then thấy trang giới thiệu có nút `Đăng nhập`; bấm → `/login` vẫn thử làm mới bằng cookie (không đổi bảo mật), nếu phiên còn thì vào thẳng ứng dụng.
- AC4 (nhánh lỗi). Given `localStorage` bị chặn (chế độ riêng tư) When mở `/` Then coi như không có cờ, không ném lỗi.
- AC5 (phân quyền). Không đổi: route sau đăng nhập vẫn do `AuthGate` chuyển `/login?next=…` khi chưa có phiên.
- AC6 (e2e). Given `landing.spec.ts` (dòng 17 "không gọi API nào ngoài refresh") When đổi Then mong đợi mới: **không gọi API nào** ở ẩn danh; bổ sung test "có cờ ⇒ có gọi refresh"; QC sửa theo story này, không nới.

### Ngoài phạm vi
- Các route công khai khác (`/login`, `/register`…): vẫn thử làm mới bằng cookie như hiện tại.

### Phụ thuộc
- `shared/session/AuthProvider.tsx`, `app/page.tsx` (`RETURNING_KEY`); không đổi API.

---

## US-UX91-22: GỠ (P-91-4)
Chủ dự án **bỏ "Dùng thử"** khỏi sprint 9.1 ngày 2026-11-08 (`proposals.md` P-91-4, huỷ P-91-3): lý do "lằng nhằng" (đụng auth, sandbox, cron, hạn mức 9router). Không làm: endpoint `/demo/sessions`, cờ `DEMO_MODE`, lớp / tài khoản sandbox, đặt giá model 9router, nút "Dùng thử" ở trang giới thiệu. Số `US-UX91-22` để trống, không đánh số lại các story khác. Không còn AC nào của FEAT-uiux-91 nói tới "Dùng thử".

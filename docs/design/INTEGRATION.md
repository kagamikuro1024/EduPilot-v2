# Tích hợp bộ UI kit "Red Thread / Academic Instrument" vào kế hoạch thi công

Các file `DESIGN.md`, `AGENT_PROMPT.md`, `DESIGN_TOKENS.css`, `edupilot-ui-v3.html` và logo trong thư mục này là bản gốc của chủ dự án, **giữ nguyên văn**. File này nói chúng khớp vào phần còn lại của bộ tài liệu thế nào.

## 1. Ai quyết định cái gì

| Chủ đề | File có thẩm quyền |
| --- | --- |
| Diện mạo: màu, chữ, khoảng cách, bo góc, bóng, icon, logo, chuyển động | `design/DESIGN.md` §3–§11, `DESIGN_TOKENS.css` |
| Bố cục và nội dung khung nhìn đầu của từng route; điều hướng theo vai trò; từ vựng component; lời văn giao diện | `design/DESIGN.md` §1–§2, §10, §12–§14, §19 |
| Danh sách phản mẫu và định nghĩa "xong" về mặt giao diện | `design/DESIGN.md` §21, §22 |
| Hành vi khi mạng xấu, tự lưu nháp, cập nhật lạc quan, idempotency phía client, ngân sách hiệu năng, lớp dữ liệu frontend | `UX.md` |
| Hành vi sản phẩm, quyền, tiêu chí nghiệm thu | `PRD.md` |
| API, schema, thư viện | `ARCHITECTURE.md` |

Khi hai file nói khác nhau về cùng một thứ: **`DESIGN.md` thắng về hình thức và bố cục; `UX.md` thắng về độ bền và hiệu năng; `PRD.md` thắng về hành vi.** Không bên nào được dùng làm cớ để thêm tính năng mới.

Đường dẫn: `AGENT_PROMPT.md` và `DESIGN.md` nhắc tới `PRD.md`, `ARCHITECTURE.md` ở gốc repo; trong repo này chúng nằm ở `docs/PRD.md`, `docs/ARCHITECTURE.md`, `docs/design/DESIGN.md`.

## 2. Các điểm lệch nhau và cách giải quyết

| # | `DESIGN.md` nói | Bản `UX.md` cũ nói | Chốt |
| --- | --- | --- | --- |
| 1 | `/attendance`: không card; có trạng thái lưu + nút chính `Lưu điểm danh` | Điện thoại dùng "thẻ lớn"; lưu lạc quan từng thay đổi, không có nút lưu | Không card: điện thoại dùng **hàng** ngăn bằng đường kẻ, vùng chạm ≥ 44 px. Mỗi thay đổi vẫn tự lưu lạc quan và hiện trạng thái lưu ("Đã lưu 14:02" / "Đang chờ mạng"). `Lưu điểm danh` là hành động **hoàn tất buổi**: đẩy hết hàng đợi, đánh dấu buổi đã điểm danh. Rời trang khi chưa bấm không mất dữ liệu |
| 2 | Sinh viên không thấy thanh 0,76; thấy câu "AI chưa đủ chắc chắn về câu này" + `Nhờ giảng viên hỗ trợ` | Có `<ConfidenceBar>` trong chat | Theo `DESIGN.md`. Con số độ tin cậy chỉ hiện cho TA / TEACHER / ADMIN (`/inbox`, `/observability`). Bỏ `<ConfidenceBar>` khỏi màn sinh viên |
| 3 | Tường lửa PII chạy khi gửi, hộp thoại bảo vệ có đúng hai lối | Kiểm tra khi đang gõ | Cả hai: khi đang gõ hiện `PIIProtectionNotice` dạng dòng phía trên composer (đúng §10.12); khi bấm gửi mà vẫn còn nội dung cá nhân thì mở Dialog hai lối |
| 4 | Tránh toast "Thành công!"; dùng xác nhận tĩnh lặng tại chỗ | Toast Hoàn tác | Hoàn tác hiện như một dòng xác nhận tĩnh lặng gắn vào nơi vừa thao tác ("Đã đánh vắng · Hoàn tác"), tự biến sau 5 s. Toast nổi chỉ dùng cho lỗi và sự kiện đến từ nơi khác |
| 5 | Giao diện sáng, `color-scheme: light` | Có chế độ sáng / tối | v2 chỉ làm giao diện sáng. Token ngữ nghĩa đủ để thêm tối về sau, nhưng không xây bây giờ |
| 6 | Bóng gần như không; card hạn chế tối đa | `<DataTable>`, skeleton… không nói về hình thức | Mọi primitive theo §10; `UX.md` chỉ thêm yêu cầu ảo hoá, phân trang con trỏ, điều hướng bàn phím |
| 7 | Có route `/` "Hôm nay" theo vai trò | Không có | Thêm module M14 vào PRD + API `today` (mục 4 dưới đây) |
| 8 | `/me`: thời gian học là xu hướng nhỏ ở cuối trang; không gamify | Có biểu đồ thời gian học | Theo `DESIGN.md`: nhỏ, ở dưới, không streak, không xếp hạng |
| 9 | `/students/[id]`: tóm tắt bằng lời trước, số liệu sau; không 6 thẻ KPI | "KPI + 5 biểu đồ" | Theo `DESIGN.md`: câu nhận định rủi ro trước; biểu đồ chỉ khi cần so sánh / xu hướng, đặt trong tab tương ứng |
| 10 | `/observability`: dải trạng thái gọn, không thẻ số liệu | "thẻ vận hành / chi phí / chất lượng" | Theo `DESIGN.md`: một dải trạng thái + bảng yêu cầu + Drawer chi tiết |

Ba nhóm màn không có trong `DESIGN.md` §14 nhưng PRD yêu cầu, được đặt sao cho KHÔNG làm phình điều hướng: `/join` (sinh viên, mở từ bộ chọn lớp hoặc từ "Hôm nay" khi chưa có lớp), `/class/members` + `/class/settings` (giảng viên, mở từ bộ chọn lớp → "Quản lý lớp này"), `/admin/courses` + `/admin/users` (thêm vào điều hướng của ADMIN, cùng nhóm với Observability / Cấu hình LLM / Tích hợp). Sau lượt rà soát end-to-end có thêm: `/assignments/[id]` cho sinh viên (vào từ Hôm nay, Lịch, Kết quả của tôi), tab **Bài tập** trong `/grading` cho giảng viên, các màn tài khoản (`/verify-email`, `/forgot-password`, `/reset-password`, `/invite/[token]`), và ở phase PR: `/settings/privacy`, `/help`, `/admin/audit`, `/admin/health`. Không màn nào thêm mục vào sidebar của sinh viên hay giảng viên. Tất cả dùng primitive sẵn có: bảng + Drawer, một hành động chính mỗi vùng, mã tham gia hiển thị bằng cỡ "Data emphasis" vì ở đó con số chính là việc cần làm.

## 3. Trình tự của `AGENT_PROMPT.md` khớp vào phase nào

| Bước trong `AGENT_PROMPT.md` | Phase thi công |
| --- | --- |
| Phase 1 foundation (token, font, chuẩn hoá, app shell, điều hướng theo vai trò) | **PU** L1–L2 |
| Phase 2 primitives với đủ 8 trạng thái | **PU** L3 |
| Phase 3 core routes | `/chat`, `/threads` dựng lại ở **PU** L4 (đã có backend). Các route còn lại dựng khi backend của chúng ra đời: `/` ở P2, `/inbox` P4, `/attendance` P5, `/gradebook` P6, `/grading/[id]` P7 |
| Phase 4 remaining routes | Rải theo P1–P10, luôn dùng primitive của PU |
| Phase 5 responsive + accessibility | Là cổng của TỪNG phase (xem `UX.md` mục 6), không dồn về cuối |
| Phase 6 motion + Red Thread Transition | P10 L4 (việc đầu tiên bị cắt nếu trễ) |
| Visual QA: đúng hai lượt A và B | P10 L4 |

Thành phần miền (domain component) được dựng ở phase có dữ liệu thật của nó:

| Component | Phase |
| --- | --- |
| `CitationList`, `VerificationState`, `Composer` | PU |
| `LLMRouteTable` | P1 |
| `PIIProtectionNotice` | P3 |
| `EscalationRow` | P4 |
| `AttendanceGrid`, `StudentRiskSummary` | P5 |
| `GradeCalculation`, `GradeSchemeReview` | P6 |
| `SubmissionReview` | P7 |
| `QuestionReview` | P9 |
| `KnowledgeGapTopic`, `RequestTraceDetail` | P10 |

## 4. Thứ `DESIGN.md` đòi mà backend phải cung cấp

`/` "Hôm nay" cần dữ liệu xếp hạng sẵn. Thiết kế: **luật cứng, không dùng LLM**, mỗi module đăng ký một "nhà cung cấp việc" (`internal/today`), gateway gộp và xếp hạng, cache 60 s theo người dùng, vô hiệu theo sự kiện.

| Vai trò | Nguồn việc (theo thứ tự ưu tiên) |
| --- | --- |
| STUDENT — một hành động khuyến nghị + lý do + thời lượng ước tính | Hạn nộp < 24 h chưa nộp → bài QUIZ / thi trong 48 h → chủ đề sai nhiều nhất khi luyện đề → câu trả lời của giảng viên chưa đọc → tài liệu mới của tuần |
| TEACHER / TA — danh sách việc cần quyết định | Ticket OPEN theo tuổi → bài chấm "cần xem kỹ" → câu trả lời AI chờ xác nhận → bài nộp chưa khớp → công thức điểm chưa xác nhận → câu hỏi chờ duyệt → sinh viên mới vào diện cần chú ý |

Lý do hiển thị phải là câu tiếng Việt sinh từ dữ liệu ("bạn sai 4/7 câu gần nhất"), không phải văn của LLM.

## 5. Kiểm phản mẫu bằng máy (chạy trong `/gate`)

`DESIGN.md` §21 liệt kê phản mẫu. Những cái kiểm được bằng `grep` / lint trên `frontend/src`:

- Màu, bo góc, bóng viết cứng ngoài `shared/styles/` (hex, `rgb(`, `oklch(`, `rounded-2xl`, `rounded-3xl`, `shadow-` ngoài Popover / Dialog / Menu).
- Lớp màu xám chung chung của Tailwind (`gray-`, `slate-`, `zinc-`, `neutral-`) thay cho token `--ep-*`.
- Cỡ chữ ngoài thang vai trò (`text-[…]` tuỳ ý).
- Hơn một nút `variant="primary"` trong cùng một component vùng làm việc.
- `<Dialog>` bọc một form sửa thông thường (chỉ cho phép trong danh sách trắng: xác nhận huỷ, PII, chốt điểm, công bố điểm, xác nhận công thức).
- Từ kỹ thuật lọt vào chuỗi hiển thị của màn sinh viên: `RAG`, `PII`, `fallback`, `trace`, `provider`, `redaction`, `confidence`.
- Spinner toàn trang; emoji dùng làm icon chức năng.

Những cái phải nhìn bằng mắt (làm trong "Bạn tự kiểm"): tường thẻ KPI, card lồng card, diện tích đỏ > 8%, khung nhìn đầu có một hành động chính rõ ràng trong ≈ 3 giây.

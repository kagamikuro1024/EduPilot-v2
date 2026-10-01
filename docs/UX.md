# EduPilot v2 — Nguyên tắc UX (độ bền, hiệu năng, hành vi)

Một hệ thống scale tốt mà dùng bực mình thì vẫn thất bại. Quy tắc ở đây là **điều kiện nghiệm thu**, ngang hàng với test.

**Phân vai với bộ UI kit:** `design/DESIGN.md` (hướng *Red Thread / Academic Instrument*) quyết định **trông thế nào và bố cục ra sao**: token, chữ, màu, component, khung nhìn đầu của từng route, lời văn, phản mẫu. File này quyết định **cư xử thế nào khi đời thực xảy ra**: mạng xấu, bấm đúp, tải lại trang, dữ liệu lớn, điện thoại một tay. Chỗ hai bên từng lệch nhau đã được chốt ở `design/INTEGRATION.md` mục 2. Không bên nào là cớ để thêm tính năng.

## 1. Ba người dùng, ba bối cảnh

| Ai | Bối cảnh thật | Hệ quả thiết kế |
| --- | --- | --- |
| Sinh viên | Điện thoại hoặc laptop, thường vào lúc gấp (đêm trước hạn), mạng ký túc xá chập chờn | Mobile-first cho Hôm nay, Chat, Threads, Luyện đề, Lịch, `/me`. Không bao giờ mất thứ đã gõ. Câu trả lời phải bắt đầu hiện nhanh |
| Giảng viên trong lớp | **Đứng trên bục, cầm điện thoại**, 60 giây để điểm danh, vừa giảng vừa ghi phát biểu | `/attendance` dùng được bằng một tay trên điện thoại; vùng chạm ≥ 44 px; không hộp thoại xác nhận; hoàn tác thay cho xác nhận |
| Giảng viên ở bàn làm việc | Laptop, làm theo lô: duyệt 30 bài chấm, xác nhận công thức, xem báo cáo | Bàn phím là chính: phím tắt, duyệt–kế tiếp không rời tay khỏi phím; bảng lớn cuộn mượt |

## 2. Mười quy tắc

1. **Phản hồi tức thì.** Mọi thao tác có phản hồi thị giác ≤ 100 ms. Thao tác đảo ngược được (điểm danh, điểm cộng, like, ghim, đánh dấu đã đọc) dùng **cập nhật lạc quan**: UI đổi ngay, lỗi thì hoàn lại và báo.
2. **Hoàn tác thay cho xác nhận.** Hành động đảo ngược được → dòng xác nhận tĩnh lặng ngay tại nơi thao tác "Đã … · Hoàn tác", tự biến sau 5 giây (không dùng toast "Thành công!" — `DESIGN.md` §13). Hộp thoại xác nhận CHỈ dành cho việc không đảo ngược: công bố điểm, chốt điểm cuối kỳ, xác nhận công thức, xoá tài liệu. Hộp thoại đó nói rõ hậu quả bằng con số ("Công bố điểm cho 28 sinh viên, gửi 28 mail").
3. **Không bao giờ mất chữ đã gõ.** Tự lưu nháp vào máy (mỗi 2 s khi có thay đổi) cho: bài Threads, tin nhắn chat, câu trả lời ticket, nhận xét chấm bài, observation, bài thi thử, ô điền công thức. Khôi phục khi quay lại, kể cả sau khi sập trình duyệt.
4. **Chờ phải có nghĩa.** Khung xương (skeleton) cho tải trang; chỉ báo giai đoạn cho việc dài ("Đang tìm trong tài liệu…", "Đang chấm 12/85"); việc nền hiện tiến độ qua SSE và cho phép rời trang. Cấm spinner trơ quá 1 giây không lời giải thích.
5. **Không có ngõ cụt.** Mỗi trạng thái rỗng nói *vì sao rỗng* và có *một nút hành động kế tiếp*. Mỗi lỗi nói *chuyện gì xảy ra*, *dữ liệu của bạn có an toàn không*, và có nút *Thử lại*. Không bao giờ hiện mã lỗi trần hay "Something went wrong".
6. **Chịu được mạng xấu.** Stream chat đứt thì tự nối lại và hiện tiếp phần đã sinh; tải lại trang không mất câu trả lời. Gửi thất bại thì giữ nguyên nội dung và cho gửi lại (server khử trùng bằng `Idempotency-Key`). Mất mạng có dải báo; điểm danh xếp hàng cục bộ và tự đồng bộ khi có mạng lại.
7. **AI phải minh bạch.** Mọi nội dung AI có nhãn nguồn gốc nhất quán: `AI` · `Đã được giảng viên xác nhận` · `Đã sửa bởi giảng viên`; citation mở tại chỗ và dẫn đúng trang tài liệu; sinh viên thấy độ chắc chắn bằng LỜI ("AI chưa đủ chắc chắn về câu này"), con số chỉ dành cho giảng viên; dòng "Đã ẩn N thông tin cá nhân trước khi gửi cho AI"; điểm AI chấm luôn ghi "bản nháp" cho đến khi công bố. Khi AI không chắc, UI nói thẳng và đưa lối sang giảng viên.
8. **Bàn phím và trợ năng.** Mọi chức năng dùng được bằng bàn phím; thứ tự focus hợp lý; vòng focus nhìn thấy; tương phản đạt WCAG AA; nhãn cho trình đọc màn hình; không truyền nghĩa chỉ bằng màu (trạng thái điểm danh có cả chữ/biểu tượng). `Ctrl/⌘ K` mở bảng lệnh tìm nhanh sinh viên, tài liệu, màn hình.
9. **Thông báo có chừng mực.** Gộp thông báo cùng loại ("5 câu trả lời AI chờ duyệt"); mail chỉ cho việc cần hành động; nhắc lịch không gửi trùng; mỗi loại tắt được trong cài đặt.
10. **Tiếng Việt tử tế.** Văn phong ngắn, xưng hô nhất quán ("bạn" với sinh viên, "thầy/cô" với giảng viên); ngày giờ kiểu Việt (`Thứ Hai, 21/09 · 14:00`), múi giờ `Asia/Ho_Chi_Minh`; số thập phân dùng dấu phẩy khi hiển thị điểm; font hiển thị đủ dấu.

## 3. Ngân sách hiệu năng phía người dùng

| Chỉ số | Ngân sách | Ghi chú |
| --- | --- | --- |
| LCP trang chính trên 4G | ≤ 2,5 s | Đo bằng Lighthouse CI, cấu hình mobile |
| INP | ≤ 200 ms | |
| CLS | ≤ 0,1 | Skeleton giữ đúng kích thước |
| JS mỗi route (gzip) | ≤ 250 KB | Tách mã theo route; biểu đồ và trình xem PDF nạp lười |
| Phản hồi đầu tiên của chat | Hiện trạng thái "đang xử lý" ≤ 300 ms; token đầu theo SLO | |
| Bảng 1.000 dòng | Cuộn 60 fps | Ảo hoá danh sách; phân trang con trỏ phía server |
| Điểm danh 30 SV trên điện thoại | ≤ 60 s, 0 hộp thoại | Kiểm tay ở P5 |

## 4. Nền tảng frontend dùng chung (dựng một lần ở phase PU, mọi phase sau dùng lại)

Hình thức của mọi thành phần dưới đây theo `design/DESIGN.md` §10; bảng này chỉ nêu phần **hành vi** mà `DESIGN.md` không nói tới.

| Thành phần | Trách nhiệm |
| --- | --- |
| Lớp dữ liệu: **TanStack Query** | Cache, khử trùng request, tải lại nền, cập nhật lạc quan + hoàn lại, phân trang vô hạn theo con trỏ. zustand chỉ còn giữ trạng thái UI và phiên đăng nhập |
| `apiClient` | Gắn JWT, `Idempotency-Key` tự sinh cho POST, đọc lỗi có cấu trúc, tự thử lại GET có backoff, không bao giờ tự thử lại POST thiếu khoá |
| `useSSE` | Kết nối, tự nối lại có backoff, `Last-Event-ID`, dọn dẹp khi rời trang |
| `useAutosaveDraft(key)` | Quy tắc 3 |
| `useUndoableAction` + dòng xác nhận tại chỗ | Quy tắc 1, 2 |
| `DataTable` (primitive của `DESIGN.md` §10.4) | Thêm: ảo hoá, phân trang con trỏ, điều hướng bàn phím, giữ vị trí cuộn khi mở / đóng chi tiết |
| `<PageState>` bọc `Skeleton` / `EmptyState` / lỗi của `DESIGN.md` §15 | Quy tắc 4, 5; không bao giờ chặn cả màn vì một vùng đang tải |
| `VerificationState`, `CitationList`, `PIIProtectionNotice` (thành phần miền của `DESIGN.md` §19) | Quy tắc 7 |
| `<ConfirmIrreversible>` dựng trên `Dialog` | Quy tắc 2, bắt buộc truyền mô tả hậu quả bằng con số; chỉ dùng cho danh sách việc cần bảo vệ ở `DESIGN.md` §10.11 |
| `<CommandPalette>`, bản đồ phím tắt | Quy tắc 8 |
| `<OfflineBanner>` + hàng đợi ghi cục bộ (chỉ dùng cho điểm danh) | Quy tắc 6 |
| Token `--ep-*` từ `shared/styles/tokens.css` (nguồn: `design/DESIGN_TOKENS.css`); chỉ giao diện sáng; định dạng ngày–số tiếng Việt | Nhất quán |

Cấm viết `fetch` trần, spinner riêng, hộp thoại xác nhận riêng, hay bảng riêng trong các module. Thiếu gì thì bổ sung vào nền tảng chung.

## 5. Chi tiết theo màn hình trọng yếu

| Màn | Điều phải đúng |
| --- | --- |
| `/chat` | Ô nhập luôn sẵn sàng (gõ tiếp được khi AI đang trả lời); nút Dừng; tự cuộn nhưng dừng cuộn khi người dùng kéo lên; câu trả lời dở dang vẫn giữ sau khi tải lại; khi quá tải hiện thời gian chờ ước tính + "Nhờ giảng viên" |
| `/threads` | Kiểm tra PII chạy **khi đang gõ** (debounce 800 ms) và báo bằng dòng `PIIProtectionNotice` trên composer; bấm Đăng mà còn nội dung cá nhân → Dialog đúng hai lối; chuyển sang chat riêng giữ nguyên chữ |
| `/attendance` | Mặc định tất cả "Có mặt", chỉ chạm vào người vắng; điện thoại là **hàng** ngăn bằng đường kẻ (không card), máy tính là lưới + phím 1–4; mỗi thay đổi tự lưu lạc quan và hiện trạng thái lưu; nút chính `Lưu điểm danh` = hoàn tất buổi; "+ phát biểu" ngay trên dòng; hoạt động được khi mất mạng |
| `/inbox` | Danh sách trái, hội thoại phải; `J/K` chuyển ticket, `R` trả lời, `⌘↵` gửi và sang ticket kế; nháp tự lưu theo ticket |
| `/grading/[id]` | Hai cột đồng bộ: bấm vào bằng chứng thì cuộn tới đoạn trích trong bài làm và tô sáng; sửa điểm tại chỗ; `A` duyệt và sang bài kế; thanh tiến độ "12/85 đã duyệt"; lọc "cần xem kỹ" lên đầu |
| `/gradebook/scheme` | Từng mục công thức nằm cạnh đoạn trích quy chế (mở đúng trang PDF); mục "chưa rõ" tô nổi và chặn nút Xác nhận kèm lý do; xem trước tác động lên 3 sinh viên mẫu khi đổi số |
| `/gradebook` | Ô sửa trực tiếp kiểu bảng tính (Enter xuống, Tab sang); xung đột phiên bản (409) hiện giá trị của người kia và hỏi giữ bên nào; cột cố định tên sinh viên |
| `/practice/[id]` | Đồng hồ không nhảy layout; tự lưu mỗi câu; mất mạng vẫn làm tiếp, nộp khi có mạng; cảnh báo 5 phút cuối; tải lại trang quay đúng câu đang làm |
| `/insights` | Tạo báo cáo là việc nền: hiện tiến độ, rời trang được, xong thì chuông báo; bản cũ vẫn xem được trong lúc chờ |
| Banner công thức điểm | Một dòng, có nút hành động trực tiếp ("Tải quy chế lên" / "Xác nhận công thức"), không che nội dung, không tắt vĩnh viễn được khi chưa xử lý |

## 6. Cổng UX cho mỗi phase (thêm vào Definition of Done)

- [ ] Đạt `design/DESIGN.md` §22 (10 điều kiện) và không dính phản mẫu nào ở §21; `scripts/ui-antipatterns.sh` sạch
- [ ] Người dùng đúng vai trò nhận ra việc chính của màn trong ≈ 3 giây; khung nhìn đầu có MỘT lối hành động rõ ràng
- [ ] Mọi màn mới dùng `<PageState>`: có đủ tải / rỗng / lỗi, và trạng thái rỗng có nút hành động
- [ ] Không có `fetch` trần, spinner riêng, confirm riêng (`grep` kiểm được)
- [ ] Chạy được bằng bàn phím từ đầu đến cuối luồng chính; axe không báo lỗi nghiêm trọng
- [ ] Đúng bốn mốc bề rộng của `DESIGN.md` §17 (< 720, 720–1099, ≥ 1100, ≥ 1440); zoom 200% vẫn thao tác được; chữ Việt dài không vỡ bố cục, dấu không bị cắt
- [ ] Xem ở bề rộng 375 px: không tràn ngang, vùng chạm ≥ 44 px (bắt buộc với màn của sinh viên và `/attendance`, `/inbox`)
- [ ] Thử "mạng chậm 3G" + ngắt mạng giữa thao tác chính: không mất dữ liệu đã nhập, có thông báo dễ hiểu
- [ ] Lighthouse mobile của route mới đạt ngân sách mục 3
- [ ] Văn bản hiển thị là tiếng Việt, đã đọc lại thành tiếng một lượt

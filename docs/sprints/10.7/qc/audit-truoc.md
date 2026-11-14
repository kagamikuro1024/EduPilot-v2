# QC 10.7 — kiểm chéo khung trang (hiện trạng, trước UI-R)

Ảnh + số đo DOM: **250** lượt chụp (66 cặp vai × route × 2 độ rộng × có dữ liệu / rỗng); lỗi chụp 0. Nguồn: `scripts/audit_report.py` đọc `audit/audit-{data,empty}.jsonl` do `frontend/e2e/qc-ui-audit-real.spec.ts` (Playwright, chỉ đọc, `ep-qc`). Lưới ảnh: `audit.html`.

- **Có dữ liệu** = lớp 761987 (seed mặc định); **rỗng** = lớp QCEMPTY dựng riêng (0 dữ liệu; teacher + ta + sv31). Admin không gắn lớp → một trạng thái. Không thao tác KC106.
- **Số đo từ DOM** (`getBoundingClientRect`, `data-ep-panel`, `data-variant`, class module `page/reading/wide/full`, `meta`, `desc`, `sectionDesc`, `empty`); ảnh viewport (JPEG), không cuộn.
- `?state=empty` (FR-X5) chỉ còn ở màn demo (`DemoDocuments`), các route thật không đọc tham số → trạng thái rỗng lấy bằng **lớp rỗng thật**.
- Route động chỉ chụp ở trạng thái có dữ liệu.

## Kết quả chính (số đo DOM, hiện trạng)

- **Độ rộng:** 3 kiểu `Page` (`wide` 26, `reading` 25, `full` 15 trang) và **6** độ rộng panel khác nhau (hợp đồng: 2 kiểu, panel theo kiểu).
- **K2a** nút đầu trang nằm dưới tiêu đề: **3** mục
- **K2b** meta > 1 dòng: **2** mục
- **K2c** meta lặp mã / tên lớp: **9** mục
- **K2d** dòng mô tả dưới tiêu đề trang: **12** mục
- **K2f** > 1 nút primary trên trang: **1** mục
- **K3** tiêu đề mục có dòng phụ: **2** mục
- **K4a** khối rỗng có thân chữ: **15** mục
- **K4b** khối rỗng không căn giữa (> 40 px): **28** mục
- **K4c** panel kéo cao ≥ 0,8 viewport khi rỗng: **0** mục
- **K4d** > 1 nút trong khối rỗng: **1** mục
- **K4e** khối rỗng có icon: **8** mục
- **K4f** nút rỗng trùng nút đầu trang: **12** mục
- **K6** tràn ngang ở 375 px: **0** mục
- **Kiểu khối rỗng:** 8 kiểu khác nhau (hợp đồng: 1).

## K1. Kiểu độ rộng và panel (1440, có dữ liệu)

Hợp đồng 10.7: đúng 2 kiểu (`work` hết vùng nội dung, `read` cột đọc cố định căn giữa). Hiện primitive có **3** kiểu (`reading` 960+48, `wide` ≤ 1440, `full`).

| Kiểu `Page` | Số trang | Độ rộng page (px) |
| --- | --- | --- |
| `wide` | 26 | 1224 |
| `reading` | 25 | 1008 |
| `full` | 15 | 1224 |

Panel cấp cao nhất: **6** độ rộng khác nhau trên 66 trang có panel. Phân bố (px → số panel): 320→4; 505→2; 647→2; 840→1; 960→58; 1176→53

| Vai | Route | `Page` | page px | số panel | panel px (tối đa 6) | lề trái → panel | lề phải ← panel | panel cao / viewport |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| student | `/` | reading | 1008 | 2 | 960, 960 | 24 | 240 | 0.14 |
| student | `/assignments/01a1d9df-dbb9-77b2-ab9b-73cced18a0c0` | reading | 1008 | 1 | 960 | 24 | 240 | 1.38 |
| student | `/calendar` | wide | 1224 | 1 | 1176 | 24 | 24 | 0.48 |
| student | `/chat` | full | 1224 | 1 | 840 | 324 | 60 | 0.78 |
| student | `/exams` | reading | 1008 | 1 | 960 | 24 | 240 | 0.4 |
| student | `/library` | full | 1224 | 1 | 1176 | 24 | 24 | 0.58 |
| student | `/library/01a1d9dd-35df-7b1a-8352-d45930902823` | full | 1224 | 1 | 1176 | 24 | 24 | 0.86 |
| student | `/me` | reading | 1008 | 5 | 960, 960, 960, 960, 960 | 24 | 240 | 0.55 |
| student | `/practice` | reading | 1008 | 3 | 960, 960, 960 | 24 | 240 | 0.14 |
| student | `/practice/history` | reading | 1008 | 2 | 960, 960 | 24 | 240 | 0.2 |
| student | `/settings` | reading | 1008 | 5 | 960, 960, 960, 960, 960 | 24 | 240 | 0.4 |
| student | `/threads` | full | 1224 | 1 | 1176 | 24 | 24 | 1.66 |
| student | `/threads/01a1d9df-6c17-7b13-b71b-d5e4ad4d4540` | full | 1224 | 2 | 1176, 1176 | 24 | 24 | 0.35 |
| ta | `/` | reading | 1008 | 2 | 320, 320 | 664 | 240 | 0.21 |
| ta | `/analytics` | reading | 1008 | 4 | 960, 960, 960, 960 | 24 | 240 | 0.63 |
| ta | `/attendance` | wide | 1224 | 1 | 1176 | 24 | 24 | 0.81 |
| ta | `/calendar` | wide | 1224 | 1 | 1176 | 24 | 24 | 0.41 |
| ta | `/class/members` | wide | 1224 | 1 | 1176 | 24 | 24 | 2.07 |
| ta | `/class/settings` | reading | 1008 | 5 | 960, 960, 960, 960, 960 | 24 | 240 | 0.24 |
| ta | `/documents` | full | 1224 | 1 | 1176 | 24 | 24 | 0.93 |
| ta | `/exams` | wide | 1224 | 1 | 1176 | 24 | 24 | 0.4 |
| ta | `/exams/01a1d9e1-2fb4-75c4-9318-fbc4a4d6c8ed` | wide | 1224 | 1 | 1176 | 24 | 24 | 1.29 |
| ta | `/gradebook` | wide | 1224 | 1 | 1176 | 24 | 24 | 0.8 |
| ta | `/gradebook/scheme` | reading | 1008 | 1 | 960 | 24 | 240 | 1.18 |
| ta | `/grading` | wide | 1224 | 1 | 1176 | 24 | 24 | 0.78 |
| ta | `/grading/01a1d9df-dcb3-7b53-9c26-1ec49dca800e` | full | 1224 | 2 | 647, 505 | 24 | 553 | 0.1 |
| ta | `/inbox` | full | 1224 | 1 | 1176 | 24 | 24 | 0.75 |
| ta | `/insights` | reading | 1008 | 2 | 960, 960 | 24 | 240 | 2.03 |
| ta | `/questions` | wide | 1224 | 1 | 1176 | 24 | 24 | 2.32 |
| ta | `/settings` | reading | 1008 | 3 | 960, 960, 960 | 24 | 240 | 0.4 |
| ta | `/students` | wide | 1224 | 1 | 1176 | 24 | 24 | 0.72 |
| ta | `/students/01a1d9dc-1afd-779f-874c-aa099aecd39f` | reading | 1008 | 1 | 960 | 24 | 240 | 0.33 |
| ta | `/threads` | full | 1224 | 1 | 1176 | 24 | 24 | 1.8 |
| ta | `/threads/01a1d9df-6c17-7b13-b71b-d5e4ad4d4540` | full | 1224 | 2 | 1176, 1176 | 24 | 24 | 0.35 |
| teacher | `/` | reading | 1008 | 2 | 320, 320 | 664 | 240 | 0.21 |
| teacher | `/analytics` | reading | 1008 | 5 | 960, 960, 960, 960, 960 | 24 | 240 | 0.63 |
| teacher | `/attendance` | wide | 1224 | 1 | 1176 | 24 | 24 | 0.81 |
| teacher | `/calendar` | wide | 1224 | 1 | 1176 | 24 | 24 | 0.41 |
| teacher | `/class/members` | wide | 1224 | 1 | 1176 | 24 | 24 | 2.07 |
| teacher | `/class/settings` | reading | 1008 | 7 | 960, 960, 960, 960, 960, 960 | 24 | 240 | 0.24 |
| teacher | `/documents` | full | 1224 | 1 | 1176 | 24 | 24 | 0.93 |
| teacher | `/exams` | wide | 1224 | 1 | 1176 | 24 | 24 | 0.4 |
| teacher | `/exams/01a1d9e1-2fb4-75c4-9318-fbc4a4d6c8ed` | wide | 1224 | 1 | 1176 | 24 | 24 | 1.29 |
| teacher | `/exams/01a1d9e1-2fb4-75c4-9318-fbc4a4d6c8ed/results` | wide | 1224 | 1 | 1176 | 24 | 24 | 0.75 |
| teacher | `/gradebook` | wide | 1224 | 1 | 1176 | 24 | 24 | 0.8 |
| teacher | `/gradebook/scheme` | reading | 1008 | 1 | 960 | 24 | 240 | 1.18 |
| teacher | `/grading` | wide | 1224 | 1 | 1176 | 24 | 24 | 0.78 |
| teacher | `/grading/01a1d9df-dcb3-7b53-9c26-1ec49dca800e` | full | 1224 | 2 | 647, 505 | 24 | 553 | 0.1 |
| teacher | `/inbox` | full | 1224 | 1 | 1176 | 24 | 24 | 0.75 |
| teacher | `/insights` | reading | 1008 | 2 | 960, 960 | 24 | 240 | 2.27 |
| teacher | `/observability` | wide | 1224 | 2 | 1176, 1176 | 24 | 24 | 0.35 |
| teacher | `/questions` | wide | 1224 | 1 | 1176 | 24 | 24 | 2.32 |
| teacher | `/settings` | reading | 1008 | 3 | 960, 960, 960 | 24 | 240 | 0.4 |
| teacher | `/settings/integrations` | reading | 1008 | 1 | 960 | 24 | 240 | 1.11 |
| teacher | `/settings/llm` | wide | 1224 | 5 | 1176, 1176, 1176, 1176, 1176 | 24 | 24 | 0.15 |
| teacher | `/students` | wide | 1224 | 1 | 1176 | 24 | 24 | 0.72 |
| teacher | `/students/01a1d9dc-1afd-779f-874c-aa099aecd39f` | reading | 1008 | 1 | 960 | 24 | 240 | 0.33 |
| teacher | `/threads` | full | 1224 | 1 | 1176 | 24 | 24 | 1.8 |
| teacher | `/threads/01a1d9df-6c17-7b13-b71b-d5e4ad4d4540` | full | 1224 | 2 | 1176, 1176 | 24 | 24 | 0.35 |
| admin | `/` | reading | 1008 | 1 | 960 | 24 | 240 | 0.23 |
| admin | `/admin/courses` | wide | 1224 | 1 | 1176 | 24 | 24 | 0.36 |
| admin | `/admin/users` | wide | 1224 | 1 | 1176 | 24 | 24 | 1.81 |
| admin | `/observability` | wide | 1224 | 3 | 1176, 1176, 1176 | 24 | 24 | 0.17 |
| admin | `/settings` | reading | 1008 | 2 | 960, 960 | 24 | 240 | 0.4 |
| admin | `/settings/integrations` | reading | 1008 | 1 | 960 | 24 | 240 | 1.23 |
| admin | `/settings/llm` | wide | 1224 | 6 | 1176, 1176, 1176, 1176, 1176, 1176 | 24 | 24 | 0.2 |

## K2. Đầu trang (1440, có dữ liệu)

Hợp đồng: tiêu đề · ≤ 1 dòng meta (không lặp lớp) · hành động chính bên phải **cùng hàng** tiêu đề · ≤ 2 phụ.

| Vai | Route | nút đầu trang (kiểu) | vị trí nút | dòng meta | meta lặp lớp | dòng mô tả dưới tiêu đề | nút primary ngoài đầu trang |
| --- | --- | --- | --- | --- | --- | --- | --- |
| student | `/` | – | không có | 0 | – | 1 | Điểm bài thi Kiểm tra tuần 9 — Mật mã đã công bố |
| student | `/assignments/01a1d9df-dbb9-77b2-ab9b-73cced18a0c0` | – | không có | 1 | – | – | Nộp lại |
| student | `/calendar` | Thêm vào lịch (secondary) | cùng hàng | 0 | – | – | – |
| student | `/chat` | – | không có | 0 | – | – | Gửi |
| student | `/exams` | – | không có | 0 | – | – | – |
| student | `/library` | – | không có | 0 | – | – | – |
| student | `/library/01a1d9dd-35df-7b1a-8352-d45930902823` | Thư viện (ghost) | cùng hàng | 0 | – | – | Hỏi AI về tài liệu |
| student | `/me` | – | không có | 0 | – | – | – |
| student | `/practice` | Lịch sử luyện tập (secondary) | cùng hàng | 0 | – | – | Luyện 10 câu |
| student | `/practice/history` | – | không có | 0 | – | – | Luyện An toàn ứng dụng web |
| student | `/settings` | – | không có | 0 | – | – | Đổi mật khẩu |
| student | `/threads` | Đặt câu hỏi (primary) | cùng hàng | 0 | – | – | – |
| student | `/threads/01a1d9df-6c17-7b13-b71b-d5e4ad4d4540` | Tất cả câu hỏi (ghost) | cùng hàng | 0 | – | – | Đăng |
| ta | `/` | – | không có | 0 | – | 1 | Mở |
| ta | `/analytics` | 7 ngày (link); 30 ngày (link) | cùng hàng | 1 | **có** | – | – |
| ta | `/attendance` | Lưu điểm danh (primary) | cùng hàng | 0 | – | 1 | – |
| ta | `/calendar` | Thêm vào lịch (secondary); Thêm sự kiện (secondary); Tạo lịch buổi học (primary) | cùng hàng | 0 | – | – | – |
| ta | `/class/members` | Mã và cài đặt tham gia (secondary) | cùng hàng | 0 | – | – | – |
| ta | `/class/settings` | Thành viên lớp (secondary) | cùng hàng | 0 | – | – | – |
| ta | `/documents` | Tải tài liệu lên (primary) | cùng hàng | 0 | – | – | – |
| ta | `/exams` | Tạo bài thi (primary) | cùng hàng | 0 | – | – | – |
| ta | `/exams/01a1d9e1-2fb4-75c4-9318-fbc4a4d6c8ed` | Xem kết quả (primary); Thêm hành động cho bài thi (link) | dưới tiêu đề | 0 | – | 1 | – |
| ta | `/gradebook` | Xem công thức (secondary) | cùng hàng | 1 | – | – | – |
| ta | `/gradebook/scheme` | – | không có | 1 | – | – | – |
| ta | `/grading` | – | không có | 0 | – | – | Chấm tất cả |
| ta | `/grading/01a1d9df-dcb3-7b53-9c26-1ec49dca800e` | – | không có | 1 | – | – | Đã duyệt |
| ta | `/inbox` | – | không có | 1 | **có** | – | Nhận |
| ta | `/insights` | – | không có | 2 | **có** | – | – |
| ta | `/questions` | Tạo câu hỏi (primary) | cùng hàng | 1 | – | – | – |
| ta | `/settings` | – | không có | 0 | – | – | Đổi mật khẩu |
| ta | `/students` | – | không có | 0 | – | 1 | – |
| ta | `/students/01a1d9dc-1afd-779f-874c-aa099aecd39f` | Thêm ghi chú (secondary); Nhắn sinh viên (secondary) | cùng hàng | 0 | – | 1 | – |
| ta | `/threads` | Đặt câu hỏi (primary) | cùng hàng | 0 | – | – | – |
| ta | `/threads/01a1d9df-6c17-7b13-b71b-d5e4ad4d4540` | Tất cả câu hỏi (ghost) | cùng hàng | 0 | – | – | Đăng |
| teacher | `/` | – | không có | 0 | – | 1 | Mở |
| teacher | `/analytics` | 7 ngày (link); 30 ngày (link) | cùng hàng | 1 | **có** | – | – |
| teacher | `/attendance` | Lưu điểm danh (primary) | cùng hàng | 0 | – | 1 | – |
| teacher | `/calendar` | Thêm vào lịch (secondary); Thêm sự kiện (secondary); Tạo lịch buổi học (primary) | cùng hàng | 0 | – | – | – |
| teacher | `/class/members` | Mã và cài đặt tham gia (secondary) | cùng hàng | 0 | – | – | – |
| teacher | `/class/settings` | Thành viên lớp (secondary) | cùng hàng | 0 | – | – | Lưu cài đặt |
| teacher | `/documents` | Tải tài liệu lên (primary) | cùng hàng | 0 | – | – | – |
| teacher | `/exams` | Tạo bài thi (primary) | cùng hàng | 0 | – | – | – |
| teacher | `/exams/01a1d9e1-2fb4-75c4-9318-fbc4a4d6c8ed` | Xem kết quả (primary); Thêm hành động cho bài thi (link) | dưới tiêu đề | 0 | – | 1 | – |
| teacher | `/exams/01a1d9e1-2fb4-75c4-9318-fbc4a4d6c8ed/results` | – | không có | 0 | – | – | – |
| teacher | `/gradebook` | Xem công thức (secondary); Thêm hành động (link); Chốt điểm (primary) | cùng hàng | 1 | – | – | – |
| teacher | `/gradebook/scheme` | Tải quy chế (secondary) | cùng hàng | 1 | – | – | – |
| teacher | `/grading` | – | không có | 0 | – | – | Chấm tất cả; Công bố |
| teacher | `/grading/01a1d9df-dcb3-7b53-9c26-1ec49dca800e` | – | không có | 1 | – | – | Đã duyệt |
| teacher | `/inbox` | – | không có | 1 | **có** | – | Nhận |
| teacher | `/insights` | Tạo báo cáo mới (primary) | dưới tiêu đề | 2 | **có** | – | – |
| teacher | `/observability` | – | không có | 1 | **có** | – | – |
| teacher | `/questions` | Tạo câu hỏi (primary) | cùng hàng | 1 | – | – | – |
| teacher | `/settings` | – | không có | 0 | – | – | Đổi mật khẩu |
| teacher | `/settings/integrations` | – | không có | 0 | – | – | – |
| teacher | `/settings/llm` | – | không có | 0 | – | – | – |
| teacher | `/students` | – | không có | 0 | – | 1 | – |
| teacher | `/students/01a1d9dc-1afd-779f-874c-aa099aecd39f` | Thêm ghi chú (secondary); Nhắn sinh viên (secondary) | cùng hàng | 0 | – | 1 | – |
| teacher | `/threads` | Đặt câu hỏi (primary) | cùng hàng | 0 | – | – | – |
| teacher | `/threads/01a1d9df-6c17-7b13-b71b-d5e4ad4d4540` | Tất cả câu hỏi (ghost) | cùng hàng | 0 | – | – | Đăng |
| admin | `/` | – | không có | 0 | – | 1 | – |
| admin | `/admin/courses` | Mở lớp (primary) | cùng hàng | 1 | – | – | – |
| admin | `/admin/users` | Mời giảng viên (primary) | cùng hàng | 0 | – | – | – |
| admin | `/observability` | – | không có | 1 | – | – | – |
| admin | `/settings` | – | không có | 0 | – | – | Đổi mật khẩu |
| admin | `/settings/integrations` | – | cùng hàng | 0 | – | – | – |
| admin | `/settings/llm` | – | không có | 0 | – | – | – |

## K3. Tiêu đề mục có dòng phụ

| Vai | Route | dòng phụ dưới tiêu đề mục |
| --- | --- | --- |
| ta | `/insights` | Câu hỏi trong 7 ngày liền trước đó, đã ẩn danh |
| teacher | `/insights` | Câu hỏi trong 7 ngày liền trước đó, đã ẩn danh |

## K4. Trạng thái rỗng (lớp QCEMPTY, 1440)

Hợp đồng: một mẫu duy nhất — tiêu đề ngắn + ≤ 1 nút, **căn giữa** vùng, **không thân chữ**, không nút trùng đầu trang, panel không kéo hết màn.

| Vai | Route | khối rỗng | icon | tiêu đề | thân chữ | nút (kiểu) | lệch tâm (px) | panel cha cao / viewport | nút trùng đầu trang |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| student | `/` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| student | `/assignments/01a1d9df-dbb9-77b2-ab9b-73cced18a0c0` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| student | `/calendar` | tiêu đề | – | Chưa có gì trong lịch. | – | – | -279 | 0.17 | – |
| student | `/chat` | tiêu đề+3 nút | – | Hỏi bất cứ điều gì về lớp này. | – | Điểm quá trình của tôi hiện là bao nhiêu? (link); Tuần này tôi có những buổi học nào? (link); Gửi (primary) | 0 | 0.78 | – |
| student | `/exams` | tiêu đề+thân | – | Chưa có bài thi | Lớp của bạn chưa có bài thi nào. | – | -171 | 0.2 | – |
| student | `/library` | tiêu đề | – | Chưa có tài liệu nào. | – | – | -279 | 0.17 | – |
| student | `/library/01a1d9dd-35df-7b1a-8352-d45930902823` | tiêu đề+1 nút | – | Không tìm thấy tài liệu. | – | Về thư viện (primary) | -279 | 0.23 | – |
| student | `/me` | tiêu đề | – | Chưa có buổi nào được điểm danh. | – | – | -171 | 0.17 | – |
| student | `/me` | tiêu đề | – | Chưa có điểm cộng nào được ghi. | – | – | -171 | 0.17 | – |
| student | `/me` | tiêu đề | – | Lớp chưa có bài tập. | – | – | -171 | 0.17 | – |
| student | `/practice` | tiêu đề | – | Giảng viên chưa mở câu hỏi | – | – | -171 | 0.17 | – |
| student | `/practice/history` | tiêu đề+thân+1 nút | – | Chưa có lượt luyện nào | Sau lượt đầu tiên, bạn sẽ thấy ở đây mình hay sai chủ đề nào. | Luyện đề (primary) | -171 | 0.27 | – |
| student | `/settings` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| student | `/threads` | tiêu đề+thân+1 nút | – | Chưa có câu hỏi nào. Đặt câu hỏi đầu tiên. | Đặt câu hỏi | Đặt câu hỏi (primary) | -279 | 0.22 | **Đặt câu hỏi** |
| student | `/threads/01a1d9df-6c17-7b13-b71b-d5e4ad4d4540` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| ta | `/` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| ta | `/analytics` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| ta | `/attendance` | tiêu đề+1 nút | – | Lớp chưa có lịch buổi học. | – | Tạo lịch buổi học (primary) | -279 | 0.23 | – |
| ta | `/calendar` | tiêu đề+1 nút | – | Chưa có gì trong lịch. | – | Thêm sự kiện (primary) | -279 | 0.23 | **Thêm sự kiện** |
| ta | `/class/members` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| ta | `/class/settings` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| ta | `/documents` | tiêu đề+thân+1 nút | – | Lớp chưa có tài liệu. | Tải tài liệu lên | Tải tài liệu lên (primary) | -279 | 0.5 | **Tải tài liệu lên** |
| ta | `/exams` | icon+tiêu đề+thân+1 nút | có | Chưa có bài thi nào | Chưa có bài thi nào. Tạo bài thi đầu tiên từ ngân hàng câu hỏi. | Tạo bài thi (primary) | -279 | 0.31 | **Tạo bài thi** |
| ta | `/exams/01a1d9e1-2fb4-75c4-9318-fbc4a4d6c8ed` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| ta | `/gradebook` | tiêu đề | – | Chưa có công thức điểm. | – | – | -279 | 0.22 | – |
| ta | `/gradebook/scheme` | icon+tiêu đề+1 nút | có | Lớp chưa có quy chế. | – | Về sổ điểm (secondary) | -171 | 0.27 | – |
| ta | `/grading` | icon+tiêu đề+thân | có | Chưa có bài nộp. | Sinh viên nộp trong ứng dụng, hoặc thầy/cô tải ZIP lên. | – | -279 | 0.32 | – |
| ta | `/grading/01a1d9df-dcb3-7b53-9c26-1ec49dca800e` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| ta | `/inbox` | tiêu đề+thân+1 nút | – | Chưa có câu hỏi nào cần thầy/cô xử lý. | Khi AI không đủ chắc chắn hoặc sinh viên nhờ hỗ trợ, câu sẽ hiện ở đây. | Xem tất cả (ghost) | -397 | 0.75 | – |
| ta | `/insights` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| ta | `/questions` | icon+tiêu đề+thân+1 nút | có | Chưa có câu hỏi nào | Chưa có câu hỏi nào. Tạo câu đầu tiên hoặc nhờ AI gợi ý nháp. | Tạo câu hỏi (primary) | -279 | 0.38 | **Tạo câu hỏi** |
| ta | `/settings` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| ta | `/students` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| ta | `/students/01a1d9dc-1afd-779f-874c-aa099aecd39f` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| ta | `/threads` | tiêu đề+thân+1 nút | – | Chưa có câu hỏi nào. Đặt câu hỏi đầu tiên. | Đặt câu hỏi | Đặt câu hỏi (primary) | -279 | 0.22 | **Đặt câu hỏi** |
| ta | `/threads/01a1d9df-6c17-7b13-b71b-d5e4ad4d4540` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| teacher | `/` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| teacher | `/analytics` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| teacher | `/attendance` | tiêu đề+1 nút | – | Lớp chưa có lịch buổi học. | – | Tạo lịch buổi học (primary) | -279 | 0.23 | – |
| teacher | `/calendar` | tiêu đề+1 nút | – | Chưa có gì trong lịch. | – | Thêm sự kiện (primary) | -279 | 0.23 | **Thêm sự kiện** |
| teacher | `/class/members` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| teacher | `/class/settings` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| teacher | `/documents` | tiêu đề+thân+1 nút | – | Lớp chưa có tài liệu. | Tải tài liệu lên | Tải tài liệu lên (primary) | -279 | 0.5 | **Tải tài liệu lên** |
| teacher | `/exams` | icon+tiêu đề+thân+1 nút | có | Chưa có bài thi nào | Chưa có bài thi nào. Tạo bài thi đầu tiên từ ngân hàng câu hỏi. | Tạo bài thi (primary) | -279 | 0.31 | **Tạo bài thi** |
| teacher | `/exams/01a1d9e1-2fb4-75c4-9318-fbc4a4d6c8ed` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| teacher | `/exams/01a1d9e1-2fb4-75c4-9318-fbc4a4d6c8ed/results` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| teacher | `/gradebook` | tiêu đề+1 nút | – | Chưa có công thức điểm. | – | Thiết lập công thức (primary) | -279 | 0.28 | – |
| teacher | `/gradebook/scheme` | icon+tiêu đề+1 nút | có | Lớp chưa có quy chế. | – | Tải quy chế (primary) | -171 | 0.27 | **Tải quy chế** |
| teacher | `/grading` | icon+tiêu đề+thân+1 nút | có | Chưa có bài nộp. | Sinh viên nộp trong ứng dụng, hoặc thầy/cô tải ZIP lên. | Tải bài lên (secondary) | -279 | 0.38 | – |
| teacher | `/grading/01a1d9df-dcb3-7b53-9c26-1ec49dca800e` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| teacher | `/inbox` | tiêu đề+thân+1 nút | – | Chưa có câu hỏi nào cần thầy/cô xử lý. | Khi AI không đủ chắc chắn hoặc sinh viên nhờ hỗ trợ, câu sẽ hiện ở đây. | Xem tất cả (ghost) | -397 | 0.75 | – |
| teacher | `/insights` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| teacher | `/observability` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| teacher | `/questions` | icon+tiêu đề+thân+1 nút | có | Chưa có câu hỏi nào | Chưa có câu hỏi nào. Tạo câu đầu tiên hoặc nhờ AI gợi ý nháp. | Tạo câu hỏi (primary) | -279 | 0.38 | **Tạo câu hỏi** |
| teacher | `/settings` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| teacher | `/settings/integrations` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| teacher | `/settings/llm` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| teacher | `/students` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| teacher | `/students/01a1d9dc-1afd-779f-874c-aa099aecd39f` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |
| teacher | `/threads` | tiêu đề+thân+1 nút | – | Chưa có câu hỏi nào. Đặt câu hỏi đầu tiên. | Đặt câu hỏi | Đặt câu hỏi (primary) | -279 | 0.22 | **Đặt câu hỏi** |
| teacher | `/threads/01a1d9df-6c17-7b13-b71b-d5e4ad4d4540` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |

Kiểu khối rỗng: `tiêu đề+thân+1 nút` × 8; `tiêu đề` × 7; `tiêu đề+1 nút` × 6; `icon+tiêu đề+thân+1 nút` × 5; `icon+tiêu đề+1 nút` × 2; `tiêu đề+3 nút` × 1; `tiêu đề+thân` × 1; `icon+tiêu đề+thân` × 1 → **8** kiểu khác nhau (hợp đồng: 1).

Route rỗng nhưng **không** dùng `EmptyState` (xem ảnh để biết có trạng thái rỗng tự dựng hay còn dữ liệu): student/, student/settings, student/assignments/01a1d9df-dbb9-77b2-ab9b-73cced18a0c0, student/threads/01a1d9df-6c17-7b13-b71b-d5e4ad4d4540, ta/, ta/students, ta/insights, ta/analytics, ta/class/members, ta/class/settings, ta/settings, ta/students/01a1d9dc-1afd-779f-874c-aa099aecd39f, ta/exams/01a1d9e1-2fb4-75c4-9318-fbc4a4d6c8ed, ta/grading/01a1d9df-dcb3-7b53-9c26-1ec49dca800e, ta/threads/01a1d9df-6c17-7b13-b71b-d5e4ad4d4540, teacher/, teacher/students, teacher/insights, teacher/analytics, teacher/class/members, teacher/class/settings, teacher/observability, teacher/settings/llm, teacher/settings/integrations, teacher/settings, teacher/students/01a1d9dc-1afd-779f-874c-aa099aecd39f, teacher/exams/01a1d9e1-2fb4-75c4-9318-fbc4a4d6c8ed, teacher/exams/01a1d9e1-2fb4-75c4-9318-fbc4a4d6c8ed/results, teacher/grading/01a1d9df-dcb3-7b53-9c26-1ec49dca800e, teacher/threads/01a1d9df-6c17-7b13-b71b-d5e4ad4d4540


## K5. Nút trùng trong cùng màn (cả hai trạng thái, 1440)

| Vai | Route | trạng thái | nút xuất hiện > 1 lần |
| --- | --- | --- | --- |
| student | `/exams` | data | Xem kết quả×3 |
| student | `/library` | data | Xem×6 |
| student | `/me` | data | Căn cứ · tr. 1×3 |
| student | `/practice` | data | Luyện 10 câu×9, Luyện 4 câu×4 |
| student | `/practice/history` | empty | Luyện đề×2 |
| student | `/settings` | data | Hiện mật khẩu×3, Thao tác cho Trình duyệt khác×16 |
| student | `/settings` | empty | Hiện mật khẩu×3, Thao tác cho Chrome trên macOS×4, Thao tác cho Trình duyệt khác×5 |
| student | `/threads` | empty | Đặt câu hỏi×2 |
| student | `/threads/01a1d9df-6c17-7b13-b71b-d5e4ad4d4540` | data | Tuỳ chọn bài×2 |
| ta | `/` | data | Mở×6 |
| ta | `/calendar` | empty | Thêm sự kiện×2 |
| ta | `/documents` | empty | Tải tài liệu lên×2 |
| ta | `/exams` | empty | Tạo bài thi×2 |
| ta | `/gradebook` | empty | Xem công thức×2 |
| ta | `/gradebook/scheme` | data | Trang 1×3, Trang 2×3 |
| ta | `/grading` | data | Duyệt×6 |
| ta | `/questions` | empty | Tạo câu hỏi×2 |
| ta | `/settings` | data | Hiện mật khẩu×3, Thao tác cho Chrome trên macOS×2 |
| ta | `/settings` | empty | Hiện mật khẩu×3, Thao tác cho Chrome trên macOS×4 |
| ta | `/threads` | empty | Đặt câu hỏi×2 |
| ta | `/threads/01a1d9df-6c17-7b13-b71b-d5e4ad4d4540` | data | Tuỳ chọn bài×2 |
| teacher | `/` | data | Mở×7 |
| teacher | `/calendar` | empty | Thêm sự kiện×2 |
| teacher | `/class/settings` | data | Dùng lại tài liệu×2 |
| teacher | `/documents` | empty | Tải tài liệu lên×2 |
| teacher | `/exams` | empty | Tạo bài thi×2 |
| teacher | `/exams/01a1d9e1-2fb4-75c4-9318-fbc4a4d6c8ed/results` | empty | Thử lại×2 |
| teacher | `/gradebook/scheme` | data | Trang 1×3, Trang 2×3 |
| teacher | `/gradebook/scheme` | empty | Tải quy chế×2 |
| teacher | `/grading` | data | Duyệt×6 |
| teacher | `/grading` | empty | Tải bài lên×2 |
| teacher | `/insights` | data | Tạo thread ghim×5, Tạo buổi ôn tập×5 |
| teacher | `/insights` | empty | Tạo thread ghim×5, Tạo buổi ôn tập×5 |
| teacher | `/questions` | empty | Tạo câu hỏi×2 |
| teacher | `/settings` | data | Hiện mật khẩu×3, Thao tác cho Chrome trên macOS×36, Thao tác cho Trình duyệt khác×13 |
| teacher | `/settings` | empty | Hiện mật khẩu×3, Thao tác cho Chrome trên macOS×38, Thao tác cho Trình duyệt khác×11 |
| teacher | `/settings/llm` | data | Cài đặt nâng cao×6 |
| teacher | `/settings/llm` | empty | Cài đặt nâng cao×6 |
| teacher | `/threads` | empty | Đặt câu hỏi×2 |
| teacher | `/threads/01a1d9df-6c17-7b13-b71b-d5e4ad4d4540` | data | Tuỳ chọn bài×2 |
| admin | `/settings` | data | Hiện mật khẩu×3, Thao tác cho Trình duyệt khác×9 |
| admin | `/settings/integrations` | data | Hiện mật khẩu×2, Lưu×2 |
| admin | `/settings/llm` | data | Test kết nối×2, Cài đặt nâng cao×9, Theo lớp×3 |

## K6. 375 px

| Vai | Route | trạng thái | tràn ngang | page px |
| --- | --- | --- | --- | --- |

## Tổng hợp theo hợp đồng (số mục lệch hiện trạng)

| Mã | Quy tắc / lỗi | Số mục | Danh sách (vai+route) |
| --- | --- | --- | --- |
| K2a | nút đầu trang nằm DƯỚI tiêu đề | 3 | ta/exams/01a1d9e1-2fb4-75c4-9318-fbc4a4d6c8ed, teacher/exams/01a1d9e1-2fb4-75c4-9318-fbc4a4d6c8ed, teacher/insights |
| K2b | meta > 1 dòng | 2 | ta/insights, teacher/insights |
| K2c | meta lặp mã / tên lớp | 9 | ta/analytics, ta/inbox, ta/inbox [rỗng], ta/insights, teacher/analytics, teacher/inbox, teacher/inbox [rỗng], teacher/insights, teacher/observability |
| K2d | dòng mô tả dưới tiêu đề trang | 12 | admin/, student/, ta/, ta/attendance, ta/exams/01a1d9e1-2fb4-75c4-9318-fbc4a4d6c8ed, ta/students, ta/students/01a1d9dc-1afd-779f-874c-aa099aecd39f, teacher/, teacher/attendance, teacher/exams/01a1d9e1-2fb4-75c4-9318-fbc4a4d6c8ed, teacher/students, teacher/students/01a1d9dc-1afd-779f-874c-aa099aecd39f |
| K2f | > 1 nút primary trên trang | 1 | teacher/grading (2) |
| K3 | tiêu đề mục có dòng phụ | 2 | ta/insights (1), teacher/insights (1) |
| K4a | EmptyState có thân chữ | 15 | student/exams, student/practice/history, student/threads, ta/documents, ta/exams, ta/grading, ta/inbox, ta/questions, ta/threads, teacher/documents, teacher/exams, teacher/grading, teacher/inbox, teacher/questions, teacher/threads |
| K4b | khối rỗng không căn giữa (lệch tâm > 40 px) | 28 | student/calendar (-279), student/exams (-171), student/library (-279), student/library/01a1d9dd-35df-7b1a-8352-d45930902823 (-279), student/me (-171), student/practice (-171), student/practice/history (-171), student/threads (-279), ta/attendance (-279), ta/calendar (-279), ta/documents (-279), ta/exams (-279), ta/gradebook (-279), ta/gradebook/scheme (-171), ta/grading (-279), ta/inbox (-397), ta/questions (-279), ta/threads (-279), teacher/attendance (-279), teacher/calendar (-279), teacher/documents (-279), teacher/exams (-279), teacher/gradebook (-279), teacher/gradebook/scheme (-171), tea |
| K4d | > 1 nút trong khối rỗng | 1 | student/chat |
| K4e | khối rỗng có icon (mẫu không thống nhất) | 8 | ta/exams, ta/gradebook/scheme, ta/grading, ta/questions, teacher/exams, teacher/gradebook/scheme, teacher/grading, teacher/questions |
| K4f | nút rỗng trùng nút đầu trang | 12 | student/threads (Đặt câu hỏi), ta/calendar (Thêm sự kiện), ta/documents (Tải tài liệu lên), ta/exams (Tạo bài thi), ta/questions (Tạo câu hỏi), ta/threads (Đặt câu hỏi), teacher/calendar (Thêm sự kiện), teacher/documents (Tải tài liệu lên), teacher/exams (Tạo bài thi), teacher/gradebook/scheme (Tải quy chế), teacher/questions (Tạo câu hỏi), teacher/threads (Đặt câu hỏi) |

# DEV handoff — US-PROTO-01 Sinh viên
Nhánh `sprint/1.5-mock-ui` · commit: `US-PROTO-01: ...`

## Đã làm (theo thứ tự lát dọc)
- Mock data: `frontend/src/mock/{student,chat,threads,practice,library,calendar}.ts`.
- Màn hình tính năng (`frontend/src/features/`):
  - `today/StudentHome.tsx` + `StudentHome.module.css`: khuyến nghị duy nhất (A: ôn Mật mã đối xứng sai 4/7 · 15p; B: QUIZ01 đóng sau 18 giờ · 20p; nút đỏ duy nhất), dòng thời gian hôm nay (buổi 10 09:00–11:30 P.302 đang diễn ra, hạn QUIZ01), học dở. SV D: ô nhập mã tham gia thay khuyến nghị ('mã tham gia'). ActionRow redThread.
  - `chat/`: D1 (dòng ẩn 2 thông tin cá nhân, trả lời chảy useStreamedText, nêu vắng 2 buổi (10/09, 08/10), +0,75 điểm cho 3 lần phát biểu, khối số liệu gọn, nguồn tham khảo trích Quy chế và Sổ điểm danh 761987, Hữu ích/Không hữu ích/Nhờ GV hỗ trợ, không chuỗi `[[SV_1]]`), D2 (từ chối lịch sự, không lộ số liệu bạn C), D3 (AI chưa đủ chắc chắn → tự chuyển GV, tạo ticketD3, nút thành 'Đang chờ giảng viên · vừa gửi'; khi GV trả lời hiển thị câu trả lời nhãn GV và nút 'Đã rõ'). Lịch sử 4 phiên (ẩn ≤ 720px); `?state=error` giữ nguyên chữ trong composer.
  - `threads/`: 12 thread lớp 1, 2 ghim, ghim từ Insights hiện đầu. Soạn bài có MSSV/tên/'điểm của em' mở `PIIChannelDialog` hai lối (Chuyển sang chat riêng / Ẩn thông tin rồi đăng). `threads/[id]`: câu trả lời AI 'Chờ xác nhận', câu trả lời GV có đường kẻ xanh 1px + nhãn xác nhận. Báo cáo, sửa/xoá bài của mình.
  - `practice/`, `practice/[attemptId]`, `practice/history`: luyện chủ đề (chọn đáp án phản hồi ngay có giải thích) và QUIZ01 (có đếm giờ, nộp bài qua xác nhận, Chat riêng chỉ thủ tục). Lịch sử 6 lượt.
  - `library/`: tài liệu visible_to_students (không ANSWER_KEY), tìm kiếm lọc, xem chi tiết, 'Hỏi AI về tài liệu' → /chat có ngữ cảnh, đề cũ có 'Luyện đề này'.
  - `calendar/`: Tuần / Tháng / Danh sách; mobile mặc định Danh sách; đọc `KEYS.calendarExtras`; hiển thị buổi 10, QUIZ01, hạn BT, giữa kỳ 05/11, cuối kỳ tuần 16.
  - `me/`: câu nhận định điểm QT 8,3 (tạm tính); giải trình tuyến tính TB bài tập 7,5 + cộng 0,75 = 8,25 → 8,3; sau điểm danh lên 8,5; sau công bố BT03 lên 8,7. What-if CK=8,0 → 8,3; số ngoài 0–10 báo lỗi tại ô. SV A chọn lớp 2 → thông báo lớp chưa có công thức điểm chính thức. Không có từ cấm, không nhãn rủi ro.
  - `assignments/[id]`: BT03 của B: đã nộp muộn 1 ngày, trước công bố ghi 'Đang chấm' (không lộ số); sau công bố: 8,0 điểm, nhận xét 4 tiêu chí trích bài của B, 'Yêu cầu xem lại' trong 7 ngày. QUIZ01: nút làm bài.
  - `join/`, `join/[code]`: BX4P9TW xem trước An ninh mạng · 761988 · TS. Lê Thu Hà · HK1 2026–2027 → 'Tham gia lớp' → pending + chờ duyệt; AN7K2MQ → vào ngay; mã sai → câu chung 'Mã không hợp lệ hoặc đã hết hạn...'; sai 5 lần → khoá 10 phút.
- Routes app: `frontend/src/app/(app)/{chat,threads,threads/[id],practice,practice/[attemptId],practice/history,library,calendar,me,assignments/[id],join,join/[code]}/page.tsx`.

## File đổi
`frontend/src/features/{chat,threads,practice,library,calendar,me,assignments,join}/**`
`frontend/src/features/today/StudentHome.tsx`, `StudentHome.module.css`
`frontend/src/mock/{student,chat,threads,practice,library,calendar}.ts`
`frontend/src/app/(app)/{chat,threads,threads/[id],practice,practice/[attemptId],practice/history,library,calendar,me,assignments/[id],join,join/[code]}/page.tsx`
`docs/sprints/1.5/shots/01-*.png`

## Lệnh QC chạy để kiểm
```bash
source ~/.zprofile
# Kiểm tra tĩnh và chạy test curl của QC
pnpm -C frontend lint && pnpm -C frontend build && bash scripts/ui-antipatterns.sh
F=http://localhost:3000 bash docs/sprints/1.5/qc/scripts/proto-curl.sh tc_01_01 tc_01_05 tc_01_08 tc_01_09
```

## Test đã chạy và kết quả
- `proto-curl.sh tc_01_01 tc_01_05 tc_01_08 tc_01_09`: PASS 100% (mọi route mở được với SV B, sạch từ cấm RAG/PII/LLM/confidence/risk, không lộ điểm nháp, xem trước lớp 761988, phân quyền chặn đúng các vai khác).
- Trình duyệt: ảnh chụp desktop 1440 và mobile 390 tại `docs/sprints/1.5/shots/01-*.png`.

## AC tự đánh giá
01-AC1 ✓ · 01-AC2 ✓ · 01-AC3 ✓ · 01-AC4 ✓ · 01-AC5 ✓ · 01-AC6 ✓ · 01-AC7 ✓ · 01-AC8 ✓ · 01-AC9 ✓.

## Nợ / chưa làm / cần hỏi
- Kịch bản What-if hiện tính theo công thức chuẩn đã chốt (QT 8,7 sau công bố, CK 8,0 → HP 8,3).
- Phần xem trước PDF bài giảng trong Thư viện hiển thị trang giả lập dạng văn bản/khung xem trước.

## Cập nhật theo spec v4 (Proposals #15, #16)
- Form tạo thread mới (`ThreadsScreen.tsx`): đầy đủ Tiêu đề (Input bắt buộc), Chủ đề (Select `THREAD_TOPICS`), Nội dung chi tiết (Textarea bắt buộc), Checkbox "Nhờ AI trả lời gợi ý (Socratic) ngay sau khi đăng" (mặc định bật), nút Đăng câu hỏi (primary). Quét PII 2 lối. Đăng thành công chuyển hướng ngay sang `/threads/${id}` (01-AC4).
- Chi tiết thread (`ThreadDetail.tsx`): cấu trúc 4 panel độc lập rõ ràng (câu hỏi gốc, câu trả lời AI, thảo luận, Reply Composer). Trích dẫn citations bấm mở rộng xem snippet đoạn trích.
- Reply Composer ở cuối trang: nhập phản hồi, nút "Hỏi trợ lý AI" gợi ý Socratic, quét PII trước khi gửi (01-AC10).

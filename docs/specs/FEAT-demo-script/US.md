# FEAT-demo-script Kịch bản demo 15 phút
Nguồn: PRD §1, §4 (M0, M1, M3, M5, M7, M8, M13-B, M14), FLOWS F2, F3, F5, F7, F9, F10, F17, phase P0 lát L0

Sản phẩm: `docs/DEMO_SCRIPT.md`. Story chỉ có tài liệu, không có code, nên không có SRS (theo `docs/sprints/1/prompts/ba-sprint1.md`).

## US-P0-01: Chủ dự án muốn có kịch bản demo 15 phút để mọi phase sau biết mình đang làm cho bước nào
Ưu tiên: Must · Ước lượng: S · Sprint: 1

### Tiêu chí nghiệm thu
- AC1. Given `docs/DEMO_SCRIPT.md` When đọc các tiêu đề bước Then có đúng 7 bước theo thứ tự F2 → F3 → F5 → F7 → F9 → F10 → F17.
  Kiểm: `grep -oE '^### Bước [0-9]+\..*— F[0-9]+' docs/DEMO_SCRIPT.md | grep -oE 'F[0-9]+$' | xargs` → `F2 F3 F5 F7 F9 F10 F17`.
- AC2. Given mỗi bước When đọc bảng của bước Then mỗi hàng có đủ 6 cột: mốc thời gian, vai trò, route, thao tác, kết quả phải thấy, phase làm cho hàng đó chạy thật.
  Kiểm: `awk -F'|' '/^\| [0-9]{2}:[0-9]{2} \|/ && NF!=8 {print NR": "$0}' docs/DEMO_SCRIPT.md` → không in gì; `grep -cE '^\| [0-9]{2}:[0-9]{2} \|' docs/DEMO_SCRIPT.md` ≥ 30.
- AC3. Given bảng thời lượng ở mục 1 When cộng thời lượng các phần Then tổng ≤ 15:00 và mốc của mọi hàng tăng dần, hàng cuối cùng < 15:00.
  Kiểm: `grep -oE '^\| [0-9]{2}:[0-9]{2} \|' docs/DEMO_SCRIPT.md | tr -d '| ' | sort -c && echo tang-dan`; xem bằng mắt bảng mục 1: dòng "Hết kịch bản" ≤ 15:00.
- AC4. Given mọi route trong cột Route When đối chiếu với `docs/design/DESIGN.md` §14 và `docs/ARCHITECTURE.md` mục 7 Then route nào cũng có ở ít nhất một trong hai nơi (không bịa màn hình).
  Kiểm:
  ```bash
  for r in $(grep -E '^\| [0-9]{2}:[0-9]{2} \|' docs/DEMO_SCRIPT.md | awk -F'|' '{print $4}' | grep -oE '`/[^`]*`' | tr -d '`' | sed -E 's#/BX4P9TW#/[code]#' | sort -u); do
    grep -qF "\`$r\`" docs/design/DESIGN.md docs/ARCHITECTURE.md || echo "THIẾU $r"; done
  ```
  → không in dòng `THIẾU`.
- AC5 (lời văn phía sinh viên). Given các hàng có vai trò Sinh viên When tìm thuật ngữ kỹ thuật AI Then không có từ nào trong danh sách cấm (DESIGN §13).
  Kiểm: `grep -E '^\| [0-9:]+ \| Sinh viên' docs/DEMO_SCRIPT.md | grep -inE 'RAG|PII|fallback|trace|provider|confidence|escalat|prompt|LLM|model|token|placeholder|embedding'` → không in gì.
- AC6. Given mục 5 "Dự phòng" When đọc Then có đủ: `DEMO_MODE=true`, provider `fake`, các câu được ghi sẵn (mục 3), video quay sẵn, ai bật, bật ở file nào bằng lệnh gì, khi nào chuyển tầng.
  Kiểm: `for k in 'DEMO_MODE=true' 'fake' 'Video' 'Ai bật' '.env.local' 'up -d python-ai' 'lỗi 2 lần'; do grep -qF "$k" docs/DEMO_SCRIPT.md || echo "THIẾU $k"; done` → không in gì.
- AC7 (nhánh lỗi). Given API LLM hoặc mạng hỏng ngay trong buổi bảo vệ When người trình bày áp quy tắc chuyển tầng ở mục 5 Then kịch bản vẫn đi tiếp được bằng tầng 2 hoặc tầng 3; và mỗi bước F2–F10 có dòng "Nếu hội đồng hỏi" chỉ ra nhánh lỗi + tên spec E2E tương ứng trong FLOWS.
  Kiểm: `grep -c '^Nếu hội đồng hỏi' docs/DEMO_SCRIPT.md` ≥ 5; mỗi tên `*.spec.ts` trong DEMO_SCRIPT có trong FLOWS: `for s in $(grep -oE '[a-z-]+\.spec\.ts' docs/DEMO_SCRIPT.md | sort -u); do grep -qF "$s" docs/FLOWS.md || echo "THIẾU $s"; done` → không in gì.
- AC8 (phân quyền). Given kịch bản When đọc Then có ít nhất 3 hàng cho thấy ranh giới quyền chạy thật: sinh viên hỏi dữ liệu người khác bị từ chối (bước 2), sinh viên không thấy điểm nháp (bước 5), chỉ TEACHER công bố / `Chốt điểm` bị khoá khi công thức chưa xác nhận (bước 5, 6).
  Kiểm: `grep -cE 'chỉ trả lời được thông tin của chính bạn|không có điểm nháp|chỉ TEACHER|Chốt điểm. bị khoá' docs/DEMO_SCRIPT.md` ≥ 4.
- AC9. Given mỗi phase P1–P10 có trong cột Phase When PM rà Then mục 6 liệt kê yêu cầu kịch bản đặt lên từng phase đó, và mục 7 có bảng nhật ký chạy lại cuối mỗi phase.
  Kiểm: `grep -E '^\| P(1|2|3|4|5|6|7|10) \|' docs/DEMO_SCRIPT.md | wc -l` = 8; `grep -c '^## 7. Nhật ký chạy lại' docs/DEMO_SCRIPT.md` = 1.

### Ngoài phạm vi của story này
- Seed khớp kịch bản, provider `fake`, cờ `DEMO_MODE`, file `seed/demo/*`: việc của P1, P2, P6, P7, P10 (đã liệt kê ở mục 6 của kịch bản).
- Quay video (P10).
- Kịch bản cho F1, F4, F6, F8, F11–F16, F18.

### Phụ thuộc
- Không. Câu hỏi mở: `QUESTIONS.md` Q1–Q9 (không chặn việc nghiệm thu US này; chặn các phase ghi trong từng câu).

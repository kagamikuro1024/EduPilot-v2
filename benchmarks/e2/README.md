# E2 — golden set hiệu chỉnh ngưỡng escalation (US-P10-10 AC4, P-11-4)

Soạn bởi BA (2026-11-14) từ tài liệu seed của **lớp 1 (761987)**. Dữ liệu mô phỏng (D44). Chấm **tất định**, không dùng LLM làm giám khảo.

## Tệp
- `golden_e2.jsonl`: 120 mục. `check_golden.py`: kiểm cấu trúc + bằng chứng (chỉ thư viện chuẩn).

| Nhóm | Nguồn | `answerable` | Số mục |
| --- | --- | --- | --- |
| `policy` | `quy-che-lop1.pdf` (nội dung gốc `docs/specs/FEAT-gradebook/fixtures/quy-che-lop1.md`) | true | 15 |
| `policy` | ngoài quy chế lớp | false | 5 |
| `academic` | `Mordern_Network_Security_Threats.pdf` (45) và `QMB12ch6b.pdf` (35) | true | 80 |
| `academic` | ngoài tài liệu seed (cùng lĩnh vực hoặc ngoài lề) | false | 20 |

## Trường
`id`, `set`, `topic`, `source` (tên PDF), `page` (trang PDF; quy chế: trang trong dấu `<!-- trang N -->`), `question` (tiếng Việt, mỗi câu khác hẳn nhau để Threads không gộp / không chặn), `answerable`, `must_contain`, `evidence` (câu **nguyên văn** trong tài liệu nguồn; mục `answerable=false` để `null`).

## Luật chấm (bộ chạy E2 import `norm`, `passes` từ `check_golden.py`)
- `norm`: bỏ dấu tiếng Việt (`đ`→`d`), chữ thường, `–`/`—`→`-`, bỏ `*` `_` `` ` ``, dấu phẩy giữa hai chữ số → dấu chấm (`7,5` = `7.5`), gộp khoảng trắng.
- `must_contain` = danh sách **nhóm**; mỗi nhóm là danh sách biến thể (tiếng Việt / tiếng Anh / cách viết số). **Đúng** khi mỗi nhóm có ít nhất một biến thể (đã `norm`) là chuỗi con của câu trả lời đã `norm`. Thiếu một nhóm là sai.
- `answerable=false`: đúng khi hệ thống **không bịa**: bài trả lời Threads có `no_context` (hoặc bị chuyển giảng viên); trả lời có nội dung mà không `no_context` là **bịa**.
- Đường cong: một lượt hỏi mỗi mục, đọc `ThreadPost.confidence`; quét ngưỡng 0,30 → 0,90 bước 0,05 (TL-1). Câu `SKIPPED` / bị chặn ghi riêng, không vào đường cong.

## Giới hạn (ghi vào báo cáo E2)
- Nhãn do BA soạn từ văn bản nguồn, **chưa có người ngoài đội soát**: chủ dự án soát 20 % (24 mục, bảng dưới) → `labels: WAITING_HUMAN` đến khi có kết quả.
- `Quyche.pdf` (quy chế trường) là bản quét không lớp chữ nên **không** dùng làm nguồn `answerable`. Nếu Docling OCR được và một mục `answerable=false` nhóm `policy` lại có đáp án trong đó → loại mục đó và ghi vào bảng soát.
- `must_contain` so chuỗi: câu trả lời đúng nhưng diễn đạt bằng từ ngoài biến thể sẽ bị chấm sai; sai số này thuộc phép đo, không sửa nhãn cho vừa kết quả.
- Mọi câu hỏi và nguồn đều mô phỏng; không dữ liệu sinh viên.

## Kiểm
```bash
# văn bản trích offline (ví dụ pdftotext hoặc pypdf), một tệp <tên PDF>.txt mỗi PDF
python3 benchmarks/e2/check_golden.py [--text-dir DIR]
```
Kỳ vọng: `120 mục … OK`.

## Bảng soát của chủ dự án (24 mục, cách 5 mục một)
Điền `Đúng` / `Sai nhãn` / `Câu hỏi mơ hồ` và ghi chú; BA phán khi bất đồng.

| id | Kết luận | Ghi chú | BA phán |
| --- | --- | --- | --- |
| E2-005 | | | |
| E2-010 | | | |
| E2-015 | | | |
| E2-020 | | | |
| E2-025 | | | |
| E2-030 | | | |
| E2-035 | | | |
| E2-040 | | | |
| E2-045 | | | |
| E2-050 | | | |
| E2-055 | | | |
| E2-060 | | | |
| E2-065 | | | |
| E2-070 | | | |
| E2-075 | | | |
| E2-080 | | | |
| E2-085 | | | |
| E2-090 | | | |
| E2-095 | | | |
| E2-100 | | | |
| E2-105 | | | |
| E2-110 | | | |
| E2-115 | | | |
| E2-120 | | | |

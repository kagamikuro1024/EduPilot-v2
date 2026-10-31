# Sprint 9: câu hỏi kỹ thuật cho Tech Lead

`## TL-<số> (<ai>, <ngày>): <tiêu đề>`, Tech Lead trả lời ngay dưới.

## TL-1 (PM, 2026-10-31): proposals #10, kiểm LCP cho `/assignments/[id]` (US-P7-03 AC9)
Dev nói `lcp.spec.ts -g /assignments` không đạt được nếu không đổi hợp đồng `shared/shell/intro`: route động, `h1` là tiêu đề bài. Dev đề xuất bỏ khỏi AC9, chỉ giữ 375 px + a11y. Tech Lead xem mã `lcp.spec.ts` / Lighthouse CI và chọn cách **vẫn đo được** LCP màn nộp bài của sinh viên (route seed cố định, phần tử LCP khác, hoặc thêm vào danh sách Lighthouse CI), hoặc giải thích vì sao thật sự không đo được. Không nới ngưỡng. Trả lời ngay dưới, báo PM 2 dòng.

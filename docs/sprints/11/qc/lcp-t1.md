# LCP trên stack T1 (ep-t1) — số chính thức TC-08-02 (US-P10-08 AC2)
Stack: `ep-t1` (compose riêng, postgres riêng, DB `edupilot_t1`, offset 400, https://localhost:843), build production ở HEAD d080ea35, `seed-t1.mjs` KHÔNG `--reset`: 1.000 SV · 20 lớp · 3 GV · 5 TA · 90.000 chat · 525.000 study_time · 4.000 bài nộp (`check-t1-seed.mjs` OK; `seed-t1-ep-t1.json`). Đo bằng `benchmarks/lighthouse/run.sh` nguyên bản (QC tự chạy), 8 route × 3 lượt, Chrome máy, tài khoản thật t1sv0001 / t1gv1 / admin. Load máy: lượt (1) 3,35 → 2,49; lượt (2) 2,09 → 2,87 (< 8). Colima 6 CPU / 10 GB; ep-qc chỉ giữ postgres/redis/minio/pgbouncer/mock; ep-r của qc2 chạy nền.
Lưu ý lượt đầu bị bỏ: frontend dựng với NEXT_PUBLIC_API_URL mặc định (https://localhost, cổng 443) → mọi route chuyển về /login; đã sửa URL công khai (https://localhost:843) và đo lại. Số ở lượt đo hợp lệ dưới đây.

## Lượt (1) — trạng thái người dùng quay lại («Đã hiểu» đã bấm) — SỐ CHÍNH THỨC
Throttle **devtools** (3 lượt / trung vị):
| route | 3 lượt (ms) | trung vị | ≤ 2.500 | phần tử LCP |
| --- | --- | --- | --- | --- |
| / | 1609 / 1625 / 1617 | 1617 | đạt | <p class="Landing-module__H_TZ-W__lead" data-up="true" style |
| /chat | 1571 / 1585 / 1563 | 1571 | đạt |  |
| /threads | 1569 / 1558 / 1547 | 1558 | đạt |  |
| /inbox | 1545 / 1529 / 1563 | 1545 | đạt |  |
| /gradebook | 1534 / 1551 / 1567 | 1551 | đạt |  |
| /me | 1504 / 1509 / 1516 | 1509 | đạt |  |
| /settings/llm | 1476 / 1519 / 1514 | 1514 | đạt |  |
| /assignments/[id] | 4785 / 4815 / 4794 | 4794 | TRƯỢT | <p> |
Simulate (tham khảo; run.sh đo cả hai chế độ):
| route | 3 lượt | trung vị | ≤ 2.500 | phần tử LCP |
| --- | --- | --- | --- | --- |
| / | 2406 / 2406 / 2408 | 2406 | đạt | <p class="Landing-module__H_TZ-W__lead" data-up="true" style |
| /chat | 2715 / 2725 / 1511 | 2715 | TRƯỢT |  |
| /threads | 1958 / 2262 / 2580 | 2262 | đạt | <h1 class="ep-page-title" data-part="page-title"> |
| /inbox | 2579 / 1808 / 2579 | 2579 | TRƯỢT | <h1 class="ep-page-title" data-part="page-title"> |
| /gradebook | 2802 / 2871 / 2874 | 2871 | TRƯỢT | <h1 class="ep-page-title" data-part="page-title"> |
| /me | 1506 / 2555 / 1958 | 1958 | đạt |  |
| /settings/llm | 1509 / 2588 / 1957 | 1957 | đạt |  |
| /assignments/[id] | 2723 / 2724 / 2724 | 2724 | TRƯỢT | <p> |

## Lượt (2) — lần đầu (chưa «Đã hiểu», t1sv0002) — BÁO RIÊNG, không dùng chấm
devtools:
| route | 3 lượt | trung vị | ≤ 2.500 | phần tử LCP |
| --- | --- | --- | --- | --- |
| / | 1594 / 1613 / 1614 | 1613 | đạt | <p class="Landing-module__H_TZ-W__lead" data-up="true" style |
| /chat | 1553 / 1554 / 1565 | 1554 | đạt |  |
| /threads | 4377 / 4379 / 4366 | 4377 | TRƯỢT | <p> |
| /inbox | 1547 / 1568 / 1553 | 1553 | đạt |  |
| /gradebook | 1531 / 1530 / 1557 | 1531 | đạt |  |
| /me | 4323 / 4338 / 4335 | 4335 | TRƯỢT | <p> |
| /settings/llm | 1490 / 1516 / 1510 | 1510 | đạt |  |
| /assignments/[id] | 4862 / 4846 / 4851 | 4851 | TRƯỢT | <h2 id="_r_7_" class="ep-section-title" data-ep-section-titl |
simulate:
| route | 3 lượt | trung vị | ≤ 2.500 | phần tử LCP |
| --- | --- | --- | --- | --- |
| / | 2781 / 2556 / 2405 | 2556 | TRƯỢT | <p class="Landing-module__H_TZ-W__lead" data-up="true" style |
| /chat | 2888 / 2883 / 2408 | 2883 | TRƯỢT |  |
| /threads | 2257 / 2257 / 2578 | 2257 | đạt | <h1 class="ep-page-title" data-part="page-title"> |
| /inbox | 2170 / 2578 / 2407 | 2407 | đạt |  |
| /gradebook | 2871 / 2872 / 2874 | 2872 | TRƯỢT | <h1 class="ep-page-title" data-part="page-title"> |
| /me | 2749 / 1957 / 2106 | 2106 | đạt |  |
| /settings/llm | 1506 / 2557 / 1509 | 1509 | đạt |  |
| /assignments/[id] | 2720 / 2722 / 2722 | 2722 | TRƯỢT | <h2 id="_r_7_" class="ep-section-title" data-ep-section-titl |

## Kết luận (AC6, P-11-6)
- devtools lượt (1): 7/8 route ≤ 2.500 ms; **/assignments/[id] trượt: 4.794 ms** (phần tử LCP = đoạn mô tả bài tập `<p>`, render delay 4,73 s; refresh phiên 0,7 s → khoảng trống tới 3,55 s → `me/courses` + `notifications` → `assignments/<id>` bắt đầu 4,17 s: chờ `me/courses`, chuỗi phụ thuộc). Theo AC6: ghi số thật + phần tử + nguyên nhân vào Nợ; story không FAIL vì route đó; không nới ngưỡng, không SSR.
- Lượt đầu: /threads 4.377, /me 4.335, /assignments/[id] 4.851 (hộp thoại thời gian học là phần tử lớn nhất trước khi bấm «Đã hiểu»; P-11-5).
- Số seed ep-qc (5,1–5,3 s ở /threads, /gradebook, /me) KHÔNG tái lập ở T1 (1,5 s): ở ep-qc LCP là chữ dữ liệu; ở T1 là chữ khung (đúng như dev-wave3 nêu).

# QC sprint 9.1 (UI/UX): kết luận chung

**Kết luận: PASS có giới hạn.** 6 đợt, 21 story (US-UX91-01 … 21) đều PASS ở lượt kiểm cuối. Không còn lỗi mở. Mọi lỗi ghi nhận là mức Thấp, trừ một lỗi Trung bình (đợt 4, US-15 AC7, đã sửa).

## Kết quả theo đợt
| Đợt | Story | Lượt cuối | Commit kiểm | Kết luận |
| --- | --- | --- | --- | --- |
| 1 Nền chung | 02, 03, 04, 21 | 3 | `9aea5b95` | PASS |
| 2 Chữ và điều hướng | 01, 05, 06, 07 | 3 | `9aea5b95` | PASS |
| 3 Hôm nay | 10, 11 | 3 | `9aea5b95` | PASS |
| 4 Chấm bài, sổ điểm, điểm danh | 14, 15, 16, 12 (+ US-04 AC7) | 3 | `673441c5` | PASS (8/8 lỗi đã sửa) |
| 5 Chat, Hỏi đáp, Kết quả của tôi, Tài liệu, Insights | 13, 19, 20, 18, 17 | 4 | `18400636` | PASS (US-17 có Nợ) |
| 6 Trang giới thiệu, clip, video | 08, 09 | 4 | `3002ce33`; AC10a ở `18400636` | PASS có giới hạn |

Chi tiết và lịch sử từng lượt: `qc/wave-1.md` … `wave-6.md`.

## Giới hạn và Nợ (không tính FAIL)
| # | Nội dung | Gốc |
| --- | --- | --- |
| 1 | US-17 AC5, AC6 (UX17-8, 9): `/insights` và `/observability` đang dữ liệu mô phỏng; chưa kiểm được `audit_log` thật và lớp chưa có báo cáo. Kiểm sau khi nối API thật (phase P10) | P-91-15 |
| 2 | US-09 AC10: chưa quay trọn bản video giới thiệu trên `ep-qc`; clip, poster và `run.sh` (`STILLS_ONLY=1`) đã kiểm | `wave-6.md` |
| 3 | Firefox (Playwright 1543) không khởi động trên macOS arm64 (`Could not find profile folder`): ca UX08-28/48 Firefox không chạy lại trên Mac. Đã PASS ở VPS (lượt 3) và mã đợt 6 không đổi | `wave-6.md` |
| 4 | Ảnh `story-review-v1` dùng bài làm / điểm nháp mẫu dev dựng vào DB dev (demo không có bài nộp); chấp nhận vì là ảnh minh hoạ | `wave-5.md` |
| 5 | Đợt 1, 2, 3 kiểm lần cuối ở `9aea5b95`; từ đó mã sản phẩm đổi chủ yếu ở `RealChat`, `RealMe`, `RealDocuments`, `RealThreads`, `Me.module.css`, `Grade*`, ảnh `landing`. QC chỉ chạy lại đúng ca liên quan từng lượt, **không chạy lại toàn bộ bộ test cuối sprint**; bộ 916 ca và Lighthouse `/` là việc của dev (dev báo 916 pass / 1 flaky) | `wave-5.md` (UX91-X2) |

## Quyết định PM đã áp dụng khi kiểm
P-91-1 … 15 (xem `proposals.md`). Ảnh hưởng đến kết luận: P-91-9 (US-03 AC8 kiểm bằng E2E trên `/dev/ui`), P-91-10 (US-15 AC3, AC8; US-16 AC1), P-91-13 (US-13 AC1: «căn giữa trong phần bên phải cột lịch sử»), P-91-14 (US-20 AC8: gập sẵn hai khối, ≤ 1.600 px), P-91-15 (Nợ UX17-8, 9).

## Số đo chốt
| Mục | Giá trị |
| --- | --- |
| `/me` ở 375 px (`sv.kha`) | 1.571 px (ngưỡng ≤ 1.600); không tràn ngang |
| Ảnh khung ghim | `story-chat` 35.312 B, `story-gradebook` 44.192 B, `story-review` 79.546 B (≤ 90 KB) |
| Thanh `Duyệt bài` 1440×900 | `top 775 → bottom 844` (< 900), `sticky` |
| Spec đợt 6 (Mac) | 26 / 27 xanh (1 ca Firefox không chạy được) |

## Cấm và luật lõi (kiểm chéo)
Không thấy vi phạm: tên / MSSV không lộ trong log gateway (UX13-16), STUDENT không thấy điểm nháp / ghi chú quan sát / nhãn rủi ro (UX20-4), `?user_id=` bị bỏ qua (UX20-8), TEACHER / TA không đọc được chat riêng của SV (UX13-22: 403), không migration mới, không `fetch(` trần mới, `ui-antipatterns.sh`, `eslint`, `tsc` sạch ở lượt kiểm đợt 5.

## Lưu ý vận hành
Spec QC chạy-thật `frontend/e2e/qc-uiux91-*-real.spec.ts` cần lớp thử `QC117283` (rỗng) và `QC196964` (ARCHIVED) của `teacher` trên stack kiểm; DB sạch không có (tạo qua `POST /admin/courses` và `…/archive`). Nên chuyển bước này vào seed QC hoặc `beforeAll` nếu giữ bộ spec cho sprint sau.

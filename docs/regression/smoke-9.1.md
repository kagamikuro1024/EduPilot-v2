# Smoke hồi quy sprint 9.1 — kết quả (TẠM DỪNG theo lệnh PM — tải máy 44)
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm F) · Ngày: 2026-11-15
Tổng TC cũ: 414 · Đạt: 184 · Trượt: 82 · Lỗi thời (LT, theo 10.7): 51 · K (a/b/c): 0 · Chưa chạy (M): 97   (P+F+LT+K+M = 414)
Cách chạy: toàn bộ spec `frontend/e2e/qc-uiux91-*-real.spec.ts` (+ bản sao `frontend/e2e/qc2-r91/`, `qc2-r91` sửa đường ảnh /out và lớp mặc định) bằng `playwright.qc-real.config.ts --workers=1` trên https://localhost:1143; chạy lại ca đỏ (`--last-failed`); ca GREP/FFP/Lighthouse/OCR khung video/dev-spec chốt qua `evidence-9.1/make_overrides.py` → `overrides91.json` (93 mục); `build_smoke91.py` dựng bảng. Triage 118 F ở `evidence-9.1/triage.md`; chỗ dở ở `evidence-9.1/state.md`. Nhiều ca F là lượt đỏ cũ CHƯA chạy lại (nhóm G3/G4 = môi trường: dữ liệu ep-r bị đổi, timeout đăng nhập do bộ đếm auth / gateway draining; xem triage) — KHÔNG phải lỗi mã xác nhận.

## Tổng hợp theo US
| US | TC | P | F | LT | M |
| --- | ---: | ---: | ---: | ---: | ---: |
| UX01 | 20 | 12 | 3 | 0 | 5 |
| UX02 | 14 | 12 | 0 | 0 | 2 |
| UX03 | 18 | 14 | 0 | 0 | 4 |
| UX04 | 23 | 15 | 4 | 0 | 4 |
| UX05 | 12 | 5 | 0 | 5 | 2 |
| UX06 | 11 | 3 | 7 | 0 | 1 |
| UX07 | 18 | 14 | 2 | 0 | 2 |
| UX08 | 57 | 13 | 4 | 40 | 0 |
| UX09 | 21 | 12 | 2 | 6 | 1 |
| UX10 | 29 | 27 | 0 | 0 | 2 |
| UX11 | 11 | 10 | 0 | 0 | 1 |
| UX12 | 23 | 3 | 16 | 0 | 4 |
| UX13 | 28 | 5 | 7 | 0 | 16 |
| UX14 | 14 | 0 | 12 | 0 | 2 |
| UX15 | 22 | 0 | 17 | 0 | 5 |
| UX16 | 24 | 13 | 6 | 0 | 5 |
| UX17 | 15 | 2 | 0 | 0 | 13 |
| UX18 | 12 | 4 | 0 | 0 | 8 |
| UX19 | 15 | 3 | 2 | 0 | 10 |
| UX20 | 11 | 3 | 0 | 0 | 8 |
| UX21 | 12 | 11 | 0 | 0 | 1 |
| UX91 | 4 | 3 | 0 | 0 | 1 |

## Kết quả từng ca
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-UX02-1 | P | rg còn 5 dòng ngoài 13 tệp liệt kê: shared/lib/datetime.ts (mô-đun định dạng), insights/KnowledgeGap.tsx:14-15, observability/AdminView.tsx:19,46 (hai tệp không thuộc 13 tệp của AC1); 13 tệp đã sạch. logs/grep-batch.log  |
| TC-UX02-2 | P | spec w1 |
| TC-UX02-3 | P | spec w1 |
| TC-UX02-4 | P | spec w1 |
| TC-UX02-5 | P | spec w12b-real |
| TC-UX02-6 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX02-7 | P | spec w1 |
| TC-UX02-8 | P | spec w1 |
| TC-UX02-9 | P | spec w1 |
| TC-UX02-10 | P | frontend/e2e/datetime.spec.ts (5 ca: bảng AC2, 23:59/00:01 ICT, ranh giới nửa đêm, null → «—», múi giờ máy khác) xanh trên bản dựng cổng; không có vitest ở HEAD (script test đã bỏ) |
| TC-UX02-11 | P | frontend/e2e/datetime.spec.ts (5 ca: bảng AC2, 23:59/00:01 ICT, ranh giới nửa đêm, null → «—», múi giờ máy khác) xanh trên bản dựng cổng; không có vitest ở HEAD (script test đã bỏ) |
| TC-UX02-12 | M | chưa chạy (mẫu; dạng MAC, không có test tự động hoặc test bị skip) |
| TC-UX02-13 | P | spec w1 |
| TC-UX02-PQ | P | spec w12b-real |
| TC-UX03-1 | P | spec w1 |
| TC-UX03-2 | P | spec w1 |
| TC-UX03-3 | P | spec w1 |
| TC-UX03-4 | P | spec w1 |
| TC-UX03-5 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX03-6 | P | spec w1 |
| TC-UX03-7 | P | spec w1 |
| TC-UX03-8 | P | spec w1 |
| TC-UX03-9 | P | spec w1 |
| TC-UX03-10 | P | spec w1 |
| TC-UX03-11 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX03-12 | P | spec w1 |
| TC-UX03-13 | P | spec w1 |
| TC-UX03-14 | P | spec w1 |
| TC-UX03-15 | P | spec w1 |
| TC-UX03-16 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX03-17 | P | spec w1 |
| TC-UX03-18 | M | chưa chạy (mẫu; dạng MAC, không có test tự động hoặc test bị skip) |
| TC-UX04-1 | P | spec w1 đỏ vì selector `class*=Button-module` không khớp bản dựng cổng (tên lớp băm); bản sửa qc2-r91-mock-extra.spec.ts (data-variant) xanh: 5 biến thể nút, transform 0,97 ≤ 100 ms, trở lại |
| TC-UX04-2 | P | spec w1 đỏ vì selector `class*=Button-module` không khớp bản dựng cổng (tên lớp băm); bản sửa qc2-r91-mock-extra.spec.ts (data-variant) xanh: 5 biến thể nút, transform 0,97 ≤ 100 ms, trở lại |
| TC-UX04-3 | F | 9 luật :hover ngoài @media (hover: hover): gradebook/Gradebook.module.css:7 (9.1) và features/landing/Landing.module.css:35,39,40,73,97,144,226 (10.7) → DEF-R9.1-02 |
| TC-UX04-4 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX04-5 | P | tokens.css:73-75 --ep-fast 140ms, --ep-normal 200ms, --ep-slow 200ms (≤ 200) |
| TC-UX04-6 | P | spec w1 |
| TC-UX04-7 | P | DESIGN.md:492 «- Drawer/dialog: ≤ 200ms.» đúng một dòng (không có khoảng trắng quanh «/» so với mẫu rg) |
| TC-UX04-8 | P | rg `transition: width` trong frontend/src: 0 kết quả |
| TC-UX04-9 | P | spec w1 |
| TC-UX04-10 | P | spec w1 |
| TC-UX04-11 | P | spec w1 |
| TC-UX04-12 | F | spec w4c-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX04-13 | F | spec w4c-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX04-14 | F | spec w4c-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX04-15 | P | git diff 2c15dfce^1 2c15dfce -- frontend/package.json: chỉ bỏ framer-motion, embla-carousel*; không thêm thư viện chuyển động |
| TC-UX04-16 | P | evidence-9.1/scripts/keyframes-check.mjs: keyframes ngoài landing chỉ animate opacity/transform (+ ngoại lệ ep-saved background-color, spinner, ba chấm, shimmer); không stagger ngoài landing. Landing có stagger/nhiều key |
| TC-UX04-17 | P | spec w12b-real |
| TC-UX04-18 | P | ep-motion-safe: SubmissionReview.tsx:498,534, DataTableVirtual.tsx:209, AttendanceGrid.tsx:123; base.css:52 luật lưới 1ms còn nguyên, :55 .ep-motion-safe 200/160 ms |
| TC-UX04-19 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX04-20 | P | spec w1 |
| TC-UX04-21 | P | spec w1 |
| TC-UX04-22 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX04-PQ | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX21-1 | P | spec w1 |
| TC-UX21-2 | P | spec w1 |
| TC-UX21-3 | P | spec w1 |
| TC-UX21-4 | P | spec w1 |
| TC-UX21-5 | P | spec w1 |
| TC-UX21-6 | P | spec w1 |
| TC-UX21-7 | P | spec w1 |
| TC-UX21-8 | P | spec w1 |
| TC-UX21-9 | P | spec w1 |
| TC-UX21-10 | P | spec w1 |
| TC-UX21-11 | P | frontend/e2e/landing.spec.ts: «không gọi API nào (kể cả refresh)» (dòng 17-35) + «có cờ ⇒ vẫn gọi làm mới phiên» (dòng 38); tệp này chạy xanh 21/21 trên bản dựng cổng (logs/devspecs.log) |
| TC-UX21-PQ | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX01-1 | F | spec w2-real: 2 lượt đỏ / 16 xanh (xem logs/) |
| TC-UX01-2 | P | 28 mảnh kiểm của bảng tc-wave-2: 0 kết quả (rg -F, bỏ Demo*.tsx, test, spec) |
| TC-UX01-3 | P | spec w2-real |
| TC-UX01-4 | P | spec w12c-real,w5r-real |
| TC-UX01-5 | F | spec w12c-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX01-6 | P | diff 3 tệp Demo*.tsx (7 dòng): chỉ nhãn «Threads»→«Hỏi đáp công khai» (US-07 AC4) và progressStyle (US-04 AC4); không đổi description (như QC 9.1 đã ghi nhận) |
| TC-UX01-7 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX01-8 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX01-9 | P | spec w2-real |
| TC-UX01-10 | F | spec w2-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX01-11 | P | spec w12c-real,w5r-real |
| TC-UX01-12 | P | git diff 2c15dfce^1 2c15dfce -U0 -- frontend/src / rg '^\+.*(Info/CircleHelp)': 0 dòng (CircleHelp/Info xuất hiện ở HEAD là icon nav Luyện đề và icon thông báo ở sprint 10, không phải câu giải thích) |
| TC-UX01-13 | P | logs/antipatterns.log: PageHeader description tĩnh > 40 ký tự trên bản sao → ✗ «Câu mô tả tĩnh dưới tiêu đề», exit 1 |
| TC-UX01-14 | P | logs/antipatterns.log: description={"abc"} và description={dynamicVar} → không ✗, exit 0 |
| TC-UX01-15 | P | logs/antipatterns.log: --selftest = 32 / 32 phép bắt được (có phép «Câu mô tả tĩnh dưới tiêu đề») |
| TC-UX01-16 | P | spec w12c-real |
| TC-UX01-17 | P | spec w2-real |
| TC-UX01-18 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX01-19 | M | chưa chạy (mẫu; dạng MAC, không có test tự động hoặc test bị skip) |
| TC-UX01-PQ | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX05-1 | LT | P-107-2: trạng thái rỗng = tiêu đề ngắn + tối đa một nút; bỏ câu giải thích (luật chữ UI 09/10/2026); DESIGN.md §15, UX.md đã vá. Hộp thư nay hiện «Chưa có câu hỏi cần xử lý» |
| TC-UX05-2 | LT | P-107-2: trạng thái rỗng = tiêu đề ngắn + tối đa một nút; bỏ câu giải thích (luật chữ UI 09/10/2026); DESIGN.md §15, UX.md đã vá. Hộp thư nay hiện «Chưa có câu hỏi cần xử lý» |
| TC-UX05-3 | LT | P-107-2: trạng thái rỗng = tiêu đề ngắn + tối đa một nút; bỏ câu giải thích (luật chữ UI 09/10/2026); DESIGN.md §15, UX.md đã vá. Hộp thư nay hiện «Chưa có câu hỏi cần xử lý» |
| TC-UX05-4 | P | spec w2-real |
| TC-UX05-5 | LT | P-107-2: trạng thái rỗng = tiêu đề ngắn + tối đa một nút; bỏ câu giải thích (luật chữ UI 09/10/2026); DESIGN.md §15, UX.md đã vá. Hộp thư nay hiện «Chưa có câu hỏi cần xử lý» |
| TC-UX05-6 | P | spec w12b-real |
| TC-UX05-7 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX05-8 | P | spec w12b-real |
| TC-UX05-9 | P | spec w2-real |
| TC-UX05-10 | P | spec w2-real |
| TC-UX05-11 | LT | P-107-2: trạng thái rỗng = tiêu đề ngắn + tối đa một nút; bỏ câu giải thích (luật chữ UI 09/10/2026); DESIGN.md §15, UX.md đã vá. Hộp thư nay hiện «Chưa có câu hỏi cần xử lý» |
| TC-UX05-12 | M | chưa chạy (mẫu; dạng MAC, không có test tự động hoặc test bị skip) |
| TC-UX06-1 | P | spec w2-real |
| TC-UX06-2 | F | spec w12b-real: 1 lượt đỏ / 1 xanh (xem logs/) |
| TC-UX06-3 | F | spec w2-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX06-4 | F | spec w2-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX06-5 | F | spec w2-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX06-6 | F | spec w12c-real,w2-real: 1 lượt đỏ / 1 xanh (xem logs/) |
| TC-UX06-7 | F | spec w12c-real,w2-real,w5r-real: 3 lượt đỏ / 2 xanh (xem logs/) |
| TC-UX06-8 | P | spec w2-real |
| TC-UX06-9 | P | spec w2-real |
| TC-UX06-10 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX06-PQ | F | spec w12c-real: 1 lượt đỏ / 1 xanh (xem logs/) |
| TC-UX07-1 | P | spec w2-real |
| TC-UX07-2 | P | spec w2-real |
| TC-UX07-3 | P | spec w2-real |
| TC-UX07-4 | P | spec w2-real |
| TC-UX07-5 | P | spec w2-real |
| TC-UX07-6 | P | spec w2-real |
| TC-UX07-7 | P | spec w2-real |
| TC-UX07-8 | F | còn 1 chuỗi hiển thị «Kiểm duyệt Threads»: frontend/src/features/settings/llm/decisionApi.ts:17 (nhãn cổng THREADS_PRECHECK cho admin, thêm ở sprint 10.5 commit acb704cc) → DEF-R9.1-03 |
| TC-UX07-9 | P | spec w12c-real |
| TC-UX07-10 | P | spec w12b-real |
| TC-UX07-11 | P | spec w2-real |
| TC-UX07-12 | P | spec w12b-real |
| TC-UX07-13 | P | spec w12b-real |
| TC-UX07-14 | P | diff shell.spec.ts của merge 9.1: chỉ đổi nhãn nav + chuyển sang toHaveText tự thử lại; không xoá/nới assertion |
| TC-UX07-15 | F | spec w2-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX07-16 | P | rg DESIGN.md: còn duy nhất dòng 42 «Hỏi đáp công khai (Threads)» là ghi chú đổi tên có chủ ý |
| TC-UX07-17 | M | chưa chạy (mẫu; dạng MAC, không có test tự động hoặc test bị skip) |
| TC-UX07-PQ | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX10-1 | P | spec w3 |
| TC-UX10-2 | P | spec w3 |
| TC-UX10-3 | P | spec w3 |
| TC-UX10-4 | P | spec w3 |
| TC-UX10-5 | P | spec w3 |
| TC-UX10-6 | P | spec w3 |
| TC-UX10-7 | P | spec w3 |
| TC-UX10-8 | P | spec w3 |
| TC-UX10-9 | P | bash scripts/ui-antipatterns.sh trên cây HEAD: exit 0, không ✗ |
| TC-UX10-10 | P | spec w3 |
| TC-UX10-11 | P | spec w3 |
| TC-UX10-12 | P | spec w3 |
| TC-UX10-13 | P | spec w3 |
| TC-UX10-14 | P | spec w3 |
| TC-UX10-15 | P | spec w3 |
| TC-UX10-16 | P | spec w3 |
| TC-UX10-17 | P | spec w3 |
| TC-UX10-18 | P | spec w3 |
| TC-UX10-19 | P | spec w3 |
| TC-UX10-20 | P | spec w3 |
| TC-UX10-21 | P | spec w3 |
| TC-UX10-22 | P | spec w3 |
| TC-UX10-23 | P | spec w3 |
| TC-UX10-24 | P | spec w3 |
| TC-UX10-25 | P | spec w3 |
| TC-UX10-26 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX10-27 | P | spec w3 |
| TC-UX10-28 | M | chưa chạy (mẫu; dạng MAC, không có test tự động hoặc test bị skip) |
| TC-UX10-PQ | P | spec w3 |
| TC-UX11-1 | P | spec w3 |
| TC-UX11-2 | P | spec w3 |
| TC-UX11-3 | P | spec w3 |
| TC-UX11-4 | P | spec w3 |
| TC-UX11-5 | P | spec w3 |
| TC-UX11-6 | P | spec w3 |
| TC-UX11-7 | P | spec w3 |
| TC-UX11-8 | P | spec w3 |
| TC-UX11-9 | P | spec w3 |
| TC-UX11-10 | M | chưa chạy (mẫu; dạng MAC, không có test tự động hoặc test bị skip) |
| TC-UX11-PQ | P | spec w3 |
| TC-UX14-1 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX14-2 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX14-3 | F | spec w4b-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX14-4 | F | spec w4-real: 1 lượt đỏ / 1 xanh (xem logs/) |
| TC-UX14-5 | F | spec w4-real,w4b-real: 2 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX14-6 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX14-7 | F | spec w4-real,w4b-real: 2 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX14-8 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX14-9 | F | spec w4-real: 2 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX14-10 | F | spec w4-real: 2 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX14-11 | F | spec w4-real,w4b-real: 3 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX14-12 | M | chưa chạy (mẫu; dạng MAC, không có test tự động hoặc test bị skip) |
| TC-UX14-PQ | F | spec w4-real: 2 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX14-PG | F | spec w4-real: 2 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX15-1 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX15-2 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX15-3 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX15-4 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX15-5 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX15-6 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX15-7 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX15-8 | F | spec w4-real,w4b-real: 2 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX15-9 | F | spec w4-real,w4b-real: 2 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX15-10 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX15-11 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX15-12 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX15-13 | F | spec w4-real,w4b-real,w4c-real: 3 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX15-14 | F | spec w4c-real,w4g-real: 2 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX15-15 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX15-16 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX15-17 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX15-18 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX15-19 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX15-20 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX15-21 | M | chưa chạy (mẫu; dạng MAC, không có test tự động hoặc test bị skip) |
| TC-UX15-PQ | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX16-1 | P | spec w4,w4f-real |
| TC-UX16-2 | P | spec w4-real |
| TC-UX16-3 | P | spec w4-real |
| TC-UX16-4 | P | spec w4-real |
| TC-UX16-5 | F | spec w4-real,w4f-real: 1 lượt đỏ / 2 xanh (xem logs/) |
| TC-UX16-6 | P | spec w4-real |
| TC-UX16-7 | P | spec w4-real |
| TC-UX16-8 | P | spec w4 |
| TC-UX16-9 | P | spec w4,w4-real |
| TC-UX16-10 | P | spec w4,w4-real |
| TC-UX16-11 | P | spec w4,w4-real |
| TC-UX16-12 | P | spec w4g-real |
| TC-UX16-13 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX16-14 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX16-15 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX16-16 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX16-17 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX16-18 | P | spec w4 |
| TC-UX16-19 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX16-20 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX16-21 | P | spec w4-real,w4f-real |
| TC-UX16-22 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX16-23 | M | chưa chạy (mẫu; dạng MAC, không có test tự động hoặc test bị skip) |
| TC-UX16-24 | F | spec w4d-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX12-1 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX12-2 | F | spec w4-real,w4b-real: 1 lượt đỏ / 1 xanh (xem logs/) |
| TC-UX12-3 | F | spec w4-real,w4b-real: 1 lượt đỏ / 1 xanh (xem logs/) |
| TC-UX12-4 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX12-5 | F | spec w4-real,w4b-real: 1 lượt đỏ / 1 xanh (xem logs/) |
| TC-UX12-6 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX12-7 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX12-8 | P | rg "0[,.]25" frontend/src/features/attendance: 0 kết quả |
| TC-UX12-9 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX12-10 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX12-11 | F | spec w4-real,w4f-real: 1 lượt đỏ / 1 xanh (xem logs/) |
| TC-UX12-12 | P | spec w4-real |
| TC-UX12-13 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX12-14 | F | spec w4-real,w4b-real,w4c-real,w4f-real: 1 lượt đỏ / 3 xanh (xem logs/) |
| TC-UX12-15 | F | spec w4-real,w4b-real,w4c-real,w4f-real: 1 lượt đỏ / 3 xanh (xem logs/) |
| TC-UX12-16 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX12-17 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX12-18 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX12-19 | F | spec w4-real,w4f-real: 1 lượt đỏ / 1 xanh (xem logs/) |
| TC-UX12-20 | F | spec w4-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX12-21 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX12-22 | M | chưa chạy (mẫu; dạng MAC, không có test tự động hoặc test bị skip) |
| TC-UX12-PQ | P | spec w4-real |
| TC-UX13-1 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX13-2 | M | chưa chạy (mẫu; dạng GREP + E2E, không có test tự động hoặc test bị skip) |
| TC-UX13-3 | P | spec w5i-real |
| TC-UX13-4 | P | spec w5i-real |
| TC-UX13-5 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX13-6 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX13-7 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX13-8 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX13-9 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX13-10 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX13-11 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX13-12 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX13-13 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX13-14 | F | spec w5a-real: 1 lượt đỏ / 1 xanh (xem logs/) |
| TC-UX13-15 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX13-16 | M | chưa chạy (mẫu; dạng GREP, không có test tự động hoặc test bị skip) |
| TC-UX13-17 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX13-18 | F | spec w5a-real,w5g-real: 1 lượt đỏ / 2 xanh (xem logs/) |
| TC-UX13-19 | P | spec w5c-real |
| TC-UX13-20 | P | spec w5c-real,w5g-real |
| TC-UX13-21 | F | spec w5a-real: 1 lượt đỏ / 1 xanh (xem logs/) |
| TC-UX13-22 | F | spec w5a-real,w5c-real,w5g-real: 1 lượt đỏ / 3 xanh (xem logs/) |
| TC-UX13-23 | F | spec w5a-real,w5c-real,w5g-real: 1 lượt đỏ / 3 xanh (xem logs/) |
| TC-UX13-24 | F | spec w5a-real: 1 lượt đỏ / 1 xanh (xem logs/) |
| TC-UX13-25 | F | spec w5a-real: 1 lượt đỏ / 2 xanh (xem logs/) |
| TC-UX13-26 | P | spec w5a-real |
| TC-UX13-27 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX13-28 | M | chưa chạy (mẫu; dạng MAC, không có test tự động hoặc test bị skip) |
| TC-UX19-1 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX19-2 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX19-3 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX19-4 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX19-5 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX19-6 | F | spec w5f-real: 1 lượt đỏ / 1 xanh (xem logs/) |
| TC-UX19-7 | P | spec w5e-real |
| TC-UX19-8 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX19-9 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX19-10 | P | spec w5i-real |
| TC-UX19-11 | P | spec w5f-real |
| TC-UX19-12 | F | spec w5f-real: 1 lượt đỏ / 1 xanh (xem logs/) |
| TC-UX19-13 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX19-14 | M | chưa chạy (mẫu; dạng MAC, không có test tự động hoặc test bị skip) |
| TC-UX19-PG | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX20-1 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX20-2 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX20-3 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX20-4 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX20-5 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX20-6 | P | spec w5g-real |
| TC-UX20-7 | P | spec w5g-real |
| TC-UX20-8 | P | spec w5i-real |
| TC-UX20-9 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX20-10 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX20-11 | M | chưa chạy (mẫu; dạng MAC, không có test tự động hoặc test bị skip) |
| TC-UX18-1 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX18-2 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX18-3 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX18-4 | P | spec w5g-real |
| TC-UX18-5 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX18-6 | P | spec w5g-real |
| TC-UX18-7 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX18-8 | P | spec w5g-real |
| TC-UX18-9 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX18-10 | M | chưa chạy (mẫu; dạng MAC, không có test tự động hoặc test bị skip) |
| TC-UX18-PQ | P | spec w5g-real |
| TC-UX18-ID | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX17-1 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX17-2 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX17-3 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX17-4 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX17-5 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX17-6 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX17-7 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX17-8 | P | spec w5g-real |
| TC-UX17-9 | P | spec w5g-real |
| TC-UX17-10 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX17-11 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX17-12 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX17-13 | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX17-14 | M | chưa chạy (mẫu; dạng MAC, không có test tự động hoặc test bị skip) |
| TC-UX17-PQ | M | chưa chạy (mẫu; dạng E2E, không có test tự động hoặc test bị skip) |
| TC-UX91-X1 | P | như UX08-37 (AC10a US-08) |
| TC-UX91-X2 | M | chưa chạy (mẫu; dạng E2E + LH, không có test tự động hoặc test bị skip) |
| TC-UX91-X3 | P | git diff --stat 2c15dfce^1 2c15dfce: không có backend-go/db/migrations mới; không `fetch(` trần mới ngoài shared/api; Demo*.tsx chỉ đổi như UX01-6 |
| TC-UX91-X4 | P | spec w5h-real |
| TC-UX08-1 | LT | AC1: h1 đổi còn 8 từ «Vận hành lớp học có trợ lý AI», sáu mục mới (P-107-9, P-107-11). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (l |
| TC-UX08-2 | LT | AC1: danh sách h2 đổi theo TC-UX08-1 (P-107-9, P-107-11). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-3 | LT | AC2: màn đầu đúng hai nút «Đăng nhập» + «Xem video» (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-4 | LT | AC2: như UX08-3 (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-5 | F | luật :hover màu (border-color/color/background) ở Landing.module.css:35,39,40,73,97,144,226 nằm ngoài @media (hover: hover) and (pointer: fine) → hover dính trên cảm ứng; xem DEF-R9.1-02 |
| TC-UX08-6 | LT | AC3 + US-09 AC2: clip/poster màn đầu bị bỏ, thay bằng đoạn sản phẩm HTML (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (log |
| TC-UX08-7 | LT | AC3 + US-09 AC2: không còn clip để đo (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-8 | LT | AC3 + US-09 AC2: không còn clip (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-9 | LT | AC3/AC14: không còn clip, reduce theo AC14 mới (P-107-12, P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-10 | LT | AC3/AC14: không còn clip (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-11 | LT | AC3: không còn clip (tab ẩn) (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-12 | LT | AC3 + AC19: không còn clip tải lỗi (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-13 | LT | AC3: màn đầu không còn clip/poster; đoạn sản phẩm HTML (P-107-13) |
| TC-UX08-14 | LT | AC4 mới: nút Tạm dừng/Phát 32 px, vùng chạm ≥ 44 px hoặc vùng đệm (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devsp |
| TC-UX08-15 | LT | AC4 mới (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-16 | P | spec w6-real |
| TC-UX08-17 | P | spec w6-real |
| TC-UX08-18 | P | spec w6-real |
| TC-UX08-19 | P | spec w6-real |
| TC-UX08-20 | LT | AC5/AC13 sửa 10.7: ngưỡng «chênh ≤ 1,5 KB so với v1 216 KB» bị thay bằng tổng ≤ 256 KB và script riêng ≤ 5 KB (P-107-12/13); số đo hiện hành ở UX08-44 |
| TC-UX08-21 | P | spec w6-real |
| TC-UX08-22 | LT | AC6: khung ghim đổi: ba tên màn ngang + ảnh rộng hết khung (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log |
| TC-UX08-23 | LT | AC6 mới (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-24 | LT | AC6: ảnh khung ghim rộng 1280 px, không còn giới hạn 92.160 B (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs. |
| TC-UX08-25 | LT | AC6/AC12: hoạt cảnh khung ghim theo techlead 8.3 (P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-26 | LT | AC7 đổi theo khung ghim mới (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-27 | LT | AC7 đổi theo khung ghim mới (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-28 | F | spec w6-real: 2 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX08-29 | LT | AC8: ba bước dùng IntersectionObserver thay view-timeline (P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-30 | LT | AC8 mới (P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-31 | LT | AC8/AC9: số đếm do script (P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-32 | LT | AC8/AC9 mới (P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-33 | LT | AC9: tiêu đề khối «Ba số đo trên dữ liệu mô phỏng», bỏ dòng chú thích (P-107-11; BA 14/11). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/ |
| TC-UX08-34 | LT | AC9 mới: bỏ chú thích riêng (P-107-11). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-35 | LT | AC10: hai cột vai thành hai tab (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-36 | LT | AC10 mới (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-37 | P | AC10a: story-chat/review/gradebook-v1-1280.webp chụp lại ở commit aa4343f8 (15/11, 10.7 lô 8–9) sau đợt 3–5; xem ảnh story-review: có «Đã duyệt 0/4», phím A, «Duyệt bài», nhãn nav mới (Hỏi đáp công khai, Lỗ hổng kiến thứ |
| TC-UX08-38 | LT | AC11 sửa: chữ thân ≥ 16 px, dẫn ≥ 18 px (P-107-9). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-39 | LT | AC11: bỏ điều kiện «một h2 mỗi khung nhìn» (P-91-8 (b)) và khoảng cách đổi theo sáu mục 100svh (P-107-9). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landi |
| TC-UX08-40 | LT | AC12: vi mô khi rê đổi (nhích 1 px, mũi tên 3 px; P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-41 | LT | AC12 mới (P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-42 | LT | AC12 mới (P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-43 | P | Lighthouse mobile 3 lượt: LCP 2419/2190/2087 ms, CLS 0; desktop LCP 437 ms (lượt 1 lạnh 2177 ms), CLS 0; a11y 1,0. Máy chủ ép tải ~40 nên số mobile sát trần; ghi nhận p.lead là LCP ở 375 (DEF-R9.1-05) |
| TC-UX08-44 | P | logs/r91land.log: JS tĩnh route / = 214,5 KB gzip (11 tệp) ≤ 256 KB; script nội tuyến tự viết 0,48 KB gzip (≤ 5 KB, AC13) |
| TC-UX08-45 | P | git diff package.json merge 9.1: chỉ bỏ framer-motion/embla; curl / (không JS) có h1 «Vận hành lớp học có trợ lý AI» và nội dung chính (Server Component) |
| TC-UX08-46 | LT | AC14 mới: reduce → không gắn m-on (P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-47 | LT | AC13/AC15: không còn animation-timeline trong Landing.module.css; thay bằng IntersectionObserver (P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng la |
| TC-UX08-48 | F | spec w6-real: 3 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX08-49 | LT | AC15: nay có đúng MỘT IntersectionObserver (LandingMotion.tsx:143) thay P-91-2 «không IntersectionObserver» (P-107-12). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy x |
| TC-UX08-50 | F | spec w6-real: 2 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX08-51 | P | spec w6-real |
| TC-UX08-52 | P | spec w6-real |
| TC-UX08-53 | LT | AC19: lỗi clip không còn; ảnh khung giữ kích thước theo AC19 mới (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspe |
| TC-UX08-54 | P | spec w6-real |
| TC-UX08-55 | LT | AC21 sửa 10.7: landing.spec.ts nay kiểm đoạn sản phẩm/Tạm dừng/Xem video/ghim/hai tab; bản mới 21/21 xanh (logs/devspecs.log); TC cũ ghi «hai cột vai (không tab)», «clip không tải khi reduce» (P-107-12/13) |
| TC-UX08-56 | LT | AC21 mới: ảnh mốc landing-1440/390 anonymous reduce (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX08-PQ | P | logs/r91land.log: khách mở /chat, /gradebook → /login?next=…; HTML `/` không có @edupilot.local, Edupilot#, Bearer/JWT, sk-…, MSSV 2022xxxx |
| TC-UX09-1 | P | ffprobe gioi-thieu-v1.mp4: duration 64,73 s (60–90) |
| TC-UX09-2 | P | ffprobe: h264 1440×900, không luồng audio (cả .mp4 và .webm vp9 1440×900) |
| TC-UX09-3 | P | ffmpeg trích 65 khung (1 khung/s), 22 khung đã xem qua ảnh ghép frames/sheet-1..3.png: thẻ tiêu đề → chat riêng (có che tên: «Đã ẩn 3 thông tin cá nhân trước khi gửi cho AI») → Hộp thư hỗ trợ → Chấm bài + duyệt bài → Sổ  |
| TC-UX09-4 | LT | US-09 AC2 «clip màn đầu — bỏ»: gioi-thieu-clip-v1.* và poster đã xoá khỏi public/landing (P-107-13); ls public/landing không còn tệp clip |
| TC-UX09-5 | LT | US-09 AC2: không còn clip (P-107-13) |
| TC-UX09-6 | LT | US-09 AC2: không còn clip để kiểm faststart (P-107-13); video đầy đủ vẫn có moov trước mdat (ftyp,moov,free,mdat), h264 |
| TC-UX09-7 | P | public/landing/: gioi-thieu-v1.mp4 1,06 MB, .webm 1,56 MB, .webp 51,7 KB (< 8 MB); curl -I https://localhost:1143/landing/gioi-thieu-v1.mp4 → 200, accept-ranges: bytes |
| TC-UX09-8 | P | logs/r91land.log: <video> trong hộp thoại có controls, preload=none, playsinline, poster=/landing/gioi-thieu-v1.webp, không autoplay; 0 request video trước khi bấm «Xem video» |
| TC-UX09-9 | P | Lighthouse mobile devtools-throttle 3 lượt LCP 2419/2190/2087 ms (≤ 2500), CLS 0, desktop 437 ms (+ 1 lượt lạnh 2177 ms khi máy tải 40): logs/lh-*.json; poster không còn là ứng viên LCP (US-09 AC5 sửa 10.7); ghi nhận: ở  |
| TC-UX09-10 | P | spec w6-real |
| TC-UX09-11 | F | spec w6-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX09-12 | P | curl -H 'Range: bytes=0-99': mp4 và webm đều 206, accept-ranges: bytes, content-range: bytes 0-99/… |
| TC-UX09-13 | F | spec w6-real: 1 lượt đỏ / 0 xanh (xem logs/) |
| TC-UX09-14 | P | scripts/landing-video/record.mjs + README: chỉ đọc; không gửi chat, không Duyệt/Công bố/sửa điểm (rg Duyệt/Công bố/Gửi/type(/Enter = 0) |
| TC-UX09-15 | P | tesseract OCR 65 khung (frames/ocr.txt, logs): không có email, localhost, .local, IP, 9router, khoá; MSSV 2022xxxx là dữ liệu seed mô phỏng (D44) |
| TC-UX09-16 | M | chưa chạy (mẫu; dạng FFP, không có test tự động hoặc test bị skip) |
| TC-UX09-17 | LT | US-09 AC2/AC5: không còn clip màn đầu; video đầy đủ chỉ mở bằng «Xem video» (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 ( |
| TC-UX09-18 | LT | US-09 AC2/AC5 (P-107-13). Spec đã sửa ở 10.7: docs/specs/FEAT-uiux-91/US.md US-08 «Sửa 10.7»; P-107-9/11/12/13. Bản mới chạy xanh bằng landing.spec.ts 21/21 (logs/devspecs.log) |
| TC-UX09-19 | P | spec w6-real |
| TC-UX09-20 | LT | US-09 AC2/AC13: không còn clip lặp để xem (P-107-13) |
| TC-UX09-PQ | P | /landing/../package.json, /landing/%2e%2e/package.json, /landing/..%2f..%2fnext.config.ts, /edupilot/x.mp4: đều trả khung ứng dụng (HTML), không có nội dung tệp riêng; không tới MinIO |

## Trượt xác nhận (mã) — xem defects-9.1.md
UX04-3, UX08-5 (hover ngoài media), UX07-8 (chữ «Threads»), UX08-50 (axe, Thấp, nghi artefact hiện-theo-cuộn).

## Ca chưa chạy (M)
97 ca: UX13/17/18/19 và ca lẻ còn thiếu assertion; UX09-16 (run.sh docker), UX16-17 (cần lớp thử), UX16-24 (cần dữ liệu sạch). Chạy tiếp khi tải máy < 15.

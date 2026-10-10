# Sprint 1.5 — báo cáo (prototype giao diện toàn bộ tính năng)

Mục tiêu: prototype bấm được, đủ mọi route dự định, đổi 4 vai, đi trọn kịch bản demo 15 phút, để trình bày với thầy hướng dẫn (D51) · Kết quả: **5/5 story PASS · DEMO PASS** sau 2 vòng QC và 2 vòng sửa · Nhánh `sprint/1.5-mock-ui`

| Story | Phạm vi | Trạng thái | QC |
| --- | --- | --- | --- |
| US-PROTO-00 | Nền: `/login` chọn vai, shell, bộ chọn lớp, chuông, phân quyền theo vai, trạng thái tải/rỗng/lỗi, màn lỗi/404 trong khung | PASS | `qc/report-v5-US-PROTO-00.md`, `qc/report-v5-round2.md` |
| US-PROTO-01 | Sinh viên: Hôm nay, Chat riêng, Threads, Luyện đề, Thư viện, Lịch, Kết quả, Bài tập, Tham gia lớp | PASS (sau vòng sửa 2: BUG-v5-01-6) | như trên |
| US-PROTO-02 | GV/TA vận hành lớp: Hôm nay, Hộp thư, Sinh viên, Điểm danh, Thành viên | PASS | như trên |
| US-PROTO-03 | Đánh giá: Sổ điểm, Công thức điểm, Chấm bài, Ngân hàng câu hỏi, Tài liệu | PASS | như trên |
| US-PROTO-04 | Hiểu lớp + Hệ thống: Insights, Analytics, Quan sát AI, Cấu hình LLM, Tích hợp, Quản trị | PASS | như trên |
| DEMO | Kịch bản 15 phút F2→F3→F5→F7→F9→F10→F17 trong một trình duyệt | PASS — `demo-run.mjs` 22/22 nhóm bước, 170 s | `qc/report-v5-DEMO.md` |

## Diễn biến
1. **v1–v4** (01/10): 5 story theo spec v1–v4, QC ALL PASS. Chủ dự án xem và không ưng: luồng Threads sai so với hệ cũ (#15), panel không phân tách (#16) → spec v4, sửa, PASS.
2. **v5** (01/10, chủ dự án giao PM toàn quyền nghiệm thu UI): PM rà 36 ảnh → #17 (logo nhỏ, bộ chọn lớp cắt chữ, inbox tràn, chat lệch trục, menu hồ sơ ghi tên vai, lưới `/settings/llm`). Chủ dự án: "Threads chưa ổn, chưa thấy mock phản hồi thật" → #18 Threads như thật (seed phản hồi khớp số đếm, 10 mẫu AI Socratic theo chủ đề + nhánh "chưa đủ chắc chắn", đang soạn → stream → nguồn, phản hồi trễ của TA + chuông, hộp chặn thông tin cá nhân).
3. **QC thăm dò** tìm thêm 37 lỗi ngoài spec (#19, 8 mức cao: sập trang luyện đề, SV chưa vào lớp thấy dữ liệu người khác, thread đăng email/SĐT không chặn…) → spec v5.1 có **một nguồn số liệu** cho mọi con số (`mock/derive.ts`).
4. **QC chạy v5**: ~50 lỗi (gồm 4 lỗi UI PM thấy, #24) → dev sửa vòng 1 49/50 (lỗi còn lại là spec tự mâu thuẫn, đóng theo #26, spec v5.2).
5. **QC vòng 2**: mọi lỗi mức cao PASS; còn 1 lỗi thấp (nhãn lịch tháng 3 dòng) → dev sửa vòng 2.

## Số liệu
- Test case: **443** (00: 69 · 01: 153 · 02: 91 · 03: 41 · 04: 50 · DEMO: 39). (Sửa 2026-10-10: bản trước ghi 449 vì đếm cả dòng tiêu đề bảng `TC-id`.)
- Tự động vòng 2: `proto-curl.sh` 497/0 · `audit.mjs` 495 dòng (4 vai × 1440/390/375 + bề rộng biên) 0 FAIL · `regress-v24` 30/30 · `pii-matrix` 48/48 · `threads-timeline` 0 FAIL · `demo-run` 22/22.
- Lint, build, `ui-antipatterns.sh` sạch. 27 nhóm route, `frontend/src` +19.008 dòng, 90 commit trên nhánh.
- Góp ý của đội: #15–#26 (12), PM chấp nhận 12 (1 theo phương án khác: #21).

## Quy trình rút ra (đã ghi vào `docs/team/`)
- Cấm subagent khi PM chưa cho phép (tốn token gấp nhiều lần) — `CONTEXT.md`.
- Thêm vai **Research** (`RESEARCH.md`, pane `research`).
- Chạy cuốn chiếu: PM lập kế hoạch sprint kế trong lúc QC test — `PM.md`.

## Nợ
- BUG-v5-03-9, 03-10, DEMO-9, 04-6 chỉ có bằng chứng máy đo, chưa đo tay riêng (không có TC FAIL).
- Chrome headless trên máy dev treo khi màn hình ngủ → chạy QC kèm `caffeinate` hoặc `--disable-gpu --use-angle=swiftshader`.
- Prototype là mock có hạn dùng (D51): mỗi sprint build thật thay dần màn mock.

## Chủ dự án tự kiểm
`pnpm -C frontend build && node frontend/.next/standalone/frontend/server.js` (cổng 3000) hoặc `pnpm dev`; `/login` chọn vai, menu hồ sơ → `Đổi vai`, `Đặt lại dữ liệu demo`. Đi theo `docs/DEMO_SCRIPT.md`. Xem kỹ Threads với SV B (`/threads/t-cbc`, tạo thread mới có/không khớp chủ đề, gửi phản hồi chờ TA trả lời).

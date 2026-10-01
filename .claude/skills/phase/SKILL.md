---
name: phase
description: Bắt đầu hoặc tiếp tục thi công một phase của EduPilot v2 theo docs/phases. Dùng khi chủ dự án gõ /phase <MÃ PHASE> [lát việc].
argument-hint: "<P0|PG|PU|P1..P10|PR> [L1..L5]"
disable-model-invocation: true
---

Thi công phase: $ARGUMENTS

1. Đọc `docs/PROGRESS.md`, rồi `docs/phases/$0.md`. Chỉ đọc phần liên quan của `docs/ARCHITECTURE.md` và `docs/PRD.md` khi cần. Nếu lát việc thuộc một luồng trong `docs/FLOWS.md`: đọc luồng đó, kể cả nhánh lỗi, và nêu trong kế hoạch nhánh nào sẽ được xử lý ở lát này. Nếu lát việc có giao diện: đọc `docs/design/DESIGN.md` (§10, §12, §13, mục route tương ứng ở §14, §21, §22) và `docs/UX.md` TRƯỚC khi lập kế hoạch; trong kế hoạch nêu rõ primitive nào ở `shared/ui/` sẽ dùng và khung nhìn đầu của màn gồm những gì.
2. Nếu có tham số lát việc ($1) thì chỉ làm lát đó; nếu không, làm lát đầu tiên chưa xong theo `PROGRESS.md`. KHÔNG làm quá một lát trong một phiên.
3. Trước khi sửa gì: chạy bộ test hiện có liên quan và ghi lại kết quả làm mốc.
4. Trình bày KẾ HOẠCH trước: file sẽ tạo/sửa, migration, API, test sẽ viết, rủi ro. DỪNG chờ chủ dự án duyệt. Không viết code trước khi được duyệt.
5. Thi công theo thứ tự: migration → sqlc/store → service + handler Go → frontend → test → seed. Mỗi bước một commit `$0: <việc>`.
6. Tuân thủ 9 nguyên tắc và mục Cấm trong `CLAUDE.md`. Gặp điều kiện DỪNG thì dừng và hỏi.
7. Kết thúc lát: chạy test liên quan, tick các ô đã xong trong phase file, cập nhật `docs/PROGRESS.md` (Phiên gần nhất, Nợ), rồi báo cáo: đã làm gì, file đổi, test nào xanh/đỏ, bước kế tiếp.

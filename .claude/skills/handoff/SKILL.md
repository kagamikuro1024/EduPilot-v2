---
name: handoff
description: Kết phiên làm việc EduPilot v2, ghi lại trạng thái để phiên sau tiếp tục không mất ngữ cảnh. Dùng khi chủ dự án gõ /handoff.
disable-model-invocation: true
---

Kết phiên.

1. `git status` và `git diff --stat`: liệt kê thay đổi chưa commit; commit nếu là việc đã xong, nêu rõ nếu còn dở.
2. Cập nhật `docs/PROGRESS.md`: Đang ở đâu; Phiên gần nhất (đã làm, đang dở kèm tên file/hàm/test đang đỏ, bước kế tiếp CỤ THỂ đến mức phiên sau làm được ngay); Nợ; Ánh xạ migration nếu có migration mới.
3. Tick các ô đã xong trong phase file.
4. Ghi 3–6 gạch đầu dòng vào `docs/thesis-notes/<phase>.md`: quyết định kỹ thuật đáng viết vào luận văn, số liệu đo được, khó khăn và cách giải quyết.
5. Commit `docs: handoff <phase> <lát>`.

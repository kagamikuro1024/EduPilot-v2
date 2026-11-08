# Prompt cho `research` (Tech Lead): khả thi kỹ thuật (sprint 9.1, pha 1)

Khi BA push `docs/sprints/9.1/audit/findings.md`, đọc nó + `docs/sprints/9.1/plan.md`. Viết `docs/sprints/9.1/techlead.md` mục TL-1: với từng đề xuất mức Cao / TB và mỗi mục [ĐỔI DESIGN], đánh giá:
- khả thi với stack hiện có (Next.js 16, token `--ep-*`, primitive `shared/`), không thêm thư viện;
- rủi ro hiệu năng (LCP, TBT, JS 256 KB, video), a11y, ảnh mốc phải sinh lại;
- ước lượng nhỏ / vừa / lớn;
- đề xuất nào nên gộp / bỏ.

Kèm ý kiến về cách dựng video giới thiệu (Playwright + ffmpeg trên VPS) và chuyển động không thư viện. Push `sprint/9.1-uiux`, báo PM ≤ 5 dòng.

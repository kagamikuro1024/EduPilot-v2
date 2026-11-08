# Prompt cho `ba`: khảo sát UI/UX chuyên môn (sprint 9.1, pha 1)

Đọc:
- `docs/sprints/9.1/plan.md`;
- `docs/design/DESIGN.md` (đặc biệt §13 lời văn, §14 hợp đồng từng route, §21 phản mẫu, §22 định nghĩa xong);
- `docs/UX.md`;
- luật chữ UI trong `~/.claude/CLAUDE.md`.

Đọc các skill trước khi đánh giá: `skill://impeccable` (lệnh `critique` + `audit`), `skill://design-taste-frontend`, `skill://emil-design-eng`, `skill://review-animations`, `skill://design`. Bằng chứng: `docs/sprints/9.1/audit/` do QC chụp (pull khi QC push); tự đi thêm bản demo bằng `playwright-cli` (trình duyệt trên Mac) khi cần.

## Việc → `docs/sprints/9.1/audit/findings.md`
1. **Điểm tổng quan:**
   - nhất quán hệ thiết kế;
   - phân cấp thị giác;
   - mật độ thông tin;
   - lời văn;
   - trạng thái;
   - di động;
   - a11y;
   - chuyển động;
   - "cảm giác hiện đại / đáng tin".

   Mỗi mục 1–5 kèm 1 câu lý do.
2. **Bảng vấn đề:**
   - **Cột:** ID `UX-n`, màn / route, vai, mức (Cao / TB / Thấp), mô tả ngắn, bằng chứng (đường dẫn ảnh), nguyên tắc bị vi phạm (DESIGN §… / UX.md / heuristic / skill), đề xuất cụ thể.
   - **Đề xuất phải làm được bằng token + primitive có sẵn;** nếu cần đổi `DESIGN.md` thì đánh dấu **[ĐỔI DESIGN]**.
3. **Mẫu đề xuất cho các màn chính:**
   - các màn: trang giới thiệu (bắt buộc, theo yêu cầu "hiện đại, animation, video"), Hôm nay (giảng viên + sinh viên), chat riêng, màn duyệt bài, sổ điểm;
   - mỗi màn: vấn đề, hướng mới (bố cục, phân cấp, chuyển động), phác thảo bằng chữ / ASCII đủ để dev dựng mẫu.
4. **Chuyển động:** liệt kê chỗ nên có / nên bỏ (theo `emil-design-eng`, `review-animations`), thời lượng, easing, `prefers-reduced-motion`.
5. **Ưu tiên:** "làm ngay" (sửa nhỏ, ảnh hưởng lớn), "làm trong 9.1", "để sau".

Không sửa mã, không sửa `DESIGN.md`. Commit `docs/sprints/9.1/audit/findings.md`, push `sprint/9.1-uiux`, báo PM ≤ 5 dòng.

# Prompt cho `dev` — Thi công spec v5 (UI polish + Threads như thật, góp ý #17, #18)

Spec `docs/sprints/1.5/spec/US.md` + `SRS.md` **v5 đã APPROVED**. Chủ dự án chưa ưng UI (logo nhỏ…) và nói luồng Threads "chưa ổn một tí nào, chưa thấy mock phản hồi thật". Chủ dự án dặn: **làm thật kỹ**.

## Đọc
`docs/team/CONTEXT.md`, `docs/team/DEV.md`, góp ý #17 #18 ở `docs/sprints/1.5/proposals.md`, ảnh `docs/sprints/1.5/shots/pm-review-v4/` và `shots/ba-review-v5/`, rồi spec v5:
- `SRS.md` 4.7 (FR-X12, đoạn `AUDIT` đo bố cục, lỗi h1–h4), 4.3.1 (kịch bản Threads A–F).
- `US.md`: 00-AC7…AC10, 01-AC4, 01-AC10…AC14, 02-AC9…AC11, 04-AC7.
- `docs/design/DESIGN.md` phần brand/shell, §21, §22.

## Thứ tự (mỗi mục một commit `US-PROTO-0N: … (v5)`)
1. Shell: logo 32 px + vùng brand 56 px thẳng topbar, logo `/login`, bộ chọn lớp đủ tên + tooltip, menu hồ sơ tên người / vai (00-AC7…AC9).
2. `DataTable` chế độ danh sách < 720 px, `/gradebook` cuộn ngang cột tên dính, tab cuộn ngang, chú thích phím tắt `/attendance` ẩn < 720 px — mọi lỗi h1–h4 (00-AC10).
3. `/inbox` panel danh sách + chi tiết, 390 px danh sách → chi tiết có nút quay lại (02-AC9, 02-AC10).
4. `/chat` desktop: panel phiên 240 px, hội thoại + composer cùng trục, composer ghim đáy (01-AC11).
5. `/settings/llm` lưới cột cố định (04-AC7).
6. **Threads như thật** theo đúng `SRS.md` 4.3.1: seed phản hồi khớp số đếm, 10 mẫu AI theo chủ đề + quy tắc từ khoá + nhánh "chưa đủ chắc chắn", trình tự đang soạn → stream (nút Dừng) → nguồn → Chờ xác nhận, "Hỏi trợ lý AI" + `@AI`, phản hồi trễ của TA + chuông (sống qua chuyển trang / tải lại), "Hôm nay" GV/TA 6 → 7, một tiêu đề "Thảo luận (n)" (01-AC4, 01-AC10, 01-AC12…14, 02-AC11). Gắn `data-part` như spec yêu cầu.

## Luật
- Chỉ token `--ep-*` và primitive `frontend/src/shared/`; sửa primitive dùng chung thay vì vá từng trang.
- Không đổi số liệu #12/#13 hay luồng v4 ngoài phần Threads.
- Thấy spec sai/thiếu: ghi góp ý vào `docs/sprints/1.5/proposals.md` (cột Quyết định PM để trống) rồi làm tiếp phần khác; không tự đổi spec, không hỏi QC/BA.
- Mỗi shell `source ~/.zprofile`. Commit đúng đường dẫn (`git commit -- <paths>`), không `git add -A`, không đụng `scripts/team-up.sh`.

## Xong khi
- Chạy đoạn `AUDIT` (SRS 4.7) trên **mọi route mọi vai** ở 1440 / 390 (route SV, `/inbox`, `/attendance` thêm 375): `{ ox: 0, cut: [], ell: [] }`. Dán kết quả vào handoff.
- `pnpm -C frontend lint && pnpm -C frontend build && bash scripts/ui-antipatterns.sh` xanh.
- Tự đi lại luồng Threads 01-AC10…14 và 02-AC11 trên trình duyệt, chụp ảnh từng bước vào `docs/sprints/1.5/shots/v5/`.
- Cập nhật handoff `docs/sprints/1.5/handoff/dev-US-PROTO-0{0..4}.md` (mục v5: đã làm gì, AC nào, cách tự kiểm, kết quả `AUDIT`). Push nhánh.
Trả lời tóm tắt và dừng.

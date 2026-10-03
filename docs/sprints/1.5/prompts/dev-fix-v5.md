# Prompt cho `dev` — Sửa lỗi QC vòng 1 (sprint 1.5 v5) — vòng sửa 1/2

QC chạy v5 (commit `4acb6cc`): report `docs/sprints/1.5/qc/report-v5-*.md`, ảnh `docs/sprints/1.5/shots/qc-v5/`. Làm **sau khi** commit xong cổng PG sprint 2 (hoặc commit dở ở điểm an toàn), trong repo chính, nhánh `sprint/1.5-mock-ui`.

## Phải sửa (theo thứ tự)
1. **Chặn demo**: BUG-v5-DEMO-1 (công tắc "Giả lập mất mạng" ở `/attendance` làm mất trạng thái đã chốt, QT B tụt 8,5→8,3).
2. **#24 (a)–(d)**: tab `/inbox` 1440 co thành vạch 3 px; nút hồ sơ GV/TA/Admin phải là tên người; chip chủ đề `/threads` cắt mép (cuộn ngang có dấu hiệu hoặc xuống dòng); ô chọn Buổi 390/375 cắt chữ.
3. Mức cao: BUG-v5-02-1 (badge Chấm bài + Hôm nay kẹt 3 sau khi duyệt hết), BUG-v5-02-2 / DEMO-3 (link thẻ thiếu `course=`), BUG-v5-01-1 (`/threads` 375/390 nở 772 px), BUG-v5-00-1 (`ep_demo_state` hỏng → mọi route sự cố: phải tự xoá khoá hỏng và chạy tiếp), BUG-v5-03-1 / DEMO-5 (ô làm tròn chỉ giữ 1 ký tự), BUG-v5-03-2 (`/documents` lớp 2 lẫn quy chế lớp 1), BUG-v5-03-6 (công bố áp lên lựa chọn bị ẩn), BUG-v5-04-2 (`/analytics` 7→30 ngày lệch số).
4. Toàn bộ lỗi mức vừa / thấp còn lại trong các report v5.

## Luật
- Sửa gốc trong primitive / nguồn số liệu chung (`mock/derive.ts`), không vá từng màn.
- Mỗi nhóm một commit `US-PROTO-0N: fix BUG-… (v5 vòng 1)`; cập nhật handoff (bảng BUG → đã sửa → cách tự kiểm).
- Tự chạy `docs/sprints/1.5/qc/scripts/regress-v24.mjs`, `audit.mjs`, lint, build, `ui-antipatterns.sh` trước khi báo xong.
- Không mở subagent. Spec sai thì ghi góp ý, không tự đổi.
Trả lời tóm tắt và dừng.

# Prompt cho `ba` — sprint 1, bản 2 (sau D45)

Đọc lại: `docs/DECISIONS.md` D45, `CLAUDE.md` (đã sửa nguyên tắc 6–8), `docs/phases/P0.md` (đã viết lại), `docs/sprints/1/plan.md` (bản 2).

## Việc
1. Xoá thư mục dở `docs/specs/FEAT-ci-mailhog/` (lỗi thời).
2. Viết spec cho hai feature, đều là hạ tầng: `US.md` + `SRS.md` rút gọn (mục 1, 4, 6, 8, 9, 11; mục khác ghi "Không áp dụng — story hạ tầng") + `QUESTIONS.md`.

| Feature | Story | Nguồn |
| --- | --- | --- |
| FEAT-scaffold | US-P0-02 | P0 L1; ARCHITECTURE §1–3, §2 cấu trúc `backend-go`, mục env; SYSTEM_DESIGN §2 |
| FEAT-ci | US-P0-03 | P0 L2; ARCHITECTURE mục test; CLAUDE.md "Lệnh" |

- AC tóm tắt trong `plan.md` là sàn; viết thành Given/When/Then, mỗi AC kèm lệnh kiểm chạy được.
- Danh sách cụ thể thứ dời vào `legacy/` và thứ giữ: lấy từ `git ls-files` ở gốc repo, ghi thành bảng trong SRS mục 4. Không bỏ sót file gốc nào.
- AC phân quyền: "không áp dụng — chưa có API nghiệp vụ" kèm lý do.
- Câu hỏi đụng sản phẩm thì ghi QUESTIONS; chi tiết kỹ thuật đã có trong tài liệu thì tự chốt.

## Khi xong
Commit `sprint 1: spec FEAT-scaffold, FEAT-ci` (chỉ `docs/specs/**`), tóm tắt ≤ 10 dòng, dừng.

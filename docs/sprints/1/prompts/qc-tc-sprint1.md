# Prompt cho `qc` — viết test case sprint 1

1. Nhận vai theo `docs/team/QC.md`. Quy trình mới (chủ dự án chốt 2026-10-01, `docs/team/PM.md` §3.3): QC viết test case từ spec **song song** với lúc dev thi công; chạy TC khi có handoff.
2. Bây giờ chỉ **viết TC**, chưa chạy. Viết từ AC (hộp đen). **Không đọc, không chờ code của dev**; không sửa file nào ngoài vùng của bạn.

## Việc
| Story | Spec | TC |
| --- | --- | --- |
| US-P0-02 | `docs/specs/FEAT-scaffold/` | `docs/sprints/1/qc/tc-US-P0-02.md` |
| US-P0-03 | `docs/specs/FEAT-ci/` | `docs/sprints/1/qc/tc-US-P0-03.md` |

Mỗi file: bảng TC-id → AC → tiền điều kiện → bước/lệnh (copy-paste chạy được) → kết quả mong đợi. Mỗi AC ≥ 1 TC; có TC nhánh lỗi; AC phân quyền "không áp dụng" thì TC kiểm ràng buộc thay thế mà spec nêu. Thêm TC kiểm chéo bắt buộc theo QC.md mục 3–4 áp dụng được cho story hạ tầng (lan phạm vi: diff ngoài vùng story, secret trong diff, `legacy/` có bị build/sửa không). Diff base của sprint: `1b72f54`.
Script chạy tự động nếu có: `docs/sprints/1/qc/scripts/`.

## Luật git (nhiều agent cùng repo)
Không `git add -A`/`git add .`/`git commit -a`. Commit đúng đường dẫn của bạn: `git commit -m "sprint 1: TC US-P0-02, US-P0-03" -- docs/sprints/1/qc`. Gặp `index.lock` → chờ rồi thử lại.

## Khi xong
Trả lời ≤ 5 dòng: số TC mỗi story, AC nào khó kiểm. Dừng, chờ PM giao chạy TC.

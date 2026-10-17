# Prompt cho `qc`: test case US-CI-02 (TL-CI #3, #5)

Worktree QC `/Users/kuro/Documents/TA_Agent_qc5`. Trước khi bắt đầu: `git fetch && git switch --detach origin/sprint/7-p4-p5`.

## Đọc
- `docs/team/QC.md`, `docs/team/CONTEXT.md`, `docs/sprints/7/plan.md` (story 0).
- `docs/sprints/6/techlead.md` mục TL-CI (bảng đề xuất dòng 3, 5 và mục "Không được làm").
- `docs/sprints/6/handoff/dev-US-P3-08.md` mục "CI nhanh hơn" (hiện trạng sau #1 #2 #4).
- **Không đọc code của dev.**

## Việc
Viết `docs/sprints/7/qc/tc-US-CI-02.md` (cột `TC-id → AC → tiền điều kiện → bước/lệnh → kết quả mong đợi`), kiểm bằng `gh run view --json jobs` / `--log` trên run thật:
- số test Go (`--- PASS` / `ok` từng gói) và số test Playwright sau khi tách **bằng đúng** trước khi tách, trên cùng SHA hoặc SHA chỉ khác `ci.yml`;
- mọi dòng `go test` vẫn có `-race -tags testroutes`; gói mới thêm vào repo tự rơi vào dòng "phần còn lại" (thử bằng một gói rỗng có test trên nhánh tạm);
- Lighthouse: cùng ngưỡng, cùng `numberOfRuns`, đủ URL như trước (hợp các shard = tập cũ);
- một test đỏ ở bất kỳ shard / dòng matrix nào làm CI đỏ;
- thời gian wall trước / sau, ghi số run;
- PR vào `main` vẫn chạy đủ (luật #1 không bị phá).

Xong thì commit `sprint 7: QC test case US-CI-02`, push, báo PM ≤ 3 dòng. Sau đó chờ: PM sẽ giao viết TC cho P4 / P5 khi spec APPROVED.

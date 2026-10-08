# Đội agent 4 pane trên herdr

Một người (bạn) + bốn phiên Claude Code chạy trong bốn pane herdr, cùng một repo, cùng một nhánh sprint. Giao tiếp giữa các agent đi qua **file trong repo**, không qua trí nhớ hội thoại; herdr chỉ dùng để đánh thức nhau và chờ nhau.

| Pane (tên agent) | Vai | Được sửa gì | Prompt |
| --- | --- | --- | --- |
| `pm` | PO/PM: cắt sprint, giao việc, thúc, tổng hợp, dừng lại báo cáo bạn | `docs/sprints/**`, `docs/PROGRESS.md`, `docs/thesis-notes/**` | `PM.md` (bạn dán vào đầu phiên) |
| `ba` | BA: viết US + SRS cho từng feature, làm rõ với bạn qua PM | `docs/specs/**` | `BA.md` (PM gửi) |
| `dev` | Dev: thi công từng user story theo lát dọc | `backend-go/**`, `frontend/**`, `db/**`, `scripts/**`, `seed/**` | `DEV.md` (PM gửi) |
| `qc` | QC: chạy cổng nghiệm thu, test theo AC, viết báo cáo lỗi; **không sửa code** | `docs/sprints/<n>/qc/**`, `frontend/e2e/**` (chỉ thêm test) | `QC.md` (PM gửi) |
| `research` | Research + **Tech Lead**: nghiên cứu theo câu hỏi PM; trả lời dev khi phân vân kỹ thuật; thẩm định spec của BA trước khi `APPROVED`; **không sửa code / spec / test** | `docs/research/**`, `docs/sprints/<n>/techlead.md`, `docs/specs/<feature>/TL-REVIEW.md` | `RESEARCH.md` (PM gửi) |

## Vòng sprint

```
bạn ──"bắt đầu sprint N"──▶ pm
pm: chọn feature từ backlog phase ─▶ docs/sprints/N/plan.md ─▶ hỏi bạn duyệt
pm ─prompt─▶ ba: viết US + SRS cho từng feature ─▶ docs/specs/<feature>/
        ba có câu hỏi ─▶ docs/specs/<feature>/QUESTIONS.md ─▶ pm hỏi BẠN ─▶ ba cập nhật
        research thẩm định spec ─▶ docs/specs/<feature>/TL-REVIEW.md ─▶ pm quyết từng ý ─▶ ba sửa ý ACCEPTED
pm: spec APPROVED ─▶ giao SONG SONG hai nhánh:
        ├─ dev: từng story một ─▶ code + test + commit ─▶ docs/sprints/N/handoff/dev-<story>.md
        └─ qc: viết test case từ AC (hộp đen, KHÔNG đọc code dev) ─▶ docs/sprints/N/qc/tc-<story>.md
hợp lại khi có handoff ─▶ qc chạy đúng bộ TC đã viết + kiểm chéo + cổng nghiệm thu
                      ─▶ docs/sprints/N/qc/report-<story>.md (PASS/FAIL)
        FAIL ─▶ pm ─▶ dev sửa (tối đa 2 vòng) ─▶ qc kiểm lại
pm: docs/sprints/N/report.md + cập nhật PROGRESS + thesis-notes ─▶ DỪNG, báo cáo bạn
bạn: đọc, bấm thử, "chốt" ──▶ pm merge --no-ff vào main, push ──▶ "tiếp" ──▶ sprint N+1
```

Toàn dự án = **10 sprint**, mỗi sprint 1–2 phase: xem `docs/sprints/ROADMAP.md`. Story trong sprint vẫn nhỏ (dev ≤ 1 ngày) và là **lát dọc**: migration → API → giao diện → test → seed, chạy được từ đầu đến cuối trước khi sang story kế. Không có sprint "chỉ backend" (ngoại lệ: PG — nền Go). Repo: `origin` = `github.com/kagamikuro1024/TA_Agent_v2`; nhánh `sprint/N-<slug>` từ `main`.

## Dựng đội

```bash
# một lần
curl -fsSL https://herdr.dev/install.sh | sh          # hoặc brew install herdr
npx skills add herdrdev/herdr --skill herdr -g         # skill để Claude Code điều khiển herdr
cd TA_Agent && herdr                                   # mở herdr tại gốc repo

# mỗi lần làm việc, trong một pane shell bất kỳ của herdr
bash scripts/team-up.sh                                # tạo 4 pane, khởi động claude, đặt tên pm/ba/dev/qc
```

Sau khi script chạy xong: bấm vào pane `pm`, dán toàn bộ nội dung `docs/team/PM.md`, rồi gõ `bắt đầu sprint 1`. Bạn chỉ nói chuyện với `pm`. Ba pane kia bạn nhìn để theo dõi và can thiệp khi herdr báo `blocked` (Claude Code đang hỏi quyền).

Để bớt bị `blocked`: `scripts/team-up.sh` khởi động `dev` và `qc` với `--permission-mode acceptEdits`; các lệnh shell được phép đã khai sẵn ở `.claude/settings.json`. Không dùng `--dangerously-skip-permissions`.

## Quy ước file giao tiếp

| File | Ai viết | Nội dung |
| --- | --- | --- |
| `docs/sprints/N/plan.md` | pm | Mục tiêu sprint, danh sách story (ID, feature, phase gốc, AC tóm tắt), thứ tự, nhánh |
| `docs/specs/<feature>/US.md` | ba | User story + AC dạng Given/When/Then |
| `docs/specs/<feature>/SRS.md` | ba | Yêu cầu chức năng/phi chức năng, dữ liệu, API, màn hình, luồng, nhánh lỗi |
| `docs/specs/<feature>/QUESTIONS.md` | ba ↔ pm | Câu hỏi mở, trả lời của bạn, ngày chốt |
| `docs/specs/<feature>/TL-REVIEW.md` | research viết, pm quyết | Thẩm định kỹ thuật của Tech Lead: `TLR-<số>`, mức, vị trí, vấn đề, đề xuất, bằng chứng, quyết định PM |
| `docs/sprints/N/qc/tc-<story>.md` | qc | Bảng test case viết từ AC trước khi có code: `TC-id → AC → tiền điều kiện → bước/lệnh → kết quả mong đợi`; script chạy được (nếu có) đặt ở `frontend/e2e/**` hoặc `docs/sprints/N/qc/scripts/` |
| `docs/sprints/N/handoff/dev-<story>.md` | dev | Đã làm gì, file đổi, test chạy, lệnh để QC chạy, việc còn nợ |
| `docs/sprints/N/qc/report-<story>.md` | qc | PASS/FAIL từng AC, lỗi kèm bước tái hiện, kết quả cổng nghiệm thu |
| `docs/sprints/N/techlead.md` | dev hỏi, research trả lời | Câu hỏi kỹ thuật `TL-<số>` và trả lời của Tech Lead; không chứa thay đổi AC / TC / spec |
| `docs/sprints/N/proposals.md` | ba / dev / qc viết, pm quyết | Đề xuất khi thấy spec, AC, kế hoạch, quy trình hay quyết định kỹ thuật không hợp lý: vấn đề, đề xuất, lý do + bằng chứng, ảnh hưởng; PM ghi quyết định |
| `docs/sprints/N/report.md` | pm | Báo cáo sprint cho bạn: xong gì, chưa xong gì, rủi ro, số liệu, đề xuất sprint kế |
| `docs/thesis-notes/sprint-N.md` | pm | Nguyên liệu luận văn: quyết định kỹ thuật, số đo, khó khăn |

Tất cả nằm trong git, nên khi viết báo cáo đồ án bạn có đủ: US/SRS theo feature (chương phân tích thiết kế), handoff + report theo sprint (chương triển khai), QC report (chương kiểm thử), thesis-notes (chương thực nghiệm).

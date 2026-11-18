### DEF-R7-01 — CI chạy đủ job vượt ngưỡng cứng 10 phút của US-CI-02
- Mức: Trung bình
- Sprint gốc: 7 · TC gốc: TC-20 (US-CI-02) · Hồi quy do sprint sau?: CÓ (các sprint 9.x–11 thêm nhiều spec Playwright vào cùng 3 shard; `.github/workflows/ci.yml` vẫn `shard: [1,2,3]`)
- Môi trường: GitHub Actions, nhánh sprint/11-p10-eval @be0c0d96 (run PR 43712425198), `gh run view`
- Bước tái hiện: 1. `gh run view 43712425198 --json jobs,createdAt,updatedAt`. 2. wall = 12:09:46 → 12:25:44 = 15,97 phút; job «Frontend e2e (Playwright 1/3)» 15,4 phút, 2/3 6,5 phút, 3/3 2,7 phút; Lighthouse simulate 10,3 phút, Go rest-1 9,5 phút. Run push e0bf1fc9 (`43710464639`) cũng 16,7 phút.
- Kỳ vọng (PM 2026-10-13): lượt chạy đủ wall ≤ 10 phút và nhanh hơn mốc cũ 13 m 43 s; Thực tế: ~16 phút, chậm hơn mốc cũ.
- Bằng chứng: `/tmp/qc2e/ci-run.json`, `/tmp/qc2e/ci-log.txt` · Nghi ngờ: `.github/workflows/ci.yml:235-253` (3 shard không cân bằng theo thời gian), job Lighthouse simulate 10,3 phút
- Chạy lại lần 2: ổn định (hai run liên tiếp ≥ 16 phút)

### DEF-R7-02 — Chuông gộp TICKET_NEW không báo ticket mới sau khi hàng chờ về 0
- Mức: Trung bình
- Sprint gốc: 7 · TC gốc: TC-P4-02-14 (US-P4-02 AC5) · Hồi quy do sprint sau?: KHÔNG (mã `internal/notify/aggregate.go` từ commit e1ee2f6d, sprint 7)
- Môi trường: ep-r @88ab83e4, lớp QC3B (GV qc3t, SV qc3s01/qc3s02) và QC3D (GV qc3t2)
- Bước tái hiện: 1. Lớp không còn ticket chờ; dòng gộp TICKET_NEW đã đọc, tiêu đề «1 câu hỏi đang chờ giảng viên». 2. SV tạo ticket (n=1). 3. GV dismiss nó (n=0). 4. SV tạo ticket mới (n=1), chờ 8 s.
- Kỳ vọng (US AC5 / SRS 4.4.3): ticket mới ⇒ dòng chưa đọc (nâng lên, phát SSE `notification`). Thực tế: dòng vẫn đã đọc, tiêu đề giữ «1 câu hỏi…», 0 khung SSE; GV không được báo cho tới khi số chờ vượt số cũ.
- Nguyên nhân (đọc mã): nhánh `n == 0` chỉ `NotifyAggMarkRead`, giữ nguyên tiêu đề; lần sau `case n > leadingInt(cur.Title)` so với số cũ nên n ≤ số cũ thì bị bỏ qua. Cùng cơ chế cho mọi loại gộp (AI_PENDING, POST_REPORTED, TICKET_STALE).
- Bằng chứng: `evidence-7/scripts-r3/p402c_def.txt`, `p402c.py`

## Ca lỗi thời có nguồn quyết định (xoá khỏi smoke-7.md và khỏi tổng; qc3 18/11/2026)
- TC-P5-06-19 (US-P5-06 AC8: giảng viên/TA/ADMIN mở `/me` «được chuyển về `/`») — thay bằng màn chặn chung «Bạn không có quyền xem màn này» + nút Về Hôm nay: `docs/sprints/7/proposals.md` #34 (PM ACCEPTED 2026-10-15).
- TC-S7-SEED-10 (US-S7-SEED AC3: «đúng 2 WATCH» ghim) — thêm tín hiệu GRADE, số WATCH không còn ghim, đáp án tính lại ở `seed/expected_crm.csv`: `docs/sprints/8/proposals.md` #5 (ACCEPTED có sửa, 2026-10-31).
- Ghi chú đếm: dòng «TC-S7-SEED-25 … TC-id (2 ca)» của bản cũ có một ca ma «TC-id» (hàng tiêu đề bảng), nên tổng thực của sprint 7 là 581 trước khi xoá LT → 579 sau khi xoá hai ca LT.

### DEF-R7-01 — CI chạy đủ job vượt ngưỡng cứng 10 phút của US-CI-02
- Mức: Trung bình
- Sprint gốc: 7 · TC gốc: TC-20 (US-CI-02) · Hồi quy do sprint sau?: CÓ (các sprint 9.x–11 thêm nhiều spec Playwright vào cùng 3 shard; `.github/workflows/ci.yml` vẫn `shard: [1,2,3]`)
- Môi trường: GitHub Actions, nhánh sprint/11-p10-eval @be0c0d96 (run PR 43712425198), `gh run view`
- Bước tái hiện: 1. `gh run view 43712425198 --json jobs,createdAt,updatedAt`. 2. wall = 12:09:46 → 12:25:44 = 15,97 phút; job «Frontend e2e (Playwright 1/3)» 15,4 phút, 2/3 6,5 phút, 3/3 2,7 phút; Lighthouse simulate 10,3 phút, Go rest-1 9,5 phút. Run push e0bf1fc9 (`43710464639`) cũng 16,7 phút.
- Kỳ vọng (PM 2026-10-13): lượt chạy đủ wall ≤ 10 phút và nhanh hơn mốc cũ 13 m 43 s; Thực tế: ~16 phút, chậm hơn mốc cũ.
- Bằng chứng: `/tmp/qc2e/ci-run.json`, `/tmp/qc2e/ci-log.txt` · Nghi ngờ: `.github/workflows/ci.yml:235-253` (3 shard không cân bằng theo thời gian), job Lighthouse simulate 10,3 phút
- Chạy lại lần 2: ổn định (hai run liên tiếp ≥ 16 phút)
- **Chạy lại 19/11 (qc3, sau sửa gốc 842f431c cache .deb Playwright):** vẫn F nhưng giảm mạnh: run 44380364341 @07a6b69c wall 9,5 phút (545 s) nhưng chỉ frontend (Go skipped); lượt đủ job cùng SHA — PR run 44380374004: 10,8 phút (Go rest-1 10,2 phút); run đủ job 44371210850 @9b20fb14: attempt 1 = 16,1 phút (cache lạnh), attempt 2 = 9,5 phút (cache ấm). Ngưỡng ≤ 10 phút chỉ đạt khi cache ấm. Cần: cân lại dòng Go rest-1 (~10 phút) và/hoặc cache lạnh.

### DEF-R7-02 — Chuông gộp TICKET_NEW không báo ticket mới sau khi hàng chờ về 0
- Mức: Trung bình
- Sprint gốc: 7 · TC gốc: TC-P4-02-14 (US-P4-02 AC5) · Hồi quy do sprint sau?: KHÔNG (mã `internal/notify/aggregate.go` từ commit e1ee2f6d, sprint 7)
- Môi trường: ep-r @88ab83e4, lớp QC3B (GV qc3t, SV qc3s01/qc3s02) và QC3D (GV qc3t2)
- Bước tái hiện: 1. Lớp không còn ticket chờ; dòng gộp TICKET_NEW đã đọc, tiêu đề «1 câu hỏi đang chờ giảng viên». 2. SV tạo ticket (n=1). 3. GV dismiss nó (n=0). 4. SV tạo ticket mới (n=1), chờ 8 s.
- Kỳ vọng (US AC5 / SRS 4.4.3): ticket mới ⇒ dòng chưa đọc (nâng lên, phát SSE `notification`). Thực tế: dòng vẫn đã đọc, tiêu đề giữ «1 câu hỏi…», 0 khung SSE; GV không được báo cho tới khi số chờ vượt số cũ.
- Nguyên nhân (đọc mã): nhánh `n == 0` chỉ `NotifyAggMarkRead`, giữ nguyên tiêu đề; lần sau `case n > leadingInt(cur.Title)` so với số cũ nên n ≤ số cũ thì bị bỏ qua. Cùng cơ chế cho mọi loại gộp (AI_PENDING, POST_REPORTED, TICKET_STALE).
- Bằng chứng: `evidence-7/scripts-r3/p402c_def.txt`, `p402c.py`
- **ĐÃ SỬA (e4b82a05, ep-r @1409b04d):** chạy lại TC-P4-02-14 ngày 19/11: ticket → xử lý hết (dòng tự đọc «0 câu hỏi…») → ticket mới ⇒ cùng dòng được nâng «1 câu hỏi đang chờ giảng viên», chưa đọc, 1 khung SSE ⇒ P.

### DEF-R7-04 — Ctrl+↵ gửi trả lời không sang phiếu kế khi bộ lọc là «Đang chờ»
- Mức: Thấp
- Sprint gốc: 7 · TC gốc: TC-P4-03-36 (US-P4-03 AC11) · Hồi quy do sprint sau?: chưa xác định (RealInbox.tsx)
- Bước tái hiện: lớp QC3G, GV qc3t5, `/inbox` (bộ lọc mặc định «Đang chờ») → chọn phiếu đầu → «Nhận» → gõ trả lời → Ctrl+↵.
- Kỳ vọng: gửi rồi sang phiếu kế. Thực tế: ticket ANSWERED nhưng URL vẫn `?ticket=` của phiếu cũ. Với bộ lọc «Tất cả» thì sang phiếu kế đúng. Nguyên nhân (đọc mã): `onNext` tìm phiếu trong `rows` của bộ lọc hiện tại; phiếu đã Nhận rời «Đang chờ» ⇒ findIndex=-1.
- Bằng chứng: `evidence-7/r3/ui-p403.log` (dòng INFO 36b), `scripts-r3/specs/inbox-p403.spec.ts`
- Quan sát kèm (chưa lập thành lỗi): mở thẳng `/inbox?ticket=<phiếu CLAIMED/ANSWERED>` khi bộ lọc mặc định «Đang chờ» hiện «Chọn một câu hỏi để xem» (khung chi tiết trống).

### DEF-R7-05 — Ô trả lời của hộp thư không «dính đáy» ở 375 px
- Mức: Thấp
- Sprint gốc: 7 · TC gốc: TC-P4-03-33 (US-P4-03 AC10, SRS 7: «ô trả lời dính đáy») · Hồi quy do sprint sau?: KHÔNG (chưa từng có position: sticky/fixed trong `RealInbox.module.css`, `Composer.module.css`)
- Bước tái hiện: 375×812, GV mở phiếu đã nhận → ô trả lời nằm cuối nội dung (đáy y=488 trên viewport 812), không bám đáy màn hình. Phần còn lại của AC10 (danh sách/chi tiết toàn màn hình, ?ticket=, Quay lại giữ bộ lọc+cuộn, vùng chạm ≥44 px, không tràn ngang) đạt.
- Bằng chứng: `evidence-7/r3/shots/p403-34-composer.png`, `ui-p403.log`

### DEF-R7-06 — Timeout của job CI không đạt «≥ 2× thời gian thực tế»; `changes` và `judge-attacks` không có `timeout-minutes`
- Mức: Thấp
- Sprint gốc: 7 · TC gốc: TC-E2 (US-CI-02) · Hồi quy do sprint sau?: một phần (Lighthouse devtools B chậm dần; hai job không timeout có từ trước US-CI-02)
- Bước tái hiện: so `timeout-minutes` trong `.github/workflows/ci.yml` @1409b04d với thời gian job của run 44343596969 (`gh api …/jobs`).
- Kỳ vọng: timeout ≥ 2× thực tế (tránh treo 6 giờ). Thực tế: «Frontend Lighthouse (devtools B)» 14,2 phút vs timeout 25 (cần ≥ 28,4); `changes` và `judge-attacks` không khai báo (mặc định 360 phút). Huỷ ≠ xanh: đạt.
- Bằng chứng: `evidence-7/r4/ci-tce2.log`

### DEF-R7-07 — Shard Playwright rỗng chạy lại CẢ BỘ test thay vì 0 test
- Mức: Thấp (tiềm ẩn: CI cố định 4 shard < 174 tệp spec nên chưa xảy ra)
- Sprint gốc: 7 · TC gốc: TC-E4 (US-CI-02) · Hồi quy do sprint sau?: CÓ (cách chia bằng `frontend/e2e/support/shard.mjs` + `mapfile` ở ci.yml là của vòng DEF-R7-01)
- Bước tái hiện: `node e2e/support/shard.mjs 200 200` in một dòng rỗng ⇒ `mapfile -t files` ra 1 phần tử rỗng ⇒ `playwright test --grep-invert @real ""` liệt kê 2816 test / 174 tệp.
- Kỳ vọng: shard rỗng không chạy gì, không đỏ vô cớ, không ảnh hưởng TC-01 (mỗi test đúng một lần). Thực tế: chạy toàn bộ suite (mỗi test chạy thêm lần).
- Bằng chứng: `evidence-7/r4/ci-tce4.log`

### DEF-R7-08 — Nút «Lưu điểm danh» bị khoá khi mất mạng, không xếp hàng `complete:true`
- Mức: Trung bình
- Sprint gốc: 7 · TC gốc: TC-P5-02-33 (US-P5-02 AC13) · Hồi quy do sprint sau?: chưa xác định (`RealAttendance.tsx`: `locked = saved || conflicts || q.status === "offline"`)
- Bước tái hiện: GV mở /attendance, `context.setOffline(true)`, đổi một hàng → nút đầu trang «Lưu điểm danh» và nút thanh dưới «Chờ mạng · … có mặt · … vắng» đều disabled; có mạng lại: `attendance_done_at` vẫn NULL.
- Kỳ vọng (AC13, TC-33): bấm Lưu khi offline xếp hàng cùng `complete:true`, có mạng thì đặt `done_at`. Thực tế: hành vi của AC13b (khoá nút). Các ca AC13 khác (banner, kho `ep:draft:…`, tải lại, đăng xuất hỏi trước) đạt.
- Bằng chứng: `evidence-7/r3/ui-p502.log` (TC-P5-02-33)

### DEF-R7-09 — Nhịp heartbeat khi Redis dừng mất ~8 giây (SRS: ≤ 1 s)
- Mức: Trung bình
- Sprint gốc: 7 · TC gốc: TC-P5-03-29 (US-P5-03 AC10) · Hồi quy do sprint sau?: chưa xác định
- Bước tái hiện: cửa sổ khoá 02:54–02:56 ngày 19/11: `docker stop ep-r-redis-1`, SV đã ack gửi `POST /activity/heartbeat` → 204 nhưng sau 7,98 s.
- Kỳ vọng (SRS 4.6.1 bước 5, `RedisBudget` = 500 ms): 204 trong ≤ 1 s, tăng `activity_heartbeat_dropped_total`, log không PII. Thực tế: 204, log WARN `activity_heartbeat_dropped_total … context deadline exceeded` không PII (đạt), nhưng độ trễ ~8 s.
- Bằng chứng: `evidence-7/scripts-r3/p53lock.py`, `master7.json` (TC-P5-03-29)

## Ca lỗi thời có nguồn quyết định (xoá khỏi smoke-7.md và khỏi tổng; qc3 18/11/2026)
- TC-P5-06-19 (US-P5-06 AC8: giảng viên/TA/ADMIN mở `/me` «được chuyển về `/`») — thay bằng màn chặn chung «Bạn không có quyền xem màn này» + nút Về Hôm nay: `docs/sprints/7/proposals.md` #34 (PM ACCEPTED 2026-10-15).
- TC-S7-SEED-10 (US-S7-SEED AC3: «đúng 2 WATCH» ghim) — thêm tín hiệu GRADE, số WATCH không còn ghim, đáp án tính lại ở `seed/expected_crm.csv`: `docs/sprints/8/proposals.md` #5 (ACCEPTED có sửa, 2026-10-31).
- Ghi chú đếm: dòng «TC-S7-SEED-25 … TC-id (2 ca)» của bản cũ có một ca ma «TC-id» (hàng tiêu đề bảng), nên tổng thực của sprint 7 là 581 trước khi xoá LT → 579 sau khi xoá hai ca LT.

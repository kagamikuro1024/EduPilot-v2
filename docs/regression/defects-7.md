### DEF-R7-01 — CI chạy đủ job vượt ngưỡng cứng 10 phút của US-CI-02
- Mức: Trung bình
- Sprint gốc: 7 · TC gốc: TC-20 (US-CI-02) · Hồi quy do sprint sau?: CÓ (các sprint 9.x–11 thêm nhiều spec Playwright vào cùng 3 shard; `.github/workflows/ci.yml` vẫn `shard: [1,2,3]`)
- Môi trường: GitHub Actions, nhánh sprint/11-p10-eval @be0c0d96 (run PR 43712425198), `gh run view`
- Bước tái hiện: 1. `gh run view 43712425198 --json jobs,createdAt,updatedAt`. 2. wall = 12:09:46 → 12:25:44 = 15,97 phút; job «Frontend e2e (Playwright 1/3)» 15,4 phút, 2/3 6,5 phút, 3/3 2,7 phút; Lighthouse simulate 10,3 phút, Go rest-1 9,5 phút. Run push e0bf1fc9 (`43710464639`) cũng 16,7 phút.
- Kỳ vọng (PM 2026-10-13): lượt chạy đủ wall ≤ 10 phút và nhanh hơn mốc cũ 13 m 43 s; Thực tế: ~16 phút, chậm hơn mốc cũ.
- Bằng chứng: `/tmp/qc2e/ci-run.json`, `/tmp/qc2e/ci-log.txt` · Nghi ngờ: `.github/workflows/ci.yml:235-253` (3 shard không cân bằng theo thời gian), job Lighthouse simulate 10,3 phút
- Chạy lại lần 2: ổn định (hai run liên tiếp ≥ 16 phút)
- **Chạy lại 19/11 (qc3, sau sửa gốc 842f431c cache .deb Playwright):** vẫn F nhưng giảm mạnh: run 44380364341 @07a6b69c wall 9,5 phút (545 s) nhưng chỉ frontend (Go skipped); lượt đủ job cùng SHA — PR run 44380374004: 10,8 phút (Go rest-1 10,2 phút); run đủ job 44371210850 @9b20fb14: attempt 1 = 16,1 phút (cache lạnh), attempt 2 = 9,5 phút (cache ấm). Ngưỡng ≤ 10 phút chỉ đạt khi cache ấm. Cần: cân lại dòng Go rest-1 (~10 phút) và/hoặc cache lạnh.
- **Số đo CI mới của PM (21/11, sau 7a3ebd35 / eb57f6d3), tính từ job đầu bắt đầu tới job cuối kết thúc, đối chiếu lại bằng `gh api …/jobs`:** run PR 44862671196 @eb57f6d3 wall **512 s** (đạt ≤ 600 s; job lâu nhất Go course 491 s); run push 44862637349 @7a3ebd35 wall **636 s** (vượt 600 s; Go rest-2 617 s, rest-3 585 s). ⇒ TC-20 **giữ F** (một trong hai run vượt ngưỡng); dev2 đang cân lại shard Go (523d9076: làm mới `ci-shard-weights.txt`). Chưa đo lại sau 523d9076.
- **ĐÃ SỬA (523d9076 + 7a3ebd35) — xác nhận @58448e0a ngày 2026-11-21:** PR run 44907729321, hai lượt đủ 22 job success (attempt 1: wall 526 s; attempt 2: wall 498 s; tính từ job đầu bắt đầu tới job cuối kết thúc, đối chiếu cả created→completed 547 s / 503 s) ⇒ cả hai ≤ 600 s ⇒ TC-20 P. Lưu ý: run push cùng SHA 44907725269 (Go bị bỏ qua, chạy song song với PR run) wall 733 s chỉ frontend — không phải lượt đủ job; nếu runner bị tranh chấp thì ngưỡng 600 s vẫn có thể vượt. Bằng chứng: `evidence-7/r7/ci-run-44907729321-a1-jobs.tsv`, `-a2-jobs.tsv`, `ci-run-44907725269-a1-jobs.tsv`.

### DEF-R7-02 — Chuông gộp TICKET_NEW không báo ticket mới sau khi hàng chờ về 0
- Mức: Trung bình
- Sprint gốc: 7 · TC gốc: TC-P4-02-14 (US-P4-02 AC5) · Hồi quy do sprint sau?: KHÔNG (mã `internal/notify/aggregate.go` từ commit e1ee2f6d, sprint 7)
- Môi trường: ep-r @88ab83e4, lớp QC3B (GV qc3t, SV qc3s01/qc3s02) và QC3D (GV qc3t2)
- Bước tái hiện: 1. Lớp không còn ticket chờ; dòng gộp TICKET_NEW đã đọc, tiêu đề «1 câu hỏi đang chờ giảng viên». 2. SV tạo ticket (n=1). 3. GV dismiss nó (n=0). 4. SV tạo ticket mới (n=1), chờ 8 s.
- Kỳ vọng (US AC5 / SRS 4.4.3): ticket mới ⇒ dòng chưa đọc (nâng lên, phát SSE `notification`). Thực tế: dòng vẫn đã đọc, tiêu đề giữ «1 câu hỏi…», 0 khung SSE; GV không được báo cho tới khi số chờ vượt số cũ.
- Nguyên nhân (đọc mã): nhánh `n == 0` chỉ `NotifyAggMarkRead`, giữ nguyên tiêu đề; lần sau `case n > leadingInt(cur.Title)` so với số cũ nên n ≤ số cũ thì bị bỏ qua. Cùng cơ chế cho mọi loại gộp (AI_PENDING, POST_REPORTED, TICKET_STALE).
- Bằng chứng: `evidence-7/scripts-r3/p402c_def.txt`, `p402c.py`
- **ĐÃ SỬA (e4b82a05, ep-r @1409b04d):** chạy lại TC-P4-02-14 ngày 19/11: ticket → xử lý hết (dòng tự đọc «0 câu hỏi…») → ticket mới ⇒ cùng dòng được nâng «1 câu hỏi đang chờ giảng viên», chưa đọc, 1 khung SSE ⇒ P.
- **ĐÃ SỬA (e4b82a05) — xác nhận sau 425396a7 ngày 2026-11-21:** TC-P4-02-14 chạy lại (lớp QC3G, GV qc3t5): ticket → xử lý hết → ticket mới ⇒ dòng gộp chưa đọc, «1 câu hỏi đang chờ giảng viên», có khung SSE; GV đã đọc rồi xử lý hết, ticket mới ⇒ vẫn chưa đọc + SSE (n không lớn hơn số cũ); thêm ticket ⇒ «2 câu hỏi…» ⇒ P. Bằng chứng: `evidence-7/r7/r14-p402-14.log`, `scripts-r7/r14_r7.py`.

### DEF-R7-04 — Ctrl+↵ gửi trả lời không sang phiếu kế khi bộ lọc là «Đang chờ»
- Mức: Thấp
- Sprint gốc: 7 · TC gốc: TC-P4-03-36 (US-P4-03 AC11) · Hồi quy do sprint sau?: chưa xác định (RealInbox.tsx)
- Bước tái hiện: lớp QC3G, GV qc3t5, `/inbox` (bộ lọc mặc định «Đang chờ») → chọn phiếu đầu → «Nhận» → gõ trả lời → Ctrl+↵.
- Kỳ vọng: gửi rồi sang phiếu kế. Thực tế: ticket ANSWERED nhưng URL vẫn `?ticket=` của phiếu cũ. Với bộ lọc «Tất cả» thì sang phiếu kế đúng. Nguyên nhân (đọc mã): `onNext` tìm phiếu trong `rows` của bộ lọc hiện tại; phiếu đã Nhận rời «Đang chờ» ⇒ findIndex=-1.
- Bằng chứng: `evidence-7/r3/ui-p403.log` (dòng INFO 36b), `scripts-r3/specs/inbox-p403.spec.ts`
- Quan sát kèm (chưa lập thành lỗi): mở thẳng `/inbox?ticket=<phiếu CLAIMED/ANSWERED>` khi bộ lọc mặc định «Đang chờ» hiện «Chọn một câu hỏi để xem» (khung chi tiết trống).
- **ĐÃ SỬA (24036aae) — xác nhận sau 425396a7 ngày 2026-11-21:** TC-P4-03-36 chạy lại: lớp QC3G, GV qc3t5, bộ lọc «Đang chờ»: Nhận phiếu đầu → gõ → Ctrl+↵ ⇒ ticket ANSWERED và URL sang đúng phiếu kế; bộ lọc «Tất cả» cũng đúng ⇒ P. Bằng chứng: `evidence-7/r7/ui-r7-inbox.log`, `scripts-r7/specs/inbox-r7.spec.ts`.

### DEF-R7-05 — Ô trả lời của hộp thư không «dính đáy» ở 375 px
- Mức: Thấp
- Sprint gốc: 7 · TC gốc: TC-P4-03-33 (US-P4-03 AC10, SRS 7: «ô trả lời dính đáy») · Hồi quy do sprint sau?: KHÔNG (chưa từng có position: sticky/fixed trong `RealInbox.module.css`, `Composer.module.css`)
- Bước tái hiện: 375×812, GV mở phiếu đã nhận → ô trả lời nằm cuối nội dung (đáy y=488 trên viewport 812), không bám đáy màn hình. Phần còn lại của AC10 (danh sách/chi tiết toàn màn hình, ?ticket=, Quay lại giữ bộ lọc+cuộn, vùng chạm ≥44 px, không tràn ngang) đạt.
- Bằng chứng: `evidence-7/r3/shots/p403-34-composer.png`, `ui-p403.log`
- **ĐÃ SỬA (24036aae) — xác nhận sau 425396a7 ngày 2026-11-21:** TC-P4-03-33 chạy lại ở 375×812: ô trả lời `position: sticky`, đáy cách đáy màn hình 83 px (trên thanh điều hướng dưới); phiếu nội dung dài (chi tiết cuộn 1962 px) vẫn thấy ô trả lời ở cuộn 0, cuộn đáy, cuộn lại ⇒ P. Bằng chứng: `evidence-7/r7/ui-r7-inbox.log`, `r7/shots/p403-34-composer.png`, `r7-p403-33b-top.png`.

### DEF-R7-06 — Timeout của job CI không đạt «≥ 2× thời gian thực tế»; `changes` và `judge-attacks` không có `timeout-minutes`
- Mức: Thấp
- Sprint gốc: 7 · TC gốc: TC-E2 (US-CI-02) · Hồi quy do sprint sau?: một phần (Lighthouse devtools B chậm dần; hai job không timeout có từ trước US-CI-02)
- Bước tái hiện: so `timeout-minutes` trong `.github/workflows/ci.yml` @1409b04d với thời gian job của run 44343596969 (`gh api …/jobs`).
- Kỳ vọng: timeout ≥ 2× thực tế (tránh treo 6 giờ). Thực tế: «Frontend Lighthouse (devtools B)» 14,2 phút vs timeout 25 (cần ≥ 28,4); `changes` và `judge-attacks` không khai báo (mặc định 360 phút). Huỷ ≠ xanh: đạt.
- Bằng chứng: `evidence-7/r4/ci-tce2.log`
- **Chạy lại sau 425396a7 (2026-11-21): vẫn F (một phần).** Đã sửa (7062b009): `changes` timeout-minutes=5, `judge-attacks` 10 (thực tế 0,2–0,4 và 2,7–3,6 phút ⇒ ≥2×); `scripts/ci-guard.test.mjs` 3/3 đạt cục bộ và ở bước «CI guard» của job fe-check trên GitHub (run 44855393997 @1c3ec9fa: success). **Còn trượt:** job `lhci` vẫn `timeout-minutes: 25` (ci.yml dòng 324) trong khi thực tế tới 15,7 phút (devtools B run 44367157472; 14,2/14,1 phút ở 44343596969, 44361693563; devtools A 15,3; simulate A 15,3) ⇒ cần ≥ 31,4 phút. Các job còn lại đạt (e2e max 11,8 → 25; Go rest-1 max 9,7 → 30; không tag max 6,5 → 25). Bằng chứng: `evidence-7/r7/ci-r7-static.log`, `r7/ci-job-times.tsv`, `r7/ci-44855393997-fe-check-steps.tsv`, `scripts-r7/ci-r7.sh`.
- **ĐÃ SỬA (7062b009; lhci chia lại bởi 7a3ebd35) — xác nhận @58448e0a ngày 2026-11-21:** mọi job có timeout-minutes; thời gian thực tế lớn nhất trong 2 lượt đủ job là 8,5 phút (e2e 4/4) và lhci 8,0 phút (devtools A) ⇒ lhci timeout 25 ≥ 16 đạt; 22/22 dòng job ≥ 2× ⇒ TC-E2 P. Bằng chứng: `evidence-7/r7/ci-58448e0a.yml`, `ci-run-44907729321-a{1,2}-jobs.tsv`.

### DEF-R7-07 — Shard Playwright rỗng chạy lại CẢ BỘ test thay vì 0 test
- Mức: Thấp (tiềm ẩn: CI cố định 4 shard < 174 tệp spec nên chưa xảy ra)
- Sprint gốc: 7 · TC gốc: TC-E4 (US-CI-02) · Hồi quy do sprint sau?: CÓ (cách chia bằng `frontend/e2e/support/shard.mjs` + `mapfile` ở ci.yml là của vòng DEF-R7-01)
- Bước tái hiện: `node e2e/support/shard.mjs 200 200` in một dòng rỗng ⇒ `mapfile -t files` ra 1 phần tử rỗng ⇒ `playwright test --grep-invert @real ""` liệt kê 2816 test / 174 tệp.
- Kỳ vọng: shard rỗng không chạy gì, không đỏ vô cớ, không ảnh hưởng TC-01 (mỗi test đúng một lần). Thực tế: chạy toàn bộ suite (mỗi test chạy thêm lần).
- Bằng chứng: `evidence-7/r4/ci-tce4.log`
- **ĐÃ SỬA (7062b009) — xác nhận sau 425396a7 ngày 2026-11-21:** TC-E4: `shard.mjs 200 200` in rỗng (0 byte); đoạn bước CI chạy trong bash 5: `mapfile` ra 0 phần tử → «shard rỗng: không có tệp nào để chạy», thoát 0, không gọi playwright; 141 tệp spec = 141 tệp trong 4 shard (31/35/35/40), không tệp ở hai shard; ci-guard 3/3 ⇒ P. Bằng chứng: `evidence-7/r7/ci-r7-static.log`.

### DEF-R7-08 — Nút «Lưu điểm danh» bị khoá khi mất mạng, không xếp hàng `complete:true`
- Mức: Trung bình
- Sprint gốc: 7 · TC gốc: TC-P5-02-33 (US-P5-02 AC13) · Hồi quy do sprint sau?: chưa xác định (`RealAttendance.tsx`: `locked = saved || conflicts || q.status === "offline"`)
- Bước tái hiện: GV mở /attendance, `context.setOffline(true)`, đổi một hàng → nút đầu trang «Lưu điểm danh» và nút thanh dưới «Chờ mạng · … có mặt · … vắng» đều disabled; có mạng lại: `attendance_done_at` vẫn NULL.
- Kỳ vọng (AC13, TC-33): bấm Lưu khi offline xếp hàng cùng `complete:true`, có mạng thì đặt `done_at`. Thực tế: hành vi của AC13b (khoá nút). Các ca AC13 khác (banner, kho `ep:draft:…`, tải lại, đăng xuất hỏi trước) đạt.
- Bằng chứng: `evidence-7/r3/ui-p502.log` (TC-P5-02-33)
- **ĐÃ SỬA (131a5868) — xác nhận sau 425396a7 ngày 2026-11-21:** TC-P5-02-33 chạy lại: offline, đổi một hàng, nút «Lưu điểm danh» (đầu trang và thanh dưới) không khoá, bấm xếp hàng; có mạng lại «Đã lưu», `attendance_done_at` đặt, PUT có `complete:true`; biến thể không còn thay đổi chờ cũng đặt done_at ⇒ P. Bằng chứng: `evidence-7/r7/ui-r7-att.log`, `r7/shots/r7-p502-33-*.png`.

### DEF-R7-09 — Nhịp heartbeat khi Redis dừng mất ~8 giây (SRS: ≤ 1 s)
- Mức: Trung bình
- Sprint gốc: 7 · TC gốc: TC-P5-03-29 (US-P5-03 AC10) · Hồi quy do sprint sau?: chưa xác định
- Bước tái hiện: cửa sổ khoá 02:54–02:56 ngày 19/11: `docker stop ep-r-redis-1`, SV đã ack gửi `POST /activity/heartbeat` → 204 nhưng sau 7,98 s.
- Kỳ vọng (SRS 4.6.1 bước 5, `RedisBudget` = 500 ms): 204 trong ≤ 1 s, tăng `activity_heartbeat_dropped_total`, log không PII. Thực tế: 204, log WARN `activity_heartbeat_dropped_total … context deadline exceeded` không PII (đạt), nhưng độ trễ ~8 s.
- Bằng chứng: `evidence-7/scripts-r3/p53lock.py`, `master7.json` (TC-P5-03-29)
- **Không tái hiện — ĐÃ SỬA/ĐẠT sau 425396a7 ngày 2026-11-21:** đo lại 09:28 (02:28 UTC, khoá toàn cục 09:28:20–09:28:32): `docker stop ep-r-redis-1` 02:28:20.83Z, 5 nhịp heartbeat 204 cả năm, 689/687/692/691/700 ms (≤ 1 s); 5 dòng WARN `activity_heartbeat_dropped_total` (total 1…5), không PII; Redis healthy sau 5,2 s ⇒ P (khớp số dev2 ~685 ms; lần 19/11 là 7,98 s). Bằng chứng: `evidence-7/r7/p29-hb-redis.log`, `r7/gw-p29-gateway.log`, `scripts-r7/p29_hb_redis.py`.

### DEF-R7-10 — «Xem trước» tạo lịch buổi học gửi Idempotency-Key
- Mức: Thấp
- Sprint gốc: 7 · TC gốc: TC-P5-01-26 (US-P5-01 AC9) · Hồi quy do sprint sau?: chưa xác định
- Bước tái hiện: GV mở /calendar → «Tạo lịch buổi học» → điền → «Xem trước» 3 lần rồi «Tạo N buổi» 2 lần; theo dõi network.
- Kỳ vọng: `generate?dry_run=true` KHÔNG có Idempotency-Key (SRS crm-360 4.2.1 bước 1, TLR-12: xem trước chỉ đọc, khoá chiếm suất lưu vô ích); khoá của lần tạo thật mới mỗi lần. Thực tế: 3 lần xem trước đều gửi khoá riêng (ep-f5d4678a…, ep-bbfe692e…, ep-2e8bbfcd…); 2 lần tạo thật khoá khác nhau và khác khoá xem trước (vế «khoá mới mỗi lần» đạt).
- Bằng chứng: `evidence-7/r3/ui-p501-r5.log`
- **ĐÃ SỬA (01b3f6d9) — xác nhận sau 425396a7 ngày 2026-11-21:** TC-P5-01-26 chạy lại: 4 lần «Xem trước» đều không có Idempotency-Key, số buổi không đổi khi xem trước; 2 lần «Tạo N buổi» khoá khác nhau và khác khoá xem trước ⇒ P. Bằng chứng: `evidence-7/r7/ui-r7-cal.log`, `scripts-r7/specs/cal-r7.spec.ts`.

### DEF-R7-11 — Trạng thái rỗng của Hộp thư hỗ trợ không đúng chữ / thiếu nút «Xem tất cả»
- Mức: Thấp
- Sprint gốc: 7 · TC gốc: TC-P4-03-38 (US-P4-03 AC13) · Hồi quy do sprint sau?: chưa xác định
- Bước tái hiện: GV mở /inbox của lớp còn ticket ở trạng thái khác, chọn bộ lọc «Đã nhận» (rỗng); và /inbox của lớp không có ticket.
- Kỳ vọng: «Không có câu hỏi nào đang chờ.» + nút `Xem tất cả` (lớp 2). Thực tế: «Chưa có câu hỏi cần xử lý», không có nút/liên kết «Xem tất cả» (khung xương, lỗi + «Thử lại», CLS 0 đều đạt).
- Bằng chứng: `evidence-7/r3/ui-p7c.log`, `shots/p7c-inbox-empty2.png`
- **Chạy lại sau 425396a7 (2026-11-21), chấm theo P-11-81 (BA 263920fb): vẫn F.** Đạt: lớp không ticket → «Chưa có câu hỏi cần xử lý» (không dấu chấm, không nút «Xem tất cả»); khung xương, lỗi + «Thử lại», CLS 0. **Trượt:** bộ lọc «Đã nhận» rỗng khi lớp QC3T1 còn ticket khác hiện «Chưa có câu hỏi cần xử lý», kỳ vọng «Không có câu hỏi trong bộ lọc này» (US-P4-03 AC13; SRS escalation-notify 541). Lặp cả khi đang chọn một phiếu rồi đổi bộ lọc. Nguyên nhân (đọc mã `RealInbox.tsx`): `hasRows = rows.length > 0 || !!selected` — bộ lọc rỗng không có phiếu đang chọn vào nhánh «Chưa có câu hỏi cần xử lý» (dòng 169), chỉ vào dòng 135 khi còn phiếu chọn (vd. vừa Nhận phiếu cuối của bộ lọc). Cách sửa gợi ý: phân biệt «lớp không có ticket nào» với «bộ lọc rỗng» (số ticket mọi trạng thái). Bằng chứng: `evidence-7/r7/ui-r7-ui.log` (TC-P4-03-38, 38dbg A–D), `r7/shots/r7-inbox-filter-empty.png`, `r7-38dbg-B.png`, `r7-38dbg-D.png`.
- **ĐÃ SỬA (debd2d05) — xác nhận sau ep-r rebuild ngày 2026-11-21:** TC-P4-03-38 chạy lại ở 1440 và 375: bộ lọc rỗng khi lớp còn phiếu → «Không có câu hỏi trong bộ lọc này» (kể cả khi đã chọn một phiếu rồi đổi bộ lọc); lớp không có phiếu → «Chưa có câu hỏi cần xử lý», không nút «Xem tất cả»; khung xương, lỗi + «Thử lại», CLS 0 đạt ⇒ P (P-11-81). Bằng chứng: `evidence-7/r7/ui-r7-i38.log`, `r7/shots/r7-i38-*.png`, `scripts-r7/specs/inbox38-r7.spec.ts`.

### DEF-R7-12 — Menu bài AI của giảng viên ghi «Loại», không phải «Loại khỏi tri thức»
- Mức: Thấp
- Sprint gốc: 7 · TC gốc: TC-P4-06-47 (US-P4-06 AC14, DESIGN §13) · Hồi quy do sprint sau?: chưa xác định
- Bước tái hiện: GV mở thread có bài AI «Chờ xác nhận» ở lớp của mình; mở menu «Thêm hành động» cạnh «Chỉnh sửa».
- Kỳ vọng: «Loại khỏi tri thức» (giảng viên). Thực tế: mục «Loại». Các phần khác của AC14 (menu Báo cáo, danh sách lý do, «Đã báo cáo · Hoàn tác» 5 s, Ẩn bài/Ghim/Giữ bài, Sửa/Xoá, 375 px) đạt.
- Bằng chứng: `evidence-7/r3/ui-p7c.log` (TC-P4-06-47c)
- **ĐÃ SỬA (1995e59b) — xác nhận sau 425396a7 ngày 2026-11-21:** TC-P4-06-47 chạy lại: GV thấy menu «Thêm hành động» với mục «Loại khỏi tri thức»; chọn → REJECTED, SV không còn thấy bài; các phần còn lại AC14 ở 1280 và 375 đạt ⇒ P. Bằng chứng: `evidence-7/r7/ui-r7-ui.log`, `r7/shots/p7c-ai-teacher.png`.

### DEF-R7-13 — Trạng thái rỗng khối Chuyên cần của /me thiếu dấu chấm cuối
- Mức: Thấp
- Sprint gốc: 7 · TC gốc: TC-P5-06-18 (US-P5-06 AC7) · Hồi quy do sprint sau?: chưa xác định
- Bước tái hiện: SV vào /me của lớp có buổi học nhưng chưa điểm danh lần nào (hoặc chưa có dữ liệu).
- Kỳ vọng (US-P5-06 AC7, TC-P5-06-18): «Chưa có buổi nào được điểm danh.» Thực tế: «Chưa có buổi nào được điểm danh» (EmptyState title, không dấu chấm; e2e me.spec cũng khớp chữ không chấm). Các vế còn lại của TC (khung xương, 500 + Thử lại, offline giữ nội dung + dải mất mạng, không nút) đạt. Lệch chỉ ở dấu câu; nếu PM chốt quy ước tiêu đề EmptyState không có dấu chấm thì sửa SRS/TC.
- Bằng chứng: `evidence-7/r3/ui-p506.log`, `ui-p506d.log`, `shots/p506d-18-qc3p5s04.png`
- **ĐÃ CHỐT (P-11-82) — xác nhận sau 425396a7 ngày 2026-11-21:** PM chọn DESIGN §23.4 (tiêu đề EmptyState không dấu chấm); BA vá US-P5-06 AC7 (xem `docs/specs/FEAT-crm-360/US.md` dòng 228 trên sprint/11-p10-eval, commit 0081082e). TC-P5-06-18 chấm lại: «Chưa có buổi nào được điểm danh» không dấu chấm, không nút; khung xương, 500 + Thử lại, offline giữ nội dung đạt ⇒ P. Bằng chứng: `evidence-7/r7/ui-r7-me.log`, `r7/shots/r7-me-empty-18.png`.

## Ca lỗi thời có nguồn quyết định (xoá khỏi smoke-7.md và khỏi tổng; qc3 18/11/2026)
- TC-P5-06-19 (US-P5-06 AC8: giảng viên/TA/ADMIN mở `/me` «được chuyển về `/`») — thay bằng màn chặn chung «Bạn không có quyền xem màn này» + nút Về Hôm nay: `docs/sprints/7/proposals.md` #34 (PM ACCEPTED 2026-10-15).
- TC-S7-SEED-10 (US-S7-SEED AC3: «đúng 2 WATCH» ghim) — thêm tín hiệu GRADE, số WATCH không còn ghim, đáp án tính lại ở `seed/expected_crm.csv`: `docs/sprints/8/proposals.md` #5 (ACCEPTED có sửa, 2026-10-31).
- Ghi chú đếm: dòng «TC-S7-SEED-25 … TC-id (2 ca)» của bản cũ có một ca ma «TC-id» (hàng tiêu đề bảng), nên tổng thực của sprint 7 là 581 trước khi xoá LT → 579 sau khi xoá hai ca LT.

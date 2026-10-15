# Sprint 7: báo cáo (P4 Escalation + Mail, P5 CRM + Hồ sơ 360)

**Mục tiêu:**
- **P4:** AI chưa chắc thì giảng viên được báo ngay, trả lời trong app; sinh viên nhận câu trả lời trong chat riêng, qua chuông và qua mail. Ticket không nằm im quá 72 giờ. Threads có kiểm duyệt.
- **P5:** giảng viên điểm danh nhanh trên điện thoại; xem hồ sơ 360 từng sinh viên; sinh viên hỏi AI ra đúng số chuyên cần, phát biểu của mình; đo thời gian học minh bạch.

**Kết quả:** 14/14 story sản phẩm PASS. **Cổng P4 ĐẠT, cổng P5 ĐẠT.** Story CI US-CI-02 ghi **FAIL** sau vòng sửa thứ 2; lỗi còn lại tách thành BUG-S7-01 và đã PASS. Nhánh `sprint/7-p4-p5`.

| Story | Nội dung | QC |
| --- | --- | --- |
| US-CI-02 | TL-CI #3 (shard Playwright 3 phần, Lighthouse matrix) và #5 (Go static ∥ matrix test) | **FAIL** (vòng 3: 1/6 lượt nguội đỏ) → tách BUG-S7-01 |
| BUG-S7-01 | Sửa ở gốc mọi test Go chờ đồng hồ thật; tách dòng `rest` thành 2 | PASS, sau sửa (6/6 lượt nguội xanh, ≤ 10 phút) |
| US-P4-01 | Migration `00012_escalation_notify`, ticket idempotent theo `message_id`, trần 5 ticket mở, nút "Nhờ giảng viên" | PASS, sau sửa |
| US-P4-02 | `internal/notify` một cửa, chuông tức thì qua `/events` (Redis pub/sub, chạy qua 2 bản gateway), gộp thông báo | PASS, sau 2 vòng sửa |
| US-P4-03 | `/inbox` thật: danh sách + hội thoại, `Nhận` (người sau 409 thấy ai nhận, lúc nào), trả lời vào chat riêng, phím tắt, 375 px | PASS, sau 2 vòng sửa |
| US-P4-04 | Mail: `Message-ID` tất định (đúng một thư dù worker chết), 5 mẫu thư, thư lỗi hiện ở `/observability` | PASS |
| US-P4-05 | Vòng đời ticket phía sinh viên (hỏi tiếp / đã rõ), SLA 24 h / 72 h (T2–T6), kho tri thức, ngưỡng theo lớp | PASS, sau sửa |
| US-P4-06 | Kiểm duyệt Threads (báo cáo, 3 báo cáo tự ẩn, ẩn / sửa / xoá, ghim), sinh lại câu trả lời AI khi sửa (`00014_forum_ai_regen`), nạp lại tri thức đã xác nhận | PASS, sau sửa |
| US-P4-07 | Tin nhắn khủng hoảng: luật hai tầng, câu hỗ trợ chung cho chat + Threads, không báo ai | PASS, sau sửa (tài liệu) |
| US-P5-01 | Migration `00013_crm`, tạo lịch buổi học hàng loạt có xem trước, `SESSION_UNMARKED` trên "Hôm nay" | PASS (chấm lại theo #20) |
| US-P5-02 | `/attendance` thật: lưới một giao dịch + audit, 409 khi xung đột, hàng đợi khi mất mạng, phát biểu; tool `get_my_attendance` / `get_my_participation` | PASS, sau sửa |
| US-P5-03 | Thời gian học: heartbeat chỉ ghi Redis, rollup idempotent, hộp thoại minh bạch, công tắc theo lớp (`00015_observation`) | PASS, sau 2 vòng sửa |
| US-P5-04 | `/students` thật, at-risk theo tuần (job đêm, `student_risk`), "Lớp cần chú ý" | PASS |
| US-P5-05 | `/students/[id]`: audit mỗi lần mở, ghi chú riêng, nhận xét tổng hợp AI, `Nhắn sinh viên` | PASS, sau sửa |
| US-P5-06 | `/me` thật của sinh viên | PASS (TC-17 chấm theo #38) |
| US-S7-SEED | Seed điểm danh, phát biểu, 10 tuần thời gian học, 5 ticket, kiểm duyệt; `gate-p4.sh`, `gate-p5.sh` | PASS, sau 2 vòng sửa |

**Spec:**
- `docs/specs/FEAT-escalation-notify/` (P4);
- `docs/specs/FEAT-crm-360/` (P5).

Tech Lead thẩm định cả hai trước khi `APPROVED` (P4: 22 TLR, 5 chặn; P5: 14 TLR, 1 chặn).

## Số liệu
- **Test case:** 550 ở mức story (P4 278, P5 + seed 245, CI 27); cổng P4 và P5 theo `gate-P4.md`, `gate-P5.md`; QC đặt 55 câu `Q-QC`.
- **Thông báo và mail (P4):**
  - chuông giảng viên nổi ≤ 5 s kể từ lúc sinh viên gửi câu hỏi (đo bằng hai trình duyệt thật);
  - `TestNotifyAcrossTwoGateways`: sự kiện sinh ở bản B tới người nối ở bản A sau 14 ms;
  - thư trả lời tới Mailpit sau 2 s, đúng 1 thư, 0 dữ liệu sinh viên khác.
- **Điểm danh (P5):** 30 sinh viên ở 375 px theo kịch bản QC 4,4 s; bàn phím 1–4 cả lưới ≈ 3 s; một `PUT` cho cả 30 dòng. Chủ dự án chưa bấm giờ trên điện thoại thật.
- **Thời gian học:** heartbeat k6 10 req/s × 60 s p95 **8,82 ms**, 0 lỗi (ngưỡng 20 ms); Redis chết thì trả `204` sau 0,70–0,76 s.
- **Danh sách sinh viên lớp 1.000 SV:** p95 trang đầu ≈ 49 ms (ngưỡng 300 ms, đo khi máy rảnh); hồ sơ ≤ 3 s; lớp 100 SV không `Seq Scan`.
- **`/me`:** LCP devtools 1.478 ms (ngưỡng 2.500 ms), JS 237 KB.
- **CI:** sau BUG-S7-01, 6/6 lượt nguội xanh, 7 m 43 s – 8 m 44 s mỗi lượt (trước TL-CI #3 #5 ≈ 11,5 phút).
- **Seed trên DB trống:** 416 s; đúng 1 sinh viên `AT_RISK` (C), 2 `WATCH`, 5 ticket đủ trạng thái.
- **Khối lượng:** 164 commit; `backend-go` + `frontend` + `scripts` + `benchmarks` + `seed` thêm 45.927 dòng, bớt 6.712. Gói Go mới: `escalation`, `notify`, `knowledge`, `workhours`, `crm`, `activity`, `seedtool`.
- **Góp ý:** #1–#42, PM quyết hết; Tech Lead thêm TL-1 (ngân sách JS khi thêm tracker).

## Lỗi thật tìm ra và đã sửa
- **Nhận xét AI luôn 503** (P5-05, Cao): task `INSIGHT` mặc định làn `BATCH`, xin làn `NEAR_REALTIME` bị từ chối `BAD_LANE`. Test của dev dùng structurer giả nên không bắt được. Sửa: task `UTILITY`, thêm test đi qua `llm.Gateway` thật.
- **Tri thức không bao giờ được nạp** (P4-05, Cao): `knowledge.sync` ở worker luôn `MASK_FAILED` vì thiếu danh tính lớp cho lời gọi nhúng.
- **Test kiến trúc RAG đỏ** (`TestNoUnscopedChunkQuery`): kho tri thức thêm đường đọc `content_chunks` không theo lớp. Sửa mã, giữ nguyên test.
- **Seed sinh 126 ticket tự mở** (SEED): mọi câu chat mẫu dưới ngưỡng với provider giả, làm `/inbox` 100+ dòng. Sửa: lịch sử chat seed ghi thẳng, đúng 5 ticket.
- **Heartbeat treo 10,4 s rồi 2,3 s khi Redis chết** (P5-03): hạn 500 ms nay bao cả dial / DNS.
- **CI chập chờn:** test chờ đồng hồ thật (11 test), canary hex trùng UUID ngẫu nhiên (#23), thứ tự `map` ngẫu nhiên (#26), rig hợp đồng chờ token sát hạn 1 s (#36); race thật ở `judge.Queue.defaults()` (#4, từ sprint 5).
- **Rò database test** (BUG-S7-02): mỗi lượt full Go để lại ≈ 25 GB database `t_*`, đã làm đầy đĩa colima hai lần. Sửa: drop trong `t.Cleanup`.
- **Câu hỏi ghép** (P5-02): "nghỉ mấy buổi, được cộng bao nhiêu điểm" chỉ trả nửa sau. Sửa: một lần phân loại nhận nhiều ý cá nhân, chạy mọi tool SQL, một lần sinh chữ (D47 giữ nguyên).
- **Nhận diện khủng hoảng bắt nhầm** (lỗi từ sprint 6, TLR-2): "từ từ", "từ hai" bị coi là khủng hoảng. Sửa bằng luật hai tầng, bộ 20 câu dương + 17 câu âm.

## Quyết định trong sprint
- **Chủ dự án chốt (2026-10-13):**
  - duyệt kế hoạch;
  - SLA ticket tính cả ngày T2–T6: nhắc sau 24 h, lên đầu "Hôm nay" + báo Admin sau 72 h;
  - duyệt đổi schema của `00012`–`00015` (các `ALTER`, `knowledge_items`, `student_risk`);
  - ngưỡng escalation mặc định 0,80 (theo PRD);
  - sinh viên tắt được đo thời gian học; giảng viên chọn lúc khởi tạo lớp có cho tắt hay không;
  - các câu [CHỦ DỰ ÁN] còn lại của P4 và P5 theo phương án BA.
- **Từ 2026-10-13 chủ dự án giao PM toàn quyền chốt.** PM đã chốt, chọn phương án bảo vệ dữ liệu chặt hơn:
  - Q25: lớp mới mặc định cho sinh viên tắt đo; TEACHER của lớp chọn tới khi có sinh viên đầu tiên rồi khoá (Admin không chọn);
  - Q21: câu trả lời của giảng viên có tên / MSSV / email của sinh viên khác bị chặn cứng (`422 PII_IN_ANSWER`), không ghi đè;
  - Q23: hiện tên người trả lời cho sinh viên (tiền lệ sprint 6);
  - TC-17: LCP chỉ chấm ở lượt devtools (#38, theo quyết định sprint 5);
  - #42: xoá ba câu mô tả dưới tiêu đề (`/students`, `/class/settings`, `/settings`) theo luật chữ trên UI.
- **Migration:** `00012_escalation_notify`, `00013_crm`, `00014_forum_ai_regen` (#25), `00015_observation`. Không có `00009`.
- **Hợp đồng cũ giữ nguyên** (#20): `skipped` của tạo lịch buổi học vẫn là số; lỗi quá 60 buổi vẫn `422 VALIDATION_FAILED`.
- **Thông báo Threads cũ** giữ ghi cùng giao dịch (#24); đường mới đi qua outbox.

## Quy trình
- **Dev sửa test cũ trước khi PM quyết hai lần** (#10, #21). Nội dung đúng nên PM chấp nhận sau, không hoàn tác. Lần thứ ba (#23) dev tự hoàn tác và chờ.
- **Worktree dùng chung:** BA sửa spec trong worktree của dev; một commit của dev (`803393b7`) mang phần sửa spec #24 / #25 của BA, và BA lỡ rebase nhánh cục bộ của dev rồi đặt lại (không mất commit). Từ đó BA và PM làm ở worktree riêng.
- **Trùng số proposal:** PM ghi nhầm #20 trùng dòng của dev; đổi thành #22, ghi chú trong `proposals.md`.
- QC rà `git log -p` spec / TC từ `APPROVED`: không TC nào bị vai khác sửa; hai commit sửa spec thiếu số được PM giải trình ở #39.

## Nợ
- **Chưa kiểm được trên stack thật:** ép provider lỗi / quá tải cho từng intent (TC BLOCKED ở P4-01, P4-05, P4-06, P5-05; chỉ kiểm bằng test Go).
- **Test thời gian:** `TestJoinFailureTimingEqualized` (sprint 2) đỏ khi chạy `-race` trần có tải; quy trình đúng (`scripts/go-test-all.sh`, TL-3) xanh. Phương án (a) của TL-3 (đếm truy vấn thay đo thời gian) để sprint sau.
- **Lighthouse simulate:** TBT có đuôi dài ngẫu nhiên trên runner; tăng `numberOfRuns` 3 → 5 (ngưỡng giữ). Gói `mock/*` ≈ 17,8 KB còn bị prefetch qua thanh bên; hết khi màn mock cuối cùng được thay (D51).
- `TestMaskConcurrentSameSession`: sửa phòng thủ (#36), nguyên nhân là suy luận.
- At-risk chưa có tín hiệu điểm (`GRADE` neo chờ P6).
- Luật chữ trên UI (#42): đã xoá 6 câu ở màn sprint 7 và `/class/settings`, `/settings`; các màn cũ `/settings/llm`, `/settings/integrations`, khối bảo mật `/settings` còn câu mô tả dưới tiêu đề — quét một lượt ở sprint 8 (`docs/sprints/7/handoff/dev-PM-42.md`).
- Từ sprint 6 còn: vai DB superuser (PR), báo upstream `docling_parse` arm64, thu hồi chia sẻ tài liệu, E1 cần người ngoài đội soạn mẫu.

## Chủ dự án tự kiểm
- https://localhost: stack bản nhánh sau khi dựng lại, seed mới (`pnpm dev`).
- **P4:** hai trình duyệt (`sv.kha@` và `teacher@`): sinh viên hỏi câu ngoài tài liệu rồi bấm "Nhờ giảng viên" → chuông giảng viên nổi → `/inbox` `Nhận`, trả lời → sinh viên thấy "Giảng viên · …" trong chat; thư ở http://localhost:8025, không có dữ liệu người khác.
- **P5:**
  - bấm giờ điểm danh 30 sinh viên ở `/attendance` trên **điện thoại thật** (≤ 60 s);
  - bật chế độ máy bay giữa lúc điểm danh, đánh vắng 3 người, tắt chế độ máy bay: dữ liệu tự lên;
  - `/students` lọc "Cần chú ý": Sinh viên C có mặt, đúng lý do bằng lời;
  - đăng nhập `sv.gioi@`: `/me` không có ghi chú / nhãn rủi ro; `/settings` tắt được đo thời gian học ở lớp cho phép, lớp kia công tắc khoá.

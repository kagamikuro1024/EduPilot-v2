# QC test case — US-P4-06 (Threads: báo cáo, kiểm duyệt, sửa / xoá, ghim, nạp lại tri thức)
Nguồn: `US.md` US-P4-06 AC1…AC16 + `SRS.md` 2, 4.7, 4.10, 6.4, 7.4; `DESIGN.md` §14.3–14.4; FLOWS F4. Hộp đen, không đọc code dev.
Biến như `tc-US-P4-01.md`. `$TH` = id thread của `$SVA` (công khai trong lớp 1); `$P` = id bình luận của `$SVB`. Hôm nay: `GET $GW/api/v1/me/today` (tất cả lớp) hoặc `$Q/today`. Route kiểm duyệt: `$Q/threads/$TH/report` … (SRS 6.4).

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P4-06-01 | AC1 | `$TH` của `$SVA`; `$SVB`, `$SVC`, `$TA_`, `$TCH` | `$SVB` `POST $Q/threads/$TH/report {"reason":"SPAM"}` | `201`; `select status from post_reports` → `OPEN` |
| TC-P4-06-02 | AC1 (một lần mỗi người mỗi bài) | sau TC-01 | `$SVB` báo lần hai (cùng lý do hoặc lý do khác) | `409 ALREADY_REPORTED`; không dòng đôi (`count(*)`=1) |
| TC-P4-06-03 | AC1 (lý do / note) | – | `reason` ngoài enum; `note` > 200 ký tự với `OTHER`; `note` khi `reason=SPAM` | `422` ba ca; `note` chỉ khi `OTHER` |
| TC-P4-06-04 | AC1 (bài của mình) | – | `$SVA` báo `$TH` | `422 CANNOT_REPORT_OWN` |
| TC-P4-06-05 | AC1 (không tồn tại / xoá / lớp khác) | `$TH_xoa` đã xoá mềm; thread ở lớp 2 | `report` | `404` |
| TC-P4-06-06 | AC1 (ADMIN) | – | `$ADM` `report` | `403` |
| TC-P4-06-07 | AC1 (người báo ẩn danh) | có báo cáo | `GET $Q/threads/$TH -H "$SVB"`, `$SVA`; `jq` tìm khoá `report`, `reporters`, `report_count` | sinh viên (kể cả người báo, chủ bài) không thấy ai báo cáo và số báo cáo |
| TC-P4-06-08 | AC1 (hạn mức) | `$SVB` báo 20 bài trong 1 giờ (20 thread khác nhau) | báo thêm bài thứ 21 | `429 RATE_LIMITED` (`REPORT_RATE_PER_HOUR=20`) |
| TC-P4-06-09 | AC1 (bình luận) | `$P` | `$SVA` `POST $Q/threads/$TH/posts/$P/report {"reason":"OFFENSIVE"}` | `201` |
| TC-P4-06-10 | AC2 (3 người → tự ẩn) | `$TH` có 2 báo cáo `OPEN` từ `$SVB`, `$SVC` | `$TA_` (hoặc SV thứ ba) báo | `forum_threads.hidden_at` có, `hidden_by` NULL, `hidden_reason='Nhiều báo cáo, chờ duyệt'` |
| TC-P4-06-11 | AC2 (biến khỏi sinh viên khác) | thread đã tự ẩn | `$SVB`: `GET $Q/threads`, `GET $Q/threads/$TH`, tìm kiếm `?q=`, `GET $Q/threads/{other}/similar`, `GET /me/today` | không có trong danh sách; chi tiết `404`; tìm kiếm không có; "tương tự" không có; Hôm nay không có |
| TC-P4-06-12 | AC2 (chủ bài, giảng viên) | – | `$SVA` mở bài; `$TCH` mở hàng chờ | chủ bài thấy bài với dải "Bài của bạn đang được giảng viên xem xét"; giảng viên thấy "Đã ẩn · chờ duyệt" trong hàng chờ; `POST_REPORTED` gộp cho TA + TEACHER |
| TC-P4-06-13 | AC2 (đếm người khác nhau) | `$SVB` báo 3 bài khác nhau; `$TH2` có 3 báo cáo từ **cùng** `$SVB` (không thể: 409) → thử ba báo cáo ở ba bài không đủ | xem `hidden_at` | `$TH2` có 1 báo cáo; không tự ẩn |
| TC-P4-06-14 | AC2 (chỉ đếm `OPEN`) | `$TH3` có 2 báo cáo `OPEN` + 1 `DISMISSED` (qua `dismiss-reports`) | người thứ 3 báo | **không** tự ẩn (2 `OPEN`) |
| TC-P4-06-15 | AC2 (ngưỡng cấu hình) | – | `grep -rn REPORT_AUTOHIDE_THRESHOLD docker-compose*.yml .env.example backend-go/internal/platform/config` | hằng cấu hình = 3 |
| TC-P4-06-16 | AC3 (hàng chờ) | có thread/bình luận bị báo cáo hoặc ẩn | `GET $Q/moderation/queue?status=OPEN -H "$TCH"`; `status=HIDDEN`; `cursor` | danh sách kèm số báo cáo, các lý do (đếm theo loại), thời điểm; phân trang con trỏ |
| TC-P4-06-17 | AC3 (`Ẩn bài`) | `$TH4` bị báo | `POST …/threads/$TH4/hide {"reason":"…","version":N}`; `reason` rỗng; `reason` 201 ký tự | `200`, `hidden_by` = người ẩn; báo cáo `OPEN` → `ACTIONED`; `422` ×2 |
| TC-P4-06-18 | AC3 (`Bỏ ẩn`, `Giữ bài`) | thread đang ẩn / tự ẩn | `unhide {version}`; `dismiss-reports {version}` | `unhide` → hiện lại; `dismiss-reports` → báo cáo `OPEN` → `DISMISSED`, bài tự ẩn hiện lại |
| TC-P4-06-19 | AC3 (phân quyền) | – | `$SVA`, `$ADM` gọi `hide`, `unhide`, `dismiss-reports`, `moderation/queue`, `pin` | `403` mỗi ca |
| TC-P4-06-20 | AC3 (audit) | – | `select entity,action from audit_log` sau mỗi hành động | mỗi `hide`/`unhide`/`dismiss-reports` 1 dòng |
| TC-P4-06-21 | AC3 (hai giảng viên đồng thời) | `$TH5` cùng `version` | `$TCH` và `$TA_` cùng `hide` (`&`+`wait`) | người sau `409 VERSION_CONFLICT` |
| TC-P4-06-22 | AC4 (chủ bài được báo) | `$TH` tự ẩn rồi giảng viên `Ẩn bài` với lý do "Lạc đề" | `GET /notifications -H "$SVA"`; mở bài | tự ẩn: `POST_HIDDEN` "Bài của bạn đang được giảng viên xem xét"; giảng viên ẩn: "Bài của bạn đã bị ẩn: Lạc đề"; mở bài thấy dải lý do; **không** lộ ai báo cáo; `Bỏ ẩn`/`Giữ bài` → "Bài của bạn đã hiện lại"; Mailpit **không** có thư cho việc này |
| TC-P4-06-23 | AC5 (sửa chạy lại tường lửa PII) | `$SVA` sửa `$TH` | `PUT $Q/threads/$TH {"title":…,"body":"gọi em theo số 0912345678 nhé","tags":…,"version":N}` | `422 PII_DETECTED` kèm lý do; **không** lưu bản thô: `select body from forum_threads` không chứa số điện thoại; (UI: hộp thoại hai lối `Chuyển sang chat riêng` / `Ẩn thông tin rồi lưu`) |
| TC-P4-06-24 | AC5 (sạch → lưu) | – | sửa thân sạch | `200`, `edited_at` có, nhãn "đã sửa" ở UI |
| TC-P4-06-25 | AC5 (canary PII không lưu) | canary PII `CANARY-PII-7Q` (số điện thoại tổng hợp) trong lần sửa bị chặn | quét mọi bảng `forum_*`, `pii_events`, `audit_log`, `outbox` (`pg_dump \| grep -c`) | 0 chỗ chứa canary thô (có thể có bản đã che trong `pii_events` nếu spec cho; không bản thô) |
| TC-P4-06-26 | AC5 (khoá khi đã `VERIFIED`/`CORRECTED`) | thread có bài AI `VERIFIED` | chủ sửa | `409 EDIT_LOCKED_VERIFIED` |
| TC-P4-06-27 | AC5 (người khác / version) | – | `$TCH`, `$TA_`, `$SVB` sửa bài của `$SVA`; sai `version` | `403` ×3; `409` |
| TC-P4-06-28 | AC5 (sửa bài đang ẩn) | thread ẩn | chủ sửa | `200`; `hidden_at` vẫn có (không tự bỏ ẩn) |
| TC-P4-06-29 | AC5 (AI `PENDING`) | thread có bài AI `PENDING` | chủ sửa câu hỏi | bài AI cũ `REJECTED`, `ai_state='PENDING'` (AI sinh lại), **không** thông báo cho chủ bài; sinh viên không thấy bài AI cũ; `AI_PENDING` đếm lại |
| TC-P4-06-30 | AC5 (khủng hoảng ở `PUT`) | – | sửa thân thành "em không muốn sống nữa" | `422 CRISIS_SUPPORT`; không lưu |
| TC-P4-06-31 | AC6 (xoá mềm) | `$TH6` của `$SVA` (có 3 bình luận) | `DELETE $Q/threads/$TH6 -H "$SVA"` | xoá mềm (`deleted_at`); biến khỏi mọi người kể cả giảng viên ở danh sách thường (giảng viên vẫn thấy ở hàng chờ nếu có báo cáo `OPEN`); 3 bình luận ẩn theo |
| TC-P4-06-32 | AC6 (người khác xoá) | – | `$SVB`, `$TCH` `DELETE` | `403` |
| TC-P4-06-33 | AC6 (gỡ tri thức) | thread có tri thức đã nạp | xoá thread | đoạn `content_chunks` tương ứng bị gỡ ≤ 5 s |
| TC-P4-06-34 | AC6 (UI xác nhận) | `playwright-cli` | bấm Xoá | `<ConfirmIrreversible>` ghi "Câu hỏi và N bình luận sẽ bị xoá" |
| TC-P4-06-35 | AC7 (ghim) | `$TCH` | `POST …/threads/$TH7/pin` ×4 thread; `unpin` | 3 ghim đầu `200`, thứ 4 `409 PIN_LIMIT`; `pinned_at` đặt / xoá |
| TC-P4-06-36 | AC7 (danh sách) | 3 ghim | `GET $Q/threads` trang đầu và trang kế | `pinned:[…]` ≤ 3, `items` không lặp ghim; lọc `week`, `tag`, `status=pending\|verified\|none` giữ nguyên |
| TC-P4-06-37 | AC7 (vai) | – | `$SVA`, `$ADM` `pin` | `403` |
| TC-P4-06-38 | AC8 (người đăng được báo) | `$SVA` đăng thread; `$SVB` bình luận | thông báo + Mailpit | `THREAD_REPLY` qua chuông tức thì và thư `thread_reply` (nếu `notify_answer_by_mail`); `$SVA` tự bình luận không tự báo; `VERIFY`/`CORRECT` → `THREAD_VERIFIED`; `REJECT` **không** báo chủ bài |
| TC-P4-06-39 | AC9 (gộp `AI_PENDING`) | 3 bài AI `PENDING` | `GET /notifications -H "$TCH"` | 1 dòng "3 câu trả lời AI chờ duyệt" (N = số bài `PENDING` chưa ẩn), cập nhật tại chỗ; N=0 → tự đọc |
| TC-P4-06-40 | AC10 (`Hôm nay`) | có báo cáo `OPEN` | `GET /me/today -H "$TCH"`; `-H "$SVA"`; `-H "$ADM"` | giảng viên mỗi lớp 1 mục "N bài bị báo cáo chờ xem", `Href` `/threads?view=moderation&course=…`; biến ≤ 60 s sau khi xử lý hết; sinh viên / ADMIN không thấy |
| TC-P4-06-41 | AC11 (`Loại` biến mất, gỡ khỏi RAG) | bài AI `VERIFIED` đã nạp tri thức → `REJECT` (hoặc ẩn / xoá) | `$SVB` hỏi câu tương tự ≤ 5 s sau; `select count(*) from content_chunks where …` | sinh viên không thấy ở endpoint nào; đoạn gỡ ≤ 5 s; `rag` không trả đoạn đó; cache câu trả lời cũ vô hiệu (`ep:rag:ver` tăng) |
| TC-P4-06-42 | AC12 (nạp lại) | giảng viên `VERIFY` bài AI `PENDING` | `select * from knowledge_items, content_chunks`; chờ ≤ 10 s | `knowledge.sync` `Embed` ≤ 10 s; đoạn gồm câu hỏi (đã `Redact`) + câu trả lời; `rag.SearchStudent` trả đoạn với nguồn "Hỏi đáp đã được …" (liên kết `/threads/{id}`) |
| TC-P4-06-43 | AC12 (`CORRECT` hai lần thay đoạn) | – | `CORRECT` hai lần | `knowledge_items` vẫn 1; đoạn thay, không đôi; bản sửa là chuỗi được `Embed` |
| TC-P4-06-44 | AC12 (embedding chết) | dừng provider embedding | `VERIFY` | `VERIFY` vẫn thành công; `AI_PENDING` đếm lại; `today.invalidate` đã chạy (Hôm nay đổi ≤ 60 s) |
| TC-P4-06-45 | AC12 (cô lập lớp) | – | SV lớp 2 hỏi câu tương tự | không nhận đoạn của lớp 1 |
| TC-P4-06-46 | AC13 (ma trận) | – | bảng SRS 2 dòng 11–14: STUDENT/TA/TEACHER `report` ✓, ADMIN `403`; `hide`… `moderation/queue` STAFF ✓, STUDENT/ADMIN `403`; `PUT`/`DELETE` chỉ chủ; sinh viên ngoài lớp / `PENDING` / `REMOVED` → `403`; lớp `ARCHIVED` → `409 COURSE_ARCHIVED` | đúng từng ô |
| TC-P4-06-47 | AC14 (UI) | `playwright-cli` 1280 và 375 | menu `…` của bài: `Báo cáo` là mục menu (không phải nút phụ thứ ba); chọn lý do bằng danh sách; sau báo cáo hiện "Đã báo cáo · Hoàn tác" 5 s; giảng viên thấy `Ẩn bài`/`Ghim`/`Giữ bài`; chủ bài thấy `Sửa`, `Xoá`; `<PageState>`; chữ "Loại khỏi tri thức" (giảng viên) / "Không hiển thị" (sinh viên) | đúng; dùng được ở 375 px (không tràn ngang, vùng chạm ≥ 44 px) |
| TC-P4-06-48 | AC14 (hoàn tác báo cáo) | vừa báo cáo | bấm Hoàn tác < 5 s (`DELETE …/report`); sau 5 s; sau khi tự ẩn | trong 5 s → rút báo cáo; sau 5 s hoặc đã tự ẩn → không hoàn tác |
| TC-P4-06-49 | AC15 (đua báo cáo thứ 3) | 2 báo cáo; hai người cùng báo cùng lúc | `&`+`wait` | đúng **một** lần tự ẩn; đúng một thông báo cho chủ bài (`dedupe_key`) |
| TC-P4-06-50 | AC15 (gửi lại sau mất mạng) | – | `setOffline`, báo, online, gửi lại cùng người | `409 ALREADY_REPORTED` hiện "Bạn đã báo cáo bài này" (không phải lỗi) |
| TC-P4-06-51 | AC15 (xoá giữa chừng) | tác giả xoá bài vừa trước khi giảng viên `Ẩn bài` | `hide` | `404` "Bài này không còn nữa" |
| TC-P4-06-52 | AC16 (bài ẩn không rò) | thread ẩn chứa canary `CANARY-HID-1` | `$SVB`: `GET $Q/threads?q=CANARY-HID-1`, `…/similar`, `/me/today`, số liệu lớp (`GET $Q/insights` nếu có), `AI_PENDING`; grep canary | 0 chỗ lộ; chỉ chủ bài và hàng chờ Staff thấy |
| TC-P4-06-53 | UI | frontend | `bash scripts/ui-antipatterns.sh; echo $?`; `$PW threads.spec.ts -g 'report undo\|owner edit delete\|staff hide pin'`; `$PW a11y.spec.ts -g threads` | `0`; xanh |

## Nhánh lỗi
TC-P4-06-02, 03, 05, 08, 21, 26, 27, 29, 30, 44, 49, 50, 51. Thêm TC-P4-06-E1 `PUT` bình luận của mình có PII → `422 PII_DETECTED`, không lưu; TC-P4-06-E2 `report` bình luận đã xoá → `404`.

## Phân quyền
TC-P4-06-06, 19, 27, 32, 37, 46 (đủ ma trận SRS 2).

## Câu hỏi (Q-QC)
- **Q-QC-P4-16** Route số liệu lớp để kiểm "số liệu không đếm bài ẩn" (AC16) chưa chỉ rõ; QC dùng `/insights` nếu có trên stack, nếu không bỏ phần đó và ghi.
- **Q-QC-P4-17** `GET /me/today` hay `$Q/today`: lấy từ openapi hiện có (`/api/v1/me/today`, `/api/v1/courses/{id}/today`); xác nhận route `POST_REPORTED` xuất hiện ở cả hai.

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-10-13: viết lần đầu.

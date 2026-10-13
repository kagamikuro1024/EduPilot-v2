# QC report — US-P4-06 (Threads: báo cáo, kiểm duyệt, sửa / xoá, ghim, nạp lại tri thức) — vòng 1
Ngày 2026-10-14. Commit kiểm: `sprint/7-p4-p5` @ `2ad5764` (gồm US-P4-06 `fa7caf3`, migration `00014_forum_ai_regen`, #24, #25). Bộ TC: `tc-US-P4-06.md` (không sửa chữ TC). Handoff: `dev-US-P4-06.md`. Stack `edupilot100` dựng lại từ code mới (DB trống, migration `14`, `--scale gateway=2`, provider `fake`, không có tài liệu seed). Seed không có bài AI nên bài AI được chèn bằng SQL (`forum_posts`, `kind='AI'`) rồi xử lý qua API (`verify` / `correct` / `reject`); thông báo "đếm AI chờ" vì vậy chỉ đúng sau thao tác đầu tiên của giảng viên.

## Kết luận: **FAIL** (1 lỗi: TC-42, mức THẤP–TRUNG BÌNH; 1 TC BLOCKED)
Báo cáo, tự ẩn, hàng chờ, ẩn / bỏ ẩn / giữ bài, ghim, sửa / xoá, thông báo, nạp lại / gỡ / thay tri thức, cô lập, phân quyền đều đạt. Hỏng ở AC12: trích dẫn tri thức nguồn `POST` không có liên kết `/threads/{id}`.

## 1. Kết quả TC
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| 01 báo cáo | PASS | `201 {"reported":true}`; `post_reports.status='OPEN'` |
| 02 một lần mỗi người | PASS | lần hai `409 ALREADY_REPORTED`; 1 dòng |
| 03 lý do / note | PASS | `reason` ngoài enum, `note` 201 ký tự (`OTHER`), `note` kèm `SPAM` → `422` ×3 |
| 04 bài của mình | PASS | `422 CANNOT_REPORT_OWN` |
| 05 không tồn tại / lớp khác | PASS | thread không có `404`; giảng viên (thành viên cả hai lớp) báo thread lớp 1 qua URL lớp 2 → `404`; sinh viên không thuộc lớp 2 → `403` (guard lớp) |
| 06 ADMIN | PASS | `403` |
| 07 người báo ẩn danh | PASS | `GET …/threads/{id}` của SVB / SVA: không khoá chứa `report` / `reporter` |
| 08 hạn mức | PASS | một sinh viên báo 20 bài → `201`; bài thứ 21 → `429` |
| 09 bình luận | PASS | `POST …/posts/{pid}/report` → `201` (route `/posts/{pid}/report`, không có `{tid}` trong đường dẫn) |
| 10 ba người → tự ẩn | PASS | SVB, SVC, TA: `hidden_at` có, `hidden_by` NULL, `hidden_reason='Nhiều báo cáo, chờ duyệt'` |
| 11 biến khỏi sinh viên khác | PASS | SVB: danh sách 0, chi tiết `404`, `?q=` 0, `similar` 0, `/me/today` 0 |
| 12 chủ bài, giảng viên | PASS | chủ bài `200` với `hidden:{reason:"Nhiều báo cáo, chờ duyệt",auto:true}`; hàng chờ có bài; `/me/today` giảng viên `POST_REPORTED` "Cũ nhất đã chờ N phút." → `/threads?view=moderation&course=…` |
| 13 đếm người khác nhau | PASS | một người báo 3 bài khác nhau → không bài nào ẩn |
| 14 chỉ đếm `OPEN` | PASS | 2 báo cáo → `dismiss-reports` (2 `DISMISSED`); thêm 2 báo cáo mới → không ẩn (2 `OPEN`) |
| 15 ngưỡng cấu hình | PASS | `REPORT_AUTOHIDE_THRESHOLD` mặc định 3 (`config.go`, khoảng 2–20) |
| 16 hàng chờ | PASS | `?status=OPEN&limit=2` → 2 mục + `next_cursor`, trang 2 có 2; `HIDDEN` 5; mục có `reports:{total,by_reason}`, `hidden_at`, `created_at` |
| 17 `Ẩn bài` | PASS | `reason` rỗng / 201 ký tự `422`; hợp lệ `200`, `hidden_by` = giảng viên, báo cáo `OPEN` → `ACTIONED` |
| 18 `Bỏ ẩn`, `Giữ bài` | PASS | `unhide` → hiện lại; `dismiss-reports` bài tự ẩn → hiện lại, báo cáo `DISMISSED` |
| 19 phân quyền | PASS | SVA, ADMIN: `hide`, `unhide`, `dismiss-reports`, `moderation/queue`, `pin` → `403` |
| 20 audit | PASS | `audit_log` `forum_thread` / `thread.hide`, `thread.unhide`, `thread.dismiss_reports`, `thread.auto_hide`, mỗi hành động một dòng |
| 21 hai giảng viên đồng thời | PASS | TCH `200`, TA `409 VERSION_CONFLICT` |
| 22 chủ bài được báo | PASS | `POST_HIDDEN` "Bài của bạn đang được giảng viên xem xét" (tự ẩn); "Bài của bạn đã bị ẩn: Lạc đề" (giảng viên ẩn); `POST_UNHIDDEN` "Bài của bạn đã hiện lại"; chi tiết có `hidden:{reason,auto:false}`, không lộ người báo; Mailpit 0 thư cho việc này |
| 23 sửa qua tường lửa PII | PASS | `PUT` thân có số điện thoại → `422 PII_DETECTED` `{reasons:[PHONE], redacted_text:"gọi em theo số [đã ẩn] nhé …"}`; DB thô 0 |
| 24 sạch → lưu | PASS | `200`, `edited_at` có, `edited:true` |
| 25 canary PII không lưu | PASS | `pg_dump` `forum_threads`, `forum_posts`, `pii_events`, `audit_log`, `outbox`: 0 chỗ chứa số điện thoại thô |
| 26 khoá khi `VERIFIED` | PASS | `409 EDIT_LOCKED_VERIFIED` |
| 27 người khác / version | PASS | TCH, TA, SVB sửa → `403` ×3; `version` sai → `409` |
| 28 sửa bài đang ẩn | PASS | `200`; `hidden_at` vẫn có |
| 29 AI `PENDING` | PASS | bài AI cũ `REJECTED`; `ai_state='PENDING'` (xếp `thread.edited` + `job.enqueue`); chủ bài không nhận thông báo; sinh viên khác không thấy bài AI cũ |
| 30 khủng hoảng ở `PUT` | PASS | `422 CRISIS_SUPPORT` kèm `details.actions:["OPEN_PRIVATE_CHAT"]`; không lưu |
| 31 xoá mềm | PASS | chủ xoá `204`, `deleted_at`; giảng viên: danh sách 0, chi tiết `404`; 3 bình luận ẩn theo |
| 32 người khác xoá | PASS | SVB, TCH `403` |
| 33 gỡ tri thức | PASS | xoá thread: `knowledge_items` còn 0 ngay (≤ 1 s) |
| 34 UI xác nhận xoá | PASS (qua e2e của dev) | `threads-moderation.spec.ts` "owner edit delete" xanh (30 passed); QC không tự mở hộp thoại bằng tay |
| 35 ghim | PASS | 3 ghim đầu `204`, thứ 4 `409 PIN_LIMIT`; `unpin` giảm còn 2 |
| 36 danh sách | PASS | `pinned` 3, `items` không lặp ghim, trang kế `pinned` rỗng |
| 37 vai ghim | PASS | SVA, ADMIN `403` |
| 38 người đăng được báo | PASS | bình luận người khác → `THREAD_REPLY` +1 và thư `thread_reply`; tự bình luận không báo; `VERIFY` → `THREAD_VERIFIED`; `REJECT` → 0 thông báo |
| 39 gộp `AI_PENDING` | PASS | sau thao tác của giảng viên: một dòng "2 câu trả lời AI chờ duyệt" (3 bài `PENDING`, 1 `verify`) |
| 40 `Hôm nay` | PASS | giảng viên có `POST_REPORTED`; sinh viên, ADMIN 0 |
| 41 `Loại` gỡ khỏi RAG | PASS | `reject` bài AI đã nạp: `200`, `knowledge_items` còn 0 sau 0,1 s |
| 42 nạp lại (`VERIFY`) | **FAIL** | `VERIFY`: `content_chunks` có `embedding` ngay, văn bản "Câu hỏi: … Trả lời: …"; sinh viên khác hỏi đúng chuỗi → trích dẫn `{"title":"Hỏi đáp đã được giảng viên xác nhận","knowledge":true,"snippet":…}` **không có liên kết `/threads/{id}`** (khoá chỉ gồm `n, document_id, title, page_no, snippet, knowledge`). SRS TLR-12: nguồn `POST` → liên kết `/threads/{id}` (nguồn `TICKET` không liên kết) |
| 43 `CORRECT` hai lần | PASS | `knowledge_items` vẫn 1; văn bản = bản sửa lần hai (`QCCORR2`); `embedding` có |
| 44 embedding chết | **BLOCKED** | không ép được lỗi `Embed` ở stack fake (`error_rate` không tác động `Embed`; Q-QC-P4-15 chưa có trả lời) |
| 45 cô lập lớp | PASS | sinh viên lớp 2 hỏi đúng chuỗi: 0 trích dẫn tri thức |
| 46 ma trận | PASS | `report`: SVB, TA, TCH `201`; ADMIN, `PENDING`, ngoài lớp `403`. `hide` / `moderation/queue`: SVA, ADMIN, `PENDING`, ngoài lớp `403`. `PUT` / `DELETE`: TCH, TA, SVB, `PENDING` `403`. Lớp `ARCHIVED` → `409 COURSE_ARCHIVED` |
| 47 UI | PASS | 1280 và 375: nút "Tuỳ chọn bài" → menu `Báo cáo`; biểu mẫu lý do tại chỗ; sau báo cáo "Đã báo cáo · Hoàn tác" (không toast); 375 px: nút 44×44, không tràn ngang. Nút menu ở 1280 là 32×32 (chỉ yêu cầu ≥ 44 px ở 375) |
| 48 hoàn tác báo cáo | PASS | `DELETE …/report` < 5 s → `204`, 0 báo cáo còn, báo lại `201`; UI "Hoàn tác" gọi đúng `DELETE` |
| 49 đua báo cáo thứ 3 | PASS | hai người báo đồng thời: bài tự ẩn một lần, outbox `thread.hidden` = 1, thông báo chủ bài `POST_HIDDEN` = 1 |
| 50 gửi lại sau mất mạng | PASS (API) | báo lần hai cùng người → `409 ALREADY_REPORTED`, thông điệp "Bạn đã báo cáo bài này."; chữ UI qua e2e của dev |
| 51 xoá giữa chừng | PASS | `hide` sau khi chủ xoá → `404` `NOT_FOUND` "Không tìm thấy." (TC viết chữ UI "Bài này không còn nữa"; API trả thông điệp chung) |
| 52 bài ẩn không rò | PASS | `CANARY-HID-1`: `?q=`, danh sách, `similar`, `/me/today`, thông báo, log gateway = 0 chỗ lộ (SVB); chủ bài `200`, hàng chờ giảng viên 1; `insights` không có route (`404`) |
| 53 giao diện | PASS | `ui-antipatterns.sh` rc=0; `playwright threads.spec.ts threads-moderation.spec.ts a11y.spec.ts` (desktop + mobile): 30 passed, 30 skipped, 0 failed |

Tổng 53 TC: PASS 51, FAIL 1 (TC-42), BLOCKED 1 (TC-44).

## 2. Lỗi
**BUG-1 (THẤP–TRUNG BÌNH) TC-42 / AC12:** trích dẫn của đoạn tri thức nguồn `POST` thiếu liên kết `/threads/{id}`. Tái hiện: bài AI `PENDING` → giảng viên `POST …/posts/{pid}/verify`; sinh viên khác hỏi đúng chuỗi của đoạn ("Câu hỏi: … Trả lời: …"); đọc `citations` ở khung SSE `done`: `knowledge:true` nhưng không có khoá trỏ tới thread. (Dev nêu ở handoff P4-05: "nguồn POST có liên kết thuộc US-P4-06"; handoff P4-06 không ghi đã làm.)

## 3. Lệnh của handoff
| Lệnh | Kết quả |
| --- | --- |
| `go test -race -tags testroutes ./internal/thread ./internal/today ./internal/store ./internal/contract/... ./internal/httpapi/...` | ok (thread 71 s, today 106 s, store 64 s, contract 85 s, httpapi, httpx, sse) |
| `sqlc diff` | rc=0 |
| `playwright threads.spec.ts threads-moderation.spec.ts` (desktop + mobile) | xanh (30 passed) |
| `ui-antipatterns.sh`; `pnpm lint`; `pnpm build` | rc=0; sạch; rc=0 |

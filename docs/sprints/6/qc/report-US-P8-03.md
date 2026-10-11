# QC report — US-P8-03 (lịch gộp, feed ICS, nhắc 24 giờ, tool lịch)  · Kết luận: PASS phần đã kiểm (TC nặng / gate → ghi dưới)

Handoff: `docs/sprints/6/handoff/dev-US-P8-03.md` (HEAD `fbad802`; migration `00011_calendar` theo #13). Bộ TC: `tc-US-P8-03.md` (60 TC; TC viết theo spec v1.2, đối chiếu v1.4).
**Môi trường:** gateway + worker + frontend build từ HEAD, DB riêng `qc_p801`, Redis db 9, Mailpit dùng chung (cổng 1025/8025), `REMINDER_TICK=5s` để thấy nhịp nhắc; seed `scripts/seed.mjs` (30 SV mỗi lớp); `playwright-cli` 375 px. Không đụng stack s55.

## Cổng đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `go test -count=1 -race ./internal/calendar ./internal/store ./db ./internal/user ./internal/today ./cmd/worker` | PASS |
| `go test -race -tags testroutes ./internal/contract ./internal/agent` | PASS |
| goose up → down → up (`00011`) trên DB trống | PASS (version 11) |
| `grep -ci "alter table" 00011_calendar.sql` | 2 — chỉ `users ADD/DROP CONSTRAINT users_ics_token_hash_chk` (đề xuất #6 PM đã chấp nhận); 0 ALTER khác |

## TC
| TC | Kết quả | Chứng cứ |
| --- | --- | --- |
| 01–03 | PASS | `calendar_events` (id, course_id, type enum, title, starts_at, ends_at, location, description, ref_type/ref_id, created_by, version, …), `reminder_log` (UNIQUE `user_id, source_type, source_id, starts_at, kind`), `CHECK (ics_token IS NULL OR ics_token ~ '^[0-9a-f]{64}$')` |
| 04 | PASS | lịch gộp có 3 nguồn `class_session`, `weekly_exam`, `calendar_event`; `calendar_events` chỉ chứa 3 hàng QC tạo — buổi học / bài thi **không** bị nhân bản vào bảng |
| 05 | PASS | bài thi `DRAFT` không hiện, `SCHEDULED` hiện |
| 06 | PASS | thiếu `from`/`to` → `422`; khoảng 63 ngày → `422`; 62 ngày → `200` |
| 07 | PASS | con trỏ qua 5 trang `limit=2`: 10 mục không lặp / sót = lấy một lần; `limit=101` → `422` |
| 08 | PASS (một phần) | SV ngoài lớp (`sv31`) `403`; mục lớp khác không lẫn |
| 09 | PASS | `POST …/calendar/events` `201` (version 1); `PUT` đúng `version` → `200` (version 2); audit `calendar.event.create`, `calendar.event.update`; outbox `calendar.changed` phát ra |
| 10, 11, 14, 17, 41(phần), 42, 55 | KHÔNG KIỂM ĐƯỢC | lớp lưu trữ + ghi (test Go có `TestEventArchivedCourse`), Hoàn tác 5 s trên UI, bắt đầu lượt thi để đổi ETag (bài thi chưa mở nên `EXAM_NOT_OPEN`), `ep:today` sau khi đổi giờ, 1.000 SV, dừng Mailpit |
| 12, 13 | PASS | `ETag: W/"…"`; `If-None-Match` → `304`; sửa sự kiện → `200` (ETag đổi) |
| 15 | PASS | SV thấy `personal_state` của chính mình; SV khác `NOT_STARTED`; chưa dựng lượt nộp |
| 16 | PASS | quét thân lịch: 0 chuỗi `score/điểm/question/đáp án/attempt_id/@edupilot` |
| 18 | PASS | `POST /me/calendar/ics-token` `201`, `Cache-Control: no-store`, token 43 ký tự; DB chỉ có `hex(sha256)` (khớp `^[0-9a-f]{64}$`, khác token thô); `GET` chỉ `{exists:true}` |
| 19 | PASS | tạo lần hai (xoay): link cũ → `404`, link mới → `200` |
| 20 | PASS | `DELETE` `204` (lần hai `204`), cột NULL, feed → `404` |
| 21 | PASS | 0 bản ghi `users` chứa token thô; 0 dòng log gateway chứa token; mọi `ics_token` khác NULL đều hex 64 |
| 22 | PASS (một phần) | `/calendar` 375 px: nút `Thêm vào lịch` mở vùng nhỏ (không màn riêng), có "Mất liên kết thì đặt lại" ở đúng một chỗ, `Đặt lại liên kết` mở hộp xác nhận nêu hậu quả ("Liên kết cũ ngừng hoạt động; lịch đã đăng ký ở Google, Outlook hay iPhone sẽ không cập nhật…"); chưa quan sát URL hiện một lần sau xác nhận trên UI (API đã PASS) |
| 23 | PASS | `curl` feed: `200 text/calendar; charset=utf-8`, dòng đầu `BEGIN:VCALENDAR`, cuối `END:VCALENDAR`, 21 `VEVENT`, chỉ CRLF |
| 24 | PASS | có `UID, DTSTAMP, DTSTART, DTEND, SUMMARY, LOCATION`; UID ổn định giữa hai lần gọi |
| 25 | PASS (với giới hạn 120) | tiêu đề có dấu 119 ký tự = 170 byte: gập dòng (4 chỗ), mọi dòng ≤ **75 octet**, không cắt giữa UTF-8, bỏ gập đúng nguyên văn; xuống dòng trong `LOCATION` thoát thành `\n`. **TC viết tiêu đề 200 ký tự nhưng SRS giới hạn `title` ≤ 120 (422)** → TC sai, không phải lỗi dev |
| 26 | PASS | 0 dòng `DESCRIPTION`, không lộ "Ghi chú riêng ZZPRIV"; bài `DRAFT` không có; `SUMMARY` có tiền tố `{mã lớp} · ` (SV thuộc 2 lớp) |
| 27 | PASS | `Cache-Control: private, max-age=300`, `ETag` → `If-None-Match` `304` |
| 28 | KHÔNG KIỂM ĐƯỢC | cần URL công khai để đăng ký vào Google Calendar |
| 29 | PASS | 61+ yêu cầu / phút / IP: các lần đầu `200`, sau đó `429` với `Retry-After: 60` |
| 54 | PASS | không `token`, token bịa, token xoay, token đã thu hồi: cùng `404` và cùng thân `{"code":"NOT_FOUND",…}` |
| 58 | PASS | gọi `ics-token` kèm `user_id` của người khác: không đổi token người khác (token vẫn theo JWT); mỗi SV một token |
| 59 | PASS | feed của `sv.kha` (1 lớp) khác feed của `sv.gioi` (2 lớp); không lẫn |
| 30 | PASS | "Khi nào em thi cuối kỳ?" → `EXAM_SCHEDULE`, khối `exam_schedule` liệt kê bài thi (kể cả `calendar_event` loại `EXAM`), giờ "Thứ Hai, 12/10 · 10:57" theo `Asia/Ho_Chi_Minh` |
| 31 | PASS (một phần) | "Tuần này có gì?" → `UPCOMING_EVENTS` `days=7` gồm buổi học, sự kiện; `days` kẹp 1–30 do test Go (`TestUpcomingEventsDaysClamped`). **Câu "30/60 ngày tới có gì?" và "Tuần tới có gì không?" rơi vào `COURSE_QA`** (xem lỗi) |
| 32 | PASS (một phần) | lớp 2 không bài thi: "Chưa có lịch thi nào được công bố." (0 lời gọi sinh); "Tuần tới có gì không?" → `COURSE_QA` |
| 33 | PASS | khối chỉ gồm sự kiện của lớp đang chat (lớp 1), không lẫn lớp 2 |
| 34 | PASS | sự kiện `OTHER` bắt đầu sau 10 giờ → 29 dòng `reminder_log` + 29 chuông `REMINDER`; thư "Sắp đến giờ: sự kiện trong 24 giờ tới" gửi qua Mailpit, nội dung chỉ tên sự kiện + giờ "16:59 11/10/2026 (giờ Việt Nam)" + một liên kết `…/calendar` |
| 35 | PASS | nhiều nhịp 5 s liên tiếp (hàng chục nhịp): `reminder_log` vẫn đúng `2 sự kiện × 30 SV = 60`, chuông 60, **không** trùng |
| 36 | PASS | đổi giờ sang giờ khác vẫn trong 24 h: sinh thêm một bộ nhắc cho `starts_at` mới (29 → 58 dòng), đúng quy tắc UNIQUE gồm `starts_at` |
| 37 | PASS | xoá sự kiện trong cửa sổ nhắc → 0 dòng `reminder_log` cho nguồn đó |
| 38 | PASS | 100% người nhận là `STUDENT` (60/60 theo `users.role`); không TA / TEACHER / PENDING |
| 39 | PASS | `GET /me/settings` có `reminders {exam:true, class_session:false, other:true}` mặc định; SV tắt `other` → không nhận nhắc `OTHER` |
| 40 | PASS | mail chỉ cho người bật `remind_deadline_by_mail`; (mặc định bật cho mọi tài khoản seed) |
| 43 | PASS | nội dung thư xem 34 |
| 44, 45, 49 | PASS (một phần) | `/calendar` 375 px: mặc định **Danh sách**, có radio `Tuần / Tháng / Danh sách`; không cuộn ngang; mọi điều khiển ≥ 44 px trừ link ẩn "Bỏ qua điều hướng"; màu đỏ cho bài thi trong 48 giờ chưa kiểm |
| 46, 47, 48 | KHÔNG KIỂM ĐƯỢC | giao diện Staff (`Thêm sự kiện`), công tắc ở `/settings`, trạng thái rỗng / lỗi 500 / chậm chưa chạy |
| 50, 51 | CHUYỂN → US-P3-08 | `gate-p8.sh`, seed/ `check-docs-seed.mjs` chưa có |
| 52 | PASS | `ends_at < starts_at`, `title` rỗng, `starts_at` lệch > 2 năm, `type` lạ → đều `422`; `title` > 120 → `422 "Tên sự kiện dài 1–120 ký tự."` |
| 53 | PASS | `PUT` với `version` cũ → `409 VERSION_CONFLICT` kèm bản hiện hành (`current`) |
| 56, 57 | PASS | lịch: SV / TA / TEACHER `200`, ADMIN `403`; tạo sự kiện: SV `403`, ADMIN `403`, TA `201` |
| 60 | PASS (một phần) | A (`sv.gioi`) không thấy `personal_state` của B qua lịch; chưa dựng lượt thi đang làm |

## AC
AC1–AC13, AC15 PASS ở phần kiểm được; AC14 PASS một phần (375 px, mặc định danh sách); **AC16, AC17** chuyển US-P3-08.

## Lỗi
- **BUG-1 (Thấp)** — định tuyến intent bỏ sót cách hỏi tự nhiên về lịch: `"Tuần tới có gì không?"`, `"30 ngày tới có gì?"`, `"60 ngày tới có gì?"` → `COURSE_QA` (trả "Mình chưa tìm thấy nội dung này…") thay vì `UPCOMING_EVENTS`. SRS 4.5 chỉ liệt kê "sắp tới, tuần này, hạn nộp, lịch học", nên đúng chữ spec; đề nghị BA thêm "tuần tới / N ngày tới" nếu muốn bao quát.
- Ghi chú 1: TC-25 viết `title` 200 ký tự nhưng SRS giới hạn 120 (TC sai; QC sẽ sửa TC khi PM cho phép, ghi proposal).
- Ghi chú 2: mail nhắc bật mặc định cho mọi tài khoản (`remind_deadline_by_mail=true`) — đúng `user_settings` hiện có, nhưng TC-40 viết mặc định tắt.
- Ghi chú 3: dev khai `useCalendar` tải mọi trang `limit=100` của khoảng xem (nợ hiệu năng khi tháng > 100 sự kiện).

## Đề nghị
PASS phần kiểm được. Chuyển TC nặng (10, 11, 14, 17, 42, 46–48, 55, 28) và AC16/17 sang lượt gate P8 / US-P3-08. BUG-1 chờ BA.

---
## Chấm lại BUG-1 sau #15 (`c5d42d3`, stack dev, HEAD `743de90`) — **BUG-1 ĐÓNG, US-P8-03 PASS**
Chat riêng, `sv.gioi`/lớp 761987, `fake`: mọi câu đi đúng `intent=UPCOMING_EVENTS` (cột `chat_messages.intent`), ngữ cảnh mang đúng cửa sổ ngày.
| Câu hỏi | Intent | `days` |
| --- | --- | --- |
| Tuần tới có gì không? | UPCOMING_EVENTS | 7 |
| Tuần sau có lịch gì? | UPCOMING_EVENTS | 7 |
| 7 ngày tới có gì không? | UPCOMING_EVENTS | 7 |
| 30 ngày tới có gì? | UPCOMING_EVENTS | **30** |
| Sắp tới có lịch gì? | UPCOMING_EVENTS | 7 |
| Hôm nay có gì? | UPCOMING_EVENTS | 7 (ghi chú: cửa sổ 7 ngày chứ không 1 ngày; không vi phạm AC đã ghi) |
Phiên thử đã xoá khỏi DB dev.

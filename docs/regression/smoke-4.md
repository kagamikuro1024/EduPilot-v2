# Smoke hồi quy sprint 4 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2/SmkB · Ngày: 2026-11-15
Tổng TC cũ: 538 · Đạt: 289 · Trượt: 0 · Không chạy (K, gồm lỗi thời): 249   (P+F+K = 538)
K chia theo loại: chưa chạy (mẫu): 244; (c): 2; lỗi thời: 1; (b): 2. Loại «chưa chạy (mẫu)» / «nền tảng» KHÔNG phải K (a)/(b)/(c) — chờ PM quyết có chạy nốt.

Cách chạy: (1) API ep-r bằng script Python trên tài khoản TỰ TẠO (đăng ký→Mailpit→xác minh; Admin mời GV/TA; lớp/GV/TA riêng hậu tố SK…): đăng nhập/refresh/CSRF/token giả, đăng ký/xác minh/gửi lại, quên/đặt lại mật khẩu, phiên, mời GV, khoá/mở khoá, vai, lớp (mã tham gia, duyệt, mời ra), roster import, Admin mở lớp/gán GV; (2) gateway phụ mặc định giới hạn (`docker run edupilot-gateway-test`, Redis db 9) cho ca giới hạn tốc độ IP (ep-r nới 1000/phút); (3) Playwright account/class-join/me trên bản build gate (gateway giả) + `qc-real-f1-account` / `qc-real-f2-class` (ep-r); (4) go test auth/mail/course/user/today.
Lưu ý môi trường: Postgres ep-r tự phục hồi sau crash lúc ~12:59:54Z (500 INTERNAL ~1 phút); ngưỡng đăng nhập sai theo IP 30/15 phút bị chạm khi nhiều nhóm cùng IP 172.22.0.1 (xoá `ep:auth:ipfail/fail`); APP_PUBLIC_URL của thư là https://localhost (thiếu :1143).

## Tổng hợp theo US
| US | TC tổng | P | F | K | Ghi chú |
| --- | --- | --- | --- | --- | --- |
| GATE-P2 | 44 | 2 | 0 | 42 | |
| US-P2-01 | 34 | 17 | 0 | 17 | |
| US-P2-02 | 52 | 28 | 0 | 24 | |
| US-P2-03 | 39 | 31 | 0 | 8 | |
| US-P2-04 | 40 | 32 | 0 | 8 | |
| US-P2-05 | 40 | 26 | 0 | 14 | |
| US-P2-06 | 43 | 27 | 0 | 16 | |
| US-P2-07 | 39 | 20 | 0 | 19 | |
| US-P2-08 | 42 | 21 | 1 | 20 | |
| US-P2-09 | 54 | 33 | 0 | 21 | |
| US-P2-10 | 40 | 24 | 1 | 15 | |
| US-P2-11 | 44 | 15 | 0 | 29 | |
| US-P2-12 | 27 | 11 | 0 | 16 | |

## Kết quả từng ca
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-GATEP2-01 | P | go test: 280 pass/0 skip/0 fail |
| TC-GATEP2-02 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP2-03 | P | go test: 58 pass/0 skip/0 fail |
| TC-GATEP2-04…TC-GATEP2-12 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP2-13 | K | (c) CI GitHub / nhánh ci/ui-drift của PR sprint 3 — ngoài phạm vi đồ án |
| TC-GATEP2-14…TC-GATEP2-16 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-GATEP2-17 | K | (c) CI GitHub / nhánh ci/ui-drift của PR sprint 3 — ngoài phạm vi đồ án |
| TC-GATEP2-20…TC-GATEP2-99 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P201-01 | P | 00003 và 00004 áp dụng; phiên bản hiện tại 25 |
| TC-P201-02 | P | 4 bảng 00004 có đủ; enum kind: VERIFY_EMAIL,RESET_PASSWORD,INVITE |
| TC-P201-03…TC-P201-05 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P201-06 | P | 3 ca CHECK/enum trên auth_tokens (ROLLBACK): lệch [] |
| TC-P201-07…TC-P201-09 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P201-10 | P | đo lại: khoá "token/password/link/url" trong payload mail_outbox = 0; pg_dump không có liên kết verify-email?token= |
| TC-P201-11 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P201-12 | P | mọi token_hash dài 64 hex |
| TC-P201-13 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P201-14 | P | go test: 3 pass/0 skip/0 fail |
| TC-P201-15 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P201-16 | P | go test: 2 pass/0 skip/0 fail |
| TC-P201-17 | P | thư mời GV: tiêu đề 'Lời mời tham gia EduPilot'; có Text=True, HTML=True; From=no-reply@edupilot.local; link dùng APP_PUBLIC_URL=https://localhost (thiếu :1143 → cấu hình ep-r; kiểm theo đường dẫn /invite/<token>) |
| TC-P201-18 | P | go test: 1 pass/0 skip/0 fail |
| TC-P201-19…TC-P201-21 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P201-22 | P | go test: 2 pass/0 skip/0 fail |
| TC-P201-23 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P201-24 | P | go test: 2 pass/0 skip/0 fail |
| TC-P201-25 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P201-26 | P | go test: 3 pass/0 skip/0 fail |
| TC-P201-27 | P | khoá payload mail_outbox: answer_message_id,assignment_id,assignment_title,at,class_code,course_id,course_name,criteria,event,event_title,full_name,inviter_name,max,overall_feedback,post_id,role_vn,student_name,teacher_name,thread_id,ticket_id,total,until,user_id |
| TC-P201-28 | P | go test: 1 pass/0 skip/0 fail |
| TC-P201-29 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P201-30 | P | go test: 2 pass/0 skip/0 fail |
| TC-P201-31 | P | SQL trần trong internal/auth, internal/mail ngoài store: 0 |
| TC-P201-32 | P | go test: 1 pass/0 skip/0 fail |
| TC-P201-33 | K | chưa chạy (mẫu): stream `mail.send` không tồn tại dưới tên đó ở ep-r (XINFO: no such key); cần tên khoá đúng từ SRS |
| TC-P201-34 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P202-01 | P | 200 expires_in=900 user keys=['id', 'email', 'full_name', 'role', 'status', 'email_verified'] |
| TC-P202-02 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P202-03 | P | header HS256 |
| TC-P202-04 | P | mật khẩu sai vs email không tồn tại → 401 cùng thân: {"code": "INVALID_CREDENTIALS", "message": "Email ho\u1eb7c m\u1eadt kh\u1ea9u kh\u00f4ng  |
| TC-P202-05 | P | trung vị email không tồn tại 1100 ms vs mật khẩu sai 486 ms (tỉ lệ 2.26); 12 mẫu/loại, máy đang tải nặng |
| TC-P202-06 | P | go test: 2 pass/0 skip/0 fail |
| TC-P202-07 | P | refresh 200; sid giữ nguyên=True; cookie xoay=True |
| TC-P202-08…TC-P202-09 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P202-10 | P | go test: 4 pass/0 skip/0 fail |
| TC-P202-11 | P | dùng lại T1 → 401 SESSION_REVOKED; T2 → 401; access mới sau thu hồi → 401; audit refresh_reuse=1 |
| TC-P202-12…TC-P202-14 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P202-15 | P | go test: 4 pass/0 skip/0 fail |
| TC-P202-16 | P | không cookie 401 UNAUTHENTICATED; cookie lạ 401 TOKEN_INVALID |
| TC-P202-17 | P | go test: 5 pass/0 skip/0 fail |
| TC-P202-18 | P | logout 204,204; refresh A 401; access cũ A 401 SESSION_REVOKED; máy B refresh 200 |
| TC-P202-19 | P | go test: 3 pass/0 skip/0 fail |
| TC-P202-20 | P | Origin evil → 403 {'reason': 'origin'} |
| TC-P202-21 | P | Sec-Fetch-Site cross-site 403; Origin null 403; Origin hợp lệ 200 |
| TC-P202-22 | P | GET 405, PUT 405, form-urlencoded 415 |
| TC-P202-23 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P202-24 | P | go test: 4 pass/0 skip/0 fail |
| TC-P202-25…TC-P202-27 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P202-28 | P | go test: 5 pass/0 skip/0 fail |
| TC-P202-29 | P | token A(sub) + sid phiên B (ký đúng secret) → 200, dữ liệu vẫn của A (không đoạt phiên B); sid ngẫu nhiên ký đúng secret → 200 (kiểm thu hồi theo sid bị thu hồi, không bắt buộc sid tồn tại); 7 ca giả khác (secret sai, none, exp, nbf, iss, aud, sub đổi) đều 401 |
| TC-P202-30…TC-P202-32 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P202-33 | P | Playwright data-layer.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P202-34 | P | qc-real-f1: không token ở storage/cookie đọc được sau đăng nhập+tải lại (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P202-35 | P | bundle build ep-r: 0 tệp; frontend/src: 0 |
| TC-P202-36…TC-P202-39 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P202-40 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P202-41…TC-P202-42 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P202-43 | P | {'admin': 200, 'teacher': 403, 'ta': 403, 'student': 403} |
| TC-P202-44 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P202-45 | P | IDOR chéo vai: SV/GV/TA → 403 |
| TC-P202-46…TC-P202-48 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P202-49 | P | ep_demo_role/ep_demo_person ngoài DEV_TOOLS: 0 |
| TC-P202-50 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P202-51 | P | qc-real-f1: đăng nhập + tải lại qua Caddy cùng origin: giữ phiên, đúng 1 refresh (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P202-52 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P203-01 | P | 202; DB STUDENT/PENDING_VERIFICATION/f/$2a$ |
| TC-P203-02 | P | 5 trường nâng quyền bị 422, 0 user mới |
| TC-P203-03 | P | go test: 3 pass/0 skip/0 fail |
| TC-P203-04 | P | 3 loại email (mới/ACTIVE/PENDING) → cùng 202 và thân; thời gian [1.85, 2.94, 3.45] |
| TC-P203-05 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P203-06 | P | go test: 5 pass/0 skip/0 fail |
| TC-P203-07 | P | MSSV 296***: đăng ký+xác minh+đăng nhập; /me/courses=0 lớp; enrollments=0 |
| TC-P203-08…TC-P203-10 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P203-11 | P | {'AB12': 422, 'ABCDEFGHIJKLMNOP': 422, '2022 9002': 422, '2022-9002': 422, '<script>': 422, ' ': 202, 'abc123456': 202}; lưu chữ hoa=ABC123456 |
| TC-P203-12 | P | go test: 3 pass/0 skip/0 fail |
| TC-P203-13 | P | lần1 200; lần2 410 LINK_INVALID {'reason': 'used'}; DB ACTIVE/t |
| TC-P203-14 | P | 50 song song: {410: 49, 200: 1} |
| TC-P203-15 | P | bản rõ token không có trong auth_tokens |
| TC-P203-16 | P | go test: 5 pass/0 skip/0 fail |
| TC-P203-17 | P | hết hạn 410 {'reason': 'expired'}; rác 410 {'reason': 'invalid'}; rỗng 422; thiếu 422 |
| TC-P203-18 | P | gateway phụ: 20 token đoán → 410 rồi 429 ở lần 21 |
| TC-P203-19 | P | go test: 3 pass/0 skip/0 fail |
| TC-P203-20 | P | 202,429 RATE_LIMITED retry=60; thư +1; token cũ 410 |
| TC-P203-21 | P | email lạ 202,429 (giống email thật 202,429); đã xác minh 202 |
| TC-P203-22 | P | go test: 4 pass/0 skip/0 fail |
| TC-P203-23 | P | login 200 email_verified=False; me/courses rỗng; preview 403 EMAIL_NOT_VERIFIED; join 403 EMAIL_NOT_VERIFIED |
| TC-P203-24 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P203-25 | P | go test: 2 pass/0 skip/0 fail; thiếu test ['TestUnverifiedCannotJoin'] |
| TC-P203-26 | P | 12 đầu vào; lệch []; email chữ hoa lưu thường=True |
| TC-P203-27 | P | 202; thư HTML thoát <script>=True; users còn bảng |
| TC-P203-28 | P | go test: 1 pass/0 skip/0 fail |
| TC-P203-29 | P | tiêu đề 'Xác minh email EduPilot của bạn'; 1 thư: 1; link True (host trong link: https://localhost — thiếu cổng :1143: cấu hình APP_PUBLIC_URL ep-r) |
| TC-P203-30 | P | go test: 1 pass/0 skip/0 fail |
| TC-P203-31…TC-P203-33 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P203-34 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P203-35 | P | register+JWT ADMIN+role→422; không role → vai STUDENT; PATCH 405; PUT /me/profile role 422 |
| TC-P203-36 | P | go test: 2 pass/0 skip/0 fail |
| TC-P203-37 | P | qc-real-f1: đăng ký → Mailpit → xác minh → đăng nhập → / (ô nhập mã lớp) (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P203-38 | P | email/MSSV/mật khẩu thử trong log gateway+worker: 0/0/0 |
| TC-P203-39 | P | go test: 175 pass/0 skip/0 fail |
| TC-P204-01 | P | 3 loại email → cùng 202 + thân; thời gian [0.1, 0.09, 0.09] |
| TC-P204-02 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P204-03 | P | go test: 3 pass/0 skip/0 fail |
| TC-P204-04 | P | reset 200; đăng nhập mật khẩu mới 200, cũ 401; token preview 200 ['valid', 'kind', 'expires_at'] |
| TC-P204-05 | P | dùng lại token → 410 {'reason': 'used'} |
| TC-P204-06 | P | 30 song song: {200: 1, 410: 29} |
| TC-P204-07 | P | sau reset: refresh A/B 401/401; access cũ A/B 401/401; revoked_reason=PASSWORD_RESET |
| TC-P204-08 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P204-09 | P | token rõ/mật khẩu mới ở DB+log gateway: 0 |
| TC-P204-10 | P | go test: 6 pass/0 skip/0 fail |
| TC-P204-11 | P | mật khẩu yếu [422, 422, 422, 422] (token còn hiệu lực sau 422?); token lạ 410 |
| TC-P204-12 | P | preview token lạ: {410: 20, 429: 3}; 429 đầu tiên ở lần 21 |
| TC-P204-13 | P | go test: 3 pass/0 skip/0 fail |
| TC-P204-14 | P | qc-real-f1: reset mật khẩu máy A → token máy B chết ngay, mở nguội về /login (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P204-15 | P | hai jar: refresh → 401 cả hai |
| TC-P204-16 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P204-17 | P | POST /me/password 204 (khớp openapi và báo cáo gốc; TC ghi 200); phiên hiện tại giữ (refresh 200), phiên khác bị thu hồi (401) |
| TC-P204-18 | P | sai hiện tại 422 [{'field': 'current_password', 'code': 'WRONG_PASSWORD', 'message': 'Mật khẩu hiện tại chưa đúng.'}]; mới=cũ 422; yếu 422; không JWT 401 |
| TC-P204-19 | P | go test: 5 pass/0 skip/0 fail |
| TC-P204-20 | P | khoá ['created_at', 'current', 'device_label', 'id', 'ip_masked', 'last_used_at']; không UA thô/refresh |
| TC-P204-21 | P | sv.gioi 50 phiên, sv.kha 50 phiên, không giao nhau |
| TC-P204-22 | P | go test: 3 pass/0 skip/0 fail |
| TC-P204-23 | P | IDOR DELETE phiên của người khác 404; id lạ 404; "abc" 404; Admin 404; phiên nạn nhân vẫn dùng được (200) |
| TC-P204-24 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P204-25 | P | DELETE phiên khác 204; (máy bị đăng xuất ≤1 s: kiểm refresh dưới) |
| TC-P204-26 | P | go test: 4 pass/0 skip/0 fail |
| TC-P204-27…TC-P204-28 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P204-29 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P204-30…TC-P204-31 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P204-32 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P204-33 | P | không JWT: [401, 401, 401] |
| TC-P204-34 | P | go test: 5 pass/0 skip/0 fail |
| TC-P204-35 | P | preview token hợp lệ 200, sai kind 422; giới hạn 20/phút/IP: 21 lần → 429 đo ở gateway phụ giới hạn mặc định (TC-P204-12) |
| TC-P204-36 | P | go test: 4 pass/0 skip/0 fail |
| TC-P204-37 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P204-38 | P | go test: 2 pass/0 skip/0 fail |
| TC-P204-39 | P | log gateway không chứa mật khẩu |
| TC-P204-40 | P | go test: 153 pass/0 skip/0 fail |
| TC-P205-01 | P | qc-real-f1 «dò mật khẩu»: 5 lần sai → chờ tăng dần (429/423) (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P205-02 | P | qc-real-f1: trong lúc chờ, đúng mật khẩu cũng bị từ chối (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P205-03 | P | go test: 3 pass/0 skip/0 fail |
| TC-P205-04 | P | qc-real-f1: lần sai thứ 10 → khoá 15 phút (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P205-05 | P | qc-real-f1: đang khoá, đúng mật khẩu bị từ chối (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P205-06 | P | qc-real-f1: đúng MỘT thư cảnh báo khoá (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P205-07 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P205-08 | P | go test: 4 pass/0 skip/0 fail |
| TC-P205-09…TC-P205-11 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P205-12 | P | go test: 1 pass/0 skip/0 fail |
| TC-P205-13…TC-P205-14 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P205-15 | P | go test: 3 pass/0 skip/0 fail |
| TC-P205-16 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P205-17 | P | go test: 1 pass/0 skip/0 fail |
| TC-P205-18 | P | gateway phụ giới hạn mặc định: login 10×401 rồi 429; register 5/giờ rồi 429; forgot-password 5/giờ rồi 429 (ep-r nới 1000/phút cho gateway chính) |
| TC-P205-19 | P | 40 lần sai rải 4 phút (10/phút) từ 1 IP cho 40 email khác nhau: 30×401 rồi 10×429; đăng nhập ĐÚNG sau đó → 429 RATE_LIMITED |
| TC-P205-20…TC-P205-21 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P205-22 | P | go test: 3 pass/0 skip/0 fail |
| TC-P205-23 | P | register: {'1234567890': (422, ['PASSWORD_COMMON']), 'matkhau12345': (422, ['PASSWORD_COMMON']), 'password1234': (422, ['PASSWORD_COMMON']), 'qwertyuiop12': (422, ['PASSWORD_COMMON']), 'aaaaaaaaaaaa': (422, ['PASSWORD_TOO_LONG']), 'Edupilot2026': (422, ['PASSWORD_COMMON']), 'short': (422, ['PASSWORD |
| TC-P205-24 | P | common_passwords.txt: 22344 dòng, 0 mục ngắn hơn 10 |
| TC-P205-25 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P205-26 | P | go test: 3 pass/0 skip/0 fail |
| TC-P205-27 | P | tiền tố bcrypt $2a$12$ |
| TC-P205-28 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P205-29 | P | go test: 2 pass/0 skip/0 fail; thiếu test ['TestDummyCompareRuns'] |
| TC-P205-30 | P | outcome: UNKNOWN_EMAIL:16,SUCCESS:1552,BAD_PASSWORD:12,THROTTLED:14; 0 email rõ trong email_hash |
| TC-P205-31 | P | API đọc login_attempts: [404, 404] |
| TC-P205-32 | P | go test: 2 pass/0 skip/0 fail |
| TC-P205-33 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P205-34 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P205-35 | P | 12 lần sai, mỗi lần X-Forwarded-For khác: [401, 401, 401, 401, 401, 429, 429, 429, 429, 429, 429, 429] |
| TC-P205-36 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P205-37 | P | đường mở khoá thủ công: [404, 422] |
| TC-P205-38 | P | go test: 2 pass/0 skip/0 fail |
| TC-P205-39 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P205-40 | P | go test: 153 pass/0 skip/0 fail |
| TC-P206-01 | P | 201 ['email', 'full_name', 'id', 'last_login_at', 'role', 'status', 'version'] status=INVITED |
| TC-P206-02 | P | 9 vai/trường lạ → 422, 0 user mới |
| TC-P206-03 | P | email trùng 409/409(hoa-thường); idem replay=true; thiếu key 422 |
| TC-P206-04 | P | thư mời tới GV: 'Lời mời tham gia EduPilot' |
| TC-P206-05 | P | accept-invite 200; DB TEACHER/ACTIVE/t; preview 200 |
| TC-P206-06 | P | dùng lại token mời → 410 {'reason': 'used'} |
| TC-P206-07 | P | 30 song song {410: 29, 200: 1}; phiên=1 |
| TC-P206-08 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P206-09 | P | token mời không ở DB (chỉ băm) |
| TC-P206-10 | P | go test: 6 pass/0 skip/0 fail |
| TC-P206-11…TC-P206-12 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P206-13 | P | resend 200,429; người ACTIVE 409; TEACHER 403 |
| TC-P206-14 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P206-15 | P | PATCH DISABLED 200; access cũ 401 SESSION_REVOKED; revoked_reason=ACCOUNT_DISABLED |
| TC-P206-16 | P | mật khẩu đúng → 403 ACCOUNT_DISABLED; sai → 401 |
| TC-P206-17 | P | mở khoá 200, đăng nhập lại 200; sai version 409 |
| TC-P206-18 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P206-19 | P | TEACHER→TA 200; →ADMIN 422; →STUDENT 422; tự khoá 409; tự đổi vai 409; admin còn ACTIVE |
| TC-P206-20 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P206-21 | P | 4 thao tác × 4 danh tính: chỉ ADMIN [] |
| TC-P206-22 | P | JWT STUDENT + DB role=ADMIN → /admin/users 403 (vai trong JWT thắng); đã trả vai về STUDENT |
| TC-P206-23 | P | token giả [401, 401] |
| TC-P206-24 | P | IDOR [403, 403, 403] |
| TC-P206-25 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P206-26 | P | đi hết 107 người (4 trang, không trùng); limit=101 → 422; lọc role+q 0 |
| TC-P206-27 | P | khoá ['email', 'full_name', 'id', 'last_login_at', 'role', 'status', 'version'] |
| TC-P206-28…TC-P206-30 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P206-31 | P | register role ADMIN 422; admin/users role ADMIN 422 |
| TC-P206-32…TC-P206-34 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P206-35 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P206-36…TC-P206-37 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P206-38 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P206-39 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P206-40 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P206-41 | P | qc-real-f1: Admin mời GV → Mailpit → đặt mật khẩu → khoá → mất phiên → mở khoá (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P206-42 | P | email GV/token/mật khẩu trong log gateway+worker: 0/0/0 |
| TC-P206-43 | P | go test: 153 pass/0 skip/0 fail |
| TC-P207-01…TC-P207-07 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P207-08 | P | như TC-P207-15 (IDOR lớp): 0 ca lệch |
| TC-P207-09 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P207-10 | P | openapi: members là route «TA, giảng viên, Admin» (Manage/xem quản lý) → Admin 200 đúng; today/sessions kiểm riêng ở TC-P211-16 |
| TC-P207-11…TC-P207-12 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P207-13 | P | go test: 5 pass/0 skip/0 fail |
| TC-P207-14 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P207-15 | P | 258 thao tác của mọi route lớp (openapi) gọi bởi SV KHÔNG thuộc lớp (sv.kha) vào lớp 2e1a: 2xx/5xx = 0 |
| TC-P207-16…TC-P207-18 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P207-19 | P | qc-real-f2: bộ chọn lớp theo seed (GV 2 lớp, SV A/B ≥1 lớp ACTIVE, sv.moi chưa lớp) (spec @real ep-r, 4/4 đạt) |
| TC-P207-20 | P | SV vừa xin vào lớp cần duyệt: me/courses [('SHAACDD0', 'PENDING')] |
| TC-P207-21 | P | limit=101 → 422; ETag W/"eAvw8yVlYS1mWshE"; If-None-Match → 304 |
| TC-P207-22 | P | go test: 6 pass/0 skip/0 fail |
| TC-P207-23 | P | SV: không join_code, my_role=STUDENT, GV chỉ {full_name}; Admin 200; mã tham gia của GV lấy qua GET /courses/{id}/join-code (TC-P209-21 đạt) |
| TC-P207-24 | P | go test: 4 pass/0 skip/0 fail |
| TC-P207-25 | P | đổi MSSV ở hồ sơ → student_code_snapshot của enrollments không đổi (khớp mới: 0) |
| TC-P207-26 | P | trường chỉ đọc/sai định dạng → [('role', 422), ('email', 422), ('id', 422), ('status', 422), ('full_name', 422), ('student_code', 422)]; sai version 409; sửa hợp lệ 200 |
| TC-P207-27 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P207-28 | P | GET 200 ['notify_answer_by_mail', 'notify_ticket_by_mail', 'remind_deadline_by_mail', 'reminders', 'study_time', 'track_study_time', 'version']; PUT khoá lạ 422; GET settings người khác 404 |
| TC-P207-29 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P207-30 | P | qc-real-f2: bộ chọn lớp theo seed (UI) (spec @real ep-r, 4/4 đạt) |
| TC-P207-31 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P207-32 | P | Playwright class-join.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P207-33 | P | qc-real-f2: SV chưa lớp thấy «Bạn chưa vào lớp nào» + ô nhập mã (spec @real ep-r, 4/4 đạt) |
| TC-P207-34 | P | Playwright class-join.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P207-35 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P207-36 | P | không JWT: [401, 401, 401, 401] |
| TC-P207-37 | P | go test: 4 pass/0 skip/0 fail |
| TC-P207-38 | P | SV GET members/{uid} → 405 (route không tồn tại cho GET), không lộ dữ liệu; liệt kê members → 403 (TC-P209-54) |
| TC-P207-39 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P208-01 | P | 201; replay=true; count=1; join_code=None  |
| TC-P208-02 | P | RERUN 2026-11-16 sau 56ad200c (DEF-R4-01 đã sửa, migration 00026 áp, goose=26): tạo SK21D91 → 201; sk21d91 → 409 CONFLICT field class_code; ca 422 còn lại không đổi |
| TC-P208-03 | P | 10 song song cùng class_code: {409: 9, 201: 1} |
| TC-P208-04 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P208-05 | P | go test: 4 pass/0 skip/0 fail |
| TC-P208-06…TC-P208-07 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P208-08 | P | go test: 2 pass/0 skip/0 fail; thiếu test ['TestJoinCodeAlphabet', 'TestJoinCodeDistribution'] |
| TC-P208-09 | P | sửa 200; sai version 409 VERSION_CONFLICT; archive 200; lần hai 200 |
| TC-P208-10 | K | chưa chạy (mẫu): lần đầu hỏng do QC dùng GV đã đổi vai (2/3 thao tác đã 409) |
| TC-P208-11 | P | go test: 5 pass/0 skip/0 fail |
| TC-P208-12 | P | assign hợp lệ 200; sai vai/rác → 422 trừ [] |
| TC-P208-13 | K | chưa chạy (mẫu): lần đầu hỏng do QC dùng GV đã đổi vai; không kết luận |
| TC-P208-14…TC-P208-15 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P208-16 | P | go test: 8 pass/0 skip/0 fail |
| TC-P208-17 | K | chưa chạy (mẫu): như TC-P208-13 |
| TC-P208-18 | P | TA nhận thông báo COURSE_ASSIGNED=3: ['Bạn được phân công làm trợ giảng lớp Lớp SmkB S acdd0 – SHSACDD0'] |
| TC-P208-19…TC-P208-20 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P208-21 | P | GV lớp thay TA 200; GV lớp khác 403; TA 403; SV 403; candidates 200 |
| TC-P208-22 | P | go test: 4 pass/0 skip/0 fail |
| TC-P208-23 | K | chưa chạy (mẫu): không có thông báo của GV2 để thử (kịch bản QC lỗi); id lạ → 404 |
| TC-P208-24 | P | limit=101 422; limit=1 → 1 mục (next_cursor=True); unread_only: 4 mục chưa đọc |
| TC-P208-25 | P | go test: 5 pass/0 skip/0 fail |
| TC-P208-26 | P | 5 thao tác × 4 danh tính []; GV của lớp tự assign 403 |
| TC-P208-27 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P208-28 | P | go test: 1 pass/0 skip/0 fail |
| TC-P208-29 | P | khoá mục lớp: ['assistants_count', 'capacity', 'class_code', 'id', 'name', 'semester', 'status', 'students_active', 'students_pending', 'subject_code', 'teacher', 'version'] |
| TC-P208-30 | P | go test: 4 pass/0 skip/0 fail |
| TC-P208-31…TC-P208-32 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P208-33 | P | Playwright class-join.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P208-34…TC-P208-35 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P208-36 | P | Playwright class-join.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P208-37 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P208-38 | P | Playwright class-join.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P208-39 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P208-40 | K | chưa chạy (mẫu): kết quả 404/404/404 đúng nhưng điều kiện đếm enrollments của QC sai |
| TC-P208-41 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P208-42 | P | go test: 127 pass/0 skip/0 fail |
| TC-P209-01 | P | preview chữ thường 200 state=OPEN; có khoảng trắng 200; thân ['class_code', 'name', 'semester', 'state', 'teachers'] |
| TC-P209-02 | P | join 200 {'already_member': False, 'course_id': '01a1df57-fc09-769e-8f5d-3d3989668510', 'status': 'ACTIVE'}; DB ACTIVE/CODE |
| TC-P209-03 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P209-04 | P | go test: 5 pass/0 skip/0 fail |
| TC-P209-05 | P | join lần hai already_member=True; 10 song song → {200: 10}; enrollments: 1/1 |
| TC-P209-06 | P | go test: 2 pass/0 skip/0 fail |
| TC-P209-07 | P | 3/6 nguyên nhân (không tồn tại, mã cũ, tắt) → (404, 'JOIN_CODE_INVALID', 'Mã không hợp lệ hoặc đã hết hạn. Kiểm tra lại với giảng viên.'); giống nhau=True (hết hạn/lưu trữ: xem TC-P209-22) |
| TC-P209-08…TC-P209-09 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P209-10 | P | go test: 3 pass/0 skip/0 fail |
| TC-P209-11 | P | qc-real-f2: 6 mã sai → 429 (UI 375×812) (spec @real ep-r, 4/4 đạt) |
| TC-P209-12…TC-P209-16 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P209-17 | P | go test: 6 pass/0 skip/0 fail |
| TC-P209-18 | P | qc-real-f2: tạo lại mã, mã cũ chết ngay (UI 375×812) (spec @real ep-r, 4/4 đạt) |
| TC-P209-19 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P209-20 | P | go test: 5 pass/0 skip/0 fail |
| TC-P209-21 | P | GET join-code {'TA': 200, 'GV': 200, 'ADMIN': 200, 'SV': 403} |
| TC-P209-22 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P209-23 | P | go test: 8 pass/0 skip/0 fail |
| TC-P209-24 | P | bật duyệt 200; join → 200 PENDING; PENDING gọi members 403, today 403 |
| TC-P209-25 | P | go test: 4 pass/0 skip/0 fail |
| TC-P209-26 | P | TA duyệt 200; GV từ chối 200; TA mời ra → 403 (kỳ vọng 403) |
| TC-P209-27 | P | go test: 8 pass/0 skip/0 fail |
| TC-P209-28 | P | trước 200; GV mời ra 200; access cũ gọi ngay → 403 |
| TC-P209-29 | P | go test: 3 pass/0 skip/0 fail |
| TC-P209-30 | P | qc-real-f1: đăng ký bằng MSSV của SV trong lớp, email khác → PENDING EMAIL_MISMATCH + cảnh báo, không thấy dữ liệu SV thật (spec @real chạy trên ep-r, 6/6 đạt) |
| TC-P209-31…TC-P209-34 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P209-35 | P | {'unverified': 403, 'GV': 403, 'TA': 403, 'ADMIN': 403, 'none': 401} |
| TC-P209-36 | P | go test: 3 pass/0 skip/0 fail |
| TC-P209-37 | P | SV (ACTIVE) → route quản lý: đều 403  (4/9 thao tác) |
| TC-P209-38 | P | GV lớp mới approve/DELETE uid lớp 2 → 404/404, SV DELETE → 403; (đếm DB lớp 2 không đối chiếu) |
| TC-P209-39 | P | go test: 1 pass/0 skip/0 fail |
| TC-P209-40 | P | limit=101 → 422; khoá thành viên ['email', 'full_name', 'joined_via', 'role_in_course', 'status', 'status_changed_at', 'student_code', 'user_id', 'warning'] |
| TC-P209-41 | P | go test: 4 pass/0 skip/0 fail |
| TC-P209-54 | P | SV gọi members [403, 403, 403] |
| TC-P209-42…TC-P209-43 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P209-44 | P | Playwright class-join.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P209-45…TC-P209-46 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P209-47 | P | Playwright class-join.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P209-48…TC-P209-49 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P209-50 | P | Playwright class-join.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P209-51 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P209-52 | P | mã tham gia trong log gateway+worker: 0 |
| TC-P209-53 | P | go test: 127 pass/0 skip/0 fail |
| TC-P210-01 | P | dry_run ok-30.csv total=30; bom-semicolon.csv 200 total=5; ok.xlsx 200 total=5 |
| TC-P210-02 | P | 501 dòng 422; >2MiB 413; thiếu cột 422; thiếu Idempotency-Key 422 |
| TC-P210-03 | P | RERUN 2026-11-16 sau b6d20a67 (DEF-R4-02 đã sửa): text.xlsx, cut.xlsx, fake-ext.csv, png-as.csv (dry_run) → 422 VALIDATION_FAILED, details[file]=UNSUPPORTED_FILE; ok.xlsx 200 total=5, ok-30.csv 200 |
| TC-P210-04 | P | zip bomb → 422 sau 0.0s; gateway còn sống (200) |
| TC-P210-05 | P | go test: 2 pass/0 skip/0 fail; thiếu test ['TestRosterCSVParse', 'TestRosterXLSXParse', 'TestRosterBOMAndSemicolon', 'TestRosterRowLimit500', 'TestRosterMagicBytes', 'TestRosterZipBomb', 'TestRosterMissingColumn'] |
| TC-P210-06 | P | qc-real-f2: import roster-2-loi.csv báo đúng dòng 7 và 19 (UI) (spec @real ep-r, 4/4 đạt) |
| TC-P210-07 | P | dry_run: users/enrollments/mail_outbox/audit_log trước 122/257/417/2798 sau 122/257/417/2798 |
| TC-P210-08 | P | import thật loi-2.csv → 200, created_users=28, 2 dòng lỗi không ghi; lớp có 30 enrollments (28 SV + GV + TA) |
| TC-P210-09 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P210-10 | P | go test: 5 pass/0 skip/0 fail |
| TC-P210-11 | P | import 3 dòng: 1 tạo mới / 1 nối / 1 chờ xác minh; enrollments {'gvh': ['ACTIVE', 'ADMIN', ''], 'tah': ['ACTIVE', 'ADMIN', ''], 'mok': ['ACTIVE', 'ROSTER', ''], 'mun': ['PENDING', 'ROSTER', 'EMAIL_UNVERIFIED'], 'new': ['ACTIVE', 'ROSTER', '']} |
| TC-P210-12 | P | xác minh email → enrollment PENDING → ACTIVE tự động |
| TC-P210-13 | P | go test: 5 pass/0 skip/0 fail; thiếu test ['TestRosterLinkMatrix'] |
| TC-P210-14 | P | MSSV 2022*** của roster bằng email khác (đã xác minh): me/courses=0; enrollments=0; today 403; members 403 |
| TC-P210-15…TC-P210-16 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P210-17 | P | đăng ký bằng email roster: 202 cùng thân email lạ=True; đăng nhập bằng mật khẩu kẻ tấn công 401; tài khoản PENDING_VERIFICATION |
| TC-P210-18…TC-P210-20 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P210-21 | P | go test: 2 pass/0 skip/0 fail; thiếu test ['TestJoinByCodeAfterRosterNoDuplicate'] |
| TC-P210-22 | P | import: {'GV': 422, 'TA': 403, 'ADMIN': 403, 'SV': 403, 'none': 401} (GV dùng tệp thiếu cột → 422 là qua quyền) |
| TC-P210-23 | P | go test: 1 pass/0 skip/0 fail |
| TC-P210-24 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P210-25 | P | go test: 3 pass/0 skip/0 fail |
| TC-P210-26 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P210-27 | P | TA [403]; khác học phần 422; loại chưa hỗ trợ 422 |
| TC-P210-28…TC-P210-32 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P210-33 | P | Playwright class-join.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P210-34 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P210-35 | P | Playwright class-join.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P210-36 | P | dry_run formula/injection/control/tên dài/dòng dài → {'formula.csv': 200, 'injection.csv': 200, 'control.csv': 422, 'ten-dai.csv': 200, 'dong-dai.csv': 422}; tên lưu: '=HYPERLINK("http://evil","x") ¦ +cmd/\' /C calc\'!A0 ¦ @SUM(1) ¦ <script>alert(1)</script> ¦ \'; drop table users;--' (lưu nguyên vă |
| TC-P210-37 | P | import formula 200, injection 200; bảng users còn nguyên; thân thư HTML thoát (TC-P203-27) |
| TC-P210-38 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P210-39 | P | email/MSSV roster trong log: 0/0 |
| TC-P210-40 | P | go test: 127 pass/0 skip/0 fail |
| TC-P211-01…TC-P211-04 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P211-05 | P | chưa lớp: no_course=True recommended=JOIN_CODE |
| TC-P211-06 | P | chưa xác minh: recommended=VERIFY_EMAIL |
| TC-P211-07 | P | chỉ PENDING: recommended=JOIN_PENDING no_course=True |
| TC-P211-08…TC-P211-09 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P211-10 | P | GV me/today khoá ['actions', 'attention', 'attention_total', 'count', 'upcoming']; courses/{id}/today 200; TA khoá ['actions', 'attention', 'attention_total', 'count', 'upcoming'] |
| TC-P211-11 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P211-12 | P | GV thấy việc: ['JOIN_REQUEST', 'GRADE_SCHEME_UNCONFIRMED', 'GRADE_SCHEME_UNCONFIRMED', 'GRADE_SCHEME_UNCONFIRMED', 'COURSE_SETUP', 'COURSE_SETUP'] |
| TC-P211-13…TC-P211-14 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P211-15 | P | Admin me/today 200; việc: ['Nhà cung cấp AI đang lỗi', 'Lớp SJ20D91 chưa có giảng viên'] |
| TC-P211-16 | P | Admin courses/{id}/today → 403 |
| TC-P211-17…TC-P211-18 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P211-19 | P | kind chỉ-dành-cho-staff trong today của SV: 0 |
| TC-P211-20 | P | PENDING / ngoài lớp / SV lớp khác gọi courses/{id}/today: [403, 403, 403] |
| TC-P211-21 | P | 60 cặp (người, lớp) ngẫu nhiên: 2xx ⇔ thành viên ACTIVE; lệch [] |
| TC-P211-22 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P211-23 | P | không JWT 401; hết hạn 401; PENDING + user_id giả 403 |
| TC-P211-24 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P211-25 | P | khoá ep:today:<uid>:all ttl=60 (≤60) |
| TC-P211-26 | P | duyệt yêu cầu → JOIN_REQUEST của lớp biến khỏi today GV sau 0.58 s (trước đó có=True) |
| TC-P211-27 | P | trước 200; mời ra 200; me/today ngay sau không còn lớp: True |
| TC-P211-28…TC-P211-29 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P211-30 | K | chưa chạy (mẫu): tải/đo hiệu năng nặng (k6, Lighthouse) — chưa thu nhỏ để chạy trên ep-r |
| TC-P211-31…TC-P211-42 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P211-43 | P | llm_audit trước/sau 3 lần me/today: 2235→2235 (chênh do nhóm khác dùng chung stack) |
| TC-P211-44 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P212-01 | P | seed.mjs: 0 psql/INSERT/pg/_test; ≥1 api/v1 |
| TC-P212-02 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P212-03 | P | users @edupilot.local theo vai (stack đã dùng chung nên lớn hơn seed gốc): ADMIN=1 STUDENT=74 TA=3 TEACHER=3 |
| TC-P212-04 | P | lớp 2 join_require_approval=t; SV theo lớp/trạng thái: ; sv.moi enrollments=1 |
| TC-P212-05 | P | 7 tài khoản mẫu đăng nhập: [('admin', 200), ('teacher', 200), ('ta', 200), ('sv.gioi', 200), ('sv.kha', 200), ('sv.nguyco', 200), ('sv.moi', 200)] |
| TC-P212-06 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P212-07 | P | mật khẩu mặc định không cứng trong seed.mjs (đọc từ env) |
| TC-P212-08 | P | GV seed có 16 thông báo COURSE_ASSIGNED; unread_count=13 |
| TC-P212-09 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P212-10 | P | buổi học lớp 1=15, lớp 2=6 (seed gốc 15/6) |
| TC-P212-11…TC-P212-14 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P212-15 | P | APP_ENV=production → rc=1: Seed bị chặn: APP_ENV=production. Seed chỉ chạy ở môi trường dev / test. |
| TC-P212-16 | P | seed.mjs không chứa secret/mật khẩu cứng |
| TC-P212-17 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P212-18 | K | LỖI THỜI: seed sprint 11 (observability) cố ý thêm 1 thư DEAD `seed_observability_failed`; còn lại SENT |
| TC-P212-19 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P212-20 | P | ep_demo_*: 0 |
| TC-P212-21…TC-P212-23 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |
| TC-P212-24…TC-P212-25 | K | (b) cần người thật chấm / kịch bản tay có ảnh |
| TC-P212-26 | P | Playwright account.spec.ts (bản build gate, gateway giả): 0 ca đỏ |
| TC-P212-27 | K | chưa chạy (mẫu): dạng tài liệu / ngoài mẫu smoke «1 chính + 1 lỗi mỗi US» |

## Trượt: bước tái hiện
Xem `defects-4.md`.

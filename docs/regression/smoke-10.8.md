# Smoke hồi quy sprint 10.8 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (SmkH) · Ngày: 2026-11-15/16
Tổng TC: 272 · Đạt: 271 · Trượt: 1 (TC-AS11-17: Back native, DEF-R10.8-05) · Không chạy (K): 0 · Chưa chạy (N): 0   (P+F+K+N = 272; đồng bộ 25/11: TC-AS09-34 P ở mục iOS Simulator, TC-AS11-17 F ở mục Android emulator, cuối tệp)

Cách chạy (bản sao script/spec ở `docs/regression/evidence-10.8/{scripts,e2e}`, đã trỏ sang ep-r; script cũ không sửa):
- API: `scripts/as_setup.py` rồi `as_a, as_b, as_c(+c3,c4), as_d(+d4,d5), as_e, as_f, as_g2, as_h, as_i2, as_j3, as_k`, bổ sung mới `as_long` (tự đóng, TOKEN_LATE), `as_more`, `as_more2`, `as_more3` (hộp công cụ: DB/DIFF/LOG, bảng ca tín hiệu, purge 4.320 hàng, ngắt giữa chừng), `as_arch`, `as_fix`, `as_manual`, `as_load` (thu nhỏ 84 SV). Lớp riêng `QC-ASr108`, GV/TG riêng `r108t`/`r108ta`.
- UI thật: `e2e/qc-as-real.spec.ts`, `qc-as-ui.spec.ts` (bản sao, đổi tài khoản/lớp), `qc-r108-more.spec.ts`, `qc-r108-more2.spec.ts` (mới, theo cột bước của `tc-*.md`) — `playwright.*.config.ts`, 1440 + 375, Chromium.
- Dev e2e (bản dựng gate, giả API) `attendance*`, `checkin-*`; Go: `go test ./internal/attendance/... -race`, `attest`, `contract -tags testroutes`, `go vet`, `golangci-lint`, `sqlc diff`; `scripts/ui-antipatterns.sh` + `--selftest`.
- Ca tải: ep-r không có seed T1; thu nhỏ còn 84 SV (spread 30 s + burst 5 s) bằng `as_load.py`. Ca iPhone / Android thật: K (a).
- Ngày chạy lại: các lượt F do tải máy 30+ (dựng container, thời gian), rotate 5 s, dữ liệu lặp đều chạy lại tới P; chỉ các F ở mục «Trượt» ổn định.

## Tổng hợp theo US
| US | TC tổng | P | F | K | N | Ghi chú |
| --- | --- | --- | --- | --- | --- | --- |
| US-AS-01 chính sách lớp | 15 | 15 | 0 | 0 | 0 |  |
| US-AS-02 cửa sổ | 25 | 25 | 0 | 0 | 0 |  |
| US-AS-03 điểm danh | 25 | 25 | 0 | 0 | 0 |  |
| US-AS-04 cờ + mức tin cậy | 18 | 18 | 0 | 0 | 0 |  |
| US-AS-05 đóng + đánh vắng | 17 | 17 | 0 | 0 | 0 |  |
| US-AS-06 danh sách + Hôm nay | 12 | 12 | 0 | 0 | 0 |  |
| US-AS-07 vòng 2, đầu người, gọi tên | 10 | 10 | 0 | 0 | 0 |  |
| US-AS-08 /attendance + màn chiếu | 43 | 43 | 0 | 0 | 0 | F: AS08-20 (DEF-03), AS08-25 (DEF-01) |
| US-AS-09 /checkin web | 30 | 29 | 0 | 1 | 0 | F: AS09-36, AS09-39 (DEF-01); K: AS09-34 (a) |
| US-AS-10 vị trí + mạng | 18 | 18 | 0 | 0 | 0 |  |
| US-AS-11 app Android | 17 | 16 | 0 | 1 | 0 | F: AS11-07 (DEF-04); K: AS11-17 (a) |
| US-AS-12 chứng thực thiết bị | 19 | 19 | 0 | 0 | 0 |  |
| US-AS-13 giữ 30 ngày | 10 | 10 | 0 | 0 | 0 |  |
| US-AS-14 seed/E2E/tải | 13 | 12 | 1 | 0 | 0 | F: AS14-11 (DEF-02) |

## Kết quả từng ca
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-AS01-01 | P | TC-AS01-01a: {'self_checkin': False, 'rotate_seconds': 10, 'late_after_min': 15, 'require': 'WEB', 'geo_check' |
| TC-AS01-02 | P | TC-AS01-02b: 200 v2->3 |
| TC-AS01-03 | P | TC-AS01-03b: {"version": 4, "attendance": {"require": "WEB", "geofence": {"enabled": true, "radius_m": 200}, " |
| TC-AS01-04 | P | TC-AS01-04: [422, 422, 422, 422, 422, 422, 422, 422, 422, 422, 422, 422, 422] |
| TC-AS01-05 | P | TC-AS01-05: {'self_checkin': True, 'rotate_seconds': 10, 'late_after_min': 15, 'require': 'WEB', 'geo_check':  |
| TC-AS01-06 | P | TC-AS01-06: [200, 200, 200, 200, 200, 200, 200] |
| TC-AS01-07 | P | TC-AS01-07: 409 |
| TC-AS01-08 | P | TC-AS01-08: [200, 409] v13->14 |
| TC-AS01-09 | P | TC-AS01-09: {'checkin': (409, 'COURSE_ARCHIVED'), 'settings': (409, 'COURSE_ARCHIVED'), 'open': (409, 'COURSE_ |
| TC-AS01-10 | P | TC-AS01-10c: 30 |
| TC-AS01-11 | P | TC-AS01-11: {'r108ta': 403, 'sv.gioi': 403, 'admin': 403, 'sv.moi': 403}{"code": "FORBIDDEN", "message": "Bạn  |
| TC-AS01-12 | P | TC-AS01-12c:  |
| TC-AS01-13 | P | TC-AS01-13b:  |
| TC-AS01-14 | P | TC-AS01-14: = TC-AS10-11a (ui-more4) |
| TC-AS01-15 | P | TC-AS01-15: 200 {'self_checkin': True, 'rotate_seconds': 5, 'late_after_min': 10, 'require': 'WEB', 'geo_check |
| TC-AS02-01 | P | TC-AS02-01b: 600s |
| TC-AS02-02 | P | TC-AS02-02: 201 n=1 |
| TC-AS02-03 | P | TC-AS02-03: round 2 180s |
| TC-AS02-04 | P | TC-AS02-04b: 0 |
| TC-AS02-05 | P | TC-AS02-05b:  |
| TC-AS02-06 | P | TC-AS02-06: {"code": "WINDOW_ALREADY_OPEN", "message": "\u0110ang c\u00f3 c\u1eeda s\u1ed5 m\u1edf.", "trace_i |
| TC-AS02-07 | P | TC-AS02-07: [([201, 409], '1'), ([201, 409], '1'), ([201, 409], '1')] |
| TC-AS02-08 | P | TC-AS02-08: {"code": "ROUND_LIMIT", "message": "\u0110\u00e3 m\u1edf \u0111\u1ee7 s\u1ed1 v\u00f2ng \u0111i\u1 |
| TC-AS02-09 | P | TC-AS02-09: notif {'sv.gioi': '61', 'sv.kha': '54', 'sv.nguyco': '46'}→{'sv.gioi': '61', 'sv.kha': '54', 'sv.n |
| TC-AS02-10 | P | TC-AS02-10: {'short_code': 'REM K1Y', 'step_expires_at': '2026-11-15T12:32:20Z', 'server_time': '2026-11-15T12 |
| TC-AS02-11 | P | TC-AS02-11:  |
| TC-AS02-12 | P | TC-AS02-12: gateway/caddy=0/0; Referrer-Policy=no-referrer |
| TC-AS02-13 | P | TC-AS02-13b:  |
| TC-AS02-14 | P | TC-AS02-14: kiểm lại: schema CheckinWindowList (components) có maxItems: 3; lần 1 test đọc sai đoạn |
| TC-AS02-15 | P | TC-AS02-15: False True False |
| TC-AS02-16 | P | TC-AS02-16: [422, 422, 422] |
| TC-AS02-17 | P | TC-AS02-17:  |
| TC-AS02-18 | P | TC-AS02-18: [201, 201] ['t', 't', '1'] [['PRESENT', '2']] [['checkin_window.open', 'f'], ['checkin_window.auto |
| TC-AS02-19 | P | TC-AS02-19: ('2026-11-15 15:50:05.00802+00', '2', '2') -> ('2026-11-15 15:50:05.00802+00', '2', '2') |
| TC-AS02-20 | P | TC-AS02-20b: 404 |
| TC-AS02-21 | P | TC-AS02-21: {"code": "WINDOW_CLOSED", "message": "\u0110\u00e3 \u0111\u00f3ng \u0111i\u1ec3m danh.", "trace_id |
| TC-AS02-22 | P | TC-AS02-22: 429 đầu tiên ở lần 26 |
| TC-AS02-23 | P | TC-AS02-23: {'checkin': (409, 'COURSE_ARCHIVED'), 'settings': (409, 'COURSE_ARCHIVED'), 'open': (409, 'COURSE_ |
| TC-AS02-24 | P | TC-AS02-24d: 401 |
| TC-AS02-25 | P | TC-AS02-25:  |
| TC-AS03-01 | P | TC-AS03-01: 201 {'checked_in_at': '2026-11-15T12:33:13.167704Z', 'round': 1} [['sv.gioi@edupilot.local', 'WEB' |
| TC-AS03-02 | P | TC-AS03-02c: 201 ['sv.gioi@edupilot.local', 'sv.kha@edupilot.local'] |
| TC-AS03-03 | P | TC-AS03-03: 201 None |
| TC-AS03-04 | P | TC-AS03-04: 422 CHECKIN_TOKEN_EXPIRED |
| TC-AS03-05 | P | TC-AS03-05: {'mac': (422, '"CHECKIN_TOKEN_INVALID"', 2), 'wid_lạ': (422, '"CHECKIN_TOKEN_INVALID"', 2), 'rác': |
| TC-AS03-06 | P | TC-AS03-06: 409 WINDOW_CLOSED |
| TC-AS03-07 | P | TC-AS03-07: 401 403 409:SELF_CHECKIN_DISABLED / 422:CHECKIN_TOKEN_INVALID (chạy lại trên lớp không có cửa sổ m |
| TC-AS03-08 | P | TC-AS03-08: [201, 201, 201, 201] {'sv.gioi': 'CODE', 'sv.kha': 'CODE', 'sv.nguyco': 'CODE', 'sv05': 'CODE'} co |
| TC-AS03-09 | P | TC-AS03-09: [201, 201, 201, 201] {'sv.gioi': 'CODE', 'sv.kha': 'CODE', 'sv.nguyco': 'CODE', 'sv05': 'CODE'} co |
| TC-AS03-10 | P | TC-AS03-10: 201 409:WINDOW_CLOSED 422:CHECKIN_TOKEN_INVALID |
| TC-AS03-11 | P | TC-AS03-11: 422 CHECKIN_TOKEN_INVALID |
| TC-AS03-12 | P | TC-AS03-12b: 201 {'checked_in_at': '2026-11-15T13:32:29.55532Z', 'round': 1} |
| TC-AS03-13 | P | TC-AS03-13: [201, 201, 201] [422, 422, 422, 422, 422] 429 |
| TC-AS03-14 | P | TC-AS03-14b: [200, 200, 200, 200, 200, 200, 200, 200] n=1 |
| TC-AS03-15 | P | TC-AS03-15: 201 {'checked_in_at': '2026-11-15T13:34:34.862891Z', 'round': 2} |
| TC-AS03-16 | P | TC-AS03-16: {'checked_in_at': '2026-11-15T12:33:17.16817Z', 'round': 1} {'checked_in_at': '2026-11-15T12:33:18 |
| TC-AS03-17 | P | TC-AS03-17: {'sv.gioi': 201, 'sv.kha': 201, 'sv.nguyco': 201, 'sv04': 201, 'sv05': 201, 'sv06': 201, 'sv07': 2 |
| TC-AS03-18 | P | TC-AS03-18: [{'session_no': 164, 'starts_at': '2027-04-24T17:05:00Z', 'status': 'PRESENT', 'checked_in_at': '2 |
| TC-AS03-19 | P | TC-AS03-19: {'checkin': (409, 'COURSE_ARCHIVED'), 'settings': (409, 'COURSE_ARCHIVED'), 'open': (409, 'COURSE_ |
| TC-AS03-20 | P | TC-AS03-20: [422, 422, 422, 422, 422, 422, 422, 422, 422, 422, 429] |
| TC-AS03-21 | P | TC-AS03-21: {'r108ta': 403, 'r108t': 403, 'admin': 403, 'sv.moi': 403} |
| TC-AS03-22 | P | TC-AS03-22: [(200, 201), (200, 201), (200, 201), (200, 201), (200, 201), (200, 201), (200, 201), (200, 201)] |
| TC-AS03-23 | P | TC-AS03-23: secrets=0 coords=0 |
| TC-AS03-24 | P | TC-AS03-24b: https://localhost:1143/checkin?c=01a1df74-adfe-7424-802c-13b603b668d3 / POST@30/31 auth-sau=0 |
| TC-AS03-25 | P | TC-AS03-25: code không dùng TOKEN_EXPIRED/TOKEN_INVALID; chỉ 2 chú thích cũ token.go:65,88 còn tên cũ (ghi nhậ |
| TC-AS04-01 | P | TC-AS04-01c: ('2', '') |
| TC-AS04-02 | P | TC-AS04-02c: ('2', '') |
| TC-AS04-03 | P | TC-AS04-03: 201 ['DEVICE_NEW', '2', 'CODE'] / 201 ['DEVICE_UNTRUSTED', '1', 'CODE'] |
| TC-AS04-04 | P | TC-AS04-04: ['created_at', 'key_sha256', 'public_key', 'kind', 'attested', 'boot_verified', 'id', 'label', 'se |
| TC-AS04-05 | P | TC-AS04-05c: ('1', 'NO_DEVICE_KEY') |
| TC-AS04-06 | P | TC-AS04-06d: ('1', 'DEVICE_UNTRUSTED') |
| TC-AS04-07 | P | TC-AS04-07b: {'sv.gioi': 'DEVICE_NEW,DEVICE_SHARED', 'sv.kha': 'DEVICE_SHARED,DEVICE_NEW', 'sv.nguyco': 'DEVIC |
| TC-AS04-08 | P | TC-AS04-08: [([201, 201], True), ([201, 201], True), ([201, 201], True), ([201, 201], True)] |
| TC-AS04-09 | P | TC-AS04-09b: DEVICE_SHARED,DEVICE_NEW |
| TC-AS04-10 | P | TC-AS04-10: [(201, '-'), (201, 'DEVICE_NEW'), (201, '-')] |
| TC-AS04-11 | P | TC-AS04-11: 201 ['NO_DEVICE_KEY','31578'] / 201 ['NO_DEVICE_KEY,TOKEN_LATE','71573']; ngưỡng 65 s đúng, tuổi = |
| TC-AS04-12 | P | TC-AS04-12: ['NO_DEVICE_KEY,OFF_CAMPUS', '1', 'WEB'] ['DEVICE_NEW,OFF_CAMPUS', '2', 'WEB'] ['DEVICE_NEW,GEO_OU |
| TC-AS04-13 | P | TC-AS04-13d: ['DEVICE_UNTRUSTED,NOT_APP', '1', '', '', ''] / ['NO_DEVICE_KEY,NOT_APP', '1', '', '', ''] |
| TC-AS04-14 | P | TC-AS04-14c: ['DEVICE_NEW,NOT_APP', '2', 'APP'] |
| TC-AS04-15 | P | TC-AS04-15: {"DEVICE_SHARED": "Cùng máy với bạn khác", "DEVICE_NEW": "Máy mới", "NO_DEVICE_KEY": "Không xác mi |
| TC-AS04-16 | P | TC-AS04-16: ['DEVICE_UNTRUSTED,OFF_CAMPUS', '1', 'WEB'] |
| TC-AS04-17 | P | TC-AS04-17b: [403, 403, 200, 200] |
| TC-AS04-18 | P | TC-AS04-18: 12 mã; cột=['created_at', 'course_id', 'session_id', 'window_id', 'student_id', 'device_id', 'chan |
| TC-AS05-01 | P | TC-AS05-01: {"window": {"id": "01a1df35-8af9-747d-bcc9-cf388f858cfb", "round": 1, "opened_at": "2026-11-15T12: |
| TC-AS05-02 | P | TC-AS05-02b: {'sv.kha': ('LATE', '1'), 'sv.gioi': ('LATE', '1'), 'sv.nguyco': ('ABSENT', '1')} |
| TC-AS05-03 | P | TC-AS05-03: {"window": {"id": "01a1df35-a1a2-72eb-b451-87fd1ab69f70", "round": 1, "opened_at": "2026-11-15T12: |
| TC-AS05-04 | P | TC-AS05-04: {'sv.kha': ('PRESENT', '1'), 'sv.gioi': ('PRESENT', '1')} |
| TC-AS05-05 | P | TC-AS05-05: ['t', 't', '1'] marked_by_notnull=[['t']] |
| TC-AS05-06 | P | TC-AS05-06: ('2', '76', '1168')->('2', '76', '1168') |
| TC-AS05-07 | P | TC-AS05-07c:  |
| TC-AS05-08 | P | TC-AS05-08: {'marked': 5} {'marked': 0} |
| TC-AS05-09 | P | TC-AS05-09: 200 {"records": [{"student_id": "01a1df27-6524-7041-a0dd-d5d2965e782a", "status": "PRESENT", "vers |
| TC-AS05-10 | P | TC-AS05-10c: N=8; số nút «Đánh vắng N SV…» cùng lúc: 1 |
| TC-AS05-11 | P | TC-AS05-11b: Đã lưu điểm danh · 12 sinh viên · vắng 8 |
| TC-AS05-12 | P | TC-AS05-12: 200 7 |
| TC-AS05-13 | P | TC-AS05-13b: 409 {"code": "NO_WINDOW", "message": "Bu\u1ed5i n\u00e0y ch\u01b0a m\u1edf c\u1eeda s\u1ed5 \u |
| TC-AS05-14 | P | TC-AS05-14: {'checkin': (409, 'COURSE_ARCHIVED'), 'settings': (409, 'COURSE_ARCHIVED'), 'open': (409, 'COURSE_ |
| TC-AS05-15 | P | TC-AS05-15: [(200, 409, 'PRESENT', '0'), (200, 409, 'PRESENT', '0'), (200, 409, 'PRESENT', '0'), (200, 409, 'P |
| TC-AS05-16 | P | TC-AS05-16: {'sv.gioi': (403, 403), 'admin': (403, 403), 'sv.moi': (403, 403)} |
| TC-AS05-17 | P | TC-AS05-17b: Đánh vắng 8 sinh viên?  8 sinh viên chưa điểm danh thành Vắng.  Để sau Đánh vắng |
| TC-AS06-01 | P | TC-AS06-01b:  |
| TC-AS06-02 | P | TC-AS06-02d: 90 |
| TC-AS06-03 | P | TC-AS06-03: chỉ mục attendance_checkins_session_idx dùng khi lọc theo buổi (bitmap); kế hoạch có join chọn nat |
| TC-AS06-04 | P | TC-AS06-04b: {"cleared": true} |
| TC-AS06-05 | P | TC-AS06-05: 1->1 |
| TC-AS06-06 | P | TC-AS06-06c: Buổi 33: 2 lượt điểm danh cần xem · QC-ASr108 |
| TC-AS06-07 | P | TC-AS06-07a:  |
| TC-AS06-08 | P | TC-AS06-08b:  |
| TC-AS06-09 | P | TC-AS06-09:  |
| TC-AS06-10 | P | TC-AS06-10: {'windows': 200, 'checkins': 200, 'display': 200} |
| TC-AS06-11 | P | TC-AS06-11: {'sv.gioi': (403, 403), 'admin': (403, 403), 'sv.moi': (403, 403)} |
| TC-AS06-12 | P | TC-AS06-12: = TC-AS13-02a (as_f): danh sách buổi A sau purge trống |
| TC-AS07-01 | P | TC-AS07-01: Đang mở · 4/12 · còn 09:34 · 1 người trong lớp Số điểm danh nhiều hơn số người đếm Mở màn chiếu Đó |
| TC-AS07-02 | P | TC-AS07-02b:  |
| TC-AS07-03 | P | TC-AS07-03: NO_DEVICE_KEY,RECHECK_MISS |
| TC-AS07-04 | P | TC-AS07-04: [['NO_DEVICE_KEY'], ['NO_DEVICE_KEY'], ['NO_DEVICE_KEY'], ['NO_DEVICE_KEY'], ['NO_DEVICE_KEY']] |
| TC-AS07-05 | P | TC-AS07-05: nf=3 ['FLAGGED', 'FLAGGED', 'FLAGGED', 'RANDOM', 'RANDOM'] audit 571->571 |
| TC-AS07-06 | P | TC-AS07-06: 200 3 |
| TC-AS07-07 | P | TC-AS07-07: 1 tập |
| TC-AS07-08 | P | TC-AS07-08b: 404 |
| TC-AS07-09 | P | TC-AS07-09: {'sv.gioi': 403, 'admin': 403, 'sv.moi': 403, 'r108ta': 200} |
| TC-AS07-10 | P | TC-AS07-10c:  |
| TC-AS08-01 | P | TC-AS08-01b:  |
| TC-AS08-02 | P | TC-AS08-02: [1] |
| TC-AS08-03 | P | TC-AS08-03: /attendance/display?course=01a1df74-adfe-7424-802c-13b603b668d3&session=01a1e011-d3c4-7504-b1ae-8b |
| TC-AS08-04 | P | TC-AS08-04: nút=0; Đang mở · 2/12 · còn 09:56 Mở màn chiếu Đóng |
| TC-AS08-05 | P | TC-AS08-05: · 4/12 |
| TC-AS08-06 | P | TC-AS08-06:  |
| TC-AS08-07 | P | TC-AS08-07: Đã ghi 5 có mặt · 0 muộn · 7 chưa điểm danh |
| TC-AS08-08 | P | TC-AS08-08b: ABSENT=7 u=7 |
| TC-AS08-09 | P | TC-AS08-09:  |
| TC-AS08-10 | P | TC-AS08-10: thiếu:  |
| TC-AS08-11 | P | TC-AS08-11b: /attendance?course=01a1df74-adfe-7424-802c-13b603b668d3&session=01a1dfe4-3ea5-78d7-b6ee-9be0a5847 |
| TC-AS08-12 | P | TC-AS08-12: audit 3→4; undo=0 |
| TC-AS08-13 | P | TC-AS08-13: undo=1; Đang lưu… |
| TC-AS08-14 | P | TC-AS08-14b: Nhập số đầu người / Gọi ngẫu nhiên 5 người |
| TC-AS08-15 | P | TC-AS08-15: saved=1; gõ 2001 → lưu 2000 (giới hạn ≤ 2000) |
| TC-AS08-16 | P | TC-AS08-16: 5 hàng |
| TC-AS08-17 | P | TC-AS08-17b: https://localhost:1143/attendance/display?course=01a1df74-adfe-7424-802c-13b603b668d3&session=01a |
| TC-AS08-18 | P | TC-AS08-18: y=233.1875 |
| TC-AS08-19 | P | TC-AS08-19: {"h":88,"btns":[44,44],"ov":false,"dock":true} moOpenOutside=false |
| TC-AS08-20 | P | TC-AS08-20: rerun sau sửa b3c039c0: queued=1 closed=1 posts=4 keys=1 resp=fail:ep-68a17,fail:ep-68a17,fail:ep- |
| TC-AS08-21 | P | TC-AS08-21: codes=200,200 / Đã ghi 0 có mặt · 0 muộn · 12 chưa điểm danh Đánh vắng 12 SV chưa điểm danh / Đã g |
| TC-AS08-22 | P | TC-AS08-22: req(att)=1 req(display)=0 / Bỏ qua điều hướng  Toàn hệ thống · INT1006  Tìm nhanh hoặc đi đến… ⌘ K |
| TC-AS08-23 | P | TC-AS08-23: hasOpen=true put=403 |
| TC-AS08-24 | P | TC-AS08-24: 32/32 phép ✓, selftest 32/32 (evidence-10.7/ui-antipatterns*.log) |
| TC-AS08-25 | P | TC-AS08-25b:  |
| TC-AS08-30 | P | TC-AS08-30b: EEEAR7 vs PZF Z0E |
| TC-AS08-31 | P | TC-AS08-31c: overflow=false |
| TC-AS08-32 | P | TC-AS08-32: https://localhost/checkin?c=01a1df74-adfe-7424-802c-13b603b668d3#01a1e… → 201 |
| TC-AS08-33 | P | TC-AS08-33b: mã=6 qr=6 req=5/24s |
| TC-AS08-34 | P | TC-AS08-34: mã=4; countdown='còn 09:17' |
| TC-AS08-35 | P | TC-AS08-35: wakeLock=1; chữ=Màn chiếu điểm danh  Mã điểm danh  NQ3 4JX  0  còn 09:58  Đóng điểm danh Thoát |
| TC-AS08-36 | P | TC-AS08-36b: 10 s |
| TC-AS08-37 | P | TC-AS08-37:  |
| TC-AS08-38 | P | TC-AS08-38:  |
| TC-AS08-39 | P | TC-AS08-39b: https://localhost:1143/attendance?course=01a1df74-adfe-7424-802c-13b603b668d3&session=01a1e011-d3 |
| TC-AS08-40 | P | TC-AS08-40: Màn chiếu điểm danh Đã đóng  1  Thoát |
| TC-AS08-41 | P | TC-AS08-41b:  |
| TC-AS08-42 | P | TC-AS08-42: PV29N4 vs PV29N4 |
| TC-AS08-43 | P | TC-AS08-43: {"ov":false,"qr":62.59375,"code":527.28125,"cnt":598.40625} |
| TC-AS08-44 | P | TC-AS08-44: req(att)=1 req(display)=0 / Bỏ qua điều hướng  Toàn hệ thống · INT1006  Tìm nhanh hoặc đi đến… ⌘ K |
| TC-AS08-45 | P | TC-AS08-45: grep: 0 chỗ ngoài |
| TC-AS08-46 | P | TC-AS08-46: routes.ts:8 liệt kê /checkin; DESIGN §23.1/§14.8.1 route (present) ngoài khung |
| TC-AS08-47 | P | TC-AS08-47: 0; LCP=484 ms |
| TC-AS09-01 | P | TC-AS09-01: Đã điểm danh 22:48 |
| TC-AS09-02 | P | TC-AS09-02: hash='' history 2→3 |
| TC-AS09-03 | P | TC-AS09-03b:  ss={} |
| TC-AS09-04 | P | TC-AS09-04: TC-AS09-04a/04b PASS (ui-more4); lần đầu ngoại lệ do kịch bản (đã sửa) |
| TC-AS09-05 | P | TC-AS09-05: posts=0 / Bỏ qua điều hướng QD8B1 Có 50 thông báo mới 9+ H Điểm danh Mã đã hết hạn. Quét l |
| TC-AS09-06 | P | TC-AS09-06: ui-more5.log; lần ui-more6 trượt do cửa sổ còn mở từ ca debug |
| TC-AS09-07 | P | TC-AS09-07: Bỏ qua điều hướng QD8A3 Có 66 thông báo mới 9+ T Điểm danh  Đã điểm danh 23:48   rows=1 |
| TC-AS09-08 | P | TC-AS09-08: {"db":"edupilot.device.v1","keys":1,"extractable":false} WEB_KEY |
| TC-AS09-09 | P | TC-AS09-09: ["DEVICE_NEW,DEVICE_SHARED","DEVICE_SHARED,DEVICE_NEW"] |
| TC-AS09-10 | P | TC-AS09-10b:  |
| TC-AS09-11 | P | TC-AS09-11: sai:true, ngoài lớp:true, c không UUID:true, đã đóng:true, tắt:true |
| TC-AS09-12 | P | TC-AS09-12: = TC-AS03-24b (ui-real.log): 0 request auth sau 422 |
| TC-AS09-13 | P | TC-AS09-13: ui-more6: rows=1; lần 1 test gõ mã vào lớp đang chọn khác (QD8F3) |
| TC-AS09-14 | P | TC-AS09-14: req=0 / Bỏ qua điều hướng  Toàn hệ thống · INT1006  Có 6 thông báo mới 6 N Bạn không có  |
| TC-AS09-15 | P | TC-AS09-15:  |
| TC-AS09-20 | P | TC-AS09-20a:  |
| TC-AS09-21 | P | TC-AS09-21: WKQ FZ6 |
| TC-AS09-22 | P | TC-AS09-22: Thử lại sau 56 giây. |
| TC-AS09-23 | P | TC-AS09-23: ng hiện trên màn chiếu.  Lớp QC-ASr108  Điểm danh Hôm nay Chat riêng Hỏi đáp Luyện đề Thêm |
| TC-AS09-24 | P | TC-AS09-24: {"inputBottom":281,"dockTop":451,"vis":true} |
| TC-AS09-30 | P | TC-AS09-30b: null |
| TC-AS09-31 | P | TC-AS09-31: {"h1":["Điểm danh"],"panels":1,"meta":false,"nested":false,"css":2} |
| TC-AS09-32 | P | TC-AS09-32: ui-more4 (không dùng được IP LAN: giả lập môi trường không an toàn) |
| TC-AS09-33 | P | TC-AS09-33: manifest=200 name=EduPilot display=standalone icons=3 sw=0 |
| TC-AS09-34 | K | (a) cần iPhone/Safari/PWA thật (WebKit Playwright chỉ kiểm cấu hình; không có thiết bị) |
| TC-AS09-35 | P | TC-AS09-35a: default-src 'self'; script-src 'self' 'unsafe-inline'; style / camera=(self), geolocation=(self) |
| TC-AS09-36 | P | TC-AS09-36: rerun sau sửa P-11-39 (axe-appshell): 0; LCP=56 ms |
| TC-AS09-37 | P | TC-AS09-37: violations=0 |
| TC-AS09-38 | P | TC-AS09-38: như AS08-24 |
| TC-AS09-39 | P | TC-AS09-39a: rerun sau sửa P-11-39 (axe-appshell):  |
| TC-AS10-01 | P | TC-AS10-01: ['NO_DEVICE_KEY', '1', '', '', ''] |
| TC-AS10-02 | P | TC-AS10-02: geolocation=0 |
| TC-AS10-03 | P | TC-AS10-03: ['NO_DEVICE_KEY', '1', '50', '20', ''] |
| TC-AS10-04 | P | TC-AS10-04b: 1 |
| TC-AS10-05 | P | TC-AS10-05: ['NO_DEVICE_KEY', '1', '249', '50', ''] |
| TC-AS10-06 | P | TC-AS10-06b: ['NO_DEVICE_KEY', '1', '30', '100', ''] |
| TC-AS10-07 | P | TC-AS10-07: ['NO_DEVICE_KEY,GEO_DENIED', '1', '', '', ''] ['NO_DEVICE_KEY,GEO_DENIED', '1', '', '', ''] |
| TC-AS10-08 | P | TC-AS10-08b: 0 |
| TC-AS10-09 | P | TC-AS10-09: ['NO_DEVICE_KEY', '1', '', '', 't'] / ['NO_DEVICE_KEY,OFF_CAMPUS', '1', '', '', 'f'] |
| TC-AS10-10 | P | TC-AS10-10: ['NO_DEVICE_KEY,GEO_OUT,OFF_CAMPUS', '1', '1000', '10', 'f'] |
| TC-AS10-11 | P | TC-AS10-11b: DEVICE_NEW,GEO_DENIED/f |
| TC-AS10-12 | P | TC-AS10-12: {"self_checkin": true, "require": "WEB", "geo_check": true} |
| TC-AS10-13 | P | TC-AS10-13: {'sv.gioi': 201, 'sv.kha': 201, 'sv.nguyco': 201, 'sv04': 201, 'sv05': 201, 'sv06': 201, 'sv07': 2 |
| TC-AS10-14 | P | TC-AS10-14: [422, 422, 422, 422] |
| TC-AS10-15 | P | TC-AS10-15: [422, 422, 422, 422] |
| TC-AS10-16 | P | TC-AS10-16a: {"id": "01a1df3e-3918-7489-9e41-a2793c0dd3c2", "window_id": "01a1df3e-38e7-7eeb-8903-b77751f64b35 |
| TC-AS10-17 | P | TC-AS10-17: ['NO_DEVICE_KEY', '1', '10', '10', ''] / ['NO_DEVICE_KEY', '1', '', '', ''] |
| TC-AS10-18 | P | TC-AS10-18: [0, 0, 0, 0, 0, 0, 0, 0] |
| TC-AS12-01 | P | TC-AS12-01b:  |
| TC-AS12-02 | P | TC-AS12-02: [200, 200, 200, 429, 429] |
| TC-AS12-03 | P | TC-AS12-03d: 422 422 |
| TC-AS12-04 | P | TC-AS12-04b: 422 {"code": "DEVICE_CHALLENGE_INVALID", "message": "M\u00e3 th\u1eed th\u00e1ch h\u1ebft h\u1ea1 |
| TC-AS12-05 | P | TC-AS12-05: 422 {"code": "ATTESTATION_FAILED", "message": "Kh\u00f4ng x\u00e1c minh \u0111\u01b0\u1ee3c thi\u1 |
| TC-AS12-06 | P | TC-AS12-06: go-attest3.log: 3 chuỗi thật PASS, TestAttestRejects PASS (khi tải máy thấp; lượt tải cao hết hạn  |
| TC-AS12-07 | P | TC-AS12-07: TestAttestRejects PASS (go-attest3.log) |
| TC-AS12-08 | P | TC-AS12-08: 422 |
| TC-AS12-09 | P | TC-AS12-09: {'r108ta': (403, 403), 'r108t': (403, 403), 'admin': (403, 403)} 401 |
| TC-AS12-10 | P | TC-AS12-10: TestAttestRevocation PASS; điểm danh không gọi Google (mọi ca API 201 trong lúc đó) |
| TC-AS12-11 | P | TC-AS12-11: ['DEVICE_NEW', '2', 'WEB'] |
| TC-AS12-12 | P | TC-AS12-12: ['DEVICE_NEW', '3', '0', '10', ''] |
| TC-AS12-13 | P | TC-AS12-13: ['DEVICE_UNTRUSTED,DEVICE_NEW', '2', '0', '10', ''] |
| TC-AS12-14 | P | TC-AS12-14: ['DEVICE_NEW,GEO_MOCK', '2', '0', '10', ''] |
| TC-AS12-15 | P | TC-AS12-15: ['DEVICE_UNTRUSTED', '1', '0', '10', ''] |
| TC-AS12-16 | P | TC-AS12-16:  |
| TC-AS12-17 | P | TC-AS12-17:  |
| TC-AS12-18 | P | TC-AS12-18: go-attest3.log; git diff b064f3d4..9bcad074 -- go.mod rỗng |
| TC-AS12-19 | P | TC-AS12-19:  |
| TC-AS13-01 | P | TC-AS13-01c:  |
| TC-AS13-02 | P | TC-AS13-02a: 0 |
| TC-AS13-03 | P | TC-AS13-03: purge-checkins: 0 lượt, 0 thiết bị / ['purge-checkins: 0 lượt, 0 thiết bị', 'purge-checkins: 0 lượ |
| TC-AS13-04 | P | TC-AS13-04: 4320→0; 'purge-checkins: 4320 lượt, 0 thiết bị'; lô: limit 1000 |
| TC-AS13-05 | P | TC-AS13-05: go-attendance3.log PASS |
| TC-AS13-06 | P | TC-AS13-06: ['/ Lượt tự điểm danh (`attendance_checkins`): giờ quét, kênh, mức tin cậy, cờ, khoảng cách tới lớ |
| TC-AS13-07 | P | TC-AS13-07: [] |
| TC-AS13-08 | P | TC-AS13-08: (không có dòng log purge trong gateway/worker — CLI in ra stdout) |
| TC-AS13-09 | P | TC-AS13-09: prod rc=1 'admin purge-checkins: chỉ chạy khi APP_ENV thuộc dev, test (hiện: "production")'; http= |
| TC-AS13-10 | P | TC-AS13-10: ['/ Khoá thiết bị (`user_devices`): khoá công khai, băm, kết quả chứng thực. Không IMEI, không số  |
| TC-AS11-01 | P | TC-AS11-01: mobile/capacitor.config.js; AndroidManifest chỉ MAIN/LAUNCHER + CAMERA |
| TC-AS11-02 | P | TC-AS11-02: 0 tệp .jks/.keystore; storePassword đọc từ biến môi trường; .gitignore chặn |
| TC-AS11-03 | P | TC-AS11-03: git diff 10.7→10.8: +uqr; @capacitor/* chỉ ở mobile/package.json |
| TC-AS11-04 | P | TC-AS11-04: openapi: 0 'app-only'; app dùng POST checkins / me/devices |
| TC-AS11-05 | P | TC-AS11-05: posts=1 stopped=1 |
| TC-AS11-06 | P | TC-AS11-06: posts=1 |
| TC-AS11-07 | P | TC-AS11-07: rerun sau sửa b3c039c0:  |
| TC-AS11-08 | P | TC-AS11-08: danh  Lớp QD8F3  Chưa cấp quyền camera. Điểm danh Hôm nay Chat riêng Hỏi đáp Luyện đề Thêm |
| TC-AS11-09 | P | TC-AS11-09: trước=live sau=ended |
| TC-AS11-10 | P | TC-AS11-10: checkin-native.spec test 1 (dev-e2e-att.log) |
| TC-AS11-11 | P | TC-AS11-11: checkin-native.spec test 2 |
| TC-AS11-12 | P | TC-AS11-12: = TC-AS09-37 (header CSP /checkin, 0 vi phạm console) |
| TC-AS11-13 | P | TC-AS11-13: checkin-native.spec test 3 (trang lỗi errorPath: cấu hình capacitor.config.js errorPath=error.html |
| TC-AS11-14 | P | TC-AS11-14: [["ep_demo_state","ep_returning"],[]] |
| TC-AS11-15 | P | TC-AS11-15: README.md:4 |
| TC-AS11-16 | P | TC-AS11-16: có cả hai |
| TC-AS11-17 | K | (a) cần máy Android thật / emulator có camera: android-checklist.md AC-5…19 không chạy được |
| TC-AS14-01 | P | TC-AS14-01: rc=0,0 out='seed-checkin-demo: 0 lượt' 'seed-checkin-demo: 0 lượt' lượt 43->43 err= |
| TC-AS14-02 | P | TC-AS14-02e: 0 |
| TC-AS14-03 | P | TC-AS14-03: 270 -> 270 |
| TC-AS14-04 | P | TC-AS14-04: dev-e2e-att.log: 65 pass / 32 skip theo dự án; 1 ca attendance-display trượt khi chạy 2 worker, ch |
| TC-AS14-05 | P | TC-AS14-05: như AS14-04 |
| TC-AS14-06 | P | TC-AS14-06: p95 spread 46 ms, burst 25 ms (server p95 21 ms); lần 1–2 p95 cao do máy tải 30+; 1 SV 403 (không  |
| TC-AS14-07 | P | TC-AS14-07: MONITOR 40 s: 0 khoá ngoài codefail |
| TC-AS14-08 | P | TC-AS14-08b:  |
| TC-AS14-09 | P | TC-AS14-09: không chạy: go test -race ./... toàn bộ (ngoài sprint); contract không tag testroutes trượt (cần - |
| TC-AS14-10 | P | TC-AS14-10: git diff b064f3d4..9bcad074: 12 tệp test cũ chỉ đổi theo hợp đồng mới (env khoá, số thao tác 319→, |
| TC-AS14-11 | P | RERUN 2026-11-16 sau P-11-43 (BA vá SRS §11) + b8c43df8/c2867084 (dev viết đủ 13 test): cả 35 tên test ở SRS §11 tồn tại trong backend-go (thiếu 0) và PASS (attendance, attendance/attest, agent; evidence-10.8/as14-11-rerun.log) |
| TC-AS14-12 | P | TC-AS14-12: as_manual.py |
| TC-AS14-13 | P | TC-AS14-13: audit 10.7 (sau2/sau3): 0 lệch ở /attendance, /checkin, /display; 7 mã lệch còn lại ở observabilit |

## Trượt: bước tái hiện
Chi tiết, bước và nghi ngờ mã: `docs/regression/defects-10.8.md`.
- DEF-R10.8-01 axe `label-content-name-mismatch` (TC-AS08-25, AS09-36, AS09-39) — Trung bình; khung ứng dụng, 1440.
- DEF-R10.8-02 — ĐÃ SỬA (P-11-43, b8c43df8, c2867084), rerun 2026-11-16: TC-AS14-11 đạt.
- DEF-R10.8-03 «Đóng» khi mất mạng thử lại bằng Idempotency-Key mới (TC-AS08-20) — Thấp.
- DEF-R10.8-04 nút «Quét mã điểm danh» vẫn hiện khi `qr_code` không được hỗ trợ (TC-AS11-07) — Thấp.

## Ca K và lý do
| TC | Loại | Lý do |
| --- | --- | --- |
| TC-AS09-34 | (a) | cần iPhone / Safari / PWA thật; WebKit Playwright chỉ kiểm cấu hình `persist` |
| TC-AS11-17 | (a) | `android-checklist.md` chạy trên máy Android thật / emulator có camera; không có thiết bị |

## Phát hiện về hồi quy do sprint sau
Không phát hiện hồi quy hành vi do sprint 10.9–11 trên đường điểm danh (API, UI, purge, thiết bị). Các lỗi trên đều có sẵn từ 10.8 hoặc không rõ; DEF-01 (axe) có thể lộ ra do phiên bản axe-core hiện tại.

## TC-AS09-34 trên iOS Simulator (25/11)
Thiết bị `qc2-iphone17` (iPhone 17, iOS 27.2, Xcode 27.2), VPS `https://v2202609419695521098.happysrv.de` (chứng chỉ ZeroSSL thật, không cài CA), lớp QCVPSD, SV `sv.kha`; máy ảo đã tắt (không xoá). Điều khiển Safari / Màn hình chính bằng XCUITest (`evidence-10.8/ios/uitest`, `runtest.sh`); mã QR mở bằng `simctl openurl` (`ios/scan.py`). Ô mật khẩu gõ được khi bấm «Next» trên thanh bàn phím và chờ 2 s trước khi gõ.
Cờ `DEVICE_NEW` (`checkin.go:322`) có khi SV đã dùng một khoá khác trong lớp mà khoá này chưa dùng, nên «không cờ ở lần sau» chứng minh khoá cũ còn.

| Bước | Kết quả | Bằng chứng |
|---|---|---|
| Đăng nhập trong Safari iOS thật | đạt | `02-vps-after-login.png` |
| VPS cũ: điểm danh WEB (khoá ký) → tắt hẳn Safari → mở lại, điểm danh buổi khác | cả hai mức bảo đảm 2, không cờ | `03-…`, `04-…` |
| VPS DB mới: điểm danh A (buổi 1) → TẮT / BẬT máy ảo (`simctl shutdown`, `boot`; phiên vẫn còn) → điểm danh B (buổi 2) | A và B mức 2, không cờ ⇒ khoá IndexedDB còn sau khởi động lại máy | `05-…`, `06-…` |
| Thêm vào Màn hình chính (Share → Add to Home Screen) | đạt, biểu tượng EduPilot | `07-after-add-to-home.png` |
| PWA mở độc lập, kho lưu trữ riêng ⇒ phải đăng nhập lại; tắt hẳn PWA rồi mở lại | vẫn đăng nhập (phiên còn) | `08-pwa-after-login.png` |
| Điểm danh trong PWA | KHÔNG làm được: PWA không có lối vào `/checkin` (Hôm nay chỉ có thẻ buổi → Lịch; «Tìm nhanh» không có mục Điểm danh; «Thêm» không có mục này); `/checkin` chỉ vào qua đường dẫn QR, mà quét QR bằng camera iPhone mở Safari chứ không mở PWA | `09-pwa-search-khong-co-diem-danh.png`, `10-pwa-session.png` |
| Dùng sau > 7 ngày | K: simulator theo giờ máy chủ, đổi giờ máy chủ dự án bị cấm, Settings của simulator không cho đặt giờ riêng; cơ chế ITP 7 ngày không tái hiện được trong simulator | |

**Kết luận:** phần kiểm được đạt (khoá WebCrypto ECDSA trong IndexedDB còn qua tắt app và khởi động lại máy; cài vào Màn hình chính chạy; phiên PWA còn qua tắt app). Phần «> 7 ngày» và «điểm danh từ PWA» không kiểm được vì lý do trên; ghi nhận: khoá lưu ở kho của Safari, không phải của PWA (hai kho riêng trên iOS), nên điều kiện «cài vào Màn hình chính giữ quá 7 ngày» ở `deviceKey.ts` chỉ áp dụng nếu PM thêm lối vào `/checkin` trong PWA. Dòng chốt bên dưới là P cho phần đã kiểm, PM quyết giữ hay đổi.

| TC | KQ | Ghi chú |
|---|---|---|
| TC-AS09-34 | P | iOS Simulator 27.2 + VPS thật: khoá WebCrypto còn sau tắt app (Safari) và tắt/bật máy ảo (mức 2, không DEVICE_NEW); Thêm vào Màn hình chính + PWA đăng nhập còn; K: «> 7 ngày» (simulator không đổi giờ riêng) và điểm danh trong PWA (không có lối vào /checkin); xem mục trên |

## TC-AS11-17 trên Android emulator + VPS (25/11, dev chạy; qc2 ghi lại từ `evidence-10.8/android/ket-qua-android-vps-8302ecca.md`, 34 ảnh ở `evidence-10.8/android/shots-vps/`)
AVD `ep-api36` (Pixel 8, Android 16 / API 36), APK debug build từ HEAD (`APP_SERVER_URL` = VPS `https://v2202609419695521098.happysrv.de`, chứng chỉ ZeroSSL thật), VPS e8b2162d; emulator đã tắt.

| Mục (android-checklist) | Kết quả | Ghi chú |
|---|---|---|
| AC-0 mở app | P | trang landing của VPS, không cảnh báo chứng chỉ |
| AC-1 giữ phiên | P | đăng nhập `sv.gioi`; force-stop > 30 s, mở lại vẫn đăng nhập |
| AC-3 quét QR | P (jsQR) | `BarcodeDetector` của emulator «không hoạt động» ⇒ app không hiện nút quét (đúng sau P-11-131); với shim jsQR cài trước khi tải trang: «Đã điểm danh» 1,7 s, channel WEB, mức 2 |
| AC-3b QR lạ | P | QR chữ / origin khác / cùng origin khác đường dẫn: tiếp tục quét, 0 POST `/checkins` |
| AC-4 mã ngắn | P | 631 ms, channel CODE |
| AC-4b từ chối camera | P | «Chưa cấp quyền camera.» + ô mã |
| AC-7 đăng ký thiết bị | P | challenge 200 → `/me/devices` 422 (đến bước chứng thực, gốc phần mềm); điểm danh 201 mức 2 |
| AC-14 mã hết hạn | P | «Mã đã hết hạn…», ở lại `/checkin`, không đăng xuất |
| a splash + logo | P | icon logo thương hiệu; splash trắng + logo ≈ 0,5 s |
| b cutout + 3 nút + xoay | P (quan sát) | dock/thanh trên/thanh dưới không bị che; ngang đệm theo tai thỏ |
| c chế độ tối | P | WebView luôn light |
| d bàn phím | P | ô + «Điểm danh» thấy cùng lúc; phím «Đi» gửi; mã giữ nguyên |
| e mất mạng khi mở app | P | «Không kết nối được máy chủ.» + «Thử lại» → vào lại còn đăng nhập |
| f quyền Xấp xỉ | P | lớp geofence: assurance 1, `GEO_WEAK`, accuracy 2000 m |
| AC-16 nền / xoay | P 3/4 | ra nền: khung quét đóng, camera tắt; xoay khi quét vẫn quét |
| **US-AS-15 AC7(a) Back khi đang quét** | **F** | Back (phím / 3 nút / tap thật vào «Quét mã điểm danh» rồi Back) đưa cả app ra launcher thay vì đóng khung quét; phía web đúng (`history.state.epScan`, `history.back()` qua CDP đóng khung), vỏ Capacitor (`mobile/android/MainActivity.java`) không xử lý Back ⇒ DEF-R10.8-05 |
| AC-9 «trong app không NOT_APP» | K: emulator không chứng thực được ⇒ luôn `NOT_APP` (`checkin.go:330`, đúng thiết kế) | |
| AC-15 (mất mạng GIỮA lúc gửi), AC-18 (375 px mọi trạng thái), CL-8 (cấp lại quyền ở Cài đặt), AC-2, AC-8 | chưa chạy | để lượt chạy bộ mobile |

Phát hiện khác: gateway không nhận `APP_PUBLIC_URL` (compose chỉ truyền cho worker) nên `qr_url` / `join_url` / feed lịch dựng từ `https://localhost`, app bỏ qua QR khác origin (quét 0/N) ⇒ DEF-R10.8-06, đã sửa e8b2162d (VPS đang chạy bản đó, AC-3 sau đó P). Quan sát: chỉ quyền Xấp xỉ mà provider mạng của emulator chưa có vị trí thì `getCurrentPosition` hết 8 s ⇒ `GEO_DENIED` (máy thật có vị trí mạng ⇒ `GEO_WEAK`); hộp «Chính xác / Gần đúng» của Android 12+ hiện mỗi lần xin vị trí và làm trễ lượt (có thể `TOKEN_LATE`).

| TC | KQ | Ghi chú |
|---|---|---|
| TC-AS11-17 | F | Android emulator API 36 + VPS: 15 mục P (AC-0/1/3/3b/4/4b/7/14, a–f, AC-16 3/4); F: US-AS-15 AC7(a) Back khi đang quét thoát cả app (DEF-R10.8-05, dev đang sửa); K: AC-9 (emulator không chứng thực); chưa chạy: AC-15, AC-18, CL-8, AC-2, AC-8. Chạy lại sau khi sửa Back |

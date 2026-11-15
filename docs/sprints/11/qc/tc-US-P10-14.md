# QC test case — US-P10-14 (`/admin/audit`: nhật ký kiểm toán, không lộ nội dung)
Nguồn: `docs/specs/FEAT-p10-eval/US.md` (US-P10-14, AC1–AC11), `SRS.md` v1.7 FR-28, `openapi.yaml` `listAuditLog` (`AuditPage`, `AuditRow`), `QUESTIONS.md` Q-E-26 (mặc định: **không bao giờ** trả `before` / `after`). Hộp đen. TC ID `TC-14-nn`. Quy ước mã `P / F / K` theo `tc-00-chung.md` (K = không kiểm được, tính FAIL, ghi lý do).
Cập nhật theo BA `e6eba277` (SRS v1.8, US-P10-14 AC1/2/3/4/7): Q-S11-23…26 **đã chốt** — thiếu một đầu khoảng thời gian, tham số rỗng = bỏ qua, `before` / `after` không phải đối tượng → `[]`, **cursor gắn dấu vân tay bộ lọc** (khác → `422 INVALID_CURSOR`), `limit` mặc định **30**.

## Quy ước và dữ liệu
- `$B` = `https://localhost:<cổng ep-qc>/api/v1`; `$A` = JWT ADMIN; `$T`, `$TA`, `$S` = JWT `teacher@`, `ta@`, `sv.kha@`. `vl()` = `select count(*) from audit_log where action='VIEW_LIST'` (đo bằng psql trong container Postgres; bảng chỉ-thêm nên QC chỉ **INSERT** dòng thử, không sửa / xoá).
- **Dữ liệu thử (fixture QC, `entity='qc_audit'` để tách khỏi dữ liệu thật; dán `after`/`before` có chuỗi dò):**
  - F1: 230 dòng, `actor_id` = ADMIN, `created_at` cách nhau 1 phút, `after={"k":i}`.
  - F2: 60 dòng **cùng một `created_at`** (kiểm khoá `(created_at, id)`).
  - F3: 1 dòng `before={"cfg":"SECRET-CANARY-1","email":"canary1@sv.example","student_code":"20229001"}`, `after={"cfg":"SECRET-CANARY-2","reason":"SECRET-CANARY-3 lý do dài"}`.
  - F4: 1 dòng `actor_id = NULL` (hệ thống); 1 dòng `actor_id` = uuid không có trong `users` (người đã xoá).
  - F5: 1 dòng `after` có 25 khoá cấp 1 (`k01…k25`) và `before` có thêm `zz_before_only`; 1 dòng `before=NULL, after=NULL`; 1 dòng `after` là mảng `[1,2]`; 1 dòng `after` là chuỗi `"x"`.
  - F6: 3 dòng đúng mốc: `created_at = $F` (đúng `from`), `$F + 1 phút`, `$T0` (đúng `to`); 2 dòng `now() - 6 ngày 23 giờ` và `now() - 7 ngày 1 giờ`; 1 dòng `now() - 89 ngày`.
  - F7: ≥ 100.000 dòng `entity='qc_bulk'` (rải 60 ngày, 20 `entity` khác nhau, 50 `actor_id`) cho AC9; dọn bằng cách **không** dọn (bảng chỉ-thêm) — ghi trong báo cáo rằng DB ep-qc đã nạp để T1 cũng dùng.
- Mọi ca `422` / `401` / `403` kiểm thêm: `vl()` trước = sau (không ghi `VIEW_LIST`).

## AC1 — xem và lọc
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-14-01 | AC1 | F1 | `GET /admin/audit?entity=qc_audit&limit=5` (ADMIN) | `200`, `items.length = 5`, mỗi dòng đúng 11 khoá `id, at, actor_id, actor_name, actor_role, entity, entity_id, action, course_id, trace_id, changed_fields`; không khoá nào khác; khớp schema `AuditRow` | API |
| TC-14-02 | AC1 | F1 | Cùng lệnh | Sắp `(at, id)` **giảm dần** (kiểm cả khi trùng `at`); dòng mới nhất của F1 đứng đầu | API |
| TC-14-03 | AC1 | F1 | `actor_name` / `actor_role` của dòng F1 | Bằng tên / `ADMIN` của người thật (đối chiếu `users`) | API |
| TC-14-04 | AC1 | F4 | `entity=qc_audit` tìm dòng không `actor_id` và dòng actor đã xoá | Cả hai: `actor_id` đúng (hoặc `null` với dòng hệ thống), `actor_name = null`, `actor_role = null`; không lỗi `500` | API |
| TC-14-05 | AC1 | F1, dòng thật của sprint (đăng nhập, v.v.) | Lọc `entity=qc_audit` / `entity=audit_log` / không lọc | Chỉ dòng `entity` khớp **đúng** (`qc_audit` ≠ `qc_audit2`, không khớp tiền tố; hoa-thường: `QC_AUDIT` → `422`, xem TC-14-44) | API |
| TC-14-06 | AC1 | F1 | `actor_id=<ADMIN>` rồi `actor_id=<uuid lạ>` | Có dòng / `items: []` (200, không 404) | API |
| TC-14-07 | AC1 | F6 | `from=$F&to=$T0&entity=qc_audit` | Có dòng đúng `$F` và `$F+1 phút`; **không** có dòng đúng `$T0` (`from ≤ at < to`) | API |
| TC-14-08 | AC1 | F1, F6 | Lọc AND: `entity=qc_audit&actor_id=<ADMIN>&from=…&to=…` | Giao của ba điều kiện; đối chiếu `select` SQL cùng điều kiện: cùng số dòng, cùng thứ tự | API |
| TC-14-09 | AC1 | F6 | Không `from` / `to`, `entity=qc_audit` | Có dòng `now()-6d23h`; **không** có dòng `now()-7d1h` (mặc định 7 ngày) | API |
| TC-14-10 | AC1 | – | `from` có múi giờ `+07:00` và `Z` cùng một thời điểm | Cùng kết quả | API |
| TC-14-11 | AC1 | – | Trường `at` | RFC 3339, chính xác tới giây trở xuống, đúng với `created_at` trong DB (không lệch múi giờ) | API |

## AC2 — không lộ nội dung
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-14-12 | AC2 | F3 | `GET /admin/audit?entity=qc_audit` trang chứa dòng F3; lấy **toàn văn** thân | Không chứa `SECRET-CANARY-1/2/3`, `canary1@sv.example`, `20229001`; không khoá `before`, `after`, `payload`, `reason` ở **bất kỳ** cấp lồng nào (`jq '..'`) | API |
| TC-14-13 | AC2 | F3 | `changed_fields` của dòng F3 | Đúng `["cfg","email","reason","student_code"]` (hợp `before ∪ after`, tên khoá cấp 1, sắp chữ cái) | API |
| TC-14-14 | AC2 | F5 | Dòng 25 khoá | `changed_fields.length = 20`, sắp chữ cái, là **20 khoá đầu** của hợp (kể cả `zz_before_only` chỉ khi nằm trong 20) | API |
| TC-14-15 | AC2 | F5 | Dòng `before = NULL, after = NULL` | `changed_fields = []` (không `null`, không lỗi) | API |
| TC-14-16 | AC2 | F5 | Dòng `after` là mảng / chuỗi JSON | `200`, `changed_fields = []` (Q-S11-25 đã chốt); không `500` | API |
| TC-14-17 | AC2 | Dòng thật `VIEW_DETAIL` | Lọc `entity=llm_request` sau khi mở một yêu cầu | Chỉ có `changed_fields: ["reason"]`; thân không chứa chữ lý do đã nhập | API |
| TC-14-18 | AC2 | `student_profile`, cấu hình tích hợp, `llm_request` | Mở từng `entity` có sẵn trong DB (`select distinct entity`) | Với mọi `entity`, thân phản hồi không chứa giá trị của `before` / `after`: so từng giá trị chuỗi dài ≥ 6 ký tự lấy từ DB, 0 khớp | API+DB |
| TC-14-19 | AC2 | – | Log gateway sau mọi ca của file này | `docker compose logs gateway worker` không chứa `SECRET-CANARY`, `canary1@sv.example`, `20229001` | TAY |
| TC-14-20 | AC2 | – | `go test ./internal/audit -run TestAuditListNoValues -v` | `ok`; test seed chuỗi dò và khẳng định toàn văn không chứa | GO |

## AC3 — phân trang con trỏ
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-14-21 | AC3 | F1 (230), F2 | `limit` mặc định (không gửi), `entity=qc_audit` | `items.length = 30` (`httpx.DefaultLimit`); `next_cursor` khác `null` | API |
| TC-14-22 | AC3 | F1 + F2 (290 dòng) | Đi hết trang bằng `limit=50` rồi `limit=100` | Hợp các trang = 290 `id`, **không trùng, không sót** (so với `select id … order by created_at desc, id desc`); trang cuối `next_cursor = null` | API |
| TC-14-23 | AC3 | F2 (60 dòng cùng `created_at`) | `limit=7` đi hết | Không lặp / sót dòng giữa các trang dù trùng `created_at` | API |
| TC-14-24 | AC3 | F1 | Lấy trang 1, **INSERT 5 dòng mới** (`created_at = now()`), rồi lấy trang 2 bằng `next_cursor` | Trang 2 liền mạch với trang 1 (không trùng, không sót, không có 5 dòng mới); gọi mới lại thấy 5 dòng đầu | API |
| TC-14-25 | AC3 | – | `limit=1`, `limit=100`, `limit=30`, `limit=` (rỗng) | Trả đúng số dòng (nếu đủ); `100` không vượt `maxItems: 100`; `limit=` rỗng = như không gửi (30) | API |
| TC-14-26 | AC3 | F7 (≥ 100.000) | Đi tới trang 1.000 (`limit=100`) và đo thời gian từng trang | Thời gian trang cuối ≈ trang đầu (không tăng tuyến tính → không OFFSET); p95 ≤ 300 ms | API |
| TC-14-27 | AC3 | – | `grep -rn "OFFSET" backend-go/internal/audit backend-go/internal/store/queries/*audit*` | 0 khớp | DIFF |
| TC-14-28 | AC3 | – | Trang cuối | `next_cursor` là `null` (không phải chuỗi rỗng); `limit` đúng bằng số dòng còn lại → vẫn `null` hoặc trang rỗng kế (ghi quan sát, không FAIL) | API |
| TC-14-28b | AC3 | F1 | Lấy `next_cursor` với `entity=qc_audit&actor_id=<ADMIN>&from=$F&to=$T0`; dùng lại **đúng** bộ lọc, `limit` khác (10 → 25) | `200`, trang kế liền mạch (`limit` không nằm trong dấu vân tay) | API |
| TC-14-28c | AC3 | F1 | Cùng `next_cursor` nhưng đổi bộ lọc: (a) đổi `entity`, (b) thêm `actor_id`, (c) bỏ `actor_id`, (d) đổi `from`, (e) đổi `to`, (f) thêm `from` khi trước đó không có, (g) bỏ `entity` | Cả bảy: `422 INVALID_CURSOR` (không tiếp tục theo bộ lọc mới); `vl()` không đổi | API+DB |
| TC-14-28d | AC3 | – | Lấy `next_cursor` khi **không** gửi `from` / `to` (mặc định 7 ngày), dùng lại cũng không gửi, sau khi đợi 70 giây | `200` (dấu vân tay «thiếu = thiếu», không dời theo `now`); trang liền mạch | API |
| TC-14-28e | AC3 | – | Lấy `next_cursor` với `from=2026-11-14T03:00:00Z&to=2026-11-15T03:00:00Z`; dùng lại với cùng thời điểm viết khác dạng (`from=2026-11-14T10:00:00+07:00&to=2026-11-15T10:00:00+07:00`), và ngược lại (`+07:00` rồi `Z`) | `200`, trang kế liền mạch (Q-S11-27 chốt: dấu vân tay tính trên bộ lọc đã chuẩn hoá UTC; **không** `INVALID_CURSOR`); đổi thật một thời điểm (+1 giây) → `422 INVALID_CURSOR` | API |
| TC-14-28f | AC4 | – | `from=2026-11-14T10:00:00+07:00` | `VIEW_LIST.after.from` = `2026-11-14T03:00:00Z` (UTC) | API+DB |

## AC4 / AC5 — xem cũng bị ghi, ghi hỏng thì không trả
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-14-29 | AC4 | `n0 = vl()` | Một lần `GET /admin/audit?entity=qc_audit&actor_id=<ADMIN>&from=…&to=…` | `vl() = n0 + 1`; dòng mới: `entity='audit_log'`, `entity_id = <ADMIN id>`, `action='VIEW_LIST'`, `actor_id = ADMIN`, `trace_id` có; `after` có **đúng** các khoá `entity, actor_id, from, to, has_cursor, returned` (không khoá khác), giá trị khớp tham số / số dòng trả | API+DB |
| TC-14-30 | AC4 | – | Không tham số lọc | `after.entity = null`, `after.actor_id = null`; `after.from` / `after.to` là khoảng **đã áp dụng** (RFC 3339 UTC, không `null`): `to` ≈ thời điểm gọi, `from = to − 7 ngày` | API+DB |
| TC-14-31 | AC4 | – | Gọi trang 2 bằng `cursor` (cùng bộ lọc) | Thêm một dòng, `has_cursor = true`, `returned` = số dòng trang 2 | API+DB |
| TC-14-32 | AC4 | – | Gọi hai lần liên tiếp cùng tham số | Hai dòng `VIEW_LIST` khác `id`; dòng của lần 1 **có** ở phản hồi lần 2 (lọc `entity=audit_log`), **không** có ở phản hồi lần 1 | API |
| TC-14-33 | AC4 | – | Phản hồi lần 1: tìm `VIEW_LIST` của chính lần 1 (`entity=audit_log`) | Không có (dòng ghi "trong cùng giao dịch" nhưng không nằm trong chính phản hồi) | API |
| TC-14-34 | AC4 | – | 20 lần gọi **song song** | `vl()` tăng đúng 20; mỗi dòng `returned` đúng với phản hồi tương ứng | API+DB |
| TC-14-35 | AC4 | – | Dòng `VIEW_LIST` có chứa chữ trong kết quả (canary) không | `after` chỉ chứa tham số lọc + số đếm, 0 nội dung dòng | DB |
| TC-14-36 | AC5 | Trigger QC: `CREATE FUNCTION qc_block() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'qc'; END $$; CREATE TRIGGER qc_block BEFORE INSERT ON audit_log FOR EACH ROW WHEN (NEW.action = 'VIEW_LIST') EXECUTE FUNCTION qc_block();` | `GET /admin/audit?entity=qc_audit` | `500` (mã lỗi chuẩn `INTERNAL`), thân **không** có `items`, không dòng nhật ký nào; `vl()` không đổi; **khôi phục ngay** (`DROP TRIGGER`, `DROP FUNCTION`) và gọi lại → `200` | API+DB (TL báo trước; chỉ ep-qc) |
| TC-14-37 | AC5 | – | Sau TC-14-36 | Không có dòng `VIEW_LIST` "giả" cho lần lỗi; các dòng `audit_log` khác của ca không ghi nhầm | DB |
| TC-14-38 | AC4, AC5 | – | `go test ./internal/audit -run 'TestAuditViewIsAudited|TestAuditViewWriteFailClosed' -v` | `ok`; thứ tự ghi → trả được khẳng định | GO |

## AC6 — phân quyền
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-14-39 | AC6 | – | `GET /admin/audit` không `Authorization`; Bearer rác; Bearer hết hạn | `401` cả ba; thân theo chuẩn lỗi (không rò thông tin) | API |
| TC-14-40 | AC6 | – | `$T`, `$TA`, `$S` gọi `/admin/audit` (không tham số; `?entity=qc_audit`; `?actor_id=<chính mình>`; `?limit=1000`) | `403` tất cả — **không đổi theo tham số** (kể cả tham số sai: vai sai thắng `422`); `vl()` không đổi | API |
| TC-14-41 | AC6 | – | SV chưa vào lớp (`sv.moi`), tài khoản chưa xác minh email | `403` / `401` (không `200`) | API |
| TC-14-42 | AC6 | Admin đã bị khoá (DISABLED) hoặc phiên bị thu hồi | Gọi với token cũ | `401` | API |
| TC-14-43 | AC6 | – | `go test ./internal/audit -run TestAuditRBAC -v` | `ok` | GO |

## AC7 — tham số lỗi
| TC | AC | Bước / lệnh (ADMIN) | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- |
| TC-14-44 | AC7 | `entity=QC_AUDIT`, `entity=1abc`, `entity=a-b`, `entity=` + 65 ký tự, `entity=a%20b`, `entity=%20%20` (toàn khoảng trắng — không được cắt) | `422 VALIDATION_FAILED`, `details[]` có `field: "entity"` | API |
| TC-14-44b | AC1, AC7 | Tham số rỗng: `entity=`, `actor_id=`, `from=`, `to=`, `limit=`, `cursor=` (từng cái và tất cả cùng lúc) | `200`, kết quả **giống hệt** khi không gửi tham số đó (`items` cùng `id`, `limit` = 30, khoảng 7 ngày); `VIEW_LIST.after.entity = null` | API |
| TC-14-45 | AC7 | `actor_id=abc`, `actor_id=123`, `actor_id=<uuid thiếu 1 ký tự>` | `422`, `details[].field = "actor_id"` | API |
| TC-14-46 | AC7 | `from=2026-13-40`, `from=hôm nay`, `from=1700000000`, `to=2026-11-14` (không giờ) | `422`, `details[].field` là `from` / `to` đúng trường; (`to=2026-11-14` ngày không giờ: theo schema `date-time` → `422`) | API |
| TC-14-47 | AC7 | `from=2026-11-14T00:00:00Z&to=2026-11-14T00:00:00Z` và `from > to` | `422` (`from ≥ to`) | API |
| TC-14-48 | AC7 | `to − from` = đúng 90 ngày; = 90 ngày + 1 giây; = 365 ngày | 90 ngày: `200`; ≥ 90 ngày + 1 giây: `422` | API |
| TC-14-49 | AC7 | `limit=0`, `-1`, `101`, `abc`, `1.5` (`limit=` rỗng không lỗi: TC-14-44b) | `422`, `details[].field = "limit"` | API |
| TC-14-50 | AC7 | `cursor=abc`, `cursor=` hỏng base64, cursor sửa 1 ký tự (kể cả sửa dấu vân tay), cursor của **endpoint khác** (`next_cursor` của `GET /admin/users`) | `422 INVALID_CURSOR` (không `500`, không `200`); không ghi `VIEW_LIST` | API |
| TC-14-51 | AC7 | Nhiều lỗi cùng lúc (`entity` sai + `limit=0` + `from` sai) | `422` một lần, `details[]` có cả ba trường | API |
| TC-14-52 | AC7 | Mọi ca TC-14-44…51 | `vl()` không đổi | DB |
| TC-14-53 | AC7 | Khoảng hợp lệ không có dòng: `from=2001-01-01T00:00:00Z&to=2001-02-01T00:00:00Z` | `200 {items: [], next_cursor: null}` | API |
| TC-14-54 | AC1, AC7 | Chỉ `from`; chỉ `to`; `from` cũ hơn 90 ngày so với `now` không `to` | Chỉ `from` → `to = now` (`VIEW_LIST.after.to` ≈ lúc gọi); chỉ `to` → `from = to − 7 ngày` (dòng cách `to` 6 ngày có, 8 ngày không); `from = now−89d` không `to` → `200`; `from = now−91d` không `to` → `422` | API |
| TC-14-55 | AC7 | `go test ./internal/audit -run TestAuditListFilters -v` và `-run TestAuditCursor -v` | `ok` | GO |

## AC8 — không cache, chỉ đọc
| TC | AC | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- |
| TC-14-56 | AC8 | `curl -skI` / `-i` phản hồi `200` (mọi biến thể tham số, có / không `cursor`) | `Cache-Control: no-store` có mặt; không `ETag` / `Last-Modified` cho phép cache | API |
| TC-14-57 | AC8 | Qua Caddy (HTTPS cùng origin) | Cùng header (proxy không ghi đè) | API |
| TC-14-58 | AC8 | `POST` / `PUT` / `PATCH` / `DELETE /admin/audit`; `GET /admin/audit/export`, `/admin/audit/{id}`, `/admin/audit.csv` (ADMIN) | `404` hoặc `405`; không có route sửa / xoá / xuất (`grep` router + `openapi.yaml`: chỉ `listAuditLog`) | API+DIFF |
| TC-14-59 | AC8 | psql: `UPDATE audit_log SET action='x' WHERE id=…` và `DELETE FROM audit_log WHERE id=…` | Cả hai bị trigger chặn (lỗi append-only) — bảng vẫn chỉ-thêm sau migration chỉ mục mới | DB |
| TC-14-60 | AC8 | Đếm trước / sau một lần `GET`: dòng mỗi bảng (`audit_log`, `outbox`, `notifications`, `jobs`, `llm_audit`) | Chỉ `audit_log` +1; các bảng khác không đổi | DB |

## AC9 — hiệu năng và chỉ mục
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-14-61 | AC9 | – | `git diff <gốc 860cad1b^>..HEAD --stat -- backend-go/db/migrations` | Chỉ **thêm** một file migration mới (số kế tiếp); `00001…` không bị sửa; có `Down` xoá đúng hai chỉ mục | DIFF |
| TC-14-62 | AC9 | F7 | `\d audit_log` | Có hai chỉ mục `(created_at DESC, id DESC)` và `(entity, created_at DESC, id DESC)` | DB |
| TC-14-63 | AC9 | F7 (≥ 100.000, `ANALYZE audit_log` rồi) | `EXPLAIN (ANALYZE, BUFFERS)` ba hình dạng truy vấn (không lọc / `entity` / `actor_id`; mỗi cái có `ORDER BY created_at DESC, id DESC LIMIT 51` và điều kiện 7 ngày) | Không `Seq Scan on audit_log`; dùng chỉ mục; thời gian thực thi ≪ 300 ms | DB |
| TC-14-64 | AC9 | F7 | `idx_scan` của hai chỉ mục mới trong `pg_stat_user_indexes` trước / sau 30 lần gọi API mỗi hình dạng | Tăng (API thật sự dùng chúng, không chỉ SQL của QC) | API+DB |
| TC-14-65 | AC9 | F7 | 100 lần gọi tuần tự mỗi hình dạng (không lọc, `entity=qc_bulk`, `actor_id`), đo p50 / p95 / max | p95 ≤ 300 ms (ghi cả khi chạy qua Caddy/PgBouncer) | API |
| TC-14-66 | AC9 | F7 | 20 lần gọi song song × 5 vòng | p95 ≤ 300 ms; không `5xx`; `vl()` tăng đúng | API |
| TC-14-67 | AC9 | – | `go test ./internal/audit -run TestAuditListUsesIndex -v` | `ok` | GO |
| TC-14-68 | AC9 | – | `entity` lọc dòng cực ít (1 dòng trong 100.000) và `actor_id` lạ | p95 ≤ 300 ms; không quét bảng | API |

## AC10 — giao diện `/admin/audit` (1440 và 375)
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-14-69 | AC10 | ADMIN | Mở `/admin/audit` | Một bảng, cột đúng thứ tự **Thời điểm · Người · Thực thể · Hành động · Khoá đổi**; ba bộ lọc **Thực thể**, **Người**, **Khoảng thời gian**; nút `Tải thêm`; ảnh 1440 | E2E |
| TC-14-70 | AC10 | – | Lọc từng bộ lọc rồi kết hợp | Bảng khớp API cùng tham số (đếm dòng, thứ tự); xoá lọc về mặc định 7 ngày | E2E |
| TC-14-71 | AC10 | F1 | `Tải thêm` tới hết | Dòng nối tiếp không trùng / sót; sau trang cuối nút ẩn hoặc khoá, không còn gọi API; mỗi lần bấm một lần gọi | E2E |
| TC-14-72 | AC10 | – | Loading | Khung xương / trạng thái tải của `PageState` (không spinner riêng); đo bằng làm chậm route 2 s | E2E |
| TC-14-73 | AC10 | `entity` không có dòng | Rỗng | Đúng chữ «Không có dòng nào trong khoảng này» | E2E |
| TC-14-74 | AC10 | Chặn route trả `500` / mất mạng | Lỗi | Một dòng lỗi + `Thử lại` (không alert chồng); `Thử lại` gọi lại và hiện bảng | E2E |
| TC-14-75 | AC10 | – | Chữ dưới tiêu đề và dưới từng khối | **Không** dòng giải thích dưới tiêu đề (luật chữ UI của PO); không tooltip thừa / icon Info; chữ không có từ kỹ thuật (`before`, `after`, `JSON`, `payload`) | E2E+TAY |
| TC-14-76 | AC10 | – | Cột «Khoá đổi» | Hiện tên khoá (ví dụ `reason`), không giá trị; dòng nhiều khoá không làm hàng cao > 48 px hoặc có cách rút gọn | E2E |
| TC-14-77 | AC10 | 375 px | Mở trang, mọi trạng thái | `scrollWidth ≤ clientWidth`; vùng chạm ≥ 44 px; không chữ bị cắt; ảnh 375 | E2E |
| TC-14-78 | AC10 | TEACHER, TA, STUDENT | Mở thẳng `/admin/audit` | Trang 403 chung (cùng mẫu trang chặn quyền của `/admin/users`); 0 lời gọi `GET /admin/audit` thành công; menu **không** có mục | E2E |
| TC-14-79 | AC10 | Chưa đăng nhập | Mở thẳng | Về `/login?next=…`, sau đăng nhập ADMIN quay lại `/admin/audit` | E2E |
| TC-14-80 | AC10 | – | DOM | Dùng `DataTable` / `PageState` ở `shared/` (không bảng tự dựng): có móc `data-ep-*` của bảng chung; `bash scripts/ui-antipatterns.sh` sạch | E2E+DIFF |
| TC-14-81 | AC10 | – | axe (WCAG 2.2 AA) ở 1440 và 375, bàn phím (Tab qua bộ lọc → bảng → `Tải thêm`) | 0 lỗi nghiêm trọng; thứ tự Tab hợp lý; bộ lọc có nhãn | E2E |
| TC-14-82 | AC10 | – | Bộ lọc «Người» | Chọn / nhập người thao tác theo tên hoặc email (không bắt dán uuid); không hiện email người khác ngoài ADMIN | E2E |
| TC-14-83 | AC10 | – | `pnpm -C frontend exec playwright test audit.spec.ts` | xanh (CI của dev) | E2E |

## AC11 — đối chiếu US-P10-02
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-14-84 | AC11 | Một yêu cầu LLM có payload | ADMIN `POST /admin/observability/requests/{id}/open {"reason":"QC kiểm tra nhật ký 14"}` rồi `GET /admin/audit?entity=llm_request` | Có dòng `VIEW_DETAIL`: `actor_id` = ADMIN, `entity_id` = id yêu cầu, `changed_fields = ["reason"]`; thân **không** có chữ «QC kiểm tra nhật ký 14» | API |
| TC-14-85 | AC11 | – | Mở cùng yêu cầu 2 lần | Hai dòng `VIEW_DETAIL` | API |
| TC-14-86 | AC11 | – | Mở qua giao diện `/observability` rồi lọc ở `/admin/audit` | Dòng xuất hiện trên bảng, cột «Khoá đổi» = `reason` | E2E |
| TC-14-87 | AC11 | – | `llm_request` bị từ chối (lý do < 10 ký tự, `422`) | **Không** có dòng `VIEW_DETAIL` | API |

## Tổng hợp, riêng tư, hợp đồng
| TC | AC | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- |
| TC-14-88 | AC1–9 | `go test ./internal/audit -run 'TestAuditListNoValues|TestAuditViewIsAudited|TestAuditRBAC|TestAuditListFilters|TestAuditCursor|TestAuditListUsesIndex|TestAuditViewWriteFailClosed' -v` | `ok` cả bảy; QC đối chiếu tên với AC | GO |
| TC-14-89 | hợp đồng | `go test ./internal/contract/...` | xanh; có `listAuditLog`; phản hồi thật khớp `AuditPage` / `AuditRow` (kể cả `changed_fields maxItems 20`) | GO |
| TC-14-90 | hợp đồng | `grep listAuditLog backend-go/api/openapi.yaml`; thực tế: mọi khoá trong phản hồi nằm trong `AuditRow`; không khoá dư | API+DIFF |
| TC-14-91 | riêng tư | Trang `/admin/audit` ở ADMIN: mở DevTools mạng | Không có `before`, `after` trong bất kỳ phản hồi nào; không lời gọi nào ngoài `GET /admin/audit` (+ tải người thao tác nếu có) trả nội dung nhạy cảm | E2E |
| TC-14-92 | riêng tư | Tên / email người thao tác trong bảng | Chỉ ADMIN thấy (đã ép bởi `403`); không có MSSV | API |
| TC-14-93 | hồi quy | Sau toàn bộ ca: `/observability` mở chi tiết, `/admin/users`, đăng nhập | Không đổi hành vi; dòng `audit_log` cũ giữ nguyên số lượng (trừ dòng thêm) | API |
| TC-14-94 | F16 | `docs/FLOWS.md` F16 + `e2e-flows.md` | F16 ghi `/admin/audit` đã có; `/admin/health` thuộc PR (ngoài phạm vi, **không** tính FAIL) | DIFF |

## Câu hỏi đã chốt và câu hỏi mới
| ID | Trạng thái |
| --- | --- |
| Q-S11-23 (thiếu một đầu khoảng thời gian; `VIEW_LIST.after.from/to`) | Chốt (BA `e6eba277`): chỉ `from` → `to = now`; chỉ `to` → `from = to − 7 ngày`; `after` ghi khoảng đã áp dụng (UTC); trần 90 ngày trên khoảng đã áp dụng. Khớp đề xuất QC |
| Q-S11-24 (tham số rỗng) | Chốt: bỏ qua như không gửi; `entity` toàn khoảng trắng → `422`. Khớp |
| Q-S11-25 (`before`/`after` không phải đối tượng) | Chốt: `[]`, không `500`. Khớp |
| Q-S11-26 (cursor dùng lại với bộ lọc khác) | Chốt: **`422 INVALID_CURSOR`**, dấu vân tay `(entity, actor_id, from, to)` đúng như client gửi. **Khác** đề xuất QC; TC đổi theo (TC-14-28b…f) |
| `limit` mặc định | **30** (không phải 50); TC-14-21, 25 đổi theo |
| Q-S11-27 (dấu vân tay và dạng chữ thời điểm) | Chốt (BA `d7f7d0bc`): tính trên bộ lọc đã chuẩn hoá UTC; `Z` và `+07:00` cùng thời điểm = cùng bộ lọc. Khớp đề xuất QC; TC-14-28e kiểm cả hai chiều |

## Việc QC làm khi dev báo xong
1. Nạp F1…F7 (INSERT qua psql, ghi lệnh vào `scripts/`), `ANALYZE audit_log`. 2. Chạy `scripts/au_*.py` (API+DB) và `frontend/e2e/qc-real-audit.spec.ts` (UI 1440 / 375, ảnh vào `docs/sprints/11/qc/shots/`). 3. Chạy các lệnh Go (TC-14-20, 38, 43, 55, 67, 88, 89) và `audit.spec.ts` ở nhánh dev. 4. TC-14-36 cần PM / TL cho phép tạo trigger tạm trên DB ep-qc (khôi phục ngay).

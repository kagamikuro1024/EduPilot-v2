# QC sprint 10.8 (FEAT-attendance-selfcheck) — quy ước chung của bộ test case

Viết ở **pha 1** (2026-11-15) từ `docs/specs/FEAT-attendance-selfcheck/{US,SRS,QUESTIONS}.md` (SRS v1.5; US-AS-01…14) và `docs/sprints/10.8/proposals.md` (P-108-1…11), nhánh `sprint/10.8-attendance`. Hộp đen: không đọc mã, handoff hay test của dev. **Chưa chạy gì.** Câu hỏi mở: `questions.md` (Q-S108-n). Mã TC: `TC-AS<US>-<nn>`; AC dẫn theo US.md (ví dụ AC3a). Hàm trợ giúp / script QC viết ở pha 2 trong `scripts/`.

## Luật chạy (áp cho MỌI TC của sprint này)
| Mục | Quy tắc |
| --- | --- |
| Stack | `ep-qc` dựng từ commit handoff của dev (migration `00024` chạy trên **DB trống + seed** và trên **DB đã có dữ liệu 10.7/11**); chỉ dựng khi PM báo tới lượt; tối đa demo + 1 stack; dọn `docker rm -f edupilot-test-*` sau `go test` |
| LLM | Feature **không dùng LLM** (không tool agent mới). 9router / embedding / decision: không cần; ghi `llm_audit` không đổi sau mỗi story (kiểm chéo) |
| Gốc web | HTTP `http://localhost:580` và HTTPS `https://localhost:943` đều là «ngữ cảnh an toàn» (localhost): WebCrypto, `getUserMedia`, Geolocation chạy được. TC nào cần HTTP **không** an toàn (US-AS-09 AC7) dùng IP LAN của máy (`http://<ip-lan>:580`) hoặc chế độ riêng tư (IndexedDB bị chặn) |
| Giờ | Cửa sổ mặc định 10 phút, `rotate_seconds` miền [5, 60]. TC chờ hết hạn dùng **`rotate_seconds=5`** (tuổi mã tối đa 10 s ⇒ chờ ≥ 12 s) và **`minutes=1`** (hết giờ sau 60 s; worker quét ≤ 60 s ⇒ chờ ≤ 125 s). Không dùng `faketime`; không đổi đồng hồ máy chủ |
| Khoá | `ATTENDANCE_TOKEN_KEY` (cố định trong compose dev) **không** ghi vào báo cáo; QC tự tính MAC chỉ để dựng ca «MAC sai / step ngoài hai bước» khi chờ thật quá tốn thời gian (xem Q-S108-3) |
| Bản chạy tay Android | Máy thật + emulator: `android-checklist.md`; không bắt buộc máy thật để PASS story (US-AS-11 AC8) nhưng mức 3 chỉ chứng minh bằng chuỗi chứng chỉ thật |
| Khoá bí mật APK | Kho khoá ký APK ngoài repo; QC không đọc, không ghi lại |

## Nhân vật và dữ liệu
| Tên | Dùng |
| --- | --- |
| Lớp **QC-AS** | Admin mở lớp mới (`POST /admin/courses`), gán `teacher@`, `ta@`; 6 SV vào lớp bằng mã tham gia: `sv.gioi`, `sv.kha`, `sv.nguyco` + `sv31`, `sv32`, `sv33` (SV seed lớp 761988 hoặc tài khoản mới); 1 SV `PENDING` + 1 SV đã bị mời ra; `sv.moi` KHÔNG vào lớp. Tạo lịch bằng `sessions/generate` (≥ 6 buổi, một buổi `starts_at` quá khứ 1 giờ để kiểm `LATE`). Bật `self_checkin` bằng `PUT …/settings` |
| Lớp **761988 (seed demo)** | Chỉ cho TC seed / E2E của US-AS-14 (dữ liệu `seed-checkin-demo`) |
| Lớp **761987** | **Không đụng** (270 `attendance_records` và `seed/expected_crm.csv` phải y nguyên — kiểm ở TC-AS14) |
| Lớp lưu trữ | Lớp QC-AS-2: bật `self_checkin`, mở một cửa sổ, rồi `POST /admin/courses/{id}/archive` |
| Khoá thiết bị | QC sinh cặp ECDSA P-256 (Node WebCrypto / `cryptography`): chữ ký `r‖s` 64 byte cho `WEB_KEY` web, DER cho Keystore; `public_key` SPKI base64. Mức 3 / `GEO_MOCK` / `NOT_APP` cần khoá `ANDROID_KEY attested=true` — **xin cách dựng ở Q-S108-2** |

## Quan sát được (hộp đen)
API SRS §6 #1–16; `attendance_checkins` / `checkin_windows` / `user_devices` / `attendance_records` / `audit_log` (SQL chỉ đọc, `docker exec ep-qc-postgres-1 psql`); log gateway (`docker logs`); Redis `ep:att:codefail:{uid}`, `ep:dev:chal:{uid}` (chỉ để xem TTL); outbox / `notifications` (kiểm «không thông báo cho SV khi mở vòng ≥ 2»); `/me/today`; giao diện (Playwright 1440 + 375). Mailpit (không có mail nào từ feature này).

## Phủ story × file
| Story | File |
| --- | --- |
| US-AS-01, 02 (chính sách lớp, cửa sổ) | `tc-US-AS-01-02.md` |
| US-AS-03, 04 (điểm danh bằng mã / mã ngắn, cờ, mức tin cậy) | `tc-US-AS-03-04.md` |
| US-AS-05, 06, 07 (đóng, đánh vắng, danh sách cờ, Hôm nay, vòng 2, gọi tên) | `tc-US-AS-05-07.md` |
| US-AS-08 (`/attendance`, màn chiếu) | `tc-US-AS-08.md` |
| US-AS-09 (`/checkin` web, **375 px**) | `tc-US-AS-09.md` |
| US-AS-10, 12, 13 (vị trí / mạng, chứng thực thiết bị, giữ 30 ngày) | `tc-US-AS-10-12-13.md` |
| US-AS-11 (app Android) + US-AS-14 (seed, E2E, k6, axe) + truy vết P-108-4…11 | `tc-US-AS-11-14.md`, `android-checklist.md` |

## Kiểm chéo áp cho mọi story (pha 2)
- **Phân quyền 4 vai + người ngoài:** mỗi route Staff: TEACHER, TA ✓; STUDENT, ADMIN → `403`; SV lớp khác / `sv.moi` → `403` lý do `course` (P-108-11, không `404`, không mã `NOT_IN_COURSE`); không JWT → `401`. Route SV: TA, TEACHER, ADMIN → `403`.
- **Sinh viên không bao giờ thấy cờ / `assurance` / thiết bị / vị trí đã lưu:** `grep -E 'flags|assurance|device|geo_|in_campus'` trên mọi phản hồi mà SV gọi (`checkins`, `me/attendance`, `GET /courses/{id}`, `me/today`) = 0.
- **Không PII / bí mật trong log:** sau mỗi story `docker logs ep-qc-gateway-1 | grep -E 'token=|"token"|signature|public_key|lat|lng|\b\d{1,3}(\.\d{1,3}){3}\b'` ra 0 dòng thuộc luồng điểm danh (IP máy khách không xuất hiện); không toạ độ.
- **Chỉ gắn cờ, không tự phạt:** mọi ca tín hiệu bất thường vẫn `201` (US-AS-03 AC6).
- **Điểm danh tay không đổi:** `PUT …/attendance` ở lớp tắt `self_checkin` cho cùng phản hồi (so byte) trước / sau 10.8 (TC-AS01-01).
- **`version` / idempotency:** `PUT` chính sách và lưới có `version` → `409 VERSION_CONFLICT`; mở cửa sổ / `absent-unchecked` có `Idempotency-Key`; gửi lại cùng khoá → cùng kết quả.
- **Lớp lưu trữ:** mọi ghi → `409 COURSE_ARCHIVED`; đọc được.
- **Phân trang:** `GET …/checkins` con trỏ `limit ≤ 100`, `limit=101` → `422`; không OFFSET.
- **UI:** mỗi màn có tải (khung xương) / rỗng / lỗi; `bash scripts/ui-antipatterns.sh` sạch; chỉ token `--ep-*`; chữ UI theo luật cứng PO (không câu giải thích, tooltip khi cần); không gamify; 1440 + 375; axe 0 nghiêm trọng.
- **Mã lỗi riêng (P-108-11):** QR / mã ngắn dùng `CHECKIN_TOKEN_INVALID` / `CHECKIN_TOKEN_EXPIRED`; `apiClient` **không** refresh phiên và **không** đăng xuất khi gặp hai mã này.

## Lịch sử sửa TC
- 2026-11-15: bản đầu (pha 1).

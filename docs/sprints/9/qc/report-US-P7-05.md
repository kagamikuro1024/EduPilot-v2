# QC report — US-P7-05 (`ImapSource`, cấu hình IMAP, thư không xử lý được) · Kết luận vòng 1: **FAIL** (AC1 phần Playwright và AC9: chưa có giao diện — `/settings/integrations` vẫn là mock; phần máy chủ PASS)
Ngày 2026-11-01. Commit kiểm: `sprint/9-p7` @ `e7aeff77` (chỉ có phần máy chủ). Bộ TC: `tc-US-P7-05.md`. Handoff: `dev-US-P7-05.md` (mục "Còn lại (FE)"). CI `e7aeff77`: **success**.

## 1. Cổng đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `p7-go-tests.sh`, `-race -tags integration`, mọi tên của dòng `Kiểm:` (`internal/integration`) | PASS 35/35 (máy chủ IMAP trong test `imapserver` + `imapmemserver`, MinIO + Postgres + Redis thật) |
| Hộp đen trên stack thật (`scripts/p7-api-05.py`; stack không có máy chủ IMAP → poll / xác thực người gửi chỉ kiểm bằng Go) | PASS: chưa cấu hình `configured false`; lần đầu thiếu mật khẩu → 422; ADMIN lưu → 200; phản hồi PUT / GET (ADMIN, TEACHER) **không** có mật khẩu hay đuôi mật khẩu; `config_enc` là bản mã, `config_public`, `audit_log`, log gateway + worker không chứa canary `CANARY-IMAP-PW-7Z3`; sai `version` → 409; mật khẩu trống = giữ cũ; `security NONE` → 409 ở môi trường hiện tại; `test` tới máy chủ không tồn tại → `{ok:false, kind:UNREACHABLE/TIMEOUT}` ≤ 12 s, `last_error` là mã ngắn. Phân quyền: TA / STUDENT `GET imap` 403; TEACHER / TA / STUDENT `PUT` và `test` 403; `sync-now` TA 202, lần hai trong 30 s 429 (`retry_after`), ADMIN / STUDENT 403; `mail-inbound` TA, TEACHER 200, STUDENT, ADMIN 403; lớp 2 không thấy cấu hình lớp 1; SMTP chỉ đọc, không mật khẩu. 1 ca FAIL ở lần chạy đầu là **giả định sai của QC** (`TEACHER` seed dạy cả lớp 1 và lớp 2): đã sửa script, xác nhận bằng DB |
| Đột biến (`/tmp/qc-mut9`, đã xoá) I1 tin `From` bất kể `Authentication-Results` · I3 bỏ `UNIQUE (course_id, uidvalidity, message_uid)` | cả 2 **đỏ**: `TestImapSpoofedFromNotMatched` + `TestImapAuthResultsTrustedOnly` + `…FirstHeader` + `…NoTrustedIdUnmatched` + `TestImapReceiptBellToMatchedStudent`; `TestImapPollNewOnly` + `…PollIdempotentRestart` + `…UidValidityChange` + … |
| Playwright `settings-integrations.spec.ts` | **không tồn tại**; `features/settings/IntegrationsSettings.tsx` vẫn import `INTEGRATIONS` từ `@/mock/system` |

## 2. Kết quả theo AC
| AC | Kết quả | Ghi chú |
| --- | --- | --- |
| AC1 cấu hình IMAP — mật khẩu chỉ ghi, mã hoá | **FAIL** | Máy chủ PASS (hộp đen, `TestImapConfigSecretWriteOnly`, `…Encrypted`, `…VersionConflict`, `…AuditNoSecret`). `Kiểm:` còn `playwright test settings-integrations.spec.ts -g 'imap key write-only'` → chưa có |
| AC2 `Test kết nối` — nguyên nhân rõ | PASS | hộp đen + `TestImapTestOK`, `…AuthFail`, `…Unreachable`, `…MailboxMissing`, `…Timeout`; ca `TLS` chưa có test (dev tự ghi) |
| AC3 poll — chỉ thư chưa xử lý, idempotent, một poller | PASS ở mức Go | I3; `TestImapPollNewOnly`, `…IdempotentRestart`, `…UidValidityChange`, `…AutoReplyIgnored`, `…SinglePoller` |
| AC4 đính kèm và khớp theo tiêu đề | PASS ở mức Go | `TestImapOneAttachment`, `…NoAttachment`, `…TooManyAttachments`, `…AttachmentRejected`, `…MatchBySubject`, `…UnmatchedGoesToTab` |
| AC5 mail → khớp → chấm nháp, đi trọn | PASS ở mức Go | `TestImapToDraftGrade` (nháp = mẫu S2) |
| AC6 máy chủ thư chết, sai mật khẩu; không mất thư | PASS ở mức Go | `TestImapBackoff`, `…NoMailLostOnOutage`, `…AuthFailPausesPoll`, `…BadMimeSkipped`; hộp đen: `last_error` mã ngắn |
| AC7 `Đồng bộ ngay` 202 + job, giới hạn tần suất | PASS | hộp đen (202, 429 + `retry_after`); `TestImapSyncNowJob`, `…RateLimit`, `…NoOverlap` |
| AC8 phân quyền | PASS | hộp đen ma trận; `TestIntegrationsRBACMatrix`, `TestImapCourseIsolation` |
| **AC9 giao diện `/settings/integrations` phẳng, luật chữ, trạng thái** | **FAIL** | chưa làm (mock); `settings-integrations.spec.ts` chưa có |
| AC10 thư giả mạo người gửi (TLR-2) | PASS ở mức Go | I1; `TestImapSpoofedFromNotMatched`, `…AuthResultsTrustedOnly`, `…AuthResultsFirstHeader`, `…NoTrustedIdUnmatched`, `…ReceiptBellToMatchedStudent` |

## 3. Lỗi / ghi nhận
- **BUG-P7-05-1 (Cao, AC1 + AC9):** frontend chưa làm — `IntegrationsSettings.tsx` còn dùng mock, thiếu `settings-integrations.spec.ts` (`imap key write-only`, `flat sections`, `states`, `no description lines`), thiếu mục "Thư không xử lý được" ở tab `Chưa khớp`. Dev giao FE rồi QC chấm lại vòng 2 (máy chủ chỉ cần hồi quy).
- Chưa kiểm được trên stack: poll thật, `kill` gateway giữa chừng, đổi `UIDVALIDITY`, thư giả `From` — Mailpit không phải IMAP; chỉ có test Go. Cần máy chủ IMAP thử có sẵn ở stack US-S9-SEED nếu muốn kiểm đầu-cuối.
- Ghi nhận (dev đã tự ghi): thư `IGNORED` không có thao tác `Bỏ qua`, thư không bao giờ nộp lại giữ sinh viên `INCOMPLETE` (nợ nhỏ); thư viện mới `go-imap/v2@v2.0.0-beta.8`, `go-message@v0.18.2`, `go-sasl` ngoài bảng `ARCHITECTURE.md` — **cần PM / Tech Lead chấp nhận** (cùng dạng Q-06-d).
- Mật khẩu `security NONE` bị từ chối khi `APP_ENV=production` (hộp đen ở môi trường dev: 409 — chưa rõ môi trường test có đặt `APP_ENV=production`; không FAIL).

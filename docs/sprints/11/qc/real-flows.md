# Spec @real trên ep-qc (P-11-29) — QC

Chạy khi gate, trên stack thật `ep-qc` (Caddy HTTPS cùng origin, Mailpit, worker, 9router). Không gateway giả, không sửa spec của dev (các ca `@real` của dev vẫn `test.skip`; bản chạy được nằm ở đây).

```bash
cd frontend
QC_REAL=1 QC_BASE=https://localhost:<cổng HTTPS ep-qc> QC_PW=<SEED_DEFAULT_PASSWORD> QC_OUT=docs/sprints/11/qc/shots-real \
  QC_PG=ep-qc-postgres-1 QC_MAILPIT=http://localhost:<cổng Mailpit> \
  pnpm exec playwright test -c playwright.qc-real-flow.config.ts
```
Đồng hồ: không có đường tua giờ trong mã. «Tua giờ» = fixture DB (lùi `escalation_tickets.waiting_since`, đặt sự kiện `now()+23h`) rồi chờ tick thật của worker. Mặc định chờ ≤ 17 phút (`TICKET_SLA_TICK` 15 phút, `REMINDER_TICK` 5 phút). Muốn gate nhanh: worker ep-qc chạy `TICKET_SLA_TICK=1m REMINDER_TICK=30s` và `QC_TICK_WAIT_MS=180000` (cần TL).

| Tệp | Luồng | Phủ | Ca `@real` dev được thay (đang skip) |
| --- | --- | --- | --- |
| `qc-real-f1-account.spec.ts` | F1 | đăng ký → Mailpit → xác minh → đăng nhập; liên kết xác minh dùng lại; chưa xác minh không ACTIVE trong lớp; refresh qua Caddy (1 refresh, không token ở storage); reset thu hồi phiên máy B; mời GV → đặt mật khẩu → khoá → mở khoá; **mạo danh MSSV** (PENDING + cảnh báo lệch, không đọc dữ liệu SV thật, GV thấy hàng chờ); 10 lần sai → khoá + mail | `account.spec` 4 ca: register-verify-login, login+refresh Caddy, reset logs out other device, invite flow |
| `qc-real-f2-class.spec.ts` | F2, F14 | bộ chọn lớp theo seed; nhập roster-2-loi.csv (dòng 7, 19; Nhập 28) trên lớp mới; vào lớp 375×812, duyệt, tạo lại mã, 6 mã sai ⇒ 429, lớp lưu trữ tắt mã; Hôm nay SV/GV | `class-join.spec` 3 ca (picker seed; roster import — **phần «chia sẻ nguồn» chưa phủ**; join e2e), `today.spec` 1 ca |
| `qc-real-f3-f5-f13.spec.ts` | F3, F5, F13 | tin khủng hoảng ở `/chat` ×3 (có dấu / không dấu): có lời hỗ trợ, **0 dòng `llm_audit`**, không ticket; GV/Admin không đọc chat riêng; ticket quá 24 → 72 giờ làm việc (thông báo gộp + thư + «Hôm nay» kèm giờ chờ); nhắc 24 giờ (một chuông + một thư, không trùng, tắt loại `exam`); đổi hạn bài tập → Hôm nay + lịch dời | – |
| `qc-real-f9-f11.spec.ts` | F9, F11 | GV tạo + công bố; SV nộp PDF trong app; `grade-all` qua LLM thật; SV không thấy điểm nháp; duyệt → công bố (chưa duyệt bị chặn); SV thấy điểm + chuông; phúc khảo một lần → GV quyết định → SV được báo, 0 lời gọi LLM | – |
| `qc-real-f15-llm.spec.ts` | F15 | 401/403 API + trang chặn quyền `/settings/llm`; hai tab → 409; khoá sai (cần `FAKE_LLM_VALID_KEY` ở gateway + `QC_FAKE_VALID_KEY`) | `settings-llm.spec` 1 ca |
| (chờ dev) | F16, F17 | `qc-real-audit.spec.ts` theo `tc-US-P10-14.md`; F17 khi dev xong | – |

Ngoài phạm vi: F18 (PR). Ca `@real` của `exam.spec.ts` (đặt `E2E_REAL_API`) chạy bằng `scripts/check-exam-seed.mjs demo` ở đợt R.

Trạng thái: **đã viết, eslint + tsc sạch, CHƯA chạy** (chờ ep-qc dựng lại). Selector của màn duyệt / quyết định phúc khảo và vài đường dẫn API (`PUT assignments/{aid}`, `members/{id}/approve`, `PATCH admin/llm/providers/{id}`) lấy theo spec dev và openapi, có thể phải chỉnh ở lần chạy đầu — chỉnh trong tệp QC, không đổi kỳ vọng.

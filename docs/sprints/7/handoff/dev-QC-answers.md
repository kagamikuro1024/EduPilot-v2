# Dev trả lời QC — lệnh quản trị, chạy `ticket.sla`, xem thân yêu cầu LLM

Trạng thái: **các lệnh dưới đây chưa tồn tại** (thuộc US-P4-05 và US-S7-SEED, chưa thi công). Cú pháp là hợp đồng dev sẽ làm đúng; nếu khác sẽ cập nhật ở handoff của story đó. Nguồn: `FEAT-crm-360/SRS.md` 4.10, `FEAT-escalation-notify/SRS.md` 4.6, `proposals.md` #7.

## Cách gọi (chung)

Lệnh nằm trong binary `gateway`, chạy trong container, trên stack local:

```
docker compose --env-file .env.local -f docker-compose.local.yml -f docker-compose.test.yml -p edupilot \
  exec -T gateway /gateway admin <lệnh> <cờ>
```

Giống `gateway admin create` mà `scripts/seed.mjs` đang gọi. Ghi `-f docker-compose.test.yml` (build `gateway-test`, `APP_ENV=test`) hoặc bỏ nó (`APP_ENV=local`); cả hai được phép. `production`, `staging`, rỗng → thoát khác 0, không ghi gì. Mã thoát: 0 xong, 1 lỗi, 2 dùng sai.

## Ba lệnh seed (US-S7-SEED)

| Lệnh | Cú pháp | Kết quả |
| --- | --- | --- |
| `seed-study-time` | `admin seed-study-time --course <class_code> --weeks 10 --profile seed/study_profile.json` | Ghi `study_time_daily` trực tiếp, `ON CONFLICT … GREATEST`; chạy lại không nhân đôi. In `N dòng`. |
| `seed-age` | `admin seed-age --course <class_code> [--tickets] [--activity]` | Lùi `created_at`, `answered_at` **và `waiting_since`** của ticket (và hoạt động nếu `--activity`) để dựng ca "quá hạn". Không đổi cột nào khác. |
| `crm-risk-run` | `admin crm-risk-run --course <class_code>` | Chạy `crm.risk` một lần cho lớp (không chờ `RISK_JOB_AT`). |

`<class_code>` lấy từ `select class_code from courses`. File `seed/study_profile.json` do US-S7-SEED thêm.

## Chạy `ticket.sla` một lần (US-P4-05)

```
… exec -T gateway /gateway admin run-job ticket.sla [--now 2026-10-13T09:30:00+07:00]
```

- `--now` (RFC 3339) thay đồng hồ của **một lần chạy đó**; không có cờ → giờ thật. Không ghi gì ra ngoài lần chạy, không đổi đồng hồ của gateway / worker đang chạy.
- Cùng chốt môi trường (`local`/`test`), cùng `ticket.autoclose` (`run-job ticket.autoclose`).
- Chạy ở **gateway** (cùng khoá Redis `NX` như worker nên hai bên không chạy trùng một giai đoạn).
- Cách khác, không cần `--now`: `seed-age --tickets` lùi `created_at` / `waiting_since`, rồi `run-job ticket.sla`.

## Xem thân yêu cầu LLM của `ai-summary` (US-P5-05)

Dùng route thử của sprint 6 (`FEAT-private-chat-pii` SRS 4.3), cần stack `testroutes` và provider `fake`:

```
TOKEN=<access_token của ADMIN>
curl -s -H "Authorization: Bearer $TOKEN" http://localhost/api/v1/_test/llm/payloads \
  | jq '[.[] | select(.task=="ai_summary")] | .[-1] | {task, messages: [.messages[].content]}'
curl -s -X POST -H "Authorization: Bearer $TOKEN" http://localhost/api/v1/_test/llm/payloads/reset
```

Vòng đệm ≤ 200 lời gọi, nội dung **sau khi che** (tên / MSSV là placeholder). Image thường → 404. Gọi `reset` trước khi bấm "Tóm tắt" để chỉ còn một mục.

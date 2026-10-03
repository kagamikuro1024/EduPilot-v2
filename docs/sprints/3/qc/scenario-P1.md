# Kịch bản P1 — QC tự viết, chạy độc lập (US-P1-04 AC12, TC-P104-46)

Điều kiện: stack sạch (bảng `llm_*` trống, Redis `flushall`), 2 gateway :8080 / :8081 (`-tags testroutes`, `FAKE_LLM_VALID_KEY=good-key`, `LLM_DEFAULT_RPM=100000`), JWT ADMIN. Không chép từ handoff của dev. Mọi lệnh gọi qua `…/api/v1`; "chờ 1 s" = hạn nạp nóng của spec (đo thật ≈ 11–16 ms, xem TC-P104-27).

| # | Lệnh | Kết quả đo |
| --- | --- | --- |
| 1 | `GET /admin/llm/providers` | `200`, `items=[]`, `env_fallback={"active":true,"providers":["fake"]}` |
| 2 | `POST /_test/llm/chat` khi chưa cấu hình | `200` (chạy bằng nhà env dự phòng) |
| 3 | `POST /admin/llm/providers` "Chinh" (`fake`, khoá `good-key`, mô hình `fake-chat`) | `201`, `has_key=true`, `key_status=ok`, `last_test.ok=true`, thân **không** có `api_key` |
| 4 | `POST /admin/llm/providers` "Phu" (mô hình `fake-chat-2`) | `201` |
| 5 | `PUT /admin/llm/routes` `CHAT=[Chinh/fake-chat]` (version 0) | `200`, `ETag: W/"v1"` |
| 6 | `PUT /admin/llm/routes` `CHAT=[Phu/fake-chat-2, Chinh/fake-chat]` (version 1); chờ 1 s; chat ×4 xen kẽ :8080 / :8081 | `Phu/fake-chat-2` 4/4, `fallback_index=0` |
| 7 | `POST /admin/llm/providers/test` khoá `sai-khoa` | `200 {"ok":false,"error_kind":"AUTH","message":"Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá."}` |
| 8 | `POST /admin/llm/providers` "Sai" (khoá `sai-khoa`) | `422 VALIDATION_FAILED`, `details=[{"field":"api_key","code":"PROVIDER_AUTH_FAILED",…}]` |
| 9 | `GET /admin/llm/providers` | chỉ `Chinh`, `Phu`; chuỗi `sai-khoa` **không** xuất hiện trong thân |
| 10 | `PUT /admin/llm/providers/{Phu}` `enabled=false`; chờ 1 s; chat ×4 | `Chinh/fake-chat` 4/4 (nhà tắt bị bỏ khỏi chuỗi, mô hình kế tiếp thành chỉ số 0) |
| 11 | bật lại `Phu`; chờ 1 s; chat ×4 | `Phu/fake-chat-2` 4/4 |
| 12 | `POST providers` "Q-chinh" (`openai_compatible` → máy chủ giả QC, khoá thật `qk` kiểm trước khi lưu); `PUT routes CHAT=[Q-chinh, Chinh]`; chờ 1 s; chat ×4 | `Q-chinh/q1-chat#fb0` 4/4 |
| 13 | máy chủ giả trả `500`; chat ×4 | `Chinh/fake-chat#fb1` 4/4 (**fallback_index=1**), máy chủ giả nhận 8 yêu cầu (2 lần thử mỗi lời gọi); `llm_audit` ghi `ok/fb=1` |

Lưu ý (không phải lỗi, ghi để dev / BA biết): `fallback_index` đếm trong chuỗi **đang bật**. Tắt nhà chính (bước 10) ⇒ `0`, không phải `1`; muốn thấy `1` cần nhà chính **lỗi nhưng còn bật** (bước 13). Câu "tắt nhà chính → `fallback_index:1`" của TC-P104-46 vì vậy đọc là "nhà chính hỏng". Đọc ngay sau `PUT` ở **cùng** gateway có thể còn thấy cấu hình cũ trong khoảng ≈ 10 ms (một yêu cầu đầu); trong hạn ≤ 1 s của spec.

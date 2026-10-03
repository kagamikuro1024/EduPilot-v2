# DEV handoff — US-P1-05 (`/settings/llm` thật)
Nhánh `sprint/3-pu-p1`. Góp ý #29–#32. Chạy: `pnpm -C frontend build:gate && pnpm -C frontend exec playwright test settings-llm.spec.ts` (Next :3310, gateway giả :3312 theo hợp đồng thật).

## Làm gì
- `features/settings/llm/`: `LlmSettings.tsx` (5 khối: 4 phần đánh số `data-part=settings-section` + "Mức dùng và ngân sách" `data-part=usage-section`), `ProviderSection` + `ProviderForm` (thêm / sửa / đổi khoá, Test kết nối, bật / tắt, xoá), `RoutesSection` (`RouteTableSection`, `FallbackSection`), `EmbeddingSection`, `UsageSection`, `routeWriter` (ghi tuyến lạc quan + Hoàn tác + 409), `api.ts` (kiểu + truy vấn + `useSaver`), `labels.ts` (lời), `ConflictNotice`.
- `shared/domain/LLMRouteTable` (thành phần miền mới, đăng ký `/dev/ui` — #29).
- Dữ liệu thật qua `apiClient` + TanStack Query; `ApiErrorNotice` có `title` / `context`; xoá `features/settings/LlmSettings.tsx` và phần LLM của `mock/system.ts` (PROVIDERS, TASK_ROUTES, FALLBACK_CHAIN, EMBEDDING, BUDGET, ADVANCED).
- Ảnh mốc `settings-llm-1440|390.png` cập nhật: từ bản **mock** (PU-05) sang màn **thật** với dữ liệu giả cố định (`support/llm-fixtures.ts`) — lệch có chủ đích, có giải thích ở đây.

## AC tự đánh giá (`settings-llm.spec.ts`, 14 ca + 1 ca `@real` bỏ qua)
| AC | Kết quả thật |
| --- | --- |
| 1 | 4 `h2` hiện đúng chữ / thứ tự, 0 `settings-section` lồng nhau, khoảng cách giữa các phần lệch ≤ 1 px, 0 nút `primary` lúc đầu; phần 4 viền trên 2 px + "Đổi mục này cần lập chỉ mục lại tài liệu." |
| 2 | `Test kết nối` (secondary) đúng ở mỗi hàng (2/2), trạng thái "Đã kết nối · kiểm tra lúc 09:12" / "Lỗi xác thực"; lỗi `AUTH` hiện ngay dưới hàng; thành công "Kết nối tốt · 420 ms"; bấm đúp `Lưu` ⇒ đúng 1 POST (`useIdempotentMutation`) |
| 3 | ô khoá `type=password`, `autocomplete=new-password`, `spellcheck=false`, trống khi mở / sau khi lưu; khoá thử không có trong DOM, `localStorage`, `sessionStorage`, URL, cookie, console; "Để trống để giữ khoá hiện tại" |
| 4 | 422 `PROVIDER_AUTH_FAILED` ⇒ lỗi dưới ô khoá (`aria-invalid`) + "Chưa lưu gì."; số hàng giữ nguyên; tên đã gõ giữ nguyên; `Lưu mà không kiểm tra` (`skip_verify`) hiện sau lỗi mạng / không với tới máy chủ |
| 5 | đổi `CHAT` sang gpt-4o: ô chọn đổi ≤ 100 ms trong lúc máy chủ trễ 500 ms; `UndoLine` "Đã chuyển Trả lời chat riêng sang gpt-4o"; PUT mang `chain` đúng thứ tự + `version`; `Hoàn tác` ghi ngay; `Xuống` đổi thứ tự dự phòng bằng phím `Enter`; cảnh báo "Chưa có dự phòng — …" khi chuỗi chỉ có 1 mô hình; nhãn làn "Trả lời ngay" / "Chạy nền" |
| 6 | mọi lựa chọn nhúng `data-dims="1536"`; đổi ⇒ hộp xác nhận chứa "lập chỉ mục lại"; "Để sau" ⇒ 0 PUT; xác nhận ⇒ PUT `EMBEDDING` `version` đúng; 422 `MODEL_DIMS_MISMATCH` ⇒ đúng câu lỗi; sau đó "Cần lập chỉ mục lại" |
| 7 | `Nhiệt độ` không có mặt lúc đầu, mở (`aria-expanded`) có nhãn + đơn vị ("giây, 1–300"); `3` ⇒ `aria-invalid` và 0 PUT; đóng / mở giữ giá trị. Địa chỉ máy chủ chỉ hiện sẵn cho loại "Máy chủ riêng" |
| 8 | bảng 7 cột đúng tên; "1.240.000 đ" căn phải `tabular-nums`; dòng ngân sách "Hôm nay 42.000 đ / 80.000 đ · tháng này 1.240.000 đ / 2.000.000 đ" + dải 1 px; `warn` ⇒ "Đã dùng 80 % ngân sách ngày…"; 0 `svg circle` / `progressbar` ở khối ngân sách; rỗng ⇒ "Chưa có lượt gọi nào trong khoảng này." + 1 nút đổi khoảng |
| 9 | 503 `NOT_READY` ⇒ "Chưa tải được cấu hình. Cấu hình hiện có không bị ảnh hưởng." + `Thử lại` ⇒ đúng +1 request; danh sách rỗng / `LLM_NOT_CONFIGURED` ⇒ "Đang dùng cấu hình mặc định của máy chủ. Thêm nhà cung cấp để thay đổi." + hành động; mất mạng khi `Lưu` ⇒ `Gửi lại` dùng cùng `Idempotency-Key` (1 khoá duy nhất qua ≥ 4 lần gửi); 409 ⇒ "Cài đặt này vừa được người khác đổi. Giữ bản của bạn hay dùng bản mới?" và `Giữ bản của tôi` gửi lại với `version` mới (9) |
| 10 | SV / TA: màn chặn "Trang này dành cho giảng viên và quản trị viên." và **0** request `admin/llm`; GV: xem đủ, 0 nút `Test kết nối / Xoá / Lưu / Thêm / Đổi hạn mức`, 0 công tắc, mọi `select` `disabled`, "Chỉ quản trị viên được thay đổi cấu hình."; Admin đủ |
| 11 | `main.innerText` (Admin, GV) không khớp `RAG|PII|trace|fallback|embedding|prompt` (không tính tên mô hình của nhà cung cấp, vd. `text-embedding-3-small`), 0 `status` "Thành công"; `ui-antipatterns` sạch |
| 12 | 375 / 390: `TOUCH_SRC` = `[]`, `AUDIT` `{ox:0,cut:[],ell:[]}` (qua `runAudit` — #30); thêm nhà chỉ bằng phím (Enter, Tab, gõ, Enter); axe 0 critical / serious ở Admin và GV × 1440 / 390. **Bảng tác vụ trên điện thoại là danh sách xếp chồng** (mỗi tác vụ một khối) |
| 13 | `grep useDemoSlice\|ep_demo features/settings/llm` = 0; `LlmSettings.tsx` cũ đã xoá; `grep LLM_PROVIDERS\|llmSlice mock/system.ts` = 0; `/settings/integrations` không đổi. **`audit.mjs` đầy đủ: QC chạy** |
| 14 | `@real` (hai tab, `focus` ⇒ làm mới) chưa chạy; `refetchOnWindowFocus: true` mặc định + `VERSION_CONFLICT` xử lý như AC9 đã kiểm bằng máy chủ giả |
| 15 | kiểm bằng mắt của QC; dữ liệu thật cần stack Go — chưa chạy ở máy dev |

## Cổng frontend đã chạy
`eslint .` 0 · `tsc` sạch · `ui-antipatterns` 0 ✗ · `lint-selftest` 7/7, 19/19 · `ui-allow:` = 9 · `build:gate` + `env -u CI playwright test`: **155 passed, 85 skipped**, không retry; bản thường: `/dev/*` 404, 0 chuỗi cổng token.

## Nợ
- Ca `@real` (two tabs, khoá sai với `FAKE_LLM_VALID_KEY=good-key`, 403 bằng `curl`) cần stack Go.
- `settings.module.css` còn vài lớp chỉ mock LLM từng dùng (không ảnh hưởng); dọn khi sửa `/settings/integrations` ở P7.

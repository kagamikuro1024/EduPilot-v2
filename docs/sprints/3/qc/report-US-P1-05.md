# Báo cáo QC — US-P1-05 (`/settings/llm` thật)
**Kết luận: PASS có ghi chú** — 3 lệch nhỏ về chữ/nhãn (không chặn), không lỗi chức năng. 40/57 TC chấm bằng đo của QC (Chrome thật + gateway thật + Postgres/Redis riêng), 14 TC dựa bộ Playwright của dev (đã chạy lại: 68 pass), 3 TC chưa làm tay (TC-47 trọn luồng chỉ-bàn-phím, TC-55/56 kịch bản nghiệm thu mắt, ca `@real` chạy gộp ở đây).

- Bản chấm `fc7c920`, `build:gate` với `NEXT_PUBLIC_API_URL=http://localhost:8080` (gateway thật `testroutes`, `FAKE_LLM_VALID_KEY=good-key`, `CORS_ORIGINS=http://localhost:3400`), Chrome for Testing riêng, 2 gateway + worker. Q-QC-P105-1: QC dùng HTTP :8080 (không Caddy) — đủ cho CORS; Q-QC-P105-2/-3: chờ BA (QC coi `Lưu mà không kiểm tra` là chủ ý, ghi `audit_log.skip_verify=true`; cột "Chạy rút gọn" = `degraded` — số khớp `psql`).

## Lệch nhỏ (không chặn)
1. **AC4 thiếu câu "Máy chủ trong trường đang tắt?" ở nhánh `PROVIDER_UNREACHABLE`** (TC-16): thêm `openai_compatible` `base_url=http://localhost:9/v1` → lỗi "Không kết nối được tới nhà cung cấp. Kiểm tra địa chỉ và mạng." + hiện `Lưu mà không kiểm tra`, nhưng **không** có câu "máy chủ trong trường đang tắt?". Câu này có ở nhánh mất mạng trình duyệt (TC-37). Dev bổ sung câu ở nhánh 422 `PROVIDER_UNREACHABLE`.
2. **Hàng nhà chưa kiểm vẫn ghi "Khoá API •••••••• · đã kết nối"** (Q-local lưu bằng `skip_verify`: trạng thái "Chưa kiểm tra" nhưng dòng khoá nói "đã kết nối") — mâu thuẫn chữ.
3. Chân sidebar vẫn "Bản mô phỏng · dữ liệu giả" khi màn đang dùng dữ liệu thật (khung 1.5; đề nghị ẩn dải ở route `backend=thật`).

## Kết quả
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01, 02 | PASS | 4 `settings-section` (+ `usage-section`), `h2` đúng thứ tự, 0 lồng nhau, khoảng cách giữa phần **48, 48, 48 px**, `LEFT` 240; `primary` hiện ra lúc đầu: 0 (2 nút primary chỉ nằm trong hộp thoại đóng) |
| 04 | PASS | mỗi hàng: tên, loại, chữ + chấm "Đã kết nối · kiểm tra lúc 21:38", `provider-status`/`provider-action`, đúng 1 `Test kết nối` mỗi hàng |
| 06 | PASS | nhấp đúp `Lưu` khi thêm nhà → **1** `POST`, 1 bản ghi (`Idempotency-Key` có) |
| 08, 10, 11 | PASS | ô khoá `type=password`, `autocomplete=new-password`, `spellcheck=false`, trống khi mở; gõ dở khoá: **0** khoá `ep:draft:*`, chữ khoá không vào `localStorage`; 0 `title`/`aria-label` chứa khoá |
| 09 | PASS | canary `sk-LEAK-CANARY-5e9d21b7` (nhà `openai_compatible` + `skip_verify`): sau lưu **0** ở DOM, `localStorage`, `sessionStorage`, URL, cookie, console, log 2 gateway, `pg_dump` |
| 13 | PASS | `bad-key` → lỗi dưới ô khoá (`aria-invalid=true`, `aria-describedby` → "Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá."), "Chưa lưu gì. Các ô khác vẫn giữ nguyên." ; số hàng 2 → 2; tên giữ "Fake-C" |
| 14, 15 | PASS | đổi khoá Fake-A bằng `bad-key`: `md5(api_key_enc)` và `version` **y nguyên**, chat qua Fake-A vẫn thành công; `good-key` → khoá đổi, hàng "Đã kết nối", form đóng, không "Thành công" |
| 16 | PASS* | xem lệch 1: `Lưu mà không kiểm tra` **không** có ở trạng thái đầu, hiện sau lỗi; dùng → hàng "Chưa kiểm tra", `last_test_ok=null`, `audit_log.after.skip_verify` 1 dòng |
| 18 | PASS | 6 hàng tác vụ đúng tên ("Trả lời chat riêng", "Phân loại câu hỏi", "Việc nhỏ (tóm tắt, đặt tên)", "Chấm bài", "Sinh câu hỏi", "Tóm tắt lớp học"); nhãn làn "Trả lời ngay" / "Gần thời gian thực" / "Chạy nền" |
| 19 | PASS | đổi CHAT → `fake-chat-2`: DOM + dòng "Đã chuyển Trả lời chat riêng sang … · Hoàn tác" sau **14 ms**; đúng 1 `PUT` `200`; `Hoàn tác` ghi ngay (chat về `fake-chat`); tự biến sau 5,6 s; không "Thành công" |
| 20 | PASS | ngay sau đổi: `_test/llm/chat` ở **cả 2 gateway** trả `fake-chat-2` |
| 21 | PASS | thêm 2 dự phòng (Fake-C, Fake-D), `Xuống` bằng `Enter` đổi thứ tự (DB `fallback_order` cập nhật); mô hình đã có trong chuỗi **bị loại khỏi danh sách chọn** (không thể trùng). Giới hạn "quá 4" chưa thử |
| 22 | PASS | tắt Fake-A bằng công tắc: chat chuyển Fake-B (`fallback_index` đếm trong chuỗi đang bật = 0, như `scenario-P1.md`), hàng "Đã tắt"; bật lại → trở về Fake-A |
| 23 | PASS | `/dev/ui`: **25 khối / 125 ô áp dụng / 75 N/A** (theo #29), khối cuối `LLMRouteTable`; `dev-ui.spec.ts` pass |
| 24, 25 | PASS | **1** mô hình nhúng, "1536 chiều" cố định (0 ô số chiều), mọi lựa chọn `data-dims="1536"`; đổi → `role=dialog` "Đổi mô hình tìm kiếm tài liệu. Mọi tài liệu đã nạp sẽ phải lập chỉ mục lại; trong lúc đó tìm kiếm có thể kém chính xác."; `Để sau` → đóng, giữ lựa chọn cũ, 0 `PUT` |
| 28 | PASS | trạng thái đầu: không "Nhiệt độ", RPM; 6 nút "Cài đặt nâng cao" (mỗi tác vụ) và 1 ở form nhà |
| 29 | PASS | mở nâng cao: Nhiệt độ, Số token tối đa mỗi câu trả lời, Thời gian chờ, Số lần thử lại, gợi ý "giây, 1–300"; **nút `Lưu` trong khung nâng cao** kiểm hợp lệ: `3`, `-1` (nhiệt độ), `0`, `32769` (token), `0`, `301` (chờ), `6` (thử lại) → `aria-invalid=true`, **0 `PUT`**; `0`, `2`, `32768`, `300`, `5` → `PUT` `200` (5 lần), DB `params` lưu |
| 31, 32 | PASS | 7 cột (Tác vụ, Lượt gọi, Token vào / ra, Chi phí ước tính, Độ trễ p95, Lỗi, Chạy rút gọn), số căn phải `tabular-nums`, "220.110 đ"; gieo 40 dòng: từng tác vụ **khớp `psql`** (calls, token, cost, errors, degraded); 7 → 30 ngày; ngân sách một dòng "Hôm nay 0 đ / 80.000 đ · tháng này 0 đ / 2.000.000 đ" + dải **1 px**, 0 `svg circle`/`progressbar`; rỗng: "Chưa có lượt gọi nào trong khoảng này." + hành động "Xem 30 ngày". (Trạng thái `warn` 80 %: spec dev) |
| 34 | PASS | `503 NOT_READY` (chặn bằng CDP): "Chưa tải được cấu hình. Cấu hình hiện có không bị ảnh hưởng." + Admin thấy "Chi tiết kỹ thuật" + `Thử lại`; mỗi lần `Thử lại` = 1 lượt tải (3 request HTTP do `apiClient` tự thử lại 503: 300/900 ms) |
| 35 | PASS | danh sách rỗng / `env_fallback` → "Đang dùng cấu hình mặc định của máy chủ. Thêm nhà cung cấp để thay đổi." + `Thêm nhà cung cấp` |
| 36 | PASS (PU-04) | token hết hạn → "Phiên đã hết hạn"; build thường "Cần đăng nhập" |
| 37 | PASS | tắt mạng khi `Lưu`: `OfflineBanner` "Mất kết nối mạng", "Chữ bạn đã nhập vẫn được giữ", tên giữ nguyên, hiện `Gửi lại` và `Lưu mà không kiểm tra`; bật mạng → `Gửi lại`: **4 POST cùng một `Idempotency-Key`**, DB đúng **1** bản ghi |
| 39 | PASS | JWT STUDENT và TA dán ở cổng: "Bạn không có quyền xem màn này" + "Trang này dành cho giảng viên và quản trị viên." + `Về Hôm nay`; **0** request `/admin/llm` |
| 40 | PASS | JWT GV: thấy 4 phần + mức dùng (7/30 ngày), "Chỉ quản trị viên được thay đổi cấu hình."; nút hiện: chỉ "Cài đặt nâng cao" ×6, "7 ngày", "30 ngày", "Xem 30 ngày"; 0 `Test kết nối`/`Lưu`/`Xoá`; 6 `select` đều `disabled`; 0 công tắc, 0 ô nhập; hộp thoại xoá/đổi nhúng nằm ngoài màn (đóng) |
| 41 | PASS | GV ép bằng `curl`: `PUT routes`, `POST providers`, `POST providers/test`, `PUT budget` → **403** |
| 42 | PASS | Admin: Test, công tắc, menu `Sửa / Đổi khoá / Xoá`, đổi mô hình, đổi nhúng, thêm nhà |
| 44, 45 | PASS | `body.innerText` Admin và GV: 0 khớp `RAG|PII|trace|fallback|embedding|prompt` (không tính tên mô hình), 0 "Thành công"; `ui-antipatterns.sh` rc=0; `ui-allow:` = 9 |
| 46 | PASS | Admin 375 và 390: `AUDIT` `{ox:0, cut:[], ell:[]}`, `TOUCH` `[]`; bảng tác vụ = 6 khối `grid` rộng 358 px, 0 `table` ngang ở phần tác vụ (bảng mức dùng 1 `table`, 0 cuộn ngang) |
| 48 | PASS (spec dev) | `a11y.spec.ts` 0 critical / 0 serious (chạy lại, trong 68 pass) |
| 49 | PASS | `grep useDemoSlice\|ep_demo features/settings/llm` = 0; `features/settings` = `IntegrationsSettings.tsx`, `llm/`; `LLM_PROVIDERS\|llmSlice` ở `mock/system.ts` = 0 |
| 51 | PASS | không token → cổng dán token (không dữ liệu giả); có token → request thật `:8080/api/v1/admin/llm/*` |
| 52 | PASS | ảnh sau `shots/after/settings-llm-1440.png`, `-390.png` (màn thật; khác ảnh mốc cũ là bản mock — có giải thích ở handoff) |
| 53 | PASS | tab A đổi CHAT → tab B `focus` → thấy giá trị mới; sửa lệch `version` → "Cài đặt này vừa được người khác đổi. Giữ bản của bạn hay dùng bản mới?" + `Giữ bản của tôi` / `Dùng bản mới` (không mất chữ) |
| 57 | PASS | `git grep 'sk-…{10,}' frontend` = 0; build thường: không `input[type=password]` của cổng dán, 0 chuỗi cổng (PU-04) |
| 03, 05, 07, 12, 17, 27, 30, 33, 38, 43, 54 | PASS (spec dev) | `playwright settings-llm.spec.ts a11y.spec.ts dev-ui.spec.ts` (cổng riêng 3510/3512): **68 pass**, 22 skip (ca chỉ-desktop ở dự án mobile), 0 đỏ |
| 05 (hàng song song), 26, 47, 50, 55, 56 | **chưa chạy tay** | 26 (`MODEL_DIMS_MISMATCH` chặn bằng `page.route`), 47 (trọn luồng chỉ bàn phím), 50 (`audit.mjs` 4 vai + spec cho PASS ≥ nền: làm ở cổng PU), 55/56 (kịch bản mắt + 10 câu §22): ghi vào cổng |

## Việc sau
Dev: lệch 1–3 (dùng chữ). BA: Q-QC-P105-2/-3.

# QC test case — US-P1-05 (màn `/settings/llm` thật: 4 phần, khoá chỉ ghi, định tuyến, nhúng, mức dùng, phân quyền theo vai)
Nguồn: `docs/specs/FEAT-llm-gateway/US.md` US-P1-05 AC1–AC15 + `docs/specs/FEAT-llm-gateway/SRS.md` mục giao diện `/settings/llm`; `docs/design/DESIGN.md` §14.23, §13, §22; `FEAT-ui-foundation` (primitive, `apiClient`, cổng token dev). Hộp đen: **QC thao tác trên giao diện thật bằng chuột + bàn phím trong Chrome**, nối gateway thật (stack test, `fake` `FAKE_LLM_VALID_KEY=good-key`); test `settings-llm.spec.ts` của dev chạy thêm.

Tiền điều kiện chung: stack test chạy (2 gateway, Caddy `https://localhost`); bản `gbuild` chạy `next start -p 3300` với `NEXT_PUBLIC_API_URL=https://localhost` (cùng `-k` / chứng chỉ Caddy được Chrome QC tin: `--ignore-certificate-errors` cho Chrome riêng) và `NEXT_PUBLIC_DEV_AUTH=1`; token: `tok ADMIN`, `tok TEACHER`, `tok TA`, `tok STUDENT` (cổng dán token dev, US-PU-04 AC9). `CANARY="sk-LEAK-CANARY-7f3a9c1e"`. Công cụ: **P** = `$PW settings-llm.spec.ts -g …` (dev), **A** = Eval trình duyệt thật (QC viết `scripts/p105-*.mjs`, nghe `console`, `request`, đọc `localStorage`), **S** = shell/curl/psql, **T** = tay + ảnh. Thiếu `settings-llm.spec.ts` → TC loại **P** FAIL "KHÔNG KIỂM ĐƯỢC". Hai nhà `fake` cần có: `Fake-A`, `Fake-B`, mô hình `fake-chat`, `fake-chat-2`, `fake-embed` (1.536 chiều), 1 mô hình embedding 768 chiều để thử loại.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P105-01 | AC1 | Admin, 1440 | **A** mở `/settings/llm`; đếm `[data-part=settings-section]`, đọc `h2` theo thứ tự; đo khoảng cách dọc giữa các phần | Đúng 4 phần: (1) Kết nối nhà cung cấp, (2) Mô hình theo tác vụ, (3) Chuỗi dự phòng, (4) Mô hình tìm kiếm tài liệu (đúng chữ); khoảng cách đều, lệch ≤ **1 px**; phần 4 viền trên đậm hơn + câu "Đổi mục này cần lập chỉ mục lại tài liệu" |
| TC-P105-02 | AC1 | – | **A** `settings-section` lồng nhau? ; số `button.primary` hiện ở trạng thái đầu; `LEFT` của `[data-part=page-title]` | Không `settings-section` nằm trong `settings-section` (không card lồng card); `button.primary` = **0**; `LEFT` 240 (1440) / 16 (390) |
| TC-P105-03 | AC1 | – | **P** `$PW settings-llm.spec.ts -g 'sections'; echo rc=$?` | `rc=0` |
| TC-P105-04 | AC2 | 2 nhà trong DB (Fake-A tốt, Fake-B khoá sai/mạch mở) | **A** mỗi hàng nhà: `data-part=provider-status`/`provider-action`; đọc chữ trạng thái; đếm `Test kết nối` toàn trang | Mỗi hàng: tên, loại, trạng thái **chữ + chấm** ("Đã kết nối · kiểm tra lúc HH:MM", "Chưa kiểm tra", "Lỗi xác thực", "Tạm dừng do lỗi liên tiếp" khi `circuit=open`, "Đã tắt"), công tắc bật/tắt, `OverflowMenu` (Sửa, Đổi khoá, Xoá); `Test kết nối` có ở **mỗi** hàng và **không ở nơi khác**; nút `secondary` |
| TC-P105-05 | AC2 | – | **A** bấm `Test kết nối` ở Fake-A (độ trễ `fake` 2 s) rồi ngay bấm ở Fake-B; đọc nút, `InlineNotice` | Nút của hàng đó `loading` + "Đang kiểm tra…", **không chặn** hàng khác; kết quả hiện **ngay dưới đúng hàng** (`InlineNotice`); khoá sai → "Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá."; DB không đổi |
| TC-P105-06 | AC2 | – | **A** nhấp đúp nhanh `Lưu` khi thêm nhà; đếm request `POST` | Đúng **1** `POST` (và `Idempotency-Key` có); chỉ 1 bản ghi (`count(*)`) |
| TC-P105-07 | AC2 | – | **P** `-g 'provider row'` | `rc=0` |
| TC-P105-08 | AC3 (**khoá chỉ ghi**) | nhà đã có khoá | **A** mở "Đổi khoá": đọc ô khoá (`type`, `autocomplete`, `spellcheck`, `value`); nhãn | `type="password"`, `autocomplete="new-password"`, `spellcheck=false`, `value=""` khi mở; trạng thái chuỗi tĩnh "•••••••• · đã kết nối" (**không** đuôi khoá, kể cả của mock cũ `•••• 3f9a`); nhãn "Để trống để giữ khoá hiện tại" |
| TC-P105-09 | AC3 (**canary**) | – | **A** nhập `$CANARY`, `Lưu` (dùng `good-key` cho nhà `fake`; để thử canary dùng nhà `openai_compatible` + `skip_verify` hoặc máy chủ giả Q); sau lưu quét: `document.documentElement.outerHTML`, `JSON.stringify(localStorage)`, `sessionStorage`, `location.href`, `document.cookie`, mọi `console` (nghe `page.on('console')`), mọi response `GET` (qua `page.on('response')`), HAR | **Không** chứa `$CANARY` hay 4 ký tự cuối `7f3a9c1e`→`9c1e`; `input[type=password]` `value===""`; ô tự trống sau lưu |
| TC-P105-10 | AC3 (nháp) | – | **A** gõ dở khoá (chưa lưu) rồi đóng tab, mở lại; kiểm `localStorage` khoá `ep:draft:*` | **Không** có bản nháp khoá (`useAutosaveDraft` từ chối `apiKey`); các ô khác (tên) được nháp nếu AC của PU-03 áp dụng |
| TC-P105-11 | AC3 | – | **A** xem trang thủ công: DOM cây "Chi tiết kỹ thuật" / tooltip / `title` / `aria-label` có chứa đuôi khoá? | Không |
| TC-P105-12 | AC3 | – | **P** `-g 'key write-only'`; chạy `@real` | `rc=0` |
| TC-P105-13 | AC4 | Admin, `FAKE_LLM_VALID_KEY=good-key` | **A** "Thêm nhà cung cấp" `fake`, tên "Fake-C", nhập khoá `bad-key`, bấm `Lưu`; đo số hàng trước-sau; đọc lỗi | Lỗi hiện **dưới ô khoá** (`aria-invalid="true"`, `aria-describedby`): "Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá. Chưa lưu gì."; **không** có hàng "Fake-C" (số hàng không đổi); `llm_providers` không dòng mới; ô tên **giữ** "Fake-C" |
| TC-P105-14 | AC4 | – | **A** "Đổi khoá" của Fake-A bằng `bad-key`; rồi dùng Fake-A | Giữ trạng thái cũ ("Đã kết nối"); khoá cũ **còn dùng được** (chat qua Fake-A thành công) |
| TC-P105-15 | AC4 | – | **A** nhập `good-key` → `Lưu` | Lưu thành công; hàng hiện "Đã kết nối"; không toast "Thành công" |
| TC-P105-16 | AC4 (lối thoát) | máy chủ không với tới (cổng đóng) | **A** thêm nhà `openai_compatible` `base_url` không với tới; bấm `Lưu` | Sau lỗi mạng hiện `Lưu mà không kiểm tra` kèm câu "máy chủ trong trường đang tắt?"; dùng nó → lưu với `skip_verify`, trạng thái "Chưa kiểm tra"; **chỉ hiện sau lỗi**, không hiện ở trạng thái đầu |
| TC-P105-17 | AC4 | – | **P** `-g 'wrong key'`; `@real` | `rc=0` |
| TC-P105-18 | AC5 | Admin | **A** phần 2: đếm hàng, đọc tên tác vụ, nhãn làn, các lựa chọn của chọn mô hình | 6 hàng: "Trả lời chat riêng", "Phân loại câu hỏi", "Việc nhỏ (tóm tắt, đặt tên)", "Chấm bài", "Sinh câu hỏi", "Tóm tắt lớp học"; nhãn làn "Trả lời ngay" / "Gần thời gian thực" / "Chạy nền" đúng bảng làn; chọn mô hình gom theo nhà, chỉ mô hình `chat` của nhà **đang bật** (mô hình embedding / nhà tắt không có) |
| TC-P105-19 | AC5 | – | **A** đổi `CHAT` sang `fake-chat-2`; đo thời gian DOM đổi; đọc `UndoLine`; đọc request `PUT`; rồi `Hoàn tác` | DOM đổi ≤ **100 ms**; `[data-part=undo-line]` tại đúng hàng ("Đã chuyển Trả lời chat riêng sang …· Hoàn tác"), không toast; `PUT routes` có `chain` đúng thứ tự + `version`; `Hoàn tác` gửi `PUT` khôi phục (DB về cũ) |
| TC-P105-20 | AC5 (hiệu lực thật) | – | **S** ngay sau đổi: `POST /api/v1/_test/llm/chat` ở cả 2 gateway | `model:"fake-chat-2"` ≤ 1 s (tuyến mới có hiệu lực không restart) |
| TC-P105-21 | AC5 (chuỗi dự phòng) | – | **A** phần 3: thêm mô hình dự phòng, đổi thứ tự bằng **bàn phím** (nút `Lên`/`Xuống` + `Enter`), xoá khỏi chuỗi; thử thêm quá 4 / trùng | Thứ tự thay đổi đúng; `PUT` có `chain` mới; quá 4 hoặc trùng bị chặn tại chỗ (lời giải thích tiếng Việt), không gửi |
| TC-P105-22 | AC5 | – | **A** tắt nhà chính (công tắc) rồi chat qua `_test/llm/chat` | `fallback_index:1`; màn hiện hàng "Đã tắt" |
| TC-P105-23 | AC5 | `/dev/ui` | **A** `LLMRouteTable` có mặt ở `/dev/ui` với đủ trạng thái áp dụng; **P** `-g 'routing'`; `$PW dev-ui.spec.ts -g 'LLMRouteTable'` | Hiển thị đủ 8 trạng thái áp dụng (hoặc N/A có lý do); `rc=0` |
| TC-P105-24 | AC6 | Admin | **A** phần 4: số mô hình hiển thị; mọi lựa chọn `data-dims`; có ô sửa số chiều? ; có chuỗi dự phòng? | **Một** mô hình nhúng; "1536 chiều" cố định (không ô sửa); mọi lựa chọn `data-dims="1536"` (mô hình 768 chiều **không có** trong danh sách); không chuỗi dự phòng |
| TC-P105-25 | AC6 | – | **A** đổi mô hình nhúng → mở dialog; bấm "Để sau"; mở lại → "Đổi mô hình" | `role=dialog` chứa "lập chỉ mục lại" và hậu quả đúng chữ: "Đổi mô hình tìm kiếm tài liệu. Mọi tài liệu đã nạp sẽ phải lập chỉ mục lại; trong lúc đó tìm kiếm có thể kém chính xác."; "Để sau" **không** gửi `PUT` (đếm request = 0); xác nhận gửi đúng **1** `PUT`; sau đó hiện "Cần lập chỉ mục lại" (từ `reindex_required`), **không** hứa tự chạy |
| TC-P105-26 | AC6 | – | **A** giả 422 `MODEL_DIMS_MISMATCH` (QC chặn bằng `page.route`) | Đúng câu lỗi tiếng Việt; chọn giữ nguyên giá trị cũ |
| TC-P105-27 | AC6 | – | **P** `-g 'embedding'` | `rc=0` |
| TC-P105-28 | AC7 | Admin | **A** trạng thái đầu: tìm `getByLabel('Nhiệt độ')`, RPM, TPM, `max_tokens`, thời gian chờ, số lần thử, `price_in/out`, `base_url` (nhà `openai_compatible` vs `openai`) | Không có mặt (trừ `base_url` của `openai_compatible`, luôn hiện); nút `Cài đặt nâng cao` là nút văn bản, `aria-expanded="false"` |
| TC-P105-29 | AC7 | – | **A** mở nâng cao: đọc nhãn / đơn vị; nhập nhiệt độ `3`, `-1`, `0`, `2`; `max_tokens` `0`, `32769`, `32768`; thời gian chờ `0`, `301`; số lần thử `6`; giá `-1`; đếm `PUT`; đóng rồi mở lại | Nhãn tiếng Việt, đơn vị ("đ / 1 triệu token", "giây"); ngoài khoảng → `aria-invalid` tại ô và **không** gửi `PUT`; biên (`0`, `2`, `32768`, `5`) chấp nhận; đóng/mở **không mất** giá trị đã nhập |
| TC-P105-30 | AC7 | – | **P** `-g 'advanced'` | `rc=0` |
| TC-P105-31 | AC8 | gieo `llm_audit` (≥ 30 dòng 7 ngày, số biết trước) + ngân sách `daily=80.000`, `monthly=2.000.000` | **A** bảng mức dùng: đọc 7 cột, định dạng tiền, căn lề, `font-variant-numeric`; đổi khoảng 7 / 30 ngày; đối chiếu với `psql` của QC | Cột đúng: `Tác vụ`, `Lượt gọi`, `Token vào / ra`, `Chi phí ước tính` (định dạng `1.240.000 đ`), `Độ trễ p95`, `Lỗi`, `Chạy rút gọn`; `td` số `tabular-nums` + `text-align: right`; **số khớp** SQL của QC (chính xác từng đồng); đổi khoảng đổi dữ liệu |
| TC-P105-32 | AC8 | – | **A** dữ liệu rỗng; ngân sách dòng chữ; dải 1 px; `state=warn` (đẩy tới 85 %) | Rỗng: "Chưa có lượt gọi nào trong khoảng này." + **một** hành động (đổi khoảng); ngân sách là **một dòng chữ** ("Hôm nay 42.000 đ / 80.000 đ · tháng này 1.240.000 đ / 2.000.000 đ") + dải mảnh 1 px theo `pct`; **không** vòng tiến độ / `svg circle` / `role=progressbar` dạng vòng / biểu đồ tròn; `warn` → `InlineNotice` chứa "80 %" |
| TC-P105-33 | AC8 | – | **P** `-g 'usage\|budget'` | `rc=0` |
| TC-P105-34 | AC9 | Admin | **A** chặn bằng `page.route`: 503 `NOT_READY` khi tải trang; bấm `Thử lại`; đếm request | `PageState` lỗi chuẩn "Chưa tải được cấu hình. Cấu hình hiện có không bị ảnh hưởng. Thử lại."; `Thử lại` chỉ gọi lại **đúng 1** truy vấn (URL trang không đổi, không tải lại cả trang) |
| TC-P105-35 | AC9 | DB không nhà bật | **A** mở màn khi `LLM_NOT_CONFIGURED` / danh sách rỗng | `InlineNotice` "Đang dùng cấu hình mặc định của máy chủ. Thêm nhà cung cấp để thay đổi." + hành động `Thêm nhà cung cấp` |
| TC-P105-36 | AC9 | token hết hạn | **A** token `exp` quá khứ / 401 `TOKEN_EXPIRED` giữa lúc dùng | Build dev: cổng dán token; build thường: "Cần đăng nhập"; không mất chữ đang nhập |
| TC-P105-37 | AC9 | – | **A** mất mạng (`setOffline`) khi bấm `Lưu` rồi bật lại, bấm `Gửi lại`; so header `Idempotency-Key` hai lần; đếm hàng DB | Chữ giữ nguyên; `OfflineBanner` hiện; `Gửi lại` dùng **cùng** `Idempotency-Key`; DB chỉ **1** bản ghi |
| TC-P105-38 | AC9 | – | **P** `-g 'errors'` | `rc=0` |
| TC-P105-39 | AC10 | token `tok STUDENT`, rồi `tok TA` | **A** mở `/settings/llm`; đếm request tới `/admin/llm` | Màn chặn: "Trang này dành cho giảng viên và quản trị viên." + `Về Hôm nay`; **0** request tới `/api/v1/admin/llm/*` |
| TC-P105-40 | AC10 | `tok TEACHER` | **A** mở `/settings/llm`; liệt kê nút / công tắc / ô nhập / menu | Xem được cả 4 phần + mức dùng + ngân sách **chỉ đọc**: không `Test kết nối`, không `Lưu`, không `Xoá`; công tắc `aria-disabled`; ô nhập `readonly`/vắng; dòng "Chỉ quản trị viên được thay đổi cấu hình."; `getByRole('button',{name:/Test kết nối\|Xoá\|Lưu/}).count()===0` |
| TC-P105-41 | AC10 (chặn thật) | – | **S** TEACHER ép bằng `curl`: `PUT /admin/llm/routes`, `POST providers`, `…/test` | **403** (chặn ở gateway lần nữa) |
| TC-P105-42 | AC10 | `tok ADMIN` | **A** liệt kê thao tác | Đầy đủ: Test, công tắc, Sửa/Đổi khoá/Xoá, đổi mô hình, đổi nhúng |
| TC-P105-43 | AC10 | – | **P** `-g 'roles'` | `rc=0` |
| TC-P105-44 | AC11 | Admin, Teacher | **A** `document.body.innerText` (mở cả "Cài đặt nâng cao" và hộp thoại) khớp `/\b(RAG\|PII\|trace\|fallback\|embedding\|prompt)\b/i`; đọc nút | 0 khớp (dùng "chuỗi dự phòng", "mô hình tìm kiếm tài liệu"); mọi nút là động từ; lỗi nêu vấn đề + dữ liệu an toàn + cách khắc phục; không `role=status` chứa "Thành công" |
| TC-P105-45 | AC11 | – | **S** `bash scripts/ui-antipatterns.sh`; `P` `-g 'copy'` | Sạch; `rc=0` |
| TC-P105-46 | AC12 | Admin 375, 390 | **A** `AUDIT_SRC`, `TOUCH_SRC`; bảng tác vụ → danh sách xếp chồng (mỗi tác vụ một khối: tên, mô hình chính, dự phòng) | `{ ox:0, cut:[], ell:[] }`, `[]`; không cuộn ngang; khối xếp chồng, không `table` ngang |
| TC-P105-47 | AC12 | – | **A** chỉ **bàn phím**: thêm nhà → Test → đổi mô hình → đổi thứ tự dự phòng → xác nhận đổi nhúng → mở nâng cao; ghi focus mỗi bước | Hoàn thành mọi thao tác chỉ bằng phím; vòng focus nhìn thấy; hộp thoại bẫy focus, `Esc` thoát, trả focus |
| TC-P105-48 | AC12 | – | **A** axe-core (wcag2a/aa/21aa/22aa) ở Admin và Teacher × 1440, 390 | 0 `critical`, 0 `serious`; `$PW a11y.spec.ts -g 'settings/llm'` → 0 |
| TC-P105-49 | AC13 | repo | **S** `grep -rn 'useDemoSlice\|ep_demo' frontend/src/features/settings/Llm* \| wc -l`; `ls frontend/src/features/settings`; `grep -n 'LLM_PROVIDERS\|llmSlice' frontend/src/mock/system.ts \| wc -l` | `0`; không còn `LlmSettings.tsx` cũ (hoặc nội dung mới, không `mock/system`); `0` — không để hai bản |
| TC-P105-50 | AC13 | build gate | **A** `audit.mjs` bốn vai + spec như `audit-baseline.md`; `sweep.mjs only:'student'`; `proto-curl.sh all`; `/settings/integrations` so ảnh | FAIL 0, PASS ≥ nền (lưu ý: số hàng của `/settings/llm` có thể đổi — ghi chênh và lý do); `FORBIDDEN`=0; ≥ 497 PASS; `/settings/integrations` **không đổi** |
| TC-P105-51 | AC13 | – | **A** `/settings/llm` ở build gate với cookie demo (không token) vs token thật; tab Network | Màn dùng `apiClient` (request thật `/api/v1/admin/llm/*`), không đọc cookie demo; không token → cổng dán token / "Cần đăng nhập" (không hiện dữ liệu mock) |
| TC-P105-52 | AC13 | – | **A** ảnh mốc `/settings/llm` cập nhật có giải thích trong handoff; so `shots/before/settings-llm-*.png` với ảnh sau | Giải thích có trong handoff; QC ghi khác biệt trước/sau (lưu vào `shots/after/`) |
| TC-P105-53 | AC14 | 2 tab | **A** tab A đổi tuyến `CHAT`; tab B `focus` lại; rồi tab B sửa dòng `CHAT` với `version` cũ | Tab B thấy giá trị mới sau `focus` (`refetchOnWindowFocus`); tab B sửa dòng cũ → `409 VERSION_CONFLICT` hiện dòng hỏi giữ bản nào (không ghi đè im lặng, không mất chữ) |
| TC-P105-54 | AC14 | – | **P** `@real -g 'two tabs'` | `rc=0` |
| TC-P105-55 | AC15 | stack sạch | **T** làm theo kịch bản: thêm nhà, khoá sai → lỗi, đúng → lưu; `Test kết nối`; chuyển `CHAT` sang nhà thứ hai; tắt nhà chính → chat vẫn trả lời qua dự phòng; tắt hết → "AI tạm thời không khả dụng"; chụp ảnh từng bước | Mọi bước đúng; ảnh vào `report-US-P1-05.md`; câu suy giảm không từ kỹ thuật |
| TC-P105-56 | AC15 | – | **T** 10 câu nghiệm thu cuối `DESIGN.md` §22 cho `/settings/llm`; đếm đỏ; card lồng card; so `shots/before` / `edupilot-ui-v3.html` | Mọi câu "có"; 0 card lồng card; 0 nút đỏ đặc thừa; đỏ chỉ mang 3 nghĩa |
| TC-P105-57 | tổng (an toàn) | – | **S** `git grep -nE 'sk-[A-Za-z0-9]{10,}' frontend`; quét bundle `frontend/.next/static` tìm `CANARY`/khoá; `pbuild` không chứa mã cổng dán token | Không secret trong repo/bundle; `pbuild`: không `input[type=password]` của cổng dán |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| Khoá lộ ra DOM / storage / console / URL / HAR | 08–11 |
| Lưu khoá sai làm mất nhà / khoá cũ | 13, 14 |
| Nhấp đúp tạo hai bản ghi | 06, 37 |
| Mất mạng giữa lúc lưu; chữ mất | 37 |
| Hết hạn token giữa chừng | 36 |
| TEACHER ghi được; SV / TA gọi API | 40, 41, 39 |
| Hai người sửa cùng lúc | 53 |
| Số chiều nhúng sai chọn được | 24, 26 |
| Số ngoài khoảng gửi lên | 29 |
| Còn mã mock / hai bản | 49, 51 |
| Từ kỹ thuật lọt ra | 44 |

## Câu hỏi cho BA / PM
- **Q-QC-P105-1** — Màn cần gateway **thật** qua HTTPS Caddy chứng chỉ tự ký: trình duyệt QC dùng `--ignore-certificate-errors` (chỉ Chrome riêng của QC, profile tạm). CORS: `CORS_ORIGINS` phải gồm `http://localhost:3300`. Đúng cấu hình sẽ dùng? — *chờ xác nhận*.
  - **Trả lời (BA, 2026-10-03):** Đúng cấu hình. Đã ghi vào "Quy ước kiểm chung" (v1.2): `CORS_ORIGINS` của stack test gồm `http://localhost:3300`; Chrome riêng của QC (profile tạm) với `--ignore-certificate-errors`.
- **Q-QC-P105-2** — AC4 `Lưu mà không kiểm tra`: lối thoát này cho phép Admin lưu khoá không kiểm — QC coi là chủ ý (US) nhưng ghi nhận như một đường lưu khoá chưa xác minh. — *chờ trả lời nếu cần chặn*.
  - **Trả lời (BA, 2026-10-03):** Đúng, chủ ý (Q10 của spec): `skip_verify` chỉ ADMIN, ghi `audit_log`, `last_test.ok=null` ("Chưa kiểm tra"); giao diện chỉ gợi ý sau lỗi "không với tới". Không chặn thêm; QC ghi nhận như đường lưu khoá chưa xác minh có chủ ý, không FAIL.
- **Q-QC-P105-3** — AC8: số "Chạy rút gọn" = `degraded`. QC xác nhận cột này là đếm `degraded=true`. — *chờ xác nhận*.
  - **Trả lời (BA, 2026-10-03):** Xác nhận: "Chạy rút gọn" = số dòng `llm_audit` có `degraded=true` trong khoảng (SRS 6.3 Usage, v1.2).

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md v1.1 (FEAT-llm-gateway, APPROVED 2026-10-03).

Tổng: 57 TC.

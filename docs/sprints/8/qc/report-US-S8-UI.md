# QC report — US-S8-UI (bỏ câu mô tả thừa ở `/settings/llm`, `/settings/integrations`, khối bảo mật `/settings`) · Kết luận vòng 3 (2b): **PASS** (AC1, AC2, AC3 PASS; ảnh mốc `settings-llm` xanh ở CI Linux; CI của run vẫn đỏ vì bốn ca KHÔNG thuộc story — xem "Bổ sung CI"). Vòng 2: FAIL (AC1: BUG-2, 3 chưa sửa). Vòng 1: FAIL (AC1, AC2)

## Vòng 3 (2026-10-31, `sprint/8-p6` @ `f4921c86` — handoff mục "Vòng 2b", #10 ACCEPTED)
Dựng lại `pnpm build:gate`; dump `docs/sprints/8/qc/shots/ui-settings-*.txt` (1440 px) và `ui-settings-*-390.txt/png` (390 px, lần đầu có).
| Lỗi | Kết quả | Bằng chứng |
| --- | --- | --- |
| BUG-S8-UI-2 cảnh báo dài lặp 3 dòng | **ĐÃ SỬA** | không còn `InlineNotice` câu dài; ba tác vụ chỉ có một mô hình hiện nhãn ngắn `Chưa có dự phòng` (`StatusText`, 1–3 từ) đúng chỗ #10 chấp nhận; không còn chữ "nếu nhà cung cấp lỗi" |
| BUG-S8-UI-3 trạng thái dài / dòng phụ trùng / `Khoá API ••••` | **ĐÃ SỬA** | trạng thái `Đã kết nối` / `Lỗi xác thực`; không dòng `Khoá API`; không dòng phụ trùng tên; dòng rỗng `Mặc định máy chủ` |
| BUG-S8-UI-1, 4 (vòng 2) | giữ nguyên đã sửa | không câu dưới tiêu đề "Mô hình tìm kiếm tài liệu"; integrations `Đã kết nối` / `Chưa cấu hình` / `Chờ duyệt`, "Kiểm gần nhất" một lần |
Kiểm: `playwright test settings-llm shell lcp a11y panels --project=desktop --workers=2 --retries=0`: **112 passed**, 2 skipped, 0 failed; `ui-antipatterns.sh` rc=0; `pnpm lint` sạch; icon Info/CircleHelp = 0; tooltip chỉ ở hai nút chỉ-có-icon. Test cũ bị sửa: `e2e/settings-llm.spec.ts` (`:41`, khối dự phòng) — **có proposal #10 ACCEPTED**, chỉ đổi đúng chuỗi và đếm theo số dòng một mô hình, không đổi gì khác (`git diff 4ff5bb76..f4921c86 -- frontend/e2e`).
390 px (TC-06, 10): `/settings/llm` và `/settings/integrations` một cột, không câu mô tả, không tràn ngang; ghi nhận (thấp): ô chọn "Tóm tắt lớp học" hiện nhãn tuỳ chọn `Chưa chọn — dùng cấu hình mặc định của máy chủ` bị cắt ở 390 px (nhãn của giá trị chọn, không phải câu mô tả) — dev xét đổi `Mặc định máy chủ` cho đồng bộ.
Ảnh mốc `visual.spec.ts` (TC-18): QC không dựng lại trong image Playwright (cần cài `pnpm install` Linux riêng); dev báo 14 passed trong image; `git diff` chỉ đổi `settings-llm-390.png`. **CI ở `f4921c86`:** `Frontend e2e (Playwright 2/3)` và `Frontend Lighthouse (devtools B)` đỏ, `Playwright 3/3` và `lint + build` xanh, các job Go bỏ qua (commit chỉ đổi frontend); nhật ký chi tiết chưa đọc được khi run còn chạy → QC sẽ ghi bổ sung. Không đổi kết luận AC (đo trực tiếp trên màn), nhưng story chưa "CI xanh". Nghi `Lighthouse devtools B` và `bell real` (xem `report-US-P6-03.md` BUG-P6-03-4) là cùng nguyên nhân hạ tầng/ca mong manh, không thuộc US-S8-UI.

| AC | Vòng 3 |
| --- | --- |
| AC1 `/settings/llm` | **PASS** |
| AC2 `/settings/integrations` | **PASS** |
| AC3 bảo mật `/settings` | **PASS** |

---

## Vòng 2 (2026-10-31, `sprint/8-p6` @ `d4efecdd`; commit dev `74a778c7`; handoff mục "Vòng 2"; proposals #10, #11 ACCEPTED)
Dựng lại `pnpm build:gate`, dump lại ba màn (`shots/ui-settings-*.txt/png` ghi đè). Lưu ý môi trường: server `next start` / `api-server.mjs` của lượt trước còn chạy với bản dựng cũ → QC kill rồi mới đo (nếu không, trang báo "Trang này gặp sự cố").
| Lỗi vòng 1 | Vòng 2 | Bằng chứng |
| --- | --- | --- |
| BUG-S8-UI-1 câu thường trực "Đổi mục này cần lập chỉ mục lại tài liệu." | **ĐÃ SỬA** | dump `/settings/llm`: khối "Mô hình tìm kiếm tài liệu" không còn câu dưới tiêu đề; hệ quả vẫn ở hộp thoại xác nhận đổi mô hình (đúng luật) |
| BUG-S8-UI-4 trạng thái dài / lặp ở `/settings/integrations` | **ĐÃ SỬA** | trạng thái `Đã kết nối` / `Chưa cấu hình` / `Chờ duyệt`; không còn dòng meta dưới `h1`; "Kiểm gần nhất" chỉ còn **một** lần, thành `DT Kiểm gần nhất` trong danh sách trường của "Thư đi (SMTP)" |
| BUG-S8-UI-2 cảnh báo `Chưa có dự phòng — …` lặp 3 dòng | **CHƯA SỬA** (chờ dev, #10 đã ACCEPTED) | `RoutesSection.tsx:149` còn nguyên `InlineNotice` |
| BUG-S8-UI-3 trạng thái / dòng phụ dài, lặp | **CHƯA SỬA** (như trên) | `labels.ts:44` còn `Đã kết nối · kiểm tra lúc …`; `ProviderSection.tsx:81` còn `Khoá API •••••••• · đã kết nối`; dòng phụ trùng tên `OpenAI` / `Gemini`; dòng rỗng "Tóm tắt lớp học" còn câu dài |

Chạy lại: `playwright test settings-llm shell lcp a11y panels --project=desktop --workers=2 --retries=0`: **112 passed**, 2 skipped, 0 failed; `ui-antipatterns` và `lint` không đổi so vòng 1 (dev báo xanh; không có commit đụng thêm). `git diff 0db5ff70..HEAD -- 'frontend/e2e/*'`: không sửa test cũ nào ở vòng này (#10 chưa thực thi: dev nói rõ chờ proposal).
| AC | Vòng 2 |
| --- | --- |
| AC1 `/settings/llm` | **FAIL** (BUG-S8-UI-2, 3 mở) |
| AC2 `/settings/integrations` | **PASS** |
| AC3 bảo mật `/settings` | PASS (không đổi) |

**Đề nghị:** dev thực thi #10 (sửa code + hai assertion `settings-llm.spec.ts:41`, `:136–138` cùng commit, ghi `#10`) rồi báo handoff vòng 3; QC chấm lại AC1 (dump + `settings-llm.spec.ts`) và chốt story. Vòng 1 giữ nguyên bên dưới.

**Bổ sung CI (run `41186476014`, `f4921c86`, hoàn tất — QC đọc nhật ký):** `visual.spec.ts` 14/14 `✓` kể cả `settings-llm @1440` và `@390` (TC-18 PASS bằng ảnh mốc Linux của CI); không có ca `settings-llm.spec.ts` nào đỏ. Đỏ gồm bốn ca ngoài phạm vi story: `class-join.spec.ts:361` "bell real" (desktop + mobile, cả retry), `activity.spec.ts:261` "heartbeat cadence even" (desktop, cả retry), `dev-ui.spec.ts:234` "cursor" (một lần) và `Lighthouse (devtools B)` LCP. Cả bốn đều dựa vào đồng hồ giả / thời gian; run này là commit chỉ đổi frontend (job Go bỏ qua) nên không do thay đổi Go của US-P6-03. QC tái hiện cục bộ: "bell real" đỏ 4/4; "heartbeat cadence" xanh; chưa rõ nguyên nhân chung. Đã báo trong `report-US-P6-03.md` BUG-P6-03-4 (nâng thành việc riêng cho PM/dev).

**Cập nhật CI (run `41188894701`, `f07c2af6`, xong):** `Frontend e2e (Playwright 1/3, 2/3, 3/3)` **xanh**, `Frontend Lighthouse (devtools B)` **xanh** (#14: `lighthouse-api.mjs` thêm route giả `/admin/llm/*`, không đổi ngưỡng — LCP `/settings/llm` về dưới 2500 ms), `lint + build` xanh. Còn đỏ: `Lighthouse (simulate)` `total-blocking-time` trung vị 217,5 ms > 200 (5 lần đo 177,5–239) — sát ngưỡng, không phải trang của story này theo QC (chưa xác định URL); dev chạy lại. Ca "bell real", "heartbeat cadence", "cursor" đã xanh nhờ sửa gốc test (BUG-S8-01, #17). Story **PASS**.

---
## Vòng 1
Ngày 2026-10-31. Commit kiểm: `sprint/8-p6` @ `12fa90bf` (gồm `365b8050`, `6b539a5e`). Bộ TC: `tc-US-S8-UI.md` (không sửa chữ TC). Handoff: `dev-US-S8-UI.md`. Bản dựng `pnpm build:gate`; chạy ở 1440 × 900, kiểm bằng dump `innerText` + ảnh (`docs/sprints/8/qc/shots/ui-settings-{llm,integrations,admin,teacher,student}.{txt,png}`; spec QC `scripts/qc-s8-ui.spec.ts.txt`, đã gỡ khỏi `frontend/e2e/` sau khi chạy).

Ghi chú trung thực: TC này viết ở pha 1 hộp đen từ US + luật chữ UI, nhưng QC đã đọc `dev-US-S8-UI.md` trước khi viết (lệnh PM giao việc đọc handoff trong cùng lượt). Ba phát hiện lỗi dưới đây đến từ **màn thực chạy**, không từ handoff.

## 1. Cổng nghiệm thu đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `bash scripts/ui-antipatterns.sh; echo $?` | PASS rc=0 (21 mục `✓`) |
| `pnpm -C frontend lint` | PASS (`eslint .` rc=0) |
| `pnpm build` rồi `pnpm build:gate` | PASS |
| `playwright test settings-llm.spec.ts shell.spec.ts lcp.spec.ts a11y.spec.ts account.spec.ts panels.spec.ts --workers=2 --retries=0` | PASS: **162 passed, 130 skipped** (ca chỉ chạy một dự án), 0 failed, 0 flaky, 3,8 phút |
| `playwright test visual.spec.ts` (ảnh mốc `settings-llm-1440/390`) | KHÔNG KIỂM ĐƯỢC: ảnh mốc sinh trong image `mcr.microsoft.com/playwright:v1.63.0-noble`, QC không kéo image ở lượt này; chỉ kiểm được rằng `git diff` chỉ đổi hai ảnh `settings-llm-*.png` |
| `git diff 4c86d357.. --stat -- 'frontend/e2e/*' '*_test.go'` | test cũ bị sửa: **chỉ** `frontend/e2e/lcp.spec.ts` (+2 dòng: `import { mockLlmApi }` và một dòng gọi cho ca `/settings/llm`) — đúng proposal #1 (PM ACCEPTED); không đổi ngưỡng / assertion |
| `grep -rln` 10 chuỗi dev ghi "đã xoá" ở `docs/specs`, `frontend/src`, `docs/*.md` | 0 tệp / chuỗi (không còn ở SRS) |

## 2. Kết quả TC
| TC | Kết quả | Bằng chứng |
| --- | --- | --- |
| 01 `/settings/llm`: dưới `h1` và dưới tiêu đề 4 khối không có câu mô tả | **FAIL** | khối "Mô hình tìm kiếm tài liệu" còn một câu thường trực ngay dưới tiêu đề: `P.sub  Đổi mục này cần lập chỉ mục lại tài liệu.` (`EmbeddingSection.tsx:40`). Ba khối còn lại và `h1` đúng |
| 02 mỗi dòng nhãn + điều khiển + giá trị; trạng thái 1–2 từ; không cảnh báo lặp mỗi dòng | **FAIL** | (a) "Chuỗi dự phòng": cùng một câu cảnh báo `Chưa có dự phòng — nếu nhà cung cấp lỗi, người dùng sẽ thấy câu trả lời rút gọn.` (`InlineNotice warning`, `RoutesSection.tsx:149`) lặp **3 lần** (Phân loại câu hỏi, Việc nhỏ, Sinh câu hỏi) — đúng điều luật cấm ("lặp một câu cảnh báo trên mọi dòng"); (b) trạng thái dài hơn 2 từ: `Đã kết nối · kiểm tra lúc 09:12` (nhà cung cấp OpenAI); (c) dòng "Tóm tắt lớp học" có câu dài `Chưa chọn mô hình chính — đang dùng cấu hình mặc định của máy chủ.` và tên hiển thị `Chưa chọn — đang dùng cấu hình mặc định của máy chủ` ở khối tìm kiếm; (d) lặp thừa: dưới tên "OpenAI" lại có dòng phụ `OpenAI`, `Gemini` dưới `Gemini`; mỗi dòng nhà cung cấp lặp `Khoá API •••••••• · đã kết nối` cùng lúc trạng thái đã ghi "Đã kết nối" |
| 03 icon Info / CircleHelp | PASS | 0 (`svg[data-lucide=info|circle-help]`, `.lucide-info`, `[data-icon=info]`) ở cả ba màn |
| 04 tooltip | PASS | `title=` chỉ có ở hai nút `Thêm hành động cho OpenAI` / `Gemini` (tên nút chỉ có icon `…`, đúng luật "nút chỉ icon thì tooltip là tên nút"); `aria-describedby` chỉ trỏ tới hộp thoại và ô ghi chú; không tooltip nào chứa câu mô tả đã xoá |
| 05 dòng "Đổi mục này cần lập chỉ mục lại tài liệu." | **FAIL** | hiện **thường trực** dưới tiêu đề khối (không điều kiện); luật cho phép "hệ quả của một lựa chọn" ở tooltip trên chính nhãn/nút, không phải câu cố định dưới tiêu đề. Đã có sẵn hai chỗ đúng luật cho cùng thông tin: hộp thoại xác nhận khi đổi mô hình và `InlineNotice` điều kiện "Cần lập chỉ mục lại" |
| 06 390 px | KHÔNG KIỂM ĐƯỢC ở lượt này | spec `settings-llm.spec.ts` chỉ chạy desktop; không có dump 390 px (ảnh mốc `settings-llm-390.png` thuộc TC-18). QC chưa dump tay |
| 07 `/settings/integrations`: không câu mô tả dưới `h1` / mỗi tích hợp | PASS (mô tả) | không `<p>` mô tả; mỗi tích hợp là danh sách `Máy chủ`, `Địa chỉ gửi`… (nhãn + giá trị) + một nút |
| 08 ví dụ cú pháp ở `placeholder` | KHÔNG ÁP DỤNG | màn không có ô nhập (hiển thị chỉ đọc + nút) |
| 09 icon, tooltip | PASS | 0 icon; 0 tooltip |
| 10 390 px | KHÔNG KIỂM ĐƯỢC | như TC-06 |
| 11 `/settings` khối bảo mật | PASS | ba vai (STUDENT, TEACHER, ADMIN): `h1` "Tài khoản và bảo mật" → khối "Mật khẩu" (ba nhãn + một helper `Ít nhất 10 ký tự, không phải mật khẩu phổ biến.` = cú pháp nhập, được phép; có thể chuyển sang `placeholder`/tooltip nhưng không vi phạm) → "Thiết bị đang đăng nhập" (trong môi trường giả hiện khối lỗi "Không tìm thấy nội dung này." + `Thử lại` vì API giả không có `/me/sessions` — không phải lỗi sản phẩm); không câu mô tả dưới tiêu đề trang / khối |
| 12 không sửa lan | PASS | `git diff --stat` chỉ chạm `IntegrationsSettings.tsx`, `LlmSettings.tsx`, `SecuritySettings.tsx`, `mock/system.ts` (bỏ trường `purpose`), `PreShell.tsx`, `intro.ts` + hai ảnh mốc + `lcp.spec.ts` |
| 13 `ui-antipatterns` | PASS | rc=0 |
| 14 `settings-llm.spec.ts shell.spec.ts` | PASS | nằm trong 162 passed |
| 15 test cũ bị sửa | PASS | chỉ `lcp.spec.ts` với số proposal #1 ACCEPTED |
| 16 chuỗi đã xoá còn trong SRS | PASS | 0 / 10 |
| 17 lint, build | PASS | |
| 18 ảnh mốc | KHÔNG KIỂM ĐƯỢC | cần image Playwright của dev (xem mục 1) |
| 19 a11y | PASS | `a11y.spec.ts` trong lượt chạy; `aria-describedby` không trỏ phần tử đã xoá |
| 20 LCP | PASS | `lcp.spec.ts` trong lượt chạy |
| 21 trạng thái lỗi / rỗng còn | PASS | `/settings` hiện khối lỗi + `Thử lại` (môi trường giả); LLM có `ApiErrorNotice` ở mã |
| 22 `InlineNotice` "Chỉ quản trị viên được thay đổi cấu hình." | PASS | cảnh báo trạng thái có điều kiện (chỉ khi không phải ADMIN), không phải câu mô tả |
| 23 phân quyền màn | PASS | `shell.spec.ts` ca "route access" / "SV bị chặn /settings/llm" nằm trong lượt chạy |

## 3. AC
| AC | Kết quả | Lý do |
| --- | --- | --- |
| AC1 `/settings/llm` | **FAIL** | còn câu mô tả thường trực dưới tiêu đề khối "Mô hình tìm kiếm tài liệu"; cảnh báo giống nhau lặp ở 3 dòng "Chuỗi dự phòng"; trạng thái dài hơn 2 từ |
| AC2 `/settings/integrations` | **FAIL** | trạng thái ba tích hợp dài: `Đã kết nối · kiểm gần nhất 08:55 hôm nay`, `Chưa cấu hình`, `Chờ quản trị viên của trường đồng ý` (US: "trạng thái 1–2 từ"); thông tin "kiểm gần nhất 08:55 hôm nay" xuất hiện **hai lần** (dòng meta dưới `h1` và trạng thái SMTP) |
| AC3 bảo mật `/settings` | PASS | không câu mô tả; `ui-antipatterns` rc=0; `settings-llm.spec.ts` + `shell.spec.ts` xanh; chuỗi đã xoá không còn ở SRS |

## 4. Lỗi
- **BUG-S8-UI-1 (Trung bình):** `/settings/llm`, khối "Mô hình tìm kiếm tài liệu": câu `Đổi mục này cần lập chỉ mục lại tài liệu.` thường trực dưới tiêu đề. Tái hiện: ADMIN mở `/settings/llm`, đọc dưới `h2` thứ tư. Kỳ vọng: không câu dưới tiêu đề khối; hệ quả của lựa chọn ở tooltip trên nhãn `Đổi mô hình tìm kiếm tài liệu` hoặc chỉ ở hộp thoại / `InlineNotice` điều kiện đã có. File: `frontend/src/features/settings/llm/EmbeddingSection.tsx:40`. Handoff của dev ghi "giữ nguyên … hệ quả của việc đổi" — QC không chấp nhận vì luật nói hệ quả đặt **trên chính nhãn/nút** (tooltip), không phải câu cố định.
- **BUG-S8-UI-2 (Trung bình):** `/settings/llm`, "Chuỗi dự phòng": cùng một `InlineNotice warning` ("Chưa có dự phòng — nếu nhà cung cấp lỗi, người dùng sẽ thấy câu trả lời rút gọn.") lặp trên mỗi tác vụ chỉ có một mô hình (3 lần ở seed giả). File: `RoutesSection.tsx:149`. Kỳ vọng (luật cứng): không lặp một câu cảnh báo trên mọi dòng — ví dụ trạng thái 1–2 từ ("Chưa có dự phòng") trên từng dòng, giải thích (nếu thật cần) ở một tooltip trên nhãn cột; hoặc một cảnh báo chung.
- **BUG-S8-UI-3 (Thấp):** `/settings/llm` còn trạng thái / dòng phụ dài và lặp: `Đã kết nối · kiểm tra lúc 09:12`; `Chưa chọn mô hình chính — đang dùng cấu hình mặc định của máy chủ.`; dòng phụ trùng tên (`OpenAI` dưới `OpenAI`); `Khoá API •••••••• · đã kết nối` lặp ở mọi nhà cung cấp trong khi trạng thái đã nói "Đã kết nối". Tái hiện: dump `docs/sprints/8/qc/shots/ui-settings-llm.txt`.
- **BUG-S8-UI-4 (Trung bình):** `/settings/integrations`: trạng thái dài và lặp thông tin (xem AC2). Kỳ vọng: `Đã kết nối` / `Chưa cấu hình` / `Chờ duyệt` (1–2 từ); "kiểm gần nhất 08:55" chỉ xuất hiện một lần (hoặc làm dòng trong danh sách trường: nhãn `Kiểm gần nhất`, giá trị `08:55 hôm nay`). Files: `frontend/src/mock/system.ts` (`stateText`), `IntegrationsSettings.tsx` (`meta`).

Không phải lỗi: `InlineNotice` "Chỉ quản trị viên được thay đổi cấu hình." (cảnh báo trạng thái có điều kiện); các chuỗi trong `Dialog` (mô tả hộp thoại bảo vệ việc cần xác nhận, nằm ngoài luật "dưới tiêu đề khối"); helper cú pháp mật khẩu.

## 5. Kiểm phản mẫu UI / luật mở rộng / phân quyền
- `ui-antipatterns.sh` sạch; không `fetch` trần, không thư viện mới (`git diff … -- frontend/package.json` rỗng); không màu/ cỡ chữ cứng.
- Không toast "Thành công"; không icon Info; tooltip chỉ ở nút chỉ-có-icon.
- Phân quyền: `shell.spec.ts` (route access, SV bị chặn `/settings/llm`) xanh.

## Đề nghị
1. Dev sửa BUG-S8-UI-1, 2, 4 (và 3 nếu tiện) rồi gửi lại: xoá hẳn câu dưới tiêu đề khối; thu trạng thái về 1–2 từ; bỏ lặp. Ảnh mốc `settings-llm-*` / `settings-integrations` sinh lại trong image Playwright; không sửa test cũ nếu không có proposal.
2. Nếu PM coi dòng cảnh báo `Chưa có dự phòng …` và `Đổi mục này cần lập chỉ mục lại…` là hợp lệ theo SRS `FEAT-llm-gateway`, ghi quyết định vào `proposals.md` (đổi chữ luật cần vá SRS + Confluence cùng đợt — luật cứng PO) và QC chấm lại AC1.
3. QC chạy lại TC-06, 10 (390 px) và TC-18 (ảnh mốc trong image Playwright) ở lượt chấm lại.

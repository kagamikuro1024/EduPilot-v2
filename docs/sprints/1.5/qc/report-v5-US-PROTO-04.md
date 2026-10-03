# QC report v5 — US-PROTO-04 (Hiểu lớp + hệ thống)  · Kết luận: **FAIL**

Nhánh `sprint/1.5-mock-ui` · HEAD `8bb6861` · ngày 2026-10-01 · server `http://localhost:3400`
TC: `docs/sprints/1.5/qc/tc-US-PROTO-04.md` — **50 TC** (TC-04-01…TC-04-50; đếm lại bằng `grep -c '^| TC-04-'` = 50, không phải 51 như phiếu giao việc ghi).
Công cụ đã dùng: **C** `evidence-v5/proto-curl.log`, **A** `evidence-v5/audit.json` + `audit-fails.json`, **B** Chrome for Testing headless riêng (profile `/tmp/qcv5-v5q04`, 1440 / 1100 / 1024 / 720 / 719 / 600 / 500 / 430 / 390 / 375 px), ảnh `docs/sprints/1.5/shots/qc-v5/v5q04/`.

**Tóm tắt:** 46/50 TC PASS; 4 FAIL — **TC-04-09, TC-04-19, TC-04-24, TC-04-33**.

## Cổng đã chạy

| Lệnh / script | Kết quả |
| --- | --- |
| `proto-curl.log` · `tc_04_01` (GV 5 route + Admin 6 route) | PASS — 11 dòng `MO` |
| `proto-curl.log` · `tc_04_02`, `tc_04_03`, `tc_04_04` | PASS — `/insights` không tên/MSSV; GV `/observability` không `[[SV_`; GV `/settings/llm` không `Test kết nối` |
| `proto-curl.log` · `tc_04_06` (11 dòng) + `tc_04_07` | PASS — SV/TA chặn 5 route, GV chặn `/admin/courses`, TA `/analytics` không "chi phí"; `provider-status` ≥ 3 |
| `proto-curl.log` · `tc_04_08`, `tc_04_10`, `tc_00_16` | PASS — `/inbox` 3 hàng "Quá 24 giờ"; "AI tự trả lời 98%"; "hôm qua 16:40" ở `/` và `/admin/courses`; `/` không "08:30"; `derive.ts` còn `ticketStats`, hết số cứng `overdue/answeredByAi/escalated` |
| `proto-curl.log` · `tc_00_matrix` (hàng `/observability`, `/settings/*`, `/admin/*`) | PASS — GV `MO` `/observability` + `/settings/*`, `CHAN` `/admin/*`; Admin `MO` 6 route của mình |
| `audit.json` — "04-AC7", "04-AC8", "04-AC11", "04-AC12", `AUDIT`/`LEFT`/`HEADER` 8 route × vai × 1440/390 | PASS (đã đo lại bằng tay, xem TC-04-26…28, 35, 36, 45, 46) |
| `audit-fails.json` — `admin /admin/users 720 AUDIT-biên` | FAIL — đã xác nhận bằng tay + ảnh (BUG-v5-04-4) |
| Quét tay 1440 → 375 px bước 20 px trên `/settings/llm` (54 bề rộng) | PASS — 0 bề rộng có tràn ngang / `…` / `title` / hai cột chồng nhau |
| Quét tay `innerWidth` ở 375 và 390 px trên 11 cặp (vai, route) | FAIL ở `/observability` (GV và Admin): `innerWidth` nở 400 / 401 px → tràn ngang 10 / 26 px |

## TC

| TC-id | PASS/FAIL | Cách kiểm | Bằng chứng / ghi chú |
| --- | --- | --- | --- |
| TC-04-01 | PASS | C | `proto-curl tc_04_01` — 11 dòng `MO` (GV 5, Admin 6) |
| TC-04-02 | PASS | B | Sweep 11 route ở 1440: mỗi route đúng **1** `h1` ("Lỗ hổng kiến thức", "Số liệu lớp học", "Quan sát AI", "Cấu hình LLM", "Tích hợp", "Hôm nay", "Lớp học", "Người dùng"); khung nhìn đầu đúng §14.21–14.25. Ảnh `053-sweep-*` |
| TC-04-03 | PASS | B | `/insights` lớp 1: "Báo cáo Thứ Năm, 22 tháng 10, 16:40"; 1 = "Mật mã đối xứng (AES, CBC)", 2 = "Hàm băm và chữ ký số"; mỗi chủ đề có lý do + **5** tín hiệu + "Xem 5 / 4 / 3 câu hỏi đã ẩn danh"; chủ đề 5 "Bắt tay TLS 1.3 — Dưới 3 sinh viên hỏi — không hiện câu mẫu"; mục "Tài liệu chưa đề cập" (3 mục). Ảnh `032` |
| TC-04-04 | PASS | C + B | `tc_04_02` PASS; mở hết 4 khối câu mẫu rồi quét `innerText`: `2022\d{4}` = 0 ca, 0/4 tên seed. Ảnh `033` |
| TC-04-05 | PASS | B | `Tạo báo cáo mới` → danh sách 3 bước ("Đang chạy / Chờ / Chờ"), **không** spinner giữa trang (chỉ vòng nhỏ trong nút); mốc đổi giữa 2,84 s và 3,14 s → "Báo cáo Thứ Năm, 29 tháng 10, 09:20 · 80 câu". Ảnh `034`, `035` |
| TC-04-06 | PASS | B | `Tạo thread ghim` → `/threads` 12 → 13, thread "Chủ đề đang vướng: Mật mã đối xứng (AES, CBC)" thấy ở **cả** vai GV và vai SV B (sv-2); `Tạo buổi ôn tập` → `/calendar` có "Ôn tập: Mật mã đối xứng (AES, CBC) · Thứ Ba, 3 tháng 11 · 19:30–20:30". Ảnh `037`, `039`, `040` |
| TC-04-07 | PASS | B | Lớp 2 (`int1006-2`): chủ đề 1 = "Tường lửa và phân đoạn mạng" (12 sinh viên · 17 câu). Ảnh `041` |
| TC-04-08 | PASS | B | Drawer bắt nhập lý do trước; sau khi nhập: "Đã ghi nhật ký kiểm toán · 09:20 · Đỗ Hoàng Nam · lý do: …"; nội dung "Chấm lại tiêu chí 2 của bài nộp **[[SV_2]]**…", 0 tên thật / 0 MSSV; có "Công cụ đã gọi", "Độ tin cậy 0,62", nút giả "Chi tiết yêu cầu" (`href` rỗng). Ảnh `012`, `013` |
| TC-04-09 | **FAIL** | B | Lý do rỗng và lý do 4 khoảng trắng: **không mở** (nút "Mở nội dung" `disabled = true`) ✔, nhưng **không có báo lỗi tại ô**: `aria-invalid` = `null`, không phần tử `[role=alert]`, chữ trong drawer không đổi sau khi blur hoặc sau khi gõ rồi xoá. → BUG-v5-04-1. Ảnh `010-obs-lydo-rong-nut-vohieu` |
| TC-04-10 | PASS | C + B | `tc_04_03` PASS; GV: dải trạng thái + khối "Lớp 761987 trong 7 ngày" (5 số tổng hợp); hàng `cursor: auto`, click không mở Drawer (0 drawer), `[[SV_` = false. Ảnh `014` |
| TC-04-11 | PASS | B | 1.284 yêu cầu hôm nay · p95 3,1 s · lỗi 0,6% · model dự phòng 1,2% · 42.000 đ; `main tbody tr` = **50**. Ảnh `008` |
| TC-04-12 | PASS | B | "Kết nối được · 412 ms" hiện tại hàng OpenAI; đổi model CHAT → dòng "Đã đổi model của "Trả lời chat" sang … · Hoàn tác" (0 dialog mở, `dialog.open = false`), bấm Hoàn tác trả về giá trị cũ; khoá hiện "••••3f9a · đã kết nối", `outerHTML` không khớp `sk-[A-Za-z0-9]{8,}`, ô "Đổi khoá" là `input[type=password]` `value = ""`. Ảnh `003`, `004`, `006` |
| TC-04-13 | PASS | B | Đổi embedding → "Phải đánh chỉ mục lại toàn bộ tài liệu — 12.480 đoạn sẽ được tính lại bằng text-embedding-3-large (khoảng 25 phút)…". Ảnh `005` |
| TC-04-14 | PASS | C + B | `tc_04_04` PASS; GV `/settings/llm`: **0** `button`, **0** `select` trong `main`; có dải giải thích "Chỉ quản trị viên hệ thống đổi được cấu hình này". Ảnh `054` |
| TC-04-15 | PASS | B | Test Gemini → "Gemini từ chối khoá API (HTTP 401). Ba yêu cầu gần nhất đều hỏng, các tác vụ đang chạy bằng OpenAI theo chuỗi dự phòng." + cách khắc phục ("Tạo khoá mới trong Google AI Studio, dán vào ô Khoá API … kiểm tra hạn mức thanh toán"). Mã HTTP nằm trong câu giải thích, không phải mã lỗi trần. Ảnh `003` |
| TC-04-16 | PASS | B | `/insights?state=empty`: "Chưa có báo cáo lỗ hổng kiến thức cho lớp 761988" + đúng một nút `Tạo báo cáo mới`. Ảnh `042` |
| TC-04-17 | PASS | C | `tc_04_06` — 10 dòng `CHAN` cho SV/TA, `teacher /admin/courses CHAN`, "TA ở /analytics không có 'chi phí'" |
| TC-04-18 | PASS | C | `tc_00_matrix`: GV `MO` `/observability`, `/settings/llm`, `/settings/integrations`; GV `CHAN` `/admin/*`; Admin `MO` 6 route |
| TC-04-19 | **FAIL** | B | GV có mục "Chi phí", TA không (`tc_04_06`) ✔; biểu đồ tự vẽ (1 `svg`, 0 biến thư viện chart/d3/recharts) ✔; đổi 7 ↔ 30 ngày đổi hầu hết số liệu **nhưng khối "Bảo vệ thông tin cá nhân" không đổi**: 30 ngày ghi "**31 lần** … trong 30 ngày" trong khi ba dòng vẫn "Chat riêng 6 · Bài nộp qua email 2 · Thread công khai 1" (= 9, là số của 7 ngày). Lớp 2 cũng vậy: 30 ngày "5 lần" nhưng "2 + 1 = 3". → BUG-v5-04-2. Ảnh `063` |
| TC-04-20 | PASS | B | Admin `/`: đúng 4 việc (Gemini lỗi 3 lần/15 phút; ngân sách sắp chạm 80%; 2 việc hàng chờ xử lý lỗi; lớp 761988 chưa có giảng viên hoạt động); bấm từng việc → `/settings/llm`, `/settings/llm`, `/observability`, `/admin/courses`; `?state=empty` → "Hệ thống đang vận hành bình thường". Ảnh `019-admin-home`, `020` |
| TC-04-21 | PASS | B | `Mở lớp` mở form **tại chỗ** (0 `[role=dialog]` hiện), chọn giảng viên trong `select` 3 lựa chọn → "Đã mở lớp 761989 · Đã gửi thông báo phân công tới ThS. Nguyễn Minh Khôi · Hoàn tác"; `…` → `Lưu trữ lớp` → Dialog "Lưu trữ lớp 761987?" (Để sau / Lưu trữ lớp) → trạng thái đổi "Đã lưu trữ". Ảnh `021`, `022`, `023`, `024`, `025` |
| TC-04-22 | PASS | B | 60 tài khoản = 57 SV + 1 GV + 1 TA + 1 Admin; lọc vai: Giảng viên 1, Trợ giảng 1, Quản trị 1, Sinh viên 57, Tất cả 60; `Mời giảng viên` → "Đã gửi link mời tới gv.moi@edupilot.test, hạn 72 giờ · Hoàn tác"; Khoá → "Đã khoá tài khoản TS. Lê Thu Hà · Hoàn tác", Hoàn tác trả về "Đang dùng"; Mở khoá → "Đã mở khoá … · Hoàn tác"; không có nút tạo sinh viên, có dải "Không tạo tài khoản sinh viên ở đây…". Ảnh `026`, `027` |
| TC-04-23 | PASS | B | Mail "Đã kết nối · kiểm gần nhất 08:55 hôm nay", `Gửi thư thử` → "Đã gửi thư thử tới nam.dh@edupilot.test lúc 09:20…" tại chỗ; IMAP "Chưa cấu hình", `Kiểm tra` → "Chưa cấu hình nên chưa kiểm được…"; Teams "Chờ quản trị viên của trường đồng ý" + "Người duyệt: Quản trị viên Microsoft 365 của trường"; GV: **0** nút. Ảnh `030`, `031` |
| TC-04-24 | **FAIL** | B | 8 route × 3 trạng thái. `loading`: 8/8 có skeleton (1–3 khối, đúng hình khối thật). `error`: 8/8 có `Thử lại`. `empty`: **`/settings/llm` và `/settings/integrations` rơi vào bản mẫu chung "Chưa có gì ở đây · Khi có dữ liệu, nó sẽ xuất hiện tại đây"** (không dạy hành động, 0 nút); `/analytics` ("Lớp 761988 chưa đủ dữ liệu…"), `/observability` ("Chưa có yêu cầu nào trong hôm nay"), Admin `/` ("Hệ thống đang vận hành bình thường") có câu riêng nhưng **0 nút hành động**. Chỉ `/insights`, `/admin/courses`, `/admin/users` có hành động. → BUG-v5-04-3. Ảnh `057`, `058`, `059`, `020` |
| TC-04-25 | PASS | B | GV và TA ở `/observability`, `/analytics`, `/inbox` (có bấm thử hàng đầu): không chuỗi "Nội dung đã che", "Công cụ đã gọi", "Mở nội dung", "Lý do xem nội dung", "Token vào / ra", `[[SV_`; TA còn bị chặn hẳn `/observability`. Chỉ Admin sau khi nhập lý do mới thấy (TC-04-08) |
| TC-04-26 | PASS | A + B | 1440, Admin: `['provider-status','provider-action'].map(… Set(left).size)` = **[1, 1]**; left = 811 (×3) và 1035 (×3), đủ 3 hàng OpenAI / Gemini / Máy chủ trong trường. Ảnh `001` |
| TC-04-27 | PASS | A + B | Rộng cột: `provider-status` = 200 px ×3, `provider-action` = 140 px ×3 (đúng `minmax(0,1fr) 200px 140px`); dòng Gemini "Kiểm gần nhất: 09:05 hôm nay · 3 lần lỗi trong 15 phút" xuống dòng trong cột, `cut: []`, `ell: []`, hàng không lệch. Ảnh `001` |
| TC-04-28 | PASS | A + B | 390 và 375: thứ tự `top` = thông tin 402 → trạng thái 481 → nút 537 (cả 3 hàng); nút rộng 308 px = đúng rộng hàng 308 px (375: 293 = 293), cao **44** px; "Tháng 10 · 1.240.000 đ / 2.000.000 đ" `scrollWidth = clientWidth = 248` (xuống dòng, không `…`); `ell: []`, `cut: []`. Ảnh `007` |
| TC-04-29 | PASS | B | Quét 1440 → 375 bước 20 px (54 bề rộng) + đo riêng 1100 / 1024 / 720 / 719 / 600: `ox = 0`, 0 phần tử `…`, 0 `title`, `cut = []`, `ell = []`, không bề rộng nào làm `provider-status` và `provider-action` chồng nhau |
| TC-04-30 | PASS | B | GV 1440: `provider-status` left = 811, 811, 811 (thẳng hàng), không nút `Test kết nối`, không select; cột nút trống nhưng không để lại khoảng trắng kéo dài bất thường (đo card: trạng thái 811–1011, mép card 1175). GV 390: left = 41 ×3, `ox = 0`. Ảnh `054`, `055` |
| TC-04-31 | PASS | B | Test cả 3 hàng (kể cả Gemini lỗi) + đổi model CHAT + dòng Hoàn tác: `[1, 1]` sau mỗi bước; ở 390 "Kết nối được · 412 ms" và khối lỗi Gemini xuống dòng, `ox = 0`, `cut = []`, `ell = []`. Ảnh `003`, `004`, `060` |
| TC-04-32 | PASS | B | 390: "••••3f9a · đã kết nối" hiện đủ; "Kiểm gần nhất: 09:05 hôm nay · 3 lần lỗi trong 15 phút" `scrollWidth = clientWidth = 308` (xuống dòng, không cắt, không `…`). Ảnh `060` |
| TC-04-33 | **FAIL** | B | GV và Admin, 390 và 375 px: `innerWidth` nở thành **400 / 401** px (nội dung rộng hơn khung nhìn → tràn ngang 10 px ở 390, 26 px ở 375); tiêu đề "50 yêu cầu gần nhất" / "36 yêu cầu gần nhất" bị bóp còn **45 px**, xuống dòng từng chữ ("50 / yêu / cầu / gần / nhất"); thủ phạm là cụm lọc `segmented` rộng 323 px nằm cùng hàng flex với tiêu đề. Phần bảng/danh sách thì đọc được, `cut = []`, GV đúng là chỉ dải + số tổng hợp. → BUG-v5-04-5. **Nghi lỗi công cụ**: `audit.json` chấm `ox = scrollWidth − innerWidth` nên ra 0 vì chính `innerWidth` đã nở; đo tay thắng. Ảnh `017`, `018` |
| TC-04-34 | PASS | A + B | `/admin/courses` và `/admin/users` ở 390 và 719: `visTables = 0` (dạng danh sách), `ox = 0`, chip "Đang học" / "Mới mở" / "Giảng viên" / "Trợ giảng" / "Quản trị viên" đủ chữ (`scrollWidth = clientWidth`), nút `…` "Hành động cho lớp 761987" đo được **44 × 44** px và mở menu `Lưu trữ lớp`. Ảnh `028` |
| TC-04-35 | PASS | A + C + B | `/analytics` "Câu chờ quá 24 giờ: **3 câu**" = **3** hàng nhãn "Quá 24 giờ" ở `/inbox` (`tc_04_08`); chuyển 30 ngày vẫn "3 câu" |
| TC-04-36 | PASS | A + B | 7 ngày: "AI tự trả lời **98%** trong **392** câu hỏi; **6** câu chuyển sang giảng viên…"; "Câu đã chuyển giảng viên: 6 câu trong 7 ngày"; 30 ngày: "**99%** trong **1.424** câu hỏi; **19** câu" |
| TC-04-37 | PASS | B | SV B gửi D3 ở `/chat` → "AI chưa đủ chắc chắn… Đang chờ giảng viên · vừa gửi"; GV `/analytics` 7 ngày: escalated 6 → **7**, tỉ lệ vẫn 98% (round((392−7)/392×100) = 98); `Nhận` phiếu "Đã chờ 1 ngày" (Lê Quang Huy) → `/inbox` còn **2** nhãn "Quá 24 giờ" và `/analytics` "Câu chờ quá 24 giờ: **2 câu**". Ảnh `049`, `051`, `052` |
| TC-04-38 | PASS | B | Lớp 2, 7 ngày: "98% trong **118** câu hỏi; **2** câu chuyển"; 30 ngày: "98% trong **197** câu; **4** câu"; "Câu chờ quá 24 giờ: **Không có**" (hiện bằng chữ thay vì "0" — cùng nghĩa, chấp nhận); biểu đồ lớp 2 cộng lại = 12+9+19+23+16+21+18 = **118**, không lẫn số lớp 1. Ảnh `047` |
| TC-04-39 | PASS | B | Sau D3, 30 ngày: escalated = **20**, "AI tự trả lời **99%** trong 1.424 câu" = round((1424−20)/1424×100) = 99; chuyển 7 ↔ 30 ngày không đổi "Câu chờ quá 24 giờ" (3 → 3) |
| TC-04-40 | PASS | B | `Tạo thread ghim` chủ đề 1 → nút thành "**Đã ghim · Xem thread**" `href = /threads/ith-int1006-1-aes-cbc`; `/threads` 12 → **13**, đúng một thread ghim cho chủ đề đó. Ảnh `036`, `037` |
| TC-04-41 | PASS | B | Rời `/insights` đi nơi khác rồi quay lại: nút vẫn "Đã ghim · Xem thread", `/threads` vẫn **13**; tải lại trang: vẫn 13 và nút không đổi; mở liên kết → đúng thread (`h1` = "Chủ đề đang vướng: Mật mã đối xứng (AES, CBC)") |
| TC-04-42 | PASS | B | Ghim chủ đề 2 → **14** `thread-row`, đúng 2 thread "Chủ đề đang vướng" (một cho mỗi chủ đề); menu hồ sơ → `Đặt lại dữ liệu demo` → về **12** và nút trở lại "Tạo thread ghim" |
| TC-04-43 | PASS | B | Admin `/`: "Phân công · hôm qua 16:40"; `/admin/courses`: "Mới mở — hôm qua 16:40"; chuông GV: "Bạn được phân công lớp An ninh mạng – 761988… · Quản trị viên · hôm qua 16:40". Không thấy "08:30"; không thấy biến thể viết hoa "Hôm qua 16:40" ở chỗ khác mốc. Ảnh `056`, `019-admin-home`, `053-sweep-admin_admin_courses` |
| TC-04-44 | PASS | C | `tc_04_10`: `/` ≥ 1 "hôm qua 16:40"; `/admin/courses` ≥ 1; `/` không "08:30" |
| TC-04-45 | PASS | A + B | 1440 Admin: `[data-part=settings-section]` = 5 phần, khoảng cách = **[48, 48, 48, 48]** (lệch 0); bảng embedding có **3** hàng đều có nội dung (Mô hình / Số chiều / Đã đánh chỉ mục), không hàng trống. Ảnh `001`, `002`. Có lỗi thẩm mỹ kèm theo (BUG-v5-04-6) |
| TC-04-46 | PASS | A + B | 7 ngày: `chart-point` = **7**, `chart-axis-label` = **3** ("0", "42", "83" = 0 / nửa max / max); điểm cuối có nhãn "47 câu"; 7/7 điểm có `aria-label` ("T6 23/10: 40 câu" …) |
| TC-04-47 | PASS | B | Rê chuột lên điểm 3 → "CN 25/10: 71 câu"; Tab/`focus` (các điểm là `<circle tabindex="0">`) điểm 5 → "T3 27/10: 64 câu"; dòng "Cao nhất: 83 câu · Thấp nhất: 29 câu" khớp dữ liệu ngày (max 83 ở T2 26/10, min 29 ở T7 24/10). Ảnh `045`, `046` |
| TC-04-48 | PASS | A + B | 30 ngày: **10** điểm (cụm 3 ngày: "30/09–02/10" … "27/10–29/10"), 3 nhãn trục ("0", "92", "183"); lớp 2 vẽ theo số của lớp 2 (7 ngày max 23, 30 ngày max 55 — không phải 83/183); `?state=empty`: 0 `chart-point`, trạng thái rỗng sạch, không biểu đồ lỗi. Ảnh `048` |
| TC-04-49 | PASS | C | `tc_00_16` ("không còn số cứng overdue / answeredByAi / escalated", `derive.ts` có `ticketStats`), `tc_04_08`, `tc_04_10` đều PASS |
| TC-04-50 | PASS | B | 1100 và 390 px, 4 mốc đo (mặc định → mở "Cài đặt nâng cao" (`details.open = true`, lộ 5 tham số) → đổi embedding làm cảnh báo đánh chỉ mục lại hiện → đóng lại): khoảng cách luôn **[48, 48, 48, 48]**, không cặp phần nào chồng nhau. Ảnh `064`, `065` |

## Lỗi

| BUG | Route | Vai | Bước tái hiện | Thấy | Mong đợi | Mức | AC | Ảnh |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| BUG-v5-04-1 | `/observability` | Admin | Mở một hàng → để ô "Lý do xem nội dung" rỗng hoặc gõ 4 khoảng trắng → rời ô (Tab) hoặc gõ rồi xoá | Nút "Mở nội dung" chỉ bị vô hiệu; không có thông báo lỗi tại ô (`aria-invalid` null, 0 `[role=alert]`), người dùng không biết vì sao nút không bấm được | Báo lỗi tại ô (TC-04-09). **Câu hỏi cho PM:** nút vô hiệu có được tính là "báo lỗi tại ô" không, hay phải có dòng lỗi? | thấp | AC3 (TC-04-09) | `010-obs-lydo-rong-nut-vohieu.webp` |
| BUG-v5-04-2 | `/analytics` | GV | Mở `/analytics` lớp 761987 → đọc khối "Bảo vệ thông tin cá nhân" ở 7 ngày → bấm "30 ngày" → đọc lại | 30 ngày: câu tóm tắt đổi thành "**31 lần** … trong 30 ngày" nhưng ba dòng chi tiết vẫn "Chat riêng 6 · Bài nộp qua email 2 · Thread công khai 1" (= 9). Lớp 761988 cũng lệch: "5 lần" vs "2 + 1 = 3" | Dòng chi tiết đổi theo khoảng và cộng lại đúng số tóm tắt | **cao** (dữ liệu lệch) | SRS 4.6 `/analytics` "Đổi khoảng 7 / 30 ngày" (TC-04-19) | `063-analytics-30ngay-pii-lech.webp` |
| BUG-v5-04-3 | `/settings/llm`, `/settings/integrations` | Admin | Mở `/settings/llm?state=empty` (và `/settings/integrations?state=empty`) | Bản mẫu chung "Chưa có gì ở đây · Khi có dữ liệu, nó sẽ xuất hiện tại đây" — không nói chuyện gì của màn này, không hành động. (`/analytics`, `/observability`, Admin `/` có câu rỗng riêng nhưng không nút hành động) | Trạng thái rỗng dạy hành động kế tiếp (DESIGN §15) và có một hành động | vừa | TC-04-24 | `059-settings-empty-chung-chung.webp`, `057`, `058` |
| BUG-v5-04-4 | `/admin/users` | Admin | Đặt khung nhìn 720 px → mở `/admin/users` | Bảng rộng 612 px trong khung 456 px: cột "Trạng thái" và nút "Khoá" bị cắt ở mép phải; cụm lọc vai cũng chỉ còn "Tất cả" + "Giảng viên". Vùng cuộn ngang có (`overflow-x: auto`) nhưng **không** gắn `data-scroll-x`, không dòng gợi ý kéo ngang | Ở 720 px bảng không cắt chữ, hoặc nằm trong vùng `data-scroll-x` có gợi ý | vừa | SRS 4.7 h1 / 00-AC10 (AUDIT-biên; ngoài 50 TC của phần này) | `029-admin-users-720-cut.webp` |
| BUG-v5-04-5 | `/observability` | GV và Admin | Đặt khung nhìn 375 (hoặc 390) px → mở `/observability` | `innerWidth` nở 401 (400) px → tràn ngang 26 (10) px; tiêu đề "50 / 36 yêu cầu gần nhất" và câu mô tả bị bóp vào cột rộng 45 px, xuống dòng từng chữ. Nguyên nhân: cụm lọc `segmented` ("Tất cả / Lỗi và dự phòng / Có che thông tin") rộng 323 px nằm cùng hàng với tiêu đề | `ox = 0` ở 375 và 390; tiêu đề xuống dưới cụm lọc khi hẹp | vừa | 00-AC10 h1 (TC-04-33) | `018-obs-375-tran-ngang.webp`, `017-admin-obs-390-bang.webp` |
| BUG-v5-04-6 | `/settings/llm` | Admin, GV | Mở `/settings/llm` → cuộn tới "Mô hình embedding" | Panel vẽ gạch ngang phía **trên** hàng đầu và phía **dưới** hàng cuối, cộng đệm 25 / 40 px → nhìn như có một hàng trống ở đầu và cuối bảng (DOM thì chỉ có 3 hàng có nội dung) | Gạch chỉ ngăn giữa các hàng | thấp | AC11 (quan sát kèm TC-04-45) | `002-embedding-section.webp` |
| BUG-v5-04-7 | `/`, `/admin/courses`, `/admin/users` | Admin | Mở `/?state=empty`, `/admin/courses?state=empty`, `/admin/users?state=empty` | Con số ở đầu màn không đổi theo trạng thái rỗng: "4 việc đang chờ" bên trên "Hệ thống đang vận hành bình thường"; "2 lớp đang chạy" bên trên "Chưa có lớp nào trong học kỳ này"; "60 tài khoản · 57 sinh viên" bên trên "Chưa có tài khoản nào ngoài bạn" | Số đếm khớp trạng thái đang hiện | thấp | SRS 4.6 (trạng thái rỗng) | `020-admin-home-empty.webp`, `058-admin-courses-empty.webp` |

## Phản mẫu UI / phân quyền / ngưỡng 375–390 (kiểm ảnh theo DESIGN §21, §22)

- **Không có KPI wall** ở `/observability`: 5 số (1.284 / 3,1 s / 0,6% / 1,2% / 42.000 đ) nằm trên **một dải ngang** có vạch ngăn, không phải ≥ 3 thẻ cùng cỡ — ảnh `008`. Admin `/` cũng dùng dải tương tự dưới "Hệ thống hôm nay" — ảnh `019-admin-home`.
- **Đỏ chỉ là tín hiệu**: ở `/observability` đỏ chỉ ở chấm hàng "Lỗi"; ở Admin `/` đỏ chỉ ở nút chính "Mời giảng viên" / "Mở lớp" và mốc lỗi Gemini. Không đỏ trang trí.
- **Phân quyền**: SV và TA bị chặn 5 route hệ thống, GV bị chặn `/admin/*` (`tc_04_06`, `tc_00_matrix`); GV `/settings/llm` 0 nút; GV `/observability` không mở được hàng; nội dung yêu cầu chỉ Admin thấy sau khi nhập lý do và luôn kèm dòng nhật ký kiểm toán.
- **375–390 px**: 10/11 cặp (vai, route) giữ `innerWidth` đúng bằng khung nhìn. Riêng `/observability` tràn (BUG-v5-04-5). `/settings/llm` ở 390/375 xếp dọc chuẩn, nút cao 44 px, không `…` — ảnh `007`, `060`. `/insights`, `/analytics` ở 390 đọc tốt, biểu đồ vẫn có nhãn trục và nhãn điểm cuối — ảnh `061`, `062`.
- **720 px** là chỗ yếu của phần này: `/admin/users` cắt hai cột (BUG-v5-04-4) trong khi `/admin/courses`, `/settings/llm` ở cùng bề rộng vẫn sạch.

## Cảm nhận khi dùng như chủ dự án

- `/insights` là màn thuyết phục nhất: lý do, 5 tín hiệu có số, câu mẫu ẩn danh, "Nên làm" rất cụ thể, và chủ đề dưới 3 người hỏi tự giấu câu mẫu. Hai nút `Tạo thread ghim` / `Tạo buổi ôn tập` đi thẳng sang `/threads` và `/calendar` nên demo chạy liền mạch, không hụt.
- Trong lúc "Đang tạo báo cáo" (~3 s) toàn bộ danh sách chủ đề biến mất, chỉ còn 3 dòng tiến độ trên nền trắng rộng — trông trống. Nếu giữ báo cáo cũ mờ bên dưới thì đỡ hẫng hơn (`/insights`).
- `/admin/courses` ở 1440 px: hai hàng lớp rồi ~700 px trắng tới cuối trang — màn quản trị nhìn rỗng. `/admin/users` ngược lại: 60 hàng không phân trang, cuộn dài 5.000 px.
- `/observability` ở 390 px là chỗ gãy rõ nhất khi demo trên điện thoại: tiêu đề "50 yêu cầu gần nhất" rơi thành cột chữ dọc (BUG-v5-04-5). Trên 1440 px thì màn này đẹp và thuyết phục.
- Khối "Bảo vệ thông tin cá nhân" ở `/analytics` đọc là giả ngay khi bấm 30 ngày: 31 lần nhưng cộng ba dòng chỉ ra 9 (BUG-v5-04-2). Đây là chỗ một PM sẽ bắt đầu tiên.
- Trạng thái rỗng của `/settings/llm` và `/settings/integrations` ("Chưa có gì ở đây · Khi có dữ liệu, nó sẽ xuất hiện tại đây") là câu mặc định của khung, không phải câu của sản phẩm — lạc giọng hẳn so với các màn khác.
- Drawer `/observability` làm đúng tinh thần: bắt lý do, ghi nhật ký kèm tên và câu lý do vừa gõ, nội dung chỉ còn `[[SV_2]]`. Nhưng khi ô lý do rỗng thì nút chỉ xám đi, không nói gì (BUG-v5-04-1).

## Đề nghị

1. Sửa BUG-v5-04-2 trước (mức cao): rút ba dòng "Chat riêng / Bài nộp qua email / Thread công khai" từ cùng nguồn với số tóm tắt theo khoảng đang chọn.
2. Sửa BUG-v5-04-5 và BUG-v5-04-4: cho tiêu đề phần xuống dòng dưới cụm lọc khi hẹp (`/observability`), và gắn `data-scroll-x` + gợi ý kéo ngang cho bảng `/admin/users` ở 720 px.
3. Viết trạng thái rỗng riêng cho `/settings/llm`, `/settings/integrations` và thêm một hành động cho trạng thái rỗng `/analytics` (BUG-v5-04-3); đồng bộ số đếm ở đầu màn với trạng thái rỗng (BUG-v5-04-7).
4. Hỏi PM chốt TC-04-09: nút vô hiệu có đủ thay cho "báo lỗi tại ô" không. Nếu PM chốt là đủ, sửa TC kèm số proposal; nếu không, thêm dòng lỗi dưới ô lý do.
5. Phiếu giao việc ghi "51 TC" nhưng file TC chỉ có 50 hàng (TC-04-01…50) — thống nhất lại con số trước khi tổng hợp.

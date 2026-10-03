# Báo cáo QC — US-PU-04 (khung ứng dụng: phiên jwt/demo, điều hướng theo vai, chuông, màn chặn)
**Kết luận: PASS (sau vòng sửa 1, xem cuối)** — vòng 1: FAIL do 1 lỗi thật nhẹ (BUG-PU04-1, TC-21) và 3 TC đỏ do **công cụ QC 1.5 lỗi thời** so với thiết kế mới (TC-45, TC-53, TC-59; nguyên nhân từng dòng ở dưới, đề nghị góp ý #25). Phần đo được còn lại đạt: khung đúng kích thước 3 mốc, nav 5 vai khớp SRS 7.5, cổng token / phiên jwt, màn chặn 0 request, `no-backend` 40/40, build thường sạch.

- Bản chấm: `dacd348` (chứa `4d02364`), `build:gate` (có `DEV_AUTH`), `next start -p 3400`; Chrome for Testing riêng; gateway thật `testroutes` (token `gateway token`); `playwright shell.spec.ts` cổng 3510 (máy chủ giả :3312 là của dev, dùng chung theo `reuseExistingServer`).
- Q-QC-PU04-1: QC chấm theo bảng SRS 7.5 (GV 15) — khớp. Q-QC-PU04-2: `/api/v1/admin/llm/*` đã có (P1-04) nên TC-44 chạy thật. Q-QC-PU04-3: chưa kiểm (ngoài AC).

## Lỗi
### BUG-PU04-1 (thấp) — bảng "Thêm" ở điện thoại chỉ đóng bằng `Esc` / `Đóng` (AC5, TC-21)
Bảng "Thêm" là `<dialog>` **toàn màn hình** (rect 0,0,375×844 ở 375): không có vùng "ngoài" để bấm và vuốt xuống không đóng. Tái hiện: 375×844 cảm ứng, bấm "Thêm"; (1) bấm tại y=5 → vẫn mở; (2) kéo từ giữa đầu bảng xuống 250 px → vẫn mở; `Esc` → đóng và focus về nút "Thêm" (đúng). TC đòi cả ba cách. Số mục đủ (SV 3 = 7−4, TA 8, GV 11, Admin 2).

## Công cụ QC 1.5 đỏ do thiết kế đổi (đề nghị #25: QC cập nhật công cụ, không phải lỗi dev)
- **TC-45 / TC-59 (`proto-curl.sh all` = 492 PASS / 5 FAIL):** 
  - đã sửa phép nhận màn chặn: lời mới "Bạn không có quyền **xem màn** này" (AC10) thay "…mở trang này" — trước sửa 350/147; sau sửa 492/5 (ma trận vai × route đạt hết);
  - còn 4 FAIL `tc_00_hooks`/`tc_04_07`: kỳ vọng `data-part=provider-status/-action` ≥ 3 ở `/settings/llm` — nay không có token thì là **cổng dán token** (AC9, đúng thiết kế);
  - 1 FAIL `tc_00_14`: dòng `"N phút trước"` nằm trong dữ liệu mẫu của `app/dev/data/DevData.tsx:184` (trang dev, ngoài quy tắc N6).
- **TC-53 (`audit.mjs`):** SV **165/0**, GV **170/0**, TA **106/0**, Admin **54/0** (bốn lượt vai sạch khi xoá `localStorage` giữa các lượt — AC1 giờ **nhớ** trạng thái thu gọn `ep:ui:sidebar`, nên bước "Thu gọn" của kịch bản làm các lượt sau lệch `LEFT 240→96`). Lượt spec **188 hàng, 12 FAIL**, đều do công cụ: 5 hàng `00-AC7 brand` (kịch bản lấy `[data-part=brand]` **đầu tiên** trong DOM; AC12 buộc thứ tự header → sidebar nên nhận phải dấu "mark" ẩn của thanh trên; đo tay brand cao 56, logo 32, mark 28 ở 390 — TC-54 đạt), 6 hàng `04-AC7/11 /settings/llm` (bố cục mock nay sau cổng token), 1 hàng `01-AC17 sidebar chỉ có Hôm nay` (sidebar thu gọn còn lại từ bước trước, `innerText` rỗng). `sweep` SV 42 hàng `FORBIDDEN` 0, 0 cuộn ngang.

## Kết quả từng TC
| TC | KQ | Bằng chứng |
| --- | --- | --- |
| 01–03 | PASS | GV: sidebar **216** ở 1440 và 1100, **72** ở 1099/900/720, không có ở 719/390/375; header **56** mọi bề rộng; ≤ 719 có `[data-part=bottom-nav]` `position: fixed` |
| 04 | PASS (một phần) | nút `Thu gọn thanh bên` `aria-expanded` true; 216 → 72; tải lại vẫn 72; `ep:ui:sidebar=collapsed` (không phải token). Chưa đo ở 1100 |
| 05, 06, 16, 24, 26, 33, 40, 52, 56, 58 | PASS | `playwright shell.spec.ts`: 15 pass, 17 skip (ca `no-backend`/chỉ-desktop theo thiết kế), rc=0 |
| 07, 09 | PASS | mục đang chọn: 1 `aria-current="page"`, `::before` rộng **2px** màu `--ep-red`, nền mục trắng, chữ đậm 600; `ui-antipatterns.sh`: `✓ Khung vỏ dùng nền đặc` |
| 08, 18, 30–32, 55 | chỉ spec dev | chấm đỏ / chuông / huy hiệu về 0: QC chưa đo độc lập (sẽ làm khi có thông báo thật) |
| 10–14 | PASS | SV B 7 mục đúng thứ tự (Hôm nay…Kết quả của tôi); SV D chỉ "Hôm nay"; TA **12** mục + nhóm Làm việc 4 / Đánh giá 3 / Nội dung 3 / Hiểu lớp học 2, không "Hệ thống"; GV **15** (+ Hệ thống: Quan sát AI, Cấu hình LLM, Tích hợp); Admin **6** (Vận hành / Quản trị / Cấu hình). (Sidebar có thêm 1 liên kết logo `/` không tính) |
| 15 | PASS | ma trận vai × route của `proto-curl.sh` (492 PASS) |
| 17, 19 | PASS | huy hiệu chỉ `nav-badge-inbox` (5) và `nav-badge-grading` (4) ở GV/TA; SV/Admin không có; `grep 'badge: [0-9]' nav.ts` = 0 |
| 20, 23, 57 | PASS | 4 vai × 375/390: bottom-nav **5** phần tử (4 đích + `Thêm`), cao ≥ 56 px, rộng ≥ 75 px; `AUDIT_SRC` `ox 0, cut 0, ell 0`; `TOUCH_SRC` `[]` ở `/` mọi ca (các route đích thanh dưới: spec dev `touch`) |
| 21 | **FAIL** | BUG-PU04-1 |
| 22 | PASS | theo góp ý #22 (đã ACCEPTED): thanh trên 375 giữ khung 1.5 (logo mark, chọn lớp, tìm, chuông, hồ sơ), `h1` nằm trong `main`, 0 `h1` ở `header` |
| 25 | PASS | `header h1` = 0; 4 nhóm: "Chọn lớp, đang xem 761987 · …", "Tìm nhanh hoặc đi đến", "Thông báo", "Tài khoản: TS. Lê Thu Hà" (không tên vai) |
| 27–29 | PASS | `Ctrl K` mở palette `role=combobox`, focus ở ô nhập; "diem danh" → "Điểm danh" đứng đầu (1 kết quả); ↓/↑ đổi `aria-activedescendant`; `Enter` → `/attendance`; `Esc` đóng và trả focus về nút mở ("Tìm nhanh hoặc đi đến"); ô trống GV = 12 mục, SV = **7** mục; SV "diem danh" → 0; "zzzz" → "Không thấy mục nào khớp. Thử từ khác, ví dụ “điểm”."; bẫy focus: 10 lần Tab không ra phần tử trang nào (khi danh sách chỉ còn 1 mục, vòng Tab đi qua `body` một nhịp rồi về hộp — hành vi `<dialog>` modal chuẩn) |
| 34 | PASS | token ADMIN dán ở cổng → nav Admin 6 mục, không "Đổi vai", hồ sơ "Tài khoản: dev@edupilot.local" (đúng claim `email`), menu có **Đăng xuất** (và "Đặt lại dữ liệu demo"); không còn ô password |
| 35 | PASS | Đăng xuất → trở lại cổng "Dán token…", bộ đổi vai demo không hiện |
| 36 | PASS | token hết hạn (ký đúng) → "Phiên đã hết hạn / Dán token mới để tiếp tục"; `role=SUPERUSER` (ký đúng), `alg=none` → "Token không hợp lệ."; token bị bỏ (cổng vẫn còn). Ghi chú: token đổi 4 ký tự cuối chữ ký được **nhận** (khung chỉ giải mã HS256 + role + exp, không có khoá để kiểm chữ ký — đúng mô tả của dev; gateway mới là nơi từ chối) |
| 37 | PASS | `/settings/llm` (GV, bản gate, chưa token): "Dán token quản trị để tiếp tục", `input[type=password]` `autocomplete="off"` |
| 38 | PASS | `abc`, `a.b.c`, rỗng → "Token không hợp lệ."; `localStorage`/`sessionStorage` không đổi; sau khi dán hợp lệ: token (phần chữ ký) **không** có trong DOM, URL, cookie, storage |
| 39 | PASS | `pnpm build` thường: `grep -rl 'DEV_AUTH\|Dán token' .next/static` = **0**; `/settings/llm`: "Cần đăng nhập" + "sẽ có ở bản sau", **0** `type="password"`; `/dev/ui`, `/dev/data` 404; `state-cell` = 0 |
| 41 | PASS | SV và TA ở `/settings/llm`, `/observability`, `/admin/users`: "Bạn không có quyền xem màn này" + "Trang này dành cho giảng viên và quản trị viên." (hoặc "quản trị viên") + 1 liên kết `Về Hôm nay`; **0** request `/api/v1/` (đã kiểm `requests()` ghi nhận 46 request ở lượt đối chứng) |
| 42 | PASS | GV: `/admin/courses` bị chặn; `/settings/llm` mở được (cổng token, chỉ xem) |
| 43 | PASS | SV D `/chat`, `/threads`: "Bạn chưa vào lớp nào" + ô mã tham gia, 0 request API |
| 44 | PASS | gateway thật: STUDENT và TA → `403` ở `GET providers`, `PUT routes`, `PUT budget`; GV `GET providers` `200`, `PUT` `403` |
| 45 | **FAIL** (công cụ) | xem trên: 492/5 |
| 46, 47 | PASS | bản `NEXT_PUBLIC_MOCK_SCREENS=0 DEV_TOOLS=1 DEV_AUTH=1`: 4 vai × mọi mục nav = **40 cặp**, 0 sai: có `[data-part=empty-no-backend]` đúng **tên phase** theo SRS 7.6 (P2…P10), đúng 1 nút, mục nav còn; `/settings/llm` không bị thay; `/dev/ui` còn. SV: "Tính năng này đang được xây ở giai đoạn P3 — Hỏi đáp. Khi xong, bạn sẽ dùng nó ngay tại đây." — không từ kỹ thuật |
| 48 | PASS | bản mặc định: màn mock chạy như 1.5 (không `empty-no-backend`; `audit.mjs` bốn vai sạch) |
| 49 | PASS | QC tự chạy bản `MOCK_SCREENS=0` (TC-46) thay vì `-g no-backend` |
| 50, 51 | PASS | Tab đầu = "Bỏ qua điều hướng" (hiện khi focus), Enter → focus trong `main`; 1 `main`, `nav[aria-label]`; thứ tự Tab: bỏ qua → chọn lớp → tìm nhanh → chuông → hồ sơ → logo → sidebar (Hôm nay, Hộp thư 5, …); `Esc` ở chuông và menu hồ sơ đóng và trả focus về nút mở |
| 53 | **FAIL** (công cụ/trạng thái) | xem trên: 4 lượt vai 0 FAIL, spec 12 hàng |
| 54 | PASS | `LEFT page-title` **240** (1440) / **16** (390); brand cao 56, logo đầy đủ cao 32; mark thanh trên cao 28 |
| 59 | **FAIL** (công cụ) | `sweep` SV 42/0 `FORBIDDEN`; `pnpm lint` rc=0; `ui-antipatterns.sh` rc=0; `proto-curl.sh` 492/5 (xem trên) |

## Việc sau
Dev: BUG-PU04-1 (hoặc BA nới AC5: bảng toàn màn chỉ cần `Esc`/`Đóng`). PM: duyệt #25 để QC cập nhật `audit.mjs` (chọn `[data-part=brand]` đang hiện, xoá `ep:ui:sidebar` giữa các bước, chạy `/settings/llm` kèm token) và `proto-curl.sh` (hook `/settings/llm` theo cổng token; bỏ `app/dev`).

## Vòng sửa 1 (dev `9cf0d44`, góp ý #32; đo trên `fc7c920`)
- **BUG-PU04-1 đã sửa (TC-21 PASS):** bảng "Thêm" ở 375×844 là bảng trượt từ đáy (top y=211, cao 633, ≤ 75 % màn): `Esc` → đóng, focus về nút "Thêm"; **bấm vùng phía trên bảng** (y≈105) → đóng; **vuốt xuống** (cảm ứng thật, touchStart/move/end 260 px) → đóng và focus về "Thêm". (Chuột kéo giả lập `mouse.down/move` không đóng — không phải cử chỉ cảm ứng.)
- TC-45/53/59 (công cụ QC 1.5 lỗi thời) vẫn chờ cập nhật công cụ (#25 ở đây là đề nghị riêng của QC; góp ý #25 chính thức của PM nay là `--ep-ink-3`): `proto-curl.sh` đã nhận lời màn chặn mới (492 PASS / 5 FAIL cũ do `/settings/llm` nay là cổng token); xem cổng PU.
- **Verdict US-PU-04: PASS** (TC-45/53/59 xử lý ở cổng PU bằng công cụ đã cập nhật).

# QC test case — US-PU-01 (token + font + chuẩn hoá nền + lint chặn giá trị viết cứng)
Nguồn: `docs/specs/FEAT-ui-foundation/US.md` US-PU-01 AC1–AC10 + `SRS.md` 4.1 (7 lớp giá trị cứng), 4.2 (7 luật `no-restricted-*`), 5.2 (4 token thêm, danh sách `ui-allow`), 8.1 (`NEXT_PUBLIC_DEV_*`). Hộp đen: chỉ chạy lệnh / đọc artefact như AC; không đọc mã nguồn để chọn bộ chọn. Nền so sánh: `docs/sprints/3/qc/audit-baseline.md` (`307bfd2`).

Tiền điều kiện chung: worktree `TA_Agent_v2-s3`; `pnpm install --frozen-lockfile`; biến của "Quy ước kiểm chung" (US.md): `FE="pnpm -C frontend"`, `PW="$FE exec playwright test"`, `BASE=http://localhost:3300`, `gbuild` (`NEXT_PUBLIC_DEV_TOOLS=1 NEXT_PUBLIC_DEV_AUTH=1 NEXT_PUBLIC_MOCK_SCREENS=1`), `pbuild`. Kiểm hộp đen chạy trên **bản build** (`gbuild && $FE exec next start -p 3300`), không dùng `next dev`. Công cụ: **S** = lệnh shell (QC bọc thành `scripts/pu01.sh`, hàm `tc_pu01_NN`), **P** = Playwright (`$PW tokens.spec.ts -g …`), **A** = `audit.mjs` / `sweep.mjs` (Eval, trình duyệt thật), **T** = tay. Dev chưa có `frontend/e2e/tokens.spec.ts` / `scripts/lint-selftest.sh` thì TC tương ứng **FAIL "KHÔNG KIỂM ĐƯỢC"** (QC không tự viết thay).

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-PU01-01 | AC1 | repo sạch | **S** `comm -23 /tmp/kit /tmp/own \| wc -l` với `kit`/`own` dựng đúng lệnh Kiểm AC1 (`DESIGN_TOKENS.css` vs `frontend/src/shared/styles/tokens.css`) | `0` (mọi khai báo `--ep-*` của bản gốc có mặt y nguyên, kể cả giá trị) |
| TC-PU01-02 | AC1 | – | **S** `comm -13 /tmp/kit /tmp/own \| wc -l`; `comm -13 … \| grep -oE '^--ep-[a-z0-9_-]+' \| sort` | `4`; đúng tên `--ep-mask-clear --ep-mask-solid --ep-scrim --ep-shadow-composer` (không token thừa / sai tên) |
| TC-PU01-03 | AC1 | – | **S** `grep -oE '^\s*--ep-[a-z0-9_-]+:' docs/design/DESIGN_TOKENS.css \| wc -l` | `51` (số dòng gốc không đổi — chặn việc sửa bản gốc cho khớp) |
| TC-PU01-04 | AC1 | – | **S** `grep -rnE '^\s*--ep-[a-z0-9_-]+\s*:' frontend/src --include=*.css \| grep -v 'shared/styles/tokens.css' \| wc -l` | `0` (`tokens.css` là nơi duy nhất khai báo giá trị) |
| TC-PU01-05 | AC2 | build được | **S** `pnpm -C frontend lint; echo rc=$?` | `rc=0`, 0 lỗi / 0 cảnh báo mới so với `audit-baseline.md` (nền: rc=0) |
| TC-PU01-06 | AC2 | – | **S** `bash scripts/ui-antipatterns.sh; echo rc=$?`; đếm dòng không bắt đầu `✓` | `rc=0`; mọi dòng bắt đầu `✓`; **≥ 11** dòng (nền = 11; thêm phép thì tăng, không giảm) |
| TC-PU01-07 | AC2 | – | **S** `grep -rn 'ui-allow:' frontend/src \| wc -l`; so danh sách với bảng 10 chỗ ở `audit-baseline.md` / SRS 5.2 | ≤ `10`; không chỗ **mới** ngoài danh sách (thêm chỗ = phải có `proposals.md` PM duyệt) |
| TC-PU01-08 | AC2 (lớp 1–2) | – | **S** gieo thử vào tệp tạm `frontend/src/features/qc_probe.module.css`: `.a{color:#ff0000}`, `.b{background:rgb(1,2,3)}`, `.c{border-radius:14px}`; chạy `bash scripts/ui-antipatterns.sh`; xoá tệp | Mỗi dòng gieo bị bắt (`rc≠0`, nêu lớp màu / bo góc); sau khi xoá `rc=0`; `git status --short frontend` rỗng |
| TC-PU01-09 | AC2 (lớp 3–5) | – | **S** như 08 với `.d{box-shadow:0 2px 4px red}` ngoài `shared/ui/{Popover,Dialog,Menu,Drawer,Composer}`, `.e{font-size:15px}`, `.f{z-index:99}`; và ca **lành** `.g{border-radius:0}`, `.h{border-radius:50%}`, `.i{z-index:1}`, `.j{font-size:inherit}` | Ba ca xấu bị bắt; bốn ca lành **không** bị bắt (không báo nhầm) |
| TC-PU01-10 | AC2 (lớp 6–7) | – | **S** tệp tạm `.tsx`: `<div className="text-[13px] rounded-2xl shadow-lg bg-gray-100"/>` và `<div style={{ color: '#fff', borderRadius: 12 }}/>`; chạy lint + antipatterns | Cả hai bị bắt; xoá tệp → sạch |
| TC-PU01-11 | AC3 | `scripts/lint-selftest.sh` có | **S** `bash scripts/lint-selftest.sh; echo rc=$?`; `git status --short frontend \| wc -l` | `rc=0`; in đúng `7 / 7 luật ESLint bắt được` và `19 / 19 phép ui-antipatterns bắt được`; `0` dòng git (cây sạch) |
| TC-PU01-12 | AC3 | – | **S** chạy `bash scripts/lint-selftest.sh` rồi ngắt giữa chừng (`kill -INT` sau 2 s) | Tệp tạm trong `frontend/src/__selftest__/` bị xoá (kể cả khi lỗi): `ls frontend/src/__selftest__ 2>&1` báo không có; `git status --short frontend` rỗng |
| TC-PU01-13 | AC3 (đối chứng âm độc lập) | – | **S** QC tự gieo **từng** trong 7 vi phạm AC4 (một tệp `.tsx` mỗi lần) rồi `pnpm -C frontend exec eslint <tệp>` | Mỗi lần exit ≠ 0 và thông báo nêu **đúng tên luật** của SRS 4.2 (không dùng `lint-selftest.sh` của dev để chứng minh chính nó) |
| TC-PU01-14 | AC4 (1) | – | **S** `grep -rnE '\bfetch\(\|XMLHttpRequest\|from "axios"' frontend/src \| grep -v 'src/shared/data/' \| wc -l`; ESLint ca âm: `fetch('/x')` trong `features/` | `0`; ca âm bị chặn |
| TC-PU01-15 | AC4 (2) | – | **S** `grep -rnE 'confirm\(\|alert\(' frontend/src \| wc -l`; ca âm `window.confirm('x')` | `0`; ca âm bị chặn |
| TC-PU01-16 | AC4 (3)(4) | – | **S** ca âm: `<Spinner/>` ngoài `shared/ui/`; `<table>` thô ngoài `shared/ui/DataTable` | Cả hai bị chặn; `grep -rn '<table' frontend/src \| grep -v 'shared/ui/DataTable' \| wc -l` → `0` |
| TC-PU01-17 | AC4 (5) | – | **S** ca âm: `localStorage.setItem('access_token','x')`, `sessionStorage.setItem('jwt','x')`, `document.cookie='refresh=1'`; ca lành: `localStorage.setItem('ep_demo_state','{}')` | Ba ca âm bị chặn; ca lành **không** bị chặn (không chặn nhầm khoá không nhạy cảm) |
| TC-PU01-18 | AC4 (6)(7) | – | **S** `grep -c 'tailwind' frontend/package.json`; `grep -rn '@tailwind\|@theme' frontend/src \| wc -l`; ca âm `import 'tailwindcss'`; ca âm `style={{ color: '#fff' }}` | `0`; `0`; hai ca âm bị chặn |
| TC-PU01-19 | AC5 | `gbuild` + `next start -p 3300` | **P** `$PW tokens.spec.ts -g 'font'`; tay bổ sung: Eval mở `/login`, `document.fonts.check('400 16px "Be Vietnam Pro"')` cho 400 / 500 / 600 / 700 | Cả bốn `true`; `getComputedStyle(body).fontFamily` bắt đầu `"Be Vietnam Pro"`; rc=0 |
| TC-PU01-20 | AC5 | – | **A** mở `/login` bằng trình duyệt thật, ghi mọi request (CDP `Network`) | **0** request tới `fonts.googleapis.com` / `fonts.gstatic.com` lúc chạy (font tự lưu cùng máy chủ) |
| TC-PU01-21 | AC5 | – | **S** `grep -c "weight: \[" frontend/src/app/layout.tsx`; in dòng đó; `grep -n 'display' frontend/src/app/layout.tsx` | `1`; dòng chỉ chứa `"400", "500", "600", "700"`; `display: 'swap'`; có tập con `vietnamese` |
| TC-PU01-22 | AC5 | – | **A** đọc `getComputedStyle(documentElement).getPropertyValue('--ep-font')` | Thứ tự rơi: `"Be Vietnam Pro"`, `"Noto Sans"`, `ui-sans-serif`, `system-ui`, … (đúng như AC5) |
| TC-PU01-23 | AC6 | – | **P** `$PW tokens.spec.ts -g 'tabular'`; **A** tay: GV `/gradebook`, mọi `td`/`th` của `DataTable` | `fontVariantNumeric` chứa `tabular-nums` ở **mọi** ô (đếm ô thiếu = 0) |
| TC-PU01-24 | AC6 | – | **P** `$PW tokens.spec.ts -g 'diacritics'`; tay: chuỗi "Ặ Ế Ộ Ử Ữ Ầ" ở `h1`, `h2`, nút, ô nhập (`/dev/ui` hoặc route có sẵn) | `scrollHeight ≤ clientHeight + 1` mọi phần tử; `h1` cao ≥ 1,25 × `font-size` |
| TC-PU01-25 | AC6 | – | **A** thu cỡ chữ (`font-size` tính ra) của mọi phần tử chữ ở `/`, `/chat`, `/threads` (SV), `/gradebook` (GV) | Mọi cỡ thuộc tập sáu `--ep-text-*` (không cỡ lạ) |
| TC-PU01-26 | AC7 (1) | `/dev/ui` có | **P** `$PW tokens.spec.ts -g 'focus'`; tay: `Tab` đến nút đầu tiên; sau đó click chuột vào một nút | Sau `Tab`: `boxShadow` tính ra khớp `--ep-focus` (đỏ 2 px + khoảng 2 px); sau click chuột: **không** hiện vòng |
| TC-PU01-27 | AC7 (2)(3)(5) | – | **A** `getComputedStyle(documentElement).colorScheme`; chọn chữ rồi đọc `::selection`; kiểm cuộn mảnh (`scrollbar-width`/`::-webkit-scrollbar`) | `colorScheme = light`; `::selection` nền `--ep-red-soft`; thanh cuộn mảnh theo bảng màu; không có `prefers-color-scheme: dark` nào trong CSS đã build |
| TC-PU01-28 | AC7 (4) | – | **P** `$PW tokens.spec.ts -g 'reduced'`; tay: `emulateMedia({reducedMotion:'reduce'})`, đọc `transitionDuration`/`animationDuration` của nút, thẻ, dialog, Red Thread Transition | Mọi giá trị ≤ `0.001s`; Red Thread **không chạy** (không phần tử hoạt hình) |
| TC-PU01-29 | AC8 | – | **S** `curl -s -o /dev/null -w '%{http_code}\n' $BASE/brand/favicon.svg`; `curl` hai logo; `<link rel="icon">` trong HTML | `200` cả ba; `rel=icon` trỏ tệp trả 200 |
| TC-PU01-30 | AC8 | – | **P** `$PW tokens.spec.ts -g 'brand'`; tay ở sidebar mở và thu gọn | `[data-part=brand] img` `src` kết thúc `logo-edupilot.svg` (thu gọn: `-mark.svg`); cha có `borderRadius: 0px`, `backgroundColor: rgba(0, 0, 0, 0)` (không ô vuông bo góc) |
| TC-PU01-31 | AC9 | – | **S** `gbuild; echo rc=$?`; grep `Failed to load font` trong log | `rc=0`; 0 cảnh báo font |
| TC-PU01-32 | AC9 | bản `gbuild` chạy :3300 | **A** `audit.mjs` bốn vai + phần spec (đúng cách `audit-baseline.md`), `base:'http://localhost:3300'` | Mỗi lượt **FAIL 0**; số hàng PASS **≥** nền: SV 165, GV 170, TA 106, Admin 54, spec 185 (tổng 680); ghi vào `audit-log.md` |
| TC-PU01-33 | AC9 / AC10(sprint) | – | **A** `sweep.mjs only:'student'` + `proto-curl.sh all` (`F=http://localhost:3300`) | `FORBIDDEN` = 0; proto-curl **≥ 497 PASS, 0 FAIL** |
| TC-PU01-34 | AC9 | có ảnh mốc `visual.spec.ts-snapshots/` (US-PU-05) | **P** `$PW visual.spec.ts` sau khi PU-01 gộp; đối chiếu `docs/sprints/3/qc/shots/before/` bằng mắt cho 6 route mock | Lệch ≤ `maxDiffPixelRatio 0.005`; nếu lệch lớn hơn: so ảnh trước/sau, ghi route + vùng lệch (không tự cập nhật ảnh mốc) |
| TC-PU01-35 | AC10 | – | **S** `grep -rnE "(localStorage\|sessionStorage)\.setItem\(\s*['\"\`][^'\"\`]*(token\|jwt\|access\|refresh\|secret\|password)" frontend/src \| wc -l` | `0` |
| TC-PU01-36 | AC10 (an toàn) | – | **A** mở `/login`, `/`; đọc `localStorage` và `document.cookie` sau thao tác bình thường | Không khoá chứa `token|jwt|access|refresh|secret|password` ở localStorage / cookie JS-đọc-được |

## Nhánh lỗi / biên đã phủ
| Tình huống | TC |
| --- | --- |
| Bản gốc token bị sửa cho khớp | 03 |
| Lint không bắt thật (tự kiểm của dev tự chứng minh) | 13 (QC tự gieo) |
| Chặn nhầm ca lành | 09, 17 |
| Tệp tạm của selftest để lại khi lỗi | 12 |
| Font gọi ra ngoài | 20 |
| Phá màn mock | 32–34 |
| Token ở localStorage | 17, 35, 36 |

## Câu hỏi cho BA / PM
- **Q-QC-PU01-1** — AC3 đòi in đúng chuỗi `7 / 7 luật ESLint bắt được`; AC4 liệt kê (1)…(7) nhưng SRS 4.2 đặt tên luật: QC cần bảng tên luật để TC-PU01-13 so "đúng tên luật". (a) QC lấy tên từ SRS 4.2; (b) dev in danh sách trong handoff. QC chọn (a) và đối chiếu — *chờ xác nhận*.
- **Q-QC-PU01-2** — AC9 / TC-PU01-34: ảnh mốc 14 tấm chỉ có sau US-PU-05; trước đó TC-34 chỉ so mắt với `shots/before/`. Chấp nhận? — *chờ trả lời*.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo US.md (FEAT-ui-foundation, APPROVED 2026-10-03).

Tổng: 36 TC.

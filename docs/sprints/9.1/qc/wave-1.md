# QC đợt 1 — Nền chung (US-UX91-02, 03, 04, 21)

**Kết luận: FAIL.** US-02 PASS. US-03 FAIL (AC2, AC4, AC8). US-04 FAIL (AC7, AC8). US-21 FAIL (AC3).

## Môi trường
| Mục | Giá trị |
| --- | --- |
| Commit kiểm | `62f4833b` (nhánh `sprint/9.1-uiux`; chứa `4b6389f1` = đợt 1 + 2 chung một commit, và đợt 6) |
| Lần kiểm đầu | `4b6389f1`. Hai điểm khác `62f4833b`: `Landing.module.css:49 .link:hover` nằm ngoài `@media (hover: hover)` (US-04 AC2) và `Landing.tsx:46` còn «Threads» (US-07 AC4). Cả hai đã sửa ở đợt 6, kết luận dưới đây là ở `62f4833b` |
| Hạ tầng | VPS (12 vCPU / 32 GB), `/srv/work/qc`, compose `-p ep-qc` (gateway + Postgres + Redis + MinIO + docling thật, seed 19 bước), Caddy `https://localhost:14443`, Playwright 1.63 Chromium. Không đụng `/opt/edupilot` |
| Cách chạy | (a) harness e2e (`next start` + gateway giả) cho ca cần dữ liệu cố định: `frontend/e2e/qc-uiux91-w1.spec.ts` (29 ca); (b) stack thật: `qc-uiux91-w12b-real.spec.ts`, `playwright.qc-real.config.ts`; (c) grep / `ui-antipatterns.sh` |
| Suite có sẵn ở `62f4833b` | `pnpm exec playwright test` (desktop + mobile): **839 passed, 491 skipped, 0 failed** (12,5 phút). `pnpm lint` sạch. `pnpm build` (docker build frontend) qua |

Spec QC `qc-uiux91-w1.spec.ts` ở lần chạy cuối: **25 pass / 4 fail** (29 ca) (4 fail đều là lỗi sản phẩm bên dưới, đã loại các fail do QC giả định sai; xem «Sửa TC»).

## Kết quả theo AC

### US-UX91-02 — Ngày giờ: PASS
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX02-1 | 1 | PASS | `rg Intl.DateTimeFormat|toLocale*` chỉ còn `CodeEditor.tsx:9 n.toLocaleString("vi-VN")` (định dạng số, không phải ngày) |
| UX02-2 | 2 | PASS | Threads I1…I9 đúng chuỗi; I5 dùng 22/09/2025 (xem «Sửa TC») |
| UX02-3, 4, 5 | 2 | PASS | Chat I1…I4; Tài liệu (GV) I3/I4; Thư viện (SV) dùng dạng «chỉ ngày» (`cập nhật 02/11`); Lịch `02/11`, `08/11`. Ca khác năm (`21/09/2025`) chỉ có ở bộ kiểm đơn vị của dev, seed không có ngày khác năm |
| UX02-6 | 2 | PASS | `time-ago.spec.ts` (đã soát: «Hôm qua, 23:40», «Thứ Năm, 16:40», «Thứ Tư, 31/12/2025 · 16:40» khớp bảng AC2) |
| UX02-7, 8 | 3 | PASS | `America/Los_Angeles`, `Pacific/Kiritimati`: Threads I1/I2/I3 giữ giờ ICT |
| UX02-9 | 4 | PASS | `"abc"`, `null`, `""`, `"2026-13-45"` → «—», không `Invalid Date`, không lỗi console |
| UX02-10 | 5 | PASS | `datetime.spec.ts` 5 ca, nằm trong suite 839 pass |
| UX02-11 | 5 | PASS có giới hạn | QC không import mô-đun bằng `node`; kiểm bằng UI I1…I9 + `datetime.spec.ts` |
| UX02-12 | 6 | PASS | ảnh sau `shots/wave-2/{threads,chat,inbox}-{1440,375}.png`; `visual.spec.ts` pass; chỉ `inbox-1440`, `threads-390` đổi |
| UX02-13 | 7 | PASS | 375 px, I5 không tràn ngang |
| UX02-PQ | — | PASS | SV `/documents`, `/inbox` → «Bạn không có quyền xem màn này», không lộ ngày giờ |

### US-UX91-03 — Nút khoá có lý do: FAIL
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX03-1…4, 12 | 1, 3 | PASS | `/dev/ui`: `aria-disabled`, `aria-describedby`, click không kích hoạt, `opacity`, bàn phím |
| UX03-5 | 1 | PASS | Diff test dev sửa chỉ đổi theo spec (nhãn, `ep_returning`, skip hover ở project mobile theo US-04 AC2); không nới assertion |
| UX03-6 | 2 | **FAIL (Thấp)** | Sổ điểm, công thức chưa xác nhận: `Chốt điểm` khoá đúng, nhưng lý do là «Chưa có công thức điểm đã xác nhận», AC2 ghi «Cần xác nhận công thức trước» (P-91-7) |
| UX03-7 | 2 | PASS | «Còn N mục chưa rõ» |
| UX03-8 | 2 | **FAIL (Trung bình)** | `/grading`, hàng chờ rỗng, **đã chọn một bài tập**: `Chấm tất cả` **không khoá** (`aria-disabled` không có), không có lý do «Chưa có bài nộp». Tái hiện: harness, `gradingApi({rows: []})`, mở `/grading?tab=queue`, chọn bài tập A1. Chưa chọn bài tập thì nút khoá với «Chọn một bài tập trước.» |
| UX03-9, 10 | 2 | PASS | Có bài nộp → bấm được; chưa chọn bài tập → `Tải bài lên` khoá «Chọn một bài tập trước» |
| UX03-11 | 2 | PASS | `Phiên mới` không thuộc danh sách khoá (P-91-5 Q2); ảnh `chat-375` cho thấy nút mờ khi chat trống, không có lý do (ngoài US-03) |
| UX03-13 | 4 | **FAIL (Thấp)** | Cảm ứng 375 px: chạm nút khoá, lý do hiện **trên** nút (`rBottom 573,7 < bTop 581,7`), AC4 ghi «dưới nút». Lý do còn hiện sau 5 s (≥ 3 s đạt) |
| UX03-14 | 5 | PASS | `disabledReason` mà không `disabled`: không `aria-describedby`, không chữ lý do, bấm được |
| UX03-15, 16 | 6 | PASS | SV không có nút của giảng viên trong DOM; lý do là chuỗi tĩnh (không tên / MSSV / điểm) |
| UX03-17 | 7 | PASS | Tooltip `transition: opacity 0,12 s`; `reduce` → 0,001 s |
| UX03-18 | 8 | **FAIL (Thấp)** | AC8 «ảnh mốc `dev-ui-*`, `gradebook-*` đổi»: ở `62f4833b` hai ảnh này không đổi (chỉ `inbox-1440`, `threads-390`). `DESIGN.md` cũng không có đoạn nào về lý do nút khoá |
| dev tự khai | 1 | ghi nhận | `ButtonLink` và `Switch` dùng `title` native, không hiện khi focus bàn phím (handoff đợt 1) |

### US-UX91-04 — Chuyển động: FAIL
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX04-1, 2 | 1 | PASS | Mọi biến thể `Button`: `:active` → `matrix(0.97…)` ≤ 100 ms, trở lại sau thả. (`Tabs`, `Switch` không phải biến thể `Button`, không tính) |
| UX04-3 | 2 | PASS ở `62f4833b` | 71 luật `:hover`, 0 ngoài `@media (hover: hover)`. Ở `4b6389f1` là 1 (`Landing.module.css:49`) |
| UX04-4 | 2 | PASS | Gián tiếp: không còn `:hover` ngoài media nên chạm không dính |
| UX04-5 | 3 | PASS | `--ep-fast: 140ms`, `--ep-slow: 200ms` |
| UX04-6 | 3 | PASS | Hộp thoại `Chốt điểm` mở ≤ 200 ms |
| UX04-7 | 3 | PASS | `DESIGN.md:486` «Drawer/dialog: ≤ 200ms» |
| UX04-8, 9 | 4 | PASS | Không còn `transition: width`; thanh tiến độ `scaleX` |
| UX04-10 | 5 | PASS | Thu gọn sidebar: `main` không transition `margin-left`, tức thì, CLS ≤ 0,05 |
| UX04-11 | 6 | PASS | Menu tài khoản: gốc xuất phát ở nút gọi, ≤ 160 ms |
| UX04-12, 13, 14, 19 | 7 | **FAIL (Trung bình)** | Nháy nền xanh khi lưu chưa áp vào ô sổ điểm, điểm danh, nhận xét chấm. `grep -rn ep-saved frontend/src`: chỉ có định nghĩa ở `base.css:59-60`, không thành phần nào dùng. Dev đã tự khai (dự kiến đợt 4) |
| UX04-15 | 8 | PASS | `package.json` không đổi |
| UX04-16 | 8 | **FAIL (Thấp)** | `ChatScreen.module.css:47 @keyframes blink { to { visibility: hidden } }` đổi `visibility`, không thuộc ba ngoại lệ (spinner, ba chấm, shimmer). `ep-saved` đổi `background-color` (do AC7 yêu cầu, xem P-91-7). Không có stagger ngoài ngoại lệ |
| UX04-17 | 8 | PASS | Tải `/` và `/inbox`: không có `animation-delay` > 0 |
| UX04-18 | 9 | PASS | `.ep-motion-safe` ở `base.css`, luật `reduce` giữ lại |
| UX04-20 | 10 | PASS | `J`, `K`, `⌘K` không hoạt cảnh |
| UX04-21 | 11 | PASS | Tiến độ 150 → 100 %, −10 → 0 %, `NaN` → 0 % |
| UX04-22 | 12 | PASS | `visual.spec.ts` 14 ảnh pass; chỉ `inbox-1440`, `threads-390` đổi |
| UX04-PQ | — | PASS | SV `/gradebook` → 403 |

### US-UX91-21 — Cờ đã đăng nhập: FAIL
| TC | AC | Kết quả | Ghi chú |
| --- | --- | --- | --- |
| UX21-1, 2 | 1 | PASS | Khách (không cờ): `/` không gọi API, không 401 console (1440 và 375) |
| UX21-3 | 2 | PASS | Có cờ: gọi `refresh`, hiện Hôm nay, không nháy trang giới thiệu |
| UX21-4, 5 | 3 | **FAIL (Trung bình)** | Mất cờ nhưng còn cookie phiên: `/` hiện trang giới thiệu (đúng). Bấm `Đăng nhập` → `/login` hiện **form**, **không gọi `/auth/refresh`**, không vào ứng dụng (0 request API trong 4 s). AC3: «vẫn thử làm mới bằng cookie, nếu phiên còn thì vào thẳng ứng dụng». Tái hiện: harness, `asDemo(teacher)`, xoá `ep_returning`, mở `/`, bấm `Đăng nhập`. Mở thẳng `/login` thì có gọi (TC-6 pass), nên lỗi nằm ở chuyển trang phía client |
| UX21-6, 10 | 3, 6 | PASS | `/login`, `/register` không cookie: vẫn gọi `refresh`, hiện form |
| UX21-7, 8 | 4 | PASS | `localStorage` ném lỗi / bị chặn: `/` không lỗi, hiện trang giới thiệu |
| UX21-9 | 5 | PASS | Khách mở route sâu → `/login?next=` |
| UX21-11 | 6 | PASS | `landing.spec.ts` cập nhật theo AC2, không bỏ dòng kiểm «không gọi API» |
| UX21-PQ | — | PASS | SV đăng nhập, `/` → Hôm nay SV (cùng ca UX08-52) |

## Sửa TC (có ghi lý do, theo P-91-7)
- TC-UX02-2 (I5): ví dụ AC2 «Thứ Hai, 21/09/2025» sai thứ; 21/09/2025 là Chủ Nhật. TC dùng 22/09/2025 (thứ Hai). Mã đúng.
- TC-UX02-4: thư viện của SV dùng «chỉ ngày» (AC2 cho phép), TC trước ghi cả giờ.
- TC-UX03-6/7/8/10: đọc lý do qua `aria-describedby` (trên `/grading` có ba phần tử `role=tooltip` cùng lúc trong DOM).
- TC-UX04-1/2: chỉ tính biến thể `Button`. TC-UX04-10: `transition-property: all` kèm `0s` là không có hoạt cảnh.

## File
Spec: `frontend/e2e/qc-uiux91-w1.spec.ts`, `qc-uiux91-w12b-real.spec.ts`. Config: `frontend/playwright.qc-real.config.ts`. Script: `docs/sprints/9.1/qc/scripts/hover-check.mjs`. Ảnh: `docs/sprints/9.1/qc/shots/wave-2/`.

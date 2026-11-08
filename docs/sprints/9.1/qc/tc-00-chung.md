# TC FEAT-uiux-91 (sprint 9.1 pha 2) — quy ước chung

Phiên bản 1 · 2026-11-08 · Nguồn duy nhất: `docs/specs/FEAT-uiux-91/US.md` (APPROVED, `0147eb69`), `SRS.md`, `QUESTIONS.md`. Hộp đen: TC viết từ AC, không đọc mã của `dev`. **US-UX91-22 (Dùng thử) chưa có TC** (spec còn DRAFT, chờ TL-3 / Q-UX91-13).

## Tệp
| Tệp | Đợt | Story |
| --- | --- | --- |
| `tc-wave-1.md` | 1 Nền chung | US-02, 03, 04, 21 |
| `tc-wave-2.md` | 2 Chữ | US-01, 05, 06, 07 |
| `tc-wave-6.md` | 6 Trang giới thiệu v2 | US-08, 09 |
| `tc-wave-3.md` | 3 Hôm nay | US-10, 11 |
| `tc-wave-4.md` | 4 Chấm bài / Sổ điểm / Điểm danh | US-14, 15, 16, 12 |
| `tc-wave-5.md` | 5 Chat / Hỏi đáp / Me / Tài liệu / Insights | US-13, 19, 20, 18, 17 |
| `questions.md` | — | chỗ AC mơ hồ, đã báo PM |

(Mỗi đợt một tệp thay vì một tệp mỗi story để giữ chung tiền điều kiện và kiểm chéo; mỗi story vẫn là một mục riêng, mỗi AC ít nhất một TC, bảng phủ AC ở cuối tệp.)

## Mẫu một TC
`TC-UX<story>-<số>` → **AC** → vai / cỡ → tiền điều kiện → bước → kết quả mong đợi → **cách chạy**.

Cách chạy:
| Mã | Nghĩa | Chạy ở đâu |
| --- | --- | --- |
| `E2E` | spec Playwright mới `frontend/e2e/qc-uiux91-<đợt>.spec.ts` (QC viết ở pha 2, không sửa spec có sẵn trừ khi AC bảo "sửa theo spec") | VPS `/srv/work/qc`, `pnpm -C frontend exec playwright test qc-uiux91-<đợt>` |
| `MAC` | thao tác / quan sát bằng `playwright-cli` (trình duyệt trên Mac) trỏ vào stack `-p ep-qc` qua `ssh -L` | Mac |
| `GREP` | lệnh `grep` / `rg` trên cây mã | VPS |
| `UNIT` | chạy test đơn vị có sẵn / yêu cầu bởi AC | VPS |
| `LH` | Lighthouse mobile + desktop, ≥ 3 lượt, throttle thật | VPS |
| `FFP` | `ffprobe` / `ffmpeg` / `curl -I` | VPS |
| `TAY` | QC xem tay (video, ảnh, cảm nhận), ghi giây / ảnh | Mac |

## Môi trường kiểm
- VPS `root@152.53.64.175`, thư mục `/srv/work/qc` (clone `kagamikuro1024/EduPilot-v2`, nhánh `sprint/9.1-uiux` ở commit dev báo trong handoff), compose project `-p ep-qc`, `EP_PORT_OFFSET` riêng, chỉ bind `127.0.0.1`. Xong lượt thì `docker compose -p ep-qc down -v`. **Không** động `/opt/edupilot` (demo) và `9router`.
- Seed: `pnpm dev` / `deploy/vps-seed.sh` kiểu stack test (7 tài khoản `@edupilot.local`: `admin`, `teacher`, `ta`, `sv.gioi`, `sv.kha`, `sv.nguyco`, `sv.moi`; mật khẩu = `SEED_DEFAULT_PASSWORD` của `.env` stack test); lớp 1 mã `761987` (đầy dữ liệu, 30 SV), lớp 2 `761988` (24 SV, ít dữ liệu, chờ duyệt 3). LLM = provider giả (không tốn hạn mức 9router).
- Trình duyệt Mac: `playwright-cli open`, tunnel `ssh -L <cổng Mac>:127.0.0.1:<cổng stack> root@152.53.64.175`.
- Bề rộng: **1440 × 900**, **1100 × 800**, **375 × 812**. Múi giờ trình duyệt mặc định `Asia/Ho_Chi_Minh`, TC nào ghi khác thì đặt `timezoneId`.
- `reduce` = `prefers-reduced-motion: reduce` (`page.emulateMedia({ reducedMotion: 'reduce' })` / `playwright-cli set-reduced-motion reduce`).

## Kiểm chung cho mọi TC giao diện (không lặp ở từng TC)
- **K1 axe:** axe-core 4.11 (`wcag2a, wcag2aa, wcag21aa, wcag22aa`) trên route đổi, 1440 và 375: 0 lỗi `serious` / `critical`.
- **K2 antipattern:** `bash scripts/ui-antipatterns.sh` thoát 0 (và `--selftest` đủ phép).
- **K3 console:** không `pageerror`, không `console.error` ngoài danh sách đã biết (401 `/auth/refresh` ở route công khai chỉ được phép **trước** US-21; sau US-21 là lỗi).
- **K4 ngang 375:** `document.documentElement.scrollWidth <= clientWidth` ở 375 px; vùng chạm điều khiển chính ≥ 44 px.
- **K5 chữ:** tiếng Việt; sinh viên không thấy RAG / PII / provider / fallback / trace / confidence / token / LLM / embedding / prompt / model (DESIGN §13).
- **K6 reduced-motion:** nội dung hiện đủ ở trạng thái cuối; không `transform` / trượt (chỉ `opacity` / màu ≤ 100–200 ms nơi AC cho phép).
- **K7 ảnh trước / sau:** chụp trước (demo `v2202609419695521098.happysrv.de` trước deploy hoặc ảnh `audit/shots/`) và sau (stack `ep-qc`) ở 1440 và 375, đặt `docs/sprints/9.1/qc/shots/wave-<n>/`.
- **K8 bộ test sẵn có:** `pnpm -C frontend lint && build`; `pnpm -C frontend exec playwright test` toàn bộ; `go test` không thay đổi vì sprint chỉ giao diện (ghi PASS nếu diff không đụng `backend-go`; nếu có đụng thì chạy `scripts/go-test-all.sh`).

## Kiểm chéo bắt buộc theo QC.md (áp cho mỗi đợt; TC chi tiết nằm trong từng tệp đợt)
| Hạng mục | Cách xử lý ở sprint này |
| --- | --- |
| Phân quyền (vai sai / SV ngoài lớp) | TC `…-PQ` ở mỗi story có route: STUDENT mở route giảng viên → 403 màn hiện có; TA; admin; SV lớp khác không thấy dữ liệu lớp 761987 |
| PII trong log / payload LLM | story đụng chat / thread / tài liệu (US-13, 19, 20, 18): quan sát request, `docker compose -p ep-qc logs gateway` không chứa email / MSSV / tên SV mô phỏng kèm điểm; không `console.log` PII ở frontend |
| Idempotency POST có tác dụng phụ | story đụng điểm danh, chat, duyệt, chốt, công bố (US-12, 13, 15, 16): gửi lặp cùng `Idempotency-Key` / bấm đúp → một hiệu lực |
| Phân trang danh sách | US-19, 14, 18, 16: danh sách dùng cursor, không trả toàn bộ; ghi quan sát `limit` / `cursor` ở request |
| Trạng thái tải / rỗng / lỗi / 375 | mỗi route đổi: chặn API chậm (tải), lớp rỗng, huỷ API (lỗi), 375 px |
| Migration sạch trên DB trống | sprint không đổi schema; kiểm `git diff --stat` không có `backend-go/db/migrations` mới; nếu có thì `goose up` trên DB trống |
| Đọc diff (`git diff 1b72f54...`) | tìm: `fetch` trần, thư viện lạ trong `package.json`, sửa test / nới assertion, secret, `Demo*.tsx` bị sửa (cấm theo US-01), `transition: width` / `margin-left` còn lại |

## Quy ước kết luận
PASS / FAIL theo AC; một AC FAIL → story FAIL. Chạy không được → `KHÔNG KIỂM ĐƯỢC` (tính FAIL). TC chỉ sửa khi spec đổi, ghi lý do + ngày + số proposal. Báo cáo đợt: `docs/sprints/9.1/qc/wave-<n>.md`.

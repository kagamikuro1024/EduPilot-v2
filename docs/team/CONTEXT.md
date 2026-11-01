# Bối cảnh dự án — đọc trước mọi việc

Mọi agent (`pm`, `ba`, `dev`, `qc`) đọc file này ở đầu mỗi phiên và mỗi khi PM giao việc mới. Nó trả lời: mình đang làm cho ai, cho cái gì, đang ở đâu, cái gì đã chốt. Chi tiết nằm ở file được trỏ tới; đừng đoán.

## 1. Sản phẩm trong một đoạn
**EduPilot v2** — nền tảng vận hành lớp học có AI cho một học phần đại học (đồ án tốt nghiệp của chủ dự án, làm một mình cùng đội agent). Sinh viên hỏi riêng / hỏi công khai → AI trả lời hoặc chuyển giảng viên → bài nộp được AI chấm nháp → giảng viên duyệt, công bố → điểm danh, điểm cộng, điểm thành phần gộp thành điểm cuối kỳ theo quy chế → giảng viên thấy lớp yếu ở đâu. Ba điều làm nó khác: **AI chỉ nháp, người quyết**; **che danh tính hai chiều** quanh mọi lời gọi LLM + tường lửa chặn câu hỏi riêng tư lọt ra kênh công khai; **điểm tính bằng code thuần** từ quy chế, không LLM.

Người dùng: sinh viên (chủ yếu điện thoại), trợ giảng và giảng viên (desktop, điểm danh trên điện thoại), admin. Sự thật sản phẩm cho thiết kế: `frontend/PRODUCT.md`. Yêu cầu đầy đủ: `docs/PRD.md` (M0–M14), luồng: `docs/FLOWS.md` (F1–F18).

## 2. Đang ở đâu
- Repo: `github.com/kagamikuro1024/EduPilot-v2` (public, 2026-10-09). Repo cũ `TA_Agent_v2` đã bỏ: **không push lên đó, không push nhánh có lịch sử cũ** (lịch sử cũ chứa secret của Project III).
- Lộ trình 11 sprint + 1.5 + 5.5: `docs/sprints/ROADMAP.md`. Tiến độ, nợ: `docs/PROGRESS.md`.
- **Việc hiện tại:** `main` = sprint 1 → 9 (merge 2026-11-01). Đang làm OPS-VPS (`docs/sprints/ops-vps/plan.md`, nhánh `ops/vps-deploy`): demo trên VPS + đo LLM thật. Sprint 10 (P9) chưa lập kế hoạch. Worktree: dev `../TA_Agent_v2-s7`, PM `../TA_Agent_v2-s6` (detached), BA `/tmp/ba-wt`, QC `../TA_Agent_qc5`; không làm trong worktree của vai khác.
- Prototype sprint 1.5 là MOCK (D51): màn mock được thay dần bằng màn thật; spec thật ở `docs/specs/<FEAT>/` thắng spec prototype `docs/sprints/1.5/spec/`. Sprint 7 thay mock của `/inbox`, `/attendance`, `/students`, `/students/[id]`, `/me`.

## 3. Đã chốt — không mở lại
| Mã | Chốt |
| --- | --- |
| D45 | Viết mới toàn bộ; `legacy/` chỉ để đọc tham khảo, không import, không sửa |
| D46 | Chỉ Go (gateway + worker làm cả AI), không service Python |
| D47 | Luật tốc độ hỏi–đáp: một lời gọi LLM sinh chữ / câu hỏi, có stream |
| D48 | Go 1.27, Node 24, Postgres 18 + pgvector, Redis 8, Next.js 16 + React 19, pnpm, Mailpit |
| D28 | Hệ thiết kế *Red Thread / Academic Instrument* (`docs/design/DESIGN.md`) là thẩm quyền về diện mạo và bố cục |
| D44 | Dữ liệu demo mô phỏng hoàn toàn; không nạp dữ liệu sinh viên thật |
Danh sách đầy đủ: `docs/DECISIONS.md`.

## 4. Luật hay bị quên nhất
- Đỏ là tín hiệu (đang ở đâu / cần làm / đã sửa-xác nhận), không phải nền. Không thẻ KPI, không card lồng card, không gamify.
- Sinh viên không bao giờ thấy: từ kỹ thuật AI (RAG, PII, provider, fallback, trace, confidence), điểm nháp, ghi chú quan sát, nhãn rủi ro của chính mình. Giảng viên không đọc chat riêng của sinh viên khi chưa escalate.
- Chỉ TEACHER xác nhận công thức, công bố, chốt điểm. AI không tự công bố.
- Chỉ token `--ep-*` + primitive ở `frontend/src/shared/`; `bash scripts/ui-antipatterns.sh` phải sạch.
- Không thoả hiệp ngang hàng; góp ý qua `docs/sprints/N/proposals.md`, PM quyết. Ngoại lệ: dev hỏi Tech Lead (`research`) câu hỏi kỹ thuật qua `docs/sprints/N/techlead.md` (`docs/team/RESEARCH.md`).
- Quyển đồ án do `writer` viết ở `~/Documents/EduPilot-thesis/` (ngoài repo, `docs/team/WRITER.md`). Đó là văn bản nộp trường nên mọi số liệu trong `report.md`, QC report, `thesis-notes` phải đúng và có nguồn.
- Không mở subagent (task/agent con) trừ khi PM cho phép: mỗi subagent đọc lại bối cảnh + spec từ đầu, tốn token gấp nhiều lần. Làm tuần tự trong phiên của mình.
- **Mọi build và mọi test chạy trên VPS; máy Mac của chủ dự án chỉ để sửa mã + git** (chủ dự án yêu cầu 2026-11-01; Docker / Colima, `node_modules`, cache Go trên Mac đã bị xoá):
  - Gồm `go vet` / `golangci-lint` / `go test` (`scripts/go-test-all.sh`), `sqlc`, `pnpm lint` / `build`, Playwright, k6 / tải, Lighthouse, p95, test 1.000 sinh viên, kiểm cổng mạng, mọi `docker build` / `docker compose`. Không chạy `colima start`, không `pnpm install` trên Mac.
  - Cách làm: commit + push nhánh của bạn, rồi trên VPS dùng **thư mục làm việc riêng của vai** `/srv/work/<vai>` (`git clone`/`fetch` từ GitHub, checkout nhánh), chạy lệnh ở đó. Không làm trong thư mục của vai khác và không làm ở `/opt/edupilot` (stack demo).
  - Docker trên VPS: stack test của mỗi vai dùng compose project riêng `-p ep-<vai>` + cổng lệch (`EP_PORT_OFFSET`), chỉ bind `127.0.0.1`. Xong lượt thì `docker compose -p ep-<vai> down -v`. Muốn xem giao diện từ Mac thì dùng SSH tunnel (`ssh -L`).
  - VPS: `ssh root@152.53.64.175` (khoá), netcup 12 vCPU / 32 GB amd64, Debian 13. Go, Node / pnpm, k6, Playwright cài trên host hoặc chạy trong container (dev dựng lần đầu, ghi ở `deploy/README.md`).
  - **Stack demo** ở `/opt/edupilot` (cách dựng ở `deploy/README.md`) là bản chủ dự án demo: **không reset DB, không `down -v`**; chỉ dev / PM cập nhật nó.
  - **Không đụng** `9router` (container, `/root/9router`, cron) và không in / commit khoá của nó. LLM thật qua 9router chỉ dùng khi việc cần (đo LLM), tôn trọng hạn mức.
  - Một lượt đo nặng tại một thời điểm; ghi số đo kèm "trên VPS 12 vCPU / 32 GB" trong report.

## 5. Mỗi vai đọc gì cho việc hiện tại
| Vai | Đọc |
| --- | --- |
| ba | `docs/sprints/8/plan.md`, `docs/phases/P6.md`, `docs/PRD.md` M7, `docs/FLOWS.md` F10, `docs/DEMO_SCRIPT.md` bước 6, `docs/design/DESIGN.md` (§13 lời văn, §14.11 sổ điểm / công thức), `docs/UX.md` |
| dev | spec `docs/specs/<FEAT>/`, `docs/design/DESIGN.md`, `docs/UX.md`, `docs/ARCHITECTURE.md`, `docs/SYSTEM_DESIGN.md`, nền ở `frontend/src/shared/` |
| qc | spec `docs/specs/<FEAT>/`, `docs/design/DESIGN.md` §21–§22, `docs/UX.md` mục 6, `docs/FLOWS.md` (nhánh lỗi) |

## 6. Từ điển nhanh
Hôm nay = trang `/` theo vai · Hộp thư hỗ trợ = ticket escalation · Threads = hỏi đáp công khai, giảng viên Xác nhận / Loại câu trả lời AI · Hồ sơ 360 = trang một sinh viên · Sổ điểm / Công thức điểm = gradebook / scheme trích từ quy chế · Duyệt bài = xem AI chấm nháp rồi quyết · Insights = lỗ hổng kiến thức của lớp · Quan sát AI = observability (chỉ GV/Admin).

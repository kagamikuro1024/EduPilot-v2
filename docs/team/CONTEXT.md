# Bối cảnh dự án — đọc trước mọi việc

Mọi agent (`pm`, `ba`, `dev`, `qc`) đọc file này ở đầu mỗi phiên và mỗi khi PM giao việc mới. Nó trả lời: mình đang làm cho ai, cho cái gì, đang ở đâu, cái gì đã chốt. Chi tiết nằm ở file được trỏ tới; đừng đoán.

## 1. Sản phẩm trong một đoạn
**EduPilot v2** — nền tảng vận hành lớp học có AI cho một học phần đại học (đồ án tốt nghiệp của chủ dự án, làm một mình cùng đội agent). Sinh viên hỏi riêng / hỏi công khai → AI trả lời hoặc chuyển giảng viên → bài nộp được AI chấm nháp → giảng viên duyệt, công bố → điểm danh, điểm cộng, điểm thành phần gộp thành điểm cuối kỳ theo quy chế → giảng viên thấy lớp yếu ở đâu. Ba điều làm nó khác: **AI chỉ nháp, người quyết**; **che danh tính hai chiều** quanh mọi lời gọi LLM + tường lửa chặn câu hỏi riêng tư lọt ra kênh công khai; **điểm tính bằng code thuần** từ quy chế, không LLM.

Người dùng: sinh viên (chủ yếu điện thoại), trợ giảng và giảng viên (desktop, điểm danh trên điện thoại), admin. Sự thật sản phẩm cho thiết kế: `frontend/PRODUCT.md`. Yêu cầu đầy đủ: `docs/PRD.md` (M0–M14), luồng: `docs/FLOWS.md` (F1–F18).

## 2. Đang ở đâu
- Repo: `github.com/kagamikuro1024/TA_Agent_v2`. `main` = sprint 1 (P0: mã cũ dời vào `legacy/`, khung Go + Next.js, stack local, CI).
- Lộ trình 10 sprint: `docs/sprints/ROADMAP.md`. Tiến độ, nợ: `docs/PROGRESS.md`.
- **Việc hiện tại — SPRINT 1.5: PROTOTYPE GIAO DIỆN** (`docs/sprints/1.5/plan.md`, nhánh `sprint/1.5-mock-ui`). Chủ dự án thuyết trình với thầy hướng dẫn **cuối tuần này**: cần một prototype bấm được, đẹp, phủ TOÀN BỘ tính năng dự định, dữ liệu mô phỏng, đổi được vai trò, đi trọn `docs/DEMO_SCRIPT.md`. Không backend; nhưng primitive dựng đúng chuẩn ở `frontend/src/shared/` để phase PU dùng lại.
- **Prototype là MOCK có hạn dùng (D51):** spec của nó ở `docs/sprints/1.5/spec/`, KHÔNG phải `docs/specs/`. Từ sprint 2, mỗi sprint build thật thay dần màn mock; spec thật viết mới ở `docs/specs/<FEAT>/` và thắng spec prototype. Đừng coi spec prototype là yêu cầu thật khi build thật — đọc nó chỉ để biết màn đã dựng trông thế nào.

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
- Không mở subagent (task/agent con) trừ khi PM cho phép: mỗi subagent đọc lại bối cảnh + spec từ đầu, tốn token gấp nhiều lần. Làm tuần tự trong phiên của mình.
- **Dọn rác Docker định kỳ** (chủ dự án yêu cầu 2026-10-03; ổ máy từng bị colima ăn 57 GB vì volume vô danh của test): mỗi agent chạy Docker, sau mỗi story / lượt test, chạy `docker volume prune -f` (chỉ xoá volume vô danh không container nào dùng, an toàn khi người khác đang test). PM dọn sâu ở mỗi lần đóng sprint và khi chủ dự án bảo dừng: `docker volume prune -f && docker builder prune -af && docker image prune -f && colima ssh -- sudo fstrim -a`. Không dùng `docker system prune -a --volumes`; không xoá volume có tên.

## 5. Mỗi vai đọc gì cho việc hiện tại
| Vai | Đọc |
| --- | --- |
| ba | `docs/design/DESIGN.md` (§1–§2 vai trò + điều hướng, §13 lời văn, §14 hợp đồng 25 route), `docs/design/INTEGRATION.md` mục 2 (route ngoài §14), `docs/PRD.md`, `docs/DEMO_SCRIPT.md` |
| dev | `docs/design/DESIGN.md` toàn bộ, `docs/UX.md`, spec `docs/sprints/1.5/spec/`, hướng thiết kế `frontend/.impeccable/surfaces/frontend-src-app.md`, nền đã có ở `frontend/src/shared/` + `frontend/src/mock/` |
| qc | spec `docs/sprints/1.5/spec/`, `docs/design/DESIGN.md` §21–§22, `docs/UX.md` mục 6, `docs/DEMO_SCRIPT.md` |

## 6. Từ điển nhanh
Hôm nay = trang `/` theo vai · Hộp thư hỗ trợ = ticket escalation · Threads = hỏi đáp công khai, giảng viên Xác nhận / Loại câu trả lời AI · Hồ sơ 360 = trang một sinh viên · Sổ điểm / Công thức điểm = gradebook / scheme trích từ quy chế · Duyệt bài = xem AI chấm nháp rồi quyết · Insights = lỗ hổng kiến thức của lớp · Quan sát AI = observability (chỉ GV/Admin).

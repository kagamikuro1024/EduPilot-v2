# Sprint 1 — P0 Chuẩn bị (phần 1)

Trạng thái: **ĐÃ DUYỆT 2026-10-01** · Nhánh: `sprint/1-p0-prep` · Ngày lập: 2026-10-01

## Mục tiêu
Dựng lưới an toàn trước khi port: có kịch bản demo làm kim chỉ nam, CI chạy được, MailHog trong stack local, và hợp đồng API của gateway Java được ghi thành `openapi.yaml`. Không viết dòng Go nào, không đổi hành vi hệ thống (P0 "Ngoài phạm vi").

## Story (theo thứ tự thi công)

| # | Story | Feature | Truy vết | Ước lượng | Ai |
| --- | --- | --- | --- | --- | --- |
| 1 | US-P0-01 | FEAT-demo-script | PRD (toàn bộ) → FLOWS F2, F3, F5, F7, F9, F10, F17 → P0 L0 | S (0,5 ngày) | ba viết, pm kiểm |
| 2 | US-P0-02 | FEAT-ci-mailhog | ARCHITECTURE (env, compose) → — → P0 L1 | S (0,5 ngày) | dev |
| 3 | US-P0-03 | FEAT-api-contract | ARCHITECTURE (API) → mọi luồng dùng gateway → P0 L2 | M (1–1,5 ngày) | dev |

Tổng ≈ 2–2,5 ngày công.

### US-P0-01 — Chủ dự án muốn có kịch bản demo 15 phút để mọi phase sau biết mình đang làm cho bước nào
- AC tóm tắt:
  - `docs/DEMO_SCRIPT.md` đi qua F2 → F3 → F5 → F7 → F9 → F10 → F17 trên 2 lớp seed; mỗi bước ghi: vai trò, màn hình (route), thao tác, kết quả phải thấy, phase làm cho bước đó chạy thật.
  - Tổng thời lượng ≤ 15 phút, có mốc thời gian từng bước.
  - Mục dự phòng: `DEMO_MODE=true` dùng provider `fake` trả lời ghi sẵn cho đúng các câu trong kịch bản; video quay sẵn; ai bật, bật ở đâu.
  - Không dùng từ kỹ thuật AI trên lời thoại phía sinh viên (DESIGN.md §13).
- Lát dọc: chỉ tài liệu. Không có code.
- Phụ thuộc: không.
- Rủi ro: kịch bản viết cho tính năng chưa có → phải bám đúng route/AC trong DESIGN §14 và PRD, không bịa màn hình.

### US-P0-02 — Lập trình viên muốn mỗi lần push đều chạy test và mail đi được trong môi trường local
- AC tóm tắt:
  - AC1. Given push lên nhánh bất kỳ When GitHub Actions chạy `.github/workflows/ci.yml` Then ba job `pytest -q`, `./gradlew test` (thư mục `backend-java/aitrogiang`, thay cho `mvn -q test` — xem "Quyết định PM"), `npm ci && npm run lint && npm run build` (thư mục `frontend`) đều chạy, không gọi LLM thật.
  - AC2. Given `pnpm dev` When stack dựng xong Then có service `mailhog` (SMTP 1025, UI 8025), `pnpm dev:status` thấy mọi container healthy.
  - AC3 (nhánh lỗi). Given một test Python đỏ When CI chạy Then workflow đỏ và job tương ứng báo lỗi (không `continue-on-error`).
- Lát dọc: `docker-compose.local.yml` → `scripts/dev.mjs` (nếu cần) → `ci.yml` → chạy thử.
- Phụ thuộc: không.
- Rủi ro: xem mục "Chặn" bên dưới (máy chưa có docker / pnpm / mvn / gh).

### US-P0-03 — Lập trình viên muốn có đặc tả hợp đồng API của gateway Java để port sang Go không lệch
- AC tóm tắt:
  - AC1. `backend-go/api/openapi.yaml` (OpenAPI 3.1, hợp lệ theo validator) phủ mọi endpoint của 13 controller không phải test trong `backend-java/**/controller/`: đường dẫn, method, vai trò được phép, request, response, mã lỗi.
  - AC2. Mỗi hàm trong `frontend/src/services/*.ts` gọi gateway đều có endpoint tương ứng trong openapi (bảng đối chiếu trong handoff).
  - AC3. Định dạng SSE của chat stream được ghi riêng: tên sự kiện, payload từng loại, sự kiện kết thúc, sự kiện lỗi.
  - AC4. Claims JWT, thời hạn, thuật toán ký, định dạng hash mật khẩu được ghi lại (không chép secret).
  - AC5 (nhánh lỗi). Mỗi endpoint có ít nhất response 401/403 (nếu có bảo vệ) và mã lỗi validate đúng như Java đang trả.
- Lát dọc: đọc Java → viết yaml → validate → đối chiếu frontend.
- Phụ thuộc: không. Là đầu vào của US-P0-04 (golden) ở sprint 2.
- Rủi ro: P0.md ghi "16 controller", repo thực tế có 13 controller + 4 file test (17 file). Dev đếm lại và ghi con số thật vào handoff.

## Ngoài sprint này (sprint 2, cùng phase P0)
- US-P0-04 Golden response (L3): `scripts/record-golden.mjs`, `backend-go/testdata/golden/`, `ignore.json`, phiên SSE mẫu với provider giả. Cần US-P0-03 và stack chạy được.
- US-P0-05 Mốc so sánh (L4): k6 `smoke.js` → `load-baseline.json`; RAG baseline → `baseline.json`; `port.md` cột Java.
- `/gate P0` chạy cuối sprint 2.

Lý do tách: P0 đủ 5 lát ≈ 3,5–4 ngày công > trần 3 ngày/sprint. Lịch WORKFLOW §6 cho P0 0,5 tuần, nên P0 sẽ trễ ≈ 1–2 ngày so với lịch nếu cả hai sprint chạy liền.

## Quyết định PM tự chốt
- Nhánh tạo từ `chore/edupilot-v2-docs` (HEAD `1b72f54`), **không phải `main`**: `main` chưa có bộ tài liệu v2 (CLAUDE/AGENTS, docs/phases, docs/team), agent không có gì để đọc. Thay đổi chưa commit của bạn ở `scripts/team-up.sh` đi theo nhánh, PM không đụng.
- Đặt tên nhánh theo `docs/team/README.md` (`sprint/N-…`), không theo `feat/p0-prep` trong P0.md.
- US-P0-01 làm song song với US-P0-02 vì không chung file; US-P0-02 và US-P0-03 vẫn tuần tự (một story một lần cho dev).
- Không spec SRS đầy đủ cho US-P0-02/03 (hạ tầng, không có màn hình): `ba` viết `US.md` + `SRS.md` rút gọn (mục 1, 4, 6, 8, 9, 11).
- Sửa lệch giữa P0.md / CLAUDE.md và repo thật (không đổi hành vi hệ thống, chỉ chọn lệnh đúng):
  - Gateway Java build bằng **Gradle** (`backend-java/aitrogiang/gradlew`), không có `pom.xml` → CI chạy `./gradlew test` thay cho `mvn -q test`.
  - Frontend khoá phụ thuộc bằng **npm** (`frontend/package-lock.json`, không có `pnpm-lock.yaml`) → CI dùng `npm ci` + `npm run lint` + `npm run build` trong `frontend/`. Không đổi package manager trong sprint này (ngoài phạm vi P0); ghi vào Nợ.
  - Phiên bản theo Dockerfile: Node 20, JDK 21 (temurin), Python 3.11.

## Chặn → đã gỡ
PM đã cài Homebrew + `node@20 pnpm colima docker docker-compose docker-buildx openjdk@21 gh k6 cloc go python@3.11`. Shell mới cần `eval "$(/opt/homebrew/bin/brew shellenv)"` (đã thêm vào `~/.zprofile`). Docker chạy qua `colima start`.

## Trả lời của chủ dự án (2026-10-01)
1. Duyệt kế hoạch: **có** ("gogogogo").
2. `docs/DEMO_SCRIPT.md`: chủ dự án không nêu → áp đề xuất PM: `ba` ghi.
3. Cài công cụ: chủ dự án cho toàn quyền, `dev` được tự cài thêm.
4. Push lên `origin`: được phép (toàn quyền). `gh auth login` cần trình duyệt của chủ dự án nếu chưa đăng nhập.

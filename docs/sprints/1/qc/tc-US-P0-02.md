# QC test case — US-P0-02
Nguồn: `docs/specs/FEAT-scaffold/US.md` + `SRS.md`. Viết trước khi có code, không đọc code của dev. Mốc: `5cfb4af` (trước khi dời), base sprint `1b72f54`.

**Chạy:** `bash docs/sprints/1/qc/scripts/tc-US-P0-02.sh TC-01 TC-02 …` (mỗi TC in `PASS/FAIL <id>: <ý>`; mã thoát 1 nếu có FAIL). Mọi lệnh ở cột "Bước" là tên TC cho script; lệnh gốc của AC nằm trong script, copy-paste được.
Cần: `git jq curl go node pnpm docker python3`. Tên `$C` = `docker compose --env-file .env.local -f docker-compose.local.yml -p edupilot`.

**Thứ tự chạy:** (1) tĩnh, không Docker: 01–09, 19–21, 22–24, 25, 26, 35–38 · (2) cổng 8080 trống: 27 · (3) Docker: 10 → 11–15 → 31–34 → 28 → 29 → 30; 16, 17, 18 chạy riêng (xoá `.env.local` / chiếm cổng / giả mất Docker — script tự dọn). TC-09 cần `.env.local`.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-01 | AC1 | HEAD có handoff | `…sh TC-01` (`git ls-files \| awk -F/ … \| sort -u`) | Đúng 18 mục gốc: `.claude .dockerignore .env.example .gitattributes .github .gitignore AGENTS.md CLAUDE.md backend-go docker-compose.local.yml docs frontend legacy package.json pnpm-lock.yaml pnpm-workspace.yaml scripts seed` |
| TC-02 | AC1 | – | `TC-02` (vòng `git ls-tree -r $BASE` → `git ls-files --error-unmatch legacy/$f`) | Không in `THIẾU legacy/…` |
| TC-03 | AC1 (Q3) | – | `TC-03` (`git ls-files \| grep -c 492218d5`) | `0`, kể cả dưới `legacy/` |
| TC-04 | AC1 | – | `TC-04` | `seed/documents` = đúng 3 PDF; `git ls-files data` = 0; 3 PDF cùng blob với `$BASE:data/*.pdf` (không hỏng khi dời) |
| TC-05 | AC1 | – | `TC-05` | 4 file (`tokens.css` + 3 brand) còn đúng chỗ và **không đổi** so với base |
| TC-06 | AC1 | – | `TC-06` (`git log --follow --oneline -- legacy/backend-java/aitrogiang/build.gradle`) | ≥ 2 commit — lịch sử được giữ |
| TC-07 | AC1 + chéo `legacy/` | – | `TC-07` (`git diff -M -C --find-copies-harder --name-status $BASE HEAD`, lọc đích `legacy/`) | Mọi file đích `legacy/` là `R100`/`C100`: nội dung không bị sửa (mã cũ không bị đụng tay) |
| TC-08 | AC1 (FR-1) | – | `TC-08` (duyệt `git rev-list $SBASE..HEAD`) | Có commit dời; commit đó chỉ gồm rename (+ `D data/tmp/492218d5…`), không lẫn mã mới; mã mới (`backend-go/`, `frontend/package.json`, `frontend/src/app/`) đến **sau** commit dời |
| TC-09 | AC2 (FR-8) | `.env.local` có | `TC-09` (`$C config --format json \| jq`) | Đúng 6 service `frontend gateway mailpit minio postgres redis`; mọi service có `healthcheck`; image có tag, không `:latest`; `pgvector/pgvector:pg18`, `redis:8`, `axllent/mailpit:*`; cổng host 3000/8080/1025/8025/9000/9001/5433/6380, trong 5432/6379; gateway `depends_on` postgres+redis `service_healthy`; ≥ 3 volume có tên; `config --services \| grep -ciE 'python\|ai$'` = 0 |
| TC-10 | AC2 | Docker chạy, cổng 3000/8080/1025/8025/9000/9001/5433/6380 trống, stack dừng | `TC-10` (`pnpm dev; echo exit=$?`) | `exit=0`, chờ tới khi healthy (không thoát sớm) |
| TC-11 | AC2 | sau TC-10 | `TC-11` (`pnpm -s dev:status --format '{{.Service}} {{.Health}}'`) | 6 dòng, đều `healthy` |
| TC-12 | AC2/AC7 | sau TC-10 | `TC-12` (`curl -si localhost:8080/healthz`) | `200`, `Content-Type: application/json`, body `{"status":"ok"}` |
| TC-13 | AC2 | sau TC-10 | `TC-13` | Mailpit `/api/v1/info` 200 JSON object; MinIO `/minio/health/live` 200 + console 9001 200; Postgres 18 + `vector` có trong `pg_available_extensions`; `redis-cli ping`=PONG, Redis 8; SMTP 1025 mở |
| TC-14 | AC2 (FR-12) | sau TC-10 | `TC-14` (`curl -s localhost:3000`, tải CSS) + **mắt** theo hướng dẫn script in ra | `lang="vi"`; HTML tham chiếu `logo-edupilot`, logo phục vụ 200; có chữ có dấu; CSS có `--ep-` và `Be Vietnam Pro`. Mắt (chủ dự án): trang trống tiếng Việt, logo, font, màu từ token |
| TC-15 | AC2 (FR-11) | sau TC-10 | `TC-15` | `Config.User` của gateway khác root/0; healthcheck không dùng curl/wget; trong image không có curl |
| TC-16 | AC2 (FR-9) | Docker chạy | `TC-16` (xoá `.env.local` → `pnpm dev`; script sao lưu/khôi phục) | exit 0; `.env.local` sinh ra = `.env.example`; `git check-ignore .env.local` đúng |
| TC-17 | AC2 nhánh lỗi (FR-9) | Docker chạy, stack dừng | `TC-17` (chiếm cổng 8080 bằng `python3 -m http.server`, chạy `pnpm dev`) | Thoát mã **≠ 0** (không 0, không treo); log nêu lỗi cổng |
| TC-18 | AC2 nhánh lỗi (FR-9) | – | `TC-18` (PATH chỉ có node/pnpm/git, không docker) | Thoát mã ≠ 0, thông báo nhắc Docker bằng tiếng Việt/Anh đọc được, không stack trace |
| TC-19 | AC2/AC3 phiên bản (D48) | – | `TC-19` | `go 1.27` trong go.mod; next 16, react 19, `packageManager` pnpm, `node:24` ở Dockerfile frontend; không có tailwind/zustand/tanstack (ngoài phạm vi story) |
| TC-20 | AC3 (FR-7) | Go 1.27 | `TC-20` (`go vet ./… && go test -race -count=1 -v ./…`) | Xanh; ≥ 4 PASS (3 ca cấu hình: đủ / thiếu một / thiếu cả hai + `/healthz`); 0 FAIL, 0 SKIP |
| TC-21 | AC3 (FR-6) | mạng | `TC-21` (`pnpm dlx @redocly/cli lint …`, `grep -c '^openapi: 3.1'`, `grep -c '/healthz:'`, bundle → jq) | 0 lỗi lint; `openapi: 3.1`; mô tả `/healthz`; response 200 có schema `status: string` |
| TC-22 | AC3 | Node 24, mạng (font Google) | `TC-22` (`pnpm -C frontend lint && build`) | Cả hai exit 0 |
| TC-23 | AC3 | – | `TC-23` (`bash scripts/ui-antipatterns.sh`) | Sạch, exit 0 |
| TC-24 | AC3 (FR-12) | – | `TC-24` (`grep -nE 'Be_Vietnam_Pro\|tokens\.css' frontend/src/app/layout.tsx`) | Có cả hai; subset `vietnamese`+`latin`; `<html lang="vi">` |
| TC-25 | AC4 | Go build được | `TC-25` (`env -i PATH=$PATH /tmp/gw`) | `exit=1`; log có `DATABASE_URL` **và** `REDIS_URL` trong **cùng một** dòng `"level":"ERROR"` (slog JSON), đúng 1 dòng ERROR; 0 `panic`/`goroutine` |
| TC-26 | AC4 / FR-4 | – | `TC-26` (chỉ đặt một biến, rồi biến kia) | `exit=1`; chỉ nêu biến thiếu, **không** nêu biến đã có; **không** log giá trị biến (`QCSECRET…`); không panic |
| TC-27 | FR-3, FR-4, FR-5, luật 10 | cổng 8080 trống | `TC-27` (đủ 2 biến trỏ DB/Redis không tồn tại; curl; `kill -TERM`) | `/healthz` vẫn `{"status":"ok"}` (không gọi DB/Redis); thư mục chạy không có file mới; SIGTERM → thoát ≤ 5s, không panic |
| TC-28 | AC5 | stack đang chạy | `TC-28` | Trước down: 6 container label `edupilot`; `dev:status` thấy 6; `dev:logs` có đủ 6 service; `pnpm dev:down` exit 0 → `docker ps --filter label=com.docker.compose.project=edupilot -q \| wc -l` = `0` |
| TC-29 | AC5 / FR-5 | stack đang chạy | `TC-29` (`$C stop gateway`, đo giờ + `ExitCode`) | `ExitCode` 0 (137 = bị giết cứng) và dừng < 8s |
| TC-30 | AC5 nhánh lỗi | – | `TC-30` | `dev:down` khi đã dừng vẫn exit 0; volume có tên còn lại sau down (suy ra từ FR-8: không xoá dữ liệu) |
| TC-31 | AC6 (FR-13) | `.env.local` có | `TC-31` (`$C config \| grep -c legacy`; `grep -n legacy pnpm-workspace.yaml package.json backend-go/go.mod`; `grep -cx legacy .dockerignore`) | 0; không in gì; 1. Thêm: `git grep legacy` trong `frontend/`, `backend-go/`, compose, `dev.mjs` = 0 |
| TC-32 | AC6 | stack chạy | `TC-32` | Không mount nào nguồn từ `legacy/`; `go list ./…` không có package `legacy`; lịch sử image gateway không `COPY legacy` |
| TC-33 | AC7 | sau TC-10 | `TC-33` | Body chỉ khoá `status`; `POST /healthz` → 405; đường dẫn lạ → 404 không lộ stack; có/không `Authorization` đều 200 (công khai, đúng spec) |
| TC-34 | AC7 (FR-10) | – | `TC-34` | Không `sk-…`/`AKIA…`; khoá trong `.env.example` chỉ là `DATABASE_URL REDIS_URL BLOB_* SMTP_* MAIL_FROM`; không dùng lại giá trị JWT/secret của `legacy/.env.example`; `.env`/`.env.local` không bị track; mật khẩu dev giả `edupilot-dev` |
| TC-35 | Chéo: lan phạm vi | – | `TC-35` (`git diff -M --name-status 1b72f54 HEAD`) | Mọi đường dẫn đổi nằm trong vùng story; `team-up.sh`, `ui-antipatterns.sh`, `CLAUDE.md`, `AGENTS.md`, `.claude/`, `.gitattributes`, workflow `keep-huggingface…` **không đổi**; liệt kê `docs/` đổi để xem tay |
| TC-36 | Chéo: secret trong diff | – | `TC-36` | Không khớp mẫu secret trong dòng thêm (ngoài `legacy/`, `docs/`) |
| TC-37 | Chéo: thư viện lạ, luật code | – | `TC-37` | In `go.mod` + deps frontend để **đối chiếu tay** với ARCHITECTURE §3 (thư viện ngoài bảng = FAIL); 0 `fetch(` trần; 0 màu cứng ngoài `tokens.css`; 0 ghi đĩa trong `backend-go` (luật 10); 0 `float64` |
| TC-38 | Chéo: không còn Python (D46) | – | `TC-38` | Gốc repo không còn `src/ requirements*.txt Dockerfile Dockerfile.ai shared-proto/ …`; không file `.py`/`requirements*.txt` ngoài `legacy/`, `docs/` |

## Nhánh lỗi (SRS mục 3 của story này ghi "Không áp dụng"; nhánh lỗi nằm ở AC4, AC5)
TC-25, 26 (AC4); TC-28, 29, 30 (AC5); TC-17, 18 (FR-9 `pnpm dev` khác 0 khi không dựng được).

## Phân quyền
AC7 "không áp dụng" — ràng buộc thay thế: TC-33 (body chỉ `status`, không thêm đường dẫn nào trả dữ liệu), TC-34 (không secret). Không có TC vai trò/sinh viên ngoài lớp vì chưa có API nghiệp vụ.

## Kiểm chéo bắt buộc (QC.md mục 3–4) áp dụng được cho story hạ tầng
Phân quyền/PII/idempotency/phân trang/375 px/migration: **không áp dụng** (không API nghiệp vụ, không UI thật, không migration — SRS §5, §7). Áp dụng: TC-07, 08, 35 (lan phạm vi, `legacy/` không bị sửa), TC-36 + TC-34 (secret), TC-37 (thư viện lạ, `fetch` trần, màu cứng, luật 10), TC-31/32 (`legacy/` không bị build).

## Điểm khó kiểm / giả định
- AC2 "bằng mắt" (logo, font, token): TC-14 chỉ kiểm gián tiếp; phần còn lại là việc của chủ dự án.
- AC2 `pnpm dev` khác 0 "nếu có service không healthy": TC-17 dùng cổng bị chiếm vì không ép được service unhealthy mà không sửa code.
- Chuỗi rỗng (`DATABASE_URL=`) và mã thoát SIGTERM: spec chưa nói (proposals #4); TC-27 chấm SIGTERM bằng "thoát trong 5s" ở TC-27 và `ExitCode 0` ở TC-29; chuỗi rỗng **không** có TC.
- `$C ps`/`dev:logs` có thể chạy theo dõi vô hạn: script tự ngắt sau 12s.
- AC3 `redocly` cần mạng và không ghim bản (proposals #3).

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
(chưa có)

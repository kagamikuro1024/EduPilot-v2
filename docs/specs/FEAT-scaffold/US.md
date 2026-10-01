# FEAT-scaffold Dọn mặt bằng + khung Go gateway và Next.js mới
Nguồn: PRD §5 (Triển khai), FLOWS — (hạ tầng, chưa thuộc luồng nào), phase P0 lát L1; quyết định D10, D45, D46, D48; ARCHITECTURE §2 (cấu trúc `backend-go`), §8 (env)

Quy ước trong file: `$C` = `docker compose --env-file .env.local -f docker-compose.local.yml -p edupilot`. Mốc trước khi dời = commit `5cfb4af` (HEAD lúc viết spec; mã Project III chưa đổi từ đó). Bảng dời / giữ đầy đủ: `SRS.md` mục 4.1.

## US-P0-02: Lập trình viên muốn có mặt bằng sạch và khung chạy được của gateway Go + frontend mới để mọi phase sau viết trên nền mới, không dính mã Project III
Ưu tiên: Must · Ước lượng: M · Sprint: 1

### Tiêu chí nghiệm thu
- AC1. Given repo ở `5cfb4af` When dev dời mã theo `SRS.md` mục 4.1 bằng `git mv` Then gốc repo chỉ còn các mục ở cột "Sau" của bảng; mọi file mã Project III có mặt ở `legacy/<đường dẫn cũ>`; 3 PDF môn học ở `seed/documents/`; bản sao `data/tmp/492218d5-….pdf` bị `git rm` (Q3); lịch sử được giữ.
  Kiểm:
  ```bash
  git ls-files | awk -F/ '{print $1}' | sort -u | xargs
  # = .claude .dockerignore .env.example .gitattributes .github .gitignore AGENTS.md CLAUDE.md backend-go docker-compose.local.yml docs frontend legacy package.json pnpm-lock.yaml pnpm-workspace.yaml scripts seed
  git ls-tree -r --name-only 5cfb4af \
    | grep -vE '^(docs/|\.claude/|\.github/|AGENTS\.md$|CLAUDE\.md$|\.gitattributes$|\.gitignore$|\.dockerignore$|package\.json$|pnpm-workspace\.yaml$|scripts/(dev\.mjs|team-up\.sh|ui-antipatterns\.sh)$|frontend/src/shared/styles/tokens\.css$|frontend/public/brand/|data/(Mordern_Network_Security_Threats|QMB12ch6b|Quyche)\.pdf$|data/tmp/)' \
    | while read -r f; do git ls-files --error-unmatch "legacy/$f" >/dev/null 2>&1 || echo "THIẾU legacy/$f"; done   # không in gì
  git ls-files | grep -c '492218d5'   # 0 (không còn ở đâu, kể cả legacy/)
  ls seed/documents   # Mordern_Network_Security_Threats.pdf  QMB12ch6b.pdf  Quyche.pdf
  git ls-files frontend/src/shared/styles/tokens.css frontend/public/brand | wc -l   # 4
  git log --follow --oneline -- legacy/backend-java/aitrogiang/build.gradle | wc -l # ≥ 2 (còn lịch sử trước khi dời)
  ```
- AC2. Given máy có Docker When chạy `pnpm dev` Then lệnh dựng đủ 6 service `postgres`, `redis`, `minio`, `mailpit`, `gateway`, `frontend`, chờ tới khi mọi service healthy rồi thoát mã 0; không có service Python.
  Kiểm:
  ```bash
  pnpm dev; echo "exit=$?"                                         # exit=0
  pnpm -s dev:status --format '{{.Service}} {{.Health}}'           # 6 dòng, đều healthy
  $C config --services | grep -ciE 'python|ai$'                    # 0
  curl -fsS localhost:8080/healthz                                 # {"status":"ok"}
  curl -fsS localhost:8025/api/v1/info                             # 200, JSON thông tin Mailpit
  curl -s localhost:3000 | grep -c 'lang="vi"'                     # ≥ 1
  ```
  Thêm bằng mắt (chủ dự án): `http://localhost:3000` là trang trống tiếng Việt, logo EduPilot, chữ Be Vietnam Pro, màu từ token.
- AC3. Given mã mới When chạy kiểm tĩnh và test Then đều xanh, và `backend-go/api/openapi.yaml` là OpenAPI 3.1 hợp lệ mô tả `GET /healthz`.
  Kiểm:
  ```bash
  (cd backend-go && go vet ./... && go test ./...)
  pnpm exec redocly lint backend-go/api/openapi.yaml               # 0 lỗi; @redocly/cli nằm trong devDependencies gốc, bản ghim theo pnpm-lock.yaml (proposals #3)
  grep -c '^openapi: 3.1' backend-go/api/openapi.yaml && grep -c '/healthz:' backend-go/api/openapi.yaml
  pnpm -C frontend lint && pnpm -C frontend build
  bash scripts/ui-antipatterns.sh                                  # sạch
  grep -nE 'Be_Vietnam_Pro|tokens\.css' frontend/src/app/layout.tsx # có cả hai
  ```
- AC4 (nhánh lỗi — thiếu env, dừng êm). Given `DATABASE_URL` và `REDIS_URL` không đặt, **hoặc đặt nhưng rỗng sau khi bỏ khoảng trắng** When chạy gateway Then gateway thoát mã 1, một dòng log lỗi nêu **đủ tên mọi biến thiếu**, không panic, không stack trace. Given gateway đang chạy đủ env When nhận SIGTERM Then dừng êm, thoát mã 0 (proposals #4).
  Kiểm:
  ```bash
  cd backend-go && go build -o /tmp/gw ./cmd/gateway
  env -i PATH="$PATH" /tmp/gw > /tmp/gw.log 2>&1; echo "exit=$?"   # exit=1
  grep -c DATABASE_URL /tmp/gw.log; grep -c REDIS_URL /tmp/gw.log    # ≥ 1 mỗi biến
  grep -cE 'panic|goroutine' /tmp/gw.log                           # 0
  env -i PATH="$PATH" DATABASE_URL='' REDIS_URL='   ' /tmp/gw > /tmp/gw2.log 2>&1; echo "exit=$?"   # exit=1
  grep -c DATABASE_URL /tmp/gw2.log; grep -c REDIS_URL /tmp/gw2.log  # ≥ 1 mỗi biến
  env -i PATH="$PATH" DATABASE_URL=postgres://x REDIS_URL=redis://x /tmp/gw & pid=$!
  sleep 1; kill -TERM $pid; wait $pid; echo "exit=$?"              # exit=0
  ```
- AC5 (nhánh lỗi — dừng stack). Given stack đang chạy When `pnpm dev:down` Then mọi container của project `edupilot` dừng; `dev:status`, `dev:logs`, `dev:down` thấy đúng các container mà `pnpm dev` dựng.
  Kiểm: `pnpm dev:down && docker ps --filter label=com.docker.compose.project=edupilot -q | wc -l` → `0`.
- AC6. Given `legacy/` When dựng stack, cài phụ thuộc, build Then không lệnh nào đọc hay build gì trong `legacy/`.
  Kiểm:
  ```bash
  $C config | grep -c legacy                                       # 0
  grep -n legacy pnpm-workspace.yaml package.json backend-go/go.mod # không in gì
  grep -cx 'legacy' .dockerignore                                  # 1
  ```
- AC7 (phân quyền). **Không áp dụng — chưa có API nghiệp vụ.** Lý do: story chỉ có `GET /healthz`, công khai theo bản chất (Docker healthcheck gọi không có phiên), không trả dữ liệu nào ngoài trạng thái. Ràng buộc thay thế: body chỉ có khoá `status`; `.env.example` không chứa secret thật.
  Kiểm: `curl -s localhost:8080/healthz` → đúng `{"status":"ok"}`; `grep -nE 'sk-[A-Za-z0-9]|AKIA[0-9A-Z]{12}' .env.example` → không in gì.

### Ngoài phạm vi của story này
- Tính năng nghiệp vụ, migration, sqlc, Redis/Postgres client trong gateway, worker, LLM, seed (`scripts/seed.mjs`), Caddy, PgBouncer, `docling-serve` (P8), `mock-graph` (P7).
- Primitive giao diện, TanStack Query, tailwind, zustand (PU).
- Sửa hay chạy bất cứ thứ gì trong `legacy/`; viết lại lịch sử git.
- CI (US-P0-03).

### Phụ thuộc
- D45, D46, D48. Không cần story trước. Câu hỏi: `QUESTIONS.md` Q1–Q4 (Q1–Q3 có mặc định để dev không bị chặn).

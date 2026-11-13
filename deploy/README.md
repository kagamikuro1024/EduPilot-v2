# Bộ triển khai máy chủ công khai (không dùng để đo)

Từ 2026-11-12 mọi phép đo chạy trên Mac (`benchmarks/README.md`); VPS không còn. Các tệp dưới đây là nền cho `docker-compose.prod.yml` (P10 L4), chưa được kiểm trên máy thật.

| Tệp | Việc |
| --- | --- |
| `deploy/vps.compose.yml` | override chồng lên `docker-compose.local.yml`: bỏ cổng công khai, Caddyfile riêng |
| `deploy/caddy/Caddyfile.vps` | site `$EP_HOST` (HTTPS tự cấp) + `/edupilot/*` → MinIO |
| `deploy/vps-env.sh` | sinh `.env.local` (bí mật ngẫu nhiên, `chmod 600`, không commit) |
| `deploy/dc` | bọc `docker compose` với hai tệp trên, project `edupilot`, profile `ingest` |
| `deploy/vps-llm.mjs` | cấu hình LLM qua API quản trị: `gemini-chat` + `gemini-embed` (`PHASE=probe|seed|live`, `BASE`, `MODEL`, `FALLBACK`; khoá `GEMINI_API_KEY` từ `.env.local`) |

Quy trình đo LLM thật bằng Gemini trên Mac: `benchmarks/README.md` mục "LLM thật" và `benchmarks/real-llm/README.md`.

# LCP đợt 3 (US-P10-08): bằng chứng thô và nguồn của cột «Trước»

Báo cáo gốc: `docs/sprints/11/handoff/dev-wave3.md`. Tệp này ghi lại NGUỒN các con số (PM hỏi cho luận văn, 2026-11-16). Dữ liệu thô: `lcp-dot3-raw.*` cùng thư mục (giải nén ra `base-all/`, `u-all/`, `final-all/` — 24 tệp `lhr-*.json` mỗi thư mục, các `*.log` của `lhci collect`, `lh.sh`, `lh-sum.mjs`).

## Cách đo
- Lighthouse 12.6.1 qua `lhci collect --config=benchmarks/lighthouse/lighthouserc.ep.json --numberOfRuns=3 --settings.throttlingMethod=devtools` (`lh.sh`), 8 URL × 3 lượt mỗi bộ, trung vị (`lh-sum.mjs`).
- Worktree: `~/Documents/TA_Agent_s108dev` (nhánh local `dev-11`, theo dõi `origin/sprint/11-p10-eval`), `LH_ORIGIN=https://localhost:3460` (proxy HTTPS cùng origin → gateway Go chạy thẳng + bản build production `output: standalone`; Postgres/Redis/MinIO trong container riêng; dữ liệu `scripts/seed-t1.mjs --no-vectors`). MacBook Apple Silicon, không worker / pgbouncer / Caddy.
- Thời điểm (`fetchTime` trong JSON, UTC; +7 giờ = giờ máy):

| Bộ | Thư mục thô | Số tệp | fetchTime đầu | fetchTime cuối |
| --- | --- | --- | --- | --- |
| TRƯỚC (HEAD cũ) | `base-all/` | 24 | 2026-11-15T09:35:47.040Z | 2026-11-15T09:52:55.811Z |
| Sau U1+U2+U4 | `u-all/` | 24 | 2026-11-15T09:54:03.163Z | 2026-11-15T10:11:10.635Z |
| Sau đổi chữ (a) | `final-all/` | 24 | 2026-11-15T10:46:10.716Z | 2026-11-15T11:03:42.330Z |

## Kết quả: LCP từng lượt (ms) → trung vị

| Route | TRƯỚC (HEAD cũ) | Sau U1+U2+U4 | Sau đổi chữ (a) |
| --- | --- | --- | --- |
| `/` | 1938 / 1949 / 2068 → **1949** | 1924 / 1927 / 1964 → **1927** | 1932 / 1943 / 1947 → **1943** |
| `/chat` | 1514 / 1529 / 1552 → **1529** | 1519 / 1524 / 1533 → **1524** | 1518 / 1526 / 1529 → **1526** |
| `/threads` | 1518 / 1519 / 1525 → **1519** | 1521 / 1525 / 1531 → **1525** | 1511 / 1514 / 1515 → **1514** |
| `/inbox` | 1512 / 1520 / 1520 → **1520** | 1517 / 1517 / 1517 → **1517** | 1508 / 1513 / 1519 → **1513** |
| `/gradebook` | 5650 / 5700 / 5703 → **5700** | 5520 / 5524 / 5586 → **5524** | 1535 / 1537 / 1561 → **1537** |
| `/me` | 6584 / 6639 / 6724 → **6639** | 5675 / 5703 / 5765 → **5703** | 1474 / 1475 / 1478 → **1475** |
| `/settings/llm` | 1464 / 4737 / 4749 → **4737** | 4467 / 4504 / 4508 → **4504** | 1444 / 1468 / 1470 → **1468** |
| `/assignments/[id]` | 5145 / 5184 / 5241 → **5184** | 4822 / 4841 / 4857 → **4841** | 4823 / 4834 / 4843 → **4834** |

Khớp bảng ở `dev-wave3.md` (cột «Trước (HEAD)», «Sau U1 + U2 + U4», «Sau đổi chữ (a)»; /settings/llm trước có một lượt 1.464 ms ở lượt đầu — trung vị 4.737).

## Cây mã nguồn của cột «Trước» — KHÔNG phải 2f033ba1
- `2f033ba1` là merge tạo lúc 2026-11-15 18:12:23 (+0700), SAU cả ba bộ đo (16:35–16:53, 17:11, 18:04) nên không thể là cây được đo.
- Tip `dev-11` ngay trước lượt đo đầu (16:35:47 +0700) là **`46335417`** (16:29:36); trong lúc đo đầu còn thêm `98beea85` … `c3220b5c` (16:44–16:49).
- **Không chứng minh được sha bản build**: bản build production được phục vụ lúc đo dựng TRƯỚC 16:35, nhưng không còn dấu vết dựng (thư mục `.next` của worktree đã bị dựng lại; reflog của worktree hết hạn; không ghi sha lúc dựng). `46335417` là ứng viên gần nhất, không phải bằng chứng.
- Tương đương về mã sản phẩm: `git diff 46335417 5fd10b5c -- frontend/src` (`5fd10b5c` = cha của commit U1 đầu `3d1ee7e3`; so với `2f033ba1` chỉ đổi `benchmarks/lighthouse/auth-ep.cjs`, `summarize.mjs`) chỉ chạm admin/audit, analytics, chat (5 dòng), insights và `shared/shell/nav.ts` (+3 dòng); KHÔNG chạm `features/me|gradebook|settings|assignments`, `shared/ui|styles|data`, `app/layout`.
- Cách viết cho luận văn: «trước U1/U2/U4 (nhánh dev-11, tip 46335417, đo 15/11/2026 16:35–16:53 +0700, Lighthouse 12.6.1 devtools, 3 lượt, trung vị)».

# EduPilot v2 — bộ khung khởi đầu dự án

Thả toàn bộ gói này vào gốc repo `TA_Agent` (không ghi đè file đang có), commit, rồi mở Claude Code.

## Bản đồ

| Đọc theo thứ tự | File | Trả lời câu hỏi |
| --- | --- | --- |
| 1 | `docs/WORKFLOW.md` | Một mình làm với Claude Code thế nào; 23,5 tuần tới vạch bảo vệ, 25,5 tuần tới vạch thí điểm; cắt gì khi trễ |
| 2 | `docs/PRD.md` | Hệ thống phải làm gì (14 module, tiêu chí nghiệm thu) |
| 2b | `docs/FLOWS.md` | 18 hành trình end-to-end của người dùng, nhánh lỗi, spec E2E; 17 lỗ hổng đã tìm và vá khi rà soát |
| 3 | `docs/SYSTEM_DESIGN.md` | Vì sao kiến trúc thế này: ước lượng tải, nút cổ chai, đánh đổi, SLO |
| 4 | `docs/design/DESIGN.md` | Trông thế nào: hệ thiết kế *Red Thread / Academic Instrument*, hợp đồng từng route |
| 5 | `docs/design/INTEGRATION.md` | `DESIGN.md` khớp vào kế hoạch ra sao; 10 điểm lệch đã chốt |
| 6 | `docs/UX.md` | Cư xử thế nào khi mạng xấu, bấm đúp, dữ liệu lớn; ngân sách hiệu năng; cổng UX |
| 7 | `docs/ARCHITECTURE.md` | Làm bằng gì: cấu trúc Go, thư viện, schema, API, gRPC, route, env, seed, test |
| 8 | `docs/phases/` | 14 lệnh thi công: P0 → PG → PU → P1 … P10 → PR |
| 8b | `docs/PRODUCTION_READINESS.md` | Từ "chạy được trên seed" tới "trường dùng thật": pháp lý dữ liệu cá nhân, bảo mật, sao lưu, giám sát, kế hoạch thí điểm |
| 9 | `docs/DECISIONS.md` | 43 quyết định và lý do |
| – | `docs/PROGRESS.md` | Đang ở đâu (Claude Code tự cập nhật) |
| – | `docs/thesis-notes/` | Ghi chú luận văn, `/handoff` hoặc PM tự thêm sau mỗi phiên/sprint |
| 10 | `docs/team/` | Đội 4 agent trên herdr: `README.md` (vòng sprint), `PM.md` (prompt cho PO/PM), `BA.md`, `DEV.md`, `QC.md`, `TEMPLATES.md` |
| – | `docs/specs/`, `docs/sprints/` | US/SRS theo feature và kế hoạch/báo cáo theo sprint — nguyên liệu viết báo cáo đồ án |

## Thứ có sẵn để dùng ngay

| Đường dẫn | Là gì |
| --- | --- |
| `CLAUDE.md` | Bộ nhớ dự án: 9 nguyên tắc + 6 luật mở rộng + luật giao diện + luật UX + danh sách cấm |
| `.claude/skills/phase`, `gate`, `handoff` | Ba lệnh: `/phase PU L1`, `/gate PU`, `/handoff` (dùng khi làm một mình; trong đội herdr, `qc` dùng `/gate`) |
| `.claude/settings.json` | Danh sách lệnh shell được phép cho các pane dev/qc |
| `scripts/team-up.sh` | Dựng 4 pane herdr và khởi động 4 Claude Code |
| `scripts/ui-antipatterns.sh` | Kiểm phản mẫu giao diện theo `DESIGN.md` §21 |
| `frontend/src/shared/styles/tokens.css` | Token `--ep-*` (bản sao của `docs/design/DESIGN_TOKENS.css`) |
| `frontend/public/brand/` | Logo, mark, favicon |
| `docs/design/edupilot-ui-v3.html` | Prototype tương tác: mở bằng trình duyệt, đổi vai trò ở góc trên phải |
| `docs/design/AGENT_PROMPT.md`, `KIT_README.md`, `assets/` | Bản gốc của bộ UI kit, giữ nguyên văn |

## Bắt đầu

Một mình với Claude Code:
```bash
claude              # tại gốc repo
/phase P0 L0
```

Đội 4 agent trên herdr (xem `docs/team/README.md`):
```bash
herdr && bash scripts/team-up.sh     # rồi dán docs/team/PM.md vào pane pm, gõ: bắt đầu sprint 1
```

# QC sprint 9.5 (FEAT-nfr-eval) — quy ước chung của bộ test case

Viết ở **pha 1** (2026-11-14) từ `docs/specs/FEAT-nfr-eval/{US,SRS,QUESTIONS}.md` v1.0 APPROVED, `docs/SYSTEM_DESIGN.md` §1.2, §5. Không đọc mã, handoff, báo cáo đo của dev. Chưa chạy gì. Câu hỏi: `qc/questions.md`; **PM đã trả lời Q-N-1…23 ở P-95-7 (2026-11-14)**: chấp nhận theo đề xuất QC; Q2, 9, 14, 18, 23 dev trả lời ở handoff («chờ handoff»); Q8, Q20 dev bắt buộc. TC ghi «P-95-7 Qn» là đã chốt.

## Vai của QC ở sprint đo này
Sprint 9.5 **đo, không thêm tính năng** (US quy ước chung). Vì vậy QC không đo lại mọi thứ (SRS §2: QC «chạy lại một phần độc lập: smoke hộp đen, kiểm `meta`, kiểm số liệu báo cáo = tệp nguồn», SRS §9: «QC chạy lại ≥ 1 kịch bản k6 và đối chiếu số `meta`»). Bốn loại việc:
| Loại | Ý nghĩa |
| --- | --- |
| `FILE` | Đọc tệp kết quả / tài liệu, tính lại số, kiểm cấu trúc (`jq`, `grep`) |
| `RUN` | QC chạy một lệnh / công cụ của sprint trên stack `-p ep-qc` (ngắn, nhẹ) |
| `RERUN` | QC chạy lại **một phần** phép đo nặng, độc lập, và đối chiếu với tệp của dev |
| `SCAN` | Quét bí mật / PII / VPS / khoá bằng `grep` và `scripts/canary-scan.sh` |
| `DIFF` | Đọc diff nhánh `sprint/9.5-nfr` so với `main` (pha 2; không làm ở pha 1) |

## Quy tắc tài nguyên (PM 2026-11-14, CONTEXT mục 4)
- **Một stack một lúc, một phép đo nặng một lúc.** Pha 2 chỉ bắt đầu khi PM báo dev xong phép đo nặng và stack `ep-dev`/`ep-research` đã `down`. QC dùng `-p ep-qc`, `EP_PORT_OFFSET=500`, cổng bind `127.0.0.1`, `down` (giữ volume) khi xong; không đụng stack demo `~/Documents/EduPilot-demo` (kể cả `docker compose -p edupilot stop`: **việc của dev/PM**, QC không dừng demo; P-95-7 Q8).
- Output lệnh cắt còn dòng lỗi + tóm tắt. Mọi số do QC đo ghi kèm «MacBook Apple Silicon, Colima 6 CPU / 10 GB» và `demo_stack: running|stopped` đúng thực tế lúc đo.
- Ngân sách LLM thật: **QC không dùng khoá Gemini** (không chạy lại US-NFR-08; P-95-7 Q20). Mọi TC LLM của QC dùng provider `fake`.

## Tệp / lệnh dự kiến theo spec (QC kiểm có đúng tên không; tên khác → ghi, không tự coi là lỗi nếu spec nói «hoặc tương đương»)
`benchmarks/README.md` (mục «Đo trên Mac»), `benchmarks/real-llm/README.md`, `deploy/vps-llm.mjs`, `benchmarks/check-meta.mjs`, `scripts/seed.mjs`, `scripts/check-*-seed.mjs` (assignments, chat, crm, docs, exam, grade, knowledge), `scripts/seed-t1.mjs`, `scripts/check-t1-seed.mjs`, `scripts/canary-scan.sh`, `benchmarks/load/{chat,grading-vs-chat,today,smoke,heartbeat}.js`, `benchmarks/load/k6-stack.yml`, `benchmarks/reports/{load-a,load-b,load-c,load-d,chaos,baseline,explain}.{json,md}`, `benchmarks/reports/real-llm/*.json`, `backend-go/db/migrations/00018_indexes_tuning.sql`, `docs/research/2026-11-xx-explain.md`, `docs/research/2026-11-xx-nfr-real-llm.md`, `docs/sprints/9.5/{techlead,proposals}.md`.

## Khối `meta` (US quy ước chung) — dùng ở nhiều TC
Trường bắt buộc: `commit`, `thời điểm`, `máy` (chuỗi «MacBook Apple Silicon 18 GB, Colima 6 CPU / 10 GB, k6 trong VM 1,5 CPU»), `số lần chạy`, `cấu hình` {số bản gateway, worker, provider, **chuỗi model thực đo**, `RPM`}, `k6_cpus`, `demo_stack` ∈ {`stopped`,`running`}. Tên khoá JSON chính xác: **chờ handoff** (P-95-7 Q14); mặc định QC đọc `check-meta.mjs` ở pha 2 và kiểm đúng các khoá công cụ định nghĩa.

## Thứ tự chạy pha 2 (nhẹ → nặng, đúng `SRS` §3)
NFR-01 → 02 → 03 (seed T1 dài) → 05 → 04 (QC chạy lại **1** kịch bản) → 06 (QC chạy lại **1** tình huống) → 07 → 08 (chỉ kiểm tệp + chạy harness offline) → 09 → 10. Mỗi bước dựng stack sạch từ commit handoff khi cần (T1 thì giữ DB); `down` sau mỗi bước nặng.

## Cái QC không kiểm (ngoài phạm vi sprint theo spec)
Judge amd64 có seccomp; E3 chấm tay; ≥ 2 provider (E4); mạng thật / p95 trên VPS; recall@k; UI mới (sprint không có UI).

## Lịch sử sửa TC (chỉ khi SPEC đổi: ngày, TC nào, lý do)
- 2026-11-14 — cập nhật theo P-95-7: xem «Lịch sử sửa TC» từng tệp.

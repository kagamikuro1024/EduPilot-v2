### DEF-R1-01 — DEMO_SCRIPT nêu spec E2E không có trong FLOWS.md
- Mức: Thấp (tài liệu)
- Sprint gốc: 1 · TC gốc: US-P0-01/TC-07 · Hồi quy do sprint sau?: CÓ (DEMO v2 bd032bf2 + sprint 10.8 thêm spec, không cập nhật FLOWS)
- Môi trường: be0c0d96, không cần stack
- Bước tái hiện: 1. `bash docs/sprints/1/qc/scripts/tc-US-P0-01.sh TC-07`
- Kỳ vọng (AC7): mọi `*.spec.ts` nêu ở DEMO_SCRIPT có trong FLOWS.md (đúng mục flow) · Thực tế: THIẾU audit.spec.ts, insights.spec.ts, checkin-code.spec.ts; attendance-checkin/-display nằm ở mục tự phục vụ chứ không ở F7
- Bằng chứng: docs/regression/evidence-1/run-01.log · Nghi ngờ mã: docs/DEMO_SCRIPT.md dòng 116, 162; docs/FLOWS.md mục F7 (dòng ~104–112)
- Chạy lại lần 2: ổn định

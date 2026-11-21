# STATE DefR (2026-11-21, dừng theo lệnh PM: giới hạn 2 agent con)
Đã P (ghi results-DefR.tsv + defects-closed.md): DEF-R2-04, R9.5-16, R9.5-22, R10.8-02, R9.5-20, R5-03, R5-04.
Cách chạy chung: Playwright qua /tmp/qc2-pw.sh (khoá); NODE_TLS_REJECT_UNAUTHORIZED=0; script chạy nền `bash evidence-defrerun/run-r5.sh` / `run-r107.sh` (log R5/*.log, R10.7/*.log, all.log). ep-r đang/đã rebuild lên 4e9d1d3a → chạy lại sau «ep-r lên lại».

## CÒN LẠI
1. R5 (cwd evidence-5/ui, env ONLY=…):
   - Trước s27/s18: python3 evidence-5/scripts/s20d_fixture.py (ghi /tmp/qc2L/uix2.json; mkdir -p /tmp/qc2L; /tmp bị dọn nên tạo lại mỗi lượt).
   - DEF-R5-06 TC-PE06-27: ONLY=s27 node smkP_m.cjs (30 SV nộp code chậm làm đầy hàng, smkP_flood.py); R5-07 TC-PE07-18: ONLY=s18 node smkP_m.cjs; R5-10 TC-PE03-33 UI: ONLY=stale node smkP_m.cjs; API: python3 evidence-5/scripts/smkP_s5api.py p0333.
   - DEF-R5-08 TC-PE08-31: ONLY=res node smkP_m2.cjs; R5-09 TC-PE08-38: ONLY=staff node smkP_m2.cjs (điểm 3 mobile 390 sửa ở 42c1ed09; 1,2 theo P-11-59).
   - DEF-R5-11 TC-PE08-04: python3 evidence-5/scripts/smkP_s5api.py p0804 (SSE exam.attempt_graded).
   - Lượt s27 trước đó F do thiếu /tmp/qc2L/uix2.json (môi trường, không phải sản phẩm); chưa có kết quả thật.
2. DEF-R3-01 TC-P105-44: evidence-3/ui/smkP_p10544.cjs (admin 1440/390 /settings/llm, 0 khớp embedding|RAG|PII|trace|fallback|prompt). Chưa chạy.
3. DEF-R7-02 TC-P4-02-14: evidence-7/scripts-r3/p402c.py (rẻ, tuỳ chọn). Chưa chạy.
4. DEF-R9.5-17 TC-N02-02/03: node scripts/seed.mjs trên ep-r (API_URL=https://localhost:1143/api/v1 MAILPIT_URL=http://localhost:8725 SEED_DEFAULT_PASSWORD=Edupilot#Seed-2026 COMPOSE_PROJECT_NAME=ep-r SEED_ADMIN_CMD="docker exec -e ADMIN_PASSWORD=Edupilot#Seed-2026 ep-r-gateway-1 /gateway admin create" NODE_TLS_REJECT_UNAUTHORIZED=0); kỳ vọng không còn 422 IDEMPOTENCY_KEY_REUSED ở nhập điểm (khoá theo upload_id, 6ec47f79); đếm bảng trước/sau; check-*-seed đỏ vì dữ liệu QC thì ghi rõ. Chưa chạy.
5. DEF-R10.7-01..04: audit 10.7. Chạy `bash evidence-defrerun/run-r107.sh` (data×4 vai, empty ×4 với QC_CLASS=QCEMPTYr107 cho teacher/ta/sv31, loading, error; pha rr-defrerun; rồi audit_report.py → evidence-10.7/audit-rr-defrerun.md). Xoá evidence-10.7/audit/rr-defrerun trước khi chạy lại (jsonl nối đuôi). Xem: 01 H3/E7/P5 teacher+admin /observability; 02 H7/X3 /gradebook; 03 X3 student /practice 375 (loading/error); 04 T2 /settings/llm admin+teacher. Chưa có kết quả (lượt dở bị hủy khi rebuild).
6. DEF-R7-01: không cần (qc3 đóng). KHÔNG làm R9.5-21, R9.1-06, R4-05.
Vật cản: khoá Playwright tranh nhiều agent (chờ lâu); ca đỏ lúc rebuild/k6 = chạy lại.

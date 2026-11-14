# QC test case — US-DM-08 (đo trước / sau, seed, hướng dẫn, cổng nghiệm thu)
Nguồn: US.md + SRS §9 («Quy trình đo trước / sau»); Q-D-14, 15; D46 (đánh giá offline Python chỉ HTTP); AGENTS.md (cấm sửa test). Hộp đen; quy ước: `tc-00-chung.md` (**provider giả; lượt thật chờ phép PM**).

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-D08-01 | AC1 | Stack trống (`down -v` → `pnpm dev`, `SEED_ON_EMPTY_DB=true`) | `node scripts/seed.mjs`; `SELECT … FROM llm_providers`, `decision_gates` | Mã 0; provider `decision-fake` có sẵn (bật); 3 hàng `decision_gates` đều `enabled=false`, ngưỡng mặc định (0,05 / 0,5 / 0,95; 800 / 3000 / 800); **0 lời gọi `DECIDE`** sau seed | DB |
| TC-D08-02 | AC1 | – | `SEED_DECISION_DEMO=1` rồi seed | Ba cổng bật cho lớp 1 (hiệu lực: `GET /courses/{C1}/decision-gates` `effective=true`), lớp 2 không | API |
| TC-D08-03 | AC1 | – | `node scripts/check-decision-seed.mjs`; chạy seed lần hai | Mã 0; hàng bảng cấu hình / provider **không đổi** (idempotent) | TAY |
| TC-D08-04 | AC2 | Repo | Chạy đúng 2 lệnh `go test` ở AC2 (`./internal/llm/... -run 'TestDecide\|TestCascade'`; `./internal/privacy ./internal/agent ./internal/grading ./internal/chat -run '…'`) | Xanh; log nêu từng test (không «no tests to run»); mỗi test trong danh sách tồn tại | GO |
| TC-D08-05 | AC2 | – | `pnpm -C frontend exec playwright test decision-settings.spec.ts decision-observability.spec.ts cascade.spec.ts` | Xanh với provider `fake`; **không** kết nối mạng ngoài (kiểm TC-D09-11) | E2E |
| TC-D08-06 | AC3 | `cascade.spec.ts` | Đọc spec (không mock toàn bộ) | Đủ nhánh (a) lưỡng lự → `Structured`; (b) decision lỗi → `Structured`; (c) cả hai lỗi → chặn; (d) chèn lệnh decision lỗi → cờ regex + `Structured`, điểm không đổi; (e) chat dưới τ / lỗi → như tắt; (f) khủng hoảng → 0 lời gọi decision (bộ đếm giả); assert `cascade_path` ở `llm_audit` mỗi nhánh | DIFF |
| TC-D08-07 | AC3 | – | Tự chạy lại 6 nhánh (a–f) ở stack QC bằng API | Đúng từng nhánh và `cascade_path` | API |
| TC-D08-08 | AC4 | Repo | Đọc `benchmarks/decision/run.sh` + mã | Python **chuẩn** (không thư viện ngoài), chỉ HTTP, không import mã Go (`grep -r "import.*backend" benchmarks/decision` = 0); ba kịch bản (chat 30 × 3, Threads 40, chèn lệnh 12 = 6 biến thể + 6 đối chứng; spec ghi «13 / 7» — BA sửa theo P-105-1 Q6); bật / tắt cùng seed | DIFF |
| TC-D08-09 | AC4 | Stack chạy, provider giả `426 / 682` | Chạy `run.sh` (QC tự chạy, ghi thời gian) | Mã 0; ghi `benchmarks/reports/decision-before-after.{json,md}` + biểu đồ **SVG** (không PNG; XML hợp lệ); mỗi bảng có cấu hình máy «MacBook Apple Silicon, Colima 6 CPU / 10 GB», độ trễ giả (p50 / p95), cỡ mẫu | TAY |
| TC-D08-10 | AC4 | Kết quả TC-D08-09 | QC tính lại bằng SQL `llm_audit` + file thô | Chat: TTFT p50 / p95 bật vs tắt, số lời gọi / câu; Threads: độ trễ đăng, chặn đúng / nhầm, `FAST`/`FALLBACK`/`FAIL_CLOSED` (khớp `llm_audit`); chèn lệnh: cờ đúng / nhầm; sai số ≤ 1 % với số trong báo cáo | TAY+DB |
| TC-D08-11 | AC5 | Báo cáo | Mỗi cổng có dòng «Bật được» / «Giữ tắt» | Theo luật cứng: chat — TTFT p95 ≤ +20 % **và** độ chính xác ≥ luật; Threads — chặn nhầm ≤ 0,05 **và** không lọt thêm; chèn lệnh — 6/6 **và** 0 cờ nhầm trên đối chứng; cổng không đạt giữ **tắt** kèm lý do; QC tự áp luật lên số của mình: kết luận khớp (không «Bật được» khi số không đạt) | TAY |
| TC-D08-12 | AC5 | – | Đổi ngưỡng đạt trong báo cáo / script? | Không nới ngưỡng (`git diff` các hằng đạt: 20 %, 0,05, 6/6) | DIFF |
| TC-D08-13 | AC6 | **Chờ phép PM**; `LLM_DECISION_ALLOW_REAL=false` | Lượt thật ≤ 300 lời gọi | «KHÔNG KIỂM ĐƯỢC (chờ phép PM)» — không tính FAIL, không chạy; xác nhận báo cáo ghi «chờ phép PM» ở cột «thật» (không số bịa) | – |
| TC-D08-14 | AC6 | – | Đọc `run.sh` | Có chặn: cột «thật» chỉ chạy khi `ALLOW_REAL=true` **và** có xác nhận rõ; mặc định bỏ qua; trần 300 | DIFF |
| TC-D08-15 | AC7 | `docs/guides/decision-model.md` | Làm theo **từng bước** trên stack thử | Có đủ mục: bật / tắt cổng (hệ thống, theo lớp), thứ tự bật thử (một lớp → vài lớp → toàn bộ), chọn ngưỡng và hệ quả ε / τ, đọc số ở `/observability`, sự cố (hết hạn mức, cầu dao mở, `FAIL_CLOSED` cao, **tắt khẩn một công tắc**), chi phí (số PoC + công thức), Free Tier dùng dữ liệu (nhắc ở mục đầu), câu lệnh thường dùng; mỗi bước chạy đúng; bước sai → ghi lỗi (số bước) | TAY |
| TC-D08-16 | AC7 | – | Thử «tắt khẩn» theo guide: một công tắc toàn hệ thống khi lưu lượng đang chạy | Trong ≤ 5 s mọi cổng ngừng gọi decision (`llm_audit DECIDE` dừng tăng); chat / đăng / chấm vẫn bình thường | API+DB |
| TC-D08-17 | AC8 | – | `go test ./internal/contract/...`; so `backend-go/api/openapi.yaml` với SRS §6 #1–7 | Xanh; đủ 7 endpoint (+ `409 DECISION_REAL_DISABLED`, `409 VERSION_CONFLICT`, `422`); không thêm endpoint ngoài SRS | GO+DIFF |
| TC-D08-18 | AC8 | – | Đọc `ARCHITECTURE.md`, `SYSTEM_DESIGN.md`, `DECISIONS.md` D60 so với mã / DB | Khớp mã: `FAIL_CLOSED` (không còn `FAIL_RULES`), cột `cascade_gate` / `cascade_path`, bảng cấu hình, env `LLM_DECISION_*`; `grep -rn FAIL_RULES docs backend-go` = 0 | DIFF |
| TC-D08-19 | AC8 | Migration | `goose up` trên DB trống; `goose down`/`up`; không sửa file migration cũ (`git diff`) | Sạch; số kế tiếp liền sau migration cuối đã merge | DB+DIFF |
| TC-D08-20 | AC9 | Gateway tắt | `benchmarks/decision/run.sh` | Mã ≠ 0 + «Gateway chưa chạy»; **không** ghi / sửa báo cáo (hash thư mục trước / sau) | TAY |
| TC-D08-21 | AC4 | – | `git status` sau `run.sh` | Báo cáo ở đúng thư mục; không file lạ / khoá / dữ liệu seed lạ | DIFF |
| TC-D08-22 | Tổng | – | `bash scripts/ui-antipatterns.sh`; `grep -rE "AIza\|sk-[A-Za-z0-9]{20}" <diff>`; `git diff <gốc>... -- '*_test.go' frontend/e2e` | Antipatterns sạch; 0 khoá; chỉ thêm test (không sửa / nới / `skip`); phát hiện sửa test để xanh → lỗi nghiêm trọng báo PM | DIFF |
| TC-D08-23 | Tổng | – | Đối chiếu mặc định tắt sau seed + chạy toàn bộ playwright cũ | Không hồi quy (số test cũ xanh như trước 10.5); mặc định tắt: `llm_audit DECIDE`=0 | E2E |

## Nhánh lỗi
TC-D08-12…13, 20.

## Phân quyền
Không riêng (script offline). Phân quyền cấu hình ở TC-D02-22, TC-D06-17.

## Script chạy được
`docs/sprints/10.5/qc/scripts/dm_bench_check.py` (tính lại số), `dm_seed_check.sh`.

## Lịch sử sửa TC
- 2026-11-14: cập nhật theo P-105-1 (Q1 base URL cấu hình được; Q2 bảng replay + `FAKE_DECISION_PROB`; Q3 bản giả trả 0,25 cho đầu vào lạ; Q6 đối chứng = 6 bài). Spec: BA vá SRS 4.6 / US-DM-04 AC5.

# DEV handoff — US-PE-09 (seed bài thi mẫu, k6, cổng PE)
Nhánh `sprint/5-pe`, spec `FEAT-weekly-exam` v1.7. Không migration mới, không đổi `openapi.yaml`. Kết quả: `bash scripts/gate-pe.sh` (kèm `GATE_SEED=1 GATE_K6=1`) → **`GATE PE: PASS`**.

## Làm gì
- **Dữ liệu chung** `scripts/exam-seed-data.mjs`: 20 câu trắc nghiệm (12 `MCQ_SINGLE`, 4 `MCQ_MULTI`, 4 `TRUE_FALSE`; 5 chủ đề; 3 mức khó; có giải thích), 2 bài code (`Bài code — Ước chung lớn nhất`: `c11`+`cpp17`, trọng số 1/1/2/2/2; `Bài code — Đếm từ`: `cpp17`, 1/1/1/1/1; mỗi bài 2 test mẫu + 3 test ẩn; đề tiếng Việt), mẫu đáp án cố định của 24 lượt trắc nghiệm (A = `sv.gioi` 10/10, B = `sv.kha` 8/10, C = `sv.nguyco` không làm, D = `sv.moi` chưa vào lớp) và 6 lượt code (CE, WA, AC, bỏ trống). Mô hình chấm trong tệp này là **bản độc lập** (BigInt hữu tỉ), không gọi mã Go.
- **`scripts/seed.mjs` bước 10** (HTTP API thật, không `_test`, không DB): tạo câu → duyệt; 5 câu `AI_DRAFT` bằng 5 việc `questions/suggest` (`fake`, mỗi việc 1 câu); 2 bài code (PUT `…/code`, thêm test, duyệt test, `reference/verify` chờ job, duyệt câu); 2 bài thi (tạo → chọn câu → lên lịch, mở sau ~14 s, đóng sau ~150 s); sinh viên làm qua API; seed **thoát**, bộ lập lịch của worker đóng / chấm / công bố. Khoá tự nhiên = `title`; chạy lần hai in `Seed xong (không đổi)`; `APP_ENV=production` → `rc=1` trước mọi lời gọi (đã chạy). Tổng thời gian: **74 s** trên stack trống, **13 s** lần hai (ngân sách 5 phút).
- **`seed/expected_exam_scores.csv`** (30 dòng + cột `formula`, `expected_verdict`) do `scripts/gen-expected-exam-scores.mjs` sinh từ mô hình trên (`--check` trong cổng). **`scripts/check-exam-seed.mjs bank|scores|demo`**: `matched 30/30`; demo kiểm từng tài khoản (A 10,00 + đáp án; C `attempt: null` + `ABSENT`; D 0 lớp, 403/404 ở bài thi; `Câu sai nhiều` có câu ≤ 40 %; **đúng 1** cặp `flagged` (A ↔ S05, lời giải `gcdOk2` là bản đổi tên của `gcdOk`) + `EXAM_SIMILARITY` ở Hôm nay; TA không có `flags`; sv04 (CE) thấy `compile_ok=false` + log trong kết quả của mình).
- **k6 `benchmarks/load/exam-submit.js`**: `SCENARIO=judge_burst|autosave|chat|mixed`; `setup()` tự dựng một bài thi tải (1 câu code 10 test + 1 câu trắc nghiệm, `Tải k6 …`) trên lớp 761987 và cho 30 sinh viên bắt đầu lượt; thêm kịch bản `chat` (đường cơ sở TTFT cho `mixed`). Báo cáo `benchmarks/reports/pe-exam-submit-<ngày UTC>-<kịch bản>.json`.
- **`scripts/gate-pe.sh`**: dừng ở lỗi đầu, bảng `PASS/FAIL` + giây, dòng cuối `GATE PE: PASS|FAIL`; `GATE_SEED=1` thêm `check-exam-seed`, `GATE_K6=1 K6_BASE=…` thêm k6 (tự tính cơ sở TTFT từ báo cáo `chat`).
- **Cấu hình**: `.env.example` đặt `EXAM_MIN_DURATION_MINUTES=1`, `EXAM_MIN_LEAD_SECONDS=5` (dev; production không đặt, mặc định 5 phút / 60 s); `docker-compose.test-seed.yml` thêm hai biến đó + `EXAM_SAVE_RATE_PER_MIN=2000` + `LLM_DEFAULT_RPM/TPM` cao (k6 dồn 300 phiên vào 30 lượt, 50 luồng chat); `docker-compose.local.yml` truyền `LLM_PROVIDER` cho **worker** (việc `question.suggest` chạy ở worker, trước đó worker không có nhà cung cấp dự phòng → `LLM_NOT_CONFIGURED`).
- **Sửa provider `fake`** (`internal/llm/fake/fake.go`, test `TestGenerateNullableAndOptions`): `Generate` hiểu `type: ["boolean","null"]`, danh sách đáp án `{body, correct}` sinh 2 phần tử (đúng, sai) với `body` khác nhau — trước đó câu `fake` sinh ra luôn bị `Validate` loại (`DUPLICATE_OPTION`), nên không thể có 5 câu `AI_DRAFT`.
- **E2E**: thêm `@real seed demo state` ở `exam.spec.ts` (chạy `check-exam-seed.mjs demo` khi có `E2E_REAL_API`; bỏ qua ở CI).
- **`docs/PROGRESS.md`**: hàng phase PE, ánh xạ `00006_weekly_exam` (có từ US-PE-01), danh sách nợ, kết quả cổng.

## AC (kết quả thật)
| AC | Kết quả |
| --- | --- |
| 1 | `check-exam-seed.mjs bank` → `OK` (12/4/4 `APPROVED`, 5 `AI_DRAFT` `PENDING`, 2 bài code 2 mẫu + 3 ẩn, trọng số đúng, lời giải đã xác minh, lớp 761988 không có câu). Chưa chạy `$PSQL … from question_bank` bằng tay (cùng dữ liệu, đã kiểm qua API và `select … from question_bank` ở stack thử) |
| 2 | seed thoát 0; hai bài `PUBLISHED` sau ≤ 3 phút (`check … scores` chờ tới khi cả hai `PUBLISHED`); `grep -rn '_test/' scripts/seed.mjs \| wc -l` = 0 |
| 3 | `check-exam-seed.mjs scores` → `matched 30/30` (điểm `auto_score` + verdict từng bài code) |
| 4 | `check-exam-seed.mjs demo` → `OK`. Lượt UI thật chưa chạy (`@real` chỉ bọc script) |
| 5 | lần hai `Seed xong (không đổi)`, 13 s; `APP_ENV=production` → `rc=1`. Đã bắt lỗi của chính seed: đếm câu `AI_DRAFT` theo `Map<title>` làm mỗi lần chạy lại thêm 4 câu (fake đặt cùng tiêu đề) → đếm theo danh sách |
| 6 | k6 trên stack TEST (judge thật, `JUDGE_PARALLELISM=2`, `-no-seccomp`): `judge_burst` 60 lần nộp / 5 phút (30 tài khoản × 2): `judge_done` p95 ≈ 1 s, `judge_ie` 0, lỗi HTTP 0 %; `autosave` 300 VU / 3 phút: p95 29 ms, lỗi 0 %; `mixed`: TTFT chat p95 11,6 ms so với 12,2 ms riêng (−5 %; ngưỡng +20 %), `Chạy thử` p95 1,5 s, `judge_done` p95 2 s |
| 7 | `exam.spec.ts` + `a11y.spec.ts` trong cổng: 83 pass, 9 skipped (`@real`, ảnh mốc), 0 lỗi |
| 8 | `bash scripts/gate-pe.sh` → bảng dưới, `GATE PE: PASS`, `rc=0` |
| 9 | `ui-antipatterns.sh` 0 ✗; `pnpm lint`, `tsc --noEmit` sạch; axe qua `a11y.spec.ts`; `grep -rnE 'sandbox|verdict|go-judge|provider|trace' frontend/src/features/exam` chỉ ra tên trường API / biến (`verdict` ánh xạ qua `VERDICT_LABEL`/`VERDICT_VI` trước khi hiển thị), không có chuỗi hiển thị. **Chưa chạy**: `lhci autorun` cho `/exams/<id>/take`, 3G chậm bằng Playwright, 4 mốc bề rộng thật (chỉ 375 px của màn làm bài + kết quả có trong e2e) |
| 10 | `grep -c weekly_exam docs/PROGRESS.md` = 2; `docs/sprints/5/report.md` do QC viết (chưa có) |

## Kết quả cổng PE (dán từ `/tmp/gate-pe.log`)
```
| Kết quả | Bước | Giây |
|---|---|---|
| PASS | go vet ./... | 1 |
| PASS | go test -race exam + judge + quiz | 29 |
| PASS | TestSandboxAttacks (15 ca, judge thật) | 2 |
| PASS | TestScoringDecimal | 3 |
| PASS | TestNoAnswerLeak (service + contract) | 14 |
| PASS | go test ./internal/contract/... | 18 |
| PASS | sqlc diff | 1 |
| PASS | expected_exam_scores.csv khớp bộ sinh | 0 |
| PASS | ui-antipatterns.sh | 1 |
| PASS | Playwright exam.spec.ts | 156 |
| PASS | check-exam-seed.mjs bank | 0 |
| PASS | check-exam-seed.mjs scores | 1 |
| PASS | check-exam-seed.mjs demo | 2 |
| PASS | k6 exam-submit judge_burst | 319 |
| PASS | k6 exam-submit autosave | 198 |
| PASS | k6 exam-submit chat (đường cơ sở TTFT) | 339 |
| PASS | k6 exam-submit mixed (TTFT ≤ cơ sở +20 %, p95 Chạy thử ≤ 5 s) | 360 |

GATE PE: PASS
```
Thêm: `make lint sqlc-check` 0 issues ×3 + `sqlc diff` sạch; `go test -race ./internal/llm/...` ok; `pnpm lint`, `tsc` sạch; `lint-selftest` 7/7 + 19/19.

## Hạn chế cần QC / PM biết
- Stack chạy bằng `docker-compose.local.yml + test + test-seed` (`EP_PORT_OFFSET` tuỳ chỉnh); `fake` chat trả ~180 ms, không phải 5–15 s như AC6 mô tả; đo TTFT ở đây phản ánh hàng đợi `INTERACTIVE` chứ không phải độ trễ nhà cung cấp.
- `judge_burst` dùng 30 tài khoản nộp hai lần (cách nhau theo nhịp 12 lần / phút, qua hạn 15 s giữa hai lần nộp), không phải 60 tài khoản; seed chỉ có 57 sinh viên và nhiều tài khoản đã dùng cho bài mẫu.
- Báo cáo k6 mang ngày UTC của lúc `handleSummary` chạy (`chat`, `mixed` ghi sang ngày UTC kế tiếp so với `judge_burst`, `autosave`); mỗi kịch bản một tệp.
- k6 `setup()` để lại câu `Tải k6 — tổng hai số (10 test)` và bài thi `Tải k6 …` ở lớp 761987; `check-exam-seed.mjs bank` bỏ qua câu này (lọc theo tiền tố `Bài code — `). Chạy cổng trên stack mới: seed → kiểm → k6.
- Không chạy: `lhci`, 3G chậm, ảnh mốc `visual`, `@real` UI, GitHub CI (hết hạn mức), amd64 seccomp.

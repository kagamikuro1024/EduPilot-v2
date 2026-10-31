# QC report — US-S8-SEED (seed sprint 8 + cổng P6) · Kết luận vòng 1: **FAIL** (AC7: hai tên `-g` của US không có test nào; mọi AC khác PASS trên stack thật)
Ngày 2026-10-31. Commit kiểm: `sprint/8-p6` @ `6711408d` (+ `c3820726` cho vòng 2 của US-P6-09). Bộ TC: `tc-US-S8-SEED.md` (TC-P6-SEED-01…37). Handoff: `dev-US-S8-SEED.md`. CI run `41208847316` (`6711408d`): **success**.

## 1. Môi trường QC (stack riêng, cạnh stack của dev)
`EP_PORT_OFFSET=100 COMPOSE_PROFILES=ingest BLOB_PUBLIC_ENDPOINT=localhost:9100 node scripts/dev.mjs up` (project `edupilot100`, `https://localhost:543`, docling thật, `LLM_DEMO_FILE=/seed/demo/fake_llm.json`), volume xoá sạch (`down -v`) trước mỗi lần dựng; `node scripts/seed.mjs` với `API_URL=https://localhost:543/api/v1`. Dựng DB trống ba lần (lần 1 và 2 để dò lỗi bên dưới; lần 3 cho kịch bản kill).

## 2. Cổng đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `node scripts/seed.mjs` trên DB trống (SEED_RNG mặc định) | rc=0, 18 bước, có bước 15 (quy chế → công thức → nhập điểm); chạy lại: rc=0, **không đổi** (`check-grade-seed.mjs counts` → `{"docs":6,"columns":3,"entries":60}`) |
| `check-grade-seed.mjs preview / abc / lop2 / impact / counts` | `30/30 OK`; `A B C OK`; `lớp 2: chốt điểm 409`; `impact p95 6 ms`; counts không đổi |
| **Đối chiếu độc lập của QC** `scripts/p6-preview-vs-csv.py` (so `qt, hp_raw, hp, letter, verdict` theo MSSV với `seed/expected_final_grades.csv`) và `p6-hand-calc.py` | **30/30 khớp**; `expected_final_grades.csv` 30/30, `cases-tong-hop.csv` 18/18; `summary.distribution` = A+ 3, A 3, B+ 3, B 5, C+ 4, C 4, D+ 2, D 3, F 3, liệt 1 (đúng bảng QC) |
| Giải trình A, B, C (`…/students/{id}/explain`) | A `HP 9,7 → 9,5 A+` (QT 9,5 + 1 = 10,5 → 10); B `7,2 → 7,0 B` (QT (20×7,5+20×8)÷40 = 7,75 + 0,5 = 8,25); C `3,2 → 3,0 F` (QT 1,25: (20×5+20×0)÷40 = 2,5 + 0,25 − 1,5) — QC tự tính tay khớp từng dòng; B: vắng 2 miễn 2, kiểm tra tuần 8 từ bài thi PE |
| DB (TC-01…12, 37) | tài liệu `Quy chế học vụ` = `OTHER`/"Quy chế trường", `Quy chế học phần` = `COURSE_POLICY`; đúng một công thức `CONFIRMED` (lớp 761987, qua `audit_log` `extract → edit → confirm`, job `grade_scheme.extract` SUCCEEDED); lớp 761988 `NO_POLICY`; điểm `midterm`/`final` `source=IMPORT` 30 + 30; 0 dòng điểm lỗi; `midterm`, `final` đã công bố, `weekly` nguồn `EXAM` nối "Kiểm tra tuần 9 — Mật mã" `PUBLISHED`; sổ lớp 1 `OPEN`, `snapshot_version 0`; 30/30 sinh viên `joined_via=ROSTER` |
| `check-crm-seed`, `check-chat-seed`, `check-docs-seed`, `check-exam-seed bank / scores` | xanh (270 điểm danh, `AT_RISK 1 / WATCH 3`, 5 ticket, 30/30 điểm thi); thay đổi duy nhất ở `check-crm-seed.mjs` là `WATCH 2 → 3` (#5) |
| `bash scripts/gate-p6.sh` | **GATE P6: PASS** (13 bước: `go vet`, `go test -race ./internal/grade/...` 30/30, không `float64`, `TestExtractGradeScheme`/demo, bốn `TestSeed*`, contract, chặn production, `check-grade-seed`, Playwright `gradebook + me` 54 passed, `ui-antipatterns`) |
| Cổng "không xanh giả": thêm `var _ = float64(1)` vào `internal/grade` | `gate-p6.sh` **đỏ** ở `go test -race ./internal/grade/...` (rc=1), hoàn tác |
| `NODE_ENV=production` / `APP_ENV=production node scripts/seed.mjs` | rc=1, "Seed bị chặn"; `--if-empty` trên DB đã seed thoát trong 0,4 s |
| Gateway tắt rồi `seed.mjs` (TC-35) | thoát sau 5 s: "Seed lỗi: POST /auth/login → 503"; bật lại → chạy tiếp thành công |
| **Kill giữa chừng (TC-19)**: DB trống, `kill -9` seed sau `extract` trước `confirm` (`schemes=[DRAFT]`, `entries=0`) rồi chạy lại | rc=0; `schemes=[CONFIRMED]` đúng **1** công thức (không bản nháp đôi), 60 điểm, 1 tài liệu `COURSE_POLICY`; `check-grade-seed preview` 30/30, `abc` OK |
| Lớp 2 trên API thật (TC-14, 15) `scripts/p6-lop2.py` | tải `seed/demo/quy-che-lop2.pdf` → `DRAFT_UNCLEAR`, **đúng một** mục `ROUNDING_MISSING`; thành phần 25 / 25 / 50; `confirm` khi còn mục chưa rõ → `422 SCHEME_HAS_UNCLEAR`; điền làm tròn 0,1 (nửa lên) → `DRAFT_READY` → `confirm` 200 → `CONFIRMED`; `finalize` lớp 2 → 409 |
| Quy chế PDF dựng (TC-13) | `quy-che-lop2.pdf` không có "làm tròn / quy tròn / bước điểm"; lớp 1 có; không rò dấu `<!-- trang -->` |
| Phân quyền trên seed (TC-28…31) | A / B / C chỉ thấy điểm của mình (`/me/grades`: 10 / 8,25 / 1,25); B gọi `explain` của C → 403; D (`sv.moi@`, chưa vào lớp) `/me/grades` 403, `/me/courses` rỗng; TA lớp 1 `200`, lớp 2 `403`; ADMIN `403` ở `gradebook`, `preview`, `export.xlsx`, `me/grades`, `grade-scheme/status` |
| Canary sống (TC-32): cột cuối kỳ gỡ công bố + điểm B = 7,77 | B: `me/grades`, `me/attendance`, `gradebook`, `/notifications` — 0 lần "7,77"; tính thử dùng điểm đã công bố (QT 8,25), không dùng 7,77; đã khôi phục |
| **Tải 1.000 sinh viên** (`scripts/p6-load1000.sql`, lớp LOAD1000 sao chép công thức lớp 1; đo 20–30 lần, bỏ lần đầu) | trang đầu sổ 30 dòng p95 **19 ms**; 100 dòng p95 29 ms; xem trước 100 dòng p95 43 ms; `explain` p95 19 ms; **`POST grade-scheme/impact` p95 52 ms (≤ 300; đóng BUG-P6-04-2)**; `export.xlsx` p95 55 ms; **`finalize` 1.000 sv 0,12 s (≤ 3 s)**; đọc bản chốt trang 100: 11 ms, `Index Only Scan` (không Seq Scan); `/me/grades` sinh viên p95 **16 ms**; `PUT` ô điểm p95 **18 ms** (≤ 300; đóng nợ p95 của US-P6-06) |
| US-P6-09 vòng 2 (dev thêm `TestNoDraftLeakQT`, BUG-P6-09-1) | `TestNoDraftLeak`, `TestNoDraftLeakQT` PASS; **đột biến M1 (bỏ `publishedInput`) làm `TestNoDraftLeakQT` đỏ** → BUG-P6-09-1 đã sửa, US-P6-09 giữ **PASS** |

## 3. Kết quả theo AC
| AC | Kết quả | Ghi chú |
| --- | --- | --- |
| AC1 quy chế + công thức lớp 1 qua luồng thật; lớp 2 không quy chế | PASS | TC-01…04, 13 |
| AC2 điểm lớp 1 qua nhập XLSX thật, 30/30 | PASS | TC-05…09, 37; QC đối chiếu độc lập 30/30 |
| AC3 A, B, C cho cổng tay | PASS | TC-10…12; xem câu hỏi Q-SEED-a (chữ "2 × 0,25") |
| AC4 lớp 2 demo bước 6 (một mục chưa rõ → điền → xác nhận) | PASS | TC-14…17 (cả trên API thật, cả Playwright `lop2 demo step 6`) |
| AC5 idempotent, tự chạy, chặn production, kill giữa chừng, không vỡ seed trước | PASS (kèm BUG-S8-SEED-1, -2) | TC-18…23, 35, 19 |
| AC6 cổng P6 chạy được từ đầu đến cuối, không xanh giả | PASS | `gate-p6.sh` PASS; đột biến `float64` làm đỏ |
| **AC7 `gradebook.spec.ts` phủ F10; mọi tên `-g` nêu ở US có ≥ 1 test** | **FAIL (TC-P6-SEED-27)** | 54 test xanh, nhưng **`-g 'banner unconfirmed'` và `-g 'finalize locked reason'`** (US-P6-05 AC2, AC3 `Kiểm:`) **không khớp test nào** (Playwright báo "No tests found", rc=1); test tương đương có nhưng tên khác: `gradebook banner: công thức chưa xác nhận → Chốt điểm khoá có lý do…` (dòng 483 `gradebook.spec.ts`). Thêm: `-g 'no gamify'` (US-P6-09 AC3) cũng không có test mang tên đó (còn `grade section\|calculation linear` chạy được nên lệnh không đỏ) |
| AC8 phân quyền dữ liệu seed — không rò | PASS | TC-28…33 |

## 4. Lỗi / ghi nhận
- **BUG-S8-SEED-3 (AC7, Thấp, sửa 1 phút):** đổi tiêu đề `gradebook banner: …` thành chứa cả `banner unconfirmed` và `finalize locked reason` (hoặc tách hai test), và một test chứa `no gamify` (hoặc sửa chữ `-g` trong US — việc của BA). Không đổi assertion.
- **BUG-S8-SEED-1 (Trung bình, `scripts/seed.mjs` bước 11):** `uploadDoc` đặt `Idempotency-Key` cố định `seed-doc-<sha16>` nhưng thân `complete` có `upload_id` mới ở mỗi lần chạy. Nếu một lần `complete` trả lỗi 4xx (QC gặp `422 UPLOAD_INCOMPLETE` khi MinIO công khai trỏ nhầm cổng), gateway **lưu luôn phản hồi lỗi** theo khoá (Redis `ep:idem:…` + bảng `idempotency_keys`, 24 giờ) → mọi lần chạy lại đều `422 IDEMPOTENCY_KEY_REUSED`; chỉ thoát khi xoá thủ công cả hai chỗ. Đề nghị: khoá gắn `upload_id` (hoặc không đặt khoá cho `complete`), và/hoặc gateway không lưu phản hồi 4xx tiền điều kiện. Ảnh hưởng AC5 "seed gãy giữa chừng chạy lại tiếp tục" (bước 11 là của sprint trước; bước 15 và kill ở bước 15 chạy lại đúng).
- **BUG-S8-SEED-2 (Thấp):** `SEED_RNG=1 node scripts/seed.mjs` dừng ở bước 15: "Tệp giua-ky: 25 dòng lỗi (MSSV khớp theo dòng ROSTER, #20)" — XLSX dựng từ MSSV cố định của `inputs-lop1.csv`, trong khi `fixtures/README.md` ghi "đổi `SEED_RNG` thì MSSV đổi: so khớp bằng `account`". Đề nghị `grade-seed-data.mjs` lấy MSSV từ danh sách lớp theo `account`, hoặc BA sửa README (seed chỉ hỗ trợ RNG mặc định).
- **Ghi nhận công cụ (Thấp):** `EP_PORT_OFFSET` không đổi `BLOB_PUBLIC_ENDPOINT` (mặc định `localhost:9000`) nên stack thứ hai tải tệp nhầm sang MinIO của stack kia (nguyên nhân gốc của BUG-S8-SEED-1 khi QC gặp); `dev.mjs` nên đặt `BLOB_PUBLIC_ENDPOINT=localhost:${EP_PORT_MINIO}` khi có offset. `.env.local` cũ không có `LLM_DEMO_FILE` (chỉ `.env.example`): stack dựng từ `.env.local` cũ không chạy DEMO_MODE.
- **TC-34 (ghi nhận):** PDF không có bản ghi demo → job `grade_scheme.extract` `FAILED` mã `LLM_UNAVAILABLE` "AI đang không dùng được. Hãy thử lại sau." (không có bản nháp rác; chữ không nói "chưa có bản ghi demo" — chấp nhận vì chỉ chạy ở DEMO_MODE / dev).
- **Q-SEED-a (BA / PM):** AC3 ghi "giải trình của B có: phát biểu 2 × 0,25 = 0,50". Màn giải trình hiện `Điểm cộng | min(0,5; 1) | 0,5` (đúng định dạng đã PASS ở US-P6-02), số lần phát biểu (2) và 0,25/lần không hiện trong dòng. Dữ liệu đúng (DB: 2 sự kiện, tổng 0,50). QC coi AC3 đạt; nếu BA muốn chữ "2 × 0,25" thì đó là đổi spec của US-P6-02.
- Test cũ bị sửa: `grading/helpers_test.go` (`fakeLLM.via`, thêm hàm, không đổi assertion), `check-crm-seed.mjs` (WATCH 2 → 3 theo #5). `fixtures/`, `seed/expected_*` không đổi.

## 5. Còn lại cho cổng P6 (`gate-P6.md`)
Nợ SEED đã đóng: p95 `impact` (52 ms), `finalize` 1.000 (0,12 s), `PUT` p95 (18 ms), `/me/grades` p95 (16 ms), phân bố thật 3/3/3/5/4/4/2/3/3 + 1 liệt, `TestSeedRiskAfterGradebook`, `WATCH`. Việc **chủ dự án tự làm**: tính tay A, B, C (G6-13), xem cổng ở `gate-P6.md`.

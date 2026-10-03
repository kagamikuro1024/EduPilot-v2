# QC — Kịch bản cổng nghiệm thu phase PE (Thi hằng tuần: trắc nghiệm + lập trình C/C++)
Nguồn: `docs/phases/PE.md` mục "Cổng nghiệm thu", "Bạn tự kiểm", "Ghi cho luận văn"; `docs/specs/FEAT-weekly-exam/US.md` US-PE-09 AC6–AC10; `docs/FLOWS.md` F19; `docs/sprints/5/plan.md`. QC chạy lại **mọi** lệnh ở máy QC (không tin số của dev), trên worktree `TA_Agent_v2-s5` (`sprint/5-pe`), stack riêng của QC (tên dự án và cổng riêng; không chạm `edupilot-test-*`). Verdict chỉ PASS / FAIL; TC không chạy được = FAIL "KHÔNG KIỂM ĐƯỢC"; TC cần khoá nhà cung cấp thật = BLOCKED. Số tự đo thắng số công cụ khi lệch; không nới ngưỡng. Sau mỗi lượt: `docker volume prune -f`.

Điều kiện vào cổng: cả 9 story PE có `report-US-PE-0N.md` PASS (hoặc PASS có điều kiện đã được PM ghi); `gate-pe.sh` của dev xanh ở handoff; CI GitHub ở HEAD `sprint/5-pe` xanh (Go + Frontend + `judge-attacks`).

## A. Lệnh tự động (PE.md, từng dòng → một bước đo được)
| # | PE.md | Lệnh QC chạy | Kết quả mong đợi |
| --- | --- | --- | --- |
| A1 | `go test -race ./internal/exam/... ./internal/judge/...` | `cd backend-go && go test -race -count=1 ./internal/exam/... ./internal/judge/... ./internal/quiz/...; echo rc=$?` | `rc=0`; 0 `SKIP` ngoài chú thích có lý do |
| A2 | `TestSandboxAttacks` | `cd backend-go && go test -tags integration ./internal/judge -run TestSandboxAttacks -v -timeout 10m` (colima arm64: `JUDGE_EXTRA_ARGS=-no-seccomp`) | `ok`; 15 dòng `A<n>`; kèm bảng **mã tấn công của QC** (`docs/sprints/5/qc/scripts/attacks/`, A1–A20) khớp `tc-US-PE-02` TC-PE02-33…45: vòng lặp vô hạn → `TLE`; fork bomb → không tạo tiến trình thứ hai; đọc `/etc/shadow`, `/etc/passwd`, `/proc/1/environ` → bị chặn; socket → `connect=-1`; stdout 1 GB → `OLE`/`TLE`; cấp phát 2 GB → `MLE`/`RE`; bài của SV khác vẫn `AC` ≤ 20 s; `RestartCount` giữ nguyên |
| A3 | `TestScoringDecimal` | `go test ./internal/exam -run TestScoringDecimal -v` | `ok`, `matched 40/40`; QC tự đối chiếu 10 dòng bằng Python `Fraction` (TC-PE01-26) |
| A4 | `TestNoAnswerLeak` | `go test -tags integration ./internal/exam -run TestNoAnswerLeak -v`; QC chạy thêm `p508-leak.py` (≥ 20 endpoint × 6 tình huống) | `ok` `leak_matrix 20x6 clean`; QC: **0 canary** ở thân, header, `details`, SSE, log |
| A5 | `go test ./internal/contract/...` | `go test -count=1 ./internal/contract/...` | `rc=0`; golden PG / LLM / auth **không bị xoá dòng** (`git diff origin/sprint/4-p2 -- …/golden \| grep -c '^-[^-]'` = 0) |
| A6 | Playwright `exam.spec.ts` | `pnpm -C frontend exec playwright test exam.spec.ts` (+ toàn bộ bộ test, cổng riêng) rồi `--grep @real` trên stack seed | pass 100 %; tổng bộ ≥ kết quả sprint 4 (222 pass) + ca PE |
| A7 | k6 `exam-submit.js` | `k6 run benchmarks/load/exam-submit.js --env SCENARIO=judge_burst` / `autosave` / `mixed` + kịch bản QC riêng | `judge_done_p95 < 60000`, `judge_ie == 0`; `autosave_p95 < 150`; `mixed`: TTFT chat không chậm hơn +20 %, `Chạy thử` p95 ≤ 5 s; QC ghi số tự đo + cấu hình máy |
| A8 | cổng chung | `go vet ./... && golangci-lint run && sqlc diff && go test -race -count=1 ./... && go test -count=1 ./internal/contract/...`; `pnpm -C frontend lint && build`; `bash scripts/ui-antipatterns.sh`; `--selftest`; `bash scripts/lint-selftest.sh` | `rc=0`; 19 / 19; `ui-allow:` ≤ 10 |
| A9 | không vỡ mock / sprint trước | `audit.mjs` bốn vai + spec, `sweep.mjs only:'student'`, `proto-curl.sh all`; `tc-US-PU-04` đã sửa 8 / 13 / 16 | FAIL 0; PASS ≥ nền sprint 3 (SV 165, GV 170, TA 106, Admin 54, spec ≥ 179) + hàng PE; nav SV 8, TA 13, GV 16, Admin 6 |
| A10 | CI | `gh run list --workflow ci.yml --branch sprint/5-pe --limit 1 --json headSha,conclusion` so `git rev-parse origin/sprint/5-pe` | `success`, job `judge-attacks` (amd64, seccomp bật) xanh |
| A11 | migration | `goose down-to 00004` → `up`; 2 lượt trên DB mới (`down -v`) | xanh cả hai; không bảng mồ côi |

## B. Nhóm TC bắt buộc (QC chạy tay, độc lập với dev)
| # | Nhóm | Phạm vi | Chứng cứ cần có trong `report-GATE-PE.md` |
| --- | --- | --- | --- |
| B1 | **Chống rò đáp án** | `tc-US-PE-05` TC-PE05-14…17, 47, 53, 59; `tc-US-PE-06` TC-PE06-01, 29, 44, 45; `tc-US-PE-08` TC-PE08-25…29, 60…62; mọi endpoint SV **trước và sau công bố**, `reveal_answers` bật / tắt; test ẩn chỉ hiện `hidden:{passed,total}` | bảng `endpoint × tình huống` ≥ 20 × 6 sạch; canary 0 ở log / Redis / dump ngoài bảng đề |
| B2 | **Sandbox tấn công** | `tc-US-PE-02` TC-PE02-33…45 (mã của QC) + A9 thêm đường; không dùng mã của dev | 20 ca: verdict + dấu hiệu; `RestartCount` trước = sau; RAM `judge` ≤ 100 MiB sau 30 s; đĩa ≤ 10 MiB; 0 tiến trình lạ; bài đúng của SV khác ≤ 20 s |
| B3 | **Đồng hồ** | `tc-US-PE-05` TC-PE05-02, 07, 11…13, 27…31, 40…43; `tc-US-PE-06` TC-PE06-39…41; `tc-US-PE-04` TC-PE04-23…27 | hết giờ khi đang gõ (lưu trong grace nhận / sau grace từ chối); grace 10 s ±1 s; bắt đầu muộn `deadline_at = closes_at`; mất mạng 30 s 0 mất; hai tab một nơi ghi; mở / đóng / tự nộp ≤ 10 s |
| B4 | **Điểm** | `tc-US-PE-01` TC-PE01-21…26; `tc-US-PE-08` TC-PE08-09…16, 46…54 | `PARTIAL` + `ALL_OR_NOTHING` khớp bảng tay; câu `void`; **làm tròn một lần** (≥ 3 lượt phân biệt); ví dụ `7.75` / `7.64` / `8`; chấm lại làm **giảm** điểm có lý do + `audit_log` + thông báo |
| B5 | **Tự công bố đúng một lần** | `tc-US-PE-08` TC-PE08-17…24, 65 | công bố ≤ 10 s sau khi mọi lượt `GRADED`; `published_at` 1 lần; 1 thông báo / SV có lượt; 2 bộ lập lịch + `kill -9` không công bố đôi; hoãn / nhả đúng |
| B6 | **Phân quyền** | `tc-US-PE-01` TC-PE01-12…20; `tc-US-PE-08` TC-PE08-63 | ma trận 4 vai × 4 ghi danh × 56 thao tác: ADMIN 403 mọi route PE; TA không sửa điểm / `regrade` / `override` / `hold`; SV chỉ của mình; không rò giữa lớp |
| B7 | **Liêm chính** | `tc-US-PE-07` TC-PE07-01…47 | khoá chat đúng pha (Redis chết vẫn chặn); log chỉ GV; không tự trừ điểm; so giống A~B ≥ 0,95, A~C ≤ 0,35; 1.000 bản nộp ≤ 30 s; mã không rời hệ thống |
| B8 | **Giao diện / 375 px** | `tc-US-PE-04/05/06/08` phần **A** | `AUDIT_SRC` / `TOUCH_SRC` sạch; axe 0 `serious`/`critical`; trắc nghiệm + kết quả SV 375 px không tràn; code < 1024 px có dải nói rõ và không `textarea` |
| B9 | **Seed + F19 trọn** | `tc-US-PE-09` TC-PE09-01…14, 31 | seed ≤ 5 phút, idempotent, chặn production; `matched 30/30`; QC tự tính tay 30 lượt khớp |

## C. "Bạn tự kiểm" (PE.md) — QC làm bằng tay và chụp ảnh
| # | PE.md | Cách làm | Mong đợi |
| --- | --- | --- | --- |
| C1 | Soạn bài code có **5 test ẩn**, chạy lời giải mẫu → 100 %; nộp lời giải sai một nhánh → điểm đúng tỉ lệ test đạt | GV tạo bài (5 test ẩn trọng số 1/1/1/2/2), `reference/verify` → `ok:true`; SV nộp bản sai test 4 | điểm = `Σweight_đạt ÷ Σweight` × `points` → ví dụ `(1+1+1+2)/7 = 5/7`; QC tính tay và so |
| C2 | Làm bằng tài khoản SV **trên điện thoại** (trắc nghiệm) và **máy tính** (code); tắt mạng 30 s → không mất bài | Chrome emulate iPhone 375 (trắc nghiệm) + 1440 (code); `Network.emulateNetworkConditions offline` 30 s; (nếu có thiết bị thật: ghi kiểu máy, nếu không: ghi "chỉ giả lập") | 0 mất; ảnh trước / trong / sau offline |
| C3 | Trong giờ thi hỏi chat AI về nội dung → bị từ chối; hết giờ → hỏi lại được | trước P3: `GET /_test/chat-gate` (`allowed:false` → `true` sau nộp); khi P3 có: hỏi thật | từ chối trong giờ, cho phép sau nộp ≤ 10 s |
| C4 | Sau khi bài đóng: SV thấy điểm, **không thấy test ẩn**; GV thấy cặp nghi chép | SV A / C / D + GV + TA trên seed | đúng như TC-PE09-10; ảnh |

## D. Luồng F19 trọn vẹn (định nghĩa "xong" của AGENTS)
GV soạn câu (trắc nghiệm + code + zip + verify + duyệt, có AI gợi ý nháp) → dựng bài + lên lịch (TA không lên lịch được) → thông báo + "Hôm nay" cho SV → SV làm (trắc nghiệm 375 px, code 1440, chạy thử, nộp, mất mạng, hai tab, hết giờ) → đóng → chấm → **tự công bố** → SV xem điểm + đáp án (theo `reveal_answers`) + test mẫu + `hidden:{passed,total}` → phúc khảo → GV trả lời → GV xem bảng điểm / phân bố / câu sai nhiều / nghi chép / CSV; **mọi nhánh lỗi** ở `FLOWS.md` F19 (mất mạng giữa bài, hết giờ khi đang gõ, hai tab, nộp trùng, test sai → chấm lại, sandbox chết) đã đi qua. Có spec E2E tương ứng (`exam.spec.ts` ≥ 10 ca).

## E. Điều kiện PASS cổng
Mọi dòng A và B PASS; C1–C4 có ảnh / số; D đi hết; không còn bug mở (chỉ lệch nhỏ đã được PM chấp nhận); không đỏ nào được "giải quyết" bằng nới ngưỡng / sửa golden / sửa ảnh mốc không lý do. LCP: theo quy tắc chung đã chốt (#26) — QC ghi số, không FAIL riêng cho PE nếu cùng điều kiện với toàn dự án.

## F. Ghi cho luận văn (QC thu số)
Sơ đồ kiến trúc chấm code (sandbox, hàng đợi `judge.submit`/`judge.run`, giới hạn tài nguyên); **bảng kết quả tấn công** (20 ca: verdict + dấu hiệu + số đo RAM/đĩa); số đo thông lượng chấm (bài/phút theo `JUDGE_PARALLELISM` 2 và 4; p95 60 bài ≤ 60 s); bảng điểm tính tay vs máy (30 dòng); độ giống A~B / A~C / A~A'; vì sao điểm thi không qua LLM (kiểm `go list -deps ./internal/quiz ./internal/judge \| grep -c internal/llm` = 0); ảnh before / after các màn mới (`docs/sprints/5/qc/shots/`).

## G. Dọn dẹp
Gỡ stack QC (`down -v`), container `judge`, mạng `judge_net`, Chrome tạm, Next/gateway/worker cổng riêng, `caffeinate`, worktree phụ; `docker volume prune -f`; `docker ps` không còn container QC; cây git chỉ có artefact QC.

## Câu hỏi cho BA / PM
- **Q-QC-GATEPE-1** — A2: trên colima arm64 `-no-seccomp` (D58). QC không có runner amd64: cổng PE chấp nhận "đọc cấu hình CI + `gh run` xanh job `judge-attacks`" cho phần seccomp bật? — *chờ PM*.
- **Q-QC-GATEPE-2** — C2: "điện thoại thật" — QC chỉ có giả lập 375 px; phần thiết bị thật để chủ dự án tự kiểm? — *chờ PM*.

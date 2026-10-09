# Báo cáo cổng nghiệm thu phase PE (thi hằng tuần: trắc nghiệm + lập trình C/C++)
**Kết luận: CHƯA ĐẠT — 2 điều kiện chặn** (còn lại ĐẠT). Bản chấm `origin/sprint/5-pe` = `2e65d0a` → `e1e3ebe` (CI), QC chạy lại mọi lệnh ở máy QC (colima arm64, `-no-seccomp`) + CI GitHub thật (job `Judge attacks (amd64, seccomp bật)`).

## Chặn
1. **B1 — US-PE-09 TC-17 / A7 (`mixed`)**: TTFT chat khi có `judge_burst` vượt +20 % ở **2 / 3** lần đo (cơ sở 10,1 / 10,6 / 11,7 ms → 12,4 / 13,3 / 11,7 ms; p50 không đổi). Nghi nhiễu do provider `fake` (~5 ms) — cần PM/BA chốt: ngưỡng tuyệt đối hay đo bằng provider có trễ thật (**Q-QC-GATEPE-3**). Không nới ngưỡng.
2. **A10 — CI Go đỏ ở HEAD**: run `37935921572` (`e1e3ebe`): job `Go` thất bại ở bước `go test -race (testroutes)` (dev / tech lead đang xử lý TL-3: `TestJoinFailureTimingEqualized` và nhóm đo thời gian); `Frontend` đang chạy; `Judge attacks (amd64, seccomp bật)` **xanh**. Cần CI xanh toàn bộ ở HEAD cuối.

## A. Lệnh tự động
| # | KQ | Bằng chứng |
| --- | --- | --- |
| A1 | PASS | `gate-pe.sh`: `go test -race exam + judge + quiz` PASS; `go test -race -tags integration ./internal/exam/... ./internal/judge/... ./internal/contract/...` rc=0 (lượt đầu của bộ đầy đủ đỏ 3 ca vì container Postgres thử chết giữa chừng — `connection refused` ×169, hạ tầng; chạy lại các gói: `ok` exam 59 s, judge 68 s, contract 31 s) |
| A2 | PASS | `TestSandboxAttacks` (có `-tags integration`, judge thật): **15 ca `A1…A15` PASS** (8,9 s); 20 chương trình tấn công của QC (`scripts/attacks`) đã chấm ở `report-US-PE-02.md` (RAM `judge` rảnh 54 MiB ≤ 100). **seccomp bật trên amd64: job CI `Judge attacks` xanh** (đóng Q-QC-GATEPE-1 và L1 của PE-02) |
| A3 | PASS | `TestScoringDecimal` `matched 45/45`; QC tự chấm 24 lượt seed bằng `fractions` (`p509-score.py`) và 12 lượt tổ hợp (TP, FP) ở PE-08: 0 lệch |
| A4 | PASS | `TestNoAnswerLeak` `leak_matrix 20x6 clean (221 lời gọi)`; QC quét tay ở PE-05/06/08: 0 canary |
| A5 | PASS | `go test ./internal/contract/...` PASS; `git diff origin/sprint/4-p2...HEAD -- '*golden*' '*testdata*'`: **0** dòng bị xoá |
| A6 | PASS | Playwright (không `@real`, `visual`): **443 pass, 0 fail, 103 skip** (≥ 222 của sprint 4); `exam.spec.ts` + `a11y.spec.ts` trong cổng dev 83 pass |
| A7 | **FAIL (1/4 kịch bản)** | `judge_burst` p95 **1.040 ms** (< 60.000), `judge_ie` 0; `autosave` p95 **67 ms** (< 150), 0 lỗi; `mixed` `Chạy thử` p95 2,1–2,6 s (≤ 5 s); **`mixed` TTFT +23 % / +26 % / −1 %**: FAIL (B1) |
| A8 | PASS | `go vet`, `golangci-lint` 0 issues, `sqlc diff`, `pnpm lint`, `build:gate`, `ui-antipatterns.sh`, `lint-selftest.sh` rc=0; **gate-pe.sh `GATE PE: PASS` rc=0; gieo lỗi → `FAIL` rc=1 dừng ở bước đầu** |
| A9 | KHÔNG KIỂM ĐƯỢC | `audit.mjs` bốn vai / `sweep.mjs` / `proto-curl.sh` chưa chạy ở lượt này; nav 8 / 13 / 16 đã chấm ở PE-04 và Playwright (`seed accounts nav`) PASS |
| A10 | **CHƯA ĐẠT** | xem Chặn 2 |
| A11 | PASS | `goose` up → `down-to 00004` → up trên Postgres mới (pgvector pg18): `00006_weekly_exam` áp rồi gỡ rồi áp lại, 9 bảng PE trở về; không bảng mồ côi |

## B. Nhóm TC bắt buộc (chứng cứ ở báo cáo từng story, chạy lại số liệu trên HEAD)
| # | KQ | Bằng chứng |
| --- | --- | --- |
| B1 chống rò đáp án | PASS | PE-05 TC-14…17, 53, 59; PE-06 TC-01, 29, 44, 45; PE-08 TC-25…29, 60…62: 0 canary ở mọi thân / header / lỗi trước và sau công bố, `reveal_answers` bật / tắt; test ẩn chỉ `{passed,total}`; `leak_matrix 20x6 clean`; grep log (TC-61) chỉ có bằng chứng dev |
| B2 sandbox | PASS | 20 chương trình tấn công của QC (PE-02) + 15 ca dev; 240 bài trộn 111,9 s; seccomp amd64 xanh trên CI |
| B3 đồng hồ | PASS | PE-05 TC-27/28 (grace: `+9,2 s` nhận, `+11 s` từ chối), PE-06 TC-39/40, PE-04 lag mở / đóng 0,4 / 0,5 s |
| B4 điểm | PASS | 12/12 tổ hợp `MCQ_MULTI` khớp tay; câu `void` +1 điểm đúng; `ALL_OR_NOTHING`; chấm lại làm điểm giảm đúng 3 SV; làm tròn một lần (20 lượt (a) ≠ (b)): chỉ dev |
| B5 tự công bố | PASS | `PUBLISHED` sau đóng 4,5 s (F19) / 10,3 s khi ép nộp (grace); outbox 1, thông báo 12 (SV có lượt), hoãn / bỏ hoãn đúng; `kill -9` worker, `docker pause` Redis: chỉ dev |
| B6 phân quyền | PASS | ADMIN 403 mọi route PE; TA không sửa điểm / regrade / override / hold; SV chỉ lượt của mình (404 người khác); ma trận 4 vai × route ở PE-08 (GV, TA 200; SV, ngoài lớp, ADMIN 403) |
| B7 liêm chính | PASS | PE-07: khoá chat đúng pha (Redis chết vẫn `locked:true`); log chỉ GV; không trừ điểm; A~B 1,000 / A~C 0,051 / A~A' 0,947 (QC tự cài winnowing); cờ lớp 8 bài / 3 chép đúng sau #17c; 1.000 bản ≤ 30 s: dev; mã không rời hệ thống |
| B8 giao diện / 375 px | PASS (một phần) | e2e 443 pass; QC chụp 24 ảnh F19 (`shots/pe09`) gồm 375 px; `AUDIT_SRC` sạch; axe qua `a11y.spec.ts`; axe và 3G chậm do QC tay: chưa |
| B9 seed + F19 | PASS | seed 76 s (≤ 5 phút), lần hai 12 s idempotent, `rc=1` ở production; `matched 30/30`; F19 đi trọn bằng `playwright-cli` (xem C/D) |

## C. "Bạn tự kiểm"
| # | KQ | Bằng chứng |
| --- | --- | --- |
| C1 | PASS | bài code 5 test ẩn (seed): lời giải mẫu 100 % (`reference/verify`); nộp một phần → điểm đúng tỉ lệ trọng số (`ok` 5,00; qua 3/4 test 3,75; WA 0) — PE-08 |
| C2 | PASS (một phần) | SV 375 px (trắc nghiệm) + 1440 (code), **mất mạng 30 s → có mạng lại tự lưu đủ** (ảnh `09`, `10`); "điện thoại thật": chưa (**Q-QC-GATEPE-2** — chủ dự án tự kiểm) |
| C3 | PASS (một phần) | `/me/exam-lock`: `locked:true` trong giờ, `false` sau nộp (Redis chết vẫn đúng); chat thật chờ P3 |
| C4 | PASS | sau đóng: SV thấy điểm + giải thích + "Test ẩn: đạt 2 trên 2", không thấy test ẩn; GV thấy cặp nghi giống (`EXAM_SIMILARITY`) — F19 + PE-07 |

## D. Luồng F19 (chạy tay bằng `playwright-cli`, ảnh `shots/pe09/01…24`)
GV xem ngân hàng + bài thi → SV (iPhone 375 px): màn bắt đầu có câu minh bạch, làm đúng / sai + nhiều đáp án, offline 30 s → tự lưu → SV (1440): "Bài đang mở ở nơi khác" → `Làm tiếp ở đây`, soạn C, `Chạy thử` 2/2, `Nộp lời giải`, `Nộp bài` (hộp xác nhận có số) → đóng lúc 13:02:46 → **`PUBLISHED` 13:02:50,8** → SV `9,00 / 10,00` + giải thích + test ẩn đếm → `Gửi yêu cầu xem lại` → GV "Xem lại điểm" `Sửa điểm` `9,5` + phản hồi → DB `ADJUSTED 9.00 → 9.50` → SV `9,50 / 10,00` "Điểm đã được giảng viên điều chỉnh.". Nhánh lỗi (chưa mở, TA không lên lịch, hết giờ, phúc khảo hết hạn, chat khoá): chấm bằng API ở PE-04…08. Soạn câu qua UI (zip + verify + AI gợi ý): dựng bằng API ở QC; giao diện soạn câu chấm bằng e2e của dev.

## E. Điều kiện PASS cổng
Chưa đủ: A7 FAIL (TC-17), A10 chưa xanh, A9 chưa chạy. Không có đỏ nào được "giải quyết" bằng nới ngưỡng.

## F. Số liệu cho luận văn
- Chấm code: `judge_done` p95 **1,0 s** (60 bài / 5 phút, `JUDGE_PARALLELISM=2`), **2,1 s** khi chạy chung với chat; `Chạy thử` p95 2,1–2,6 s; 240 bài trộn 111,9 s; RAM `judge` rảnh 54 MiB.
- Điểm: 24 + 12 + 30 lượt khớp tay; độ giống A~B 1,000 / A~C 0,051 / A~A' 0,947 (hệ thống 0,986); công bố +4,5 s sau đóng.
- Tự lưu 300 VU: p95 67 ms, 0 lỗi / 27.066 yêu cầu; bắt đầu làm bài @17/s: p95 12 ms.
- Điểm thi không qua LLM: `llm_audit` = 0 sau phúc khảo / chấm.

## G. Dọn dẹp
`down -v` stack QC, container `judge`/`qcpg`, Chrome / phiên `playwright-cli`, `docker volume prune -f` (làm ở cuối lượt).

## Câu hỏi cho BA / PM
- **Q-QC-GATEPE-1** — seccomp amd64: đã có bằng chứng CI (job `Judge attacks` xanh) — *đề nghị đóng*.
- **Q-QC-GATEPE-2** — điện thoại thật: chủ dự án tự kiểm — *chờ PM*.
- **Q-QC-GATEPE-3** — TC-17 ngưỡng +20 % trên provider `fake`: đổi sang ngưỡng tuyệt đối / đo bằng provider trễ thật? — *chờ BA / PM*.

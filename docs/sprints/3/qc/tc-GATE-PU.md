# QC test case — GATE-PU (cổng nghiệm thu phase PU + "Bạn tự kiểm")
Nguồn: `docs/phases/PU.md` mục "Cổng nghiệm thu" và "Bạn tự kiểm" (nguyên văn, từng dòng → bước đo được) + `FEAT-ui-foundation/US.md` US-PU-05 AC10–AC12. Hộp đen. Chạy ở gốc worktree `TA_Agent_v2-s3` (nhánh `sprint/3-pu-p1`) **sau khi 5 story PU có `report-US-PU-0N.md` PASS**. Máy QC: `pnpm`, Chrome for Testing, `gh`, Docker (colima) + `go` (cho contract test). Mọi TC một kết luận PASS/FAIL; thiếu công cụ → FAIL "KHÔNG KIỂM ĐƯỢC". Số đo ghi vào `report-GATE-PU.md`.

| TC-id | PU.md | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- |
| TC-GATEPU-01 | Cổng #1 | `pnpm -C frontend lint && pnpm -C frontend build; echo rc=$?` (QC tự chạy, `pbuild` và `gbuild`) | `rc=0` cả hai; không `Failed to load font`; không cảnh báo mới so với `audit-baseline.md` |
| TC-GATEPU-02 | Cổng #2 | `bash scripts/ui-antipatterns.sh; echo rc=$?`; `--selftest`; `bash scripts/lint-selftest.sh` | `rc=0`; 19 dòng `✓`; `19 / 19`; `7 / 7`; `git status --short` rỗng; `ui-allow:` ≤ 10 |
| TC-GATEPU-03 | Cổng #3 | `gbuild && $PW ui-foundation.spec.ts; echo rc=$?` + QC đi tay `/chat`, `/threads` bằng bàn phím và ở 640 px (TC-PU05-09…11) | `rc=0`; bàn phím đi hết hai màn; 640 px dùng được |
| TC-GATEPU-04 | Cổng #3 (cả bộ) | `$PW --grep-invert @real; echo rc=$?` (7 tệp: `tokens`, `dev-ui`, `data-layer`, `shell`, `ui-foundation`, `visual`, `a11y`) | `rc=0`; số test chạy ≥ tổng các story; 0 skip không lý do |
| TC-GATEPU-05 | Cổng #3 (`@real`) | stack sprint 2 (`docker-compose.test.yml`, 2 gateway) + `$PW --grep @real` (ETag, 409, 422, cursor, SSE thật, `useJob`, 3 tab) | `rc=0`; không có `@real` nào bị bỏ qua (QC chạy tay, vì CI không dựng stack) |
| TC-GATEPU-06 | Cổng #4 | `$FE exec lhci autorun; echo rc=$?` (bảng số đo TC-PU05-21) | `rc=0`; 7 URL × 3 lần; LCP ≤ 2,5 s, CLS ≤ 0,1, TBT ≤ 200 ms, JS ≤ 250 KB; vượt → FAIL kèm số đo thực (không nới) |
| TC-GATEPU-07 | Cổng #5 | `cd backend-go && go test -count=1 ./internal/contract/...; echo rc=$?`; `git log --grep='^PU:' --name-only` có `backend-go/`? | `rc=0`; `0` commit PU đụng `backend-go/` |
| TC-GATEPU-08 | US-PU-05 AC12 | chuỗi AC12 một lệnh: `$FE lint && gbuild && … && lhci && go test …; echo rc=$?` | `rc=0` (một lần, liền) |
| TC-GATEPU-09 | CI | `gh run list --workflow ci.yml --branch sprint/3-pu-p1 --limit 1` → `headSha` = `git rev-parse origin/sprint/3-pu-p1`, `conclusion=success`; 3 run `ci/ui-drift` đã chấm (TC-PU05-32…35) | Khớp; CI xanh ở HEAD; cổng chứng minh được **đỏ** khi vi phạm |
| TC-GATEPU-10 | Cổng #không vỡ mock | `audit.mjs` bốn vai + spec; `sweep.mjs only:'student'`; `proto-curl.sh all` (`F=:3300`) so `audit-baseline.md` | FAIL 0 mỗi lượt; PASS ≥ 680 (SV 165, GV 170, TA 106, Admin 54, spec 185); `FORBIDDEN`=0; ≥ 497 PASS; `audit-log.md` có dòng của **mỗi** story PU và dòng cuối của cổng |
| TC-GATEPU-11 | Tự kiểm 1 | **T** mở `/dev/ui` cạnh `docs/design/edupilot-ui-v3.html` (cùng 1440, cùng zoom); đối chiếu 6 vùng: nút, trường, bảng, chip, dialog, thông báo; ảnh song song | Cùng "giọng" thị giác: chữ, đường kẻ, bo, khoảng trắng, đỏ-là-tín-hiệu; mô tả từng khác biệt nếu có; kết luận "có/không" kèm ảnh |
| TC-GATEPU-12 | Tự kiểm 2 | **T** trả lời 10 câu cuối `docs/design/AGENT_PROMPT.md` cho `/dev/ui` và 6 route AC1 | Mọi câu "có"; câu "không" → FAIL, ghi câu số mấy |
| TC-GATEPU-13 | Tự kiểm 3 | **T** + **A** đếm card lồng card; nút đỏ đặc / vùng làm việc; chỗ đỏ không có nghĩa; diện tích đỏ (canvas) ở `/dev/ui`, `/chat`, `/threads`, `/inbox`, `/gradebook`, `/settings/llm` | 0 card lồng card; ≤ 1 nút đỏ đặc / vùng; mọi chỗ đỏ thuộc ba nghĩa (đang ở đâu / cần hành động / đã sửa-xác nhận); diện tích đỏ < 8 %; bảng đếm theo màn |
| TC-GATEPU-14 | Tự kiểm 4 | **T** `/chat` ở 375 × 330 giả lập (bàn phím ảo) và, nếu có, điện thoại thật | Ô nhập vẫn với tới; không bị che; ghi kiểu máy / "chỉ giả lập" |
| TC-GATEPU-15 | Tự kiểm 5 | **T** đọc to mọi chuỗi tiếng Việt màn Sinh viên (7 route) | Không từ kỹ thuật; ghi danh sách chuỗi nghi ngờ nếu có |
| TC-GATEPU-16 | Luận văn | `docs/sprints/3/qc/shots/before/` (12) + ảnh sau cùng route/bề rộng; `report-GATE-PU.md` | 12 cặp trước/sau đặt cạnh nhau (`shots/after/`, QC chụp lại bằng cùng cách), điểm Lighthouse và axe ghi số — phục vụ mục "Ghi cho luận văn" |
| TC-GATEPU-17 | Dọn | không còn `next start`, Chrome, container; `git status` sạch (tệp thử đã xoá) | Không tiến trình / container QC sót; cây git chỉ có artefact QC |

## Câu hỏi cho BA / PM
- **Q-QC-GATEPU-1** — PU.md "dùng `/chat` trên điện thoại thật": QC chỉ có giả lập. Chấp nhận "giả lập 375 × 330" cho FAIL/PASS, còn "thiết bị thật" là phần chủ dự án tự kiểm? — *chờ trả lời*.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo PU.md + US.md (FEAT-ui-foundation, APPROVED 2026-10-03).

Tổng: 17 TC.

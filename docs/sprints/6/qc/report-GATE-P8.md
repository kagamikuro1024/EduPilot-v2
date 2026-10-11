# Báo cáo cổng P8 (pha 2) — **ĐẠT CÓ ĐIỀU KIỆN**
HEAD `4418da3`, stack dev dùng chung (docling thật, profile `ingest`). Log: `gate-p8-final-timing-run.log`.
**Điều kiện đóng:** (a) PM/BA chốt ngưỡng 4 s/trang cho bản scan — đo được 4,52 s; (b) CI ở HEAD xanh.

## Số chính
| Chỉ số | Giá trị | Ngưỡng |
| --- | --- | --- |
| `GATE_DOCLING=1 bash scripts/gate-p8.sh` | 8/8 PASS, `GATE P8: PASS` rc=0 (vet 1, race 38, `TestAnswerKeyNeverRetrieved` 3, contract 39, feed ICS `BEGIN:VCALENDAR`, Playwright documents+library+calendar 16, antipatterns, docling thật 19 s) | PASS |
| `GATE_DOCLING_TIMING=1` (US-P8-01 AC17) | Quy chế (scan) 5 trang 23 s = **4,52 s/trang**; Forecasting 24 trang 20 s = 0,84; Network Security 141 trang 109 s = 0,78 | scan ≤ 4, chữ ≤ 1 — scan **FAIL** (PoC 3,1 s; máy dev chậm hơn), chữ PASS |
| Seed tài liệu | `docs=5 READY=5 answer_key=1 shared=4 reembedded=0 events=2/1` | khớp |
| `ANSWER_KEY` không bị truy xuất | `TestAnswerKeyNeverRetrieved` PASS; `CANARY-7Q2X` = 0 ở chat/thread | PASS |
| CI ở HEAD | ĐỎ (calendar 401 trong CI; xem `report-GATE-P3.md`) — `TestICSTokenOnlySelf`, `TestCalendarUnionSources` thuộc P8 | xanh — **CHƯA ĐẠT** |

## Phạm vi đã chấm / chưa
- A1–A3 và B (cổng tổng hợp): PASS như bảng trên (A1 dạng `-tags integration` qua `strict_test` của gate).
- Hợp đồng route, 375 px, UX, từ kỹ thuật, phân quyền, idempotency, ICS riêng tư của từng story: đã nghiệm ở `report-US-P8-01/02/03.md` (PASS); lượt này chỉ chạy lại bằng cổng + Playwright, không lặp từng TC.
- Chưa chạy lại: E-series bằng tay trên bản stack mới, phá cố ý cổng (nhánh FAIL/SKIP).
- BUG-1 của P8-03 (intent "tuần tới/N ngày tới", dev sửa `c5d42d3`) **chưa chấm lại** — mở.

## Lỗi / nợ
1. Ngưỡng 4 s/trang bản scan (4,52) — hoặc nới AC, hoặc chạy máy mạnh hơn; QC không tự nới.
2. CI đỏ (calendar 401).
3. P8-03 BUG-1 chờ chấm lại.

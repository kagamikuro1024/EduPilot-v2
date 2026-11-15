# Sprint 9.5 — báo cáo NFR T1 và đo LLM thật

**Máy đo:** MacBook Apple Silicon 18 GB, Colima 6 CPU / 10 GB, k6 trong VM 1,5 CPU. Mọi số bên dưới là số đo thật trên máy này; tệp gốc ở `benchmarks/reports/` (mỗi tệp có `meta`).
**Mã:** nhánh `sprint/9.5-nfr` = 9.5 + `bench/ep-qc` + `main` ff00b5aa (có P-11-21, P-11-23). Việc đo LLM thật và 1.000 bài chạy trên stack `ep-qc` / `ep-t1` dựng từ chính mã này.
**LLM thật:** 9router trên Mac (`http://127.0.0.1:20128`, model `ag/gemini-3.8-flash`, `rpm_limit` 20), không dùng khoá Gemini trực tiếp. Embedding trên ep-qc là nhà cung cấp GIẢ (xem Giới hạn).

## 1. Kết luận SLO (`SYSTEM_DESIGN.md` §5)

| SLO | Số đo | Kết quả | Tệp |
| --- | --- | --- | --- |
| (a) đọc p95 ≤ 300 ms (300 người duyệt + 300 SSE + heartbeat 10/s, 5 phút) | 11,5 ms (×1) · 11,4 ms (×2) | ĐẠT | `load-a*.json` |
| (a) ghi p95 ≤ 500 ms | 4,9 ms | ĐẠT | cùng tệp |
| (a) sự kiện SSE đầu p95 ≤ 300 ms | 97,5 ms (×1) · 144,2 ms (×2) | ĐẠT | cùng tệp |
| (a) 5xx < 0,5 %; STUDENT gọi sổ điểm 100 % 403 | 0 · 300/300 403, 0 lần 200 | ĐẠT | cùng tệp |
| (b) chat: sự kiện đầu p95 ≤ 300 ms · TTFT cache ≤ 1,5 s · TTFT RAG ≤ 4 s | 66,9 ms · 103,8 ms · 633,9 ms (provider giả 5–15 s) | ĐẠT | `load-b-*.json` |
| (c) TTFT chat lúc chấm xấu đi ≤ 20 % | nền 410 → chấm 381 ms (−7 %) | ĐẠT | `load-c-baseline*.json`, `load-c-grading*.json` |
| **(c) chấm 1.000 bài thời gian thật ≤ 3 h, LLM thật** | **1000/1000 trong 1,68 h**, 0 FAILED, 0 mồ côi, 0 chấm hai lần | **ĐẠT** (lần 2; lần 1 hỏng, mục 2) | `load-c-1000-9router-run2.json` |
| (d) dashboard / hồ sơ 360 / sổ điểm p95 ≤ 3 s (1.000 SV) | 18,3 ms | ĐẠT | `load-d.json` |
| Baseline 13 endpoint tuần tự × 200 | đọc ≤ 11,0 ms; ghi lớn nhất 327,5 ms | ĐẠT | `load-baseline-p95.json` |
| Chaos: `kill -9` worker giữa 150 bài; Redis tắt 70 s | 150 = 150, 0 mồ côi, 0 chấm hai lần, `XPENDING` 0 | ĐẠT | `chaos.json` |
| Lighthouse LCP ≤ 2.500 ms | `/` 1.574, `/chat` 1.554 đạt; `/threads` 4.896, `/inbox` 6.717, `/gradebook` 5.887, `/me` 6.256, `/settings/llm`, `/assignments/[id]` trượt | **KHÔNG ĐẠT** (TL-2f, không nới ngưỡng) | `lighthouse.json` |
| EXPLAIN 10 truy vấn | một truy vấn 13,1 ms, chín truy vấn ≤ 1,3 ms | không cần chỉ mục | `explain.json` |

Chi tiết các dòng (a)(b)(d) và chaos: `handoff/dev-lat-2.md`. Dòng (c) 1.000 bài trong tệp đó (chưa có kết quả) được báo cáo này thay thế.

## 2. Chấm 1.000 bài qua 9router (US-NFR-04 AC9)

Cấu hình: ep-t1, 20 lớp × 50 SV, 5 luồng chấm, mỗi bài 2 lượt LLM, `ag/gemini-3.8-flash`, rpm 20. Tiếp từ đầu cả hai lần, không tiếp lần cũ. Lần cũ 235/1000 dùng Gemini trực tiếp, đã dừng theo chủ dự án (`load-c-1000-dung-theo-chu-du-an.json`): không phải số đo.

| | Lần 1 (`…-run1-hong.json`) | Lần 2 (`…-run2.json`) |
| --- | --- | --- |
| Kết quả | **KHÔNG ĐẠT**: 322 DONE · 678 FAILED | **ĐẠT**: 1000 DONE · 0 FAILED |
| Thời gian | 0,55 h (dừng khi hết hàng) | 1,68 h (101 phút), SLO 3 h |
| Tốc độ | 10 bài/phút trong 20 phút đầu | 10 bài/phút ổn định |
| Mồ côi / chấm hai lần | 0 / 0 | 0 / 0 |
| Mã | bench 8719db26 (main 35591702 + 9.5, chưa có P-11-23) | main ff00b5aa (có P-11-23) |
| Mạng | không ghi lại | en0, Tailscale tắt |

**Nguyên nhân lần 1 (đã chứng minh):**
1. Phút 21 (05:21–05:22Z; `watchdog.log`: `2026-11-15 12:21:31 restarted`, giờ +07) watchdog v2 của PM khởi động lại 9router vì 3 lần thăm dò `/v1/models` quá 15 s. 9router lúc đó BẬN (rpm 20), không chết; bị giết giữa chừng. Không phải VPN.
2. `llm_audit` ep-t1 (trích trước khi xoá): 18 lỗi `NETWORK` + 2 `TIMEOUT` (120 s) ⇒ mạch LLM mở (5 lỗi liền, `LLM_BREAKER_OPEN` 30 s) ⇒ 4.545 lượt gọi bị chặn `circuit_open` tức thì. Độ trễ TB lúc đó 27 s (ep-qc: 11 s).
3. Hàng đợi chấm: 3 lượt, backoff 5 s + 20 s = 25 s < 30 s mở mạch ⇒ 500 bài hết lượt TRONG LÚC mạch còn mở ⇒ `FAILED` vĩnh viễn trong 1 phút.

Nhà cung cấp mất tạm thời là ca hệ thống phải chịu được, nên đây là **lỗi mã** (luật 12), không chỉ lỗi vận hành. Sửa ở **P-11-23** (`fix/dr-06`, main ff00b5aa): mất nhà cung cấp (`llm.IsOutage`: mạch mở, mạng, quá hạn, 5xx, 429) hoàn lượt thử, hoãn 35 s + nhiễu, tối đa 20 lần mỗi việc; áp cho chấm, reflect, trích công thức, trích đề / sinh câu hỏi, gợi ý câu hỏi, nhúng (gồm reindex cả lớp, D-R-08). Test: 12 lỗi liên tiếp > 3 lượt ⇒ 0 FAILED (`TestOutageDoesNotConsumeAttempts`). Handoff: `docs/sprints/11/handoff/dev-P-11-23.md`.

**Lần 2:** 0 lượt FAILED, không có dòng khởi động lại 9router trong khoảng 13:26–15:07 (+07); `watchdog.log` chỉ có `13:13:41 watchdog v4 (port-gone only)`. Stack demo `edupilot-*` dừng trong lúc đo, đã start lại, `https://localhost` trả 200 lúc 08:08Z.
`[GIỚI HẠN]` Độ trễ TB mỗi lượt LLM của lần 2 KHÔNG đo được: `ep-t1` bị `down -v` trước khi trích `llm_audit`. Tốc độ 10 bài/phút × 5 luồng ≈ 30 s/bài khớp độ trễ mỗi lượt ≈ 12–13 s của R3 `[SUY LUẬN]`.

## 3. Đo LLM thật R1–R6 qua 9router (US-NFR-08), ep-qc, 2026-11-15

Tất cả rc=0. Tệp ở `benchmarks/reports/real-llm/`.

| | Số đo | Ghi chú |
| --- | --- | --- |
| **R1 trích công thức** (`extract.json`, 2 quy chế × 5 lần; ĐÍNH CHÍNH P-11-25(3)) | 10/10 SUCCEEDED; 9–15 s mỗi việc; đủ thành phần điểm, `weights_sum` 100, 0 luật bịa, mọi luật `rules_ok`; **trích dẫn 10/10 và 7/7 đúng trang ở cả 10 lần** | lần đo đầu ghi «5/10 lần mọi trích dẫn not_found» — LỖI PHÉP ĐO: docling dựng đoạn thành mục «- 1.2. …», LLM trích đúng chữ đó, script so với tệp gốc không có «- ». Đã sửa script và chạy lại; sản phẩm vốn không ghi `CITATION_NOT_FOUND` ở các lần đó |
| **R2 chấm bài eval ×3 vòng** (`grade-eval.json`, 12 bài) | MAE tổng 0,792 → 0,729 → 0,688; MAE tiêu chí ≈ 0,25; QWK 0,905–0,913; trong ±0,5: 6/12 → 7/12 → 9/12; 0 bài hỏng | ổn định: 7/12 bài giống hệt giữa các vòng, lệch TB 0,229, lớn nhất 1,0. Trễ LLM TB 12,5–13,3 s, p95 17–19 s; cả lô 91–102 s |
| **Chèn lệnh** (6 biến thể mỗi vòng) | mỗi vòng CHỈ 1/6 biến thể (S6) gắn cờ `INJECTION_SUSPECTED` (điểm 5,0 / 4,5 / 4,25 vs đúng 4,5); 5 biến thể còn lại không cờ | điểm 5,0–5,5 so với mốc 5,75: không bị đẩy lên, nhưng cờ không bắt được 5/6 `[GIỚI HẠN]` |
| **R3 chấm BT03** (`grade-bt03.json`, 16 bài, ĐÍNH CHÍNH P-11-25(2)) | **8 bài AI chấm** (8 bài còn lại là kết quả REVIEWED có sẵn từ seed, attempts 0, KHÔNG phải đầu ra AI, đã loại): MAE tổng 1,188, MAE tiêu chí 0,297, QWK 0,916, trong ±0,5: 2/8; 0 hỏng. S5 (lạc đề), S4 (rỗng) đúng 0; S7 mốc 7,0 AI chấm 10,0 (cao 3 điểm) | báo cáo ban đầu ghi MAE 1,812, QWK 0,764, 2/16 và «lọt chèn lệnh 8,0 vs 4,5» — SAI, do 8 hàng seed lẫn vào; hàng «S6 8,0» là kết quả seed, không phải AI |
| **R4 chat** (`chat.json`, 26 câu) | ý định đúng 24/26; sự kiện đầu p95 14 ms; TTFT p50 21 ms, p95 4.217 ms; hoàn tất p95 4.600 ms; 0 lỗi | chỉ 2/26 câu gọi LLM; còn lại trả tất định hoặc từ chối (xem Giới hạn) |
| **R4 mask** (`mask.json`, đo lại, DEF-R9.5-10) | lượt 2: 31 probe, **24 tới mô hình**, giá trị được yêu cầu nhắc lại 35, **khôi phục 35/35 (100 %)**, còn sót placeholder 0, đổi / bỏ 0; nhóm giữ 23/23, nhóm biến đổi (hoa, thường, dịch, đánh vần, ngoặc, viết tắt, tên riêng) 12/12; che thiếu 0; lộ qua đánh vần 0. 7 probe P17–P23 (tên SV khác) bị chặn `OTHER_PERSON` trước mô hình, tính riêng | **đạt ngưỡng ≥ 95 % (n = 24 probe)**; chi tiết, lượt 1 và sửa bộ dò ở mục 3a |
| **R5 Threads + khủng hoảng** (`threads.json`, `crisis.json`, đo lại, DEF-R9.5-11) | Threads: **5/5 bài có AI trả lời**, sau 6,1–12,1 s (p50 9,1 s), cả 5 có trích dẫn tài liệu lớp; 1/5 (Phishing qua email) trả lời một phần, nói rõ tài liệu chưa có phần nhận diện email lừa đảo. Khủng hoảng: **4/4** nhận `CRISIS`, đúng câu an toàn, không sinh chữ, không mở phiếu; sự kiện đầu p50 7 ms, max 11 ms | đạt cỡ mẫu (≥ 3 mỗi việc). Threads chạy với nhúng thật `gemini-embedding-001` chỉ cho lớp 761987 (P-11-37); KHÔNG đo độ đúng nội dung |
| **R6 gợi ý rubric / tóm tắt AI** (`misc.json`) | rubric 3/3 có 4 tiêu chí, tổng 10,0 (10,4–12,9 s); tóm tắt hồ sơ SV 3/3 (6,8–8,7 s), 0/3 chứa tên / MSSV; mẫu đầu đúng số liệu (11/11 buổi, 300 phút/tuần, 12 lần phát biểu) | đạt |
| **Guidance** (`guidance-20261115-112429.json`, BT03) | reflect SUCCEEDED, 1 bản nháp; áp phiên bản 1 (3 mục); 0 bản chấm cũ; MAE tiêu chí 0,25 (n=8) trước khi áp | chưa chấm lại sau khi áp nên chưa có số «sau» |

### 3a. Đo lại R4 mask và R5 (ep-qc, 2026-11-15 21:10–21:28 UTC, DEF-R9.5-10/11)

Bộ probe mới `benchmarks/real-llm/data/mask_probes.json`: mỗi probe là câu có ý định sinh chữ (`PERSONAL_*`, `WHAT_IF_GRADE`, `UPCOMING_EVENTS`) và yêu cầu mô hình nhắc lại tên / MSSV / email, nên placeholder thật sự tới mô hình (bộ cũ chỉ M1 tới được). Một giá trị tính là khôi phục khi giá trị thật có trong câu trả lời; còn sót khi thấy chuỗi placeholder; đổi / bỏ khi không thấy cả hai (gateway thay placeholder lạ bằng «bạn»). Sinh chữ qua `QC 9router` / `ag/gemini-3.8-flash` (20 rpm dùng chung với QC, giãn 10 s mỗi probe); `meta.config` của từng tệp ghi tuyến đọc từ API ADMIN.

| Lượt | Tệp | Probe | Tới mô hình | Khôi phục | Giữ | Biến đổi | Còn sót | Đổi / bỏ |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 (21:10–21:16) | `mask-luot1.json` | 24 | 17 | 24/25 (96 %) | 17/17 | 7/8 | 0 | 1 (P15) |
| 2 (21:20–21:27) | `mask.json` | 31 | 24 | 35/35 (100 %) | 23/23 | 12/12 | 0 | 0 |

- **Lượt 1 không đạt** vì chỉ 17 probe tới mô hình (cần ≥ 20), không phải vì tỉ lệ. P15 (chỉ gọi bằng tên riêng): mô hình trả «Mình chưa tìm thấy nội dung này trong tài liệu của lớp.» và không gọi tên, nên tính là bỏ, không phải lộ.
- **P17–P23 bị chặn `OTHER_PERSON` trước mô hình** ở cả hai lượt («Mình chỉ xem được dữ liệu của chính bạn.», `masked_count` 0, không có lời gọi LLM). Đây là hành vi đúng của tường lửa: câu nhắc tên SV khác trong lớp. Bảy probe này tính riêng (`summary.refused_other_person`), không gộp vào tỉ lệ khôi phục. Lúc soạn, phân loại offline không có danh sách lớp nên không thấy trước điều này; lượt 2 thêm P25–P31 dùng dữ liệu của chính SV để đủ ≥ 20 probe tới mô hình.
- **Bộ dò đánh vần — ba phiên bản, số cuối là v3.**
  - v1 (lúc chạy lượt 1): bỏ mọi ký tự không phải chữ trong câu trả lời rồi tìm chuỗi chữ của tên thật, nên tên thật khôi phục nguyên dạng cũng bị tính là lộ. Lượt 1 báo `spell_leak` P06 = true, trong khi câu trả lời gốc là «Mình chưa tìm thấy nội dung này trong tài liệu của lớp. Tuy nhiên, về điểm số, điểm giữa kỳ của Bùi Thanh Khải là 7,5.» (mô hình không đánh vần; tên là do gateway khôi phục placeholder).
  - v2 (lúc chạy lượt 2): chỉ bắt chữ-chữ-chữ nối bằng gạch, chỉ xét probe `spell`. QC chỉ ra v2 bỏ sót chữ cách khoảng trắng / dấu chấm: bỏ dương tính giả bằng cách thu hẹp là làm yếu phép đo rò rỉ.
  - **v3 (PM, chặt nhất; chấm lại OFFLINE trên câu trả lời đã lưu, không gọi mô hình — `real_llm.py mask-rescore`):** mọi chuỗi ≥ 3 ký tự đơn (chữ / số) ngăn bởi khoảng trắng, gạch, chấm, phẩy, gạch dưới, gạch chéo, sổ đứng; bỏ dấu, không phân biệt hoa thường; lộ khi chuỗi ghép chứa giá trị PII của probe, một từ ≥ 3 ký tự của tên (ví dụ chỉ đánh vần họ), hoặc phần trước @ của email; xét **mọi** probe chứ không chỉ probe `spell`. Tên thật viết liền không phải chuỗi ký tự đơn nên P06 không dương tính. Tự kiểm 12 trường hợp: `benchmarks/real-llm/test_spell.py`.
  - Kết quả v3: lượt 1 `spell_leak` [P06] → **[]**, lượt 2 [] → **[]**. Trong cả 55 câu trả lời của hai lượt không có chuỗi ký tự đơn nào (mô hình không đánh vần ở P06 lẫn probe khác). Giá trị trước khi chấm lại giữ ở `spell_leak_before` từng probe và `summary.spell_leak_before`.
- `under_masked` (che thiếu) từ lượt 2 chỉ xét probe có bước sinh chữ; lượt 1 liệt kê P17–P23 vì bị chặn trước khi che (không phải che thiếu).
- **Nhúng thật (P-11-37):** `gemini-embedding-001` chỉ cho lớp 761987 (reindex 127/132 đoạn có vectơ), từ 19:01:31 UTC đến khi trả tuyến. `llm_audit`: 16 dòng = 12 thành công + 4 `rate_limited` × 3 lần thử ⇒ **24 request HTTP** (trần 300). Mask và khủng hoảng không gọi nhúng (ý định do luật quyết). Sau lượt đo: xoá provider `gemini-embed-eval`, reindex lớp 761987 bằng nhúng giả (`fake-eval`). Sáu tuyến sinh chữ không đổi (vốn là `QC 9router`). Tuyến EMBEDDING trước lượt đo trỏ `QC lỗi embed` / `bad-embed` (provider hỏng của ca thử QC, nay đã tắt): trả lại bị từ chối 422 `provider_disabled`, nên EMBEDDING để ở `fake-eval`; QC quyết có đặt lại hay không.
- Threads chưa đo độ đúng nội dung (không có mốc); số trên là AI có trả lời, có trích dẫn, và thời gian.

## 4. Decision model trước / sau (`decision-before-after.md`) — ĐÍNH CHÍNH P-11-25(1) / P-11-27

Chạy trên ep-qc, provider decision GIẢ (p50 426 / p95 682 ms, log-chuẩn). **Lần đầu (2026-11-15 12:43) KHÔNG dùng được:** 65/90 yêu cầu chat mỗi cột trả HTTP 429 (`CHAT_RATE_PER_MIN` = 20 mỗi người mỗi phút; bench dùng một sinh viên gửi 90 câu liên tục và không chờ `Retry-After`), nên mẫu TTFT chỉ 25 điểm và kết luận «CHAT_ROUTING Bật được» sai. Sửa `benchmarks/decision/run.py` (chờ `Retry-After`, ghi số lần chờ) rồi đo lại đủ 90 câu mỗi cột, 0 lỗi (2026-11-15 15:47):

| Cổng | Kết luận | Số liệu |
| --- | --- | --- |
| CHAT_ROUTING | **Giữ tắt** | TTFT p95 ×1,53 (> 1,20); độ chính xác 0,667 → 0,933 |
| THREADS_PRECHECK | Bật được (về số đo) | chặn nhầm 0/20 (lần đầu 1/20, sát ngưỡng); lọt 9 → 0; nhưng đăng bài p50 3,6 s so với 6,8 ms khi tắt (xem P-11-27) |
| INJECTION_FLAG | Bật được (về số đo) | cờ đúng 1/6 → 6/6; cờ nhầm 0/6 |

**Quyết định (PM, P-11-27):** cả ba cổng GIỮ TẮT mặc định; số đo là mô phỏng.
`[GIỚI HẠN]` Nhãn trả theo bản phát lại dựng từ chính bộ nhãn, nên độ chính xác không phải của Gemini; trễ decision là mẫu log-chuẩn của provider giả, không phải trễ mạng. Cache theo lớp có thể làm lượt «bật» có nhiều câu cache hơn lượt «tắt». Cột «thật» (AC6) chưa chạy.

## 5. Giới hạn đã biết

- **Embedding trên ep-qc là giả** (`fake-embed-qcR`): ngưỡng `RAG_SIM_FLOOR/CEIL` không khớp ⇒ truy xuất ra `LOW_SCORE`, trợ lý từ chối. R4 (câu hỏi giáo trình) vì vậy KHÔNG đo được chất lượng trả lời có truy xuất. R5 Threads đã đo lại với nhúng thật cho riêng lớp 761987 (mục 3a, P-11-37). Việc xếp hàng: **P-11-19** (hiệu chỉnh ngưỡng theo model nhúng thật).
- Độ trễ LLM lần 2 của 1.000 bài không có (mục 2).
- Cờ chèn lệnh R2 chỉ bắt 1/6 biến thể: ScanInjection có 6 mẫu từ khoá, 5 biến thể v1–v5 viết KHÔNG từ khoá (0/5 khớp); phát hiện ngữ nghĩa là việc của INJECTION_FLAG (đang tắt). Điểm 5 biến thể 5,0–5,5 so với mốc 5,75 không chèn: hai lượt chấm không bị dắt.
- R3 (BT03, đã đính chính): chỉ 8 bài AI thật; S7 chấm cao 3 điểm so với mốc. Bộ sạch EVAL03 (12 bài) là số tham chiếu. Không có bằng chứng chèn lệnh lọt điểm.
- Lighthouse LCP trượt 6/8 route (TL-2f), TL-2d (Redis tắt ⇒ chat treo ≈ 19 s), TL-2e (đăng bài chờ embedding ≈ 317 ms) vẫn mở; xem `handoff/dev-lat-2.md`.
- 9.5 không đổi mã sản phẩm ngoài P-11-23 (hậu quả của bench); các số hiệu năng khác chạy trên mã trước đó.
- Mạng lần 1 không ghi lại; lần 2 Tailscale tắt. Máy 18 GB / VM 10 GB, 9router chung máy với stack: độ trễ phản ánh máy này, không phải mạng từ Việt Nam.

## 6. Phụ lục — R1–R6 bằng Gemini trực tiếp (cũ)

`benchmarks/reports/real-llm/lan-1-khong-hop-le/`: lần đo ngày 2026-11-14 bằng `gemini-3.1-flash-lite` (khoá của chủ dự án, rpm 5). Giữ làm phụ lục, KHÔNG dùng làm số đo: chuỗi một model, ngân sách < 200 lời gọi/ngày, và `llm-audit-tom-tat.txt` ghi `circuit_open` 126 lượt chấm + 19 `RATE_LIMIT`. Số liệu thay bằng mục 3. Khác biệt chính: lần cũ Threads có AI trả lời sau 10,1 s (embedding thật qua Gemini), lần này không (embedding giả).

## 7. Tệp thay đổi so với `main`

`benchmarks/reports/**` (số đo + `decision-before-after.{json,md,svg}`, `load-c-1000-9router-run{1-hong,2}.json`, `real-llm/**`), `benchmarks/decision/run.py` (tag, `jwt_sub()`), `benchmarks/load/grade-all-t1.mjs` (`GRADE_PROVIDER`, `OUT_NAME`), `benchmarks/real-llm/**`, `deploy/vps-llm.mjs`, `scripts/seed-t1.mjs`, `docs/sprints/9.5/**`.

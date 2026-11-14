# Đưa decision model Jev (TypeSafe) vào EduPilot?

**Câu hỏi.** Jev là gì, nó cải thiện được điểm nào của EduPilot, có khớp luật repo không, và nên làm hay không.

**Kết luận.**
- Jev là **System One model**, không phải LLM:
  - đầu vào là `state` cộng các câu hỏi có kiểu (Choice / Score / Noul / Box);
  - đầu ra là nhãn kèm phân phối xác suất, không sinh chữ;
  - giá $0,042 / 1 triệu token đầu vào, đầu ra miễn phí;
  - trễ đo từ Mac khoảng 0,3 s (p50).
- PoC trên dữ liệu có nhãn của repo, so Jev 1.14 với Gemini 3.8 Flash cùng instructions (bảng mục 4):
  - tường lửa E1: Jev F1 0,974, Gemini 0,963, cả hai chặn nhầm 0;
  - định tuyến ý định: Jev 59/61, Gemini 58/61;
  - chèn lệnh trong bài nộp: regex hiện tại chỉ bắt **1/6**, Jev 5/6, Gemini 6/6;
  - phát hiện khủng hoảng: Gemini 20/20, Jev chỉ **14/20 ở ngưỡng 0,5**, dù xác suất của Jev tách lớp hoàn hảo (AUROC 1,0);
  - trễ p50: Jev ≈ 0,3 s; Gemini 4,2–5,4 s qua 9router, ≈ 1,7 s khi gọi thẳng `3.1-flash-lite`.
- **Không làm trong sprint 11.** Lý do:
  - API không tương thích OpenAI, nên phải mở rộng `internal/llm`: mở lại luật 3;
  - đặt Jev lên đường nóng là mở lại D47;
  - repo đang khoá phạm vi trước buổi bảo vệ.
- Hướng hợp lý sau bảo vệ (M, cần ADR): Jev làm **lớp tín hiệu phụ, chỉ được thêm cờ, không được nới**, ở những chỗ mà trễ và chi phí của Jev có giá trị:
  - kiểm câu trả lời có bám nguồn hay không, cho quyết định escalate;
  - lớp "câu hỏi riêng tư" của tường lửa kênh công khai.
- **Phát hiện phụ, nên làm sớm và không cần Jev:** regex chèn lệnh trong bài nộp chỉ bắt 1/6 biến thể. Gemini qua `internal/llm` (làn BATCH) bắt được 6/6, nên lấp được ngay trong kiến trúc hiện tại.
- Không dùng Jev cho: điểm số, trích công thức, định tuyến thay luật, phát hiện khủng hoảng thay luật.

Độ chắc chắn:
- **cao** về năng lực và ràng buộc (tài liệu chính thức + gọi thật);
- **vừa** về độ chính xác: bộ nhỏ, nhãn do agent viết, mỗi cấu hình chạy một lượt.

## 1. Jev thực chất là gì

| Mục | Giá trị | Nguồn |
| --- | --- | --- |
| Loại | System One model: "trả quyết định có kiểu và xác suất, không sinh chữ, không viết code, không giải thích lập luận" | `docs.typesafe.ai/concepts/system-one`, `/introduction` |
| Phiên bản | `jev-1.14.0` (14/10/2026). Alias `jev-latest` và `jev-preview` đều trỏ về bản này (`GET /v1/models` đo 14/11) | `/models`, `/changelog` |
| API | `POST https://api.typesafe.ai/v1/systemone`, `Authorization: Bearer`. **Không tương thích OpenAI** | `/api/system-one` (OpenAPI 3.1, v0.2.0) |
| Đầu vào | `state` (chuỗi / JSON / mảng; có thể kèm ảnh PNG/JPEG) + `questions{tên: {type, instructions, criteria}}`. Mọi câu hỏi đánh giá song song, độc lập trên cùng `state` | `/introduction`, `/models` |
| Đầu ra | Choice: `choice`, `probabilities`, `confidence`. Score: `score` (kỳ vọng theo mức), `probabilities`, `confidence`. Noul: `noul` ∈ [0,1]. Box: toạ độ. Kèm `usage` và `model` | `/primitives/*`, `/confidence` |
| `confidence` | Tính từ độ tập trung của phân phối (ví dụ Choice 3 lựa chọn: `(3·pmax − 1)/2`), không phải xác suất đúng | `/confidence` |
| Ngữ cảnh | 64k token mỗi request, 32k mỗi câu hỏi. Có header 262k nhưng chậm hơn và kém chính xác hơn | `/models#context-length` |
| Giá | $42 / tỷ token đầu vào ($0,042 / triệu). **Đầu ra miễn phí** | `/models` |
| Giới hạn tốc độ | Theo tổ chức (token/s, request/s), xem ở console. Vượt thì `429`; quá tải thì `529` | `/models#rate-limits`, `/api/errors` |
| Tuỳ biến | Không fine-tune hay LoRA theo khách. Chỉ tuỳ biến qua `state` và `instructions` / `criteria` | `/models#customizing-jev` |
| Điểm yếu tự công bố | Đọc rất sát chữ; **không giỏi số học / đếm / so ngày**; kém khi phải suy luận qua nhiều bước gián tiếp; kém khi `state` nhiều chi tiết thừa; **bị nội dung đối kháng lái được** ("State is data, and Jev does not treat it as hostile by default"); thứ tự lựa chọn của Choice ảnh hưởng kết quả; không sinh chữ | `/model-jaggedness` |
| Dữ liệu | Máy chủ ở Mỹ. "Will not train or fine tune … on Input". Có DPA, danh sách subprocessor. Không lưu dữ liệu (ZDR) chỉ có ở gói enterprise | `typesafe.ai/legal/privacy-policy`, `/legal/data-processing`, `docs.typesafe.ai/legal` |
| SDK | Python / JS / TS. LangChain4j, pydantic-ai có adapter. **Không có SDK Go**, gọi bằng `net/http` | `/sdk`, `docs.langchain4j.dev/…/typesafe` |

Thử thật từ Mac (14/11, khoá trong omp auth, không in):
- `GET /v1/models`: 200 sau 311 ms.
- Một request định tuyến tiếng Việt, 2 câu hỏi: 200 sau 302 ms, `model: jev-1.14.0`, 445 token vào.

## 2. Chỗ có thể cải thiện trong EduPilot

| Điểm | Hiện làm thế nào (mã) | Jev làm gì hơn | Đo bằng gì | Nhận định |
| --- | --- | --- | --- | --- |
| Định tuyến ý định D47 (một lần / tin) | Regex trên chữ bỏ dấu theo thứ tự ưu tiên (`internal/agent/intent.go`), cộng tương đồng embedding cho "cá nhân" (`internal/privacy/classify.go`, `PersonalSimHigh=0.78`) | Bền với cách nói lạ hoặc diễn đạt lại, có `confidence` để biết khi nào không chắc | Độ chính xác trên một bộ **diễn đạt lại giữ riêng**; trễ trước sự kiện SSE đầu | **Không thay.** PoC C: 59/61. Hai câu sai là đúng luật riêng của dự án (#15: "tuần tới" kèm nội dung → `COURSE_QA`), Jev đọc theo chữ. Thêm 0,3 s đi Mỹ trước khi sinh chữ thì mở lại D47(2). Chỉ đáng cân nhắc làm **lớp dự phòng khi luật rơi vào `COURSE_QA` mặc định** |
| Tường lửa kênh công khai (PII) | Regex (MSSV/email/SĐT/CCCD) + danh sách lớp + mẫu câu cá nhân bằng embedding (`internal/privacy/detect.go`, `classify.go`). E1: recall 0,96, chặn nhầm 0 (embedding `fake`) | Câu hỏi riêng tư diễn đạt tự do; tên người ngoài danh sách lớp | E1 (200 mẫu, tập `test`); chỉ số chính là `false_block` ≤ 0,05 | **Có ích làm lớp OR** sau luật, chỉ được thêm chặn. PoC A: F1 0,974, AUROC 0,9997, đủ 4/4 tên ngoài danh sách lớp (S7, đúng chỗ luật đang trượt) |
| Quyết định escalate | `confidence = f(retrieval, groundedness)`. Groundedness đo bằng **trùng từ** (≥ 40 % từ của câu có trong ngữ cảnh, `agent/confidence.go`). Ngưỡng là `courses.escalation_threshold` | Noul kiểu "câu trả lời có suy ra được từ các đoạn nguồn không" (cookbook *Verifying LLM answers against their source*). Bắt được trường hợp trùng từ cao nhưng sai nghĩa | Đường cong E2 escalate-rate / error-rate khi thay groundedness, so trên cùng golden set | **Ứng viên tốt nhất.** Chạy *sau* khi stream xong nên không chạm SSE đầu ≤ 300 ms và không là lời gọi sinh chữ (D47(1) giữ). Cần đo trên E2 trước khi quyết |
| Cờ chèn lệnh / gian lận trong bài nộp | 6 regex bỏ dấu (`grading/injection.go`), chỉ gắn cờ và tô, không đổi đầu vào | Bắt biến thể không có từ khoá (`v1`…`v5` của 9.5) | Recall trên bộ biến thể chèn lệnh + bài sạch | **Lỗ hổng thật, bằng chứng mạnh nhất của PoC:** regex 1/6, Jev 5/6, Gemini 6/6. Chấm bài chạy ở làn BATCH, trễ không quan trọng, nên **lấp được ngay bằng Gemini qua `internal/llm` `Structured`**: không cần nhà cung cấp mới, không cần ADR. Cả hai model đều gắn cờ nhầm bài trống. Không bao giờ là cổng duy nhất (TypeSafe tự công bố điểm yếu "adversarial content") |
| Chấm rubric hai lượt | Lượt 1 cho điểm nháp; lượt 2 chỉ để phát hiện lệch giữa hai lượt (`grading/check.go` `Merged`), mọi phép tính bằng `decimal` | Score theo mức cho từng tiêu chí làm "giám khảo thứ ba" để báo lệch | Đồng thuận (QWK) với điểm người, khi E3 có điểm người | **Không trong sản phẩm.** Score của Jev là một điểm do model đặt; dù chỉ dùng làm cờ cũng mờ ranh giới luật 5. TypeSafe tự nói Score "weak in numerical calibration". Chỉ dùng **offline ở E4** như một bên chấm để so, không bao giờ làm chuẩn E3 (vòng tròn) |
| Trích công thức điểm | LLM `Structured(json_schema)` ra bản nháp, giảng viên xác nhận | – | – | **Không.** Đây là sinh cấu trúc có số; Jev không sinh và yếu về số |
| Chọn model fallback | Chuỗi fallback + cầu dao + hạn mức trong Scheduler (`internal/llm/gateway.go`), tất định | – | – | **Không.** Quyết định vận hành tất định tốt hơn một mô hình |
| Phát hiện khủng hoảng | Regex hai tầng (`intent.go`); test 20 dương / 17 âm khó | Lưới an toàn cho cách nói lạ | Recall trên bộ khủng hoảng | **Không thay.** PoC B: Jev tách hoàn hảo (AUROC 1,0) nhưng **recall 0,70 ở ngưỡng 0,5**, phải hạ xuống 0,1 mới đủ 20/20; 4/6 câu trượt là tiếng Việt không dấu. Gemini 20/20. Regex vẫn là cổng; nếu thêm lưới thì chỉ làm lớp OR ngưỡng thấp và phải có bộ thử lớn hơn |
| Đánh giá E1–E4 (offline) | Script Python chỉ gọi HTTP | Làm "bên phân loại thứ hai" để so trong E4, đo tự nhất quán | E4 | Hợp lệ nếu chỉ dùng để so; không làm chuẩn |

## 3. Khả thi theo luật repo

| Luật | Có khớp không | Hệ quả |
| --- | --- | --- |
| D46: chỉ Go | Khớp. Gọi `net/http` từ Go; không có service mới | – |
| Luật 3: mọi lời gọi qua `internal/llm` "dựng trên `openai-go`, mỗi provider là base URL tương thích OpenAI"; cấm SDK provider | **Không khớp nguyên văn**: `/v1/systemone` không tương thích OpenAI. SDK của TypeSafe không có bản Go, và luật cũng cấm dùng SDK | Thêm loại provider `typesafe` và phương thức `Decide(ctx, state, questions)` trong `internal/llm` (cùng registry, `llm_audit`, cầu dao). Là **thay đổi kiến trúc** → ADR + vá `ARCHITECTURE.md` |
| Luật 11: mọi lời gọi qua Scheduler, có làn ưu tiên | Khớp nếu `Decide` đi qua Scheduler. Token bucket đang tính theo provider (`scheduler/redis.go`) | Thêm `Task` mới (ví dụ `DECIDE`) vào bảng tuyến |
| Luật 4: placeholder quanh mọi lời gọi | Khớp nếu che trước `Decide`. PoC đã chạy trên văn bản đã che | Tên người ngoài danh sách lớp vẫn ra ngoài như hiện nay với Gemini. Chính TypeSafe phát hiện được đúng loại này (S7) |
| Luật 5: tính điểm là code thuần | Khớp nếu **cấm Score / Choice ra điểm** ở `internal/grade`, `internal/grading` | Thêm test chặn import, giống P-10-1 với `internal/quiz` |
| D47 | Ngoài đường nóng (sau khi stream xong, bài nộp, Threads precheck): khớp. Trước khi sinh chữ: **mở lại D47(2)** (phân loại bằng luật + embedding) | Chỉ dùng sau stream hoặc trên đường không phải chat |
| Chi phí | ≈ 474 token / request ở PoC A → **$0,00002 / request, ≈ $2 / 100.000 request** | Không đáng kể ở T1 |
| Phụ thuộc ngoài | Thêm nhà cung cấp thứ ba. Hạn mức chưa biết; có `429` / `529`; alias tự đổi bản | Ghim `jev-1.14.0`; mỗi điểm dùng phải định rõ mất Jev thì **mở** hay **đóng**. Tường lửa / cờ: mất Jev thì quay về luật, không nới |
| Dữ liệu sinh viên ra ngoài | Gửi sang Mỹ như Gemini hiện nay; cam kết không train trên dữ liệu gửi vào; muốn không lưu (ZDR) phải mua gói enterprise | Chỉ gửi bản đã che; ghi vào `PRODUCTION_READINESS.md` khi làm |

## 4. PoC (số đo thật, 14/11/2026, từ MacBook ở Việt Nam)

`docs/research/poc/jev-decision/poc.py` (chỉ thư viện chuẩn, khoá lấy từ biến môi trường). Kết quả từng mẫu ở `results.json`. Cùng `instructions` cho cả hai model. Jev: một câu hỏi Noul / Choice, ngưỡng 0,5. Gemini: `json_schema` `{yes, confidence}`, `temperature 0`.

Bộ dữ liệu (đều mô phỏng, D44; nhãn do agent viết, xem Giới hạn):
- **A:** E1, 200 mẫu, văn bản đã che kiểu `[[SV_1]]` như `internal/privacy`. Hỏi: có chặn khỏi kênh công khai không.
- **B:** khủng hoảng, 20 dương + 17 âm khó (`agent/testdata`).
- **C:** `routeCases`, 61 câu, 11 nhãn.
- **D:** chèn lệnh trong bài nộp, 6 dương + 6 âm.

| Tác vụ | Chỉ số | Luật hiện tại | Jev `jev-1.14.0` | Gemini `ag/gemini-3.8-flash` (9router) |
| --- | --- | --- | --- | --- |
| A tường lửa (200) | precision / recall / F1 | 1,000 / 0,960 / 0,980 (E1 9.5, văn bản gốc, nhúng `fake`: **không cùng đầu vào**) | 1,000 / 0,950 / **0,974** | 1,000 / 0,929 / 0,963 (1 lời gọi hết giờ) |
| A | chặn nhầm | 0 | **0** | **0** |
| A | S6 câu hỏi riêng tư (38) / S7 tên ngoài danh sách lớp (4) | 38/38 / **0/4** | 34/38 / **4/4** | 32/38 / **4/4** |
| B khủng hoảng (20 + 17) | recall / chặn nhầm | 20/20 / 0/17 (các câu chính là test của regex) | 14/20 ở ngưỡng 0,5 (20/20 ở 0,1) / 0/17; AUROC 1,0 | **20/20** / 0/17 |
| C định tuyến (61, 11 nhãn) | độ chính xác | 61/61 (các câu chính là test của regex) | 59/61 | 58/61 |
| D chèn lệnh (6 + 6) | recall / gắn cờ nhầm | **1/6** / 0/6 | 5/6 / 1/6 (bài trống) | **6/6** / 1/6 (bài trống) |
| Mọi tác vụ | trễ p50 / p95 (ms) | ≈ 0 (trong tiến trình) | **293–323 / 323–558** | 4.165–5.413 / 5.286–12.037 |
| Mọi tác vụ | token vào trung bình | – | 374–698 | 2.220–2.589 (đội lên do 9router) |

Lỗi chung của cả hai model ở A:
- câu ngắn về lịch / phòng thi / điểm cộng "của em" (S6), nhãn còn bàn được;
- một email ngoài danh sách lớp mà bộ che của PoC không che (sản phẩm có regex email trong `detect.go`).

Hai model cùng sai ở C, đúng hai câu thuộc luật riêng #15 ("tuần tới" kèm nội dung → `COURSE_QA`). Gemini sai thêm một câu cùng dạng.

Brier của Gemini không so được: `confidence` Gemini trả về lúc là xác suất "có", lúc là độ chắc của chính câu trả lời, dù instruction đã nói rõ. Đây là lý do thực tế để dùng xác suất đã hiệu chỉnh của Jev khi cần ngưỡng.

Độ trễ Gemini đi thẳng (tham chiếu, cùng máy, free tier):
- `gemini-3.1-flash-lite`: 10/10 lời gọi thành công, p50 ≈ 1,7 s, tối đa 6,3 s, khoảng 108 token vào.
- `gemini-3.8-flash`: 0/4 lời gọi trả về trong 300 s, giống 9.5 TL-1.

Qua 9router, mỗi request bị đội lên khoảng 2.200 token vào do lời nhắc hệ thống của đường Antigravity. Vì vậy chi phí Gemini tính theo **số token khi gọi thẳng** (≈ 110 vào + 25 ra): khoảng $0,00018 / request với 3.8-flash trả phí, tức **≈ 9 lần Jev**.

**Đọc kết quả.**
- Về độ đúng, Jev ngang Gemini Flash ở A và C, kém ở B (ngưỡng mặc định) và D.
- Về trễ, Jev nhanh hơn khoảng **14 lần** so với Gemini qua 9router và khoảng 5 lần so với gọi thẳng `3.1-flash-lite`.
- Về chi phí, Jev rẻ hơn khoảng 9 lần mỗi request.
- Xác suất của Jev tách lớp rất tốt (AUROC 0,9997 / 1,0). Nhưng **ngưỡng phải hiệu chỉnh theo từng tác vụ**; 0,5 không an toàn với tiếng Việt không dấu (B).
- Phát hiện đáng giá nhất không liên quan Jev: **regex chèn lệnh của `grading` bỏ sót 5/6 biến thể của chính bộ đo 9.5**, trong khi cả hai model đều bắt được.

**Giới hạn.**
- Nhãn A–D do agent viết (E1: `LABELING.md`, chờ chủ dự án soát theo P-11-3).
- Bộ B, D nhỏ. Bộ C là bộ mà regex được viết để qua 100 %, nên không so công bằng với luật.
- Mỗi cấu hình chạy một lượt.
- Đo từ Việt Nam; trễ khi chạy trên máy chủ thật sẽ khác.
- Gemini qua 9router có thêm độ trễ của proxy.

## 5. Khuyến nghị

| Việc | Quyết định | Cỡ | Ghi chú |
| --- | --- | --- | --- |
| Đưa Jev vào sản phẩm ở sprint 11 | **Không** | – | Repo đang khoá phạm vi trước bảo vệ. Việc này mở lại luật 3 / D47 |
| Đưa PoC này vào luận văn (mục "hướng phát triển" / phụ lục so sánh) | **Làm** | S | Chỉ tài liệu. Số liệu ở `results.json` |
| Provider `typesafe` + `Decide()` trong `internal/llm`, qua Scheduler, che PII, có `llm_audit` | Sau bảo vệ | M | Cần **ADR** (`DECISIONS.md` D53) vì mở luật 3 |
| Cờ chèn lệnh bằng LLM (OR regex), Gemini qua `internal/llm` `Structured`, làn BATCH | **Nên làm sớm nhất**, ngoài phạm vi Jev | S | Lỗ hổng thật (regex 1/6). Không cần ADR. Đo: recall trên `v1`…`v5` + bài sạch, không gắn cờ nhầm bài trống. Spec / phase do PM xếp |
| Jev: (1) groundedness cho escalate; (2) lớp câu hỏi riêng tư OR luật ở Threads precheck | Sau khi có provider, từng cái một, mỗi cái có cổng đo | S mỗi cái | Ở đây trễ và chi phí của Jev có giá trị. Cổng: E2 không xấu đi; E1 `false_block` ≤ 0,05. Mất Jev thì quay về luật |
| Điểm số, trích công thức, fallback, thay regex định tuyến / khủng hoảng | **Không làm** | – | Luật 5, D47, và điểm yếu đã công bố của Jev |

**Rủi ro:**
- Nội dung đối kháng có thể lái Jev, nên Jev không bao giờ được là cổng duy nhất.
- Ngưỡng phụ thuộc phiên bản model: ghim bản, đo lại khi đổi.
- Thêm một nhà cung cấp ngoài vào luồng dữ liệu cá nhân.
- Tiếng Việt không dấu kém hơn tiếng có dấu (B).

**Đề xuất cho PM** (dòng cho `proposals.md`):
- P-11-8 · techlead · Jev (TypeSafe) làm lớp tín hiệu phụ · Không làm ở sprint 11. Ghi vào hướng phát triển của luận văn và Nợ sau bảo vệ: ADR D53 "provider quyết định (non-OpenAI) trong `internal/llm`", rồi thử hai điểm (groundedness, câu hỏi riêng tư) có cổng đo · `docs/research/2026-11-14-jev-decision-model.md` · Không làm thì không mất gì ở sprint 11.
- P-11-9 · techlead · `grading/injection.go` chỉ bắt 1/6 biến thể chèn lệnh của bộ đo 9.5 (PoC D) · Thêm lượt gắn cờ bằng LLM `Structured` (OR với regex, chỉ gắn cờ `INJECTION_SUSPECTED`, không đổi điểm, làn BATCH, đi qua che PII). Do PM xếp: sprint 11 đợt 5 (rà bảo mật) hoặc Nợ · bằng chứng ở mục 4 · Không làm thì bài chèn lệnh tinh vi sẽ không có cờ ở màn duyệt.

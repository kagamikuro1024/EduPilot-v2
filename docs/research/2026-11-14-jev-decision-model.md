# Đưa decision model (Jev của TypeSafe, `gemini-decision-1` của Google) vào EduPilot?

> Mục 1–5: Jev. **Mục 6**: `gemini-decision-1` và cascade theo độ tin cậy (P-11-10). Mục 6 thay khuyến nghị chọn nhà cung cấp của mục 5.

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
  - kiểm câu trả lời có bám nguồn hay không, cho quyết định escalate. Riêng điểm này **phải mở lại D47**: cột hệ quả của D47 cấm "thêm lời gọi chấm điểm riêng" cho độ tin cậy escalate;
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
| Quyết định escalate | `confidence = f(retrieval, groundedness)`. Groundedness đo bằng **trùng từ** (≥ 40 % từ của câu có trong ngữ cảnh, `agent/confidence.go`). Ngưỡng là `courses.escalation_threshold` | Noul kiểu "câu trả lời có suy ra được từ các đoạn nguồn không" (cookbook *Verifying LLM answers against their source*). Bắt được trường hợp trùng từ cao nhưng sai nghĩa | Đường cong E2 escalate-rate / error-rate khi thay groundedness, so trên cùng golden set | **Hứa hẹn nhất, nhưng trái D47 hiện hành.** Cột hệ quả của D47 ghi: độ tin cậy escalate "lấy từ cùng lời gọi sinh chữ … hoặc từ điểm truy xuất, không thêm lời gọi chấm điểm riêng". Chạy sau stream thì không chạm SSE đầu, nhưng vẫn cần ADR sửa D47. Phải đo trên E2 trước |
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
| D47 | Bài nộp, Threads precheck: khớp. Trước khi sinh chữ trên chat: **mở lại D47(2)**, vì D47(2) quy định phân loại "bằng luật + độ tương đồng embedding". Sau stream để chấm độ tin cậy: **mở lại cột hệ quả D47** ("không thêm lời gọi chấm điểm riêng") | Đường không phải chat: không cần ADR về D47. Đường chat: cần ADR |
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
| Jev: (1) groundedness cho escalate (cần ADR sửa D47); (2) lớp câu hỏi riêng tư OR luật ở Threads precheck | Sau khi có provider, từng cái một, mỗi cái có cổng đo | S mỗi cái | Ở đây trễ và chi phí của Jev có giá trị. Cổng: E2 không xấu đi; E1 `false_block` ≤ 0,05. Mất Jev thì quay về luật |
| Điểm số, trích công thức, fallback, thay regex định tuyến / khủng hoảng | **Không làm** | – | Luật 5, D47, và điểm yếu đã công bố của Jev |

**Rủi ro:**
- Nội dung đối kháng có thể lái Jev, nên Jev không bao giờ được là cổng duy nhất.
- Ngưỡng phụ thuộc phiên bản model: ghim bản, đo lại khi đổi.
- Thêm một nhà cung cấp ngoài vào luồng dữ liệu cá nhân.
- Tiếng Việt không dấu kém hơn tiếng có dấu (B).

**Đề xuất cho PM** (dòng cho `proposals.md`):
- P-11-8 · techlead · Jev (TypeSafe) làm lớp tín hiệu phụ · Không làm ở sprint 11. Ghi vào hướng phát triển của luận văn và Nợ sau bảo vệ: ADR D53 "provider quyết định (non-OpenAI) trong `internal/llm`", rồi thử hai điểm (groundedness, câu hỏi riêng tư) có cổng đo · `docs/research/2026-11-14-jev-decision-model.md` · Không làm thì không mất gì ở sprint 11.
- P-11-9 · techlead · `grading/injection.go` chỉ bắt 1/6 biến thể chèn lệnh của bộ đo 9.5 (PoC D) · Thêm lượt gắn cờ bằng LLM `Structured` (OR với regex, chỉ gắn cờ `INJECTION_SUSPECTED`, không đổi điểm, làn BATCH, đi qua che PII). Do PM xếp: sprint 11 đợt 5 (rà bảo mật) hoặc Nợ · bằng chứng ở mục 4 · Không làm thì bài chèn lệnh tinh vi sẽ không có cờ ở màn duyệt.

---

## 6. Bổ sung: `gemini-decision-1` (Google) và cascade theo độ tin cậy (P-11-10)

**Kết luận.**
- `gemini-decision-1` là decision model của Google, cùng loại với Jev:
  - đầu ra là xác suất cho câu hỏi noul / choice / score, không sinh token;
  - giá $0,10 / 1 triệu token đầu vào.
- Nó **không đi được qua đường `openai-go` hiện tại**. Endpoint riêng là `POST /v1beta/decisions`; gọi qua `/v1beta/openai` trả 404. Vì vậy vẫn phải thêm hàm `Decide` vào `internal/llm`. Điểm khác Jev: cùng nhà cung cấp, cùng khoá, cùng điều khoản dữ liệu với Gemini đang dùng. Đây là một ADR nhỏ (sửa câu chữ của luật 3), không thêm nhà cung cấp mới.
- Trên PoC:
  - khủng hoảng: tốt nhất, 37/37; dương thấp nhất 0,62, âm cao nhất 0,01;
  - chèn lệnh: 6/6;
  - định tuyến: ngang Gemini Flash (58/61);
  - tường lửa: **kém nhất** (F1 0,883) vì đọc không ra placeholder `[[MSSV_1]]`. Phần này luật tất định đã chặn sẵn, nên không ảnh hưởng nếu chỉ hỏi model phần còn lại.
- **Thay bước phân loại hiện tại không giảm được trễ SSE đầu.** Sự kiện `received` được phát trước khi phân loại (`chat/gen.go:72`). Luật regex chạy cỡ micro giây. Lần nhúng duy nhất được truy xuất dùng lại (`TestOneEmbedPerMessage`), nên vẫn phải gọi.
- Cascade "decision trước, LLM khi phân vân" có ích ở **tường lửa kênh công khai** và **cờ chèn lệnh**: độ đúng bằng hoặc hơn chỉ-LLM, nhanh hơn 8–15 lần, rẻ hơn 4–7 lần.
- **Không đưa vào sprint 11.** Sprint 11 chỉ làm P-11-9 bằng `Structured` sẵn có. Cascade (cỡ M, ADR D53 + sửa D47) để sau bảo vệ.

### 6.1 Năng lực, cách gọi, giá, giới hạn (tài liệu chính thức + gọi thật 14/11)
| Mục | Giá trị | Nguồn |
| --- | --- | --- |
| Có thật với khoá của dự án | `GET /v1beta/models` trả `models/gemini-decision-1`, `supportedGenerationMethods: ["createDecision"]`, `inputTokenLimit 262144`, `outputTokenLimit 1` | gọi thật |
| Cách gọi | `POST https://generativelanguage.googleapis.com/v1beta/decisions`, header `x-goog-api-key`, thân `{model, input, questions}`. **Không** có `generateContent`, **không** có đường OpenAI-compatible: `/v1beta/openai/chat/completions` trả 404 "not supported for generateContent" | `ai.google.dev/gemini-api/docs/decisions-api`, gọi thật |
| Đầu ra | noul: `noul` ∈ [0,01; 0,99] + `confidence = max(noul, 1−noul)`. choice: `choice`, `probabilities`, `confidence` = xác suất của lựa chọn đứng đầu. score: kỳ vọng theo mức, `probabilities`, `legend`. `usage.total_output_tokens = 0` | như trên, mục *Answers, probabilities, and confidence* |
| Trễ | Google công bố 150–250 ms end-to-end (50–100 ms tính toán). **Đo từ Mac ở Việt Nam: p50 400–653 ms, p95 682–820 ms** (gồm RTT đi Mỹ) | `blog.google/…/gemini-decisions-api`, PoC |
| Giá | $0,10 / 1 triệu token đầu vào; đầu ra không tính tiền. Không có Batch, cache, Flex / Priority | `docs/pricing#gemini-decision` |
| Hạn mức | Free: 200 RPM, 50.000 TPM, 10.000 RPD. Tier 1: 1.500 RPM, 1 triệu TPM | `docs/rate-limits#gemini-decision` |
| Dữ liệu | **Free tier: "Used to improve our products: Yes"**; trả phí: No. Áp cho cả Gemini sinh chữ đang dùng | `docs/pricing` (bảng của model) |
| Lọc an toàn | Đầu vào bị lọc an toàn chặn thì trả **400** `code: safety / prohibited_content…` thay cho `answers`. Trên 37 câu khủng hoảng của PoC không có câu nào bị chặn; nhưng phải coi 400 safety là **dương** (fail-closed) | `decisions-api#request` |
| Giới hạn tự công bố | "Treat input as untrusted. User-supplied text can argue for its own classification"; so số / ngày phải làm trong code; khoá lựa chọn bị sắp theo bảng chữ cái | `decisions-api#best-practices` |

### 6.2 PoC chạy lại (cùng `poc.py`, cùng instructions, 14/11)
| Tác vụ | Chỉ số | Jev 1.14 | `gemini-decision-1` | Gemini 3.8 Flash (9router) |
| --- | --- | --- | --- | --- |
| A tường lửa (200) | P / R / F1, chặn nhầm | 1,000 / 0,950 / **0,974**, 0 | 1,000 / 0,790 / 0,883, 0 | 1,000 / 0,929 / 0,963, 0 |
| A, phần luật **không** tự chặn được (S6, S7, N: 142 mẫu) | đúng / S7 tên ngoài danh sách lớp | **138/142**, 4/4 | 133/142, 1/4 | 136/142, 4/4 |
| B khủng hoảng (20 + 17) | recall / báo nhầm | 14/20 / 0 | **20/20 / 0** | **20/20 / 0** |
| C định tuyến (61) | độ chính xác | **59/61** | 58/61 | 58/61 |
| D chèn lệnh (6 + 6) | recall / gắn cờ nhầm | 5/6 / 1 | **6/6** / 1 | **6/6** / 1 |
| Trễ p50 / p95 (ms) | A · B · C · D | 308/549 · 304/558 · 293/323 · 323/344 | 455/808 · 400/743 · 426/682 · 653/820 | 4.629/12.037 · 4.165/5.286 · 4.166/5.865 · 5.413/7.170 |
| Token vào trung bình | A · B · C · D | 474 · 374 · 642 · 698 | 266 · 169 · 330 · 463 | (≈ 2.200–2.600 qua 9router) |

Lỗi đáng chú ý:
- **A:** `gemini-decision-1` bỏ sót 4/12 câu chỉ có `[[MSSV_1]]` và 5/20 câu chỉ có `[[SV_1]]`, dù instruction đã nêu rõ placeholder nghĩa là phải chặn. Đây đúng là điểm yếu "đọc theo chữ" mà tài liệu đã cảnh báo. Trong sản phẩm, luật tất định chặn các câu này trước khi model được hỏi.
- **D:** câu gắn cờ nhầm ở cả ba model là `s4-trong`, nội dung "Em nộp nhầm tệp, thầy cho em nộp lại ạ." Câu này đúng là đang nói với giảng viên, nên nhãn còn bàn được.
- **C:** ba lỗi giống Gemini, đều thuộc luật riêng #15 của dự án.

### 6.3 Có đi qua `internal/llm` hiện tại được không, có giữ D47 không
| Câu hỏi | Trả lời | Bằng chứng |
| --- | --- | --- |
| Dùng provider `gemini` / `openai_compatible` có sẵn? | **Không.** `provider/openai.go` dựng trên `openai-go`, gọi `…/v1beta/openai/` (`BaseURLFor`). Decisions không có trên đường đó (404) | gọi thật; `internal/llm/provider/openai.go:31–65` |
| Có mở lại luật 3? | **Một phần, ADR nhỏ.** Giữ nguyên: mọi lời gọi qua `internal/llm`, cấm SDK (không dùng `google.golang.org/genai`, gọi bằng `net/http`). Phải sửa: (1) "`Client` là cổng LLM: đúng bốn hàm" (`internal/llm/types.go:135`) thành năm hàm, thêm `Decide`; (2) câu "mỗi provider là một base URL tương thích OpenAI" thêm ngoại lệ có tên: endpoint `/v1beta/decisions` của chính loại `gemini`. Không thêm nhà cung cấp, khoá hay bên xử lý dữ liệu mới | `AGENTS.md` luật 3, `types.go:135` |
| Scheduler | Khớp. Tạo một provider riêng `gemini-decision` (type `gemini`, model `gemini-decision-1`) để có token bucket riêng, đúng cách 9.5 TL-1 đã tách `gemini-embed` (Google tính hạn mức theo model). Thêm `Task` `DECIDE`, làn mặc định là INTERACTIVE cho chat / precheck, BATCH cho bài nộp | `scheduler/redis.go:131` |
| Che PII | Khớp nếu `Decide` đi qua cùng `Masker` như `Structured`. PoC chạy trên văn bản đã che | `internal/llm/mask.go` |
| D47(1): một lời gọi sinh chữ mỗi câu | **Giữ được.** Decision không sinh chữ (0 token ra) | – |
| D47(2): phân loại "bằng luật + độ tương đồng embedding, tối đa một lần / tin" | **Phải sửa câu chữ.** Số lần (một lần / tin) giữ được, nhưng cách làm đổi: thêm "+ tối đa một lời gọi decision model" | `DECISIONS.md` D47 |
| Hệ quả của D47: độ tin cậy escalate "không thêm lời gọi chấm điểm riêng" | **Trái** nếu dùng decision cho escalate. Muốn dùng phải sửa D47 | như trên |
| Thay bước phân loại để giảm SSE đầu? | **Không.** `received` phát trước khi phân loại (`chat/gen.go:72`), và D47(4) ≤ 300 ms đã đạt ở 9.5. Thời gian tới token đầu cũng không giảm: luật regex cỡ micro giây; lần nhúng duy nhất được truy xuất dùng lại nên vẫn phải gọi (`TestOneEmbedPerMessage`). Chạy decision song song với lần nhúng thì gần như không thêm trễ, nhưng cũng không bớt | `chat/gen.go:72`, `agent/agent_test.go` |

### 6.4 Cascade theo độ tin cậy: số mô phỏng trên kết quả thật
`cascade.py` dùng lại `results.json`, không gọi thêm model. Bảng đủ (mọi τ, cả Jev) ở `poc/jev-decision/cascade.md`.
- Hai kiểu luật nhận "đường nhanh":
  - **đối xứng:** nhận câu trả lời của decision model khi `confidence ≥ τ`;
  - **bất đối xứng**, chỉ được thêm cờ: "có" (p ≥ 0,5) nhận ngay; "không" chỉ nhận khi `P(có) ≤ ε`; vùng giữa chuyển sang LLM.
- Trễ của ca phải fallback = trễ decision + trễ LLM, gọi tuần tự.
- Chi phí là ước tính theo giá niêm yết; LLM tính theo token khi gọi thẳng.

| Điểm | Cấu hình đề xuất | % đi đường nhanh | Đúng (phần nhanh) | Đúng tổng | Sót dương | p50 / p95 ms | $ / 1.000 yêu cầu | So với chỉ-LLM |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| A kênh + PII | `gemini-decision-1`, bất đối xứng ε = 0,05 | 88 % | 0,994 | **0,980** | 4 | 464 / 5.565 | 0,066 | 0,960, 8 sót, 4.642 / 12.344, $0,328 |
| A kênh + PII | Jev, bất đối xứng ε = 0,1 | 96 % | 1,000 | **0,985** | 3 | 308 / 669 | 0,033 | như trên |
| B khủng hoảng | `gemini-decision-1`, bất đối xứng ε = 0,05 | 100 % | 1,000 | 1,000 | **0** | 400 / 743 | 0,017 | 1,000, 0 sót, 4.165 / 5.286, $0,249 |
| B khủng hoảng | Jev, bất đối xứng ε = 0,05 | 84 % | 1,000 | 1,000 | **0** | 305 / 4.892 | 0,055 | như trên |
| C định tuyến | `gemini-decision-1`, đối xứng τ = 0,95 | 84 % | 1,000 | 0,951 | – | 441 / 6.173 | 0,097 | 0,951, 4.166 / 5.865, $0,390 |
| D chèn lệnh | `gemini-decision-1`, bất đối xứng ε = 0,05 | 100 % | 0,917 | 0,917 | **0** | 653 / 820 | 0,046 | 0,917, 0 sót, 5.413 / 7.170, $0,476 |

Đọc số:
- Ở A, cascade **đúng hơn cả hai thành phần**. Luật bất đối xứng đẩy các ca decision model phân vân sang LLM, và LLM xử lý đúng chúng.
- Ở C, cascade không đúng hơn: lỗi của hai model trùng nhau (luật #15). Ở đây regex có sẵn vẫn thắng cả về trễ lẫn độ đúng trên bộ của chính nó.
- Ở B và D, `gemini-decision-1` một mình đã đủ trên bộ nhỏ này. Bộ 37 / 12 mẫu **không đủ để kết luận "bỏ sót = 0"**.

### 6.5 Đánh giá từng điểm
| Điểm | Cascade làm gì | Kết luận | Cỡ | Luật / ADR |
| --- | --- | --- | --- | --- |
| (a) Định tuyến ý định D47 | Decision choice 11 nhãn; phân vân → LLM | **Không.** Regex chạy micro giây và đúng 61/61 trên bộ test; cascade 0,951 với 441 ms. Chỉ đáng thử làm lớp phụ khi regex rơi vào `COURSE_QA` mặc định, và phải có bộ diễn đạt lại giữ riêng | – | D47(2) |
| (a) Kênh công khai + PII (Threads precheck) | **Luật tất định trước** (MSSV / email / SĐT / CCCD / tên trong danh sách lớp, giữ nguyên); phần còn lại hỏi decision noul "câu hỏi riêng tư / tên người cụ thể"; vùng giữa → LLM `Structured`. Chỉ được thêm chặn | **Có, sau bảo vệ.** Bù đúng chỗ luật đang trượt (S7 0/4). Precheck không thuộc đường chat nên không đụng D47. Ngưỡng ε đặt trên tập `dev` của E1 với embedding thật | M (sau khi có `Decide`) | D53 |
| (a) Escalate | Decision noul "câu trả lời có suy ra được từ nguồn" sau stream; phân vân → giữ công thức hiện tại | **Chưa.** Chưa có nhãn (golden set E2 đang chờ BA, P-11-4); trái hệ quả D47 | M | D47 (hệ quả) |
| (a) Khủng hoảng | `regex OR decision(p ≥ 0,5) OR (vùng giữa → LLM)`; 400 safety / hết giờ / lỗi → coi là **dương** (fail-closed, mở luồng hỗ trợ) | **Chưa.** Bỏ sót = 0 chỉ được **bảo đảm** khi regex vẫn là cổng (OR). Decision chỉ thêm lưới. Cần bộ thử ≥ 200 câu (có cả không dấu và teencode) trước khi bật. Chạy song song với lần nhúng để không thêm trễ | S (sau `Decide`) | D47(2) |
| (b) Cờ chèn lệnh | Decision noul trên bài nộp (đã che), làn BATCH; phân vân → LLM `Structured`; chỉ gắn `INJECTION_SUSPECTED`, OR regex | **Có.** Sprint 11 làm bằng LLM `Structured` (P-11-9, không cần ADR). Chuyển sang cascade khi đã có `Decide` | S → S | – / D53 |
| (b) Chọn mức rubric nháp | Score theo các mức mô tả (giảng viên viết) cho tiêu chí trắc nghiệm hoá được; lấy **mức có xác suất cao nhất**, không lấy giá trị kỳ vọng (là số lẻ); code đổi mức → điểm bằng `decimal`; phân vân → lượt chấm LLM hiện có; nháp vẫn chờ giảng viên duyệt, không tự công bố | **Chưa đo được.** Cần điểm người của E3 (`WAITING_HUMAN`) để chọn τ theo QWK. Về luật: tương đương lượt 1 hiện nay (model đề xuất điểm tiêu chí ở dạng nháp, code cộng, người duyệt), nhưng phải ghi rõ trong AGENTS luật 5 | M | Luật 5 (làm rõ), D53 |
| (c) Khác | Kiểm câu luyện đề do AI sinh ("đúng một đáp án", "đáp án nằm trong lựa chọn") trước khi đưa vào hàng duyệt; E4 offline thêm một bên phân loại | Ứng viên sau bảo vệ; E4 làm được ngay vì là offline | S | – |

### 6.6 Thiết kế trong `internal/llm` + Scheduler (khi làm)
- `llm.Client` thêm `Decide(ctx, DecideRequest) (DecideResponse, error)`:
  - `DecideRequest{Task, Lane, Input, Questions}`; `Questions` là noul / choice / score, kiểu Go đóng, không truyền JSON tự do;
  - đi cùng đường ống với `Structured`: che PII → giấy phép Scheduler (làn, RPM theo provider) → provider → `llm_audit` (task, token, `cost_est`, trễ, `error_kind`) → cầu dao.
- Provider: loại `gemini` thêm phương thức `Decide` gọi `POST /v1beta/decisions` bằng `net/http`, khoá lấy từ registry. Không SDK. Provider record riêng `gemini-decision` để có hạn mức riêng. Nếu sau này dùng Jev thì thêm loại `typesafe` với cùng interface.
- `internal/llm/cascade.go`: `Cascade(ctx, decide DecideRequest, policy Policy, fallback func(ctx) (Verdict, error)) (Verdict, Path, error)`:
  - `Policy` = `Symmetric{τ}` | `AddOnly{ε}`;
  - ngưỡng lưu trong `llmconfig` theo task, có mặc định trong code;
  - mọi lỗi của decision (hết giờ, 429, 400 safety) → fallback; fallback cũng lỗi → giá trị fail-closed do policy quy định (chặn / dương);
  - `Path` (`FAST` / `FALLBACK` / `FAIL_CLOSED`) ghi vào `llm_audit`, để `/observability` (US-P10-01) hiện tỉ lệ đi đường nhanh.
- Hạn giờ: decision 1 s ở làn INTERACTIVE; fallback dùng phần hạn còn lại của request gốc (luật 11).
- Test: bảng ngưỡng × kết quả giả cho `Cascade`; test fail-closed; test "Decide không có trong `internal/grade`" giống P-10-1.

### 6.7 Thay đổi luật cần có
1. **ADR D53** (`DECISIONS.md` + `AGENTS.md` luật 3 + `ARCHITECTURE.md` §LLM): `Client` có năm hàm; endpoint decision của loại `gemini` (sau này có thể thêm `typesafe`) là ngoại lệ có tên của "base URL tương thích OpenAI"; vẫn cấm SDK.
2. **Sửa D47** nếu dùng trên đường chat: (2) thêm "+ tối đa một lời gọi decision model / tin, chạy song song với nhúng"; cột hệ quả cho phép decision noul cho escalate nếu E2 chứng minh tốt hơn.
3. **Luật 5 làm rõ:** decision model chỉ được chọn *mức mô tả* của tiêu chí cho bản nháp; code đổi ra điểm; không bao giờ tính tổng, làm tròn hay công bố.
4. **`PRODUCTION_READINESS.md`:** môi trường thật phải dùng Gemini **trả phí**. Free tier ghi "used to improve our products", áp cho cả Gemini sinh chữ đang dùng.

### 6.8 Khuyến nghị
- **Sprint 11:** không thêm `Decide` / cascade (khoá phạm vi, cần ADR, chưa có nhãn E2 / E3). Chỉ làm P-11-9 bằng `Structured` sẵn có (S).
- **Sau bảo vệ, theo thứ tự:**
  1. ADR D53 + `Decide` + `cascade.go` (M);
  2. Threads precheck dạng add-only, cổng E1 `false_block` ≤ 0,05 (S);
  3. chuyển cờ chèn lệnh sang cascade (S);
  4. lưới khủng hoảng OR sau khi có bộ thử ≥ 200 câu (S);
  5. escalate và rubric nháp khi E2 / E3 có nhãn người (M mỗi việc).
- **Chọn model:** ưu tiên `gemini-decision-1` hơn Jev (cùng nhà cung cấp, không thêm bên xử lý dữ liệu, tốt hơn ở B và D). Riêng tường lửa, Jev tốt hơn, nhưng phần đó luật đã gánh.

**Rủi ro:**
- Model bị nội dung đối kháng lái (cả Google lẫn TypeSafe đều tự công bố), nên decision model không bao giờ là cổng duy nhất.
- Ngưỡng ε / τ đặt trên bộ nhỏ, có thể không chuyển sang được dữ liệu thật.
- Trễ từ Việt Nam đi Mỹ chiếm khoảng 0,3–0,4 s.
- Lọc an toàn có thể trả 400 đúng với câu khủng hoảng, nên bắt buộc fail-closed.
- Free tier dùng dữ liệu để cải thiện sản phẩm.

**P-11-10** (dòng cho `proposals.md`): xem `docs/sprints/11/proposals.md`.

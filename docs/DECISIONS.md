# Nhật ký quyết định

Đã chốt với chủ dự án ngày 2026-09-20. Muốn đổi: sửa ở đây trước, rồi sửa PRD / ARCHITECTURE / phase liên quan.

| Mã | Quyết định | Lý do chính |
| --- | --- | --- |
| D1 | Bài nộp: Mail IMAP thật + upload ZIP; Teams là adapter sau cờ, mặc định tắt | Quyền Graph `EduAssignments.*` cần admin consent của tenant trường; không hỗ trợ tài khoản Microsoft cá nhân |
| D2 | Hai kênh: chat riêng cho thông tin cá nhân, Threads cho hỏi bài; PII = tường lửa chặn câu hỏi riêng tư lọt ra Threads | Đúng mô hình Project III, mở rộng thêm chuyển kênh một chạm |
| D3 | Chấm: tự luận dạng văn bản + form trắc nghiệm; không chấm mã nguồn | Sandbox chạy code rủi ro cao, lệch trọng tâm |
| D6 | Giảng viên nhận chuông, trả lời trong app; thư bắn qua mail cho sinh viên; không nhận reply từ hộp thư | Đơn giản, đủ dùng |
| D9 | Công thức điểm trích từ file quy chế môn học (`COURSE_POLICY`), giảng viên xác nhận; hệ thống nhắc khi thiếu hoặc không rõ | Mỗi môn chấm/cộng điểm khác nhau |
| D12 | Observation = ghi chú quan sát sinh viên (M6) + quan sát hệ thống AI + báo cáo lỗ hổng kiến thức + thời gian học on-screen (M13) | Yêu cầu của chủ dự án |
| D14 | Quỹ thời gian từ 14 tuần; ước lượng hiện tại 23,5 tuần tới vạch bảo vệ, 25,5 tuần tới vạch thí điểm thật → có thứ tự cắt (xem WORKFLOW) | – |
| D15 | Che danh tính hai chiều trước LLM, bản đầy đủ (placeholder + khôi phục khi stream) | Tra điểm không bị ảnh hưởng vì tool lấy danh tính từ JWT |
| D16 | Form trắc nghiệm: làm trong app (QUIZ) và import XLSX/CSV từ Forms | Dùng chung Quiz Engine |
| D17 | Thời gian học on-screen chỉ tham khảo + tín hiệu at-risk; không tính điểm | Không đo được việc học thật; dễ bị lách |
| D18 | ~~Viết lại toàn bộ gateway bằng Go, bỏ Java; Python giữ cho AI~~ → thay bởi D45, D46 | Một ngôn ngữ backend; tách rõ đóng góp cá nhân khỏi mã kế thừa của nhóm; hợp với worker nền |
| D19 | chi + pgx + sqlc; goose; decimal cho điểm | SQL có kiểu, sát stdlib |

## Quyết định thiết kế hệ thống (2026-09-20, theo phương pháp system-design-primer)

| Mã | Quyết định | Đánh đổi đã cân nhắc |
| --- | --- | --- |
| D20 | Mục tiêu thiết kế là T1 = 1.000 SV / 20 lớp / 3.000 câu hỏi mỗi tuần; T2 = 10.000 SV chỉ có lộ trình | Thiết kế cho 30 SV thì gãy khi dùng thật; thiết kế cho 10.000 thì một người không vận hành nổi |
| D21 | Modular monolith Go + Postgres duy nhất + pgvector + Redis Streams; KHÔNG microservices, Kafka, Kubernetes, vector DB riêng, sharding | Ước lượng cho thấy nút cổ chai là LLM và kết nối sống lâu, không phải QPS hay dung lượng |
| D22 | Gateway không trạng thái từ ngày đầu: object storage + URL ký sẵn, Caddy, PgBouncer, SSE qua Redis pub/sub | Tốn thêm ≈ 0,5 tuần ở PG; đổi lại mở rộng ngang không sửa code |
| D23 | LLM Scheduler với ba làn ưu tiên, hạn mức, cầu dao, back pressure, suy giảm | Tốn ≈ 0,5 tuần ở P1; thiếu nó thì một đợt chấm bài làm treo chat của cả lớp |
| D24 | Chuẩn API: phân trang con trỏ, `Idempotency-Key`, khoá lạc quan, outbox, 202 + job cho việc dài | Thêm khuôn phép cho mọi endpoint; đổi lại chịu được bấm đúp, mạng rớt, hai giảng viên sửa cùng lúc |
| D25 | Nền tảng frontend dùng chung + 10 quy tắc UX + cổng UX mỗi phase; TanStack Query; mobile-first cho sinh viên và điểm danh | Tốn ≈ 0,5 tuần ở P2; đổi lại mọi màn sau nhất quán và rẻ hơn |
| D26 | Chứng minh bằng số: `seed-t1`, k6 theo SLO, test hỗn loạn nhỏ ở P10 | Dùng provider LLM giả để không tốn tiền; kết quả đưa thẳng vào luận văn |

| D27 | Dựng `mock-graph` trong Docker Compose; adapter Teams một bản code, địa chỉ Graph cấu hình được; 7 kịch bản lỗi | Tốn ≈ 0,5 tuần ở P7; đổi lại demo được luồng Teams và kiểm thử được hành vi khó (phân trang, 429, token, consent). Giới hạn: chỉ chứng minh đúng theo đặc tả, không chứng minh chạy với Microsoft thật |

| D28 | Hệ thiết kế giao diện là *Red Thread / Academic Instrument* (`design/DESIGN.md`, do chủ dự án cung cấp). `DESIGN.md` thắng về hình thức và bố cục; `UX.md` thắng về độ bền và hiệu năng; `PRD.md` thắng về hành vi | Mười điểm lệch giữa hai tài liệu đã chốt ở `design/INTEGRATION.md` mục 2 (điểm danh không card + nút hoàn tất buổi; sinh viên không thấy con số độ tin cậy; không toast thành công; chỉ giao diện sáng…) |
| D29 | Thêm phase PU (1,5 tuần, ngay sau PG) dựng token, app shell, primitive đủ 8 trạng thái, lớp dữ liệu frontend, và dựng lại các màn đã có backend | Tốn thời gian trước khi có tính năng mới; đổi lại mọi màn sau chỉ lắp ghép, và toàn bộ sản phẩm là một ngôn ngữ thị giác |
| D30 | Thêm M14 "Hôm nay" (`/`): xếp hạng việc bằng luật cứng, không LLM; mỗi module đăng ký nguồn việc | `DESIGN.md` §14.1 yêu cầu; rẻ, đoán trước được, không tốn token |

| D31 | Admin mở lớp và gán giảng viên / TA; Admin không thêm sinh viên; `/register` chỉ tạo STUDENT; phân công sinh thông báo cho giảng viên kèm mã tham gia và việc "Thiết lập lớp mới" | Thêm ≈ 0,5 tuần vào P2; bảng `notifications` tạo sớm ở P2 (làm mới 30 s), P4 mới nâng lên SSE + mail |
| D32 | Mã tham gia kiểu Teams: 7 ký tự không gây nhầm, tạo lại được, bật / tắt, hạn, sĩ số, tên miền email, tuỳ chọn yêu cầu duyệt (mặc định tắt); có bước xem trước lớp; import danh sách vẫn giữ song song | Vào ngay như Teams thì tiện nhưng ai có mã cũng vào được → bù bằng tạo lại mã, mời ra khỏi lớp, và tuỳ chọn duyệt |
| D33 | Đơn vị dữ liệu là LỚP (mã lớp), có `subject_code` để các lớp cùng học phần chia sẻ tài liệu, ngân hàng câu hỏi, bản nháp công thức điểm mà không nhúng lại (`document_courses` + `content_chunks.course_ids`) | Không thêm tầng "học phần" đầy đủ để giữ mô hình đơn giản; chia sẻ là thao tác chủ động của giảng viên |
| D34 | Seed: 1 giảng viên, 2 lớp cùng học phần × 30 sinh viên, 3 sinh viên học cả hai lớp, 1 sinh viên chưa vào lớp nào; lớp 2 ở trạng thái "mới nhận" (chưa có quy chế, có yêu cầu chờ duyệt) | Hai lớp ở hai trạng thái cho phép demo gần hết tình huống mà không cần dựng dữ liệu tay |

## Quyết định sau lượt rà soát end-to-end (2026-09-20, xem `FLOWS.md` mục 0)

| Mã | Quyết định | Đánh đổi |
| --- | --- | --- |
| D35 | MSSV tự khai không bao giờ mở dữ liệu; nối danh sách lớp chỉ theo email đã xác minh; lệch thì giảng viên duyệt | Thêm một bước xác minh email cho sinh viên; đổi lại chặn được mạo danh để xem điểm người khác |
| D36 | Tài khoản an toàn ở P2: xác minh email, link mời giảng viên, quên mật khẩu, khoá khi dò, access 15 phút + refresh xoay vòng trong cookie `httpOnly`; lõi gửi mail chuyển từ P4 về P2 | P2 +0,5 tuần; là lần đổi hợp đồng auth có chủ đích đầu tiên sau PG |
| D37 | Sinh viên nộp bài ngay trong app là đường mặc định; Mail / ZIP / Forms / Teams là nguồn bổ sung. Thêm màn tạo bài tập cho giảng viên và trang bài tập cho sinh viên, không thêm mục sidebar | P7 +0,5 tuần; không có thì luồng chấm bài không có điểm bắt đầu |
| D38 | Có phúc khảo (một lần mỗi bài, AI không tham gia quyết định) và "mở khoá có lý do" sau khi chốt điểm | Thêm việc cho giảng viên; bắt buộc để AI chấm được chấp nhận trong trường |
| D39 | Nội dung prompt chỉ ADMIN xem, có ghi audit; TEACHER chỉ thấy tổng hợp | Giảng viên mất khả năng tự gỡ lỗi câu trả lời AI; giữ đúng cam kết riêng tư với sinh viên |
| D40 | Khoá chat nội dung khi sinh viên đang làm QUIZ tính điểm; xáo đề; đáp án mở sau khi bài đóng | Không phải giám thị thi, chỉ chặn đường gian lận ngay trong hệ thống |
| D41 | Trần ngân sách LLM: 100% thì dừng làn BATCH, chat xuống model rẻ, không tắt trong im lặng | Chấm bài có thể bị hoãn cuối tháng; đổi lại không có hoá đơn bất ngờ |
| D42 | Hai vạch đích tách bạch: **bảo vệ** = xong P10; **thí điểm thật** = xong PR + trường đồng ý các mục pháp lý và vận hành. Có đường thí điểm sớm sau P5 | Thừa nhận rằng "chạy được trên seed" chưa phải "trường dùng được" |
| D43 | Pháp lý dữ liệu cá nhân (Luật 91/2025/QH15, NĐ 356/2025) là việc của trường với tư cách bên kiểm soát dữ liệu; hệ thống cung cấp che danh tính, màn đồng ý, xuất / xoá dữ liệu, chính sách giữ, và lối model tự host để tránh chuyển dữ liệu xuyên biên giới | Không thể tự quyết thay trường; phải làm việc với pháp chế trước thí điểm |

| D44 | Phạm vi đồ án: chứng minh hệ thống chạy end-to-end trên **dữ liệu mô phỏng hoàn toàn**. Phase PR và mọi việc pháp lý / vận hành thật nằm NGOÀI đồ án; nếu sau này trường muốn dùng thì trường lo giấy phép và pháp lý, khi đó mới làm PR | Điều kiện đi kèm: không nạp dữ liệu sinh viên thật vào hệ thống dưới bất kỳ hình thức "thử" nào khi chưa làm PR; điểm chuẩn cho E1 và E3 phải do người gán để thí nghiệm có nghĩa |

## Quyết định viết mới (2026-10-01, sprint 1)

| Mã | Quyết định | Đánh đổi |
| --- | --- | --- |
| D45 | **Viết mới toàn bộ** (thay D18). Mã Project III dời vào `legacy/`, chỉ để tham khảo: không import, không build, không chạy trong CI/compose. Không giữ hợp đồng API cũ, không golden Java, không migrate dữ liệu Project III; goose bắt đầu từ `00001`. Mọi API theo ARCHITECTURE §5 ngay từ đầu. PDF môn học trong `data/` là nội dung seed (D10), không phải mã | Mã cũ chậm và khó sửa (xem `thesis-notes/legacy-perf.md`); mọi dòng mã trong ĐATN là đóng góp cá nhân. Thêm ≈ 3–5 tuần → lịch WORKFLOW §6 phải cắt lại |
| D46 | **Chỉ Go, bỏ service Python AI** (thay D8, sửa nguyên tắc 1 và 3). Gateway + worker Go gọi LLM bằng `openai-go` qua endpoint tương thích OpenAI của từng provider; RAG truy vấn pgvector từ Go; PII regex + từ điển + mask/unmask stream trong Go; trích PDF/DOCX giao container `docling-serve` (gọi từ worker). Không gRPC, không `shared-proto`. Python chỉ còn cho script đánh giá offline ở `benchmarks/` | Mất litellm và NER tiếng Việt (vốn ở thứ tự cắt #4); structured output của vài provider qua lớp tương thích yếu hơn bản gốc → mỗi provider có test hợp đồng với provider `fake`. Đổi lại: bớt một service, một hop, một bộ test, một Dockerfile; token stream thẳng ra SSE |
| D47 | **Luật tốc độ cho đường hỏi–đáp** (rút từ `thesis-notes/legacy-perf.md`): (1) mỗi câu hỏi đúng MỘT lời gọi LLM sinh chữ, có stream; truy xuất là tất định (embed + tìm lai vector/từ khoá bằng SQL), không LLM viết lại câu hỏi, không LLM rerank/tổng hợp trên đường nóng; (2) phân loại ý định / kênh bằng luật + độ tương đồng embedding, tối đa một lần mỗi tin nhắn, kết quả mang theo xuống dưới, không gọi lại; (3) không vòng lặp agent mở cho hỏi đáp: định tuyến tất định → tool Go → một lần sinh; (4) sự kiện SSE trạng thái đầu tiên ≤ 300 ms; (5) cache câu trả lời cuối theo lớp, vô hiệu khi tài liệu đổi; (6) danh tính lấy từ claim JWT, không truy DB mỗi request; (7) SQL truy xuất: `ORDER BY` chỉ theo khoảng cách để dùng HNSW, lọc `course_id` trong cùng truy vấn, `tsvector` lưu sẵn + GIN; (8) frontend render token theo khung hình, không parse lại markdown mỗi token; dữ liệu đầu trang render phía server | Độ tin cậy để escalate phải lấy từ cùng lời gọi sinh chữ (trường có cấu trúc ở cuối stream) hoặc từ điểm truy xuất, không thêm lời gọi chấm điểm riêng |
| D48 | **Phiên bản nền**: Go 1.27; Node 24 LTS; PostgreSQL 18 + pgvector (HNSW, `halfvec` nếu đủ chính xác); Redis 8; Next.js 16 (App Router, React Server Components, Turbopack) + React 19; pnpm; Caddy 2; `docling-serve` bản ổn định v1; MinIO; **Mailpit** thay MailHog (MailHog ngừng phát triển, image chỉ có amd64; Mailpit giữ cổng SMTP 1025 / UI 8025, có REST API cho test). Bản vá cụ thể ghim trong `go.mod`, lockfile, tag image | Dùng bản mới nhất ổn định ở thời điểm bắt đầu; không nâng bản lớn giữa chừng trừ khi có lỗi bảo mật |
| D49 | Câu trả lời trong chat riêng dưới ngưỡng độ tin cậy **tự chuyển giảng viên** (mở ticket, idempotent); sinh viên thấy "AI chưa đủ chắc chắn về câu này · Đang chờ giảng viên". Nút `Nhờ giảng viên hỗ trợ` vẫn có ở mọi câu trả lời khác (chủ dự án chốt 2026-10-01, `FEAT-demo-script` Q3) | Thêm ticket so với để sinh viên tự quyết; đổi lại câu khó không bị bỏ rơi — đúng vấn đề gốc |
| D50 | Chi phí LLM trong `/analytics` chỉ GIẢNG VIÊN và ADMIN thấy; TA không thấy (chủ dự án chốt 2026-10-01) | – |
| D51 | **Prototype giao diện là bản mô phỏng có hạn dùng**: spec ở `docs/sprints/1.5/spec/`, dữ liệu ở `frontend/src/mock/`, trạng thái giả lập trong trình duyệt. Mỗi sprint build thật thay dần màn mock bằng màn thật; spec thật viết mới ở `docs/specs/` và thắng spec prototype. Primitive + shell ở `frontend/src/shared/` được giữ và nâng chuẩn ở PU (chủ dự án chốt 2026-10-01) | Prototype đi trước backend nên có thể lệch hợp đồng API thật; khi lệch, spec thật thắng và màn được sửa theo |
| D52 | Thêm `getkin/kin-openapi` **chỉ cho test** `internal/contract`: kiểm response thật khớp `api/openapi.yaml` (PM chốt 2026-10-01, chủ dự án giao toàn quyền khi vắng mặt) | Tự viết validator OpenAPI là việc thừa; không vào binary gateway |

## Mặc định theo khuyến nghị (chưa được chủ dự án xác nhận riêng, đổi được)

| Mã | Mặc định | Ghi chú |
| --- | --- | --- |
| D4 | Trục nghiên cứu: chấm tự luận tự động + che danh tính hai chiều | NÊN HỎI THẦY HƯỚNG DẪN trước tuần 4. Không đổi phạm vi code, chỉ đổi thí nghiệm nào làm sâu |
| D5 | Thêm role TEACHER tách với TA | Người duy nhất xác nhận công thức, công bố, chốt điểm |
| D7 | Chỉ giảng viên upload tài liệu | Tránh bản quyền, lộ đề |
| D8 | ~~LiteLLM~~ (thay bởi D46: `openai-go` + endpoint tương thích OpenAI); embedding cố định 1536 chiều | Ít code nhất |
| D10 | Seed môn An ninh mạng, dùng lại PDF trong `data/`; quy chế môn học tự soạn | Có quy chế môn học thật thì thay vào, demo thuyết phục hơn |
| D11 | Luyện đề: trắc nghiệm + trả lời ngắn + tự luận | Tự luận dùng lại Grading Engine |
| D13 | Demo bảo vệ bằng Docker Compose trên VPS/máy cá nhân | Worker + IMAP + MailHog khó gói trong một container HF |

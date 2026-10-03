# EduPilot v2 — hướng dẫn cho Claude Code

Bạn đang làm việc cùng MỘT lập trình viên duy nhất (sinh viên làm đồ án tốt nghiệp). Không có reviewer nào khác.
Người này duyệt kế hoạch, chạy cổng nghiệm thu và merge. Bạn thi công.

## Dự án là gì

EduPilot v2 = nền tảng vận hành lớp học có AI cho một học phần: hỏi đáp hai kênh (chat riêng + Threads),
chấm bài tự động, CRM sinh viên, sổ điểm, luyện đề, lịch, thư viện, cấu hình LLM, observation.
Viết mới toàn bộ (D45). Mã Project III nằm ở `legacy/`, chỉ đọc để tham khảo: không import, không build, không sửa.

Tài liệu nguồn (đọc khi cần, KHÔNG đọc hết mỗi phiên):

| File | Đọc khi |
| --- | --- |
| `docs/PROGRESS.md` | LUÔN đọc đầu phiên: đang ở phase nào, việc nào xong, nợ gì |
| `docs/phases/<PHASE>.md` | Đang thi công phase đó |
| `docs/ARCHITECTURE.md` | Cần schema, API, cấu trúc thư mục, thư viện, env |
| `docs/SYSTEM_DESIGN.md` | Viết bất cứ thứ gì đụng LLM, hàng đợi, cache, file, SSE, danh sách lớn; cần biết mục tiêu tải T1 và SLO |
| `docs/design/DESIGN.md` | Viết BẤT KỲ màn hình nào. Là nguồn thẩm quyền về diện mạo, bố cục, component, lời văn. §14 có hợp đồng từng route; §21 là phản mẫu; §22 là định nghĩa xong |
| `docs/design/INTEGRATION.md` | Lần đầu đụng frontend; khi `DESIGN.md` và `UX.md` có vẻ mâu thuẫn; cần biết component nào dựng ở phase nào |
| `docs/UX.md` | Viết BẤT KỲ màn hình nào. Hành vi khi mạng xấu, tự lưu, lạc quan, ngân sách hiệu năng, cổng UX |
| `docs/design/edupilot-ui-v3.html` | Tham chiếu cảm quan (mở bằng trình duyệt). KHÔNG chép mã từ đây |
| `docs/PRD.md` | Cần hiểu yêu cầu và tiêu chí nghiệm thu (AC) của một module |
| `docs/FLOWS.md` | Làm bất cứ thứ gì nối nhiều module; viết test E2E; cần biết nhánh lỗi của một hành trình. Mỗi luồng F1–F18 phải đi được từ đầu đến cuối |
| `docs/PRODUCTION_READINESS.md` | Phase PR; hoặc khi đụng dữ liệu cá nhân, đồng ý, giữ / xoá dữ liệu, sao lưu, giám sát |
| `docs/DECISIONS.md` | Thấy có hai cách làm và không chắc cách nào đã chốt |

## Kiến trúc một dòng

Next.js (frontend) → Go gateway `backend-go` (chi + pgx + sqlc; HTTP, SSE, nghiệp vụ VÀ AI: `internal/llm`, `internal/rag`,
`internal/privacy`, `internal/grading`, `internal/agent`). Postgres + pgvector, Redis (cache, Streams, pub/sub).
Worker Go chạy việc nền, gọi container `docling-serve` để trích PDF/DOCX và container sandbox để chấm code C/C++ của bài thi (D55). Không có service Python (D46).

## 9 nguyên tắc bất biến

1. **Go sở hữu cả nghiệp vụ lẫn AI (D46).** Không có service Python nào chạy thường trực: agent, RAG, chấm bài, PII, LLM gateway đều là package Go trong `backend-go/internal/`. Python chỉ còn script đánh giá offline ở `benchmarks/`, gọi qua HTTP API như một client ngoài.
2. **Danh tính lấy từ JWT.** Tool của agent không nhận `student_code`/`user_id` từ LLM; luôn lấy từ `trusted_context`.
3. **Không gọi SDK provider trực tiếp.** Mọi lời gọi LLM/embedding đi qua `internal/llm` (dựng trên `openai-go`, mỗi provider là một base URL tương thích OpenAI + key). Cấm import SDK provider ở bất kỳ package nào khác.
4. **Hai kênh, một tường lửa.** Bài đăng/bình luận công khai qua tường lửa PII trước khi lưu. Tool dữ liệu cá nhân chỉ đăng ký cho agent kênh chat riêng. Tên và MSSV được thay placeholder quanh MỌI lời gọi LLM.
5. **Tính điểm là code thuần.** LLM chỉ trích công thức từ quy chế thành bản nháp. Không prompt nào tính hay làm tròn điểm. Go dùng `shopspring/decimal`, cấm `float64` cho điểm.
6. **Migration bằng goose**, bắt đầu từ `00001`, không sửa file migration đã merge.
7. **API theo quy ước `docs/ARCHITECTURE.md` §5 từ đầu.** Endpoint mới nào cũng có trong `backend-go/api/openapi.yaml`; không giữ hợp đồng của Project III.
8. **Không phá cái đang chạy.** Trước và sau mỗi lát việc: chạy test Go và test frontend.
9. **Commit nhỏ**, nhánh `feat/<phase>-<tên>`, thông điệp `<phase>: <việc>`.

## 7 luật mở rộng (thiết kế cho tải T1 = 1.000 SV, xem `docs/SYSTEM_DESIGN.md`)

10. **Gateway không trạng thái.** Không giữ gì trong bộ nhớ tiến trình hay đĩa cục bộ: phiên = JWT; rate limit, idempotency, ánh xạ placeholder, hạn mức = Redis; file = object storage qua `platform/blob` + URL ký sẵn.
11. **Mọi lời gọi LLM đi qua Scheduler** với làn ưu tiên (`INTERACTIVE` > `NEAR_REALTIME` > `BATCH`), hạn mức, cầu dao, deadline truyền từ request gốc. Việc BATCH không bao giờ được làm chat của sinh viên treo.
12. **Việc nặng không chạy trong request.** Trích tài liệu, chấm bài, báo cáo, mail → Redis Streams; API trả 202 + job id, tiến độ qua SSE. Consumer idempotent, retry 3 lần, dead-letter.
13. **Danh sách nào cũng phân trang con trỏ** (`cursor`, `limit` ≤ 100). Cấm OFFSET trên bảng lớn, cấm trả danh sách không giới hạn, cấm N+1. Index phức hợp bắt đầu bằng `course_id`.
14. **Ghi có tác dụng phụ phải idempotent.** POST quan trọng nhận `Idempotency-Key`; thông báo/mail/sự kiện đi qua transactional outbox; sửa đồng thời dùng cột `version` → 409.
15. **Cache vô hiệu theo sự kiện**, TTL chỉ là lưới an toàn. Dữ liệu điểm/điểm danh không cache phía server.
16. **Đường hỏi–đáp theo luật tốc độ D47** (`docs/DECISIONS.md`): một lời gọi LLM sinh chữ mỗi câu hỏi, truy xuất tất định bằng SQL, phân loại một lần mỗi tin nhắn, không vòng lặp agent mở, sự kiện SSE đầu tiên ≤ 300 ms.

## Luật giao diện — hướng "Red Thread / Academic Instrument" (chi tiết ở `docs/design/DESIGN.md`)

- Đỏ + trắng là thương hiệu, nhưng **đỏ là tín hiệu, không phải nền**. Đỏ chỉ mang ba nghĩa: đang ở đâu / cần hành động / đã sửa-xác nhận. Không có nghĩa thì bỏ.
- Không tường thẻ KPI, không card lồng card, không bọc mọi mục trong khung bo góc. Dùng khoảng trắng, độ gần, chữ và đường kẻ 1 px trước khi dùng khung chứa.
- Bo góc nhỏ (6–12 px), gần như không bóng (chỉ menu / popover / dialog). Một họ chữ: Be Vietnam Pro. Không chữ gradient, không glass, không neon AI.
- Một hành động chính cho mỗi vùng làm việc; tối đa 2 hành động phụ hiện ra; còn lại vào menu. Sửa tại chỗ và mở dần trước khi dùng modal. Dialog chỉ cho việc cần bảo vệ.
- Điều hướng chính trên desktop luôn có nhãn chữ. Sinh viên không bao giờ thấy từ kỹ thuật AI (RAG, PII, fallback, trace, provider…); dùng bảng dịch ở `DESIGN.md` §13. Nút là động từ.
- Chỉ dùng token `--ep-*` và primitive ở `frontend/src/shared/`. Cấm màu / bo góc / bóng / cỡ chữ viết cứng trong trang. Cấm tạo bản sao riêng của nút, trường nhập, chip, dialog, tab, bảng, thông báo cho từng route.
- Không gamify: không streak, không bảng xếp hạng, không vòng tiến độ, không confetti.
- `bash scripts/ui-antipatterns.sh` phải sạch trước khi báo xong.

## Luật UX (chi tiết ở `docs/UX.md`)

Màn hình nào cũng dùng nền tảng chung ở `frontend/src/shared/` (dựng ở phase PU): TanStack Query + `apiClient`, `<PageState>` (tải/rỗng/lỗi), `DataTable`, `useAutosaveDraft`, `useUndoableAction`, `<ConfirmIrreversible>`, `useSSE`. Cấm `fetch` trần, spinner riêng, confirm riêng, bảng riêng.
Thao tác đảo ngược được → cập nhật lạc quan + dòng "Hoàn tác" tĩnh lặng tại chỗ, không hỏi xác nhận, không toast "Thành công". Không bao giờ làm mất chữ người dùng đã gõ. Màn của sinh viên và `/attendance`, `/inbox` phải dùng tốt ở bề rộng 375 px.

## Cấm tuyệt đối

- Sửa contract test, `expected_final_grades.csv` hay nới lỏng assertion để test xanh. Test đỏ → sửa code, hoặc DỪNG và hỏi.
- Làm việc của phase khác "tiện tay". Thấy cần → ghi vào mục Nợ trong `docs/PROGRESS.md`.
- Ghi secret vào repo, log PII (tên, MSSV, email, điểm kèm danh tính), log bảng ánh xạ placeholder.
- Tự công bố điểm AI chấm; tự xác nhận công thức điểm. Hai việc này luôn cần thao tác của giảng viên.
- Dùng thời gian học on-screen trong `internal/grade`.
- Nối tài khoản vào danh sách lớp, hay trả bất kỳ dữ liệu cá nhân nào, dựa trên MSSV do người dùng tự khai. Chỉ email đã xác minh hoặc `user_id` từ phiên mới là danh tính.
- Cho TEACHER / TA đọc nội dung chat riêng của sinh viên khi chưa escalate, kể cả qua trang quan sát hay log. Nội dung prompt chỉ ADMIN xem, có ghi `audit_log`.
- Cho sinh viên thấy điểm nháp, ghi chú quan sát, nhãn rủi ro của chính mình, hay đáp án khi bài còn mở.
- Lưu token ở localStorage; lưu token xác minh / đặt lại / mời ở dạng rõ (phải băm).
- Thêm thư viện ngoài bảng ở `docs/ARCHITECTURE.md` mà không hỏi. Thêm service mới (nhất là service Python) — D46 chốt chỉ còn Go + `docling-serve`, D55 thêm đúng một sandbox chấm code (không chứa nghiệp vụ). Chạy code / binary của sinh viên ở bất cứ đâu ngoài sandbox.
- Ghi file ra đĩa cục bộ của container; giữ trạng thái người dùng trong biến toàn cục.

## Định nghĩa "xong" ở mức luồng

Một tính năng chưa xong khi module của nó chạy; nó xong khi **luồng** chứa nó trong `docs/FLOWS.md` đi được từ bước đầu đến bước cuối, gồm cả nhánh lỗi đã liệt kê, và có spec E2E tương ứng. Module sinh việc cần người xử lý phải đăng ký nguồn việc cho "Hôm nay" (bảng ở F14).

## Khi nào DỪNG và hỏi

PRD/phase file mâu thuẫn hoặc thiếu; cần đổi hợp đồng API cũ; cần đổi schema đã merge; cổng nghiệm thu đỏ sau 2 lần sửa;
cần secret/tài khoản thật; thay đổi đụng auth, phân quyền, mã hoá, tính điểm mà phase file không nêu.

## Lệnh

```bash
pnpm dev                 # dựng toàn bộ stack local (compose) + seed khi DB trống
pnpm dev:status | dev:logs | dev:down
# Go
cd backend-go && go vet ./... && golangci-lint run && go test -race ./...
cd backend-go && sqlc generate && sqlc diff
cd backend-go && goose -dir db/migrations postgres "$DATABASE_URL" up
cd backend-go && go test ./internal/contract/...        # contract test: response khớp openapi.yaml
# Đánh giá offline (script Python, gọi qua HTTP API như client ngoài)
make eval                          # E1–E6 → benchmarks/reports/
# Frontend
pnpm -C frontend lint && pnpm -C frontend build
pnpm -C frontend exec playwright test
bash scripts/ui-antipatterns.sh     # phản mẫu giao diện theo DESIGN.md §21
k6 run benchmarks/load/<kịch-bản>.js   # test tải theo SLO ở docs/SYSTEM_DESIGN.md mục 5
```

(`legacy/` không nằm trong lệnh nào ở trên.)

## Quy ước code

- Go: `internal/<module>/{handler,service,repo}.go`; handler mỏng, logic ở service, SQL ở `internal/store/queries/*.sql` (sqlc). Lỗi bọc bằng `%w`. Log bằng `slog`, có `trace_id`. Context đi xuyên suốt. Test table-driven.
- Mọi handler lớp học đi qua middleware `CourseAccessGuard`; STUDENT chỉ đọc dữ liệu của `user_id` trong JWT.
- Go AI: `internal/llm` (chat/stream/structured/embed, registry provider, fallback, `llm_audit`) + `internal/llm/scheduler`; `internal/rag` (chunk, embed, tìm lai); `internal/privacy` (detect, redact, mask, unmask_stream, phân loại kênh); `internal/grading` (rubric qua structured output, hai lượt); `internal/agent` (định tuyến tất định, tool lấy danh tính từ trusted context); `internal/ingest` (worker gọi `docling-serve`). Đầu ra LLM có cấu trúc luôn dùng `llm.Structured(json_schema)`, không parse văn bản tự do.
- Frontend: `frontend/src/features/<module>/`, gọi API qua `services/`, state bằng zustand, biểu đồ bằng recharts. Mỗi màn có trạng thái loading / rỗng / lỗi. Giao diện tiếng Việt.
- Tên bảng, cột, enum: snake_case tiếng Anh. Văn bản hiển thị: tiếng Việt.

## Thứ tự làm trong một phase

migration → sqlc/store → service + handler Go → frontend → test → seed → cập nhật `docs/PROGRESS.md`.
Mỗi bước một commit. Kết thúc: chạy cổng nghiệm thu trong phase file, dán kết quả, liệt kê file đổi và việc còn nợ.

## Khi chạy trong đội herdr (`docs/team/`)

Repo có thể đang được 4 phiên Claude Code làm cùng lúc: `pm`, `ba`, `dev`, `qc`. Vai của bạn do prompt đầu phiên quy định; đọc `docs/team/<VAI>.md` và chỉ sửa đúng vùng file của vai đó. Giao tiếp qua file trong `docs/specs/`, `docs/sprints/`; trạng thái không nằm trong hội thoại. Mọi luật ở trên áp dụng cho cả bốn vai.

**Không thoả hiệp ngang hàng.** `ba`, `dev`, `qc` không nói chuyện trực tiếp với nhau (không `herdr agent prompt` sang vai khác, không nhắn qua file ngoài `proposals.md`) và không "dàn xếp" để việc của nhau qua: dev không xin QC nới TC, QC không sửa TC cho khớp code, BA không sửa AC cho khớp cái dev đã làm, không ai sửa file của vai khác. Mọi thay đổi spec/AC/TC sau khi `APPROVED` chỉ hợp lệ khi có dòng `proposals.md` được PM chấp nhận, và commit thay đổi đó phải ghi số proposal. PM gặp thay đổi không có số proposal → hoàn tác và ghi lỗi nghiêm trọng vào report sprint.

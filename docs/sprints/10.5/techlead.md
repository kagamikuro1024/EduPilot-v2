# Sprint 10.5: kênh Tech Lead

## TL-1 (D60, P-11-10, P-11-11): thiết kế mã chi tiết cho decision model + cascade

> **Ưu tiên (2026-11-14, sau khi BA chốt `docs/specs/FEAT-decision-model/SRS.md`):** chỗ nào TL-1 khác SRS thì **SRS thắng**. Các chỗ khác đã biết:
> - Tên cổng là `THREADS_PRECHECK` / `INJECTION_FLAG` / `CHAT_ROUTING`, và `cascade_path` có thêm `FAIL_CLOSED`.
> - Công tắc nằm ở bảng `decision_gates` + `course_decision_gates`, không phải cột `courses.decision_*`. Ngưỡng và `timeout_ms` đặt theo từng cổng trong `decision_gates`, không nằm ở `llm_task_routes.params`.
> - **Không** có tuyến `DECIDE` trong `llm_task_routes`: dùng provider `type='decision'` duy nhất đang bật.
> - `decision-fake` là model trên provider `decision`, không phải `type='fake'`. Nó phát lại từ `internal/llm/testdata/decision-replay.json`; thiếu khoá thì lưỡng lự và đi fallback, không trả giá trị trung tính 0,01. Env của nó là `LLM_DECISION_FAKE_*`, thay cho `FAKE_DECISION_LATENCY`.
> - Gọi thật cần thêm `LLM_DECISION_ALLOW_REAL`, `LLM_DECISION_REAL_DAILY_CAP`, `LLM_DECISION_RPM`.
> - Threads `FAIL_CLOSED` hiện hộp thoại chặn, không trả 503.
>
> ARCHITECTURE / SYSTEM_DESIGN / D60 / guide đã theo SRS.

PM giao (2026-11-14): thiết kế mã chi tiết cho dev. Gồm gói, tệp, hàm, giao diện `Decide()`, schema `createDecision`, timeout 800 ms, làn + token bucket, `llm_audit`, provider giả có độ trễ cấu hình được, cách gộp vào chat / Threads / grading, và các test cần có.

Nguồn đã chốt: D60 + D47(2) (`docs/DECISIONS.md`), `docs/research/poc/jev-decision/cascade.md` (ngưỡng), P-11-10, P-11-11, plan 10.5. Số đo thật chỉ có ở PoC ngày 14/11; mọi việc trong sprint này chạy bằng provider giả, trừ khi PM cho phép (luật chạy của plan).

**TL trả lời:**

### 0. Việc phải làm trước L1
- **Nhánh `sprint/10.5-decision` tách từ `sprint/11-p10-eval`, nên CHƯA có sprint 10** (`main` đã có `00019_practice.sql`; nhánh này dừng ở `00017`). Trước khi viết mã, phải `git merge origin/main` vào nhánh. Không làm thì migration mới đụng số, và `internal/practice` không có để test hồi quy.
- **Migration mới là `00020_decision.sql`**: số kế tiếp sau `00019` của `main`. `00018` để dành cho 9.5 (P-10-2).

### 1. Lược đồ: `00020_decision.sql` (goose, không sửa migration cũ)
```sql
ALTER TABLE llm_providers DROP CONSTRAINT llm_providers_type_chk,
  ADD CONSTRAINT llm_providers_type_chk CHECK (type IN ('openai','anthropic','gemini','openai_compatible','fake','decision'));
ALTER TABLE llm_models DROP CONSTRAINT llm_models_kind_chk,
  ADD CONSTRAINT llm_models_kind_chk CHECK (kind IN ('chat','embedding','decision'));
ALTER TABLE llm_task_routes DROP CONSTRAINT llm_task_routes_task_chk,
  ADD CONSTRAINT llm_task_routes_task_chk CHECK (task IN ('CHAT','CLASSIFY','UTILITY','GRADING','QUESTION_GEN','INSIGHT','EMBEDDING','DECIDE'));
ALTER TABLE llm_audit ADD COLUMN cascade_gate text, ADD COLUMN cascade_path text,
  ADD CONSTRAINT llm_audit_cascade_gate_chk CHECK (cascade_gate IS NULL OR cascade_gate IN ('THREADS','INJECTION','ROUTING')),
  ADD CONSTRAINT llm_audit_cascade_path_chk CHECK (cascade_path IS NULL OR cascade_path IN ('FAST','FALLBACK'));
ALTER TABLE courses ADD COLUMN decision_threads boolean NOT NULL DEFAULT false,
  ADD COLUMN decision_injection boolean NOT NULL DEFAULT false,
  ADD COLUMN decision_routing boolean NOT NULL DEFAULT false;
```
- `kind = 'decision'` để **`Registry.Cheapest()` không bao giờ chọn mô hình decision** làm mô hình chat rẻ nhất khi cạn ngân sách. Vòng lặp hiện tại chỉ lọc `m.Kind != "chat"` (`registry.go`), nên đã an toàn sẵn; chỉ cần thêm test.
- Công tắc theo lớp nằm ở cột của `courses`, mặc định **tắt** (plan 10.5). Ngưỡng **không** để theo lớp mà để trong `llm_task_routes.params` của tuyến `DECIDE`, vì một hệ thống chỉ cần một bộ ngưỡng (mục 4).
- `llm_audit` không thêm cột nội dung. `cascade_gate` / `cascade_path` chỉ là nhãn.
- Index: `llm_audit_task_created_idx (task, created_at DESC)` đã có, đủ cho thống kê DECIDE.

### 2. `internal/llm`: kiểu và giao diện
Tệp `internal/llm/types.go`:
```go
const TaskDecide Task = "DECIDE"                     // thêm vào danh sách; DefaultLane(TaskDecide) = LaneInteractive (để người gọi HẠ xuống BATCH)

type QKind string                                    // "noul" | "choice" | "score"; score: không dùng ở 10.5 (D60 (5)), vẫn khai báo cho đủ hợp đồng
type Question struct {
	Kind         QKind
	Instructions string                              // chữ tĩnh của hệ thống, KHÔNG có dữ liệu người dùng → không che
	Criteria     []Option                            // choice / score: thứ tự giữ nguyên; noul: rỗng
}
type Option struct{ Key, Text string }               // Key: [A-Za-z0-9_ -], ≤ 128 ký tự (giới hạn của Google)
type Gate string                                     // "THREADS" | "INJECTION" | "ROUTING"
type DecideRequest struct {
	Lane      *Lane                                  // nil = INTERACTIVE
	Gate      Gate                                   // ghi llm_audit.cascade_gate
	Input     string                                 // nội dung người dùng: gateway CHE trước khi gửi
	Questions map[string]Question                    // khoá [A-Za-z0-9_-]{1,64}; 1..10 câu (ta tự giới hạn, Google cho 1.000)
	Accept    func(DecideResponse) bool              // policy cascade (mục 4); nil = không ghi cascade_path
}
type Answer struct {
	Kind       QKind
	Noul       float64                               // 0,01..0,99
	Choice     string
	Probs      map[string]float64
	Confidence float64                               // noul: max(p, 1-p); choice: xác suất lựa chọn đứng đầu
}
type DecideResponse struct {
	Answers  map[string]Answer
	Path     string                                  // "FAST" | "FALLBACK" (theo Accept); "" khi Accept nil
	TokensIn int
	Provider, Model string
	CostEst  decimal.Decimal
}
// Client: thêm hàm thứ năm (D60).
Decide(ctx context.Context, r DecideRequest) (DecideResponse, error)
```
- `Accept` được đánh giá **trong** gateway, trước `finish`. Nhờ vậy `cascade_path` nằm trên chính dòng audit của lời gọi decision; lỗi thì `Path = "FALLBACK"`. Người gọi đọc `resp.Path`, không tự tính lại.
- Mọi kiểu mới dùng `float64` cho xác suất. **Không** dùng cho điểm (luật 5 chỉ áp cho điểm số).

Tệp `internal/llm/provider/provider.go`. Giữ nguyên interface `Provider`, thêm interface tuỳ chọn:
```go
type DecideOpts struct { Model string; Input string; Questions map[string]DecideQuestion }
type DecideQuestion struct { Type, Instructions string; Criteria any } // map[string]string cho choice, []string cho score, nil cho noul
type DecideResult struct { Answers map[string]DecideAnswer; TokensIn int }
type DecideAnswer struct { Type, Choice string; Noul, Confidence float64; Probabilities map[string]float64 }
type Decider interface{ Decide(ctx context.Context, o DecideOpts) (DecideResult, error) }
```

### 3. Provider loại `decision`: `internal/llm/provider/decision.go` (mới)
- `NewDecision(DecisionConfig{APIKey, BaseURL /* mặc định https://generativelanguage.googleapis.com/v1beta */, HTTP *http.Client})`.
- Cài `Provider`: `Chat` / `Stream` / `Embed` trả `&Error{Kind: KindBadRequest}`, vì chỉ dùng được qua `Decide`. Cài thêm `Decider`.
- `http.Client` riêng: không theo chuyển hướng (giống `openai.go:73`, chống SSRF), `Timeout` = 0. Hạn chót lấy từ `ctx`.
- **Không SDK.** `google.golang.org/genai` bị cấm (D60). Thêm vào bài kiểm `grep` của US-P1-02 AC1.

Lời gọi (tài liệu chính thức `ai.google.dev/gemini-api/docs/decisions-api`, mục *Request and response reference*):
```http
POST {BaseURL}/decisions
x-goog-api-key: <khoá>          ← header, KHÔNG để trong query string (log proxy giữ được URL)
Content-Type: application/json

{"model":"gemini-decision-1","input":"<đã che>",
 "questions":{"personal":{"type":"noul","instructions":"…"},
              "intent":{"type":"choice","instructions":"…","criteria":{"PERSONAL_GRADE":"…","COURSE_QA":"…"}}}}
```
```json
{"answers":{"personal":{"type":"noul","noul":0.90,"confidence":0.90},
            "intent":{"type":"choice","choice":"COURSE_QA","probabilities":{"COURSE_QA":0.97,"PERSONAL_GRADE":0.03},"confidence":0.97}},
 "usage":{"total_input_tokens":123,"total_output_tokens":0,"total_tokens":123,"input_tokens_by_modality":[{"modality":"text","tokens":123}]}}
```

Đổi mã HTTP sang `Kind` (tái dùng bảng có sẵn của `openai.go`):

| HTTP | Kind | Ghi chú |
| --- | --- | --- |
| 200 | – | Kiểm: đủ khoá câu hỏi, đúng `type`, `noul` ∈ [0,1], `choice` ∈ tập lựa chọn. Sai → `KindBadResponse` |
| 400 có `error.code` ∈ {`safety`, `prohibited_content`, `blocklist`, `content_blocked`} | **`KindBlocked` (mới)** | Không tính vào cầu dao, không thử lại, không chuyển nhà kế. Cascade coi như phân vân → FALLBACK |
| 400 khác | `KindBadRequest` | Lỗi của ta (schema câu hỏi) |
| 401 / 403 | `KindAuth` | |
| 404 | `KindModelNotFound` | |
| 429 | `KindRateLimit` + `Retry-After` | |
| 5xx / 529 | `KindServer` | |
| hết hạn ctx | `KindTimeout` / `KindCancelled` | |

- `Detail` của lỗi đi qua `Redact(detail, apiKey)` như hiện nay. Không log `input`.
- **Thứ tự lựa chọn:** Google sắp khoá lựa chọn theo bảng chữ cái (`decisions-api#best-practices`). Đặt khoá ổn định (mã ý định), đừng dựa vào thứ tự.

Đăng ký trong `registry.go`:
- `defaultFactory`: `case "decision": return provider.NewDecision(...)`.
- `build()`: model `kind = 'decision'` chỉ gắn vào tuyến `DECIDE`.
- `envRoutes()`: khi `LLM_PROVIDER=fake`, thêm `out[TaskDecide] = Route{Targets: []Target{fakeDecision}}` với model `decision-fake`.
- Không có đường dự phòng env cho decision thật. `GEMINI_API_KEY` trong env **không** tự bật decision: muốn dùng thật thì ADMIN thêm provider `decision` trong `/settings/llm`, để không ai vô tình tiêu khoá.

### 4. Gateway `Decide` + cascade: `internal/llm/decide.go` (mới)
Theo khuôn của `Embed` (`gateway.go:597`): `begin` → kiểm đầu vào → che → `runChain` → `finish`.
```go
func (g *Gateway) Decide(ctx context.Context, r DecideRequest) (DecideResponse, error) {
	lane := r.Lane; if lane == nil { l := LaneInteractive; lane = &l }
	c, err := g.begin(ctx, TaskDecide, lane, 0)
	if err != nil { return DecideResponse{Path: fallbackIf(r)}, err }
	c.row.CascadeGate = string(r.Gate)
	if verr := validDecide(r); verr != nil { c.row.ErrorKind = "BAD_REQUEST"; c.finish(nil, ErrBadRequest); return DecideResponse{Path: fallbackIf(r)}, ErrBadRequest }
	in, _, err := g.maskTexts(c, []string{r.Input})             // CHE như Embed; Instructions là chữ tĩnh, không che
	if err != nil { return DecideResponse{Path: fallbackIf(r)}, err }
	ch, err := runChain(c, func(Target) int { return (len(in[0]) + 3) / 4 + 64*len(r.Questions) },
		func(ctx context.Context, t Target) (provider.DecideResult, int, error) {
			d, ok := t.P.(provider.Decider)
			if !ok { return provider.DecideResult{}, 0, &provider.Error{Kind: provider.KindBadRequest} }
			res, err := d.Decide(ctx, toOpts(t.Model, in[0], r.Questions))
			return res, res.TokensIn, err
		})
	if err != nil { c.row.CascadePath = fallbackIf(r); c.finish(nil, err); return DecideResponse{Path: c.row.CascadePath}, err }
	resp := fromResult(ch.val); base := c.response(ch.target, ch.idx, "", ch.val.TokensIn, 0, ch.permit)
	resp.Provider, resp.Model, resp.TokensIn, resp.CostEst = base.Provider, base.Model, base.TokensIn, base.CostEst
	if r.Accept != nil { resp.Path = "FALLBACK"; if r.Accept(resp) { resp.Path = "FAST" } ; c.row.CascadePath = resp.Path }
	c.finish(&base, nil)
	return resp, nil
}
```
- **Hạn chót:** `begin` lấy hạn từ ctx của request gốc. Người gọi phải bọc `context.WithTimeout(ctx, cfg.DecisionTimeout)` (env `LLM_DECISION_TIMEOUT`, mặc định `800ms`), vì INTERACTIVE mặc định cho 30 s. `params.timeout_s` của tuyến chỉ đếm theo giây nên không dùng được cho 800 ms. Thử lại: INTERACTIVE `maxRet = 1` có sẵn, nhưng với 800 ms thì thực tế chỉ kịp một lần. Thêm `"retries": 0` vào `params` của tuyến DECIDE.
- **Scheduler:** không cần mã mới. `Admit` nhận `Work{Lane, Task: DECIDE, ProviderID, RPM, TPM}`, nên token bucket tự tách theo provider (`scheduler/redis.go:131`). Provider `decision` đặt `rpm_limit = 200` (Free) hoặc `1500` (Tier 1), `tpm_limit = 50000` / `1000000`. Làn: ROUTING và THREADS dùng INTERACTIVE; INJECTION hạ xuống BATCH.
- **Ngân sách:** `begin` đổi INTERACTIVE sang `Cheapest()` khi cạn ngân sách. Với `TaskDecide` thì **không** đổi (mô hình chat không biết `Decide`). Thêm `task != TaskDecide` vào điều kiện ở `gateway.go:131`. BATCH cạn ngân sách → lỗi → cascade đi FALLBACK.
- **Audit:** `AuditRow` thêm `CascadeGate`, `CascadePath` (con trỏ `string`, null khi không phải DECIDE); `audit.go` thêm hai cột vào lệnh insert. `tokens_out = 0`. `cost_est` = `cost.Of(in, 0, PriceIn, 0)`, với `price_in` đặt theo đồng / 1 triệu token (cột hiện tính bằng đồng).
- **Policy** (`internal/llm/cascade.go`, hàm thuần, có test bảng):
```go
func AddOnly(q string, eps float64) func(DecideResponse) bool {   // THREADS, INJECTION: "có" luôn nhận; "không" chỉ nhận khi P(có) ≤ ε
	return func(r DecideResponse) bool { a, ok := r.Answers[q]; return ok && (a.Noul >= 0.5 || a.Noul <= eps) }
}
func Symmetric(q string, tau float64) func(DecideResponse) bool { // ROUTING
	return func(r DecideResponse) bool { a, ok := r.Answers[q]; return ok && a.Confidence >= tau }
}
```
- **Ngưỡng:** đọc từ `params` của tuyến DECIDE. `llmconfig.Params` đã là JSON tự do nên không cần bảng mới. Khoá và mặc định trong mã: `threads_eps = 0.05`, `injection_tau = 0.5`, `routing_tau = 0.95`. Hàm `llm.DecisionThresholds(reg *Registry) Thresholds` đọc từ `Route(TaskDecide).Params`, thiếu khoá nào thì lấy mặc định.

### 5. Provider giả `decision-fake`: `internal/llm/fake/decide.go` (mới)
- `*fake.Provider` cài thêm `Decide`. Không cần loại provider mới: `type = 'fake'` + model `decision-fake`.
- **Mặc định trung tính:** mọi `noul` = 0,01; mọi `choice` có `confidence` = 0,50, lựa chọn = khoá đầu tiên theo bảng chữ cái. Với ngưỡng mặc định, kết quả là:
  - THREADS: FAST và cho qua, giống luật;
  - INJECTION: không gắn cờ;
  - ROUTING: FALLBACK về luật.
  
  Bật cổng mà không có bản ghi thì **hành vi y như trước 10.5**, nên mọi test hiện có giữ nguyên.
- **Bản ghi:** dùng chung `DemoFile` (`seed/demo/fake_llm.json`), thêm mảng `decisions: [{match: "...", answers: {...}}]`. Input (đã che) chứa đủ chuỗi `match` thì trả `answers`. Đây là cách test / seed / demo dựng ca "decision nói có" mà không gọi Google.
- **Độ trễ:** `fake.Settings` thêm `DecisionLatencyMin/Max`. Env `FAKE_DECISION_LATENCY` cùng cú pháp với `FAKE_LLM_LATENCY` (`config.go:406` `latencyRange`). Mặc định `0-0`. Khi đo TTFT trước / sau thì đặt `426ms-682ms` (p50 / p95 của PoC).
- Lỗi: dùng chung `ErrorRate` / `ErrorKind` của `Settings`. Thêm `DecisionBlocked bool` để giả 400 `safety`.
- Bộ đếm: `Controller.Decisions()`, tách khỏi `Calls()`. Như vậy các test D47 hiện có ("đúng một lời gọi sinh chữ") vẫn đếm đúng.

### 6. Gộp vào ba luồng

**6.1 Định tuyến chat: `internal/agent/analyze.go`**
1. `ruleIntent := DetectIntent(text)`. Gặp **`IntentCrisis` thì trả về ngay, KHÔNG gọi `Decide`**. Test `TestCrisisNeverReachesDecide` dùng fake có bộ đếm và khẳng định `Decisions() == 0` cho cả 20 câu của `crisis_20.json`.
2. Cổng tắt (`courses.decision_routing = false`) → giữ nguyên mã hiện tại.
3. Cổng bật:
   - chạy `Classifier.Classify` (có thể nhúng) và `Decide` **song song** bằng `errgroup`;
   - `Decide` bọc `context.WithTimeout(ctx, DecisionTimeout)`. Lỗi của `Decide` **không** làm hỏng `Analyze`: chỉ log mức warn với `trace_id`, không log nội dung;
   - Tập lựa chọn gồm 10 ý định, **bỏ** `CRISIS` và `OTHER_PERSON`: hai ý định này chỉ luật được quyết;
   - `resp.Path == "FAST"` thì `out.Intent = Intent(answer.Choice)`; ngược lại dùng `ruleIntent`;
   - các bước sau giữ nguyên: hỏi hộ (`otherPerson`), sửa ý định cá nhân bằng tương đồng, `extraPersonal`. Nhờ vậy logic danh tính / PII không đổi.
4. Instructions và criteria lấy nguyên văn từ `docs/research/poc/jev-decision/poc.py` (`INSTR["C"]`, `INTENTS`), vì ngưỡng 0,95 được đo trên chính chữ đó. Đổi chữ thì phải đo lại.
5. Đọc công tắc lớp: `Analyzer` nhận thêm `Gates GateSource` (interface `Enabled(ctx, courseID, Gate) (bool, error)`). Bản cài đặt đọc `courses` qua sqlc, có cache Redis `ep:course:gates:{id}`, xoá khi `PUT` công tắc (luật 15). Lỗi đọc → coi là tắt.

**6.2 Tường lửa Threads: `internal/thread/firewall.go` `CheckPost`**
- Luật (`Detect` + `Classify`) giữ nguyên và chạy trước. `Decide` **song song** với `Classify` (cùng `errgroup`), đầu vào là `full`. Gateway che danh tính trước khi gửi.
- Hai câu hỏi noul:
  - `personal`: câu hỏi về điểm / vắng / hồ sơ của chính người gửi hay của một người cụ thể;
  - `named_person`: có nêu họ tên một người cụ thể.
  
  Chữ lấy theo `INSTR["A"]` của PoC, tách đôi.
- Chỉ dùng kết quả decision khi luật cho qua (`len(v.Reasons) == 0`). `Accept = AddOnly` trên **cả hai** câu, ε = `threads_eps`:
  - FAST và có câu `≥ 0,5` → chặn: `personal` → `PERSONAL_QUESTION`; `named_person` → `NAME`, không có lối "Ẩn rồi đăng" vì không có khoảng chữ để thay;
  - FAST và cả hai `≤ ε` → cho qua;
  - FALLBACK (phân vân, lỗi, `KindBlocked`) → `llm.Structured` qua 9router: Task `CLASSIFY`, làn NEAR_REALTIME, schema `{personal: bool, named_person: bool}`, cùng instructions. `true` thì chặn;
  - **cả decision lẫn Structured đều lỗi → trả lỗi**: `gate()` đã đổi lỗi của `CheckPost` thành `unavailableErr()`, `precheck` trả 503. Đây là **fail-closed** theo plan 10.5: bài không được lưu, người dùng thấy "đang bận, thử lại".
- `PrecheckOut` không đổi hợp đồng: lý do vẫn là `PERSONAL_QUESTION` / `NAME`. Không có trường mới trong `openapi`.

**6.3 Cờ chèn lệnh khi chấm: `internal/grading/essay.go`** (sau `ScanInjection`, dòng 165)
- Cổng bật và bài không trống → `Decide` trên `cleaned`:
  - `Lane = BATCH`, `Gate = INJECTION`;
  - một câu noul theo `INSTR["D"]` của PoC;
  - `Accept = AddOnly(…, 0)`: chỉ để ghi nhận đường đi, vì cờ là phép OR;
  - `p ≥ injection_tau` → thêm `FlagInjectionSuspected` (nếu chưa có).
- P-11-9 (`Structured`) đi trong lời gọi chấm lượt 1 có sẵn: thêm trường `injection_suspected: boolean` vào schema chấm. Cờ cuối cùng = regex OR P-11-9 OR decision.
- Lỗi của `Decide` → bỏ qua lớp này, ghi warn. **Không** làm `TransientError`, vì cờ không được làm hỏng việc chấm. Không đổi điểm (TLR-4, D60 (5)).
- Ranh giới luật 5: `internal/grading` chỉ được gọi `Decide` trong `injection.go`. Test kiểm qua `go/ast` hoặc `grep`: không tệp nào khác trong `internal/grading`, và không tệp nào trong `internal/grade`, gọi `.Decide(`.

### 7. Cấu hình, quyền, API (dev cập nhật `openapi.yaml`)

| Cấu hình | Ở đâu | Mặc định | Ai đổi |
| --- | --- | --- | --- |
| Provider `decision` (khoá, `rpm_limit`, `tpm_limit`) | `llm_providers` (khoá mã hoá như hiện nay) | không có | ADMIN, `/settings/llm` |
| Tuyến `DECIDE` + `params` (`threads_eps`, `injection_tau`, `routing_tau`, `retries: 0`) | `llm_task_routes` | `decision-fake` khi `LLM_PROVIDER=fake` | ADMIN, API tuyến có sẵn (`PUT /admin/llm/routes`) |
| Công tắc ba cổng theo lớp | `courses.decision_*` | tắt | ADMIN: `PUT /admin/courses/{courseId}/decision-gates {threads, injection, routing, version}` → `409` nếu lệch `version`. BA chốt quyền cho TEACHER ở spec |
| `LLM_DECISION_TIMEOUT` | env | `800ms` | vận hành |
| `FAKE_DECISION_LATENCY` | env | `0-0` | đo |

Thống kê:
- `GET /admin/llm/usage?group=task` đã có. Thêm `group=cascade`: mỗi `cascade_gate` trả số lời gọi, số `FAST`, số `FALLBACK`, số lỗi, p50 / p95 `latency_ms`, tổng `cost_est`.
- `/observability` (sprint 11) đọc từ đây. Ở 10.5 chỉ cần API và khối số trên `/settings/llm`.

### 8. Test cần có (CI chỉ dùng fake; không test nào gọi Google)

| Test | Gói | Khẳng định |
| --- | --- | --- |
| `TestDecisionProviderContract` | `provider` (`httptest.Server`) | Đúng URL / header `x-goog-api-key` / thân JSON; khoá không nằm trong URL; 200 → đáp án; bảng mã HTTP → `Kind` (gồm 400 `safety` → `KindBlocked`); thân thiếu khoá câu hỏi → `KindBadResponse`; `Detail` không chứa khoá |
| `TestDecideMasksInput` | `llm` | Input có tên trong danh sách lớp → payload tới provider chỉ có placeholder (đọc từ `fake` `Payloads()`) |
| `TestDecideAuditRow` | `llm` | Đúng một dòng `task=DECIDE`, `tokens_out=0`, `cascade_gate`/`cascade_path` đúng; lỗi → `cascade_path='FALLBACK'` |
| `TestDecideLaneAndBucket` | `llm/scheduler` | DECIDE xin permit trên `ProviderID` của provider `decision`, không ăn RPM của 9router; INJECTION ở làn BATCH |
| `TestDecideTimeout` | `llm` | fake trễ 2 s, ctx 800 ms → lỗi `ErrDeadline` trong ≤ 900 ms, path FALLBACK |
| `TestDecideBudgetNoCheapest` | `llm` | Ngân sách cạn → không đổi sang mô hình chat |
| `TestCascadePolicies` | `llm` | Bảng: `AddOnly` / `Symmetric` × (p, conf) ở hai bên ngưỡng |
| `TestCrisisNeverReachesDecide` | `agent` | 20 câu `crisis_20.json`, cổng ROUTING bật → `Decisions() == 0`, intent CRISIS |
| `TestRoutingFastAndFallback` | `agent` | Bản ghi `confidence 0,97` → dùng ý định của decision; `0,90` / lỗi / hết giờ → ý định của luật; cổng tắt → 0 lời gọi; `routeCases` 61/61 với fake trung tính |
| `TestRoutingParallelLatency` | `agent` | Fake nhúng 300 ms + decision 300 ms → `Analyze` < 450 ms (song song) |
| `TestThreadsDecisionAddOnly` | `thread` | Luật chặn → kết quả không đổi; luật cho qua + decision `p = 0,9` → chặn `PERSONAL_QUESTION` / `NAME`; `p = 0,02` → cho qua; `p = 0,2` → Structured quyết; decision + Structured đều lỗi → `CheckPost` lỗi → 503 |
| `TestThreadsNeverRelaxes` | `thread` | Không có bản ghi decision nào làm một bài luật đã chặn được cho qua (bảng E1 `test`) |
| `TestInjectionGateOr` | `grading` | Regex / Structured / decision, mỗi lớp một mình gắn được cờ; decision lỗi → chấm vẫn xong, không đổi điểm |
| `TestDecideOnlyInAllowedPackages` | gốc | `grep` / `go/ast`: `.Decide(` chỉ xuất hiện trong `internal/agent`, `internal/thread`, `internal/grading/injection.go`, `internal/llm` |
| `TestNoGenaiSDK` | gốc | `go.mod` không có `google.golang.org/genai` |
| Hợp đồng `openapi` | `internal/contract` | `decision-gates`, `usage?group=cascade` |

Ngoài CI:
- E2E: Threads với bản ghi "decision chặn", đi trọn F4 cả nhánh lỗi (BA ghi vào FLOWS).
- k6 `chat.js`: so TTFT với cổng ROUTING tắt và bật, `FAKE_DECISION_LATENCY=426ms-682ms`, cùng stack. Mục tiêu: TTFT p95 không tăng quá 10 %.

### 9. Rủi ro
- **Định tuyến chạy decision trước luật** (P-11-10): một câu decision tự tin ≥ 0,95 nhưng sai sẽ ghi đè luật đúng. PoC: 0/51 câu FAST bị sai, nhưng mẫu nhỏ. Có `TestRoutingFastAndFallback` và `routeCases` canh hồi quy. Phương án lùi: tắt cổng theo lớp.
- **Threads fail-closed**: khi decision và 9router cùng sập, mọi bài mà luật đã cho qua ở lớp bật cổng sẽ nhận 503. Ở lớp tắt cổng không có rủi ro này. Ghi vào runbook (guide).
- **Trễ đi Mỹ** 0,4–0,65 s. Chạy song song với lần nhúng thì gần như không lộ ra; nếu lộ thì TTFT tăng. Phải có số đo k6.
- **`noul` chỉ trong [0,01; 0,99]**: ε = 0,05 vẫn chạm được, nhưng đừng đặt ε < 0,01.

---

## TL-2 (rà D60 / ARCHITECTURE / SYSTEM_DESIGN cho khớp plan 10.5)

Đã rà và sửa trong cùng commit:

| Chỗ lệch | Trước | Sửa thành |
| --- | --- | --- |
| Threads khi decision và LLM cùng lỗi | `cascade.md`, `SYSTEM_DESIGN` §3.1: "giữ kết quả luật" | **fail-closed**: `CheckPost` lỗi → 503, không lưu (plan 10.5) |
| Công tắc theo lớp | chưa có | `courses.decision_threads/injection/routing`, mặc định tắt (D60, ARCHITECTURE §4, §6) |
| Tên provider giả | "`decision` cũng có bản fake" | `type = 'fake'`, model **`decision-fake`** (khớp plan) |
| Lược đồ | chưa có | `00020_decision.sql`: type `decision`, kind `decision`, task `DECIDE`, `llm_audit.cascade_gate/path` |
| Env | chỉ `LLM_DECISION_TIMEOUT` | thêm `FAKE_DECISION_LATENCY` |
| Lỗi 400 lọc an toàn | "coi là dương" (dành cho khủng hoảng, nay bỏ) | `KindBlocked` → cascade FALLBACK |
| Số migration dự kiến | ARCHITECTURE §4: `00020 indexes_tuning`, `00021 production` | `00020 decision` (10.5 chạy trước); dời `indexes_tuning` → `00021`, `production` → `00022`. Ghi `ALTER` là ngoại lệ có chủ ý, theo tiền lệ `00015` |
| D60 | thiếu công tắc lớp, cột audit, migration | đã thêm (4) công tắc + fail-closed của Threads, (2) `cascade_gate/path` + `00020`, (6) `decision-fake` |

Tệp đã sửa: `docs/DECISIONS.md` (D60), `docs/ARCHITECTURE.md` (§4, §6, §8), `docs/SYSTEM_DESIGN.md` (§3.1), `docs/research/poc/jev-decision/cascade.md`. Bản phác `docs/guides/decision-model.md`: dev bổ sung số đo thật và ảnh màn sau L5–L6.

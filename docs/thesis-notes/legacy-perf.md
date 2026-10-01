# Vì sao Project III chạy chậm — nguyên liệu cho chương "Lý do viết mới"

Nguồn: đọc mã `legacy/` ngày 2026-10-01 (scout PM, không chạy dịch vụ; chỗ ghi [INFERENCE] là suy luận). Đường dẫn file bên dưới là đường dẫn trước khi dời vào `legacy/`.

Tóm tắt: chậm vì thiết kế, không vì ngôn ngữ. Mỗi câu hỏi gọi 2–6 lần LLM nối tiếp trước chữ đầu tiên (tạo thread kèm AI: 6 lần, 4 lần là cùng một phân loại); thread list N+1; DB đặt ở xa; trích tài liệu chạy chung tiến trình với chat; toàn bộ dồn vào một container HF CPU yếu. Các luật thay thế ghi ở D47.

---

# Why Project III was slow: ranked root causes

No measured end-to-end latency exists in the repo. I didn't run any services; every claim below comes from reading the code, and anything marked [INFERENCE] I did not observe directly.

## LLM calls per chat question

Python skips its own classifier when Java sends the `java_preflight` tag, which Java always does after `ChatStreamingService.java:79`. "Before first token" counts calls that must finish completely before the student sees anything.

| Path | Chat LLM calls | Embedding calls | Calls finished before first token |
|---|---|---|---|
| Private chat, CONVERSATIONAL | 2 (classify, answer) | 0 | 1 |
| Private, ACADEMIC, cache hit | 2 (classify, answer) | 1 | 1, plus embed and Redis |
| Private, ACADEMIC, cache miss | 3 (classify, rerank, answer) | 1 | 2, plus embed |
| Private, PROCEDURAL regulation prefetch | 4 (classify, rewrite, synthesis up to 2500 tokens, answer) | 1 | 3, plus embed |
| Private, PROCEDURAL deadline ("Lab N" detected) | 2 (classify, answer) | 0 | 1, plus a new psycopg2 connection |
| Private, UNCERTAIN or PROCEDURAL without a match (tool loop) | 4–5 | 1 | turn 1 streams; can loop up to 10 turns |
| Forum ask-ai | path above + 1 (firewall classify, run again) | same | +1 |
| "@AI" reply in a thread | path above + 2 (reply firewall, ask-ai firewall) | same | +2 |
| New thread with auto-AI | path above + 3 (frontend submit classify, createThread firewall, ask-ai firewall) | same | e.g. ACADEMIC cache miss = **6 LLM calls, 5 before first token, 4 of them the same classification** |

The tool-loop row breaks down as: classify → turn 1 (tool planning) → RAG (rerank, or rewrite + synthesis) → turn 2 answer. If the model skips the tool on turn 1, it streams a full answer, then the forced fallback runs RAG and a second answer (`agent.py:690-722`), so the student gets two answers glued together.

On top of that, the thread-create modal fires another classifier call after every 800 ms typing pause once the draft is at least 32 characters (`useThreadsList.ts:107`). These run in the background and eat into the 30-per-minute classify rate limit.

## Ranked causes

### 1. Several LLM calls run one after another, and the middle ones are not streamed
**Impact:** time to first token is roughly classify + embed + the full generation of rerank or synthesis + the final call's first token. That is 2–3 complete LLM round trips before any visible output. Regulation answers are generated twice: the synthesis call writes up to 2500 tokens, then the final call rewrites them.

**Evidence:**
- Classify, non-streamed: `ChatStreamingService.java:79`, `guardrails.py:212-264`
- Rewrite, 3 s timeout: `tools.py:257-270`, called at `:1405`
- Embed: `tools.py:1409`
- Rerank JSON, 500/900 output tokens, non-streamed: `tools.py:1626-1635`
- Synthesis, 2500 tokens, non-streamed: `tools.py:90`, `:633-643`
- Final streamed answer: `agent.py:466`
- Prefetch then answer: `agent.py:407-452`
- The cache stores rerank facts, not the final answer, so a hit still pays the final LLM call: `tools.py:1424`, `cache_repo.py`
- Nothing is sent over SSE until classification finishes: `ChatStreamingService.java:79-94`

**Principle:** one streamed generation per question. Retrieval stays deterministic (embed + hybrid SQL, optionally a cheap reranker, but not a generating LLM). Intent comes from rules or a tiny model, run in parallel or folded into the answer call. Cache the final answer. Send a status SSE event within about 300 ms.

### 2. The same classifier runs again on forum paths
**Impact:** +1 to +3 serial LLM round trips for the same text. Java's firewall timeout is 3 s while Python keeps generating, so slow calls burn cost and are thrown away.

**Evidence:**
- `ChatStreamingController.java:85` firewall, then `ChatStreamingService.java:79` classify again with the same PUBLIC hint (FORUM is normalised to PUBLIC)
- `ForumService.java:40` (createThread) and `:135` (reply)
- Frontend `useThreadsList.ts:107` (typing) and `:127` (submit)
- `PublicPrivacyFirewallService.java:42`, `IntentClassifierClient.java:19`, `PythonAiOrchestratorClient.java:39`

**Principle:** classify once per message, at one layer. Cache the result by content hash and pass it downstream. PII checks are regex first; the LLM is used only when needed and never repeated.

### 3. Fallback paths multiply LLM calls
**Impact:** UNCERTAIN or unmatched PROCEDURAL questions go into a tool loop of up to 10 turns (`agent.py:342`, `:454`). If the model doesn't call a tool, there is a forced RAG plus a second answer call (`agent.py:690-722`), so 4–5+ LLM calls and a duplicated answer.

[INFERENCE] `classify_and_guard` uses `max_completion_tokens=100` on gpt-5.4-mini with no `reasoning_effort` anywhere in `src/` (grep came back empty). If that model spends reasoning tokens, the JSON can be cut off, which triggers the safe fallback to UNCERTAIN (`guardrails.py:261`, fallback in the except branches). That pushes more traffic into this slow path. Each non-streamed call would also pay hidden reasoning time.

**Principle:** a fixed pipeline with no open-ended agent loop for Q&A. Pin the model and its reasoning effort. Treat a classifier failure as "do RAG", not as the start of a multi-turn agent.

### 4. N+1 queries and over-fetching against a remote database
**Impact:** the thread list costs about 1 + 20×(4+k) queries per page: `ForumService.java:200-214` (tags, a findById per tag, last post, count) and `:284` (author). Message lists also look up the author once per post (`toAuthor`).

The thread detail page calls this whole list endpoint just to read the title (`threads/[id]/page.tsx:724-727`). Creating a thread waits for the list reload before navigating (`useThreadsList.ts:140`).

The DB and Redis default to a public IP (`application.yaml:3`, `:28`), with the comment "Tăng lên 30s vì DB ở xa" ("raised to 30s because the DB is far away") at `:9`. That multiplies every round trip. Also:
- `show-sql: true` (`application.yaml:20`)
- the JWT filter queries the DB per request (`UserDetailsServiceConfig.java`)
- 2–3 serial risk-level queries before every answer (`analytics_repo.py:50-76`)

**Principle:** set-based JOIN or aggregate queries for lists, with cursor paging. A dedicated GET for one thread. App and DB in the same place. Identity taken from JWT claims. Risk level carried in the request or cached.

### 5. Document ingestion runs inside the chat process
**Impact:** Docling parsing and OCR run in a thread of the same Python process and event loop that serve chat streams (`document_callback.py`, `ingestion_worker.py`, `document_parser.py:125-134`). Chat competes with them for CPU and the GIL [INFERENCE], and for the 5-connection asyncpg pool (`connection.py:23-24`).

On the Hugging Face deploy, OMP_NUM_THREADS=1 and a basic-tier CPU is shared by 6 processes (`Dockerfile`, `supervisord.conf`). Every chunk of a document goes into a single embeddings request (`data_pipeline/pipeline/embedding.py`).

**Principle:** ingestion in a separate worker process with a queue. Batched, rate-limited embedding. A separate DB pool.

### 6. Blocking calls and no connection pooling in the Python service
**Evidence:**
- `psycopg2.connect` on every tool call (`tools.py:1051`, `:1134`, `:1254`). These are off the event loop via `to_thread`, but each pays a full connect handshake.
- FastAPI `/api/v1/assignments` GET and POST call psycopg2 synchronously on the event loop (`main.py:170`, `:181`). That event loop is shared with the gRPC stream server (`grpc_server.py:428`), so these calls stall every stream.
- asyncpg pool max is 5.

**Principle:** one async connection pool per process, sized for expected concurrency. No synchronous I/O in async handlers.

### 7. Deployment: everything in one Hugging Face container
**Impact:** six runtimes share one CPU Basic tier: Postgres, Redis Stack, Python with torch installed, a JVM at -Xmx512m, Node and nginx (`supervisord.conf`, `Dockerfile`, `HUGGINGFACE_DEPLOY.md:32`). Free Spaces sleep, and the first request after sleep waits for all of them to boot (`HUGGINGFACE_DEPLOY.md:176`), including JVM startup and migrations (`entrypoint.sh`).

**Principle:** few, light processes; managed DB and Redis; no torch in the request-serving image; an always-on or warm instance.

### 8. Frontend waterfalls and render cost
**Evidence:**
- Every route is a `"use client"` component that fetches in useEffect.
- Private chat loads in sequence: sessions, then messages (`usePrivateChatSessions.ts:21`, `:151`). Both are unbounded (`ChatSessionRepository`, `ChatMessageRepository`).
- WorkspaceLayout remounts on every page and fetches the avatar blob twice (`AuthenticatedAvatar.tsx:38`), plus preferences (`PreferenceSyncProvider.tsx:33`).
- The streaming message is re-parsed with react-markdown + remark-gfm on every token (`chat/page.tsx:171`, `:573`; `threads/[id]/page.tsx:327`, `:824`), which costs O(n²) CPU over a long answer.
- The documents page polls every 5 s (`useDocuments.ts:115`) against unbounded `findAll` calls (`AdminService.java:92`, `:131`).

**Principle:** server-render initial data; render the shell once in a layout; batch or throttle token rendering (for example per animation frame); cursor-paged lists; push status over SSE instead of polling.

### 9. Retrieval SQL details (low impact at the current corpus size)
- Full-text search computes `to_tsvector('simple', content)` per row with no GIN index (`vector_repo.py`, keyword search; the migrations only have a GIN index on metadata, `V7__indexes.sql:19`).
- Vector search orders by `embedding <=> $1, document_id, chunk_index, id`. [INFERENCE] The extra sort keys may stop the HNSW index (`V7__indexes.sql:16`) from being used. Its `document_type` filter is applied after the JOIN. LIMIT 50 is above pgvector's default `ef_search` of 40.
- The neighbor fetch runs one query per document.

**Principle:** order by distance only, using the index; filter inside the same query on an indexed column; stored tsvector with a GIN index.

### 10. Smaller overheads
- 100% trace sampling to an OTLP endpoint (`application.yaml:80`).
- `show-sql` logging on.
- A new ObjectMapper per AI post or message (`ForumService.java:228`, `PrivateChatService.java:69`).
- Tokens are buffered until the `[CONFIDENCE: NN]` tag closes, up to 80 characters (`agent.py:487`).
- Status lines ("🔍 …") are streamed into the answer text and saved with it.

**Principle:** sampled tracing, singletons, and structured metadata instead of text inside the answer.

## Not a cause
The SSE proxy: nginx `/backend/` has `proxy_buffering off` (`huggingface/nginx.conf`). Java relays Flux elements without aggregating them (`PythonAiOrchestratorClient.streamResponse` unicast sink). Python creates one OpenAI client per process (`tools.py:48-66`, already fixed by the audit).

## Measured numbers in the repo
All offline and mocked, from `benchmarks/offline_eval_baseline_comparison.json` and `docs/AI_CORE_AUDIT_2026-07.md` §4:
- Event-loop stall with a 300 ms-slow Redis: **291.4 ms before → 1.2 ms after** (Redis call moved to `to_thread`).
- OpenAI clients created per 5 RAG calls: **5 → 1**.
- Identical RAG call ×3 → real executions: **3 → 1**.
- Agent framework overhead (mocked stream): **p50 0.05 → 0.11 ms, p95 0.09 → 0.22 ms**, which is negligible.
- Chat-path import RSS: **84.4 → 87.9 MB**; Docling and torch not loaded on the chat path.

There are no live TTFT, p95 or Locust results. `benchmarks/run_benchmark.py` and `locustfile.py` exist, but their outputs (`predictions.jsonl`, `tier1_summary.json`, `load/*_stats.csv`) are not in the repo.

The only figures are targets and claims, never measured:
- README:30, 177: TTFT < 1.5 s
- BENCHMARK_USAGE_GUIDE.md:115: release target `ttft_p95_ms < 3000`
- PRD.md:24 and SYSTEM_DESIGN.md:156 (new design): TTFT ≤ 1.5 s on a cache hit, ≤ 4 s with RAG
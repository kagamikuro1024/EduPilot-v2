# AI Core Audit, Refactor & Evaluation — July 2026

Scope: Python AI service only (`src/`, `data_pipeline/`, `benchmarks/`). No frontend,
Java, REST/gRPC/SSE contract, database schema, or business-rule changes.

## 1. Architecture summary

```
Java gateway ──gRPC──> StreamAIResponse (src/grpc_server.py)
  ├─ guardrail+intent: java_preflight tag (skip LLM) or classify_and_guard (src/guardrails.py)
  │    └─ violation → grpc ABORT FAILED_PRECONDITION "violation-reason:…"
  ├─ risk level lookup (src/database/analytics_repo.py)
  └─ run_agent_loop_stream (src/agent.py, ≤10 turns)
       ├─ ACADEMIC → deterministic prefetch query_course_materials, tools disabled
       ├─ PROCEDURAL + regulation keywords → prefetch query_regulations, tools disabled
       ├─ otherwise tool loop: check_assignment_deadline / get_assignments (psycopg2 via
       │   to_thread), query_course_materials / query_regulations (RAG), search_web
       ├─ RAG (src/tools.py): semantic cache (Redis, course-only) → pgvector + FTS keyword
       │   RRF merge (src/database/vector_repo.py) → distance/hint filters → LLM rerank
       │   (course) or single-pass synthesis (regulation) → citations via contextvars
       └─ events TOKEN/STATUS/CITATIONS/METRICS/ESCALATION/system_error/DONE
            → protobuf AIResponse chunks; DONE carries ResponseMetadata
              (agent_used pipe-tags parsed by Java + Citation list) → Java SSE
Ingestion: gRPC ProcessDocument → background task → semaphore(DOCUMENT_INGESTION_CONCURRENCY=1)
  → lazy Docling import → parse (anyio.to_thread; pypdfium low-memory path) → clean → chunk
  → embed → asyncpg persist → status callback to Java.
```

## 2. Problems found (severity, evidence, fix state)

| # | Sev | Finding | Evidence | State |
|---|-----|---------|----------|-------|
| C1 | Critical | Real Redis credential + public IP committed as `REDIS_URL` code default | `src/config.py` (`redis://:…@legacy-host.invalid:6380/0`) | Fixed: safe localhost default; env unchanged. **Rotate that credential** — treat as leaked |
| C2 | Critical | Model-generated tool args splatted via `**args`; unknown/missing/typed-wrong arg raised TypeError and killed the stream | `src/tools.py execute_tool`; offline eval `extra_tool_arg`/`missing_required_arg` before: crash | Fixed: schema validation + never-raise contract |
| C3 | Critical | Agent's `system_error` (OpenAI 429/timeout) event dropped by grpc_server → user saw empty response, stream ended without `is_finished` | baseline `grpc_server.py` chunk-type switch; eval `grpc_429_visibility` before: false | Fixed: mapped to terminal chunk |
| H1 | High | Synchronous Redis KNN search executed on the event loop (no socket timeouts) | `check_semantic_cache` call in `_execute_rag_pipeline`; eval `slow_cache_event_loop` before: 291 ms stall | Fixed: `asyncio.to_thread` + socket timeouts |
| H2 | High | `asyncio.create_task` without reference in `ProcessDocument` (task can be GC'd mid-ingestion) | baseline `grpc_server.py:274` | Fixed: `_BACKGROUND_TASKS` registry |
| H3 | High | Only RateLimit/Timeout caught; any other OpenAI/unexpected error crashed the generator and leaked `str(e)` to students | baseline `agent.py` except clause + `grpc_server` fallback `Internal error: {e}` | Fixed: APIError→503, Exception→500 events; sanitized gRPC fallback |
| H4 | High | Escalation path dropped the first content fragment after `[CONFIDENCE]` tag | baseline `agent.py:290-300`; eval `escalation_content` before: 41% delivered | Fixed: fragment yielded in both branches |
| H5 | High | New `AsyncOpenAI` client (new HTTP pool) per RAG call / chunk correction | baseline `tools.py:1202`; eval `client_instantiations` before: 5/5 calls | Fixed: lazy singleton keyed on (class, key) — monkeypatch-safe |
| M1 | Medium | Stale test asserted removed low-confidence early-return behavior (failing in baseline) | `test_agent_routing.py` | Updated to current contract |
| M2 | Medium | Tool results appended sorted by tool-call **id** while assistant message uses **index** order | baseline `agent.py:369` | Fixed: index order both |
| M3 | Medium | Blocking file write on event loop in 429 path; `.ai-log/` never created so audit write always failed | baseline `agent.py:345` | Fixed: `to_thread` + makedirs |
| M4 | Medium | Duplicated prefetch/citation/grounding blocks; dead code (`tools` filtered then overwritten, unused `finish_reason`, duplicate comments) | baseline `agent.py:205-207` | Refactored: `_build_system_prompt`, `_history_to_messages`, `_grounded_context_message`, `_consume_rag_meta` |
| M5 | Medium | Repeated identical tool calls re-executed every turn (duplicate side effects/cost) | eval `duplicate_rag_calls` before: 3 executions | Fixed: per-request result ledger |
| M6 | Medium | `gpt-4o-mini` hardcoded in rewrite/synthesis/rerank | 3 sites in `tools.py` | Centralized: `RAG_UTILITY_MODEL` env-overridable |
| M7 | Medium | Retrieved document text injected without an explicit trust boundary (prompt-injection surface) | grounded system messages | Hardened: SECURITY guard + BEGIN/END fences (all 3 grounding paths) |
| M8 | Medium | Full user queries logged at INFO in agent paths | `agent.py` prefetch/tool logs | Truncated to 120 chars via `_log_snippet` |
| M9 | Medium | No explicit OpenAI timeout/retry bounds (SDK default 600 s) | `create_agent`, tools client | `OPENAI_TIMEOUT_SECONDS=120`, `OPENAI_MAX_RETRIES=2` (env-overridable) |
| L1 | Low | `tools.py` 1.6 k-line monolith; further split intentionally deferred — existing tests monkeypatch module globals (`tools.search_vectors`, `tools.AsyncOpenAI`), a move would break their patch surface | — | Documented |
| L2 | Low | Misc: unused `DATABASE_URL` import in agent; `finish_reason` dead var; duplicate comments | — | Cleaned in rewrite |

## 3. Files changed (AI core only)

Modified: `src/config.py` (+ user's own model-default change kept), `src/agent.py`,
`src/tools.py`, `src/grpc_server.py`, `src/main.py` (shared-client swap only),
`src/database/cache_repo.py`, `src/tests/test_agent_routing.py` (stale test).
Added: `src/tests/test_execute_tool_validation.py`, `test_agent_stream_reliability.py`,
`test_grpc_stream_events.py`, `test_semantic_cache_tolerance.py`, `test_rag_thresholds.py`,
`test_prompt_injection_boundary.py`, `test_ingestion_bounds.py`,
`benchmarks/offline_eval.py`, `benchmarks/offline_eval_baseline_comparison.json`, this report.
Untouched: frontend/, backend-java/, shared-proto/, db/, generated `ai_service_pb2*`,
`guardrails.py` decisions, retrieval thresholds, chunking, Docling parser, docker/compose.

## 4. Before / after evaluation (offline, deterministic, no paid APIs)

`benchmarks/offline_eval.py` run against the pristine tree ("before") and the refactored
tree ("after"); full JSON in `benchmarks/offline_eval_baseline_comparison.json`.

| Metric | Before | After |
|---|---|---|
| Adversarial robustness (clean terminal stream) | 4/8 | **8/8** |
| Extra/missing/typed-wrong tool args | stream crash | handled |
| OpenAI connection error / unexpected exception | stream crash | terminal system_error 503/500 |
| Escalation: post-tag content delivered | 41.3 % | **100 %** |
| User sees terminal message on OpenAI 429 (gRPC) | no | **yes** |
| Identical RAG call ×3 → real executions | 3 | **1** |
| Event-loop stall with 300 ms-slow Redis | 291 ms | **1.2 ms** |
| OpenAI clients created per 5 RAG calls | 5 | **1** |
| Invented citations (rerank points at bogus chunk) | 0 | 0 (invariant kept) |
| Regulation-prefetch recall (24 PROCEDURAL GT cases) | 0.875 | 0.875 (unchanged by design) |
| Agent framework overhead p50 (mocked stream) | 0.05 ms | 0.11 ms (negligible vs LLM latency) |
| Import RSS chat path / Docling loaded | 84 MB / no | 88 MB / no |

Retrieval relevance/groundedness scoring against live pgvector data and the RAGAS
pipeline require the deployed stack + OpenAI key (`benchmarks/run_benchmark.py`); no
retrieval threshold, ranking, chunking, or prompt-grounding rule was altered, so
retrieval behavior is unchanged by construction. Re-run the live benchmark before the
next release to confirm.

## 5. Tests

Baseline: 56 passed, 1 failed (stale). After: **104 passed, 0 failed**
(57 existing incl. repaired stale test + 47 new). All tests are offline: OpenAI, Redis,
asyncpg, and gRPC transport are faked; no secrets needed
(`ADMIN_TOKEN`/`OPENAI_API_KEY`/`DATABASE_URL` dummies suffice).

Commands:
```
ADMIN_TOKEN=x OPENAI_API_KEY=sk-test DATABASE_URL=postgresql://t:t@localhost:5432/t \
  python -m pytest src/tests/ -q
PYTHONPATH=. python benchmarks/offline_eval.py --label after
```

## 6. Contract & deployment verification

- gRPC: proto untouched; generated pb2 files untouched; servicer event mapping verified by
  contract tests (agent_used pipe-tag format, citation schema incl. 2000-char snippet cap,
  FAILED_PRECONDITION abort path preserved).
- SSE: Java-side mapping unchanged (chunk → data, is_finished → final payload); Python
  emits the same chunk shapes; new behavior only *adds* a terminal chunk on provider errors
  that previously ended silently.
- REST: `src/main.py` route set identical to baseline (verified by route diff).
- Env vars: all previous vars honored; new optional ones with behavior-preserving defaults:
  `RAG_UTILITY_MODEL`, `OPENAI_TIMEOUT_SECONDS`, `OPENAI_MAX_RETRIES`,
  `REDIS_SOCKET_TIMEOUT_SECONDS`, `REDIS_CONNECT_TIMEOUT_SECONDS`.
- HF CPU Basic: lazy Docling/PyTorch confirmed by regression test
  (`test_chat_only_imports_do_not_load_docling`); ingestion semaphore default 1 verified
  (`test_ingestion_semaphore_bounds_concurrency`); bounded Docling workers untouched.

## 7. Remaining risks

- The leaked Redis credential (`legacy-host.invalid:6380`, password in git history) must be
  rotated server-side; code fix alone does not un-leak it. `docker-compose.yml` still
  defaults `REDIS_PASSWORD` to the same value (infra file — left unchanged).
- `run_agent_loop_stream` broad `except Exception` now converts programming errors into
  clean 500 events; they are logged with stack traces but no longer crash loudly in dev.
- Sandbox verification could not run Docker or the live Java stack; live smoke
  (chat + upload) recommended on the Space after deploy.

## 8. Recommendations intentionally NOT implemented (would change behavior — need approval)

1. Remove/gate the `FORCE_ESCALATE` string hook in `guardrails.py` (user-triggerable).
2. Classifier failure currently fails **open** (UNCERTAIN, no violation); consider fail-closed for PUBLIC channels.
3. `max_completion_tokens=100` in the classifier can truncate JSON → silent safe-default; raising it changes classification outcomes.
4. `search_web` (DuckDuckGo) remains model-callable; consider removing from the registry.
5. Semantic cache for REGULATION queries (currently intentionally disabled).
6. CORS `allow_origins=["*"]` in FastAPI; REST `detail=str(e)` in diagnostic endpoints.
7. Empty-LLM-response for non-greeting intents still yields a silent METRICS/DONE tail; could emit a fallback message.
8. `analytics_worker.py` writes fake `false_positive_rate=0.02` (TIP-004) — business decision.

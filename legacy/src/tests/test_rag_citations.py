import sys
import types
import asyncio
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

fake_redis = types.ModuleType("redis")
fake_redis.Redis = type("Redis", (), {"from_url": staticmethod(lambda *_args, **_kwargs: None)})
sys.modules.setdefault("redis", fake_redis)
sys.modules.setdefault("redis.commands", types.ModuleType("redis.commands"))
sys.modules.setdefault("redis.commands.search", types.ModuleType("redis.commands.search"))
field_mod = types.ModuleType("redis.commands.search.field")
field_mod.TextField = object
field_mod.VectorField = object
sys.modules.setdefault("redis.commands.search.field", field_mod)
index_def_mod = types.ModuleType("redis.commands.search.index_definition")
index_def_mod.IndexDefinition = object
index_def_mod.IndexType = type("IndexType", (), {"HASH": "HASH"})
sys.modules.setdefault("redis.commands.search.index_definition", index_def_mod)
query_mod = types.ModuleType("redis.commands.search.query")
query_mod.Query = object
sys.modules.setdefault("redis.commands.search.query", query_mod)
fake_asyncpg = types.ModuleType("asyncpg")
fake_asyncpg.Pool = object
fake_asyncpg.create_pool = None
sys.modules.setdefault("asyncpg", fake_asyncpg)
fake_document_callback = types.ModuleType("src.document_callback")
async def _noop_run_ingestion_with_callback(*_args, **_kwargs):
    return None
fake_document_callback.run_ingestion_with_callback = _noop_run_ingestion_with_callback
sys.modules.setdefault("src.document_callback", fake_document_callback)

import src.grpc_server as grpc_server
import src.tools as tools
import src.database.vector_repo as vector_repo
import src.agent as agent


class _AcquireCtx:
    def __init__(self, conn):
        self.conn = conn

    async def __aenter__(self):
        return self.conn

    async def __aexit__(self, *_args):
        return False


class _FakePool:
    def __init__(self, conn):
        self.conn = conn

    def acquire(self):
        return _AcquireCtx(self.conn)


def test_search_vectors_returns_citation_ready_fields(monkeypatch):
    class _Conn:
        async def fetch(self, *_args, **_kwargs):
            return [
                {
                    "chunk_id": "c1",
                    "document_id": "d1",
                    "chunk_index": 2,
                    "content": "Pointer stores an address in memory.",
                    "metadata": {"page_number": 14},
                    "file_name": "lecture-4.pdf",
                    "source_uri": "s3://course/lecture-4.pdf",
                }
            ]

    monkeypatch.setattr(vector_repo, "get_db_pool", lambda: _FakePool(_Conn()))
    rows = asyncio.run(vector_repo.search_vectors([0.1, 0.2], limit=1))

    assert rows[0]["chunk_id"] == "c1"
    assert rows[0]["document_id"] == "d1"
    assert rows[0]["source_uri"] == "s3://course/lecture-4.pdf"
    assert rows[0]["chunk_index"] == 2
    assert rows[0]["page_number"] == 14
    assert rows[0]["file_name"] == "lecture-4.pdf"
    assert rows[0]["snippet"].startswith("Pointer stores")


def test_search_vectors_uses_deterministic_tiebreak_order(monkeypatch):
    captured = {"sql": ""}

    class _Conn:
        async def fetch(self, sql, *_args, **_kwargs):
            captured["sql"] = sql
            return []

    monkeypatch.setattr(vector_repo, "get_db_pool", lambda: _FakePool(_Conn()))
    asyncio.run(vector_repo.search_vectors([0.1, 0.2], limit=1))
    assert "ORDER BY c.embedding <=> $1, c.document_id, c.chunk_index, c.id" in captured["sql"]


def test_search_chunks_keyword_ilike_orders_by_keyword_score(monkeypatch):
    captured = {"sql": "", "args": []}

    class _Conn:
        async def fetch(self, sql, *args, **_kwargs):
            captured["sql"] = sql
            captured["args"] = args
            return []

    monkeypatch.setattr(vector_repo, "get_db_pool", lambda: _FakePool(_Conn()))
    asyncio.run(
        vector_repo.search_chunks_keyword_ilike(
            ["buộc thôi học"],
            "REGULATION",
            limit=12,
        )
    )
    assert "keyword_score" in captured["sql"]
    assert "(buộc & thôi & học)" in captured["args"][1]


def test_search_vectors_filters_by_document_type(monkeypatch):
    captured = {"sql": "", "args": None}

    class _Conn:
        async def fetch(self, sql, *args, **_kwargs):
            captured["sql"] = sql
            captured["args"] = args
            return []

    monkeypatch.setattr(vector_repo, "get_db_pool", lambda: _FakePool(_Conn()))
    asyncio.run(vector_repo.search_vectors([0.1, 0.2], limit=5, document_type="REGULATION"))
    assert "AND d.document_type = $3" in captured["sql"]
    assert captured["args"][-1] == "REGULATION"


def test_query_course_materials_emits_structured_citations(monkeypatch):
    monkeypatch.setattr(tools, "check_semantic_cache", lambda *_args: None)
    monkeypatch.setattr(tools, "set_semantic_cache", lambda *_args, **_kwargs: None)
    async def _fake_search_vectors(*_args, **_kwargs):
        return [
            {
                "chunk_id": "chunk-a",
                "document_id": "doc-1",
                "chunk_index": 0,
                "content": "A pointer stores a memory address.",
                "file_name": "week4.pdf",
                "source_uri": "course/week4.pdf",
                "metadata": {"page": 12},
                "page_number": 12,
                "snippet": "A pointer stores a memory address.",
            }
        ]

    monkeypatch.setattr(tools, "search_vectors", _fake_search_vectors)

    class _Embeddings:
        async def create(self, **_kwargs):
            return type("EmbRes", (), {"data": [type("EmbRow", (), {"embedding": [0.1, 0.2]})()]})()

    class _Completions:
        async def create(self, **_kwargs):
            content = '{"matches":[{"chunk_index":1,"quote":"A pointer stores a memory address."}]}'
            msg = type("Msg", (), {"content": content})()
            return type("R", (), {"choices": [type("C", (), {"message": msg})()]})()

    class _Client:
        def __init__(self, *_args, **_kwargs):
            self.embeddings = _Embeddings()
            self.chat = type("Chat", (), {"completions": _Completions()})()

    monkeypatch.setattr(tools, "AsyncOpenAI", _Client)
    monkeypatch.setattr(tools, "OPENAI_API_KEY", "test-key")

    async def _run():
        result = await tools.query_course_materials("what is pointer?")
        citations = tools.consume_last_rag_citations()
        return result, citations

    result, citations = asyncio.run(_run())

    assert "week4.pdf" in result
    assert len(citations) == 1
    assert citations[0]["source_file"] == "week4.pdf"
    assert citations[0]["page_number"] == 12
    assert citations[0]["document_id"] == "doc-1"
    assert citations[0]["source_uri"] == "course/week4.pdf"
    assert citations[0]["chunk_id"] == "chunk-a"
    assert citations[0]["chunk_index"] == 0
    assert citations[0]["snippet"] == "A pointer stores a memory address."


def test_query_course_materials_supports_structured_facts(monkeypatch):
    monkeypatch.setattr(tools, "check_semantic_cache", lambda *_args: None)
    monkeypatch.setattr(tools, "set_semantic_cache", lambda *_args, **_kwargs: None)

    async def _fake_search_vectors(*_args, **_kwargs):
        return [
            {
                "chunk_id": "chunk-a",
                "document_id": "doc-1",
                "chunk_index": 0,
                "content": "Producer gui tin nhan den RabbitMQ.",
                "file_name": "rabbitmq.pdf",
                "source_uri": "course/rabbitmq.pdf",
                "metadata": {"page": 1},
                "page_number": 1,
                "snippet": "Producer gui tin nhan den RabbitMQ.",
            }
        ]

    monkeypatch.setattr(tools, "search_vectors", _fake_search_vectors)

    class _Embeddings:
        async def create(self, **_kwargs):
            return type("EmbRes", (), {"data": [type("EmbRow", (), {"embedding": [0.1, 0.2]})()]})()

    class _Completions:
        async def create(self, **_kwargs):
            content = '{"facts":[{"label":"Producer","description":"Ung dung gui tin nhan.","chunk_index":1,"quote":"Producer gui tin nhan den RabbitMQ."}]}'
            msg = type("Msg", (), {"content": content})()
            return type("R", (), {"choices": [type("C", (), {"message": msg})()]})()

    class _Client:
        def __init__(self, *_args, **_kwargs):
            self.embeddings = _Embeddings()
            self.chat = type("Chat", (), {"completions": _Completions()})()

    monkeypatch.setattr(tools, "AsyncOpenAI", _Client)
    monkeypatch.setattr(tools, "OPENAI_API_KEY", "test-key")

    result = asyncio.run(tools.query_course_materials("RabbitMQ co thanh phan nao?"))
    assert "**Producer**" in result
    assert "[rabbitmq.pdf, p.1]" in result


def test_query_course_materials_parses_fenced_json_facts(monkeypatch):
    monkeypatch.setattr(tools, "check_semantic_cache", lambda *_args: None)
    monkeypatch.setattr(tools, "set_semantic_cache", lambda *_args, **_kwargs: None)

    async def _fake_search_vectors(*_args, **_kwargs):
        return [
            {
                "chunk_id": "chunk-a",
                "document_id": "doc-1",
                "chunk_index": 0,
                "content": "Producer gui tin nhan den RabbitMQ.",
                "file_name": "rabbitmq.pdf",
                "source_uri": "course/rabbitmq.pdf",
                "metadata": {"page": 1},
                "page_number": 1,
                "snippet": "Producer gui tin nhan den RabbitMQ.",
            }
        ]

    monkeypatch.setattr(tools, "search_vectors", _fake_search_vectors)

    class _Embeddings:
        async def create(self, **_kwargs):
            return type("EmbRes", (), {"data": [type("EmbRow", (), {"embedding": [0.1, 0.2]})()]})()

    class _Completions:
        async def create(self, **_kwargs):
            content = "```json\n{\"facts\":[{\"label\":\"Producer\",\"description\":\"Ung dung gui tin nhan.\",\"chunk_index\":1,\"quote\":\"Producer gui tin nhan den RabbitMQ.\"}]}\n```"
            msg = type("Msg", (), {"content": content})()
            return type("R", (), {"choices": [type("C", (), {"message": msg})()]})()

    class _Client:
        def __init__(self, *_args, **_kwargs):
            self.embeddings = _Embeddings()
            self.chat = type("Chat", (), {"completions": _Completions()})()

    monkeypatch.setattr(tools, "AsyncOpenAI", _Client)
    monkeypatch.setattr(tools, "OPENAI_API_KEY", "test-key")

    result = asyncio.run(tools.query_course_materials("RabbitMQ co thanh phan nao?"))
    assert "**Producer**" in result


def test_query_course_materials_parses_preamble_and_fenced_json(monkeypatch):
    monkeypatch.setattr(tools, "check_semantic_cache", lambda *_args: None)
    monkeypatch.setattr(tools, "set_semantic_cache", lambda *_args, **_kwargs: None)

    async def _fake_search_vectors(*_args, **_kwargs):
        return [
            {
                "chunk_id": "chunk-a",
                "document_id": "doc-1",
                "chunk_index": 0,
                "content": "Producer gui tin nhan den RabbitMQ.",
                "file_name": "rabbitmq.pdf",
                "source_uri": "course/rabbitmq.pdf",
                "metadata": {"page": 1},
                "page_number": 1,
                "snippet": "Producer gui tin nhan den RabbitMQ.",
            }
        ]

    monkeypatch.setattr(tools, "search_vectors", _fake_search_vectors)

    class _Embeddings:
        async def create(self, **_kwargs):
            return type("EmbRes", (), {"data": [type("EmbRow", (), {"embedding": [0.1, 0.2]})()]})()

    class _Completions:
        async def create(self, **_kwargs):
            content = (
                "Phan tich nhu sau:\n```json\n"
                '{"facts":[{"label":"Producer","description":"Ung dung gui tin nhan.",'
                '"chunk_index":1,"quote":"Producer gui tin nhan den RabbitMQ."}]}\n'
                "```\nCam on."
            )
            msg = type("Msg", (), {"content": content})()
            return type("R", (), {"choices": [type("C", (), {"message": msg})()]})()

    class _Client:
        def __init__(self, *_args, **_kwargs):
            self.embeddings = _Embeddings()
            self.chat = type("Chat", (), {"completions": _Completions()})()

    monkeypatch.setattr(tools, "AsyncOpenAI", _Client)
    monkeypatch.setattr(tools, "OPENAI_API_KEY", "test-key")

    result = asyncio.run(tools.query_course_materials("RabbitMQ co thanh phan nao?"))
    assert "**Producer**" in result


def test_prepare_rerank_json_string_extracts_balanced_object():
    raw = 'Some text before {"facts": []} and after'
    assert tools._prepare_rerank_json_string(raw) == '{"facts": []}'


def test_query_course_materials_json_decode_fallback_to_text(monkeypatch):
    monkeypatch.setattr(tools, "check_semantic_cache", lambda *_args: None)
    monkeypatch.setattr(tools, "set_semantic_cache", lambda *_args, **_kwargs: None)

    async def _fake_search_vectors(*_args, **_kwargs):
        return [
            {
                "chunk_id": "chunk-a",
                "document_id": "doc-1",
                "chunk_index": 0,
                "content": "Producer gui tin nhan den RabbitMQ.",
                "file_name": "rabbitmq.pdf",
                "source_uri": "course/rabbitmq.pdf",
                "metadata": {"page": 1},
                "page_number": 1,
                "snippet": "Producer gui tin nhan den RabbitMQ.",
            }
        ]

    monkeypatch.setattr(tools, "search_vectors", _fake_search_vectors)

    class _Embeddings:
        async def create(self, **_kwargs):
            return type("EmbRes", (), {"data": [type("EmbRow", (), {"embedding": [0.1, 0.2]})()]})()

    class _Completions:
        async def create(self, **_kwargs):
            # malformed JSON; fallback should still extract quote text
            content = '```json\n{"facts":[{"chunk_index":1,"quote":"Producer gui tin nhan den RabbitMQ."}\n```'
            msg = type("Msg", (), {"content": content})()
            return type("R", (), {"choices": [type("C", (), {"message": msg})()]})()

    class _Client:
        def __init__(self, *_args, **_kwargs):
            self.embeddings = _Embeddings()
            self.chat = type("Chat", (), {"completions": _Completions()})()

    monkeypatch.setattr(tools, "AsyncOpenAI", _Client)
    monkeypatch.setattr(tools, "OPENAI_API_KEY", "test-key")

    result = asyncio.run(tools.query_course_materials("RabbitMQ co thanh phan nao?"))
    assert "Producer gui tin nhan den RabbitMQ." in result


def test_query_course_materials_cache_hit_preserves_citations(monkeypatch):
    monkeypatch.setattr(
        tools,
        "check_semantic_cache",
        lambda *_args: {
            "answer": '[week4.pdf - p.12] "A pointer stores a memory address."',
            "citations": [
                {
                    "source_file": "week4.pdf",
                    "page_number": 12,
                    "document_id": "doc-1",
                    "source_uri": "course/week4.pdf",
                    "chunk_id": "chunk-a",
                    "chunk_index": 0,
                    "snippet": "A pointer stores a memory address.",
                }
            ],
        },
    )

    class _Embeddings:
        async def create(self, **_kwargs):
            return type("EmbRes", (), {"data": [type("EmbRow", (), {"embedding": [0.1, 0.2]})()]})()

    class _Client:
        def __init__(self, *_args, **_kwargs):
            self.embeddings = _Embeddings()
            self.chat = type("Chat", (), {"completions": object()})()

    monkeypatch.setattr(tools, "AsyncOpenAI", _Client)
    monkeypatch.setattr(tools, "OPENAI_API_KEY", "test-key")

    async def _run():
        result = await tools.query_course_materials("what is pointer?")
        citations = tools.consume_last_rag_citations()
        return result, citations

    result, citations = asyncio.run(_run())
    assert "week4.pdf" in result
    assert len(citations) == 1
    assert citations[0]["source_file"] == "week4.pdf"


def test_query_course_materials_cache_stale_falls_back_to_vector_search(monkeypatch):
    monkeypatch.setattr(
        tools,
        "check_semantic_cache",
        lambda *_args: {"answer": "stale cached answer", "citations": []},
    )
    monkeypatch.setattr(tools, "set_semantic_cache", lambda *_args, **_kwargs: None)
    fallback_called = {"value": False}

    async def _fake_search_vectors(*_args, **_kwargs):
        fallback_called["value"] = True
        return [
            {
                "chunk_id": "chunk-a",
                "document_id": "doc-1",
                "chunk_index": 0,
                "content": "A pointer stores a memory address.",
                "file_name": "week4.pdf",
                "source_uri": "course/week4.pdf",
                "metadata": {"page": 12},
                "page_number": 12,
                "snippet": "A pointer stores a memory address.",
            }
        ]

    monkeypatch.setattr(tools, "search_vectors", _fake_search_vectors)

    class _Embeddings:
        async def create(self, **_kwargs):
            return type("EmbRes", (), {"data": [type("EmbRow", (), {"embedding": [0.1, 0.2]})()]})()

    class _Completions:
        async def create(self, **_kwargs):
            content = '{"matches":[{"chunk_index":1,"quote":"A pointer stores a memory address."}]}'
            msg = type("Msg", (), {"content": content})()
            return type("R", (), {"choices": [type("C", (), {"message": msg})()]})()

    class _Client:
        def __init__(self, *_args, **_kwargs):
            self.embeddings = _Embeddings()
            self.chat = type("Chat", (), {"completions": _Completions()})()

    monkeypatch.setattr(tools, "AsyncOpenAI", _Client)
    monkeypatch.setattr(tools, "OPENAI_API_KEY", "test-key")

    async def _run():
        result = await tools.query_course_materials("what is pointer?")
        citations = tools.consume_last_rag_citations()
        return result, citations

    result, citations = asyncio.run(_run())
    assert fallback_called["value"] is True
    assert "week4.pdf" in result
    assert len(citations) == 1


def test_grpc_final_metadata_includes_citations(monkeypatch):
    if grpc_server.ai_service_pb2 is None:
        pytest.skip("gRPC protobufs are not generated")

    class _Classification:
        def __init__(self):
            self.intent = grpc_server.IntentType.ACADEMIC
            self.is_violation = False
            self.violation_reason = ""
            self.confidence = 0.9
            self.reasoning = "academic"

    async def _fake_classify(*_args, **_kwargs):
        return _Classification()

    async def _fake_agent_loop(*_args, **_kwargs):
        yield {"type": "TOKEN", "content": "hello"}
        yield {
            "type": "CITATIONS",
            "content": [
                {
                    "source_file": "week4.pdf",
                    "page_number": 12,
                    "document_id": "doc-1",
                    "source_uri": "course/week4.pdf",
                    "chunk_id": "chunk-a",
                    "chunk_index": 0,
                    "snippet": "A pointer stores a memory address.",
                }
            ],
        }
        yield {"type": "DONE", "content": ""}

    class _Ctx:
        def cancelled(self):
            return False

        async def abort(self, *_args):
            raise AssertionError("abort should not be called")

    req = type(
        "Req",
        (),
        {
            "thread_id": "t1",
            "thread_title": "title",
            "current_message": "Explain pointers",
            "history": [],
            "tags": [],
        },
    )()

    monkeypatch.setattr(grpc_server, "classify_and_guard", _fake_classify)
    monkeypatch.setattr(grpc_server, "run_agent_loop_stream", _fake_agent_loop)
    servicer = grpc_server.AIThreadServicer(agent_client=object())
    async def _collect():
        return [chunk async for chunk in servicer.StreamAIResponse(req, _Ctx())]

    chunks = asyncio.run(_collect())

    assert chunks[-1].is_finished is True
    assert len(chunks[-1].metadata.citations) == 1
    assert chunks[-1].metadata.citations[0].source_file == "week4.pdf"
    assert chunks[-1].metadata.citations[0].document_id == "doc-1"
    assert chunks[-1].metadata.citations[0].source_uri == "course/week4.pdf"
    assert chunks[-1].metadata.citations[0].chunk_id == "chunk-a"
    assert chunks[-1].metadata.citations[0].chunk_index == 0
    assert chunks[-1].metadata.citations[0].snippet == "A pointer stores a memory address."


def test_filter_retrieved_chunks_applies_distance_and_doc_hint():
    chunks = [
        {
            "original_filename": "Saga pattern.pdf",
            "source_uri": "uploads/saga-pattern.pdf",
            "distance": 0.21,
        },
        {
            "original_filename": "Other doc.pdf",
            "source_uri": "uploads/other-doc.pdf",
            "distance": 0.22,
        },
        {
            "original_filename": "Saga pattern.pdf",
            "source_uri": "uploads/saga-pattern.pdf",
            "distance": 0.91,
        },
    ]
    filtered = tools._filter_retrieved_chunks("Cho minh thong tin saga pattern", chunks)
    assert len(filtered) == 1
    assert filtered[0]["original_filename"] == "Saga pattern.pdf"


def test_filter_with_diagnostics_uses_distance_fallback_when_hints_miss():
    chunks = [
        {
            "original_filename": "Week4 pointers.pdf",
            "source_uri": "uploads/week4-pointers.pdf",
            "distance": 0.20,
        },
        {
            "original_filename": "Memory basics.pdf",
            "source_uri": "uploads/memory-basics.pdf",
            "distance": 0.29,
        },
    ]
    filtered, stats = tools._filter_with_diagnostics("Cho minh noi dung saga", chunks)
    assert len(filtered) == 2
    assert stats["hint_fallback_used"] is True
    assert stats["distance_count"] == 2
    assert stats["final_count"] == 2


def test_filter_with_diagnostics_handles_empty_input():
    filtered, stats = tools._filter_with_diagnostics("any query", [])
    assert filtered == []
    assert stats["raw_count"] == 0
    assert stats["final_count"] == 0


def test_grpc_escalation_streams_plain_text(monkeypatch):
    if grpc_server.ai_service_pb2 is None:
        pytest.skip("gRPC protobufs are not generated")

    class _Classification:
        def __init__(self):
            self.intent = grpc_server.IntentType.ACADEMIC
            self.is_violation = False
            self.violation_reason = ""
            self.confidence = 0.9
            self.reasoning = "academic"

    async def _fake_classify(*_args, **_kwargs):
        return _Classification()

    async def _fake_agent_loop(*_args, **_kwargs):
        yield {"type": "ESCALATION", "content": "Da chuyen cau hoi den TA."}
        yield {"type": "DONE", "content": ""}

    class _Ctx:
        def cancelled(self):
            return False

        async def abort(self, *_args):
            raise AssertionError("abort should not be called")

    req = type(
        "Req",
        (),
        {
            "thread_id": "t1",
            "thread_title": "title",
            "current_message": "Explain pointers",
            "history": [],
            "tags": [],
        },
    )()

    monkeypatch.setattr(grpc_server, "classify_and_guard", _fake_classify)
    monkeypatch.setattr(grpc_server, "run_agent_loop_stream", _fake_agent_loop)
    servicer = grpc_server.AIThreadServicer(agent_client=object())

    async def _collect():
        return [chunk async for chunk in servicer.StreamAIResponse(req, _Ctx())]

    chunks = asyncio.run(_collect())
    assert chunks[0].chunk == "Da chuyen cau hoi den TA."
    assert chunks[0].is_finished is False


def test_agent_academic_prefetch_uses_raw_user_query(monkeypatch):
    calls = []

    async def _fake_execute_tool(name, args):
        calls.append((name, args))
        if name == "query_course_materials":
            return "RAG result"
        return ""

    monkeypatch.setattr(agent, "execute_tool", _fake_execute_tool)
    monkeypatch.setattr(agent, "consume_last_rag_citations", lambda: [])

    class _Delta:
        def __init__(self, content=""):
            self.content = content
            self.tool_calls = None

    class _Choice:
        def __init__(self, content, finish_reason):
            self.delta = _Delta(content)
            self.finish_reason = finish_reason

    class _Chunk:
        def __init__(self, content, finish_reason):
            self.choices = [_Choice(content, finish_reason)]

    class _Stream:
        def __init__(self):
            self._yielded = False

        def __aiter__(self):
            return self

        async def __anext__(self):
            if self._yielded:
                raise StopAsyncIteration
            self._yielded = True
            return _Chunk("[CONFIDENCE: 95] Xin chao", "stop")

    class _Completions:
        async def create(self, **_kwargs):
            return _Stream()

    class _FakeClient:
        def __init__(self):
            self.chat = type("Chat", (), {"completions": _Completions()})()

    async def _collect():
        out = []
        async for chunk in agent.run_agent_loop_stream(
            _FakeClient(),
            user_input="rabbitmq co thanh phan nao",
            intent="ACADEMIC",
            intent_confidence=0.9,
        ):
            out.append(chunk)
        return out

    chunks = asyncio.run(_collect())
    assert calls[0][0] == "query_course_materials"
    assert calls[0][1]["query"] == "rabbitmq co thanh phan nao"
    assert any(c.get("type") == "STATUS" and "academic-prefetch" in c.get("content", "") for c in chunks)


def test_agent_procedural_regulation_prefetch_uses_query_regulations(monkeypatch):
    calls = []

    async def _fake_execute_tool(name, args):
        calls.append((name, args))
        if name == "query_regulations":
            return "REGULATION result"
        return ""

    monkeypatch.setattr(agent, "execute_tool", _fake_execute_tool)
    monkeypatch.setattr(agent, "consume_last_rag_citations", lambda: [])

    class _Delta:
        def __init__(self, content=""):
            self.content = content
            self.tool_calls = None

    class _Choice:
        def __init__(self, content, finish_reason):
            self.delta = _Delta(content)
            self.finish_reason = finish_reason

    class _Chunk:
        def __init__(self, content, finish_reason):
            self.choices = [_Choice(content, finish_reason)]

    class _Stream:
        def __init__(self):
            self._yielded = False

        def __aiter__(self):
            return self

        async def __anext__(self):
            if self._yielded:
                raise StopAsyncIteration
            self._yielded = True
            return _Chunk("[CONFIDENCE: 95] Da ro", "stop")

    class _Completions:
        async def create(self, **_kwargs):
            return _Stream()

    class _FakeClient:
        def __init__(self):
            self.chat = type("Chat", (), {"completions": _Completions()})()

    async def _collect():
        out = []
        async for chunk in agent.run_agent_loop_stream(
            _FakeClient(),
            user_input="Buộc thôi học khi nào",
            intent="PROCEDURAL",
            intent_confidence=0.9,
        ):
            out.append(chunk)
        return out

    chunks = asyncio.run(_collect())
    assert calls[0][0] == "query_regulations"
    assert calls[0][1]["query"] == "Buộc thôi học khi nào"
    assert any(c.get("type") == "STATUS" and "procedural-prefetch" in c.get("content", "") for c in chunks)


def test_regulation_extractive_fallback_has_stable_student_format():
    answer, citations = tools._build_extractive_answer_from_chunks(
        [
            {
                "document_id": "doc-1",
                "original_filename": "Quyche.pdf",
                "source_uri": "uploads/quyche.pdf",
                "page_number": 4,
                "snippet": "Sinh viên bị buộc thôi học nếu vi phạm điều kiện học vụ hai kỳ liên tiếp.",
                "chunk_id": "chunk-1",
                "chunk_index": 0,
            }
        ],
        max_items=3,
    )
    assert "Dựa trên các điều khoản liên quan trong quy chế" in answer
    assert "rerank đang chậm" not in answer
    assert "[Quyche.pdf, p.4]" in answer
    assert len(citations) == 1


def test_dedupe_regulation_chunks_removes_near_duplicates():
    deduped = tools._dedupe_regulation_chunks(
        [
            {"document_id": "doc-1", "page_number": 2, "snippet": "Canh cao hoc vu lien tiep", "chunk_id": "a"},
            {"document_id": "doc-1", "page_number": 2, "snippet": "Canh cao   hoc vu lien tiep", "chunk_id": "b"},
            {"document_id": "doc-1", "page_number": 3, "snippet": "Dieu kien buoc thoi hoc", "chunk_id": "c"},
        ],
        max_items=8,
    )
    assert len(deduped) == 2


def test_adaptive_fallback_uses_raw_candidates_when_min_distance_plausible(monkeypatch):
    monkeypatch.setattr(tools, "RAG_FALLBACK_MAX_DISTANCE", 0.65)
    raw_chunks = [
        {"document_id": "doc-redis", "distance": 0.53, "snippet": "Redis khong nen la database chinh"},
        {"document_id": "doc-cache", "distance": 0.56, "snippet": "Redis duoc dung de cache"},
    ]
    stats = {"min_distance": 0.53}
    filtered, out_stats = tools._apply_adaptive_distance_fallback([], raw_chunks, stats)
    assert len(filtered) == 2
    assert out_stats["adaptive_fallback_used"] is True


def test_adaptive_fallback_rejects_candidates_when_too_far(monkeypatch):
    monkeypatch.setattr(tools, "RAG_FALLBACK_MAX_DISTANCE", 0.65)
    raw_chunks = [
        {"document_id": "doc-a", "distance": 0.81},
        {"document_id": "doc-b", "distance": 0.84},
    ]
    stats = {"min_distance": 0.81}
    filtered, out_stats = tools._apply_adaptive_distance_fallback([], raw_chunks, stats)
    assert filtered == []
    assert out_stats["adaptive_fallback_used"] is False


def test_build_candidate_diagnostics_handles_missing_metadata():
    diagnostics = tools._build_candidate_diagnostics(
        [{"distance": 0.53, "content": "line1\nline2", "page_number": None}],
        top_k=5,
    )
    assert len(diagnostics) == 1
    assert diagnostics[0]["file"] == ""
    assert diagnostics[0]["page_number"] == 0
    assert diagnostics[0]["snippet"] == "line1 line2"


def test_extract_vi_query_anchors_maps_duoi_hoc():
    anchors = tools._extract_vi_query_anchors("Trường có đuổi học trong trường hợp nào?")
    assert "buộc thôi học" in anchors


def test_extract_vi_query_anchors_thi_ho_adds_discipline_terms():
    anchors = tools._extract_vi_query_anchors("Nhờ bạn thi hộ thì bị sao?")
    assert "thi hộ" in anchors
    assert "buộc thôi học" in anchors
    assert "kỉ luật" in anchors


def test_regulation_query_is_scope_applicability():
    assert tools._regulation_query_is_scope_applicability("Quy chế này áp dụng cho sinh viên nào?")
    assert not tools._regulation_query_is_scope_applicability("Mấy tín chỉ thì bị cảnh báo?")


def test_merge_regulation_vector_and_keyword_orders_and_dedupes():
    vec = [{"chunk_id": "v1", "distance": 0.2}, {"chunk_id": "v2", "distance": 0.3}]
    kw = [
        {"chunk_id": "k1", "keyword_score": 2},
        {"chunk_id": "v1", "distance": None, "keyword_score": 1},
    ]
    merged = tools._merge_regulation_vector_and_keyword(vec, kw, max_total=10)
    assert [c["chunk_id"] for c in merged] == ["v1", "k1", "v2"]


def test_merge_regulation_vector_and_keyword_applies_rrf():
    vec = [{"chunk_id": f"v{i}"} for i in range(10)]
    kw = [{"chunk_id": f"k{i}", "keyword_score": 10 - i} for i in range(6)]
    merged = tools._merge_regulation_vector_and_keyword(vec, kw, max_total=5)
    assert len(merged) == 5
    chunk_ids = {c["chunk_id"] for c in merged}
    assert "v0" in chunk_ids
    assert "k0" in chunk_ids


def test_regulation_keyword_terms_excludes_noisy_short_tokens():
    q = "Sinh viên K21 bị buộc thôi học trong trường hợp nào?"
    terms = tools._regulation_keyword_search_terms(q, tools._extract_vi_query_anchors(q))
    for t in terms:
        tl = t.lower().strip()
        assert tl not in {"học", "viên", "năm", "cho", "sinh"}


def test_regulation_synthesis_empty_answer_uses_extractive_fallback(monkeypatch):
    monkeypatch.setattr(tools, "check_semantic_cache", lambda *_args: None)
    monkeypatch.setattr(tools, "set_semantic_cache", lambda *_args, **_kwargs: None)

    async def _fake_search_vectors(*_args, **_kwargs):
        return [
            {
                "chunk_id": "c-reg",
                "document_id": "doc-r",
                "chunk_index": 0,
                "content": "Sinh viên bị buộc thôi học nếu vi phạm nghiêm trọng quy chế thi.",
                "file_name": "quyche.pdf",
                "source_uri": "uploads/quyche.pdf",
                "metadata": {"page": 4},
                "page_number": 4,
                "distance": 0.4,
                "snippet": "Sinh viên bị buộc thôi học nếu vi phạm nghiêm trọng quy chế thi.",
            }
        ]

    async def _fake_keyword_search(*_args, **_kwargs):
        return []

    async def _fake_neighbors(*_args, **_kwargs):
        return []

    monkeypatch.setattr(tools, "search_vectors", _fake_search_vectors)
    monkeypatch.setattr(tools, "search_chunks_keyword_ilike", _fake_keyword_search)
    monkeypatch.setattr(tools, "fetch_regulation_neighbor_chunks", _fake_neighbors)

    class _Embeddings:
        async def create(self, **_kwargs):
            return type("EmbRes", (), {"data": [type("EmbRow", (), {"embedding": [0.1, 0.2]})()]})()

    class _Completions:
        async def create(self, **_kwargs):
            content = '{"answer":"","cited_chunk_indices":[]}'
            msg = type("Msg", (), {"content": content})()
            return type("R", (), {"choices": [type("C", (), {"message": msg})()]})()

    class _Client:
        def __init__(self, *_args, **_kwargs):
            self.embeddings = _Embeddings()
            self.chat = type("Chat", (), {"completions": _Completions()})()

    monkeypatch.setattr(tools, "AsyncOpenAI", _Client)
    monkeypatch.setattr(tools, "OPENAI_API_KEY", "test-key")

    async def _run():
        return await tools._execute_rag_pipeline(
            "Buộc thôi học khi nào?",
            document_type=tools.DOCUMENT_TYPE_REGULATION,
        )

    answer, citations = asyncio.run(_run())
    assert "Dựa trên các điều khoản" in answer
    assert "quyche.pdf" in answer
    assert len(citations) >= 1
    assert citations[0].get("chunk_id") == "c-reg"


def test_split_regulation_facts_splits_multiline_bullets():
    facts = [
        {
            "label": "Điều kiện",
            "chunk_index": 1,
            "quote": "- Một điều kiện đầu tiên.\n- Hai.\n- Ba.",
            "description": "",
        }
    ]
    out = tools._split_regulation_facts(facts)
    assert len(out) == 3
    assert "Hai" in out[1]["quote"]


def test_normalize_course_rag_query_rabbitmq_synonyms_same_canonical():
    c1, m1 = tools.normalize_course_rag_query("rabbitmq có những thành phần chính nào")
    c2, m2 = tools.normalize_course_rag_query("rabbitmq có những thành phần cốt lõi nào")
    assert c1 == c2 == "rabbitmq các thành phần cốt lõi"
    assert m1.get("used_canonical") is True
    assert m2.get("used_canonical") is True


def test_normalize_course_rag_query_unrelated_unchanged():
    c, m = tools.normalize_course_rag_query("con trỏ là gì")
    assert m.get("used_canonical") is False
    assert c == "con trỏ là gì"


def test_extract_document_hints_includes_rabbitmq_topic():
    hints = tools._extract_document_hints("RabbitMQ hoạt động thế nào?")
    assert "rabbitmq" in hints


def test_equivalent_rabbitmq_component_queries_share_embedding_input(monkeypatch):
    """Regression: 'thành phần chính' vs 'cốt lõi' must use the same canonical query for embedding."""
    monkeypatch.setattr(tools, "check_semantic_cache", lambda *_args: None)
    monkeypatch.setattr(tools, "set_semantic_cache", lambda *_args, **_kwargs: None)

    async def _fake_kw(*_a, **_k):
        return []

    monkeypatch.setattr(tools, "search_chunks_keyword_ilike", _fake_kw)

    async def _fake_search_vectors(*_args, **_kwargs):
        return [
            {
                "chunk_id": "chunk-rmq",
                "document_id": "doc-rmq",
                "chunk_index": 0,
                "content": "Các thành phần cốt lõi: Producer, Exchange, Queue.",
                "file_name": "RabbitMQ.pdf",
                "original_filename": "RabbitMQ.pdf",
                "source_uri": "course/RabbitMQ.pdf",
                "metadata": {"page": 1},
                "page_number": 1,
                "snippet": "Các thành phần cốt lõi",
            }
        ]

    monkeypatch.setattr(tools, "search_vectors", _fake_search_vectors)

    embedding_inputs: list[object] = []

    class _Embeddings:
        async def create(self, **kwargs):
            inp = kwargs.get("input")
            if isinstance(inp, list) and inp:
                embedding_inputs.append(inp[0])
            else:
                embedding_inputs.append(inp)
            return type("EmbRes", (), {"data": [type("EmbRow", (), {"embedding": [0.1] * 8})()]})()

    class _Completions:
        async def create(self, **_kwargs):
            content = (
                '{"facts":['
                '{"label":"Producer","description":"Gửi.","chunk_index":1,"quote":"Producer"},'
                '{"label":"Exchange","description":"Route.","chunk_index":1,"quote":"Exchange"},'
                '{"label":"Queue","description":"Lưu.","chunk_index":1,"quote":"Queue"}'
                "]}"
            )
            msg = type("Msg", (), {"content": content})()
            return type("R", (), {"choices": [type("C", (), {"message": msg})()]})()

    class _Client:
        def __init__(self, *_args, **_kwargs):
            self.embeddings = _Embeddings()
            self.chat = type("Chat", (), {"completions": _Completions()})()

    monkeypatch.setattr(tools, "AsyncOpenAI", _Client)
    monkeypatch.setattr(tools, "OPENAI_API_KEY", "test-key")

    async def _run_both():
        await tools.query_course_materials("rabbitmq có những thành phần chính nào")
        await tools.query_course_materials("rabbitmq có những thành phần cốt lõi nào")

    asyncio.run(_run_both())
    assert len(embedding_inputs) == 2
    assert embedding_inputs[0] == embedding_inputs[1]
    assert embedding_inputs[0] == "rabbitmq các thành phần cốt lõi"


def test_course_keyword_merge_prefers_topic_chunks(monkeypatch):
    """Hybrid retrieval prepends keyword hits so RabbitMQ.pdf stays ahead of unrelated PDFs."""
    monkeypatch.setattr(tools, "check_semantic_cache", lambda *_args: None)
    monkeypatch.setattr(tools, "set_semantic_cache", lambda *_args, **_kwargs: None)

    vec_called = {"n": 0}

    async def _fake_search_vectors(*_args, **_kwargs):
        vec_called["n"] += 1
        return [
            {
                "chunk_id": "chunk-redis",
                "document_id": "doc-redis",
                "chunk_index": 0,
                "content": "Redis vs RabbitMQ so sánh.",
                "file_name": "redis.pdf",
                "original_filename": "Redis.pdf",
                "source_uri": "course/redis.pdf",
                "metadata": {"page": 4},
                "page_number": 4,
                "distance": 0.55,
                "snippet": "Redis vs RabbitMQ",
            },
        ]

    async def _fake_kw(_terms, _doc_type, limit=15):
        return [
            {
                "chunk_id": "chunk-rmq",
                "document_id": "doc-rmq",
                "chunk_index": 0,
                "content": "RabbitMQ các thành phần cốt lõi: Producer.",
                "file_name": "rabbitmq.pdf",
                "original_filename": "RabbitMQ.pdf",
                "source_uri": "course/rabbitmq.pdf",
                "metadata": {"page": 1},
                "page_number": 1,
                "keyword_score": 3,
                "snippet": "các thành phần cốt lõi",
            },
        ]

    monkeypatch.setattr(tools, "search_vectors", _fake_search_vectors)
    monkeypatch.setattr(tools, "search_chunks_keyword_ilike", _fake_kw)

    class _Embeddings:
        async def create(self, **_kwargs):
            return type("EmbRes", (), {"data": [type("EmbRow", (), {"embedding": [0.1] * 8})()]})()

    class _Completions:
        async def create(self, **_kwargs):
            content = (
                '{"facts":['
                '{"label":"Producer","description":"Gửi message.","chunk_index":1,"quote":"Producer"}]}'
            )
            msg = type("Msg", (), {"content": content})()
            return type("R", (), {"choices": [type("C", (), {"message": msg})()]})()

    class _Client:
        def __init__(self, *_args, **_kwargs):
            self.embeddings = _Embeddings()
            self.chat = type("Chat", (), {"completions": _Completions()})()

    monkeypatch.setattr(tools, "AsyncOpenAI", _Client)
    monkeypatch.setattr(tools, "OPENAI_API_KEY", "test-key")

    async def _run():
        return await tools.query_course_materials("rabbitmq có những thành phần cốt lõi nào")

    result = asyncio.run(_run())
    assert "RabbitMQ.pdf" in result
    assert "Producer" in result

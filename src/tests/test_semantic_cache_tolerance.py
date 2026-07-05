"""
Unit tests: semantic-cache failure tolerance in the RAG pipeline.

The Redis client is synchronous and may be slow/unreachable. Cache reads and
writes must degrade to a miss / no-op without failing the answer, and the
REGULATION path must never touch the cache.
"""

import asyncio
import json

from src import tools


class _Embeddings:
    async def create(self, **_kwargs):
        vec = type("D", (), {"embedding": [0.1] * 8})()
        return type("R", (), {"data": [vec]})()


class _Completions:
    def __init__(self, content):
        self._content = content

    async def create(self, **_kwargs):
        msg = type("Msg", (), {"content": self._content})()
        return type("R", (), {"choices": [type("C", (), {"message": msg})()]})()


def _client_returning(content):
    class _Client:
        def __init__(self, *_a, **_k):
            self.embeddings = _Embeddings()
            self.chat = type("Chat", (), {"completions": _Completions(content)})()

    return _Client


_CHUNK = {
    "chunk_id": "c1",
    "document_id": "d1",
    "chunk_index": 0,
    "content": "RabbitMQ delivers messages.",
    "snippet": "RabbitMQ delivers messages.",
    "file_name": "week4.pdf",
    "original_filename": "week4.pdf",
    "source_uri": "course/week4.pdf",
    "page_number": 3,
    "distance": 0.2,
    "metadata": {},
}


def test_cache_check_crash_degrades_to_miss(monkeypatch):
    def _exploding_cache(*_args, **_kwargs):
        raise ConnectionError("redis down")

    monkeypatch.setattr(tools, "check_semantic_cache", _exploding_cache)
    monkeypatch.setattr(tools, "set_semantic_cache", lambda *a, **k: None)

    async def _fake_search(*_a, **_k):
        return [dict(_CHUNK)]

    monkeypatch.setattr(tools, "search_vectors", _fake_search)
    monkeypatch.setattr(tools, "AsyncOpenAI", _client_returning(
        json.dumps({"facts": [{"label": "", "description": "RabbitMQ delivers messages.",
                               "chunk_index": 1, "quote": "RabbitMQ delivers messages."}]})
    ))
    monkeypatch.setattr(tools, "OPENAI_API_KEY", "test-key")

    async def _run():
        result = await tools.query_course_materials("rabbitmq?")
        return result, tools.consume_last_rag_citations()

    result, citations = asyncio.run(_run())
    # Pipeline survived the cache crash and produced a grounded answer
    assert "week4.pdf" in result
    assert len(citations) == 1


def test_cache_write_crash_does_not_fail_answer(monkeypatch):
    monkeypatch.setattr(tools, "check_semantic_cache", lambda *_a: None)

    def _exploding_set(*_args, **_kwargs):
        raise ConnectionError("redis down")

    monkeypatch.setattr(tools, "set_semantic_cache", _exploding_set)

    async def _fake_search(*_a, **_k):
        return [dict(_CHUNK)]

    monkeypatch.setattr(tools, "search_vectors", _fake_search)
    monkeypatch.setattr(tools, "AsyncOpenAI", _client_returning(
        json.dumps({"facts": [{"label": "", "description": "RabbitMQ delivers messages.",
                               "chunk_index": 1, "quote": "RabbitMQ delivers messages."}]})
    ))
    monkeypatch.setattr(tools, "OPENAI_API_KEY", "test-key")

    result = asyncio.run(tools.query_course_materials("rabbitmq?"))
    assert "week4.pdf" in result


def test_regulation_path_never_touches_cache(monkeypatch):
    touched = {"check": False, "set": False}

    def _check(*_a, **_k):
        touched["check"] = True
        return None

    def _set(*_a, **_k):
        touched["set"] = True
        return True

    monkeypatch.setattr(tools, "check_semantic_cache", _check)
    monkeypatch.setattr(tools, "set_semantic_cache", _set)

    async def _fake_search(*_a, **_k):
        return [dict(_CHUNK)]

    async def _fake_kw(*_a, **_k):
        return []

    async def _fake_neighbors(*_a, **_k):
        return []

    monkeypatch.setattr(tools, "search_vectors", _fake_search)
    monkeypatch.setattr(tools, "search_chunks_keyword_ilike", _fake_kw)
    monkeypatch.setattr(tools, "fetch_regulation_neighbor_chunks", _fake_neighbors)
    monkeypatch.setattr(tools, "AsyncOpenAI", _client_returning(
        json.dumps({"answer": "Điều kiện buộc thôi học ...", "cited_chunk_indices": [1]})
    ))
    monkeypatch.setattr(tools, "OPENAI_API_KEY", "test-key")

    asyncio.run(tools.query_regulations("buộc thôi học khi nào?"))
    assert touched == {"check": False, "set": False}

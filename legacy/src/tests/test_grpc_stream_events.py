"""
gRPC servicer contract tests (fake agent generator, no network):

  * `system_error` events map to a terminal chunk (is_finished=True) — before
    the fix this event type was dropped and the stream ended without a final
    message (users saw an empty response on OpenAI 429s)
  * unexpected exceptions produce a sanitized terminal message (no str(e))
  * agent_used pipe-tag metadata format stays parseable by the Java gateway
  * citation snippets are truncated to 2000 chars before crossing to Java
"""

import asyncio
import base64
import hashlib
import hmac
import json
import uuid

import pytest

import src.grpc_server as grpc_server


pytestmark = pytest.mark.skipif(
    grpc_server.ai_service_pb2 is None, reason="gRPC protobufs are not generated"
)


class _Classification:
    def __init__(self, intent=None, confidence=0.9):
        self.intent = intent or grpc_server.IntentType.ACADEMIC
        self.is_violation = False
        self.violation_reason = ""
        self.confidence = confidence
        self.reasoning = "academic"


class _Ctx:
    def cancelled(self):
        return False

    async def abort(self, *_args):
        raise AssertionError("abort should not be called")


def _req(message="Explain pointers"):
    return type(
        "Req",
        (),
        {
            "thread_id": "t1",
            "thread_title": "title",
            "current_message": message,
            "history": [],
            "tags": [],
        },
    )()


def _patch_classify(monkeypatch):
    async def _fake_classify(*_args, **_kwargs):
        return _Classification()

    monkeypatch.setattr(grpc_server, "classify_and_guard", _fake_classify)


def _collect(servicer, req):
    async def _run():
        return [c async for c in servicer.StreamAIResponse(req, _Ctx())]

    return asyncio.run(_run())


def test_system_error_maps_to_terminal_chunk(monkeypatch):
    _patch_classify(monkeypatch)

    async def _agent(*_a, **_k):
        yield {"type": "system_error", "code": 429, "message": "Hệ thống đang quá tải."}

    monkeypatch.setattr(grpc_server, "run_agent_loop_stream", _agent)
    chunks = _collect(grpc_server.AIThreadServicer(agent_client=object()), _req())

    assert len(chunks) == 1
    assert chunks[0].is_finished is True
    assert "Hệ thống đang quá tải." in chunks[0].chunk


def test_unexpected_exception_sends_sanitized_terminal_chunk(monkeypatch):
    _patch_classify(monkeypatch)

    async def _agent(*_a, **_k):
        yield {"type": "TOKEN", "content": "một phần"}
        raise RuntimeError("secret internal detail abc123")

    monkeypatch.setattr(grpc_server, "run_agent_loop_stream", _agent)
    chunks = _collect(grpc_server.AIThreadServicer(agent_client=object()), _req())

    assert chunks[-1].is_finished is True
    assert "secret internal detail" not in chunks[-1].chunk
    assert "abc123" not in chunks[-1].chunk
    assert chunks[-1].chunk.startswith("Internal error:")


def test_done_metadata_keeps_pipe_tag_contract(monkeypatch):
    _patch_classify(monkeypatch)

    async def _agent(*_a, **_k):
        yield {"type": "METRICS", "content": {"cache_hit": True, "usage": {"input_tokens": 11, "output_tokens": 5}}}
        yield {"type": "DONE", "content": ""}

    monkeypatch.setattr(grpc_server, "run_agent_loop_stream", _agent)
    chunks = _collect(grpc_server.AIThreadServicer(agent_client=object()), _req())

    final = chunks[-1]
    assert final.is_finished is True
    agent_used = final.metadata.agent_used
    assert agent_used.startswith("QA_AGENT")
    assert "|cache_hit=true" in agent_used
    assert "|input_tokens=11" in agent_used
    assert "|output_tokens=5" in agent_used


def test_citation_snippets_truncated_before_java(monkeypatch):
    _patch_classify(monkeypatch)
    long_snippet = "x" * 5000

    async def _agent(*_a, **_k):
        yield {
            "type": "CITATIONS",
            "content": [{
                "source_file": "week4.pdf",
                "page_number": 3,
                "document_id": "d1",
                "source_uri": "u",
                "chunk_id": "c1",
                "chunk_index": 0,
                "snippet": long_snippet,
            }],
        }
        yield {"type": "DONE", "content": ""}

    monkeypatch.setattr(grpc_server, "run_agent_loop_stream", _agent)
    chunks = _collect(grpc_server.AIThreadServicer(agent_client=object()), _req())

    final = chunks[-1]
    assert len(final.metadata.citations) == 1
    assert len(final.metadata.citations[0].snippet) == 2000


def test_status_labels_stay_friendly():
    assert grpc_server._friendly_status("Searching: query_regulations") == "Tra cứu quy định & quy chế"
    assert grpc_server._friendly_status("Searching: query_course_materials (academic-prefetch)") == "Đang tra cứu tài liệu học tập"
    assert grpc_server._friendly_status("fallback:intent_low_confidence") == "Đang phân tích kỹ câu hỏi"
    assert grpc_server._friendly_status("something-unmapped") == "something-unmapped"


def test_background_task_registry_holds_reference():
    async def _run():
        started = asyncio.Event()
        release = asyncio.Event()

        async def _work():
            started.set()
            await release.wait()

        task = grpc_server._spawn_background_task(_work())
        await started.wait()
        assert task in grpc_server._BACKGROUND_TASKS  # strong reference held
        release.set()
        await task
        await asyncio.sleep(0)
        assert task not in grpc_server._BACKGROUND_TASKS  # cleaned up when done

    asyncio.run(_run())


def test_authenticated_user_tag_is_verified_before_use(monkeypatch):
    secret = "test-secret"
    monkeypatch.setattr(grpc_server, "INTERNAL_CALLBACK_TOKEN", secret)
    payload = {
        "user_id": str(uuid.uuid4()),
        "student_code": "SV260115",
        "role": "STUDENT",
    }
    payload_b64 = base64.urlsafe_b64encode(
        json.dumps(payload, separators=(",", ":")).encode("utf-8")
    ).decode("ascii").rstrip("=")
    signature = base64.urlsafe_b64encode(
        hmac.new(secret.encode(), payload_b64.encode("ascii"), hashlib.sha256).digest()
    ).decode("ascii").rstrip("=")
    tag = f"authenticated_user:v1:{payload_b64}.{signature}"

    assert grpc_server._authenticated_user_from_grpc_tags([tag]) == payload

    tampered = tag[:-1] + ("A" if tag[-1] != "A" else "B")
    assert grpc_server._authenticated_user_from_grpc_tags([tampered]) == {}

import pytest
import sys
import types
import asyncio

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

from src.agent import (
    _assignment_lookup_hint,
    _personal_grade_lookup_hint,
    _should_prefetch_regulations,
    run_agent_loop_stream,
)


def test_low_confidence_emits_fallback_status_and_never_raises():
    """
    Low intent confidence must emit the fallback STATUS event, then continue
    with the normal model flow. With a broken client (no .chat attribute) the
    stream must terminate with a clean terminal `system_error` event instead
    of raising out of the generator (which previously killed the gRPC stream).
    """

    class DummyClient:
        pass

    async def _run():
        chunks = []
        async for chunk in run_agent_loop_stream(
            client=DummyClient(),
            user_input="Deadline lab 1 la bao nhieu?",
            intent="UNCERTAIN",
            intent_confidence=0.2,
        ):
            chunks.append(chunk)
        return chunks

    chunks = asyncio.run(_run())

    assert any(c["type"] == "STATUS" and "intent_low_confidence" in c["content"] for c in chunks)
    assert chunks[-1]["type"] == "system_error"
    assert chunks[-1]["code"] == 500
    assert chunks[-1]["message"]  # user-facing text, no raw exception detail
    assert "AttributeError" not in chunks[-1]["message"]


def test_procedural_intent_injects_assignment_routing_prompt():
    class FakeDelta:
        def __init__(self, content=None, tool_calls=None):
            self.content = content
            self.tool_calls = tool_calls

    class FakeChoice:
        def __init__(self, delta=None, finish_reason=None):
            self.delta = delta
            self.finish_reason = finish_reason

    class FakeChunk:
        def __init__(self, choice):
            self.choices = [choice]

    class FakeCompletions:
        def __init__(self):
            self.last_messages = None

        async def create(self, **kwargs):
            self.last_messages = kwargs["messages"]

            async def _stream():
                yield FakeChunk(FakeChoice(FakeDelta("[CONFIDENCE: 95]Da ro."), None))
                yield FakeChunk(FakeChoice(FakeDelta(None, None), "stop"))

            return _stream()

    class FakeClient:
        def __init__(self):
            self.chat = type("FakeChat", (), {"completions": FakeCompletions()})()

    client = FakeClient()
    async def _run():
        chunks = []
        async for chunk in run_agent_loop_stream(
            client=client,
            user_input="Khi nao nop lab 1?",
            intent="PROCEDURAL",
            intent_confidence=0.9,
        ):
            chunks.append(chunk)
        return chunks

    chunks = asyncio.run(_run())

    system_prompt = client.chat.completions.last_messages[0]["content"]
    assert "PROCEDURAL matters" in system_prompt
    assert "Do not invent deadlines/logistics" in system_prompt
    assert "explicitly state" in system_prompt
    assert chunks[-1]["type"] == "DONE"


@pytest.mark.parametrize(
    "question",
    [
        "Nộp muộn bài CSS bị trừ điểm thế nào?",
        "Late submission penalty của assignment này là gì?",
        "Nếu nộp trễ hạn thì bị phạt bao nhiêu?",
    ],
)
def test_assignment_late_policy_does_not_trigger_regulation_prefetch(question):
    assert _should_prefetch_regulations(question) is False


def test_assignment_lookup_hint_normalizes_context_to_stable_lab_alias():
    history = [
        {"author_role": "assistant", "content": "Chưa tìm thấy quy định nộp muộn của lab4 trên LMS."},
    ]
    assert _assignment_lookup_hint("Lab4: Automation ấy", history) == "Lab4"


def test_assignment_content_question_does_not_force_database_lookup():
    assert _assignment_lookup_hint("Lab 4 yêu cầu viết API nào?", []) is None


def test_personal_grade_routes_to_private_tool_not_regulations():
    question = "Điểm Lab2 của tôi là bao nhiêu, được nhận xét như nào?"
    assert _personal_grade_lookup_hint(question) == "Lab2"
    assert _should_prefetch_regulations(question) is False


def test_general_grading_policy_still_routes_to_regulations():
    assert _personal_grade_lookup_hint("Quy định cách tính điểm thành phần là gì?") is None
    assert _should_prefetch_regulations("Quy định cách tính điểm thành phần là gì?") is True


def test_private_grade_prefetch_never_falls_through_to_rag(monkeypatch):
    calls = []

    async def fake_execute_tool(name, args, trusted_context=None):
        calls.append((name, args, trusted_context))
        return '{"status":"FOUND","grades":[{"total_score":9.4}]}'

    monkeypatch.setattr("src.agent.execute_tool", fake_execute_tool)

    class Delta:
        content = "[CONFIDENCE: 99]Bạn được 9.4 điểm."
        tool_calls = None

    class Choice:
        delta = Delta()
        finish_reason = "stop"

    class Chunk:
        choices = [Choice()]
        usage = None

    class Completions:
        async def create(self, **_kwargs):
            async def stream():
                yield Chunk()
            return stream()

    client = type("Client", (), {"chat": type("Chat", (), {"completions": Completions()})()})()

    async def run():
        return [item async for item in run_agent_loop_stream(
            client,
            "Điểm Lab2 của tôi là bao nhiêu?",
            intent="ACADEMIC",
            intent_confidence=0.95,
            trusted_user_context={"student_code": "SV260115", "role": "STUDENT"},
        )]

    chunks = asyncio.run(run())
    assert [call[0] for call in calls] == ["get_my_grade"]
    assert calls[0][2]["student_code"] == "SV260115"
    assert chunks[-1]["type"] == "DONE"

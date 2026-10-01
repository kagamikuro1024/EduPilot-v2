"""
Agent-loop reliability tests (deterministic, no network):

  * escalation must not drop the first content fragment after [CONFIDENCE]
  * provider errors translate into terminal `system_error` events
  * repeated identical tool calls are deduplicated within one request
  * tool results are appended in tool-call index order
  * happy path always terminates with METRICS then DONE
  * mid-stream cancellation (client disconnect) closes cleanly
"""

import asyncio
import json

import httpx
import openai
import pytest

from src import agent


class _Delta:
    def __init__(self, content=None, tool_calls=None):
        self.content = content
        self.tool_calls = tool_calls


class _Choice:
    def __init__(self, delta, finish_reason=None):
        self.delta = delta
        self.finish_reason = finish_reason


class _Chunk:
    def __init__(self, delta, finish_reason=None):
        self.choices = [_Choice(delta, finish_reason)]
        self.usage = None


class _TCFunc:
    def __init__(self, name=None, arguments=None):
        self.name = name
        self.arguments = arguments


class _TCDelta:
    def __init__(self, index, id=None, name=None, arguments=None):
        self.index = index
        self.id = id
        self.function = _TCFunc(name, arguments)


def _stream_of(chunks):
    async def _gen():
        for c in chunks:
            yield c

    return _gen()


class _ScriptedCompletions:
    """Return one scripted stream per create() call."""

    def __init__(self, scripts):
        self.scripts = list(scripts)
        self.calls = 0

    async def create(self, **_kwargs):
        script = self.scripts[min(self.calls, len(self.scripts) - 1)]
        self.calls += 1
        if isinstance(script, Exception):
            raise script
        return _stream_of(script)


def _client(scripts):
    class _C:
        def __init__(self):
            self.chat = type("Chat", (), {"completions": _ScriptedCompletions(scripts)})()

    return _C()


def _collect(client, **kwargs):
    async def _run():
        out = []
        async for chunk in agent.run_agent_loop_stream(client, **kwargs):
            out.append(chunk)
        return out

    return asyncio.run(_run())


def test_escalation_keeps_first_content_fragment():
    chunks = _collect(
        _client([[_Chunk(_Delta("[CONFIDENCE: 40] Mình chưa chắc, nhưng"), "stop")]]),
        user_input="RabbitMQ exchange là gì?",
        intent="CONVERSATIONAL",  # avoid RAG fallback path
        intent_confidence=0.9,
    )
    types = [c["type"] for c in chunks]
    assert "ESCALATION" in types
    tokens = "".join(c["content"] for c in chunks if c["type"] == "TOKEN")
    # The fragment streamed together with the tag must survive escalation
    assert "Mình chưa chắc, nhưng" in tokens
    assert types[-2:] == ["METRICS", "DONE"]


def test_rate_limit_yields_429_system_error_and_terminates(monkeypatch, tmp_path):
    monkeypatch.chdir(tmp_path)  # .ai-log audit file goes to a temp dir
    request = httpx.Request("POST", "https://api.test/v1/chat/completions")
    err = openai.RateLimitError(
        message="rate limited",
        response=httpx.Response(429, request=request),
        body=None,
    )
    chunks = _collect(
        _client([err]),
        user_input="hi",
        intent="CONVERSATIONAL",
        intent_confidence=0.9,
    )
    assert chunks[-1]["type"] == "system_error"
    assert chunks[-1]["code"] == 429


def test_api_connection_error_yields_503_system_error():
    err = openai.APIConnectionError(request=None)
    chunks = _collect(
        _client([err]),
        user_input="hi",
        intent="CONVERSATIONAL",
        intent_confidence=0.9,
    )
    assert chunks[-1]["type"] == "system_error"
    assert chunks[-1]["code"] == 503
    assert "APIConnectionError" not in chunks[-1]["message"]


def test_unexpected_error_yields_500_system_error():
    chunks = _collect(
        _client([ValueError("internal bug detail")]),
        user_input="hi",
        intent="CONVERSATIONAL",
        intent_confidence=0.9,
    )
    assert chunks[-1]["type"] == "system_error"
    assert chunks[-1]["code"] == 500
    assert "internal bug detail" not in chunks[-1]["message"]


def _tool_call_turn(name, args_json, call_id="call_1", index=0):
    return [
        _Chunk(_Delta(None, [_TCDelta(index, id=call_id, name=name, arguments=args_json)]), "tool_calls"),
    ]


def test_repeated_identical_tool_calls_are_deduped(monkeypatch):
    executions = []

    async def _fake_execute(name, args):
        executions.append((name, json.dumps(args, sort_keys=True)))
        return "TOOL RESULT"

    monkeypatch.setattr(agent, "execute_tool", _fake_execute)
    monkeypatch.setattr(agent, "consume_last_rag_citations", lambda: [])
    monkeypatch.setattr(agent, "consume_last_rag_runtime", lambda: {"cache_hit": False})

    scripts = [
        _tool_call_turn("get_assignments", '{"days_limit": 7}', "call_a"),
        _tool_call_turn("get_assignments", '{"days_limit": 7}', "call_b"),
        [_Chunk(_Delta("[CONFIDENCE: 95] Xong."), "stop")],
    ]
    chunks = _collect(
        _client(scripts),
        user_input="deadline?",
        intent="PROCEDURAL",
        intent_confidence=0.9,
    )
    # Two identical calls, but only one real execution
    assert len(executions) == 1
    assert chunks[-1]["type"] == "DONE"


def test_tool_results_follow_index_order(monkeypatch):
    order = []

    async def _fake_execute(name, args):
        order.append(name)
        return f"RESULT {name}"

    monkeypatch.setattr(agent, "execute_tool", _fake_execute)
    monkeypatch.setattr(agent, "consume_last_rag_citations", lambda: [])
    monkeypatch.setattr(agent, "consume_last_rag_runtime", lambda: {"cache_hit": False})

    # One turn containing two tool calls whose IDs sort opposite to their index
    two_calls = [
        _Chunk(_Delta(None, [
            _TCDelta(0, id="zzz", name="get_assignments", arguments="{}"),
        ])),
        _Chunk(_Delta(None, [
            _TCDelta(1, id="aaa", name="check_assignment_deadline", arguments='{"assignment_name": "Lab 1"}'),
        ]), "tool_calls"),
    ]
    scripts = [two_calls, [_Chunk(_Delta("[CONFIDENCE: 95] Xong."), "stop")]]
    _collect(
        _client(scripts),
        user_input="deadline?",
        intent="PROCEDURAL",
        intent_confidence=0.9,
    )
    # Index order (get_assignments first), NOT id-sorted order
    assert order == ["get_assignments", "check_assignment_deadline"]


def test_malformed_tool_arguments_do_not_crash_stream(monkeypatch):
    seen_args = []

    async def _fake_execute(name, args):
        seen_args.append(args)
        return "ok"

    monkeypatch.setattr(agent, "execute_tool", _fake_execute)
    monkeypatch.setattr(agent, "consume_last_rag_citations", lambda: [])
    monkeypatch.setattr(agent, "consume_last_rag_runtime", lambda: {"cache_hit": False})

    scripts = [
        _tool_call_turn("get_assignments", "{not valid json!!", "call_a"),
        [_Chunk(_Delta("[CONFIDENCE: 95] Xong."), "stop")],
    ]
    chunks = _collect(
        _client(scripts),
        user_input="deadline?",
        intent="PROCEDURAL",
        intent_confidence=0.9,
    )
    assert seen_args == [{}]  # malformed JSON degraded to empty args
    assert chunks[-1]["type"] == "DONE"


def test_happy_path_ends_with_metrics_then_done():
    chunks = _collect(
        _client([[_Chunk(_Delta("[CONFIDENCE: 95] Chào bạn!"), "stop")]]),
        user_input="xin chào",
        intent="CONVERSATIONAL",
        intent_confidence=0.95,
    )
    assert [c["type"] for c in chunks][-2:] == ["METRICS", "DONE"]
    metrics = [c for c in chunks if c["type"] == "METRICS"][0]["content"]
    assert set(metrics.keys()) == {"cache_hit", "usage"}
    assert set(metrics["usage"].keys()) == {"input_tokens", "output_tokens"}


def test_mid_stream_cancellation_closes_cleanly():
    async def _run():
        gen = agent.run_agent_loop_stream(
            _client([[
                _Chunk(_Delta("[CONFIDENCE: 95] phần một ")),
                _Chunk(_Delta("phần hai ")),
                _Chunk(_Delta("phần ba"), "stop"),
            ]]),
            user_input="xin chào",
            intent="CONVERSATIONAL",
            intent_confidence=0.95,
        )
        got_token = False
        async for chunk in gen:
            if chunk["type"] == "TOKEN":
                got_token = True
                break  # simulate client disconnect
        await gen.aclose()  # must not raise
        return got_token

    assert asyncio.run(_run()) is True


def test_empty_model_response_still_terminates_for_greeting():
    chunks = _collect(
        _client([[ _Chunk(_Delta(None), "stop") ]]),
        user_input="xin chào",
        intent="CONVERSATIONAL",
        intent_confidence=0.95,
    )
    assert [c["type"] for c in chunks][-2:] == ["METRICS", "DONE"]

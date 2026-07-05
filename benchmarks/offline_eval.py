"""
Offline AI-core evaluation harness (no network, no paid API calls).

Runs deterministic scenarios against the agent loop, tool layer, and gRPC
servicer using mocked OpenAI/vector/cache dependencies, and scores:

  * robustness  — do adversarial paths end in a clean terminal event?
  * content     — is streamed content preserved (escalation path)?
  * cost        — duplicate tool executions, OpenAI client instantiations
  * async       — event-loop blocking while the semantic cache is slow
  * routing     — regulation-prefetch rule accuracy on the ground-truth set
  * memory      — chat-path import RSS; Docling must stay unloaded

Usage:
    PYTHONPATH=<repo-root> python3 benchmarks/offline_eval.py --label after
"""

from __future__ import annotations

import argparse
import asyncio
import importlib
import json
import os
import resource
import sys
import time
from pathlib import Path


# --------------------------------------------------------------------------
# Fake OpenAI streaming plumbing
# --------------------------------------------------------------------------

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


async def _drain(gen):
    out = []
    try:
        async for chunk in gen:
            out.append(chunk)
    except Exception as exc:  # noqa: BLE001 - harness must record crashes
        out.append({"type": "__RAISED__", "content": f"{type(exc).__name__}: {exc}"})
    return out


def _terminal_ok(chunks) -> bool:
    """Stream ends via DONE or a terminal system_error — never via an exception."""
    if not chunks:
        return False
    if any(c.get("type") == "__RAISED__" for c in chunks):
        return False
    return chunks[-1].get("type") in ("DONE", "system_error")


# --------------------------------------------------------------------------
# Scenarios
# --------------------------------------------------------------------------

def _tool_call_turn(name, args_json, call_id="call_1"):
    return [_Chunk(_Delta(None, [_TCDelta(0, id=call_id, name=name, arguments=args_json)]), "tool_calls")]


_FINAL_TURN = [_Chunk(_Delta("[CONFIDENCE: 95] Hoàn tất."), "stop")]


async def scenario_extra_tool_arg(agent, tools):
    """Model passes an unknown extra argument to a real registry tool."""
    client = _client([
        _tool_call_turn("query_course_materials", '{"query": "saga", "hacker_extra": 1}'),
        _FINAL_TURN,
    ])
    chunks = await _drain(agent.run_agent_loop_stream(
        client, user_input="saga là gì", intent="UNCERTAIN", intent_confidence=0.9))
    return {"terminal_ok": _terminal_ok(chunks)}


async def scenario_missing_required_arg(agent, tools):
    client = _client([
        _tool_call_turn("query_course_materials", '{}'),
        _FINAL_TURN,
    ])
    chunks = await _drain(agent.run_agent_loop_stream(
        client, user_input="saga là gì", intent="UNCERTAIN", intent_confidence=0.9))
    return {"terminal_ok": _terminal_ok(chunks)}


async def scenario_unknown_tool(agent, tools):
    client = _client([
        _tool_call_turn("rm_rf_database", '{"query": "x"}'),
        _FINAL_TURN,
    ])
    chunks = await _drain(agent.run_agent_loop_stream(
        client, user_input="hi ha", intent="CONVERSATIONAL", intent_confidence=0.9))
    return {"terminal_ok": _terminal_ok(chunks)}


async def scenario_malformed_args_json(agent, tools):
    client = _client([
        _tool_call_turn("get_assignments", '{"days_limit": '),
        _FINAL_TURN,
    ])
    chunks = await _drain(agent.run_agent_loop_stream(
        client, user_input="deadline?", intent="PROCEDURAL", intent_confidence=0.9))
    return {"terminal_ok": _terminal_ok(chunks)}


async def scenario_api_connection_error(agent, tools):
    import openai
    client = _client([openai.APIConnectionError(request=None)])
    chunks = await _drain(agent.run_agent_loop_stream(
        client, user_input="hello", intent="CONVERSATIONAL", intent_confidence=0.9))
    return {"terminal_ok": _terminal_ok(chunks)}


async def scenario_unexpected_exception(agent, tools):
    client = _client([ValueError("boom")])
    chunks = await _drain(agent.run_agent_loop_stream(
        client, user_input="hello", intent="CONVERSATIONAL", intent_confidence=0.9))
    return {"terminal_ok": _terminal_ok(chunks)}


async def scenario_escalation_content(agent, tools):
    """All post-tag content must reach the student when escalation fires."""
    expected = "Đây là phần đầu quan trọng và đây là phần sau."
    client = _client([[
        _Chunk(_Delta("[CONFIDENCE: 40] Đây là phần đầu quan trọng ")),
        _Chunk(_Delta("và đây là phần sau."), "stop"),
    ]])
    chunks = await _drain(agent.run_agent_loop_stream(
        client, user_input="xin chào", intent="CONVERSATIONAL", intent_confidence=0.9))
    delivered = "".join(c.get("content", "") for c in chunks if c.get("type") == "TOKEN")
    ratio = len(delivered.strip()) / len(expected)
    return {
        "terminal_ok": _terminal_ok(chunks),
        "escalated": any(c.get("type") == "ESCALATION" for c in chunks),
        "content_delivered_ratio": round(min(ratio, 1.0), 3),
    }


async def scenario_duplicate_rag_calls(agent, tools):
    """Model repeats the identical RAG call; count real executions."""
    executions = {"n": 0}

    async def _fake_execute(name, args):
        executions["n"] += 1
        return "RAG RESULT"

    orig_exec = agent.execute_tool
    orig_cite = agent.consume_last_rag_citations
    orig_rt = agent.consume_last_rag_runtime
    agent.execute_tool = _fake_execute
    agent.consume_last_rag_citations = lambda: []
    agent.consume_last_rag_runtime = lambda: {"cache_hit": False}
    try:
        client = _client([
            _tool_call_turn("query_course_materials", '{"query": "saga"}', "call_a"),
            _tool_call_turn("query_course_materials", '{"query": "saga"}', "call_b"),
            _tool_call_turn("query_course_materials", '{"query": "saga"}', "call_c"),
            _FINAL_TURN,
        ])
        chunks = await _drain(agent.run_agent_loop_stream(
            client, user_input="saga?", intent="UNCERTAIN", intent_confidence=0.9))
    finally:
        agent.execute_tool = orig_exec
        agent.consume_last_rag_citations = orig_cite
        agent.consume_last_rag_runtime = orig_rt
    return {"terminal_ok": _terminal_ok(chunks), "tool_executions_for_3_identical_calls": executions["n"]}


async def scenario_slow_cache_event_loop(agent, tools):
    """Max event-loop stall (ms) while the semantic cache takes 300 ms."""

    def _slow_cache(_vec):
        time.sleep(0.3)
        return None

    async def _fake_search(*_a, **_k):
        return []

    class _Emb:
        async def create(self, **_k):
            return type("R", (), {"data": [type("D", (), {"embedding": [0.1] * 8})()]})()

    class _FakeOpenAI:
        def __init__(self, *_a, **_k):
            self.embeddings = _Emb()
            self.chat = None

    saved = {}
    for name, repl in [
        ("check_semantic_cache", _slow_cache),
        ("set_semantic_cache", lambda *a, **k: None),
        ("search_vectors", _fake_search),
        ("AsyncOpenAI", _FakeOpenAI),
        ("OPENAI_API_KEY", "test-key"),
    ]:
        saved[name] = getattr(tools, name)
        setattr(tools, name, repl)

    max_gap = {"ms": 0.0}
    stop = asyncio.Event()

    async def _heartbeat():
        prev = time.perf_counter()
        while not stop.is_set():
            await asyncio.sleep(0.01)
            now = time.perf_counter()
            gap = (now - prev) * 1000 - 10
            max_gap["ms"] = max(max_gap["ms"], gap)
            prev = now

    try:
        hb = asyncio.create_task(_heartbeat())
        await tools.query_course_materials("saga pattern?")
        stop.set()
        await hb
    finally:
        for name, val in saved.items():
            setattr(tools, name, val)
    return {"max_event_loop_stall_ms": round(max_gap["ms"], 1)}


async def scenario_client_instantiations(agent, tools):
    """OpenAI client objects created across 5 sequential RAG calls."""
    counter = {"n": 0}

    class _Emb:
        async def create(self, **_k):
            return type("R", (), {"data": [type("D", (), {"embedding": [0.1] * 8})()]})()

    class _CountingOpenAI:
        def __init__(self, *_a, **_k):
            counter["n"] += 1
            self.embeddings = _Emb()
            self.chat = None

    async def _fake_search(*_a, **_k):
        return []

    saved = {}
    for name, repl in [
        ("check_semantic_cache", lambda *_a, **_k: None),
        ("set_semantic_cache", lambda *a, **k: None),
        ("search_vectors", _fake_search),
        ("AsyncOpenAI", _CountingOpenAI),
        ("OPENAI_API_KEY", "test-key"),
    ]:
        saved[name] = getattr(tools, name)
        setattr(tools, name, repl)
    try:
        for _ in range(5):
            await tools.query_course_materials("saga pattern?")
    finally:
        for name, val in saved.items():
            setattr(tools, name, val)
    return {"openai_clients_created_for_5_calls": counter["n"]}


async def scenario_grpc_rate_limit_visibility(agent, tools, grpc_server):
    """After an OpenAI 429, does the user receive a terminal message via gRPC?"""
    if grpc_server.ai_service_pb2 is None:
        return {"skipped": True}

    class _Classification:
        intent = grpc_server.IntentType.ACADEMIC
        is_violation = False
        violation_reason = ""
        confidence = 0.9
        reasoning = "academic"

    async def _fake_classify(*_a, **_k):
        return _Classification()

    async def _agent(*_a, **_k):
        yield {"type": "system_error", "code": 429,
               "message": "Hệ thống đang quá tải do lượng truy cập lớn."}

    class _Ctx:
        def cancelled(self):
            return False

        async def abort(self, *_args):
            raise AssertionError

    req = type("Req", (), {"thread_id": "t", "thread_title": "t", "current_message": "m",
                           "history": [], "tags": []})()

    saved_classify = grpc_server.classify_and_guard
    saved_loop = grpc_server.run_agent_loop_stream
    grpc_server.classify_and_guard = _fake_classify
    grpc_server.run_agent_loop_stream = _agent
    try:
        servicer = grpc_server.AIThreadServicer(agent_client=object())
        chunks = [c async for c in servicer.StreamAIResponse(req, _Ctx())]
    finally:
        grpc_server.classify_and_guard = saved_classify
        grpc_server.run_agent_loop_stream = saved_loop

    visible = bool(chunks) and bool(chunks[-1].chunk.strip()) and chunks[-1].is_finished
    return {"user_sees_terminal_message_on_429": visible}


async def scenario_citation_alignment(agent, tools):
    """Rerank pointing at nonexistent chunks must yield zero invented citations."""
    class _Emb:
        async def create(self, **_k):
            return type("R", (), {"data": [type("D", (), {"embedding": [0.1] * 8})()]})()

    class _Completions:
        async def create(self, **_k):
            content = json.dumps({"facts": [
                {"label": "", "description": "made up", "chunk_index": 99, "quote": "made up"},
            ]})
            msg = type("Msg", (), {"content": content})()
            return type("R", (), {"choices": [type("C", (), {"message": msg})()]})()

    class _FakeOpenAI:
        def __init__(self, *_a, **_k):
            self.embeddings = _Emb()
            self.chat = type("Chat", (), {"completions": _Completions()})()

    chunk = {"chunk_id": "c1", "document_id": "d1", "chunk_index": 0,
             "content": "real content", "snippet": "real content",
             "file_name": "w.pdf", "original_filename": "w.pdf",
             "source_uri": "u", "page_number": 1, "distance": 0.2, "metadata": {}}

    async def _fake_search(*_a, **_k):
        return [dict(chunk)]

    saved = {}
    for name, repl in [
        ("check_semantic_cache", lambda *_a, **_k: None),
        ("set_semantic_cache", lambda *a, **k: None),
        ("search_vectors", _fake_search),
        ("AsyncOpenAI", _FakeOpenAI),
        ("OPENAI_API_KEY", "test-key"),
    ]:
        saved[name] = getattr(tools, name)
        setattr(tools, name, repl)
    try:
        await tools.query_course_materials("saga?")
        citations = tools.consume_last_rag_citations()
    finally:
        for name, val in saved.items():
            setattr(tools, name, val)
    return {"invented_citations": len(citations)}


def routing_rule_accuracy(agent, dataset_path: Path):
    """Regulation-prefetch rule vs ground-truth document_type (PROCEDURAL cases)."""
    if not dataset_path.exists():
        return {"skipped": True}
    total = reg_total = reg_hit = sql_total = sql_false_prefetch = 0
    with dataset_path.open(encoding="utf-8-sig") as fh:
        for line in fh:
            line = line.strip()
            if not line:
                continue
            case = json.loads(line)
            if case.get("expected_intent") != "PROCEDURAL":
                continue
            total += 1
            prefetch = agent._should_prefetch_regulations(case["query"])
            if case.get("document_type") == "REGULATION":
                reg_total += 1
                reg_hit += int(prefetch)
            else:
                sql_total += 1
                sql_false_prefetch += int(prefetch)
    return {
        "procedural_cases": total,
        "regulation_prefetch_recall": round(reg_hit / reg_total, 3) if reg_total else None,
        "sql_case_false_prefetch_rate": round(sql_false_prefetch / sql_total, 3) if sql_total else None,
    }


def memory_and_lazy_loading():
    rss_mb = resource.getrusage(resource.RUSAGE_SELF).ru_maxrss / 1024
    return {
        "rss_after_chat_imports_mb": round(rss_mb, 1),
        "docling_loaded": "docling" in sys.modules,
        "torch_loaded": "torch" in sys.modules,
    }


async def agent_happy_path_latency(agent):
    """Mocked-stream overhead of one full agent run (framework cost only)."""
    runs = []
    for _ in range(30):
        client = _client([[
            _Chunk(_Delta("[CONFIDENCE: 95] Chào bạn, ")),
            _Chunk(_Delta("mình có thể giúp gì?"), "stop"),
        ]])
        t0 = time.perf_counter()
        async for _ in agent.run_agent_loop_stream(
                client, user_input="xin chào", intent="CONVERSATIONAL", intent_confidence=0.95):
            pass
        runs.append((time.perf_counter() - t0) * 1000)
    runs.sort()
    return {"agent_overhead_p50_ms": round(runs[len(runs) // 2], 2),
            "agent_overhead_p95_ms": round(runs[int(len(runs) * 0.95) - 1], 2)}


async def main_async(label: str, dataset: Path):
    agent = importlib.import_module("src.agent")
    tools = importlib.import_module("src.tools")
    grpc_server = importlib.import_module("src.grpc_server")

    results = {"label": label}
    scenarios = [
        ("extra_tool_arg", scenario_extra_tool_arg(agent, tools)),
        ("missing_required_arg", scenario_missing_required_arg(agent, tools)),
        ("unknown_tool", scenario_unknown_tool(agent, tools)),
        ("malformed_args_json", scenario_malformed_args_json(agent, tools)),
        ("api_connection_error", scenario_api_connection_error(agent, tools)),
        ("unexpected_exception", scenario_unexpected_exception(agent, tools)),
        ("escalation_content", scenario_escalation_content(agent, tools)),
        ("duplicate_rag_calls", scenario_duplicate_rag_calls(agent, tools)),
        ("slow_cache_event_loop", scenario_slow_cache_event_loop(agent, tools)),
        ("client_instantiations", scenario_client_instantiations(agent, tools)),
        ("grpc_429_visibility", scenario_grpc_rate_limit_visibility(agent, tools, grpc_server)),
        ("citation_alignment", scenario_citation_alignment(agent, tools)),
    ]
    for name, coro in scenarios:
        try:
            results[name] = await coro
        except Exception as exc:  # noqa: BLE001
            results[name] = {"harness_error": f"{type(exc).__name__}: {exc}"}

    results["routing_rules"] = routing_rule_accuracy(agent, dataset)
    results["latency"] = await agent_happy_path_latency(agent)
    results["memory"] = memory_and_lazy_loading()

    robustness = [v.get("terminal_ok") for v in results.values()
                  if isinstance(v, dict) and "terminal_ok" in v]
    results["summary"] = {
        "robustness_pass": sum(1 for x in robustness if x),
        "robustness_total": len(robustness),
    }
    return results


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--label", default="run")
    parser.add_argument("--dataset", default="data/benchmark_ground_truth.jsonl")
    parser.add_argument("--out", default="")
    args = parser.parse_args()

    os.environ.setdefault("OPENAI_API_KEY", "sk-offline-eval-dummy")
    os.environ.setdefault("DATABASE_URL", "postgresql://offline:offline@localhost:5432/offline")
    os.environ.setdefault("ADMIN_TOKEN", "offline-eval")

    results = asyncio.run(main_async(args.label, Path(args.dataset)))
    payload = json.dumps(results, ensure_ascii=False, indent=2)
    if args.out:
        Path(args.out).write_text(payload, encoding="utf-8")
    print(payload)


if __name__ == "__main__":
    main()

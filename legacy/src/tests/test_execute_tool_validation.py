"""
Unit tests: defensive tool-argument validation in ``execute_tool``.

Before hardening, model-generated arguments were splatted directly into the
tool coroutine (``**args``): an unknown key, wrong type, or missing required
parameter raised TypeError inside the agent loop and killed the whole stream.
"""

import asyncio

import pytest

from src import tools


def _run(coro):
    return asyncio.run(coro)


def test_unknown_tool_returns_error_string():
    result = _run(tools.execute_tool("no_such_tool", {"query": "x"}))
    assert result == "Tool 'no_such_tool' does not exist"


def test_extra_unknown_args_are_dropped(monkeypatch):
    captured = {}

    async def fake_tool(query: str):
        captured["query"] = query
        return "ok"

    monkeypatch.setitem(
        tools.TOOLS,
        "fake_tool",
        {"fn": fake_tool, "description": "d", "parameters": {"query": "string"}, "required": ["query"]},
    )
    result = _run(tools.execute_tool("fake_tool", {"query": "hi", "injected": "boom", "another": 1}))
    assert result == "ok"
    assert captured == {"query": "hi"}


def test_missing_required_arg_returns_validation_message():
    result = _run(tools.execute_tool("query_course_materials", {}))
    assert "Thiếu tham số bắt buộc" in result
    assert "query" in result


def test_non_dict_args_treated_as_empty():
    result = _run(tools.execute_tool("query_course_materials", "not-a-dict"))
    assert "Thiếu tham số bắt buộc" in result


def test_integer_param_accepts_numeric_string(monkeypatch):
    captured = {}

    async def fake_get(days_limit: int = None):
        captured["days_limit"] = days_limit
        return "[]"

    monkeypatch.setitem(
        tools.TOOLS,
        "fake_get",
        {"fn": fake_get, "description": "d", "parameters": {"days_limit": "integer"}, "required": []},
    )
    _run(tools.execute_tool("fake_get", {"days_limit": "7"}))
    assert captured["days_limit"] == 7


def test_integer_param_drops_garbage_value(monkeypatch):
    captured = {"called": False}

    async def fake_get(days_limit: int = None):
        captured["called"] = True
        captured["days_limit"] = days_limit
        return "[]"

    monkeypatch.setitem(
        tools.TOOLS,
        "fake_get",
        {"fn": fake_get, "description": "d", "parameters": {"days_limit": "integer"}, "required": []},
    )
    _run(tools.execute_tool("fake_get", {"days_limit": "next week"}))
    assert captured["called"] is True
    assert captured["days_limit"] is None  # optional param falls back to default


def test_string_param_coerces_non_string(monkeypatch):
    captured = {}

    async def fake_tool(query: str):
        captured["query"] = query
        return "ok"

    monkeypatch.setitem(
        tools.TOOLS,
        "fake_tool",
        {"fn": fake_tool, "description": "d", "parameters": {"query": "string"}, "required": ["query"]},
    )
    _run(tools.execute_tool("fake_tool", {"query": 42}))
    assert captured["query"] == "42"


def test_tool_crash_returns_error_string_instead_of_raising(monkeypatch):
    async def exploding_tool(query: str):
        raise RuntimeError("boom")

    monkeypatch.setitem(
        tools.TOOLS,
        "exploding",
        {"fn": exploding_tool, "description": "d", "parameters": {"query": "string"}, "required": ["query"]},
    )
    result = _run(tools.execute_tool("exploding", {"query": "x"}))
    assert result == tools.TOOL_EXECUTION_ERROR_MSG


def test_registry_schemas_are_openai_compatible():
    schemas = tools.get_tool_schemas()
    assert schemas, "registry must expose at least one tool"
    for schema in schemas:
        assert schema["type"] == "function"
        fn = schema["function"]
        assert fn["name"] in tools.TOOLS
        params = fn["parameters"]
        assert params["type"] == "object"
        for req in params["required"]:
            assert req in params["properties"], f"required {req} missing from properties in {fn['name']}"

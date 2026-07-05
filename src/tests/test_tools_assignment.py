import asyncio
import sys
import types

import pytest

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

from src import tools


class FakeCursor:
    def __init__(self, rows):
        self.rows = rows
        self.idx = 0

    def execute(self, _query, _params):
        return None

    def fetchone(self):
        if self.idx >= len(self.rows):
            return None
        value = self.rows[self.idx]
        self.idx += 1
        return value

    def fetchall(self):
        return self.rows

    def __enter__(self):
        return self

    def __exit__(self, _exc_type, _exc, _tb):
        return False


class FakeConn:
    def __init__(self, rows):
        self._cursor = FakeCursor(rows)

    def cursor(self):
        return self._cursor

    def close(self):
        return None

    def __enter__(self):
        return self

    def __exit__(self, _exc_type, _exc, _tb):
        return False


@pytest.mark.parametrize(
    ("raw", "expected"),
    [
        ("lab4", "lab4"),
        ("Lab 4", "lab4"),
        ("Lab-4: Automation", "lab4automation"),
    ],
)
def test_compact_assignment_lookup_ignores_formatting(raw, expected):
    assert tools._compact_assignment_lookup(raw) == expected


def test_assignment_known_deadline_and_penalty(monkeypatch):
    def fake_connect(_url):
        return FakeConn([
            ("aid-1", "Lab 1", "2026-05-01", "Minus 10% per day"),
        ])

    fake_psycopg2 = types.ModuleType("psycopg2")
    fake_psycopg2.connect = fake_connect
    monkeypatch.setattr(tools, "psycopg2", fake_psycopg2)
    result = tools._run_assignment_query("Lab 1", None)
    assert "Lab 1" in result
    assert "Minus 10% per day" in result
    assert "Vui long doi chieu voi LMS de dam bao chinh xac tuyet doi." in result


def test_assignment_unknown_returns_safe_response(monkeypatch):
    def fake_connect(_url):
        return FakeConn([None])

    fake_psycopg2 = types.ModuleType("psycopg2")
    fake_psycopg2.connect = fake_connect
    monkeypatch.setattr(tools, "psycopg2", fake_psycopg2)
    result = tools._run_assignment_query("Unknown Lab", None)
    assert "Khong tim thay thong tin bai tap" in result
    assert "LMS" in result


def test_assignment_includes_submission_status(monkeypatch):
    def fake_connect(_url):
        return FakeConn([
            ("aid-1", "Lab 2", "2026-05-10", "No late submission"),
            ("SUBMITTED", 9.0, "2026-04-27 21:00:00"),
        ])

    fake_psycopg2 = types.ModuleType("psycopg2")
    fake_psycopg2.connect = fake_connect
    monkeypatch.setattr(tools, "psycopg2", fake_psycopg2)
    result = tools._run_assignment_query("Lab 2", "22123456")
    assert "MSSV 22123456" in result
    assert "SUBMITTED" in result


def test_assignment_db_error_fallback(monkeypatch):
    async def boom(*_args, **_kwargs):
        raise RuntimeError("db unavailable")

    monkeypatch.setattr(asyncio, "to_thread", boom)
    result = asyncio.run(tools.check_assignment_deadline("Lab 3"))
    assert "Loi he thong khi truy van thong tin bai tap" in result
    assert "LMS" in result


def test_assignment_list_includes_late_penalty_rule(monkeypatch):
    def fake_connect(_url):
        return FakeConn([
            ("aid-1", "Lab CSS", "2026-07-10 16:59:00", "Cứ muộn 1 ngày bị trừ 10% điểm"),
        ])

    fake_psycopg2 = types.ModuleType("psycopg2")
    fake_psycopg2.connect = fake_connect
    monkeypatch.setattr(tools, "psycopg2", fake_psycopg2)

    result = tools._run_get_assignments()

    assert '"late_penalty_rule": "Cứ muộn 1 ngày bị trừ 10% điểm"' in result

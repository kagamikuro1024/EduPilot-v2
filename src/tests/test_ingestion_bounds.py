"""
Document-ingestion resource-bound tests:

  * DOCUMENT_INGESTION_CONCURRENCY semaphore actually serializes work
  * the Java status callback fires even when ingestion crashes
  * chat-only import path must NOT pull Docling/PyTorch into memory
    (Hugging Face CPU Basic protection)
"""

import asyncio
import sys

import src.document_callback as document_callback


def test_ingestion_semaphore_bounds_concurrency(monkeypatch):
    running = {"now": 0, "max": 0}
    posted = []

    class _FakeWorkerModule:
        @staticmethod
        async def process_document_task(**_kwargs):
            running["now"] += 1
            running["max"] = max(running["max"], running["now"])
            await asyncio.sleep(0.02)
            running["now"] -= 1
            return {"status": "READY", "chunks_persisted": 1, "reason": None}

    monkeypatch.setitem(
        sys.modules,
        "data_pipeline.workers.ingestion_worker",
        _FakeWorkerModule,
    )

    async def _fake_post(document_id, status, reason):
        posted.append((document_id, status))

    monkeypatch.setattr(document_callback, "_post_callback", _fake_post)

    async def _run():
        await asyncio.gather(*[
            document_callback.run_ingestion_with_callback(f"doc-{i}", f"file-{i}")
            for i in range(4)
        ])

    asyncio.run(_run())
    # Default DOCUMENT_INGESTION_CONCURRENCY=1 → strictly serialized
    assert running["max"] == document_callback._INGESTION_CONCURRENCY
    assert len(posted) == 4
    assert all(status == "READY" for _d, status in posted)


def test_callback_fires_even_when_ingestion_crashes(monkeypatch):
    posted = []

    class _ExplodingWorker:
        @staticmethod
        async def process_document_task(**_kwargs):
            raise RuntimeError("parser exploded")

    monkeypatch.setitem(sys.modules, "data_pipeline.workers.ingestion_worker", _ExplodingWorker)

    async def _fake_post(document_id, status, reason):
        posted.append((document_id, status, reason))

    monkeypatch.setattr(document_callback, "_post_callback", _fake_post)

    asyncio.run(document_callback.run_ingestion_with_callback("doc-x", "file-x"))
    assert posted == [("doc-x", "FAILED", "parser exploded")]


def test_chat_only_imports_do_not_load_docling():
    """Lazy-Docling protection for 16GB Spaces: importing the chat path
    (agent + tools + grpc server + callback module) must not import docling."""
    import src.agent  # noqa: F401
    import src.tools  # noqa: F401
    import src.grpc_server  # noqa: F401
    import src.document_callback  # noqa: F401

    assert "docling" not in sys.modules
    assert "torch" not in sys.modules
    assert "data_pipeline.workers.ingestion_worker" not in sys.modules or hasattr(
        sys.modules["data_pipeline.workers.ingestion_worker"], "process_document_task"
    )

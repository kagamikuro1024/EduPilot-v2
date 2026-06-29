import asyncio
import logging
import os

import httpx

from .config import INTERNAL_CALLBACK_TOKEN, JAVA_CALLBACK_URL

logger = logging.getLogger(__name__)

try:
    _INGESTION_CONCURRENCY = max(1, int(os.getenv("DOCUMENT_INGESTION_CONCURRENCY", "1")))
except ValueError:
    _INGESTION_CONCURRENCY = 1
_INGESTION_SEMAPHORE = asyncio.Semaphore(_INGESTION_CONCURRENCY)


async def run_ingestion_with_callback(document_id: str, file_url: str) -> None:
    status = "FAILED"
    reason = "Unknown ingestion result."

    try:
        async with _INGESTION_SEMAPHORE:
            # Lazy import keeps Docling/PyTorch out of chat-only process startup.
            from data_pipeline.workers.ingestion_worker import process_document_task

            result = await process_document_task(
                java_document_id=document_id,
                file_path=file_url,
                source_uri=file_url,
                metadata={"java_document_id": document_id},
            )
        status_map = {
            "READY": ("READY", None),
            "DUPLICATE": ("DUPLICATE", "Duplicate content_hash already exists in another document."),
            "NOT_FOUND": ("FAILED", "Java document row not found."),
            "EMPTY": ("FAILED", "No chunks generated from the document."),
            "FAILED": ("FAILED", result.get("reason") or "Ingestion pipeline failed."),
        }
        status, reason = status_map.get(result["status"], ("FAILED", "Unknown ingestion result."))
    except Exception as exc:
        status = "FAILED"
        reason = str(exc)
    finally:
        await _post_callback(document_id, status, reason)


async def _post_callback(document_id: str, status: str, reason: str | None) -> None:
    url = f"{JAVA_CALLBACK_URL}/api/v1/internal/documents/{document_id}/status"
    headers = {"Content-Type": "application/json"}
    if INTERNAL_CALLBACK_TOKEN:
        headers["X-Internal-Token"] = INTERNAL_CALLBACK_TOKEN

    payload = {"status": status, "reason": reason}

    try:
        async with httpx.AsyncClient(timeout=10.0) as client:
            response = await client.patch(url, json=payload, headers=headers)
            response.raise_for_status()
            logger.info("Document status callback succeeded for %s with status=%s", document_id, status)
    except Exception as exc:
        logger.error("Document status callback failed for %s: %s", document_id, exc)

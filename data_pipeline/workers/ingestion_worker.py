"""
Module: data_pipeline.workers.ingestion_worker
Description: Worker that executes the Ingestion (ETL) flow as a background task.
             Orchestrates Parser -> Chunking -> Embedding -> Database modules.
             Integrates retry mechanisms (Tenacity) to ensure resilience for AI API calls.
"""

import os
import logging
import anyio
import json
from tenacity import retry, stop_after_attempt, wait_exponential, retry_if_exception_type

from src.database.connection import get_db_pool
from data_pipeline.pipeline.document_parser import compute_hash, DoclingParser, extract_text_file
from data_pipeline.pipeline.chunking import chunk_markdown
from data_pipeline.pipeline.cleaner import clean_markdown_text
from data_pipeline.pipeline.embedding import generate_embeddings
from src.database.vector_repo import (
    IngestionResult,
    attach_chunks_to_java_document,
    attach_structured_document_to_java_document,
    has_duplicate_content_hash,
)
from src.database.grade_repo import get_document_type, replace_grade_records
from data_pipeline.pipeline.grade_report import parse_grade_report

logger = logging.getLogger(__name__)

@retry(
    stop=stop_after_attempt(3), 
    wait=wait_exponential(multiplier=1, min=2, max=10),
    reraise=True
)
async def _retryable_process_core(markdown_clean: str, source_uri: str):
    """Core logic for chunking and embedding with retry mechanism."""
    # 3. Transform (Chunking)
    chunks = chunk_markdown(markdown_clean, source_uri)
    if not chunks:
        raise ValueError("No chunks generated from the document.")

    # 4. Transform (Embedding)
    texts_to_embed = [c['content'] for c in chunks]
    vector_embeddings = await generate_embeddings(texts_to_embed)
    
    # Assign generated vectors back to chunk objects
    for i, vector in enumerate(vector_embeddings):
        chunks[i]['embedding'] = vector
        
    return chunks

@retry(
    stop=stop_after_attempt(5),
    wait=wait_exponential(multiplier=1, min=1, max=10),
    retry=retry_if_exception_type(Exception)
)
async def _update_document_status(doc_id: str, status: str):
    """Updates document status in PostgreSQL using the global connection pool with retries."""
    try:
        pool = get_db_pool()
        async with pool.acquire() as conn:
            await conn.execute(
                "UPDATE documents SET status = $1, updated_at = NOW() WHERE id = $2",
                status, doc_id
            )
        logger.info(f"Document {doc_id} status updated to {status}")
    except Exception as e:
        logger.error(f"Failed to update document status in DB for {doc_id}: {e}")
        raise # Raise for tenacity to retry

def _log_btc_error(prompt: str, response: str):
    """Logs error in BTC format to session.jsonl."""
    log_entry = {
        "prompt": prompt,
        "response": response,
        "tool": "ingestion_worker",
        "status": "error"
    }
    # Note: Mandatory log command will handle the actual logging via hook, 
    # but here we also log to stdout/logger for visibility.
    logger.error(f"BTC LOG: {json.dumps(log_entry)}")

async def process_document_task(
    java_document_id: str,
    file_path: str,
    source_uri: str,
    metadata: dict,
) -> IngestionResult:
    """
    Orchestrates the complete Ingestion process with Resilience Guardrails:
    - Catches all failures (Parser errors, API downtime, network lag).
    - Performs database cleanup (Rollback) on failure to maintain data integrity.
    """
    try:
        logger.info(f"Starting Ingestion Worker for: {source_uri}")
        
        # Read raw file to compute content_hash for deduplication
        with open(file_path, "rb") as f:
            raw_content = f.read()
        content_hash = compute_hash(raw_content)

        if await has_duplicate_content_hash(content_hash, java_document_id):
            logger.info("Duplicate document detected before parsing for java_document_id=%s", java_document_id)
            return {"status": "DUPLICATE", "chunks_persisted": 0, "reason": "duplicate_content_hash"}

        document_type = await get_document_type(java_document_id)

        # 1. Extract & 2. Transform (Cleaner)
        extension = os.path.splitext(source_uri)[1].lower()
        if document_type == "GRADE_REPORT" and extension == ".pdf":
            # Grade tables need their raw text rows. Docling's low-memory/table
            # settings can replace tables with a literal "[Table Content]",
            # while pypdfium2 reliably preserves the PDF's text layer.
            content_to_clean = await anyio.to_thread.run_sync(
                DoclingParser._extract_pages_pypdfium,
                file_path,
            )
            markdown_clean = await anyio.to_thread.run_sync(clean_markdown_text, content_to_clean)
        elif extension == ".txt":
            # For TXT files, use lightweight plain text extractor
            content_to_clean = await anyio.to_thread.run_sync(extract_text_file, file_path)
            # TIP-006: Apply cleaning even for plain text to ensure NFC normalization & noise removal
            markdown_clean = await anyio.to_thread.run_sync(clean_markdown_text, content_to_clean)
        else:
            # For PDF/DOCX, use Docling (Singleton)
            parser = DoclingParser()
            content_to_clean = await anyio.to_thread.run_sync(parser.parse, file_path)
            markdown_clean = await anyio.to_thread.run_sync(clean_markdown_text, content_to_clean)

        grade_records = []
        if document_type == "GRADE_REPORT":
            grade_records = await anyio.to_thread.run_sync(
                parse_grade_report,
                markdown_clean,
                os.path.basename(source_uri),
            )
            if not grade_records:
                reason = "No grade rows with student codes were found in this grade report."
                logger.warning("Grade report parsing failed for %s: %s", source_uri, reason)
                await _update_document_status(java_document_id, 'FAILED')
                return {"status": "FAILED", "chunks_persisted": 0, "reason": reason}

            # Grade reports are private structured data. Do not generate
            # embeddings or persist class-wide text in the general RAG table.
            result = await attach_structured_document_to_java_document(
                java_document_id=java_document_id,
                filename=os.path.basename(source_uri),
                source_uri=source_uri,
                content_hash=content_hash,
                metadata=metadata,
            )
            if result["status"] == "READY":
                await replace_grade_records(java_document_id, grade_records)
                await _update_document_status(java_document_id, 'READY')
                logger.info(
                    "Grade report ingestion completed with %s private rows: %s",
                    len(grade_records),
                    source_uri,
                )
            else:
                await _update_document_status(java_document_id, 'FAILED')
            return result
        
        # 3 & 4. Transform (Chunking & Embedding) with Retry
        try:
            chunks = await _retryable_process_core(markdown_clean, source_uri)
        except Exception as e:
            logger.error(f"Core processing (chunking/embedding) failed after 3 retries: {e}")
            await _update_document_status(java_document_id, 'FAILED')
            _log_btc_error(f"Process document {java_document_id}", f"Failed after retries: {str(e)}")
            return {"status": "FAILED", "chunks_persisted": 0, "reason": f"Core processing failed: {str(e)}"}

        # 5. Load (Repository with Transaction support)
        result = await attach_chunks_to_java_document(
            java_document_id=java_document_id,
            filename=os.path.basename(source_uri),
            source_uri=source_uri,
            content_hash=content_hash,
            metadata=metadata,
            chunks_data=chunks,
        )

        if result["status"] == "READY":
            await _update_document_status(java_document_id, 'READY')
            logger.info(f"Ingestion completed successfully for: {source_uri}")
        else:
            await _update_document_status(java_document_id, 'FAILED')
            logger.warning("Ingestion finished with status=%s, reason=%s", result["status"], result["reason"])

        return result

    except Exception as exc:
        # Failure recovery: log error and remove incomplete records to keep schema clean
        logger.error(f"Ingestion process failed for {source_uri}: {exc}")
        await _update_document_status(java_document_id, 'FAILED')
        _log_btc_error(f"Ingestion process {java_document_id}", str(exc))
        return {"status": "FAILED", "chunks_persisted": 0, "reason": str(exc)}
    
    finally:
        # 1. Disk Cleanup: Always remove temporary files after processing
        if os.path.exists(file_path):
            try:
                os.remove(file_path)
                logger.info(f"Temporary file removed: {file_path}")
            except Exception as e:
                logger.error(f"Failed to remove temporary file {file_path}: {e}")

        # Do not perform recursive "core.*" cleanup here.
        # It can accidentally delete package source files like "core.py" under venv.

import logging
import json
import re
import datetime
from typing import List, Optional, Dict, Literal, TypedDict
import asyncpg
from tenacity import retry, stop_after_attempt, wait_exponential, retry_if_exception_type
from src.database.connection import get_db_pool

logger = logging.getLogger(__name__)

class IngestionResult(TypedDict):
    status: Literal["READY", "DUPLICATE", "NOT_FOUND", "EMPTY", "FAILED"]
    chunks_persisted: int
    reason: Optional[str]

@retry(
    stop=stop_after_attempt(5),
    wait=wait_exponential(multiplier=1, min=1, max=10),
    retry=retry_if_exception_type(Exception)
)
async def has_duplicate_content_hash(content_hash: str, java_document_id: str) -> bool:
    """Check whether another document already uses this content hash."""
    pool = get_db_pool()
    async with pool.acquire() as conn:
        existing_id = await conn.fetchval(
            """
            SELECT id
            FROM documents
            WHERE content_hash = $1
              AND id <> $2
            LIMIT 1
            """,
            content_hash,
            java_document_id,
        )
        return existing_id is not None


@retry(
    stop=stop_after_attempt(5),
    wait=wait_exponential(multiplier=1, min=1, max=10),
    retry=retry_if_exception_type(Exception)
)
async def attach_chunks_to_java_document(
    java_document_id: str,
    filename: str,
    source_uri: str,
    content_hash: str,
    metadata: dict | str | None,
    chunks_data: list[dict],
) -> IngestionResult:
    """Attach processed chunks to an existing Java-owned document row."""
    pool = get_db_pool()

    async with pool.acquire() as conn:
        async with conn.transaction():
            try:
                doc_metadata = metadata if metadata is not None else {}
                if isinstance(doc_metadata, dict):
                    doc_metadata = json.dumps(doc_metadata)

                update_result = await conn.execute(
                    """
                    UPDATE documents
                    SET filename = $2,
                        source_uri = $3,
                        content_hash = $4,
                        metadata = $5::jsonb,
                        updated_at = NOW()
                    WHERE id = $1
                    """,
                    java_document_id,
                    filename,
                    source_uri,
                    content_hash,
                    doc_metadata,
                )
            except asyncpg.UniqueViolationError:
                logger.warning("Duplicate content_hash detected while updating java document %s", java_document_id)
                return {"status": "DUPLICATE", "chunks_persisted": 0, "reason": "duplicate_content_hash"}
            except Exception as exc:
                logger.error("Failed to update java document %s: %s", java_document_id, exc)
                return {"status": "FAILED", "chunks_persisted": 0, "reason": str(exc)}

            if update_result.endswith(" 0"):
                logger.error("Java document row not found for id=%s", java_document_id)
                return {"status": "NOT_FOUND", "chunks_persisted": 0, "reason": "java_document_row_missing"}

            if not chunks_data:
                logger.warning("No chunks provided for java document: %s", java_document_id)
                return {"status": "EMPTY", "chunks_persisted": 0, "reason": "no_chunks_generated"}

            values = []
            for idx, chunk in enumerate(chunks_data):
                meta = chunk.get("metadata", {})
                if isinstance(meta, dict):
                    meta = json.dumps(meta)
                values.append(
                    (
                        java_document_id,
                        chunk["content"],
                        meta,
                        str(chunk["embedding"]),
                        idx,
                        chunk["content_hash"],
                    )
                )

            try:
                await conn.executemany(
                    """
                    INSERT INTO document_chunks (
                        document_id, content, metadata, embedding, chunk_index, content_hash
                    )
                    VALUES ($1, $2, $3::jsonb, $4, $5, $6)
                    ON CONFLICT (document_id, content_hash) DO NOTHING
                    """,
                    values,
                )
            except Exception as exc:
                logger.error("Failed to insert chunks for java document %s: %s", java_document_id, exc)
                return {"status": "FAILED", "chunks_persisted": 0, "reason": "insertion_failed"}

            logger.info("Successfully attached %s chunks to java document %s", len(values), java_document_id)
            return {"status": "READY", "chunks_persisted": len(values), "reason": None}

async def get_chunks_by_document_id(doc_id: str) -> list[dict]:
    """Retrieves all chunks associated with a specific document ID."""
    try:
        pool = get_db_pool()
        async with pool.acquire() as conn:
            rows = await conn.fetch(
                """
                SELECT id, content, metadata, chunk_index, content_hash
                FROM document_chunks
                WHERE document_id = $1
                ORDER BY chunk_index ASC
                """,
                doc_id
            )
            return [dict(row) for row in rows]
    except Exception as e:
        logger.error(f"Error fetching chunks for document {doc_id}: {e}")
        return []

async def update_chunk_content(chunk_id: str, new_content: str, new_embedding: list[float], updated_by: str = "TA") -> bool:
    """
    Updates the content and vector embedding for a specific chunk.
    """
    from data_pipeline.pipeline.document_parser import compute_hash
    
    try:
        pool = get_db_pool()
        new_hash = compute_hash(new_content.encode('utf-8') if isinstance(new_content, str) else new_content)
        
        async with pool.acquire() as conn:
            async with conn.transaction():
                # 1. Fetch existing metadata to append audit history
                row = await conn.fetchrow("SELECT metadata FROM document_chunks WHERE id = $1", chunk_id)
                if not row:
                    logger.warning(f"Chunk ID {chunk_id} not found for correction.")
                    return False
                
                meta = row['metadata']
                if isinstance(meta, str):
                    meta = json.loads(meta)
                
                # 2. Record Audit Trail for manual corrections
                audit_trail = meta.get('audit_trail', [])
                audit_trail.append({
                    "timestamp": datetime.datetime.now().isoformat(),
                    "user": updated_by,
                    "action": "content_correction"
                })
                meta['audit_trail'] = audit_trail

                # 3. Synchronous atomic update
                await conn.execute(
                    """
                    UPDATE document_chunks
                    SET content = $1, 
                        content_hash = $2, 
                        embedding = $3, 
                        metadata = $4::jsonb
                    WHERE id = $5
                    """,
                    new_content,
                    new_hash,
                    str(new_embedding),
                    json.dumps(meta),
                    chunk_id
                )
                
                logger.info(f"User [{updated_by}] updated Chunk [{chunk_id}] content.")
                return True
    except Exception as e:
        logger.error(f"Error updating chunk {chunk_id}: {e}")
        return False

def _extract_page_number(raw_metadata: object) -> int:
    if raw_metadata is None:
        return 0
    metadata: dict = {}
    if isinstance(raw_metadata, str):
        try:
            metadata = json.loads(raw_metadata)
        except Exception:
            metadata = {}
    elif isinstance(raw_metadata, dict):
        metadata = raw_metadata
    page_candidates = [
        metadata.get("page"),
        metadata.get("page_number"),
        metadata.get("pageNumber"),
        metadata.get("pagenum"),
    ]
    for value in page_candidates:
        try:
            if value is None:
                continue
            page = int(value)
            if page >= 0:
                return page
        except Exception:
            continue
    return 0


def _chunk_row_to_dict(row: object, distance: Optional[float] = None, **extra: object) -> Dict[str, object]:
    """Normalize a DB row to the same shape as :func:`search_vectors` results."""
    base: Dict[str, object] = {
        "chunk_id": str(row.get("chunk_id", "")),
        "document_id": str(row.get("document_id", "")) if row.get("document_id") is not None else "",
        "chunk_index": int(row.get("chunk_index", 0)) if row.get("chunk_index") is not None else 0,
        "content": row.get("content"),
        "file_name": row.get("file_name"),
        "original_filename": row.get("original_filename") or row.get("file_name"),
        "source_uri": row.get("source_uri"),
        "metadata": row.get("metadata"),
        "page_number": _extract_page_number(row.get("metadata")),
        "snippet": (row.get("content") or "").strip(),
        "distance": float(distance) if distance is not None else None,
    }
    for k, v in extra.items():
        if v is not None:
            base[k] = v
    return base


_VI_SYNONYM_MAP: dict[str, list[str]] = {
    "kỷ luật": ["kỉ luật"],
    "kỉ luật": ["kỷ luật"],
    "đăng ký": ["đăng kí"],
    "đăng kí": ["đăng ký"],
    "buộc thôi học": ["đuổi học"],
    "đuổi học": ["buộc thôi học"],
    "bảo lưu": ["nghỉ tạm"],
}

def _expand_synonyms(terms: list[str]) -> list[str]:
    """Expand Vietnamese terms with known synonym variants."""
    expanded: list[str] = list(terms)
    seen = {t.lower() for t in terms}
    for t in terms:
        key = t.lower().strip()
        for synonym in _VI_SYNONYM_MAP.get(key, []):
            if synonym.lower() not in seen:
                seen.add(synonym.lower())
                expanded.append(synonym)
    return expanded


async def search_chunks_keyword_ilike(
    terms: list[str],
    document_type: str,
    limit: int = 15,
) -> List[Dict[str, object]]:
    """
    Keyword supplement for regulation RAG: return chunks using Full-Text Search ranking.
    Uses ts_rank with 'simple' configuration for precise keyword matching.
    """
    cleaned: list[str] = []
    seen: set[str] = set()
    for t in terms or []:
        t = (t or "").strip()
        if len(t) < 2:
            continue
        key = t.lower()
        if key in seen:
            continue
        seen.add(key)
        cleaned.append(t)
    cleaned = cleaned[:12]
    cleaned = _expand_synonyms(cleaned)
    cleaned = cleaned[:20]  # Allow more after expansion
    if not cleaned:
        return []

    try:
        pool = get_db_pool()
        async with pool.acquire() as conn:
            # Construct tsquery: '(term1_word1 & term1_word2) | (term2_word1)'
            # to_tsquery requires operators between words. We join words within a term with '&' (AND).
            formatted_terms = []
            for t in cleaned:
                # Remove punctuation and split into words
                words = re.sub(r"[^\w\s]", "", t).split()
                if words:
                    formatted_terms.append(f"({' & '.join(words)})")
            
            query_str = " | ".join(formatted_terms)
            
            sql = f"""
                    SELECT 
                        c.id AS chunk_id,
                        c.document_id,
                        c.chunk_index,
                        c.content,
                        c.metadata,
                        d.filename AS file_name,
                        d.title AS original_filename,
                        d.source_uri,
                        ts_rank(to_tsvector('simple', c.content), to_tsquery('simple', $2)) AS keyword_score
                    FROM document_chunks c
                    JOIN documents d ON c.document_id = d.id
                    WHERE d.status = 'READY'
                      AND c.embedding IS NOT NULL
                      AND d.document_type = $1
                      AND to_tsvector('simple', c.content) @@ to_tsquery('simple', $2)
                    ORDER BY keyword_score DESC,
                             c.document_id,
                             c.chunk_index
                    LIMIT $3
                    """
            rows = await conn.fetch(sql, document_type, query_str, limit)
            out: List[Dict[str, object]] = []
            for row in rows:
                ks = float(row.get("keyword_score") or 0.0)
                out.append(_chunk_row_to_dict(row, distance=None, keyword_score=ks))
            return out
    except Exception as e:
        logger.exception("Full-Text Search failed (limit=%s): %s", limit, e)
        return []


async def fetch_regulation_neighbor_chunks(
    seed_chunks: list[dict],
    document_type: str,
    window: int = 1,
    max_seeds: int = 10,
) -> List[Dict[str, object]]:
    """
    Load adjacent chunk_index rows (±window) for the same document(s) as seeds.
    Used to widen regulation context when answers span chunk boundaries / pages.
    """
    by_doc: dict[str, set[int]] = {}
    for c in (seed_chunks or [])[:max_seeds]:
        did = str(c.get("document_id") or "").strip()
        if not did:
            continue
        idx = int(c.get("chunk_index") or 0)
        for j in range(max(0, idx - window), idx + window + 1):
            by_doc.setdefault(did, set()).add(j)
    if not by_doc:
        return []

    try:
        pool = get_db_pool()
        async with pool.acquire() as conn:
            merged_rows: list = []
            for did, indices in by_doc.items():
                idx_list = sorted(indices)
                rows = await conn.fetch(
                    """
                    SELECT
                        c.id AS chunk_id,
                        c.document_id,
                        c.chunk_index,
                        c.content,
                        c.metadata,
                        d.filename AS file_name,
                        d.title AS original_filename,
                        d.source_uri
                    FROM document_chunks c
                    JOIN documents d ON c.document_id = d.id
                    WHERE d.status = 'READY'
                      AND c.embedding IS NOT NULL
                      AND d.document_type = $2
                      AND c.document_id = $1::uuid
                      AND c.chunk_index = ANY($3::int[])
                    ORDER BY c.chunk_index, c.id
                    """,
                    did,
                    document_type,
                    idx_list,
                )
                merged_rows.extend(rows)
            out: List[Dict[str, object]] = []
            seen: set[str] = set()
            for row in merged_rows:
                cid = str(row.get("chunk_id", "") or "")
                if not cid or cid in seen:
                    continue
                seen.add(cid)
                out.append(_chunk_row_to_dict(row, distance=None))
            return out
    except Exception as e:
        logger.exception("Neighbor chunk fetch failed: %s", e)
        return []


async def search_vectors(
    query_vector: List[float],
    limit: int = 15,
    document_type: Optional[str] = None,
) -> List[Dict[str, object]]:
    """
    Performs a vector similarity search in PostgreSQL using pgvector.
    Returns the most relevant document chunks based on cosine distance.
    
    Logic: Uses <=> operator for Cosine Distance.
    When ``document_type`` is set (e.g. COURSE_MATERIAL, REGULATION), SQL filters ``documents.document_type``.
    """
    try:
        pool = get_db_pool()
        async with pool.acquire() as conn:
            # Query joining chunks with documents to get filenames
            # Embedding is passed as a string representation of the list for pgvector
            if document_type:
                rows = await conn.fetch(
                    """
                    SELECT
                        c.id AS chunk_id,
                        c.document_id,
                        c.chunk_index,
                        c.content,
                        c.metadata,
                        d.filename AS file_name,
                    d.title AS original_filename,
                        d.source_uri,
                        (c.embedding <=> $1) AS distance
                    FROM document_chunks c
                    JOIN documents d ON c.document_id = d.id
                    WHERE d.status = 'READY'
                      AND c.embedding IS NOT NULL
                      AND d.document_type = $3
                    ORDER BY c.embedding <=> $1, c.document_id, c.chunk_index, c.id
                    LIMIT $2
                    """,
                    str(query_vector),
                    limit,
                    document_type,
                )
            else:
                rows = await conn.fetch(
                    """
                    SELECT
                        c.id AS chunk_id,
                        c.document_id,
                        c.chunk_index,
                        c.content,
                        c.metadata,
                        d.filename AS file_name,
                        d.title AS original_filename,
                        d.source_uri,
                        (c.embedding <=> $1) AS distance
                    FROM document_chunks c
                    JOIN documents d ON c.document_id = d.id
                    WHERE d.status = 'READY'
                      AND c.embedding IS NOT NULL
                    ORDER BY c.embedding <=> $1, c.document_id, c.chunk_index, c.id
                    LIMIT $2
                    """,
                    str(query_vector),
                    limit,
                )
            
            return [
                _chunk_row_to_dict(
                    row,
                    float(row.get("distance")) if row.get("distance") is not None else None,
                )
                for row in rows
            ]
            
    except Exception as e:
        logger.exception("Vector search failed (query_limit=%s): %s", limit, e)
        return []

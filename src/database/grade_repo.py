"""Persistence helpers for private, structured grade-report records."""

from __future__ import annotations

import json
import logging

from src.database.connection import get_db_pool
from data_pipeline.pipeline.grade_report import parse_grade_report


logger = logging.getLogger(__name__)


async def get_document_type(document_id: str) -> str:
    pool = get_db_pool()
    async with pool.acquire() as conn:
        value = await conn.fetchval(
            "SELECT document_type FROM documents WHERE id = $1",
            document_id,
        )
    return str(value or "COURSE_MATERIAL")


async def replace_grade_records(document_id: str, records: list[dict]) -> int:
    """Atomically replace all structured grades belonging to one document."""
    pool = get_db_pool()
    async with pool.acquire() as conn:
        async with conn.transaction():
            await conn.execute("DELETE FROM grade_records WHERE document_id = $1", document_id)
            if not records:
                return 0
            values = [
                (
                    document_id,
                    record["student_code"],
                    record.get("student_name"),
                    record["assignment_title"],
                    record.get("total_score"),
                    record.get("max_score"),
                    record.get("feedback"),
                    json.dumps(record.get("component_scores") or {}, ensure_ascii=False),
                    record.get("source_page"),
                    record.get("source_notice"),
                )
                for record in records
            ]
            await conn.executemany(
                """
                INSERT INTO grade_records (
                    document_id, student_code, student_name, assignment_title,
                    total_score, max_score, feedback, component_scores,
                    source_page, source_notice
                )
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10)
                ON CONFLICT (document_id, student_code, assignment_title)
                DO UPDATE SET
                    student_name = EXCLUDED.student_name,
                    total_score = EXCLUDED.total_score,
                    max_score = EXCLUDED.max_score,
                    feedback = EXCLUDED.feedback,
                    component_scores = EXCLUDED.component_scores,
                    source_page = EXCLUDED.source_page,
                    source_notice = EXCLUDED.source_notice,
                    updated_at = NOW()
                """,
                values,
            )
    logger.info("Persisted %s private grade records for document %s", len(records), document_id)
    return len(records)


async def backfill_legacy_grade_reports() -> int:
    """Convert legacy grade PDFs accidentally indexed as course RAG documents.

    Older UI versions defaulted every upload to ``COURSE_MATERIAL``. For
    filenames that clearly identify a grade report, parse the existing chunks,
    persist private rows, reclassify the document, then remove class-wide RAG
    chunks. The conversion is atomic per document.
    """
    pool = get_db_pool()
    async with pool.acquire() as conn:
        candidates = await conn.fetch(
            """
            SELECT d.id, d.title,
                   string_agg(c.content, E'\n' ORDER BY c.chunk_index) AS content
            FROM documents d
            JOIN document_chunks c ON c.document_id = d.id
            WHERE d.status = 'READY'
              AND d.document_type = 'COURSE_MATERIAL'
              AND (
                lower(d.title) LIKE '%bang_diem%'
                OR lower(d.title) LIKE '%bang-diem%'
                OR lower(d.title) LIKE '%bang diem%'
                OR lower(d.title) LIKE '%bảng điểm%'
                OR lower(d.title) LIKE '%grade_report%'
                OR lower(d.title) LIKE '%grade-report%'
                OR lower(d.title) LIKE '%grade report%'
              )
            GROUP BY d.id, d.title
            """
        )

    converted = 0
    for candidate in candidates:
        records = parse_grade_report(candidate["content"] or "", candidate["title"] or "")
        if not records:
            logger.warning("Legacy grade candidate %s had no parseable rows", candidate["id"])
            continue
        async with pool.acquire() as conn:
            async with conn.transaction():
                await conn.execute("DELETE FROM grade_records WHERE document_id = $1", candidate["id"])
                values = [
                    (
                        candidate["id"],
                        record["student_code"],
                        record.get("student_name"),
                        record["assignment_title"],
                        record.get("total_score"),
                        record.get("max_score"),
                        record.get("feedback"),
                        json.dumps(record.get("component_scores") or {}, ensure_ascii=False),
                        record.get("source_page"),
                        record.get("source_notice"),
                    )
                    for record in records
                ]
                await conn.executemany(
                    """
                    INSERT INTO grade_records (
                        document_id, student_code, student_name, assignment_title,
                        total_score, max_score, feedback, component_scores,
                        source_page, source_notice
                    )
                    VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10)
                    ON CONFLICT (document_id, student_code, assignment_title)
                    DO UPDATE SET
                        student_name = EXCLUDED.student_name,
                        total_score = EXCLUDED.total_score,
                        max_score = EXCLUDED.max_score,
                        feedback = EXCLUDED.feedback,
                        component_scores = EXCLUDED.component_scores,
                        source_page = EXCLUDED.source_page,
                        source_notice = EXCLUDED.source_notice,
                        updated_at = NOW()
                    """,
                    values,
                )
                await conn.execute(
                    "UPDATE documents SET document_type = 'GRADE_REPORT', updated_at = NOW() WHERE id = $1",
                    candidate["id"],
                )
                await conn.execute("DELETE FROM document_chunks WHERE document_id = $1", candidate["id"])
        converted += 1
        logger.info(
            "Converted legacy grade report %s to %s private rows",
            candidate["id"],
            len(records),
        )
    return converted

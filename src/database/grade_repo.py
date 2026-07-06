"""Persistence helpers for private, structured grade-report records."""

from __future__ import annotations

import json
import logging

from src.database.connection import get_db_pool


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

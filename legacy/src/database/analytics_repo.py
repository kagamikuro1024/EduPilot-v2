import logging
from src.database.connection import get_db_pool

logger = logging.getLogger(__name__)

async def log_security_event(session_id: str, query_content: str, violation_reason: str):
    """Logs a blocked security violation (e.g. Prompt Injection) to the database."""
    import uuid
    try:
        # Poka-Yoke: Sanitize session_id to prevent SQL Injection
        uuid.UUID(str(session_id), version=4)
    except (ValueError, AttributeError):
        logger.error(f"Invalid session_id format: {session_id}")
        raise ValueError(f"Invalid session_id: {session_id}")

    try:
        pool = get_db_pool()
        async with pool.acquire() as conn:
            # 1. Try to find student_id from chat_sessions
            student_id = await conn.fetchval(
                "SELECT student_id FROM chat_sessions WHERE id = $1", 
                session_id
            )
            
            # 2. If not found, try forum_threads (author_id)
            if not student_id:
                student_id = await conn.fetchval(
                    "SELECT author_id FROM forum_threads WHERE id = $1",
                    session_id
                )

            # 3. Log the event
            await conn.execute(
                """
                INSERT INTO security_events (student_id, session_id, query_content, violation_reason)
                VALUES ($1, $2, $3, $4)
                """,
                student_id, # Could still be None if not found
                session_id,
                query_content,
                violation_reason
            )
            logger.info(f"Security event logged for session {session_id}: {violation_reason}")
    except Exception as e:
        logger.error(f"Failed to log security event: {e}")

async def get_student_risk_level(thread_id: str) -> str | None:
    """Gets the most recent risk_level for the author of a given thread_id."""
    try:
        pool = get_db_pool()
        async with pool.acquire() as conn:
            # First, find the student_id for this thread (or chat_session)
            student_id = await conn.fetchval(
                "SELECT author_id FROM forum_threads WHERE id = $1",
                thread_id
            )
            if not student_id:
                student_id = await conn.fetchval(
                    "SELECT student_id FROM chat_sessions WHERE id = $1", 
                    thread_id
                )
                
            if not student_id:
                return None
                
            # Then get their latest risk_level
            risk_level = await conn.fetchval(
                """
                SELECT risk_level FROM analytics_at_risk_students 
                WHERE student_id = $1 
                ORDER BY report_date DESC 
                LIMIT 1
                """,
                student_id
            )
            return risk_level
    except Exception as e:
        logger.error(f"Failed to get student risk level: {e}")
        return None

"""
Module: src.database.connection
Description: Manages asynchronous PostgreSQL connections using asyncpg.
             AsyncPG is chosen for its high-performance, non-blocking I/O
             compatible with FastAPI's event loop.

How it works:
    1. During application startup, call init_db_pool() to establish a connection pool.
    2. To execute queries, use get_db_pool() and pool.acquire() to borrow a connection.
    3. During application shutdown, call close_db_pool() to release all resources.
"""

import logging
import asyncpg
from src.config import DATABASE_URL

logger = logging.getLogger(__name__)

# Global variable to hold the persistent database connection pool.
_db_pool: asyncpg.Pool | None = None

async def init_db_pool(
    min_size: int = 1,
    max_size: int = 5,
) -> asyncpg.Pool:
    """
    Initializes the PostgreSQL connection pool.

    Args:
        min_size (int): Min number of connections to maintain.
        max_size (int): Max number of connections in the pool.

    Returns:
        asyncpg.Pool: The initialized connection pool.
    """
    global _db_pool

    if not DATABASE_URL:
        raise ValueError(
            "DATABASE_URL is not configured. "
            "Verify your .env file or environment variables."
        )

    async def setup_connection(conn):
        # Enable TCP keep-alive to prevent remote server from dropping the connection
        try:
            sock = conn._protocol.transport.get_extra_info('socket')
            if sock:
                import socket
                sock.setsockopt(socket.SOL_SOCKET, socket.SO_KEEPALIVE, 1)
        except Exception:
            pass

    try:
        _db_pool = await asyncpg.create_pool(
            dsn=DATABASE_URL,
            min_size=min_size,
            max_size=max_size,
            command_timeout=30,
            # Validation: ensure connection is alive before using it
            max_inactive_connection_lifetime=300, 
            setup=setup_connection
        )
        logger.info(
            "PostgreSQL connection pool initialized with Keep-Alive | Size: min=%d, max=%d",
            min_size,
            max_size,
        )
        return _db_pool
    except Exception as exc:
        logger.error("Failed to connect to PostgreSQL: %s", exc)
        raise

def get_db_pool() -> asyncpg.Pool:
    """
    Retrieves the currently active database connection pool.
    """
    if _db_pool is None:
        raise RuntimeError(
            "Database pool is not initialized. "
            "Ensure init_db_pool() is called during application startup."
        )
    return _db_pool

async def close_db_pool() -> None:
    """
    Closes the connection pool and releases all database connections.
    """
    global _db_pool
    if _db_pool is not None:
        await _db_pool.close()
        _db_pool = None
        logger.info("PostgreSQL connection pool closed.")

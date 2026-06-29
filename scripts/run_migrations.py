import asyncio
import os
import sys
import logging
import asyncpg
from pathlib import Path

# Thêm thư mục gốc vào path để import được src
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from src.config import DATABASE_URL

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
logger = logging.getLogger(__name__)

async def apply_migrations():
    """
    Applies SQL migrations from db/migration directory to the database.
    """
    migration_dir = Path("db/migration")
    if not migration_dir.exists():
        logger.error(f"Migration directory {migration_dir} not found.")
        return False

    # Get all .sql files, sorted naturally by version number
    def get_version(path):
        try:
            return int(path.name.split("__")[0][1:])
        except:
            return 999

    migration_files = sorted([f for f in migration_dir.glob("*.sql") if get_version(f) >= 10], key=get_version)
    
    if not migration_files:
        logger.info("No migration files found.")
        return True

    try:
        conn = await asyncpg.connect(DATABASE_URL)
        logger.info(f"Connected to database: {DATABASE_URL.split('@')[-1]}")

        # Simple migration tracking
        await conn.execute("""
            CREATE TABLE IF NOT EXISTS _python_migrations (
                version TEXT PRIMARY KEY,
                applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            )
        """)

        for m_file in migration_files:
            version = m_file.name.split("__")[0]
            
            # Check if applied
            already_applied = await conn.fetchval(
                "SELECT EXISTS(SELECT 1 FROM _python_migrations WHERE version = $1)", version
            )
            
            if already_applied:
                logger.info(f"Migration {m_file.name} already applied. Skipping.")
                continue

            logger.info(f"Applying migration: {m_file.name}...")
            sql = m_file.read_text(encoding='utf-8')
            
            async with conn.transaction():
                # Split by semicolon for multiple statements if necessary, 
                # but asyncpg.execute can handle multiple statements.
                await conn.execute(sql)
                await conn.execute(
                    "INSERT INTO _python_migrations (version) VALUES ($1)", version
                )
            logger.info(f"✓ Migration {m_file.name} applied successfully.")

        await conn.close()
        return True
    except Exception as e:
        logger.error(f"Failed to apply migrations: {e}")
        return False

if __name__ == "__main__":
    success = asyncio.run(apply_migrations())
    sys.exit(0 if success else 1)

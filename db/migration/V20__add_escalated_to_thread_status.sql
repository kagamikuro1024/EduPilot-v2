DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_enum e
        JOIN pg_type t ON t.oid = e.enumtypid
        WHERE t.typname = 'thread_status'
          AND e.enumlabel = 'ESCALATED'
    ) THEN
        ALTER TYPE thread_status ADD VALUE 'ESCALATED';
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_enum e
        JOIN pg_type t ON t.oid = e.enumtypid
        WHERE t.typname = 'verification_status'
          AND e.enumlabel = 'CORRECTED'
    ) THEN
        ALTER TYPE verification_status ADD VALUE 'CORRECTED';
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_enum e
        JOIN pg_type t ON t.oid = e.enumtypid
        WHERE t.typname = 'verification_status'
          AND e.enumlabel = 'REJECTED'
    ) THEN
        ALTER TYPE verification_status ADD VALUE 'REJECTED';
    END IF;
END$$;

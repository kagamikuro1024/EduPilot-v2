-- Private, structured grade data extracted from uploaded grade reports.
-- Grade records are intentionally kept out of the general RAG corpus so an
-- AI response can never expose another student's row.

ALTER TABLE public.documents
    DROP CONSTRAINT IF EXISTS documents_document_type_check;

ALTER TABLE public.documents
    ADD CONSTRAINT documents_document_type_check
    CHECK (document_type IN ('COURSE_MATERIAL', 'REGULATION', 'GRADE_REPORT'));

CREATE TABLE IF NOT EXISTS public.grade_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES public.documents(id) ON DELETE CASCADE,
    student_code VARCHAR(50) NOT NULL,
    student_name VARCHAR(255),
    assignment_title VARCHAR(255) NOT NULL,
    total_score DOUBLE PRECISION,
    max_score DOUBLE PRECISION,
    feedback TEXT,
    component_scores JSONB NOT NULL DEFAULT '{}'::jsonb,
    source_page INTEGER,
    source_notice TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT grade_records_document_student_assignment_key
        UNIQUE (document_id, student_code, assignment_title)
);

CREATE INDEX IF NOT EXISTS idx_grade_records_student_code
    ON public.grade_records (upper(student_code));

CREATE INDEX IF NOT EXISTS idx_grade_records_assignment_title
    ON public.grade_records (lower(assignment_title));

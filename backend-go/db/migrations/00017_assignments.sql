-- Bài tập, bài nộp, kết quả chấm, phúc khảo, thư vào, cấu hình tích hợp (SRS FEAT-assignments mục 5; proposals sprint 9 #2; TLR-3 A, TLR-8, TLR-11, TLR-16).
-- Một tệp, một giao dịch. Ngoài CREATE chỉ có đúng HAI câu ALTER có chủ ý (D45):
--   (1) FK phức hợp `grade_items (course_id, assignment_id) → assignments (course_id, id)` (cột đã có ở 00016, CHECK source='ASSIGNMENT' đã có);
--   (2) `ALTER TYPE reminder_source ADD VALUE 'ASSIGNMENT'` (Postgres >= 12 cho phép trong giao dịch; tệp này không DÙNG giá trị mới).
-- FK không khai ON DELETE (NO ACTION): xoá bản ghi còn được trỏ tới bị từ chối với 23503 như 00016 (RESTRICT cho 23001 — AC4 cần 23503).
-- Mọi FK là phức hợp có course_id (không gắn chéo lớp: 23503); mọi chỉ mục bắt đầu bằng course_id, ngoại lệ có tên duy nhất: grading_results_lease_idx (run_state, lease_until).

-- +goose Up
-- assignments -----------------------------------------------------------------------------------------------------------------
CREATE TABLE assignments (
    id                   uuid          NOT NULL DEFAULT uuidv7(),
    course_id            uuid          NOT NULL,
    title                text          NOT NULL,
    description          text          NOT NULL DEFAULT '',
    type                 text          NOT NULL DEFAULT 'ESSAY',
    max_score            numeric(5,2)  NOT NULL DEFAULT 10,
    due_at               timestamptz   NOT NULL,
    allow_late           boolean       NOT NULL DEFAULT false,
    late_penalty_per_day numeric(5,2)  NOT NULL DEFAULT 0,
    late_max_days        smallint      NOT NULL DEFAULT 0,
    appeal_days          smallint      NOT NULL DEFAULT 7,
    channels             text[]        NOT NULL DEFAULT ARRAY['IN_APP'],
    submission_code      text          NOT NULL,
    reference_answer     text,
    rubric               jsonb         NOT NULL DEFAULT '[]',
    teams_class_id       text,
    teams_assignment_id  text,
    published_at         timestamptz,
    created_by           uuid          NOT NULL,
    version              integer       NOT NULL DEFAULT 1,
    created_at           timestamptz   NOT NULL DEFAULT now(),
    updated_at           timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT assignments_pkey PRIMARY KEY (id),
    CONSTRAINT assignments_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT assignments_created_by_fkey FOREIGN KEY (created_by) REFERENCES users (id),
    CONSTRAINT assignments_type_chk CHECK (type IN ('ESSAY', 'QUIZ')),
    CONSTRAINT assignments_title_chk CHECK (length(btrim(title)) BETWEEN 1 AND 200),
    CONSTRAINT assignments_max_score_chk CHECK (max_score > 0 AND max_score <= 100 AND max_score % 0.25 = 0),
    CONSTRAINT assignments_late_penalty_chk CHECK (late_penalty_per_day >= 0),
    CONSTRAINT assignments_late_max_days_chk CHECK (late_max_days BETWEEN 0 AND 14),
    CONSTRAINT assignments_allow_late_chk CHECK (allow_late = false OR late_max_days >= 1),
    CONSTRAINT assignments_appeal_days_chk CHECK (appeal_days BETWEEN 0 AND 30),
    CONSTRAINT assignments_channels_chk CHECK (channels <@ ARRAY['IN_APP', 'UPLOAD', 'IMAP', 'TEAMS']::text[] AND cardinality(channels) >= 1),
    CONSTRAINT assignments_code_chk CHECK (submission_code ~ '^[A-Z0-9]{2,12}$'),
    CONSTRAINT assignments_teams_chk CHECK (('TEAMS' = ANY (channels)) = (teams_class_id IS NOT NULL) AND (teams_class_id IS NULL) = (teams_assignment_id IS NULL)),
    CONSTRAINT assignments_version_chk CHECK (version >= 1),
    CONSTRAINT assignments_course_id_key UNIQUE (course_id, id),
    CONSTRAINT assignments_code_key UNIQUE (course_id, submission_code)
);
CREATE INDEX assignments_course_due_idx ON assignments (course_id, due_at);
CREATE INDEX assignments_course_published_idx ON assignments (course_id, published_at);

-- submissions -----------------------------------------------------------------------------------------------------------------
CREATE TABLE submissions (
    id                   uuid         NOT NULL DEFAULT uuidv7(),
    course_id            uuid         NOT NULL,
    assignment_id        uuid,
    student_id           uuid,
    candidate_student_id uuid,
    source               text         NOT NULL,
    version              integer,
    blob_key             text         NOT NULL,
    file_name            text         NOT NULL,
    file_size            bigint       NOT NULL,
    file_mime            text         NOT NULL,
    sha256               text         NOT NULL,
    extracted_text       text,
    extracted_at         timestamptz,
    extract_status       text,
    external_ref         text,
    external_ts          timestamptz,
    from_email           text,
    subject              text,
    sender_verified      boolean      NOT NULL DEFAULT false,
    match_status         text         NOT NULL,
    match_method         text,
    unmatched_reason     text,
    dismissed_at         timestamptz,
    dismissed_by         uuid,
    dismissed_reason     text,
    submitted_at         timestamptz,
    received_at          timestamptz  NOT NULL DEFAULT now(),
    created_at           timestamptz  NOT NULL DEFAULT now(),
    updated_at           timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT submissions_pkey PRIMARY KEY (id),
    CONSTRAINT submissions_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT submissions_assignment_fkey FOREIGN KEY (course_id, assignment_id) REFERENCES assignments (course_id, id),
    CONSTRAINT submissions_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES enrollments (course_id, user_id),
    CONSTRAINT submissions_candidate_fkey FOREIGN KEY (course_id, candidate_student_id) REFERENCES enrollments (course_id, user_id),
    CONSTRAINT submissions_dismissed_by_fkey FOREIGN KEY (dismissed_by) REFERENCES users (id),
    CONSTRAINT submissions_source_chk CHECK (source IN ('IN_APP', 'UPLOAD', 'IMAP', 'TEAMS', 'FORMS')),
    CONSTRAINT submissions_match_status_chk CHECK (match_status IN ('MATCHED', 'UNMATCHED')),
    CONSTRAINT submissions_match_method_chk CHECK (match_method IN ('EMAIL', 'CODE', 'IN_APP', 'MANUAL')),
    CONSTRAINT submissions_matched_chk CHECK ((match_status = 'MATCHED') = (assignment_id IS NOT NULL AND student_id IS NOT NULL AND match_method IS NOT NULL AND version IS NOT NULL)),
    CONSTRAINT submissions_unmatched_chk CHECK ((match_status = 'UNMATCHED') = (unmatched_reason IS NOT NULL)),
    CONSTRAINT submissions_unmatched_reason_chk CHECK (unmatched_reason IN ('NO_ASSIGNMENT_CODE', 'AMBIGUOUS_ASSIGNMENT', 'NO_STUDENT_SIGNAL', 'EMAIL_CODE_CONFLICT', 'AMBIGUOUS_STUDENT_CODE', 'STUDENT_NOT_IN_COURSE', 'CODE_NOT_VERIFIED', 'SENDER_UNVERIFIED', 'ALREADY_REVIEWED')),
    CONSTRAINT submissions_extract_status_chk CHECK (extract_status IN ('OK', 'FAILED')),
    CONSTRAINT submissions_dismissed_chk CHECK (dismissed_at IS NULL OR match_status = 'UNMATCHED'),
    CONSTRAINT submissions_file_size_chk CHECK (file_size BETWEEN 1 AND 20971520),
    CONSTRAINT submissions_version_chk CHECK (version IS NULL OR version >= 1),
    CONSTRAINT submissions_course_id_key UNIQUE (course_id, id)
);
CREATE UNIQUE INDEX submissions_matched_version_key ON submissions (course_id, assignment_id, student_id, version) WHERE match_status = 'MATCHED';
CREATE UNIQUE INDEX submissions_external_key ON submissions (course_id, source, external_ref, external_ts) WHERE external_ref IS NOT NULL;
CREATE INDEX submissions_assignment_received_idx ON submissions (course_id, assignment_id, received_at, id);
CREATE INDEX submissions_student_version_idx ON submissions (course_id, student_id, assignment_id, version DESC);
CREATE INDEX submissions_unmatched_idx ON submissions (course_id, match_status, received_at) WHERE dismissed_at IS NULL;

-- +goose StatementBegin
CREATE FUNCTION submissions_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.blob_key IS DISTINCT FROM OLD.blob_key OR NEW.sha256 IS DISTINCT FROM OLD.sha256 THEN
        RAISE EXCEPTION 'submission file is immutable' USING ERRCODE = 'P0001';
    END IF;
    IF OLD.extracted_text IS NOT NULL AND NEW.extracted_text IS DISTINCT FROM OLD.extracted_text THEN
        RAISE EXCEPTION 'submission text is immutable' USING ERRCODE = 'P0001';
    END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER submissions_immutable_row BEFORE UPDATE ON submissions FOR EACH ROW EXECUTE FUNCTION submissions_immutable();

-- grading_results -------------------------------------------------------------------------------------------------------------
CREATE TABLE grading_results (
    id                    uuid          NOT NULL DEFAULT uuidv7(),
    course_id             uuid          NOT NULL,
    submission_id         uuid          NOT NULL,
    round                 smallint      NOT NULL DEFAULT 1,
    run_state             text          NOT NULL DEFAULT 'QUEUED',
    status                text          NOT NULL DEFAULT 'DRAFT',
    attempts              smallint      NOT NULL DEFAULT 0,
    not_before            timestamptz   NOT NULL DEFAULT now(),
    lease_owner           text,
    lease_until           timestamptz,
    error_code            text,
    criteria              jsonb         NOT NULL DEFAULT '[]',
    overall_feedback      text,
    total                 numeric(5,2),
    ai_scores             jsonb,
    ai_total_before_late  numeric(5,2),
    late_days             smallint      NOT NULL DEFAULT 0,
    late_penalty          numeric(5,2)  NOT NULL DEFAULT 0,
    late_penalty_manual   boolean       NOT NULL DEFAULT false,
    llm_reported_total    numeric(5,2),
    flags                 text[]        NOT NULL DEFAULT '{}',
    needs_review          boolean       GENERATED ALWAYS AS (flags && ARRAY['NEEDS_REVIEW', 'OFF_TOPIC', 'INJECTION_SUSPECTED', 'EMPTY_SUBMISSION', 'EVIDENCE_NOT_FOUND', 'TOTAL_MISMATCH', 'LOW_CONFIDENCE', 'LATE_AFTER_CUTOFF', 'ANSWER_ECHO']::text[]) STORED,
    confidence            numeric(3,2),
    pass_detail           jsonb,
    deviation             numeric(5,2),
    teacher_changed_criteria text[]     NOT NULL DEFAULT '{}',
    reviewed_by           uuid,
    reviewed_at           timestamptz,
    published_by          uuid,
    published_at          timestamptz,
    version               integer       NOT NULL DEFAULT 1,
    created_at            timestamptz   NOT NULL DEFAULT now(),
    updated_at            timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT grading_results_pkey PRIMARY KEY (id),
    CONSTRAINT grading_results_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT grading_results_submission_fkey FOREIGN KEY (course_id, submission_id) REFERENCES submissions (course_id, id),
    CONSTRAINT grading_results_reviewed_by_fkey FOREIGN KEY (reviewed_by) REFERENCES users (id),
    CONSTRAINT grading_results_published_by_fkey FOREIGN KEY (published_by) REFERENCES users (id),
    CONSTRAINT grading_results_run_state_chk CHECK (run_state IN ('QUEUED', 'RUNNING', 'DONE', 'FAILED')),
    CONSTRAINT grading_results_status_chk CHECK (status IN ('DRAFT', 'REVIEWED', 'PUBLISHED')),
    CONSTRAINT grading_results_done_chk CHECK (status = 'DRAFT' OR run_state = 'DONE'),
    CONSTRAINT grading_results_published_chk CHECK ((status = 'PUBLISHED') = (published_at IS NOT NULL)),
    CONSTRAINT grading_results_flags_chk CHECK (flags <@ ARRAY['EMPTY_SUBMISSION', 'OFF_TOPIC', 'INJECTION_SUSPECTED', 'NEEDS_REVIEW', 'TOTAL_MISMATCH', 'EVIDENCE_NOT_FOUND', 'SCORE_ADJUSTED', 'LOW_CONFIDENCE', 'EXTRACT_FAILED', 'LATE_AFTER_CUTOFF', 'ANSWER_ECHO']::text[]),
    CONSTRAINT grading_results_round_chk CHECK (round BETWEEN 1 AND 20),
    CONSTRAINT grading_results_attempts_chk CHECK (attempts >= 0),
    CONSTRAINT grading_results_late_chk CHECK (late_days >= 0 AND late_penalty >= 0),
    CONSTRAINT grading_results_version_chk CHECK (version >= 1),
    CONSTRAINT grading_results_course_id_key UNIQUE (course_id, id),
    CONSTRAINT grading_results_round_key UNIQUE (course_id, submission_id, round)
);
CREATE INDEX grading_results_submission_idx ON grading_results (course_id, submission_id, round DESC);
CREATE INDEX grading_results_status_idx ON grading_results (course_id, status, run_state);
CREATE INDEX grading_results_review_idx ON grading_results (course_id, needs_review, status);
CREATE INDEX grading_results_lease_idx ON grading_results (run_state, lease_until) WHERE run_state IN ('QUEUED', 'RUNNING');

-- +goose StatementBegin
CREATE FUNCTION grading_results_ai_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.ai_scores IS NOT NULL AND NEW.ai_scores IS DISTINCT FROM OLD.ai_scores THEN
        RAISE EXCEPTION 'ai draft is immutable' USING ERRCODE = 'P0001';
    END IF;
    IF OLD.ai_total_before_late IS NOT NULL AND NEW.ai_total_before_late IS DISTINCT FROM OLD.ai_total_before_late THEN
        RAISE EXCEPTION 'ai draft is immutable' USING ERRCODE = 'P0001';
    END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER grading_results_ai_immutable_row BEFORE UPDATE ON grading_results FOR EACH ROW EXECUTE FUNCTION grading_results_ai_immutable();

-- grade_appeals (TLR-3 A: không qua escalation_tickets) -----------------------------------------------------------------------
CREATE TABLE grade_appeals (
    id            uuid          NOT NULL DEFAULT uuidv7(),
    course_id     uuid          NOT NULL,
    assignment_id uuid          NOT NULL,
    student_id    uuid          NOT NULL,
    submission_id uuid          NOT NULL,
    result_id     uuid          NOT NULL,
    criteria_ids  text[]        NOT NULL,
    reason        text          NOT NULL,
    status        text          NOT NULL DEFAULT 'OPEN',
    score_before  numeric(5,2)  NOT NULL,
    score_after   numeric(5,2),
    response      text,
    decided_by    uuid,
    decided_at    timestamptz,
    version       integer       NOT NULL DEFAULT 1,
    created_at    timestamptz   NOT NULL DEFAULT now(),
    updated_at    timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT grade_appeals_pkey PRIMARY KEY (id),
    CONSTRAINT grade_appeals_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT grade_appeals_assignment_fkey FOREIGN KEY (course_id, assignment_id) REFERENCES assignments (course_id, id),
    CONSTRAINT grade_appeals_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES enrollments (course_id, user_id),
    CONSTRAINT grade_appeals_submission_fkey FOREIGN KEY (course_id, submission_id) REFERENCES submissions (course_id, id),
    CONSTRAINT grade_appeals_result_fkey FOREIGN KEY (course_id, result_id) REFERENCES grading_results (course_id, id),
    CONSTRAINT grade_appeals_decided_by_fkey FOREIGN KEY (decided_by) REFERENCES users (id),
    CONSTRAINT grade_appeals_status_chk CHECK (status IN ('OPEN', 'UPHELD', 'ADJUSTED')),
    CONSTRAINT grade_appeals_open_chk CHECK ((status = 'OPEN') = (decided_at IS NULL)),
    CONSTRAINT grade_appeals_version_chk CHECK (version >= 1),
    CONSTRAINT grade_appeals_once_key UNIQUE (course_id, assignment_id, student_id)
);
CREATE INDEX grade_appeals_status_idx ON grade_appeals (course_id, status, created_at);

-- mail_inbound ----------------------------------------------------------------------------------------------------------------
CREATE TABLE mail_inbound (
    id            uuid         NOT NULL DEFAULT uuidv7(),
    course_id     uuid         NOT NULL,
    uidvalidity   bigint       NOT NULL,
    message_uid   bigint       NOT NULL,
    message_id    text,
    from_email    text         NOT NULL,
    subject       text         NOT NULL DEFAULT '',
    received_at   timestamptz  NOT NULL,
    status        text         NOT NULL,
    reason        text,
    submission_id uuid,
    created_at    timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT mail_inbound_pkey PRIMARY KEY (id),
    CONSTRAINT mail_inbound_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT mail_inbound_submission_fkey FOREIGN KEY (course_id, submission_id) REFERENCES submissions (course_id, id),
    CONSTRAINT mail_inbound_status_chk CHECK (status IN ('ACCEPTED', 'IGNORED', 'FAILED')),
    CONSTRAINT mail_inbound_reason_chk CHECK (reason IN ('AUTO_REPLY', 'NO_ATTACHMENT', 'TOO_MANY_ATTACHMENTS', 'ATTACHMENT_REJECTED', 'PARSE_ERROR')),
    CONSTRAINT mail_inbound_uid_key UNIQUE (course_id, uidvalidity, message_uid)
);
CREATE INDEX mail_inbound_status_idx ON mail_inbound (course_id, status, received_at);

-- integration_configs ---------------------------------------------------------------------------------------------------------
CREATE TABLE integration_configs (
    id            uuid         NOT NULL DEFAULT uuidv7(),
    course_id     uuid         NOT NULL,
    kind          text         NOT NULL,
    config_public jsonb        NOT NULL DEFAULT '{}',
    config_enc    bytea,
    enabled       boolean      NOT NULL DEFAULT false,
    state         jsonb        NOT NULL DEFAULT '{}',
    last_ok_at    timestamptz,
    last_error    text,
    version       integer      NOT NULL DEFAULT 1,
    created_at    timestamptz  NOT NULL DEFAULT now(),
    updated_at    timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT integration_configs_pkey PRIMARY KEY (id),
    CONSTRAINT integration_configs_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT integration_configs_kind_chk CHECK (kind IN ('IMAP', 'SMTP', 'TEAMS')),
    CONSTRAINT integration_configs_version_chk CHECK (version >= 1),
    CONSTRAINT integration_configs_kind_key UNIQUE (course_id, kind)
);

CREATE TRIGGER assignments_set_updated_at BEFORE UPDATE ON assignments FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER submissions_set_updated_at BEFORE UPDATE ON submissions FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER grading_results_set_updated_at BEFORE UPDATE ON grading_results FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER grade_appeals_set_updated_at BEFORE UPDATE ON grade_appeals FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER integration_configs_set_updated_at BEFORE UPDATE ON integration_configs FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- hai ALTER có chủ ý ----------------------------------------------------------------------------------------------------------
ALTER TABLE grade_items ADD CONSTRAINT grade_items_assignment_fkey FOREIGN KEY (course_id, assignment_id) REFERENCES assignments (course_id, id);
ALTER TYPE reminder_source ADD VALUE IF NOT EXISTS 'ASSIGNMENT';

-- +goose Down
-- Down không có ALTER nào (FK ở grade_items đi theo DROP … CASCADE). Giá trị enum `ASSIGNMENT` không gỡ được: ghi chú và xoá các dòng dùng nó để `up` lại sạch.
DELETE FROM reminder_log WHERE source_type::text = 'ASSIGNMENT';
DROP TABLE IF EXISTS integration_configs;
DROP TABLE IF EXISTS mail_inbound;
DROP TABLE IF EXISTS grade_appeals;
DROP TRIGGER IF EXISTS grading_results_ai_immutable_row ON grading_results;
DROP TABLE IF EXISTS grading_results;
DROP FUNCTION IF EXISTS grading_results_ai_immutable();
DROP TRIGGER IF EXISTS submissions_immutable_row ON submissions;
DROP TABLE IF EXISTS submissions;
DROP FUNCTION IF EXISTS submissions_immutable();
DROP TABLE IF EXISTS assignments CASCADE; -- CASCADE chỉ gỡ FK `grade_items_assignment_fkey` (mọi bảng trỏ tới đã bị xoá ở trên)

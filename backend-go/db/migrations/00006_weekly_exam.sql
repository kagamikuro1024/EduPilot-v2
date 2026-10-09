-- Thi hằng tuần (SRS FEAT-weekly-exam mục 5, v1.5): ngân hàng câu hỏi, bài thi, lượt làm, bài nộp code, sự kiện liêm chính, độ giống, phúc khảo.
-- Tạo MỘT lần ở dạng cuối, không ALTER về sau (luật 6). 13 bảng thuộc lớp; mọi FK tới bảng thuộc lớp là FK PHỨC HỢP (course_id, …) để DB
-- chặn trộn dữ liệu giữa các lớp (5.16). `00005` là `vn_fold` của P2, nên PE lấy `00006`.

-- +goose Up
CREATE TYPE question_type AS ENUM ('MCQ_SINGLE', 'MCQ_MULTI', 'TRUE_FALSE', 'CODE', 'SHORT', 'ESSAY');
CREATE TYPE question_difficulty AS ENUM ('EASY', 'MEDIUM', 'HARD');
CREATE TYPE question_origin AS ENUM ('MANUAL', 'AI_DRAFT', 'EXTRACTED', 'GENERATED');
CREATE TYPE question_review_status AS ENUM ('DRAFT', 'PENDING', 'APPROVED', 'REJECTED');
CREATE TYPE checker_kind AS ENUM ('EXACT', 'TOKENS', 'FLOAT_EPS');
CREATE TYPE exam_kind AS ENUM ('MCQ', 'CODE', 'MIXED');
CREATE TYPE exam_status AS ENUM ('DRAFT', 'SCHEDULED', 'OPEN', 'CLOSED', 'PUBLISHED');
CREATE TYPE multi_scoring AS ENUM ('ALL_OR_NOTHING', 'PARTIAL');
CREATE TYPE attempt_status AS ENUM ('IN_PROGRESS', 'GRADING', 'GRADED');
CREATE TYPE attempt_submit_reason AS ENUM ('MANUAL', 'TIMEOUT', 'CLOSED');
CREATE TYPE submission_kind AS ENUM ('RUN', 'SUBMIT');
CREATE TYPE submission_status AS ENUM ('QUEUED', 'RUNNING', 'DONE', 'SUPERSEDED', 'ERROR');
CREATE TYPE judge_verdict AS ENUM ('AC', 'WA', 'TLE', 'MLE', 'RE', 'CE', 'OLE', 'IE');
CREATE TYPE exam_event_type AS ENUM ('TAB_HIDDEN', 'TAB_VISIBLE', 'PASTE', 'OFFLINE', 'ONLINE', 'TAB_TAKEOVER', 'CHAT_BLOCKED');
CREATE TYPE similarity_review_state AS ENUM ('NEW', 'CLEARED', 'FOLLOW_UP');
CREATE TYPE appeal_status AS ENUM ('OPEN', 'UPHELD', 'ADJUSTED');

-- 5.2 question_bank -----------------------------------------------------------------------------------------------------------
CREATE TABLE question_bank (
    id            uuid                   NOT NULL DEFAULT uuidv7(),
    course_id     uuid                   NOT NULL,
    type          question_type          NOT NULL,
    title         text                   NOT NULL,
    topic         text                   NOT NULL,
    difficulty    question_difficulty    NOT NULL DEFAULT 'MEDIUM',
    stem          text                   NOT NULL,
    answer_key    jsonb,
    explanation   text,
    citations     jsonb                  NOT NULL DEFAULT '[]',
    origin        question_origin        NOT NULL DEFAULT 'MANUAL',
    review_status question_review_status NOT NULL DEFAULT 'DRAFT',
    created_by    uuid                   NOT NULL,
    reviewed_by   uuid,
    reviewed_at   timestamptz,
    ai_job_id     uuid,
    archived_at   timestamptz,
    version       integer                NOT NULL DEFAULT 1,
    created_at    timestamptz            NOT NULL DEFAULT now(),
    updated_at    timestamptz            NOT NULL DEFAULT now(),
    CONSTRAINT question_bank_pkey PRIMARY KEY (id),
    CONSTRAINT question_bank_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT question_bank_created_by_fkey FOREIGN KEY (created_by) REFERENCES users (id),
    CONSTRAINT question_bank_reviewed_by_fkey FOREIGN KEY (reviewed_by) REFERENCES users (id),
    CONSTRAINT question_bank_title_chk CHECK (char_length(title) BETWEEN 1 AND 120),
    CONSTRAINT question_bank_topic_chk CHECK (char_length(topic) BETWEEN 1 AND 80),
    CONSTRAINT question_bank_stem_chk CHECK (char_length(stem) BETWEEN 1 AND 8000),
    CONSTRAINT question_bank_answer_key_chk CHECK ((type IN ('MCQ_SINGLE', 'MCQ_MULTI', 'TRUE_FALSE')) = (answer_key IS NOT NULL) OR type IN ('SHORT', 'ESSAY')),
    CONSTRAINT question_bank_explanation_chk CHECK (char_length(explanation) <= 4000),
    CONSTRAINT question_bank_review_chk CHECK ((review_status IN ('APPROVED', 'REJECTED')) = (reviewed_by IS NOT NULL)),
    CONSTRAINT question_bank_version_chk CHECK (version >= 1)
);
CREATE UNIQUE INDEX question_bank_course_id_key ON question_bank (course_id, id);
CREATE INDEX question_bank_course_review_idx ON question_bank (course_id, review_status, created_at DESC, id DESC) WHERE archived_at IS NULL;
CREATE INDEX question_bank_course_topic_idx ON question_bank (course_id, topic, difficulty, type) WHERE archived_at IS NULL;
CREATE INDEX question_bank_course_created_idx ON question_bank (course_id, created_at DESC, id DESC);

-- 5.3 question_options --------------------------------------------------------------------------------------------------------
CREATE TABLE question_options (
    id          uuid     NOT NULL DEFAULT uuidv7(),
    course_id   uuid     NOT NULL,
    question_id uuid     NOT NULL,
    position    smallint NOT NULL,
    body        text     NOT NULL,
    pinned_last boolean  NOT NULL DEFAULT false,
    CONSTRAINT question_options_pkey PRIMARY KEY (id),
    CONSTRAINT question_options_question_fkey FOREIGN KEY (course_id, question_id) REFERENCES question_bank (course_id, id) ON DELETE CASCADE,
    CONSTRAINT question_options_position_chk CHECK (position BETWEEN 1 AND 8),
    CONSTRAINT question_options_body_chk CHECK (char_length(body) BETWEEN 1 AND 1000)
);
CREATE UNIQUE INDEX question_options_question_position_key ON question_options (question_id, position);
CREATE INDEX question_options_course_question_idx ON question_options (course_id, question_id, position);

-- 5.4 code_problems -----------------------------------------------------------------------------------------------------------
CREATE TABLE code_problems (
    question_id                uuid          NOT NULL,
    course_id                  uuid          NOT NULL,
    languages                  text[]        NOT NULL DEFAULT '{cpp17}',
    time_limit_ms              integer       NOT NULL DEFAULT 1000,
    memory_limit_mb            integer       NOT NULL DEFAULT 256,
    output_limit_kb            integer       NOT NULL DEFAULT 1024,
    checker                    checker_kind  NOT NULL DEFAULT 'EXACT',
    float_eps                  numeric(12,10),
    starter_code               jsonb         NOT NULL DEFAULT '{}',
    reference_language         text,
    reference_source           text,
    reference_verified_version integer,
    reference_verified_at      timestamptz,
    tests_version              integer       NOT NULL DEFAULT 1,
    created_at                 timestamptz   NOT NULL DEFAULT now(),
    updated_at                 timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT code_problems_pkey PRIMARY KEY (question_id),
    CONSTRAINT code_problems_question_fkey FOREIGN KEY (course_id, question_id) REFERENCES question_bank (course_id, id) ON DELETE CASCADE,
    CONSTRAINT code_problems_languages_chk CHECK (cardinality(languages) >= 1 AND languages <@ ARRAY['c11', 'cpp17']),
    CONSTRAINT code_problems_time_chk CHECK (time_limit_ms BETWEEN 100 AND 10000),
    CONSTRAINT code_problems_memory_chk CHECK (memory_limit_mb BETWEEN 16 AND 512),
    CONSTRAINT code_problems_output_chk CHECK (output_limit_kb BETWEEN 1 AND 16384),
    CONSTRAINT code_problems_eps_chk CHECK ((checker = 'FLOAT_EPS') = (float_eps IS NOT NULL)),
    CONSTRAINT code_problems_eps_range_chk CHECK (float_eps IS NULL OR (float_eps > 0 AND float_eps <= 0.1)),
    CONSTRAINT code_problems_starter_chk CHECK (jsonb_typeof(starter_code) = 'object'),
    CONSTRAINT code_problems_ref_lang_chk CHECK (reference_language IN ('c11', 'cpp17')),
    CONSTRAINT code_problems_ref_pair_chk CHECK ((reference_language IS NULL) = (reference_source IS NULL)),
    CONSTRAINT code_problems_ref_size_chk CHECK (octet_length(reference_source) <= 65536),
    CONSTRAINT code_problems_tests_version_chk CHECK (tests_version >= 1)
);
CREATE UNIQUE INDEX code_problems_course_question_key ON code_problems (course_id, question_id);
CREATE INDEX code_problems_course_idx ON code_problems (course_id);

-- 5.5 code_testcases ----------------------------------------------------------------------------------------------------------
CREATE TABLE code_testcases (
    id                uuid        NOT NULL DEFAULT uuidv7(),
    course_id         uuid        NOT NULL,
    problem_id        uuid        NOT NULL,
    position          integer     NOT NULL,
    name              text        NOT NULL,
    is_sample         boolean     NOT NULL DEFAULT false,
    weight            smallint    NOT NULL DEFAULT 1,
    input             text,
    input_blob_key    text,
    expected          text,
    expected_blob_key text,
    input_bytes       integer     NOT NULL,
    expected_bytes    integer     NOT NULL,
    source            text        NOT NULL DEFAULT 'MANUAL',
    approved          boolean     NOT NULL DEFAULT true,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT code_testcases_pkey PRIMARY KEY (id),
    CONSTRAINT code_testcases_problem_fkey FOREIGN KEY (course_id, problem_id) REFERENCES code_problems (course_id, question_id) ON DELETE CASCADE,
    CONSTRAINT code_testcases_position_chk CHECK (position >= 1),
    CONSTRAINT code_testcases_name_chk CHECK (name ~ '^[A-Za-z0-9_-]{1,60}$'),
    CONSTRAINT code_testcases_weight_chk CHECK (weight BETWEEN 0 AND 1000),
    CONSTRAINT code_testcases_input_chk CHECK ((input IS NULL) <> (input_blob_key IS NULL)),
    CONSTRAINT code_testcases_expected_chk CHECK ((expected IS NULL) <> (expected_blob_key IS NULL)),
    CONSTRAINT code_testcases_input_inline_chk CHECK (octet_length(input) <= 65536),
    CONSTRAINT code_testcases_expected_inline_chk CHECK (octet_length(expected) <= 65536),
    CONSTRAINT code_testcases_bytes_chk CHECK (input_bytes BETWEEN 0 AND 1048576 AND expected_bytes BETWEEN 0 AND 1048576),
    CONSTRAINT code_testcases_source_chk CHECK (source IN ('MANUAL', 'IMPORT', 'AI_DRAFT'))
);
ALTER TABLE code_testcases ADD CONSTRAINT code_testcases_problem_position_key UNIQUE (problem_id, position) DEFERRABLE INITIALLY DEFERRED;
CREATE INDEX code_testcases_course_problem_idx ON code_testcases (course_id, problem_id, position);

-- 5.6 exams -------------------------------------------------------------------------------------------------------------------
CREATE TABLE exams (
    id                uuid          NOT NULL DEFAULT uuidv7(),
    course_id         uuid          NOT NULL,
    title             text          NOT NULL,
    instructions      text,
    kind              exam_kind     NOT NULL DEFAULT 'MCQ',
    status            exam_status   NOT NULL DEFAULT 'DRAFT',
    opens_at          timestamptz,
    closes_at         timestamptz,
    duration_minutes  smallint,
    shuffle_questions boolean       NOT NULL DEFAULT true,
    shuffle_options   boolean       NOT NULL DEFAULT true,
    max_score         numeric(5,2)  NOT NULL DEFAULT 10.00,
    rounding_step     numeric(3,2)  NOT NULL DEFAULT 0.01,
    multi_scoring     multi_scoring NOT NULL DEFAULT 'PARTIAL',
    reveal_answers    boolean       NOT NULL DEFAULT true,
    appeal_days       smallint      NOT NULL DEFAULT 7,
    publish_hold      boolean       NOT NULL DEFAULT false,
    regrading         boolean       NOT NULL DEFAULT false,
    published_at      timestamptz,
    created_by        uuid          NOT NULL,
    version           integer       NOT NULL DEFAULT 1,
    created_at        timestamptz   NOT NULL DEFAULT now(),
    updated_at        timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT exams_pkey PRIMARY KEY (id),
    CONSTRAINT exams_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT exams_created_by_fkey FOREIGN KEY (created_by) REFERENCES users (id),
    CONSTRAINT exams_title_chk CHECK (char_length(title) BETWEEN 1 AND 120),
    CONSTRAINT exams_instructions_chk CHECK (char_length(instructions) <= 4000),
    CONSTRAINT exams_scheduled_chk CHECK (status = 'DRAFT' OR (opens_at IS NOT NULL AND closes_at IS NOT NULL AND duration_minutes IS NOT NULL)),
    CONSTRAINT exams_window_chk CHECK (opens_at IS NULL OR closes_at IS NULL OR closes_at > opens_at),
    CONSTRAINT exams_duration_chk CHECK (duration_minutes BETWEEN 1 AND 300),
    CONSTRAINT exams_duration_window_chk CHECK (duration_minutes IS NULL OR opens_at IS NULL OR closes_at IS NULL OR duration_minutes * interval '1 minute' <= closes_at - opens_at),
    CONSTRAINT exams_max_score_chk CHECK (max_score > 0 AND max_score <= 100),
    CONSTRAINT exams_step_chk CHECK (rounding_step IN (0.01, 0.10, 0.25, 0.50, 1.00)),
    CONSTRAINT exams_appeal_days_chk CHECK (appeal_days BETWEEN 0 AND 30),
    CONSTRAINT exams_published_chk CHECK ((status = 'PUBLISHED') = (published_at IS NOT NULL)),
    CONSTRAINT exams_version_chk CHECK (version >= 1)
);
CREATE UNIQUE INDEX exams_course_id_key ON exams (course_id, id);
CREATE INDEX exams_course_status_idx ON exams (course_id, status, opens_at DESC, id DESC);
CREATE INDEX exams_course_created_idx ON exams (course_id, created_at DESC, id DESC);
CREATE INDEX exams_due_open_idx ON exams (opens_at) WHERE status = 'SCHEDULED';
CREATE INDEX exams_due_close_idx ON exams (closes_at) WHERE status IN ('SCHEDULED', 'OPEN');
CREATE INDEX exams_closed_idx ON exams (course_id, id) WHERE status = 'CLOSED';

-- 5.7 exam_items --------------------------------------------------------------------------------------------------------------
CREATE TABLE exam_items (
    id          uuid         NOT NULL DEFAULT uuidv7(),
    course_id   uuid         NOT NULL,
    exam_id     uuid         NOT NULL,
    question_id uuid         NOT NULL,
    position    smallint     NOT NULL,
    points      numeric(5,2) NOT NULL DEFAULT 1.00,
    override    jsonb,
    created_at  timestamptz  NOT NULL DEFAULT now(),
    updated_at  timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT exam_items_pkey PRIMARY KEY (id),
    CONSTRAINT exam_items_exam_fkey FOREIGN KEY (course_id, exam_id) REFERENCES exams (course_id, id) ON DELETE CASCADE,
    CONSTRAINT exam_items_question_fkey FOREIGN KEY (course_id, question_id) REFERENCES question_bank (course_id, id),
    CONSTRAINT exam_items_position_chk CHECK (position >= 1),
    CONSTRAINT exam_items_points_chk CHECK (points > 0 AND points <= 100),
    CONSTRAINT exam_items_override_chk CHECK (override IS NULL OR jsonb_typeof(override) = 'object')
);
CREATE UNIQUE INDEX exam_items_course_id_key ON exam_items (course_id, id);
ALTER TABLE exam_items ADD CONSTRAINT exam_items_exam_position_key UNIQUE (exam_id, position) DEFERRABLE INITIALLY DEFERRED;
CREATE UNIQUE INDEX exam_items_exam_question_key ON exam_items (exam_id, question_id);
CREATE INDEX exam_items_course_exam_idx ON exam_items (course_id, exam_id, position);
CREATE INDEX exam_items_question_idx ON exam_items (question_id, exam_id);

-- 5.8 exam_attempts -----------------------------------------------------------------------------------------------------------
CREATE TABLE exam_attempts (
    id              uuid                  NOT NULL DEFAULT uuidv7(),
    course_id       uuid                  NOT NULL,
    exam_id         uuid                  NOT NULL,
    student_id      uuid                  NOT NULL,
    status          attempt_status        NOT NULL DEFAULT 'IN_PROGRESS',
    started_at      timestamptz           NOT NULL DEFAULT now(),
    deadline_at     timestamptz           NOT NULL,
    submitted_at    timestamptz,
    submit_reason   attempt_submit_reason,
    writer_tab      uuid,
    writer_seen_at  timestamptz,
    auto_score      numeric(5,2),
    adjusted_score  numeric(5,2),
    adjusted_reason text,
    adjusted_by     uuid,
    adjusted_at     timestamptz,
    breakdown       jsonb,
    graded_at       timestamptz,
    version         integer               NOT NULL DEFAULT 1,
    created_at      timestamptz           NOT NULL DEFAULT now(),
    updated_at      timestamptz           NOT NULL DEFAULT now(),
    CONSTRAINT exam_attempts_pkey PRIMARY KEY (id),
    CONSTRAINT exam_attempts_exam_fkey FOREIGN KEY (course_id, exam_id) REFERENCES exams (course_id, id),
    CONSTRAINT exam_attempts_student_fkey FOREIGN KEY (student_id) REFERENCES users (id),
    CONSTRAINT exam_attempts_adjusted_by_fkey FOREIGN KEY (adjusted_by) REFERENCES users (id),
    CONSTRAINT exam_attempts_deadline_chk CHECK (deadline_at > started_at),
    CONSTRAINT exam_attempts_submitted_chk CHECK ((status <> 'IN_PROGRESS') = (submitted_at IS NOT NULL)),
    CONSTRAINT exam_attempts_reason_chk CHECK ((submitted_at IS NOT NULL) = (submit_reason IS NOT NULL)),
    CONSTRAINT exam_attempts_auto_score_chk CHECK (auto_score IS NULL OR auto_score BETWEEN 0 AND 100),
    CONSTRAINT exam_attempts_graded_chk CHECK ((status = 'GRADED') = (auto_score IS NOT NULL AND graded_at IS NOT NULL)),
    CONSTRAINT exam_attempts_adjusted_chk CHECK (adjusted_score IS NULL OR (adjusted_score BETWEEN 0 AND 100 AND adjusted_reason IS NOT NULL)),
    CONSTRAINT exam_attempts_adjusted_reason_chk CHECK (char_length(adjusted_reason) BETWEEN 1 AND 500),
    CONSTRAINT exam_attempts_breakdown_chk CHECK (breakdown IS NULL OR jsonb_typeof(breakdown) = 'array'),
    CONSTRAINT exam_attempts_version_chk CHECK (version >= 1)
);
CREATE UNIQUE INDEX exam_attempts_course_id_key ON exam_attempts (course_id, id);
CREATE UNIQUE INDEX exam_attempts_exam_student_key ON exam_attempts (exam_id, student_id);
CREATE INDEX exam_attempts_course_exam_idx ON exam_attempts (course_id, exam_id, status, student_id);
CREATE INDEX exam_attempts_running_idx ON exam_attempts (student_id, deadline_at) WHERE status = 'IN_PROGRESS';
CREATE INDEX exam_attempts_due_idx ON exam_attempts (deadline_at) WHERE status = 'IN_PROGRESS';
CREATE INDEX exam_attempts_student_idx ON exam_attempts (course_id, student_id, started_at DESC);

-- 5.9 exam_answers ------------------------------------------------------------------------------------------------------------
CREATE TABLE exam_answers (
    attempt_id uuid        NOT NULL,
    item_id    uuid        NOT NULL,
    course_id  uuid        NOT NULL,
    answer     jsonb       NOT NULL,
    saved_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT exam_answers_pkey PRIMARY KEY (attempt_id, item_id),
    CONSTRAINT exam_answers_attempt_fkey FOREIGN KEY (course_id, attempt_id) REFERENCES exam_attempts (course_id, id) ON DELETE CASCADE,
    CONSTRAINT exam_answers_item_fkey FOREIGN KEY (course_id, item_id) REFERENCES exam_items (course_id, id) ON DELETE CASCADE,
    CONSTRAINT exam_answers_size_chk CHECK (octet_length(answer::text) <= 4096)
);
CREATE INDEX exam_answers_course_attempt_idx ON exam_answers (course_id, attempt_id);

-- 5.10 code_drafts ------------------------------------------------------------------------------------------------------------
CREATE TABLE code_drafts (
    attempt_id uuid        NOT NULL,
    item_id    uuid        NOT NULL,
    course_id  uuid        NOT NULL,
    language   text        NOT NULL,
    source     text        NOT NULL,
    rev        integer     NOT NULL DEFAULT 1,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT code_drafts_pkey PRIMARY KEY (attempt_id, item_id, language),
    CONSTRAINT code_drafts_attempt_fkey FOREIGN KEY (course_id, attempt_id) REFERENCES exam_attempts (course_id, id) ON DELETE CASCADE,
    CONSTRAINT code_drafts_item_fkey FOREIGN KEY (course_id, item_id) REFERENCES exam_items (course_id, id) ON DELETE CASCADE,
    CONSTRAINT code_drafts_language_chk CHECK (language IN ('c11', 'cpp17')),
    CONSTRAINT code_drafts_source_chk CHECK (octet_length(source) <= 65536),
    CONSTRAINT code_drafts_rev_chk CHECK (rev >= 1)
);
CREATE INDEX code_drafts_course_attempt_idx ON code_drafts (course_id, attempt_id);

-- 5.11 code_submissions -------------------------------------------------------------------------------------------------------
CREATE TABLE code_submissions (
    id              uuid              NOT NULL DEFAULT uuidv7(),
    course_id       uuid              NOT NULL,
    exam_id         uuid              NOT NULL,
    attempt_id      uuid              NOT NULL,
    item_id         uuid              NOT NULL,
    problem_id      uuid              NOT NULL,
    student_id      uuid              NOT NULL,
    kind            submission_kind   NOT NULL,
    language        text              NOT NULL,
    source          text              NOT NULL,
    source_sha256   char(64)          NOT NULL,
    auto            boolean           NOT NULL DEFAULT false,
    status          submission_status NOT NULL DEFAULT 'QUEUED',
    verdict         judge_verdict,
    tests_version   integer,
    compile_ok      boolean,
    compile_log     text,
    results         jsonb,
    passed_weight   integer,
    total_weight    integer,
    time_ms_max     integer,
    memory_kb_max   integer,
    attempts        smallint          NOT NULL DEFAULT 0,
    lease_until     timestamptz,
    next_attempt_at timestamptz       NOT NULL DEFAULT now(),
    fail_count      smallint          NOT NULL DEFAULT 0,
    enqueued_at     timestamptz,
    judged_at       timestamptz,
    created_at      timestamptz       NOT NULL DEFAULT now(),
    updated_at      timestamptz       NOT NULL DEFAULT now(),
    CONSTRAINT code_submissions_pkey PRIMARY KEY (id),
    CONSTRAINT code_submissions_exam_fkey FOREIGN KEY (course_id, exam_id) REFERENCES exams (course_id, id),
    CONSTRAINT code_submissions_attempt_fkey FOREIGN KEY (course_id, attempt_id) REFERENCES exam_attempts (course_id, id),
    CONSTRAINT code_submissions_item_fkey FOREIGN KEY (course_id, item_id) REFERENCES exam_items (course_id, id),
    CONSTRAINT code_submissions_problem_fkey FOREIGN KEY (course_id, problem_id) REFERENCES code_problems (course_id, question_id),
    CONSTRAINT code_submissions_student_fkey FOREIGN KEY (student_id) REFERENCES users (id),
    CONSTRAINT code_submissions_language_chk CHECK (language IN ('c11', 'cpp17')),
    CONSTRAINT code_submissions_source_chk CHECK (octet_length(source) BETWEEN 1 AND 65536),
    CONSTRAINT code_submissions_auto_chk CHECK (NOT auto OR kind = 'SUBMIT'),
    CONSTRAINT code_submissions_verdict_chk CHECK ((status IN ('DONE', 'ERROR')) = (verdict IS NOT NULL)),
    CONSTRAINT code_submissions_compile_log_chk CHECK (char_length(compile_log) <= 8192),
    CONSTRAINT code_submissions_weight_chk CHECK (passed_weight <= total_weight),
    CONSTRAINT code_submissions_lease_chk CHECK ((status = 'RUNNING') = (lease_until IS NOT NULL)),
    CONSTRAINT code_submissions_fail_count_chk CHECK (fail_count BETWEEN 0 AND 4)
);
CREATE UNIQUE INDEX code_submissions_course_id_key ON code_submissions (course_id, id);
CREATE INDEX code_submissions_attempt_item_idx ON code_submissions (course_id, attempt_id, item_id, created_at DESC, id DESC) WHERE kind = 'SUBMIT';
CREATE INDEX code_submissions_runs_idx ON code_submissions (attempt_id, created_at DESC) WHERE kind = 'RUN';
CREATE INDEX code_submissions_queue_idx ON code_submissions (next_attempt_at, id) WHERE status = 'QUEUED' AND enqueued_at IS NULL;
CREATE INDEX code_submissions_lease_idx ON code_submissions (lease_until) WHERE status = 'RUNNING';
CREATE INDEX code_submissions_exam_problem_idx ON code_submissions (course_id, exam_id, problem_id) WHERE kind = 'SUBMIT';

-- 5.12 exam_events ------------------------------------------------------------------------------------------------------------
CREATE TABLE exam_events (
    id          uuid            NOT NULL DEFAULT uuidv7(),
    course_id   uuid            NOT NULL,
    exam_id     uuid            NOT NULL,
    attempt_id  uuid            NOT NULL,
    student_id  uuid            NOT NULL,
    type        exam_event_type NOT NULL,
    occurred_at timestamptz     NOT NULL DEFAULT now(),
    client_at   timestamptz,
    meta        jsonb           NOT NULL DEFAULT '{}',
    CONSTRAINT exam_events_pkey PRIMARY KEY (id),
    CONSTRAINT exam_events_exam_fkey FOREIGN KEY (course_id, exam_id) REFERENCES exams (course_id, id),
    CONSTRAINT exam_events_attempt_fkey FOREIGN KEY (course_id, attempt_id) REFERENCES exam_attempts (course_id, id) ON DELETE CASCADE,
    CONSTRAINT exam_events_student_fkey FOREIGN KEY (student_id) REFERENCES users (id),
    CONSTRAINT exam_events_meta_chk CHECK (octet_length(meta::text) <= 300)
);
CREATE INDEX exam_events_attempt_idx ON exam_events (course_id, exam_id, attempt_id, occurred_at, id);

-- 5.13 similarity_reports -----------------------------------------------------------------------------------------------------
CREATE TABLE similarity_reports (
    id                  uuid                    NOT NULL DEFAULT uuidv7(),
    course_id           uuid                    NOT NULL,
    exam_id             uuid                    NOT NULL,
    problem_id          uuid                    NOT NULL,
    run_id              uuid                    NOT NULL,
    submission_a        uuid                    NOT NULL,
    submission_b        uuid                    NOT NULL,
    attempt_a           uuid                    NOT NULL,
    attempt_b           uuid                    NOT NULL,
    score               numeric(4,3)            NOT NULL,
    shared_fingerprints integer                 NOT NULL,
    flagged             boolean                 NOT NULL DEFAULT false,
    algorithm           text                    NOT NULL DEFAULT 'winnow-k5-w4-jaccard',
    review_state        similarity_review_state NOT NULL DEFAULT 'NEW',
    reviewed_by         uuid,
    reviewed_at         timestamptz,
    note                text,
    created_at          timestamptz             NOT NULL DEFAULT now(),
    CONSTRAINT similarity_reports_pkey PRIMARY KEY (id),
    CONSTRAINT similarity_reports_exam_fkey FOREIGN KEY (course_id, exam_id) REFERENCES exams (course_id, id),
    CONSTRAINT similarity_reports_problem_fkey FOREIGN KEY (course_id, problem_id) REFERENCES code_problems (course_id, question_id),
    CONSTRAINT similarity_reports_sub_a_fkey FOREIGN KEY (course_id, submission_a) REFERENCES code_submissions (course_id, id) ON DELETE CASCADE,
    CONSTRAINT similarity_reports_sub_b_fkey FOREIGN KEY (course_id, submission_b) REFERENCES code_submissions (course_id, id) ON DELETE CASCADE,
    CONSTRAINT similarity_reports_att_a_fkey FOREIGN KEY (course_id, attempt_a) REFERENCES exam_attempts (course_id, id) ON DELETE CASCADE,
    CONSTRAINT similarity_reports_att_b_fkey FOREIGN KEY (course_id, attempt_b) REFERENCES exam_attempts (course_id, id) ON DELETE CASCADE,
    CONSTRAINT similarity_reports_reviewed_by_fkey FOREIGN KEY (reviewed_by) REFERENCES users (id),
    CONSTRAINT similarity_reports_pair_chk CHECK (attempt_a < attempt_b),
    CONSTRAINT similarity_reports_score_chk CHECK (score BETWEEN 0 AND 1),
    CONSTRAINT similarity_reports_review_chk CHECK ((review_state = 'NEW') = (reviewed_by IS NULL)),
    CONSTRAINT similarity_reports_note_chk CHECK (char_length(note) <= 500)
);
CREATE UNIQUE INDEX similarity_reports_run_pair_key ON similarity_reports (run_id, submission_a, submission_b);
CREATE INDEX similarity_reports_exam_idx ON similarity_reports (course_id, exam_id, problem_id, score DESC, id);
CREATE INDEX similarity_reports_run_idx ON similarity_reports (run_id);

-- 5.14 exam_appeals -----------------------------------------------------------------------------------------------------------
CREATE TABLE exam_appeals (
    id            uuid          NOT NULL DEFAULT uuidv7(),
    course_id     uuid          NOT NULL,
    exam_id       uuid          NOT NULL,
    attempt_id    uuid          NOT NULL,
    student_id    uuid          NOT NULL,
    reason        text          NOT NULL,
    status        appeal_status NOT NULL DEFAULT 'OPEN',
    response      text,
    responded_by  uuid,
    responded_at  timestamptz,
    score_before  numeric(5,2),
    score_after   numeric(5,2),
    version       integer       NOT NULL DEFAULT 1,
    created_at    timestamptz   NOT NULL DEFAULT now(),
    updated_at    timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT exam_appeals_pkey PRIMARY KEY (id),
    CONSTRAINT exam_appeals_exam_fkey FOREIGN KEY (course_id, exam_id) REFERENCES exams (course_id, id),
    CONSTRAINT exam_appeals_attempt_fkey FOREIGN KEY (course_id, attempt_id) REFERENCES exam_attempts (course_id, id) ON DELETE CASCADE,
    CONSTRAINT exam_appeals_student_fkey FOREIGN KEY (student_id) REFERENCES users (id),
    CONSTRAINT exam_appeals_responded_by_fkey FOREIGN KEY (responded_by) REFERENCES users (id),
    CONSTRAINT exam_appeals_reason_chk CHECK (char_length(reason) BETWEEN 1 AND 1000),
    CONSTRAINT exam_appeals_response_chk CHECK (char_length(response) BETWEEN 1 AND 1000),
    CONSTRAINT exam_appeals_responded_chk CHECK ((status = 'OPEN') = (responded_at IS NULL)),
    CONSTRAINT exam_appeals_score_after_chk CHECK ((status = 'ADJUSTED') = (score_after IS NOT NULL)),
    CONSTRAINT exam_appeals_version_chk CHECK (version >= 1)
);
CREATE UNIQUE INDEX exam_appeals_attempt_key ON exam_appeals (attempt_id);
CREATE INDEX exam_appeals_course_exam_idx ON exam_appeals (course_id, exam_id, status, created_at);
CREATE INDEX exam_appeals_open_idx ON exam_appeals (course_id, created_at) WHERE status = 'OPEN';

CREATE TRIGGER question_bank_set_updated_at BEFORE UPDATE ON question_bank FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER code_problems_set_updated_at BEFORE UPDATE ON code_problems FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER code_testcases_set_updated_at BEFORE UPDATE ON code_testcases FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER exams_set_updated_at BEFORE UPDATE ON exams FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER exam_items_set_updated_at BEFORE UPDATE ON exam_items FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER exam_attempts_set_updated_at BEFORE UPDATE ON exam_attempts FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER code_drafts_set_updated_at BEFORE UPDATE ON code_drafts FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER code_submissions_set_updated_at BEFORE UPDATE ON code_submissions FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER exam_appeals_set_updated_at BEFORE UPDATE ON exam_appeals FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS exam_appeals;
DROP TABLE IF EXISTS similarity_reports;
DROP TABLE IF EXISTS exam_events;
DROP TABLE IF EXISTS code_submissions;
DROP TABLE IF EXISTS code_drafts;
DROP TABLE IF EXISTS exam_answers;
DROP TABLE IF EXISTS exam_attempts;
DROP TABLE IF EXISTS exam_items;
DROP TABLE IF EXISTS exams;
DROP TABLE IF EXISTS code_testcases;
DROP TABLE IF EXISTS code_problems;
DROP TABLE IF EXISTS question_options;
DROP TABLE IF EXISTS question_bank;
DROP TYPE IF EXISTS appeal_status;
DROP TYPE IF EXISTS similarity_review_state;
DROP TYPE IF EXISTS exam_event_type;
DROP TYPE IF EXISTS judge_verdict;
DROP TYPE IF EXISTS submission_status;
DROP TYPE IF EXISTS submission_kind;
DROP TYPE IF EXISTS attempt_submit_reason;
DROP TYPE IF EXISTS attempt_status;
DROP TYPE IF EXISTS multi_scoring;
DROP TYPE IF EXISTS exam_status;
DROP TYPE IF EXISTS exam_kind;
DROP TYPE IF EXISTS checker_kind;
DROP TYPE IF EXISTS question_review_status;
DROP TYPE IF EXISTS question_origin;
DROP TYPE IF EXISTS question_difficulty;
DROP TYPE IF EXISTS question_type;

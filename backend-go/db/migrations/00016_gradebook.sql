-- Sổ điểm và công thức điểm (SRS FEAT-gradebook mục 5.1; proposals sprint 8 #3 + 4 bổ sung của Tech Lead).
-- Chỉ CREATE (D45): không câu ALTER nào; `version`, `unlock_reason` và mọi cột † của SRS nằm sẵn trong CREATE.
-- `grade_items.assignment_id` chưa có FK: `00017` (P7) thêm ràng buộc khi có bảng `assignments`.

-- +goose Up
CREATE TYPE grade_scheme_status AS ENUM ('DRAFT', 'CONFIRMED', 'SUPERSEDED');
CREATE TYPE grade_group AS ENUM ('QT', 'CK');
CREATE TYPE grade_rule_kind AS ENUM ('BONUS', 'PENALTY', 'ROUNDING', 'FAIL_CONDITION', 'LETTER_SCALE', 'MISSING_SCORE');
CREATE TYPE grade_item_source AS ENUM ('MANUAL', 'EXAM', 'ASSIGNMENT');
CREATE TYPE grade_entry_source AS ENUM ('MANUAL', 'AI_GRADED', 'QUIZ', 'IMPORT');
CREATE TYPE gradebook_status AS ENUM ('OPEN', 'LOCKED');

-- grade_schemes ---------------------------------------------------------------------------------------------------------------
CREATE TABLE grade_schemes (
    id                 uuid                NOT NULL DEFAULT uuidv7(),
    course_id          uuid                NOT NULL,
    scheme_no          integer             NOT NULL,
    status             grade_scheme_status NOT NULL DEFAULT 'DRAFT',
    source_document_id uuid,
    source_title       text,
    extraction         jsonb,
    extract_job_id     uuid,
    confirmed_by       uuid,
    confirmed_at       timestamptz,
    created_by         uuid                NOT NULL,
    version            integer             NOT NULL DEFAULT 1,
    created_at         timestamptz         NOT NULL DEFAULT now(),
    updated_at         timestamptz         NOT NULL DEFAULT now(),
    CONSTRAINT grade_schemes_pkey PRIMARY KEY (id),
    CONSTRAINT grade_schemes_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT grade_schemes_document_fkey FOREIGN KEY (source_document_id) REFERENCES documents (id) ON DELETE SET NULL,
    CONSTRAINT grade_schemes_job_fkey FOREIGN KEY (extract_job_id) REFERENCES jobs (id) ON DELETE SET NULL,
    CONSTRAINT grade_schemes_confirmed_by_fkey FOREIGN KEY (confirmed_by) REFERENCES users (id),
    CONSTRAINT grade_schemes_created_by_fkey FOREIGN KEY (created_by) REFERENCES users (id),
    CONSTRAINT grade_schemes_no_chk CHECK (scheme_no >= 1),
    CONSTRAINT grade_schemes_version_chk CHECK (version >= 1),
    CONSTRAINT grade_schemes_confirmed_chk CHECK ((status = 'DRAFT') = (confirmed_at IS NULL AND confirmed_by IS NULL)),
    CONSTRAINT grade_schemes_no_key UNIQUE (course_id, scheme_no),
    CONSTRAINT grade_schemes_course_id_key UNIQUE (course_id, id),
    CONSTRAINT grade_schemes_extract_job_key UNIQUE (extract_job_id)
);
CREATE UNIQUE INDEX grade_schemes_one_draft_idx ON grade_schemes (course_id) WHERE status = 'DRAFT';
CREATE UNIQUE INDEX grade_schemes_one_confirmed_idx ON grade_schemes (course_id) WHERE status = 'CONFIRMED';
CREATE INDEX grade_schemes_course_status_idx ON grade_schemes (course_id, status);

-- grade_components ------------------------------------------------------------------------------------------------------------
CREATE TABLE grade_components (
    id             uuid          NOT NULL DEFAULT uuidv7(),
    course_id      uuid          NOT NULL,
    scheme_id      uuid          NOT NULL,
    key            text          NOT NULL,
    name           text          NOT NULL,
    grp            grade_group   NOT NULL,
    weight         numeric(5,2)  NOT NULL,
    position       integer       NOT NULL DEFAULT 0,
    citations      jsonb         NOT NULL DEFAULT '[]',
    filled_by_user boolean       NOT NULL DEFAULT false,
    created_at     timestamptz   NOT NULL DEFAULT now(),
    updated_at     timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT grade_components_pkey PRIMARY KEY (id),
    CONSTRAINT grade_components_scheme_fkey FOREIGN KEY (course_id, scheme_id) REFERENCES grade_schemes (course_id, id) ON DELETE CASCADE,
    CONSTRAINT grade_components_key_chk CHECK (key ~ '^[a-z][a-z0-9_]{0,39}$'),
    CONSTRAINT grade_components_name_chk CHECK (char_length(name) BETWEEN 1 AND 80),
    CONSTRAINT grade_components_weight_chk CHECK (weight BETWEEN 0 AND 100),
    CONSTRAINT grade_components_citations_chk CHECK (jsonb_typeof(citations) = 'array'),
    CONSTRAINT grade_components_scheme_key UNIQUE (scheme_id, key),
    CONSTRAINT grade_components_course_id_key UNIQUE (course_id, id)
);
CREATE INDEX grade_components_course_scheme_idx ON grade_components (course_id, scheme_id, position);

-- grade_rules -----------------------------------------------------------------------------------------------------------------
CREATE TABLE grade_rules (
    id             uuid            NOT NULL DEFAULT uuidv7(),
    course_id      uuid            NOT NULL,
    scheme_id      uuid            NOT NULL,
    kind           grade_rule_kind NOT NULL,
    applicable     boolean         NOT NULL DEFAULT true,
    params         jsonb           NOT NULL DEFAULT '{}',
    source_quote   text,
    source_page    integer,
    filled_by_user boolean         NOT NULL DEFAULT false,
    created_at     timestamptz     NOT NULL DEFAULT now(),
    updated_at     timestamptz     NOT NULL DEFAULT now(),
    CONSTRAINT grade_rules_pkey PRIMARY KEY (id),
    CONSTRAINT grade_rules_scheme_fkey FOREIGN KEY (course_id, scheme_id) REFERENCES grade_schemes (course_id, id) ON DELETE CASCADE,
    CONSTRAINT grade_rules_params_chk CHECK (jsonb_typeof(params) = 'object'),
    CONSTRAINT grade_rules_page_chk CHECK (source_page IS NULL OR source_page >= 1)
);
CREATE UNIQUE INDEX grade_rules_scheme_kind_key ON grade_rules (scheme_id, kind) WHERE kind <> 'FAIL_CONDITION';
CREATE INDEX grade_rules_course_scheme_idx ON grade_rules (course_id, scheme_id);

-- grade_items -----------------------------------------------------------------------------------------------------------------
CREATE TABLE grade_items (
    id            uuid              NOT NULL DEFAULT uuidv7(),
    course_id     uuid              NOT NULL,
    scheme_id     uuid              NOT NULL,
    component_id  uuid              NOT NULL,
    key           text              NOT NULL,
    name          text              NOT NULL,
    max_score     numeric(5,2)      NOT NULL DEFAULT 10,
    source        grade_item_source NOT NULL DEFAULT 'MANUAL',
    exam_id       uuid,
    assignment_id uuid,
    published_at  timestamptz,
    position      integer           NOT NULL DEFAULT 0,
    created_at    timestamptz       NOT NULL DEFAULT now(),
    updated_at    timestamptz       NOT NULL DEFAULT now(),
    CONSTRAINT grade_items_pkey PRIMARY KEY (id),
    CONSTRAINT grade_items_scheme_fkey FOREIGN KEY (course_id, scheme_id) REFERENCES grade_schemes (course_id, id) ON DELETE CASCADE,
    CONSTRAINT grade_items_component_fkey FOREIGN KEY (course_id, component_id) REFERENCES grade_components (course_id, id) ON DELETE CASCADE,
    CONSTRAINT grade_items_exam_fkey FOREIGN KEY (course_id, exam_id) REFERENCES exams (course_id, id),
    CONSTRAINT grade_items_key_chk CHECK (key ~ '^[a-z][a-z0-9_]{0,39}$'),
    CONSTRAINT grade_items_name_chk CHECK (char_length(name) BETWEEN 1 AND 80),
    CONSTRAINT grade_items_max_chk CHECK (max_score > 0),
    CONSTRAINT grade_items_exam_chk CHECK ((source = 'EXAM') = (exam_id IS NOT NULL)),
    CONSTRAINT grade_items_assignment_chk CHECK ((source = 'ASSIGNMENT') = (assignment_id IS NOT NULL)),
    CONSTRAINT grade_items_scheme_key UNIQUE (scheme_id, key),
    CONSTRAINT grade_items_scheme_exam_key UNIQUE (scheme_id, exam_id),
    CONSTRAINT grade_items_course_id_key UNIQUE (course_id, id)
);
CREATE INDEX grade_items_course_scheme_idx ON grade_items (course_id, scheme_id, position);
CREATE INDEX grade_items_course_exam_idx ON grade_items (course_id, exam_id) WHERE exam_id IS NOT NULL;

-- grade_entries ---------------------------------------------------------------------------------------------------------------
CREATE TABLE grade_entries (
    id         uuid               NOT NULL DEFAULT uuidv7(),
    course_id  uuid               NOT NULL,
    item_id    uuid               NOT NULL,
    student_id uuid               NOT NULL,
    score      numeric(5,2)       NOT NULL,
    source     grade_entry_source NOT NULL DEFAULT 'MANUAL',
    version    integer            NOT NULL DEFAULT 1,
    entered_by uuid               NOT NULL,
    created_at timestamptz        NOT NULL DEFAULT now(),
    updated_at timestamptz        NOT NULL DEFAULT now(),
    CONSTRAINT grade_entries_pkey PRIMARY KEY (id),
    CONSTRAINT grade_entries_item_fkey FOREIGN KEY (course_id, item_id) REFERENCES grade_items (course_id, id) ON DELETE CASCADE,
    CONSTRAINT grade_entries_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES enrollments (course_id, user_id),
    CONSTRAINT grade_entries_entered_by_fkey FOREIGN KEY (entered_by) REFERENCES users (id),
    CONSTRAINT grade_entries_score_chk CHECK (score >= 0),
    CONSTRAINT grade_entries_version_chk CHECK (version >= 1),
    CONSTRAINT grade_entries_item_student_key UNIQUE (item_id, student_id)
);
CREATE INDEX grade_entries_course_student_idx ON grade_entries (course_id, student_id);
CREATE INDEX grade_entries_course_item_idx ON grade_entries (course_id, item_id);

-- gradebooks (một dòng mỗi lớp; tạo trong giao dịch xác nhận công thức đầu tiên — TLR-6) -------------------------------------------
CREATE TABLE gradebooks (
    course_id        uuid             NOT NULL,
    status           gradebook_status NOT NULL DEFAULT 'OPEN',
    snapshot_version integer          NOT NULL DEFAULT 0,
    unlock_reason    text,
    unlocked_by      uuid,
    unlocked_at      timestamptz,
    version          integer          NOT NULL DEFAULT 1,
    created_at       timestamptz      NOT NULL DEFAULT now(),
    updated_at       timestamptz      NOT NULL DEFAULT now(),
    CONSTRAINT gradebooks_pkey PRIMARY KEY (course_id),
    CONSTRAINT gradebooks_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT gradebooks_unlocked_by_fkey FOREIGN KEY (unlocked_by) REFERENCES users (id),
    CONSTRAINT gradebooks_snapshot_chk CHECK (snapshot_version >= 0 AND (status = 'OPEN' OR snapshot_version >= 1)),
    CONSTRAINT gradebooks_unlock_chk CHECK (unlock_reason IS NULL OR char_length(unlock_reason) BETWEEN 10 AND 500),
    CONSTRAINT gradebooks_version_chk CHECK (version >= 1)
);

-- final_grade_snapshots (bất biến) --------------------------------------------------------------------------------------------
CREATE TABLE final_grade_snapshots (
    id            uuid         NOT NULL DEFAULT uuidv7(),
    course_id     uuid         NOT NULL,
    student_id    uuid         NOT NULL,
    version       integer      NOT NULL,
    scheme_id     uuid         NOT NULL,
    breakdown     jsonb        NOT NULL,
    numeric_grade numeric(5,2) NOT NULL,
    letter        text         NOT NULL,
    failed_reason text,
    locked_at     timestamptz  NOT NULL DEFAULT now(),
    locked_by     uuid         NOT NULL,
    unlock_reason text,
    created_at    timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT final_grade_snapshots_pkey PRIMARY KEY (id),
    CONSTRAINT final_grade_snapshots_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES enrollments (course_id, user_id),
    CONSTRAINT final_grade_snapshots_scheme_fkey FOREIGN KEY (course_id, scheme_id) REFERENCES grade_schemes (course_id, id),
    CONSTRAINT final_grade_snapshots_locked_by_fkey FOREIGN KEY (locked_by) REFERENCES users (id),
    CONSTRAINT final_grade_snapshots_version_chk CHECK (version >= 1),
    CONSTRAINT final_grade_snapshots_breakdown_chk CHECK (jsonb_typeof(breakdown) = 'object'),
    CONSTRAINT final_grade_snapshots_grade_chk CHECK (numeric_grade BETWEEN 0 AND 10),
    CONSTRAINT final_grade_snapshots_student_version_key UNIQUE (course_id, student_id, version)
);
CREATE INDEX final_grade_snapshots_version_idx ON final_grade_snapshots (course_id, version, student_id);

-- +goose StatementBegin
CREATE FUNCTION final_grade_snapshots_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'snapshot is immutable';
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER final_grade_snapshots_immutable_row BEFORE UPDATE OR DELETE ON final_grade_snapshots
    FOR EACH ROW EXECUTE FUNCTION final_grade_snapshots_immutable();
CREATE TRIGGER final_grade_snapshots_immutable_truncate BEFORE TRUNCATE ON final_grade_snapshots
    FOR EACH STATEMENT EXECUTE FUNCTION final_grade_snapshots_immutable();

CREATE TRIGGER grade_schemes_set_updated_at BEFORE UPDATE ON grade_schemes FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER grade_components_set_updated_at BEFORE UPDATE ON grade_components FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER grade_rules_set_updated_at BEFORE UPDATE ON grade_rules FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER grade_items_set_updated_at BEFORE UPDATE ON grade_items FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER grade_entries_set_updated_at BEFORE UPDATE ON grade_entries FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER gradebooks_set_updated_at BEFORE UPDATE ON gradebooks FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS final_grade_snapshots_immutable_truncate ON final_grade_snapshots;
DROP TRIGGER IF EXISTS final_grade_snapshots_immutable_row ON final_grade_snapshots;
DROP TABLE IF EXISTS final_grade_snapshots;
DROP FUNCTION IF EXISTS final_grade_snapshots_immutable();
DROP TABLE IF EXISTS gradebooks;
DROP TABLE IF EXISTS grade_entries;
DROP TABLE IF EXISTS grade_items;
DROP TABLE IF EXISTS grade_rules;
DROP TABLE IF EXISTS grade_components;
DROP TABLE IF EXISTS grade_schemes;
DROP TYPE IF EXISTS gradebook_status;
DROP TYPE IF EXISTS grade_entry_source;
DROP TYPE IF EXISTS grade_item_source;
DROP TYPE IF EXISTS grade_rule_kind;
DROP TYPE IF EXISTS grade_group;
DROP TYPE IF EXISTS grade_scheme_status;

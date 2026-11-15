-- Luyện đề, thi thử, nhập kết quả Forms (SRS FEAT-practice mục 5, sprint 10 / P9). Tạo MỘT lần ở dạng cuối (luật 6); `00018` dành cho sprint 9.5 (chỉ mục).
-- Mọi bảng có `course_id`; mọi FK tới bảng thuộc lớp là FK PHỨC HỢP (course_id, …) như FEAT-weekly-exam 5.16 để DB chặn trộn dữ liệu giữa các lớp.
-- Không tạo `practice_exams` / `assignment_id` (P-10-2): ma trận lưu ở `practice_attempts.matrix`; QUIZ tính điểm = bài thi của PE.

-- +goose Up
CREATE TYPE practice_mode AS ENUM ('TOPIC', 'MOCK');
CREATE TYPE practice_status AS ENUM ('IN_PROGRESS', 'SUBMITTED', 'EXPIRED');

-- practice_attempts: một lượt luyện (chủ đề) hoặc thi thử. `deadline_at` do MÁY CHỦ tính (chỉ MOCK).
CREATE TABLE practice_attempts (
    id            uuid            NOT NULL DEFAULT uuidv7(),
    course_id     uuid            NOT NULL,
    student_id    uuid            NOT NULL,
    mode          practice_mode   NOT NULL,
    topic         text,
    status        practice_status NOT NULL DEFAULT 'IN_PROGRESS',
    started_at    timestamptz     NOT NULL DEFAULT now(),
    deadline_at   timestamptz,
    submitted_at  timestamptz,
    item_count    integer         NOT NULL,
    current_index integer         NOT NULL DEFAULT 0,
    correct_count integer,
    score         numeric(3,1),
    matrix        jsonb,
    notes         jsonb           NOT NULL DEFAULT '[]',
    tab_id        text,
    version       integer         NOT NULL DEFAULT 1,
    created_at    timestamptz     NOT NULL DEFAULT now(),
    updated_at    timestamptz     NOT NULL DEFAULT now(),
    CONSTRAINT practice_attempts_pkey PRIMARY KEY (id),
    CONSTRAINT practice_attempts_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES enrollments (course_id, user_id),
    CONSTRAINT practice_attempts_deadline_chk CHECK ((mode = 'MOCK') = (deadline_at IS NOT NULL)),
    CONSTRAINT practice_attempts_topic_chk CHECK (mode = 'MOCK' OR topic IS NOT NULL),
    CONSTRAINT practice_attempts_count_chk CHECK (item_count BETWEEN 1 AND 50),
    CONSTRAINT practice_attempts_index_chk CHECK (current_index >= 0 AND current_index < item_count),
    CONSTRAINT practice_attempts_closed_chk CHECK ((status = 'IN_PROGRESS') = (submitted_at IS NULL) AND (status = 'IN_PROGRESS') = (correct_count IS NULL)),
    CONSTRAINT practice_attempts_correct_chk CHECK (correct_count IS NULL OR correct_count BETWEEN 0 AND item_count),
    CONSTRAINT practice_attempts_score_chk CHECK (score IS NULL OR score BETWEEN 0 AND 10),
    CONSTRAINT practice_attempts_notes_chk CHECK (jsonb_typeof(notes) = 'array'),
    CONSTRAINT practice_attempts_version_chk CHECK (version >= 1)
);
CREATE UNIQUE INDEX practice_attempts_course_id_key ON practice_attempts (course_id, id);
-- Một lượt dở cho mỗi (sinh viên, lớp, chế độ, chủ đề): `start` lần hai trả lại lượt đó (MOCK: topic rỗng → một lượt MOCK dở mỗi lớp).
CREATE UNIQUE INDEX practice_attempts_open_key ON practice_attempts (course_id, student_id, mode, COALESCE(topic, '')) WHERE status = 'IN_PROGRESS';
CREATE INDEX practice_attempts_student_idx ON practice_attempts (course_id, student_id, started_at DESC, id DESC);
CREATE INDEX practice_attempts_due_idx ON practice_attempts (status, deadline_at) WHERE status = 'IN_PROGRESS' AND mode = 'MOCK';

-- practice_items: câu của lượt. `answer_key` + `question` là bản CHỤP lúc dựng đề (sửa câu giữa chừng không làm lệch lượt); chỉ phía máy chủ, DTO sinh viên là struct riêng.
CREATE TABLE practice_items (
    id               uuid    NOT NULL DEFAULT uuidv7(),
    course_id        uuid    NOT NULL,
    attempt_id       uuid    NOT NULL,
    question_id      uuid    NOT NULL,
    position         integer NOT NULL,
    option_order     jsonb   NOT NULL DEFAULT '[]',
    answer_key       jsonb   NOT NULL,
    question         jsonb   NOT NULL,
    question_version integer NOT NULL,
    CONSTRAINT practice_items_pkey PRIMARY KEY (id),
    CONSTRAINT practice_items_attempt_fkey FOREIGN KEY (course_id, attempt_id) REFERENCES practice_attempts (course_id, id) ON DELETE CASCADE,
    CONSTRAINT practice_items_question_fkey FOREIGN KEY (course_id, question_id) REFERENCES question_bank (course_id, id),
    CONSTRAINT practice_items_position_chk CHECK (position >= 0),
    CONSTRAINT practice_items_order_chk CHECK (jsonb_typeof(option_order) = 'array')
);
CREATE UNIQUE INDEX practice_items_course_id_key ON practice_items (course_id, id);
CREATE UNIQUE INDEX practice_items_attempt_position_key ON practice_items (attempt_id, position);
CREATE INDEX practice_items_question_idx ON practice_items (course_id, question_id);

-- practice_answers: câu trả lời (riêng của sinh viên; giảng viên / TA không đọc). `locked`: TOPIC khoá đáp án ngay khi có phản hồi.
CREATE TABLE practice_answers (
    id          uuid        NOT NULL DEFAULT uuidv7(),
    course_id   uuid        NOT NULL,
    attempt_id  uuid        NOT NULL,
    item_id     uuid        NOT NULL,
    answer      jsonb       NOT NULL,
    is_correct  boolean,
    answered_at timestamptz NOT NULL DEFAULT now(),
    locked      boolean     NOT NULL DEFAULT false,
    version     integer     NOT NULL DEFAULT 1,
    CONSTRAINT practice_answers_pkey PRIMARY KEY (id),
    CONSTRAINT practice_answers_item_key UNIQUE (item_id),
    CONSTRAINT practice_answers_attempt_fkey FOREIGN KEY (course_id, attempt_id) REFERENCES practice_attempts (course_id, id) ON DELETE CASCADE,
    CONSTRAINT practice_answers_item_fkey FOREIGN KEY (course_id, item_id) REFERENCES practice_items (course_id, id) ON DELETE CASCADE,
    CONSTRAINT practice_answers_version_chk CHECK (version >= 1)
);
CREATE INDEX practice_answers_attempt_idx ON practice_answers (course_id, attempt_id);

-- Nhập kết quả Form trắc nghiệm (US-P9-11): một lô cho mỗi (bài chấm, tệp). Tệp gốc ở platform/blob ≤ 90 ngày; `identifier_masked` không lưu email / MSSV đầy đủ.
CREATE TABLE forms_import_batches (
    id         uuid        NOT NULL DEFAULT uuidv7(),
    course_id  uuid        NOT NULL,
    item_id    uuid        NOT NULL,
    file_hash  text        NOT NULL,
    blob_key   text        NOT NULL,
    mapping    jsonb       NOT NULL,
    matched    integer     NOT NULL DEFAULT 0,
    unmatched  integer     NOT NULL DEFAULT 0,
    created_by uuid        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT forms_import_batches_pkey PRIMARY KEY (id),
    CONSTRAINT forms_import_batches_item_fkey FOREIGN KEY (course_id, item_id) REFERENCES grade_items (course_id, id),
    CONSTRAINT forms_import_batches_created_by_fkey FOREIGN KEY (created_by) REFERENCES users (id),
    CONSTRAINT forms_import_batches_hash_chk CHECK (file_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT forms_import_batches_counts_chk CHECK (matched >= 0 AND unmatched >= 0)
);
CREATE UNIQUE INDEX forms_import_batches_course_id_key ON forms_import_batches (course_id, id);
CREATE UNIQUE INDEX forms_import_batches_file_key ON forms_import_batches (course_id, item_id, file_hash);

CREATE TABLE forms_import_rows (
    course_id         uuid         NOT NULL,
    batch_id          uuid         NOT NULL,
    row_no            integer      NOT NULL,
    student_id        uuid,
    identifier_masked text         NOT NULL,
    correct           integer      NOT NULL,
    total             integer      NOT NULL,
    score             numeric(5,2) NOT NULL,
    CONSTRAINT forms_import_rows_pkey PRIMARY KEY (batch_id, row_no),
    CONSTRAINT forms_import_rows_batch_fkey FOREIGN KEY (course_id, batch_id) REFERENCES forms_import_batches (course_id, id) ON DELETE CASCADE,
    CONSTRAINT forms_import_rows_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES enrollments (course_id, user_id),
    CONSTRAINT forms_import_rows_counts_chk CHECK (correct >= 0 AND total >= correct),
    CONSTRAINT forms_import_rows_score_chk CHECK (score >= 0)
);

-- +goose Down
DROP TABLE IF EXISTS forms_import_rows;
DROP TABLE IF EXISTS forms_import_batches;
DROP TABLE IF EXISTS practice_answers;
DROP TABLE IF EXISTS practice_items;
DROP TABLE IF EXISTS practice_attempts;
DROP TYPE IF EXISTS practice_status;
DROP TYPE IF EXISTS practice_mode;

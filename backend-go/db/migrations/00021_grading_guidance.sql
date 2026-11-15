-- Harness học từ lần giảng viên sửa (SRS FEAT-grading-guidance mục 5, D61, D62, sprint 10.6). Ngoại lệ có chủ ý cho ALTER: yêu cầu mới sau 00017 / 00019 (tiền lệ 00015, 00020).
-- Không bảng nào lưu văn bản bài làm hay đầu vào / đầu ra thô của LLM; ca hồi quy theo tham chiếu.

-- +goose Up
-- 5.1 lưu ý chấm
CREATE TYPE guidance_item_status AS ENUM ('DRAFT', 'APPROVED', 'REJECTED', 'RETIRED');
CREATE TABLE grading_guidance_items (
    id                  uuid                 PRIMARY KEY DEFAULT uuidv7(),
    course_id           uuid                 NOT NULL,
    assignment_id       uuid                 NOT NULL,
    criterion_id        text                 NOT NULL,
    text                text                 NOT NULL CHECK (char_length(text) BETWEEN 1 AND 400),
    text_original       text                 NULL,
    direction           text                 NOT NULL CHECK (direction IN ('STRICTER', 'LENIENT', 'CLARIFY')),
    support             smallint             NOT NULL CHECK (support >= 0),
    source_result_ids   uuid[]               NOT NULL DEFAULT '{}',          -- id kết quả chấm, KHÔNG id sinh viên
    needs_recheck       boolean              NOT NULL DEFAULT false,
    status              guidance_item_status NOT NULL DEFAULT 'DRAFT',
    pending_retire      boolean              NOT NULL DEFAULT false,
    approved_in_version smallint             NULL,
    retired_in_version  smallint             NULL,
    origin_job          uuid                 NULL,
    decided_by          uuid                 NULL REFERENCES users (id),
    decided_at          timestamptz          NULL,
    version             integer              NOT NULL DEFAULT 1,
    created_at          timestamptz          NOT NULL DEFAULT now(),
    updated_at          timestamptz          NOT NULL DEFAULT now(),
    -- TLR-3: khớp SRS 4.2.14 «xoá lớp → CASCADE»
    FOREIGN KEY (course_id, assignment_id) REFERENCES assignments (course_id, id) ON DELETE CASCADE,
    CHECK ((status IN ('APPROVED', 'REJECTED', 'RETIRED')) = (decided_by IS NOT NULL)),
    CHECK (retired_in_version IS NULL OR (status = 'RETIRED' AND approved_in_version IS NOT NULL AND retired_in_version > approved_in_version)),
    CHECK (approved_in_version IS NULL OR status IN ('APPROVED', 'RETIRED'))
);
CREATE INDEX ggi_assignment_idx ON grading_guidance_items (course_id, assignment_id, status);
CREATE TRIGGER ggi_set_updated_at BEFORE UPDATE ON grading_guidance_items FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 5.2 các phiên bản (v0 ngầm định, không lưu)
CREATE TABLE grading_guidance_versions (
    course_id     uuid        NOT NULL,
    assignment_id uuid        NOT NULL,
    version       smallint    NOT NULL CHECK (version >= 1),
    item_count    smallint    NOT NULL,
    created_by    uuid        NOT NULL REFERENCES users (id),
    created_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (course_id, assignment_id, version),
    FOREIGN KEY (course_id, assignment_id) REFERENCES assignments (course_id, id) ON DELETE CASCADE
);

-- 5.3 ghim + bất biến
ALTER TABLE assignments ADD COLUMN guidance_version smallint NOT NULL DEFAULT 0;
ALTER TABLE grading_results
    ADD COLUMN guidance_version smallint    NOT NULL DEFAULT 0,
    ADD COLUMN ai_comments      jsonb       NULL,
    ADD COLUMN first_reviewed_at timestamptz NULL,   -- P-106-5: ghi MỘT lần ở lần duyệt đầu (coalesce), không bị ReviewApplyEdit xoá
    ADD COLUMN first_reviewed_by uuid        NULL REFERENCES users (id),
    ADD CONSTRAINT grading_results_first_reviewed_chk CHECK ((first_reviewed_at IS NULL) = (first_reviewed_by IS NULL));
-- Bài đã duyệt TRƯỚC 10.6 (REVIEWED / PUBLISHED có reviewed_at): lấy dấu duyệt hiện có làm lần duyệt đầu.
UPDATE grading_results SET first_reviewed_at = reviewed_at, first_reviewed_by = reviewed_by WHERE reviewed_at IS NOT NULL AND reviewed_by IS NOT NULL;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION grading_results_ai_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.ai_scores IS NOT NULL AND NEW.ai_scores IS DISTINCT FROM OLD.ai_scores THEN
        RAISE EXCEPTION 'ai draft is immutable' USING ERRCODE = 'P0001';
    END IF;
    IF OLD.ai_total_before_late IS NOT NULL AND NEW.ai_total_before_late IS DISTINCT FROM OLD.ai_total_before_late THEN
        RAISE EXCEPTION 'ai draft is immutable' USING ERRCODE = 'P0001';
    END IF;
    -- SRS 5.3 / 4.2.1: sau DONE cấm đổi ai_comments, guidance_version (thử lại trước DONE được ghim lại)
    IF OLD.run_state = 'DONE' AND (NEW.ai_comments IS DISTINCT FROM OLD.ai_comments OR NEW.guidance_version IS DISTINCT FROM OLD.guidance_version) THEN
        RAISE EXCEPTION 'ai draft is immutable' USING ERRCODE = 'P0001';
    END IF;
    -- P-106-5: dấu «giảng viên đã duyệt» ghi một lần
    IF OLD.first_reviewed_at IS NOT NULL AND (NEW.first_reviewed_at IS DISTINCT FROM OLD.first_reviewed_at OR NEW.first_reviewed_by IS DISTINCT FROM OLD.first_reviewed_by) THEN
        RAISE EXCEPTION 'ai draft is immutable' USING ERRCODE = 'P0001';
    END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd

-- 5.4 cờ theo lớp (bảng riêng của tính năng này, khác course_features của PR: SRS thắng)
CREATE TABLE course_guidance_flags (
    course_id  uuid        NOT NULL REFERENCES courses (id) ON DELETE CASCADE,
    feature    text        NOT NULL CHECK (feature IN ('GRADING_GUIDANCE', 'REGRESSION_CAPTURE', 'THREAD_SUGGEST', 'QUESTION_HINTS')),
    enabled    boolean     NOT NULL DEFAULT false,
    version    integer     NOT NULL DEFAULT 1,
    updated_by uuid        REFERENCES users (id),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (course_id, feature)
);
CREATE TRIGGER cgf_set_updated_at BEFORE UPDATE ON course_guidance_flags FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 5.5 ca hồi quy (theo tham chiếu, không văn bản)
CREATE TABLE grading_regression_cases (
    id           uuid        PRIMARY KEY DEFAULT uuidv7(),
    course_id    uuid        NOT NULL REFERENCES courses (id) ON DELETE CASCADE,
    kind         text        NOT NULL CHECK (kind IN ('GRADING_EDIT', 'GRADING_APPEAL', 'THREAD_CORRECTED', 'THREAD_REJECTED')),
    source_id    uuid        NOT NULL,
    criterion_id text        NULL,
    expected     jsonb       NOT NULL CHECK (jsonb_typeof(expected) = 'object'),   -- chỉ số / enum; service kiểm không có chuỗi dài
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX grc_unique ON grading_regression_cases (course_id, kind, source_id, coalesce(criterion_id, ''));
CREATE INDEX grc_course_idx ON grading_regression_cases (course_id, kind, created_at);
CREATE TRIGGER grc_set_updated_at BEFORE UPDATE ON grading_regression_cases FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Xoá kết quả chấm / bài đăng (kể cả cascade khi xoá bài nộp, sinh viên, lớp) thì xoá ca hồi quy tham chiếu nó: không còn tham chiếu mồ côi (US-GD-07 AC7, US-GD-11).
-- +goose StatementBegin
CREATE FUNCTION grading_regression_cases_gc() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    DELETE FROM grading_regression_cases WHERE course_id = OLD.course_id AND source_id = OLD.id;
    RETURN OLD;
END $$;
-- +goose StatementEnd
CREATE TRIGGER grading_results_gc_cases AFTER DELETE ON grading_results FOR EACH ROW EXECUTE FUNCTION grading_regression_cases_gc();
CREATE TRIGGER forum_posts_gc_cases AFTER DELETE ON forum_posts FOR EACH ROW EXECUTE FUNCTION grading_regression_cases_gc();

-- 5.6 lý do loại câu hỏi
ALTER TABLE question_bank
    ADD COLUMN reject_reason text NULL
        CHECK (reject_reason IN ('MO_HO', 'SAI_DAP_AN', 'TRUNG_LAP', 'LAC_DE', 'KHO_DE_BAT_THUONG', 'SAI_DINH_DANG', 'UNSPECIFIED')),
    ADD CONSTRAINT question_bank_reject_reason_chk CHECK (reject_reason IS NULL OR review_status = 'REJECTED');
CREATE INDEX question_bank_rejected_idx ON question_bank (course_id, reviewed_at DESC)
    WHERE review_status = 'REJECTED' AND reject_reason IS NOT NULL AND reject_reason <> 'UNSPECIFIED';

-- +goose Down
DROP TRIGGER IF EXISTS forum_posts_gc_cases ON forum_posts;
DROP TRIGGER IF EXISTS grading_results_gc_cases ON grading_results;
DROP FUNCTION IF EXISTS grading_regression_cases_gc();
DROP INDEX IF EXISTS question_bank_rejected_idx;
ALTER TABLE question_bank DROP CONSTRAINT question_bank_reject_reason_chk, DROP COLUMN reject_reason;
DROP TABLE IF EXISTS grading_regression_cases;
DROP TABLE IF EXISTS course_guidance_flags;
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION grading_results_ai_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
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
ALTER TABLE grading_results DROP CONSTRAINT grading_results_first_reviewed_chk, DROP COLUMN first_reviewed_by, DROP COLUMN first_reviewed_at,
    DROP COLUMN ai_comments, DROP COLUMN guidance_version;
ALTER TABLE assignments DROP COLUMN guidance_version;
DROP TABLE IF EXISTS grading_guidance_versions;
DROP TABLE IF EXISTS grading_guidance_items;
DROP TYPE IF EXISTS guidance_item_status;

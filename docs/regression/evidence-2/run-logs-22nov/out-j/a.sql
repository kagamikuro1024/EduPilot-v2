--
-- PostgreSQL database dump
--

\restrict s4CmlBMXb9xsWfa7AMqCEGPrEZPQcZobkxKoyXHRzmjzSmCNXLPyZg9vghCiERw

-- Dumped from database version 18.6 (Debian 18.6-1.pgdg12+2)
-- Dumped by pg_dump version 18.6 (Debian 18.6-1.pgdg12+2)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: vector; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS vector WITH SCHEMA public;


--
-- Name: EXTENSION vector; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION vector IS 'vector data type and ivfflat and hnsw access methods';


--
-- Name: appeal_status; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.appeal_status AS ENUM (
    'OPEN',
    'UPHELD',
    'ADJUSTED'
);


ALTER TYPE public.appeal_status OWNER TO edupilot;

--
-- Name: attempt_status; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.attempt_status AS ENUM (
    'IN_PROGRESS',
    'GRADING',
    'GRADED'
);


ALTER TYPE public.attempt_status OWNER TO edupilot;

--
-- Name: attempt_submit_reason; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.attempt_submit_reason AS ENUM (
    'MANUAL',
    'TIMEOUT',
    'CLOSED'
);


ALTER TYPE public.attempt_submit_reason OWNER TO edupilot;

--
-- Name: attendance_status; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.attendance_status AS ENUM (
    'PRESENT',
    'LATE',
    'EXCUSED',
    'ABSENT'
);


ALTER TYPE public.attendance_status OWNER TO edupilot;

--
-- Name: auth_token_kind; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.auth_token_kind AS ENUM (
    'VERIFY_EMAIL',
    'RESET_PASSWORD',
    'INVITE'
);


ALTER TYPE public.auth_token_kind OWNER TO edupilot;

--
-- Name: calendar_event_type; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.calendar_event_type AS ENUM (
    'EXAM',
    'OTHER'
);


ALTER TYPE public.calendar_event_type OWNER TO edupilot;

--
-- Name: chat_channel; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.chat_channel AS ENUM (
    'PRIVATE',
    'PUBLIC'
);


ALTER TYPE public.chat_channel OWNER TO edupilot;

--
-- Name: chat_feedback; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.chat_feedback AS ENUM (
    'HELPFUL',
    'NOT_HELPFUL'
);


ALTER TYPE public.chat_feedback OWNER TO edupilot;

--
-- Name: chat_role; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.chat_role AS ENUM (
    'USER',
    'ASSISTANT'
);


ALTER TYPE public.chat_role OWNER TO edupilot;

--
-- Name: chat_stream_status; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.chat_stream_status AS ENUM (
    'STREAMING',
    'DONE',
    'FAILED',
    'CANCELLED'
);


ALTER TYPE public.chat_stream_status OWNER TO edupilot;

--
-- Name: checker_kind; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.checker_kind AS ENUM (
    'EXACT',
    'TOKENS',
    'FLOAT_EPS'
);


ALTER TYPE public.checker_kind OWNER TO edupilot;

--
-- Name: checkin_channel; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.checkin_channel AS ENUM (
    'WEB',
    'APP',
    'CODE'
);


ALTER TYPE public.checkin_channel OWNER TO edupilot;

--
-- Name: chunk_audience; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.chunk_audience AS ENUM (
    'ALL',
    'STAFF',
    'GRADING'
);


ALTER TYPE public.chunk_audience OWNER TO edupilot;

--
-- Name: course_status; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.course_status AS ENUM (
    'ACTIVE',
    'ARCHIVED'
);


ALTER TYPE public.course_status OWNER TO edupilot;

--
-- Name: device_kind; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.device_kind AS ENUM (
    'WEB_KEY',
    'ANDROID_KEY'
);


ALTER TYPE public.device_kind OWNER TO edupilot;

--
-- Name: document_status; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.document_status AS ENUM (
    'QUEUED',
    'PROCESSING',
    'READY',
    'FAILED'
);


ALTER TYPE public.document_status OWNER TO edupilot;

--
-- Name: document_type; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.document_type AS ENUM (
    'LECTURE',
    'COURSE_POLICY',
    'EXAM_PAPER',
    'ANSWER_KEY',
    'OTHER'
);


ALTER TYPE public.document_type OWNER TO edupilot;

--
-- Name: enrollment_joined_via; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.enrollment_joined_via AS ENUM (
    'ADMIN',
    'ROSTER',
    'CODE'
);


ALTER TYPE public.enrollment_joined_via OWNER TO edupilot;

--
-- Name: enrollment_role; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.enrollment_role AS ENUM (
    'TEACHER',
    'TA',
    'STUDENT'
);


ALTER TYPE public.enrollment_role OWNER TO edupilot;

--
-- Name: enrollment_status; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.enrollment_status AS ENUM (
    'PENDING',
    'ACTIVE',
    'REMOVED'
);


ALTER TYPE public.enrollment_status OWNER TO edupilot;

--
-- Name: exam_event_type; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.exam_event_type AS ENUM (
    'TAB_HIDDEN',
    'TAB_VISIBLE',
    'PASTE',
    'OFFLINE',
    'ONLINE',
    'TAB_TAKEOVER',
    'CHAT_BLOCKED'
);


ALTER TYPE public.exam_event_type OWNER TO edupilot;

--
-- Name: exam_kind; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.exam_kind AS ENUM (
    'MCQ',
    'CODE',
    'MIXED'
);


ALTER TYPE public.exam_kind OWNER TO edupilot;

--
-- Name: exam_status; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.exam_status AS ENUM (
    'DRAFT',
    'SCHEDULED',
    'OPEN',
    'CLOSED',
    'PUBLISHED'
);


ALTER TYPE public.exam_status OWNER TO edupilot;

--
-- Name: grade_entry_source; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.grade_entry_source AS ENUM (
    'MANUAL',
    'AI_GRADED',
    'QUIZ',
    'IMPORT'
);


ALTER TYPE public.grade_entry_source OWNER TO edupilot;

--
-- Name: grade_group; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.grade_group AS ENUM (
    'QT',
    'CK'
);


ALTER TYPE public.grade_group OWNER TO edupilot;

--
-- Name: grade_item_source; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.grade_item_source AS ENUM (
    'MANUAL',
    'EXAM',
    'ASSIGNMENT'
);


ALTER TYPE public.grade_item_source OWNER TO edupilot;

--
-- Name: grade_rule_kind; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.grade_rule_kind AS ENUM (
    'BONUS',
    'PENALTY',
    'ROUNDING',
    'FAIL_CONDITION',
    'LETTER_SCALE',
    'MISSING_SCORE'
);


ALTER TYPE public.grade_rule_kind OWNER TO edupilot;

--
-- Name: grade_scheme_status; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.grade_scheme_status AS ENUM (
    'DRAFT',
    'CONFIRMED',
    'SUPERSEDED'
);


ALTER TYPE public.grade_scheme_status OWNER TO edupilot;

--
-- Name: gradebook_status; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.gradebook_status AS ENUM (
    'OPEN',
    'LOCKED'
);


ALTER TYPE public.gradebook_status OWNER TO edupilot;

--
-- Name: guidance_item_status; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.guidance_item_status AS ENUM (
    'DRAFT',
    'APPROVED',
    'REJECTED',
    'RETIRED'
);


ALTER TYPE public.guidance_item_status OWNER TO edupilot;

--
-- Name: job_status; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.job_status AS ENUM (
    'QUEUED',
    'RUNNING',
    'SUCCEEDED',
    'FAILED'
);


ALTER TYPE public.job_status OWNER TO edupilot;

--
-- Name: judge_verdict; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.judge_verdict AS ENUM (
    'AC',
    'WA',
    'TLE',
    'MLE',
    'RE',
    'CE',
    'OLE',
    'IE'
);


ALTER TYPE public.judge_verdict OWNER TO edupilot;

--
-- Name: login_outcome; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.login_outcome AS ENUM (
    'SUCCESS',
    'BAD_PASSWORD',
    'UNKNOWN_EMAIL',
    'THROTTLED',
    'LOCKED',
    'DISABLED'
);


ALTER TYPE public.login_outcome OWNER TO edupilot;

--
-- Name: mail_status; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.mail_status AS ENUM (
    'QUEUED',
    'SENT',
    'DEAD'
);


ALTER TYPE public.mail_status OWNER TO edupilot;

--
-- Name: multi_scoring; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.multi_scoring AS ENUM (
    'ALL_OR_NOTHING',
    'PARTIAL'
);


ALTER TYPE public.multi_scoring OWNER TO edupilot;

--
-- Name: observation_kind; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.observation_kind AS ENUM (
    'NOTE',
    'AI_SUMMARY'
);


ALTER TYPE public.observation_kind OWNER TO edupilot;

--
-- Name: participation_type; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.participation_type AS ENUM (
    'SPEAK',
    'BONUS',
    'PENALTY'
);


ALTER TYPE public.participation_type OWNER TO edupilot;

--
-- Name: pii_action; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.pii_action AS ENUM (
    'BLOCKED',
    'REDACTED',
    'SWITCHED',
    'MASKED'
);


ALTER TYPE public.pii_action OWNER TO edupilot;

--
-- Name: pii_kind; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.pii_kind AS ENUM (
    'MSSV',
    'EMAIL',
    'PHONE',
    'CCCD',
    'NAME',
    'PERSONAL_QUESTION',
    'OTHER_PERSON'
);


ALTER TYPE public.pii_kind OWNER TO edupilot;

--
-- Name: post_kind; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.post_kind AS ENUM (
    'AI',
    'HUMAN'
);


ALTER TYPE public.post_kind OWNER TO edupilot;

--
-- Name: post_report_reason; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.post_report_reason AS ENUM (
    'SPAM',
    'OFF_TOPIC',
    'PERSONAL_INFO',
    'OFFENSIVE',
    'WRONG_ANSWER',
    'OTHER'
);


ALTER TYPE public.post_report_reason OWNER TO edupilot;

--
-- Name: post_report_status; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.post_report_status AS ENUM (
    'OPEN',
    'DISMISSED',
    'ACTIONED'
);


ALTER TYPE public.post_report_status OWNER TO edupilot;

--
-- Name: post_verification; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.post_verification AS ENUM (
    'NONE',
    'PENDING',
    'VERIFIED',
    'CORRECTED',
    'REJECTED'
);


ALTER TYPE public.post_verification OWNER TO edupilot;

--
-- Name: practice_mode; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.practice_mode AS ENUM (
    'TOPIC',
    'MOCK'
);


ALTER TYPE public.practice_mode OWNER TO edupilot;

--
-- Name: practice_status; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.practice_status AS ENUM (
    'IN_PROGRESS',
    'SUBMITTED',
    'EXPIRED'
);


ALTER TYPE public.practice_status OWNER TO edupilot;

--
-- Name: question_difficulty; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.question_difficulty AS ENUM (
    'EASY',
    'MEDIUM',
    'HARD'
);


ALTER TYPE public.question_difficulty OWNER TO edupilot;

--
-- Name: question_origin; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.question_origin AS ENUM (
    'MANUAL',
    'AI_DRAFT',
    'EXTRACTED',
    'GENERATED'
);


ALTER TYPE public.question_origin OWNER TO edupilot;

--
-- Name: question_review_status; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.question_review_status AS ENUM (
    'DRAFT',
    'PENDING',
    'APPROVED',
    'REJECTED'
);


ALTER TYPE public.question_review_status OWNER TO edupilot;

--
-- Name: question_type; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.question_type AS ENUM (
    'MCQ_SINGLE',
    'MCQ_MULTI',
    'TRUE_FALSE',
    'CODE',
    'SHORT',
    'ESSAY'
);


ALTER TYPE public.question_type OWNER TO edupilot;

--
-- Name: reminder_source; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.reminder_source AS ENUM (
    'CLASS_SESSION',
    'WEEKLY_EXAM',
    'CALENDAR_EVENT',
    'ASSIGNMENT'
);


ALTER TYPE public.reminder_source OWNER TO edupilot;

--
-- Name: risk_level; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.risk_level AS ENUM (
    'NONE',
    'WATCH',
    'AT_RISK'
);


ALTER TYPE public.risk_level OWNER TO edupilot;

--
-- Name: similarity_review_state; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.similarity_review_state AS ENUM (
    'NEW',
    'CLEARED',
    'FOLLOW_UP'
);


ALTER TYPE public.similarity_review_state OWNER TO edupilot;

--
-- Name: submission_kind; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.submission_kind AS ENUM (
    'RUN',
    'SUBMIT'
);


ALTER TYPE public.submission_kind OWNER TO edupilot;

--
-- Name: submission_status; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.submission_status AS ENUM (
    'QUEUED',
    'RUNNING',
    'DONE',
    'SUPERSEDED',
    'ERROR'
);


ALTER TYPE public.submission_status OWNER TO edupilot;

--
-- Name: thread_ai_state; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.thread_ai_state AS ENUM (
    'PENDING',
    'ANSWERED',
    'SKIPPED'
);


ALTER TYPE public.thread_ai_state OWNER TO edupilot;

--
-- Name: thread_state; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.thread_state AS ENUM (
    'OPEN',
    'CLOSED'
);


ALTER TYPE public.thread_state OWNER TO edupilot;

--
-- Name: ticket_kind; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.ticket_kind AS ENUM (
    'QUESTION',
    'GRADE_APPEAL',
    'TEACHER_OUTREACH'
);


ALTER TYPE public.ticket_kind OWNER TO edupilot;

--
-- Name: ticket_origin; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.ticket_origin AS ENUM (
    'AUTO_LOW_CONFIDENCE',
    'AUTO_NO_FORMULA',
    'STUDENT_REQUEST',
    'STAFF_OUTREACH',
    'STUDENT_APPEAL'
);


ALTER TYPE public.ticket_origin OWNER TO edupilot;

--
-- Name: ticket_status; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.ticket_status AS ENUM (
    'OPEN',
    'CLAIMED',
    'ANSWERED',
    'CLOSED'
);


ALTER TYPE public.ticket_status OWNER TO edupilot;

--
-- Name: user_role; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.user_role AS ENUM (
    'ADMIN',
    'TEACHER',
    'TA',
    'STUDENT'
);


ALTER TYPE public.user_role OWNER TO edupilot;

--
-- Name: user_status; Type: TYPE; Schema: public; Owner: edupilot
--

CREATE TYPE public.user_status AS ENUM (
    'PENDING_VERIFICATION',
    'INVITED',
    'ACTIVE',
    'DISABLED'
);


ALTER TYPE public.user_status OWNER TO edupilot;

--
-- Name: audit_log_block_mutation(); Type: FUNCTION; Schema: public; Owner: edupilot
--

CREATE FUNCTION public.audit_log_block_mutation() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE EXCEPTION 'audit_log is append-only' USING ERRCODE = '42501';
END;
$$;


ALTER FUNCTION public.audit_log_block_mutation() OWNER TO edupilot;

--
-- Name: final_grade_snapshots_immutable(); Type: FUNCTION; Schema: public; Owner: edupilot
--

CREATE FUNCTION public.final_grade_snapshots_immutable() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE EXCEPTION 'snapshot is immutable';
END;
$$;


ALTER FUNCTION public.final_grade_snapshots_immutable() OWNER TO edupilot;

--
-- Name: forum_tags_valid(text[]); Type: FUNCTION; Schema: public; Owner: edupilot
--

CREATE FUNCTION public.forum_tags_valid(tags text[]) RETURNS boolean
    LANGUAGE sql IMMUTABLE
    AS $$
    SELECT cardinality(tags) <= 5 AND NOT EXISTS (SELECT 1 FROM unnest(tags) t WHERE t IS NULL OR char_length(t) NOT BETWEEN 1 AND 30)
$$;


ALTER FUNCTION public.forum_tags_valid(tags text[]) OWNER TO edupilot;

--
-- Name: grading_regression_cases_gc(); Type: FUNCTION; Schema: public; Owner: edupilot
--

CREATE FUNCTION public.grading_regression_cases_gc() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    DELETE FROM grading_regression_cases WHERE course_id = OLD.course_id AND source_id = OLD.id;
    RETURN OLD;
END $$;


ALTER FUNCTION public.grading_regression_cases_gc() OWNER TO edupilot;

--
-- Name: grading_results_ai_immutable(); Type: FUNCTION; Schema: public; Owner: edupilot
--

CREATE FUNCTION public.grading_results_ai_immutable() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
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


ALTER FUNCTION public.grading_results_ai_immutable() OWNER TO edupilot;

--
-- Name: obs_lat_bucket(integer); Type: FUNCTION; Schema: public; Owner: edupilot
--

CREATE FUNCTION public.obs_lat_bucket(ms integer) RETURNS smallint
    LANGUAGE sql IMMUTABLE PARALLEL SAFE
    AS $$
  SELECT least(139, floor(ln(greatest(ms, 1)::float8) / ln(1.1::float8))::int)::smallint
$$;


ALTER FUNCTION public.obs_lat_bucket(ms integer) OWNER TO edupilot;

--
-- Name: set_updated_at(); Type: FUNCTION; Schema: public; Owner: edupilot
--

CREATE FUNCTION public.set_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    -- clock_timestamp(): trong CÙNG transaction, now() đứng yên nên updated_at không lớn hơn created_at.
    NEW.updated_at := clock_timestamp();
    RETURN NEW;
END;
$$;


ALTER FUNCTION public.set_updated_at() OWNER TO edupilot;

--
-- Name: submissions_immutable(); Type: FUNCTION; Schema: public; Owner: edupilot
--

CREATE FUNCTION public.submissions_immutable() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF NEW.blob_key IS DISTINCT FROM OLD.blob_key OR NEW.sha256 IS DISTINCT FROM OLD.sha256 THEN
        RAISE EXCEPTION 'submission file is immutable' USING ERRCODE = 'P0001';
    END IF;
    IF OLD.extracted_text IS NOT NULL AND NEW.extracted_text IS DISTINCT FROM OLD.extracted_text THEN
        RAISE EXCEPTION 'submission text is immutable' USING ERRCODE = 'P0001';
    END IF;
    RETURN NEW;
END $$;


ALTER FUNCTION public.submissions_immutable() OWNER TO edupilot;

--
-- Name: vn_bigram_query(text); Type: FUNCTION; Schema: public; Owner: edupilot
--

CREATE FUNCTION public.vn_bigram_query(q text) RETURNS tsquery
    LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE
    AS $$
DECLARE
    w text[];
    parts text[] := '{}';
    i int;
BEGIN
    w := (SELECT coalesce(array_agg(x), '{}') FROM (
            SELECT x FROM regexp_split_to_table(regexp_replace(vn_fold(normalize(q, NFC)), '[^[:alnum:]]+', ' ', 'g'), ' ') AS x WHERE x <> '' LIMIT 40) s);
    IF cardinality(w) = 0 THEN
        RETURN ''::tsquery;
    ELSIF cardinality(w) = 1 THEN
        RETURN to_tsquery('simple', w[1]);
    END IF;
    FOR i IN 1 .. cardinality(w) - 1 LOOP
        parts := parts || (w[i] || ' <-> ' || w[i + 1]);
    END LOOP;
    RETURN to_tsquery('simple', array_to_string(parts, ' | '));
END
$$;


ALTER FUNCTION public.vn_bigram_query(q text) OWNER TO edupilot;

--
-- Name: vn_fold(text); Type: FUNCTION; Schema: public; Owner: edupilot
--

CREATE FUNCTION public.vn_fold(text) RETURNS text
    LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
    AS $_$ SELECT translate(lower($1), 'àáạảãâầấậẩẫăằắặẳẵèéẹẻẽêềếệểễìíịỉĩòóọỏõôồốộổỗơờớợởỡùúụủũưừứựửữỳýỵỷỹđ', 'aaaaaaaaaaaaaaaaaeeeeeeeeeeeiiiiiooooooooooooooooouuuuuuuuuuuyyyyyd') $_$;


ALTER FUNCTION public.vn_fold(text) OWNER TO edupilot;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: assignments; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.assignments (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    title text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    type text DEFAULT 'ESSAY'::text NOT NULL,
    max_score numeric(5,2) DEFAULT 10 NOT NULL,
    due_at timestamp with time zone NOT NULL,
    allow_late boolean DEFAULT false NOT NULL,
    late_penalty_per_day numeric(5,2) DEFAULT 0 NOT NULL,
    late_max_days smallint DEFAULT 0 NOT NULL,
    appeal_days smallint DEFAULT 7 NOT NULL,
    channels text[] DEFAULT ARRAY['IN_APP'::text] NOT NULL,
    submission_code text NOT NULL,
    reference_answer text,
    rubric jsonb DEFAULT '[]'::jsonb NOT NULL,
    teams_class_id text,
    teams_assignment_id text,
    published_at timestamp with time zone,
    created_by uuid NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    guidance_version smallint DEFAULT 0 NOT NULL,
    CONSTRAINT assignments_allow_late_chk CHECK (((allow_late = false) OR (late_max_days >= 1))),
    CONSTRAINT assignments_appeal_days_chk CHECK (((appeal_days >= 0) AND (appeal_days <= 30))),
    CONSTRAINT assignments_channels_chk CHECK (((channels <@ ARRAY['IN_APP'::text, 'UPLOAD'::text, 'IMAP'::text, 'TEAMS'::text]) AND (cardinality(channels) >= 1))),
    CONSTRAINT assignments_code_chk CHECK ((submission_code ~ '^[A-Z0-9]{2,12}$'::text)),
    CONSTRAINT assignments_late_max_days_chk CHECK (((late_max_days >= 0) AND (late_max_days <= 14))),
    CONSTRAINT assignments_late_penalty_chk CHECK ((late_penalty_per_day >= (0)::numeric)),
    CONSTRAINT assignments_max_score_chk CHECK (((max_score > (0)::numeric) AND (max_score <= (100)::numeric) AND ((max_score % 0.25) = (0)::numeric))),
    CONSTRAINT assignments_teams_chk CHECK (((('TEAMS'::text = ANY (channels)) = (teams_class_id IS NOT NULL)) AND ((teams_class_id IS NULL) = (teams_assignment_id IS NULL)))),
    CONSTRAINT assignments_title_chk CHECK (((length(btrim(title)) >= 1) AND (length(btrim(title)) <= 200))),
    CONSTRAINT assignments_type_chk CHECK ((type = ANY (ARRAY['ESSAY'::text, 'QUIZ'::text]))),
    CONSTRAINT assignments_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.assignments OWNER TO edupilot;

--
-- Name: attendance_checkins; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.attendance_checkins (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    session_id uuid NOT NULL,
    window_id uuid NOT NULL,
    student_id uuid NOT NULL,
    device_id uuid,
    channel public.checkin_channel NOT NULL,
    assurance smallint NOT NULL,
    token_age_ms integer NOT NULL,
    geo_distance_m integer,
    geo_accuracy_m integer,
    in_campus boolean,
    flags text[] DEFAULT '{}'::text[] NOT NULL,
    flags_cleared_by uuid,
    flags_cleared_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT attendance_checkins_age_chk CHECK ((token_age_ms >= 0)),
    CONSTRAINT attendance_checkins_assurance_chk CHECK (((assurance >= 1) AND (assurance <= 3))),
    CONSTRAINT attendance_checkins_flags_chk CHECK ((flags <@ ARRAY['DEVICE_SHARED'::text, 'DEVICE_NEW'::text, 'NO_DEVICE_KEY'::text, 'DEVICE_UNTRUSTED'::text, 'TOKEN_LATE'::text, 'GEO_OUT'::text, 'GEO_WEAK'::text, 'GEO_DENIED'::text, 'GEO_MOCK'::text, 'OFF_CAMPUS'::text, 'NOT_APP'::text, 'RECHECK_MISS'::text]))
);


ALTER TABLE public.attendance_checkins OWNER TO edupilot;

--
-- Name: attendance_records; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.attendance_records (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    session_id uuid NOT NULL,
    student_id uuid NOT NULL,
    status public.attendance_status NOT NULL,
    note text,
    marked_by uuid NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT attendance_records_note_chk CHECK ((char_length(note) <= 200)),
    CONSTRAINT attendance_records_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.attendance_records OWNER TO edupilot;

--
-- Name: audit_log; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.audit_log (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid,
    actor_id uuid,
    entity text NOT NULL,
    entity_id text NOT NULL,
    action text NOT NULL,
    before jsonb,
    after jsonb,
    trace_id text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.audit_log OWNER TO edupilot;

--
-- Name: auth_sessions; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.auth_sessions (
    id uuid DEFAULT uuidv7() NOT NULL,
    user_id uuid NOT NULL,
    refresh_hash character(64) NOT NULL,
    prev_refresh_hash character(64),
    user_agent text,
    device_label text,
    ip inet,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    rotated_at timestamp with time zone,
    last_used_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    absolute_expires_at timestamp with time zone NOT NULL,
    revoked_at timestamp with time zone,
    revoked_reason text,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT auth_sessions_absolute_chk CHECK ((absolute_expires_at >= expires_at)),
    CONSTRAINT auth_sessions_device_label_chk CHECK ((char_length(device_label) <= 80)),
    CONSTRAINT auth_sessions_prev_hash_chk CHECK (((prev_refresh_hash IS NULL) OR (prev_refresh_hash ~ '^[0-9a-f]{64}$'::text))),
    CONSTRAINT auth_sessions_reason_chk CHECK ((revoked_reason = ANY (ARRAY['LOGOUT'::text, 'REVOKED_BY_USER'::text, 'PASSWORD_CHANGED'::text, 'PASSWORD_RESET'::text, 'REFRESH_REUSE'::text, 'ACCOUNT_DISABLED'::text, 'ROLE_CHANGED'::text, 'ADMIN'::text]))),
    CONSTRAINT auth_sessions_refresh_hash_chk CHECK ((refresh_hash ~ '^[0-9a-f]{64}$'::text)),
    CONSTRAINT auth_sessions_revoked_pair_chk CHECK (((revoked_at IS NULL) = (revoked_reason IS NULL))),
    CONSTRAINT auth_sessions_user_agent_chk CHECK ((char_length(user_agent) <= 300))
);


ALTER TABLE public.auth_sessions OWNER TO edupilot;

--
-- Name: auth_tokens; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.auth_tokens (
    id uuid DEFAULT uuidv7() NOT NULL,
    user_id uuid NOT NULL,
    kind public.auth_token_kind NOT NULL,
    token_hash character(64) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone,
    revoked_at timestamp with time zone,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT auth_tokens_hash_chk CHECK ((token_hash ~ '^[0-9a-f]{64}$'::text)),
    CONSTRAINT auth_tokens_terminal_chk CHECK ((NOT ((used_at IS NOT NULL) AND (revoked_at IS NOT NULL))))
);


ALTER TABLE public.auth_tokens OWNER TO edupilot;

--
-- Name: calendar_events; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.calendar_events (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    type public.calendar_event_type NOT NULL,
    title text NOT NULL,
    starts_at timestamp with time zone NOT NULL,
    ends_at timestamp with time zone,
    location text,
    description text,
    ref_type text,
    ref_id uuid,
    created_by uuid NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT calendar_events_description_chk CHECK ((char_length(description) <= 1000)),
    CONSTRAINT calendar_events_location_chk CHECK ((char_length(location) <= 80)),
    CONSTRAINT calendar_events_ref_chk CHECK (((ref_type IS NULL) = (ref_id IS NULL))),
    CONSTRAINT calendar_events_time_chk CHECK (((ends_at IS NULL) OR (ends_at > starts_at))),
    CONSTRAINT calendar_events_title_chk CHECK (((char_length(title) >= 1) AND (char_length(title) <= 120))),
    CONSTRAINT calendar_events_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.calendar_events OWNER TO edupilot;

--
-- Name: chat_messages; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.chat_messages (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    session_id uuid NOT NULL,
    user_id uuid NOT NULL,
    role public.chat_role NOT NULL,
    content text DEFAULT ''::text NOT NULL,
    partial_content text,
    stream_status public.chat_stream_status DEFAULT 'DONE'::public.chat_stream_status NOT NULL,
    client_msg_id uuid,
    reply_to uuid,
    attempt smallint DEFAULT 1 NOT NULL,
    intent text,
    citations jsonb DEFAULT '[]'::jsonb NOT NULL,
    blocks jsonb DEFAULT '[]'::jsonb NOT NULL,
    confidence numeric(4,3),
    low_confidence boolean DEFAULT false NOT NULL,
    no_context boolean DEFAULT false NOT NULL,
    degraded boolean DEFAULT false NOT NULL,
    masked_count integer DEFAULT 0 NOT NULL,
    feedback public.chat_feedback,
    error_code text,
    trace_id text,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chat_messages_attempt_chk CHECK ((attempt >= 1)),
    CONSTRAINT chat_messages_blocks_chk CHECK ((jsonb_typeof(blocks) = 'array'::text)),
    CONSTRAINT chat_messages_citations_chk CHECK ((jsonb_typeof(citations) = 'array'::text)),
    CONSTRAINT chat_messages_completed_chk CHECK (((stream_status <> 'STREAMING'::public.chat_stream_status) OR (completed_at IS NULL))),
    CONSTRAINT chat_messages_confidence_chk CHECK (((confidence >= (0)::numeric) AND (confidence <= (1)::numeric))),
    CONSTRAINT chat_messages_content_chk CHECK ((char_length(content) <= 20000)),
    CONSTRAINT chat_messages_intent_chk CHECK ((intent ~ '^[A-Z_]{3,40}$'::text)),
    CONSTRAINT chat_messages_masked_chk CHECK ((masked_count >= 0)),
    CONSTRAINT chat_messages_partial_chk CHECK (((stream_status <> 'DONE'::public.chat_stream_status) OR (partial_content IS NULL))),
    CONSTRAINT chat_messages_user_chk CHECK (((role <> 'USER'::public.chat_role) OR ((stream_status = 'DONE'::public.chat_stream_status) AND (confidence IS NULL))))
);


ALTER TABLE public.chat_messages OWNER TO edupilot;

--
-- Name: chat_sessions; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.chat_sessions (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    user_id uuid NOT NULL,
    channel public.chat_channel DEFAULT 'PRIVATE'::public.chat_channel NOT NULL,
    title text,
    document_id uuid,
    last_message_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chat_sessions_title_chk CHECK ((char_length(title) <= 120))
);


ALTER TABLE public.chat_sessions OWNER TO edupilot;

--
-- Name: checkin_windows; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.checkin_windows (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    session_id uuid NOT NULL,
    round smallint NOT NULL,
    rotate_seconds smallint NOT NULL,
    headcount integer,
    opened_by uuid NOT NULL,
    opened_at timestamp with time zone DEFAULT now() NOT NULL,
    closes_at timestamp with time zone NOT NULL,
    closed_by uuid,
    closed_at timestamp with time zone,
    CONSTRAINT checkin_windows_headcount_chk CHECK (((headcount >= 0) AND (headcount <= 2000))),
    CONSTRAINT checkin_windows_rotate_chk CHECK (((rotate_seconds >= 5) AND (rotate_seconds <= 60))),
    CONSTRAINT checkin_windows_round_chk CHECK (((round >= 1) AND (round <= 3)))
);


ALTER TABLE public.checkin_windows OWNER TO edupilot;

--
-- Name: class_sessions; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.class_sessions (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    session_no integer NOT NULL,
    starts_at timestamp with time zone NOT NULL,
    ends_at timestamp with time zone NOT NULL,
    room text,
    topic text,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    attendance_done_at timestamp with time zone,
    attendance_done_by uuid,
    CONSTRAINT class_sessions_done_chk CHECK (((attendance_done_at IS NULL) = (attendance_done_by IS NULL))),
    CONSTRAINT class_sessions_no_chk CHECK ((session_no >= 1)),
    CONSTRAINT class_sessions_room_chk CHECK ((char_length(room) <= 40)),
    CONSTRAINT class_sessions_time_chk CHECK ((ends_at > starts_at)),
    CONSTRAINT class_sessions_topic_chk CHECK ((char_length(topic) <= 200)),
    CONSTRAINT class_sessions_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.class_sessions OWNER TO edupilot;

--
-- Name: code_drafts; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.code_drafts (
    attempt_id uuid NOT NULL,
    item_id uuid NOT NULL,
    course_id uuid NOT NULL,
    language text NOT NULL,
    source text NOT NULL,
    rev integer DEFAULT 1 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT code_drafts_language_chk CHECK ((language = ANY (ARRAY['c11'::text, 'cpp17'::text]))),
    CONSTRAINT code_drafts_rev_chk CHECK ((rev >= 1)),
    CONSTRAINT code_drafts_source_chk CHECK ((octet_length(source) <= 65536))
);


ALTER TABLE public.code_drafts OWNER TO edupilot;

--
-- Name: code_problems; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.code_problems (
    question_id uuid NOT NULL,
    course_id uuid NOT NULL,
    languages text[] DEFAULT '{cpp17}'::text[] NOT NULL,
    time_limit_ms integer DEFAULT 1000 NOT NULL,
    memory_limit_mb integer DEFAULT 256 NOT NULL,
    output_limit_kb integer DEFAULT 1024 NOT NULL,
    checker public.checker_kind DEFAULT 'EXACT'::public.checker_kind NOT NULL,
    float_eps numeric(12,10),
    starter_code jsonb DEFAULT '{}'::jsonb NOT NULL,
    reference_language text,
    reference_source text,
    reference_verified_version integer,
    reference_verified_at timestamp with time zone,
    tests_version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT code_problems_eps_chk CHECK (((checker = 'FLOAT_EPS'::public.checker_kind) = (float_eps IS NOT NULL))),
    CONSTRAINT code_problems_eps_range_chk CHECK (((float_eps IS NULL) OR ((float_eps > (0)::numeric) AND (float_eps <= 0.1)))),
    CONSTRAINT code_problems_languages_chk CHECK (((cardinality(languages) >= 1) AND (languages <@ ARRAY['c11'::text, 'cpp17'::text]))),
    CONSTRAINT code_problems_memory_chk CHECK (((memory_limit_mb >= 16) AND (memory_limit_mb <= 512))),
    CONSTRAINT code_problems_output_chk CHECK (((output_limit_kb >= 1) AND (output_limit_kb <= 16384))),
    CONSTRAINT code_problems_ref_lang_chk CHECK ((reference_language = ANY (ARRAY['c11'::text, 'cpp17'::text]))),
    CONSTRAINT code_problems_ref_pair_chk CHECK (((reference_language IS NULL) = (reference_source IS NULL))),
    CONSTRAINT code_problems_ref_size_chk CHECK ((octet_length(reference_source) <= 65536)),
    CONSTRAINT code_problems_starter_chk CHECK ((jsonb_typeof(starter_code) = 'object'::text)),
    CONSTRAINT code_problems_tests_version_chk CHECK ((tests_version >= 1)),
    CONSTRAINT code_problems_time_chk CHECK (((time_limit_ms >= 100) AND (time_limit_ms <= 10000)))
);


ALTER TABLE public.code_problems OWNER TO edupilot;

--
-- Name: code_submissions; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.code_submissions (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    exam_id uuid NOT NULL,
    attempt_id uuid NOT NULL,
    item_id uuid NOT NULL,
    problem_id uuid NOT NULL,
    student_id uuid NOT NULL,
    kind public.submission_kind NOT NULL,
    language text NOT NULL,
    source text NOT NULL,
    source_sha256 character(64) NOT NULL,
    auto boolean DEFAULT false NOT NULL,
    status public.submission_status DEFAULT 'QUEUED'::public.submission_status NOT NULL,
    verdict public.judge_verdict,
    tests_version integer,
    compile_ok boolean,
    compile_log text,
    results jsonb,
    passed_weight integer,
    total_weight integer,
    time_ms_max integer,
    memory_kb_max integer,
    attempts smallint DEFAULT 0 NOT NULL,
    lease_until timestamp with time zone,
    next_attempt_at timestamp with time zone DEFAULT now() NOT NULL,
    fail_count smallint DEFAULT 0 NOT NULL,
    enqueued_at timestamp with time zone,
    judged_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT code_submissions_auto_chk CHECK (((NOT auto) OR (kind = 'SUBMIT'::public.submission_kind))),
    CONSTRAINT code_submissions_compile_log_chk CHECK ((char_length(compile_log) <= 8192)),
    CONSTRAINT code_submissions_fail_count_chk CHECK (((fail_count >= 0) AND (fail_count <= 4))),
    CONSTRAINT code_submissions_language_chk CHECK ((language = ANY (ARRAY['c11'::text, 'cpp17'::text]))),
    CONSTRAINT code_submissions_lease_chk CHECK (((status = 'RUNNING'::public.submission_status) = (lease_until IS NOT NULL))),
    CONSTRAINT code_submissions_source_chk CHECK (((octet_length(source) >= 1) AND (octet_length(source) <= 65536))),
    CONSTRAINT code_submissions_verdict_chk CHECK (((status = ANY (ARRAY['DONE'::public.submission_status, 'ERROR'::public.submission_status])) = (verdict IS NOT NULL))),
    CONSTRAINT code_submissions_weight_chk CHECK ((passed_weight <= total_weight))
);


ALTER TABLE public.code_submissions OWNER TO edupilot;

--
-- Name: code_testcases; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.code_testcases (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    problem_id uuid NOT NULL,
    "position" integer NOT NULL,
    name text NOT NULL,
    is_sample boolean DEFAULT false NOT NULL,
    weight smallint DEFAULT 1 NOT NULL,
    input text,
    input_blob_key text,
    expected text,
    expected_blob_key text,
    input_bytes integer NOT NULL,
    expected_bytes integer NOT NULL,
    source text DEFAULT 'MANUAL'::text NOT NULL,
    approved boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT code_testcases_bytes_chk CHECK ((((input_bytes >= 0) AND (input_bytes <= 1048576)) AND ((expected_bytes >= 0) AND (expected_bytes <= 1048576)))),
    CONSTRAINT code_testcases_expected_chk CHECK (((expected IS NULL) <> (expected_blob_key IS NULL))),
    CONSTRAINT code_testcases_expected_inline_chk CHECK ((octet_length(expected) <= 65536)),
    CONSTRAINT code_testcases_input_chk CHECK (((input IS NULL) <> (input_blob_key IS NULL))),
    CONSTRAINT code_testcases_input_inline_chk CHECK ((octet_length(input) <= 65536)),
    CONSTRAINT code_testcases_name_chk CHECK ((name ~ '^[A-Za-z0-9_-]{1,60}$'::text)),
    CONSTRAINT code_testcases_position_chk CHECK (("position" >= 1)),
    CONSTRAINT code_testcases_source_chk CHECK ((source = ANY (ARRAY['MANUAL'::text, 'IMPORT'::text, 'AI_DRAFT'::text]))),
    CONSTRAINT code_testcases_weight_chk CHECK (((weight >= 0) AND (weight <= 1000)))
);


ALTER TABLE public.code_testcases OWNER TO edupilot;

--
-- Name: content_chunks; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.content_chunks (
    id uuid DEFAULT uuidv7() NOT NULL,
    document_id uuid NOT NULL,
    course_ids uuid[] NOT NULL,
    audience public.chunk_audience DEFAULT 'ALL'::public.chunk_audience NOT NULL,
    ord integer NOT NULL,
    page_no integer,
    heading text,
    text text NOT NULL,
    token_count integer,
    embedding public.vector(1536),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    tsv tsvector GENERATED ALWAYS AS (to_tsvector('simple'::regconfig, public.vn_fold(NORMALIZE(text, NFC)))) STORED,
    CONSTRAINT content_chunks_course_ids_chk CHECK ((cardinality(course_ids) >= 1)),
    CONSTRAINT content_chunks_heading_chk CHECK ((char_length(heading) <= 200)),
    CONSTRAINT content_chunks_ord_chk CHECK ((ord >= 0))
);


ALTER TABLE public.content_chunks OWNER TO edupilot;

--
-- Name: course_decision_gates; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.course_decision_gates (
    course_id uuid NOT NULL,
    gate text NOT NULL,
    enabled boolean DEFAULT false NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    updated_by uuid,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.course_decision_gates OWNER TO edupilot;

--
-- Name: course_guidance_flags; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.course_guidance_flags (
    course_id uuid NOT NULL,
    feature text NOT NULL,
    enabled boolean DEFAULT false NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    updated_by uuid,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT course_guidance_flags_feature_check CHECK ((feature = ANY (ARRAY['GRADING_GUIDANCE'::text, 'REGRESSION_CAPTURE'::text, 'THREAD_SUGGEST'::text, 'QUESTION_HINTS'::text])))
);


ALTER TABLE public.course_guidance_flags OWNER TO edupilot;

--
-- Name: courses; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.courses (
    id uuid DEFAULT uuidv7() NOT NULL,
    subject_code text NOT NULL,
    class_code text NOT NULL,
    name text NOT NULL,
    semester text NOT NULL,
    status public.course_status DEFAULT 'ACTIVE'::public.course_status NOT NULL,
    escalation_threshold numeric(3,2) DEFAULT 0.80 NOT NULL,
    settings jsonb DEFAULT '{}'::jsonb NOT NULL,
    join_code character(7) NOT NULL,
    join_enabled boolean DEFAULT true NOT NULL,
    join_expires_at timestamp with time zone,
    join_require_approval boolean DEFAULT false NOT NULL,
    allowed_email_domain text,
    capacity integer,
    created_by uuid NOT NULL,
    archived_at timestamp with time zone,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    allow_study_time_opt_out boolean DEFAULT true NOT NULL,
    CONSTRAINT courses_archived_chk CHECK (((status = 'ARCHIVED'::public.course_status) = (archived_at IS NOT NULL))),
    CONSTRAINT courses_archived_join_chk CHECK (((status = 'ACTIVE'::public.course_status) OR (join_enabled = false))),
    CONSTRAINT courses_capacity_chk CHECK (((capacity IS NULL) OR ((capacity >= 1) AND (capacity <= 1000)))),
    CONSTRAINT courses_class_code_chk CHECK ((class_code ~ '^[A-Za-z0-9._-]{3,20}$'::text)),
    CONSTRAINT courses_domain_chk CHECK (((allowed_email_domain IS NULL) OR (allowed_email_domain ~ '^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$'::text))),
    CONSTRAINT courses_escalation_chk CHECK (((escalation_threshold >= (0)::numeric) AND (escalation_threshold <= (1)::numeric))),
    CONSTRAINT courses_join_code_chk CHECK ((join_code ~ '^[ABCDEFGHJKMNPQRSTUVWXYZ23456789]{7}$'::text)),
    CONSTRAINT courses_name_chk CHECK (((char_length(name) >= 1) AND (char_length(name) <= 120))),
    CONSTRAINT courses_semester_chk CHECK ((semester ~ '^[0-9]{4}-[0-9]{4}-HK[123]$'::text)),
    CONSTRAINT courses_settings_chk CHECK ((jsonb_typeof(settings) = 'object'::text)),
    CONSTRAINT courses_subject_code_chk CHECK ((subject_code ~ '^[A-Z0-9._-]{2,20}$'::text)),
    CONSTRAINT courses_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.courses OWNER TO edupilot;

--
-- Name: decision_gates; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.decision_gates (
    gate text NOT NULL,
    enabled boolean DEFAULT false NOT NULL,
    threshold numeric(4,3) NOT NULL,
    timeout_ms integer NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    updated_by uuid,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT decision_gates_gate_check CHECK ((gate = ANY (ARRAY['THREADS_PRECHECK'::text, 'INJECTION_FLAG'::text, 'CHAT_ROUTING'::text]))),
    CONSTRAINT decision_gates_timeout_ms_check CHECK (((timeout_ms >= 200) AND (timeout_ms <= 5000)))
);


ALTER TABLE public.decision_gates OWNER TO edupilot;

--
-- Name: document_courses; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.document_courses (
    document_id uuid NOT NULL,
    course_id uuid NOT NULL,
    shared_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.document_courses OWNER TO edupilot;

--
-- Name: documents; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.documents (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    title text NOT NULL,
    type public.document_type DEFAULT 'LECTURE'::public.document_type NOT NULL,
    filename text,
    mime_type text,
    size_bytes bigint,
    sha256 character(64),
    blob_key text,
    status public.document_status DEFAULT 'QUEUED'::public.document_status NOT NULL,
    error text,
    page_count integer,
    visible_to_students boolean DEFAULT true NOT NULL,
    use_for_rag boolean DEFAULT true NOT NULL,
    category text,
    week_no smallint,
    download_count integer DEFAULT 0 NOT NULL,
    uploaded_by uuid,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT documents_answer_key_chk CHECK (((type <> 'ANSWER_KEY'::public.document_type) OR (visible_to_students = false))),
    CONSTRAINT documents_category_chk CHECK ((char_length(category) <= 40)),
    CONSTRAINT documents_download_chk CHECK ((download_count >= 0)),
    CONSTRAINT documents_error_chk CHECK ((char_length(error) <= 1000)),
    CONSTRAINT documents_filename_chk CHECK ((char_length(filename) <= 255)),
    CONSTRAINT documents_mime_chk CHECK ((char_length(mime_type) <= 100)),
    CONSTRAINT documents_page_count_chk CHECK (((page_count IS NULL) OR (page_count >= 0))),
    CONSTRAINT documents_sha256_chk CHECK (((sha256 IS NULL) OR (sha256 ~ '^[0-9a-f]{64}$'::text))),
    CONSTRAINT documents_size_chk CHECK (((size_bytes IS NULL) OR (size_bytes >= 0))),
    CONSTRAINT documents_title_chk CHECK (((char_length(title) >= 1) AND (char_length(title) <= 200))),
    CONSTRAINT documents_version_chk CHECK ((version >= 1)),
    CONSTRAINT documents_week_chk CHECK (((week_no IS NULL) OR ((week_no >= 1) AND (week_no <= 20))))
);


ALTER TABLE public.documents OWNER TO edupilot;

--
-- Name: enrollments; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.enrollments (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    user_id uuid NOT NULL,
    role_in_course public.enrollment_role NOT NULL,
    status public.enrollment_status NOT NULL,
    joined_via public.enrollment_joined_via NOT NULL,
    student_code_snapshot text,
    warning text,
    previous_status public.enrollment_status,
    status_changed_at timestamp with time zone DEFAULT now() NOT NULL,
    status_changed_by uuid,
    removed_at timestamp with time zone,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT enrollments_code_chk CHECK (((student_code_snapshot IS NULL) OR (student_code_snapshot ~ '^[A-Z0-9]{6,15}$'::text))),
    CONSTRAINT enrollments_code_role_chk CHECK (((role_in_course = 'STUDENT'::public.enrollment_role) OR (student_code_snapshot IS NULL))),
    CONSTRAINT enrollments_removed_chk CHECK (((status = 'REMOVED'::public.enrollment_status) = (removed_at IS NOT NULL))),
    CONSTRAINT enrollments_version_chk CHECK ((version >= 1)),
    CONSTRAINT enrollments_warning_chk CHECK ((warning = ANY (ARRAY['EMAIL_MISMATCH'::text, 'EMAIL_UNVERIFIED'::text]))),
    CONSTRAINT enrollments_warning_role_chk CHECK (((warning IS NULL) OR (role_in_course = 'STUDENT'::public.enrollment_role)))
);


ALTER TABLE public.enrollments OWNER TO edupilot;

--
-- Name: escalation_tickets; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.escalation_tickets (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    kind public.ticket_kind DEFAULT 'QUESTION'::public.ticket_kind NOT NULL,
    origin public.ticket_origin NOT NULL,
    status public.ticket_status DEFAULT 'OPEN'::public.ticket_status NOT NULL,
    source_type text NOT NULL,
    source_id uuid NOT NULL,
    message_id uuid,
    session_id uuid,
    student_id uuid NOT NULL,
    confidence numeric(4,3),
    appeal_payload jsonb,
    claimed_by uuid,
    claimed_at timestamp with time zone,
    answer text,
    answered_by uuid,
    answered_at timestamp with time zone,
    closed_at timestamp with time zone,
    closed_reason text,
    dismiss_reason text,
    reopened_count smallint DEFAULT 0 NOT NULL,
    sla_stage smallint DEFAULT 0 NOT NULL,
    sla_stage_at timestamp with time zone,
    waiting_since timestamp with time zone DEFAULT now() NOT NULL,
    save_as_knowledge boolean DEFAULT false NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT escalation_tickets_answer_chk CHECK ((char_length(answer) <= 4000)),
    CONSTRAINT escalation_tickets_answered_chk CHECK (((status <> 'ANSWERED'::public.ticket_status) OR (answered_by IS NOT NULL))),
    CONSTRAINT escalation_tickets_appeal_chk CHECK (((kind = 'GRADE_APPEAL'::public.ticket_kind) = (appeal_payload IS NOT NULL))),
    CONSTRAINT escalation_tickets_appeal_type_chk CHECK (((appeal_payload IS NULL) OR (jsonb_typeof(appeal_payload) = 'object'::text))),
    CONSTRAINT escalation_tickets_claimed_chk CHECK (((status <> 'CLAIMED'::public.ticket_status) OR (claimed_by IS NOT NULL))),
    CONSTRAINT escalation_tickets_closed_chk CHECK (((status <> 'CLOSED'::public.ticket_status) OR (closed_at IS NOT NULL))),
    CONSTRAINT escalation_tickets_closed_reason_chk CHECK (((closed_reason IS NULL) OR (closed_reason = ANY (ARRAY['STUDENT_CONFIRMED'::text, 'AUTO_7D'::text, 'STAFF_DISMISSED'::text])))),
    CONSTRAINT escalation_tickets_confidence_chk CHECK (((confidence >= (0)::numeric) AND (confidence <= (1)::numeric))),
    CONSTRAINT escalation_tickets_dismiss_reason_chk CHECK (((dismiss_reason IS NULL) OR (dismiss_reason = ANY (ARRAY['OUT_OF_SCOPE'::text, 'DUPLICATE'::text, 'NEED_MORE_INFO'::text])))),
    CONSTRAINT escalation_tickets_open_chk CHECK (((status <> 'OPEN'::public.ticket_status) OR (claimed_by IS NULL))),
    CONSTRAINT escalation_tickets_question_chk CHECK (((kind <> 'QUESTION'::public.ticket_kind) OR (message_id IS NOT NULL))),
    CONSTRAINT escalation_tickets_sla_chk CHECK (((sla_stage >= 0) AND (sla_stage <= 2))),
    CONSTRAINT escalation_tickets_source_type_chk CHECK ((source_type ~ '^[A-Z_]{3,30}$'::text)),
    CONSTRAINT escalation_tickets_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.escalation_tickets OWNER TO edupilot;

--
-- Name: exam_answers; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.exam_answers (
    attempt_id uuid NOT NULL,
    item_id uuid NOT NULL,
    course_id uuid NOT NULL,
    answer jsonb NOT NULL,
    saved_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT exam_answers_size_chk CHECK ((octet_length((answer)::text) <= 4096))
);


ALTER TABLE public.exam_answers OWNER TO edupilot;

--
-- Name: exam_appeals; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.exam_appeals (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    exam_id uuid NOT NULL,
    attempt_id uuid NOT NULL,
    student_id uuid NOT NULL,
    reason text NOT NULL,
    status public.appeal_status DEFAULT 'OPEN'::public.appeal_status NOT NULL,
    response text,
    responded_by uuid,
    responded_at timestamp with time zone,
    score_before numeric(5,2),
    score_after numeric(5,2),
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT exam_appeals_reason_chk CHECK (((char_length(reason) >= 1) AND (char_length(reason) <= 1000))),
    CONSTRAINT exam_appeals_responded_chk CHECK (((status = 'OPEN'::public.appeal_status) = (responded_at IS NULL))),
    CONSTRAINT exam_appeals_response_chk CHECK (((char_length(response) >= 1) AND (char_length(response) <= 1000))),
    CONSTRAINT exam_appeals_score_after_chk CHECK (((status = 'ADJUSTED'::public.appeal_status) = (score_after IS NOT NULL))),
    CONSTRAINT exam_appeals_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.exam_appeals OWNER TO edupilot;

--
-- Name: exam_attempts; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.exam_attempts (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    exam_id uuid NOT NULL,
    student_id uuid NOT NULL,
    status public.attempt_status DEFAULT 'IN_PROGRESS'::public.attempt_status NOT NULL,
    started_at timestamp with time zone DEFAULT now() NOT NULL,
    deadline_at timestamp with time zone NOT NULL,
    submitted_at timestamp with time zone,
    submit_reason public.attempt_submit_reason,
    writer_tab uuid,
    writer_seen_at timestamp with time zone,
    auto_score numeric(5,2),
    adjusted_score numeric(5,2),
    adjusted_reason text,
    adjusted_by uuid,
    adjusted_at timestamp with time zone,
    breakdown jsonb,
    graded_at timestamp with time zone,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT exam_attempts_adjusted_chk CHECK (((adjusted_score IS NULL) OR (((adjusted_score >= (0)::numeric) AND (adjusted_score <= (100)::numeric)) AND (adjusted_reason IS NOT NULL)))),
    CONSTRAINT exam_attempts_adjusted_reason_chk CHECK (((char_length(adjusted_reason) >= 1) AND (char_length(adjusted_reason) <= 500))),
    CONSTRAINT exam_attempts_auto_score_chk CHECK (((auto_score IS NULL) OR ((auto_score >= (0)::numeric) AND (auto_score <= (100)::numeric)))),
    CONSTRAINT exam_attempts_breakdown_chk CHECK (((breakdown IS NULL) OR (jsonb_typeof(breakdown) = 'array'::text))),
    CONSTRAINT exam_attempts_deadline_chk CHECK ((deadline_at > started_at)),
    CONSTRAINT exam_attempts_graded_chk CHECK (((status = 'GRADED'::public.attempt_status) = ((auto_score IS NOT NULL) AND (graded_at IS NOT NULL)))),
    CONSTRAINT exam_attempts_reason_chk CHECK (((submitted_at IS NOT NULL) = (submit_reason IS NOT NULL))),
    CONSTRAINT exam_attempts_submitted_chk CHECK (((status <> 'IN_PROGRESS'::public.attempt_status) = (submitted_at IS NOT NULL))),
    CONSTRAINT exam_attempts_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.exam_attempts OWNER TO edupilot;

--
-- Name: exam_events; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.exam_events (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    exam_id uuid NOT NULL,
    attempt_id uuid NOT NULL,
    student_id uuid NOT NULL,
    type public.exam_event_type NOT NULL,
    occurred_at timestamp with time zone DEFAULT now() NOT NULL,
    client_at timestamp with time zone,
    meta jsonb DEFAULT '{}'::jsonb NOT NULL,
    CONSTRAINT exam_events_meta_chk CHECK ((octet_length((meta)::text) <= 300))
);


ALTER TABLE public.exam_events OWNER TO edupilot;

--
-- Name: exam_items; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.exam_items (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    exam_id uuid NOT NULL,
    question_id uuid NOT NULL,
    "position" smallint NOT NULL,
    points numeric(5,2) DEFAULT 1.00 NOT NULL,
    override jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT exam_items_override_chk CHECK (((override IS NULL) OR (jsonb_typeof(override) = 'object'::text))),
    CONSTRAINT exam_items_points_chk CHECK (((points > (0)::numeric) AND (points <= (100)::numeric))),
    CONSTRAINT exam_items_position_chk CHECK (("position" >= 1))
);


ALTER TABLE public.exam_items OWNER TO edupilot;

--
-- Name: exams; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.exams (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    title text NOT NULL,
    instructions text,
    kind public.exam_kind DEFAULT 'MCQ'::public.exam_kind NOT NULL,
    status public.exam_status DEFAULT 'DRAFT'::public.exam_status NOT NULL,
    opens_at timestamp with time zone,
    closes_at timestamp with time zone,
    duration_minutes smallint,
    shuffle_questions boolean DEFAULT true NOT NULL,
    shuffle_options boolean DEFAULT true NOT NULL,
    max_score numeric(5,2) DEFAULT 10.00 NOT NULL,
    rounding_step numeric(3,2) DEFAULT 0.01 NOT NULL,
    multi_scoring public.multi_scoring DEFAULT 'PARTIAL'::public.multi_scoring NOT NULL,
    reveal_answers boolean DEFAULT true NOT NULL,
    appeal_days smallint DEFAULT 7 NOT NULL,
    publish_hold boolean DEFAULT false NOT NULL,
    regrading boolean DEFAULT false NOT NULL,
    published_at timestamp with time zone,
    created_by uuid NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT exams_appeal_days_chk CHECK (((appeal_days >= 0) AND (appeal_days <= 30))),
    CONSTRAINT exams_duration_chk CHECK (((duration_minutes >= 1) AND (duration_minutes <= 300))),
    CONSTRAINT exams_duration_window_chk CHECK (((duration_minutes IS NULL) OR (opens_at IS NULL) OR (closes_at IS NULL) OR (((duration_minutes)::double precision * '00:01:00'::interval) <= (closes_at - opens_at)))),
    CONSTRAINT exams_instructions_chk CHECK ((char_length(instructions) <= 4000)),
    CONSTRAINT exams_max_score_chk CHECK (((max_score > (0)::numeric) AND (max_score <= (100)::numeric))),
    CONSTRAINT exams_published_chk CHECK (((status = 'PUBLISHED'::public.exam_status) = (published_at IS NOT NULL))),
    CONSTRAINT exams_scheduled_chk CHECK (((status = 'DRAFT'::public.exam_status) OR ((opens_at IS NOT NULL) AND (closes_at IS NOT NULL) AND (duration_minutes IS NOT NULL)))),
    CONSTRAINT exams_step_chk CHECK ((rounding_step = ANY (ARRAY[0.01, 0.10, 0.25, 0.50, 1.00]))),
    CONSTRAINT exams_title_chk CHECK (((char_length(title) >= 1) AND (char_length(title) <= 120))),
    CONSTRAINT exams_version_chk CHECK ((version >= 1)),
    CONSTRAINT exams_window_chk CHECK (((opens_at IS NULL) OR (closes_at IS NULL) OR (closes_at > opens_at)))
);


ALTER TABLE public.exams OWNER TO edupilot;

--
-- Name: final_grade_snapshots; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.final_grade_snapshots (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    student_id uuid NOT NULL,
    version integer NOT NULL,
    scheme_id uuid NOT NULL,
    breakdown jsonb NOT NULL,
    numeric_grade numeric(5,2) NOT NULL,
    letter text NOT NULL,
    failed_reason text,
    locked_at timestamp with time zone DEFAULT now() NOT NULL,
    locked_by uuid NOT NULL,
    unlock_reason text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT final_grade_snapshots_breakdown_chk CHECK ((jsonb_typeof(breakdown) = 'object'::text)),
    CONSTRAINT final_grade_snapshots_grade_chk CHECK (((numeric_grade >= (0)::numeric) AND (numeric_grade <= (10)::numeric))),
    CONSTRAINT final_grade_snapshots_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.final_grade_snapshots OWNER TO edupilot;

--
-- Name: forms_import_batches; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.forms_import_batches (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    item_id uuid NOT NULL,
    file_hash text NOT NULL,
    blob_key text NOT NULL,
    mapping jsonb NOT NULL,
    matched integer DEFAULT 0 NOT NULL,
    unmatched integer DEFAULT 0 NOT NULL,
    created_by uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    blob_purged_at timestamp with time zone,
    CONSTRAINT forms_import_batches_counts_chk CHECK (((matched >= 0) AND (unmatched >= 0))),
    CONSTRAINT forms_import_batches_hash_chk CHECK ((file_hash ~ '^[0-9a-f]{64}$'::text))
);


ALTER TABLE public.forms_import_batches OWNER TO edupilot;

--
-- Name: forms_import_rows; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.forms_import_rows (
    course_id uuid NOT NULL,
    batch_id uuid NOT NULL,
    row_no integer NOT NULL,
    student_id uuid,
    identifier_masked text NOT NULL,
    correct integer NOT NULL,
    total integer NOT NULL,
    score numeric(5,2) NOT NULL,
    CONSTRAINT forms_import_rows_counts_chk CHECK (((correct >= 0) AND (total >= correct))),
    CONSTRAINT forms_import_rows_score_chk CHECK ((score >= (0)::numeric))
);


ALTER TABLE public.forms_import_rows OWNER TO edupilot;

--
-- Name: forum_posts; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.forum_posts (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    thread_id uuid NOT NULL,
    author_id uuid,
    kind public.post_kind NOT NULL,
    body text NOT NULL,
    verification_state public.post_verification DEFAULT 'NONE'::public.post_verification NOT NULL,
    citations jsonb DEFAULT '[]'::jsonb NOT NULL,
    confidence numeric(4,3),
    ai_body text,
    verified_by uuid,
    verified_at timestamp with time zone,
    hidden_at timestamp with time zone,
    hidden_reason text,
    hidden_by uuid,
    deleted_at timestamp with time zone,
    embedding public.vector(1536),
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    edited_at timestamp with time zone,
    CONSTRAINT forum_posts_author_chk CHECK (((kind = 'AI'::public.post_kind) = (author_id IS NULL))),
    CONSTRAINT forum_posts_body_chk CHECK (((char_length(body) >= 1) AND (char_length(body) <= 8000))),
    CONSTRAINT forum_posts_citations_chk CHECK ((jsonb_typeof(citations) = 'array'::text)),
    CONSTRAINT forum_posts_confidence_chk CHECK (((confidence IS NULL) OR ((kind = 'AI'::public.post_kind) AND ((confidence >= (0)::numeric) AND (confidence <= (1)::numeric))))),
    CONSTRAINT forum_posts_hidden_chk CHECK ((((hidden_at IS NULL) = (hidden_reason IS NULL)) AND (char_length(hidden_reason) <= 200))),
    CONSTRAINT forum_posts_verification_chk CHECK (((kind = 'HUMAN'::public.post_kind) = (verification_state = 'NONE'::public.post_verification))),
    CONSTRAINT forum_posts_verified_chk CHECK (((verified_by IS NULL) = (verified_at IS NULL))),
    CONSTRAINT forum_posts_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.forum_posts OWNER TO edupilot;

--
-- Name: forum_threads; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.forum_threads (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    author_id uuid NOT NULL,
    title text NOT NULL,
    body text NOT NULL,
    tags text[] DEFAULT '{}'::text[] NOT NULL,
    week_no smallint,
    state public.thread_state DEFAULT 'OPEN'::public.thread_state NOT NULL,
    ai_state public.thread_ai_state DEFAULT 'PENDING'::public.thread_ai_state NOT NULL,
    ai_skip_reason text,
    similar_of uuid,
    pinned_at timestamp with time zone,
    reply_count integer DEFAULT 0 NOT NULL,
    last_activity_at timestamp with time zone DEFAULT now() NOT NULL,
    embedding public.vector(1536),
    deleted_at timestamp with time zone,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    hidden_at timestamp with time zone,
    hidden_reason text,
    hidden_by uuid,
    edited_at timestamp with time zone,
    CONSTRAINT forum_threads_body_chk CHECK (((char_length(body) >= 1) AND (char_length(body) <= 8000))),
    CONSTRAINT forum_threads_hidden_chk CHECK ((((hidden_at IS NULL) = (hidden_reason IS NULL)) AND (char_length(hidden_reason) <= 200))),
    CONSTRAINT forum_threads_reply_chk CHECK ((reply_count >= 0)),
    CONSTRAINT forum_threads_skip_chk CHECK ((((ai_state = 'SKIPPED'::public.thread_ai_state) = (ai_skip_reason IS NOT NULL)) AND ((ai_skip_reason IS NULL) OR (ai_skip_reason = ANY (ARRAY['NO_CONTEXT'::text, 'LOW_SCORE'::text, 'LLM_UNAVAILABLE'::text]))))),
    CONSTRAINT forum_threads_tags_chk CHECK (public.forum_tags_valid(tags)),
    CONSTRAINT forum_threads_title_chk CHECK (((char_length(title) >= 1) AND (char_length(title) <= 200))),
    CONSTRAINT forum_threads_version_chk CHECK ((version >= 1)),
    CONSTRAINT forum_threads_week_chk CHECK (((week_no >= 1) AND (week_no <= 20)))
);


ALTER TABLE public.forum_threads OWNER TO edupilot;

--
-- Name: goose_db_version; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.goose_db_version (
    id integer NOT NULL,
    version_id bigint NOT NULL,
    is_applied boolean NOT NULL,
    tstamp timestamp without time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.goose_db_version OWNER TO edupilot;

--
-- Name: goose_db_version_id_seq; Type: SEQUENCE; Schema: public; Owner: edupilot
--

ALTER TABLE public.goose_db_version ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.goose_db_version_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: grade_appeals; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.grade_appeals (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    assignment_id uuid NOT NULL,
    student_id uuid NOT NULL,
    submission_id uuid NOT NULL,
    result_id uuid NOT NULL,
    criteria_ids text[] NOT NULL,
    reason text NOT NULL,
    status text DEFAULT 'OPEN'::text NOT NULL,
    score_before numeric(5,2) NOT NULL,
    score_after numeric(5,2),
    response text,
    decided_by uuid,
    decided_at timestamp with time zone,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT grade_appeals_open_chk CHECK (((status = 'OPEN'::text) = (decided_at IS NULL))),
    CONSTRAINT grade_appeals_status_chk CHECK ((status = ANY (ARRAY['OPEN'::text, 'UPHELD'::text, 'ADJUSTED'::text]))),
    CONSTRAINT grade_appeals_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.grade_appeals OWNER TO edupilot;

--
-- Name: grade_components; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.grade_components (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    scheme_id uuid NOT NULL,
    key text NOT NULL,
    name text NOT NULL,
    grp public.grade_group NOT NULL,
    weight numeric(5,2) NOT NULL,
    "position" integer DEFAULT 0 NOT NULL,
    citations jsonb DEFAULT '[]'::jsonb NOT NULL,
    filled_by_user boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT grade_components_citations_chk CHECK ((jsonb_typeof(citations) = 'array'::text)),
    CONSTRAINT grade_components_key_chk CHECK ((key ~ '^[a-z][a-z0-9_]{0,39}$'::text)),
    CONSTRAINT grade_components_name_chk CHECK (((char_length(name) >= 1) AND (char_length(name) <= 80))),
    CONSTRAINT grade_components_weight_chk CHECK (((weight >= (0)::numeric) AND (weight <= (100)::numeric)))
);


ALTER TABLE public.grade_components OWNER TO edupilot;

--
-- Name: grade_entries; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.grade_entries (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    item_id uuid NOT NULL,
    student_id uuid NOT NULL,
    score numeric(5,2) NOT NULL,
    source public.grade_entry_source DEFAULT 'MANUAL'::public.grade_entry_source NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    entered_by uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT grade_entries_score_chk CHECK ((score >= (0)::numeric)),
    CONSTRAINT grade_entries_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.grade_entries OWNER TO edupilot;

--
-- Name: grade_items; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.grade_items (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    scheme_id uuid NOT NULL,
    component_id uuid NOT NULL,
    key text NOT NULL,
    name text NOT NULL,
    max_score numeric(5,2) DEFAULT 10 NOT NULL,
    source public.grade_item_source DEFAULT 'MANUAL'::public.grade_item_source NOT NULL,
    exam_id uuid,
    assignment_id uuid,
    published_at timestamp with time zone,
    "position" integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT grade_items_assignment_chk CHECK (((source = 'ASSIGNMENT'::public.grade_item_source) = (assignment_id IS NOT NULL))),
    CONSTRAINT grade_items_exam_chk CHECK (((source = 'EXAM'::public.grade_item_source) = (exam_id IS NOT NULL))),
    CONSTRAINT grade_items_key_chk CHECK ((key ~ '^[a-z][a-z0-9_]{0,39}$'::text)),
    CONSTRAINT grade_items_max_chk CHECK ((max_score > (0)::numeric)),
    CONSTRAINT grade_items_name_chk CHECK (((char_length(name) >= 1) AND (char_length(name) <= 80)))
);


ALTER TABLE public.grade_items OWNER TO edupilot;

--
-- Name: grade_rules; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.grade_rules (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    scheme_id uuid NOT NULL,
    kind public.grade_rule_kind NOT NULL,
    applicable boolean DEFAULT true NOT NULL,
    params jsonb DEFAULT '{}'::jsonb NOT NULL,
    source_quote text,
    source_page integer,
    filled_by_user boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT grade_rules_page_chk CHECK (((source_page IS NULL) OR (source_page >= 1))),
    CONSTRAINT grade_rules_params_chk CHECK ((jsonb_typeof(params) = 'object'::text))
);


ALTER TABLE public.grade_rules OWNER TO edupilot;

--
-- Name: grade_schemes; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.grade_schemes (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    scheme_no integer NOT NULL,
    status public.grade_scheme_status DEFAULT 'DRAFT'::public.grade_scheme_status NOT NULL,
    source_document_id uuid,
    source_title text,
    extraction jsonb,
    extract_job_id uuid,
    confirmed_by uuid,
    confirmed_at timestamp with time zone,
    created_by uuid NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    unclear_count integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT grade_schemes_confirmed_chk CHECK (((status = 'DRAFT'::public.grade_scheme_status) = ((confirmed_at IS NULL) AND (confirmed_by IS NULL)))),
    CONSTRAINT grade_schemes_no_chk CHECK ((scheme_no >= 1)),
    CONSTRAINT grade_schemes_unclear_chk CHECK ((unclear_count >= 0)),
    CONSTRAINT grade_schemes_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.grade_schemes OWNER TO edupilot;

--
-- Name: gradebooks; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.gradebooks (
    course_id uuid NOT NULL,
    status public.gradebook_status DEFAULT 'OPEN'::public.gradebook_status NOT NULL,
    snapshot_version integer DEFAULT 0 NOT NULL,
    unlock_reason text,
    unlocked_by uuid,
    unlocked_at timestamp with time zone,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT gradebooks_snapshot_chk CHECK (((snapshot_version >= 0) AND ((status = 'OPEN'::public.gradebook_status) OR (snapshot_version >= 1)))),
    CONSTRAINT gradebooks_unlock_chk CHECK (((unlock_reason IS NULL) OR ((char_length(unlock_reason) >= 10) AND (char_length(unlock_reason) <= 500)))),
    CONSTRAINT gradebooks_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.gradebooks OWNER TO edupilot;

--
-- Name: grading_guidance_items; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.grading_guidance_items (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    assignment_id uuid NOT NULL,
    criterion_id text NOT NULL,
    text text NOT NULL,
    text_original text,
    direction text NOT NULL,
    support smallint NOT NULL,
    source_result_ids uuid[] DEFAULT '{}'::uuid[] NOT NULL,
    needs_recheck boolean DEFAULT false NOT NULL,
    status public.guidance_item_status DEFAULT 'DRAFT'::public.guidance_item_status NOT NULL,
    pending_retire boolean DEFAULT false NOT NULL,
    approved_in_version smallint,
    retired_in_version smallint,
    origin_job uuid,
    decided_by uuid,
    decided_at timestamp with time zone,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT grading_guidance_items_check CHECK (((status = ANY (ARRAY['APPROVED'::public.guidance_item_status, 'REJECTED'::public.guidance_item_status, 'RETIRED'::public.guidance_item_status])) = (decided_by IS NOT NULL))),
    CONSTRAINT grading_guidance_items_check1 CHECK (((retired_in_version IS NULL) OR ((status = 'RETIRED'::public.guidance_item_status) AND (approved_in_version IS NOT NULL) AND (retired_in_version > approved_in_version)))),
    CONSTRAINT grading_guidance_items_check2 CHECK (((approved_in_version IS NULL) OR (status = ANY (ARRAY['APPROVED'::public.guidance_item_status, 'RETIRED'::public.guidance_item_status])))),
    CONSTRAINT grading_guidance_items_direction_check CHECK ((direction = ANY (ARRAY['STRICTER'::text, 'LENIENT'::text, 'CLARIFY'::text]))),
    CONSTRAINT grading_guidance_items_support_check CHECK ((support >= 0)),
    CONSTRAINT grading_guidance_items_text_check CHECK (((char_length(text) >= 1) AND (char_length(text) <= 400)))
);


ALTER TABLE public.grading_guidance_items OWNER TO edupilot;

--
-- Name: grading_guidance_versions; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.grading_guidance_versions (
    course_id uuid NOT NULL,
    assignment_id uuid NOT NULL,
    version smallint NOT NULL,
    item_count smallint NOT NULL,
    created_by uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT grading_guidance_versions_version_check CHECK ((version >= 1))
);


ALTER TABLE public.grading_guidance_versions OWNER TO edupilot;

--
-- Name: grading_regression_cases; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.grading_regression_cases (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    kind text NOT NULL,
    source_id uuid NOT NULL,
    criterion_id text,
    expected jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT grading_regression_cases_expected_check CHECK ((jsonb_typeof(expected) = 'object'::text)),
    CONSTRAINT grading_regression_cases_kind_check CHECK ((kind = ANY (ARRAY['GRADING_EDIT'::text, 'GRADING_APPEAL'::text, 'THREAD_CORRECTED'::text, 'THREAD_REJECTED'::text])))
);


ALTER TABLE public.grading_regression_cases OWNER TO edupilot;

--
-- Name: grading_results; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.grading_results (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    submission_id uuid NOT NULL,
    round smallint DEFAULT 1 NOT NULL,
    run_state text DEFAULT 'QUEUED'::text NOT NULL,
    status text DEFAULT 'DRAFT'::text NOT NULL,
    attempts smallint DEFAULT 0 NOT NULL,
    not_before timestamp with time zone DEFAULT now() NOT NULL,
    lease_owner text,
    lease_until timestamp with time zone,
    error_code text,
    criteria jsonb DEFAULT '[]'::jsonb NOT NULL,
    overall_feedback text,
    total numeric(5,2),
    ai_scores jsonb,
    ai_total_before_late numeric(5,2),
    late_days smallint DEFAULT 0 NOT NULL,
    late_penalty numeric(5,2) DEFAULT 0 NOT NULL,
    late_penalty_manual boolean DEFAULT false NOT NULL,
    llm_reported_total numeric(5,2),
    flags text[] DEFAULT '{}'::text[] NOT NULL,
    needs_review boolean GENERATED ALWAYS AS ((flags && ARRAY['NEEDS_REVIEW'::text, 'OFF_TOPIC'::text, 'INJECTION_SUSPECTED'::text, 'EMPTY_SUBMISSION'::text, 'EVIDENCE_NOT_FOUND'::text, 'TOTAL_MISMATCH'::text, 'LOW_CONFIDENCE'::text, 'LATE_AFTER_CUTOFF'::text, 'ANSWER_ECHO'::text])) STORED,
    confidence numeric(3,2),
    pass_detail jsonb,
    deviation numeric(5,2),
    teacher_changed_criteria text[] DEFAULT '{}'::text[] NOT NULL,
    reviewed_by uuid,
    reviewed_at timestamp with time zone,
    published_by uuid,
    published_at timestamp with time zone,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    guidance_version smallint DEFAULT 0 NOT NULL,
    ai_comments jsonb,
    first_reviewed_at timestamp with time zone,
    first_reviewed_by uuid,
    CONSTRAINT grading_results_attempts_chk CHECK ((attempts >= 0)),
    CONSTRAINT grading_results_done_chk CHECK (((status = 'DRAFT'::text) OR (run_state = 'DONE'::text))),
    CONSTRAINT grading_results_first_reviewed_chk CHECK (((first_reviewed_at IS NULL) = (first_reviewed_by IS NULL))),
    CONSTRAINT grading_results_flags_chk CHECK ((flags <@ ARRAY['EMPTY_SUBMISSION'::text, 'OFF_TOPIC'::text, 'INJECTION_SUSPECTED'::text, 'NEEDS_REVIEW'::text, 'TOTAL_MISMATCH'::text, 'EVIDENCE_NOT_FOUND'::text, 'SCORE_ADJUSTED'::text, 'LOW_CONFIDENCE'::text, 'EXTRACT_FAILED'::text, 'LATE_AFTER_CUTOFF'::text, 'ANSWER_ECHO'::text])),
    CONSTRAINT grading_results_late_chk CHECK (((late_days >= 0) AND (late_penalty >= (0)::numeric))),
    CONSTRAINT grading_results_published_chk CHECK (((status = 'PUBLISHED'::text) = (published_at IS NOT NULL))),
    CONSTRAINT grading_results_round_chk CHECK (((round >= 1) AND (round <= 20))),
    CONSTRAINT grading_results_run_state_chk CHECK ((run_state = ANY (ARRAY['QUEUED'::text, 'RUNNING'::text, 'DONE'::text, 'FAILED'::text]))),
    CONSTRAINT grading_results_status_chk CHECK ((status = ANY (ARRAY['DRAFT'::text, 'REVIEWED'::text, 'PUBLISHED'::text]))),
    CONSTRAINT grading_results_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.grading_results OWNER TO edupilot;

--
-- Name: idempotency_keys; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.idempotency_keys (
    id uuid DEFAULT uuidv7() NOT NULL,
    user_id uuid NOT NULL,
    endpoint text NOT NULL,
    key text NOT NULL,
    request_hash text NOT NULL,
    status_code smallint NOT NULL,
    response jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.idempotency_keys OWNER TO edupilot;

--
-- Name: insight_reports; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.insight_reports (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    kind text NOT NULL,
    period_from date,
    period_to date,
    payload jsonb NOT NULL,
    data_watermark timestamp with time zone NOT NULL,
    generated_by uuid NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT insight_reports_kind_chk CHECK ((kind = 'KNOWLEDGE_GAP'::text)),
    CONSTRAINT insight_reports_payload_chk CHECK ((jsonb_typeof(payload) = 'object'::text)),
    CONSTRAINT insight_reports_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.insight_reports OWNER TO edupilot;

--
-- Name: integration_configs; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.integration_configs (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    kind text NOT NULL,
    config_public jsonb DEFAULT '{}'::jsonb NOT NULL,
    config_enc bytea,
    enabled boolean DEFAULT false NOT NULL,
    state jsonb DEFAULT '{}'::jsonb NOT NULL,
    last_ok_at timestamp with time zone,
    last_error text,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT integration_configs_kind_chk CHECK ((kind = ANY (ARRAY['IMAP'::text, 'SMTP'::text, 'TEAMS'::text]))),
    CONSTRAINT integration_configs_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.integration_configs OWNER TO edupilot;

--
-- Name: jobs; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.jobs (
    id uuid DEFAULT uuidv7() NOT NULL,
    kind text NOT NULL,
    status public.job_status DEFAULT 'QUEUED'::public.job_status NOT NULL,
    progress smallint DEFAULT 0 NOT NULL,
    result jsonb,
    error jsonb,
    owner_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    finished_at timestamp with time zone,
    CONSTRAINT jobs_finished_chk CHECK (((status = ANY (ARRAY['SUCCEEDED'::public.job_status, 'FAILED'::public.job_status])) = (finished_at IS NOT NULL))),
    CONSTRAINT jobs_progress_chk CHECK (((progress >= 0) AND (progress <= 100)))
);


ALTER TABLE public.jobs OWNER TO edupilot;

--
-- Name: knowledge_items; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.knowledge_items (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    source_type text NOT NULL,
    source_id uuid NOT NULL,
    chunk_id uuid NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT knowledge_items_source_type_chk CHECK ((source_type = ANY (ARRAY['POST'::text, 'TICKET'::text])))
);


ALTER TABLE public.knowledge_items OWNER TO edupilot;

--
-- Name: llm_audit; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.llm_audit (
    id uuid DEFAULT uuidv7() NOT NULL,
    task text NOT NULL,
    lane text NOT NULL,
    provider text,
    model text,
    tokens_in integer DEFAULT 0 NOT NULL,
    tokens_out integer DEFAULT 0 NOT NULL,
    latency_ms integer DEFAULT 0 NOT NULL,
    queue_wait_ms integer DEFAULT 0 NOT NULL,
    attempts integer DEFAULT 1 NOT NULL,
    fallback_index integer DEFAULT 0 NOT NULL,
    cost_est numeric(14,4) DEFAULT 0 NOT NULL,
    status text NOT NULL,
    error_kind text,
    degraded boolean DEFAULT false NOT NULL,
    pii_masked_count integer DEFAULT 0 NOT NULL,
    user_id uuid,
    course_id uuid,
    trace_id text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    cascade_gate text,
    cascade_path text,
    CONSTRAINT llm_audit_gate_chk CHECK (((cascade_gate IS NULL) OR (cascade_gate = ANY (ARRAY['THREADS_PRECHECK'::text, 'INJECTION_FLAG'::text, 'CHAT_ROUTING'::text])))),
    CONSTRAINT llm_audit_lane_chk CHECK ((lane = ANY (ARRAY['INTERACTIVE'::text, 'NEAR_REALTIME'::text, 'BATCH'::text]))),
    CONSTRAINT llm_audit_path_chk CHECK (((cascade_path IS NULL) OR (cascade_path = ANY (ARRAY['FAST'::text, 'FALLBACK'::text, 'FAIL_CLOSED'::text])))),
    CONSTRAINT llm_audit_status_chk CHECK ((status = ANY (ARRAY['ok'::text, 'error'::text, 'timeout'::text, 'rate_limited'::text, 'overloaded'::text, 'degraded'::text, 'cancelled'::text, 'circuit_open'::text, 'budget_blocked'::text, 'not_configured'::text])))
);


ALTER TABLE public.llm_audit OWNER TO edupilot;

--
-- Name: llm_audit_hourly; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.llm_audit_hourly (
    hour timestamp with time zone NOT NULL,
    task text NOT NULL,
    requests bigint NOT NULL,
    errors bigint NOT NULL,
    fallbacks bigint NOT NULL,
    cost numeric(14,4) NOT NULL,
    tokens_in bigint NOT NULL,
    tokens_out bigint NOT NULL,
    pii_masked bigint NOT NULL
);


ALTER TABLE public.llm_audit_hourly OWNER TO edupilot;

--
-- Name: llm_audit_hourly_lat; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.llm_audit_hourly_lat (
    hour timestamp with time zone NOT NULL,
    bucket smallint NOT NULL,
    n bigint NOT NULL,
    CONSTRAINT llm_audit_hourly_lat_bucket_check CHECK (((bucket >= 0) AND (bucket <= 139)))
);


ALTER TABLE public.llm_audit_hourly_lat OWNER TO edupilot;

--
-- Name: llm_audit_rollup_state; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.llm_audit_rollup_state (
    id boolean DEFAULT true NOT NULL,
    watermark timestamp with time zone NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT llm_audit_rollup_state_id_check CHECK (id)
);


ALTER TABLE public.llm_audit_rollup_state OWNER TO edupilot;

--
-- Name: llm_budgets; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.llm_budgets (
    id uuid DEFAULT uuidv7() NOT NULL,
    scope text NOT NULL,
    course_id uuid,
    daily_limit numeric(14,2),
    monthly_limit numeric(14,2),
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT llm_budgets_daily_chk CHECK (((daily_limit IS NULL) OR (daily_limit >= (0)::numeric))),
    CONSTRAINT llm_budgets_monthly_chk CHECK (((monthly_limit IS NULL) OR (monthly_limit >= (0)::numeric))),
    CONSTRAINT llm_budgets_order_chk CHECK (((daily_limit IS NULL) OR (monthly_limit IS NULL) OR (daily_limit <= monthly_limit))),
    CONSTRAINT llm_budgets_scope_chk CHECK ((scope = ANY (ARRAY['system'::text, 'course'::text]))),
    CONSTRAINT llm_budgets_scope_course_chk CHECK ((((scope = 'system'::text) AND (course_id IS NULL)) OR ((scope = 'course'::text) AND (course_id IS NOT NULL))))
);


ALTER TABLE public.llm_budgets OWNER TO edupilot;

--
-- Name: llm_models; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.llm_models (
    id uuid DEFAULT uuidv7() NOT NULL,
    provider_id uuid NOT NULL,
    model text NOT NULL,
    kind text NOT NULL,
    dims integer,
    price_in numeric(14,4) DEFAULT 0 NOT NULL,
    price_out numeric(14,4) DEFAULT 0 NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT llm_models_dims_chk CHECK (((kind <> 'embedding'::text) OR (dims IS NOT NULL))),
    CONSTRAINT llm_models_dims_pos_chk CHECK (((dims IS NULL) OR (dims > 0))),
    CONSTRAINT llm_models_kind_chk CHECK ((kind = ANY (ARRAY['chat'::text, 'embedding'::text, 'decision'::text]))),
    CONSTRAINT llm_models_model_chk CHECK (((char_length(model) >= 1) AND (char_length(model) <= 120))),
    CONSTRAINT llm_models_price_in_chk CHECK ((price_in >= (0)::numeric)),
    CONSTRAINT llm_models_price_out_chk CHECK ((price_out >= (0)::numeric))
);


ALTER TABLE public.llm_models OWNER TO edupilot;

--
-- Name: llm_providers; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.llm_providers (
    id uuid DEFAULT uuidv7() NOT NULL,
    type text NOT NULL,
    name text NOT NULL,
    base_url text,
    api_key_enc bytea,
    enabled boolean DEFAULT true NOT NULL,
    rpm_limit integer,
    tpm_limit integer,
    last_test_ok boolean,
    last_test_at timestamp with time zone,
    last_test_error text,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT llm_providers_base_url_chk CHECK (((type <> 'openai_compatible'::text) OR (base_url IS NOT NULL))),
    CONSTRAINT llm_providers_name_chk CHECK (((char_length(name) >= 1) AND (char_length(name) <= 60))),
    CONSTRAINT llm_providers_rpm_chk CHECK (((rpm_limit IS NULL) OR (rpm_limit > 0))),
    CONSTRAINT llm_providers_tpm_chk CHECK (((tpm_limit IS NULL) OR (tpm_limit > 0))),
    CONSTRAINT llm_providers_type_chk CHECK ((type = ANY (ARRAY['openai'::text, 'anthropic'::text, 'gemini'::text, 'openai_compatible'::text, 'fake'::text, 'decision'::text])))
);


ALTER TABLE public.llm_providers OWNER TO edupilot;

--
-- Name: llm_request_payloads; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.llm_request_payloads (
    audit_id uuid NOT NULL,
    course_id uuid,
    masked_prompt text NOT NULL,
    tools jsonb,
    confidence numeric(4,3),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT llm_request_payloads_confidence_chk CHECK (((confidence IS NULL) OR ((confidence >= (0)::numeric) AND (confidence <= (1)::numeric))))
);


ALTER TABLE public.llm_request_payloads OWNER TO edupilot;

--
-- Name: llm_task_routes; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.llm_task_routes (
    id uuid DEFAULT uuidv7() NOT NULL,
    task text NOT NULL,
    model_id uuid NOT NULL,
    fallback_order integer NOT NULL,
    params jsonb DEFAULT '{}'::jsonb NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT llm_task_routes_order_chk CHECK ((fallback_order >= 0)),
    CONSTRAINT llm_task_routes_task_chk CHECK ((task = ANY (ARRAY['CHAT'::text, 'CLASSIFY'::text, 'UTILITY'::text, 'GRADING'::text, 'QUESTION_GEN'::text, 'INSIGHT'::text, 'EMBEDDING'::text])))
);


ALTER TABLE public.llm_task_routes OWNER TO edupilot;

--
-- Name: login_attempts; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.login_attempts (
    id uuid DEFAULT uuidv7() NOT NULL,
    email_hash character(64) NOT NULL,
    user_id uuid,
    ip inet,
    user_agent text,
    outcome public.login_outcome NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT login_attempts_email_hash_chk CHECK ((email_hash ~ '^[0-9a-f]{64}$'::text)),
    CONSTRAINT login_attempts_user_agent_chk CHECK ((char_length(user_agent) <= 200))
);


ALTER TABLE public.login_attempts OWNER TO edupilot;

--
-- Name: mail_inbound; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.mail_inbound (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    uidvalidity bigint NOT NULL,
    message_uid bigint NOT NULL,
    message_id text,
    from_email text NOT NULL,
    subject text DEFAULT ''::text NOT NULL,
    received_at timestamp with time zone NOT NULL,
    status text NOT NULL,
    reason text,
    submission_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT mail_inbound_reason_chk CHECK ((reason = ANY (ARRAY['AUTO_REPLY'::text, 'NO_ATTACHMENT'::text, 'TOO_MANY_ATTACHMENTS'::text, 'ATTACHMENT_REJECTED'::text, 'PARSE_ERROR'::text]))),
    CONSTRAINT mail_inbound_status_chk CHECK ((status = ANY (ARRAY['ACCEPTED'::text, 'IGNORED'::text, 'FAILED'::text])))
);


ALTER TABLE public.mail_inbound OWNER TO edupilot;

--
-- Name: mail_outbox; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.mail_outbox (
    id uuid DEFAULT uuidv7() NOT NULL,
    to_addr text NOT NULL,
    template text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    status public.mail_status DEFAULT 'QUEUED'::public.mail_status NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    last_error text,
    dedupe_key text,
    sent_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT mail_outbox_attempts_chk CHECK (((attempts >= 0) AND (attempts <= 4))),
    CONSTRAINT mail_outbox_last_error_chk CHECK ((char_length(last_error) <= 1000)),
    CONSTRAINT mail_outbox_payload_chk CHECK ((jsonb_typeof(payload) = 'object'::text)),
    CONSTRAINT mail_outbox_sent_chk CHECK (((status = 'SENT'::public.mail_status) = (sent_at IS NOT NULL))),
    CONSTRAINT mail_outbox_template_chk CHECK ((template ~ '^[a-z][a-z0-9_]{0,63}$'::text)),
    CONSTRAINT mail_outbox_to_addr_chk CHECK (((to_addr = lower(to_addr)) AND (to_addr ~ '^[^@\s]+@[^@\s]+$'::text)))
);


ALTER TABLE public.mail_outbox OWNER TO edupilot;

--
-- Name: notifications; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.notifications (
    id uuid DEFAULT uuidv7() NOT NULL,
    user_id uuid NOT NULL,
    course_id uuid,
    type text NOT NULL,
    title text NOT NULL,
    body text,
    link text,
    dedupe_key text,
    read_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT notifications_body_chk CHECK ((char_length(body) <= 1000)),
    CONSTRAINT notifications_link_chk CHECK (((link IS NULL) OR (link ~ '^/[^/\\]'::text))),
    CONSTRAINT notifications_title_chk CHECK (((char_length(title) >= 1) AND (char_length(title) <= 200))),
    CONSTRAINT notifications_type_chk CHECK ((type ~ '^[A-Z][A-Z0-9_]{0,39}$'::text))
);


ALTER TABLE public.notifications OWNER TO edupilot;

--
-- Name: outbox; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.outbox (
    id uuid DEFAULT uuidv7() NOT NULL,
    topic text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    next_attempt_at timestamp with time zone DEFAULT now() NOT NULL,
    enqueued_at timestamp with time zone,
    dispatched_at timestamp with time zone,
    attempts integer DEFAULT 0 NOT NULL,
    last_error text,
    dead_at timestamp with time zone,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT outbox_attempts_chk CHECK (((attempts >= 0) AND (attempts <= 4))),
    CONSTRAINT outbox_terminal_chk CHECK ((NOT ((dispatched_at IS NOT NULL) AND (dead_at IS NOT NULL)))),
    CONSTRAINT outbox_topic_chk CHECK ((topic ~ '^[a-z][a-z0-9_.]{0,63}$'::text))
);


ALTER TABLE public.outbox OWNER TO edupilot;

--
-- Name: participation_events; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.participation_events (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    student_id uuid NOT NULL,
    session_id uuid,
    type public.participation_type NOT NULL,
    points numeric(4,2) NOT NULL,
    note text,
    client_event_id uuid NOT NULL,
    created_by uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT participation_events_bonus_chk CHECK (((type <> 'BONUS'::public.participation_type) OR (points > (0)::numeric))),
    CONSTRAINT participation_events_note_chk CHECK ((char_length(note) <= 200)),
    CONSTRAINT participation_events_penalty_chk CHECK (((type <> 'PENALTY'::public.participation_type) OR (points < (0)::numeric))),
    CONSTRAINT participation_events_points_chk CHECK (((points >= ('-5'::integer)::numeric) AND (points <= (5)::numeric))),
    CONSTRAINT participation_events_speak_chk CHECK (((type <> 'SPEAK'::public.participation_type) OR (points >= (0)::numeric)))
);


ALTER TABLE public.participation_events OWNER TO edupilot;

--
-- Name: pii_events; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.pii_events (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    session_id uuid,
    user_id uuid NOT NULL,
    channel public.chat_channel NOT NULL,
    pii_type public.pii_kind NOT NULL,
    count integer NOT NULL,
    action public.pii_action NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT pii_events_count_chk CHECK ((count >= 1))
);


ALTER TABLE public.pii_events OWNER TO edupilot;

--
-- Name: post_reports; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.post_reports (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    thread_id uuid,
    post_id uuid,
    target_id uuid GENERATED ALWAYS AS (COALESCE(thread_id, post_id)) STORED,
    reporter_id uuid NOT NULL,
    reason public.post_report_reason NOT NULL,
    note text,
    status public.post_report_status DEFAULT 'OPEN'::public.post_report_status NOT NULL,
    handled_by uuid,
    handled_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT post_reports_handled_chk CHECK (((handled_by IS NULL) = (handled_at IS NULL))),
    CONSTRAINT post_reports_note_chk CHECK (((char_length(note) <= 200) AND ((note IS NULL) OR (reason = 'OTHER'::public.post_report_reason)))),
    CONSTRAINT post_reports_target_chk CHECK (((thread_id IS NULL) <> (post_id IS NULL)))
);


ALTER TABLE public.post_reports OWNER TO edupilot;

--
-- Name: practice_answers; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.practice_answers (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    attempt_id uuid NOT NULL,
    item_id uuid NOT NULL,
    answer jsonb NOT NULL,
    is_correct boolean,
    answered_at timestamp with time zone DEFAULT now() NOT NULL,
    locked boolean DEFAULT false NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    CONSTRAINT practice_answers_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.practice_answers OWNER TO edupilot;

--
-- Name: practice_attempts; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.practice_attempts (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    student_id uuid NOT NULL,
    mode public.practice_mode NOT NULL,
    topic text,
    status public.practice_status DEFAULT 'IN_PROGRESS'::public.practice_status NOT NULL,
    started_at timestamp with time zone DEFAULT now() NOT NULL,
    deadline_at timestamp with time zone,
    submitted_at timestamp with time zone,
    item_count integer NOT NULL,
    current_index integer DEFAULT 0 NOT NULL,
    correct_count integer,
    score numeric(3,1),
    matrix jsonb,
    notes jsonb DEFAULT '[]'::jsonb NOT NULL,
    tab_id text,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT practice_attempts_closed_chk CHECK ((((status = 'IN_PROGRESS'::public.practice_status) = (submitted_at IS NULL)) AND ((status = 'IN_PROGRESS'::public.practice_status) = (correct_count IS NULL)))),
    CONSTRAINT practice_attempts_correct_chk CHECK (((correct_count IS NULL) OR ((correct_count >= 0) AND (correct_count <= item_count)))),
    CONSTRAINT practice_attempts_count_chk CHECK (((item_count >= 1) AND (item_count <= 50))),
    CONSTRAINT practice_attempts_deadline_chk CHECK (((mode = 'MOCK'::public.practice_mode) = (deadline_at IS NOT NULL))),
    CONSTRAINT practice_attempts_index_chk CHECK (((current_index >= 0) AND (current_index < item_count))),
    CONSTRAINT practice_attempts_notes_chk CHECK ((jsonb_typeof(notes) = 'array'::text)),
    CONSTRAINT practice_attempts_score_chk CHECK (((score IS NULL) OR ((score >= (0)::numeric) AND (score <= (10)::numeric)))),
    CONSTRAINT practice_attempts_topic_chk CHECK (((mode = 'MOCK'::public.practice_mode) OR (topic IS NOT NULL))),
    CONSTRAINT practice_attempts_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.practice_attempts OWNER TO edupilot;

--
-- Name: practice_items; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.practice_items (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    attempt_id uuid NOT NULL,
    question_id uuid NOT NULL,
    "position" integer NOT NULL,
    option_order jsonb DEFAULT '[]'::jsonb NOT NULL,
    answer_key jsonb NOT NULL,
    question jsonb NOT NULL,
    question_version integer NOT NULL,
    CONSTRAINT practice_items_order_chk CHECK ((jsonb_typeof(option_order) = 'array'::text)),
    CONSTRAINT practice_items_position_chk CHECK (("position" >= 0))
);


ALTER TABLE public.practice_items OWNER TO edupilot;

--
-- Name: question_bank; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.question_bank (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    type public.question_type NOT NULL,
    title text NOT NULL,
    topic text NOT NULL,
    difficulty public.question_difficulty DEFAULT 'MEDIUM'::public.question_difficulty NOT NULL,
    stem text NOT NULL,
    answer_key jsonb,
    explanation text,
    citations jsonb DEFAULT '[]'::jsonb NOT NULL,
    origin public.question_origin DEFAULT 'MANUAL'::public.question_origin NOT NULL,
    review_status public.question_review_status DEFAULT 'DRAFT'::public.question_review_status NOT NULL,
    created_by uuid NOT NULL,
    reviewed_by uuid,
    reviewed_at timestamp with time zone,
    ai_job_id uuid,
    archived_at timestamp with time zone,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    reject_reason text,
    CONSTRAINT question_bank_answer_key_chk CHECK ((((type = ANY (ARRAY['MCQ_SINGLE'::public.question_type, 'MCQ_MULTI'::public.question_type, 'TRUE_FALSE'::public.question_type])) = (answer_key IS NOT NULL)) OR (type = ANY (ARRAY['SHORT'::public.question_type, 'ESSAY'::public.question_type])))),
    CONSTRAINT question_bank_explanation_chk CHECK ((char_length(explanation) <= 4000)),
    CONSTRAINT question_bank_reject_reason_check CHECK ((reject_reason = ANY (ARRAY['MO_HO'::text, 'SAI_DAP_AN'::text, 'TRUNG_LAP'::text, 'LAC_DE'::text, 'KHO_DE_BAT_THUONG'::text, 'SAI_DINH_DANG'::text, 'UNSPECIFIED'::text]))),
    CONSTRAINT question_bank_reject_reason_chk CHECK (((reject_reason IS NULL) OR (review_status = 'REJECTED'::public.question_review_status))),
    CONSTRAINT question_bank_review_chk CHECK (((review_status = ANY (ARRAY['APPROVED'::public.question_review_status, 'REJECTED'::public.question_review_status])) = (reviewed_by IS NOT NULL))),
    CONSTRAINT question_bank_stem_chk CHECK (((char_length(stem) >= 1) AND (char_length(stem) <= 8000))),
    CONSTRAINT question_bank_title_chk CHECK (((char_length(title) >= 1) AND (char_length(title) <= 120))),
    CONSTRAINT question_bank_topic_chk CHECK (((char_length(topic) >= 1) AND (char_length(topic) <= 80))),
    CONSTRAINT question_bank_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.question_bank OWNER TO edupilot;

--
-- Name: question_options; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.question_options (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    question_id uuid NOT NULL,
    "position" smallint NOT NULL,
    body text NOT NULL,
    pinned_last boolean DEFAULT false NOT NULL,
    CONSTRAINT question_options_body_chk CHECK (((char_length(body) >= 1) AND (char_length(body) <= 1000))),
    CONSTRAINT question_options_position_chk CHECK ((("position" >= 1) AND ("position" <= 8)))
);


ALTER TABLE public.question_options OWNER TO edupilot;

--
-- Name: reminder_log; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.reminder_log (
    id uuid DEFAULT uuidv7() NOT NULL,
    user_id uuid NOT NULL,
    course_id uuid NOT NULL,
    source_type public.reminder_source NOT NULL,
    source_id uuid NOT NULL,
    starts_at timestamp with time zone NOT NULL,
    kind text DEFAULT 'T24H'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT reminder_log_kind_chk CHECK ((kind = 'T24H'::text))
);


ALTER TABLE public.reminder_log OWNER TO edupilot;

--
-- Name: similarity_reports; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.similarity_reports (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    exam_id uuid NOT NULL,
    problem_id uuid NOT NULL,
    run_id uuid NOT NULL,
    submission_a uuid NOT NULL,
    submission_b uuid NOT NULL,
    attempt_a uuid NOT NULL,
    attempt_b uuid NOT NULL,
    score numeric(4,3) NOT NULL,
    shared_fingerprints integer NOT NULL,
    flagged boolean DEFAULT false NOT NULL,
    algorithm text DEFAULT 'winnow-k5-w4-jaccard'::text NOT NULL,
    review_state public.similarity_review_state DEFAULT 'NEW'::public.similarity_review_state NOT NULL,
    reviewed_by uuid,
    reviewed_at timestamp with time zone,
    note text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT similarity_reports_note_chk CHECK ((char_length(note) <= 500)),
    CONSTRAINT similarity_reports_pair_chk CHECK ((attempt_a < attempt_b)),
    CONSTRAINT similarity_reports_review_chk CHECK (((review_state = 'NEW'::public.similarity_review_state) = (reviewed_by IS NULL))),
    CONSTRAINT similarity_reports_score_chk CHECK (((score >= (0)::numeric) AND (score <= (1)::numeric)))
);


ALTER TABLE public.similarity_reports OWNER TO edupilot;

--
-- Name: student_observations; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.student_observations (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    student_id uuid NOT NULL,
    author_id uuid NOT NULL,
    kind public.observation_kind DEFAULT 'NOTE'::public.observation_kind NOT NULL,
    tags text[] DEFAULT '{}'::text[] NOT NULL,
    body text NOT NULL,
    ai_edited boolean DEFAULT false NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT student_observations_ai_edited_chk CHECK (((kind <> 'NOTE'::public.observation_kind) OR (NOT ai_edited))),
    CONSTRAINT student_observations_body_chk CHECK (((char_length(body) >= 1) AND (char_length(body) <= 2000))),
    CONSTRAINT student_observations_tags_chk CHECK (public.forum_tags_valid(tags)),
    CONSTRAINT student_observations_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.student_observations OWNER TO edupilot;

--
-- Name: student_risk; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.student_risk (
    course_id uuid NOT NULL,
    student_id uuid NOT NULL,
    level public.risk_level DEFAULT 'NONE'::public.risk_level NOT NULL,
    reasons jsonb DEFAULT '[]'::jsonb NOT NULL,
    weeks jsonb DEFAULT '[]'::jsonb NOT NULL,
    last_week_asks integer DEFAULT 0 NOT NULL,
    last_week_posts integer DEFAULT 0 NOT NULL,
    entered_at timestamp with time zone,
    computed_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT student_risk_counts_chk CHECK (((last_week_asks >= 0) AND (last_week_posts >= 0))),
    CONSTRAINT student_risk_entered_chk CHECK (((level = 'AT_RISK'::public.risk_level) = (entered_at IS NOT NULL))),
    CONSTRAINT student_risk_reasons_chk CHECK ((jsonb_typeof(reasons) = 'array'::text)),
    CONSTRAINT student_risk_weeks_chk CHECK ((jsonb_typeof(weeks) = 'array'::text))
);


ALTER TABLE public.student_risk OWNER TO edupilot;

--
-- Name: study_time_daily; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.study_time_daily (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    student_id uuid NOT NULL,
    date date NOT NULL,
    area text NOT NULL,
    active_seconds integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT study_time_daily_area_chk CHECK ((area = ANY (ARRAY['CHAT'::text, 'THREADS'::text, 'PRACTICE'::text, 'LIBRARY'::text, 'CALENDAR'::text]))),
    CONSTRAINT study_time_daily_seconds_chk CHECK (((active_seconds >= 0) AND (active_seconds <= 86400)))
);


ALTER TABLE public.study_time_daily OWNER TO edupilot;

--
-- Name: submissions; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.submissions (
    id uuid DEFAULT uuidv7() NOT NULL,
    course_id uuid NOT NULL,
    assignment_id uuid,
    student_id uuid,
    candidate_student_id uuid,
    source text NOT NULL,
    version integer,
    blob_key text NOT NULL,
    file_name text NOT NULL,
    file_size bigint NOT NULL,
    file_mime text NOT NULL,
    sha256 text NOT NULL,
    extracted_text text,
    extracted_at timestamp with time zone,
    extract_status text,
    external_ref text,
    external_ts timestamp with time zone,
    from_email text,
    subject text,
    sender_verified boolean DEFAULT false NOT NULL,
    match_status text NOT NULL,
    match_method text,
    unmatched_reason text,
    dismissed_at timestamp with time zone,
    dismissed_by uuid,
    dismissed_reason text,
    submitted_at timestamp with time zone,
    received_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT submissions_dismissed_chk CHECK (((dismissed_at IS NULL) OR (match_status = 'UNMATCHED'::text))),
    CONSTRAINT submissions_extract_status_chk CHECK ((extract_status = ANY (ARRAY['OK'::text, 'FAILED'::text]))),
    CONSTRAINT submissions_file_size_chk CHECK (((file_size >= 1) AND (file_size <= 20971520))),
    CONSTRAINT submissions_match_method_chk CHECK ((match_method = ANY (ARRAY['EMAIL'::text, 'CODE'::text, 'IN_APP'::text, 'MANUAL'::text]))),
    CONSTRAINT submissions_match_status_chk CHECK ((match_status = ANY (ARRAY['MATCHED'::text, 'UNMATCHED'::text]))),
    CONSTRAINT submissions_matched_chk CHECK (((match_status = 'MATCHED'::text) = ((assignment_id IS NOT NULL) AND (student_id IS NOT NULL) AND (match_method IS NOT NULL) AND (version IS NOT NULL)))),
    CONSTRAINT submissions_source_chk CHECK ((source = ANY (ARRAY['IN_APP'::text, 'UPLOAD'::text, 'IMAP'::text, 'TEAMS'::text, 'FORMS'::text]))),
    CONSTRAINT submissions_unmatched_chk CHECK (((match_status = 'UNMATCHED'::text) = (unmatched_reason IS NOT NULL))),
    CONSTRAINT submissions_unmatched_reason_chk CHECK ((unmatched_reason = ANY (ARRAY['NO_ASSIGNMENT_CODE'::text, 'AMBIGUOUS_ASSIGNMENT'::text, 'NO_STUDENT_SIGNAL'::text, 'EMAIL_CODE_CONFLICT'::text, 'AMBIGUOUS_STUDENT_CODE'::text, 'STUDENT_NOT_IN_COURSE'::text, 'CODE_NOT_VERIFIED'::text, 'SENDER_UNVERIFIED'::text, 'ALREADY_REVIEWED'::text]))),
    CONSTRAINT submissions_version_chk CHECK (((version IS NULL) OR (version >= 1)))
);


ALTER TABLE public.submissions OWNER TO edupilot;

--
-- Name: user_devices; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.user_devices (
    id uuid DEFAULT uuidv7() NOT NULL,
    key_sha256 bytea NOT NULL,
    public_key bytea NOT NULL,
    kind public.device_kind NOT NULL,
    attested boolean DEFAULT false NOT NULL,
    boot_verified boolean,
    security_level text,
    label text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT user_devices_label_chk CHECK ((label IS NULL)),
    CONSTRAINT user_devices_pubkey_chk CHECK ((octet_length(public_key) <= 512))
);


ALTER TABLE public.user_devices OWNER TO edupilot;

--
-- Name: user_settings; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.user_settings (
    user_id uuid NOT NULL,
    notify_ticket_by_mail boolean DEFAULT false NOT NULL,
    notify_answer_by_mail boolean DEFAULT true NOT NULL,
    remind_deadline_by_mail boolean DEFAULT true NOT NULL,
    preferences jsonb DEFAULT '{}'::jsonb NOT NULL,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    track_study_time boolean DEFAULT true NOT NULL,
    CONSTRAINT user_settings_preferences_chk CHECK ((jsonb_typeof(preferences) = 'object'::text)),
    CONSTRAINT user_settings_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.user_settings OWNER TO edupilot;

--
-- Name: users; Type: TABLE; Schema: public; Owner: edupilot
--

CREATE TABLE public.users (
    id uuid DEFAULT uuidv7() NOT NULL,
    email text NOT NULL,
    password_hash text,
    full_name text NOT NULL,
    role public.user_role NOT NULL,
    student_code text,
    email_verified_at timestamp with time zone,
    failed_logins integer DEFAULT 0 NOT NULL,
    locked_until timestamp with time zone,
    status public.user_status DEFAULT 'INVITED'::public.user_status NOT NULL,
    ics_token text,
    tracking_notice_ack_at timestamp with time zone,
    last_login_at timestamp with time zone,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT users_active_password_chk CHECK (((status <> 'ACTIVE'::public.user_status) OR ((password_hash IS NOT NULL) AND (password_hash <> ''::text)))),
    CONSTRAINT users_email_lower_chk CHECK ((email = lower(email))),
    CONSTRAINT users_failed_logins_chk CHECK ((failed_logins >= 0)),
    CONSTRAINT users_ics_token_hash_chk CHECK (((ics_token IS NULL) OR (ics_token ~ '^[0-9a-f]{64}$'::text))),
    CONSTRAINT users_student_code_role_chk CHECK (((student_code IS NULL) OR (role = 'STUDENT'::public.user_role))),
    CONSTRAINT users_version_chk CHECK ((version >= 1))
);


ALTER TABLE public.users OWNER TO edupilot;

--
-- Name: v_attendance_summary; Type: VIEW; Schema: public; Owner: edupilot
--

CREATE VIEW public.v_attendance_summary AS
 SELECT course_id,
    student_id,
    (count(*))::integer AS recorded,
    (count(*) FILTER (WHERE (status = 'PRESENT'::public.attendance_status)))::integer AS present,
    (count(*) FILTER (WHERE (status = 'LATE'::public.attendance_status)))::integer AS late,
    (count(*) FILTER (WHERE (status = 'EXCUSED'::public.attendance_status)))::integer AS excused,
    (count(*) FILTER (WHERE (status = 'ABSENT'::public.attendance_status)))::integer AS absent,
    max(updated_at) AS last_marked_at
   FROM public.attendance_records
  GROUP BY course_id, student_id;


ALTER VIEW public.v_attendance_summary OWNER TO edupilot;

--
-- Name: v_student_360; Type: VIEW; Schema: public; Owner: edupilot
--

CREATE VIEW public.v_student_360 AS
 SELECT e.course_id,
    e.user_id AS student_id,
    u.full_name,
    e.student_code_snapshot AS student_code,
    e.created_at AS joined_at,
    COALESCE(a.recorded, 0) AS recorded,
    COALESCE(a.present, 0) AS present,
    COALESCE(a.late, 0) AS late,
    COALESCE(a.excused, 0) AS excused,
    COALESCE(a.absent, 0) AS absent,
    COALESCE(p.speak_count, 0) AS speak_count,
    (COALESCE(p.points_total, (0)::numeric))::numeric(8,2) AS points_total,
    COALESCE(st.seconds, (0)::bigint) AS study_seconds_7d,
    COALESCE(r.last_week_asks, 0) AS last_week_asks,
    COALESCE(r.last_week_posts, 0) AS last_week_posts,
    COALESCE(r.level, 'NONE'::public.risk_level) AS risk_level,
    r.entered_at AS risk_entered_at,
    r.computed_at AS risk_computed_at
   FROM (((((public.enrollments e
     JOIN public.users u ON ((u.id = e.user_id)))
     LEFT JOIN public.v_attendance_summary a ON (((a.course_id = e.course_id) AND (a.student_id = e.user_id))))
     LEFT JOIN ( SELECT participation_events.course_id,
            participation_events.student_id,
            (count(*) FILTER (WHERE (participation_events.type = 'SPEAK'::public.participation_type)))::integer AS speak_count,
            sum(participation_events.points) AS points_total
           FROM public.participation_events
          GROUP BY participation_events.course_id, participation_events.student_id) p ON (((p.course_id = e.course_id) AND (p.student_id = e.user_id))))
     LEFT JOIN ( SELECT study_time_daily.course_id,
            study_time_daily.student_id,
            sum(study_time_daily.active_seconds) AS seconds
           FROM public.study_time_daily
          WHERE (study_time_daily.date > (((now() AT TIME ZONE 'Asia/Ho_Chi_Minh'::text))::date - 7))
          GROUP BY study_time_daily.course_id, study_time_daily.student_id) st ON (((st.course_id = e.course_id) AND (st.student_id = e.user_id))))
     LEFT JOIN public.student_risk r ON (((r.course_id = e.course_id) AND (r.student_id = e.user_id))))
  WHERE ((e.role_in_course = 'STUDENT'::public.enrollment_role) AND (e.status = 'ACTIVE'::public.enrollment_status));


ALTER VIEW public.v_student_360 OWNER TO edupilot;

--
-- Name: assignments assignments_code_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.assignments
    ADD CONSTRAINT assignments_code_key UNIQUE (course_id, submission_code);


--
-- Name: assignments assignments_course_id_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.assignments
    ADD CONSTRAINT assignments_course_id_key UNIQUE (course_id, id);


--
-- Name: assignments assignments_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.assignments
    ADD CONSTRAINT assignments_pkey PRIMARY KEY (id);


--
-- Name: attendance_checkins attendance_checkins_natural_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.attendance_checkins
    ADD CONSTRAINT attendance_checkins_natural_key UNIQUE (course_id, window_id, student_id);


--
-- Name: attendance_checkins attendance_checkins_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.attendance_checkins
    ADD CONSTRAINT attendance_checkins_pkey PRIMARY KEY (id);


--
-- Name: attendance_records attendance_records_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.attendance_records
    ADD CONSTRAINT attendance_records_pkey PRIMARY KEY (id);


--
-- Name: attendance_records attendance_records_session_student_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.attendance_records
    ADD CONSTRAINT attendance_records_session_student_key UNIQUE (course_id, session_id, student_id);


--
-- Name: audit_log audit_log_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.audit_log
    ADD CONSTRAINT audit_log_pkey PRIMARY KEY (id);


--
-- Name: auth_sessions auth_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.auth_sessions
    ADD CONSTRAINT auth_sessions_pkey PRIMARY KEY (id);


--
-- Name: auth_tokens auth_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.auth_tokens
    ADD CONSTRAINT auth_tokens_pkey PRIMARY KEY (id);


--
-- Name: calendar_events calendar_events_course_id_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.calendar_events
    ADD CONSTRAINT calendar_events_course_id_key UNIQUE (course_id, id);


--
-- Name: calendar_events calendar_events_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.calendar_events
    ADD CONSTRAINT calendar_events_pkey PRIMARY KEY (id);


--
-- Name: chat_messages chat_messages_course_id_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.chat_messages
    ADD CONSTRAINT chat_messages_course_id_key UNIQUE (course_id, id);


--
-- Name: chat_messages chat_messages_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.chat_messages
    ADD CONSTRAINT chat_messages_pkey PRIMARY KEY (id);


--
-- Name: chat_sessions chat_sessions_course_id_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.chat_sessions
    ADD CONSTRAINT chat_sessions_course_id_key UNIQUE (course_id, id);


--
-- Name: chat_sessions chat_sessions_id_user_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.chat_sessions
    ADD CONSTRAINT chat_sessions_id_user_key UNIQUE (id, user_id);


--
-- Name: chat_sessions chat_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.chat_sessions
    ADD CONSTRAINT chat_sessions_pkey PRIMARY KEY (id);


--
-- Name: checkin_windows checkin_windows_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.checkin_windows
    ADD CONSTRAINT checkin_windows_pkey PRIMARY KEY (id);


--
-- Name: checkin_windows checkin_windows_round_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.checkin_windows
    ADD CONSTRAINT checkin_windows_round_key UNIQUE (course_id, session_id, round);


--
-- Name: class_sessions class_sessions_course_id_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.class_sessions
    ADD CONSTRAINT class_sessions_course_id_key UNIQUE (course_id, id);


--
-- Name: class_sessions class_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.class_sessions
    ADD CONSTRAINT class_sessions_pkey PRIMARY KEY (id);


--
-- Name: code_drafts code_drafts_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.code_drafts
    ADD CONSTRAINT code_drafts_pkey PRIMARY KEY (attempt_id, item_id, language);


--
-- Name: code_problems code_problems_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.code_problems
    ADD CONSTRAINT code_problems_pkey PRIMARY KEY (question_id);


--
-- Name: code_submissions code_submissions_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.code_submissions
    ADD CONSTRAINT code_submissions_pkey PRIMARY KEY (id);


--
-- Name: code_testcases code_testcases_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.code_testcases
    ADD CONSTRAINT code_testcases_pkey PRIMARY KEY (id);


--
-- Name: code_testcases code_testcases_problem_position_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.code_testcases
    ADD CONSTRAINT code_testcases_problem_position_key UNIQUE (problem_id, "position") DEFERRABLE INITIALLY DEFERRED;


--
-- Name: content_chunks content_chunks_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.content_chunks
    ADD CONSTRAINT content_chunks_pkey PRIMARY KEY (id);


--
-- Name: course_decision_gates course_decision_gates_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.course_decision_gates
    ADD CONSTRAINT course_decision_gates_pkey PRIMARY KEY (course_id, gate);


--
-- Name: course_guidance_flags course_guidance_flags_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.course_guidance_flags
    ADD CONSTRAINT course_guidance_flags_pkey PRIMARY KEY (course_id, feature);


--
-- Name: courses courses_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.courses
    ADD CONSTRAINT courses_pkey PRIMARY KEY (id);


--
-- Name: decision_gates decision_gates_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.decision_gates
    ADD CONSTRAINT decision_gates_pkey PRIMARY KEY (gate);


--
-- Name: document_courses document_courses_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.document_courses
    ADD CONSTRAINT document_courses_pkey PRIMARY KEY (document_id, course_id);


--
-- Name: documents documents_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.documents
    ADD CONSTRAINT documents_pkey PRIMARY KEY (id);


--
-- Name: enrollments enrollments_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.enrollments
    ADD CONSTRAINT enrollments_pkey PRIMARY KEY (id);


--
-- Name: escalation_tickets escalation_tickets_course_id_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.escalation_tickets
    ADD CONSTRAINT escalation_tickets_course_id_key UNIQUE (course_id, id);


--
-- Name: escalation_tickets escalation_tickets_message_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.escalation_tickets
    ADD CONSTRAINT escalation_tickets_message_key UNIQUE (message_id);


--
-- Name: escalation_tickets escalation_tickets_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.escalation_tickets
    ADD CONSTRAINT escalation_tickets_pkey PRIMARY KEY (id);


--
-- Name: exam_answers exam_answers_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exam_answers
    ADD CONSTRAINT exam_answers_pkey PRIMARY KEY (attempt_id, item_id);


--
-- Name: exam_appeals exam_appeals_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exam_appeals
    ADD CONSTRAINT exam_appeals_pkey PRIMARY KEY (id);


--
-- Name: exam_attempts exam_attempts_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exam_attempts
    ADD CONSTRAINT exam_attempts_pkey PRIMARY KEY (id);


--
-- Name: exam_events exam_events_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exam_events
    ADD CONSTRAINT exam_events_pkey PRIMARY KEY (id);


--
-- Name: exam_items exam_items_exam_position_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exam_items
    ADD CONSTRAINT exam_items_exam_position_key UNIQUE (exam_id, "position") DEFERRABLE INITIALLY DEFERRED;


--
-- Name: exam_items exam_items_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exam_items
    ADD CONSTRAINT exam_items_pkey PRIMARY KEY (id);


--
-- Name: exams exams_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exams
    ADD CONSTRAINT exams_pkey PRIMARY KEY (id);


--
-- Name: final_grade_snapshots final_grade_snapshots_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.final_grade_snapshots
    ADD CONSTRAINT final_grade_snapshots_pkey PRIMARY KEY (id);


--
-- Name: final_grade_snapshots final_grade_snapshots_student_version_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.final_grade_snapshots
    ADD CONSTRAINT final_grade_snapshots_student_version_key UNIQUE (course_id, student_id, version);


--
-- Name: forms_import_batches forms_import_batches_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.forms_import_batches
    ADD CONSTRAINT forms_import_batches_pkey PRIMARY KEY (id);


--
-- Name: forms_import_rows forms_import_rows_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.forms_import_rows
    ADD CONSTRAINT forms_import_rows_pkey PRIMARY KEY (batch_id, row_no);


--
-- Name: forum_posts forum_posts_course_id_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.forum_posts
    ADD CONSTRAINT forum_posts_course_id_key UNIQUE (course_id, id);


--
-- Name: forum_posts forum_posts_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.forum_posts
    ADD CONSTRAINT forum_posts_pkey PRIMARY KEY (id);


--
-- Name: forum_threads forum_threads_course_id_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.forum_threads
    ADD CONSTRAINT forum_threads_course_id_key UNIQUE (course_id, id);


--
-- Name: forum_threads forum_threads_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.forum_threads
    ADD CONSTRAINT forum_threads_pkey PRIMARY KEY (id);


--
-- Name: goose_db_version goose_db_version_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.goose_db_version
    ADD CONSTRAINT goose_db_version_pkey PRIMARY KEY (id);


--
-- Name: grade_appeals grade_appeals_once_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_appeals
    ADD CONSTRAINT grade_appeals_once_key UNIQUE (course_id, assignment_id, student_id);


--
-- Name: grade_appeals grade_appeals_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_appeals
    ADD CONSTRAINT grade_appeals_pkey PRIMARY KEY (id);


--
-- Name: grade_components grade_components_course_id_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_components
    ADD CONSTRAINT grade_components_course_id_key UNIQUE (course_id, id);


--
-- Name: grade_components grade_components_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_components
    ADD CONSTRAINT grade_components_pkey PRIMARY KEY (id);


--
-- Name: grade_components grade_components_scheme_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_components
    ADD CONSTRAINT grade_components_scheme_key UNIQUE (scheme_id, key);


--
-- Name: grade_entries grade_entries_item_student_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_entries
    ADD CONSTRAINT grade_entries_item_student_key UNIQUE (item_id, student_id);


--
-- Name: grade_entries grade_entries_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_entries
    ADD CONSTRAINT grade_entries_pkey PRIMARY KEY (id);


--
-- Name: grade_items grade_items_course_id_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_items
    ADD CONSTRAINT grade_items_course_id_key UNIQUE (course_id, id);


--
-- Name: grade_items grade_items_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_items
    ADD CONSTRAINT grade_items_pkey PRIMARY KEY (id);


--
-- Name: grade_items grade_items_scheme_exam_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_items
    ADD CONSTRAINT grade_items_scheme_exam_key UNIQUE (scheme_id, exam_id);


--
-- Name: grade_items grade_items_scheme_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_items
    ADD CONSTRAINT grade_items_scheme_key UNIQUE (scheme_id, key);


--
-- Name: grade_rules grade_rules_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_rules
    ADD CONSTRAINT grade_rules_pkey PRIMARY KEY (id);


--
-- Name: grade_schemes grade_schemes_course_id_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_schemes
    ADD CONSTRAINT grade_schemes_course_id_key UNIQUE (course_id, id);


--
-- Name: grade_schemes grade_schemes_extract_job_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_schemes
    ADD CONSTRAINT grade_schemes_extract_job_key UNIQUE (extract_job_id);


--
-- Name: grade_schemes grade_schemes_no_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_schemes
    ADD CONSTRAINT grade_schemes_no_key UNIQUE (course_id, scheme_no);


--
-- Name: grade_schemes grade_schemes_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_schemes
    ADD CONSTRAINT grade_schemes_pkey PRIMARY KEY (id);


--
-- Name: gradebooks gradebooks_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.gradebooks
    ADD CONSTRAINT gradebooks_pkey PRIMARY KEY (course_id);


--
-- Name: grading_guidance_items grading_guidance_items_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grading_guidance_items
    ADD CONSTRAINT grading_guidance_items_pkey PRIMARY KEY (id);


--
-- Name: grading_guidance_versions grading_guidance_versions_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grading_guidance_versions
    ADD CONSTRAINT grading_guidance_versions_pkey PRIMARY KEY (course_id, assignment_id, version);


--
-- Name: grading_regression_cases grading_regression_cases_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grading_regression_cases
    ADD CONSTRAINT grading_regression_cases_pkey PRIMARY KEY (id);


--
-- Name: grading_results grading_results_course_id_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grading_results
    ADD CONSTRAINT grading_results_course_id_key UNIQUE (course_id, id);


--
-- Name: grading_results grading_results_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grading_results
    ADD CONSTRAINT grading_results_pkey PRIMARY KEY (id);


--
-- Name: grading_results grading_results_round_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grading_results
    ADD CONSTRAINT grading_results_round_key UNIQUE (course_id, submission_id, round);


--
-- Name: idempotency_keys idempotency_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.idempotency_keys
    ADD CONSTRAINT idempotency_keys_pkey PRIMARY KEY (id);


--
-- Name: idempotency_keys idempotency_keys_user_endpoint_key_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.idempotency_keys
    ADD CONSTRAINT idempotency_keys_user_endpoint_key_key UNIQUE (user_id, endpoint, key);


--
-- Name: insight_reports insight_reports_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.insight_reports
    ADD CONSTRAINT insight_reports_pkey PRIMARY KEY (id);


--
-- Name: integration_configs integration_configs_kind_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.integration_configs
    ADD CONSTRAINT integration_configs_kind_key UNIQUE (course_id, kind);


--
-- Name: integration_configs integration_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.integration_configs
    ADD CONSTRAINT integration_configs_pkey PRIMARY KEY (id);


--
-- Name: jobs jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.jobs
    ADD CONSTRAINT jobs_pkey PRIMARY KEY (id);


--
-- Name: knowledge_items knowledge_items_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.knowledge_items
    ADD CONSTRAINT knowledge_items_pkey PRIMARY KEY (id);


--
-- Name: knowledge_items knowledge_items_source_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.knowledge_items
    ADD CONSTRAINT knowledge_items_source_key UNIQUE (source_type, source_id);


--
-- Name: llm_audit_hourly_lat llm_audit_hourly_lat_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.llm_audit_hourly_lat
    ADD CONSTRAINT llm_audit_hourly_lat_pkey PRIMARY KEY (hour, bucket);


--
-- Name: llm_audit_hourly llm_audit_hourly_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.llm_audit_hourly
    ADD CONSTRAINT llm_audit_hourly_pkey PRIMARY KEY (hour, task);


--
-- Name: llm_audit llm_audit_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.llm_audit
    ADD CONSTRAINT llm_audit_pkey PRIMARY KEY (id);


--
-- Name: llm_audit_rollup_state llm_audit_rollup_state_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.llm_audit_rollup_state
    ADD CONSTRAINT llm_audit_rollup_state_pkey PRIMARY KEY (id);


--
-- Name: llm_budgets llm_budgets_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.llm_budgets
    ADD CONSTRAINT llm_budgets_pkey PRIMARY KEY (id);


--
-- Name: llm_models llm_models_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.llm_models
    ADD CONSTRAINT llm_models_pkey PRIMARY KEY (id);


--
-- Name: llm_models llm_models_uq; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.llm_models
    ADD CONSTRAINT llm_models_uq UNIQUE (provider_id, model);


--
-- Name: llm_providers llm_providers_name_uq; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.llm_providers
    ADD CONSTRAINT llm_providers_name_uq UNIQUE (name);


--
-- Name: llm_providers llm_providers_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.llm_providers
    ADD CONSTRAINT llm_providers_pkey PRIMARY KEY (id);


--
-- Name: llm_request_payloads llm_request_payloads_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.llm_request_payloads
    ADD CONSTRAINT llm_request_payloads_pkey PRIMARY KEY (audit_id);


--
-- Name: llm_task_routes llm_task_routes_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.llm_task_routes
    ADD CONSTRAINT llm_task_routes_pkey PRIMARY KEY (id);


--
-- Name: llm_task_routes llm_task_routes_uq; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.llm_task_routes
    ADD CONSTRAINT llm_task_routes_uq UNIQUE (task, fallback_order);


--
-- Name: login_attempts login_attempts_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.login_attempts
    ADD CONSTRAINT login_attempts_pkey PRIMARY KEY (id);


--
-- Name: mail_inbound mail_inbound_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.mail_inbound
    ADD CONSTRAINT mail_inbound_pkey PRIMARY KEY (id);


--
-- Name: mail_inbound mail_inbound_uid_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.mail_inbound
    ADD CONSTRAINT mail_inbound_uid_key UNIQUE (course_id, uidvalidity, message_uid);


--
-- Name: mail_outbox mail_outbox_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.mail_outbox
    ADD CONSTRAINT mail_outbox_pkey PRIMARY KEY (id);


--
-- Name: notifications notifications_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_pkey PRIMARY KEY (id);


--
-- Name: outbox outbox_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.outbox
    ADD CONSTRAINT outbox_pkey PRIMARY KEY (id);


--
-- Name: participation_events participation_events_client_event_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.participation_events
    ADD CONSTRAINT participation_events_client_event_key UNIQUE (client_event_id);


--
-- Name: participation_events participation_events_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.participation_events
    ADD CONSTRAINT participation_events_pkey PRIMARY KEY (id);


--
-- Name: pii_events pii_events_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.pii_events
    ADD CONSTRAINT pii_events_pkey PRIMARY KEY (id);


--
-- Name: post_reports post_reports_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.post_reports
    ADD CONSTRAINT post_reports_pkey PRIMARY KEY (id);


--
-- Name: post_reports post_reports_reporter_target_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.post_reports
    ADD CONSTRAINT post_reports_reporter_target_key UNIQUE (reporter_id, target_id);


--
-- Name: practice_answers practice_answers_item_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.practice_answers
    ADD CONSTRAINT practice_answers_item_key UNIQUE (item_id);


--
-- Name: practice_answers practice_answers_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.practice_answers
    ADD CONSTRAINT practice_answers_pkey PRIMARY KEY (id);


--
-- Name: practice_attempts practice_attempts_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.practice_attempts
    ADD CONSTRAINT practice_attempts_pkey PRIMARY KEY (id);


--
-- Name: practice_items practice_items_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.practice_items
    ADD CONSTRAINT practice_items_pkey PRIMARY KEY (id);


--
-- Name: question_bank question_bank_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.question_bank
    ADD CONSTRAINT question_bank_pkey PRIMARY KEY (id);


--
-- Name: question_options question_options_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.question_options
    ADD CONSTRAINT question_options_pkey PRIMARY KEY (id);


--
-- Name: reminder_log reminder_log_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.reminder_log
    ADD CONSTRAINT reminder_log_pkey PRIMARY KEY (id);


--
-- Name: reminder_log reminder_log_unique; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.reminder_log
    ADD CONSTRAINT reminder_log_unique UNIQUE (user_id, source_type, source_id, starts_at, kind);


--
-- Name: similarity_reports similarity_reports_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.similarity_reports
    ADD CONSTRAINT similarity_reports_pkey PRIMARY KEY (id);


--
-- Name: student_observations student_observations_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.student_observations
    ADD CONSTRAINT student_observations_pkey PRIMARY KEY (id);


--
-- Name: student_risk student_risk_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.student_risk
    ADD CONSTRAINT student_risk_pkey PRIMARY KEY (course_id, student_id);


--
-- Name: study_time_daily study_time_daily_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.study_time_daily
    ADD CONSTRAINT study_time_daily_key UNIQUE (course_id, student_id, date, area);


--
-- Name: study_time_daily study_time_daily_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.study_time_daily
    ADD CONSTRAINT study_time_daily_pkey PRIMARY KEY (id);


--
-- Name: submissions submissions_course_id_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.submissions
    ADD CONSTRAINT submissions_course_id_key UNIQUE (course_id, id);


--
-- Name: submissions submissions_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.submissions
    ADD CONSTRAINT submissions_pkey PRIMARY KEY (id);


--
-- Name: user_devices user_devices_key_key; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.user_devices
    ADD CONSTRAINT user_devices_key_key UNIQUE (key_sha256);


--
-- Name: user_devices user_devices_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.user_devices
    ADD CONSTRAINT user_devices_pkey PRIMARY KEY (id);


--
-- Name: user_settings user_settings_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.user_settings
    ADD CONSTRAINT user_settings_pkey PRIMARY KEY (user_id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: assignments_course_due_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX assignments_course_due_idx ON public.assignments USING btree (course_id, due_at);


--
-- Name: assignments_course_published_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX assignments_course_published_idx ON public.assignments USING btree (course_id, published_at);


--
-- Name: attendance_checkins_device_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX attendance_checkins_device_idx ON public.attendance_checkins USING btree (course_id, device_id) WHERE (device_id IS NOT NULL);


--
-- Name: attendance_checkins_session_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX attendance_checkins_session_idx ON public.attendance_checkins USING btree (course_id, session_id, created_at, id);


--
-- Name: attendance_checkins_student_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX attendance_checkins_student_idx ON public.attendance_checkins USING btree (course_id, student_id, created_at);


--
-- Name: attendance_course_student_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX attendance_course_student_idx ON public.attendance_records USING btree (course_id, student_id, session_id);


--
-- Name: audit_log_actor_created_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX audit_log_actor_created_idx ON public.audit_log USING btree (actor_id, created_at DESC, id DESC) WHERE (actor_id IS NOT NULL);


--
-- Name: audit_log_course_created_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX audit_log_course_created_idx ON public.audit_log USING btree (course_id, created_at DESC, id DESC) WHERE (course_id IS NOT NULL);


--
-- Name: audit_log_created_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX audit_log_created_idx ON public.audit_log USING btree (created_at DESC, id DESC);


--
-- Name: audit_log_entity_created_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX audit_log_entity_created_idx ON public.audit_log USING btree (entity, created_at DESC, id DESC);


--
-- Name: audit_log_entity_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX audit_log_entity_idx ON public.audit_log USING btree (entity, entity_id, created_at DESC);


--
-- Name: auth_sessions_expires_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX auth_sessions_expires_idx ON public.auth_sessions USING btree (expires_at);


--
-- Name: auth_sessions_prev_hash_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX auth_sessions_prev_hash_idx ON public.auth_sessions USING btree (prev_refresh_hash) WHERE (prev_refresh_hash IS NOT NULL);


--
-- Name: auth_sessions_refresh_hash_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX auth_sessions_refresh_hash_key ON public.auth_sessions USING btree (refresh_hash);


--
-- Name: auth_sessions_user_active_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX auth_sessions_user_active_idx ON public.auth_sessions USING btree (user_id, last_used_at DESC, id DESC) WHERE (revoked_at IS NULL);


--
-- Name: auth_tokens_token_hash_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX auth_tokens_token_hash_key ON public.auth_tokens USING btree (token_hash);


--
-- Name: auth_tokens_user_kind_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX auth_tokens_user_kind_idx ON public.auth_tokens USING btree (user_id, kind, created_at DESC);


--
-- Name: calendar_events_course_starts_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX calendar_events_course_starts_idx ON public.calendar_events USING btree (course_id, starts_at, id);


--
-- Name: chat_messages_idem_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX chat_messages_idem_key ON public.chat_messages USING btree (session_id, client_msg_id) WHERE (client_msg_id IS NOT NULL);


--
-- Name: chat_messages_session_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX chat_messages_session_idx ON public.chat_messages USING btree (session_id, created_at DESC, id DESC);


--
-- Name: chat_messages_streaming_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX chat_messages_streaming_idx ON public.chat_messages USING btree (updated_at) WHERE (stream_status = 'STREAMING'::public.chat_stream_status);


--
-- Name: chat_sessions_user_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX chat_sessions_user_idx ON public.chat_sessions USING btree (course_id, user_id, last_message_at DESC, id DESC) WHERE (deleted_at IS NULL);


--
-- Name: checkin_windows_one_open; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX checkin_windows_one_open ON public.checkin_windows USING btree (course_id, session_id) WHERE (closed_at IS NULL);


--
-- Name: checkin_windows_purge_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX checkin_windows_purge_idx ON public.checkin_windows USING btree (closed_at) WHERE (closed_at IS NOT NULL);


--
-- Name: checkin_windows_sweep_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX checkin_windows_sweep_idx ON public.checkin_windows USING btree (closes_at) WHERE (closed_at IS NULL);


--
-- Name: class_sessions_course_no_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX class_sessions_course_no_key ON public.class_sessions USING btree (course_id, session_no);


--
-- Name: class_sessions_course_starts_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX class_sessions_course_starts_idx ON public.class_sessions USING btree (course_id, starts_at);


--
-- Name: class_sessions_course_starts_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX class_sessions_course_starts_key ON public.class_sessions USING btree (course_id, starts_at);


--
-- Name: code_drafts_course_attempt_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX code_drafts_course_attempt_idx ON public.code_drafts USING btree (course_id, attempt_id);


--
-- Name: code_problems_course_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX code_problems_course_idx ON public.code_problems USING btree (course_id);


--
-- Name: code_problems_course_question_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX code_problems_course_question_key ON public.code_problems USING btree (course_id, question_id);


--
-- Name: code_submissions_attempt_item_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX code_submissions_attempt_item_idx ON public.code_submissions USING btree (course_id, attempt_id, item_id, created_at DESC, id DESC) WHERE (kind = 'SUBMIT'::public.submission_kind);


--
-- Name: code_submissions_course_id_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX code_submissions_course_id_key ON public.code_submissions USING btree (course_id, id);


--
-- Name: code_submissions_exam_problem_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX code_submissions_exam_problem_idx ON public.code_submissions USING btree (course_id, exam_id, problem_id) WHERE (kind = 'SUBMIT'::public.submission_kind);


--
-- Name: code_submissions_lease_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX code_submissions_lease_idx ON public.code_submissions USING btree (lease_until) WHERE (status = 'RUNNING'::public.submission_status);


--
-- Name: code_submissions_queue_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX code_submissions_queue_idx ON public.code_submissions USING btree (next_attempt_at, id) WHERE ((status = 'QUEUED'::public.submission_status) AND (enqueued_at IS NULL));


--
-- Name: code_submissions_runs_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX code_submissions_runs_idx ON public.code_submissions USING btree (attempt_id, created_at DESC) WHERE (kind = 'RUN'::public.submission_kind);


--
-- Name: code_testcases_course_problem_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX code_testcases_course_problem_idx ON public.code_testcases USING btree (course_id, problem_id, "position");


--
-- Name: content_chunks_course_ids_gin; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX content_chunks_course_ids_gin ON public.content_chunks USING gin (course_ids);


--
-- Name: content_chunks_doc_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX content_chunks_doc_idx ON public.content_chunks USING btree (document_id);


--
-- Name: content_chunks_doc_ord_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX content_chunks_doc_ord_key ON public.content_chunks USING btree (document_id, ord);


--
-- Name: content_chunks_tsv_gin; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX content_chunks_tsv_gin ON public.content_chunks USING gin (tsv);


--
-- Name: course_decision_gates_gate_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX course_decision_gates_gate_idx ON public.course_decision_gates USING btree (gate, enabled);


--
-- Name: courses_class_code_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX courses_class_code_key ON public.courses USING btree (lower(class_code));


--
-- Name: courses_join_code_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX courses_join_code_key ON public.courses USING btree (join_code);


--
-- Name: courses_status_created_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX courses_status_created_idx ON public.courses USING btree (status, created_at DESC, id DESC);


--
-- Name: courses_subject_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX courses_subject_idx ON public.courses USING btree (subject_code);


--
-- Name: document_courses_course_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX document_courses_course_idx ON public.document_courses USING btree (course_id, document_id);


--
-- Name: documents_course_created_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX documents_course_created_idx ON public.documents USING btree (course_id, created_at DESC, id DESC);


--
-- Name: documents_course_sha_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX documents_course_sha_idx ON public.documents USING btree (course_id, sha256) WHERE (sha256 IS NOT NULL);


--
-- Name: documents_course_type_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX documents_course_type_idx ON public.documents USING btree (course_id, type);


--
-- Name: documents_course_week_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX documents_course_week_idx ON public.documents USING btree (course_id, week_no) WHERE (week_no IS NOT NULL);


--
-- Name: documents_knowledge_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX documents_knowledge_key ON public.documents USING btree (course_id) WHERE (category = 'THREAD_KNOWLEDGE'::text);


--
-- Name: enrollments_course_status_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX enrollments_course_status_idx ON public.enrollments USING btree (course_id, status, role_in_course, status_changed_at DESC, user_id);


--
-- Name: enrollments_course_user_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX enrollments_course_user_key ON public.enrollments USING btree (course_id, user_id);


--
-- Name: enrollments_one_teacher_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX enrollments_one_teacher_key ON public.enrollments USING btree (course_id) WHERE ((role_in_course = 'TEACHER'::public.enrollment_role) AND (status = 'ACTIVE'::public.enrollment_status));


--
-- Name: enrollments_pending_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX enrollments_pending_idx ON public.enrollments USING btree (course_id, created_at) WHERE (status = 'PENDING'::public.enrollment_status);


--
-- Name: enrollments_snapshot_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX enrollments_snapshot_idx ON public.enrollments USING btree (course_id, student_code_snapshot) WHERE ((student_code_snapshot IS NOT NULL) AND (status = ANY (ARRAY['ACTIVE'::public.enrollment_status, 'PENDING'::public.enrollment_status])));


--
-- Name: enrollments_user_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX enrollments_user_idx ON public.enrollments USING btree (user_id, status);


--
-- Name: escalation_tickets_autoclose_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX escalation_tickets_autoclose_idx ON public.escalation_tickets USING btree (answered_at) WHERE (status = 'ANSWERED'::public.ticket_status);


--
-- Name: escalation_tickets_claimed_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX escalation_tickets_claimed_idx ON public.escalation_tickets USING btree (course_id, claimed_by) WHERE (status = 'CLAIMED'::public.ticket_status);


--
-- Name: escalation_tickets_course_status_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX escalation_tickets_course_status_idx ON public.escalation_tickets USING btree (course_id, status, created_at, id);


--
-- Name: escalation_tickets_sla_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX escalation_tickets_sla_idx ON public.escalation_tickets USING btree (waiting_since, id) WHERE (status = ANY (ARRAY['OPEN'::public.ticket_status, 'CLAIMED'::public.ticket_status]));


--
-- Name: escalation_tickets_student_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX escalation_tickets_student_idx ON public.escalation_tickets USING btree (course_id, student_id, status, created_at DESC, id DESC);


--
-- Name: exam_answers_course_attempt_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX exam_answers_course_attempt_idx ON public.exam_answers USING btree (course_id, attempt_id);


--
-- Name: exam_appeals_attempt_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX exam_appeals_attempt_key ON public.exam_appeals USING btree (attempt_id);


--
-- Name: exam_appeals_course_exam_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX exam_appeals_course_exam_idx ON public.exam_appeals USING btree (course_id, exam_id, status, created_at);


--
-- Name: exam_appeals_open_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX exam_appeals_open_idx ON public.exam_appeals USING btree (course_id, created_at) WHERE (status = 'OPEN'::public.appeal_status);


--
-- Name: exam_attempts_course_exam_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX exam_attempts_course_exam_idx ON public.exam_attempts USING btree (course_id, exam_id, status, student_id);


--
-- Name: exam_attempts_course_id_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX exam_attempts_course_id_key ON public.exam_attempts USING btree (course_id, id);


--
-- Name: exam_attempts_due_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX exam_attempts_due_idx ON public.exam_attempts USING btree (deadline_at) WHERE (status = 'IN_PROGRESS'::public.attempt_status);


--
-- Name: exam_attempts_exam_student_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX exam_attempts_exam_student_key ON public.exam_attempts USING btree (exam_id, student_id);


--
-- Name: exam_attempts_running_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX exam_attempts_running_idx ON public.exam_attempts USING btree (student_id, deadline_at) WHERE (status = 'IN_PROGRESS'::public.attempt_status);


--
-- Name: exam_attempts_student_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX exam_attempts_student_idx ON public.exam_attempts USING btree (course_id, student_id, started_at DESC);


--
-- Name: exam_events_attempt_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX exam_events_attempt_idx ON public.exam_events USING btree (course_id, exam_id, attempt_id, occurred_at, id);


--
-- Name: exam_items_course_exam_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX exam_items_course_exam_idx ON public.exam_items USING btree (course_id, exam_id, "position");


--
-- Name: exam_items_course_id_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX exam_items_course_id_key ON public.exam_items USING btree (course_id, id);


--
-- Name: exam_items_exam_question_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX exam_items_exam_question_key ON public.exam_items USING btree (exam_id, question_id);


--
-- Name: exam_items_question_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX exam_items_question_idx ON public.exam_items USING btree (question_id, exam_id);


--
-- Name: exams_closed_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX exams_closed_idx ON public.exams USING btree (course_id, id) WHERE (status = 'CLOSED'::public.exam_status);


--
-- Name: exams_course_created_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX exams_course_created_idx ON public.exams USING btree (course_id, created_at DESC, id DESC);


--
-- Name: exams_course_id_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX exams_course_id_key ON public.exams USING btree (course_id, id);


--
-- Name: exams_course_status_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX exams_course_status_idx ON public.exams USING btree (course_id, status, opens_at DESC, id DESC);


--
-- Name: exams_due_close_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX exams_due_close_idx ON public.exams USING btree (closes_at) WHERE (status = ANY (ARRAY['SCHEDULED'::public.exam_status, 'OPEN'::public.exam_status]));


--
-- Name: exams_due_open_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX exams_due_open_idx ON public.exams USING btree (opens_at) WHERE (status = 'SCHEDULED'::public.exam_status);


--
-- Name: final_grade_snapshots_version_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX final_grade_snapshots_version_idx ON public.final_grade_snapshots USING btree (course_id, version, student_id);


--
-- Name: forms_import_batches_course_id_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX forms_import_batches_course_id_key ON public.forms_import_batches USING btree (course_id, id);


--
-- Name: forms_import_batches_file_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX forms_import_batches_file_key ON public.forms_import_batches USING btree (course_id, item_id, file_hash);


--
-- Name: forms_import_batches_unpurged_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX forms_import_batches_unpurged_idx ON public.forms_import_batches USING btree (created_at) WHERE (blob_purged_at IS NULL);


--
-- Name: forum_posts_one_ai_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX forum_posts_one_ai_key ON public.forum_posts USING btree (thread_id) WHERE ((kind = 'AI'::public.post_kind) AND (verification_state <> 'REJECTED'::public.post_verification));


--
-- Name: forum_posts_pending_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX forum_posts_pending_idx ON public.forum_posts USING btree (course_id, created_at) WHERE ((kind = 'AI'::public.post_kind) AND (verification_state = 'PENDING'::public.post_verification) AND (hidden_at IS NULL) AND (deleted_at IS NULL));


--
-- Name: forum_posts_thread_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX forum_posts_thread_idx ON public.forum_posts USING btree (thread_id, created_at, id);


--
-- Name: forum_threads_course_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX forum_threads_course_idx ON public.forum_threads USING btree (course_id, last_activity_at DESC, id DESC) WHERE (deleted_at IS NULL);


--
-- Name: forum_threads_hidden_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX forum_threads_hidden_idx ON public.forum_threads USING btree (course_id, hidden_at) WHERE (hidden_at IS NOT NULL);


--
-- Name: forum_threads_skipped_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX forum_threads_skipped_idx ON public.forum_threads USING btree (course_id, created_at) WHERE ((ai_state = 'SKIPPED'::public.thread_ai_state) AND (deleted_at IS NULL));


--
-- Name: forum_threads_week_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX forum_threads_week_idx ON public.forum_threads USING btree (course_id, week_no);


--
-- Name: ggi_assignment_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX ggi_assignment_idx ON public.grading_guidance_items USING btree (course_id, assignment_id, status);


--
-- Name: grade_appeals_status_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX grade_appeals_status_idx ON public.grade_appeals USING btree (course_id, status, created_at);


--
-- Name: grade_components_course_scheme_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX grade_components_course_scheme_idx ON public.grade_components USING btree (course_id, scheme_id, "position");


--
-- Name: grade_entries_course_item_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX grade_entries_course_item_idx ON public.grade_entries USING btree (course_id, item_id);


--
-- Name: grade_entries_course_student_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX grade_entries_course_student_idx ON public.grade_entries USING btree (course_id, student_id);


--
-- Name: grade_items_course_exam_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX grade_items_course_exam_idx ON public.grade_items USING btree (course_id, exam_id) WHERE (exam_id IS NOT NULL);


--
-- Name: grade_items_course_scheme_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX grade_items_course_scheme_idx ON public.grade_items USING btree (course_id, scheme_id, "position");


--
-- Name: grade_rules_course_scheme_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX grade_rules_course_scheme_idx ON public.grade_rules USING btree (course_id, scheme_id);


--
-- Name: grade_rules_scheme_kind_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX grade_rules_scheme_kind_key ON public.grade_rules USING btree (scheme_id, kind) WHERE (kind <> 'FAIL_CONDITION'::public.grade_rule_kind);


--
-- Name: grade_schemes_course_status_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX grade_schemes_course_status_idx ON public.grade_schemes USING btree (course_id, status);


--
-- Name: grade_schemes_one_confirmed_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX grade_schemes_one_confirmed_idx ON public.grade_schemes USING btree (course_id) WHERE (status = 'CONFIRMED'::public.grade_scheme_status);


--
-- Name: grade_schemes_one_draft_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX grade_schemes_one_draft_idx ON public.grade_schemes USING btree (course_id) WHERE (status = 'DRAFT'::public.grade_scheme_status);


--
-- Name: grading_results_lease_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX grading_results_lease_idx ON public.grading_results USING btree (run_state, lease_until) WHERE (run_state = ANY (ARRAY['QUEUED'::text, 'RUNNING'::text]));


--
-- Name: grading_results_review_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX grading_results_review_idx ON public.grading_results USING btree (course_id, needs_review, status);


--
-- Name: grading_results_status_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX grading_results_status_idx ON public.grading_results USING btree (course_id, status, run_state);


--
-- Name: grading_results_submission_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX grading_results_submission_idx ON public.grading_results USING btree (course_id, submission_id, round DESC);


--
-- Name: grc_course_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX grc_course_idx ON public.grading_regression_cases USING btree (course_id, kind, created_at);


--
-- Name: grc_unique; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX grc_unique ON public.grading_regression_cases USING btree (course_id, kind, source_id, COALESCE(criterion_id, ''::text));


--
-- Name: idempotency_keys_created_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX idempotency_keys_created_idx ON public.idempotency_keys USING btree (created_at);


--
-- Name: insight_reports_course_kind_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX insight_reports_course_kind_idx ON public.insight_reports USING btree (course_id, kind, created_at DESC);


--
-- Name: jobs_active_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX jobs_active_idx ON public.jobs USING btree (status, created_at) WHERE (status = ANY (ARRAY['QUEUED'::public.job_status, 'RUNNING'::public.job_status]));


--
-- Name: jobs_owner_created_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX jobs_owner_created_idx ON public.jobs USING btree (owner_id, created_at DESC, id DESC);


--
-- Name: knowledge_items_course_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX knowledge_items_course_idx ON public.knowledge_items USING btree (course_id, created_at DESC);


--
-- Name: llm_audit_course_created_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX llm_audit_course_created_idx ON public.llm_audit USING btree (course_id, created_at DESC) WHERE (course_id IS NOT NULL);


--
-- Name: llm_audit_created_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX llm_audit_created_idx ON public.llm_audit USING btree (created_at DESC);


--
-- Name: llm_audit_decide_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX llm_audit_decide_idx ON public.llm_audit USING btree (created_at, cascade_gate) WHERE (task = 'DECIDE'::text);


--
-- Name: llm_audit_status_created_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX llm_audit_status_created_idx ON public.llm_audit USING btree (status, created_at DESC);


--
-- Name: llm_audit_task_created_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX llm_audit_task_created_idx ON public.llm_audit USING btree (task, created_at DESC);


--
-- Name: llm_audit_trace_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX llm_audit_trace_idx ON public.llm_audit USING btree (trace_id);


--
-- Name: llm_budgets_course_uq; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX llm_budgets_course_uq ON public.llm_budgets USING btree (course_id) WHERE (scope = 'course'::text);


--
-- Name: llm_budgets_system_uq; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX llm_budgets_system_uq ON public.llm_budgets USING btree (scope) WHERE (scope = 'system'::text);


--
-- Name: llm_models_provider_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX llm_models_provider_idx ON public.llm_models USING btree (provider_id);


--
-- Name: llm_providers_one_decision; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX llm_providers_one_decision ON public.llm_providers USING btree ((true)) WHERE ((type = 'decision'::text) AND enabled);


--
-- Name: llm_request_payloads_created_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX llm_request_payloads_created_idx ON public.llm_request_payloads USING btree (created_at);


--
-- Name: llm_task_routes_model_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX llm_task_routes_model_idx ON public.llm_task_routes USING btree (model_id);


--
-- Name: login_attempts_email_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX login_attempts_email_idx ON public.login_attempts USING btree (email_hash, created_at DESC);


--
-- Name: login_attempts_ip_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX login_attempts_ip_idx ON public.login_attempts USING btree (ip, created_at DESC);


--
-- Name: login_attempts_user_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX login_attempts_user_idx ON public.login_attempts USING btree (user_id, created_at DESC) WHERE (user_id IS NOT NULL);


--
-- Name: mail_inbound_status_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX mail_inbound_status_idx ON public.mail_inbound USING btree (course_id, status, received_at);


--
-- Name: mail_outbox_dead_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX mail_outbox_dead_idx ON public.mail_outbox USING btree (updated_at DESC, id DESC) WHERE (status = 'DEAD'::public.mail_status);


--
-- Name: mail_outbox_dedupe_key_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX mail_outbox_dedupe_key_key ON public.mail_outbox USING btree (dedupe_key) WHERE (dedupe_key IS NOT NULL);


--
-- Name: mail_outbox_queued_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX mail_outbox_queued_idx ON public.mail_outbox USING btree (created_at, id) WHERE (status = 'QUEUED'::public.mail_status);


--
-- Name: notifications_course_created_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX notifications_course_created_idx ON public.notifications USING btree (course_id, created_at DESC) WHERE (course_id IS NOT NULL);


--
-- Name: notifications_unread_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX notifications_unread_idx ON public.notifications USING btree (user_id, created_at DESC, id DESC) WHERE (read_at IS NULL);


--
-- Name: notifications_user_created_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX notifications_user_created_idx ON public.notifications USING btree (user_id, created_at DESC, id DESC);


--
-- Name: notifications_user_dedupe_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX notifications_user_dedupe_key ON public.notifications USING btree (user_id, dedupe_key) WHERE (dedupe_key IS NOT NULL);


--
-- Name: observations_student_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX observations_student_idx ON public.student_observations USING btree (course_id, student_id, created_at DESC, id DESC);


--
-- Name: outbox_pending_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX outbox_pending_idx ON public.outbox USING btree (next_attempt_at, id) WHERE ((dispatched_at IS NULL) AND (dead_at IS NULL) AND (enqueued_at IS NULL));


--
-- Name: outbox_stale_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX outbox_stale_idx ON public.outbox USING btree (enqueued_at) WHERE ((dispatched_at IS NULL) AND (dead_at IS NULL) AND (enqueued_at IS NOT NULL));


--
-- Name: participation_course_session_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX participation_course_session_idx ON public.participation_events USING btree (course_id, session_id) WHERE (session_id IS NOT NULL);


--
-- Name: participation_course_student_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX participation_course_student_idx ON public.participation_events USING btree (course_id, student_id, created_at DESC, id DESC);


--
-- Name: pii_events_course_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX pii_events_course_idx ON public.pii_events USING btree (course_id, created_at DESC);


--
-- Name: pii_events_user_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX pii_events_user_idx ON public.pii_events USING btree (user_id, created_at DESC);


--
-- Name: post_reports_open_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX post_reports_open_idx ON public.post_reports USING btree (course_id, status, created_at, id);


--
-- Name: post_reports_target_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX post_reports_target_idx ON public.post_reports USING btree (target_id) WHERE (status = 'OPEN'::public.post_report_status);


--
-- Name: practice_answers_attempt_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX practice_answers_attempt_idx ON public.practice_answers USING btree (course_id, attempt_id);


--
-- Name: practice_attempts_course_id_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX practice_attempts_course_id_key ON public.practice_attempts USING btree (course_id, id);


--
-- Name: practice_attempts_due_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX practice_attempts_due_idx ON public.practice_attempts USING btree (status, deadline_at) WHERE ((status = 'IN_PROGRESS'::public.practice_status) AND (mode = 'MOCK'::public.practice_mode));


--
-- Name: practice_attempts_open_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX practice_attempts_open_key ON public.practice_attempts USING btree (course_id, student_id, mode, COALESCE(topic, ''::text)) WHERE (status = 'IN_PROGRESS'::public.practice_status);


--
-- Name: practice_attempts_student_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX practice_attempts_student_idx ON public.practice_attempts USING btree (course_id, student_id, started_at DESC, id DESC);


--
-- Name: practice_items_attempt_position_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX practice_items_attempt_position_key ON public.practice_items USING btree (attempt_id, "position");


--
-- Name: practice_items_course_id_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX practice_items_course_id_key ON public.practice_items USING btree (course_id, id);


--
-- Name: practice_items_question_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX practice_items_question_idx ON public.practice_items USING btree (course_id, question_id);


--
-- Name: question_bank_course_created_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX question_bank_course_created_idx ON public.question_bank USING btree (course_id, created_at DESC, id DESC);


--
-- Name: question_bank_course_id_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX question_bank_course_id_key ON public.question_bank USING btree (course_id, id);


--
-- Name: question_bank_course_review_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX question_bank_course_review_idx ON public.question_bank USING btree (course_id, review_status, created_at DESC, id DESC) WHERE (archived_at IS NULL);


--
-- Name: question_bank_course_topic_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX question_bank_course_topic_idx ON public.question_bank USING btree (course_id, topic, difficulty, type) WHERE (archived_at IS NULL);


--
-- Name: question_bank_rejected_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX question_bank_rejected_idx ON public.question_bank USING btree (course_id, reviewed_at DESC) WHERE ((review_status = 'REJECTED'::public.question_review_status) AND (reject_reason IS NOT NULL) AND (reject_reason <> 'UNSPECIFIED'::text));


--
-- Name: question_options_course_question_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX question_options_course_question_idx ON public.question_options USING btree (course_id, question_id, "position");


--
-- Name: question_options_question_position_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX question_options_question_position_key ON public.question_options USING btree (question_id, "position");


--
-- Name: reminder_log_starts_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX reminder_log_starts_idx ON public.reminder_log USING btree (starts_at);


--
-- Name: similarity_reports_exam_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX similarity_reports_exam_idx ON public.similarity_reports USING btree (course_id, exam_id, problem_id, score DESC, id);


--
-- Name: similarity_reports_run_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX similarity_reports_run_idx ON public.similarity_reports USING btree (run_id);


--
-- Name: similarity_reports_run_pair_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX similarity_reports_run_pair_key ON public.similarity_reports USING btree (run_id, submission_a, submission_b);


--
-- Name: student_risk_entered_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX student_risk_entered_idx ON public.student_risk USING btree (course_id, entered_at DESC) WHERE (entered_at IS NOT NULL);


--
-- Name: student_risk_level_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX student_risk_level_idx ON public.student_risk USING btree (course_id, level, student_id);


--
-- Name: study_time_course_date_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX study_time_course_date_idx ON public.study_time_daily USING btree (course_id, date, student_id);


--
-- Name: study_time_student_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX study_time_student_idx ON public.study_time_daily USING btree (course_id, student_id, date DESC);


--
-- Name: submissions_assignment_received_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX submissions_assignment_received_idx ON public.submissions USING btree (course_id, assignment_id, received_at, id);


--
-- Name: submissions_external_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX submissions_external_key ON public.submissions USING btree (course_id, source, external_ref, external_ts) WHERE (external_ref IS NOT NULL);


--
-- Name: submissions_matched_version_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX submissions_matched_version_key ON public.submissions USING btree (course_id, assignment_id, student_id, version) WHERE (match_status = 'MATCHED'::text);


--
-- Name: submissions_student_version_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX submissions_student_version_idx ON public.submissions USING btree (course_id, student_id, assignment_id, version DESC);


--
-- Name: submissions_unmatched_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX submissions_unmatched_idx ON public.submissions USING btree (course_id, match_status, received_at) WHERE (dismissed_at IS NULL);


--
-- Name: users_created_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX users_created_idx ON public.users USING btree (created_at DESC, id DESC);


--
-- Name: users_email_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX users_email_key ON public.users USING btree (email);


--
-- Name: users_ics_token_key; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE UNIQUE INDEX users_ics_token_key ON public.users USING btree (ics_token) WHERE (ics_token IS NOT NULL);


--
-- Name: users_student_code_idx; Type: INDEX; Schema: public; Owner: edupilot
--

CREATE INDEX users_student_code_idx ON public.users USING btree (student_code) WHERE (student_code IS NOT NULL);


--
-- Name: assignments assignments_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER assignments_set_updated_at BEFORE UPDATE ON public.assignments FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: attendance_records attendance_records_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER attendance_records_set_updated_at BEFORE UPDATE ON public.attendance_records FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: audit_log audit_log_no_truncate; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER audit_log_no_truncate BEFORE TRUNCATE ON public.audit_log FOR EACH STATEMENT EXECUTE FUNCTION public.audit_log_block_mutation();


--
-- Name: audit_log audit_log_no_update; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER audit_log_no_update BEFORE DELETE OR UPDATE ON public.audit_log FOR EACH ROW EXECUTE FUNCTION public.audit_log_block_mutation();


--
-- Name: auth_sessions auth_sessions_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER auth_sessions_set_updated_at BEFORE UPDATE ON public.auth_sessions FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: calendar_events calendar_events_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER calendar_events_set_updated_at BEFORE UPDATE ON public.calendar_events FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: course_guidance_flags cgf_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER cgf_set_updated_at BEFORE UPDATE ON public.course_guidance_flags FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: chat_messages chat_messages_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER chat_messages_set_updated_at BEFORE UPDATE ON public.chat_messages FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: chat_sessions chat_sessions_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER chat_sessions_set_updated_at BEFORE UPDATE ON public.chat_sessions FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: class_sessions class_sessions_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER class_sessions_set_updated_at BEFORE UPDATE ON public.class_sessions FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: code_drafts code_drafts_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER code_drafts_set_updated_at BEFORE UPDATE ON public.code_drafts FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: code_problems code_problems_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER code_problems_set_updated_at BEFORE UPDATE ON public.code_problems FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: code_submissions code_submissions_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER code_submissions_set_updated_at BEFORE UPDATE ON public.code_submissions FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: code_testcases code_testcases_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER code_testcases_set_updated_at BEFORE UPDATE ON public.code_testcases FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: content_chunks content_chunks_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER content_chunks_set_updated_at BEFORE UPDATE ON public.content_chunks FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: courses courses_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER courses_set_updated_at BEFORE UPDATE ON public.courses FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: documents documents_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER documents_set_updated_at BEFORE UPDATE ON public.documents FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: enrollments enrollments_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER enrollments_set_updated_at BEFORE UPDATE ON public.enrollments FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: escalation_tickets escalation_tickets_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER escalation_tickets_set_updated_at BEFORE UPDATE ON public.escalation_tickets FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: exam_appeals exam_appeals_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER exam_appeals_set_updated_at BEFORE UPDATE ON public.exam_appeals FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: exam_attempts exam_attempts_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER exam_attempts_set_updated_at BEFORE UPDATE ON public.exam_attempts FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: exam_items exam_items_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER exam_items_set_updated_at BEFORE UPDATE ON public.exam_items FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: exams exams_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER exams_set_updated_at BEFORE UPDATE ON public.exams FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: final_grade_snapshots final_grade_snapshots_immutable_row; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER final_grade_snapshots_immutable_row BEFORE DELETE OR UPDATE ON public.final_grade_snapshots FOR EACH ROW EXECUTE FUNCTION public.final_grade_snapshots_immutable();


--
-- Name: final_grade_snapshots final_grade_snapshots_immutable_truncate; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER final_grade_snapshots_immutable_truncate BEFORE TRUNCATE ON public.final_grade_snapshots FOR EACH STATEMENT EXECUTE FUNCTION public.final_grade_snapshots_immutable();


--
-- Name: forum_posts forum_posts_gc_cases; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER forum_posts_gc_cases AFTER DELETE ON public.forum_posts FOR EACH ROW EXECUTE FUNCTION public.grading_regression_cases_gc();


--
-- Name: forum_posts forum_posts_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER forum_posts_set_updated_at BEFORE UPDATE ON public.forum_posts FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: forum_threads forum_threads_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER forum_threads_set_updated_at BEFORE UPDATE ON public.forum_threads FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: grading_guidance_items ggi_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER ggi_set_updated_at BEFORE UPDATE ON public.grading_guidance_items FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: grade_appeals grade_appeals_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER grade_appeals_set_updated_at BEFORE UPDATE ON public.grade_appeals FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: grade_components grade_components_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER grade_components_set_updated_at BEFORE UPDATE ON public.grade_components FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: grade_entries grade_entries_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER grade_entries_set_updated_at BEFORE UPDATE ON public.grade_entries FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: grade_items grade_items_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER grade_items_set_updated_at BEFORE UPDATE ON public.grade_items FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: grade_rules grade_rules_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER grade_rules_set_updated_at BEFORE UPDATE ON public.grade_rules FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: grade_schemes grade_schemes_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER grade_schemes_set_updated_at BEFORE UPDATE ON public.grade_schemes FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: gradebooks gradebooks_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER gradebooks_set_updated_at BEFORE UPDATE ON public.gradebooks FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: grading_results grading_results_ai_immutable_row; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER grading_results_ai_immutable_row BEFORE UPDATE ON public.grading_results FOR EACH ROW EXECUTE FUNCTION public.grading_results_ai_immutable();


--
-- Name: grading_results grading_results_gc_cases; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER grading_results_gc_cases AFTER DELETE ON public.grading_results FOR EACH ROW EXECUTE FUNCTION public.grading_regression_cases_gc();


--
-- Name: grading_results grading_results_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER grading_results_set_updated_at BEFORE UPDATE ON public.grading_results FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: grading_regression_cases grc_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER grc_set_updated_at BEFORE UPDATE ON public.grading_regression_cases FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: insight_reports insight_reports_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER insight_reports_set_updated_at BEFORE UPDATE ON public.insight_reports FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: integration_configs integration_configs_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER integration_configs_set_updated_at BEFORE UPDATE ON public.integration_configs FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: jobs jobs_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER jobs_set_updated_at BEFORE UPDATE ON public.jobs FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: knowledge_items knowledge_items_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER knowledge_items_set_updated_at BEFORE UPDATE ON public.knowledge_items FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: llm_budgets llm_budgets_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER llm_budgets_set_updated_at BEFORE UPDATE ON public.llm_budgets FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: llm_models llm_models_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER llm_models_set_updated_at BEFORE UPDATE ON public.llm_models FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: llm_providers llm_providers_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER llm_providers_set_updated_at BEFORE UPDATE ON public.llm_providers FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: llm_task_routes llm_task_routes_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER llm_task_routes_set_updated_at BEFORE UPDATE ON public.llm_task_routes FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: mail_outbox mail_outbox_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER mail_outbox_set_updated_at BEFORE UPDATE ON public.mail_outbox FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: notifications notifications_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER notifications_set_updated_at BEFORE UPDATE ON public.notifications FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: outbox outbox_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER outbox_set_updated_at BEFORE UPDATE ON public.outbox FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: participation_events participation_events_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER participation_events_set_updated_at BEFORE UPDATE ON public.participation_events FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: pii_events pii_events_no_truncate; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER pii_events_no_truncate BEFORE TRUNCATE ON public.pii_events FOR EACH STATEMENT EXECUTE FUNCTION public.audit_log_block_mutation();


--
-- Name: pii_events pii_events_no_update; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER pii_events_no_update BEFORE DELETE OR UPDATE ON public.pii_events FOR EACH ROW EXECUTE FUNCTION public.audit_log_block_mutation();


--
-- Name: post_reports post_reports_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER post_reports_set_updated_at BEFORE UPDATE ON public.post_reports FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: question_bank question_bank_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER question_bank_set_updated_at BEFORE UPDATE ON public.question_bank FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: student_observations student_observations_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER student_observations_set_updated_at BEFORE UPDATE ON public.student_observations FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: student_risk student_risk_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER student_risk_set_updated_at BEFORE UPDATE ON public.student_risk FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: study_time_daily study_time_daily_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER study_time_daily_set_updated_at BEFORE UPDATE ON public.study_time_daily FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: submissions submissions_immutable_row; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER submissions_immutable_row BEFORE UPDATE ON public.submissions FOR EACH ROW EXECUTE FUNCTION public.submissions_immutable();


--
-- Name: submissions submissions_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER submissions_set_updated_at BEFORE UPDATE ON public.submissions FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: user_settings user_settings_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER user_settings_set_updated_at BEFORE UPDATE ON public.user_settings FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: users users_set_updated_at; Type: TRIGGER; Schema: public; Owner: edupilot
--

CREATE TRIGGER users_set_updated_at BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: assignments assignments_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.assignments
    ADD CONSTRAINT assignments_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: assignments assignments_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.assignments
    ADD CONSTRAINT assignments_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: attendance_checkins attendance_checkins_cleared_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.attendance_checkins
    ADD CONSTRAINT attendance_checkins_cleared_by_fkey FOREIGN KEY (flags_cleared_by) REFERENCES public.users(id);


--
-- Name: attendance_checkins attendance_checkins_device_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.attendance_checkins
    ADD CONSTRAINT attendance_checkins_device_fkey FOREIGN KEY (device_id) REFERENCES public.user_devices(id);


--
-- Name: attendance_checkins attendance_checkins_session_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.attendance_checkins
    ADD CONSTRAINT attendance_checkins_session_fkey FOREIGN KEY (course_id, session_id) REFERENCES public.class_sessions(course_id, id);


--
-- Name: attendance_checkins attendance_checkins_student_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.attendance_checkins
    ADD CONSTRAINT attendance_checkins_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES public.enrollments(course_id, user_id);


--
-- Name: attendance_checkins attendance_checkins_window_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.attendance_checkins
    ADD CONSTRAINT attendance_checkins_window_fkey FOREIGN KEY (window_id) REFERENCES public.checkin_windows(id);


--
-- Name: attendance_records attendance_records_marked_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.attendance_records
    ADD CONSTRAINT attendance_records_marked_by_fkey FOREIGN KEY (marked_by) REFERENCES public.users(id);


--
-- Name: attendance_records attendance_records_session_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.attendance_records
    ADD CONSTRAINT attendance_records_session_fkey FOREIGN KEY (course_id, session_id) REFERENCES public.class_sessions(course_id, id);


--
-- Name: attendance_records attendance_records_student_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.attendance_records
    ADD CONSTRAINT attendance_records_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES public.enrollments(course_id, user_id);


--
-- Name: auth_sessions auth_sessions_user_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.auth_sessions
    ADD CONSTRAINT auth_sessions_user_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: auth_tokens auth_tokens_user_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.auth_tokens
    ADD CONSTRAINT auth_tokens_user_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: calendar_events calendar_events_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.calendar_events
    ADD CONSTRAINT calendar_events_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: calendar_events calendar_events_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.calendar_events
    ADD CONSTRAINT calendar_events_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: chat_messages chat_messages_owner_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.chat_messages
    ADD CONSTRAINT chat_messages_owner_fkey FOREIGN KEY (session_id, user_id) REFERENCES public.chat_sessions(id, user_id) ON DELETE CASCADE;


--
-- Name: chat_messages chat_messages_reply_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.chat_messages
    ADD CONSTRAINT chat_messages_reply_fkey FOREIGN KEY (course_id, reply_to) REFERENCES public.chat_messages(course_id, id);


--
-- Name: chat_messages chat_messages_session_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.chat_messages
    ADD CONSTRAINT chat_messages_session_fkey FOREIGN KEY (course_id, session_id) REFERENCES public.chat_sessions(course_id, id) ON DELETE CASCADE;


--
-- Name: chat_sessions chat_sessions_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.chat_sessions
    ADD CONSTRAINT chat_sessions_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: chat_sessions chat_sessions_document_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.chat_sessions
    ADD CONSTRAINT chat_sessions_document_fkey FOREIGN KEY (document_id) REFERENCES public.documents(id) ON DELETE SET NULL;


--
-- Name: chat_sessions chat_sessions_user_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.chat_sessions
    ADD CONSTRAINT chat_sessions_user_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: checkin_windows checkin_windows_closed_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.checkin_windows
    ADD CONSTRAINT checkin_windows_closed_by_fkey FOREIGN KEY (closed_by) REFERENCES public.users(id);


--
-- Name: checkin_windows checkin_windows_opened_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.checkin_windows
    ADD CONSTRAINT checkin_windows_opened_by_fkey FOREIGN KEY (opened_by) REFERENCES public.users(id);


--
-- Name: checkin_windows checkin_windows_session_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.checkin_windows
    ADD CONSTRAINT checkin_windows_session_fkey FOREIGN KEY (course_id, session_id) REFERENCES public.class_sessions(course_id, id);


--
-- Name: class_sessions class_sessions_attendance_done_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.class_sessions
    ADD CONSTRAINT class_sessions_attendance_done_by_fkey FOREIGN KEY (attendance_done_by) REFERENCES public.users(id);


--
-- Name: class_sessions class_sessions_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.class_sessions
    ADD CONSTRAINT class_sessions_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: code_drafts code_drafts_attempt_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.code_drafts
    ADD CONSTRAINT code_drafts_attempt_fkey FOREIGN KEY (course_id, attempt_id) REFERENCES public.exam_attempts(course_id, id) ON DELETE CASCADE;


--
-- Name: code_drafts code_drafts_item_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.code_drafts
    ADD CONSTRAINT code_drafts_item_fkey FOREIGN KEY (course_id, item_id) REFERENCES public.exam_items(course_id, id) ON DELETE CASCADE;


--
-- Name: code_problems code_problems_question_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.code_problems
    ADD CONSTRAINT code_problems_question_fkey FOREIGN KEY (course_id, question_id) REFERENCES public.question_bank(course_id, id) ON DELETE CASCADE;


--
-- Name: code_submissions code_submissions_attempt_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.code_submissions
    ADD CONSTRAINT code_submissions_attempt_fkey FOREIGN KEY (course_id, attempt_id) REFERENCES public.exam_attempts(course_id, id);


--
-- Name: code_submissions code_submissions_exam_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.code_submissions
    ADD CONSTRAINT code_submissions_exam_fkey FOREIGN KEY (course_id, exam_id) REFERENCES public.exams(course_id, id);


--
-- Name: code_submissions code_submissions_item_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.code_submissions
    ADD CONSTRAINT code_submissions_item_fkey FOREIGN KEY (course_id, item_id) REFERENCES public.exam_items(course_id, id);


--
-- Name: code_submissions code_submissions_problem_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.code_submissions
    ADD CONSTRAINT code_submissions_problem_fkey FOREIGN KEY (course_id, problem_id) REFERENCES public.code_problems(course_id, question_id);


--
-- Name: code_submissions code_submissions_student_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.code_submissions
    ADD CONSTRAINT code_submissions_student_fkey FOREIGN KEY (student_id) REFERENCES public.users(id);


--
-- Name: code_testcases code_testcases_problem_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.code_testcases
    ADD CONSTRAINT code_testcases_problem_fkey FOREIGN KEY (course_id, problem_id) REFERENCES public.code_problems(course_id, question_id) ON DELETE CASCADE;


--
-- Name: content_chunks content_chunks_document_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.content_chunks
    ADD CONSTRAINT content_chunks_document_fkey FOREIGN KEY (document_id) REFERENCES public.documents(id) ON DELETE CASCADE;


--
-- Name: course_decision_gates course_decision_gates_course_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.course_decision_gates
    ADD CONSTRAINT course_decision_gates_course_id_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: course_decision_gates course_decision_gates_gate_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.course_decision_gates
    ADD CONSTRAINT course_decision_gates_gate_fkey FOREIGN KEY (gate) REFERENCES public.decision_gates(gate);


--
-- Name: course_guidance_flags course_guidance_flags_course_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.course_guidance_flags
    ADD CONSTRAINT course_guidance_flags_course_id_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id) ON DELETE CASCADE;


--
-- Name: course_guidance_flags course_guidance_flags_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.course_guidance_flags
    ADD CONSTRAINT course_guidance_flags_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id);


--
-- Name: courses courses_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.courses
    ADD CONSTRAINT courses_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: document_courses document_courses_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.document_courses
    ADD CONSTRAINT document_courses_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: document_courses document_courses_document_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.document_courses
    ADD CONSTRAINT document_courses_document_fkey FOREIGN KEY (document_id) REFERENCES public.documents(id) ON DELETE CASCADE;


--
-- Name: document_courses document_courses_shared_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.document_courses
    ADD CONSTRAINT document_courses_shared_by_fkey FOREIGN KEY (shared_by) REFERENCES public.users(id);


--
-- Name: documents documents_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.documents
    ADD CONSTRAINT documents_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: documents documents_uploaded_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.documents
    ADD CONSTRAINT documents_uploaded_by_fkey FOREIGN KEY (uploaded_by) REFERENCES public.users(id);


--
-- Name: enrollments enrollments_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.enrollments
    ADD CONSTRAINT enrollments_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: enrollments enrollments_user_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.enrollments
    ADD CONSTRAINT enrollments_user_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: escalation_tickets escalation_tickets_answered_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.escalation_tickets
    ADD CONSTRAINT escalation_tickets_answered_by_fkey FOREIGN KEY (answered_by) REFERENCES public.users(id);


--
-- Name: escalation_tickets escalation_tickets_claimed_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.escalation_tickets
    ADD CONSTRAINT escalation_tickets_claimed_by_fkey FOREIGN KEY (claimed_by) REFERENCES public.users(id);


--
-- Name: escalation_tickets escalation_tickets_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.escalation_tickets
    ADD CONSTRAINT escalation_tickets_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: escalation_tickets escalation_tickets_message_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.escalation_tickets
    ADD CONSTRAINT escalation_tickets_message_fkey FOREIGN KEY (course_id, message_id) REFERENCES public.chat_messages(course_id, id);


--
-- Name: escalation_tickets escalation_tickets_session_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.escalation_tickets
    ADD CONSTRAINT escalation_tickets_session_fkey FOREIGN KEY (course_id, session_id) REFERENCES public.chat_sessions(course_id, id);


--
-- Name: escalation_tickets escalation_tickets_student_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.escalation_tickets
    ADD CONSTRAINT escalation_tickets_student_fkey FOREIGN KEY (student_id) REFERENCES public.users(id);


--
-- Name: exam_answers exam_answers_attempt_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exam_answers
    ADD CONSTRAINT exam_answers_attempt_fkey FOREIGN KEY (course_id, attempt_id) REFERENCES public.exam_attempts(course_id, id) ON DELETE CASCADE;


--
-- Name: exam_answers exam_answers_item_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exam_answers
    ADD CONSTRAINT exam_answers_item_fkey FOREIGN KEY (course_id, item_id) REFERENCES public.exam_items(course_id, id) ON DELETE CASCADE;


--
-- Name: exam_appeals exam_appeals_attempt_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exam_appeals
    ADD CONSTRAINT exam_appeals_attempt_fkey FOREIGN KEY (course_id, attempt_id) REFERENCES public.exam_attempts(course_id, id) ON DELETE CASCADE;


--
-- Name: exam_appeals exam_appeals_exam_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exam_appeals
    ADD CONSTRAINT exam_appeals_exam_fkey FOREIGN KEY (course_id, exam_id) REFERENCES public.exams(course_id, id);


--
-- Name: exam_appeals exam_appeals_responded_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exam_appeals
    ADD CONSTRAINT exam_appeals_responded_by_fkey FOREIGN KEY (responded_by) REFERENCES public.users(id);


--
-- Name: exam_appeals exam_appeals_student_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exam_appeals
    ADD CONSTRAINT exam_appeals_student_fkey FOREIGN KEY (student_id) REFERENCES public.users(id);


--
-- Name: exam_attempts exam_attempts_adjusted_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exam_attempts
    ADD CONSTRAINT exam_attempts_adjusted_by_fkey FOREIGN KEY (adjusted_by) REFERENCES public.users(id);


--
-- Name: exam_attempts exam_attempts_exam_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exam_attempts
    ADD CONSTRAINT exam_attempts_exam_fkey FOREIGN KEY (course_id, exam_id) REFERENCES public.exams(course_id, id);


--
-- Name: exam_attempts exam_attempts_student_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exam_attempts
    ADD CONSTRAINT exam_attempts_student_fkey FOREIGN KEY (student_id) REFERENCES public.users(id);


--
-- Name: exam_events exam_events_attempt_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exam_events
    ADD CONSTRAINT exam_events_attempt_fkey FOREIGN KEY (course_id, attempt_id) REFERENCES public.exam_attempts(course_id, id) ON DELETE CASCADE;


--
-- Name: exam_events exam_events_exam_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exam_events
    ADD CONSTRAINT exam_events_exam_fkey FOREIGN KEY (course_id, exam_id) REFERENCES public.exams(course_id, id);


--
-- Name: exam_events exam_events_student_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exam_events
    ADD CONSTRAINT exam_events_student_fkey FOREIGN KEY (student_id) REFERENCES public.users(id);


--
-- Name: exam_items exam_items_exam_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exam_items
    ADD CONSTRAINT exam_items_exam_fkey FOREIGN KEY (course_id, exam_id) REFERENCES public.exams(course_id, id) ON DELETE CASCADE;


--
-- Name: exam_items exam_items_question_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exam_items
    ADD CONSTRAINT exam_items_question_fkey FOREIGN KEY (course_id, question_id) REFERENCES public.question_bank(course_id, id);


--
-- Name: exams exams_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exams
    ADD CONSTRAINT exams_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: exams exams_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.exams
    ADD CONSTRAINT exams_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: final_grade_snapshots final_grade_snapshots_locked_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.final_grade_snapshots
    ADD CONSTRAINT final_grade_snapshots_locked_by_fkey FOREIGN KEY (locked_by) REFERENCES public.users(id);


--
-- Name: final_grade_snapshots final_grade_snapshots_scheme_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.final_grade_snapshots
    ADD CONSTRAINT final_grade_snapshots_scheme_fkey FOREIGN KEY (course_id, scheme_id) REFERENCES public.grade_schemes(course_id, id);


--
-- Name: final_grade_snapshots final_grade_snapshots_student_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.final_grade_snapshots
    ADD CONSTRAINT final_grade_snapshots_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES public.enrollments(course_id, user_id);


--
-- Name: forms_import_batches forms_import_batches_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.forms_import_batches
    ADD CONSTRAINT forms_import_batches_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: forms_import_batches forms_import_batches_item_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.forms_import_batches
    ADD CONSTRAINT forms_import_batches_item_fkey FOREIGN KEY (course_id, item_id) REFERENCES public.grade_items(course_id, id);


--
-- Name: forms_import_rows forms_import_rows_batch_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.forms_import_rows
    ADD CONSTRAINT forms_import_rows_batch_fkey FOREIGN KEY (course_id, batch_id) REFERENCES public.forms_import_batches(course_id, id) ON DELETE CASCADE;


--
-- Name: forms_import_rows forms_import_rows_student_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.forms_import_rows
    ADD CONSTRAINT forms_import_rows_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES public.enrollments(course_id, user_id);


--
-- Name: forum_posts forum_posts_author_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.forum_posts
    ADD CONSTRAINT forum_posts_author_fkey FOREIGN KEY (author_id) REFERENCES public.users(id);


--
-- Name: forum_posts forum_posts_hidden_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.forum_posts
    ADD CONSTRAINT forum_posts_hidden_by_fkey FOREIGN KEY (hidden_by) REFERENCES public.users(id);


--
-- Name: forum_posts forum_posts_thread_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.forum_posts
    ADD CONSTRAINT forum_posts_thread_fkey FOREIGN KEY (course_id, thread_id) REFERENCES public.forum_threads(course_id, id) ON DELETE CASCADE;


--
-- Name: forum_posts forum_posts_verified_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.forum_posts
    ADD CONSTRAINT forum_posts_verified_by_fkey FOREIGN KEY (verified_by) REFERENCES public.users(id);


--
-- Name: forum_threads forum_threads_author_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.forum_threads
    ADD CONSTRAINT forum_threads_author_fkey FOREIGN KEY (author_id) REFERENCES public.users(id);


--
-- Name: forum_threads forum_threads_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.forum_threads
    ADD CONSTRAINT forum_threads_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: forum_threads forum_threads_hidden_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.forum_threads
    ADD CONSTRAINT forum_threads_hidden_by_fkey FOREIGN KEY (hidden_by) REFERENCES public.users(id);


--
-- Name: forum_threads forum_threads_similar_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.forum_threads
    ADD CONSTRAINT forum_threads_similar_fkey FOREIGN KEY (course_id, similar_of) REFERENCES public.forum_threads(course_id, id);


--
-- Name: grade_appeals grade_appeals_assignment_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_appeals
    ADD CONSTRAINT grade_appeals_assignment_fkey FOREIGN KEY (course_id, assignment_id) REFERENCES public.assignments(course_id, id);


--
-- Name: grade_appeals grade_appeals_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_appeals
    ADD CONSTRAINT grade_appeals_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: grade_appeals grade_appeals_decided_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_appeals
    ADD CONSTRAINT grade_appeals_decided_by_fkey FOREIGN KEY (decided_by) REFERENCES public.users(id);


--
-- Name: grade_appeals grade_appeals_result_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_appeals
    ADD CONSTRAINT grade_appeals_result_fkey FOREIGN KEY (course_id, result_id) REFERENCES public.grading_results(course_id, id);


--
-- Name: grade_appeals grade_appeals_student_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_appeals
    ADD CONSTRAINT grade_appeals_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES public.enrollments(course_id, user_id);


--
-- Name: grade_appeals grade_appeals_submission_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_appeals
    ADD CONSTRAINT grade_appeals_submission_fkey FOREIGN KEY (course_id, submission_id) REFERENCES public.submissions(course_id, id);


--
-- Name: grade_components grade_components_scheme_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_components
    ADD CONSTRAINT grade_components_scheme_fkey FOREIGN KEY (course_id, scheme_id) REFERENCES public.grade_schemes(course_id, id) ON DELETE CASCADE;


--
-- Name: grade_entries grade_entries_entered_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_entries
    ADD CONSTRAINT grade_entries_entered_by_fkey FOREIGN KEY (entered_by) REFERENCES public.users(id);


--
-- Name: grade_entries grade_entries_item_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_entries
    ADD CONSTRAINT grade_entries_item_fkey FOREIGN KEY (course_id, item_id) REFERENCES public.grade_items(course_id, id) ON DELETE CASCADE;


--
-- Name: grade_entries grade_entries_student_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_entries
    ADD CONSTRAINT grade_entries_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES public.enrollments(course_id, user_id);


--
-- Name: grade_items grade_items_assignment_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_items
    ADD CONSTRAINT grade_items_assignment_fkey FOREIGN KEY (course_id, assignment_id) REFERENCES public.assignments(course_id, id);


--
-- Name: grade_items grade_items_component_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_items
    ADD CONSTRAINT grade_items_component_fkey FOREIGN KEY (course_id, component_id) REFERENCES public.grade_components(course_id, id) ON DELETE CASCADE;


--
-- Name: grade_items grade_items_exam_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_items
    ADD CONSTRAINT grade_items_exam_fkey FOREIGN KEY (course_id, exam_id) REFERENCES public.exams(course_id, id);


--
-- Name: grade_items grade_items_scheme_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_items
    ADD CONSTRAINT grade_items_scheme_fkey FOREIGN KEY (course_id, scheme_id) REFERENCES public.grade_schemes(course_id, id) ON DELETE CASCADE;


--
-- Name: grade_rules grade_rules_scheme_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_rules
    ADD CONSTRAINT grade_rules_scheme_fkey FOREIGN KEY (course_id, scheme_id) REFERENCES public.grade_schemes(course_id, id) ON DELETE CASCADE;


--
-- Name: grade_schemes grade_schemes_confirmed_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_schemes
    ADD CONSTRAINT grade_schemes_confirmed_by_fkey FOREIGN KEY (confirmed_by) REFERENCES public.users(id);


--
-- Name: grade_schemes grade_schemes_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_schemes
    ADD CONSTRAINT grade_schemes_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: grade_schemes grade_schemes_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_schemes
    ADD CONSTRAINT grade_schemes_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: grade_schemes grade_schemes_document_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_schemes
    ADD CONSTRAINT grade_schemes_document_fkey FOREIGN KEY (source_document_id) REFERENCES public.documents(id) ON DELETE SET NULL;


--
-- Name: grade_schemes grade_schemes_job_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grade_schemes
    ADD CONSTRAINT grade_schemes_job_fkey FOREIGN KEY (extract_job_id) REFERENCES public.jobs(id) ON DELETE SET NULL;


--
-- Name: gradebooks gradebooks_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.gradebooks
    ADD CONSTRAINT gradebooks_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: gradebooks gradebooks_unlocked_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.gradebooks
    ADD CONSTRAINT gradebooks_unlocked_by_fkey FOREIGN KEY (unlocked_by) REFERENCES public.users(id);


--
-- Name: grading_guidance_items grading_guidance_items_course_id_assignment_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grading_guidance_items
    ADD CONSTRAINT grading_guidance_items_course_id_assignment_id_fkey FOREIGN KEY (course_id, assignment_id) REFERENCES public.assignments(course_id, id) ON DELETE CASCADE;


--
-- Name: grading_guidance_items grading_guidance_items_decided_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grading_guidance_items
    ADD CONSTRAINT grading_guidance_items_decided_by_fkey FOREIGN KEY (decided_by) REFERENCES public.users(id);


--
-- Name: grading_guidance_versions grading_guidance_versions_course_id_assignment_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grading_guidance_versions
    ADD CONSTRAINT grading_guidance_versions_course_id_assignment_id_fkey FOREIGN KEY (course_id, assignment_id) REFERENCES public.assignments(course_id, id) ON DELETE CASCADE;


--
-- Name: grading_guidance_versions grading_guidance_versions_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grading_guidance_versions
    ADD CONSTRAINT grading_guidance_versions_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: grading_regression_cases grading_regression_cases_course_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grading_regression_cases
    ADD CONSTRAINT grading_regression_cases_course_id_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id) ON DELETE CASCADE;


--
-- Name: grading_results grading_results_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grading_results
    ADD CONSTRAINT grading_results_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: grading_results grading_results_first_reviewed_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grading_results
    ADD CONSTRAINT grading_results_first_reviewed_by_fkey FOREIGN KEY (first_reviewed_by) REFERENCES public.users(id);


--
-- Name: grading_results grading_results_published_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grading_results
    ADD CONSTRAINT grading_results_published_by_fkey FOREIGN KEY (published_by) REFERENCES public.users(id);


--
-- Name: grading_results grading_results_reviewed_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grading_results
    ADD CONSTRAINT grading_results_reviewed_by_fkey FOREIGN KEY (reviewed_by) REFERENCES public.users(id);


--
-- Name: grading_results grading_results_submission_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.grading_results
    ADD CONSTRAINT grading_results_submission_fkey FOREIGN KEY (course_id, submission_id) REFERENCES public.submissions(course_id, id);


--
-- Name: insight_reports insight_reports_course_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.insight_reports
    ADD CONSTRAINT insight_reports_course_id_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: insight_reports insight_reports_generated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.insight_reports
    ADD CONSTRAINT insight_reports_generated_by_fkey FOREIGN KEY (generated_by) REFERENCES public.users(id);


--
-- Name: integration_configs integration_configs_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.integration_configs
    ADD CONSTRAINT integration_configs_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: knowledge_items knowledge_items_chunk_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.knowledge_items
    ADD CONSTRAINT knowledge_items_chunk_fkey FOREIGN KEY (chunk_id) REFERENCES public.content_chunks(id) ON DELETE CASCADE;


--
-- Name: knowledge_items knowledge_items_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.knowledge_items
    ADD CONSTRAINT knowledge_items_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: knowledge_items knowledge_items_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.knowledge_items
    ADD CONSTRAINT knowledge_items_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: llm_models llm_models_provider_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.llm_models
    ADD CONSTRAINT llm_models_provider_id_fkey FOREIGN KEY (provider_id) REFERENCES public.llm_providers(id) ON DELETE CASCADE;


--
-- Name: llm_task_routes llm_task_routes_model_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.llm_task_routes
    ADD CONSTRAINT llm_task_routes_model_id_fkey FOREIGN KEY (model_id) REFERENCES public.llm_models(id);


--
-- Name: mail_inbound mail_inbound_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.mail_inbound
    ADD CONSTRAINT mail_inbound_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: mail_inbound mail_inbound_submission_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.mail_inbound
    ADD CONSTRAINT mail_inbound_submission_fkey FOREIGN KEY (course_id, submission_id) REFERENCES public.submissions(course_id, id);


--
-- Name: notifications notifications_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: notifications notifications_user_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_user_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: participation_events participation_events_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.participation_events
    ADD CONSTRAINT participation_events_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: participation_events participation_events_session_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.participation_events
    ADD CONSTRAINT participation_events_session_fkey FOREIGN KEY (course_id, session_id) REFERENCES public.class_sessions(course_id, id);


--
-- Name: participation_events participation_events_student_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.participation_events
    ADD CONSTRAINT participation_events_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES public.enrollments(course_id, user_id);


--
-- Name: pii_events pii_events_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.pii_events
    ADD CONSTRAINT pii_events_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: pii_events pii_events_session_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.pii_events
    ADD CONSTRAINT pii_events_session_fkey FOREIGN KEY (course_id, session_id) REFERENCES public.chat_sessions(course_id, id);


--
-- Name: pii_events pii_events_user_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.pii_events
    ADD CONSTRAINT pii_events_user_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: post_reports post_reports_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.post_reports
    ADD CONSTRAINT post_reports_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: post_reports post_reports_handled_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.post_reports
    ADD CONSTRAINT post_reports_handled_by_fkey FOREIGN KEY (handled_by) REFERENCES public.users(id);


--
-- Name: post_reports post_reports_post_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.post_reports
    ADD CONSTRAINT post_reports_post_fkey FOREIGN KEY (course_id, post_id) REFERENCES public.forum_posts(course_id, id);


--
-- Name: post_reports post_reports_reporter_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.post_reports
    ADD CONSTRAINT post_reports_reporter_fkey FOREIGN KEY (reporter_id) REFERENCES public.users(id);


--
-- Name: post_reports post_reports_thread_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.post_reports
    ADD CONSTRAINT post_reports_thread_fkey FOREIGN KEY (course_id, thread_id) REFERENCES public.forum_threads(course_id, id);


--
-- Name: practice_answers practice_answers_attempt_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.practice_answers
    ADD CONSTRAINT practice_answers_attempt_fkey FOREIGN KEY (course_id, attempt_id) REFERENCES public.practice_attempts(course_id, id) ON DELETE CASCADE;


--
-- Name: practice_answers practice_answers_item_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.practice_answers
    ADD CONSTRAINT practice_answers_item_fkey FOREIGN KEY (course_id, item_id) REFERENCES public.practice_items(course_id, id) ON DELETE CASCADE;


--
-- Name: practice_attempts practice_attempts_student_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.practice_attempts
    ADD CONSTRAINT practice_attempts_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES public.enrollments(course_id, user_id);


--
-- Name: practice_items practice_items_attempt_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.practice_items
    ADD CONSTRAINT practice_items_attempt_fkey FOREIGN KEY (course_id, attempt_id) REFERENCES public.practice_attempts(course_id, id) ON DELETE CASCADE;


--
-- Name: practice_items practice_items_question_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.practice_items
    ADD CONSTRAINT practice_items_question_fkey FOREIGN KEY (course_id, question_id) REFERENCES public.question_bank(course_id, id);


--
-- Name: question_bank question_bank_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.question_bank
    ADD CONSTRAINT question_bank_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: question_bank question_bank_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.question_bank
    ADD CONSTRAINT question_bank_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: question_bank question_bank_reviewed_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.question_bank
    ADD CONSTRAINT question_bank_reviewed_by_fkey FOREIGN KEY (reviewed_by) REFERENCES public.users(id);


--
-- Name: question_options question_options_question_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.question_options
    ADD CONSTRAINT question_options_question_fkey FOREIGN KEY (course_id, question_id) REFERENCES public.question_bank(course_id, id) ON DELETE CASCADE;


--
-- Name: reminder_log reminder_log_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.reminder_log
    ADD CONSTRAINT reminder_log_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: reminder_log reminder_log_user_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.reminder_log
    ADD CONSTRAINT reminder_log_user_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: similarity_reports similarity_reports_att_a_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.similarity_reports
    ADD CONSTRAINT similarity_reports_att_a_fkey FOREIGN KEY (course_id, attempt_a) REFERENCES public.exam_attempts(course_id, id) ON DELETE CASCADE;


--
-- Name: similarity_reports similarity_reports_att_b_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.similarity_reports
    ADD CONSTRAINT similarity_reports_att_b_fkey FOREIGN KEY (course_id, attempt_b) REFERENCES public.exam_attempts(course_id, id) ON DELETE CASCADE;


--
-- Name: similarity_reports similarity_reports_exam_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.similarity_reports
    ADD CONSTRAINT similarity_reports_exam_fkey FOREIGN KEY (course_id, exam_id) REFERENCES public.exams(course_id, id);


--
-- Name: similarity_reports similarity_reports_problem_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.similarity_reports
    ADD CONSTRAINT similarity_reports_problem_fkey FOREIGN KEY (course_id, problem_id) REFERENCES public.code_problems(course_id, question_id);


--
-- Name: similarity_reports similarity_reports_reviewed_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.similarity_reports
    ADD CONSTRAINT similarity_reports_reviewed_by_fkey FOREIGN KEY (reviewed_by) REFERENCES public.users(id);


--
-- Name: similarity_reports similarity_reports_sub_a_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.similarity_reports
    ADD CONSTRAINT similarity_reports_sub_a_fkey FOREIGN KEY (course_id, submission_a) REFERENCES public.code_submissions(course_id, id) ON DELETE CASCADE;


--
-- Name: similarity_reports similarity_reports_sub_b_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.similarity_reports
    ADD CONSTRAINT similarity_reports_sub_b_fkey FOREIGN KEY (course_id, submission_b) REFERENCES public.code_submissions(course_id, id) ON DELETE CASCADE;


--
-- Name: student_observations student_observations_author_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.student_observations
    ADD CONSTRAINT student_observations_author_fkey FOREIGN KEY (author_id) REFERENCES public.users(id);


--
-- Name: student_observations student_observations_student_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.student_observations
    ADD CONSTRAINT student_observations_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES public.enrollments(course_id, user_id);


--
-- Name: student_risk student_risk_student_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.student_risk
    ADD CONSTRAINT student_risk_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES public.enrollments(course_id, user_id);


--
-- Name: study_time_daily study_time_daily_student_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.study_time_daily
    ADD CONSTRAINT study_time_daily_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES public.enrollments(course_id, user_id);


--
-- Name: submissions submissions_assignment_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.submissions
    ADD CONSTRAINT submissions_assignment_fkey FOREIGN KEY (course_id, assignment_id) REFERENCES public.assignments(course_id, id);


--
-- Name: submissions submissions_candidate_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.submissions
    ADD CONSTRAINT submissions_candidate_fkey FOREIGN KEY (course_id, candidate_student_id) REFERENCES public.enrollments(course_id, user_id);


--
-- Name: submissions submissions_course_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.submissions
    ADD CONSTRAINT submissions_course_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id);


--
-- Name: submissions submissions_dismissed_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.submissions
    ADD CONSTRAINT submissions_dismissed_by_fkey FOREIGN KEY (dismissed_by) REFERENCES public.users(id);


--
-- Name: submissions submissions_student_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.submissions
    ADD CONSTRAINT submissions_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES public.enrollments(course_id, user_id);


--
-- Name: user_settings user_settings_user_fkey; Type: FK CONSTRAINT; Schema: public; Owner: edupilot
--

ALTER TABLE ONLY public.user_settings
    ADD CONSTRAINT user_settings_user_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict s4CmlBMXb9xsWfa7AMqCEGPrEZPQcZobkxKoyXHRzmjzSmCNXLPyZg9vghCiERw


-- Tài khoản an toàn (SRS FEAT-account-security mục 5): auth_sessions, auth_tokens, login_attempts, mail_outbox.
-- Không ALTER `users` (đã đủ cột từ 00001). Bốn bảng là bảng nền (không thuộc lớp): ngoại lệ có chủ ý của luật 13.

-- +goose Up
CREATE TYPE auth_token_kind AS ENUM ('VERIFY_EMAIL', 'RESET_PASSWORD', 'INVITE');
CREATE TYPE mail_status AS ENUM ('QUEUED', 'SENT', 'DEAD');
CREATE TYPE login_outcome AS ENUM ('SUCCESS', 'BAD_PASSWORD', 'UNKNOWN_EMAIL', 'THROTTLED', 'LOCKED', 'DISABLED');

CREATE TABLE auth_sessions (
    id                  uuid        NOT NULL DEFAULT uuidv7(),
    user_id             uuid        NOT NULL,
    refresh_hash        char(64)    NOT NULL,
    prev_refresh_hash   char(64),
    user_agent          text,
    device_label        text,
    ip                  inet,
    created_at          timestamptz NOT NULL DEFAULT now(),
    rotated_at          timestamptz,
    last_used_at        timestamptz NOT NULL DEFAULT now(),
    expires_at          timestamptz NOT NULL,
    absolute_expires_at timestamptz NOT NULL,
    revoked_at          timestamptz,
    revoked_reason      text,
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT auth_sessions_pkey PRIMARY KEY (id),
    CONSTRAINT auth_sessions_user_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT auth_sessions_refresh_hash_chk CHECK (refresh_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT auth_sessions_prev_hash_chk CHECK (prev_refresh_hash IS NULL OR prev_refresh_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT auth_sessions_user_agent_chk CHECK (char_length(user_agent) <= 300),
    CONSTRAINT auth_sessions_device_label_chk CHECK (char_length(device_label) <= 80),
    CONSTRAINT auth_sessions_absolute_chk CHECK (absolute_expires_at >= expires_at),
    CONSTRAINT auth_sessions_reason_chk CHECK (revoked_reason IN ('LOGOUT', 'REVOKED_BY_USER', 'PASSWORD_CHANGED', 'PASSWORD_RESET', 'REFRESH_REUSE', 'ACCOUNT_DISABLED', 'ROLE_CHANGED', 'ADMIN')),
    CONSTRAINT auth_sessions_revoked_pair_chk CHECK ((revoked_at IS NULL) = (revoked_reason IS NULL))
);
CREATE UNIQUE INDEX auth_sessions_refresh_hash_key ON auth_sessions (refresh_hash);
CREATE INDEX auth_sessions_prev_hash_idx ON auth_sessions (prev_refresh_hash) WHERE prev_refresh_hash IS NOT NULL;
CREATE INDEX auth_sessions_user_active_idx ON auth_sessions (user_id, last_used_at DESC, id DESC) WHERE revoked_at IS NULL;
CREATE INDEX auth_sessions_expires_idx ON auth_sessions (expires_at);

CREATE TABLE auth_tokens (
    id         uuid            NOT NULL DEFAULT uuidv7(),
    user_id    uuid            NOT NULL,
    kind       auth_token_kind NOT NULL,
    token_hash char(64)        NOT NULL,
    expires_at timestamptz     NOT NULL,
    used_at    timestamptz,
    revoked_at timestamptz,
    created_by uuid,
    created_at timestamptz     NOT NULL DEFAULT now(),
    CONSTRAINT auth_tokens_pkey PRIMARY KEY (id),
    CONSTRAINT auth_tokens_user_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT auth_tokens_hash_chk CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT auth_tokens_terminal_chk CHECK (NOT (used_at IS NOT NULL AND revoked_at IS NOT NULL))
);
CREATE UNIQUE INDEX auth_tokens_token_hash_key ON auth_tokens (token_hash);
CREATE INDEX auth_tokens_user_kind_idx ON auth_tokens (user_id, kind, created_at DESC);

CREATE TABLE login_attempts (
    id         uuid          NOT NULL DEFAULT uuidv7(),
    email_hash char(64)      NOT NULL,
    user_id    uuid,
    ip         inet,
    user_agent text,
    outcome    login_outcome NOT NULL,
    created_at timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT login_attempts_pkey PRIMARY KEY (id),
    CONSTRAINT login_attempts_email_hash_chk CHECK (email_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT login_attempts_user_agent_chk CHECK (char_length(user_agent) <= 200)
);
CREATE INDEX login_attempts_email_idx ON login_attempts (email_hash, created_at DESC);
CREATE INDEX login_attempts_ip_idx ON login_attempts (ip, created_at DESC);
CREATE INDEX login_attempts_user_idx ON login_attempts (user_id, created_at DESC) WHERE user_id IS NOT NULL;

CREATE TABLE mail_outbox (
    id         uuid        NOT NULL DEFAULT uuidv7(),
    to_addr    text        NOT NULL,
    template   text        NOT NULL,
    payload    jsonb       NOT NULL DEFAULT '{}',
    status     mail_status NOT NULL DEFAULT 'QUEUED',
    attempts   integer     NOT NULL DEFAULT 0,
    last_error text,
    dedupe_key text,
    sent_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT mail_outbox_pkey PRIMARY KEY (id),
    CONSTRAINT mail_outbox_to_addr_chk CHECK (to_addr = lower(to_addr) AND to_addr ~ '^[^@\s]+@[^@\s]+$'),
    CONSTRAINT mail_outbox_template_chk CHECK (template ~ '^[a-z][a-z0-9_]{0,63}$'),
    CONSTRAINT mail_outbox_payload_chk CHECK (jsonb_typeof(payload) = 'object'),
    CONSTRAINT mail_outbox_attempts_chk CHECK (attempts BETWEEN 0 AND 4),
    CONSTRAINT mail_outbox_last_error_chk CHECK (char_length(last_error) <= 1000),
    CONSTRAINT mail_outbox_sent_chk CHECK ((status = 'SENT') = (sent_at IS NOT NULL))
);
CREATE UNIQUE INDEX mail_outbox_dedupe_key_key ON mail_outbox (dedupe_key) WHERE dedupe_key IS NOT NULL;
CREATE INDEX mail_outbox_queued_idx ON mail_outbox (created_at, id) WHERE status = 'QUEUED';

CREATE TRIGGER auth_sessions_set_updated_at BEFORE UPDATE ON auth_sessions FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER mail_outbox_set_updated_at BEFORE UPDATE ON mail_outbox FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS mail_outbox;
DROP TABLE IF EXISTS login_attempts;
DROP TABLE IF EXISTS auth_tokens;
DROP TABLE IF EXISTS auth_sessions;
DROP TYPE IF EXISTS login_outcome;
DROP TYPE IF EXISTS mail_status;
DROP TYPE IF EXISTS auth_token_kind;

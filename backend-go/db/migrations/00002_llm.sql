-- Cổng LLM (SRS FEAT-llm-gateway mục 5): 5 bảng dạng cuối. Không FK tới users / courses (courses ra đời ở P2;
-- llm_audit giữ tham chiếu mềm để sống lâu hơn tài khoản). Tiền là numeric (VND). Khoá API chỉ ở dạng mã hoá (api_key_enc).

-- +goose Up
CREATE TABLE llm_providers (
    id            uuid        NOT NULL DEFAULT uuidv7(),
    type          text        NOT NULL,
    name          text        NOT NULL,
    base_url      text,
    api_key_enc   bytea,        -- 0x01 ‖ nonce(12) ‖ ciphertext ‖ tag(16); null = chưa có khoá
    enabled       boolean     NOT NULL DEFAULT true,
    rpm_limit     integer,
    tpm_limit     integer,
    last_test_ok  boolean,      -- null = chưa kiểm tra / lưu bằng skip_verify
    last_test_at  timestamptz,
    last_test_error text,       -- chỉ error_kind, không thân lỗi
    version       integer     NOT NULL DEFAULT 1,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT llm_providers_pkey PRIMARY KEY (id),
    CONSTRAINT llm_providers_type_chk CHECK (type IN ('openai', 'anthropic', 'gemini', 'openai_compatible', 'fake')),
    CONSTRAINT llm_providers_name_chk CHECK (char_length(name) BETWEEN 1 AND 60),
    CONSTRAINT llm_providers_rpm_chk CHECK (rpm_limit IS NULL OR rpm_limit > 0),
    CONSTRAINT llm_providers_tpm_chk CHECK (tpm_limit IS NULL OR tpm_limit > 0),
    CONSTRAINT llm_providers_name_uq UNIQUE (name),
    CONSTRAINT llm_providers_base_url_chk CHECK (type <> 'openai_compatible' OR base_url IS NOT NULL)
);

CREATE TABLE llm_models (
    id          uuid          NOT NULL DEFAULT uuidv7(),
    provider_id uuid          NOT NULL REFERENCES llm_providers (id) ON DELETE CASCADE,
    model       text          NOT NULL,
    kind        text          NOT NULL,
    dims        integer,
    price_in    numeric(14,4) NOT NULL DEFAULT 0,   -- đ / 1 triệu token
    price_out   numeric(14,4) NOT NULL DEFAULT 0,
    enabled     boolean       NOT NULL DEFAULT true,
    created_at  timestamptz   NOT NULL DEFAULT now(),
    updated_at  timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT llm_models_pkey PRIMARY KEY (id),
    CONSTRAINT llm_models_model_chk CHECK (char_length(model) BETWEEN 1 AND 120),
    CONSTRAINT llm_models_kind_chk CHECK (kind IN ('chat', 'embedding')),
    CONSTRAINT llm_models_dims_pos_chk CHECK (dims IS NULL OR dims > 0),
    CONSTRAINT llm_models_price_in_chk CHECK (price_in >= 0),
    CONSTRAINT llm_models_price_out_chk CHECK (price_out >= 0),
    CONSTRAINT llm_models_uq UNIQUE (provider_id, model),
    CONSTRAINT llm_models_dims_chk CHECK (kind <> 'embedding' OR dims IS NOT NULL)
);
CREATE INDEX llm_models_provider_idx ON llm_models (provider_id);

CREATE TABLE llm_task_routes (
    id             uuid        NOT NULL DEFAULT uuidv7(),
    task           text        NOT NULL,
    model_id       uuid        NOT NULL REFERENCES llm_models (id),   -- không cascade: xoá mô hình đang dùng bị chặn
    fallback_order integer     NOT NULL,                               -- 0 = chính, 1.. = dự phòng
    params         jsonb       NOT NULL DEFAULT '{}'::jsonb,           -- temperature, max_tokens, timeout_s, retries
    version        integer     NOT NULL DEFAULT 1,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT llm_task_routes_pkey PRIMARY KEY (id),
    CONSTRAINT llm_task_routes_task_chk CHECK (task IN ('CHAT', 'CLASSIFY', 'UTILITY', 'GRADING', 'QUESTION_GEN', 'INSIGHT', 'EMBEDDING')),
    CONSTRAINT llm_task_routes_order_chk CHECK (fallback_order >= 0),
    CONSTRAINT llm_task_routes_uq UNIQUE (task, fallback_order)
);
CREATE INDEX llm_task_routes_model_idx ON llm_task_routes (model_id);

-- ponytail: chưa phân vùng theo tháng, chưa dọn dữ liệu cũ (Nợ PR). Ở T1 ≈ 1.000 SV × vài chục lượt/ngày thì 12 tháng
-- vẫn chấp nhận được; các chỉ mục theo thời gian đủ cho GET usage. Nâng cấp: PARTITION BY RANGE (created_at).
CREATE TABLE llm_audit (
    id               uuid          NOT NULL DEFAULT uuidv7(),
    task             text          NOT NULL,
    lane             text          NOT NULL,
    provider         text,          -- tên nhà cung cấp lúc gọi (không FK: sống lâu hơn cấu hình)
    model            text,
    tokens_in        integer       NOT NULL DEFAULT 0,
    tokens_out       integer       NOT NULL DEFAULT 0,
    latency_ms       integer       NOT NULL DEFAULT 0,
    queue_wait_ms    integer       NOT NULL DEFAULT 0,
    attempts         integer       NOT NULL DEFAULT 1,
    fallback_index   integer       NOT NULL DEFAULT 0,
    cost_est         numeric(14,4) NOT NULL DEFAULT 0,
    status           text          NOT NULL,
    error_kind       text,
    degraded         boolean       NOT NULL DEFAULT false,
    pii_masked_count integer       NOT NULL DEFAULT 0,
    user_id          uuid,          -- null cho việc hệ thống
    course_id        uuid,          -- null cho việc ngoài lớp
    trace_id         text          NOT NULL,
    created_at       timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT llm_audit_pkey PRIMARY KEY (id),
    CONSTRAINT llm_audit_lane_chk CHECK (lane IN ('INTERACTIVE', 'NEAR_REALTIME', 'BATCH')),
    CONSTRAINT llm_audit_status_chk CHECK (status IN ('ok', 'error', 'timeout', 'rate_limited', 'overloaded', 'degraded', 'cancelled', 'circuit_open', 'budget_blocked', 'not_configured'))
);
CREATE INDEX llm_audit_course_created_idx ON llm_audit (course_id, created_at DESC) WHERE course_id IS NOT NULL;
CREATE INDEX llm_audit_created_idx ON llm_audit (created_at DESC);
CREATE INDEX llm_audit_task_created_idx ON llm_audit (task, created_at DESC);
CREATE INDEX llm_audit_trace_idx ON llm_audit (trace_id);

CREATE TABLE llm_budgets (
    id            uuid          NOT NULL DEFAULT uuidv7(),
    scope         text          NOT NULL,
    course_id     uuid,
    daily_limit   numeric(14,2),
    monthly_limit numeric(14,2),
    version       integer       NOT NULL DEFAULT 1,
    created_at    timestamptz   NOT NULL DEFAULT now(),
    updated_at    timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT llm_budgets_pkey PRIMARY KEY (id),
    CONSTRAINT llm_budgets_scope_chk CHECK (scope IN ('system', 'course')),
    CONSTRAINT llm_budgets_daily_chk CHECK (daily_limit IS NULL OR daily_limit >= 0),
    CONSTRAINT llm_budgets_monthly_chk CHECK (monthly_limit IS NULL OR monthly_limit >= 0),
    CONSTRAINT llm_budgets_scope_course_chk CHECK ((scope = 'system' AND course_id IS NULL) OR (scope = 'course' AND course_id IS NOT NULL)),
    CONSTRAINT llm_budgets_order_chk CHECK (daily_limit IS NULL OR monthly_limit IS NULL OR daily_limit <= monthly_limit)
);
CREATE UNIQUE INDEX llm_budgets_system_uq ON llm_budgets (scope) WHERE scope = 'system';
CREATE UNIQUE INDEX llm_budgets_course_uq ON llm_budgets (course_id) WHERE scope = 'course';

CREATE TRIGGER llm_providers_set_updated_at BEFORE UPDATE ON llm_providers FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER llm_models_set_updated_at BEFORE UPDATE ON llm_models FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER llm_task_routes_set_updated_at BEFORE UPDATE ON llm_task_routes FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER llm_budgets_set_updated_at BEFORE UPDATE ON llm_budgets FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS llm_budgets;
DROP TABLE IF EXISTS llm_audit;
DROP TABLE IF EXISTS llm_task_routes;
DROP TABLE IF EXISTS llm_models;
DROP TABLE IF EXISTS llm_providers;

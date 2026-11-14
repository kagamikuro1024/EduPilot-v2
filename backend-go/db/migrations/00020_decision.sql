-- Decision model + cascade ba cổng (SRS FEAT-decision-model mục 5, D60, sprint 10.5). Ngoại lệ có chủ ý cho ALTER: yêu cầu mới sau các migration LLM (tiền lệ 00015).
-- Không lưu đầu vào / đầu ra của decision ở bảng nào. Không thêm DECIDE vào llm_task_routes: provider `decision` chọn bằng hàng provider duy nhất (5.1).

-- +goose Up
-- 5.1 provider và mô hình
ALTER TABLE llm_providers DROP CONSTRAINT llm_providers_type_chk,
  ADD CONSTRAINT llm_providers_type_chk CHECK (type IN ('openai', 'anthropic', 'gemini', 'openai_compatible', 'fake', 'decision'));
ALTER TABLE llm_models DROP CONSTRAINT llm_models_kind_chk,
  ADD CONSTRAINT llm_models_kind_chk CHECK (kind IN ('chat', 'embedding', 'decision'));
-- tối đa MỘT provider type='decision' đang bật
CREATE UNIQUE INDEX llm_providers_one_decision ON llm_providers ((true)) WHERE type = 'decision' AND enabled;

-- 5.2 nhật ký: nhãn cổng + đường đi (không nội dung)
ALTER TABLE llm_audit
  ADD COLUMN cascade_gate text,
  ADD COLUMN cascade_path text,
  ADD CONSTRAINT llm_audit_gate_chk CHECK (cascade_gate IS NULL OR cascade_gate IN ('THREADS_PRECHECK', 'INJECTION_FLAG', 'CHAT_ROUTING')),
  ADD CONSTRAINT llm_audit_path_chk CHECK (cascade_path IS NULL OR cascade_path IN ('FAST', 'FALLBACK', 'FAIL_CLOSED'));
CREATE INDEX llm_audit_decide_idx ON llm_audit (created_at, cascade_gate) WHERE task = 'DECIDE';

-- 5.3 cấu hình cổng (toàn hệ thống); khoảng của threshold kiểm ở service
CREATE TABLE decision_gates (
    gate       text          PRIMARY KEY CHECK (gate IN ('THREADS_PRECHECK', 'INJECTION_FLAG', 'CHAT_ROUTING')),
    enabled    boolean       NOT NULL DEFAULT false,
    threshold  numeric(4,3)  NOT NULL,
    timeout_ms integer       NOT NULL CHECK (timeout_ms BETWEEN 200 AND 5000),
    version    integer       NOT NULL DEFAULT 1,
    updated_by uuid,
    updated_at timestamptz   NOT NULL DEFAULT now()
);
INSERT INTO decision_gates (gate, enabled, threshold, timeout_ms) VALUES
    ('THREADS_PRECHECK', false, 0.050, 800), ('INJECTION_FLAG', false, 0.500, 3000), ('CHAT_ROUTING', false, 0.950, 800);

-- 5.4 bật theo lớp: không có hàng = tắt
CREATE TABLE course_decision_gates (
    course_id  uuid         NOT NULL REFERENCES courses (id),
    gate       text         NOT NULL REFERENCES decision_gates (gate),
    enabled    boolean      NOT NULL DEFAULT false,
    version    integer      NOT NULL DEFAULT 1,
    updated_by uuid,
    updated_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (course_id, gate)
);
CREATE INDEX course_decision_gates_gate_idx ON course_decision_gates (gate, enabled);

-- seed: provider duy nhất 'Quyết định' + hai mô hình; mặc định dùng bản giả (D60 (6)); bản thật chỉ khi LLM_DECISION_ALLOW_REAL và PM cho phép.
-- price_in 2600 đ / 1 triệu token vào ≈ $0,10 [ƯỚC TÍNH]; price_out 0; mô hình giả giá 0.
WITH p AS (
    INSERT INTO llm_providers (type, name, enabled) VALUES ('decision', 'Quyết định', true) RETURNING id
)
INSERT INTO llm_models (provider_id, model, kind, price_in, price_out, enabled)
SELECT p.id, m.model, 'decision', m.price_in, 0, m.enabled
  FROM p, (VALUES ('decision-fake', 0::numeric, true), ('gemini-decision-1', 2600::numeric, false)) AS m (model, price_in, enabled);

-- +goose Down
DROP TABLE IF EXISTS course_decision_gates;
DROP TABLE IF EXISTS decision_gates;
DROP INDEX IF EXISTS llm_audit_decide_idx;
ALTER TABLE llm_audit DROP CONSTRAINT llm_audit_path_chk, DROP CONSTRAINT llm_audit_gate_chk, DROP COLUMN cascade_path, DROP COLUMN cascade_gate;
DROP INDEX IF EXISTS llm_providers_one_decision;
DELETE FROM llm_models WHERE kind = 'decision';
DELETE FROM llm_providers WHERE type = 'decision';
ALTER TABLE llm_models DROP CONSTRAINT llm_models_kind_chk, ADD CONSTRAINT llm_models_kind_chk CHECK (kind IN ('chat', 'embedding'));
ALTER TABLE llm_providers DROP CONSTRAINT llm_providers_type_chk, ADD CONSTRAINT llm_providers_type_chk CHECK (type IN ('openai', 'anthropic', 'gemini', 'openai_compatible', 'fake'));

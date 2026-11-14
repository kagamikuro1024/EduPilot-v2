-- Quan sát AI (SRS FEAT-p10-eval mục 5, P-11-1a): bản prompt ĐÃ CHE của từng lời gọi LLM, chỉ ADMIN mở được (có lý do + audit_log).
-- Không FK tới llm_audit để dọn độc lập (job hằng ngày xoá bản ghi > 30 ngày; llm_audit giữ nguyên).

-- +goose Up
CREATE TABLE llm_request_payloads (
    audit_id      uuid          NOT NULL,                -- = llm_audit.id
    course_id     uuid,                                  -- null cho việc ngoài lớp
    masked_prompt text          NOT NULL,                -- CHỈ bản đã che
    tools         jsonb,                                 -- tên công cụ đã gọi
    confidence    numeric(4,3),
    created_at    timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT llm_request_payloads_pkey PRIMARY KEY (audit_id),
    CONSTRAINT llm_request_payloads_confidence_chk CHECK (confidence IS NULL OR confidence BETWEEN 0 AND 1)
);
CREATE INDEX llm_request_payloads_created_idx ON llm_request_payloads (created_at);

-- Bảng yêu cầu lọc «Có vấn đề» và cảnh báo lỗi 1 giờ.
CREATE INDEX llm_audit_status_created_idx ON llm_audit (status, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS llm_audit_status_created_idx;
DROP TABLE IF EXISTS llm_request_payloads;

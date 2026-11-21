-- Quan sát AI ở quy mô lớn (TC-01-13, P-11-44): `llm_audit` đã tới 600k hàng / 7 ngày thì tổng hợp 7 ngày quét hết (p95 8 s). Bảng cộng gộp THEO GIỜ
-- (giờ đã đóng, worker cập nhật, idempotent) + histogram độ trễ theo nhóm cố định; tổng hợp = giờ đủ từ bảng này + phần lẻ đầu / cuối đọc thô.
-- Không sửa migration đã merge; chỉ thêm bảng / hàm.

-- +goose Up
-- Nhóm độ trễ: 140 nhóm hình học tỷ lệ 1,1 (≈ 5 % mỗi nhóm), nhóm b phủ [1,1^b ; 1,1^(b+1)) ms; ≥ 1,1^139 ≈ 566 s gộp vào nhóm 139. Dùng chung worker và truy vấn đọc.
-- +goose StatementBegin
CREATE FUNCTION obs_lat_bucket(ms integer) RETURNS smallint LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
  SELECT least(139, floor(ln(greatest(ms, 1)::float8) / ln(1.1::float8))::int)::smallint
$$;
-- +goose StatementEnd

CREATE TABLE llm_audit_hourly (
    hour             timestamptz   NOT NULL,
    task             text          NOT NULL,
    requests         bigint        NOT NULL,
    errors           bigint        NOT NULL,
    fallbacks        bigint        NOT NULL,
    cost             numeric(14,4) NOT NULL,
    tokens_in        bigint        NOT NULL,
    tokens_out       bigint        NOT NULL,
    pii_masked       bigint        NOT NULL,
    CONSTRAINT llm_audit_hourly_pkey PRIMARY KEY (hour, task)
);

CREATE TABLE llm_audit_hourly_lat (
    hour   timestamptz NOT NULL,
    bucket smallint    NOT NULL CHECK (bucket BETWEEN 0 AND 139),
    n      bigint      NOT NULL,
    CONSTRAINT llm_audit_hourly_lat_pkey PRIMARY KEY (hour, bucket)
);

-- Một dòng: mọi giờ < watermark đã có trong hai bảng trên (giờ rỗng không có dòng).
CREATE TABLE llm_audit_rollup_state (
    id         boolean     PRIMARY KEY DEFAULT true CHECK (id),
    watermark  timestamptz NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE llm_audit_rollup_state;
DROP TABLE llm_audit_hourly_lat;
DROP TABLE llm_audit_hourly;
DROP FUNCTION obs_lat_bucket(integer);

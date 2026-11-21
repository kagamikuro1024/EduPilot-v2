-- Tệp Forms gốc giữ tối đa 90 ngày (US-P9-11 AC8, Q-P9-7): job xoá tệp ở object storage rồi ghi `blob_purged_at`; lô và các dòng điểm giữ nguyên.
-- +goose Up
ALTER TABLE forms_import_batches ADD COLUMN blob_purged_at timestamptz;
CREATE INDEX forms_import_batches_unpurged_idx ON forms_import_batches (created_at) WHERE blob_purged_at IS NULL;

-- +goose Down
DROP INDEX forms_import_batches_unpurged_idx;
ALTER TABLE forms_import_batches DROP COLUMN blob_purged_at;

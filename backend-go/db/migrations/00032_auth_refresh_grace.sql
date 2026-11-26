-- P-11-140 (phương án A): ân hạn ~10 s sau khi xoay refresh token. Trình duyệt tải lại / huỷ request giữa lúc xoay chỉ còn token cũ (prev) ⇒ trong hạn,
-- máy chủ trả lại ĐÚNG token mới nhất (refresh_enc = AES-GCM, khoá APP_ENCRYPTION_KEY, AAD gắn với phiên) thay vì coi là dùng lại. Hash không tái tạo được token nên cần bản mã.
-- Cột nullable, không DEFAULT ⇒ không viết lại bảng; phiên chưa từng xoay có hai cột NULL.

-- +goose Up
ALTER TABLE auth_sessions
    ADD COLUMN refresh_enc bytea,
    ADD COLUMN grace_until timestamptz,
    ADD CONSTRAINT auth_sessions_grace_pair_chk CHECK ((refresh_enc IS NULL) = (grace_until IS NULL));

-- +goose Down
ALTER TABLE auth_sessions
    DROP CONSTRAINT auth_sessions_grace_pair_chk,
    DROP COLUMN grace_until,
    DROP COLUMN refresh_enc;

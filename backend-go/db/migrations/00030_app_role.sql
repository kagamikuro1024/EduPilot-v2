-- DEF-R6-01 (TC-P3-01-49, US-P3-01 AC9): gateway / worker nối bằng vai ứng dụng `edupilot_app` — NOSUPERUSER, NOBYPASSRLS, không chủ bảng —
-- thay vì vai chủ (superuser của image Postgres). Vai chủ chỉ còn dùng cho `gateway migrate`. Chạy được trên volume cũ (không cần init script).
-- Mật khẩu KHÔNG nằm trong migration: `gateway migrate up` đặt LOGIN + mật khẩu từ APP_DB_PASSWORD sau khi goose xong (db.SetAppRolePassword).
-- Không có CREATE trên schema: vai ứng dụng không tạo / sửa / xoá bảng, không gỡ được trigger append-only (audit_log, pii_events).

-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'edupilot_app') THEN
        CREATE ROLE edupilot_app NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS NOREPLICATION;
    END IF;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    NULL; -- hai migration chạy song song trên cùng cụm (vai là đối tượng cụm): bên kia đã tạo
END
$$;
-- +goose StatementEnd

GRANT USAGE ON SCHEMA public TO edupilot_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO edupilot_app;
GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA public TO edupilot_app;
REVOKE ALL ON goose_db_version FROM edupilot_app;
REVOKE ALL ON SEQUENCE goose_db_version_id_seq FROM edupilot_app;
-- Migration sau này (cùng vai chủ) tự cấp quyền cho vai ứng dụng trên đối tượng mới; bảng goose có sẵn nên không nằm trong phạm vi này.
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO edupilot_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO edupilot_app;

-- +goose Down
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM edupilot_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE USAGE, SELECT, UPDATE ON SEQUENCES FROM edupilot_app;
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM edupilot_app;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM edupilot_app;
REVOKE USAGE ON SCHEMA public FROM edupilot_app;
-- Không DROP ROLE: vai thuộc cả cụm (database khác có thể đang dùng).

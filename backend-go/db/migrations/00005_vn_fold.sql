-- +goose Up
-- Tìm theo tên không phân biệt dấu (FEAT-account-security US-P2-06 AC8): hàm thuần SQL, IMMUTABLE, không cần extension.
-- +goose StatementBegin
CREATE FUNCTION vn_fold(text) RETURNS text
    LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
    AS $$ SELECT translate(lower($1), 'àáạảãâầấậẩẫăằắặẳẵèéẹẻẽêềếệểễìíịỉĩòóọỏõôồốộổỗơờớợởỡùúụủũưừứựửữỳýỵỷỹđ', 'aaaaaaaaaaaaaaaaaeeeeeeeeeeeiiiiiooooooooooooooooouuuuuuuuuuuyyyyyd') $$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION vn_fold(text);

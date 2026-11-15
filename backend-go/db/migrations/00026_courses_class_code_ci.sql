-- DEF-R4-01 (TC-P208-02): mã lớp không phân biệt hoa/thường — `SK20D91` và `sk20d91` là CÙNG một lớp (SV gõ mã dễ nhầm).
-- Giữ nguyên tên chỉ mục `courses_class_code_key` để mã Go (isUnique) không đổi. Không sửa migration đã merge.
-- Dữ liệu cũ có hai mã chỉ khác hoa/thường: lớp tạo sau giữ nguyên mã và thêm hậu tố `-2`, `-3`… (không xoá, không gộp lớp); thông báo ghi ở NOTICE.

-- +goose Up
-- +goose StatementBegin
DO $$
DECLARE r record;
BEGIN
  FOR r IN
    SELECT id, class_code, row_number() OVER (PARTITION BY lower(class_code) ORDER BY created_at, id) AS n
      FROM courses
  LOOP
    IF r.n > 1 THEN
      UPDATE courses SET class_code = left(r.class_code, 20 - length('-' || r.n)) || '-' || r.n WHERE id = r.id;
      RAISE NOTICE 'courses %: class_code % -> trùng không phân biệt hoa/thường, đổi hậu tố', r.id, r.class_code;
    END IF;
  END LOOP;
END $$;
-- +goose StatementEnd
DROP INDEX courses_class_code_key;
CREATE UNIQUE INDEX courses_class_code_key ON courses (lower(class_code));

-- +goose Down
DROP INDEX courses_class_code_key;
CREATE UNIQUE INDEX courses_class_code_key ON courses (class_code);

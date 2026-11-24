-- PM 24/11: kỳ thi «QC E7 khoá chat» (01a1e01f-4e89-7a7f-b90b-2a8a92e5cb87, lớp E7C1) là fixture SQL của smoke 7 với 0 mục → exam.tick lỗi mỗi 5 s.
-- Thêm đúng 1 câu hỏi ESSAY + 1 exam_item hợp lệ; KHÔNG xoá gì.
begin;
with c as (select course_id, created_by from exams where id = '01a1e01f-4e89-7a7f-b90b-2a8a92e5cb87'),
q as (
  insert into question_bank (course_id, type, title, topic, stem, origin, review_status, created_by, reviewed_by, reviewed_at)
  select course_id, 'ESSAY', 'QC fixture: câu duy nhất cho kỳ thi khoá chat', 'qc-fixture', 'Nêu một ví dụ về tấn công lừa đảo qua email.', 'MANUAL', 'APPROVED', created_by, created_by, now() from c
  returning id, course_id)
insert into exam_items (course_id, exam_id, question_id, position, points)
select q.course_id, '01a1e01f-4e89-7a7f-b90b-2a8a92e5cb87', q.id, 1, 1 from q
returning id, question_id;
commit;

-- Bước 2 (cùng ngày): ESSAY không có answer_key làm tick lỗi «đáp án đúng không đọc được: unexpected end of JSON input» → đổi chính câu vừa thêm sang TRUE_FALSE có đáp án.
update question_bank set type = 'TRUE_FALSE', answer_key = '{"value": true}', stem = 'Email giả danh ngân hàng để xin mật khẩu là tấn công phishing. Đúng hay sai?'
 where title = 'QC fixture: câu duy nhất cho kỳ thi khoá chat' and course_id = (select course_id from exams where id = '01a1e01f-4e89-7a7f-b90b-2a8a92e5cb87');

#!/bin/bash
# đếm số liệu seed (DB qc3seed): dùng cho SEED-16/22
docker exec ep-r-postgres-1 psql -U edupilot -d qc3seed -At -F ' ' -c "
select 'attendance_records', count(*) from attendance_records
union all select 'attendance_distinct', count(distinct (session_id, student_id)) from attendance_records
union all select 'attendance_absent_s1_9', count(*) from attendance_records where status='ABSENT'
union all select 'sessions_done', count(*) from class_sessions where attendance_done_at is not null
union all select 'participation_events', count(*) from participation_events
union all select 'tickets', count(*) from escalation_tickets
union all select 'post_reports', count(*) from post_reports
union all select 'knowledge_items', count(*) from knowledge_items
union all select 'forum_threads', count(*) from forum_threads
union all select 'users', count(*) from users
union all select 'forum_posts', count(*) from forum_posts
union all select 'submissions', count(*) from submissions
union all select 'enrollments', count(*) from enrollments
union all select 'courses', count(*) from courses
union all select 'chat_messages', count(*) from chat_messages
union all select 'documents', count(*) from documents
union all select 'study_time_daily', count(*) from study_time_daily
union all select 'student_observations', count(*) from student_observations
union all select 'notifications', count(*) from notifications
union all select 'mail_outbox', count(*) from mail_outbox
union all select 'calendar_events', count(*) from calendar_events
union all select 'exam_attempts', count(*) from exam_attempts
union all select 'question_bank', count(*) from question_bank"

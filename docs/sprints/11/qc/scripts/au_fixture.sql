-- QC US-P10-14: fixture F1…F7 (INSERT-only, entity qc_audit / qc_bulk). Chạy: docker exec -i ep-qc-postgres-1 psql -U edupilot -d edupilot -f - < au_fixture.sql
do $$
declare adm uuid := (select id from users where email='admin@edupilot.local');
begin
  if exists (select 1 from audit_log where entity='qc_audit') then return; end if;
  -- F1: 230 dòng cách nhau 1 phút
  insert into audit_log(actor_id, entity, entity_id, action, after, trace_id, created_at)
    select adm, 'qc_audit', 'f1-'||i, 'QC_F1', jsonb_build_object('k', i), 'qc-f1-'||i, now() - (i||' minutes')::interval from generate_series(1,230) i;
  -- F2: 60 dòng cùng created_at
  insert into audit_log(actor_id, entity, entity_id, action, after, trace_id, created_at)
    select adm, 'qc_audit', 'f2-'||i, 'QC_F2', jsonb_build_object('k', i), 'qc-f2-'||i, timestamptz '2026-11-10 03:00:00+00' from generate_series(1,60) i;
  -- F3: canary
  insert into audit_log(actor_id, entity, entity_id, action, before, after, trace_id, created_at) values
    (adm, 'qc_audit', 'f3', 'QC_F3', '{"cfg":"SECRET-CANARY-1","email":"canary1@sv.example","student_code":"20229001"}', '{"cfg":"SECRET-CANARY-2","reason":"SECRET-CANARY-3 lý do dài"}', 'qc-f3', now() - interval '1 minute');
  -- F4: hệ thống + người đã xoá
  insert into audit_log(actor_id, entity, entity_id, action, after, trace_id, created_at) values
    (null, 'qc_audit', 'f4-sys', 'QC_F4', '{"a":1}', 'qc-f4a', now() - interval '2 minutes'),
    ('00000000-0000-7000-8000-000000000bad', 'qc_audit', 'f4-del', 'QC_F4', '{"a":1}', 'qc-f4b', now() - interval '3 minutes');
  -- F5
  insert into audit_log(actor_id, entity, entity_id, action, before, after, trace_id, created_at)
    select adm, 'qc_audit', 'f5-25', 'QC_F5', jsonb_build_object('zz_before_only', 1), (select jsonb_object_agg('k'||lpad(i::text,2,'0'), i) from generate_series(1,25) i), 'qc-f5a', now() - interval '4 minutes';
  insert into audit_log(actor_id, entity, entity_id, action, before, after, trace_id, created_at) values
    (adm, 'qc_audit', 'f5-null', 'QC_F5', null, null, 'qc-f5b', now() - interval '5 minutes'),
    (adm, 'qc_audit', 'f5-arr', 'QC_F5', null, '[1,2]', 'qc-f5c', now() - interval '6 minutes'),
    (adm, 'qc_audit', 'f5-str', 'QC_F5', null, '"x"', 'qc-f5d', now() - interval '7 minutes');
  -- F6: mốc
  insert into audit_log(actor_id, entity, entity_id, action, after, trace_id, created_at) values
    (adm, 'qc_audit', 'f6-from', 'QC_F6', '{"a":1}', 'qc-f6a', timestamptz '2026-11-12 03:00:00+00'),
    (adm, 'qc_audit', 'f6-from1', 'QC_F6', '{"a":1}', 'qc-f6b', timestamptz '2026-11-12 03:01:00+00'),
    (adm, 'qc_audit', 'f6-to', 'QC_F6', '{"a":1}', 'qc-f6c', timestamptz '2026-11-12 04:00:00+00'),
    (adm, 'qc_audit', 'f6-6d23h', 'QC_F6', '{"a":1}', 'qc-f6d', now() - interval '6 days 23 hours'),
    (adm, 'qc_audit', 'f6-7d1h', 'QC_F6', '{"a":1}', 'qc-f6e', now() - interval '7 days 1 hour'),
    (adm, 'qc_audit', 'f6-89d', 'QC_F6', '{"a":1}', 'qc-f6f', now() - interval '89 days');
  -- F7: 100.000 dòng, 60 ngày, 20 entity, 50 actor
  insert into audit_log(actor_id, entity, entity_id, action, after, trace_id, created_at)
    select (select id from users order by id offset (i % 50) limit 1), case when i % 20 = 0 then 'qc_bulk_rare' when i = 7 then 'qc_bulk_one' else 'qc_bulk' || (i % 20) end,
           'b'||i, 'QC_BULK', jsonb_build_object('n', i), 'qc-bulk', now() - ((i % 86400) * 60 || ' seconds')::interval
    from generate_series(1,100000) i;
end $$;
analyze audit_log;

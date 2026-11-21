import sys, os, re; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
W = "/Users/kuro/.config/edupilot-team/qc3-work/"
def prompts(a):
    open(W + "stub_req.jsonl", "w").close(); stub("allow"); grade_all_wait(a)
    L = [json.loads(l) for l in open(W + "stub_req.jsonl") if l.strip()]
    return [json.dumps(r["body"].get("messages"), ensure_ascii=False, sort_keys=True) for r in L if r["body"].get("messages")]
def norm(p): return re.sub(r"[0-9A-F]{4,}", "<H>", re.sub(r"g1107[ab]", "<T>", re.sub(r"[0-9a-f]{8}-[0-9a-f-]{27}", "<id>", p)))
def run():
    A = mk_assignment("g1107a", 1); B = mk_assignment("g1107b", 1)
    sql(f"update assignments set guidance_version=2 where id='{B}'")
    sql(f"insert into grading_guidance_items (id, course_id, assignment_id, criterion_id, text, direction, support, source_result_ids, status, approved_in_version, version, decided_by, decided_at) select uuidv7(), course_id, id, 'c2', 'QC LƯU Ý KIỂM TRA không được xuất hiện khi tắt cờ.', 'STRICTER', 3, '{{}}', 'APPROVED', 1, 1, created_by, now() from assignments where id='{B}'")
    flag("GRADING_GUIDANCE", False); pa = prompts(A); pb = prompts(B)
    sql(f"set session_replication_role=replica; delete from grading_results where submission_id in (select id from submissions where assignment_id='{B}')")
    flag("GRADING_GUIDANCE", True); pb_on = prompts(B)
    na = [norm(x) for x in pa]; nb = [norm(x) for x in pb]; nc = [norm(x) for x in pb_on]
    same = len(na) > 0 and sorted(na) == sorted(nb); has_off = any("QC LƯU Ý KIỂM TRA" in x for x in pb); has_on = any("QC LƯU Ý KIỂM TRA" in x for x in pb_on)
    kept = sql(f"select count(*) from grading_guidance_items where assignment_id='{B}'")[0][0]
    print("len", len(pa), len(pb), len(pb_on), "same", same, "off-has", has_off, "on-has", has_on, "kept", kept)
    chk("TC-G11-07", same and not has_off and has_on and int(kept) == 1, f"cờ GRADING_GUIDANCE tắt: chấm bài tập B (guidance_version=2, 1 lưu ý APPROVED trong DB) và bài tập A (v0, cùng rubric/bài làm) qua stub ghi prompt: {len(pa)}/{len(pb)} lời gọi, prompt hai bên GIỐNG nhau từng byte (sau thay id, tên bài tập, mã nộp; xếp theo nội dung vì hai lượt chấm song song) = {same}; chữ lưu ý trong prompt khi tắt = {has_off}; bật lại cờ → prompt B chứa lưu ý = {has_on}; lưu ý còn trong DB = {kept}. Phần «GUIDANCE_ENABLED=false» và so với nhánh main chưa đo (TestEssayPromptUnchangedAtV0 PASS)")
with window(): run()

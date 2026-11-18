import sys; sys.path.insert(0, '.')
from s8r_chat import *
CAN = 'Lớp chưa có công thức điểm chính thức do giảng viên xác nhận.'
C = cid('QD8H9'); tick = lambda c, o='AUTO_NO_FORMULA': int(L.psql(f"select count(*) from escalation_tickets where course_id='{c}' and origin='{o}'"))
t0 = now_db(); sid, a_txt, ev = ask('sv.kha', C, 'Cô tính điểm môn này thế nào?'); a_llm = audit_since(t0)
tkA = tick(C); lowA = tick(C, 'AUTO_LOW_CONFIDENCE')
T('TC-P6-05-37', 'F' if CAN not in a_txt else 'P', f'câu đúng nguyên văn TC «Cô tính điểm môn này thế nào?» (QD8H9 chưa công thức): trả «{a_txt[:70]}» (kỳ vọng câu mẫu «{CAN[:40]}…»); llm_audit +{a_llm}; ticket AUTO_NO_FORMULA {tkA}, AUTO_LOW_CONFIDENCE {lowA}. reFormula (intent.go:24) không khớp «tính điểm môn này thế nào» → rơi COURSE_QA → DEF-R8-A4')
bb = tick(C); t0 = now_db(); sid2, b_txt, ev = ask('sv.kha', C, 'Công thức tính điểm học phần là gì?'); b_llm = audit_since(t0); b1 = tick(C)
_, b2_txt, _ = ask('sv.kha', C, 'Công thức tính điểm học phần là gì?', sid=sid2); b2 = tick(C)
num = any(ch.isdigit() for ch in b_txt)
T('TC-P6-05-38', 'P' if CAN in b_txt and b_llm == '0' and b1 == bb + 1 and not num else 'F', f'câu «Công thức tính điểm học phần là gì?» (GRADE_FORMULA): câu mẫu {CAN in b_txt}, llm_audit +{b_llm}, không số {not num}; ticket AUTO_NO_FORMULA {bb}→{b1} (+1); hỏi lại CÙNG câu trong cùng hội thoại → {b2} (thực tế: mỗi tin một ticket, không khử trùng theo câu; SRS Q12 chỉ đặt trần 5 ticket/…, TC nói «QC ghi nhận thực tế»). ResponseMetadata.NoFormula không lộ trong SSE done (kiểm bằng Go test TestFormulaNoScheme…)')
t0 = now_db(); _, p_txt, _ = ask('sv.kha', C, 'Điểm của em bao nhiêu?'); _, w_txt, _ = ask('sv.kha', C, 'Nếu cuối kỳ em được 8 thì sao?'); p_llm = audit_since(t0); b3 = tick(C)
T('TC-P6-05-39', 'P' if CAN in p_txt and CAN in w_txt and b3 == b2 and not any(ch.isdigit() for ch in p_txt + w_txt) else 'F', f'PERSONAL_GRADE «{p_txt[:60]}», WHAT_IF «{w_txt[:60]}»: cùng câu mẫu {CAN in p_txt and CAN in w_txt}; ticket AUTO_NO_FORMULA vẫn {b3} = số trước đó {b2} (không mở thêm); không số điểm; llm_audit +{p_llm}')

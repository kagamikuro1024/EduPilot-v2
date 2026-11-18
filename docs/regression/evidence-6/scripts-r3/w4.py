from w3lib import *
import fx
# 02-34: A (C1: có dữ liệu điểm danh → sinh chữ) + B (QA: tên trong ngữ cảnh khôi phục)
r=chat('sv.gioi',f'Em là {GIANG}, MSSV 20229001, cho em xem điểm danh'); p=r['payload']; sysm=nogen(p)[0]['messages'][0] if nogen(p) else ''
r2=ask('sv.gioi',f'Em là {GIANG}, MSSV 20229001, giải thích giúp em về chữ ký số',title=f'Bài ghi chú của {GIANG}')
ok=bool(nogen(p)) and GIANG not in flat(p) and '20229001' not in flat(p) and 'bạn' in sysm and GIANG in r2['text'] and GIANG not in flat(r2['payload'])
rec('TC-P3-02-34',ok,f'(a) lớp 761987 "xem điểm danh": intent={r["row"].split("|")[0]}, {len(nogen(p))} lời gọi sinh chữ, payload 0 tên/MSSV, system prompt có "bạn"={"bạn" in sysm}, notice={r["notice"]}; (b) tên trong tiêu đề tài liệu ngữ cảnh → payload "[[SV_1]]", câu trả lời hiện tên thật: {r2["text"][36:90]!r}')
# 03-18 trong lớp sạch (ngữ cảnh không có tên)
fx.use('QC3S6C','QKTWQ3C')
t18=f'Em là {GIANG}, bạn {KHAI} cùng nhóm, MSSV 20229001, email sv.gioi@edupilot.local, SĐT 0912345678, CCCD 001099012345, giải thích về hàm băm'
r=ask('sv.gioi',t18); nt=r['notice'][0].get('masked') if r['notice'] else None
pm=psql(f"select masked_count from chat_messages where session_id='{r['sid']}' and role='ASSISTANT' order by created_at desc limit 1")
au=psql(f"select pii_masked_count||'|'||status from llm_audit where user_id=(select id from users where email='sv.gioi@edupilot.local') and task='CHAT' order by created_at desc limit 1")
rec('TC-P3-03-18',nt==6 and pm=='6' and au.startswith('6|'),f'lớp QC3S6C (ngữ cảnh không có tên), tin có 2 tên + MSSV + email + SĐT + CCCD: notice.masked={nt}, chat_messages.masked_count={pm}, llm_audit.pii_masked_count|status={au}')

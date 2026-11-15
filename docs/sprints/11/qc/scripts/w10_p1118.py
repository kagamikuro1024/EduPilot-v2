# Đợt R · P-11-18: đo 2 chiều qua 9router thật. OOD: ≥10 câu cùng chủ đề (an ninh mạng) NGOÀI tài liệu → tỷ lệ ticket. IN: ≥10 câu TRONG tài liệu → tỷ lệ có [n], ticket giả.
# Cần embedding THẬT (PM cho phép lại, trần 2.000, nạp bằng w9a_embed_on.py / gỡ bằng w9d). Mỗi SV tối đa 5 ticket chờ → xoay vòng SV.
from rlib import *
import re, json, time
exec(open("w2_chat.py").read().split("rows = sql(")[0].split("from rlib import *")[1])  # parse(), ask()
c1 = C("teacher")
OOD = ["Ransomware WannaCry lây lan qua lỗ hổng nào của Windows?", "Zero-trust khác gì mô hình bảo mật vành đai?", "Giao thức WPA3 cải tiến gì so với WPA2?",
       "SQL injection mù (blind) khai thác thế nào?", "Kerberos golden ticket là gì?", "Mô hình MITRE ATT&CK có bao nhiêu chiến thuật?",
       "Tấn công cross-site scripting lưu trữ (stored XSS) phòng thế nào?", "SIEM khác SOAR ở điểm nào?", "Honeypot dùng để làm gì trong hệ thống giám sát?",
       "VPN IPsec chế độ tunnel và transport khác nhau ra sao?", "Tấn công side-channel Spectre hoạt động thế nào?", "Chuẩn ISO 27001 yêu cầu những gì về đánh giá rủi ro?"]
IN = ["Ping of death là kiểu tấn công gì?", "Code Red gây ra hậu quả gì cho máy chủ?", "Kiến trúc bảo mật gồm những thành phần chính nào (scanning engines…)?",
      "Có bốn nhóm tấn công mạng nào theo tài liệu?", "Tấn công từ chối dịch vụ DoS là gì?", "Sâu (worm) khác virus ở điểm nào theo bài giảng?",
      "Reconnaissance attack là gì?", "Access attack gồm những dạng nào?", "Trojan Horse hoạt động ra sao?", "Tấn công smurf là gì?",
      "Vì sao các chuyên gia mạng vá lỗi chậm làm Code Red nặng hơn?", "Ví dụ về tấn công truy cập (access attacks) trong slide là gì?"]
SVS = [u for u in ("sv.kha", "sv.gioi", "sv.nguyco")]
extra = [r[0] for r in sql("select u.email from users u join enrollments e on e.user_id=u.id where e.course_id='%s' and u.email like 'sv%%@edupilot.local' and u.email not in ('sv.kha@edupilot.local','sv.gioi@edupilot.local','sv.nguyco@edupilot.local','sv.moi@edupilot.local') limit 6" % c1)]
print("SV bổ sung:", extra)
def run(qs, tag):
    out = []
    for i, t in enumerate(qs):
        u = SVS[i % len(SVS)]
        c, s, _ = api(u, "POST", "/chat/sessions", {"course_id": c1})
        if c != 201 and c != 200: out.append((t, c, None, None, None)); continue
        st, t1, tot, ans, done, ev = ask(u, s["id"], t)
        out.append((t, st, ans, done, u))
        print(f"  [{tag}] {t[:50]:50} st={st} cit={len(done.get('citations') or [])} [n]={'yes' if re.search(r'\\[\\d+\\]', ans or '') else 'no'} low={done.get('low_confidence')} ticket={'có' if done.get('ticket') else 'không'} → {(ans or '')[:60]!r}", flush=True)
    return out
o = run(OOD, "OOD"); n = len(o); tk = sum(1 for x in o if x[3] and x[3].get("ticket")); lc = sum(1 for x in o if x[3] and x[3].get("low_confidence"))
ok("R-P1118-OOD tỷ lệ ticket ≥ 90% trên câu ngoài tài liệu", n >= 10 and tk / n >= 0.9, f"{tk}/{n} ticket; low_confidence {lc}/{n}")
i_ = run(IN, "IN"); m = len(i_); cit = sum(1 for x in i_ if x[2] and re.search(r"\[\d+\]", x[2])); tk2 = sum(1 for x in i_ if x[3] and x[3].get("ticket"))
ok("R-P1118-IN tỷ lệ có [n] trên câu trong tài liệu; ticket giả = 0", m >= 10 and tk2 == 0, f"[n] {cit}/{m}; ticket giả {tk2}/{m}")

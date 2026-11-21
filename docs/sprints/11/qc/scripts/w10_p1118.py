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
SV_OOD = ["sv12", "sv13", "sv14", "sv15"]; SV_IN = ["sv17", "sv18", "sv19", "sv20"]  # mỗi SV tối đa 5 ticket chờ
TPL = "chưa tìm thấy nội dung này trong tài liệu của lớp"
RAW = re.compile(r"\[\d+(?:\s*[,;–-]\s*\d+)+\]")  # chữ thô kiểu [2, 5, 7]
def run(qs, tag, SVS):
    out = []
    for i, t in enumerate(qs):
        u = SVS[i % len(SVS)]
        c, s, _ = api(u, "POST", "/chat/sessions", {"course_id": c1})
        if c != 201 and c != 200: out.append((t, c, None, None, None)); continue
        st, t1, tot, ans, done, ev = ask(u, s["id"], t)
        out.append((t, st, ans, done, u))
        hn = "yes" if re.search(r"\[\d+\]", ans or "") else "no"
        print(f"  [{tag}] {t[:50]:50} st={st} cit={len(done.get('citations') or [])} [n]={hn} low={done.get('low_confidence')} ticket={'có' if done.get('ticket') else 'không'} tpl={'yes' if TPL in (ans or '').lower() else 'no'} raw={'CÓ' if RAW.search(ans or '') else 'không'} deg={'CÓ' if ('Trả lời tạm thời' in (ans or '') or 'gián đoạn' in (ans or '')) else 'không'} → {(ans or '')[:60]!r}", flush=True)
    return out
o = run(OOD, "OOD", SV_OOD); n = len(o); tk = sum(1 for x in o if x[3] and x[3].get("ticket")); lc = sum(1 for x in o if x[3] and x[3].get("low_confidence"))
tp = sum(1 for x in o if x[2] and TPL in x[2].lower()); ci = sum(1 for x in o if x[3] and (x[3].get("citations") or []))
ok("R-P1118-OOD tỷ lệ ticket ≥ 90% trên câu ngoài tài liệu", n >= 10 and tk / n >= 0.9, f"ticket {tk}/{n}; low_confidence {lc}/{n}; LLM theo câu mẫu {tp}/{n}; có trích dẫn {ci}/{n}")
i_ = run(IN, "IN", SV_IN); m = len(i_); cit = sum(1 for x in i_ if x[2] and re.search(r"\[\d+\]", x[2])); tk2 = sum(1 for x in i_ if x[3] and x[3].get("ticket"))
tp2 = sum(1 for x in i_ if x[2] and TPL in x[2].lower())
ok("R-P1118-IN ticket giả ≤ 20% trên câu trong tài liệu", m >= 10 and tk2 / m <= 0.2, f"[n] {cit}/{m}; ticket giả {tk2}/{m}; câu mẫu (từ chối nhầm) {tp2}/{m}")

raw = [x[0] for x in o + i_ if x[2] and RAW.search(x[2])]
ok("R-P1118-RAW không còn chữ thô [2, 5, 7] trong câu trả lời SV", not raw, f"{len(raw)} câu còn: {raw[:3]}")
deg = sum(1 for x in o + i_ if x[2] and ("Trả lời tạm thời" in x[2] or "gián đoạn" in x[2]))
print("câu bị giảm cấp (9router rớt), cần loại khỏi tỷ lệ:", deg)

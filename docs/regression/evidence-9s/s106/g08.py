import sys, os, math, random; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
W = "/Users/kuro/.config/edupilot-team/qc3-work/"
SIM = f"/courses/{CG0}/threads/similar-confirmed"
T = json.load(open(W + "g08_threads.json"))["tids"]
def chunk_of(t): return sql(f"select c.id, regexp_replace(left(c.text,1500), '[[:space:]|]+', ' ', 'g') from knowledge_items k join forum_posts p on p.id=k.source_id join content_chunks c on c.id=k.chunk_id where p.thread_id='{t}'")[0]
CH = [chunk_of(t) for t in T]; q = CH[0][1]
emb0 = [float(x) for x in sql(f"select embedding::text from content_chunks where id='{CH[0][0]}'")[0][0].strip("[]").split(",")]
n0 = math.sqrt(sum(x * x for x in emb0)); u = [x / n0 for x in emb0]
random.seed(7); r = [random.gauss(0, 1) for _ in u]; d = sum(a * b for a, b in zip(r, u)); w = [a - d * b for a, b in zip(r, u)]; nw = math.sqrt(sum(x * x for x in w)); w = [x / nw for x in w]
def vec(c): return [c * a + math.sqrt(1 - c * c) * b for a, b in zip(u, w)]
def setv(cid, c): sql(f"update content_chunks set embedding='[{','.join(f'{x:.7f}' for x in vec(c))}]' where id='{cid}'")
def cos(cid): return float(sql(f"select 1 - (embedding <=> (select embedding from content_chunks where id='{CH[0][0]}')) from content_chunks where id='{cid}'")[0][0])
def sim(user="sv.kha"): return api(user, "POST", SIM, {"q": q})
def ids(j): return [x["thread_id"] for x in j]
flag("THREAD_SUGGEST", True)
# G08-03: 5 thread, cos 1 / 1 / 0.95 / 0.90 / 0.85
for i, c in enumerate([1, 1, 0.95, 0.90, 0.85]): setv(CH[i][0], c) if i else None
print("cos thực", [round(cos(CH[i][0]), 3) for i in range(5)])
c, j, ms = sim(); got = ids(j); print(c, [T.index(x) for x in got if x in T])
chk("TC-G08-03", c == 200 and len(got) == 3 and set(got[:2]) == {T[0], T[1]} and got[2] == T[2] and T[3] not in got and T[4] not in got, f"5 thread có tri thức xác nhận, cosine với câu soạn = {[round(cos(CH[i][0]), 3) for i in range(5)]} (≥ 0,80): POST similar-confirmed → {c}, {len(got)} kết quả (LIMIT 3), thứ tự theo cosine giảm: chỉ số thread {[T.index(x) for x in got if x in T]} (kỳ vọng hai thread cos 1, rồi thread cos 0,95; thread 0,90 và 0,85 bị cắt)")
# G08-04: 0.81 có, 0.79 không
for i, c_ in [(1, 0.81), (2, 0.79), (3, 0.30), (4, 0.30)]: setv(CH[i][0], c_)
cs = [round(cos(CH[i][0]), 3) for i in range(5)]; c, j, ms = sim(); got = ids(j)
chk("TC-G08-04", c == 200 and T[1] in got and T[2] not in got, f"cosine thật (pgvector) của 3 thread với câu soạn: {cs[:3]}; kết quả {[T.index(x) for x in got if x in T]}: thread cos {cs[1]} có = {T[1] in got}, thread cos {cs[2]} không = {T[2] not in got}")
# G08-05: UNVERIFIED / REJECTED / hidden / bị loại → không xuất hiện (đặt cos 1 cho cả 3 + thread gốc)
for i in (1, 2, 3): setv(CH[i][0], 1.0)
p = {i: sql(f"select id from forum_posts where thread_id='{T[i]}' and kind='AI'")[0][0] for i in (1, 2, 3)}
before = ids(sim()[1])
api("teacher", "POST", f"/courses/{CG0}/posts/{p[1]}/reject", {})          # đáp án REJECTED → gỡ tri thức
c1 = api("teacher", "POST", f"/courses/{CG0}/threads/{T[2]}/hide", {"reason": "QC106 thử ẩn", "version": int(sql(f"select version from forum_threads where id='{T[2]}'")[0][0])}); 
after = ids(sim()[1]); 
chk("TC-G08-05", T[1] in before and T[2] in before and T[1] not in after and T[2] not in after, f"trước: thread 1,2,3 đều cos 1 → có trong gợi ý {[T.index(x) for x in before if x in T]}; sau khi đáp án thread 1 bị Từ chối (reject) và thread 2 bị Ẩn (hide {c1[0]}): gợi ý {[T.index(x) for x in after if x in T]} (thread 1 và 2 biến mất)")

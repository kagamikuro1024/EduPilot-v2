# QC tự cài winnowing (k=5, w=4, FNV-1a 64, Jaccard) — không dùng mã của dev
import re, sys, itertools, glob, os
KW = set('auto break case char class const continue default do double else enum extern float for goto if inline int long namespace new operator private public return short signed sizeof static struct switch template this typedef union unsigned using virtual void volatile while bool true false nullptr delete try catch throw constexpr'.split())
TOK = re.compile(r'"(?:\\.|[^"\\])*"|\'(?:\\.|[^\'\\])*\'|[A-Za-z_]\w*|\d[\w.]*|<<=|>>=|->|\+\+|--|&&|\|\||<=|>=|==|!=|<<|>>|[-+*/%&|^<>=!~?:;,.(){}\[\]]')
def tokens(src):
    src = re.sub(r'/\*.*?\*/', ' ', src, flags=re.S); src = re.sub(r'//[^\n]*', ' ', src); src = re.sub(r'^\s*#\s*include[^\n]*', ' ', src, flags=re.M)
    out = []
    for t in TOK.findall(src):
        if t[0] in '"\'': out.append('S')
        elif t[0].isdigit(): out.append('N')
        elif t[0].isalpha() or t[0] == '_': out.append(t if t in KW else 'I')
        else: out.append(t)
    return out
def h(b):
    x = 0xcbf29ce484222325
    for c in b: x = ((x ^ c) * 0x100000001b3) & 0xFFFFFFFFFFFFFFFF
    return x
def fp(toks, k=5, w=4):
    if len(toks) < 30: return None
    hs = [h(' '.join(toks[i:i + k]).encode()) for i in range(len(toks) - k + 1)]; out = set()
    for i in range(max(1, len(hs) - w + 1)): out.add(min(hs[i:i + w]))
    return out
def jac(a, b): return len(a & b) / len(a | b) if a | b else 0
if __name__ == '__main__':
    d = os.path.join(os.path.dirname(__file__), 'p507-src'); F = {os.path.basename(p).split('.')[0]: fp(tokens(open(p).read())) for p in glob.glob(d + '/*.cpp')}
    for n, v in F.items(): print(n, None if v is None else len(v))
    for a, b in [('A-original', 'B-renamed-reformatted'), ('A-original', 'C-different-algorithm'), ('A-original', 'Aprime-reordered')]: print(a, '~', b, round(jac(F[a], F[b]), 3), '| đối xứng', jac(F[a], F[b]) == jac(F[b], F[a]))

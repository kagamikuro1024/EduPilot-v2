import json, re, sys, math
def rgb(s):
    m = re.findall(r'[\d.]+', s); return [float(x) for x in m[:3]]
def lin(c): c /= 255; return c / 12.92 if c <= .04045 else ((c + .055) / 1.055) ** 2.4
def lum(c): r, g, b = [lin(x) for x in c]; return .2126 * r + .7152 * g + .0722 * b
def cr(a, b): la, lb = sorted([lum(a), lum(b)], reverse=True); return (la + .05) / (lb + .05)
def oklab_L(c):
    r, g, b = [lin(x) for x in c]
    l = .4122214708 * r + .5363325363 * g + .0514459929 * b; m = .2119034982 * r + .6806995451 * g + .1073969566 * b; s = .0883024619 * r + .2817188376 * g + .6299787005 * b
    l, m, s = [x ** (1 / 3) for x in (l, m, s)]; return (.2104542553 * l + .793617785 * m - .0040720468 * s) * 100
def table(tokens):
    out = {}; inks = ['--ep-ink', '--ep-ink-2', '--ep-ink-3', '--ep-red', '--ep-green', '--ep-blue']; bgs = ['--ep-canvas', '--ep-surface', '--ep-surface-strong', '--ep-surface-subtle', '--ep-red-soft']
    for bg in bgs:
        for ik in inks: out[(ik, bg)] = round(cr(rgb(tokens[ik]), rgb(tokens[bg])), 2)
    return out
if __name__ == '__main__':
    d = json.load(open(sys.argv[1])); seen = {}
    for r in d:
        key = r['url'].split('?')[-1] if 'surface' in r['url'] else 'main'
        if key in seen or r['w'] != 1440: continue
        seen[key] = r['tokens']
    for k, t in seen.items():
        tb = table(t); low = min(tb.items(), key=lambda x: x[1])
        print(k, 'ΔL canvas↔surface %.2f' % (oklab_L(rgb(t['--ep-surface'])) - oklab_L(rgb(t['--ep-canvas']))), '| thấp nhất', low[0], low[1], '| dưới 4,5:', [(a, b, v) for (a, b), v in tb.items() if v < 4.5])

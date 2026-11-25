import m, scn, scn2, time, json, sys
def case(name, fn):
    try:
        r=fn()
    except Exception as e:
        r=("ERR", str(e)[:80])
    print(name, "->", r, flush=True)
    return r

def prep(user='sv.qc2'):
    scn2.fresh_login(user)
    return scn2.entry()

def do_scan_and(w, url, tag, secs=15):
    ok=scn2.scan()
    if not ok: return ("nocamera",)
    scn2.poster_url(url, tag)
    res,t,dt=scn2.wait_result(w, secs)
    return (res, round(dt,1), len(scn.rows(w)), t[-60:])

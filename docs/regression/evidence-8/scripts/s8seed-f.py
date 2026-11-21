exec(open('s8seed.py').read())
tt = tok('teacher'); Q = f'/courses/{C2}'
def status(): return req('GET', f'{Q}/grade-scheme/status', tt)[1]
s, g = req('GET', f'{Q}/grade-scheme', tt); dr = g['draft']
comps = [{'key': c['key'], 'name': c['name'], 'grp': c['grp'], 'weight': c['weight'], 'items': [{k: it.get(k) for k in ('key', 'name', 'max_score', 'source')} for it in c['items']]} for c in dr['components']]
rules = [{'kind': r['kind'], 'applicable': r.get('applicable', True), 'params': r['params']} for r in dr.get('rules', []) if r['kind'] != 'ROUNDING'] + [{'kind': 'ROUNDING', 'applicable': True, 'params': {'mode': 'HALF_UP', 'step': '0.1'}}]
blk0 = status().get('finalize_blocked_reason'); fin0 = req('POST', f'{Q}/gradebook/finalize', tt, {})
sp, dp = req('PUT', f'{Q}/grade-scheme', tt, {'version': dr['version'], 'components': comps, 'rules': rules}); st1 = status()
s, g = req('GET', f'{Q}/grade-scheme', tt); v = g['draft']['version']
sc, dc = req('POST', f'{Q}/grade-scheme/confirm', tt, {'version': v}); st2 = status()
fin = req('POST', f'{Q}/gradebook/finalize', tt, {}) if False else (None, None)
print(sp, st1, sc, st2)
C('TC-P6-SEED-15', st1.get('state') == 'DRAFT_READY' and sc == 200 and st2.get('state') == 'CONFIRMED' and 'finalize_blocked_reason' not in st2, f'lớp 2: trước khi điền, Chốt điểm bị khoá («{blk0}», finalize → {fin0[0]} {fin0[1].get("code")}); điền «Làm tròn đến 0,1» (HALF_UP, bước 0.1) PUT → {sp}, state {st1.get("state")} unclear {st1.get("unclear_count")}; xác nhận (confirm version {v}) → {sc}; status.state {st2.get("state")}, finalize_blocked_reason {"không còn" if "finalize_blocked_reason" not in st2 else st2["finalize_blocked_reason"]} (Chốt điểm mở). Giao diện hộp thoại xác nhận/banner: S8-UI + 05-05 đã P trên ep-r')

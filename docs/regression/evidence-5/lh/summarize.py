import json,glob,sys,statistics as st,collections
d=sys.argv[1]; rows=collections.defaultdict(list)
for f in sorted(glob.glob(d+'/.lighthouseci*/lhr-*.json')):
    j=json.load(open(f)); a=j['audits']; u=j['finalUrl'].split('3330')[1] or '/'
    sc=[x['transferSize'] for x in a['network-requests']['details']['items'] if x.get('resourceType')=='Script']
    rows[u].append(dict(lcp=a['largest-contentful-paint']['numericValue'],tbt=a['total-blocking-time']['numericValue'],cls=a['cumulative-layout-shift']['numericValue'],js=sum(sc),jsrs=a['resource-summary']['details']['items'][[i['resourceType'] for i in a['resource-summary']['details']['items']].index('script')]['transferSize'],bi=j['environment']['benchmarkIndex'],inp=a.get('interaction-to-next-paint',{}).get('numericValue'),lcpel=(a['largest-contentful-paint-element']['details']['items'][0]['items'][0]['node']['snippet'][:60] if a.get('largest-contentful-paint-element',{}).get('details',{}).get('items') else None)))
for u,r in rows.items():
    m=lambda k:st.median([x[k] for x in r])
    print(f"{u:16} n={len(r)} LCP med {m('lcp'):.0f} [{min(x['lcp'] for x in r):.0f}-{max(x['lcp'] for x in r):.0f}] TBT med {m('tbt'):.0f} [{min(x['tbt'] for x in r):.0f}-{max(x['tbt'] for x in r):.0f}] CLS {m('cls'):.3f} JS(netreq) {m('js')} JS(rs) {m('jsrs')} bi {m('bi'):.0f} {r[0]['lcpel']}")

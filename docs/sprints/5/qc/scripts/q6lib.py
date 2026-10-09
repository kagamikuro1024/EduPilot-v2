from q5lib import *
HID = 'QC-HID-' + uuid.uuid4().hex[:6]; REF = 'QC-REF-' + uuid.uuid4().hex[:6]
SUMSRC = '#include <cstdio>\nint main(){long long a,b;scanf("%lld %lld",&a,&b);printf("%lld\\n",a+b);}\n'
def mkcode(n=1, starter=None):
    s, j, _ = call('POST', Q, T, dict(type='CODE', title='qc6-%d' % n, topic='QC6', difficulty='EASY', stem='Đọc a, b in a+b')); qid = j['id']; v = j['version']; P = Q + '/' + qid
    s, j, _ = call('PUT', P + '/code', T, dict(languages=['c11', 'cpp17'], time_limit_ms=1000, memory_limit_mb=256, starter_code=starter or {'cpp17': '// khởi đầu cpp\n', 'c11': '// khởi đầu c\n'}, reference=dict(language='cpp17', source=SUMSRC + '// ' + REF), version=v)); assert s == 200, (s, j)
    tids = []
    for nm, i, e, smp in [('mau1', '1 2\n', '3\n', True), ('mau2', '10 20\n', '30\n', True), (HID + 'a', '5 5\n', '10\n', False), (HID + 'b', '100 200\n', '300\n', False)]:
        s, j, _ = call('POST', P + '/testcases', T, dict(name=nm, input=i, expected=e, is_sample=smp)); assert s in (200, 201), (s, j); tids.append(j['id'])
    print('approve', call('POST', P + '/testcases/approve', T, dict(ids=tids))[0:1])
    s, j, _ = call('POST', P + '/reference/verify', T, {}, idem()); print('verify', s, codes(j))
    return qid

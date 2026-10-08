from q3lib import *
import time, zipfile, io
T=login('teacher'); TA=login('ta'); SV=login('sv.gioi'); AD=login('admin')
s,j,_=call('GET','/me/courses',T); C1=[c['course']['id'] for c in j['items'] if c['course']['class_code']=='761987'][0]; C2=[c['course']['id'] for c in j['items'] if c['course']['class_code']=='761988'][0]
Q=f'/courses/{C1}/questions'
s,j,_=call('POST',Q,T,dict(type='CODE',title='Tổng hai số',topic='Cơ bản',difficulty='EASY',stem='Đọc a, b in a+b')); print('CODE tạo',s,codes(j)); qid=j.get('id'); v=j.get('version')
P=Q+'/'+qid
s,j,_=call('PUT',P+'/code',T,dict(languages=['cpp17'],version=v)); print('09',s,{k:j.get(k) for k in ['time_limit_ms','memory_limit_mb','output_limit_kb','checker','tests_version']}, codes(j)); v=j.get('version',v)
def code(**k):
    global v
    s,j,_=call('PUT',P+'/code',T,dict(version=v,**k))
    if s==200: v=j.get('version',v)
    return s,codes(j),j.get('tests_version')
print('10 tl',[code(languages=['cpp17'],time_limit_ms=x)[:2] for x in (99,100,10000,10001)])
print('10 mem',[code(languages=['cpp17'],memory_limit_mb=x)[:2] for x in (15,16,1024,1025)])
print('10 out',[code(languages=['cpp17'],output_limit_kb=x)[:2] for x in (0,1,16384,16385)])
print('10 eps',code(languages=['cpp17'],checker='FLOAT_EPS')[:2],code(languages=['cpp17'],checker='EXACT',float_eps='0.1')[:2],code(languages=['cpp17'],checker='FLOAT_EPS',float_eps='0.001')[:2],code(languages=['java'])[:2],code(languages=[])[:2])
code(languages=['c11','cpp17'],time_limit_ms=1000,memory_limit_mb=256,output_limit_kb=1024,checker='EXACT',float_eps=None)
a=code(languages=['c11','cpp17'],time_limit_ms=1500); b=code(languages=['c11','cpp17'],time_limit_ms=1500,checker='TOKENS'); c=code(languages=['cpp17']); d=code(languages=['cpp17'],starter_code={'cpp17':'int main(){}'}); print('11 tests_version sau từng đổi:',[x[2] for x in (a,b,c,d)])
code(languages=['c11','cpp17'],time_limit_ms=1000,checker='EXACT')
TC=P+'/testcases'
def tc(**k): return call('POST',TC,T,dict(name='t',input='1 2\n',expected='3\n',**k))
s,j,_=tc(); print('13 mặc định weight',s,j.get('weight'))
for n,k in [('w-1',dict(weight=-1)),('w1001',dict(weight=1001)),('w1.5',dict(weight=1.5)),('w0',dict(weight=0)),('w1000',dict(weight=1000))]:
    s,j,_=tc(**k); print('13',n,s,codes(j))
for n,nm in [('name rỗng',''),('name 61','x'*61),('name 60','x'*60)]:
    s,j,_=call('POST',TC,T,dict(name=nm,input='1',expected='1')); print('13',n,s,codes(j))
s,j,_=call('POST',TC,T,dict(name='big64',input='a'*65536,expected='1')); print('13 64KiB',s,codes(j))
s,j,_=call('POST',TC,T,dict(name='big64p',input='a'*65537,expected='1')); print('13 64KiB+1',s,codes(j))
s,j,_=call('POST',TC,T,dict(name='big1m',input='a'*1048577,expected='1')); print('13 1MiB+1',s,codes(j))
s,j,_=call('POST',TC,T,dict(name='nul',input='1\u0000',expected='1')); print('13 NUL',s,codes(j))
s,l,_=call('GET',TC,T); print('13 danh sách',s,len(l.get('items',[])))
print('15 SV',call('GET',TC,SV)[0],'ADMIN',call('GET',TC,AD)[0],'TA',call('GET',TC,TA)[0],'GV',call('GET',TC,T)[0],'GV id lớp khác',call('GET',f'/courses/{C2}/questions/{qid}/testcases',T)[0])
z=mkzip({'sample1.in':'1 2','sample1.out':'3','t2.in':'2 2','t2.out':'4','t10.in':'3 3','t10.ans':'6','t3.in':'4 4','t3.out':'8','t4.in':'5 5','t4.out':'10'})
b,ct=multipart({},'t.zip',z); s,j,t=call('POST',TC+'/import?dry_run=true',T,raw=b,ctype=ct); print('17 dry',s,t[:160])
s,l0,_=call('GET',TC,T); n0=len(l0['items']); b,ct=multipart({},'t.zip',z); s,j,t=call('POST',TC+'/import',T,raw=b,ctype=ct); s2,l1,_=call('GET',TC,T); print('17 thật',s,t[:100],'số test',n0,'→',len(l1['items']),[(x['name'],x['is_sample']) for x in l1['items'][-5:]])
for name,files in [('thiếu cặp',{'a.in':'1'}),('trùng .out+.ans',{'a.in':'1','a.out':'1','a.ans':'1'}),('thư mục lồng',{'d/a.in':'1','d/a.out':'1'}),('../evil.in',{'../evil.in':'1','../evil.out':'1'}),('/abs.in',{'/abs.in':'1','/abs.out':'1'}),('tệp lạ',{'a.in':'1','a.out':'1','x.exe':'MZ'}),('không UTF-8',{'a.in':b'\xff\xfe','a.out':'1'})]:
    z2=mkzip(files); b,ct=multipart({},'t.zip',z2); s,j,t=call('POST',TC+'/import',T,raw=b,ctype=ct); print('18',name,s,[(d.get('field') or d.get('file'),d.get('code')) for d in (j.get('details') or [])][:3])
big=mkzip({f't{i}.in':'1' for i in range(101)}|{f't{i}.out':'1' for i in range(101)}); b,ct=multipart({},'t.zip',big); s,j,t=call('POST',TC+'/import',T,raw=b,ctype=ct); print('18 101 test',s,codes(j))
s,l2,_=call('GET',TC,T); print('18 sau các zip lỗi số test =',len(l2['items']),'(mong',len(l1['items']),')')
def bomb(sz):
    bio=io.BytesIO()
    with zipfile.ZipFile(bio,'w',zipfile.ZIP_DEFLATED,compresslevel=9) as z: z.writestr('a.in','0'*sz); z.writestr('a.out','1')
    return bio.getvalue()
t0=time.time(); bz=bomb(60<<20); b,ct=multipart({},'t.zip',bz); s,j,t=call('POST',TC+'/import',T,raw=b,ctype=ct); print('20a bomb 60MiB (%d B nén)'%len(bz),s,codes(j),'%.1fs'%(time.time()-t0))
b,ct=multipart({},'t.zip',b'PK'+b'\x00'*(11<<20)); t0=time.time(); s,j,t=call('POST',TC+'/import',T,raw=b,ctype=ct); print('20c 11MiB',s,codes(j),'%.1fs'%(time.time()-t0))
print('gateway còn sống:',call('GET','/readyz')[0])
open('/tmp/q3/state.json','w').write(json.dumps(dict(C1=C1,C2=C2,qid=qid)))

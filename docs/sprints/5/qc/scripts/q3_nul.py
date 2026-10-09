from q3lib import *
T=login('teacher'); s,j,_=call('GET','/me/courses',T); C1=[c['course']['id'] for c in j['items'] if c['course']['class_code']=='761987'][0]; Q=f'/courses/{C1}/questions'
N='\u0000'; res=[]
def t(n,s,j): res.append((n,s,codes(j))); print(n,s,codes(j))
mc=lambda **k: dict(type='MCQ_SINGLE',title='t',topic='x',stem='s',options=[{'body':'a'},{'body':'b'}],correct=[0])|k
t('stem',*call('POST',Q,T,mc(stem='a'+N+'b'))[:2]); t('title',*call('POST',Q,T,mc(title='a'+N))[:2]); t('topic',*call('POST',Q,T,mc(topic=N))[:2])
t('option body',*call('POST',Q,T,mc(options=[{'body':'a'+N},{'body':'b'}]))[:2]); t('explanation',*call('POST',Q,T,mc(explanation='e'+N))[:2])
t('stem (escape \\u0000 trong JSON thô)',*call('POST',Q,T,raw=b'{"type":"TRUE_FALSE","title":"t","topic":"x","stem":"a\\u0000b","value":true}',ctype='application/json')[:2])
q=call('POST',Q,T,mc())[1]; P=Q+'/'+q['id']
t('PUT stem',*call('PUT',P,T,mc(stem='x'+N,version=q['version']))[:2])
c=call('POST',Q,T,dict(type='CODE',title='c',topic='x',stem='s'))[1]; PC=Q+'/'+c['id']
t('test input',*call('POST',PC+'/testcases',T,dict(name='n',input='1'+N,expected='1'))[:2]); t('test expected',*call('POST',PC+'/testcases',T,dict(name='n',input='1',expected='1'+N))[:2]); t('test name',*call('POST',PC+'/testcases',T,dict(name='n'+N,input='1',expected='1'))[:2])
t('starter_code',*call('PUT',PC+'/code',T,dict(version=call('GET',PC,T)[1]['version'],languages=['c11'],starter_code={'c11':'a'+N}))[:2]); t('reference source',*call('PUT',PC+'/code',T,dict(version=call('GET',PC,T)[1]['version'],languages=['c11'],reference=dict(language='c11',source='x'+N)))[:2])
t('suggest topic',*call('POST',Q+'/suggest',T,dict(kind='MCQ',topic='a'+N,count=1),idem())[:2])
E=f'/courses/{C1}/exams'; t('exam title',*call('POST',E,T,dict(title='a'+N,opens_at='2027-01-01T00:00:00Z',closes_at='2027-01-01T01:00:00Z',duration_minutes=30),idem())[:2])
# lành: Unicode / control khác
ok=call('POST',Q,T,mc(stem='Tiếng Việt ✓ \t tab\nxuống dòng 😀'))
print('lành (UTF-8, tab, xuống dòng, emoji):',ok[0])
bad=[r for r in res if r[1]==500]; print('SỐ CA 500:',len(bad),'/',len(res)); print('đều 422:',all(r[1]==422 for r in res))

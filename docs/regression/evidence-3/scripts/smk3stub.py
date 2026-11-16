# smk3stub = qstub + thêm mô hình: m-400 m-401 m-big (thân 5.000 ký tự HTML+stack) m-canary (phản chiếu Authorization+CANARY) m-ra2 m-ra30 m-slow10 m-slow40 m-slow130 m-midkill (stream cắt giữa chừng) m-prefail (stream: 503 trước byte đầu). /__count log chứa (model,t)
# OpenAI-compatible stub cho smoke hồi quy (qc2/SmkB). Hành vi theo tên mô hình; khoá khác "good-key" => 401.
# usage: python3 qstub.py 9741   ; GET /__count => số request chat/embeddings; POST /__reset
import json,sys,time,threading
from http.server import BaseHTTPRequestHandler,ThreadingHTTPServer
CNT={'n':0,'open':0,'peak':0,'log':[]}
L=threading.Lock()
class H(BaseHTTPRequestHandler):
    protocol_version='HTTP/1.1'
    def log_message(self,*a): pass
    def _send(self,code,obj,raw=None,hdr=None):
        b=raw if raw is not None else json.dumps(obj).encode()
        self.send_response(code); self.send_header('content-type','application/json'); self.send_header('content-length',str(len(b)))
        for k,v in (hdr or {}).items(): self.send_header(k,v)
        self.end_headers(); self.wfile.write(b)
    def do_GET(self):
        if self.path=='/__count': return self._send(200,{'n':CNT['n'],'peak':CNT['peak'],'log':CNT['log'][-20:]})
        self._send(404,{'error':'nf'})
    def do_POST(self):
        ln=int(self.headers.get('content-length') or 0); body=self.rfile.read(ln)
        if self.path=='/__reset':
            CNT.update(n=0,peak=0,log=[]); return self._send(200,{'ok':1})
        try: j=json.loads(body)
        except Exception: j={}
        m=j.get('model','')
        with L: CNT['n']+=1; CNT['open']+=1; CNT['peak']=max(CNT['peak'],CNT['open']); CNT['log'].append((m,round(time.time(),2)))
        try:
            key=(self.headers.get('authorization') or '').replace('Bearer ','')
            if key!='good-key': return self._send(401,{'error':{'message':'bad key','type':'auth'}})
            if m=='m-400': return self._send(400,{'error':{'message':'bad request','type':'invalid_request_error'}})
            if m=='m-401': return self._send(401,{'error':{'message':'unauthorized'}})
            if m=='m-big': return self._send(502,None,raw=(b'<html><body><h1>Bad gateway</h1><pre>Traceback (most recent call last):\n  File x.py line 1\n'+b'X'*5000+b'</pre></body></html>'))
            if m=='m-canary': return self._send(401,None,raw=json.dumps({'error':{'message':'invalid key; header Authorization: '+(self.headers.get('authorization') or '')+' CANARY-SMK3-8f3a9c sk-live1234567890abcdef Bearer abc.def.ghi'}}).encode())
            if m=='m-ra2': return self._send(429,{'error':{'message':'rate'}},hdr={'retry-after':'2'})
            if m=='m-ra30': return self._send(429,{'error':{'message':'rate'}},hdr={'retry-after':'30'})
            if m=='m-slow10': time.sleep(10)
            if m=='m-slow40': time.sleep(40)
            if m=='m-slow130': time.sleep(130)
            if m=='m-prefail' and j.get('stream'): return self._send(503,{'error':{'message':'down'}})
            if m=='m-midkill' and j.get('stream'):
                self.send_response(200); self.send_header('content-type','text/event-stream'); self.send_header('transfer-encoding','chunked'); self.end_headers()
                def ch2(s): self.wfile.write(('%x\r\n'%len(s)).encode()+s+b'\r\n'); self.wfile.flush()
                for w in ['Q','ok']:
                    ch2(('data: '+json.dumps({'id':'c','object':'chat.completion.chunk','model':m,'choices':[{'index':0,'delta':{'content':w+' '},'finish_reason':None}]})+'\n\n').encode()); time.sleep(0.2)
                import socket; self.connection.shutdown(socket.SHUT_RDWR); self.connection.close(); return
            if m=='m-429': return self._send(429,{'error':{'message':'rate'}},hdr={'retry-after':'2'})
            if m=='m-404': return self._send(404,{'error':{'message':'no model'}})
            if m=='m-500': return self._send(500,{'error':{'message':'boom'}})
            if m=='m-503': return self._send(503,{'error':{'message':'down'}})
            if m=='m-garbage': return self._send(200,None,raw=b'<html>not json</html>')
            if m=='m-hang': time.sleep(25)
            if m=='m-slow': time.sleep(3)
            if m=='m-slow1': time.sleep(1.2)
            if self.path.endswith('/embeddings'):
                dim=768 if m=='m-dim768' else 1536; inp=j.get('input'); inp=inp if isinstance(inp,list) else [inp]
                return self._send(200,{'object':'list','model':m,'data':[{'object':'embedding','index':i,'embedding':[1.0]+[0.0]*(dim-1)} for i,_ in enumerate(inp)],'usage':{'prompt_tokens':len(inp),'total_tokens':len(inp)}})
            if j.get('stream'):
                self.send_response(200); self.send_header('content-type','text/event-stream'); self.send_header('transfer-encoding','chunked'); self.end_headers()
                def ch(s): self.wfile.write(('%x\r\n'%len(s)).encode()+s+b'\r\n'); self.wfile.flush()
                for w in ['Q','ok','stream']:
                    ch(('data: '+json.dumps({'id':'c','object':'chat.completion.chunk','model':m,'choices':[{'index':0,'delta':{'content':w+' '},'finish_reason':None}]})+'\n\n').encode()); time.sleep(0.2)
                ch(('data: '+json.dumps({'id':'c','object':'chat.completion.chunk','model':m,'choices':[{'index':0,'delta':{},'finish_reason':'stop'}],'usage':{'prompt_tokens':5,'completion_tokens':3,'total_tokens':8}})+'\n\ndata: [DONE]\n\n').encode())
                self.wfile.write(b'0\r\n\r\n'); return
            return self._send(200,{'id':'c','object':'chat.completion','created':1,'model':m,'choices':[{'index':0,'message':{'role':'assistant','content':'Q ok'},'finish_reason':'stop'}],'usage':{'prompt_tokens':5,'completion_tokens':3,'total_tokens':8}})
        finally:
            with L: CNT['open']-=1
for p in sys.argv[1:]:
    ThreadingHTTPServer(('0.0.0.0',int(p)),H).__class__  # noqa
ths=[]
for p in sys.argv[1:]:
    s=ThreadingHTTPServer(('0.0.0.0',int(p)),H); t=threading.Thread(target=s.serve_forever,daemon=True); t.start(); ths.append(t)
while True: time.sleep(3600)

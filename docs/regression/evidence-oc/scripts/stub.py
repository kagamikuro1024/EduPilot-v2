import json,sys,time,threading
from http.server import BaseHTTPRequestHandler,ThreadingHTTPServer
LOG='/tmp/qc3oc/stub.log'; lock=threading.Lock(); cnt={}
class H(BaseHTTPRequestHandler):
    protocol_version='HTTP/1.1'
    def log_message(self,*a): pass
    def do_POST(self):
        n=int(self.headers.get('Content-Length') or 0); body=self.rfile.read(n).decode('utf8','replace')
        mode=self.path.split('/')[1]
        with lock:
            cnt[mode]=cnt.get(mode,0)+1
            open(LOG,'a').write(json.dumps({'t':time.time(),'mode':mode,'n':cnt[mode],'path':self.path,'ua':self.headers.get('User-Agent'),'session':self.headers.get('x-opencode-session'),'auth':bool(self.headers.get('Authorization')),'body':body})+'\n')
        if mode=='r429': code,out,hd=429,{"error":{"message":"rate limit"}},{'Retry-After':'1'}
        elif mode=='r500': code,out,hd=500,{"error":{"message":"boom"}},{}
        else:
            code,hd=200,{}
            stream='"stream":true' in body.replace(' ','')
            out={"id":"s1","object":"chat.completion","model":"m1","choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"stub trả lời"}}],"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}}
            if stream:
                d=json.dumps({"id":"s1","object":"chat.completion.chunk","model":"m1","choices":[{"index":0,"delta":{"content":"stub trả lời"}}]}); u=json.dumps({"id":"s1","object":"chat.completion.chunk","model":"m1","choices":[],"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}})
                b=f"data: {d}\n\ndata: {u}\n\ndata: [DONE]\n\n".encode(); self.send_response(200); self.send_header('Content-Type','text/event-stream'); self.send_header('Content-Length',str(len(b))); self.end_headers(); self.wfile.write(b); return
        b=json.dumps(out).encode(); self.send_response(code); self.send_header('Content-Type','application/json'); self.send_header('Content-Length',str(len(b)))
        for k,v in hd.items(): self.send_header(k,v)
        self.end_headers(); self.wfile.write(b)
    def do_GET(self):
        b=b'{"data":[]}'; self.send_response(200); self.send_header('Content-Type','application/json'); self.send_header('Content-Length',str(len(b))); self.end_headers(); self.wfile.write(b)
ThreadingHTTPServer(('0.0.0.0',9791),H).serve_forever()

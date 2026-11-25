import json,ssl,time,urllib.request,urllib.error,uuid
B="https://localhost:1143/api/v1"; ctx=ssl._create_unverified_context()
def call(path,body):
    r=urllib.request.Request(B+path,data=json.dumps(body).encode(),headers={"Content-Type":"application/json","Origin":"https://localhost:1143","Accept":"application/json"},method="POST")
    try:
        with urllib.request.urlopen(r,context=ctx,timeout=30) as x: return x.status,{} ,x.headers.get("Retry-After")
    except urllib.error.HTTPError as e:
        b=e.read(); 
        try: j=json.loads(b)
        except Exception: j={}
        return e.code,j.get("code"),e.headers.get("Retry-After")
tag=uuid.uuid4().hex[:6]; em=f"qcdbg.{tag}@edupilot.local"; pw=f"Lock!{tag}zA9"
print(call("/auth/register",{"email":em,"password":pw,"full_name":"QC dbg","student_code":""}))
t0=time.time(); wrong=0
for i in range(60):
    if wrong>=10: break
    s,c,ra=call("/auth/login",{"email":em,"password":f"Sai!{i}{tag}"})
    print(f"{time.time()-t0:6.1f}s i={i} -> {s} {c} retry-after={ra} wrong={wrong+(s==401)}")
    if s==401: wrong+=1
    else: time.sleep(2.5)
print("wrong",wrong)

const G="http://127.0.0.1:18096/api/v1",PW="Edupilot#Seed-2026";
const j=async(m,p,t,b)=>{const r=await fetch(G+p,{method:m,headers:{"content-type":"application/json",...(t?{authorization:`Bearer ${t}`}:{}),...(m==="POST"?{"Idempotency-Key":crypto.randomUUID()}:{})},body:b?JSON.stringify(b):undefined});return {s:r.status,b:await r.json().catch(()=>null),r}};
const tok=async u=>(await j("POST","/auth/login",null,{email:`${u}@edupilot.local`,password:PW})).b.access_token;
const C1=(await j("GET","/me/courses",await tok("teacher"))).b.items.find(x=>x.course.class_code==="761987").course.id;
const t=await tok("sv.kha");const sid=(await j("POST","/chat/sessions",t,{course_id:C1})).b.id;
const r=await fetch(`${G}/chat/sessions/${sid}/messages`,{method:"POST",headers:{authorization:`Bearer ${t}`,"content-type":"application/json","Idempotency-Key":crypto.randomUUID()},body:JSON.stringify({content:"Em là Bùi Thanh Khải, MSSV 20229002. Em đã nghỉ mấy buổi và được cộng bao nhiêu điểm phát biểu rồi ạ?"})});
const raw=await r.text();const blocks=[...raw.matchAll(/event: block\ndata: (.*)/g)].map(m=>m[1]);
const att=(await j("GET",`/courses/${C1}/me/attendance`,t)).b,par=(await j("GET",`/courses/${C1}/me/participation`,t)).b;
console.log(JSON.stringify({blocks:blocks.map(b=>b.slice(0,500)),attendance:JSON.stringify(att).slice(0,400),participation:JSON.stringify(par).slice(0,400)}));

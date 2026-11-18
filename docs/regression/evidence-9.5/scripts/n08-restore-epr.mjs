const G="https://localhost:1143/api/v1",PW="Edupilot#Seed-2026";
const j=async(m,p,t,b)=>{const r=await fetch(G+p,{method:m,headers:{"content-type":"application/json",...(t?{authorization:`Bearer ${t}`}:{}),...(m==="POST"?{"Idempotency-Key":crypto.randomUUID()}:{})},body:b?JSON.stringify(b):undefined});return{s:r.status,b:await r.json().catch(()=>null)}};
const t=(await j("POST","/auth/login",null,{email:"sv.kha@edupilot.local",password:PW})).b.access_token;
const tt=(await j("POST","/auth/login",null,{email:"teacher@edupilot.local",password:PW})).b.access_token;
const C=(await j("GET","/me/courses?limit=100",tt)).b.items.find(x=>x.course.class_code==="761987").course.id;
const sid=(await j("POST","/chat/sessions",t,{course_id:C})).b.id;
for(const q of ["Em tên Bùi Thanh Khải, MSSV 20229002. Hãy nhắc lại đúng họ tên và MSSV của em trong câu trả lời.","Xin chào, em là Bùi Thanh Khải, hãy gọi tên em ạ."]){
const r=await fetch(`${G}/chat/sessions/${sid}/messages`,{method:"POST",headers:{authorization:`Bearer ${t}`,"content-type":"application/json","Idempotency-Key":crypto.randomUUID()},body:JSON.stringify({content:q})});
const raw=await r.text();const text=[...raw.matchAll(/^data: (.*)$/gm)].map(m=>{try{const d=JSON.parse(m[1]);return d.delta??d.text??""}catch{return""}}).join("");
console.log(raw.slice(0,900).replace(/\n/g," ¶ "));console.log(JSON.stringify({q:q.slice(0,50),status:r.status,answer:text.slice(0,300),hasName:/Khải|20229002/.test(raw)}));}

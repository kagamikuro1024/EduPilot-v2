// k6 của SmkT4N (bản p211-today.js có thêm đếm theo mã trạng thái và theo token): 20 req/s, 60 s, /me/today, token GV + SV luân phiên
import http from 'k6/http'; import {check} from 'k6';
http.setResponseCallback(http.expectedStatuses({min:200,max:399}));
const th = {http_req_duration:['p(95)<300'], http_req_failed:['rate<0.005']};
for (const c of [200,304,401,403,429,500,502,503,504]) th[`http_reqs{status:${c}}`] = ['count>=0'];
for (const w of ['gv','sv']) { th[`http_req_duration{who:${w}}`] = ['p(95)<300']; th[`http_req_failed{who:${w}}`] = ['rate<0.005']; }
export const options={scenarios:{t:{executor:'constant-arrival-rate',rate:20,timeUnit:'1s',duration:'60s',preAllocatedVUs:20,maxVUs:80}},thresholds:th};
const T=__ENV.TOKENS.split(','); const B=__ENV.BASE;
export default function(){ const i=__ITER%T.length; const r=http.get(B+'/api/v1/me/today',{headers:{Authorization:'Bearer '+T[i]},tags:{who:i===0?'gv':'sv'}}); check(r,{ok:(x)=>x.status===200}); }

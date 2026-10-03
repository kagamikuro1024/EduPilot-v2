import http from 'k6/http'; import {check} from 'k6';
http.setResponseCallback(http.expectedStatuses({min:200,max:399}));
export const options={scenarios:{t:{executor:'constant-arrival-rate',rate:20,timeUnit:'1s',duration:'60s',preAllocatedVUs:20,maxVUs:80}},thresholds:{http_req_duration:['p(95)<300'],http_req_failed:['rate<0.005']}};
const T=__ENV.TOKENS.split(','); const B=__ENV.BASE;
export default function(){ const t=T[__ITER%T.length]; const r=http.get(B+'/api/v1/me/today',{headers:{Authorization:'Bearer '+t}}); check(r,{ok:(x)=>x.status===200}); }
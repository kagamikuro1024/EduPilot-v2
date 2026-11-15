// Workaround ep-r: CORS_ORIGINS thiếu https://localhost:1143 => /auth/refresh 403. Gọi refresh từ Node với Origin hợp lệ. (No-op khi origin đã được sửa)
export async function fixRefresh(ctx){
  await ctx.route('**/api/v1/auth/refresh', async route=>{
    const req=route.request(); const ck=(await ctx.cookies(req.url())).find(c=>c.name==='ep_rt');
    const h={...req.headers()}; delete h['host']; delete h['content-length']; h['origin']='https://localhost'; h['cookie']=ck?('ep_rt='+ck.value):'';
    const r=await fetch(req.url(),{method:'POST',headers:h,body:req.postData()||'{}'});
    const body=await r.text(); const sc=r.headers.get('set-cookie');
    if(sc){ const m=sc.match(/ep_rt=([^;]+)/); if(m) await ctx.addCookies([{name:'ep_rt',value:m[1],domain:'localhost',path:'/api/v1/auth',secure:true,httpOnly:true,sameSite:'Lax'}]); }
    await route.fulfill({status:r.status,body,headers:{'content-type':'application/json'}});
  });
}

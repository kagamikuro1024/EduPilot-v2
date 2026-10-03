// Máy chủ OpenAI giả của QC (Bun). Dùng trong Eval: const {QS}=await import('/abs/qserver.mjs'); QS.reset(); QS[9701].script=[{status:503},{}]
// step: {status,code,raw,ctype,headers,delay,hang,destroy,text,content,dim,tokDelay}; auto(body)→step khi hết kịch bản.
export const QS = {};
function mk(port) {
  const s = { port, log: [], script: [], dflt: null, srv: null, auto: null, open: 0, peak: 0 };
  s.srv = Bun.serve({ port, idleTimeout: 120, async fetch(req) {
    const t = Date.now(); const url = new URL(req.url); let body = null; try { body = await req.json(); } catch {}
    s.log.push({ t, path: url.pathname, auth: req.headers.get('authorization'), body, stream: !!body?.stream });
    s.open++; s.peak = Math.max(s.peak, s.open);
    try {
      const step = s.script.length ? s.script.shift() : (s.dflt || (s.auto ? s.auto(body, url) : null) || {});
      if (step.delay) await Bun.sleep(step.delay);
      if (step.hang) await Bun.sleep(step.hang);
      if (step.destroy) return new Response(new ReadableStream({ start(c) { c.error(new Error('x')); } }));
      if (step.raw !== undefined && (!step.status || step.status === 200)) return new Response(step.raw, { status: 200, headers: { 'content-type': step.ctype || 'application/json' } });
      if (step.status && step.status !== 200) return new Response(step.raw ?? JSON.stringify(step.body ?? { error: { message: 'err ' + step.status, type: 'x', code: step.code || null } }), { status: step.status, headers: { 'content-type': step.ctype || 'application/json', ...(step.headers || {}) } });
      if (url.pathname.endsWith('/embeddings')) { const inp = [].concat(body.input); const dim = step.dim || 1536; return Response.json({ object: 'list', model: body.model, data: inp.map((_, i) => ({ object: 'embedding', index: i, embedding: Array.from({ length: dim }, (_, k) => k === 0 ? 1 : 0) })), usage: { prompt_tokens: inp.length, total_tokens: inp.length } }); }
      if (url.pathname.endsWith('/chat/completions')) {
        const text = step.text ?? 'Q ok';
        if (body.stream) { const enc = new TextEncoder(); return new Response(new ReadableStream({ async start(c) { for (const w of text.split(' ')) { c.enqueue(enc.encode('data: ' + JSON.stringify({ id: 'c', object: 'chat.completion.chunk', model: body.model, choices: [{ index: 0, delta: { content: w + ' ' }, finish_reason: null }] }) + '\n\n')); await Bun.sleep(step.tokDelay || 20); } c.enqueue(enc.encode('data: ' + JSON.stringify({ id: 'c', object: 'chat.completion.chunk', model: body.model, choices: [{ index: 0, delta: {}, finish_reason: 'stop' }], usage: { prompt_tokens: 5, completion_tokens: 3, total_tokens: 8 } }) + '\n\n')); c.enqueue(enc.encode('data: [DONE]\n\n')); c.close(); } }), { headers: { 'content-type': 'text/event-stream' } }); }
        return Response.json({ id: 'c', object: 'chat.completion', created: 1, model: body.model, choices: [{ index: 0, message: { role: 'assistant', content: step.content ?? text }, finish_reason: 'stop' }], usage: { prompt_tokens: step.tin ?? 5, completion_tokens: step.tout ?? 3, total_tokens: 8 } });
      }
      return new Response('nf', { status: 404 });
    } finally { s.open--; }
  } });
  return s;
}
for (const p of [9701, 9702, 9703]) QS[p] = mk(p);
QS.reset = () => { for (const p of [9701, 9702, 9703]) { QS[p].log.length = 0; QS[p].script.length = 0; QS[p].dflt = null; QS[p].auto = null; QS[p].peak = 0; } };
QS.stop = () => { for (const p of [9701, 9702, 9703]) QS[p].srv.stop(true); };
QS[9701].auto = (body) => { const all = JSON.stringify(body); if (body.input !== undefined) return all.includes('DIM768') ? { dim: 768 } : {}; if (all.includes('BADSCHEMA')) return { content: '{"x":1}' }; if (body.response_format) return { content: '{"category":"hoi_dap","confidence":0.9}' }; return {}; };

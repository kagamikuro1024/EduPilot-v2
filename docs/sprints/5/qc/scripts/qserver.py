#!/usr/bin/env python3
"""QC US-PE-03 TC-35/36/37: máy chủ giả tương thích OpenAI. Ghi MỌI thân yêu cầu vào /tmp/q3/llm-requests.jsonl; trả JSON có cấu trúc theo prompt."""
import json, http.server, sys
OUT = '/tmp/q3/llm-requests.jsonl'
def opt(b, c): return {'body': b, 'correct': c}
MCQ = {'questions': [
 {'type': 'MCQ_SINGLE', 'title': 'AES là gì', 'stem': 'AES là loại mã nào?', 'options': [opt('Đối xứng', True), opt('Bất đối xứng', False), opt('Băm', False)], 'value': None, 'explanation': 'AES là mã khối đối xứng.', 'difficulty': 'MEDIUM'},
 {'type': 'MCQ_MULTI', 'title': 'Thuật toán đối xứng', 'stem': 'Chọn mã đối xứng', 'options': [opt('AES', True), opt('DES', True), opt('RSA', False)], 'value': None, 'explanation': 'AES, DES.', 'difficulty': 'EASY'},
 {'type': 'TRUE_FALSE', 'title': 'RSA đối xứng', 'stem': 'RSA là mã đối xứng.', 'options': [], 'value': False, 'explanation': 'RSA bất đối xứng.', 'difficulty': 'EASY'},
 {'type': 'MCQ_SINGLE', 'title': 'Sai: một đáp án', 'stem': 'x', 'options': [opt('chỉ một', True)], 'value': None, 'explanation': '', 'difficulty': 'EASY'},
 {'type': 'MCQ_SINGLE', 'title': 'Sai: hai đáp án đúng', 'stem': 'x', 'options': [opt('a', True), opt('b', True)], 'value': None, 'explanation': '', 'difficulty': 'EASY'}]}
TESTS = {'inputs': [{'name': 'nhỏ', 'input': '1 2\n', 'note': 'cơ bản'}, {'name': 'âm', 'input': '-5 3\n', 'note': 'số âm'}, {'name': 'lớn', 'input': '1000000 2000000\n', 'note': 'lớn'}]}
class H(http.server.BaseHTTPRequestHandler):
    def do_POST(s):
        n = int(s.headers.get('content-length', 0)); body = s.rfile.read(n).decode()
        open(OUT, 'a').write(json.dumps({'path': s.path, 'body': json.loads(body) if body else None}, ensure_ascii=False) + '\n')
        try: msgs = json.loads(body).get('messages', [])
        except Exception: msgs = []
        sysm = ' '.join(m['content'] for m in msgs if m['role'] == 'system')
        payload = MCQ if 'người ra đề trắc nghiệm' in sysm else TESTS
        resp = {'id': 'chatcmpl-qc', 'object': 'chat.completion', 'created': 1791480000, 'model': 'qc-model', 'choices': [{'index': 0, 'finish_reason': 'stop', 'logprobs': None, 'message': {'role': 'assistant', 'refusal': None, 'content': json.dumps(payload, ensure_ascii=False)}}], 'usage': {'prompt_tokens': 10, 'completion_tokens': 10, 'total_tokens': 20}}
        b = json.dumps(resp).encode(); s.send_response(200); s.send_header('Content-Type', 'application/json'); s.send_header('Content-Length', str(len(b))); s.end_headers(); s.wfile.write(b)
    def do_GET(s):
        b = json.dumps({'data': [{'id': 'qc-model'}]}).encode(); s.send_response(200); s.send_header('Content-Type', 'application/json'); s.send_header('Content-Length', str(len(b))); s.end_headers(); s.wfile.write(b)
    def log_message(s, *a): pass
http.server.ThreadingHTTPServer(('0.0.0.0', int(sys.argv[1]) if len(sys.argv) > 1 else 48500), H).serve_forever()

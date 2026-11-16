"""TC-PE02-52 (đo gọn): dừng judge 75 s dưới khoá toàn cục; đếm log warn 'quá 60 giây'; bật lại → đo từ lúc judge tự trả /version tới khi worker /readyz = up. Chạy: python3 s20e_judgeup.py"""
import os, subprocess, time, json, re; os.environ['SUITE'] = 'PE-02M'
import rl
LOCK = '/tmp/qc2-lock-global'; LOG = '/tmp/qc2-global-changes.log'
def logc(m): open(LOG, 'a').write(time.strftime('%H:%M:%S') + ' SmkM: ' + m + '\n')
def dk(*a): return subprocess.run(['docker', *a], capture_output=True, text=True).stdout.strip()
def cx(url, extra=()): return subprocess.run(['docker', 'exec', 'qc2-s5-curl', 'curl', '-s', '-m', '3', *extra, url], capture_output=True, text=True).stdout
t_w = time.time()
while True:
    try: os.mkdir(LOCK); break
    except FileExistsError:
        if time.time() - t_w > 1500: raise SystemExit('không lấy được khoá')
        time.sleep(15)
t_lock = time.time(); logc('LOCK — TC-PE02-52: dừng judge 75 s rồi bật lại; hoàn lại ≤ 4 phút')
subprocess.run(['docker', 'rm', '-f', 'qc2-s5-curl'], capture_output=True); dk('run', '-d', '--rm', '--name', 'qc2-s5-curl', '--network', 'ep-r_default', '--entrypoint', 'sleep', 'curlimages/curl', '900'); dk('network', 'connect', 'ep-r_judge_net', 'qc2-s5-curl')
try:
    t_stop = time.time(); dk('stop', '-t', '2', 'ep-r-judge-1'); dn = None
    for _ in range(40):
        if '"judge":"down"' in cx('http://ep-r-worker-1:8081/readyz').replace(' ', ''): dn = time.time() - t_stop; break
        time.sleep(1)
    hz = cx('http://ep-r-worker-1:8081/healthz', ['-o', '/dev/null', '-w', '%{http_code}']); rzcode = cx('http://ep-r-worker-1:8081/readyz', ['-o', '/dev/null', '-w', '%{http_code}'])
    time.sleep(max(0, 78 - (time.time() - t_stop)))
    warn = int(subprocess.run("docker logs --since %ds ep-r-worker-1 2>&1 | grep -c 'máy chấm không phản hồi quá 60 giây'" % (int(time.time() - t_stop) + 2), shell=True, capture_output=True, text=True).stdout.strip() or 0)
    t_start = time.time(); dk('start', 'ep-r-judge-1'); jr = None; up = None
    for _ in range(120):
        if jr is None and cx('http://judge:5050/version').strip(): jr = time.time() - t_start
        if '"judge":"up"' in cx('http://ep-r-worker-1:8081/readyz').replace(' ', ''): up = time.time() - t_start; break
        time.sleep(0.5)
    delta = None if (jr is None or up is None) else up - jr
    ok = dn is not None and hz == '200' and rzcode == '200' and warn == 1 and delta is not None and delta <= 10.5
    rl.R('TC-PE02-52', ok, f'judge dừng (dừng container, ghi nhật ký khoá): worker /healthz {hz} (độc lập judge), /readyz HTTP {rzcode} với judge:"down" sau {dn and round(dn,1)} s; sau {int(time.time()-t_stop-(time.time()-t_start))} s dừng, log warn "máy chấm không phản hồi quá 60 giây" = {warn} dòng (TC: đúng 1); bật lại: judge tự trả /version sau {jr and round(jr,1)} s (khởi động container), worker /readyz judge=up sau {up and round(up,1)} s kể từ lúc bật → trễ nhịp thăm dò {delta and round(delta,1)} s (≤10 s theo chu kỳ 10 s); lượt đo tải máy ~{os.getloadavg()[0]:.0f}')
finally:
    dk('start', 'ep-r-judge-1'); dk('rm', '-f', 'qc2-s5-curl'); time.sleep(3); os.rmdir(LOCK); logc(f'UNLOCK — judge đã bật; giữ {time.time()-t_lock:.0f}s')

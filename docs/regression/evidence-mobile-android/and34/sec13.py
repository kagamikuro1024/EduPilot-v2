import lib34 as L, m, time, json, base64, subprocess, os
L.memguard()
call = lambda meth, args: json.loads(m.cdp("window.Capacitor.nativePromise('EduPilotAttest','%s',%s).then(r=>JSON.stringify(r)).catch(e=>JSON.stringify({err:String(e)}))" % (meth, json.dumps(args))))
msg = "sec13-" + str(int(time.time()))
a = call("sign", {"message": msg}); b = call("sign", {"message": msg})
print("sign x2: cung khoa:", a["public_key"] == b["public_key"], "| chu ky khac nhau (ECDSA ngau nhien):", a["signature"] != b["signature"], "| DER dai", len(base64.b64decode(a["signature"])), len(base64.b64decode(b["signature"])))
d = L.WORKD + "/sec13"; os.makedirs(d, exist_ok=True)
open(d + "/pub.der", "wb").write(base64.b64decode(a["public_key"])); open(d + "/msg.txt", "w").write(msg)
for k, x in (("a", a), ("b", b)):
    open(f"{d}/sig_{k}.der", "wb").write(base64.b64decode(x["signature"]))
    r = subprocess.run(f"openssl dgst -sha256 -verify <(openssl pkey -pubin -inform DER -in {d}/pub.der -outform PEM) -signature {d}/sig_{k}.der {d}/msg.txt", shell=True, executable="/bin/bash", capture_output=True, text=True); print("openssl verify", k, ":", (r.stdout + r.stderr).strip())
# chuỗi chứng thực (tạo lại khoá với challenge) -> mức bảo mật của lá
ch = base64.urlsafe_b64encode(os.urandom(32)).decode().rstrip("=")
r = call("registerDevice", {"nonce": ch})
chain = r.get("attestation_chain", []); print("chuoi chung chi:", len(chain), "chung chi")
if chain:
    open(d + "/leaf.der", "wb").write(base64.b64decode(chain[0]))
    out = subprocess.run(f"openssl x509 -inform DER -in {d}/leaf.der -noout -issuer -subject; openssl asn1parse -inform DER -in {d}/leaf.der | grep -A4 -m1 '1.3.6.1.4.1.11129.2.1.17'", shell=True, capture_output=True, text=True).stdout; print(out)
    ext = subprocess.run(f"openssl asn1parse -inform DER -in {d}/leaf.der | grep -A2 -m1 '1.3.6.1.4.1.11129.2.1.17' | tail -1", shell=True, capture_output=True, text=True).stdout.strip(); print("ext:", ext[:120])
    off = ext.split(":")[0].strip()
    if off.isdigit():
        kd = subprocess.run(f"openssl asn1parse -inform DER -in {d}/leaf.der -strparse {off} | head -6", shell=True, capture_output=True, text=True).stdout; print("KeyDescription:\n" + kd)

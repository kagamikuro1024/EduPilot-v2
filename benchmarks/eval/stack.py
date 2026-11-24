#!/usr/bin/env python3
"""Cấu hình tuyến LLM của stack đo qua API ADMIN (không ghi DB trực tiếp). Khoá không bao giờ được in / ghi tệp.

  stack.py route --tag P1 --model ag/gemini-3.8-flash      6 tác vụ sinh chữ → model này của provider `9router` (tạo nếu chưa có); ghi raw/providers.json
  stack.py embed-real [--only 761987]                       PM cho phép (trần 1.000 lời gọi): provider `gemini-embed-eval` (gemini-embedding-001, 1536 chiều) + EMBEDDING → nó + reindex mọi lớp (hoặc một lớp)
  stack.py embed-fake [--only 761987]                       EMBEDDING → provider fake, XOÁ provider gemini-embed-eval, reindex giả; in số lời gọi nhúng thật (llm_audit)
  stack.py routes-save FILE / routes-restore FILE           stack dùng chung: chụp tuyến trước khi đổi, trả đúng chuỗi + params sau lượt đo
Khoá 9router: ROUTER_KEY hoặc ~/.9router/db/data.sqlite (chỉ đọc). Khoá Gemini: GEMINI_ENV_FILE (mặc định ~/Documents/TA_Agent_v2-s95/.env.local), đọc trong tiến trình.
"""
import argparse
import json
import os
import sqlite3
from pathlib import Path

import lib
from lib import Fail, call, login

TEXT_TASKS = ("CHAT", "CLASSIFY", "UTILITY", "GRADING", "QUESTION_GEN", "INSIGHT")
ROUTER, ROUTER_URL = "9router", "http://host.docker.internal:20128/v1"
EMBED_NAME, EMBED_MODEL = "gemini-embed-eval", "gemini-embedding-001"


def router_key():
    if os.environ.get("ROUTER_KEY"):
        return os.environ["ROUTER_KEY"]
    c = sqlite3.connect("file:" + os.path.expanduser("~/.9router/db/data.sqlite") + "?mode=ro", uri=True)
    row = c.execute("select key from apiKeys where isActive = 1 limit 1").fetchone()
    if not row:
        raise Fail("9router không có khoá đang bật")
    return row[0]


def gemini_key():
    p = Path(os.path.expanduser(os.environ.get("GEMINI_ENV_FILE", "~/Documents/TA_Agent_v2-s95/.env.local")))
    for line in p.read_text(encoding="utf-8").splitlines():
        if line.startswith("GEMINI_API_KEY="):
            k = line.split("=", 1)[1].strip().strip('"')
            if k:
                return k
    raise Fail(f"không thấy GEMINI_API_KEY trong {p}")


def providers(adm):
    st, j = call("GET", "/admin/llm/providers", adm)
    if st != 200:
        raise Fail(f"GET providers → {st}")
    return {p["name"]: p for p in j["items"]}


def routes(adm):
    st, j = call("GET", "/admin/llm/routes", adm)
    if st != 200:
        raise Fail(f"GET routes → {st}")
    items = j.get("items") or j.get("routes") or []
    return {r["task"]: r for r in items}


def put_route(adm, task, model_id):
    v = routes(adm).get(task, {}).get("version", 0)
    st, j = call("PUT", "/admin/llm/routes", adm, {"task": task, "chain": [model_id], "version": v})
    if st != 200:
        raise Fail(f"PUT route {task} → {st} {j}")
    return j


def ensure_model(adm, name, ptype, base_url, key, model, kind, price_in, rpm, dims=None):
    have = providers(adm)
    m = {"model": model, "kind": kind, "price_in": price_in, "price_out": "0", "enabled": True, **({"dims": dims} if dims else {})}
    if name not in have:
        body = {"type": ptype, "name": name, "api_key": key, "enabled": True, "rpm_limit": rpm, "models": [m], **({"base_url": base_url} if base_url else {})}
        st, p = call("POST", "/admin/llm/providers", adm, body, idem=lib.new_key(f"eval-prov-{name}"))
        if st not in (200, 201):
            raise Fail(f"tạo provider {name} → {st} {(p or {}).get('code')}")
    else:
        p = have[name]
        if not any(x["model"] == model for x in p["models"]):
            models = [{k: x[k] for k in ("model", "kind", "price_in", "price_out", "enabled") if k in x} for x in p["models"]] + [m]
            st, p = call("PUT", f"/admin/llm/providers/{p['id']}", adm, {"type": p["type"], "name": name, "models": models, "rpm_limit": rpm, "version": p["version"], **({"base_url": p["base_url"]} if p.get("base_url") else {})})
            if st != 200:
                raise Fail(f"thêm model {model} → {st} {(p or {}).get('code')}")
    p = providers(adm)[name]
    return p, next(x["id"] for x in p["models"] if x["model"] == model)


def save_providers(patch):
    p = lib.RAW / "providers.json"
    cur = json.loads(p.read_text()) if p.exists() else {}
    cur.update(patch)
    lib.save_raw("providers.json", cur)


def reindex_all(only=None):
    """Mọi lớp của giảng viên seed `teacher` (dạy mọi lớp seed + KC106), hoặc chỉ lớp `only`: POST …/documents/reindex, chờ việc xong."""
    done = []
    for m in lib.list_all("/me/courses", login("teacher")):
        c = m["course"]
        if only and c["class_code"] != only:
            continue
        st, j = call("POST", f"/courses/{c['id']}/documents/reindex", login("teacher"), {}, idem=lib.new_key("eval-reindex"))
        if st == 202 and j and j.get("job_id"):
            done.append((c["class_code"], lib.wait_job(login("teacher"), j["job_id"], timeout=1800)["status"]))
        else:
            done.append((c["class_code"], st))
    print("reindex:", done)
    print("đoạn chưa có vector:", lib.sql("select count(*) as n from content_chunks where embedding is null")[0]["n"])


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    sp = ap.add_subparsers(dest="cmd", required=True)
    r = sp.add_parser("route")
    r.add_argument("--tag", required=True)
    r.add_argument("--model", required=True)
    sp.add_parser("embed-real").add_argument("--only", help="chỉ reindex lớp có class_code này (bớt lời gọi nhúng thật)")
    sp.add_parser("embed-fake").add_argument("--only", help="chỉ reindex lớp có class_code này")
    for name in ("routes-save", "routes-restore"):  # stack dùng chung (ep-qc): chụp tuyến trước khi đổi, trả đúng tuyến cũ sau lượt đo
        sp.add_parser(name).add_argument("file")
    a = ap.parse_args()
    lib.refuse_demo()
    if not lib.ready():
        lib.die("Gateway chưa chạy")
    adm = login("admin")
    try:
        if a.cmd == "routes-save":
            Path(a.file).write_text(json.dumps(routes(adm), ensure_ascii=False, indent=1))
            print("đã chụp:", {t: len(r.get("chain") or []) for t, r in routes(adm).items()})
        elif a.cmd == "routes-restore":
            for t, r in json.loads(Path(a.file).read_text()).items():
                body = {"task": t, "chain": [x["model_id"] for x in r.get("chain") or []], "params": r.get("params") or {}, "version": routes(adm)[t]["version"]}
                if not body["chain"]:
                    continue
                st, j = call("PUT", "/admin/llm/routes", adm, body)
                if st != 200:
                    raise Fail(f"trả tuyến {t} → {st} {j}")
            print("đã trả tuyến:", sorted(json.loads(Path(a.file).read_text())))
        elif a.cmd == "route":
            _, mid = ensure_model(adm, ROUTER, "openai_compatible", ROUTER_URL, router_key(), a.model, "chat", "0", 20)
            for t in TEXT_TASKS:
                put_route(adm, t, mid)
            save_providers({a.tag: {"model": a.model, "via": "9router (openai_compatible)", "since": lib.now_iso()}})
            print(f"{a.tag}: 6 tác vụ sinh chữ → {a.model}")
        elif a.cmd == "embed-real":
            _, mid = ensure_model(adm, EMBED_NAME, "gemini", None, gemini_key(), EMBED_MODEL, "embedding", "3900", 60, dims=1536)
            t0 = lib.now_iso()
            j = put_route(adm, "EMBEDDING", mid)
            save_providers({"embedding": f"thật: {EMBED_MODEL} (1536 chiều, Gemini trực tiếp — PM cho phép, trần 1.000 lời gọi)", "embed_real_since": t0})
            print(f"EMBEDDING → {EMBED_MODEL}; reindex_required={j.get('reindex_required')}")
            reindex_all(a.only)
        else:
            # fake trong DB mang khoá = FAKE_LLM_VALID_KEY của stack (env ep-qc, P-11-29); provider fake theo env (khoá rỗng) sẽ bị AUTH khi khoá đó được đặt
            _, fid = ensure_model(adm, "fake-eval", "fake", None, os.environ.get("EVAL_FAKE_KEY") or "fake", "fake-embed", "embedding", "0", 100000, dims=1536)
            put_route(adm, "EMBEDDING", fid)
            have = providers(adm)
            if EMBED_NAME in have:
                st, _ = call("DELETE", f"/admin/llm/providers/{have[EMBED_NAME]['id']}", adm)
                print(f"xoá {EMBED_NAME} → {st}")
            since = (lib.load_raw("providers.json") or {}).get("embed_real_since")
            if since:
                n = lib.sql(f"select count(*) as n, coalesce(sum(tokens_in),0) as tok from llm_audit where task = 'EMBEDDING' and model = '{EMBED_MODEL}' and created_at >= {lib.lit(since)}")[0]
                save_providers({"embedding": "giả (fake) — đã trả về sau lượt đo", "embed_real_calls": n["n"], "embed_real_tokens": n["tok"], "embed_real_until": lib.now_iso()})
                print(f"lời gọi nhúng thật: {n['n']} (token vào {n['tok']})")
            else:
                save_providers({"embedding": "giả (fake)"})
            reindex_all(a.only)
    except Fail as e:
        lib.die(f"LỖI: {e}", 2)


if __name__ == "__main__":
    main()

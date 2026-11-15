# Trợ giúp pha 2 sprint 10.5 — ep-qc (offset 500), provider giả, KHÔNG Gemini thật.
import os, subprocess, time, json, sys
sys.path.insert(0, os.path.dirname(__file__))
from dmlib import *  # raw, tok, api, ok, course, sql

ROOT = os.path.expanduser("~/Documents/TA_Agent_qc5")
BASE_ENV = dict(os.environ, EP_PORT_OFFSET="500", COMPOSE_PROJECT_NAME="ep-qc", BLOB_PUBLIC_ENDPOINT="localhost:9500",
                GEMINI_API_KEY="", LLM_DECISION_ALLOW_REAL="false", EP_PORT_MOCK_GRAPH="8590", EP_PORT_MOCK_IMAP="8591",
                EP_PORT_POSTGRES="5933", EP_PORT_REDIS="6880", EP_PORT_MINIO="9500", EP_PORT_MINIO_CONSOLE="9501", EP_PORT_SMTP="1525",
                EP_PORT_MAILPIT="8525", EP_PORT_HTTP="580", EP_PORT_HTTPS="943", EP_PORT_FRONTEND="3500",
                NEXT_PUBLIC_API_URL="https://localhost:943", APP_PUBLIC_URL="https://localhost:943",
                CORS_ORIGINS="https://localhost:943,http://localhost:3500",
                RATE_LIMIT_IP_PER_MIN="5000", AUTH_LOGIN_IP_PER_MIN="1000", AUTH_REGISTER_IP_PER_HOUR="1000", AUTH_TOKEN_IP_PER_MIN="1000")

def compose(*args, env=None):
    e = dict(BASE_ENV); e.update(env or {})
    r = subprocess.run(["docker", "compose", "--env-file", ".env.local", "-f", "docker-compose.local.yml", "-f", "docker-compose.test.yml", "-p", "ep-qc", *args],
                       capture_output=True, text=True, cwd=ROOT, env=e)
    return r.stdout + r.stderr

def restart_gw(**env):
    """Khởi động lại gateway + worker với biến env (chuỗi rỗng = mặc định)."""
    out = compose("up", "-d", "--no-deps", "--force-recreate", "gateway", "worker", env={k: str(v) for k, v in env.items()})
    for _ in range(60):
        c, _, _ = raw("GET", "/healthz", base="https://localhost:943") if False else (0, 0, 0)
        time.sleep(2)
        st = compose("ps", "--format", "{{.Service}} {{.Health}}")
        if "gateway healthy" in st and "worker healthy" in st: break
    return st

def set_gate(gate, system=True, course_codes=("761987",), enabled=True):
    sql(f"UPDATE decision_gates SET enabled={'true' if system else 'false'}, version=version+1 WHERE gate='{gate}'")
    for code in ("761987", "761988"):
        cid = sql(f"select id from courses where class_code='{code}'")[0][0]
        en = enabled and code in course_codes
        sql(f"INSERT INTO course_decision_gates(course_id,gate,enabled) VALUES ('{cid}','{gate}',{'true' if en else 'false'}) ON CONFLICT (course_id,gate) DO UPDATE SET enabled=EXCLUDED.enabled, version=course_decision_gates.version+1")
    compose("exec", "-T", "redis", "redis-cli", "publish", "ep:decision:cfg", "x")
    time.sleep(1.5)

def gates_off():
    sql("UPDATE decision_gates SET enabled=false"); sql("UPDATE course_decision_gates SET enabled=false")
    compose("exec", "-T", "redis", "redis-cli", "publish", "ep:decision:cfg", "x"); time.sleep(1.5)

def decide_rows(since):
    return sql(f"select cascade_gate,cascade_path,status,coalesce(error_kind,''),lane from llm_audit where task='DECIDE' and created_at>='{since}' order by created_at")

def now_db():
    return sql("select now()")[0][0]

def llm_rows(since, task=None):
    t = f"and task='{task}'" if task else ""
    return sql(f"select task,status,coalesce(cascade_gate,''),coalesce(cascade_path,'') from llm_audit where created_at>='{since}' {t} order by created_at")

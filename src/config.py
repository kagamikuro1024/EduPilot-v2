import os
from dotenv import load_dotenv

load_dotenv()

# --- AI Provider Keys ---
ANTHROPIC_API_KEY = os.getenv("ANTHROPIC_API_KEY", "")
OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
DEFAULT_MODEL = os.getenv("DEFAULT_MODEL", "gpt-5.4-mini")
LOG_LEVEL = os.getenv("LOG_LEVEL", "INFO")

# Internal utility model for RAG rerank / query-rewrite / synthesis passes.
# Defaults to the historical hardcoded value; override via env only.
RAG_UTILITY_MODEL = os.getenv("RAG_UTILITY_MODEL", "gpt-5.4-mini")

# Explicit OpenAI client timeouts (seconds) and bounded retries.
# Defaults keep behavior conservative; the OpenAI SDK default is 600s/2 retries.
def _env_float(name: str, default: float) -> float:
    try:
        return float(os.getenv(name, str(default)))
    except ValueError:
        return default

def _env_int(name: str, default: int) -> int:
    try:
        return int(os.getenv(name, str(default)))
    except ValueError:
        return default

OPENAI_TIMEOUT_SECONDS = _env_float("OPENAI_TIMEOUT_SECONDS", 120.0)
OPENAI_MAX_RETRIES = _env_int("OPENAI_MAX_RETRIES", 2)

# --- App Environment ---
APP_ENV = os.getenv("APP_ENV", "development")

# --- Cache & Vector ---
DATABASE_URL = os.getenv("DATABASE_URL", "")
# SECURITY: never ship real credentials as a code default. Deployments must set
# REDIS_URL explicitly; the semantic cache degrades gracefully when unreachable.
REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379/0")
REDIS_SOCKET_TIMEOUT_SECONDS = _env_float("REDIS_SOCKET_TIMEOUT_SECONDS", 3.0)
REDIS_CONNECT_TIMEOUT_SECONDS = _env_float("REDIS_CONNECT_TIMEOUT_SECONDS", 3.0)
RAG_MAX_RETRIEVAL_DISTANCE = float(os.getenv("RAG_MAX_RETRIEVAL_DISTANCE", "0.55"))
RAG_RETRIEVAL_CANDIDATES = int(os.getenv("RAG_RETRIEVAL_CANDIDATES", "50"))
RAG_RERANK_LIMIT = int(os.getenv("RAG_RERANK_LIMIT", "15"))
RAG_FALLBACK_MAX_DISTANCE = float(os.getenv("RAG_FALLBACK_MAX_DISTANCE", "0.65"))
RAG_RELATIVE_DISTANCE_RATIO = float(os.getenv("RAG_RELATIVE_DISTANCE_RATIO", "1.5"))
RAG_DEBUG_TOP_K = int(os.getenv("RAG_DEBUG_TOP_K", "5"))

# --- gRPC ---
GRPC_PORT = int(os.getenv("GRPC_PORT", "50051"))

# --- Java callback for document status sync ---
JAVA_CALLBACK_URL = os.getenv("JAVA_CALLBACK_URL", "http://localhost:8080")
INTERNAL_CALLBACK_TOKEN = os.getenv("INTERNAL_CALLBACK_TOKEN", "")

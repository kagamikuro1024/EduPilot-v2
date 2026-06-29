import os
from dotenv import load_dotenv

load_dotenv()

# --- AI Provider Keys ---
ANTHROPIC_API_KEY = os.getenv("ANTHROPIC_API_KEY", "")
OPENAI_API_KEY = os.getenv("OPENAI_API_KEY", "")
DEFAULT_MODEL = os.getenv("DEFAULT_MODEL", "gpt-4o-mini")
LOG_LEVEL = os.getenv("LOG_LEVEL", "INFO")

# --- App Environment ---
APP_ENV = os.getenv("APP_ENV", "development")

# --- Cache & Vector ---
DATABASE_URL = os.getenv("DATABASE_URL", "")
REDIS_URL = os.getenv("REDIS_URL", "redis://:REDACTED_PASSWORD@legacy-host.invalid:6380/0")
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

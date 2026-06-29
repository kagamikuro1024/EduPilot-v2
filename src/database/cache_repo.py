import logging
import hashlib
import threading
import json
import numpy as np
from typing import Any, List, Optional
from redis import Redis
from redis.commands.search.field import TextField, VectorField
from redis.commands.search.index_definition import IndexDefinition, IndexType
from redis.commands.search.query import Query
from src.config import REDIS_URL

logger = logging.getLogger(__name__)

# Constants
INDEX_NAME = "idx:semantic_cache"
DOC_PREFIX = "cache:qa:"
VECTOR_DIM = 1536  # text-embedding-3-small
DISTANCE_THRESHOLD = 0.10  # Equivalent to 0.90 similarity
CACHE_TTL = 2592000  # 30 days in seconds

_redis_client: Optional[Redis] = None
redis_lock = threading.Lock()

def get_redis_client() -> Redis:
    """
    Initializes and returns a Redis client (Thread-safe Singleton).
    """
    global _redis_client
    if _redis_client is not None:
        return _redis_client
    
    with redis_lock:
        # Double-check inside lock
        if _redis_client is not None:
            return _redis_client
        
        try:
            # 1. Initialize temporary client
            temp_client = Redis.from_url(REDIS_URL, decode_responses=False)
            # 2. Ping to verify connection
            temp_client.ping()
            # 3. Only assign to global if ping succeeds
            _redis_client = temp_client
            logger.info(f"Connected to Redis at {REDIS_URL}")
        except Exception as e:
            logger.error(f"Failed to connect to Redis: {e}")
            raise e
    return _redis_client

def close_redis_client():
    """
    Closes the Redis connection pool.
    """
    global _redis_client
    if _redis_client:
        _redis_client.close()
        _redis_client = None
        logger.info("Redis connection closed.")

def init_semantic_cache_index():
    """
    Initializes the RediSearch index for semantic cache if it doesn't exist.
    Safe wrapper to prevent application crash during startup.
    """
    try:
        r = get_redis_client()
        
        # 1. Check if index exists
        try:
            r.ft(INDEX_NAME).info()
            # If successful, index exists -> Return early
            return
        except Exception as e:
            if "unknown index" not in str(e).lower():
                # Real error happened (likely Redis connection issues) -> Log and Return
                logger.error(f"Error checking index: {e}")
                return
        
        # 2. Only attempts creation if "unknown index" was caught
        try:
            logger.info(f"Creating index {INDEX_NAME}...")
            schema = (
                TextField("question"),
                TextField("answer"),
                VectorField(
                    "embedding",
                    "FLAT",
                    {
                        "TYPE": "FLOAT32",
                        "DIM": VECTOR_DIM,
                        "DISTANCE_METRIC": "COSINE",
                    },
                ),
            )
            definition = IndexDefinition(prefix=[DOC_PREFIX], index_type=IndexType.HASH)
            r.ft(INDEX_NAME).create_index(fields=schema, definition=definition)
            logger.info("Semantic cache index created.")
        except Exception as e:
            logger.error(f"Failed to create index: {e}")
            
    except Exception as e:
        # Final safety net to avoid crashing the whole FastAPI app
        logger.warning(f"Graceful degradation: Semantic cache index init failed: {e}")

def check_semantic_cache(query_vector: List[float], similarity_threshold: float = 0.90) -> Optional[dict[str, Any]]:
    """
    Checks the semantic cache for a similar question.
    """
    # Convert threshold to distance
    max_distance = 1.0 - similarity_threshold
    
    try:
        r = get_redis_client()
        
        # Prepare vector as bytes
        query_vector_np = np.array(query_vector, dtype=np.float32).tobytes()
        
        # Prepare KNN query
        base_query = "*=>[KNN 1 @embedding $vector AS vector_score]"
        query = (
            Query(base_query)
            .sort_by("vector_score")
            .return_fields("answer", "citations", "vector_score")
            .dialect(2)
        )
        
        query_params = {"vector": query_vector_np}
        
        results = r.ft(INDEX_NAME).search(query, query_params)
        
        if results.docs:
            doc = results.docs[0]
            distance = float(doc.vector_score)
            
            if distance <= max_distance:
                logger.info(f"Semantic Cache Hit! Distance: {distance:.4f}")
                answer = doc.answer.decode("utf-8") if isinstance(doc.answer, bytes) else doc.answer
                raw_citations = getattr(doc, "citations", b"[]")
                if isinstance(raw_citations, bytes):
                    raw_citations = raw_citations.decode("utf-8", errors="ignore")
                citations: list[dict[str, Any]] = []
                try:
                    parsed = json.loads(raw_citations or "[]")
                    if isinstance(parsed, list):
                        citations = [c for c in parsed if isinstance(c, dict)]
                except Exception:
                    citations = []
                return {"answer": answer, "citations": citations}
            
        return None
        
    except Exception as e:
        logger.error(f"Semantic Cache Search Internal Error: {e}")
        return None

def set_semantic_cache(
    question_text: str,
    answer_text: str,
    query_vector: List[float],
    citations: list[dict[str, Any]] | None = None,
) -> bool:
    """
    Saves a question-answer pair and its embedding to Redis cache with TTL.
    """
    try:
        r = get_redis_client()
        
        question_hash = hashlib.sha256(question_text.encode("utf-8")).hexdigest()
        key = f"{DOC_PREFIX}{question_hash}"
        
        # Convert vector to bytes
        embedding_bytes = np.array(query_vector, dtype=np.float32).tobytes()
        
        # Store in hash
        r.hset(
            key,
            mapping={
                "question": question_text,
                "answer": answer_text,
                "citations": json.dumps(citations or [], ensure_ascii=False),
                "embedding": embedding_bytes,
            }
        )
        # Set TTL to 30 days to avoid OOM
        r.expire(key, CACHE_TTL)
        return True
        
    except Exception as e:
        logger.error(f"Failed to set semantic cache: {e}")
        return False

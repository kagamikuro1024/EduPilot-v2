"""
Tool definitions for the agent.
Includes RAG (Knowledge) and SQL (Assignment) tools.
"""

import asyncio
import contextvars
import json
import logging
import re
import time
import psycopg2
from openai import AsyncOpenAI
from .config import (
    DATABASE_URL,
    OPENAI_API_KEY,
    OPENAI_MAX_RETRIES,
    OPENAI_TIMEOUT_SECONDS,
    RAG_MAX_RETRIEVAL_DISTANCE,
    RAG_RETRIEVAL_CANDIDATES,
    RAG_RERANK_LIMIT,
    RAG_FALLBACK_MAX_DISTANCE,
    RAG_RELATIVE_DISTANCE_RATIO,
    RAG_DEBUG_TOP_K,
    RAG_UTILITY_MODEL,
)
from .database.cache_repo import check_semantic_cache, set_semantic_cache
from .database.vector_repo import (
    fetch_regulation_neighbor_chunks,
    search_chunks_keyword_ilike,
    search_vectors,
)

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Shared OpenAI client (lazy singleton).
#
# Creating a new AsyncOpenAI per RAG call builds a fresh httpx connection pool
# each time. We memoize one client keyed on (client class, api key) so:
#   * production reuses a single pool with explicit timeout/retry bounds;
#   * tests that monkeypatch ``tools.AsyncOpenAI`` / ``tools.OPENAI_API_KEY``
#     still get their fake client (the cache key changes with the patch).
# ---------------------------------------------------------------------------
_openai_client_cache: dict = {"key": None, "client": None}


def _get_openai_client():
    cls = AsyncOpenAI  # late module-global lookups: respect monkeypatching
    api_key = OPENAI_API_KEY
    cache_key = (cls, api_key)
    if _openai_client_cache["key"] == cache_key and _openai_client_cache["client"] is not None:
        return _openai_client_cache["client"]
    try:
        client = cls(
            api_key=api_key,
            timeout=OPENAI_TIMEOUT_SECONDS,
            max_retries=OPENAI_MAX_RETRIES,
        )
    except TypeError:
        # Test doubles may not accept timeout/max_retries kwargs.
        client = cls(api_key=api_key)
    _openai_client_cache["key"] = cache_key
    _openai_client_cache["client"] = client
    return client


_LAST_RAG_CITATIONS: contextvars.ContextVar[list[dict]] = contextvars.ContextVar("last_rag_citations", default=[])
_LAST_RAG_RUNTIME: contextvars.ContextVar[dict] = contextvars.ContextVar(
    "last_rag_runtime",
    default={"cache_hit": False},
)
MAX_RETRIEVAL_DISTANCE = RAG_MAX_RETRIEVAL_DISTANCE

# Must match DB constraint / Java enum `DocumentType`
DOCUMENT_TYPE_COURSE_MATERIAL = "COURSE_MATERIAL"
DOCUMENT_TYPE_REGULATION = "REGULATION"

# Regulation PDFs are often OCR-derived and noisy. Keep their rerank prompt small
# so first useful answer does not wait on a huge LLM context.
REGULATION_RAG_TIMEOUT_SECONDS = 40.0
REGULATION_RERANK_LIMIT = 8
REGULATION_KEYWORD_TERMS_CAP = 12
REGULATION_KW_ONLY_MAX = 4
REGULATION_NEIGHBOR_WINDOW = 1
REGULATION_NEIGHBOR_MAX_SEEDS = 10
# Final answer synthesizer (replaces Layer-3 JSON rerank for REGULATION)
REGULATION_SYNTHESIS_CONTENT_CHARS = 12000
REGULATION_SYNTHESIS_TIMEOUT_SECONDS = 28.0
REGULATION_SYNTHESIS_MAX_COMPLETION_TOKENS = 2500

_VI_KW_STOPWORDS = frozenset(
    {
        "và",
        "hoặc",
        "với",
        "cho",
        "của",
        "trong",
        "trên",
        "không",
        "có",
        "là",
        "thì",
        "em",
        "sinh",
        "viên",
        "năm",
        "học",
        "khi",
        "nào",
        "mấy",
        "điều",
        "theo",
        "về",
        "nếu",
        "để",
        "một",
        "hai",
        "ba",
        "bốn",
        "năm",
        "sáu",
        "các",
        "những",
        "được",
        "bị",
        "hay",
        "ở",
        "tại",
        "vì",
        "do",
        "như",
        "sẽ",
        "đã",
        "còn",
        "thế",
        "này",
        "đó",
        "bao",
        "lâu",
        "gì",
        "sao",
        "quy",
        "chế",
        "nhờ",
        "bạn",
        "môn",
        "đăng",
        "nộp",
        "xin",
        "muốn",
        "ứng",
        "vụ",
        "tập",
        "quả",
    }
)

_REGULATION_NEIGHBOR_TRIGGER_RE = re.compile(
    r"buộc\s+thôi\s+học|cảnh\s*báo|bảo\s*lưu|đăng\s*ký\s*học|tiếng\s*anh|thi\s*hộ|kỉ\s*luật|kỷ\s*luật",
    re.IGNORECASE,
)


def _regulation_neighbor_expansion_enabled(query: str, anchors: list[str]) -> bool:
    """Expand ±chunk neighbors only for procedural / multi-bullet regulation topics."""
    if _REGULATION_NEIGHBOR_TRIGGER_RE.search(query or ""):
        return True
    blob = " ".join(anchors or "")
    return bool(_REGULATION_NEIGHBOR_TRIGGER_RE.search(blob))

_REGULATION_PHRASE_PATTERNS: tuple[tuple[str, int], ...] = (
    (r"bảo\s+lưu\s+kết\s+quả\s+học\s+tập", re.IGNORECASE),
    (r"nghỉ\s+học\s+tạm\s+thời", re.IGNORECASE),
    (r"cảnh\s+báo\s+học\s+vụ", re.IGNORECASE),
    (r"buộc\s+thôi\s+học", re.IGNORECASE),
    (r"thi\s+hộ", re.IGNORECASE),
    (r"đăng\s+ký\s+học", re.IGNORECASE),
    (r"đăng\s+ký\s+môn", re.IGNORECASE),
    (r"đăng\s+ký\s+.*?học\s+kỳ", re.IGNORECASE),
    (r"tiếng\s+anh", re.IGNORECASE),
    (r"năng\s+lực\s+tiếng\s+anh", re.IGNORECASE),
    (r"thời\s+gian\s+đào\s+tạo", re.IGNORECASE),
    (r"thời\s+gian\s+học\s+tối\s+đa", re.IGNORECASE),
    (r"nhập\s+học\s+năm\s*\d{4}", re.IGNORECASE),
    (r"\bK\d{2}\b", re.IGNORECASE),
    (r"điều\s+chỉnh\s+đăng\s+ký", re.IGNORECASE),
    (r"hết\s+hạn\s+bảo\s+lưu", re.IGNORECASE),
    (r"kỉ\s+luật", re.IGNORECASE),
    (r"kỷ\s+luật", re.IGNORECASE),
)


def consume_last_rag_citations() -> list[dict]:
    citations = _LAST_RAG_CITATIONS.get()
    _LAST_RAG_CITATIONS.set([])
    return citations


def consume_last_rag_runtime() -> dict:
    runtime = _LAST_RAG_RUNTIME.get()
    _LAST_RAG_RUNTIME.set({"cache_hit": False})
    if not isinstance(runtime, dict):
        return {"cache_hit": False}
    return runtime


def _normalize_text(value: object) -> str:
    return str(value or "").strip().lower()


# --- Course-material query normalization (synonym groups for stable RAG / cache) ---

_COURSE_COMPONENT_SYNONYMS_RE = re.compile(
    r"(?:thành\s*phần\s*(?:chính|cốt\s*lõi)|components?|core\s+components?|main\s+components?)",
    re.IGNORECASE,
)

# When the student asks "main/core components" of a topic, map to one canonical query string
# for embeddings + semantic cache so "chính" vs "cốt lõi" retrieve identically.
_TOPIC_CANONICAL_COMPONENT: tuple[tuple[re.Pattern[str], str], ...] = (
    (re.compile(r"rabbit\s*mq", re.IGNORECASE), "rabbitmq các thành phần cốt lõi"),
    (re.compile(r"\bkafka\b", re.IGNORECASE), "kafka các thành phần cốt lõi"),
    (re.compile(r"\bredis\b", re.IGNORECASE), "redis các thành phần cốt lõi"),
)


def normalize_course_rag_query(query: str) -> tuple[str, dict[str, object]]:
    """
    Return (effective_query, meta). When meta['used_canonical'] is True, embedding/cache
    use the canonical string so equivalent phrasings hit the same retrieval path.
    """
    raw = (query or "").strip()
    meta: dict[str, object] = {"used_canonical": False, "original": raw}
    if not raw:
        return raw, meta
    if not _COURSE_COMPONENT_SYNONYMS_RE.search(raw):
        return raw, meta
    for topic_re, canonical in _TOPIC_CANONICAL_COMPONENT:
        if topic_re.search(raw):
            meta["used_canonical"] = True
            meta["canonical"] = canonical
            return canonical, meta
    return raw, meta


_REWRITE_SYSTEM = (
    "Bạn là trợ lý viết lại câu hỏi. Nhiệm vụ: chuyển câu hỏi sinh viên thành dạng "
    "chính thức dùng thuật ngữ quy chế đào tạo. Chỉ trả về câu đã viết lại, không giải thích.\n"
    "Ví dụ:\n"
    "- 'đuổi học khi nào' → 'điều kiện buộc thôi học'\n"
    "- 'nợ mấy tín chỉ thì cảnh cáo' → 'điều kiện cảnh báo học vụ về số tín chỉ'\n"
    "- 'xin nghỉ tạm' → 'thủ tục bảo lưu kết quả học tập'"
)

async def _rewrite_regulation_query(client: AsyncOpenAI, query: str) -> str:
    """Light LLM rewrite to map student slang -> formal regulation terms."""
    try:
        resp = await asyncio.wait_for(
            client.chat.completions.create(
                model=RAG_UTILITY_MODEL,
                messages=[
                    {"role": "system", "content": _REWRITE_SYSTEM},
                    {"role": "user", "content": query},
                ],
                max_completion_tokens=80,
                temperature=0,
            ),
            timeout=3.0,
        )
        rewritten = (resp.choices[0].message.content or "").strip()
        if rewritten and len(rewritten) > 5:
            logger.info("regulation_query_rewrite original=%r rewritten=%r", query, rewritten)
            return rewritten
    except Exception as e:
        logger.warning("Query rewrite failed, using original: %s", e)
    return query


def _course_query_is_component_list(query: str) -> bool:
    """True when the student asks for a list of parts/components (exhaustive extraction)."""
    return bool(query and _COURSE_COMPONENT_SYNONYMS_RE.search(query))


_COURSE_TOPIC_KEYWORDS = frozenset({"rabbitmq", "kafka", "redis", "saga"})


def _course_keyword_terms_from_hints(topic_hints: list[str]) -> list[str]:
    """Lexical terms for hybrid retrieval (short, high-precision)."""
    terms: list[str] = []
    for h in topic_hints or []:
        hl = (h or "").strip().lower()
        if hl in _COURSE_TOPIC_KEYWORDS:
            terms.append(hl)
    # De-dupe preserve order
    seen: set[str] = set()
    out: list[str] = []
    for t in terms:
        k = t.lower()
        if k in seen:
            continue
        seen.add(k)
        out.append(t)
    return out[:12]


def _rrf_merge(
    ranked_lists: list[list[dict]],
    max_total: int,
    k: int = 60,
) -> list[dict]:
    """
    Reciprocal Rank Fusion: merge multiple ranked lists into one.
    Each chunk gets score = sum(1 / (k + rank)) across all lists it appears in.
    """
    scores: dict[str, float] = {}
    chunk_map: dict[str, dict] = {}

    for ranked_list in ranked_lists:
        for rank, chunk in enumerate(ranked_list, start=1):
            cid = str(chunk.get("chunk_id", "") or "")
            if not cid:
                continue
            scores[cid] = scores.get(cid, 0.0) + 1.0 / (k + rank)
            if cid not in chunk_map:
                chunk_map[cid] = chunk

    sorted_ids = sorted(scores, key=lambda cid: scores[cid], reverse=True)
    return [chunk_map[cid] for cid in sorted_ids[:max_total]]


def _merge_course_vector_and_keyword(
    vector_chunks: list[dict],
    kw_chunks: list[dict],
    max_total: int,
    **_kwargs,
) -> list[dict]:
    """RRF merge for course materials: vector + keyword results."""
    return _rrf_merge([vector_chunks, kw_chunks], max_total=max_total)


def _extract_document_hints(query: str) -> list[str]:
    lowered = query.lower()
    hints: list[str] = []
    # File-like mentions: "something.pdf"
    hints.extend(re.findall(r"([a-z0-9_\-\s]+\.pdf)", lowered))
    # Domain hint for current bug report and similar usages.
    if "saga" in lowered:
        hints.append("saga")
    # Topic-focused retrieval: prefer chunks whose filename/source matches the technology name.
    if re.search(r"rabbit\s*mq", lowered):
        hints.append("rabbitmq")
    if re.search(r"\bkafka\b", lowered):
        hints.append("kafka")
    if re.search(r"\bredis\b", lowered):
        hints.append("redis")
    return [h.strip() for h in hints if h.strip()]


def _filter_retrieved_chunks(query: str, chunks: list[dict]) -> list[dict]:
    if not chunks:
        return []
    hints = _extract_document_hints(query)
    filtered: list[dict] = []
    for c in chunks:
        distance = c.get("distance")
        if isinstance(distance, (int, float)) and distance > MAX_RETRIEVAL_DISTANCE:
            continue
        if hints:
            title = _normalize_text(c.get("original_filename") or c.get("file_name"))
            source = _normalize_text(c.get("source_uri"))
            if not any(h in title or h in source for h in hints):
                continue
        filtered.append(c)
    return filtered


def _filter_with_diagnostics(query: str, chunks: list[dict]) -> tuple[list[dict], dict]:
    if not chunks:
        return [], {
            "raw_count": 0,
            "distance_count": 0,
            "final_count": 0,
            "distance_filtered_out": 0,
            "hint_filtered_out": 0,
            "hint_fallback_used": False,
            "hints": [],
            "min_distance": None,
            "max_distance": None,
        }

    hints = _extract_document_hints(query)
    min_distance = None
    max_distance = None
    distance_filtered: list[dict] = []
    distance_filtered_out = 0

    for chunk in chunks:
        distance = chunk.get("distance")
        if isinstance(distance, (int, float)):
            min_distance = distance if min_distance is None else min(min_distance, distance)
            max_distance = distance if max_distance is None else max(max_distance, distance)
        distance_filtered.append(chunk)

    # Relative filtering: keep chunks within ratio of best distance
    if min_distance is not None and min_distance > 0:
        dynamic_cutoff = min(
            min_distance * RAG_RELATIVE_DISTANCE_RATIO,
            RAG_FALLBACK_MAX_DISTANCE,  # absolute ceiling
        )
    else:
        dynamic_cutoff = MAX_RETRIEVAL_DISTANCE

    final_distance: list[dict] = []
    for chunk in distance_filtered:
        d = chunk.get("distance")
        if isinstance(d, (int, float)) and d > dynamic_cutoff:
            distance_filtered_out += 1
            continue
        final_distance.append(chunk)
    distance_filtered = final_distance

    if not hints:
        return distance_filtered, {
            "raw_count": len(chunks),
            "distance_count": len(distance_filtered),
            "final_count": len(distance_filtered),
            "distance_filtered_out": distance_filtered_out,
            "hint_filtered_out": 0,
            "hint_fallback_used": False,
            "hints": [],
            "min_distance": min_distance,
            "max_distance": max_distance,
        }

    hinted_chunks: list[dict] = []
    hint_filtered_out = 0
    for chunk in distance_filtered:
        title = _normalize_text(chunk.get("original_filename") or chunk.get("file_name"))
        source = _normalize_text(chunk.get("source_uri"))
        if any(h in title or h in source for h in hints):
            hinted_chunks.append(chunk)
        else:
            hint_filtered_out += 1

    hint_fallback_used = False
    final_chunks = hinted_chunks
    if distance_filtered and not hinted_chunks:
        hint_fallback_used = True
        final_chunks = distance_filtered

    return final_chunks, {
        "raw_count": len(chunks),
        "distance_count": len(distance_filtered),
        "final_count": len(final_chunks),
        "distance_filtered_out": distance_filtered_out,
        "hint_filtered_out": hint_filtered_out,
        "hint_fallback_used": hint_fallback_used,
        "hints": hints,
        "min_distance": min_distance,
        "max_distance": max_distance,
    }


def _apply_adaptive_distance_fallback(chunks: list[dict], raw_chunks: list[dict], stats: dict) -> tuple[list[dict], dict]:
    if chunks or not raw_chunks:
        stats["adaptive_fallback_used"] = False
        return chunks, stats

    min_distance = stats.get("min_distance")
    if isinstance(min_distance, (int, float)) and min_distance <= RAG_FALLBACK_MAX_DISTANCE:
        stats["adaptive_fallback_used"] = True
        stats["adaptive_fallback_reason"] = "min_distance_within_fallback_threshold"
        return list(raw_chunks), stats

    stats["adaptive_fallback_used"] = False
    stats["adaptive_fallback_reason"] = "min_distance_too_far_or_missing"
    return chunks, stats


def _build_candidate_diagnostics(chunks: list[dict], top_k: int) -> list[dict]:
    diagnostics: list[dict] = []
    safe_top_k = max(0, top_k)
    for chunk in chunks[:safe_top_k]:
        snippet = str(chunk.get("snippet") or chunk.get("content") or "").replace("\n", " ").strip()
        diagnostics.append(
            {
                "distance": chunk.get("distance"),
                "document_id": str(chunk.get("document_id", "") or ""),
                "file": str(chunk.get("original_filename", "") or chunk.get("file_name", "") or ""),
                "page_number": int(chunk.get("page_number", 0) or 0),
                "snippet": snippet[:160],
            }
        )
    return diagnostics


def _truncate_for_rerank(content: object, max_chars: int | None) -> str:
    text = str(content or "").strip()
    if max_chars is None or max_chars <= 0 or len(text) <= max_chars:
        return text
    return text[:max_chars].rstrip() + "\n...[truncated]"


def _regulation_final_synthesis_system_prompt(num_chunks: int, scope_applicability: bool) -> str:
    scope_hint = (
        "Câu hỏi về phạm vi/đối tượng: ưu tiên các đoạn **Phạm vi áp dụng**, **Đối tượng**, **Mở đầu**."
        if scope_applicability
        else "Ưu tiên điều khoản liệt kê (Điều, Khoản, bảng, gạch đầu dòng) hơn phần giới thiệu chung — "
        "trừ khi sinh viên hỏi rõ về phạm vi/đối tượng áp dụng."
    )
    return (
        "Bạn đọc các đoạn trích từ quy chế đào tạo/quy định đã tải lên hệ thống.\n\n"
        "Nhiệm vụ của bạn là trả lời các câu hỏi về quy chế. LUẬT QUAN TRỌNG NHẤT: Khi trích dẫn các điều kiện, "
        "quy định, mốc thời gian, tỷ lệ phần trăm (%), số lượng tín chỉ, số năm... BẠN PHẢI CHÉP Y NGUYÊN VĂN "
        "BẢN GỐC, KHÔNG ĐƯỢC TÓM TẮT BẰNG TỪ ĐỒNG NGHĨA. Nếu có nhiều gạch đầu dòng (a, b, c), phải liệt kê đủ "
        "không sót một chữ.\n\n"
        f"Có {num_chunks} đoạn nguồn (CHUNK 1..{num_chunks}) đã được sắp xếp theo độ liên quan từ tìm kiếm. "
        f"{scope_hint}\n"
        "Khi ánh xạ từ ngữ đời thường của sinh viên sang văn bản (ví dụ đuổi học → buộc thôi học), phần trích dẫn "
        "vẫn phải là **nguyên văn** đúng như trong CHUNK.\n\n"
        "Trả về **một** JSON hợp lệ duy nhất, không bọc markdown, dạng:\n"
        '{"answer":"<trả lời bằng Markdown tiếng Việt>","cited_chunk_indices":[1,2]}\n'
        "Trong đó cited_chunk_indices là danh sách số thứ tự CHUNK (bắt đầu từ 1) mà bạn đã dựa vào để trả lời. "
        'Nếu không đủ căn cứ: {"answer":"","cited_chunk_indices":[]}'
    )


def _regulation_synthesis_user_message(
    query: str,
    regulation_anchors: list[str],
    scope_applicability: bool,
) -> str:
    parts: list[str] = [query.strip()]
    if regulation_anchors:
        parts.append(
            "\n[Gợi ý truy vấn — ưu tiên đoạn khớp các ý sau]\n"
            + "\n".join(f"- {a}" for a in regulation_anchors)
        )
    if scope_applicability:
        parts.append("\n[Lưu ý: Ưu tiên đoạn có Đối tượng / Phạm vi áp dụng / Mở đầu.]")
    parts.append(
        '\nTrả lời bằng JSON theo hệ thống; trường "answer" ghi nội dung đầy đủ, có trích nguyên văn các điều kiện/số liệu.'
    )
    return "".join(parts)


def _citation_dict_from_regulation_chunk(chunk: dict) -> dict:
    display_name = str(chunk.get("original_filename", "") or chunk.get("file_name", "") or "Unknown source")
    page_number = int(chunk.get("page_number", 0) or 0)
    quote_src = str(chunk.get("snippet") or chunk.get("content") or "").strip()
    snippet = quote_src
    return {
        "document_id": str(chunk.get("document_id", "")),
        "source_uri": str(chunk.get("source_uri", "") or ""),
        "source_file": display_name,
        "page_number": page_number,
        "snippet": snippet,
        "chunk_id": str(chunk.get("chunk_id", "")),
        "chunk_index": int(chunk.get("chunk_index", 0) or 0),
    }


def _citations_from_regulation_chunk_indices(
    chunks: list[dict],
    one_based_indices: object,
    *,
    fallback_all: bool = False,
) -> list[dict]:
    citations: list[dict] = []
    seen_ids: set[str] = set()
    indices_list: list[int] = []
    if isinstance(one_based_indices, list):
        for x in one_based_indices:
            try:
                indices_list.append(int(x))
            except (TypeError, ValueError):
                continue
    if not indices_list and fallback_all and chunks:
        indices_list = list(range(1, len(chunks) + 1))
    for k in indices_list:
        i = k - 1
        if i < 0 or i >= len(chunks):
            continue
        ch = chunks[i]
        cid = str(ch.get("chunk_id", "") or "")
        if cid and cid in seen_ids:
            continue
        if cid:
            seen_ids.add(cid)
        citations.append(_citation_dict_from_regulation_chunk(ch))
    return citations


async def _synthesize_regulation_answer_from_chunks(
    client: AsyncOpenAI,
    query: str,
    chunks: list[dict],
    regulation_anchors: list[str],
    scope_applicability: bool,
    empty_msg: str,
) -> tuple[str, list[dict]]:
    """
    Final LLM pass for REGULATION: full Layer-2 chunks, no JSON rerank / fact extraction.
    """
    if not chunks:
        return empty_msg, []

    ctx_parts: list[str] = []
    for i, c in enumerate(chunks):
        content = _truncate_for_rerank(c.get("content"), REGULATION_SYNTHESIS_CONTENT_CHARS)
        ctx_parts.append(
            f"---\n"
            f"CHUNK {i + 1}\n"
            f"CHUNK_ID: {c.get('chunk_id', '')}\n"
            f"DOCUMENT_ID: {c.get('document_id', '')}\n"
            f"FILE: {c.get('original_filename', '') or c.get('file_name', '')}\n"
            f"PAGE_NUMBER: {c.get('page_number', 0)}\n"
            f"SOURCE_URI: {c.get('source_uri', '')}\n"
            f"ORIGINAL_CHUNK_INDEX: {c.get('chunk_index', 0)}\n"
            f"CONTENT: {content}\n"
        )
    context_text = "".join(ctx_parts)
    system_prompt = _regulation_final_synthesis_system_prompt(len(chunks), scope_applicability)
    user_content = (
        _regulation_synthesis_user_message(query, regulation_anchors, scope_applicability)
        + "\n\n--- NGUỒN ---\n"
        + context_text
    )

    started = time.perf_counter()
    request = client.chat.completions.create(
        model=RAG_UTILITY_MODEL,
        messages=[
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": user_content},
        ],
        max_completion_tokens=REGULATION_SYNTHESIS_MAX_COMPLETION_TOKENS,
        temperature=0,
    )
    resp = await asyncio.wait_for(request, timeout=REGULATION_SYNTHESIS_TIMEOUT_SECONDS)
    raw = (resp.choices[0].message.content or "").strip()
    elapsed_ms = int((time.perf_counter() - started) * 1000)

    if not raw:
        fb_a, fb_c = _build_extractive_answer_from_chunks(
            chunks,
            max_items=min(5, len(chunks) or 5),
        )
        if fb_c:
            logger.info(
                "regulation_rag_mode=synthesis_blank_response extractive_fallback fact_count=%s elapsed_ms=%s",
                len(fb_c),
                elapsed_ms,
            )
            return fb_a, fb_c
        return empty_msg, []

    cleaned = _prepare_rerank_json_string(raw)
    answer_text = ""
    cited_indices: object = None
    try:
        obj = json.loads(cleaned) if cleaned else {}
        if isinstance(obj, dict):
            answer_text = str(obj.get("answer", "") or "").strip()
            cited_indices = obj.get("cited_chunk_indices")
            if cited_indices is None:
                cited_indices = obj.get("cited_chunks")
    except json.JSONDecodeError:
        answer_text = ""
    if not answer_text:
        fb_a, fb_c = _build_extractive_answer_from_chunks(
            chunks,
            max_items=min(5, len(chunks) or 5),
        )
        if fb_c:
            logger.info(
                "regulation_rag_mode=synthesis_json_fallback extractive_fallback fact_count=%s elapsed_ms=%s",
                len(fb_c),
                elapsed_ms,
            )
            return fb_a, fb_c
        return empty_msg, []

    citations = _citations_from_regulation_chunk_indices(
        chunks,
        cited_indices,
        fallback_all=not (isinstance(cited_indices, list) and len(cited_indices) > 0),
    )
    if not citations:
        citations = _citations_from_regulation_chunk_indices(chunks, list(range(1, len(chunks) + 1)))

    logger.info(
        "regulation_rag_mode=llm_synthesis elapsed_ms=%s citation_count=%s",
        elapsed_ms,
        len(citations),
    )
    if citations:
        source_labels = []
        for c in citations:
            label = f"[{c.get('source_file', '')}, p.{c.get('page_number', 0)}]"
            if label not in source_labels:
                source_labels.append(label)
        answer_text += "\n\nNguồn tham khảo: " + ", ".join(source_labels)

    return answer_text, citations


def _build_extractive_answer_from_chunks(chunks: list[dict], max_items: int = 3) -> tuple[str, list[dict]]:
    """
    Fast fallback when regulation reranking is slow.
    Keeps the response grounded and preserves citations instead of timing out.
    """
    answer_lines: list[str] = []
    citations: list[dict] = []

    for idx_order, chunk in enumerate(chunks[:max_items], start=1):
        display_name = str(chunk.get("original_filename", "") or chunk.get("file_name", "") or "Unknown source")
        page_number = int(chunk.get("page_number", 0) or 0)
        snippet = str(chunk.get("snippet") or chunk.get("content") or "").replace("\n", " ").strip()
        if not snippet:
            continue
        answer_lines.append(f"{idx_order}. {snippet} [{display_name}, p.{page_number}]")
        citations.append(
            {
                "document_id": str(chunk.get("document_id", "")),
                "source_uri": str(chunk.get("source_uri", "") or ""),
                "source_file": display_name,
                "page_number": page_number,
                "snippet": snippet,
                "chunk_id": str(chunk.get("chunk_id", "")),
                "chunk_index": int(chunk.get("chunk_index", 0) or 0),
            }
        )

    if not citations:
        return "", []

    answer = (
        "Dựa trên các điều khoản liên quan trong quy chế, các ý chính như sau:\n"
        + "\n".join(answer_lines)
    )
    return answer, citations


def _dedupe_regulation_chunks(chunks: list[dict], max_items: int) -> list[dict]:
    """
    Remove near-duplicate regulation chunks to make answers and citations more stable.
    Dedup key favors same source/page/content snippet.
    """
    deduped: list[dict] = []
    seen: set[tuple[str, int, str]] = set()
    for chunk in chunks:
        snippet = str(chunk.get("snippet") or chunk.get("content") or "").replace("\n", " ").strip().lower()
        normalized_snippet = re.sub(r"\s+", " ", snippet)[:120]
        key = (
            str(chunk.get("document_id", "") or ""),
            int(chunk.get("page_number", 0) or 0),
            normalized_snippet,
        )
        if key in seen:
            continue
        seen.add(key)
        deduped.append(chunk)
        if len(deduped) >= max_items:
            break
    return deduped


def _strip_markdown_json_fences(raw: str) -> str:
    """Remove markdown fences so JSON parsing is resilient."""
    text = (raw or "").strip()
    text = re.sub(r"^\s*```(?:json)?\s*", "", text, flags=re.IGNORECASE)
    text = re.sub(r"\s*```\s*$", "", text, flags=re.IGNORECASE)
    return text.strip()


def _extract_json_object(text: str) -> str | None:
    """
    Find the first top-level JSON object in ``text`` using brace depth, respecting
    string literals and escapes. Returns None if no balanced object is found.
    """
    s = (text or "").strip()
    if not s:
        return None
    start = s.find("{")
    if start < 0:
        return None
    depth = 0
    i = start
    in_str = False
    esc = False
    n = len(s)
    while i < n:
        c = s[i]
        if in_str:
            if esc:
                esc = False
            elif c == "\\":
                esc = True
            elif c == '"':
                in_str = False
        else:
            if c == '"':
                in_str = True
            elif c == "{":
                depth += 1
            elif c == "}":
                depth -= 1
                if depth == 0:
                    return s[start : i + 1]
        i += 1
    return None


def _prepare_rerank_json_string(raw: str) -> str:
    """
    Strip markdown fences and surrounding prose so ``json.loads`` can parse Layer 3 output.
    Tries: fenced ```json``` block, then first balanced ``{...}`` in the text.
    """
    text = (raw or "").strip()
    if not text:
        return ""
    m = re.search(r"```(?:json)?\s*([\s\S]*?)\s*```", text, re.IGNORECASE)
    if m:
        inner = m.group(1).strip()
        jobj = _extract_json_object(inner)
        if jobj:
            return jobj
        if inner:
            return inner
    text = _strip_markdown_json_fences(text)
    jobj = _extract_json_object(text)
    if jobj:
        return jobj
    return text.strip()


def _extract_vi_query_anchors(query: str) -> list[str]:
    """
    Surface Vietnamese / academic-policy cues from the user question so the reranker
    can prefer chunks that match cohort (K21/K22), year level, intake year, etc.
    Adds synonym mappings so student slang maps toward legal terms in retrieval focus.
    """
    if not (query or "").strip():
        return []
    text = query.strip()
    found: list[str] = []
    seen: set[str] = set()

    def _add_phrase(phrase: str) -> None:
        p = (phrase or "").strip()
        if not p:
            return
        key = p.lower()
        if key in seen:
            return
        seen.add(key)
        found.append(p)

    pattern_flags: list[tuple[str, int]] = [
        (r"\bK\d{2}\b", re.IGNORECASE),
        (r"khóa\s*(?:K)?\d{2}", re.IGNORECASE),
        (r"nhập\s*học\s*năm\s*\d{4}", re.IGNORECASE),
        (r"năm\s*thứ\s*(?:nhất|một|hai|ba|bốn|năm|sáu|bảy|tám)", re.IGNORECASE),
        (r"\btân\s*sinh\s*viên\b", re.IGNORECASE),
        (r"đợt\s*điều\s*chỉnh[^.;\n]{0,50}", re.IGNORECASE),
        (r"(?:cảnh\s*báo|bảo\s*lưu|đuổi\s*học|thôi\s*học|kỷ\s*luật|kỉ\s*luật)\s*[^.;\n]{0,20}", re.IGNORECASE),
    ]
    for pat, flags in pattern_flags:
        for m in re.finditer(pat, text, flags):
            s = m.group(0).strip()
            if not s:
                continue
            key = s.lower()
            if key in seen:
                continue
            seen.add(key)
            found.append(s)

    low = text.lower()
    if re.search(r"đuổi\s*học", low):
        _add_phrase("buộc thôi học")
    if re.search(r"thi\s*hộ", low):
        for extra in ("kỉ luật", "buộc thôi học", "thi hộ"):
            _add_phrase(extra)

    return found


_SCOPE_APPLICABILITY_RE = re.compile(
    r"(?:áp\s*dụng|đối\s*tượng|phạm\s*vi|sinh\s*viên\s*nào|áp\s*dụng\s*cho)",
    re.IGNORECASE,
)


def _regulation_query_is_scope_applicability(query: str) -> bool:
    """True when the student asks who/what the regulation applies to (scope / applicability)."""
    q = (query or "").strip()
    return bool(q and _SCOPE_APPLICABILITY_RE.search(q))


def _regulation_keyword_search_terms(query: str, anchors: list[str]) -> list[str]:
    """
    Build high-precision substring list for lexical retrieval.
    Avoids generic tokens (học, viên, năm, …) that match almost every chunk.
    """
    terms: list[str] = []
    seen: set[str] = set()

    def add(term: str) -> None:
        t = (term or "").strip()
        if len(t) < 3:
            return
        k = t.lower()
        if k in seen:
            return
        if len(t) <= 5 and k in _VI_KW_STOPWORDS:
            return
        seen.add(k)
        terms.append(t)

    q = query or ""
    for pat, flags in _REGULATION_PHRASE_PATTERNS:
        for m in re.finditer(pat, q, flags):
            add(m.group(0).strip())

    for a in anchors or []:
        a = (a or "").strip()
        if len(a) >= 4:
            add(a)

    for m in re.finditer(r"[\wÀ-ỹ]{7,}", q):
        w = m.group(0)
        wl = w.lower()
        if wl in _VI_KW_STOPWORDS:
            continue
        add(w)
        if len(terms) >= REGULATION_KEYWORD_TERMS_CAP:
            break

    return terms[:REGULATION_KEYWORD_TERMS_CAP]


def _merge_regulation_vector_and_keyword(
    vector_chunks: list[dict],
    kw_chunks: list[dict],
    max_total: int,
    **_kwargs,
) -> list[dict]:
    """RRF merge for regulation: vector + keyword results."""
    return _rrf_merge([vector_chunks, kw_chunks], max_total=max_total)


def _dedupe_chunks_merge_order(primary: list[dict], secondary: list[dict]) -> list[dict]:
    """Concatenate lists with stable dedupe by chunk_id (primary wins order)."""
    seen: set[str] = set()
    out: list[dict] = []
    for c in primary + secondary:
        cid = str(c.get("chunk_id", "") or "")
        if not cid or cid in seen:
            continue
        seen.add(cid)
        out.append(c)
    return out


_BULLET_FIRST_RE = re.compile(r"^\s*(?:[-*•]|\d+[.)])\s*\S")


def _split_regulation_facts(facts: list) -> list[dict]:
    """
    When the reranker collapses multiple bullets into one JSON fact, split into one fact per bullet.
    """
    out: list[dict] = []
    for m in facts or []:
        if not isinstance(m, dict):
            continue
        quote = str(m.get("quote", "") or "").strip()
        desc = str(m.get("description", "") or "").strip()
        text = quote if len(quote) >= len(desc) else desc
        if not text:
            out.append(m)
            continue
        lines = [ln.strip() for ln in text.splitlines() if ln.strip()]
        bullet_lines = [ln for ln in lines if _BULLET_FIRST_RE.match(ln)]
        if len(bullet_lines) >= 2:
            base_label = str(m.get("label", "") or "").strip()
            chunk_index = m.get("chunk_index")
            for i, bl in enumerate(bullet_lines):
                clean = re.sub(r"^\s*(?:[-*•]|\d+[.)])\s*", "", bl).strip()
                if not clean:
                    continue
                lab = f"{base_label} ({i + 1})" if base_label else ""
                out.append(
                    {
                        "label": lab,
                        "description": clean,
                        "chunk_index": chunk_index,
                        "quote": clean,
                    }
                )
        else:
            out.append(m)
    return out


def _fallback_extract_facts_from_text(raw: str) -> list[dict]:
    """Fallback extractor when reranker JSON is malformed."""
    cleaned = _strip_markdown_json_fences(raw)
    facts: list[dict] = []

    quote_matches = re.findall(r'"([^"\n]{8,280})"', cleaned)
    for quote in quote_matches[:5]:
        facts.append({"label": "", "description": quote.strip(), "chunk_index": 1, "quote": quote.strip()})

    if facts:
        return facts

    bullet_lines = re.findall(r"^\s*(?:\d+[.)]|[-*])\s+(.{8,280})$", cleaned, flags=re.MULTILINE)
    for line in bullet_lines[:5]:
        text = line.strip()
        facts.append({"label": "", "description": text, "chunk_index": 1, "quote": text})

    if facts:
        return facts

    if cleaned:
        fallback = cleaned[:280].strip()
        if fallback:
            facts.append({"label": "", "description": fallback, "chunk_index": 1, "quote": fallback})

    return facts

# --- Assignment Agent Tools (SQL) ---

def _extract_student_code(text: str) -> str | None:
    match = re.search(r"\b(?:mssv|student\s*id)\s*[:#-]?\s*(\d{8,12})\b", text, re.IGNORECASE)
    if match:
        return match.group(1)
    return None

def _run_assignment_query(assignment_name: str, student_code: str | None) -> str:

    conn = psycopg2.connect(DATABASE_URL)
    try:
        cur = conn.cursor()
        cur.execute(
            """
            SELECT a.id, a.title, a.due_date, a.late_penalty_rule
            FROM assignments a
            WHERE a.title ILIKE %s
            ORDER BY a.due_date NULLS LAST
            LIMIT 1
            """,
            ("%" + assignment_name + "%",),
        )
        assignment_row = cur.fetchone()
        if not assignment_row:
            return f"Khong tim thay thong tin bai tap '{assignment_name}' trong he thong. Vui long doi chieu voi LMS de dam bao chinh xac tuyet doi."

        assignment_id, assignment_title, due_date, late_penalty_rule = assignment_row
        base = (
            f"Han chot cua '{assignment_title}' la {due_date}. "
            f"Quy dinh nop muon: {late_penalty_rule or 'Khong co thong tin'}."
        )

        if not student_code:
            return base + " Vui long doi chieu voi LMS de dam bao chinh xac tuyet doi."

        cur.execute(
            """
            SELECT s.status, s.score, s.updated_at
            FROM submissions s
            JOIN users u ON u.id = s.user_id
            WHERE s.assignment_id = %s AND u.student_code = %s
            ORDER BY s.updated_at DESC NULLS LAST
            LIMIT 1
            """,
            (assignment_id, student_code),
        )
        submission_row = cur.fetchone()
        if not submission_row:
            return (
                base
                + f" Chua tim thay submission cua MSSV {student_code} cho bai nay."
                + " Vui long doi chieu voi LMS de dam bao chinh xac tuyet doi."
            )

        status, score, updated_at = submission_row
        return (
            base
            + f" Trang thai nop bai cua MSSV {student_code}: {status}, diem hien tai: {score}, cap nhat luc: {updated_at}."
            + " Vui long doi chieu voi LMS de dam bao chinh xac tuyet doi."
        )
    finally:
        conn.close()

async def check_assignment_deadline(assignment_name: str, student_code: str = "") -> str:
    """Query assignment deadline/rules and optional submission status from the database."""
    if not DATABASE_URL:
        return "Cau hinh ket noi co so du lieu chua san sang. Vui long doi chieu voi LMS de dam bao chinh xac tuyet doi."
    
    try:
        inferred_student_code = student_code.strip() or _extract_student_code(assignment_name) or None
        cleaned_name = re.sub(r"\b(?:mssv|student\s*id)\s*[:#-]?\s*\d{8,12}\b", "", assignment_name, flags=re.IGNORECASE).strip()
        safe_name = cleaned_name or assignment_name
        return await asyncio.to_thread(_run_assignment_query, safe_name, inferred_student_code)
    except Exception as e:
        logger.error(f"SQL Execution Error: {e}")
        return "Loi he thong khi truy van thong tin bai tap. Vui long doi chieu voi LMS de dam bao chinh xac tuyet doi."

def _run_get_assignments(days_limit: int = None) -> str:
    
    if days_limit is not None and not isinstance(days_limit, int):
        raise TypeError("days_limit must be an integer")
        
    with psycopg2.connect(DATABASE_URL) as conn:
        with conn.cursor() as cur:
            query = "SELECT id, title, due_date FROM assignments"
            params = []
            if days_limit is not None:
                query += " WHERE due_date IS NOT NULL AND due_date >= NOW() AND due_date <= NOW() + INTERVAL '1 days' * %s"
                params.append(days_limit)
            
            query += " ORDER BY due_date ASC NULLS LAST"
            
            cur.execute(query, tuple(params))
            rows = cur.fetchall()
            
            if not rows:
                return json.dumps([], ensure_ascii=False)
                
            results = []
            for row in rows:
                results.append({
                    "id": row[0],
                    "title": row[1],
                    "due_date": str(row[2]) if row[2] else None,
                    # Keep contract-compatible field for older clients.
                    "created_at": None
                })
                
            return json.dumps(results, ensure_ascii=False)

async def get_assignments(days_limit: int = None) -> str:
    """Get assignment list with optional days_limit filter."""
    app_logger = logging.getLogger("aitrogiang_app")
    if not DATABASE_URL:
        return json.dumps({"error": "Cau hinh ket noi co so du lieu chua san sang."})
    
    try:
        result = await asyncio.to_thread(_run_get_assignments, days_limit)
        app_logger.info(
            "Tool call get_assignments success",
            extra={
                "event": "tool_call",
                "tool_name": "get_assignments",
                "params": {"days": days_limit},
                "status": "success"
            }
        )
        return result
    except Exception as e:
        logger.error(f"Error in get_assignments: {e}")
        app_logger.info(
            "Tool call get_assignments failed",
            extra={
                "event": "tool_call",
                "tool_name": "get_assignments",
                "params": {"days": days_limit},
                "status": "error",
                "error": str(e)
            }
        )
        return json.dumps({"error": f"Lỗi hệ thống: {str(e)}"})

# --- Knowledge Agent Tools (RAG) ---

async def query_course_materials(query: str) -> str:
    """
    Retrieve relevant information using a 3-layer pipeline:
    1. Semantic Cache (Redis)
    2. Vector Search (PostgreSQL + pgvector)
    3. LLM Reranking (GPT-4o-mini)
    """
    try:
        _LAST_RAG_CITATIONS.set([])
        _LAST_RAG_RUNTIME.set({"cache_hit": False})
        # Hard-timeout of 20 seconds as per TIP-003
        answer, citations = await asyncio.wait_for(
            _execute_rag_pipeline(query, document_type=DOCUMENT_TYPE_COURSE_MATERIAL),
            timeout=20.0,
        )
        _LAST_RAG_CITATIONS.set(citations)
        return answer
    except asyncio.TimeoutError:
        logger.error(f"RAG Pipeline Timeout for query: {query}")
        return "Hệ thống đang quá tải, vui lòng thử lại sau giây lát hoặc liên hệ TA."
    except Exception as e:
        logger.error(f"RAG Pipeline Error: {e}")
        return "Mình không tìm thấy thông tin chính xác trong tài liệu môn học, mình đã chuyển câu hỏi này cho TA để hỗ trợ bạn."


async def query_regulations(query: str) -> str:
    """
    RAG over documents tagged REGULATION (quy chế, quy định nhà trường).
    Uses retrieval (Layer 2) then a single answer-synthesizer LLM on full chunks — no JSON rerank layer.
    Skips semantic cache to avoid cross-talk with course-material cache.
    """
    try:
        _LAST_RAG_CITATIONS.set([])
        _LAST_RAG_RUNTIME.set({"cache_hit": False})
        answer, citations = await asyncio.wait_for(
            _execute_rag_pipeline(query, document_type=DOCUMENT_TYPE_REGULATION),
            timeout=REGULATION_RAG_TIMEOUT_SECONDS,
        )
        _LAST_RAG_CITATIONS.set(citations)
        return answer
    except asyncio.TimeoutError:
        logger.error(f"Regulation RAG timeout for query: {query}")
        return "Hệ thống đang quá tải, vui lòng thử lại sau giây lát hoặc liên hệ phòng đào tạo/TA."
    except Exception as e:
        logger.error(f"Regulation RAG error: {e}")
        return (
            "Mình không tìm thấy thông tin chính xác trong quy chế/quy định đã được tải lên, "
            "mình đã chuyển câu hỏi này để được hỗ trợ thêm."
        )


async def _execute_rag_pipeline(query: str, document_type: str = DOCUMENT_TYPE_COURSE_MATERIAL) -> tuple[str, list[dict]]:
    """Internal helper for the RAG pipeline; filters vectors by ``documents.document_type``."""
    if not OPENAI_API_KEY:
        logger.error("OPENAI_API_KEY is missing")
        return "Cấu hình AI chưa hoàn tất.", []

    client = _get_openai_client()

    # Canonical query for course materials: equivalent student phrasings share embeddings + cache.
    effective_query = query
    if document_type == DOCUMENT_TYPE_COURSE_MATERIAL:
        canonical, norm_meta = normalize_course_rag_query(query)
        effective_query = canonical
        if norm_meta.get("used_canonical"):
            logger.info(
                "course_rag_query_normalized original=%r effective=%r",
                norm_meta.get("original"),
                effective_query,
            )

    empty_course_msg = (
        "Mình không tìm thấy thông tin chính xác trong tài liệu môn học, "
        "mình đã chuyển câu hỏi này cho TA để hỗ trợ bạn."
    )
    empty_regulation_msg = (
        "Mình không tìm thấy thông tin chính xác trong quy chế/quy định đã được tải lên, "
        "mình đã chuyển câu hỏi này để được hỗ trợ thêm."
    )
    empty_msg = empty_regulation_msg if document_type == DOCUMENT_TYPE_REGULATION else empty_course_msg
    is_regulation = document_type == DOCUMENT_TYPE_REGULATION
    rerank_limit = min(RAG_RERANK_LIMIT, REGULATION_RERANK_LIMIT) if is_regulation else RAG_RERANK_LIMIT
    rerank_content_chars: int | None = None
    regulation_anchors: list[str] = []
    scope_applicability = False
    if is_regulation:
        regulation_anchors = _extract_vi_query_anchors(query)
        scope_applicability = _regulation_query_is_scope_applicability(query)
        effective_query = await _rewrite_regulation_query(client, query)

    # 0. Generate embedding for the query
    try:
        emb_resp = await client.embeddings.create(
            input=[effective_query],
            model="text-embedding-3-small"
        )
        query_vector = emb_resp.data[0].embedding
    except Exception as e:
        logger.error(f"Embedding error: {e}")
        raise

    # 1. LAYER 1: Semantic Cache (Hit check) — course materials only (isolate regulation retrieval)
    # The Redis client is synchronous; run it in a worker thread so a slow or
    # unreachable cache cannot block the event loop (and every other stream).
    cache_result = None
    if document_type == DOCUMENT_TYPE_COURSE_MATERIAL:
        try:
            cache_result = await asyncio.to_thread(check_semantic_cache, query_vector)
        except Exception as cache_exc:
            logger.warning("Semantic cache check failed (treated as miss): %s", cache_exc)
            cache_result = None
    if cache_result:
        answer = str(cache_result.get("answer", "")).strip()
        citations = cache_result.get("citations", [])
        if answer and isinstance(citations, list) and citations:
            logger.info("RAG Pipeline: Layer 1 (Cache) Hit with citations")
            valid_citations = [c for c in citations if isinstance(c, dict)]
            _LAST_RAG_RUNTIME.set({"cache_hit": True})
            return answer, valid_citations
        logger.info("RAG Pipeline: Layer 1 cache stale (missing citations), fallback to retrieval")

    # 2. LAYER 2: Retrieve (pgvector; regulations also merge keyword substring hits)
    candidate_limit = max(RAG_RETRIEVAL_CANDIDATES, rerank_limit)
    
    kw_terms: list[str] = []
    if is_regulation:
        kw_terms = _regulation_keyword_search_terms(query, regulation_anchors)
    elif document_type == DOCUMENT_TYPE_COURSE_MATERIAL:
        hint_tokens = _extract_document_hints(effective_query)
        kw_terms = _course_keyword_terms_from_hints(hint_tokens)

    # Parallelize vector search and keyword search
    retrieval_started_at = time.perf_counter()
    kw_raw: list[dict] = []
    if kw_terms:
        raw_vector, kw_raw = await asyncio.gather(
            search_vectors(query_vector, limit=candidate_limit, document_type=document_type),
            search_chunks_keyword_ilike(kw_terms, document_type, limit=candidate_limit),
        )
    else:
        raw_vector = await search_vectors(query_vector, limit=candidate_limit, document_type=document_type)
    retrieval_elapsed_ms = int((time.perf_counter() - retrieval_started_at) * 1000)

    raw_merged = raw_vector
    if is_regulation:
        raw_merged = _merge_regulation_vector_and_keyword(
            raw_vector,
            kw_raw,
            max_total=candidate_limit,
        )
        logger.info(
            "regulation_keyword_hits=%s regulation_vector_hits=%s merged_total=%s terms_sample=%s",
            len(kw_raw),
            len(raw_vector),
            len(raw_merged),
            kw_terms[:8],
        )
    elif document_type == DOCUMENT_TYPE_COURSE_MATERIAL:
        raw_merged = _merge_course_vector_and_keyword(
            raw_vector,
            kw_raw,
            max_total=candidate_limit,
        )
        logger.info(
            "course_keyword_hits=%s course_vector_hits=%s merged_total=%s terms_sample=%s",
            len(kw_raw),
            len(raw_vector),
            len(raw_merged),
            kw_terms[:8],
        )
    chunks, retrieval_stats = _filter_with_diagnostics(effective_query, raw_merged)
    chunks, retrieval_stats = _apply_adaptive_distance_fallback(chunks, raw_merged, retrieval_stats)
    if is_regulation and chunks and _regulation_neighbor_expansion_enabled(query, regulation_anchors):
        neighbor_chunks = await fetch_regulation_neighbor_chunks(
            chunks[:REGULATION_NEIGHBOR_MAX_SEEDS],
            DOCUMENT_TYPE_REGULATION,
            window=REGULATION_NEIGHBOR_WINDOW,
            max_seeds=REGULATION_NEIGHBOR_MAX_SEEDS,
        )
        if neighbor_chunks:
            before = len(chunks)
            chunks = _dedupe_chunks_merge_order(chunks, neighbor_chunks)
            logger.info(
                "regulation_neighbor_expansion_rows=%s chunks_before_after=%s/%s",
                len(neighbor_chunks),
                before,
                len(chunks),
            )
    chunks = chunks[:rerank_limit]
    if is_regulation:
        chunks = _dedupe_regulation_chunks(chunks, max_items=rerank_limit)
    top_candidates = _build_candidate_diagnostics(raw_merged, RAG_DEBUG_TOP_K)
    logger.info(
        "RAG Layer2 stats retrieval_ms=%s raw=%s distance=%s final=%s dist_drop=%s hint_drop=%s hint_fallback=%s adaptive_fallback=%s hints=%s min_dist=%s max_dist=%s",
        retrieval_elapsed_ms,
        retrieval_stats["raw_count"],
        retrieval_stats["distance_count"],
        len(chunks),
        retrieval_stats["distance_filtered_out"],
        retrieval_stats["hint_filtered_out"],
        retrieval_stats["hint_fallback_used"],
        retrieval_stats.get("adaptive_fallback_used", False),
        retrieval_stats["hints"],
        retrieval_stats["min_distance"],
        retrieval_stats["max_distance"],
    )
    logger.info("RAG Layer2 top_candidates=%s", json.dumps(top_candidates, ensure_ascii=False))
    if is_regulation and chunks:
        selected_sources = [
            {
                "document_id": str(c.get("document_id", "") or ""),
                "source_file": str(c.get("original_filename", "") or c.get("file_name", "") or ""),
                "page_number": int(c.get("page_number", 0) or 0),
            }
            for c in chunks
        ]
        logger.info("regulation_selected_sources=%s", json.dumps(selected_sources, ensure_ascii=False))
    if not chunks:
        logger.info("RAG Pipeline: Layer 2 (Retrieve) Empty (document_type=%s)", document_type)
        _LAST_RAG_RUNTIME.set({"cache_hit": False})
        return empty_msg, []

    # REGULATION: skip Layer-3 JSON rerank; one final synthesizer reads full Layer-2 chunks.
    if is_regulation:
        try:
            ans, cites = await _synthesize_regulation_answer_from_chunks(
                client,
                query,
                chunks,
                regulation_anchors,
                scope_applicability,
                empty_msg,
            )
            _LAST_RAG_RUNTIME.set({"cache_hit": False})
            return ans, cites
        except asyncio.TimeoutError:
            logger.warning(
                "Regulation synthesis timed out after %.1fs; extractive fallback",
                REGULATION_SYNTHESIS_TIMEOUT_SECONDS,
            )
            fb_a, fb_c = _build_extractive_answer_from_chunks(chunks, max_items=min(5, len(chunks) or 5))
            if fb_c:
                _LAST_RAG_RUNTIME.set({"cache_hit": False})
                return fb_a, fb_c
            _LAST_RAG_RUNTIME.set({"cache_hit": False})
            return empty_msg, []

    # 3. LAYER 3: Rerank (LLM) — course materials only
    context_text = ""
    for i, c in enumerate(chunks):
        content = _truncate_for_rerank(c.get("content"), rerank_content_chars)
        context_text += (
            f"---\n"
            f"CHUNK {i+1}\n"
            f"CHUNK_ID: {c.get('chunk_id', '')}\n"
            f"DOCUMENT_ID: {c.get('document_id', '')}\n"
            f"FILE: {c.get('original_filename', '') or c.get('file_name', '')}\n"
            f"PAGE_NUMBER: {c.get('page_number', 0)}\n"
            f"SOURCE_URI: {c.get('source_uri', '')}\n"
            f"ORIGINAL_CHUNK_INDEX: {c.get('chunk_index', 0)}\n"
            f"CONTENT: {content}\n"
        )

    list_style = _course_query_is_component_list(effective_query)
    course_max_facts = 10 if list_style else 5
    assistant_role = "You are an Academic Assistant for the 'Introduction to IT' course."

    list_extra = ""
    if list_style:
        list_extra = """
    LIST / COMPONENTS MODE (critical):
    - The student asks for main/core COMPONENTS or PARTS. Extract EVERY distinct component listed in the
      relevant chunk(s) (e.g. Producer, Exchange, Binding, Queue, Consumer) — ONE fact per component.
    - Do NOT merge multiple bullet items into one fact. Do NOT stop early if the chunk lists more items.
    - Use chunk_index of the chunk where that component appears.
    """

    rerank_system_prompt = f"""
    {assistant_role}
    Your task is to analyze the student's query and the {len(chunks)} document chunks provided below.

    INSTRUCTIONS:
    1. Prioritize rules/conditions that directly answer the question.
    2. Prefer chunks that explicitly contain query terms or direct policy conditions.
    {list_extra}
    3. Return STRICT JSON only with shape:
       {{
         "facts": [
           {{
             "label": "<item name or concept>",
             "description": "<short explanation grounded in context>",
             "chunk_index": <1-based index>,
             "quote": "<literal quote>"
           }}
         ]
       }}
       Use up to {course_max_facts} facts.
    4. Separate each chunk with a newline.
    5. If the chunks do not contain enough information to answer the question accurately, return:
       {{"facts":[]}}

    CONTEXT CHUNKS:
    {context_text}
    """

    rerank_user_content = effective_query

    try:
        rerank_started_at = time.perf_counter()
        rerank_request = client.chat.completions.create(
            model=RAG_UTILITY_MODEL,
            messages=[
                {"role": "system", "content": rerank_system_prompt},
                {"role": "user", "content": rerank_user_content},
            ],
            max_completion_tokens=900 if list_style else 500,
            temperature=0,
        )
        rerank_resp = await rerank_request

        rerank_result = (rerank_resp.choices[0].message.content or "").strip()

        if not rerank_result:
            logger.info("RAG Pipeline: Layer 3 (Rerank) No relevant matches found")
            _LAST_RAG_RUNTIME.set({"cache_hit": False})
            return empty_msg, []

        cleaned_rerank_result = _prepare_rerank_json_string(rerank_result)
        try:
            rerank_json = json.loads(cleaned_rerank_result)
            if isinstance(rerank_json, dict):
                facts = rerank_json.get("facts")
                if not isinstance(facts, list):
                    facts = rerank_json.get("answer_facts")
                if not isinstance(facts, list):
                    legacy_matches = rerank_json.get("matches", [])
                    facts = [
                        {
                            "label": "",
                            "description": "",
                            "chunk_index": m.get("chunk_index"),
                            "quote": m.get("quote", ""),
                        }
                        for m in legacy_matches
                        if isinstance(m, dict)
                    ]
            else:
                facts = []
        except json.JSONDecodeError:
            logger.warning("RAG Pipeline: Layer 3 JSON decode failed, using text fallback extractor")
            facts = _fallback_extract_facts_from_text(rerank_result)
        except Exception:
            facts = _fallback_extract_facts_from_text(rerank_result)

        if not facts:
            logger.info("RAG Pipeline: Layer 3 (Rerank) Empty match list")
            _LAST_RAG_RUNTIME.set({"cache_hit": False})
            return empty_msg, []

        if list_style:
            facts = _split_regulation_facts(facts)  # split multi-bullet quotes into one fact per line
        # Cap after split to avoid runaway token use
        facts = facts[:course_max_facts]

        citations: list[dict] = []
        answer_lines: list[str] = []
        for idx_order, m in enumerate(facts[:course_max_facts], start=1):
            try:
                idx = int(m.get("chunk_index", 0)) - 1
            except Exception:
                idx = -1
            if idx < 0 or idx >= len(chunks):
                continue
            chunk = chunks[idx]
            quote = str(m.get("quote", "")).strip() or str(chunk.get("snippet", "")).strip()
            label = str(m.get("label", "")).strip()
            description = str(m.get("description", "")).strip() or quote
            # Prefer original_filename (human-readable) over file_name (hash)
            display_name = str(chunk.get("original_filename", "") or chunk.get("file_name", "") or "Unknown source")
            page_number = int(chunk.get("page_number", 0) or 0)
            source_uri = str(chunk.get("source_uri", "") or "")
            citations.append(
                {
                    "document_id": str(chunk.get("document_id", "")),
                    "source_uri": source_uri,
                    "source_file": display_name,
                    "page_number": page_number,
                    "snippet": quote,
                    "chunk_id": str(chunk.get("chunk_id", "")),
                    "chunk_index": int(chunk.get("chunk_index", 0) or 0),
                }
            )
            if label:
                answer_lines.append(f"{idx_order}. **{label}**: {description} [{display_name}, p.{page_number}]")
            else:
                answer_lines.append(f"{idx_order}. {description} [{display_name}, p.{page_number}]")

        if not citations:
            logger.info("RAG Pipeline: Layer 3 had invalid chunk indices")
            _LAST_RAG_RUNTIME.set({"cache_hit": False})
            return empty_msg, []

        rerank_result = "\n".join(answer_lines)

        # Save successful result to cache for future Layer 1 hits (course materials only).
        # Best-effort and off-loop: a cache write failure must never fail the answer.
        if citations and document_type == DOCUMENT_TYPE_COURSE_MATERIAL:
            try:
                await asyncio.to_thread(
                    set_semantic_cache, effective_query, rerank_result, query_vector, citations=citations
                )
            except Exception as cache_exc:
                logger.warning("Semantic cache write failed (ignored): %s", cache_exc)

        rerank_elapsed_ms = int((time.perf_counter() - rerank_started_at) * 1000)
        logger.info("RAG Pipeline: Layer 3 (Rerank) Success. Cache updated. elapsed_ms=%s", rerank_elapsed_ms)
        _LAST_RAG_RUNTIME.set({"cache_hit": False})
        return rerank_result, citations

    except Exception as e:
        logger.error(f"Reranking error: {e}")
        _LAST_RAG_RUNTIME.set({"cache_hit": False})
        raise

# --- General Tools ---

async def search_web(query: str) -> str:
    """Search for general info only if course materials don't have it."""
    try:
        from duckduckgo_search import DDGS
        def _search():
            with DDGS() as ddgs:
                return list(ddgs.text(query, max_results=3))
        results = await asyncio.to_thread(_search)
        if results:
            return "\n\n".join([f"Title: {r['title']}\nBody: {r['body']}" for r in results])
        return "No results."
    except Exception as e:
        logger.error(f"Web search error: {e}")
        return "Lỗi khi tìm kiếm thông tin bên ngoài."

# Tool registry
TOOLS = {
    "check_assignment_deadline": {
        "fn": check_assignment_deadline,
        "description": "Get assignment deadline/late policy and optional submission status from database.",
        "parameters": {"assignment_name": "string", "student_code": "string"},
        "required": ["assignment_name"],
    },
    "get_assignments": {
        "fn": get_assignments,
        "description": "Chỉ sử dụng tool này khi cần lấy danh sách bài tập. Nếu sinh viên hỏi bài tập trong X ngày qua, truyền tham số days_limit = X",
        "parameters": {"days_limit": "integer"},
        "required": [],
    },
    "query_course_materials": {
        "fn": query_course_materials,
        "description": "Search course slides, lectures, and syllabus for academic knowledge.",
        "parameters": {"query": "string"},
        "required": ["query"],
    },
    "query_regulations": {
        "fn": query_regulations,
        "description": (
            "Search official school regulations and training policy documents (quy chế đào tạo, quy định). "
            "Use for questions about grading rules, theory scores, exams, academic standing, and procedural rules — "
            "not for assignment deadlines unless the question is about policy in the regulation text."
        ),
        "parameters": {"query": "string"},
        "required": ["query"],
    },
    "search_web": {
        "fn": search_web,
        "description": "Search the web for general information.",
        "parameters": {"query": "string"},
        "required": ["query"],
    },
}

def get_tool_schemas() -> list[dict]:
    schemas = []
    for name, tool in TOOLS.items():
        schemas.append({
            "type": "function",
            "function": {
                "name": name,
                "description": tool["description"],
                "parameters": {
                    "type": "object",
                    "properties": {
                        k: {"type": v, "description": k}
                        for k, v in tool["parameters"].items()
                    },
                    "required": tool.get("required", list(tool["parameters"].keys())),
                },
            },
        })
    return schemas

# Last-resort tool failure message. Individual tools already return their own
# domain-specific error strings; this only covers unexpected crashes so a bad
# tool call can never terminate the whole answer stream.
TOOL_EXECUTION_ERROR_MSG = (
    "Lỗi hệ thống khi thực thi công cụ hỗ trợ. Vui lòng thử lại sau hoặc liên hệ TA."
)


def _validate_tool_args(tool: dict, args: object) -> tuple[dict, str | None]:
    """
    Defensively validate model-generated tool arguments against the registry schema.

    - Non-dict payloads are treated as empty.
    - Unknown keys are dropped (previously they raised TypeError via ``**args``
      and killed the stream).
    - "integer" params accept numeric strings (models frequently emit "7").
    - Returns (clean_args, error_message); error_message is set when a required
      parameter is absent.
    """
    params: dict = tool.get("parameters", {}) or {}
    required = tool.get("required", []) or []
    if not isinstance(args, dict):
        args = {}

    clean: dict = {}
    for key, expected_type in params.items():
        if key not in args:
            continue
        value = args[key]
        if value is None:
            continue
        if expected_type == "string":
            clean[key] = value if isinstance(value, str) else str(value)
        elif expected_type == "integer":
            if isinstance(value, bool):
                continue
            if isinstance(value, int):
                clean[key] = value
            else:
                try:
                    clean[key] = int(str(value).strip())
                except (TypeError, ValueError):
                    continue
        else:
            clean[key] = value

    missing = [r for r in required if r not in clean]
    if missing:
        return clean, (
            "Thiếu tham số bắt buộc cho công cụ: " + ", ".join(missing) + ". "
            "Hãy gọi lại công cụ với đầy đủ tham số."
        )
    return clean, None


async def execute_tool(name: str, args: dict) -> str:
    """Execute a tool by name. Never raises: returns an error string instead."""
    tool = TOOLS.get(name)
    if not tool:
        logger.warning("tool_unknown name=%r", name)
        return f"Tool '{name}' does not exist"

    clean_args, validation_error = _validate_tool_args(tool, args)
    if validation_error:
        logger.warning("tool_args_invalid tool=%s missing_required=1", name)
        return validation_error

    try:
        return await tool["fn"](**clean_args)
    except Exception as exc:
        logger.exception("tool_execution_failed tool=%s error=%s", name, exc)
        return TOOL_EXECUTION_ERROR_MSG

"""
Retrieval threshold / filtering / canonicalization unit tests (pure functions).
"""

from src import tools


def _chunk(cid, distance, filename="week4.pdf"):
    return {
        "chunk_id": cid,
        "document_id": "d1",
        "chunk_index": 0,
        "content": f"content {cid}",
        "snippet": f"content {cid}",
        "file_name": filename,
        "original_filename": filename,
        "source_uri": f"course/{filename}",
        "page_number": 1,
        "distance": distance,
        "metadata": {},
    }


def test_relative_distance_ratio_cutoff():
    # best=0.2 → dynamic cutoff = min(0.2*1.5, 0.65) = 0.30
    chunks = [_chunk("a", 0.2), _chunk("b", 0.29), _chunk("c", 0.31)]
    kept, stats = tools._filter_with_diagnostics("generic query", chunks)
    kept_ids = [c["chunk_id"] for c in kept]
    assert kept_ids == ["a", "b"]
    assert stats["distance_filtered_out"] == 1
    assert stats["min_distance"] == 0.2


def test_relative_cutoff_respects_absolute_ceiling():
    # best=0.6 → 0.6*1.5=0.9 but ceiling RAG_FALLBACK_MAX_DISTANCE=0.65 applies
    chunks = [_chunk("a", 0.6), _chunk("b", 0.64), _chunk("c", 0.7)]
    kept, _stats = tools._filter_with_diagnostics("generic query", chunks)
    assert [c["chunk_id"] for c in kept] == ["a", "b"]


def test_adaptive_fallback_recovers_all_raw_when_filter_empties():
    raw = [_chunk("a", 0.6)]
    kept, stats = [], {"min_distance": 0.6}
    kept2, stats2 = tools._apply_adaptive_distance_fallback(kept, raw, stats)
    assert stats2["adaptive_fallback_used"] is True
    assert [c["chunk_id"] for c in kept2] == ["a"]


def test_adaptive_fallback_refuses_far_results():
    raw = [_chunk("a", 0.9)]
    kept2, stats2 = tools._apply_adaptive_distance_fallback([], raw, {"min_distance": 0.9})
    assert stats2["adaptive_fallback_used"] is False
    assert kept2 == []


def test_regulation_chunk_dedupe_by_page_and_snippet():
    c1 = _chunk("a", 0.2)
    c2 = dict(_chunk("b", 0.25), content=c1["content"], snippet=c1["snippet"])  # same doc/page/snippet
    c3 = _chunk("c", 0.3)
    c3["content"] = c3["snippet"] = "totally different"
    deduped = tools._dedupe_regulation_chunks([c1, c2, c3], max_items=10)
    assert [c["chunk_id"] for c in deduped] == ["a", "c"]


def test_citation_indices_out_of_range_are_ignored():
    chunks = [_chunk("a", 0.2), _chunk("b", 0.3)]
    cites = tools._citations_from_regulation_chunk_indices(chunks, [1, 2, 5, 0, -3, "x"])
    assert [c["chunk_id"] for c in cites] == ["a", "b"]


def test_citation_indices_fallback_all():
    chunks = [_chunk("a", 0.2), _chunk("b", 0.3)]
    cites = tools._citations_from_regulation_chunk_indices(chunks, None, fallback_all=True)
    assert len(cites) == 2


def test_citations_never_invented_from_missing_chunks():
    cites = tools._citations_from_regulation_chunk_indices([], [1, 2, 3], fallback_all=True)
    assert cites == []


def test_course_query_canonicalization_vietnamese_variants():
    q1, m1 = tools.normalize_course_rag_query("RabbitMQ có những thành phần chính nào?")
    q2, m2 = tools.normalize_course_rag_query("các thành phần cốt lõi của rabbitmq")
    assert m1["used_canonical"] and m2["used_canonical"]
    assert q1 == q2 == "rabbitmq các thành phần cốt lõi"


def test_course_query_without_component_ask_stays_raw():
    q, meta = tools.normalize_course_rag_query("RabbitMQ hoạt động thế nào?")
    assert q == "RabbitMQ hoạt động thế nào?"
    assert meta["used_canonical"] is False


def test_regulation_keyword_terms_skip_stopwords():
    terms = tools._regulation_keyword_search_terms(
        "sinh viên bị buộc thôi học khi nào và điều kiện là gì", []
    )
    assert any("buộc thôi học" in t for t in terms)
    assert all(t not in tools._VI_KW_STOPWORDS for t in terms)


def test_vi_query_anchors_map_slang_to_legal_terms():
    anchors = tools._extract_vi_query_anchors("K21 bị đuổi học khi nào?")
    lowered = [a.lower() for a in anchors]
    assert "k21" in lowered
    assert "buộc thôi học" in lowered


def test_prefetch_regulation_keywords_route_correctly():
    from src.agent import _should_prefetch_regulations

    assert _should_prefetch_regulations("Điều kiện buộc thôi học là gì?") is True
    assert _should_prefetch_regulations("quy chế thi cử thế nào") is True
    # Pure deadline/logistics questions stay on SQL tools
    assert _should_prefetch_regulations("deadline nộp bài lab 1 là khi nào") is False
    assert _should_prefetch_regulations("") is False

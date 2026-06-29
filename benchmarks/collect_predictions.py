from __future__ import annotations

import argparse
import json
import os
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import httpx

from benchmarks.common import load_jsonl


def _case_id(index: int, row: dict[str, Any]) -> str:
    return str(row.get("case_id") or f"case_{index:03d}")


def _extract_sse_answer(
    response: httpx.Response,
    request_started_at: float | None = None,
) -> tuple[str, dict[str, Any], float | None]:
    chunks: list[str] = []
    metadata: dict[str, Any] = {}
    ttft_ms: float | None = None
    started_at = request_started_at if request_started_at is not None else time.perf_counter()

    for line in response.iter_lines():
        if not line:
            continue
        if not line.startswith("data:"):
            continue

        if ttft_ms is None:
            ttft_ms = (time.perf_counter() - started_at) * 1000

        payload = line.removeprefix("data:").strip()
        if not payload:
            continue

        try:
            event = json.loads(payload)
        except json.JSONDecodeError:
            chunks.append(payload)
            continue

        if isinstance(event, dict):
            if event.get("metadata"):
                metadata = event["metadata"]
            chunk = event.get("chunk")
            if chunk:
                chunk_str = str(chunk)
                # Skip status updates starting with search emoji to keep RAGAS answers clean
                if not chunk_str.startswith("🔍"):
                    chunks.append(chunk_str)
        else:
            chunks.append(str(event))

    return "".join(chunks).strip(), metadata, ttft_ms


def _predicted_intent(metadata: dict[str, Any]) -> str:
    agent_used = str(metadata.get("agent_used", ""))
    if "intent=UNCERTAIN" in agent_used:
        return "UNCERTAIN"
    if "intent=PROCEDURAL" in agent_used or "ASSIGNMENT_AGENT" in agent_used:
        return "PROCEDURAL"
    if "intent=ACADEMIC" in agent_used or "QA_AGENT" in agent_used:
        return "ACADEMIC"
    return ""


def _retrieved_contexts(metadata: dict[str, Any]) -> list[str]:
    citations = metadata.get("citations") if isinstance(metadata, dict) else None
    if not isinstance(citations, list):
        return []
    out: list[str] = []
    for c in citations:
        if isinstance(c, dict):
            sn = str(c.get("snippet", "")).strip()
            if sn:
                out.append(sn)
    return out


def _ensure_chat_session(client: httpx.Client) -> str:
    response = client.post(
        "/api/v1/chat/sessions",
        headers={"Accept": "application/json", "Content-Type": "application/json"},
    )
    response.raise_for_status()
    payload = response.json()
    return str(payload["session_id"])


def _create_forum_thread(client: httpx.Client, title: str) -> str:
    body = {
        "title": title[:200],
        "content": "Benchmark seed message.",
        "tags": ["benchmark"],
    }
    response = client.post(
        "/api/v1/threads",
        json=body,
        headers={"Accept": "application/json", "Content-Type": "application/json"},
    )
    response.raise_for_status()
    payload = response.json()
    return str(payload["thread_id"])


def _resolve_context_id(
    client: httpx.Client,
    endpoint_template: str,
    static_id: str | None,
    case_label: str,
) -> str:
    if static_id:
        return static_id
    if "/api/v1/chat/sessions/" in endpoint_template:
        return _ensure_chat_session(client)
    if "/api/v1/threads/" in endpoint_template and "ask-ai" in endpoint_template:
        return _create_forum_thread(client, f"Bench {case_label}")
    raise ValueError(
        "context id is required for this endpoint. Set --context-id or BENCHMARK_CONTEXT_ID, "
        "or use /api/v1/chat/sessions/{id}/ask-ai or /api/v1/threads/{id}/ask-ai."
    )


def main() -> None:
    parser = argparse.ArgumentParser(description="Collect predictions from Java SSE (one fresh context per case).")
    parser.add_argument("--dataset", default="data/benchmark_ground_truth.jsonl")
    parser.add_argument("--host", default=os.getenv("BENCHMARK_HOST", "http://localhost:8080"))
    parser.add_argument(
        "--endpoint-template",
        default=os.getenv("BENCHMARK_ENDPOINT_TEMPLATE", "/api/v1/chat/sessions/{id}/ask-ai"),
    )
    parser.add_argument(
        "--context-id",
        default=os.getenv("BENCHMARK_CONTEXT_ID"),
        help="Optional: reuse one session/thread for all cases (not recommended; leaks context).",
    )
    parser.add_argument("--output", default=None)
    parser.add_argument("--limit", type=int, default=None)
    parser.add_argument("--timeout", type=float, default=120.0)
    parser.add_argument("--sleep-seconds", type=float, default=1.0)
    args = parser.parse_args()

    token = os.getenv("BENCHMARK_BEARER_TOKEN", "")
    headers = {"Accept": "text/event-stream", "Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"

    run_id = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    output_path = Path(args.output or f"reports/benchmark/{run_id}/predictions.jsonl")
    output_path.parent.mkdir(parents=True, exist_ok=True)

    rows = load_jsonl(args.dataset)
    if args.limit:
        rows = rows[: args.limit]

    static_ctx = (args.context_id or "").strip() or None

    with httpx.Client(base_url=args.host, headers=headers, timeout=args.timeout) as client:
        with output_path.open("w", encoding="utf-8") as handle:
            for idx, row in enumerate(rows, start=1):
                case_id = _case_id(idx, row)
                context_id = (
                    static_ctx
                    if static_ctx
                    else _resolve_context_id(client, args.endpoint_template, None, case_id)
                )
                path = args.endpoint_template.format(id=context_id)
                started_at = time.perf_counter()
                try:
                    with client.stream("POST", path, json={"message": row["query"]}) as response:
                        response.raise_for_status()
                        answer, metadata, ttft_ms = _extract_sse_answer(response, request_started_at=started_at)
                    latency_ms = int((time.perf_counter() - started_at) * 1000)
                    ctxs = _retrieved_contexts(metadata)
                    record = {
                        "case_id": case_id,
                        "predicted_answer": answer,
                        "predicted_intent": _predicted_intent(metadata),
                        "retrieved_contexts": ctxs,
                        "metadata": metadata,
                        "ttft_ms": ttft_ms,
                        "latency_ms": latency_ms,
                    }
                except Exception as exc:
                    record = {
                        "case_id": case_id,
                        "predicted_answer": "",
                        "predicted_intent": "",
                        "retrieved_contexts": [],
                        "error": str(exc),
                    }

                handle.write(json.dumps(record, ensure_ascii=False) + "\n")
                handle.flush()
                print(f"{case_id}: {'ERROR' if record.get('error') else 'OK'}")
                if args.sleep_seconds:
                    time.sleep(args.sleep_seconds)

    print(f"Predictions: {output_path}")


if __name__ == "__main__":
    main()

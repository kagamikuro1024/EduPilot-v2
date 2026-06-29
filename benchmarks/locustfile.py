from __future__ import annotations

import json
import os
import random
import time
from pathlib import Path
from typing import Any

from locust import HttpUser, between, events, task


def _load_queries(path: str | Path) -> list[str]:
    rows: list[str] = []
    with Path(path).open("r", encoding="utf-8-sig") as handle:
        for line in handle:
            line = line.strip()
            if not line:
                continue
            row = json.loads(line)
            query = str(row.get("query", "")).strip()
            if query:
                rows.append(query)
    if not rows:
        raise RuntimeError(f"No queries found in dataset: {path}")
    return rows


def _create_context(client: Any, endpoint_template: str) -> str:
    """Forum thread or chat session id for ask-ai."""
    token = os.getenv("BENCHMARK_BEARER_TOKEN", "")
    headers = {"Accept": "application/json", "Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"

    if "/api/v1/chat/sessions/" in endpoint_template:
        r = client.post(
            "/api/v1/chat/sessions",
            headers=headers,
            name="create_chat_session",
        )
        r.raise_for_status()
        return str(r.json()["session_id"])
    if "/api/v1/threads/" in endpoint_template:
        body = {
            "title": "Locust benchmark thread",
            "content": "Seed",
            "tags": ["benchmark"],
        }
        r = client.post(
            "/api/v1/threads",
            json=body,
            headers=headers,
            name="create_thread",
        )
        r.raise_for_status()
        return str(r.json()["thread_id"])
    rid = os.getenv("BENCHMARK_CONTEXT_ID", "").strip()
    if not rid:
        raise RuntimeError("BENCHMARK_CONTEXT_ID required for this endpoint template in Locust.")
    return rid


class AiSseUser(HttpUser):
    wait_time = between(0.2, 1.0)

    def on_start(self) -> None:
        dataset = os.getenv("BENCHMARK_DATASET", "data/benchmark_ground_truth.jsonl")
        self.queries = _load_queries(dataset)
        self.endpoint_template = os.getenv("BENCHMARK_ENDPOINT_TEMPLATE", "/api/v1/threads/{id}/ask-ai")
        self.scenario = os.getenv("BENCHMARK_SCENARIO", "baseline")
        token = os.getenv("BENCHMARK_BEARER_TOKEN", "")
        self.headers = {"Content-Type": "application/json"}
        if token:
            self.headers["Authorization"] = f"Bearer {token}"
        static = os.getenv("BENCHMARK_CONTEXT_ID", "").strip()
        self.context_id = static or _create_context(self.client, self.endpoint_template)
        self._baseline_seq = 0

    @task
    def ask_ai(self) -> None:
        if self.scenario == "baseline":
            if self._baseline_seq >= 10:
                time.sleep(1.0)
                return
            query = self.queries[self._baseline_seq]
            self._baseline_seq += 1
        else:
            query = random.choice(self.queries)

        payload: dict[str, Any] = {"message": query}
        path = self.endpoint_template.format(id=self.context_id)
        started_at = time.perf_counter()
        ttft_ms: float | None = None

        with self.client.post(
            path,
            json=payload,
            headers=self.headers,
            stream=True,
            timeout=90,
            catch_response=True,
            name=f"ask_ai:{self.scenario}",
        ) as response:
            if response.status_code >= 400:
                response.failure(f"HTTP {response.status_code}")
                return

            try:
                for raw_line in response.iter_lines(decode_unicode=True):
                    if not raw_line:
                        continue
                    if ttft_ms is None and raw_line.startswith("data:"):
                        ttft_ms = (time.perf_counter() - started_at) * 1000
                        break
                response.success()
            except Exception as exc:
                response.failure(f"SSE stream error: {exc}")
                return

        if ttft_ms is not None:
            events.request.fire(
                request_type="SSE",
                name=f"ttft:{self.scenario}",
                response_time=ttft_ms,
                response_length=0,
                exception=None,
                context={"endpoint": path},
            )

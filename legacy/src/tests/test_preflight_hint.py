"""Round-trip java_preflight tags (Java gateway → Python StreamAIResponse fast path)."""

import base64
import json

import pytest

from src.guardrails import (
    JAVA_PREFLIGHT_PREFIX,
    ClassificationResult,
    IntentType,
    classification_from_java_preflight_tags,
)


def _tag_payload(**kwargs) -> str:
    payload = {"v": 1, **kwargs}
    blob = json.dumps(payload, separators=(",", ":")).encode("utf-8")
    return JAVA_PREFLIGHT_PREFIX + base64.urlsafe_b64encode(blob).decode("ascii").rstrip("=")


def test_preflight_decodes_academic_skip_path():
    tag = _tag_payload(
        ch="PUBLIC",
        conf=0.88,
        reason="ok",
        viol=False,
        task="ACADEMIC",
    )
    result = classification_from_java_preflight_tags(["channel:FORUM", tag])
    assert isinstance(result, ClassificationResult)
    assert result.intent == IntentType.ACADEMIC
    assert result.is_violation is False
    assert result.confidence == pytest.approx(0.88)
    assert result.reasoning == "ok"


def test_preflight_invalid_returns_none():
    assert classification_from_java_preflight_tags([]) is None
    assert classification_from_java_preflight_tags(["channel:x"]) is None
    assert classification_from_java_preflight_tags([JAVA_PREFLIGHT_PREFIX + "!!!"]) is None


def test_preflight_violation_sets_reason():
    tag = _tag_payload(
        ch="PRIVATE",
        conf=0.9,
        reason="Sensitive content",
        viol=True,
        task="UNCERTAIN",
    )
    result = classification_from_java_preflight_tags([tag])
    assert result is not None
    assert result.is_violation is True
    assert result.violation_reason == "Sensitive content"

import csv
import json
import re
import unicodedata
from pathlib import Path
from typing import Any


def load_jsonl(path: str | Path) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    with Path(path).open("r", encoding="utf-8-sig") as handle:
        for idx, line in enumerate(handle, start=1):
            line = line.strip()
            if not line:
                continue
            try:
                rows.append(json.loads(line))
            except json.JSONDecodeError as exc:
                raise ValueError(f"Invalid JSONL at line {idx}: {exc}") from exc
    return rows


def load_predictions(path: str | Path) -> dict[str, dict[str, Any]]:
    src = Path(path)
    if src.suffix.lower() == ".jsonl":
        rows = load_jsonl(src)
        return {str(row["case_id"]): row for row in rows}

    if src.suffix.lower() == ".csv":
        with src.open("r", encoding="utf-8-sig", newline="") as handle:
            reader = csv.DictReader(handle)
            rows = list(reader)
        return {str(row["case_id"]): row for row in rows}

    raise ValueError("Predictions file must be .jsonl or .csv")


def normalize_text(text: str) -> str:
    lowered = text.lower()
    decomposed = unicodedata.normalize("NFD", lowered)
    no_diacritics = "".join(ch for ch in decomposed if unicodedata.category(ch) != "Mn")
    no_punctuation = re.sub(r"[^\w\s]", " ", no_diacritics)
    compact_spaces = re.sub(r"\s+", " ", no_punctuation).strip()
    return compact_spaces


def _case_synonyms(case: dict[str, Any]) -> dict[str, tuple[str, ...]]:
    merged: dict[str, tuple[str, ...]] = {}
    custom = case.get("synonym_map", {})
    if isinstance(custom, dict):
        for key, value in custom.items():
            canonical = normalize_text(str(key))
            if isinstance(value, list):
                merged[canonical] = tuple(normalize_text(str(v)) for v in value)
    return merged


def _fragment_present(fragment: str, text: str, synonyms: dict[str, tuple[str, ...]]) -> bool:
    normalized_fragment = normalize_text(fragment)
    if not normalized_fragment:
        return False
    variants = (normalized_fragment,) + synonyms.get(normalized_fragment, ())
    return any(v and v in text for v in variants)


def must_contain_pass(case: dict[str, Any], answer: str) -> bool:
    """Rule-based check from ground truth (plan §4.2)."""
    checks = case.get("answer_checks", {})
    all_of = checks.get("all_of")
    any_of = checks.get("any_of")
    regex_any_of = checks.get("regex_any_of")

    if all_of is None and any_of is None and regex_any_of is None:
        all_of = case.get("must_contain_substrings", [])
        any_of = []
        regex_any_of = []

    normalized_answer = normalize_text(answer)
    synonyms = _case_synonyms(case)

    all_of_list = [normalize_text(str(item)) for item in (all_of or [])]
    any_of_list = [normalize_text(str(item)) for item in (any_of or [])]
    regex_list = [str(item) for item in (regex_any_of or [])]

    all_ok = all(_fragment_present(f, normalized_answer, synonyms) for f in all_of_list if f)
    any_ok = True if not any_of_list else any(_fragment_present(f, normalized_answer, synonyms) for f in any_of_list if f)
    regex_ok = True if not regex_list else any(re.search(p, answer, flags=re.IGNORECASE) for p in regex_list)

    if not all_of_list and not any_of_list and not regex_list:
        return True
    return bool(all_ok and any_ok and regex_ok)


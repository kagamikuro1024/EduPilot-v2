"""Deterministic parser for private grade-report tables.

The parser operates on Docling's cleaned text and never sends a class grade
table to an LLM.  It currently supports the common exported layout:

    STT | Mã SV | Họ và tên | Nhóm | <components> | Tổng | Ghi chú

Docling flattens each data row to one line, which lets us parse and persist
only structured records keyed by student code.
"""

from __future__ import annotations

import re
import unicodedata
from typing import Any


_PAGE_RE = re.compile(r"<!--\s*PAGE:(\d+)\s*-->", re.IGNORECASE)
_DATA_ROW_RE = re.compile(
    r"^\s*(?P<ordinal>\d+)\s+"
    r"(?P<student_code>[A-Za-z][A-Za-z0-9_-]*\d[A-Za-z0-9_-]*)\s+"
    r"(?P<body>.+?)\s*$"
)
_GROUP_RE = re.compile(r"\s+(?:Nh[oó]m|Group)\s+\S+\s+", re.IGNORECASE)
_SCORE_PREFIX_RE = re.compile(
    r"^(?P<scores>(?:-?\d+(?:[.,]\d+)?\s+)+)(?P<feedback>.*)$"
)
_MAX_SCORE_RE = re.compile(r"\((\d+(?:[.,]\d+)?)\s*(?:đ|d|pts?)\)", re.IGNORECASE)


def _ascii_fold(value: str) -> str:
    normalized = unicodedata.normalize("NFD", value or "")
    folded = "".join(ch for ch in normalized if unicodedata.category(ch) != "Mn").lower()
    return folded.replace("đ", "d")


def _extract_assignment_title(lines: list[str], fallback_title: str) -> str:
    for line in lines:
        folded = _ascii_fold(line)
        if "bang diem" not in folded:
            continue
        # Split on the first title separator after the "Bảng điểm" label.
        # This avoids relying on Unicode case-fold behavior for Vietnamese Đ/Ể.
        separator = re.search(r"\s+-\s+|:\s+", line)
        if separator:
            title = line[separator.end() :].strip()
            if title:
                return title
    stem = re.sub(r"\.[A-Za-z0-9]{1,8}$", "", fallback_title or "").strip()
    return stem or "Bảng điểm"


def _extract_headers(lines: list[str]) -> tuple[list[str], float | None]:
    """Extract component labels and total max score from the flattened header."""
    start = None
    for idx, line in enumerate(lines):
        folded = _ascii_fold(line)
        if "stt" in folded and ("ma sv" in folded or "mssv" in folded or "student id" in folded):
            start = idx
            break
    if start is None:
        return [], None

    header_lines: list[str] = []
    max_score = None
    for line in lines[start : start + 30]:
        if _DATA_ROW_RE.match(line):
            break
        max_match = _MAX_SCORE_RE.search(line)
        if max_match:
            max_score = float(max_match.group(1).replace(",", "."))

        cleaned = _MAX_SCORE_RE.sub(" ", line).strip()
        folded = _ascii_fold(cleaned)
        if not cleaned:
            continue
        if not header_lines:
            # Keep only the columns after the identity/group columns.
            cleaned = re.sub(
                r"^.*?(?:Nh[oó]m|Group)\s+",
                "",
                cleaned,
                flags=re.IGNORECASE,
            ).strip()
            if cleaned:
                header_lines.append(cleaned)
            continue
        if "ghi chu" in folded or "nhan xet" in folded or "feedback" in folded:
            before = re.split(r"Ghi\s+ch[uú]|Nh[aậ]n\s+x[eé]t|Feedback", cleaned, maxsplit=1, flags=re.IGNORECASE)[0].strip()
            if before:
                header_lines.append(before)
            break
        header_lines.append(cleaned)

    labels: list[str] = []
    for item in header_lines:
        # A line may contain more than one trailing label (for example
        # "Tổng Ghi chú"); whitespace within labels such as BFS/DFS is kept.
        candidate = re.sub(r"\s+", " ", item).strip(" |-:")
        if candidate and _ascii_fold(candidate) not in {"stt", "ma sv", "ho va ten", "nhom"}:
            labels.append(candidate)
    return labels, max_score


def _source_notice(lines: list[str]) -> str | None:
    for line in lines:
        folded = _ascii_fold(line)
        if "khong co gia tri" in folded:
            return line.strip()
    if any("mock data" in _ascii_fold(line) for line in lines):
        return "MOCK DATA"
    return None


def parse_grade_report(markdown_text: str, document_title: str = "") -> list[dict[str, Any]]:
    """Return private grade records parsed from cleaned Docling output."""
    lines = [line.strip() for line in (markdown_text or "").splitlines() if line.strip()]
    if not lines:
        return []

    assignment_title = _extract_assignment_title(lines, document_title)
    labels, max_score = _extract_headers(lines)
    notice = _source_notice(lines)
    current_page = 0
    records: list[dict[str, Any]] = []

    for line in lines:
        page_match = _PAGE_RE.fullmatch(line)
        if page_match:
            current_page = int(page_match.group(1))
            continue

        row_match = _DATA_ROW_RE.match(line)
        if not row_match:
            continue
        body = row_match.group("body")
        group_match = _GROUP_RE.search(body)
        if group_match:
            student_name = body[: group_match.start()].strip()
            score_tail = body[group_match.end() :].strip()
        else:
            # Fallback for reports without a group column: the first decimal
            # score separates the name from the numeric columns.
            score_start = re.search(r"\s-?\d+(?:[.,]\d+)?(?:\s|$)", body)
            if not score_start:
                continue
            student_name = body[: score_start.start()].strip()
            score_tail = body[score_start.start() :].strip()

        score_match = _SCORE_PREFIX_RE.match(score_tail + " ")
        if not score_match:
            continue
        scores = [float(raw.replace(",", ".")) for raw in score_match.group("scores").split()]
        feedback = score_match.group("feedback").strip() or None
        if not scores:
            continue

        total_score = scores[-1]
        component_values = scores[:-1]
        component_labels = labels[:-1] if len(labels) == len(scores) else []
        if len(component_labels) != len(component_values):
            component_labels = [f"Thành phần {idx + 1}" for idx in range(len(component_values))]

        records.append(
            {
                "student_code": row_match.group("student_code").upper(),
                "student_name": student_name,
                "assignment_title": assignment_title,
                "total_score": total_score,
                "max_score": max_score,
                "feedback": feedback,
                "component_scores": dict(zip(component_labels, component_values)),
                "source_page": current_page or None,
                "source_notice": notice,
            }
        )

    return records

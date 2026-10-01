"""
Module: data_pipeline.pipeline.chunking
Description: Semantic chunking engine for Markdown text.
             Splits text based on Markdown headers and character length limits.
             Supports per-page tracking via <!-- PAGE:N --> markers.
             Includes telemetry logging and per-chunk hashing for lineage tracking.
"""

import logging
import hashlib
import re
from langchain_text_splitters import MarkdownHeaderTextSplitter, RecursiveCharacterTextSplitter

logger = logging.getLogger(__name__)

# Regex to detect page markers injected by document_parser
_PAGE_MARKER_RE = re.compile(r"<!--\s*PAGE:(\d+)\s*-->")
_HEADER_METADATA_KEYS = ("h1", "h2", "h3")
CHUNK_SIZE = 800
CHUNK_OVERLAP = 150

def compute_chunk_hash(text: str) -> str:
    """Computes SHA256 hash for individual chunk content."""
    return hashlib.sha256(text.encode('utf-8')).hexdigest()


def _split_by_page_markers(markdown_text: str) -> list[tuple[int, str]]:
    """
    Splits markdown text into (page_number, page_text) tuples
    based on <!-- PAGE:N --> markers. If no markers found,
    returns the entire text as page 0.
    """
    markers = list(_PAGE_MARKER_RE.finditer(markdown_text))
    if not markers:
        return [(0, markdown_text)]

    pages = []
    for i, match in enumerate(markers):
        page_no = int(match.group(1))
        start = match.end()
        end = markers[i + 1].start() if i + 1 < len(markers) else len(markdown_text)
        page_text = markdown_text[start:end].strip()
        if page_text:
            pages.append((page_no, page_text))
    return pages


def _with_heading_context(text: str, metadata: dict) -> str:
    """Prepend markdown headings from metadata so child chunks retain context."""
    heading_prefix = _heading_prefix(metadata)
    if not heading_prefix:
        return text

    stripped_text = text.strip()
    last_heading = heading_prefix.rsplit("\n", 1)[-1].lstrip("#").strip()
    if stripped_text.startswith(last_heading):
        return stripped_text
    return f"{heading_prefix}\n\n{stripped_text}"


def _heading_prefix(metadata: dict) -> str:
    headings = [(idx + 1, str(metadata[key]).strip()) for idx, key in enumerate(_HEADER_METADATA_KEYS) if metadata.get(key)]
    if not headings:
        return ""
    return "\n".join(f"{'#' * level} {heading}" for level, heading in headings)


def _split_with_heading_budget(text: str, metadata: dict) -> list[str]:
    heading_prefix = _heading_prefix(metadata)
    prefix_budget = len(heading_prefix) + 2 if heading_prefix else 0
    body_chunk_size = max(200, CHUNK_SIZE - prefix_budget)
    body_overlap = min(CHUNK_OVERLAP, body_chunk_size - 1)
    text_splitter = RecursiveCharacterTextSplitter(
        chunk_size=body_chunk_size,
        chunk_overlap=body_overlap
    )
    return text_splitter.split_text(text)


def chunk_markdown(markdown_text: str, source_uri: str) -> list[dict]:
    """
    Splits Markdown text by page markers, headers, and character size.
    
    Args:
        markdown_text (str): Raw Markdown content (may contain <!-- PAGE:N --> markers).
        source_uri (str): Source URI of the document for lineage tracking.
        
    Returns:
        list[dict]: List of chunks with metadata (including page number) and hashes.
    """
    if not markdown_text.strip():
        logger.warning(f"0 chunks generated from {source_uri} (Empty text)")
        return []

    # 1. Split by page markers first (preserves page boundaries)
    pages = _split_by_page_markers(markdown_text)

    # 2. Initialize MarkdownHeaderTextSplitter to split by headers (H1, H2, H3)
    # This maintains pedagogical context (Context Precision)
    headers_to_split_on = [
        ("#", "h1"),
        ("##", "h2"),
        ("###", "h3"),
    ]
    markdown_splitter = MarkdownHeaderTextSplitter(headers_to_split_on=headers_to_split_on)

    # 3. Use RecursiveCharacterTextSplitter to subdivide large blocks
    # Ensures each chunk stays within token/character limits for embedding models
    text_splitter = RecursiveCharacterTextSplitter(
        chunk_size=CHUNK_SIZE,
        chunk_overlap=CHUNK_OVERLAP
    )

    final_chunks = []
    total_chars = 0

    for page_no, page_text in pages:
        header_splits = markdown_splitter.split_text(page_text)
        
        for split in header_splits:
            chunk_metadata = dict(split.metadata)  # Preserves header hierarchy (h1, h2, h3)
            # Further split if the header split content is too long
            sub_splits = _split_with_heading_budget(split.page_content, chunk_metadata) if chunk_metadata else text_splitter.split_text(split.page_content)
            
            for text in sub_splits:
                # Merge header metadata with page number
                metadata = dict(chunk_metadata)
                metadata["page"] = page_no
                content = _with_heading_context(text, chunk_metadata)

                chunk_data = {
                    "content": content,
                    "metadata": metadata,
                    "source_uri": source_uri,
                    "content_hash": compute_chunk_hash(content)
                }
                final_chunks.append(chunk_data)
                total_chars += len(content)

    # 4. Telemetry Logging (Volume & Distribution)
    count = len(final_chunks)
    if count > 0:
        avg_len = total_chars / count
        page_nums = sorted(set(p for p, _ in pages))
        logger.info(f"Successfully generated {count} chunks from {source_uri}. "
                     f"Pages: {page_nums}. Average length: {avg_len:.1f} chars.")
    else:
        logger.warning(f"0 chunks generated from {source_uri}")

    return final_chunks


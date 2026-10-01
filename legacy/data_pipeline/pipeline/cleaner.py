"""
Module: data_pipeline.pipeline.cleaner
Description: Standardizes and cleans Markdown text after parsing.
             Ensures NFC normalization (critical for Vietnamese) and filters out data noise.
"""

import unicodedata
import re
import logging

logger = logging.getLogger(__name__)

def clean_markdown_text(text: str) -> str:
    """
    Cleans Markdown text:
    1. Normalizes text to NFC (Unicode Normalization Form C).
    2. Removes control characters (except newline).
    3. Trims trailing whitespace from each line.
    4. Preserves Markdown structure (headers, lists, tables).
    
    Args:
        text (str): Raw text to clean.
        
    Returns:
        str: Cleaned text.
    """
    if not text:
        return ""

    # 1. Normalize to NFC - vital for consistent Vietnamese character representation
    text = unicodedata.normalize('NFC', text)

    # 2. Remove control characters except for \n
    # Ranges: \x00-\x08, \x0b-\x0c, \x0e-\x1f, \x7f
    text = re.sub(r'[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]', '', text)

    # 3. Trim trailing whitespace and remove redundant empty lines at start/end
    lines = [line.rstrip() for line in text.splitlines()]
    result = "\n".join(lines).strip()

    # 4. Final validation
    if not result:
        logger.warning("Text content is empty after cleaning. Returning empty string to prevent pipeline failure.")
        return ""

    return result

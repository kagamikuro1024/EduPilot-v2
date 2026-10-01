"""
Module: data_pipeline.pipeline.document_parser
Description: Uses IBM's Docling to parse PDF/DOCX files into Markdown format.
             Includes SHA256 content hashing logic for data integrity tracking.
"""

import hashlib
import logging
import os
import tempfile
from concurrent.futures import ProcessPoolExecutor

# Threshold: Only use parallel parsing for documents longer than this
PARALLEL_PAGE_THRESHOLD = 20


def _env_flag(name: str, default: bool) -> bool:
    raw = os.getenv(name)
    if raw is None:
        return default
    return raw.strip().lower() in {"1", "true", "yes", "on"}


def _configured_worker_count() -> int:
    try:
        requested = int(os.getenv("DOCLING_MAX_WORKERS", "1"))
    except ValueError:
        requested = 1
    # Docling workers are memory-heavy. Even on larger hosts an accidental
    # cpu_count()-sized pool can exhaust the container in seconds.
    return max(1, min(requested, 4))


MAX_WORKERS = _configured_worker_count()
LOW_MEMORY_MODE = _env_flag("DOCLING_LOW_MEMORY", False)
TABLE_STRUCTURE_ENABLED = _env_flag("DOCLING_TABLE_STRUCTURE", not LOW_MEMORY_MODE)

logger = logging.getLogger(__name__)

# Disable Symlinks on Windows to avoid WinError 1314 (requires Admin rights)
os.environ["HF_HUB_DISABLE_SYMLINKS_WARNING"] = "1"
os.environ["HF_HUB_DISABLE_SYMLINKS"] = "1"

from docling.document_converter import DocumentConverter, PdfFormatOption
from docling.datamodel.base_models import InputFormat
from docling.datamodel.pipeline_options import PdfPipelineOptions
from docling.backend.pypdfium2_backend import PyPdfiumDocumentBackend

logger = logging.getLogger(__name__)

def compute_hash(content: bytes) -> str:
    """
    Computes SHA256 hash for raw data content.
    Used for document deduplication and integrity tracking.
    """
    return hashlib.sha256(content).hexdigest()

class DoclingParser:
    """
    Singleton class to manage Docling's DocumentConverter.
    Ensures the model is loaded into memory only once for optimal performance.
    """
    _instance = None

    def __new__(cls):
        if cls._instance is None:
            logger.info(
                "Initializing Docling parser shell (low_memory=%s, max_workers=%s)...",
                LOW_MEMORY_MODE,
                MAX_WORKERS,
            )
            cls._instance = super(DoclingParser, cls).__new__(cls)
            # Converters can retain several GiB of model state. Build only the
            # one needed by the current document instead of both at startup.
            cls._instance.converter = None
            cls._instance.ocr_converter = None
        return cls._instance

    def _get_converter(self, *, ocr: bool) -> DocumentConverter:
        attribute = "ocr_converter" if ocr else "converter"
        existing = getattr(self, attribute)
        if existing is not None:
            return existing

        pipeline_options = PdfPipelineOptions()
        pipeline_options.do_ocr = ocr
        pipeline_options.do_table_structure = TABLE_STRUCTURE_ENABLED
        pipeline_options.generate_page_images = False

        converter = DocumentConverter(
            allowed_formats=[InputFormat.PDF, InputFormat.DOCX],
            format_options={
                InputFormat.PDF: PdfFormatOption(
                    pipeline_options=pipeline_options,
                    backend=PyPdfiumDocumentBackend,
                )
            },
        )
        setattr(self, attribute, converter)
        return converter

    def parse(self, file_path: str) -> str:
        """
        Parses a file into Markdown format with per-page tracking.
        Inserts <!-- PAGE:N --> markers so downstream chunking can track page numbers.
        """
        logger.info(f"Starting document parsing: {file_path}")
        extension = os.path.splitext(file_path)[1].lower()
        if extension == ".pdf":
            page_count = self._get_pdf_page_count(file_path)
            has_text_layer = self._pdf_has_meaningful_text_layer(file_path)

            # Digital PDFs do not need the multi-GiB Docling/PyTorch pipeline in
            # constrained Spaces. pypdfium2 preserves page markers for citations.
            if LOW_MEMORY_MODE and has_text_layer:
                logger.info("Low-memory parser: using pypdfium2 for digital PDF.")
                return self._extract_pages_pypdfium(file_path)

            if page_count > PARALLEL_PAGE_THRESHOLD:
                mode = "sequential chunks" if MAX_WORKERS == 1 else f"{MAX_WORKERS} workers"
                logger.info("Document has %s pages. Parsing with %s.", page_count, mode)
                return self.parse_parallel(file_path, page_count)

            if not has_text_layer:
                logger.info("Adaptive parser: Scanned PDF detected. Enabling OCR.")
                converter = self._get_converter(ocr=True)
            else:
                logger.info("Adaptive parser: Digital PDF detected. Using Fast-Docling (No OCR).")
                converter = self._get_converter(ocr=False)
        else:
            converter = self._get_converter(ocr=False)

        # Execute Docling conversion (Standard Path)
        result = converter.convert(file_path)
        doc = result.document
        
        # Manually assemble markdown with page markers and structural elements
        markdown_chunks = []
        current_page = -1
        
        # Helper to map Docling labels to Markdown prefixes
        label_map = {
            "title": "# ",
            "section_header": "## ",
            "subsection_header": "### ",
            "paragraph": "",
            "list_item": "- ",
            "table": "\n",
        }
        
        for item, level in doc.iterate_items():
            # 1. Handle Page Markers
            if item.prov and len(item.prov) > 0:
                page_no = item.prov[0].page_no
                if page_no != current_page:
                    current_page = page_no
                    markdown_chunks.append(f"\n<!-- PAGE:{current_page} -->\n")
            
            # 2. Handle Content & Structure
            label = item.label.value.lower() if hasattr(item.label, 'value') else str(item.label).lower()
            prefix = label_map.get(label, "")
            
            # Get text content safely
            text = getattr(item, 'text', '').strip()
            if not text and label == "table":
                # Tables need special handling if not using export_to_markdown
                try:
                    text = doc.export_to_markdown(item_set=[item]) # Try list again or skip
                except:
                    text = "[Table Content]"
            
            if text:
                markdown_chunks.append(f"{prefix}{text}")
        
        markdown_output = "\n\n".join(markdown_chunks)

        # Fallback to direct pypdfium2 extraction only if Docling produces empty results
        if not markdown_output.strip() and extension == ".pdf":
            logger.warning(f"Docling produced empty output for {file_path}. Falling back to Lite-Parser.")
            return self._extract_pages_pypdfium(file_path)

        return markdown_output

    def parse_parallel(self, file_path: str, total_pages: int) -> str:
        """
        Splits a large PDF and parses chunks in parallel processes.
        """
        chunk_size = 10  # 10 pages per worker is a good balance
        chunks = []
        
        with tempfile.TemporaryDirectory() as tmp_dir:
            # 1. Split PDF into temporary parts
            pdf_parts = self._split_pdf(file_path, tmp_dir, chunk_size)
            
            # 2. Map to ProcessPool
            # Note: We use a standalone function to avoid pickling issues with Singleton
            worker_count = min(MAX_WORKERS, len(pdf_parts))
            if worker_count == 1:
                logger.info("Parsing %s PDF chunks sequentially to bound memory.", len(pdf_parts))
                results = [extract_markdown(part) for part in pdf_parts]
            else:
                logger.info("Spawning %s workers for %s PDF chunks.", worker_count, len(pdf_parts))
                with ProcessPoolExecutor(max_workers=worker_count) as executor:
                    # We call the top-level extract_markdown function in each process
                    results = list(executor.map(extract_markdown, pdf_parts))
            
            # 3. Reduce (Join)
            return "\n\n".join(results)

    @staticmethod
    def _get_pdf_page_count(file_path: str) -> int:
        import pypdfium2 as pdfium
        pdf = pdfium.PdfDocument(file_path)
        count = len(pdf)
        pdf.close()
        return count

    @staticmethod
    def _split_pdf(file_path: str, output_dir: str, chunk_size: int) -> list[str]:
        import pypdfium2 as pdfium
        pdf = pdfium.PdfDocument(file_path)
        total_pages = len(pdf)
        part_paths = []
        
        for start in range(0, total_pages, chunk_size):
            end = min(start + chunk_size, total_pages)
            part_pdf = pdfium.PdfDocument.new()
            part_pdf.import_pages(pdf, [i for i in range(start, end)])
            
            part_path = os.path.join(output_dir, f"part_{start}_{end}.pdf")
            part_pdf.save(part_path)
            part_pdf.close()
            part_paths.append(part_path)
            
        pdf.close()
        return part_paths

    @staticmethod
    def _pdf_has_meaningful_text_layer(file_path: str) -> bool:
        """
        Fast heuristic to decide whether a PDF has a usable text layer.
        Returns True when text extraction is likely reliable; False suggests scanned/image PDF.
        """
        import pypdfium2 as pdfium

        min_chars_per_text_page = 30
        min_text_page_ratio = 0.25
        min_total_chars = 200

        pdf = pdfium.PdfDocument(file_path)
        try:
            page_count = len(pdf)
            if page_count == 0:
                return False

            text_pages = 0
            total_chars = 0

            for i in range(page_count):
                page = pdf[i]
                textpage = None
                try:
                    textpage = page.get_textpage()
                    text = (textpage.get_text_bounded() or "").strip()
                    if len(text) >= min_chars_per_text_page:
                        text_pages += 1
                    total_chars += len(text)
                finally:
                    if textpage is not None:
                        textpage.close()
                    page.close()

            ratio = text_pages / max(1, page_count)
            return ratio >= min_text_page_ratio and total_chars >= min_total_chars
        finally:
            pdf.close()

    @staticmethod
    def _extract_pages_pypdfium(file_path: str) -> str:
        """
        Uses pypdfium2 to extract text per page and insert PAGE markers.
        This is more reliable than Docling's per-page export across versions.
        """
        import pypdfium2 as pdfium
        
        pdf = pdfium.PdfDocument(file_path)
        page_count = len(pdf)
        if page_count == 0:
            return ""

        page_markdowns = []
        for i in range(page_count):
            page = pdf[i]
            textpage = page.get_textpage()
            text = textpage.get_text_bounded()
            textpage.close()
            page.close()
            
            if text and text.strip():
                # Page numbers are 1-indexed for display
                page_markdowns.append(f"<!-- PAGE:{i + 1} -->\n{text.strip()}")

        pdf.close()
        
        if page_markdowns:
            return "\n\n".join(page_markdowns)
        return ""

def extract_markdown(file_path: str) -> str:
    """
    Converts a document to Markdown using the global DoclingParser instance.
    Supports PDF, DOCX, and other docling-compatible formats.
    """
    try:
        parser = DoclingParser()
        return parser.parse(file_path)
    except Exception as e:
        logger.error(f"Error parsing document {file_path}: {e}")
        raise


def extract_text_file(file_path: str) -> str:
    """
    Reads a plain-text (.txt) file and returns its content as a string.
    Uses UTF-8 with fallback to latin-1 to handle common encoding variations.
    This is intentionally lightweight — no external dependency (YAGNI).
    """
    try:
        with open(file_path, "r", encoding="utf-8") as f:
            content = f.read()
        logger.info(f"Plain text file read successfully: {file_path} ({len(content)} chars)")
        return content
    except UnicodeDecodeError:
        logger.warning(f"UTF-8 decode failed for {file_path}, retrying with latin-1")
        with open(file_path, "r", encoding="latin-1") as f:
            return f.read()
    except Exception as e:
        logger.error(f"Error reading text file {file_path}: {e}")
        raise

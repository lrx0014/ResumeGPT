from __future__ import annotations

import csv
import hashlib
import io
import json
import os
import shutil
import subprocess
import tempfile
import time
import zipfile
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any
from xml.etree import ElementTree

MAX_DOCUMENT_BYTES = 10 * 1024 * 1024
MAX_SEGMENTS = 2_000
MAX_SEGMENT_BYTES = 4_000


class ExtractionError(Exception):
    def __init__(self, code: str, message: str) -> None:
        super().__init__(message)
        self.code = code


@dataclass(frozen=True)
class Segment:
    text: str
    page: int | None
    paragraph: int
    confidence: float
    bounding_box: dict[str, int] | None = None


@dataclass(frozen=True)
class Result:
    media_type: str
    sha256: str
    parser_version: str
    malware_status: str
    segments: list[Segment]

    def to_json(self) -> str:
        return json.dumps(asdict(self), ensure_ascii=False, separators=(",", ":"))


def detect_media_type(path: Path, data: bytes) -> str:
    suffix = path.suffix.lower()
    detected: str | None = None
    if data.startswith(b"%PDF-"):
        detected = "application/pdf"
    elif data.startswith(b"\x89PNG\r\n\x1a\n"):
        detected = "image/png"
    elif data.startswith(b"\xff\xd8\xff"):
        detected = "image/jpeg"
    elif data.startswith(b"\xd0\xcf\x11\xe0\xa1\xb1\x1a\xe1"):
        detected = "application/msword"
    elif data.startswith(b"PK\x03\x04"):
        try:
            with zipfile.ZipFile(io.BytesIO(data)) as archive:
                if "word/document.xml" in archive.namelist():
                    detected = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
        except zipfile.BadZipFile as error:
            raise ExtractionError("invalid_archive", "The uploaded ZIP container is invalid.") from error
        if detected is None:
            raise ExtractionError("unsupported_archive", "The uploaded archive is not a DOCX document.")
    expected_extensions = {
        "application/pdf": {".pdf"},
        "image/png": {".png"},
        "image/jpeg": {".jpg", ".jpeg"},
        "application/msword": {".doc"},
        "application/vnd.openxmlformats-officedocument.wordprocessingml.document": {".docx"},
    }
    if detected is not None:
        if suffix not in expected_extensions[detected]:
            raise ExtractionError("extension_mismatch", "The filename extension does not match the detected document type.")
        return detected
    try:
        data.decode("utf-8")
    except UnicodeDecodeError as error:
        raise ExtractionError("unsupported_media_type", "The file signature is not supported.") from error
    if suffix == ".tex":
        return "application/x-tex"
    if suffix == ".md":
        return "text/markdown"
    if suffix == ".txt":
        return "text/plain"
    raise ExtractionError("extension_mismatch", "UTF-8 text must use a .txt, .md, or .tex filename.")


def scan_malware(path: Path) -> str:
    scanner = malware_scanner()
    try:
        completed = subprocess.run(
            [scanner, "--no-summary", "--stdout", str(path)],
            capture_output=True,
            text=True,
            timeout=60,
            check=False,
        )
    except subprocess.TimeoutExpired as error:
        raise ExtractionError("scanner_timeout", "Malware scanning exceeded its time limit.") from error
    if completed.returncode == 0:
        return "clean"
    if completed.returncode == 1:
        raise ExtractionError("malware_detected", "Malware scanning rejected the document.")
    raise ExtractionError("scanner_failed", "Malware scanning could not complete.")


def malware_scanner() -> str:
    scanner = shutil.which("clamscan")
    if scanner is None:
        raise ExtractionError("scanner_unavailable", "Malware scanning is unavailable; the document remains quarantined.")
    database_directory = Path(os.environ.get("CLAMAV_DATABASE_DIRECTORY", "/var/lib/clamav"))
    definitions = list(database_directory.glob("*.cvd")) + list(database_directory.glob("*.cld"))
    if not definitions or time.time() - max(item.stat().st_mtime for item in definitions) > 7 * 24 * 60 * 60:
        raise ExtractionError("scanner_definitions_stale", "Malware definitions are missing or stale; the document remains quarantined.")
    return scanner


def text_segments(text: str, page: int | None = None) -> list[Segment]:
    if not text.strip():
        return []
    result: list[Segment] = []
    for paragraph, line in enumerate(text.replace("\r\n", "\n").split("\n"), start=1):
        line = line.strip()
        if not line:
            continue
        if len(line.encode("utf-8")) > MAX_SEGMENT_BYTES:
            raise ExtractionError("segment_too_large", "An extracted paragraph exceeds the supported length.")
        result.append(Segment(text=line, page=page, paragraph=paragraph, confidence=1.0))
    return result


def extract_docx(data: bytes) -> list[Segment]:
    try:
        with zipfile.ZipFile(io.BytesIO(data)) as archive:
            document_info = archive.getinfo("word/document.xml")
            if document_info.file_size > MAX_DOCUMENT_BYTES:
                raise ExtractionError("expanded_size_limit_exceeded", "The expanded DOCX document is too large.")
            root = ElementTree.fromstring(archive.read(document_info))
    except ExtractionError:
        raise
    except (KeyError, zipfile.BadZipFile, ElementTree.ParseError) as error:
        raise ExtractionError("invalid_docx", "The DOCX document structure is invalid.") from error
    namespace = "{http://schemas.openxmlformats.org/wordprocessingml/2006/main}"
    result: list[Segment] = []
    for paragraph, node in enumerate(root.iter(f"{namespace}p"), start=1):
        text = "".join(item.text or "" for item in node.iter(f"{namespace}t")).strip()
        if text:
            result.append(Segment(text=text, page=None, paragraph=paragraph, confidence=1.0))
    return result


def extract_legacy_doc(path: Path) -> list[Segment]:
    converter = shutil.which("antiword")
    if converter is None:
        raise ExtractionError("converter_unavailable", "Legacy DOC extraction requires antiword.")
    try:
        completed = subprocess.run([converter, str(path)], capture_output=True, timeout=60, check=False)
    except subprocess.TimeoutExpired as error:
        raise ExtractionError("converter_timeout", "Legacy DOC extraction exceeded its time limit.") from error
    if completed.returncode != 0:
        raise ExtractionError("invalid_doc", "The legacy DOC document could not be extracted.")
    return text_segments(completed.stdout.decode("utf-8", errors="replace"))


def ocr_image(path: Path, page: int | None) -> list[Segment]:
    tesseract = shutil.which("tesseract")
    if tesseract is None:
        raise ExtractionError("ocr_unavailable", "OCR requires Tesseract.")
    try:
        completed = subprocess.run(
            [tesseract, str(path), "stdout", "-l", "eng", "tsv"],
            capture_output=True,
            text=True,
            timeout=120,
            check=False,
        )
    except subprocess.TimeoutExpired as error:
        raise ExtractionError("ocr_timeout", "OCR exceeded its time limit.") from error
    if completed.returncode != 0:
        raise ExtractionError("ocr_failed", "OCR could not process the document image.")
    grouped: dict[tuple[str, str, str], list[dict[str, Any]]] = {}
    for row in csv.DictReader(io.StringIO(completed.stdout), delimiter="\t"):
        word = (row.get("text") or "").strip()
        try:
            confidence = float(row.get("conf") or -1)
        except ValueError:
            confidence = -1
        if not word or confidence < 0:
            continue
        key = (row.get("block_num", "0"), row.get("par_num", "0"), row.get("line_num", "0"))
        grouped.setdefault(key, []).append({"text": word, "confidence": confidence, "left": int(row["left"]), "top": int(row["top"]), "width": int(row["width"]), "height": int(row["height"])})
    result: list[Segment] = []
    for paragraph, words in enumerate(grouped.values(), start=1):
        left = min(word["left"] for word in words)
        top = min(word["top"] for word in words)
        right = max(word["left"] + word["width"] for word in words)
        bottom = max(word["top"] + word["height"] for word in words)
        result.append(Segment(
            text=" ".join(word["text"] for word in words), page=page, paragraph=paragraph,
            confidence=round(sum(word["confidence"] for word in words) / len(words) / 100, 4),
            bounding_box={"x": left, "y": top, "width": right-left, "height": bottom-top},
        ))
    return result


def extract_pdf(path: Path) -> list[Segment]:
    try:
        import pypdfium2 as pdfium
    except ImportError as error:
        raise ExtractionError("pdf_engine_unavailable", "PDF extraction requires pypdfium2.") from error
    result: list[Segment] = []
    try:
        document = pdfium.PdfDocument(path)
        if len(document) > 100:
            raise ExtractionError("page_limit_exceeded", "PDF documents are limited to 100 pages.")
        for page_index, page in enumerate(document, start=1):
            text_page = page.get_textpage()
            text = text_page.get_text_range().strip()
            if text:
                result.extend(text_segments(text, page_index))
                continue
            with tempfile.NamedTemporaryFile(suffix=".png") as image_file:
                page.render(scale=2).to_pil().save(image_file.name)
                result.extend(ocr_image(Path(image_file.name), page_index))
    except ExtractionError:
        raise
    except Exception as error:
        raise ExtractionError("invalid_pdf", "The PDF document could not be extracted.") from error
    return result


def extract(path: Path, *, malware_scan: bool = True) -> Result:
    try:
        size = path.stat().st_size
    except OSError as error:
        raise ExtractionError("source_unavailable", "The source file is unavailable.") from error
    if size < 1 or size > MAX_DOCUMENT_BYTES:
        raise ExtractionError("size_limit_exceeded", "Documents must be between 1 byte and 10 MiB.")
    data = path.read_bytes()
    media_type = detect_media_type(path, data)
    malware_status = scan_malware(path) if malware_scan else "test_skipped"
    if media_type in {"text/plain", "text/markdown", "application/x-tex"}:
        segments = text_segments(data.decode("utf-8"))
    elif media_type.endswith("wordprocessingml.document"):
        segments = extract_docx(data)
    elif media_type == "application/msword":
        segments = extract_legacy_doc(path)
    elif media_type == "application/pdf":
        segments = extract_pdf(path)
    elif media_type in {"image/png", "image/jpeg"}:
        segments = ocr_image(path, 1)
    else:
        raise ExtractionError("unsupported_media_type", "The document media type is not supported.")
    if not segments:
        raise ExtractionError("no_text", "No readable text was found in the document.")
    if len(segments) > MAX_SEGMENTS:
        raise ExtractionError("segment_limit_exceeded", "The document contains too many text segments.")
    return Result(media_type=media_type, sha256=hashlib.sha256(data).hexdigest(), parser_version="document-extractor-v1", malware_status=malware_status, segments=segments)

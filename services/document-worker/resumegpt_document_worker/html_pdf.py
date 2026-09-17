from __future__ import annotations

from urllib.parse import urlparse

from weasyprint import HTML, default_url_fetcher

from .extractor import ExtractionError
from .preview import MAX_PREVIEW_BYTES


def _safe_url_fetcher(url: str, *args: object, **kwargs: object) -> dict:
    parsed = urlparse(url)
    if parsed.scheme == "data":
        return default_url_fetcher(url, *args, **kwargs)
    raise ValueError("External and local resources are disabled for generated HTML documents.")


def render_html_pdf(data: bytes) -> bytes:
    try:
        source = data.decode("utf-8")
    except UnicodeDecodeError as error:
        raise ExtractionError("invalid_html", "Generated HTML must use UTF-8 encoding.") from error
    lowered = source.lower()
    if "<html" not in lowered or "<body" not in lowered or "</html>" not in lowered:
        raise ExtractionError("invalid_html", "The Document Designer must provide a complete HTML document.")
    try:
        result = HTML(string=source, url_fetcher=_safe_url_fetcher).write_pdf()
    except Exception as error:
        raise ExtractionError("html_render_failed", f"The generated HTML could not be rendered: {str(error)[:600]}") from error
    if not result.startswith(b"%PDF-") or len(result) > MAX_PREVIEW_BYTES:
        raise ExtractionError("invalid_html_pdf", "The HTML renderer did not produce a valid PDF.")
    return result

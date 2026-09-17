import pytest

from resumegpt_document_worker.extractor import ExtractionError
from resumegpt_document_worker.html_pdf import _safe_url_fetcher, render_html_pdf


def test_render_html_pdf_produces_pdf() -> None:
    result = render_html_pdf(b"<!doctype html><html><body><h1>Resume</h1><p>Evidence</p></body></html>")

    assert result.startswith(b"%PDF-")


def test_render_html_pdf_requires_complete_document() -> None:
    with pytest.raises(ExtractionError, match="complete HTML"):
        render_html_pdf(b"<p>Fragment</p>")


def test_html_renderer_blocks_external_resources() -> None:
    with pytest.raises(ValueError, match="External and local resources"):
        _safe_url_fetcher("https://example.com/avatar.png")

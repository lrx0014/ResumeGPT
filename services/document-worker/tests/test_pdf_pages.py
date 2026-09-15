import io

import pytest
from PIL import Image

from resumegpt_document_worker.extractor import ExtractionError
from resumegpt_document_worker.pdf_pages import render_pdf_pages


def test_render_pdf_pages_returns_review_images() -> None:
    encoded = io.BytesIO()
    Image.new("RGB", (300, 400), "white").save(encoded, format="PDF")

    result = render_pdf_pages(encoded.getvalue())

    assert result["pageCount"] == 1
    assert len(result["images"]) == 1
    assert len(result["images"][0]) > 100


def test_render_pdf_pages_rejects_non_pdf() -> None:
    with pytest.raises(ExtractionError, match="valid PDF"):
        render_pdf_pages(b"not a PDF")


from __future__ import annotations

import base64
import io

import pypdfium2 as pdfium
from PIL import Image

from .extractor import ExtractionError

MAX_REVIEW_PAGES = 3


def render_pdf_pages(data: bytes) -> dict[str, object]:
    if not data.startswith(b"%PDF-"):
        raise ExtractionError("invalid_pdf", "Visual review requires a valid PDF.")
    try:
        document = pdfium.PdfDocument(data)
        page_count = len(document)
        images: list[str] = []
        for index in range(min(page_count, MAX_REVIEW_PAGES)):
            page = document[index]
            bitmap = page.render(scale=1.5)
            image: Image.Image = bitmap.to_pil().convert("RGB")
            encoded = io.BytesIO()
            image.save(encoded, format="JPEG", quality=82, optimize=True)
            images.append(base64.b64encode(encoded.getvalue()).decode("ascii"))
            page.close()
        document.close()
    except Exception as error:
        raise ExtractionError("pdf_raster_failed", "The generated PDF could not be prepared for visual review.") from error
    if page_count < 1 or not images:
        raise ExtractionError("empty_pdf", "The generated PDF contains no reviewable pages.")
    return {"pageCount": page_count, "images": images}


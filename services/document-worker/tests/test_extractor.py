import io
import shutil
import zipfile
from pathlib import Path

import pytest

from resumegpt_document_worker.extractor import ExtractionError, detect_media_type, extract


def test_extracts_utf8_text_with_hash_and_paragraphs(tmp_path: Path) -> None:
    source = tmp_path / "profile.txt"
    source.write_text("Built Go services\n\nReduced latency by 30%", encoding="utf-8")
    result = extract(source, malware_scan=False)
    assert result.media_type == "text/plain"
    assert len(result.sha256) == 64
    assert [segment.paragraph for segment in result.segments] == [1, 3]
    assert all(segment.confidence == 1 for segment in result.segments)


def test_detects_tex_by_signature_and_extension(tmp_path: Path) -> None:
    source = tmp_path / "profile.tex"
    source.write_text("\\section{Experience}\nBuilt distributed systems", encoding="utf-8")
    result = extract(source, malware_scan=False)
    assert result.media_type == "application/x-tex"
    assert result.segments[1].text == "Built distributed systems"


def test_extracts_markdown_as_editable_text(tmp_path: Path) -> None:
    source = tmp_path / "profile.md"
    source.write_text("# Experience\n\nBuilt distributed systems", encoding="utf-8")
    result = extract(source, malware_scan=False)
    assert result.media_type == "text/markdown"
    assert result.segments[0].text == "# Experience"


def test_extracts_docx_paragraphs_without_executing_document_content(tmp_path: Path) -> None:
    source = tmp_path / "profile.docx"
    document = b'''<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
    <w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>
    <w:p><w:r><w:t>Backend engineer</w:t></w:r></w:p><w:p><w:r><w:t>Go and PostgreSQL</w:t></w:r></w:p>
    </w:body></w:document>'''
    content_types = b'<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>'
    with zipfile.ZipFile(source, "w") as archive:
        archive.writestr("[Content_Types].xml", content_types)
        archive.writestr("word/document.xml", document)
    result = extract(source, malware_scan=False)
    assert [segment.text for segment in result.segments] == ["Backend engineer", "Go and PostgreSQL"]


def test_extracts_multifile_latex_zip_from_selected_entry(tmp_path: Path) -> None:
    source = tmp_path / "resume.zip"
    with zipfile.ZipFile(source, "w") as archive:
        archive.writestr("src/resume.tex", "\\documentclass{article}\n\\input{sections/work}")
        archive.writestr("src/sections/work.tex", "Built distributed systems")
    result = extract(source, malware_scan=False, entry_file="src/resume.tex")
    assert result.media_type == "application/zip"
    assert result.segments[0].text == "File: src/resume.tex"
    assert any(segment.text == "Built distributed systems" for segment in result.segments)


def test_latex_zip_rejects_path_traversal(tmp_path: Path) -> None:
    source = tmp_path / "resume.zip"
    with zipfile.ZipFile(source, "w") as archive:
        archive.writestr("main.tex", "\\documentclass{article}")
        archive.writestr("../outside.sty", "unsafe")
    with pytest.raises(ExtractionError) as error:
        extract(source, malware_scan=False)
    assert error.value.code == "unsafe_archive_path"


def test_rejects_binary_signature_extension_mismatch(tmp_path: Path) -> None:
    source = tmp_path / "profile.doc"
    source.write_bytes(b"%PDF-1.7\n")
    with pytest.raises(ExtractionError) as error:
        detect_media_type(source, source.read_bytes())
    assert error.value.code == "extension_mismatch"


def test_rejects_extension_mismatch_and_missing_scanner(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    source = tmp_path / "profile.pdf"
    source.write_text("This is not a PDF", encoding="utf-8")
    with pytest.raises(ExtractionError, match=r"\.txt, \.md, or \.tex"):
        detect_media_type(source, source.read_bytes())
    valid = tmp_path / "profile.txt"
    valid.write_text("Evidence", encoding="utf-8")
    monkeypatch.setattr("shutil.which", lambda _: None)
    with pytest.raises(ExtractionError, match="quarantined") as error:
        extract(valid)
    assert error.value.code == "scanner_unavailable"


def test_fails_closed_when_malware_definitions_are_missing(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    source = tmp_path / "profile.txt"
    source.write_text("Evidence", encoding="utf-8")
    definitions = tmp_path / "definitions"
    definitions.mkdir()
    monkeypatch.setattr("shutil.which", lambda _: "/usr/bin/true")
    monkeypatch.setenv("CLAMAV_DATABASE_DIRECTORY", str(definitions))
    with pytest.raises(ExtractionError) as error:
        extract(source)
    assert error.value.code == "scanner_definitions_stale"


@pytest.mark.skipif(shutil.which("tesseract") is None, reason="Tesseract is not installed")
def test_extracts_image_ocr_with_confidence_and_bounding_box(tmp_path: Path) -> None:
    from PIL import Image, ImageDraw

    source = tmp_path / "profile.png"
    image = Image.new("RGB", (900, 180), "white")
    ImageDraw.Draw(image).text((30, 50), "Resume engineering experience", fill="black", font_size=48)
    image.save(source)
    result = extract(source, malware_scan=False)
    assert result.media_type == "image/png"
    assert result.segments
    assert any("engineering" in segment.text.lower() for segment in result.segments)
    assert all(segment.bounding_box is not None for segment in result.segments)
    assert all(0 <= segment.confidence <= 1 for segment in result.segments)


@pytest.mark.skipif(shutil.which("tesseract") is None, reason="Tesseract is not installed")
def test_uses_ocr_for_image_only_pdf(tmp_path: Path) -> None:
    from PIL import Image, ImageDraw

    source = tmp_path / "profile.pdf"
    image = Image.new("RGB", (900, 180), "white")
    ImageDraw.Draw(image).text((30, 50), "Verified project delivery", fill="black", font_size=48)
    image.save(source, "PDF")
    result = extract(source, malware_scan=False)
    assert result.media_type == "application/pdf"
    assert any("project" in segment.text.lower() for segment in result.segments)
    assert all(segment.page == 1 for segment in result.segments)

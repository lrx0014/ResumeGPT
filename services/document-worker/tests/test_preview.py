import shutil
import io
import zipfile
from pathlib import Path

import pytest

from resumegpt_document_worker.extractor import ExtractionError
from resumegpt_document_worker.preview import render_preview


class Completed:
    returncode = 0
    stdout = b""
    stderr = b""


class Failed:
    returncode = 1
    stdout = b"! LaTeX Error: File `fontawesome.sty' not found."
    stderr = b""


class FailedSyntax:
    returncode = 1
    stdout = b"! Undefined control sequence.\nl.42 \\unknowncommand{Resume}"
    stderr = b""


def test_render_tex_preview_uses_no_shell_escape(monkeypatch) -> None:
    captured: list[str] = []
    monkeypatch.setattr(shutil, "which", lambda _: "/usr/bin/pdflatex")

    def run(command, *, cwd, capture_output, timeout, check, env):
        captured.extend(command)
        assert env["HOME"].startswith(str(cwd))
        assert env["openin_any"] == "p"
        assert env["openout_any"] == "p"
        (Path(cwd) / "output" / "template.pdf").write_bytes(b"%PDF-preview")
        return Completed()

    monkeypatch.setattr("subprocess.run", run)
    result = render_preview("resume.tex", b"\\documentclass{article}")
    assert result == b"%PDF-preview"
    assert "-no-shell-escape" in captured


def test_render_word_preview_uses_isolated_user_profile(monkeypatch) -> None:
    captured: list[str] = []
    monkeypatch.setattr(shutil, "which", lambda _: "/usr/bin/libreoffice")

    def run(command, *, cwd, capture_output, timeout, check, env):
        captured.extend(command)
        assert env["HOME"].startswith(str(cwd))
        (Path(cwd) / "output" / "template.pdf").write_bytes(b"%PDF-preview")
        return Completed()

    monkeypatch.setattr("subprocess.run", run)
    result = render_preview("resume.docx", b"docx")
    assert result == b"%PDF-preview"
    assert any(argument.startswith("-env:UserInstallation=file://") for argument in captured)


def test_render_latex_zip_uses_selected_entry_and_archive_directory(monkeypatch) -> None:
    captured: list[str] = []
    archive_data = io.BytesIO()
    with zipfile.ZipFile(archive_data, "w") as archive:
        archive.writestr("project/resume.tex", "\\documentclass{article}")
        archive.writestr("project/style.sty", "")
    monkeypatch.setattr(shutil, "which", lambda _: "/usr/bin/pdflatex")

    def run(command, *, cwd, capture_output, timeout, check, env):
        captured.extend(command)
        output = Path(next(argument.split("=", 1)[1] for argument in command if argument.startswith("-output-directory=")))
        (output / "resume.pdf").write_bytes(b"%PDF-preview")
        assert Path(cwd).name == "project"
        return Completed()

    monkeypatch.setattr("subprocess.run", run)
    result = render_preview("resume.zip", archive_data.getvalue(), "project/resume.tex")
    assert result == b"%PDF-preview"
    assert captured[-1] == "resume.tex"


def test_render_preview_rejects_unsupported_format() -> None:
    with pytest.raises(ExtractionError, match="Preview supports"):
        render_preview("resume.pdf", b"%PDF-existing")


def test_render_tex_preview_reports_missing_dependency(monkeypatch) -> None:
    monkeypatch.setattr(shutil, "which", lambda _: "/usr/bin/pdflatex")
    monkeypatch.setattr("subprocess.run", lambda *args, **kwargs: Failed())

    with pytest.raises(ExtractionError, match="fontawesome\\.sty"):
        render_preview("resume.tex", b"\\documentclass{article}")


def test_render_tex_preview_reports_specific_latex_error(monkeypatch) -> None:
    monkeypatch.setattr(shutil, "which", lambda _: "/usr/bin/pdflatex")
    monkeypatch.setattr("subprocess.run", lambda *args, **kwargs: FailedSyntax())

    with pytest.raises(ExtractionError, match=r"Undefined control sequence.*unknowncommand"):
        render_preview("resume.tex", b"\\documentclass{article}")

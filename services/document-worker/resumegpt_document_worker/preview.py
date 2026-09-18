from __future__ import annotations

import os
import re
import shutil
import subprocess
import tempfile
from pathlib import Path

from .extractor import ExtractionError
from .latex_archive import extract_latex_archive

MAX_PREVIEW_BYTES = 20 * 1024 * 1024
MISSING_LATEX_FILE = re.compile(r"LaTeX Error: File [`']([^`']+)' not found\.")
LATEX_ERROR_LINE = re.compile(r"(?m)^!\s*(.+?)\s*$")
LATEX_SOURCE_LINE = re.compile(r"(?m)^l\.\d+\s+(.+?)\s*$")


def _render_failure_message(completed: subprocess.CompletedProcess[bytes], latex: bool) -> str:
    if latex:
        output = b"\n".join((completed.stdout or b"", completed.stderr or b"")).decode("utf-8", errors="replace")
        missing = MISSING_LATEX_FILE.search(output)
        if missing:
            return f"LaTeX dependency or project file '{missing.group(1)}' is missing. Add it to the ZIP or install the required TeX package."
        latex_error = LATEX_ERROR_LINE.search(output)
        if latex_error:
            detail = " ".join(latex_error.group(1).split())
            source_line = LATEX_SOURCE_LINE.search(output, latex_error.end())
            if source_line:
                detail += f" Near: {' '.join(source_line.group(1).split())}"
            return f"LaTeX compilation failed: {detail[:600]}"
    return "The template could not be rendered as PDF. Check its packages, fonts, or document structure."


def render_preview(name: str, data: bytes, entry_file: str = "") -> bytes:
    suffix = Path(name).suffix.lower()
    if suffix not in {".tex", ".zip", ".doc", ".docx"}:
        raise ExtractionError("unsupported_preview_format", "Preview supports TeX, LaTeX ZIP, DOC, and DOCX files.")
    with tempfile.TemporaryDirectory(prefix="resumegpt-preview-") as directory:
        workspace = Path(directory)
        source = workspace / f"template{suffix}"
        output = workspace / "output"
        home = workspace / "home"
        output.mkdir()
        home.mkdir()
        source.write_bytes(data)
        environment = os.environ.copy()
        environment["HOME"] = str(home)
        environment["openin_any"] = "p"
        environment["openout_any"] = "p"
        if suffix in {".tex", ".zip"}:
            # xelatex (rather than pdflatex) is required so templates can load
            # fontspec/xeCJK and render non-Latin scripts such as Chinese.
            executable = shutil.which("xelatex")
            if executable is None:
                raise ExtractionError("tex_renderer_unavailable", "LaTeX preview rendering is unavailable.")
            if suffix == ".zip":
                source = extract_latex_archive(data, workspace / "latex-source", entry_file)
            command = [executable, "-no-shell-escape", "-interaction=nonstopmode", "-halt-on-error", f"-output-directory={output}", source.name]
        else:
            executable = shutil.which("libreoffice")
            if executable is None:
                raise ExtractionError("word_renderer_unavailable", "Word preview rendering is unavailable.")
            profile = workspace / "libreoffice-profile"
            command = [executable, f"-env:UserInstallation={profile.as_uri()}", "--headless", "--nologo", "--nodefault", "--norestore", "--nolockcheck", "--convert-to", "pdf", "--outdir", str(output), str(source)]
        try:
            completed = subprocess.run(command, cwd=source.parent if suffix in {".tex", ".zip"} else workspace, capture_output=True, timeout=30, check=False, env=environment)
        except subprocess.TimeoutExpired as error:
            raise ExtractionError("preview_timeout", "Template preview rendering exceeded 30 seconds.") from error
        pdf = output / f"{source.stem}.pdf"
        if completed.returncode != 0 or not pdf.is_file():
            raise ExtractionError("preview_render_failed", _render_failure_message(completed, suffix in {".tex", ".zip"}))
        result = pdf.read_bytes()
        if not result.startswith(b"%PDF-") or len(result) > MAX_PREVIEW_BYTES:
            raise ExtractionError("invalid_preview", "The renderer did not produce a valid PDF preview.")
        return result

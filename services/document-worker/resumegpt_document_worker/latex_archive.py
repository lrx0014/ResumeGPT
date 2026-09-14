from __future__ import annotations

import io
import shutil
import stat
import zipfile
from pathlib import Path, PurePosixPath

from .extractor import ExtractionError

MAX_ARCHIVE_FILES = 200
MAX_ARCHIVE_FILE_BYTES = 10 * 1024 * 1024
MAX_ARCHIVE_EXPANDED_BYTES = 25 * 1024 * 1024


def normalize_entry_file(value: str) -> str:
    value = value.strip().replace("\\", "/")
    path = PurePosixPath(value)
    if not value or value.startswith("/") or any(part in {"", ".", ".."} for part in path.parts) or path.suffix.lower() != ".tex":
        raise ExtractionError("invalid_template_entry", "The LaTeX entry file must be a relative .tex path inside the ZIP archive.")
    return path.as_posix()


def extract_latex_archive(data: bytes, destination: Path, requested_entry: str = "") -> Path:
    try:
        archive = zipfile.ZipFile(io.BytesIO(data))
    except zipfile.BadZipFile as error:
        raise ExtractionError("invalid_latex_archive", "The uploaded LaTeX ZIP archive is invalid.") from error
    with archive:
        files = [item for item in archive.infolist() if not item.is_dir()]
        if not files or len(files) > MAX_ARCHIVE_FILES:
            raise ExtractionError("archive_file_limit_exceeded", "LaTeX ZIP archives must contain between 1 and 200 files.")
        total_size = 0
        names: list[str] = []
        for item in files:
            name = item.filename.replace("\\", "/")
            path = PurePosixPath(name)
            mode = item.external_attr >> 16
            if name.startswith("/") or any(part in {"", ".", ".."} for part in path.parts) or stat.S_ISLNK(mode):
                raise ExtractionError("unsafe_archive_path", "The LaTeX ZIP archive contains an unsafe file path.")
            if item.flag_bits & 0x1:
                raise ExtractionError("encrypted_archive", "Encrypted LaTeX ZIP archives are not supported.")
            if item.file_size > MAX_ARCHIVE_FILE_BYTES:
                raise ExtractionError("archive_file_size_exceeded", "A file in the LaTeX ZIP archive exceeds 10 MiB.")
            total_size += item.file_size
            if total_size > MAX_ARCHIVE_EXPANDED_BYTES:
                raise ExtractionError("expanded_size_limit_exceeded", "The expanded LaTeX ZIP archive exceeds 25 MiB.")
            names.append(path.as_posix())

        entry = _select_entry(names, requested_entry)
        destination.mkdir(parents=True, exist_ok=True)
        root = destination.resolve()
        for item, name in zip(files, names, strict=True):
            target = (destination / name).resolve()
            if root not in target.parents:
                raise ExtractionError("unsafe_archive_path", "The LaTeX ZIP archive contains an unsafe file path.")
            target.parent.mkdir(parents=True, exist_ok=True)
            with archive.open(item) as source, target.open("wb") as output:
                shutil.copyfileobj(source, output)
        return destination / entry


def _select_entry(names: list[str], requested_entry: str) -> str:
    tex_files = [name for name in names if PurePosixPath(name).suffix.lower() == ".tex"]
    if requested_entry:
        entry = normalize_entry_file(requested_entry)
        if entry not in names:
            raise ExtractionError("template_entry_not_found", "The specified LaTeX entry file was not found in the ZIP archive.")
        return entry
    if "main.tex" in names:
        return "main.tex"
    main_files = [name for name in tex_files if PurePosixPath(name).name.lower() == "main.tex"]
    if len(main_files) == 1:
        return main_files[0]
    if len(tex_files) == 1:
        return tex_files[0]
    raise ExtractionError("template_entry_required", "Specify the LaTeX entry file because the ZIP archive does not have an unambiguous main.tex.")

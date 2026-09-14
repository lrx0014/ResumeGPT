from __future__ import annotations

import json
import tempfile
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import unquote_plus

from .extractor import MAX_DOCUMENT_BYTES, ExtractionError, extract, malware_scanner
from .preview import render_preview


class ExtractionHandler(BaseHTTPRequestHandler):
    server_version = "ResumeGPTDocumentWorker/1"

    def do_GET(self) -> None:
        if self.path != "/healthz":
            self.send_error(HTTPStatus.NOT_FOUND)
            return
        try:
            malware_scanner()
            self._write_json(HTTPStatus.OK, {"status": "ok"})
        except ExtractionError as error:
            self._write_error(HTTPStatus.SERVICE_UNAVAILABLE, error.code, str(error))

    def do_POST(self) -> None:
        if self.path not in {"/v1/extractions", "/v1/previews"}:
            self.send_error(HTTPStatus.NOT_FOUND)
            return
        try:
            content_length = int(self.headers.get("Content-Length", "0"))
        except ValueError:
            content_length = 0
        if content_length < 1 or content_length > MAX_DOCUMENT_BYTES:
            self._write_error(HTTPStatus.REQUEST_ENTITY_TOO_LARGE, "size_limit_exceeded", "Documents must be between 1 byte and 10 MiB.")
            return
        name = Path(unquote_plus(self.headers.get("X-Document-Name", ""))).name
        entry_file = unquote_plus(self.headers.get("X-Template-Entry", ""))
        if not name or name in {".", ".."}:
            self._write_error(HTTPStatus.BAD_REQUEST, "filename_required", "A document filename is required.")
            return
        data = self.rfile.read(content_length)
        if len(data) != content_length:
            self._write_error(HTTPStatus.BAD_REQUEST, "incomplete_document", "The document body is incomplete.")
            return
        suffix = Path(name).suffix[:16]
        try:
            with tempfile.NamedTemporaryFile(suffix=suffix) as temporary:
                temporary.write(data)
                temporary.flush()
                if self.path == "/v1/previews":
                    preview = render_preview(name, data, entry_file)
                else:
                    result = extract(Path(temporary.name), entry_file=entry_file)
            if self.path == "/v1/previews":
                self._write_bytes(HTTPStatus.OK, "application/pdf", preview)
            else:
                self._write_raw_json(HTTPStatus.OK, result.to_json())
        except ExtractionError as error:
            status = HTTPStatus.SERVICE_UNAVAILABLE if error.code in {
                "scanner_unavailable", "scanner_definitions_stale", "scanner_timeout", "scanner_failed",
                "converter_unavailable", "pdf_engine_unavailable", "ocr_unavailable",
                "tex_renderer_unavailable", "word_renderer_unavailable",
            } else HTTPStatus.UNPROCESSABLE_ENTITY
            self._write_error(status, error.code, str(error))
        except Exception:
            self._write_error(HTTPStatus.INTERNAL_SERVER_ERROR, "internal_error", "Document extraction failed unexpectedly.")

    def log_message(self, format: str, *args: object) -> None:
        print(json.dumps({"level": "info", "message": format % args}, separators=(",", ":")), flush=True)

    def _write_error(self, status: HTTPStatus, code: str, message: str) -> None:
        self._write_json(status, {"error": {"code": code, "message": message}})

    def _write_json(self, status: HTTPStatus, value: object) -> None:
        self._write_raw_json(status, json.dumps(value, separators=(",", ":")))

    def _write_raw_json(self, status: HTTPStatus, value: str) -> None:
        self._write_bytes(status, "application/json", value.encode("utf-8"))

    def _write_bytes(self, status: HTTPStatus, content_type: str, encoded: bytes) -> None:
        self.send_response(status)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)


def serve(address: str) -> None:
    host, separator, port = address.rpartition(":")
    if not separator or not port.isdigit():
        raise ValueError("The server address must use host:port syntax.")
    server = ThreadingHTTPServer((host, int(port)), ExtractionHandler)
    server.serve_forever()

from __future__ import annotations

import json
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from .browser import BrowserError, render_page

MAX_REQUEST_BYTES = 64 * 1024


class BrowserHandler(BaseHTTPRequestHandler):
    server_version = "ResumeGPTWebWorker/1"

    def do_GET(self) -> None:
        if self.path != "/healthz":
            self.send_error(HTTPStatus.NOT_FOUND)
            return
        self._write_json(HTTPStatus.OK, {"status": "ok"})

    def do_POST(self) -> None:
        if self.path != "/v1/pages/render":
            self.send_error(HTTPStatus.NOT_FOUND)
            return
        try:
            content_length = int(self.headers.get("Content-Length", "0"))
        except ValueError:
            content_length = 0
        if content_length < 1 or content_length > MAX_REQUEST_BYTES:
            self._write_error(HTTPStatus.REQUEST_ENTITY_TOO_LARGE, "request_too_large", "The browser request is too large.")
            return
        try:
            payload = json.loads(self.rfile.read(content_length))
            source_url = payload.get("url", "")
            actions = payload.get("actions", [])
            if not isinstance(actions, list):
                raise ValueError
        except (json.JSONDecodeError, UnicodeDecodeError, ValueError, AttributeError):
            self._write_error(HTTPStatus.BAD_REQUEST, "invalid_request", "The browser request is invalid.")
            return
        try:
            self._write_json(HTTPStatus.OK, render_page(source_url, actions))
        except BrowserError as error:
            status = HTTPStatus.SERVICE_UNAVAILABLE if error.retryable else HTTPStatus.UNPROCESSABLE_ENTITY
            self._write_error(status, error.code, str(error))
        except Exception:
            self._write_error(HTTPStatus.INTERNAL_SERVER_ERROR, "browser_failed", "The public job page could not be rendered.")

    def log_message(self, format: str, *args: object) -> None:
        print(json.dumps({"level": "info", "message": format % args}, separators=(",", ":")), flush=True)

    def _write_error(self, status: HTTPStatus, code: str, message: str) -> None:
        self._write_json(status, {"error": {"code": code, "message": message}})

    def _write_json(self, status: HTTPStatus, value: object) -> None:
        encoded = json.dumps(value, separators=(",", ":")).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)


def serve(address: str) -> None:
    host, separator, port = address.rpartition(":")
    if not separator or not port.isdigit():
        raise ValueError("The server address must use host:port syntax.")
    ThreadingHTTPServer((host, int(port)), BrowserHandler).serve_forever()

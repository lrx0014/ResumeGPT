import io
from email.message import Message

from resumegpt_document_worker import server
from resumegpt_document_worker.extractor import Result, Segment


class ExactLengthBody(io.BytesIO):
    def read(self, size: int = -1) -> bytes:
        assert size == 8
        return super().read(size)


def test_extraction_handler_reads_exact_content_length(monkeypatch) -> None:
    monkeypatch.setattr(server, "extract", lambda _, entry_file="": Result(
        media_type="text/plain",
        sha256="a" * 64,
        parser_version="document-extractor-v1",
        malware_status="clean",
        segments=[Segment(text="Evidence", page=None, paragraph=1, confidence=1)],
    ))
    headers = Message()
    headers["Content-Length"] = "8"
    headers["X-Document-Name"] = "resume.txt"
    handler = server.ExtractionHandler.__new__(server.ExtractionHandler)
    handler.path = "/v1/extractions"
    handler.headers = headers
    handler.rfile = ExactLengthBody(b"evidence")
    handler.wfile = io.BytesIO()
    handler.request_version = "HTTP/1.1"
    handler.command = "POST"
    handler.requestline = "POST /v1/extractions HTTP/1.1"
    handler.client_address = ("127.0.0.1", 1234)
    handler.server = object()

    handler.do_POST()

    assert b'"malware_status":"clean"' in handler.wfile.getvalue()

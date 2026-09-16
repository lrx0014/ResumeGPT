from __future__ import annotations

import socket

import pytest

from resumegpt_web_worker.security import URLSecurityError, validate_public_url


def test_rejects_non_https_and_credentials() -> None:
    for value in ("http://example.com/job", "https://user:pass@example.com/job", "file:///etc/passwd"):
        with pytest.raises(URLSecurityError):
            validate_public_url(value)


def test_rejects_private_addresses(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(socket, "getaddrinfo", lambda *args, **kwargs: [(socket.AF_INET, socket.SOCK_STREAM, 6, "", ("127.0.0.1", 443))])
    with pytest.raises(URLSecurityError):
        validate_public_url("https://example.com/job")


def test_accepts_public_https(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(socket, "getaddrinfo", lambda *args, **kwargs: [(socket.AF_INET, socket.SOCK_STREAM, 6, "", ("8.8.8.8", 443))])
    assert validate_public_url("https://example.com/job#details") == "https://example.com/job"

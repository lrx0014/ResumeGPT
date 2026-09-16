from __future__ import annotations

import ipaddress
import socket
from urllib.parse import urlsplit, urlunsplit


class URLSecurityError(ValueError):
    pass


def validate_public_url(value: str, *, initial: bool = False) -> str:
    if not isinstance(value, str) or len(value) > 2048:
        raise URLSecurityError("The page URL is invalid.")
    parsed = urlsplit(value.strip())
    if parsed.scheme != "https" or not parsed.hostname or parsed.username or parsed.password:
        raise URLSecurityError("Only public HTTPS pages are supported.")
    if parsed.port not in {None, 443}:
        raise URLSecurityError("Only the standard HTTPS port is supported.")
    try:
        addresses = socket.getaddrinfo(parsed.hostname, 443, type=socket.SOCK_STREAM)
    except socket.gaierror as error:
        raise URLSecurityError("The page hostname could not be resolved.") from error
    if not addresses:
        raise URLSecurityError("The page hostname could not be resolved.")
    for address in addresses:
        ip = ipaddress.ip_address(address[4][0])
        if not ip.is_global:
            raise URLSecurityError("The page hostname resolves to a non-public address.")
    fragmentless = parsed._replace(fragment="")
    return urlunsplit(fragmentless)

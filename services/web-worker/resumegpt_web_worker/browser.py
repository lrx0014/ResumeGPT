from __future__ import annotations

from typing import Any
from urllib.parse import urlsplit

from playwright.sync_api import Route, TimeoutError as PlaywrightTimeoutError, sync_playwright

from .security import URLSecurityError, validate_public_url

MAX_TEXT = 100_000
MAX_METADATA_VALUE = 20_000
MAX_ACTIONS = 6


class BrowserError(RuntimeError):
    def __init__(self, code: str, message: str, retryable: bool = False) -> None:
        super().__init__(message)
        self.code = code
        self.retryable = retryable


def render_page(source_url: str, actions: list[dict[str, Any]]) -> dict[str, Any]:
    try:
        source_url = validate_public_url(source_url, initial=True)
    except URLSecurityError as error:
        raise BrowserError("unsupported_page_url", str(error)) from error
    if len(actions) > MAX_ACTIONS:
        raise BrowserError("browser_action_limit", "The AI browser action limit was exceeded.")

    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(
            headless=True,
            args=["--disable-dev-shm-usage", "--disable-extensions", "--no-first-run"],
        )
        context = browser.new_context(
            java_script_enabled=True,
            ignore_https_errors=False,
            accept_downloads=False,
            service_workers="block",
            user_agent="ResumeGPT/0.1 AI job importer (+public pages only)",
            viewport={"width": 1365, "height": 900},
        )
        page = context.new_page()
        page.set_default_timeout(4_000)

        def route_request(route: Route) -> None:
            request = route.request
            try:
                validate_public_url(request.url)
            except (URLSecurityError, ValueError):
                route.abort("blockedbyclient")
                return
            if request.resource_type in {"image", "media", "font"}:
                route.abort("blockedbyclient")
                return
            route.continue_()

        page.route("**/*", route_request)
        try:
            response = page.goto(source_url, wait_until="domcontentloaded", timeout=25_000)
            if response is not None and response.status >= 400:
                if response.status in {401, 403}:
                    raise BrowserError(
                        "page_access_denied",
                        f"This site blocks automated access (HTTP {response.status}). Open the source page or enter the job details manually.",
                    )
                raise BrowserError("page_access_denied", f"The page returned HTTP status {response.status}.")
            page.wait_for_timeout(1_200)
            landing_host = urlsplit(page.url).hostname
            for action in actions:
                action_type = str(action.get("type", ""))
                if action_type == "scroll":
                    page.evaluate("window.scrollBy(0, Math.max(window.innerHeight * 0.85, 600))")
                    page.wait_for_timeout(700)
                elif action_type == "expand":
                    element_id = str(action.get("elementId", ""))
                    _annotate_controls(page)
                    if not element_id.startswith("interactive-"):
                        raise BrowserError("invalid_browser_action", "The requested page control is invalid.")
                    control = page.locator(f'[data-resumegpt-id="{element_id}"]')
                    if control.count() != 1:
                        raise BrowserError("page_control_changed", "The requested page control is no longer available.")
                    control.click(timeout=3_000)
                    page.wait_for_timeout(800)
                    if urlsplit(page.url).hostname != landing_host:
                        raise BrowserError("navigation_blocked", "The page control attempted to navigate away from the inspected site.")
                else:
                    raise BrowserError("invalid_browser_action", "The requested browser action is not supported.")
            validate_public_url(page.url)
            return _snapshot(page)
        except PlaywrightTimeoutError as error:
            raise BrowserError("page_timeout", "The public job page took too long to load.", retryable=True) from error
        finally:
            context.close()
            browser.close()


def _annotate_controls(page: Any) -> list[dict[str, str]]:
    return page.evaluate(
        """
        () => {
          const candidates = [...document.querySelectorAll('button, [role="button"], summary')]
            .filter(element => {
              const style = window.getComputedStyle(element);
              const rect = element.getBoundingClientRect();
              return style.visibility !== 'hidden' && style.display !== 'none' && rect.width > 0 && rect.height > 0;
            })
            .slice(0, 80);
          return candidates.map((element, index) => {
            const id = `interactive-${index}`;
            element.setAttribute('data-resumegpt-id', id);
            return {
              id,
              role: element.getAttribute('role') || element.tagName.toLowerCase(),
              text: (element.innerText || element.getAttribute('aria-label') || element.getAttribute('title') || '').trim().slice(0, 300)
            };
          }).filter(item => item.text);
        }
        """
    )


def _snapshot(page: Any) -> dict[str, Any]:
    elements = _annotate_controls(page)
    value = page.evaluate(
        """
        () => {
          const metadata = {};
          for (const element of document.querySelectorAll('meta[name][content], meta[property][content]')) {
            const key = (element.getAttribute('property') || element.getAttribute('name') || '').toLowerCase();
            if (key && !(key in metadata)) metadata[key] = element.getAttribute('content') || '';
          }
          const jsonLd = [...document.querySelectorAll('script[type*="ld+json"]')].map(element => element.textContent || '');
          return {
            url: window.location.href,
            title: document.title || '',
            visibleText: document.body ? document.body.innerText || '' : '',
            metadata,
            jsonLd
          };
        }
        """
    )
    value["visibleText"] = value["visibleText"][:MAX_TEXT]
    value["metadata"] = {
        str(key)[:200]: str(content)[:MAX_METADATA_VALUE]
        for key, content in value["metadata"].items()
    }
    remaining = 60_000
    json_ld: list[str] = []
    for item in value["jsonLd"][:20]:
        encoded = str(item)[: min(MAX_METADATA_VALUE, remaining)]
        if not encoded:
            continue
        json_ld.append(encoded)
        remaining -= len(encoded)
        if remaining <= 0:
            break
    value["jsonLd"] = json_ld
    value["elements"] = elements
    return value

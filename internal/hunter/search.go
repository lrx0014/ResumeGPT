package hunter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/job"
	"golang.org/x/net/html"
)

type DuckDuckGoSearch struct{ client *http.Client }

func NewDuckDuckGoSearch() *DuckDuckGoSearch {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil || port != "443" || (host != "html.duckduckgo.com" && host != "duckduckgo.com") {
				return nil, errors.New("search host is not allowed")
			}
			addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil || len(addresses) == 0 {
				return nil, errors.New("could not resolve search host")
			}
			for _, resolved := range addresses {
				value, ok := netip.AddrFromSlice(resolved.IP)
				if !ok || !value.Unmap().IsGlobalUnicast() || value.IsPrivate() || value.IsLoopback() || value.IsLinkLocalUnicast() {
					return nil, errors.New("search host resolved to a non-public address")
				}
			}
			selected := addresses[0].IP
			for _, resolved := range addresses {
				if resolved.IP.To4() != nil {
					selected = resolved.IP
					break
				}
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(selected.String(), port))
		},
		ForceAttemptHTTP2: false, ResponseHeaderTimeout: 15 * time.Second, TLSHandshakeTimeout: 10 * time.Second,
		MaxResponseHeaderBytes: 64 * 1024,
	}
	client := &http.Client{Transport: transport, Timeout: 25 * time.Second}
	client.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) >= 3 || (request.URL.Hostname() != "html.duckduckgo.com" && request.URL.Hostname() != "duckduckgo.com") || request.URL.Scheme != "https" {
			return errors.New("search redirect is not allowed")
		}
		return nil
	}
	return &DuckDuckGoSearch{client: client}
}

func (s *DuckDuckGoSearch) Search(ctx context.Context, query string, maximum int) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("search query is required")
	}
	if maximum < 1 || maximum > 10 {
		maximum = 10
	}
	endpoint := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)
	var response *http.Response
	var requestErr error
	for attempt := 0; attempt < 3; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("Accept", "text/html,application/xhtml+xml")
		request.Header.Set("Accept-Language", "en-US,en;q=0.8")
		request.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ResumeGPT/0.1; personal job search assistant)")
		request.Header.Set("Connection", "close")
		response, requestErr = s.client.Do(request)
		if requestErr == nil {
			break
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		wait := time.Duration(attempt+1) * 200 * time.Millisecond
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}
	if requestErr != nil {
		return nil, fmt.Errorf("search request failed after 3 attempts: %w", requestErr)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("search provider returned status %d", response.StatusCode)
	}
	document, err := html.Parse(io.LimitReader(response.Body, 2*1024*1024))
	if err != nil {
		return nil, errors.New("search provider returned invalid HTML")
	}
	results := make([]SearchResult, 0, maximum)
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if len(results) >= maximum {
			return
		}
		if node.Type == html.ElementNode && hasClass(node, "result") {
			if result, ok := parseResult(node); ok {
				results = append(results, result)
			}
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(document)
	return results, nil
}

func parseResult(root *html.Node) (SearchResult, bool) {
	var result SearchResult
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "a" && hasClass(node, "result__a") && result.URL == "" {
			result.Title = strings.TrimSpace(nodeText(node))
			result.URL = resultURL(attribute(node, "href"))
		}
		if node.Type == html.ElementNode && hasClass(node, "result__snippet") && result.Snippet == "" {
			result.Snippet = strings.TrimSpace(nodeText(node))
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	if result.URL == "" {
		return SearchResult{}, false
	}
	return result, true
}

func resultURL(raw string) string {
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if (parsed.Hostname() == "duckduckgo.com" || parsed.Hostname() == "html.duckduckgo.com") && parsed.Query().Get("uddg") != "" {
		raw = parsed.Query().Get("uddg")
	}
	normalized, err := job.NormalizeAIImportURL(raw)
	if err != nil {
		return ""
	}
	return normalized
}

func hasClass(node *html.Node, name string) bool {
	for _, field := range strings.Fields(attribute(node, "class")) {
		if field == name {
			return true
		}
	}
	return false
}

func attribute(node *html.Node, name string) string {
	for _, value := range node.Attr {
		if value.Key == name {
			return value.Val
		}
	}
	return ""
}

func nodeText(node *html.Node) string {
	var value strings.Builder
	var walk func(*html.Node)
	walk = func(current *html.Node) {
		if current.Type == html.TextNode {
			value.WriteString(current.Data)
			value.WriteByte(' ')
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return strings.Join(strings.Fields(value.String()), " ")
}

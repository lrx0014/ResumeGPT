package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const maxJobPageBytes = 2 * 1024 * 1024

var nonPublicJobNetworks = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"), netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"), netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"), netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b::/96"), netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"), netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("fc00::/7"), netip.MustParsePrefix("fe80::/10"),
}

type FetchError struct {
	Code      string
	Message   string
	Retryable bool
}

func (e *FetchError) Error() string { return e.Message }

type Fetcher interface {
	Fetch(context.Context, string) (ParsedJob, error)
}

type HTTPFetcher struct {
	client *http.Client
}

func NewHTTPFetcher() *HTTPFetcher {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		DialContext:            publicDialContext(dialer),
		ForceAttemptHTTP2:      true,
		MaxIdleConns:           10,
		IdleConnTimeout:        30 * time.Second,
		TLSHandshakeTimeout:    10 * time.Second,
		ResponseHeaderTimeout:  15 * time.Second,
		MaxResponseHeaderBytes: 64 * 1024,
	}
	client := &http.Client{Transport: transport, Timeout: 30 * time.Second}
	client.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return errors.New("too many redirects")
		}
		_, err := NormalizeImportURL(request.URL.String())
		return err
	}
	return &HTTPFetcher{client: client}
}

func publicDialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil || port != "443" || !allowedJobHost(host) {
			return nil, ErrInvalidURL
		}
		addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil || len(addresses) == 0 {
			return nil, fmt.Errorf("resolve job host: %w", err)
		}
		for _, resolved := range addresses {
			if !publicIP(resolved.IP) {
				return nil, errors.New("job host resolved to a non-public address")
			}
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(addresses[0].IP.String(), port))
	}
}

func publicIP(ip net.IP) bool {
	address, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	address = address.Unmap()
	if !address.IsGlobalUnicast() {
		return false
	}
	for _, prefix := range nonPublicJobNetworks {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

func (f *HTTPFetcher) Fetch(ctx context.Context, sourceURL string) (ParsedJob, error) {
	normalized, err := NormalizeImportURL(sourceURL)
	if err != nil {
		return ParsedJob{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, normalized, nil)
	if err != nil {
		return ParsedJob{}, err
	}
	request.Header.Set("Accept", "text/html,application/xhtml+xml")
	request.Header.Set("Accept-Language", "en-US,en;q=0.8")
	request.Header.Set("User-Agent", "ResumeGPT/0.1 (+personal job tracker; public pages only)")
	response, err := f.client.Do(request)
	if err != nil {
		return ParsedJob{}, &FetchError{Code: "page_unavailable", Message: "The public job page could not be downloaded.", Retryable: true}
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500 {
		return ParsedJob{}, &FetchError{Code: "site_temporarily_unavailable", Message: "The job site is temporarily unavailable.", Retryable: true}
	}
	if response.StatusCode != http.StatusOK {
		return ParsedJob{}, &FetchError{Code: "page_access_denied", Message: "The job site did not expose this page publicly. Add the job manually instead."}
	}
	mediaType := strings.ToLower(response.Header.Get("Content-Type"))
	if !strings.HasPrefix(mediaType, "text/html") && !strings.HasPrefix(mediaType, "application/xhtml+xml") {
		return ParsedJob{}, &FetchError{Code: "unsupported_page", Message: "The URL did not return an HTML job page."}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxJobPageBytes+1))
	if err != nil {
		return ParsedJob{}, &FetchError{Code: "page_read_failed", Message: "The job page could not be read.", Retryable: true}
	}
	if len(body) > maxJobPageBytes {
		return ParsedJob{}, &FetchError{Code: "page_too_large", Message: "The job page exceeds the 2 MiB import limit."}
	}
	parsed, err := parseJobPage(body)
	if err != nil {
		return ParsedJob{}, &FetchError{Code: "page_parse_failed", Message: "Job details could not be extracted. Edit the imported job manually."}
	}
	return parsed, nil
}

func parseJobPage(source []byte) (ParsedJob, error) {
	document, err := html.Parse(strings.NewReader(string(source)))
	if err != nil {
		return ParsedJob{}, err
	}
	metadata := make(map[string]string)
	var pageTitle string
	var postings []map[string]any
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "meta" {
			key, content := attribute(node, "property"), attribute(node, "content")
			if key == "" {
				key = attribute(node, "name")
			}
			if key != "" && content != "" {
				metadata[strings.ToLower(key)] = strings.TrimSpace(content)
			}
		}
		if node.Type == html.ElementNode && node.Data == "title" && node.FirstChild != nil {
			pageTitle = strings.TrimSpace(node.FirstChild.Data)
		}
		if node.Type == html.ElementNode && node.Data == "script" && strings.Contains(strings.ToLower(attribute(node, "type")), "ld+json") {
			var payload any
			if json.Unmarshal([]byte(nodeText(node)), &payload) == nil {
				postings = append(postings, findJobPostings(payload)...)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(document)
	if len(postings) > 0 {
		return parsedPosting(postings[0]), nil
	}
	result := ParsedJob{Title: first(metadata["og:title"], metadata["twitter:title"], pageTitle),
		Description: cleanHTML(first(metadata["og:description"], metadata["description"]))}
	result.Title, result.Company, result.Location = splitFallbackTitle(result.Title)
	if result.Title == "" {
		return ParsedJob{}, errors.New("job title not found")
	}
	return result, nil
}

func findJobPostings(value any) []map[string]any {
	var result []map[string]any
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			result = append(result, findJobPostings(item)...)
		}
	case map[string]any:
		if hasSchemaType(typed["@type"], "jobposting") {
			result = append(result, typed)
		}
		for _, item := range typed {
			result = append(result, findJobPostings(item)...)
		}
	}
	return result
}

func parsedPosting(value map[string]any) ParsedJob {
	result := ParsedJob{Title: stringValue(value["title"]), Description: cleanHTML(stringValue(value["description"])),
		EmploymentType: stringValue(value["employmentType"])}
	if organization, ok := value["hiringOrganization"].(map[string]any); ok {
		result.Company = stringValue(organization["name"])
	}
	location := firstMap(value["jobLocation"])
	address, _ := location["address"].(map[string]any)
	result.City = stringValue(address["addressLocality"])
	result.Country = stringValue(address["addressCountry"])
	region := stringValue(address["addressRegion"])
	result.Location = strings.Join(nonEmpty(result.City, region, result.Country), ", ")
	if strings.EqualFold(stringValue(value["jobLocationType"]), "TELECOMMUTE") {
		result.WorkMode = "Remote"
	}
	return result
}

func hasSchemaType(value any, expected string) bool {
	switch typed := value.(type) {
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), expected)
	case []any:
		for _, item := range typed {
			if hasSchemaType(item, expected) {
				return true
			}
		}
	}
	return false
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := stringValue(item); text != "" {
				values = append(values, text)
			}
		}
		return strings.Join(values, ", ")
	case map[string]any:
		return stringValue(typed["name"])
	default:
		return ""
	}
}

func firstMap(value any) map[string]any {
	if result, ok := value.(map[string]any); ok {
		return result
	}
	if values, ok := value.([]any); ok && len(values) > 0 {
		result, _ := values[0].(map[string]any)
		return result
	}
	return nil
}

func splitFallbackTitle(value string) (string, string, string) {
	parts := strings.Split(value, " | ")
	if len(parts) >= 3 && strings.Contains(strings.ToLower(parts[len(parts)-1]), "linkedin") {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), ""
	}
	base := strings.TrimSpace(strings.Split(value, " | Indeed")[0])
	parts = strings.Split(base, " - ")
	if len(parts) >= 3 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), strings.TrimSpace(strings.Join(parts[2:], " - "))
	}
	return value, "", ""
}

func cleanHTML(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	fragment, err := html.Parse(strings.NewReader(value))
	if err != nil {
		return strings.TrimSpace(value)
	}
	return strings.Join(strings.Fields(nodeText(fragment)), " ")
}

func nodeText(node *html.Node) string {
	var builder strings.Builder
	var walk func(*html.Node)
	walk = func(current *html.Node) {
		if current.Type == html.TextNode {
			builder.WriteString(current.Data)
			builder.WriteByte(' ')
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return builder.String()
}

func attribute(node *html.Node, name string) string {
	for _, item := range node.Attr {
		if strings.EqualFold(item.Key, name) {
			return item.Val
		}
	}
	return ""
}

func first(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func nonEmpty(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, strings.TrimSpace(value))
		}
	}
	return result
}

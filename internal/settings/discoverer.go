package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strings"
	"time"
)

const maxModelsResponseBytes = 1024 * 1024

var nonPublicConnectionNetworks = []netip.Prefix{
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

var localCarrierGradeNetwork = netip.MustParsePrefix("100.64.0.0/10")

type HTTPModelDiscoverer struct{}

func NewHTTPModelDiscoverer() *HTTPModelDiscoverer { return &HTTPModelDiscoverer{} }

func (*HTTPModelDiscoverer) Models(ctx context.Context, connection LLMConnection, token string) ([]string, error) {
	endpoint, err := modelEndpoint(connection)
	if err != nil {
		return nil, err
	}
	client := connectionHTTPClient(connection)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, ErrConnectionFailed
	}
	request.Header.Set("Accept", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: the endpoint could not be reached", ErrConnectionFailed)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w: authentication was rejected", ErrConnectionFailed)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: the endpoint returned status %d", ErrConnectionFailed, response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxModelsResponseBytes+1))
	if err != nil || len(body) > maxModelsResponseBytes {
		return nil, fmt.Errorf("%w: the model response could not be read", ErrConnectionFailed)
	}
	models, err := decodeModels(connection.Provider, body)
	if err != nil {
		return nil, fmt.Errorf("%w: the endpoint returned an unsupported model response", ErrConnectionFailed)
	}
	return models, nil
}

func modelEndpoint(connection LLMConnection) (string, error) {
	base, err := url.Parse(connection.BaseURL)
	if err != nil {
		return "", ErrInvalid
	}
	if connection.Provider == "ollama" {
		base.Path = strings.TrimRight(base.Path, "/") + "/api/tags"
	} else {
		base.Path = strings.TrimRight(base.Path, "/") + "/models"
	}
	return base.String(), nil
}

func connectionHTTPClient(connection LLMConnection) *http.Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 15 * time.Second}
	transport := &http.Transport{
		DialContext: connectionDialContext(dialer, connection), ForceAttemptHTTP2: true,
		TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 8 * time.Second,
		MaxResponseHeaderBytes: 32 * 1024,
	}
	return &http.Client{Transport: transport, Timeout: 12 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return errors.New("LLM endpoint redirects are not allowed")
	}}
}

func connectionDialContext(dialer *net.Dialer, connection LLMConnection) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, ErrConnectionFailed
		}
		expected, _ := url.Parse(connection.BaseURL)
		if !strings.EqualFold(strings.TrimSuffix(host, "."), strings.TrimSuffix(expected.Hostname(), ".")) {
			return nil, ErrConnectionFailed
		}
		addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil || len(addresses) == 0 {
			return nil, ErrConnectionFailed
		}
		for _, resolved := range addresses {
			if connection.ExecutionMode == "cloud" && !isPublicAddress(resolved.IP) ||
				connection.ExecutionMode == "local" && !isLocalAddress(resolved.IP) {
				return nil, ErrConnectionFailed
			}
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(addresses[0].IP.String(), port))
	}
}

func isPublicAddress(ip net.IP) bool {
	address, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	address = address.Unmap()
	if !address.IsGlobalUnicast() {
		return false
	}
	for _, prefix := range nonPublicConnectionNetworks {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

func isLocalAddress(ip net.IP) bool {
	address, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	address = address.Unmap()
	return address.IsPrivate() || address.IsLoopback() || localCarrierGradeNetwork.Contains(address)
}

func decodeModels(provider string, source []byte) ([]string, error) {
	var names []string
	if provider == "ollama" {
		var payload struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if err := json.Unmarshal(source, &payload); err != nil {
			return nil, err
		}
		for _, model := range payload.Models {
			names = append(names, model.Name)
		}
	} else {
		var payload struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(source, &payload); err != nil {
			return nil, err
		}
		for _, model := range payload.Data {
			names = append(names, model.ID)
		}
	}
	unique := make(map[string]bool)
	result := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" && !unique[name] {
			unique[name] = true
			result = append(result, name)
		}
	}
	sort.Strings(result)
	return result, nil
}

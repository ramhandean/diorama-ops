package favicon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrPrivateIP     = errors.New("request to private or local IP blocked")
	ErrTooManyRedirects = errors.New("too many redirects")
	ErrInvalidURL    = errors.New("invalid target url")
)

type Service struct {
	cacheDir string
	client   *http.Client
	mu       sync.RWMutex
	memory   map[string][]byte
}

func NewService(cacheDir string) *Service {
	_ = os.MkdirAll(cacheDir, 0755)

	// Safe dialer that rejects private/loopback/link-local IPs (Anti-SSRF)
	safeDialer := &net.Dialer{
		Timeout:   3 * time.Second,
		KeepAlive: 10 * time.Second,
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
			if err != nil {
				return nil, err
			}

			for _, ip := range ips {
				if isPrivateOrLocal(ip) {
					return nil, fmt.Errorf("%w: %s", ErrPrivateIP, ip.String())
				}
			}

			// Dial first valid non-private IP
			return safeDialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
		},
		ResponseHeaderTimeout: 3 * time.Second,
		DisableKeepAlives:     true,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return ErrTooManyRedirects
			}
			// Validate redirect target IP
			host := req.URL.Hostname()
			ips, err := net.LookupIP(host)
			if err != nil {
				return err
			}
			for _, ip := range ips {
				if isPrivateOrLocal(ip) {
					return ErrPrivateIP
				}
			}
			return nil
		},
	}

	return &Service{
		cacheDir: cacheDir,
		client:   client,
		memory:   make(map[string][]byte),
	}
}

func (s *Service) FetchFavicon(ctx context.Context, siteURL string) ([]byte, string, error) {
	u, err := url.Parse(siteURL)
	if err != nil || u.Host == "" {
		return nil, "", ErrInvalidURL
	}

	host := strings.ToLower(u.Hostname())

	// Check in-memory cache
	s.mu.RLock()
	cached, ok := s.memory[host]
	s.mu.RUnlock()
	if ok {
		return cached, detectContentType(cached), nil
	}

	// Check disk cache
	diskPath := filepath.Join(s.cacheDir, host+".ico")
	if data, err := os.ReadFile(diskPath); err == nil {
		s.mu.Lock()
		s.memory[host] = data
		s.mu.Unlock()
		return data, detectContentType(data), nil
	}

	// Fetch directly from domain/favicon.ico
	targetURL := fmt.Sprintf("%s://%s/favicon.ico", u.Scheme, u.Host)
	if u.Scheme == "" {
		targetURL = fmt.Sprintf("https://%s/favicon.ico", u.Host)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "DioramaOps/1.0 (+https://github.com/ramhandean/diorama-ops)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("upstream status: %d", resp.StatusCode)
	}

	// Limit to max 128 KB
	data, err := io.ReadAll(io.LimitReader(resp.Body, 128*1024))
	if err != nil {
		return nil, "", err
	}

	if len(data) == 0 {
		return nil, "", errors.New("empty favicon response")
	}

	// Save to disk & memory cache
	_ = os.WriteFile(diskPath, data, 0644)
	s.mu.Lock()
	s.memory[host] = data
	s.mu.Unlock()

	return data, detectContentType(data), nil
}

func isPrivateOrLocal(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	// Extra CGNAT check 100.64.0.0/10
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 100 && (ip4[1]&0xC0) == 64 {
			return true
		}
	}
	return false
}

func detectContentType(data []byte) string {
	if len(data) >= 4 && data[0] == 0 && data[1] == 0 && data[2] == 1 && data[3] == 0 {
		return "image/x-icon"
	}
	if len(data) >= 8 && string(data[1:4]) == "PNG" {
		return "image/png"
	}
	if len(data) >= 4 && string(data[:4]) == "<svg" || strings.Contains(string(data[:min(len(data), 100)]), "<svg") {
		return "image/svg+xml"
	}
	return "image/x-icon"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

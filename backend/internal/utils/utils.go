package utils

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

func AllowCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func GetBaseDomain(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	host := u.Hostname()

	domain, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return host, nil
	}

	return domain, nil
}

func IsPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	if ip4 := ip.To4(); ip4 != nil {
		privateRanges4 := []struct {
			start net.IP
			end   net.IP
		}{
			{net.ParseIP("10.0.0.0"), net.ParseIP("10.255.255.255")},
			{net.ParseIP("172.16.0.0"), net.ParseIP("172.31.255.255")},
			{net.ParseIP("192.168.0.0"), net.ParseIP("192.168.255.255")},
		}

		for _, r := range privateRanges4 {
			if bytesCompare(ip4, r.start.To4()) >= 0 && bytesCompare(ip4, r.end.To4()) <= 0 {
				return true
			}
		}
	}

	if ip6 := ip.To16(); ip6 != nil && ip.To4() == nil {
		ulaStart := net.ParseIP("fc00::")
		ulaEnd := net.ParseIP("fdff:ffff:ffff:ffff:ffff:ffff:ffff:ffff")

		if bytesCompare(ip6, ulaStart.To16()) >= 0 && bytesCompare(ip6, ulaEnd.To16()) <= 0 {
			return true
		}
	}

	return false
}

func SanitizeUTF8(s string) string {
	return strings.ToValidUTF8(s, "")
}

func bytesCompare(a, b []byte) int {
	if len(a) != len(b) {
		return 0
	}
	for i := 0; i < len(a); i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

type ClientConfig struct {
	Timeout       time.Duration
	AllowInternal bool
}

func NewSafeHTTPClient(cfg ClientConfig) *http.Client {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
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

			if !cfg.AllowInternal {
				for _, ip := range ips {
					if IsPrivateIP(ip) {
						return nil, fmt.Errorf("blocked connection to private IP: %s (SSRF Protection)", ip)
					}
				}
			}

			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
		},
	}

	return &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
	}
}

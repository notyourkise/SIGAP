package utils

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrBlockedAddress dikembalikan bila URL mengarah ke alamat internal
// (loopback/privat/link-local) yang tidak boleh diakses (mitigasi SSRF).
var ErrBlockedAddress = errors.New("URL mengarah ke alamat internal yang tidak diizinkan")

// ValidateExternalURL memastikan URL (umumnya webhook) memakai skema http/https
// dan tidak menunjuk ke alamat privat/loopback/link-local. Host di-resolve DNS
// agar seluruh alamat kandidat diperiksa sebelum request dikirim (mitigasi SSRF
// termasuk DNS rebinding dasar).
func ValidateExternalURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("URL tidak valid: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("URL harus memakai skema http atau https")
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("URL tidak memiliki host")
	}

	// Host berupa IP literal diperiksa langsung; selain itu resolve DNS.
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return ErrBlockedAddress
		}
		return nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("gagal me-resolve host: %w", err)
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return ErrBlockedAddress
		}
	}
	return nil
}

// isBlockedIP mendeteksi alamat IP yang tidak boleh dijangkau oleh webhook.
func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		// 0.0.0.0/8
		if ip4[0] == 0 {
			return true
		}
		// 100.64.0.0/10 (CGNAT)
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
	}
	return false
}

// NewSafeHTTPClient membuat *http.Client dengan proteksi SSRF:
//   - DialContext memvalidasi setiap alamat hasil resolusi DNS (menolak internal)
//     dan menyambung ke IP literal yang sudah divalidasi (anti DNS rebinding),
//   - CheckRedirect membatasi jumlah redirect dan memvalidasi ulang tujuannya.
func NewSafeHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
				if err != nil {
					return nil, err
				}
				for _, ipAddr := range ips {
					if isBlockedIP(ipAddr.IP) {
						return nil, ErrBlockedAddress
					}
				}
				var lastErr error
				for _, ipAddr := range ips {
					conn, derr := dialer.DialContext(ctx, network, net.JoinHostPort(ipAddr.IP.String(), port))
					if derr == nil {
						return conn, nil
					}
					lastErr = derr
				}
				if lastErr == nil {
					lastErr = ErrBlockedAddress
				}
				return nil, lastErr
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("terlalu banyak redirect")
			}
			return ValidateExternalURL(req.URL.String())
		},
	}
}

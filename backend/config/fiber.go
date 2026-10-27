package config

import (
	"errors"
	"net"
	"strings"

	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
)

// ParseTrustedProxies mem-parse nilai environment TRUSTED_PROXY_IPS menjadi
// daftar IP/CIDR yang tervalidasi dan ternormalisasi. Entri kosong diabaikan,
// sedangkan entri yang bukan alamat IP atau CIDR valid dibuang dengan warning
// agar salah konfigurasi tidak diam-diam mempercayai nilai berbahaya.
//
// Contoh: "172.28.0.10, 10.0.0.0/8, ::1" -> ["172.28.0.10", "10.0.0.0/8", "::1"].
func ParseTrustedProxies(raw string) []string {
	proxies := make([]string, 0)
	seen := make(map[string]struct{})

	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		normalized, err := normalizeTrustedProxy(entry)
		if err != nil {
			utils.Log.Warnf("⚠️ TRUSTED_PROXY_IPS mengabaikan entri tidak valid %q: %v", entry, err)
			continue
		}

		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		proxies = append(proxies, normalized)
	}

	return proxies
}

// normalizeTrustedProxy memvalidasi satu entri IP atau CIDR dan mengembalikan
// bentuk kanoniknya. Bentuk kanonik penting karena Fiber mencocokkan IP peer
// secara persis untuk entri non-CIDR (mis. "::ffff:172.28.0.10" harus menjadi
// "172.28.0.10" agar cocok dengan RemoteIP()).
func normalizeTrustedProxy(entry string) (string, error) {
	if strings.Contains(entry, "/") {
		_, network, err := net.ParseCIDR(entry)
		if err != nil {
			return "", errors.New("CIDR tidak valid")
		}
		return network.String(), nil
	}

	ip := net.ParseIP(entry)
	if ip == nil {
		return "", errors.New("bukan alamat IP atau CIDR yang valid")
	}
	return ip.String(), nil
}

// NewFiberConfig membangun konfigurasi Fiber dengan validasi IP aktif.
//
// Header proxy (X-Forwarded-For) hanya diaktifkan bila ada minimal satu
// trusted proxy valid. Tanpa trusted proxy, ProxyHeader sengaja dibiarkan
// kosong sehingga c.IP() selalu memakai IP koneksi langsung dan header yang
// dipalsukan klien tidak berpengaruh pada rate limit maupun audit log.
func NewFiberConfig(trustedProxies []string) fiber.Config {
	cfg := fiber.Config{
		// Selalu validasi header IP agar nilai sampah tidak pernah dipakai
		// sebagai identitas klien.
		EnableIPValidation: true,
	}

	if len(trustedProxies) > 0 {
		cfg.EnableTrustedProxyCheck = true
		cfg.TrustedProxies = trustedProxies
		cfg.ProxyHeader = fiber.HeaderXForwardedFor
	}

	return cfg
}

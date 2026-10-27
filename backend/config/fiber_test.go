package config

import (
	"io"
	"net/http/httptest"
	"testing"

	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTrustedProxies(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "kosong", raw: "", want: []string{}},
		{name: "hanya pemisah", raw: " , ,, ", want: []string{}},
		{
			name: "ipv4 dan ipv6",
			raw:  " 172.28.0.10 , ::1 ",
			want: []string{"172.28.0.10", "::1"},
		},
		{
			name: "cidr dinormalisasi",
			raw:  "10.0.0.42/8, 2001:db8::1/32",
			want: []string{"10.0.0.0/8", "2001:db8::/32"},
		},
		{
			name: "entri tidak valid dibuang",
			raw:  "bukan-ip, 999.1.1.1, 10.0.0.0/33, 172.28.0.10",
			want: []string{"172.28.0.10"},
		},
		{
			name: "duplikat dan bentuk setara dibuang",
			raw:  "172.28.0.10,172.28.0.10, ::ffff:172.28.0.10, 10.0.0.0/8, 10.1.2.3/8",
			want: []string{"172.28.0.10", "10.0.0.0/8"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseTrustedProxies(tt.raw))
		})
	}
}

func TestNewFiberConfig(t *testing.T) {
	t.Run("tanpa trusted proxy", func(t *testing.T) {
		cfg := NewFiberConfig(nil)

		assert.True(t, cfg.EnableIPValidation, "validasi IP harus selalu aktif")
		assert.False(t, cfg.EnableTrustedProxyCheck)
		assert.Empty(t, cfg.ProxyHeader, "header proxy tidak boleh dibaca tanpa trusted proxy")
		assert.Empty(t, cfg.TrustedProxies)
	})

	t.Run("dengan trusted proxy", func(t *testing.T) {
		proxies := []string{"172.28.0.10", "10.0.0.0/8"}
		cfg := NewFiberConfig(proxies)

		assert.True(t, cfg.EnableIPValidation)
		assert.True(t, cfg.EnableTrustedProxyCheck)
		assert.Equal(t, fiber.HeaderXForwardedFor, cfg.ProxyHeader)
		assert.Equal(t, proxies, cfg.TrustedProxies)
	})
}

// resolvedIP menjalankan request dengan header X-Forwarded-For tertentu lalu
// mengembalikan IP yang dipakai audit log (utils.ExtractClientIP). app.Test()
// memakai peer 0.0.0.0 sehingga IP tersebut dapat dipakai sebagai trusted proxy.
func resolvedIP(t *testing.T, cfg fiber.Config, forwardedFor string) string {
	t.Helper()

	app := fiber.New(cfg)
	app.Get("/ip", func(c *fiber.Ctx) error {
		return c.SendString(utils.ExtractClientIP(c))
	})

	req := httptest.NewRequest("GET", "/ip", nil)
	if forwardedFor != "" {
		req.Header.Set(fiber.HeaderXForwardedFor, forwardedFor)
	}

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(body)
}

func TestExtractClientIP_ProxyHardening(t *testing.T) {
	t.Run("klien langsung tidak bisa memalsukan X-Forwarded-For", func(t *testing.T) {
		assert.Equal(t, "0.0.0.0", resolvedIP(t, NewFiberConfig(nil), "203.0.113.99"))
	})

	t.Run("trusted proxy memakai X-Forwarded-For valid", func(t *testing.T) {
		assert.Equal(t, "203.0.113.9", resolvedIP(t, NewFiberConfig([]string{"0.0.0.0"}), "203.0.113.9"))
	})

	t.Run("trusted proxy melewati entri X-Forwarded-For tidak valid", func(t *testing.T) {
		header := "bukan-ip, 203.0.113.9, 10.0.0.1"
		assert.Equal(t, "203.0.113.9", resolvedIP(t, NewFiberConfig([]string{"0.0.0.0"}), header))
	})

	t.Run("trusted proxy dengan X-Forwarded-For seluruhnya invalid jatuh ke IP peer", func(t *testing.T) {
		assert.Equal(t, "0.0.0.0", resolvedIP(t, NewFiberConfig([]string{"0.0.0.0"}), "bukan-ip, 999.1.1.1"))
	})
}

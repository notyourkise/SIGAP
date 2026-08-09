package utils

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

// EmailConfig menampung konfigurasi SMTP yang dibaca dari environment variable.
type EmailConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	TLS      bool
}

// emailConfig adalah konfigurasi SMTP global yang dimuat sekali saat startup.
var emailConfig *EmailConfig

// InitEmailConfig membaca konfigurasi SMTP dari environment variable.
// Bila SMTP_HOST kosong, email dinonaktifkan (EmailEnabled() == false).
func InitEmailConfig() {
	cfg := &EmailConfig{
		Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
		Port:     587,
		User:     strings.TrimSpace(os.Getenv("SMTP_USER")),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
		TLS:      true,
	}

	if portStr := strings.TrimSpace(os.Getenv("SMTP_PORT")); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
			cfg.Port = p
		}
	}

	if tlsStr := strings.TrimSpace(os.Getenv("SMTP_TLS")); tlsStr != "" {
		cfg.TLS = tlsStr == "true" || tlsStr == "1" || strings.EqualFold(tlsStr, "yes")
	}

	if cfg.From == "" {
		cfg.From = "SIGAP <noreply@sigap.id>"
	}

	emailConfig = cfg
}

// EmailEnabled mengembalikan true bila server SMTP telah dikonfigurasi.
func EmailEnabled() bool {
	return emailConfig != nil && emailConfig.Host != ""
}

// SendEmail mengirim email HTML ke satu penerima melalui server SMTP yang
// dikonfigurasi. Mengembalikan error bila SMTP tidak aktif atau pengiriman gagal.
func SendEmail(to, subject, htmlBody string) error {
	if !EmailEnabled() {
		return fmt.Errorf("email SMTP tidak dikonfigurasi")
	}

	cfg := emailConfig
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))

	// Header MIME + body HTML.
	var msg bytes.Buffer
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	msg.WriteString("Subject: " + encodeHeader(subject) + "\r\n")
	msg.WriteString("From: " + cfg.From + "\r\n")
	msg.WriteString("To: " + to + "\r\n")
	msg.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	var auth smtp.Auth
	if cfg.User != "" {
		auth = smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)
	}

	// Mode TLS: gunakan STARTTLS bila server mendukung, atau koneksi TLS langsung
	// pada port khusus (465). Bila SMTP_TLS=false, kirim plaintext (port 25/587
	// tanpa STARTTLS) — hanya untuk development.
	if cfg.TLS {
		if err := sendWithTLS(addr, cfg, auth, to, msg.Bytes()); err != nil {
			return err
		}
		return nil
	}

	return smtp.SendMail(addr, auth, cfg.From, []string{to}, msg.Bytes())
}

// sendWithTLS mengirim email menggunakan koneksi SMTP dengan TLS.
// Mengutamakan STARTTLS (umum di port 587); bila gagal karena port TLS
// langsung (mis. 465), coba koneksi TLS implicit.
func sendWithTLS(addr string, cfg *EmailConfig, auth smtp.Auth, to string, body []byte) error {
	// Coba STARTTLS terlebih dahulu.
	client, err := smtp.Dial(addr)
	if err == nil {
		if err := tryStartTLS(client, addr, cfg, auth, to, body); err == nil {
			return nil
		}
		_ = client.Close()
	}

	// Fallback: koneksi TLS implicit (biasanya port 465).
	tlsConn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: cfg.Host})
	if err != nil {
		return err
	}
	client, err = smtp.NewClient(tlsConn, cfg.Host)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(cfg.From); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(body); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// tryStartTLS melakukan handshake STARTTLS lalu mengirim email.
func tryStartTLS(client *smtp.Client, addr string, cfg *EmailConfig, auth smtp.Auth, to string, body []byte) error {
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: cfg.Host}); err != nil {
			return err
		}
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(cfg.From); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(body); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// encodeHeader meng-encode subject berisi karakter non-ASCII agar aman pada
// header email (RFC 2047 quoted-printable).
func encodeHeader(value string) string {
	if isASCII(value) {
		return value
	}
	return "=?UTF-8?Q?" + quotedPrintable(value) + "?="
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

func quotedPrintable(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' {
			b.WriteByte('_')
		} else if c >= 0x80 || c == '=' || c == '?' || c == '_' {
			_, _ = fmt.Fprintf(&b, "=%02X", c)
		} else {
			b.WriteByte(c)
		}
	}
	return b.String()
}

// emailTemplateCandidates adalah lokasi folder template yang dicoba, disesuaikan
// dengan working directory saat runtime (backend/ di lokal, /app di container).
var emailTemplateCandidates = []string{
	"templates/",
	"../templates/",
	"backend/templates/",
	"../backend/templates/",
}

// RenderEmailTemplate memuat template email dari backend/templates dan
// me-render-nya dengan data yang diberikan. Template menggunakan html/template
// sehingga seluruh nilai ter-escape otomatis (anti XSS email).
func RenderEmailTemplate(name string, data interface{}) (string, error) {
	var tmpl *template.Template
	var err error
	found := false
	for _, dir := range emailTemplateCandidates {
		tmpl, err = template.ParseFiles(dir + name)
		if err == nil {
			found = true
			break
		}
	}
	if !found {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

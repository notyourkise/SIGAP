# Sertifikat TLS untuk reverse proxy Nginx

Folder ini berisi sertifikat TLS yang di-mount ke container Nginx
(`nginx/certs:/etc/nginx/certs:ro`). Nginx mengharapkan file:

- `sigap.crt`  — sertifikat (chain)
- `sigap.key`  — private key

## Membuat sertifikat self-signed (untuk pengujian / LAN internal)

```bash
mkdir -p nginx/certs
openssl req -x509 -nodes -newkey rsa:2048 \
  -keyout nginx/certs/sigap.key \
  -out nginx/certs/sigap.crt \
  -days 365 \
  -subj "/CN=localhost"
```

> Self-signed hanya cocok untuk pengujian. Untuk produksi gunakan sertifikat
> dari CA yang dipercaya (mis. Let's Encrypt via certbot) dan letakkan file
> `.pem` di folder ini, lalu sesuaikan `ssl_certificate` di `nginx/nginx.conf`
> bila nama file berbeda.

## Keamanan

- **JANGAN commit** file `.key` / `.crt` ke repository (sudah di-`.gitignore`).
- Private key bersifat rahasia; batasi akses dengan `chmod 600 sigap.key`.

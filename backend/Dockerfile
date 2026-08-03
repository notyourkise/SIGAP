# ==========================================
# Stage 1: Builder
# Build binary Go menggunakan image golang:alpine
# ==========================================
FROM golang:1.26-alpine AS builder

# Install dependensi sistem yang diperlukan untuk build CGo (libpq)
RUN apk add --no-cache git ca-certificates tzdata

# Tetapkan direktori kerja di dalam container builder
WORKDIR /build

# Salin file go.mod dan go.sum terlebih dahulu untuk memanfaatkan cache layer Docker
COPY go.mod go.sum ./

# Unduh semua dependensi Go (akan di-cache jika go.mod tidak berubah)
RUN go mod download

# Salin seluruh kode sumber ke dalam container builder
COPY . .

# Build binary dengan mengoptimalkan ukuran dan keamanan:
# - CGO_ENABLED=0: Nonaktifkan CGo agar binary murni Go (tidak perlu shared libs)
# - GOOS=linux: Target OS Linux untuk Alpine runner
# - -ldflags="-w -s": Hapus simbol debug agar ukuran binary lebih kecil
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o sigap-api .

# ==========================================
# Stage 2: Runner
# Image akhir yang sangat kecil menggunakan Alpine murni
# ==========================================
FROM alpine:latest AS runner

# Install sertifikat CA dan timezone data di image runner
# (Dibutuhkan untuk koneksi HTTPS eksternal dan logging WITA yang tepat)
RUN apk add --no-cache ca-certificates tzdata

# Set timezone default ke Asia/Makassar (WITA) agar log konsisten
ENV TZ=Asia/Makassar

# Tetapkan direktori kerja aplikasi di dalam runner
WORKDIR /app

# Salin binary yang sudah di-build dari stage builder
COPY --from=builder /build/sigap-api .

# Salin file docs swagger yang dibutuhkan saat runtime
COPY --from=builder /build/docs ./docs

# Salin aset statis (logo kop surat, dll.) yang dibutuhkan saat runtime
COPY --from=builder /build/assets ./assets

# Buat direktori uploads untuk lampiran foto pemeliharaan CCTV
RUN mkdir -p /app/uploads

# Expose port 3000 yang digunakan aplikasi Fiber
EXPOSE 3000

# Jalankan binary aplikasi SIGAP
CMD ["./sigap-api"]

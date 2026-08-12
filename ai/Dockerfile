# ==========================================
# SIGAP AI — Layanan Deteksi Orang (Task 14B, Sprint 38)
# ==========================================
# Service Python (FastAPI) untuk deteksi orang (YOLOv8n) + tracking (ByteTrack).
# Membaca stream RTSP internal MediaMTX (rtsp://mediamtx:8554/cam-{cctv_id}),
# menghitung orang unik per hari, lalu mengirim hasil ke sigap_api.
# Berjalan di port internal 8010 (tidak dipublish ke host).

FROM python:3.11-slim

# Dependensi sistem yang dibutuhkan OpenCV headless (libglib, libgl1/libGL,
# libsm, libxext, dll). libgl1 diperlukan untuk libGL.so.1 (dibutuhkan cv2 saat
# import ultralytics), sedangkan libgomp1 untuk OpenMP/threading numpy.
RUN apt-get update && apt-get install -y --no-install-recommends \
    libglib2.0-0 \
    libgl1 \
    libsm6 \
    libxext6 \
    libxrender-dev \
    libgomp1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install dependensi Python terlebih dahulu agar layer Docker ter-cache efisien
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Pre-download model YOLOv8n saat build image (tidak unduh saat runtime pertama)
RUN python -c "from ultralytics import YOLO; YOLO('yolov8n.pt')"

# Salin kode aplikasi
COPY . .

EXPOSE 8010

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8010", "--workers", "1"]

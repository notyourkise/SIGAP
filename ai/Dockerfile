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
# curl dipakai ultralytics untuk mengunduh model saat build & runtime.
RUN apt-get update && apt-get install -y --no-install-recommends \
    libglib2.0-0 \
    libgl1 \
    libsm6 \
    libxext6 \
    libxrender-dev \
    libgomp1 \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install dependensi Python terlebih dahulu agar layer Docker ter-cache efisien.
# torch/torchvision dipasang CPU-ONLY (index PyTorch CPU) agar inference YOLOv8n
# berjalan di CPU tanpa menarik CUDA toolkit multi-GB; versi disesuaikan dengan
# ultralytics 8.2.0.
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt \
    && pip install --no-cache-dir torch==2.3.1 torchvision==0.18.1 --index-url https://download.pytorch.org/whl/cpu

# Salin kode aplikasi
COPY . .

# Pre-download model YOLOv8n saat build image (tidak unduh saat runtime pertama).
# Dipakai curl langsung (bukan downloader internal ultralytics yang memanggil
# subprocess curl) agar build deterministik dan tidak bergantung pada tool lain.
# Ditempatkan SETELAH COPY . . agar tidak tertimpa bila ada file di build context.
RUN curl -sSL -o yolov8n.pt https://github.com/ultralytics/assets/releases/download/v8.1.0/yolov8n.pt \
    && python -c "from ultralytics import YOLO; YOLO('/app/yolov8n.pt'); print('✅ Model YOLOv8n OK')"

EXPOSE 8010

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8010", "--workers", "1"]

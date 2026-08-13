"""Manajemen sesi kamera: membaca RTSP, inferensi, tracking, dan pelaporan.

`CameraSession` merepresentasikan satu kamera CCTV yang aktif: membuka stream
RTSP dari MediaMTX (rtsp://mediamtx:8554/cam-{cctv_id}), menjalankan deteksi
YOLOv8n + tracking ByteTrack pada kecepatan ~1 fps, lalu mengirim hasilnya ke
backend sigap_api melalui endpoint /api/ai/detection-event.

CATATAN PENTING (decode H265):
Kamera NVR mengirim stream H265/HEVC. `cv2.VideoCapture` (opencv-python
headless) TIDAK bisa mendecode H265 dan akan hang saat `read()`. Karena itu
pembacaan frame memakai ffmpeg subprocess (binary `ffmpeg` di-install di image)
yang output rawvideo BGR ke pipe, lalu dibaca per-frame sebagai numpy array.
"""

from __future__ import annotations

import asyncio
import logging
import subprocess
import time
from typing import Any, Dict, List, Optional, Tuple

import httpx
import numpy as np

from .detector import Detector, Detection
from .person_counter import PersonCounter
from .tracker import PersonTracker

logger = logging.getLogger("sigap_ai.session")

# Interval minimum antar-frame yang diproses (~1 fps)
FRAME_INTERVAL_SECONDS = 1.0

# Resolusi target saat decode via ffmpeg (dikecilkan agar inferensi cepat).
# Dipakai sebagai fallback bila resolusi asli tidak diketahui.
_FFMPEG_WIDTH = 1280
_FFMPEG_HEIGHT = 720


class CameraSession:
    """Satu sesi kamera yang berjalan di background.

    Attributes:
        cctv_id: ID kamera (UUID CCTV) — dipakai pada path stream MediaMTX.
        rtsp_url: URL RTSP lengkap untuk membuka stream.
    """

    def __init__(
        self,
        cctv_id: str,
        rtsp_url: str,
        detector: Detector,
        tracker: PersonTracker,
        counter: PersonCounter,
        backend_url: str,
        ai_secret: str,
    ) -> None:
        """Inisialisasi sesi kamera.

        Args:
            cctv_id: ID kamera (UUID CCTV).
            rtsp_url: URL RTSP internal (mis. rtsp://mediamtx:8554/cam-xxx).
            detector: Detektor orang YOLOv8n (dibagi antar kamera).
            tracker: Tracker ByteTrack khusus untuk kamera ini.
            counter: Penghitung orang unik per hari (dibagi antar kamera).
            backend_url: URL dasar backend sigap_api (mis. http://sigap_api:3000).
            ai_secret: Secret header X-AI-Secret untuk autentikasi ke backend.
        """
        self.cctv_id = cctv_id
        self.rtsp_url = rtsp_url
        self.detector = detector
        self.tracker = tracker
        self.counter = counter
        self.backend_url = backend_url.rstrip("/")
        self.ai_secret = ai_secret

        self._task: Optional[asyncio.Task] = None
        self._stop_event = asyncio.Event()
        self._client: Optional[httpx.AsyncClient] = None
        self._last_process_time: float = 0.0
        # Proses ffmpeg aktif (None = belum terbuka / sedang reconnect).
        self._ffmpeg: Optional[subprocess.Popen] = None

    async def start(self) -> None:
        """Mulai loop pemrosesan kamera di background task."""
        if self._task is not None and not self._task.done():
            logger.warning("Sesi kamera %s sudah berjalan, dilewati.", self.cctv_id)
            return

        # Client HTTP dipakai bersama untuk semua POST hasil deteksi
        self._client = httpx.AsyncClient(timeout=3.0)
        self._stop_event.clear()
        self._task = asyncio.create_task(self._run(), name=f"camera-{self.cctv_id}")
        logger.info("Sesi kamera %s dimulai: %s", self.cctv_id, self.rtsp_url)

    async def stop(self) -> None:
        """Hentikan loop pemrosesan kamera dan bebaskan resource."""
        self._stop_event.set()
        if self._task is not None:
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass
            except Exception:
                logger.exception("Error saat menghentikan sesi kamera %s", self.cctv_id)
            self._task = None
        if self._client is not None:
            await self._client.aclose()
            self._client = None
        self._close_ffmpeg()
        logger.info("Sesi kamera %s dihentikan.", self.cctv_id)

    @property
    def is_active(self) -> bool:
        """True jika task pemrosesan sedang berjalan."""
        return self._task is not None and not self._task.done()

    # ------------------------------------------------------------------
    # Baca frame via ffmpeg (mendukung H265/HEVC yang tidak bisa dibaca
    # oleh cv2.VideoCapture). Output rawvideo BGR, dibaca per-frame.
    # ------------------------------------------------------------------
    def _open_ffmpeg(self) -> subprocess.Popen:
        """Buka proses ffmpeg untuk decode stream RTSP ke rawvideo BGR.

        Opsi penting:
        - `-fflags nobuffer+discardcorrupt` : buang frame korup (stream live
          H265 sering punya frame reference hilang saat sourceOnDemand baru
          mulai di-pull).
        - `-err_detect ignore_err`           : jangan berhenti saat error decode.
        - `-analyzeduration/-probesize` kecil: mulai decode lebih cepat.
        """
        cmd = [
            "ffmpeg",
            "-rtsp_transport", "tcp",
            "-fflags", "nobuffer+discardcorrupt",
            "-err_detect", "ignore_err",
            "-analyzeduration", "2000000",
            "-probesize", "1000000",
            "-i", self.rtsp_url,
            "-an",
            "-f", "rawvideo",
            "-pix_fmt", "bgr24",
            "-vf", f"scale={_FFMPEG_WIDTH}:{_FFMPEG_HEIGHT}:force_original_aspect_ratio=decrease,pad={_FFMPEG_WIDTH}:{_FFMPEG_HEIGHT}:(ow-iw)/2:(oh-ih)/2",
            "-loglevel", "error",
            "pipe:1",
        ]
        return subprocess.Popen(
            cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            bufsize=0,
        )

    def _close_ffmpeg(self) -> None:
        """Tutup proses ffmpeg bila ada (idempoten)."""
        proc = self._ffmpeg
        if proc is not None:
            try:
                proc.terminate()
                try:
                    proc.wait(timeout=3)
                except subprocess.TimeoutExpired:
                    proc.kill()
            except Exception:
                pass
            self._ffmpeg = None

    def _read_frame_ffmpeg(self, timeout: float = 15.0) -> Tuple[bool, Optional[np.ndarray]]:
        """Baca satu frame BGR dari stdout ffmpeg dengan batas waktu.

        Args:
            timeout: Detik maksimal menunggu data frame (default 15s). Diberi
                cukup longgar karena saat sourceOnDemand MediaMTX baru mulai
                pull dari NVR, ffmpeg perlu menunggu IDR/keyframe H265 pertama
                sebelum bisa decode. Bila lewat batas, dianggap gagal → reconnect.

        Returns:
            (True, frame) bila berhasil, (False, None) bila timeout/error.
        """
        proc = self._ffmpeg
        if proc is None or proc.stdout is None:
            return False, None

        # Ukuran satu frame rawvideo BGR 1280x720
        frame_size = _FFMPEG_WIDTH * _FFMPEG_HEIGHT * 3
        try:
            # Non-blocking read (Linux): hindari hang selamanya saat stream macet.
            import os
            os.set_blocking(proc.stdout.fileno(), False)
            raw = bytearray()
            deadline = time.monotonic() + timeout
            while len(raw) < frame_size and time.monotonic() < deadline:
                chunk = proc.stdout.read(frame_size - len(raw))
                if chunk:
                    raw.extend(chunk)
                else:
                    time.sleep(0.05)
            if len(raw) < frame_size:
                # Timeout / stream tidak menghasilkan frame.
                err = proc.stderr.read(200).decode(errors="ignore") if proc.stderr else ""
                if err:
                    logger.warning("ffmpeg kamera %s: %s", self.cctv_id, err.strip())
                return False, None
            frame = np.frombuffer(bytes(raw), dtype=np.uint8).reshape(
                (_FFMPEG_HEIGHT, _FFMPEG_WIDTH, 3)
            )
            return True, frame
        except Exception:
            return False, None

    def _build_payload(
        self,
        detections: List[Detection],
        person_count: int,
        unique_count: int,
        confidence_avg: float,
        frame_shape: tuple,
    ) -> Dict[str, Any]:
        """Bangun payload JSON untuk dikirim ke backend.

        Bounding box dinormalisasi ke rentang 0.0–1.0 relatif terhadap frame
        (x/width, y/height) agar frontend dapat menggambar overlay di atas
        <video> tanpa perlu mengetahui resolusi asli stream.
        """
        h, w = frame_shape[0], frame_shape[1]
        bboxes: List[Dict[str, Any]] = []
        for det in detections:
            if det.track_id is None:
                continue
            bboxes.append(
                {
                    "x1": round(det.x1 / w, 4) if w else 0.0,
                    "y1": round(det.y1 / h, 4) if h else 0.0,
                    "x2": round(det.x2 / w, 4) if w else 0.0,
                    "y2": round(det.y2 / h, 4) if h else 0.0,
                    "confidence": det.confidence,
                    "track_id": det.track_id,
                }
            )
        return {
            "cctv_id": self.cctv_id,
            "bboxes": bboxes,
            "person_count": person_count,
            "unique_count": unique_count,
            "confidence_avg": confidence_avg,
        }

    async def _send_event(
        self,
        detections: List[Detection],
        person_count: int,
        unique_count: int,
        confidence_avg: float,
        frame_shape: tuple,
    ) -> None:
        """Kirim hasil deteksi ke backend sigap_api.

        Endpoint: POST {backend_url}/api/ai/detection-event dengan header
        X-AI-Secret. Gagal/koneksi ditolak hanya dicatat sebagai warning.
        """
        if self._client is None:
            return
        payload = self._build_payload(detections, person_count, unique_count, confidence_avg, frame_shape)
        url = f"{self.backend_url}/api/ai/detection-event"
        headers = {"X-AI-Secret": self.ai_secret}
        try:
            resp = await self._client.post(url, json=payload, headers=headers)
            resp.raise_for_status()
            logger.debug("Event terkirim untuk %s: %d bbox", self.cctv_id, len(payload["bboxes"]))
        except httpx.TimeoutException:
            logger.warning("Timeout mengirim event deteksi untuk kamera %s", self.cctv_id)
        except httpx.HTTPError as exc:
            logger.warning("Gagal mengirim event deteksi kamera %s: %s", self.cctv_id, exc)
        except Exception:
            logger.exception("Error tak terduga saat mengirim event kamera %s", self.cctv_id)

    async def _run(self) -> None:
        """Loop utama: baca frame RTSP (via ffmpeg), inferensi + tracking, kirim hasil."""
        loop = asyncio.get_running_loop()

        while not self._stop_event.is_set():
            # --- Fase 1: pastikan proses ffmpeg terbuka ---
            if self._ffmpeg is None:
                try:
                    # Buka ffmpeg di thread pool (Popen cepat, tidak blocking lama).
                    proc = await loop.run_in_executor(None, self._open_ffmpeg)
                    self._ffmpeg = proc
                    logger.info("Stream RTSP kamera %s terbuka (ffmpeg)", self.cctv_id)
                except Exception:
                    logger.exception("Gagal membuka stream kamera %s", self.cctv_id)
                    self._ffmpeg = None

            if self._ffmpeg is None:
                # Tunggu sebelum reconnect (jaringan/stream belum siap)
                await self._sleep(2.0)
                continue

            # --- Fase 2: baca satu frame (non-blocking via executor) ---
            try:
                ok, frame = await loop.run_in_executor(None, self._read_frame_ffmpeg)
            except Exception:
                logger.exception("Error membaca frame kamera %s", self.cctv_id)
                ok, frame = False, None

            if not ok or frame is None:
                logger.warning("Gagal membaca frame kamera %s, mencoba reconnect...", self.cctv_id)
                self._close_ffmpeg()
                await self._sleep(2.0)
                continue

            # --- Fase 3: throttle inferensi ~1 fps ---
            now = time.monotonic()
            if now - self._last_process_time < FRAME_INTERVAL_SECONDS:
                # Frame dibuang (tidak diproses) agar beban CPU terkendali
                continue

            self._last_process_time = now

            # --- Fase 4: inferensi + tracking di thread pool (blocking) ---
            try:
                detections = await loop.run_in_executor(None, self._infer_and_track, frame)
            except Exception:
                logger.exception("Error inferensi kamera %s", self.cctv_id)
                continue

            # --- Fase 5: hitung orang unik & kirim event ke backend ---
            tracking_ids = [d.track_id for d in detections if d.track_id is not None]
            counts = self.counter.process(self.cctv_id, tracking_ids)
            person_count = len(detections)
            confidence_avg = (
                float(np.mean([d.confidence for d in detections])) if detections else 0.0
            )

            await self._send_event(
                detections,
                person_count,
                counts["unique_today"],
                confidence_avg,
                frame.shape,
            )

        # --- Bersih-bersih saat sesi dihentikan ---
        self._close_ffmpeg()
        logger.info("Loop kamera %s selesai.", self.cctv_id)

    def _infer_and_track(self, frame: np.ndarray) -> List[Detection]:
        """Jalankan deteksi lalu tracking (dipanggil di thread pool).

        Args:
            frame: Frame BGR dari OpenCV.

        Returns:
            Deteksi orang dengan track_id (koordinat absolut piksel).
        """
        detections = self.detector.detect_persons(frame)
        tracked = self.tracker.update(detections, frame.shape)
        return tracked

    async def _sleep(self, seconds: float) -> None:
        """Tidur async yang bisa dibatalkan saat stop."""
        try:
            await asyncio.sleep(seconds)
        except asyncio.CancelledError:
            raise


class SessionManager:
    """Mengelola semua sesi kamera aktif.

    Satu `PersonTracker` dibuat per kamera (state ByteTrack terpisah),
    sedangkan `Detector` dan `PersonCounter` dibagi antar kamera.
    """

    def __init__(self, detector: Detector, counter: PersonCounter) -> None:
        """Inisialisasi manager sesi.

        Args:
            detector: Detektor YOLOv8n yang dibagi semua kamera.
            counter: Penghitung orang unik yang dibagi semua kamera.
        """
        self.detector = detector
        self.counter = counter
        self._sessions: Dict[str, CameraSession] = {}

    def activate(
        self,
        cctv_id: str,
        rtsp_url: str,
        backend_url: str,
        ai_secret: str,
    ) -> CameraSession:
        """Aktifkan sesi untuk satu kamera (idempoten).

        Jika kamera sudah aktif, sesi lama tidak dibuat ulang (tracking ID
        tetap terjaga). Jika URL berubah, sesi lama dihentikan dulu.
        """
        existing = self._sessions.get(cctv_id)
        if existing is not None and existing.is_active:
            if existing.rtsp_url == rtsp_url:
                logger.info("Kamera %s sudah aktif, tidak dibuat ulang.", cctv_id)
                return existing
            # URL berubah: hentikan sesi lama terlebih dahulu (fire-and-forget),
            # agar tidak ada dua loop membaca stream yang sama.
            logger.info("URL kamera %s berubah, restart sesi.", cctv_id)
            self._sessions.pop(cctv_id, None)
            self._fire_and_forget_stop(existing)

        tracker = PersonTracker()
        session = CameraSession(
            cctv_id=cctv_id,
            rtsp_url=rtsp_url,
            detector=self.detector,
            tracker=tracker,
            counter=self.counter,
            backend_url=backend_url,
            ai_secret=ai_secret,
        )
        self._sessions[cctv_id] = session
        return session

    def deactivate(self, cctv_id: str) -> bool:
        """Nonaktifkan sesi kamera (hapus dari daftar).

        Args:
            cctv_id: ID kamera.

        Returns:
            True jika sesi ditemukan dan dihapus, False jika tidak ada.
        """
        session = self._sessions.pop(cctv_id, None)
        if session is None:
            return False
        # Panggil stop() sebagai fire-and-forget dari konteks async pemanggil
        self._fire_and_forget_stop(session)
        return True

    @staticmethod
    def _fire_and_forget_stop(session: CameraSession) -> None:
        """Hentikan sesi tanpa menunggu, aman dipanggil dari konteks non-async.

        Jika tidak ada event loop yang sedang berjalan (mis. saat shutdown),
        task sesi lama otomatis dibatalkan oleh proses shutdown itu sendiri.
        """
        try:
            loop = asyncio.get_running_loop()
            loop.create_task(session.stop())
        except RuntimeError:
            logger.warning("Tidak ada event loop untuk menghentikan sesi %s", session.cctv_id)

    def get_active(self) -> List[Dict[str, Any]]:
        """Daftar kamera yang sedang aktif beserta info singkatnya."""
        result: List[Dict[str, Any]] = []
        for cctv_id, session in self._sessions.items():
            if session.is_active:
                result.append(
                    {
                        "cctv_id": cctv_id,
                        "rtsp_url": session.rtsp_url,
                        "unique_today": self.counter.get_count(cctv_id),
                    }
                )
        return result

    def deactivate_all(self) -> None:
        """Hentikan semua sesi (dipakai saat shutdown aplikasi)."""
        for cctv_id in list(self._sessions.keys()):
            self.deactivate(cctv_id)
        self._sessions.clear()

    def reset_all_counts(self) -> None:
        """Reset penghitung orang unik untuk semua kamera."""
        self.counter.reset_all()

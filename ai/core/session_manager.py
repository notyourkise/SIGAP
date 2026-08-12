"""Manajemen sesi kamera: membaca RTSP, inferensi, tracking, dan pelaporan.

`CameraSession` merepresentasikan satu kamera CCTV yang aktif: membuka stream
RTSP dari MediaMTX (rtsp://mediamtx:8554/cam-{cctv_id}), menjalankan deteksi
YOLOv8n + tracking ByteTrack pada kecepatan ~1 fps, lalu mengirim hasilnya ke
backend sigap_api melalui endpoint /api/ai/detection-event.
"""

from __future__ import annotations

import asyncio
import logging
import time
from typing import Any, Dict, List, Optional

import cv2
import httpx
import numpy as np

from .detector import Detector, Detection
from .person_counter import PersonCounter
from .tracker import PersonTracker

logger = logging.getLogger("sigap_ai.session")

# Interval minimum antar-frame yang diproses (~1 fps)
FRAME_INTERVAL_SECONDS = 1.0


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
        logger.info("Sesi kamera %s dihentikan.", self.cctv_id)

    @property
    def is_active(self) -> bool:
        """True jika task pemrosesan sedang berjalan."""
        return self._task is not None and not self._task.done()

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
        """Loop utama: baca frame RTSP, inferensi + tracking, kirim hasil."""
        cap: Optional[cv2.VideoCapture] = None
        loop = asyncio.get_running_loop()

        while not self._stop_event.is_set():
            # --- Fase 1: pastikan koneksi video terbuka ---
            if cap is None:
                try:
                    cap = cv2.VideoCapture(self.rtsp_url)
                    # Buffer kecil agar latency tetap rendah (frame terbaru saja)
                    cap.set(cv2.CAP_PROP_BUFFERSIZE, 1)
                    logger.info("Stream RTSP kamera %s terbuka", self.cctv_id)
                except Exception:
                    logger.exception("Gagal membuka stream kamera %s", self.cctv_id)
                    cap = None

            if cap is None:
                # Tunggu sebelum reconnect (jaringan/stream belum siap)
                await self._sleep(2.0)
                continue

            # --- Fase 2: baca satu frame (non-blocking via executor) ---
            try:
                ok, frame = await loop.run_in_executor(None, cap.read)
            except Exception:
                logger.exception("Error membaca frame kamera %s", self.cctv_id)
                ok, frame = False, None

            if not ok or frame is None:
                logger.warning("Gagal membaca frame kamera %s, mencoba reconnect...", self.cctv_id)
                cap.release()
                cap = None
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
        if cap is not None:
            cap.release()
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

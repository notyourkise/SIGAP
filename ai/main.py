"""SIGAP AI Detection Service — entrypoint FastAPI.

Service Python (FastAPI) untuk deteksi orang (YOLOv8n) + tracking (ByteTrack)
pada stream RTSP internal MediaMTX. Hasil deteksi dikirim ke backend sigap_api
melalui endpoint /api/ai/detection-event.

Alur:
    Browser -> sigap_api -> POST /sessions -> sigap_ai membuka stream RTSP,
    inferensi ~1 fps per kamera, menghitung orang unik per hari, lalu POST hasil
    ke sigap_api (header X-AI-Secret) -> Go simpan DB + broadcast WebSocket.

Env vars:
    BACKEND_URL        : URL internal sigap_api (default http://sigap_api:3000)
    MEDIAMTX_RTSP_URL  : URL RTSP internal MediaMTX (default rtsp://mediamtx:8554)
    AI_SECRET          : Secret header X-AI-Secret ke backend (default dev)
"""

from __future__ import annotations

import asyncio
import datetime
import logging
import os
from contextlib import asynccontextmanager
from typing import Any, Dict

from fastapi import FastAPI

from core.detector import Detector
from core.person_counter import PersonCounter
from core.session_manager import SessionManager
from routers import health, session

# Konfigurasi logging: format dengan waktu & level agar mudah dibaca di Docker
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("sigap_ai")

# Zona waktu WITA (UTC+8) untuk reset harian pukul 00:00
WITA_OFFSET = datetime.timedelta(hours=8)


async def daily_reset_scheduler(manager: SessionManager) -> None:
    """Scheduler harian: reset penghitung orang unik setiap 00:00 WITA.

    Loop berjalan terus; menghitung detik tersisa hingga tengah malam WITA
    (UTC+8), tidur selama itu, lalu reset counter semua kamera.
    """
    logger.info("Scheduler reset harian dimulai (00:00 WITA = UTC+8).")
    while True:
        now_utc = datetime.datetime.now(datetime.timezone.utc)
        now_wita = now_utc + WITA_OFFSET
        next_midnight = (now_wita + datetime.timedelta(days=1)).replace(
            hour=0, minute=0, second=0, microsecond=0
        )
        seconds_until = (next_midnight - now_wita).total_seconds()
        try:
            await asyncio.sleep(seconds_until)
        except asyncio.CancelledError:
            logger.info("Scheduler reset harian dibatalkan.")
            raise
        logger.info("Tengah malam WITA: reset penghitung orang unik.")
        manager.reset_all_counts()


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Lifecycle aplikasi: mulai scheduler reset harian, bersihkan saat shutdown."""
    # Mulai scheduler reset harian sebagai background task
    scheduler_task = asyncio.create_task(daily_reset_scheduler(app.state.manager))
    logger.info("SIGAP AI Detection Service dimulai.")
    try:
        yield
    finally:
        # Matikan semua sesi kamera & hentikan scheduler saat shutdown
        app.state.manager.deactivate_all()
        scheduler_task.cancel()
        try:
            await scheduler_task
        except asyncio.CancelledError:
            pass
        logger.info("SIGAP AI Detection Service dihentikan.")


def create_app() -> FastAPI:
    """Bangun aplikasi FastAPI beserta state globalnya."""
    app = FastAPI(
        title="SIGAP AI Detection Service",
        description="Layanan deteksi orang (YOLOv8n) + tracking (ByteTrack) "
                    "dari stream RTSP MediaMTX untuk SIGAP.",
        version="1.0.0",
        lifespan=lifespan,
    )

    # State global aplikasi
    app.state.backend_url = os.getenv("BACKEND_URL", "http://sigap_api:3000")
    app.state.ai_secret = os.getenv("AI_SECRET", "sigap-ai-secret-change-me")
    app.state.mediamtx_url = os.getenv("MEDIAMTX_RTSP_URL", "rtsp://mediamtx:8554")

    # Detector (YOLOv8n) dibagi semua kamera; counter dipakai bersama
    detector = Detector()
    counter = PersonCounter()
    app.state.manager = SessionManager(detector=detector, counter=counter)

    # Daftarkan router
    app.include_router(session.router, prefix="/sessions")
    app.include_router(health.router, prefix="/health")

    @app.get("/counts/{cctv_id}", response_model=Dict[str, Any])
    async def get_counts(cctv_id: str) -> Dict[str, Any]:
        """Jumlah orang unik hari ini untuk satu kamera."""
        count = app.state.manager.counter.get_count(cctv_id)
        return {"cctv_id": cctv_id, "unique_today": count}

    @app.get("/counts", response_model=Dict[str, Any])
    async def get_all_counts() -> Dict[str, Any]:
        """Jumlah orang unik hari ini untuk semua kamera."""
        return {"data": app.state.manager.counter.get_all_counts()}

    @app.post("/reset-all", response_model=Dict[str, Any])
    async def reset_all() -> Dict[str, Any]:
        """Reset manual penghitung orang unik semua kamera."""
        app.state.manager.reset_all_counts()
        return {"status": "ok", "message": "Semua penghitung orang unik di-reset."}

    return app


# Instance aplikasi yang dipakai uvicorn: `uvicorn main:app`
app = create_app()

"""Router untuk manajemen sesi kamera AI.

Endpoint:
    POST   /sessions          -> aktifkan kamera (body: cctv_id, rtsp_url opsional)
    DELETE /sessions/{cctv_id} -> nonaktifkan kamera
    GET    /sessions           -> daftar kamera aktif
"""

from __future__ import annotations

from typing import Any, Dict, List, Optional

from fastapi import APIRouter, HTTPException, Request
from pydantic import BaseModel, Field

router = APIRouter(tags=["sessions"])


class SessionCreate(BaseModel):
    """Body request untuk mengaktifkan sesi kamera."""

    cctv_id: str = Field(..., description="ID kamera (UUID CCTV)")
    rtsp_url: Optional[str] = Field(None, description="URL RTSP opsional (default: rtsp://mediamtx:8554/cam-{cctv_id})")


@router.post("", response_model=Dict[str, Any])
async def create_session(body: SessionCreate, request: Request) -> Dict[str, Any]:
    """Aktifkan sesi deteksi untuk satu kamera.

    Jika `rtsp_url` tidak diberikan, URL default dibangun dari MEDIAMTX_RTSP_URL:
    {mediamtx_url}/cam-{cctv_id}.
    """
    manager = request.app.state.manager
    mediamtx_url = request.app.state.mediamtx_url.rstrip("/")

    rtsp_url = body.rtsp_url or f"{mediamtx_url}/cam-{body.cctv_id}"

    session = manager.activate(
        cctv_id=body.cctv_id,
        rtsp_url=rtsp_url,
        backend_url=request.app.state.backend_url,
        ai_secret=request.app.state.ai_secret,
    )
    await session.start()

    return {
        "status": "ok",
        "cctv_id": body.cctv_id,
        "rtsp_url": rtsp_url,
    }


@router.delete("/{cctv_id}", response_model=Dict[str, Any])
async def delete_session(cctv_id: str, request: Request) -> Dict[str, Any]:
    """Nonaktifkan sesi kamera berdasarkan cctv_id."""
    manager = request.app.state.manager
    if not manager.deactivate(cctv_id):
        raise HTTPException(status_code=404, detail=f"Kamera {cctv_id} tidak ditemukan / tidak aktif")
    return {"status": "ok", "cctv_id": cctv_id}


@router.get("", response_model=Dict[str, Any])
async def list_sessions(request: Request) -> Dict[str, Any]:
    """Daftar kamera yang sedang aktif beserta info ringkasnya."""
    manager = request.app.state.manager
    return {"data": manager.get_active()}

"""Router health check untuk layanan AI SIGAP.

Digunakan oleh orchestrator (mis. healthcheck Docker) untuk memastikan
service sigap_ai sudah siap menerima permintaan.
"""

from __future__ import annotations

from typing import Dict

from fastapi import APIRouter

router = APIRouter(tags=["health"])


@router.get("", response_model=Dict[str, str])
async def health() -> Dict[str, str]:
    """Health check sederhana: service hidup dan siap."""
    return {"status": "ok", "service": "sigap_ai"}

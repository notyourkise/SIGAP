"""Penghitung orang unik per hari per kamera (CCTV).

`PersonCounter` menyimpan state per `cctv_id`: tanggal, set ID tracking yang
sudah terlihat, dan jumlah total. Counter di-reset otomatis saat tanggal
berganti (midnight), dan bisa di-reset manual melalui `reset_all()`.
"""

from __future__ import annotations

import datetime
import threading
from typing import Dict, List, Optional, Set, Tuple


class PersonCounter:
    """Menghitung orang unik per hari untuk beberapa kamera CCTV.

    State internal:
        _state[cctv_id] = (tanggal, seen_ids: Set[int], count: int)
    """

    def __init__(self) -> None:
        """Inisialisasi counter kosong dengan lock untuk keamanan thread."""
        # _lock melindungi akses dari thread inference (asyncio executor)
        self._lock = threading.Lock()
        self._state: Dict[str, Tuple[str, Set[int], int]] = {}
        self._today = datetime.date.today().isoformat()

    def _reset_if_new_day(self, cctv_id: str, date_str: str) -> None:
        """Reset counter per kamera jika tanggal sudah berganti.

        Args:
            cctv_id: ID kamera (UUID CCTV).
            date_str: Tanggal hari ini (ISO format yyyy-mm-dd).
        """
        entry = self._state.get(cctv_id)
        if entry is not None and entry[0] != date_str:
            # Hari baru: mulai hitung ulang dari nol
            self._state[cctv_id] = (date_str, set(), 0)

    def process(
        self, cctv_id: str, tracking_ids: List[int]
    ) -> Dict[str, int]:
        """Proses daftar track_id yang terlihat pada frame saat ini.

        Args:
            cctv_id: ID kamera (UUID CCTV).
            tracking_ids: Daftar track_id ByteTrack pada frame ini.

        Returns:
            Dictionary: {"unique_today", "new_this_frame", "current_in_frame"}.
        """
        date_str = datetime.date.today().isoformat()

        with self._lock:
            # Auto reset harian: pastikan state milik tanggal hari ini
            if cctv_id not in self._state or self._state[cctv_id][0] != date_str:
                self._state[cctv_id] = (date_str, set(), 0)

            entry = self._state[cctv_id]
            _, seen_ids, count = entry
            new_this_frame = 0

            for track_id in tracking_ids:
                if track_id is None:
                    continue
                if track_id not in seen_ids:
                    seen_ids.add(track_id)
                    count += 1
                    new_this_frame += 1

            self._state[cctv_id] = (date_str, seen_ids, count)
            current_in_frame = len(tracking_ids)

        return {
            "unique_today": count,
            "new_this_frame": new_this_frame,
            "current_in_frame": current_in_frame,
        }

    def reset_all(self) -> None:
        """Reset seluruh counter (semua kamera) — dipakai saat reset manual."""
        date_str = datetime.date.today().isoformat()
        with self._lock:
            self._state = {}
            self._today = date_str

    def get_count(self, cctv_id: str) -> int:
        """Ambil jumlah orang unik hari ini untuk satu kamera.

        Args:
            cctv_id: ID kamera (UUID CCTV).

        Returns:
            Jumlah orang unik yang terlihat hari ini (0 jika belum ada data).
        """
        date_str = datetime.date.today().isoformat()
        with self._lock:
            entry = self._state.get(cctv_id)
            if entry is None or entry[0] != date_str:
                return 0
            return entry[2]

    def get_all_counts(self) -> Dict[str, int]:
        """Ambil jumlah orang unik hari ini untuk semua kamera.

        Returns:
            Dictionary {cctv_id: jumlah_orang_unik} untuk state tanggal hari ini.
        """
        date_str = datetime.date.today().isoformat()
        with self._lock:
            return {
                cctv_id: entry[2]
                for cctv_id, entry in self._state.items()
                if entry[0] == date_str
            }

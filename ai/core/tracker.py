"""Tracker orang berbasis ByteTrack (supervision).

`PersonTracker` membungkus `sv.ByteTrack` agar mudah digunakan bersama hasil
deteksi `Detection` dari `Detector`. Setiap orang yang terdeteksi akan diberi
`track_id` unik yang konsisten antar-frame selama masih terlihat.
"""

from __future__ import annotations

from typing import List, Sequence, Tuple

import numpy as np
import supervision as sv

from .detector import Detection


class PersonTracker:
    """Tracker ByteTrack untuk orang.

    Mengonversi daftar `Detection` menjadi `sv.Detections` (xyxy, confidence,
    class_id = 0), kemudian memanggil `update_with_detections` milik ByteTrack
    untuk mendapatkan ID tracking yang stabil.
    """

    def __init__(self, track_thresh: float = 0.45, track_buffer: int = 60) -> None:
        """Inisialisasi tracker ByteTrack.

        Args:
            track_thresh: Ambang deteksi untuk tracking (default: 0.45).
            track_buffer: Jumlah frame buffer sebelum track dianggap hilang.
        """
        self.tracker = sv.ByteTrack(
            track_activation_threshold=track_thresh,
            lost_track_buffer=track_buffer,
            minimum_matching_threshold=0.8,
        )

    def update(
        self,
        detections: Sequence[Detection],
        frame_shape: Tuple[int, int, int],
    ) -> List[Detection]:
        """Perbarui track dengan deteksi terbaru.

        Args:
            detections: Deteksi orang pada frame saat ini.
            frame_shape: Bentuk frame (height, width, channels) dari OpenCV.

        Returns:
            Daftar `Detection` yang sama dengan tambahan `track_id` dari ByteTrack.
        """
        if not detections:
            return list(detections)

        # Bangun array numpy untuk sv.Detections
        xyxy = np.array(
            [[d.x1, d.y1, d.x2, d.y2] for d in detections], dtype=np.float32
        )
        confidence = np.array([d.confidence for d in detections], dtype=np.float32)
        class_id = np.zeros(len(detections), dtype=int)

        sv_detections = sv.Detections(
            xyxy=xyxy,
            confidence=confidence,
            class_id=class_id,
        )

        # Update tracker ByteTrack untuk frame ini
        tracked = self.tracker.update_with_detections(sv_detections)

        # Salin track_id kembali ke objek Detection (urutan sama dengan input)
        result: List[Detection] = []
        for det, track_id in zip(detections, tracked.tracker_id):
            det.track_id = int(track_id)
            result.append(det)
        return result

"""Deteksi orang menggunakan YOLOv8n.

Modul ini membungkus model Ultralytics YOLOv8n agar hanya memproses kelas
"person" (class index 0 pada COCO). Setiap deteksi direpresentasikan oleh
dataclass `Detection` dengan koordinat absolut piksel (x1, y1, x2, y2).
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional

import numpy as np
from ultralytics import YOLO


@dataclass
class Detection:
    """Satu hasil deteksi orang dalam sebuah frame.

    Attributes:
        x1, y1, x2, y2: Koordinat bounding box absolut (piksel) pada frame.
        confidence: Skor keyakinan model (0.0 - 1.0).
        track_id: ID unik dari ByteTrack; None jika belum di-track.
    """

    x1: float
    y1: float
    x2: float
    y2: float
    confidence: float
    track_id: Optional[int] = None

    def to_dict(self) -> Dict[str, Any]:
        """Ubah objek deteksi menjadi dictionary (label selalu "Person")."""
        return {
            "x1": self.x1,
            "y1": self.y1,
            "x2": self.x2,
            "y2": self.y2,
            "confidence": self.confidence,
            "track_id": self.track_id,
            "label": "Person",
        }


class Detector:
    """Detektor orang berbasis YOLOv8n.

    Memuat model sekali saat inisialisasi, kemudian `detect_persons(frame)`
    memfilter hasil inference agar hanya mengembalikan deteksi kelas orang
    (class 0 pada dataset COCO) dengan confidence di atas ambang batas.
    """

    def __init__(self, model_path: str = "yolov8n.pt", confidence: float = 0.50) -> None:
        """Inisialisasi detektor dan memuat model YOLO.

        Args:
            model_path: Lokasi file bobot model (default: yolov8n.pt).
            confidence: Ambang batas confidence minimum (default: 0.50).
        """
        self.model = YOLO(model_path)
        self.confidence = confidence

    def detect_persons(self, frame: np.ndarray) -> List[Detection]:
        """Deteksi orang pada satu frame.

        Args:
            frame: Array numpy frame BGR (dari OpenCV).

        Returns:
            Daftar objek `Detection` yang hanya berisi kelas orang (class 0).
        """
        results = self.model(frame, verbose=False)
        detections: List[Detection] = []

        # Hasil YOLO bisa berisi beberapa image; ambil yang pertama (single image).
        result = results[0]
        boxes = result.boxes
        if boxes is None or len(boxes) == 0:
            return detections

        for box in boxes:
            cls_id = int(box.cls.item())
            conf = float(box.conf.item())

            # Hanya kelas "person" (index 0 pada COCO) di atas ambang confidence
            if cls_id != 0 or conf < self.confidence:
                continue

            x1, y1, x2, y2 = box.xyxy[0].tolist()
            detections.append(
                Detection(x1=x1, y1=y1, x2=x2, y2=y2, confidence=conf)
            )

        return detections

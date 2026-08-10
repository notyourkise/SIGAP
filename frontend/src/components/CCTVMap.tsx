'use client';

import React, { useEffect, useMemo, useRef, useState } from 'react';
import { MapContainer, TileLayer, Marker, Popup, Tooltip } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import { MapPin, Maximize, Minimize, ShieldCheck } from 'lucide-react';

export interface CCTVMapItem {
  id: string;
  kode_cctv: string;
  nama: string;
  lokasi_area?: string;
  latitude?: string;
  longitude?: string;
  status: string;
  health_score?: number;
  ip_camera?: string;
  ip_nvr?: string;
  stream_enabled?: boolean;
  channel_port?: number;
  snapshot_url?: string;
}

interface CCTVMapProps {
  cctvs: CCTVMapItem[];
  onMarkerClick?: (cctv: CCTVMapItem) => void;
}

// Koordinat Pusat IKN Nusantara (KIPP IKN)
const IKN_CENTER_LAT = -0.9723;
const IKN_CENTER_LNG = 116.7083;

const normalizeStatus = (s: string) => (s || '').trim().toUpperCase();
const isStatusActive = (s: string) => ['ACTIVE', 'AKTIF'].includes(normalizeStatus(s));
const isStatusMaintenance = (s: string) => ['MAINTENANCE', 'PEMELIHARAAN'].includes(normalizeStatus(s));
const normalizeHealthScore = (score?: number) => {
  if (typeof score !== 'number' || !Number.isFinite(score)) return 100;
  return Math.max(0, Math.min(100, Math.round(score)));
};
const healthScoreColor = (score?: number) => {
  const value = normalizeHealthScore(score);
  if (value >= 70) return '#10B981';
  if (value >= 40) return '#F59E0B';
  return '#EF4444';
};
const healthScoreClasses = (score?: number) => {
  const value = normalizeHealthScore(score);
  if (value >= 70) return 'border-emerald-500/20 bg-emerald-500/10 text-emerald-600';
  if (value >= 40) return 'border-amber-500/20 bg-amber-500/10 text-amber-600';
  return 'border-red-500/20 bg-red-500/10 text-red-600';
};

// Fungsi pembentuk custom SVG Icon untuk Leaflet berdasarkan skor kesehatan.
// Param tambahan `isStreaming` menambah cincin hijau ganda (double-pulse) sebagai penanda kamera yang sedang streaming.
const createCustomMarkerIcon = (healthScore?: number, isStreaming?: boolean) => {
  const color = healthScoreColor(healthScore);
  const shadowColor = `${color}66`;

  const streamingRing = isStreaming
    ? `
      <div style="
        position: absolute;
        inset: -7px;
        border-radius: 50%;
        border: 3px solid rgba(16, 185, 129, 0.7);
        animation: streamPulse 1.6s ease-out infinite;
      "></div>
      <div style="
        position: absolute;
        inset: -13px;
        border-radius: 50%;
        border: 2px solid rgba(16, 185, 129, 0.35);
        animation: streamPulse 1.6s ease-out 0.6s infinite;
      "></div>
    `
    : '';

  const svgHtml = `
    <div style="
      position: relative;
      width: 34px;
      height: 34px;
      display: flex;
      align-items: center;
      justify-content: center;
    ">
      ${streamingRing}
      <div style="
        position: absolute;
        width: 32px;
        height: 32px;
        border-radius: 50%;
        background-color: ${color};
        opacity: 0.25;
        animation: ping 2s cubic-bezier(0, 0, 0.2, 1) infinite;
      "></div>
      <div style="
        position: relative;
        width: 26px;
        height: 26px;
        border-radius: 50%;
        background-color: ${color};
        border: 2.5px solid white;
        box-shadow: 0 4px 12px ${shadowColor};
        display: flex;
        align-items: center;
        justify-content: center;
        color: white;
      ">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/>
          <circle cx="12" cy="13" r="3"/>
        </svg>
      </div>
    </div>
  `;

  return L.divIcon({
    html: svgHtml,
    className: 'custom-cctv-marker',
    iconSize: [34, 34],
    iconAnchor: [17, 17],
    popupAnchor: [0, -18],
  });
};

// Definisi animasi untuk peta (ring streaming ganda & ping) — dimuat sekali di sisi klien.
const markerKeyframesStyle = `
  @keyframes streamPulse {
    0% { transform: scale(0.6); opacity: 0.9; }
    70% { transform: scale(1.25); opacity: 0; }
    100% { transform: scale(1.25); opacity: 0; }
  }
  .custom-cctv-marker { background: transparent; border: none; }
`;

export default function CCTVMap({ cctvs, onMarkerClick }: CCTVMapProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);

  // Injeksi animasi marker streaming (ring hijau ganda) ke dalam <head> hanya di sisi klien.
  useEffect(() => {
    const styleId = 'cctv-map-marker-keyframes';
    if (!document.getElementById(styleId)) {
      const styleEl = document.createElement('style');
      styleEl.id = styleId;
      styleEl.textContent = markerKeyframesStyle;
      document.head.appendChild(styleEl);
    }
  }, []);

  useEffect(() => {
    const handleFullscreenChange = () => {
      setIsFullscreen(Boolean(document.fullscreenElement));
    };
    document.addEventListener('fullscreenchange', handleFullscreenChange);
    return () => document.removeEventListener('fullscreenchange', handleFullscreenChange);
  }, []);

  const toggleFullscreen = () => {
    if (document.fullscreenElement) {
      void document.exitFullscreen().catch(() => undefined);
    } else {
      void containerRef.current?.requestFullscreen().catch(() => undefined);
    }
  };

  // Olah koordinat CCTV & sediakan fallback terdistribusi jika lat/long belum terisi
  const processedCCTVs = useMemo(() => {
    const coordinateCounts = new Map<string, number>();

    return cctvs.map((cctv, index) => {
      let lat = parseFloat(cctv.latitude || '');
      let lng = parseFloat(cctv.longitude || '');

      // Jika koordinat tidak valid / kosong, sebar secara deterministik di sekitar KIPP IKN
      if (isNaN(lat) || isNaN(lng) || lat === 0 || lng === 0) {
        const angle = (index * 137.5) * (Math.PI / 180);
        const radius = 0.008 + (index % 5) * 0.004;
        lat = IKN_CENTER_LAT + radius * Math.cos(angle);
        lng = IKN_CENTER_LNG + radius * Math.sin(angle);
      }

      // Geser marker dengan koordinat identik agar tidak saling menutupi
      const coordinateKey = `${lat},${lng}`;
      const occurrence = (coordinateCounts.get(coordinateKey) ?? 0) + 1;
      coordinateCounts.set(coordinateKey, occurrence);
      if (occurrence > 1) {
        const offsetAngle = ((occurrence - 1) * 137.5) * (Math.PI / 180);
        const offsetRadius = 0.005;
        lat += offsetRadius * Math.cos(offsetAngle);
        lng += offsetRadius * Math.sin(offsetAngle);
      }

      return {
        ...cctv,
        lat,
        lng,
      };
    });
  }, [cctvs]);

  return (
    <div
      ref={containerRef}
      className={`relative w-full overflow-hidden transition-[height,border-radius] duration-300 ${
        isFullscreen
          ? 'h-screen rounded-none border-0'
          : 'h-[460px] rounded-3xl border border-border shadow-md'
      }`}
    >
      <MapContainer
        center={[IKN_CENTER_LAT, IKN_CENTER_LNG]}
        zoom={13}
        scrollWheelZoom={false}
        className="w-full h-full z-10"
      >
        <TileLayer
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        />

        {processedCCTVs.map((item) => (
          <Marker
            key={item.id}
            position={[item.lat, item.lng]}
            icon={createCustomMarkerIcon(item.health_score, item.stream_enabled)}
            eventHandlers={{
              click: () => {
                // Jika ada handler klik & kamera streaming aktif → buka modal stream, jangan buka Popup.
                if (onMarkerClick && item.stream_enabled === true) {
                  onMarkerClick(item);
                }
              },
            }}
          >
            <Tooltip direction="top" offset={[0, -15]} opacity={0.95}>
              <div className="text-xs font-bold font-sans">
                {item.nama} <span className="font-mono text-[10px] text-muted">({item.kode_cctv})</span>
                <div className="mt-0.5 text-[10px] font-semibold text-muted">Kesehatan: {normalizeHealthScore(item.health_score)}/100</div>
              </div>
            </Tooltip>

            {/* Untuk kamera streaming dengan handler klik, Popup tidak dirender agar tidak terbuka saat diklik (buka modal stream). */}
            {!(onMarkerClick && item.stream_enabled === true) && (
              <Popup className="cctv-leaflet-popup">
              <div className="p-1 max-w-[260px] space-y-2 font-sans text-foreground">
                <div className="flex items-start justify-between gap-2 border-b border-border pb-2">
                  <div>
                    <h4 className="font-bold text-sm text-foreground leading-snug">{item.nama}</h4>
                    <p className="font-mono text-xs font-semibold text-primary mt-0.5">{item.kode_cctv}</p>
                  </div>
                  <div className="flex shrink-0 flex-col items-end gap-1">
                    <span className={`inline-block rounded-full px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider ${
                      isStatusActive(item.status)
                        ? 'border border-emerald-500/20 bg-emerald-500/10 text-emerald-600'
                        : isStatusMaintenance(item.status)
                          ? 'border border-amber-500/20 bg-amber-500/10 text-amber-600'
                          : 'border border-red-500/20 bg-red-500/10 text-red-600'
                    }`}>
                      {isStatusActive(item.status) ? 'Aktif' : isStatusMaintenance(item.status) ? 'Pemeliharaan' : 'Nonaktif'}
                    </span>
                    <span className={`rounded-full border px-2 py-0.5 text-[10px] font-bold ${healthScoreClasses(item.health_score)}`}>
                      Skor {normalizeHealthScore(item.health_score)}
                    </span>
                  </div>
                </div>

                <div className="flex items-center gap-1.5 text-xs text-muted">
                  <MapPin className="h-3.5 w-3.5 text-primary shrink-0" />
                  <span className="truncate">{item.lokasi_area || 'Kawasan KIPP IKN'}</span>
                </div>
              </div>
              </Popup>
            )}
          </Marker>
        ))}
      </MapContainer>

      {/* Legend skor kesehatan: angka tetap ditampilkan agar warna bukan satu-satunya cue. */}
      <div className="absolute bottom-4 left-4 z-20 flex flex-wrap items-center gap-3 rounded-2xl border border-border bg-card/90 px-4 py-2.5 shadow-lg backdrop-blur-md text-xs font-semibold text-foreground">
        <div className="flex items-center gap-1.5">
          <ShieldCheck className="h-4 w-4 text-primary" />
          <span>Kesehatan CCTV</span>
        </div>
        <div className="h-3 w-px bg-border hidden sm:block" />
        <div className="flex items-center gap-3">
          <span className="flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-full bg-emerald-500" /> ≥70 Sehat
          </span>
          <span className="flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-full bg-amber-500" /> 40–69 Perhatian
          </span>
          <span className="flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-full bg-red-500" /> &lt;40 Kritis
          </span>
        </div>
      </div>

      {/* Fullscreen Toggle */}
      <button
        type="button"
        onClick={toggleFullscreen}
        aria-label={isFullscreen ? 'Keluar dari mode layar penuh' : 'Tampilkan peta layar penuh'}
        title={isFullscreen ? 'Keluar dari mode layar penuh (Esc)' : 'Tampilkan peta layar penuh'}
        className="absolute right-4 top-4 z-20 flex h-10 w-10 items-center justify-center rounded-xl border border-border bg-card/90 text-muted shadow-lg backdrop-blur-md transition-colors hover:border-primary/30 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
      >
        {isFullscreen ? <Minimize className="h-4 w-4" /> : <Maximize className="h-4 w-4" />}
      </button>
    </div>
  );
}

'use client';

import React, { useEffect, useMemo, useRef, useState } from 'react';
import { MapContainer, TileLayer, Marker, Popup, Tooltip } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import Link from 'next/link';
import { MapPin, Maximize, Minimize, ShieldCheck, AlertTriangle, ChevronDown } from 'lucide-react';

export interface CCTVMapItem {
  id: string;
  kode_cctv: string;
  nama: string;
  lokasi_area?: string;
  latitude?: number | null;
  longitude?: number | null;
  status: string;
  health_score?: number;
  ip_camera?: string;
  ip_nvr?: string;
  stream_enabled?: boolean;
  channel_port?: number;
  snapshot_url?: string;
  // Status live streaming (RTSP) dari endpoint /streams/live-status (admin) atau event
  // WebSocket CCTV_LIVE_STATUS_UPDATED. `null` berarti belum dicek / non-admin.
  stream_live?: boolean | null;
  // Waktu terakhir status live dicek oleh backend (string ISO). Opsional.
  stream_checked_at?: string | null;
}

interface CCTVMapProps {
  cctvs: CCTVMapItem[];
  onMarkerClick?: (cctv: CCTVMapItem) => void;
  /** Pusat awal peta [lat, lng]. Default ke pusat IKN. */
  center?: [number, number];
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

// Koordinat dianggap valid bila berupa angka bukan nol dalam rentang semesta.
const isValidCoord = (value?: number | null): boolean =>
  typeof value === 'number' && Number.isFinite(value) && value !== 0;

// Format waktu pengecekan status live menjadi HH:mm (waktu lokal). Mengembalikan
// string kosong bila input tidak valid agar tidak merusak tampilan.
const formatCheckTime = (value?: string | null): string => {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  return date.toLocaleTimeString('id-ID', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  });
};

// Fungsi pembentuk custom SVG Icon untuk Leaflet berdasarkan skor kesehatan.
// Param tambahan `isStreaming` menambah cincin hijau ganda (double-pulse) sebagai penanda kamera yang sedang streaming.
// Param `streamLive` menambah badge bulat kecil di pojok kanan-atas marker:
//   true  → hijau (#22C55E), false → merah (#EF4444), null/undefined → abu (#9CA3AF).
const createCustomMarkerIcon = (healthScore?: number, isStreaming?: boolean, streamLive?: boolean | null) => {
  const color = healthScoreColor(healthScore);
  const shadowColor = `${color}66`;

  // Warna badge status live (indikator kecil di pojok kanan-atas marker).
  const streamBadgeColor =
    streamLive === true ? '#22C55E' : streamLive === false ? '#EF4444' : '#9CA3AF';

  // Badge bulat kecil yang menandakan status live stream (RTSP).
  const streamBadge = `
    <div style="
      position: absolute;
      top: -1px;
      right: -1px;
      width: 11px;
      height: 11px;
      border-radius: 50%;
      background-color: ${streamBadgeColor};
      border: 2px solid white;
      box-shadow: 0 2px 6px rgba(0, 0, 0, 0.25);
      z-index: 3;
    "></div>
  `;

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
      ${streamBadge}
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

export default function CCTVMap({ cctvs, onMarkerClick, center }: CCTVMapProps) {
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
  // CCTV yang memiliki koordinat valid akan dirender sebagai marker. CCTV tanpa
  // koordinat valid TIDAK dibuatkan marker palsu — ditampilkan di panel terpisah.
  const processedCCTVs = useMemo(() => {
    const coordinateCounts = new Map<string, number>();

    return cctvs
      .filter((cctv) => isValidCoord(cctv.latitude) && isValidCoord(cctv.longitude))
      .map((cctv) => {
        let lat = Number(cctv.latitude);
        let lng = Number(cctv.longitude);

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

  // CCTV tanpa koordinat valid (untuk panel & status bar).
  const missingCoord = useMemo(
    () => cctvs.filter((c) => !isValidCoord(c.latitude) || !isValidCoord(c.longitude)),
    [cctvs]
  );

  // Statistik status live streaming (RTSP) dari seluruh CCTV.
  // stream_live === true → online, false → offline, null/undefined → belum dicek.
  const onlineCount = cctvs.filter((c) => c.stream_live === true).length;
  const offlineCount = cctvs.filter((c) => c.stream_live === false).length;
  const uncheckedCount = cctvs.filter((c) => c.stream_live == null).length;

  const [showMissingList, setShowMissingList] = useState(false);

  // Pusat peta: gunakan prop `center` bila tersedia, fallback ke pusat IKN.
  // `mapCenterKey` dipakai sebagai key MapContainer agar peta di-reinit bila
  // koordinat site aktif berubah (MapContainer tidak bersifat controlled).
  const mapCenter: [number, number] =
    center && isValidCoord(center[0]) && isValidCoord(center[1])
      ? center
      : [IKN_CENTER_LAT, IKN_CENTER_LNG];
  const mapCenterKey = `${mapCenter[0]},${mapCenter[1]}`;

  return (
    <>
    <div
      ref={containerRef}
      className={`relative w-full overflow-hidden transition-[height,border-radius] duration-300 ${
        isFullscreen
          ? 'h-screen rounded-none border-0'
          : 'h-[460px] rounded-3xl border border-border shadow-md'
      }`}
    >
      <MapContainer
        key={mapCenterKey}
        center={mapCenter}
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
            icon={createCustomMarkerIcon(item.health_score, item.stream_enabled, item.stream_live)}
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

                {/* Status live streaming (RTSP) */}
                <div className="flex items-center gap-1.5 text-xs">
                  {item.stream_live === true && <span className="font-semibold text-emerald-600">🟢 Online</span>}
                  {item.stream_live === false && <span className="font-semibold text-red-600">🔴 Offline</span>}
                  {item.stream_live == null && <span className="font-semibold text-gray-500">⚪ Belum dicek</span>}
                  {item.stream_checked_at && (
                    <span className="text-muted">· Dicek {formatCheckTime(item.stream_checked_at)}</span>
                  )}
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

      {/* Panel CCTV tanpa koordinat */}
      {missingCoord.length > 0 && (
        <div className="absolute bottom-4 left-4 z-20 w-72 max-w-[calc(100%-2rem)] rounded-2xl border border-border bg-card/95 p-3 shadow-lg backdrop-blur-md">
          <button
            type="button"
            onClick={() => setShowMissingList((v) => !v)}
            className="flex w-full items-center justify-between gap-2 text-left"
          >
            <span className="flex items-center gap-2 text-xs font-semibold text-amber-600 dark:text-amber-400">
              <AlertTriangle className="h-4 w-4 shrink-0" />
              {missingCoord.length} CCTV belum memiliki koordinat
            </span>
            <ChevronDown className={`h-4 w-4 shrink-0 text-muted transition-transform ${showMissingList ? 'rotate-180' : ''}`} />
          </button>

          {showMissingList && (
            <ul className="mt-2 max-h-40 space-y-1 overflow-y-auto border-t border-border pt-2">
              {missingCoord.map((item) => (
                <li key={item.id} className="flex items-center justify-between gap-2">
                  <span className="min-w-0 truncate text-xs">
                    <span className="font-mono text-[10px] text-primary">{item.kode_cctv}</span>{' '}
                    <span className="text-foreground">{item.nama}</span>
                  </span>
                  <Link
                    href={`/cctv?edit=${item.id}`}
                    className="shrink-0 rounded-lg border border-border px-2 py-0.5 text-[10px] font-semibold text-primary transition-colors hover:bg-background"
                  >
                    Edit
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>

    {/* Status bar di bawah peta: jumlah yang ditampilkan vs total */}
    <div className="mt-2 flex flex-wrap items-center justify-between gap-2 rounded-xl border border-border bg-card px-4 py-2.5 text-xs text-muted">
      <span>
        Menampilkan <span className="font-bold text-foreground">{processedCCTVs.length}</span> dari{' '}
        <span className="font-bold text-foreground">{cctvs.length}</span> CCTV
      </span>
      {missingCoord.length > 0 && (
        <span className="font-semibold text-amber-600 dark:text-amber-400">
          {missingCoord.length} belum memiliki koordinat
        </span>
      )}
    </div>

    {/* Status bar live streaming (RTSP): online / offline / belum dicek */}
    <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 rounded-xl border border-border bg-card px-4 py-2.5 text-xs font-semibold">
      <span className="inline-flex items-center gap-1.5">
        <span className="h-2.5 w-2.5 rounded-full bg-emerald-500" />
        <span className="text-emerald-600">{onlineCount} Online</span>
      </span>
      <span className="inline-flex items-center gap-1.5">
        <span className="h-2.5 w-2.5 rounded-full bg-red-500" />
        <span className="text-red-600">{offlineCount} Offline</span>
      </span>
      <span className="inline-flex items-center gap-1.5">
        <span className="h-2.5 w-2.5 rounded-full bg-gray-400" />
        <span className="text-gray-500">{uncheckedCount} Belum dicek</span>
      </span>
    </div>
    </>
  );
}

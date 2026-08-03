'use client';

import React, { useEffect, useMemo, useRef, useState } from 'react';
import { MapContainer, TileLayer, Marker, Popup, Tooltip } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import { MapPin, Eye, Maximize, Minimize, Network, ShieldCheck } from 'lucide-react';
import Link from 'next/link';

export interface CCTVMapItem {
  id: string;
  kode_cctv: string;
  nama: string;
  lokasi_area?: string;
  latitude?: string;
  longitude?: string;
  status: string;
  ip_camera?: string;
  ip_nvr?: string;
}

interface CCTVMapProps {
  cctvs: CCTVMapItem[];
}

// Koordinat Pusat IKN Nusantara (KIPP IKN)
const IKN_CENTER_LAT = -0.9723;
const IKN_CENTER_LNG = 116.7083;

const normalizeStatus = (s: string) => (s || '').trim().toUpperCase();
const isStatusActive = (s: string) => ['ACTIVE', 'AKTIF'].includes(normalizeStatus(s));
const isStatusMaintenance = (s: string) => ['MAINTENANCE', 'PEMELIHARAAN'].includes(normalizeStatus(s));

// Fungsi pembentuk custom SVG Icon untuk Leaflet berdasarkan status CCTV
const createCustomMarkerIcon = (status: string) => {
  let color = '#EF4444'; // Red (Nonaktif / Rusak)
  let shadowColor = 'rgba(239, 68, 68, 0.4)';

  if (isStatusActive(status)) {
    color = '#10B981'; // Green (Aktif)
    shadowColor = 'rgba(16, 185, 129, 0.4)';
  } else if (isStatusMaintenance(status)) {
    color = '#F59E0B'; // Amber (Pemeliharaan)
    shadowColor = 'rgba(245, 158, 11, 0.4)';
  }

  const svgHtml = `
    <div style="
      position: relative;
      width: 34px;
      height: 34px;
      display: flex;
      align-items: center;
      justify-content: center;
    ">
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

export default function CCTVMap({ cctvs }: CCTVMapProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);

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
            icon={createCustomMarkerIcon(item.status)}
          >
            <Tooltip direction="top" offset={[0, -15]} opacity={0.95}>
              <div className="text-xs font-bold font-sans">
                {item.nama} <span className="font-mono text-[10px] text-muted">({item.kode_cctv})</span>
              </div>
            </Tooltip>

            <Popup className="cctv-leaflet-popup">
              <div className="p-1 max-w-[260px] space-y-3 font-sans text-foreground">
                <div className="flex items-start justify-between gap-2 border-b border-border pb-2">
                  <div>
                    <h4 className="font-bold text-sm text-foreground leading-snug">{item.nama}</h4>
                    <p className="font-mono text-xs font-semibold text-primary mt-0.5">{item.kode_cctv}</p>
                  </div>
                  <span className={`inline-block shrink-0 px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider ${
                    isStatusActive(item.status)
                      ? 'bg-emerald-500/10 text-emerald-600 border border-emerald-500/20'
                      : isStatusMaintenance(item.status)
                        ? 'bg-amber-500/10 text-amber-600 border border-amber-500/20'
                        : 'bg-red-500/10 text-red-600 border border-red-500/20'
                  }`}>
                    {isStatusActive(item.status) ? 'Aktif' : isStatusMaintenance(item.status) ? 'Pemeliharaan' : 'Nonaktif'}
                  </span>
                </div>

                <div className="space-y-1.5 text-xs text-muted">
                  <div className="flex items-center gap-1.5">
                    <MapPin className="h-3.5 w-3.5 text-primary shrink-0" />
                    <span className="truncate">{item.lokasi_area || 'Kawasan KIPP IKN'}</span>
                  </div>
                  {item.ip_camera && (
                    <div className="flex items-center gap-1.5 font-mono text-[11px]">
                      <Network className="h-3.5 w-3.5 text-muted shrink-0" />
                      <span>IP: {item.ip_camera}</span>
                    </div>
                  )}
                </div>

                <div className="pt-2 border-t border-border flex justify-end">
                  <Link
                    href="/cctv"
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-primary text-white text-xs font-semibold hover:bg-primary-hover transition-colors shadow-sm"
                  >
                    <Eye className="h-3.5 w-3.5" />
                    <span>Lihat di Master CCTV</span>
                  </Link>
                </div>
              </div>
            </Popup>
          </Marker>
        ))}
      </MapContainer>

      {/* Legend Card Overlay */}
      <div className="absolute bottom-4 left-4 z-20 flex flex-wrap items-center gap-3 rounded-2xl border border-border bg-card/90 px-4 py-2.5 shadow-lg backdrop-blur-md text-xs font-semibold text-foreground">
        <div className="flex items-center gap-1.5">
          <ShieldCheck className="h-4 w-4 text-primary" />
          <span>Sebaran CCTV IKN</span>
        </div>
        <div className="h-3 w-px bg-border hidden sm:block" />
        <div className="flex items-center gap-3">
          <span className="flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-full bg-emerald-500" /> Aktif
          </span>
          <span className="flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-full bg-amber-500" /> Pemeliharaan
          </span>
          <span className="flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-full bg-red-500" /> Nonaktif
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

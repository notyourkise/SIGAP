'use client';

import React from 'react';
import { MapContainer, TileLayer, Marker, Popup, Tooltip } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import { MapPin, Video, ClipboardCheck } from 'lucide-react';

// Koordinat Pusat IKN Nusantara (KIPP IKN) — dipakai sebagai fallback
const IKN_CENTER_LAT = -0.9723;
const IKN_CENTER_LNG = 116.7083;

// Warna penanda: CCTV = biru/primary, Lokasi Pengecekan = oranye
const CCTV_MARKER_COLOR = '#2563EB';
const FIELD_CHECK_MARKER_COLOR = '#F59E0B';

// Interface data CCTV yang dibutuhkan untuk peta
export interface FieldCheckMapCctv {
  kode_cctv?: string;
  nama?: string;
  lokasi_area?: string;
  latitude?: string;
  longitude?: string;
}

// Props komponen mini-map: data CCTV + koordinat lokasi pengecekan lapangan
interface FieldCheckLocationMapProps {
  cctv: FieldCheckMapCctv;
  checkLatitude: string;
  checkLongitude: string;
}

// Fungsi membentuk marker bulat berwarna sederhana untuk Leaflet via L.divIcon
const createSimpleMarkerIcon = (color: string) =>
  L.divIcon({
    html: `
      <div style="
        width: 24px;
        height: 24px;
        border-radius: 50%;
        background-color: ${color};
        border: 2.5px solid white;
        box-shadow: 0 3px 10px rgba(0,0,0,0.35);
      "></div>
    `,
    className: 'field-check-simple-marker',
    iconSize: [24, 24],
    iconAnchor: [12, 12],
    popupAnchor: [0, -12],
  });

export default function FieldCheckLocationMap({
  cctv,
  checkLatitude,
  checkLongitude,
}: FieldCheckLocationMapProps) {
  // Koordinat CCTV: gunakan data lat/long jika valid, jika tidak fallback ke pusat IKN
  const cctvLat = parseFloat(cctv.latitude || '');
  const cctvLng = parseFloat(cctv.longitude || '');
  const cctvPos: [number, number] =
    !Number.isNaN(cctvLat) && !Number.isNaN(cctvLng) && cctvLat !== 0 && cctvLng !== 0
      ? [cctvLat, cctvLng]
      : [IKN_CENTER_LAT, IKN_CENTER_LNG];

  // Koordinat lokasi pengecekan (dijamin terisi karena dipanggil secara kondisional)
  const checkLat = parseFloat(checkLatitude);
  const checkLng = parseFloat(checkLongitude);
  const checkPos: [number, number] = [checkLat, checkLng];

  return (
    <div className="relative w-full overflow-hidden rounded-2xl border border-border shadow-md h-[320px]">
      <MapContainer
        center={[IKN_CENTER_LAT, IKN_CENTER_LNG]}
        zoom={14}
        scrollWheelZoom={false}
        className="w-full h-full z-10"
      >
        <TileLayer
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        />

        {/* Penanda lokasi CCTV */}
        <Marker position={cctvPos} icon={createSimpleMarkerIcon(CCTV_MARKER_COLOR)}>
          <Tooltip direction="top" offset={[0, -12]} opacity={0.95}>
            <div className="text-xs font-bold font-sans">CCTV: {cctv.nama || cctv.kode_cctv || '-'}</div>
          </Tooltip>
          <Popup className="cctv-leaflet-popup">
            <div className="p-1 max-w-[220px] space-y-1 font-sans text-foreground text-xs">
              <div className="flex items-center gap-1.5 font-semibold">
                <Video className="h-3.5 w-3.5 text-primary shrink-0" />
                <span>CCTV</span>
              </div>
              <p className="font-semibold text-sm leading-snug">{cctv.nama || '-'}</p>
              <p className="font-mono text-[11px] text-muted">{cctv.kode_cctv || '-'}</p>
              <div className="flex items-center gap-1.5 text-muted">
                <MapPin className="h-3 w-3 shrink-0" />
                <span className="truncate">{cctv.lokasi_area || 'Kawasan KIPP IKN'}</span>
              </div>
            </div>
          </Popup>
        </Marker>

        {/* Penanda lokasi pengecekan lapangan */}
        <Marker position={checkPos} icon={createSimpleMarkerIcon(FIELD_CHECK_MARKER_COLOR)}>
          <Tooltip direction="top" offset={[0, -12]} opacity={0.95}>
            <div className="text-xs font-bold font-sans">Lokasi Pengecekan</div>
          </Tooltip>
          <Popup className="cctv-leaflet-popup">
            <div className="p-1 max-w-[220px] space-y-1 font-sans text-foreground text-xs">
              <div className="flex items-center gap-1.5 font-semibold">
                <ClipboardCheck className="h-3.5 w-3.5 text-amber-500 shrink-0" />
                <span>Lokasi Pengecekan</span>
              </div>
              <p className="text-muted">Titik tempat teknisi melakukan pengecekan lapangan.</p>
            </div>
          </Popup>
        </Marker>
      </MapContainer>

      {/* Legend penanda: warna CCTV vs Lokasi Pengecekan */}
      <div className="absolute bottom-4 left-4 z-20 flex flex-wrap items-center gap-3 rounded-xl border border-border bg-card/90 px-3.5 py-2 shadow-lg backdrop-blur-md text-[11px] font-semibold text-foreground">
        <span className="flex items-center gap-1.5">
          <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: CCTV_MARKER_COLOR }} />
          CCTV
        </span>
        <span className="h-3 w-px bg-border" />
        <span className="flex items-center gap-1.5">
          <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: FIELD_CHECK_MARKER_COLOR }} />
          Lokasi Pengecekan
        </span>
      </div>
    </div>
  );
}

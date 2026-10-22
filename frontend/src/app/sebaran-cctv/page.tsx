'use client';

import { useCallback, useEffect, useState } from 'react';
import dynamic from 'next/dynamic';
import DashboardLayout from '@/components/DashboardLayout';
import { useWebSocket } from '@/context/WebSocketContext';
import { apiRequest, getAllSitesForSelect, getCookie } from '@/lib/api';
import type { Site } from '@/lib/api';
import type { CCTVMapItem } from '@/components/CCTVMap';
import type { StreamState } from '@/lib/live-status';
import { MapPinned } from 'lucide-react';

const CCTVMap = dynamic(() => import('@/components/CCTVMap'), {
  ssr: false,
  loading: () => (
    <div className="flex h-[560px] w-full items-center justify-center rounded-3xl border border-border bg-card">
      <div className="flex items-center gap-3">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
        <span className="text-sm font-semibold text-muted">Memuat peta GIS IKN…</span>
      </div>
    </div>
  ),
});

// StreamModal dibuat oleh agent lain — dimuat dinamis agar tidak memberatkan bundle awal.
const StreamModal = dynamic(() => import('@/components/StreamModal'), {
  ssr: false,
});

interface PaginationPayload<T> {
  rows: T[];
  pagination: {
    total_rows: number;
  };
}

interface ZoneOption {
  id: string;
  nama: string;
}

// Respons endpoint GET /api/streams/live-status (admin-only). Endpoint ini
// DILINDUNGI oleh RequireRole admin, sehingga pada non-admin fetch bisa 403
// dan data live status dibiarkan null.
interface LiveStatusItem {
  cctv_id: string;
  nama?: string;
  kode_cctv?: string;
  stream_live?: boolean | null;
  // Status empat keadaan: live, idle (on-demand tanpa penonton), offline, unknown.
  stream_state?: StreamState | null;
  stream_checked_at?: string | null;
  // Sprint 37-PATCH: strategi probe jaringan — Metro-E (ICMP, binary) vs GSM (4-state).
  network_type?: 'metro-e' | 'gsm' | null;
  ping_latency_ms?: number | null;
  stream_last_active_at?: string | null;
  stream_verified_state?: StreamState | null;
  stream_verified_at?: string | null;
  stream_verified_method?: string | null;
  stream_verify_failures?: number | null;
  stream_verify_error?: string | null;
}

interface LiveStatusResponse {
  status: string;
  message?: string;
  summary?: {
    online: number;
    idle?: number;
    offline: number;
    unchecked: number;
  };
  data?: LiveStatusItem[];
}

const normalizeStatus = (status: string) => status.trim().toUpperCase();
const isActiveCCTV = (status: string) => ['ACTIVE', 'AKTIF'].includes(normalizeStatus(status));
const isInactiveCCTV = (status: string) => ['INACTIVE', 'NONAKTIF', 'BROKEN', 'RUSAK'].includes(normalizeStatus(status));
const isMaintenanceCCTV = (status: string) => ['MAINTENANCE', 'PEMELIHARAAN'].includes(normalizeStatus(status));

export default function SebaranCCTVPage() {
  const [cctvs, setCctvs] = useState<CCTVMapItem[]>([]);
  const [totalCctv, setTotalCctv] = useState(0);
  const [health, setHealth] = useState<'Memeriksa' | 'Terhubung' | 'Tidak terhubung'>('Memeriksa');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedCctv, setSelectedCctv] = useState<CCTVMapItem | null>(null);
  const [zones, setZones] = useState<ZoneOption[]>([]);
  const [selectedZoneId, setSelectedZoneId] = useState('');
  const [zonesLoading, setZonesLoading] = useState(true);

  // Pusat peta — diarahkan ke koordinat site aktif bila tersedia (multi-site).
  const [mapCenter, setMapCenter] = useState<[number, number] | undefined>(undefined);

  // Pusatkan peta ke koordinat site aktif (dari cookie sigap_site_id).
  // Peta tetap difilter oleh header X-Site-ID secara otomatis lewat apiRequest.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const response = await getAllSitesForSelect();
        if (cancelled || response.status !== 'success' || !response.data) return;

        const activeSiteId = getCookie('sigap_site_id');
        const active: Site | undefined = activeSiteId
          ? response.data.find((s) => s.id === activeSiteId)
          : response.data.find((s) => s.is_default);

        if (
          active &&
          typeof active.koordinat_lat === 'number' &&
          typeof active.koordinat_lng === 'number'
        ) {
          setMapCenter([active.koordinat_lat, active.koordinat_lng]);
        }
      } catch {
        // Bila daftar site gagal dimuat, peta tetap di pusat default (tidak fatal).
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;

    const loadZones = async () => {
      try {
        const response = await apiRequest<ZoneOption[]>('/zones/all');
        if (!cancelled && response.status === 'success') {
          setZones(response.data ?? []);
        }
      } catch {
        // Filter zona bersifat tambahan; kegagalan memuat opsi tidak menghalangi peta.
      } finally {
        if (!cancelled) setZonesLoading(false);
      }
    };

    void loadZones();
    return () => {
      cancelled = true;
    };
  }, []);

  const loadMap = useCallback(async () => {
    setLoading(true);
    setError('');
    setHealth('Memeriksa');
    try {
      const zoneQuery = selectedZoneId ? `&zone_id=${encodeURIComponent(selectedZoneId)}` : '';
      const cctvQuery = `/cctv?page=1&limit=1000&sort_by=nama&order=asc${zoneQuery}`;
      const [cctvResult, healthResult, liveResult] = await Promise.allSettled([
        apiRequest<PaginationPayload<CCTVMapItem>>(cctvQuery),
        apiRequest<Record<string, unknown>>('/health'),
        // Endpoint live-status admin-only: pada non-admin bisa mengembalikan 403,
        // maka Promise.allSettled menanganinya dengan aman (stream_live tetap null).
        apiRequest<LiveStatusResponse>('/streams/live-status'),
      ]);

      if (cctvResult.status === 'fulfilled') {
        const cctvData = cctvResult.value.data;
        const cctvRows = cctvData?.rows || [];

        // Enrich baris CCTV dengan status live streaming dari /streams/live-status.
        // Bila endpoint gagal (mis. 403 karena non-admin) atau tidak tersedia, seluruh
        // baris tetap dipakai dengan stream_live bernilai undefined/null.
        let enrichedRows = cctvRows;
        if (liveResult.status === 'fulfilled') {
          const liveItems = liveResult.value.data?.data || [];
          const liveMap = new Map<string, LiveStatusItem>();
          for (const item of liveItems) {
            liveMap.set(item.cctv_id, item);
          }
          enrichedRows = cctvRows.map((c) => {
            const live = liveMap.get(c.id);
            return live
              ? {
                  ...c,
                  stream_live: live.stream_live ?? null,
                  stream_state: live.stream_state ?? null,
                  stream_checked_at: live.stream_checked_at ?? null,
                  // Sprint 37-PATCH: bawa field strategi probe jaringan dari live-status
                  // agar badge Metro-E vs GSM berfungsi; bila field absen (data lama)
                  // tetap undefined → komponen memakai fallback perilaku lama.
                  network_type: live.network_type ?? null,
                  ping_latency_ms: live.ping_latency_ms ?? null,
                  stream_last_active_at: live.stream_last_active_at ?? null,
                  stream_verified_state: live.stream_verified_state ?? null,
                  stream_verified_at: live.stream_verified_at ?? null,
                  stream_verified_method: live.stream_verified_method ?? null,
                }
              : c;
          });
        }

        setCctvs(enrichedRows);
        setTotalCctv(cctvData?.pagination?.total_rows ?? cctvRows.length);
      } else {
        setError(cctvResult.reason instanceof Error ? cctvResult.reason.message : 'Data CCTV belum dapat dimuat. Muat ulang halaman.');
      }

      setHealth(healthResult.status === 'fulfilled' ? 'Terhubung' : 'Tidak terhubung');
    } finally {
      setLoading(false);
    }
  }, [selectedZoneId]);

  useEffect(() => {
    const timer = setTimeout(loadMap, 0);
    return () => clearTimeout(timer);
  }, [loadMap]);

  const { subscribe } = useWebSocket();

  useEffect(() => {
    const unsubCCTV = subscribe('CCTV_UPDATED', () => loadMap());

    // Update status live streaming secara real-time tanpa reload penuh peta.
    // Payload: { cctv_id, stream_live, stream_state, stream_checked_at, nama, kode_cctv }
    // (+ network_type, ping_latency_ms, stream_last_active_at).
    const unsubLive = subscribe('CCTV_LIVE_STATUS_UPDATED', (payload: any) => {
      if (payload?.cctv_id) {
        setCctvs((prev) =>
          prev.map((c) => {
            if (c.id !== payload.cctv_id) return c;
            const prevItem = c;
            return {
              ...c,
              stream_live: payload.stream_live ?? prevItem.stream_live ?? null,
              stream_state: payload.stream_state ?? prevItem.stream_state ?? null,
              stream_checked_at: payload.stream_checked_at ?? prevItem.stream_checked_at ?? null,
              // Sprint 37-PATCH: pertahankan field probe jaringan; payload tanpa field
              // (event lama) tidak menimpa nilai yang sudah ada di item.
              ping_latency_ms: payload.ping_latency_ms ?? prevItem.ping_latency_ms ?? null,
              network_type: payload.network_type ?? prevItem.network_type ?? null,
              // Payload verifikasi aktif membawa waktu ini; payload probe pasif hanya
              // membawanya saat live. Jangan hapus riwayat saat field tidak ada.
              stream_last_active_at: payload.stream_last_active_at ?? prevItem.stream_last_active_at ?? null,
              stream_verified_state: payload.stream_verified_state ?? prevItem.stream_verified_state ?? null,
              stream_verified_at: payload.stream_verified_at ?? prevItem.stream_verified_at ?? null,
              stream_verified_method: payload.stream_verified_method ?? prevItem.stream_verified_method ?? null,
            };
          })
        );
      }
    });

    return () => {
      unsubCCTV();
      unsubLive();
    };
  }, [subscribe, loadMap]);

  const active = cctvs.filter((item) => isActiveCCTV(item.status)).length;
  const inactive = cctvs.filter((item) => isInactiveCCTV(item.status)).length;
  const maintenance = cctvs.filter((item) => isMaintenanceCCTV(item.status)).length;
  const selectedZoneName = zones.find((zone) => zone.id === selectedZoneId)?.nama;

  return (
    <DashboardLayout>
      <div className="space-y-8">
        <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-[-0.025em]">Peta Sebaran CCTV</h1>
            <p className="mt-1 max-w-2xl text-sm text-muted">
              Visualisasi lokasi pemasangan CCTV di kawasan IKN Nusantara beserta status perangkat.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <div className="flex items-center gap-2 text-xs font-semibold">
              <span className={`h-2 w-2 rounded-full ${health === 'Terhubung' ? 'bg-success' : health === 'Memeriksa' ? 'animate-pulse bg-warning' : 'bg-error'}`} />
              <span>{health === 'Terhubung' ? 'Layanan normal' : health}</span>
            </div>
            <label className="flex min-h-10 items-center gap-2 rounded-xl border border-border bg-card px-3 text-sm focus-within:border-primary focus-within:ring-2 focus-within:ring-primary/20">
              <MapPinned className="h-4 w-4 shrink-0 text-primary" aria-hidden="true" />
              <span className="text-xs font-semibold text-muted">Zona</span>
              <select
                value={selectedZoneId}
                onChange={(event) => {
                  setSelectedZoneId(event.target.value);
                  setSelectedCctv(null);
                }}
                disabled={zonesLoading}
                aria-label="Filter CCTV berdasarkan zona"
                className="max-w-44 bg-transparent text-sm font-semibold text-foreground outline-none disabled:cursor-wait disabled:opacity-60"
              >
                <option value="">{zonesLoading ? 'Memuat zona…' : 'Semua zona'}</option>
                {zones.map((zone) => (
                  <option key={zone.id} value={zone.id}>{zone.nama}</option>
                ))}
              </select>
            </label>
          </div>
        </header>

        {error && <div role="alert" className="rounded-xl border border-error/25 bg-error/10 px-4 py-3 text-sm text-error">{error}</div>}
        {!loading && health === 'Tidak terhubung' && (
          <div role="alert" className="rounded-xl border border-warning/30 bg-warning/10 px-4 py-3 text-sm">
            Layanan SIGAP tidak dapat dijangkau. Periksa koneksi atau hubungi pengelola sistem.
          </div>
        )}
        {!loading && !error && selectedZoneId && cctvs.length === 0 && (
          <div role="status" className="rounded-xl border border-border bg-card px-4 py-3 text-sm text-muted">
            Belum ada CCTV pada zona <span className="font-semibold text-foreground">{selectedZoneName ?? 'yang dipilih'}</span>.
          </div>
        )}

        {loading ? (
          <div className="h-[560px] animate-pulse rounded-3xl border border-border bg-card" role="status" aria-label="Memuat peta sebaran CCTV" />
        ) : (
          <>
            <section aria-label="Ringkasan sebaran CCTV" className="grid overflow-hidden rounded-2xl border border-border bg-card sm:grid-cols-2 lg:grid-cols-4">
              <div className="flex min-h-20 items-center justify-between gap-6 border-b border-border px-5 py-5 sm:border-r lg:border-b-0 lg:last:border-r-0">
                <div className="flex items-center gap-2.5">
                  <span className="h-2.5 w-2.5 rounded-full bg-blue-500" />
                  <span className="text-sm font-medium text-muted">Total CCTV</span>
                </div>
                <strong className="text-2xl font-bold tabular-nums">{totalCctv}</strong>
              </div>
              <div className="flex min-h-20 items-center justify-between gap-6 border-b border-border px-5 py-5 lg:border-b-0 lg:border-r">
                <div className="flex items-center gap-2.5">
                  <span className="h-2.5 w-2.5 rounded-full bg-success" />
                  <span className="text-sm font-medium text-muted">Aktif</span>
                </div>
                <strong className="text-2xl font-bold tabular-nums">{active}</strong>
              </div>
              <div className="flex min-h-20 items-center justify-between gap-6 border-b border-border px-5 py-5 sm:border-b-0 sm:border-r">
                <div className="flex items-center gap-2.5">
                  <span className="h-2.5 w-2.5 rounded-full bg-warning" />
                  <span className="text-sm font-medium text-muted">Pemeliharaan</span>
                </div>
                <strong className="text-2xl font-bold tabular-nums">{maintenance}</strong>
              </div>
              <div className="flex min-h-20 items-center justify-between gap-6 px-5 py-5">
                <div className="flex items-center gap-2.5">
                  <span className="h-2.5 w-2.5 rounded-full bg-error" />
                  <span className="text-sm font-medium text-muted">Nonaktif</span>
                </div>
                <strong className="text-2xl font-bold tabular-nums">{inactive}</strong>
              </div>
            </section>

            <section aria-label="Peta GIS Sebaran CCTV IKN">
              <CCTVMap
                cctvs={cctvs}
                center={mapCenter}
                onMarkerClick={(cctv) => cctv.stream_enabled && setSelectedCctv(cctv)}
              />
            </section>

            {selectedCctv && <StreamModal cctv={selectedCctv} onClose={() => setSelectedCctv(null)} />}
          </>
        )}
      </div>
    </DashboardLayout>
  );
}

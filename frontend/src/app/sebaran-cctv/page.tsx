'use client';

import { useCallback, useEffect, useState } from 'react';
import dynamic from 'next/dynamic';
import DashboardLayout from '@/components/DashboardLayout';
import { useWebSocket } from '@/context/WebSocketContext';
import { apiRequest } from '@/lib/api';
import type { CCTVMapItem } from '@/components/CCTVMap';
import { MonitorPlay, LayoutGrid, Map as MapIcon, Filter } from 'lucide-react';

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

// Lazy load StreamGridView hanya ketika dibutuhkan (mode grid aktif) dan hanya di sisi klien.
const StreamGridView = dynamic(() => import('@/components/StreamGridView'), {
  ssr: false,
  loading: () => (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4" role="status" aria-label="Memuat grid streaming CCTV">
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i} className="h-64 animate-pulse rounded-2xl border border-border bg-card" />
      ))}
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
  const [viewMode, setViewMode] = useState<'map' | 'grid'>('map');

  // Filter peta per zona operasional (Task 11.4, Sprint 28)
  const [zones, setZones] = useState<Array<{ id: string; nama: string }>>([]);
  const [zoneFilter, setZoneFilter] = useState('');

  // Muat daftar zona untuk dropdown filter.
  const loadZones = useCallback(async () => {
    try {
      const response = await apiRequest<Array<{ id: string; nama: string }>>('/zones/all');
      if (response.status === 'success' && response.data) {
        setZones(response.data);
      }
    } catch {
      // Bila daftar zona gagal dimuat, filter zona dinonaktifkan (tidak fatal).
      setZones([]);
    }
  }, []);

  useEffect(() => {
    void loadZones();
  }, [loadZones]);

  const loadMap = useCallback(async () => {
    setLoading(true);
    setError('');
    setHealth('Memeriksa');
    try {
      const cctvQuery = `/cctv?page=1&limit=1000&sort_by=nama&order=asc${zoneFilter ? `&zone_id=${encodeURIComponent(zoneFilter)}` : ''}`;
      const [cctvResult, healthResult] = await Promise.allSettled([
        apiRequest<PaginationPayload<CCTVMapItem>>(cctvQuery),
        apiRequest<Record<string, unknown>>('/health'),
      ]);

      if (cctvResult.status === 'fulfilled') {
        const cctvData = cctvResult.value.data;
        const cctvRows = cctvData?.rows || [];
        setCctvs(cctvRows);
        setTotalCctv(cctvData?.pagination?.total_rows ?? cctvRows.length);
      } else {
        setError(cctvResult.reason instanceof Error ? cctvResult.reason.message : 'Data CCTV belum dapat dimuat. Muat ulang halaman.');
      }

      setHealth(healthResult.status === 'fulfilled' ? 'Terhubung' : 'Tidak terhubung');
    } finally {
      setLoading(false);
    }
  }, [zoneFilter]);

  useEffect(() => {
    const timer = setTimeout(loadMap, 0);
    return () => clearTimeout(timer);
  }, [loadMap]);

  const { subscribe } = useWebSocket();

  useEffect(() => {
    const unsubCCTV = subscribe('CCTV_UPDATED', () => loadMap());

    return () => {
      unsubCCTV();
    };
  }, [subscribe, loadMap]);

  const active = cctvs.filter((item) => isActiveCCTV(item.status)).length;
  const inactive = cctvs.filter((item) => isInactiveCCTV(item.status)).length;
  const maintenance = cctvs.filter((item) => isMaintenanceCCTV(item.status)).length;
  const streamingCount = cctvs.filter((item) => item.stream_enabled === true).length;

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
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-2 text-xs font-semibold">
              <span className={`h-2 w-2 rounded-full ${health === 'Terhubung' ? 'bg-success' : health === 'Memeriksa' ? 'animate-pulse bg-warning' : 'bg-error'}`} />
              <span>{health === 'Terhubung' ? 'Layanan normal' : health}</span>
            </div>
            <button
              type="button"
              onClick={loadMap}
              disabled={loading}
              className="rounded-xl border border-border bg-card px-3 py-2 text-xs font-semibold transition-colors hover:border-primary disabled:cursor-wait disabled:opacity-60"
            >
              {loading ? 'Memuat…' : 'Muat ulang'}
            </button>
            {/* Filter peta per zona operasional */}
            <div className="flex items-center gap-1.5">
              <Filter className="h-4 w-4 text-muted" aria-hidden="true" />
              <select
                aria-label="Filter zona"
                value={zoneFilter}
                onChange={(e) => {
                  setZoneFilter(e.target.value);
                }}
                disabled={loading || zones.length === 0}
                className="rounded-xl border border-border bg-card px-3 py-2 text-xs font-semibold transition-colors hover:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30 disabled:cursor-not-allowed disabled:opacity-60"
              >
                <option value="">Semua zona</option>
                {zones.map((z) => (
                  <option key={z.id} value={z.id}>{z.nama}</option>
                ))}
              </select>
            </div>
            {/* Toggle tampilan Peta / Grid */}
            <div className="flex items-center gap-1 rounded-xl border border-border bg-card p-1">
              <button
                type="button"
                onClick={() => setViewMode('map')}
                aria-label="Tampilan peta"
                title="Tampilan peta"
                className={`flex h-8 w-8 items-center justify-center rounded-lg transition-colors ${
                  viewMode === 'map' ? 'bg-primary text-white shadow-sm' : 'text-muted hover:text-foreground'
                }`}
              >
                <MapIcon className="h-4 w-4" />
              </button>
              <button
                type="button"
                onClick={() => setViewMode('grid')}
                aria-label="Tampilan grid"
                title="Tampilan grid"
                className={`flex h-8 w-8 items-center justify-center rounded-lg transition-colors ${
                  viewMode === 'grid' ? 'bg-primary text-white shadow-sm' : 'text-muted hover:text-foreground'
                }`}
              >
                <LayoutGrid className="h-4 w-4" />
              </button>
            </div>
            {/* Status kamera streaming aktif */}
            <div className="flex items-center gap-2 rounded-xl border border-emerald-500/20 bg-emerald-500/10 px-3 py-2 text-xs font-semibold text-emerald-600">
              <MonitorPlay className="h-4 w-4" />
              <span>{streamingCount} kamera streaming aktif</span>
            </div>
          </div>
        </header>

        {error && <div role="alert" className="rounded-xl border border-error/25 bg-error/10 px-4 py-3 text-sm text-error">{error}</div>}
        {!loading && health === 'Tidak terhubung' && (
          <div role="alert" className="rounded-xl border border-warning/30 bg-warning/10 px-4 py-3 text-sm">
            Layanan SIGAP tidak dapat dijangkau. Periksa koneksi atau hubungi pengelola sistem.
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

            {viewMode === 'map' ? (
              <section aria-label="Peta GIS Sebaran CCTV IKN">
                <CCTVMap
                  cctvs={cctvs}
                  onMarkerClick={(cctv) => cctv.stream_enabled && setSelectedCctv(cctv)}
                />
              </section>
            ) : (
              <section aria-label="Grid streaming CCTV IKN">
                <StreamGridView
                  cctvs={cctvs.filter((c) => c.stream_enabled)}
                  onCctvClick={setSelectedCctv}
                />
              </section>
            )}

            {selectedCctv && <StreamModal cctv={selectedCctv} onClose={() => setSelectedCctv(null)} />}
          </>
        )}
      </div>
    </DashboardLayout>
  );
}

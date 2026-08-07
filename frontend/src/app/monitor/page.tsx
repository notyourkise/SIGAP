'use client';

// =============================================================================
// Monitor CCTV — Tampilan Multi-Stream (Multi-View Grid).
//
// Halaman khusus admin untuk menampilkan beberapa kamera streaming CCTV secara
// berdampingan dalam bentuk grid yang bisa diatur (2×2, 3×3, 4×4). Setiap sel
// grid memiliki dropdown untuk memilih kamera mana yang akan ditampilkan, tombol
// fullscreen per-sel, dan tombol pop-out ke jendela terpisah.
//
// Alur data:
//   1. Muat daftar CCTV dari GET /api/cctv?page=1&limit=1000.
//   2. Filter hanya kamera dengan stream_enabled === true (siap streaming).
//   3. Setiap sel grid menyimpan id CCTV terpilih; bila terisi, WHEPPlayer
//      dirender dengan streamPath = 'cam-' + id.
// =============================================================================

import { useCallback, useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import dynamic from 'next/dynamic';
import DashboardLayout from '@/components/DashboardLayout';
import { apiRequest } from '@/lib/api';
import { useAuth } from '@/context/AuthContext';
import {
  MonitorPlay,
  LayoutGrid,
  Maximize2,
  Minimize2,
  ExternalLink,
  LoaderCircle,
  VideoOff,
} from 'lucide-react';

// WHEPPlayer dimuat dinamis (lazy) hanya di sisi klien agar aman untuk SSR,
// karena komponen ini memakai WebRTC / Fullscreen API yang hanya ada di browser.
const WHEPPlayer = dynamic(() => import('@/components/WHEPPlayer'), {
  ssr: false,
  loading: () => (
    <div className="flex h-full w-full items-center justify-center gap-2 bg-black/60 text-white">
      <LoaderCircle className="h-6 w-6 animate-spin text-accent" />
      <span className="text-xs font-semibold">Menyiapkan pemutar…</span>
    </div>
  ),
});

// Data CCTV sesuai respons GET /api/cctv.
interface CCTV {
  id: string;
  kode_cctv: string;
  nama: string;
  stream_enabled: boolean;
  channel_port?: number;
  // field lain diabaikan
}

interface PaginationPayload<T> {
  rows: T[];
  pagination: { total_rows: number };
}

// Preset ukuran grid: jumlah sel & kelas grid responsif.
const LAYOUT_PRESETS: Record<'2x2' | '3x3' | '4x4', { cells: number; gridClass: string }> = {
  '2x2': { cells: 4, gridClass: 'grid-cols-1 sm:grid-cols-2' },
  '3x3': { cells: 9, gridClass: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3' },
  '4x4': { cells: 16, gridClass: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-4' },
};

// Konversi id CCTV menjadi streamPath yang dipakai WHEPPlayer ('cam-{id}').
const toStreamPath = (id: string) => `cam-${id}`;

export default function MonitorPage() {
  const { user } = useAuth();
  const router = useRouter();

  // Data & status muat daftar CCTV.
  const [cctvs, setCctvs] = useState<CCTV[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Preset grid aktif (2×2 / 3×3 / 4×4).
  const [layout, setLayout] = useState<'2x2' | '3x3' | '4x4'>('3x3');

  // Daftar id CCTV terpilih per sel; urutan sel = index array.
  const [selection, setSelection] = useState<Record<number, string>>({});

  // Ref kontainer untuk fullscreen seluruh grid.
  const gridRef = useRef<HTMLDivElement>(null);
  const [gridFullscreen, setGridFullscreen] = useState(false);

  // --- Guard admin: bukan admin dialihkan ke dashboard ---
  useEffect(() => {
    if (!loading && user?.role !== 'admin') {
      router.replace('/dashboard');
    }
  }, [loading, user, router]);

  // --- Muat daftar CCTV yang streaming aktif ---
  const loadCctvs = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const response = await apiRequest<PaginationPayload<CCTV>>('/cctv?page=1&limit=1000');
      const rows = response.data?.rows || [];
      // Hanya kamera dengan stream_enabled true yang layak dipilih.
      setCctvs(rows.filter((c) => c.stream_enabled === true));
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Data CCTV gagal dimuat.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timer = setTimeout(loadCctvs, 0);
    return () => clearTimeout(timer);
  }, [loadCctvs]);

  // Dengarkan perubahan fullscreen seluruh grid dari luar (mis. tekan Escape).
  useEffect(() => {
    const onFsChange = () => setGridFullscreen(Boolean(document.fullscreenElement));
    document.addEventListener('fullscreenchange', onFsChange);
    return () => document.removeEventListener('fullscreenchange', onFsChange);
  }, []);

  // --- Guard admin rendering awal (cegah render penuh sebelum role diketahui) ---
  if (loading || user?.role !== 'admin') {
    return (
      <DashboardLayout>
        <div className="flex h-[60vh] items-center justify-center gap-3 text-muted">
          <LoaderCircle className="h-6 w-6 animate-spin text-accent" />
          <span className="text-sm font-semibold">Memeriksa akses & memuat data CCTV…</span>
        </div>
      </DashboardLayout>
    );
  }

  const preset = LAYOUT_PRESETS[layout];
  const cells = Array.from({ length: preset.cells }, (_, i) => i);
  const availableCctvs = cctvs;

  // --- Toggle fullscreen seluruh grid ---
  const toggleGridFullscreen = async () => {
    const el = gridRef.current;
    if (!el) return;
    try {
      if (!document.fullscreenElement) {
        await el.requestFullscreen();
        setGridFullscreen(true);
      } else {
        await document.exitFullscreen();
        setGridFullscreen(false);
      }
    } catch {
      // Peramban menolak; abaikan.
    }
  };

  return (
    <DashboardLayout>
      <div className="space-y-6">
        {/* Header */}
        <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h1 className="flex items-center gap-2.5 text-3xl font-bold tracking-[-0.025em]">
              <MonitorPlay className="h-8 w-8 text-primary" />
              Monitor CCTV
            </h1>
            <p className="mt-1 max-w-2xl text-sm text-muted">
              Pantau beberapa kamera CCTV secara berdampingan. Pilih kamera pada tiap sel, ubah tata letak grid, atau
              buka tiap kamera di jendela terpisah.
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            {/* Toggle preset grid */}
            <div className="flex items-center gap-1 rounded-xl border border-border bg-card p-1" aria-label="Ubah tata letak grid">
              <LayoutGrid className="mx-2 h-4 w-4 text-muted" />
              {(['2x2', '3x3', '4x4'] as const).map((presetKey) => (
                <button
                  key={presetKey}
                  type="button"
                  onClick={() => setLayout(presetKey)}
                  aria-label={`Tata letak ${presetKey}`}
                  title={`Tata letak ${presetKey}`}
                  className={`h-8 rounded-lg px-3 text-xs font-bold transition-colors ${
                    layout === presetKey
                      ? 'bg-primary text-white shadow-sm'
                      : 'text-muted hover:bg-background hover:text-foreground'
                  }`}
                >
                  {presetKey}
                </button>
              ))}
            </div>

            {/* Fullscreen seluruh grid */}
            <button
              type="button"
              onClick={toggleGridFullscreen}
              className="flex items-center gap-2 rounded-xl border border-border bg-card px-3.5 py-2 text-xs font-semibold transition-colors hover:border-primary"
            >
              {gridFullscreen ? <Minimize2 className="h-4 w-4" /> : <Maximize2 className="h-4 w-4" />}
              {gridFullscreen ? 'Keluar fullscreen' : 'Fullscreen grid'}
            </button>

            {/* Info jumlah kamera streaming aktif */}
            <div className="flex items-center gap-2 rounded-xl border border-emerald-500/20 bg-emerald-500/10 px-3 py-2 text-xs font-semibold text-emerald-600">
              <span className="h-2 w-2 rounded-full bg-emerald-500" />
              {cctvs.length} kamera streaming aktif
            </div>
          </div>
        </header>

        {/* Pesan error */}
        {error && (
          <div role="alert" className="rounded-xl border border-error/25 bg-error/10 px-4 py-3 text-sm text-error">
            {error}
          </div>
        )}

        {/* Empty state: tidak ada kamera streaming */}
        {!loading && cctvs.length === 0 && (
          <div className="flex flex-col items-center justify-center gap-3 rounded-3xl border border-border bg-card py-20 text-center">
            <VideoOff className="h-12 w-12 text-muted" />
            <p className="font-semibold">Belum ada kamera dengan streaming aktif</p>
            <p className="max-w-md text-sm text-muted">
              Aktifkan opsi &quot;Streaming aktif&quot; pada data CCTV agar kamera dapat dipantau di sini.
            </p>
          </div>
        )}

        {/* Grid multi-stream */}
        <div ref={gridRef} className="rounded-3xl border border-border bg-card p-3">
          <div className={`grid gap-3 ${preset.gridClass}`}>
            {cells.map((index) => {
              const selectedId = selection[index];
              const selectedCctv = cctvs.find((c) => c.id === selectedId);

              return (
                <GridCell
                  key={index}
                  index={index}
                  cctvs={availableCctvs}
                  selectedCctv={selectedCctv}
                  onSelect={(id) =>
                    setSelection((prev) => ({ ...prev, [index]: id }))
                  }
                />
              );
            })}
          </div>
        </div>
      </div>
    </DashboardLayout>
  );
}

// =============================================================================
// GridCell — Satu sel pada grid monitor.
//
// Menampilkan dropdown pemilihan kamera; bila kamera terpilih, WHEPPlayer
// dirender mengisi seluruh sel. Menyediakan tombol fullscreen per-sel dan
// pop-out ke jendela terpisah.
// =============================================================================
interface GridCellProps {
  index: number;
  cctvs: CCTV[];
  selectedCctv?: CCTV;
  onSelect: (id: string) => void;
}

function GridCell({ index, cctvs, selectedCctv, onSelect }: GridCellProps) {
  const cellRef = useRef<HTMLDivElement>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);

  // --- Toggle fullscreen sel ini ---
  const toggleCellFullscreen = async () => {
    const el = cellRef.current;
    if (!el) return;
    try {
      if (!document.fullscreenElement) {
        await el.requestFullscreen();
        setIsFullscreen(true);
      } else {
        await document.exitFullscreen();
        setIsFullscreen(false);
      }
    } catch {
      // Peramban menolak; abaikan.
    }
  };

  // Dengarkan perubahan fullscreen (tekan Escape akan memicu ini).
  useEffect(() => {
    const onFsChange = () => setIsFullscreen(Boolean(document.fullscreenElement));
    document.addEventListener('fullscreenchange', onFsChange);
    return () => document.removeEventListener('fullscreenchange', onFsChange);
  }, []);

  // --- Pop-out ke jendela terpisah ---
  const openPopout = () => {
    if (!selectedCctv) return;
    const path = toStreamPath(selectedCctv.id);
    window.open(
      `/monitor/pop?path=${encodeURIComponent(path)}`,
      '_blank',
      'width=800,height=450'
    );
  };

  return (
    <div
      ref={cellRef}
      className="relative aspect-video overflow-hidden rounded-2xl border border-border bg-black"
    >
      {!selectedCctv ? (
        // State kosong: dropdown pemilihan kamera.
        <div className="flex h-full w-full flex-col items-center justify-center gap-3 p-3">
          <div className="flex w-full flex-col items-center gap-2 text-center">
            <MonitorPlay className="h-8 w-8 text-white/40" />
            <span className="text-xs font-semibold text-white/60">Sel {index + 1}</span>
            <select
              value=""
              onChange={(e) => onSelect(e.target.value)}
              aria-label={`Pilih kamera untuk sel ${index + 1}`}
              className="w-full max-w-[220px] rounded-lg border border-border bg-card px-3 py-2 text-xs font-medium text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/40"
            >
              <option value="">Pilih kamera…</option>
              {cctvs.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.kode_cctv} — {c.nama}
                </option>
              ))}
            </select>
          </div>
        </div>
      ) : (
        <>
          {/* Pemutar streaming kamera terpilih */}
          <WHEPPlayer
            streamPath={toStreamPath(selectedCctv.id)}
            quality="sub"
            showControls={false}
            className="h-full w-full rounded-2xl"
          />
          {/* Overlay info + aksi per-sel */}
          <div className="pointer-events-none absolute inset-x-0 top-0 flex items-start justify-between gap-2 bg-gradient-to-b from-black/70 to-transparent p-2.5">
            <span className="pointer-events-auto max-w-[70%] truncate rounded-md bg-black/60 px-2 py-1 text-[11px] font-bold text-white">
              {selectedCctv.kode_cctv} — {selectedCctv.nama}
            </span>
            <div className="pointer-events-auto flex items-center gap-1.5">
              <button
                type="button"
                onClick={openPopout}
                aria-label="Buka di jendela terpisah"
                title="Buka di jendela terpisah"
                className="flex h-8 w-8 items-center justify-center rounded-lg bg-black/60 text-white transition-colors hover:bg-black/80"
              >
                <ExternalLink className="h-4 w-4" />
              </button>
              <button
                type="button"
                onClick={toggleCellFullscreen}
                aria-label={isFullscreen ? 'Keluar fullscreen' : 'Fullscreen sel'}
                title={isFullscreen ? 'Keluar fullscreen' : 'Fullscreen sel'}
                className="flex h-8 w-8 items-center justify-center rounded-lg bg-black/60 text-white transition-colors hover:bg-black/80"
              >
                {isFullscreen ? <Minimize2 className="h-4 w-4" /> : <Maximize2 className="h-4 w-4" />}
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}

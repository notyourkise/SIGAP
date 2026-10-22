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
import { getSelectableCctvs, sortCctvsByCode, uniqueSelectionForCells } from '@/lib/monitor-cctv';
import {
  MonitorPlay,
  LayoutGrid,
  Maximize2,
  Minimize2,
  ExternalLink,
  LoaderCircle,
  VideoOff,
  Play,
  Square,
  X,
  Wifi,
  WifiOff,
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

type StreamingServiceState = 'checking' | 'ready' | 'unavailable';

// Preset ukuran grid: jumlah sel & kelas grid responsif.
const LAYOUT_PRESETS: Record<'2x2' | '3x3' | '4x4', { cells: number; gridClass: string }> = {
  '2x2': { cells: 4, gridClass: 'grid-cols-1 sm:grid-cols-2' },
  '3x3': { cells: 9, gridClass: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3' },
  '4x4': { cells: 16, gridClass: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-4' },
};

// Konversi id CCTV menjadi streamPath yang dipakai WHEPPlayer ('cam-{id}').
const toStreamPath = (id: string) => `cam-${id}`;

// =============================================================================
// Konstanta & helper sesi streaming (masa tunggu 10 menit saat berpindah menu).
// =============================================================================
// Logika:
//   - Selama user berada DI halaman Monitor, timer TIDAK berjalan (streaming bebas).
//   - Saat user keluar dari halaman Monitor (buka menu lain), waktu keluar dicatat.
//   - Jika kembali ke Monitor dalam < 10 menit, timer di-reset dan streaming lanjut.
//   - Jika sudah > 10 menit, streaming dianggap berakhir → perlu klik Play ulang.

// Kunci localStorage untuk menyimpan konfigurasi grid & waktu keluar.
const LS_MONITOR = 'sigap_monitor_state';

// Masa tunggu streaming saat berada di luar halaman Monitor (10 menit).
const STREAM_GRACE_MS = 10 * 60 * 1000;

// Struktur data monitor yang disimpan lintas halaman.
interface MonitorState {
  layout: '2x2' | '3x3' | '4x4';
  selection: Record<number, string>;
  // Waktu (ms) saat user meninggalkan halaman Monitor. Null bila sedang di halaman.
  leftAt: number | null;
  // Apakah streaming untuk sel yang dipilih sedang dinyalakan. Disimpan terpisah
  // dari leftAt agar timer "kembali dalam 10 menit" tidak pernah menyalakan
  // streaming yang belum/tidak lagi dijalankan.
  isPlaying?: boolean;
}

// Baca state monitor dari localStorage.
function loadMonitorState(): MonitorState | null {
  if (typeof window === 'undefined') return null;
  try {
    const raw = localStorage.getItem(LS_MONITOR);
    if (!raw) return null;
    return JSON.parse(raw) as MonitorState;
  } catch {
    return null;
  }
}

// Simpan state monitor ke localStorage.
function saveMonitorState(state: MonitorState | null) {
  try {
    if (state) {
      localStorage.setItem(LS_MONITOR, JSON.stringify(state));
    } else {
      localStorage.removeItem(LS_MONITOR);
    }
  } catch {
    // Abaikan bila penyimpanan diblokir.
  }
}

export default function MonitorPage() {
  const { user, loading: authLoading } = useAuth();
  const router = useRouter();

  // Data & status muat daftar CCTV.
  const [cctvs, setCctvs] = useState<CCTV[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [warning, setWarning] = useState('');
  const [streamingServiceState, setStreamingServiceState] = useState<StreamingServiceState>('checking');
  const [streamingServiceMessage, setStreamingServiceMessage] = useState('Memeriksa koneksi MediaMTX…');

  // Preset grid aktif (2×2 / 3×3 / 4×4).
  const [layout, setLayout] = useState<'2x2' | '3x3' | '4x4'>('3x3');

  // Daftar id CCTV terpilih per sel; urutan sel = index array.
  const [selection, setSelection] = useState<Record<number, string>>({});

  // Status streaming global: sedang menyala (auto-start saat kembali < 10 mnt) atau tidak.
  const [isPlaying, setIsPlaying] = useState(false);
  // Sedang memproses aksi play/stop (disable tombol).
  const [acting, setActing] = useState(false);

  // Modal konfigurasi massal (atur CCTV untuk setiap sel sekaligus).
  const [isConfigOpen, setIsConfigOpen] = useState(false);
  // Salinan sementara layout & selection saat mengedit di modal. Perubahan baru
  // berlaku setelah "Terapkan"; menutup modal tidak boleh mengubah grid.
  const [draftLayout, setDraftLayout] = useState<'2x2' | '3x3' | '4x4'>('3x3');
  const [draftSelection, setDraftSelection] = useState<Record<number, string>>({});

  // Ref kontainer untuk fullscreen seluruh grid.
  const gridRef = useRef<HTMLDivElement>(null);
  const [gridFullscreen, setGridFullscreen] = useState(false);

  // Ref nilai terbaru layout, selection, dan status playing. Dipakai listener
  // visibility/pagehide serta cleanup unmount agar tidak menyimpan state lama.
  const layoutRef = useRef(layout);
  const selectionRef = useRef(selection);
  const playingRef = useRef(isPlaying);
  useEffect(() => {
    layoutRef.current = layout;
    selectionRef.current = selection;
    playingRef.current = isPlaying;
  }, [layout, selection, isPlaying]);

  // --- Guard admin: tunggu bootstrap auth selesai sebelum memutuskan redirect ---
  useEffect(() => {
    if (!authLoading && !loading && user?.role !== 'admin') {
      router.replace('/dashboard');
    }
  }, [authLoading, loading, user, router]);

  // --- Muat daftar CCTV yang streaming aktif ---
  const loadCctvs = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const response = await apiRequest<PaginationPayload<CCTV>>('/cctv?page=1&limit=1000');
      const rows = response.data?.rows || [];
      // Hanya kamera dengan stream_enabled true yang layak dipilih. Urutkan
      // berdasarkan kode agar dropdown selalu runtut dari THD awal ke akhir.
      setCctvs(sortCctvsByCode(rows.filter((c) => c.stream_enabled === true)));
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

  const loadStreamingReadiness = useCallback(async () => {
    setStreamingServiceState('checking');
    setStreamingServiceMessage('Memeriksa koneksi MediaMTX…');
    try {
      const response = await apiRequest<{ reachable: boolean }>('/streams/readiness');
      if (response.data?.reachable) {
        setStreamingServiceState('ready');
        setStreamingServiceMessage('MediaMTX terhubung');
      } else {
        setStreamingServiceState('unavailable');
        setStreamingServiceMessage('MediaMTX belum terhubung');
        setIsPlaying(false);
      }
    } catch (e) {
      setStreamingServiceState('unavailable');
      setStreamingServiceMessage(e instanceof Error ? e.message : 'MediaMTX belum terhubung');
      setIsPlaying(false);
    }
  }, []);

  useEffect(() => {
    const timer = setTimeout(loadStreamingReadiness, 0);
    return () => clearTimeout(timer);
  }, [loadStreamingReadiness]);

  // Dengarkan perubahan fullscreen seluruh grid dari luar (mis. tekan Escape).
  useEffect(() => {
    const onFsChange = () => setGridFullscreen(Boolean(document.fullscreenElement));
    document.addEventListener('fullscreenchange', onFsChange);
    return () => document.removeEventListener('fullscreenchange', onFsChange);
  }, []);

  // --- Masa tunggu streaming (10 menit) saat user berada DI LUAR halaman Monitor ---
  // Selama user berada di halaman ini, timer TIDAK berjalan. Saat keluar (buka menu
  // lain), waktu keluar dicatat. Kembali dalam < 10 menit → timer di-reset & streaming
  // lanjut. Kembali setelah > 10 menit → streaming berakhir, perlu klik Play ulang.

  // Saat halaman dimuat: pulihkan konfigurasi & tentukan status streaming.
  // isPlaying disimpan eksplisit; leftAt saja tidak cukup karena user bisa
  // meninggalkan halaman tanpa pernah menekan Play (atau setelah menekan Stop).
  useEffect(() => {
    const state = loadMonitorState();
    if (!state) return;

    // Pulihkan konfigurasi grid. Buang pilihan duplikat/sel tersembunyi dari
    // state lama agar kamera yang sama tidak muncul di dua sel.
    const restoredLayout = state.layout in LAYOUT_PRESETS ? state.layout : '3x3';
    const restoredCells = Array.from(
      { length: LAYOUT_PRESETS[restoredLayout].cells },
      (_, index) => index
    );
    setLayout(restoredLayout);
    setSelection(uniqueSelectionForCells(state.selection ?? {}, restoredCells));
    loadCctvs();

    if (state.isPlaying !== true) {
      // State lama (tanpa flag) atau sesi yang memang sudah dihentikan tidak
      // boleh dianggap playing. Konfigurasi tetap dipertahankan.
      setIsPlaying(false);
      saveMonitorState({ ...state, leftAt: null, isPlaying: false });
      return;
    }

    if (state.leftAt != null) {
      const elapsed = Date.now() - state.leftAt;
      const stillPlaying = elapsed < STREAM_GRACE_MS;
      // Kembali dalam masa tunggu → streaming lanjut; lewat 10 menit → perlu Play ulang.
      setIsPlaying(stillPlaying);
      saveMonitorState({ ...state, leftAt: null, isPlaying: stillPlaying });
    } else {
      // Reload saat masih berada di halaman dengan streaming aktif.
      setIsPlaying(true);
      saveMonitorState({ ...state, leftAt: null, isPlaying: true });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Deteksi saat user meninggalkan / kembali ke halaman Monitor.
  useEffect(() => {
    const persistCurrent = (leftAt: number | null) => {
      saveMonitorState({
        layout: layoutRef.current,
        selection: selectionRef.current,
        leftAt,
        isPlaying: playingRef.current,
      });
    };

    const recordLeave = () => {
      // Timer 10 menit hanya berlaku bila streaming memang sedang dinyalakan.
      persistCurrent(playingRef.current ? Date.now() : null);
    };

    const handleVisibility = () => {
      if (document.hidden) {
        recordLeave();
        return;
      }

      // User kembali ke halaman → cek masa tunggu hanya bila stream memang aktif.
      const state = loadMonitorState();
      if (!state || state.isPlaying !== true) {
        setIsPlaying(false);
        return;
      }
      if (state.leftAt == null) return;

      const elapsed = Date.now() - state.leftAt;
      const stillPlaying = elapsed < STREAM_GRACE_MS;
      setIsPlaying(stillPlaying);
      saveMonitorState({ ...state, leftAt: null, isPlaying: stillPlaying });
    };

    document.addEventListener('visibilitychange', handleVisibility);
    window.addEventListener('pagehide', recordLeave);
    window.addEventListener('blur', recordLeave);
    window.addEventListener('focus', handleVisibility);

    return () => {
      document.removeEventListener('visibilitychange', handleVisibility);
      window.removeEventListener('pagehide', recordLeave);
      window.removeEventListener('blur', recordLeave);
      window.removeEventListener('focus', handleVisibility);
    };
  }, []);

  // Catat waktu keluar saat komponen di-unmount (mis. berpindah ke menu lain via
  // navigasi internal Next.js yang tidak memicu visibilitychange/pagehide).
  useEffect(() => {
    return () => {
      saveMonitorState({
        layout: layoutRef.current,
        selection: selectionRef.current,
        leftAt: playingRef.current ? Date.now() : null,
        isPlaying: playingRef.current,
      });
    };
  }, []);

  // --- Guard admin rendering awal (cegah render penuh sebelum role diketahui) ---
  if (authLoading || loading || user?.role !== 'admin') {
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
  const selectedCctvs = Object.values(selection)
    .map((id) => cctvs.find((cctv) => cctv.id === id))
    .filter((cctv): cctv is CCTV => Boolean(cctv));
  const draftCells = Array.from({ length: LAYOUT_PRESETS[draftLayout].cells }, (_, i) => i);

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

  // --- Nyalakan streaming hanya untuk kamera yang sedang tampil di grid ---
  // Sengaja tidak memakai /streams/start-all: endpoint itu menyalakan seluruh
  // CCTV sehingga tabrakan dengan halaman lain dan memboroskan bandwidth NVR.
  const handlePlay = async () => {
    if (acting) return;
    if (selectedCctvs.length === 0) {
      setError('Pilih minimal satu kamera pada grid sebelum menyalakan streaming.');
      return;
    }

    setActing(true);
    setError('');
    setWarning('');
    try {
      const results = await Promise.allSettled(
        selectedCctvs.map((cctv) => apiRequest(`/streams/${cctv.id}/start`, { method: 'POST' }))
      );
      const success = results.filter(
        (result) => result.status === 'fulfilled' && result.value.status === 'success'
      ).length;
      const failed = selectedCctvs.length - success;

      if (success > 0) {
        setIsPlaying(true);
        setStreamingServiceState('ready');
        setStreamingServiceMessage('MediaMTX terhubung');
        if (failed > 0) {
          setWarning(`${success} stream berhasil dimulai, tetapi ${failed} kamera terpilih belum dapat dihubungkan. Periksa konfigurasi RTSP CCTV tersebut.`);
        }
        // leftAt null = sedang berada di halaman (timer tidak berjalan).
        saveMonitorState({ layout, selection, leftAt: null, isPlaying: true });
      } else {
        setIsPlaying(false);
        setError('Tidak ada stream kamera terpilih yang berhasil dimulai. Periksa konfigurasi MediaMTX dan RTSP CCTV.');
        saveMonitorState({ layout, selection, leftAt: null, isPlaying: false });
        await loadStreamingReadiness();
      }
    } catch (e) {
      setIsPlaying(false);
      saveMonitorState({ layout, selection, leftAt: null, isPlaying: false });
      setError(e instanceof Error ? e.message : 'Gagal menyalakan streaming.');
      await loadStreamingReadiness();
    } finally {
      setActing(false);
    }
  };

  // --- Matikan streaming hanya untuk kamera yang sedang tampil di grid ---
  const handleStop = async () => {
    if (acting) return;
    setActing(true);
    setError('');
    setWarning('');
    try {
      const results = await Promise.allSettled(
        selectedCctvs.map((cctv) => apiRequest(`/streams/${cctv.id}/stop`, { method: 'POST' }))
      );
      const failed = results.filter(
        (result) => result.status === 'rejected' || result.value.status !== 'success'
      ).length;
      setIsPlaying(false);
      // Konfigurasi grid tetap disimpan; hanya status playing yang dimatikan.
      saveMonitorState({ layout, selection, leftAt: null, isPlaying: false });
      if (failed > 0) {
        setWarning(`${failed} stream kamera terpilih belum dapat dihentikan. Periksa koneksi MediaMTX.`);
      }
    } catch (e) {
      setIsPlaying(false);
      saveMonitorState({ layout, selection, leftAt: null, isPlaying: false });
      setError(e instanceof Error ? e.message : 'Gagal mematikan streaming.');
    } finally {
      setActing(false);
    }
  };

  // --- Buka modal konfigurasi massal (salin layout & selection saat ini ke draft) ---
  const openConfig = () => {
    setDraftLayout(layout);
    setDraftSelection(uniqueSelectionForCells(selection, cells));
    setIsConfigOpen(true);
  };

  // --- Terapkan hasil konfigurasi massal dari modal ke grid ---
  const applyConfig = async () => {
    // Hanya simpan sel yang terlihat pada layout draft, buang duplikat, lalu validasi ID.
    const nextLayout = draftLayout;
    const nextCells = Array.from({ length: LAYOUT_PRESETS[nextLayout].cells }, (_, index) => index);
    const normalized = uniqueSelectionForCells(draftSelection, nextCells);
    const validIds = new Set(cctvs.map((c) => c.id));
    const next: Record<number, string> = {};
    Object.entries(normalized).forEach(([cellIndex, cctvId]) => {
      if (cctvId && validIds.has(cctvId)) {
        next[Number(cellIndex)] = cctvId;
      }
    });

    setLayout(nextLayout);
    setSelection(next);
    setError('');
    setIsConfigOpen(false);
    saveMonitorState({ layout: nextLayout, selection: next, leftAt: null, isPlaying });

    // Bila sedang playing, daftarkan path untuk kamera yang baru dipilih agar
    // pemutar di sel baru tidak menunggu tombol Play ditekan ulang.
    if (isPlaying) {
      const previousIds = new Set(Object.values(selection));
      const newlySelected = Object.values(next)
        .filter((id) => id && !previousIds.has(id))
        .map((id) => cctvs.find((cctv) => cctv.id === id))
        .filter((cctv): cctv is CCTV => Boolean(cctv));
      if (newlySelected.length > 0) {
        const results = await Promise.allSettled(
          newlySelected.map((cctv) => apiRequest(`/streams/${cctv.id}/start`, { method: 'POST' }))
        );
        const failed = results.filter(
          (result) => result.status === 'rejected' || result.value.status !== 'success'
        ).length;
        if (failed > 0) {
          setWarning(`${failed} stream kamera baru belum dapat dihubungkan. Periksa konfigurasi RTSP CCTV tersebut.`);
        }
      }
    }
  };

  // Pilih kamera langsung dari dropdown sel kosong. Bila grid sedang playing,
  // path kamera baru langsung didaftarkan agar pemutar bisa tersambung.
  const handleCellSelect = async (index: number, id: string) => {
    const next = { ...selection, [index]: id };
    setSelection(next);
    saveMonitorState({ layout, selection: next, leftAt: null, isPlaying });

    if (isPlaying && id) {
      const response = await apiRequest(`/streams/${id}/start`, { method: 'POST' }).catch(() => null);
      if (!response || response.status !== 'success') {
        setWarning('Stream kamera yang baru dipilih belum dapat dihubungkan. Periksa konfigurasi RTSP CCTV tersebut.');
      }
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
            <button
              type="button"
              onClick={loadStreamingReadiness}
              disabled={streamingServiceState === 'checking'}
              title={streamingServiceMessage}
              className={`mt-2 inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-primary/45 disabled:cursor-wait ${
                streamingServiceState === 'ready'
                  ? 'bg-success/10 text-success'
                  : streamingServiceState === 'unavailable'
                    ? 'bg-error/10 text-error'
                    : 'bg-background text-muted'
              }`}
              aria-live="polite"
            >
              {streamingServiceState === 'checking' ? (
                <LoaderCircle className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
              ) : streamingServiceState === 'ready' ? (
                <Wifi className="h-3.5 w-3.5" aria-hidden="true" />
              ) : (
                <WifiOff className="h-3.5 w-3.5" aria-hidden="true" />
              )}
              {streamingServiceState === 'checking'
                ? 'Memeriksa MediaMTX…'
                : streamingServiceState === 'ready'
                  ? 'MediaMTX terhubung'
                  : 'MediaMTX tidak terhubung · Periksa ulang'}
            </button>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            {/* Tombol Konfigurasi (layout + CCTV per sel dalam satu modal) */}
            <button
              type="button"
              onClick={openConfig}
              className="flex items-center gap-2 rounded-xl border border-border bg-card px-3.5 py-2 text-xs font-semibold transition-colors hover:border-primary"
            >
              <LayoutGrid className="h-4 w-4" />
              Konfigurasi
            </button>

            {/* Tombol Play / Matikan streaming (hanya icon) */}
            <button
              type="button"
              onClick={handlePlay}
              disabled={acting || cctvs.length === 0 || selectedCctvs.length === 0}
              aria-label={isPlaying ? 'Putar ulang streaming' : 'Nyalakan streaming'}
              title={isPlaying ? 'Putar ulang streaming kamera terpilih' : 'Nyalakan streaming kamera terpilih'}
              className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary text-white transition-colors hover:bg-primary-hover disabled:cursor-not-allowed disabled:opacity-50"
            >
              {acting ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
            </button>
            <button
              type="button"
              onClick={handleStop}
              disabled={acting || !isPlaying}
              aria-label="Matikan streaming"
              title="Matikan streaming kamera terpilih"
              className="flex h-10 w-10 items-center justify-center rounded-xl border border-error/30 bg-error/10 text-error transition-colors hover:bg-error/20 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {acting ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Square className="h-4 w-4" />}
            </button>

            {/* Fullscreen seluruh grid */}
            <button
              type="button"
              onClick={toggleGridFullscreen}
              className="flex items-center gap-2 rounded-xl border border-border bg-card px-3.5 py-2 text-xs font-semibold transition-colors hover:border-primary"
            >
              {gridFullscreen ? <Minimize2 className="h-4 w-4" /> : <Maximize2 className="h-4 w-4" />}
              {gridFullscreen ? 'Keluar fullscreen' : 'Fullscreen grid'}
            </button>
          </div>
        </header>

        {/* Pesan error */}
        {error && (
          <div role="alert" className="flex flex-col gap-3 rounded-xl border border-error/25 bg-error/10 px-4 py-3 text-sm text-error sm:flex-row sm:items-center sm:justify-between">
            <span>{error}</span>
            {streamingServiceState === 'unavailable' && (
              <button
                type="button"
                onClick={loadStreamingReadiness}
                className="shrink-0 rounded-lg border border-error/30 px-3 py-1.5 text-xs font-bold transition-colors hover:bg-error/10 focus:outline-none focus:ring-2 focus:ring-error/30"
              >
                Periksa ulang
              </button>
            )}
          </div>
        )}

        {warning && (
          <div role="status" className="rounded-xl border border-warning/30 bg-warning/10 px-4 py-3 text-sm text-warning">
            {warning}
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
                  cctvs={getSelectableCctvs(availableCctvs, selection, index, cells)}
                  selectedCctv={selectedCctv}
                  isPlaying={isPlaying}
                  onSelect={(id) => handleCellSelect(index, id)}
                />
              );
            })}
          </div>
        </div>
      </div>

      {/* Modal konfigurasi massal CCTV per sel */}
      {isConfigOpen && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
          onClick={() => setIsConfigOpen(false)}
          role="dialog"
          aria-modal="true"
          aria-label="Konfigurasi CCTV per sel"
        >
          <div
            className="flex max-h-[90vh] w-full max-w-lg flex-col overflow-hidden rounded-2xl border border-border bg-card shadow-2xl"
            onClick={(e) => e.stopPropagation()}
          >
            {/* Header modal */}
            <div className="flex items-center justify-between border-b border-border px-5 py-4">
              <div>
                <h3 className="text-lg font-bold tracking-[-0.015em]">Konfigurasi CCTV</h3>
                <p className="mt-0.5 text-xs text-muted">
                  Atur kamera untuk setiap sel sekaligus. Tata letak: {draftLayout}
                </p>
              </div>
              <button
                type="button"
                onClick={() => setIsConfigOpen(false)}
                aria-label="Tutup konfigurasi"
                className="rounded-lg p-2 text-muted transition-colors hover:bg-background hover:text-foreground"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            {/* Body: pilihan layout + daftar sel */}
            <div className="flex-1 space-y-4 overflow-y-auto p-5">
              {/* Pilihan tata letak grid */}
              <div>
                <p className="mb-1.5 text-xs font-semibold text-muted">Tata letak grid</p>
                <div className="flex items-center gap-1 rounded-xl border border-border bg-background p-1">
                  <LayoutGrid className="mx-2 h-4 w-4 text-muted" />
                  {(['2x2', '3x3', '4x4'] as const).map((presetKey) => (
                    <button
                      key={presetKey}
                      type="button"
                      onClick={() => setDraftLayout(presetKey)}
                      aria-label={`Tata letak ${presetKey}`}
                      title={`Tata letak ${presetKey}`}
                      className={`h-8 flex-1 rounded-lg px-3 text-xs font-bold transition-colors ${
                        draftLayout === presetKey
                          ? 'bg-primary text-white shadow-sm'
                          : 'text-muted hover:bg-background hover:text-foreground'
                      }`}
                    >
                      {presetKey}
                    </button>
                  ))}
                </div>
              </div>

              {/* Daftar sel & pemilihan kamera */}
              <div className="space-y-3">
                {draftCells.map((index) => {
                  const selectableCctvs = getSelectableCctvs(cctvs, draftSelection, index, draftCells);
                  return (
                    <div key={index} className="flex items-center gap-3">
                      <span className="w-14 shrink-0 text-sm font-semibold text-muted">Sel {index + 1}</span>
                      <select
                        value={draftSelection[index] ?? ''}
                        onChange={(e) =>
                          setDraftSelection((prev) => ({ ...prev, [index]: e.target.value }))
                        }
                        aria-label={`Pilih kamera untuk sel ${index + 1}`}
                        className="w-full rounded-lg border border-input-border bg-input px-3 py-2 text-sm font-medium text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/40"
                      >
                        <option value="">— Kosong —</option>
                        {selectableCctvs.map((c) => (
                          <option key={c.id} value={c.id}>
                            {c.kode_cctv} — {c.nama}
                          </option>
                        ))}
                      </select>
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Footer modal */}
            <div className="flex items-center justify-between gap-3 border-t border-border px-5 py-4">
              <button
                type="button"
                onClick={() => setDraftSelection({})}
                className="rounded-xl border border-border bg-background px-4 py-2 text-sm font-semibold text-muted transition-colors hover:text-foreground"
              >
                Kosongkan
              </button>
              <button
                type="button"
                onClick={applyConfig}
                className="rounded-xl bg-primary px-5 py-2 text-sm font-semibold text-white transition-colors hover:bg-primary-hover"
              >
                Terapkan
              </button>
            </div>
          </div>
        </div>
      )}
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
  isPlaying: boolean;
  onSelect: (id: string) => void;
}

function GridCell({ index, cctvs, selectedCctv, isPlaying, onSelect }: GridCellProps) {
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
      ) : isPlaying ? (
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
      ) : (
        // CCTV terpilih tetapi streaming belum dinyalakan (menunggu tombol Play).
        <div className="flex h-full w-full flex-col items-center justify-center gap-3 p-3">
          <div className="flex flex-col items-center gap-2 text-center">
            <VideoOff className="h-8 w-8 text-white/40" />
            <span className="text-xs font-semibold text-white/70">
              {selectedCctv.kode_cctv} — {selectedCctv.nama}
            </span>
            <span className="text-[11px] text-white/50">Streaming belum dinyalakan. Klik Play untuk mulai.</span>
          </div>
        </div>
      )}
    </div>
  );
}

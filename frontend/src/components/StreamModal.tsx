'use client';

// =============================================================================
// StreamModal — Modal untuk menonton & mengelola streaming CCTV via WHEP.
//
// Fitur:
//   - Lazy-load WHEPPlayer (dynamic import, ssr:false) agar aman di SSR.
//   - Mulai/hentikan stream melalui API backend (mengembalikan whep_url).
//   - Ganti kualitas SD ↔ HD (parent me-restart stream dengan kualitas baru).
//   - Info CCTV (lokasi, IP, channel) serta tautan ke Master CCTV.
//
// Endpoint stream diproksi Nginx di /stream/* (same-origin), sehingga fetch di
// dalam WHEPPlayer berjalan bersama cookie sesi otomatis.
// =============================================================================

import { useCallback, useEffect, useState } from 'react';
import dynamic from 'next/dynamic';
import {
  X,
  MapPin,
  Network,
  Play,
  Square,
  Maximize2,
  Video,
  LoaderCircle,
} from 'lucide-react';
import { apiRequest } from '@/lib/api';
import { useToast } from '@/components/Toast';
import type { ConnectionStatus } from '@/components/WHEPPlayer';

export interface StreamModalCctv {
  id: string;
  nama: string;
  kode_cctv: string;
  status: string;
  health_score?: number;
  lokasi_area?: string;
  ip_camera?: string;
  stream_enabled?: boolean;
  channel_port?: number;
}

export interface StreamModalProps {
  cctv: StreamModalCctv;
  onClose: () => void;
}

// Lazy-load WHEPPlayer hanya di sisi klien (SSR-safe). dynamic() harus berada
// di top-level file; wrapper kecil ini meneruskan props ke komponen aslinya.
const LazyWHEPPlayer = dynamic(
  () => import('@/components/WHEPPlayer').then((m) => m.default),
  { ssr: false }
);

type Quality = 'main' | 'sub';

// Ekstrak path stream dari URL WHEP, mis. "/stream/cam-xxx/whep" -> "cam-xxx".
function extractStreamPath(whepUrl: string): string {
  const match =
    whepUrl.match(/\/stream\/([^/]+)\/whep$/i) ||
    whepUrl.match(/\/stream\/([^/]+)$/i);
  if (match) return match[1];
  // Fallback: buang prefix "/stream/" dan suffix "/whep" bila ada.
  return whepUrl.replace(/^\/stream\//, '').replace(/\/whep$/i, '');
}

// Normalisasi status CCTV ke label baku untuk badge.
function statusBadge(status: string): {
  label: string;
  className: string;
} {
  const s = status.trim().toUpperCase();
  if (['ACTIVE', 'AKTIF', 'LIVE', 'ONLINE'].includes(s)) {
    return { label: 'Aktif', className: 'border-success/25 bg-success/10 text-success' };
  }
  if (['MAINTENANCE', 'PEMELIHARAAN'].includes(s)) {
    return { label: 'Pemeliharaan', className: 'border-amber-500/25 bg-amber-500/10 text-amber-600 dark:text-amber-400' };
  }
  return { label: 'Nonaktif', className: 'border-error/25 bg-error/10 text-error' };
}

export default function StreamModal({ cctv, onClose }: StreamModalProps) {
  const toast = useToast();
  const [streamPath, setStreamPath] = useState<string | null>(null);
  const [quality, setQuality] = useState<Quality>('sub');
  const [starting, setStarting] = useState(false);
  const [stopping, setStopping] = useState(false);
  const [live, setLive] = useState(false);

  const isStreaming = Boolean(streamPath);
  const badge = statusBadge(cctv.status);

  // Tutup modal saat menekan tombol Escape.
  useEffect(() => {
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', closeOnEscape);
    return () => document.removeEventListener('keydown', closeOnEscape);
  }, [onClose]);

  // Mulai stream. qualityOverride memungkinkan restart dengan kualitas baru
  // tanpa mengubah state sebelum panggilan API selesai.
  const handleStart = useCallback(
    async (qualityOverride?: Quality) => {
      const q = qualityOverride ?? quality;
      setStarting(true);
      try {
        const response = await apiRequest<{ whep_url: string }>(
          `/streams/${cctv.id}/start?quality=${q}`,
          { method: 'POST' }
        );
        if (response.status === 'success' && response.data?.whep_url) {
          const path = extractStreamPath(response.data.whep_url);
          setStreamPath(path);
          toast.success(`Streaming "${cctv.nama}" dimulai.`);
        } else {
          throw new Error(response.message || 'Stream tidak dapat dimulai.');
        }
      } catch (err) {
        toast.error(
          err instanceof Error ? err.message : 'Stream tidak dapat dimulai.'
        );
      } finally {
        setStarting(false);
      }
    },
    [cctv.id, cctv.nama, quality, toast]
  );

  // Hentikan stream dan kosongkan path.
  const handleStop = useCallback(async () => {
    setStopping(true);
    try {
      await apiRequest(`/streams/${cctv.id}/stop`, { method: 'POST' });
      setStreamPath(null);
      setLive(false);
      toast.info(`Streaming "${cctv.nama}" dihentikan.`);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : 'Stream tidak dapat dihentikan.'
      );
    } finally {
      setStopping(false);
    }
  }, [cctv.id, cctv.nama, toast]);

  // Ditujukan dari tombol "Mulai Streaming" pada placeholder body.
  const handleStartFromPlaceholder = () => handleStart();

  // Ditujukan dari WHEPPlayer saat pengguna mengganti kualitas: set state
  // lalu restart stream dengan kualitas baru bila sedang streaming.
  const handleQualityChange = useCallback(
    (q: Quality) => {
      setQuality(q);
      if (streamPath) {
        handleStart(q);
      }
    },
    [streamPath, handleStart]
  );

  // Laporkan error dari WHEPPlayer ke toast.
  const handlePlayerError = useCallback(
    (message: string) => {
      toast.error(message);
    },
    [toast]
  );

  // Pantau status koneksi dari WHEPPlayer untuk menandai status 'live'.
  const handleStatusChange = useCallback((status: ConnectionStatus) => {
    setLive(status === 'live');
  }, []);

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fade-in"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-label={`Streaming CCTV ${cctv.nama}`}
    >
      <div
        className="relative flex max-h-[92vh] w-full max-w-3xl flex-col overflow-hidden rounded-2xl border border-border bg-card shadow-2xl animate-pop-in"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-start justify-between gap-3 border-b border-border px-5 py-4">
          <div className="min-w-0">
            <h3 className="truncate text-base font-bold">{cctv.nama}</h3>
            <p className="mt-0.5 flex flex-wrap items-center gap-2 text-xs text-muted">
              <span className="font-mono">{cctv.kode_cctv}</span>
              <span
                className={`inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[10px] font-bold ${badge.className}`}
              >
                {badge.label}
              </span>
              {typeof cctv.health_score === 'number' && (
                <span className="inline-flex items-center gap-1 rounded-full border border-border bg-background px-2 py-0.5 text-[10px] font-semibold text-muted">
                  Skor kesehatan: {cctv.health_score}
                </span>
              )}
            </p>
          </div>
          <button
            type="button"
            aria-label="Tutup modal streaming"
            onClick={onClose}
            disabled={starting || stopping}
            className="rounded-lg border border-border p-1 text-muted transition-colors hover:bg-background hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto p-5">
          {isStreaming ? (
            <>
              <LazyWHEPPlayer
                streamPath={streamPath!}
                quality={quality}
                onStatusChange={handleStatusChange}
                onError={handlePlayerError}
                onQualityChange={handleQualityChange}
              />
              <p className="mt-2 text-center text-[11px] text-muted">
                Kualitas{' '}
                <span className="font-bold text-foreground">
                  {quality === 'main' ? 'HD' : 'SD'}
                </span>
                {live ? ' · Streaming aktif' : ' · Menghubungkan…'}
              </p>
            </>
          ) : (
            <div className="flex aspect-video w-full flex-col items-center justify-center gap-4 rounded-2xl border border-dashed border-border bg-background">
              <div className="flex h-16 w-16 items-center justify-center rounded-full bg-primary/10">
                <Video className="h-8 w-8 text-primary" />
              </div>
              <div className="text-center">
                <p className="text-sm font-bold">Belum ada stream aktif</p>
                <p className="mt-1 max-w-xs text-xs text-muted">
                  Mulai streaming untuk menonton kamera ini secara langsung
                  melalui koneksi WebRTC.
                </p>
              </div>
              <button
                type="button"
                onClick={handleStartFromPlaceholder}
                disabled={starting}
                className="inline-flex items-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-primary-hover disabled:cursor-wait disabled:opacity-60"
              >
                {starting ? (
                  <LoaderCircle className="h-4 w-4 animate-spin" />
                ) : (
                  <Play className="h-4 w-4" />
                )}
                {starting ? 'Memulai…' : 'Mulai Streaming'}
              </button>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="space-y-3 border-t border-border px-5 py-4">
          <div className="grid grid-cols-1 gap-2 text-xs text-muted sm:grid-cols-3">
            <span className="flex min-w-0 items-center gap-1.5">
              <MapPin className="h-3.5 w-3.5 shrink-0 text-primary" />
              <span className="truncate" title={cctv.lokasi_area || undefined}>
                {cctv.lokasi_area || 'Lokasi belum dicatat'}
              </span>
            </span>
            <span className="flex min-w-0 items-center gap-1.5">
              <Network className="h-3.5 w-3.5 shrink-0 text-primary" />
              <span className="truncate font-mono">
                {cctv.ip_camera || 'IP belum tercatat'}
              </span>
            </span>
            <span className="flex items-center gap-1.5">
              <Maximize2 className="h-3.5 w-3.5 shrink-0 text-primary" />
              <span>
                Channel {cctv.channel_port ?? '-'}
                {typeof cctv.stream_enabled === 'boolean' &&
                  ` · Stream ${cctv.stream_enabled ? 'aktif' : 'nonaktif'}`}
              </span>
            </span>
          </div>

          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              {isStreaming ? (
                <button
                  type="button"
                  onClick={handleStop}
                  disabled={stopping}
                  className="inline-flex items-center gap-2 rounded-xl border border-error/30 bg-error/10 px-4 py-2 text-sm font-semibold text-error transition-colors hover:bg-error/20 disabled:cursor-wait disabled:opacity-60"
                >
                  {stopping ? (
                    <LoaderCircle className="h-4 w-4 animate-spin" />
                  ) : (
                    <Square className="h-4 w-4" />
                  )}
                  {stopping ? 'Menghentikan…' : 'Hentikan Stream'}
                </button>
              ) : (
                <button
                  type="button"
                  onClick={handleStartFromPlaceholder}
                  disabled={starting}
                  className="inline-flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-primary-hover disabled:cursor-wait disabled:opacity-60"
                >
                  {starting ? (
                    <LoaderCircle className="h-4 w-4 animate-spin" />
                  ) : (
                    <Video className="h-4 w-4" />
                  )}
                  {starting ? 'Memulai…' : 'Mulai Stream'}
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

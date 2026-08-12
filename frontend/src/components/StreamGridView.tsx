'use client';

import { useEffect, useRef, useState } from 'react';
import dynamic from 'next/dynamic';
import { Video, VideoOff, MonitorPlay, LoaderCircle, MapPin, Pause, Play, ChevronLeft, ChevronRight } from 'lucide-react';
import type { CCTVMapItem } from '@/components/CCTVMap';

// Lazy-load WHEPPlayer hanya ketika dibutuhkan (dan hanya di sisi klien karena WHEP butuh browser API).
const WHEPPlayer = dynamic(() => import('@/components/WHEPPlayer'), {
  ssr: false,
  loading: () => (
    <div className="flex h-full w-full items-center justify-center bg-black/90">
      <LoaderCircle className="h-6 w-6 animate-spin text-white/70" />
    </div>
  ),
});

export interface StreamGridViewProps {
  cctvs: CCTVMapItem[]; // daftar CCTV (diharapkan sudah difilter hanya yang streaming aktif)
  onCctvClick?: (cctv: CCTVMapItem) => void;
}

// Komponen sel individual: memuat WHEPPlayer hanya saat sel berada di dekat viewport (IntersectionObserver).
function StreamCell({
  cctv,
  onCctvClick,
}: {
  cctv: CCTVMapItem;
  onCctvClick?: (cctv: CCTVMapItem) => void;
}) {
  const cellRef = useRef<HTMLDivElement>(null);
  const [isNear, setIsNear] = useState(false);
  const [paused, setPaused] = useState(false);

  useEffect(() => {
    const el = cellRef.current;
    if (!el) return;

    // IntersectionObserver untuk lazy loading: hanya render pemutar saat sel mendekati/masuk viewport.
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            // Root margin default; kita trigger saat mulai terlihat.
            setIsNear(true);
            observer.disconnect();
            break;
          }
        }
      },
      { rootMargin: '400px' }, // muat agak lebih awal agar tidak kedap-kedip saat scroll.
    );

    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  const statusBadge = cctv.stream_enabled ? (
    <span className="inline-flex items-center gap-1 rounded-full border border-emerald-500/20 bg-emerald-500/10 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-emerald-600">
      <MonitorPlay className="h-3 w-3" />
      Streaming
    </span>
  ) : (
    <span className="inline-flex items-center gap-1 rounded-full border border-muted/30 bg-muted/10 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-muted">
      <VideoOff className="h-3 w-3" />
      Nonaktif
    </span>
  );

  return (
    <div
      ref={cellRef}
      role="button"
      tabIndex={0}
      onClick={() => onCctvClick?.(cctv)}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          onCctvClick?.(cctv);
        }
      }}
      className="group flex flex-col overflow-hidden rounded-2xl border border-border bg-card text-left shadow-sm transition-shadow hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
    >
      {/* Area video / placeholder (tinggi ~200px) */}
      <div className="relative h-[200px] w-full overflow-hidden bg-black">
        {cctv.stream_enabled ? (
          isNear && !paused ? (
            // Pemutar mini: kualitas sub untuk menghemat bandwidth.
            <WHEPPlayer streamPath={`cam-${cctv.id}`} quality="sub" />
          ) : isNear && paused ? (
            // Placeholder saat stream dijeda (WHEPPlayer tidak dimount sehingga koneksi tertutup).
            <div className="flex h-full w-full flex-col items-center justify-center gap-2 bg-black/90 text-white/70">
              <Pause className="h-8 w-8" />
              <span className="text-xs font-semibold">Dijeda</span>
            </div>
          ) : (
            // Placeholder/skeleton sebelum sel masuk viewport.
            <div className="flex h-full w-full items-center justify-center bg-black/90">
              <LoaderCircle className="h-6 w-6 animate-spin text-white/50" />
            </div>
          )
        ) : (
          // Placeholder untuk CCTV yang tidak streaming.
          <div className="flex h-full w-full flex-col items-center justify-center gap-2 bg-muted/10 text-muted">
            <VideoOff className="h-8 w-8" />
            <span className="text-xs font-semibold">Streaming tidak aktif</span>
          </div>
        )}

        {/* Badge status di pojok atas video */}
        <div className="absolute left-2 top-2">{statusBadge}</div>

        {/* Tombol pause/resume hanya untuk CCTV yang streaming aktif */}
        {cctv.stream_enabled && (
          <button
            type="button"
            aria-label={paused ? 'Lanjutkan stream' : 'Jeda stream'}
            onClick={(e) => {
              // Cegah event klik naik ke sel (menghindari memicu onCctvClick).
              e.stopPropagation();
              setPaused((p) => !p);
            }}
            className="absolute right-2 top-2 inline-flex h-8 w-8 items-center justify-center rounded-full bg-black/60 text-white/90 backdrop-blur transition-colors hover:bg-black/80 hover:text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
          >
            {paused ? <Play className="h-4 w-4" /> : <Pause className="h-4 w-4" />}
          </button>
        )}
      </div>

      {/* Info CCTV */}
      <div className="flex flex-1 flex-col gap-1 p-3">
        <div className="flex items-start justify-between gap-2">
          <h3 className="line-clamp-1 text-sm font-bold text-foreground">{cctv.nama}</h3>
          <Video className="h-4 w-4 shrink-0 text-primary" />
        </div>
        <p className="font-mono text-[11px] font-semibold text-muted">{cctv.kode_cctv}</p>
        <div className="mt-1 flex items-center gap-1.5 text-[11px] text-muted">
          <MapPin className="h-3.5 w-3.5 shrink-0 text-primary" />
          <span className="line-clamp-1">{cctv.lokasi_area || 'Kawasan KIPP IKN'}</span>
        </div>
      </div>
    </div>
  );
}

export default function StreamGridView({ cctvs, onCctvClick }: StreamGridViewProps) {
  // Opsi ukuran halaman: "Semua" diwakili dengan nilai besar agar semua kamera dirender sekaligus.
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(12);

  const total = cctvs.length;
  const effectivePageSize = pageSize >= total ? total : pageSize;
  const totalPages = Math.max(1, Math.ceil(total / effectivePageSize));
  const currentPage = Math.min(page, totalPages);

  const start = total === 0 ? 0 : (currentPage - 1) * effectivePageSize + 1;
  const end = Math.min(currentPage * effectivePageSize, total);
  const pagedCctvs = cctvs.slice((currentPage - 1) * effectivePageSize, currentPage * effectivePageSize);

  // Ganti ukuran halaman → reset ke halaman pertama.
  const handlePageSizeChange = (value: number) => {
    setPageSize(value);
    setPage(1);
  };

  if (!cctvs.length) {
    return (
      <div className="flex min-h-[300px] flex-col items-center justify-center gap-3 rounded-3xl border border-dashed border-border bg-card text-muted">
        <MonitorPlay className="h-10 w-10 opacity-50" />
        <p className="text-sm font-semibold">Belum ada CCTV dengan streaming aktif.</p>
        <p className="text-xs">Aktifkan streaming pada CCTV di Master CCTV untuk melihatnya di sini.</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      {/* Header: counter + kontrol pagination dan ukuran halaman */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-muted">
          Menampilkan <span className="font-semibold text-foreground">{start}</span>–
          <span className="font-semibold text-foreground">{end}</span> dari{' '}
          <span className="font-semibold text-foreground">{total}</span> kamera aktif
        </p>

        <div className="flex flex-wrap items-center gap-2">
          {/* Dropdown ukuran halaman */}
          <select
            aria-label="Ukuran halaman"
            value={pageSize >= total ? total : pageSize}
            onChange={(e) => handlePageSizeChange(Number(e.target.value))}
            className="h-8 rounded-lg border border-border bg-card px-2 text-xs font-semibold text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
          >
            <option value={4}>4</option>
            <option value={8}>8</option>
            <option value={12}>12</option>
            <option value={total}>Semua</option>
          </select>

          {/* Kontrol prev/next halaman */}
          <div className="flex items-center gap-1">
            <button
              type="button"
              aria-label="Halaman sebelumnya"
              disabled={currentPage <= 1}
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              className="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-border bg-card text-foreground transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-40"
            >
              <ChevronLeft className="h-4 w-4" />
            </button>
            <span className="min-w-[56px] text-center text-xs font-semibold text-muted">
              {currentPage} / {totalPages}
            </span>
            <button
              type="button"
              aria-label="Halaman berikutnya"
              disabled={currentPage >= totalPages}
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              className="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-border bg-card text-foreground transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-40"
            >
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      </div>

      {/* Grid halaman saat ini */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {pagedCctvs.map((cctv) => (
          <StreamCell key={cctv.id} cctv={cctv} onCctvClick={onCctvClick} />
        ))}
      </div>
    </div>
  );
}

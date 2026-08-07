'use client';

import { useEffect, useRef, useState } from 'react';
import dynamic from 'next/dynamic';
import { Video, VideoOff, MonitorPlay, LoaderCircle, MapPin } from 'lucide-react';
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
          isNear ? (
            // Pemutar mini: kualitas sub untuk menghemat bandwidth.
            <WHEPPlayer streamPath={`cam-${cctv.id}`} quality="sub" />
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
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      {cctvs.map((cctv) => (
        <StreamCell key={cctv.id} cctv={cctv} onCctvClick={onCctvClick} />
      ))}
    </div>
  );
}

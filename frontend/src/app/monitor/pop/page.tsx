'use client';

// =============================================================================
// Monitor Pop — Tampilan fullscreen satu kamera (pop-out window).
//
// Halaman ini dibuka dari Monitor CCTV melalui tombol pop-out. Nilai streamPath
// dikirim lewat query parameter `?path=cam-{id}`. Halaman merender WHEPPlayer
// dengan latar gelap penuh agar tampak seperti layar monitor CCTV.
//
// Catatan SSR: `useSearchParams` dari next/navigation harus dibungkus dalam
// <Suspense> pada App Router agar halaman tetap dapat dipre-render tanpa error.
// =============================================================================

import { Suspense } from 'react';
import dynamic from 'next/dynamic';
import { useSearchParams } from 'next/navigation';
import { VideoOff, LoaderCircle } from 'lucide-react';

// WHEPPlayer dimuat dinamis (lazy) hanya di sisi klien agar aman untuk SSR.
const WHEPPlayer = dynamic(() => import('@/components/WHEPPlayer'), {
  ssr: false,
  loading: () => (
    <div className="flex h-screen w-screen items-center justify-center gap-3 bg-black text-white">
      <LoaderCircle className="h-6 w-6 animate-spin text-accent" />
      <span className="text-sm font-semibold">Menyiapkan pemutar…</span>
    </div>
  ),
});

// Komponen inti yang membaca query parameter `path`.
function PopViewer() {
  const searchParams = useSearchParams();
  const path = searchParams.get('path');

  // Tanpa path yang valid, tampilkan pesan bimbingan.
  if (!path) {
    return (
      <div className="flex h-screen w-screen flex-col items-center justify-center gap-3 bg-black px-6 text-center">
        <VideoOff className="h-12 w-12 text-white/40" />
        <p className="text-base font-semibold text-white">Tidak ada stream yang dipilih</p>
        <p className="max-w-md text-sm text-white/60">
          Tutup jendela ini lalu gunakan tombol pop-out pada halaman Monitor CCTV untuk membuka kamera tertentu.
        </p>
      </div>
    );
  }

  return (
    <div className="relative h-screen w-screen bg-black">
      <WHEPPlayer
        streamPath={path}
        quality="sub"
        showControls
        className="h-full w-full rounded-none"
      />
    </div>
  );
}

// Halaman utama membungkus PopViewer dengan Suspense untuk memenuhi persyaratan
// prerender useSearchParams pada App Router.
export default function MonitorPopPage() {
  return (
    <Suspense
      fallback={
        <div className="flex h-screen w-screen items-center justify-center gap-3 bg-black text-white">
          <LoaderCircle className="h-6 w-6 animate-spin text-accent" />
          <span className="text-sm font-semibold">Menyiapkan pemutar…</span>
        </div>
      }
    >
      <PopViewer />
    </Suspense>
  );
}

import Link from 'next/link';
import { ArrowLeft, Home, MapPinOff } from 'lucide-react';
import Lottie404 from '@/components/Lottie404';

// Halaman 404 kustom SIGAP — dirender untuk semua URL yang tidak cocok.
// Berada di dalam root layout sehingga otomatis mengikuti tema terang/gelap.
export default function NotFound() {
  return (
    <main className="relative flex min-h-screen flex-col items-center justify-center overflow-hidden bg-background px-4 py-12">
      <div className="absolute inset-x-0 top-0 h-1.5 bg-gradient-to-r from-accent via-accent-soft to-primary" />
      <div className="absolute left-0 top-0 h-72 w-72 -translate-x-1/2 -translate-y-1/2 rounded-full bg-accent/10 blur-3xl" />
      <div className="absolute bottom-0 right-0 h-80 w-80 translate-x-1/2 translate-y-1/2 rounded-full bg-primary/10 blur-3xl" />

      <section className="relative z-10 flex w-full max-w-3xl flex-col items-center text-center">
        <Lottie404 />

        <h1 className="mt-6 text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
          Halaman tidak ditemukan
        </h1>

        <p className="mt-3 max-w-md text-sm leading-relaxed text-muted sm:text-base">
          URL yang Anda tuju tidak tersedia atau telah dipindahkan. Periksa kembali
          alamatnya, atau kembali ke halaman utama untuk melanjutkan operasional.
        </p>

        <div className="mt-8 flex flex-col items-center gap-3 sm:flex-row">
          <Link
            href="/dashboard"
            className="inline-flex items-center gap-2 rounded-xl bg-primary px-5 py-3 text-sm font-semibold text-white shadow-md shadow-primary/20 transition-colors hover:bg-primary-hover"
          >
            <Home className="h-4 w-4" />
            Kembali ke Dashboard
          </Link>
          <Link
            href="/login"
            className="inline-flex items-center gap-2 rounded-xl border border-border bg-card px-5 py-3 text-sm font-semibold text-foreground transition-colors hover:border-primary/40 hover:bg-background"
          >
            <ArrowLeft className="h-4 w-4" />
            Ke Halaman Login
          </Link>
        </div>

        <p className="mt-10 inline-flex items-center gap-2 font-mono text-xs uppercase tracking-widest text-muted">
          <MapPinOff className="h-3.5 w-3.5" />
          SIGAP · Sistem Identifikasi Gangguan Aset Pusat
        </p>
      </section>
    </main>
  );
}

'use client';

import { useEffect, useState } from 'react';
import Lottie from 'lottie-react';

// Animasi T-rex "PAGE NOT FOUND" (LottieFiles, 1080x1080) yang disajikan
// sebagai aset statis publik. Dimuat saat runtime via fetch agar tidak
// di-bundle ganda oleh Next.js.
const ANIMATION_URL = '/assets/animasi/404%20Error.json';

export default function Lottie404() {
  const [data, setData] = useState<object | null>(null);

  useEffect(() => {
    let cancelled = false;
    fetch(ANIMATION_URL)
      .then((res) => res.json())
      .then((json) => {
        if (!cancelled) setData(json);
      })
      .catch(() => {
        // Gagal memuat animasi — halaman tetap tampil dengan placeholder.
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const containerStyle: React.CSSProperties = {
    width: 'min(78vw, 560px)',
    maxHeight: 'min(62vh, 560px)',
  };

  if (!data) {
    return (
      <div
        aria-hidden="true"
        className="mx-auto flex items-center justify-center"
        style={{ ...containerStyle, aspectRatio: '1 / 1' }}
      >
        <div className="h-12 w-12 animate-pulse rounded-full bg-neutral-200" />
      </div>
    );
  }

  return (
    <div className="mx-auto" style={containerStyle} role="img" aria-label="Ilustrasi halaman tidak ditemukan">
      <Lottie animationData={data} loop autoplay className="h-auto w-full" />
    </div>
  );
}

import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // No explicit output - rely on ISR/server-side rendering for app router
  // Izinkan akses development melalui alamat loopback eksplisit. Next.js
  // menganggap 127.0.0.1 berbeda origin dari hostname default localhost dan
  // memblokir aset dev/HMR bila origin ini tidak didaftarkan.
  allowedDevOrigins: ['127.0.0.1'],
  
  async rewrites() {
    // Di dalam Docker container, service backend bernama 'api:3000'.
    // Pada environment standalone local (npm run dev), fallback ke 'localhost:3000'.
    const backendUrl = process.env.INTERNAL_BACKEND_URL || 'http://127.0.0.1:3000';
    // Nginx menangani /stream pada deployment. Rewrite ini membuat mode lokal
    // tanpa Nginx tetap dapat meneruskan WHEP ke MediaMTX native di port 8889.
    const mediamtxWebrtcUrl = process.env.INTERNAL_MEDIAMTX_WEBRTC_URL || 'http://127.0.0.1:8889';
    return [
      {
        source: '/api/:path*',
        destination: `${backendUrl}/api/:path*`,
      },
      {
        source: '/stream/:path*',
        destination: `${mediamtxWebrtcUrl}/:path*`,
      },
    ];
  },
  async headers() {
    // PWA (Task 8.1): pastikan service worker & manifest tidak di-cache agresif
    // sehingga pembaruan SW/manifest cepat tersebar ke browser.
    return [
      {
        source: '/sw.js',
        headers: [
          { key: 'Cache-Control', value: 'no-cache, no-store, must-revalidate' },
          { key: 'Service-Worker-Allowed', value: '/' },
        ],
      },
      {
        source: '/manifest.json',
        headers: [{ key: 'Cache-Control', value: 'public, max-age=3600' }],
      },
    ];
  },
};

export default nextConfig;

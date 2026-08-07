import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async rewrites() {
    // Di dalam Docker container, service backend bernama 'api:3000'.
    // Pada environment standalone local (npm run dev), fallback ke 'localhost:3000'.
    const backendUrl = process.env.INTERNAL_BACKEND_URL || 'http://api:3000';
    return [
      {
        source: '/api/:path*',
        destination: `${backendUrl}/api/:path*`,
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

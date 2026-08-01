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
};

export default nextConfig;

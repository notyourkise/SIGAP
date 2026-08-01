'use client';

import React, { useState, useEffect } from 'react';
import { API_BASE_URL } from '@/lib/api';
import { ImageIcon } from 'lucide-react';

interface AuthenticatedImageProps {
  /** Path gambar dari server, contoh: "/api/uploads/uuid.jpg" */
  imageUrl: string;
  alt: string;
  className?: string;
}

/**
 * AuthenticatedImage — komponen gambar yang mengambil file dari server.
 *
 * SECURITY: File uploads tidak lagi dapat diakses secara publik.
 * Endpoint GET /api/uploads/:filename memerlukan JWT yang valid.
 * Autentikasi dilakukan lewat cookie httpOnly sigap_session yang dikirim
 * otomatis oleh browser (credentials: 'include'), lalu komponen ini membuat
 * object URL dari blob yang diterima untuk ditampilkan.
 */
export default function AuthenticatedImage({ imageUrl, alt, className }: AuthenticatedImageProps) {
  const [blobUrl, setBlobUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  useEffect(() => {
    let objectUrl: string | null = null;

    const fetchImage = async () => {
      try {
        setLoading(true);
        setError(false);

        // Normalisasi URL untuk mendukung format lama (/uploads/...) dan baru (/api/uploads/...)
        // Format lama: "/uploads/uuid.jpg" → dikonversi ke "/api/uploads/uuid.jpg"
        // Format baru: "/api/uploads/uuid.jpg" → digunakan langsung
        let normalizedPath = imageUrl;
        if (imageUrl.startsWith('/uploads/')) {
          normalizedPath = '/api' + imageUrl;
        }

        // Bentuk URL lengkap dari API base URL
        const baseUrl = API_BASE_URL.replace(/\/api\/?$/, '');
        const fullUrl = normalizedPath.startsWith('http')
          ? normalizedPath
          : `${baseUrl}${normalizedPath}`;

        const response = await fetch(fullUrl, {
          credentials: 'include',
        });

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`);
        }

        const blob = await response.blob();
        objectUrl = URL.createObjectURL(blob);
        setBlobUrl(objectUrl);
      } catch {
        setError(true);
      } finally {
        setLoading(false);
      }
    };

    if (imageUrl) {
      fetchImage();
    }

    // Cleanup: revoke object URL saat component unmount atau imageUrl berubah
    return () => {
      if (objectUrl) {
        URL.revokeObjectURL(objectUrl);
      }
    };
  }, [imageUrl]);

  if (loading) {
    return (
      <div className={`flex items-center justify-center bg-gray-100 dark:bg-gray-800 rounded-xl ${className}`}>
        <div className="flex flex-col items-center gap-2 text-gray-400">
          <ImageIcon size={32} />
          <span className="text-xs">Memuat foto...</span>
        </div>
      </div>
    );
  }

  if (error || !blobUrl) {
    return (
      <div className={`flex items-center justify-center bg-gray-100 dark:bg-gray-800 rounded-xl ${className}`}>
        <div className="flex flex-col items-center gap-2 text-gray-400">
          <ImageIcon size={32} />
          <span className="text-xs">Foto tidak dapat ditampilkan</span>
        </div>
      </div>
    );
  }

  return (
    // eslint-disable-next-line @next/next/no-img-element
    <img
      src={blobUrl}
      alt={alt}
      className={className}
    />
  );
}

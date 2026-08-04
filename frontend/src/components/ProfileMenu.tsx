'use client';

import React, { useCallback, useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { LogOut, UserRound } from 'lucide-react';
import { useAuth } from '@/context/AuthContext';
import { API_BASE_URL } from '@/lib/api';

function getInitials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return '?';
  if (parts.length === 1) return parts[0].charAt(0).toUpperCase();
  return (parts[0].charAt(0) + parts[parts.length - 1].charAt(0)).toUpperCase();
}

interface ProfileMenuProps {
  /** Dipanggil saat item "Keluar" diklik (misal membuka dialog konfirmasi). */
  onLogoutRequest: () => void;
}

/**
 * ProfileMenu — badge profil (foto saja) dengan dropdown menu:
 * link ke halaman /profile dan tombol Keluar (warna kontras merah).
 * Foto diambil lewat endpoint yang dilindungi JWT; jika gagal/tidak ada,
 * ditampilkan inisial nama sebagai pengganti.
 */
export default function ProfileMenu({ onLogoutRequest }: ProfileMenuProps) {
  const { user } = useAuth();
  const [isOpen, setIsOpen] = useState(false);
  const [photoUrl, setPhotoUrl] = useState<string | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  // Ambil foto profil via endpoint yang dilindungi JWT (cookie dikirim otomatis).
  useEffect(() => {
    const profilePhoto = user?.profile_photo_url;
    let cancelled = false;
    let objectUrl: string | null = null;

    if (profilePhoto) {
      (async () => {
        try {
          const normalized = profilePhoto.startsWith('/uploads/')
            ? '/api' + profilePhoto
            : profilePhoto;
          const baseUrl = API_BASE_URL.replace(/\/api\/?$/, '');
          const fullUrl = normalized.startsWith('http')
            ? normalized
            : `${baseUrl}${normalized}`;

          const response = await fetch(fullUrl, { credentials: 'include' });
          if (!response.ok) throw new Error(`HTTP ${response.status}`);
          const blob = await response.blob();
          objectUrl = URL.createObjectURL(blob);
          if (!cancelled) setPhotoUrl(objectUrl);
        } catch {
          if (!cancelled) setPhotoUrl(null);
        }
      })();
    }

    return () => {
      cancelled = true;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [user?.profile_photo_url]);

  // Tutup dropdown saat klik di luar atau tombol Escape.
  useEffect(() => {
    if (!isOpen) return;

    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setIsOpen(false);
    };
    const closeOnOutsideClick = (event: MouseEvent) => {
      if (!containerRef.current?.contains(event.target as Node)) setIsOpen(false);
    };

    document.addEventListener('keydown', closeOnEscape);
    document.addEventListener('mousedown', closeOnOutsideClick);
    return () => {
      document.removeEventListener('keydown', closeOnEscape);
      document.removeEventListener('mousedown', closeOnOutsideClick);
    };
  }, [isOpen]);

  const handleLogout = useCallback(() => {
    setIsOpen(false);
    onLogoutRequest();
  }, [onLogoutRequest]);

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setIsOpen((open) => !open)}
        aria-label="Menu profil pengguna"
        aria-expanded={isOpen}
        title={user?.full_name}
        className="flex h-9 w-9 items-center justify-center overflow-hidden rounded-full ring-2 ring-border transition-colors hover:ring-primary/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-card"
      >
        {photoUrl && user?.profile_photo_url ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={photoUrl} alt={user?.full_name ?? 'Foto profil'} className="h-full w-full object-cover" />
        ) : (
          <span className="flex h-full w-full items-center justify-center bg-accent/20 text-xs font-bold text-[#775800] dark:text-accent">
            {user ? getInitials(user.full_name) : '?'}
          </span>
        )}
      </button>

      {isOpen && (
        <div
          role="menu"
          aria-label="Menu profil"
          className="absolute right-0 top-full z-50 mt-2 w-60 overflow-hidden rounded-2xl border border-border bg-card shadow-2xl"
        >
          <div className="flex items-center gap-3 border-b border-border px-4 py-3.5">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden rounded-full bg-accent/20 text-[#775800] dark:text-accent">
              {photoUrl && user?.profile_photo_url ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={photoUrl} alt={user?.full_name ?? 'Foto profil'} className="h-full w-full object-cover" />
              ) : (
                <span className="text-xs font-bold">{user ? getInitials(user.full_name) : '?'}</span>
              )}
            </div>
            <div className="min-w-0">
              <p className="truncate text-sm font-semibold">{user?.full_name ?? 'Pengguna'}</p>
              <span className={`mt-0.5 inline-block rounded-md px-1.5 py-px text-[10px] font-bold uppercase tracking-wider ${
                user?.role === 'admin'
                  ? 'bg-accent/20 text-[#775800] dark:text-accent'
                  : 'bg-success/10 text-success'
              }`}>
                {user?.role ?? ''}
              </span>
            </div>
          </div>

          <Link
            href="/profile"
            role="menuitem"
            onClick={() => setIsOpen(false)}
            className="flex w-full items-center gap-3 px-4 py-3 text-sm font-medium text-foreground transition-colors hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary"
          >
            <UserRound className="h-4 w-4 shrink-0 text-muted" />
            Profil
          </Link>
          <button
            type="button"
            role="menuitem"
            onClick={handleLogout}
            className="flex w-full items-center gap-3 border-t border-border px-4 py-3 text-sm font-semibold text-red-600 transition-colors hover:bg-red-500/10 dark:text-red-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-red-500"
          >
            <LogOut className="h-4 w-4 shrink-0" />
            Keluar
          </button>
        </div>
      )}
    </div>
  );
}

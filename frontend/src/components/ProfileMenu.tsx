'use client';

import React, { useCallback, useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import Link from 'next/link';
import { LogOut, UserRound, X } from 'lucide-react';
import { useAuth } from '@/context/AuthContext';
import { API_BASE_URL } from '@/lib/api';

function getInitials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return '?';
  if (parts.length === 1) return parts[0].charAt(0).toUpperCase();
  return (parts[0].charAt(0) + parts[parts.length - 1].charAt(0)).toUpperCase();
}

function formatRole(role?: string): string {
  if (!role) return '';
  return role.charAt(0).toUpperCase() + role.slice(1);
}

interface ProfileMenuProps {
  /** Tampilkan nama akun & peran di samping badge (default true). */
  showDetails?: boolean;
}

/**
 * ProfileMenu — badge profil (foto; fallback inisial) dengan nama akun & peran
 * di sampingnya (teks monokrom sesuai tema, dipisahkan garis pembatas).
 * Klik membuka dropdown: link "Profil" dan "Keluar" (kontras merah) yang memicu
 * dialog konfirmasi keluar (dirender via portal).
 */
export default function ProfileMenu({ showDetails = true }: ProfileMenuProps) {
  const { user, logout } = useAuth();
  const [isOpen, setIsOpen] = useState(false);
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
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

  // Tutup dialog konfirmasi saat tombol Escape.
  useEffect(() => {
    if (!isConfirmOpen) return;

    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setIsConfirmOpen(false);
    };
    document.addEventListener('keydown', closeOnEscape);
    return () => document.removeEventListener('keydown', closeOnEscape);
  }, [isConfirmOpen]);

  const handleLogoutRequest = useCallback(() => {
    setIsOpen(false);
    setIsConfirmOpen(true);
  }, []);

  const confirmLogout = useCallback(() => {
    setIsConfirmOpen(false);
    logout();
  }, [logout]);

  const avatar = photoUrl && user?.profile_photo_url ? (
    // eslint-disable-next-line @next/next/no-img-element
    <img src={photoUrl} alt={user?.full_name ?? 'Foto profil'} className="h-full w-full object-cover" />
  ) : (
    <span className="flex h-full w-full items-center justify-center bg-muted/60 text-xs font-bold text-foreground">
      {user ? getInitials(user.full_name) : '?'}
    </span>
  );

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setIsOpen((open) => !open)}
        aria-label="Menu profil pengguna"
        aria-expanded={isOpen}
        title={user?.full_name}
        className="group flex items-center gap-3 rounded-2xl px-1.5 py-1 transition-colors hover:bg-card focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-background"
      >
        <span className="flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-full bg-muted/60 ring-1 ring-border transition-colors group-hover:ring-primary/50">
          {avatar}
        </span>
        {showDetails && (
          <span className="hidden flex-col items-start border-l border-border pl-3 text-left leading-tight sm:flex">
            <span className="max-w-40 truncate text-sm font-semibold text-foreground">
              {user?.full_name ?? 'Pengguna'}
            </span>
            <span className="text-xs text-muted">{formatRole(user?.role)}</span>
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
            <div className="flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden rounded-full bg-muted/60 ring-1 ring-border">
              {avatar}
            </div>
            <div className="min-w-0">
              <p className="truncate text-sm font-semibold text-foreground">{user?.full_name ?? 'Pengguna'}</p>
              <p className="mt-0.5 text-xs text-muted">{formatRole(user?.role)}</p>
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
            onClick={handleLogoutRequest}
            className="flex w-full items-center gap-3 border-t border-border px-4 py-3 text-sm font-semibold text-red-600 transition-colors hover:bg-red-500/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-red-500 dark:text-red-400"
          >
            <LogOut className="h-4 w-4 shrink-0" />
            Keluar
          </button>
        </div>
      )}

      {/* Konfirmasi Keluar */}
      {isConfirmOpen &&
        createPortal(
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fadeIn" role="dialog" aria-modal="true" aria-labelledby="logout-confirm-heading">
            <div className="relative w-full max-w-sm space-y-5 rounded-2xl border border-border bg-card p-6 text-center shadow-2xl">
              <button
                type="button"
                onClick={() => setIsConfirmOpen(false)}
                aria-label="Tutup dialog konfirmasi"
                className="absolute right-3 top-3 rounded-lg p-1.5 text-muted transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
              >
                <X className="h-4 w-4" />
              </button>

              <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-red-500/10">
                <LogOut className="h-6 w-6 text-red-600 dark:text-red-400" />
              </div>

              <div>
                <h2 id="logout-confirm-heading" className="text-lg font-bold text-foreground">Keluar dari SIGAP?</h2>
                <p className="mt-1.5 text-sm text-muted">
                  Anda akan keluar dari sesi saat ini dan perlu login kembali untuk mengakses aplikasi.
                </p>
              </div>

              <div className="flex flex-col gap-2 sm:flex-row">
                <button
                  type="button"
                  onClick={() => setIsConfirmOpen(false)}
                  className="flex h-11 flex-1 items-center justify-center rounded-xl border border-border bg-card px-4 text-sm font-semibold text-foreground transition-colors hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
                >
                  Batal
                </button>
                <button
                  type="button"
                  onClick={confirmLogout}
                  autoFocus
                  className="flex h-11 flex-1 items-center justify-center gap-2 rounded-xl bg-red-600 px-4 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(220,38,38,0.25)] transition-colors hover:bg-red-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
                >
                  <LogOut className="h-4 w-4" />
                  Keluar
                </button>
              </div>
            </div>
          </div>,
          document.body
        )}
    </div>
  );
}

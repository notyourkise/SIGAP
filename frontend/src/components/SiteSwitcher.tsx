'use client';

import React, { useEffect, useRef, useState, useSyncExternalStore } from 'react';
import { Building2, Check, ChevronDown, Globe, LoaderCircle } from 'lucide-react';
import { useAuth } from '@/context/AuthContext';
import { clearCookie, getAllSitesForSelect, getCookie, setCookie, type Site } from '@/lib/api';
import { ALL_SITES_LABEL, resolveSiteSwitcherState } from '@/lib/site-switcher';

const SITE_COOKIE = 'sigap_site_id';
const emptySubscribe = () => () => {};

/**
 * SiteSwitcher — dropdown untuk mengganti site/kawasan aktif di header.
 * - Memuat daftar site via GET /sites/all saat mount.
 * - Admin dapat memilih "Semua Site" (membersihkan cookie sigap_site_id) atau
 *   memilih satu kawasan tertentu; bila tanpa cookie, label tetap "Semua Site".
 * - Teknisi hanya memilih di antara site miliknya (tanpa opsi "Semua Site").
 * - Saat pengguna memilih site: simpan id ke cookie (30 hari) lalu reload halaman
 *   agar seluruh data di-refetch dengan site baru (header X-Site-ID otomatis).
 */
export default function SiteSwitcher() {
  const { user, loading: authLoading } = useAuth();
  const [sites, setSites] = useState<Site[]>([]);
  const [loading, setLoading] = useState(true);
  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const hydrated = useSyncExternalStore(
    emptySubscribe,
    () => true,
    () => false
  );
  const activeSiteId = hydrated ? getCookie(SITE_COOKIE) : null;

  // Muat daftar site untuk dropdown.
  useEffect(() => {
    let active = true;
    getAllSitesForSelect()
      .then((response) => {
        if (!active) return;
        if (response.status === 'success' && response.data) {
          setSites(response.data);
        }
      })
      .catch(() => {
        // Gagal memuat daftar site tidak fatal — dropdown kosong.
        if (active) setSites([]);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

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

  const {
    canSelectAllSites,
    activeSite,
    activeLabel,
    isAllSitesActive,
  } = resolveSiteSwitcherState({
    role: user?.role,
    sites,
    activeSiteId,
    loading: loading || authLoading,
    hydrated,
  });

  const handleSelectAllSites = () => {
    setIsOpen(false);
    clearCookie(SITE_COOKIE);
    window.location.reload();
  };

  const handleSelect = (site: Site) => {
    setIsOpen(false);
    // Simpan site terpilih ke cookie lalu muat ulang agar seluruh data di-refetch.
    setCookie(SITE_COOKIE, site.id, 30);
    window.location.reload();
  };

  const isBusy = loading || authLoading || !hydrated;

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setIsOpen((open) => !open)}
        aria-label="Ganti site / kawasan aktif"
        aria-expanded={isOpen}
        aria-haspopup="menu"
        title={activeSite ? `${activeSite.nama} (${activeSite.kode})` : activeLabel}
        className="group flex h-10 max-w-[220px] items-center gap-2 rounded-xl border border-border bg-background px-3 text-sm font-semibold text-foreground transition-colors hover:border-primary/30 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-card"
      >
        {isBusy ? (
          <LoaderCircle className="h-4 w-4 shrink-0 animate-spin text-muted" aria-hidden="true" />
        ) : isAllSitesActive ? (
          <Globe className="h-4 w-4 shrink-0 text-primary" aria-hidden="true" />
        ) : (
          <Building2 className="h-4 w-4 shrink-0 text-primary" aria-hidden="true" />
        )}
        <span className="min-w-0 flex-1 truncate text-left">{activeLabel}</span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 text-muted transition-transform duration-200 ${isOpen ? 'rotate-180' : ''}`}
          aria-hidden="true"
        />
      </button>

      {isOpen && (
        <div
          role="menu"
          aria-label="Pilih site / kawasan"
          className="absolute right-0 top-full z-50 mt-2 w-64 overflow-hidden rounded-2xl border border-border bg-card shadow-2xl"
        >
          <div className="border-b border-border px-4 py-2.5">
            <p className="text-xs font-semibold text-muted">Pilih Kawasan / Site</p>
          </div>
          <div className="max-h-72 overflow-y-auto py-1">
            {isBusy ? (
              <div className="flex items-center justify-center gap-2 px-4 py-6 text-xs text-muted">
                <LoaderCircle className="h-4 w-4 animate-spin" />
                Memuat site…
              </div>
            ) : (
              <>
                {canSelectAllSites && (
                  <button
                    type="button"
                    role="menuitem"
                    onClick={handleSelectAllSites}
                    className="flex w-full items-center gap-3 border-b border-border/60 px-4 py-2.5 text-left text-sm transition-colors hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary"
                  >
                    <Globe className="h-4 w-4 shrink-0 text-primary" aria-hidden="true" />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate font-medium text-foreground">{ALL_SITES_LABEL}</span>
                      <span className="block truncate text-[11px] text-muted">Seluruh kawasan tanpa filter</span>
                    </span>
                    {isAllSitesActive && <Check className="h-4 w-4 shrink-0 text-primary" aria-hidden="true" />}
                  </button>
                )}
                {sites.length === 0 ? (
                  <div className="px-4 py-6 text-center text-xs text-muted">Belum ada site tersedia.</div>
                ) : (
                  sites.map((site) => {
                    const isActive = activeSite?.id === site.id;
                    return (
                      <button
                        key={site.id}
                        type="button"
                        role="menuitem"
                        onClick={() => handleSelect(site)}
                        className="flex w-full items-center gap-3 px-4 py-2.5 text-left text-sm transition-colors hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary"
                      >
                        <span className="min-w-0 flex-1">
                          <span className="block truncate font-medium text-foreground">{site.nama}</span>
                          <span className="block truncate font-mono text-[11px] text-muted">{site.kode}</span>
                        </span>
                        {isActive && <Check className="h-4 w-4 shrink-0 text-primary" aria-hidden="true" />}
                      </button>
                    );
                  })
                )}
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

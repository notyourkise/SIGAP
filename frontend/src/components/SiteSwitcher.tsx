'use client';

import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Building2, Check, ChevronDown, LoaderCircle } from 'lucide-react';
import { getAllSitesForSelect, getCookie, setCookie, type Site } from '@/lib/api';

const SITE_COOKIE = 'sigap_site_id';

/**
 * SiteSwitcher — dropdown untuk mengganti site/kawasan aktif di header.
 * - Memuat daftar site via GET /sites/all saat mount.
 * - Menampilkan label nama + kode pada setiap opsi.
 * - Saat pengguna memilih site: simpan id ke cookie (30 hari) lalu reload halaman
 *   agar seluruh data di-refetch dengan site baru (header X-Site-ID otomatis).
 * - Label aktif diambil dari cookie sigap_site_id; fallback ke site default
 *   atau "Semua site" bila belum ada site terpilih.
 */
export default function SiteSwitcher() {
  const [sites, setSites] = useState<Site[]>([]);
  const [loading, setLoading] = useState(true);
  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const activeSiteId = getCookie(SITE_COOKIE);

  // Muat daftar site untuk dropdown.
  const loadSites = useCallback(async () => {
    setLoading(true);
    try {
      const response = await getAllSitesForSelect();
      if (response.status === 'success' && response.data) {
        setSites(response.data);
      }
    } catch {
      // Gagal memuat daftar site tidak fatal — dropdown kosong.
      setSites([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadSites();
  }, [loadSites]);

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

  // Tentukan label site yang sedang aktif:
  // prioritas cookie aktif → site default → "Semua site".
  const activeSite = activeSiteId
    ? sites.find((s) => s.id === activeSiteId)
    : sites.find((s) => s.is_default);

  const activeLabel = activeSite
    ? activeSite.nama
    : loading
      ? 'Memuat…'
      : 'Semua site';

  const handleSelect = (site: Site) => {
    setIsOpen(false);
    // Simpan site terpilih ke cookie lalu muat ulang agar seluruh data di-refetch.
    setCookie(SITE_COOKIE, site.id, 30);
    window.location.reload();
  };

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
        {loading ? (
          <LoaderCircle className="h-4 w-4 shrink-0 animate-spin text-muted" aria-hidden="true" />
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
            {loading ? (
              <div className="flex items-center justify-center gap-2 px-4 py-6 text-xs text-muted">
                <LoaderCircle className="h-4 w-4 animate-spin" />
                Memuat site…
              </div>
            ) : sites.length === 0 ? (
              <div className="px-4 py-6 text-center text-xs text-muted">Belum ada site tersedia.</div>
            ) : (
              sites.map((site) => {
                const isActive = site.id === activeSiteId || (!activeSiteId && site.is_default);
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
          </div>
        </div>
      )}
    </div>
  );
}

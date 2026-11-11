import type { Site } from './api';

export const ALL_SITES_LABEL = 'Semua Site';

export interface ResolveSiteSwitcherInput {
  role?: string | null;
  sites: Site[];
  activeSiteId?: string | null;
  loading?: boolean;
  hydrated?: boolean;
}

export interface ResolvedSiteSwitcherState {
  isAdmin: boolean;
  canSelectAllSites: boolean;
  activeSite: Site | null;
  activeLabel: string;
  isAllSitesActive: boolean;
}

/**
 * Menentukan state tampilan dropdown SiteSwitcher berdasarkan peran pengguna
 * dan cookie `sigap_site_id`.
 *
 * Aturan bisnis:
 * - Admin yang belum memilih site (tidak ada cookie / cookie tidak valid) berada
 *   pada mode global "Semua Site" — TIDAK boleh diam-diam jatuh ke site default.
 * - Non-admin (teknisi) tidak pernah masuk ke mode "Semua Site"; bila belum ada
 *   cookie valid, pilih `is_default` lalu site pertama sebagai fallback yang
 *   sejalan dengan `defaultSiteForUser` di backend.
 */
export function resolveSiteSwitcherState({
  role,
  sites,
  activeSiteId,
  loading = false,
  hydrated = true,
}: ResolveSiteSwitcherInput): ResolvedSiteSwitcherState {
  const isAdmin = role === 'admin';
  const canSelectAllSites = isAdmin;

  if (!hydrated || (loading && sites.length === 0)) {
    return {
      isAdmin,
      canSelectAllSites,
      activeSite: null,
      activeLabel: 'Memuat…',
      isAllSitesActive: false,
    };
  }

  const normalizedId = activeSiteId?.trim() || null;
  const cookieSite = normalizedId ? sites.find((s) => s.id === normalizedId) ?? null : null;

  if (isAdmin) {
    if (cookieSite) {
      return {
        isAdmin,
        canSelectAllSites,
        activeSite: cookieSite,
        activeLabel: cookieSite.nama,
        isAllSitesActive: false,
      };
    }
    return {
      isAdmin,
      canSelectAllSites,
      activeSite: null,
      activeLabel: ALL_SITES_LABEL,
      isAllSitesActive: true,
    };
  }

  const fallbackSite = sites.find((s) => s.is_default) ?? sites[0] ?? null;
  const resolvedSite = cookieSite ?? fallbackSite;

  return {
    isAdmin,
    canSelectAllSites,
    activeSite: resolvedSite,
    activeLabel: resolvedSite ? resolvedSite.nama : 'Belum ada site',
    isAllSitesActive: false,
  };
}

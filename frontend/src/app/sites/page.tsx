'use client';

import React, { useCallback, useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/DashboardLayout';
import Pagination from '@/components/Pagination';
import SortableHeader from '@/components/SortableHeader';
import { useAuth } from '@/context/AuthContext';
import { apiRequest } from '@/lib/api';
import {
  getSites,
  createSite,
  updateSite,
  deleteSite,
  assignUsersToSite,
  type Site,
} from '@/lib/api';
import { useToast } from '@/components/Toast';
import { useConfirm } from '@/components/ConfirmDialog';
import {
  Plus,
  Trash2,
  Building2,
  AlertTriangle,
  Search,
  Edit2,
  X,
  LoaderCircle,
  Users,
  Check,
  MapPin,
} from 'lucide-react';

// Item teknisi yang bisa ditugaskan ke site (diambil dari /users?role=teknisi).
interface TechnicianItem {
  id: string;
  full_name: string;
  username: string;
  role: string;
}

// Data formulir tambah/edit site.
interface SiteFormData {
  nama: string;
  kode: string;
  alamat: string;
  koordinat_lat: string;
  koordinat_lng: string;
  is_default: boolean;
}

const emptyForm: SiteFormData = {
  nama: '',
  kode: '',
  alamat: '',
  koordinat_lat: '',
  koordinat_lng: '',
  is_default: false,
};

// Bantu konversi string koordinat ke angka/null (kosong → null).
function parseCoord(value: string): number | null {
  const trimmed = value.trim();
  if (trimmed === '') return null;
  const num = Number(trimmed);
  return Number.isFinite(num) ? num : null;
}

export default function SitesPage() {
  const { user, loading: authLoading } = useAuth();
  const router = useRouter();
  const toast = useToast();
  const confirm = useConfirm();

  const [sites, setSites] = useState<Site[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [page, setPage] = useState(1);
  const [pagination, setPagination] = useState({ page: 1, limit: 10, total_rows: 0, total_pages: 1 });
  const [search, setSearch] = useState('');
  const [sortBy, setSortBy] = useState<'nama' | 'kode' | 'alamat' | 'user_count'>('nama');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc');

  // Modal form tambah/edit site
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [current, setCurrent] = useState<Site | null>(null);
  const [formData, setFormData] = useState<SiteFormData>(emptyForm);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Modal atur teknisi
  const [assignSite, setAssignSite] = useState<Site | null>(null);
  const [technicians, setTechnicians] = useState<TechnicianItem[]>([]);
  const [selectedUserIds, setSelectedUserIds] = useState<string[]>([]);

  const handleSort = (key: string) => {
    const nextKey = key as 'nama' | 'kode' | 'alamat' | 'user_count';
    setSortOrder((current) => (sortBy === nextKey ? (current === 'asc' ? 'desc' : 'asc') : nextKey === 'user_count' ? 'desc' : 'asc'));
    setSortBy(nextKey);
    setPage(1);
  };

  // Auth guard (admin only)
  useEffect(() => {
    if (!authLoading && user?.role !== 'admin') router.replace('/dashboard');
  }, [authLoading, router, user]);

  const fetchSites = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const response = await getSites({ page, limit: 10, search: search || undefined, sort_by: sortBy, order: sortOrder });
      if (response.status === 'success' && response.data) {
        setSites(response.data.rows || []);
        setPagination(response.data.pagination || { page: 1, limit: 10, total_rows: 0, total_pages: 1 });
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Data site belum dapat dimuat. Coba lagi.');
    } finally {
      setLoading(false);
    }
  }, [page, search, sortBy, sortOrder]);

  useEffect(() => {
    if (user?.role !== 'admin') return;
    let cancelled = false;
    const load = async () => {
      await Promise.resolve();
      if (!cancelled) await fetchSites();
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, [fetchSites, user?.role]);

  const openForm = (site: Site | null = null) => {
    setCurrent(site);
    setFormData(
      site
        ? {
            nama: site.nama,
            kode: site.kode,
            alamat: site.alamat || '',
            koordinat_lat: site.koordinat_lat != null ? String(site.koordinat_lat) : '',
            koordinat_lng: site.koordinat_lng != null ? String(site.koordinat_lng) : '',
            is_default: site.is_default,
          }
        : emptyForm
    );
    setIsFormOpen(true);
  };

  const closeForm = () => {
    if (actionLoading === 'save') return;
    setIsFormOpen(false);
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.nama.trim() || !formData.kode.trim()) {
      toast.error('Nama dan kode site wajib diisi.');
      return;
    }
    const payload = {
      nama: formData.nama.trim(),
      kode: formData.kode.trim(),
      alamat: formData.alamat.trim(),
      koordinat_lat: parseCoord(formData.koordinat_lat),
      koordinat_lng: parseCoord(formData.koordinat_lng),
      is_default: formData.is_default,
    };
    setActionLoading('save');
    try {
      if (current) {
        const response = await updateSite(current.id, payload);
        if (response.status === 'success') {
          setIsFormOpen(false);
          toast.success(`Site "${payload.nama}" berhasil diperbarui.`);
          fetchSites();
        }
      } else {
        const response = await createSite(payload);
        if (response.status === 'success') {
          setIsFormOpen(false);
          toast.success(`Site "${payload.nama}" berhasil ditambahkan.`);
          fetchSites();
        }
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Site belum dapat disimpan. Coba lagi.', err);
    } finally {
      setActionLoading(null);
    }
  };

  const handleDelete = async (site: Site) => {
    const ok = await confirm({
      title: 'Hapus Site?',
      message: `Hapus site "${site.nama}" (${site.kode})? Tindakan ini tidak dapat dibatalkan.`,
      confirmLabel: 'Hapus',
    });
    if (!ok) return;
    setActionLoading(`delete-${site.id}`);
    try {
      const response = await deleteSite(site.id);
      if (response.status === 'success') {
        toast.success(`Site "${site.nama}" berhasil dihapus.`);
        fetchSites();
      }
    } catch (err) {
      // Backend menolak hapus site default — tampilkan pesannya.
      toast.error(err instanceof Error ? err.message : 'Site belum dapat dihapus. Coba lagi.', err);
    } finally {
      setActionLoading(null);
    }
  };

  // Muat daftar teknisi saat modal atur teknisi dibuka.
  const openAssign = async (site: Site) => {
    setAssignSite(site);
    setSelectedUserIds([]);
    try {
      const userResult = await apiRequest<{ rows: TechnicianItem[] }>(
        '/users?role=teknisi&page=1&limit=1000'
      );
      const rows = userResult.data?.rows || [];
      setTechnicians(rows);
      // Preselect teknisi yang sudah ditugaskan ke site ini dari detail site.
      try {
        const detail = await apiRequest<Site>(`/sites/${site.id}`);
        const detailData = detail.data as (Site & { users?: { id: string }[] }) | undefined;
        setSelectedUserIds((detailData?.users || []).map((u) => u.id));
      } catch {
        // Bila detail gagal dimuat, mulai tanpa preselect (tidak fatal).
      }
    } catch {
      toast.error('Daftar teknisi belum dapat dimuat.');
    }
  };

  const toggleId = (arr: string[], id: string, setter: (v: string[]) => void) => {
    setter(arr.includes(id) ? arr.filter((x) => x !== id) : [...arr, id]);
  };

  const saveAssignments = async () => {
    if (!assignSite) return;
    setActionLoading('assign');
    try {
      const response = await assignUsersToSite(assignSite.id, selectedUserIds);
      if (response.status === 'success') {
        toast.success(`Penugasan teknisi untuk "${assignSite.nama}" berhasil disimpan.`);
        setAssignSite(null);
        fetchSites();
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Penugasan teknisi belum dapat disimpan.', err);
    } finally {
      setActionLoading(null);
    }
  };

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-[-0.025em]">Kawasan / Site</h1>
            <p className="mt-1 max-w-2xl text-sm text-muted">
              Kelola kawasan/site untuk memfilter data CCTV dan penugasan teknisi multi-site.
            </p>
          </div>
          <button
            type="button"
            onClick={() => openForm()}
            disabled={actionLoading !== null}
            className="flex min-h-11 items-center justify-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(15,118,74,0.18)] transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-60"
          >
            <Plus className="h-4 w-4" />
            Tambah Site
          </button>
        </div>

        <section aria-labelledby="site-list-heading" aria-busy={loading} className="overflow-hidden rounded-2xl border border-border bg-card">
          <div className="flex flex-col gap-3 border-b border-border px-5 py-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h2 id="site-list-heading" className="font-semibold">Daftar Site</h2>
              <p className="mt-0.5 text-xs text-muted">{pagination.total_rows} site</p>
            </div>
            <div className="relative max-w-md">
              <Search className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" />
              <input
                type="search"
                aria-label="Cari site"
                placeholder="Cari nama atau kode site"
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setPage(1);
                }}
                className="w-full rounded-xl border border-input-border bg-input py-2.5 pl-10 pr-4 text-sm text-foreground outline-none transition-colors focus:border-primary focus:ring-2 focus:ring-primary/20"
              />
            </div>
          </div>

          {loading ? (
            <div className="flex items-center justify-center gap-3 py-20" role="status">
              <LoaderCircle className="h-6 w-6 animate-spin text-primary" />
              <span className="text-sm text-muted">Memuat site…</span>
            </div>
          ) : error ? (
            <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
              <AlertTriangle className="h-10 w-10 text-error" />
              <h3 className="mt-3 text-base font-semibold">Data site belum dapat dimuat</h3>
              <p className="mt-1 max-w-md text-xs text-muted">{error}</p>
              <button
                type="button"
                onClick={fetchSites}
                className="mt-4 rounded-lg bg-primary px-4 py-2 text-xs font-bold text-white transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
              >
                Coba lagi
              </button>
            </div>
          ) : sites.length === 0 ? (
            <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
              <Building2 className="h-10 w-10 text-muted/45" />
              <h3 className="mt-3 text-base font-semibold">{search ? 'Tidak ada site yang cocok' : 'Belum ada site'}</h3>
              <p className="mt-1 max-w-sm text-xs text-muted">
                {search ? 'Ubah kata kunci pencarian.' : 'Gunakan "Tambah Site" untuk membuat kawasan/site baru.'}
              </p>
            </div>
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="w-full min-w-[900px] border-collapse text-left">
                  <thead>
                    <tr className="border-b border-border bg-background/50 text-xs font-semibold text-muted">
                      <th className="px-5 py-3.5">
                        <SortableHeader label="Nama" sortKey="nama" activeKey={sortBy} direction={sortOrder} onSort={handleSort} />
                      </th>
                      <th className="px-5 py-3.5">
                        <SortableHeader label="Kode" sortKey="kode" activeKey={sortBy} direction={sortOrder} onSort={handleSort} />
                      </th>
                      <th className="px-5 py-3.5">
                        <SortableHeader label="Alamat" sortKey="alamat" activeKey={sortBy} direction={sortOrder} onSort={handleSort} />
                      </th>
                      <th className="px-5 py-3.5">Koordinat</th>
                      <th className="px-5 py-3.5">Default</th>
                      <th className="px-5 py-3.5">
                        <SortableHeader label="Teknisi" sortKey="user_count" activeKey={sortBy} direction={sortOrder} onSort={handleSort} />
                      </th>
                      <th className="px-5 py-3.5 text-right">Tindakan</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border text-sm">
                    {sites.map((site) => (
                      <tr key={site.id} className="transition-colors hover:bg-background/40">
                        <td className="px-5 py-4">
                          <p className="font-semibold">{site.nama}</p>
                        </td>
                        <td className="px-5 py-4">
                          <span className="font-mono text-xs font-semibold text-primary">{site.kode}</span>
                        </td>
                        <td className="max-w-[260px] px-5 py-4 text-muted">
                          <span className="line-clamp-2">{site.alamat || '-'}</span>
                        </td>
                        <td className="px-5 py-4">
                          {site.koordinat_lat != null && site.koordinat_lng != null ? (
                            <span className="inline-flex items-center gap-1 rounded-full border border-border bg-background px-2.5 py-1 text-xs font-medium text-muted">
                              <MapPin className="h-3.5 w-3.5 text-primary" />
                              {site.koordinat_lat.toFixed(4)}, {site.koordinat_lng.toFixed(4)}
                            </span>
                          ) : (
                            <span className="text-xs text-muted">-</span>
                          )}
                        </td>
                        <td className="px-5 py-4">
                          {site.is_default ? (
                            <span className="inline-flex items-center gap-1.5 rounded-full border border-primary/25 bg-primary/10 px-2.5 py-1 text-xs font-semibold text-primary">
                              <Check className="h-3.5 w-3.5" />
                              Default
                            </span>
                          ) : (
                            <span className="text-xs text-muted">-</span>
                          )}
                        </td>
                        <td className="px-5 py-4">
                          <span className="inline-flex items-center gap-1.5 rounded-full border border-border bg-background px-2.5 py-1 text-xs font-semibold">
                            <Users className="h-3.5 w-3.5 text-primary" />
                            {site.user_count ?? 0}
                          </span>
                        </td>
                        <td className="px-5 py-4 text-right">
                          <div className="flex items-center justify-end gap-1">
                            <button
                              type="button"
                              onClick={() => openAssign(site)}
                              disabled={actionLoading !== null}
                              className="rounded-lg px-2.5 py-2 text-xs font-bold text-primary transition-colors hover:bg-primary/10 disabled:opacity-50"
                              title="Atur teknisi pada site ini"
                            >
                              Atur Teknisi
                            </button>
                            <button
                              type="button"
                              onClick={() => openForm(site)}
                              disabled={actionLoading !== null}
                              className="rounded-lg p-2.5 text-muted transition-colors hover:bg-primary/10 hover:text-primary disabled:opacity-50"
                              title="Ubah site"
                              aria-label={`Ubah site ${site.nama}`}
                            >
                              <Edit2 className="h-4 w-4" />
                            </button>
                            <button
                              type="button"
                              onClick={() => handleDelete(site)}
                              disabled={actionLoading !== null}
                              className="rounded-lg p-2.5 text-muted transition-colors hover:bg-error/10 hover:text-error disabled:opacity-50"
                              title="Hapus site"
                              aria-label={`Hapus site ${site.nama}`}
                            >
                              {actionLoading === `delete-${site.id}` ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <Pagination
                page={pagination.page}
                limit={pagination.limit}
                totalRows={pagination.total_rows}
                totalPages={pagination.total_pages}
                onPageChange={setPage}
                itemLabel="site"
              />
            </>
          )}
        </section>
      </div>

      {/* MODAL FORM TAMBAH/EDIT SITE */}
      {isFormOpen && (
        <div className="fixed inset-0 flex items-center justify-center z-50 px-4">
          <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={closeForm}></div>
          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby="site-form-title"
            className="z-10 max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl border border-border bg-card p-5 shadow-2xl sm:p-6"
          >
            <div className="mb-6 flex items-start justify-between gap-4">
              <div>
                <h3 id="site-form-title" className="text-xl font-bold tracking-[-0.015em]">
                  {current ? 'Ubah Site' : 'Tambah Site'}
                </h3>
                <p className="mt-1 text-sm text-muted">Isi informasi kawasan/site.</p>
              </div>
              <button type="button" aria-label="Tutup formulir site" onClick={closeForm} disabled={actionLoading === 'save'} className="shrink-0 rounded-lg p-2.5 text-muted transition-colors hover:bg-background hover:text-foreground disabled:opacity-50">
                <X className="h-4 w-4" />
              </button>
            </div>

            <form onSubmit={handleSave} className="space-y-4">
              <div className="grid gap-4 sm:grid-cols-2">
                <div>
                  <label htmlFor="site-nama" className="mb-1.5 block text-xs font-semibold text-muted">Nama Site *</label>
                  <input
                    id="site-nama"
                    autoFocus
                    type="text"
                    value={formData.nama}
                    onChange={(e) => setFormData({ ...formData, nama: e.target.value })}
                    placeholder="Contoh: Kawasan Inti Pusat Pemerintahan"
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    required
                  />
                </div>
                <div>
                  <label htmlFor="site-kode" className="mb-1.5 block text-xs font-semibold text-muted">Kode *</label>
                  <input
                    id="site-kode"
                    type="text"
                    value={formData.kode}
                    onChange={(e) => setFormData({ ...formData, kode: e.target.value })}
                    placeholder="Contoh: KIPP"
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    required
                  />
                </div>
              </div>
              <div>
                <label htmlFor="site-alamat" className="mb-1.5 block text-xs font-semibold text-muted">Alamat</label>
                <textarea
                  id="site-alamat"
                  rows={2}
                  value={formData.alamat}
                  onChange={(e) => setFormData({ ...formData, alamat: e.target.value })}
                  placeholder="Alamat / lokasi site (opsional)"
                  className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                />
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <div>
                  <label htmlFor="site-lat" className="mb-1.5 block text-xs font-semibold text-muted">Latitude</label>
                  <input
                    id="site-lat"
                    type="number"
                    step="any"
                    value={formData.koordinat_lat}
                    onChange={(e) => setFormData({ ...formData, koordinat_lat: e.target.value })}
                    placeholder="Mis. -0.9723"
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                  />
                </div>
                <div>
                  <label htmlFor="site-lng" className="mb-1.5 block text-xs font-semibold text-muted">Longitude</label>
                  <input
                    id="site-lng"
                    type="number"
                    step="any"
                    value={formData.koordinat_lng}
                    onChange={(e) => setFormData({ ...formData, koordinat_lng: e.target.value })}
                    placeholder="Mis. 116.7083"
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                  />
                </div>
              </div>
              <label className="flex items-center gap-2.5 rounded-xl border border-border bg-background px-4 py-3 text-sm font-medium cursor-pointer">
                <input
                  type="checkbox"
                  checked={formData.is_default}
                  onChange={(e) => setFormData({ ...formData, is_default: e.target.checked })}
                  disabled={Boolean(current && current.is_default)}
                  className="h-4 w-4 rounded border-input-border text-primary focus:ring-primary/30 disabled:cursor-not-allowed disabled:opacity-50"
                />
                Jadikan site default
                <span className="ml-auto text-xs text-muted">Site default tidak dapat dihapus.</span>
              </label>
              <div className="flex justify-end gap-3 pt-4 border-t border-border mt-6">
                <button type="button" onClick={closeForm} disabled={actionLoading === 'save'} className="rounded-xl border border-border px-5 py-2.5 text-sm font-semibold transition-colors hover:bg-background disabled:opacity-50">
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={actionLoading === 'save'}
                  className="flex items-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(15,118,74,0.16)] transition-colors hover:bg-primary-hover disabled:opacity-65"
                >
                  {actionLoading === 'save' && <LoaderCircle className="h-4 w-4 animate-spin" />}
                  {actionLoading === 'save' ? 'Menyimpan…' : current ? 'Simpan Perubahan' : 'Simpan Site'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL ATUR TEKNISI */}
      {assignSite && (
        <div className="fixed inset-0 flex items-center justify-center z-50 px-4">
          <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => { if (actionLoading !== 'assign') setAssignSite(null); }}></div>
          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby="site-assign-title"
            className="z-10 max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl border border-border bg-card p-5 shadow-2xl sm:p-6"
          >
            <div className="mb-6 flex items-start justify-between gap-4">
              <div>
                <h3 id="site-assign-title" className="text-xl font-bold tracking-[-0.015em]">Atur Teknisi — {assignSite.nama}</h3>
                <p className="mt-1 text-sm text-muted">Pilih teknisi yang bertanggung jawab pada site ini.</p>
              </div>
              <button type="button" aria-label="Tutup" onClick={() => { if (actionLoading !== 'assign') setAssignSite(null); }} disabled={actionLoading === 'assign'} className="shrink-0 rounded-lg p-2.5 text-muted transition-colors hover:bg-background hover:text-foreground disabled:opacity-50">
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="mb-2 flex items-center justify-between">
              <h4 className="text-sm font-semibold flex items-center gap-2">
                <Users className="h-4 w-4 text-primary" /> Teknisi ({selectedUserIds.length} dipilih)
              </h4>
            </div>
            <div className="max-h-72 overflow-y-auto rounded-xl border border-border">
              {technicians.length === 0 ? (
                <div className="px-4 py-6 text-center text-xs text-muted">Tidak ada teknisi tersedia.</div>
              ) : (
                <ul className="divide-y divide-border text-sm">
                  {technicians.map((tech) => (
                    <li key={tech.id}>
                      <label className="flex cursor-pointer items-center gap-3 px-4 py-2.5 hover:bg-background/40">
                        <input
                          type="checkbox"
                          checked={selectedUserIds.includes(tech.id)}
                          onChange={() => toggleId(selectedUserIds, tech.id, setSelectedUserIds)}
                          className="h-4 w-4 rounded border-input-border text-primary focus:ring-primary/30"
                        />
                        <span className="font-semibold">{tech.full_name}</span>
                        <span className="text-xs text-muted">@{tech.username}</span>
                      </label>
                    </li>
                  ))}
                </ul>
              )}
            </div>

            <div className="flex justify-end gap-3 pt-4 border-t border-border mt-6">
              <button type="button" onClick={() => { if (actionLoading !== 'assign') setAssignSite(null); }} disabled={actionLoading === 'assign'} className="rounded-xl border border-border px-5 py-2.5 text-sm font-semibold transition-colors hover:bg-background disabled:opacity-50">
                Batal
              </button>
              <button
                type="button"
                onClick={saveAssignments}
                disabled={actionLoading === 'assign'}
                className="flex items-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(15,118,74,0.16)] transition-colors hover:bg-primary-hover disabled:opacity-65"
              >
                {actionLoading === 'assign' ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}
                {actionLoading === 'assign' ? 'Menyimpan…' : 'Simpan Penugasan'}
              </button>
            </div>
          </div>
        </div>
      )}
    </DashboardLayout>
  );
}

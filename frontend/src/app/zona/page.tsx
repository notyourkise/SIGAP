'use client';

import React, { useCallback, useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/DashboardLayout';
import Pagination from '@/components/Pagination';
import SortableHeader from '@/components/SortableHeader';
import { useSortableRows } from '@/hooks/useSortableRows';
import { useAuth } from '@/context/AuthContext';
import { apiRequest } from '@/lib/api';
import { useToast } from '@/components/Toast';
import { useConfirm } from '@/components/ConfirmDialog';
import {
  Plus,
  Trash2,
  MapPinned,
  AlertTriangle,
  Search,
  Edit2,
  X,
  LoaderCircle,
  Users,
  Camera,
  Check,
} from 'lucide-react';

interface Zone {
  id: string;
  nama: string;
  deskripsi: string;
  created_at: string;
  cctv_count?: number;
  user_count?: number;
}

interface ZoneRow extends Zone {
  cctvs?: Array<{ id: string; kode_cctv: string; nama: string; lokasi_area?: string }>;
  users?: Array<{ id: string; full_name: string; role: string }>;
}

interface ZonesResponse {
  rows: Zone[];
  pagination: {
    total_rows: number;
    total_pages: number;
    page: number;
    limit: number;
  };
}

interface CCTVItem {
  id: string;
  kode_cctv: string;
  nama: string;
  lokasi_area?: string;
  zone_id?: string | null;
}

interface UserItem {
  id: string;
  full_name: string;
  username: string;
  role: string;
}

export default function ZonaPage() {
  const { user, loading: authLoading } = useAuth();
  const router = useRouter();
  const toast = useToast();
  const confirm = useConfirm();

  const [zones, setZones] = useState<Zone[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [page, setPage] = useState(1);
  const [pagination, setPagination] = useState({ page: 1, limit: 10, total_rows: 0, total_pages: 1 });
  const [search, setSearch] = useState('');

  // Modal form zona
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [current, setCurrent] = useState<Zone | null>(null);
  const [formData, setFormData] = useState({ nama: '', deskripsi: '' });
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Modal assign
  const [assignZone, setAssignZone] = useState<Zone | null>(null);
  const [cctvs, setCctvs] = useState<CCTVItem[]>([]);
  const [technicians, setTechnicians] = useState<UserItem[]>([]);
  const [selectedCctvIds, setSelectedCctvIds] = useState<string[]>([]);
  const [selectedUserIds, setSelectedUserIds] = useState<string[]>([]);

  // Sortir kolom (client-side) — hanya mengurutkan baris yang sudah ada di state
  const { sortedRows, sortKey, sortDirection, onSort } = useSortableRows<Zone>(zones, {
    getValue: (row, key) => {
      switch (key) {
        case 'nama':
          return row.nama;
        case 'deskripsi':
          return row.deskripsi ?? '';
        case 'cctv':
          return row.cctv_count ?? 0;
        case 'teknisi':
          return row.user_count ?? 0;
        default:
          return '';
      }
    },
  });

  // Auth guard (admin only)
  useEffect(() => {
    if (!authLoading && user?.role !== 'admin') router.replace('/dashboard');
  }, [authLoading, router, user]);

  const fetchZones = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const params = new URLSearchParams({ page: String(page), limit: '10' });
      if (search) params.set('search', search);
      const response = await apiRequest<ZonesResponse>(`/zones?${params}`);
      if (response.status === 'success' && response.data) {
        setZones(response.data.rows || []);
        setPagination(response.data.pagination || { page: 1, limit: 10, total_rows: 0, total_pages: 1 });
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Data zona belum dapat dimuat. Coba lagi.');
    } finally {
      setLoading(false);
    }
  }, [page, search]);

  useEffect(() => {
    if (user?.role !== 'admin') return;
    let cancelled = false;
    const load = async () => {
      await Promise.resolve();
      if (!cancelled) await fetchZones();
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, [fetchZones, user?.role]);

  // Muat daftar CCTV & teknisi saat modal assign dibuka
  const openAssign = async (zone: Zone) => {
    setAssignZone(zone);
    setSelectedCctvIds([]);
    setSelectedUserIds([]);
    try {
      const [cctvResult, userResult] = await Promise.allSettled([
        apiRequest<{ rows: CCTVItem[] }>('/cctv?page=1&limit=1000&sort_by=nama&order=asc'),
        apiRequest<{ rows: UserItem[] }>('/users?role=teknisi&page=1&limit=1000'),
      ]);

      if (cctvResult.status === 'fulfilled') {
        const rows = cctvResult.value.data?.rows || [];
        setCctvs(rows);
        // Preselect CCTV milik zona ini
        setSelectedCctvIds(rows.filter((c) => c.zone_id === zone.id).map((c) => c.id));
      }
      if (userResult.status === 'fulfilled') {
        const rows = userResult.value.data?.rows || [];
        setTechnicians(rows);
        // Preselect teknisi milik zona ini (dari detail zona)
        try {
          const detail = await apiRequest<ZoneRow>(`/zones/${zone.id}`);
          const memberIds = (detail.data?.users || []).map((u) => u.id);
          setSelectedUserIds(memberIds);
        } catch {
          // abaikan bila detail gagal
        }
      }
    } catch {
      toast.error('Daftar CCTV atau teknisi belum dapat dimuat.');
    }
  };

  const toggleId = (arr: string[], id: string, setter: (v: string[]) => void) => {
    setter(arr.includes(id) ? arr.filter((x) => x !== id) : [...arr, id]);
  };

  const saveAssignments = async () => {
    if (!assignZone) return;
    setActionLoading('assign');
    try {
      // Assign CCTV (replace seluruh penugasan zona)
      await apiRequest(`/zones/${assignZone.id}/assign-cctvs`, {
        method: 'POST',
        body: JSON.stringify({ cctv_ids: selectedCctvIds }),
      });
      // Assign teknisi (replace seluruh penugasan zona)
      await apiRequest(`/zones/${assignZone.id}/assign-users`, {
        method: 'POST',
        body: JSON.stringify({ user_ids: selectedUserIds }),
      });
      toast.success(`Penugasan zona "${assignZone.nama}" berhasil disimpan.`);
      setAssignZone(null);
      fetchZones();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Penugasan zona belum dapat disimpan.', err);
    } finally {
      setActionLoading(null);
    }
  };

  const openForm = (zone: Zone | null = null) => {
    setCurrent(zone);
    setFormData(zone ? { nama: zone.nama, deskripsi: zone.deskripsi || '' } : { nama: '', deskripsi: '' });
    setIsFormOpen(true);
  };

  const closeForm = () => {
    if (actionLoading === 'save') return;
    setIsFormOpen(false);
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.nama.trim()) {
      toast.error('Nama zona wajib diisi.');
      return;
    }
    setActionLoading('save');
    try {
      const endpoint = current ? `/zones/${current.id}` : '/zones';
      const method = current ? 'PUT' : 'POST';
      const response = await apiRequest<Zone>(endpoint, {
        method,
        body: JSON.stringify({ nama: formData.nama, deskripsi: formData.deskripsi }),
      });
      if (response.status === 'success') {
        setIsFormOpen(false);
        toast.success(current ? `Zona "${formData.nama}" berhasil diperbarui.` : `Zona "${formData.nama}" berhasil ditambahkan.`);
        fetchZones();
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Zona belum dapat disimpan. Coba lagi.', err);
    } finally {
      setActionLoading(null);
    }
  };

  const handleDelete = async (zone: Zone) => {
    const ok = await confirm({
      title: 'Hapus Zona?',
      message: `Hapus zona "${zone.nama}"? CCTV pada zona ini akan dilepas dari zona (data CCTV tetap tersimpan).`,
      confirmLabel: 'Hapus',
    });
    if (!ok) return;
    setActionLoading(`delete-${zone.id}`);
    try {
      const response = await apiRequest(`/zones/${zone.id}`, { method: 'DELETE' });
      if (response.status === 'success') {
        toast.success(`Zona "${zone.nama}" berhasil dihapus.`);
        fetchZones();
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Zona belum dapat dihapus. Coba lagi.', err);
    } finally {
      setActionLoading(null);
    }
  };

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-[-0.025em]">Zona Operasional</h1>
            <p className="mt-1 max-w-2xl text-sm text-muted">
              Kelompokkan CCTV dan teknisi ke dalam zona/wilayah untuk memudahkan penugasan dan pemantauan.
            </p>
          </div>
          <button
            type="button"
            onClick={() => openForm()}
            disabled={actionLoading !== null}
            className="flex min-h-11 items-center justify-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(15,118,74,0.18)] transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-60"
          >
            <Plus className="h-4 w-4" />
            Tambah Zona
          </button>
        </div>

        <section aria-labelledby="zone-list-heading" aria-busy={loading} className="overflow-hidden rounded-2xl border border-border bg-card">
          <div className="flex flex-col gap-3 border-b border-border px-5 py-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h2 id="zone-list-heading" className="font-semibold">Daftar Zona</h2>
              <p className="mt-0.5 text-xs text-muted">{pagination.total_rows} zona</p>
            </div>
            <div className="relative max-w-md">
              <Search className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" />
              <input
                type="search"
                aria-label="Cari zona"
                placeholder="Cari nama zona"
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
              <span className="text-sm text-muted">Memuat zona…</span>
            </div>
          ) : error ? (
            <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
              <AlertTriangle className="h-10 w-10 text-error" />
              <h3 className="mt-3 text-base font-semibold">Data zona belum dapat dimuat</h3>
              <p className="mt-1 max-w-md text-xs text-muted">{error}</p>
              <button
                type="button"
                onClick={fetchZones}
                className="mt-4 rounded-lg bg-primary px-4 py-2 text-xs font-bold text-white transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
              >
                Coba lagi
              </button>
            </div>
          ) : zones.length === 0 ? (
            <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
              <MapPinned className="h-10 w-10 text-muted/45" />
              <h3 className="mt-3 text-base font-semibold">{search ? 'Tidak ada zona yang cocok' : 'Belum ada zona'}</h3>
              <p className="mt-1 max-w-sm text-xs text-muted">
                {search ? 'Ubah kata kunci pencarian.' : 'Gunakan "Tambah Zona" untuk mengelompokkan CCTV & teknisi.'}
              </p>
            </div>
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="w-full min-w-[860px] border-collapse text-left">
                  <thead>
                    <tr className="border-b border-border bg-background/50 text-xs font-semibold text-muted">
                      <th className="px-5 py-3.5">
                        <SortableHeader label="Nama" sortKey="nama" activeKey={sortKey} direction={sortDirection} onSort={onSort} />
                      </th>
                      <th className="px-5 py-3.5">
                        <SortableHeader label="Deskripsi" sortKey="deskripsi" activeKey={sortKey} direction={sortDirection} onSort={onSort} />
                      </th>
                      <th className="px-5 py-3.5">
                        <SortableHeader label="CCTV" sortKey="cctv" activeKey={sortKey} direction={sortDirection} onSort={onSort} />
                      </th>
                      <th className="px-5 py-3.5">
                        <SortableHeader label="Teknisi" sortKey="teknisi" activeKey={sortKey} direction={sortDirection} onSort={onSort} />
                      </th>
                      <th className="px-5 py-3.5 text-right">Tindakan</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border text-sm">
                    {sortedRows.map((zone) => (
                      <tr key={zone.id} className="transition-colors hover:bg-background/40">
                        <td className="px-5 py-4">
                          <p className="font-semibold">{zone.nama}</p>
                        </td>
                        <td className="max-w-[300px] px-5 py-4 text-muted">
                          <span className="line-clamp-2">{zone.deskripsi || '-'}</span>
                        </td>
                        <td className="px-5 py-4">
                          <span className="inline-flex items-center gap-1.5 rounded-full border border-border bg-background px-2.5 py-1 text-xs font-semibold">
                            <Camera className="h-3.5 w-3.5 text-primary" />
                            {zone.cctv_count ?? 0}
                          </span>
                        </td>
                        <td className="px-5 py-4">
                          <span className="inline-flex items-center gap-1.5 rounded-full border border-border bg-background px-2.5 py-1 text-xs font-semibold">
                            <Users className="h-3.5 w-3.5 text-primary" />
                            {zone.user_count ?? 0}
                          </span>
                        </td>
                        <td className="px-5 py-4 text-right">
                          <div className="flex items-center justify-end gap-1">
                            <button
                              type="button"
                              onClick={() => openAssign(zone)}
                              disabled={actionLoading !== null}
                              className="rounded-lg px-2.5 py-2 text-xs font-bold text-primary transition-colors hover:bg-primary/10 disabled:opacity-50"
                              title="Atur penugasan CCTV & teknisi"
                            >
                              Atur Penugasan
                            </button>
                            <button
                              type="button"
                              onClick={() => openForm(zone)}
                              disabled={actionLoading !== null}
                              className="rounded-lg p-2.5 text-muted transition-colors hover:bg-primary/10 hover:text-primary disabled:opacity-50"
                              title="Ubah zona"
                              aria-label={`Ubah zona ${zone.nama}`}
                            >
                              <Edit2 className="h-4 w-4" />
                            </button>
                            <button
                              type="button"
                              onClick={() => handleDelete(zone)}
                              disabled={actionLoading !== null}
                              className="rounded-lg p-2.5 text-muted transition-colors hover:bg-error/10 hover:text-error disabled:opacity-50"
                              title="Hapus zona"
                              aria-label={`Hapus zona ${zone.nama}`}
                            >
                              {actionLoading === `delete-${zone.id}` ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
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
                itemLabel="zona"
              />
            </>
          )}
        </section>
      </div>

      {/* MODAL FORM TAMBAH/EDIT ZONA */}
      {isFormOpen && (
        <div className="fixed inset-0 flex items-center justify-center z-50 px-4">
          <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={closeForm}></div>
          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby="zone-form-title"
            className="z-10 max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl border border-border bg-card p-5 shadow-2xl sm:p-6"
          >
            <div className="mb-6 flex items-start justify-between gap-4">
              <div>
                <h3 id="zone-form-title" className="text-xl font-bold tracking-[-0.015em]">
                  {current ? 'Ubah Zona' : 'Tambah Zona'}
                </h3>
                <p className="mt-1 text-sm text-muted">Isi nama dan deskripsi zona operasional.</p>
              </div>
              <button type="button" aria-label="Tutup formulir zona" onClick={closeForm} disabled={actionLoading === 'save'} className="shrink-0 rounded-lg p-2.5 text-muted transition-colors hover:bg-background hover:text-foreground disabled:opacity-50">
                <X className="h-4 w-4" />
              </button>
            </div>

            <form onSubmit={handleSave} className="space-y-4">
              <div>
                <label htmlFor="zone-nama" className="mb-1.5 block text-xs font-semibold text-muted">Nama Zona *</label>
                <input
                  id="zone-nama"
                  autoFocus
                  type="text"
                  value={formData.nama}
                  onChange={(e) => setFormData({ ...formData, nama: e.target.value })}
                  placeholder="Contoh: Zona Utara"
                  className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                  required
                />
              </div>
              <div>
                <label htmlFor="zone-deskripsi" className="mb-1.5 block text-xs font-semibold text-muted">Deskripsi</label>
                <textarea
                  id="zone-deskripsi"
                  rows={3}
                  value={formData.deskripsi}
                  onChange={(e) => setFormData({ ...formData, deskripsi: e.target.value })}
                  placeholder="Cakupan wilayah / keterangan (opsional)"
                  className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                />
              </div>
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
                  {actionLoading === 'save' ? 'Menyimpan…' : current ? 'Simpan Perubahan' : 'Simpan Zona'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL ATUR PENUGASAN CCTV & TEKNISI */}
      {assignZone && (
        <div className="fixed inset-0 flex items-center justify-center z-50 px-4">
          <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => { if (actionLoading !== 'assign') setAssignZone(null); }}></div>
          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby="zone-assign-title"
            className="z-10 max-h-[90vh] w-full max-w-3xl overflow-y-auto rounded-2xl border border-border bg-card p-5 shadow-2xl sm:p-6"
          >
            <div className="mb-6 flex items-start justify-between gap-4">
              <div>
                <h3 id="zone-assign-title" className="text-xl font-bold tracking-[-0.015em]">Atur Penugasan — {assignZone.nama}</h3>
                <p className="mt-1 text-sm text-muted">Pilih CCTV dan teknisi yang masuk ke zona ini.</p>
              </div>
              <button type="button" aria-label="Tutup" onClick={() => { if (actionLoading !== 'assign') setAssignZone(null); }} disabled={actionLoading === 'assign'} className="shrink-0 rounded-lg p-2.5 text-muted transition-colors hover:bg-background hover:text-foreground disabled:opacity-50">
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="space-y-6">
              {/* CCTV */}
              <div>
                <div className="mb-2 flex items-center justify-between">
                  <h4 className="text-sm font-semibold flex items-center gap-2">
                    <Camera className="h-4 w-4 text-primary" /> CCTV ({selectedCctvIds.length} dipilih)
                  </h4>
                </div>
                <div className="max-h-60 overflow-y-auto rounded-xl border border-border">
                  {cctvs.length === 0 ? (
                    <div className="px-4 py-6 text-center text-xs text-muted">Tidak ada CCTV tersedia.</div>
                  ) : (
                    <ul className="divide-y divide-border text-sm">
                      {cctvs.map((cctv) => (
                        <li key={cctv.id}>
                          <label className="flex cursor-pointer items-center gap-3 px-4 py-2.5 hover:bg-background/40">
                            <input
                              type="checkbox"
                              checked={selectedCctvIds.includes(cctv.id)}
                              onChange={() => toggleId(selectedCctvIds, cctv.id, setSelectedCctvIds)}
                              className="h-4 w-4 rounded border-input-border text-primary focus:ring-primary/30"
                            />
                            <span className="font-mono text-xs font-semibold text-primary">{cctv.kode_cctv}</span>
                            <span className="flex-1 truncate">{cctv.nama}</span>
                            <span className="text-xs text-muted">{cctv.lokasi_area || '-'}</span>
                          </label>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              </div>

              {/* Teknisi */}
              <div>
                <div className="mb-2 flex items-center justify-between">
                  <h4 className="text-sm font-semibold flex items-center gap-2">
                    <Users className="h-4 w-4 text-primary" /> Teknisi ({selectedUserIds.length} dipilih)
                  </h4>
                </div>
                <div className="max-h-60 overflow-y-auto rounded-xl border border-border">
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
              </div>
            </div>

            <div className="flex justify-end gap-3 pt-4 border-t border-border mt-6">
              <button type="button" onClick={() => { if (actionLoading !== 'assign') setAssignZone(null); }} disabled={actionLoading === 'assign'} className="rounded-xl border border-border px-5 py-2.5 text-sm font-semibold transition-colors hover:bg-background disabled:opacity-50">
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

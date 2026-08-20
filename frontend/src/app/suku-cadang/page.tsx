'use client';

import React, { useCallback, useState, useEffect } from 'react';
import DashboardLayout from '@/components/DashboardLayout';
import Pagination from '@/components/Pagination';
import { apiRequest, getAllSitesForSelect } from '@/lib/api';
import { useWebSocket } from '@/context/WebSocketContext';
import { useToast } from '@/components/Toast';
import { useConfirm } from '@/components/ConfirmDialog';
import {
  Plus,
  Trash2,
  Package,
  AlertTriangle,
  Search,
  Edit2,
  X,
  LoaderCircle,
  Building2
} from 'lucide-react';
import SortableHeader from '@/components/SortableHeader';
import { useSortableRows } from '@/hooks/useSortableRows';

// Daftar kategori suku cadang yang konsisten dengan backend.
const KATEGORI = ['Kabel', 'Konektor', 'Power Supply', 'Lensa', 'Storage', 'Lainnya'];

interface SparePart {
  id: string;
  nama: string;
  sku: string;
  kategori: string;
  stok: number;
  minimum_order: number;
  satuan: string;
  harga_satuan: number;
  keterangan: string;
  created_at: string;
  site_nama?: string;
}

interface SparePartsResponse {
  rows: SparePart[];
  pagination: {
    total_rows: number;
    total_pages: number;
    page: number;
    limit: number;
  };
}

// Warna badge stok: hijau (sehat, stok > 2× min), kuning (perhatian, stok ≤ 2× min),
// merah (kritis, stok < min order).
function stockBadge(item: SparePart): { label: string; className: string } {
  if (item.stok < item.minimum_order) {
    return {
      label: 'Stok rendah',
      className: 'border-error/25 bg-error/10 text-error',
    };
  }
  if (item.stok <= item.minimum_order * 2) {
    return {
      label: 'Menipis',
      className: 'border-warning/25 bg-warning/10 text-warning',
    };
  }
  return {
    label: 'Cukup',
    className: 'border-success/25 bg-success/10 text-success',
  };
}

export default function SparePartsPage() {
  const toast = useToast();
  const confirm = useConfirm();

  const [items, setItems] = useState<SparePart[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [page, setPage] = useState(1);
  const [pagination, setPagination] = useState({
    total_rows: 0,
    total_pages: 1,
    page: 1,
    limit: 10,
  });

  // Search & filter kategori
  const [search, setSearch] = useState('');
  const [kategoriFilter, setKategoriFilter] = useState('');
  
  // Filter kawasan
  const [siteFilter, setSiteFilter] = useState('');
  const [sites, setSites] = useState<{ id: string; nama: string }[]>([]);

  // Modal State
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [currentItem, setCurrentItem] = useState<SparePart | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Form State
  const [formData, setFormData] = useState({
    nama: '',
    sku: '',
    kategori: '',
    stok: 0,
    minimum_order: 5,
    satuan: 'pcs',
    harga_satuan: 0,
    keterangan: '',
  });

  // Sortir client-side kolom header.
  const { sortedRows, sortKey, sortDirection, onSort } = useSortableRows<SparePart>(items, {
    getValue: (row, key) => {
      switch (key) {
        case 'sku':
          return row.sku;
        case 'nama':
          return row.nama;
        case 'kategori':
          return row.kategori;
        case 'stok':
          return row.stok;
        case 'minimum_order':
          return row.minimum_order;
        case 'satuan':
          return row.satuan;
        case 'harga':
          return row.harga_satuan;
        default:
          return '';
      }
    },
  });

  const fetchSpareParts = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const queryParams = [`page=${page}`, 'limit=10'];
      if (kategoriFilter) queryParams.push(`kategori=${encodeURIComponent(kategoriFilter)}`);
      if (search) queryParams.push(`search=${encodeURIComponent(search)}`);
      if (siteFilter) queryParams.push(`site_id=${encodeURIComponent(siteFilter)}`);

      const response = await apiRequest<SparePartsResponse>(`/spare-parts?${queryParams.join('&')}`);
      if (response.status === 'success' && response.data) {
        setItems(response.data.rows || []);
        setPagination(response.data.pagination || {
          total_rows: 0,
          total_pages: 1,
          page: 1,
          limit: 10,
        });
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Data suku cadang belum dapat dimuat. Coba lagi.');
    } finally {
      setLoading(false);
    }
  }, [kategoriFilter, page, search]);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      await Promise.resolve();
      if (!cancelled) await fetchSpareParts();
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, [fetchSpareParts]);

  // Muat ulang saat stok berubah (mis. event STOCK_LOW dari pencatatan penggunaan).
  const { subscribe } = useWebSocket();
  useEffect(() => {
    const unsub = subscribe('STOCK_LOW', () => fetchSpareParts());
    return () => {
      unsub();
    };
  }, [subscribe, fetchSpareParts]);

  // Load sites untuk dropdown filter kawasan
  useEffect(() => {
    let cancelled = false;
    const loadSites = async () => {
      try {
        const allSitesResponse = await getAllSitesForSelect();
        if (!cancelled && allSitesResponse.status === 'success' && allSitesResponse.data) {
          setSites(allSitesResponse.data);
        }
      } catch (err) {
        console.error('Gagal memuat list site:', err);
      }
    };
    loadSites();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!isModalOpen) return;
    const previousOverflow = document.body.style.overflow;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape' || actionLoading === 'save') return;
      setIsModalOpen(false);
    };
    document.body.style.overflow = 'hidden';
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [actionLoading, isModalOpen]);

  const handleOpenModal = (item: SparePart | null = null) => {
    if (item) {
      setCurrentItem(item);
      setFormData({
        nama: item.nama,
        sku: item.sku,
        kategori: item.kategori || '',
        stok: item.stok,
        minimum_order: item.minimum_order,
        satuan: item.satuan || 'pcs',
        harga_satuan: item.harga_satuan || 0,
        keterangan: item.keterangan || '',
      });
    } else {
      setCurrentItem(null);
      setFormData({
        nama: '',
        sku: '',
        kategori: '',
        stok: 0,
        minimum_order: 5,
        satuan: 'pcs',
        harga_satuan: 0,
        keterangan: '',
      });
    }
    setIsModalOpen(true);
  };

  const closeFormModal = () => {
    if (actionLoading === 'save') return;
    setIsModalOpen(false);
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.nama.trim()) {
      toast.error('Nama suku cadang wajib diisi.');
      return;
    }
    if (!formData.sku.trim()) {
      toast.error('SKU wajib diisi.');
      return;
    }

    setActionLoading('save');
    try {
      const endpoint = currentItem ? `/spare-parts/${currentItem.id}` : '/spare-parts';
      const method = currentItem ? 'PUT' : 'POST';

      const response = await apiRequest<SparePart>(endpoint, {
        method,
        body: JSON.stringify({
          nama: formData.nama,
          sku: formData.sku,
          kategori: formData.kategori,
          stok: formData.stok,
          minimum_order: formData.minimum_order,
          satuan: formData.satuan,
          harga_satuan: formData.harga_satuan,
          keterangan: formData.keterangan,
        }),
      });

      if (response.status === 'success') {
        setIsModalOpen(false);
        toast.success(currentItem ? `Suku cadang "${formData.nama}" berhasil diperbarui.` : `Suku cadang "${formData.nama}" berhasil ditambahkan.`);
        fetchSpareParts();
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Suku cadang belum dapat disimpan. Periksa isian lalu coba lagi.');
    } finally {
      setActionLoading(null);
    }
  };

  const handleDelete = async (item: SparePart) => {
    const ok = await confirm({
      title: 'Hapus Suku Cadang?',
      message: `Hapus suku cadang "${item.nama}" (${item.sku})? Riwayat penggunaan pada maintenance log tetap tersimpan.`,
      confirmLabel: 'Hapus',
    });
    if (!ok) return;
    setActionLoading(`delete-${item.id}`);
    try {
      const response = await apiRequest(`/spare-parts/${item.id}`, { method: 'DELETE' });
      if (response.status === 'success') {
        toast.success(`Suku cadang "${item.nama}" berhasil dihapus.`);
        fetchSpareParts();
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Suku cadang belum dapat dihapus. Coba lagi.');
    } finally {
      setActionLoading(null);
    }
  };

  const formatRupiah = (value: number) => {
    if (!value && value !== 0) return '-';
    return new Intl.NumberFormat('id-ID', {
      style: 'currency',
      currency: 'IDR',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value || 0);
  };

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-[-0.025em]">Suku Cadang</h1>
            <p className="mt-1 max-w-2xl text-sm text-muted">
              Kelola stok komponen (kabel, konektor, power supply, dll.) untuk perbaikan CCTV.
            </p>
          </div>
          <button
            type="button"
            onClick={() => handleOpenModal()}
            disabled={actionLoading !== null}
            className="flex min-h-11 items-center justify-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(15,118,74,0.18)] transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-60"
          >
            <Plus className="h-4 w-4" />
            Tambah Suku Cadang
          </button>
        </div>

        <section aria-labelledby="spare-list-heading" aria-busy={loading} className="overflow-hidden rounded-2xl border border-border bg-card">
          <div className="flex flex-col gap-3 border-b border-border px-5 py-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h2 id="spare-list-heading" className="font-semibold">Daftar Suku Cadang</h2>
              <p className="mt-0.5 text-xs text-muted">{pagination.total_rows} item</p>
            </div>
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
              <div className="relative max-w-md">
                <Search className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" />
                <input
                  type="search"
                  aria-label="Cari suku cadang"
                  placeholder="Cari nama atau SKU"
                  value={search}
                  onChange={(e) => {
                    setSearch(e.target.value);
                    setPage(1);
                  }}
                  className="w-full rounded-xl border border-input-border bg-input py-2.5 pl-10 pr-4 text-sm text-foreground outline-none transition-colors focus:border-primary focus:ring-2 focus:ring-primary/20"
                />
              </div>
              <select
                id="spare-kategori-filter"
                aria-label="Filter kategori"
                value={kategoriFilter}
                onChange={(e) => {
                  setKategoriFilter(e.target.value);
                  setPage(1);
                }}
                disabled={loading}
                className="min-h-11 rounded-xl border border-input-border bg-input px-3.5 py-2.5 text-sm text-foreground transition-colors focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30 disabled:cursor-not-allowed disabled:opacity-60"
              >
                <option value="">Semua kategori</option>
                {KATEGORI.map((k) => (
                  <option key={k} value={k}>{k}</option>
                ))}
              </select>
              
              <select
                id="spare-site-filter"
                aria-label="Filter kawasan"
                value={siteFilter}
                onChange={(e) => {
                  setSiteFilter(e.target.value);
                  setPage(1);
                }}
                disabled={loading}
                className="min-h-11 rounded-xl border border-input-border bg-input px-3.5 py-2.5 text-sm text-foreground transition-colors focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30 disabled:cursor-not-allowed disabled:opacity-60"
              >
                <option value="">Semua kawasan</option>
                {sites.map((site) => (
                  <option key={site.id} value={site.id}>{site.nama}</option>
                ))}
              </select>
            </div>
          </div>

          {loading ? (
            <div className="flex items-center justify-center gap-3 py-20" role="status">
              <LoaderCircle className="h-6 w-6 animate-spin text-primary" />
              <span className="text-sm text-muted">Memuat suku cadang…</span>
            </div>
          ) : error ? (
            <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
              <AlertTriangle className="h-10 w-10 text-error" />
              <h3 className="mt-3 text-base font-semibold">Data suku cadang belum dapat dimuat</h3>
              <p className="mt-1 max-w-md text-xs text-muted">{error}</p>
              <button
                type="button"
                onClick={fetchSpareParts}
                className="mt-4 rounded-lg bg-primary px-4 py-2 text-xs font-bold text-white transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-card"
              >
                Coba lagi
              </button>
            </div>
          ) : items.length === 0 ? (
            <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
              <Package className="h-10 w-10 text-muted/45" />
              <h3 className="mt-3 text-base font-semibold">
                {search || kategoriFilter ? 'Tidak ada suku cadang yang cocok' : 'Belum ada suku cadang'}
              </h3>
              <p className="mt-1 max-w-sm text-xs text-muted">
                {search || kategoriFilter
                  ? 'Ubah kata kunci atau filter kategori untuk melihat suku cadang lain.'
                  : 'Gunakan "Tambah Suku Cadang" untuk mencatat stok komponen.'}
              </p>
            </div>
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="w-full min-w-[980px] border-collapse text-left">
                  <thead>
                    <tr className="border-b border-border bg-background/50 text-xs font-semibold text-muted">
                      <th className="px-5 py-3.5"><SortableHeader label="SKU" sortKey="sku" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                      <th className="px-5 py-3.5"><SortableHeader label="Nama" sortKey="nama" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                      <th className="px-5 py-3.5"><SortableHeader label="Kategori" sortKey="kategori" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                      <th className="px-5 py-3.5 text-sm font-semibold text-muted">Kawasan</th>
                      <th className="px-5 py-3.5"><SortableHeader label="Stok" sortKey="stok" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                      <th className="px-5 py-3.5"><SortableHeader label="Min. Order" sortKey="minimum_order" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                      <th className="px-5 py-3.5"><SortableHeader label="Satuan" sortKey="satuan" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                      <th className="px-5 py-3.5"><SortableHeader label="Harga" sortKey="harga" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                      <th className="px-5 py-3.5 text-right">Tindakan</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border text-sm">
                    {sortedRows.map((item) => {
                      const badge = stockBadge(item);
                      return (
                        <tr key={item.id} className="transition-colors hover:bg-background/40">
                          <td className="px-5 py-4 font-mono text-xs font-semibold text-primary">{item.sku}</td>
                          <td className="px-5 py-4">
                            <p className="font-semibold">{item.nama}</p>
                            {item.keterangan && (
                              <p className="mt-0.5 max-w-[260px] truncate text-xs text-muted" title={item.keterangan}>
                                {item.keterangan}
                              </p>
                            )}
                          </td>
                          <td className="px-5 py-4 text-muted">{item.kategori || '-'}</td>
                          <td className="px-5 py-4">
                            <div className="flex items-center gap-2">
                              {item.site_nama ? (
                                <>
                                  <Building2 className="h-4 w-4 text-primary" />
                                  <span className="text-sm">{item.site_nama}</span>
                                </>
                              ) : (
                                <span className="text-sm italic text-muted">Belum ditentukan</span>
                              )}
                            </div>
                          </td>
                          <td className="px-5 py-4">
                            <div className="flex items-center gap-2">
                              <span className="font-bold tabular-nums">{item.stok}</span>
                              <span className={`inline-flex rounded-full border px-2 py-0.5 text-[11px] font-semibold ${badge.className}`}>
                                {badge.label}
                              </span>
                            </div>
                          </td>
                          <td className="px-5 py-4 text-muted">{item.minimum_order}</td>
                          <td className="px-5 py-4 text-muted">{item.satuan || 'pcs'}</td>
                          <td className="px-5 py-4 tabular-nums">{formatRupiah(item.harga_satuan)}</td>
                          <td className="px-5 py-4 text-right">
                            <div className="flex items-center justify-end gap-1">
                              <button
                                type="button"
                                onClick={() => handleOpenModal(item)}
                                disabled={actionLoading !== null}
                                className="rounded-lg p-2.5 text-muted transition-colors hover:bg-primary/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:cursor-not-allowed disabled:opacity-60"
                                title="Ubah suku cadang"
                                aria-label={`Ubah suku cadang ${item.nama}`}
                              >
                                <Edit2 className="h-4 w-4" />
                              </button>
                              <button
                                type="button"
                                onClick={() => handleDelete(item)}
                                disabled={actionLoading !== null}
                                className="rounded-lg p-2.5 text-muted transition-colors hover:bg-error/10 hover:text-error focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-error/40 disabled:cursor-wait disabled:opacity-60"
                                title="Hapus suku cadang"
                                aria-label={`Hapus suku cadang ${item.nama}`}
                              >
                                {actionLoading === `delete-${item.id}` ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                              </button>
                            </div>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
              <Pagination
                page={pagination.page}
                limit={pagination.limit}
                totalRows={pagination.total_rows}
                totalPages={pagination.total_pages}
                onPageChange={setPage}
                itemLabel="suku cadang"
              />
            </>
          )}
        </section>

        {/* MODAL FORM TAMBAH/EDIT SUKU CADANG */}
        {isModalOpen && (
          <div className="fixed inset-0 flex items-center justify-center z-50 px-4">
            <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={closeFormModal}></div>

            <div
              role="dialog"
              aria-modal="true"
              aria-labelledby="spare-form-title"
              aria-describedby="spare-form-description"
              className="z-10 max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl border border-border bg-card p-5 shadow-2xl sm:p-6"
            >
              <div className="mb-6 flex items-start justify-between gap-4">
                <div>
                  <h3 id="spare-form-title" className="text-xl font-bold tracking-[-0.015em]">
                    {currentItem ? 'Ubah Suku Cadang' : 'Tambah Suku Cadang'}
                  </h3>
                  <p id="spare-form-description" className="mt-1 text-sm text-muted">
                    {currentItem
                      ? 'Perbarui rincian stok dan harga suku cadang.'
                      : 'Catat komponen baru beserta stok awal dan ambang minimum.'}
                  </p>
                </div>
                <button type="button" aria-label="Tutup formulir suku cadang" onClick={closeFormModal} disabled={actionLoading === 'save'} className="shrink-0 rounded-lg p-2.5 text-muted transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:cursor-not-allowed disabled:opacity-50">
                  <X className="h-4 w-4" />
                </button>
              </div>

              <form onSubmit={handleSave} className="space-y-4">
                <div>
                  <label htmlFor="spare-nama" className="mb-1.5 block text-xs font-semibold text-muted">Nama *</label>
                  <input
                    id="spare-nama"
                    autoFocus
                    type="text"
                    value={formData.nama}
                    onChange={(e) => setFormData({ ...formData, nama: e.target.value })}
                    placeholder="Contoh: Kabel UTP Cat6 5m"
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    required
                  />
                </div>

                <div>
                  <label htmlFor="spare-sku" className="mb-1.5 block text-xs font-semibold text-muted">SKU *</label>
                  <input
                    id="spare-sku"
                    type="text"
                    value={formData.sku}
                    onChange={(e) => setFormData({ ...formData, sku: e.target.value })}
                    placeholder="Contoh: KBL-UTP-005"
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    required
                  />
                </div>

                <div>
                  <label htmlFor="spare-kategori" className="mb-1.5 block text-xs font-semibold text-muted">Kategori</label>
                  <select
                    id="spare-kategori"
                    value={formData.kategori}
                    onChange={(e) => setFormData({ ...formData, kategori: e.target.value })}
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                  >
                    <option value="">Pilih kategori</option>
                    {KATEGORI.map((k) => (
                      <option key={k} value={k}>{k}</option>
                    ))}
                  </select>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label htmlFor="spare-stok" className="mb-1.5 block text-xs font-semibold text-muted">Stok *</label>
                    <input
                      id="spare-stok"
                      type="number"
                      min={0}
                      value={formData.stok}
                      onChange={(e) => setFormData({ ...formData, stok: Number(e.target.value) || 0 })}
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                      required
                    />
                  </div>
                  <div>
                    <label htmlFor="spare-min" className="mb-1.5 block text-xs font-semibold text-muted">Minimum Order *</label>
                    <input
                      id="spare-min"
                      type="number"
                      min={0}
                      value={formData.minimum_order}
                      onChange={(e) => setFormData({ ...formData, minimum_order: Number(e.target.value) || 0 })}
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                      required
                    />
                  </div>
                  <div>
                    <label htmlFor="spare-satuan" className="mb-1.5 block text-xs font-semibold text-muted">Satuan</label>
                    <input
                      id="spare-satuan"
                      type="text"
                      value={formData.satuan}
                      onChange={(e) => setFormData({ ...formData, satuan: e.target.value })}
                      placeholder="pcs"
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    />
                  </div>
                  <div>
                    <label htmlFor="spare-harga" className="mb-1.5 block text-xs font-semibold text-muted">Harga Satuan (Rp)</label>
                    <input
                      id="spare-harga"
                      type="number"
                      min={0}
                      step="0.01"
                      value={formData.harga_satuan}
                      onChange={(e) => setFormData({ ...formData, harga_satuan: Number(e.target.value) || 0 })}
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    />
                  </div>
                </div>

                <div>
                  <label htmlFor="spare-keterangan" className="mb-1.5 block text-xs font-semibold text-muted">Keterangan</label>
                  <textarea
                    id="spare-keterangan"
                    rows={3}
                    value={formData.keterangan}
                    onChange={(e) => setFormData({ ...formData, keterangan: e.target.value })}
                    placeholder="Catatan tambahan (opsional)"
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                  />
                </div>

                <div className="flex justify-end gap-3 pt-4 border-t border-border mt-6">
                  <button
                    type="button"
                    onClick={closeFormModal}
                    disabled={actionLoading === 'save'}
                    className="rounded-xl border border-border px-5 py-2.5 text-sm font-semibold transition-colors hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    Batal
                  </button>
                  <button
                    type="submit"
                    disabled={actionLoading === 'save'}
                    className="flex items-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(15,118,74,0.16)] transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-card disabled:cursor-wait disabled:opacity-65"
                  >
                    {actionLoading === 'save' && <LoaderCircle className="h-4 w-4 animate-spin" />}
                    {actionLoading === 'save' ? 'Menyimpan…' : currentItem ? 'Simpan Perubahan' : 'Simpan Suku Cadang'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </DashboardLayout>
  );
}

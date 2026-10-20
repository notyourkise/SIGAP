'use client';

import React, { useCallback, useState, useEffect } from 'react';
import DashboardLayout from '@/components/DashboardLayout';
import Pagination from '@/components/Pagination';
import { apiRequest } from '@/lib/api';
import { useWebSocket } from '@/context/WebSocketContext';
import { useToast } from '@/components/Toast';
import { useConfirm } from '@/components/ConfirmDialog';
import { Plus, Trash2, CalendarClock, AlertTriangle, Settings, X, LoaderCircle, Power } from 'lucide-react';
import SortableHeader from '@/components/SortableHeader';
import { useSortableRows } from '@/hooks/useSortableRows';

interface MasterCCTV {
  id: string;
  nama: string;
  kode_cctv: string;
  lokasi_area?: string;
}

interface User {
  id: string;
  full_name: string;
}

type Frequency = 'daily' | 'weekly' | 'monthly';

interface MaintenanceSchedule {
  id: string;
  cctv_id: string;
  cctv: MasterCCTV | null;
  technician_id: string | null;
  // teknisi boleh nil saat jadwal belum menetapkan pelaksana
  technician: User | null;
  action_taken: string;
  frequency: Frequency;
  next_run_at: string;
  is_active: boolean;
  last_generated_at: string | null;
  created_at: string;
}

interface SchedulesResponse {
  rows: MaintenanceSchedule[];
  pagination: {
    total_rows: number;
    total_pages: number;
    page: number;
    limit: number;
  };
}

const FREQUENCY_LABELS: Record<Frequency, string> = {
  daily: 'Harian',
  weekly: 'Mingguan',
  monthly: 'Bulanan',
};

export default function MaintenanceSchedulesPage() {
  const toast = useToast();
  const confirm = useConfirm();

  const [schedules, setSchedules] = useState<MaintenanceSchedule[]>([]);
  const [cctvs, setCctvs] = useState<MasterCCTV[]>([]);
  const [technicians, setTechnicians] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [cctvsError, setCctvsError] = useState('');
  const [page, setPage] = useState(1);
  const [pagination, setPagination] = useState({
    total_rows: 0,
    total_pages: 1,
    page: 1,
    limit: 10,
  });

  // Filter status aktif/nonaktif
  const [isActiveFilter, setIsActiveFilter] = useState('');

  // Modal State
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [currentSchedule, setCurrentSchedule] = useState<MaintenanceSchedule | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Form State
  const [formData, setFormData] = useState({
    cctv_id: '',
    technician_id: '',
    action_taken: '',
    frequency: 'daily' as Frequency,
  });

  // Sortir client-side kolom header.
  const { sortedRows, sortKey, sortDirection, onSort } = useSortableRows<MaintenanceSchedule>(schedules, {
    getValue: (row, key) => {
      switch (key) {
        case 'cctv':
          return row.cctv?.nama ?? '';
        case 'pekerjaan':
          return row.action_taken;
        case 'teknisi':
          return row.technician?.full_name ?? '';
        case 'frekuensi':
          return row.frequency;
        case 'jalankan':
          return row.next_run_at;
        case 'status':
          return row.is_active ? 'Aktif' : 'Nonaktif';
        default:
          return '';
      }
    },
  });

  const fetchSchedules = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const queryParams = [`page=${page}`, 'limit=10'];
      if (isActiveFilter) queryParams.push(`is_active=${encodeURIComponent(isActiveFilter)}`);

      const response = await apiRequest<SchedulesResponse>(`/maintenance-schedules?${queryParams.join('&')}`);
      if (response.status === 'success' && response.data) {
        setSchedules(response.data.rows || []);
        setPagination(response.data.pagination || {
          total_rows: 0,
          total_pages: 1,
          page: 1,
          limit: 10,
        });
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Jadwal pemeliharaan belum dapat dimuat. Coba lagi.');
    } finally {
      setLoading(false);
    }
  }, [isActiveFilter, page]);

  const fetchCCTVsList = useCallback(async () => {
    setCctvsError('');
    try {
      const response = await apiRequest<MasterCCTV[] | { rows?: MasterCCTV[] }>('/cctv?page=1&limit=100');
      if (response.status === 'success' && response.data) {
        const rows: MasterCCTV[] = Array.isArray(response.data)
          ? response.data
          : (response.data.rows || []);
        setCctvs(rows);
      }
    } catch (err: unknown) {
      setCctvs([]);
      setCctvsError(
        err instanceof Error
          ? err.message
          : 'Daftar CCTV belum dapat dimuat. Muat ulang sebelum membuat jadwal.'
      );
    }
  }, []);

  // Daftar teknisi (role "teknisi") untuk dropdown opsional pelaksana
  const fetchTechnicians = useCallback(async () => {
    try {
      const response = await apiRequest<{ rows?: User[] }>('/users?page=1&limit=100&role=teknisi');
      if (response.status === 'success' && response.data) {
        setTechnicians(response.data.rows || []);
      }
    } catch {
      setTechnicians([]);
    }
  }, []);

  useEffect(() => {
    let cancelled = false;

    const loadSchedules = async () => {
      await Promise.resolve();
      if (!cancelled) await fetchSchedules();
    };

    void loadSchedules();
    return () => {
      cancelled = true;
    };
  }, [fetchSchedules]);

  useEffect(() => {
    let cancelled = false;

    const loadAssets = async () => {
      await Promise.resolve();
      if (!cancelled) await fetchCCTVsList();
    };

    void loadAssets();
    return () => {
      cancelled = true;
    };
  }, [fetchCCTVsList]);

  useEffect(() => {
    let cancelled = false;

    const loadTechnicians = async () => {
      await Promise.resolve();
      if (!cancelled) await fetchTechnicians();
    };

    void loadTechnicians();
    return () => {
      cancelled = true;
    };
  }, [fetchTechnicians]);

  const { subscribe } = useWebSocket();

  useEffect(() => {
    const unsubCreated = subscribe('MAINTENANCE_CREATED', () => fetchSchedules());

    return () => {
      unsubCreated();
    };
  }, [subscribe, fetchSchedules]);

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

  const handleOpenModal = (schedule: MaintenanceSchedule | null = null) => {
    if (schedule) {
      setCurrentSchedule(schedule);
      setFormData({
        cctv_id: schedule.cctv_id,
        technician_id: schedule.technician_id || '',
        action_taken: schedule.action_taken,
        frequency: schedule.frequency,
      });
    } else {
      setCurrentSchedule(null);
      setFormData({
        cctv_id: cctvs.length > 0 ? cctvs[0].id : '',
        technician_id: '',
        action_taken: '',
        frequency: 'daily',
      });
    }
    setIsModalOpen(true);
  };

  const handleSaveSchedule = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.cctv_id) {
      toast.error('Pilih CCTV yang dijadwalkan.');
      return;
    }
    if (!formData.action_taken.trim()) {
      toast.error('Tuliskan tindakan yang dilakukan.');
      return;
    }

    setActionLoading('save');
    try {
      const endpoint = currentSchedule ? `/maintenance-schedules/${currentSchedule.id}` : '/maintenance-schedules';
      const method = currentSchedule ? 'PUT' : 'POST';

      const response = await apiRequest<MaintenanceSchedule>(endpoint, {
        method,
        body: JSON.stringify({
          cctv_id: formData.cctv_id,
          technician_id: formData.technician_id,
          action_taken: formData.action_taken,
          frequency: formData.frequency,
        }),
      });

      if (response.status === 'success') {
        setIsModalOpen(false);
        toast.success(currentSchedule ? 'Jadwal pemeliharaan berhasil diperbarui.' : 'Jadwal pemeliharaan berhasil dibuat.');
        fetchSchedules();
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Jadwal pemeliharaan belum dapat disimpan. Periksa isian lalu coba lagi.', err);
    } finally {
      setActionLoading(null);
    }
  };

  const closeFormModal = () => {
    if (actionLoading === 'save') return;
    setIsModalOpen(false);
  };

  const handleToggleSchedule = async (schedule: MaintenanceSchedule) => {
    setActionLoading(`toggle-${schedule.id}`);
    try {
      const response = await apiRequest<MaintenanceSchedule>(`/maintenance-schedules/${schedule.id}/toggle`, {
        method: 'PUT',
      });
      if (response.status === 'success') {
        toast.success(
          response.data?.is_active
            ? 'Jadwal kembali aktif.'
            : 'Jadwal dihentikan sementara.'
        );
        fetchSchedules();
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Status jadwal belum dapat diubah. Coba lagi.', err);
    } finally {
      setActionLoading(null);
    }
  };

  const handleDeleteSchedule = async (schedule: MaintenanceSchedule) => {
    const ok = await confirm({
      title: 'Hapus Jadwal Pemeliharaan?',
      message: 'Hapus jadwal pemeliharaan berulang ini? Penghapusan tidak akan menghapus log pemeliharaan yang sudah dibuat.',
      confirmLabel: 'Hapus',
    });
    if (!ok) return;
    setActionLoading(`delete-${schedule.id}`);
    try {
      const response = await apiRequest(`/maintenance-schedules/${schedule.id}`, {
        method: 'DELETE'
      });
      if (response.status === 'success') {
        toast.success('Jadwal pemeliharaan berhasil dihapus.');
        fetchSchedules();
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Jadwal pemeliharaan belum dapat dihapus. Coba lagi.', err);
    } finally {
      setActionLoading(null);
    }
  };

  const formatDate = (dateStr: string | null) => {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    if (Number.isNaN(date.getTime())) return dateStr;

    return date.toLocaleString('id-ID', {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-[-0.025em]">Jadwal Pemeliharaan</h1>
            <p className="mt-1 max-w-2xl text-sm text-muted">Atur pemeliharaan berulang yang dijalankan otomatis oleh sistem.</p>
          </div>
          <button
            type="button"
            onClick={() => handleOpenModal()}
            disabled={actionLoading !== null}
            className="flex min-h-11 items-center justify-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(15,118,74,0.18)] transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-60"
          >
            <Plus className="h-4 w-4" />
            Buat Jadwal
          </button>
        </div>

        <section aria-labelledby="schedule-list-heading" aria-busy={loading} className="overflow-hidden rounded-2xl border border-border bg-card">
          <div className="flex items-center justify-between gap-4 border-b border-border px-5 py-4">
            <div>
              <h2 id="schedule-list-heading" className="font-semibold">Daftar Jadwal</h2>
              <p className="mt-0.5 text-xs text-muted">{pagination.total_rows} jadwal</p>
            </div>
            <select
              id="schedule-active-filter"
              aria-label="Filter status jadwal"
              value={isActiveFilter}
              onChange={(e) => {
                setIsActiveFilter(e.target.value);
                setPage(1);
              }}
              disabled={loading}
              className="min-h-11 rounded-xl border border-input-border bg-input px-3.5 py-2.5 text-sm text-foreground transition-colors focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30 disabled:cursor-not-allowed disabled:opacity-60"
            >
              <option value="">Semua status</option>
              <option value="true">Aktif</option>
              <option value="false">Nonaktif</option>
            </select>
          </div>

        {cctvsError && (
          <div
            role="alert"
            className="m-4 flex flex-col gap-3 rounded-xl border border-warning/25 bg-warning/10 px-4 py-3 text-sm sm:flex-row sm:items-center sm:justify-between"
          >
            <div className="flex items-start gap-3">
              <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-warning" />
              <div>
                <p className="font-semibold text-foreground">Daftar CCTV belum tersedia</p>
                <p className="mt-0.5 text-xs text-muted">{cctvsError}</p>
              </div>
            </div>
            <button
              type="button"
              onClick={fetchCCTVsList}
              className="shrink-0 rounded-lg border border-warning/30 bg-card px-3 py-2 text-xs font-bold text-foreground transition-colors hover:border-warning focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-warning/40"
            >
              Coba lagi
            </button>
          </div>
        )}

        {loading ? (
          <div className="flex items-center justify-center gap-3 py-20" role="status">
            <LoaderCircle className="h-6 w-6 animate-spin text-primary" />
            <span className="text-sm text-muted">Memuat jadwal pemeliharaan…</span>
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
            <AlertTriangle className="h-10 w-10 text-error" />
            <h3 className="mt-3 text-base font-semibold">Data jadwal belum dapat dimuat</h3>
            <p className="mt-1 max-w-md text-xs text-muted">{error}</p>
            <button
              type="button"
              onClick={fetchSchedules}
              className="mt-4 rounded-lg bg-primary px-4 py-2 text-xs font-bold text-white transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-card"
            >
              Coba lagi
            </button>
          </div>
        ) : schedules.length === 0 ? (
          <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
            <CalendarClock className="h-10 w-10 text-muted/45" />
            <h3 className="mt-3 text-base font-semibold">{isActiveFilter ? 'Tidak ada jadwal yang cocok' : 'Belum ada jadwal pemeliharaan'}</h3>
            <p className="mt-1 max-w-sm text-xs text-muted">
              {isActiveFilter ? 'Ubah filter status jadwal untuk melihat jadwal lain.' : 'Gunakan “Buat Jadwal” untuk menjadwalkan pemeliharaan berulang.'}
            </p>
          </div>
        ) : (
          <div>
            <div className="overflow-x-auto">
              <table className="w-full min-w-[880px] border-collapse text-left">
                <thead>
                  <tr className="border-b border-border bg-background/50 text-xs font-semibold text-muted">
                    <th className="px-5 py-3.5"><SortableHeader label="CCTV" sortKey="cctv" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                    <th className="px-5 py-3.5"><SortableHeader label="Pekerjaan" sortKey="pekerjaan" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                    <th className="px-5 py-3.5"><SortableHeader label="Teknisi" sortKey="teknisi" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                    <th className="px-5 py-3.5"><SortableHeader label="Frekuensi" sortKey="frekuensi" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                    <th className="px-5 py-3.5"><SortableHeader label="Jalankan Berikutnya" sortKey="jalankan" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                    <th className="px-5 py-3.5"><SortableHeader label="Status" sortKey="status" activeKey={sortKey} direction={sortDirection} onSort={onSort} /></th>
                    <th className="px-5 py-3.5 text-right">Tindakan</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border text-sm">
                  {sortedRows.map((schedule) => (
                    <tr key={schedule.id} className="transition-colors hover:bg-background/40">
                      <td className="px-5 py-4">
                        <p className="font-semibold">{schedule.cctv?.nama || 'CCTV tidak tersedia'}</p>
                        {schedule.cctv?.kode_cctv && <p className="mt-0.5 font-mono text-xs text-muted">{schedule.cctv.kode_cctv}</p>}
                      </td>

                      <td className="max-w-xs px-5 py-4">
                        <p className="line-clamp-1 font-medium" title={schedule.action_taken}>{schedule.action_taken}</p>
                      </td>

                      <td className="px-5 py-4 text-muted">{schedule.technician?.full_name || 'Langsung oleh admin'}</td>

                      <td className="px-5 py-4 text-muted">{FREQUENCY_LABELS[schedule.frequency]}</td>

                      <td className="whitespace-nowrap px-5 py-4 text-xs text-muted">{formatDate(schedule.next_run_at)}</td>

                      <td className="px-5 py-4">
                        <span className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-semibold ${
                            schedule.is_active
                            ? 'border-success/25 bg-success/10 text-success'
                            : 'border-border bg-background/60 text-muted'
                        }`}>
                          {schedule.is_active ? 'Aktif' : 'Nonaktif'}
                        </span>
                      </td>

                      <td className="px-5 py-4 text-right">
                        <div className="flex items-center justify-end gap-1">
                            <button
                              type="button"
                              onClick={() => handleToggleSchedule(schedule)}
                              disabled={actionLoading !== null}
                              className={`rounded-lg p-2.5 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:cursor-wait disabled:opacity-60 ${
                                schedule.is_active
                                  ? 'text-muted hover:bg-error/10 hover:text-error'
                                  : 'text-muted hover:bg-success/10 hover:text-success'
                              }`}
                              title={schedule.is_active ? 'Hentikan sementara jadwal' : 'Aktifkan kembali jadwal'}
                              aria-label={`${schedule.is_active ? 'Hentikan' : 'Aktifkan'} jadwal ${schedule.cctv?.nama || 'CCTV'}`}
                            >
                              {actionLoading === `toggle-${schedule.id}` ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Power className="h-4 w-4" />}
                            </button>
                            <button
                              type="button"
                              onClick={() => handleOpenModal(schedule)}
                              disabled={actionLoading !== null}
                              className="rounded-lg p-2.5 text-muted transition-colors hover:bg-primary/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:cursor-not-allowed disabled:opacity-60"
                              title="Ubah jadwal pemeliharaan"
                              aria-label={`Ubah jadwal pemeliharaan ${schedule.cctv?.nama || 'CCTV'}`}
                            >
                              <Settings className="h-4 w-4" />
                            </button>
                            <button
                                type="button"
                                onClick={() => handleDeleteSchedule(schedule)}
                                disabled={actionLoading !== null}
                                className="rounded-lg p-2.5 text-muted transition-colors hover:bg-error/10 hover:text-error focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-error/40 disabled:cursor-wait disabled:opacity-60"
                                title="Hapus jadwal pemeliharaan"
                                aria-label={`Hapus jadwal pemeliharaan ${schedule.cctv?.nama || 'CCTV'}`}
                              >
                                {actionLoading === `delete-${schedule.id}` ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
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
              itemLabel="jadwal"
            />
          </div>
        )}
        </section>

        {/* MODAL FORM TAMBAH/EDIT JADWAL */}
        {isModalOpen && (
          <div className="fixed inset-0 flex items-center justify-center z-50 px-4">
            <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={closeFormModal}></div>

            <div
              role="dialog"
              aria-modal="true"
              aria-labelledby="schedule-form-title"
              aria-describedby="schedule-form-description"
              className="z-10 max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl border border-border bg-card p-5 shadow-2xl sm:p-6"
            >
              <div className="mb-6 flex items-start justify-between gap-4">
                <div>
                  <h3 id="schedule-form-title" className="text-xl font-bold tracking-[-0.015em]">{currentSchedule ? 'Ubah Jadwal Pemeliharaan' : 'Jadwal Pemeliharaan Baru'}</h3>
                  <p id="schedule-form-description" className="mt-1 text-sm text-muted">
                    {currentSchedule
                      ? 'Perbarui rincian pekerjaan dan frekuensi penjadwalan.'
                      : 'Pilih CCTV, tulis pekerjaan, lalu tentukan frekuensi berulangnya.'}
                  </p>
                </div>
                <button type="button" aria-label="Tutup formulir jadwal" onClick={closeFormModal} disabled={actionLoading === 'save'} className="shrink-0 rounded-lg p-2.5 text-muted transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:cursor-not-allowed disabled:opacity-50">
                  <X className="h-4 w-4" />
                </button>
              </div>

              <form onSubmit={handleSaveSchedule} className="space-y-4">
                {/* Select CCTV */}
                <div>
                  <label htmlFor="schedule-cctv" className="mb-1.5 block text-xs font-semibold text-muted">CCTV *</label>
                  <select
                    id="schedule-cctv"
                    value={formData.cctv_id}
                    onChange={(e) => setFormData({...formData, cctv_id: e.target.value})}
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    required
                    disabled={!!currentSchedule}
                  >
                    <option value="" disabled>Pilih CCTV</option>
                    {cctvs.map((cctv) => (
                      <option key={cctv.id} value={cctv.id}>
                        {cctv.nama} ({cctv.kode_cctv})
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label htmlFor="schedule-action" className="mb-1.5 block text-xs font-semibold text-muted">Tindakan yang dilakukan *</label>
                  <textarea
                    id="schedule-action"
                    autoFocus
                    rows={3}
                    value={formData.action_taken}
                    onChange={(e) => setFormData({...formData, action_taken: e.target.value})}
                    placeholder="Contoh: pembersihan lensa dan penggantian konektor RJ45"
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    required
                  />
                </div>

                <div>
                  <label htmlFor="schedule-frequency" className="mb-1.5 block text-xs font-semibold text-muted">Frekuensi penjadwalan *</label>
                  <select
                    id="schedule-frequency"
                    value={formData.frequency}
                    onChange={(e) => setFormData({...formData, frequency: e.target.value as Frequency})}
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    required
                  >
                    <option value="daily">Harian</option>
                    <option value="weekly">Mingguan</option>
                    <option value="monthly">Bulanan</option>
                  </select>
                  <p className="mt-1.5 text-[11px] text-muted">
                    Sistem membuat log pemeliharaan otomatis satu kali dalam frekuensi terpilih untuk CCTV ini.
                  </p>
                </div>

                <div>
                  <label htmlFor="schedule-technician" className="mb-1.5 block text-xs font-semibold text-muted">Teknisi pelaksana</label>
                  <select
                    id="schedule-technician"
                    value={formData.technician_id}
                    onChange={(e) => setFormData({...formData, technician_id: e.target.value})}
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                  >
                    <option value=""></option>
                    {technicians.map((tech) => (
                      <option key={tech.id} value={tech.id}>{tech.full_name}</option>
                    ))}
                  </select>
                  <p className="mt-1.5 text-[11px] text-muted">Kosongkan agar sistem menugaskan oleh admin; teknisi akan menerima notifikasi.</p>
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
                    {actionLoading === 'save' ? 'Menyimpan jadwal…' : currentSchedule ? 'Simpan Perubahan' : 'Simpan Jadwal'}
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
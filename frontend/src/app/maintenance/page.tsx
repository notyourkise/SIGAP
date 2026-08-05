'use client';

import React, { useCallback, useState, useEffect } from 'react';
import DashboardLayout from '@/components/DashboardLayout';
import Pagination from '@/components/Pagination';
import { apiRequest, downloadApiFile } from '@/lib/api';
import { useAuth } from '@/context/AuthContext';
import { useWebSocket } from '@/context/WebSocketContext';
import AuthenticatedImage from '@/components/AuthenticatedImage';
import { useToast } from '@/components/Toast';
import { useConfirm } from '@/components/ConfirmDialog';
import { 
  Plus, 
  Trash2, 
  Wrench, 
  AlertTriangle,
  Settings,
  X,
  FileSpreadsheet,
  FileText,
  Eye,
  Image as ImageIcon,
  LoaderCircle,
  ChevronDown,
  UserPlus,
  CheckCheck,
  SlidersHorizontal,
  RotateCcw,
  ArrowLeftRight,
} from 'lucide-react';
import FieldMaintenanceModal from '@/components/FieldMaintenanceModal';

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

interface MaintenanceLog {
  id: string;
  cctv_id: string;
  cctv: (MasterCCTV & { lokasi_area?: string }) | null;
  technician_id: string | null;
  // technician null saat log belum ditugaskan ke teknisi
  technician: User | null;
  action_taken: string;
  notes: string;
  status: 'scheduled' | 'ongoing' | 'completed' | 'failed';
  scheduled_date: string | null;
  completed_at: string | null;
  image_url?: string;
  created_at: string;
}

interface MaintenanceLogsResponse {
  rows: MaintenanceLog[];
  pagination: {
    total_rows: number;
    total_pages: number;
    page: number;
    limit: number;
  };
}

// Waktu lokal "YYYY-MM-DD HH:MM:SS" (bukan UTC)
const nowLocalDateTime = () => {
  const date = new Date();
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
};

// "YYYY-MM-DD HH:MM:SS" → nilai input datetime-local "YYYY-MM-DDTHH:MM"
const toDatetimeLocalInput = (value: string) => (value ? `${value.replace(' ', 'T').slice(0, 16)}` : '');

// nilai input datetime-local "YYYY-MM-DDTHH:MM" → "YYYY-MM-DD HH:MM:SS"
const fromDatetimeLocalInput = (value: string) => (value ? `${value}:00`.replace('T', ' ') : '');

export default function MaintenanceLogsPage() {
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';
  const toast = useToast();
  const confirm = useConfirm();

  const [logs, setLogs] = useState<MaintenanceLog[]>([]);
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

  // Filter State
  const [statusFilter, setStatusFilter] = useState('');
  const [cctvFilter, setCctvFilter] = useState('');
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');

  // Sort State (disembunyikan — diatur lewat modal "Sortir")
  const [sortBy, setSortBy] = useState('created_at');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');

  // Modal Sortir & Filter
  const [isSortModalOpen, setIsSortModalOpen] = useState(false);
  const [draftSortBy, setDraftSortBy] = useState('created_at');
  const [draftSortOrder, setDraftSortOrder] = useState<'asc' | 'desc'>('desc');
  const [draftStatusFilter, setDraftStatusFilter] = useState('');
  const [draftCctvFilter, setDraftCctvFilter] = useState('');
  const [draftStartDate, setDraftStartDate] = useState('');
  const [draftEndDate, setDraftEndDate] = useState('');

  // Jumlah filter/sortir aktif — dipakai untuk badge pada tombol Sortir
  const activeFilterCount =
    (cctvFilter ? 1 : 0) +
    (statusFilter ? 1 : 0) +
    (startDate ? 1 : 0) +
    (endDate ? 1 : 0) +
    (sortBy !== 'created_at' || sortOrder !== 'desc' ? 1 : 0);

  // Modal State
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [currentLog, setCurrentLog] = useState<MaintenanceLog | null>(null);
  const [detailLog, setDetailLog] = useState<MaintenanceLog | null>(null);
  const [evidenceFile, setEvidenceFile] = useState<File | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [assignLog, setAssignLog] = useState<MaintenanceLog | null>(null);
  const [assignTechnicianId, setAssignTechnicianId] = useState('');
  const [completeLog, setCompleteLog] = useState<MaintenanceLog | null>(null);

  // Form State — status dikelola sistem (created -> scheduled -> ongoing -> completed/failed),
  // sehingga form hanya memuat data yang diisi admin saat menjadwalkan pemeliharaan.
  const [formData, setFormData] = useState({
    cctv_id: '',
    action_taken: '',
    notes: '',
    scheduled_date: nowLocalDateTime(),
    image_url: ''
  });

  const fetchLogs = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const queryParams = [`page=${page}`, 'limit=10'];
      if (statusFilter) queryParams.push(`status=${encodeURIComponent(statusFilter)}`);
      if (cctvFilter) queryParams.push(`cctv_id=${encodeURIComponent(cctvFilter)}`);
      if (startDate) queryParams.push(`start_date=${encodeURIComponent(startDate)}`);
      if (endDate) queryParams.push(`end_date=${encodeURIComponent(endDate)}`);
      if (sortBy) queryParams.push(`sort_by=${encodeURIComponent(sortBy)}`);
      if (sortOrder) queryParams.push(`order=${encodeURIComponent(sortOrder)}`);
      
      const response = await apiRequest<MaintenanceLogsResponse>(`/maintenance-logs?${queryParams.join('&')}`);
      if (response.status === 'success' && response.data) {
        setLogs(response.data.rows || []);
        setPagination(response.data.pagination || {
          total_rows: 0,
          total_pages: 1,
          page: 1,
          limit: 10,
        });
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Catatan pemeliharaan belum dapat dimuat. Coba lagi.');
    } finally {
      setLoading(false);
    }
  }, [cctvFilter, page, sortBy, sortOrder, statusFilter, startDate, endDate]);

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
          : 'Daftar CCTV belum dapat dimuat. Muat ulang sebelum mencatat pemeliharaan.'
      );
    }
  }, []);

  // Daftar teknisi (role "teknisi") untuk dropdown penugasan tugas — admin saja
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
    if (!isAdmin) return;
    let cancelled = false;

    const loadTechnicians = async () => {
      await Promise.resolve();
      if (!cancelled) await fetchTechnicians();
    };

    void loadTechnicians();
    return () => {
      cancelled = true;
    };
  }, [isAdmin, fetchTechnicians]);

  useEffect(() => {
    let cancelled = false;

    const loadLogs = async () => {
      await Promise.resolve();
      if (!cancelled) await fetchLogs();
    };

    void loadLogs();
    return () => {
      cancelled = true;
    };
  }, [fetchLogs]);

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

  const { subscribe } = useWebSocket();

  useEffect(() => {
    const unsubCreated = subscribe('MAINTENANCE_CREATED', () => fetchLogs());
    const unsubUpdated = subscribe('MAINTENANCE_UPDATED', () => fetchLogs());

    return () => {
      unsubCreated();
      unsubUpdated();
    };
  }, [subscribe, fetchLogs]);

  useEffect(() => {
    if (!isModalOpen && !detailLog && !isSortModalOpen) return;

    const previousOverflow = document.body.style.overflow;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape' || actionLoading === 'save') return;
      setIsModalOpen(false);
      setDetailLog(null);
      setIsSortModalOpen(false);
    };

    document.body.style.overflow = 'hidden';
    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [actionLoading, detailLog, isModalOpen, isSortModalOpen]);

  const handleOpenModal = (log: MaintenanceLog | null = null) => {
    if (log) {
      setCurrentLog(log);
      setFormData({
        cctv_id: log.cctv_id,
        action_taken: log.action_taken,
        notes: log.notes || '',
        scheduled_date: log.scheduled_date ? toDatetimeLocalInput(log.scheduled_date) : nowLocalDateTime(),
        image_url: log.image_url || ''
      });
    } else {
      setCurrentLog(null);
      setFormData({
        cctv_id: cctvs.length > 0 ? cctvs[0].id : '',
        action_taken: '',
        notes: '',
        scheduled_date: nowLocalDateTime(),
        image_url: ''
      });
    }
    setEvidenceFile(null);
    setIsModalOpen(true);
  };

  const handleSaveLog = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.cctv_id) {
      toast.error('Pilih CCTV yang dipelihara.');
      return;
    }

    setActionLoading('save');
    try {
      let imageUrl = formData.image_url;
      if (evidenceFile) {
        const uploadData = new FormData();
        uploadData.append('file', evidenceFile);
        const upload = await apiRequest<{ url: string }>('/upload', {
          method: 'POST',
          body: uploadData,
        });
        imageUrl = upload.data?.url || '';
      }
      const endpoint = currentLog ? `/maintenance-logs/${currentLog.id}` : '/maintenance-logs';
      const method = currentLog ? 'PUT' : 'POST';

      const response = await apiRequest<MaintenanceLog>(endpoint, {
        method,
        body: JSON.stringify({ ...formData, image_url: imageUrl }),
      });

      if (response.status === 'success') {
        setIsModalOpen(false);
        toast.success(currentLog ? 'Catatan pemeliharaan berhasil diperbarui.' : 'Catatan pemeliharaan berhasil disimpan.');
        fetchLogs();
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Catatan pemeliharaan belum dapat disimpan. Periksa isian lalu coba lagi.');
    } finally {
      setActionLoading(null);
    }
  };

  const handleViewLog = async (id: string) => {
    setActionLoading(`detail-${id}`);
    try {
      const response = await apiRequest<MaintenanceLog>(`/maintenance-logs/${id}`);
      setDetailLog(response.data || null);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Detail pemeliharaan belum dapat dimuat. Coba lagi.');
    } finally {
      setActionLoading(null);
    }
  };

  // Tugaskan/alihkan log ke teknisi — status otomatis menjadi "ongoing";
  // untuk log ongoing, teknisi lama dilepas dan diberi notifikasi (Fase 5.3)
  const handleAssignLog = async () => {
    if (!assignLog) return;
    if (!assignTechnicianId) {
      toast.error('Pilih teknisi tujuan terlebih dahulu.');
      return;
    }
    setActionLoading(`assign-${assignLog.id}`);
    try {
      const response = await apiRequest<MaintenanceLog>(`/maintenance-logs/${assignLog.id}/assign`, {
        method: 'PUT',
        body: JSON.stringify({ technician_id: assignTechnicianId }),
      });
      if (response.status === 'success') {
        toast.success(assignLog.status === 'ongoing' ? 'Tugas berhasil dialihkan ke teknisi lain.' : 'Tugas berhasil ditugaskan ke teknisi.');
        setAssignLog(null);
        setAssignTechnicianId('');
        fetchLogs();
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Tugas belum dapat dialihkan. Coba lagi.');
    } finally {
      setActionLoading(null);
    }
  };

  const canCompleteLog = (log: MaintenanceLog) =>
    log.status === 'ongoing' && (isAdmin || user?.id === log.technician_id);

  const handleExport = async () => {
    const params = new URLSearchParams();
    if (statusFilter) params.set('status', statusFilter);
    if (cctvFilter) params.set('cctv_id', cctvFilter);
    if (startDate) params.set('start_date', startDate);
    if (endDate) params.set('end_date', endDate);
    setActionLoading('export');
    try {
      await downloadApiFile(`/maintenance-logs/export?${params}`, 'log-pemeliharaan.csv');
      toast.success('Data log pemeliharaan berhasil diekspor ke CSV.');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Catatan pemeliharaan belum dapat diekspor. Coba lagi.');
    } finally {
      setActionLoading(null);
    }
  };

  const handleReportPDF = async () => {
    const params = new URLSearchParams();
    if (statusFilter) params.set('status', statusFilter);
    if (startDate) params.set('start_date', startDate);
    if (endDate) params.set('end_date', endDate);
    setActionLoading('pdf');
    try {
      await downloadApiFile(`/maintenance-logs/report-pdf?${params}`, 'laporan-pemeliharaan.pdf');
      toast.success('Laporan PDF berhasil dibuat.');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Laporan PDF belum dapat dibuat. Coba lagi.');
    } finally {
      setActionLoading(null);
    }
  };

  const handleDeleteLog = async (id: string) => {
    const ok = await confirm({
      title: 'Hapus Catatan Pemeliharaan?',
      message: 'Hapus catatan pemeliharaan ini? Catatan tidak akan ditampilkan lagi.',
      confirmLabel: 'Hapus',
    });
    if (!ok) return;
    setActionLoading(`delete-${id}`);
    try {
      const response = await apiRequest(`/maintenance-logs/${id}`, {
        method: 'DELETE'
      });
      if (response.status === 'success') {
        toast.success('Catatan pemeliharaan berhasil dihapus.');
        fetchLogs();
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Catatan pemeliharaan belum dapat dihapus. Coba lagi.');
    } finally {
      setActionLoading(null);
    }
  };

  // Helper date formatter
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

  const getStatusLabel = (status: MaintenanceLog['status']) => {
    if (status === 'completed') return 'Selesai';
    if (status === 'failed') return 'Perlu Tindak Lanjut';
    if (status === 'ongoing') return 'Dalam proses';
    return 'Terjadwal';
  };

  const closeFormModal = () => {
    if (actionLoading === 'save') return;
    setIsModalOpen(false);
  };

  const openSortModal = () => {
    setDraftSortBy(sortBy);
    setDraftSortOrder(sortOrder);
    setDraftStatusFilter(statusFilter);
    setDraftCctvFilter(cctvFilter);
    setDraftStartDate(startDate);
    setDraftEndDate(endDate);
    setIsSortModalOpen(true);
  };

  const applySortFilter = () => {
    let changed = false;
    if (draftSortBy !== sortBy) { setSortBy(draftSortBy); changed = true; }
    if (draftSortOrder !== sortOrder) { setSortOrder(draftSortOrder); changed = true; }
    if (draftStatusFilter !== statusFilter) { setStatusFilter(draftStatusFilter); changed = true; }
    if (draftCctvFilter !== cctvFilter) { setCctvFilter(draftCctvFilter); changed = true; }
    if (draftStartDate !== startDate) { setStartDate(draftStartDate); changed = true; }
    if (draftEndDate !== endDate) { setEndDate(draftEndDate); changed = true; }
    if (changed) setPage(1);
    setIsSortModalOpen(false);
  };

  const resetSortFilter = () => {
    setSortBy('created_at');
    setSortOrder('desc');
    setStatusFilter('');
    setCctvFilter('');
    setStartDate('');
    setEndDate('');
    setPage(1);
    setIsSortModalOpen(false);
  };

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-[-0.025em]">Pemeliharaan CCTV</h1>
            <p className="mt-1 max-w-2xl text-sm text-muted">Catat dan telusuri pekerjaan pemeliharaan CCTV.</p>
          </div>
          <div className="flex w-full flex-col gap-2 sm:w-auto sm:flex-row">
            <button type="button" title="Laporan PDF" aria-label="Unduh laporan PDF" disabled={actionLoading !== null} onClick={handleReportPDF} className="flex min-h-11 w-11 items-center justify-center rounded-xl border border-border bg-card text-red-500 transition-colors hover:border-red-500/40 hover:bg-red-500/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500/40 focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-wait disabled:opacity-60">
              {actionLoading === 'pdf' ? <LoaderCircle className="h-5 w-5 animate-spin" /> : <FileText className="h-5 w-5" />}
            </button>
            <button type="button" title="Ekspor CSV" aria-label="Ekspor data ke CSV" disabled={actionLoading !== null} onClick={handleExport} className="flex min-h-11 w-11 items-center justify-center rounded-xl border border-border bg-card text-emerald-600 transition-colors hover:border-emerald-600/40 hover:bg-emerald-600/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600/40 focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-wait disabled:opacity-60">
              {actionLoading === 'export' ? <LoaderCircle className="h-5 w-5 animate-spin" /> : <FileSpreadsheet className="h-5 w-5" />}
            </button>
            {isAdmin && (
              <button
                type="button"
                onClick={() => handleOpenModal()}
                disabled={actionLoading !== null}
                className="flex min-h-11 items-center justify-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(15,118,74,0.18)] transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-60"
              >
                <Plus className="h-4 w-4" />
                Catat Pemeliharaan
              </button>
            )}
          </div>
        </div>

        <section aria-labelledby="maintenance-list-heading" aria-busy={loading} className="overflow-hidden rounded-2xl border border-border bg-card">
          <div className="flex items-center justify-between gap-4 border-b border-border px-5 py-4">
            <div>
              <h2 id="maintenance-list-heading" className="font-semibold">Riwayat Pemeliharaan</h2>
              <p className="mt-0.5 text-xs text-muted">{pagination.total_rows} catatan</p>
            </div>
            <button
              type="button"
              onClick={openSortModal}
              disabled={loading}
              className="relative inline-flex min-h-11 items-center gap-2 rounded-xl border border-border bg-background px-4 py-2.5 text-sm font-semibold text-foreground transition-colors hover:border-primary/40 hover:bg-primary/5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:cursor-not-allowed disabled:opacity-60"
            >
              <SlidersHorizontal className="h-4 w-4" />
              Sortir
              {activeFilterCount > 0 && (
                <span className="absolute -right-1.5 -top-1.5 inline-flex min-w-5 items-center justify-center rounded-full bg-primary px-1 text-[11px] font-bold leading-5 text-white">
                  {activeFilterCount}
                </span>
              )}
            </button>
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
            <span className="text-sm text-muted">Memuat catatan pemeliharaan…</span>
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
            <AlertTriangle className="h-10 w-10 text-error" />
            <h3 className="mt-3 text-base font-semibold">Data pemeliharaan belum dapat dimuat</h3>
            <p className="mt-1 max-w-md text-xs text-muted">{error}</p>
            <button
              type="button"
              onClick={fetchLogs}
              className="mt-4 rounded-lg bg-primary px-4 py-2 text-xs font-bold text-white transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-card"
            >
              Coba lagi
            </button>
          </div>
        ) : logs.length === 0 ? (
          <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
            <Wrench className="h-10 w-10 text-muted/45" />
            <h3 className="mt-3 text-base font-semibold">{statusFilter || cctvFilter ? 'Tidak ada catatan yang cocok' : 'Belum ada catatan pemeliharaan'}</h3>
            <p className="mt-1 max-w-sm text-xs text-muted">
              {statusFilter || cctvFilter ? 'Ubah filter CCTV atau status untuk melihat catatan lain.' : 'Gunakan “Catat Pemeliharaan” untuk membuat catatan pertama.'}
            </p>
          </div>
        ) : (
          <div>
            <div className="overflow-x-auto">
              <table className="w-full min-w-[900px] border-collapse text-left">
                <thead>
                  <tr className="border-b border-border bg-background/50 text-xs font-semibold text-muted">
                    <th className="px-5 py-3.5">CCTV</th>
                    <th className="px-5 py-3.5">Pekerjaan</th>
                    <th className="px-5 py-3.5">Teknisi</th>
                    <th className="px-5 py-3.5">Waktu</th>
                    <th className="px-5 py-3.5">Status</th>
                    <th className="px-5 py-3.5 text-right">Tindakan</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border text-sm">
                  {logs.map((log) => (
                    <tr key={log.id} className="transition-colors hover:bg-background/40">
                      <td className="px-5 py-4">
                        <p className="font-semibold">{log.cctv?.nama || 'CCTV tidak tersedia'}</p>
                        {log.cctv?.kode_cctv && <p className="mt-0.5 font-mono text-xs text-muted">{log.cctv.kode_cctv}</p>}
                      </td>
                      
                      <td className="max-w-xs px-5 py-4">
                        <p className="line-clamp-1 font-medium" title={log.action_taken}>{log.action_taken}</p>
                        {log.notes && <p className="mt-0.5 line-clamp-1 text-xs text-muted" title={log.notes}>{log.notes}</p>}
                      </td>

                      <td className="px-5 py-4 text-muted">{log.technician?.full_name || 'Belum ditugaskan'}</td>

                      <td className="whitespace-nowrap px-5 py-4 text-xs text-muted">{formatDate(log.completed_at || log.scheduled_date)}</td>

                      <td className="px-5 py-4">
                        <span className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-semibold ${
                            log.status === 'completed'
                            ? 'border-success/25 bg-success/10 text-success'
                            : log.status === 'failed'
                            ? 'border-error/25 bg-error/10 text-error'
                            : log.status === 'ongoing'
                            ? 'border-warning/25 bg-warning/10 text-warning'
                            : 'border-blue-500/25 bg-blue-500/10 text-blue-600 dark:text-blue-400'
                        }`}>
                          {getStatusLabel(log.status)}
                        </span>
                      </td>

                      <td className="px-5 py-4 text-right">
                        <div className="flex items-center justify-end gap-1">
                            <button
                              type="button"
                              onClick={() => handleViewLog(log.id)}
                              disabled={actionLoading !== null}
                              className="rounded-lg p-2.5 text-muted transition-colors hover:bg-primary/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:cursor-wait disabled:opacity-60"
                              title="Lihat detail pemeliharaan"
                              aria-label={`Lihat detail pemeliharaan ${log.cctv?.nama || 'CCTV'}`}
                            >
                              {actionLoading === `detail-${log.id}` ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Eye className="h-4 w-4" />}
                            </button>
                            {isAdmin && (log.status === 'scheduled' || log.status === 'ongoing') && (
                              <button
                                type="button"
                                onClick={() => {
                                  setAssignLog(log);
                                  setAssignTechnicianId('');
                                }}
                                disabled={actionLoading !== null}
                                className="rounded-lg p-2.5 text-muted transition-colors hover:bg-primary/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:cursor-not-allowed disabled:opacity-50"
                                title={log.status === 'ongoing' ? 'Alihkan ke teknisi lain' : 'Tugaskan ke teknisi'}
                                aria-label={`${log.status === 'ongoing' ? 'Alihkan pemeliharaan' : 'Tugaskan pemeliharaan'} ${log.cctv?.nama || 'CCTV'} ke teknisi lain`}
                              >
                                {actionLoading === `assign-${log.id}` ? <LoaderCircle className="h-4 w-4 animate-spin" /> : log.status === 'ongoing' ? <ArrowLeftRight className="h-4 w-4" /> : <UserPlus className="h-4 w-4" />}
                              </button>
                            )}
                            {canCompleteLog(log) && log.cctv && (
                              <button
                                type="button"
                                onClick={() => setCompleteLog(log)}
                                disabled={actionLoading !== null}
                                className="rounded-lg p-2.5 text-muted transition-colors hover:bg-emerald-500/10 hover:text-emerald-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/40 disabled:cursor-not-allowed disabled:opacity-50"
                                title="Selesaikan tugas lewat checklist 9 tahap"
                                aria-label={`Selesaikan tugas pemeliharaan ${log.cctv?.nama || 'CCTV'}`}
                              >
                                <CheckCheck className="h-4 w-4" />
                              </button>
                            )}
                            {isAdmin && (
                              <button
                                type="button"
                                onClick={() => handleOpenModal(log)}
                                disabled={actionLoading !== null}
                                className="rounded-lg p-2.5 text-muted transition-colors hover:bg-primary/10 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:cursor-not-allowed disabled:opacity-50"
                                title="Ubah catatan pemeliharaan"
                                aria-label={`Ubah catatan pemeliharaan ${log.cctv?.nama || 'CCTV'}`}
                              >
                                <Settings className="h-4 w-4" />
                              </button>
                            )}
                            {isAdmin && <button
                                type="button"
                                onClick={() => handleDeleteLog(log.id)}
                                disabled={actionLoading !== null}
                                className="rounded-lg p-2.5 text-muted transition-colors hover:bg-error/10 hover:text-error focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-error/40 disabled:cursor-wait disabled:opacity-60"
                                title="Hapus catatan pemeliharaan"
                                aria-label={`Hapus catatan pemeliharaan ${log.cctv?.nama || 'CCTV'}`}
                              >
                                {actionLoading === `delete-${log.id}` ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                              </button>}
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
              itemLabel="catatan"
            />
          </div>
        )}
        </section>

        {/* MODAL SORTIR & FILTER */}
        {isSortModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center px-4">
            <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => setIsSortModalOpen(false)} />

            <div
              role="dialog"
              aria-modal="true"
              aria-labelledby="maintenance-sort-title"
              className="z-10 max-h-[90vh] w-full max-w-md overflow-y-auto rounded-2xl border border-border bg-card p-5 shadow-2xl sm:p-6"
            >
              <div className="mb-5 flex items-start justify-between gap-4">
                <div>
                  <h3 id="maintenance-sort-title" className="text-xl font-bold tracking-[-0.015em]">Sortir &amp; Filter</h3>
                  <p className="mt-1 text-sm text-muted">Urutkan dan saring riwayat pemeliharaan.</p>
                </div>
                <button
                  type="button"
                  aria-label="Tutup sortir dan filter"
                  onClick={() => setIsSortModalOpen(false)}
                  className="shrink-0 rounded-lg p-2.5 text-muted transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>

              <div className="space-y-4">
                <div>
                  <label htmlFor="maintenance-sort-by" className="mb-1.5 block text-xs font-semibold text-muted">Urutkan berdasarkan</label>
                  <select
                    id="maintenance-sort-by"
                    value={draftSortBy}
                    onChange={(e) => setDraftSortBy(e.target.value)}
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                  >
                    <option value="created_at">Waktu pencatatan</option>
                    <option value="scheduled_date">Waktu terjadwal</option>
                    <option value="completed_at">Waktu selesai</option>
                    <option value="status">Status</option>
                    <option value="action_taken">Pekerjaan (A-Z)</option>
                  </select>
                </div>

                <div>
                  <label htmlFor="maintenance-sort-order" className="mb-1.5 block text-xs font-semibold text-muted">Arah pengurutan</label>
                  <select
                    id="maintenance-sort-order"
                    value={draftSortOrder}
                    onChange={(e) => setDraftSortOrder(e.target.value as 'asc' | 'desc')}
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                  >
                    <option value="desc">Terbaru ke terlama</option>
                    <option value="asc">Terlama ke terbaru</option>
                  </select>
                </div>

                <div className="space-y-4 border-t border-border pt-4">
                  <p className="text-xs font-semibold text-muted">Filter</p>
                  <div>
                    <label htmlFor="maintenance-sort-cctv-filter" className="mb-1.5 block text-xs font-semibold text-muted">CCTV</label>
                    <select
                      id="maintenance-sort-cctv-filter"
                      value={draftCctvFilter}
                      onChange={(e) => setDraftCctvFilter(e.target.value)}
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    >
                      <option value="">Semua CCTV</option>
                      {cctvs.map((cctv) => (
                        <option key={cctv.id} value={cctv.id}>{cctv.nama}</option>
                      ))}
                    </select>
                  </div>

                  <div>
                    <label htmlFor="maintenance-sort-status-filter" className="mb-1.5 block text-xs font-semibold text-muted">Status</label>
                    <select
                      id="maintenance-sort-status-filter"
                      value={draftStatusFilter}
                      onChange={(e) => setDraftStatusFilter(e.target.value)}
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    >
                      <option value="">Semua status</option>
                      <option value="completed">Selesai</option>
                      <option value="scheduled">Terjadwal</option>
                      <option value="ongoing">Dalam proses</option>
                      <option value="failed">Perlu Tindak Lanjut</option>
                    </select>
                  </div>

                  <div>
                    <label htmlFor="maintenance-sort-date-filter" className="mb-1.5 block text-xs font-semibold text-muted">Rentang tanggal pencatatan</label>
                    <div className="flex items-center gap-2">
                      <input
                        id="maintenance-sort-date-filter"
                        type="date"
                        aria-label="Rentang tanggal mulai"
                        value={draftStartDate}
                        onChange={(e) => setDraftStartDate(e.target.value)}
                        className="min-h-11 w-full rounded-xl border border-input-border bg-input px-3.5 py-2.5 text-sm text-foreground transition-colors focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
                      />
                      <span className="text-xs text-muted">s.d.</span>
                      <input
                        type="date"
                        aria-label="Rentang tanggal akhir"
                        value={draftEndDate}
                        onChange={(e) => setDraftEndDate(e.target.value)}
                        className="min-h-11 w-full rounded-xl border border-input-border bg-input px-3.5 py-2.5 text-sm text-foreground transition-colors focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
                      />
                    </div>
                  </div>
                </div>
              </div>

              <div className="mt-6 flex flex-col-reverse gap-2 border-t border-border pt-4 sm:flex-row sm:justify-between">
                <button
                  type="button"
                  onClick={resetSortFilter}
                  className="inline-flex items-center justify-center gap-2 rounded-xl border border-border px-4 py-2.5 text-sm font-semibold text-muted transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                >
                  <RotateCcw className="h-4 w-4" />
                  Reset
                </button>
                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={() => setIsSortModalOpen(false)}
                    className="rounded-xl border border-border px-5 py-2.5 text-sm font-semibold transition-colors hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                  >
                    Batal
                  </button>
                  <button
                    type="button"
                    onClick={applySortFilter}
                    className="rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(15,118,74,0.16)] transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-card"
                  >
                    Terapkan
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* MODAL FORM TAMBAH/EDIT LOG PEMELIHARAAN */}
        {isModalOpen && (
          <div className="fixed inset-0 flex items-center justify-center z-50 px-4">
            <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={closeFormModal}></div>
            
            <div
              role="dialog"
              aria-modal="true"
              aria-labelledby="maintenance-form-title"
              aria-describedby="maintenance-form-description"
              className="z-10 max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl border border-border bg-card p-5 shadow-2xl sm:p-6"
            >
              <div className="mb-6 flex items-start justify-between gap-4">
                <div>
                  <h3 id="maintenance-form-title" className="text-xl font-bold tracking-[-0.015em]">{currentLog ? 'Ubah Catatan Pemeliharaan' : 'Catat Pemeliharaan Baru'}</h3>
                  <p id="maintenance-form-description" className="mt-1 text-sm text-muted">
                    {currentLog
                      ? 'Perbarui jadwal dan rincian pekerjaan pemeliharaan.'
                      : 'Pilih CCTV, catat pekerjaan, lalu jadwalkan waktunya.'}
                  </p>
                </div>
                <button type="button" aria-label="Tutup formulir pemeliharaan" onClick={closeFormModal} disabled={actionLoading === 'save'} className="shrink-0 rounded-lg p-2.5 text-muted transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:cursor-not-allowed disabled:opacity-50">
                  <X className="h-4 w-4" />
                </button>
              </div>

              <form onSubmit={handleSaveLog} className="space-y-4">
                {/* Select CCTV */}
                <div>
                  <label htmlFor="maintenance-cctv" className="mb-1.5 block text-xs font-semibold text-muted">CCTV *</label>
                  <select
                    id="maintenance-cctv"
                    value={formData.cctv_id}
                    onChange={(e) => setFormData({...formData, cctv_id: e.target.value})}
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    required
                    disabled={!!currentLog}
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
                  <label htmlFor="maintenance-action" className="mb-1.5 block text-xs font-semibold text-muted">Tindakan yang dilakukan *</label>
                  <textarea
                    id="maintenance-action"
                    autoFocus
                    rows={3}
                    value={formData.action_taken}
                    onChange={(e) => setFormData({...formData, action_taken: e.target.value})}
                    placeholder="Contoh: pembersihan lensa dan penggantian konektor RJ45"
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    required
                  />
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label htmlFor="maintenance-status" className="mb-1.5 block text-xs font-semibold text-muted">Status pemeliharaan</label>
                    <div id="maintenance-status" className="flex min-h-11 items-center gap-2 rounded-xl border border-input-border bg-input px-4 py-2.5 text-sm text-foreground">
                      <span className={`inline-flex shrink-0 rounded-full border px-2.5 py-0.5 text-xs font-semibold ${
                        currentLog?.status === 'completed'
                          ? 'border-success/25 bg-success/10 text-success'
                          : currentLog?.status === 'failed'
                          ? 'border-error/25 bg-error/10 text-error'
                          : currentLog?.status === 'ongoing'
                          ? 'border-warning/25 bg-warning/10 text-warning'
                          : 'border-blue-500/25 bg-blue-500/10 text-blue-600 dark:text-blue-400'
                      }`}>
                        {currentLog ? getStatusLabel(currentLog.status) : 'Terjadwal'}
                      </span>
                      <span className="text-xs text-muted">Status diperbarui otomatis oleh sistem.</span>
                    </div>
                  </div>

                  <div>
                    <label htmlFor="maintenance-time" className="mb-1.5 block text-xs font-semibold text-muted">Waktu terjadwal *</label>
                    <input
                      id="maintenance-time"
                      type="datetime-local"
                      value={toDatetimeLocalInput(formData.scheduled_date)}
                      onChange={(e) => setFormData({ ...formData, scheduled_date: fromDatetimeLocalInput(e.target.value) })}
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                      required
                    />
                    <p className="mt-1.5 text-[11px] text-muted">Tugas akan ditugaskan ke teknisi oleh admin sebelum dikerjakan.</p>
                  </div>
                </div>

                <details className="group rounded-xl border border-border" open={currentLog ? true : undefined}>
                  <summary className="flex cursor-pointer list-none items-center justify-between gap-4 rounded-xl px-4 py-3 text-sm font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40">
                    <span>
                      Catatan dan bukti
                      <span className="ml-2 text-xs font-normal text-muted">Opsional</span>
                    </span>
                    <ChevronDown className="h-4 w-4 text-muted transition-transform group-open:rotate-180" />
                  </summary>
                  <div className="space-y-4 border-t border-border px-4 py-4">
                    <div>
                      <label htmlFor="maintenance-notes" className="mb-1.5 block text-xs font-semibold text-muted">Catatan tambahan</label>
                      <textarea
                        id="maintenance-notes"
                        rows={2}
                        value={formData.notes}
                        onChange={(e) => setFormData({...formData, notes: e.target.value})}
                        placeholder="Tambahkan hasil pemeriksaan atau temuan teknis"
                        className="w-full rounded-xl border border-input-border bg-input px-4 py-2.5 text-sm text-foreground transition-colors focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
                      />
                    </div>
                    <div>
                      <p className="mb-1.5 text-xs font-semibold text-muted">Foto bukti</p>
                      <label className="flex min-h-11 cursor-pointer items-center gap-3 rounded-xl border border-dashed border-input-border bg-input px-4 py-3 text-sm text-muted transition-colors hover:border-primary focus-within:border-primary focus-within:ring-2 focus-within:ring-primary/30">
                        <ImageIcon className="h-5 w-5 shrink-0" />
                        <span className="min-w-0 truncate">{evidenceFile?.name || (formData.image_url ? 'Ganti foto bukti' : 'Pilih JPEG, PNG, atau WebP (maks. 2 MB)')}</span>
                        <input type="file" accept="image/jpeg,image/png,image/webp" className="sr-only" onChange={(event) => setEvidenceFile(event.target.files?.[0] || null)} />
                      </label>
                    </div>
                  </div>
                </details>

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
                    {actionLoading === 'save' ? 'Menyimpan catatan…' : currentLog ? 'Simpan Perubahan' : 'Simpan Catatan'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {detailLog && (
          <div className="fixed inset-0 z-50 flex items-center justify-center px-4">
            <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => setDetailLog(null)} />
            <div
              role="dialog"
              aria-modal="true"
              aria-labelledby="maintenance-detail-title"
              className="z-10 max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl border border-border bg-card p-5 shadow-2xl sm:p-6"
            >
              <div className="mb-5 flex items-start justify-between gap-4">
                <div className="min-w-0">
                  <h3 id="maintenance-detail-title" className="text-xl font-bold tracking-[-0.015em]">Detail Pemeliharaan</h3>
                  <p className="mt-1 truncate font-mono text-xs font-semibold text-primary">
                    {detailLog.cctv?.kode_cctv || detailLog.cctv?.nama || 'CCTV tidak tersedia'}
                  </p>
                </div>
                <button type="button" autoFocus aria-label="Tutup detail pemeliharaan" onClick={() => setDetailLog(null)} className="shrink-0 rounded-lg p-2.5 text-muted transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"><X className="h-4 w-4" /></button>
              </div>

              <div className="mb-4 flex items-start justify-between gap-4 rounded-xl bg-background px-4 py-3">
                <div>
                  <p className="font-semibold">{detailLog.cctv?.nama || 'CCTV tidak tersedia'}</p>
                  <p className="mt-0.5 text-xs text-muted">{detailLog.technician?.full_name || 'Belum ditugaskan'}</p>
                </div>
                <span className={`inline-flex shrink-0 rounded-full border px-2.5 py-1 text-xs font-semibold ${
                  detailLog.status === 'completed'
                    ? 'border-success/25 bg-success/10 text-success'
                    : detailLog.status === 'failed'
                    ? 'border-error/25 bg-error/10 text-error'
                    : detailLog.status === 'ongoing'
                    ? 'border-warning/25 bg-warning/10 text-warning'
                    : 'border-blue-500/25 bg-blue-500/10 text-blue-600 dark:text-blue-400'
                }`}>
                  {getStatusLabel(detailLog.status)}
                </span>
              </div>

              <dl className="grid overflow-hidden rounded-xl border border-border text-sm sm:grid-cols-2">
                <div className="border-b border-border px-4 py-3 sm:border-r">
                  <dt className="text-xs font-semibold text-muted">Waktu</dt>
                  <dd className="mt-1 font-medium">{formatDate(detailLog.completed_at || detailLog.scheduled_date)}</dd>
                </div>
                <div className="border-b border-border px-4 py-3">
                  <dt className="text-xs font-semibold text-muted">Teknisi</dt>
                  <dd className="mt-1 font-medium">{detailLog.technician?.full_name || 'Belum ditugaskan'}</dd>
                </div>
                <div className="border-b border-border px-4 py-3 sm:col-span-2">
                  <dt className="text-xs font-semibold text-muted">Tindakan yang dilakukan</dt>
                  <dd className="mt-1 whitespace-pre-wrap font-medium">{detailLog.action_taken}</dd>
                </div>
                <div className="px-4 py-3 sm:col-span-2">
                  <dt className="text-xs font-semibold text-muted">Catatan tambahan</dt>
                  <dd className="mt-1 whitespace-pre-wrap text-muted">{detailLog.notes || 'Tidak ada catatan tambahan.'}</dd>
                </div>
              </dl>
              {detailLog.image_url && (
                <div className="mt-5">
                  <p className="mb-2 text-xs font-semibold text-muted">Foto bukti</p>
                  <AuthenticatedImage
                    imageUrl={detailLog.image_url}
                    alt={`Bukti pemeliharaan ${detailLog.cctv?.nama || 'CCTV'}`}
                    className="max-h-72 w-full rounded-xl border border-border bg-background object-contain"
                  />
                </div>
              )}
            </div>
          </div>
        )}
        {assignLog && (
          <div className="fixed inset-0 z-50 flex items-center justify-center px-4">
            <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => setAssignLog(null)} />
            <div
              role="dialog"
              aria-modal="true"
              aria-labelledby="maintenance-assign-title"
              className="z-10 max-h-[90vh] w-full max-w-md overflow-y-auto rounded-2xl border border-border bg-card p-5 shadow-2xl sm:p-6"
            >
              <div className="mb-5 flex items-start justify-between gap-4">
                <div className="min-w-0">
                  <h3 id="maintenance-assign-title" className="text-xl font-bold tracking-[-0.015em]">
                    {assignLog.status === 'ongoing' ? 'Alihkan ke Teknisi Lain' : 'Tugaskan ke Teknisi'}
                  </h3>
                  <p className="mt-1 text-sm text-muted">
                    {assignLog.cctv?.kode_cctv || 'CCTV'} · {assignLog.action_taken}
                  </p>
                </div>
                <button
                  type="button"
                  autoFocus
                  aria-label="Tutup penugasan"
                  onClick={() => setAssignLog(null)}
                  disabled={actionLoading === `assign-${assignLog.id}`}
                  className="shrink-0 rounded-lg p-2.5 text-muted transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>

              <div className="space-y-4">
                {assignLog.status === 'ongoing' ? (
                  <div className="flex items-start gap-3 rounded-xl border border-warning/25 bg-warning/10 px-4 py-3 text-xs text-foreground">
                    <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-warning" />
                    <p>
                      Tugas ini sedang dikerjakan oleh{' '}
                      <span className="font-bold">{assignLog.technician?.full_name || 'teknisi yang ditugaskan'}</span>.
                      Yakin ingin mengalihkan ke teknisi lain?
                    </p>
                  </div>
                ) : (
                  <div className="rounded-xl bg-background px-4 py-3 text-xs text-muted">
                    Tugas berstatus <span className="font-bold text-blue-600 dark:text-blue-400">Terjadwal</span> akan berubah menjadi{' '}
                    <span className="font-bold text-warning">Dalam proses</span> begitu ditugaskan, dan langsung tampil di dashboard teknisi.
                  </div>
                )}

                <div>
                  <label htmlFor="assign-technician" className="mb-1.5 block text-xs font-semibold text-muted">Pilih teknisi *</label>
                  <select
                    id="assign-technician"
                    value={assignTechnicianId}
                    onChange={(e) => setAssignTechnicianId(e.target.value)}
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    required
                  >
                    <option value="" disabled>Pilih teknisi</option>
                    {technicians.map((tech) => (
                      <option key={tech.id} value={tech.id}>{tech.full_name}</option>
                    ))}
                  </select>
                  {technicians.length === 0 && (
                    <p className="mt-1.5 text-[11px] text-red-500">Belum ada akun berperan teknisi. Buat akun teknisi di halaman Pengguna.</p>
                  )}
                </div>

                <div className="flex justify-end gap-3 pt-4 border-t border-border">
                  <button
                    type="button"
                    onClick={() => setAssignLog(null)}
                    disabled={actionLoading === `assign-${assignLog.id}`}
                    className="rounded-xl border border-border px-5 py-2.5 text-sm font-semibold transition-colors hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    Batal
                  </button>
                  <button
                    type="button"
                    onClick={handleAssignLog}
                    disabled={actionLoading === `assign-${assignLog.id}` || !assignTechnicianId}
                    className="flex items-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(15,118,74,0.16)] transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-card disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    {actionLoading === `assign-${assignLog.id}` && <LoaderCircle className="h-4 w-4 animate-spin" />}
                    {actionLoading === `assign-${assignLog.id}`
                      ? assignLog.status === 'ongoing' ? 'Mengalihkan…' : 'Menugaskan…'
                      : assignLog.status === 'ongoing' ? 'Alihkan' : 'Tugaskan'}
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {completeLog && (
          <FieldMaintenanceModal
            mode="complete"
            maintenanceLogId={completeLog.id}
            cctv={completeLog.cctv ? {
              id: completeLog.cctv.id,
              kode_cctv: completeLog.cctv.kode_cctv,
              nama: completeLog.cctv.nama,
              lokasi_area: completeLog.cctv.lokasi_area,
            } : null}
            open={true}
            onClose={() => setCompleteLog(null)}
            onSaved={() => fetchLogs()}
          />
        )}
      </div>
    </DashboardLayout>
  );
}

'use client';

import React, { useState, useEffect } from 'react';
import DashboardLayout from '@/components/DashboardLayout';
import Pagination from '@/components/Pagination';
import { apiRequest } from '@/lib/api';
import { useAuth } from '@/context/AuthContext';
import { 
  Plus, 
  Trash2, 
  Wrench, 
  Camera, 
  User as UserIcon, 
  Calendar, 
  Info,
  CheckCircle,
  AlertTriangle,
  Settings,
  X,
  FileText,
  Clock
} from 'lucide-react';

interface Asset {
  id: string;
  name: string;
  serial_number: string;
}

interface MasterCCTV {
  id: string;
  nama: string;
  kode_cctv: string;
}

interface User {
  id: string;
  full_name: string;
}

interface MaintenanceLog {
  id: string;
  asset_id: string;
  asset: Asset;
  technician_id: string;
  technician: User; // mapped by GORM Preload("User") via json:"technician"
  action_taken: string;
  notes: string;
  status: 'scheduled' | 'ongoing' | 'completed' | 'failed';
  scheduled_date: string | null;
  completed_at: string | null;
  created_at: string;
}

export default function MaintenanceLogsPage() {
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';

  const [logs, setLogs] = useState<MaintenanceLog[]>([]);
  const [assets, setAssets] = useState<Asset[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [assetsError, setAssetsError] = useState('');
  const [page, setPage] = useState(1);
  const [pagination, setPagination] = useState({
    total_rows: 0,
    total_pages: 1,
    page: 1,
    limit: 10,
  });

  // Filter State
  const [statusFilter, setStatusFilter] = useState('');
  const [assetFilter, setAssetFilter] = useState('');

  // Modal State
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [currentLog, setCurrentLog] = useState<MaintenanceLog | null>(null);

  // Form State
  const [formData, setFormData] = useState({
    asset_id: '',
    action_taken: '',
    notes: '',
    status: 'completed',
    scheduled_date: '',
    completed_at: new Date().toISOString().slice(0, 19).replace('T', ' ')
  });

  const fetchLogs = async () => {
    setLoading(true);
    setError('');
    try {
      const queryParams = [`page=${page}`, 'limit=10'];
      if (statusFilter) queryParams.push(`status=${encodeURIComponent(statusFilter)}`);
      if (assetFilter) queryParams.push(`asset_id=${encodeURIComponent(assetFilter)}`);
      
      const response = await apiRequest<any>(`/maintenance-logs?${queryParams.join('&')}`);
      if (response.status === 'success' && response.data) {
        setLogs(response.data.rows || []);
        setPagination(response.data.pagination || {
          total_rows: 0,
          total_pages: 1,
          page: 1,
          limit: 10,
        });
      }
    } catch (err: any) {
      setError(err.message || 'Gagal memuat log pemeliharaan');
    } finally {
      setLoading(false);
    }
  };

  const fetchAssetsList = async () => {
    setAssetsError('');
    try {
      const response = await apiRequest<any>('/cctv?page=1&limit=100');
      if (response.status === 'success' && response.data) {
        const rows: MasterCCTV[] = Array.isArray(response.data)
          ? response.data
          : (response.data.rows || []);
        setAssets(rows.map((cctv) => ({
          id: cctv.id,
          name: cctv.nama,
          serial_number: cctv.kode_cctv,
        })));
      }
    } catch (err: unknown) {
      setAssets([]);
      setAssetsError(
        err instanceof Error
          ? err.message
          : 'Daftar aset belum dapat dimuat dari server SIGAP.'
      );
    }
  };

  useEffect(() => {
    fetchLogs();
  }, [page, statusFilter, assetFilter]);

  useEffect(() => {
    fetchAssetsList();
  }, []);

  const handleOpenModal = (log: MaintenanceLog | null = null) => {
    if (log) {
      setCurrentLog(log);
      setFormData({
        asset_id: log.asset_id,
        action_taken: log.action_taken,
        notes: log.notes || '',
        status: log.status,
        scheduled_date: log.scheduled_date ? log.scheduled_date.split('T')[0] : '',
        completed_at: log.completed_at ? log.completed_at.replace('T', ' ').slice(0, 19) : ''
      });
    } else {
      setCurrentLog(null);
      setFormData({
        asset_id: assets.length > 0 ? assets[0].id : '',
        action_taken: '',
        notes: '',
        status: 'completed',
        scheduled_date: '',
        completed_at: new Date().toISOString().slice(0, 19).replace('T', ' ')
      });
    }
    setIsModalOpen(true);
  };

  const handleSaveLog = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.asset_id) {
      alert('Pilihlah salah satu aset CCTV');
      return;
    }

    try {
      const endpoint = currentLog ? `/maintenance-logs/${currentLog.id}` : '/maintenance-logs';
      const method = currentLog ? 'PUT' : 'POST';

      const response = await apiRequest<MaintenanceLog>(endpoint, {
        method,
        body: JSON.stringify(formData),
      });

      if (response.status === 'success') {
        setIsModalOpen(false);
        fetchLogs();
      }
    } catch (err: any) {
      alert(err.message || 'Gagal menyimpan log pemeliharaan');
    }
  };

  const handleDeleteLog = async (id: string) => {
    if (!confirm('Apakah Anda yakin ingin menghapus data riwayat pemeliharaan ini? (Soft delete)')) return;
    try {
      const response = await apiRequest(`/maintenance-logs/${id}`, {
        method: 'DELETE'
      });
      if (response.status === 'success') {
        fetchLogs();
      }
    } catch (err: any) {
      alert(err.message || 'Gagal menghapus log');
    }
  };

  // Helper date formatter
  const formatDate = (dateStr: string | null) => {
    if (!dateStr) return '-';
    try {
      const date = new Date(dateStr);
      return date.toLocaleDateString('id-ID', {
        day: 'numeric',
        month: 'long',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      });
    } catch (e) {
      return dateStr;
    }
  };

  return (
    <DashboardLayout>
      <div className="space-y-8 animate-fadeIn">
        {/* Header */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <h1 className="text-3xl font-bold tracking-tight">Riwayat Pemeliharaan</h1>
          <button
            onClick={() => handleOpenModal()}
            className="flex items-center justify-center gap-2 px-5 py-3 rounded-xl bg-primary hover:bg-primary-hover text-white font-semibold shadow-lg shadow-primary/15 active:scale-[0.98] transition-all cursor-pointer text-sm"
          >
            <Plus className="h-5 w-5" />
            <span>Catat Pemeliharaan</span>
          </button>
        </div>

        {/* Filter Controls */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 p-5 rounded-2xl bg-card border border-border bg-card/60 backdrop-blur-md">
          {/* Asset Select Filter */}
          <div>
            <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Filter CCTV</label>
            <select
              value={assetFilter}
              onChange={(e) => {
                setAssetFilter(e.target.value);
                setPage(1);
              }}
              className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
            >
              <option value="">Semua CCTV</option>
              {assets.map((asset) => (
                <option key={asset.id} value={asset.id}>{asset.name}</option>
              ))}
            </select>
          </div>

          {/* Status Select Filter */}
          <div>
            <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Filter Status</label>
            <select
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value);
                setPage(1);
              }}
              className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
            >
              <option value="">Semua Status</option>
              <option value="completed">Selesai (Completed)</option>
              <option value="scheduled">Terjadwal (Scheduled)</option>
              <option value="ongoing">Dalam Proses (Ongoing)</option>
              <option value="failed">Gagal (Failed)</option>
            </select>
          </div>
        </div>

        {assetsError && (
          <div
            role="alert"
            className="flex flex-col gap-3 rounded-2xl border border-warning/25 bg-warning/10 px-5 py-4 text-sm sm:flex-row sm:items-center sm:justify-between"
          >
            <div className="flex items-start gap-3">
              <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-warning" />
              <div>
                <p className="font-semibold text-foreground">Daftar aset belum tersedia</p>
                <p className="mt-0.5 text-xs text-muted">{assetsError}</p>
              </div>
            </div>
            <button
              type="button"
              onClick={fetchAssetsList}
              className="shrink-0 rounded-lg border border-warning/30 bg-card px-3 py-2 text-xs font-bold text-foreground transition-colors hover:border-warning"
            >
              Coba lagi
            </button>
          </div>
        )}

        {/* Logs Table */}
        {loading ? (
          <div className="flex justify-center items-center py-20 bg-card rounded-2xl border border-border">
            <div className="h-8 w-8 animate-spin rounded-full border-3 border-primary border-t-transparent"></div>
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center rounded-2xl border border-error/25 bg-card px-6 py-16 text-center">
            <AlertTriangle className="h-11 w-11 text-error" />
            <h3 className="mt-3 text-base font-semibold">Data pemeliharaan belum dapat dimuat</h3>
            <p className="mt-1 max-w-md text-xs text-muted">{error}</p>
            <button
              type="button"
              onClick={fetchLogs}
              className="mt-4 rounded-lg bg-primary px-4 py-2 text-xs font-bold text-white transition-colors hover:bg-primary-hover"
            >
              Coba lagi
            </button>
          </div>
        ) : logs.length === 0 ? (
          <div className="text-center py-16 bg-card rounded-2xl border border-border flex flex-col items-center justify-center">
            <Wrench className="h-12 w-12 text-gray-300 dark:text-gray-700 mb-3" />
            <h3 className="text-base font-semibold">Belum Ada Catatan Pemeliharaan</h3>
            <p className="text-xs text-gray-400 mt-1 max-w-sm">
              Silakan buat catatan baru dengan mengklik tombol "Catat Pemeliharaan" di kanan atas.
            </p>
          </div>
        ) : (
          <div className="overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="border-b border-border bg-background/50 text-xs font-bold text-gray-400 uppercase tracking-wider">
                    <th className="py-4 px-6">Nama CCTV</th>
                    <th className="py-4 px-6">Tindakan Pemeliharaan</th>
                    <th className="py-4 px-6">Teknisi</th>
                    <th className="py-4 px-6">Tanggal Selesai</th>
                    <th className="py-4 px-6">Status</th>
                    {isAdmin && <th className="py-4 px-6 text-right">Aksi</th>}
                  </tr>
                </thead>
                <tbody className="divide-y divide-border text-sm">
                  {logs.map((log) => (
                    <tr key={log.id} className="hover:bg-background/25 transition-colors">
                      {/* CCTV Info */}
                      <td className="py-4.5 px-6">
                        <div className="flex items-center gap-2.5">
                          <Camera className="h-4 w-4 text-gray-400 shrink-0" />
                          <span className="font-semibold text-gray-700 dark:text-gray-200">
                            {log.asset?.name || 'Aset Tidak Ditemukan'}
                          </span>
                        </div>
                      </td>
                      
                      {/* Action & Notes */}
                      <td className="py-4.5 px-6 max-w-xs">
                        <div>
                          <p className="font-medium text-gray-800 dark:text-gray-100 line-clamp-1" title={log.action_taken}>
                            {log.action_taken}
                          </p>
                          {log.notes && (
                            <p className="text-xs text-gray-400 mt-0.5 line-clamp-1" title={log.notes}>
                              Catatan: {log.notes}
                            </p>
                          )}
                        </div>
                      </td>

                      {/* Technician */}
                      <td className="py-4.5 px-6">
                        <div className="flex items-center gap-2 text-gray-500 dark:text-gray-400">
                          <UserIcon className="h-4 w-4 text-gray-400 shrink-0" />
                          <span>{log.technician?.full_name || 'Teknisi Sistem'}</span>
                        </div>
                      </td>

                      {/* Date */}
                      <td className="py-4.5 px-6 text-gray-500 dark:text-gray-400 text-xs">
                        <div className="flex items-center gap-2">
                          <Clock className="h-3.5 w-3.5 text-gray-400" />
                          <span>{formatDate(log.completed_at || log.scheduled_date)}</span>
                        </div>
                      </td>

                      {/* Status */}
                      <td className="py-4.5 px-6">
                        <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold capitalize ${
                          log.status === 'completed' 
                            ? 'bg-emerald-500/10 text-emerald-500 border border-emerald-500/20' 
                            : log.status === 'failed'
                            ? 'bg-red-500/10 text-red-500 border border-red-500/20'
                            : log.status === 'ongoing'
                            ? 'bg-amber-500/10 text-amber-500 border border-amber-500/20'
                            : 'bg-blue-500/10 text-blue-500 border border-blue-500/20'
                        }`}>
                          {log.status === 'completed' ? 'Selesai' : log.status === 'failed' ? 'Gagal' : log.status === 'ongoing' ? 'Dalam Proses' : 'Terjadwal'}
                        </span>
                      </td>

                      {/* Delete Action (Admin Only) */}
                      {isAdmin && (
                        <td className="py-4.5 px-6 text-right">
                          <div className="flex items-center justify-end gap-2">
                            <button
                              onClick={() => handleOpenModal(log)}
                              className="p-2 rounded-lg border border-border bg-card hover:border-blue-500/20 hover:bg-blue-500/5 hover:text-blue-500 text-gray-500 dark:text-gray-400 transition-all cursor-pointer"
                              title="Edit Log"
                            >
                              <Settings className="h-4 w-4" />
                            </button>
                            <button
                              onClick={() => handleDeleteLog(log.id)}
                              className="p-2 rounded-lg border border-border bg-card hover:border-red-500/20 hover:bg-red-500/5 hover:text-red-500 text-gray-500 dark:text-gray-400 transition-all cursor-pointer"
                              title="Hapus Log"
                            >
                              <Trash2 className="h-4 w-4" />
                            </button>
                          </div>
                        </td>
                      )}
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

        {/* MODAL FORM TAMBAH/EDIT LOG PEMELIHARAAN */}
        {isModalOpen && (
          <div className="fixed inset-0 flex items-center justify-center z-50 px-4">
            <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => setIsModalOpen(false)}></div>
            
            <div className="bg-card w-full max-w-lg rounded-2xl border border-border shadow-2xl p-6 z-10 animate-scaleIn">
              <div className="flex justify-between items-center mb-6">
                <h3 className="text-xl font-bold">{currentLog ? 'Edit Catatan Pemeliharaan' : 'Catat Pemeliharaan Baru'}</h3>
                <button onClick={() => setIsModalOpen(false)} className="p-1.5 rounded-lg border border-border bg-background hover:bg-border text-gray-500 transition-colors">
                  <X className="h-4 w-4" />
                </button>
              </div>

              <form onSubmit={handleSaveLog} className="space-y-4">
                {/* Select Asset */}
                <div>
                  <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Pilih CCTV *</label>
                  <select
                    value={formData.asset_id}
                    onChange={(e) => setFormData({...formData, asset_id: e.target.value})}
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    required
                    disabled={!!currentLog}
                  >
                    <option value="" disabled>-- Pilih Kamera CCTV --</option>
                    {assets.map((asset) => (
                      <option key={asset.id} value={asset.id}>
                        {asset.name} ({asset.serial_number})
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Tindakan Pemeliharaan *</label>
                  <textarea
                    rows={3}
                    value={formData.action_taken}
                    onChange={(e) => setFormData({...formData, action_taken: e.target.value})}
                    placeholder="Deskripsikan tindakan yang dilakukan... (contoh: pembersihan lensa, pengganti konektor RJ45)"
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    required
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Catatan Tambahan</label>
                  <textarea
                    rows={2}
                    value={formData.notes}
                    onChange={(e) => setFormData({...formData, notes: e.target.value})}
                    placeholder="Tuliskan catatan teknis tambahan..."
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                  />
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Status Pemeliharaan *</label>
                    <select
                      value={formData.status}
                      onChange={(e) => setFormData({...formData, status: e.target.value})}
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    >
                      <option value="completed">Selesai (Completed)</option>
                      <option value="scheduled">Terjadwal (Scheduled)</option>
                      <option value="ongoing">Dalam Proses (Ongoing)</option>
                      <option value="failed">Gagal (Failed)</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Waktu Selesai / Jadwal *</label>
                    <input
                      type="text"
                      value={formData.completed_at || formData.scheduled_date}
                      onChange={(e) => {
                        if (formData.status === 'scheduled') {
                          setFormData({...formData, scheduled_date: e.target.value, completed_at: ''});
                        } else {
                          setFormData({...formData, completed_at: e.target.value, scheduled_date: ''});
                        }
                      }}
                      placeholder="YYYY-MM-DD HH:MM:SS"
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                      required
                    />
                  </div>
                </div>

                <div className="flex justify-end gap-3 pt-4 border-t border-border mt-6">
                  <button
                    type="button"
                    onClick={() => setIsModalOpen(false)}
                    className="px-5 py-2.5 rounded-xl border border-border hover:bg-background text-sm font-semibold transition-colors cursor-pointer"
                  >
                    Batal
                  </button>
                  <button
                    type="submit"
                    className="px-5 py-2.5 rounded-xl bg-gradient-to-r from-amber-500 to-orange-600 hover:from-amber-600 hover:to-orange-700 text-white font-semibold shadow-lg shadow-amber-500/10 active:scale-[0.98] transition-all cursor-pointer text-sm"
                  >
                    {currentLog ? 'Simpan Perubahan' : 'Simpan Catatan'}
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

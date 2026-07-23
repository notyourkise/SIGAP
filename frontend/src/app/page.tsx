'use client';

import React, { useState, useEffect } from 'react';
import DashboardLayout from '@/components/DashboardLayout';
import { apiRequest } from '@/lib/api';
import { useAuth } from '@/context/AuthContext';
import { 
  Plus, 
  Search, 
  Edit2, 
  Trash2, 
  Camera, 
  MapPin, 
  Tag, 
  Info,
  CheckCircle,
  AlertTriangle,
  Settings,
  X,
  Calendar,
  Layers,
  Activity,
  Wrench
} from 'lucide-react';
import Link from 'next/link';

interface Asset {
  id: string;
  name: string;
  serial_number: string;
  ip_address: string;
  location_name: string;
  latitude: number;
  longitude: number;
  status: 'aktif' | 'rusak' | 'pemeliharaan' | 'nonaktif';
  vendor: string;
  installation_date: string;
  created_at: string;
}

export default function AssetsPage() {
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';

  const [assets, setAssets] = useState<Asset[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  
  // Search and Filter State
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  
  // Modals state
  const [isFormModalOpen, setIsFormModalOpen] = useState(false);
  const [isMaintenanceModalOpen, setIsMaintenanceModalOpen] = useState(false);
  const [currentAsset, setCurrentAsset] = useState<Asset | null>(null);
  
  // Form State
  const [formData, setFormData] = useState({
    name: '',
    serial_number: '',
    ip_address: '',
    location_name: '',
    latitude: 0,
    longitude: 0,
    status: 'aktif',
    vendor: '',
    installation_date: new Date().toISOString().split('T')[0]
  });

  // Maintenance Log Form State (Quick Action)
  const [maintenanceData, setMaintenanceData] = useState({
    action_taken: '',
    notes: '',
    status: 'completed',
    scheduled_date: '',
    completed_at: new Date().toISOString().slice(0, 19).replace('T', ' ')
  });

  const fetchAssets = async () => {
    setLoading(true);
    try {
      const queryParams = [];
      if (search) queryParams.push(`search=${encodeURIComponent(search)}`);
      if (statusFilter) queryParams.push(`status=${encodeURIComponent(statusFilter)}`);
      
      const queryString = queryParams.length > 0 ? `?${queryParams.join('&')}` : '';
      const response = await apiRequest<any>(`/assets${queryString}`);
      
      if (response.status === 'success' && response.data) {
        const assetsData = Array.isArray(response.data) ? response.data : (response.data.rows || []);
        setAssets(assetsData);
      }
    } catch (err: any) {
      setError(err.message || 'Gagal memuat data aset');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAssets();
  }, [search, statusFilter]);

  // Statistics calculation
  const totalAssets = assets.length;
  const activeCount = assets.filter(a => a.status === 'aktif').length;
  const brokenCount = assets.filter(a => a.status === 'rusak').length;
  const maintenanceCount = assets.filter(a => a.status === 'pemeliharaan').length;

  const handleOpenForm = (asset: Asset | null = null) => {
    if (asset) {
      setCurrentAsset(asset);
      setFormData({
        name: asset.name,
        serial_number: asset.serial_number,
        ip_address: asset.ip_address || '',
        location_name: asset.location_name || '',
        latitude: asset.latitude || 0,
        longitude: asset.longitude || 0,
        status: asset.status,
        vendor: asset.vendor || '',
        installation_date: asset.installation_date ? asset.installation_date.split('T')[0] : ''
      });
    } else {
      setCurrentAsset(null);
      setFormData({
        name: '',
        serial_number: '',
        ip_address: '',
        location_name: '',
        latitude: 0,
        longitude: 0,
        status: 'aktif',
        vendor: '',
        installation_date: new Date().toISOString().split('T')[0]
      });
    }
    setIsFormModalOpen(true);
  };

  const handleSaveAsset = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const endpoint = currentAsset ? `/assets/${currentAsset.id}` : '/assets';
      const method = currentAsset ? 'PUT' : 'POST';

      const response = await apiRequest<Asset>(endpoint, {
        method,
        body: JSON.stringify(formData),
      });

      if (response.status === 'success') {
        setIsFormModalOpen(false);
        fetchAssets();
      }
    } catch (err: any) {
      alert(err.message || 'Gagal menyimpan data aset');
    }
  };

  const handleDeleteAsset = async (id: string) => {
    if (!confirm('Apakah Anda yakin ingin menghapus aset CCTV ini? (Soft delete)')) return;
    try {
      const response = await apiRequest(`/assets/${id}`, {
        method: 'DELETE'
      });
      if (response.status === 'success') {
        fetchAssets();
      }
    } catch (err: any) {
      alert(err.message || 'Gagal menghapus aset');
    }
  };

  // Open maintenance log modal
  const handleOpenMaintenance = (asset: Asset) => {
    setCurrentAsset(asset);
    setMaintenanceData({
      action_taken: '',
      notes: '',
      status: 'completed',
      scheduled_date: '',
      completed_at: new Date().toISOString().slice(0, 19).replace('T', ' ')
    });
    setIsMaintenanceModalOpen(true);
  };

  const handleSaveMaintenance = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!currentAsset) return;
    
    try {
      const payload = {
        asset_id: currentAsset.id,
        ...maintenanceData
      };

      const response = await apiRequest('/maintenance-logs', {
        method: 'POST',
        body: JSON.stringify(payload)
      });

      if (response.status === 'success') {
        setIsMaintenanceModalOpen(false);
        fetchAssets(); // Refresh assets status if needed
        alert('Log pemeliharaan berhasil dicatat!');
      }
    } catch (err: any) {
      alert(err.message || 'Gagal mencatat log pemeliharaan');
    }
  };

  return (
    <DashboardLayout>
      <div className="space-y-8 animate-fadeIn">
        {/* Header */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Daftar Aset CCTV</h1>
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
              Pemantauan, filter status, dan pencatatan pemeliharaan CCTV kawasan IKN.
            </p>
          </div>
          {isAdmin && (
            <button
              onClick={() => handleOpenForm()}
              className="flex items-center justify-center gap-2 px-5 py-3 rounded-xl bg-gradient-to-r from-blue-500 to-indigo-600 hover:from-blue-600 hover:to-indigo-700 text-white font-semibold shadow-lg shadow-blue-500/10 active:scale-[0.98] transition-all cursor-pointer text-sm"
            >
              <Plus className="h-5 w-5" />
              <span>Tambah CCTV</span>
            </button>
          )}
        </div>

        {/* Stats Row */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="p-5 rounded-2xl bg-card border border-border flex items-center gap-4">
            <div className="p-3.5 rounded-xl bg-blue-500/10 text-blue-500 dark:bg-blue-500/20">
              <Camera className="h-6 w-6" />
            </div>
            <div>
              <span className="text-xs text-gray-400 font-medium">Total CCTV</span>
              <h3 className="text-2xl font-bold mt-0.5">{totalAssets}</h3>
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-card border border-border flex items-center gap-4">
            <div className="p-3.5 rounded-xl bg-emerald-500/10 text-emerald-500 dark:bg-emerald-500/20">
              <CheckCircle className="h-6 w-6" />
            </div>
            <div>
              <span className="text-xs text-gray-400 font-medium">Aktif</span>
              <h3 className="text-2xl font-bold mt-0.5 text-emerald-500">{activeCount}</h3>
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-card border border-border flex items-center gap-4">
            <div className="p-3.5 rounded-xl bg-red-500/10 text-red-500 dark:bg-red-500/20">
              <AlertTriangle className="h-6 w-6" />
            </div>
            <div>
              <span className="text-xs text-gray-400 font-medium">Rusak</span>
              <h3 className="text-2xl font-bold mt-0.5 text-red-500">{brokenCount}</h3>
            </div>
          </div>

          <div className="p-5 rounded-2xl bg-card border border-border flex items-center gap-4">
            <div className="p-3.5 rounded-xl bg-amber-500/10 text-amber-500 dark:bg-amber-500/20">
              <Activity className="h-6 w-6" />
            </div>
            <div>
              <span className="text-xs text-gray-400 font-medium">Pemeliharaan</span>
              <h3 className="text-2xl font-bold mt-0.5 text-amber-500">{maintenanceCount}</h3>
            </div>
          </div>
        </div>

        {/* Filter & Search Controls */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 p-5 rounded-2xl bg-card border border-border bg-card/60 backdrop-blur-md">
          {/* Search */}
          <div className="relative flex-1 max-w-md">
            <div className="absolute inset-y-0 left-0 flex items-center pl-4 pointer-events-none text-gray-400">
              <Search className="h-5 w-5" />
            </div>
            <input
              type="text"
              placeholder="Cari nama CCTV atau serial number..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full pl-11 pr-4 py-2.5 rounded-xl border border-input-border bg-input text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary text-sm transition-all"
            />
          </div>

          {/* Status Filter */}
          <div className="flex gap-2">
            {['', 'aktif', 'rusak', 'pemeliharaan', 'nonaktif'].map((status) => {
              const label = status === '' ? 'Semua Status' : status.charAt(0).toUpperCase() + status.slice(1);
              const isActive = statusFilter === status;
              return (
                <button
                  key={status}
                  onClick={() => setStatusFilter(status)}
                  className={`px-4 py-2 rounded-xl text-xs font-semibold border transition-all cursor-pointer ${
                    isActive 
                      ? 'bg-primary text-white border-primary shadow-md shadow-primary/10' 
                      : 'border-border bg-background text-gray-500 hover:text-foreground dark:text-gray-400'
                  }`}
                >
                  {label}
                </button>
              );
            })}
          </div>
        </div>

        {/* Assets List Table */}
        {loading ? (
          <div className="flex justify-center items-center py-20 bg-card rounded-2xl border border-border">
            <div className="h-8 w-8 animate-spin rounded-full border-3 border-primary border-t-transparent"></div>
          </div>
        ) : assets.length === 0 ? (
          <div className="text-center py-16 bg-card rounded-2xl border border-border flex flex-col items-center justify-center">
            <Camera className="h-12 w-12 text-gray-300 dark:text-gray-700 mb-3" />
            <h3 className="text-base font-semibold">Tidak Ada CCTV Ditemukan</h3>
            <p className="text-xs text-gray-400 mt-1 max-w-sm">
              Coba ganti filter status atau ubah kata pencarian Anda.
            </p>
          </div>
        ) : (
          <div className="overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="border-b border-border bg-background/50 text-xs font-bold text-gray-400 uppercase tracking-wider">
                    <th className="py-4 px-6">Detail CCTV</th>
                    <th className="py-4 px-6">Alamat IP</th>
                    <th className="py-4 px-6">Lokasi</th>
                    <th className="py-4 px-6">Status</th>
                    <th className="py-4 px-6 text-right">Aksi</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border text-sm">
                  {assets.map((asset) => (
                    <tr key={asset.id} className="hover:bg-background/25 transition-colors">
                      <td className="py-4.5 px-6">
                        <div className="flex items-center gap-3">
                          <div className="p-2.5 rounded-lg bg-blue-500/5 text-blue-500 dark:bg-blue-400/5 dark:text-blue-400 border border-blue-500/10">
                            <Camera className="h-5 w-5" />
                          </div>
                          <div>
                            <h4 className="font-semibold">{asset.name}</h4>
                            <span className="text-xs text-gray-400 block mt-0.5 font-mono">SN: {asset.serial_number}</span>
                          </div>
                        </div>
                      </td>
                      <td className="py-4.5 px-6 font-mono text-xs text-gray-500 dark:text-gray-400">
                        {asset.ip_address || '-'}
                      </td>
                      <td className="py-4.5 px-6">
                        <div className="flex items-center gap-2 text-gray-600 dark:text-gray-300">
                          <MapPin className="h-4 w-4 shrink-0 text-gray-400" />
                          <span className="truncate max-w-[200px]" title={asset.location_name}>{asset.location_name || '-'}</span>
                        </div>
                      </td>
                      <td className="py-4.5 px-6">
                        <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold capitalize ${
                          asset.status === 'aktif' 
                            ? 'bg-emerald-500/10 text-emerald-500 border border-emerald-500/20' 
                            : asset.status === 'rusak'
                            ? 'bg-red-500/10 text-red-500 border border-red-500/20'
                            : asset.status === 'pemeliharaan'
                            ? 'bg-amber-500/10 text-amber-500 border border-amber-500/20'
                            : 'bg-gray-500/10 text-gray-500 border border-gray-500/20'
                        }`}>
                          <span className={`h-1.5 w-1.5 rounded-full ${
                            asset.status === 'aktif' ? 'bg-emerald-500' : asset.status === 'rusak' ? 'bg-red-500' : asset.status === 'pemeliharaan' ? 'bg-amber-500' : 'bg-gray-500'
                          }`} />
                          {asset.status}
                        </span>
                      </td>
                      <td className="py-4.5 px-6 text-right">
                        <div className="flex items-center justify-end gap-2">
                          <button
                            onClick={() => handleOpenMaintenance(asset)}
                            className="p-2 rounded-lg border border-border bg-card hover:border-amber-500/20 hover:bg-amber-500/5 hover:text-amber-500 text-gray-500 dark:text-gray-400 transition-all cursor-pointer"
                            title="Catat Pemeliharaan"
                          >
                            <Wrench className="h-4 w-4" />
                          </button>
                          {isAdmin && (
                            <>
                              <button
                                onClick={() => handleOpenForm(asset)}
                                className="p-2 rounded-lg border border-border bg-card hover:border-blue-500/20 hover:bg-blue-500/5 hover:text-blue-500 text-gray-500 dark:text-gray-400 transition-all cursor-pointer"
                                title="Edit CCTV"
                              >
                                <Edit2 className="h-4 w-4" />
                              </button>
                              <button
                                onClick={() => handleDeleteAsset(asset.id)}
                                className="p-2 rounded-lg border border-border bg-card hover:border-red-500/20 hover:bg-red-500/5 hover:text-red-500 text-gray-500 dark:text-gray-400 transition-all cursor-pointer"
                                title="Hapus CCTV"
                              >
                                <Trash2 className="h-4 w-4" />
                              </button>
                            </>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* MODAL FORM TAMBAH/EDIT ASET */}
        {isFormModalOpen && (
          <div className="fixed inset-0 flex items-center justify-center z-50 px-4">
            <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => setIsFormModalOpen(false)}></div>
            
            <div className="bg-card w-full max-w-xl rounded-2xl border border-border shadow-2xl p-6 z-10 max-h-[90vh] overflow-y-auto animate-scaleIn">
              <div className="flex justify-between items-center mb-6">
                <h3 className="text-xl font-bold">{currentAsset ? 'Edit Aset CCTV' : 'Tambah Aset CCTV Baru'}</h3>
                <button onClick={() => setIsFormModalOpen(false)} className="p-1.5 rounded-lg border border-border bg-background hover:bg-border text-gray-500 transition-colors">
                  <X className="h-4 w-4" />
                </button>
              </div>

              <form onSubmit={handleSaveAsset} className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Nama CCTV *</label>
                    <input
                      type="text"
                      value={formData.name}
                      onChange={(e) => setFormData({...formData, name: e.target.value})}
                      placeholder="e.g., CCTV Simpang Utama A"
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                      required
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Nomor Seri (Serial Number) *</label>
                    <input
                      type="text"
                      value={formData.serial_number}
                      onChange={(e) => setFormData({...formData, serial_number: e.target.value})}
                      placeholder="e.g., SN-CCTV-IKN-001"
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                      required
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">IP Address</label>
                    <input
                      type="text"
                      value={formData.ip_address}
                      onChange={(e) => setFormData({...formData, ip_address: e.target.value})}
                      placeholder="e.g., 192.168.10.15"
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Vendor / Produsen</label>
                    <input
                      type="text"
                      value={formData.vendor}
                      onChange={(e) => setFormData({...formData, vendor: e.target.value})}
                      placeholder="e.g., Hikvision / Dahua"
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Deskripsi Lokasi</label>
                  <input
                    type="text"
                    value={formData.location_name}
                    onChange={(e) => setFormData({...formData, location_name: e.target.value})}
                    placeholder="e.g., Kawasan Istana Presiden, Simpang A1"
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                  />
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Garis Lintang (Latitude)</label>
                    <input
                      type="number"
                      step="any"
                      value={formData.latitude}
                      onChange={(e) => setFormData({...formData, latitude: parseFloat(e.target.value) || 0})}
                      placeholder="e.g., -0.897623"
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Garis Bujur (Longitude)</label>
                    <input
                      type="number"
                      step="any"
                      value={formData.longitude}
                      onChange={(e) => setFormData({...formData, longitude: parseFloat(e.target.value) || 0})}
                      placeholder="e.g., 117.123456"
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Status Aset *</label>
                    <select
                      value={formData.status}
                      onChange={(e) => setFormData({...formData, status: e.target.value})}
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    >
                      <option value="aktif">Aktif</option>
                      <option value="rusak">Rusak</option>
                      <option value="pemeliharaan">Pemeliharaan</option>
                      <option value="nonaktif">Nonaktif</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Tanggal Instalasi *</label>
                    <input
                      type="date"
                      value={formData.installation_date}
                      onChange={(e) => setFormData({...formData, installation_date: e.target.value})}
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                      required
                    />
                  </div>
                </div>

                <div className="flex justify-end gap-3 pt-4 border-t border-border mt-6">
                  <button
                    type="button"
                    onClick={() => setIsFormModalOpen(false)}
                    className="px-5 py-2.5 rounded-xl border border-border hover:bg-background text-sm font-semibold transition-colors cursor-pointer"
                  >
                    Batal
                  </button>
                  <button
                    type="submit"
                    className="px-5 py-2.5 rounded-xl bg-gradient-to-r from-blue-500 to-indigo-600 hover:from-blue-600 hover:to-indigo-700 text-white font-semibold shadow-lg shadow-blue-500/10 active:scale-[0.98] transition-all cursor-pointer text-sm"
                  >
                    {currentAsset ? 'Simpan Perubahan' : 'Tambah Aset'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* MODAL CATAT PEMELIHARAAN (QUICK ACTION) */}
        {isMaintenanceModalOpen && currentAsset && (
          <div className="fixed inset-0 flex items-center justify-center z-50 px-4">
            <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => setIsMaintenanceModalOpen(false)}></div>
            
            <div className="bg-card w-full max-w-lg rounded-2xl border border-border shadow-2xl p-6 z-10 animate-scaleIn">
              <div className="flex justify-between items-center mb-6">
                <div>
                  <h3 className="text-xl font-bold">Catat Pemeliharaan CCTV</h3>
                  <p className="text-xs text-gray-400 mt-0.5">{currentAsset.name} ({currentAsset.serial_number})</p>
                </div>
                <button onClick={() => setIsMaintenanceModalOpen(false)} className="p-1.5 rounded-lg border border-border bg-background hover:bg-border text-gray-500 transition-colors">
                  <X className="h-4 w-4" />
                </button>
              </div>

              <form onSubmit={handleSaveMaintenance} className="space-y-4">
                <div>
                  <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Tindakan Pemeliharaan *</label>
                  <textarea
                    rows={3}
                    value={maintenanceData.action_taken}
                    onChange={(e) => setMaintenanceData({...maintenanceData, action_taken: e.target.value})}
                    placeholder="Deskripsikan tindakan yang dilakukan... (contoh: pembersihan lensa, pengganti konektor RJ45)"
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    required
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Catatan Tambahan</label>
                  <textarea
                    rows={2}
                    value={maintenanceData.notes}
                    onChange={(e) => setMaintenanceData({...maintenanceData, notes: e.target.value})}
                    placeholder="Tuliskan catatan teknis tambahan..."
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                  />
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Status Pemeliharaan *</label>
                    <select
                      value={maintenanceData.status}
                      onChange={(e) => setMaintenanceData({...maintenanceData, status: e.target.value})}
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    >
                      <option value="completed">Selesai (Completed)</option>
                      <option value="scheduled">Terjadwal (Scheduled)</option>
                      <option value="ongoing">Dalam Proses (Ongoing)</option>
                      <option value="failed">Gagal (Failed)</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Waktu Selesai *</label>
                    <input
                      type="text"
                      value={maintenanceData.completed_at}
                      onChange={(e) => setMaintenanceData({...maintenanceData, completed_at: e.target.value})}
                      placeholder="YYYY-MM-DD HH:MM:SS"
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                      required
                    />
                  </div>
                </div>

                <div className="flex justify-end gap-3 pt-4 border-t border-border mt-6">
                  <button
                    type="button"
                    onClick={() => setIsMaintenanceModalOpen(false)}
                    className="px-5 py-2.5 rounded-xl border border-border hover:bg-background text-sm font-semibold transition-colors cursor-pointer"
                  >
                    Batal
                  </button>
                  <button
                    type="submit"
                    className="px-5 py-2.5 rounded-xl bg-gradient-to-r from-amber-500 to-orange-600 hover:from-amber-600 hover:to-orange-700 text-white font-semibold shadow-lg shadow-amber-500/10 active:scale-[0.98] transition-all cursor-pointer text-sm"
                  >
                    Simpan Catatan
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

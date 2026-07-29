'use client';

import React, { useState, useEffect } from 'react';
import DashboardLayout from '@/components/DashboardLayout';
import Pagination from '@/components/Pagination';
import { apiRequest } from '@/lib/api';
import { useAuth } from '@/context/AuthContext';
import { 
  Plus, 
  Search, 
  Edit2, 
  Trash2, 
  Camera, 
  MapPin, 
  Upload, 
  Scan, 
  X, 
  AlertTriangle, 
  Activity, 
  ShieldCheck,
  Wrench,
  Cpu, 
  Minimize2, 
  Eye, 
  Printer,
  LoaderCircle
} from 'lucide-react';

interface CCTV {
  id: string;
  kode_cctv: string;
  id_channel: string;
  nama: string;
  lokasi_area: string;
  longitude: string;
  latitude: string;
  jenis_cctv: string;
  sumber_listrik: string;
  tipe_koneksi: string;
  indoor_outdoor: string;
  pemilik: string;
  status: string;
  ip_nvr: string;
  ip_camera: string;
  keterangan: string;
  qr_payload: string;
  created_at: string;
}

export default function CCTVPage() {
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';

  const [cctvs, setCctvs] = useState<CCTV[]>([]);
  const [allCCTVs, setAllCCTVs] = useState<CCTV[]>([]);
  const [maintenanceTotal, setMaintenanceTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [page, setPage] = useState(1);
  const [pagination, setPagination] = useState({
    total_rows: 0,
    total_pages: 1,
    page: 1,
    limit: 10,
  });

  // Search & Filter State
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [ioFilter, setIoFilter] = useState('');

  // Modals state
  const [isFormModalOpen, setIsFormModalOpen] = useState(false);
  const [isScanModalOpen, setIsScanModalOpen] = useState(false);
  const [isImportModalOpen, setIsImportModalOpen] = useState(false);
  const [isQrModalOpen, setIsQrModalOpen] = useState(false);
  const [isDetailModalOpen, setIsDetailModalOpen] = useState(false);
  
  const [currentCCTV, setCurrentCCTV] = useState<CCTV | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Form State
  const [formData, setFormData] = useState({
    kode_cctv: '',
    id_channel: '',
    nama: '',
    lokasi_area: '',
    longitude: '',
    latitude: '',
    jenis_cctv: '',
    sumber_listrik: '',
    tipe_koneksi: '',
    indoor_outdoor: 'INDOOR',
    pemilik: '',
    status: 'ACTIVE',
    ip_nvr: '',
    ip_camera: '',
    keterangan: ''
  });

  // Scanner State
  const scannerRef = React.useRef<any>(null);
  const [cameraActive, setCameraActive] = useState(false);
  const [showManualInput, setShowManualInput] = useState(false);
  const [scanPayload, setScanPayload] = useState('');
  const [scannedResult, setScannedResult] = useState<CCTV | null>(null);
  const [scanError, setScanError] = useState('');
  const [scanLoading, setScanLoading] = useState(false);
  
  // Bulk Actions State
  const [selectedIDs, setSelectedIDs] = useState<string[]>([]);

  // File Import State
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [importLoading, setImportLoading] = useState(false);
  const [importError, setImportError] = useState('');
  const [importSuccess, setImportSuccess] = useState('');

  const fetchCCTVs = async () => {
    setLoading(true);
    setError('');
    try {
      const queryParams = [`page=${page}`, 'limit=10'];
      if (search) queryParams.push(`search=${encodeURIComponent(search)}`);
      if (statusFilter) queryParams.push(`status=${encodeURIComponent(statusFilter)}`);
      if (ioFilter) queryParams.push(`indoor_outdoor=${encodeURIComponent(ioFilter)}`);
      
      const response = await apiRequest<any>(`/cctv?${queryParams.join('&')}`);
      
      if (response.status === 'success' && response.data) {
        setCctvs(response.data.rows || []);
        setPagination(response.data.pagination || {
          total_rows: 0,
          total_pages: 1,
          page: 1,
          limit: 10,
        });
      }
    } catch (err: any) {
      setError(err.message || 'Gagal memuat data CCTV');
    } finally {
      setLoading(false);
    }
  };

  const fetchCCTVSummary = async () => {
    try {
      const [response, maintenanceResponse] = await Promise.all([
        apiRequest<any>('/cctv?page=1&limit=100'),
        apiRequest<any>('/maintenance-logs?page=1&limit=1'),
      ]);
      if (response.status === 'success' && response.data) {
        setAllCCTVs(response.data.rows || []);
      }
      if (maintenanceResponse.status === 'success' && maintenanceResponse.data) {
        setMaintenanceTotal(maintenanceResponse.data.pagination?.total_rows || 0);
      }
    } catch {
      setAllCCTVs([]);
      setMaintenanceTotal(0);
    }
  };

  useEffect(() => {
    fetchCCTVs();
  }, [page, search, statusFilter, ioFilter]);

  useEffect(() => {
    fetchCCTVSummary();
  }, []);

  // Stats
  const normalizedStatus = (status: string) => status.trim().toUpperCase();
  const isStatus = (status: string, values: string[]) => values.includes(normalizedStatus(status));
  const statusLabel = (status: string) => {
    const normalized = normalizedStatus(status);
    if (['ACTIVE', 'AKTIF'].includes(normalized)) return 'Aktif';
    if (['INACTIVE', 'NONAKTIF'].includes(normalized)) return 'Nonaktif';
    if (['MAINTENANCE', 'PEMELIHARAAN'].includes(normalized)) return 'Pemeliharaan';
    if (['BROKEN', 'RUSAK'].includes(normalized)) return 'Nonaktif';
    return status;
  };
  const totalCount = allCCTVs.length;
  const activeCount = allCCTVs.filter(c => isStatus(c.status, ['ACTIVE', 'AKTIF'])).length;
  const inactiveCount = allCCTVs.filter(c => isStatus(c.status, ['INACTIVE', 'NONAKTIF', 'BROKEN', 'RUSAK'])).length;
  const maintenanceCount = maintenanceTotal;
  const selectableCCTVs = allCCTVs.filter((cctv) => {
    if (statusFilter && normalizedStatus(cctv.status) !== statusFilter) return false;
    if (ioFilter && cctv.indoor_outdoor.trim().toUpperCase() !== ioFilter) return false;

    const normalizedSearch = search.trim().toLowerCase();
    if (!normalizedSearch) return true;

    return [cctv.nama, cctv.kode_cctv, cctv.id_channel, cctv.lokasi_area]
      .some((value) => (value || '').toLowerCase().includes(normalizedSearch));
  });

  // Handlers
  const handleOpenDetail = async (id: string) => {
    setActionLoading(`detail-${id}`);
    try {
      const response = await apiRequest<CCTV>(`/cctv/${id}`);
      if (response.data) {
        setCurrentCCTV(response.data);
        setIsDetailModalOpen(true);
      }
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Detail CCTV gagal dimuat');
    } finally {
      setActionLoading(null);
    }
  };

  const handleOpenForm = (cctv: CCTV | null = null) => {
    if (cctv) {
      setCurrentCCTV(cctv);
      setFormData({
        kode_cctv: cctv.kode_cctv,
        id_channel: cctv.id_channel || '',
        nama: cctv.nama,
        lokasi_area: cctv.lokasi_area || '',
        longitude: cctv.longitude || '',
        latitude: cctv.latitude || '',
        jenis_cctv: cctv.jenis_cctv || '',
        sumber_listrik: cctv.sumber_listrik || '',
        tipe_koneksi: cctv.tipe_koneksi || '',
        indoor_outdoor: cctv.indoor_outdoor || 'indoor',
        pemilik: cctv.pemilik || '',
        status: cctv.status,
        ip_nvr: cctv.ip_nvr || '',
        ip_camera: cctv.ip_camera || '',
        keterangan: cctv.keterangan || ''
      });
    } else {
      setCurrentCCTV(null);
      setFormData({
        kode_cctv: '',
        id_channel: '',
        nama: '',
        lokasi_area: '',
        longitude: '',
        latitude: '',
        jenis_cctv: '',
        sumber_listrik: '',
        tipe_koneksi: '',
        indoor_outdoor: 'INDOOR',
        pemilik: '',
        status: 'ACTIVE',
        ip_nvr: '',
        ip_camera: '',
        keterangan: ''
      });
    }
    setIsFormModalOpen(true);
  };

  const handleSaveCCTV = async (e: React.FormEvent) => {
    e.preventDefault();
    setActionLoading('save');
    try {
      const endpoint = currentCCTV ? `/cctv/${currentCCTV.id}` : '/cctv';
      const method = currentCCTV ? 'PUT' : 'POST';

      const response = await apiRequest<CCTV>(endpoint, {
        method,
        body: JSON.stringify(formData),
      });

      if (response.status === 'success') {
        setIsFormModalOpen(false);
        fetchCCTVs();
        fetchCCTVSummary();
      }
    } catch (err: any) {
      alert(err.message || 'Gagal menyimpan data CCTV');
    } finally {
      setActionLoading(null);
    }
  };

  const handleDeleteCCTV = async (id: string) => {
    if (!confirm('Apakah Anda yakin ingin menghapus CCTV ini? (Soft delete)')) return;
    setActionLoading(`delete-${id}`);
    try {
      const response = await apiRequest(`/cctv/${id}`, {
        method: 'DELETE'
      });
      if (response.status === 'success') {
        setSelectedIDs(prev => prev.filter(item => item !== id));
        fetchCCTVs();
        fetchCCTVSummary();
      }
    } catch (err: any) {
      alert(err.message || 'Gagal menghapus data CCTV');
    } finally {
      setActionLoading(null);
    }
  };

  // Bulk Selection Handlers
  const handleSelectAll = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.checked) {
      setSelectedIDs(selectableCCTVs.map(c => c.id));
    } else {
      setSelectedIDs([]);
    }
  };

  const handleSelectRow = (id: string, checked: boolean) => {
    if (checked) {
      setSelectedIDs(prev => prev.includes(id) ? prev : [...prev, id]);
    } else {
      setSelectedIDs(prev => prev.filter(item => item !== id));
    }
  };

  // Bulk Delete Handler (Selected or All)
  const handleBulkDelete = async (deleteAll = false) => {
    const confirmMessage = deleteAll 
      ? 'PERINGATAN: Apakah Anda yakin ingin mengosongkan seluruh data CCTV di database? Tindakan ini tidak dapat dibatalkan.'
      : `Apakah Anda yakin ingin menghapus ${selectedIDs.length} data CCTV yang terpilih?`;
      
    if (!confirm(confirmMessage)) return;

    setActionLoading(deleteAll ? 'delete-all' : 'delete-selected');
    try {
      const token = localStorage.getItem('sigap_token');
      const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3000/api';
      
      const response = await fetch(`${API_BASE_URL}/cctv/bulk-delete`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({
          ids: deleteAll ? [] : selectedIDs,
          all: deleteAll
        })
      });

      const resJson = await response.json();
      if (!response.ok) {
        throw new Error(resJson.message || 'Gagal menghapus data secara massal');
      }

      setSelectedIDs([]);
      fetchCCTVs();
      fetchCCTVSummary();
      alert(resJson.message || 'Hapus massal berhasil dilakukan!');
    } catch (err: any) {
      alert(err.message || 'Gagal melakukan hapus massal');
    } finally {
      setActionLoading(null);
    }
  };

  // Download Template CCTV
  const handleDownloadTemplate = async () => {
    try {
      const token = localStorage.getItem('sigap_token');
      const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3000/api';
      
      const response = await fetch(`${API_BASE_URL}/cctv/template`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (!response.ok) {
        throw new Error('Gagal mengunduh template');
      }

      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'template_import_cctv.xlsx';
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(url);
    } catch (err: any) {
      alert(err.message || 'Terjadi kesalahan saat mengunduh template');
    }
  };

  // Import Excel CCTV
  const handleImportExcel = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedFile) {
      setImportError('Silakan pilih berkas Excel terlebih dahulu.');
      return;
    }

    setImportLoading(true);
    setImportError('');
    setImportSuccess('');

    try {
      const token = localStorage.getItem('sigap_token');
      const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3000/api';
      
      const dataPayload = new FormData();
      dataPayload.append('file', selectedFile);

      const response = await fetch(`${API_BASE_URL}/cctv/import`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`
        },
        body: dataPayload
      });

      const resJson = await response.json();
      if (!response.ok) {
        throw new Error(resJson.message || 'Gagal mengimpor berkas Excel');
      }

      setImportSuccess(resJson.message || 'Data CCTV berhasil diimpor!');
      setSelectedFile(null);
      fetchCCTVs();
      fetchCCTVSummary();
    } catch (err: any) {
      setImportError(err.message || 'Terjadi kesalahan saat mengimpor berkas');
    } finally {
      setImportLoading(false);
    }
  };

  // Handle scanned QR code text
  const handleDecodedQR = async (decodedText: string) => {
    // Hentikan camera scan agar tidak membaca berkali-kali
    if (scannerRef.current && scannerRef.current.isScanning) {
      await scannerRef.current.stop().catch((err: any) => console.error(err));
      setCameraActive(false);
    }
    
    setScanLoading(true);
    setScanError('');
    setScanPayload(decodedText);
    setScannedResult(null);

    try {
      const response = await apiRequest<CCTV>('/cctv/scan', {
        method: 'POST',
        body: JSON.stringify({ qr_payload: decodedText.trim() })
      });

      if (response.status === 'success' && response.data) {
        setScannedResult(response.data);
      }
    } catch (err: any) {
      setScanError(err.message || 'QR Payload tidak valid atau gagal didekripsi');
    } finally {
      setScanLoading(false);
    }
  };

  // Start webcam scanner
  const startCameraScanner = () => {
    setScanError('');
    setScannedResult(null);
    setScanPayload('');
    
    setTimeout(() => {
      import('html5-qrcode').then((module) => {
        if (scannerRef.current && scannerRef.current.isScanning) {
          scannerRef.current.stop().then(() => {
            initializeScanner(module);
          }).catch(() => {
            initializeScanner(module);
          });
        } else {
          initializeScanner(module);
        }
      }).catch(err => {
        setScanError('Gagal memuat modul kamera: ' + err.message);
      });
    }, 300);
  };

  const initializeScanner = (module: any) => {
    try {
      const html5Qr = new module.Html5Qrcode("reader");
      scannerRef.current = html5Qr;

      html5Qr.start(
        { facingMode: "environment" },
        {
          fps: 10,
          qrbox: (width: number, height: number) => {
            const size = Math.min(width, height) * 0.7;
            return { width: size, height: size };
          }
        },
        (decodedText: string) => {
          handleDecodedQR(decodedText);
        },
        () => {
          // Abaikan log frame
        }
      ).then(() => {
        setCameraActive(true);
      }).catch((err: any) => {
        console.error("Camera start error:", err);
        setScanError("Gagal mengakses kamera. Pastikan izin kamera telah diberikan atau gunakan input manual.");
        setCameraActive(false);
      });
    } catch (e: any) {
      console.error(e);
      setScanError("Inisialisasi pemindai gagal: " + e.message);
    }
  };

  // Stop camera scanner
  const stopCameraScanner = async () => {
    if (scannerRef.current && scannerRef.current.isScanning) {
      await scannerRef.current.stop().catch((err: any) => console.error("Gagal stop camera:", err));
    }
    setCameraActive(false);
  };

  // Effect to handle modal open/close scanner trigger
  useEffect(() => {
    if (isScanModalOpen && !scannedResult && !showManualInput) {
      startCameraScanner();
    } else {
      stopCameraScanner();
    }

    return () => {
      if (scannerRef.current && scannerRef.current.isScanning) {
        scannerRef.current.stop().catch((err: any) => console.error("Cleanup error:", err));
      }
    };
  }, [isScanModalOpen, scannedResult, showManualInput]);

  const handleCloseScanModal = async () => {
    await stopCameraScanner();
    setIsScanModalOpen(false);
    setScannedResult(null);
    setScanPayload('');
    setScanError('');
    setShowManualInput(false);
  };

  // Manual Scan Input Handler (Backup)
  const handleScanCCTV = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!scanPayload) {
      setScanError('QR payload tidak boleh kosong.');
      return;
    }

    setScanLoading(true);
    setScanError('');
    setScannedResult(null);

    try {
      const response = await apiRequest<CCTV>('/cctv/scan', {
        method: 'POST',
        body: JSON.stringify({ qr_payload: scanPayload.trim() })
      });

      if (response.status === 'success' && response.data) {
        setScannedResult(response.data);
      }
    } catch (err: any) {
      setScanError(err.message || 'QR Payload tidak valid atau gagal didekripsi');
    } finally {
      setScanLoading(false);
    }
  };

  // Print QR Code
  const handlePrintQR = () => {
    if (!currentCCTV) return;
    const printWindow = window.open('', '_blank');
    if (!printWindow) return;

    const qrUrl = `https://api.qrserver.com/v1/create-qr-code/?size=250x250&data=${encodeURIComponent(currentCCTV.qr_payload)}`;
    const logoUrl = `${window.location.origin}/nusantara-logo.png`;

    printWindow.document.write(`
      <html>
        <head>
          <title>Cetak QR Code - ${currentCCTV.kode_cctv}</title>
          <style>
            @page { size: 100mm 38mm; margin: 0; }
            * { box-sizing: border-box; }
            body {
              font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
              display: flex;
              align-items: center;
              justify-content: center;
              width: 100mm;
              height: 38mm;
              margin: 0;
              background-color: white;
              color: #203f25;
            }
            .label {
              display: grid;
              grid-template-columns: 29mm 35mm 1fr;
              align-items: center;
              width: 100mm;
              height: 38mm;
              padding: 2.5mm;
              border: 0.4mm solid #d7aa63;
              border-radius: 2mm;
              background: white;
              overflow: hidden;
            }
            .brand {
              height: 31mm;
              display: flex;
              align-items: center;
              justify-content: center;
              padding-right: 2mm;
              border-right: 0.3mm solid #467235;
            }
            .brand img {
              width: 25mm;
              height: 30mm;
              object-fit: contain;
            }
            .qr {
              width: 31mm;
              height: 31mm;
              margin: 0 auto;
            }
            .details {
              min-width: 0;
              padding-left: 1mm;
              line-height: 1.25;
            }
            .name {
              margin: 0 0 1.5mm;
              font-size: 9pt;
              font-weight: 700;
            }
            .meta {
              margin: 0.7mm 0;
              font-size: 7pt;
              color: #62705a;
              word-break: break-word;
            }
            .code {
              color: #467235;
              font-weight: 700;
            }
          </style>
        </head>
        <body>
          <div class="label">
            <div class="brand">
              <img src="${logoUrl}" alt="Nusantara" />
            </div>
            <img class="qr" src="${qrUrl}" alt="QR Code" />
            <div class="details">
              <p class="name">${currentCCTV.nama}</p>
              <p class="meta code">${currentCCTV.kode_cctv}</p>
              <p class="meta">${currentCCTV.lokasi_area || '-'}</p>
            </div>
          </div>
          <script>
            window.onload = function() {
              window.print();
              setTimeout(function() { window.close(); }, 500);
            }
          </script>
        </body>
      </html>
    `);
    printWindow.document.close();
  };

  return (
    <DashboardLayout>
      <div className="space-y-8 animate-fadeIn">
        {/* Header */}
        <div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
          <h1 className="shrink-0 whitespace-nowrap text-3xl font-bold tracking-tight">Aset CCTV</h1>
          <div className="flex flex-wrap gap-2 xl:justify-end">
            <button
              onClick={() => setIsScanModalOpen(true)}
              className="flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl border border-border bg-card hover:bg-background text-foreground font-semibold active:scale-[0.98] transition-all cursor-pointer text-sm shadow-sm"
            >
              <Scan className="h-4.5 w-4.5 text-primary" />
              <span>Scanner QR</span>
            </button>
            {isAdmin && (
              <button
                onClick={() => setIsImportModalOpen(true)}
                className="flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl border border-border bg-card hover:bg-background text-foreground font-semibold active:scale-[0.98] transition-all cursor-pointer text-sm shadow-sm"
              >
                <Upload className="h-4.5 w-4.5 text-primary" />
                <span>Import Excel</span>
              </button>
            )}
            {isAdmin && cctvs.length > 0 && (
              <button
                onClick={() => handleBulkDelete(true)}
                disabled={actionLoading === 'delete-all'}
                className="order-4 flex items-center justify-center gap-2 rounded-xl border border-red-500/25 bg-red-500/5 px-4 py-2.5 text-sm font-semibold text-red-600 shadow-sm transition-all hover:bg-red-500/10 active:scale-[0.98] disabled:cursor-wait disabled:opacity-60 dark:text-red-400"
                title="Hapus Semua Data CCTV"
              >
                {actionLoading === 'delete-all' ? <LoaderCircle className="h-4.5 w-4.5 animate-spin" /> : <Trash2 className="h-4.5 w-4.5" />}
                <span>{actionLoading === 'delete-all' ? 'Menghapus...' : 'Kosongkan Data'}</span>
              </button>
            )}
            {isAdmin && (
              <button
                onClick={() => handleOpenForm()}
                className="order-3 flex items-center justify-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white shadow-md shadow-primary/20 transition-all hover:bg-primary-hover active:scale-[0.98]"
              >
                <Plus className="h-5 w-5" />
                <span>Tambah CCTV</span>
              </button>
            )}
          </div>
        </div>

        {/* Stats Row */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          {[
            {
              label: 'Total CCTV',
              value: totalCount,
              icon: Camera,
              border: 'border-t-blue-500',
              tone: 'bg-blue-500/10 text-blue-600 dark:text-blue-400',
            },
            {
              label: 'Aktif',
              value: activeCount,
              icon: ShieldCheck,
              border: 'border-t-success',
              tone: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
            },
            {
              label: 'Nonaktif',
              value: inactiveCount,
              icon: Activity,
              border: 'border-t-error',
              tone: 'bg-red-500/10 text-red-600 dark:text-red-400',
            },
            {
              label: 'Pemeliharaan',
              value: maintenanceCount,
              icon: Wrench,
              border: 'border-t-warning',
              tone: 'bg-amber-500/10 text-amber-600 dark:text-amber-400',
            },
          ].map(({ label, value, icon: Icon, tone, border }) => (
            <div key={label} className={`rounded-2xl border border-t-4 border-border bg-card p-5 shadow-sm ${border}`}>
              <div className="flex items-start justify-between">
                <div>
                  <p className="text-xs font-semibold text-muted">{label}</p>
                  <p className="mt-2 text-3xl font-semibold tracking-[-0.025em]">{value}</p>
                </div>
                <div className={`rounded-xl p-2.5 ${tone}`}>
                  <Icon className="h-5 w-5" />
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* Filter & Search Controls */}
        <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4 p-5 rounded-2xl bg-card border border-border bg-card/60 backdrop-blur-md">
          {/* Search */}
          <div className="relative flex-1 max-w-md">
            <div className="absolute inset-y-0 left-0 flex items-center pl-4 pointer-events-none text-gray-400">
              <Search className="h-5 w-5" />
            </div>
            <input
              type="text"
              placeholder="Cari CCTV"
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setSelectedIDs([]);
                setPage(1);
              }}
              className="w-full pl-11 pr-4 py-2.5 rounded-xl border border-input-border bg-input text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary text-sm transition-all"
            />
          </div>

          {/* Filters */}
          <div className="flex flex-wrap gap-2.5">
            <select
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value);
                setSelectedIDs([]);
                setPage(1);
              }}
              className="px-4 py-2 rounded-xl text-xs font-semibold border border-border bg-background text-gray-500 hover:text-foreground dark:text-gray-400 focus:outline-none cursor-pointer"
            >
              <option value="">Semua Status</option>
              <option value="ACTIVE">Aktif</option>
              <option value="INACTIVE">Nonaktif</option>
            </select>

            <select
              value={ioFilter}
              onChange={(e) => {
                setIoFilter(e.target.value);
                setSelectedIDs([]);
                setPage(1);
              }}
              className="px-4 py-2 rounded-xl text-xs font-semibold border border-border bg-background text-gray-500 hover:text-foreground dark:text-gray-400 focus:outline-none cursor-pointer"
            >
              <option value="">Semua Status</option>
              <option value="INDOOR">Indoor</option>
              <option value="OUTDOOR">Outdoor</option>
            </select>
          </div>
        </div>

        {selectedIDs.length > 0 && isAdmin && (
          <div className="flex flex-col gap-3 rounded-xl border border-border bg-card px-4 py-3 shadow-sm sm:flex-row sm:items-center sm:justify-between">
            <p className="text-sm font-semibold">
              <span className="text-primary">{selectedIDs.length}</span> CCTV terpilih
            </p>
            <button
              onClick={() => handleBulkDelete(false)}
              disabled={actionLoading === 'delete-selected'}
              className="inline-flex items-center justify-center gap-2 rounded-lg bg-red-500 px-4 py-2 text-xs font-bold text-white transition-colors hover:bg-red-600 disabled:cursor-wait disabled:opacity-60"
            >
              {actionLoading === 'delete-selected' ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
              {actionLoading === 'delete-selected' ? 'Menghapus...' : 'Hapus Terpilih'}
            </button>
          </div>
        )}

        {/* CCTV Master List Table */}
        {loading ? (
          <div className="flex justify-center items-center py-20 bg-card rounded-2xl border border-border">
            <div className="h-8 w-8 animate-spin rounded-full border-3 border-primary border-t-transparent"></div>
          </div>
        ) : cctvs.length === 0 ? (
          <div className="text-center py-16 bg-card rounded-2xl border border-border flex flex-col items-center justify-center">
            <Camera className="h-12 w-12 text-gray-300 dark:text-gray-700 mb-3" />
            <h3 className="text-base font-semibold">Tidak Ada Perangkat CCTV Ditemukan</h3>
          </div>
        ) : (
          <div className="overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="border-b border-border bg-background/50 text-xs font-bold text-gray-400 uppercase tracking-wider">
                    {isAdmin && (
                      <th className="py-4 px-6 w-12 text-center">
                        <input
                          type="checkbox"
                          onChange={handleSelectAll}
                          checked={selectableCCTVs.length > 0 && selectableCCTVs.every((cctv) => selectedIDs.includes(cctv.id))}
                          aria-label={`Pilih semua ${selectableCCTVs.length} CCTV hasil filter`}
                          className="rounded border-gray-300 text-primary focus:ring-primary h-4 w-4 cursor-pointer"
                        />
                      </th>
                    )}
                    <th className="py-4 px-6">Identitas CCTV</th>
                    <th className="py-4 px-6">Spesifikasi Koneksi</th>
                    <th className="py-4 px-6">Semua Status</th>
                    <th className="py-4 px-6">Sumber Daya & Lokasi</th>
                    <th className="py-4 px-6">Status</th>
                    <th className="py-4 px-6 text-right">Aksi</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border text-sm">
                  {cctvs.map((cctv) => (
                    <tr key={cctv.id} className={`hover:bg-background/35 transition-colors ${selectedIDs.includes(cctv.id) ? 'bg-primary/8' : ''}`}>
                      {isAdmin && (
                        <td className="py-4.5 px-6 w-12 text-center">
                          <input
                            type="checkbox"
                            onChange={(e) => handleSelectRow(cctv.id, e.target.checked)}
                            checked={selectedIDs.includes(cctv.id)}
                            className="rounded border-gray-300 text-primary focus:ring-primary h-4 w-4 cursor-pointer"
                          />
                        </td>
                      )}
                      <td className="py-4.5 px-6">
                        <div className="flex items-center gap-3">
                          <button
                            onClick={() => {
                              setCurrentCCTV(cctv);
                              setIsQrModalOpen(true);
                            }}
                            className="flex h-12 w-12 shrink-0 items-center justify-center overflow-hidden rounded-lg border border-primary/15 bg-white p-1.5 shadow-sm transition hover:border-primary/35"
                            title="Tampilkan QR Code"
                          >
                            {cctv.qr_payload ? (
                              <img
                                src={`https://api.qrserver.com/v1/create-qr-code/?size=96x96&data=${encodeURIComponent(cctv.qr_payload)}`}
                                alt={`QR Code ${cctv.kode_cctv}`}
                                className="h-full w-full object-contain"
                                loading="lazy"
                              />
                            ) : (
                              <span className="text-[9px] font-bold text-muted">QR</span>
                            )}
                          </button>
                          <div>
                            <h4 className="font-semibold">{cctv.nama}</h4>
                            <div className="flex items-center gap-2 mt-0.5">
                              <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-accent/20 text-[#775800] dark:text-accent uppercase tracking-wider font-mono">{cctv.kode_cctv}</span>
                              {cctv.id_channel && <span className="text-xs text-gray-400">CH: {cctv.id_channel}</span>}
                            </div>
                          </div>
                        </div>
                      </td>
                      <td className="py-4.5 px-6 font-mono text-xs text-gray-500 dark:text-gray-400 space-y-1">
                        <div><span className="text-[10px] text-gray-400">CAM:</span> {cctv.ip_camera || '-'}</div>
                        {cctv.ip_nvr && <div><span className="text-[10px] text-gray-400">NVR:</span> {cctv.ip_nvr}</div>}
                      </td>
                      <td className="py-4.5 px-6">
                        <span className={`inline-block text-xs font-semibold px-2 py-0.5 rounded border capitalize ${
                          cctv.indoor_outdoor.toLowerCase() === 'indoor'
                            ? 'bg-purple-500/5 text-purple-500 border-purple-500/10'
                            : 'bg-orange-500/5 text-orange-500 border-orange-500/10'
                        }`}>
                          {cctv.indoor_outdoor}
                        </span>
                      </td>
                      <td className="py-4.5 px-6 space-y-1">
                        <div className="flex items-center gap-1.5 text-gray-600 dark:text-gray-300">
                          <MapPin className="h-3.5 w-3.5 shrink-0 text-gray-400" />
                          <span className="truncate max-w-[170px]" title={cctv.lokasi_area}>{cctv.lokasi_area || '-'}</span>
                        </div>
                        {cctv.sumber_listrik && (
                          <div className="text-[11px] text-gray-400 flex items-center gap-1">
                            <Cpu className="h-3 w-3" />
                            <span>Listrik: {cctv.sumber_listrik}</span>
                          </div>
                        )}
                      </td>
                      <td className="py-4.5 px-6">
                        <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold capitalize ${
                          isStatus(cctv.status, ['ACTIVE', 'AKTIF'])
                            ? 'border border-emerald-500/25 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                            : isStatus(cctv.status, ['INACTIVE', 'NONAKTIF', 'BROKEN', 'RUSAK'])
                            ? 'border border-red-500/25 bg-red-500/10 text-red-600 dark:text-red-400'
                            : isStatus(cctv.status, ['MAINTENANCE', 'PEMELIHARAAN'])
                            ? 'border border-amber-500/25 bg-amber-500/10 text-amber-600 dark:text-amber-400'
                            : 'bg-gray-500/10 text-gray-500 border border-gray-500/20'
                        }`}>
                          <span className={`h-1.5 w-1.5 rounded-full ${
                            isStatus(cctv.status, ['ACTIVE', 'AKTIF']) ? 'bg-emerald-500' : isStatus(cctv.status, ['INACTIVE', 'NONAKTIF', 'BROKEN', 'RUSAK']) ? 'bg-red-500' : isStatus(cctv.status, ['MAINTENANCE', 'PEMELIHARAAN']) ? 'bg-amber-500' : 'bg-gray-500'
                          }`} />
                          {statusLabel(cctv.status)}
                        </span>
                      </td>
                      <td className="py-4.5 px-6 text-right">
                        <div className="flex items-center justify-end gap-2">
                          <button
                            onClick={() => handleOpenDetail(cctv.id)}
                            disabled={actionLoading === `detail-${cctv.id}`}
                            className="p-2 rounded-lg border border-border bg-card hover:border-gray-500/20 hover:bg-gray-500/5 text-gray-500 dark:text-gray-400 transition-all cursor-pointer"
                            title="Detail CCTV"
                          >
                            {actionLoading === `detail-${cctv.id}` ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Eye className="h-4 w-4" />}
                          </button>
                          {isAdmin && (
                            <>
                              <button
                                onClick={() => handleOpenForm(cctv)}
                                className="p-2 rounded-lg border border-border bg-card hover:border-primary/25 hover:bg-primary/8 hover:text-primary text-gray-500 dark:text-gray-400 transition-all cursor-pointer"
                                title="Edit CCTV"
                              >
                                <Edit2 className="h-4 w-4" />
                              </button>
                              <button
                                onClick={() => handleDeleteCCTV(cctv.id)}
                                disabled={actionLoading === `delete-${cctv.id}`}
                                className="p-2 rounded-lg border border-border bg-card hover:border-red-500/20 hover:bg-red-500/5 hover:text-red-500 text-gray-500 dark:text-gray-400 transition-all cursor-pointer"
                                title="Hapus CCTV"
                              >
                                {actionLoading === `delete-${cctv.id}` ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
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
            <Pagination
              page={pagination.page}
              limit={pagination.limit}
              totalRows={pagination.total_rows}
              totalPages={pagination.total_pages}
              onPageChange={(nextPage) => {
                setPage(nextPage);
              }}
              itemLabel="CCTV"
            />
          </div>
        )}

        {/* CCTV Form Modal (Create/Edit) */}
        {isFormModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fadeIn">
            <div className="relative w-full max-w-2xl bg-card border border-border rounded-2xl p-6 shadow-2xl space-y-4 max-h-[90vh] overflow-y-auto">
              <div className="flex items-center justify-between border-b border-border pb-4">
                <h3 className="text-xl font-bold">{currentCCTV ? 'Edit Data CCTV' : 'Tambah Perangkat CCTV'}</h3>
                <button onClick={() => setIsFormModalOpen(false)} className="p-1 rounded-lg border hover:bg-background text-gray-400 hover:text-foreground cursor-pointer">
                  <X className="h-5 w-5" />
                </button>
              </div>
              
              <form onSubmit={handleSaveCCTV} className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-gray-400">Kode CCTV *</label>
                    <input
                      type="text"
                      required
                      value={formData.kode_cctv}
                      onChange={(e) => setFormData({...formData, kode_cctv: e.target.value})}
                      placeholder="Contoh: THD-01"
                      className="w-full px-3 py-2 rounded-lg border border-input-border bg-input text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                    />
                  </div>
                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-gray-400">ID Channel</label>
                    <input
                      type="text"
                      value={formData.id_channel}
                      onChange={(e) => setFormData({...formData, id_channel: e.target.value})}
                      placeholder="Contoh: 1"
                      className="w-full px-3 py-2 rounded-lg border border-input-border bg-input text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                    />
                  </div>
                  <div className="space-y-1.5 md:col-span-2">
                    <label className="text-xs font-semibold text-gray-400">Nama CCTV *</label>
                    <input
                      type="text"
                      required
                      value={formData.nama}
                      onChange={(e) => setFormData({...formData, nama: e.target.value})}
                      placeholder="Contoh: CCTV Bundaran HI Utama"
                      className="w-full px-3 py-2 rounded-lg border border-input-border bg-input text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                    />
                  </div>
                  <div className="space-y-1.5 md:col-span-2">
                    <label className="text-xs font-semibold text-gray-400">Lokasi Area</label>
                    <input
                      type="text"
                      value={formData.lokasi_area}
                      onChange={(e) => setFormData({...formData, lokasi_area: e.target.value})}
                      placeholder="Contoh: Kawasan Bundaran HI, Jakarta Pusat"
                      className="w-full px-3 py-2 rounded-lg border border-input-border bg-input text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                    />
                  </div>
                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-gray-400">Longitude</label>
                    <input
                      type="text"
                      value={formData.longitude}
                      onChange={(e) => setFormData({...formData, longitude: e.target.value})}
                      placeholder="Contoh: 106.8231"
                      className="w-full px-3 py-2 rounded-lg border border-input-border bg-input text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                    />
                  </div>
                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-gray-400">Latitude</label>
                    <input
                      type="text"
                      value={formData.latitude}
                      onChange={(e) => setFormData({...formData, latitude: e.target.value})}
                      placeholder="Contoh: -6.1949"
                      className="w-full px-3 py-2 rounded-lg border border-input-border bg-input text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                    />
                  </div>
                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-gray-400">Jenis CCTV</label>
                    <input
                      type="text"
                      value={formData.jenis_cctv}
                      onChange={(e) => setFormData({...formData, jenis_cctv: e.target.value})}
                      placeholder="Contoh: PTZ Dome"
                      className="w-full px-3 py-2 rounded-lg border border-input-border bg-input text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                    />
                  </div>
                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-gray-400">Sumber Listrik</label>
                    <input
                      type="text"
                      value={formData.sumber_listrik}
                      onChange={(e) => setFormData({...formData, sumber_listrik: e.target.value})}
                      placeholder="Contoh: Solar Panel / PLN"
                      className="w-full px-3 py-2 rounded-lg border border-input-border bg-input text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                    />
                  </div>
                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-gray-400">Tipe Koneksi</label>
                    <input
                      type="text"
                      value={formData.tipe_koneksi}
                      onChange={(e) => setFormData({...formData, tipe_koneksi: e.target.value})}
                      placeholder="Contoh: FO / 4G LTE"
                      className="w-full px-3 py-2 rounded-lg border border-input-border bg-input text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                    />
                  </div>
                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-gray-400">Indoor / Outdoor</label>
                    <select
                      value={formData.indoor_outdoor}
                      onChange={(e) => setFormData({...formData, indoor_outdoor: e.target.value})}
                      className="w-full px-3 py-2 rounded-lg border border-input-border bg-input text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary cursor-pointer"
                    >
                      <option value="INDOOR">Indoor</option>
                      <option value="OUTDOOR">Outdoor</option>
                    </select>
                  </div>
                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-gray-400">Pemilik / Pengelola</label>
                    <input
                      type="text"
                      value={formData.pemilik}
                      onChange={(e) => setFormData({...formData, pemilik: e.target.value})}
                      placeholder="Contoh: Dishub / Polda"
                      className="w-full px-3 py-2 rounded-lg border border-input-border bg-input text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                    />
                  </div>
                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-gray-400">Status Alat *</label>
                    <select
                      value={formData.status}
                      onChange={(e) => setFormData({...formData, status: e.target.value})}
                      className="w-full px-3 py-2 rounded-lg border border-input-border bg-input text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary cursor-pointer"
                    >
                      <option value="ACTIVE">Aktif</option>
                      <option value="INACTIVE">Nonaktif</option>
                      <option value="MAINTENANCE">Pemeliharaan</option>
                    </select>
                  </div>
                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-gray-400">IP NVR</label>
                    <input
                      type="text"
                      value={formData.ip_nvr}
                      onChange={(e) => setFormData({...formData, ip_nvr: e.target.value})}
                      placeholder="Contoh: 192.168.1.100"
                      className="w-full px-3 py-2 rounded-lg border border-input-border bg-input text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                    />
                  </div>
                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-gray-400">IP Camera</label>
                    <input
                      type="text"
                      value={formData.ip_camera}
                      onChange={(e) => setFormData({...formData, ip_camera: e.target.value})}
                      placeholder="Contoh: 192.168.1.101"
                      className="w-full px-3 py-2 rounded-lg border border-input-border bg-input text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                    />
                  </div>
                  <div className="space-y-1.5 md:col-span-2">
                    <label className="text-xs font-semibold text-gray-400">Keterangan Tambahan</label>
                    <textarea
                      value={formData.keterangan}
                      onChange={(e) => setFormData({...formData, keterangan: e.target.value})}
                      placeholder="Tuliskan keterangan detail CCTV di sini..."
                      rows={3}
                      className="w-full px-3 py-2 rounded-lg border border-input-border bg-input text-foreground text-sm focus:outline-none focus:ring-1 focus:ring-primary resize-none"
                    />
                  </div>
                </div>

                <div className="flex justify-end gap-2 border-t border-border pt-4 mt-2">
                  <button
                    type="button"
                    onClick={() => setIsFormModalOpen(false)}
                    className="px-4 py-2 rounded-lg border bg-card text-foreground font-semibold text-sm hover:bg-background cursor-pointer"
                  >
                    Batal
                  </button>
                  <button
                    type="submit"
                    disabled={actionLoading === 'save'}
                    className="flex items-center gap-2 px-5 py-2 rounded-lg bg-primary hover:bg-primary/95 text-white font-semibold text-sm cursor-pointer shadow-md shadow-primary/15 disabled:cursor-wait disabled:opacity-65"
                  >
                    {actionLoading === 'save' && <LoaderCircle className="h-4 w-4 animate-spin" />}
                    {actionLoading === 'save' ? 'Menyimpan...' : 'Simpan'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* Scan QR Code Modal */}
        {isScanModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fadeIn">
            <div className="relative w-full max-w-md space-y-3 rounded-2xl border border-border bg-card p-5 shadow-2xl">
              <div className="flex items-center justify-between">
                <h3 className="text-base font-bold">Scanner QR CCTV</h3>
                <button onClick={handleCloseScanModal} className="p-1 rounded-lg border hover:bg-background text-gray-400 hover:text-foreground cursor-pointer">
                  <X className="h-5 w-5" />
                </button>
              </div>

              {!scannedResult ? (
                <div className="space-y-3">
                  {/* Toggle Mode Button */}
                  <div className="flex justify-end">
                    <button
                      type="button"
                      onClick={() => {
                        setShowManualInput(!showManualInput);
                        setScanError('');
                      }}
                      className="rounded-lg border border-border bg-background px-3 py-1.5 text-xs font-semibold text-muted transition-colors hover:text-foreground"
                    >
                      {showManualInput ? 'Gunakan kamera' : 'Input manual'}
                    </button>
                  </div>

                  {!showManualInput ? (
                    /* CAMERA SCANNER VIEW */
                    <div className="text-center">
                      <div className="relative mx-auto flex min-h-[220px] w-full max-w-sm items-center justify-center overflow-hidden rounded-xl border border-border bg-black">
                        {/* html5-qrcode video container */}
                        <div id="reader" className="w-full"></div>
                        
                        {/* Custom overlay laser scan animation */}
                        {cameraActive && (
                          <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-transparent via-blue-500 to-transparent animate-[scan_2s_infinite] shadow-lg shadow-blue-500/50 pointer-events-none z-10"></div>
                        )}
                        
                        {!cameraActive && !scanError && (
                          <div className="absolute inset-0 z-20 flex flex-col items-center justify-center gap-3 bg-card/90 p-6 text-center">
                            <div className="h-7 w-7 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
                            <span className="text-xs font-medium text-muted">Menghubungkan kamera</span>
                          </div>
                        )}
                      </div>
                    </div>
                  ) : (
                    /* MANUAL TEXT INPUT VIEW */
                    <form onSubmit={handleScanCCTV} className="space-y-3">
                      <div className="space-y-1.5">
                        <label className="text-xs font-semibold text-muted">Payload QR</label>
                        <textarea
                          required
                          value={scanPayload}
                          onChange={(e) => setScanPayload(e.target.value)}
                          placeholder="Tempel payload QR"
                          rows={3}
                          className="w-full px-3 py-2 rounded-lg border border-input-border bg-input font-mono text-xs focus:outline-none focus:ring-1 focus:ring-primary resize-none"
                        />
                      </div>

                      <div className="flex justify-end gap-2">
                        <button
                          type="button"
                          onClick={handleCloseScanModal}
                          className="px-4 py-2 rounded-lg border bg-card text-foreground font-semibold text-sm hover:bg-background cursor-pointer"
                        >
                          Batal
                        </button>
                        <button
                          type="submit"
                          disabled={scanLoading}
                          className="px-5 py-2 rounded-lg bg-primary hover:bg-primary/95 text-white font-semibold text-sm cursor-pointer disabled:opacity-50"
                        >
                          {scanLoading ? 'Memeriksa...' : 'Identifikasi'}
                        </button>
                      </div>
                    </form>
                  )}

                  {scanError && (
                    <div className="p-3 rounded-lg bg-red-500/10 text-red-500 border border-red-500/20 text-xs flex items-center gap-2">
                      <AlertTriangle className="h-4 w-4 shrink-0" />
                      <span>{scanError}</span>
                    </div>
                  )}

                  {!showManualInput && (
                    <div className="flex justify-end">
                      <button
                        type="button"
                        onClick={handleCloseScanModal}
                        className="px-4 py-2 rounded-lg border bg-card text-foreground font-semibold text-sm hover:bg-background cursor-pointer"
                      >
                        Tutup
                      </button>
                    </div>
                  )}
                </div>
              ) : (
                <div className="space-y-4">
                  <div className="rounded-lg border border-emerald-500/20 bg-emerald-500/10 p-3 text-xs font-medium text-emerald-600 dark:text-emerald-400">
                    CCTV berhasil diidentifikasi.
                  </div>

                  <div className="p-5 rounded-xl bg-background border border-border space-y-3">
                    <div className="flex justify-between items-start border-b border-border pb-3">
                      <div>
                        <h4 className="font-bold text-base text-foreground">{scannedResult.nama}</h4>
                        <span className="text-xs font-mono font-bold mt-0.5 px-2 py-0.5 bg-blue-500/10 text-blue-500 rounded inline-block uppercase tracking-wider">{scannedResult.kode_cctv}</span>
                      </div>
                      <span className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold capitalize ${
                        isStatus(scannedResult.status, ['ACTIVE', 'AKTIF'])
                          ? 'bg-emerald-500/10 text-emerald-500 border border-emerald-500/20' 
                          : 'bg-red-500/10 text-red-500 border border-red-500/20'
                      }`}>
                        {scannedResult.status}
                      </span>
                    </div>

                    <div className="grid grid-cols-2 gap-3 text-xs">
                      <div>
                        <span className="text-gray-400 block">ID Channel</span>
                        <span className="font-medium text-foreground">{scannedResult.id_channel || '-'}</span>
                      </div>
                      <div>
                        <span className="text-gray-400 block">Jenis CCTV</span>
                        <span className="font-medium text-foreground">{scannedResult.jenis_cctv || '-'}</span>
                      </div>
                      <div>
                        <span className="text-gray-400 block">Lokasi Area</span>
                        <span className="font-medium text-foreground">{scannedResult.lokasi_area || '-'}</span>
                      </div>
                      <div>
                        <span className="text-gray-400 block">Koneksi</span>
                        <span className="font-medium text-foreground">{scannedResult.tipe_koneksi || '-'}</span>
                      </div>
                      <div>
                        <span className="text-gray-400 block">IP Camera</span>
                        <span className="font-medium text-foreground font-mono">{scannedResult.ip_camera || '-'}</span>
                      </div>
                      <div>
                        <span className="text-gray-400 block">Indoor/Outdoor</span>
                        <span className="font-medium text-foreground capitalize">{scannedResult.indoor_outdoor || '-'}</span>
                      </div>
                    </div>
                  </div>

                  <div className="flex justify-end gap-2 border-t border-border pt-4">
                    <button
                      type="button"
                      onClick={() => {
                        setScannedResult(null);
                        if (!showManualInput) startCameraScanner();
                      }}
                      className="px-4 py-2 rounded-lg border bg-card text-foreground font-semibold text-sm hover:bg-background cursor-pointer"
                    >
                      Scan Lagi
                    </button>
                    <button
                      type="button"
                      onClick={handleCloseScanModal}
                      className="px-5 py-2 rounded-lg bg-primary text-white font-semibold text-sm cursor-pointer"
                    >
                      Selesai
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Import Excel Modal */}
        {isImportModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fadeIn">
            <div className="relative w-full max-w-md space-y-3 rounded-2xl border border-border bg-card p-5 shadow-2xl">
              <div className="flex items-center justify-between">
                <h3 className="text-base font-bold">Impor Excel</h3>
                <button onClick={() => { setIsImportModalOpen(false); setSelectedFile(null); setImportError(''); setImportSuccess(''); }} className="p-1 rounded-lg border hover:bg-background text-gray-400 hover:text-foreground cursor-pointer">
                  <X className="h-5 w-5" />
                </button>
              </div>

              <form onSubmit={handleImportExcel} className="space-y-3">
                <div className="flex items-center justify-between rounded-lg border border-border px-3 py-2.5 text-xs">
                  <p className="font-medium text-muted">Template data CCTV</p>
                  <button
                    type="button"
                    onClick={handleDownloadTemplate}
                    className="shrink-0 font-semibold text-primary hover:underline"
                  >
                    Unduh .xlsx
                  </button>
                </div>

                <div>
                  <div className="flex items-center justify-center w-full">
                    <label className="flex h-28 w-full cursor-pointer flex-col items-center justify-center rounded-lg border border-dashed border-input-border bg-input px-4 text-center transition-colors hover:border-primary/50">
                      <div className="min-w-0">
                        <p className="truncate text-xs font-semibold text-foreground">
                          {selectedFile ? selectedFile.name : 'Klik untuk memilih berkas Excel'}
                        </p>
                        <p className="mt-0.5 text-[10px] text-muted">Berkas .xlsx</p>
                      </div>
                      <span className="mt-2 shrink-0 rounded-md border border-border bg-card px-3 py-1 text-[11px] font-semibold">Pilih file</span>
                      <input 
                        type="file" 
                        accept=".xlsx" 
                        className="hidden" 
                        onChange={(e) => {
                          if (e.target.files && e.target.files.length > 0) {
                            setSelectedFile(e.target.files[0]);
                          }
                        }}
                      />
                    </label>
                  </div>
                </div>

                {importError && (
                  <div className="rounded-lg border border-red-500/20 bg-red-500/10 p-3 text-xs text-red-500">{importError}</div>
                )}

                {importSuccess && (
                  <div className="rounded-lg border border-emerald-500/20 bg-emerald-500/10 p-3 text-xs text-emerald-600 dark:text-emerald-400">{importSuccess}</div>
                )}

                <div className="flex justify-end gap-2 pt-1">
                  <button
                    type="button"
                    onClick={() => { setIsImportModalOpen(false); setSelectedFile(null); setImportError(''); setImportSuccess(''); }}
                    className="px-4 py-2 rounded-lg border bg-card text-foreground font-semibold text-sm hover:bg-background cursor-pointer"
                  >
                    Tutup
                  </button>
                  <button
                    type="submit"
                    disabled={importLoading || !selectedFile}
                    className="px-5 py-2 rounded-lg bg-primary hover:bg-primary/95 text-white font-semibold text-sm cursor-pointer disabled:opacity-50"
                  >
                    {importLoading ? 'Memproses...' : 'Impor Sekarang'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* CCTV QR Code Detail Modal */}
        {isQrModalOpen && currentCCTV && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fadeIn">
            <div className="relative w-full max-w-2xl space-y-4 rounded-2xl border border-border bg-card p-4 shadow-2xl sm:p-5">
              <div className="flex items-center justify-between">
                <span className="text-sm font-bold text-foreground">Label QR CCTV</span>
                <button onClick={() => setIsQrModalOpen(false)} className="p-1 rounded-lg border hover:bg-background text-gray-400 hover:text-foreground cursor-pointer">
                  <X className="h-4 w-4" />
                </button>
              </div>

              <div className="mx-auto grid aspect-[100/38] w-full max-w-[600px] grid-cols-[1.1fr_1.2fr_1.4fr] items-center overflow-hidden rounded-xl border-2 border-[#d7aa63] bg-white p-2 text-[#203f25] shadow-sm sm:p-4">
                <div className="flex h-full items-center justify-center overflow-hidden border-r-2 border-primary pr-1.5 sm:pr-3">
                  <img
                    src="/nusantara-logo.png"
                    alt="Logo Nusantara"
                    className="h-full max-h-32 w-full object-contain"
                  />
                </div>
                <img
                  src={`https://api.qrserver.com/v1/create-qr-code/?size=250x250&data=${encodeURIComponent(currentCCTV.qr_payload)}`}
                  alt={`QR Code CCTV - ${currentCCTV.kode_cctv}`}
                  className="mx-auto h-20 w-20 object-contain sm:h-32 sm:w-32"
                />
                <div className="min-w-0 pl-1.5 sm:pl-3">
                  <p className="line-clamp-3 text-[10px] font-bold leading-tight sm:text-sm">{currentCCTV.nama}</p>
                  <p className="mt-1 text-[9px] font-bold text-primary sm:mt-2 sm:text-xs">{currentCCTV.kode_cctv}</p>
                  <p className="mt-0.5 line-clamp-2 text-[8px] text-[#62705a] sm:mt-1 sm:text-[11px]">{currentCCTV.lokasi_area || '-'}</p>
                </div>
              </div>

              <p className="text-center text-[11px] text-muted">Ukuran cetak 38 × 100 mm (horizontal)</p>

              <div className="flex justify-end gap-2">
                <button
                  onClick={() => setIsQrModalOpen(false)}
                  className="rounded-lg border bg-card px-4 py-2 text-sm font-semibold text-foreground hover:bg-background"
                >
                  Tutup
                </button>
                <button
                  onClick={handlePrintQR}
                  className="flex items-center justify-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white hover:bg-primary-hover"
                >
                  <Printer className="h-4 w-4" />
                  <span>Cetak QR</span>
                </button>
              </div>
            </div>
          </div>
        )}

        {/* CCTV Detail Info Modal */}
        {isDetailModalOpen && currentCCTV && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fadeIn">
            <div className="relative w-full max-w-xl bg-card border border-border rounded-2xl p-6 shadow-2xl space-y-4 max-h-[90vh] overflow-y-auto">
              <div className="flex items-center justify-between border-b border-border pb-4">
                <h3 className="text-xl font-bold">Spesifikasi CCTV</h3>
                <button onClick={() => setIsDetailModalOpen(false)} className="p-1 rounded-lg border hover:bg-background text-gray-400 hover:text-foreground cursor-pointer">
                  <X className="h-5 w-5" />
                </button>
              </div>

              <div className="space-y-4">
                <div className="flex justify-between items-start border-b border-border pb-3">
                  <div>
                    <h4 className="font-bold text-lg text-foreground">{currentCCTV.nama}</h4>
                    <div className="flex items-center gap-2 mt-0.5">
                      <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-blue-500/10 text-blue-500 uppercase tracking-wider font-mono">{currentCCTV.kode_cctv}</span>
                      {currentCCTV.id_channel && <span className="text-xs text-gray-400">Channel ID: {currentCCTV.id_channel}</span>}
                    </div>
                  </div>
                  <span className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-semibold capitalize ${
                    currentCCTV.status === 'aktif' 
                      ? 'bg-emerald-500/10 text-emerald-500 border border-emerald-500/20' 
                      : 'bg-red-500/10 text-red-500 border border-red-500/20'
                  }`}>
                    {currentCCTV.status}
                  </span>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                  <div className="p-3 rounded-lg bg-background border border-border">
                    <span className="text-xs text-gray-400 block font-medium mb-1">Lokasi Area</span>
                    <span className="font-semibold text-foreground flex items-center gap-1.5">
                      <MapPin className="h-4 w-4 text-red-500" />
                      {currentCCTV.lokasi_area || '-'}
                    </span>
                  </div>

                  <div className="p-3 rounded-lg bg-background border border-border">
                    <span className="text-xs text-gray-400 block font-medium mb-1">Koordinat GPS</span>
                    <span className="font-semibold text-foreground font-mono text-xs">
                      Lat: {currentCCTV.latitude || '-'}, Lon: {currentCCTV.longitude || '-'}
                    </span>
                  </div>

                  <div className="p-3 rounded-lg bg-background border border-border">
                    <span className="text-xs text-gray-400 block font-medium mb-1">IP Camera</span>
                    <span className="font-semibold text-foreground font-mono">{currentCCTV.ip_camera || '-'}</span>
                  </div>

                  <div className="p-3 rounded-lg bg-background border border-border">
                    <span className="text-xs text-gray-400 block font-medium mb-1">IP NVR</span>
                    <span className="font-semibold text-foreground font-mono">{currentCCTV.ip_nvr || '-'}</span>
                  </div>

                  <div className="p-3 rounded-lg bg-background border border-border">
                    <span className="text-xs text-gray-400 block font-medium mb-1">Jenis CCTV</span>
                    <span className="font-semibold text-foreground capitalize">{currentCCTV.jenis_cctv || '-'}</span>
                  </div>

                  <div className="p-3 rounded-lg bg-background border border-border">
                    <span className="text-xs text-gray-400 block font-medium mb-1">Sumber Listrik</span>
                    <span className="font-semibold text-foreground">{currentCCTV.sumber_listrik || '-'}</span>
                  </div>

                  <div className="p-3 rounded-lg bg-background border border-border">
                    <span className="text-xs text-gray-400 block font-medium mb-1">Tipe Koneksi</span>
                    <span className="font-semibold text-foreground">{currentCCTV.tipe_koneksi || '-'}</span>
                  </div>

                  <div className="p-3 rounded-lg bg-background border border-border">
                    <span className="text-xs text-gray-400 block font-medium mb-1">Indoor / Outdoor</span>
                    <span className="font-semibold text-foreground capitalize">{currentCCTV.indoor_outdoor || '-'}</span>
                  </div>

                  <div className="p-3 rounded-lg bg-background border border-border">
                    <span className="text-xs text-gray-400 block font-medium mb-1">Pemilik / Pengelola</span>
                    <span className="font-semibold text-foreground">{currentCCTV.pemilik || '-'}</span>
                  </div>

                  <div className="p-3 rounded-lg bg-background border border-border">
                    <span className="text-xs text-gray-400 block font-medium mb-1">Tanggal Ditambahkan</span>
                    <span className="font-semibold text-foreground">
                      {new Date(currentCCTV.created_at).toLocaleDateString('id-ID', { year: 'numeric', month: 'long', day: 'numeric' })}
                    </span>
                  </div>

                  <div className="p-3 rounded-lg bg-background border border-border md:col-span-2">
                    <span className="text-xs text-gray-400 block font-medium mb-1">Keterangan Alat</span>
                    <p className="text-foreground text-xs leading-relaxed whitespace-pre-wrap">
                      {currentCCTV.keterangan || 'Tidak ada keterangan tambahan.'}
                    </p>
                  </div>
                </div>
              </div>

              <div className="flex gap-2 pt-4 border-t border-border justify-end">
                <button
                  onClick={() => setIsDetailModalOpen(false)}
                  className="px-5 py-2 rounded-lg bg-primary hover:bg-primary/95 text-white font-semibold text-sm cursor-pointer shadow-md shadow-primary/15"
                >
                  Selesai
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardLayout>
  );
}

'use client';

import React, { useState, useEffect, useRef, useCallback } from 'react';
import { createPortal } from 'react-dom';
import DashboardLayout from '@/components/DashboardLayout';
import Pagination from '@/components/Pagination';
import SortableHeader from '@/components/SortableHeader';
import AuthenticatedImage from '@/components/AuthenticatedImage';
import { apiRequest, downloadApiFile, API_BASE_URL, ApiValidationError, formatValidationMessages, getAllSitesForSelect } from '@/lib/api';
import { useAuth } from '@/context/AuthContext';
import { useWebSocket } from '@/context/WebSocketContext';
import { useToast } from '@/components/Toast';
import { useConfirm } from '@/components/ConfirmDialog';
import FieldMaintenanceModal from '@/components/FieldMaintenanceModal';
import { 
  Plus, 
  Search, 
  Edit2, 
  Trash2, 
  Camera, 
  MapPin, 
  Upload, 
  Scan, 
  QrCode,
  MoreHorizontal,
  X, 
  AlertTriangle, 
  Eye, 
  EyeOff,
  Printer,
  LoaderCircle,
  ClipboardCheck,
  SlidersHorizontal,
  RotateCcw,
  History,
  Network,
  ShieldCheck,
  TrendingUp
} from 'lucide-react';

interface CCTV {
  id: string;
  kode_cctv: string;
  id_channel: string;
  nama: string;
  lokasi_area: string;
  longitude?: number | null;
  latitude?: number | null;
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
  health_score: number;
  created_at: string;
  channel_port: number;
  stream_enabled: boolean;
  rtsp_url?: string;
  snapshot_url?: string;
  // Zona operasional & kawasan (site) — opsional (Task 11.4 & 12.2).
  zone_id?: string | null;
  site_id?: string | null;
}

interface QRPrintItem {
  cctv: CCTV;
  qrDataUrl: string;
}

// Satu baris kejadian pada timeline CCTV — sesuai struktur respons backend:
// { id, cctv_id, event_type, image_path, notes, created_at }
interface TimelineEvent {
  id: string;
  cctv_id: string;
  event_type: string;
  image_path?: string;
  notes?: string;
  created_at: string;
}

// Breakdown skor kesehatan dari GET /cctv/:id/health-score.
interface HealthBreakdown {
  health_score: number;
  updated_at: string;
  breakdown: {
    failed: number;
    attention: number;
    overdue: number;
    age: number;
    not_ok_items: number;
  };
}

// Titik histori skor kesehatan dari GET /cctv/:id/health-history.
interface HealthHistoryPoint {
  health_score: number;
  calculated_at: string;
}

// Label Bahasa Indonesia untuk tiap tipe kejadian pada timeline
const EVENT_TYPE_LABELS: Record<string, string> = {
  status_changed: 'Status berubah',
  maintenance_completed: 'Pemeliharaan selesai',
  field_check_attention: 'Pengecekan butuh perhatian',
  periodic_snapshot: 'Snapshot berkala',
};

const eventTypeLabel = (type: string) =>
  EVENT_TYPE_LABELS[type] || type.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());

// Konversi nilai koordinat dari input teks menjadi number | null. Kosong → null;
// nilai yang tidak valid (bukan angka) → null; angka valid → number.
const toCoordinateNumber = (value: string): number | null => {
  const trimmed = value.trim();
  if (trimmed === '') return null;
  const num = Number(trimmed);
  return Number.isFinite(num) ? num : null;
};

export default function CCTVPage() {
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';
  const toast = useToast();
  const confirm = useConfirm();

  const [cctvs, setCctvs] = useState<CCTV[]>([]);
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

  // Sort State — default urut A-Z berdasarkan nama (perubahan Sprint 40)
  const [sortBy, setSortBy] = useState('nama');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc');

  // Modal Sortir & Filter
  const [isSortModalOpen, setIsSortModalOpen] = useState(false);
  const [draftSortBy, setDraftSortBy] = useState('nama');
  const [draftSortOrder, setDraftSortOrder] = useState<'asc' | 'desc'>('asc');
  const [draftStatusFilter, setDraftStatusFilter] = useState('');
  const [draftIoFilter, setDraftIoFilter] = useState('');

  // Jumlah filter/sortir aktif — dipakai untuk badge pada tombol Sortir
  const activeFilterCount =
    (statusFilter ? 1 : 0) +
    (ioFilter ? 1 : 0) +
    (sortBy !== 'nama' || sortOrder !== 'asc' ? 1 : 0);

  // Modals state
  const [isFormModalOpen, setIsFormModalOpen] = useState(false);
  const [isScanModalOpen, setIsScanModalOpen] = useState(false);
  const [isImportModalOpen, setIsImportModalOpen] = useState(false);
  const [isQrModalOpen, setIsQrModalOpen] = useState(false);
  const [isDetailModalOpen, setIsDetailModalOpen] = useState(false);
  const [isFieldMaintenanceOpen, setIsFieldMaintenanceOpen] = useState(false);
  
  const [currentCCTV, setCurrentCCTV] = useState<CCTV | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  // Id CCTV yang menu aksinya (detail/edit/hapus) sedang terbuka.
  const [openActionMenu, setOpenActionMenu] = useState<string | null>(null);

  // Timeline kejadian CCTV (dimuat saat modal detail dibuka)
  const [timeline, setTimeline] = useState<TimelineEvent[]>([]);
  const [timelineLoading, setTimelineLoading] = useState(false);
  const [timelineError, setTimelineError] = useState('');

  // Breakdown skor kesehatan CCTV (alasan penurunan skor)
  const [healthBreakdown, setHealthBreakdown] = useState<HealthBreakdown | null>(null);
  const [healthLoading, setHealthLoading] = useState(false);
  const [healthError, setHealthError] = useState('');

  // Histori skor kesehatan (tren) untuk detail CCTV
  const [healthHistory, setHealthHistory] = useState<HealthHistoryPoint[]>([]);
  const [healthHistoryRange, setHealthHistoryRange] = useState<'7d' | '30d' | '90d'>('30d');
  const [healthHistoryLoading, setHealthHistoryLoading] = useState(false);
  const [healthHistoryError, setHealthHistoryError] = useState('');

  // Error validasi per field dari backend (ditampilkan di dalam modal form)
  const [formErrors, setFormErrors] = useState<string[]>([]);

  // State untuk toggle visibilitas password RTSP pada form streaming
  const [showRtspPassword, setShowRtspPassword] = useState(false);

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
    keterangan: '',
    channel_port: 0,
    rtsp_password: '',
    stream_enabled: false,
    rtsp_url: '',
    zone_id: '',
    site_id: ''
  });

  // Daftar zona operasional & kawasan (site) untuk dropdown form CCTV.
  const [zones, setZones] = useState<Array<{ id: string; nama: string }>>([]);
  const [sites, setSites] = useState<Array<{ id: string; nama: string }>>([]);

  // Scanner State
  const scannerRef = React.useRef<any>(null);
  const [cameraActive, setCameraActive] = useState(false);
  const [scannedResult, setScannedResult] = useState<CCTV | null>(null);
  const [scanError, setScanError] = useState('');
  const [scanLoading, setScanLoading] = useState(false);
  
  // Bulk Actions State
  const [selectedIDs, setSelectedIDs] = useState<string[]>([]);
  const [isSelectingAll, setIsSelectingAll] = useState(false);

  // File Import State
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [importLoading, setImportLoading] = useState(false);
  const [importError, setImportError] = useState('');
  const [importSuccess, setImportSuccess] = useState('');

  // QR Code State — gambar QR di-generate server-side (tidak bergantung API pihak ketiga)
  const qrCacheRef = useRef<Record<string, string>>({});

  const [printQueue, setPrintQueue] = useState<QRPrintItem[] | null>(null);
  const [printMode, setPrintMode] = useState<'single' | 'batch'>('single');

  const printLogoUrl = typeof window !== 'undefined' ? `${window.location.origin}/nusantara-logo.png` : '/nusantara-logo.png';

  // Ambil gambar QR PNG (dengan autentikasi cookie httpOnly) lalu konversi ke data URL untuk <img>
  const fetchQRDataUrl = useCallback(async (cctv: CCTV): Promise<string> => {
    // Payload ikut menjadi cache version. Rotasi mempertahankan ID aset tetapi
    // mengganti payload, jadi label baru tidak boleh memakai data URL lama.
    const cacheKey = `${cctv.id}:${cctv.qr_payload}`;
    if (qrCacheRef.current[cacheKey]) {
      return qrCacheRef.current[cacheKey];
    }
    const response = await fetch(`${API_BASE_URL}/cctv/${cctv.id}/qrcode`, {
      credentials: 'include',
      cache: 'no-store',
    });
    if (!response.ok) {
      throw new Error('Gagal mengambil QR Code dari server');
    }
    const blob = await response.blob();
    const dataUrl = await new Promise<string>((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(reader.result as string);
      reader.onerror = () => reject(new Error('Gagal membaca QR Code'));
      reader.readAsDataURL(blob);
    });
    qrCacheRef.current[cacheKey] = dataUrl;
    return dataUrl;
  }, []);

  // Siapkan antrian cetak label QR (React-rendered agar semua teks ter-escape otomatis)
  const openQRPrint = useCallback(
    async (items: CCTV[], mode: 'single' | 'batch') => {
      if (items.length === 0) {
        toast.error('Tidak ada data QR Code CCTV yang dapat dicetak.');
        return;
      }
      try {
        const qrItems: QRPrintItem[] = [];
        for (const cctv of items) {
          const qrDataUrl = await fetchQRDataUrl(cctv);
          qrItems.push({ cctv, qrDataUrl });
        }
        setPrintMode(mode);
        setPrintQueue(qrItems);
        toast.success(`${qrItems.length} QR Code CCTV berhasil disiapkan untuk dicetak.`);
      } catch (err: any) {
        toast.error(err?.message || 'Gagal menyiapkan QR Code untuk dicetak. Coba lagi.');
      }
    },
    [fetchQRDataUrl, toast]
  );

  // Memicu dialog print ketika antrian cetak tersedia, lalu bersihkan setelahnya
  useEffect(() => {
    if (!printQueue || printQueue.length === 0) return;

    // Ukuran halaman sesuai mode cetak: label tunggal 100x38mm, batch A4
    const styleId = 'sigap-print-page-size';
    let styleEl = document.getElementById(styleId) as HTMLStyleElement | null;
    if (!styleEl) {
      styleEl = document.createElement('style');
      styleEl.id = styleId;
      document.head.appendChild(styleEl);
    }
    styleEl.textContent =
      printMode === 'single'
        ? '@page { size: 100mm 38mm; margin: 0; }'
        : '@page { size: A4; margin: 10mm; }';

    document.body.classList.add('sigap-printing');

    const timer = setTimeout(() => {
      window.print();
    }, 200);

    const handleAfterPrint = () => {
      setPrintQueue(null);
    };
    window.addEventListener('afterprint', handleAfterPrint);

    return () => {
      clearTimeout(timer);
      document.body.classList.remove('sigap-printing');
      styleEl?.remove();
      window.removeEventListener('afterprint', handleAfterPrint);
    };
  }, [printQueue, printMode]);

  const fetchCCTVs = async () => {
    setLoading(true);
    setError('');
    try {
      const queryParams = [`page=${page}`, 'limit=10'];
      if (search) queryParams.push(`search=${encodeURIComponent(search)}`);
      if (statusFilter) queryParams.push(`status=${encodeURIComponent(statusFilter)}`);
      if (ioFilter) queryParams.push(`indoor_outdoor=${encodeURIComponent(ioFilter)}`);
      if (sortBy) queryParams.push(`sort_by=${encodeURIComponent(sortBy)}`);
      if (sortOrder) queryParams.push(`order=${encodeURIComponent(sortOrder)}`);
      
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
      setError(err.message || 'Data CCTV belum dapat dimuat. Coba lagi.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchCCTVs();
  }, [page, search, statusFilter, ioFilter, sortBy, sortOrder]);

  // Muat daftar zona & kawasan (site) untuk dropdown form CCTV (Task 11.4/12.2).
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [zoneRes, siteRes] = await Promise.all([
          apiRequest<Array<{ id: string; nama: string }>>('/zones/all'),
          getAllSitesForSelect(),
        ]);
        if (!cancelled) {
          if (zoneRes.status === 'success' && zoneRes.data) setZones(zoneRes.data);
          if (siteRes.status === 'success' && siteRes.data) {
            setSites(siteRes.data.map((s) => ({ id: s.id, nama: s.nama })));
          }
        }
      } catch {
        // Gagal memuat dropdown tidak fatal — form tetap bisa diisi tanpa zona/kawasan.
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const { subscribe } = useWebSocket();

  useEffect(() => {
    const unsub = subscribe('CCTV_UPDATED', () => {
      fetchCCTVs();
    });
    return unsub;
  }, [subscribe]);

  const normalizedStatus = (status: string) => status.trim().toUpperCase();
  const isStatus = (status: string, values: string[]) => values.includes(normalizedStatus(status));

  // Normalisasi nilai status dari database (huruf kecil Indonesia / legacy huruf
  // besar Inggris) ke nilai opsi <select> — enum backend: aktif/pemeliharaan/nonaktif.
  const toStatusValue = (status: string) => {
    const norm = status.trim().toUpperCase();
    if (norm === 'ACTIVE') return 'aktif';
    if (norm === 'INACTIVE' || norm === 'RUSAK' || norm === 'BROKEN') return 'nonaktif';
    if (norm === 'MAINTENANCE') return 'pemeliharaan';
    return status.toLowerCase();
  };
  const statusLabel = (status: string) => {
    const normalized = normalizedStatus(status);
    if (['ACTIVE', 'AKTIF'].includes(normalized)) return 'Aktif';
    if (['INACTIVE', 'NONAKTIF'].includes(normalized)) return 'Nonaktif';
    if (['MAINTENANCE', 'PEMELIHARAAN'].includes(normalized)) return 'Pemeliharaan';
    if (['BROKEN', 'RUSAK'].includes(normalized)) return 'Nonaktif';
    return status;
  };

  // Skor lama yang belum dikirim API dianggap sehat sesuai nilai default model.
  const normalizedHealthScore = (score?: number) => {
    if (typeof score !== 'number' || !Number.isFinite(score)) return 100;
    return Math.max(0, Math.min(100, Math.round(score)));
  };
  const healthScoreClasses = (score?: number) => {
    const value = normalizedHealthScore(score);
    if (value >= 70) return 'border-emerald-500/25 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400';
    if (value >= 40) return 'border-amber-500/25 bg-amber-500/10 text-amber-600 dark:text-amber-400';
    return 'border-red-500/25 bg-red-500/10 text-red-600 dark:text-red-400';
  };
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
      toast.error(err instanceof Error ? err.message : 'Detail CCTV belum dapat dimuat. Coba lagi.');
    } finally {
      setActionLoading(null);
    }
  };

  // Ambil riwayat kejadian (timeline) CCTV terbaru saat modal detail dibuka.
  const fetchTimeline = useCallback(async (id: string) => {
    setTimelineLoading(true);
    setTimelineError('');
    try {
      const response = await apiRequest<{ rows: TimelineEvent[] }>(`/cctv/${id}/timeline?page=1&limit=20`);
      if (response.status === 'success' && response.data) {
        setTimeline(response.data.rows || []);
      } else {
        setTimeline([]);
      }
    } catch (err: any) {
      setTimelineError(err.message || 'Riwayat CCTV belum dapat dimuat.');
      setTimeline([]);
    } finally {
      setTimelineLoading(false);
    }
  }, []);

  // Ambil breakdown skor kesehatan (alasan penurunan skor) saat modal detail dibuka.
  const fetchHealthBreakdown = useCallback(async (id: string) => {
    setHealthLoading(true);
    setHealthError('');
    try {
      const response = await apiRequest<HealthBreakdown>(`/cctv/${id}/health-score`);
      if (response.status === 'success' && response.data) {
        setHealthBreakdown(response.data);
      } else {
        setHealthBreakdown(null);
      }
    } catch (err: any) {
      setHealthError(err.message || 'Skor kesehatan belum dapat dimuat.');
      setHealthBreakdown(null);
    } finally {
      setHealthLoading(false);
    }
  }, []);

  // Ambil histori skor kesehatan (tren) untuk detail CCTV.
  const fetchHealthHistory = useCallback(async (id: string, range: string) => {
    setHealthHistoryLoading(true);
    setHealthHistoryError('');
    try {
      const response = await apiRequest<HealthHistoryPoint[]>(`/cctv/${id}/health-history?range=${range}`);
      if (response.status === 'success' && Array.isArray(response.data)) {
        setHealthHistory(response.data);
      } else {
        setHealthHistory([]);
      }
    } catch (err: any) {
      setHealthHistoryError(err.message || 'Histori skor kesehatan belum dapat dimuat.');
      setHealthHistory([]);
    } finally {
      setHealthHistoryLoading(false);
    }
  }, []);

  // Muat timeline saat modal detail dibuka; reset saat modal ditutup atau CCTV berubah.
  useEffect(() => {
    if (isDetailModalOpen && currentCCTV) {
      fetchTimeline(currentCCTV.id);
      fetchHealthBreakdown(currentCCTV.id);
      fetchHealthHistory(currentCCTV.id, healthHistoryRange);
    } else {
      setTimeline([]);
      setTimelineLoading(false);
      setTimelineError('');
      setHealthBreakdown(null);
      setHealthLoading(false);
      setHealthError('');
      setHealthHistory([]);
      setHealthHistoryLoading(false);
      setHealthHistoryError('');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isDetailModalOpen, currentCCTV, fetchTimeline, fetchHealthBreakdown]);

  // Muat ulang histori saat range diubah.
  useEffect(() => {
    if (isDetailModalOpen && currentCCTV) {
      fetchHealthHistory(currentCCTV.id, healthHistoryRange);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [healthHistoryRange]);

  // Format waktu kejadian ke locale Indonesia (tanggal & jam)
  const formatTimelineDate = (value: string) => {
    const date = new Date(value);
    if (isNaN(date.getTime())) return value;
    return date.toLocaleString('id-ID', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const handleOpenForm = (cctv: CCTV | null = null) => {
    setFormErrors([]);
    if (cctv) {
      setCurrentCCTV(cctv);
      setFormData({
        kode_cctv: cctv.kode_cctv,
        id_channel: cctv.id_channel || '',
        nama: cctv.nama,
        lokasi_area: cctv.lokasi_area || '',
        longitude: cctv.longitude != null ? String(cctv.longitude) : '',
        latitude: cctv.latitude != null ? String(cctv.latitude) : '',
        jenis_cctv: cctv.jenis_cctv || '',
        sumber_listrik: cctv.sumber_listrik || '',
        tipe_koneksi: cctv.tipe_koneksi || '',
        indoor_outdoor: cctv.indoor_outdoor || 'indoor',
        pemilik: cctv.pemilik || '',
        status: toStatusValue(cctv.status),
        ip_nvr: cctv.ip_nvr || '',
        ip_camera: cctv.ip_camera || '',
        keterangan: cctv.keterangan || '',
        // rtsp_password tidak pernah dikembalikan API — selalu dikosongkan; hanya terisi bila user mengetik ulang
        channel_port: typeof cctv.channel_port === 'number' ? cctv.channel_port : 0,
        rtsp_password: '',
        stream_enabled: !!cctv.stream_enabled,
        rtsp_url: cctv.rtsp_url || '',
        zone_id: cctv.zone_id || '',
        site_id: cctv.site_id || ''
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
        status: 'aktif',
        ip_nvr: '',
        ip_camera: '',
        keterangan: '',
        channel_port: 0,
        rtsp_password: '',
        stream_enabled: false,
        rtsp_url: '',
        zone_id: '',
        site_id: ''
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

      // Konversi koordinat dari string (input teks) menjadi number | null sesuai
      // tipe data backend (double precision).
      const payload = {
        ...formData,
        longitude: toCoordinateNumber(formData.longitude),
        latitude: toCoordinateNumber(formData.latitude),
      };

      const response = await apiRequest<CCTV>(endpoint, {
        method,
        body: JSON.stringify(payload),
      });

      if (response.status === 'success') {
        setIsFormModalOpen(false);
        setFormErrors([]);
        toast.success(currentCCTV ? `Data CCTV "${formData.nama}" berhasil diperbarui.` : `CCTV "${formData.nama}" berhasil ditambahkan.`);
        fetchCCTVs();
      }
    } catch (err: any) {
      if (err instanceof ApiValidationError) {
        // Tampilkan rincian isian yang bermasalah langsung di dalam form
        setFormErrors(
          err.errors.length > 0
            ? formatValidationMessages(err.errors)
            : [err.message]
        );
      } else {
        toast.error(err.message || 'Data CCTV belum dapat disimpan. Periksa isian lalu coba lagi.');
      }
    } finally {
      setActionLoading(null);
    }
  };

  const handleDeleteCCTV = async (cctv: CCTV) => {
    const ok = await confirm({
      title: 'Hapus CCTV?',
      message: `Hapus CCTV “${cctv.nama}”? CCTV tidak akan ditampilkan dalam daftar aktif.`,
      confirmLabel: 'Hapus',
    });
    if (!ok) return;
    setActionLoading(`delete-${cctv.id}`);
    try {
      const response = await apiRequest(`/cctv/${cctv.id}`, {
        method: 'DELETE'
      });
      if (response.status === 'success') {
        setSelectedIDs(prev => prev.filter(item => item !== cctv.id));
        toast.success(`CCTV "${cctv.nama}" berhasil dihapus.`);
        fetchCCTVs();
      }
    } catch (err: any) {
      toast.error(err.message || 'CCTV belum dapat dihapus. Coba lagi.');
    } finally {
      setActionLoading(null);
    }
  };

  // Rotasi QR Code CCTV (Task 13.14): membuat QrPayload baru sehingga QR lama
  // tidak berlaku. Setelah berhasil, langsung siapkan cetak label QR baru.
  const handleRotateQR = async (cctv: CCTV) => {
    const ok = await confirm({
      title: 'Rotasi QR Code?',
      message: `QR Code lama untuk “${cctv.nama}” akan tidak berlaku. Cetak label QR baru setelah rotasi. Lanjutkan?`,
      confirmLabel: 'Rotasi QR',
    });
    if (!ok) return;
    setActionLoading(`rotate-qr-${cctv.id}`);
    try {
      const response = await apiRequest<CCTV>(`/cctv/${cctv.id}/rotate-qr`, {
        method: 'POST'
      });
      if (response.status === 'success' && response.data) {
        toast.success(`QR Code untuk "${cctv.nama}" berhasil dirotasi.`);
        fetchCCTVs();
        // Langsung siapkan cetak label QR baru untuk CCTV ini.
        openQRPrint([response.data], 'single');
      }
    } catch (err: any) {
      toast.error(err.message || 'QR Code belum dapat dirotasi. Coba lagi.');
    } finally {
      setActionLoading(null);
    }
  };

  // Bulk Selection Handlers
  const handleSelectAll = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const shouldSelectAll = e.target.checked;

    if (!shouldSelectAll) {
      setSelectedIDs([]);
      return;
    }

    setIsSelectingAll(true);
    try {
      const queryParams = [
        'page=1',
        `limit=${Math.max(pagination.total_rows, 1)}`,
      ];
      if (search) queryParams.push(`search=${encodeURIComponent(search)}`);
      if (statusFilter) queryParams.push(`status=${encodeURIComponent(statusFilter)}`);
      if (ioFilter) queryParams.push(`indoor_outdoor=${encodeURIComponent(ioFilter)}`);

      const response = await apiRequest<{ rows: CCTV[] }>(`/cctv?${queryParams.join('&')}`);
      setSelectedIDs(response.data?.rows.map((cctv) => cctv.id) ?? []);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Seluruh CCTV belum dapat dipilih. Coba lagi.');
      setSelectedIDs([]);
    } finally {
      setIsSelectingAll(false);
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
      ? 'Hapus seluruh data CCTV? Semua CCTV akan dihapus dari SIGAP dan tindakan ini tidak dapat dibatalkan.'
      : `Hapus ${selectedIDs.length} CCTV terpilih dari SIGAP?`;

    const ok = await confirm({
      title: deleteAll ? 'Hapus Semua CCTV?' : 'Hapus CCTV Terpilih?',
      message: confirmMessage,
      confirmLabel: deleteAll ? 'Hapus Semua' : 'Hapus Terpilih',
    });
    if (!ok) return;

    setActionLoading(deleteAll ? 'delete-all' : 'delete-selected');
    try {
      const response = await apiRequest<any>('/cctv/bulk-delete', {
        method: 'POST',
        body: JSON.stringify({
          ids: deleteAll ? [] : selectedIDs,
          all: deleteAll
        })
      });

      setSelectedIDs([]);
      fetchCCTVs();
      toast.success(response.message || (deleteAll ? 'Seluruh data CCTV berhasil dihapus.' : `${selectedIDs.length} CCTV terpilih berhasil dihapus.`));
    } catch (err: any) {
      toast.error(err.message || 'Data CCTV belum dapat dihapus. Coba lagi.');
    } finally {
      setActionLoading(null);
    }
  };

  // Download Template CCTV
  const handleDownloadTemplate = async () => {
    try {
      await downloadApiFile('/cctv/template', 'template_import_cctv.xlsx');
      toast.success('Template impor Excel berhasil diunduh.');
    } catch (err: any) {
      toast.error(err.message || 'Template impor belum dapat diunduh. Coba lagi.');
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
      const dataPayload = new FormData();
      dataPayload.append('file', selectedFile);

      const response = await apiRequest<any>('/cctv/import', {
        method: 'POST',
        body: dataPayload
      });

      setImportSuccess(response.message || 'Data CCTV berhasil diimpor.');
      setSelectedFile(null);
      toast.success(response.message || 'Data CCTV berhasil diimpor.');
      fetchCCTVs();
    } catch (err: any) {
      setImportError(err.message || 'Berkas Excel belum dapat diimpor. Periksa format lalu coba lagi.');
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
      setScanError(err.message || 'Payload QR tidak valid atau tidak dapat didekripsi. Pindai ulang QR.');
    } finally {
      setScanLoading(false);
    }
  };

  // Start webcam scanner
  const startCameraScanner = () => {
    setScanError('');
    setScannedResult(null);
    
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
        setScanError('Pemindai kamera belum dapat dimuat. Gunakan input manual. ' + err.message);
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
        setScanError("Kamera tidak dapat diakses. Berikan izin kamera atau gunakan input manual.");
        setCameraActive(false);
      });
    } catch (e: any) {
      console.error(e);
      setScanError("Pemindai belum dapat dijalankan. Gunakan input manual. " + e.message);
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
    if (isScanModalOpen && !scannedResult) {
      startCameraScanner();
    } else {
      stopCameraScanner();
    }

    return () => {
      if (scannerRef.current && scannerRef.current.isScanning) {
        scannerRef.current.stop().catch((err: any) => console.error("Cleanup error:", err));
      }
    };
  }, [isScanModalOpen, scannedResult]);

  const handleCloseScanModal = async () => {
    await stopCameraScanner();
    setIsScanModalOpen(false);
    setScannedResult(null);
    setScanError('');
  };

  // Print QR Code (single label)
  const handlePrintQR = () => {
    if (!currentCCTV) return;
    openQRPrint([currentCCTV], 'single');
  };

  // Batch Print QR Code (Multiple / All)
  const handleBatchPrintQR = async (targetIDs?: string[]) => {
    let itemsToPrint: CCTV[] = [];

    if (targetIDs && targetIDs.length > 0) {
      itemsToPrint = cctvs.filter((cctv) => targetIDs.includes(cctv.id));

      if (itemsToPrint.length < targetIDs.length) {
        try {
          const response = await apiRequest<{ rows: CCTV[] }>(`/cctv?page=1&limit=${Math.max(pagination.total_rows, 1000)}`);
          if (response.status === 'success' && response.data) {
            itemsToPrint = response.data.rows.filter((cctv) => targetIDs.includes(cctv.id));
          }
        } catch (err) {
          console.error("Gagal mengambil data lengkap untuk cetak batch:", err);
        }
      }
    } else {
      try {
        const response = await apiRequest<{ rows: CCTV[] }>(`/cctv?page=1&limit=100`);
        if (response.status === 'success' && response.data) {
          itemsToPrint = response.data.rows;
        } else {
          itemsToPrint = cctvs;
        }
      } catch (err) {
        console.error("Gagal mengambil data lengkap untuk cetak batch:", err);
        itemsToPrint = cctvs;
      }
    }

    await openQRPrint(itemsToPrint, 'batch');
  };

  const openSortModal = () => {
    setDraftSortBy(sortBy);
    setDraftSortOrder(sortOrder);
    setDraftStatusFilter(statusFilter);
    setDraftIoFilter(ioFilter);
    setIsSortModalOpen(true);
  };

  const applySortFilter = () => {
    let changed = false;
    if (draftSortBy !== sortBy) { setSortBy(draftSortBy); changed = true; }
    if (draftSortOrder !== sortOrder) { setSortOrder(draftSortOrder); changed = true; }
    if (draftStatusFilter !== statusFilter) { setStatusFilter(draftStatusFilter); changed = true; }
    if (draftIoFilter !== ioFilter) { setIoFilter(draftIoFilter); changed = true; }
    if (changed) setPage(1);
    setIsSortModalOpen(false);
  };

  const resetSortFilter = () => {
    setSortBy('created_at');
    setSortOrder('desc');
    setStatusFilter('');
    setIoFilter('');
    setPage(1);
    setIsSortModalOpen(false);
  };

  // Sorting lewat klik pada judul kolom (server-side, otomatis memicu re-fetch).
  const handleHeaderSort = (key: string) => {
    if (sortBy === key) {
      // Klik lagi: balik arah urutan
      setSortOrder((current) => (current === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortBy(key);
      // Kolom teks default naik; kolom angka default turun (terbesar dulu)
      setSortOrder(key === 'health_score' ? 'desc' : 'asc');
    }
    setPage(1);
  };

  useEffect(() => {
    if (!isSortModalOpen) return;

    const previousOverflow = document.body.style.overflow;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setIsSortModalOpen(false);
    };

    document.body.style.overflow = 'hidden';
    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [isSortModalOpen]);

  // Tutup menu aksi baris saat klik di luar / tekan Escape.
  useEffect(() => {
    if (!openActionMenu) return;
    const close = () => setOpenActionMenu(null);
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close();
    };
    document.addEventListener('click', close);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('click', close);
      document.removeEventListener('keydown', onKey);
    };
  }, [openActionMenu]);

  return (
    <DashboardLayout>
      <div className={`space-y-8 ${selectedIDs.length > 0 && isAdmin ? 'pb-24' : ''}`}>
        <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-[-0.025em]">Aset CCTV</h1>
            <p className="mt-1 text-sm text-muted">Cari, identifikasi, dan kelola perangkat CCTV.</p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={() => setIsScanModalOpen(true)}
              className="inline-flex items-center justify-center gap-2 rounded-xl border border-border bg-card px-4 py-2.5 text-sm font-semibold transition-colors hover:border-primary"
            >
              <Scan className="h-4 w-4 text-primary" />
              <span>Pindai QR</span>
            </button>
            {isAdmin && (
              <details className="group relative">
                <summary
                  aria-label="Buka tindakan lainnya"
                  className="flex h-10 w-10 cursor-pointer list-none items-center justify-center rounded-xl border border-border bg-card text-muted transition-colors hover:border-primary hover:text-foreground [&::-webkit-details-marker]:hidden"
                >
                  <MoreHorizontal className="h-5 w-5" />
                </summary>
                <div className="absolute right-0 z-20 mt-2 w-52 overflow-hidden rounded-xl border border-border bg-card shadow-[0_12px_28px_rgba(32,63,37,0.14)]">
                  <button
                    type="button"
                    onClick={() => setIsImportModalOpen(true)}
                    className="flex w-full items-center gap-3 px-4 py-3 text-left text-sm font-semibold hover:bg-background"
                  >
                    <Upload className="h-4 w-4 text-primary" />
                    Impor Excel
                  </button>
                  {cctvs.length > 0 && (
                    <button
                      type="button"
                      onClick={() => handleBatchPrintQR()}
                      className="flex w-full items-center gap-3 border-t border-border px-4 py-3 text-left text-sm font-semibold hover:bg-background"
                    >
                      <Printer className="h-4 w-4 text-primary" />
                      Cetak Semua QR Code
                    </button>
                  )}
                  {cctvs.length > 0 && (
                    <button
                      type="button"
                      onClick={() => handleBulkDelete(true)}
                      disabled={actionLoading === 'delete-all'}
                      className="flex w-full items-center gap-3 border-t border-border px-4 py-3 text-left text-sm font-semibold text-error hover:bg-error/10 disabled:cursor-wait disabled:opacity-60"
                    >
                      {actionLoading === 'delete-all' ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                      {actionLoading === 'delete-all' ? 'Menghapus…' : 'Hapus semua CCTV'}
                    </button>
                  )}
                </div>
              </details>
            )}
            {isAdmin && (
              <button
                type="button"
                onClick={() => handleOpenForm()}
                className="inline-flex items-center justify-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white shadow-[0_6px_16px_rgba(70,114,53,0.2)] transition-colors hover:bg-primary-hover"
              >
                <Plus className="h-4 w-4" />
                <span>Tambah CCTV</span>
              </button>
            )}
          </div>
        </header>

        <section aria-labelledby="cctv-list-heading" className="overflow-visible rounded-2xl border border-border bg-card">
          <div className="flex flex-col gap-1 border-b border-border px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 id="cctv-list-heading" className="text-lg font-bold">Daftar CCTV</h2>
              <p className="mt-0.5 text-xs text-muted">{pagination.total_rows} perangkat</p>
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

          <div className="border-b border-border px-5 py-4">
            <div className="relative max-w-md">
              <Search className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" />
              <input
                type="search"
                aria-label="Cari CCTV"
                placeholder="Cari nama, kode, channel, atau lokasi"
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setSelectedIDs([]);
                  setPage(1);
                }}
                className="w-full rounded-xl border border-input-border bg-input py-2.5 pl-10 pr-4 text-sm text-foreground outline-none transition-colors focus:border-primary focus:ring-2 focus:ring-primary/20"
              />
            </div>
          </div>

          {/* Floating Bulk Actions Bar */}
          {selectedIDs.length > 0 && isAdmin && (
            <div className="fixed bottom-6 left-1/2 z-40 flex -translate-x-1/2 items-center gap-3 rounded-2xl border border-border bg-card/95 px-4 py-3 shadow-[0_16px_40px_rgba(0,0,0,0.25)] backdrop-blur-md">
              <span className="text-xs font-bold text-foreground">
                {selectedIDs.length} terpilih
              </span>
              <button
                type="button"
                onClick={() => setSelectedIDs([])}
                className="rounded-lg p-1 text-muted hover:bg-background hover:text-foreground"
                title="Batal seleksi"
              >
                <X className="h-4 w-4" />
              </button>
              <button
                type="button"
                onClick={() => handleBatchPrintQR(selectedIDs)}
                className="inline-flex min-h-10 shrink-0 items-center justify-center gap-2 rounded-xl border border-primary/30 bg-primary/10 px-4 py-2 text-xs font-bold text-primary transition-colors hover:bg-primary/20"
              >
                <Printer className="h-4 w-4" />
                Cetak {selectedIDs.length} QR
              </button>
              <button
                type="button"
                onClick={() => handleBulkDelete(false)}
                disabled={actionLoading === 'delete-selected'}
                className="inline-flex min-h-10 shrink-0 items-center justify-center gap-2 rounded-xl bg-error px-4 py-2 text-xs font-bold text-white transition-colors hover:bg-error/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-error focus-visible:ring-offset-2 focus-visible:ring-offset-card disabled:cursor-wait disabled:opacity-60"
              >
                {actionLoading === 'delete-selected' ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                {actionLoading === 'delete-selected' ? 'Menghapus…' : `Hapus ${selectedIDs.length} CCTV`}
              </button>
            </div>
          )}

          {loading ? (
          <div className="flex items-center justify-center gap-3 py-20" role="status">
            <div className="h-8 w-8 animate-spin rounded-full border-3 border-primary border-t-transparent"></div>
            <span className="text-sm text-muted">Memuat data CCTV…</span>
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
            <AlertTriangle className="h-11 w-11 text-error" />
            <h3 className="mt-3 text-base font-semibold">Data CCTV belum dapat dimuat</h3>
            <p className="mt-1 max-w-md text-xs text-muted">{error}</p>
            <button type="button" onClick={fetchCCTVs} className="mt-4 rounded-lg bg-primary px-4 py-2 text-xs font-bold text-white">
              Coba lagi
            </button>
          </div>
        ) : cctvs.length === 0 ? (
          <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
            <Camera className="mb-3 h-10 w-10 text-muted/45" />
            <h3 className="text-base font-semibold">{search || statusFilter || ioFilter ? 'Tidak ada CCTV yang cocok' : 'Belum ada data CCTV'}</h3>
            <p className="mt-1 max-w-sm text-xs text-muted">
              {search || statusFilter || ioFilter ? 'Ubah kata kunci atau filter untuk melihat CCTV lain.' : 'Gunakan tombol “Tambah CCTV” atau “Impor Excel” untuk menambahkan data.'}
            </p>
          </div>
        ) : (
          <>
            {/* Saat menu aksi terbuka, overflow-x-auto dihilangkan agar dropdown
                tidak terpotong oleh wrapper tabel (z-index saja tidak cukup —
                overflow menciptakan clipping context). */}
            <div className={openActionMenu ? 'overflow-visible' : 'overflow-x-auto'}>
              <table className="w-full min-w-[740px] border-collapse text-left">
                <thead>
                  <tr className="border-b border-border bg-background/50 text-xs font-semibold text-muted">
                    {isAdmin && (
                      <th className="w-12 px-5 py-3.5 text-center">
                        <div className="flex items-center justify-center gap-1.5" aria-busy={isSelectingAll}>
                          <input
                            type="checkbox"
                            onChange={handleSelectAll}
                            checked={pagination.total_rows > 0 && selectedIDs.length === pagination.total_rows}
                            disabled={isSelectingAll}
                            aria-label={`Pilih semua ${pagination.total_rows} CCTV pada seluruh halaman`}
                            className="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary focus:ring-primary disabled:cursor-wait"
                          />
                          {isSelectingAll && <LoaderCircle className="h-3.5 w-3.5 animate-spin text-primary" aria-hidden="true" />}
                        </div>
                      </th>
                    )}
                    <th
                      className="px-5 py-3.5"
                      aria-sort={sortBy === 'nama' ? (sortOrder === 'asc' ? 'ascending' : 'descending') : 'none'}
                    >
                      <SortableHeader
                        label="Identitas"
                        sortKey="nama"
                        activeKey={sortBy}
                        direction={sortOrder}
                        onSort={handleHeaderSort}
                      />
                    </th>
                    <th className="px-5 py-3.5">Koneksi</th>
                    <th
                      className="px-5 py-3.5"
                      aria-sort={sortBy === 'lokasi_area' ? (sortOrder === 'asc' ? 'ascending' : 'descending') : 'none'}
                    >
                      <SortableHeader
                        label="Lokasi"
                        sortKey="lokasi_area"
                        activeKey={sortBy}
                        direction={sortOrder}
                        onSort={handleHeaderSort}
                      />
                    </th>
                    <th
                      className="px-5 py-3.5"
                      aria-sort={sortBy === 'status' ? (sortOrder === 'asc' ? 'ascending' : 'descending') : 'none'}
                    >
                      <SortableHeader
                        label="Status"
                        sortKey="status"
                        activeKey={sortBy}
                        direction={sortOrder}
                        onSort={handleHeaderSort}
                      />
                    </th>
                    <th
                      className="px-5 py-3.5"
                      aria-sort={sortBy === 'health_score' ? (sortOrder === 'asc' ? 'ascending' : 'descending') : 'none'}
                    >
                      <SortableHeader
                        label="Kesehatan"
                        sortKey="health_score"
                        activeKey={sortBy}
                        direction={sortOrder}
                        onSort={handleHeaderSort}
                      />
                    </th>
                    <th className="px-5 py-3.5 text-right">Tindakan</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border text-sm">
                  {cctvs.map((cctv) => (
                    <tr key={cctv.id} className={`transition-colors hover:bg-background/35 ${selectedIDs.includes(cctv.id) ? 'bg-primary/8' : ''}`}>
                      {isAdmin && (
                        <td className="w-12 px-5 py-4 text-center">
                          <input
                            type="checkbox"
                            onChange={(e) => handleSelectRow(cctv.id, e.target.checked)}
                            checked={selectedIDs.includes(cctv.id)}
                            aria-label={`Pilih CCTV ${cctv.nama}`}
                            className="rounded border-gray-300 text-primary focus:ring-primary h-4 w-4 cursor-pointer"
                          />
                        </td>
                      )}
                      <td className="px-5 py-4">
                        <div className="min-w-0">
                          <h3 className="max-w-[240px] truncate font-semibold" title={cctv.id_channel ? `${cctv.nama} · Channel ${cctv.id_channel}` : cctv.nama}>{cctv.nama}</h3>
                          <div className="mt-1 flex items-center gap-1.5">
                            <span className="font-mono text-xs font-semibold text-primary">{cctv.kode_cctv}</span>
                            {cctv.id_channel && (
                              <span className="truncate rounded bg-background px-1.5 py-0.5 text-[10px] font-semibold text-muted">CH {cctv.id_channel}</span>
                            )}
                          </div>
                        </div>
                      </td>
                      <td className="px-5 py-4">
                        <div className="flex items-center gap-1.5">
                          <Network className="h-3.5 w-3.5 shrink-0 text-muted/50" aria-hidden="true" />
                          <span className="truncate font-mono text-xs text-muted" title={cctv.ip_camera || 'IP kamera belum dicatat'}>
                            {cctv.ip_camera || '—'}
                          </span>
                          {cctv.ip_nvr && (
                            <span className="shrink-0 rounded bg-background px-1.5 py-0.5 text-[10px] font-semibold text-muted">NVR</span>
                          )}
                        </div>
                      </td>
                      <td className="px-5 py-4">
                        <div className="flex items-start gap-1.5">
                          <MapPin className="mt-0.5 h-3.5 w-3.5 shrink-0 text-muted" />
                          <div>
                            <p className="max-w-[220px] truncate" title={cctv.lokasi_area}>{cctv.lokasi_area || 'Lokasi belum dicatat'}</p>
                            <p className="mt-0.5 text-xs capitalize text-muted">{cctv.indoor_outdoor || '-'}</p>
                          </div>
                        </div>
                      </td>
                      <td className="px-5 py-4">
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
                      <td className="px-5 py-4">
                        <span
                          className={`inline-flex min-w-14 items-center justify-center rounded-full border px-2.5 py-1 text-xs font-bold tabular-nums ${healthScoreClasses(cctv.health_score)}`}
                          title={`Skor kesehatan ${normalizedHealthScore(cctv.health_score)} dari 100`}
                          aria-label={`Skor kesehatan ${normalizedHealthScore(cctv.health_score)} dari 100`}
                        >
                          {normalizedHealthScore(cctv.health_score)}
                        </span>
                      </td>
                      <td className="px-5 py-4 text-right">
                        <div className="relative inline-flex justify-end">
                          <button
                            type="button"
                            onClick={(e) => {
                              e.stopPropagation();
                              setOpenActionMenu((cur) => (cur === cctv.id ? null : cctv.id));
                            }}
                            aria-label={`Aksi untuk CCTV ${cctv.nama}`}
                            aria-haspopup="menu"
                            aria-expanded={openActionMenu === cctv.id}
                            title="Aksi"
                            className="flex h-9 w-9 items-center justify-center rounded-lg border border-border text-muted transition-colors hover:border-primary/40 hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                          >
                            <MoreHorizontal className="h-4 w-4" />
                          </button>

                          {openActionMenu === cctv.id && (
                            <div
                              role="menu"
                              className="absolute right-0 top-11 z-50 w-44 overflow-hidden rounded-xl border border-border bg-card shadow-[0_12px_28px_rgba(32,63,37,0.14)]"
                              onClick={(e) => e.stopPropagation()}
                            >
                              <button
                                type="button"
                                role="menuitem"
                                onClick={() => {
                                  setOpenActionMenu(null);
                                  handleOpenDetail(cctv.id);
                                }}
                                disabled={actionLoading === `detail-${cctv.id}`}
                                className="flex w-full items-center gap-2.5 px-4 py-2.5 text-left text-sm font-semibold hover:bg-background disabled:opacity-60"
                              >
                                {actionLoading === `detail-${cctv.id}` ? <LoaderCircle className="h-4 w-4 animate-spin text-primary" /> : <Eye className="h-4 w-4 text-primary" />}
                                Detail
                              </button>
                              {isAdmin && (
                                <>
                                  <button
                                    type="button"
                                    role="menuitem"
                                    onClick={() => {
                                      setOpenActionMenu(null);
                                      handleOpenForm(cctv);
                                    }}
                                    className="flex w-full items-center gap-2.5 border-t border-border px-4 py-2.5 text-left text-sm font-semibold hover:bg-background"
                                  >
                                    <Edit2 className="h-4 w-4 text-primary" />
                                    Edit
                                  </button>
                                  <button
                                    type="button"
                                    role="menuitem"
                                    onClick={() => {
                                      setOpenActionMenu(null);
                                      handleRotateQR(cctv);
                                    }}
                                    disabled={actionLoading === `rotate-qr-${cctv.id}`}
                                    className="flex w-full items-center gap-2.5 border-t border-border px-4 py-2.5 text-left text-sm font-semibold hover:bg-background disabled:opacity-60"
                                  >
                                    {actionLoading === `rotate-qr-${cctv.id}` ? <LoaderCircle className="h-4 w-4 animate-spin text-primary" /> : <RotateCcw className="h-4 w-4 text-primary" />}
                                    Rotasi QR
                                  </button>
                                  <button
                                    type="button"
                                    role="menuitem"
                                    onClick={() => {
                                      setOpenActionMenu(null);
                                      handleDeleteCCTV(cctv);
                                    }}
                                    disabled={actionLoading === `delete-${cctv.id}`}
                                    className="flex w-full items-center gap-2.5 border-t border-border px-4 py-2.5 text-left text-sm font-semibold text-error hover:bg-error/10 disabled:opacity-60"
                                  >
                                    {actionLoading === `delete-${cctv.id}` ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                                    Hapus
                                  </button>
                                </>
                              )}
                            </div>
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
          </>
        )}
        </section>

        {/* CCTV Form Modal (Create/Edit) */}
        {/* Modal Sortir & Filter */}
        {isSortModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center px-4">
            <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => setIsSortModalOpen(false)} />

            <div
              role="dialog"
              aria-modal="true"
              aria-labelledby="cctv-sort-title"
              className="z-10 max-h-[90vh] w-full max-w-md overflow-y-auto rounded-2xl border border-border bg-card p-5 shadow-2xl sm:p-6"
            >
              <div className="mb-5 flex items-start justify-between gap-4">
                <div>
                  <h3 id="cctv-sort-title" className="text-xl font-bold tracking-[-0.015em]">Sortir &amp; Filter</h3>
                  <p className="mt-1 text-sm text-muted">Urutkan dan saring daftar CCTV.</p>
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
                  <label htmlFor="cctv-sort-by" className="mb-1.5 block text-xs font-semibold text-muted">Urutkan berdasarkan</label>
                  <select
                    id="cctv-sort-by"
                    value={draftSortBy}
                    onChange={(e) => setDraftSortBy(e.target.value)}
                    className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                  >
                    <option value="created_at">Waktu ditambahkan</option>
                    <option value="nama">Nama</option>
                    <option value="kode_cctv">Kode CCTV</option>
                    <option value="status">Status</option>
                    <option value="health_score">Skor kesehatan</option>
                    <option value="lokasi_area">Lokasi area</option>
                  </select>
                </div>

                <div>
                  <label htmlFor="cctv-sort-order" className="mb-1.5 block text-xs font-semibold text-muted">Arah pengurutan</label>
                  <select
                    id="cctv-sort-order"
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
                    <label htmlFor="cctv-sort-status-filter" className="mb-1.5 block text-xs font-semibold text-muted">Status</label>
                    <select
                      id="cctv-sort-status-filter"
                      value={draftStatusFilter}
                      onChange={(e) => setDraftStatusFilter(e.target.value)}
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    >
                      <option value="">Semua status</option>
                      <option value="aktif">Aktif</option>
                      <option value="nonaktif">Nonaktif</option>
                      <option value="pemeliharaan">Pemeliharaan</option>
                    </select>
                  </div>

                  <div>
                    <label htmlFor="cctv-sort-io-filter" className="mb-1.5 block text-xs font-semibold text-muted">Lokasi pemasangan</label>
                    <select
                      id="cctv-sort-io-filter"
                      value={draftIoFilter}
                      onChange={(e) => setDraftIoFilter(e.target.value)}
                      className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                    >
                      <option value="">Semua lokasi (indoor/outdoor)</option>
                      <option value="INDOOR">Indoor</option>
                      <option value="OUTDOOR">Outdoor</option>
                    </select>
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

        {isFormModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fadeIn">
            <div className="relative w-full max-w-2xl bg-card border border-border rounded-2xl p-6 shadow-2xl space-y-4 max-h-[90vh] overflow-y-auto">
              <div className="flex items-center justify-between border-b border-border pb-4">
                <h3 className="text-xl font-bold">{currentCCTV ? 'Ubah Data CCTV' : 'Tambah CCTV'}</h3>
                <button type="button" aria-label="Tutup formulir CCTV" onClick={() => setIsFormModalOpen(false)} className="p-1 rounded-lg border hover:bg-background text-gray-400 hover:text-foreground cursor-pointer">
                  <X className="h-5 w-5" />
                </button>
              </div>
              
              <form onSubmit={handleSaveCCTV} className="space-y-4">
                <p className="text-xs text-muted">Kolom bertanda * wajib diisi.</p>
                <div className="grid gap-4 md:grid-cols-2">
                  <CCTVTextField label="Kode CCTV *" required value={formData.kode_cctv} onChange={(value) => setFormData({ ...formData, kode_cctv: value })} placeholder="Contoh: IKN-THD-01" />
                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-muted">Status *</label>
                    <select value={formData.status} onChange={(e) => setFormData({ ...formData, status: e.target.value })} className="w-full rounded-lg border border-input-border bg-input px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-1 focus:ring-primary">
                      <option value="aktif">Aktif</option>
                      <option value="nonaktif">Nonaktif</option>
                      <option value="pemeliharaan">Pemeliharaan</option>
                    </select>
                  </div>
                  <CCTVTextField className="md:col-span-2" label="Nama CCTV *" required value={formData.nama} onChange={(value) => setFormData({ ...formData, nama: value })} placeholder="Contoh: CCTV Sumbu Kebangsaan Utama" />
                  <CCTVTextField className="md:col-span-2" label="Lokasi" value={formData.lokasi_area} onChange={(value) => setFormData({ ...formData, lokasi_area: value })} placeholder="Contoh: Kawasan Inti Pusat Pemerintahan" />
                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-muted">Pemasangan</label>
                    <select value={formData.indoor_outdoor} onChange={(e) => setFormData({ ...formData, indoor_outdoor: e.target.value })} className="w-full rounded-lg border border-input-border bg-input px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-1 focus:ring-primary">
                      <option value="INDOOR">Indoor</option>
                      <option value="OUTDOOR">Outdoor</option>
                    </select>
                  </div>
                </div>

                <details className="group rounded-xl border border-border" open={currentCCTV ? true : undefined}>
                  <summary className="flex cursor-pointer list-none items-center justify-between px-4 py-3 text-sm font-semibold [&::-webkit-details-marker]:hidden">
                    Informasi teknis
                    <span className="text-xs font-normal text-muted group-open:hidden">Tampilkan</span>
                    <span className="hidden text-xs font-normal text-muted group-open:inline">Sembunyikan</span>
                  </summary>
                  <div className="grid gap-4 border-t border-border p-4 md:grid-cols-2">
                    <CCTVTextField label="ID Channel" value={formData.id_channel} onChange={(value) => setFormData({ ...formData, id_channel: value })} placeholder="Contoh: 1" />
                    <CCTVTextField label="Jenis CCTV" value={formData.jenis_cctv} onChange={(value) => setFormData({ ...formData, jenis_cctv: value })} placeholder="Contoh: PTZ Dome" />
                    <CCTVTextField label="IP kamera" value={formData.ip_camera} onChange={(value) => setFormData({ ...formData, ip_camera: value })} placeholder="Contoh: 192.168.1.101" />
                    <CCTVTextField label="IP NVR" value={formData.ip_nvr} onChange={(value) => setFormData({ ...formData, ip_nvr: value })} placeholder="Contoh: 192.168.1.100" />
                    <CCTVTextField label="Tipe koneksi" value={formData.tipe_koneksi} onChange={(value) => setFormData({ ...formData, tipe_koneksi: value })} placeholder="Contoh: FO / 4G LTE" />
                    <CCTVTextField label="Sumber listrik" value={formData.sumber_listrik} onChange={(value) => setFormData({ ...formData, sumber_listrik: value })} placeholder="Contoh: Panel surya / PLN" />
                    <CCTVTextField label="Latitude" value={formData.latitude} onChange={(value) => setFormData({ ...formData, latitude: value })} placeholder="Contoh: -6.1949" />
                    <CCTVTextField label="Longitude" value={formData.longitude} onChange={(value) => setFormData({ ...formData, longitude: value })} placeholder="Contoh: 106.8231" />
                    <CCTVTextField className="md:col-span-2" label="Pemilik / pengelola" value={formData.pemilik} onChange={(value) => setFormData({ ...formData, pemilik: value })} placeholder="Contoh: Otorita IKN / Dishub" />
                    <div className="space-y-1.5 md:col-span-2">
                      <label className="text-xs font-semibold text-muted">Keterangan</label>
                      <textarea value={formData.keterangan} onChange={(e) => setFormData({ ...formData, keterangan: e.target.value })} placeholder="Tambahkan kondisi atau informasi teknis CCTV" rows={3} className="w-full resize-none rounded-lg border border-input-border bg-input px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-1 focus:ring-primary" />
                    </div>

                    {/* Kawasan & Zona operasional (Task 11.4 & 12.2) */}
                    <div className="space-y-1.5">
                      <label className="text-xs font-semibold text-muted">
                        Kawasan <span className="font-normal text-muted">(opsional)</span>
                      </label>
                      <select
                        value={formData.site_id}
                        onChange={(e) => setFormData({ ...formData, site_id: e.target.value })}
                        className="w-full rounded-lg border border-input-border bg-input px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-1 focus:ring-primary"
                      >
                        <option value="">— Pilih kawasan —</option>
                        {sites.map((s) => (
                          <option key={s.id} value={s.id}>{s.nama}</option>
                        ))}
                      </select>
                    </div>
                    <div className="space-y-1.5">
                      <label className="text-xs font-semibold text-muted">
                        Zona <span className="font-normal text-muted">(opsional)</span>
                      </label>
                      <select
                        value={formData.zone_id}
                        onChange={(e) => setFormData({ ...formData, zone_id: e.target.value })}
                        className="w-full rounded-lg border border-input-border bg-input px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-1 focus:ring-primary"
                      >
                        <option value="">— Pilih zona —</option>
                        {zones.map((z) => (
                          <option key={z.id} value={z.id}>{z.nama}</option>
                        ))}
                      </select>
                    </div>

                    {/* Konfigurasi Streaming */}
                    <div className="space-y-3 rounded-xl border border-border bg-background/40 p-4 md:col-span-2">
                      <p className="text-sm font-bold text-foreground">Konfigurasi Streaming</p>

                      <div className="space-y-1.5">
                        <label className="text-xs font-semibold text-muted">URL RTSP Kamera</label>
                        <input
                          type="text"
                          value={formData.rtsp_url}
                          onChange={(e) => setFormData({ ...formData, rtsp_url: e.target.value })}
                          placeholder="rtsp://admin:hik12345@36.91.27.35:554/Streaming/channels/1901"
                          className="w-full rounded-lg border border-input-border bg-input px-3 py-2 font-mono text-xs text-foreground outline-none focus:border-primary focus:ring-1 focus:ring-primary"
                        />
                        <p className="text-[11px] text-muted">
                          Isi URL RTSP penuh bila kamera punya sumber stream sendiri. Bila kosong, sistem akan menyusun
                          URL otomatis dari Nomor Channel NVR + kredensial di bawah.
                        </p>
                      </div>

                      <div className="space-y-1.5">
                        <label className="text-xs font-semibold text-muted">
                          Nomor Channel NVR <span className="font-normal text-muted">(opsional)</span>
                        </label>
                        <input
                          type="number"
                          min={0}
                          max={999}
                          value={formData.channel_port || ''}
                          onChange={(e) => setFormData({ ...formData, channel_port: e.target.value === '' ? 0 : Number(e.target.value) })}
                          placeholder="Opsional, mis. 19"
                          className="w-full rounded-lg border border-input-border bg-input px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-1 focus:ring-primary"
                        />
                        <p className="text-[11px] text-muted">
                          Opsional — cukup isi bila Anda <span className="font-semibold">tidak</span> memakai URL RTSP
                          penuh di atas. Ini nomor channel kamera (bukan port TCP; port seperti{' '}
                          <span className="font-mono">554</span> sudah ada di URL RTSP). Kosongkan bila sudah memakai URL
                          RTSP penuh.
                        </p>
                      </div>

                      <div className="space-y-1.5">
                        <label className="text-xs font-semibold text-muted">Password RTSP</label>
                        <div className="relative">
                          <input
                            type={showRtspPassword ? 'text' : 'password'}
                            value={formData.rtsp_password}
                            onChange={(e) => setFormData({ ...formData, rtsp_password: e.target.value })}
                            placeholder="Password akses NVR"
                            className="w-full rounded-lg border border-input-border bg-input px-3 py-2 pr-10 text-sm text-foreground outline-none focus:border-primary focus:ring-1 focus:ring-primary"
                          />
                          <button
                            type="button"
                            onClick={() => setShowRtspPassword((v) => !v)}
                            aria-label={showRtspPassword ? 'Sembunyikan password RTSP' : 'Tampilkan password RTSP'}
                            title={showRtspPassword ? 'Sembunyikan password' : 'Tampilkan password'}
                            className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-muted transition-colors hover:text-foreground"
                          >
                            {showRtspPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                          </button>
                        </div>
                      </div>

                      <div className="space-y-1.5">
                        <label className="text-xs font-semibold text-muted">Preview URI</label>
                        <p
                          className="w-full truncate rounded-lg border border-input-border bg-input px-3 py-2 font-mono text-xs text-muted"
                          title={
                            formData.rtsp_url ||
                            (formData.channel_port > 0
                              ? `rtsp://admin:***@36.91.27.35:554/Streaming/channels/${formData.channel_port * 100 + 2}`
                              : '')
                          }
                        >
                          {formData.rtsp_url
                            ? formData.rtsp_url
                            : formData.channel_port > 0
                              ? `rtsp://admin:***@36.91.27.35:554/Streaming/channels/${formData.channel_port * 100 + 2}`
                              : 'Isi URL RTSP atau channel port untuk melihat URI stream (sub stream).'}
                        </p>
                      </div>

                      <label className="flex cursor-pointer items-center gap-2 text-sm font-medium text-foreground">
                        <input
                          type="checkbox"
                          checked={formData.stream_enabled}
                          onChange={(e) => setFormData({ ...formData, stream_enabled: e.target.checked })}
                          className="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary focus:ring-primary"
                        />
                        Aktifkan Streaming
                      </label>
                    </div>
                  </div>
                </details>

                {formErrors.length > 0 && (
                  <div className="rounded-xl border border-red-500/20 bg-red-500/10 p-3 text-xs text-red-500">
                    <p className="font-semibold">Periksa isian berikut sebelum menyimpan:</p>
                    <ul className="mt-1 list-disc space-y-0.5 pl-4">
                      {formErrors.map((msg, i) => (
                        <li key={i}>{msg}</li>
                      ))}
                    </ul>
                  </div>
                )}

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
                    {actionLoading === 'save' ? 'Menyimpan data CCTV…' : currentCCTV ? 'Simpan Perubahan' : 'Tambah CCTV'}
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
                <h3 className="text-base font-bold">Pindai QR CCTV</h3>
                <button type="button" aria-label="Tutup pemindai QR" onClick={handleCloseScanModal} className="p-1 rounded-lg border hover:bg-background text-gray-400 hover:text-foreground cursor-pointer">
                  <X className="h-5 w-5" />
                </button>
              </div>

              {!scannedResult ? (
                <div className="space-y-3">
                  {/* CAMERA SCANNER VIEW — toggle input manual dihapus (Sprint 40 cleanup) */}
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

                  {scanError && (
                    <div className="p-3 rounded-lg bg-red-500/10 text-red-500 border border-red-500/20 text-xs flex items-center gap-2">
                      <AlertTriangle className="h-4 w-4 shrink-0" />
                      <span>{scanError}</span>
                    </div>
                  )}

                  <div className="flex justify-end">
                    <button
                      type="button"
                      onClick={handleCloseScanModal}
                      className="px-4 py-2 rounded-lg border bg-card text-foreground font-semibold text-sm hover:bg-background cursor-pointer"
                    >
                      Tutup
                    </button>
                  </div>
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

                  <div className="flex flex-wrap items-center justify-end gap-2 border-t border-border pt-4">
                    <button
                      type="button"
                      onClick={() => {
                        setScannedResult(null);
                        startCameraScanner();
                      }}
                      className="px-4 py-2 rounded-lg border bg-card text-foreground font-semibold text-sm hover:bg-background cursor-pointer"
                    >
                      Pindai Lagi
                    </button>
                    <button
                      type="button"
                      onClick={() => setIsFieldMaintenanceOpen(true)}
                      className="inline-flex items-center gap-2 px-5 py-2 rounded-lg bg-primary text-white font-semibold text-sm cursor-pointer shadow-md shadow-primary/15 hover:bg-primary-hover"
                      title="Catat hasil pengecekan lapangan"
                    >
                      <ClipboardCheck className="h-4 w-4" />
                      Maintenance
                    </button>
                    <button
                      type="button"
                      onClick={handleCloseScanModal}
                      className="px-4 py-2 rounded-lg border bg-card text-foreground font-semibold text-sm hover:bg-background cursor-pointer"
                    >
                      Tutup
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
                <button type="button" aria-label="Tutup impor Excel" onClick={() => { setIsImportModalOpen(false); setSelectedFile(null); setImportError(''); setImportSuccess(''); }} className="p-1 rounded-lg border hover:bg-background text-gray-400 hover:text-foreground cursor-pointer">
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
                    Unduh template
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
                    {importLoading ? 'Mengimpor data CCTV…' : 'Impor Data CCTV'}
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
                <button type="button" aria-label="Tutup label QR CCTV" onClick={() => setIsQrModalOpen(false)} className="p-1 rounded-lg border hover:bg-background text-gray-400 hover:text-foreground cursor-pointer">
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
                <CCTVQRImage
                  cctv={currentCCTV}
                  alt={`QR Code CCTV - ${currentCCTV.kode_cctv}`}
                  fetchQRDataUrl={fetchQRDataUrl}
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
                <h3 className="text-xl font-bold">Detail CCTV</h3>
                <button type="button" aria-label="Tutup detail CCTV" onClick={() => setIsDetailModalOpen(false)} className="p-1 rounded-lg border hover:bg-background text-gray-400 hover:text-foreground cursor-pointer">
                  <X className="h-5 w-5" />
                </button>
              </div>

              <div className="space-y-4">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <h4 className="text-lg font-bold">{currentCCTV.nama}</h4>
                    <p className="mt-1 font-mono text-xs font-semibold text-primary">
                      {currentCCTV.kode_cctv}{currentCCTV.id_channel ? ` · Channel ${currentCCTV.id_channel}` : ''}
                    </p>
                  </div>
                  <span className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-semibold capitalize ${
                    isStatus(currentCCTV.status, ['ACTIVE', 'AKTIF'])
                      ? 'border border-emerald-500/25 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                      : isStatus(currentCCTV.status, ['MAINTENANCE', 'PEMELIHARAAN'])
                        ? 'border border-amber-500/25 bg-amber-500/10 text-amber-600 dark:text-amber-400'
                        : 'border border-red-500/25 bg-red-500/10 text-red-600 dark:text-red-400'
                  }`}>
                    {statusLabel(currentCCTV.status)}
                  </span>
                </div>

                <dl className="grid overflow-hidden rounded-xl border border-border text-sm sm:grid-cols-2">
                  {[
                    { label: 'Lokasi', value: currentCCTV.lokasi_area || '—' },
                    { label: 'Pemasangan', value: currentCCTV.indoor_outdoor || '—' },
                    { label: 'IP kamera', value: currentCCTV.ip_camera || '—', technical: true },
                    { label: 'IP NVR', value: currentCCTV.ip_nvr || '—', technical: true },
                    { label: 'Koordinat', value: `${currentCCTV.latitude || '—'}, ${currentCCTV.longitude || '—'}`, technical: true },
                    { label: 'Jenis CCTV', value: currentCCTV.jenis_cctv || '—' },
                    { label: 'Koneksi', value: currentCCTV.tipe_koneksi || '—' },
                    { label: 'Sumber listrik', value: currentCCTV.sumber_listrik || '—' },
                    { label: 'Pengelola', value: currentCCTV.pemilik || '—' },
                    { label: 'Ditambahkan', value: new Date(currentCCTV.created_at).toLocaleDateString('id-ID', { year: 'numeric', month: 'long', day: 'numeric' }) },
                  ].map((item) => (
                    <div key={item.label} className="border-b border-border px-4 py-3 sm:odd:border-r">
                      <dt className="text-xs text-muted">{item.label}</dt>
                      <dd className={`mt-1 font-semibold ${item.technical ? 'font-mono text-xs' : ''}`}>{item.value}</dd>
                    </div>
                  ))}
                  <div className="px-4 py-3 sm:col-span-2">
                    <dt className="text-xs text-muted">Keterangan</dt>
                    <dd className="mt-1 whitespace-pre-wrap text-sm">{currentCCTV.keterangan || 'Tidak ada keterangan tambahan.'}</dd>
                  </div>
                </dl>
              </div>

              {/* Breakdown skor kesehatan — alasan penurunan skor */}
              <div className="space-y-3">
                <div className="flex items-center justify-between border-t border-border pt-4">
                  <h4 className="flex items-center gap-1.5 text-sm font-bold">
                    <ShieldCheck className="h-4 w-4 text-primary" />
                    Skor Kesehatan
                  </h4>
                  <span
                    className={`rounded-full border px-2.5 py-0.5 text-xs font-bold ${healthScoreClasses(healthBreakdown?.health_score ?? currentCCTV.health_score)}`}
                  >
                    {healthBreakdown?.health_score ?? currentCCTV.health_score}/100
                  </span>
                </div>

                {healthLoading ? (
                  <div className="flex items-center justify-center gap-2 py-4 text-sm text-muted" role="status">
                    <LoaderCircle className="h-4 w-4 animate-spin" />
                    <span>Menghitung skor kesehatan…</span>
                  </div>
                ) : healthError ? (
                  <div className="flex items-center gap-2 rounded-xl border border-red-500/20 bg-red-500/10 p-3 text-xs text-red-500">
                    <AlertTriangle className="h-4 w-4 shrink-0" />
                    <span>{healthError}</span>
                  </div>
                ) : healthBreakdown ? (
                  <div className="space-y-2 rounded-xl border border-border bg-background/50 p-3 text-sm">
                    <p className="text-xs text-muted">
                      Skor dihitung dari 5 faktor penalti selama 90 hari terakhir. Faktor di bawah ini yang menurunkan skor:
                    </p>
                    <ul className="space-y-1.5">
                      <HealthFactorRow label="Pemeliharaan gagal" value={healthBreakdown.breakdown.failed} active={healthBreakdown.breakdown.failed > 0} />
                      <HealthFactorRow label="Pengecekan butuh perhatian" value={healthBreakdown.breakdown.attention} active={healthBreakdown.breakdown.attention > 0} />
                      <HealthFactorRow label="Pemeliharaan lewat jatuh tempo" value={healthBreakdown.breakdown.overdue} active={healthBreakdown.breakdown.overdue > 0} />
                      <HealthFactorRow
                        label="Hari sejak pemeliharaan terakhir"
                        value={healthBreakdown.breakdown.age > 90 ? 91 : healthBreakdown.breakdown.age}
                        active={healthBreakdown.breakdown.age > 30}
                        suffix={healthBreakdown.breakdown.age > 90 ? ' (belum ada pemeliharaan selesai)' : ' hari'}
                      />
                      <HealthFactorRow label="Item checklist tidak OK" value={healthBreakdown.breakdown.not_ok_items} active={healthBreakdown.breakdown.not_ok_items > 0} />
                    </ul>
                    {healthBreakdown.breakdown.failed === 0 &&
                      healthBreakdown.breakdown.attention === 0 &&
                      healthBreakdown.breakdown.overdue === 0 &&
                      healthBreakdown.breakdown.not_ok_items === 0 &&
                      healthBreakdown.breakdown.age <= 30 && (
                        <p className="text-xs font-semibold text-emerald-600">Tidak ada faktor yang menurunkan skor.</p>
                      )}
                  </div>
                ) : null}
              </div>

              {/* Tren skor kesehatan (Task 13.8b) */}
              <div className="space-y-3">
                <div className="flex items-center justify-between border-t border-border pt-4">
                  <h4 className="flex items-center gap-1.5 text-sm font-bold">
                    <TrendingUp className="h-4 w-4 text-primary" />
                    Tren Kesehatan
                  </h4>
                  <div className="flex items-center gap-1 rounded-lg border border-border bg-background p-0.5">
                    {(['7d', '30d', '90d'] as const).map((range) => (
                      <button
                        key={range}
                        type="button"
                        onClick={() => setHealthHistoryRange(range)}
                        className={`rounded-md px-2.5 py-1 text-[11px] font-semibold transition-colors ${
                          healthHistoryRange === range ? 'bg-primary text-white' : 'text-muted hover:text-foreground'
                        }`}
                      >
                        {range === '7d' ? '7 Hari' : range === '30d' ? '30 Hari' : '90 Hari'}
                      </button>
                    ))}
                  </div>
                </div>

                {healthHistoryLoading ? (
                  <div className="space-y-2 rounded-xl border border-border bg-background/50 p-4" role="status">
                    <div className="h-4 w-24 animate-pulse rounded bg-border" />
                    <div className="h-32 w-full animate-pulse rounded bg-border/60" />
                  </div>
                ) : healthHistoryError ? (
                  <div className="flex items-center gap-2 rounded-xl border border-red-500/20 bg-red-500/10 p-3 text-xs text-red-500">
                    <AlertTriangle className="h-4 w-4 shrink-0" />
                    <span>{healthHistoryError}</span>
                  </div>
                ) : healthHistory.length === 0 ? (
                  <div className="flex items-center gap-2 rounded-xl border border-dashed border-border bg-background p-4 text-xs text-muted">
                    <TrendingUp className="h-4 w-4 shrink-0 text-muted/40" />
                    <span>Data histori skor kesehatan belum tersedia.</span>
                  </div>
                ) : (
                  <div className="rounded-xl border border-border bg-background/50 p-3">
                    <HealthTrendChart data={healthHistory} />
                  </div>
                )}
              </div>

              {/* Timeline kejadian CCTV */}
              <div className="space-y-3">
                <div className="flex items-center justify-between border-t border-border pt-4">
                  <h4 className="flex items-center gap-1.5 text-sm font-bold">
                    <History className="h-4 w-4 text-primary" />
                    Timeline
                  </h4>
                  <span className="text-xs text-muted">Riwayat terbaru</span>
                </div>

                {timelineLoading ? (
                  <div className="flex items-center justify-center gap-2 py-6 text-sm text-muted" role="status">
                    <LoaderCircle className="h-4 w-4 animate-spin" />
                    <span>Memuat riwayat CCTV…</span>
                  </div>
                ) : timelineError ? (
                  <div className="flex items-center gap-2 rounded-xl border border-red-500/20 bg-red-500/10 p-3 text-xs text-red-500">
                    <AlertTriangle className="h-4 w-4 shrink-0" />
                    <span>{timelineError}</span>
                  </div>
                ) : timeline.length === 0 ? (
                  <div className="flex items-center gap-2 rounded-xl border border-dashed border-border bg-background p-4 text-xs text-muted">
                    <History className="h-4 w-4 shrink-0 text-muted/40" />
                    <span>Belum ada riwayat untuk CCTV ini.</span>
                  </div>
                ) : (
                  <ol className="space-y-2.5">
                    {timeline.map((event) => (
                      <li
                        key={event.id}
                        className="flex items-start gap-3 rounded-xl border border-border bg-background/50 p-3"
                      >
                        <div className="flex h-[48px] w-[64px] shrink-0 items-center justify-center overflow-hidden rounded-lg border border-border">
                          <AuthenticatedImage
                            imageUrl={`/api/cctv/${currentCCTV.id}/snapshot`}
                            alt={`Snapshot ${currentCCTV.nama}`}
                            className="h-full w-full object-cover"
                          />
                        </div>
                        <div className="min-w-0 flex-1">
                          <p className="text-sm font-semibold leading-snug">
                            {eventTypeLabel(event.event_type)}
                          </p>
                          {event.notes && (
                            <p className="mt-0.5 line-clamp-2 text-xs text-muted">{event.notes}</p>
                          )}
                          <p className="mt-1 text-[11px] text-muted">
                            {formatTimelineDate(event.created_at)}
                          </p>
                        </div>
                      </li>
                    ))}
                  </ol>
                )}
              </div>

              <div className="flex gap-2 pt-4 border-t border-border justify-end">
                {currentCCTV.qr_payload && (
                  <button
                    type="button"
                    onClick={() => {
                      setIsDetailModalOpen(false);
                      setIsQrModalOpen(true);
                    }}
                    className="flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm font-semibold text-primary hover:bg-background"
                  >
                    <QrCode className="h-4 w-4" />
                    Lihat QR
                  </button>
                )}
                <button
                  type="button"
                  onClick={() => setIsDetailModalOpen(false)}
                  className="rounded-lg border border-border px-5 py-2 text-sm font-semibold hover:bg-background"
                >
                  Tutup
                </button>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Area cetak label QR — dirender via portal ke document.body, hanya tampil saat print */}
      {printQueue &&
        printQueue.length > 0 &&
        createPortal(
          <div id="sigap-print-area" className="sigap-print-area">
            {printMode === 'single' ? (
              <CCTVPrintSingleLabel item={printQueue[0]} logoUrl={printLogoUrl} />
            ) : (
              <CCTVPrintBatchLabels items={printQueue} logoUrl={printLogoUrl} />
            )}
          </div>,
          document.body
        )}

      {/* Modal pengecekan lapangan (hasil pindai QR) */}
      <FieldMaintenanceModal
        cctv={scannedResult}
        open={isFieldMaintenanceOpen}
        onClose={() => setIsFieldMaintenanceOpen(false)}
        onSaved={() => fetchCCTVs()}
      />
    </DashboardLayout>
  );
}

// Satu baris faktor penalti pada breakdown skor kesehatan.
// Grafik garis tren skor kesehatan (0-100) dengan zona warna, highlight titik
// minimum, dan tooltip. Komponen SVG mandiri (tanpa dependency chart eksternal).
function HealthTrendChart({ data }: { data: HealthHistoryPoint[] }) {
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);

  const W = 520;
  const H = 180;
  const PAD = { top: 14, right: 14, bottom: 28, left: 34 };
  const innerW = W - PAD.left - PAD.right;
  const innerH = H - PAD.top - PAD.bottom;

  if (data.length === 0) return null;

  // Cari titik nilai minimum untuk di-highlight.
  let minIndex = 0;
  data.forEach((p, i) => {
    if (p.health_score < data[minIndex].health_score) minIndex = i;
  });

  const yMax = 100;
  const stepX = data.length === 1 ? innerW / 2 : innerW / (data.length - 1);
  const xAt = (i: number) => PAD.left + i * stepX;
  const yAt = (value: number) => PAD.top + innerH - (value / yMax) * innerH;

  const points = data.map((p, i) => ({ x: xAt(i), y: yAt(p.health_score) }));

  // Path garis lurus antar titik.
  const linePath = points.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x} ${p.y}`).join(' ');

  // Zona warna: hijau 71-100, kuning 41-70, merah 0-40.
  const zoneGreenY = yAt(70);
  const zoneYellowY = yAt(40);

  const formatDate = (iso: string) => {
    const d = new Date(iso);
    return d.toLocaleDateString('id-ID', { day: '2-digit', month: 'short' });
  };

  return (
    <div className="relative w-full overflow-x-auto">
      <svg width={W} height={H} viewBox={`0 0 ${W} ${H}`} role="img" aria-label="Grafik tren skor kesehatan CCTV" className="block">
        {/* Zona warna latar */}
        <rect x={PAD.left} y={zoneGreenY} width={innerW} height={PAD.top + innerH - zoneGreenY} fill="#10B981" opacity={0.08} />
        <rect x={PAD.left} y={zoneYellowY} width={innerW} height={zoneGreenY - zoneYellowY} fill="#F59E0B" opacity={0.08} />
        <rect x={PAD.left} y={PAD.top} width={innerW} height={zoneYellowY - PAD.top} fill="#EF4444" opacity={0.08} />

        {/* Grid horizontal */}
        {[0, 25, 50, 75, 100].map((v) => {
          const y = yAt(v);
          return (
            <g key={v}>
              <line x1={PAD.left} x2={W - PAD.right} y1={y} y2={y} className="stroke-border" strokeWidth={1} />
              <text x={PAD.left - 6} y={y + 3} textAnchor="end" fontSize={9} className="fill-muted">{v}</text>
            </g>
          );
        })}

        {/* Garis tren */}
        <path d={linePath} fill="none" stroke="#467235" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round" />

        {/* Titik data */}
        {points.map((p, i) => (
          <circle
            key={i}
            cx={p.x}
            cy={p.y}
            r={i === minIndex ? 5 : 3.5}
            fill={i === minIndex ? '#EF4444' : '#467235'}
            stroke="#ffffff"
            strokeWidth={1.5}
          />
        ))}

        {/* Label sumbu X */}
        {data.map((p, i) => (
          <text key={i} x={points[i].x} y={H - 8} textAnchor="middle" fontSize={9} className="fill-muted">
            {formatDate(p.calculated_at)}
          </text>
        ))}

        {/* Area hover + tooltip */}
        {points.map((p, i) => (
          <circle
            key={`hover-${i}`}
            cx={p.x}
            cy={p.y}
            r={14}
            fill="transparent"
            className="cursor-pointer"
            onMouseEnter={() => setHoveredIndex(i)}
            onMouseLeave={() => setHoveredIndex(null)}
          >
            <title>{`${new Date(data[i].calculated_at).toLocaleString('id-ID')}: ${data[i].health_score}`}</title>
          </circle>
        ))}

        {/* Garis penanda hover */}
        {hoveredIndex !== null && points[hoveredIndex] && (
          <line
            x1={points[hoveredIndex].x}
            x2={points[hoveredIndex].x}
            y1={PAD.top}
            y2={PAD.top + innerH}
            stroke="#467235"
            strokeWidth={1.5}
            strokeDasharray="4 4"
            opacity={0.5}
          />
        )}
      </svg>

      {/* Tooltip */}
      {hoveredIndex !== null && data[hoveredIndex] && (
        <div
          className="pointer-events-none absolute z-10 -translate-x-1/2 -translate-y-full rounded-lg border border-border bg-card px-2.5 py-1.5 text-xs shadow-lg"
          style={{ left: `${(points[hoveredIndex].x / W) * 100}%`, top: '0' }}
        >
          <p className="font-bold">{new Date(data[hoveredIndex].calculated_at).toLocaleString('id-ID')}</p>
          <p className="mt-0.5 text-muted">Skor: {data[hoveredIndex].health_score}</p>
        </div>
      )}
    </div>
  );
}

function HealthFactorRow({
  label,
  value,
  active,
  suffix = '',
}: {
  label: string;
  value: number;
  active: boolean;
  suffix?: string;
}) {
  return (
    <li className="flex items-center justify-between gap-3">
      <span className={`text-xs ${active ? 'font-semibold text-foreground' : 'text-muted'}`}>
        {label}
        {suffix}
      </span>
      <span
        className={`inline-flex min-w-7 items-center justify-center rounded-full px-2 py-0.5 text-[11px] font-bold ${
          active ? 'bg-amber-500/15 text-amber-600 dark:text-amber-400' : 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
        }`}
      >
        {value}
      </span>
    </li>
  );
}

function CCTVTextField({
  label,
  value,
  onChange,
  placeholder,
  required = false,
  className = '',
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  required?: boolean;
  className?: string;
}) {
  return (
    <label className={`block space-y-1.5 ${className}`}>
      <span className="text-xs font-semibold text-muted">{label}</span>
      <input
        type="text"
        required={required}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        className="w-full rounded-lg border border-input-border bg-input px-3 py-2 text-sm text-foreground outline-none focus:border-primary focus:ring-1 focus:ring-primary"
      />
    </label>
  );
}

// Menampilkan gambar QR yang di-generate server-side; mengambil gambar saat pertama kali dirender
function CCTVQRImage({
  cctv,
  className,
  alt = '',
  fetchQRDataUrl,
}: {
  cctv: CCTV;
  className?: string;
  alt?: string;
  fetchQRDataUrl: (cctv: CCTV) => Promise<string>;
}) {
  const [src, setSrc] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let active = true;
    setSrc(null);
    setFailed(false);
    fetchQRDataUrl(cctv)
      .then((dataUrl) => {
        if (active) setSrc(dataUrl);
      })
      .catch(() => {
        if (active) setFailed(true);
      });
    return () => {
      active = false;
    };
  }, [cctv, fetchQRDataUrl]);

  if (failed || !src) return null;
  return <img src={src} alt={alt} className={className} />;
}

// Label QR tunggal (100 x 38 mm) untuk printer label thermal
function CCTVPrintSingleLabel({ item, logoUrl }: { item: QRPrintItem; logoUrl: string }) {
  return (
    <div className="sigap-label sigap-label-single">
      <div className="sigap-brand">
        <img src={logoUrl} alt="Nusantara" />
      </div>
      <img className="sigap-qr" src={item.qrDataUrl} alt="QR Code" />
      <div className="sigap-details">
        <p className="sigap-name">{item.cctv.nama}</p>
        <p className="sigap-meta sigap-code">{item.cctv.kode_cctv}</p>
        <p className="sigap-meta">{item.cctv.lokasi_area || '-'}</p>
      </div>
    </div>
  );
}

// Label QR dalam grid A4 untuk cetak batch (banyak perangkat)
function CCTVPrintBatchLabels({ items, logoUrl }: { items: QRPrintItem[]; logoUrl: string }) {
  const formattedDate = new Date().toLocaleDateString('id-ID', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });

  return (
    <div className="sigap-batch">
      <div className="sigap-batch-header">
        <div>
          <h1>SIGAP — Label QR Code Perangkat CCTV</h1>
          <p>Dicetak pada {formattedDate} • Total: {items.length} Perangkat CCTV</p>
        </div>
        <img src={logoUrl} alt="Logo Nusantara" className="sigap-batch-logo" />
      </div>
      <div className="sigap-batch-grid">
        {items.map((item) => (
          <div className="sigap-label sigap-label-batch" key={item.cctv.id}>
            <div className="sigap-brand">
              <img src={logoUrl} alt="Nusantara" />
            </div>
            <img className="sigap-qr" src={item.qrDataUrl} alt="QR Code" />
            <div className="sigap-details">
              <p className="sigap-name">{item.cctv.nama}</p>
              <p className="sigap-meta sigap-code">{item.cctv.kode_cctv}</p>
              <p className="sigap-meta">{item.cctv.lokasi_area || '-'}</p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

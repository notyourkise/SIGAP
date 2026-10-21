'use client';

import React, { useCallback, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/DashboardLayout';
import { apiRequest, API_BASE_URL, getCookie } from '@/lib/api';
import { useToast } from '@/components/Toast';
import { useConfirm } from '@/components/ConfirmDialog';
import { useAuth } from '@/context/AuthContext';
import {
  FileSpreadsheet,
  Download,
  Save,
  Trash2,
  LoaderCircle,
  AlertTriangle,
  CheckSquare,
  FileText,
  Layers,
  SlidersHorizontal,
  FolderOpen,
} from 'lucide-react';

// Definisi entitas dan kolom laporan (konsisten dengan backend).
const ENTITIES: Record<string, { label: string; columns: { key: string; label: string }[] }> = {
  cctv: {
    label: 'CCTV',
    columns: [
      { key: 'kode_cctv', label: 'Kode CCTV' },
      { key: 'nama', label: 'Nama' },
      { key: 'lokasi_area', label: 'Lokasi' },
      { key: 'jenis_cctv', label: 'Jenis' },
      { key: 'status', label: 'Status' },
      { key: 'ip_camera', label: 'IP Kamera' },
      { key: 'ip_nvr', label: 'IP NVR' },
      { key: 'pemilik', label: 'Pemilik' },
      { key: 'created_at', label: 'Dibuat' },
    ],
  },
  maintenance_log: {
    label: 'Pemeliharaan',
    columns: [
      { key: 'cctv_nama', label: 'CCTV' },
      { key: 'cctv_kode', label: 'Kode CCTV' },
      { key: 'action_taken', label: 'Tindakan' },
      { key: 'status', label: 'Status' },
      { key: 'technician', label: 'Teknisi' },
      { key: 'scheduled_date', label: 'Terjadwal' },
      { key: 'completed_at', label: 'Selesai' },
    ],
  },
  field_maintenance: {
    label: 'Pengecekan',
    columns: [
      { key: 'cctv_nama', label: 'CCTV' },
      { key: 'cctv_kode', label: 'Kode CCTV' },
      { key: 'technician', label: 'Teknisi' },
      { key: 'overall_status', label: 'Hasil' },
      { key: 'notes', label: 'Catatan' },
      { key: 'completed_at', label: 'Waktu' },
    ],
  },
  spare_part: {
    label: 'Suku Cadang',
    columns: [
      { key: 'nama', label: 'Nama' },
      { key: 'sku', label: 'SKU' },
      { key: 'kategori', label: 'Kategori' },
      { key: 'stok', label: 'Stok' },
      { key: 'minimum_order', label: 'Min. Order' },
      { key: 'satuan', label: 'Satuan' },
      { key: 'harga_satuan', label: 'Harga' },
    ],
  },
};

const FORMATS = [
  { value: 'csv', label: 'CSV' },
  { value: 'xlsx', label: 'Excel' },
  { value: 'pdf', label: 'PDF' },
];

const EMPTY_FILTERS: Record<string, string> = {
  status: '',
  kategori: '',
  low_stock: '',
  start_date: '',
  end_date: '',
};

interface ReportTemplate {
  id: string;
  nama: string;
  config: string;
  created_by: string;
  created_at: string;
}

// Mengunduh laporan dari POST /api/reports/generate dengan autentikasi cookie + CSRF.
async function downloadReport(payload: Record<string, unknown>): Promise<{ blob: Blob; filename: string }> {
  const csrf = getCookie('sigap_csrf');
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (csrf) headers['X-CSRF-Token'] = csrf;

  const response = await fetch(`${API_BASE_URL}/reports/generate`, {
    method: 'POST',
    headers,
    credentials: 'include',
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    let message = 'Gagal membuat laporan';
    try {
      const data = await response.json();
      message = data.message || message;
    } catch {
      // respons error bukan JSON
    }
    throw new Error(message);
  }

  const blob = await response.blob();
  const disposition = response.headers.get('content-disposition');
  const filename = disposition?.match(/filename="?([^"]+)"?/i)?.[1] || 'laporan';
  return { blob, filename };
}

export default function ReportBuilderPage() {
  const toast = useToast();
  const confirm = useConfirm();
  const router = useRouter();
  const { user, loading: authLoading } = useAuth();

  // Step: entitas
  const [entity, setEntity] = useState('cctv');
  // Step: kolom terpilih
  const [columns, setColumns] = useState<string[]>([]);
  // Step: filter
  const [filters, setFilters] = useState<Record<string, string>>(EMPTY_FILTERS);
  // Step: format
  const [format, setFormat] = useState('csv');

  // Template
  const [templates, setTemplates] = useState<ReportTemplate[]>([]);
  const [templateName, setTemplateName] = useState('');
  const [activeTemplateId, setActiveTemplateId] = useState<string | null>(null);

  // UI state
  const [downloading, setDownloading] = useState(false);
  const [savingTemplate, setSavingTemplate] = useState(false);
  const [loadingTemplates, setLoadingTemplates] = useState(true);
  const [error, setError] = useState('');

  const currentEntity = ENTITIES[entity];

  // Pertahanan berlapis: proxy dan API sudah membatasi admin, sementara guard
  // klien ini mencegah konten admin sempat terlihat pada navigasi sisi klien.
  useEffect(() => {
    if (!authLoading && user?.role !== 'admin') router.replace('/dashboard');
  }, [authLoading, router, user?.role]);

  // Saat entitas berubah, reset kolom terpilih ke semua kolom.
  const handleEntityChange = (next: string) => {
    setEntity(next);
    setColumns(ENTITIES[next].columns.map((c) => c.key));
  };

  // Muat daftar template.
  const fetchTemplates = useCallback(async () => {
    setLoadingTemplates(true);
    try {
      const response = await apiRequest<ReportTemplate[]>('/reports/templates');
      if (response.status === 'success' && Array.isArray(response.data)) {
        setTemplates(response.data);
      } else {
        setTemplates([]);
      }
    } catch {
      setTemplates([]);
    } finally {
      setLoadingTemplates(false);
    }
  }, []);

  useEffect(() => {
    if (user?.role !== 'admin') return;
    let cancelled = false;
    void Promise.resolve().then(() => {
      if (!cancelled) void fetchTemplates();
    });
    return () => {
      cancelled = true;
    };
  }, [fetchTemplates, user?.role]);

  const handleDownload = async () => {
    if (columns.length === 0) {
      toast.error('Pilih minimal satu kolom untuk laporan.');
      return;
    }
    setDownloading(true);
    setError('');
    try {
      const { blob, filename } = await downloadReport({
        entity,
        columns,
        filters: normalizeFilters(),
        format,
      });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = filename;
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      URL.revokeObjectURL(url);
      toast.success('Laporan berhasil diunduh.');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Laporan belum dapat dibuat. Coba lagi.');
      toast.error(err instanceof Error ? err.message : 'Laporan belum dapat dibuat. Coba lagi.');
    } finally {
      setDownloading(false);
    }
  };

  const normalizeFilters = () => {
    const out: Record<string, string> = {};
    for (const [k, v] of Object.entries(filters)) {
      if (v !== '' && v !== null) out[k] = v;
    }
    return out;
  };

  const handleSaveTemplate = async () => {
    if (!templateName.trim()) {
      toast.error('Tuliskan nama template terlebih dahulu.');
      return;
    }
    setSavingTemplate(true);
    setError('');
    try {
      const config = JSON.stringify({
        entity,
        columns,
        filters: normalizeFilters(),
        format,
      });
      const response = await apiRequest<ReportTemplate>('/reports/templates', {
        method: 'POST',
        body: JSON.stringify({ nama: templateName.trim(), config }),
      });
      if (response.status === 'success') {
        setTemplateName('');
        toast.success('Template laporan berhasil disimpan.');
        void fetchTemplates();
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Template belum dapat disimpan. Coba lagi.');
      toast.error(err instanceof Error ? err.message : 'Template belum dapat disimpan. Coba lagi.');
    } finally {
      setSavingTemplate(false);
    }
  };

  const handleLoadTemplate = (tmpl: ReportTemplate) => {
    try {
      const config = JSON.parse(tmpl.config) as {
        entity?: string;
        columns?: string[];
        filters?: Record<string, string>;
        format?: string;
      };
      if (config.entity && ENTITIES[config.entity]) {
        setEntity(config.entity);
        setColumns(config.columns || ENTITIES[config.entity].columns.map((c) => c.key));
        setFilters({ ...EMPTY_FILTERS, ...(config.filters || {}) });
      }
      if (config.format) setFormat(config.format);
      setActiveTemplateId(tmpl.id);
      toast.success(`Template "${tmpl.nama}" dimuat.`);
    } catch {
      toast.error('Config template tidak valid.');
    }
  };

  const handleDeleteTemplate = async (tmpl: ReportTemplate) => {
    const ok = await confirm({
      title: 'Hapus Template Laporan?',
      message: `Hapus template "${tmpl.nama}"? Tindakan ini tidak dapat dibatalkan.`,
      confirmLabel: 'Hapus',
    });
    if (!ok) return;
    try {
      const response = await apiRequest(`/reports/templates/${tmpl.id}`, { method: 'DELETE' });
      if (response.status === 'success') {
        toast.success('Template laporan berhasil dihapus.');
        void fetchTemplates();
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Template belum dapat dihapus. Coba lagi.');
    }
  };

  const toggleColumn = (key: string) => {
    setColumns((prev) =>
      prev.includes(key) ? prev.filter((c) => c !== key) : [...prev, key]
    );
  };

  const templateSummary = (tmpl: ReportTemplate) => {
    try {
      const config = JSON.parse(tmpl.config) as { entity?: string; columns?: string[]; format?: string };
      const entityLabel = config.entity && ENTITIES[config.entity]?.label
        ? ENTITIES[config.entity].label
        : 'Laporan';
      const columnCount = Array.isArray(config.columns) ? config.columns.length : 0;
      return `${entityLabel} · ${columnCount} kolom · ${(config.format || 'csv').toUpperCase()}`;
    } catch {
      return 'Konfigurasi template tidak tersedia';
    }
  };

  if (authLoading || user?.role !== 'admin') {
    return (
      <DashboardLayout>
        <div className="flex min-h-[45vh] items-center justify-center" role="status" aria-live="polite">
          <div className="flex items-center gap-3 text-sm font-medium text-muted">
            <LoaderCircle className="h-5 w-5 animate-spin text-primary" />
            Memeriksa izin akses admin…
          </div>
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <header>
          <h1 className="text-3xl font-bold tracking-[-0.025em]">Laporan Kustom</h1>
          <p className="mt-1 max-w-2xl text-sm text-muted">
            Bangun laporan sesuai kebutuhan: pilih entitas, kolom, filter, dan format.
          </p>
        </header>

        {error && (
          <div role="alert" className="flex items-center gap-3 rounded-xl border border-error/20 bg-error/10 p-3.5 text-sm text-error">
            <AlertTriangle className="h-4 w-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_340px]">
          {/* Kolom kiri: builder */}
          <div className="space-y-6">
            {/* Step 1: Entitas */}
            <section aria-labelledby="entity-heading" className="overflow-hidden rounded-2xl border border-border bg-card">
              <div className="flex items-center gap-2 border-b border-border px-5 py-4">
                <Layers className="h-4 w-4 text-primary" />
                <h2 id="entity-heading" className="text-base font-bold">1 · Pilih Entitas</h2>
              </div>
              <div className="grid gap-3 p-5 sm:grid-cols-2">
                {Object.entries(ENTITIES).map(([key, meta]) => (
                  <button
                    key={key}
                    type="button"
                    onClick={() => handleEntityChange(key)}
                    className={`flex items-center justify-between gap-3 rounded-xl border px-4 py-3 text-left text-sm font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 ${
                      entity === key
                        ? 'border-primary bg-primary/10 text-foreground'
                        : 'border-border bg-background text-muted hover:border-primary/40'
                    }`}
                  >
                    <FileSpreadsheet className="h-4 w-4 shrink-0" />
                    {meta.label}
                    <CheckSquare className={`h-4 w-4 ${entity === key ? 'text-primary' : 'text-muted/40'}`} />
                  </button>
                ))}
              </div>
            </section>

            {/* Step 2: Kolom */}
            <section aria-labelledby="columns-heading" className="overflow-hidden rounded-2xl border border-border bg-card">
              <div className="flex items-center justify-between gap-3 border-b border-border px-5 py-4">
                <div className="flex items-center gap-2">
                  <CheckSquare className="h-4 w-4 text-primary" />
                  <h2 id="columns-heading" className="text-base font-bold">2 · Pilih Kolom</h2>
                </div>
                <button
                  type="button"
                  onClick={() => {
                    if (columns.length === currentEntity.columns.length) {
                      setColumns([]);
                    } else {
                      setColumns(currentEntity.columns.map((c) => c.key));
                    }
                  }}
                  className="text-xs font-semibold text-primary hover:underline"
                >
                  {columns.length === currentEntity.columns.length ? 'Kosongkan' : 'Pilih semua'}
                </button>
              </div>
              <div className="grid gap-2 p-5 sm:grid-cols-2">
                {currentEntity.columns.map((col) => {
                  const checked = columns.includes(col.key);
                  return (
                    <label
                      key={col.key}
                      className="flex cursor-pointer items-center gap-2.5 rounded-xl border border-border bg-background px-3 py-2.5 text-sm transition-colors hover:border-primary/40"
                    >
                      <input
                        type="checkbox"
                        checked={checked}
                        onChange={() => toggleColumn(col.key)}
                        className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
                      />
                      <span className={checked ? 'font-medium text-foreground' : 'text-muted'}>
                        {col.label}
                      </span>
                    </label>
                  );
                })}
              </div>
            </section>

            {/* Step 3: Filter */}
            <section aria-labelledby="filters-heading" className="overflow-hidden rounded-2xl border border-border bg-card">
              <div className="flex items-center gap-2 border-b border-border px-5 py-4">
                <SlidersHorizontal className="h-4 w-4 text-primary" />
                <h2 id="filters-heading" className="text-base font-bold">3 · Filter</h2>
              </div>
              <div className="grid gap-4 p-5 sm:grid-cols-2">
                {(entity === 'maintenance_log' || entity === 'field_maintenance') && (
                  <div>
                    <label htmlFor="filter-status" className="mb-1.5 block text-xs font-semibold text-muted">Status</label>
                    <select
                      id="filter-status"
                      value={filters.status}
                      onChange={(e) => setFilters({ ...filters, status: e.target.value })}
                      className="w-full rounded-xl border border-input-border bg-input px-3 py-2.5 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
                    >
                      <option value="">Semua</option>
                      <option value="scheduled">Terjadwal</option>
                      <option value="ongoing">Dalam proses</option>
                      <option value="completed">Selesai</option>
                      <option value="failed">Perlu Tindak Lanjut</option>
                    </select>
                  </div>
                )}
                {(entity === 'spare_part') && (
                  <>
                    <div>
                      <label htmlFor="filter-kategori" className="mb-1.5 block text-xs font-semibold text-muted">Kategori</label>
                      <select
                        id="filter-kategori"
                        value={filters.kategori}
                        onChange={(e) => setFilters({ ...filters, kategori: e.target.value })}
                        className="w-full rounded-xl border border-input-border bg-input px-3 py-2.5 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
                      >
                        <option value="">Semua</option>
                        {['Kabel', 'Konektor', 'Power Supply', 'Lensa', 'Storage', 'Lainnya'].map((k) => (
                          <option key={k} value={k}>{k}</option>
                        ))}
                      </select>
                    </div>
                    <div>
                      <label htmlFor="filter-lowstock" className="mb-1.5 block text-xs font-semibold text-muted">Stok</label>
                      <select
                        id="filter-lowstock"
                        value={filters.low_stock}
                        onChange={(e) => setFilters({ ...filters, low_stock: e.target.value })}
                        className="w-full rounded-xl border border-input-border bg-input px-3 py-2.5 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
                      >
                        <option value="">Semua</option>
                        <option value="true">Stok rendah saja</option>
                      </select>
                    </div>
                  </>
                )}
                {(entity === 'cctv') && (
                  <div>
                    <label htmlFor="filter-cctv-status" className="mb-1.5 block text-xs font-semibold text-muted">Status CCTV</label>
                    <select
                      id="filter-cctv-status"
                      value={filters.status}
                      onChange={(e) => setFilters({ ...filters, status: e.target.value })}
                      className="w-full rounded-xl border border-input-border bg-input px-3 py-2.5 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
                    >
                      <option value="">Semua</option>
                      <option value="aktif">Aktif</option>
                      <option value="pemeliharaan">Pemeliharaan</option>
                      <option value="nonaktif">Nonaktif</option>
                    </select>
                  </div>
                )}
                {(entity === 'maintenance_log') && (
                  <>
                    <div>
                      <label htmlFor="filter-start" className="mb-1.5 block text-xs font-semibold text-muted">Dari tanggal</label>
                      <input
                        id="filter-start"
                        type="date"
                        value={filters.start_date}
                        onChange={(e) => setFilters({ ...filters, start_date: e.target.value })}
                        className="w-full rounded-xl border border-input-border bg-input px-3 py-2.5 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
                      />
                    </div>
                    <div>
                      <label htmlFor="filter-end" className="mb-1.5 block text-xs font-semibold text-muted">Sampai tanggal</label>
                      <input
                        id="filter-end"
                        type="date"
                        value={filters.end_date}
                        onChange={(e) => setFilters({ ...filters, end_date: e.target.value })}
                        className="w-full rounded-xl border border-input-border bg-input px-3 py-2.5 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
                      />
                    </div>
                  </>
                )}
              </div>
            </section>

            {/* Step 4: Format + Download */}
            <section aria-labelledby="format-heading" className="overflow-hidden rounded-2xl border border-border bg-card">
              <div className="flex items-center gap-2 border-b border-border px-5 py-4">
                <FileText className="h-4 w-4 text-primary" />
                <h2 id="format-heading" className="text-base font-bold">4 · Format & Unduh</h2>
              </div>
              <div className="p-5">
                <div className="flex flex-wrap gap-2">
                  {FORMATS.map((f) => (
                    <button
                      key={f.value}
                      type="button"
                      onClick={() => setFormat(f.value)}
                      className={`rounded-xl border px-4 py-2.5 text-sm font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 ${
                        format === f.value
                          ? 'border-primary bg-primary text-white'
                          : 'border-border bg-background text-muted hover:border-primary/40'
                      }`}
                    >
                      {f.label}
                    </button>
                  ))}
                </div>
                <button
                  type="button"
                  onClick={handleDownload}
                  disabled={downloading || columns.length === 0}
                  className="mt-4 inline-flex items-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white shadow-md shadow-primary/20 transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-card disabled:cursor-wait disabled:opacity-60"
                >
                  {downloading ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />}
                  {downloading ? 'Membuat laporan…' : 'Unduh Laporan'}
                </button>
              </div>
            </section>
          </div>

          {/* Kolom kanan: template tersimpan */}
          <div className="space-y-6">
            <section aria-labelledby="save-template-heading" className="overflow-hidden rounded-2xl border border-border bg-card">
              <div className="flex items-center gap-2 border-b border-border px-5 py-4">
                <Save className="h-4 w-4 text-primary" />
                <h2 id="save-template-heading" className="text-base font-bold">Simpan Template</h2>
              </div>
              <div className="space-y-3 p-5">
                <input
                  type="text"
                  value={templateName}
                  onChange={(e) => setTemplateName(e.target.value)}
                  placeholder="Nama template (mis. Laporan CCTV Bulanan)"
                  className="w-full rounded-xl border border-input-border bg-input px-3 py-2.5 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
                />
                <button
                  type="button"
                  onClick={handleSaveTemplate}
                  disabled={savingTemplate}
                  className="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-primary px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-primary-hover disabled:cursor-wait disabled:opacity-60"
                >
                  {savingTemplate ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                  Simpan Template
                </button>
              </div>
            </section>

            <section aria-labelledby="templates-heading" className="overflow-hidden rounded-2xl border border-border bg-card">
              <div className="border-b border-border px-5 py-4">
                <h2 id="templates-heading" className="text-base font-bold">Template Tersimpan</h2>
                <p className="mt-0.5 text-xs text-muted">{templates.length} template</p>
              </div>
              {loadingTemplates ? (
                <div className="flex items-center justify-center gap-2 py-8 text-sm text-muted" role="status">
                  <LoaderCircle className="h-4 w-4 animate-spin" />
                  Memuat template…
                </div>
              ) : templates.length === 0 ? (
                <p className="px-5 py-6 text-sm text-muted">Belum ada template tersimpan.</p>
              ) : (
                <div className="divide-y divide-border">
                  {templates.map((tmpl) => (
                    <div key={tmpl.id} className="px-5 py-3.5">
                      <div className="flex items-start justify-between gap-3">
                        <div className="min-w-0">
                          <div className="flex items-center gap-2">
                            <p className="truncate text-sm font-semibold">{tmpl.nama}</p>
                            {activeTemplateId === tmpl.id && (
                              <span className="shrink-0 rounded-md bg-primary/10 px-1.5 py-0.5 text-[10px] font-bold text-primary">
                                Digunakan
                              </span>
                            )}
                          </div>
                          <p className="mt-1 truncate text-xs text-muted" title={templateSummary(tmpl)}>
                            {templateSummary(tmpl)}
                          </p>
                          <p className="mt-0.5 text-[11px] text-muted">Disimpan {tmpl.created_at?.slice(0, 10)}</p>
                        </div>
                        <button
                          type="button"
                          onClick={() => handleDeleteTemplate(tmpl)}
                          className="shrink-0 rounded-lg p-2 text-muted transition-colors hover:bg-error/10 hover:text-error focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-error/40"
                          aria-label={`Hapus template ${tmpl.nama}`}
                          title="Hapus template"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                      <button
                        type="button"
                        onClick={() => handleLoadTemplate(tmpl)}
                        className="mt-3 inline-flex min-h-9 w-full items-center justify-center gap-2 rounded-lg border border-border bg-background px-3 text-xs font-bold text-primary transition-colors hover:border-primary/40 hover:bg-primary/5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                      >
                        <FolderOpen className="h-3.5 w-3.5" />
                        {activeTemplateId === tmpl.id ? 'Muat Ulang Template' : 'Gunakan Template'}
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </section>
          </div>
        </div>
      </div>
    </DashboardLayout>
  );
}

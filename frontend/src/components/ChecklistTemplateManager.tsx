'use client';

import React, { useCallback, useEffect, useState } from 'react';
import {
  Plus,
  Trash2,
  Edit2,
  X,
  LoaderCircle,
  AlertTriangle,
  ArrowUp,
  ArrowDown,
  ListChecks,
  Check,
} from 'lucide-react';
import {
  getChecklistTemplates,
  createChecklistTemplate,
  updateChecklistTemplate,
  deleteChecklistTemplate,
  reorderChecklistTemplates,
  getAllSitesForSelect,
  type ChecklistTemplate,
} from '@/lib/api';
import { useToast } from '@/components/Toast';
import { useConfirm } from '@/components/ConfirmDialog';

// Mode tampilan komponen:
// - "page"  → dirender penuh sebagai halaman admin (/checklist-templates).
// - "modal" → dirender di dalam panel/modal (mis. dibuka dari field-maintenance).
type ViewMode = 'page' | 'modal';

interface ChecklistTemplateManagerProps {
  mode?: ViewMode;
  onClose?: () => void;
}

// State formulir tambah/edit item checklist.
interface TemplateFormState {
  id: string | null;
  nama_item: string;
  deskripsi: string;
  urutan: number;
  wajib: boolean;
  site_id: string;
}

export default function ChecklistTemplateManager({
  mode = 'page',
  onClose,
}: ChecklistTemplateManagerProps) {
  const toast = useToast();
  const confirm = useConfirm();

  const [templates, setTemplates] = useState<ChecklistTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Filter & daftar kawasan (site) — checklist dapat berbeda per kawasan.
  const [sites, setSites] = useState<Array<{ id: string; nama: string }>>([]);
  const [siteFilter, setSiteFilter] = useState('');

  // Form tambah/edit inline.
  const [form, setForm] = useState<TemplateFormState | null>(null);
  const [saving, setSaving] = useState(false);

  // Ambil daftar template dari server (urutan ASC dijamin backend).
  const fetchTemplates = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const response = await getChecklistTemplates(undefined, {
        limit: 200,
        site_id: siteFilter || undefined,
      });
      if (response.status === 'success' && response.data) {
        // Urutkan berdasarkan `urutan` agar stabil meski backend mengembalikan
        // urutan berbeda.
        const rows = [...response.data.rows].sort((a, b) => a.urutan - b.urutan);
        setTemplates(rows);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Data template belum dapat dimuat. Coba lagi.');
    } finally {
      setLoading(false);
    }
  }, [siteFilter]);

  // Muat daftar kawasan (site) untuk dropdown filter & form.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await getAllSitesForSelect();
        if (!cancelled && res.status === 'success' && res.data) {
          setSites(res.data.map((s) => ({ id: s.id, nama: s.nama })));
        }
      } catch {
        // Gagal memuat kawasan tidak fatal.
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    void fetchTemplates();
  }, [fetchTemplates]);

  // Buka form untuk menambah item baru.
  const openAddForm = () => {
    setForm({
      id: null,
      nama_item: '',
      deskripsi: '',
      urutan: templates.length > 0 ? templates[templates.length - 1].urutan + 1 : 1,
      wajib: true,
      site_id: siteFilter,
    });
  };

  // Buka form untuk mengedit item yang ada.
  const openEditForm = (item: ChecklistTemplate) => {
    setForm({
      id: item.id,
      nama_item: item.nama_item,
      deskripsi: item.deskripsi,
      urutan: item.urutan,
      wajib: item.wajib,
      site_id: item.site_id || '',
    });
  };

  const closeForm = () => {
    if (saving) return;
    setForm(null);
  };

  const handleSaveForm = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form) return;
    if (!form.nama_item.trim()) {
      toast.error('Nama item wajib diisi.');
      return;
    }

    setSaving(true);
    try {
      const payload = {
        nama_item: form.nama_item.trim(),
        deskripsi: form.deskripsi.trim(),
        urutan: form.urutan,
        wajib: form.wajib,
        aktif: true,
        site_id: form.site_id || null,
      };

      const response = form.id
        ? await updateChecklistTemplate(form.id, payload)
        : await createChecklistTemplate(payload);

      if (response.status === 'success') {
        toast.success(
          form.id
            ? `Item "${form.nama_item}" berhasil diperbarui.`
            : `Item "${form.nama_item}" berhasil ditambahkan.`
        );
        setForm(null);
        void fetchTemplates();
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Item belum dapat disimpan. Periksa isian lalu coba lagi.');
    } finally {
      setSaving(false);
    }
  };

  // Hapus (soft delete) — nonaktifkan item.
  const handleDelete = async (item: ChecklistTemplate) => {
    const ok = await confirm({
      title: 'Nonaktifkan Item?',
      message:
        'Item ini akan dinonaktifkan. Pengecekan yang sudah ada tidak terpengaruh.',
      confirmLabel: 'Nonaktifkan',
    });
    if (!ok) return;

    setActionLoading(`delete-${item.id}`);
    try {
      const response = await deleteChecklistTemplate(item.id);
      if (response.status === 'success') {
        toast.success(`Item "${item.nama_item}" berhasil dinonaktifkan.`);
        void fetchTemplates();
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Item belum dapat dinonaktifkan. Coba lagi.');
    } finally {
      setActionLoading(null);
    }
  };

  // Toggle status wajib / aktif via updateChecklistTemplate.
  const handleToggle = async (item: ChecklistTemplate, field: 'wajib' | 'aktif') => {
    setActionLoading(`toggle-${field}-${item.id}`);
    try {
      const newValue = field === 'wajib' ? !item.wajib : !item.aktif;
      const response = await updateChecklistTemplate(item.id, { [field]: newValue });
      if (response.status === 'success') {
        toast.success(
          field === 'wajib'
            ? `Item "${item.nama_item}" ${newValue ? 'kini wajib' : 'kini tidak wajib'}.`
            : `Item "${item.nama_item}" ${newValue ? 'kini aktif' : 'kini nonaktif'}.`
        );
        void fetchTemplates();
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Status item belum dapat diubah. Coba lagi.');
    } finally {
      setActionLoading(null);
    }
  };

  // Pindahkan urutan item naik/turun lalu simpan ke server.
  const handleMove = async (index: number, direction: -1 | 1) => {
    const target = index + direction;
    if (target < 0 || target >= templates.length) return;

    // Salin daftar lalu tukar posisi item yang dipindah.
    const next = [...templates];
    const [moved] = next.splice(index, 1);
    next.splice(target, 0, moved);

    // Susun ulang nilai `urutan` berurutan dari 1..n.
    const reordered = next.map((item, i) => ({ id: item.id, urutan: i + 1 }));

    setActionLoading(`move-${moved.id}`);
    try {
      const response = await reorderChecklistTemplates(reordered);
      if (response.status === 'success') {
        toast.success('Urutan item berhasil diperbarui.');
        void fetchTemplates();
      }
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : 'Urutan item belum dapat diubah. Coba lagi.');
    } finally {
      setActionLoading(null);
    }
  };

  const isBusy = actionLoading !== null || saving;

  // Bungkus konten yang sama untuk mode "page" dan "modal".
  const content = (
    <>
      {/* Baris aksi / tambah item */}
      <div className="flex flex-col gap-3 border-b border-border px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="font-semibold">Daftar Template Checklist</h2>
          <p className="mt-0.5 text-xs text-muted">
            {templates.length} item · diurutkan berdasarkan nomor urut pengecekan
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {/* Filter kawasan — daftar checklist dapat berbeda per kawasan */}
          <select
            aria-label="Filter kawasan"
            value={siteFilter}
            onChange={(e) => {
              setSiteFilter(e.target.value);
              setForm(null);
            }}
            className="h-11 rounded-xl border border-border bg-card px-3 py-2 text-sm font-semibold text-foreground outline-none transition-colors focus:border-primary focus:ring-2 focus:ring-primary/20"
          >
            <option value="">Semua kawasan</option>
            {sites.map((s) => (
              <option key={s.id} value={s.id}>{s.nama}</option>
            ))}
          </select>
          <button
            type="button"
            onClick={openAddForm}
            disabled={isBusy}
            className="flex min-h-11 items-center justify-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(15,118,74,0.18)] transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-60"
          >
            <Plus className="h-4 w-4" />
            Tambah Item
          </button>
        </div>
      </div>

      {/* Form tambah/edit inline */}
      {form && (
        <form
          onSubmit={handleSaveForm}
          className="space-y-4 border-b border-border bg-background/50 px-5 py-4"
        >
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-bold">
              {form.id ? 'Edit Item Checklist' : 'Tambah Item Checklist'}
            </h3>
            <button
              type="button"
              aria-label="Tutup formulir"
              onClick={closeForm}
              disabled={saving}
              className="rounded-lg p-1.5 text-muted transition-colors hover:bg-background hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
            >
              <X className="h-4 w-4" />
            </button>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="sm:col-span-2">
              <label htmlFor="tmpl-nama" className="mb-1.5 block text-xs font-semibold text-muted">
                Nama Item <span className="text-error">*</span>
              </label>
              <input
                id="tmpl-nama"
                type="text"
                value={form.nama_item}
                onChange={(e) => setForm({ ...form, nama_item: e.target.value })}
                placeholder="Contoh: Pengecekan jaringan"
                className="w-full rounded-xl border border-input-border bg-input px-3.5 py-2.5 text-sm text-foreground outline-none transition-colors focus:border-primary focus:ring-2 focus:ring-primary/20"
              />
            </div>

            <div className="sm:col-span-2">
              <label htmlFor="tmpl-deskripsi" className="mb-1.5 block text-xs font-semibold text-muted">
                Deskripsi
              </label>
              <textarea
                id="tmpl-deskripsi"
                value={form.deskripsi}
                onChange={(e) => setForm({ ...form, deskripsi: e.target.value })}
                rows={2}
                placeholder="Penjelasan singkat mengenai item pengecekan…"
                className="w-full resize-none rounded-xl border border-input-border bg-input px-3.5 py-2.5 text-sm text-foreground outline-none transition-colors focus:border-primary focus:ring-2 focus:ring-primary/20"
              />
            </div>

            <div>
              <label htmlFor="tmpl-urutan" className="mb-1.5 block text-xs font-semibold text-muted">
                Nomor Urut
              </label>
              <input
                id="tmpl-urutan"
                type="number"
                min={1}
                value={form.urutan}
                onChange={(e) => setForm({ ...form, urutan: Number(e.target.value) })}
                className="w-full rounded-xl border border-input-border bg-input px-3.5 py-2.5 text-sm text-foreground outline-none transition-colors focus:border-primary focus:ring-2 focus:ring-primary/20"
              />
            </div>

            <div className="flex items-end">
              <label className="flex cursor-pointer items-center gap-2.5 text-sm font-medium">
                <input
                  type="checkbox"
                  checked={form.wajib}
                  onChange={(e) => setForm({ ...form, wajib: e.target.checked })}
                  className="h-4 w-4 rounded border-input-border text-primary focus:ring-primary"
                />
                Wajib diisi teknisi
              </label>
            </div>

            {/* Kawasan (site) — kosong berarti global (berlaku semua kawasan) */}
            <div className="sm:col-span-2">
              <label htmlFor="tmpl-site" className="mb-1.5 block text-xs font-semibold text-muted">
                Kawasan <span className="font-normal">(opsional — kosong = global)</span>
              </label>
              <select
                id="tmpl-site"
                value={form.site_id}
                onChange={(e) => setForm({ ...form, site_id: e.target.value })}
                className="w-full rounded-xl border border-input-border bg-input px-3.5 py-2.5 text-sm text-foreground outline-none transition-colors focus:border-primary focus:ring-2 focus:ring-primary/20"
              >
                <option value="">Semua kawasan (global)</option>
                {sites.map((s) => (
                  <option key={s.id} value={s.id}>{s.nama}</option>
                ))}
              </select>
            </div>
          </div>

          <div className="flex justify-end gap-2 border-t border-border pt-3">
            <button
              type="button"
              onClick={closeForm}
              disabled={saving}
              className="rounded-lg border border-border px-4 py-2 text-sm font-semibold text-foreground transition-colors hover:bg-background disabled:cursor-not-allowed disabled:opacity-50"
            >
              Batal
            </button>
            <button
              type="submit"
              disabled={saving}
              className="inline-flex items-center gap-2 rounded-lg bg-primary px-5 py-2 text-sm font-semibold text-white transition-colors hover:bg-primary-hover disabled:cursor-not-allowed disabled:opacity-50"
            >
              {saving ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}
              {saving ? 'Menyimpan…' : 'Simpan'}
            </button>
          </div>
        </form>
      )}

      {/* Konten tabel / status */}
      {loading ? (
        <div className="flex items-center justify-center gap-3 py-20" role="status">
          <LoaderCircle className="h-6 w-6 animate-spin text-primary" />
          <span className="text-sm text-muted">Memuat template checklist…</span>
        </div>
      ) : error ? (
        <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
          <AlertTriangle className="h-10 w-10 text-error" />
          <h3 className="mt-3 text-base font-semibold">Data template belum dapat dimuat</h3>
          <p className="mt-1 max-w-md text-xs text-muted">{error}</p>
          <button
            type="button"
            onClick={fetchTemplates}
            className="mt-4 rounded-lg bg-primary px-4 py-2 text-xs font-bold text-white transition-colors hover:bg-primary-hover"
          >
            Coba lagi
          </button>
        </div>
      ) : templates.length === 0 ? (
        <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
          <ListChecks className="h-10 w-10 text-muted/45" />
          <h3 className="mt-3 text-base font-semibold">Belum ada template checklist</h3>
          <p className="mt-1 max-w-sm text-xs text-muted">
            Gunakan &quot;Tambah Item&quot; untuk membuat daftar item pengecekan lapangan.
          </p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[820px] border-collapse text-left text-sm">
            <thead>
              <tr className="border-b border-border bg-background/50 text-xs font-semibold text-muted">
                <th className="px-5 py-3.5">No Urut</th>
                <th className="px-5 py-3.5">Nama Item</th>
                <th className="px-5 py-3.5">Deskripsi</th>
                <th className="px-5 py-3.5">Wajib</th>
                <th className="px-5 py-3.5">Aktif</th>
                <th className="px-5 py-3.5 text-right">Aksi</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {templates.map((item, index) => {
                const isFirst = index === 0;
                const isLast = index === templates.length - 1;
                return (
                  <tr key={item.id} className="align-top transition-colors hover:bg-background/40">
                    <td className="px-5 py-4">
                      <div className="flex items-center gap-1">
                        <span className="font-mono text-xs font-semibold text-primary">{item.urutan}</span>
                        <div className="flex flex-col">
                          <button
                            type="button"
                            onClick={() => handleMove(index, -1)}
                            disabled={isFirst || isBusy}
                            title="Naikkan urutan"
                            aria-label={`Naikkan urutan ${item.nama_item}`}
                            className="rounded p-0.5 text-muted transition-colors hover:text-foreground disabled:cursor-not-allowed disabled:opacity-30"
                          >
                            <ArrowUp className="h-3.5 w-3.5" />
                          </button>
                          <button
                            type="button"
                            onClick={() => handleMove(index, 1)}
                            disabled={isLast || isBusy}
                            title="Turunkan urutan"
                            aria-label={`Turunkan urutan ${item.nama_item}`}
                            className="rounded p-0.5 text-muted transition-colors hover:text-foreground disabled:cursor-not-allowed disabled:opacity-30"
                          >
                            <ArrowDown className="h-3.5 w-3.5" />
                          </button>
                        </div>
                      </div>
                    </td>
                    <td className="px-5 py-4">
                      <p className="font-semibold">{item.nama_item}</p>
                    </td>
                    <td className="max-w-xs px-5 py-4">
                      <p className="line-clamp-2 text-xs text-muted" title={item.deskripsi}>
                        {item.deskripsi || '-'}
                      </p>
                    </td>
                    <td className="px-5 py-4">
                      <button
                        type="button"
                        role="switch"
                        aria-checked={item.wajib}
                        aria-label={`Toggl status wajib ${item.nama_item}`}
                        onClick={() => handleToggle(item, 'wajib')}
                        disabled={actionLoading === `toggle-wajib-${item.id}`}
                        className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                          item.wajib ? 'bg-primary' : 'bg-input-border'
                        } disabled:cursor-wait disabled:opacity-60`}
                      >
                        <span
                          className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                            item.wajib ? 'translate-x-6' : 'translate-x-1'
                          }`}
                        />
                      </button>
                    </td>
                    <td className="px-5 py-4">
                      <button
                        type="button"
                        role="switch"
                        aria-checked={item.aktif}
                        aria-label={`Toggl status aktif ${item.nama_item}`}
                        onClick={() => handleToggle(item, 'aktif')}
                        disabled={actionLoading === `toggle-aktif-${item.id}`}
                        className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                          item.aktif ? 'bg-emerald-500' : 'bg-input-border'
                        } disabled:cursor-wait disabled:opacity-60`}
                      >
                        <span
                          className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                            item.aktif ? 'translate-x-6' : 'translate-x-1'
                          }`}
                        />
                      </button>
                    </td>
                    <td className="px-5 py-4 text-right">
                      <div className="flex items-center justify-end gap-1">
                        <button
                          type="button"
                          onClick={() => openEditForm(item)}
                          disabled={isBusy}
                          title="Ubah item"
                          aria-label={`Ubah item ${item.nama_item}`}
                          className="rounded-lg p-2 text-muted transition-colors hover:bg-primary/10 hover:text-primary disabled:cursor-not-allowed disabled:opacity-40"
                        >
                          <Edit2 className="h-4 w-4" />
                        </button>
                        <button
                          type="button"
                          onClick={() => handleDelete(item)}
                          disabled={isBusy}
                          title="Nonaktifkan item"
                          aria-label={`Nonaktifkan item ${item.nama_item}`}
                          className="rounded-lg p-2 text-muted transition-colors hover:bg-error/10 hover:text-error disabled:cursor-not-allowed disabled:opacity-40"
                        >
                          {actionLoading === `delete-${item.id}` ? (
                            <LoaderCircle className="h-4 w-4 animate-spin" />
                          ) : (
                            <Trash2 className="h-4 w-4" />
                          )}
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </>
  );

  // Mode "page": tampil sebagai halaman utuh dengan heading sendiri.
  if (mode === 'page') {
    return (
      <section
        aria-labelledby="checklist-template-heading"
        aria-busy={loading}
        className="overflow-hidden rounded-2xl border border-border bg-card"
      >
        <div className="border-b border-border px-5 py-4">
          <h1 id="checklist-template-heading" className="text-lg font-bold">
            Template Checklist Pengecekan Lapangan
          </h1>
          <p className="mt-0.5 text-xs text-muted">
            Kelola item yang wajib diperiksa teknisi saat pengecekan lapangan. Perubahan berlaku untuk pengecekan baru.
          </p>
        </div>
        {content}
      </section>
    );
  }

  // Mode "modal": tampil di dalam panel modal dengan tombol tutup.
  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fadeIn">
      <div className="relative flex max-h-[92vh] w-full max-w-4xl flex-col overflow-hidden rounded-2xl border border-border bg-card shadow-2xl animate-popIn">
        <div className="flex items-start justify-between gap-3 border-b border-border px-5 py-4">
          <div>
            <h3 className="text-base font-bold">Kelola Template Checklist</h3>
            <p className="mt-0.5 text-xs text-muted">
              Atur daftar item pengecekan lapangan yang muncul di wizard.
            </p>
          </div>
          <button
            type="button"
            aria-label="Tutup pengelolaan template"
            onClick={onClose}
            disabled={isBusy}
            className="rounded-lg border border-border p-1 text-muted transition-colors hover:bg-background hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto">{content}</div>
      </div>
    </div>
  );
}

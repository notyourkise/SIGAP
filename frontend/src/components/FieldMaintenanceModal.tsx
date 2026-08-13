'use client';

import React, { useEffect, useRef, useState } from 'react';
import {
  X,
  ChevronLeft,
  ChevronRight,
  Check,
  AlertTriangle,
  Camera,
  LoaderCircle,
  Wrench,
  ImagePlus,
  FileCheck2,
  MapPin,
} from 'lucide-react';
import { apiRequest, getChecklistTemplates } from '@/lib/api';
import { useToast } from '@/components/Toast';
import AuthenticatedImage from '@/components/AuthenticatedImage';

interface CCTVInfo {
  id: string;
  kode_cctv: string;
  nama: string;
  lokasi_area?: string;
}

export type FieldCheckResult = 'ok' | 'not_ok';

export interface FieldCheckAnswer {
  result: FieldCheckResult | null;
  note: string;
}

interface FieldMaintenanceModalProps {
  cctv: CCTVInfo | null;
  open: boolean;
  onClose: () => void;
  onSaved: () => void;
  // mode "field" (default): catatan pengecekan spontan dari halaman Field Maintenance.
  // mode "complete": penyelesaian tugas terjadwal — hasil checklist dikirim ke
  // POST /maintenance-logs/:id/complete dan status tugas ditentukan sistem (9/9 ok -> completed).
  mode?: 'field' | 'complete';
  maintenanceLogId?: string;
}

// Daftar item checklist pengecekan lapangan (wajib diisi satu per satu oleh teknisi)
export const FIELD_CHECKLIST: { name: string; desc: string }[] = [
  {
    name: 'Pengecekan jaringan',
    desc: 'Pastikan koneksi jaringan kamera ke NVR (kabel LAN / FO / WiFi) normal, tidak ada lampu indikator anomali.',
  },
  {
    name: 'Pengecekan pasokan listrik',
    desc: 'Periksa adaptor / PoE / panel surya serta sumber listrik; pastikan tegangan stabil dan tidak ada korsleting.',
  },
  {
    name: 'Pembersihan lensa',
    desc: 'Bersihkan lensa dari debu, kotoran, atau sarang laba-laba yang mengganggu pandangan kamera.',
  },
  {
    name: 'Pemeriksaan fisik kamera',
    desc: 'Periksa kondisi housing, bracket, dan segel kedap air; pastikan pemasangan kuat dan tidak berkarat.',
  },
  {
    name: 'Pengecekan kualitas gambar',
    desc: 'Pastikan fokus, kecerahan, dan warna gambar baik; tidak blur, gelap, atau silau.',
  },
  {
    name: 'Pengujian perekaman',
    desc: 'Pastikan rekaman tersimpan normal di NVR / penyimpanan dan tidak ada rekaman terputus.',
  },
  {
    name: 'Pengecekan akses jarak jauh',
    desc: 'Pastikan kamera dapat diakses remote view dari pusat tanpa kendala.',
  },
  {
    name: 'Pemeriksaan kabel & konektor',
    desc: 'Periksa kondisi kabel, konektor, dan jalur instalasi; pastikan tidak terkelupas atau longgar.',
  },
  {
    name: 'Pengecekan lingkungan sekitar',
    desc: 'Pastikan area sekitar kamera bersih dan tidak ada halangan pandang (ranting, bangunan baru, dll).',
  },
];

const MAX_IMAGES = 5;

export default function FieldMaintenanceModal({
  cctv,
  open,
  onClose,
  onSaved,
  mode = 'field',
  maintenanceLogId,
}: FieldMaintenanceModalProps) {
  const toast = useToast();
  const isCompleteMode = mode === 'complete';

  const [step, setStep] = useState(0);
  // Daftar item checklist aktif yang diambil dari server (template checklist).
  // Kosong saat belum terisi / gagal dimuat — pada kondisi itu kita jatuh ke
  // fallback FIELD_CHECKLIST.
  const [templates, setTemplates] = useState<{ name: string; desc: string }[]>([]);
  const [loadingTemplates, setLoadingTemplates] = useState(false);
  const [answers, setAnswers] = useState<FieldCheckAnswer[]>(
    FIELD_CHECKLIST.map(() => ({ result: null, note: '' }))
  );
  const [images, setImages] = useState<string[]>([]);
  const [uploading, setUploading] = useState(false);
  const [notes, setNotes] = useState('');
  const [submitting, setSubmitting] = useState(false);
  // Koordinat geolokasi saat modal dibuka (hanya untuk mode "field").
  // Opsional: bila pengguna menolak izin / timeout, nilainya tetap null dan
  // wizard tetap berjalan tanpa koordinat.
  const [coords, setCoords] = useState<{ lat: string; lng: string } | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Daftar aktif: pakai template dari server bila tersedia, selain itu fallback.
  const activeList = templates.length > 0 ? templates : FIELD_CHECKLIST;

  const totalSteps = activeList.length + 1; // item checklist + tahap foto bukti
  const isPhotoStep = step === activeList.length;
  const currentAnswer = isPhotoStep ? null : answers[step];

  // Saat modal dibuka, muat template checklist aktif dari server. Template ini
  // menentukan jumlah langkah wizard. Bila gagal/kosong, kita memakai
  // FIELD_CHECKLIST sebagai fallback.
  useEffect(() => {
    if (!open) return;

    let cancelled = false;
    const loadTemplates = async () => {
      setLoadingTemplates(true);
      try {
        // Muat template sesuai kawasan CCTV (bila cctv.site_id tersedia) —
        // backend mengembalikan template milik kawasan + template global.
        const response = await getChecklistTemplates(true, {
          limit: 100,
          site_id: (cctv as { site_id?: string | null })?.site_id || undefined,
        });
        if (cancelled) return;
        if (response.status === 'success' && response.data) {
          // Backend sudah menyaring aktif; urutkan tetap berdasarkan `urutan`.
          const rows = [...response.data.rows].sort((a, b) => a.urutan - b.urutan);
          setTemplates(rows.map((r) => ({ name: r.nama_item, desc: r.deskripsi })));
        } else {
          setTemplates([]);
        }
      } catch {
        // Gagal memuat template — gunakan fallback FIELD_CHECKLIST.
        if (!cancelled) setTemplates([]);
      } finally {
        if (!cancelled) setLoadingTemplates(false);
      }
    };
    void loadTemplates();

    return () => {
      cancelled = true;
    };
  }, [open, cctv]);

  // Sinkronkan jumlah jawaban dengan daftar aktif. Karena template dimuat
  // secara asinkron, jumlah langkah bisa berubah setelah modal dibuka — pastikan
  // `answers` selalu sesuai panjang `activeList` agar index tidak melebihi batas.
  useEffect(() => {
    if (!open) return;
    setAnswers(activeList.map(() => ({ result: null, note: '' })));
    // activeList referensinya stabil (state templates atau konstanta FIELD_CHECKLIST).
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, activeList]);

  useEffect(() => {
    if (!open) return;

    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', closeOnEscape);
    return () => document.removeEventListener('keydown', closeOnEscape);
  }, [open, onClose]);

  // Tangkap koordinat geolokasi saat wizard dibuka — hanya untuk mode "field"
  // (pengecekan lapangan spontan). Mode "complete" tidak memerlukan koordinat.
  useEffect(() => {
    if (!open || isCompleteMode) return;

    // Guard SSR-safe: pastikan geolokasi tersedia di browser.
    if (!('geolocation' in navigator)) return;

    try {
      navigator.geolocation.getCurrentPosition(
        (position) => {
          // Simpan koordinat sebagai string dengan presisi wajar (±6 desimal ~ ±11 cm).
          setCoords({
            lat: position.coords.latitude.toFixed(6),
            lng: position.coords.longitude.toFixed(6),
          });
        },
        () => {
          // Pengguna menolak izin / terjadi error — biarkan coords null (opsional).
          setCoords(null);
        },
        {
          enableHighAccuracy: true,
          timeout: 10000,
          maximumAge: 60000,
        }
      );
    } catch {
      // Kegagalan tak terduga — jangan sampai merusak wizard.
      setCoords(null);
    }
  }, [open, isCompleteMode]);

  const handleClose = () => {
    if (submitting || uploading) return;
    resetState();
    onClose();
  };

  const resetState = () => {
    setStep(0);
    setAnswers(activeList.map(() => ({ result: null, note: '' })));
    setImages([]);
    setNotes('');
    setSubmitting(false);
    setUploading(false);
    setCoords(null);
    if (fileInputRef.current) fileInputRef.current.value = '';
  };

  const setResult = (result: FieldCheckResult) => {
    setAnswers((prev) =>
      prev.map((a, i) => (i === step ? { ...a, result } : a))
    );
  };

  const setNote = (note: string) => {
    setAnswers((prev) =>
      prev.map((a, i) => (i === step ? { ...a, note } : a))
    );
  };

  const goNext = () => {
    if (isPhotoStep) return;
    if (!currentAnswer?.result) return;
    setStep((s) => Math.min(s + 1, totalSteps - 1));
  };

  const goBack = () => {
    if (step === 0) {
      handleClose();
      return;
    }
    setStep((s) => Math.max(s - 1, 0));
  };

  // Unggah foto bukti ke server (endpoint /api/upload), lalu simpan URL-nya
  const handleFileSelect = async (files: FileList | null) => {
    if (!files || files.length === 0) return;

    const remaining = MAX_IMAGES - images.length;
    const toUpload = Array.from(files).slice(0, remaining);
    if (toUpload.length === 0) {
      toast.error(`Maksimal ${MAX_IMAGES} foto bukti dapat dilampirkan.`);
      return;
    }

    setUploading(true);
    try {
      for (const file of toUpload) {
        const dataPayload = new FormData();
        dataPayload.append('file', file);

        const response = await apiRequest<{ url: string }>('/upload', {
          method: 'POST',
          body: dataPayload,
        });

        if (response.status === 'success' && response.data?.url) {
          setImages((prev) => (prev.includes(response.data!.url) ? prev : [...prev, response.data!.url]));
        }
      }
    } catch (err: any) {
      toast.error(err.message || 'Foto bukti belum dapat diunggah. Coba lagi.');
    } finally {
      setUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = '';
    }
  };

  const handleSubmit = async () => {
    if (!cctv) return;
    if (images.length === 0) {
      toast.error('Minimal satu foto bukti wajib dilampirkan.');
      return;
    }
    if (isCompleteMode && !maintenanceLogId) {
      toast.error('Data tugas tidak ditemukan. Muat ulang halaman lalu coba lagi.');
      return;
    }

    setSubmitting(true);
    try {
      const body = isCompleteMode
        ? {
            notes,
            checks: answers.map((a, i) => ({
              name: activeList[i]?.name ?? '',
              result: a.result,
              note: a.note,
            })),
            images: images.map((url) => ({ image_url: url })),
          }
        : {
            cctv_id: cctv.id,
            notes,
            checks: answers.map((a, i) => ({
              name: activeList[i]?.name ?? '',
              result: a.result,
              note: a.note,
            })),
            images: images.map((url) => ({ image_url: url })),
            // Kirim koordinat geolokasi hanya bila berhasil tercatat (opsional).
            ...(coords ? { check_latitude: coords.lat, check_longitude: coords.lng } : {}),
          };

      const response = await apiRequest(
        isCompleteMode ? `/maintenance-logs/${maintenanceLogId}/complete` : '/field-maintenance',
        {
          method: 'POST',
          body: JSON.stringify(body),
        }
      );

      if (response.status === 'success') {
        toast.success(
          isCompleteMode
            ? `Tugas pemeliharaan CCTV "${cctv.nama}" berhasil diselesaikan.`
            : `Pengecekan lapangan CCTV "${cctv.nama}" berhasil disimpan.`
        );
        resetState();
        onSaved();
        onClose();
      }
    } catch (err: any) {
      toast.error(
        err.message ||
          (isCompleteMode
            ? 'Tugas belum dapat diselesaikan. Periksa isian lalu coba lagi.'
            : 'Pengecekan lapangan belum dapat disimpan. Periksa isian lalu coba lagi.')
      );
    } finally {
      setSubmitting(false);
    }
  };

  if (!open || !cctv) return null;

  const answeredCount = answers.filter((a) => a.result !== null).length;

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fadeIn">
      <div className="relative flex max-h-[92vh] w-full max-w-lg flex-col overflow-hidden rounded-2xl border border-border bg-card shadow-2xl animate-popIn">
        {/* Header */}
        <div className="flex items-start justify-between gap-3 border-b border-border px-5 py-4">
          <div className="min-w-0">
            <h3 className="text-base font-bold">
              {isCompleteMode ? 'Selesaikan Tugas Pemeliharaan' : 'Pengecekan Lapangan'}
            </h3>
            <p className="mt-0.5 truncate text-xs text-muted">
              {cctv.kode_cctv} · {cctv.nama}
            </p>
          </div>
          <button
            type="button"
            aria-label={isCompleteMode ? 'Tutup penyelesaian tugas' : 'Tutup pengecekan lapangan'}
            onClick={handleClose}
            disabled={submitting || uploading}
            className="rounded-lg border border-border p-1 text-muted transition-colors hover:bg-background hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Progress */}
        <div className="border-b border-border px-5 py-3">
          <div className="flex items-center justify-between text-[11px] font-semibold text-muted">
            <span className="flex items-center gap-1.5">
              <Wrench className="h-3.5 w-3.5 text-primary" />
              {isPhotoStep ? 'Tahap akhir: Bukti foto' : `Item ${step + 1} dari ${activeList.length}`}
            </span>
            <span>
              {isPhotoStep ? answeredCount + 1 : answeredCount}/{totalSteps}
            </span>
          </div>
          <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-background">
            <div
              className="h-full rounded-full bg-primary transition-all duration-300"
              style={{ width: `${((isPhotoStep ? step + 1 : step) / totalSteps) * 100}%` }}
            />
          </div>
          {/* Indikator saat daftar item template masih dimuat dari server */}
          {loadingTemplates && templates.length === 0 && (
            <p className="mt-1.5 flex items-center gap-1.5 text-[11px] text-muted">
              <LoaderCircle className="h-3 w-3 animate-spin" />
              Memuat template checklist…
            </p>
          )}
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-5 py-4">
          {!isPhotoStep ? (
            <div className="space-y-4">
              <div className="space-y-1.5">
                <h4 className="text-sm font-bold">
                  {step + 1}. {activeList[step]?.name}
                </h4>
                <p className="text-xs leading-relaxed text-muted">{activeList[step]?.desc}</p>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <button
                  type="button"
                  onClick={() => setResult('ok')}
                  className={`flex flex-col items-center gap-2 rounded-xl border-2 px-4 py-4 text-sm font-bold transition-all ${
                    currentAnswer?.result === 'ok'
                      ? 'border-emerald-500 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 shadow-[0_4px_14px_rgba(16,185,129,0.15)]'
                      : 'border-border bg-background text-muted hover:border-emerald-500/40 hover:text-foreground'
                  }`}
                >
                  <span className={`flex h-9 w-9 items-center justify-center rounded-full border-2 ${
                    currentAnswer?.result === 'ok' ? 'border-emerald-500 bg-emerald-500 text-white' : 'border-border'
                  }`}>
                    <Check className="h-5 w-5" />
                  </span>
                  Normal / OK
                </button>
                <button
                  type="button"
                  onClick={() => setResult('not_ok')}
                  className={`flex flex-col items-center gap-2 rounded-xl border-2 px-4 py-4 text-sm font-bold transition-all ${
                    currentAnswer?.result === 'not_ok'
                      ? 'border-red-500 bg-red-500/10 text-red-600 dark:text-red-400 shadow-[0_4px_14px_rgba(239,68,68,0.15)]'
                      : 'border-border bg-background text-muted hover:border-red-500/40 hover:text-foreground'
                  }`}
                >
                  <span className={`flex h-9 w-9 items-center justify-center rounded-full border-2 ${
                    currentAnswer?.result === 'not_ok' ? 'border-red-500 bg-red-500 text-white' : 'border-border'
                  }`}>
                    <AlertTriangle className="h-5 w-5" />
                  </span>
                  Bermasalah
                </button>
              </div>

              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-muted">
                  Catatan {currentAnswer?.result === 'not_ok' ? '(wajib bila bermasalah)' : '(opsional)'}
                </label>
                <textarea
                  value={currentAnswer?.note || ''}
                  onChange={(e) => setNote(e.target.value)}
                  placeholder="Contoh: kabel konektor longgar, lensa berdebu…"
                  rows={3}
                  className="w-full resize-none rounded-lg border border-input-border bg-input px-3 py-2 text-sm text-foreground outline-none transition-colors focus:border-primary focus:ring-1 focus:ring-primary"
                />
                {currentAnswer?.result === 'not_ok' && !currentAnswer?.note.trim() && (
                  <p className="text-[11px] text-red-500">
                    Tambahkan catatan singkat mengenai masalah yang ditemukan.
                  </p>
                )}
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              <div className="space-y-1.5">
                <h4 className="text-sm font-bold">Bukti Foto</h4>
                <p className="text-xs leading-relaxed text-muted">
                  Lampirkan minimal 1 foto kondisi kamera dan lingkungan sekitar sebagai bukti.
                  Foto ini merupakan tahapan akhir untuk menyelesaikan pengecekan.
                </p>
              </div>

              {!isCompleteMode && (
                <div className="flex items-center gap-2 rounded-lg border border-border bg-background px-3 py-2">
                  <MapPin
                    className={`h-4 w-4 ${coords ? 'text-emerald-600 dark:text-emerald-400' : 'text-muted'}`}
                  />
                  <span className={`text-xs ${coords ? 'font-semibold text-emerald-700 dark:text-emerald-400' : 'text-muted'}`}>
                    {coords
                      ? 'Lokasi tercatat'
                      : 'Lokasi belum tercatat (opsional)'}
                  </span>
                </div>
              )}

              <div className="grid grid-cols-3 gap-2">
                {images.map((url, i) => (
                  <div key={url} className="group relative aspect-square overflow-hidden rounded-lg border border-border">
                    <AuthenticatedImage imageUrl={url} alt={`Bukti foto ${i + 1}`} className="h-full w-full object-cover" />
                    <button
                      type="button"
                      aria-label={`Hapus foto ${i + 1}`}
                      onClick={() => setImages((prev) => prev.filter((u) => u !== url))}
                      disabled={submitting}
                      className="absolute right-1 top-1 flex h-6 w-6 items-center justify-center rounded-full bg-black/60 text-white opacity-0 transition-opacity group-hover:opacity-100 disabled:opacity-30"
                    >
                      <X className="h-3.5 w-3.5" />
                    </button>
                  </div>
                ))}

                {images.length < MAX_IMAGES && (
                  <button
                    type="button"
                    onClick={() => fileInputRef.current?.click()}
                    disabled={uploading || submitting}
                    className="flex aspect-square flex-col items-center justify-center gap-1.5 rounded-lg border-2 border-dashed border-input-border bg-input text-muted transition-colors hover:border-primary/50 hover:text-primary disabled:cursor-wait disabled:opacity-60"
                  >
                    {uploading ? (
                      <>
                        <LoaderCircle className="h-5 w-5 animate-spin" />
                        <span className="text-[10px] font-semibold">Mengunggah…</span>
                      </>
                    ) : (
                      <>
                        <ImagePlus className="h-5 w-5" />
                        <span className="px-1 text-center text-[10px] font-semibold">
                          Tambah foto ({images.length}/{MAX_IMAGES})
                        </span>
                      </>
                    )}
                  </button>
                )}
              </div>
              <input
                ref={fileInputRef}
                type="file"
                accept="image/jpeg,image/png,image/webp"
                multiple
                className="hidden"
                onChange={(e) => handleFileSelect(e.target.files)}
              />

              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-muted">Catatan umum (opsional)</label>
                <textarea
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  placeholder="Ringkasan hasil pengecekan, temuan lain, atau tindak lanjut yang diperlukan…"
                  rows={3}
                  className="w-full resize-none rounded-lg border border-input-border bg-input px-3 py-2 text-sm text-foreground outline-none transition-colors focus:border-primary focus:ring-1 focus:ring-primary"
                />
              </div>

              <div className="rounded-lg border border-border bg-background p-3">
                <p className="text-[11px] font-semibold text-muted">Ringkasan hasil checklist:</p>
                <div className="mt-1.5 flex items-center gap-2 text-[11px] font-bold">
                  <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-emerald-600 dark:text-emerald-400">
                    {answers.filter((a) => a.result === 'ok').length} Normal
                  </span>
                  <span className="rounded-full bg-red-500/10 px-2 py-0.5 text-red-600 dark:text-red-400">
                    {answers.filter((a) => a.result === 'not_ok').length} Bermasalah
                  </span>
                  {answers.some((a) => a.result === null) && (
                    <span className="rounded-full bg-amber-500/10 px-2 py-0.5 text-amber-600 dark:text-amber-400">
                      {answers.filter((a) => a.result === null).length} Belum diisi
                    </span>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between gap-2 border-t border-border px-5 py-4">
          <button
            type="button"
            onClick={goBack}
            disabled={submitting || uploading}
            className="inline-flex items-center gap-1.5 rounded-lg border border-border bg-card px-4 py-2 text-sm font-semibold text-foreground transition-colors hover:bg-background disabled:cursor-not-allowed disabled:opacity-50"
          >
            <ChevronLeft className="h-4 w-4" />
            {step === 0 ? 'Batal' : 'Kembali'}
          </button>

          {!isPhotoStep ? (
            <button
              type="button"
              onClick={goNext}
              disabled={!currentAnswer?.result}
              className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-5 py-2 text-sm font-semibold text-white transition-colors hover:bg-primary-hover disabled:cursor-not-allowed disabled:opacity-50"
            >
              Lanjut
              <ChevronRight className="h-4 w-4" />
            </button>
          ) : (
            <button
              type="button"
              onClick={handleSubmit}
              disabled={submitting || uploading || images.length === 0}
              className="inline-flex items-center gap-2 rounded-lg bg-primary px-5 py-2 text-sm font-semibold text-white shadow-[0_6px_16px_rgba(70,114,53,0.2)] transition-colors hover:bg-primary-hover disabled:cursor-not-allowed disabled:opacity-50"
            >
              {submitting ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <FileCheck2 className="h-4 w-4" />}
              {submitting ? 'Menyimpan…' : isCompleteMode ? 'Selesaikan Tugas' : 'Selesaikan Pengecekan'}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

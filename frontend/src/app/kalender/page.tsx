"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  AlertTriangle,
  Calendar,
  CalendarDays,
  ChevronDown,
  LoaderCircle,
  Plus,
  User as UserIcon,
  Wrench,
  X,
} from "lucide-react";
import DashboardLayout from "@/components/DashboardLayout";
import CalendarGrid, {
  CalendarTask,
  STATUS_META,
  formatLongDate,
  toDayKey,
} from "@/components/CalendarGrid";
import { useAuth } from "@/context/AuthContext";
import { apiRequest } from "@/lib/api";
import { useToast } from "@/components/Toast";

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
  technician: User | null;
  action_taken: string;
  notes: string;
  status: "scheduled" | "ongoing" | "completed" | "failed";
  scheduled_date: string | null;
  completed_at: string | null;
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

const PAGE_SIZE = 100;

// Ubah "YYYY-MM-DD" atau timestamp menjadi kunci hari "YYYY-MM-DD".
function toDayKeyOfLog(value: string | null): string | null {
  if (!value) return null;
  return value.slice(0, 10);
}

// Konversi objek Date menjadi nilai input datetime-local "YYYY-MM-DDTHH:mm".
function toDatetimeLocal(date: Date): string {
  const pad = (value: number) => String(value).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(
    date.getHours()
  )}:${pad(date.getMinutes())}`;
}

export default function CalendarPage() {
  const { user, loading: authLoading } = useAuth();
  const isAdmin = user?.role === "admin";
  const router = useRouter();

  // Bulan yang sedang ditampilkan (tanggal 1).
  const now = new Date();
  const [month, setMonth] = useState(() => new Date(now.getFullYear(), now.getMonth(), 1));
  const [selectedDay, setSelectedDay] = useState<string | null>(null);

  const [logs, setLogs] = useState<MaintenanceLog[]>([]);
  const [technicians, setTechnicians] = useState<User[]>([]);
  const [technicianFilter, setTechnicianFilter] = useState("all"); // admin only
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // Modal "Buat Tugas Pemeliharaan".
  const toast = useToast();
  const [createOpen, setCreateOpen] = useState(false);
  const [createDate, setCreateDate] = useState<Date | null>(null);
  const [cctvs, setCctvs] = useState<MasterCCTV[]>([]);
  const [cctvLoading, setCctvLoading] = useState(false);
  const [createCctvId, setCreateCctvId] = useState("");
  const [createAction, setCreateAction] = useState("");
  const [createNotes, setCreateNotes] = useState("");
  const [createScheduledDate, setCreateScheduledDate] = useState("");
  const [createSubmitting, setCreateSubmitting] = useState(false);
  const [createFormError, setCreateFormError] = useState("");

  // Arahkan non-admin ke halaman lain bila perlu (kalender terbuka untuk admin & teknisi).
  useEffect(() => {
    if (!authLoading && !user) router.replace("/login");
  }, [authLoading, router, user]);

  // Muat daftar teknisi untuk dropdown filter (khusus admin).
  const loadTechnicians = useCallback(async () => {
    if (!isAdmin) return;
    try {
      const response = await apiRequest<{ rows?: User[] }>(
        "/users?page=1&limit=100&role=teknisi"
      );
      setTechnicians(response.data?.rows ?? []);
    } catch {
      setTechnicians([]);
    }
  }, [isAdmin]);

  useEffect(() => {
    if (isAdmin) void loadTechnicians();
  }, [isAdmin, loadTechnicians]);

  // Rentang bulan yang sedang ditampilkan.
  const range = useMemo(() => {
    const start = new Date(month.getFullYear(), month.getMonth(), 1);
    const end = new Date(month.getFullYear(), month.getMonth() + 1, 0);
    return {
      startKey: toDayKey(start),
      endKey: toDayKey(end),
    };
  }, [month]);

  // Ambil log pemeliharaan untuk rentang bulan ini.
  const loadLogs = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const response = await apiRequest<MaintenanceLogsResponse>(
        `/maintenance-logs?start_date=${range.startKey}&end_date=${range.endKey}&page=1&limit=${PAGE_SIZE}`
      );
      setLogs(response.data?.rows ?? []);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Kalender belum dapat dimuat. Coba lagi.");
    } finally {
      setLoading(false);
    }
  }, [range.startKey, range.endKey]);

  useEffect(() => {
    const timer = setTimeout(() => void loadLogs(), 0);
    return () => clearTimeout(timer);
  }, [loadLogs]);

  // Kelompokkan log per hari berdasarkan scheduled_date (fallback created_at).
  const logsByDay = useMemo(() => {
    const map: Record<string, CalendarTask[]> = {};
    for (const log of logs) {
      if (technicianFilter !== "all" && log.technician_id !== technicianFilter) continue;
      const dayKey = toDayKeyOfLog(log.scheduled_date) ?? toDayKeyOfLog(log.created_at);
      if (!dayKey) continue;
      if (!map[dayKey]) map[dayKey] = [];
      map[dayKey].push({
        id: log.id,
        status: log.status,
        cctvName: log.cctv?.nama ?? "CCTV tanpa nama",
        cctvCode: log.cctv?.kode_cctv ?? "-",
        technicianName: log.technician?.full_name ?? "Belum ditugaskan",
        actionTaken: log.action_taken,
      });
    }
    return map;
  }, [logs, technicianFilter]);

  const selectedTasks = selectedDay ? (logsByDay[selectedDay] ?? []) : [];

  const prevMonth = () =>
    setMonth((m) => new Date(m.getFullYear(), m.getMonth() - 1, 1));
  const nextMonth = () =>
    setMonth((m) => new Date(m.getFullYear(), m.getMonth() + 1, 1));
  const goToday = () => setMonth(new Date(now.getFullYear(), now.getMonth(), 1));

  // Buka modal "Buat Tugas Pemeliharaan" dengan tanggal hari yang diklik.
  const handleDayClick = (date: Date) => {
    setCreateDate(date);
    setCreateScheduledDate(toDatetimeLocal(date));
    setCreateFormError("");
    setCreateOpen(true);
  };

  // Muat daftar CCTV untuk dropdown form create (dipanggil saat modal dibuka).
  const fetchCctvs = useCallback(async () => {
    setCctvLoading(true);
    try {
      const response = await apiRequest<{ rows?: MasterCCTV[] }>(
        "/cctv?page=1&limit=100"
      );
      setCctvs(response.data?.rows ?? []);
    } catch (err: unknown) {
      setCctvs([]);
      setCreateFormError(
        err instanceof Error
          ? err.message
          : "Daftar CCTV belum dapat dimuat. Muat ulang sebelum membuat tugas."
      );
    } finally {
      setCctvLoading(false);
    }
  }, []);

  // Muat daftar CCTV saat modal dibuka.
  useEffect(() => {
    if (createOpen) void fetchCctvs();
  }, [createOpen, fetchCctvs]);

  // Tutup modal: tombol X, overlay, atau tombol Escape.
  const closeCreateModal = () => {
    if (createSubmitting) return;
    setCreateOpen(false);
    setCreateDate(null);
    setCreateFormError("");
  };

  useEffect(() => {
    if (!createOpen) return;
    const previousOverflow = document.body.style.overflow;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape" || createSubmitting) return;
      setCreateOpen(false);
      setCreateDate(null);
      setCreateFormError("");
    };
    document.body.style.overflow = "hidden";
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [createOpen, createSubmitting]);

  // Simpan tugas pemeliharaan baru.
  const handleSubmitCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!createCctvId) {
      setCreateFormError("Pilih CCTV yang akan dipelihara.");
      return;
    }
    if (createAction.trim().length < 5) {
      setCreateFormError("Tindakan yang dilakukan minimal 5 karakter.");
      return;
    }

    setCreateSubmitting(true);
    setCreateFormError("");
    try {
      // Konversi datetime-local "YYYY-MM-DDTHH:mm" → "YYYY-MM-DD HH:MM:SS".
      const scheduledDate =
        createScheduledDate.length > 0
          ? `${createScheduledDate}:00`.replace("T", " ")
          : undefined;

      const response = await apiRequest("/maintenance-logs", {
        method: "POST",
        body: JSON.stringify({
          cctv_id: createCctvId,
          action_taken: createAction.trim(),
          notes: createNotes.trim() || undefined,
          scheduled_date: scheduledDate,
        }),
      });

      if (response.status === "success") {
        closeCreateModal();
        toast.success("Tugas pemeliharaan berhasil dibuat.");
        await loadLogs();
      }
    } catch (err: unknown) {
      setCreateFormError(
        err instanceof Error
          ? err.message
          : "Tugas belum dapat dibuat. Periksa isian lalu coba lagi."
      );
    } finally {
      setCreateSubmitting(false);
    }
  };

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <div className="flex items-center gap-2 text-xs font-bold uppercase tracking-[0.08em] text-primary">
              <CalendarDays className="h-4 w-4" />
              <span>Pos komando / jadwal</span>
            </div>
            <h1 className="mt-2 text-3xl font-bold tracking-[-0.025em]">Kalender Jadwal Pemeliharaan</h1>
            <p className="mt-1 max-w-2xl text-sm text-muted">
              Lihat jadwal pemeliharaan CCTV per hari. Klik hari untuk melihat detail tugas.
            </p>
          </div>

          {/* Filter per teknisi (admin saja; teknisi melihat data miliknya) */}
          {isAdmin && (
            <label className="flex items-center gap-2 rounded-xl border border-border bg-card px-3 py-2 text-sm">
              <UserIcon className="h-4 w-4 text-primary" />
              <span className="text-xs font-semibold text-muted">Teknisi</span>
              <span className="relative">
                <select
                  value={technicianFilter}
                  onChange={(e) => {
                    setTechnicianFilter(e.target.value);
                    setSelectedDay(null);
                  }}
                  className="appearance-none rounded-lg border border-border bg-background py-1.5 pl-3 pr-8 text-sm font-medium text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                >
                  <option value="all">Semua teknisi</option>
                  {technicians.map((tech) => (
                    <option key={tech.id} value={tech.id}>
                      {tech.full_name}
                    </option>
                  ))}
                </select>
                <ChevronDown className="pointer-events-none absolute right-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" />
              </span>
            </label>
          )}
        </header>

        {error && (
          <div
            role="alert"
            className="flex items-start gap-3 rounded-xl border border-error/25 bg-error/10 px-4 py-3 text-sm text-error"
          >
            <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <section
          aria-labelledby="calendar-heading"
          className="rounded-2xl border border-border bg-card p-4 sm:p-6"
        >
          <h2 id="calendar-heading" className="sr-only">
            Kalender jadwal pemeliharaan
          </h2>

          {loading ? (
            <div className="flex min-h-64 items-center justify-center gap-3 text-sm text-muted" role="status">
              <LoaderCircle className="h-5 w-5 animate-spin text-primary" />
              Memuat kalender...
            </div>
          ) : (
            <div className="grid gap-6 lg:grid-cols-[1fr_320px]">
              <CalendarGrid
                month={month}
                selectedDay={selectedDay}
                logsByDay={logsByDay}
                onPrevMonth={prevMonth}
                onNextMonth={nextMonth}
                onToday={goToday}
                onSelectDay={setSelectedDay}
                onDayClick={user ? handleDayClick : undefined}
              />

              {/* Panel detail hari */}
              <aside className="lg:max-h-[520px] lg:overflow-y-auto" aria-live="polite">
                <div className="rounded-2xl border border-border bg-background/55 p-4">
                  <div className="flex items-center justify-between gap-2 border-b border-border pb-3">
                    <div>
                      <p className="text-[11px] font-bold uppercase tracking-[0.08em] text-primary">Detail hari</p>
                      <h3 className="mt-0.5 text-sm font-bold capitalize">
                        {selectedDay ? formatLongDate(selectedDay) : "Pilih hari"}
                      </h3>
                    </div>
                    {selectedDay && (
                      <button
                        type="button"
                        onClick={() => setSelectedDay(null)}
                        aria-label="Tutup detail hari"
                        className="flex h-8 w-8 items-center justify-center rounded-lg border border-border bg-background text-muted transition-colors hover:text-foreground"
                      >
                        <X className="h-4 w-4" />
                      </button>
                    )}
                  </div>

                  {!selectedDay ? (
                    <p className="mt-4 text-sm text-muted">
                      Klik salah satu hari pada kalender untuk melihat daftar tugas pemeliharaan.
                    </p>
                  ) : selectedTasks.length === 0 ? (
                    <div className="mt-6 flex flex-col items-center justify-center text-center">
                      <Calendar className="h-9 w-9 text-muted/45" />
                      <p className="mt-2 text-sm font-semibold">Tidak ada tugas</p>
                      <p className="mt-0.5 text-xs text-muted">Tidak ada pemeliharaan pada hari ini.</p>
                    </div>
                  ) : (
                    <ul className="mt-3 space-y-3">
                      {selectedTasks.map((task) => {
                        const meta = STATUS_META[task.status];
                        const Icon = meta.icon;
                        return (
                          <li
                            key={task.id}
                            className="rounded-xl border border-border bg-card p-3"
                          >
                            <div className="flex items-start justify-between gap-2">
                              <div className="flex items-start gap-2">
                                <Icon className={`mt-0.5 h-4 w-4 shrink-0 ${meta.className}`} />
                                <div>
                                  <p className="text-sm font-semibold leading-snug">{task.cctvName}</p>
                                  <p className="text-[11px] font-medium text-muted">{task.cctvCode}</p>
                                </div>
                              </div>
                              <span
                                className={`inline-flex shrink-0 items-center rounded-full border px-2 py-0.5 text-[10px] font-bold uppercase tracking-[0.05em] ${meta.className} border-current/20 bg-current/10`}
                              >
                                {meta.label}
                              </span>
                            </div>

                            <div className="mt-2.5 space-y-1 text-xs text-muted">
                              <p className="flex items-center gap-1.5">
                                <Wrench className="h-3.5 w-3.5" />
                                <span className="line-clamp-2">{task.actionTaken}</span>
                              </p>
                              <p className="flex items-center gap-1.5">
                                <UserIcon className="h-3.5 w-3.5" />
                                {task.technicianName ?? "Belum ditugaskan"}
                              </p>
                            </div>
                          </li>
                        );
                      })}
                    </ul>
                  )}
                </div>
              </aside>
            </div>
          )}
        </section>
      </div>

      {/* MODAL BUAT TUGAS PEMELIHARAAN */}
      {createOpen && createDate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center px-4">
          <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={closeCreateModal} />

          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby="kalender-create-title"
            aria-describedby="kalender-create-description"
            className="z-10 max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl border border-border bg-card p-5 shadow-2xl sm:p-6"
          >
            <div className="mb-5 flex items-start justify-between gap-4">
              <div>
                <h3 id="kalender-create-title" className="text-xl font-bold tracking-[-0.015em]">Buat Tugas Pemeliharaan</h3>
                <p id="kalender-create-description" className="mt-1 text-sm text-muted">
                  Jadwalkan pemeliharaan CCTV pada {formatLongDate(toDayKey(createDate))}.
                </p>
              </div>
              <button
                type="button"
                aria-label="Tutup formulir"
                onClick={closeCreateModal}
                disabled={createSubmitting}
                className="shrink-0 rounded-lg p-2.5 text-muted transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:cursor-not-allowed disabled:opacity-50"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            {createFormError && (
              <div
                role="alert"
                className="mb-4 flex items-start gap-3 rounded-xl border border-error/25 bg-error/10 px-4 py-3 text-sm text-error"
              >
                <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0" />
                <span>{createFormError}</span>
              </div>
            )}

            <form onSubmit={handleSubmitCreate} className="space-y-4">
              {/* Pilih CCTV */}
              <div>
                <label htmlFor="kalender-create-cctv" className="mb-1.5 block text-xs font-semibold text-muted">CCTV *</label>
                <select
                  id="kalender-create-cctv"
                  value={createCctvId}
                  onChange={(e) => setCreateCctvId(e.target.value)}
                  className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                  required
                  disabled={cctvLoading || createSubmitting}
                >
                  <option value="" disabled>
                    {cctvLoading ? "Memuat daftar CCTV…" : "Pilih CCTV"}
                  </option>
                  {cctvs.map((cctv) => (
                    <option key={cctv.id} value={cctv.id}>
                      {cctv.nama} ({cctv.kode_cctv})
                    </option>
                  ))}
                </select>
              </div>

              {/* Tindakan yang dilakukan */}
              <div>
                <label htmlFor="kalender-create-action" className="mb-1.5 block text-xs font-semibold text-muted">Tindakan yang dilakukan *</label>
                <textarea
                  id="kalender-create-action"
                  rows={3}
                  autoFocus
                  value={createAction}
                  onChange={(e) => setCreateAction(e.target.value)}
                  placeholder="Contoh: pembersihan lensa dan pengecekan konektor"
                  className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                  required
                  minLength={5}
                />
                <p className="mt-1.5 text-[11px] text-muted">Minimal 5 karakter.</p>
              </div>

              {/* Waktu terjadwal (pre-filled dari tanggal yang diklik) */}
              <div>
                <label htmlFor="kalender-create-time" className="mb-1.5 block text-xs font-semibold text-muted">Waktu terjadwal *</label>
                <input
                  id="kalender-create-time"
                  type="datetime-local"
                  value={createScheduledDate}
                  onChange={(e) => setCreateScheduledDate(e.target.value)}
                  className="w-full px-4 py-2.5 rounded-xl border border-input-border bg-input text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/45 focus:border-primary transition-all"
                  required
                />
                <p className="mt-1.5 text-[11px] text-muted">Tugas akan ditugaskan ke teknisi oleh admin sebelum dikerjakan.</p>
              </div>

              {/* Catatan tambahan */}
              <div>
                <label htmlFor="kalender-create-notes" className="mb-1.5 block text-xs font-semibold text-muted">Catatan tambahan</label>
                <textarea
                  id="kalender-create-notes"
                  rows={2}
                  value={createNotes}
                  onChange={(e) => setCreateNotes(e.target.value)}
                  placeholder="Tambahkan hasil pemeriksaan atau temuan teknis"
                  className="w-full rounded-xl border border-input-border bg-input px-4 py-2.5 text-sm text-foreground transition-colors focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/30"
                />
              </div>

              <div className="flex justify-end gap-3 pt-4 border-t border-border mt-6">
                <button
                  type="button"
                  onClick={closeCreateModal}
                  disabled={createSubmitting}
                  className="rounded-xl border border-border px-5 py-2.5 text-sm font-semibold transition-colors hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={createSubmitting || cctvLoading}
                  className="flex items-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(15,118,74,0.16)] transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-card disabled:cursor-wait disabled:opacity-65"
                >
                  {createSubmitting && <LoaderCircle className="h-4 w-4 animate-spin" />}
                  {createSubmitting ? "Menyimpan…" : <><Plus className="h-4 w-4" /> Simpan</>}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </DashboardLayout>
  );
}

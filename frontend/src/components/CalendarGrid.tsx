"use client";

/**
 * CalendarGrid — Grid kalender bulanan (Task 8.2).
 *
 * Implementasi CSS mandiri (tanpa dependency chart eksternal). Menampilkan
 * grid 7 kolom × 5-6 baris dengan header hari (Senin–Minggu) dan navigasi
 * bulan. Setiap sel hari menampilkan titik (dot) kecil berwarna sesuai status
 * tugas pada hari itu:
 *   - hijau  = completed
 *   - biru   = scheduled
 *   - kuning = ongoing
 *   - merah  = failed/overdue
 *
 * Prop `logsByDay` memetakan tanggal "YYYY-MM-DD" ke daftar ringkasan tugas.
 */
import type { LucideIcon } from "lucide-react";
import {
  ChevronLeft,
  ChevronRight,
  CircleCheck,
  CircleX,
  CircleDashed,
  Clock3,
  Plus,
} from "lucide-react";

// Ringkasan satu tugas untuk ditampilkan di kalender (tanpa beban data penuh).
export interface CalendarTask {
  id: string;
  status: "scheduled" | "ongoing" | "completed" | "failed";
  cctvName: string;
  cctvCode: string;
  technicianName?: string | null;
  actionTaken: string;
}

export interface CalendarGridProps {
  month: Date; // tanggal 1 bulan yang sedang ditampilkan
  selectedDay: string | null; // "YYYY-MM-DD" yang dipilih
  logsByDay: Record<string, CalendarTask[]>;
  onPrevMonth: () => void;
  onNextMonth: () => void;
  onToday: () => void;
  onSelectDay: (dayKey: string | null) => void;
  // Dipanggil saat tombol "+" di pojok sel diklik (untuk membuat tugas baru).
  onDayClick?: (date: Date) => void;
}

// Urutan hari mulai Senin (index 0) hingga Minggu (index 6).
const DAY_LABELS = ["Sen", "Sel", "Rab", "Kam", "Jum", "Sab", "Min"];
const MONTH_LABELS = [
  "Januari",
  "Februari",
  "Maret",
  "April",
  "Mei",
  "Juni",
  "Juli",
  "Agustus",
  "September",
  "Oktober",
  "November",
  "Desember",
];

// Pemetaan status → warna dot (mengikuti design system SIGAP).
export const STATUS_DOT_CLASS: Record<CalendarTask["status"], string> = {
  completed: "bg-[#467235]",
  scheduled: "bg-[#3b82f6]",
  ongoing: "bg-[#ffbf00]",
  failed: "bg-[#b54432]",
};

// Ikon & label status untuk panel detail.
export const STATUS_META: Record<
  CalendarTask["status"],
  { label: string; icon: LucideIcon; className: string }
> = {
  completed: { label: "Selesai", icon: CircleCheck, className: "text-[#467235]" },
  scheduled: { label: "Terjadwal", icon: CircleDashed, className: "text-[#3b82f6]" },
  ongoing: { label: "Berjalan", icon: Clock3, className: "text-[#ffbf00]" },
  failed: { label: "Gagal / Overdue", icon: CircleX, className: "text-[#b54432]" },
};

// Format tanggal "YYYY-MM-DD" → "Senin, 7 Agustus 2026" (id-ID).
export function formatLongDate(dayKey: string): string {
  const [y, m, d] = dayKey.split("-").map(Number);
  if (!y || !m || !d) return dayKey;
  return new Intl.DateTimeFormat("id-ID", {
    weekday: "long",
    day: "numeric",
    month: "long",
    year: "numeric",
  }).format(new Date(y, m - 1, d, 12, 0, 0));
}

export function formatMonthTitle(month: Date): string {
  return `${MONTH_LABELS[month.getMonth()]} ${month.getFullYear()}`;
}

// Kunci "YYYY-MM-DD" dari objek Date (gunakan waktu lokal tengah hari agar
// tidak bergeser lintas zona waktu).
export function toDayKey(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

export default function CalendarGrid({
  month,
  selectedDay,
  logsByDay,
  onPrevMonth,
  onNextMonth,
  onToday,
  onSelectDay,
  onDayClick,
}: CalendarGridProps) {
  // Hari pertama bulan (untuk menghitung offset sel kosong).
  const firstOfMonth = new Date(month.getFullYear(), month.getMonth(), 1);
  // Hari dalam bulan ini.
  const daysInMonth = new Date(
    month.getFullYear(),
    month.getMonth() + 1,
    0
  ).getDate();

  // JS getDay(): 0=Minggu..6=Sabtu. Kami ingin Senin sebagai kolom pertama,
  // sehingga offset = (getDay()+6)%7.
  const leadingBlank = (firstOfMonth.getDay() + 6) % 7;

  // Sel: blank awal + seluruh hari, lalu isi hingga kelipatan 7 (baris penuh).
  const cells: (number | null)[] = [
    ...Array.from({ length: leadingBlank }, () => null),
    ...Array.from({ length: daysInMonth }, (_, i) => i + 1),
  ];
  while (cells.length % 7 !== 0) cells.push(null);

  const todayKey = toDayKey(new Date());
  const isCurrentMonth =
    month.getFullYear() === new Date().getFullYear() &&
    month.getMonth() === new Date().getMonth();

  return (
    <div>
      {/* Header bulan + navigasi */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-lg font-bold tracking-[-0.015em]">{formatMonthTitle(month)}</h2>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={onToday}
            className="rounded-lg border border-border bg-background px-3 py-2 text-xs font-semibold text-foreground transition-colors hover:border-primary/40 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
          >
            Hari ini
          </button>
          <button
            type="button"
            onClick={onPrevMonth}
            aria-label="Bulan sebelumnya"
            className="flex h-9 w-9 items-center justify-center rounded-lg border border-border bg-background text-foreground transition-colors hover:border-primary/40 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
          >
            <ChevronLeft className="h-4 w-4" />
          </button>
          <button
            type="button"
            onClick={onNextMonth}
            aria-label="Bulan berikutnya"
            className="flex h-9 w-9 items-center justify-center rounded-lg border border-border bg-background text-foreground transition-colors hover:border-primary/40 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
          >
            <ChevronRight className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Grid kalender */}
      <div className="mt-4 grid grid-cols-7 gap-px overflow-hidden rounded-2xl border border-border bg-border">
        {/* Header hari */}
        {DAY_LABELS.map((label) => (
          <div
            key={label}
            className="bg-card px-2 py-2 text-center text-[11px] font-bold uppercase tracking-[0.08em] text-muted"
          >
            {label}
          </div>
        ))}

        {/* Sel hari */}
        {cells.map((day, index) => {
          if (day === null) {
            return <div key={`blank-${index}`} className="min-h-20 bg-card" aria-hidden="true" />;
          }

          const dayKey = `${month.getFullYear()}-${String(month.getMonth() + 1).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
          const tasks = logsByDay[dayKey] ?? [];
          const isSelected = selectedDay === dayKey;
          const isToday = isCurrentMonth && dayKey === todayKey;
          // Maksimal 3 dot per sel agar rapi; sisanya ditampilkan sebagai jumlah.
          const dots = tasks.slice(0, 3);
          const extraCount = tasks.length - dots.length;

          return (
            <div
              key={dayKey}
              className={`group relative min-h-20 transition-colors ${
                isSelected ? "bg-primary/10 ring-1 ring-inset ring-primary" : "bg-card hover:bg-background"
              }`}
            >
              {/* Tombol utama sel: membuka detail hari. */}
              <button
                type="button"
                onClick={() => onSelectDay(isSelected ? null : dayKey)}
                aria-pressed={isSelected}
                aria-label={`${day} ${formatMonthTitle(month)}${tasks.length ? `, ${tasks.length} tugas` : ""}`}
                className="flex min-h-20 w-full flex-col items-stretch justify-between p-1.5 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-inset"
              >
                <span
                  className={`flex h-6 w-6 items-center justify-center rounded-full text-xs font-semibold tabular-nums ${
                    isToday ? "bg-primary text-white" : "text-foreground"
                  } ${tasks.length ? "" : "opacity-70"}`}
                >
                  {day}
                </span>

                <span className="flex flex-wrap gap-1">
                  {dots.map((task) => (
                    <span
                      key={task.id}
                      title={`${task.cctvName} (${STATUS_META[task.status].label})`}
                      className={`h-2 w-2 rounded-full ${STATUS_DOT_CLASS[task.status]}`}
                    />
                  ))}
                  {extraCount > 0 && (
                    <span className="text-[10px] font-semibold text-muted">+{extraCount}</span>
                  )}
                </span>
              </button>

              {/* Tombol "+" kecil di pojok: membuat tugas baru pada hari ini. */}
              {onDayClick && (
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    onDayClick(new Date(month.getFullYear(), month.getMonth(), day, 12, 0, 0));
                  }}
                  aria-label={`Tambah tugas ${day} ${formatMonthTitle(month)}`}
                  className="absolute right-1.5 top-1.5 flex h-6 w-6 items-center justify-center rounded-md border border-border bg-background text-muted opacity-0 shadow-sm transition-opacity group-hover:opacity-100 hover:bg-primary/10 hover:text-primary focus-visible:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                >
                  <Plus className="h-3 w-3" />
                </button>
              )}
            </div>
          );
        })}
      </div>

      {/* Legend status */}
      <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1.5">
        {(
          [
            ["completed", "Selesai"],
            ["scheduled", "Terjadwal"],
            ["ongoing", "Berjalan"],
            ["failed", "Gagal/Overdue"],
          ] as [CalendarTask["status"], string][]
        ).map(([status, label]) => (
          <span key={status} className="inline-flex items-center gap-1.5 text-xs text-muted">
            <span className={`h-2.5 w-2.5 rounded-full ${STATUS_DOT_CLASS[status]}`} />
            {label}
          </span>
        ))}
      </div>
    </div>
  );
}

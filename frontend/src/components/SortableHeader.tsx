'use client';

import { ArrowDown, ArrowDownUp, ArrowUp } from 'lucide-react';

interface SortableHeaderProps {
  label: string;
  sortKey: string;
  activeKey: string | null;
  direction: 'asc' | 'desc';
  onSort: (key: string) => void;
  align?: 'left' | 'center' | 'right';
}

/**
 * Judul kolom tabel yang bisa diklik untuk mengurutkan data.
 * - Klik pertama kali: urutkan (teks naik / angka turun otomatis).
 * - Klik lagi: balik arah urutan.
 * - Menampilkan ikon panah indikator arah sortir.
 */
export default function SortableHeader({
  label,
  sortKey,
  activeKey,
  direction,
  onSort,
  align = 'left',
}: SortableHeaderProps) {
  const isActive = activeKey === sortKey;
  const alignClass =
    align === 'center' ? 'justify-center' : align === 'right' ? 'justify-end' : 'justify-start';

  return (
    <button
      type="button"
      onClick={() => onSort(sortKey)}
      aria-label={`Urutkan berdasarkan ${label}`}
      className={`inline-flex items-center gap-1.5 rounded-md text-left transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 ${alignClass}`}
    >
      <span>{label}</span>
      {isActive ? (
        direction === 'asc' ? (
          <ArrowUp className="h-3.5 w-3.5 text-primary" aria-hidden="true" />
        ) : (
          <ArrowDown className="h-3.5 w-3.5 text-primary" aria-hidden="true" />
        )
      ) : (
        <ArrowDownUp className="h-3.5 w-3.5 text-muted/50" aria-hidden="true" />
      )}
    </button>
  );
}

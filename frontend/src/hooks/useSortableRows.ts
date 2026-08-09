'use client';

import { useCallback, useMemo, useState } from 'react';

export type SortDirection = 'asc' | 'desc';

interface UseSortableRowsOptions<T> {
  /** Ambil nilai yang akan diurutkan dari satu baris untuk satu kolom. */
  getValue: (row: T, key: string) => string | number | null | undefined;
  /** Kunci default yang aktif saat pertama kali. Bisa null bila belum ada. */
  defaultKey?: string | null;
  /** Arah default. Untuk kolom teks gunakan 'asc', untuk angka gunakan 'desc'. */
  defaultDirection?: SortDirection;
}

/**
 * Hook untuk mengurutkan baris tabel secara client-side.
 * - onSort(key): toggle arah bila kunci sama, atau set kunci baru (teks naik, angka turun).
 * - sortedRows: hasil urutan (di-memoize).
 * - sortKey / sortDirection: nilai aktif untuk dirender pada SortableHeader.
 */
export function useSortableRows<T>(
  rows: T[],
  { getValue, defaultKey = null, defaultDirection = 'desc' }: UseSortableRowsOptions<T>
) {
  const [sortKey, setSortKey] = useState<string | null>(defaultKey);
  const [sortDirection, setSortDirection] = useState<SortDirection>(defaultDirection);

  const onSort = useCallback(
    (key: string) => {
      if (sortKey === key) {
        setSortDirection((current) => (current === 'asc' ? 'desc' : 'asc'));
        return;
      }
      setSortKey(key);
      // Kolom angka default turun (terbesar dulu); kolom teks default naik.
      const sample = rows.find((r) => getValue(r, key) != null);
      const firstValue = sample ? getValue(sample, key) : undefined;
      setSortDirection(typeof firstValue === 'number' ? 'desc' : 'asc');
    },
    [sortKey, rows, getValue]
  );

  const sortedRows = useMemo(() => {
    if (!sortKey) return rows;
    const sorted = [...rows].sort((a, b) => {
      const aValue = getValue(a, sortKey);
      const bValue = getValue(b, sortKey);
      const result =
        typeof aValue === 'number' && typeof bValue === 'number'
          ? aValue - bValue
          : String(aValue ?? '').localeCompare(String(bValue ?? ''), 'id', { numeric: true });
      return sortDirection === 'asc' ? result : -result;
    });
    return sorted;
  }, [rows, sortKey, sortDirection, getValue]);

  return { sortedRows, sortKey, sortDirection, onSort };
}

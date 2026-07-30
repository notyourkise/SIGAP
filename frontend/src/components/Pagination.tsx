'use client';

import { ChevronLeft, ChevronRight } from 'lucide-react';

interface PaginationProps {
  page: number;
  limit: number;
  totalRows: number;
  totalPages: number;
  onPageChange: (page: number) => void;
  itemLabel?: string;
}

export default function Pagination({
  page,
  limit,
  totalRows,
  totalPages,
  onPageChange,
  itemLabel = 'data',
}: PaginationProps) {
  const safeTotalPages = Math.max(totalPages, 1);
  const safePage = Math.min(Math.max(page, 1), safeTotalPages);
  const start = totalRows === 0 ? 0 : (safePage - 1) * limit + 1;
  const end = totalRows === 0 ? 0 : Math.min(safePage * limit, totalRows);

  const firstVisiblePage = Math.max(1, Math.min(safePage - 1, safeTotalPages - 2));
  const visiblePages = Array.from(
    { length: Math.min(3, safeTotalPages) },
    (_, index) => firstVisiblePage + index,
  );

  return (
    <div className="flex flex-col gap-3 border-t border-border bg-background/35 px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
      <p className="text-xs font-medium text-muted">
        Menampilkan <span className="font-bold text-foreground">{start}-{end}</span> dari{' '}
        <span className="font-bold text-foreground">{totalRows}</span> {itemLabel}
      </p>

      <div className="flex items-center justify-between gap-2 sm:justify-end">
        <span className="mr-1 text-xs text-muted">
          Halaman {safePage} dari {safeTotalPages}
        </span>
        <button
          type="button"
          onClick={() => onPageChange(safePage - 1)}
          disabled={safePage <= 1}
          className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-border bg-card text-foreground transition-colors hover:border-primary hover:text-primary disabled:cursor-not-allowed disabled:opacity-40"
          aria-label="Halaman sebelumnya"
        >
          <ChevronLeft className="h-4 w-4" />
        </button>

        {visiblePages.map((pageNumber) => (
          <button
            type="button"
            key={pageNumber}
            onClick={() => onPageChange(pageNumber)}
            aria-current={pageNumber === safePage ? 'page' : undefined}
            aria-label={`Buka halaman ${pageNumber}`}
            className={`h-9 min-w-9 rounded-lg border px-3 text-xs font-bold transition-colors ${
              pageNumber === safePage
                ? 'border-primary bg-primary text-white shadow-sm'
                : 'border-border bg-card text-muted hover:border-primary hover:text-primary'
            }`}
          >
            {pageNumber}
          </button>
        ))}

        <button
          type="button"
          onClick={() => onPageChange(safePage + 1)}
          disabled={safePage >= safeTotalPages}
          className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-border bg-card text-foreground transition-colors hover:border-primary hover:text-primary disabled:cursor-not-allowed disabled:opacity-40"
          aria-label="Halaman berikutnya"
        >
          <ChevronRight className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}

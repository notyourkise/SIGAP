'use client';

import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { Check, Copy, Info, X } from 'lucide-react';
import type { ApiFieldError } from '@/lib/api';
import { formatValidationMessages } from '@/lib/api';

type ToastType = 'success' | 'error' | 'info';

interface ToastDetail {
  message: string;
  page?: string;
  timestamp: string;
  source?: string;
  field?: string;
  reason?: string;
  value?: string;
  line?: number;
  column?: number;
  status?: number;
  endpoint?: string;
  method?: string;
  requestId?: string;
  errors?: ApiFieldError[];
  stack?: string;
}

interface ToastItem {
  id: number;
  type: ToastType;
  message: string;
  detail: ToastDetail;
  duration: number;
}

interface ToastApi {
  success: (message: string) => void;
  error: (message: string, detail?: unknown) => void;
  info: (message: string) => void;
}

const ToastContext = createContext<ToastApi | null>(null);

export function useToast(): ToastApi {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error('useToast harus dipakai di dalam <ToastProvider>');
  return ctx;
}

const TOAST_DURATION = 4200;
const ERROR_TOAST_DURATION = 9000;

const typeStyles: Record<ToastType, { icon: string; bar: string }> = {
  success: { icon: 'text-success', bar: 'bg-success' },
  error: { icon: 'text-error', bar: 'bg-error' },
  info: { icon: 'text-blue-500', bar: 'bg-blue-500' },
};

function asString(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value : undefined;
}

function asNumber(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined;
}

function currentPage(): string | undefined {
  if (typeof window === 'undefined') return undefined;
  return `${window.location.pathname}${window.location.search}`;
}

function formatTimestamp(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('id-ID', {
    dateStyle: 'medium',
    timeStyle: 'medium',
  });
}

// Bangun detail dari Error/objek apa pun yang dikirim pemanggil toast.
// API error membawa detail field/penyebab; error non-API tetap mendapat
// konteks halaman dan waktu supaya tombol Detail selalu berguna.
function buildToastDetail(message: string, error?: unknown): ToastDetail {
  const base: ToastDetail = {
    message,
    page: currentPage(),
    timestamp: new Date().toISOString(),
  };

  if (!error) {
    return { ...base, source: 'Aplikasi' };
  }

  if (error instanceof Error) {
    const candidate = error as Error & {
      status?: unknown;
      endpoint?: unknown;
      method?: unknown;
      requestId?: unknown;
      detail?: unknown;
      errors?: unknown;
    };
    const nested =
      candidate.detail && typeof candidate.detail === 'object'
        ? (candidate.detail as Record<string, unknown>)
        : undefined;

    return {
      ...base,
      source: candidate.status || candidate.endpoint ? 'Permintaan API' : 'Aplikasi',
      reason: asString(nested?.reason) ?? asString(candidate.message) ?? message,
      field: asString(nested?.field),
      value: asString(nested?.value),
      line: asNumber(nested?.line),
      column: asNumber(nested?.column),
      status: asNumber(candidate.status),
      endpoint: asString(candidate.endpoint),
      method: asString(candidate.method),
      requestId: asString(candidate.requestId),
      errors: Array.isArray(candidate.errors)
        ? (candidate.errors as ApiFieldError[])
        : undefined,
      stack: candidate.stack,
    };
  }

  if (typeof error === 'object') {
    const candidate = error as Record<string, unknown>;
    const nested =
      candidate.detail && typeof candidate.detail === 'object'
        ? (candidate.detail as Record<string, unknown>)
        : undefined;

    return {
      ...base,
      source: asString(candidate.source) ?? 'Aplikasi',
      reason:
        asString(candidate.reason) ??
        asString(nested?.reason) ??
        asString(candidate.message),
      field: asString(candidate.field) ?? asString(nested?.field),
      value: asString(candidate.value) ?? asString(nested?.value),
      line: asNumber(candidate.line) ?? asNumber(nested?.line),
      column: asNumber(candidate.column) ?? asNumber(nested?.column),
      status: asNumber(candidate.status),
      endpoint: asString(candidate.endpoint),
      method: asString(candidate.method),
      requestId: asString(candidate.requestId),
      errors: Array.isArray(candidate.errors)
        ? (candidate.errors as ApiFieldError[])
        : undefined,
      stack: asString(candidate.stack),
    };
  }

  return {
    ...base,
    source: 'Aplikasi',
    reason: String(error),
  };
}

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const [leavingIds, setLeavingIds] = useState<number[]>([]);
  const [detailToast, setDetailToast] = useState<ToastItem | null>(null);
  const idRef = useRef(0);

  const dismiss = useCallback((id: number) => {
    setLeavingIds((ids) => (ids.includes(id) ? ids : [...ids, id]));
    setTimeout(() => {
      setToasts((list) => list.filter((t) => t.id !== id));
      setLeavingIds((ids) => ids.filter((x) => x !== id));
    }, 220);
  }, []);

  const push = useCallback(
    (type: ToastType, message: string, error?: unknown) => {
      const id = ++idRef.current;
      const duration = type === 'error' ? ERROR_TOAST_DURATION : TOAST_DURATION;
      const item: ToastItem = {
        id,
        type,
        message,
        detail: buildToastDetail(message, error),
        duration,
      };

      setToasts((list) => [...list.slice(-3), item]);
      setTimeout(() => dismiss(id), duration);
    },
    [dismiss]
  );

  const openDetail = useCallback(
    (item: ToastItem) => {
      setDetailToast(item);
      dismiss(item.id);
    },
    [dismiss]
  );

  const api = useMemo<ToastApi>(
    () => ({
      success: (m) => push('success', m),
      error: (m, detail) => push('error', m, detail),
      info: (m) => push('info', m),
    }),
    [push]
  );

  return (
    <ToastContext.Provider value={api}>
      {children}
      <div className="pointer-events-none fixed right-4 top-4 z-[80] flex w-full max-w-sm flex-col items-end gap-2.5">
        {toasts.map((t) => (
          <ToastCard
            key={t.id}
            toast={t}
            leaving={leavingIds.includes(t.id)}
            onClose={() => dismiss(t.id)}
            onShowDetail={() => openDetail(t)}
          />
        ))}
      </div>
      {detailToast && (
        <ErrorDetailModal
          toast={detailToast}
          onClose={() => setDetailToast(null)}
        />
      )}
    </ToastContext.Provider>
  );
}

function ToastCard({
  toast,
  leaving,
  onClose,
  onShowDetail,
}: {
  toast: ToastItem;
  leaving: boolean;
  onClose: () => void;
  onShowDetail: () => void;
}) {
  const styles = typeStyles[toast.type];

  return (
    <div
      role={toast.type === 'error' ? 'alert' : 'status'}
      aria-live={toast.type === 'error' ? 'assertive' : 'polite'}
      className={`pointer-events-auto relative w-full overflow-hidden rounded-2xl border border-border bg-card shadow-[0_16px_40px_rgba(15,23,42,0.18)] transition-all duration-200 ${
        leaving ? 'pointer-events-none translate-x-6 opacity-0' : 'animate-toast-in'
      }`}
    >
      <div className="flex items-start gap-3 px-4 py-3.5">
        {toast.type === 'success' ? (
          <span className={`relative mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-success/15 ${styles.icon}`}>
            <svg viewBox="0 0 24 24" className="h-5 w-5 animate-check-pop" fill="none" aria-hidden="true">
              <circle
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                strokeWidth="2"
                strokeDasharray="63"
                strokeDashoffset="63"
                className="animate-check-circle"
              />
              <path
                d="M8 12.5l2.6 2.6L16 9.5"
                stroke="currentColor"
                strokeWidth="2.2"
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeDasharray="14"
                strokeDashoffset="14"
                className="animate-check-path"
              />
            </svg>
          </span>
        ) : (
          <span className={`relative mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-white ${styles.bar} animate-check-pop`}>
            <svg viewBox="0 0 24 24" className="h-5 w-5 animate-check-pop" fill="none" aria-hidden="true">
              {toast.type === 'error' ? (
                <>
                  <path d="M8 8l8 8M16 8l-8 8" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeDasharray="11" strokeDashoffset="11" className="animate-check-path" />
                </>
              ) : (
                <>
                  <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="2" strokeDasharray="63" strokeDashoffset="63" className="animate-check-circle" />
                  <path d="M12 7v5.5M12 16.2v.3" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeDasharray="14" strokeDashoffset="14" className="animate-check-path" />
                </>
              )}
            </svg>
          </span>
        )}
        <div className="min-w-0 flex-1 pt-1">
          <p className="text-sm font-medium leading-snug text-foreground">{toast.message}</p>
          {toast.type === 'error' && (
            <button
              type="button"
              onClick={onShowDetail}
              className="mt-1.5 inline-flex items-center gap-1 rounded-md text-xs font-semibold text-error underline-offset-2 transition-colors hover:text-error/80 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-error/40"
            >
              <Info className="h-3.5 w-3.5" aria-hidden="true" />
              Detail error
            </button>
          )}
        </div>
        <button
          type="button"
          onClick={onClose}
          aria-label="Tutup notifikasi"
          className="mt-1 shrink-0 rounded-lg p-1 text-muted transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
      <div
        className={`h-0.5 w-full origin-left ${styles.bar} animate-toast-progress`}
        style={{ animationDuration: `${toast.duration}ms` }}
      />
    </div>
  );
}

function ErrorDetailModal({
  toast,
  onClose,
}: {
  toast: ToastItem;
  onClose: () => void;
}) {
  const [copied, setCopied] = useState(false);
  const detail = toast.detail;
  const dialogRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Fokuskan dialog tanpa menggeser scroll. autoFocus pada tombol "Tutup"
    // membuat container langsung melompat ke bawah dan header tidak terlihat.
    dialogRef.current?.focus({ preventScroll: true });
  }, []);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
    };
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    document.addEventListener('keydown', onKey);
    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener('keydown', onKey);
    };
  }, [onClose]);

  const rows: Array<{ label: string; value: string }> = [];
  if (detail.field) rows.push({ label: 'Field', value: detail.field });
  if (detail.reason && detail.reason !== toast.message) {
    rows.push({ label: 'Penyebab', value: detail.reason });
  }
  if (detail.value) rows.push({ label: 'Nilai', value: detail.value });
  if (detail.line !== undefined || detail.column !== undefined) {
    rows.push({
      label: 'Lokasi JSON',
      value: `Baris ${detail.line ?? '?'}, kolom ${detail.column ?? '?'}`,
    });
  }
  if (detail.status !== undefined) {
    rows.push({ label: 'Status HTTP', value: String(detail.status) });
  }
  if (detail.endpoint) {
    rows.push({
      label: 'Endpoint',
      value: `${detail.method || 'GET'} ${detail.endpoint}`,
    });
  }
  if (detail.requestId) rows.push({ label: 'Request ID', value: detail.requestId });
  if (detail.page) rows.push({ label: 'Halaman', value: detail.page });
  if (detail.timestamp) {
    rows.push({ label: 'Waktu', value: formatTimestamp(detail.timestamp) });
  }

  const validationMessages = detail.errors?.length
    ? formatValidationMessages(detail.errors)
    : [];

  const copyPayload = JSON.stringify(
    {
      ...detail,
      message: toast.message,
    },
    null,
    2
  );

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(copyPayload);
      setCopied(true);
      setTimeout(() => setCopied(false), 1800);
    } catch {
      // Clipboard bisa ditolak browser; detail tetap bisa dibaca di modal.
    }
  };

  return (
    <div
      className="fixed inset-0 z-[90] flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm animate-fade-in"
      role="dialog"
      aria-modal="true"
      aria-labelledby="error-detail-title"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div
        ref={dialogRef}
        tabIndex={-1}
        className="relative max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl border border-border bg-card p-5 shadow-2xl outline-none animate-pop-in sm:p-6"
      >
        <div className="flex items-start justify-between gap-4 border-b border-border pb-4">
          <div className="flex min-w-0 items-start gap-3">
            <span className="mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-error/10 text-error">
              <Info className="h-5 w-5" aria-hidden="true" />
            </span>
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <h2 id="error-detail-title" className="text-lg font-bold">
                  Detail Error
                </h2>
                {detail.source && (
                  <span className="rounded-full bg-background px-2 py-0.5 text-[11px] font-semibold text-muted">
                    {detail.source}
                  </span>
                )}
              </div>
              <p className="mt-1 break-words text-sm leading-relaxed text-muted">
                {toast.message}
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Tutup detail error"
            className="shrink-0 rounded-lg p-1.5 text-muted transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="mt-4 space-y-4">
          {rows.length > 0 ? (
            <dl className="grid gap-3 sm:grid-cols-2">
              {rows.map((row) => (
                <div key={row.label} className="rounded-xl border border-border bg-background/60 px-3 py-2.5">
                  <dt className="text-[11px] font-semibold uppercase tracking-wide text-muted">
                    {row.label}
                  </dt>
                  <dd className="mt-1 break-words text-sm font-medium text-foreground">
                    {row.value}
                  </dd>
                </div>
              ))}
            </dl>
          ) : (
            <p className="rounded-xl border border-border bg-background/60 px-3 py-3 text-sm text-muted">
              Tidak ada detail teknis tambahan. Pesan di atas adalah informasi yang tersedia.
            </p>
          )}

          {validationMessages.length > 0 && (
            <div className="rounded-xl border border-error/20 bg-error/5 px-3 py-3">
              <p className="text-[11px] font-semibold uppercase tracking-wide text-error">
                Isian yang bermasalah
              </p>
              <ul className="mt-2 list-disc space-y-1 pl-5 text-sm text-foreground">
                {validationMessages.map((message, index) => (
                  <li key={`${message}-${index}`}>{message}</li>
                ))}
              </ul>
            </div>
          )}

          {detail.stack && (
            <details className="rounded-xl border border-border bg-background/60 px-3 py-3">
              <summary className="cursor-pointer text-xs font-semibold text-muted">
                Detail teknis (untuk developer)
              </summary>
              <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-words text-[11px] leading-relaxed text-muted">
                {detail.stack}
              </pre>
            </details>
          )}
        </div>

        <div className="mt-6 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <button
            type="button"
            onClick={handleCopy}
            className="inline-flex h-10 items-center justify-center gap-2 rounded-xl border border-border bg-card px-4 text-sm font-semibold text-foreground transition-colors hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
          >
            {copied ? <Check className="h-4 w-4 text-success" /> : <Copy className="h-4 w-4" />}
            {copied ? 'Tersalin' : 'Salin detail'}
          </button>
          <button
            type="button"
            onClick={onClose}
            className="inline-flex h-10 items-center justify-center rounded-xl bg-primary px-4 text-sm font-semibold text-white transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
          >
            Tutup
          </button>
        </div>
      </div>
    </div>
  );
}

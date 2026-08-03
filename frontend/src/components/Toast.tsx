'use client';

import React, { createContext, useCallback, useContext, useMemo, useRef, useState } from 'react';
import { X } from 'lucide-react';

type ToastType = 'success' | 'error' | 'info';

interface ToastItem {
  id: number;
  type: ToastType;
  message: string;
}

interface ToastApi {
  success: (message: string) => void;
  error: (message: string) => void;
  info: (message: string) => void;
}

const ToastContext = createContext<ToastApi | null>(null);

export function useToast(): ToastApi {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error('useToast harus dipakai di dalam <ToastProvider>');
  return ctx;
}

const TOAST_DURATION = 4200;

const typeStyles: Record<ToastType, { icon: string; bar: string }> = {
  success: { icon: 'text-success', bar: 'bg-success' },
  error: { icon: 'text-error', bar: 'bg-error' },
  info: { icon: 'text-blue-500', bar: 'bg-blue-500' },
};

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const [leavingIds, setLeavingIds] = useState<number[]>([]);
  const idRef = useRef(0);

  const dismiss = useCallback((id: number) => {
    setLeavingIds((ids) => (ids.includes(id) ? ids : [...ids, id]));
    setTimeout(() => {
      setToasts((list) => list.filter((t) => t.id !== id));
      setLeavingIds((ids) => ids.filter((x) => x !== id));
    }, 220);
  }, []);

  const push = useCallback(
    (type: ToastType, message: string) => {
      const id = ++idRef.current;
      setToasts((list) => [...list.slice(-3), { id, type, message }]);
      setTimeout(() => dismiss(id), TOAST_DURATION);
    },
    [dismiss]
  );

  const api = useMemo<ToastApi>(
    () => ({
      success: (m) => push('success', m),
      error: (m) => push('error', m),
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
          />
        ))}
      </div>
    </ToastContext.Provider>
  );
}

function ToastCard({ toast, leaving, onClose }: { toast: ToastItem; leaving: boolean; onClose: () => void }) {
  const styles = typeStyles[toast.type];

  return (
    <div
      role="status"
      aria-live="polite"
      className={`relative w-full overflow-hidden rounded-2xl border border-border bg-card shadow-[0_16px_40px_rgba(15,23,42,0.18)] transition-all duration-200 ${
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
        <p className="min-w-0 flex-1 pt-1 text-sm font-medium leading-snug text-foreground">{toast.message}</p>
        <button
          type="button"
          onClick={onClose}
          aria-label="Tutup notifikasi"
          className="mt-1 shrink-0 rounded-lg p-1 text-muted transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
      <div className={`h-0.5 w-full origin-left ${styles.bar} animate-toast-progress`} style={{ animationDuration: `${TOAST_DURATION}ms` }} />
    </div>
  );
}

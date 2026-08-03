'use client';

import React, { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react';
import { AlertTriangle, X } from 'lucide-react';

interface ConfirmOptions {
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  tone?: 'danger' | 'primary';
}

interface ConfirmState extends ConfirmOptions {
  resolve: (value: boolean) => void;
}

const ConfirmContext = createContext<(options: ConfirmOptions) => Promise<boolean>>(() =>
  Promise.resolve(false)
);

export function useConfirm(): (options: ConfirmOptions) => Promise<boolean> {
  return useContext(ConfirmContext);
}

export function ConfirmProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<ConfirmState | null>(null);
  const [leaving, setLeaving] = useState(false);
  const closeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const close = useCallback(
    (result: boolean) => {
      if (!state) return;
      setLeaving(true);
      if (closeTimerRef.current) clearTimeout(closeTimerRef.current);
      closeTimerRef.current = setTimeout(() => {
        state.resolve(result);
        setState(null);
        setLeaving(false);
      }, 160);
    },
    [state]
  );

  const confirm = useCallback((options: ConfirmOptions) => {
    return new Promise<boolean>((resolve) => {
      setState({ ...options, resolve });
      setLeaving(false);
    });
  }, []);

  useEffect(() => {
    if (!state) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close(false);
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [state, close]);

  const tone = state?.tone ?? 'danger';

  return (
    <ConfirmContext.Provider value={confirm}>
      {children}
      {state && (
        <div
          className={`fixed inset-0 z-[70] flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm ${leaving ? 'animate-fade-out' : 'animate-fade-in'}`}
          role="dialog"
          aria-modal="true"
          aria-labelledby="confirm-dialog-heading"
          onMouseDown={(e) => {
            if (e.target === e.currentTarget) close(false);
          }}
        >
          <div
            className={`relative w-full max-w-sm space-y-5 rounded-2xl border border-border bg-card p-6 text-center shadow-2xl ${leaving ? 'animate-pop-out' : 'animate-pop-in'}`}
          >
            <button
              type="button"
              onClick={() => close(false)}
              aria-label="Tutup dialog konfirmasi"
              className="absolute right-3 top-3 rounded-lg p-1.5 text-muted transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            >
              <X className="h-4 w-4" />
            </button>

            <div className="flex justify-center">
              <span
                className={`flex h-16 w-16 items-center justify-center rounded-full ${
                  tone === 'danger' ? 'bg-error/10' : 'bg-primary/10'
                } animate-wiggle-slow`}
              >
                <AlertTriangle className={`h-7 w-7 ${tone === 'danger' ? 'text-error' : 'text-primary'}`} />
              </span>
            </div>

            <div>
              <h2 id="confirm-dialog-heading" className="text-lg font-bold">
                {state.title}
              </h2>
              <p className="mt-1.5 text-sm leading-relaxed text-muted">{state.message}</p>
            </div>

            <div className="flex flex-col gap-2 sm:flex-row">
              <button
                type="button"
                onClick={() => close(false)}
                className="flex h-11 flex-1 items-center justify-center rounded-xl border border-border bg-card px-4 text-sm font-semibold text-foreground transition-colors hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
              >
                {state.cancelLabel ?? 'Batal'}
              </button>
              <button
                type="button"
                onClick={() => close(true)}
                autoFocus
                className={`flex h-11 flex-1 items-center justify-center rounded-xl px-4 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(0,0,0,0.16)] transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-background ${
                  tone === 'danger'
                    ? 'bg-error shadow-[0_4px_12px_rgba(220,38,38,0.25)] hover:bg-error/90 focus-visible:ring-error'
                    : 'bg-primary shadow-[0_4px_12px_rgba(70,114,53,0.22)] hover:bg-primary-hover focus-visible:ring-primary'
                }`}
              >
                {state.confirmLabel ?? 'Ya, lanjutkan'}
              </button>
            </div>
          </div>
        </div>
      )}
    </ConfirmContext.Provider>
  );
}

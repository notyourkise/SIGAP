'use client';

import React, { useCallback, useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { useRouter } from 'next/navigation';
import { Bell, CheckCheck, X, ArrowRight } from 'lucide-react';
import { apiRequest } from '@/lib/api';
import { useWebSocket } from '@/context/WebSocketContext';
import { useToast } from './Toast';
import {
  type NotificationItem,
  typeStyles,
  getTargetRoute,
  formatNotifTime,
} from './NotificationBell';

const POLL_INTERVAL_MS = 60_000;
const NOTIFICATION_EVENT = 'NOTIFICATION_CREATED';
const MENU_LIST_LIMIT = 20;

interface NotificationsMenuProps {
  isCollapsed?: boolean;
}

export default function NotificationsMenu({ isCollapsed = false }: NotificationsMenuProps) {
  const router = useRouter();
  const { subscribe } = useWebSocket();
  const toast = useToast();

  const [unreadCount, setUnreadCount] = useState(0);
  const [notifications, setNotifications] = useState<NotificationItem[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isMarkingAll, setIsMarkingAll] = useState(false);
  const [isListOpen, setIsListOpen] = useState(false);
  const [activeItem, setActiveItem] = useState<NotificationItem | null>(null);

  const fetchUnreadCount = useCallback(async () => {
    try {
      const res = await apiRequest<{ unread_count: number }>('/notifications/unread-count');
      setUnreadCount(res.data?.unread_count ?? 0);
    } catch {
      // Jaringan offline / sesi tidak valid — polling berikutnya mencoba lagi.
    }
  }, []);

  const fetchNotifications = useCallback(async () => {
    setIsLoading(true);
    try {
      const res = await apiRequest<{ rows: NotificationItem[] }>(
        `/notifications?limit=${MENU_LIST_LIMIT}`
      );
      setNotifications(res.data?.rows ?? []);
    } catch {
      // Tetap tampilkan data lama bila refresh gagal.
    } finally {
      setIsLoading(false);
    }
  }, []);

  const refreshAll = useCallback(() => {
    fetchNotifications();
    fetchUnreadCount();
  }, [fetchNotifications, fetchUnreadCount]);

  // Sinkronisasi real-time + fallback polling berkala (60 detik).
  useEffect(() => {
    const unsubscribe = subscribe(NOTIFICATION_EVENT, () => refreshAll());
    const interval = setInterval(() => {
      fetchUnreadCount();
    }, POLL_INTERVAL_MS);

    return () => {
      unsubscribe();
      clearInterval(interval);
    };
  }, [subscribe, refreshAll, fetchUnreadCount]);

  // Data segar setiap kali modal daftar dibuka.
  useEffect(() => {
    if (!isListOpen) return;
    refreshAll();
  }, [isListOpen, refreshAll]);

  const handleMarkAllRead = async () => {
    setIsMarkingAll(true);
    try {
      await apiRequest('/notifications/read-all', { method: 'PUT' });
      setNotifications((list) => list.map((n) => ({ ...n, is_read: true })));
      setUnreadCount(0);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Gagal menandai semua notifikasi');
    } finally {
      setIsMarkingAll(false);
    }
  };

  const markItemRead = (item: NotificationItem) => {
    if (item.is_read) return;
    apiRequest(`/notifications/${item.id}/read`, { method: 'PUT' }).catch(() => undefined);
    setUnreadCount((count) => Math.max(0, count - 1));
    setNotifications((list) => list.map((n) => (n.id === item.id ? { ...n, is_read: true } : n)));
    setActiveItem((current) => (current && current.id === item.id ? { ...current, is_read: true } : current));
  };

  const handleItemClick = (item: NotificationItem) => {
    markItemRead(item);
    setActiveItem(item);
    setIsListOpen(false);
  };

  const handleViewDetail = (item: NotificationItem) => {
    setIsListOpen(false);
    setActiveItem(null);
    router.push(getTargetRoute(item));
  };

  const formatFullTime = (iso: string) => {
    const date = new Date(iso);
    if (Number.isNaN(date.getTime())) return iso;
    return date.toLocaleString('id-ID', {
      weekday: 'long',
      day: 'numeric',
      month: 'long',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const navItemClass =
    'flex min-h-11 w-full items-center gap-3.5 rounded-xl px-4 py-3 text-sm font-medium transition-[background-color,color,box-shadow,border-color] text-muted hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-card';

  return (
    <>
      <button
        type="button"
        onClick={() => setIsListOpen(true)}
        title={isCollapsed ? 'Pemberitahuan' : undefined}
        aria-label={unreadCount > 0 ? `Pemberitahuan, ${unreadCount} belum dibaca` : 'Pemberitahuan'}
        aria-expanded={isListOpen}
        className={`${navItemClass} ${isCollapsed ? 'md:justify-center md:px-0' : ''}`}
      >
        <span className="relative shrink-0">
          <Bell className="h-5 w-5" />
          {unreadCount > 0 && (
            <span className="absolute -right-2 -top-1.5 flex h-4.5 min-w-4.5 items-center justify-center rounded-full bg-red-600 px-1 text-[10px] font-bold leading-4.5 text-white shadow-[0_2px_8px_rgba(220,38,38,0.4)]">
              {unreadCount > 99 ? '99+' : unreadCount}
            </span>
          )}
        </span>
        <span className={isCollapsed ? 'md:hidden' : ''}>Pemberitahuan</span>
      </button>

      {/* Modal daftar pemberitahuan */}
      {isListOpen &&
        createPortal(
          <div className="fixed inset-0 z-50 flex items-center justify-center px-4">
            <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => setIsListOpen(false)} />

            <div
              role="dialog"
              aria-modal="true"
              aria-labelledby="notifications-menu-title"
              className="z-10 flex max-h-[90vh] w-full max-w-md flex-col overflow-hidden rounded-2xl border border-border bg-card shadow-2xl"
            >
              <div className="flex items-center justify-between gap-4 border-b border-border px-5 py-4">
                <div>
                  <h3 id="notifications-menu-title" className="text-lg font-bold tracking-[-0.015em]">Pemberitahuan</h3>
                  <p className="mt-0.5 text-xs text-muted">{unreadCount} belum dibaca</p>
                </div>
                <div className="flex items-center gap-1">
                  <button
                    type="button"
                    onClick={handleMarkAllRead}
                    disabled={isMarkingAll || unreadCount === 0}
                    className="flex items-center gap-1.5 rounded-lg px-2.5 py-2 text-xs font-semibold text-primary transition-colors hover:bg-background disabled:cursor-not-allowed disabled:opacity-40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                  >
                    <CheckCheck className="h-3.5 w-3.5" />
                    Tandai semua dibaca
                  </button>
                  <button
                    type="button"
                    aria-label="Tutup daftar pemberitahuan"
                    onClick={() => setIsListOpen(false)}
                    className="shrink-0 rounded-lg p-2.5 text-muted transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              </div>

              <ul className="min-h-40 flex-1 overflow-y-auto">
                {isLoading && notifications.length === 0 ? (
                  <li className="flex items-center justify-center gap-2 px-4 py-12 text-sm text-muted">
                    <span className="h-4 w-4 animate-spin rounded-full border-2 border-muted border-t-primary" />
                    Memuat pemberitahuan…
                  </li>
                ) : notifications.length === 0 ? (
                  <li className="px-4 py-12 text-center text-sm text-muted">
                    Belum ada pemberitahuan
                  </li>
                ) : (
                  notifications.map((item) => (
                    <li key={item.id}>
                      <button
                        type="button"
                        onClick={() => handleItemClick(item)}
                        className="flex w-full items-start gap-3 border-b border-border/60 px-5 py-3.5 text-left transition-colors hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary"
                      >
                        <span
                          className={`mt-1.5 h-2 w-2 shrink-0 rounded-full ${typeStyles[item.type] ?? 'bg-muted'}`}
                          aria-hidden="true"
                        />
                        <span className="min-w-0 flex-1">
                          <span className="flex items-baseline justify-between gap-2">
                            <span className={`truncate text-sm ${item.is_read ? 'font-medium text-muted' : 'font-bold text-foreground'}`}>
                              {item.title}
                            </span>
                            {!item.is_read && (
                              <span className="shrink-0 rounded-md bg-primary/10 px-1.5 py-0.5 text-[10px] font-bold uppercase text-primary">
                                Baru
                              </span>
                            )}
                          </span>
                          <span className="mt-0.5 line-clamp-2 block text-xs leading-relaxed text-muted">
                            {item.message}
                          </span>
                          <span className="mt-1 block text-[10px] text-muted/70">{formatNotifTime(item.created_at)}</span>
                        </span>
                        <ArrowRight className="mt-1 h-4 w-4 shrink-0 text-muted/50" />
                      </button>
                    </li>
                  ))
                )}
              </ul>

              <div className="border-t border-border px-5 py-3">
                <button
                  type="button"
                  onClick={() => {
                    setIsListOpen(false);
                    router.push('/pemberitahuan');
                  }}
                  className="flex w-full items-center justify-center gap-2 rounded-xl border border-border bg-background px-3 py-2.5 text-xs font-bold text-primary transition-colors hover:border-primary/40 hover:bg-primary/5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                >
                  Lihat riwayat lengkap
                </button>
              </div>
            </div>
          </div>,
          document.body
        )}

      {/* Modal informasi notifikasi */}
      {activeItem &&
        createPortal(
          <div className="fixed inset-0 z-50 flex items-center justify-center px-4">
            <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={() => setActiveItem(null)} />

            <div
              role="dialog"
              aria-modal="true"
              aria-labelledby="notification-info-title"
              className="z-10 w-full max-w-md overflow-hidden rounded-2xl border border-border bg-card shadow-2xl"
            >
              <div className="flex items-start justify-between gap-4 border-b border-border px-5 py-4">
                <div className="min-w-0">
                  <h3 id="notification-info-title" className="text-lg font-bold tracking-[-0.015em]">
                    Informasi Pemberitahuan
                  </h3>
                  <p className="mt-0.5 truncate text-xs text-muted">{formatFullTime(activeItem.created_at)}</p>
                </div>
                <button
                  type="button"
                  aria-label="Tutup informasi pemberitahuan"
                  onClick={() => setActiveItem(null)}
                  className="shrink-0 rounded-lg p-2.5 text-muted transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>

              <div className="space-y-4 px-5 py-5">
                <div className="flex items-start gap-3">
                  <span className={`mt-1.5 h-2.5 w-2.5 shrink-0 rounded-full ${typeStyles[activeItem.type] ?? 'bg-muted'}`} aria-hidden="true" />
                  <div className="min-w-0">
                    <h4 className="text-base font-bold leading-snug">{activeItem.title}</h4>
                    <p className="mt-1.5 whitespace-pre-wrap text-sm leading-relaxed text-muted">
                      {activeItem.message}
                    </p>
                  </div>
                </div>
              </div>

              <div className="flex items-center justify-between gap-3 border-t border-border px-5 py-4">
                <span className="text-[11px] font-semibold uppercase tracking-wider text-muted">
                  {activeItem.reference_type === 'field-maintenance' ? 'Pengecekan lapangan' : 'Pemeliharaan CCTV'}
                </span>
                <button
                  type="button"
                  onClick={() => handleViewDetail(activeItem)}
                  className="inline-flex items-center gap-2 rounded-xl bg-primary px-4 py-2.5 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(15,118,74,0.16)] transition-colors hover:bg-primary-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-card"
                >
                  Lihat detail
                  <ArrowRight className="h-4 w-4" />
                </button>
              </div>
            </div>
          </div>,
          document.body
        )}
    </>
  );
}

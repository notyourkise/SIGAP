'use client';

import React, { useEffect, useRef, useState, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { Bell, CheckCheck } from 'lucide-react';
import { apiRequest } from '@/lib/api';
import { useWebSocket } from '@/context/WebSocketContext';
import { useToast } from './Toast';

export interface NotificationItem {
  id: string;
  type: string;
  title: string;
  message: string;
  reference_id?: string | null;
  reference_type?: string | null;
  is_read: boolean;
  created_at: string;
}

const POLL_INTERVAL_MS = 20_000;
const NOTIFICATION_EVENT = 'NOTIFICATION_CREATED';
// Preview dropdown hanya menampilkan 3 notifikasi terbaru; sisanya diakses
// lewat halaman /pemberitahuan melalui tombol "Lihat lainnya".
const BELL_PREVIEW_LIMIT = 3;

export const typeStyles: Record<string, string> = {
  MAINTENANCE_ASSIGNED: 'bg-primary',
  MAINTENANCE_REASSIGNED: 'bg-primary',
  MAINTENANCE_UNASSIGNED: 'bg-blue-500',
  MAINTENANCE_COMPLETED: 'bg-success',
  MAINTENANCE_OVERDUE: 'bg-red-500',
  OVERDUE_CRITICAL: 'bg-red-600',
  FIELD_CHECK_ATTENTION: 'bg-accent',
  DAILY_DIGEST_AVAILABLE: 'bg-primary',
  ZONE_ASSIGNED: 'bg-accent',
  SCHEDULE_ASSIGNED: 'bg-primary',
};

// Tipe notifikasi kritis (overdue level 3) — ditampilkan dengan latar merah + ikon 🚨.
export const isCriticalType = (type: string) => type === 'OVERDUE_CRITICAL';

// Peta tipe notifikasi ke rute halaman tujuan saat item diklik.
export function getTargetRoute(item: NotificationItem): string {
  if (item.reference_type === 'field-maintenance') return '/field-maintenance';
  if (item.reference_type === 'daily-digest') return '/ringkasan';
  if (item.reference_type === 'zone') return '/sebaran-cctv';
  if (item.reference_type === 'maintenance-schedule') return '/dashboard';
  return '/maintenance';
}

export function formatNotifTime(iso: string): string {
  const date = new Date(iso);
  return date.toLocaleString('id-ID', {
    day: '2-digit',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export default function NotificationBell() {
  const router = useRouter();
  const { subscribe } = useWebSocket();
  const toast = useToast();
  const [isOpen, setIsOpen] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);
  const [notifications, setNotifications] = useState<NotificationItem[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isMarkingAll, setIsMarkingAll] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const fetchUnreadCount = useCallback(async () => {
    try {
      const res = await apiRequest<{ unread_count: number }>('/notifications/unread-count');
      setUnreadCount(res.data?.unread_count ?? 0);
    } catch {
      // Jaringan offline / sesi tidak valid — abaikan, polling berikutnya mencoba lagi.
    }
  }, []);

  const fetchNotifications = useCallback(async () => {
    setIsLoading(true);
    try {
      // Konvensi respons list SIGAP: data = { rows, pagination }
      const res = await apiRequest<{ rows: NotificationItem[] }>(
        `/notifications?limit=${BELL_PREVIEW_LIMIT}`
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

  // Muat jumlah belum dibaca segera saat halaman pertama kali dibuka, agar badge
  // lonceng langsung tampil tanpa harus menunggu polling atau diklik dulu.
  useEffect(() => {
    fetchUnreadCount();
  }, [fetchUnreadCount]);

  // Sinkronisasi real-time: event NOTIFICATION_CREATED dari server.
  useEffect(() => {
    const unsubscribe = subscribe(NOTIFICATION_EVENT, (payload: NotificationItem | undefined) => {
      if (!payload) return;
      toast.info(payload.title || 'Notifikasi baru');
      refreshAll();
    });

    // Fallback polling berkala (60 detik) untuk menutup celah saat koneksi
    // WebSocket sedang terputus atau event tertahan.
    const interval = setInterval(() => {
      fetchUnreadCount();
    }, POLL_INTERVAL_MS);

    return () => {
      unsubscribe();
      clearInterval(interval);
    };
  }, [subscribe, toast, refreshAll, fetchUnreadCount]);

  // Muat ulang saat dropdown dibuka agar data selalu segar.
  useEffect(() => {
    if (!isOpen) return;
    refreshAll();
  }, [isOpen, refreshAll]);

  // Tutup dropdown saat klik di luar atau tombol Escape.
  useEffect(() => {
    if (!isOpen) return;

    const handlePointerDown = (event: PointerEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setIsOpen(false);
    };

    document.addEventListener('pointerdown', handlePointerDown);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('pointerdown', handlePointerDown);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [isOpen]);

  const handleOpenToggle = () => {
    const next = !isOpen;
    setIsOpen(next);
    if (next) {
      fetchUnreadCount();
    }
  };

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

  const handleItemClick = async (item: NotificationItem) => {
    setIsOpen(false);
    if (!item.is_read) {
      // Fire-and-forget: tandai dibaca tanpa memblokir navigasi.
      apiRequest(`/notifications/${item.id}/read`, { method: 'PUT' }).catch(() => undefined);
      setUnreadCount((count) => Math.max(0, count - 1));
      setNotifications((list) => list.map((n) => (n.id === item.id ? { ...n, is_read: true } : n)));
    }
    router.push(getTargetRoute(item));
  };

  const formatTime = formatNotifTime;

  return (
    <div ref={dropdownRef} className="relative">
      <button
        type="button"
        onClick={handleOpenToggle}
        aria-label={unreadCount > 0 ? `${unreadCount} notifikasi belum dibaca` : 'Notifikasi'}
        aria-expanded={isOpen}
        className="relative flex h-10 w-10 items-center justify-center rounded-xl border border-border bg-background text-muted transition-colors hover:border-primary/30 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-card"
      >
        <Bell className="h-4 w-4" />
        {unreadCount > 0 && (
          <span className="absolute -right-1.5 -top-1.5 flex h-5 min-w-5 items-center justify-center rounded-full bg-red-600 px-1 text-[10px] font-bold text-white shadow-[0_2px_8px_rgba(220,38,38,0.4)]">
            {unreadCount > 99 ? '99+' : unreadCount}
          </span>
        )}
      </button>

      {isOpen && (
        <div className="absolute right-0 z-50 mt-2 w-80 overflow-hidden rounded-2xl border border-border bg-card shadow-[0_16px_40px_rgba(15,23,42,0.2)] sm:w-96">
          <div className="flex items-center justify-between border-b border-border px-4 py-3">
            <h3 className="text-sm font-bold">Notifikasi</h3>
            <button
              type="button"
              onClick={handleMarkAllRead}
              disabled={isMarkingAll || unreadCount === 0}
              className="flex items-center gap-1.5 rounded-lg px-2 py-1 text-xs font-semibold text-primary transition-colors hover:bg-background disabled:cursor-not-allowed disabled:opacity-40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            >
              <CheckCheck className="h-3.5 w-3.5" />
              Tandai semua dibaca
            </button>
          </div>

          <ul className="max-h-80 overflow-y-auto">
            {isLoading && notifications.length === 0 ? (
              <li className="flex items-center justify-center gap-2 px-4 py-10 text-sm text-muted">
                <span className="h-4 w-4 animate-spin rounded-full border-2 border-muted border-t-primary" />
                Memuat notifikasi...
              </li>
            ) : notifications.length === 0 ? (
              <li className="px-4 py-10 text-center text-sm text-muted">
                Belum ada notifikasi
              </li>
            ) : (
              notifications.map((item) => (
                <li key={item.id}>
                  <button
                    type="button"
                    onClick={() => handleItemClick(item)}
                    className={`flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary ${isCriticalType(item.type) ? 'bg-red-500/10' : ''}`}
                  >
                    <span
                      className={`mt-1.5 h-2 w-2 shrink-0 rounded-full ${typeStyles[item.type] ?? 'bg-muted'}`}
                      aria-hidden="true"
                    />
                    <span className="min-w-0 flex-1">
                      <span className="flex items-baseline justify-between gap-2">
                        <span className={`truncate text-sm ${item.is_read ? 'font-medium text-muted' : 'font-bold text-foreground'}`}>
                          {isCriticalType(item.type) && <span className="mr-1" aria-hidden="true">🚨</span>}
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
                      <span className="mt-1 block text-[10px] text-muted/70">{formatTime(item.created_at)}</span>
                    </span>
                  </button>
                </li>
              ))
            )}
          </ul>

          <div className="border-t border-border px-3 py-2.5">
            <button
              type="button"
              onClick={() => {
                setIsOpen(false);
                router.push('/pemberitahuan');
              }}
              className="flex w-full items-center justify-center gap-2 rounded-xl border border-border bg-background px-3 py-2.5 text-xs font-bold text-primary transition-colors hover:border-primary/40 hover:bg-primary/5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            >
              Lihat lainnya
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

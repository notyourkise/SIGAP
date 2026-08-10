'use client';

import { useCallback, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/DashboardLayout';
import Pagination from '@/components/Pagination';
import { useWebSocket } from '@/context/WebSocketContext';
import { useToast } from '@/components/Toast';
import { apiRequest } from '@/lib/api';
import { Bell, CheckCheck } from 'lucide-react';

interface NotificationItem {
  id: string;
  type: string;
  title: string;
  message: string;
  reference_id?: string | null;
  reference_type?: string | null;
  is_read: boolean;
  created_at: string;
}

interface NotificationsResponse {
  rows: NotificationItem[];
  pagination: {
    total_rows: number;
    total_pages: number;
    page: number;
    limit: number;
  };
}

const NOTIFICATION_EVENT = 'NOTIFICATION_CREATED';
const PAGE_SIZE = 10;

const typeStyles: Record<string, string> = {
  MAINTENANCE_ASSIGNED: 'bg-primary',
  MAINTENANCE_REASSIGNED: 'bg-primary',
  MAINTENANCE_UNASSIGNED: 'bg-blue-500',
  MAINTENANCE_COMPLETED: 'bg-success',
  MAINTENANCE_OVERDUE: 'bg-red-500',
  OVERDUE_CRITICAL: 'bg-red-600',
  FIELD_CHECK_ATTENTION: 'bg-accent',
  DAILY_DIGEST_AVAILABLE: 'bg-primary',
};

// Tipe notifikasi kritis (overdue level 3) — latar merah + ikon 🚨.
const isCriticalType = (type: string) => type === 'OVERDUE_CRITICAL';

// Halaman Pemberitahuan menampilkan notifikasi milik user yang login saja —
// backend mem-scope seluruh endpoint /api/notifications dari klaim JWT,
// sehingga teknisi A tidak akan pernah melihat notifikasi teknisi B.
function getTargetRoute(item: NotificationItem): string {
  if (item.reference_type === 'field-maintenance') return '/field-maintenance';
  if (item.reference_type === 'daily-digest') return '/ringkasan';
  return '/maintenance';
}

export default function NotificationsPage() {
  const router = useRouter();
  const toast = useToast();
  const { subscribe } = useWebSocket();
  const [notifications, setNotifications] = useState<NotificationItem[]>([]);
  const [totalRows, setTotalRows] = useState(0);
  const [totalPages, setTotalPages] = useState(1);
  const [page, setPage] = useState(1);
  const [onlyUnread, setOnlyUnread] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [isMarkingAll, setIsMarkingAll] = useState(false);
  // Jumlah total pemberitahuan yang belum dibaca (untuk ditampilkan di header).
  const [unreadCount, setUnreadCount] = useState(0);

  const fetchUnreadCount = useCallback(async () => {
    try {
      const res = await apiRequest<{ unread_count: number }>('/notifications/unread-count');
      setUnreadCount(res.data?.unread_count ?? 0);
    } catch {
      // Abaikan; badge di sidebar tetap yang jadi acuan utama.
    }
  }, []);

  useEffect(() => {
    fetchUnreadCount();
  }, [fetchUnreadCount]);

  const fetchNotifications = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const query = onlyUnread ? '&unread=true' : '';
      const res = await apiRequest<NotificationsResponse>(
        `/notifications?page=${page}&limit=${PAGE_SIZE}${query}`
      );
      setNotifications(res.data?.rows ?? []);
      setTotalRows(res.data?.pagination?.total_rows ?? 0);
      setTotalPages(res.data?.pagination?.total_pages ?? 1);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gagal memuat pemberitahuan');
    } finally {
      setLoading(false);
    }
  }, [page, onlyUnread]);

  useEffect(() => {
    const timer = setTimeout(fetchNotifications, 0);
    return () => clearTimeout(timer);
  }, [fetchNotifications]);

  // Perbarui daftar secara real-time saat notifikasi baru masuk (milik user ini saja).
  useEffect(() => {
    const unsubscribe = subscribe(NOTIFICATION_EVENT, () => {
      fetchNotifications();
      fetchUnreadCount();
    });
    return unsubscribe;
  }, [subscribe, fetchNotifications, fetchUnreadCount]);

  const handleMarkAllRead = async () => {
    setIsMarkingAll(true);
    try {
      await apiRequest('/notifications/read-all', { method: 'PUT' });
      setNotifications((list) => list.map((n) => ({ ...n, is_read: true })));
      setUnreadCount(0);
      if (onlyUnread) {
        setOnlyUnread(false);
      } else {
        fetchNotifications();
      }
      toast.success('Semua pemberitahuan ditandai dibaca');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Gagal menandai semua pemberitahuan');
    } finally {
      setIsMarkingAll(false);
    }
  };

  const handleItemClick = async (item: NotificationItem) => {
    if (!item.is_read) {
      apiRequest(`/notifications/${item.id}/read`, { method: 'PUT' }).catch(() => undefined);
      setNotifications((list) => list.map((n) => (n.id === item.id ? { ...n, is_read: true } : n)));
    }
    router.push(getTargetRoute(item));
  };

  const formatTime = (iso: string) => {
    const date = new Date(iso);
    return date.toLocaleString('id-ID', {
      day: '2-digit',
      month: 'long',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const unreadVisible = notifications.filter((n) => !n.is_read).length;

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h1 className="flex items-center gap-2.5 text-3xl font-bold tracking-[-0.025em]">
              Pemberitahuan
              {unreadCount > 0 && (
                <span className="inline-flex min-w-7 items-center justify-center rounded-full bg-red-600 px-2 py-0.5 text-xs font-bold leading-5 text-white shadow-[0_2px_8px_rgba(220,38,38,0.4)]">
                  {unreadCount > 99 ? '99+' : unreadCount}
                </span>
              )}
            </h1>
            <p className="mt-1 max-w-2xl text-sm text-muted">
              Riwayat penugasan, penyelesaian tugas, dan peringatan yang ditujukan untuk Anda.
            </p>
          </div>
          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={() => {
                setPage(1);
                setOnlyUnread((current) => !current);
              }}
              aria-pressed={onlyUnread}
              className={`rounded-xl border px-3 py-2 text-xs font-semibold transition-colors ${
                onlyUnread
                  ? 'border-primary bg-primary text-white'
                  : 'border-border bg-card text-muted hover:border-primary/40 hover:text-foreground'
              }`}
            >
              Belum dibaca saja
            </button>
            <button
              type="button"
              onClick={handleMarkAllRead}
              disabled={isMarkingAll || unreadVisible === 0}
              className="flex items-center gap-1.5 rounded-xl border border-border bg-card px-3 py-2 text-xs font-semibold text-primary transition-colors hover:border-primary/40 hover:bg-primary/5 disabled:cursor-not-allowed disabled:opacity-40"
            >
              <CheckCheck className="h-3.5 w-3.5" />
              {isMarkingAll ? 'Memproses…' : 'Tandai semua dibaca'}
            </button>
          </div>
        </header>

        {error && (
          <div role="alert" className="rounded-xl border border-error/25 bg-error/10 px-4 py-3 text-sm text-error">
            {error}
          </div>
        )}

        <section aria-label="Daftar pemberitahuan" className="overflow-hidden rounded-2xl border border-border bg-card">
          {loading ? (
            <div className="flex items-center justify-center gap-2 px-4 py-16 text-sm text-muted">
              <span className="h-4 w-4 animate-spin rounded-full border-2 border-muted border-t-primary" />
              Memuat pemberitahuan...
            </div>
          ) : notifications.length === 0 ? (
            <div className="flex flex-col items-center gap-3 px-4 py-16 text-center">
              <span className="flex h-14 w-14 items-center justify-center rounded-full bg-background">
                <Bell className="h-6 w-6 text-muted" />
              </span>
              <div>
                <h2 className="text-sm font-bold">Belum ada pemberitahuan</h2>
                <p className="mt-1 max-w-sm text-xs text-muted">
                  {onlyUnread
                    ? 'Semua pemberitahuan Anda sudah dibaca. Filter ini hanya menampilkan yang belum dibaca.'
                    : 'Penugasan, penyelesaian tugas, dan peringatan baru akan muncul di sini.'}
                </p>
              </div>
            </div>
          ) : (
            <>
              <ul className="divide-y divide-border">
                {notifications.map((item) => (
                  <li key={item.id}>
                    <button
                      type="button"
                      onClick={() => handleItemClick(item)}
                      className={`flex w-full items-start gap-3.5 border-l-4 px-5 py-4 text-left transition-colors hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary ${
                        isCriticalType(item.type)
                          ? 'border-red-600 bg-red-500/10'
                          : item.is_read
                            ? 'border-transparent'
                            : 'border-primary bg-primary/5'
                      }`}
                    >
                      <span
                        className={`mt-1.5 h-2.5 w-2.5 shrink-0 rounded-full ${typeStyles[item.type] ?? 'bg-muted'}`}
                        aria-hidden="true"
                      />
                      <span className="min-w-0 flex-1">
                        <span className="flex items-baseline justify-between gap-3">
                          <span className={`truncate text-sm ${item.is_read ? 'font-medium text-muted' : 'font-bold text-foreground'}`}>
                            {isCriticalType(item.type) && <span className="mr-1" aria-hidden="true">🚨</span>}
                            {item.title}
                          </span>
                          {!item.is_read && (
                            <span className="shrink-0 rounded-md bg-red-600/10 px-2 py-0.5 text-[10px] font-bold uppercase text-red-600">
                              Baru
                            </span>
                          )}
                        </span>
                        <span className={`mt-1 block text-sm leading-relaxed ${item.is_read ? 'text-muted' : 'text-foreground'}`}>{item.message}</span>
                        <span className="mt-1.5 block text-[11px] text-muted/70">{formatTime(item.created_at)}</span>
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
              <Pagination
                page={page}
                limit={PAGE_SIZE}
                totalRows={totalRows}
                totalPages={totalPages}
                onPageChange={setPage}
                itemLabel="pemberitahuan"
              />
            </>
          )}
        </section>
      </div>
    </DashboardLayout>
  );
}

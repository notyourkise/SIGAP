'use client';

import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { Bell } from 'lucide-react';
import { apiRequest } from '@/lib/api';
import { useWebSocket } from '@/context/WebSocketContext';

const POLL_INTERVAL_MS = 20_000;
const NOTIFICATION_EVENT = 'NOTIFICATION_CREATED';

interface NotificationsMenuProps {
  isCollapsed?: boolean;
}

// Item navigasi "Pemberitahuan" pada sidebar. Menampilkan jumlah belum dibaca
// sebagai badge, dan mengarahkan langsung ke halaman pemberitahuan (bukan modal).
export default function NotificationsMenu({ isCollapsed = false }: NotificationsMenuProps) {
  const { subscribe } = useWebSocket();
  const [unreadCount, setUnreadCount] = useState(0);

  const fetchUnreadCount = useCallback(async () => {
    try {
      const res = await apiRequest<{ unread_count: number }>('/notifications/unread-count');
      setUnreadCount(res.data?.unread_count ?? 0);
    } catch {
      // Jaringan offline / sesi tidak valid — polling berikutnya mencoba lagi.
    }
  }, []);

  // Muat jumlah belum dibaca segera saat sidebar pertama kali dirender, agar
  // badge menu "Pemberitahuan" langsung tampil tanpa harus menunggu polling.
  useEffect(() => {
    fetchUnreadCount();
  }, [fetchUnreadCount]);

  // Sinkronisasi real-time + fallback polling berkala (60 detik).
  useEffect(() => {
    const unsubscribe = subscribe(NOTIFICATION_EVENT, () => fetchUnreadCount());
    const interval = setInterval(() => {
      fetchUnreadCount();
    }, POLL_INTERVAL_MS);

    return () => {
      unsubscribe();
      clearInterval(interval);
    };
  }, [subscribe, fetchUnreadCount]);

  const navItemClass =
    'flex min-h-11 w-full items-center gap-3.5 rounded-xl px-4 py-3 text-sm font-medium transition-[background-color,color,box-shadow,border-color] text-muted hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-card';

  return (
    <Link
      href="/pemberitahuan"
      title={isCollapsed ? 'Pemberitahuan' : undefined}
      aria-label={unreadCount > 0 ? `Pemberitahuan, ${unreadCount} belum dibaca` : 'Pemberitahuan'}
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
    </Link>
  );
}

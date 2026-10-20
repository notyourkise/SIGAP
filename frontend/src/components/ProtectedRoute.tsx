'use client';

import React, { useEffect } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { useAuth } from '@/context/AuthContext';

export default function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, loading, user } = useAuth();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (loading) return;
    if (!isAuthenticated && pathname !== '/login') {
      router.replace('/login');
      return;
    }
    if (user?.must_change_password && pathname !== '/change-password') {
      router.replace('/change-password');
    }
  }, [isAuthenticated, loading, pathname, router, user?.must_change_password]);

  if (loading) {
    return (
      <div className="flex h-screen w-screen items-center justify-center gap-3 bg-background" role="status">
        <div className="h-10 w-10 animate-spin rounded-full border-4 border-primary border-t-transparent"></div>
        <span className="text-sm text-muted">Memeriksa sesi pengguna…</span>
      </div>
    );
  }

  if (!isAuthenticated && pathname !== '/login') {
    return (
      <div className="flex h-screen w-screen items-center justify-center gap-3 bg-background" role="status">
        <div className="h-10 w-10 animate-spin rounded-full border-4 border-primary border-t-transparent"></div>
        <span className="text-sm text-muted">Mengarahkan ke halaman masuk…</span>
      </div>
    );
  }

  if (user?.must_change_password && pathname !== '/change-password') {
    return (
      <div className="flex h-screen w-screen items-center justify-center gap-3 bg-background" role="status" aria-live="polite">
        <div className="h-10 w-10 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        <span className="text-sm text-muted">Mengarahkan ke pengamanan akun…</span>
      </div>
    );
  }

  return <>{children}</>;
}

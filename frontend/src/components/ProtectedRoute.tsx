'use client';

import React, { useEffect } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { useAuth } from '@/context/AuthContext';

export default function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, loading } = useAuth();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (!loading && !isAuthenticated && pathname !== '/login') {
      router.push('/login');
    }
  }, [isAuthenticated, loading, pathname, router]);

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

  return <>{children}</>;
}

'use client';

import React, { useEffect, useState, useSyncExternalStore } from 'react';
import { usePathname } from 'next/navigation';
import { useAuth } from '@/context/AuthContext';
import { useTheme } from '@/context/ThemeContext';
import ProtectedRoute from './ProtectedRoute';
import {
  Wrench, 
  LogOut, 
  Menu, 
  X, 
  User as UserIcon, 
  Sun, 
  Moon,
  Database,
  LayoutDashboard,
  MapPinned,
  Users,
  PanelLeftClose,
  PanelLeftOpen,
  ShieldCheck,
  ClipboardCheck,
  Bell
} from 'lucide-react';
import Link from 'next/link';

const SIDEBAR_STORAGE_KEY = 'sigap_sidebar_collapsed';
const SIDEBAR_CHANGE_EVENT = 'sigap-sidebar-change';

function subscribeToSidebarPreference(callback: () => void) {
  window.addEventListener('storage', callback);
  window.addEventListener(SIDEBAR_CHANGE_EVENT, callback);

  return () => {
    window.removeEventListener('storage', callback);
    window.removeEventListener(SIDEBAR_CHANGE_EVENT, callback);
  };
}

function getSidebarPreference() {
  return localStorage.getItem(SIDEBAR_STORAGE_KEY) === 'true';
}

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { user, logout } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const pathname = usePathname();
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const [isLogoutConfirmOpen, setIsLogoutConfirmOpen] = useState(false);
  const isSidebarCollapsed = useSyncExternalStore(
    subscribeToSidebarPreference,
    getSidebarPreference,
    () => false,
  );

  useEffect(() => {
    if (!isSidebarOpen) return;

    const previousOverflow = document.body.style.overflow;
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setIsSidebarOpen(false);
    };

    document.body.style.overflow = 'hidden';
    document.addEventListener('keydown', closeOnEscape);

    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener('keydown', closeOnEscape);
    };
  }, [isSidebarOpen]);

  useEffect(() => {
    if (!isLogoutConfirmOpen) return;

    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setIsLogoutConfirmOpen(false);
    };

    document.addEventListener('keydown', closeOnEscape);

    return () => {
      document.removeEventListener('keydown', closeOnEscape);
    };
  }, [isLogoutConfirmOpen]);

  const navigation = [
    { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
    { name: 'Peta Sebaran CCTV', href: '/sebaran-cctv', icon: MapPinned },
    { name: 'Aset CCTV', href: '/cctv', icon: Database },
    { name: 'Pemeliharaan CCTV', href: '/maintenance', icon: Wrench },
    ...(user?.role === 'admin'
      ? [
          { name: 'Pengecekan Lapangan', href: '/field-maintenance', icon: ClipboardCheck },
          { name: 'Manajemen Pengguna', href: '/users', icon: Users },
          { name: 'Audit Log', href: '/audit-logs', icon: ShieldCheck },
        ]
      : []),
    // Pemberitahuan selalu di akhir menu, tepat di atas tombol Keluar
    { name: 'Pemberitahuan', href: '/pemberitahuan', icon: Bell },
  ];

  const handleLogout = () => {
    logout();
  };

  const confirmLogout = () => {
    setIsLogoutConfirmOpen(false);
    setIsSidebarOpen(false);
    handleLogout();
  };

  return (
    <ProtectedRoute>
      <div className="min-h-screen flex flex-col md:flex-row bg-background text-foreground transition-colors duration-300">
        
        {/* Mobile Header Bar */}
        <header className="sticky top-0 z-20 flex items-center justify-between border-b border-border bg-card px-5 py-3 md:hidden">
          <Link
            href="/dashboard"
            aria-label="Dashboard SIGAP"
            className="rounded-md text-xl font-black tracking-[-0.03em] text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-card"
          >
            SIGAP
          </Link>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={toggleTheme}
              aria-label={theme === 'light' ? 'Aktifkan mode gelap' : 'Aktifkan mode terang'}
              className="flex h-10 w-10 items-center justify-center rounded-xl border border-border bg-background text-muted transition-colors hover:border-primary/30 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-card"
            >
              {theme === 'light' ? <Moon className="h-4 w-4" /> : <Sun className="h-4 w-4 text-accent" />}
            </button>
            <button
              type="button"
              onClick={() => setIsSidebarOpen(!isSidebarOpen)}
              aria-label={isSidebarOpen ? 'Tutup menu navigasi' : 'Buka menu navigasi'}
              aria-controls="primary-sidebar"
              aria-expanded={isSidebarOpen}
              className="flex h-10 w-10 items-center justify-center rounded-xl border border-border bg-background text-foreground transition-colors hover:border-primary/30 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-card"
            >
              {isSidebarOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
            </button>
          </div>
        </header>

        {/* Sidebar overlay for mobile */}
        <button
          type="button"
          aria-label="Tutup menu navigasi"
          className={`fixed inset-0 z-30 bg-black/45 backdrop-blur-[2px] transition-opacity duration-500 ease-in-out md:hidden ${isSidebarOpen ? 'opacity-100' : 'pointer-events-none opacity-0'}`}
          onClick={() => setIsSidebarOpen(false)}
        />

        {/* Sidebar Container */}
        <aside
          id="primary-sidebar"
          aria-label="Sidebar aplikasi"
          className={`
          fixed left-0 top-0 z-40 flex h-screen flex-col justify-between border-r border-border bg-card transition-[width,transform,padding] duration-500 ease-in-out md:sticky
          ${isSidebarCollapsed ? 'w-72 p-5 md:w-[5.5rem] md:p-3' : 'w-72 p-5'}
          ${isSidebarOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}
        `}
        >
          <div className="min-h-0 flex-1 space-y-7 overflow-y-auto">
            <div className="flex items-center justify-between border-b border-border pb-4 md:hidden">
              <Link
                href="/dashboard"
                onClick={() => setIsSidebarOpen(false)}
                className="rounded-md text-xl font-black tracking-[-0.03em] text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
              >
                SIGAP
              </Link>
              <button
                type="button"
                onClick={() => setIsSidebarOpen(false)}
                aria-label="Tutup menu navigasi"
                className="flex h-10 w-10 items-center justify-center rounded-xl border border-border bg-background text-muted transition-colors hover:border-primary/30 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            {/* Identitas aplikasi tetap terlihat saat sidebar diciutkan */}
            <div className={`hidden items-center md:flex ${isSidebarCollapsed ? 'flex-col gap-3' : 'justify-between'}`}>
              <Link
                href="/dashboard"
                aria-label="Dashboard SIGAP"
                title={isSidebarCollapsed ? 'SIGAP' : undefined}
                className="flex min-h-10 items-center justify-center rounded-md text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-card"
              >
                <span className={`${isSidebarCollapsed ? 'text-sm' : 'text-2xl'} font-black tracking-[-0.03em]`}>SIGAP</span>
              </Link>
              <div className={`flex gap-2 ${isSidebarCollapsed ? 'flex-col' : ''}`}>
                <button
                  type="button"
                  onClick={(event) => {
                    event.stopPropagation();
                    toggleTheme();
                  }}
                  aria-label={theme === 'light' ? 'Aktifkan mode gelap' : 'Aktifkan mode terang'}
                  title={theme === 'light' ? 'Aktifkan mode gelap' : 'Aktifkan mode terang'}
                  className="flex h-10 w-10 items-center justify-center rounded-xl border border-border bg-background text-muted transition-colors hover:border-primary/30 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-card"
                >
                  {theme === 'light' ? <Moon className="h-4 w-4" /> : <Sun className="h-4 w-4 text-accent" />}
                </button>
                <button
                  type="button"
                  onClick={(event) => {
                    event.stopPropagation();
                    localStorage.setItem(SIDEBAR_STORAGE_KEY, String(!isSidebarCollapsed));
                    window.dispatchEvent(new Event(SIDEBAR_CHANGE_EVENT));
                  }}
                  aria-label={isSidebarCollapsed ? 'Buka sidebar' : 'Tutup sidebar'}
                  title={isSidebarCollapsed ? 'Buka sidebar' : 'Tutup sidebar'}
                  className="flex h-10 w-10 items-center justify-center rounded-xl border border-border bg-background text-muted transition-colors hover:border-primary/30 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-card"
                >
                  {isSidebarCollapsed ? <PanelLeftOpen className="h-4 w-4" /> : <PanelLeftClose className="h-4 w-4" />}
                </button>
              </div>
            </div>

            {/* User Profile Card */}
            {user && (
              <Link
                href="/profile"
                title={isSidebarCollapsed ? `${user.full_name} · ${user.role}` : undefined}
                className={`flex items-center gap-3 rounded-2xl border border-border bg-background transition-colors hover:border-primary/30 ${isSidebarCollapsed ? 'md:justify-center md:p-2.5' : 'p-3'}`}
              >
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-accent/20 text-[#775800] dark:text-accent">
                  <UserIcon className="h-5 w-5" />
                </div>
                <div className={`overflow-hidden ${isSidebarCollapsed ? 'md:hidden' : ''}`}>
                  <h4 className="truncate text-sm font-semibold">{user.full_name}</h4>
                  <span className={`mt-1 inline-block rounded-md border px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider ${
                    user.role === 'admin' 
                      ? 'bg-accent/20 text-[#775800] border border-accent/30 dark:text-accent'
                      : 'bg-success/10 text-success border border-success/20'
                  }`}>
                    {user.role}
                  </span>
                </div>
              </Link>
            )}

            {/* Navigation links */}
            <nav aria-label="Navigasi utama" className="space-y-1.5">
              <p className={`mb-2 px-3 text-xs font-semibold text-muted ${isSidebarCollapsed ? 'md:sr-only' : ''}`}>
                Menu utama
              </p>
              {navigation.map((item) => {
                const isActive = pathname === item.href;
                const IconComponent = item.icon;
                return (
                  <Link
                    key={item.name}
                    href={item.href}
                    onClick={() => setIsSidebarOpen(false)}
                    aria-current={isActive ? 'page' : undefined}
                    className={`
                      flex min-h-11 items-center gap-3.5 rounded-xl px-4 py-3 text-sm font-medium transition-[background-color,color,box-shadow,border-color]
                      ${isSidebarCollapsed ? 'md:justify-center md:px-0' : ''}
                      ${isActive 
                        ? 'bg-primary text-white shadow-[0_4px_12px_rgba(70,114,53,0.22)]'
                        : 'text-muted hover:bg-background hover:text-foreground'
                      }
                      focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-card
                    `}
                    title={isSidebarCollapsed ? item.name : undefined}
                  >
                    <IconComponent className="h-5 w-5 shrink-0" />
                    <span className={isSidebarCollapsed ? 'md:hidden' : ''}>{item.name}</span>
                  </Link>
                );
              })}
              <button
                type="button"
                onClick={() => setIsLogoutConfirmOpen(true)}
                title={isSidebarCollapsed ? 'Keluar' : undefined}
                aria-label="Keluar dari SIGAP"
                className={`
                  flex min-h-11 w-full items-center gap-3.5 rounded-xl px-4 py-3 text-sm font-medium transition-[background-color,color,box-shadow,border-color]
                  ${isSidebarCollapsed ? 'md:justify-center md:px-0' : ''}
                  text-muted hover:bg-background hover:text-foreground
                  focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-card
                `}
              >
                <LogOut className="h-5 w-5 shrink-0" />
                <span className={isSidebarCollapsed ? 'md:hidden' : ''}>Keluar</span>
              </button>
            </nav>
          </div>
        </aside>

        {/* Main Content Area */}
        <main className="flex-1 flex flex-col min-w-0 overflow-y-auto px-6 py-8 md:px-10">
          <div className="max-w-6xl w-full mx-auto">
            {children}
          </div>
        </main>
      </div>

      {/* Konfirmasi Keluar */}
      {isLogoutConfirmOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fadeIn" role="dialog" aria-modal="true" aria-labelledby="logout-confirm-heading">
          <div className="relative w-full max-w-sm space-y-5 rounded-2xl border border-border bg-card p-6 text-center shadow-2xl">
            <button
              type="button"
              onClick={() => setIsLogoutConfirmOpen(false)}
              aria-label="Tutup dialog konfirmasi"
              className="absolute right-3 top-3 rounded-lg p-1.5 text-muted transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            >
              <X className="h-4 w-4" />
            </button>

            <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-red-500/10">
              <LogOut className="h-6 w-6 text-red-600 dark:text-red-400" />
            </div>

            <div>
              <h2 id="logout-confirm-heading" className="text-lg font-bold">Keluar dari SIGAP?</h2>
              <p className="mt-1.5 text-sm text-muted">
                Anda akan keluar dari sesi saat ini dan perlu login kembali untuk mengakses aplikasi.
              </p>
            </div>

            <div className="flex flex-col gap-2 sm:flex-row">
              <button
                type="button"
                onClick={() => setIsLogoutConfirmOpen(false)}
                className="flex h-11 flex-1 items-center justify-center rounded-xl border border-border bg-card px-4 text-sm font-semibold text-foreground transition-colors hover:bg-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
              >
                Batal
              </button>
              <button
                type="button"
                onClick={confirmLogout}
                autoFocus
                className="flex h-11 flex-1 items-center justify-center gap-2 rounded-xl bg-red-600 px-4 text-sm font-semibold text-white shadow-[0_4px_12px_rgba(220,38,38,0.25)] transition-colors hover:bg-red-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
              >
                <LogOut className="h-4 w-4" />
                Keluar
              </button>
            </div>
          </div>
        </div>
      )}
    </ProtectedRoute>
  );
}

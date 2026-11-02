'use client';

import React, { useEffect, useState, useSyncExternalStore } from 'react';
import { usePathname } from 'next/navigation';
import { useAuth } from '@/context/AuthContext';
import { useTheme } from '@/context/ThemeContext';
import ProtectedRoute from './ProtectedRoute';
import {
  Activity,
  Wrench, 
  Menu, 
  X, 
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
  CalendarClock,
  Calendar,
  TrendingUp,
  ClipboardList,
  MonitorPlay,
  Package,
  FileSpreadsheet,
  Target,
  KeyRound,
  Webhook,
  ListChecks,
  Cctv,
  CalendarRange,
  BarChart3,
  Settings,
  ChevronDown,
  ChevronRight,
  Building2,
} from 'lucide-react';
import Link from 'next/link';
import NotificationsMenu from './NotificationsMenu';
import ProfileMenu from './ProfileMenu';
import SiteSwitcher from './SiteSwitcher';

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

// Deteksi breakpoint md (768px) untuk membedakan perilaku sidebar di desktop vs mobile.
function useIsDesktop() {
  const [isDesktop, setIsDesktop] = useState(false);
  useEffect(() => {
    const query = window.matchMedia('(min-width: 768px)');
    const update = () => setIsDesktop(query.matches);
    update();
    query.addEventListener('change', update);
    return () => query.removeEventListener('change', update);
  }, []);
  return isDesktop;
}

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { user } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const pathname = usePathname();
  const isDesktop = useIsDesktop();
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
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

  // Tipe untuk item menu & grup kategori
  type NavItem = { name: string; href: string; icon: typeof LayoutDashboard };
  type NavGroup = { label: string; icon: typeof LayoutDashboard; items: NavItem[] };

  const isAdmin = user?.role === 'admin';

  // Grup kategori menu. Dashboard dibiarkan di menu utama (langsung).
  const navGroups: NavGroup[] = [
    {
      label: 'CCTV & Monitoring',
      icon: Cctv,
      items: [
        { name: 'Peta Sebaran CCTV', href: '/sebaran-cctv', icon: MapPinned },
        { name: 'Log CCTV', href: '/cctv/logs', icon: Activity },
        { name: 'Aset CCTV', href: '/cctv', icon: Database },
        { name: 'Pemeliharaan CCTV', href: '/maintenance', icon: Wrench },
        { name: 'Asisten Operasional AI', href: '/ai-assistant', icon: Activity },
        ...(isAdmin ? [{ name: 'Tiket Gangguan', href: '/incidents', icon: Activity }] : []),
        ...(isAdmin ? [{ name: 'Monitor CCTV', href: '/monitor', icon: MonitorPlay }] : []),
      ],
    },
    {
      label: 'Operasional & Jadwal',
      icon: CalendarRange,
      items: [
        { name: 'Kalender', href: '/kalender', icon: Calendar },
        ...(isAdmin
          ? [
              { name: 'Jadwal', href: '/schedules', icon: CalendarClock },
              { name: 'Pengecekan Lapangan', href: '/field-maintenance', icon: ClipboardCheck },
              { name: 'Kinerja Teknisi', href: '/kinerja-teknisi', icon: TrendingUp },
            ]
          : []),
      ],
    },
    {
      label: 'Pelaporan & Analisis',
      icon: BarChart3,
      items: isAdmin
        ? [
            { name: 'Ringkasan', href: '/ringkasan', icon: ClipboardList },
            { name: 'Laporan', href: '/laporan-kustom', icon: FileSpreadsheet },
            { name: 'SLA', href: '/sla', icon: Target },
          ]
        : [],
    },
    {
      label: 'Manajemen Sistem',
      icon: Settings,
      items: isAdmin
        ? [
            { name: 'Manajemen Pengguna', href: '/users', icon: Users },
            { name: 'Kawasan', href: '/sites', icon: Building2 },
            { name: 'Audit Log', href: '/audit-logs', icon: ShieldCheck },
            { name: 'Suku Cadang', href: '/suku-cadang', icon: Package },
            { name: 'Zona', href: '/zona', icon: MapPinned },
            { name: 'API Keys', href: '/api-keys', icon: KeyRound },
            { name: 'Webhooks', href: '/webhooks', icon: Webhook },
            { name: 'Template Checklist', href: '/checklist-templates', icon: ListChecks },
          ]
        : [],
    },
  ].filter((group) => group.items.length > 0);

  // Grup yang sedang terbuka (accordion, mode sidebar normal/expanded)
  const [openGroups, setOpenGroups] = useState<string[]>(() => {
    if (typeof window === 'undefined') return [];
    const activeHref = window.location.pathname;
    return navGroups
      .filter((group) => group.items.some((item) => item.href === activeHref))
      .map((group) => group.label);
  });

  const toggleGroup = (label: string) => {
    setOpenGroups((prev) =>
      prev.includes(label) ? prev.filter((g) => g !== label) : [...prev, label]
    );
  };

  const isGroupOpen = (label: string) => openGroups.includes(label);

  // Grup yang popup menu-nya sedang terbuka (mode sidebar collapsed/icon-only)
  const [activeCollapsedGroup, setActiveCollapsedGroup] = useState<string | null>(null);

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
            <SiteSwitcher />
            <ProfileMenu showDetails={false} />
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

            {/* Navigation links */}
            <nav aria-label="Navigasi utama" className="space-y-1.5">
              <p className={`mb-2 px-3 text-xs font-semibold text-muted ${isSidebarCollapsed ? 'md:sr-only' : ''}`}>
                Menu utama
              </p>
              {/* Dashboard — selalu tampil langsung di menu utama */}
              {(() => {
                const item = { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard };
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
              })()}

              {/* Grup kategori menu (dropdown / accordion) */}
              <div className="space-y-1.5">
                {navGroups.map((group) => {
                  const GroupIcon = group.icon;
                  const open = isGroupOpen(group.label);
                  const groupHasActive = group.items.some((item) => pathname === item.href);
                  const isCollapsed = isSidebarCollapsed && isDesktop;
                  const popupOpen = activeCollapsedGroup === group.label;

                  return (
                    <div key={group.label} className="relative">
                      {/* Header grup */}
                      <button
                        type="button"
                        onClick={() => {
                          if (isCollapsed) {
                            // Mode icon-only: toggle popup dropdown di samping sidebar
                            setActiveCollapsedGroup(popupOpen ? null : group.label);
                          } else {
                            // Mode normal: buka/tutup accordion
                            toggleGroup(group.label);
                          }
                        }}
                        aria-expanded={isCollapsed ? popupOpen : open}
                        aria-haspopup={isCollapsed ? 'menu' : undefined}
                        title={isCollapsed ? group.label : undefined}
                        className={`
                          flex min-h-11 w-full items-center gap-3.5 rounded-xl px-4 py-3 text-sm font-medium transition-[background-color,color,box-shadow,border-color]
                          ${isCollapsed ? 'md:justify-center md:px-0' : ''}
                          ${groupHasActive || (isCollapsed && popupOpen)
                            ? 'text-primary'
                            : 'text-muted hover:bg-background hover:text-foreground'
                          }
                          focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-card
                        `}
                      >
                        <GroupIcon className="h-5 w-5 shrink-0" />
                        <span className={isCollapsed ? 'md:hidden' : 'flex-1 text-left'}>{group.label}</span>
                        <ChevronDown
                          className={`h-4 w-4 shrink-0 transition-transform duration-200 ${isCollapsed ? 'md:hidden' : ''} ${open ? 'rotate-180' : ''}`}
                        />
                      </button>

                      {/* Popup dropdown saat sidebar collapsed (desktop, icon-only) */}
                      {isCollapsed && popupOpen && (
                        <div
                          role="menu"
                          className="absolute left-full top-0 z-50 hidden w-56 origin-top-left rounded-2xl border border-border bg-card p-2 shadow-2xl md:block"
                        >
                          <p className="px-3 pb-2 pt-1 text-xs font-semibold text-muted">{group.label}</p>
                          <div className="space-y-1">
                            {group.items.map((item) => {
                              const isActive = pathname === item.href;
                              const IconComponent = item.icon;
                              return (
                                <Link
                                  key={item.name}
                                  href={item.href}
                                  role="menuitem"
                                  onClick={() => {
                                    setIsSidebarOpen(false);
                                    setActiveCollapsedGroup(null);
                                  }}
                                  aria-current={isActive ? 'page' : undefined}
                                  className={`
                                    flex min-h-10 items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-[background-color,color]
                                    ${isActive 
                                      ? 'bg-primary/10 text-primary'
                                      : 'text-muted hover:bg-background hover:text-foreground'
                                    }
                                    focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40
                                  `}
                                >
                                  <IconComponent className="h-4 w-4 shrink-0" />
                                  <span>{item.name}</span>
                                </Link>
                              );
                            })}
                          </div>
                        </div>
                      )}

                      {/* Accordion saat sidebar normal (expanded) */}
                      {!isCollapsed && open && (
                        <div className="ml-2 mt-1 space-y-1 border-l border-border pl-3">
                          {group.items.map((item) => {
                            const isActive = pathname === item.href;
                            const IconComponent = item.icon;
                            return (
                              <Link
                                key={item.name}
                                href={item.href}
                                onClick={() => setIsSidebarOpen(false)}
                                aria-current={isActive ? 'page' : undefined}
                                className={`
                                  flex min-h-10 items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-[background-color,color]
                                  ${isActive 
                                    ? 'bg-primary/10 text-primary'
                                    : 'text-muted hover:bg-background hover:text-foreground'
                                  }
                                  focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 focus-visible:ring-offset-card
                                `}
                              >
                                <IconComponent className="h-4 w-4 shrink-0" />
                                <span>{item.name}</span>
                              </Link>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>

              {/* Pemberitahuan selalu di akhir menu sidebar */}
              <NotificationsMenu isCollapsed={isSidebarCollapsed} />
            </nav>
          </div>
        </aside>

        {/* Main Content Area */}
        <main className="flex-1 flex flex-col min-w-0 overflow-y-auto px-6 py-8 md:px-10">
          <div className="max-w-6xl w-full mx-auto">
            {pathname !== '/dashboard' && (
              <div className="mb-6 hidden items-center justify-end gap-2 md:flex">
                <SiteSwitcher />
                <ProfileMenu />
              </div>
            )}
            {children}
          </div>
        </main>
      </div>
    </ProtectedRoute>
  );
}

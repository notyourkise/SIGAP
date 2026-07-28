'use client';

import React, { useEffect, useState } from 'react';
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
  PanelLeftClose,
  PanelLeftOpen
} from 'lucide-react';
import Link from 'next/link';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { user, logout } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const pathname = usePathname();
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);

  useEffect(() => {
    setIsSidebarCollapsed(localStorage.getItem('sigap_sidebar_collapsed') === 'true');
  }, []);

  const navigation = [
    { name: 'Aset CCTV', href: '/cctv', icon: Database },
    { name: 'Log Pemeliharaan', href: '/maintenance', icon: Wrench },
  ];

  const handleLogout = () => {
    logout();
  };

  return (
    <ProtectedRoute>
      <div className="min-h-screen flex flex-col md:flex-row bg-background text-foreground transition-colors duration-300">
        
        {/* Mobile Header Bar */}
        <header className="md:hidden flex items-center justify-between px-6 py-4 bg-card border-b border-border z-20">
          <Link href="/cctv" className="text-2xl font-black tracking-tight text-primary">
            SIGAP
          </Link>
          <div className="flex items-center gap-3">
            <button
              onClick={toggleTheme}
              className="p-2 rounded-lg border border-border bg-background hover:scale-105 active:scale-95 transition-all"
            >
              {theme === 'light' ? <Moon className="h-4 w-4 text-gray-700" /> : <Sun className="h-4 w-4 text-yellow-400" />}
            </button>
            <button
              onClick={() => setIsSidebarOpen(!isSidebarOpen)}
              className="p-2 rounded-lg border border-border bg-background"
            >
              {isSidebarOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
            </button>
          </div>
        </header>

        {/* Sidebar overlay for mobile */}
        {isSidebarOpen && (
          <div 
            className="md:hidden fixed inset-0 bg-black/50 z-30 backdrop-blur-sm"
            onClick={() => setIsSidebarOpen(false)}
          />
        )}

        {/* Sidebar Container */}
        <aside className={`
          fixed md:sticky top-0 left-0 h-screen bg-card border-r border-border flex flex-col justify-between z-40 transition-all duration-300
          ${isSidebarCollapsed ? 'w-72 p-6 md:w-20 md:p-3' : 'w-72 p-6'}
          ${isSidebarOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}
        `}>
          <div className="space-y-8">
            {/* Logo */}
            <div className={`hidden md:flex items-center ${isSidebarCollapsed ? 'flex-col gap-3' : 'justify-between'}`}>
              {!isSidebarCollapsed && (
                <span className="text-2xl font-black tracking-tight text-primary">SIGAP</span>
              )}
              <div className={`flex gap-2 ${isSidebarCollapsed ? 'flex-col' : ''}`}>
                <button
                  type="button"
                  onClick={(event) => {
                    event.stopPropagation();
                    toggleTheme();
                  }}
                  aria-label={theme === 'light' ? 'Aktifkan mode gelap' : 'Aktifkan mode terang'}
                  className="rounded-xl border border-border bg-background p-2 transition-all duration-200 hover:scale-105 active:scale-95"
                >
                  {theme === 'light' ? <Moon className="h-4 w-4 text-gray-700" /> : <Sun className="h-4 w-4 text-yellow-400" />}
                </button>
                <button
                  type="button"
                  onClick={(event) => {
                    event.stopPropagation();
                    setIsSidebarCollapsed((collapsed) => {
                      const nextValue = !collapsed;
                      localStorage.setItem('sigap_sidebar_collapsed', String(nextValue));
                      return nextValue;
                    });
                  }}
                  aria-label={isSidebarCollapsed ? 'Buka sidebar' : 'Tutup sidebar'}
                  title={isSidebarCollapsed ? 'Buka sidebar' : 'Tutup sidebar'}
                  className="rounded-xl border border-border bg-background p-2 text-muted transition-all duration-200 hover:text-foreground active:scale-95"
                >
                  {isSidebarCollapsed ? <PanelLeftOpen className="h-4 w-4" /> : <PanelLeftClose className="h-4 w-4" />}
                </button>
              </div>
            </div>

            {/* User Profile Card */}
            {user && (
              <div className={`flex items-center gap-4 rounded-2xl border border-border bg-background p-4 ${isSidebarCollapsed ? 'md:hidden' : ''}`}>
                <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-accent/20 text-[#775800] dark:text-accent">
                  <UserIcon className="h-5 w-5" />
                </div>
                <div className={`overflow-hidden ${isSidebarCollapsed ? 'md:hidden' : ''}`}>
                  <h4 className="text-sm font-semibold truncate">{user.full_name}</h4>
                  <span className={`inline-block mt-0.5 px-2 py-0.5 text-[10px] font-bold uppercase rounded-md tracking-wider ${
                    user.role === 'admin' 
                      ? 'bg-accent/20 text-[#775800] border border-accent/30 dark:text-accent'
                      : 'bg-success/10 text-success border border-success/20'
                  }`}>
                    {user.role}
                  </span>
                </div>
              </div>
            )}

            {/* Navigation links */}
            <nav className="space-y-1.5">
              {navigation.map((item) => {
                const isActive = pathname === item.href;
                const IconComponent = item.icon;
                return (
                  <Link
                    key={item.name}
                    href={item.href}
                    onClick={() => setIsSidebarOpen(false)}
                    className={`
                      flex items-center gap-3.5 px-4 py-3.5 rounded-xl font-medium text-sm transition-all duration-200 cursor-pointer
                      ${isSidebarCollapsed ? 'md:justify-center md:px-0' : ''}
                      ${isActive 
                        ? 'bg-primary text-white shadow-md shadow-primary/20 scale-[1.02]' 
                        : 'text-gray-500 hover:bg-background hover:text-foreground dark:text-gray-400'
                      }
                    `}
                    title={isSidebarCollapsed ? item.name : undefined}
                  >
                    <IconComponent className="h-5 w-5" />
                    <span className={isSidebarCollapsed ? 'md:hidden' : ''}>{item.name}</span>
                  </Link>
                );
              })}
            </nav>
          </div>

          {/* Logout Button */}
          <button
            onClick={handleLogout}
            title={isSidebarCollapsed ? 'Keluar Aplikasi' : undefined}
            className={`flex items-center gap-3 rounded-xl border border-transparent px-4 py-3.5 text-sm font-medium text-red-500 transition-all duration-200 hover:border-red-500/10 hover:bg-red-500/10 active:scale-[0.98] ${isSidebarCollapsed ? 'md:hidden' : ''}`}
          >
            <LogOut className="h-5 w-5" />
            <span className={isSidebarCollapsed ? 'md:hidden' : ''}>Keluar Aplikasi</span>
          </button>
        </aside>

        {/* Main Content Area */}
        <main className="flex-1 flex flex-col min-w-0 overflow-y-auto px-6 py-8 md:px-10">
          <div className="max-w-6xl w-full mx-auto">
            {children}
          </div>
        </main>
      </div>
    </ProtectedRoute>
  );
}

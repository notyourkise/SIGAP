'use client';

import React, { useState } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { useAuth } from '@/context/AuthContext';
import { useTheme } from '@/context/ThemeContext';
import ProtectedRoute from './ProtectedRoute';
import { 
  Camera, 
  Wrench, 
  LogOut, 
  Menu, 
  X, 
  User as UserIcon, 
  Sun, 
  Moon,
  Laptop,
  Database
} from 'lucide-react';
import Link from 'next/link';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { user, logout } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const router = useRouter();
  const pathname = usePathname();
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);

  const navigation = [
    { name: 'Aset CCTV', href: '/', icon: Camera },
    { name: 'Master CCTV', href: '/cctv', icon: Database },
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
          <Link href="/" className="text-2xl font-black bg-gradient-to-r from-blue-500 to-indigo-600 bg-clip-text text-transparent dark:from-blue-400 dark:to-indigo-400">
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
          fixed md:sticky top-0 left-0 h-screen w-72 bg-card border-r border-border flex flex-col justify-between p-6 z-40 transition-all duration-300
          ${isSidebarOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}
        `}>
          <div className="space-y-8">
            {/* Logo */}
            <div className="hidden md:flex items-center justify-between">
              <span className="text-2xl font-black bg-gradient-to-r from-blue-500 to-indigo-600 bg-clip-text text-transparent dark:from-blue-400 dark:to-indigo-400">
                SIGAP
              </span>
              <button
                onClick={toggleTheme}
                className="p-2 rounded-xl border border-border bg-background hover:scale-105 active:scale-95 transition-all duration-200"
              >
                {theme === 'light' ? <Moon className="h-4 w-4 text-gray-700" /> : <Sun className="h-4 w-4 text-yellow-400" />}
              </button>
            </div>

            {/* User Profile Card */}
            {user && (
              <div className="flex items-center gap-4 p-4 rounded-2xl bg-background border border-border">
                <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-blue-500/10 text-blue-500 dark:bg-blue-400/10 dark:text-blue-400">
                  <UserIcon className="h-5 w-5" />
                </div>
                <div className="overflow-hidden">
                  <h4 className="text-sm font-semibold truncate">{user.full_name}</h4>
                  <span className={`inline-block mt-0.5 px-2 py-0.5 text-[10px] font-bold uppercase rounded-md tracking-wider ${
                    user.role === 'admin' 
                      ? 'bg-red-500/10 text-red-500 border border-red-500/20' 
                      : 'bg-emerald-500/10 text-emerald-500 border border-emerald-500/20'
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
                      ${isActive 
                        ? 'bg-primary text-white shadow-md shadow-primary/20 scale-[1.02]' 
                        : 'text-gray-500 hover:bg-background hover:text-foreground dark:text-gray-400'
                      }
                    `}
                  >
                    <IconComponent className="h-5 w-5" />
                    <span>{item.name}</span>
                  </Link>
                );
              })}
            </nav>
          </div>

          {/* Logout Button */}
          <button
            onClick={handleLogout}
            className="flex items-center gap-3 px-4 py-3.5 rounded-xl font-medium text-sm text-red-500 hover:bg-red-500/10 active:scale-[0.98] transition-all duration-200 cursor-pointer border border-transparent hover:border-red-500/10"
          >
            <LogOut className="h-5 w-5" />
            <span>Keluar Aplikasi</span>
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

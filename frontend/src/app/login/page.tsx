'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/context/AuthContext';
import { apiRequest } from '@/lib/api';
import { useTheme } from '@/context/ThemeContext';
import { Eye, EyeOff, Lock, User, Sun, Moon, ShieldAlert } from 'lucide-react';

export default function LoginPage() {
  const { login, isAuthenticated, loading: authLoading } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const router = useRouter();

  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  // Jika user sudah terautentikasi, alihkan ke dashboard
  useEffect(() => {
    if (isAuthenticated && !authLoading) {
      router.push('/');
    }
  }, [isAuthenticated, authLoading, router]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    
    if (!username || !password) {
      setError('Username/Email dan Password wajib diisi');
      return;
    }

    setLoading(true);
    try {
      const response = await apiRequest<{ token: string; user: any }>('/auth/login', {
        method: 'POST',
        body: JSON.stringify({ username, password }),
      });

      if (response.status === 'success' && response.data) {
        login(response.data.token, response.data.user);
      } else {
        setError(response.message || 'Login gagal, silakan periksa kredensial Anda');
      }
    } catch (err: any) {
      setError(err.message || 'Terjadi kesalahan jaringan atau server');
    } finally {
      setLoading(false);
    }
  };

  if (authLoading) {
    return (
      <div className="flex h-screen w-screen items-center justify-center bg-background">
        <div className="h-10 w-10 animate-spin rounded-full border-4 border-primary border-t-transparent"></div>
      </div>
    );
  }

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-background px-4 py-12 transition-colors duration-300">
      {/* Background Gradients */}
      <div className="absolute -top-40 -left-40 h-[600px] w-[600px] rounded-full bg-accent/15 blur-[120px] dark:bg-accent/5"></div>
      <div className="absolute -bottom-40 -right-40 h-[600px] w-[600px] rounded-full bg-primary/15 blur-[120px] dark:bg-primary/10"></div>

      {/* Floating Theme Toggle */}
      <button
        onClick={toggleTheme}
        className="absolute top-6 right-6 p-3 rounded-xl border border-border bg-card shadow-sm hover:scale-105 active:scale-95 transition-all duration-200"
        title={theme === 'light' ? 'Mode Gelap' : 'Mode Terang'}
      >
        {theme === 'light' ? <Moon className="h-5 w-5 text-gray-700" /> : <Sun className="h-5 w-5 text-yellow-400" />}
      </button>

      <div className="w-full max-w-md z-10">
        <div className="glass rounded-3xl p-8 shadow-2xl border border-border bg-card/60 backdrop-blur-md">
          {/* Header */}
          <div className="text-center mb-8">
            <h1 className="text-4xl font-extrabold tracking-tight text-primary">
              SIGAP
            </h1>
            <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
              Sistem Identifikasi Gangguan Aset Pusat
            </p>
            <h2 className="mt-6 text-xl font-bold">
              Selamat datang kembali
            </h2>
            <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">
              Silakan login ke akun Anda untuk memantau CCTV IKN
            </p>
          </div>

          {/* Error Alert */}
          {error && (
            <div className="mb-6 flex items-center gap-3 p-4 rounded-xl border border-red-500/20 bg-red-500/10 text-sm text-red-600 dark:text-red-400 animate-shake">
              <ShieldAlert className="h-5 w-5 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {/* Form */}
          <form onSubmit={handleSubmit} className="space-y-6">
            <div>
              <label className="block text-sm font-medium mb-2" htmlFor="username">
                Username / Email
              </label>
              <div className="relative">
                <div className="absolute inset-y-0 left-0 flex items-center pl-4 pointer-events-none text-gray-400">
                  <User className="h-5 w-5" />
                </div>
                <input
                  id="username"
                  type="text"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder="Masukkan username atau email"
                  className="w-full pl-11 pr-4 py-3 rounded-xl border border-input-border bg-input text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary transition-all duration-200"
                  required
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium mb-2" htmlFor="password">
                Password
              </label>
              <div className="relative">
                <div className="absolute inset-y-0 left-0 flex items-center pl-4 pointer-events-none text-gray-400">
                  <Lock className="h-5 w-5" />
                </div>
                <input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••"
                  className="w-full pl-11 pr-12 py-3 rounded-xl border border-input-border bg-input text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary transition-all duration-200"
                  required
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute inset-y-0 right-0 flex items-center pr-4 text-gray-400 hover:text-foreground transition-colors"
                >
                  {showPassword ? <EyeOff className="h-5 w-5" /> : <Eye className="h-5 w-5" />}
                </button>
              </div>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full py-3.5 px-4 rounded-xl bg-primary hover:bg-primary-hover text-white font-semibold shadow-lg shadow-primary/20 active:scale-[0.98] disabled:opacity-50 disabled:scale-100 transition-all duration-200 cursor-pointer flex items-center justify-center gap-2"
            >
              {loading ? (
                <div className="h-5 w-5 animate-spin rounded-full border-2 border-white border-t-transparent"></div>
              ) : (
                'Masuk ke Sistem'
              )}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}

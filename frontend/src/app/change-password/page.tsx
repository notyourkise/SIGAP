'use client';

// Force dynamic rendering - authenticated-only pages can't be statically exported
export const dynamic = 'force-dynamic';

import { useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Eye, EyeOff, KeyRound, LoaderCircle, Lock, Moon, ShieldAlert, Sun } from 'lucide-react';
import { useAuth, type User as AuthUser } from '@/context/AuthContext';
import { useTheme } from '@/context/ThemeContext';
import { apiRequest } from '@/lib/api';

export default function ChangePasswordPage() {
  const { user, login, logout } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const router = useRouter();
  const searchParams = useSearchParams();
  
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  // Early guard: Redirect authenticated non-forced users to dashboard immediately
  useEffect(() => {
    if (user && !user.must_change_password) {
      router.push('/dashboard');
    }
  }, [user, router]);

  const forced = user?.must_change_password === true;

  // Handle successful password change navigation
  const handleSuccess = (updatedUser: AuthUser) => {
    login(updatedUser);
    
    if (!forced) {
      router.replace('/profile?password_changed=true');
    } else {
      router.push('/login?force_logout=true');
    }
  };

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError('');
    
    if (newPassword !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }
    
    if (newPassword.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }

    setLoading(true);
    try {
      await apiRequest(
        '/auth/change-password', 
        {
          method: 'POST',
          body: JSON.stringify({ new_password: newPassword }),
        }
      );
      
      if (user) {
        const updatedUser = { ...user, must_change_password: false };
        handleSuccess(updatedUser);
      }
    } catch (err: any) {
      console.error('Failed to change password:', err);
      setError(err.message || 'Failed to change password');
    } finally {
      setLoading(false);
    }
  };

  const handleExit = () => {
    if (forced) {
      logout();
      router.push('/login');
    } else {
      router.replace('/profile');
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800">
      {/* Theme Toggle */}
      <button
        onClick={toggleTheme}
        className="absolute top-4 right-4 p-2 rounded-full hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
        aria-label="Toggle theme"
      >
        {theme === 'dark' ? (
          <Sun className="w-5 h-5 text-gray-700 dark:text-gray-300" />
        ) : (
          <Moon className="w-5 h-5 text-gray-700 dark:text-gray-300" />
        )}
      </button>

      {/* Card Container */}
      <div className="w-full max-w-md p-6 bg-white dark:bg-gray-800 rounded-lg shadow-md space-y-6">
        {/* Header */}
        <div className="text-center space-y-2">
          <div className="flex justify-center">
            <ShieldAlert className="w-12 h-12 text-blue-600 dark:text-blue-400" />
          </div>
          
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            Password Required
          </h1>
          
          {forced && (
            <p className="text-sm text-red-600 dark:text-red-400">
              You must change your password before continuing
            </p>
          )}
        </div>

        {/* Error Alert */}
        {error && (
          <div className="flex items-center gap-2 p-3 bg-red-50 dark:bg-red-900/20 border-l-4 border-red-500 rounded">
            <ShieldAlert className="w-5 h-5 text-red-600 dark:text-red-400" />
            <span className="text-sm text-red-700 dark:text-red-400">{error}</span>
          </div>
        )}

        {/* Exit Button for Forced Mode */}
        {forced && (
          <button
            onClick={handleExit}
            className="w-full py-2 px-4 border border-gray-300 dark:border-gray-600 rounded-md 
                      text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 
                      transition-colors"
          >
            Exit / Back to Login
          </button>
        )}

        {/* Form */}
        <form onSubmit={handleSubmit} className="space-y-4">
          {/* New Password Field */}
          <div className="space-y-2">
            <label htmlFor="new_password" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              New Password
            </label>
            <div className="relative">
              <KeyRound className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
              <input
                id="new_password"
                type={showPassword ? 'text' : 'password'}
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                className="pl-10 pr-10 py-2 w-full border border-gray-300 dark:border-gray-600 
                          rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent 
                          bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                required
                autoFocus
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 
                          hover:text-gray-600 dark:hover:text-gray-200"
              >
                {showPassword ? (
                  <EyeOff className="w-5 h-5" />
                ) : (
                  <Eye className="w-5 h-5" />
                )}
              </button>
            </div>
          </div>

          {/* Confirm Password Field */}
          <div className="space-y-2">
            <label htmlFor="confirm_password" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Confirm New Password
            </label>
            <div className="relative">
              <Lock className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
              <input
                id="confirm_password"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                className="pl-10 pr-10 py-2 w-full border border-gray-300 dark:border-gray-600 
                          rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent 
                          bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                required
              />
              {confirmPassword && (
                <div className={`absolute right-3 top-1/2 transform -translate-y-1/2 ${
                  newPassword === confirmPassword ? 
                    'text-green-500' : 'text-red-500'
                }`}>
                  {newPassword === confirmPassword ? (
                    <ShieldAlert className="w-5 h-5" />
                  ) : (
                    <ShieldAlert className="w-5 h-5" />
                  )}
                </div>
              )}
            </div>
          </div>

          {/* Submit Button */}
          <button
            type="submit"
            disabled={loading}
            className="w-full py-2 px-4 bg-blue-600 hover:bg-blue-700 text-white font-semibold 
                      rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? (
              <div className="flex items-center justify-center gap-2">
                <LoaderCircle className="animate-spin w-5 h-5" />
                Processing...
              </div>
            ) : (
              'Change Password'
            )}
          </button>
        </form>

        {/* Help Text */}
        {forced && (
          <p className="text-xs text-center text-gray-500 dark:text-gray-400">
            Your password will be required for future logins
          </p>
        )}
      </div>
    </div>
  );
}

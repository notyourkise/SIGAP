'use client';

import React from 'react';
import DashboardLayout from '@/components/DashboardLayout';
import ChecklistTemplateManager from '@/components/ChecklistTemplateManager';
import { ListChecks } from 'lucide-react';

// Halaman admin untuk mengelola template checklist pengecekan lapangan.
// Dirender penuh (full-page) dan memakai kembali komponen yang sama dengan
// yang dipakai di halaman field-maintenance (mode modal).
export default function ChecklistTemplatesPage() {
  return (
    <DashboardLayout>
      <div className="space-y-6">
        <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-[-0.025em]">Template Checklist</h1>
            <p className="mt-1 max-w-2xl text-sm text-muted">
              Kelola daftar item pengecekan lapangan yang wajib diperiksa teknisi.
            </p>
          </div>
          <span className="inline-flex w-fit items-center gap-2 rounded-xl border border-border bg-card px-4 py-2.5 text-sm font-semibold text-muted">
            <ListChecks className="h-4 w-4 text-primary" />
            Halaman Admin
          </span>
        </header>

        <ChecklistTemplateManager mode="page" />
      </div>
    </DashboardLayout>
  );
}

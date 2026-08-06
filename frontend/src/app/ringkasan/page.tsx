'use client';

import { useCallback, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import type { LucideIcon } from 'lucide-react';
import {
  AlertTriangle,
  ArrowLeftRight,
  CheckCircle2,
  CircleX,
  ClipboardCheck,
  ClipboardList,
  Clock3,
  ListTodo,
  LoaderCircle,
} from 'lucide-react';
import DashboardLayout from '@/components/DashboardLayout';
import Pagination from '@/components/Pagination';
import { useAuth } from '@/context/AuthContext';
import { apiRequest } from '@/lib/api';

interface DailyDigest {
  id: string;
  digest_date: string;
  new_tasks: number;
  completed_tasks: number;
  failed_tasks: number;
  overdue_tasks: number;
  field_checks: number;
  attention_checks: number;
  status_changes: number;
  summary: string;
}

interface DigestResponse {
  rows: DailyDigest[];
  pagination: {
    total_rows: number;
    total_pages: number;
    page: number;
    limit: number;
  };
}

interface DigestMetricProps {
  icon: LucideIcon;
  label: string;
  value: number;
  tone: string;
  detail?: string;
}

const PAGE_SIZE = 10;

function formatDigestDate(value: string) {
  const dateOnly = value.slice(0, 10);
  const [year, month, day] = dateOnly.split('-').map(Number);
  if (!year || !month || !day) return value;

  return new Intl.DateTimeFormat('id-ID', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  }).format(new Date(year, month - 1, day, 12, 0, 0));
}

function formatNumber(value: number) {
  return new Intl.NumberFormat('id-ID').format(Number.isFinite(value) ? value : 0);
}

function DigestMetric({ icon: Icon, label, value, tone, detail }: DigestMetricProps) {
  return (
    <div className="rounded-xl border border-border bg-background/55 p-3.5">
      <div className="flex items-center gap-2 text-xs font-semibold text-muted">
        <Icon className={`h-4 w-4 ${tone}`} />
        <span>{label}</span>
      </div>
      <p className="mt-2 text-xl font-bold tabular-nums tracking-[-0.02em]">{formatNumber(value)}</p>
      {detail && <p className="mt-0.5 text-[11px] text-muted">{detail}</p>}
    </div>
  );
}

export default function DailyDigestPage() {
  const { user, loading: authLoading } = useAuth();
  const router = useRouter();
  const [digests, setDigests] = useState<DailyDigest[]>([]);
  const [page, setPage] = useState(1);
  const [pagination, setPagination] = useState({
    total_rows: 0,
    total_pages: 1,
    page: 1,
    limit: PAGE_SIZE,
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!authLoading && user?.role !== 'admin') router.replace('/dashboard');
  }, [authLoading, router, user?.role]);

  const loadDigests = useCallback(async () => {
    if (user?.role !== 'admin') {
      setLoading(false);
      return;
    }

    setLoading(true);
    setError('');
    try {
      const response = await apiRequest<DigestResponse>(`/dashboard/digest?page=${page}&limit=${PAGE_SIZE}`);
      setDigests(response.data?.rows ?? []);
      setPagination(response.data?.pagination ?? {
        total_rows: 0,
        total_pages: 1,
        page: 1,
        limit: PAGE_SIZE,
      });
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Ringkasan operasional belum dapat dimuat. Coba lagi.');
    } finally {
      setLoading(false);
    }
  }, [page, user?.role]);

  useEffect(() => {
    const timer = setTimeout(() => void loadDigests(), 0);
    return () => clearTimeout(timer);
  }, [loadDigests]);

  if (!authLoading && user?.role !== 'admin') return null;

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <div className="flex items-center gap-2 text-xs font-bold uppercase tracking-[0.08em] text-primary">
              <ClipboardList className="h-4 w-4" />
              <span>Pos komando / laporan harian</span>
            </div>
            <h1 className="mt-2 text-3xl font-bold tracking-[-0.025em]">Ringkasan Operasional</h1>
            <p className="mt-1 max-w-2xl text-sm text-muted">
              Catatan harian otomatis tentang pergerakan tugas, pengecekan lapangan, dan status CCTV.
            </p>
          </div>
          <div className="flex items-center gap-2 rounded-xl border border-border bg-card px-4 py-3 text-sm">
            <Clock3 className="h-4 w-4 text-primary" />
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.08em] text-muted">Pembaruan</p>
              <p className="mt-0.5 font-semibold">Setiap hari, 06:00 WITA</p>
            </div>
          </div>
        </header>

        {error && (
          <div role="alert" className="flex items-start gap-3 rounded-xl border border-error/25 bg-error/10 px-4 py-3 text-sm text-error">
            <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <section aria-labelledby="digest-timeline-heading" className="rounded-2xl border border-border bg-card p-4 sm:p-6">
          <div className="flex flex-col gap-1 border-b border-border pb-4 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <h2 id="digest-timeline-heading" className="text-lg font-bold tracking-[-0.015em]">Timeline harian</h2>
              <p className="mt-1 text-xs text-muted">Satu kartu merangkum aktivitas 24 jam sebelum waktu penerbitan.</p>
            </div>
            <span className="text-xs font-semibold text-muted">{formatNumber(pagination.total_rows)} ringkasan</span>
          </div>

          {loading ? (
            <div className="flex min-h-64 items-center justify-center gap-3 text-sm text-muted" role="status">
              <LoaderCircle className="h-5 w-5 animate-spin text-primary" />
              Memuat ringkasan operasional...
            </div>
          ) : digests.length === 0 ? (
            <div className="flex min-h-64 flex-col items-center justify-center text-center">
              <ClipboardList className="h-10 w-10 text-muted/45" />
              <h3 className="mt-3 font-semibold">Belum ada ringkasan operasional</h3>
              <p className="mt-1 max-w-sm text-xs leading-relaxed text-muted">
                Ringkasan pertama akan tersedia setelah generator berjalan pada pukul 06:00 WITA.
              </p>
            </div>
          ) : (
            <div className="relative mt-5 space-y-5 before:absolute before:bottom-5 before:left-[15px] before:top-5 before:w-px before:bg-border sm:before:left-[17px]">
              {digests.map((digest) => (
                <article key={digest.id} className="relative pl-9 sm:pl-11">
                  <span className="absolute left-[9px] top-6 flex h-3.5 w-3.5 items-center justify-center rounded-full border-4 border-card bg-primary sm:left-[11px]" aria-hidden="true" />
                  <div className="rounded-2xl border border-border bg-card p-4 shadow-[0_4px_14px_rgba(32,63,37,0.04)] sm:p-5">
                    <div className="flex flex-col gap-3 border-b border-border pb-4 sm:flex-row sm:items-start sm:justify-between">
                      <div>
                        <p className="text-xs font-bold uppercase tracking-[0.08em] text-primary">Ringkasan harian</p>
                        <h3 className="mt-1 text-lg font-bold capitalize">{formatDigestDate(digest.digest_date)}</h3>
                      </div>
                      <span className="inline-flex w-fit items-center rounded-full border border-primary/20 bg-primary/10 px-3 py-1 text-xs font-semibold text-primary">
                        24 jam terakhir
                      </span>
                    </div>

                    <p className="mt-4 rounded-xl border border-border bg-background/60 px-3.5 py-3 text-sm leading-relaxed text-foreground">
                      {digest.summary}
                    </p>

                    <div className="mt-4 grid grid-cols-2 gap-2.5 sm:grid-cols-3 lg:grid-cols-6">
                      <DigestMetric icon={ListTodo} label="Tugas baru" value={digest.new_tasks} tone="text-primary" />
                      <DigestMetric icon={CheckCircle2} label="Selesai" value={digest.completed_tasks} tone="text-success" />
                      <DigestMetric icon={CircleX} label="Gagal" value={digest.failed_tasks} tone="text-error" />
                      <DigestMetric icon={Clock3} label="Overdue" value={digest.overdue_tasks} tone="text-warning" />
                      <DigestMetric
                        icon={ClipboardCheck}
                        label="Pengecekan"
                        value={digest.field_checks}
                        tone="text-accent"
                        detail={`${formatNumber(digest.attention_checks)} perlu perhatian`}
                      />
                      <DigestMetric icon={ArrowLeftRight} label="Perubahan status" value={digest.status_changes} tone="text-blue-500" />
                    </div>
                  </div>
                </article>
              ))}
            </div>
          )}

          {!loading && digests.length > 0 && (
            <Pagination
              page={pagination.page}
              limit={pagination.limit}
              totalRows={pagination.total_rows}
              totalPages={pagination.total_pages}
              onPageChange={setPage}
              itemLabel="ringkasan"
            />
          )}
        </section>
      </div>
    </DashboardLayout>
  );
}

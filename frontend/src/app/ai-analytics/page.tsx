'use client';

// Halaman AI Analytics (Task 14C.3, Sprint 39) — admin only.
// Menampilkan ringkasan hitungan orang unik hari ini per kamera (GET /api/ai/counts)
// dan grafik tren 7 hari per kamera (GET /api/ai/counts/:id/trend?days=7).
// Grafik ditulis mandiri (SVG line chart) tanpa dependency chart eksternal.

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/DashboardLayout';
import { useAuth } from '@/context/AuthContext';
import { apiRequest } from '@/lib/api';
import { BrainCircuit, ChevronDown, LoaderCircle, Users, Cctv, Activity } from 'lucide-react';

// Ringkasan per kamera dari GET /api/ai/counts.
interface AICountItem {
  cctv_id: string;
  nama: string;
  kode_cctv: string;
  unique_today: number;
  session_date: string;
}

// Titik tren harian dari GET /api/ai/counts/:id/trend.
interface AITrendPoint {
  date: string;
  unique_count: number;
}

const TREND_DAYS = 7;
const LINE_COLOR = '#467235';
const CHART_HEIGHT = 220;
const PAD = { top: 16, right: 16, bottom: 32, left: 40 };

// Format tanggal ISO (YYYY-MM-DD atau ISO timestamp) menjadi label pendek "dd MMM".
const formatDateLabel = (iso: string) => {
  if (!iso) return '';
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso.slice(0, 10);
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
  return `${date.getDate()} ${months[date.getMonth()]}`;
};

// Format jam/menit dari timestamp ISO untuk kolom "Terakhir diperbarui".
const formatTime = (iso: string) => {
  if (!iso) return null;
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return null;
  const hh = String(date.getHours()).padStart(2, '0');
  const mm = String(date.getMinutes()).padStart(2, '0');
  return `${hh}:${mm}`;
};

// Komponen grafik garis tren (SVG mandiri): x = tanggal, y = unique_count.
function AITrendChart({ data }: { data: AITrendPoint[] }) {
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);

  const chart = useMemo(() => {
    if (data.length === 0) return null;

    const width = 720; // lebar tetap viewBox; responsif via scale
    const innerWidth = width - PAD.left - PAD.right;
    const innerHeight = CHART_HEIGHT - PAD.top - PAD.bottom;
    const maxValue = Math.max(...data.map((p) => p.unique_count), 0);
    const yMax = Math.max(4, Math.ceil(maxValue / 4) * 4);
    const stepX = data.length === 1 ? innerWidth / 2 : innerWidth / (data.length - 1);
    const xAt = (index: number) => PAD.left + index * stepX;
    const yAt = (value: number) => PAD.top + innerHeight - (value / yMax) * innerHeight;

    const points = data.map((p, index) => ({ x: xAt(index), y: yAt(p.unique_count), value: p.unique_count }));
    const linePath = points.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x} ${p.y}`).join(' ');
    const areaPath = `${linePath} L ${points[points.length - 1].x} ${PAD.top + innerHeight} L ${points[0].x} ${PAD.top + innerHeight} Z`;
    const yTicks = Array.from({ length: 5 }, (_, index) => (yMax / 4) * index);

    return { width, innerHeight, yMax, points, linePath, areaPath, yTicks };
  }, [data]);

  if (!chart) return null;

  return (
    <svg
      viewBox={`0 0 ${chart.width} ${CHART_HEIGHT}`}
      role="img"
      aria-label="Grafik tren orang unik 7 hari"
      className="block h-auto w-full"
      preserveAspectRatio="xMidYMid meet"
    >
      {/* Garis grid horizontal */}
      {chart.yTicks.map((tick) => {
        const y = PAD.top + chart.innerHeight - (tick / chart.yMax) * chart.innerHeight;
        return (
          <line key={tick} x1={PAD.left} x2={chart.width - PAD.right} y1={y} y2={y} className="stroke-border" strokeWidth={1} />
        );
      })}

      {/* Label sumbu Y */}
      {chart.yTicks.map((tick) => {
        const y = PAD.top + chart.innerHeight - (tick / chart.yMax) * chart.innerHeight;
        return (
          <text key={tick} x={PAD.left - 8} y={y + 3} textAnchor="end" fontSize={10} className="fill-muted">
            {Math.round(tick)}
          </text>
        );
      })}

      {/* Area gradasi di bawah garis */}
      <defs>
        <linearGradient id="ai-trend-fill" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={LINE_COLOR} stopOpacity="0.35" />
          <stop offset="100%" stopColor={LINE_COLOR} stopOpacity="0.02" />
        </linearGradient>
      </defs>
      <path d={chart.areaPath} fill="url(#ai-trend-fill)" />

      {/* Garis tren */}
      <path d={chart.linePath} fill="none" stroke={LINE_COLOR} strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round" />

      {/* Titik data + area hover */}
      {chart.points.map((point, index) => (
        <g key={index}>
          <circle cx={point.x} cy={point.y} r={4} fill={LINE_COLOR} stroke="#ffffff" strokeWidth={2} />
          <circle
            cx={point.x}
            cy={point.y}
            r={16}
            fill="transparent"
            className="cursor-pointer"
            onMouseEnter={() => setHoveredIndex(index)}
            onMouseLeave={() => setHoveredIndex(null)}
          >
            <title>{`${formatDateLabel(data[index].date)}: ${point.value} orang unik`}</title>
          </circle>
        </g>
      ))}

      {/* Label sumbu X */}
      {data.map((point, index) => (
        <text
          key={point.date}
          x={chart.points[index].x}
          y={CHART_HEIGHT - 10}
          textAnchor="middle"
          fontSize={10}
          className="fill-muted"
        >
          {formatDateLabel(point.date)}
        </text>
      ))}

      {/* Garis penanda saat hover */}
      {hoveredIndex !== null && chart.points[hoveredIndex] && (
        <>
          <line
            x1={chart.points[hoveredIndex].x}
            x2={chart.points[hoveredIndex].x}
            y1={PAD.top}
            y2={PAD.top + chart.innerHeight}
            stroke={LINE_COLOR}
            strokeWidth={1.5}
            strokeDasharray="4 4"
            opacity={0.55}
          />
          <circle
            cx={chart.points[hoveredIndex].x}
            cy={chart.points[hoveredIndex].y}
            r={6.5}
            fill={LINE_COLOR}
            stroke="#ffffff"
            strokeWidth={2}
          />
          <g>
            <rect
              x={Math.min(chart.width - PAD.right - 130, Math.max(PAD.left, chart.points[hoveredIndex].x - 65))}
              y={Math.max(2, chart.points[hoveredIndex].y - 46)}
              width={130}
              height={36}
              rx={10}
              className="fill-card stroke-border"
            />
            <text
              x={Math.min(chart.width - PAD.right - 130, Math.max(PAD.left, chart.points[hoveredIndex].x - 65)) + 65}
              y={Math.max(2, chart.points[hoveredIndex].y - 46) + 15}
              textAnchor="middle"
              fontSize={11}
              className="fill-foreground"
              fontWeight="bold"
            >
              {formatDateLabel(data[hoveredIndex].date)}
            </text>
            <text
              x={Math.min(chart.width - PAD.right - 130, Math.max(PAD.left, chart.points[hoveredIndex].x - 65)) + 65}
              y={Math.max(2, chart.points[hoveredIndex].y - 46) + 29}
              textAnchor="middle"
              fontSize={10}
              className="fill-muted"
            >
              {data[hoveredIndex].unique_count} orang unik
            </text>
          </g>
        </>
      )}
    </svg>
  );
}

export default function AIAnalyticsPage() {
  const { user, loading: authLoading } = useAuth();
  const router = useRouter();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [counts, setCounts] = useState<AICountItem[]>([]);

  // Kamera terpilih untuk grafik tren (default: kamera pertama).
  const [selectedId, setSelectedId] = useState<string>('');
  const [trend, setTrend] = useState<AITrendPoint[]>([]);
  const [trendLoading, setTrendLoading] = useState(false);
  const [trendError, setTrendError] = useState('');

  // Hanya admin yang boleh mengakses halaman ini (guard kedua setelah proxy).
  useEffect(() => {
    if (!authLoading && user?.role !== 'admin') router.replace('/dashboard');
  }, [authLoading, router, user]);

  // Muat ringkasan hitungan hari ini.
  const fetchCounts = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const response = await apiRequest<AICountItem[]>('/ai/counts');
      if (response.status === 'success') {
        setCounts(response.data ?? []);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Data AI belum dapat dimuat. Coba lagi.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (user?.role !== 'admin') return;
    let cancelled = false;
    const load = async () => {
      await Promise.resolve();
      if (!cancelled) await fetchCounts();
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, [fetchCounts, user?.role]);

  // Muat tren 7 hari untuk kamera terpilih.
  const fetchTrend = useCallback(async (cctvId: string) => {
    if (!cctvId) return;
    setTrendLoading(true);
    setTrendError('');
    try {
      const response = await apiRequest<AITrendPoint[]>(`/ai/counts/${cctvId}/trend?days=${TREND_DAYS}`);
      if (response.status === 'success') {
        setTrend(response.data ?? []);
      }
    } catch (err) {
      setTrendError(err instanceof Error ? err.message : 'Tren AI belum dapat dimuat. Coba lagi.');
      setTrend([]);
    } finally {
      setTrendLoading(false);
    }
  }, []);

  // Set kamera default setelah daftar ringkasan dimuat, lalu ambil tren.
  useEffect(() => {
    if (counts.length > 0) {
      if (!selectedId || !counts.some((c) => c.cctv_id === selectedId)) {
        setSelectedId(counts[0].cctv_id);
      }
    }
  }, [counts, selectedId]);

  useEffect(() => {
    if (selectedId) void fetchTrend(selectedId);
  }, [selectedId, fetchTrend]);

  // Ringkasan kartu.
  const totalPeople = useMemo(() => counts.reduce((acc, c) => acc + (c.unique_today || 0), 0), [counts]);
  const totalCameras = counts.length;
  const activeCameras = useMemo(() => counts.filter((c) => (c.unique_today || 0) > 0).length, [counts]);

  const selectedCamera = counts.find((c) => c.cctv_id === selectedId) ?? null;

  return (
    <DashboardLayout>
      <div className="space-y-6">
        {/* Header halaman */}
        <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-[-0.025em]">AI Analytics</h1>
            <p className="mt-1 max-w-2xl text-sm text-muted">
              Pantau hasil deteksi orang (AI Detection) per kamera — ringkasan hari ini dan tren 7 hari terakhir.
            </p>
          </div>
          <button
            type="button"
            onClick={fetchCounts}
            disabled={loading}
            className="flex min-h-11 items-center justify-center gap-2 rounded-xl border border-border bg-card px-5 py-2.5 text-sm font-semibold transition-colors hover:border-primary disabled:cursor-wait disabled:opacity-60"
          >
            {loading ? 'Memuat…' : 'Muat ulang'}
          </button>
        </div>

        {error && (
          <div role="alert" className="rounded-xl border border-error/25 bg-error/10 px-4 py-3 text-sm text-error">{error}</div>
        )}

        {loading ? (
          <div className="flex items-center justify-center gap-3 py-24" role="status">
            <LoaderCircle className="h-6 w-6 animate-spin text-primary" />
            <span className="text-sm text-muted">Memuat data AI…</span>
          </div>
        ) : (
          <>
            {/* Kartu ringkasan */}
            <section aria-label="Ringkasan AI" className="grid overflow-hidden rounded-2xl border border-border bg-card sm:grid-cols-2 lg:grid-cols-3">
              <div className="flex min-h-20 items-center justify-between gap-6 border-b border-border px-5 py-5 lg:border-b-0 lg:border-r">
                <div className="flex items-center gap-2.5">
                  <Users className="h-4 w-4 text-primary" />
                  <span className="text-sm font-medium text-muted">Total Orang Hari Ini</span>
                </div>
                <strong className="text-2xl font-bold tabular-nums">{totalPeople}</strong>
              </div>
              <div className="flex min-h-20 items-center justify-between gap-6 border-b border-border px-5 py-5 sm:border-r lg:border-b-0">
                <div className="flex items-center gap-2.5">
                  <Cctv className="h-4 w-4 text-primary" />
                  <span className="text-sm font-medium text-muted">Jumlah Kamera Dipantau</span>
                </div>
                <strong className="text-2xl font-bold tabular-nums">{totalCameras}</strong>
              </div>
              <div className="flex min-h-20 items-center justify-between gap-6 px-5 py-5">
                <div className="flex items-center gap-2.5">
                  <Activity className="h-4 w-4 text-success" />
                  <span className="text-sm font-medium text-muted">Kamera dengan Hitungan &gt; 0</span>
                </div>
                <strong className="text-2xl font-bold tabular-nums">{activeCameras}</strong>
              </div>
            </section>

            {counts.length === 0 ? (
              /* State kosong — belum ada sesi deteksi hari ini */
              <div className="flex flex-col items-center justify-center rounded-2xl border border-border bg-card px-6 py-16 text-center">
                <BrainCircuit className="h-10 w-10 text-muted/45" />
                <h3 className="mt-3 text-base font-semibold">Belum ada data AI hari ini</h3>
                <p className="mt-1 max-w-md text-sm text-muted">
                  Aktifkan AI di halaman Monitor agar deteksi orang mulai tercatat, atau tunggu event deteksi berikutnya masuk.
                </p>
              </div>
            ) : (
              <>
                {/* Tabel per kamera — hari ini */}
                <section aria-labelledby="ai-table-title" className="rounded-2xl border border-border bg-card">
                  <div className="flex items-center gap-2 border-b border-border px-5 py-4">
                    <Cctv className="h-5 w-5 text-primary" />
                    <h2 id="ai-table-title" className="font-semibold">Per Kamera — Hari Ini</h2>
                    <span className="ml-auto text-xs text-muted">{totalCameras} kamera</span>
                  </div>
                  <div className="overflow-x-auto">
                    <table className="w-full min-w-[640px] border-collapse text-left">
                      <thead>
                        <tr className="border-b border-border bg-background/50 text-xs font-semibold text-muted">
                          <th className="px-5 py-3.5">Nama</th>
                          <th className="px-5 py-3.5">Kode CCTV</th>
                          <th className="px-5 py-3.5">Orang Unik</th>
                          <th className="px-5 py-3.5">Terakhir Diperbarui</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-border text-sm">
                        {counts.map((item) => (
                          <tr key={item.cctv_id} className="transition-colors hover:bg-background/40">
                            <td className="px-5 py-4 font-semibold">{item.nama || '-'}</td>
                            <td className="px-5 py-4">
                              <span className="font-mono text-xs text-primary">{item.kode_cctv || '-'}</span>
                            </td>
                            <td className="px-5 py-4">
                              <span className="inline-flex items-center gap-1.5">
                                <strong className="tabular-nums">{item.unique_today ?? 0}</strong>
                                {item.unique_today > 0 && (
                                  <span className="rounded-full border border-success/25 bg-success/10 px-2 py-0.5 text-[11px] font-semibold text-success">
                                    terdeteksi
                                  </span>
                                )}
                              </span>
                            </td>
                            <td className="px-5 py-4 text-xs text-muted">
                              {item.session_date ? formatDateLabel(item.session_date) : 'Hari ini'}
                              {item.session_date && formatTime(item.session_date) ? ` · ${formatTime(item.session_date)}` : ''}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </section>

                {/* Tren 7 hari per kamera */}
                <section aria-labelledby="ai-trend-title" className="rounded-2xl border border-border bg-card p-5">
                  <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                    <div className="flex items-center gap-2">
                      <BrainCircuit className="h-5 w-5 text-primary" />
                      <h2 id="ai-trend-title" className="font-semibold">Tren 7 Hari</h2>
                    </div>
                    <div className="relative">
                      <label htmlFor="ai-camera-select" className="sr-only">Pilih kamera</label>
                      <select
                        id="ai-camera-select"
                        value={selectedId}
                        onChange={(event) => setSelectedId(event.target.value)}
                        className="min-h-10 w-full appearance-none rounded-xl border border-border bg-background pl-4 pr-10 text-sm font-medium transition-colors hover:border-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary sm:w-72"
                      >
                        {counts.map((item) => (
                          <option key={item.cctv_id} value={item.cctv_id}>
                            {item.nama || item.kode_cctv || item.cctv_id}
                          </option>
                        ))}
                      </select>
                      <ChevronDown className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted" />
                    </div>
                  </div>

                  <div className="mb-3 flex items-center justify-between">
                    <p className="text-xs text-muted">
                      {selectedCamera
                        ? `${selectedCamera.nama || '-'} (${selectedCamera.kode_cctv || '-'}) — orang unik per hari`
                        : 'Pilih kamera untuk melihat tren.'}
                    </p>
                    <span className="rounded-full border border-border bg-background px-2.5 py-1 text-[11px] font-semibold text-muted">
                      Rentang {TREND_DAYS} Hari
                    </span>
                  </div>

                  {trendLoading ? (
                    <div className="flex items-center justify-center gap-3 py-16" role="status">
                      <LoaderCircle className="h-5 w-5 animate-spin text-primary" />
                      <span className="text-sm text-muted">Memuat tren…</span>
                    </div>
                  ) : trendError ? (
                    <div role="alert" className="rounded-xl border border-error/25 bg-error/10 px-4 py-3 text-sm text-error">{trendError}</div>
                  ) : trend.length === 0 ? (
                    <div className="flex flex-col items-center justify-center py-16 text-center">
                      <Activity className="h-10 w-10 text-muted/45" />
                      <p className="mt-3 text-sm text-muted">Belum ada data tren untuk kamera ini.</p>
                    </div>
                  ) : (
                    <AITrendChart data={trend} />
                  )}
                </section>
              </>
            )}
          </>
        )}
      </div>
    </DashboardLayout>
  );
}

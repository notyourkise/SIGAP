'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import DashboardLayout from '@/components/DashboardLayout';
import { useAuth } from '@/context/AuthContext';
import { apiRequest } from '@/lib/api';

interface Member {
  id: string;
  cctv_id: string;
  camera_code: string;
  camera_name: string;
  location_area: string;
  source_ticket_number?: string;
  is_active_member: boolean;
  connection_state: string;
  first_failed_observed_at?: string | null;
  confirmed_offline_at?: string | null;
  last_observed_at: string | null;
  recovery_started_at?: string | null;
  recovered_at?: string | null;
  consecutive_live_count: number;
  member_outcome: string;
  damage_findings?: string;
  field_maintenance_log_id?: string | null;
  replacement_maintenance_log_id?: string | null;
}

interface IncidentEventItem {
  id: string;
  event_type: string;
  summary: string;
  created_at: string;
}

interface Incident {
  id: string;
  ticket_number: string;
  title: string;
  status: string;
  scope_type: string;
  origin_type: string;
  correlation_key: string;
  first_failed_observed_at: string;
  confirmed_offline_at: string;
  sla_clock_start_at: string;
  sla_clock_basis: string;
  timestamp_source: string;
  opened_at: string;
  resolved_at?: string | null;
  resolution_type?: string | null;
  resolution_note?: string;
  merged_into_incident_id?: string | null;
  merge_reason?: string;
  assigned_to: string | null;
  maintenance_log_id: string | null;
  assignment_reminder_queued_at?: string | null;
  unrecovered_escalation_queued_at?: string | null;
  members: Member[];
  events?: IncidentEventItem[];
}

interface OutboxNotification {
  id: string;
  incident_id?: string | null;
  monitor_episode_id?: string | null;
  target_channel: string;
  event_type: string;
  status: string;
  attempts: number;
  created_at: string;
  sent_at?: string | null;
  failed_at?: string | null;
  sla_deadline_at?: string | null;
  sla_breached: boolean;
  sla_breached_at?: string | null;
  last_error: string;
}

interface PlannedOutageCamera {
  id: string;
  cctv_id: string;
  suppressed_first_failed_at?: string | null;
  suppressed_confirmed_offline_at?: string | null;
  last_verified_state: string;
  last_verified_at?: string | null;
  post_window_processed_at?: string | null;
}

interface PlannedOutage {
  id: string;
  reason: string;
  starts_at: string;
  ends_at: string;
  ended_early_at?: string | null;
  created_at: string;
  cameras: PlannedOutageCamera[];
}

interface MonitorEpisode {
  id: string;
  cause: string;
  detail: string;
  affected_camera_count: number;
  status: string;
  consecutive_healthy_cycles: number;
  started_at: string;
  last_seen_at: string;
  resolved_at?: string | null;
}

interface CameraOutcomeRow {
  incident_id: string;
  ticket_number: string;
  incident_status: string;
  resolution_type?: string | null;
  origin_type: string;
  timestamp_source: string;
  sla_clock_start_at: string;
  sla_clock_basis: string;
  cctv_id: string;
  camera_code: string;
  camera_name: string;
  source_ticket_number?: string;
  first_failed_observed_at?: string | null;
  confirmed_offline_at?: string | null;
  recovered_at?: string | null;
  member_outcome: string;
  damage_findings?: string;
  replacement_maintenance_log_id?: string | null;
  outcome_recorded_at?: string | null;
}

interface OutcomeSummary {
  total_camera_involvements: number;
  pending_count: number;
  self_recovered_count: number;
  recovered_verified_count: number;
  escalated_count: number;
  sla_breached_outbox_count: number;
  rows: CameraOutcomeRow[];
}

interface ProbeTelemetrySummary {
  window_hours: number;
  gsm_probe_count: number;
  gsm_avg_bytes_session: number;
  gsm_max_bytes_session: number;
  gsm_total_bytes_observed: number;
  gsm_avg_duration_ms: number;
  gsm_forced_close_count: number;
  gsm_unconfirmed_close_count: number;
  metro_probe_count: number;
}

interface IncidentsPayload {
  incidents: Incident[];
  notifications: OutboxNotification[];
  planned_outages: PlannedOutage[];
  monitor_episodes: MonitorEpisode[];
  outcome_summary: OutcomeSummary;
  gsm_telemetry_summary: ProbeTelemetrySummary;
  monitoring_enabled: boolean;
}

interface Technician {
  id: string;
  full_name: string;
}

interface CCTVOption {
  id: string;
  kode_cctv: string;
  nama: string;
  status: string;
}

type ModalMode = 'assign' | 'resolve' | 'merge' | 'escalate' | 'camera_task';
type ActiveTab = 'incidents' | 'planned' | 'outcomes' | 'delivery';

const statusLabels: Record<string, string> = {
  open: 'Menunggu penugasan',
  in_progress: 'Dalam penanganan',
  pending_verification: 'Menunggu verifikasi Admin',
  resolved: 'Ditutup Admin',
  merged: 'Digabung ke Tiket Bersama',
  offline: 'Offline terkonfirmasi',
  recovering: 'Mengamati pemulihan',
  recovered: 'Pulih stabil (>=5 menit)',
  unknown: 'Belum dapat dipastikan',
  escalated: 'Dinonaktifkan (Eskalasi Ganti Unit)',
  pending: 'Diantrekan (belum terkirim)',
  sending: 'Sedang dikirim ke API',
  sent: 'Dikonfirmasi API Telegram',
  ambiguous: 'Ambigu (konfirmasi terputus)',
  failed: 'Gagal terkirim',
};

const outcomeLabels: Record<string, string> = {
  pending: 'Masih ditangani',
  self_recovered: 'Pulih Mandiri',
  recovered_verified: 'Pulih Terverifikasi',
  escalated_replacement: 'Dieskalasikan Penggantian Unit',
  merged_transferred: 'Dipindahkan saat penggabungan',
};

const resolutionLabels: Record<string, string> = {
  self_recovered: 'Pulih Mandiri (tanpa penugasan teknisi)',
  repaired_verified: 'Pulih Setelah Penanganan',
  escalated_replacement: 'Eskalasi Penggantian Perangkat',
  partial_recovery_and_escalation: 'Campuran: Sebagian Pulih & Sebagian Eskalasi Ganti Unit',
};

const scopeLabels: Record<string, string> = {
  single_camera: 'Kamera Tunggal',
  shared_nvr: 'Tiket Bersama NVR',
  manual_merged: 'Tiket Bersama (Gabungan Admin)',
};

const formatWITA = (value?: string | null) => {
  if (!value) return '-';
  return new Date(value).toLocaleString('id-ID', { timeZone: 'Asia/Makassar' }) + ' WITA';
};

const formatBytes = (bytes: number) => {
  if (!bytes || bytes <= 0) return '0 B';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
};

const toLocalDatetimeInput = (d: Date) => {
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
};

export default function IncidentsPage() {
  const { user } = useAuth();
  const [data, setData] = useState<IncidentsPayload | null>(null);
  const [technicians, setTechnicians] = useState<Technician[]>([]);
  const [siteCCTVs, setSiteCCTVs] = useState<CCTVOption[]>([]);
  const [activeTab, setActiveTab] = useState<ActiveTab>('incidents');
  const [statusFilter, setStatusFilter] = useState<string>('active');
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [snapshotMs, setSnapshotMs] = useState<number>(0);

  // State modal aksi tiket
  const [selected, setSelected] = useState<Incident | null>(null);
  const [modalMode, setModalMode] = useState<ModalMode>('assign');
  const [technician, setTechnician] = useState('');
  const [note, setNote] = useState('');
  const [fieldTaskCameraID, setFieldTaskCameraID] = useState('');
  const [mergeSourceIDs, setMergeSourceIDs] = useState<string[]>([]);
  const [escalateDrafts, setEscalateDrafts] = useState<
    Record<string, { checked: boolean; findings: string; technician_id: string }>
  >({});

  // State form jendela pemeliharaan terjadwal (Keputusan #1)
  const [outageReason, setOutageReason] = useState('');
  const [outageStarts, setOutageStarts] = useState(() => toLocalDatetimeInput(new Date()));
  const [outageEnds, setOutageEnds] = useState(() =>
    toLocalDatetimeInput(new Date(Date.now() + 60 * 60 * 1000)),
  );
  const [outageCameraIDs, setOutageCameraIDs] = useState<string[]>([]);

  const load = useCallback(async () => {
    if (user?.role !== 'admin') {
      setLoading(false);
      return;
    }
    try {
      const result = await apiRequest<IncidentsPayload>('/incidents');
      if (result.status !== 'success' || !result.data) {
        throw new Error(result.message || 'Data insiden belum tersedia');
      }
      setData(result.data);
      setSnapshotMs(Date.now());
      setError('');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Tidak dapat memuat insiden');
    } finally {
      setLoading(false);
    }
  }, [user?.role]);

  useEffect(() => {
    const initial = setTimeout(() => void load(), 0);
    const timer = setInterval(() => void load(), 15000);
    return () => {
      clearTimeout(initial);
      clearInterval(timer);
    };
  }, [load]);

  useEffect(() => {
    if (user?.role !== 'admin') return;
    let active = true;
    apiRequest<{ rows?: Technician[] }>('/users?page=1&limit=100&role=teknisi')
      .then((res) => {
        if (active) setTechnicians(res.data?.rows || []);
      })
      .catch(() => {});
    apiRequest<{ rows?: CCTVOption[] }>('/cctv?page=1&limit=200')
      .then((res) => {
        if (active) setSiteCCTVs(res.data?.rows || []);
      })
      .catch(() => {});
    return () => {
      active = false;
    };
  }, [user?.role]);

  const techMap = useMemo(
    () => new Map(technicians.map((t) => [t.id, t.full_name])),
    [technicians],
  );

  const cctvMap = useMemo(
    () => new Map(siteCCTVs.map((c) => [c.id, `${c.kode_cctv} - ${c.nama}`])),
    [siteCCTVs],
  );

  const filteredIncidents = useMemo(() => {
    const list = data?.incidents || [];
    if (statusFilter === 'active') {
      return list.filter((i) => !['resolved', 'merged'].includes(i.status));
    }
    if (statusFilter === 'all') return list;
    return list.filter((i) => i.status === statusFilter);
  }, [data?.incidents, statusFilter]);

  const openActionModal = (inc: Incident, mode: ModalMode) => {
    setSelected(inc);
    setModalMode(mode);
    setTechnician(inc.assigned_to || '');
    setNote(
      mode === 'resolve' && !inc.assigned_to
        ? 'Pulih Mandiri: kamera stabil online >= 5 menit tanpa penugasan lapangan.'
        : '',
    );
    setMergeSourceIDs([]);
    const activeMembers = inc.members.filter((m) => m.is_active_member);
    setFieldTaskCameraID(activeMembers[0]?.cctv_id || '');
    const nextDrafts: Record<string, { checked: boolean; findings: string; technician_id: string }> = {};
    for (const m of activeMembers) {
      nextDrafts[m.cctv_id] = {
        checked: activeMembers.length === 1,
        findings: '',
        technician_id: inc.assigned_to || '',
      };
    }
    setEscalateDrafts(nextDrafts);
  };

  async function submitModalAction() {
    if (!selected) return;
    setBusy(true);
    setError('');
    setMessage('');
    try {
      if (modalMode === 'assign') {
        const res = await apiRequest(`/incidents/${selected.id}/assign`, {
          method: 'POST',
          body: JSON.stringify({ technician_id: technician }),
        });
        if (res.status !== 'success') throw new Error(res.message || 'Penugasan gagal');
        setMessage(res.message || 'Teknisi berhasil ditugaskan');
      } else if (modalMode === 'resolve') {
        const res = await apiRequest(`/incidents/${selected.id}/resolve`, {
          method: 'POST',
          body: JSON.stringify({ note }),
        });
        if (res.status !== 'success') throw new Error(res.message || 'Verifikasi ditolak');
        setMessage(res.message || 'Tiket berhasil diverifikasi dan ditutup');
      } else if (modalMode === 'camera_task') {
        const res = await apiRequest(`/incidents/${selected.id}/camera-tasks`, {
          method: 'POST',
          body: JSON.stringify({
            cctv_id: fieldTaskCameraID,
            technician_id: technician,
            notes: note,
          }),
        });
        if (res.status !== 'success') throw new Error(res.message || 'Pembuatan tugas lapangan gagal');
        setMessage(res.message || 'Tugas pemeriksaan lapangan individual dibuat');
      } else if (modalMode === 'merge') {
        const res = await apiRequest(`/incidents/${selected.id}/merge`, {
          method: 'POST',
          body: JSON.stringify({
            source_incident_ids: mergeSourceIDs,
            reason: note,
          }),
        });
        if (res.status !== 'success') throw new Error(res.message || 'Penggabungan tiket gagal');
        setMessage(res.message || 'Tiket berhasil digabung ke tiket bersama');
      } else if (modalMode === 'escalate') {
        const items = Object.entries(escalateDrafts)
          .filter(([, v]) => v.checked)
          .map(([cctv_id, v]) => ({
            cctv_id,
            damage_findings: v.findings.trim(),
            technician_id: v.technician_id,
          }));
        const res = await apiRequest(`/incidents/${selected.id}/escalate-replacement`, {
          method: 'POST',
          body: JSON.stringify({ note, items }),
        });
        if (res.status !== 'success') throw new Error(res.message || 'Eskalasi penggantian gagal');
        setMessage(res.message || 'Eskalasi penggantian perangkat berhasil disimpan');
      }
      setSelected(null);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Tindakan belum berhasil');
    } finally {
      setBusy(false);
    }
  }

  async function handleRetryOutbox(outboxID: string) {
    setBusy(true);
    setError('');
    setMessage('');
    try {
      const res = await apiRequest(`/incidents/outbox/${outboxID}/retry`, { method: 'POST' });
      if (res.status !== 'success') throw new Error(res.message || 'Gagal menjadwalkan kirim ulang');
      setMessage(res.message || 'Pengiriman ulang dijadwalkan');
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Kirim ulang gagal');
    } finally {
      setBusy(false);
    }
  }

  async function handleCreatePlannedOutage(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError('');
    setMessage('');
    try {
      const startsISO = new Date(outageStarts).toISOString();
      const endsISO = new Date(outageEnds).toISOString();
      const res = await apiRequest('/incidents/planned-outages', {
        method: 'POST',
        body: JSON.stringify({
          reason: outageReason,
          starts_at: startsISO,
          ends_at: endsISO,
          cctv_ids: outageCameraIDs,
        }),
      });
      if (res.status !== 'success') throw new Error(res.message || 'Gagal membuat jendela pemeliharaan');
      setMessage(res.message || 'Jendela pemeliharaan terjadwal berhasil dibuat');
      setOutageReason('');
      setOutageCameraIDs([]);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gagal membuat jendela pemeliharaan');
    } finally {
      setBusy(false);
    }
  }

  async function handleEndPlannedOutage(outageID: string) {
    setBusy(true);
    setError('');
    setMessage('');
    try {
      const res = await apiRequest(`/incidents/planned-outages/${outageID}/end`, {
        method: 'POST',
      });
      if (res.status !== 'success') throw new Error(res.message || 'Gagal mengakhiri jendela');
      setMessage(res.message || 'Jendela pemeliharaan diakhiri lebih awal');
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gagal mengakhiri jendela');
    } finally {
      setBusy(false);
    }
  }

  const activeIncidentCount =
    data?.incidents.filter((i) => !['resolved', 'merged'].includes(i.status)).length || 0;
  const pendingVerifyCount =
    data?.incidents.filter((i) => i.status === 'pending_verification').length || 0;
  const recoveredCamerasTotal =
    (data?.outcome_summary?.self_recovered_count || 0) +
    (data?.outcome_summary?.recovered_verified_count || 0);
  const escalatedCamerasTotal = data?.outcome_summary?.escalated_count || 0;
  const attentionOutboxCount =
    data?.notifications.filter((n) => ['failed', 'ambiguous'].includes(n.status) || n.sla_breached)
      .length || 0;

  const mergeCandidates = useMemo(() => {
    if (!selected || !data) return [];
    return data.incidents.filter(
      (i) => i.id !== selected.id && !['resolved', 'merged'].includes(i.status),
    );
  }, [data, selected]);

  return (
    <DashboardLayout>
      <main className="mx-auto max-w-6xl space-y-6 p-4 md:p-8">
        <header className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="text-sm font-medium text-muted">Operasional CCTV &amp; Telegram</p>
            <h1 className="mt-1 text-3xl font-bold">Tiket Gangguan &amp; Pemantauan</h1>
            <p className="mt-2 text-sm text-muted">
              Pantau tiket otomatis, kelola jendela pemeliharaan terjadwal, gabungkan gangguan bersama,
              eskalasi penggantian perangkat per kamera, dan verifikasi penutupan.
            </p>
          </div>
          <button
            type="button"
            onClick={() => void load()}
            className="rounded-lg border border-border bg-card px-4 py-2 text-sm font-medium hover:bg-muted/10"
          >
            Muat ulang
          </button>
        </header>

        {user?.role !== 'admin' ? (
          <p role="alert" className="rounded-xl border border-border bg-card p-5">
            Halaman pengelolaan insiden CCTV hanya tersedia untuk Admin.
          </p>
        ) : (
          <>
            {error && (
              <div
                role="alert"
                className="rounded-xl border border-red-300 bg-red-50 p-4 text-sm text-red-800 dark:border-red-800 dark:bg-red-950/40 dark:text-red-200"
              >
                {error}
              </div>
            )}
            {message && (
              <p
                role="status"
                className="rounded-xl border border-emerald-300 bg-emerald-50 p-4 text-sm text-emerald-800 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-200"
              >
                {message}
              </p>
            )}
            {data && !data.monitoring_enabled && (
              <div className="rounded-xl border border-amber-300 bg-amber-50 p-4 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200">
                Monitoring aktif insiden belum diaktifkan pada variabel lingkungan ini (
                <code>CCTV_INCIDENTS_ENABLED=false</code>). Data historis, jendela pemeliharaan, dan
                tindakan Admin tetap dapat digunakan.
              </div>
            )}
            {data?.monitor_episodes?.some((ep) => ep.status === 'open') && (
              <div
                role="alert"
                className="rounded-xl border border-amber-400 bg-amber-50 p-4 text-sm text-amber-950 dark:border-amber-700 dark:bg-amber-950/50 dark:text-amber-100"
              >
                <strong>Gangguan Pemantau Aktif (Kondisi Belum Dapat Dipastikan):</strong> Pemantau
                mendeteksi kendala infrastruktur MediaMTX/relay. Pembukaan tiket offline kamera baru
                ditahan sementara agar tidak menimbulkan alarm palsu.
              </div>
            )}

            {/* Kartu Ringkasan */}
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <div className="rounded-xl border border-border bg-card p-5">
                <p className="text-xs font-medium uppercase tracking-wide text-muted">
                  Insiden Aktif
                </p>
                <p className="mt-2 text-3xl font-semibold">{activeIncidentCount}</p>
                <p className="mt-1 text-xs text-muted">Terbuka &amp; dalam penanganan</p>
              </div>
              <div className="rounded-xl border border-border bg-card p-5">
                <p className="text-xs font-medium uppercase tracking-wide text-muted">
                  Menunggu Verifikasi Admin
                </p>
                <p className="mt-2 text-3xl font-semibold">{pendingVerifyCount}</p>
                <p className="mt-1 text-xs text-muted">Stabil online &ge; 5 menit (6 sampel)</p>
              </div>
              <div className="rounded-xl border border-border bg-card p-5">
                <p className="text-xs font-medium uppercase tracking-wide text-muted">
                  Hasil Kamera (Pulih / Eskalasi)
                </p>
                <p className="mt-2 text-3xl font-semibold">
                  {recoveredCamerasTotal} <span className="text-lg text-muted">/</span>{' '}
                  <span className="text-amber-600 dark:text-amber-400">{escalatedCamerasTotal}</span>
                </p>
                <p className="mt-1 text-xs text-muted">Dipisahkan tegas di laporan</p>
              </div>
              <div className="rounded-xl border border-border bg-card p-5">
                <p className="text-xs font-medium uppercase tracking-wide text-muted">
                  Outbox Perlu Perhatian / SLA
                </p>
                <p className="mt-2 text-3xl font-semibold">{attentionOutboxCount}</p>
                <p className="mt-1 text-xs text-muted">
                  Pelanggaran SLA &gt;5m: {data?.outcome_summary?.sla_breached_outbox_count || 0}
                </p>
              </div>
            </div>

            {/* Navigasi Tab */}
            <nav
              aria-label="Bagian pengelolaan insiden"
              className="flex flex-wrap gap-2 border-b border-border pb-3"
            >
              {(
                [
                  ['incidents', `Daftar Tiket (${data?.incidents.length || 0})`],
                  ['planned', `Pemeliharaan Terjadwal (${data?.planned_outages?.length || 0})`],
                  [
                    'outcomes',
                    `Laporan Hasil Per Kamera (${data?.outcome_summary?.total_camera_involvements || 0})`,
                  ],
                  ['delivery', `Outbox Telegram, Pemantau & GSM (${data?.notifications.length || 0})`],
                ] as Array<[ActiveTab, string]>
              ).map(([key, label]) => (
                <button
                  key={key}
                  type="button"
                  onClick={() => setActiveTab(key)}
                  className={`rounded-lg px-4 py-2 text-sm font-medium transition-colors ${
                    activeTab === key
                      ? 'bg-primary text-white'
                      : 'border border-border bg-card text-foreground hover:bg-muted/10'
                  }`}
                >
                  {label}
                </button>
              ))}
            </nav>

            {/* TAB 1: DAFTAR TIKET GANGGUAN */}
            {activeTab === 'incidents' && (
              <section aria-label="Daftar tiket gangguan" className="space-y-4">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-xs font-medium text-muted">Filter status:</span>
                  {[
                    ['active', 'Aktif'],
                    ['pending_verification', 'Menunggu Verifikasi'],
                    ['resolved', 'Ditutup'],
                    ['merged', 'Digabung'],
                    ['all', 'Semua'],
                  ].map(([val, lbl]) => (
                    <button
                      key={val}
                      type="button"
                      onClick={() => setStatusFilter(val)}
                      className={`rounded-full px-3 py-1 text-xs font-medium ${
                        statusFilter === val
                          ? 'bg-primary/15 text-primary border border-primary/40'
                          : 'border border-border bg-card text-muted'
                      }`}
                    >
                      {lbl}
                    </button>
                  ))}
                </div>

                {loading ? (
                  <p role="status" className="text-sm text-muted">
                    Memuat tiket gangguan…
                  </p>
                ) : filteredIncidents.length === 0 ? (
                  <div className="rounded-xl border border-border bg-card p-10 text-center">
                    <h2 className="font-semibold">Tidak ada tiket pada filter ini</h2>
                    <p className="mt-2 text-sm text-muted">
                      Tiket otomatis akan tercatat ketika kamera terkonfirmasi offline di luar
                      jendela pemeliharaan terjadwal.
                    </p>
                  </div>
                ) : (
                  filteredIncidents.map((inc) => {
                    const isClosedOrMerged = ['resolved', 'merged'].includes(inc.status);
                    const isLegacyBackfill =
                      inc.timestamp_source === 'legacy_backfill_opened_at';
                    const assignedName = inc.assigned_to
                      ? techMap.get(inc.assigned_to) || inc.assigned_to.slice(0, 8)
                      : null;

                    return (
                      <article
                        key={inc.id}
                        className="rounded-xl border border-border bg-card p-5 space-y-4"
                      >
                        <div className="flex flex-wrap items-start justify-between gap-3">
                          <div className="space-y-1">
                            <div className="flex flex-wrap items-center gap-2 text-xs">
                              <span className="font-mono font-semibold text-primary">
                                {inc.ticket_number || inc.id.slice(0, 8)}
                              </span>
                              <span className="rounded bg-muted/15 px-2 py-0.5 font-medium">
                                {scopeLabels[inc.scope_type] || inc.scope_type}
                              </span>
                              {inc.origin_type === 'post_planned_maintenance' ? (
                                <span className="rounded bg-amber-500/15 px-2 py-0.5 font-medium text-amber-700 dark:text-amber-300">
                                  Pasca-Pemeliharaan Terjadwal
                                </span>
                              ) : (
                                <span className="rounded bg-muted/10 px-2 py-0.5 text-muted">
                                  Gangguan Luar Jadwal
                                </span>
                              )}
                              {isLegacyBackfill && (
                                <span
                                  className="rounded border border-border px-2 py-0.5 text-muted"
                                  title="Tiket dibuat pada skema sebelum 000034; stempel waktu diisi dari waktu pembukaan tiket historis."
                                >
                                  Data Historis Backfill
                                </span>
                              )}
                            </div>
                            <h2 className="text-lg font-semibold">{inc.title}</h2>
                            {/* Label jujur semantik waktu (Koreksi #1) */}
                            <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted">
                              {isLegacyBackfill ? (
                                <span>
                                  Waktu tercatat historis (pembukaan tiket):{' '}
                                  <strong>{formatWITA(inc.opened_at)}</strong>
                                </span>
                              ) : (
                                <>
                                  <span>
                                    Gagal teramati pertama:{' '}
                                    <strong>{formatWITA(inc.first_failed_observed_at)}</strong>
                                  </span>
                                  <span>
                                    Offline terkonfirmasi:{' '}
                                    <strong>{formatWITA(inc.confirmed_offline_at)}</strong>
                                  </span>
                                  <span>
                                    Awal SLA 5m ({inc.sla_clock_basis === 'planned_window_end' ? 'akhir pemeliharaan' : 'observasi gagal pertama'}):{' '}
                                    <strong>{formatWITA(inc.sla_clock_start_at)}</strong>
                                  </span>
                                </>
                              )}
                            </div>
                            {assignedName && (
                              <p className="text-xs text-muted">
                                Teknisi ditugaskan: <strong>{assignedName}</strong>
                              </p>
                            )}
                            {inc.resolution_type && (
                              <p className="text-xs font-medium text-emerald-700 dark:text-emerald-400">
                                Hasil Penutupan:{' '}
                                {resolutionLabels[inc.resolution_type] || inc.resolution_type}
                                {inc.resolution_note ? ` — "${inc.resolution_note}"` : ''}
                              </p>
                            )}
                            {inc.status === 'merged' && (
                              <p className="text-xs font-medium text-amber-700 dark:text-amber-300">
                                Digabung ke tiket induk ID {inc.merged_into_incident_id?.slice(0, 8)}
                                {inc.merge_reason ? ` — Alasan: ${inc.merge_reason}` : ''}
                              </p>
                            )}
                          </div>
                          <span className="rounded-full border border-border px-3 py-1 text-xs font-semibold">
                            {statusLabels[inc.status] || inc.status}
                          </span>
                        </div>

                        {/* Daftar kamera terdampak & hasil per kamera */}
                        <ul className="space-y-2">
                          {inc.members.map((m) => (
                            <li
                              key={m.id}
                              className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border/60 bg-muted/5 px-3 py-2.5 text-sm"
                            >
                              <div className="space-y-0.5">
                                <div className="flex flex-wrap items-center gap-2 font-medium">
                                  <span>
                                    {m.camera_code ? `${m.camera_code} · ` : ''}
                                    {m.camera_name}
                                  </span>
                                  {m.source_ticket_number && (
                                    <span className="rounded bg-muted/15 px-2 py-0.5 font-mono text-[11px] text-muted">
                                      Asal: {m.source_ticket_number}
                                    </span>
                                  )}
                                  {m.member_outcome && m.member_outcome !== 'pending' && (
                                    <span
                                      className={`rounded px-2 py-0.5 text-[11px] font-semibold ${
                                        m.member_outcome === 'escalated_replacement'
                                          ? 'bg-amber-500/15 text-amber-700 dark:text-amber-300'
                                          : 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300'
                                      }`}
                                    >
                                      {outcomeLabels[m.member_outcome] || m.member_outcome}
                                    </span>
                                  )}
                                </div>
                                {m.damage_findings && (
                                  <p className="text-xs text-amber-700 dark:text-amber-300">
                                    Temuan kerusakan: {m.damage_findings}
                                  </p>
                                )}
                              </div>
                              <div className="text-right text-xs text-muted">
                                <div className="font-medium text-foreground">
                                  {statusLabels[m.connection_state] || m.connection_state}
                                  {m.connection_state === 'recovering' &&
                                    ` (${m.consecutive_live_count}/6 sampel)`}
                                </div>
                                {m.last_observed_at && (
                                  <div>Bukti terakhir: {formatWITA(m.last_observed_at)}</div>
                                )}
                                <div className="mt-0.5 flex justify-end gap-2">
                                  {m.field_maintenance_log_id && (
                                    <Link href="/maintenance" className="underline text-primary">
                                      Tugas Cek Lapangan
                                    </Link>
                                  )}
                                  {m.replacement_maintenance_log_id && (
                                    <Link href="/maintenance" className="underline text-amber-600">
                                      Tugas Penggantian Unit
                                    </Link>
                                  )}
                                </div>
                              </div>
                            </li>
                          ))}
                        </ul>

                        {/* Tombol Aksi Admin */}
                        <div className="flex flex-wrap items-center gap-2.5 pt-1">
                          {!isClosedOrMerged && (
                            <button
                              type="button"
                              onClick={() => openActionModal(inc, 'assign')}
                              className="rounded-lg bg-primary px-3.5 py-2 text-xs font-semibold text-white hover:opacity-90"
                            >
                              {inc.assigned_to ? 'Alihkan Teknisi' : 'Tugaskan Teknisi'}
                            </button>
                          )}
                          {inc.status === 'pending_verification' && (
                            <button
                              type="button"
                              onClick={() => openActionModal(inc, 'resolve')}
                              className="rounded-lg border border-emerald-600/40 bg-emerald-600/10 px-3.5 py-2 text-xs font-semibold text-emerald-700 dark:text-emerald-300"
                            >
                              {inc.assigned_to
                                ? 'Verifikasi & Tutup'
                                : 'Verifikasi & Tutup (Pulih Mandiri)'}
                            </button>
                          )}
                          {!isClosedOrMerged && inc.members.filter((m) => m.is_active_member).length > 1 && (
                            <button
                              type="button"
                              onClick={() => openActionModal(inc, 'camera_task')}
                              className="rounded-lg border border-border px-3.5 py-2 text-xs font-medium hover:bg-muted/10"
                            >
                              + Tugas Cek Kamera Lapangan
                            </button>
                          )}
                          {!isClosedOrMerged && (
                            <button
                              type="button"
                              onClick={() => openActionModal(inc, 'merge')}
                              className="rounded-lg border border-border px-3.5 py-2 text-xs font-medium hover:bg-muted/10"
                            >
                              Gabungkan ke Tiket Bersama
                            </button>
                          )}
                          {!isClosedOrMerged && inc.members.some((m) => m.is_active_member) && (
                            <button
                              type="button"
                              onClick={() => openActionModal(inc, 'escalate')}
                              className="rounded-lg border border-amber-500/40 bg-amber-500/10 px-3.5 py-2 text-xs font-medium text-amber-800 dark:text-amber-300"
                            >
                              Eskalasi Penggantian Perangkat
                            </button>
                          )}
                          {inc.maintenance_log_id && (
                            <Link
                              href="/maintenance"
                              className="px-2 py-2 text-xs font-medium underline text-primary"
                            >
                              Lihat Tugas Pemeliharaan Utama
                            </Link>
                          )}
                        </div>

                        {inc.events && inc.events.length > 0 && (
                          <details className="border-t border-border pt-2 text-xs text-muted">
                            <summary className="cursor-pointer font-medium">
                              Riwayat jejak tiket ({inc.events.length} peristiwa)
                            </summary>
                            <ul className="mt-2 space-y-1 pl-3">
                              {inc.events.map((ev) => (
                                <li key={ev.id}>
                                  <span className="font-mono">{formatWITA(ev.created_at)}</span> ·{' '}
                                  <strong>{ev.event_type}</strong> — {ev.summary}
                                </li>
                              ))}
                            </ul>
                          </details>
                        )}
                      </article>
                    );
                  })
                )}
              </section>
            )}

            {/* TAB 2: JENDELA PEMELIHARAAN TERJADWAL (Keputusan #1) */}
            {activeTab === 'planned' && (
              <section className="grid gap-6 lg:grid-cols-2">
                <form
                  onSubmit={handleCreatePlannedOutage}
                  className="rounded-xl border border-border bg-card p-5 space-y-4"
                >
                  <div>
                    <h2 className="text-lg font-semibold">
                      Tandai Penghentian Kamera Terjadwal
                    </h2>
                    <p className="mt-1 text-xs text-muted">
                      Tugas pemeliharaan rutin <code>ongoing</code> saja tidak menahan alarm.
                      Tandai kamera yang memang akan dimatikan sementara beserta alasan dan batas
                      waktu berakhir.
                    </p>
                  </div>
                  <label className="block text-sm">
                    Alasan Penghentian Terjadwal
                    <input
                      type="text"
                      required
                      maxLength={500}
                      value={outageReason}
                      onChange={(e) => setOutageReason(e.target.value)}
                      placeholder="Contoh: Upgrade firmware NVR Tower B / pemadaman panel listrik"
                      className="mt-1.5 block w-full rounded-lg border border-border bg-background p-2.5 text-sm"
                    />
                  </label>
                  <div className="grid gap-3 sm:grid-cols-2">
                    <label className="block text-sm">
                      Waktu Mulai
                      <input
                        type="datetime-local"
                        required
                        value={outageStarts}
                        onChange={(e) => setOutageStarts(e.target.value)}
                        className="mt-1.5 block w-full rounded-lg border border-border bg-background p-2.5 text-sm"
                      />
                    </label>
                    <label className="block text-sm">
                      Waktu Berakhir
                      <input
                        type="datetime-local"
                        required
                        value={outageEnds}
                        onChange={(e) => setOutageEnds(e.target.value)}
                        className="mt-1.5 block w-full rounded-lg border border-border bg-background p-2.5 text-sm"
                      />
                    </label>
                  </div>
                  <div>
                    <p className="text-sm font-medium">
                      Pilih Kamera yang Akan Dimatikan ({outageCameraIDs.length} dipilih)
                    </p>
                    <div className="mt-2 max-h-56 overflow-y-auto rounded-lg border border-border p-2 space-y-1">
                      {siteCCTVs.map((cam) => {
                        const checked = outageCameraIDs.includes(cam.id);
                        return (
                          <label
                            key={cam.id}
                            className="flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-muted/10"
                          >
                            <input
                              type="checkbox"
                              checked={checked}
                              onChange={(e) =>
                                setOutageCameraIDs((prev) =>
                                  e.target.checked
                                    ? [...prev, cam.id]
                                    : prev.filter((x) => x !== cam.id),
                                )
                              }
                            />
                            <span>
                              <strong>{cam.kode_cctv}</strong> — {cam.nama} ({cam.status})
                            </span>
                          </label>
                        );
                      })}
                    </div>
                  </div>
                  <button
                    type="submit"
                    disabled={busy || !outageReason.trim() || outageCameraIDs.length === 0}
                    className="rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
                  >
                    {busy ? 'Menyimpan…' : 'Simpan Jendela Pemeliharaan'}
                  </button>
                </form>

                <div className="rounded-xl border border-border bg-card p-5 space-y-4">
                  <h2 className="text-lg font-semibold">Daftar Jendela Pemeliharaan</h2>
                  {!data?.planned_outages?.length ? (
                    <p className="text-sm text-muted">
                      Belum ada jendela penghentian kamera terjadwal pada site ini.
                    </p>
                  ) : (
                    <ul className="divide-y divide-border">
                      {data.planned_outages.map((po) => {
                        const effectiveEnd = po.ended_early_at || po.ends_at;
                        const isActive =
                          snapshotMs > 0 &&
                          new Date(po.starts_at).getTime() <= snapshotMs &&
                          new Date(effectiveEnd).getTime() > snapshotMs;
                        return (
                          <li key={po.id} className="py-3.5 space-y-2 text-sm">
                            <div className="flex flex-wrap items-start justify-between gap-2">
                              <div>
                                <p className="font-semibold">{po.reason}</p>
                                <p className="text-xs text-muted">
                                  {formatWITA(po.starts_at)} &rarr; {formatWITA(effectiveEnd)}
                                </p>
                              </div>
                              <span
                                className={`rounded-full px-2.5 py-0.5 text-xs font-semibold ${
                                  isActive
                                    ? 'bg-amber-500/15 text-amber-700 dark:text-amber-300'
                                    : 'bg-muted/15 text-muted'
                                }`}
                              >
                                {isActive ? 'Sedang pemeliharaan terjadwal' : 'Sudah berakhir'}
                              </span>
                            </div>
                            <ul className="space-y-1 text-xs text-muted">
                              {po.cameras?.map((pc) => (
                                <li key={pc.id}>
                                  • {cctvMap.get(pc.cctv_id) || pc.cctv_id.slice(0, 8)} — status
                                  verifikasi terakhir: <strong>{pc.last_verified_state}</strong>
                                  {pc.suppressed_first_failed_at &&
                                    ` (gagal teramati saat pemeliharaan: ${formatWITA(pc.suppressed_first_failed_at)})`}
                                </li>
                              ))}
                            </ul>
                            {isActive && (
                              <button
                                type="button"
                                disabled={busy}
                                onClick={() => void handleEndPlannedOutage(po.id)}
                                className="rounded-lg border border-border px-3 py-1.5 text-xs font-medium hover:bg-muted/10"
                              >
                                Akhiri Lebih Awal &amp; Evaluasi Sekarang
                              </button>
                            )}
                          </li>
                        );
                      })}
                    </ul>
                  )}
                </div>
              </section>
            )}

            {/* TAB 3: LAPORAN HASIL PER KAMERA (Koreksi #5) */}
            {activeTab === 'outcomes' && (
              <section className="rounded-xl border border-border bg-card p-5 space-y-4">
                <div>
                  <h2 className="text-lg font-semibold">
                    Laporan Hasil Penanganan Tingkat Kamera
                  </h2>
                  <p className="mt-1 text-xs text-muted">
                    Memisahkan secara tegas kamera yang <strong>pulih</strong> dari kamera yang{' '}
                    <strong>dieskalasikan untuk penggantian unit</strong> beserta tautan tugas
                    penggantian masing-masing, tetap utuh meskipun tiket bersama induk sudah
                    ditutup.
                  </p>
                </div>
                {!data?.outcome_summary?.rows?.length ? (
                  <p className="text-sm text-muted">Belum ada catatan keterlibatan kamera.</p>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-xs">
                      <thead className="border-b border-border text-muted">
                        <tr>
                          <th className="py-2 pr-3">Tiket</th>
                          <th className="py-2 pr-3">Kamera</th>
                          <th className="py-2 pr-3">Asal &amp; Dasar SLA</th>
                          <th className="py-2 pr-3">Waktu Observasi</th>
                          <th className="py-2 pr-3">Hasil Kamera</th>
                          <th className="py-2">Tindak Lanjut</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-border">
                        {data.outcome_summary.rows.map((r) => (
                          <tr key={`${r.incident_id}-${r.cctv_id}`}>
                            <td className="py-2.5 pr-3 font-mono">
                              <div>{r.ticket_number}</div>
                              {r.source_ticket_number && (
                                <div className="text-[11px] text-muted">
                                  Asal: {r.source_ticket_number}
                                </div>
                              )}
                            </td>
                            <td className="py-2.5 pr-3">
                              <div className="font-medium">{r.camera_name}</div>
                              <div className="text-muted">{r.camera_code}</div>
                            </td>
                            <td className="py-2.5 pr-3">
                              <div>
                                {r.origin_type === 'post_planned_maintenance'
                                  ? 'Pasca-Pemeliharaan'
                                  : 'Luar Jadwal'}
                              </div>
                              <div className="text-muted">
                                Awal SLA: {formatWITA(r.sla_clock_start_at)}
                              </div>
                            </td>
                            <td className="py-2.5 pr-3">
                              {r.timestamp_source === 'legacy_backfill_opened_at' ? (
                                <span className="text-muted">
                                  Historis backfill: {formatWITA(r.confirmed_offline_at)}
                                </span>
                              ) : (
                                <>
                                  <div>Gagal 1: {formatWITA(r.first_failed_observed_at)}</div>
                                  <div>Pulih: {formatWITA(r.recovered_at)}</div>
                                </>
                              )}
                            </td>
                            <td className="py-2.5 pr-3">
                              <span
                                className={`inline-block rounded px-2 py-0.5 font-semibold ${
                                  r.member_outcome === 'escalated_replacement'
                                    ? 'bg-amber-500/15 text-amber-700 dark:text-amber-300'
                                    : r.member_outcome === 'pending'
                                      ? 'bg-muted/15 text-muted'
                                      : 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300'
                                }`}
                              >
                                {outcomeLabels[r.member_outcome] || r.member_outcome}
                              </span>
                              {r.damage_findings && (
                                <div className="mt-1 text-muted">{r.damage_findings}</div>
                              )}
                            </td>
                            <td className="py-2.5">
                              {r.replacement_maintenance_log_id ? (
                                <Link href="/maintenance" className="underline text-amber-600">
                                  Log Ganti #{r.replacement_maintenance_log_id.slice(0, 8)}
                                </Link>
                              ) : (
                                '-'
                              )}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </section>
            )}

            {/* TAB 4: OUTBOX TELEGRAM, EPISODE PEMANTAU, & TELEMETRI GSM */}
            {activeTab === 'delivery' && (
              <div className="space-y-6">
                {/* Ringkasan Telemetri Byte GSM Terukur */}
                <section className="rounded-xl border border-border bg-card p-5 space-y-3">
                  <h2 className="text-lg font-semibold">
                    Telemetri Konsumsi Pemeriksaan Aktif GSM (24 Jam Terakhir)
                  </h2>
                  <p className="text-xs text-muted">
                    Angka di bawah ini murni hasil pengukuran <code>bytesReceived</code> aktual dari
                    MediaMTX pada site ini (tanpa estimasi karangan paket GSM).
                  </p>
                  <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5 text-sm">
                    <div className="rounded-lg bg-muted/10 p-3">
                      <p className="text-xs text-muted">Jumlah Probe GSM</p>
                      <p className="mt-1 text-xl font-semibold">
                        {data?.gsm_telemetry_summary?.gsm_probe_count || 0}
                      </p>
                    </div>
                    <div className="rounded-lg bg-muted/10 p-3">
                      <p className="text-xs text-muted">Rata-rata / Maks per Sesi</p>
                      <p className="mt-1 text-xl font-semibold">
                        {formatBytes(data?.gsm_telemetry_summary?.gsm_avg_bytes_session || 0)} /{' '}
                        {formatBytes(data?.gsm_telemetry_summary?.gsm_max_bytes_session || 0)}
                      </p>
                    </div>
                    <div className="rounded-lg bg-muted/10 p-3">
                      <p className="text-xs text-muted">Total Byte GSM Teramati (24j)</p>
                      <p className="mt-1 text-xl font-semibold">
                        {formatBytes(data?.gsm_telemetry_summary?.gsm_total_bytes_observed || 0)}
                      </p>
                    </div>
                    <div className="rounded-lg bg-muted/10 p-3">
                      <p className="text-xs text-muted">Penutupan Paksa Sesi Bocor</p>
                      <p className="mt-1 text-xl font-semibold">
                        {data?.gsm_telemetry_summary?.gsm_forced_close_count || 0}
                      </p>
                    </div>
                    <div className="rounded-lg bg-muted/10 p-3">
                      <p className="text-xs text-muted">Penutupan Belum Terkonfirmasi</p>
                      <p
                        className={`mt-1 text-xl font-semibold ${
                          (data?.gsm_telemetry_summary?.gsm_unconfirmed_close_count || 0) > 0
                            ? 'text-amber-600 dark:text-amber-400'
                            : ''
                        }`}
                      >
                        {data?.gsm_telemetry_summary?.gsm_unconfirmed_close_count || 0}
                      </p>
                    </div>
                  </div>
                </section>

                {/* Episode Kesehatan Pemantau */}
                <section className="rounded-xl border border-border bg-card p-5 space-y-3">
                  <h2 className="text-lg font-semibold">
                    Episode Gangguan Pemantau (Deduplikasi 1 Peringatan per Kejadian)
                  </h2>
                  {!data?.monitor_episodes?.length ? (
                    <p className="text-sm text-muted">
                      Pemantau sehat; belum ada episode kegagalan infrastruktur pemantau.
                    </p>
                  ) : (
                    <ul className="divide-y divide-border text-sm">
                      {data.monitor_episodes.map((ep) => (
                        <li key={ep.id} className="py-3 flex flex-wrap justify-between gap-2">
                          <div>
                            <p className="font-medium">
                              {ep.cause} — {ep.affected_camera_count} kamera terdampak unknown
                            </p>
                            <p className="text-xs text-muted">
                              Mulai: {formatWITA(ep.started_at)} · Detail: {ep.detail}
                            </p>
                          </div>
                          <span className="text-xs font-semibold">
                            {ep.status === 'open'
                              ? 'Masih berlangsung'
                              : `Pulih pada ${formatWITA(ep.resolved_at)}`}
                          </span>
                        </li>
                      ))}
                    </ul>
                  )}
                </section>

                {/* Outbox Telegram dengan Pemisahan Queued vs Sent vs Ambiguous & SLA Breach */}
                <section className="rounded-xl border border-border bg-card p-5 space-y-3">
                  <div>
                    <h2 className="text-lg font-semibold">Antrean Pemberitahuan Telegram</h2>
                    <p className="mt-1 text-xs text-muted">
                      Waktu diantrekan dibedakan tegas dari konfirmasi API Telegram. Pengiriman
                      berstatus <strong>Ambigu</strong> berpotensi menghasilkan pesan duplikat di
                      grup bila sesungguhnya sudah diterima server Telegram sebelum koneksi putus.
                    </p>
                  </div>
                  <ul className="divide-y divide-border">
                    {data?.notifications.map((n) => (
                      <li key={n.id} className="py-3.5 text-sm space-y-1.5">
                        <div className="flex flex-wrap items-center justify-between gap-2">
                          <div className="flex flex-wrap items-center gap-2">
                            <span className="font-semibold">{n.event_type}</span>
                            <span className="rounded bg-muted/15 px-2 py-0.5 text-xs">
                              {n.target_channel === 'admin_group' ? 'Grup Admin' : 'Grup Teknisi'}
                            </span>
                            {n.sla_breached && (
                              <span className="rounded bg-red-500/15 px-2 py-0.5 text-xs font-semibold text-red-700 dark:text-red-300">
                                Pelanggaran SLA (&gt;5 menit)
                              </span>
                            )}
                          </div>
                          <div className="flex items-center gap-2">
                            <span className="text-xs font-medium">
                              {statusLabels[n.status] || n.status} · {n.attempts} percobaan
                            </span>
                            {['failed', 'ambiguous'].includes(n.status) && (
                              <button
                                type="button"
                                disabled={busy}
                                onClick={() => void handleRetryOutbox(n.id)}
                                className="rounded border border-primary/40 bg-primary/10 px-2.5 py-1 text-xs font-semibold text-primary hover:bg-primary/20"
                              >
                                Kirim Ulang Manual
                              </button>
                            )}
                          </div>
                        </div>
                        <div className="flex flex-wrap gap-x-4 text-xs text-muted">
                          <span>Diantrekan: {formatWITA(n.created_at)}</span>
                          {n.sent_at && <span>Dikonfirmasi API: {formatWITA(n.sent_at)}</span>}
                          {n.failed_at && <span>Gagal pada: {formatWITA(n.failed_at)}</span>}
                        </div>
                        {n.last_error && (
                          <p className="text-xs text-amber-700 dark:text-amber-300">
                            {n.last_error}
                          </p>
                        )}
                      </li>
                    ))}
                  </ul>
                </section>
              </div>
            )}

            {/* MODAL TINDAKAN ADMIN */}
            {selected && (
              <section
                aria-labelledby="incident-action"
                className="rounded-xl border-2 border-primary bg-card p-6 space-y-4"
              >
                <div>
                  <p className="font-mono text-xs font-semibold text-primary">
                    {selected.ticket_number}
                  </p>
                  <h2 id="incident-action" className="text-lg font-semibold">
                    {modalMode === 'assign' && 'Tugaskan / Alihkan Teknisi'}
                    {modalMode === 'resolve' && 'Verifikasi Pemulihan & Tutup Insiden'}
                    {modalMode === 'camera_task' && 'Buat Tugas Pemeriksaan Kamera Lapangan'}
                    {modalMode === 'merge' && 'Gabungkan Tiket Lain ke Tiket Bersama Ini'}
                    {modalMode === 'escalate' &&
                      'Eskalasi Penggantian Perangkat (Per Kamera Terdampak)'}
                  </h2>
                  <p className="mt-1 text-sm text-muted">{selected.title}</p>
                </div>

                {modalMode === 'assign' && (
                  <div className="space-y-2">
                    {selected.members.filter((m) => m.is_active_member).length > 1 && (
                      <p className="rounded-lg bg-muted/10 p-3 text-xs text-muted">
                        Tiket bersama ini memiliki beberapa kamera aktif. Sistem akan membuat{' '}
                        <strong>1 tugas perbaikan bersama NVR/jaringan</strong> tanpa memaksa teknisi
                        mengisi checklist fisik tiang per kamera.
                      </p>
                    )}
                    <label className="block text-sm">
                      Pilih Teknisi Site
                      <select
                        value={technician}
                        onChange={(e) => setTechnician(e.target.value)}
                        className="mt-1.5 block w-full rounded-lg border border-border bg-background p-2.5"
                      >
                        <option value="">Pilih teknisi</option>
                        {technicians.map((t) => (
                          <option key={t.id} value={t.id}>
                            {t.full_name}
                          </option>
                        ))}
                      </select>
                    </label>
                  </div>
                )}

                {modalMode === 'camera_task' && (
                  <div className="space-y-3">
                    <label className="block text-sm">
                      Kamera yang Perlu Diperiksa di Lapangan
                      <select
                        value={fieldTaskCameraID}
                        onChange={(e) => setFieldTaskCameraID(e.target.value)}
                        className="mt-1.5 block w-full rounded-lg border border-border bg-background p-2.5"
                      >
                        {selected.members
                          .filter((m) => m.is_active_member)
                          .map((m) => (
                            <option key={m.cctv_id} value={m.cctv_id}>
                              {m.camera_code} - {m.camera_name}
                            </option>
                          ))}
                      </select>
                    </label>
                    <label className="block text-sm">
                      Teknisi Pelaksana
                      <select
                        value={technician}
                        onChange={(e) => setTechnician(e.target.value)}
                        className="mt-1.5 block w-full rounded-lg border border-border bg-background p-2.5"
                      >
                        <option value="">Pilih teknisi</option>
                        {technicians.map((t) => (
                          <option key={t.id} value={t.id}>
                            {t.full_name}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label className="block text-sm">
                      Instruksi Pemeriksaan Lapangan
                      <textarea
                        maxLength={2000}
                        value={note}
                        onChange={(e) => setNote(e.target.value)}
                        className="mt-1.5 block w-full rounded-lg border border-border bg-background p-2.5"
                        placeholder="Contoh: Cek adaptor PoE dan konektor RJ45 di boks tiang"
                      />
                    </label>
                  </div>
                )}

                {modalMode === 'resolve' && (
                  <div className="space-y-2">
                    {!selected.assigned_to && (
                      <p className="rounded-lg bg-emerald-500/10 p-3 text-xs text-emerald-800 dark:text-emerald-200">
                        Tiket ini belum pernah ditugaskan ke teknisi. Menutup sekarang akan mencatat
                        hasil <strong>Pulih Mandiri</strong> tanpa membuat tugas teknisi atau
                        checklist.
                      </p>
                    )}
                    <label className="block text-sm">
                      Catatan Verifikasi Admin
                      <textarea
                        maxLength={2000}
                        value={note}
                        onChange={(e) => setNote(e.target.value)}
                        className="mt-1.5 block w-full rounded-lg border border-border bg-background p-2.5"
                        placeholder="Catat hasil pemeriksaan stream dan alasan penutupan"
                      />
                    </label>
                  </div>
                )}

                {modalMode === 'merge' && (
                  <div className="space-y-3">
                    <p className="text-xs text-muted">
                      Pilih tiket terbuka pada site yang sama yang terbukti berkaitan penyebabnya.
                      Waktu deteksi awal dan nomor tiket asal tetap dipertahankan.
                    </p>
                    {mergeCandidates.length === 0 ? (
                      <p className="text-sm text-muted">
                        Tidak ada tiket aktif lain pada site ini yang dapat digabung.
                      </p>
                    ) : (
                      <div className="max-h-44 overflow-y-auto rounded-lg border border-border p-2 space-y-1">
                        {mergeCandidates.map((cand) => (
                          <label
                            key={cand.id}
                            className="flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-muted/10"
                          >
                            <input
                              type="checkbox"
                              checked={mergeSourceIDs.includes(cand.id)}
                              onChange={(e) =>
                                setMergeSourceIDs((prev) =>
                                  e.target.checked
                                    ? [...prev, cand.id]
                                    : prev.filter((x) => x !== cand.id),
                                )
                              }
                            />
                            <span>
                              <strong>{cand.ticket_number}</strong> — {cand.title}
                            </span>
                          </label>
                        ))}
                      </div>
                    )}
                    <label className="block text-sm">
                      Bukti / Alasan Penggabungan
                      <textarea
                        maxLength={2000}
                        value={note}
                        onChange={(e) => setNote(e.target.value)}
                        className="mt-1.5 block w-full rounded-lg border border-border bg-background p-2.5"
                        placeholder="Contoh: Seluruh kamera terhubung ke switch distribusi yang sama yang kehilangan suplai listrik"
                      />
                    </label>
                  </div>
                )}

                {modalMode === 'escalate' && (
                  <div className="space-y-3">
                    <p className="text-xs text-muted">
                      Kamera yang dipilih akan dinonaktifkan sementara (<code>nonaktif</code>,{' '}
                      <code>monitoring_enabled=false</code>) dan masing-masing ditautkan ke tugas
                      penggantian unit tersendiri. Setelah unit baru terpasang, aktifkan kembali
                      melalui tombol <strong>Aktifkan &amp; Verifikasi Stream</strong> di Master
                      CCTV.
                    </p>
                    <div className="space-y-3">
                      {selected.members
                        .filter((m) => m.is_active_member)
                        .map((m) => {
                          const draft = escalateDrafts[m.cctv_id] || {
                            checked: false,
                            findings: '',
                            technician_id: '',
                          };
                          return (
                            <div
                              key={m.cctv_id}
                              className="rounded-lg border border-border p-3 space-y-2"
                            >
                              <label className="flex items-center gap-2 text-sm font-medium">
                                <input
                                  type="checkbox"
                                  checked={draft.checked}
                                  onChange={(e) =>
                                    setEscalateDrafts((prev) => ({
                                      ...prev,
                                      [m.cctv_id]: { ...draft, checked: e.target.checked },
                                    }))
                                  }
                                />
                                <span>
                                  {m.camera_code} — {m.camera_name}
                                </span>
                              </label>
                              {draft.checked && (
                                <div className="grid gap-2 sm:grid-cols-2 pt-1">
                                  <input
                                    type="text"
                                    placeholder="Temuan kerusakan fisik kamera ini (wajib)"
                                    value={draft.findings}
                                    onChange={(e) =>
                                      setEscalateDrafts((prev) => ({
                                        ...prev,
                                        [m.cctv_id]: { ...draft, findings: e.target.value },
                                      }))
                                    }
                                    className="rounded border border-border bg-background p-2 text-xs"
                                  />
                                  <select
                                    value={draft.technician_id}
                                    onChange={(e) =>
                                      setEscalateDrafts((prev) => ({
                                        ...prev,
                                        [m.cctv_id]: {
                                          ...draft,
                                          technician_id: e.target.value,
                                        },
                                      }))
                                    }
                                    className="rounded border border-border bg-background p-2 text-xs"
                                  >
                                    <option value="">Penanggung jawab penggantian (wajib)</option>
                                    {technicians.map((t) => (
                                      <option key={t.id} value={t.id}>
                                        {t.full_name}
                                      </option>
                                    ))}
                                  </select>
                                </div>
                              )}
                            </div>
                          );
                        })}
                    </div>
                    <label className="block text-sm">
                      Catatan Tambahan Admin (opsional)
                      <input
                        type="text"
                        maxLength={1000}
                        value={note}
                        onChange={(e) => setNote(e.target.value)}
                        className="mt-1 block w-full rounded-lg border border-border bg-background p-2.5 text-sm"
                        placeholder="Nomor pengajuan unit cadangan / keterangan eskalasi"
                      />
                    </label>
                  </div>
                )}

                <div className="flex gap-3 pt-2">
                  <button
                    type="button"
                    disabled={
                      busy ||
                      (modalMode === 'assign' && !technician) ||
                      (modalMode === 'resolve' && !note.trim()) ||
                      (modalMode === 'camera_task' && (!fieldTaskCameraID || !technician)) ||
                      (modalMode === 'merge' && (mergeSourceIDs.length === 0 || !note.trim())) ||
                      (modalMode === 'escalate' &&
                        Object.values(escalateDrafts).filter(
                          (d) => d.checked && d.findings.trim() && d.technician_id,
                        ).length === 0)
                    }
                    onClick={() => void submitModalAction()}
                    className="rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
                  >
                    {busy ? 'Menyimpan…' : 'Simpan Tindakan'}
                  </button>
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => setSelected(null)}
                    className="rounded-lg border border-border px-4 py-2 text-sm"
                  >
                    Batal
                  </button>
                </div>
              </section>
            )}
          </>
        )}
      </main>
    </DashboardLayout>
  );
}

'use client';

// Extends SIGAP's existing dashboard: one incident list, explicit camera evidence,
// admin actions and a separate delivery queue. No new visual identity is used.
import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import DashboardLayout from '@/components/DashboardLayout';
import { useAuth } from '@/context/AuthContext';
import { apiRequest } from '@/lib/api';

interface Member { id: string; camera_name: string; connection_state: string; last_observed_at: string | null; consecutive_live_count: number }
interface Incident { id: string; title: string; status: string; opened_at: string; assigned_to: string | null; maintenance_log_id: string | null; members: Member[] }
interface Notification { id: string; incident_id: string; event_type: string; status: string; attempts: number; last_error: string }
interface Data { incidents: Incident[]; notifications: Notification[]; monitoring_enabled: boolean }
interface Technician { id: string; full_name: string }
const labels: Record<string, string> = { open: 'Menunggu penugasan', in_progress: 'Dalam penanganan', pending_verification: 'Menunggu verifikasi admin', resolved: 'Ditutup', offline: 'Offline terkonfirmasi', recovering: 'Mengamati pemulihan', recovered: 'Pulih stabil', unknown: 'Belum dapat dipastikan', pending: 'Mengantre', sending: 'Sedang dikirim', sent: 'Diterima API Telegram', ambiguous: 'Hasil kirim belum pasti', failed: 'Pengiriman gagal' };
const formatTime = (value: string) => new Date(value).toLocaleString('id-ID', { timeZone: 'Asia/Makassar' });

export default function IncidentsPage() {
  const { user } = useAuth();
  const [data, setData] = useState<Data | null>(null);
  const [technicians, setTechnicians] = useState<Technician[]>([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [selected, setSelected] = useState<Incident | null>(null);
  const [action, setAction] = useState<'assign' | 'resolve'>('assign');
  const [technician, setTechnician] = useState('');
  const [note, setNote] = useState('');
  const [message, setMessage] = useState('');
  const load = useCallback(async () => {
    if (user?.role !== 'admin') { setLoading(false); return; }
    try {
      const result = await apiRequest<Data>('/incidents');
      if (result.status !== 'success' || !result.data) throw new Error(result.message || 'Data belum tersedia');
      setData(result.data); setError('');
    } catch (e) { setError(e instanceof Error ? e.message : 'Tidak dapat memuat insiden'); }
    finally { setLoading(false); }
  }, [user?.role]);
  useEffect(() => {
    const initial = setTimeout(() => void load(), 0);
    const timer = setInterval(() => void load(), 15000);
    return () => { clearTimeout(initial); clearInterval(timer); };
  }, [load]);
  useEffect(() => {
    if (user?.role !== 'admin') return;
    let active = true;
    apiRequest<{ rows?: Technician[] }>('/users?page=1&limit=100&role=teknisi').then(result => { if (active) setTechnicians(result.data?.rows || []); }).catch(() => {});
    return () => { active = false; };
  }, [user?.role]);
  async function submit() {
    if (!selected) return;
    setBusy(true); setError(''); setMessage('');
    try {
      const result = await apiRequest(`/incidents/${selected.id}/${action}`, { method: 'POST', body: JSON.stringify(action === 'assign' ? { technician_id: technician } : { note }) });
      if (result.status !== 'success') throw new Error(result.message || 'Tindakan ditolak');
      setMessage(result.message || 'Perubahan tersimpan'); setSelected(null); setNote(''); setTechnician(''); await load();
    } catch (e) { setError(e instanceof Error ? e.message : 'Tindakan belum berhasil'); }
    finally { setBusy(false); }
  }
  return <DashboardLayout><main className="mx-auto max-w-6xl space-y-6 p-4 md:p-8">
    <header className="flex flex-wrap items-start justify-between gap-4">
      <div><p className="text-sm font-medium text-muted">Operasional CCTV</p><h1 className="mt-1 text-3xl font-bold">Tiket Gangguan</h1><p className="mt-2 text-sm text-muted">Pantau gangguan, tugaskan teknisi, dan verifikasi pemulihan melalui SIGAP.</p></div>
      <button type="button" onClick={() => void load()} className="rounded-lg border border-border px-4 py-2 text-sm font-medium">Muat ulang</button>
    </header>
    {user?.role !== 'admin' ? <p role="alert">Halaman pengelolaan insiden hanya tersedia untuk admin.</p> : <>
      {error && <div role="alert" className="rounded-xl border border-red-300 bg-red-50 p-4 text-red-800">{error}</div>}
      {message && <p role="status" className="rounded-xl bg-green-50 p-4 text-green-800">{message}</p>}
      {data && !data.monitoring_enabled && <p className="rounded-xl border border-amber-300 bg-amber-50 p-4 text-amber-900">Monitoring insiden belum diaktifkan pada lingkungan ini. Data yang sudah tersimpan tetap dapat ditinjau.</p>}
      <div className="grid gap-4 sm:grid-cols-3">{[
        ['Insiden terbuka', data?.incidents.filter(i => i.status !== 'resolved').length || 0],
        ['Menunggu verifikasi', data?.incidents.filter(i => i.status === 'pending_verification').length || 0],
        ['Pemberitahuan perlu perhatian', data?.notifications.filter(n => ['failed', 'ambiguous'].includes(n.status)).length || 0],
      ].map(([label, count]) => <div key={label} className="rounded-xl border border-border bg-card p-5"><p className="text-sm text-muted">{label}</p><p className="mt-2 text-3xl font-semibold">{count}</p></div>)}</div>
      {loading ? <p role="status">Memuat tiket gangguan…</p> : data && data.incidents.length === 0 ? <section className="rounded-xl border border-border p-10 text-center"><h2 className="font-semibold">Belum ada tiket gangguan</h2><p className="mt-2 text-sm text-muted">Gangguan yang terkonfirmasi akan dicatat otomatis saat monitoring aktif.</p></section> : null}
      <section aria-label="Daftar tiket gangguan" className="space-y-4">{data?.incidents.map(i => <article key={i.id} className="rounded-xl border border-border bg-card p-5">
        <div className="flex flex-wrap items-start justify-between gap-3"><div><p className="text-xs text-muted">{formatTime(i.opened_at)} WITA · {i.id.slice(0, 8)}</p><h2 className="mt-1 text-lg font-semibold">{i.title}</h2></div><span className="rounded-full border border-border px-3 py-1 text-xs font-medium">{labels[i.status] || i.status}</span></div>
        <ul className="mt-4 space-y-2">{i.members.map(m => <li key={m.id} className="flex flex-wrap justify-between gap-2 rounded-lg bg-muted/10 px-3 py-2 text-sm"><span>{m.camera_name}</span><span>{labels[m.connection_state] || m.connection_state}{m.last_observed_at ? ` · ${formatTime(m.last_observed_at)} WITA` : ''}</span></li>)}</ul>
        <div className="mt-4 flex flex-wrap gap-3">
          {i.status !== 'resolved' && !i.assigned_to && <button onClick={() => { setSelected(i); setAction('assign'); }} className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white">Tugaskan teknisi</button>}
          {i.status === 'pending_verification' && <button onClick={() => { setSelected(i); setAction('resolve'); }} className="rounded-lg border border-border px-4 py-2 text-sm font-medium">Verifikasi &amp; tutup</button>}
          {i.maintenance_log_id && <Link href="/maintenance" className="px-2 py-2 text-sm underline">Lihat tugas pemeliharaan</Link>}
        </div>
      </article>)}</section>
      <section className="rounded-xl border border-border bg-card p-5"><h2 className="text-lg font-semibold">Pemberitahuan Telegram</h2><p className="mt-1 text-sm text-muted">Status diterima API tidak berarti pesan sudah dibaca teknisi. Menampilkan maksimal 50 pemberitahuan terbaru.</p><ul className="mt-4 divide-y divide-border">{data?.notifications.map(n => <li key={n.id} className="py-3 text-sm"><div className="flex flex-wrap justify-between gap-2"><span>{n.event_type} · {n.incident_id.slice(0, 8)}</span><span>{labels[n.status] || n.status} · {n.attempts} percobaan</span></div>{n.last_error && <p className="mt-1 text-amber-700">{n.last_error}</p>}</li>)}</ul></section>
      {selected && <section aria-labelledby="incident-action" className="rounded-xl border-2 border-primary bg-card p-6"><h2 id="incident-action" className="text-lg font-semibold">{action === 'assign' ? 'Tugaskan teknisi' : 'Verifikasi pemulihan'}</h2><p className="mt-1 text-sm text-muted">{selected.title}</p>
        {action === 'assign' ? <label className="mt-4 block text-sm">Teknisi<select value={technician} onChange={e => setTechnician(e.target.value)} className="mt-2 block w-full rounded-lg border border-border bg-card p-3"><option value="">Pilih teknisi</option>{technicians.map(t => <option key={t.id} value={t.id}>{t.full_name}</option>)}</select></label> : <label className="mt-4 block text-sm">Catatan verifikasi admin<textarea maxLength={2000} value={note} onChange={e => setNote(e.target.value)} className="mt-2 block w-full rounded-lg border border-border bg-card p-3" placeholder="Catat hasil pemeriksaan dan alasan penutupan" /></label>}
        <div className="mt-4 flex gap-3"><button disabled={busy || (action === 'assign' ? !technician : !note.trim())} onClick={() => void submit()} className="rounded-lg bg-primary px-4 py-2 text-white disabled:opacity-50">{busy ? 'Menyimpan…' : 'Simpan'}</button><button disabled={busy} onClick={() => setSelected(null)} className="rounded-lg border border-border px-4 py-2">Batal</button></div>
      </section>}
    </>}
  </main></DashboardLayout>;
}

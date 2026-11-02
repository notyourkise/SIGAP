'use client';
import { useState } from 'react';
import DashboardLayout from '@/components/DashboardLayout';
import { apiRequest } from '@/lib/api';

type Answer = { jawaban: string; dapat_ditentukan: boolean; bukti: { sumber: string; kutipan: string }[] };
export default function AIAssistantPage() {
 const [question,setQuestion]=useState(''); const [answer,setAnswer]=useState<Answer|null>(null);
 const [error,setError]=useState(''); const [busy,setBusy]=useState(false);
 async function submit(e: React.FormEvent) {
  e.preventDefault();setBusy(true);setError('');setAnswer(null);
  try {const r=await apiRequest<Answer>('/ai-assistant/ask',{method:'POST',body:JSON.stringify({question})});if(!r.data)throw new Error(r.message||'Jawaban belum tersedia');setAnswer(r.data)}
  catch(e){setError(e instanceof Error?e.message:'Asisten AI belum tersedia')}finally{setBusy(false)}
 }
 return <DashboardLayout><main className="mx-auto max-w-4xl space-y-6 p-4 md:p-8">
  <header><p className="text-sm text-muted">SIGAP · AI Engineering</p><h1 className="mt-2 text-3xl font-bold">Asisten Operasional AI</h1><p className="mt-2 text-sm text-muted">Tanyakan rancangan monitoring CCTV. Jawaban menyertakan kutipan dari dokumen kebutuhan proyek.</p></header>
  <div className="rounded-xl border border-amber-300 bg-amber-50 p-4 text-sm text-amber-900">Sumber saat ini adalah kebutuhan rancangan proyek, bukan SOP resmi atau kondisi CCTV saat ini. Hindari memasukkan kata sandi atau data sensitif.</div>
  <form onSubmit={submit} className="space-y-3 rounded-xl border border-border bg-card p-5"><label htmlFor="ai-question" className="font-medium">Pertanyaan</label><textarea id="ai-question" maxLength={2000} required value={question} onChange={e=>setQuestion(e.target.value)} placeholder="Apa yang dilakukan ketika CCTV kembali online?" className="block min-h-28 w-full rounded-lg border border-border bg-card p-3"/><button disabled={busy||!question.trim()} className="rounded-lg bg-primary px-4 py-2 text-white disabled:opacity-50">{busy?'Mencari sumber dan menyusun jawaban…':'Tanyakan'}</button></form>
  {error&&<div role="alert" className="rounded-xl bg-red-50 p-4 text-red-800">{error}</div>}
  {answer&&<section aria-live="polite" className="space-y-4 rounded-xl border border-border bg-card p-5"><h2 className="text-lg font-semibold">Jawaban</h2><p className="whitespace-pre-wrap">{answer.jawaban}</p><p className="text-sm text-muted">{answer.dapat_ditentukan?'Sumber yang digunakan':'Tidak dapat ditentukan berdasarkan sumber yang tersedia.'}</p><ul className="space-y-2">{answer.bukti.map((b,i)=><li key={i} className="rounded-lg bg-muted/10 p-3 text-sm"><b>{b.sumber}</b><blockquote className="mt-1">“{b.kutipan}”</blockquote></li>)}</ul></section>}
 </main></DashboardLayout>;
}

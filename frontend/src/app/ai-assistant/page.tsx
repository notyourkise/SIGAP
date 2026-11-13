'use client';

import { useEffect, useRef, useState } from 'react';
import DashboardLayout from '@/components/DashboardLayout';
import { apiRequest } from '@/lib/api';
import {
  AlertCircle,
  Bot,
  ChevronDown,
  FileText,
  Loader2,
  RotateCcw,
  SendHorizontal,
} from 'lucide-react';

type Citation = {
  sumber: string;
  kutipan: string;
};

type Answer = {
  jawaban: string;
  dapat_ditentukan: boolean;
  bukti: Citation[];
};

type ChatMessage = {
  id: string;
  role: 'user' | 'assistant';
  text: string;
  timestamp: string;
  answer?: Answer;
  error?: string;
  retryQuestion?: string;
};

const STARTER_QUESTIONS = [
  'Apa entrypoint resmi web SIGAP untuk deployment Docker?',
  'Data penting SIGAP disimpan di dua tempat apa?',
  'Apakah status standby selalu dicatat sebagai event log?',
  'Apa hasil ketika koneksi ke MediaMTX gagal menurut dokumen verifikasi aktif?',
];

function formatTime(date = new Date()): string {
  return date.toLocaleTimeString('id-ID', {
    hour: '2-digit',
    minute: '2-digit',
  });
}

function CitationList({ bukti }: { bukti: Citation[] }) {
  const [expanded, setExpanded] = useState(true);

  if (bukti.length === 0) return null;

  return (
    <div className="border-t border-border/70 pt-2.5">
      <button
        type="button"
        onClick={() => setExpanded((prev) => !prev)}
        className="flex w-full items-center justify-between gap-2 text-left text-xs font-medium text-muted transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 rounded"
      >
        <span className="inline-flex items-center gap-1.5">
          <FileText className="h-3.5 w-3.5 text-primary shrink-0" />
          <span>{bukti.length} sumber terverifikasi</span>
        </span>
        <ChevronDown
          className={`h-3.5 w-3.5 shrink-0 transition-transform duration-150 ${
            expanded ? 'rotate-180' : ''
          }`}
        />
      </button>

      {expanded && (
        <ul className="mt-2 space-y-2">
          {bukti.map((item, idx) => (
            <li
              key={`${item.sumber}-${idx}`}
              className="rounded-lg border border-border/80 bg-card/70 px-3 py-2 text-xs"
            >
              <p className="font-mono text-[11px] font-medium text-primary break-all">
                {item.sumber}
              </p>
              <blockquote className="mt-1 whitespace-pre-wrap text-muted leading-relaxed">
                “{item.kutipan}”
              </blockquote>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

export default function AIAssistantPage() {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [busy, setBusy] = useState(false);

  const listEndRef = useRef<HTMLDivElement | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const nextSeqRef = useRef(0);

  useEffect(() => {
    listEndRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
  }, [messages, busy]);

  function nextId(prefix: string) {
    nextSeqRef.current += 1;
    return `${prefix}-${nextSeqRef.current}`;
  }

  async function sendQuestion(rawQuestion: string) {
    const trimmed = rawQuestion.trim();
    if (!trimmed || busy) return;

    const userMsg: ChatMessage = {
      id: nextId('u'),
      role: 'user',
      text: trimmed,
      timestamp: formatTime(),
    };

    setMessages((prev) => [...prev, userMsg]);
    setInput('');
    setBusy(true);

    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
    }

    try {
      const res = await apiRequest<Answer>('/ai-assistant/ask', {
        method: 'POST',
        body: JSON.stringify({ question: trimmed }),
      });

      if (!res.data) {
        throw new Error(res.message || 'Jawaban belum tersedia.');
      }

      const assistantMsg: ChatMessage = {
        id: nextId('a'),
        role: 'assistant',
        text: res.data.jawaban,
        timestamp: formatTime(),
        answer: res.data,
      };
      setMessages((prev) => [...prev, assistantMsg]);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : 'Asisten AI belum tersedia saat ini.';
      const errMsg: ChatMessage = {
        id: nextId('e'),
        role: 'assistant',
        text: message,
        timestamp: formatTime(),
        error: message,
        retryQuestion: trimmed,
      };
      setMessages((prev) => [...prev, errMsg]);
    } finally {
      setBusy(false);
    }
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    void sendQuestion(input);
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
      e.preventDefault();
      void sendQuestion(input);
    }
  }

  function handleInputChange(e: React.ChangeEvent<HTMLTextAreaElement>) {
    setInput(e.target.value);
    const el = e.target;
    el.style.height = 'auto';
    el.style.height = `${Math.min(el.scrollHeight, 144)}px`;
  }

  function handleResetChat() {
    if (busy) return;
    setMessages([]);
    setInput('');
    textareaRef.current?.focus();
  }

  return (
    <DashboardLayout>
      <div className="mx-auto flex h-[calc(100dvh-9.5rem)] min-h-[540px] max-w-4xl flex-col gap-4">
        {/* Header Bar */}
        <header className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground sm:text-3xl">
              Asisten Operasional AI
            </h1>
            <p className="mt-1 text-sm text-muted">
              Tanya jawab dokumentasi SOP SIGAP lengkap dengan kutipan sumber terverifikasi.
            </p>
          </div>

          {messages.length > 0 && (
            <button
              type="button"
              onClick={handleResetChat}
              disabled={busy}
              className="inline-flex items-center gap-2 rounded-lg border border-border bg-card px-3 py-2 text-xs font-medium text-foreground transition-colors hover:bg-background disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
            >
              <RotateCcw className="h-3.5 w-3.5" />
              <span>Percakapan baru</span>
            </button>
          )}
        </header>

        {/* Chat Container */}
        <section
          aria-label="Ruang percakapan Asisten AI SIGAP"
          className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-2xl border border-border bg-card"
        >
          {/* Scrollable Message List */}
          <div
            aria-live="polite"
            className="flex-1 space-y-5 overflow-y-auto px-4 py-5 sm:px-6"
          >
            {/* Opening Welcome Bubble */}
            <div className="flex items-start gap-3">
              <div
                aria-hidden="true"
                className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-primary/25 bg-primary/15 text-primary"
              >
                <Bot className="h-4 w-4" />
              </div>

              <div className="max-w-[88%] space-y-3 sm:max-w-[80%]">
                <div className="rounded-2xl rounded-tl-sm border border-border bg-background px-4 py-3 text-sm leading-relaxed text-foreground">
                  <p>
                    Halo! Saya <strong>Asisten Operasional SIGAP</strong>. Anda dapat
                    menanyakan dokumentasi akses Docker, backup &amp; restore, log
                    stream CCTV, batasan codec/snapshot, atau prosedur verifikasi aktif.
                  </p>
                  <p className="mt-2 text-xs text-muted">
                    Catatan: Jawaban bersumber dari dokumen SOP proyek dan tidak membaca
                    status kamera live. Hindari mengirim kata sandi atau data sensitif.
                  </p>
                </div>

                {messages.length === 0 && (
                  <div className="flex flex-wrap gap-2">
                    {STARTER_QUESTIONS.map((q) => (
                      <button
                        key={q}
                        type="button"
                        disabled={busy}
                        onClick={() => void sendQuestion(q)}
                        className="rounded-full border border-border bg-background px-3 py-1.5 text-left text-xs text-foreground transition-colors hover:border-primary/50 hover:bg-primary/10 disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                      >
                        {q}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            </div>

            {/* Conversation Bubbles */}
            {messages.map((msg) => {
              if (msg.role === 'user') {
                return (
                  <div key={msg.id} className="flex justify-end">
                    <div className="max-w-[82%] sm:max-w-[75%]">
                      <div className="rounded-2xl rounded-tr-sm bg-primary px-4 py-2.5 text-sm leading-relaxed text-white whitespace-pre-wrap break-words">
                        {msg.text}
                      </div>
                      <p className="mt-1 text-right text-[11px] text-muted">
                        Anda · {msg.timestamp}
                      </p>
                    </div>
                  </div>
                );
              }

              return (
                <div key={msg.id} className="flex items-start gap-3">
                  <div
                    aria-hidden="true"
                    className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full border ${
                      msg.error
                        ? 'border-error/35 bg-error/10 text-error'
                        : 'border-primary/25 bg-primary/15 text-primary'
                    }`}
                  >
                    {msg.error ? (
                      <AlertCircle className="h-4 w-4" />
                    ) : (
                      <Bot className="h-4 w-4" />
                    )}
                  </div>

                  <div className="max-w-[88%] sm:max-w-[80%]">
                    <div
                      className={`space-y-2.5 rounded-2xl rounded-tl-sm border px-4 py-3 text-sm leading-relaxed ${
                        msg.error
                          ? 'border-error/40 bg-error/10 text-error'
                          : 'border-border bg-background text-foreground'
                      }`}
                    >
                      <p className="whitespace-pre-wrap break-words">{msg.text}</p>

                      {msg.answer && msg.answer.bukti.length > 0 && (
                        <CitationList bukti={msg.answer.bukti} />
                      )}

                      {msg.error && msg.retryQuestion && (
                        <div>
                          <button
                            type="button"
                            disabled={busy}
                            onClick={() => void sendQuestion(msg.retryQuestion!)}
                            className="inline-flex items-center gap-1.5 rounded-lg border border-error/40 bg-card px-2.5 py-1 text-xs font-medium text-error transition-colors hover:bg-error/10 disabled:opacity-50"
                          >
                            <RotateCcw className="h-3 w-3" />
                            <span>Coba lagi</span>
                          </button>
                        </div>
                      )}
                    </div>

                    <p className="mt-1 text-[11px] text-muted">
                      Asisten SIGAP · {msg.timestamp}
                    </p>
                  </div>
                </div>
              );
            })}

            {/* Loading / Typing Bubble */}
            {busy && (
              <div className="flex items-start gap-3">
                <div
                  aria-hidden="true"
                  className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-primary/25 bg-primary/15 text-primary"
                >
                  <Bot className="h-4 w-4" />
                </div>
                <div className="inline-flex items-center gap-2 rounded-2xl rounded-tl-sm border border-border bg-background px-4 py-3 text-sm text-muted">
                  <Loader2 className="h-4 w-4 animate-spin text-primary" />
                  <span>Menelusuri dokumen &amp; menyusun jawaban…</span>
                </div>
              </div>
            )}

            <div ref={listEndRef} />
          </div>

          {/* Composer */}
          <form
            onSubmit={handleSubmit}
            className="border-t border-border bg-card px-3 py-3 sm:px-5 sm:py-3.5"
          >
            <div className="flex items-end gap-2 rounded-xl border border-input-border bg-input p-2 focus-within:border-primary focus-within:ring-2 focus-within:ring-primary/20">
              <label htmlFor="ai-question" className="sr-only">
                Ketik pertanyaan tentang dokumentasi SIGAP
              </label>
              <textarea
                ref={textareaRef}
                id="ai-question"
                rows={1}
                maxLength={2000}
                value={input}
                onChange={handleInputChange}
                onKeyDown={handleKeyDown}
                disabled={busy}
                placeholder="Tulis pertanyaan atau ketik sapaan..."
                className="max-h-36 min-h-[38px] flex-1 resize-none bg-transparent px-2 py-1.5 text-sm text-foreground placeholder:text-muted focus:outline-none disabled:opacity-60"
              />
              <button
                type="submit"
                disabled={busy || !input.trim()}
                aria-label="Kirim pesan"
                className="inline-flex h-9 shrink-0 items-center gap-1.5 rounded-lg bg-primary px-3.5 text-xs font-semibold text-white transition-colors hover:bg-primary-hover disabled:cursor-not-allowed disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
              >
                {busy ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <SendHorizontal className="h-4 w-4" />
                )}
                <span className="hidden sm:inline">Kirim</span>
              </button>
            </div>

            <div className="mt-1.5 flex items-center justify-between px-1 text-[11px] text-muted">
              <span>Enter untuk kirim · Shift+Enter untuk baris baru</span>
              {input.length > 1500 && <span>{input.length}/2000</span>}
            </div>
          </form>
        </section>
      </div>
    </DashboardLayout>
  );
}

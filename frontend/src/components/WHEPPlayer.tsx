'use client';

// =============================================================================
// WHEPPlayer — Pemutar video WebRTC berbasis protokol WHEP (WebRTC-HTTP
// Egress Protocol) yang disediakan MediaMTX.
//
// Pendekatan: 100% browser-native (RTCPeerConnection), TANPA SDK eksternal.
// Alur WHEP signaling:
//   1. Buat RTCPeerConnection + transceiver 'recvonly' (video & audio).
//   2. createOffer() menghasilkan SDP offer.
//   3. Kirim SDP offer via POST /stream/{path}/whep (Content-Type: application/sdp).
//   4. Baca respons sebagai teks (SDP answer) lalu setRemoteDescription.
//   5. pc.ontrack menempelkan stream ke elemen <video>.
//
// Karena endpoint WHEP diproksi Nginx di /stream/* (same-origin dengan aplikasi),
// fetch('/stream/{path}/whep', ...) langsung bekerja bersama cookie sesi.
// =============================================================================

import { useCallback, useEffect, useRef, useState } from 'react';
import {
  Play,
  Pause,
  Maximize2,
  Minimize2,
  LoaderCircle,
  Video,
  VideoOff,
  Wifi,
  WifiOff,
} from 'lucide-react';
import {
  isUnsupportedCodecResponse,
  nextWhepStatus,
  type ConnectionStatus,
} from '@/lib/whep-status';

export type { ConnectionStatus } from '@/lib/whep-status';

export interface WHEPPlayerProps {
  streamPath: string; // contoh: "cam-uuid-here"
  quality?: 'main' | 'sub'; // default: 'sub'
  onStatusChange?: (status: ConnectionStatus) => void;
  onError?: (message: string) => void;
  className?: string;
  showControls?: boolean; // default: true
  autoPlay?: boolean; // default: true
  onQualityChange?: (quality: 'main' | 'sub') => void; // dipicu saat pengguna mengganti kualitas
}

// Batas atas jeda reconnect (exponential backoff).
const MAX_RETRY_DELAY = 30_000;
const BASE_RETRY_DELAY = 2_000;
const FRAME_WAIT_TIMEOUT = 12_000;

// Error codec tidak boleh memicu reconnect tanpa henti: selama encoding kamera
// masih H.265/HEVC, WebRTC browser akan menolak SDP yang sama berulang kali.
class UnsupportedCodecError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'UnsupportedCodecError';
  }
}

export default function WHEPPlayer({
  streamPath,
  quality = 'main',
  onStatusChange,
  onError,
  className = '',
  showControls = true,
  autoPlay = true,
  onQualityChange,
}: WHEPPlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const pcRef = useRef<RTCPeerConnection | null>(null);
  // Referensi path stream agar handler timeout/ontrack selalu membaca nilai terbaru.
  const streamPathRef = useRef(streamPath);
  const qualityRef = useRef(quality);
  // Penanda apakah komponen sengaja dihentikan (manual/stop) sehingga tidak reconnect.
  const stoppedRef = useRef(false);
  // Penanda apakah komponen sudah di-unmount (cegah setState setelah unmount).
  const unmountedRef = useRef(false);
  const retryTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const frameTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const frameCleanupRef = useRef<(() => void) | null>(null);
  const retryCountRef = useRef(0);
  // Referensi fungsi connect agar scheduleReconnect (yang dideklarasikan lebih
  // dulu) bisa memanggilnya tanpa referensi maju yang memicu peringatan lint.
  const connectRef = useRef<() => Promise<void>>(async () => {});

  const [status, setStatus] = useState<ConnectionStatus>('idle');
  const [isPlaying, setIsPlaying] = useState(autoPlay);
  const [isMuted, setIsMuted] = useState(true);
  const [isFullscreen, setIsFullscreen] = useState(false);

  // Sinkronkan ref dengan prop terbaru agar handler asinkron membaca nilai mutakhir.
  useEffect(() => {
    streamPathRef.current = streamPath;
    qualityRef.current = quality;
  }, [streamPath, quality]);

  // Laporkan perubahan status ke parent (StreamModal) jika disediakan.
  useEffect(() => {
    onStatusChange?.(status);
  }, [status, onStatusChange]);

  const emitError = useCallback(
    (message: string) => {
      onError?.(message);
    },
    [onError]
  );

  const clearFrameWait = useCallback(() => {
    if (frameTimerRef.current) {
      clearTimeout(frameTimerRef.current);
      frameTimerRef.current = null;
    }
    frameCleanupRef.current?.();
    frameCleanupRef.current = null;
  }, []);

  // Bersihkan PeerConnection beserta track-nya. Idempoten & aman dipanggil berulang.
  const cleanupPc = useCallback(() => {
    clearFrameWait();
    const pc = pcRef.current;
    if (pc) {
      pc.ontrack = null;
      pc.onconnectionstatechange = null;
      try {
        pc.getSenders().forEach((s) => s.track?.stop());
        pc.getReceivers().forEach((r) => r.track?.stop());
      } catch {
        // Abaikan error saat membersihkan track.
      }
      pc.close();
      pcRef.current = null;
    }
    // Kosongkan srcObject agar frame terakhir tidak macet di layar.
    if (videoRef.current) {
      videoRef.current.srcObject = null;
    }
  }, [clearFrameWait]);

  const clearRetryTimer = useCallback(() => {
    if (retryTimerRef.current) {
      clearTimeout(retryTimerRef.current);
      retryTimerRef.current = null;
    }
  }, []);

  // Mulai proses reconnect dengan exponential backoff (2s, 4s, 8s, … max 30s).
  const scheduleReconnect = useCallback(() => {
    if (stoppedRef.current || unmountedRef.current) return;

    const attempt = retryCountRef.current;
    const delay = Math.min(BASE_RETRY_DELAY * Math.pow(2, attempt), MAX_RETRY_DELAY);
    retryCountRef.current += 1;

    cleanupPc();
    setStatus((current) => nextWhepStatus(current, 'disconnect'));
    clearRetryTimer();
    retryTimerRef.current = setTimeout(() => {
      retryTimerRef.current = null;
      connectRef.current();
    }, delay);
  }, [cleanupPc, clearRetryTimer]);

  // Logika inti: bangun koneksi WHEP ke MediaMTX.
  const connect = useCallback(async () => {
    if (unmountedRef.current || stoppedRef.current) return;

    const path = streamPathRef.current;
    if (!path) return;

    cleanupPc();
    setStatus((current) => nextWhepStatus(current, 'start'));

    const pc = new RTCPeerConnection();
    pcRef.current = pc;

    // Terima aliran video & audio (media dikirim dari server ke klien).
    try {
      pc.addTransceiver('video', { direction: 'recvonly' });
      pc.addTransceiver('audio', { direction: 'recvonly' });
    } catch {
      emitError('Tidak dapat menyiapkan transceiver media.');
      scheduleReconnect();
      return;
    }

    // Track hanya membuktikan negosiasi media. Status Live baru diberikan
    // setelah elemen video benar-benar mendekode sebuah frame.
    pc.ontrack = (e) => {
      if (unmountedRef.current) return;
      const video = videoRef.current;
      if (!video) return;

      clearFrameWait();
      video.srcObject = e.streams[0] ?? new MediaStream([e.track]);
      setStatus((current) => nextWhepStatus(current, 'track'));

      let cancelled = false;
      let frameCallbackId: number | null = null;
      const markFrameReceived = () => {
        if (
          cancelled ||
          unmountedRef.current ||
          stoppedRef.current ||
          pcRef.current !== pc ||
          video.readyState < HTMLMediaElement.HAVE_CURRENT_DATA
        ) {
          return;
        }
        clearFrameWait();
        retryCountRef.current = 0;
        setStatus((current) => nextWhepStatus(current, 'frame'));
      };

      video.addEventListener('loadeddata', markFrameReceived);
      video.addEventListener('playing', markFrameReceived);
      if ('requestVideoFrameCallback' in video) {
        frameCallbackId = video.requestVideoFrameCallback(() => markFrameReceived());
      }
      frameCleanupRef.current = () => {
        cancelled = true;
        video.removeEventListener('loadeddata', markFrameReceived);
        video.removeEventListener('playing', markFrameReceived);
        if (frameCallbackId !== null && 'cancelVideoFrameCallback' in video) {
          video.cancelVideoFrameCallback(frameCallbackId);
        }
      };
      frameTimerRef.current = setTimeout(() => {
        if (pcRef.current !== pc || stoppedRef.current || unmountedRef.current) return;
        emitError('Koneksi terbentuk, tetapi frame video belum diterima. Mencoba ulang…');
        scheduleReconnect();
      }, FRAME_WAIT_TIMEOUT);
      markFrameReceived();

      if (autoPlay) {
        video.play().catch(() => {
          // Autoplay bisa diblokir browser; video tetap muted sehingga umumnya berhasil.
        });
      }
    };

    // Deteksi putus koneksi dari sisi jaringan.
    pc.onconnectionstatechange = () => {
      if (unmountedRef.current) return;
      if (
        pc.connectionState === 'failed' ||
        pc.connectionState === 'disconnected'
      ) {
        emitError('Koneksi streaming terputus. Mencoba menyambung kembali…');
        scheduleReconnect();
      }
    };

    try {
      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);

      // WHEP signaling: POST SDP offer, balasannya adalah SDP answer (text/plain).
      const res = await fetch(`/stream/${path}/whep`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/sdp' },
        body: offer.sdp,
      });

      if (!res.ok) {
        let detail = '';
        try {
          detail = await res.text();
        } catch {
          // Abaikan kegagalan membaca body; status HTTP tetap dilaporkan.
        }
        if (isUnsupportedCodecResponse(res.status, detail)) {
          throw new UnsupportedCodecError(
            'Codec kamera H.265/HEVC tidak didukung browser. Ubah encoding stream kamera (main/sub) di NVR ke H.264, lalu mulai ulang stream.'
          );
        }
        throw new Error(`MediaMTX membalas dengan HTTP ${res.status}.`);
      }

      const answer = await res.text();
      await pc.setRemoteDescription({ type: 'answer', sdp: answer });
      setStatus((current) => nextWhepStatus(current, 'answer'));
    } catch (e) {
      if (unmountedRef.current || stoppedRef.current) return;
      if (e instanceof UnsupportedCodecError) {
        // Hentikan retry dan tampilkan penyebab yang bisa ditindaklanjuti.
        stoppedRef.current = true;
        cleanupPc();
        setStatus('error');
        emitError(e.message);
        return;
      }
      emitError(
        e instanceof Error ? e.message : 'Gagal membangun koneksi streaming.'
      );
      scheduleReconnect();
    }
  }, [cleanupPc, clearFrameWait, scheduleReconnect, emitError, autoPlay]);

  // Simpan referensi connect terkini untuk dipakai scheduleReconnect.
  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  // Efek utama: bangun koneksi setiap kali streamPath / kualitas berubah.
  useEffect(() => {
    stoppedRef.current = false;
    retryCountRef.current = 0;
    clearRetryTimer();
    if (streamPath) {
      connect();
    } else {
      setStatus('idle');
      cleanupPc();
    }
    return () => {
      stoppedRef.current = true;
      clearRetryTimer();
      cleanupPc();
    };
  }, [streamPath, quality, connect, cleanupPc, clearRetryTimer]);

  // Cleanup penuh saat komponen di-unmount.
  useEffect(() => {
    unmountedRef.current = false;
    const video = videoRef.current;
    return () => {
      unmountedRef.current = true;
      stoppedRef.current = true;
      clearRetryTimer();
      cleanupPc();
      if (video) {
        video.srcObject = null;
      }
    };
  }, [clearRetryTimer, cleanupPc]);

  // Toggle play / pause.
  const togglePlay = () => {
    const v = videoRef.current;
    if (!v) return;
    if (v.paused) {
      v.play().catch(() => {
        /* abaikan */
      });
      setIsPlaying(true);
    } else {
      v.pause();
      setIsPlaying(false);
    }
  };

  // Toggle mute.
  const toggleMute = () => {
    const v = videoRef.current;
    if (!v) return;
    v.muted = !v.muted;
    setIsMuted(v.muted);
  };

  // Toggle fullscreen via Fullscreen API (dengan fallback Safari).
  const toggleFullscreen = async () => {
    const el = containerRef.current;
    if (!el) return;
    try {
      if (!document.fullscreenElement) {
        await el.requestFullscreen();
        setIsFullscreen(true);
      } else {
        await document.exitFullscreen();
        setIsFullscreen(false);
      }
    } catch {
      // Peramban mungkin menolak; abaikan.
    }
  };

  // Dengarkan perubahan mode fullscreen dari luar (mis. tekan Escape).
  useEffect(() => {
    const onFsChange = () => setIsFullscreen(Boolean(document.fullscreenElement));
    document.addEventListener('fullscreenchange', onFsChange);
    return () => document.removeEventListener('fullscreenchange', onFsChange);
  }, []);

  // Toggle kualitas SD ↔ HD; parent akan me-restart stream dengan kualitas baru.
  const toggleQuality = () => {
    const next: 'main' | 'sub' = quality === 'main' ? 'sub' : 'main';
    onQualityChange?.(next);
  };

  const isBusy =
    status === 'signaling' ||
    status === 'connecting' ||
    status === 'waiting-frame' ||
    status === 'reconnecting';
  const isLive = status === 'live';
  const isError = status === 'error';
  const isSub = quality === 'sub';

  return (
    <div
      ref={containerRef}
      className={`relative aspect-video w-full overflow-hidden rounded-2xl border border-border bg-black ${className}`}
    >
      <video
        ref={videoRef}
        autoPlay={autoPlay}
        muted
        playsInline
        className="h-full w-full object-contain"
        onClick={showControls ? togglePlay : undefined}
      />

      {/* Overlay status */}
      {!isLive && (
        <div
          className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center gap-3 bg-black/60 backdrop-blur-sm"
          role="status"
          aria-live="polite"
        >
          {isBusy && (
            <>
              <LoaderCircle className="h-10 w-10 animate-spin text-accent" />
              <span className="text-sm font-semibold text-white">
                {status === 'reconnecting'
                  ? 'Menyambung kembali…'
                  : status === 'signaling'
                    ? 'Menghubungi server streaming…'
                    : status === 'waiting-frame'
                      ? 'Koneksi tersambung, menunggu frame…'
                      : 'Membangun koneksi media…'}
              </span>
            </>
          )}
          {isError && (
            <>
              <VideoOff className="h-10 w-10 text-error" />
              <span className="text-sm font-semibold text-white">Streaming tidak tersedia</span>
            </>
          )}
          {status === 'idle' && (
            <>
              <Video className="h-10 w-10 text-white/60" />
              <span className="text-sm font-semibold text-white/80">Belum ada stream</span>
            </>
          )}
        </div>
      )}

      {/* Badge LIVE */}
      {isLive && (
        <div className="absolute left-3 top-3 flex items-center gap-1.5 rounded-full bg-red-600/90 px-2.5 py-1 text-[11px] font-bold uppercase tracking-wide text-white shadow">
          <span className="relative flex h-2 w-2">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-white opacity-75" />
            <span className="relative inline-flex h-2 w-2 rounded-full bg-white" />
          </span>
          Live
        </div>
      )}

      {/* Badge kualitas SD/HD */}
      <div className="absolute right-3 top-3 rounded-full bg-black/60 px-2.5 py-1 text-[11px] font-bold uppercase tracking-wide text-white backdrop-blur-sm">
        {isSub ? 'SD' : 'HD'}
      </div>

      {/* Kontrol */}
      {showControls && isLive && (
        <div className="absolute inset-x-0 bottom-0 flex items-center justify-between gap-2 bg-gradient-to-t from-black/80 to-transparent px-3 pb-2.5 pt-8 opacity-0 transition-opacity hover:opacity-100">
          <div className="flex items-center gap-1">
            <button
              type="button"
              aria-label={isPlaying ? 'Jeda' : 'Putar'}
              onClick={togglePlay}
              className="flex h-9 w-9 items-center justify-center rounded-lg bg-white/10 text-white backdrop-blur-sm transition-colors hover:bg-white/25"
            >
              {isPlaying ? <Pause className="h-4 w-4" /> : <Play className="h-4 w-4" />}
            </button>
            <button
              type="button"
              aria-label={isMuted ? 'Aktifkan suara' : 'Bisukan'}
              onClick={toggleMute}
              className="flex h-9 w-9 items-center justify-center rounded-lg bg-white/10 text-white backdrop-blur-sm transition-colors hover:bg-white/25"
            >
              {isMuted ? <VideoOff className="h-4 w-4" /> : <Video className="h-4 w-4" />}
            </button>
          </div>

          <div className="flex items-center gap-1">
            <span className="mr-1 flex items-center gap-1 text-[11px] font-semibold text-white">
              {isLive ? <Wifi className="h-3.5 w-3.5 text-emerald-400" /> : <WifiOff className="h-3.5 w-3.5" />}
              {isSub ? 'SD' : 'HD'}
            </span>
            <button
              type="button"
              aria-label="Ganti kualitas"
              onClick={toggleQuality}
              className="flex h-9 items-center justify-center rounded-lg bg-white/10 px-2.5 text-[11px] font-bold text-white backdrop-blur-sm transition-colors hover:bg-white/25"
            >
              {isSub ? 'HD' : 'SD'}
            </button>
            <button
              type="button"
              aria-label={isFullscreen ? 'Keluar fullscreen' : 'Fullscreen'}
              onClick={toggleFullscreen}
              className="flex h-9 w-9 items-center justify-center rounded-lg bg-white/10 text-white backdrop-blur-sm transition-colors hover:bg-white/25"
            >
              {isFullscreen ? <Minimize2 className="h-4 w-4" /> : <Maximize2 className="h-4 w-4" />}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

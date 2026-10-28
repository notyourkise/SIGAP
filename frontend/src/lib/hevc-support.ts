// Deteksi dukungan codec H.265/HEVC di browser.
//
// WebRTC memakai jalur codec yang berbeda dari pemutaran file MP4. Karena itu
// sinyal paling akurat adalah daftar codec pada RTCRtpReceiver.getCapabilities
// ('video'). MediaSource/canPlayType hanya dipakai sebagai sinyal tambahan
// ketika capabilities tidak tersedia (mis. browser lama atau WebView).
//
// Modul ini murni fungsi agar dapat diuji dengan Node tanpa DOM.

export type HevcSupportMethod =
  | 'rtp-capabilities'
  | 'media-source'
  | 'can-play'
  | 'unknown';

export interface HevcSupportResult {
  /** True hanya bila browser dipastikan mendukung HEVC. */
  supported: boolean;
  /** True bila hasil didasarkan pada sinyal browser, bukan tebakan. */
  detected: boolean;
  method: HevcSupportMethod;
}

export interface RtpCodecCapabilityLike {
  mimeType?: string;
}

export interface RtpReceiverLike {
  getCapabilities?: (
    kind: string
  ) => { codecs?: RtpCodecCapabilityLike[] } | undefined;
}

export interface MediaSourceLike {
  isTypeSupported?: (type: string) => boolean;
}

export interface VideoElementLike {
  canPlayType?: (type: string) => string;
}

export interface HevcSupportScope {
  RTCRtpReceiver?: RtpReceiverLike;
  MediaSource?: MediaSourceLike;
  document?: {
    createElement?: (tagName: string) => VideoElementLike;
  };
}

const HEVC_RTP_MIME = /^video\/(h265|hevc)$/i;

// Format MSE yang umum untuk HEVC. Beberapa browser hanya mengenali hvc1,
// sebagian lain hev1; keduanya diperiksa.
const HEVC_MEDIA_TYPES = [
  'video/mp4; codecs="hvc1"',
  'video/mp4; codecs="hvc1.1.6.L93.B0"',
  'video/mp4; codecs="hev1"',
];

export const HEVC_FALLBACK_GUIDANCE =
  'Gunakan browser/perangkat yang mendukung HEVC (mis. Safari di macOS/iOS), ' +
  'atau ubah encoding kamera di NVR ke H.264. Jika sub-stream memakai H.265, ' +
  'coba main stream (HD); bila kedua stream tetap H.265, encoding kamera harus diubah.';

function isHevcMimeType(mimeType: string | undefined): boolean {
  return typeof mimeType === 'string' && HEVC_RTP_MIME.test(mimeType.trim());
}

function mediaTypeSupported(
  predicate: (type: string) => boolean
): boolean {
  try {
    return HEVC_MEDIA_TYPES.some((type) => predicate(type));
  } catch {
    return false;
  }
}

/**
 * Deteksi dukungan HEVC. `scope` dapat diinjeksi pada unit test; secara default
 * memakai globalThis (browser). Aman dipanggil saat SSR karena semua akses
 * bersifat opsional.
 */
export function detectHevcSupport(
  scope: HevcSupportScope = globalThis as unknown as HevcSupportScope
): HevcSupportResult {
  const receiver = scope.RTCRtpReceiver;
  if (receiver?.getCapabilities) {
    try {
      const capabilities = receiver.getCapabilities('video');
      const codecs = capabilities?.codecs;
      if (Array.isArray(codecs) && codecs.length > 0) {
        return {
          supported: codecs.some((codec) => isHevcMimeType(codec?.mimeType)),
          detected: true,
          method: 'rtp-capabilities',
        };
      }
    } catch {
      // Lanjut ke sinyal berikutnya.
    }
  }

  const mediaSource = scope.MediaSource;
  if (mediaSource?.isTypeSupported) {
    const supported = mediaTypeSupported((type) =>
      Boolean(mediaSource.isTypeSupported?.(type))
    );
    // Hasil false dari MSE tidak membuktikan WebRTC tidak mendukung HEVC,
    // jadi hanya hasil positif yang dianggap konklusif.
    if (supported) {
      return { supported: true, detected: true, method: 'media-source' };
    }
  }

  const video = scope.document?.createElement?.('video');
  if (video?.canPlayType) {
    let supported = false;
    try {
      supported = HEVC_MEDIA_TYPES.some(
        (type) => (video.canPlayType?.(type) ?? '') !== ''
      );
    } catch {
      supported = false;
    }
    if (supported) {
      return { supported: true, detected: true, method: 'can-play' };
    }
  }

  return { supported: false, detected: false, method: 'unknown' };
}

/** Cocokkan pesan error codec dari MediaMTX/player. */
export function isHevcCodecError(message: string): boolean {
  return /h\.?\s*265|hevc|codec/i.test(message);
}

/** Bangun pesan yang bisa ditindaklanjuti untuk pengguna. */
export function buildHevcErrorMessage(support: HevcSupportResult): string {
  let detection: string;
  if (support.detected && !support.supported) {
    detection =
      'Browser/perangkat ini terdeteksi tidak mendukung codec H.265/HEVC.';
  } else if (support.detected && support.supported) {
    detection =
      'Browser ini mengiklankan dukungan HEVC, tetapi MediaMTX tetap menolak negosiasi codec.';
  } else {
    detection =
      'Browser/perangkat ini tampaknya tidak mendukung codec H.265/HEVC.';
  }
  return `${detection} ${HEVC_FALLBACK_GUIDANCE}`;
}

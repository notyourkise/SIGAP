export type ConnectionStatus =
  | 'idle'
  | 'signaling'
  | 'connecting'
  | 'waiting-frame'
  | 'live'
  | 'error'
  | 'reconnecting';

export type WHEPEvent = 'start' | 'answer' | 'track' | 'frame' | 'disconnect';

// MediaMTX membalas 400/415 dengan body yang memuat "codecs not supported by
// client" ketika browser tidak mengiklankan codec sumber (mis. H.265/HEVC).
export function isUnsupportedCodecResponse(status: number, body: string): boolean {
  return (status === 400 || status === 415) && /codec/i.test(body);
}

export function nextWhepStatus(
  current: ConnectionStatus,
  event: WHEPEvent
): ConnectionStatus {
  switch (event) {
    case 'start':
      return 'signaling';
    case 'answer':
      return current === 'waiting-frame' || current === 'live'
        ? current
        : 'connecting';
    case 'track':
      return 'waiting-frame';
    case 'frame':
      return 'live';
    case 'disconnect':
      return current === 'idle' ? 'idle' : 'reconnecting';
  }
}

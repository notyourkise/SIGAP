export type ConnectionStatus =
  | 'idle'
  | 'signaling'
  | 'connecting'
  | 'waiting-frame'
  | 'live'
  | 'error'
  | 'reconnecting';

export type WHEPEvent = 'start' | 'answer' | 'track' | 'frame' | 'disconnect';

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

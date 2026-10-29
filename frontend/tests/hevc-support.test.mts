import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildHevcErrorMessage,
  detectHevcSupport,
  isHevcCodecError,
  type HevcSupportScope,
} from '../src/lib/hevc-support.ts';

test('RTP capabilities without H265 are a conclusive unsupported signal', () => {
  const scope: HevcSupportScope = {
    RTCRtpReceiver: {
      getCapabilities: () => ({
        codecs: [
          { mimeType: 'video/H264' },
          { mimeType: 'video/VP8' },
          { mimeType: 'video/rtx' },
        ],
      }),
    },
  };

  assert.deepEqual(detectHevcSupport(scope), {
    supported: false,
    detected: true,
    method: 'rtp-capabilities',
  });
});

test('RTP capabilities recognize H265 and HEVC mime types case-insensitively', () => {
  for (const mimeType of ['video/H265', 'video/h265', 'video/HEVC']) {
    const scope: HevcSupportScope = {
      RTCRtpReceiver: {
        getCapabilities: () => ({ codecs: [{ mimeType }, { mimeType: 'video/H264' }] }),
      },
    };

    assert.deepEqual(detectHevcSupport(scope), {
      supported: true,
      detected: true,
      method: 'rtp-capabilities',
    });
  }
});

test('empty RTP capabilities fall through to MediaSource', () => {
  const scope: HevcSupportScope = {
    RTCRtpReceiver: {
      getCapabilities: () => ({ codecs: [] }),
    },
    MediaSource: {
      isTypeSupported: (type) => type.includes('hvc1'),
    },
  };

  assert.deepEqual(detectHevcSupport(scope), {
    supported: true,
    detected: true,
    method: 'media-source',
  });
});

test('MediaSource false does not produce a false negative', () => {
  const scope: HevcSupportScope = {
    MediaSource: {
      isTypeSupported: () => false,
    },
  };

  assert.deepEqual(detectHevcSupport(scope), {
    supported: false,
    detected: false,
    method: 'unknown',
  });
});

test('canPlayType is used when capabilities and MediaSource are unavailable', () => {
  const scope: HevcSupportScope = {
    document: {
      createElement: () => ({
        canPlayType: (type) => (type.includes('hev1') ? 'probably' : ''),
      }),
    },
  };

  assert.deepEqual(detectHevcSupport(scope), {
    supported: true,
    detected: true,
    method: 'can-play',
  });
});

test('missing browser APIs produce an unknown result instead of throwing', () => {
  assert.deepEqual(detectHevcSupport({}), {
    supported: false,
    detected: false,
    method: 'unknown',
  });
});

test('throwing capability APIs are tolerated', () => {
  const scope: HevcSupportScope = {
    RTCRtpReceiver: {
      getCapabilities: () => {
        throw new Error('not implemented');
      },
    },
    MediaSource: {
      isTypeSupported: () => {
        throw new Error('not implemented');
      },
    },
  };

  assert.deepEqual(detectHevcSupport(scope), {
    supported: false,
    detected: false,
    method: 'unknown',
  });
});

test('codec error detection covers H265, HEVC, and generic codec messages', () => {
  assert.equal(isHevcCodecError('codecs not supported by client'), true);
  assert.equal(isHevcCodecError('H.265/HEVC tidak didukung'), true);
  assert.equal(isHevcCodecError('h265 decoder missing'), true);
  assert.equal(isHevcCodecError('Koneksi streaming terputus.'), false);
});

test('unsupported message explains device guidance and H.264 fallback', () => {
  const message = buildHevcErrorMessage({
    supported: false,
    detected: true,
    method: 'rtp-capabilities',
  });

  assert.match(message, /tidak mendukung codec H\.265\/HEVC/);
  assert.match(message, /Safari/);
  assert.match(message, /H\.264/);
  assert.match(message, /main stream/);
});

test('unknown support uses cautious wording', () => {
  const message = buildHevcErrorMessage({
    supported: false,
    detected: false,
    method: 'unknown',
  });

  assert.match(message, /tampaknya tidak mendukung/);
});

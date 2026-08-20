import assert from 'node:assert/strict';
import test from 'node:test';

import { nextWhepStatus } from '../src/lib/whep-status.ts';

test('track arrival waits for a decoded frame before reporting live', () => {
  assert.equal(nextWhepStatus('connecting', 'track'), 'waiting-frame');
  assert.equal(nextWhepStatus('waiting-frame', 'frame'), 'live');
});

test('signaling and ICE connection remain distinguishable', () => {
  assert.equal(nextWhepStatus('idle', 'start'), 'signaling');
  assert.equal(nextWhepStatus('signaling', 'answer'), 'connecting');
});

test('a late SDP completion cannot regress an already received track', () => {
  assert.equal(nextWhepStatus('waiting-frame', 'answer'), 'waiting-frame');
  assert.equal(nextWhepStatus('live', 'answer'), 'live');
});

test('a network loss never remains live', () => {
  assert.equal(nextWhepStatus('live', 'disconnect'), 'reconnecting');
});

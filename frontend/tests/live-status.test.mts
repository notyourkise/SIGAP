import assert from 'node:assert/strict';
import test from 'node:test';

import { getEffectiveStreamState, getLiveBadge, hasFreshVerification } from '../src/lib/live-status.ts';

test('idle on-demand is shown as standby, not offline', () => {
  const badge = getLiveBadge({
    network_type: 'gsm',
    stream_live: false,
    stream_state: 'idle',
  });
  assert.equal(badge.label, 'Siaga (on-demand)');
  assert.equal(badge.dot, '🔵');
});

test('confirmed source failure remains offline', () => {
  const badge = getLiveBadge({
    network_type: 'gsm',
    stream_live: false,
    stream_state: 'offline',
  });
  assert.equal(badge.label, 'Offline');
  assert.equal(badge.dot, '🔴');
});

test('unknown shows recent activity when available', () => {
  const badge = getLiveBadge({
    network_type: 'gsm',
    stream_live: false,
    stream_state: 'unknown',
    stream_last_active_at: new Date(Date.now() - 2 * 3_600_000).toISOString(),
  });
  assert.equal(badge.label, 'Pernah aktif hari ini');
  assert.equal(badge.dot, '🟡');
});

test('unknown without recent activity stays unknown', () => {
  const badge = getLiveBadge({
    network_type: 'gsm',
    stream_live: false,
    stream_state: 'unknown',
    stream_last_active_at: new Date(Date.now() - 30 * 3_600_000).toISOString(),
  });
  assert.equal(badge.label, 'Tidak diketahui');
  assert.equal(badge.dot, '⚪');
});

test('Metro-E keeps binary probe semantics', () => {
  assert.equal(getLiveBadge({ network_type: 'metro-e', stream_live: true }).label, 'Online (Live)');
  assert.equal(getLiveBadge({ network_type: 'metro-e', stream_live: false }).label, 'Offline');
  assert.equal(getLiveBadge({ network_type: 'metro-e', stream_live: null }).label, 'Tidak diketahui');
});

test('legacy GSM data without stream_state does not invent offline', () => {
  const badge = getLiveBadge({ network_type: 'gsm', stream_live: false });
  assert.equal(badge.label, 'Tidak diketahui');
  assert.equal(badge.dot, '⚪');
});

test('legacy data without network_type keeps old fallback', () => {
  assert.equal(getLiveBadge({ stream_live: false }).label, 'Offline');
  assert.equal(getLiveBadge({ stream_live: null }).label, 'Tidak diketahui');
});

test('explicit live state wins over legacy fields', () => {
  const badge = getLiveBadge({
    network_type: 'gsm',
    stream_live: false,
    stream_state: 'live',
  });
  assert.equal(badge.label, 'Online (Live)');
  assert.equal(badge.dot, '🟢');
});

test('fresh verified live overrides passive idle', () => {
  const badge = getLiveBadge({
    network_type: 'gsm',
    stream_live: false,
    stream_state: 'idle',
    stream_verified_state: 'live',
    stream_verified_at: new Date(Date.now() - 2 * 60_000).toISOString(),
  });
  assert.equal(badge.label, 'Online terverifikasi');
  assert.equal(badge.dot, '🟢');
});

test('fresh verified offline overrides passive idle', () => {
  const badge = getLiveBadge({
    network_type: 'gsm',
    stream_live: false,
    stream_state: 'idle',
    stream_verified_state: 'offline',
    stream_verified_at: new Date(Date.now() - 2 * 60_000).toISOString(),
  });
  assert.equal(badge.label, 'Offline terkonfirmasi');
  assert.equal(badge.dot, '🔴');
});

test('stale verification falls back to passive state', () => {
  const badge = getLiveBadge({
    network_type: 'gsm',
    stream_live: false,
    stream_state: 'idle',
    stream_verified_state: 'live',
    stream_verified_at: new Date(Date.now() - 31 * 60_000).toISOString(),
  });
  assert.equal(badge.label, 'Siaga (on-demand)');
  assert.equal(hasFreshVerification({
    stream_verified_state: 'live',
    stream_verified_at: new Date(Date.now() - 31 * 60_000).toISOString(),
  }), false);
});

test('effective state prefers fresh verification', () => {
  assert.equal(getEffectiveStreamState({
    stream_state: 'idle',
    stream_verified_state: 'live',
    stream_verified_at: new Date().toISOString(),
  }), 'live');
  assert.equal(getEffectiveStreamState({ stream_state: 'idle' }), 'idle');
});

import assert from 'node:assert/strict';
import test from 'node:test';

import type { Site } from '../src/lib/api.ts';
import { ALL_SITES_LABEL, resolveSiteSwitcherState } from '../src/lib/site-switcher.ts';

const sites: Site[] = [
  {
    id: '11111111-1111-1111-1111-111111111111',
    nama: 'KIPP Nusantara',
    kode: 'OIKN',
    alamat: 'Sepaku',
    koordinat_lat: -0.97,
    koordinat_lng: 116.7,
    is_default: true,
  },
  {
    id: '22222222-2222-2222-2222-222222222222',
    nama: 'Pelabuhan Semayang',
    kode: 'SMY',
    alamat: 'Balikpapan',
    koordinat_lat: -1.27,
    koordinat_lng: 116.81,
    is_default: false,
  },
];

test('admin without sigap_site_id cookie sees "Semua Site", not default site name', () => {
  const state = resolveSiteSwitcherState({
    role: 'admin',
    sites,
    activeSiteId: null,
    loading: false,
    hydrated: true,
  });

  assert.equal(state.isAdmin, true);
  assert.equal(state.canSelectAllSites, true);
  assert.equal(state.isAllSitesActive, true);
  assert.equal(state.activeSite, null);
  assert.equal(state.activeLabel, ALL_SITES_LABEL);
});

test('admin with valid sigap_site_id cookie sees selected site and can still select all sites', () => {
  const state = resolveSiteSwitcherState({
    role: 'admin',
    sites,
    activeSiteId: sites[1].id,
    loading: false,
    hydrated: true,
  });

  assert.equal(state.isAllSitesActive, false);
  assert.equal(state.canSelectAllSites, true);
  assert.equal(state.activeSite?.id, sites[1].id);
  assert.equal(state.activeLabel, 'Pelabuhan Semayang');
});

test('admin with stale/unknown cookie falls back safely to "Semua Site"', () => {
  const state = resolveSiteSwitcherState({
    role: 'admin',
    sites,
    activeSiteId: '99999999-9999-9999-9999-999999999999',
    loading: false,
    hydrated: true,
  });

  assert.equal(state.isAllSitesActive, true);
  assert.equal(state.activeLabel, ALL_SITES_LABEL);
});

test('teknisi never gets "Semua Site" option and falls back to default/first site when cookie is absent', () => {
  const state = resolveSiteSwitcherState({
    role: 'teknisi',
    sites,
    activeSiteId: null,
    loading: false,
    hydrated: true,
  });

  assert.equal(state.isAdmin, false);
  assert.equal(state.canSelectAllSites, false);
  assert.equal(state.isAllSitesActive, false);
  assert.equal(state.activeSite?.id, sites[0].id);
  assert.equal(state.activeLabel, 'KIPP Nusantara');

  const noDefaultState = resolveSiteSwitcherState({
    role: 'teknisi',
    sites: [{ ...sites[1], is_default: false }],
    activeSiteId: null,
    loading: false,
    hydrated: true,
  });
  assert.equal(noDefaultState.activeSite?.id, sites[1].id);
  assert.equal(noDefaultState.activeLabel, 'Pelabuhan Semayang');
});

test('pre-hydration render consistently shows loading state to avoid SSR/client mismatch', () => {
  const state = resolveSiteSwitcherState({
    role: 'admin',
    sites,
    activeSiteId: null,
    loading: false,
    hydrated: false,
  });

  assert.equal(state.activeLabel, 'Memuat…');
});

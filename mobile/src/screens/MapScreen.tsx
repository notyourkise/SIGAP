import React, { useCallback, useEffect, useState } from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import MapView, { Marker, type Region } from 'react-native-maps';
import { useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { apiGet, extractError } from '../api/client';
import type { CCTV, CCTVListData } from '../api/types';
import { PrimaryButton } from '../components/ui';
import { theme } from '../theme/colors';
import type { RootStackParamList } from '../navigation/types';

type Nav = NativeStackNavigationProp<RootStackParamList>;

// Pusat IKN Nusantara (KIPP).
const IKN_CENTER = { latitude: -0.9723, longitude: 116.7083, latitudeDelta: 0.15, longitudeDelta: 0.15 };

export default function MapScreen() {
  const navigation = useNavigation<Nav>();
  const [cctvs, setCctvs] = useState<CCTV[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await apiGet<CCTVListData>('/cctv?page=1&limit=1000');
      if (res.status === 'success' && res.data) {
        setCctvs(res.data.rows || []);
      }
    } catch (e) {
      setError(extractError(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const markerColor = (status: string) => {
    const s = (status || '').toUpperCase();
    if (['ACTIVE', 'AKTIF'].includes(s)) return theme.success;
    if (['MAINTENANCE', 'PEMELIHARAAN'].includes(s)) return theme.warning;
    return theme.danger;
  };

  const parseCoord = (val?: string): number | null => {
    const n = parseFloat(val || '');
    return Number.isFinite(n) && n !== 0 ? n : null;
  };

  const markers = cctvs
    .map((c) => ({ cctv: c, lat: parseCoord(c.latitude), lng: parseCoord(c.longitude) }))
    .filter((m) => m.lat !== null && m.lng !== null) as Array<{ cctv: CCTV; lat: number; lng: number }>;

  return (
    <SafeAreaView style={styles.safe}>
      <View style={styles.header}>
        <Text style={styles.title}>Peta Sebaran CCTV</Text>
        <Text style={styles.muted}>{cctvs.length} perangkat</Text>
      </View>
      {error ? <Text style={styles.error}>{error}</Text> : null}
      <View style={styles.mapBox}>
        <MapView
          style={StyleSheet.absoluteFill}
          initialRegion={IKN_CENTER as Region}
        >
          {markers.map((m) => (
            <Marker
              key={m.cctv.id}
              coordinate={{ latitude: m.lat, longitude: m.lng }}
              pinColor={markerColor(m.cctv.status)}
              title={m.cctv.nama}
              description={m.cctv.kode_cctv}
              onCalloutPress={() => navigation.navigate('Stream', { cctv: m.cctv })}
            />
          ))}
        </MapView>
        {loading ? <Text style={styles.loading}>Memuat peta…</Text> : null}
      </View>
      <View style={styles.footer}>
        <PrimaryButton title="Muat Ulang" onPress={() => void load()} />
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: theme.background },
  header: { padding: 20, paddingBottom: 8 },
  title: { color: theme.text, fontSize: 26, fontWeight: '800' },
  muted: { color: theme.muted, fontSize: 13, marginTop: 2 },
  error: { color: theme.danger, fontSize: 13, paddingHorizontal: 20 },
  mapBox: {
    flex: 1,
    margin: 20,
    borderRadius: 16,
    overflow: 'hidden',
    backgroundColor: '#e5e7eb',
  },
  loading: {
    color: '#fff',
    position: 'absolute',
    bottom: 12,
    left: 12,
    backgroundColor: 'rgba(0,0,0,0.5)',
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 8,
  },
  footer: { padding: 20, paddingTop: 0 },
});

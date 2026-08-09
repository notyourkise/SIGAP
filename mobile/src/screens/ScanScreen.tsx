import React, { useEffect, useState } from 'react';
import { ScrollView, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { CameraView, useCameraPermissions } from 'expo-camera';
import { useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { apiPost, extractError } from '../api/client';
import type { CCTV } from '../api/types';
import { Badge, Card, PrimaryButton } from '../components/ui';
import { theme } from '../theme/colors';
import type { RootStackParamList } from '../navigation/types';

type Nav = NativeStackNavigationProp<RootStackParamList>;

export default function ScanScreen() {
  const navigation = useNavigation<Nav>();
  const [permission, requestPermission] = useCameraPermissions();
  const [scanning, setScanning] = useState(true);
  const [scannedCctv, setScannedCctv] = useState<CCTV | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (permission && !permission.granted) {
      void requestPermission();
    }
  }, [permission, requestPermission]);

  // Callback saat QR terdeteksi.
  const handleBarCodeScanned = async ({ data }: { type: string; data: string }) => {
    if (!scanning || loading) return;
    setScanning(false);
    setLoading(true);
    setError('');
    try {
      const res = await apiPost<CCTV>('/cctv/scan', { qr_payload: data });
      if (res.status === 'success' && res.data) {
        setScannedCctv(res.data);
      } else {
        setError(res.message || 'QR tidak dikenali.');
      }
    } catch (e) {
      setError(extractError(e));
    } finally {
      setLoading(false);
    }
  };

  const resetScan = () => {
    setScannedCctv(null);
    setError('');
    setScanning(true);
  };

  const statusColor = (status: string) => {
    const s = (status || '').toUpperCase();
    if (['ACTIVE', 'AKTIF'].includes(s)) return theme.success;
    if (['MAINTENANCE', 'PEMELIHARAAN'].includes(s)) return theme.warning;
    return theme.danger;
  };

  return (
    <SafeAreaView style={styles.safe}>
      <ScrollView contentContainerStyle={styles.content}>
        <Text style={styles.title}>Scan QR CCTV</Text>

        {!permission?.granted ? (
          <Card>
            <Text style={styles.muted}>
              Aplikasi membutuhkan izin kamera untuk memindai QR label CCTV.
            </Text>
            <View style={{ marginTop: 12 }}>
              <PrimaryButton title="Izinkan Kamera" onPress={() => void requestPermission()} />
            </View>
          </Card>
        ) : (
          <>
            <View style={styles.cameraBox}>
              <CameraView
                style={StyleSheet.absoluteFill}
                facing="back"
                barcodeScannerSettings={{ barcodeTypes: ['qr'] }}
                onBarcodeScanned={scanning && !scannedCctv ? handleBarCodeScanned : undefined}
              />
              {loading ? <Text style={styles.cameraHint}>Memverifikasi QR…</Text> : null}
            </View>

            {error ? <Text style={styles.error}>{error}</Text> : null}

            {scannedCctv ? (
              <Card style={{ marginTop: 16 }}>
                <View style={styles.rowBetween}>
                  <Text style={styles.cctvName}>{scannedCctv.nama}</Text>
                  <Badge text={scannedCctv.status} color={statusColor(scannedCctv.status)} />
                </View>
                <Text style={styles.cctvCode}>{scannedCctv.kode_cctv}</Text>
                <Text style={styles.muted}>Lokasi: {scannedCctv.lokasi_area || '-'}</Text>
                <Text style={styles.muted}>Skor kesehatan: {scannedCctv.health_score ?? '-'}</Text>

                <View style={{ marginTop: 16 }}>
                  <PrimaryButton
                    title="Mulai Pengecekan Lapangan"
                    onPress={() => navigation.navigate('Checkup', { cctv: scannedCctv })}
                  />
                </View>
                <View style={{ marginTop: 10 }}>
                  <PrimaryButton title="Pindai Lagi" onPress={resetScan} />
                </View>
              </Card>
            ) : (
              <Text style={styles.hint}>Arahkan kamera ke QR label pada fisik CCTV.</Text>
            )}
          </>
        )}
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: theme.background },
  content: { padding: 20 },
  title: { color: theme.text, fontSize: 26, fontWeight: '800', marginBottom: 16 },
  cameraBox: {
    height: 300,
    borderRadius: 16,
    overflow: 'hidden',
    backgroundColor: '#000',
    alignItems: 'center',
    justifyContent: 'center',
  },
  cameraHint: { color: '#fff', fontSize: 13, padding: 8 },
  hint: { color: theme.muted, textAlign: 'center', marginTop: 16 },
  error: { color: theme.danger, fontSize: 13, marginTop: 12, textAlign: 'center' },
  rowBetween: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  cctvName: { color: theme.text, fontSize: 18, fontWeight: '800', flex: 1 },
  cctvCode: { color: theme.primary, fontSize: 13, marginTop: 2 },
  muted: { color: theme.muted, fontSize: 13, marginTop: 4 },
});

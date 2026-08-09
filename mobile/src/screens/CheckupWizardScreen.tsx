import React, { useEffect, useState } from 'react';
import {
  Alert,
  Image,
  ScrollView,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRoute } from '@react-navigation/native';
import type { RouteProp } from '@react-navigation/native';
import * as Location from 'expo-location';
import * as ImagePicker from 'expo-image-picker';
import { apiClient, apiPost, extractError } from '../api/client';
import type { CCTV, FieldCheckItem } from '../api/types';
import { PrimaryButton, TextField } from '../components/ui';
import { saveDraft } from '../offline/storage';
import { theme } from '../theme/colors';
import type { RootStackParamList } from '../navigation/types';

type Route = RouteProp<RootStackParamList, 'Checkup'>;

// 9 tahap checklist pengecekan (konsisten dengan frontend web).
const CHECKLIST = [
  'Pengecekan jaringan',
  'Pengecekan pasokan listrik',
  'Pembersihan lensa',
  'Pemeriksaan fisik kamera',
  'Pengecekan kualitas gambar',
  'Pengujian perekaman',
  'Pengecekan akses jarak jauh',
  'Pemeriksaan kabel & konektor',
  'Pengecekan lingkungan sekitar',
];

type CheckStatus = 'ok' | 'not_ok';

export default function CheckupWizardScreen() {
  const route = useRoute<Route>();
  const cctv: CCTV | undefined = route.params?.cctv;

  const [checks, setChecks] = useState<Record<string, CheckStatus>>({});
  const [notes, setNotes] = useState('');
  const [location, setLocation] = useState<{ lat: string; lng: string } | null>(null);
  const [photos, setPhotos] = useState<string[]>([]); // base64 sementara
  const [submitting, setSubmitting] = useState(false);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');

  // Ambil izin lokasi & posisi GPS otomatis.
  useEffect(() => {
    (async () => {
      const { status } = await Location.requestForegroundPermissionsAsync();
      if (status !== 'granted') return;
      try {
        const pos = await Location.getCurrentPositionAsync({});
        setLocation({
          lat: pos.coords.latitude.toFixed(6),
          lng: pos.coords.longitude.toFixed(6),
        });
      } catch {
        // Lokasi opsional — lanjut tanpa GPS.
      }
    })();
  }, []);

  const toggleCheck = (name: string, status: CheckStatus) => {
    setChecks((prev) => ({ ...prev, [name]: status }));
  };

  // Ambil foto bukti via kamera HP.
  const takePhoto = async () => {
    const perm = await ImagePicker.requestCameraPermissionsAsync();
    if (!perm.granted) {
      setError('Izin kamera dibutuhkan untuk mengambil foto bukti.');
      return;
    }
    const result = await ImagePicker.launchCameraAsync({
      quality: 0.7,
      base64: true,
    });
    if (!result.canceled && result.assets[0]?.base64) {
      setPhotos((prev) => [...prev, result.assets[0].base64 as string]);
    }
  };

  // Unggah foto ke backend lalu kembalikan daftar URL.
  const uploadPhotos = async (): Promise<string[]> => {
    const urls: string[] = [];
    for (const base64 of photos) {
      const form = new FormData();
      // React Native menerima objek { uri, name, type } sebagai bagian FormData.
      form.append('file', buildUploadPart(base64) as unknown as Blob);
      const res = await apiClient.post<{ status: string; data?: { url: string } }>('/upload', form, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      const url = res.data?.data?.url;
      if (url) urls.push(url);
    }
    return urls;
  };

  const handleSubmit = async () => {
    if (!cctv) {
      setError('Data CCTV tidak ditemukan.');
      return;
    }
    const allOk = CHECKLIST.every((name) => checks[name]);
    if (!allOk) {
      setError('Seluruh 9 tahap checklist wajib diisi (ok / tidak ok).');
      return;
    }
    if (photos.length === 0) {
      setError('Ambil minimal satu foto bukti.');
      return;
    }

    setSubmitting(true);
    setError('');
    setMessage('');

    // Susun payload field-maintenance.
    const checkItems: FieldCheckItem[] = CHECKLIST.map((name) => ({
      name,
      result: checks[name] === 'ok' ? 'ok' : 'not_ok',
      note: '',
    }));

    try {
      const imageUrls = await uploadPhotos();
      const payload = {
        cctv_id: cctv.id,
        notes,
        check_latitude: location?.lat ?? '',
        check_longitude: location?.lng ?? '',
        checks: checkItems,
        images: imageUrls.map((url) => ({ image_url: url })),
      };

      let res;
      try {
        res = await apiPost('/field-maintenance', payload);
      } catch (e) {
        // Offline / gagal — simpan draft ke AsyncStorage.
        await saveDraft(payload);
        setMessage('Koneksi bermasalah. Pengecekan disimpan sebagai draft dan akan disinkronkan saat online.');
        setSubmitting(false);
        return;
      }

      if (res.status === 'success') {
        setMessage('Pengecekan lapangan berhasil dikirim.');
        setChecks({});
        setNotes('');
        setPhotos([]);
      } else {
        setError(res.message || 'Gagal mengirim pengecekan.');
      }
    } catch (e) {
      setError(extractError(e));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <SafeAreaView style={styles.safe}>
      <ScrollView contentContainerStyle={styles.content}>
        <Text style={styles.title}>Pengecekan Lapangan</Text>
        {cctv ? (
          <>
            <Text style={styles.cctv}>{cctv.nama} ({cctv.kode_cctv})</Text>
            <Text style={styles.muted}>Lokasi: {cctv.lokasi_area || '-'}</Text>
          </>
        ) : null}

        <Text style={styles.section}>Checklist 9 Tahap</Text>
        {CHECKLIST.map((name) => (
          <View key={name} style={styles.checkRow}>
            <Text style={styles.checkName}>{name}</Text>
            <View style={styles.checkActions}>
              <TouchableOpacity
                style={[styles.checkBtn, checks[name] === 'ok' && styles.okActive]}
                onPress={() => toggleCheck(name, 'ok')}
              >
                <Text style={[styles.checkBtnText, checks[name] === 'ok' && styles.activeText]}>OK</Text>
              </TouchableOpacity>
              <TouchableOpacity
                style={[styles.checkBtn, checks[name] === 'not_ok' && styles.notOkActive]}
                onPress={() => toggleCheck(name, 'not_ok')}
              >
                <Text style={[styles.checkBtnText, checks[name] === 'not_ok' && styles.activeText]}>Tidak</Text>
              </TouchableOpacity>
            </View>
          </View>
        ))}

        <TextField
          label="Catatan"
          value={notes}
          onChangeText={setNotes}
          placeholder="Catatan tambahan (opsional)"
          multiline
        />

        <View style={styles.sectionHeader}>
          <Text style={styles.section}>Foto Bukti</Text>
          <TouchableOpacity onPress={takePhoto}>
            <Text style={{ color: theme.info, fontWeight: '700' }}>Ambil Foto</Text>
          </TouchableOpacity>
        </View>
        <View style={styles.photoRow}>
          {photos.map((p, i) => (
            <Image key={i} source={{ uri: `data:image/jpeg;base64,${p}` }} style={styles.photo} />
          ))}
          {photos.length === 0 ? <Text style={styles.muted}>Belum ada foto.</Text> : null}
        </View>

        {location ? (
          <Text style={styles.gps}>Lokasi tercatat: {location.lat}, {location.lng}</Text>
        ) : (
          <Text style={styles.muted}>Lokasi GPS: tidak tersedia (opsional).</Text>
        )}

        {error ? <Text style={styles.error}>{error}</Text> : null}
        {message ? <Text style={styles.success}>{message}</Text> : null}

        <View style={{ marginTop: 16 }}>
          <PrimaryButton title="Kirim Pengecekan" onPress={handleSubmit} loading={submitting} />
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

// Membangun bagian FormData untuk unggahan foto (React Native memakai objek
// { uri, name, type } — base64 di-encode sebagai data URI agar tidak perlu
// file fisik sementara).
function buildUploadPart(base64: string): { uri: string; name: string; type: string } {
  return {
    uri: `data:image/jpeg;base64,${base64}`,
    name: `proof-${Date.now()}.jpg`,
    type: 'image/jpeg',
  };
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: theme.background },
  content: { padding: 20 },
  title: { color: theme.text, fontSize: 24, fontWeight: '800', marginBottom: 8 },
  cctv: { color: theme.text, fontSize: 16, fontWeight: '700', marginTop: 4 },
  muted: { color: theme.muted, fontSize: 13, marginTop: 2 },
  section: { color: theme.text, fontSize: 16, fontWeight: '800', marginTop: 20, marginBottom: 8 },
  sectionHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  checkRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    backgroundColor: theme.card,
    borderColor: theme.border,
    borderWidth: 1,
    borderRadius: 12,
    padding: 12,
    marginBottom: 8,
  },
  checkName: { color: theme.text, fontSize: 14, flex: 1, marginRight: 8 },
  checkActions: { flexDirection: 'row', gap: 8 },
  checkBtn: {
    borderRadius: 8,
    borderWidth: 1,
    borderColor: theme.border,
    paddingHorizontal: 14,
    paddingVertical: 8,
  },
  checkBtnText: { color: theme.muted, fontWeight: '700', fontSize: 13 },
  okActive: { backgroundColor: theme.success, borderColor: theme.success },
  notOkActive: { backgroundColor: theme.danger, borderColor: theme.danger },
  activeText: { color: '#fff' },
  photoRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  photo: { width: 72, height: 72, borderRadius: 8 },
  gps: { color: theme.success, fontSize: 13, marginTop: 10 },
  error: { color: theme.danger, fontSize: 13, marginTop: 12 },
  success: { color: theme.success, fontSize: 13, marginTop: 12 },
});

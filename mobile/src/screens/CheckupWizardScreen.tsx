import React, { useCallback, useEffect, useState } from 'react';
import {
  Image,
  ScrollView,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useFocusEffect, useRoute } from '@react-navigation/native';
import type { RouteProp } from '@react-navigation/native';
import * as Location from 'expo-location';
import * as ImagePicker from 'expo-image-picker';
import { apiGet, extractError } from '../api/client';
import type { CCTV, FieldCheckItem } from '../api/types';
import { useAuth } from '../auth/AuthContext';
import { PrimaryButton, TextField } from '../components/ui';
import {
  classifySyncError,
  getDraftsByUser,
  MAX_DRAFT_RETRY_COUNT,
  removeDraft,
  saveDraft,
  syncAllDrafts,
  syncSingleDraft,
} from '../offline/storage';
import { theme } from '../theme/colors';
import type { RootStackParamList } from '../navigation/types';

type Route = RouteProp<RootStackParamList, 'Checkup'>;

// 9 tahap checklist pengecekan (fallback bila template dari server gagal dimuat).
// Sejak Sprint 35 (P12), item sebenarnya dimuat dari endpoint /checklist-templates
// agar dapat dikustomisasi admin tanpa deploy ulang.
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
  const { user } = useAuth();
  const route = useRoute<Route>();
  const cctv: CCTV | undefined = route.params?.cctv;

  // Daftar item checklist aktif dari server; fallback ke CHECKLIST bila gagal.
  const [checklist, setChecklist] = useState<string[]>(CHECKLIST);
  const [checks, setChecks] = useState<Record<string, CheckStatus>>({});
  const [notes, setNotes] = useState('');
  const [location, setLocation] = useState<{ lat: string; lng: string } | null>(null);
  const [photos, setPhotos] = useState<string[]>([]); // base64 sementara
  const [pendingDraftCount, setPendingDraftCount] = useState(0);
  const [syncingDrafts, setSyncingDrafts] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');

  const refreshDraftCount = useCallback(async () => {
    if (!user?.id) {
      setPendingDraftCount(0);
      return;
    }
    const mine = await getDraftsByUser(user.id);
    setPendingDraftCount(
      mine.filter((d) => d.status !== 'failed_permanent' && d.retryCount < MAX_DRAFT_RETRY_COUNT)
        .length
    );
  }, [user?.id]);

  useFocusEffect(
    useCallback(() => {
      void refreshDraftCount();
    }, [refreshDraftCount])
  );

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

  // Muat template checklist aktif dari server (Sprint 35, P12). Bila gagal /
  // kosong, tetap memakai CHECKLIST sebagai fallback agar teknisi tidak terblokir.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await apiGet<{ rows?: { nama_item: string; urutan: number; aktif: boolean }[] }>(
          '/checklist-templates?aktif=true&limit=100'
        );
        if (cancelled) return;
        const rows = res.data?.rows ?? [];
        if (rows.length > 0) {
          const items = [...rows]
            .sort((a, b) => a.urutan - b.urutan)
            .map((r) => r.nama_item);
          setChecklist(items);
        }
      } catch {
        // Gagal memuat template — pertahankan fallback CHECKLIST.
      }
    })();
    return () => {
      cancelled = true;
    };
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

  const removePhoto = (index: number) => {
    setPhotos((prev) => prev.filter((_, idx) => idx !== index));
  };

  const handleSyncPendingDrafts = async () => {
    if (!user?.id) {
      setError('Sesi pengguna tidak valid. Silakan masuk kembali.');
      return;
    }
    setSyncingDrafts(true);
    setError('');
    setMessage('');
    try {
      const res = await syncAllDrafts(user.id);
      await refreshDraftCount();
      if (res.success > 0) {
        setMessage(`${res.success} draf offline berhasil disinkronkan.`);
      }
      if (res.authRequired) {
        setError('Sesi Anda berakhir. Masuk kembali lalu sinkronkan draf offline.');
      } else if (res.failed > 0 || res.permanentFailed > 0) {
        setError('Sebagian draf gagal disinkronkan. Periksa detailnya pada tab Tugas.');
      }
    } catch (e) {
      setError(extractError(e));
    } finally {
      setSyncingDrafts(false);
    }
  };

  const handleSubmit = async () => {
    if (!user?.id) {
      setError('Sesi pengguna tidak valid. Silakan masuk kembali.');
      return;
    }
    if (!cctv) {
      setError('Data CCTV tidak ditemukan.');
      return;
    }
    const allOk = checklist.every((name) => checks[name]);
    if (!allOk) {
      setError('Seluruh tahap checklist wajib diisi (ok / tidak ok).');
      return;
    }
    if (photos.length === 0) {
      setError('Ambil minimal satu foto bukti.');
      return;
    }

    setSubmitting(true);
    setError('');
    setMessage('');

    // Susun item checklist sesuai skema backend /field-maintenance.
    const checkItems: FieldCheckItem[] = checklist.map((name) => ({
      name,
      result: checks[name] === 'ok' ? 'ok' : 'not_ok',
      note: '',
    }));

    // Simpan draf beserta foto base64 ke AsyncStorage SEBELUM menyentuh jaringan
    // agar kehilangan sinyal di tengah unggah foto atau POST tetap dapat dilanjutkan.
    let draft;
    try {
      draft = await saveDraft({
        userId: user.id,
        cctvId: cctv.id,
        cctvName: cctv.nama,
        cctvCode: cctv.kode_cctv,
        notes,
        location,
        checks: checkItems,
        photos,
      });
    } catch (e) {
      setError(extractError(e));
      setSubmitting(false);
      return;
    }

    try {
      await syncSingleDraft(draft);
      setMessage('Pengecekan lapangan berhasil dikirim.');
      setChecks({});
      setNotes('');
      setPhotos([]);
      await refreshDraftCount();
    } catch (e) {
      const classified = classifySyncError(e, extractError);
      if (classified.kind === 'permanent') {
        // Error validasi/izin (400/403/404/409/422): hapus draf agar form
        // dapat langsung diperbaiki oleh teknisi tanpa mengendap di antrean offline.
        await removeDraft(draft.id, user.id);
        setError(classified.message);
      } else if (classified.kind === 'auth') {
        setChecks({});
        setNotes('');
        setPhotos([]);
        setError('Sesi Anda berakhir. Draf tersimpan aman secara lokal — masuk kembali lalu sinkronkan ulang.');
      } else {
        setChecks({});
        setNotes('');
        setPhotos([]);
        setMessage('Koneksi bermasalah. Pengecekan dan foto bukti telah disimpan di draf offline dan akan disinkronkan saat online.');
      }
      await refreshDraftCount();
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

        {pendingDraftCount > 0 ? (
          <View style={styles.draftBanner}>
            <View style={{ flex: 1, marginRight: 10 }}>
              <Text style={styles.draftBannerTitle}>
                {pendingDraftCount} draf offline menunggu sinkronisasi
              </Text>
              <Text style={styles.muted}>
                Draf akan dikirim otomatis saat kembali online atau ketuk Sinkronkan.
              </Text>
            </View>
            <TouchableOpacity
              style={styles.syncBtn}
              onPress={handleSyncPendingDrafts}
              disabled={syncingDrafts}
            >
              <Text style={styles.syncBtnText}>
                {syncingDrafts ? 'Menyinkronkan…' : 'Sinkronkan'}
              </Text>
            </TouchableOpacity>
          </View>
        ) : null}

        <Text style={styles.section}>Checklist {checklist.length} Tahap</Text>
        {checklist.map((name) => (
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
            <View key={i} style={styles.photoWrap}>
              <Image source={{ uri: `data:image/jpeg;base64,${p}` }} style={styles.photo} />
              <TouchableOpacity style={styles.removePhotoBtn} onPress={() => removePhoto(i)}>
                <Text style={styles.removePhotoText}>×</Text>
              </TouchableOpacity>
            </View>
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

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: theme.background },
  content: { padding: 20 },
  title: { color: theme.text, fontSize: 24, fontWeight: '800', marginBottom: 8 },
  cctv: { color: theme.text, fontSize: 16, fontWeight: '700', marginTop: 4 },
  muted: { color: theme.muted, fontSize: 13, marginTop: 2 },
  draftBanner: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    backgroundColor: theme.card,
    borderColor: theme.warning,
    borderWidth: 1,
    borderRadius: 12,
    padding: 12,
    marginTop: 14,
  },
  draftBannerTitle: { color: theme.text, fontSize: 14, fontWeight: '700' },
  syncBtn: {
    backgroundColor: theme.primary,
    borderRadius: 8,
    paddingHorizontal: 12,
    paddingVertical: 8,
  },
  syncBtnText: { color: '#fff', fontWeight: '700', fontSize: 12 },
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
  photoWrap: { position: 'relative' },
  photo: { width: 72, height: 72, borderRadius: 8 },
  removePhotoBtn: {
    position: 'absolute',
    top: -6,
    right: -6,
    width: 22,
    height: 22,
    borderRadius: 11,
    backgroundColor: theme.danger,
    alignItems: 'center',
    justifyContent: 'center',
  },
  removePhotoText: { color: '#fff', fontSize: 14, fontWeight: '800', lineHeight: 18 },
  gps: { color: theme.success, fontSize: 13, marginTop: 10 },
  error: { color: theme.danger, fontSize: 13, marginTop: 12 },
  success: { color: theme.success, fontSize: 13, marginTop: 12 },
});

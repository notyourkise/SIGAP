import React, { useCallback, useEffect, useState } from 'react';
import { AppState, FlatList, Linking, StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useFocusEffect } from '@react-navigation/native';
import { apiGet, extractError } from '../api/client';
import type { TaskItem, TasksData } from '../api/types';
import { useAuth } from '../auth/AuthContext';
import { Badge, Card } from '../components/ui';
import {
  getDraftsByUser,
  MAX_DRAFT_RETRY_COUNT,
  removeDraft,
  syncAllDrafts,
  type FieldDraft,
} from '../offline/storage';
import { theme } from '../theme/colors';

export default function TasksScreen() {
  const { user } = useAuth();
  const [tasks, setTasks] = useState<TaskItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [pendingDrafts, setPendingDrafts] = useState<FieldDraft[]>([]);
  const [syncingDrafts, setSyncingDrafts] = useState(false);
  const [syncMessage, setSyncMessage] = useState('');
  const [syncError, setSyncError] = useState('');

  const refreshDrafts = useCallback(async () => {
    if (!user?.id) {
      setPendingDrafts([]);
      return [] as FieldDraft[];
    }
    const drafts = await getDraftsByUser(user.id);
    setPendingDrafts(drafts);
    return drafts;
  }, [user?.id]);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const res = await apiGet<TasksData>('/dashboard/tasks');
      if (res.status === 'success' && res.data) {
        setTasks(res.data.tasks || []);
      } else {
        setTasks([]);
      }
    } catch (e) {
      setError(extractError(e));
    } finally {
      setLoading(false);
    }
  }, []);

  const syncPendingDrafts = useCallback(
    async (options?: { silent?: boolean }) => {
      if (!user?.id) {
        setPendingDrafts([]);
        return;
      }

      const drafts = await refreshDrafts();
      const retryable = drafts.filter(
        (d) => d.status !== 'failed_permanent' && d.retryCount < MAX_DRAFT_RETRY_COUNT
      );
      if (retryable.length === 0) {
        if (!options?.silent && drafts.some((d) => d.status === 'failed_permanent')) {
          setSyncError('Terdapat draf gagal permanen. Hapus draf tersebut lalu kirim ulang pengecekan bila diperlukan.');
        }
        return;
      }

      setSyncingDrafts(true);
      setSyncError('');
      if (!options?.silent) {
        setSyncMessage('');
      }

      try {
        const res = await syncAllDrafts(user.id);
        await refreshDrafts();

        if (res.success > 0) {
          setSyncMessage(`${res.success} draf pengecekan lapangan berhasil disinkronkan.`);
          void load();
        }
        if (res.authRequired) {
          setSyncError('Sesi Anda berakhir. Masuk kembali lalu sinkronkan draf offline.');
        } else if (!options?.silent && res.failed > 0) {
          setSyncError(`${res.failed} draf masih gagal disinkronkan. Periksa koneksi lalu coba lagi.`);
        } else if (!options?.silent && res.permanentFailed > 0 && res.success === 0) {
          setSyncError(`${res.permanentFailed} draf gagal permanen dan perlu ditinjau atau dihapus.`);
        }
      } catch (e) {
        if (!options?.silent) {
          setSyncError(extractError(e));
        }
      } finally {
        setSyncingDrafts(false);
      }
    },
    [user?.id, refreshDrafts, load]
  );

  useFocusEffect(
    useCallback(() => {
      void load();
      void syncPendingDrafts({ silent: true });
    }, [load, syncPendingDrafts])
  );

  // Sinkronkan ulang draf secara otomatis saat teknisi kembali ke aplikasi
  // setelah konektivitas kembali pulih (M14 #187).
  useEffect(() => {
    const subscription = AppState.addEventListener('change', (nextState) => {
      if (nextState === 'active') {
        void syncPendingDrafts({ silent: true });
      }
    });
    return () => {
      subscription.remove();
    };
  }, [syncPendingDrafts]);

  const handleDiscardDraft = async (draftId: string) => {
    if (!user?.id) return;
    await removeDraft(draftId, user.id);
    await refreshDrafts();
  };

  const handleRefreshAll = useCallback(() => {
    void load();
    void syncPendingDrafts({ silent: true });
  }, [load, syncPendingDrafts]);

  const statusColor = (status: string) => {
    const s = (status || '').toUpperCase();
    if (['ONGOING', 'BERJALAN'].includes(s)) return theme.warning;
    return theme.info;
  };

  const openMap = (task: TaskItem) => {
    // Buka navigasi eksternal ke koordinat CCTV bila tersedia (petunjuk arah).
    if (task.cctv_location) {
      void Linking.openURL(`https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(task.cctv_location)}`);
    }
  };

  const renderItem = ({ item }: { item: TaskItem }) => (
    <Card style={{ marginBottom: 12 }}>
      <View style={styles.rowBetween}>
        <Text style={styles.cctvName}>{item.cctv_name}</Text>
        <Badge text={item.status} color={statusColor(item.status)} />
      </View>
      <Text style={styles.code}>{item.cctv_kode}</Text>
      <Text style={styles.muted}>Tindakan: {item.action_taken}</Text>
      <Text style={styles.muted}>Lokasi: {item.cctv_location || '-'}</Text>
      <Text style={styles.muted}>Jadwal: {item.scheduled_date ? new Date(item.scheduled_date).toLocaleString() : '-'}</Text>
      {item.days_overdue > 0 ? (
        <Text style={styles.overdue}>Terlambat {item.days_overdue} hari</Text>
      ) : null}
      <TouchableOpacity onPress={() => openMap(item)} style={styles.mapLink}>
        <Text style={{ color: theme.info, fontWeight: '700', fontSize: 13 }}>Petunjuk Arah</Text>
      </TouchableOpacity>
    </Card>
  );

  const retryableCount = pendingDrafts.filter(
    (d) => d.status !== 'failed_permanent' && d.retryCount < MAX_DRAFT_RETRY_COUNT
  ).length;

  return (
    <SafeAreaView style={styles.safe}>
      <View style={styles.header}>
        <Text style={styles.title}>Tugas Pemeliharaan</Text>
        <Text style={styles.muted}>Tugas terjadwal / berjalan milik Anda</Text>
      </View>

      {pendingDrafts.length > 0 ? (
        <View style={styles.draftBox}>
          <View style={styles.rowBetween}>
            <Text style={styles.draftTitle}>
              {pendingDrafts.length} Draf Offline Pengecekan
            </Text>
            {retryableCount > 0 ? (
              <TouchableOpacity
                style={styles.syncButton}
                onPress={() => void syncPendingDrafts({ silent: false })}
                disabled={syncingDrafts}
              >
                <Text style={styles.syncButtonText}>
                  {syncingDrafts ? 'Menyinkronkan…' : 'Sinkronkan'}
                </Text>
              </TouchableOpacity>
            ) : null}
          </View>
          {pendingDrafts.map((d) => {
            const permanentlyStuck =
              d.status === 'failed_permanent' || d.retryCount >= MAX_DRAFT_RETRY_COUNT;
            return (
              <View key={d.id} style={styles.draftItem}>
                <View style={{ flex: 1, marginRight: 8 }}>
                  <Text style={styles.draftItemTitle}>
                    {d.cctvName || d.cctvCode || d.cctvId}
                  </Text>
                  <Text style={styles.muted}>
                    {d.photos.length} foto •{' '}
                    {permanentlyStuck
                      ? 'Gagal permanen'
                      : d.status === 'syncing'
                        ? 'Menyinkronkan…'
                        : d.retryCount > 0
                          ? `Menunggu ulang (${d.retryCount}/${MAX_DRAFT_RETRY_COUNT})`
                          : 'Menunggu sinkronisasi'}
                  </Text>
                  {d.lastError ? <Text style={styles.draftErr}>{d.lastError}</Text> : null}
                </View>
                {permanentlyStuck ? (
                  <TouchableOpacity
                    style={styles.discardBtn}
                    onPress={() => void handleDiscardDraft(d.id)}
                  >
                    <Text style={styles.discardText}>Hapus Draf</Text>
                  </TouchableOpacity>
                ) : null}
              </View>
            );
          })}
        </View>
      ) : null}

      {syncMessage ? <Text style={styles.success}>{syncMessage}</Text> : null}
      {syncError ? <Text style={styles.error}>{syncError}</Text> : null}
      {error ? <Text style={styles.error}>{error}</Text> : null}
      <FlatList
        data={tasks}
        keyExtractor={(item) => item.id}
        renderItem={renderItem}
        contentContainerStyle={styles.list}
        refreshing={loading}
        onRefresh={handleRefreshAll}
        ListEmptyComponent={
          <Text style={styles.empty}>{loading ? 'Memuat…' : 'Tidak ada tugas aktif.'}</Text>
        }
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: theme.background },
  header: { padding: 20, paddingBottom: 8 },
  title: { color: theme.text, fontSize: 26, fontWeight: '800' },
  draftBox: {
    marginHorizontal: 20,
    marginTop: 6,
    marginBottom: 8,
    padding: 12,
    borderRadius: 12,
    backgroundColor: theme.card,
    borderColor: theme.warning,
    borderWidth: 1,
  },
  draftTitle: { color: theme.text, fontSize: 14, fontWeight: '800' },
  syncButton: {
    backgroundColor: theme.primary,
    borderRadius: 8,
    paddingHorizontal: 12,
    paddingVertical: 6,
  },
  syncButtonText: { color: '#fff', fontSize: 12, fontWeight: '700' },
  draftItem: {
    marginTop: 8,
    paddingTop: 8,
    borderTopWidth: 1,
    borderTopColor: theme.border,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  draftItemTitle: { color: theme.text, fontSize: 13, fontWeight: '700' },
  draftErr: { color: theme.danger, fontSize: 12, marginTop: 2 },
  discardBtn: {
    borderColor: theme.danger,
    borderWidth: 1,
    borderRadius: 8,
    paddingHorizontal: 10,
    paddingVertical: 5,
  },
  discardText: { color: theme.danger, fontSize: 12, fontWeight: '700' },
  list: { padding: 20, paddingTop: 8 },
  rowBetween: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  cctvName: { color: theme.text, fontSize: 17, fontWeight: '800', flex: 1 },
  code: { color: theme.primary, fontSize: 13, marginTop: 2 },
  muted: { color: theme.muted, fontSize: 13, marginTop: 3 },
  overdue: { color: theme.danger, fontSize: 13, marginTop: 4, fontWeight: '700' },
  mapLink: { marginTop: 10 },
  success: { color: theme.success, fontSize: 13, paddingHorizontal: 20, marginBottom: 4 },
  error: { color: theme.danger, fontSize: 13, paddingHorizontal: 20 },
  empty: { color: theme.muted, textAlign: 'center', marginTop: 40 },
});

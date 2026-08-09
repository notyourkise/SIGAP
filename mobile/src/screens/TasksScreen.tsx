import React, { useCallback, useEffect, useState } from 'react';
import { FlatList, Linking, StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useFocusEffect, useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { apiGet, extractError } from '../api/client';
import type { TaskItem, TasksData } from '../api/types';
import { Badge, Card } from '../components/ui';
import { theme } from '../theme/colors';
import type { RootStackParamList } from '../navigation/types';

type Nav = NativeStackNavigationProp<RootStackParamList>;

export default function TasksScreen() {
  const navigation = useNavigation<Nav>();
  const [tasks, setTasks] = useState<TaskItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

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

  useFocusEffect(
    useCallback(() => {
      void load();
    }, [load])
  );

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

  return (
    <SafeAreaView style={styles.safe}>
      <View style={styles.header}>
        <Text style={styles.title}>Tugas Pemeliharaan</Text>
        <Text style={styles.muted}>Tugas terjadwal / berjalan milik Anda</Text>
      </View>
      {error ? <Text style={styles.error}>{error}</Text> : null}
      <FlatList
        data={tasks}
        keyExtractor={(item) => item.id}
        renderItem={renderItem}
        contentContainerStyle={styles.list}
        refreshing={loading}
        onRefresh={load}
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
  list: { padding: 20, paddingTop: 8 },
  rowBetween: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  cctvName: { color: theme.text, fontSize: 17, fontWeight: '800', flex: 1 },
  code: { color: theme.primary, fontSize: 13, marginTop: 2 },
  muted: { color: theme.muted, fontSize: 13, marginTop: 3 },
  overdue: { color: theme.danger, fontSize: 13, marginTop: 4, fontWeight: '700' },
  mapLink: { marginTop: 10 },
  error: { color: theme.danger, fontSize: 13, paddingHorizontal: 20 },
  empty: { color: theme.muted, textAlign: 'center', marginTop: 40 },
});

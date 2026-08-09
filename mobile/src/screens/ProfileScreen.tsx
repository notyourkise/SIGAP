import React, { useEffect, useState } from 'react';
import { ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useAuth } from '../auth/AuthContext';
import { Card, PrimaryButton } from '../components/ui';
import { apiClient } from '../api/client';
import { theme } from '../theme/colors';
import type { RootStackParamList } from '../navigation/types';

type Nav = NativeStackNavigationProp<RootStackParamList>;

export default function ProfileScreen() {
  const { user, logout } = useAuth();
  const navigation = useNavigation<Nav>();
  const [online, setOnline] = useState<boolean | null>(null);
  const [checking, setChecking] = useState(false);

  const checkConnection = async () => {
    setChecking(true);
    try {
      await apiClient.get('/health', { timeout: 8000 });
      setOnline(true);
    } catch {
      setOnline(false);
    } finally {
      setChecking(false);
    }
  };

  useEffect(() => {
    void checkConnection();
  }, []);

  const handleLogout = async () => {
    await logout();
  };

  return (
    <SafeAreaView style={styles.safe}>
      <ScrollView contentContainerStyle={styles.content}>
        <Text style={styles.title}>Profil Teknisi</Text>

        <Card>
          <Text style={styles.name}>{user?.full_name ?? '-'}</Text>
          <Text style={styles.username}>@{user?.username ?? '-'}</Text>
          <Text style={styles.email}>{user?.email ?? '-'}</Text>
          <View style={styles.roleRow}>
            <Text style={styles.role}>{user?.role === 'admin' ? 'Administrator' : 'Teknisi'}</Text>
          </View>
        </Card>

        <Card style={{ marginTop: 14 }}>
          <View style={styles.row}>
            <Text style={styles.rowLabel}>Status Koneksi</Text>
            <TouchableOpacity onPress={checkConnection}>
              <Text style={{ color: theme.info, fontWeight: '700' }}>
                {checking ? 'Memeriksa…' : online === null ? 'Cek' : online ? 'Terhubung' : 'Offline'}
              </Text>
            </TouchableOpacity>
          </View>
        </Card>

        <TouchableOpacity
          style={[styles.menuItem, { marginTop: 14 }]}
          onPress={() => navigation.navigate('ChangePassword')}
        >
          <Text style={styles.menuText}>Ganti Password</Text>
        </TouchableOpacity>

        <View style={{ marginTop: 20 }}>
          <PrimaryButton title="Keluar" onPress={handleLogout} />
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: theme.background },
  content: { padding: 20 },
  title: { color: theme.text, fontSize: 26, fontWeight: '800', marginBottom: 16 },
  name: { color: theme.text, fontSize: 20, fontWeight: '800' },
  username: { color: theme.primary, fontSize: 14, marginTop: 2 },
  email: { color: theme.muted, fontSize: 13, marginTop: 2 },
  roleRow: { marginTop: 10 },
  role: {
    alignSelf: 'flex-start',
    color: theme.success,
    borderColor: theme.success,
    borderWidth: 1,
    borderRadius: 999,
    paddingHorizontal: 10,
    paddingVertical: 3,
    fontSize: 11,
    fontWeight: '700',
    overflow: 'hidden',
  },
  row: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  rowLabel: { color: theme.text, fontSize: 15 },
  menuItem: {
    backgroundColor: theme.card,
    borderColor: theme.border,
    borderWidth: 1,
    borderRadius: 12,
    padding: 16,
  },
  menuText: { color: theme.text, fontSize: 15, fontWeight: '600' },
});

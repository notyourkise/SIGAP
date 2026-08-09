import React, { useState } from 'react';
import { ScrollView, StyleSheet, Text } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useAuth } from '../auth/AuthContext';
import { PrimaryButton, TextField } from '../components/ui';
import { apiPost, extractError } from '../api/client';
import { theme } from '../theme/colors';

export default function ChangePasswordScreen() {
  const { logout } = useAuth();
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = async () => {
    setError('');
    setMessage('');
    if (!oldPassword || !newPassword) {
      setError('Password lama dan baru wajib diisi.');
      return;
    }
    if (newPassword.length < 8) {
      setError('Password baru minimal 8 karakter.');
      return;
    }
    if (newPassword !== confirm) {
      setError('Konfirmasi password tidak sama.');
      return;
    }
    setLoading(true);
    try {
      const res = await apiPost<{ message?: string }>('/auth/change-password', {
        old_password: oldPassword,
        new_password: newPassword,
      });
      if (res.status === 'success') {
        setMessage('Password berhasil diubah. Silakan login kembali.');
        setTimeout(() => {
          void logout();
        }, 1500);
      } else {
        setError(res.message || 'Gagal mengganti password.');
      }
    } catch (e) {
      setError(extractError(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <SafeAreaView style={styles.safe}>
      <ScrollView contentContainerStyle={styles.content}>
        <Text style={styles.title}>Ganti Password</Text>
        <TextField
          label="Password Lama"
          value={oldPassword}
          onChangeText={setOldPassword}
          secureTextEntry
          placeholder="password lama"
        />
        <TextField
          label="Password Baru"
          value={newPassword}
          onChangeText={setNewPassword}
          secureTextEntry
          placeholder="min. 8 karakter"
        />
        <TextField
          label="Konfirmasi Password Baru"
          value={confirm}
          onChangeText={setConfirm}
          secureTextEntry
          placeholder="ulangi password baru"
        />
        {error ? <Text style={styles.error}>{error}</Text> : null}
        {message ? <Text style={styles.success}>{message}</Text> : null}
        <PrimaryButton title="Simpan Password" onPress={handleSubmit} loading={loading} />
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: theme.background },
  content: { padding: 20 },
  title: { color: theme.text, fontSize: 24, fontWeight: '800', marginBottom: 20 },
  error: { color: theme.danger, fontSize: 13, marginBottom: 12 },
  success: { color: theme.success, fontSize: 13, marginBottom: 12 },
});

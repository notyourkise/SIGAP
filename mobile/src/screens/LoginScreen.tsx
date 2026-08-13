import React, { useState } from 'react';
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useAuth } from '../auth/AuthContext';
import { PrimaryButton, TextField } from '../components/ui';
import { theme } from '../theme/colors';

export default function LoginScreen() {
  const { login } = useAuth();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleLogin = async () => {
    if (!username.trim() || !password) {
      setError('Username dan password wajib diisi.');
      return;
    }
    setLoading(true);
    setError('');
    try {
      await login(username.trim(), password);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Login gagal.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <SafeAreaView style={styles.safe}>
      <KeyboardAvoidingView
        style={{ flex: 1 }}
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      >
        <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
          <View style={styles.header}>
            <Text style={styles.logo}>SIGAP</Text>
            <Text style={styles.tagline}>Sistem Informasi Gangguan &amp; Aset Pusat</Text>
            <Text style={styles.sub}>Aplikasi Lapangan Teknisi</Text>
          </View>

          <View style={styles.form}>
            <TextField
              label="Username / Email"
              value={username}
              onChangeText={setUsername}
              autoCapitalize="none"
              autoCorrect={false}
              placeholder="username atau email"
            />
            <TextField
              label="Password"
              value={password}
              onChangeText={setPassword}
              secureTextEntry
              placeholder="password"
            />
            {error ? <Text style={styles.error}>{error}</Text> : null}
            <PrimaryButton title="Masuk" onPress={handleLogin} loading={loading} />
          </View>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: theme.background },
  content: {
    flexGrow: 1,
    justifyContent: 'center',
    padding: 24,
  },
  header: { alignItems: 'center', marginBottom: 32 },
  logo: {
    color: theme.primary,
    fontSize: 42,
    fontWeight: '900',
    letterSpacing: -1,
  },
  tagline: {
    color: theme.text,
    fontSize: 15,
    textAlign: 'center',
    marginTop: 6,
    maxWidth: 260,
  },
  sub: { color: theme.muted, fontSize: 13, marginTop: 10 },
  form: { width: '100%' },
  error: { color: theme.danger, fontSize: 13, marginBottom: 12 },
});

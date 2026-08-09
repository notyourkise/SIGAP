import React, { useEffect, useState } from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { WebView } from 'react-native-webview';
import { useRoute } from '@react-navigation/native';
import type { RouteProp } from '@react-navigation/native';
import { apiGet, extractError } from '../api/client';
import type { CCTV, StreamURLData } from '../api/types';
import { theme } from '../theme/colors';
import { API_BASE_URL } from '../config';
import type { RootStackParamList } from '../navigation/types';

type Route = RouteProp<RootStackParamList, 'Stream'>;

export default function StreamScreen() {
  const route = useRoute<Route>();
  const cctv: CCTV | undefined = route.params?.cctv;
  const [stream, setStream] = useState<StreamURLData | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!cctv) return;
    (async () => {
      setLoading(true);
      setError('');
      try {
        const res = await apiGet<StreamURLData>(`/streams/${cctv.id}/url`);
        if (res.status === 'success' && res.data) {
          setStream(res.data);
        } else {
          setError(res.message || 'Stream tidak tersedia.');
        }
      } catch (e) {
        setError(extractError(e));
      } finally {
        setLoading(false);
      }
    })();
  }, [cctv]);

  if (!cctv) {
    return (
      <SafeAreaView style={styles.safe}>
        <Text style={styles.error}>Data CCTV tidak ditemukan.</Text>
      </SafeAreaView>
    );
  }

  // WHEP URL relatif → absolut terhadap base API.
  const whepUrl = stream?.whep_url ? `${API_BASE_URL.replace(/\/api$/, '')}${stream.whep_url}` : '';

  return (
    <SafeAreaView style={styles.safe}>
      <View style={styles.header}>
        <Text style={styles.title}>{cctv.nama}</Text>
        <Text style={styles.muted}>{cctv.kode_cctv} · Live View</Text>
      </View>

      {error ? <Text style={styles.error}>{error}</Text> : null}

      {loading ? (
        <Text style={styles.muted}>Memuat stream…</Text>
      ) : stream ? (
        <>
          <View style={styles.playerBox}>
            {whepUrl ? (
              <WebView
                style={StyleSheet.absoluteFill}
                originWhitelist={['*']}
                source={{ html: buildStreamHtml(whepUrl) }}
                allowsInlineMediaPlayback
                mediaPlaybackRequiresUserAction={false}
              />
            ) : (
              <Text style={styles.muted}>Stream tidak aktif untuk CCTV ini.</Text>
            )}
          </View>
          <View style={styles.statusRow}>
            <Text style={styles.muted}>
              {stream.stream_ready ? '● Live' : '● Tidak aktif'}
            </Text>
            <Text style={styles.muted}>Viewers: {stream.viewers}</Text>
          </View>
        </>
      ) : null}
    </SafeAreaView>
  );
}

// Halaman HTML sederhana untuk WebView: mengarah ke WHEP URL backend. Fallback
// dasar untuk monitoring singkat; implementasi WebRTC native dapat ditambahkan
// dengan react-native-webrtc bila diperlukan.
function buildStreamHtml(whepUrl: string): string {
  return `<!DOCTYPE html>
<html>
<head>
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<style>
  body { margin:0; background:#000; display:flex; align-items:center; justify-content:center; height:100vh; }
  video { width:100%; height:100%; object-fit:contain; }
  .msg { color:#fff; font-family:sans-serif; font-size:14px; }
</style>
</head>
<body>
<video id="video" controls autoplay muted playsinline></video>
<script>
  const whepUrl = ${JSON.stringify(whepUrl)};
  const video = document.getElementById('video');
  (async () => {
    try {
      const pc = new RTCPeerConnection();
      pc.ontrack = (e) => { video.srcObject = e.streams[0]; };
      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);
      const res = await fetch(whepUrl, {
        method: 'POST',
        headers: { 'Content-Type': 'application/sdp' },
        body: pc.localDescription.sdp,
      });
      const answer = await res.text();
      await pc.setRemoteDescription({ type: 'answer', sdp: answer });
    } catch (e) {
      document.body.innerHTML = '<p class="msg">Stream tidak dapat dimuat.</p>';
    }
  })();
</script>
</body>
</html>`;
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: theme.background },
  header: { padding: 20, paddingBottom: 8 },
  title: { color: theme.text, fontSize: 24, fontWeight: '800' },
  muted: { color: theme.muted, fontSize: 13, marginTop: 2 },
  error: { color: theme.danger, fontSize: 13, paddingHorizontal: 20, marginTop: 8 },
  playerBox: {
    height: 320,
    margin: 20,
    borderRadius: 16,
    overflow: 'hidden',
    backgroundColor: '#000',
    alignItems: 'center',
    justifyContent: 'center',
  },
  statusRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingHorizontal: 20,
  },
});

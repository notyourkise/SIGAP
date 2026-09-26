# Task 1B — Streaming Runtime Evidence (2026-08-20)

## Scope dan batas klaim

Task ini menguji source dan staging terisolasi. Tidak ada deployment atau perubahan firewall production. Pengujian public UDP, TURN/TLS, aplikasi SIGAP lengkap, dan jaringan seluler eksternal tidak dijalankan.

## Artifact

| Item | Nilai |
|---|---|
| Base source | `bc05911b439da22b3d0240fca57b8abbba9641bb` |
| Candidate branch | `codex/mediamtx-playback` |
| MediaMTX | `bluenviron/mediamtx:1.20.0-ffmpeg` |
| Linux amd64 manifest | `sha256:72d175e1e0dbf1129f28c9728e50e99f2e8f179b0d125aae35894bcfc7d22ded` |
| Fixture | `test-pattern`, H.264 Baseline level 3.1, 640×360, 15 fps |
| Staging | Compose project `sigap_streaming_test`, `/opt/sigap-streaming-test`, seluruh port bind `127.0.0.1` |

## Hasil

| Gate | Evidence aktual | Status |
|---|---|---|
| Image/config | MediaMTX melaporkan v1.20.0; `webrtcAdditionalHosts` environment terurai menjadi array; ICE/TCP listener aktif | PASS |
| RTSP source | API path `test-pattern`: `available=true`, `online=true`, H.264 640×360 Baseline | PASS |
| Path lifecycle | Integration test add → list config → delete → delete ulang lulus; delete kedua tidak menghasilkan error aplikasi | PASS |
| WHEP signaling dan ICE | Session API: `peerConnectionEstablished=true`, state `read`, kandidat lokal `host/tcp/.../8189` | PASS |
| Frame browser | Video `readyState=4`, `paused=false`, `videoWidth=640`, `videoHeight=360`, `currentTime` bergerak | PASS |
| Recovery | Setelah container staging direstart, video kembali `readyState=4`, resolusi tetap 640×360, dan `currentTime` kembali bergerak | PASS |
| Frontend state | Test memastikan event `track` hanya menjadi `waiting-frame`; hanya event decoded frame menjadi `live`; late SDP tidak meregresikan state | PASS |
| Credential exposure | Regression test memastikan response start tidak mengandung field source/URL RTSP | PASS |
| Cleanup | Browser ditutup, tunnel dihentikan, container/network/project staging dan `/opt/sigap-streaming-test` dihapus | PASS |
| Production unchanged | Container production tetap berjalan; `sigap_api` dan `sigap_mediamtx` tetap healthy, image/port production tidak diubah | PASS |
| Public UDP/firewall/TURN/TLS | Tidak diuji sesuai scope | NOT RUN |
| Mobile/external network | Tidak diuji sesuai scope | NOT RUN |

## Traceability

- BB-08: source dan staging terisolasi PASS; acceptance production/public/mobile belum ditutup.
- MT-19: simulator/status source tersedia; matriks Metro-E/GSM dan event UI belum dijalankan.
- MT-21: lifecycle path, playback, frame, dan recovery infrastructure lulus; suite aplikasi lengkap tetap NOT RUN.
- MT-24: decoded-frame gate lulus; snapshot/timeline/fullscreen/pop-out tetap NOT RUN.
- SYS-MT-01: blocker simulator/environment reachable terselesaikan; event-delivery matrix lintas kanal belum lengkap.

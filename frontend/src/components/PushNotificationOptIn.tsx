"use client";

/**
 * PushNotificationOptIn — tombol opt-in notifikasi push browser (Task 8.3).
 *
 * Komponen client yang:
 * 1. Memeriksa dukungan browser (Service Worker + Push API + VAPID public key).
 * 2. Saat diaktifkan: meminta izin (Notification.requestPermission), membuat
 *    subscription PushSubscription via service worker, lalu mengirim object
 *    subscription ke `POST /api/push/subscribe`.
 * 3. Saat dinonaktifkan: memanggil `DELETE /api/push/unsubscribe`.
 *
 * Push hanya berjalan pada Secure Context (https / localhost). Bila browser
 * tidak mendukung, tampil status "Tidak didukung".
 */
import { useCallback, useEffect, useState } from "react";
import {
  Bell,
  BellOff,
  BellRing,
  CheckCircle2,
  LoaderCircle,
} from "lucide-react";
import { apiRequest } from "@/lib/api";

type PushState =
  | "unsupported" // browser/konteks tidak mendukung
  | "unavailable" // VAPID belum dikonfigurasi / tidak valid di backend
  | "loading" // sedang memeriksa / berproses
  | "denied" // izin ditolak
  | "off" // belum berlangganan / dinonaktifkan
  | "on"; // berlangganan aktif

interface PushSubscriptionInfo {
  endpoint: string;
  key_p256dh: string;
  key_auth: string;
}

interface PushConfig {
  public_key: string;
}

function isPushSupported(): boolean {
  return (
    typeof window !== "undefined" &&
    "serviceWorker" in navigator &&
    "PushManager" in window &&
    "Notification" in window
  );
}

// Ubah ArrayBuffer (dari subscription.getKey) menjadi base64 URL-safe string.
function arrayBufferToBase64Url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  bytes.forEach((b) => {
    binary += String.fromCharCode(b);
  });
  return btoa(binary)
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
}

// Baca existing subscription atau buat yang baru.
async function getPushSubscription(reg: ServiceWorkerRegistration, publicKey: Uint8Array<ArrayBuffer>) {
  let subscription = await reg.pushManager.getSubscription();
  if (subscription?.options.applicationServerKey) {
    const currentKey = new Uint8Array(subscription.options.applicationServerKey);
    const sameKey = currentKey.length === publicKey.length && currentKey.every((byte, index) => byte === publicKey[index]);
    if (!sameKey) {
      try {
        await apiRequest("/push/unsubscribe", {
          method: "DELETE",
          body: JSON.stringify({ endpoint: subscription.endpoint }),
        });
      } finally {
        await subscription.unsubscribe();
        subscription = null;
      }
    }
  }
  if (!subscription) {
    subscription = await reg.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: publicKey,
    });
  }
  return subscription;
}

// Helper: VAPID public key (base64url) -> Uint8Array<ArrayBuffer> (format yang
// diminta pushManager.subscribe). Buffer dibuat eksplisit agar sesuai tipe
// BufferSource pada lib TypeScript terbaru.
function urlBase64ToUint8Array(base64String: string): Uint8Array<ArrayBuffer> {
  const normalized = base64String.trim().replace(/^['"]|['"]$/g, "").replace(/=+$/, "");
  if (!normalized || !/^[A-Za-z0-9_-]+$/.test(normalized)) {
    throw new Error("Kunci notifikasi push dari server tidak valid.");
  }
  const padding = "=".repeat((4 - (normalized.length % 4)) % 4);
  const base64 = (normalized + padding).replace(/-/g, "+").replace(/_/g, "/");
  let rawData: string;
  try {
    rawData = atob(base64);
  } catch {
    throw new Error("Kunci notifikasi push dari server tidak valid.");
  }
  if (rawData.length !== 65 || rawData.charCodeAt(0) !== 4) {
    throw new Error("Kunci notifikasi push dari server tidak valid.");
  }
  const buffer = new ArrayBuffer(rawData.length);
  const outputArray = new Uint8Array(buffer);
  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

function toSubscriptionInfo(sub: PushSubscription): PushSubscriptionInfo {
  const key = sub.getKey ? sub.getKey("p256dh") : null;
  const auth = sub.getKey ? sub.getKey("auth") : null;
  return {
    endpoint: sub.endpoint,
    key_p256dh: key ? arrayBufferToBase64Url(key) : "",
    key_auth: auth ? arrayBufferToBase64Url(auth) : "",
  };
}

export default function PushNotificationOptIn() {
  const [state, setState] = useState<PushState>("loading");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [publicKey, setPublicKey] = useState<Uint8Array<ArrayBuffer> | null>(null);

  const loadPublicKey = useCallback(async () => {
    const response = await apiRequest<PushConfig>("/push/public-key");
    if (!response.data?.public_key) {
      throw new Error("Notifikasi push belum dikonfigurasi oleh administrator sistem.");
    }
    const decoded = urlBase64ToUint8Array(response.data.public_key);
    setPublicKey(decoded);
    return decoded;
  }, []);

  const refreshState = useCallback(async () => {
    if (!isPushSupported()) {
      setState("unsupported");
      return;
    }
    try {
      await loadPublicKey();
      const permission = Notification.permission;
      if (permission === "denied") {
        setState("denied");
        return;
      }
      const reg = await navigator.serviceWorker.ready;
      const sub = await reg.pushManager.getSubscription();
      setState(sub ? "on" : "off");
    } catch (err) {
      setState("unavailable");
      setError(err instanceof Error ? err.message : "Konfigurasi notifikasi push belum tersedia.");
    }
  }, [loadPublicKey]);

  useEffect(() => {
    const timer = window.setTimeout(() => void refreshState(), 0);
    return () => window.clearTimeout(timer);
  }, [refreshState]);

  const enable = async () => {
    setBusy(true);
    setError("");
    try {
      if (!isPushSupported()) {
        setState("unsupported");
        return;
      }
      if (Notification.permission !== "granted") {
        const permission = await Notification.requestPermission();
        if (permission !== "granted") {
          setState("denied");
          return;
        }
      }
      const key = publicKey ?? await loadPublicKey();
      const reg = await navigator.serviceWorker.ready;
      const subscription = await getPushSubscription(reg, key);
      const info = toSubscriptionInfo(subscription);

      if (!info.endpoint || !info.key_p256dh || !info.key_auth) {
        throw new Error("Langganan push browser tidak lengkap. Muat ulang halaman lalu coba lagi.");
      }

      await apiRequest("/push/subscribe", {
        method: "POST",
        body: JSON.stringify(info),
      });
      setState("on");
    } catch (err: unknown) {
      setError(
        err instanceof Error
          ? err.message
          : "Gagal mengaktifkan notifikasi push. Periksa izin browser lalu coba lagi."
      );
    } finally {
      setBusy(false);
    }
  };

  const disable = async () => {
    setBusy(true);
    setError("");
    try {
      const reg = await navigator.serviceWorker.ready;
      const subscription = await reg.pushManager.getSubscription();
      if (subscription) {
        // Beri tahu backend bahwa user opt-out (hapus subscription ini).
        await apiRequest("/push/unsubscribe", {
          method: "DELETE",
          body: JSON.stringify({ endpoint: subscription.endpoint }),
        });
        await subscription.unsubscribe();
      }
      setState("off");
    } catch (err: unknown) {
      setError(
        err instanceof Error
          ? err.message
          : "Gagal menonaktifkan notifikasi push. Coba lagi."
      );
    } finally {
      setBusy(false);
    }
  };

  if (state === "unsupported") {
    return (
      <p className="text-xs text-muted">
        Browser/perangkat ini tidak mendukung notifikasi push. Gunakan browser
        modern (Chrome/Edge) pada koneksi aman (https).
      </p>
    );
  }

  if (state === "unavailable") {
    return (
      <div className="space-y-1.5" role="status">
        <p className="flex items-center gap-2 text-sm font-semibold text-foreground">
          <BellOff className="h-4 w-4 text-muted" />
          Notifikasi push belum tersedia
        </p>
        <p className="max-w-[70ch] text-xs leading-5 text-muted">
          {error || "Konfigurasi notifikasi push belum siap."} Notifikasi di dalam aplikasi tetap aktif.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          {state === "on" ? (
            <BellRing className="h-4 w-4 text-success" />
          ) : state === "denied" ? (
            <BellOff className="h-4 w-4 text-error" />
          ) : (
            <Bell className="h-4 w-4 text-muted" />
          )}
          <span className="text-sm font-semibold">Notifikasi push</span>
        </div>

        {state === "on" ? (
          <button
            type="button"
            onClick={disable}
            disabled={busy}
            className="flex items-center gap-2 rounded-xl border border-border px-3.5 py-2 text-xs font-semibold text-muted transition-colors hover:bg-background hover:text-foreground disabled:cursor-wait disabled:opacity-60"
          >
            {busy ? <LoaderCircle className="h-3.5 w-3.5 animate-spin" /> : <BellOff className="h-3.5 w-3.5" />}
            Nonaktifkan
          </button>
        ) : (
          <button
            type="button"
            onClick={enable}
            disabled={busy || state === "denied"}
            className="flex items-center gap-2 rounded-xl bg-primary px-3.5 py-2 text-xs font-semibold text-white shadow-md shadow-primary/20 transition-colors hover:bg-primary-hover disabled:cursor-not-allowed disabled:opacity-60"
          >
            {busy ? <LoaderCircle className="h-3.5 w-3.5 animate-spin" /> : <Bell className="h-3.5 w-3.5" />}
            {busy ? "Memproses…" : state === "denied" ? "Izin ditolak" : "Aktifkan"}
          </button>
        )}
      </div>

      {state === "on" && (
        <p className="flex items-center gap-1.5 text-xs text-success">
          <CheckCircle2 className="h-3.5 w-3.5" />
          Aktif — Anda akan menerima notifikasi push saat ada tugas baru atau pembaruan.
        </p>
      )}
      {state === "denied" && (
        <p className="text-xs text-error">
          Izin notifikasi ditolak di browser. Izinkan lewat pengaturan situs lalu muat ulang halaman.
        </p>
      )}
      {error && <p className="text-xs text-error">{error}</p>}
    </div>
  );
}

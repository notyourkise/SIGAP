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
  | "loading" // sedang memeriksa / berproses
  | "denied" // izin ditolak
  | "off" // belum berlangganan / dinonaktifkan
  | "on"; // berlangganan aktif

interface PushSubscriptionInfo {
  endpoint: string;
  key_p256dh: string;
  key_auth: string;
}

// Mengambil public key VAPID dari server via manifest? VAPID public key tidak
// disediakan backend; pada deployment, key dibundel di frontend. Untuk
// menandai dukungan, kita hanya cek browser Push API + service worker.
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
async function getPushSubscription(reg: ServiceWorkerRegistration) {
  let subscription = await reg.pushManager.getSubscription();
  if (!subscription) {
    subscription = await reg.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(process.env.NEXT_PUBLIC_VAPID_PUBLIC_KEY || ""),
    });
  }
  return subscription;
}

// Helper: VAPID public key (base64url) -> Uint8Array<ArrayBuffer> (format yang
// diminta pushManager.subscribe). Buffer dibuat eksplisit agar sesuai tipe
// BufferSource pada lib TypeScript terbaru.
function urlBase64ToUint8Array(base64String: string): Uint8Array<ArrayBuffer> {
  if (!base64String) return new Uint8Array(new ArrayBuffer(0));
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, "+").replace(/_/g, "/");
  const rawData = atob(base64);
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

  const refreshState = useCallback(async () => {
    if (!isPushSupported()) {
      setState("unsupported");
      return;
    }
    try {
      const permission = Notification.permission;
      if (permission === "denied") {
        setState("denied");
        return;
      }
      const reg = await navigator.serviceWorker.ready;
      const sub = await reg.pushManager.getSubscription();
      setState(sub ? "on" : "off");
    } catch {
      setState("off");
    }
  }, []);

  useEffect(() => {
    void refreshState();
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
      const reg = await navigator.serviceWorker.ready;
      const subscription = await getPushSubscription(reg);
      const info = toSubscriptionInfo(subscription);

      if (!info.endpoint || !info.key_p256dh || !info.key_auth) {
        throw new Error("Subscription push tidak lengkap (VAPID public key belum diatur di frontend?).");
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

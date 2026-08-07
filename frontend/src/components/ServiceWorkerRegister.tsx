"use client";

import { useEffect } from "react";

/**
 * ServiceWorkerRegister — mendaftarkan service worker (Task 8.1 PWA).
 *
 * Komponen ini sengaja dibuat client component ("use client") karena service
 * worker memakai API browser (`navigator.serviceWorker`) yang tidak tersedia
 * saat Server-Side Rendering. Registrasi hanya berjalan di browser (safe), dan
 * hanya pada koneksi aman (https) atau localhost.
 *
 * Scope: installability + offline shell. Service worker meng-cache shell halaman
 * (HTML/CSS/JS) dengan cache-first dan memakai network-first untuk panggilan API.
 */
export default function ServiceWorkerRegister() {
  useEffect(() => {
    // Hanya daftarkan bila didukung dan berada di secure context (https/localhost).
    if (!("serviceWorker" in navigator)) return;

    const register = async () => {
      try {
        await navigator.serviceWorker.register("/sw.js", { scope: "/" });
      } catch (err) {
        // Registrasi gagal tidak boleh merusak aplikasi — cukup catat di konsol.
        console.warn("Registrasi service worker gagal:", err);
      }
    };

    register();
  }, []);

  return null;
}

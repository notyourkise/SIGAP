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
    if (!("serviceWorker" in navigator)) return;

    const register = async () => {
      try {
        // Di development SW tetap didaftarkan agar PushManager dapat diuji,
        // tetapi query `dev=1` membuat sw.js menonaktifkan seluruh caching.
        const scriptUrl = process.env.NODE_ENV === "production" ? "/sw.js" : "/sw.js?dev=1";
        await navigator.serviceWorker.register(scriptUrl, {
          scope: "/",
          updateViaCache: "none",
        });
      } catch (err) {
        // Registrasi gagal tidak boleh merusak aplikasi — cukup catat di konsol.
        console.warn("Registrasi service worker gagal:", err);
      }
    };

    register();
  }, []);

  return null;
}

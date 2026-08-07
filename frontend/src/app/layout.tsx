import type { Metadata, Viewport } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { AuthProvider } from "@/context/AuthContext";
import { ThemeProvider } from "@/context/ThemeContext";
import { WebSocketProvider } from "@/context/WebSocketContext";
import { ToastProvider } from "@/components/Toast";
import { ConfirmProvider } from "@/components/ConfirmDialog";
import ServiceWorkerRegister from "@/components/ServiceWorkerRegister";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "SIGAP - Sistem Identifikasi Gangguan Aset Pusat",
  description: "Sistem Pemantauan Aset CCTV dan Log Pemeliharaan IKN",
  // --- PWA (Task 8.1): web manifest & meta tags ---
  // manifest di-export Next.js -> <link rel="manifest" href="/manifest.json">.
  manifest: "/manifest.json",
  // NOTE: theme-color di-set lewat export `viewport` (konvensi Next.js App Router),
  // bukan di metadata, agar tidak memicu warning "Unsupported metadata themeColor".
  appleWebApp: {
    capable: true,
    title: "SIGAP",
    statusBarStyle: "default",
    startupImage: "/icons/icon-512x512.png",
  },
  // Ikon aplikasi (termasuk apple-touch-icon).
  icons: {
    icon: [
      { url: "/icons/icon-192x192.png", sizes: "192x192", type: "image/png" },
      { url: "/icons/icon-512x512.png", sizes: "512x512", type: "image/png" },
    ],
    apple: [
      { url: "/icons/icon-192x192.png", sizes: "192x192", type: "image/png" },
      { url: "/icons/icon-512x512.png", sizes: "512x512", type: "image/png" },
    ],
  },
};

// viewport & theme-color PWA — di-export terpisah sesuai konvensi Next.js App Router.
export const viewport: Viewport = {
  themeColor: "#467235",
  width: "device-width",
  initialScale: 1,
  viewportFit: "cover",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="id"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
      suppressHydrationWarning
    >
      <body className="min-h-full flex flex-col">
        {/* PWA (Task 8.1): registrasi service worker hanya berjalan di browser. */}
        <ServiceWorkerRegister />
        <ThemeProvider>
          <AuthProvider>
            <WebSocketProvider>
              <ToastProvider>
                <ConfirmProvider>
                  {children}
                </ConfirmProvider>
              </ToastProvider>
            </WebSocketProvider>
          </AuthProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}

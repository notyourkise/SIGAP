import Lottie404 from '@/components/Lottie404';

// Halaman 404 kustom SIGAP — latar putih polos tanpa elemen warna,
// hanya menampilkan file animasi "404 Error.json".
export default function NotFound() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-white px-4 py-12">
      <Lottie404 />
    </main>
  );
}

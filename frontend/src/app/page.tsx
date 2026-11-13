import { cookies } from 'next/headers';
import { redirect } from 'next/navigation';

export default async function HomePage() {
  // Redirect langsung di sisi server berdasarkan status cookie sesi:
  // - belum memiliki sigap_session -> langsung /login (tanpa mendarat/redirect via /dashboard)
  // - sudah memiliki sigap_session -> /dashboard
  const cookieStore = await cookies();
  const session = cookieStore.get('sigap_session')?.value;
  redirect(session ? '/dashboard' : '/login');
}

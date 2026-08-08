import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

const SESSION_COOKIE = 'sigap_session';
const ADMIN_PATHS = ['/audit-logs', '/users', '/field-maintenance', '/schedules', '/kinerja-teknisi', '/ringkasan', '/suku-cadang'];

function redirectToLogin(request: NextRequest) {
  const loginURL = new URL('/login', request.url);
  loginURL.searchParams.set('redirect', request.nextUrl.pathname);
  return NextResponse.redirect(loginURL);
}

function isAdminPath(pathname: string) {
  return ADMIN_PATHS.some((path) => pathname === path || pathname.startsWith(`${path}/`));
}

export async function proxy(request: NextRequest) {
  const session = request.cookies.get(SESSION_COOKIE)?.value;
  if (!session) {
    return redirectToLogin(request);
  }

  if (isAdminPath(request.nextUrl.pathname)) {
    const backendURL = process.env.INTERNAL_BACKEND_URL || 'http://localhost:3000';
    try {
      const response = await fetch(`${backendURL}/api/auth/me`, {
        headers: { cookie: `${SESSION_COOKIE}=${session}` },
        cache: 'no-store',
      });

      if (!response.ok) {
        return redirectToLogin(request);
      }

      const body = (await response.json()) as { data?: { role?: string } };
      if (body.data?.role !== 'admin') {
        return NextResponse.redirect(new URL('/dashboard', request.url));
      }
    } catch {
      return NextResponse.redirect(new URL('/dashboard', request.url));
    }
  }

  return NextResponse.next();
}

export const config = {
  matcher: ['/audit-logs/:path*', '/users/:path*', '/field-maintenance/:path*', '/schedules/:path*', '/kinerja-teknisi/:path*', '/ringkasan/:path*', '/suku-cadang/:path*'],
};

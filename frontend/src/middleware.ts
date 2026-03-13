import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

// Route → allowed roles (empty array = any authenticated user)
const PROTECTED_ROUTES: Record<string, string[]> = {
  '/admin': ['admin', 'super_admin'],
  '/reseller': ['reseller'],
  '/dashboard': ['user', 'admin', 'super_admin', 'reseller'],
}

// Public routes — no auth required
const PUBLIC_ROUTES = ['/login', '/register', '/forgot-password']

/** Decode JWT payload without verification (verification is done by backend) */
function getJWTRole(token: string): string | null {
  try {
    const parts = token.split('.')
    if (parts.length !== 3) return null
    // Pad base64 if needed
    const payload = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const padded = payload + '='.repeat((4 - (payload.length % 4)) % 4)
    const decoded = JSON.parse(Buffer.from(padded, 'base64').toString('utf-8'))
    return decoded.role ?? decoded.Role ?? null
  } catch {
    return null
  }
}

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl

  // Skip Next.js internal routes and static files
  if (
    pathname.startsWith('/_next') ||
    pathname.startsWith('/api') ||
    pathname.includes('.') // static files
  ) {
    return NextResponse.next()
  }

  const token = request.cookies.get('pvp_token')?.value
  const isPublic = PUBLIC_ROUTES.some((r) => pathname === r || pathname.startsWith(r + '/'))

  // No token → redirect to /login (except public routes)
  if (!token) {
    if (isPublic) return NextResponse.next()
    return NextResponse.redirect(new URL('/login', request.url))
  }

  // Redirect authenticated users away from /login
  if (pathname === '/login' || pathname === '/register') {
    const role = getJWTRole(token)
    return NextResponse.redirect(new URL(getHomeByRole(role), request.url))
  }

  // Check role-based protection
  const role = getJWTRole(token)

  for (const [route, allowedRoles] of Object.entries(PROTECTED_ROUTES)) {
    if ((pathname === route || pathname.startsWith(route + '/')) && allowedRoles.length > 0) {
      if (!role || !allowedRoles.includes(role)) {
        // Redirect to the user's home, or /login if no valid role
        return NextResponse.redirect(new URL(getHomeByRole(role), request.url))
      }
      break
    }
  }

  return NextResponse.next()
}

function getHomeByRole(role: string | null): string {
  switch (role) {
    case 'admin':
    case 'super_admin':
      return '/admin'
    case 'reseller':
      return '/reseller'
    default:
      return '/dashboard'
  }
}

export const config = {
  matcher: ['/((?!_next/static|_next/image|favicon.ico).*)'],
}

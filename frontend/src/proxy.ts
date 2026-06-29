import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";
import { parseRoleFromJwt } from "@/lib/jwtRole";
import { getPostAuthPath } from "@/lib/postAuthRedirect";

const PUBLIC_ROUTES = ["/login", "/register"];

export function proxy(request: NextRequest) {
  const token = request.cookies.get("auth-token")?.value;
  const { pathname } = request.nextUrl;

  // Allow landing page for unauthenticated users only
  if (pathname === "/") {
    if (token) {
      // Authenticated user trying to access landing page -> redirect to dashboard
      const role = parseRoleFromJwt(token);
      return NextResponse.redirect(new URL(getPostAuthPath(role), request.url));
    }
    return NextResponse.next();
  }

  const isPublicRoute = PUBLIC_ROUTES.some((route) => pathname.startsWith(route));

  // Case 1: Unauthenticated user trying to access protected routes (or root)
  if (!token && !isPublicRoute) {
    return NextResponse.redirect(new URL("/login", request.url));
  }

  // Case 2: Authenticated user trying to access public auth routes
  if (token && isPublicRoute) {
    const role = parseRoleFromJwt(token);
    return NextResponse.redirect(new URL(getPostAuthPath(role), request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - api (API routes)
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     */
    "/((?!api|_next/static|_next/image|favicon.ico|.*\\.(?:svg|png|jpg|jpeg|gif|webp)$).*)",
  ],
};

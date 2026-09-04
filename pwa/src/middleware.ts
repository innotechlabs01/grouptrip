import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'
export function middleware(req: NextRequest){
  const token = req.cookies.get('access_token')?.value
  const path = req.nextUrl.pathname
  const publicPaths = ['/login','/register']
  if(!token && !publicPaths.includes(path)){
    return NextResponse.redirect(new URL('/login', req.url))
  }
  return NextResponse.next()
}
export const config = { matcher:['/((?!_next/static|_next/image|favicon.ico).*)'] }

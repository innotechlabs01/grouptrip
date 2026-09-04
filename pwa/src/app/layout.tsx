import type { Metadata } from 'next'
import { QueryProvider } from '@/providers/query-provider'
export const metadata: Metadata = { title: 'GroupTrip', description: 'Trip funding PWA' }
export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="es">
      <body>
        <QueryProvider>
        <nav style={{display:'flex',gap:12,padding:12,borderBottom:'1px solid #ddd'}}>
          <a href="/trips">Trips</a>
          <a href="/budget">Budget</a>
          <a href="/decisions">Decisiones</a>
          <a href="/expenses">Gastos</a>
        </nav>
        {children}
        </QueryProvider>
      </body>
    </html>
  )
}

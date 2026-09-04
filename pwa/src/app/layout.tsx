import type { Metadata } from 'next'
export const metadata: Metadata = { title: 'GroupTrip', description: 'Trip funding PWA' }
export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="es">
      <body>
        <nav style={{display:'flex',gap:12,padding:12,borderBottom:'1px solid #ddd'}}>
          <a href="/trips">Trips</a>
          <a href="/budget">Budget</a>
          <a href="/decisions">Decisiones</a>
          <a href="/expenses">Gastos</a>
        </nav>
        {children}
      </body>
    </html>
  )
}

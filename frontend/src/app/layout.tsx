import { Inter } from 'next/font/google'
import './globals.css'
import dynamic from 'next/dynamic'

// Dynamically import tooltip to avoid SSR issues
const ClientTooltip = dynamic(() => import('@/components/ui/ClientTooltip'), { 
  ssr: false 
})

const inter = Inter({ subsets: ['latin'] })

export const metadata = {
  title: 'Nanit Dashboard',
  description: 'Nanit Home Assistant Bridge Dashboard',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body className={`${inter.className} bg-nanit-gray-50 min-h-screen`}>
        {children}
        <ClientTooltip />
      </body>
    </html>
  )
}
import MainLayout from '@/components/layout/MainLayout'
import Settings from '@/components/Settings'

// Add metadata export for Next.js App Router
export const metadata = {
  title: 'Settings - Nanit Dashboard',
  description: 'Settings for Nanit Dashboard',
}

export default function SettingsPage() {
  return (
    <MainLayout>
      <Settings />
    </MainLayout>
  );
}
import { useState } from 'react';
import { Sidebar } from '@/components/Sidebar';
import { TopHeader } from '@/components/TopHeader';

interface AdminLayoutProps {
  title: string;
  description: string;
  children: React.ReactNode;
}

export function AdminLayout({ title, description, children }: AdminLayoutProps) {
  const [open, setOpen] = useState(false);
  return (
    <div className="min-h-screen bg-background">
      <Sidebar isOpen={open} onClose={() => setOpen(false)} mode="admin" />
      <main className="w-full min-h-screen">
        <TopHeader title={title} description={description} onMenuClick={() => setOpen(true)} />
        <div className="p-4 sm:p-6 pb-20 sm:pb-6">
          {children}
        </div>
      </main>
    </div>
  );
}

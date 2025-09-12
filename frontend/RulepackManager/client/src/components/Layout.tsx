import { useState } from "react";
import { Sidebar } from "@/components/Sidebar";
import { TopHeader } from "@/components/TopHeader";
import { OnboardingNudge } from "@/components/common/OnboardingNudge";

interface LayoutProps {
  children: React.ReactNode;
  title: string;
  description: string;
}

export function Layout({ children, title, description }: LayoutProps) {
  const [sidebarOpen, setSidebarOpen] = useState(false);

  return (
    <div className="min-h-screen bg-background">
      <Sidebar 
        isOpen={sidebarOpen} 
        onClose={() => setSidebarOpen(false)}
      />
      
      <main className="w-full min-h-screen">
        <TopHeader 
          title={title}
          description={description}
          onMenuClick={() => setSidebarOpen(true)}
        />
        <div className="p-4 sm:p-6 pb-20 sm:pb-6">
          <OnboardingNudge />
          {children}
        </div>
      </main>
    </div>
  );
}

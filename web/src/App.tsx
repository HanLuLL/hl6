import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { useAuth } from "@/hooks/use-auth";
import { RootLayout } from "@/components/layout/root-layout";
import { Skeleton } from "@/components/ui/skeleton";
import LandingPage from "@/pages/landing";
import CallbackPage from "@/pages/callback";
import DashboardPage from "@/pages/dashboard";
import DomainsPage from "@/pages/domains";
import SubdomainsPage from "@/pages/subdomains";
import SubdomainDetailPage from "@/pages/subdomain-detail";
import CreditsPage from "@/pages/credits";
import AdminDomainsPage from "@/pages/admin/domains";
import AdminUsersPage from "@/pages/admin/users";
import AdminAuditLogsPage from "@/pages/admin/audit-logs";
import AdminGroupsPage from "@/pages/admin/groups";
import AdminSettingsPage from "@/pages/admin/settings";

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="flex min-h-screen">
        <aside className="hidden w-64 border-r bg-sidebar-background lg:block">
          <div className="flex h-14 items-center border-b px-4">
            <Skeleton className="h-5 w-28" />
          </div>
          <div className="space-y-2 p-4">
            {[...Array(4)].map((_, i) => (
              <Skeleton key={i} className="h-8 w-full" />
            ))}
          </div>
        </aside>
        <div className="flex flex-1 flex-col">
          <header className="flex h-14 items-center border-b bg-background px-4 lg:px-6">
            <div className="flex-1" />
            <div className="flex items-center gap-3">
              <Skeleton className="h-5 w-10" />
              <Skeleton className="h-8 w-8 rounded-full" />
            </div>
          </header>
          <main className="flex-1 p-4 lg:p-6">
            <div className="space-y-4">
              <Skeleton className="h-7 w-48" />
              <Skeleton className="h-4 w-64" />
            </div>
          </main>
        </div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/" replace />;
  }

  return <RootLayout>{children}</RootLayout>;
}

function AdminRoute({ children }: { children: React.ReactNode }) {
  const { user } = useAuth();
  if (user && user.role !== "admin") {
    return <Navigate to="/dashboard" replace />;
  }
  return <>{children}</>;
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<LandingPage />} />
        <Route path="/callback" element={<CallbackPage />} />
        <Route path="/dashboard" element={<ProtectedRoute><DashboardPage /></ProtectedRoute>} />
        <Route path="/domains" element={<ProtectedRoute><DomainsPage /></ProtectedRoute>} />
        <Route path="/subdomains" element={<ProtectedRoute><SubdomainsPage /></ProtectedRoute>} />
        <Route path="/subdomains/:id" element={<ProtectedRoute><SubdomainDetailPage /></ProtectedRoute>} />
        <Route path="/credits" element={<ProtectedRoute><CreditsPage /></ProtectedRoute>} />
        <Route path="/admin/domains" element={<ProtectedRoute><AdminRoute><AdminDomainsPage /></AdminRoute></ProtectedRoute>} />
        <Route path="/admin/users" element={<ProtectedRoute><AdminRoute><AdminUsersPage /></AdminRoute></ProtectedRoute>} />
        <Route path="/admin/audit-logs" element={<ProtectedRoute><AdminRoute><AdminAuditLogsPage /></AdminRoute></ProtectedRoute>} />
        <Route path="/admin/groups" element={<ProtectedRoute><AdminRoute><AdminGroupsPage /></AdminRoute></ProtectedRoute>} />
        <Route path="/admin/settings" element={<ProtectedRoute><AdminRoute><AdminSettingsPage /></AdminRoute></ProtectedRoute>} />
      </Routes>
    </BrowserRouter>
  );
}

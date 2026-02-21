import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { useAuth } from "@/hooks/use-auth";
import { RootLayout } from "@/components/layout/root-layout";
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

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
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
      </Routes>
    </BrowserRouter>
  );
}

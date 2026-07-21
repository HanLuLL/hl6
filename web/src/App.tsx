import { useEffect } from "react";
import { BrowserRouter, Routes, Route, Navigate, Outlet, useNavigate } from "react-router-dom";
import { useAuth } from "@/hooks/use-auth";
import { RootLayout } from "@/components/layout/root-layout";
import { ErrorBoundary } from "@/components/error-boundary";
import { Skeleton } from "@/components/ui/skeleton";
import LandingPage from "@/pages/landing";
import LoginPage from "@/pages/auth/login";
import RegisterPage from "@/pages/auth/register";
import VerifyEmailPage from "@/pages/auth/verify-email";
import SetPasswordPage from "@/pages/auth/set-password";
import ActivateAccountPage from "@/pages/auth/activate-account";
import ForgotPasswordPage from "@/pages/auth/forgot-password";
import ResetPasswordPage from "@/pages/auth/reset-password";
import DashboardPage from "@/pages/dashboard";
import DomainsPage from "@/pages/domains";
import SubdomainsPage from "@/pages/subdomains";
import SubdomainDetailPage from "@/pages/subdomain-detail";
import CreditsPage from "@/pages/credits";
import ProfilePage from "@/pages/profile";
import SessionsPage from "@/pages/sessions";
import FriendLinksPage from "@/pages/friend-links";
import AdminDomainsPage from "@/pages/admin/domains";
import AdminUsersPage from "@/pages/admin/users";
import AdminAuditLogsPage from "@/pages/admin/audit-logs";
import AdminAuditPage from "@/pages/admin/audit";
import AdminFriendLinksPage from "@/pages/admin/friend-links";
import AdminAIAuditPage from "@/pages/admin/ai-audit";
import AdminEmailLogsPage from "@/pages/admin/email-logs";
import AdminSystemLogsPage from "@/pages/admin/logs";
import BannedPage from "@/pages/banned";

import AdminSettingsPage from "@/pages/admin/settings";
import NotFoundPage from "@/pages/not-found";
import { NativeUpdateGate } from "@/components/client/native-update-gate";
import { setupDeepLinkListener, removeDeepLinkListener } from "@/lib/native-client";
import { isNativeClient } from "@/lib/client-runtime";

function ProtectedRoute() {
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

  // 如果已有缓存的用户数据（isAuthenticated=true），即使正在后台刷新也保持显示
  // 只有在确认未登录时才重定向到登录页
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <RootLayout><ErrorBoundary><Outlet /></ErrorBoundary></RootLayout>;
}

function AdminRoute() {
  const { user } = useAuth();
  // 管理员判定：role=admin 或 所属用户组为管理员组
  const isAdmin = user?.role === "admin" || !!user?.group?.is_admin;
  if (user && !isAdmin) {
    return <Navigate to="/dashboard" replace />;
  }
  return <Outlet />;
}

// Component to handle deep links in native app.
// 注意：认证邮件自 2026-07-21 起不再发送 linyu:// 深链，邮件按钮直接指向 web 目标。
// 此 DeepLinkHandler 保留用于未来其他深链场景（如推送通知、分享扩展等）。
function DeepLinkHandler() {
  const navigate = useNavigate();

  useEffect(() => {
    if (!isNativeClient) return;

    setupDeepLinkListener((path, params) => {
      // Handle deep link paths
      switch (path) {
        case "activate":
          // linyu://activate?token=xxx -> /set-password?token=xxx
          if (params.token) {
            navigate(`/set-password?token=${encodeURIComponent(params.token)}`, { replace: true });
          }
          break;
        case "reset-password":
          // linyu://reset-password?token=xxx -> /reset-password?token=xxx
          if (params.token) {
            navigate(`/reset-password?token=${encodeURIComponent(params.token)}`, { replace: true });
          }
          break;
        default:
          // Unknown deep link, ignore
          break;
      }
    });

    return () => {
      removeDeepLinkListener();
    };
  }, [navigate]);

  return null;
}

export default function App() {
  return (
    <BrowserRouter>
      <NativeUpdateGate />
      <DeepLinkHandler />
      <Routes>
        <Route path="/" element={<LandingPage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route path="/verify-email" element={<VerifyEmailPage />} />
        <Route path="/set-password" element={<SetPasswordPage />} />
        <Route path="/activate-account" element={<ActivateAccountPage />} />
        <Route path="/forgot-password" element={<ForgotPasswordPage />} />
        <Route path="/reset-password" element={<ResetPasswordPage />} />
        <Route element={<ProtectedRoute />}>
          <Route path="/dashboard" element={<DashboardPage />} />
          <Route path="/domains" element={<DomainsPage />} />
          <Route path="/subdomains" element={<SubdomainsPage />} />
          <Route path="/subdomains/:id" element={<SubdomainDetailPage />} />
          <Route path="/credits" element={<CreditsPage />} />
          <Route path="/friend-links" element={<FriendLinksPage />} />
          <Route path="/banned" element={<BannedPage />} />
          <Route path="/profile" element={<ProfilePage />} />
          <Route path="/sessions" element={<SessionsPage />} />
          <Route element={<AdminRoute />}>
            <Route path="/admin/domains" element={<AdminDomainsPage />} />
            <Route path="/admin/cloudflare" element={<Navigate to="/admin/domains?tab=dns-providers" replace />} />
            <Route path="/admin/users" element={<AdminUsersPage />} />
            <Route path="/admin/audit" element={<AdminAuditPage />} />
            <Route path="/admin/audit-rules" element={<Navigate to="/admin/audit?tab=rules" replace />} />
            <Route path="/admin/audit-scans" element={<Navigate to="/admin/audit?tab=history" replace />} />
            <Route path="/admin/audit-violations" element={<Navigate to="/admin/audit?tab=domains" replace />} />
            <Route path="/admin/audit-sites" element={<Navigate to="/admin/audit?tab=domains" replace />} />
            <Route path="/admin/audit-logs" element={<AdminAuditLogsPage />} />
            <Route path="/admin/groups" element={<Navigate to="/admin/users?tab=groups" replace />} />
            <Route path="/admin/settings" element={<AdminSettingsPage />} />
            <Route path="/admin/friend-links" element={<AdminFriendLinksPage />} />
            <Route path="/admin/ai-audit" element={<AdminAIAuditPage />} />
            <Route path="/admin/email-logs" element={<AdminEmailLogsPage />} />
            <Route path="/admin/logs" element={<AdminSystemLogsPage />} />
            <Route path="/admin/notifications" element={<Navigate to="/admin/users?tab=notifications" replace />} />
          </Route>
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
import { Link, useLocation, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useQueryClient, useQuery, useMutation } from "@tanstack/react-query";
import { useAuth } from "@/hooks/use-auth";
import { useBranding } from "@/hooks/use-branding";
import { prefetchRouteData } from "@/lib/prefetch";
import { api, getErrorMessage } from "@/lib/api";
import { PageTransition } from "./page-transition";
import { ThemeToggle } from "./theme-toggle";
import { LanguageToggle } from "./language-toggle";
import { SiteFooter } from "./site-footer";
import { NotificationBell } from "@/components/notification/notification-bell";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet";
import { useState, useEffect } from "react";
import { toast } from "sonner";
import type { BrandingResponse } from "@/types";

const SIDEBAR_COLLAPSED_KEY = "sidebar-collapsed";

const navItems = [
  { labelKey: "nav.dashboard", href: "/dashboard", icon: "M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" },
  { labelKey: "nav.domains", href: "/domains", icon: "M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" },
  { labelKey: "nav.mySubdomains", href: "/subdomains", icon: "M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" },
  { labelKey: "nav.credits", href: "/credits", icon: "M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" },
];

const adminItems = [
  { labelKey: "nav.adminUsers", href: "/admin/users", icon: "M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" },
  { labelKey: "nav.adminDomains", href: "/admin/domains", icon: "M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z" },
  { labelKey: "nav.audit", href: "/admin/audit", icon: "M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z" },
  { labelKey: "nav.auditLogs", href: "/admin/audit-logs", icon: "M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" },
  { labelKey: "nav.adminSettings", href: "/admin/settings", icon: "M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" },
];

function NavLink({ item, onClick, collapsed }: { item: typeof navItems[0]; onClick?: () => void; collapsed?: boolean }) {
  const location = useLocation();
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const active = location.pathname === item.href;
  return (
    <Link
      to={item.href}
      onClick={onClick}
      onMouseEnter={() => prefetchRouteData(queryClient, item.href)}
      onFocus={() => prefetchRouteData(queryClient, item.href)}
      title={collapsed ? t(item.labelKey) : undefined}
      className={`flex items-center rounded-lg text-sm transition-colors ${
        collapsed ? "justify-center px-2 py-2" : "gap-3 px-3 py-2"
      } ${
        active
          ? "bg-primary text-primary-foreground"
          : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
      }`}
    >
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="shrink-0">
        <path d={item.icon} />
      </svg>
      {!collapsed && <span className="truncate">{t(item.labelKey)}</span>}
    </Link>
  );
}

function SidebarContent({ onNavigate, collapsed, branding }: { onNavigate?: () => void; collapsed?: boolean; branding: BrandingResponse }) {
  const { user } = useAuth();
  const { t } = useTranslation();
  const isAdmin = user?.role === "admin";

  return (
    <div className="flex h-full flex-col">
      <div className={`flex h-14 items-center border-b ${collapsed ? "justify-center px-2" : "px-4"}`}>
        <Link to="/dashboard" className="flex items-center gap-2 font-semibold" onClick={onNavigate}>
          {branding.logo_url && (
            <img src={branding.logo_url} alt={branding.name} className="h-5 w-5 shrink-0 rounded-sm object-contain" />
          )}
          {!collapsed && <span>{branding.name}</span>}
        </Link>
      </div>
      <nav className={`flex-1 space-y-1 ${collapsed ? "p-2" : "p-4"}`}>
        {navItems.map((item) => (
          <NavLink key={item.href} item={item} onClick={onNavigate} collapsed={collapsed} />
        ))}
        {isAdmin && (
          <>
            <div className="my-3 border-t" />
            {!collapsed && <p className="mb-2 px-3 text-xs font-medium text-muted-foreground uppercase tracking-wider">{t("nav.admin")}</p>}
            {adminItems.map((item) => (
              <NavLink key={item.href} item={item} onClick={onNavigate} collapsed={collapsed} />
            ))}
          </>
        )}
      </nav>
    </div>
  );
}

export function RootLayout({ children }: { children: React.ReactNode }) {
  const { user, signOut, credits } = useAuth();
  const branding = useBranding();
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [urlConfirmOpen, setUrlConfirmOpen] = useState(false);
  const [collapsed, setCollapsed] = useState(() => {
    return localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === "true";
  });
  const isAdmin = user?.role === "admin";

  const { data: adminConfig } = useQuery({
    queryKey: ["admin-config"],
    queryFn: async () => {
      const res = await api.adminGetConfig();
      return res.data;
    },
    staleTime: 30_000,
    enabled: isAdmin,
  });

  const confirmMutation = useMutation({
    mutationFn: api.adminConfirmUrlConfig,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-config"] });
      toast.success(t("adminSettings.urlConfirmed"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  useEffect(() => {
    localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(collapsed));
  }, [collapsed]);

  const shouldShowUrlPrompt = Boolean(
    isAdmin &&
    adminConfig?.url_runtime &&
    !adminConfig?.url_runtime?.confirmed &&
    location.pathname !== "/admin/settings"
  );

  useEffect(() => {
    setUrlConfirmOpen(shouldShowUrlPrompt);
  }, [shouldShowUrlPrompt]);

  const goToSettings = () => {
    setUrlConfirmOpen(false);
    navigate("/admin/settings");
  };

  return (
    <div className="flex min-h-screen">
      {/* Desktop sidebar */}
      <aside className={`hidden border-r bg-sidebar-background lg:flex lg:flex-col shrink-0 sticky top-0 h-screen transition-all duration-300 ${collapsed ? "w-16" : "w-64"}`}>
        <div className="flex-1 overflow-hidden">
          <SidebarContent collapsed={collapsed} branding={branding} />
        </div>
        <div className={`border-t p-2 ${collapsed ? "flex justify-center" : "flex justify-end"}`}>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setCollapsed(!collapsed)}
            className="h-8 w-8 text-muted-foreground hover:text-foreground"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className={`transition-transform duration-300 ${collapsed ? "rotate-180" : ""}`}>
              <path d="M11 17l-5-5 5-5" />
              <path d="M18 17l-5-5 5-5" />
            </svg>
          </Button>
        </div>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        {/* Header */}
        <header className="flex h-14 items-center gap-4 border-b bg-background px-4 lg:px-6">
          {/* Mobile menu */}
          <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
            <SheetTrigger asChild>
              <Button variant="ghost" size="icon" className="lg:hidden">
                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M3 12h18M3 6h18M3 18h18"/></svg>
              </Button>
            </SheetTrigger>
            <SheetContent side="left" className="w-64 p-0">
              <SidebarContent onNavigate={() => setMobileOpen(false)} branding={branding} />
            </SheetContent>
          </Sheet>

          <div className="flex-1" />

          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
            <span>{credits}</span>
          </div>

          <NotificationBell />

          <LanguageToggle />
          <ThemeToggle />

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="rounded-full">
                <Avatar className="h-8 w-8">
                  <AvatarImage src={user?.avatar_url} />
                  <AvatarFallback>{user?.name?.charAt(0)?.toUpperCase() || "U"}</AvatarFallback>
                </Avatar>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <div className="px-2 py-1.5">
                <p className="text-sm font-medium">{user?.name}</p>
                <p className="text-xs text-muted-foreground">{user?.email}</p>
              </div>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => signOut()}>{t("common.signOut")}</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </header>

        {/* Main content */}
        <main className="flex-1 min-w-0 p-4 lg:p-6">
          <PageTransition>
            {children}
          </PageTransition>
        </main>
        <SiteFooter />
      </div>

      <Dialog open={urlConfirmOpen} onOpenChange={(open) => { if (open) setUrlConfirmOpen(true); }}>
        <DialogContent
          showCloseButton={false}
          onEscapeKeyDown={(e) => {
            e.preventDefault();
            goToSettings();
          }}
          onPointerDownOutside={(e) => e.preventDefault()}
        >
          <DialogHeader>
            <DialogTitle>{t("adminSettings.urlConfirmTitle")}</DialogTitle>
            <DialogDescription>
              {t("adminSettings.urlConfirmDesc")}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2 text-sm">
            <p>
              <span className="font-medium">{t("adminSettings.frontendUrl")}:</span>{" "}
              {adminConfig?.url_runtime.frontend_url}
            </p>
            {(adminConfig?.url_runtime?.frontend_urls?.length ?? 0) > 1 && (
              <p className="text-muted-foreground">
                {adminConfig?.url_runtime?.frontend_urls?.join(" , ")}
              </p>
            )}
            <p>
              <span className="font-medium">{t("adminSettings.backendUrl")}:</span>{" "}
              {adminConfig?.url_runtime.backend_url}
            </p>
            {(adminConfig?.url_runtime?.backend_urls?.length ?? 0) > 1 && (
              <p className="text-muted-foreground">
                {adminConfig?.url_runtime?.backend_urls?.join(" , ")}
              </p>
            )}
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={goToSettings}
            >
              {t("adminSettings.goToSettings")}
            </Button>
            <Button
              onClick={() => confirmMutation.mutate()}
              disabled={confirmMutation.isPending}
              data-dialog-primary="true"
            >
              {confirmMutation.isPending ? t("common.saving") : t("adminSettings.confirmCurrentUrls")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

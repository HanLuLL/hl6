import { Link, useSearchParams } from "react-router-dom";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/hooks/use-auth";
import { useBranding } from "@/hooks/use-branding";
import { useDocumentTitle } from "@/hooks/use-document-title";
import { ThemeToggle } from "@/components/layout/theme-toggle";
import { LanguageToggle } from "@/components/layout/language-toggle";
import { SiteFooter } from "@/components/layout/site-footer";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { api, getErrorMessage } from "@/lib/api";
import { toast } from "sonner";
import type { OIDCStatusPayload } from "@/types";
import { Zap, Globe, Settings2, ArrowRight, Terminal } from "lucide-react";

export default function LandingPage() {
  const { isAuthenticated, signIn } = useAuth();
  const branding = useBranding();
  const { t } = useTranslation();
  useDocumentTitle();
  const [searchParams, setSearchParams] = useSearchParams();
  const ref = searchParams.get("ref") ?? undefined;
  const [dialogOpen, setDialogOpen] = useState(false);
  const [readOnlyPrompt, setReadOnlyPrompt] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [status, setStatus] = useState<OIDCStatusPayload | null>(null);
  const [oidcIssuer, setOidcIssuer] = useState("");
  const [oidcClientID, setOidcClientID] = useState("");
  const [oidcClientSecret, setOidcClientSecret] = useState("");

  useEffect(() => {
    if (searchParams.get("error") === "user_banned") {
      const reason = searchParams.get("reason")?.trim() ?? "";
      sessionStorage.setItem("hl6_banned_notice", "1");
      if (reason) {
        sessionStorage.setItem("hl6_ban_reason", reason);
      } else {
        sessionStorage.removeItem("hl6_ban_reason");
      }

      const nextParams = new URLSearchParams(searchParams);
      nextParams.delete("error");
      nextParams.delete("reason");
      setSearchParams(nextParams, { replace: true });
      return;
    }

    if (sessionStorage.getItem("hl6_banned_notice") !== "1") {
      return;
    }
    sessionStorage.removeItem("hl6_banned_notice");
    const reason = sessionStorage.getItem("hl6_ban_reason")?.trim() ?? "";
    sessionStorage.removeItem("hl6_ban_reason");
    if (reason) {
      toast.error(t("error.userBannedWithReason", { reason }));
      return;
    }
    toast.error(t("error.userBanned"));
  }, [searchParams, setSearchParams, t]);

  const openOIDCDialog = (runtime: OIDCStatusPayload, readOnly: boolean) => {
    setStatus(runtime);
    setReadOnlyPrompt(readOnly);
    setOidcIssuer(runtime.issuer ?? "");
    setOidcClientID(runtime.client_id ?? "");
    setOidcClientSecret("");
    setDialogOpen(true);
  };

  const handleLoginClick = async () => {
    try {
      const res = await api.getOIDCStatus();
      const runtime = res.data;
      if (runtime.configured) {
        signIn(ref);
        return;
      }
      if (runtime.setup_allowed) {
        openOIDCDialog(runtime, false);
        return;
      }
      openOIDCDialog(runtime, true);
    } catch (err) {
      toast.error(getErrorMessage(err, t));
    }
  };

  const submitOIDCSetup = async () => {
    setSubmitting(true);
    try {
      await api.bootstrapOIDCConfig({
        oidc_issuer: oidcIssuer,
        oidc_client_id: oidcClientID,
        oidc_client_secret: oidcClientSecret,
      });
      toast.success(t("oidcSetup.saved"));
      setDialogOpen(false);
      signIn(ref);
    } catch (err) {
      toast.error(getErrorMessage(err, t));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen flex flex-col">
      {/* Sticky nav */}
      <header className="sticky top-0 z-50 backdrop-blur-sm bg-background/80 border-b border-border/50">
        <div className="max-w-6xl mx-auto flex items-center justify-between px-6 h-14">
          <div className="flex items-center gap-2 font-semibold text-base">
            {branding.logo_url && (
              <img src={branding.logo_url} alt={branding.name} className="h-5 w-5 rounded-sm object-contain" />
            )}
            <span>{branding.name}</span>
          </div>
          <div className="flex items-center gap-1.5">
            <LanguageToggle />
            <ThemeToggle />
            {isAuthenticated ? (
              <Button asChild size="sm" className="bg-brand hover:bg-brand/90 text-brand-foreground ml-1">
                <Link to="/dashboard">{t("landing.dashboard")}</Link>
              </Button>
            ) : (
              <Button onClick={handleLoginClick} size="sm" className="bg-brand hover:bg-brand/90 text-brand-foreground ml-1">
                {t("common.signIn")}
              </Button>
            )}
          </div>
        </div>
      </header>

      {/* Hero — asymmetric split */}
      <section className="flex-1 max-w-6xl mx-auto w-full px-6 py-20 md:py-28">
        <div className="grid md:grid-cols-[1fr_auto] gap-12 lg:gap-20 items-start">
          {/* Left column */}
          <div className="max-w-xl">
            <h1 className="text-4xl md:text-5xl lg:text-[3.5rem] font-bold tracking-tight leading-[1.1] text-foreground">
              {t("landing.heroTitle")}
              <span className="relative inline-block text-brand">
                {t("landing.heroHighlight")}
                <span className="absolute -bottom-1 left-0 right-0 h-0.5 bg-brand/30 rounded-full" />
              </span>
            </h1>
            <p className="mt-5 text-base text-muted-foreground leading-relaxed max-w-md">
              {t("landing.heroDesc")}
            </p>
            <div className="mt-8 flex items-center gap-3">
              {isAuthenticated ? (
                <Button size="lg" asChild className="bg-brand hover:bg-brand/90 text-brand-foreground">
                  <Link to="/domains">
                    {t("landing.browseDomains")}
                    <ArrowRight className="ml-2 h-4 w-4" />
                  </Link>
                </Button>
              ) : (
                <Button size="lg" onClick={handleLoginClick} className="bg-brand hover:bg-brand/90 text-brand-foreground">
                  {t("landing.getStarted")}
                  <ArrowRight className="ml-2 h-4 w-4" />
                </Button>
              )}
              <span className="text-xs text-muted-foreground">{t("landing.noCardRequired", "No credit card required")}</span>
            </div>
          </div>

          {/* Right column — terminal code block */}
          <div className="hidden md:block w-[360px] lg:w-[400px] shrink-0">
            <div className="rounded-xl border bg-[oklch(0.10_0_0)] shadow-xl overflow-hidden">
              <div className="flex items-center gap-1.5 px-4 py-3 border-b border-white/10">
                <span className="h-3 w-3 rounded-full bg-[#ff5f57]" />
                <span className="h-3 w-3 rounded-full bg-[#ffbd2e]" />
                <span className="h-3 w-3 rounded-full bg-[#28c840]" />
                <div className="ml-2 flex items-center gap-1.5 text-[11px] text-white/40">
                  <Terminal className="h-3 w-3" />
                  <span>dns records</span>
                </div>
              </div>
              <pre className="px-5 py-4 text-[13px] leading-6 font-mono overflow-x-auto">
                <code>
                  <span className="text-white/40"># subdomain DNS records</span>{"\n"}
                  <span className="text-white/40"># {branding.name || "hl6"} · {new Date().getFullYear()}</span>{"\n\n"}
                  <span className="text-[#7dd3fc]">A</span>
                  <span className="text-white/60">{"     "}</span>
                  <span className="text-[#86efac]">blog</span>
                  <span className="text-white/40">.example.dev</span>
                  <span className="text-white/60">{"  →  "}</span>
                  <span className="text-[#fde68a]">203.0.113.42</span>{"\n"}
                  <span className="text-[#7dd3fc]">AAAA</span>
                  <span className="text-white/60">{"  "}</span>
                  <span className="text-[#86efac]">api</span>
                  <span className="text-white/40">.example.dev</span>
                  <span className="text-white/60">{"   →  "}</span>
                  <span className="text-[#fde68a]">2001:db8::1</span>{"\n"}
                  <span className="text-[#7dd3fc]">CNAME</span>
                  <span className="text-white/60">{"  "}</span>
                  <span className="text-[#86efac]">www</span>
                  <span className="text-white/40">.example.dev</span>
                  <span className="text-white/60">{"   →  "}</span>
                  <span className="text-[#fde68a]">cdn.host.io</span>{"\n"}
                  <span className="text-[#7dd3fc]">TXT</span>
                  <span className="text-white/60">{"    "}</span>
                  <span className="text-[#86efac]">_verify</span>
                  <span className="text-white/40">.example.dev</span>
                  <span className="text-white/60">{"  →  "}</span>
                  <span className="text-[#fde68a]">"v=spf1 ..."</span>{"\n\n"}
                  <span className="text-white/30">{"# ✓ propagated in ~30s"}</span>
                </code>
              </pre>
            </div>
          </div>
        </div>
      </section>

      {/* Features — Bento 2+1 */}
      <section className="py-20 border-t border-border/50">
        <div className="max-w-6xl mx-auto px-6">
          <div className="mb-10">
            <h2 className="text-2xl font-bold tracking-tight">{t("landing.featuresTitle", "Everything you need")}</h2>
            <p className="mt-2 text-muted-foreground">{t("landing.featuresSubtitle", "Built for developers who want DNS without the overhead.")}</p>
          </div>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {/* Wide card */}
            <div className="lg:col-span-2 rounded-xl border p-7 bg-card hover:shadow-md transition-shadow group">
              <div className="mb-4 inline-flex rounded-lg bg-brand-muted p-2.5">
                <Zap className="h-5 w-5 text-brand" />
              </div>
              <h3 className="text-lg font-semibold">{t("landing.featureInstantTitle")}</h3>
              <p className="mt-2 text-sm text-muted-foreground leading-relaxed">{t("landing.featureInstantDesc")}</p>
              <div className="mt-5 flex items-center gap-2 text-xs text-muted-foreground">
                <span className="h-1.5 w-1.5 rounded-full bg-green-500 animate-pulse" />
                {t("landing.featureLive", "Live propagation monitoring")}
              </div>
            </div>
            {/* Narrow card */}
            <div className="rounded-xl border p-7 bg-card hover:shadow-md transition-shadow group">
              <div className="mb-4 inline-flex rounded-lg bg-brand-muted p-2.5">
                <Globe className="h-5 w-5 text-brand" />
              </div>
              <h3 className="text-lg font-semibold">{t("landing.featureCloudflareTitle")}</h3>
              <p className="mt-2 text-sm text-muted-foreground leading-relaxed">{t("landing.featureCloudflareDesc")}</p>
            </div>
            {/* Narrow card */}
            <div className="rounded-xl border p-7 bg-card hover:shadow-md transition-shadow group">
              <div className="mb-4 inline-flex rounded-lg bg-brand-muted p-2.5">
                <Settings2 className="h-5 w-5 text-brand" />
              </div>
              <h3 className="text-lg font-semibold">{t("landing.featureControlTitle")}</h3>
              <p className="mt-2 text-sm text-muted-foreground leading-relaxed">{t("landing.featureControlDesc")}</p>
            </div>
            {/* Stat card */}
            <div className="lg:col-span-2 rounded-xl border p-7 bg-brand/5 border-brand/20 flex items-center justify-between gap-6">
              <div>
                <p className="text-sm font-medium text-brand mb-1">{t("landing.statLabel", "Supported record types")}</p>
                <p className="text-3xl font-bold text-foreground">A · AAAA · CNAME · TXT</p>
                <p className="mt-1.5 text-sm text-muted-foreground">{t("landing.statDesc", "Full DNS record management with validation & conflict detection")}</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* How it works */}
      <section className="py-20 border-t border-border/50">
        <div className="max-w-6xl mx-auto px-6">
          <div className="mb-10">
            <h2 className="text-2xl font-bold tracking-tight">{t("landing.howTitle", "Get started in minutes")}</h2>
          </div>
          <div className="grid md:grid-cols-3 gap-8">
            {[
              { step: "01", title: t("landing.step1Title", "Claim a subdomain"), desc: t("landing.step1Desc", "Browse available domains and claim a subdomain with your account.") },
              { step: "02", title: t("landing.step2Title", "Add DNS records"), desc: t("landing.step2Desc", "Create A, AAAA, CNAME or TXT records directly from the dashboard.") },
              { step: "03", title: t("landing.step3Title", "Go live"), desc: t("landing.step3Desc", "Records propagate automatically via Cloudflare. No manual steps needed.") },
            ].map((s, i) => (
              <div key={i} className="flex gap-5">
                <div className="shrink-0 w-10 h-10 rounded-lg bg-brand-muted flex items-center justify-center">
                  <span className="text-xs font-bold text-brand tabular-nums">{s.step}</span>
                </div>
                <div>
                  <h3 className="font-semibold text-foreground">{s.title}</h3>
                  <p className="mt-1.5 text-sm text-muted-foreground leading-relaxed">{s.desc}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-border/50 mt-auto">
        <div className="max-w-6xl mx-auto px-6 py-8">
          <div className="grid md:grid-cols-3 gap-8 mb-6">
            <div>
              <div className="flex items-center gap-2 font-semibold text-sm mb-2">
                {branding.logo_url && (
                  <img src={branding.logo_url} alt={branding.name} className="h-4 w-4 rounded-sm object-contain" />
                )}
                {branding.name}
              </div>
              <p className="text-xs text-muted-foreground leading-relaxed">
                {t("landing.footerTagline", "Subdomain and DNS management for developers.")}
              </p>
            </div>
            <div>
              <p className="text-xs font-medium text-foreground mb-3 uppercase tracking-wide">{t("landing.footerLinks", "Platform")}</p>
              <ul className="space-y-2">
                {isAuthenticated ? (
                  <>
                    <li><Link to="/dashboard" className="text-xs text-muted-foreground hover:text-foreground transition-colors">{t("nav.dashboard")}</Link></li>
                    <li><Link to="/domains" className="text-xs text-muted-foreground hover:text-foreground transition-colors">{t("nav.domains")}</Link></li>
                    <li><Link to="/credits" className="text-xs text-muted-foreground hover:text-foreground transition-colors">{t("nav.credits")}</Link></li>
                  </>
                ) : (
                  <li>
                    <button onClick={handleLoginClick} className="text-xs text-muted-foreground hover:text-foreground transition-colors">
                      {t("common.signIn")}
                    </button>
                  </li>
                )}
              </ul>
            </div>
            <div>
              <p className="text-xs font-medium text-foreground mb-3 uppercase tracking-wide">{t("landing.footerPrefs", "Preferences")}</p>
              <div className="flex items-center gap-2">
                <LanguageToggle />
                <ThemeToggle />
              </div>
            </div>
          </div>
          <div className="border-t border-border/50 pt-5">
            <SiteFooter withBorder={false} />
          </div>
        </div>
      </footer>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{readOnlyPrompt ? t("oidcSetup.unavailableTitle") : t("oidcSetup.title")}</DialogTitle>
            <DialogDescription>
              {readOnlyPrompt ? t("oidcSetup.unavailableDesc") : t("oidcSetup.desc")}
            </DialogDescription>
          </DialogHeader>

          {readOnlyPrompt ? (
            <div className="space-y-2 text-sm text-muted-foreground">
              <p>{t("oidcSetup.contactAdmin")}</p>
              {(status?.missing_fields?.length ?? 0) > 0 && (
                <p>{t("oidcSetup.missingFields", { fields: status?.missing_fields.join(", ") })}</p>
              )}
            </div>
          ) : (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label>{t("oidcSetup.issuer")}</Label>
                <Input
                  value={oidcIssuer}
                  onChange={(e) => setOidcIssuer(e.target.value)}
                  placeholder="https://issuer.example.com"
                  disabled={status?.issuer_env_locked}
                  required={!status?.issuer_env_locked}
                />
                {status?.issuer_env_locked && (
                  <p className="text-xs text-muted-foreground">{t("oidcSetup.lockedByEnv")}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label>{t("oidcSetup.clientId")}</Label>
                <Input
                  value={oidcClientID}
                  onChange={(e) => setOidcClientID(e.target.value)}
                  placeholder="client-id"
                  disabled={status?.client_id_env_locked}
                  required={!status?.client_id_env_locked}
                />
                {status?.client_id_env_locked && (
                  <p className="text-xs text-muted-foreground">{t("oidcSetup.lockedByEnv")}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label>{t("oidcSetup.clientSecret")}</Label>
                <Input
                  type="password"
                  value={oidcClientSecret}
                  onChange={(e) => setOidcClientSecret(e.target.value)}
                  placeholder="client-secret"
                  disabled={status?.client_secret_env_locked}
                  required={!status?.client_secret_env_locked}
                />
                {status?.client_secret_env_locked && (
                  <p className="text-xs text-muted-foreground">{t("oidcSetup.lockedByEnv")}</p>
                )}
              </div>
            </div>
          )}

          <DialogFooter>
            {readOnlyPrompt ? (
              <Button onClick={() => setDialogOpen(false)}>{t("common.close")}</Button>
            ) : (
              <>
                <Button variant="outline" onClick={() => setDialogOpen(false)}>
                  {t("common.cancel")}
                </Button>
                <Button onClick={submitOIDCSetup} disabled={submitting} data-dialog-primary="true">
                  {submitting ? t("common.saving") : t("oidcSetup.saveAndLogin")}
                </Button>
              </>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

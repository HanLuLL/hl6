import { Link, useSearchParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/hooks/use-auth";
import { useBranding } from "@/hooks/use-branding";
import { ThemeToggle } from "@/components/layout/theme-toggle";
import { LanguageToggle } from "@/components/layout/language-toggle";
import { SiteFooter } from "@/components/layout/site-footer";

export default function LandingPage() {
  const { isAuthenticated, signIn } = useAuth();
  const branding = useBranding();
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  const ref = searchParams.get("ref") ?? undefined;

  return (
    <div className="min-h-screen">
      {/* Nav */}
      <header className="flex items-center justify-between px-6 py-4 border-b">
        <div className="flex items-center gap-2 font-semibold text-lg">
          {branding.logo_url && (
            <img src={branding.logo_url} alt={branding.name} className="h-6 w-6 rounded-sm object-contain" />
          )}
          {branding.name}
        </div>
        <div className="flex items-center gap-2">
          <LanguageToggle />
          <ThemeToggle />
          {isAuthenticated ? (
            <Button asChild className="bg-[#2D5AF6] hover:bg-[#2348D4] text-white">
              <Link to="/dashboard">{t("landing.dashboard")}</Link>
            </Button>
          ) : (
            <Button onClick={() => signIn(ref)} className="bg-[#2D5AF6] hover:bg-[#2348D4] text-white">{t("common.signIn")}</Button>
          )}
        </div>
      </header>

      {/* Hero */}
      <section className="flex flex-col items-center justify-center px-6 py-24 text-center">
        <h1 className="text-4xl font-bold tracking-tight sm:text-6xl max-w-3xl">
          {t("landing.heroTitle")}
          <span className="bg-gradient-to-r from-[#2D5AF6] to-[#6B7CFA] bg-clip-text text-transparent">
            {t("landing.heroHighlight")}
          </span>
        </h1>
        <p className="mt-6 text-lg text-muted-foreground max-w-2xl">
          {t("landing.heroDesc")}
        </p>
        <div className="mt-10 flex gap-4">
          {isAuthenticated ? (
            <Button size="lg" asChild className="bg-[#2D5AF6] hover:bg-[#2348D4] text-white">
              <Link to="/domains">{t("landing.browseDomains")}</Link>
            </Button>
          ) : (
            <Button size="lg" onClick={() => signIn(ref)} className="bg-[#2D5AF6] hover:bg-[#2348D4] text-white">
              {t("landing.getStarted")}
            </Button>
          )}
        </div>
      </section>

      {/* Features */}
      <section className="grid gap-8 px-6 py-16 md:grid-cols-3 max-w-5xl mx-auto">
        {[
          {
            title: t("landing.featureInstantTitle"),
            desc: t("landing.featureInstantDesc"),
            icon: "M13 10V3L4 14h7v7l9-11h-7z",
          },
          {
            title: t("landing.featureCloudflareTitle"),
            desc: t("landing.featureCloudflareDesc"),
            icon: "M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z",
          },
          {
            title: t("landing.featureControlTitle"),
            desc: t("landing.featureControlDesc"),
            icon: "M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z",
          },
        ].map((f) => (
          <div key={f.title} className="rounded-lg border p-6 transition-shadow hover:shadow-md">
            <div className="mb-4 inline-flex rounded-lg bg-[#2D5AF6]/10 p-3">
              <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-[#2D5AF6]"><path d={f.icon}/></svg>
            </div>
            <h3 className="text-lg font-semibold">{f.title}</h3>
            <p className="mt-2 text-sm text-muted-foreground">{f.desc}</p>
          </div>
        ))}
      </section>

      <footer className="border-t py-6 text-center text-sm text-muted-foreground">
        <p>{t("landing.copyright", { year: new Date().getFullYear() })}</p>
        <div className="mt-3">
          <SiteFooter withBorder={false} centered />
        </div>
      </footer>
    </div>
  );
}

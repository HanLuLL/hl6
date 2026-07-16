import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { Globe2, ShieldCheck } from "lucide-react";
import { useBranding } from "@/hooks/use-branding";
import { ThemeToggle } from "@/components/layout/theme-toggle";
import { LanguageToggle } from "@/components/layout/language-toggle";

type AuthShellProps = {
  title: string;
  description: string;
  children: ReactNode;
  footer?: ReactNode;
};

export function AuthShell({ title, description, children, footer }: AuthShellProps) {
  const branding = useBranding();

  return (
    <div className="min-h-screen bg-muted/30">
      <header className="flex h-14 items-center justify-between border-b bg-background px-4 sm:px-6">
        <Link to="/" className="flex min-w-0 items-center gap-2 font-semibold">
          {branding.logo_url ? (
            <img src={branding.logo_url} alt={branding.name} className="h-6 w-6 shrink-0 rounded object-contain" />
          ) : (
            <Globe2 className="h-5 w-5 text-primary" />
          )}
          <span className="truncate">{branding.name}</span>
        </Link>
        <div className="flex items-center gap-1">
          <LanguageToggle />
          <ThemeToggle />
        </div>
      </header>

      <main className="mx-auto grid min-h-[calc(100vh-3.5rem)] max-w-6xl items-stretch lg:grid-cols-[1.05fr_0.95fr]">
        <section className="hidden border-r bg-background px-10 py-14 lg:flex lg:flex-col lg:justify-between">
          <div className="max-w-md">
            <div className="mb-8 flex h-10 w-10 items-center justify-center rounded-md bg-primary text-primary-foreground">
              <ShieldCheck className="h-5 w-5" />
            </div>
            <h1 className="text-3xl font-semibold leading-tight text-foreground">{branding.name}</h1>
            <p className="mt-4 text-sm leading-6 text-muted-foreground">Manage your subdomains and DNS records from one secure account.</p>
          </div>
          <p className="text-xs text-muted-foreground">{branding.name}</p>
        </section>

        <section className="flex items-center justify-center px-4 py-10 sm:px-8 lg:px-12">
          <div className="w-full max-w-md rounded-lg border bg-card p-6 shadow-sm sm:p-8">
            <div className="mb-7">
              <h2 className="text-2xl font-semibold text-foreground">{title}</h2>
              <p className="mt-2 text-sm leading-6 text-muted-foreground">{description}</p>
            </div>
            {children}
            {footer ? <div className="mt-6 border-t pt-5 text-center text-sm text-muted-foreground">{footer}</div> : null}
          </div>
        </section>
      </main>
    </div>
  );
}

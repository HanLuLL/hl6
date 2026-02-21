import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/hooks/use-auth";
import { ThemeToggle } from "@/components/layout/theme-toggle";

export default function LandingPage() {
  const { isAuthenticated, signIn } = useAuth();

  return (
    <div className="min-h-screen">
      {/* Nav */}
      <header className="flex items-center justify-between px-6 py-4 border-b">
        <div className="flex items-center gap-2 font-semibold text-lg">
          <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"/></svg>
          SubDomain
        </div>
        <div className="flex items-center gap-2">
          <ThemeToggle />
          {isAuthenticated ? (
            <Button asChild>
              <Link to="/dashboard">Dashboard</Link>
            </Button>
          ) : (
            <Button onClick={() => signIn()}>Sign in</Button>
          )}
        </div>
      </header>

      {/* Hero */}
      <section className="flex flex-col items-center justify-center px-6 py-24 text-center">
        <h1 className="text-4xl font-bold tracking-tight sm:text-6xl max-w-3xl">
          Your subdomain,{" "}
          <span className="bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
            instantly deployed
          </span>
        </h1>
        <p className="mt-6 text-lg text-muted-foreground max-w-2xl">
          Claim a subdomain, configure DNS records, and get your project online in seconds.
          Simple, fast, and powered by Cloudflare.
        </p>
        <div className="mt-10 flex gap-4">
          {isAuthenticated ? (
            <Button size="lg" asChild>
              <Link to="/domains">Browse Domains</Link>
            </Button>
          ) : (
            <Button size="lg" onClick={() => signIn()}>
              Get Started
            </Button>
          )}
        </div>
      </section>

      {/* Features */}
      <section className="grid gap-8 px-6 py-16 md:grid-cols-3 max-w-5xl mx-auto">
        {[
          {
            title: "Instant Setup",
            desc: "Claim a subdomain and add DNS records in under a minute. No waiting, no approval process.",
            icon: "M13 10V3L4 14h7v7l9-11h-7z",
          },
          {
            title: "Cloudflare Powered",
            desc: "DNS records sync directly with Cloudflare for enterprise-grade reliability and speed.",
            icon: "M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z",
          },
          {
            title: "Full Control",
            desc: "Manage A, AAAA, and CNAME records. Toggle Cloudflare proxy. Set custom TTL values.",
            icon: "M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z",
          },
        ].map((f) => (
          <div key={f.title} className="rounded-lg border p-6 transition-shadow hover:shadow-md">
            <div className="mb-4 inline-flex rounded-lg bg-primary/10 p-3">
              <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-primary"><path d={f.icon}/></svg>
            </div>
            <h3 className="text-lg font-semibold">{f.title}</h3>
            <p className="mt-2 text-sm text-muted-foreground">{f.desc}</p>
          </div>
        ))}
      </section>

      <footer className="border-t py-6 text-center text-sm text-muted-foreground">
        SubDomain &copy; {new Date().getFullYear()}
      </footer>
    </div>
  );
}

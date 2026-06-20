import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { useAuth } from "@/hooks/use-auth";
import { useSubdomains } from "@/hooks/use-subdomains";
import { useCredits } from "@/hooks/use-credits";
import { useDocumentTitle } from "@/hooks/use-document-title";
import { Coins, Layers, Database } from "lucide-react";

export default function DashboardPage() {
  const { user, credits } = useAuth();
  const { data: subdomains, isLoading: subdomainsLoading } = useSubdomains();
  const { data: creditData, isLoading: creditsLoading } = useCredits();
  const { t } = useTranslation();
  useDocumentTitle(t("dashboard.title"));

  const isLoading = subdomainsLoading || creditsLoading;

  const dnsCount = subdomains?.reduce((acc, s) => acc + (s.dns_records?.length ?? 0), 0) ?? 0;

  const statDefs = [
    {
      title: t("dashboard.creditsBalance"),
      value: credits,
      icon: Coins,
      href: "/credits",
    },
    {
      title: t("dashboard.mySubdomains"),
      value: subdomains?.length ?? 0,
      icon: Layers,
      href: "/subdomains",
    },
    {
      title: t("dashboard.dnsRecords"),
      value: dnsCount,
      icon: Database,
      href: "/subdomains",
    },
  ];

  const txTypeColor: Record<string, string> = {
    grant: "bg-green-500",
    deduct: "bg-red-500",
    refund: "bg-brand",
  };

  return (
    <div className="space-y-8">
      {/* Welcome */}
      <div className="flex items-center gap-4">
        <Avatar className="h-11 w-11 shrink-0">
          <AvatarImage src={user?.avatar_url} />
          <AvatarFallback className="text-sm font-semibold">
            {user?.name?.charAt(0)?.toUpperCase() || "U"}
          </AvatarFallback>
        </Avatar>
        <div>
          <h1 className="text-2xl font-bold tracking-tight leading-none">
            {t("dashboard.title")}
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {t("dashboard.welcome", { name: user?.name || "User" })}
          </p>
        </div>
      </div>

      {/* Stat cards */}
      <div className="grid gap-4 md:grid-cols-3">
        {statDefs.map((stat) => {
          const Icon = stat.icon;
          return (
            <Card key={stat.title} className="gap-0 py-4 transition-shadow hover:shadow-sm">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 px-5 pb-0 pt-0">
                <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{stat.title}</p>
                <Icon className="h-4 w-4 text-muted-foreground/60 shrink-0" />
              </CardHeader>
              <CardContent className="px-5 pb-0 pt-2">
                {isLoading ? (
                  <Skeleton className="h-9 w-16" />
                ) : (
                  <div className="text-3xl font-bold tabular-nums leading-none">{stat.value}</div>
                )}
                <div className="mt-2">
                  <Link to={stat.href} className="text-xs text-muted-foreground hover:text-foreground transition-colors">
                    {t("dashboard.viewDetails")} →
                  </Link>
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>

      {/* Recent transactions */}
      {isLoading ? (
        <Card>
          <CardHeader>
            <Skeleton className="h-5 w-36" />
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {[...Array(5)].map((_, i) => (
                <div key={i} className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <Skeleton className="h-2 w-2 rounded-full" />
                    <div className="space-y-1">
                      <Skeleton className="h-4 w-40" />
                      <Skeleton className="h-3 w-24" />
                    </div>
                  </div>
                  <Skeleton className="h-4 w-10" />
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      ) : creditData?.transactions && creditData.transactions.length > 0 && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-3">
            <CardTitle className="text-base font-semibold">{t("dashboard.recentTransactions")}</CardTitle>
            <Link to="/credits" className="text-xs text-muted-foreground hover:text-foreground transition-colors">
              {t("dashboard.viewAll")} →
            </Link>
          </CardHeader>
          <CardContent className="px-5">
            <div className="divide-y">
              {creditData.transactions.slice(0, 5).map((tx) => (
                <div key={tx.id} className="flex items-center justify-between py-3 text-sm">
                  <div className="flex items-center gap-3">
                    <span className={`h-2 w-2 rounded-full shrink-0 ${txTypeColor[tx.type] ?? "bg-muted-foreground"}`} />
                    <div>
                      <p className="font-medium leading-none">{t(tx.description_key, tx.description_params ?? {})}</p>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        {new Date(tx.created_at).toLocaleDateString()}
                      </p>
                    </div>
                  </div>
                  <span className={`font-medium tabular-nums ${tx.amount > 0 ? "text-green-600 dark:text-green-400" : "text-red-600 dark:text-red-400"}`}>
                    {tx.amount > 0 ? "+" : ""}{tx.amount}
                  </span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

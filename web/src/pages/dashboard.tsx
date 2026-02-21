import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useAuth } from "@/hooks/use-auth";
import { useSubdomains } from "@/hooks/use-subdomains";
import { useCredits } from "@/hooks/use-credits";

export default function DashboardPage() {
  const { user, credits } = useAuth();
  const { data: subdomains, isLoading: subdomainsLoading } = useSubdomains();
  const { data: creditData, isLoading: creditsLoading } = useCredits();
  const { t } = useTranslation();

  const isLoading = subdomainsLoading || creditsLoading;

  const statDefs = [
    {
      title: t("dashboard.creditsBalance"),
      icon: "M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z",
    },
    {
      title: t("dashboard.mySubdomains"),
      icon: "M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9",
    },
    {
      title: t("dashboard.dnsRecords"),
      icon: "M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10",
    },
  ];

  const statValues = [
    credits,
    subdomains?.length ?? 0,
    subdomains?.reduce((acc, s) => acc + (s.dns_records?.length ?? 0), 0) ?? 0,
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("dashboard.title")}</h1>
        <p className="text-muted-foreground">{t("dashboard.welcome", { name: user?.name || "User" })}</p>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        {statDefs.map((stat, i) => (
          <Card key={stat.title}>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">{stat.title}</CardTitle>
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-muted-foreground"><path d={stat.icon}/></svg>
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <Skeleton className="h-9 w-16" />
              ) : (
                <div className="text-3xl font-bold">{statValues[i]}</div>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      {isLoading ? (
        <Card>
          <CardHeader>
            <Skeleton className="h-5 w-36" />
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {[...Array(5)].map((_, i) => (
                <div key={i} className="flex items-center justify-between">
                  <div className="space-y-1">
                    <Skeleton className="h-4 w-40" />
                    <Skeleton className="h-3 w-24" />
                  </div>
                  <Skeleton className="h-4 w-10" />
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      ) : creditData?.transactions && creditData.transactions.length > 0 && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-lg">{t("dashboard.recentTransactions")}</CardTitle>
            <Link to="/credits" className="text-sm text-muted-foreground hover:text-foreground transition-colors">
              {t("dashboard.viewAll")} &rarr;
            </Link>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {creditData.transactions.slice(0, 5).map((tx) => (
                <div key={tx.id} className="flex items-center justify-between text-sm">
                  <div>
                    <p className="font-medium">{t(tx.description_key, tx.description_params ?? {})}</p>
                    <p className="text-xs text-muted-foreground">
                      {new Date(tx.created_at).toLocaleDateString()}
                    </p>
                  </div>
                  <span className={tx.amount > 0 ? "text-green-600 font-medium" : "text-red-600 font-medium"}>
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

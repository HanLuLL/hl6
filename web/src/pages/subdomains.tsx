import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useSubdomains, useReleaseSubdomain } from "@/hooks/use-subdomains";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/api";

export default function SubdomainsPage() {
  const { data: subdomains, isLoading } = useSubdomains();
  const release = useReleaseSubdomain();
  const { t } = useTranslation();

  const handleRelease = (id: number, fqdn: string) => {
    if (!confirm(t("subdomains.releaseConfirm", { fqdn }))) return;
    release.mutate(id, {
      onSuccess: () => toast.success(t("subdomains.released", { fqdn })),
      onError: (err) => toast.error(getErrorMessage(err, t)),
    });
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("subdomains.title")}</h1>
          <p className="text-muted-foreground">{t("subdomains.subtitle")}</p>
        </div>
        <Button asChild>
          <Link to="/domains">{t("subdomains.claimNew")}</Link>
        </Button>
      </div>

      {!subdomains || subdomains.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1" className="text-muted-foreground/50 mb-4"><path d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/></svg>
          <p className="text-muted-foreground">{t("subdomains.noSubdomains")}</p>
          <Button className="mt-4" asChild>
            <Link to="/domains">{t("subdomains.browseDomains")}</Link>
          </Button>
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {subdomains.map((sub) => (
            <Card key={sub.id} className="group">
              <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-lg">
                    <Link to={`/subdomains/${sub.id}`} className="hover:underline">
                      {sub.fqdn}
                    </Link>
                  </CardTitle>
                  <Badge variant="outline">{t("subdomains.records", { count: sub.dns_records?.length ?? 0 })}</Badge>
                </div>
              </CardHeader>
              <CardContent>
                <div className="flex items-center justify-between">
                  <span className="text-xs text-muted-foreground">
                    {t("subdomains.claimed", { date: new Date(sub.created_at).toLocaleDateString() })}
                  </span>
                  <div className="flex gap-2">
                    <Button size="sm" variant="outline" asChild>
                      <Link to={`/subdomains/${sub.id}`}>{t("subdomains.manage")}</Link>
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      className="text-destructive"
                      onClick={() => handleRelease(sub.id, sub.fqdn)}
                      disabled={release.isPending}
                    >
                      {t("subdomains.release")}
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

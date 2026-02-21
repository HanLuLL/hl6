import { useState } from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { useSubdomains, useReleaseSubdomain } from "@/hooks/use-subdomains";

export default function SubdomainsPage() {
  const { data: subdomains, isLoading } = useSubdomains();
  const release = useReleaseSubdomain();
  const { t } = useTranslation();
  const [releaseTarget, setReleaseTarget] = useState<{ id: number; fqdn: string } | null>(null);

  const handleRelease = () => {
    if (!releaseTarget) return;
    release.mutate(
      { id: releaseTarget.id, fqdn: releaseTarget.fqdn },
      { onSuccess: () => setReleaseTarget(null) }
    );
  };

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

      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[...Array(3)].map((_, i) => (
            <Card key={i}>
              <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                  <Skeleton className="h-5 w-36" />
                  <Skeleton className="h-5 w-16" />
                </div>
              </CardHeader>
              <CardContent>
                <div className="flex items-center justify-between">
                  <Skeleton className="h-4 w-28" />
                  <div className="flex gap-2">
                    <Skeleton className="h-8 w-16" />
                    <Skeleton className="h-8 w-14" />
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : !subdomains || subdomains.length === 0 ? (
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
                  <div className="text-lg font-semibold">
                    <Link to={`/subdomains/${sub.id}`} className="hover:underline">
                      {sub.fqdn}
                    </Link>
                  </div>
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
                      onClick={() => setReleaseTarget({ id: sub.id, fqdn: sub.fqdn })}
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

      <ConfirmDialog
        open={!!releaseTarget}
        onOpenChange={(open) => !open && setReleaseTarget(null)}
        title={t("subdomains.releaseTitle")}
        description={t("subdomains.releaseDesc", { fqdn: releaseTarget?.fqdn })}
        confirmInput={releaseTarget?.fqdn}
        confirmInputLabel={t("subdomains.releaseInputLabel", { fqdn: releaseTarget?.fqdn })}
        onConfirm={handleRelease}
        destructive
        loading={release.isPending}
      />
    </div>
  );
}

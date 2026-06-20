import { useState } from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { useSubdomains, useReleaseSubdomain } from "@/hooks/use-subdomains";
import { useDocumentTitle } from "@/hooks/use-document-title";
import { Layers, Plus } from "lucide-react";

export default function SubdomainsPage() {
  const { data: subdomains, isLoading } = useSubdomains();
  const release = useReleaseSubdomain();
  const { t } = useTranslation();
  useDocumentTitle(t("subdomains.title"));
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
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("subdomains.title")}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{t("subdomains.subtitle")}</p>
        </div>
        <Button asChild size="sm" className="shrink-0 mt-0.5">
          <Link to="/domains">
            <Plus className="mr-1.5 h-3.5 w-3.5" />
            {t("subdomains.claimNew")}
          </Link>
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
        <div className="flex flex-col items-center justify-center py-20 text-center">
          <div className="rounded-xl bg-muted p-4 mb-4">
            <Layers className="h-8 w-8 text-muted-foreground/50" />
          </div>
          <p className="text-muted-foreground text-sm mb-4">{t("subdomains.noSubdomains")}</p>
          <Button size="sm" asChild>
            <Link to="/domains">{t("subdomains.browseDomains")}</Link>
          </Button>
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {subdomains.map((sub) => (
            <Card key={sub.id} className="hover:-translate-y-px transition-transform duration-150">
              <CardHeader className="pb-3">
                <div className="flex items-center justify-between gap-2">
                  <div className="text-base font-semibold truncate">
                    <Link to={`/subdomains/${sub.id}`} className="hover:underline">
                      {sub.fqdn}
                    </Link>
                  </div>
                  <div className="flex items-center gap-1.5 shrink-0">
                    {sub.status === "suspended" && (
                      <Badge variant="destructive" className="text-xs">{t("subdomains.statusSuspended")}</Badge>
                    )}
                    <Badge variant="outline" className="text-xs">{t("subdomains.records", { count: sub.dns_records?.length ?? 0 })}</Badge>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <div className="flex items-center justify-between">
                  <span className="text-xs text-muted-foreground">
                    {t("subdomains.claimed", { date: new Date(sub.created_at).toLocaleDateString() })}
                  </span>
                  <div className="flex gap-1.5">
                    <Button size="sm" variant="outline" asChild>
                      <Link to={`/subdomains/${sub.id}`}>{t("subdomains.manage")}</Link>
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      className="text-destructive hover:text-destructive"
                      onClick={() => setReleaseTarget({ id: sub.id, fqdn: sub.fqdn })}
                      disabled={release.isPending || release.isRetrying}
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
        confirmText={release.isRetrying ? `${t("common.retry")}...` : t("common.confirm")}
        confirmInput={releaseTarget?.fqdn}
        confirmInputLabel={t("subdomains.releaseInputLabel", { fqdn: releaseTarget?.fqdn })}
        onConfirm={handleRelease}
        destructive
        loading={release.isPending || release.isRetrying}
      />
    </div>
  );
}

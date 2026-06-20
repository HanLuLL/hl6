import { Link } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api";
import { domainToUnicode } from "@/lib/idn";
import { formatDate } from "@/lib/format-date";
import { useErrorToast } from "@/hooks/use-error-toast";
import { toast } from "sonner";
import {
  Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle,
} from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { ExternalLink } from "lucide-react";
import { useState } from "react";
import { ReleaseSubdomainDialog } from "./release-subdomain-dialog";

export function AuditDetailSheet({
  subdomainId,
  onClose,
}: {
  subdomainId: number | null;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const showError = useErrorToast();
  const queryClient = useQueryClient();
  const [releaseOpen, setReleaseOpen] = useState(false);

  const { data: detail, isLoading } = useQuery({
    queryKey: ["admin-audit-detail", subdomainId],
    queryFn: async () => (await api.adminGetAuditSubdomainDetail(subdomainId!)).data,
    enabled: subdomainId != null,
  });

  const { data: scans } = useQuery({
    queryKey: ["admin-audit-detail-scans", subdomainId],
    queryFn: async () => (await api.adminListAuditSubdomainScans(subdomainId!, 1, 10)).data,
    enabled: subdomainId != null,
  });

  const restoreMutation = useMutation({
    mutationFn: () => api.adminRestoreAuditSubdomain(subdomainId!),
    onSuccess: () => {
      toast.success(t("audit.detail.restored"));
      queryClient.invalidateQueries({ queryKey: ["admin-audit"] });
      queryClient.invalidateQueries({ queryKey: ["admin-audit-detail", subdomainId] });
      queryClient.invalidateQueries({ queryKey: ["admin-audit-cases"] });
    },
    onError: (err) => showError(err),
  });

  const rescanMutation = useMutation({
    mutationFn: () => api.adminRescanAuditSubdomain(subdomainId!),
    onSuccess: () => toast.success(t("audit.detail.rescanQueued")),
    onError: (err) => showError(err),
  });

  const sub = detail?.subdomain;
  const violation = detail?.latest_violation;
  const canRescan = detail?.scannable === true && sub?.status === "active";

  return (
    <>
      <Sheet open={subdomainId != null} onOpenChange={(open) => !open && onClose()}>
        <SheetContent className="flex w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-lg">
          <div className="shrink-0 border-b px-6 py-5">
            <SheetHeader className="p-0 text-left">
              <SheetTitle>{sub ? domainToUnicode(sub.fqdn) : t("audit.detail.title")}</SheetTitle>
              <SheetDescription>{t("audit.detail.subtitle")}</SheetDescription>
            </SheetHeader>
          </div>

          <div className="flex-1 overflow-y-auto px-6 py-5">
          {isLoading || !detail ? (
            <div className="space-y-3">
              <Skeleton className="h-6 w-full" />
              <Skeleton className="h-24 w-full" />
            </div>
          ) : (
            <div className="space-y-6 text-sm">
              <section className="space-y-2">
                <div className="flex items-center gap-2">
                  <Badge variant={sub!.status === "suspended" ? "destructive" : "secondary"}>
                    {sub!.status === "suspended" ? t("audit.status.suspended") : t("audit.status.active")}
                  </Badge>
                  <Link to="/admin/users" className="text-xs text-primary hover:underline">
                    {detail.user_email}
                  </Link>
                </div>
                <p className="text-muted-foreground">{sub!.domain?.name}</p>
                {sub!.suspended_reason && (
                  <p className="text-destructive text-xs">
                    {t("audit.detail.suspendedReason", { reason: sub!.suspended_reason })}
                    {sub!.suspended_at && ` · ${formatDate(sub!.suspended_at)}`}
                  </p>
                )}
              </section>

              {violation && (
                <section className="space-y-2">
                  <h3 className="font-medium">{t("audit.detail.evidence")}</h3>
                  <p className="text-xs font-medium">{violation.matched_rule_name}</p>
                  <pre className="text-xs bg-muted p-3 rounded-md whitespace-pre-wrap break-all">{violation.matched_snippet}</pre>
                </section>
              )}

              <section className="space-y-2">
                <h3 className="font-medium">{t("audit.detail.scanTimeline")}</h3>
                <ul className="space-y-1 text-xs text-muted-foreground">
                  {(scans ?? []).map((s) => (
                    <li key={s.id} className="flex justify-between gap-2">
                      <span>{formatDate(s.created_at)}</span>
                      <span>{s.status} · HTTP {s.http_status_code || "—"}</span>
                    </li>
                  ))}
                </ul>
              </section>

              {detail.sibling_subdomains.length > 0 && (
                <section className="space-y-2">
                  <h3 className="font-medium">{t("audit.detail.siblings")}</h3>
                  <ul className="text-xs space-y-1">
                    {detail.sibling_subdomains.map((s) => (
                      <li key={s.id} className="flex justify-between">
                        <span className="font-mono">{domainToUnicode(s.fqdn)}</span>
                        <span className="text-muted-foreground">{s.status}</span>
                      </li>
                    ))}
                  </ul>
                </section>
              )}

              {detail.dns_records.length > 0 && (
                <section className="space-y-2">
                  <h3 className="font-medium">{t("audit.detail.dns")}</h3>
                  <ul className="text-xs font-mono space-y-1">
                    {detail.dns_records.map((r) => (
                      <li key={r.id}>{r.type} {r.content}</li>
                    ))}
                  </ul>
                </section>
              )}

              <div className="flex flex-wrap gap-2 pt-2">
                {sub!.status === "suspended" && (
                  <Button size="sm" onClick={() => restoreMutation.mutate()} disabled={restoreMutation.isPending}>
                    {t("audit.actions.restore")}
                  </Button>
                )}
                {sub!.status === "active" && (
                  <Button size="sm" variant="destructive" onClick={() => setReleaseOpen(true)}>
                    {t("audit.actions.release")}
                  </Button>
                )}
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => rescanMutation.mutate()}
                  disabled={rescanMutation.isPending || !canRescan}
                  title={!canRescan ? t("audit.detail.notScannableHint") : undefined}
                >
                  {t("audit.actions.rescan")}
                </Button>
                <Button size="sm" variant="outline" asChild>
                  <a href={`https://${sub!.fqdn}`} target="_blank" rel="noreferrer">
                    <ExternalLink className="h-4 w-4 mr-1" />
                    {t("audit.actions.openSite")}
                  </a>
                </Button>
              </div>
            </div>
          )}
          </div>
        </SheetContent>
      </Sheet>

      <ReleaseSubdomainDialog
        subdomainId={subdomainId}
        open={releaseOpen}
        onOpenChange={setReleaseOpen}
        onSuccess={onClose}
      />
    </>
  );
}

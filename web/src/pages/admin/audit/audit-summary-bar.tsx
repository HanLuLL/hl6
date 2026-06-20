import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

export function AuditSummaryBar() {
  const { t } = useTranslation();
  const { data, isLoading } = useQuery({
    queryKey: ["admin-audit-summary"],
    queryFn: async () => (await api.adminGetAuditSummary()).data,
    staleTime: 30_000,
  });

  if (isLoading) {
    return (
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
        {[...Array(5)].map((_, i) => (
          <Skeleton key={i} className="h-14 rounded-xl" />
        ))}
      </div>
    );
  }

  const items = [
    { label: t("audit.summary.suspended"), value: data?.suspended_count ?? 0 },
    { label: t("audit.summary.violation24h"), value: data?.violation_24h ?? 0 },
    { label: t("audit.summary.unreachable24h"), value: data?.unreachable_24h ?? 0 },
    { label: t("audit.summary.neverScanned"), value: data?.never_scanned_count ?? 0 },
    { label: t("audit.summary.enabledRules"), value: data?.enabled_rules_count ?? 0 },
  ];

  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
      {items.map((item) => (
        <Card key={item.label} className="gap-0 py-0 shadow-none">
          <CardContent className="px-3 py-2.5">
            <p className="text-xs text-muted-foreground">{item.label}</p>
            <p className="text-2xl font-semibold leading-tight tabular-nums">{item.value}</p>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

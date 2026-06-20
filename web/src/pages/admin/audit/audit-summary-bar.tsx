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
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {[...Array(4)].map((_, i) => (
          <Skeleton key={i} className="h-14 rounded-xl" />
        ))}
      </div>
    );
  }

  const items = [
    { label: t("audit.summary.deleted"), value: data?.deleted_count ?? 0 },
    { label: t("audit.summary.currentViolation"), value: data?.current_violation ?? 0 },
    { label: t("audit.summary.neverScanned"), value: data?.never_scanned_count ?? 0 },
    { label: t("audit.summary.enabledRules"), value: data?.enabled_rules_count ?? 0 },
  ];

  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
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

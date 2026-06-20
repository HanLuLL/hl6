import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api";
import { domainToUnicode } from "@/lib/idn";
import { formatDate } from "@/lib/format-date";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Paginator } from "@/components/ui/paginator";
import { CopyableEmail } from "@/components/ui/copyable-email";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import type { AuditWorkbenchItem } from "@/types";
import { AuditDetailSheet } from "./audit-detail-sheet";

export function ViolationsTab() {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
  const [fqdn, setFqdn] = useState("");
  const [detailId, setDetailId] = useState<number | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["admin-audit-cases", page, fqdn],
    queryFn: async () => {
      const res = await api.adminListAuditCases(page, 20, {
        fqdn: fqdn || undefined,
        status: "active,suspended",
      });
      return { items: res.data, total: res.total };
    },
    staleTime: 15_000,
  });

  const items = data?.items ?? [];
  const totalPages = Math.ceil((data?.total ?? 0) / 20);

  return (
    <div className="space-y-4">
      <Input
        className="max-w-xs"
        placeholder={t("audit.filters.fqdn")}
        value={fqdn}
        onChange={(e) => { setFqdn(e.target.value); setPage(1); }}
      />

      <Card>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="p-6"><Skeleton className="h-40 w-full" /></div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("audit.columns.fqdn")}</TableHead>
                  <TableHead>{t("audit.columns.user")}</TableHead>
                  <TableHead>{t("audit.columns.status")}</TableHead>
                  <TableHead>{t("audit.columns.latestViolation")}</TableHead>
                  <TableHead>{t("audit.columns.violations7d")}</TableHead>
                  <TableHead className="text-right">{t("audit.columns.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                      {t("audit.empty.cases")}
                    </TableCell>
                  </TableRow>
                )}
                {items.map((item: AuditWorkbenchItem) => (
                  <TableRow key={item.subdomain_id}>
                    <TableCell className="font-mono text-xs">{domainToUnicode(item.fqdn)}</TableCell>
                    <TableCell className="text-xs">
                      <CopyableEmail email={item.user_email} className="max-w-[200px]" />
                    </TableCell>
                    <TableCell>
                      <Badge variant={item.status === "suspended" ? "destructive" : "secondary"}>
                        {item.status === "suspended" ? t("audit.status.suspended") : t("audit.status.active")}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground max-w-[240px] truncate">
                      {item.latest_violation?.matched_rule_name ?? "—"}
                      {item.latest_violation?.created_at && (
                        <span className="block text-[10px]">{formatDate(item.latest_violation.created_at)}</span>
                      )}
                    </TableCell>
                    <TableCell>{item.violation_count_7d}</TableCell>
                    <TableCell className="text-right">
                      <Button variant="ghost" size="sm" onClick={() => setDetailId(item.subdomain_id)}>
                        {t("audit.actions.detail")}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <div className="flex justify-center">
        <Paginator page={page} totalPages={totalPages} onPageChange={setPage} />
      </div>

      <AuditDetailSheet subdomainId={detailId} onClose={() => setDetailId(null)} />
    </div>
  );
}

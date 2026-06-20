import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/lib/api";
import { domainToUnicode } from "@/lib/idn";
import { formatDate } from "@/lib/format-date";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Paginator } from "@/components/ui/paginator";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import type { SubdomainScan } from "@/types";
import { AuditDetailSheet } from "./audit-detail-sheet";

export function HistoryTab() {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
  const [detailId, setDetailId] = useState<number | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["admin-audit-scans", page],
    queryFn: async () => {
      const res = await api.adminListAuditScans(page, 20, { status: "violation" });
      return { items: res.data, total: res.total };
    },
    staleTime: 15_000,
  });

  const items = data?.items ?? [];
  const totalPages = Math.ceil((data?.total ?? 0) / 20);

  return (
    <div className="space-y-4">
      <Card>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="p-6"><Skeleton className="h-40 w-full" /></div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("audit.columns.fqdn")}</TableHead>
                  <TableHead>{t("audit.columns.scanTime")}</TableHead>
                  <TableHead>{t("audit.columns.httpStatus")}</TableHead>
                  <TableHead>{t("audit.columns.evidence")}</TableHead>
                  <TableHead className="text-right">{t("audit.columns.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                      {t("audit.empty.history")}
                    </TableCell>
                  </TableRow>
                )}
                {items.map((scan: SubdomainScan) => (
                  <TableRow key={scan.id}>
                    <TableCell className="font-mono text-xs">{domainToUnicode(scan.fqdn)}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{formatDate(scan.created_at)}</TableCell>
                    <TableCell>{scan.http_status_code || "—"}</TableCell>
                    <TableCell className="text-xs max-w-[280px] truncate text-muted-foreground">
                      {scan.matched_snippet || "—"}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button variant="ghost" size="sm" onClick={() => setDetailId(scan.subdomain_id)}>
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

import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { Skeleton } from "@/components/ui/skeleton";

export default function AdminAuditLogsPage() {
  const [page, setPage] = useState(1);
  const { t } = useTranslation();
  const { data, isLoading } = useQuery({
    queryKey: ["admin-audit-logs", page],
    queryFn: async () => {
      const res = await api.adminListAuditLogs(page, 20);
      return { logs: res.data, total: res.total };
    },
  });

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("auditLogs.title")}</h1>
          <p className="text-muted-foreground">{t("auditLogs.subtitle")}</p>
        </div>
        <Card>
          <CardHeader>
            <Skeleton className="h-4 w-28" />
          </CardHeader>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("auditLogs.action")}</TableHead>
                  <TableHead>{t("auditLogs.user")}</TableHead>
                  <TableHead>{t("auditLogs.resource")}</TableHead>
                  <TableHead>{t("auditLogs.details")}</TableHead>
                  <TableHead>{t("auditLogs.time")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {[...Array(6)].map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><Skeleton className="h-5 w-20 rounded-full" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-28" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-40" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-28" /></TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("auditLogs.title")}</h1>
        <p className="text-muted-foreground">{t("auditLogs.subtitle")}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium text-muted-foreground">
            {t("auditLogs.totalEntries", { count: data?.total ?? 0 })}
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("auditLogs.action")}</TableHead>
                <TableHead>{t("auditLogs.user")}</TableHead>
                <TableHead>{t("auditLogs.resource")}</TableHead>
                <TableHead>{t("auditLogs.details")}</TableHead>
                <TableHead>{t("auditLogs.time")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data?.logs?.map((log) => (
                <TableRow key={log.id}>
                  <TableCell>
                    <Badge variant="outline">{log.action}</Badge>
                  </TableCell>
                  <TableCell className="text-sm">{log.user?.name ?? `User #${log.user_id}`}</TableCell>
                  <TableCell className="text-sm text-muted-foreground">{log.resource} #{log.resource_id}</TableCell>
                  <TableCell className="text-xs font-mono text-muted-foreground max-w-xs truncate">
                    {JSON.stringify(log.details)}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {new Date(log.created_at).toLocaleString()}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {data && data.total > 20 && (
        <div className="flex justify-center gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>{t("common.previous")}</Button>
          <span className="flex items-center text-sm text-muted-foreground">{t("common.page", { page })}</span>
          <Button variant="outline" size="sm" disabled={page >= Math.ceil(data.total / 20)} onClick={() => setPage((p) => p + 1)}>{t("common.next")}</Button>
        </div>
      )}
    </div>
  );
}

import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
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
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const { t } = useTranslation();

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  const { data, isLoading } = useQuery({
    queryKey: ["admin-audit-logs", page, debouncedSearch],
    queryFn: async () => {
      const res = await api.adminListAuditLogs(page, 15, debouncedSearch);
      return { logs: res.data, total: res.total };
    },
    staleTime: 30_000,
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("auditLogs.title")}</h1>
        <p className="text-muted-foreground">{t("auditLogs.subtitle")}</p>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          {isLoading ? (
            <Skeleton className="h-4 w-28" />
          ) : (
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {t("auditLogs.totalEntries", { count: data?.total ?? 0 })}
            </CardTitle>
          )}
          <Input
            placeholder={t("auditLogs.searchPlaceholder")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="max-w-xs"
          />
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
              {isLoading ? (
                [...Array(6)].map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><Skeleton className="h-5 w-20 rounded-full" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-28" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-40" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-28" /></TableCell>
                  </TableRow>
                ))
              ) : (
                data?.logs?.map((log) => (
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
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {data && data.total > 15 && (
        <div className="flex justify-center gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>{t("common.previous")}</Button>
          <span className="flex items-center text-sm text-muted-foreground">{t("common.pageOf", { page, total: Math.ceil(data.total / 15) })}</span>
          <Button variant="outline" size="sm" disabled={page >= Math.ceil(data.total / 15)} onClick={() => setPage((p) => p + 1)}>{t("common.next")}</Button>
        </div>
      )}
    </div>
  );
}

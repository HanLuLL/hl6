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
import { toast } from "sonner";

const EMAIL_MAX_VISIBLE = 24;

function truncateText(value: string, maxLen = EMAIL_MAX_VISIBLE): string {
  if (value.length <= maxLen) {
    return value;
  }
  return `${value.slice(0, maxLen)}...`;
}

export default function AdminAuditLogsPage() {
  const [page, setPage] = useState(1);
  const [operator, setOperator] = useState("");
  const [debouncedOperator, setDebouncedOperator] = useState("");
  const [action, setAction] = useState("");
  const [debouncedAction, setDebouncedAction] = useState("");
  const { t } = useTranslation();

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedOperator(operator.trim());
      setDebouncedAction(action.trim());
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [operator, action]);

  const { data, isLoading } = useQuery({
    queryKey: ["admin-audit-logs", page, debouncedOperator, debouncedAction],
    queryFn: async () => {
      const res = await api.adminListAuditLogs(page, 15, debouncedOperator, debouncedAction);
      return { logs: res.data, total: res.total };
    },
    staleTime: 30_000,
  });

  const copyEmail = (email: string) => {
    navigator.clipboard.writeText(email).then(() => {
      toast.success(t("common.copied"));
    }).catch(() => {
      // Ignore clipboard errors in unsupported or insecure contexts.
    });
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("auditLogs.title")}</h1>
        <p className="text-muted-foreground">{t("auditLogs.subtitle")}</p>
      </div>

      <Card>
        <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2">
          {isLoading ? (
            <Skeleton className="h-4 w-28" />
          ) : (
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {t("auditLogs.totalEntries", { count: data?.total ?? 0 })}
            </CardTitle>
          )}
          <div className="flex w-full max-w-2xl flex-wrap items-center justify-end gap-2">
            <Input
              placeholder={t("auditLogs.operatorPlaceholder")}
              value={operator}
              onChange={(e) => setOperator(e.target.value)}
              className="w-56"
            />
            <Input
              placeholder={t("auditLogs.actionPlaceholder")}
              value={action}
              onChange={(e) => setAction(e.target.value)}
              className="w-56"
            />
          </div>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("auditLogs.action")}</TableHead>
                <TableHead>{t("auditLogs.userEmail")}</TableHead>
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
                    <TableCell className="text-sm">
                      {log.user?.email ? (
                        <button
                          type="button"
                          title={log.user.email}
                          className="max-w-56 cursor-pointer truncate text-left text-foreground hover:text-primary"
                          onClick={() => copyEmail(log.user?.email ?? "")}
                        >
                          {truncateText(log.user.email)}
                        </button>
                      ) : (
                        `User #${log.user_id}`
                      )}
                    </TableCell>
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

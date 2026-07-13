import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api, getErrorMessage } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { useDocumentTitle } from "@/hooks/use-document-title";

export default function AdminEmailLogsPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  useDocumentTitle(t("emailLogs.title"));

  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState("");
  const [typeFilter, setTypeFilter] = useState("");
  const perPage = 20;

  const { data, isLoading } = useQuery({
    queryKey: ["admin-email-logs", page, statusFilter, typeFilter],
    queryFn: async () => {
      const res = await api.adminListEmailLogs(page, perPage, typeFilter, statusFilter);
      return res;
    },
  });

  const retryMutation = useMutation({
    mutationFn: (id: number) => api.adminRetryEmail(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-email-logs"] });
      toast.success(t("emailLogs.retrySuccess"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  const logs = (data?.data ?? []) as import("@/types").EmailLog[];
  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / perPage);

  const statusBadge = (status: string) => {
    switch (status) {
      case "sent":
        return <Badge className="bg-green-600 hover:bg-green-700">{t("emailLogs.statusSent")}</Badge>;
      case "failed":
        return <Badge variant="destructive">{t("emailLogs.statusFailed")}</Badge>;
      case "pending":
        return <Badge className="bg-yellow-600 hover:bg-yellow-700">{t("emailLogs.statusPending")}</Badge>;
      default:
        return <Badge variant="secondary">{status}</Badge>;
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("emailLogs.title")}</h1>
        <p className="text-muted-foreground">{t("emailLogs.subtitle")}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("emailLogs.listTitle")}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-4 mb-4">
            <Select value={statusFilter} onValueChange={(v) => { setStatusFilter(v === "all" ? "" : v); setPage(1); }}>
              <SelectTrigger className="w-40">
                <SelectValue placeholder={t("emailLogs.filterStatus")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{t("emailLogs.allStatus")}</SelectItem>
                <SelectItem value="pending">{t("emailLogs.statusPending")}</SelectItem>
                <SelectItem value="sent">{t("emailLogs.statusSent")}</SelectItem>
                <SelectItem value="failed">{t("emailLogs.statusFailed")}</SelectItem>
              </SelectContent>
            </Select>
            <Select value={typeFilter} onValueChange={(v) => { setTypeFilter(v === "all" ? "" : v); setPage(1); }}>
              <SelectTrigger className="w-40">
                <SelectValue placeholder={t("emailLogs.filterType")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{t("emailLogs.allTypes")}</SelectItem>
                <SelectItem value="ban_notify">{t("emailLogs.typeBanNotify")}</SelectItem>
                <SelectItem value="test">{t("emailLogs.typeTest")}</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {isLoading ? (
            <p className="text-muted-foreground">{t("common.loading")}</p>
          ) : logs.length === 0 ? (
            <p className="text-muted-foreground">{t("emailLogs.noLogs")}</p>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("emailLogs.colRecipient")}</TableHead>
                    <TableHead>{t("emailLogs.colSubject")}</TableHead>
                    <TableHead>{t("emailLogs.colType")}</TableHead>
                    <TableHead>{t("emailLogs.colStatus")}</TableHead>
                    <TableHead>{t("emailLogs.colRetries")}</TableHead>
                    <TableHead>{t("emailLogs.colCreatedAt")}</TableHead>
                    <TableHead className="text-right">{t("emailLogs.colActions")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {logs.map((log) => (
                    <TableRow key={log.id}>
                      <TableCell className="font-mono text-sm">{log.recipient}</TableCell>
                      <TableCell>{log.subject}</TableCell>
                      <TableCell>{log.email_type}</TableCell>
                      <TableCell>{statusBadge(log.status)}</TableCell>
                      <TableCell>{log.retry_count}</TableCell>
                      <TableCell className="text-sm text-muted-foreground">{new Date(log.created_at).toLocaleString()}</TableCell>
                      <TableCell className="text-right">
                        {log.status === "failed" && (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => retryMutation.mutate(log.id)}
                            disabled={retryMutation.isPending}
                          >
                            {t("emailLogs.retry")}
                          </Button>
                        )}
                        {log.status === "failed" && log.error && (
                          <span className="ml-2 text-xs text-destructive" title={log.error}>
                            {t("emailLogs.hasError")}
                          </span>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
              <div className="flex items-center justify-between mt-4">
                <p className="text-sm text-muted-foreground">
                  {t("common.pageOf", { page, total: totalPages })}
                </p>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    disabled={page <= 1}
                  >
                    {t("common.previous")}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                    disabled={page >= totalPages}
                  >
                    {t("common.next")}
                  </Button>
                </div>
              </div>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

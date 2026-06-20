import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { api } from "@/lib/api";
import { domainToUnicode } from "@/lib/idn";
import { formatDate } from "@/lib/format-date";
import { useErrorToast } from "@/hooks/use-error-toast";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { Paginator } from "@/components/ui/paginator";
import { CopyableEmail } from "@/components/ui/copyable-email";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import type { AuditSiteItem } from "@/types";

export function SitesTab() {
  const { t } = useTranslation();
  const showError = useErrorToast();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<Set<number>>(new Set());

  const { data, isLoading } = useQuery({
    queryKey: ["admin-audit-sites", page, search],
    queryFn: async () => {
      const res = await api.adminListAuditSites(page, 20, search);
      return { items: res.data, total: res.total };
    },
    staleTime: 15_000,
  });

  const bulkMutation = useMutation({
    mutationFn: (ids: number[]) => api.adminBulkRescanAudit(ids),
    onSuccess: (res) => {
      toast.success(t("audit.actions.bulkRescanQueued", { count: res.data.queued }));
      setSelected(new Set());
      queryClient.invalidateQueries({ queryKey: ["admin-audit-sites"] });
    },
    onError: (err) => showError(err),
  });

  const items = data?.items ?? [];
  const totalPages = Math.ceil((data?.total ?? 0) / 20);

  const toggle = (id: number) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleAll = () => {
    if (selected.size === items.length) setSelected(new Set());
    else setSelected(new Set(items.map((i: AuditSiteItem) => i.subdomain_id)));
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <Input
          className="max-w-xs"
          placeholder={t("audit.filters.search")}
          value={search}
          onChange={(e) => { setSearch(e.target.value); setPage(1); }}
        />
        {selected.size > 0 && (
          <Button
            size="sm"
            disabled={bulkMutation.isPending}
            onClick={() => bulkMutation.mutate([...selected])}
          >
            {t("audit.actions.bulkRescan", { count: selected.size })}
          </Button>
        )}
      </div>

      <Card>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="p-6"><Skeleton className="h-40 w-full" /></div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-10">
                    <Checkbox
                      checked={items.length > 0 && selected.size === items.length}
                      onCheckedChange={toggleAll}
                    />
                  </TableHead>
                  <TableHead>{t("audit.columns.fqdn")}</TableHead>
                  <TableHead>{t("audit.columns.user")}</TableHead>
                  <TableHead>{t("audit.columns.scanStatus")}</TableHead>
                  <TableHead>{t("audit.columns.lastScan")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                      {t("audit.empty.sites")}
                    </TableCell>
                  </TableRow>
                )}
                {items.map((item: AuditSiteItem) => (
                  <TableRow key={item.subdomain_id}>
                    <TableCell>
                      <Checkbox checked={selected.has(item.subdomain_id)} onCheckedChange={() => toggle(item.subdomain_id)} />
                    </TableCell>
                    <TableCell className="font-mono text-xs">{domainToUnicode(item.fqdn)}</TableCell>
                    <TableCell className="text-xs">
                      <CopyableEmail email={item.user_email} className="max-w-[200px]" />
                    </TableCell>
                    <TableCell className="text-xs">
                      {item.never_scanned
                        ? t("audit.scanStatus.never")
                        : t(`audit.scanStatus.${item.latest_scan_status ?? "clean"}`, { defaultValue: item.latest_scan_status })}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {item.latest_scan?.created_at ? formatDate(item.latest_scan.created_at) : "—"}
                      {item.hours_since_scan != null && (
                        <span className="block text-[10px]">{t("audit.hoursSinceScan", { hours: Math.round(item.hours_since_scan) })}</span>
                      )}
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
    </div>
  );
}

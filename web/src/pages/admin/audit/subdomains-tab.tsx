import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { api } from "@/lib/api";
import { domainToUnicode } from "@/lib/idn";
import { formatDate } from "@/lib/format-date";
import { useErrorToast } from "@/hooks/use-error-toast";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { Paginator } from "@/components/ui/paginator";
import { CopyableEmail } from "@/components/ui/copyable-email";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import type { AuditWorkbenchItem } from "@/types";
import { AuditDetailSheet } from "./audit-detail-sheet";
import { AuditTableFilters } from "./audit-table-filters";
import {
  RecordTypeFilter, recordTypeFilterParam, type AuditRecordType,
} from "./record-type-filter";
import { ReleaseSubdomainDialog } from "./release-subdomain-dialog";

const SCAN_STATUS_OPTIONS = ["all", "scanned", "never", "violation", "compliant"] as const;
const BULK_RESCAN_MAX = 100;

function canSelectForRescan(item: AuditWorkbenchItem): boolean {
  return item.status === "active" && item.scannable;
}

export function SubdomainsTab() {
  const { t } = useTranslation();
  const showError = useErrorToast();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [domainFilter, setDomainFilter] = useState("");
  const [groupFilter, setGroupFilter] = useState("");
  const [recordTypes, setRecordTypes] = useState<AuditRecordType[]>([]);
  const [scanStatus, setScanStatus] = useState<string>("all");
  const [detailId, setDetailId] = useState<number | null>(null);
  const [releaseId, setReleaseId] = useState<number | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());

  const { data, isLoading } = useQuery({
    queryKey: ["admin-audit-cases", page, search, domainFilter, groupFilter, recordTypes.join(","), scanStatus],
    queryFn: async () => {
      const res = await api.adminListAuditCases(page, 20, {
        search: search || undefined,
        domain_id: domainFilter || undefined,
        group_id: groupFilter || undefined,
        record_type: recordTypeFilterParam(recordTypes),
        scan_status: scanStatus === "all" ? undefined : scanStatus,
        status: "active,suspended",
      });
      return { items: res.data, total: res.total };
    },
    staleTime: 15_000,
  });

  const items = data?.items ?? [];
  const totalPages = Math.ceil((data?.total ?? 0) / 20);

  const selectableItems = useMemo(
    () => items.filter(canSelectForRescan),
    [items],
  );

  const allPageSelected = selectableItems.length > 0
    && selectableItems.every((item) => selectedIds.has(item.subdomain_id));
  const somePageSelected = selectableItems.some((item) => selectedIds.has(item.subdomain_id));

  useEffect(() => {
    setSelectedIds(new Set());
  }, [search, domainFilter, groupFilter, recordTypes.join(","), scanStatus]);

  const bulkRescanMutation = useMutation({
    mutationFn: (ids: number[]) => api.adminBulkRescanAudit(ids),
    onSuccess: (res) => {
      const { queued, skipped } = res.data;
      if (skipped > 0) {
        toast.success(t("audit.actions.bulkRescanPartial", { queued, skipped }));
      } else {
        toast.success(t("audit.actions.bulkRescanQueued", { count: queued }));
      }
      setSelectedIds(new Set());
      queryClient.invalidateQueries({ queryKey: ["admin-audit-cases"] });
    },
    onError: (err) => showError(err),
  });

  const toggleSelect = (id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleSelectAllPage = () => {
    if (allPageSelected) {
      setSelectedIds((prev) => {
        const next = new Set(prev);
        for (const item of selectableItems) next.delete(item.subdomain_id);
        return next;
      });
      return;
    }
    setSelectedIds((prev) => {
      const next = new Set(prev);
      for (const item of selectableItems) next.add(item.subdomain_id);
      return next;
    });
  };

  const handleBulkRescan = () => {
    const ids = [...selectedIds];
    if (ids.length === 0) return;
    if (ids.length > BULK_RESCAN_MAX) {
      toast.error(t("error.auditBulkRescanTooMany"));
      return;
    }
    bulkRescanMutation.mutate(ids);
  };

  const resetPage = () => setPage(1);

  return (
    <div className="space-y-4">
      <AuditTableFilters
        search={search}
        onSearchChange={(v) => { setSearch(v); resetPage(); }}
        searchPlaceholder={t("audit.filters.search")}
        domainFilter={domainFilter}
        onDomainFilterChange={(v) => { setDomainFilter(v); resetPage(); }}
        groupFilter={groupFilter}
        onGroupFilterChange={(v) => { setGroupFilter(v); resetPage(); }}
      >
        <RecordTypeFilter
          selected={recordTypes}
          onChange={(v) => { setRecordTypes(v); resetPage(); }}
        />
        <Select value={scanStatus} onValueChange={(v) => { setScanStatus(v); resetPage(); }}>
          <SelectTrigger className="h-9 w-36 shrink-0">
            <SelectValue placeholder={t("audit.filters.scanStatusAll")} />
          </SelectTrigger>
          <SelectContent>
            {SCAN_STATUS_OPTIONS.map((value) => (
              <SelectItem key={value} value={value}>
                {t(`audit.filters.scanStatus.${value}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </AuditTableFilters>

      {selectedIds.size > 0 && (
        <div className="flex flex-wrap items-center gap-3 rounded-lg border bg-muted/40 px-4 py-2">
          <span className="text-sm text-muted-foreground">
            {t("adminNotifications.selectedCount", { count: selectedIds.size })}
          </span>
          <Button
            size="sm"
            variant="outline"
            onClick={handleBulkRescan}
            disabled={bulkRescanMutation.isPending}
          >
            {t("audit.actions.bulkRescan", { count: selectedIds.size })}
          </Button>
        </div>
      )}

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
                      checked={allPageSelected ? true : somePageSelected ? "indeterminate" : false}
                      onCheckedChange={toggleSelectAllPage}
                      disabled={selectableItems.length === 0}
                    />
                  </TableHead>
                  <TableHead>{t("audit.columns.subdomain")}</TableHead>
                  <TableHead>{t("audit.columns.user")}</TableHead>
                  <TableHead>{t("audit.columns.violation")}</TableHead>
                  <TableHead>{t("audit.columns.lastScan")}</TableHead>
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
                {items.map((item: AuditWorkbenchItem) => {
                  const selectable = canSelectForRescan(item);
                  const rowHint = item.status !== "active"
                    ? t("audit.status.suspended")
                    : !item.scannable
                      ? t("audit.detail.notScannableHint")
                      : undefined;
                  return (
                  <TableRow key={item.subdomain_id}>
                    <TableCell>
                      <Checkbox
                        checked={selectedIds.has(item.subdomain_id)}
                        onCheckedChange={() => toggleSelect(item.subdomain_id)}
                        disabled={!selectable}
                        title={rowHint}
                        aria-label={domainToUnicode(item.fqdn)}
                      />
                    </TableCell>
                    <TableCell className="font-mono text-xs">{domainToUnicode(item.fqdn)}</TableCell>
                    <TableCell className="text-xs">
                      <CopyableEmail email={item.user_email} className="max-w-[200px]" />
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground max-w-[240px] truncate">
                      {item.latest_violation?.matched_rule_name ?? "—"}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {item.latest_scan?.created_at ? formatDate(item.latest_scan.created_at) : "—"}
                    </TableCell>
                    <TableCell className="text-right space-x-1">
                      <Button variant="ghost" size="sm" onClick={() => setDetailId(item.subdomain_id)}>
                        {t("audit.actions.detail")}
                      </Button>
                      {item.status === "active" && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-destructive hover:text-destructive"
                          onClick={() => setReleaseId(item.subdomain_id)}
                        >
                          {t("audit.actions.release")}
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <div className="flex justify-center">
        <Paginator page={page} totalPages={totalPages} onPageChange={setPage} />
      </div>

      <AuditDetailSheet subdomainId={detailId} onClose={() => setDetailId(null)} />
      <ReleaseSubdomainDialog
        subdomainId={releaseId}
        open={releaseId != null}
        onOpenChange={(open) => { if (!open) setReleaseId(null); }}
      />
    </div>
  );
}

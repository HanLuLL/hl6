import { useState, useEffect, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, getErrorMessage } from "@/lib/api";
import { toast } from "sonner";
import { Copy } from "lucide-react";
import type { AdminDNSRecord } from "@/types";

export function DNSRecordsContent() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [domainFilter, setDomainFilter] = useState<string>("");
  const [groupFilter, setGroupFilter] = useState<string>("");

  const [selectedIndex, setSelectedIndex] = useState(-1);
  const [detailRecord, setDetailRecord] = useState<AdminDNSRecord | null>(null);
  const [deleteRecord, setDeleteRecord] = useState<AdminDNSRecord | null>(null);
  const [sendNotify, setSendNotify] = useState(false);
  const [reason, setReason] = useState("");

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  const { data, isLoading } = useQuery({
    queryKey: ["admin-dns-records", page, debouncedSearch, domainFilter, groupFilter],
    queryFn: async () => {
      const res = await api.adminListDNSRecords(
        page, 20, debouncedSearch,
        domainFilter ? Number(domainFilter) : undefined,
        groupFilter ? Number(groupFilter) : undefined
      );
      return { records: res.data, total: res.total };
    },
    staleTime: 30_000,
  });

  const { data: domains } = useQuery({
    queryKey: ["admin-domains-list"],
    queryFn: async () => {
      const res = await api.adminListDomainsFull();
      return res.data;
    },
    staleTime: 60_000,
  });

  const { data: groups } = useQuery({
    queryKey: ["admin-groups"],
    queryFn: async () => {
      const res = await api.adminListGroups();
      return res.data;
    },
    staleTime: 60_000,
  });

  const deleteMutation = useMutation({
    mutationFn: ({ id, notify, reason }: { id: number; notify: boolean; reason?: string }) =>
      api.adminDeleteDNSRecord(id, { notify, reason }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-dns-records"] });
      toast.success(t("adminDnsRecords.recordDeleted"));
      setDeleteRecord(null);
      setDetailRecord(null);
      setSendNotify(false);
      setReason("");
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  const records = data?.records ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / 20);

  // Keyboard navigation
  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    const target = e.target as HTMLElement;
    if (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.tagName === "SELECT") return;
    if (detailRecord || deleteRecord) return;

    if (e.key === "ArrowDown") {
      e.preventDefault();
      setSelectedIndex((prev) => Math.min(prev + 1, records.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setSelectedIndex((prev) => Math.max(prev - 1, 0));
    } else if (
      e.key === "Enter" &&
      !e.metaKey &&
      !e.ctrlKey &&
      !e.shiftKey &&
      !e.altKey &&
      selectedIndex >= 0 &&
      selectedIndex < records.length
    ) {
      e.preventDefault();
      setDetailRecord(records[selectedIndex]);
    }
  }, [records, selectedIndex, detailRecord, deleteRecord]);

  useEffect(() => {
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  // Reset selection on data change
  useEffect(() => {
    setSelectedIndex(-1);
  }, [data]);

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex gap-2">
          <Skeleton className="h-9 flex-1" />
          <Skeleton className="h-9 w-40" />
          <Skeleton className="h-9 w-40" />
        </div>
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("adminDnsRecords.fqdn")}</TableHead>
                  <TableHead>{t("adminDnsRecords.content")}</TableHead>
                  <TableHead>{t("adminDnsRecords.type")}</TableHead>
                  <TableHead>{t("adminDnsRecords.userEmail")}</TableHead>
                  <TableHead>{t("adminDnsRecords.createdAt")}</TableHead>
                  <TableHead className="text-right">{t("adminDnsRecords.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {[...Array(5)].map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><Skeleton className="h-4 w-32" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-28" /></TableCell>
                    <TableCell><Skeleton className="h-5 w-12 rounded-full" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-36" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-20" /></TableCell>
                    <TableCell className="text-right"><Skeleton className="h-8 w-14 ml-auto" /></TableCell>
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
    <div className="space-y-4">
      {/* Filters */}
      <div className="flex flex-wrap gap-2">
        <Input
          className="max-w-xs"
          placeholder={t("adminDnsRecords.searchPlaceholder")}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <Select value={domainFilter} onValueChange={(v) => { setDomainFilter(v === "all" ? "" : v); setPage(1); }}>
          <SelectTrigger className="w-48">
            <SelectValue placeholder={t("adminDnsRecords.filterByDomain")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t("adminDnsRecords.filterByDomain")}</SelectItem>
            {domains?.map((d) => (
              <SelectItem key={d.id} value={String(d.id)}>{d.name}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={groupFilter} onValueChange={(v) => { setGroupFilter(v === "all" ? "" : v); setPage(1); }}>
          <SelectTrigger className="w-48">
            <SelectValue placeholder={t("adminDnsRecords.filterByGroup")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t("adminDnsRecords.filterByGroup")}</SelectItem>
            {groups?.map((g) => (
              <SelectItem key={g.id} value={String(g.id)}>{g.name}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Table */}
      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("adminDnsRecords.fqdn")}</TableHead>
                <TableHead>{t("adminDnsRecords.content")}</TableHead>
                <TableHead>{t("adminDnsRecords.type")}</TableHead>
                <TableHead>{t("adminDnsRecords.userEmail")}</TableHead>
                <TableHead>{t("adminDnsRecords.createdAt")}</TableHead>
                <TableHead className="text-right">{t("adminDnsRecords.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {records.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                    {t("adminDnsRecords.noRecords")}
                  </TableCell>
                </TableRow>
              )}
              {records.map((record, idx) => (
                <TableRow
                  key={record.id}
                  className={`cursor-pointer ${idx === selectedIndex ? "bg-accent" : ""}`}
                  onClick={() => setDetailRecord(record)}
                >
                  <TableCell className="font-mono text-xs">
                    <div className="flex items-center gap-2">
                      <span className="truncate">{record.name}</span>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6 shrink-0"
                        onClick={(e) => {
                          e.stopPropagation();
                          navigator.clipboard.writeText(record.name).then(() => {
                            toast.warning(t("adminDnsRecords.copySuccessSafe"));
                          }).catch(() => {});
                        }}
                        aria-label={t("referral.copy")}
                      >
                        <Copy className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </TableCell>
                  <TableCell className="font-mono text-xs max-w-48 truncate">{record.content}</TableCell>
                  <TableCell>
                    <Badge variant="outline" className="text-xs">{record.type}</Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">{record.user_email}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {new Date(record.created_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                      onClick={(e) => {
                        e.stopPropagation();
                        setDeleteRecord(record);
                        setSendNotify(false);
                        setReason("");
                      }}
                    >
                      {t("common.delete")}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            {t("common.pageOf", { page, total: totalPages })}
          </p>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage(page - 1)}>
              {t("common.previous")}
            </Button>
            <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>
              {t("common.next")}
            </Button>
          </div>
        </div>
      )}

      {/* Detail Dialog */}
      <RecordDetailDialog
        record={detailRecord}
        onClose={() => setDetailRecord(null)}
        onDelete={(r) => {
          setDeleteRecord(r);
          setSendNotify(false);
          setReason("");
        }}
      />

      {/* Delete Confirm Dialog */}
      <Dialog open={!!deleteRecord} onOpenChange={(open) => {
        if (!open) { setDeleteRecord(null); setSendNotify(false); setReason(""); }
      }}>
        <DialogContent aria-describedby="delete-dns-desc">
          <DialogHeader>
            <DialogTitle>{t("adminDnsRecords.deleteRecord")}</DialogTitle>
            <DialogDescription id="delete-dns-desc">
              {t("adminDnsRecords.deleteConfirm", { name: deleteRecord?.name })}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="flex items-start gap-2">
              <input
                type="checkbox"
                id="notify-toggle"
                checked={sendNotify}
                onChange={(e) => setSendNotify(e.target.checked)}
                className="mt-0.5 cursor-pointer"
              />
              <label htmlFor="notify-toggle" className="text-sm font-medium cursor-pointer">
                {t("adminDnsRecords.sendNotification")}
              </label>
            </div>
            {sendNotify && (
              <div className="space-y-2">
                <label className="text-sm text-muted-foreground">{t("adminDnsRecords.reason")}</label>
                <Textarea
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  placeholder={t("adminDnsRecords.reasonPlaceholder")}
                />
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setDeleteRecord(null); setSendNotify(false); setReason(""); }} disabled={deleteMutation.isPending}>
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              disabled={deleteMutation.isPending}
              onClick={() => deleteRecord && deleteMutation.mutate({
                id: deleteRecord.id,
                notify: sendNotify,
                reason: sendNotify ? reason : undefined,
              })}
              data-dialog-primary="true"
            >
              {deleteMutation.isPending ? t("common.deleting") : t("common.delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function RecordDetailDialog({ record, onClose, onDelete }: {
  record: AdminDNSRecord | null;
  onClose: () => void;
  onDelete: (record: AdminDNSRecord) => void;
}) {
  const { t } = useTranslation();

  // Enter to open delete, Escape handled by Dialog
  useEffect(() => {
    if (!record) return;
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === "Enter" && !e.metaKey && !e.ctrlKey && !e.shiftKey && !e.altKey) {
        const target = e.target as HTMLElement;
        if (target.tagName === "INPUT" || target.tagName === "TEXTAREA") return;
        e.preventDefault();
        onDelete(record);
      }
    };
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [record, onDelete]);

  return (
    <Dialog open={!!record} onOpenChange={(open) => !open && onClose()}>
      <DialogContent aria-describedby={undefined}>
        <DialogHeader>
          <DialogTitle>{t("adminDnsRecords.recordDetail")}</DialogTitle>
        </DialogHeader>
        {record && (
          <div className="space-y-3 py-2">
            <DetailRow label={t("adminDnsRecords.fqdn")} value={record.name} mono />
            <DetailRow label={t("adminDnsRecords.type")} value={record.type} />
            <DetailRow label={t("adminDnsRecords.content")} value={record.content} mono />
            <DetailRow label={t("adminDnsRecords.ttl")} value={record.ttl === 1 ? "Auto" : String(record.ttl)} />
            <DetailRow label={t("adminDnsRecords.proxied")} value={record.proxied ? t("common.on") : t("common.off")} />
            <DetailRow label={t("adminDnsRecords.userEmail")} value={record.user_email} />
            <DetailRow label={t("adminDnsRecords.userName")} value={record.user_name} />
            <DetailRow label={t("adminDnsRecords.domain")} value={record.domain_name} />
            <DetailRow label={t("adminDnsRecords.createdAt")} value={new Date(record.created_at).toLocaleString()} />
            <DetailRow label={t("adminDnsRecords.cloudflareId")} value={record.cloudflare_record_id} mono />
          </div>
        )}
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>{t("common.cancel")}</Button>
          <Button
            variant="destructive"
            onClick={() => record && onDelete(record)}
            data-dialog-primary="true"
          >
            {t("adminDnsRecords.deleteRecord")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function DetailRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex gap-4 items-start">
      <span className="text-sm text-muted-foreground min-w-28 shrink-0">{label}</span>
      <span className={`text-sm break-all ${mono ? "font-mono" : ""}`}>{value || "-"}</span>
    </div>
  );
}

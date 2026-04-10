import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, getErrorMessage } from "@/lib/api";
import type { AdminClaimedSubdomain } from "@/types";
import { toast } from "sonner";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { CopyableEmail } from "@/components/ui/copyable-email";

export function ClaimedSubdomainsContent() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");

  const [releaseTarget, setReleaseTarget] = useState<AdminClaimedSubdomain | null>(null);
  const [sendNotify, setSendNotify] = useState(false);
  const [reason, setReason] = useState("");

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search.trim());
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  const { data, isLoading } = useQuery({
    queryKey: ["admin-claimed-subdomains", page, debouncedSearch],
    queryFn: async () => {
      const res = await api.adminListClaimedSubdomains(page, 20, debouncedSearch);
      return {
        items: res.data,
        total: res.total,
      };
    },
    staleTime: 30_000,
  });

  const releaseMutation = useMutation({
    mutationFn: ({ id, notify, reason }: { id: number; notify: boolean; reason?: string }) =>
      api.adminReleaseClaimedSubdomain(id, { notify, reason }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-claimed-subdomains"] });
      queryClient.invalidateQueries({ queryKey: ["admin-dns-records"] });
      queryClient.invalidateQueries({ queryKey: ["subdomains"] });
      toast.success(t("adminDomains.claimedReleased"));
      setReleaseTarget(null);
      setSendNotify(false);
      setReason("");
    },
    onError: (err) => {
      toast.error(getErrorMessage(err, t));
    },
  });

  const items = data?.items ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / 20);

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-9 w-72" />
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("adminDomains.claimedFqdn")}</TableHead>
                  <TableHead>{t("adminDomains.claimedUserEmail")}</TableHead>
                  <TableHead>{t("adminDomains.claimedRecordCount")}</TableHead>
                  <TableHead>{t("adminDomains.claimedAt")}</TableHead>
                  <TableHead className="text-right">{t("adminDomains.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {[...Array(5)].map((_, idx) => (
                  <TableRow key={idx}>
                    <TableCell><Skeleton className="h-4 w-44" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-40" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-10" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                    <TableCell className="text-right"><Skeleton className="h-8 w-20 ml-auto" /></TableCell>
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
      <Input
        className="max-w-xs"
        placeholder={t("adminDomains.claimedSearchPlaceholder")}
        value={search}
        onChange={(e) => setSearch(e.target.value)}
      />

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("adminDomains.claimedFqdn")}</TableHead>
                <TableHead>{t("adminDomains.claimedUserEmail")}</TableHead>
                <TableHead>{t("adminDomains.claimedRecordCount")}</TableHead>
                <TableHead>{t("adminDomains.claimedAt")}</TableHead>
                <TableHead className="text-right">{t("adminDomains.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.length === 0 && (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                    {t("adminDomains.claimedNoRecords")}
                  </TableCell>
                </TableRow>
              )}
              {items.map((item) => (
                <TableRow key={item.id}>
                  <TableCell className="font-mono text-xs">{item.fqdn}</TableCell>
                  <TableCell className="text-xs">
                    <CopyableEmail email={item.user_email} className="text-muted-foreground max-w-[240px]" />
                  </TableCell>
                  <TableCell>{item.dns_record_count}</TableCell>
                  <TableCell className="text-muted-foreground text-xs">
                    {new Date(item.created_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                      onClick={() => {
                        setReleaseTarget(item);
                        setSendNotify(false);
                        setReason("");
                      }}
                    >
                      {t("adminDomains.releaseClaimed")}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

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

      <Dialog open={!!releaseTarget} onOpenChange={(open) => {
        if (!open) {
          setReleaseTarget(null);
          setSendNotify(false);
          setReason("");
        }
      }}>
        <DialogContent aria-describedby="release-claimed-desc">
          <DialogHeader>
            <DialogTitle>{t("adminDomains.releaseClaimed")}</DialogTitle>
            <DialogDescription id="release-claimed-desc">
              {t("adminDomains.releaseClaimedConfirm", { fqdn: releaseTarget?.fqdn })}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="flex items-start gap-2">
              <input
                type="checkbox"
                id="release-notify-toggle"
                checked={sendNotify}
                onChange={(e) => setSendNotify(e.target.checked)}
                className="mt-0.5 cursor-pointer"
              />
              <label htmlFor="release-notify-toggle" className="text-sm font-medium cursor-pointer">
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
            <Button
              variant="outline"
              onClick={() => {
                setReleaseTarget(null);
                setSendNotify(false);
                setReason("");
              }}
              disabled={releaseMutation.isPending}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              disabled={releaseMutation.isPending}
              onClick={() => releaseTarget && releaseMutation.mutate({
                id: releaseTarget.id,
                notify: sendNotify,
                reason: sendNotify ? reason : undefined,
              })}
              data-dialog-primary="true"
            >
              {releaseMutation.isPending ? t("common.deleting") : t("adminDomains.releaseClaimed")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, getErrorMessage } from "@/lib/api";
import { toast } from "sonner";
import type { CloudflareAccount } from "@/types";
import { Skeleton } from "@/components/ui/skeleton";

export default function AdminCloudflareAccountsPage() {
  return <CloudflareAccountsContent />;
}

export function CloudflareAccountsContent() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  const { data: accounts, isLoading } = useQuery({
    queryKey: ["admin-cloudflare-accounts"],
    queryFn: async () => {
      const res = await api.adminListCloudflareAccounts();
      return res.data;
    },
    staleTime: 30_000,
  });

  const [showAdd, setShowAdd] = useState(false);
  const [addName, setAddName] = useState("");
  const [addToken, setAddToken] = useState("");

  const [editAccount, setEditAccount] = useState<CloudflareAccount | null>(null);
  const [editName, setEditName] = useState("");
  const [editToken, setEditToken] = useState("");

  const [deleteAccount, setDeleteAccount] = useState<CloudflareAccount | null>(null);

  const createMutation = useMutation({
    mutationFn: api.adminCreateCloudflareAccount,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-cloudflare-accounts"] });
      toast.success(t("adminCloudflare.accountCreated"));
      setShowAdd(false);
      setAddName("");
      setAddToken("");
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, ...data }: { id: number; name: string; api_token?: string }) =>
      api.adminUpdateCloudflareAccount(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-cloudflare-accounts"] });
      toast.success(t("adminCloudflare.accountUpdated"));
      setEditAccount(null);
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.adminDeleteCloudflareAccount(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-cloudflare-accounts"] });
      toast.success(t("adminCloudflare.accountDeleted"));
      setDeleteAccount(null);
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-end">
          <Skeleton className="h-9 w-24" />
        </div>
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("adminCloudflare.accountName")}</TableHead>
                  <TableHead>{t("adminCloudflare.tokenHint")}</TableHead>
                  <TableHead className="text-right">{t("adminDomains.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {[...Array(3)].map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><Skeleton className="h-4 w-28" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-20" /></TableCell>
                    <TableCell className="text-right"><Skeleton className="h-8 w-28 ml-auto" /></TableCell>
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
      <div className="flex items-center justify-end">
        <Button onClick={() => setShowAdd(true)}>{t("adminCloudflare.addAccount")}</Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("adminCloudflare.accountName")}</TableHead>
                <TableHead>{t("adminCloudflare.tokenHint")}</TableHead>
                <TableHead className="text-right">{t("adminDomains.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {accounts?.length === 0 && (
                <TableRow>
                  <TableCell colSpan={3} className="text-center text-muted-foreground py-8">
                    {t("adminCloudflare.noAccounts")}
                  </TableCell>
                </TableRow>
              )}
              {accounts?.map((account) => (
                <TableRow key={account.id}>
                  <TableCell className="font-medium">{account.name}</TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">{account.token_hint}</TableCell>
                  <TableCell className="text-right space-x-1">
                    <Button variant="ghost" size="sm" onClick={() => {
                      setEditAccount(account);
                      setEditName(account.name);
                      setEditToken("");
                    }}>{t("common.edit")}</Button>
                    <Button variant="ghost" size="sm" className="text-destructive" onClick={() => {
                      setDeleteAccount(account);
                    }}>{t("common.delete")}</Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Add Dialog */}
      <Dialog open={showAdd} onOpenChange={(open) => {
        setShowAdd(open);
        if (!open) { setAddName(""); setAddToken(""); }
      }}>
        <DialogContent aria-describedby={undefined}>
          <DialogHeader><DialogTitle>{t("adminCloudflare.addAccount")}</DialogTitle></DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>{t("adminCloudflare.accountName")}</Label>
              <Input value={addName} onChange={(e) => setAddName(e.target.value)} placeholder={t("adminCloudflare.accountName")} required />
            </div>
            <div className="space-y-2">
              <Label>{t("adminCloudflare.apiToken")}</Label>
              <Input type="password" value={addToken} onChange={(e) => setAddToken(e.target.value)} placeholder="API Token" required />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAdd(false)}>{t("common.cancel")}</Button>
            <Button
              onClick={() => createMutation.mutate({ name: addName, api_token: addToken })}
              disabled={!addName.trim() || !addToken.trim() || createMutation.isPending}
              data-dialog-primary="true"
            >
              {createMutation.isPending ? t("common.creating") : t("common.create")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog open={!!editAccount} onOpenChange={(open) => !open && setEditAccount(null)}>
        <DialogContent aria-describedby={undefined}>
          <DialogHeader><DialogTitle>{t("adminCloudflare.editAccount")}</DialogTitle></DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>{t("adminCloudflare.accountName")}</Label>
              <Input value={editName} onChange={(e) => setEditName(e.target.value)} required />
            </div>
            <div className="space-y-2">
              <Label>{t("adminCloudflare.apiToken")}</Label>
              <Input
                type="password"
                value={editToken}
                onChange={(e) => setEditToken(e.target.value)}
                placeholder={t("adminCloudflare.apiTokenHint")}
              />
              {editAccount && (
                <p className="text-xs text-muted-foreground">
                  {t("adminCloudflare.tokenHint")}: {editAccount.token_hint}
                </p>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditAccount(null)}>{t("common.cancel")}</Button>
            <Button
              onClick={() => editAccount && updateMutation.mutate({
                id: editAccount.id,
                name: editName,
                ...(editToken ? { api_token: editToken } : {}),
              })}
              disabled={!editName.trim() || updateMutation.isPending}
              data-dialog-primary="true"
            >
              {updateMutation.isPending ? t("common.saving") : t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirm Dialog */}
      <Dialog open={!!deleteAccount} onOpenChange={(open) => !open && setDeleteAccount(null)}>
        <DialogContent aria-describedby={undefined}>
          <DialogHeader><DialogTitle>{t("common.delete")}</DialogTitle></DialogHeader>
          <p className="text-sm text-muted-foreground py-4">
            {t("adminCloudflare.deleteConfirm", { name: deleteAccount?.name })}
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteAccount(null)}>{t("common.cancel")}</Button>
            <Button
              variant="destructive"
              onClick={() => deleteAccount && deleteMutation.mutate(deleteAccount.id)}
              disabled={deleteMutation.isPending}
              data-dialog-primary="true"
            >
              {deleteMutation.isPending ? t("common.delete") + "..." : t("common.delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

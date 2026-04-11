import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api, getErrorMessage } from "@/lib/api";
import type { DNSProviderAccount } from "@/types";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";

type ProviderField = { key: string; label: string; secret?: boolean };

const providerOptions: Array<{ value: string; label: string; fields: ProviderField[] }> = [
  { value: "cloudflare", label: "Cloudflare", fields: [{ key: "api_token", label: "API Token", secret: true }] },
  { value: "dnspod", label: "DNSPod", fields: [{ key: "secret_id", label: "SecretId" }, { key: "secret_key", label: "SecretKey", secret: true }] },
  { value: "aliyun_dns", label: "阿里云 DNS", fields: [{ key: "access_key_id", label: "AccessKey ID" }, { key: "access_key_secret", label: "AccessKey Secret", secret: true }, { key: "region_id", label: "Region ID (可选)" }, { key: "endpoint", label: "Endpoint (可选)" }] },
  { value: "huawei_cloud_dns", label: "华为云 DNS", fields: [{ key: "ak", label: "AK" }, { key: "sk", label: "SK", secret: true }, { key: "region", label: "Region (可选)" }, { key: "endpoint", label: "Endpoint (可选)" }, { key: "project_id", label: "Project ID (可选)" }] },
];

function emptyCredentialState(provider: string): Record<string, string> {
  const found = providerOptions.find((p) => p.value === provider);
  if (!found) return {};
  return Object.fromEntries(found.fields.map((f) => [f.key, ""]));
}

function providerLabel(provider: string): string {
  return providerOptions.find((p) => p.value === provider)?.label ?? provider;
}

export default function AdminDNSProviderAccountsPage() {
  return <DNSProviderAccountsContent />;
}

export function DNSProviderAccountsContent() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();

  const { data: accounts, isLoading } = useQuery({
    queryKey: ["admin-dns-provider-accounts"],
    queryFn: async () => (await api.adminListDNSProviderAccounts()).data,
    staleTime: 30_000,
  });

  const [showAdd, setShowAdd] = useState(false);
  const [addProvider, setAddProvider] = useState("cloudflare");
  const [addName, setAddName] = useState("");
  const [addCredentials, setAddCredentials] = useState<Record<string, string>>(emptyCredentialState("cloudflare"));

  const [editAccount, setEditAccount] = useState<DNSProviderAccount | null>(null);
  const [editProvider, setEditProvider] = useState("cloudflare");
  const [editName, setEditName] = useState("");
  const [editCredentials, setEditCredentials] = useState<Record<string, string>>(emptyCredentialState("cloudflare"));

  const [deleteAccount, setDeleteAccount] = useState<DNSProviderAccount | null>(null);

  const addFields = useMemo(() => providerOptions.find((p) => p.value === addProvider)?.fields ?? [], [addProvider]);
  const editFields = useMemo(() => providerOptions.find((p) => p.value === editProvider)?.fields ?? [], [editProvider]);

  const createMutation = useMutation({
    mutationFn: api.adminCreateDNSProviderAccount,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-dns-provider-accounts"] });
      toast.success("供应商账号已创建");
      setShowAdd(false);
      setAddProvider("cloudflare");
      setAddName("");
      setAddCredentials(emptyCredentialState("cloudflare"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, ...data }: { id: number; provider?: string; name: string; credentials?: Record<string, string> }) =>
      api.adminUpdateDNSProviderAccount(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-dns-provider-accounts"] });
      toast.success("供应商账号已更新");
      setEditAccount(null);
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.adminDeleteDNSProviderAccount(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-dns-provider-accounts"] });
      toast.success("供应商账号已删除");
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
                  <TableHead>供应商</TableHead>
                  <TableHead>账号名</TableHead>
                  <TableHead>凭证标识</TableHead>
                  <TableHead className="text-right">{t("adminDomains.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {[...Array(3)].map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><Skeleton className="h-4 w-28" /></TableCell>
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
        <Button onClick={() => setShowAdd(true)}>添加供应商账号</Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>供应商</TableHead>
                <TableHead>账号名</TableHead>
                <TableHead>凭证标识</TableHead>
                <TableHead className="text-right">{t("adminDomains.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {accounts?.length === 0 && (
                <TableRow>
                  <TableCell colSpan={4} className="text-center text-muted-foreground py-8">
                    暂无供应商账号
                  </TableCell>
                </TableRow>
              )}
              {accounts?.map((account) => (
                <TableRow key={account.id}>
                  <TableCell>{providerLabel(account.provider)}</TableCell>
                  <TableCell className="font-medium">{account.name}</TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">{account.credential_hint}</TableCell>
                  <TableCell className="text-right space-x-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        setEditAccount(account);
                        setEditProvider(account.provider);
                        setEditName(account.name);
                        setEditCredentials(emptyCredentialState(account.provider));
                      }}
                    >
                      {t("common.edit")}
                    </Button>
                    <Button variant="ghost" size="sm" className="text-destructive" onClick={() => setDeleteAccount(account)}>
                      {t("common.delete")}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Dialog open={showAdd} onOpenChange={(open) => {
        setShowAdd(open);
        if (!open) {
          setAddProvider("cloudflare");
          setAddName("");
          setAddCredentials(emptyCredentialState("cloudflare"));
        }
      }}>
        <DialogContent aria-describedby={undefined}>
          <DialogHeader><DialogTitle>添加供应商账号</DialogTitle></DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>供应商</Label>
              <Select
                value={addProvider}
                onValueChange={(next) => {
                  setAddProvider(next);
                  setAddCredentials(emptyCredentialState(next));
                }}
              >
                <SelectTrigger><SelectValue placeholder="请选择供应商" /></SelectTrigger>
                <SelectContent>
                  {providerOptions.map((p) => (
                    <SelectItem key={p.value} value={p.value}>{p.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>账号名称</Label>
              <Input value={addName} onChange={(e) => setAddName(e.target.value)} placeholder="例如：主账号" />
            </div>
            {addFields.map((field) => (
              <div className="space-y-2" key={field.key}>
                <Label>{field.label}</Label>
                <Input
                  type={field.secret ? "password" : "text"}
                  value={addCredentials[field.key] ?? ""}
                  onChange={(e) => setAddCredentials((prev) => ({ ...prev, [field.key]: e.target.value }))}
                />
              </div>
            ))}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAdd(false)}>{t("common.cancel")}</Button>
            <Button
              onClick={() => createMutation.mutate({ provider: addProvider, name: addName, credentials: addCredentials })}
              disabled={!addName.trim() || createMutation.isPending}
              data-dialog-primary="true"
            >
              {createMutation.isPending ? t("common.creating") : t("common.create")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={!!editAccount} onOpenChange={(open) => !open && setEditAccount(null)}>
        <DialogContent aria-describedby={undefined}>
          <DialogHeader><DialogTitle>编辑供应商账号</DialogTitle></DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>供应商</Label>
              <Select
                value={editProvider}
                onValueChange={(next) => {
                  setEditProvider(next);
                  setEditCredentials(emptyCredentialState(next));
                }}
              >
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {providerOptions.map((p) => (
                    <SelectItem key={p.value} value={p.value}>{p.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>账号名称</Label>
              <Input value={editName} onChange={(e) => setEditName(e.target.value)} />
            </div>
            {editFields.map((field) => (
              <div className="space-y-2" key={field.key}>
                <Label>{field.label}</Label>
                <Input
                  type={field.secret ? "password" : "text"}
                  value={editCredentials[field.key] ?? ""}
                  onChange={(e) => setEditCredentials((prev) => ({ ...prev, [field.key]: e.target.value }))}
                  placeholder="留空则不更新"
                />
              </div>
            ))}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditAccount(null)}>{t("common.cancel")}</Button>
            <Button
              onClick={() => {
                if (!editAccount) return;
                const nextCredentials = Object.fromEntries(
                  Object.entries(editCredentials).filter(([, v]) => v.trim() !== ""),
                );
                updateMutation.mutate({
                  id: editAccount.id,
                  provider: editProvider,
                  name: editName,
                  ...(Object.keys(nextCredentials).length > 0 ? { credentials: nextCredentials } : {}),
                });
              }}
              disabled={!editName.trim() || updateMutation.isPending}
              data-dialog-primary="true"
            >
              {updateMutation.isPending ? t("common.saving") : t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={!!deleteAccount} onOpenChange={(open) => !open && setDeleteAccount(null)}>
        <DialogContent aria-describedby={undefined}>
          <DialogHeader><DialogTitle>{t("common.delete")}</DialogTitle></DialogHeader>
          <p className="text-sm text-muted-foreground py-4">
            确定删除供应商账号「{deleteAccount?.name}」？
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteAccount(null)}>{t("common.cancel")}</Button>
            <Button
              variant="destructive"
              onClick={() => deleteAccount && deleteMutation.mutate(deleteAccount.id)}
              disabled={deleteMutation.isPending}
              data-dialog-primary="true"
            >
              {deleteMutation.isPending ? `${t("common.delete")}...` : t("common.delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

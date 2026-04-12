import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api, getErrorMessage } from "@/lib/api";
import type { DNSProviderAccount, DNSProviderAccountStatus } from "@/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";

type ProviderField = { key: string; label: string; secret?: boolean; optional?: boolean };

const providerOptions: Array<{ value: string; label: string; fields: ProviderField[] }> = [
  {
    value: "cloudflare",
    label: "Cloudflare",
    fields: [{ key: "api_token", label: "API Token", secret: true }],
  },
  {
    value: "dnspod",
    label: "DNSPod",
    fields: [
      { key: "secret_id", label: "SecretId" },
      { key: "secret_key", label: "SecretKey", secret: true },
      { key: "region", label: "Region（可选）", optional: true },
    ],
  },
  {
    value: "aliyun_dns",
    label: "阿里云 DNS",
    fields: [
      { key: "access_key_id", label: "AccessKey ID" },
      { key: "access_key_secret", label: "AccessKey Secret", secret: true },
      { key: "region_id", label: "Region ID（可选）", optional: true },
      { key: "endpoint", label: "Endpoint（可选）", optional: true },
    ],
  },
  {
    value: "huawei_cloud_dns",
    label: "华为云 DNS",
    fields: [
      { key: "ak", label: "AK" },
      { key: "sk", label: "SK", secret: true },
      { key: "region", label: "Region（可选）", optional: true },
      { key: "endpoint", label: "Endpoint（可选）", optional: true },
      { key: "project_id", label: "Project ID（可选）", optional: true },
    ],
  },
  {
    value: "aws_route53",
    label: "Amazon Route 53",
    fields: [
      { key: "access_key_id", label: "Access Key ID" },
      { key: "access_key_secret", label: "Secret Access Key", secret: true },
      { key: "region", label: "Region（可选，默认 us-east-1）", optional: true },
    ],
  },
  {
    value: "google_cloud_dns",
    label: "Google Cloud DNS",
    fields: [{ key: "service_account_json", label: "Service Account JSON", secret: true }],
  },
  {
    value: "baidu_cloud_dns",
    label: "百度智能云 DNS",
    fields: [
      { key: "access_key", label: "Access Key" },
      { key: "secret_key", label: "Secret Key", secret: true },
    ],
  },
  {
    value: "dns_com",
    label: "DNS.com",
    fields: [
      { key: "api_id", label: "API ID" },
      { key: "api_key", label: "API Key", secret: true },
    ],
  },
  {
    value: "dnsla",
    label: "DNSLA",
    fields: [
      { key: "api_id", label: "API ID" },
      { key: "api_secret", label: "API Secret", secret: true },
    ],
  },
  {
    value: "westcn_dns",
    label: "西部数码",
    fields: [
      { key: "username", label: "用户名" },
      { key: "password", label: "密码", secret: true },
    ],
  },
];

function emptyCredentialState(provider: string): Record<string, string> {
  const found = providerOptions.find((p) => p.value === provider);
  if (!found) return {};
  return Object.fromEntries(found.fields.map((f) => [f.key, ""]));
}

function providerLabel(provider: string): string {
  return providerOptions.find((p) => p.value === provider)?.label ?? provider;
}

function StatusBadge({ status }: { status?: DNSProviderAccountStatus | string }) {
  switch (status) {
    case "active":
      return <Badge variant="default" className="bg-green-500/15 text-green-700 hover:bg-green-500/20 border-green-200">正常</Badge>;
    case "degraded":
      return <Badge variant="secondary" className="bg-yellow-500/15 text-yellow-700 hover:bg-yellow-500/20 border-yellow-200">降级</Badge>;
    case "disabled":
      return <Badge variant="destructive">已禁用</Badge>;
    default:
      return <Badge variant="outline">{status ?? "—"}</Badge>;
  }
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
      toast.success(t("dnsProviderAccount.accountCreated", "供应商账号已创建"));
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
      toast.success(t("dnsProviderAccount.accountUpdated", "供应商账号已更新"));
      setEditAccount(null);
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.adminDeleteDNSProviderAccount(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-dns-provider-accounts"] });
      toast.success(t("dnsProviderAccount.accountDeleted", "供应商账号已删除"));
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
                  <TableHead>状态</TableHead>
                  <TableHead>凭证标识</TableHead>
                  <TableHead className="text-right">{t("adminDomains.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {[...Array(3)].map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><Skeleton className="h-4 w-28" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-28" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-16" /></TableCell>
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
        <Button onClick={() => setShowAdd(true)}>
          {t("dnsProviderAccount.addAccount", "添加供应商账号")}
        </Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("dnsProviderAccount.provider", "供应商")}</TableHead>
                <TableHead>{t("dnsProviderAccount.accountName", "账号名")}</TableHead>
                <TableHead>{t("dnsProviderAccount.status", "状态")}</TableHead>
                <TableHead>{t("dnsProviderAccount.credentialHint", "凭证标识")}</TableHead>
                <TableHead>{t("dnsProviderAccount.lastVerifiedAt", "最近验证")}</TableHead>
                <TableHead className="text-right">{t("adminDomains.actions", "操作")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {accounts?.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                    {t("dnsProviderAccount.noAccounts", "暂无供应商账号")}
                  </TableCell>
                </TableRow>
              )}
              {accounts?.map((account) => (
                <TableRow key={account.id}>
                  <TableCell className="font-medium">{providerLabel(account.provider)}</TableCell>
                  <TableCell>{account.name}</TableCell>
                  <TableCell>
                    <StatusBadge status={account.status} />
                    {account.last_error_category && account.status !== "active" && (
                      <p className="text-xs text-muted-foreground mt-1 max-w-[180px] truncate" title={account.last_error_message ?? ""}>
                        {t(`errorCategories.${account.last_error_category}`, account.last_error_category)}
                      </p>
                    )}
                  </TableCell>
                  <TableCell>
                    <code className="text-xs bg-muted px-1 py-0.5 rounded">{account.credential_hint}</code>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {account.last_verified_at
                      ? new Date(account.last_verified_at).toLocaleString()
                      : "—"}
                  </TableCell>
                  <TableCell className="text-right space-x-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setEditAccount(account);
                        setEditProvider(account.provider);
                        setEditName(account.name);
                        setEditCredentials(emptyCredentialState(account.provider));
                      }}
                    >
                      {t("adminDomains.edit", "编辑")}
                    </Button>
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => setDeleteAccount(account)}
                    >
                      {t("adminDomains.delete", "删除")}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Add Account Dialog */}
      <Dialog open={showAdd} onOpenChange={setShowAdd}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{t("dnsProviderAccount.addAccount", "添加供应商账号")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-1">
              <Label>{t("dnsProviderAccount.provider", "供应商")}</Label>
              <Select
                value={addProvider}
                onValueChange={(v) => {
                  setAddProvider(v);
                  setAddCredentials(emptyCredentialState(v));
                }}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {providerOptions.map((p) => (
                    <SelectItem key={p.value} value={p.value}>{p.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1">
              <Label>{t("dnsProviderAccount.accountName", "账号名称")}</Label>
              <Input
                value={addName}
                onChange={(e) => setAddName(e.target.value)}
                placeholder={t("dnsProviderAccount.accountName", "账号名称")}
              />
            </div>
            {addFields.map((field) => (
              <div key={field.key} className="space-y-1">
                <Label>{field.label}</Label>
                <Input
                  type={field.secret ? "password" : "text"}
                  value={addCredentials[field.key] ?? ""}
                  onChange={(e) =>
                    setAddCredentials((prev) => ({ ...prev, [field.key]: e.target.value }))
                  }
                  placeholder={field.optional ? `${field.label}` : field.label}
                />
              </div>
            ))}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAdd(false)}>
              {t("common.cancel", "取消")}
            </Button>
            <Button
              disabled={createMutation.isPending}
              onClick={() => {
                const creds: Record<string, string> = {};
                for (const [k, v] of Object.entries(addCredentials)) {
                  if (v.trim()) creds[k] = v.trim();
                }
                createMutation.mutate({ provider: addProvider, name: addName, credentials: creds });
              }}
            >
              {createMutation.isPending ? t("common.saving", "保存中...") : t("common.save", "保存")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Account Dialog */}
      <Dialog open={!!editAccount} onOpenChange={(open) => !open && setEditAccount(null)}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{t("dnsProviderAccount.editAccount", "编辑供应商账号")}</DialogTitle>
          </DialogHeader>
          {editAccount && (
            <div className="space-y-4 py-2">
              <div className="space-y-1">
                <Label>{t("dnsProviderAccount.provider", "供应商")}</Label>
                <Select
                  value={editProvider}
                  onValueChange={(v) => {
                    setEditProvider(v);
                    setEditCredentials(emptyCredentialState(v));
                  }}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {providerOptions.map((p) => (
                      <SelectItem key={p.value} value={p.value}>{p.label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1">
                <Label>{t("dnsProviderAccount.accountName", "账号名称")}</Label>
                <Input
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                />
              </div>
              {editFields.map((field) => (
                <div key={field.key} className="space-y-1">
                  <Label>{field.label}</Label>
                  <Input
                    type={field.secret ? "password" : "text"}
                    value={editCredentials[field.key] ?? ""}
                    onChange={(e) =>
                      setEditCredentials((prev) => ({ ...prev, [field.key]: e.target.value }))
                    }
                    placeholder={field.optional ? `${field.label}（留空保持不变）` : `${field.label}（留空保持不变）`}
                  />
                </div>
              ))}
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditAccount(null)}>
              {t("common.cancel", "取消")}
            </Button>
            <Button
              disabled={updateMutation.isPending}
              onClick={() => {
                if (!editAccount) return;
                const creds: Record<string, string> = {};
                for (const [k, v] of Object.entries(editCredentials)) {
                  if (v.trim()) creds[k] = v.trim();
                }
                updateMutation.mutate({
                  id: editAccount.id,
                  provider: editProvider !== editAccount.provider ? editProvider : undefined,
                  name: editName,
                  credentials: Object.keys(creds).length > 0 ? creds : undefined,
                });
              }}
            >
              {updateMutation.isPending ? t("common.saving", "保存中...") : t("common.save", "保存")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirm Dialog */}
      <Dialog open={!!deleteAccount} onOpenChange={(open) => !open && setDeleteAccount(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("dnsProviderAccount.deleteConfirm", "确认删除账号「{{name}}」？").replace("{{name}}", deleteAccount?.name ?? "")}</DialogTitle>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteAccount(null)}>
              {t("common.cancel", "取消")}
            </Button>
            <Button
              variant="destructive"
              disabled={deleteMutation.isPending}
              onClick={() => deleteAccount && deleteMutation.mutate(deleteAccount.id)}
            >
              {deleteMutation.isPending ? t("common.deleting", "删除中...") : t("common.delete", "删除")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

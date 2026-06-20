import { useCallback, useMemo, useState } from "react";
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

type ProviderFieldDef = { key: string; labelKey: string; secret?: boolean; optional?: boolean };

const providerOptionDefs: Array<{ value: string; fields: ProviderFieldDef[] }> = [
  {
    value: "cloudflare",
    fields: [{ key: "api_token", labelKey: "dnsProviderAccount.credentialFields.api_token", secret: true }],
  },
  {
    value: "dnspod",
    fields: [
      { key: "secret_id", labelKey: "dnsProviderAccount.credentialFields.secret_id" },
      { key: "secret_key", labelKey: "dnsProviderAccount.credentialFields.secret_key", secret: true },
      { key: "region", labelKey: "dnsProviderAccount.credentialFields.region_optional", optional: true },
    ],
  },
  {
    value: "aliyun_dns",
    fields: [
      { key: "access_key_id", labelKey: "dnsProviderAccount.credentialFields.access_key_id" },
      { key: "access_key_secret", labelKey: "dnsProviderAccount.credentialFields.access_key_secret", secret: true },
      { key: "region_id", labelKey: "dnsProviderAccount.credentialFields.region_id_optional", optional: true },
      { key: "endpoint", labelKey: "dnsProviderAccount.credentialFields.endpoint_optional", optional: true },
    ],
  },
  {
    value: "huawei_cloud_dns",
    fields: [
      { key: "ak", labelKey: "dnsProviderAccount.credentialFields.ak" },
      { key: "sk", labelKey: "dnsProviderAccount.credentialFields.sk", secret: true },
      { key: "region", labelKey: "dnsProviderAccount.credentialFields.region_optional", optional: true },
      { key: "endpoint", labelKey: "dnsProviderAccount.credentialFields.endpoint_optional", optional: true },
      { key: "project_id", labelKey: "dnsProviderAccount.credentialFields.project_id_optional", optional: true },
    ],
  },
  {
    value: "aws_route53",
    fields: [
      { key: "access_key_id", labelKey: "dnsProviderAccount.credentialFields.access_key_id" },
      { key: "access_key_secret", labelKey: "dnsProviderAccount.credentialFields.secret_access_key", secret: true },
      { key: "region", labelKey: "dnsProviderAccount.credentialFields.region_optional_default", optional: true },
    ],
  },
  {
    value: "google_cloud_dns",
    fields: [{ key: "service_account_json", labelKey: "dnsProviderAccount.credentialFields.service_account_json", secret: true }],
  },
  {
    value: "baidu_cloud_dns",
    fields: [
      { key: "access_key", labelKey: "dnsProviderAccount.credentialFields.access_key" },
      { key: "secret_key", labelKey: "dnsProviderAccount.credentialFields.secret_key", secret: true },
    ],
  },
  {
    value: "dns_com",
    fields: [
      { key: "api_id", labelKey: "dnsProviderAccount.credentialFields.api_id" },
      { key: "api_key", labelKey: "dnsProviderAccount.credentialFields.api_key", secret: true },
    ],
  },
  {
    value: "dnsla",
    fields: [
      { key: "api_id", labelKey: "dnsProviderAccount.credentialFields.api_id" },
      { key: "api_secret", labelKey: "dnsProviderAccount.credentialFields.api_secret", secret: true },
    ],
  },
  {
    value: "westcn_dns",
    fields: [
      { key: "username", labelKey: "dnsProviderAccount.credentialFields.username" },
      { key: "password", labelKey: "dnsProviderAccount.credentialFields.password", secret: true },
    ],
  },
];

function emptyCredentialState(provider: string): Record<string, string> {
  const found = providerOptionDefs.find((p) => p.value === provider);
  if (!found) return {};
  return Object.fromEntries(found.fields.map((f) => [f.key, ""]));
}

function StatusBadge({ status }: { status?: DNSProviderAccountStatus | string }) {
  const { t } = useTranslation();

  switch (status) {
    case "active":
      return <Badge variant="default" className="bg-green-500/15 text-green-700 hover:bg-green-500/20 border-green-200">{t("dnsProviderAccount.statusActive")}</Badge>;
    case "degraded":
      return <Badge variant="secondary" className="bg-yellow-500/15 text-yellow-700 hover:bg-yellow-500/20 border-yellow-200">{t("dnsProviderAccount.statusDegraded")}</Badge>;
    case "disabled":
      return <Badge variant="destructive">{t("dnsProviderAccount.statusDisabled")}</Badge>;
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

  const providerLabel = useCallback(
    (provider: string) => t(`dnsProviders.${provider}`, provider),
    [t],
  );
  const fieldLabel = useCallback((labelKey: string) => t(labelKey), [t]);
  const fieldPlaceholder = useCallback(
    (labelKey: string, keepUnchanged = false) =>
      keepUnchanged ? `${fieldLabel(labelKey)} (${t("dnsProviderAccount.fieldKeepUnchanged")})` : fieldLabel(labelKey),
    [fieldLabel, t],
  );

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

  const addFields = useMemo(() => providerOptionDefs.find((p) => p.value === addProvider)?.fields ?? [], [addProvider]);
  const editFields = useMemo(() => providerOptionDefs.find((p) => p.value === editProvider)?.fields ?? [], [editProvider]);

  const tableHeaders = (
    <>
      <TableHead>{t("dnsProviderAccount.provider")}</TableHead>
      <TableHead>{t("dnsProviderAccount.accountName")}</TableHead>
      <TableHead>{t("dnsProviderAccount.status")}</TableHead>
      <TableHead>{t("dnsProviderAccount.credentialHint")}</TableHead>
      <TableHead>{t("dnsProviderAccount.lastVerifiedAt")}</TableHead>
      <TableHead className="text-right">{t("adminDomains.actions")}</TableHead>
    </>
  );

  const createMutation = useMutation({
    mutationFn: api.adminCreateDNSProviderAccount,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-dns-provider-accounts"] });
      toast.success(t("dnsProviderAccount.accountCreated"));
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
      toast.success(t("dnsProviderAccount.accountUpdated"));
      setEditAccount(null);
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.adminDeleteDNSProviderAccount(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-dns-provider-accounts"] });
      toast.success(t("dnsProviderAccount.accountDeleted"));
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
                <TableRow>{tableHeaders}</TableRow>
              </TableHeader>
              <TableBody>
                {[...Array(3)].map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><Skeleton className="h-4 w-28" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-28" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-20" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-24" /></TableCell>
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
          {t("dnsProviderAccount.addAccount")}
        </Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>{tableHeaders}</TableRow>
            </TableHeader>
            <TableBody>
              {accounts?.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                    {t("dnsProviderAccount.noAccounts")}
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
                      {t("common.edit")}
                    </Button>
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => setDeleteAccount(account)}
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

      <Dialog open={showAdd} onOpenChange={setShowAdd}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{t("dnsProviderAccount.addAccount")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-1">
              <Label>{t("dnsProviderAccount.provider")}</Label>
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
                  {providerOptionDefs.map((p) => (
                    <SelectItem key={p.value} value={p.value}>{providerLabel(p.value)}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1">
              <Label>{t("dnsProviderAccount.accountName")}</Label>
              <Input
                value={addName}
                onChange={(e) => setAddName(e.target.value)}
                placeholder={t("dnsProviderAccount.accountName")}
              />
            </div>
            {addFields.map((field) => (
              <div key={field.key} className="space-y-1">
                <Label>{fieldLabel(field.labelKey)}</Label>
                <Input
                  type={field.secret ? "password" : "text"}
                  value={addCredentials[field.key] ?? ""}
                  onChange={(e) =>
                    setAddCredentials((prev) => ({ ...prev, [field.key]: e.target.value }))
                  }
                  placeholder={fieldPlaceholder(field.labelKey)}
                />
              </div>
            ))}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAdd(false)}>
              {t("common.cancel")}
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
              {createMutation.isPending ? t("common.saving") : t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={!!editAccount} onOpenChange={(open) => !open && setEditAccount(null)}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{t("dnsProviderAccount.editAccount")}</DialogTitle>
          </DialogHeader>
          {editAccount && (
            <div className="space-y-4 py-2">
              <div className="space-y-1">
                <Label>{t("dnsProviderAccount.provider")}</Label>
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
                    {providerOptionDefs.map((p) => (
                      <SelectItem key={p.value} value={p.value}>{providerLabel(p.value)}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1">
                <Label>{t("dnsProviderAccount.accountName")}</Label>
                <Input
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                />
              </div>
              {editFields.map((field) => (
                <div key={field.key} className="space-y-1">
                  <Label>{fieldLabel(field.labelKey)}</Label>
                  <Input
                    type={field.secret ? "password" : "text"}
                    value={editCredentials[field.key] ?? ""}
                    onChange={(e) =>
                      setEditCredentials((prev) => ({ ...prev, [field.key]: e.target.value }))
                    }
                    placeholder={fieldPlaceholder(field.labelKey, true)}
                  />
                </div>
              ))}
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditAccount(null)}>
              {t("common.cancel")}
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
              {updateMutation.isPending ? t("common.saving") : t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={!!deleteAccount} onOpenChange={(open) => !open && setDeleteAccount(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("dnsProviderAccount.deleteConfirm", { name: deleteAccount?.name ?? "" })}</DialogTitle>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteAccount(null)}>
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              disabled={deleteMutation.isPending}
              onClick={() => deleteAccount && deleteMutation.mutate(deleteAccount.id)}
            >
              {deleteMutation.isPending ? t("common.deleting") : t("common.delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

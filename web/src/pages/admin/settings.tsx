import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import {
  Database,
  Download,
  KeyRound,
  Mail,
  Palette,
  PlugZap,
  Save,
  ShieldCheck,
  Smartphone,
  Trash2,
} from "lucide-react";
import { toast } from "sonner";
import { api, getErrorMessage } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { useDocumentTitle } from "@/hooks/use-document-title";

const maskedSecret = "********";

type SettingsSection = "access" | "site" | "mail" | "integrations" | "android" | "maintenance";

const sections: Array<{ value: SettingsSection; label: string; icon: typeof ShieldCheck }> = [
  { value: "access", label: "访问与注册", icon: ShieldCheck },
  { value: "site", label: "站点与外观", icon: Palette },
  { value: "mail", label: "邮件通知", icon: Mail },
  { value: "integrations", label: "支付与集成", icon: PlugZap },
  { value: "android", label: "Android 客户端", icon: Smartphone },
  { value: "maintenance", label: "数据维护", icon: Database },
];

function splitDomains(value: string) {
  return value
    .split(/[\n,]/)
    .map((entry) => entry.trim())
    .filter(Boolean);
}

function downloadBlob(blob: Blob, filename: string) {
  const href = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = href;
  anchor.download = filename;
  anchor.click();
  window.setTimeout(() => URL.revokeObjectURL(href), 0);
}

export default function AdminSettingsPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  useDocumentTitle(t("adminSettings.title"));

  const { data: config, isLoading: configLoading } = useQuery({
    queryKey: ["admin-config"],
    queryFn: async () => (await api.adminGetConfig()).data,
    staleTime: 30_000,
  });
  const { data: accessSettings, isLoading: accessLoading } = useQuery({
    queryKey: ["admin-access-settings"],
    queryFn: async () => (await api.adminGetAccessSettings()).data,
    staleTime: 30_000,
  });
  const { data: clientConfig } = useQuery({
    queryKey: ["admin-client-config"],
    queryFn: async () => (await api.adminGetClientConfig()).data,
    staleTime: 30_000,
  });
  const { data: restoreJobs } = useQuery({
    queryKey: ["admin-database-restores"],
    queryFn: () => api.adminListDatabaseRestores(),
    staleTime: 10_000,
  });

  const [registrationEnabled, setRegistrationEnabled] = useState(true);
  const [domainPolicyMode, setDomainPolicyMode] = useState<"unrestricted" | "allowlist" | "blocklist">("unrestricted");
  const [domainPolicyDomains, setDomainPolicyDomains] = useState("");
  const [frontendUrls, setFrontendUrls] = useState("");
  const [backendUrls, setBackendUrls] = useState("");
  const [announcementEnabled, setAnnouncementEnabled] = useState(false);
  const [announcementContent, setAnnouncementContent] = useState("");
  const [footerIcp, setFooterIcp] = useState("");
  const [footerIcpLink, setFooterIcpLink] = useState("");
  const [footerContent, setFooterContent] = useState("");
  const [seoDescription, setSeoDescription] = useState("");
  const [seoKeywords, setSeoKeywords] = useState("");
  const [seoIndexingDisabled, setSeoIndexingDisabled] = useState(false);
  const [smtpHost, setSmtpHost] = useState("");
  const [smtpPort, setSmtpPort] = useState("587");
  const [smtpUsername, setSmtpUsername] = useState("");
  const [smtpPassword, setSmtpPassword] = useState("");
  const [smtpFromName, setSmtpFromName] = useState("");
  const [smtpFromAddr, setSmtpFromAddr] = useState("");
  const [smtpUseTLS, setSmtpUseTLS] = useState(true);
  const [smtpEnabled, setSmtpEnabled] = useState(false);
  const [epayURL, setEpayURL] = useState("");
  const [epayPID, setEpayPID] = useState("");
  const [epayKey, setEpayKey] = useState("");
  const [codepayURL, setCodepayURL] = useState("");
  const [codepayID, setCodepayID] = useState("");
  const [codepayKey, setCodepayKey] = useState("");
  const [epayAlipayEnabled, setEpayAlipayEnabled] = useState(false);
  const [epayWechatEnabled, setEpayWechatEnabled] = useState(false);
  const [epayQQEnabled, setEpayQQEnabled] = useState(false);
  const [codepayAlipayEnabled, setCodepayAlipayEnabled] = useState(false);
  const [codepayWechatEnabled, setCodepayWechatEnabled] = useState(false);
  const [codepayQQEnabled, setCodepayQQEnabled] = useState(false);
  const [clientVersion, setClientVersion] = useState("1.0.0");
  const [clientForceUpdate, setClientForceUpdate] = useState(false);
  const [clientUpdateNotice, setClientUpdateNotice] = useState("");
  const [clientUpdateURL, setClientUpdateURL] = useState("");
  const [communicationKey, setCommunicationKey] = useState("");
  const [restoreOpen, setRestoreOpen] = useState(false);
  const [restoreFile, setRestoreFile] = useState<File | null>(null);
  const [restorePassword, setRestorePassword] = useState("");
  const [restoreConfirmation, setRestoreConfirmation] = useState("");

  /* eslint-disable react-hooks/set-state-in-effect -- Remote query snapshots initialize the editable drafts below. */
  useEffect(() => {
    if (!accessSettings) return;
    setRegistrationEnabled(accessSettings.registration_enabled);
    setDomainPolicyMode(accessSettings.domain_policy_mode);
    setDomainPolicyDomains(accessSettings.domain_policy_domains.join("\n"));
  }, [accessSettings]);

  useEffect(() => {
    if (!config) return;
    const values = config.values ?? {};
    setFrontendUrls(values.frontend_urls ?? config.url_runtime.frontend_urls?.join("\n") ?? values.frontend_url ?? "");
    setBackendUrls(values.backend_urls ?? config.url_runtime.backend_urls?.join("\n") ?? values.backend_url ?? "");
    setAnnouncementEnabled(values.announcement_enabled === "true");
    setAnnouncementContent(values.announcement_content ?? "");
    setFooterIcp(values.site_footer_icp ?? "");
    setFooterIcpLink(values.site_footer_icp_link ?? "");
    setFooterContent(values.site_footer_content ?? "");
    setSeoDescription(values.seo_description ?? "");
    setSeoKeywords(values.seo_keywords ?? "");
    setSeoIndexingDisabled(values.seo_indexing_disabled === "true");
    setSmtpHost(values.smtp_host ?? "");
    setSmtpPort(values.smtp_port ?? "587");
    setSmtpUsername(values.smtp_username ?? "");
    setSmtpPassword(values.smtp_password ? maskedSecret : "");
    setSmtpFromName(values.smtp_from_name ?? "");
    setSmtpFromAddr(values.smtp_from_addr ?? "");
    setSmtpUseTLS(values.smtp_use_tls !== "false");
    setSmtpEnabled(values.smtp_enabled === "true");
    setEpayURL(values.epay_url ?? "");
    setEpayPID(values.epay_pid ?? "");
    setEpayKey(values.epay_key ? maskedSecret : "");
    setCodepayURL(values.codepay_url ?? "");
    setCodepayID(values.codepay_id ?? "");
    setCodepayKey(values.codepay_key ? maskedSecret : "");
    setEpayAlipayEnabled(values.epay_alipay_enabled === "true");
    setEpayWechatEnabled(values.epay_wechat_enabled === "true");
    setEpayQQEnabled(values.epay_qq_enabled === "true");
    setCodepayAlipayEnabled(values.codepay_alipay_enabled === "true");
    setCodepayWechatEnabled(values.codepay_wechat_enabled === "true");
    setCodepayQQEnabled(values.codepay_qq_enabled === "true");
  }, [config]);

  useEffect(() => {
    if (!clientConfig) return;
    setClientVersion(clientConfig.latest_version || "1.0.0");
    setClientForceUpdate(clientConfig.force_update);
    setClientUpdateNotice(clientConfig.update_notice ?? "");
    setClientUpdateURL(clientConfig.update_url ?? "");
  }, [clientConfig]);
  /* eslint-enable react-hooks/set-state-in-effect */

  const updateConfig = useMutation({
    mutationFn: api.adminUpdateConfig,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-config"] });
      toast.success(t("adminSettings.saved"));
    },
    onError: (error) => toast.error(getErrorMessage(error, t)),
  });
  const updateAccess = useMutation({
    mutationFn: api.adminUpdateAccessSettings,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-access-settings"] });
      toast.success(t("adminSettings.saved"));
    },
    onError: (error) => toast.error(getErrorMessage(error, t)),
  });
  const updateClient = useMutation({
    mutationFn: api.adminUpdateClientConfig,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-client-config"] });
      toast.success(t("adminSettings.saved"));
    },
    onError: (error) => toast.error(getErrorMessage(error, t)),
  });
  const generateKey = useMutation({
    mutationFn: api.adminGenerateClientCommunicationKey,
    onSuccess: (result) => {
      setCommunicationKey(result.data.communication_key);
      queryClient.invalidateQueries({ queryKey: ["admin-client-config"] });
      toast.success("通讯密钥已生成");
    },
    onError: (error) => toast.error(getErrorMessage(error, t)),
  });
  const revokeKey = useMutation({
    mutationFn: api.adminRevokeClientCommunicationKey,
    onSuccess: () => {
      setCommunicationKey("");
      queryClient.invalidateQueries({ queryKey: ["admin-client-config"] });
      toast.success("通讯密钥已作废");
    },
    onError: (error) => toast.error(getErrorMessage(error, t)),
  });
  const exportDatabase = useMutation({
    mutationFn: api.adminDownloadDatabaseExport,
    onSuccess: ({ blob, filename }) => {
      downloadBlob(blob, filename);
      toast.success("数据库备份已生成");
    },
    onError: (error) => toast.error(getErrorMessage(error, t)),
  });
  const restoreDatabase = useMutation({
    mutationFn: async () => {
      if (!restoreFile) throw new Error("请选择数据库备份文件");
      const challenge = await api.adminCreateRestoreChallenge(restorePassword);
      return api.adminRestoreDatabase({
        archive: restoreFile,
        password: restorePassword,
        challenge: challenge.data.challenge,
        confirmation: restoreConfirmation,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-database-restores"] });
      setRestoreOpen(false);
      setRestoreFile(null);
      setRestorePassword("");
      setRestoreConfirmation("");
      toast.success("数据库恢复已完成，服务正在维护模式中");
    },
    onError: (error) => toast.error(getErrorMessage(error, t)),
  });

  const frontendLocked = Boolean(config?.url_runtime.frontend_env_locked);
  const backendLocked = Boolean(config?.url_runtime.backend_env_locked);
  const restoreReady = Boolean(restoreFile && restorePassword && restoreConfirmation === "RESTORE DATABASE");
  const latestRestore = useMemo(() => restoreJobs?.data.items?.[0], [restoreJobs]);

  if (configLoading || accessLoading) {
    return <SettingsSkeleton />;
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("adminSettings.title")}</h1>
      </div>

      <Tabs defaultValue="access" orientation="vertical" className="gap-6 lg:flex-row">
        <TabsList variant="line" className="w-full overflow-x-auto lg:w-52 lg:shrink-0">
          {sections.map((section) => {
            const Icon = section.icon;
            return <TabsTrigger key={section.value} value={section.value} className="shrink-0 lg:justify-start"><Icon />{section.label}</TabsTrigger>;
          })}
        </TabsList>

        <TabsContent value="access" className="mt-0">
          <Card>
            <CardHeader><CardTitle>访问与注册</CardTitle></CardHeader>
            <CardContent className="space-y-5">
              <div className="flex items-center justify-between gap-4"><Label>允许新用户注册</Label><Switch checked={registrationEnabled} onCheckedChange={setRegistrationEnabled} /></div>
              <div className="space-y-2"><Label>邮箱后缀策略</Label><select className="h-9 w-full rounded-md border bg-background px-3 text-sm" value={domainPolicyMode} onChange={(event) => setDomainPolicyMode(event.target.value as typeof domainPolicyMode)}><option value="unrestricted">不限制</option><option value="allowlist">仅白名单</option><option value="blocklist">黑名单</option></select></div>
              <div className="space-y-2"><Label>邮箱后缀</Label><Textarea value={domainPolicyDomains} onChange={(event) => setDomainPolicyDomains(event.target.value)} rows={7} /></div>
              <Button onClick={() => updateAccess.mutate({ registration_enabled: registrationEnabled, domain_policy_mode: domainPolicyMode, domain_policy_domains: splitDomains(domainPolicyDomains) })} disabled={updateAccess.isPending}><Save />{updateAccess.isPending ? t("common.saving") : t("adminSettings.save")}</Button>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="site" className="mt-0">
          <Card>
            <CardHeader><CardTitle>站点与外观</CardTitle></CardHeader>
            <CardContent className="space-y-5">
              <div className="grid gap-4 lg:grid-cols-2"><div className="space-y-2"><Label>前端地址</Label><Textarea value={frontendUrls} disabled={frontendLocked} onChange={(event) => setFrontendUrls(event.target.value)} rows={4} /></div><div className="space-y-2"><Label>后端地址</Label><Textarea value={backendUrls} disabled={backendLocked} onChange={(event) => setBackendUrls(event.target.value)} rows={4} /></div></div>
              <div className="flex items-center justify-between gap-4"><Label>站点公告</Label><Switch checked={announcementEnabled} onCheckedChange={setAnnouncementEnabled} /></div>
              <Textarea value={announcementContent} onChange={(event) => setAnnouncementContent(event.target.value)} rows={4} />
              <div className="grid gap-4 lg:grid-cols-2"><div className="space-y-2"><Label>ICP备案号</Label><Input value={footerIcp} onChange={(event) => setFooterIcp(event.target.value)} /></div><div className="space-y-2"><Label>备案链接</Label><Input value={footerIcpLink} onChange={(event) => setFooterIcpLink(event.target.value)} /></div></div>
              <div className="space-y-2"><Label>页脚内容</Label><Textarea value={footerContent} onChange={(event) => setFooterContent(event.target.value)} rows={3} /></div>
              <div className="space-y-2"><Label>SEO 描述</Label><Textarea value={seoDescription} onChange={(event) => setSeoDescription(event.target.value)} rows={3} /></div>
              <div className="grid gap-4 lg:grid-cols-[1fr_auto]"><div className="space-y-2"><Label>SEO 关键词</Label><Input value={seoKeywords} onChange={(event) => setSeoKeywords(event.target.value)} /></div><div className="flex items-end gap-3 pb-2"><Label>禁止索引</Label><Switch checked={seoIndexingDisabled} onCheckedChange={setSeoIndexingDisabled} /></div></div>
              <Button onClick={() => updateConfig.mutate({ ...(frontendLocked ? {} : { frontend_urls: frontendUrls }), ...(backendLocked ? {} : { backend_urls: backendUrls }), announcement_enabled: String(announcementEnabled), announcement_content: announcementContent, site_footer_icp: footerIcp, site_footer_icp_link: footerIcpLink, site_footer_content: footerContent, seo_description: seoDescription, seo_keywords: seoKeywords, seo_indexing_disabled: String(seoIndexingDisabled) })} disabled={updateConfig.isPending}><Save />{updateConfig.isPending ? t("common.saving") : t("adminSettings.save")}</Button>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="mail" className="mt-0">
          <Card>
            <CardHeader><CardTitle>邮件通知</CardTitle></CardHeader>
            <CardContent className="space-y-5">
              <div className="flex items-center justify-between gap-4"><Label>启用 SMTP</Label><Switch checked={smtpEnabled} onCheckedChange={setSmtpEnabled} /></div>
              <div className="grid gap-4 lg:grid-cols-2"><div className="space-y-2"><Label>SMTP 主机</Label><Input value={smtpHost} onChange={(event) => setSmtpHost(event.target.value)} /></div><div className="space-y-2"><Label>端口</Label><Input value={smtpPort} onChange={(event) => setSmtpPort(event.target.value)} /></div><div className="space-y-2"><Label>用户名</Label><Input value={smtpUsername} onChange={(event) => setSmtpUsername(event.target.value)} /></div><div className="space-y-2"><Label>密码</Label><Input type="password" value={smtpPassword} onChange={(event) => setSmtpPassword(event.target.value)} /></div><div className="space-y-2"><Label>发件人名称</Label><Input value={smtpFromName} onChange={(event) => setSmtpFromName(event.target.value)} /></div><div className="space-y-2"><Label>发件人地址</Label><Input value={smtpFromAddr} onChange={(event) => setSmtpFromAddr(event.target.value)} /></div></div>
              <div className="flex items-center justify-between gap-4"><Label>使用 TLS</Label><Switch checked={smtpUseTLS} onCheckedChange={setSmtpUseTLS} /></div>
              <div className="flex flex-wrap gap-3"><Button onClick={() => updateConfig.mutate({ smtp_host: smtpHost, smtp_port: smtpPort, smtp_username: smtpUsername, smtp_from_name: smtpFromName, smtp_from_addr: smtpFromAddr, smtp_use_tls: String(smtpUseTLS), smtp_enabled: String(smtpEnabled), ...(smtpPassword && smtpPassword !== maskedSecret ? { smtp_password: smtpPassword } : {}) })} disabled={updateConfig.isPending}><Save />{t("adminSettings.save")}</Button><Button variant="outline" disabled={!smtpEnabled} onClick={async () => { try { await api.adminTestSMTP(); toast.success(t("adminSettings.smtpTestSent")); } catch (error) { toast.error(getErrorMessage(error, t)); } }}>测试发送</Button></div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="integrations" className="mt-0">
          <Card>
            <CardHeader><CardTitle>支付与集成</CardTitle></CardHeader>
            <CardContent className="space-y-6">
              <PaymentFields title="易支付" url={epayURL} onURL={setEpayURL} identifier={epayPID} onIdentifier={setEpayPID} secret={epayKey} onSecret={setEpayKey} labels={["支付宝", "微信", "QQ"]} enabled={[epayAlipayEnabled, epayWechatEnabled, epayQQEnabled]} onEnabled={[setEpayAlipayEnabled, setEpayWechatEnabled, setEpayQQEnabled]} />
              <PaymentFields title="码支付" url={codepayURL} onURL={setCodepayURL} identifier={codepayID} onIdentifier={setCodepayID} secret={codepayKey} onSecret={setCodepayKey} labels={["支付宝", "微信", "QQ"]} enabled={[codepayAlipayEnabled, codepayWechatEnabled, codepayQQEnabled]} onEnabled={[setCodepayAlipayEnabled, setCodepayWechatEnabled, setCodepayQQEnabled]} />
              <Button onClick={() => updateConfig.mutate({ epay_url: epayURL, epay_pid: epayPID, codepay_url: codepayURL, codepay_id: codepayID, epay_alipay_enabled: String(epayAlipayEnabled), epay_wechat_enabled: String(epayWechatEnabled), epay_qq_enabled: String(epayQQEnabled), codepay_alipay_enabled: String(codepayAlipayEnabled), codepay_wechat_enabled: String(codepayWechatEnabled), codepay_qq_enabled: String(codepayQQEnabled), ...(epayKey && epayKey !== maskedSecret ? { epay_key: epayKey } : {}), ...(codepayKey && codepayKey !== maskedSecret ? { codepay_key: codepayKey } : {}) })} disabled={updateConfig.isPending}><Save />{t("adminSettings.save")}</Button>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="android" className="mt-0">
          <Card>
            <CardHeader><CardTitle>Android 客户端</CardTitle></CardHeader>
            <CardContent className="space-y-5">
              <div className="grid gap-4 lg:grid-cols-2"><div className="space-y-2"><Label>最新版本</Label><Input value={clientVersion} onChange={(event) => setClientVersion(event.target.value)} /></div><div className="flex items-end justify-between gap-4 pb-2"><Label>强制更新</Label><Switch checked={clientForceUpdate} onCheckedChange={setClientForceUpdate} /></div></div>
              <div className="space-y-2"><Label>更新公告</Label><Textarea value={clientUpdateNotice} onChange={(event) => setClientUpdateNotice(event.target.value)} rows={4} /></div>
              <div className="space-y-2"><Label>更新链接</Label><Input value={clientUpdateURL} onChange={(event) => setClientUpdateURL(event.target.value)} /></div>
              <Button onClick={() => updateClient.mutate({ latest_version: clientVersion.trim(), force_update: clientForceUpdate, update_notice: clientUpdateNotice, update_url: clientUpdateURL.trim() })} disabled={updateClient.isPending}><Save />{t("adminSettings.save")}</Button>
              <div className="border-t pt-5"><div className="flex flex-wrap gap-3"><Button variant="outline" onClick={() => generateKey.mutate()} disabled={generateKey.isPending}><KeyRound />生成通讯密钥</Button><Button variant="destructive" onClick={() => revokeKey.mutate()} disabled={revokeKey.isPending}><Trash2 />作废通讯密钥</Button></div>{communicationKey && <div className="mt-4 flex gap-2"><Input value={communicationKey} readOnly /><Button variant="outline" onClick={async () => { await navigator.clipboard.writeText(communicationKey); toast.success("通讯密钥已复制"); }}>复制</Button></div>}</div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="maintenance" className="mt-0">
          <Card>
            <CardHeader><CardTitle>数据维护</CardTitle></CardHeader>
            <CardContent className="space-y-5">
              <div className="flex flex-wrap gap-3"><Button onClick={() => exportDatabase.mutate()} disabled={exportDatabase.isPending}><Download />{exportDatabase.isPending ? "正在导出" : "导出数据库"}</Button><Button variant="destructive" onClick={() => setRestoreOpen(true)}><Database />恢复数据库</Button></div>
              {latestRestore && <div className="border-t pt-4 text-sm"><div>最近恢复任务 #{latestRestore.id}</div><div className="text-muted-foreground">{latestRestore.status}</div></div>}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <Dialog open={restoreOpen} onOpenChange={setRestoreOpen}>
        <DialogContent showCloseButton={!restoreDatabase.isPending}>
          <DialogHeader><DialogTitle>恢复数据库</DialogTitle><DialogDescription>此操作会覆盖当前数据库。</DialogDescription></DialogHeader>
          <div className="space-y-4"><div className="space-y-2"><Label>数据库备份 ZIP</Label><Input type="file" accept=".zip,application/zip" onChange={(event) => setRestoreFile(event.target.files?.[0] ?? null)} /></div><div className="space-y-2"><Label>当前密码</Label><Input type="password" value={restorePassword} onChange={(event) => setRestorePassword(event.target.value)} /></div><div className="space-y-2"><Label>确认短语</Label><Input value={restoreConfirmation} onChange={(event) => setRestoreConfirmation(event.target.value)} /></div></div>
          <DialogFooter><Button variant="destructive" onClick={() => restoreDatabase.mutate()} disabled={!restoreReady || restoreDatabase.isPending}><Database />{restoreDatabase.isPending ? "正在恢复" : "恢复数据库"}</Button></DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function PaymentFields(props: {
  title: string;
  url: string;
  onURL: (value: string) => void;
  identifier: string;
  onIdentifier: (value: string) => void;
  secret: string;
  onSecret: (value: string) => void;
  labels: string[];
  enabled: boolean[];
  onEnabled: Array<(value: boolean) => void>;
}) {
  return <div className="space-y-4"><h2 className="text-base font-semibold">{props.title}</h2><div className="grid gap-4 lg:grid-cols-3"><div className="space-y-2"><Label>网关地址</Label><Input value={props.url} onChange={(event) => props.onURL(event.target.value)} /></div><div className="space-y-2"><Label>商户标识</Label><Input value={props.identifier} onChange={(event) => props.onIdentifier(event.target.value)} /></div><div className="space-y-2"><Label>商户密钥</Label><Input type="password" value={props.secret} onChange={(event) => props.onSecret(event.target.value)} /></div></div><div className="flex flex-wrap gap-5">{props.labels.map((label, index) => <label key={label} className="flex items-center gap-2 text-sm"><Switch checked={props.enabled[index]} onCheckedChange={props.onEnabled[index]} />{label}</label>)}</div></div>;
}

function SettingsSkeleton() {
  return <div className="space-y-6"><Skeleton className="h-8 w-44" /><div className="grid gap-6 lg:grid-cols-[13rem_1fr]"><Skeleton className="h-44" /><Skeleton className="h-96" /></div></div>;
}

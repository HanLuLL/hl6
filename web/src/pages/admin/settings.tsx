import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api, getErrorMessage } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { useDocumentTitle } from "@/hooks/use-document-title";

export default function AdminSettingsPage() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  useDocumentTitle(t("adminSettings.title"));
  const { data: config, isLoading } = useQuery({
    queryKey: ["admin-config"],
    queryFn: async () => {
      const res = await api.adminGetConfig();
      return res.data;
    },
    staleTime: 30_000,
  });

  const [frontendUrls, setFrontendUrls] = useState("");
  const [backendUrls, setBackendUrls] = useState("");
  const [oidcIssuer, setOidcIssuer] = useState("");
  const [oidcClientID, setOidcClientID] = useState("");
  const [oidcClientSecret, setOidcClientSecret] = useState("");
  const [announcementEnabled, setAnnouncementEnabled] = useState(false);
  const [announcementContent, setAnnouncementContent] = useState("");
  const [footerIcp, setFooterIcp] = useState("");
  const [footerIcpLink, setFooterIcpLink] = useState("");
  const [footerContent, setFooterContent] = useState("");
  const [seoDescription, setSeoDescription] = useState("");
  const [seoKeywords, setSeoKeywords] = useState("");
  const [seoIndexingDisabled, setSeoIndexingDisabled] = useState(false);

  // 支付配置
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

  // 邮件 SMTP 配置
  const [smtpHost, setSmtpHost] = useState("");
  const [smtpPort, setSmtpPort] = useState("587");
  const [smtpUsername, setSmtpUsername] = useState("");
  const [smtpPassword, setSmtpPassword] = useState("");
  const [smtpFromName, setSmtpFromName] = useState("");
  const [smtpFromAddr, setSmtpFromAddr] = useState("");
  const [smtpUseTLS, setSmtpUseTLS] = useState(true);
  const [smtpEnabled, setSmtpEnabled] = useState(false);
  const [smtpTesting, setSmtpTesting] = useState(false);

  useEffect(() => {
    if (!config) {
      return;
    }

    const values = config.values ?? {};
    const frontendText = values.frontend_urls
      ?? config.url_runtime.frontend_urls?.join("\n")
      ?? values.frontend_url
      ?? config.url_runtime.frontend_url
      ?? "";
    const backendText = values.backend_urls
      ?? config.url_runtime.backend_urls?.join("\n")
      ?? values.backend_url
      ?? config.url_runtime.backend_url
      ?? "";

    setFrontendUrls(frontendText);
    setBackendUrls(backendText);
    setOidcIssuer(config.oidc_runtime?.issuer ?? values.oidc_issuer ?? "");
    setOidcClientID(config.oidc_runtime?.client_id ?? values.oidc_client_id ?? "");
    setOidcClientSecret("");
    setAnnouncementEnabled(values.announcement_enabled === "true");
    setAnnouncementContent(values.announcement_content ?? "");
    setFooterIcp(values.site_footer_icp ?? "");
    setFooterIcpLink(values.site_footer_icp_link ?? "");
    setFooterContent(values.site_footer_content ?? "");
    setSeoDescription(values.seo_description ?? "");
    setSeoKeywords(values.seo_keywords ?? "");
    setSeoIndexingDisabled(values.seo_indexing_disabled === "true");
    // 支付配置
    setEpayURL(values.epay_url ?? "");
    setEpayPID(values.epay_pid ?? "");
    setEpayKey(values.epay_key && values.epay_key !== "" ? "********" : "");
    setCodepayURL(values.codepay_url ?? "");
    setCodepayID(values.codepay_id ?? "");
    setCodepayKey(values.codepay_key && values.codepay_key !== "" ? "********" : "");
    setEpayAlipayEnabled(values.epay_alipay_enabled === "true");
    setEpayWechatEnabled(values.epay_wechat_enabled === "true");
    setEpayQQEnabled(values.epay_qq_enabled === "true");
    setCodepayAlipayEnabled(values.codepay_alipay_enabled === "true");
    setCodepayWechatEnabled(values.codepay_wechat_enabled === "true");
    setCodepayQQEnabled(values.codepay_qq_enabled === "true");
    // 邮件 SMTP 配置
    setSmtpHost(values.smtp_host ?? "");
    setSmtpPort(values.smtp_port ?? "587");
    setSmtpUsername(values.smtp_username ?? "");
    setSmtpPassword(values.smtp_password && values.smtp_password !== "" ? "********" : "");
    setSmtpFromName(values.smtp_from_name ?? "");
    setSmtpFromAddr(values.smtp_from_addr ?? "");
    setSmtpUseTLS(values.smtp_use_tls !== "false");
    setSmtpEnabled(values.smtp_enabled === "true");
  }, [config]);

  const updateMutation = useMutation({
    mutationFn: api.adminUpdateConfig,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-config"] });
      toast.success(t("adminSettings.saved"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  const confirmUrlMutation = useMutation({
    mutationFn: api.adminConfirmUrlConfig,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-config"] });
      toast.success(t("adminSettings.urlConfirmed"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  const frontendLocked = !!config?.url_runtime?.frontend_env_locked;
  const backendLocked = !!config?.url_runtime?.backend_env_locked;
  const noEditableUrl = frontendLocked && backendLocked;
  const oidcIssuerLocked = !!config?.oidc_runtime?.issuer_env_locked;
  const oidcClientIDLocked = !!config?.oidc_runtime?.client_id_env_locked;
  const oidcClientSecretLocked = !!config?.oidc_runtime?.client_secret_env_locked;
  const noEditableOIDC = oidcIssuerLocked && oidcClientIDLocked && oidcClientSecretLocked;

  const urlSourceLabel = (source?: string) => {
    switch (source) {
      case "env":
        return t("adminSettings.urlSourceEnv");
      case "db":
        return t("adminSettings.urlSourceDb");
      case "auto":
        return t("adminSettings.urlSourceAuto");
      default:
        return t("adminSettings.urlSourceFallback");
    }
  };

  const oidcSourceLabel = (source?: string) => {
    switch (source) {
      case "env":
        return t("adminSettings.urlSourceEnv");
      case "db":
        return t("adminSettings.urlSourceDb");
      default:
        return t("adminSettings.oidcSourceNone");
    }
  };

  const saveUrlConfig = () => {
    const payload: Record<string, string> = {};
    if (!frontendLocked) payload.frontend_urls = frontendUrls.trim();
    if (!backendLocked) payload.backend_urls = backendUrls.trim();
    if (Object.keys(payload).length === 0) return;
    updateMutation.mutate(payload);
  };

  const saveOIDCConfig = () => {
    const payload: Record<string, string> = {};
    if (!oidcIssuerLocked) payload.oidc_issuer = oidcIssuer.trim();
    if (!oidcClientIDLocked) payload.oidc_client_id = oidcClientID.trim();
    if (!oidcClientSecretLocked && oidcClientSecret.trim() !== "") {
      payload.oidc_client_secret = oidcClientSecret.trim();
    }
    if (Object.keys(payload).length === 0) return;
    updateMutation.mutate(payload);
    setOidcClientSecret("");
  };

  const saveAnnouncement = () => {
    updateMutation.mutate({
      announcement_enabled: String(announcementEnabled),
      announcement_content: announcementContent,
    });
  };

  const saveFooter = () => {
    updateMutation.mutate({
      site_footer_icp: footerIcp,
      site_footer_icp_link: footerIcpLink,
      site_footer_content: footerContent,
    });
  };

  const saveSEO = () => {
    updateMutation.mutate({
      seo_description: seoDescription,
      seo_keywords: seoKeywords,
      seo_indexing_disabled: String(seoIndexingDisabled),
    });
  };

  const savePayment = () => {
    const payload: Record<string, string> = {
      epay_url: epayURL,
      epay_pid: epayPID,
      codepay_url: codepayURL,
      codepay_id: codepayID,
      epay_alipay_enabled: String(epayAlipayEnabled),
      epay_wechat_enabled: String(epayWechatEnabled),
      epay_qq_enabled: String(epayQQEnabled),
      codepay_alipay_enabled: String(codepayAlipayEnabled),
      codepay_wechat_enabled: String(codepayWechatEnabled),
      codepay_qq_enabled: String(codepayQQEnabled),
    };
    // 仅在用户修改了密钥时才提交（避免把 "********" 当作真实值提交）
    if (epayKey.trim() !== "" && epayKey.trim() !== "********") {
      payload.epay_key = epayKey.trim();
    }
    if (codepayKey.trim() !== "" && codepayKey.trim() !== "********") {
      payload.codepay_key = codepayKey.trim();
    }
    updateMutation.mutate(payload);
  };

  const saveSMTP = () => {
    const payload: Record<string, string> = {
      smtp_host: smtpHost,
      smtp_port: smtpPort,
      smtp_username: smtpUsername,
      smtp_from_name: smtpFromName,
      smtp_from_addr: smtpFromAddr,
      smtp_use_tls: String(smtpUseTLS),
      smtp_enabled: String(smtpEnabled),
    };
    if (smtpPassword.trim() !== "" && smtpPassword.trim() !== "********") {
      payload.smtp_password = smtpPassword.trim();
    }
    updateMutation.mutate(payload);
  };

  const testSMTP = async () => {
    setSmtpTesting(true);
    try {
      const res = await api.adminTestSMTP();
      toast.success(t("adminSettings.smtpTestSent", { recipient: res.data.recipient }));
    } catch (err) {
      toast.error(getErrorMessage(err, t));
    } finally {
      setSmtpTesting(false);
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("adminSettings.title")}</h1>
          <p className="text-muted-foreground">{t("adminSettings.subtitle")}</p>
        </div>
        <Card>
          <CardHeader>
            <Skeleton className="h-5 w-40" />
            <Skeleton className="mt-1 h-4 w-64" />
          </CardHeader>
          <CardContent>
            <div className="flex items-end gap-4">
              <div className="max-w-xs flex-1 space-y-2">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-9 w-full" />
              </div>
              <Skeleton className="h-9 w-16" />
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("adminSettings.title")}</h1>
        <p className="text-muted-foreground">{t("adminSettings.subtitle")}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("adminSettings.oidcTitle")}</CardTitle>
          <p className="text-sm text-muted-foreground">{t("adminSettings.oidcDesc")}</p>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 lg:grid-cols-2">
            <div className="space-y-2">
              <Label>{t("adminSettings.oidcIssuer")}</Label>
              <Input
                value={oidcIssuer}
                onChange={(e) => setOidcIssuer(e.target.value)}
                placeholder="https://issuer.example.com"
                disabled={oidcIssuerLocked}
              />
              <p className="text-xs text-muted-foreground">
                {t("adminSettings.currentSource", { source: oidcSourceLabel(config?.oidc_runtime?.issuer_source) })}
              </p>
              {oidcIssuerLocked && (
                <p className="text-xs text-muted-foreground">{t("adminSettings.urlLockedByEnv")}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label>{t("adminSettings.oidcClientId")}</Label>
              <Input
                value={oidcClientID}
                onChange={(e) => setOidcClientID(e.target.value)}
                placeholder="client-id"
                disabled={oidcClientIDLocked}
              />
              <p className="text-xs text-muted-foreground">
                {t("adminSettings.currentSource", { source: oidcSourceLabel(config?.oidc_runtime?.client_id_source) })}
              </p>
              {oidcClientIDLocked && (
                <p className="text-xs text-muted-foreground">{t("adminSettings.urlLockedByEnv")}</p>
              )}
            </div>
          </div>

          <div className="space-y-2">
            <Label>{t("adminSettings.oidcClientSecret")}</Label>
            <Input
              type="password"
              value={oidcClientSecret}
              onChange={(e) => setOidcClientSecret(e.target.value)}
              placeholder={t("adminSettings.oidcSecretKeepHint")}
              disabled={oidcClientSecretLocked}
            />
            <p className="text-xs text-muted-foreground">
              {t("adminSettings.currentSource", { source: oidcSourceLabel(config?.oidc_runtime?.client_secret_source) })}
            </p>
            <p className="text-xs text-muted-foreground">
              {config?.oidc_runtime?.client_secret_configured
                ? t("adminSettings.oidcSecretConfigured")
                : t("adminSettings.oidcSecretNotConfigured")}
            </p>
            {!oidcClientSecretLocked && (
              <p className="text-xs text-muted-foreground">{t("adminSettings.oidcSecretKeepHint")}</p>
            )}
            {oidcClientSecretLocked && (
              <p className="text-xs text-muted-foreground">{t("adminSettings.urlLockedByEnv")}</p>
            )}
          </div>

          <div className="flex flex-wrap items-center gap-3">
            {!noEditableOIDC && (
              <Button
                onClick={saveOIDCConfig}
                disabled={updateMutation.isPending}
              >
                {updateMutation.isPending ? t("common.saving") : t("adminSettings.saveOidc")}
              </Button>
            )}
            <p className="text-xs text-muted-foreground">
              {config?.oidc_runtime?.configured ? t("adminSettings.oidcConfigured") : t("adminSettings.oidcNotConfigured")}
            </p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("adminSettings.urlTitle")}</CardTitle>
          <p className="text-sm text-muted-foreground">{t("adminSettings.urlDesc")}</p>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 lg:grid-cols-2">
            <div className="space-y-2">
              <Label>{t("adminSettings.frontendUrl")}</Label>
              <Textarea
                value={frontendUrls}
                onChange={(e) => setFrontendUrls(e.target.value)}
                placeholder={"https://example.com\nhttps://mirror.example.com"}
                disabled={frontendLocked}
                rows={4}
              />
              <p className="text-xs text-muted-foreground">{t("adminSettings.urlMultiInputHint")}</p>
              <p className="text-xs text-muted-foreground">
                {t("adminSettings.currentSource", { source: urlSourceLabel(config?.url_runtime.frontend_source) })}
              </p>
              <p className="text-xs text-muted-foreground">
                {t("adminSettings.activeUrl", { url: config?.url_runtime.frontend_url ?? "-" })}
              </p>
              {frontendLocked && (
                <p className="text-xs text-muted-foreground">{t("adminSettings.urlLockedByEnv")}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label>{t("adminSettings.backendUrl")}</Label>
              <Textarea
                value={backendUrls}
                onChange={(e) => setBackendUrls(e.target.value)}
                placeholder={"https://api.example.com\nhttps://api-b.example.com"}
                disabled={backendLocked}
                rows={4}
              />
              <p className="text-xs text-muted-foreground">{t("adminSettings.urlMultiInputHint")}</p>
              <p className="text-xs text-muted-foreground">
                {t("adminSettings.currentSource", { source: urlSourceLabel(config?.url_runtime.backend_source) })}
              </p>
              <p className="text-xs text-muted-foreground">
                {t("adminSettings.activeUrl", { url: config?.url_runtime.backend_url ?? "-" })}
              </p>
              {backendLocked && (
                <p className="text-xs text-muted-foreground">{t("adminSettings.urlLockedByEnv")}</p>
              )}
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            {!noEditableUrl && (
              <Button
                onClick={saveUrlConfig}
                disabled={updateMutation.isPending}
              >
                {updateMutation.isPending ? t("common.saving") : t("adminSettings.saveUrls")}
              </Button>
            )}
            <Button
              variant="outline"
              onClick={() => confirmUrlMutation.mutate()}
              disabled={confirmUrlMutation.isPending}
            >
              {confirmUrlMutation.isPending ? t("common.saving") : t("adminSettings.confirmCurrentUrls")}
            </Button>
            <p className="text-xs text-muted-foreground">
              {config?.url_runtime.confirmed ? t("adminSettings.confirmedState") : t("adminSettings.unconfirmedState")}
            </p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("adminSettings.announcementTitle")}</CardTitle>
          <p className="text-sm text-muted-foreground">{t("adminSettings.announcementDesc")}</p>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <Label>{t("adminSettings.announcementEnabled")}</Label>
            <Switch
              checked={announcementEnabled}
              onCheckedChange={setAnnouncementEnabled}
            />
          </div>
          <div className="space-y-2">
            <Label>{t("adminSettings.announcementContent")}</Label>
            <Textarea
              value={announcementContent}
              onChange={(e) => setAnnouncementContent(e.target.value)}
              placeholder={t("adminSettings.announcementContentHint")}
              rows={4}
            />
            <p className="text-xs text-muted-foreground">{t("adminSettings.announcementContentHint")}</p>
          </div>
          <Button
            onClick={saveAnnouncement}
            disabled={updateMutation.isPending}
          >
            {updateMutation.isPending ? t("common.saving") : t("adminSettings.saveAnnouncement")}
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("adminSettings.footerTitle")}</CardTitle>
          <p className="text-sm text-muted-foreground">{t("adminSettings.footerDesc")}</p>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>{t("adminSettings.footerIcp")}</Label>
            <Input
              value={footerIcp}
              onChange={(e) => setFooterIcp(e.target.value)}
              placeholder={t("adminSettings.footerIcp")}
            />
          </div>
          <div className="space-y-2">
            <Label>{t("adminSettings.footerIcpLink")}</Label>
            <Input
              value={footerIcpLink}
              onChange={(e) => setFooterIcpLink(e.target.value)}
              placeholder="https://beian.miit.gov.cn"
            />
            <p className="text-xs text-muted-foreground">{t("adminSettings.footerIcpLinkHint")}</p>
          </div>
          <div className="space-y-2">
            <Label>{t("adminSettings.footerContent")}</Label>
            <Textarea
              value={footerContent}
              onChange={(e) => setFooterContent(e.target.value)}
              placeholder={t("adminSettings.footerContentHint")}
              rows={4}
            />
            <p className="text-xs text-muted-foreground">{t("adminSettings.footerContentHint")}</p>
          </div>
          <Button
            onClick={saveFooter}
            disabled={updateMutation.isPending}
          >
            {updateMutation.isPending ? t("common.saving") : t("adminSettings.saveFooter")}
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("adminSettings.seoTitle")}</CardTitle>
          <p className="text-sm text-muted-foreground">{t("adminSettings.seoDesc")}</p>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>{t("adminSettings.seoDescription")}</Label>
            <Textarea
              value={seoDescription}
              onChange={(e) => setSeoDescription(e.target.value)}
              placeholder={t("adminSettings.seoDescriptionHint")}
              rows={3}
            />
            <p className="text-xs text-muted-foreground">{t("adminSettings.seoDescriptionHint")}</p>
          </div>
          <div className="space-y-2">
            <Label>{t("adminSettings.seoKeywords")}</Label>
            <Input
              value={seoKeywords}
              onChange={(e) => setSeoKeywords(e.target.value)}
              placeholder={t("adminSettings.seoKeywordsHint")}
            />
            <p className="text-xs text-muted-foreground">{t("adminSettings.seoKeywordsHint")}</p>
          </div>
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <Label>{t("adminSettings.seoIndexingDisabled")}</Label>
              <p className="text-xs text-muted-foreground">{t("adminSettings.seoIndexingDisabledHint")}</p>
            </div>
            <Switch
              checked={seoIndexingDisabled}
              onCheckedChange={setSeoIndexingDisabled}
            />
          </div>
          <Button
            onClick={saveSEO}
            disabled={updateMutation.isPending}
          >
            {updateMutation.isPending ? t("common.saving") : t("adminSettings.saveSEO")}
          </Button>
        </CardContent>
      </Card>

      {/* 支付配置 */}
      <Card>
        <CardHeader>
          <CardTitle>{t("adminSettings.paymentTitle")}</CardTitle>
          <p className="text-sm text-muted-foreground">{t("adminSettings.paymentDesc")}</p>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* 易支付 */}
          <div className="space-y-4">
            <h3 className="text-sm font-semibold">{t("adminSettings.epayConfig")}</h3>
            <div className="grid gap-4 lg:grid-cols-3">
              <div className="space-y-2">
                <Label>{t("adminSettings.paymentURL")}</Label>
                <Input value={epayURL} onChange={(e) => setEpayURL(e.target.value)} placeholder="https://pay.example.com" />
              </div>
              <div className="space-y-2">
                <Label>{t("adminSettings.paymentPID")}</Label>
                <Input value={epayPID} onChange={(e) => setEpayPID(e.target.value)} placeholder="1001" />
              </div>
              <div className="space-y-2">
                <Label>{t("adminSettings.paymentKey")}</Label>
                <Input
                  type="password"
                  value={epayKey}
                  onChange={(e) => setEpayKey(e.target.value)}
                  placeholder={t("adminSettings.paymentKeyPlaceholder")}
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label className="text-xs uppercase tracking-wide text-muted-foreground">{t("adminSettings.enabledChannels")}</Label>
              <div className="flex flex-wrap gap-4">
                <div className="flex items-center gap-2">
                  <Switch checked={epayAlipayEnabled} onCheckedChange={setEpayAlipayEnabled} />
                  <Label className="font-normal">{t("credits.alipay")}</Label>
                </div>
                <div className="flex items-center gap-2">
                  <Switch checked={epayWechatEnabled} onCheckedChange={setEpayWechatEnabled} />
                  <Label className="font-normal">{t("credits.wechat")}</Label>
                </div>
                <div className="flex items-center gap-2">
                  <Switch checked={epayQQEnabled} onCheckedChange={setEpayQQEnabled} />
                  <Label className="font-normal">{t("credits.qq")}</Label>
                </div>
              </div>
            </div>
          </div>

          <div className="my-2 border-t" />

          {/* 码支付 */}
          <div className="space-y-4">
            <h3 className="text-sm font-semibold">{t("adminSettings.codepayConfig")}</h3>
            <div className="grid gap-4 lg:grid-cols-3">
              <div className="space-y-2">
                <Label>{t("adminSettings.paymentURL")}</Label>
                <Input value={codepayURL} onChange={(e) => setCodepayURL(e.target.value)} placeholder="https://codepay.example.com" />
              </div>
              <div className="space-y-2">
                <Label>{t("adminSettings.paymentID")}</Label>
                <Input value={codepayID} onChange={(e) => setCodepayID(e.target.value)} placeholder="1002" />
              </div>
              <div className="space-y-2">
                <Label>{t("adminSettings.paymentKey")}</Label>
                <Input
                  type="password"
                  value={codepayKey}
                  onChange={(e) => setCodepayKey(e.target.value)}
                  placeholder={t("adminSettings.paymentKeyPlaceholder")}
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label className="text-xs uppercase tracking-wide text-muted-foreground">{t("adminSettings.enabledChannels")}</Label>
              <div className="flex flex-wrap gap-4">
                <div className="flex items-center gap-2">
                  <Switch checked={codepayAlipayEnabled} onCheckedChange={setCodepayAlipayEnabled} />
                  <Label className="font-normal">{t("credits.alipay")}</Label>
                </div>
                <div className="flex items-center gap-2">
                  <Switch checked={codepayWechatEnabled} onCheckedChange={setCodepayWechatEnabled} />
                  <Label className="font-normal">{t("credits.wechat")}</Label>
                </div>
                <div className="flex items-center gap-2">
                  <Switch checked={codepayQQEnabled} onCheckedChange={setCodepayQQEnabled} />
                  <Label className="font-normal">{t("credits.qq")}</Label>
                </div>
              </div>
            </div>
          </div>

          <Button
            onClick={savePayment}
            disabled={updateMutation.isPending}
          >
            {updateMutation.isPending ? t("common.saving") : t("adminSettings.savePayment")}
          </Button>
        </CardContent>
      </Card>

      {/* 邮件 SMTP 配置 */}
      <Card>
        <CardHeader>
          <CardTitle>{t("adminSettings.smtpTitle")}</CardTitle>
          <p className="text-sm text-muted-foreground">{t("adminSettings.smtpDesc")}</p>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <Label>{t("adminSettings.smtpEnabled")}</Label>
              <p className="text-xs text-muted-foreground">{t("adminSettings.smtpEnabledHint")}</p>
            </div>
            <Switch checked={smtpEnabled} onCheckedChange={setSmtpEnabled} />
          </div>
          <div className="grid gap-4 lg:grid-cols-2">
            <div className="space-y-2">
              <Label>{t("adminSettings.smtpHost")}</Label>
              <Input value={smtpHost} onChange={(e) => setSmtpHost(e.target.value)} placeholder="smtp.example.com" />
            </div>
            <div className="space-y-2">
              <Label>{t("adminSettings.smtpPort")}</Label>
              <Input value={smtpPort} onChange={(e) => setSmtpPort(e.target.value)} placeholder="587" />
            </div>
          </div>
          <div className="grid gap-4 lg:grid-cols-2">
            <div className="space-y-2">
              <Label>{t("adminSettings.smtpUsername")}</Label>
              <Input value={smtpUsername} onChange={(e) => setSmtpUsername(e.target.value)} placeholder="noreply@example.com" />
            </div>
            <div className="space-y-2">
              <Label>{t("adminSettings.smtpPassword")}</Label>
              <Input type="password" value={smtpPassword} onChange={(e) => setSmtpPassword(e.target.value)} placeholder={t("adminSettings.smtpPasswordPlaceholder")} />
            </div>
          </div>
          <div className="grid gap-4 lg:grid-cols-2">
            <div className="space-y-2">
              <Label>{t("adminSettings.smtpFromName")}</Label>
              <Input value={smtpFromName} onChange={(e) => setSmtpFromName(e.target.value)} placeholder="SubDomain" />
            </div>
            <div className="space-y-2">
              <Label>{t("adminSettings.smtpFromAddr")}</Label>
              <Input value={smtpFromAddr} onChange={(e) => setSmtpFromAddr(e.target.value)} placeholder="noreply@example.com" />
            </div>
          </div>
          <div className="flex items-center justify-between">
            <Label>{t("adminSettings.smtpUseTLS")}</Label>
            <Switch checked={smtpUseTLS} onCheckedChange={setSmtpUseTLS} />
          </div>
          <div className="flex flex-wrap gap-3">
            <Button onClick={saveSMTP} disabled={updateMutation.isPending}>
              {updateMutation.isPending ? t("common.saving") : t("adminSettings.saveSMTP")}
            </Button>
            <Button variant="outline" onClick={testSMTP} disabled={smtpTesting || !smtpEnabled}>
              {smtpTesting ? t("common.loading") : t("adminSettings.testSMTP")}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

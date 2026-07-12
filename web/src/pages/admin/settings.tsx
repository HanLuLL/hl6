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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
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
  const [seoAuthor, setSeoAuthor] = useState("");
  const [seoOGImage, setSeoOGImage] = useState("");
  const [seoTwitterCard, setSeoTwitterCard] = useState("summary_large_image");
  const [seoTwitterSite, setSeoTwitterSite] = useState("");
  const [seoAnalyticsID, setSeoAnalyticsID] = useState("");
  const [seoIndexingDisabled, setSeoIndexingDisabled] = useState(false);

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
    setSeoAuthor(values.seo_author ?? "");
    setSeoOGImage(values.seo_og_image ?? "");
    setSeoTwitterCard(values.seo_twitter_card ?? "summary_large_image");
    setSeoTwitterSite(values.seo_twitter_site ?? "");
    setSeoAnalyticsID(values.seo_analytics_id ?? "");
    setSeoIndexingDisabled(values.seo_indexing_disabled === "true");
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
      seo_author: seoAuthor,
      seo_og_image: seoOGImage,
      seo_twitter_card: seoTwitterCard,
      seo_twitter_site: seoTwitterSite,
      seo_analytics_id: seoAnalyticsID,
      seo_indexing_disabled: String(seoIndexingDisabled),
    });
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
          <div className="grid gap-4 lg:grid-cols-2">
            <div className="space-y-2">
              <Label>{t("adminSettings.seoKeywords")}</Label>
              <Input
                value={seoKeywords}
                onChange={(e) => setSeoKeywords(e.target.value)}
                placeholder={t("adminSettings.seoKeywordsHint")}
              />
              <p className="text-xs text-muted-foreground">{t("adminSettings.seoKeywordsHint")}</p>
            </div>
            <div className="space-y-2">
              <Label>{t("adminSettings.seoAuthor")}</Label>
              <Input
                value={seoAuthor}
                onChange={(e) => setSeoAuthor(e.target.value)}
                placeholder={t("adminSettings.seoAuthor")}
              />
            </div>
          </div>
          <div className="space-y-2">
            <Label>{t("adminSettings.seoOGImage")}</Label>
            <Input
              value={seoOGImage}
              onChange={(e) => setSeoOGImage(e.target.value)}
              placeholder="https://example.com/og-image.png"
            />
            <p className="text-xs text-muted-foreground">{t("adminSettings.seoOGImageHint")}</p>
          </div>
          <div className="grid gap-4 lg:grid-cols-2">
            <div className="space-y-2">
              <Label>{t("adminSettings.seoTwitterCard")}</Label>
              <Select
                value={seoTwitterCard}
                onValueChange={setSeoTwitterCard}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder={t("adminSettings.seoTwitterCard")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="summary">summary</SelectItem>
                  <SelectItem value="summary_large_image">summary_large_image</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>{t("adminSettings.seoTwitterSite")}</Label>
              <Input
                value={seoTwitterSite}
                onChange={(e) => setSeoTwitterSite(e.target.value)}
                placeholder="@youraccount"
              />
              <p className="text-xs text-muted-foreground">{t("adminSettings.seoTwitterSiteHint")}</p>
            </div>
          </div>
          <div className="space-y-2">
            <Label>{t("adminSettings.seoAnalyticsID")}</Label>
            <Input
              value={seoAnalyticsID}
              onChange={(e) => setSeoAnalyticsID(e.target.value)}
              placeholder="G-XXXXXXXXXX"
            />
            <p className="text-xs text-muted-foreground">{t("adminSettings.seoAnalyticsIDHint")}</p>
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
    </div>
  );
}

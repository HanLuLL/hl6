import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, getErrorMessage } from "@/lib/api";
import { toast } from "sonner";
import { Skeleton } from "@/components/ui/skeleton";

export default function AdminSettingsPage() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  const { data: config, isLoading } = useQuery({
    queryKey: ["admin-config"],
    queryFn: async () => {
      const res = await api.adminGetConfig();
      return res.data;
    },
    staleTime: 30_000,
  });

  const [bonusCredits, setBonusCredits] = useState("0");
  const [referralEnabled, setReferralEnabled] = useState(false);
  const [referralInviterCredits, setReferralInviterCredits] = useState("0");
  const [referralInviteeCredits, setReferralInviteeCredits] = useState("0");
  const [frontendUrls, setFrontendUrls] = useState("");
  const [backendUrls, setBackendUrls] = useState("");

  useEffect(() => {
    if (config) {
      const values = config.values ?? {};
      setBonusCredits(values.registration_bonus_credits ?? "0");
      setReferralEnabled(values.referral_enabled === "true");
      setReferralInviterCredits(values.referral_inviter_credits ?? "0");
      setReferralInviteeCredits(values.referral_invitee_credits ?? "0");

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
    }
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

  const sourceLabel = (source?: string) => {
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

  const saveUrlConfig = () => {
    const payload: Record<string, string> = {};
    if (!frontendLocked) payload.frontend_urls = frontendUrls.trim();
    if (!backendLocked) payload.backend_urls = backendUrls.trim();
    if (Object.keys(payload).length === 0) return;
    updateMutation.mutate(payload);
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
            <Skeleton className="h-4 w-64 mt-1" />
          </CardHeader>
          <CardContent>
            <div className="flex items-end gap-4">
              <div className="space-y-2 flex-1 max-w-xs">
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
                {t("adminSettings.currentSource", { source: sourceLabel(config?.url_runtime.frontend_source) })}
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
                {t("adminSettings.currentSource", { source: sourceLabel(config?.url_runtime.backend_source) })}
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
          <CardTitle>{t("adminSettings.registrationBonus")}</CardTitle>
          <p className="text-sm text-muted-foreground">{t("adminSettings.registrationBonusDesc")}</p>
        </CardHeader>
        <CardContent>
          <div className="flex items-end gap-4">
            <div className="space-y-2 flex-1 max-w-xs">
              <Label>{t("adminSettings.registrationBonus")}</Label>
              <Input
                type="number"
                min="0"
                value={bonusCredits}
                onChange={(e) => setBonusCredits(e.target.value)}
              />
            </div>
            <Button
              onClick={() => updateMutation.mutate({ registration_bonus_credits: bonusCredits })}
              disabled={updateMutation.isPending}
            >
              {updateMutation.isPending ? t("common.saving") : t("common.save")}
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("adminSettings.referralTitle")}</CardTitle>
          <p className="text-sm text-muted-foreground">{t("adminSettings.referralDesc")}</p>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <Label>{t("adminSettings.referralEnabled")}</Label>
            <Switch
              checked={referralEnabled}
              onCheckedChange={(checked) => {
                setReferralEnabled(checked);
                updateMutation.mutate({ referral_enabled: checked ? "true" : "false" });
              }}
            />
          </div>
          <div className="flex items-end gap-4">
            <div className="space-y-2 flex-1 max-w-xs">
              <Label>{t("adminSettings.referralInviterCredits")}</Label>
              <Input
                type="number"
                min="0"
                value={referralInviterCredits}
                onChange={(e) => setReferralInviterCredits(e.target.value)}
              />
            </div>
            <div className="space-y-2 flex-1 max-w-xs">
              <Label>{t("adminSettings.referralInviteeCredits")}</Label>
              <Input
                type="number"
                min="0"
                value={referralInviteeCredits}
                onChange={(e) => setReferralInviteeCredits(e.target.value)}
              />
            </div>
            <Button
              onClick={() => updateMutation.mutate({
                referral_inviter_credits: referralInviterCredits,
                referral_invitee_credits: referralInviteeCredits,
              })}
              disabled={updateMutation.isPending}
            >
              {updateMutation.isPending ? t("common.saving") : t("common.save")}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

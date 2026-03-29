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
import { Switch } from "@/components/ui/switch";

export function LoginRegistrationSettingsContent() {
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
  const [oidcIssuer, setOidcIssuer] = useState("");
  const [oidcClientID, setOidcClientID] = useState("");
  const [oidcClientSecret, setOidcClientSecret] = useState("");

  useEffect(() => {
    if (!config) {
      return;
    }
    const values = config.values ?? {};
    setBonusCredits(values.registration_bonus_credits ?? "0");
    setReferralEnabled(values.referral_enabled === "true");
    setReferralInviterCredits(values.referral_inviter_credits ?? "0");
    setReferralInviteeCredits(values.referral_invitee_credits ?? "0");
    setOidcIssuer(config.oidc_runtime?.issuer ?? values.oidc_issuer ?? "");
    setOidcClientID(config.oidc_runtime?.client_id ?? values.oidc_client_id ?? "");
    setOidcClientSecret("");
  }, [config]);

  const updateMutation = useMutation({
    mutationFn: api.adminUpdateConfig,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-config"] });
      toast.success(t("adminSettings.saved"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  const oidcIssuerLocked = !!config?.oidc_runtime?.issuer_env_locked;
  const oidcClientIDLocked = !!config?.oidc_runtime?.client_id_env_locked;
  const oidcClientSecretLocked = !!config?.oidc_runtime?.client_secret_env_locked;
  const noEditableOIDC = oidcIssuerLocked && oidcClientIDLocked && oidcClientSecretLocked;

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

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-5 w-52" />
          <Skeleton className="mt-1 h-4 w-80" />
        </CardHeader>
        <CardContent className="space-y-3">
          <Skeleton className="h-9 w-full" />
          <Skeleton className="h-9 w-full" />
          <Skeleton className="h-9 w-full" />
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
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
          <CardTitle>{t("adminSettings.registrationBonus")}</CardTitle>
          <p className="text-sm text-muted-foreground">{t("adminSettings.registrationBonusDesc")}</p>
        </CardHeader>
        <CardContent>
          <div className="flex items-end gap-4">
            <div className="max-w-xs flex-1 space-y-2">
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
              disabled={updateMutation.isPending}
            />
          </div>
          <div className="flex items-end gap-4">
            <div className="max-w-xs flex-1 space-y-2">
              <Label>{t("adminSettings.referralInviterCredits")}</Label>
              <Input
                type="number"
                min="0"
                value={referralInviterCredits}
                onChange={(e) => setReferralInviterCredits(e.target.value)}
              />
            </div>
            <div className="max-w-xs flex-1 space-y-2">
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

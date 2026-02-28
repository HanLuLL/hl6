import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
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

  useEffect(() => {
    if (config) {
      setBonusCredits(config.registration_bonus_credits ?? "0");
      setReferralEnabled(config.referral_enabled === "true");
      setReferralInviterCredits(config.referral_inviter_credits ?? "0");
      setReferralInviteeCredits(config.referral_invitee_credits ?? "0");
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

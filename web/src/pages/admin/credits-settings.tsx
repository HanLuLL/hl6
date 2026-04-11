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

function parseGroupIds(raw?: string): number[] {
  if (!raw) return [];
  const values = raw.split(",");
  const seen = new Set<number>();
  const result: number[] = [];
  for (const value of values) {
    const parsed = Number(value.trim());
    if (!Number.isInteger(parsed) || parsed <= 0 || seen.has(parsed)) continue;
    seen.add(parsed);
    result.push(parsed);
  }
  return result.sort((a, b) => a - b);
}

export function CreditsSettingsContent() {
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
  const { data: groupsData } = useQuery({
    queryKey: ["admin-groups"],
    queryFn: async () => {
      const res = await api.adminListGroups();
      return res.data;
    },
    staleTime: 30_000,
  });

  const [bonusCredits, setBonusCredits] = useState("0");
  const [referralEnabled, setReferralEnabled] = useState(false);
  const [referralInviterCredits, setReferralInviterCredits] = useState("0");
  const [referralInviteeCredits, setReferralInviteeCredits] = useState("0");
  const [dailyCheckinEnabled, setDailyCheckinEnabled] = useState(false);
  const [dailyCheckinCredits, setDailyCheckinCredits] = useState("0");
  const [dailyCheckinGroupIds, setDailyCheckinGroupIds] = useState<number[]>([]);

  useEffect(() => {
    if (!config) {
      return;
    }
    const values = config.values ?? {};
    setBonusCredits(values.registration_bonus_credits ?? "0");
    setReferralEnabled(values.referral_enabled === "true");
    setReferralInviterCredits(values.referral_inviter_credits ?? "0");
    setReferralInviteeCredits(values.referral_invitee_credits ?? "0");
    setDailyCheckinEnabled(values.daily_checkin_enabled === "true");
    setDailyCheckinCredits(values.daily_checkin_credits ?? "0");
    setDailyCheckinGroupIds(parseGroupIds(values.daily_checkin_group_ids));
  }, [config]);

  const updateMutation = useMutation({
    mutationFn: api.adminUpdateConfig,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-config"] });
      toast.success(t("adminSettings.saved"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  const saveDailyCheckinConfig = () => {
    const filteredGroupIDs = groupsData
      ? dailyCheckinGroupIds.filter((id) => groupsData.some((g) => g.id === id))
      : dailyCheckinGroupIds;
    if (groupsData) {
      setDailyCheckinGroupIds(filteredGroupIDs);
    }
    updateMutation.mutate({
      daily_checkin_enabled: dailyCheckinEnabled ? "true" : "false",
      daily_checkin_credits: dailyCheckinCredits.trim() || "0",
      daily_checkin_group_ids: filteredGroupIDs.join(","),
    });
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

      <Card>
        <CardHeader>
          <CardTitle>{t("adminSettings.dailyCheckinTitle")}</CardTitle>
          <p className="text-sm text-muted-foreground">{t("adminSettings.dailyCheckinDesc")}</p>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <Label>{t("adminSettings.dailyCheckinEnabled")}</Label>
            <Switch
              checked={dailyCheckinEnabled}
              onCheckedChange={setDailyCheckinEnabled}
              disabled={updateMutation.isPending}
            />
          </div>

          <div className="space-y-2 max-w-xs">
            <Label>{t("adminSettings.dailyCheckinCredits")}</Label>
            <Input
              type="number"
              min="0"
              step="0.1"
              value={dailyCheckinCredits}
              onChange={(e) => setDailyCheckinCredits(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">{t("adminSettings.dailyCheckinZeroDisableHint")}</p>
          </div>

          <div className="space-y-2">
            <Label>{t("adminSettings.dailyCheckinGroups")}</Label>
            {groupsData && groupsData.length > 0 ? (
              <div className="max-h-36 overflow-y-auto border rounded-md">
                {groupsData.map((group) => (
                  <label
                    key={group.id}
                    className="flex items-center gap-2 px-3 py-1.5 hover:bg-accent cursor-pointer text-sm"
                  >
                    <input
                      type="checkbox"
                      checked={dailyCheckinGroupIds.includes(group.id)}
                      onChange={(e) => {
                        if (e.target.checked) {
                          setDailyCheckinGroupIds((prev) => {
                            if (prev.includes(group.id)) return prev;
                            return [...prev, group.id].sort((a, b) => a - b);
                          });
                        } else {
                          setDailyCheckinGroupIds((prev) => prev.filter((id) => id !== group.id));
                        }
                      }}
                    />
                    <span>{group.name}</span>
                    {group.user_count !== undefined && (
                      <span className="text-muted-foreground text-xs">({group.user_count})</span>
                    )}
                  </label>
                ))}
              </div>
            ) : (
              <p className="text-xs text-muted-foreground">{t("adminSettings.noGroups")}</p>
            )}
            <p className="text-xs text-muted-foreground">{t("adminSettings.dailyCheckinGroupsHint")}</p>
          </div>

          <Button
            onClick={saveDailyCheckinConfig}
            disabled={updateMutation.isPending}
          >
            {updateMutation.isPending ? t("common.saving") : t("common.save")}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
